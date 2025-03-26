# Crypto Wallet Balance Checker

A Go application that generates random cryptocurrency wallet private keys and checks balances across multiple blockchains (EVM and non-EVM) without requiring any API keys.

## Features

- 🔑 **Generates** random cryptocurrency wallet private keys & addresses
- 💰 **Checks balances** across multiple blockchains simultaneously (including Bitcoin)
- 🚀 **High performance** with concurrent requests and proxy support
- 🔒 **No API keys required** - scrapes balance data directly from explorers
- 🛡️ **Smart rate limit handling** with 60-second cooldown for affected chains
- 💻 **Clean, colorful output** with emoji indicators showing only wallets with balances
- 📊 **Saves results** to JSON for easy processing
- 🧮 **Works with both** Ethereum Virtual Machine (EVM) chains and Bitcoin

## Supported Blockchains

### Non-EVM Chains
- Bitcoin (BTC) - Legacy, P2SH, and Bech32 address formats

### EVM Chains
- Ethereum (ETH)
- Binance Smart Chain (BNB)
- Polygon (MATIC)
- Avalanche (AVAX)
- Fantom (FTM)
- Optimism (ETH)
- Arbitrum (ETH)
- Base (ETH)
- Celo (CELO)

## Installation

### Prerequisites

- Go 1.18 or higher

### Installing Go

#### For Windows:
1. Download the Go installer from [golang.org/dl/](https://golang.org/dl/)
2. Run the installer and follow the instructions
3. Add Go to your PATH if needed

#### For macOS:
```bash
# Using Homebrew
brew install go
```

#### For Linux:
```bash
# For Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# For CentOS/RHEL
sudo yum install golang
```

### Setting up the Project

1. Clone this repository or download and extract the ZIP file:
```bash
git clone https://github.com/yourusername/crypto-wallet-checker.git
cd crypto-wallet-checker
```

2. Build the project:
```bash
go build -o wallet-checker
```

## Usage

Run the application with default settings:
```bash
./wallet-checker
```

### Available Command Line Flags

- `-wallets <n>`: Number of wallets to generate (default: 100)
- `-batch <n>`: Batch size for concurrent checking (default: 10)
- `-delay <ms>`: Delay between requests in milliseconds (default: 20)
- `-output <file>`: Output JSON file (default: "wallets_with_balance.json")
- `-goroutines <n>`: Maximum goroutines to use (default: 50)
- `-log <level>`: Log level [debug, info, warn, error] (default: info)
- `-chains <list>`: Comma-separated chains to check (default: all)
- `-infinite <bool>`: Run in infinite mode (default: true)

### Examples

Check all chains with 50 wallets in a single batch:
```bash
./wallet-checker -wallets 50 -batch 50
```

Only check Ethereum and Binance Smart Chain:
```bash
./wallet-checker -chains ethereum,binance
```

Check only Bitcoin addresses:
```bash
./wallet-checker -chains bitcoin
```

Check Bitcoin and Ethereum with optimized settings:
```bash
./wallet-checker -chains bitcoin,ethereum -batch 40 -delay 5
```

Run with maximum debug information:
```bash
./wallet-checker -log debug -wallets 10 -batch 5 -delay 500
```

Improved performance mode (for powerful machines):
```bash
./wallet-checker -wallets 200 -batch 40 -delay 5 -goroutines 100
```

## Output

When a wallet with a balance is found, the application will output a colorful message with emoji indicators:

```
💰 Ethereum: 0x1a2b3c4d5e6f... = 0.125
💰 Bitcoin: 1A2b3C4d5E6f... = 0.012
```

All wallets with balances are automatically saved to `wallets_with_balance.json` in the following format:

```json
[
  {
    "address": "0x1a2b3c4d5e6f...",
    "privateKey": "0x1a2b3c4d5e6f...",
    "chain": "ethereum",
    "balance": "0.125",
    "hasBalance": true,
    "chain_type": "evm"
  },
  {
    "address": "1A2b3C4d5E6f...",
    "privateKey": "0x1a2b3c4d5e6f...",
    "chain": "bitcoin",
    "balance": "0.012",
    "hasBalance": true,
    "chain_type": "bitcoin"
  }
]
```

## Performance Tips

- For faster checking, use lower `-delay` values (e.g., `-delay 10`)
- Increase `-batch` size for more concurrent checks
- Use `-goroutines` to control CPU usage (higher = faster but more CPU)

## Notes

- The application handles rate limiting (429 errors) automatically
- Running with `-log debug` shows detailed information but slows down processing

## License

This project is licensed under the MIT License - see the LICENSE file for details.