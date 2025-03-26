package utils

import (
        "bytes"
        "fmt"
        "io"
        "net/http"
        "strings"
        "time"
)

// HTTPClient is a wrapper around the standard http client with additional functionality
type HTTPClient struct {
        client *http.Client
}

// NewHTTPClient creates a new HTTP client with sane defaults
func NewHTTPClient() *HTTPClient {
        client := &http.Client{
                Timeout: 10 * time.Second,
                Transport: &http.Transport{
                        MaxIdleConns:        100,
                        MaxIdleConnsPerHost: 100,
                        IdleConnTimeout:     90 * time.Second,
                },
        }
        
        return &HTTPClient{
                client: client,
        }
}

// Get performs an HTTP GET request with a customizable user agent and anti-bot protection bypass
func (c *HTTPClient) Get(url, userAgent string) (string, error) {
        maxRetries := 3
        var lastErr error
        
        // Check if this is a specific explorer with stronger bot protection
        isArbitrumOrBase := false
        if url != "" && (strings.Contains(url, "arbiscan.io") || strings.Contains(url, "basescan.org")) {
                isArbitrumOrBase = true
                maxRetries = 5 // More retries for these sites
        }
        
        for attempt := 0; attempt < maxRetries; attempt++ {
                // Random delay between 100-300ms to make requests look more human
                randomDelay := 100 + (time.Now().UnixNano() % 200)
                time.Sleep(time.Duration(randomDelay) * time.Millisecond)
                
                // Create a new request
                req, err := http.NewRequest("GET", url, nil)
                if err != nil {
                        return "", fmt.Errorf("error creating request: %v", err)
                }
                
                // Set comprehensive headers to mimic a real browser - this helps bypass anti-bot protections
                req.Header.Set("User-Agent", userAgent)
                req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
                req.Header.Set("Accept-Language", "en-US,en;q=0.9")
                req.Header.Set("Accept-Encoding", "gzip, deflate, br")
                req.Header.Set("Connection", "keep-alive")
                req.Header.Set("Upgrade-Insecure-Requests", "1")
                req.Header.Set("Cache-Control", "max-age=0")
                req.Header.Set("Sec-Fetch-Dest", "document")
                req.Header.Set("Sec-Fetch-Mode", "navigate")
                req.Header.Set("Sec-Fetch-Site", "none")
                req.Header.Set("Sec-Fetch-User", "?1")
                req.Header.Set("DNT", "1")
                req.Header.Set("Pragma", "no-cache")
                
                // For Arbitrum and Base, add extra headers that might help bypass protections
                if isArbitrumOrBase {
                    req.Header.Set("sec-ch-ua", "\"Chromium\";v=\"112\", \"Google Chrome\";v=\"112\", \"Not:A-Brand\";v=\"99\"")
                    req.Header.Set("sec-ch-ua-mobile", "?0")
                    req.Header.Set("sec-ch-ua-platform", "\"Windows\"")
                    
                    // Larger set of possible referrers for these sites
                    advancedReferrers := []string{
                        "https://www.google.com/search?q=arbitrum+explorer",
                        "https://www.google.com/search?q=base+blockchain+explorer",
                        "https://twitter.com/arbitrum",
                        "https://coinmarketcap.com/currencies/arbitrum/",
                        "https://etherscan.io/",
                        "https://ethereum.org/en/",
                    }
                    req.Header.Set("Referer", advancedReferrers[attempt%len(advancedReferrers)])
                } else {
                    // Standard referrers for other explorers
                    referrers := []string{
                            "https://www.google.com/",
                            "https://search.brave.com/",
                            "https://duckduckgo.com/",
                            "https://www.bing.com/",
                    }
                    req.Header.Set("Referer", referrers[attempt%len(referrers)])
                }
                
                // Perform the request
                resp, err := c.client.Do(req)
                if err != nil {
                        lastErr = fmt.Errorf("error performing request: %v", err)
                        time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
                        continue
                }
                defer resp.Body.Close()
                
                // Check status code
                if resp.StatusCode != http.StatusOK {
                        lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
                        // Only retry certain status codes
                        if resp.StatusCode == http.StatusForbidden || 
                           resp.StatusCode == http.StatusTooManyRequests || 
                           resp.StatusCode >= 500 {
                                // More aggressive backoff for stronger anti-bot sites
                                if isArbitrumOrBase {
                                    time.Sleep(time.Duration(1000*(attempt+1)) * time.Millisecond)
                                } else {
                                    time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
                                }
                                continue
                        }
                        return "", lastErr
                }
                
                // Read the response body
                body, err := io.ReadAll(resp.Body)
                if err != nil {
                        return "", fmt.Errorf("error reading response body: %v", err)
                }
                
                // If there's any indication of Cloudflare or other protection in the HTML,
                // we might need to retry with a different approach
                if isArbitrumOrBase && (bytes.Contains(body, []byte("Cloudflare")) || 
                   bytes.Contains(body, []byte("challenge")) || 
                   bytes.Contains(body, []byte("captcha"))) {
                        lastErr = fmt.Errorf("detected bot protection page")
                        time.Sleep(time.Duration(1000*(attempt+1)) * time.Millisecond)
                        continue
                }
                
                return string(body), nil
        }
        
        return "", fmt.Errorf("maximum retries reached: %v", lastErr)
}

// Post performs an HTTP POST request with a customizable user agent and body
func (c *HTTPClient) Post(url, userAgent, contentType string, body []byte) (string, error) {
        maxRetries := 3
        var lastErr error
        
        for attempt := 0; attempt < maxRetries; attempt++ {
                // Create a new request with the provided body
                bodyReader := bytes.NewReader(body)
                req, err := http.NewRequest("POST", url, bodyReader)
                if err != nil {
                        return "", fmt.Errorf("error creating request: %v", err)
                }
                
                // Set comprehensive headers to mimic a real browser - similar to Get method
                req.Header.Set("User-Agent", userAgent)
                req.Header.Set("Content-Type", contentType)
                req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
                req.Header.Set("Accept-Language", "en-US,en;q=0.9")
                req.Header.Set("Accept-Encoding", "gzip, deflate, br")
                req.Header.Set("Connection", "keep-alive")
                req.Header.Set("Sec-Fetch-Dest", "document")
                req.Header.Set("Sec-Fetch-Mode", "navigate")
                req.Header.Set("Sec-Fetch-Site", "same-origin")
                req.Header.Set("DNT", "1")
                req.Header.Set("Pragma", "no-cache")
                
                // Randomize the referrer slightly to look more legitimate
                referrers := []string{
                        "https://www.google.com/",
                        "https://search.brave.com/",
                        "https://duckduckgo.com/",
                        "https://www.bing.com/",
                }
                req.Header.Set("Referer", referrers[attempt%len(referrers)])
                
                // Perform the request
                resp, err := c.client.Do(req)
                if err != nil {
                        lastErr = fmt.Errorf("error performing request: %v", err)
                        time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
                        continue
                }
                defer resp.Body.Close()
                
                // Check status code
                if resp.StatusCode != http.StatusOK {
                        lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
                        // Only retry certain status codes
                        if resp.StatusCode == http.StatusForbidden || 
                           resp.StatusCode == http.StatusTooManyRequests || 
                           resp.StatusCode >= 500 {
                                time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
                                continue
                        }
                        return "", lastErr
                }
                
                // Read the response body
                responseBody, err := io.ReadAll(resp.Body)
                if err != nil {
                        return "", fmt.Errorf("error reading response body: %v", err)
                }
                
                return string(responseBody), nil
        }
        
        return "", fmt.Errorf("maximum retries reached: %v", lastErr)
}

// SetTimeout sets the timeout for the HTTP client
func (c *HTTPClient) SetTimeout(timeout time.Duration) {
        c.client.Timeout = timeout
}
