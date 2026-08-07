# TigerWallet Enterprise Tech Stack

## Overview
- Languages: 8
- Databases: 30+
- Modules: 100+
- Blockchains: 100+
- Microservices: 250+

---

## Language-to-Layer Map

### Rust - Security & Cryptography Layer
**Components**: wallet_core, blockchain_engines, swap_engine, security_center, indexers, wallet_cloud

Memory-safe, zero-cost abstractions, no garbage collector. The only language you trust with private keys and cryptographic operations at scale.

### Go - Backend & Microservices Layer
**Components**: api_gateway, asset_management, staking_hub, copy_trading, payments, notifications, user_services, real_time_ws

Goroutines handle millions of concurrent connections. Fast compile, low memory, excellent for microservices, REST, gRPC, and WebSocket servers.

### Python - AI / ML / Analytics Layer
**Components**: ai_layer, risk_detection, scam_detection, analytics, portfolio_advisor, market_intelligence

PyTorch, TensorFlow, LangChain, scikit-learn. The ecosystem for ML/AI is unmatched. Also ideal for data pipelines and analytics warehousing.

### TypeScript - Web / Browser Layer
**Components**: browser_extensions, web3_browser, admin_console, nft_frontend, launchpad_ui

Web3 ecosystem standard (ethers.js, wagmi, viem). React + Vite for extensions. Next.js for admin console. Type-safe Web3 interactions.

### Flutter / Dart - Mobile & Desktop Layer
**Components**: android_app, ios_app, harmonyos_app, tablet_app, desktop_wallet

One codebase for Android, iOS, desktop. Pixel-perfect UI, native performance via Rust FFI for crypto ops. Trust Wallet uses similar architecture.

### Java / Kotlin - Enterprise & Fiat Layer
**Components**: fiat_gateway, enterprise_features, compliance_tools, banking_integrations

Banking APIs, SWIFT integrations, and enterprise compliance systems are Java-first. Mature ecosystem for PCI-DSS, KYC/AML pipelines, and RBAC systems.

---

## Frontend & Client Apps

### Mobile Apps (Android / iOS / HarmonyOS)
| Technology | Purpose |
|------------|----------|
| Flutter/Dart | Primary UI |
| Rust (FFI) | Crypto operations |
| PostgreSQL API + Hive cache | Synced data and bounded offline cache |
| Hive | Cache |
| flutter_secure_storage | Key storage |

### Desktop Wallet (Win / Mac / Linux)
| Technology | Purpose |
|------------|----------|
| Flutter | UI |
| Rust | Crypto backend |
| PostgreSQL API + Hive cache | Synced data and bounded offline cache |
| OS Keychain | Key storage |
| Tauri (alternative) | Lightweight option |

### Browser Extensions (Chrome / Firefox / Brave / Edge)
| Technology | Purpose |
|------------|----------|
| TypeScript | Primary language |
| React 18 | UI components |
| Vite | Build tool |
| ethers.js v6 | Ethereum |
| wagmi | React hooks |
| chrome.storage | Extension storage |

### Admin Console
| Technology | Purpose |
|------------|----------|
| TypeScript | Primary |
| Next.js 14 | Framework |
| React | UI |
| Tailwind CSS | Styling |
| shadcn/ui | Components |
| tRPC | Type-safe API |

---

## Wallet Core (Rust)

### Security Mandate
Wallet Core must NEVER be written in JavaScript, Python, or Go. Rust's memory safety prevents buffer overflows, use-after-free, and side-channel attacks that have caused real wallet hacks.

### Key Libraries
| Library | Purpose |
|---------|---------|
| secp256k1 | ECDSA signatures |
| ed25519 | Ed25519 (Solana, Aptos) |
| ring | Cryptographic operations |
| bip32 | HD derivation |
| bip39 | Mnemonic phrases |
| alloy-rs | Ethereum (NEW - recommended) |

### Key Features
- BIP-32/39/44 HD wallet derivation
- Multi-chain address generation (EVM, Bitcoin, Solana, TRON, Cosmos)
- EIP-4337 Account Abstraction
- MPC (Multi-Party Computation)
- Social Recovery (Shamir Secret Sharing)
- MultiSig wallet support

---

## Backend Services (Go)

### Microservices
| Service | Purpose |
|---------|----------|
| api_gateway | Main entry point |
| auth_service | Authentication |
| wallet_service | Wallet operations |
| portfolio_service | Portfolio tracking |
| swap_service | DEX aggregation |
| bridge_service | Cross-chain bridges |
| staking_service | Staking operations |
| nft_service | NFT operations |
| notification_service | Push notifications |
| analytics_service | Analytics |
| admin_service | Admin panel |

### Key Technologies
- Gin framework for HTTP
- pgx for PostgreSQL
- go-redis for Redis
- nats-io for messaging
- go-ethereum for EVM

---

## Blockchain Connectivity

### Tier 1 - Critical
| Chain | Language | Library |
|-------|----------|---------|
| Ethereum | Rust | alloy-rs |
| Bitcoin | Rust | bitcoin-rs |
| Solana | Rust | solana-rs |
| TON | Rust | ton-rs |

### Tier 2 - High Priority
| Chain | Language | Library |
|-------|----------|---------|
| BNB Chain | Rust | alloy-rs |
| Polygon | Rust | alloy-rs |
| Arbitrum | Rust | alloy-rs |
| Optimism | Rust | alloy-rs |
| Base | Rust | alloy-rs |

### Tier 3 - Medium Priority
| Chain | Language | Library |
|-------|----------|---------|
| Avalanche | Rust | alloy-rs |
| Aptos | Rust | aptoss-rs |
| Sui | Rust | sui-rs |
| Cosmos | Go | cosmos-sdk |
| TRON | Java | trongrid |

---

## Database Architecture

### Primary Databases

| Database | Purpose | Example Usage |
|----------|---------|---------------|
| PostgreSQL | User data, transactions | User accounts, tx history |
| Redis | Cache, sessions, rate limiting | API cache, session store |
| TimescaleDB | Time-series portfolio data | Historical balances |
| ClickHouse | Analytics, logs, events | Trading analytics, logs |
| MongoDB | Semi-structured data | NFT metadata |
| Qdrant | Vector search | AI embeddings |
| NATS | Real-time messaging | Internal events |
| Kafka | Event streaming | Blockchain events |
| S3/MinIO | File storage | Backups, images |

---

## AI Layer (Python)

### Components
| Component | Purpose | Libraries |
|-----------|---------|-----------|
| portfolio_advisor | LLM-powered analysis | LangChain, OpenAI |
| risk_detection | Fraud detection | PyTorch, GNN |
| scam_detection | Scam token detection | scikit-learn |
| market_analysis | Sentiment analysis | Transformers |
| transaction_explainer | AI tx explanation | LLM API |
| support_assistant | AI chatbot | LangChain, RAG |

---

## Security Center

### Components
| Component | Purpose |
|-----------|--------|
| anti_phishing | URL/domain scanning |
| smart_contract_scanner | Contract audit |
| transaction_simulator | Tenderly/Blowfish API |
| address_reputation | Chainalysis/TRM |
| biometric_security | FaceID/TouchID |
| siem_monitoring | Elasticsearch, Grafana |

---

## Infrastructure

### Container & Orchestration
- Docker
- Kubernetes (K8s)
- Helm Charts
- Istio (Service Mesh)

### CI/CD
- GitHub Actions
- ArgoCD
- Terraform

### Observability
- Prometheus + Grafana (metrics)
- Loki (logs)
- Tempo (tracing)
- OpenTelemetry

### Secrets
- HashiCorp Vault
- AWS KMS
- HSM

---

## Build Order Recommendation

### Phase 1 - Core (Months 1-6)
1. Rust wallet_core
2. Flutter mobile app
3. EVM + Bitcoin + Solana connectivity
4. Go backend API
5. PostgreSQL + Redis
6. Chrome extension

### Phase 2 - Features (Months 7-12)
1. DEX aggregator
2. Staking hub
3. NFT gallery
4. WalletConnect v2
5. TON support
6. Transaction simulator
7. Swap & bridge
8. ClickHouse analytics

### Phase 3 - Intelligence (Months 13-18)
1. Python AI layer
2. Scam detection
3. Portfolio advisor
4. Copy trading
5. Market intelligence
6. Fiat gateway
7. MPC wallet option
8. Social recovery

---

## Tech Stack Summary

| Component | Language | Database | Key Libraries |
|-----------|----------|----------|---------------|
| Wallet Core | Rust | N/A | alloy, bitcoin, bip32 |
| Mobile | Flutter | PostgreSQL API + Hive cache | web3dart, solana |
| Extension | TypeScript | localStorage | ethers, web3-react |
| Desktop | Flutter/Rust | PostgreSQL API + encrypted bounded cache | Tauri |
| Backend | Go | PostgreSQL + Redis | gin, pgx, nats |
| Analytics | Python | TimescaleDB | torch, transformers |
| AI | Python | Qdrant | langchain |
| Security | Rust/Python | PostgreSQL | tenderly, blowfish |
| Indexers | Rust | PostgreSQL | custom |

---

## Key Recommendations

1. **alloy-rs** - Replace deprecated ethers-rs
2. **MPC** - Like Binance Web3 Wallet
3. **EIP-4337** - Account Abstraction
4. **Tauri** - Over Electron for desktop
5. **Bitcoin Ordinals** - Major feature
6. **TON** - Priority due to Telegram users

---

## Comparison with Competitors

| Wallet | Core | Mobile | Backend | AI |
|--------|------|--------|---------|-----|
| Trust Wallet | C++ | Swift/Kotlin | Not disclosed | No |
| Bitget Wallet | Rust | Flutter | Go | MCP |
| Binance Web3 | MPC | Flutter | Not disclosed | No |
| OKX Wallet | TypeScript | Flutter | Go | Python |
| MetaMask | TypeScript | React Native | Not disclosed | No |
| **TigerWallet** | **Rust** | **Flutter** | **Go** | **Python** |

TigerWallet combines the best of all wallets with MPC, Account Abstraction, and TON-first strategy.
