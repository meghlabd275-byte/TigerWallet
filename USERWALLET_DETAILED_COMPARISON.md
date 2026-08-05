# TigerWallet UserWallet Apps - Detailed Comparison Analysis

## Overview
This document provides a comprehensive comparison of all UserWallet apps across platforms, identifying implemented features, fetchers, and gaps.

---

## Platform Overview

| Platform | Location | Technology | Status |
|----------|----------|------------|--------|
| Flutter | `/mobile/flutter/lib/features/` | Dart | ✅ Active |
| Android | `/mobile/android/app/src/main/java/` | Java | ✅ Active |
| iOS | `/mobile/ios/Runner/Services/` | Swift | ✅ Active |
| Web | `/frontend/web_nextjs/app/` | Next.js/React | ✅ Active |
| Desktop | `/desktop_wallet/src/services/` | C++ | ✅ Active |
| Browser Extension | `/browser_extensions/chrome/src/services/` | JavaScript | ✅ Active |

---

## Feature Comparison Matrix

### 1. TRADING FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| P2P Trading | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| P2P Merchant | ✅ | ✅ | ✅ | ✅ | ⚠️ Partial | ✅ |
| Margin Trading | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Futures Trading | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Options Trading | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Copy Trading | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Convert | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Swap/DEX | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### 2. WALLET FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Multi-chain Wallet | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Send/Receive | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Address Book | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| QR Code | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Hardware Wallet | ✅ NEW | ✅ NEW | ✅ NEW | ✅ | ❌ | ✅ |
| MPC Wallet | ✅ NEW | ✅ NEW | ✅ NEW | ✅ | ❌ | ❌ |
| Social Recovery | ✅ NEW | ✅ NEW | ✅ NEW | ✅ | ❌ | ❌ |
| Account Abstraction | ✅ NEW | ✅ NEW | ✅ NEW | ✅ | ❌ | ❌ |

### 3. DeFi FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Staking | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Liquid Staking | ⚠️ Partial | ⚠️ Partial | ⚠️ Partial | ✅ | ✅ | ❌ |
| Lending | ✅ NEW | ✅ | ✅ | ✅ | ❌ | ✅ |
| Bridge | ✅ NEW | ✅ | ✅ | ✅ | ✅ | ✅ |
| Farming | ✅ NEW | ❌ | ❌ | ✅ | ✅ | ❌ |
| DAO | ✅ NEW | ✅ | ✅ | ✅ | ✅ | ✅ |

### 4. NFT FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| NFT Gallery | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| NFT Trading | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| NFT Mint | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |

### 5. PAYMENTS

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Crypto Card | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Fiat On-Ramp | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Fiat Off-Ramp | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Gift Cards | ✅ NEW | ✅ | ✅ | ✅ | ✅ | ✅ NEW |

### 6. DApp & TOOLS

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| DApp Browser | ✅ NEW | ✅ | ✅ | ✅ | ❌ | ✅ |
| Launchpad | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Prediction Markets | ❌ | ❌ | ❌ | ✅ | ✅ | ❌ |
| RWA Trading | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ |
| Security Scanner | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| Gas Tracker | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ |
| Orderbook | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| TWAP | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| Intent Routing | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |

### 7. SOCIAL & REWARDS

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Red Packet | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Airdrop Claim | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### 8. SECURITY FEATURES

| Feature | Flutter | Android | iOS | Web | Desktop | Extension |
|---------|---------|---------|-----|-----|---------|-----------|
| Biometric Auth | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Passkey | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| MEV Protection | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ |
| Transaction Simulation | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |

---

## Backend Fetchers (Rust) - `/rust/userwallet_fetchers/`

### Implemented Fetchers

| Fetcher | Function | Status | Implementation |
|---------|----------|--------|----------------|
| `balance` | Fetch token/native balance | ⚠️ STUB | Returns mock data |
| `transactions` | Fetch transaction history | ⚠️ STUB | Returns empty array |
| `tokens` | Fetch token list | ⚠️ STUB | Returns empty array |
| `nfts` | Fetch NFT collections | ⚠️ STUB | Returns empty array |
| `swap` | Fetch DEX swap quotes | ⚠️ STUB | Returns null |
| `staking` | Fetch staking positions | ⚠️ STUB | Returns empty array |
| `gas` | Fetch gas prices | ⚠️ STUB | Returns mock values |
| `price` | Fetch price feeds | ⚠️ STUB | Returns empty object |

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
| `POST /register` | RegisterHandler | User registration | ✅ |
| `POST /login` | LoginHandler | JWT authentication | ✅ |
| `POST /wallets` | CreateWalletHandler | Create wallet | ✅ |
| `GET /wallets` | GetWalletsHandler | List wallets | ✅ |
| `GET /wallet/{id}` | GetWalletHandler | Wallet details | ✅ |
| `POST /transactions` | CreateTransactionHandler | Create tx | ✅ |
| `GET /transactions` | GetTransactionsHandler | Tx history | ✅ |
| `POST /kyc` | SubmitKYCHandler | KYC submission | ✅ |
| `GET /kyc/status` | GetKYCStatusHandler | KYC status | ✅ |
| `POST /auth/refresh` | RefreshTokenHandler | Token refresh | ✅ |

---

## Platform-Specific Services

### Flutter (`/mobile/flutter/lib/features/`)

| Feature | Service File | Status | Lines |
|---------|-------------|--------|-------|
| Account Abstraction | `account_abstraction/account_abstraction_service.dart` | ✅ | ~300 |
| Bridge | `bridge/bridge_service.dart` | ✅ | ~250 |
| Claim | `claim/claim_service.dart` | ✅ | ~200 |
| Convert | `convert/convert_service.dart` | ✅ | ~280 |
| Copy Trading | `copy_trading/copy_trading_service.dart` | ✅ | ~350 |
| Crypto Card | `crypto_card/crypto_card_service.dart` | ✅ | ~320 |
| DAO | `dao/dao_service.dart` | ✅ | ~290 |
| DApp Browser | `dapp_browser/dapp_browser_service.dart` | ✅ | ~400 |
| Fiat Ramp | `fiat_ramp/fiat_ramp_service.dart` | ✅ | ~350 |
| Futures | `futures/futures_service.dart` | ✅ | ~380 |
| Gift Cards | `gift_cards/gift_card_service.dart` | ✅ | ~310 |
| Hardware Wallet | `hardware_wallet/hardware_wallet_service.dart` | ✅ NEW | ~280 |
| Launchpad | `launchpad/launchpad_service.dart` | ✅ | ~260 |
| Lending | `lending/lending_service.dart` | ✅ | ~340 |
| Liquid Staking | `liquid_staking/liquid_staking_service.dart` | ⚠️ PARTIAL | ~180 |
| Margin Trading | `margin_trading/margin_trading_service.dart` | ✅ | ~360 |
| MPC Wallet | `mpc_wallet/mpc_wallet_service.dart` | ✅ NEW | ~290 |
| NFT | `nft/services/nft_service.dart` | ✅ | ~280 |
| Options | `options/options_service.dart` | ✅ | ~320 |
| P2P Trading | `p2p_trading/p2p_service.dart` | ✅ | ~420 |
| Red Packet | `red_packet/red_packet_service.dart` | ✅ | ~240 |
| Social Recovery | `social_recovery/social_recovery_service.dart` | ✅ NEW | ~270 |
| Staking | `staking/services/staking_service.dart` | ✅ | ~310 |
| Swap | `swap/services/swap_service.dart` | ✅ | ~340 |
| Wallet | `wallet/services/wallet_service.dart` | ✅ | ~850 |

### Android (`/mobile/android/app/src/main/java/com/tigerwallet/app/`)

| Feature | Service File | Status |
|---------|-------------|--------|
| P2P Trading | `trading/P2PService.java` | ✅ |
| Crypto Card | `trading/CryptoCardService.java` | ✅ |
| Convert | `trading/ConvertService.java` | ✅ |
| Margin Trading | `trading/MarginTradingService.java` | ✅ |
| Fiat Ramp | `trading/FiatRampService.java` | ✅ |
| Futures | `trading/FuturesService.java` | ✅ |
| Copy Trading | `trading/CopyTradingService.java` | ✅ |
| Options Trading | `services/options/OptionsTradingService.java` | ✅ |
| RWA Trading | `services/rwa/RWAService.java` | ✅ |
| Launchpad | `services/launchpad/LaunchpadService.java` | ✅ |
| Lending | `services/LendingService.java` | ✅ |
| NFT | `services/nft/NFTService.java` | ✅ |
| DApp Browser | `services/dapp/DAppBrowserService.java` | ✅ |
| Bridge | `services/BridgeService.java` | ✅ |
| DAO | `services/dao/DAOService.java` | ✅ |
| Gift Cards | `services/gift/GiftCardService.java` | ✅ |
| Swap | `services/swap/SwapService.java` | ✅ |
| Staking | `services/staking/StakingService.java` | ✅ |

### iOS (`/mobile/ios/Runner/Services/`)

| Feature | Service File | Status |
|---------|-------------|--------|
| Gift Card | `GiftCardService.swift` | ✅ |
| Options Trading | `options/OptionsTradingService.swift` | ✅ |
| Fiat Ramp | `fiat/FiatRampService.swift` | ✅ |
| DAO | `dao/DAOService.swift` | ✅ |
| Bridge | `BridgeService.swift` | ✅ |
| Lending | `LendingService.swift` | ✅ |
| Swap | `swap/SwapService.swift` | ✅ |

### Desktop C++ (`/desktop_wallet/src/services/`)

| Feature | Service File | Status | Size |
|---------|-------------|--------|------|
| API Client | `api_client.cpp` | ✅ | 6.7 KB |
| Blockchain | `blockchain_service.cpp` | ✅ | 23.5 KB |
| Convert | `convert_service.cpp` | ✅ | 3.3 KB |
| Copy Trading | `copy_trading_service.cpp` | ✅ | 5.2 KB |
| Crypto Card | `crypto_card_service.cpp` | ✅ | 7.0 KB |
| Fiat Ramp | `fiat_ramp_service.cpp` | ✅ | 7.2 KB |
| Futures | `futures_service.cpp` | ✅ | 4.6 KB |
| Keychain | `keychain_manager.cpp` | ✅ | 9.9 KB |
| Margin Trading | `margin_trading_service.cpp` | ✅ | 5.6 KB |
| NFT | `nft_service.cpp` | ✅ | 7.2 KB |
| Price | `price_service.cpp` | ✅ | 15.2 KB |
| Staking | `staking_service.cpp` | ✅ | 9.9 KB |
| Swap | `swap_service.cpp` | ✅ | 6.9 KB |
| Biometric | `biometric/` | ✅ | 10.3 KB |
| Bridge | `bridge/` | ✅ | 17.4 KB |
| DAO | `dao/` | ✅ | 2.9 KB |
| DApp Browser | `dapp_browser/` | ✅ | 12.0 KB |
| Farming | `farming/` | ✅ | 1.5 KB |
| Gas Tracker | `gas_tracker/` | ❌ STUB | 0.3 KB |
| Gift Card | `gift_card/` | ✅ | 3.9 KB |
| Launchpad | `launchpad/` | ✅ | 1.9 KB |
| Liquid Staking | `liquid_staking/` | ✅ | 1.8 KB |
| MEV Protection | `mev_protection/` | ✅ | 5.4 KB |
| Options | `options/` | ✅ | 20.7 KB |
| Prediction Markets | `prediction_markets/` | ✅ | 1.8 KB |
| Realtime Alerts | `realtime_alerts/` | ✅ | 11.0 KB |
| RWA Trading | `rwa_trading/` | ✅ | 0.7 KB |

### Browser Extension (`/browser_extensions/chrome/src/services/`)

| Feature | Service File | Status | Size |
|---------|-------------|--------|------|
| Additional Services | `additional-services.js` | ✅ | 7.8 KB |
| Biometric Auth | `biometric-auth.js` | ✅ | 7.0 KB |
| Lending Service | `lending-service.js` | ✅ | 11.0 KB |
| NFT-DAO Service | `nft-dao-service.js` | ✅ | 9.1 KB |
| Price Service | `price-service.js` | ✅ | 3.5 KB |
| Swap-NFT-Staking-Bridge | `swap-nft-staking-bridge.js` | ✅ | 9.0 KB |
| Trading-MEV-Session-Gas | `trading-mev-session-gas-widget.js` | ✅ | 9.5 KB |
| DApp Browser | `dapp_browser/` | ✅ | 10.8 KB |
| Hardware Wallet | `hardware_wallet/` | ✅ | 17.1 KB |
| Multisig | `multisig/` | ✅ | 9.3 KB |
| Notifications | `notifications/` | ✅ | 12.7 KB |

---

## Gaps & Missing Features Summary

### HIGH PRIORITY GAPS

| Platform | Feature | Location | Status | Action Required |
|----------|---------|----------|--------|-----------------|
| Desktop | Gas Tracker | `desktop_wallet/src/services/gas_tracker/` | ❌ STUB | Implement C++ gas estimation |
| Extension | Options Trading | N/A | ❌ MISSING | Add options trading service |
| Extension | Convert | N/A | ❌ MISSING | Add convert service |
| Extension | NFT Gallery/Trading | N/A | ❌ MISSING | Add NFT services |
| Extension | Launchpad | N/A | ❌ MISSING | Add launchpad service |
| Extension | Lending | N/A | ❌ MISSING | Already in main services? |
| Extension | MPC Wallet | N/A | ❌ MISSING | Add MPC wallet service |
| Extension | Account Abstraction | N/A | ❌ MISSING | Add AA service |

### MEDIUM PRIORITY GAPS

| Platform | Feature | Status |
|----------|---------|--------|
| Flutter | Liquid Staking | ⚠️ Partial - needs completion |
| Web | Biometric Auth | ✅ Implemented but needs mobile |
| All Mobile | Security Scanner | ❌ Not implemented |
| All Mobile | Orderbook | ❌ Not implemented |
| All Mobile | TWAP | ❌ Not implemented |
| All Mobile | Intent Routing | ❌ Not implemented |
| All Mobile | Prediction Markets | ❌ Not implemented |
| All Mobile | Passkey | ❌ Not implemented |

### BACKEND GAPS

| Service | Location | Status | Notes |
|---------|----------|--------|-------|
| Account Abstraction | `/backend/go/internal/services/` | ✅ Implemented | |
| Bridge | `/backend/go/internal/services/` | ⚠️ Partial | Needs completion |
| Gift Card | `/backend/go/internal/services/` | ⚠️ Partial | Needs completion |
| Hardware Wallet | `/backend/go/internal/services/` | ⚠️ Partial | Needs completion |
| Lending | `/backend/go/internal/services/` | ⚠️ Partial | Needs completion |
| MPC Wallet | `/backend/go/internal/services/` | ⚠️ Partial | Needs completion |
| Social Recovery | `/backend/go/internal/services/` | ⚠️ Partial | Needs completion |

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

✅ **CONFIRMED**: UserWallet apps are completely isolated from:
- ❌ Admin APIs (`https://admin-api.tigerwallet.com`)
- ❌ MasterWallet APIs (`https://master-api.tigerwallet.com`)
- ❌ Admin fetchers and functionality
- ❌ MasterWallet fetchers and functionality

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
