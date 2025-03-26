package explorer

import (
        "fmt"
        "regexp"
        "strconv"
        "strings"
        "sync"
        "time"

        "cryptowallet/utils"
        "cryptowallet/wallet"
)

// BalanceChecker checks wallet balances across blockchain explorers
type BalanceChecker struct {
        requestDelay    int
        chains          []ChainInfo
        httpClient      *utils.HTTPClient
        logger          *utils.Logger
        proxyManager    *utils.ProxyManager
        rateLimitedChains map[string]time.Time  // Map tracking which chains are rate limited and when to retry
        rateLimitMutex   sync.RWMutex           // Mutex for thread-safe access to rate limit map
}

// NewBalanceChecker creates a new balance checker instance
func NewBalanceChecker(requestDelay int, chains []ChainInfo, logger *utils.Logger) *BalanceChecker {
        client := utils.NewHTTPClient()
        return &BalanceChecker{
                requestDelay:      requestDelay,
                chains:            chains,
                httpClient:        client,
                logger:            logger,
                proxyManager:      nil,
                rateLimitedChains: make(map[string]time.Time),
                rateLimitMutex:    sync.RWMutex{},
        }
}

// SetProxyManager sets the proxy manager for the balance checker
func (bc *BalanceChecker) SetProxyManager(proxyManager *utils.ProxyManager) {
        bc.proxyManager = proxyManager
        bc.httpClient.SetProxyManager(proxyManager, bc.logger)
}

// CheckWalletBalances checks a wallet's balance across multiple chains
func (bc *BalanceChecker) CheckWalletBalances(w wallet.Wallet) []wallet.WalletWithBalance {
        var results []wallet.WalletWithBalance
        
        // Initialize with empty results for each chain
        for _, chain := range bc.chains {
                results = append(results, wallet.WalletWithBalance{
                        Address:    w.Address,
                        PrivateKey: w.PrivateKey,
                        Chain:      chain.Name,
                        Balance:    "0",
                        HasBalance: false,
                })
        }
        
        // Create a wait group to check chains in parallel
        var wg sync.WaitGroup
        resultsMutex := &sync.Mutex{}
        
        // Check each chain in parallel, but skip rate-limited ones
        for i, chain := range bc.chains {
            // Skip this chain if it's currently rate-limited
            bc.rateLimitMutex.RLock()
            retryTime, isRateLimited := bc.rateLimitedChains[chain.Name]
            bc.rateLimitMutex.RUnlock()
            
            // If the chain is rate-limited but the cool-down period has elapsed, remove it
            if isRateLimited && time.Now().After(retryTime) {
                bc.rateLimitMutex.Lock()
                delete(bc.rateLimitedChains, chain.Name)
                bc.rateLimitMutex.Unlock()
                isRateLimited = false
            }
            
            // Skip this chain if it's still rate-limited
            if isRateLimited {
                continue
            }
            
            wg.Add(1)
            go func(idx int, c ChainInfo) {
                defer wg.Done()
                // Add a tiny delay to stagger requests slightly
                time.Sleep(time.Duration(bc.requestDelay/10) * time.Millisecond)
                
                result := bc.checkBalanceOnChain(w, c)
                
                // Update results
                resultsMutex.Lock()
                results[idx] = result
                resultsMutex.Unlock()
            }(i, chain)
        }
        
        // Wait for all checks to complete
        wg.Wait()
        
        return results
}

// checkBalanceOnChain checks a wallet's balance on a specific blockchain
func (bc *BalanceChecker) checkBalanceOnChain(w wallet.Wallet, chain ChainInfo) wallet.WalletWithBalance {
        // First, validate the address for this specific chain type
        if !bc.IsValidAddress(w.Address, chain) {
            // Skip checking chains if the address format doesn't match the chain type
            return wallet.WalletWithBalance{
                Address:    w.Address,
                PrivateKey: w.PrivateKey,
                Chain:      chain.Name,
                Balance:    "0",
                HasBalance: false,
            }
        }
        
        // Create the URL for the address on this explorer
        url := fmt.Sprintf(chain.AddressURL, w.Address)
        
        // Set up the result with default values
        result := wallet.WalletWithBalance{
                Address:    w.Address,
                PrivateKey: w.PrivateKey,
                Chain:      chain.Name,
                Balance:    "0",
                HasBalance: false,
        }
        
        // Apply chain-specific extra delay if needed, but only in debug mode
        // In normal operation, we skip this for maximum speed
        if chain.ExtraDelay > 0 && bc.logger.IsDebugEnabled() {
                time.Sleep(time.Duration(chain.ExtraDelay) * time.Millisecond)
        }
        
        // Make the HTTP request with optimized error handling
        html, err := bc.httpClient.Get(url, chain.UserAgent)
        if err != nil {
                // Check if it's a rate limit error (status code 429 or other indicators)
                if strings.Contains(err.Error(), "429") || 
                   strings.Contains(err.Error(), "too many requests") ||
                   strings.Contains(err.Error(), "rate limit") {
                    // Temporarily disable this chain for 60 seconds
                    bc.rateLimitMutex.Lock()
                    bc.rateLimitedChains[chain.Name] = time.Now().Add(60 * time.Second)
                    bc.rateLimitMutex.Unlock()
                    
                    // Log the rate limit once at WARN level (not DEBUG)
                    bc.logger.Warn(fmt.Sprintf("🚫 Rate limit hit on %s chain - disabling for 60 seconds", chain.Name))
                }
                return result
        }
        
        // Parse the balance from the HTML - skip excessive logging for better performance
        balance, err := bc.parseBalance(html, chain.BalancePattern)
        if err != nil {
                // No need to log zero balances, they're the vast majority
                return result
        }
        
        // Parse the balance as a float to check if it's greater than zero
        balanceFloat, err := strconv.ParseFloat(balance, 64)
        if err != nil {
                // Only log in debug mode
                bc.logger.Debug(fmt.Sprintf("Error parsing balance '%s' as float: %v", balance, err))
                return result
        }
        
        // Update the result
        result.Balance = balance
        result.HasBalance = balanceFloat > 0
        
        // Set the chain type based on whether this is an EVM chain or not
        if chain.IsEVM {
            result.ChainType = "evm"
        } else if chain.Name == "bitcoin" {
            result.ChainType = "bitcoin"
        }
        
        // If balance is found, it will be shown in the main output, 
        // no need to duplicate the log here
        
        return result
}

// parseBalance extracts the balance from HTML using a regex pattern
func (bc *BalanceChecker) parseBalance(html, pattern string) (string, error) {
        // Try to match the balance pattern
        re := regexp.MustCompile(pattern)
        matches := re.FindStringSubmatch(html)
        
        if len(matches) < 2 {
                // Alternative approach: try simpler parsing
                return bc.fallbackBalanceParsing(html)
        }
        
        return matches[1], nil
}

// fallbackBalanceParsing tries a more generic approach to find balances
func (bc *BalanceChecker) fallbackBalanceParsing(html string) (string, error) {
        // Modern etherscan-family patterns
        modernPatterns := []string{
                // Modern etherscan pattern with text-$ class (most common now)
                `<div class="card-body">[\s\S]*?<span class="text-[$][^"]*">(\d+\.\d+) [A-Z]+</span>`,
                // Alternative modern pattern
                `<div[^>]*>[^<]*Balance[\s\S]*?<span[^>]*>(\d+\.\d+) [A-Z]+</span>`,
        }
        
        for _, pattern := range modernPatterns {
                re := regexp.MustCompile(pattern)
                matches := re.FindStringSubmatch(html)
                
                if len(matches) >= 2 {
                        return matches[1], nil
                }
        }
        
        // Legacy patterns for older etherscan versions
        legacyPatterns := []string{
                // Older column-based layout
                `<div class="col-md-8">(\d+\.\d+) [A-Z]+</div>`,
                // Other common formats
                `Balance:</span>\s*(\d+\.\d+)`,
                `Balance</div>\s*<div[^>]*>(\d+\.\d+)`,
                `Balance:\s*(\d+\.\d+)`,
                `data-balance=['"](\d+\.\d+)['"]`,
                // Table-based layouts
                `<td[^>]*>(\d+\.\d+) [A-Z]+</td>`,
        }
        
        for _, pattern := range legacyPatterns {
                re := regexp.MustCompile(pattern)
                matches := re.FindStringSubmatch(html)
                
                if len(matches) >= 2 {
                        return matches[1], nil
                }
        }
        
        // Check for any number that might be a balance near "Balance" text
        // This is a last resort approach
        balancePattern := `Balance[^<>]*?(\d+\.\d+)`
        re := regexp.MustCompile(balancePattern)
        matches := re.FindStringSubmatch(html)
        
        if len(matches) >= 2 {
                return matches[1], nil
        }
        
        // Check if the page indicates zero balance
        zeroIndicators := []string{
                "0 ETH", "0 BNB", "0 MATIC", "0 FTM", "0 AVAX", "0 CELO", "0 ETH",
                "Balance: 0", "Balance: 0.0", "<span class=\"text-$[^\"]>0</span>",
                "0 Token", "0 Tokens", "Balance</div>\\s*<div[^>]*>0<",
        }
        
        for _, zero := range zeroIndicators {
                if strings.Contains(html, zero) {
                        return "0", nil
                }
        }
        
        return "", fmt.Errorf("could not find balance in the HTML")
}

// IsValidAddress checks if an address is valid for the specified chain
func (bc *BalanceChecker) IsValidAddress(address string, chain ChainInfo) bool {
        if chain.IsEVM {
                // EVM addresses are 42 characters (0x + 40 hex characters)
                if len(address) != 42 || !strings.HasPrefix(address, "0x") {
                        return false
                }
                
                // Check if the address contains only hex characters after 0x
                hexPart := address[2:]
                for _, c := range hexPart {
                        if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
                                return false
                        }
                }
                
                return true
        } else if chain.Name == "bitcoin" {
                // Bitcoin addresses can be legacy (P2PKH), P2SH, or Bech32 (SegWit)
                
                // Legacy bitcoin addresses start with 1 and are 26-35 characters
                if strings.HasPrefix(address, "1") && len(address) >= 26 && len(address) <= 35 {
                        return true
                }
                
                // P2SH addresses start with 3 and are 26-35 characters 
                if strings.HasPrefix(address, "3") && len(address) >= 26 && len(address) <= 35 {
                        return true
                }
                
                // Bech32 (SegWit) addresses start with bc1 and are longer
                if strings.HasPrefix(address, "bc1") && len(address) >= 14 && len(address) <= 74 {
                        return true
                }
                
                return false
        }
        
        // If it's not a known chain type, be permissive
        return true
}

// IsValidAddressForAnyChain checks if an address is valid for any supported chain
func (bc *BalanceChecker) IsValidAddressForAnyChain(address string) bool {
        // Check if it's valid for at least one chain
        for _, chain := range bc.chains {
                if bc.IsValidAddress(address, chain) {
                        return true
                }
        }
        
        return false
}
