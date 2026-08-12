# TigerWallet UserWallet Apps - Detailed Comparison Analysis

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

> **STATUS UPDATE (2026-08-11):** Since this file was generated (2026-08-05),
> the user-wallet client stack has been repaired to match the parity matrix
> shown in section 1. Verified current state: all clients target canonical
> `go/wallet_api` (:8443); 0 `Math.random()` fake-crypto calls remain;
> `rust/userwallet_fetchers` builds clean (delegates to wallet_api, no stubs);
> `frontend/web_nextjs` wallet transactions.ts EVM path fully wired + dynamic
> `/api/v1/transactions/[txHash]` route; light/dark theme on every web_nextjs
> page (0 `dark:` variants) with mobile theme managers;
> `permission_service`/`connection_api`/`monitoring_dashboard` build+vet clean
> (bcrypt for passwords; PostgreSQL+Redis).

## Overview
This document provides a comprehensive comparison of all UserWallet apps across platforms, identifying implemented features, fetchers, and gaps.

---

## Platform Overview

| Platform | Location | Technology | Status |
|----------|----------|------------|--------|
| Flutter | `/mobile/flutter/lib/features/` | Dart | ‚úÖ Active |
| Android | `/mobile/android/app/src/main/java/` | Java | ‚úÖ Active |
| iOS | `/mobile/ios/Runner/Services/` | Swift | ‚úÖ Active |
| Web | `/frontend/web_nextjs/app/` | Next.js/React | ‚úÖ Active |
| Desktop | `/desktop_wallet/src/services/` | C++ | ‚úÖ Active |
| Browser Extension | `/browser_extensions/chrome/src/services/` | JavaScript | ‚úÖ Active |

---

## Feature Comparison Matrix

### 1. TRADING FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| P2P Trading | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| P2P Merchant | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚ö†ÔłŹ Partial | ‚úÖ |
| Margin Trading | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Futures Trading | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Options Trading | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚ĚĆ |
| Copy Trading | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Convert | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚ĚĆ |
| Swap/DEX | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |

### 2. WALLET FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Multi-chain Wallet | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Send/Receive | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Address Book | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| QR Code | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Hardware Wallet | ‚úÖ NEW | ‚úÖ NEW | ‚úÖ NEW | ‚úÖ | ‚ĚĆ | ‚úÖ |
| MPC Wallet | ‚úÖ NEW | ‚úÖ NEW | ‚úÖ NEW | ‚úÖ | ‚ĚĆ | ‚ĚĆ |
| Social Recovery | ‚úÖ NEW | ‚úÖ NEW | ‚úÖ NEW | ‚úÖ | ‚ĚĆ | ‚ĚĆ |
| Account Abstraction | ‚úÖ NEW | ‚úÖ NEW | ‚úÖ NEW | ‚úÖ | ‚ĚĆ | ‚ĚĆ |

### 3. DeFi FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Staking | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Liquid Staking | ‚ö†ÔłŹ Partial | ‚ö†ÔłŹ Partial | ‚ö†ÔłŹ Partial | ‚úÖ | ‚úÖ | ‚ĚĆ |
| Lending | ‚úÖ NEW | ‚úÖ | ‚úÖ | ‚úÖ | ‚ĚĆ | ‚úÖ |
| Bridge | ‚úÖ NEW | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Farming | ‚úÖ NEW | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚úÖ | ‚ĚĆ |
| DAO | ‚úÖ NEW | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |

### 4. NFT FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| NFT Gallery | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚ĚĆ |
| NFT Trading | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚ĚĆ |
| NFT Mint | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚ĚĆ | ‚ĚĆ |

### 5. PAYMENTS

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Crypto Card | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Fiat On-Ramp | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Fiat Off-Ramp | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Gift Cards | ‚úÖ NEW | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ NEW |

### 6. DApp & TOOLS

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| DApp Browser | ‚úÖ NEW | ‚úÖ | ‚úÖ | ‚úÖ | ‚ĚĆ | ‚úÖ |
| Launchpad | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚ĚĆ |
| Prediction Markets | ‚ĚĆ | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚úÖ | ‚ĚĆ |
| RWA Trading | ‚ĚĆ | ‚úÖ | ‚ĚĆ | ‚úÖ | ‚úÖ | ‚ĚĆ |
| Security Scanner | ‚ĚĆ | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚ĚĆ | ‚ĚĆ |
| Gas Tracker | ‚ĚĆ | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚ĚĆ | ‚úÖ |
| Orderbook | ‚ĚĆ | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚ĚĆ | ‚ĚĆ |
| TWAP | ‚ĚĆ | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚ĚĆ | ‚ĚĆ |
| Intent Routing | ‚ĚĆ | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚ĚĆ | ‚ĚĆ |

### 7. SOCIAL & REWARDS

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Red Packet | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Airdrop Claim | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |

### 8. SECURITY FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Biometric Auth | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ | ‚úÖ |
| Passkey | ‚ĚĆ | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚ĚĆ | ‚ĚĆ |
| MEV Protection | ‚ĚĆ | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚úÖ | ‚úÖ |
| Transaction Simulation | ‚ĚĆ | ‚ĚĆ | ‚ĚĆ | ‚úÖ | ‚ĚĆ | ‚ĚĆ |

---

## Backend Fetchers (Rust) - `/rust/userwallet_fetchers/`

### Implemented Fetchers

| Fetcher | Function | Status | Implementation |
|---------|----------|--------|----------------|
| `balance` | Fetch token/native balance | ‚ö†ÔłŹ STUB | Returns mock data |
| `transactions` | Fetch transaction history | ‚ö†ÔłŹ STUB | Returns empty array |
| `tokens` | Fetch token list | ‚ö†ÔłŹ STUB | Returns empty array |
| `nfts` | Fetch NFT collections | ‚ö†ÔłŹ STUB | Returns empty array |
| `swap` | Fetch DEX swap quotes | ‚ö†ÔłŹ STUB | Returns null |
| `staking` | Fetch staking positions | ‚ö†ÔłŹ STUB | Returns empty array |
| `gas` | Fetch gas prices | ‚ö†ÔłŹ STUB | Returns mock values |
| `price` | Fetch price feeds | ‚ö†ÔłŹ STUB | Returns empty object |

### Missing Fetchers (NOT IMPLEMENTED)

| Fetcher | Reason |
|---------|--------|
| Bridge | Not implemented |
| Lending | Not implemented |
| NFT Trading | Not implemented |
| Options Pricing | Not implemented |
| Futures Data | Not implemented |
| Margin Trading Data | Not implemented |
| P2P Orders | Not implemented |
| Copy Trading Data | Not implemented |
| DAO Governance | Not implemented |
| Gift Card Balance | Not implemented |
| Fiat Ramp Quotes | Not implemented |
| DApp Registry | Not implemented |
| Price Alerts | Not implemented |

---

## Backend Services (Go) - `/user_services/go/main.go`

### Implemented API Endpoints

| Endpoint | Handler | Function | Status |
|----------|---------|----------|--------|
| `POST /register` | RegisterHandler | User registration | ‚úÖ |
| `POST /login` | LoginHandler | JWT authentication | ‚úÖ |
| `POST /wallets` | CreateWalletHandler | Create wallet | ‚úÖ |
| `GET /wallets` | GetWalletsHandler | List wallets | ‚úÖ |
| `GET /wallet/{id}` | GetWalletHandler | Wallet details | ‚úÖ |
| `POST /transactions` | CreateTransactionHandler | Create tx | ‚úÖ |
| `GET /transactions` | GetTransactionsHandler | Tx history | ‚úÖ |
| `POST /kyc` | SubmitKYCHandler | KYC submission | ‚úÖ |
| `GET /kyc/status` | GetKYCStatusHandler | KYC status | ‚úÖ |
| `POST /auth/refresh` | RefreshTokenHandler | Token refresh | ‚úÖ |

---

## Platform-Specific Services

### Flutter (`/mobile/flutter/lib/features/`)

| Feature | Service File | Status | Lines |
|---------|-------------|--------|-------|
| Account Abstraction | `account_abstraction/account_abstraction_service.dart` | ‚úÖ | ~300 |
| Bridge | `bridge/bridge_service.dart` | ‚úÖ | ~250 |
| Claim | `claim/claim_service.dart` | ‚úÖ | ~200 |
| Convert | `convert/convert_service.dart` | ‚úÖ | ~280 |
| Copy Trading | `copy_trading/copy_trading_service.dart` | ‚úÖ | ~350 |
| Crypto Card | `crypto_card/crypto_card_service.dart` | ‚úÖ | ~320 |
| DAO | `dao/dao_service.dart` | ‚úÖ | ~290 |
| DApp Browser | `dapp_browser/dapp_browser_service.dart` | ‚úÖ | ~400 |
| Fiat Ramp | `fiat_ramp/fiat_ramp_service.dart` | ‚úÖ | ~350 |
| Futures | `futures/futures_service.dart` | ‚úÖ | ~380 |
| Gift Cards | `gift_cards/gift_card_service.dart` | ‚úÖ | ~310 |
| Hardware Wallet | `hardware_wallet/hardware_wallet_service.dart` | ‚úÖ NEW | ~280 |
| Launchpad | `launchpad/launchpad_service.dart` | ‚úÖ | ~260 |
| Lending | `lending/lending_service.dart` | ‚úÖ | ~340 |
| Liquid Staking | `liquid_staking/liquid_staking_service.dart` | ‚ö†ÔłŹ PARTIAL | ~180 |
| Margin Trading | `margin_trading/margin_trading_service.dart` | ‚úÖ | ~360 |
| MPC Wallet | `mpc_wallet/mpc_wallet_service.dart` | ‚úÖ NEW | ~290 |
| NFT | `nft/services/nft_service.dart` | ‚úÖ | ~280 |
| Options | `options/options_service.dart` | ‚úÖ | ~320 |
| P2P Trading | `p2p_trading/p2p_service.dart` | ‚úÖ | ~420 |
| Red Packet | `red_packet/red_packet_service.dart` | ‚úÖ | ~240 |
| Social Recovery | `social_recovery/social_recovery_service.dart` | ‚úÖ NEW | ~270 |
| Staking | `staking/services/staking_service.dart` | ‚úÖ | ~310 |
| Swap | `swap/services/swap_service.dart` | ‚úÖ | ~340 |
| Wallet | `wallet/services/wallet_service.dart` | ‚úÖ | ~850 |

### Android (`/mobile/android/app/src/main/java/com/tigerwallet/app/`)

| Feature | Service File | Status |
|---------|-------------|--------|
| P2P Trading | `trading/P2PService.java` | ‚úÖ |
| Crypto Card | `trading/CryptoCardService.java` | ‚úÖ |
| Convert | `trading/ConvertService.java` | ‚úÖ |
| Margin Trading | `trading/MarginTradingService.java` | ‚úÖ |
| Fiat Ramp | `trading/FiatRampService.java` | ‚úÖ |
| Futures | `trading/FuturesService.java` | ‚úÖ |
| Copy Trading | `trading/CopyTradingService.java` | ‚úÖ |
| Options Trading | `services/options/OptionsTradingService.java` | ‚úÖ |
| RWA Trading | `services/rwa/RWAService.java` | ‚úÖ |
| Launchpad | `services/launchpad/LaunchpadService.java` | ‚úÖ |
| Lending | `services/LendingService.java` | ‚úÖ |
| NFT | `services/nft/NFTService.java` | ‚úÖ |
| DApp Browser | `services/dapp/DAppBrowserService.java` | ‚úÖ |
| Bridge | `services/BridgeService.java` | ‚úÖ |
| DAO | `services/dao/DAOService.java` | ‚úÖ |
| Gift Cards | `services/gift/GiftCardService.java` | ‚úÖ |
| Swap | `services/swap/SwapService.java` | ‚úÖ |
| Staking | `services/staking/StakingService.java` | ‚úÖ |

### iOS (`/mobile/ios/Runner/Services/`)

| Feature | Service File | Status |
|---------|-------------|--------|
| Gift Card | `GiftCardService.swift` | ‚úÖ |
| Options Trading | `options/OptionsTradingService.swift` | ‚úÖ |
| Fiat Ramp | `fiat/FiatRampService.swift` | ‚úÖ |
| DAO | `dao/DAOService.swift` | ‚úÖ |
| Bridge | `BridgeService.swift` | ‚úÖ |
| Lending | `LendingService.swift` | ‚úÖ |
| Swap | `swap/SwapService.swift` | ‚úÖ |

### Desktop C++ (`/desktop_wallet/src/services/`)

| Feature | Service File | Status | Size |
|---------|-------------|--------|------|
| API Client | `api_client.cpp` | ‚úÖ | 6.7 KB |
| Blockchain | `blockchain_service.cpp` | ‚úÖ | 23.5 KB |
| Convert | `convert_service.cpp` | ‚úÖ | 3.3 KB |
| Copy Trading | `copy_trading_service.cpp` | ‚úÖ | 5.2 KB |
| Crypto Card | `crypto_card_service.cpp` | ‚úÖ | 7.0 KB |
| Fiat Ramp | `fiat_ramp_service.cpp` | ‚úÖ | 7.2 KB |
| Futures | `futures_service.cpp` | ‚úÖ | 4.6 KB |
| Keychain | `keychain_manager.cpp` | ‚úÖ | 9.9 KB |
| Margin Trading | `margin_trading_service.cpp` | ‚úÖ | 5.6 KB |
| NFT | `nft_service.cpp` | ‚úÖ | 7.2 KB |
| Price | `price_service.cpp` | ‚úÖ | 15.2 KB |
| Staking | `staking_service.cpp` | ‚úÖ | 9.9 KB |
| Swap | `swap_service.cpp` | ‚úÖ | 6.9 KB |
| Biometric | `biometric/` | ‚úÖ | 10.3 KB |
| Bridge | `bridge/` | ‚úÖ | 17.4 KB |
| DAO | `dao/` | ‚úÖ | 2.9 KB |
| DApp Browser | `dapp_browser/` | ‚úÖ | 12.0 KB |
| Farming | `farming/` | ‚úÖ | 1.5 KB |
| Gas Tracker | `gas_tracker/` | ‚ĚĆ STUB | 0.3 KB |
| Gift Card | `gift_card/` | ‚úÖ | 3.9 KB |
| Launchpad | `launchpad/` | ‚úÖ | 1.9 KB |
| Liquid Staking | `liquid_staking/` | ‚úÖ | 1.8 KB |
| MEV Protection | `mev_protection/` | ‚úÖ | 5.4 KB |
| Options | `options/` | ‚úÖ | 20.7 KB |
| Prediction Markets | `prediction_markets/` | ‚úÖ | 1.8 KB |
| Realtime Alerts | `realtime_alerts/` | ‚úÖ | 11.0 KB |
| RWA Trading | `rwa_trading/` | ‚úÖ | 0.7 KB |

### Browser Extension (`/browser_extensions/chrome/src/services/`)

| Feature | Service File | Status | Size |
|---------|-------------|--------|------|
| Additional Services | `additional-services.js` | ‚úÖ | 7.8 KB |
| Biometric Auth | `biometric-auth.js` | ‚úÖ | 7.0 KB |
| Lending Service | `lending-service.js` | ‚úÖ | 11.0 KB |
| NFT-DAO Service | `nft-dao-service.js` | ‚úÖ | 9.1 KB |
| Price Service | `price-service.js` | ‚úÖ | 3.5 KB |
| Swap-NFT-Staking-Bridge | `swap-nft-staking-bridge.js` | ‚úÖ | 9.0 KB |
| Trading-MEV-Session-Gas | `trading-mev-session-gas-widget.js` | ‚úÖ | 9.5 KB |
| DApp Browser | `dapp_browser/` | ‚úÖ | 10.8 KB |
| Hardware Wallet | `hardware_wallet/` | ‚úÖ | 17.1 KB |
| Multisig | `multisig/` | ‚úÖ | 9.3 KB |
| Notifications | `notifications/` | ‚úÖ | 12.7 KB |

---

## Gaps & Missing Features Summary

### HIGH PRIORITY GAPS

| Platform | Feature | Location | Status | Action Required |
|----------|---------|----------|--------|-----------------|
| Desktop | Gas Tracker | `desktop_wallet/src/services/gas_tracker/` | ‚ĚĆ STUB | Implement C++ gas estimation |
| Extension | Options Trading | N/A | ‚ĚĆ MISSING | Add options trading service |
| Extension | Convert | N/A | ‚ĚĆ MISSING | Add convert service |
| Extension | NFT Gallery/Trading | N/A | ‚ĚĆ MISSING | Add NFT services |
| Extension | Launchpad | N/A | ‚ĚĆ MISSING | Add launchpad service |
| Extension | Lending | N/A | ‚ĚĆ MISSING | Already in main services? |
| Extension | MPC Wallet | N/A | ‚ĚĆ MISSING | Add MPC wallet service |
| Extension | Account Abstraction | N/A | ‚ĚĆ MISSING | Add AA service |

### MEDIUM PRIORITY GAPS

| Platform | Feature | Status |
|----------|---------|--------|
| Flutter | Liquid Staking | ‚ö†ÔłŹ Partial - needs completion |
| Web | Biometric Auth | ‚úÖ Implemented but needs mobile |
| All Mobile | Security Scanner | ‚ĚĆ Not implemented |
| All Mobile | Orderbook | ‚ĚĆ Not implemented |
| All Mobile | TWAP | ‚ĚĆ Not implemented |
| All Mobile | Intent Routing | ‚ĚĆ Not implemented |
| All Mobile | Prediction Markets | ‚ĚĆ Not implemented |
| All Mobile | Passkey | ‚ĚĆ Not implemented |

### BACKEND GAPS

| Service | Location | Status | Notes |
|---------|----------|--------|-------|
| Account Abstraction | `/backend/go/internal/services/` | ‚úÖ Implemented | |
| Bridge | `/backend/go/internal/services/` | ‚ö†ÔłŹ Partial | Needs completion |
| Gift Card | `/backend/go/internal/services/` | ‚ö†ÔłŹ Partial | Needs completion |
| Hardware Wallet | `/backend/go/internal/services/` | ‚ö†ÔłŹ Partial | Needs completion |
| Lending | `/backend/go/internal/services/` | ‚ö†ÔłŹ Partial | Needs completion |
| MPC Wallet | `/backend/go/internal/services/` | ‚ö†ÔłŹ Partial | Needs completion |
| Social Recovery | `/backend/go/internal/services/` | ‚ö†ÔłŹ Partial | Needs completion |

### RUST FETCHER GAPS

All fetchers in `/rust/userwallet_fetchers/` return mock/stub data:
- No real blockchain RPC connections
- No actual price feed integrations
- No real swap quote integrations
- No staking data integrations

---

## Completion Status Summary

| Platform | Total Features | Implemented | Missing | Completion % |
|----------|---------------|--------------|---------|--------------|
| Flutter | 27 | 26 | 1 | 96% |
| Web (NextJS) | 50+ | 50+ | 0 | 100% |
| Desktop | 28 | 27 | 1 | 96% |
| Android | 20+ | 20+ | 0 | 100% |
| iOS | 15+ | 15+ | 0 | 100% |
| Browser Extension | 18 | 11 | 7 | 61% |

---

## Separation Verification

‚úÖ **CONFIRMED**: UserWallet apps are completely isolated from:
- ‚ĚĆ Admin APIs (`https://admin-api.tigerwallet.com`)
- ‚ĚĆ MasterWallet APIs (`https://master-api.tigerwallet.com`)
- ‚ĚĆ Admin fetchers and functionality
- ‚ĚĆ MasterWallet fetchers and functionality

All UserWallet components are in:
- `/rust/userwallet_fetchers/` - UserWallet only
- `/user_services/go/` - UserWallet only
- `/mobile/flutter/lib/features/` - UserWallet only
- `/mobile/android/app/src/main/java/com/tigerwallet/app/` - UserWallet only
- `/mobile/ios/Runner/Services/` - UserWallet only
- `/desktop_wallet/src/services/` - UserWallet only
- `/browser_extensions/chrome/src/services/` - UserWallet only
- `/frontend/web_nextjs/app/` - Mix of UserWallet (no admin_fees, admin_listing, master_wallet, super_admin, etc.)

---

*Generated: 2026-08-05*
