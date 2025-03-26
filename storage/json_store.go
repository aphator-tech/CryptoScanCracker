package storage

import (
        "encoding/json"
        "fmt"
        "os"
        "sync"
        "time"

        "cryptowallet/wallet"
)

// WalletsCollection represents the JSON structure for storing wallets
type WalletsCollection struct {
        Wallets     []wallet.WalletWithBalance `json:"wallets"`
        TotalCount  int                        `json:"total_count"`
        GeneratedAt string                     `json:"generated_at"`
        UpdatedAt   string                     `json:"updated_at"`
}

// JSONStore handles storing wallet data in JSON format
type JSONStore struct {
        filename  string
        wallets   []wallet.WalletWithBalance
        mu        sync.Mutex
        createdAt time.Time
}

// NewJSONStore creates a new JSON store
func NewJSONStore(filename string) *JSONStore {
        return &JSONStore{
                filename:  filename,
                wallets:   []wallet.WalletWithBalance{},
                createdAt: time.Now(),
        }
}

// AddWallet adds a wallet with balance to the store
func (s *JSONStore) AddWallet(wallet wallet.WalletWithBalance) {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        s.wallets = append(s.wallets, wallet)
}

// AddWallets adds multiple wallets to the store
func (s *JSONStore) AddWallets(wallets []wallet.WalletWithBalance) {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        s.wallets = append(s.wallets, wallets...)
}

// GetWallets returns all wallets in the store
func (s *JSONStore) GetWallets() []wallet.WalletWithBalance {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        // Return a copy to prevent race conditions
        walletsCopy := make([]wallet.WalletWithBalance, len(s.wallets))
        copy(walletsCopy, s.wallets)
        
        return walletsCopy
}

// Count returns the number of wallets in the store
func (s *JSONStore) Count() int {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        return len(s.wallets)
}

// Clear removes all wallets from the store
func (s *JSONStore) Clear() {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        s.wallets = []wallet.WalletWithBalance{}
}

// Save writes the wallets to the JSON file
func (s *JSONStore) Save() error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        // Create the collection object
        collection := WalletsCollection{
                Wallets:     s.wallets,
                TotalCount:  len(s.wallets),
                GeneratedAt: s.createdAt.Format(time.RFC3339),
                UpdatedAt:   time.Now().Format(time.RFC3339),
        }
        
        // Marshal the collection to JSON
        jsonData, err := json.MarshalIndent(collection, "", "  ")
        if err != nil {
                return fmt.Errorf("error marshaling JSON: %v", err)
        }
        
        // Write to file
        err = os.WriteFile(s.filename, jsonData, 0644)
        if err != nil {
                return fmt.Errorf("error writing to file: %v", err)
        }
        
        return nil
}

// Load reads wallets from the JSON file
func (s *JSONStore) Load() error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        // Check if the file exists
        _, err := os.Stat(s.filename)
        if os.IsNotExist(err) {
                // File doesn't exist, start with empty collection
                return nil
        }
        
        // Read the file
        jsonData, err := os.ReadFile(s.filename)
        if err != nil {
                return fmt.Errorf("error reading file: %v", err)
        }
        
        // Unmarshal the JSON
        var collection WalletsCollection
        err = json.Unmarshal(jsonData, &collection)
        if err != nil {
                return fmt.Errorf("error unmarshaling JSON: %v", err)
        }
        
        // Update the store
        s.wallets = collection.Wallets
        
        return nil
}
