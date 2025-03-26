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
        },
        {
                Name:           "binance",
                ExplorerURL:    "https://bscscan.com",
                AddressURL:     "https://bscscan.com/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) BNB</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        },
        {
                Name:           "polygon",
                ExplorerURL:    "https://polygonscan.com",
                AddressURL:     "https://polygonscan.com/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) MATIC</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        },
        {
                Name:           "fantom",
                ExplorerURL:    "https://ftmscan.com",
                AddressURL:     "https://ftmscan.com/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) FTM</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        },
        {
                Name:           "avalanche",
                ExplorerURL:    "https://snowtrace.io",
                AddressURL:     "https://snowtrace.io/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) AVAX</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        },
        {
                Name:           "optimism",
                ExplorerURL:    "https://optimistic.etherscan.io",
                AddressURL:     "https://optimistic.etherscan.io/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) ETH</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        },
        {
                Name:           "arbitrum",
                ExplorerURL:    "https://arbiscan.io",
                AddressURL:     "https://arbiscan.io/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) ETH</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        },
        {
                Name:           "celo",
                ExplorerURL:    "https://celoscan.io",
                AddressURL:     "https://celoscan.io/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) CELO</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        },
        {
                Name:           "base",
                ExplorerURL:    "https://basescan.org",
                AddressURL:     "https://basescan.org/address/%s",
                BalancePattern: `(?:<div class="card-body">|<span class="text-muted">Balance</span>)[\s\S]*?<span[^>]*>(\d+(?:\.\d+)?) ETH</span>`,
                UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        },
}

// GetChainList returns a list of ChainInfo based on comma-separated chain names
// If "all" is specified, all supported chains are returned
func GetChainList(chainsArg string) []ChainInfo {
        if chainsArg == "all" {
                return supportedChains
        }
        
        chainNames := strings.Split(chainsArg, ",")
        var selectedChains []ChainInfo
        
        for _, name := range chainNames {
                name = strings.TrimSpace(strings.ToLower(name))
                for _, chain := range supportedChains {
                        if chain.Name == name {
                                selectedChains = append(selectedChains, chain)
                                break
                        }
                }
        }
        
        // If no valid chains were selected, return all
        if len(selectedChains) == 0 {
                return supportedChains
        }
        
        return selectedChains
}
