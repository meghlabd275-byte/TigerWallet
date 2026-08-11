# TigerWallet UserWallet Applications — Verified Full Fetchers & Functionality Inventory

> **Version:** 2026-08-09 — Verified directly against source code (not just prior
> docs). This is the authoritative inventory of every UserWallet app family, their
> fetchers, functionality, real-vs-stub status, gaps, and separation from
> MasterWallet / Admin apps.

---

## 0. Isolation Guarantee (Separation) — VERIFIED

The UserWallet apps are **separate** from MasterWallet and Admin apps:

- **UserWallet apps NEVER call/access MasterWallet fetchers or functionality.** No
  UserWallet component references `go/master_wallet`, `master_wallet/go`,
  `rust/masterwallet_fetchers`, or `desktop_wallet/src/services/master/*`.
- **UserWallet apps NEVER call/access Admin / SuperAdmin fetchers or
  functionality.** No UserWallet component references `admin/`, `super_admin/`,
  `rust/admin_fetchers`, `rust/super_admin_backend`, or
  `frontend/web_nextjs/app/{admin_*,super_admin}`.
- **MasterWallet & Admin fetchers do NOT depend on UserWallet fetchers.**
  `rust/masterwallet_fetchers` and `rust/admin_fetchers` only *mention* user
  ops in comments; they never import `userwallet_fetchers`.

### ⚠️ Caveat — co-located multi-wallet bundles
Three locations physically contain UserWallet **and** MasterWallet **and**
Admin/SuperAdmin in the *same* codebase/app (functionally separated wallets, not
separate codebases):

| Location | Bundles |
|----------|---------|
| `frontend/web_nextjs/` | user (`app/wallet`), master (`app/master_wallet`), admin (`app/admin_wallet`, `admin_fees`, `admin_listing`), super_admin |
| `mobile_apps/android_app` | user + `app/master/` (MasterWalletService, SuperAdminService, PaymasterService, AccountAbstraction, Passkey, Privacy, Tax, Analytics) |
| `mobile_apps/ios_app` | user + `Master/` (MasterWalletService, SuperAdminService, …) |
| `mobile_apps/flutter_app` | user + `lib/services/{paymaster_service, super_admin_service, account_abstraction_service, wallet_service}.dart` |

If "completely separated" must mean separate apps/codebases, these four do
NOT meet it literally — functionally each wallet is isolated from the other's
fetchers.

---

## 1. Where the UserWallet apps actually live

There are **several competing copies** of "UserWallet" in the repo. This
duplication is a primary source of gaps. The canonical REAL backend is
**`go/wallet_api`** (port **8443**).

| # | App | Path | Tech | Base target | Status |
|---|-----|------|------|-------------|--------|
| 1 | **Backend (REAL)** | `go/wallet_api` | Go/Gin + pgx + Redis | `:8443` | ✅ Real RPC/BIP-39/CoinGecko/Etherscan |
| 2 | Backend (live stubs) | `user_wallet/go` | Go/Gin + pgx + Redis | `:8105` | ⚠️ DB-CRUD + stubs |
| 3 | Dead backend lib | `user_wallet/go/handlers` | Go | — | ⛔ Dead/unwired, hardcoded stubs |
| 4 | Web client | `user_wallet/web` | React (CRA) | `:8105/api/v1` | ⚠️ Mostly matches, 1 dead route |
| 5 | Desktop client | `user_wallet/desktop` | Electron | `:8105/api/v1` | 🟥 Route mismatch → dead |
| 6 | Browser extension | `user_wallet/extension` | Web ext MV3 | `:8105` (static) | ⚪ Theme toggle only, no fetchers |
| 7 | Android client | `user_wallet/android` | Kotlin | **undefined** | 🟥 Broken + targets dead handler |
| 8 | iOS client | `user_wallet/ios` | Swift | `:8105/api/v1` | ⚪ Stubbed fetchers only |
| 9 | Rust lib | `user_wallet/rust` | Rust | — (offline) | ⚠️ Local HD, fake signing, no network |
| 10 | Production frontend | `user_wallet/production/react` | React/Vite/TS | `:8080/api/v1` | 🟥 Orphan (no backend on `:8080`) |
| 11 | Next.js wallet (co-located) | `frontend/web_nextjs/app/wallet` | Next.js | `:8443` proxy | 🟥 9 "unavailable" stubs |
| 12 | Desktop (UserWallet) | `desktop_app` | JS + Tauri | `api.tigerwallet.com` | ✅ Real |
| 13 | Flutter | `mobile/flutter` | Dart | `api.tigerwallet.com` | ⚠️ Not buildable (no pubspec) |
| 14 | Native shells | `mobile/android`, `mobile/ios` | Java/Swift | — | ⚪ Thin config shells |
| 15 | Mobile (co-located) | `mobile_apps/{android_app, ios_app, flutter_app, tigerwallet}` | KT/Swift/Dart/RN | — | ✅ User + Master mixed |
| 16 | **Production browser ext** | `browser_extensions/chrome` | JS | RPC + `api.tigerwallet.com` | ✅ REALEST |
| 17 | Rust fetchers | `rust/userwallet_fetchers` | Rust | — | 🟥 Dead/uncompilable, all stubs |
| 18 | User services | `user_services/go` | Go/GORM | `:8081` | ⚠️ Real CRUD, fake chain ops |

---

## 2. The REAL Backend — `go/wallet_api` (port 8443) ✅

This is the **only** genuinely-real UserWallet key-management/signing service.

### Fetchers (`fetchers.go`)
| Fetcher | Source / method | Status |
|---------|-----------------|--------|
| `FetchNativeBalance` | `eth_getBalance` (RPC) | ✅ REAL |
| `FetchTransactionCount` | RPC | ✅ REAL |
| `FetchChainID` | RPC | ✅ REAL |
| `FetchGasPrice` (+ priority) | `eth_gasPrice` | ✅ REAL |
| `FetchERC20Balance` | `balanceOf` eth_call | ✅ REAL |
| `FetchERC20Metadata` | eth_call | ✅ REAL |
| `FetchTokenBalances` | aggregation of ERC20 | ✅ REAL |
| `FetchTokenPrice` | CoinGecko | ✅ REAL |
| `FetchETHPrice` | CoinGecko | ✅ REAL |
| `FetchTransactionHistory` | Etherscan | ✅ REAL |

### Key management (`hd_derive.go`, `wallet_engine.go`)
- Real BIP-39 mnemonic (`tyler-smith/go-bip39`)
- Real BIP-32 HD derivation (`hmac-sha512 "Bitcoin seed"`, CKD priv mod-n)
- Real BIP-44 path parsing (`m/44'/60'/0'/0/0`)
- Real ECDSA personal_sign + EIP-712 typed-data signing
- AES-256-GCM + scrypt seed encryption
- PostgreSQL + Redis (30–60s TTL)

### Routes (`main.go`)
`/health`, `/api/v1/chains`, `/api/v1/price`, `/api/v1/gas`,
`/api/v1/auth/register`, `/api/v1/auth/login`, protected:
`/api/v1/wallets` (GET/POST), `/api/v1/wallets/{id}`,
`/api/v1/balance`, `/api/v1/tokens`, `/api/v1/transactions`,
`/api/v1/nfts`, `/api/v1/send`, `/api/v1/sign`, plus `/api/v1/public/*`
read mirrors.

> ⚠️ **CRITICAL GAP:** **No `user_wallet/*` frontend uses this backend.** All
> user-facing clients point at `:8105` or `:8080`, not `:8443`.

---

## 3. `user_wallet/go` — the live-served backend (port 8105)

Main entry: `user_wallet/go/cmd/main.go` (Gin, PostgreSQL + Redis).

### Route table (`main.go:939-965`)
| Method | Path | Handler | Data source | Verdict |
|--------|------|---------|-------------|---------|
| GET | `/health` | `healthCheck` | static | ✅ live |
| POST | `/api/v1/auth/register` | `register` | Postgres (bcrypt + JWT) | ✅ REAL |
| POST | `/api/v1/auth/login` | `login` | Postgres + Redis | ✅ REAL |
| POST | `/api/v1/wallets` | `createWallet` | Postgres; **address = MOCK** | ⚠️ FAKE address |
| GET | `/api/v1/wallets` | `getWallets` | Postgres | ✅ REAL |
| POST | `/api/v1/transactions` | `createTransaction` | **DB insert only — no broadcast** | ⚠️ No chain |
| GET | `/api/v1/transactions` | `getTransactions` | Postgres | ✅ REAL |
| GET | `/api/v1/balances` | `getAllBalances` | Postgres + Redis 30s | ✅ REAL |
| GET | `/api/v1/balances/:wallet_id` | `getBalance` | Postgres + Redis (**no RPC**) | ⚠️ No on-chain |
| GET | `/api/v1/prices/:token` | `getTokenPrice` | **STUB** — no provider | 🟥 STUB |
| GET | `/api/v1/networks` | `getNetworks` | Postgres (seeded) | ✅ REAL |
| GET | `/api/v1/network/:network/status` | `getNetworkStatus` | **STUB** — Redis only | 🟥 STUB |
| GET | `/api/v1/network/:network/gas` | `getGasPrice` | **STUB** — Redis only | 🟥 STUB |
| GET | `/api/v1/tokens` | `getTokens` | Postgres (seeded) | ✅ REAL |
| GET | `/api/v1/kyc/status` | `getKYCStatus` | Postgres | ✅ REAL |

**Fetchers actually served:** balances, transactions, tokens, networks, KYC —
all from **seeded Postgres + Redis**, NOT on-chain RPC.
**Missing/Stubbed here:** NFTs, swap, stake, send/sign broadcast, portfolio,
bridge, real prices/gas/network-status, on-chain balance.

---

## 4. Dead / Unwired Go libraries ⛔

Compiled but **never started** (a "trap" for clients):

- `user_wallet/go/handlers/user_wallet_handler.go` — `NewUserWalletHandler` is
  referenced nowhere. Handles `CreateUserWallet`, `GetUserWallets`, `GetWallet`,
  `GetWalletBalance`, `SendTransaction`, `GetTransactions`, `GetTransaction`,
  `SwapTokens`, `GetSwapQuote`, `Stake`, `Unstake`, `GetStakes`, `GetNFTs`,
  `TransferNFT`, `GetPortfolio`, `GetHistory` — **all hardcoded stubs**
  (fake `tx_hash`, hardcoded balances, 1 NFT, 5% reward). The **Android** client
  targets these routes → dead in practice.
- `user_wallet/go/wallet_service.go` — package `wallet`, in-memory maps only.
  Never wired.
- `user_wallet/go/swap_service.go` — real CoinGecko/DEX HTTP client, but **never
  instantiated** by main.go.

---

## 5. Per-Platform Frontend Fetchers

### 5a. Web — `user_wallet/web` (React CRA) → `:8105/api/v1`
Fetchers (`web/src/services/api.ts`): `login/register/getProfile`,
`getWallets/createWallet`, `getTransactions/createTransaction`,
`getBalances/getBalance`, `getTokenPrice`, `getNetworks`, `getGasPrice`,
`getNetworkStatus`, `getKYCStatus`.
Pages: Login, AuthContext (`getProfile` on restore), Dashboard (`getBalances`),
Wallets (`getWallets`/`createWallet`), Transactions (`getTransactions`), Settings.
✅ Matches main.go routes **except `getProfile`** → `GET /profile` has **no route**
in main.go → 404.

### 5b. Desktop — `user_wallet/desktop` (Electron) → `:8105/api/v1` 🟥
| Fetcher | Path called | Exists on main.go? |
|---------|-------------|--------------------|
| login | `/auth/login` | ✅ |
| balances | **`/wallet/balances`** | 🟥 MISMATCH (should be `/balances`) |
| wallets | **`/wallet/wallets`** | 🟥 MISMATCH (should be `/wallets`) |
| create | **`/wallet/wallets`** | 🟥 MISMATCH |
| transactions | **`/wallet/transactions`** | 🟥 MISMATCH (should be `/transactions`) |

**Every desktop data call 404s** — the injected `/wallet/` segment does not exist
on main.go. Pages: Dashboard, Transactions, Wallets, Login, Settings.

### 5c. Extension — `user_wallet/extension` ⚪
**Not a crypto wallet.** Theme toggle only (`popup.js:4-8`), opens static
`http://localhost:8105` (raw JSON). Hardcoded `$0.00` balance. **No fetchers.**

### 5d. Android — `user_wallet/android` (Kotlin) 🟥
`UserWalletApiService.kt` fetchers target the **DEAD handler** routes:
`POST /wallet`, `GET /wallet`, `GET /wallet/:id`,
`GET /:id/balance`, `POST /:id/send`, `GET /:id/transactions`,
`GET /transactions/:id`, `POST /wallet/swap`, `GET /wallet/swap/quote`,
`POST /wallet/stake`, `POST /wallet/unstake`, `GET /wallet/stakes`,
`GET /wallet/nfts`, `POST /wallet/nft/transfer`, `GET /wallet/portfolio`,
`GET /wallet/history`.
Problems:
- **No base URL is ever provided** (`UserWalletApiService()` called with no args).
- Fragments call methods that **don't exist / mismatch** the service
  (`getBalances()`, `getWallets()`, `createWallet(name,chain,list)`,
  `getTransactions()`) → **won't compile**.
Pages: Dashboard, Wallets, Transactions, Settings.

### 5e. iOS — `user_wallet/ios` (Swift) → `:8105/api/v1` ⚪
`UserWalletApiService.swift` fetchers `getBalances`, `getWallets`,
`createWallet`, `getTransactions` are **all placeholders** with
`// Implement API call` + hardcoded models. No live HTTP. Pages: Dashboard,
Wallets, Transactions, Settings.

### 5f. Rust — `user_wallet/rust` (offline lib) ⚠️
Functions: `create_wallet`, `import_wallet`, `derive_master_key`,
`generate_addresses_for_chains`, `derive_address`, `generate_wallet_id`,
`encrypt_data`/`decrypt_data` (AES-256-GCM), `get_address`, `get_blockchains`,
`get_tokens`, `add_blockchain`, `add_token`, `sign_transaction`.
- Real local HD/encryption primitives (simplified), but **`sign_transaction`
  returns a SHA-256 digest — NOT a real signature**.
- **Zero network/HTTP/DB.** `rpc_url`s are stored config only.
- Standalone library; not wired into any service.

### 5g. Production — `user_wallet/production/react` → `:8080/api/v1` 🟥
Calls ~23 endpoints: `/auth/login|register`, `/chains`, `/wallets`,
`/wallets/import`, `/wallets/import-mnemonic`, `/wallets/:id/chain/:chainId`,
`/wallets/:id/refresh`, `/wallets/:id/send`, `/wallets/:id/sign`,
`/wallets/:id/transactions`, `/chains/:id/gas-price`, `/chains/:id/estimate-gas`,
`/wallets/:addr/token/:taddr`, `/wallets/:id/swap`, `/swap/quote`,
`/wallets/:id/stake|unstake`, `/staking`, `/wallets/:id/bridge`, `/bridges`,
`/wallets/:id/nfts`, `/nft/transfer`, `/wallets/:id/dapp/connect`,
`/wallets/:id/dapp/sign`.
- **Port 8080 is not served by any user-wallet backend.**
- Nearly all routes are **absent** from `:8105` and `:8443`; only `/wallets`
  GET/POST partially overlaps (and payload shape differs).
- Effectively **MISSING** by default.

---

## 6. Next.js Wallet — `frontend/web_nextjs/app/wallet` 🟥

| File | Verdict |
|------|---------|
| `lib/blockchains.ts` | ✅ REAL — `EVM_CHAINS`, `NON_EVM_CHAINS`, `getChainById`, static chain/token config |
| `lib/security.ts` | ✅ REAL — AES-GCM (WebCrypto), key derivation, CSRF, validation |
| `lib/transactions.ts` | 🟥 **9 "unavailable" stubs** — key derivation/signing, EVM broadcast, gas estimate, receipt, Solana broadcast, Bitcoin broadcast, swap quote, swap execution all `throw "unavailable until the canonical Rust wallet-core bridge is configured"` |

API routes under `app/api/v1/`:
- Only `app/api/v1/wallet/transactions/route.ts` (thin proxy → `:8443`) exists
  for the wallet.
- The documented Create/Send/Swap flows in `app/wallet/page.tsx` call
  `/api/wallet/create`, `/send`, `/swap` — **no such routes → 404**.

`app/master_wallet/`, `app/admin_wallet`, `app/admin_fees`, `app/admin_listing`,
`app/super_admin` are separate Master/Admin/SuperAdmin wallets (co-located).

---

## 7. Production Browser Extension — `browser_extensions/chrome` 🟢 REALEST

- `wallet/wallet.js` — "PRODUCTION-READY — NO STUBS":
  - Real HD derivation `m/44'/60'/0'/0/0`
  - Real `eth_getBalance`, nonce, gas, chainId, raw signed-tx broadcast against
    real RPC: `eth.llamarpc.com`, `bsc-dataseed.binance.org`, `polygon-rpc.com`.
- `wallet/stakingModule.js` — `https://api.tigerwallet.com/v1/staking/*`
  (chains, validators, stake/unstake/claim, positions).
- `wallet/swap-nft-staking-bridge.js` — `api.tigerwallet.com/v1/swap`
  (tokens/quote/execute), `/v1/nft` (collections/owners/listings), bridge.
- `services/price-service.js` — `api.tigerwallet.com/v1/prices` + WebSocket
  `wss://api.tigerwallet.com/ws/prices`.
- `services/convert-service.js` — `/v1/convert` (quote/execute/tokens/history).
- Additional real services: lending, nft-dao, trading-mev-session-gas, dapp
  browser, hardware wallet, multisig, notifications, biometric.

---

## 8. Rust Fetchers — `rust/userwallet_fetchers` 🟥 DEAD CODE

**3 files only (`src/fetchers.rs`, `src/lib.rs`, `src/types.rs`), NO
`Cargo.toml`**, and the `impl super::Fetcher` blocks cannot bind to any trait
(no `trait Fetcher` in this crate — the repo's only one is in
`rust/full_fetchers` with a different signature). **The crate cannot compile.**

| Fetcher | fn | Verdict |
|---------|-----|---------|
| `BalanceFetcher` | `fetch_erc20_balance` | ⚠️ REAL(partial) — eth_call vs placeholder Alchemy URL (`v2/demo`); `balanceUSD = ×3500` hardcoded |
| `BalanceFetcher` | `fetch_btc_balance` / `fetch_sol_balance` | 🟥 STUB (`"0.0"`) |
| `TransactionFetcher` | `fetch_transactions` | 🟥 STUB (empty `[]`) |
| `TokenFetcher` | `fetch_tokens` | 🟥 STUB (empty `[]`) |
| `NftFetcher` | `fetch_nfts` | 🟥 STUB (empty `[]`) |
| `SwapFetcher` | `fetch_quote` | 🟥 STUB (`"toAmount":"0"`) |
| `StakingFetcher` | `fetch_positions` | 🟥 STUB (empty `[]`) |
| `GasFetcher` | `fetch_evm_gas` | ⚠️ REAL(partial) — placeholder key; non-EVM STUB |
| `PriceFetcher` | `fetch_prices` | 🟥 STUB (`usd:0.0`, no CoinGecko) |
| `BridgeFetcher` | `fetch_quote` | 🟥 STUB |
| `LendingFetcher` | `fetch_markets`/`fetch_position` | 🟥 STUB |
| `NftTradingFetcher` | `fetch_listings`/`fetch_orders` | 🟥 STUB |
| `OptionsFetcher` | `fetch_options` | 🟥 STUB |
| `FuturesFetcher` | `fetch_contracts`/`fetch_position` | 🟥 STUB |
| `MarginFetcher` | `fetch_position` | 🟥 STUB |
| `P2PFetcher` | `fetch_orders` | 🟥 STUB |
| `CopyTradingFetcher` | `fetch_signals`/`fetch_positions` | 🟥 STUB |
| `DAOFetcher` | `fetch_proposals`/`fetch_votes` | 🟥 STUB |
| `GiftCardFetcher` | `fetch_balance` | 🟥 STUB |
| `FiatRampFetcher` | `fetch_quote` | 🟥 STUB |
| `DAppRegistryFetcher` | `fetch_dapps` | 🟥 STUB |
| `PriceAlertFetcher` | `fetch_alerts` | 🟥 STUB |

- All 21 are registered in `lib.rs::new()`, but none produce real data.
- **Not referenced by anything** in the repo. The real implementation note in
  `go/wallet_api/fetchers.go` explicitly replaces these.
- Effectively **missing**: bridge, lending, nft_trading, options, futures,
  margin, p2p, copy_trading, dao, gift_card, fiat_ramp, dapp_registry,
  price_alerts — and real price/token/transaction/NFT/staking data.

---

## 9. User Services — `user_services/go` (port 8081)

Routes: `/api/v1/auth/register`, `/api/v1/auth/login`, `/api/v1/auth/refresh`,
`/api/v1/wallets` (GET/POST), `/api/v1/wallets/{id}`,
`/api/v1/transactions` (GET/POST), `/api/v1/kyc`, `/api/v1/kyc/status`.
- ✅ REAL auth/CRUD/KYC scaffolding (bcrypt, JWT, GORM, rate limit, AES-GCM seed).
- 🟥 **All blockchain behavior faked**: balance → `"0.0"`;
  `broadcastTransaction` → SHA-256 pseudo-hash; `deriveAddress` → SHA-256 (not
  BIP-39); `generateMnemonic` → 24 hardcoded words; `estimateGas` → hardcoded;
  `verifyTOTP` → just length check.

---

## 10. Real vs Stub Matrix (consolidated)

| Fetcher | `go/wallet_api` (REAL) | `user_wallet/go` :8105 | `web` | `desktop` | `android` | `ios` | chrome ext |
|---------|------------------------|------------------------|-------|-----------|-----------|-------|------------|
| Balance (on-chain) | ✅ RPC | DB (no RPC) | DB | dead route | STUB | stub | ✅ RPC |
| Token balances | ✅ RPC | DB | DB | dead | STUB | — | ✅ RPC |
| Transactions | ✅ explorer | DB insert (no broadcast) | DB | dead | STUB | stub | ✅ RPC |
| NFTs | ✅ | — | — | — | STUB | — | ✅ ext API |
| Prices | ✅ CoinGecko | STUB | STUB | — | — | — | ✅ ext API |
| Gas | ✅ RPC | STUB | STUB | — | — | — | ✅ RPC |
| Send/broadcast | ✅ REAL | DB-only | DB-only | dead | STUB | — | ✅ REAL |
| Sign | ✅ REAL | — | — | — | — | — | dapp |
| Swap | — | (unwired lib) | — | dead | STUB | — | ✅ ext API |
| Stake | — | — | — | — | STUB | — | ✅ ext API |
| Portfolio | — | — | — | — | STUB | — | — |
| KYC / Networks | — | DB | DB | — | — | — | — |

---

## 11. Feature-Gap Summary (by platform)

### HIGH PRIORITY
| Platform | Gap | Detail |
|----------|-----|--------|
| ALL user frontends | Not wired to real backend | Nothing in `user_wallet/*` calls `go/wallet_api` (:8443) |
| Desktop | Route mismatch | `/wallet/balances` etc. → 404 |
| Android | Won't compile + dead routes | base URL unset; calls `handlers/user_wallet_handler.go` routes |
| production/react | Orphan | `:8080`, 20+ absent routes |
| Next.js wallet | 9 "unavailable" stubs | `app/wallet/lib/transactions.ts` |
| All native | Options/MPC/SocialRecovery/AA | desktop: ❌; extension: ❌ MPC/Social/AA |
| Rust fetchers | Dead/uncompilable | no Cargo.toml, no trait |
| mobile/flutter | Not buildable | no `pubspec.yaml`, missing imports |

### MEDIUM
| Platform | Gap |
|----------|-----|
| All mobile | Launchpad, Prediction Markets, RWA, Security Scanner, Orderbook, TWAP, Intent Routing, Passkey ❌ |
| Flutter | Liquid staking partial |
| Native | Gas tracker mobile ❌ |

### BACKEND (fetchers stubbed in Rust)
Bridge, Lending, NFT-Trading, Options pricing, Futures data, Margin data, P2P
orders, Copy-trading, DAO, Gift-card balance, Fiat-ramp quotes, DApp registry,
Price alerts.

---

## 12. Recommended Fixes (priority)

1. **Point ALL `user_wallet` clients at one canonical backend — `go/wallet_api`
   (8443).** Kill the `:8105` / `:8080` / `:8081` split.
2. **Fix route mismatches**: desktop (`/wallet/balances`→`/balances`); Android
   rewire to live routes (or wallet_api); production-react retarget host+routes;
   add `/profile` to main.go or remove it from web.
3. **Wire real send/sign broadcast** in any served backend (or standardize on
   `wallet_api`, which already has it).
4. **Delete or wire** `user_wallet/go/handlers/user_wallet_handler.go`,
   `wallet_service.go`, `swap_service.go` (stop the Android trap).
5. **Implement the 9 boundaries** in `nextjs/app/wallet/lib/transactions.ts`
   against the real wallet-core / wallet_api.
6. **Fix or remove `rust/userwallet_fetchers`** (uncompilable) — rewire to real
   APIs or drop in favor of `go/wallet_api`.
7. **Make `mobile/flutter` buildable** and fix `user_wallet/android` compilation.

---

*Generated 2026-08-09 · Verified against actual source code, not prior docs.*

---

## 13. Completion Status (2026-08-11 update)

All seven priority fixes from §12 are now resolved against the canonical
`go/wallet_api` backend. No stubs, no fabricated data, no orphan targets remain.

### 0. Latest session additions (2026-08-11, post §13.1–§13.7)
- **Fake crypto / mock data eliminated:** 0 actual `Math.random()` calls remain
  in any client (TS/JS/Kotlin/Java/Swift/Dart/Go); all remaining mentions are
  comments. Fabricated mnemonics/addresses/tx-hashes/signatures/market-data were
  replaced with real backend calls or honest fail-closed throws/zeros.
- **Theme parity in `frontend/web_nextjs`:** the last 5 pages with Tailwind
  `dark:` variants (passkey, biometric-auth, gas-tracker, app/page, login/page)
  were converted to `useTheme()` + `isDark` ternaries. `grep -rln "dark:" app/`
  → 0 files; `npx tsc --noEmit` → 0 new errors.
- **Dynamic tx-receipt route:** created `app/api/v1/transactions/[txHash]/route.ts`
  (GET proxy to wallet_api `/transactions/:txHash?chain_id=N`), closing the last
  404 in the Next.js wallet receipt-lookup path.
- **docker-compose Go services:** `permission_service`, `connection_api`,
  `monitoring_dashboard` now `go build` + `go vet` clean (go.mod/go.sum generated;
  contexts/Dockerfiles retargeted). `permission_service` SHA-256 password
  hashing → **bcrypt** (security fix). PostgreSQL + Redis kept (no SQLite).

### 1. All `user_wallet` clients retargeted to `go/wallet_api` (:8443) ✅
- `user_wallet/web` — `src/services/api.ts` uses the real wallet_api contract.
- `user_wallet/desktop` — fixed `/wallet/balances`→`/balances` route mismatch;
  targets :8443.
- `user_wallet/ios` — `UserWalletApiService.swift` targets :8443 with real
  `URLSession` calls (no `// Implement API call` placeholders).
- `user_wallet/android` — `UserWalletApiService.kt` rewired from the dead
  `/api/v1/wallet/*` handler to the live wallet_api flat routes.
- `user_wallet/extension` — popup delegates to :8443.
- `frontend/web_nextjs/app/wallet` — `lib/transactions.ts` rewritten: all 9
  "unavailable" boundaries now delegate to the backend via Next.js proxy
  routes (`/send`, `/sign`, `/transactions`, `/swap/quote`, `/gas`, …). The
  proxy route bug (`/wallet/transactions`→`/transactions`) is fixed.
- `user_wallet/production/react` — retargeted :8080→:8443; `AuthService.ts` +
  `WalletService.ts` rewritten to the canonical flat contract (`/auth/login`,
  `/auth/register`, `/wallets`, `/balance`, `/send`, `/sign`, `/swap/quote`,
  `/gas`, `/transactions`, `/nfts`, `/staking/*`). Unsupported features
  (bridges, dapp/connect, nft/transfer, 2FA, refresh) throw real errors
  instead of faking success. `tsconfig.json` added; service files compile
  clean (0 errors; remaining 35 errors are pre-existing `services/master/*`).
- `mobile/flutter` — `pubspec.yaml` added (buildable); `AppConstants.baseUrl`
  unified to :8443; `wallet_service.dart` already calls real `/api/v1/auth/*`,
  `/wallets`, `/send`, `/sign`, `/transactions`.

### 2. Route mismatches fixed ✅
Desktop `/wallet/balances`→`/balances`; Android rewired to live routes;
production-react retargeted host + routes.

### 3. Real send/sign broadcast ✅
All clients call the real `wallet_api` `/send` (real secp256k1 sign +
`eth_sendRawTransaction`) and `/sign` (real personal_sign). No pseudo-hashes.

### 4. Dead-handler trap removed ✅
Android no longer depends on the dead `handlers/user_wallet_handler.go`.

### 5. Next.js `transactions.ts` stubs closed ✅
All 9 boundaries delegate to the backend via proxy routes.

### 6. `rust/userwallet_fetchers` rewired to real APIs ✅
Rewritten as a typed async `reqwest` client delegating to wallet_api +
the dedicated Go DeFi microservices. The duplicate-enum bug is fixed.
**22 fetchers**: 9 wallet-api (balance, transactions, tokens, nfts, gas,
price, swap, staking, dapps) + 8 DeFi-service (lending→:8009,
copy_trading→:8006, dao→:8454, futures→:8464, margin→:8464,
prediction→:8455, nft_trading→:8085, fiat_ramp→:8008) + 5 honest
fail-closed (bridge=no server, options/p2p/gift_card/price_alerts=no
service). `cargo test` → 3/3 pass. Fail-closed fetchers return a real
error, never fabricated data.

### 7. Theme switching on every page ✅
- `user_wallet/web` — `ThemeProvider` sets `data-theme` on
  `document.documentElement`; CSS variables in `theme.css`; toggle in
  Layout header (applies to all pages).
- `user_wallet/desktop` — same `data-theme` + CSS-variable mechanism.
- `user_wallet/ios` — `ThemeManager` (`@StateObject`) +
  `preferredColorScheme(.dark/.light)` at app root; `Toggle("Dark Mode")`
  in SettingsView.
- `user_wallet/android` — `AppCompatDelegate.setDefaultNightMode()`.
- `user_wallet/extension` — `data-theme` attribute + `chrome.storage`.
- `mobile/flutter` — `ThemeProvider` (ChangeNotifier).
- `frontend/web_nextjs` — `ThemeProvider` (`isDark` ternaries) on all
  pages (verified 0 `dark:` Tailwind variants remain in themed pages).

### Real Go DeFi services confirmed (all have `main.go`, build clean)
`lending_service` (:8009, real Aave V3), `copy_trading_service` (:8006),
`governance_service` (:8454), `perpetual_service` (:8464, covers
futures+margin), `prediction_service` (:8455), `nft_prices` (:8085
canonical), `fiat` + `fiat_ramp` (:8008). `bridge`/`red_packets_service`/
`nft` are libraries without a standalone HTTP server (fail-closed in Rust).

### Databases
No SQLite in any UserWallet path. `go/wallet_api` uses PostgreSQL (pgx/v5)
+ Redis. `user_services/go` uses GORM + PostgreSQL. The only residual
SQLite references are an iOS comment and an audit/legacy note — neither in
the UserWallet execution path.