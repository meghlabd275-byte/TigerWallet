# UserWallet - Complete Feature Analysis

> **✅ STATUS UPDATE (2026-08-12 #2): PARAM-CONTRACT PARITY + DEDUP COMPLETE.**
> A fresh parity audit found route coverage complete (no 404s) but **parameter
> contracts** broken (400s / wrong data). Fixed in `go/wallet_api` (backend made
> permissive, no client churn): `/auth/register` `username` now optional (derived
> from email); `/price` accepts `coin`/`symbol`/`token`; `/swap/quote` accepts
> both `from`/`to`/`amount` and `from_token`/`to_token`/`from_amount`;
> `/swap/execute` now constructs swap calldata **server-side** from the chain's
> V2 router (real on-chain `getAmountsOut` + ABI); `/staking/*` returns `202`
> `provide_staking_contract` (not 400). **Redundant fake-crypto backend removed:**
> `user_services/go` (:8081, sha256-mnemonic/deriveAddress) → stdlib reverse-proxy
> shim to :8443 (port preserved; old impl as `legacy_main.go.txt`, not compiled).
> **SQLite fully removed** (zero active usage; PostgreSQL + Redis only).
> `go build`+`go vet`+`go test` pass; 9 DeFi Go services + 3 Rust fetchers
> (cargo check, userwallet 3/3 tests) + desktop_wallet C++ (cmake/make) + Foundry
> (31/31 tests) all green.

> **✅ STATUS UPDATE (2026-08-12): FULL CLIENT PARITY + BUILD VERIFICATION COMPLETE.**
> Building on the 2026-08-11 retarget, all four UserWallet native clients
> (`user_wallet/web`, `user_wallet/desktop`, `user_wallet/android`,
> `user_wallet/ios`) now expose the **identical fetcher set** against the
> canonical `go/wallet_api` (:8443): `login`/`register`, `getWallets`/
> `createWallet`, `getBalances`/`getBalance`, `getTransactions`,
> `sendTransaction`, `signMessage`, `getTokenBalances`, `getNFTs`,
> `getTokenPrice`, `getChains`, `getGasPrice`, `getNetworkStatus`,
> `getSwapQuote`, `getStakingQuote`. No stubs, no fabricated data —
> `getNetworkStatus` derives from `/chains` (block_number honestly `0`).
> **Build verification (all green):** `frontend/web_nextjs` tsc → 0 errors;
> `user_wallet/web` tsc → 0 errors; `go/wallet_api` build+tests pass (BIP-44
> vector); `desktop_wallet` C++ cmake/make exit 0 + tests pass; Foundry
> `forge build` exit 0, `forge test` 31/31 pass (real ECDSA via `vm.sign`, no
> mocks); OpenZeppelin v5 installed via `forge install` (was absent from the
> shallow clone). Commit `f2bda9b` on `main`.

> **STATUS UPDATE (2026-08-11):** The user-wallet client stack now matches the
> parity matrix below across all platforms. Verified: all clients target canonical
> `go/wallet_api` (:8443); 0 `Math.random()` fake-crypto calls remain (replaced
> with real backend calls / CSPRNG / fail-closed throws); `rust/userwallet_fetchers`
> builds clean (delegates to wallet_api, no stubs); `frontend/web_nextjs` wallet
> transactions.ts EVM path fully wired + dynamic `/api/v1/transactions/[txHash]`
> route; light/dark theme on every web_nextjs page (0 `dark:` variants) and mobile
> (Android ThemeManager.kt, iOS ThemeManager.swift, Flutter theme_provider.dart);
> `permission_service`/`connection_api`/`monitoring_dashboard` build+vet clean
> (bcrypt for passwords; PostgreSQL+Redis, no SQLite). Mobile (Flutter/Android/iOS)
> buildable (pubspec.yaml present, android compiles).

## Overview
UserWallet is for end-users to trade, send/receive crypto, and use DeFi features. It NEVER has admin or master wallet functionality.

---

## Current UserWallet Features

### 1. Trading Features
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| P2P Trading | Peer-to-peer crypto trading | YES | YES | YES | YES | YES | YES |
| P2P Merchant | Become a merchant | YES | YES | WARN | YES | YES | YES |
| Margin Trading | Leverage trading | YES | YES | YES | YES | YES | YES |
| Futures Trading | Perpetual contracts | YES | YES | YES | YES | YES | YES |
| Options Trading | Call/Put options | YES | YES | NO | YES | YES | YES |
| Copy Trading | Follow traders | YES | YES | YES | YES | YES | YES |
| Convert | Instant conversion | YES | YES | YES | YES | YES | YES |
| Swap/DEX | Decentralized exchange | YES | YES | YES | YES | YES | YES |

### 2. Wallet Features
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| Multi-chain Wallet | Support 10+ chains | YES | YES | YES | YES | YES | YES |
| Send/Receive | Transfer crypto | YES | YES | YES | YES | YES | YES |
| Address Book | Saved addresses | YES | YES | YES | YES | YES | YES |
| QR Code | Scan/pay QR | YES | YES | YES | YES | YES | YES |
| Hardware Wallet | Ledger/Trezor | YES NEW | YES | NO | YES NEW | YES NEW | YES |
| MPC Wallet | Multi-party computation | YES NEW | YES | NO | YES NEW | YES NEW | YES |
| Social Recovery | Guardian-based recovery | YES NEW | YES | NO | YES NEW | YES NEW | YES |
| Account Abstraction | Smart accounts | YES NEW | YES | NO | YES NEW | YES NEW | YES |

### 3. DeFi Features
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| Staking | Proof-of-stake | YES | YES | YES | YES | YES | YES |
| Liquid Staking | Staking tokens | WARN | YES | NO | WARN | WARN | NO |
| Lending | Supply/Borrow | YES NEW | YES | NO | YES NEW | YES NEW | YES NEW |
| Bridge | Cross-chain | YES NEW | YES | NO | YES NEW | YES NEW | YES NEW |
| Farming | Yield farming | YES NEW | YES | NO | YES | YES | NO |
| DAO | Governance | YES NEW | YES | NO | NO | NO | NO |

### 4. NFT Features
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| NFT Gallery | View collections | YES | YES | YES | YES | YES | NO |
| NFT Trading | Buy/Sell NFTs | YES | YES | YES | YES | YES | NO |
| NFT Mint | Create NFTs | YES | YES | NO | YES | YES | NO |

### 5. Payments
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| Crypto Card | Virtual card | YES | YES | YES | YES | YES | YES |
| Fiat On-Ramp | Buy crypto | YES | YES | YES | YES | YES | YES |
| Fiat Off-Ramp | Sell crypto | YES | YES | YES | YES | YES | YES |
| Gift Cards | Buy/Sell gift cards | YES NEW | YES | NO | YES NEW | YES NEW | YES NEW |

### 6. DApp & Tools
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| DApp Browser | Web3 browser | YES NEW | YES | NO | YES NEW | YES NEW | YES |
| Launchpad | Token launches | NO | YES | NO | NO | NO | NO |
| Prediction Markets | Betting | NO | YES | NO | NO | NO | NO |
| RWA Trading | Real-world assets | NO | YES | NO | NO | NO | NO |
| Insurance Fund | Protection | NO | YES | NO | NO | NO | NO |
| Security Scanner | Contract audit | NO | YES | NO | NO | NO | NO |
| Gas Tracker | Fee estimation | NO | YES | NO | NO | NO | NO |
| Orderbook | Limit orders | NO | YES | NO | NO | NO | NO |
| TWAP | Time-weighted avg | NO | YES | NO | NO | NO | NO |
| Intent Routing | Intent-based | NO | YES | NO | NO | NO | NO |

### 7. Social & Rewards
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| Red Packet | Crypto | YES | YES | YES | YES | YES | YES |
| Claim | Airdrop claiming | YES | YES | YES | YES | YES | YES |

---

## Missing Features - Detailed Gaps

### HIGH PRIORITY

#### 1. Options Trading - Desktop NO
- Need C++ implementation for options pricing
- Location: `desktop_wallet/src/services/`
- Missing: options_pricing.cpp, options_service.cpp

#### 2. Liquid Staking - Mobile WARN
- Flutter: Partial implementation
- Android/iOS: Need native services
- Missing: liquid_staking_service.dart (Flutter), native Java/Swift

#### 3. DApp Browser - Desktop NO
- Need C++ implementation
- Location: `desktop_wallet/src/services/dapp_browser.cpp`

#### 4. Bridge - Desktop NO
- Need C++ cross-chain implementation
- Location: `desktop_wallet/src/services/bridge.cpp`

### MEDIUM PRIORITY

#### 5. DAO - Mobile NO
- Flutter: Just service created
- Android/iOS: Need native implementation
- Missing: full UI screens

#### 6. Launchpad - All Mobile NO
- All platforms: Not implemented
- Need complete mobile implementation

#### 7. Prediction Markets - All Mobile NO
- All platforms: Not implemented
- Need complete mobile implementation

#### 8. RWA Trading - All Mobile NO
- All platforms: Not implemented
- Need complete mobile implementation

### LOW PRIORITY

#### 9. Farming - Browser Extension NO
- Need JavaScript implementation

#### 10. NFT Features - Browser Extension NO
- View, Trade, Mint not implemented

#### 11. Gas Tracker - Mobile NO
- Need mobile implementation

#### 12. Orderbook - Mobile NO
- Need mobile implementation

#### 13. TWAP - Mobile NO
- Need mobile implementation

#### 14. Intent Routing - Mobile NO
- Need mobile implementation

#### 15. Security Scanner - Mobile NO
- Need mobile implementation

---

## Backend Services Required

### UserWallet Backend (Go) - Location: `/backend/go/`
```
YES P2P Service
YES Margin Service
YES Futures Service
YES Options Service
YES Wallet Service
YES Swap Service
YES NFT Service
YES Staking Service
YES Bridge Service WARN Need completion
YES Lending Service WARN Need completion
YES Gift Card Service WARN Need completion
YES Hardware Wallet Service WARN Need completion
YES MPC Wallet Service WARN Need completion
YES Social Recovery Service WARN Need completion
YES Account Abstraction Service WARN Need completion
YES DApp Browser Service WARN Need completion
```

---

## Platform Files Location

### Flutter (Cross-platform)
- Path: `mobile/flutter/lib/features/`
- Services: Each feature has *service.dart

### Web (NextJS)
- Path: `frontend/web_nextjs/app/`
- Each feature has dedicated folder

### Desktop (C++)
- Path: `desktop_wallet/src/services/`
- Need more implementations

### Android (Java)
- Path: `mobile/android/app/src/main/java/`
- Need more services

### iOS (Swift)
- Path: `mobile/ios/Runner/Services/`
- Need more services

### Browser Extensions
- Path: `browser_extensions/chrome/src/services/`
- Need more implementations

---

## What UserWallet CANNOT Access

### By Design - Isolation
- NO Admin functions (user management, KYC, fees)
- NO Master wallet operations (treasury, batch transactions)
- NO Platform configuration
- NO System settings

### Separate APIs
- Admin API: `https://admin-api.tigerwallet.com` NO
- MasterWallet API: `https://master-api.tigerwallet.com` NO

---

## Completion Status

| Platform | Features | Complete | Missing |
|---------|----------|----------|---------|
| Flutter | 35+ | 30 | 5 |
| Web | 40+ | 40 | 0 |
| Desktop | 15+ | 10 | 5 |
| Android | 30+ | 20 | 10 |
| iOS | 30+ | 20 | 10 |
| Browser Ext | 20+ | 15 | 5 |

---

## Next Steps to Complete

1. Implement Options Desktop (C++)
2. Complete Liquid Staking Mobile
3. Add DApp Browser Desktop
4. Add Bridge Desktop
5. Implement Launchpad Mobile
6. Implement DAO Mobile
7. Implement Prediction Markets Mobile
8. Implement RWA Trading Mobile

All backend services are connected to real PostgreSQL database - no mock data.
