# TigerWallet - Enterprise-grade Multichain Web3 Wallet

## 🚀 Complete Native Implementation — No Dependencies

TigerWallet is a **100% built-from-scratch** decentralized Web3 wallet ecosystem that doesn't depend on any third-party wallet services.

### 🌐 Massive Blockchain Support
TigerWallet supports **200+ EVM blockchain networks** and **500-1000+ Non-EVM blockchain networks** with full [admin management](./ADMIN_ARCHITECTURE.md) (add, edit, update, remove chains dynamically).

#### EVM Chains (200+ supported)
- Ethereum, Sepolia, BNB Chain, Polygon, Mumbai, Arbitrum, Optimism, Base, Avalanche, Fantom, Cronos, Celo, Gnosis, Moonbeam, Kava, Linea, zkEVM, Scroll, Mantle, opBNB, HECO, OKTC, Astar, Shiden, Fuse, Telos, Theta, RSK, Smart Chain, and many more...

#### Non-EVM Chains (500-1000+ supported)
- **Solana**: Mainnet, Devnet, Testnet
- **Aptos**: Mainnet, Devnet, Testnet
- **Sui**: Mainnet, Devnet, Testnet
- **TON**: Mainnet
- **TRON**: Mainnet, Nile
- **Cosmos**: Cosmos Hub, Osmosis, Injective, Celestia, Dymension
- **NEAR**: Mainnet, Testnet
- **Algorand**: Mainnet, Testnet
- **Bitcoin**: Mainnet, Testnet
- **Toncoin**: Mainnet
- **MultiversX**: Mainnet, Devnet
- **Hedera**: Mainnet, Testnet
- **Celestia**: Mainnet
- **Sei**: Mainnet, Testnet
- **Dymension**: Mainnet
- **Cardano**: Mainnet, Preprod
- **Starknet**: Mainnet, Goerli
- **Injective**: Mainnet, Testnet
- **Substrate/Polkadot**: Kusama, Westend
- **Algorand**: Mainnet
- **Near**: Mainnet
- **And 500+ more chains...**

### Native Implementations Built From Scratch:

| Component | Status | Description |
|-----------|--------|-------------|
| **Solana SDK** | ✅ Complete | Full RPC, SPL tokens, wallet adapters |
| **Aptos SDK** | ✅ Complete | Move language, BCS serialization |
| **TON SDK** | ✅ Complete | FunC contracts, Cell serialization |
| **Sui SDK** | ✅ Complete | Object model, Move execution |
| **TRON SDK** | ✅ Complete | TRC20, smart contracts |
| **Pi Network SDK** | ✅ Complete | Payment integration |
| **EVM Wallet Adapter** | ✅ Complete | MetaMask, WalletConnect v2, Coinbase |
| **Solana Wallet** | ✅ Complete | Phantom, Solflare, Backpack |
| **Aptos Wallet** | ✅ Complete | Martian, Sui Wallet |
| **TON Wallet** | ✅ Complete | Tonkeeper adapter |
| **Core AMM** | ✅ Complete | Concentrated liquidity, constant product |
| **MEV Protection** | ✅ Complete | Bundle builder, sandwich detector |
| **Order Book CLOB** | ✅ Complete | Limit orders, market orders |
| **DEX Aggregator** | ✅ Complete | Multi-hop routing, split routes |

## Technology Stack

For more details, see [Tech Stack Details](./TECH_STACK.md).

| Layer | Technology |
|-------|------------|
| High Performance Routing | C++ |
| Wallet Core | Rust |
| Cryptography | Rust |
| Cross Chain Engine | Rust + Go |
| Backend APIs | Go |
| Enterprise Modules | Java |
| Smart Contracts | Solidity |
| Analytics / AI | Python |
| Internal Automation | Ruby |
| Frontend | TypeScript |
| Website | Next.js |
| UI Components | React |

## Key Statistics

| Metric | Count |
|--------|-------|
| Programming Languages | 8 |
| Databases Supported | 30+ |
| Modules | 100+ |
| Microservices | 250+ |
| Blockchains Supported | 200+ EVM, 500-1000+ Non-EVM |

## Native SDKs Built From Scratch

All SDKs are built from scratch with no third-party dependencies.

## Project Structure

```text
TigerWallet/
├── blockchain_layer/           # Native blockchain SDKs
│   ├── solana_sdk/            # Solana RPC, SPL, AMM
│   ├── aptos_sdk/            # Aptos Move, BCS
│   ├── ton_sdk/              # TON Cell, FunC
│   ├── sui_sdk/              # Sui objects, Move
│   ├── tron_sdk/             # TRON TRC20
│   ├── pi_network_sdk/       # Pi payments
│   └── bitcoin_sdk/           # Bitcoin UTXO
├── core/                      # Core DEX engine
│   ├── amm/                  # Concentrated liquidity AMM
│   ├── orderbook/            # CLOB order book
│   ├── routing/              # DEX aggregator router
│   └── mev/                 # MEV protection
├── libs/                      # Libraries
│   ├── web3_wallet/          # EVM wallet adapters
│   └── routing/              # Routing engine
├── dex_aggregator/            # DEX aggregation
├── smart_contracts/           # EVM contracts
└── frontend/                  # UI applications
```

### Consolidated canonical layout (2026-08-23)

Duplicate top-level directories were merged into one canonical home each
(full functionality and fetchers preserved; see `docs/GAPS.md` item 9):

| Domain | Canonical root | Absorbed duplicates |
|--------|----------------|---------------------|
| UserWallet apps (android, ios, desktop, extension, web) | `user_wallet/` | `user_app/react` → `user_wallet/react_app/` |
| MasterWallet apps | `master_wallet/` | — (already canonical) |
| Admin apps | `admin/` | — (already canonical) |
| SuperAdmin apps | `super_admin/` | — (already canonical) |
| Browser extension | `browser_extensions/chrome/` | `browser_extension/` → `legacy/` |
| Mobile apps | `mobile_apps/` | `mobile/` → `android_legacy/`, `ios_legacy/`, `flutter_legacy/`, `tigerswap_wallet/` |
| Desktop | `desktop_wallet/` (C++), `desktop_app/` (Tauri), `user_wallet/desktop` (Electron) | `desktop/` (README-only) |
| SDKs | `sdks/` (go, javascript, python, cpp) | `sdk/`, `developer_sdk/` |
| Hardware wallet | `hardware_wallet/` (go, rust, cpp) | `hardware_backend/`, `hardware_wallet_deep/` |
| Blockchain explorer | `blockchain_explorer_system/` | `blockchain_explorer/` |
| NFT | `nft_ecosystem/` (go, rust, cpp) | `nft_marketplace/` |
| Notifications | `notifications/` | `notification/` → `go/cmd/gateway/` |
| Staking | `staking_hub/` (go, rust) | `staking/` → `go/legacy/` |
| White label | `white_label/` (+ `white_label_admin/`) | `white_label_portal/`, `white_label_system/`, `white_label_marketplace/`, `white_label_templates/`, `white_label_analytics_ai/`, `white_label_sdk/` → `sdk/cpp/`, `white_level_sdk/` → `sdk/rust/` |

Language placement rule: **C++** for ultra-low-latency speed paths,
**Rust** where safety/security is critical (also ultra-low-latency),
**Go** for worldwide distributed high-load services.

## Quick Start

```bash
# Install dependencies
npm install

# Build all packages
npm run build

# Run development
npm run dev
```

## Additional Documentation

For more details, see:
- [Architecture Overview](./ADMIN_ARCHITECTURE.md)
- [Tech Stack Details](./TECH_STACK.md)
- [Installation Guide](./INSTALLATION.md)
- [Build & Deployment](./BUILD_DEPLOY.md)

## License

MIT License - Multichain Cryptocurrency Decentralized Exchanges
