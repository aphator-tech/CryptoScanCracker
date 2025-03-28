package main

import (
        "flag"
        "fmt"
        "os"
        "os/signal"
        "runtime"
        "strings"
        "sync"
        "syscall"
        "time"

        "cryptowallet/explorer"
        "cryptowallet/storage"
        "cryptowallet/utils"
        "cryptowallet/wallet"
)

// Command line flags
var (
        numWallets      = flag.Int("wallets", 100, "Number of wallets to generate and check per batch (will run continuously)")
        batchSize       = flag.Int("batch", 10, "Batch size for concurrent wallet checking")
        requestDelay    = flag.Int("delay", 20, "Delay between requests in milliseconds (lower = faster)")
        outputFile      = flag.String("output", "wallets_with_balance.json", "Output JSON file for wallets with balance")
        maxGoroutines   = flag.Int("goroutines", 50, "Maximum number of concurrent goroutines (higher = faster)")
        logLevel        = flag.String("log", "info", "Log level (debug, info, warn, error)")
        selectedChains  = flag.String("chains", "all", "Comma-separated list of chains to check (or 'all')")
        infiniteMode    = flag.Bool("infinite", true, "Run in infinite mode until stopped")
)

func main() {
        flag.Parse()
        
        // Setup logger - force to be less verbose, only showing balances and critical errors
        // We're overriding the log level to make output cleaner
        if *logLevel != "debug" {
            *logLevel = "warn" // Only show warnings, errors, and balance results
        }
        logger := utils.NewLogger(*logLevel)
        logger.Info(utils.ColorCyan("💼 Crypto Wallet Balance Checker Started"))
        
        // Setup signal handling for graceful shutdown
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        
        // Initialize JSON store
        store := storage.NewJSONStore(*outputFile)
        
        // Parse chains to check - use env.txt settings if available
        var chainNames []string
        if useEnvSettings, ok := utils.ReadEnvBool("USE_ENV_CHAINS"); ok && useEnvSettings {
            // Get chain list from env.txt
            logger.Info("Using chain configuration from env.txt")
            chainNames = getEnabledChainsFromEnv(logger)
        } else {
            // Use command line arguments for chain names
            logger.Info(fmt.Sprintf("Using command line chains: %s", *selectedChains))
            
            if strings.TrimSpace(*selectedChains) != "" && *selectedChains != "all" {
                // Parse comma-separated chain names from command line
                for _, name := range strings.Split(*selectedChains, ",") {
                    chainNames = append(chainNames, strings.TrimSpace(strings.ToLower(name)))
                }
            } else {
                // Use GetChainList if selectedChains is "all" or empty
                for _, chain := range explorer.GetChainList(*selectedChains) {
                    chainNames = append(chainNames, chain.Name)
                }
            }
        }
        
        if len(chainNames) == 0 {
            // Just use a subset of chains to avoid rate limiting and improve performance
            logger.Warn("Using a subset of faster chains to avoid rate limits - selecting reliable chains only")
            // Only use chains that don't rate limit as much, including Bitcoin
            chainNames = []string{"bitcoin", "binance", "polygon", "avalanche", "fantom", "celo"}
        }
        
        // Convert chain names to ChainInfo objects
        logger.Info(fmt.Sprintf("Attempting to get ChainInfo for chains: %v", chainNames))
        chainList := explorer.GetChainsByNames(chainNames)
        
        logger.Info(fmt.Sprintf("Checking balances on %d chains: %v", len(chainList), getChainNames(chainList)))
        
        // Initialize wallet generator
        generator := wallet.NewGenerator(logger)
        
        // Initialize proxy manager if enabled
        var proxyManager *utils.ProxyManager
        if useProxies, ok := utils.ReadEnvBool("USE_PROXIES"); ok && useProxies {
            proxyUrl, proxyOk := utils.ReadEnv("PROXY_URL")
            if proxyOk && proxyUrl != "" {
                logger.Info("Initializing proxy support...")
                proxyManager = utils.NewProxyManager(proxyUrl, true, logger)
                proxyCount := proxyManager.GetProxyCount()
                if proxyCount > 0 {
                    logger.Info(fmt.Sprintf("Successfully loaded %d proxies", proxyCount))
                } else {
                    logger.Warn("Failed to load proxies, continuing without proxies")
                    proxyManager = nil
                }
            } else {
                logger.Warn("Proxy support enabled but no PROXY_URL specified, continuing without proxies")
            }
        }
        
        // Initialize balance checker with proxy support and faster request delay
        balanceChecker := explorer.NewBalanceChecker(
                *requestDelay,  // Use command line delay parameter
                chainList,
                logger,
        )
        
        // Set proxy manager if available
        if proxyManager != nil {
            balanceChecker.SetProxyManager(proxyManager)
        }
        
        // Setup worker pool - use more workers for better performance
        numCores := runtime.NumCPU()
        maxWorkers := *maxGoroutines
        
        // Check if we have a value in env.txt
        if maxConcurrent, ok := utils.ReadEnvInt("MAX_CONCURRENT_PROXIES"); ok && proxyManager != nil {
            // Use the configured value for proxies
            maxWorkers = maxConcurrent
            logger.Info(fmt.Sprintf("Using %d workers from env.txt configuration", maxWorkers))
        } else if *maxGoroutines <= 0 {
            // Default to 4x cores if not specified for faster processing
            maxWorkers = numCores * 4
        }
        
        logger.Info(fmt.Sprintf("Using %d worker goroutines", maxWorkers))
        
        // Create work channels with larger buffers for better throughput
        walletChan := make(chan wallet.Wallet, *batchSize * 4)
        resultChan := make(chan wallet.WalletWithBalance, *batchSize * 4)
        done := make(chan struct{})
        
        // Start worker pool
        var wg sync.WaitGroup
        for i := 0; i < maxWorkers; i++ {
                wg.Add(1)
                go func() {
                        defer wg.Done()
                        for w := range walletChan {
                                walletWithBalances := balanceChecker.CheckWalletBalances(w)
                                hasAnyBalance := false
                                
                                for _, wb := range walletWithBalances {
                                        if wb.HasBalance {
                                                hasAnyBalance = true
                                                resultChan <- wb
                                        }
                                }
                                
                                // Print wallet check result with timestamp
                                timestamp := time.Now().Format("15:04:05")
                                if hasAnyBalance {
                                        fmt.Printf("[%s] %s - %s\n", 
                                                timestamp, 
                                                utils.ColorYellow(w.Address), 
                                                utils.ColorGreen("✅ BALANCE FOUND!"))
                                } else {
                                        fmt.Printf("[%s] %s - %s\n", 
                                                timestamp, 
                                                utils.ColorYellow(w.Address), 
                                                utils.ColorRed("❌ No balance"))
                                }
                        }
                }()
        }
        
        // Start result handler with colorful, simplified output
        go func() {
                for result := range resultChan {
                        // Use colorful output with emoji indicators for wallet type
                        
                        // Choose emoji based on chain type
                        walletEmoji := "💰" // Default emoji
                        
                        // Add special emojis for different chain types
                        if result.ChainType == "bitcoin" {
                            walletEmoji = "₿"  // Bitcoin symbol
                        } else if strings.EqualFold(result.Chain, "ethereum") {
                            walletEmoji = "Ξ"  // Ethereum symbol
                        } else if strings.EqualFold(result.Chain, "binance") {
                            walletEmoji = "🟨" // Yellow for Binance
                        } else if strings.EqualFold(result.Chain, "polygon") {
                            walletEmoji = "🟪" // Purple for Polygon
                        } else if strings.EqualFold(result.Chain, "avalanche") {
                            walletEmoji = "🔺" // Red triangle for Avalanche
                        } else if strings.EqualFold(result.Chain, "fantom") {
                            walletEmoji = "👻" // Ghost for Fantom
                        }
                        
                        // Green for the chain name, yellow for the address, and cyan for the balance
                        fmt.Printf("%s %s: %s = %s\n", 
                                walletEmoji,
                                utils.ColorGreen(result.Chain), 
                                utils.ColorYellow(result.Address), 
                                utils.ColorCyan(result.Balance))
                        
                        store.AddWallet(result)
                }
                close(done)
        }()
        
        // Start wallet generation and checking
        if *infiniteMode {
                logger.Info("Running in infinite mode - will continue until manually stopped")
        } else {
                logger.Info(fmt.Sprintf("Generating and checking %d wallets", *numWallets))
        }
        
        walletsProcessed := 0
        walletsWithBalance := 0
        targetWallets := *numWallets
        
        // Process wallet generation in batches
        batchNum := 0
        
        // Main loop - either runs until we reach the target, or forever in infinite mode
        for *infiniteMode || walletsProcessed < targetWallets {
                select {
                case <-sigChan:
                        logger.Info("Received interrupt signal, shutting down...")
                        goto cleanup
                default:
                        // In infinite mode, always process full batches
                        var currentBatchSize int
                        if *infiniteMode {
                                currentBatchSize = *batchSize
                        } else {
                                currentBatchSize = min(*batchSize, targetWallets-walletsProcessed)
                        }
                        
                        batchNum++
                        
                        // Generate and send wallets to workers
                        for i := 0; i < currentBatchSize; i++ {
                                w := generator.GenerateWallet()
                                walletChan <- w
                                walletsProcessed++
                        }
                        
                        // Periodically save results in the background without cluttering output
                        if batchNum%50 == 0 {
                                walletsWithBalance = store.Count()
                                
                                // Only save results - don't display progress bar
                                err := store.Save()
                                if err != nil {
                                        logger.Error(fmt.Sprintf("Error saving results: %v", err))
                                }
                        }
                        
                        // If we've reached the initial target in infinite mode, reset the counter to avoid integer overflow
                        if *infiniteMode && walletsProcessed >= 1000000 {
                                logger.Info(fmt.Sprintf("Processed %d wallets, resetting counter", walletsProcessed))
                                walletsProcessed = 0
                                batchNum = 0
                        }
                }
        }
        
cleanup:
        // Cleanup and save final results
        logger.Info("Finishing up...")
        close(walletChan)
        wg.Wait()
        close(resultChan)
        <-done
        
        walletsWithBalance = store.Count()
        logger.Info(fmt.Sprintf("Finished checking %d wallets, found %d with balance", 
                walletsProcessed, walletsWithBalance))
        
        err := store.Save()
        if err != nil {
                logger.Error(fmt.Sprintf("Error saving final results: %v", err))
                os.Exit(1)
        }
        
        logger.Info(fmt.Sprintf("Results saved to %s", *outputFile))
}

// getEnabledChainsFromEnv reads chain configuration from env.txt
func getEnabledChainsFromEnv(logger *utils.Logger) []string {
    // Updated to include Bitcoin as the first chain in the list
    allChains := []string{"bitcoin", "ethereum", "binance", "polygon", "avalanche", "fantom", "optimism", "arbitrum", "base", "celo"}
    enabledChains := []string{}
    
    for _, chain := range allChains {
        if enabled, ok := utils.ReadEnvBool(chain); ok && enabled {
            // Convert to uppercase first letter for consistency
            enabledChains = append(enabledChains, chain)
            logger.Debug(fmt.Sprintf("Chain enabled: %s", chain))
        } else {
            logger.Debug(fmt.Sprintf("Chain disabled: %s", chain))
        }
    }
    
    return enabledChains
}

func min(a, b int) int {
        if a < b {
                return a
        }
        return b
}

// getChainNames extracts the names of chains from a list of ChainInfo
func getChainNames(chains []explorer.ChainInfo) []string {
        names := make([]string, len(chains))
        for i, chain := range chains {
                names[i] = chain.Name
        }
        return names
}
