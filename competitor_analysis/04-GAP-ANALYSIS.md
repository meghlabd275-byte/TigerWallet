# TIGERWALLET — GAP ANALYSIS vs COMPETITORS

**Companion docs:** [`03-TIGERWALLET-AUDIT.md`](./03-TIGERWALLET-AUDIT.md) (what is real/fake) and [`02-COMPETITOR-MATRIX.md`](./02-COMPETITOR-MATRIX.md) (what the competition ships). This file turns the audit findings into specific gaps per functional area, plus a prioritized roadmap.

Legend: 🔴 **Missing/Not built** · 🟠 **Stub or fake (exists in name, not real)** · 🟡 **Partial/real-but-incomplete** · 🟢 **Real & competition-grade.**

---

## Gap matrix

| # | Capability | Competitor bar | TigerWallet status | Verdict |
|---|---|---|---|---|
| 1 | **Real multi-chain HD wallet (user-facing)** | All 11 | 🔴 `go/wallet_api` is real, but **no frontend actually uses it end-to-end**. Web `app/wallet` calls nonexistent routes and fabricates `0x…`; mobile is fake. | 🔴 |
| 2 | **Onboarding / create-import by seed** | All 11 | 🟡 Real mnemonic gen exists (`master_wallet/page.tsx`, backend) but not wired into a working create/import UI. | 🟡 |
| 3 | **Hardware wallet support (Ledger/Trezor/Keystone)** | 5 of 11 | 🟠 `hardware_wallet/` dir exists but is stub/placeholder; no actual USB/HID/RPC integration. | 🟠 |
| 4 | **Multisig (real on-chain)** | OKX/Coinbase/enterprise | 🟠 `go/multisig_service` uses P-256 + *"In production, would broadcast"* — not a functioning multisig. | 🟠 |
| 5 | **Ethereum + EVM + Solana + BTC multi-chain support (working)** | All 11 | 🟡 EVM path real in `go/wallet_api`; Solana = no program, fake PDA; BTC partial (`wallet_core/bitcoin.rs` primitives). | 🟡 |
| 6 | **dApp browser** | 9 of 11 | 🟠 `dapp_browser/go` walletconnect service real; but **no shipping browser UI** wired to it. | 🟠 |
| 7 | **WalletConnect (incoming + outgoing)** | ~10 of 11 | 🟢 `go/walletconnect` relay + `dapp_browser/go` handlePersonalSign/typed-data are real. **Best-in-repo.** | 🟢 |
| 8 | **Injected EIP-1193 provider (extension)** | ~8 of 11 | 🟡 Extension delegates signing to backend (correct), but provider/`window.ethereum` parity with MetaMask-style dApp flow not proven end-to-end. | 🟡 |
| 9 | **Swap / DEX aggregator (auto-router, multi-dex)** | All 11 | 🟠 `swap_and_dex`, `dex_aggregator`, `cross_chain_aggregator`, `dex_connectors` dirs + `go/services/swap_service` all **hardcoded/simulated**. No real router. | 🟠 |
| 10 | **Cross-chain bridge** | Several | 🟠 `bridge/`, `cross_chain_protocol/` are stubs/unverified. | 🟠 |
| 11 | **Staking / DeFi yield** | All 11 | 🟠 `advanced_staking`, `staking_hub`, `liquid_staking`, `defi_*` + Go Lido service = **fabricated** txns (`"In production, …"`). | 🟠 |
| 12 | **Lending/borrowing (Aave-style)** | Many | 🔴 `defi_service` returns **`"txhash", nil`** and fake health factor. | 🔴 |
| 13 | **Perpetual futures / margin / options** | Trust/MetaMask/Phantom | 🟠 `perpetual_*`, `options_trading`, `margin_trading` mostly stub/simulated. | 🟠 |
| 14 | **Limit orders** | Trust/OKX | 🟡 `user_features/limit_orders/rust` present but not connected to any live market. | 🟡 |
| 15 | **Copy-trading / trading bots / MM** | OKX/Bybit-class | 🟠 Go/Rust services return **hardcoded/demo data**; `bot_*` UI admin real but no execution. | 🟠 |
| 16 | **Fiat on-ramp / buy with card** | All 11 | 🟠 `fiat_ramp`/`fiat_onramp` seed providers/rates **in memory**; no real on-ramp integration. | 🟠 |
| 17 | **Debit/crypto card & cash/money features** | MetaMask/Phantom/TokenPocket/Trust | 🔴 `crypto_card`, `payment_card` not real; no card partner. | 🔴 |
| 18 | **NFT support** | All 11 | 🟡 `nft_*` + CoinGecko/Etherscan NFT fetch in backend (real read layer); no mint/market/transfer flow. | 🟡 |
| 19 | **Custom token management** | All 11 | 🟢 `token_registry.go` + `token_management` real read layer; UI incomplete. | 🟡 |
| 20 | **Portfolio analytics / price charts** | All 11 | 🟢 Real CoinGecko/Etherscan fetchers in `go/wallet_api`; **`trading_charts`/`portfolio` are stubs**. | 🟡 |
| 21 | **On-chain analytics terminal / smart-money signals** | OKX/Phantom | 🔴 Not built (only marketing dirs). | 🔴 |
| 22 | **Prediction markets / RWA / tokenized assets** | MetaMask/Phantom 2026 | 🔴 `prediction_markets`/`rwa_trading` unverified/stub. | 🔴 |
| 23 | **ENS / naming** | MetaMask+ | 🟠 `ens/` service real-read but set-record *"would broadcast in production"*. | 🟠 |
| 24 | **Smart wallet / ERC-4337 account abstraction (gasless, paymaster)** | Coinbase/MetaMask | 🟠 Canonical base lib present but **no deployable Account/Paymaster/Factory**; `legacy_aa` is fake; P-256 MPC. | 🟠 |
| 25 | **MPC / embedded wallet / WaaS (enterprise)** | MetaMask/OKX/Coinbase | 🟠 `go/mpc` P-256 + accept-anything verify; not production MPC. | 🟠 |
| 26 | **Wallet-as-a-Service / SDK for devs** | MetaMask/Trust/OKX | 🟠 `sdk/`, `sdks/`, `wallet_cloud`, `embedded_wallet_sdk` = mostly scaffold. | 🟠 |
| 27 | **Social recovery / passkeys / social login** | Coinbase/Fireblocks-class | 🟠 `social_recovery`, `passkeys_auth`, `passkey` = placeholder. | 🟠 |
| 28 | **Gasless tx / gas-account / paymaster** | MetaMask/Bitget | 🟠 `gasless_tx`, `gas_account`, `paymaster_sdk` — paymaster is the fake (length==64) one. | 🟠 |
| 29 | **Fiat tax export / accounting** | Many | 🟡 `tax_export`/`tax_integration` present; unverified depth. | 🟡 |
| 30 | **Security: scam/honeypot/token scanner** | All 11 | 🟢 `honeypot_detector` + `wallet_guardian` are REAL (RPC reads + rule engine). `token_scanner`/`dapp_scanner` thin. | 🟡 |
| 31 | **Security: phishing/domain protection, MEV shield, tx simulation** | MetaMask/Trust/Phantom | 🟡 `mev_protection`, `transaction_shield`, `transaction_simulator` partially real; UI coverage thin. | 🟡 |
| 32 | **KYC/AML/compliance** | Exchanges | 🟠 `security/kyc_aml` = partition header/placeholder; no provider integration. | 🟠 |
| 33 | **2FA / biometric / face unlock** | Most | 🟠 `auth`/`biometric`/`multi_device_sync` = thin/stub; real 2FA only as admin JWT. | 🟠 |
| 34 | **Multi-platform parity (iOS/Android/ext/desktop)** | All 11 | 🔴 Only backend + extension + a single web page are real; the rest differ or are fake. | 🔴 |
| 35 | **White-label / B2B multi-tenant** | OKX WaaS | 🟡 `white_label*` admin/DB plumbing real; wallet-feature breadth inside is still the stub surface. | 🟡 |
| 36 | **Production infra (observability, monitoring, scaling)** | All | 🟢 Real `monitoring_dashboard`, `observability`, `notification`, compose stack (10 services). | 🟢 |

---

## What is genuinely "real & done" (build on these)

1. **Key management & signing — `go/wallet_api`** (BIP-39/32/44, secp256k1, EIP-1559/191/712, AES-GCM, scrypt, PG+Redis). This is a **real, shippable wallet engine**.
2. **WalletConnect v2 relay + personal-sign/typed-data handler** (`go/walletconnect`, `dapp_browser/go`).
3. **EIP-191 signature service.**
4. **ZK Schnorr prover** (`core/rust/zk_infrastructure`).
5. **Honeypot detector + wallet guardian** (real RPC-driven scam/token security).
6. **Real coin/token/price/NFT fetchers** (CoinGecko/Etherscan/eth balance) in `go/wallet_api`.
7. **Operational plumbing**: Postgres/Redis, admin DB layers, monitoring, compose.

---

## Top gaps to close (priority order)

1. **Wire the real backend into ONE shipping frontend.** Pick web (`Next.js`) as the flagship: replace the broken `/api/wallet/*` + `generateRandomAddress()` flow with real `/api/v1` calls to `wallet_api` (create → get address → sign → broadcast). Delete the fake fallback. This alone converts the repo from "engine with a broken demo" to "working wallet."
2. **Fix mobile.** Rewrite Flutter/Android mnemonic/derivation to use a real BIP-39 lib (or the backend) — current code is non-compliant, uses XOR/sha256-KDF, zero keys, and fabricated hashes. Ship one real mobile app rather than three fake ones.
3. **ERC-4337 smart wallet (the biggest differentiation vs peers).** Port `AccountFactory.sol` to `PackedUserOperation` + extend `BaseAccount`/`BasePaymaster` (AGENTS.md already flags this); vendor OpenZeppelin so `forge build` passes; deploy a real account/paymaster/factory.
4. **Real swap/router + ledger service.** Replace hardcoded `swap_service`/`defi_service` with a real aggregator hitting `eth_call`/`eth_sendRawTransaction` through the existing real RPC layer.
5. **Hardware-wallet + MPC with real math** — replace P-256 "MPC" (accept-anything verify) with proper threshold/ECDSA; wire Ledger/Trezor support into the extension.
6. **Fiat on-ramp + card partners** — a real provider (MoonPay/Sardine etc.) rather than in-memory seed data.
7. **Solana program** — write a real Anchor program instead of the C++ fake-PDA; or at least drop the false claim.
8. **Remove build-breakers**: delete/pull the committed 22 MB ELF binary, add `go.mod` to the orphan modules, fix the missing `go.mod` set, and make `forge build` green.

---

## Honest bottom line

**The engineering core (key mgmt, signing, WC2, ZK, scam-detection, fetchers) is real and well above most hobby repos and is comparable to what competitors ship inside their secure enclaves.** But **the product is not built** — the vast majority of user-facing features in the 141-module spec are stubs/fakes, and there is **no single end-to-end working wallet** across web/mobile/desktop. Relative to Trust/MetaMask/OKX/Phantom, which ship real (and increasingly unified) products, TigerWallet's gap is **not more plumbing — it is finishing and shipping at least one real product surface end-to-end**, plus making the ERC-4337/smart-wallet and mobile layers real rather than decorative.