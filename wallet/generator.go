package wallet

import (
        "encoding/hex"
        "fmt"
        "strings"

        "cryptowallet/utils"
        "github.com/btcsuite/btcd/btcec/v2"
        "golang.org/x/crypto/sha3"
)

// Wallet represents an Ethereum/EVM wallet
type Wallet struct {
        PrivateKey string
        Address    string
}

// WalletWithBalance extends Wallet with balance information
type WalletWithBalance struct {
        Address    string  `json:"address"`
        PrivateKey string  `json:"private_key"`
        Chain      string  `json:"chain"`
        Balance    string  `json:"balance"`
        HasBalance bool    `json:"has_balance"`
}

// Generator handles wallet generation
type Generator struct {
        logger *utils.Logger
}

// NewGenerator creates a new wallet generator
func NewGenerator(logger *utils.Logger) *Generator {
        return &Generator{
                logger: logger,
        }
}

// GenerateWallet generates a new random Ethereum/EVM wallet
func (g *Generator) GenerateWallet() Wallet {
        // Generate a secp256k1 private key (what Ethereum uses)
        privateKey, err := btcec.NewPrivateKey()
        if err != nil {
                g.logger.Error(fmt.Sprintf("Error generating private key: %v", err))
                // In production code, we would handle this more gracefully
                panic(err)
        }

        // Convert private key to hex
        privateKeyBytes := privateKey.Serialize()
        privateKeyHex := hex.EncodeToString(privateKeyBytes)

        // Get the public key and derive the address
        publicKey := privateKey.PubKey()
        publicKeyBytes := publicKey.SerializeUncompressed()[1:] // Skip the first byte (0x04)

        // Keccak256 hash of public key (Ethereum address derivation)
        h := sha3.NewLegacyKeccak256()
        h.Write(publicKeyBytes)
        hash := h.Sum(nil)

        // Take the last 20 bytes for the Ethereum address
        address := "0x" + hex.EncodeToString(hash[len(hash)-20:])

        g.logger.Debug(fmt.Sprintf("Generated wallet with address: %s", address))

        return Wallet{
                PrivateKey: privateKeyHex,
                Address:    address,
        }
}

// ValidatePrivateKey validates a private key string
func (g *Generator) ValidatePrivateKey(privateKeyHex string) bool {
        // Remove 0x prefix if present
        privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
        
        // Check if the private key is a valid hex string
        privateKeyBytes, err := hex.DecodeString(privateKeyHex)
        if err != nil {
                return false
        }
        
        // Check if the private key is of valid length
        if len(privateKeyHex) != 64 { // 32 bytes = 64 hex chars
                return false
        }
        
        // Check if it's a valid secp256k1 private key
        // btcec.PrivKeyFromBytes doesn't return an error, so we need to handle differently
        privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
        if privateKey == nil {
                return false
        }
        
        return true
}

// PrivateKeyToAddress converts a private key to an Ethereum address
func (g *Generator) PrivateKeyToAddress(privateKeyHex string) (string, error) {
        // Remove 0x prefix if present
        privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
        
        // Decode private key from hex
        privateKeyBytes, err := hex.DecodeString(privateKeyHex)
        if err != nil {
                return "", fmt.Errorf("invalid private key: %v", err)
        }
        
        // Parse as a btcec private key
        privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
        if privateKey == nil {
                return "", fmt.Errorf("invalid private key format")
        }
        
        // Get public key and derive address (same process as in GenerateWallet)
        publicKey := privateKey.PubKey()
        publicKeyBytes := publicKey.SerializeUncompressed()[1:] // Skip prefix byte
        
        h := sha3.NewLegacyKeccak256()
        h.Write(publicKeyBytes)
        hash := h.Sum(nil)
        
        // Return address with 0x prefix
        return "0x" + hex.EncodeToString(hash[len(hash)-20:]), nil
}
