package main

import (
        "flag"
        "fmt"
        "os"
        "os/signal"
        "runtime"
        "sync"
        "syscall"

        "cryptowallet/explorer"
        "cryptowallet/storage"
        "cryptowallet/utils"
        "cryptowallet/wallet"
)

// Command line flags
var (
        numWallets      = flag.Int("wallets", 100, "Number of wallets to generate and check")
        batchSize       = flag.Int("batch", 10, "Batch size for concurrent wallet checking")
        requestDelay    = flag.Int("delay", 200, "Delay between requests in milliseconds")
        outputFile      = flag.String("output", "wallets_with_balance.json", "Output JSON file for wallets with balance")
        maxGoroutines   = flag.Int("goroutines", 5, "Maximum number of concurrent goroutines")
        logLevel        = flag.String("log", "info", "Log level (debug, info, warn, error)")
        selectedChains  = flag.String("chains", "all", "Comma-separated list of chains to check (or 'all')")
)

func main() {
        flag.Parse()
        
        // Setup logger
        logger := utils.NewLogger(*logLevel)
        logger.Info("Starting wallet generator and balance checker")
        
        // Setup signal handling for graceful shutdown
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        
        // Initialize JSON store
        store := storage.NewJSONStore(*outputFile)
        
        // Parse chains to check
        chainList := explorer.GetChainList(*selectedChains)
        logger.Info(fmt.Sprintf("Checking balances on %d chains", len(chainList)))
        
        // Initialize wallet generator
        generator := wallet.NewGenerator(logger)
        
        // Initialize balance checker
        balanceChecker := explorer.NewBalanceChecker(
                *requestDelay,
                chainList,
                logger,
        )
        
        // Setup worker pool
        numCores := runtime.NumCPU()
        maxWorkers := *maxGoroutines
        if maxWorkers <= 0 || maxWorkers > numCores*2 {
                maxWorkers = numCores
        }
        logger.Info(fmt.Sprintf("Using %d worker goroutines", maxWorkers))
        
        // Create work channels
        walletChan := make(chan wallet.Wallet, *batchSize)
        resultChan := make(chan wallet.WalletWithBalance, *batchSize)
        done := make(chan struct{})
        
        // Start worker pool
        var wg sync.WaitGroup
        for i := 0; i < maxWorkers; i++ {
                wg.Add(1)
                go func() {
                        defer wg.Done()
                        for w := range walletChan {
                                walletWithBalances := balanceChecker.CheckWalletBalances(w)
                                for _, wb := range walletWithBalances {
                                        if wb.HasBalance {
                                                resultChan <- wb
                                        }
                                }
                        }
                }()
        }
        
        // Start result handler
        go func() {
                for result := range resultChan {
                        logger.Info(fmt.Sprintf("Found wallet with balance: %s on %s: %s", 
                                result.Address, result.Chain, result.Balance))
                        store.AddWallet(result)
                }
                close(done)
        }()
        
        // Start wallet generation and checking
        logger.Info(fmt.Sprintf("Generating and checking %d wallets", *numWallets))
        
        walletsProcessed := 0
        walletsWithBalance := 0
        
        // Process wallet generation in batches
        batchNum := 0
        for walletsProcessed < *numWallets {
                select {
                case <-sigChan:
                        logger.Info("Received interrupt signal, shutting down...")
                        goto cleanup
                default:
                        batchSize := min(*batchSize, *numWallets-walletsProcessed)
                        batchNum++
                        
                        logger.Debug(fmt.Sprintf("Processing batch %d (%d wallets)", batchNum, batchSize))
                        
                        // Generate and send wallets to workers
                        for i := 0; i < batchSize; i++ {
                                w := generator.GenerateWallet()
                                walletChan <- w
                                walletsProcessed++
                        }
                        
                        // Periodically save results
                        if batchNum%10 == 0 {
                                walletsWithBalance = store.Count()
                                logger.Info(fmt.Sprintf("Progress: %d/%d wallets checked, found %d with balance", 
                                        walletsProcessed, *numWallets, walletsWithBalance))
                                
                                err := store.Save()
                                if err != nil {
                                        logger.Error(fmt.Sprintf("Error saving results: %v", err))
                                } else {
                                        logger.Debug("Saved intermediate results")
                                }
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

func min(a, b int) int {
        if a < b {
                return a
        }
        return b
}
