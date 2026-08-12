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
| Flutter | `/mobile/flutter/lib/features/` | Dart | ACTIVE |
| Android | `/mobile/android/app/src/main/java/` | Java | ACTIVE |
| iOS | `/mobile/ios/Runner/Services/` | Swift | ACTIVE |
| Web | `/frontend/web_nextjs/app/` | Next.js/React | ACTIVE |
| Desktop | `/desktop_wallet/src/services/` | C++ | ACTIVE |
| Browser Extension | `/browser_extensions/chrome/src/services/` | JavaScript | ACTIVE |

---

## Feature Comparison Matrix

### 1. TRADING FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| P2P Trading | YES | YES | YES | YES | YES | YES |
| P2P Merchant | YES | YES | YES | YES | PARTIAL | YES |
| Margin Trading | YES | YES | YES | YES | YES | YES |
| Futures Trading | YES | YES | YES | YES | YES | YES |
| Options Trading | YES | YES | YES | YES | YES | NO |
| Copy Trading | YES | YES | YES | YES | YES | YES |
| Convert | YES | YES | YES | YES | YES | NO |
| Swap/DEX | YES | YES | YES | YES | YES | YES |

### 2. WALLET FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Multi-chain Wallet | YES | YES | YES | YES | YES | YES |
| Send/Receive | YES | YES | YES | YES | YES | YES |
| Address Book | YES | YES | YES | YES | YES | YES |
| QR Code | YES | YES | YES | YES | YES | YES |
| Hardware Wallet | YES NEW | YES NEW | YES NEW | YES | NO | YES |
| MPC Wallet | YES NEW | YES NEW | YES NEW | YES | NO | NO |
| Social Recovery | YES NEW | YES NEW | YES NEW | YES | NO | NO |
| Account Abstraction | YES NEW | YES NEW | YES NEW | YES | NO | NO |

### 3. DeFi FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Staking | YES | YES | YES | YES | YES | YES |
| Liquid Staking | PARTIAL | PARTIAL | PARTIAL | YES | YES | NO |
| Lending | YES NEW | YES | YES | YES | NO | YES |
| Bridge | YES NEW | YES | YES | YES | YES | YES |
| Farming | YES NEW | NO | NO | YES | YES | NO |
| DAO | YES NEW | YES | YES | YES | YES | YES |

### 4. NFT FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| NFT Gallery | YES | YES | YES | YES | YES | NO |
| NFT Trading | YES | YES | YES | YES | YES | NO |
| NFT Mint | YES | YES | YES | YES | NO | NO |

### 5. PAYMENTS

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Crypto Card | YES | YES | YES | YES | YES | YES |
| Fiat On-Ramp | YES | YES | YES | YES | YES | YES |
| Fiat Off-Ramp | YES | YES | YES | YES | YES | YES |
| Gift Cards | YES NEW | YES | YES | YES | YES | YES NEW |

### 6. DApp & TOOLS

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| DApp Browser | YES NEW | YES | YES | YES | NO | YES |
| Launchpad | YES | YES | YES | YES | YES | NO |
| Prediction Markets | NO | NO | NO | YES | YES | NO |
| RWA Trading | NO | YES | NO | YES | YES | NO |
| Security Scanner | NO | NO | NO | YES | NO | NO |
| Gas Tracker | NO | NO | NO | YES | NO | YES |
| Orderbook | NO | NO | NO | YES | NO | NO |
| TWAP | NO | NO | NO | YES | NO | NO |
| Intent Routing | NO | NO | NO | YES | NO | NO |

### 7. SOCIAL & REWARDS

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Red Packet | YES | YES | YES | YES | YES | YES |
| Airdrop Claim | YES | YES | YES | YES | YES | YES |

### 8. SECURITY FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Biometric Auth | YES | YES | YES | YES | YES | YES |
| Passkey | NO | NO | NO | YES | NO | NO |
| MEV Protection | NO | NO | NO | YES | YES | YES |
| Transaction Simulation | NO | NO | NO | YES | NO | NO |

---

## Backend Fetchers (Rust) - `/rust/userwallet_fetchers/`

### Implemented Fetchers

| Fetcher | Function | Status | Implementation |
|---------|----------|--------|----------------|
| `balance` | Fetch token/native balance | STUB | Returns mock data |
| `transactions` | Fetch transaction history | STUB | Returns empty array |
| `tokens` | Fetch token list | STUB | Returns empty array |
| `nfts` | Fetch NFT collections | STUB | Returns empty array |
| `swap` | Fetch DEX swap quotes | STUB | Returns null |
| `staking` | Fetch staking positions | STUB | Returns empty array |
| `gas` | Fetch gas prices | STUB | Returns mock values |
| `price` | Fetch price feeds | STUB | Returns empty object |

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
| `POST /register` | RegisterHandler | User registration | YES |
| `POST /login` | LoginHandler | JWT authentication | YES |
| `POST /wallets` | CreateWalletHandler | Create wallet | YES |
| `GET /wallets` | GetWalletsHandler | List wallets | YES |
| `GET /wallet/{id}` | GetWalletHandler | Wallet details | YES |
| `POST /transactions` | CreateTransactionHandler | Create tx | YES |
| `GET /transactions` | GetTransactionsHandler | Tx history | YES |
| `POST /kyc` | SubmitKYCHandler | KYC submission | YES |
| `GET /kyc/status` | GetKYCStatusHandler | KYC status | YES |
| `POST /auth/refresh` | RefreshTokenHandler | Token refresh | YES |

---

## Platform-Specific Services

### Flutter (`/mobile/flutter/lib/features/`)

| Feature | Service File | Status | Lines |
|---------|-------------|--------|-------|
| Account Abstraction | `account_abstraction/account_abstraction_service.dart` | YES | ~300 |
| Bridge | `bridge/bridge_service.dart` | YES | ~250 |
| Claim | `claim/claim_service.dart` | YES | ~200 |
| Convert | `convert/convert_service.dart` | YES | ~280 |
| Copy Trading | `copy_trading/copy_trading_service.dart` | YES | ~350 |
| Crypto Card | `crypto_card/crypto_card_service.dart` | YES | ~320 |
| DAO | `dao/dao_service.dart` | YES | ~290 |
| DApp Browser | `dapp_browser/dapp_browser_service.dart` | YES | ~400 |
| Fiat Ramp | `fiat_ramp/fiat_ramp_service.dart` | YES | ~350 |
| Futures | `futures/futures_service.dart` | YES | ~380 |
| Gift Cards | `gift_cards/gift_card_service.dart` | YES | ~310 |
| Hardware Wallet | `hardware_wallet/hardware_wallet_service.dart` | YES NEW | ~280 |
| Launchpad | `launchpad/launchpad_service.dart` | YES | ~260 |
| Lending | `lending/lending_service.dart` | YES | ~340 |
| Liquid Staking | `liquid_staking/liquid_staking_service.dart` | WARN PARTIAL | ~180 |
| Margin Trading | `margin_trading/margin_trading_service.dart` | YES | ~360 |
| MPC Wallet | `mpc_wallet/mpc_wallet_service.dart` | YES NEW | ~290 |
| NFT | `nft/services/nft_service.dart` | YES | ~280 |
| Options | `options/options_service.dart` | YES | ~320 |
| P2P Trading | `p2p_trading/p2p_service.dart` | YES | ~420 |
| Red Packet | `red_packet/red_packet_service.dart` | YES | ~240 |
| Social Recovery | `social_recovery/social_recovery_service.dart` | YES NEW | ~270 |
| Staking | `staking/services/staking_service.dart` | YES | ~310 |
| Swap | `swap/services/swap_service.dart` | YES | ~340 |
| Wallet | `wallet/services/wallet_service.dart` | YES | ~850 |

### Android (`/mobile/android/app/src/main/java/com/tigerwallet/app/`)

| Feature | Service File | Status |
|---------|-------------|--------|
| P2P Trading | `trading/P2PService.java` | YES |
| Crypto Card | `trading/CryptoCardService.java` | YES |
| Convert | `trading/ConvertService.java` | YES |
| Margin Trading | `trading/MarginTradingService.java` | YES |
| Fiat Ramp | `trading/FiatRampService.java` | YES |
| Futures | `trading/FuturesService.java` | YES |
| Copy Trading | `trading/CopyTradingService.java` | YES |
| Options Trading | `services/options/OptionsTradingService.java` | YES |
| RWA Trading | `services/rwa/RWAService.java` | YES |
| Launchpad | `services/launchpad/LaunchpadService.java` | YES |
| Lending | `services/LendingService.java` | YES |
| NFT | `services/nft/NFTService.java` | YES |
| DApp Browser | `services/dapp/DAppBrowserService.java` | YES |
| Bridge | `services/BridgeService.java` | YES |
| DAO | `services/dao/DAOService.java` | YES |
| Gift Cards | `services/gift/GiftCardService.java` | YES |
| Swap | `services/swap/SwapService.java` | YES |
| Staking | `services/staking/StakingService.java` | YES |

### iOS (`/mobile/ios/Runner/Services/`)

| Feature | Service File | Status |
|---------|-------------|--------|
| Gift Card | `GiftCardService.swift` | YES |
| Options Trading | `options/OptionsTradingService.swift` | YES |
| Fiat Ramp | `fiat/FiatRampService.swift` | YES |
| DAO | `dao/DAOService.swift` | YES |
| Bridge | `BridgeService.swift` | YES |
| Lending | `LendingService.swift` | YES |
| Swap | `swap/SwapService.swift` | YES |

### Desktop C++ (`/desktop_wallet/src/services/`)

| Feature | Service File | Status | Size |
|---------|-------------|--------|------|
| API Client | `api_client.cpp` | YES | 6.7 KB |
| Blockchain | `blockchain_service.cpp` | YES | 23.5 KB |
| Convert | `convert_service.cpp` | YES | 3.3 KB |
| Copy Trading | `copy_trading_service.cpp` | YES | 5.2 KB |
| Crypto Card | `crypto_card_service.cpp` | YES | 7.0 KB |
| Fiat Ramp | `fiat_ramp_service.cpp` | YES | 7.2 KB |
| Futures | `futures_service.cpp` | YES | 4.6 KB |
| Keychain | `keychain_manager.cpp` | YES | 9.9 KB |
| Margin Trading | `margin_trading_service.cpp` | YES | 5.6 KB |
| NFT | `nft_service.cpp` | YES | 7.2 KB |
| Price | `price_service.cpp` | YES | 15.2 KB |
| Staking | `staking_service.cpp` | YES | 9.9 KB |
| Swap | `swap_service.cpp` | YES | 6.9 KB |
| Biometric | `biometric/` | YES | 10.3 KB |
| Bridge | `bridge/` | YES | 17.4 KB |
| DAO | `dao/` | YES | 2.9 KB |
| DApp Browser | `dapp_browser/` | YES | 12.0 KB |
| Farming | `farming/` | YES | 1.5 KB |
| Gas Tracker | `gas_tracker/` | NO STUB | 0.3 KB |
| Gift Card | `gift_card/` | YES | 3.9 KB |
| Launchpad | `launchpad/` | YES | 1.9 KB |
| Liquid Staking | `liquid_staking/` | YES | 1.8 KB |
| MEV Protection | `mev_protection/` | YES | 5.4 KB |
| Options | `options/` | YES | 20.7 KB |
| Prediction Markets | `prediction_markets/` | YES | 1.8 KB |
| Realtime Alerts | `realtime_alerts/` | YES | 11.0 KB |
| RWA Trading | `rwa_trading/` | YES | 0.7 KB |

### Browser Extension (`/browser_extensions/chrome/src/services/`)

| Feature | Service File | Status | Size |
|---------|-------------|--------|------|
| Additional Services | `additional-services.js` | YES | 7.8 KB |
| Biometric Auth | `biometric-auth.js` | YES | 7.0 KB |
| Lending Service | `lending-service.js` | YES | 11.0 KB |
| NFT-DAO Service | `nft-dao-service.js` | YES | 9.1 KB |
| Price Service | `price-service.js` | YES | 3.5 KB |
| Swap-NFT-Staking-Bridge | `swap-nft-staking-bridge.js` | YES | 9.0 KB |
| Trading-MEV-Session-Gas | `trading-mev-session-gas-widget.js` | YES | 9.5 KB |
| DApp Browser | `dapp_browser/` | YES | 10.8 KB |
| Hardware Wallet | `hardware_wallet/` | YES | 17.1 KB |
| Multisig | `multisig/` | YES | 9.3 KB |
| Notifications | `notifications/` | YES | 12.7 KB |

---

## Gaps & Missing Features Summary

### HIGH PRIORITY GAPS

| Platform | Feature | Location | Status | Action Required |
|----------|---------|----------|--------|-----------------|
| Desktop | Gas Tracker | `desktop_wallet/src/services/gas_tracker/` | NO STUB | Implement C++ gas estimation |
| Extension | Options Trading | N/A | NO MISSING | Add options trading service |
| Extension | Convert | N/A | NO MISSING | Add convert service |
| Extension | NFT Gallery/Trading | N/A | NO MISSING | Add NFT services |
| Extension | Launchpad | N/A | NO MISSING | Add launchpad service |
| Extension | Lending | N/A | NO MISSING | Already in main services? |
| Extension | MPC Wallet | N/A | NO MISSING | Add MPC wallet service |
| Extension | Account Abstraction | N/A | NO MISSING | Add AA service |

### MEDIUM PRIORITY GAPS

| Platform | Feature | Status |
|----------|---------|--------|
| Flutter | Liquid Staking | PARTIAL - needs completion |
| Web | Biometric Auth | YES Implemented but needs mobile |
| All Mobile | Security Scanner | NO Not implemented |
| All Mobile | Orderbook | NO Not implemented |
| All Mobile | TWAP | NO Not implemented |
| All Mobile | Intent Routing | NO Not implemented |
| All Mobile | Prediction Markets | NO Not implemented |
| All Mobile | Passkey | NO Not implemented |

### BACKEND GAPS

| Service | Location | Status | Notes |
|---------|----------|--------|-------|
| Account Abstraction | `/backend/go/internal/services/` | YES Implemented | |
| Bridge | `/backend/go/internal/services/` | PARTIAL | Needs completion |
| Gift Card | `/backend/go/internal/services/` | PARTIAL | Needs completion |
| Hardware Wallet | `/backend/go/internal/services/` | PARTIAL | Needs completion |
| Lending | `/backend/go/internal/services/` | PARTIAL | Needs completion |
| MPC Wallet | `/backend/go/internal/services/` | PARTIAL | Needs completion |
| Social Recovery | `/backend/go/internal/services/` | PARTIAL | Needs completion |

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

YES **CONFIRMED**: UserWallet apps are completely isolated from:
- NO Admin APIs (`https://admin-api.tigerwallet.com`)
- NO MasterWallet APIs (`https://master-api.tigerwallet.com`)
- NO Admin fetchers and functionality
- NO MasterWallet fetchers and functionality

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
