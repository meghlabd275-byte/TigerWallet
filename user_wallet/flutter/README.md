# TigerWallet UserWallet — Flutter

Thin REST/WS client over the canonical `go/wallet_api` backend (:8443). Same
feature surface as web / android / ios / desktop_app / extension / rust.

- Onboarding: Create Wallet / Import Wallet -> backup (copy + encrypted export).
- Full feature hub: send/auto-send, swap/AMM, staking, bridge, DeFi/lending,
  trading (perp+margin), earn (launchpool/token-sales), social (copy-trading,
  P2P, DAO, prediction), NFTs, identity (KYC/ENS), payments (cards/ramp),
  security, terminal, fees, organization (devices/address-book/price alerts),
  non-EVM, approvals, multisig, dApps/WalletConnect, chains/tokens.
- Light/dark theme with persistence (`ThemeService`).
- Backend URL user-configurable and persisted (Dashboard -> Backend settings).
- Live price feed over `/api/v1/ws` (`LiveFeedSocket`).
- Separation rule honored: only calls wallet_api; no MasterWallet/Admin backend.

## Run

    flutter pub get
    flutter run            # android / ios / web / desktop

Verified: `flutter analyze` 0 issues; `flutter build web --release` PASS
(Flutter 3.27.4, Dart 3.6.2).
