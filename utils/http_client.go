package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTPClient is a wrapper around the standard http client with additional functionality
type HTTPClient struct {
	client      *http.Client
	proxyManager *ProxyManager
	logger      *Logger
}

// NewHTTPClient creates a new HTTP client with optimized settings for high performance
func NewHTTPClient() *HTTPClient {
	client := &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        500,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
			DisableCompression:  false,
			ForceAttemptHTTP2:   true,
			// Optimized dial settings for faster connections
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
			// More permissive TLS config
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Don't skip SSL verification
				MinVersion:         tls.VersionTLS12,
			},
		},
	}
	
	return &HTTPClient{
		client: client,
		proxyManager: nil,
		logger: nil,
	}
}

// SetProxyManager sets the proxy manager for this HTTP client
func (c *HTTPClient) SetProxyManager(pm *ProxyManager, logger *Logger) {
	c.proxyManager = pm
	c.logger = logger
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
	
	// If we have a proxy manager, check if we should use it
	var currentProxy *Proxy
	var proxyClient *http.Client
	var usingProxy bool
	
	if c.proxyManager != nil && c.proxyManager.IsEnabled() {
		// Check if we've hit rate limits yet
		rateLimitHit, _ := GetRuntimeBool("RATE_LIMIT_HIT")
		
		// Only use proxies if rate limits have been hit
		if rateLimitHit {
			// Get a proxy
			proxy, err := c.proxyManager.GetNextProxy()
			if err != nil {
				c.logger.Debug(fmt.Sprintf("Failed to get proxy: %v", err))
			} else if proxy != nil {
				currentProxy = proxy
				proxyClient, err = c.proxyManager.GetHttpClient(proxy)
				if err != nil {
					c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
					c.proxyManager.ReleaseProxy(proxy, false)
					currentProxy = nil
				} else {
					usingProxy = true
					c.logger.Debug(fmt.Sprintf("Using proxy: %s", proxy.URL))
				}
			}
		}
	}
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Random delay between 50-150ms to make requests look more human but faster overall
		randomDelay := 50 + (time.Now().UnixNano() % 100)
		time.Sleep(time.Duration(randomDelay) * time.Millisecond)
		
		// Create a new request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			if currentProxy != nil {
				c.proxyManager.ReleaseProxy(currentProxy, false)
			}
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
		
		// Perform the request using either the proxy client or the default client
		var resp *http.Response
		var reqErr error
		if usingProxy {
			resp, reqErr = proxyClient.Do(req)
		} else {
			resp, reqErr = c.client.Do(req)
		}
		
		if reqErr != nil {
			lastErr = fmt.Errorf("error performing request: %v", reqErr)
			
			// If using proxy and request failed, try a different proxy
			if usingProxy && currentProxy != nil {
				c.proxyManager.ReleaseProxy(currentProxy, false)
				// Try to get a new proxy for the next attempt
				proxy, err := c.proxyManager.GetNextProxy()
				if err != nil || proxy == nil {
					c.logger.Debug("Failed to get a new proxy, falling back to direct connection")
					usingProxy = false
					currentProxy = nil
				} else {
					currentProxy = proxy
					proxyClient, err = c.proxyManager.GetHttpClient(proxy)
					if err != nil {
						c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
						c.proxyManager.ReleaseProxy(proxy, false)
						usingProxy = false
						currentProxy = nil
					} else {
						c.logger.Debug(fmt.Sprintf("Switched to proxy: %s", proxy.URL))
					}
				}
			}
			
			time.Sleep(time.Duration(300*(attempt+1)) * time.Millisecond)
			continue
		}
		defer resp.Body.Close()
		
		// Check status code
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			
			// Check if this is a rate limit response (429 Too Many Requests or 403 Forbidden)
			if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden) {
				// Detect rate limit and switch to proxy mode if we're not already using one
				if !usingProxy && c.proxyManager != nil && c.proxyManager.IsEnabled() {
					c.logger.Info("Rate limit detected! Switching to proxy mode...")
					SetRuntimeValue("RATE_LIMIT_HIT", "true")
					
					// Get a proxy for the next attempt
					proxy, err := c.proxyManager.GetNextProxy()
					if err != nil || proxy == nil {
						c.logger.Debug(fmt.Sprintf("Failed to get proxy after rate limit: %v", err))
					} else {
						currentProxy = proxy
						proxyClient, err = c.proxyManager.GetHttpClient(proxy)
						if err != nil {
							c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
							c.proxyManager.ReleaseProxy(proxy, false)
							currentProxy = nil
						} else {
							usingProxy = true
							c.logger.Info(fmt.Sprintf("Switched to proxy after rate limit: %s", proxy.URL))
						}
					}
				}
			}
			
			// If using proxy and got a bad status, try a different proxy
			if usingProxy && currentProxy != nil && (resp.StatusCode == http.StatusForbidden || 
			   resp.StatusCode == http.StatusTooManyRequests) {
				c.proxyManager.ReleaseProxy(currentProxy, false)
				// Try to get a new proxy for the next attempt
				proxy, err := c.proxyManager.GetNextProxy()
				if err != nil || proxy == nil {
					c.logger.Debug("Failed to get a new proxy, falling back to direct connection")
					usingProxy = false
					currentProxy = nil
				} else {
					currentProxy = proxy
					proxyClient, err = c.proxyManager.GetHttpClient(proxy)
					if err != nil {
						c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
						c.proxyManager.ReleaseProxy(proxy, false)
						usingProxy = false
						currentProxy = nil
					} else {
						c.logger.Debug(fmt.Sprintf("Switched to proxy: %s", proxy.URL))
					}
				}
			}
			
			// Only retry certain status codes
			if resp.StatusCode == http.StatusForbidden || 
			   resp.StatusCode == http.StatusTooManyRequests || 
			   resp.StatusCode >= 500 {
				// More aggressive backoff for stronger anti-bot sites
				if isArbitrumOrBase {
					time.Sleep(time.Duration(800*(attempt+1)) * time.Millisecond)
				} else {
					time.Sleep(time.Duration(300*(attempt+1)) * time.Millisecond)
				}
				continue
			}
			
			// If we get here, we're not going to retry, so release the proxy if we were using one
			if usingProxy && currentProxy != nil {
				c.proxyManager.ReleaseProxy(currentProxy, false)
			}
			
			return "", lastErr
		}
		
		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			if usingProxy && currentProxy != nil {
				c.proxyManager.ReleaseProxy(currentProxy, false)
			}
			return "", fmt.Errorf("error reading response body: %v", err)
		}
		
		// If there's any indication of Cloudflare or other protection in the HTML,
		// we might need to retry with a different approach
		if isArbitrumOrBase && (bytes.Contains(body, []byte("Cloudflare")) || 
		   bytes.Contains(body, []byte("challenge")) || 
		   bytes.Contains(body, []byte("captcha"))) {
			lastErr = fmt.Errorf("detected bot protection page")
			
			// If not already using proxy, enable proxy mode
			if !usingProxy && c.proxyManager != nil && c.proxyManager.IsEnabled() {
				c.logger.Info("Bot protection detected! Switching to proxy mode...")
				SetRuntimeValue("RATE_LIMIT_HIT", "true")
				
				// Try to get a proxy
				proxy, err := c.proxyManager.GetNextProxy()
				if err != nil || proxy == nil {
					c.logger.Debug(fmt.Sprintf("Failed to get proxy after protection detection: %v", err))
				} else {
					currentProxy = proxy
					proxyClient, err = c.proxyManager.GetHttpClient(proxy)
					if err != nil {
						c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
						c.proxyManager.ReleaseProxy(proxy, false)
						currentProxy = nil
					} else {
						usingProxy = true
						c.logger.Info(fmt.Sprintf("Switched to proxy after protection detection: %s", proxy.URL))
					}
				}
			} else if usingProxy && currentProxy != nil {
				// If using a proxy, try a different one
				c.proxyManager.ReleaseProxy(currentProxy, false)
				// Try to get a new proxy for the next attempt
				proxy, err := c.proxyManager.GetNextProxy()
				if err != nil || proxy == nil {
					c.logger.Debug("Failed to get a new proxy, falling back to direct connection")
					usingProxy = false
					currentProxy = nil
				} else {
					currentProxy = proxy
					proxyClient, err = c.proxyManager.GetHttpClient(proxy)
					if err != nil {
						c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
						c.proxyManager.ReleaseProxy(proxy, false)
						usingProxy = false
						currentProxy = nil
					} else {
						c.logger.Debug(fmt.Sprintf("Switched to proxy: %s", proxy.URL))
					}
				}
			}
			
			time.Sleep(time.Duration(800*(attempt+1)) * time.Millisecond)
			continue
		}
		
		// If we got here, the request was successful
		if usingProxy && currentProxy != nil {
			c.proxyManager.ReleaseProxy(currentProxy, true)
		}
		
		return string(body), nil
	}
	
	// If we get here, we've exhausted all retries, so release the proxy if we were using one
	if usingProxy && currentProxy != nil {
		c.proxyManager.ReleaseProxy(currentProxy, false)
	}
	
	return "", fmt.Errorf("maximum retries reached: %v", lastErr)
}

// Post performs an HTTP POST request with a customizable user agent and body
func (c *HTTPClient) Post(url, userAgent, contentType string, body []byte) (string, error) {
	maxRetries := 3
	var lastErr error
	
	// If we have a proxy manager, check if we should use it
	var currentProxy *Proxy
	var proxyClient *http.Client
	var usingProxy bool
	
	if c.proxyManager != nil && c.proxyManager.IsEnabled() {
		// Check if we've hit rate limits yet
		rateLimitHit, _ := GetRuntimeBool("RATE_LIMIT_HIT")
		
		// Only use proxies if rate limits have been hit
		if rateLimitHit {
			// Get a proxy
			proxy, err := c.proxyManager.GetNextProxy()
			if err != nil {
				c.logger.Debug(fmt.Sprintf("Failed to get proxy: %v", err))
			} else if proxy != nil {
				currentProxy = proxy
				proxyClient, err = c.proxyManager.GetHttpClient(proxy)
				if err != nil {
					c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
					c.proxyManager.ReleaseProxy(proxy, false)
					currentProxy = nil
				} else {
					usingProxy = true
					c.logger.Debug(fmt.Sprintf("Using proxy: %s", proxy.URL))
				}
			}
		}
	}
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Random delay between 50-150ms to make requests look more human but faster overall
		randomDelay := 50 + (time.Now().UnixNano() % 100)
		time.Sleep(time.Duration(randomDelay) * time.Millisecond)
		
		// Create a new request with the provided body
		bodyReader := bytes.NewReader(body)
		req, err := http.NewRequest("POST", url, bodyReader)
		if err != nil {
			if currentProxy != nil {
				c.proxyManager.ReleaseProxy(currentProxy, false)
			}
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
		
		// Perform the request using either the proxy client or the default client
		var resp *http.Response
		var reqErr error
		if usingProxy {
			resp, reqErr = proxyClient.Do(req)
		} else {
			resp, reqErr = c.client.Do(req)
		}
		
		if reqErr != nil {
			lastErr = fmt.Errorf("error performing request: %v", reqErr)
			
			// If using proxy and request failed, try a different proxy
			if usingProxy && currentProxy != nil {
				c.proxyManager.ReleaseProxy(currentProxy, false)
				// Try to get a new proxy for the next attempt
				proxy, err := c.proxyManager.GetNextProxy()
				if err != nil || proxy == nil {
					c.logger.Debug("Failed to get a new proxy, falling back to direct connection")
					usingProxy = false
					currentProxy = nil
				} else {
					currentProxy = proxy
					proxyClient, err = c.proxyManager.GetHttpClient(proxy)
					if err != nil {
						c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
						c.proxyManager.ReleaseProxy(proxy, false)
						usingProxy = false
						currentProxy = nil
					} else {
						c.logger.Debug(fmt.Sprintf("Switched to proxy: %s", proxy.URL))
					}
				}
			}
			
			time.Sleep(time.Duration(300*(attempt+1)) * time.Millisecond)
			continue
		}
		defer resp.Body.Close()
		
		// Check status code
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			
			// Check if this is a rate limit response (429 Too Many Requests or 403 Forbidden)
			if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden) {
				// Detect rate limit and switch to proxy mode if we're not already using one
				if !usingProxy && c.proxyManager != nil && c.proxyManager.IsEnabled() {
					c.logger.Info("Rate limit detected! Switching to proxy mode...")
					SetRuntimeValue("RATE_LIMIT_HIT", "true")
					
					// Get a proxy for the next attempt
					proxy, err := c.proxyManager.GetNextProxy()
					if err != nil || proxy == nil {
						c.logger.Debug(fmt.Sprintf("Failed to get proxy after rate limit: %v", err))
					} else {
						currentProxy = proxy
						proxyClient, err = c.proxyManager.GetHttpClient(proxy)
						if err != nil {
							c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
							c.proxyManager.ReleaseProxy(proxy, false)
							currentProxy = nil
						} else {
							usingProxy = true
							c.logger.Info(fmt.Sprintf("Switched to proxy after rate limit: %s", proxy.URL))
						}
					}
				}
			}
			
			// If using proxy and got a bad status, try a different proxy
			if usingProxy && currentProxy != nil && (resp.StatusCode == http.StatusForbidden || 
			   resp.StatusCode == http.StatusTooManyRequests) {
				c.proxyManager.ReleaseProxy(currentProxy, false)
				// Try to get a new proxy for the next attempt
				proxy, err := c.proxyManager.GetNextProxy()
				if err != nil || proxy == nil {
					c.logger.Debug("Failed to get a new proxy, falling back to direct connection")
					usingProxy = false
					currentProxy = nil
				} else {
					currentProxy = proxy
					proxyClient, err = c.proxyManager.GetHttpClient(proxy)
					if err != nil {
						c.logger.Debug(fmt.Sprintf("Failed to create proxy client: %v", err))
						c.proxyManager.ReleaseProxy(proxy, false)
						usingProxy = false
						currentProxy = nil
					} else {
						c.logger.Debug(fmt.Sprintf("Switched to proxy: %s", proxy.URL))
					}
				}
			}
			
			// Only retry certain status codes
			if resp.StatusCode == http.StatusForbidden || 
			   resp.StatusCode == http.StatusTooManyRequests || 
			   resp.StatusCode >= 500 {
				time.Sleep(time.Duration(300*(attempt+1)) * time.Millisecond)
				continue
			}
			
			// If we get here, we're not going to retry, so release the proxy if we were using one
			if usingProxy && currentProxy != nil {
				c.proxyManager.ReleaseProxy(currentProxy, false)
			}
			
			return "", lastErr
		}
		
		// Read the response body
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			if usingProxy && currentProxy != nil {
				c.proxyManager.ReleaseProxy(currentProxy, false)
			}
			return "", fmt.Errorf("error reading response body: %v", err)
		}
		
		// If we got here, the request was successful
		if usingProxy && currentProxy != nil {
			c.proxyManager.ReleaseProxy(currentProxy, true)
		}
		
		return string(responseBody), nil
	}
	
	// If we get here, we've exhausted all retries, so release the proxy if we were using one
	if usingProxy && currentProxy != nil {
		c.proxyManager.ReleaseProxy(currentProxy, false)
	}
	
	return "", fmt.Errorf("maximum retries reached: %v", lastErr)
}

// SetTimeout sets the timeout for the HTTP client
func (c *HTTPClient) SetTimeout(timeout time.Duration) {
	c.client.Timeout = timeout
}