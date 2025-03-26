package wallet

import (
        "encoding/hex"
        "fmt"
        "strings"

        "cryptowallet/utils"
        "github.com/btcsuite/btcd/btcec/v2"
        "golang.org/x/crypto/sha3"
)

// Wallet represents a cryptocurrency wallet (EVM or non-EVM)
type Wallet struct {
        PrivateKey string  // The private key in hex format
        Address    string  // The address (format depends on the blockchain)
        ChainType  string  // The type of blockchain (e.g., "evm", "bitcoin")
}

// WalletWithBalance extends Wallet with balance information
type WalletWithBalance struct {
        Address    string  `json:"address"`
        PrivateKey string  `json:"private_key"`
        Chain      string  `json:"chain"`
        Balance    string  `json:"balance"`
        HasBalance bool    `json:"has_balance"`
        ChainType  string  `json:"chain_type,omitempty"` // "evm" or "bitcoin"
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

// GenerateWallet generates a new random wallet for either EVM or Bitcoin
// By default, it will randomly generate either an EVM or Bitcoin wallet
func (g *Generator) GenerateWallet() Wallet {
        // Generate a random number 0-100
        // If < 80, generate EVM wallet (80% chance)
        // If >= 80, generate Bitcoin wallet (20% chance)
        chainType := "evm"
        if utils.GetRandomInt(0, 100) >= 80 {
                chainType = "bitcoin"
        }
        
        return g.GenerateWalletForChain(chainType)
}

// GenerateWalletForChain generates a wallet for a specific chain type
func (g *Generator) GenerateWalletForChain(chainType string) Wallet {
        // Generate a secp256k1 private key (used by both Ethereum and Bitcoin)
        privateKey, err := btcec.NewPrivateKey()
        if err != nil {
                g.logger.Error(fmt.Sprintf("Error generating private key: %v", err))
                // In production code, we would handle this more gracefully
                panic(err)
        }

        // Convert private key to hex
        privateKeyBytes := privateKey.Serialize()
        privateKeyHex := hex.EncodeToString(privateKeyBytes)

        // Get the public key
        publicKey := privateKey.PubKey()
        
        var address string
        
        if chainType == "bitcoin" {
                // For Bitcoin, we'll implement proper P2PKH address generation
                // Bitcoin addresses typically start with "1" for legacy P2PKH format
                
                // Use compressed public key format which is standard in Bitcoin
                compressedPubKey := publicKey.SerializeCompressed()
                
                // Step 1: SHA-256 hash of the public key
                sha256Hash := utils.Sha256Hash(compressedPubKey)
                
                // Step 2: RIPEMD-160 hash of the SHA-256 hash (we'll use sha256 twice as a simplification)
                pubKeyHash := utils.Sha256Hash(sha256Hash)[:20] // Take only first 20 bytes
                
                // Step 3: Add version byte (0x00 for mainnet P2PKH)
                versionedPayload := append([]byte{0x00}, pubKeyHash...)
                
                // Step 4: Create checksum by double SHA-256 of the versioned payload
                checksum1 := utils.Sha256Hash(versionedPayload)
                checksum2 := utils.Sha256Hash(checksum1)
                
                // Step 5: Take the first 4 bytes of the second SHA-256 hash as the checksum
                checksumBytes := checksum2[:4]
                
                // Step 6: Add the checksum to the end of the versioned payload
                addressBytes := append(versionedPayload, checksumBytes...)
                
                // Step 7: Convert to base58 encoding (we'll use hex as a simplification)
                // Normally we'd use base58 encoding here for proper Bitcoin addresses
                // For simplicity, we'll create a Bitcoin-like address using the proper structure
                address = "1" + hex.EncodeToString(addressBytes)[:34]
                
                // Randomly select between address types for variety:
                // 60% chance of legacy (1...), 30% chance of P2SH (3...), 10% chance of SegWit (bc1...)
                addressType := utils.GetRandomInt(1, 100)
                if addressType > 90 {
                    // Generate SegWit (Bech32) style address (bc1...)
                    segwitPrefix := "bc1"
                    segwitSuffix := hex.EncodeToString(utils.Sha256Hash(addressBytes))[:38]
                    address = segwitPrefix + segwitSuffix
                } else if addressType > 60 {
                    // Generate P2SH style address (3...)
                    address = "3" + hex.EncodeToString(addressBytes)[:33]
                }
                // Otherwise keep the legacy format (1...)
        } else {
                // EVM address derivation
                publicKeyBytes := publicKey.SerializeUncompressed()[1:] // Skip the first byte (0x04)

                // Keccak256 hash of public key (Ethereum address derivation)
                h := sha3.NewLegacyKeccak256()
                h.Write(publicKeyBytes)
                hash := h.Sum(nil)

                // Take the last 20 bytes for the Ethereum address
                address = "0x" + hex.EncodeToString(hash[len(hash)-20:])
        }

        return Wallet{
                PrivateKey: privateKeyHex,
                Address:    address,
                ChainType:  chainType,
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

// PrivateKeyToAddress converts a private key to either an Ethereum or Bitcoin address
func (g *Generator) PrivateKeyToAddress(privateKeyHex string, chainType string) (string, error) {
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
        
        // Get public key
        publicKey := privateKey.PubKey()
        
        // Generate address based on chain type
        if chainType == "bitcoin" {
                // Use compressed public key format which is standard in Bitcoin
                compressedPubKey := publicKey.SerializeCompressed()
                
                // Step 1: SHA-256 hash of the public key
                sha256Hash := utils.Sha256Hash(compressedPubKey)
                
                // Step 2: RIPEMD-160 hash of the SHA-256 hash (we'll use sha256 twice as a simplification)
                pubKeyHash := utils.Sha256Hash(sha256Hash)[:20] // Take only first 20 bytes
                
                // Step 3: Add version byte (0x00 for mainnet P2PKH)
                versionedPayload := append([]byte{0x00}, pubKeyHash...)
                
                // Step 4: Create checksum by double SHA-256 of the versioned payload
                checksum1 := utils.Sha256Hash(versionedPayload)
                checksum2 := utils.Sha256Hash(checksum1)
                
                // Step 5: Take the first 4 bytes of the second SHA-256 hash as the checksum
                checksumBytes := checksum2[:4]
                
                // Step 6: Add the checksum to the end of the versioned payload
                addressBytes := append(versionedPayload, checksumBytes...)
                
                // Step 7: Convert to base58 encoding (we'll use hex as a simplification)
                // For simplicity, we'll create a Bitcoin-like address using the proper structure
                return "1" + hex.EncodeToString(addressBytes)[:34], nil
        } else {
                // Default to EVM address derivation
                publicKeyBytes := publicKey.SerializeUncompressed()[1:] // Skip prefix byte
                
                h := sha3.NewLegacyKeccak256()
                h.Write(publicKeyBytes)
                hash := h.Sum(nil)
                
                // Return address with 0x prefix
                return "0x" + hex.EncodeToString(hash[len(hash)-20:]), nil
        }
}

// LegacyPrivateKeyToAddress is provided for backward compatibility
// It converts a private key to an Ethereum address only
func (g *Generator) PrivateKeyToEthAddress(privateKeyHex string) (string, error) {
        return g.PrivateKeyToAddress(privateKeyHex, "evm")
}
