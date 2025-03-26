package utils

import (
        "bufio"
        "fmt"
        "io"
        "net/http"
        "net/url"
        "os"
        "strings"
        "sync"
        "time"
)

// ProxyType represents the type of proxy
type ProxyType int

const (
        HTTP ProxyType = iota
        SOCKS4
        SOCKS5
)

// Proxy represents a proxy server
type Proxy struct {
        URL       string
        Type      ProxyType
        LastUsed  time.Time
        FailCount int
        InUse     bool
}

// ProxyManager handles proxy rotation
type ProxyManager struct {
        proxies         []*Proxy
        proxyIndex      int
        mutex           sync.Mutex
        proxyTimeout    time.Duration
        maxFails        int
        logger          *Logger
        proxyUrl        string
        enabled         bool
        lastRefreshTime time.Time
        refreshInterval time.Duration
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(proxyUrl string, enabled bool, logger *Logger) *ProxyManager {
        pm := &ProxyManager{
                proxies:         make([]*Proxy, 0),
                proxyIndex:      0,
                proxyTimeout:    10 * time.Second,
                maxFails:        3,
                logger:          logger,
                proxyUrl:        proxyUrl,
                enabled:         enabled,
                refreshInterval: 60 * time.Minute, // Set to 1 hour for proxy updates
        }

        // Set timeout from env.txt if available
        if timeout, ok := ReadEnvInt("PROXY_TIMEOUT_SECONDS"); ok {
                pm.proxyTimeout = time.Duration(timeout) * time.Second
        }

        // Set max fails from env.txt if available
        if maxFails, ok := ReadEnvInt("PROXY_MAX_FAILS"); ok {
                pm.maxFails = maxFails
        }
        
        // Set refresh interval from env.txt if available
        if refreshMins, ok := ReadEnvInt("PROXY_REFRESH_MINUTES"); ok && refreshMins > 0 {
                pm.refreshInterval = time.Duration(refreshMins) * time.Minute
                logger.Debug(fmt.Sprintf("Setting proxy refresh interval to %d minutes", refreshMins))
        }

        if enabled {
                err := pm.LoadProxies()
                if err != nil {
                        logger.Error(fmt.Sprintf("Error loading proxies: %v", err))
                }
        }

        return pm
}

// LoadProxies loads proxies from the proxy list URL
func (pm *ProxyManager) LoadProxies() error {
        pm.mutex.Lock()
        defer pm.mutex.Unlock()

        // Return early if not enabled
        if !pm.enabled {
                return nil
        }

        pm.lastRefreshTime = time.Now()
        pm.logger.Info("Loading proxies from URL...")

        // If the proxy URL is a file, load from file
        if strings.HasPrefix(pm.proxyUrl, "file://") {
                filePath := strings.TrimPrefix(pm.proxyUrl, "file://")
                return pm.loadProxiesFromFile(filePath)
        }

        // Otherwise, load from HTTP
        resp, err := http.Get(pm.proxyUrl)
        if err != nil {
                return fmt.Errorf("error fetching proxy list: %v", err)
        }
        defer resp.Body.Close()

        return pm.parseProxyList(resp.Body)
}

// loadProxiesFromFile loads proxies from a file
func (pm *ProxyManager) loadProxiesFromFile(filePath string) error {
        file, err := os.Open(filePath)
        if err != nil {
                return fmt.Errorf("error opening proxy file: %v", err)
        }
        defer file.Close()

        return pm.parseProxyList(file)
}

// parseProxyList parses the proxy list
func (pm *ProxyManager) parseProxyList(r io.Reader) error {
        scanner := bufio.NewScanner(r)
        var newProxies []*Proxy

        for scanner.Scan() {
                line := strings.TrimSpace(scanner.Text())
                if line == "" || strings.HasPrefix(line, "#") {
                        continue
                }

                proxy := &Proxy{
                        URL:       line,
                        LastUsed:  time.Time{},
                        FailCount: 0,
                        InUse:     false,
                }

                // Determine proxy type
                if strings.HasPrefix(line, "http://") {
                        proxy.Type = HTTP
                } else if strings.HasPrefix(line, "socks4://") {
                        proxy.Type = SOCKS4
                } else if strings.HasPrefix(line, "socks5://") {
                        proxy.Type = SOCKS5
                } else {
                        // Default to HTTP if no schema provided
                        proxy.URL = "http://" + line
                        proxy.Type = HTTP
                }

                newProxies = append(newProxies, proxy)
        }

        if scanner.Err() != nil {
                return fmt.Errorf("error reading proxy list: %v", scanner.Err())
        }

        if len(newProxies) == 0 {
                return fmt.Errorf("no valid proxies found")
        }

        // Replace the proxies with the new list
        pm.proxies = newProxies
        pm.proxyIndex = 0

        pm.logger.Info(fmt.Sprintf("Loaded %d proxies", len(pm.proxies)))
        return nil
}

// GetNextProxy returns the next available proxy
func (pm *ProxyManager) GetNextProxy() (*Proxy, error) {
        if !pm.enabled || len(pm.proxies) == 0 {
                return nil, nil // No proxy mode or no proxies available
        }

        pm.mutex.Lock()
        defer pm.mutex.Unlock()

        // Check if we need to refresh the proxy list - use a goroutine to avoid blocking
        if time.Since(pm.lastRefreshTime) > pm.refreshInterval {
                // Update the refresh time immediately to prevent multiple refreshes
                pm.lastRefreshTime = time.Now()
                go func() {
                        err := pm.LoadProxies()
                        if err != nil {
                                pm.logger.Error(fmt.Sprintf("Error refreshing proxies: %v", err))
                        }
                }()
        }

        // Optimization: Use a faster algorithm to find an available proxy
        // Try at most len(proxies) times to find an available proxy
        proxyCount := len(pm.proxies)
        for attempt := 0; attempt < proxyCount; attempt++ {
                proxy := pm.proxies[pm.proxyIndex]
                
                // Move to the next proxy for the next call
                pm.proxyIndex = (pm.proxyIndex + 1) % proxyCount
                
                // Skip proxies that have failed too many times
                if proxy.FailCount > pm.maxFails {
                        continue
                }
                
                // If proxy is not in use and hasn't been used recently, use it
                if !proxy.InUse && time.Since(proxy.LastUsed) > pm.proxyTimeout {
                        proxy.InUse = true
                        proxy.LastUsed = time.Now()
                        return proxy, nil
                }
        }
        
        // If we get here, we've checked all proxies and none are available
        // Reset the "in use" flag for proxies that haven't failed too many times
        resetCount := 0
        for _, p := range pm.proxies {
                if p.FailCount <= pm.maxFails {
                        p.InUse = false
                        resetCount++
                }
        }
        
        // If we have some valid proxies after resetting the flags, try again once
        if resetCount > 0 {
                proxy := pm.proxies[pm.proxyIndex]
                pm.proxyIndex = (pm.proxyIndex + 1) % proxyCount
                
                if proxy.FailCount <= pm.maxFails {
                        proxy.InUse = true
                        proxy.LastUsed = time.Now()
                        return proxy, nil
                }
        }
        
        // No proxy available at this time
        return nil, fmt.Errorf("no available proxies")
}

// ReleaseProxy marks a proxy as no longer in use
func (pm *ProxyManager) ReleaseProxy(proxy *Proxy, success bool) {
        if proxy == nil || !pm.enabled {
                return
        }

        pm.mutex.Lock()
        defer pm.mutex.Unlock()

        proxy.InUse = false
        if !success {
                proxy.FailCount++
                if proxy.FailCount > pm.maxFails {
                        pm.logger.Debug(fmt.Sprintf("Proxy %s has failed too many times, marking as unusable", proxy.URL))
                }
        } else {
                // Reset fail count on success
                proxy.FailCount = 0
        }
}

// GetProxyCount returns the number of loaded proxies
func (pm *ProxyManager) GetProxyCount() int {
        pm.mutex.Lock()
        defer pm.mutex.Unlock()
        return len(pm.proxies)
}

// GetActiveProxyCount returns the number of currently active proxies
func (pm *ProxyManager) GetActiveProxyCount() int {
        pm.mutex.Lock()
        defer pm.mutex.Unlock()
        
        count := 0
        for _, proxy := range pm.proxies {
                if proxy.InUse && proxy.FailCount <= pm.maxFails {
                        count++
                }
        }
        return count
}

// GetHttpClient returns an http.Client configured to use the given proxy
func (pm *ProxyManager) GetHttpClient(proxy *Proxy) (*http.Client, error) {
        if proxy == nil || !pm.enabled {
                return &http.Client{}, nil
        }

        proxyURL, err := url.Parse(proxy.URL)
        if err != nil {
                return nil, err
        }

        return &http.Client{
                Transport: &http.Transport{
                        Proxy: http.ProxyURL(proxyURL),
                },
                Timeout: pm.proxyTimeout,
        }, nil
}

// IsEnabled returns whether the proxy manager is enabled
func (pm *ProxyManager) IsEnabled() bool {
        return pm.enabled
}