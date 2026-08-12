# UserWallet - Complete Feature Analysis

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
| P2P Trading | Peer-to-peer crypto trading | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| P2P Merchant | Become a merchant | вњ… | вњ… | вљ пёЏ | вњ… | вњ… | вњ… |
| Margin Trading | Leverage trading | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Futures Trading | Perpetual contracts | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Options Trading | Call/Put options | вњ… | вњ… | вќЊ | вњ… | вњ… | вњ… |
| Copy Trading | Follow traders | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Convert | Instant conversion | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Swap/DEX | Decentralized exchange | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |

### 2. Wallet Features
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| Multi-chain Wallet | Support 10+ chains | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Send/Receive | Transfer crypto | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Address Book | Saved addresses | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| QR Code | Scan/pay QR | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Hardware Wallet | Ledger/Trezor | вњ… NEW | вњ… | вќЊ | вњ… NEW | вњ… NEW | вњ… |
| MPC Wallet | Multi-party computation | вњ… NEW | вњ… | вќЊ | вњ… NEW | вњ… NEW | вњ… |
| Social Recovery | Guardian-based recovery | вњ… NEW | вњ… | вќЊ | вњ… NEW | вњ… NEW | вњ… |
| Account Abstraction | Smart accounts | вњ… NEW | вњ… | вќЊ | вњ… NEW | вњ… NEW | вњ… |

### 3. DeFi Features
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| Staking | Proof-of-stake | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Liquid Staking | Staking tokens | вљ пёЏ | вњ… | вќЊ | вљ пёЏ | вљ пёЏ | вќЊ |
| Lending | Supply/Borrow | вњ… NEW | вњ… | вќЊ | вњ… NEW | вњ… NEW | вњ… NEW |
| Bridge | Cross-chain | вњ… NEW | вњ… | вќЊ | вњ… NEW | вњ… NEW | вњ… NEW |
| Farming | Yield farming | вњ… NEW | вњ… | вќЊ | вњ… | вњ… | вќЊ |
| DAO | Governance | вњ… NEW | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |

### 4. NFT Features
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| NFT Gallery | View collections | вњ… | вњ… | вњ… | вњ… | вњ… | вќЊ |
| NFT Trading | Buy/Sell NFTs | вњ… | вњ… | вњ… | вњ… | вњ… | вќЊ |
| NFT Mint | Create NFTs | вњ… | вњ… | вќЊ | вњ… | вњ… | вќЊ |

### 5. Payments
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| Crypto Card | Virtual card | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Fiat On-Ramp | Buy crypto | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Fiat Off-Ramp | Sell crypto | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Gift Cards | Buy/Sell gift cards | вњ… NEW | вњ… | вќЊ | вњ… NEW | вњ… NEW | вњ… NEW |

### 6. DApp & Tools
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| DApp Browser | Web3 browser | вњ… NEW | вњ… | вќЊ | вњ… NEW | вњ… NEW | вњ… |
| Launchpad | Token launches | вќЊ | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |
| Prediction Markets | Betting | вќЊ | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |
| RWA Trading | Real-world assets | вќЊ | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |
| Insurance Fund | Protection | вќЊ | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |
| Security Scanner | Contract audit | вќЊ | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |
| Gas Tracker | Fee estimation | вќЊ | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |
| Orderbook | Limit orders | вќЊ | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |
| TWAP | Time-weighted avg | вќЊ | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |
| Intent Routing | Intent-based | вќЊ | вњ… | вќЊ | вќЊ | вќЊ | вќЊ |

### 7. Social & Rewards
| Feature | Description | Flutter | Web | Desktop | Android | iOS | Browser Ext |
|---------|-------------|---------|-----|---------|---------|-----|-------------|
| Red Packet | CryptoзєўеЊ… | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |
| Claim | Airdrop claiming | вњ… | вњ… | вњ… | вњ… | вњ… | вњ… |

---

## Missing Features - Detailed Gaps

### HIGH PRIORITY

#### 1. Options Trading - Desktop вќЊ
- Need C++ implementation for options pricing
- Location: `desktop_wallet/src/services/`
- Missing: options_pricing.cpp, options_service.cpp

#### 2. Liquid Staking - Mobile вљ пёЏ
- Flutter: Partial implementation
- Android/iOS: Need native services
- Missing: liquid_staking_service.dart (Flutter), native Java/Swift

#### 3. DApp Browser - Desktop вќЊ
- Need C++ implementation
- Location: `desktop_wallet/src/services/dapp_browser.cpp`

#### 4. Bridge - Desktop вќЊ
- Need C++ cross-chain implementation
- Location: `desktop_wallet/src/services/bridge.cpp`

### MEDIUM PRIORITY

#### 5. DAO - Mobile вќЊ
- Flutter: Just service created
- Android/iOS: Need native implementation
- Missing: full UI screens

#### 6. Launchpad - All Mobile вќЊ
- All platforms: Not implemented
- Need complete mobile implementation

#### 7. Prediction Markets - All Mobile вќЊ
- All platforms: Not implemented
- Need complete mobile implementation

#### 8. RWA Trading - All Mobile вќЊ
- All platforms: Not implemented
- Need complete mobile implementation

### LOW PRIORITY

#### 9. Farming - Browser Extension вќЊ
- Need JavaScript implementation

#### 10. NFT Features - Browser Extension вќЊ
- View, Trade, Mint not implemented

#### 11. Gas Tracker - Mobile вќЊ
- Need mobile implementation

#### 12. Orderbook - Mobile вќЊ
- Need mobile implementation

#### 13. TWAP - Mobile вќЊ
- Need mobile implementation

#### 14. Intent Routing - Mobile вќЊ
- Need mobile implementation

#### 15. Security Scanner - Mobile вќЊ
- Need mobile implementation

---

## Backend Services Required

### UserWallet Backend (Go) - Location: `/backend/go/`
```
вњ… P2P Service
вњ… Margin Service
вњ… Futures Service
вњ… Options Service
вњ… Wallet Service
вњ… Swap Service
вњ… NFT Service
вњ… Staking Service
вњ… Bridge Service вљ пёЏ Need completion
вњ… Lending Service вљ пёЏ Need completion
вњ… Gift Card Service вљ пёЏ Need completion
вњ… Hardware Wallet Service вљ пёЏ Need completion
вњ… MPC Wallet Service вљ пёЏ Need completion
вњ… Social Recovery Service вљ пёЏ Need completion
вњ… Account Abstraction Service вљ пёЏ Need completion
вњ… DApp Browser Service вљ пёЏ Need completion
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
- вќЊ Admin functions (user management, KYC, fees)
- вќЊ Master wallet operations (treasury, batch transactions)
- вќЊ Platform configuration
- вќЊ System settings

### Separate APIs
- Admin API: `https://admin-api.tigerwallet.com` вќЊ
- MasterWallet API: `https://master-api.tigerwallet.com` вќЊ

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
