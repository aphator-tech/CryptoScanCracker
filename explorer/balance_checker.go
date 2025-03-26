package explorer

import (
        "fmt"
        "regexp"
        "strconv"
        "strings"
        "time"

        "cryptowallet/utils"
        "cryptowallet/wallet"
)

// BalanceChecker checks wallet balances across blockchain explorers
type BalanceChecker struct {
        requestDelay int
        chains       []ChainInfo
        httpClient   *utils.HTTPClient
        logger       *utils.Logger
}

// NewBalanceChecker creates a new balance checker instance
func NewBalanceChecker(requestDelay int, chains []ChainInfo, logger *utils.Logger) *BalanceChecker {
        return &BalanceChecker{
                requestDelay: requestDelay,
                chains:       chains,
                httpClient:   utils.NewHTTPClient(),
                logger:       logger,
        }
}

// CheckWalletBalances checks a wallet's balance across multiple chains
func (bc *BalanceChecker) CheckWalletBalances(w wallet.Wallet) []wallet.WalletWithBalance {
        var results []wallet.WalletWithBalance
        
        for _, chain := range bc.chains {
                result := bc.checkBalanceOnChain(w, chain)
                results = append(results, result)
                
                // Rate limiting
                time.Sleep(time.Duration(bc.requestDelay) * time.Millisecond)
        }
        
        return results
}

// checkBalanceOnChain checks a wallet's balance on a specific blockchain
func (bc *BalanceChecker) checkBalanceOnChain(w wallet.Wallet, chain ChainInfo) wallet.WalletWithBalance {
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
        
        // Make the HTTP request
        bc.logger.Debug(fmt.Sprintf("Checking balance on %s: %s", chain.Name, w.Address))
        
        html, err := bc.httpClient.Get(url, chain.UserAgent)
        if err != nil {
                bc.logger.Error(fmt.Sprintf("Error checking balance on %s: %v", chain.Name, err))
                return result
        }
        
        // Parse the balance from the HTML
        balance, err := bc.parseBalance(html, chain.BalancePattern)
        if err != nil {
                bc.logger.Debug(fmt.Sprintf("No balance found on %s: %v", chain.Name, err))
                return result
        }
        
        // Parse the balance as a float to check if it's greater than zero
        balanceFloat, err := strconv.ParseFloat(balance, 64)
        if err != nil {
                bc.logger.Error(fmt.Sprintf("Error parsing balance '%s' as float: %v", balance, err))
                return result
        }
        
        // Update the result
        result.Balance = balance
        result.HasBalance = balanceFloat > 0
        
        if result.HasBalance {
                bc.logger.Info(fmt.Sprintf("Found wallet with balance on %s: %s = %s", 
                        chain.Name, w.Address, balance))
        }
        
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
        // Common balance indicators in blockchain explorers
        balanceIndicators := []string{
                `Balance:</span>\s*(\d+\.\d+)`,
                `Balance</div>\s*<div[^>]*>(\d+\.\d+)`,
                `Balance:\s*(\d+\.\d+)`,
                `data-balance=['"](\d+\.\d+)['"]`,
        }
        
        // Try each pattern
        for _, pattern := range balanceIndicators {
                re := regexp.MustCompile(pattern)
                matches := re.FindStringSubmatch(html)
                
                if len(matches) >= 2 {
                        return matches[1], nil
                }
        }
        
        // Check for any number that might be a balance
        // This is a last resort approach
        balancePattern := `Balance[^<>]*?(\d+\.\d+)`
        re := regexp.MustCompile(balancePattern)
        matches := re.FindStringSubmatch(html)
        
        if len(matches) >= 2 {
                return matches[1], nil
        }
        
        // Check if the page indicates zero balance
        zeroIndicators := []string{
                "0 ETH", "0 BNB", "0 MATIC", "0 FTM", "0 AVAX", "0 CELO",
                "Balance: 0", "Balance: 0.0",
        }
        
        for _, zero := range zeroIndicators {
                if strings.Contains(html, zero) {
                        return "0", nil
                }
        }
        
        return "", fmt.Errorf("could not find balance in the HTML")
}

// IsValidAddress checks if an address is a valid EVM address
func (bc *BalanceChecker) IsValidAddress(address string) bool {
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
}
