# Crypto Wallet Explorer

A Go-powered application that generates random cryptocurrency wallet addresses and scans multiple blockchains for balances, designed for cryptocurrency enthusiasts and researchers.

## Features

- 🔑 **Generates** random cryptocurrency wallet addresses
- 💰 **Scans balances** across multiple blockchain networks simultaneously 
- 🚀 **High performance** with concurrent processing and batch operations
- 🔒 **No API keys required** - communicates directly with blockchain explorers
- 🛡️ **Smart rate limit protection** with automatic cooldown for affected chains
- 💻 **Real-time feedback** showing each wallet check with timestamp and status
- 📊 **Saves results** to JSON for later analysis
- 🔄 **Multiple chain support** for both Bitcoin-type and Ethereum-compatible networks

## Supported Blockchains

### Bitcoin-type Networks
- Bitcoin (BTC) - Multiple address formats supported

### Ethereum-compatible Networks
- Ethereum (ETH)
- Binance Smart Chain (BNB)
- Polygon (MATIC)
- Avalanche (AVAX)
- Fantom (FTM)
- Optimism (ETH)
- Arbitrum (ETH)
- Base (ETH)
- Celo (CELO)

## Getting Started

### Requirements

- Go 1.18 or higher installed on your system

### Quick Start (From ZIP File)

1. **Extract the ZIP file** to a location of your choice
   
2. **Open a terminal/command prompt** and navigate to the extracted folder:
   ```bash
   cd path/to/extracted/folder
   ```

3. **Build the application**:
   ```bash
   go build -o wallet-explorer
   ```

4. **Run the application**:
   ```bash
   # On Windows
   wallet-explorer.exe -wallets 100 -batch 10
   
   # On macOS/Linux
   ./wallet-explorer -wallets 100 -batch 10
   ```

## Command Line Options

- `-wallets <number>`: Total wallet addresses to generate and check (default: 100)
- `-batch <number>`: Number of wallets to process in each batch (default: 10)
- `-delay <milliseconds>`: Delay between requests to avoid rate limits (default: 20)
- `-output <filename>`: Name of output JSON file (default: "wallets_with_balance.json")
- `-goroutines <number>`: Maximum goroutines to use (default: 50)
- `-log <level>`: Log level [debug, info, warn, error] (default: info)
- `-chains <list>`: Comma-separated list of chains to check (default: all available)
- `-infinite <true/false>`: Run in continuous mode (default: true)

## Usage Examples

Check a smaller set of wallets across all chains:
```bash
./wallet-explorer -wallets 50 -batch 15
```

Focus only on Bitcoin network:
```bash
./wallet-explorer -chains bitcoin -wallets 200
```

Check specific chains with optimized settings:
```bash
./wallet-explorer -chains bitcoin,ethereum,binance -batch 20 -delay 10
```

Set warning-only logs for less console output:
```bash
./wallet-explorer -log warn -wallets 1000 -batch 20
```

High-performance settings for powerful computers:
```bash
./wallet-explorer -wallets 500 -batch 50 -delay 5 -goroutines 100
```

## Output Display

The application shows real-time wallet checking with timestamp and status:

```
[09:15:23] 0x7a3b4c5d6e7f8g9h... - ❌ No balance
[09:15:24] bc1abc123def456g... - ❌ No balance
[09:15:25] 0x1f2e3d4c5b6a7... - ✅ BALANCE FOUND!
```

When a wallet with balance is found, details are also shown:
```
₿ Bitcoin: bc1abc123def456g... = 0.00123
```

All wallets with balances are saved to the output file in this format:
```json
[
  {
    "address": "0x1a2b3c4d5e6f...",
    "privateKey": "0x1a2b3c4d5e6f...",
    "chain": "ethereum",
    "balance": "0.125",
    "hasBalance": true,
    "chain_type": "evm"
  }
]
```

## Tips for Better Performance

- Lower `-delay` values increase speed but may trigger rate limits
- Larger `-batch` sizes process more wallets simultaneously
- Choose specific chains with `-chains` to focus scanning
- Use `-log warn` to reduce console output and improve performance

## Legal and Educational Use

This tool is designed for educational and research purposes only. Always ensure you comply with all applicable laws and terms of service when using blockchain explorers.

## License

This project is licensed under the MIT License.