package wallet

import (
        "crypto/ecdsa"
        "crypto/rand"
        "encoding/hex"
        "fmt"
        "math/big"
        "strings"

        "crypto/elliptic"

        "cryptowallet/utils"
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
        // Generate private key using elliptic curve cryptography
        privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
        if err != nil {
                g.logger.Error(fmt.Sprintf("Error generating private key: %v", err))
                // In production code, we would handle this more gracefully
                panic(err)
        }

        // Convert private key to hex
        privateKeyBytes := privateKey.D.Bytes()
        privateKeyHex := hex.EncodeToString(privateKeyBytes)

        // Derive address from public key
        address := g.deriveAddress(privateKey)

        g.logger.Debug(fmt.Sprintf("Generated wallet with address: %s", address))

        return Wallet{
                PrivateKey: privateKeyHex,
                Address:    address,
        }
}

// deriveAddress derives an Ethereum address from a private key
func (g *Generator) deriveAddress(privateKey *ecdsa.PrivateKey) string {
        // Get the public key
        publicKey := privateKey.Public()
        publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
        if !ok {
                g.logger.Error("Error casting public key to ECDSA")
                panic("Error casting public key to ECDSA")
        }

        // Skip the first byte which contains the key prefix (0x04)
        publicKeyBytes := elliptic.Marshal(publicKeyECDSA.Curve, publicKeyECDSA.X, publicKeyECDSA.Y)[1:]

        // Keccak256 hash of public key
        h := sha3.NewLegacyKeccak256()
        h.Write(publicKeyBytes)
        hash := h.Sum(nil)

        // Take the last 20 bytes of the hash
        address := hash[len(hash)-20:]
        
        // Return the address with 0x prefix
        return "0x" + hex.EncodeToString(address)
}

// ValidatePrivateKey validates a private key string
func (g *Generator) ValidatePrivateKey(privateKeyHex string) bool {
        // Remove 0x prefix if present
        privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
        
        // Check if the private key is a valid hex string
        _, err := hex.DecodeString(privateKeyHex)
        if err != nil {
                return false
        }
        
        // Check if the private key is of valid length
        return len(privateKeyHex) == 64 // 32 bytes = 64 hex chars
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
        
        // Create big int from bytes
        privateKeyBigInt := new(big.Int).SetBytes(privateKeyBytes)
        
        // Create private key from big int
        privateKey := new(ecdsa.PrivateKey)
        privateKey.Curve = elliptic.P256()
        privateKey.D = privateKeyBigInt
        
        // Calculate public key
        privateKey.PublicKey.X, privateKey.PublicKey.Y = privateKey.Curve.ScalarBaseMult(privateKeyBytes)
        
        // Derive address
        address := g.deriveAddress(privateKey)
        
        return address, nil
}
