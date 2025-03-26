package explorer

import (
        "strings"
)

// ChainInfo contains information about an EVM compatible blockchain
type ChainInfo struct {
        Name           string
        ExplorerURL    string
        AddressURL     string
        BalancePattern string
        UserAgent      string
        ExtraDelay     int    // Additional delay in milliseconds for this specific chain
        Enabled        bool   // Whether this chain is enabled
}

// List of supported EVM chains with their explorer URLs
var supportedChains = []ChainInfo{
        {
                Name:           "ethereum",
                ExplorerURL:    "https://etherscan.io",
                AddressURL:     "https://etherscan.io/address/%s",
                // More flexible pattern that works with different variations of Etherscan display
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) ETH</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
                ExtraDelay:     0,
                Enabled:        true,
        },
        {
                Name:           "binance",
                ExplorerURL:    "https://bscscan.com",
                AddressURL:     "https://bscscan.com/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) BNB</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
                ExtraDelay:     0,
                Enabled:        true,
        },
        {
                Name:           "polygon",
                ExplorerURL:    "https://polygonscan.com",
                AddressURL:     "https://polygonscan.com/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) MATIC</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
                ExtraDelay:     0,
                Enabled:        true,
        },
        {
                Name:           "fantom",
                ExplorerURL:    "https://ftmscan.com",
                AddressURL:     "https://ftmscan.com/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) FTM</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
                ExtraDelay:     0,
                Enabled:        true,
        },
        {
                Name:           "avalanche",
                ExplorerURL:    "https://snowtrace.io",
                AddressURL:     "https://snowtrace.io/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) AVAX</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
                ExtraDelay:     0,
                Enabled:        true,
        },
        {
                Name:           "optimism",
                ExplorerURL:    "https://optimistic.etherscan.io",
                AddressURL:     "https://optimistic.etherscan.io/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) ETH</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
                ExtraDelay:     0,
                Enabled:        true,
        },
        {
                Name:           "arbitrum",
                ExplorerURL:    "https://arbiscan.io",
                AddressURL:     "https://arbiscan.io/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) ETH</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
                ExtraDelay:     1000, // Extra 1 second delay for this chain
                Enabled:        false, // Temporarily disable due to 403 errors
        },
        {
                Name:           "celo",
                ExplorerURL:    "https://celoscan.io",
                AddressURL:     "https://celoscan.io/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) CELO</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
                ExtraDelay:     0,
                Enabled:        true,
        },
        {
                Name:           "base",
                ExplorerURL:    "https://basescan.org",
                AddressURL:     "https://basescan.org/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) ETH</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
                ExtraDelay:     1000, // Extra 1 second delay for this chain
                Enabled:        false, // Temporarily disable due to 403 errors
        },
}

// GetChainList returns a list of ChainInfo based on comma-separated chain names
// If "all" is specified, all supported and enabled chains are returned
func GetChainList(chainsArg string) []ChainInfo {
        // Filter to only include enabled chains
        var enabledChains []ChainInfo
        for _, chain := range supportedChains {
                if chain.Enabled {
                        enabledChains = append(enabledChains, chain)
                }
        }
        
        if chainsArg == "all" {
                return enabledChains
        }
        
        chainNames := strings.Split(chainsArg, ",")
        var selectedChains []ChainInfo
        
        for _, name := range chainNames {
                name = strings.TrimSpace(strings.ToLower(name))
                for _, chain := range enabledChains {
                        if chain.Name == name {
                                selectedChains = append(selectedChains, chain)
                                break
                        }
                }
        }
        
        // If no valid chains were selected, return all enabled chains
        if len(selectedChains) == 0 {
                return enabledChains
        }
        
        return selectedChains
}

// GetChainsByNames returns a list of ChainInfo based on exact chain names
func GetChainsByNames(chainNames []string) []ChainInfo {
        var selectedChains []ChainInfo
        
        // Create a map for fast lookups
        chainMap := make(map[string]ChainInfo)
        for _, chain := range supportedChains {
                chainMap[chain.Name] = chain
        }
        
        // Select chains by name
        for _, name := range chainNames {
                name = strings.TrimSpace(strings.ToLower(name))
                if chain, ok := chainMap[name]; ok {
                        // Override the built-in enabled flag with what's in the config
                        chain.Enabled = true
                        selectedChains = append(selectedChains, chain)
                }
        }
        
        // If no valid chains were selected, return all enabled chains
        if len(selectedChains) == 0 {
                for _, chain := range supportedChains {
                        if chain.Enabled {
                                selectedChains = append(selectedChains, chain)
                        }
                }
        }
        
        return selectedChains
}
