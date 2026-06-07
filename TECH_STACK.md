# TigerWallet Complete Technology Stack

## Comprehensive Analysis & Recommendations

Based on deep analysis of Trust Wallet, Bitget Wallet, Binance Web3 Wallet, OKX Wallet, KuCoin Wallet, Coinbase Wallet, MetaMask, Phantom, Rabby Wallet

---

## Executive Summary

| Component | Language | Database | Why |
|-----------|----------|----------|-----|
| Wallet Core | Rust + C++ | N/A | Security, performance |
| Mobile Apps | Flutter/Swift/Kotlin | SQLite/Realm | Cross-platform |
| Browser Extension | TypeScript/React | localStorage | Web standards |
| Desktop | Flutter/Rust | SQLite | Native performance |
| Backend APIs | Go | PostgreSQL + Redis | High concurrency |
| Analytics | Python | TimescaleDB + ClickHouse | Time-series data |
| AI Layer | Python | Qdrant + Redis | Vector search |
| Blockchain Indexers | Rust + Go | PostgreSQL | Fast indexing |
| Real-time | Go | NATS + Redis | Streaming |
| Security | Rust + Python | PostgreSQL | Risk analysis |

---

## Deep Analysis by Wallet

### Trust Wallet
- **Core**: C++ with Swift/Kotlin bindings
- **Storage**: iOS Keychain, Android Keystore
- **Keys**: On-device AES encryption

### Bitget Wallet
- **Security**: TEE (AWS Nitro Enclaves)
- **Key Protection**: Double Encryption Storage Mechanism
- **AI**: MCP server for AI agents

### Binance Web3 Wallet
- **Key System**: MPC (3-shard TSS)
- **Storage**: Device + Cloud (iCloud/Google Drive)
- **Integration**: JavaScript connectors

### OKX Wallet
- **SDKs**: TypeScript, Go, Python, PHP, Java
- **Analytics**: QuestDB (time-series)
- **Chains**: 130+ supported

### Coinbase Wallet
- **Smart Wallet**: ERC-4337 compatible
- **SDKs**: npm, CocoaPods, Maven
- **Keys**: User-held (self-custody)

### MetaMask
- **Language**: TypeScript
- **RPC**: Infura
- **Recovery**: Shamir's Secret Sharing

### Phantom
- **Focus**: Solana (also ETH, Polygon)
- **SDKs**: React SDK, JavaScript
- **Keys**: Managed by Phantom

### Rabby Wallet
- **Architecture**: JS/TypeScript mono-repo
- **Provider**: Ethereum injection
- **Mobile**: @rabby-wallet packages

---

## TigerWallet Recommended Stack

### 1. Wallet Core (Rust + C++)

**Purpose**: Security-critical cryptographic operations

**Languages**:
- Rust (primary)
- C++ (performance-critical)
- Swift (iOS bindings)
- Kotlin (Android bindings)

**Libraries**:
```toml
[dependencies]
# Cryptography
k256 = "0.13"           # secp256k1 (Ethereum)
ed25519-dalek = "2.0"     # Ed25519 (Solana, Aptos)
bip32 = "0.4"           # BIP-32 HD derivation
bip39 = "0.2"           # BIP-39 mnemonic
sha2 = "0.10"          # SHA-256/512
aes-gcm = "0.10"         # AES-256-GCM encryption
chacha20poly1305 = "0.10" # XChaCha20-Poly1305

# Bitcoin
bitcoin = "0.31"          # Bitcoin primitives
bitcoincore = "0.24"      # Rust Bitcoin

# Solana
solana-program = "1.18"    # Solana programs
ed25519-dalek = "2.0"     # Ed25519 signatures

# Move (Aptos/Sui)
move-core-types = "0.0.4"   # Aptos Move

# Serialization
serde = "1.0"             # JSON serialization
serde_json = "1.0"

# Async
tokio = "1.35"            # Async runtime

# Database
rusqlite = "0.31"         # SQLite

# Zeroize for secure memory
zeroize = "1.7"           # Memory safety
```

**Database**: None (local encrypted storage only)
- iOS: Keychain with Secure Enclave
- Android: Keystore with hardware-backed keys

**Files**:
```
wallet_core/
├── src/
│   ├── lib.rs              # Main entry
│   ├── mnemonic.rs         # BIP-39
│   ├── key_derivation.rs  # BIP-32/44
│   ├── address.rs         # Address generation
│   ├── signing.rs       # Transaction signing
│   ├── encryption.rs    # AES-256-GCM
│   ├── evm.rs          # EVM chains
│   ├── bitcoin.rs      # Bitcoin
│   ├── solana.rs       # Solana
│   ├── cosmos.rs      # Cosmos
│   ├── tron.rs        # TRON
│   ├── ton.rs         # TON
│   └── lib.rs
├── Cargo.toml
└── tests/
```

---

### 2. Mobile Apps (Flutter + Dart)

**Purpose**: Cross-platform mobile wallet

**Languages**:
- Dart (primary UI)
- Rust (via FFI for crypto)
- Swift (iOS native)
- Kotlin (Android native)

**Dependencies** (pubspec.yaml):
```yaml
dependencies:
  flutter:
    sdk: flutter
  
  # State Management
  flutter_bloc: ^8.1.3
  riverpod: ^2.4.9
  
  # Local Storage
  hive: ^2.2.3
  hive_flutter: ^1.1.0
  flutter_secure_storage: ^9.0.0
  sqflite: ^2.3.0
  
  # Networking
  dio: ^5.3.3
  web_socket_channel: ^2.4.0
  connectivity_plus: ^5.0.2
  
  # Blockchain
  web3dart: ^2.0.0
  solana_dart: ^1.0.0
  tron: ^1.0.0
  
  # Crypto
  crypto: ^3.0.3
  bip39: ^1.0.6
  pointycastle: ^3.7.3
  ed25519_hd_key: ^2.0.0
  flutter_rust_bridge: ^2.0.0
  
  # Utils
  intl: ^0.18.1
  uuid: ^4.2.1
  qr_flutter: ^4.1.0
  share_plus: ^7.2.1
  url_launcher: ^6.2.2
  
  # Auth
  local_auth: ^2.1.8
  biometrics: ^1.0.0
  
  # UI
  google_fonts: ^5.0.0
  flutter_svg: ^2.0.9
  shimmer: ^3.0.0
  cached_network_image: ^3.3.1
  
  # Wallet Connect
  walletconnect_dart: ^1.0.0
```

**Database**:
- SQLite (transactions, balances)
- Realm (optional, for complex objects)
- Hive (fast key-value)

**Files**:
```
mobile_apps/
├── android_app/
│   ├── lib/
│   │   ├── main.dart
│   │   ├── app.dart
│   │   ├── core/
│   │   │   ├── theme/
│   │   │   ├── constants/
│   │   │   └── utils/
│   │   ├── features/
│   │   │   ├── wallet/
│   │   │   ├── swap/
│   │   │   ├── stake/
│   │   │   ├── nft/
│   │   │   └── settings/
│   │   ├── services/
│   │   │   ├── wallet_service.dart
│   │   │   ├── rpc_service.dart
│   │   │   └── swap_service.dart
│   │   └── widgets/
│   └── pubspec.yaml
├── ios_app/
├── tablet_app/
└── harmonyos_app/
```

---

### 3. Browser Extensions (TypeScript + React)

**Purpose**: Chrome, Edge, Firefox, Brave extensions

**Languages**:
- TypeScript (primary)
- React (UI components)
- Vite (build tool)

**Dependencies** (package.json):
```json
{
  "dependencies": {
    "ethers": "^6.10.0",
    "@solana/web3.js": "^1.87.0",
    "@solana/wallet-adapter-react": "^0.15.32",
    "@web3-react/core": "^6.1.9",
    "@web3-react/injected-connector": "^6.0.7",
    "@web3-react/walletconnect-connector": "^6.2.10",
    "bn.js": "^5.2.1",
    "bip39": "^3.0.4",
    "lucide-react": "^0.303.0",
    "zustand": "^4.4.7",
    "framer-motion": "^10.17.4"
  },
  "devDependencies": {
    "@types/chrome": "^0.0.251",
    "@typescript-eslint/eslint-plugin": "^6.15.0",
    "@vitejs/plugin-react": "^4.2.1",
    "autoprefixer": "^10.4.16",
    "eslint": "^8.56.0",
    "postcss": "^8.4.32",
    "tailwindcss": "^3.4.0",
    "typescript": "^5.3.3",
    "vite": "^5.0.10",
    "webextension-polyfill": "^0.10.0"
  }
}
```

**Database**: Extension local storage only

**Files**:
```
browser_extensions/
├── chrome_extension/
│   ├── src/
│   │   ├── background.ts    # Service worker
│   │   ├── content.ts     # Content script
│   │   ├── popup.tsx    # Extension popup
│   │   ├── options.tsx # Settings page
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── stores/
│   │   └── utils/
│   ├── public/
│   │   ├── manifest.json
│   │   └── icons/
│   ├── package.json
│   └── vite.config.ts
├── edge_extension/
├── firefox_extension/
└── brave_extension/
```

---

### 4. Desktop Wallet (Flutter + Rust)

**Purpose**: Windows, macOS, Linux desktop

**Languages**:
- Flutter (UI - single codebase)
- Rust (via FFI for crypto)

**Note**: Recommend Tauri over Electron
- Tauri: 10MB (system WebView)
- Electron: 150MB (bundled Chromium)
- Better attack surface for security product

**Files**:
```
desktop_wallet/
├── windows_wallet/
├── macos_wallet/
└── linux_wallet/
```

---

### 5. Backend APIs (Go)

**Purpose**: High-concurrency microservices

**Languages**:
- Go (primary)
- TypeScript (optional for Node.js services)

**Dependencies** (go.mod):
```go
module github.com/tigerwallet/backend

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/go-redis/redis/v8 v8.11.5
    github.com/gorilla/websocket v1.5.1
    github.com/jackc/pgx/v5 v5.5.1
    github.com/joho/godotenv v1.5.1
    github.com/rs/zerolog v1.31.0
    github.com/spf13/viper v1.18.2
    github.com/nats-io/nats.go v1.30.0
    github.com/ethereum/go-ethereum v1.12.0
    github.com/gagliardetto/solana-go v1.26.0
)
```

**Database**:
- PostgreSQL (user data, transactions)
- Redis (caching, sessions)
- NATS (real-time events)

**Microservices**:
```
backend_services/
├── api_gateway/          # Main API entry
├── auth_service/        # Authentication
├── wallet_service/     # Wallet operations
├── portfolio_service/   # Portfolio tracking
├── swap_service/        # DEX aggregation
├── bridge_service/       # Cross-chain bridges
├── staking_service/     # Staking operations
├── nft_service/        # NFT operations
├── notification_service/# Push notifications
├── analytics_service/    # Analytics
└── admin_service/     # Admin panel
```

---

### 6. Analytics Platform (Python)

**Purpose**: AI/ML, data science, analytics

**Languages**:
- Python (primary)
- PyTorch/TensorFlow (ML)

**Dependencies** (requirements.txt):
```python
# Core
pandas>=2.0.0
numpy>=1.24.0
scipy>=1.11.0

# ML/AI
torch>=2.0.0
tensorflow>=2.13.0
transformers>=4.35.0
langchain>=0.1.0

# Database
psycopg2-binary>=2.9.9
redis>=5.0.0
sqlalchemy>=2.0.0

# Time-series
timescaleb>=2.0.0
influxdb-client>=1.40.0

# API
fastapi>=0.104.0
uvicorn>=0.24.0
aiohttp>=3.9.0

# Visualization
matplotlib>=3.8.0
seaborn>=0.13.0
plotly>=5.18.0
```

**Database**:
- TimescaleDB (time-series portfolio data)
- ClickHouse (analytics, logs)
- PostgreSQL (structured data)

**Files**:
```
ai_layer/
├── portfolio_advisor/
│   ├── portfolio_optimizer.py
│   └── risk_calculator.py
├── risk_detection/
│   ├── fraud_detector.py
│   └── anomaly_detector.py
├── scam_detection/
│   ├── scam_token_detector.py
│   └── phishing_detector.py
├── market_analysis/
│   ├── price_predictor.py
│   └── sentiment_analysis.py
├── transaction_explainer/
│   └── calldata_explainer.py
└── support_assistant/
    └── ai_chatbot.py
```

---

### 7. Blockchain Indexers (Rust + Go)

**Purpose**: Fast blockchain data indexing

**Languages**:
- Rust (high-performance indexing)
- Go (API services)

**Database**:
- PostgreSQL (indexed data)
- ClickHouse (analytics)

**Files**:
```
data_platform/
├── blockchain_indexers/
│   ├── evm_indexer/
│   │   ├── src/
│   │   │   ├── main.rs
│   │   │   ├── blocks.rs
│   │   │   ├── transactions.rs
│   │   │   └── tokens.rs
│   │   └── Cargo.toml
│   ├── bitcoin_indexer/
│   ├── solana_indexer/
│   └── ton_indexer/
├── market_data_engine/
│   ├── src/
│   │   ├── price_feed.rs
│   │   └── market_data.rs
│   └── Cargo.toml
└── realtime_streaming/
    ├── kafka/
    └── nats/
```

---

### 8. Security Services (Rust + Python)

**Purpose**: Transaction simulation, fraud detection

**Languages**:
- Rust (simulation)
- Python (ML models)

**Files**:
```
security_center/
├── transaction_simulator/
│   ├── tenderly_client.py
│   └── blowfish_client.py
├── risk_scoring/
│   ├── address_scorer.py
│   └── fraud_detector.py
├── anti_phishing/
│   ├── domain_scanner.py
│   └── url_validator.py
└── wallet_guardian/
    └── security_rules.rs
```

---

### 9. Infrastructure (DevOps)

**Languages/Tools**:
- Docker (containers)
- Kubernetes (orchestration)
- Terraform (infrastructure)
- ArgoCD (GitOps)
- Vault (secrets)
- Prometheus + Grafana (monitoring)
- Loki (logging)
- OpenTelemetry (tracing)

**Files**:
```
devops/
├── kubernetes/
│   ├── base/
│   ├── services/
│   └── ingress/
├── helm/
│   ├── api-gateway/
│   ├── wallet-service/
│   └── analytics/
├── terraform/
│   ├── aws/
│   ├── gcp/
│   └── azure/
├── docker/
│   ├── api-gateway/
│   ├── wallet-service/
│   └── analytics/
├── argocd/
│   └── applications.yaml
└── monitoring/
    ├── prometheus.yaml
    ├── grafana.yaml
    └── loki.yaml
```

---

## Database Architecture

### Primary Databases

| Database | Purpose | Example Usage |
|----------|---------|--------------|
| PostgreSQL | User data, transactions | User accounts, tx history |
| Redis | Cache, sessions, rate limiting | API cache, session store |
| TimescaleDB | Time-series portfolio data | Historical balances |
| ClickHouse | Analytics, logs, events | Trading analytics, logs |
| MongoDB | Semi-structured data | NFT metadata |
| Qdrant | Vector search | AI embeddings |
| NATS | Real-time messaging | Internal events |
| Kafka | Event streaming | Blockchain events |
| S3/MinIO | File storage | Backups, images |

### Recommended Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Frontend Layer                    │
│  (Flutter Mobile, React Extension, Desktop)        │
└─────────────────────┬───────────────────────────────┘
                    │ HTTPS/WSS
┌────────────────────▼───────────────────────────────┐
│                  API Gateway                       │
│                    (Go)                          │
└─────────────────────┬───────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
   ┌────▼────┐ ┌────▼────┐ ┌────▼────┐
   │  Gin    │ │  Gin    │ │  Gin    │
   │Service1 │ │Service2 │ │Service3 │
   └────┬────┘ └────┬────┘ └────┬────┘
        │             │             │
   ┌────▼────────────▼────────────▼────┐
   │      Data Layer                   │
   │  PostgreSQL + Redis + NATS     │
   └────────────────────────────────┘
                    │
         ┌──────────┼──────────┐
         │          │          │
    ┌────▼────┐ ┌──▼──┐ ┌────▼────┐
    │Timescale│ │Click│ │ Qdrant │
    │DB      │ │House│ │        │
    └────────┘ └─────┘ └────────┘
```

---

## Chain Support Recommendations

### Tier 1 (Critical)
| Chain | Chain ID | Language | RPC Library |
|-------|---------|---------|-----------|
| Ethereum | 1 | Rust | alloy-rs |
| Bitcoin | 0 | Rust | bitcoin-rs |
| Solana | 101 | Rust | solana-rs |
| TON | 607 | Rust | ton-rs |

### Tier 2 (High Priority)
| Chain | Chain ID | Language | RPC Library |
|-------|---------|---------|-----------|
| BNB Chain | 56 | Rust | alloy-rs |
| Polygon | 137 | Rust | alloy-rs |
| Arbitrum | 42161 | Rust | alloy-rs |
| Optimism | 10 | Rust | alloy-rs |
| Base | 8453 | Rust | alloy-rs |

### Tier 3 (Medium Priority)
| Chain | Chain ID | Language | RPC Library |
|-------|---------|---------|-----------|
| Avalanche | 43114 | Rust | alloy-rs |
| Aptos | 1 | Rust | aptoss-rs |
| Sui | 1 | Rust | sui-rs |
| Cosmos | 118 | Go | cosmos-sdk |
| TRON | 7281265 | Java | trongrid |

---

## Build Order Recommendation

1. **Wallet Core** → Security-critical, must be solid first
2. **Mobile App** → Primary user interface
3. **EVM + BTC + SOL** → Most used chains
4. **Go Backend** → API infrastructure
5. **Chrome Extension** → Browser DApp support
6. **DEX Aggregator** → Swap functionality
7. **TON** → Telegram integration (900M users)
8. **AI Layer** → Advanced features

---

## Key Libraries by Function

### Rust (Wallet Core)
- `alloy-rs` - Ethereum (replaces deprecated ethers-rs)
- `bitcoin` - Bitcoin primitives
- `solana-program` - Solana programs
- `bip32` / `bip39` - Key derivation
- `ed25519-dalek` - Ed25519 signatures

### TypeScript (Extensions)
- `ethers` v6 - Ethereum
- `@solana/web3.js` - Solana
- `@web3-react/*` - WalletConnect
- `zustand` - State management

### Go (Backend)
- `gin-gonic/gin` - HTTP framework
- `go-redis/redis` - Redis client
- `jackc/pgx` - PostgreSQL
- `nats-io/nats` - Messaging

### Python (AI)
- `transformers` - LLM models
- `torch` - Deep learning
- `langchain` - AI agents
- `qdrant` - Vector search

---

## Summary

TigerWallet should use:

| Layer | Language | Database | Key Libraries |
|-------|----------|----------|---------------|
| Wallet Core | Rust/C++ | N/A | alloy, bitcoin, bip32 |
| Mobile | Flutter | SQLite | web3dart, solana |
| Extension | TypeScript | localStorage | ethers, web3-react |
| Desktop | Flutter/Rust | SQLite | Same as mobile |
| Backend | Go | PostgreSQL + Redis | gin, pgx, nats |
| Analytics | Python | TimescaleDB | torch, transformers |
| Indexers | Rust | PostgreSQL | custom |
| AI | Python | Qdrant | langchain |
| Security | Rust/Python | PostgreSQL | tenderly, blowfish |

This stack matches enterprise-grade wallets like Trust Wallet, Bitget, and OKX while incorporating modern improvements like MPC, Account Abstraction, and AI-powered features.