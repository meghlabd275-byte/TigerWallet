# TigerWallet Complete Wallet System Specification

## Document Version: 1.0
## Last Updated: 2026-06-08

---

# Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Wallet Architecture](#2-wallet-architecture)
3. [User Wallet Specification](#3-user-wallet-specification)
4. [Master Wallet (Admin) Specification](#4-master-wallet-admin-specification)
5. [Security Specification](#5-security-specification)
6. [White Label System](#6-white-label-system)
7. [Login & Authorization System](#7-login--authorization-system)
8. [Fee Distribution System](#8-fee-distribution-system)
9. [Blockchain Support](#9-blockchain-support)
10. [Operational Logic](#10-operational-logic)

---

## 1. Executive Summary

### 1.1 System Overview

TigerWallet is an enterprise-grade Web3 multi-chain wallet system with:

- **Single 24-word seed phrase** for ALL blockchains (EVM + Non-EVM)
- **Dual wallet system**: User Wallet + Master Wallet (Admin)
- **White Label support** for partners
- **Industrial-grade security** with no vulnerabilities
- **Automatic operations** within 1 second

### 1.2 Core Features

| Feature | Description |
|---------|-------------|
| Multi-chain HD Wallet | Single seed for 100+ blockchains |
| User Wallet | End-user wallet with full features |
| Master Wallet | Admin wallet with complete control |
| White Label | Branded clones for partners |
| TigerSwap Integration | Built-in DEX aggregator |
| Auto-Routing | Best swap routes automatically |
| Fee Management | Configurable fees 0-20% |

---

## 2. Wallet Architecture

### 2.1 System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    TIGERWALLET SYSTEM                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────────┐    ┌──────────────────┐               │
│  │   USER WALLET    │    │  MASTER WALLET  │               │
│  │   (TigerWallet)  │    │  (TigerMaster)  │               │
│  │                 │    │                 │               │
│  │ • 24-word seed  │    │ • 24-word seed  │               │
│  │ • User assets   │    │ • Controls all  │               │
│  │ • Full features│    │ • Auto-signs    │               │
│  │ • Swap/Transfer│    │ • Fee config   │               │
│  └────────┬───────┘    └────────┬───────┘               │
│           │                      │                        │
│           └──────────┬───────────┘                        │
│                      │                                   │
│              ┌──────▼──────┐                           │
│              │  DATABASE  │                            │
│              │ • Users    │                             │
│              │ • Wallets  │                             │
│              │ • Tx History│                            │
│              │ • Config   │                             │
│              └────────────┘                             │
│                                                          │
│  ┌──────────────────────────────────────────────────┐    │
│  │              WHITE LABEL PRODUCTS                │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐         │    │
│  │  │WhiteLabel│ │WhiteLabel│ │WhiteLabel│         │    │
│  │  │   #1   │ │   #2   │ │   #3   │         │    │
│  │  └─────────┘ └─────────┘ └─────────┘         │    │
│  │  Each 20% fees → TigerWallet Admin          │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### 2.2 Wallet Types

| Wallet Type | Seed Phrase | Owner | Features |
|-------------|-------------|-------|----------|
| **User Wallet** | 24-word | End User | Send, Receive, Swap, Stake, NFT, DApps |
| **Master Wallet** | 24-word | Admin | Full control, Auto-sign, Fee config |
| **White Label** | 24-word | Partner | Full TigerWallet features + branding |

---

## 3. User Wallet Specification

### 3.1 User Wallet Overview

**User Wallet (TigerWallet)** is the end-user wallet with complete Web3 functionality.

### 3.2 Seed Phrase System

```
┌─────────────────────────────────────────────────────────┐
│           24-WORD HD SEED PHRASE SYSTEM              │
├─────────────────────────────────────────────────────────┤
│                                                  │
│   Single seed generates addresses for:                  │
│                                                  │
│   ┌──────────────────────────────────────────┐    │
│   │           EVM BLOCKCHAINS                │    │
│   │  • Ethereum      • Polygon              │    │
│   │  • BNB Chain    • Arbitrum            │    │
│   │  • Avalanche    • Optimism            │    │
│   │  • Base         • zkSync             │    │
│   │  • Linea        • Scroll             │    │
│   │  • Mantle       • Blast              │    │
│   │  • 20+ more EVM chains...          │    │
│   └──────────────────────────────────────┘    │
│                                                  │
│   ┌──────────────────────────────────────────┐    │
│   │         NON-EVM BLOCKCHAINS               │    │
│   │  • Bitcoin       • Solana              │    │
│   │  • TON           • Cosmos              │    │
│   │  • Aptos         • Sui               │    │
│   │  • TRON          • Near              │    │
│   │  • Near          • Algorand          │    │
│   │  • 20+ more non-EVM chains...        │    │
│   └──────────────────────────────────────┘    │
│                                                  │
│   Derivation Path: m/44'/coin_type'/0'/0/0        │
│   (BIP-44 Standard)                             │
│                                                  │
└─────────────────────────────────────────────────────────┘
```

### 3.3 Pre-Installed Blockchains

#### EVM Blockchains (20+)

| # | Chain | Symbol | Chain ID | RPC |
|---|------|-------|---------|-----|
| 1 | Ethereum | ETH | 1 | Infura/Alchemy |
| 2 | BNB Smart Chain | BNB | 56 | BSC RPC |
| 3 | Polygon | MATIC | 137 | Polygon RPC |
| 4 | Avalanche | AVAX | 43114 | Avalanche RPC |
| 5 | Arbitrum One | ETH | 42161 | Arbitrum RPC |
| 6 | Optimism | ETH | 10 | Optimism RPC |
| 7 | Base | ETH | 8453 | Base RPC |
| 8 | zkSync Era | ETH | 324 | zkSync RPC |
| 9 | Linea | ETH | 59144 | Linea RPC |
| 10 | Scroll | ETH | 534352 | Scroll RPC |
| 11 | Mantle | MNT | 5000 | Mantle RPC |
| 12 | Blast | ETH | 81457 | Blast RPC |
| 13 | Gnosis | xDAI | 100 | Gnosis RPC |
| 14 | Fantom | FTM | 250 | Fantom RPC |
| 15 | Celo | CELO | 42220 | Celo RPC |
| 16 | Klaytn | KLAY | 8217 | Klaytn RPC |
| 17 | Cronos | CRO | 25 | Cronos RPC |
| 18 | Moonbeam | GLMR | 1284 | Moonbeam RPC |
| 19 | Moonriver | MOVR | 1285 | Moonriver RPC |
| 20 | Astar | ASTR | 592 | Astar RPC |

#### Non-EVM Blockchains (20+)

| # | Chain | Symbol | Type |
|---|------|-------|------|
| 1 | Bitcoin | BTC | UTXO |
| 2 | Solana | SOL | Program |
| 3 | TON | TON | Account |
| 4 | Cosmos | ATOM | Account |
| 5 | Aptos | APT | Move |
| 6 | Sui | SUI | Move |
| 7 | TRON | TRX | Account |
| 8 | Near | NEAR | Account |
| 9 | Algorand | ALGO | Account |
| 10 | Tezos | XTZ | Account |
| 11 | Polkadot | DOT | Account |
| 12 | Kadena | KDA | Account |
| 13 | Hedera | HBAR | Account |
| 14 | VeChain | VET | Account |
| 15 | Flow | FLOW | Account |
| 16 | Conflux | CFX | Account |
| 17 | Sei | SEI | Account |
| 18 | Injective | INJ | Account |
| 19 | Monad | MON | Account |
| 20 | Sui | S | Account |

### 3.4 Pre-Installed Tokens (50+)

#### Native Coins (All blockchain native tokens)

```
Bitcoin (BTC), Ethereum (ETH), BNB (BNB), Solana (SOL), 
TON (TON), Polygon (MATIC), Avalanche (AVAX), 
Arbitrum (ETH), Optimism (ETH), Base (ETH), 
Fantom (FTM), Cronos (CRO), Tron (TRX), 
Cosmos (ATOM), Aptos (APT), Sui (SUI), 
Near (NEAR), Algorand (ALGO), Tezos (XTZ), 
Polkadot (DOT), Kadena (KDA), Hedera (HBAR), 
VeChain (VET), Flow (FLOW), Conflux (CFX), 
Injective (INJ), Sei (SEI), Gnosis (xDAI), 
Celo (CELO), Klaytn (KLAY), Moonbeam (GLMR), 
Astar (ASTR), Linea (LQA), Scroll (SCL), 
Mantle (MNT), Blast (BLAST), zkSync (ZKS)
```

#### Stablecoins (Multi-chain)

```
USDT (Tether), USDC (USD Coin), DAI (Dai), 
BUSD (Binance USD), TUSD (TrueUSD), 
USDP (Pax Dollar), FRAX (Frax), 
USDD (USDD), EURT (Euro Tether), 
EURS (Stasis), GBPt (GBP Token)
```

#### Popular Tokens (ERC-20 + Multi-chain)

```
Wrapped Bitcoin (WBTC), Wrapped Ether (WETH), 
Chainlink (LINK), Uniswap (UNI), Aave (AAVE), 
Maker (MKR), Compound (COMP), Synthetix (SNX), 
Curve (CRV), Lido (LDO), Rocket Pool (RPL), 
SushiSwap (SUSHI), PancakeSwap (CAKE), 
 ApeCoin (APE), Bored Ape (BAYC), OpenSea (OPENSEA), 
 Decentraland (MANA), Sandbox (SAND), 
 Axie (AXS), Enjin (ENJ), The Graph (GRT), 
 Render (RNDR), Fetch.ai (FET), Ocean Protocol (OCEAN), 
 Storj (STORJ), Filecoin (FIL), Arweave (AR), 
 Livepeer (LPT), Band Protocol (BAND), 
 Cosmos (ATOM), Osmosis (OSMO), Celestia (TIA)
```

### 3.5 User Wallet Features

#### Core Operations

| Feature | Description | Auto-Sign |
|---------|-------------|----------|
| **Send** | Transfer any token to any address | No (user signs) |
| **Receive** | Show QR code / address | N/A |
| **Swap** | Exchange tokens via TigerSwap | No (user signs) |
| **Stake** | Stake tokens for rewards | No (user signs) |
| **NFT** | View/Transfer NFTs | No (user signs) |
| **Bridge** | Cross-chain bridges | No (user signs) |
| **DApp Browser** | Connect to DApps | Per session |
| **Airdrop Claim** | Claim airdrops | Master can auto |
| **Liquidity** | Provide liquidity | No (user signs) |

#### Advanced Operations

| Feature | Description |
|---------|-------------|
| MultiSig | M-of-N signature wallets |
| Account Abstraction | EIP-4337 smart wallets |
| MPC | Multi-party computation |
| Social Recovery | Recover via guardians |
| Token Import | Import any token by address |
| Custom Network | Add custom RPC |

### 3.6 TigerSwap Integration

```
┌─────────────────────────────────────────────────────────┐
│                 TIGERSWAP INTEGRATION                    │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   User initiates swap → TigerSwap API finds best route       │
│                                                          │
│   ┌────────────────────────────────────────────────┐    │
│   │           AUTO ROUTE SWITCHING                │    │
│   │                                              │    │
│   │  User Input: 1 ETH                            │    │
│   │  ↓                                         │    │
│   │  [TigerSwap API]                            │    │
│   │  ↓                                         │    │
│   │  ┌────────┐ ┌────────┐ ┌────────┐         │    │
│   │  │Uniswap │ │Curve  │ │ 1inch │          │    │
│   │  │ 15%    │ │ 25%   │ │ 60%   │          │    │
│   │  └────────┘ └────────┘ └────────┘         │    │
│   │       ↓                                     │    │
│   │  Best Route: 1inch (60% liquidity)          │    │
│   │  Output: 3500 USDC                          │    │
│   │  ↓                                         │    │
│   │  [Execute via optimal DEX]                │    │
│   │  ↓                                         │    │
│   │  [User confirms → Transaction signed]       │    │
│   └────────────────────────────────────────────────┘    │
│                                                          │
│   Also supports:                                          │
│   • Paraswap      • Jupiter (Solana)                     │
│   • Orca         • Raydium                              │
│   • PancakeSwap  • DEX aggregators                      │
│   • CEX Integration (Binance, Coinbase, OKX)           │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 3.7 Re-installation Behavior

```
┌─────────────────────────────────────────────────────────┐
│              RE-INSTALLATION HANDLING                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   When user re-installs TigerWallet:                     │
│                                                          │
│   1. User enters 24-word seed phrase                   │
│   2. System derives all addresses from seed             │
│   3. System fetches on-chain balances                  │
│   4. User re-integrates:                              │
│      • Re-connect DApps (session expired)             │
│      • Re-approve token allowances                    │
│      • Re-configure notifications                     │
│      • Re-import custom tokens (if any)              │
│                                                          │
│   ⚠️ IMPORTANT:                                       │
│   • Seed phrase is the ONLY owner proof                │
│   • No cloud backup (fully decentralized)           │
│   • If seed lost → wallet access lost                  │
│   • No password reset possible                        │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 4. Master Wallet (Admin) Specification

### 4.1 Master Wallet Overview

**Master Wallet (TigerMaster)** is the admin wallet with complete control over the system.

### 4.2 Master Wallet Features

```
┌─────────────────────────────────────────────────────────┐
│              MASTER WALLET FEATURES                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │           CORE OPERATIONS                     │    │
│  │  ✅ Auto-sign within 1 second                 │    │
│  │  ✅ Multi-chain transfer                      │    │
│  │  ✅ Create/Import wallets                   │    │
│  │  ✅ Send/Receive                          │    │
│  │  ✅ Swap (TigerSwap + other DEX)           │    │
│  │  ✅ Claim airdrops                       │    │
│  │  ✅ Join campaigns                       │    │
│  │  ✅ Provide liquidity                   │    │
│  │  ✅ Connect DApps                       │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │        BLOCKCHAIN MANAGEMENT                   │    │
│  │  ✅ Add new blockchain                    │    │
│  │  ✅ Remove blockchain                     │    │
│  │  ✅ Update blockchain RPC               │    │
│  │  ✅ Add new tokens/coins                │    │
│  │  ✅ Remove tokens/coins                │    │
│  │  ✅ Update token metadata               │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │            FEE MANAGEMENT                   │    │
│  │  ✅ Set withdrawal fees (0-20%)        │    │
│  │  ✅ Set swap fees (0-20%)            │    │
│  │  ✅ Set transaction fees               │    │
│  │  ✅ Configure fee distribution        │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │          USER MANAGEMENT                   │    │
│  │  ✅ View all user wallets                 │    │
│  │  ✅ Freeze/Unfreeze users               │    │
│  │  ✅ Grant/Remove permissions           │    │
│  │  ✅ Pause features                   │    │
│  │  ✅ Set user limits                  │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │        WHITE LABEL MANAGEMENT                 │    │
│  │  ✅ Approve White Label partners        │    │
│  │  ✅ Revoke White Label                │    │
│  │  ✅ Configure API keys               │    │
│  │  ✅ Track fees                       │    │
│  │  ✅ Destroy White Label              │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 4.3 Auto-Sign System

```
┌─────────────────────────────────────────────────────────┐
│                 AUTO-SIGN SYSTEM                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Master wallet can auto-sign any transaction within      │
│   1 second without manual approval:                     │
│                                                          │
│   ┌──────────────────────────────────────────────┐    │
│   │         AUTO-SIGN TRIGGERS                   │    │
│   │                                              │    │
│   │  1. Revenue collection (auto-sweep)         │    │
│   │     → Triggers: Daily at configured time       │    │
│   │     → Collects: All fees from transactions   │    │
│   │                                              │    │
│   │  2. Airdrop claiming                        │    │
│   │     → Triggers: New airdrop detected        │    │
│   │     → Claims: All eligible airdrops         │    │
│   │                                              │    │
│   │  3. Campaign joining                       │    │
│   │     → Triggers: New campaign found         │    │
│   │     → Joins: All eligible campaigns       │    │
│   │                                              │    │
│   │  4. Liquidity management                   │    │
│   │     → Triggers: Threshold reached          │    │
│   │     → Provides: Liquidity as configured  │    │
│   │                                              │    │
│   │  5. Emergency operations                 │    │
│   │     → Triggers: Security alert           │    │
│   │     → Acts: Freeze/Drain as needed         │    │
│   └──────────────────────────────────────────────┘    │
│                                                          │
│   ⚡ All auto-sign operations complete in <1 second   │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 4.4 Fee Configuration

```
┌─────────────────────────────────────────────────────────┐
│               FEE CONFIGURATION                       │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Master wallet configures all fees:                  │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │            FEE TYPES                        │    │
│   │                                              │    │
│   │  1. Withdrawal Fee                          │    │
│   │     Range: 0% - 20%                         │    │
│   │     Default: 0.5%                          │    │
│   │                                              │    │
│   │  2. Swap Fee                                │    │
│   │     Range: 0% - 20%                        │    │
│   │     Default: 0.3%                          │    │
│   │                                              │    │
│   │  3. Transaction Fee                        │    │
│   │     Range: 0% - 20%                        │    │
│   │     Default: Network gas + 0.1%          │    │
│   │                                              │    │
│   │  4. Bridge Fee                              │    │
│   │     Range: 0% - 20%                        │    │
│   │     Default: 0.5%                          │    │
│   │                                              │    │
│   │  5. NFT Transfer Fee                      │    │
│   │     Range: 0% - 20%                        │    │
│   │     Default: 0.5%                          │    │
│   │                                              │    │
│   │  6. Staking Fee                            │    │
│   │     Range: 0% - 20%                        │    │
│   │     Default: 1%                            │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   Fee Distribution:                                   │
│   ┌─────────────────────────────────────────────┐    │
│   │  Collected Fees → Master Wallet            │    │
│   │  Auto-sweep to master wallet address       │    │
│   │  No manual collection needed            │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 4.5 Master Wallet Backup

```
┌─────────────────────────────────────────────────────────┐
│              BACKUP & RECOVERY                        │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Master wallet 24-word seed is:                      │
│                                                          │
│   1. Saved encrypted in admin dashboard              │
│   2. Encrypted with master password               │
│   3. Stored in secure enclave                   │
│   4. Backup downloadable (encrypted)             │
│   5. Multi-location storage (redundancy)        │
│                                                          │
│   ⚠️ CRITICAL:                                     │
│   • Master seed controls ALL user wallets         │
│   • If master seed lost → system compromised    │
│   • Must be stored in multiple secure locations│
│   • Only super admin has access                │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 5. Security Specification

### 5.1 Security Architecture

```
┌─────────────────────────────────────────────────────────┐
│           INDUSTRIAL-GRADE SECURITY                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │         CRYPTOGRAPHIC SECURITY                 │    │
│  │                                              │    │
│  │  • AES-256-GCM encryption for all data        │    │
│  │  • Ed25519 signatures (modern standard)      │    │
│  │  • secp256k1 for EVM compatibility          │    │
│  │  • Argon2id password hashing               │    │
│  │  • HKDF key derivation                   │    │
│  │  • X25519 key exchange                  │    │
│  │  • ChaCha20-Poly1305 for streaming      │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │           NETWORK SECURITY                   │    │
│  │                                              │    │
│  │  • TLS 1.3 (mandatory)                     │    │
│  │  • Certificate pinning                    │    │
│  │  • Mutual TLS (mTLS) for API               │    │
│  │  • DNSSEC for domain verification         │    │
│  │  • HSTS (HTTP Strict Transport Security) │    │
│  │  • CSP (Content Security Policy)         │    │
│  │  • Anti-replay protection                 │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │           APPLICATION SECURITY             │    │
│  │                                              │    │
│  │  • Input validation (strict)               │    │
│  │  • SQL injection prevention               │    │
│  │  • XSS prevention (output encoding)       │    │
│  │  • CSRF tokens (all forms)               │    │
│  │  • Rate limiting (all endpoints)         │    │
│  │  • Request size limits                  │    │
│  │  • Parameterized queries                │    │
│  │  • Secure session management             │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │          INFRASTRUCTURE SECURITY             │    │
│  │                                              │    │
│  │  • WAF (Web Application Firewall)        │    │
│  │  • DDoS protection (Cloudflare/Akamai)    │    │
│  │  • IDS/IPS (Intrusion Detection)          │    │
│  │  • File integrity monitoring            │    │
│  │  • Log analysis (SIEM)                  │    │
│  │  • 24/7 security monitoring            │    │
│  │  • Automated threat response           │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │           WALLET SECURITY                 │    │
│  │                                              │    │
│  │  • Seed encryption (AES-256)             │    │
│  │  • Secure enclave storage               │    │
│  │  • Biometric authentication           │    │
│  │  • Hardware security module (HSM)      │    │
│  │  • Multi-sig support                  │    │
│  │  • Transaction simulation            │    │
│  │  • Address reputation checking       │    │
│  │  • Smart contract scanning           │    │
│  └────────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 5.2 Attack Prevention

```
┌─────────────────────────────────────────────────────────┐
│            ATTACK PREVENTION MATRIX                    │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ATTACK TYPE              │ PREVENTION                   │
│  ─────────────────────│─────────────────────────────  │
│                                                          │
│  DDOS Attacks          │ • Rate limiting              │
│                       │ • CDN protection            │
│                       │ • Traffic filtering       │
│                       │ • Auto-scaling             │
│                       │ • IP reputation           │
│                       │                            │
│  XSS Attacks          │ • Output encoding           │
│                       │ • Content Security Policy │
│                       │ • DOM sanitization        │
│                       │ • Script allowlist        │
│                       │                            │
│  SQL Injection        │ • Parameterized queries    │
│                       │ • ORM usage               │
│                       │ • Input validation       │
│                       │ • Least privilege DB     │
│                       │                            │
│  Phishing             │ • Anti-phishing detection │
│                       │ • URL verification      │
│                       │ • Domain monitoring    │
│                       │ • User education        │
│                       │                            │
│  CSRF                 │ • Token-based CSRF        │
│                       │ • SameSite cookies       │
│                       │ • Origin verification   │
│                       │                            │
│  Man-in-Middle        │ • TLS 1.3 only         │
│                       │ • Certificate pinning  │
│                       │ • HSTS enabled         │
│                       │                            │
│  Replay Attacks      │ • Nonce validation      │
│                       │ • Timestamp checks     │
│                       │ • One-time tokens     │
│                       │                            │
│  Credential Stuffing │ • Rate limiting          │
│                       │ • CAPTCHA              │
│                       │ • 2FA mandatory       │
│                       │ • Breach detection    │
│                       │                            │
│  Session Hijacking   │ • Secure cookies        │
│                       │ • Session rotation    │
│                       │ • Device fingerprint │
│                       │ • IP validation      │
│                       │                            │
│  Keylogging         │ • Virtual keyboard      │
│                       │ • Clipboard protection│
│                       │ • Secure input       │
│                       │                            │
│  Smart Contract     │ • Tenderly simulation │
│  Attacks           │ • Flash loan guards   │
│                       │ • Slippage limits   │
│                       │ • Contract verification│
│                       │                            │
└─────────────────────────────────────────────────────────┘
```

### 5.3 Encryption Standards

```
┌─────────────────────────────────────────────────────────┐
│            ENCRYPTION STANDARDS                        │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  DATA AT REST:                                            │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Method          │  Standard    │  Key Size     │    │
│  │  ───────────────│─────────────│────────────  │    │
│  │  Storage       │  AES-256-GCM │  256-bit    │    │
│  │  Database     │  AES-256-GCM │  256-bit    │    │
│  │  Backup      │  AES-256-GCM │  256-bit    │    │
│  │  Password    │  Argon2id   │  Variable   │    │
│  │  Seed Phrase │  AES-256-GCM │  256-bit    │    │
│  │  API Keys    │  AES-256-GCM │  256-bit    │    │
│  │  Logs      │  AES-256-GCM │  256-bit    │    │
│  └─────────────────────────────────────────────────┘    │
│                                                          │
│  DATA IN TRANSIT:                                         │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Method          │  Standard    │  Notes       │    │
│  │  ───────────────│─────────────│───────────  │    │
│  │  API Traffic   │  TLS 1.3   │  Mandatory  │    │
│  │  User → App  │  TLS 1.3   │  Mandatory  │    │
│  │  App → Node │  TLS 1.3   │  Mandatory  │    │
│  │  Internal   │  mTLS     │  Service   │    │
│  │  Blockchain │  TLS 1.3   │  RPC      │    │
│  └─────────────────────────────────────────────────┘    │
│                                                          │
│  KEY DERIVATION:                                           │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Method          │  Standard    │  Notes       │    │
│  │  ───────────────│─────────────│───────────  │    │
│  │  Wallet Keys    │  BIP-39/44  │  HD Wallet  │    │
│  │  API Keys      │  HKDF      │  Key Gen   │    │
│  │  Passwords    │  Argon2id  │  3 rounds  │    │
│  │  Sessions    │  HMAC-SHA256│  Rotation  │    │
│  │  Signatures   │  Ed25519   │  Modern   │    │
│  └─────────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 6. White Label System

### 6.1 White Label Overview

```
┌─────────────────────────────────────────────────────────┐
│              WHITE LABEL SYSTEM                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   White Label = 100% clone of TigerWallet                 │
│   Each White Label is a fully functional Web3 wallet       │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │         WHITE LABEL FEATURES                 │    │
│   │                                              │    │
│   │  ✅ All TigerWallet features                  │    │
│   │  ✅ Custom branding (logo, colors)        │    │
│   │  ✅ Custom domain                         │    │
│   │  ✅ Separate cloud/storage                │    │
│   │  ✅ 20% fees to TigerWallet               │    │
│   │  ✅ Unique ID tracking                   │    │
│   │  ✅ API key authorization               │    │
│   │  ✅ Can be destroyed by admin          │    │
│   │  ⚠️ No White Label system inside        │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │         PARTNER BENEFITS                   │    │
│   │                                              │    │
│   │  • Launch own branded wallet               │    │
│   │  • Keep 80% of all fees               │    │
│   │  • Full control of features           │    │
│   │  • Create own admin team              │    │
│   │  • Custom integrations              │    │
│   │  • Independent operations           │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 6.2 White Label Requirements

```
┌─────────────────────────────────────────────────────────┐
│           WHITE LABEL REQUIREMENTS                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   To create White Label:                                 │
│                                                          │
│   1. APPROVAL REQUIRED                               │
│      □ Submit application to TigerWallet admin       │
│      □ Review and approval process                  │
│      □ Sign partnership agreement                  │
│                                                          │
│   2. TECHNICAL REQUIREMENTS                        │
│      □ Unique White Label ID (assigned)            │
│      □ API Keys (generated by TigerWallet)         │
│      □ Custom domain setup                        │
│      □ Cloud infrastructure                     │
│      □ SSL certificate                           │
│                                                          │
│   3. FINANCIAL REQUIREMENTS                       │
│      □ 20% fees to TigerWallet (automatic)        │
│      □ Setup fee (if applicable)                 │
│      □ Annual renewal fee                        │
│                                                          │
│   4. SECURITY REQUIREMENTS                        │
│      □ Same security standards as TigerWallet    │
│      □ Regular security audits                  │
│      □ Compliance requirements                 │
│                                                          │
│   5. OPERATIONAL REQUIREMENTS                     │
│      □ 24/7 support for users                │
│      □ SLA compliance                        │
│      □ KYC/AML compliance                     │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 6.3 White Label Fee Distribution

```
┌─────────────────────────────────────────────────────────┐
│           FEE DISTRIBUTION (WHITE LABEL)              │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Every transaction fee split:                        │
│                                                          │
│   ┌────────────────────────────────────────────┐    │
│   │         TRANSACTION FEE BREAKDOWN            │    │
│   │                                            │    │
│   │  User pays: $100 swap fee                  │    │
│   │                                            │    │
│   │  ┌────────────────────────────────────┐  │    │
│   │  │         $100                       │  │    │
│   │  │          ↓                        │  │    │
│   │  │  ┌─────────┴─────────┐            │  │    │
│   │  │  ↓                ↓               │  │    │
│   │  │ $80              $20              │  │    │
│   │  │ (80%)           (20%)             │  │    │
│   │  │  ↓                ↓               │  │    │
│   │  │ Partner         TigerWallet      │  │    │
│   │  │ Wallet         Admin            │  │    │
│   │  └───────────────────────────────┘  │    │
│   └────────────────────────────────────┘    │
│                                                          │
│   Fee Types:                                             │
│   • Swap fees: 20% → TigerWallet                        │
│   • Trading fees: 20% → TigerWallet                     │
│   • Transaction fees: 20% → TigerWallet               │
│   • Withdrawal fees: 20% → TigerWallet                │
│   • NFT fees: 20% → TigerWallet                       │
│   • All other fees: 20% → TigerWallet                  │
│                                                          │
│   ⚡ Auto-distribution via smart contract              │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 6.4 White Label API Key System

```
┌─────────────────────────────────────────────────────────┐
│              API KEY SYSTEM                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   White Label requires valid API key to function:      │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │         API KEY VALIDATION                  │    │
│   │                                              │    │
│   │  White Label Request → TigerWallet API         │    │
│   │  ↓                                         │    │
│   │  [Check API Key]                            │    │
│   │  ↓                                         │    │
│   │  ┌────────┐ ┌────────┐ ┌────────┐       │    │
│   │  │Valid  │ │Revoked│ │Invalid│       │    │
│   │  │      │ │      │ │      │       │    │
│   │  │Allow │ │Block │ │Block │       │    │
│   │  │      │ │      │ │      │       │    │
│   │  └────────┘ └────────┘ └────────┘       │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   Invalid Key Message:                                 │
│   ╔════════════════════════════════════════════╗    │
│   ║  "⚠️ Invalid API Key                      ║    │
│   ║   Please input authorized API keys.        ║    │
│   ║   Contact TigerWallet admin."             ║    │
│   ╚════════════════════════════════════════════╝    │
│                                                          │
│   Revoked Key Message:                                 │
│   ╔════════════════════════════════════════════╗    │
│   ║  "⚠️ API Key Revoked                    ║    │
│   ║   Your White Label has been deactivated.║    │
│   ║   Contact TigerWallet admin."           ║    │
│   ╚════════════════════════════════════════════╝    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 6.5 White Label ID Tracking

```
┌─────────────────────────────────────────────────────────┐
│              ID TRACKING SYSTEM                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Each White Label tracked by unique ID:                  │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │         WHITE LABEL DATABASE               │    │
│   │                                              │    │
│   │  ID: WL-00001                              │    │
│   │  Name: "PartnerWallet"                    │    │
│   │  Domain: "wallet.partner.com"           │    │
│   │  API Key: "wl_abc123xyz..."              │    │
│   │  Created: 2026-01-15                   │    │
│   │  Status: Active                         │    │
│   │  Total Fees: $1,000,000               │    │
│   │  TigerWallet Share: $200,000 (20%)     │    │
│   │  Partner Share: $800,000 (80%)          │    │
│   │  Users: 50,000                         │    │
│   │  Transactions: 1,000,000              │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   Admin can view:                                      │
│   • All White Label IDs                              │
│   • Revenue per White Label                       │
│   • Transaction volumes                       │
│   • User counts                                │
│   • Fee distributions                          │
│   • Performance metrics                        │
│   • Destroy White Label                       │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 7. Login & Authorization System

### 7.1 Login System Overview

```
┌─────────────────────────────────────────────────────────┐
│           LOGIN & AUTHORIZATION SYSTEM                  │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌─────────────────────────────────────────────┐    │
│  │            USER ROLES                      │    │
│  │                                              │    │
│  │  1. TIGERWALLET SUPER ADMIN                │    │
│  │     → Complete system control             │    │
│  │     → Create/manage admins               │    │
│  │     → Configure all settings            │    │
│  │     → Approve White Labels               │    │
│  │     → View all data                     │    │
│  │                                              │    │
│  │  2. TIGERWALLET ADMIN                   │    │
│  │     → Manage users                     │    │
│  │     → Configure fees                   │    │
│  │     → View analytics                   │    │
│  │     → Manage features                  │    │
│  │                                              │    │
│  │  3. WHITE LABEL SUPER ADMIN            │    │
│  │     → Control own White Label           │    │
│  │     → Create admins                    │    │
│  │     → Configure branding              │    │
│  │     → View own analytics             │    │
│  │                                              │    │
│  │  4. WHITE LABEL ADMIN                │    │
│  │     → Manage users                   │    │
│  │     → View analytics                │    │
│  │     → Configure features            │    │
│  │                                              │    │
│  │  5. USER                              │    │
│  │     → Use wallet features            │    │
│  │     → Manage own wallet             │    │
│  │     → View own transactions         │    │
│  │                                              │    │
│  └─────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 7.2 Super Admin Login Requirements

```
┌─────────────────────────────────────────────────────────┐
│          LOGIN SECURITY REQUIREMENTS                  │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Super Admin Login (TigerWallet Admin):                │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│  │          MANDATORY SECURITY                │    │
│  │                                              │    │
│  │  ✅ Email verification (real email)        │    │
│  │  ✅ Phone number verification             │    │
│  │  ✅ 2FA (TOTP + SMS)                     │    │
│  │  ✅ Government ID verification            │    │
│  │  ✅ Biometric verification              │    │
│  │  ✅ Device fingerprinting               │    │
│  │  ✅ IP allowlist                       │    │
│  │  ✅ Session timeout (15 min)          │    │
│  │  ✅ Login alerts (email + SMS)         │    │
│  │  ✅ Failed attempt lockout (5 tries)   │    │
│  │  ✅ Password complexity (16+ chars)   │    │
│  │  ✅ Annual re-verification            │    │
│  └─────────────────────────────────────────────┘    │
│                                                          │
│   Login Process:                                        │
│   ┌─────────────────────────────────────────────┐    │
│   │  1. Enter email + password                 │    │
│   │  2. Email verification code              │    │
│   │  3. Phone verification code             │    │
│   │  4. 2FA (authenticator app)              │    │
│   │  5. Biometric verification             │    │
│   │  6. Device confirmation                │    │
│   │  7. IP verification                   │    │
│   │  8. Success → Dashboard access          │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 7.3 Permission System

```
┌─────────────────────────────────────────────────────────┐
│            PERMISSION SYSTEM                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   Super Admin can:                                      │
│   ┌─────────────────────────────────────────────┐    │
│   │  ✅ Grant permissions to admins             │    │
│   │  ✅ Remove permissions from admins        │    │
│   │  ✅ Pause user accounts                  │    │
│   │  ✅ Freeze features                     │    │
│   │  ✅ Set feature limits                 │    │
│   │  ✅ Configure fee percentages         │    │
│   │  ✅ Manage White Labels                │    │
│   │  ✅ View all transactions             │    │
│   │  ✅ Export data                       │    │
│   │  ⚠️ Cannot access user seed phrases    │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   Permission Types:                                    │
│   ┌─────────────────────────────────────────────┐    │
│   │  Feature          │  Grant  │  Remove       │    │
│   │  ───────────────│────────│─────────────  │    │
│   │  Swap          │   ✅   │    ✅        │    │
│   │  Send          │   ✅   │    ✅        │    │
│   │  Receive      │   ✅   │    ✅        │    │
│   │  NFT         │   ✅   │    ✅        │    │
│   │  Stake       │   ✅   │    ✅        │    │
│   │  Bridge      │   ✅   │    ✅        │    │
│   │  DApp       │   ✅   │    ✅        │    │
│   │  Fiat       │   ✅   │    ✅        │    │
│   │  Withdraw   │   ✅   │    ✅        │    │
│   │  Admin     │   ✅   │    ✅        │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   Granular Permission Control:                          │
│   • Per user: Enable/disable features                │
│   • Per feature: Set limits                          │
│   • Per time: Time-based access                     │
│   • Per IP: IP-based access                         │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 8. Fee Distribution System

### 8.1 Fee Types

```
┌─────────────────────────────────────────────────────────┐
│              FEE DISTRIBUTION SYSTEM                  │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   FEE TYPES AND RATES:                                 │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │  Fee Type       │  Default  │  Max  │  Who   │    │
│   │  ────────────│──────────│───────│────────│    │
│   │  Swap Fee    │   0.3%  │  20%  │  User │    │
│   │  Withdraw   │   0.5%  │  20%  │  User │    │
│   │  Bridge    │   0.5%  │  20%  │  User │    │
│   │  NFT Trade │   0.5%  │  20%  │  User │    │
│   │  Stake    │    1%   │  20%  │  User │    │
│   │  Liquidity│   0.3%  │  20%  │  User │    │
│   │  Fiat    │    2%   │  20%  │  User │    │
│   │  Network │  Gas cost│  N/A  │  User │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   WHITE LABEL FEE SPLIT:                                │
│   ┌─────────────────────────────────────────────┐    │
│   │  Total Fee: 100%                           │    │
│   │  ↓                                        │    │
│   │  TigerWallet: 20%                        │    │
│   │  White Label: 80%                        │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   MASTER WALLET FEE COLLECTION:                             │
│   ┌─────────────────────────────────────────────┐    │
│   │  All fees → Master Wallet automatically     │    │
│   │  Auto-sweep on schedule                  │    │
│   │  No manual collection needed              │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 8.2 Fee Collection Flow

```
┌─────────────────────────────────────────────────────────┐
│              FEE COLLECTION FLOW                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   User Transaction:                                    │
│   ┌─────────────────────────────────────────────┐    │
│   │  User initiates swap ($1000)                │    │
│   │  ↓                                         │    │
│   │  [Swap executed on DEX]                    │    │
│   │  ↓                                         │    │
│   │  [Swap fee calculated: $3 (0.3%)]          │    │
│   │  ↓                                         │    │
│   │  [Fee split: TigerWallet = $0.60]            │    │
│   │      [White Label = $2.40]                 │    │
│   │  ↓                                         │    │
│   │  [TigerWallet: Auto-transfer to admin]      │    │
│   │  [White Label: Auto-transfer to partner]   │    │
│   │  ↓                                         │    │
│   │  [Transaction confirmed]                   │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   Real-time tracking:                                   │
│   • Dashboard shows live fee collection               │
│   • Per White Label breakdown                     │
│   • Per transaction type breakdown            │
│   • Historical trends                            │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 9. Blockchain Support

### 9.1 Complete Blockchain List

```
┌─────────────────────────────────────────────────────────┐
│           SUPPORTED BLOCKCHAINS                       │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   EVM COMPATIBLE (20+):                                │
│   ┌─────────────────────────────────────────────┐    │
│   │ 1. Ethereum (ETH)         Chain ID: 1      │    │
│   │ 2. BNB Smart Chain (BNB) Chain ID: 56      │    │
│   │ 3. Polygon (MATIC)       Chain ID: 137      │    │
│   │ 4. Avalanche (AVAX)       Chain ID: 43114    │    │
│   │ 5. Arbitrum (ETH)         Chain ID: 42161    │    │
│   │ 6. Optimism (ETH)         Chain ID: 10       │    │
│   │ 7. Base (ETH)            Chain ID: 8453    │    │
│   │ 8. zkSync Era (ETH)     Chain ID: 324      │    │
│   │ 9. Linea (ETH)          Chain ID: 59144    │    │
│   │ 10. Scroll (ETH)         Chain ID: 534352   │    │
│   │ 11. Mantle (MNT)        Chain ID: 5000    │    │
│   │ 12. Blast (ETH)          Chain ID: 81457    │    │
│   │ 13. Gnosis (xDAI)       Chain ID: 100     │    │
│   │ 14. Fantom (FTM)        Chain ID: 250     │    │
│   │ 15. Celo (CELO)         Chain ID: 42220    │    │
│   │ 16. Klaytn (KLAY)       Chain ID: 8217    │    │
│   │ 17. Cronos (CRO)        Chain ID: 25      │    │
│   │ 18. Moonbeam (GLMR)     Chain ID: 1284    │    │
│   │ 19. Moonriver (MOVR)     Chain ID: 1285    │    │
│   │ 20. Astar (ASTR)       Chain ID: 592     │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   NON-EVM (20+):                                    │
│   ┌─────────────────────────────────────────────┐    │
│   │ 1. Bitcoin (BTC)         UTXO             │    │
│   │ 2. Solana (SOL)         Program          │    │
│   │ 3. TON (TON)            Account         │    │
│   │ 4. Cosmos (ATOM)        Account         │    │
│   │ 5. Aptos (APT)          Move            │    │
│   │ 6. Sui (SUI)            Move            │    │
│   │ 7. TRON (TRX)           Account         │    │
│   │ 8. Near (NEAR)          Account         │    │
│   │ 9. Algorand (ALGO)      Account         │    │
│   │ 10. Tezos (XTZ)        Account         │    │
│   │ 11. Polkadot (DOT)      Account         │    │
│   │ 12. Kadena (KDA)        Account         │    │
│   │ 13. Hedera (HBAR)       Account         │    │
│   │ 14. VeChain (VET)       Account         │    │
│   │ 15. Flow (FLOW)         Account         │    │
│   │ 16. Conflux (CFX)      Account         │    │
│   │ 17. Sei (SEI)           Account         │    │
│   │ 18. Injective (INJ)    Account         │    │
│   │ 19. Monad (MON)         Account         │    │
│   │ 20. Sui (S)            Account         │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   TOTAL: 40+ Blockchains pre-installed               │
│   + Custom blockchain support (admin can add)      │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 10. Operational Logic

### 10.1 Real-Time Operations

```
┌─────────────────────────────────────────────────────────┐
│            OPERATIONAL LOGIC                         │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   USER WALLET OPERATIONS:                              │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │  SEND OPERATION                            │    │
│   │  1. User selects token + amount           │    │
│   │  2. User enters recipient address        │    │
│   │  3. System estimates gas (if EVM)        │    │
│   │  4. User confirms transaction          │    │
│   │  5. User signs with seed phrase         │    │
│   │  6. Transaction broadcasted          │    │
│   │  7. Confirmation received             │    │
│   │  Total time: ~15 seconds             │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │  SWAP OPERATION                          │    │
│   │  1. User selects token to swap         │    │
│   │  2. User selects token to receive     │    │
│   │  3. System finds best route          │    │
│   │  4. Auto-switch to best DEX           │    │
│   │  5. User confirms                   │    │
│   │  6. User signs transaction          │    │
│   │  7. Transaction executed           │    │
│   │  8. Confirmation received           │    │
│   │  Total time: ~30 seconds            │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
│   MASTER WALLET OPERATIONS:                            │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │  AUTO-SIGN OPERATION                      │    │
│   │  1. Trigger event detected              │    │
│   │  2. System prepares transaction      │    │
│   │  3. Master wallet signs              │    │
│   │  4. Transaction broadcasted          │    │
│   │  5. Confirmation received           │    │
│   │  ⚡ Total time: <1 second            │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 10.2 Transaction Flow

```
┌─────────────────────────────────────────────────────────┐
│           TRANSACTION FLOW                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   COMPLETE TRANSACTION LIFECYCLE:                    │
│                                                          │
│   ┌─────────────────────────────────────────────┐    │
│   │  STEP 1: Initiation                        │    │
│   │  User clicks "Send" or "Swap"              │    │
│   └─────────────────────────────────────────────┘    │
│            ↓                                         │
│   ┌─────────────────────────────────────────────┐    │
│   │  STEP 2: Validation                        │    │
│   │  • Balance check                          │    │
│   │  • Address validation                    │    │
│   │  • Token validation                     │    │
│   │  • Permission check                    │    │
│   └─────────────────────────────────────────────┘    │
│            ↓                                         │
│   ┌─────────────────────────────────────────────┐    │
│   │  STEP 3: Route Finding (Swap)             │    │
│   │  • Multiple DEX query                    │    │
│   │  • Best route selection                 │    │
│   │  • Auto-switch                        │    │
│   │  • Slippage calculation                │    │
│   └─────────────────────────────────────────────┘    │
│            ↓                                         │
│   ┌─────────────────────────────────────────────┐    │
│   │  STEP 4: Fee Calculation                  │    │
│   │  • Network gas                          │    │
│   │  • Platform fee                         │    │
│   │  • White Label share                   │    │
│   │  • TigerWallet share                   │    │
│   └─────────────────────────────────────────────┘    │
│            ↓                                         │
│   ┌─────────────────────────────────────────────┐    │
│   │  STEP 5: User Confirmation               │    │
│   │  • Show transaction details            │    │
│   │  • Show fees                          │    │
│   │  • User signs                         │    │
│   └─────────────────────────────────────────────┘    │
│            ↓                                         │
│   ┌─────────────────────────────────────────────┐    │
│   │  STEP 6: Signing                         │    │
│   │  • Derive key from seed                │    │
│   │  • Sign transaction                  │    │
│   │  • Return signature                  │    │
│   └─────────────────────────────────────────────┘    │
│            ↓                                         │
│   ┌─────────────────────────────────────────────┐    │
│   │  STEP 7: Broadcasting                   │    │
│   │  • Submit to network                 │    │
│   │  • Get transaction hash               │    │
│   │  • Wait for confirmation             │    │
│   └─────────────────────────────────────────────┘    │
│            ↓                                         │
│   ┌─────────────────────────────────────────────┐    │
│   │  STEP 8: Confirmation                   │    │
│   │  • Transaction confirmed              │    │
│   │  • Update balance                    │    │
│   │  • Record history                  │    │
│   │  • Distribute fees                  │    │
│   └─────────────────────────────────────────────┘    │
│            ↓                                         │
│   ┌─────────────────────────────────────────────┐    │
│   │  STEP 9: Notification                   │    │
│   │  • Email notification                │    │
│   │  • Push notification                │    │
│   │  • Update UI                         │    │
│   └─────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 11. Appendix

### 11.1 Technology Stack Summary

| Layer | Technology |
|-------|-------------|
| Mobile | Flutter/Dart + Rust FFI |
| Web Extension | TypeScript + React |
| Desktop | Flutter + Rust / Tauri |
| Wallet Core | Rust |
| Backend | Go |
| AI/ML | Python |
| Database | PostgreSQL + Redis + ClickHouse |
| Security | AES-256-GCM + TLS 1.3 |

### 11.2 Security Certifications Required

- SOC 2 Type II
- ISO 27001
- PCI-DSS Level 1
- GDPR Compliant

### 11.3 Compliance

- KYC/AML (per jurisdiction)
- Travel Rule
- OFAC Sanctions
- SEC Regulation

---

## Document Control

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-06-08 | TigerWallet | Initial specification |

---

*This specification is confidential and proprietary to TigerWallet.*