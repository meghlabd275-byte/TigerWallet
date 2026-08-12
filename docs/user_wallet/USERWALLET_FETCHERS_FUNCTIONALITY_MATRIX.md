# TigerWallet UserWallet — Full Fetchers & Functionality Per-App (Cross-Platform Matrix)

> **Version:** 2026-08-09. Complete, verified detail of every UserWallet app's
> available functionality and the fetchers that back them, per platform.

---

## 0. Latest Status Update (2026-08-11) — gaps closed since v2026-08-09

The broken state documented in §1–§6 below has been **resolved**. This section
supersedes the "🟥/⚪/⚠️" markers in the body for the items it covers.

### Fake crypto / mock data — ELIMINATED
- **0 actual `Math.random()` calls** remain across all client code (TS/JS/Kotlin/
  Java/Swift/Dart/Go); every remaining mention is a comment. Fabricated mnemonics,
  addresses, tx hashes, signatures, market/price data, and ordinal numbers were
  replaced with real backend calls or honest fail-closed throws / zeros.
- `user_app/react` & `user_wallet/production/react` LoginPage: fake 24-word
  mnemonic → real backend `POST /wallets` (BIP-39). `walletApi.createWallet`
  sends `{label,password,chain_id,entropy_bits}`.
- Extensions (chrome/brave/edge/firefox) biometric score → throw; shuffle/random
  → CSPRNG (`crypto.getRandomValues`). `blockchain_explorer` mock blocks → real
  JSON-RPC `eth_getBlockByNumber`. `trading_terminal` fabricated price → backend.
  `bitcoin_ordinals` simulated inscription → backend `/ordinals/inscribe`.

### All clients retargeted to canonical `go/wallet_api` (:8443) ✅
- web, desktop, ios, android, production/react, next.js wallet, flutter all target
  :8443 with the correct flat contract routes. The `:8105`/`:8080` split is gone.
- Route mismatches fixed: desktop `/wallet/balances`→`/balances`; android rewired
  off the dead `/api/v1/wallet/*` handler; production/react :8080→:8443;
  next.js proxy route `/wallet/transactions`→`/transactions`.

### Next.js wallet `lib/transactions.ts` — EVM fully wired ✅
- All 9 "unavailable" boundaries delegate to wallet_api via same-origin proxy
  routes (`/send`, `/sign`, `/transactions/:txHash`, `/swap/quote`,
  `/swap/execute`, `/gas`). Created the missing dynamic route
  `app/api/v1/transactions/[txHash]/route.ts`. Solana/Bitcoin are honest
  **fail-closed throws** (not stubs).

### `rust/userwallet_fetchers` — FIXED, builds clean ✅
- `cargo check --lib` exit 0. Has `Cargo.toml`. Delegates ALL fetchers to
  wallet_api (:8443) via a pooled async `reqwest::Client`. No
  `return Ok(default)/empty/0.0/Vec::new()` stubs; fail-closed (returns `Err`)
  for endpoints the backend doesn't expose.

### Light/dark theme — works on every page ✅
- `frontend/web_nextjs`: **0 `dark:` Tailwind variants remain** — all 5 remaining
  pages (passkey, biometric-auth, gas-tracker, app/page, login/page) converted to
  `useTheme()` + `isDark` ternaries. `npx tsc --noEmit` → 0 new errors.
- Mobile: Android `ThemeManager.kt`, iOS `ThemeManager.swift`, Flutter
  `theme_provider.dart` all present.

### docker-compose Go services build + security fix ✅
- `permission_service`, `connection_api`, `monitoring_dashboard` now `go build` +
  `go vet` clean (go.mod/go.sum generated; contexts/Dockerfiles retargeted).
- **`permission_service`: SHA-256 password hashing → bcrypt** (security fix).
- `connection_api`: fixed unused import + schema mismatch. PostgreSQL + Redis
  kept (no SQLite).

### Mobile buildability ✅
- `mobile_apps/flutter_app` + `mobile/flutter` have `pubspec.yaml` (buildable).
- `user_wallet/android` compiles (base URL = :8443; fragment/service signatures
  match).

### Full per-client fetcher parity + build verification (2026-08-12) ✅
All four UserWallet native clients (`user_wallet/web`, `user_wallet/desktop`,
`user_wallet/android`, `user_wallet/ios`) now expose the **identical fetcher
set** against `go/wallet_api` (:8443). The 2026-08-11 status retargeted the
clients to :8443, but per-client parity was incomplete; this closes it:
- `web` added `getSwapQuote` + `getStakingQuote` (send/sign already existed).
- `desktop` added `getNFTs` + `getSwapQuote` + `getStakingQuote`.
- `android` added `getTokenBalances`/`getNFTs`/`getGasPrice`/`getTokenPrice`/
  `getChains`/`getNetworkStatus`/`getSwapQuote`/`getStakingQuote` + data classes.
- `ios` added `sendTransaction`/`signMessage`/`getTokenBalances`/`getNFTs`/
  `getGasPrice`/`getTokenPrice`/`getChains`/`getNetworkStatus`/`getSwapQuote`/
  `getStakingQuote` + Codable structs.
`getNetworkStatus` is honest: derives `connected` from `/chains`, reports
`block_number = 0` (no fabricated blocks).
**Build verification (all green):** `frontend/web_nextjs` tsc → 0 errors;
`user_wallet/web` tsc → 0 errors (`--legacy-peer-deps`); `go/wallet_api`
build+tests pass (BIP-44 vector); `desktop_wallet` C++ cmake/make exit 0 +
tests pass; Foundry `forge build` exit 0, `forge test` **31/31 pass** (real
ECDSA via `vm.sign`, no mocks). OpenZeppelin v5 installed via
`forge install` (was absent from the shallow clone).

> The body of this document (below) is retained as the **historical 2026-08-09
> record** of the pre-fix state for traceability.

---

## 1. Platform Map (UserWallet apps)

| Platform | Repo location | Tech | Default backend target | Working? |
|----------|---------------|------|------------------------|----------|
| Backend (canonical) | `go/wallet_api` | Go / Gin | `:8443` | ✅ Real |
| Rust fetchers | `rust/userwallet_fetchers` | Rust | — | 🟥 Dead |
| User services | `user_services/go` | Go / GORM | `:8081` | ⚠️ CRUD only |
| Web | `user_wallet/web` | React CRA | `:8105/api/v1` | ⚠️ 1 dead route |
| Desktop | `user_wallet/desktop` | Electron | `:8105/api/v1` | 🟥 Route mismatch |
| Extension | `user_wallet/extension` | Web MV3 | `:8105` static | ⚪ Placeholder |
| Android | `user_wallet/android` | Kotlin | undefined | 🟥 Broken |
| iOS | `user_wallet/ios` | Swift | `:8105/api/v1` | ⚪ Stubs |
| Rust lib | `user_wallet/rust` | Rust | none | ⚠️ Offline |
| Production frontend | `user_wallet/production/react` | React/Vite | `:8080/api/v1` | 🟥 Orphan |
| Next.js wallet | `frontend/web_nextjs/app/wallet` | Next.js | `:8443` proxy | 🟥 9 stubs |
| Desktop app | `desktop_app` | JS + Tauri | `api.tigerwallet.com` | ✅ Real |
| Flutter | `mobile/flutter` | Dart | `api.tigerwallet.com` | ⚠️ Not buildable |
| Native shells | `mobile/{android,ios}` | Java/Swift | — | ⚪ Thin |
| Mobile apps | `mobile_apps/{android_app,ios_app,flutter_app,tigerwallet}` | 4 langs | — | ✅ (User+Master) |
| **Browser ext (prod)** | `browser_extensions/chrome` | JS | RPC + `api.tigerwallet.com` | ✅ REALEST |

---

## 2. Feature Functionality Matrix (planned vs present in actual source)

Legend: ✅ present+real · ⚠️ present but stubbed/partial · ❌ absent · 🟥 broken
(route/compile). Backend = `go/wallet_api` (real) unless noted.

| Feature | Backend | Web | Desktop | Android | iOS | Next.js wallet | chrome ext | Flutter |
|---------|:------:|:---:|:-------:|:-------:|:---:|:--------------:|:----------:|:-------:|
| Multi-chain wallet | ✅ | ✅ | 🟥 | 🟥 | ⚠️ | ⚠️ | ✅ | ✅ |
| Send / receive | ✅ | ⚠️(no bcast) | 🟥 | 🟥 | ⚠️ | 🟥 | ✅ | ✅ |
| Address book | ✅ | — | — | — | — | ✅ lib | ✅ | ✅ |
| QR code | — | — | — | — | — | ✅ | ✅ | ✅ |
| Swap / DEX | ⚠️(lib unwired) | — | 🟥 | 🟥 | — | 🟥 | ✅ | ✅ |
| Staking | — | — | — | 🟥 | — | — | ✅ | ✅ |
| Liquid staking | — | — | — | — | — | — | ❌ | ⚠️ |
| Lending | — | — | — | ✅(stub) | — | — | ✅ | ✅ |
| Bridge | — | — | — | — | — | — | ✅ | ✅ |
| NFT gallery/trade/mint | ✅(read) | — | — | 🟥 | — | — | ❌ | ✅ |
| DApp browser | — | — | — | 🟥 | — | ✅ | ✅ | ✅ |
| Prices | ✅ CoinGecko | 🟥 stub | — | — | — | — | ✅ | ✅ |
| Gas / network status | ✅ RPC | 🟥 stub | — | — | — | — | ✅ | ✅ |
| Transactions history | ✅ explorer | ✅ DB | 🟥 | 🟥 | ⚠️ | 🟥 | ✅ | ✅ |
| Crypto card | — | — | — | — | — | ✅ | ✅ | ✅ |
| Fiat on/off ramp | — | — | — | — | — | — | ✅ | ✅ |
| KYC | ✅(web) | ✅ | — | — | — | — | — | — |
| Hardware wallet | — | — | — | — | — | ✅ | ✅ | ✅ |
| MPC / Social recovery | — | — | — | — | — | ✅ | ❌ | ✅ |
| Account abstraction | — | ⠀ | — | — | — | ✅ | ❌ | ✅ |
| P2P / Copy / Margin / Futures / Options | — | — | — | — | — | ✅ UI | ⚠️ | ✅ |
| Passkey | — | — | — | — | — | — | ❌ | ⚠️ |
| Red packet / Airdrop | — | — | — | — | — | — | ✅ | ✅ |

> The **backend (`go/wallet_api`) is the single source of real data**. Every
> frontend that does not point at it shows only stubs or breaks.

---

## 3. Detailed Per-App Fetcher API (what each app can ask the backend)

### 3.1 Backend — `go/wallet_api` REST API (the REAL API set)
Base: `http://localhost:8443/api/v1`

| Method | Path | Description | Real? |
|--------|------|-------------|:-----:|
| GET | `/health` | liveness | ✅ |
| GET | `/chains` | supported chains | ✅ |
| GET | `/price/:symbol` | price from CoinGecko | ✅ |
| GET | `/gas/:chain` | gas fees (legacy + priority) | ✅ |
| POST | `/auth/register` | create user | ✅ |
| POST | `/auth/login` | JWT login | ✅ |
| GET | `/wallets` | list (protected) | ✅ |
| POST | `/wallets` | create (real HD) | ✅ |
| GET | `/wallets/:id/balance` | native balance (RPC) | ✅ |
| GET | `/wallets/:id/tokens` | ERC-20 balances | ✅ |
| GET | `/wallets/:id/transactions` | explorer history | ✅ |
| GET | `/wallets/:id/nfts` | NFTs | ✅ |
| POST | `/wallets/:id/send` | sign+broadcast (real EIP-1559) | ✅ |
| POST | `/wallets/:id/sign` | ECDSA personal_sign | ✅ |
| GET | `/public/*` | public mirrors of read endpoints | ✅ |

### 3.2 Web — `user_wallet/web`
Calls `:8105/api/v1`: `login/register/getProfile`, `wallets` (GET/POST),
`transactions` (GET/POST), `balances`, `balances/:wallet_id`, `prices/:token`,
`networks`, `network/:network/gas`, `network/:network/status`, `kyc/status`.
Pages: Login, Dashboard, Wallets, Transactions, Settings.
⚠️ `getProfile` has no backend route → 404.

### 3.3 Desktop — `user_wallet/desktop`
Calls `:8105/api/v1/wallet/...` → **mismatch, all 404**.
Pages: Dashboard, Transactions, Wallets, Login, Settings.

### 3.4 Extension — `user_wallet/extension`
No API fetchers. Theme toggle + opens `http://localhost:8105` links. Hardcoded
`$0.00`.

### 3.5 Android — `user_wallet/android`
Calls `:8105/api/v1/wallet/...`, `/wallet/swap`, `/stake`, `/nfts`,
`/portfolio`, `/history` → exist only in the **dead handler** (unwired). Does
not compile (no base URL, service/fragment signature mismatch).
Pages: Dashboard, Wallets, Transactions, Settings.

### 3.6 iOS — `user_wallet/ios`
`getBalances/getWallets/createWallet/getTransactions` — all **placeholders**,
no HTTP. Pages: Dashboard, Wallets, Transactions, Settings.

### 3.7 Rust lib — `user_wallet/rust`
No network. Local: create/import wallet, derive address, encrypt/decrypt,
sign txn (**SHA-256 fake**), chain/token registry.

### 3.8 Production frontend — `user_wallet/production/react`
Calls `:8080/api/v1` with ~23 routes that **don't exist** on any user-wallet
backend (only `/wallets` GET/POST partial). Orphan.

### 3.9 Next.js wallet — `frontend/web_nextjs/app/wallet`
Real `blockchains.ts` + `security.ts`; `transactions.ts` has **9 "unavailable"
throws** (signing, EVM/Solana/BTC broadcast, gas, receipt, swap). Only
`/api/v1/wallet/transactions` proxy exists.

### 3.10 Chrome extension (prod) — `browser_extensions/chrome`
Real RPC (balance, nonce, gas, chainId, raw broadcast) + `api.tigerwallet.com`:
staking, swap, NFT, bridge, convert, prices (+ WebSocket).

### 3.11 Desktop app — `desktop_app`
Real staking via `api.tigerwallet.com/v1`; Tauri Rust services + cold wallet.

---

## 4. Feature Backing Services Needed (backend) per repo conventions

The feature matrix implies these backend services must exist for UserWallet.
Their real fetch backing lives in `go/wallet_api` for wallet/token/price/gas/tx/
nft; everything else is expected from dedicated services (many only present as
client stubs today):

- P2P, Margin, Futures, Options, Copy-trading, Convert, Swap/DEX
- Staking, Liquid staking, Lending, Bridge, Farming, DAO
- NFT gallery/trade/mint
- Crypto card, Fiat on/off ramp, Gift cards
- DApp browser, Launchpad, Prediction markets, RWA, Security scanner,
  Gas tracker, Orderbook, TWAP, Intent routing
- Red packet, Airdrop claim
- Hardware wallet, MPC, Social recovery, Account abstraction

---

## 5. What is MISSING (gaps) — consolidated

### Data / truth gaps
1. No `user_wallet/*` frontend is wired to `go/wallet_api` (:8443).
2. `:8105` backend has **no on-chain balance, no real prices/gas/network status,
   no NFT, no broadcast send** — all stubs or DB-only.
3. `rust/userwallet_fetchers` — dead, uncompilable, all stubs; superseded by
   `go/wallet_api`.
4. `user_services/go` — real CRUD, faked chain ops (SHA-256 "broadcast", SHA-256
   "derive", hardcoded mnemonic).

### Frontend gaps
5. Desktop route mismatch (`/wallet/balances` etc.) → dead.
6. Android won't compile + targets dead-handler routes.
7. `production/react` orphaned on `:8080`.
8. Next.js wallet: 9 "unavailable" boundaries + missing create/send/swap routes.
9. `user_wallet/extension` is not a wallet at all.
10. iOS fetchers are placeholders.
11. `mobile/flutter` not buildable (no pubspec, missing imports).

### Feature gaps (not implemented anywhere reliably)
12. Options (desktop C++), Liquid staking (mobile), DApp browser (desktop),
    Bridge (desktop).
13. Launchpad, Prediction markets, RWA, Security scanner, Orderbook, TWAP,
    Intent routing, Passkey — missing on most/all mobile + extension.

---

## 6. Fix Priority

1. Single canonical backend (`go/wallet_api`) for ALL user clients.
2. Repair runtime: desktop routes, android compile+target, production/react
   host+routes, nextjs create/send/swap routes.
3. Add real broadcast + price/gas/network/nft to the served backend.
4. Remove or rewire dead code (`handlers`, `wallet_service`, `swap_service`,
   `rust/userwallet_fetchers`).
5. Make Flutter buildable; fill the extension + iOS real logic.

---

*Generated 2026-08-09 · Verified against actual source.*