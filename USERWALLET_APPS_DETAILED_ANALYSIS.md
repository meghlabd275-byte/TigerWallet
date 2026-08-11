# TigerWallet UserWallet Applications — Full Fetchers & Functionality Inventory

> Comprehensive analysis of all UserWallet apps (web, desktop, extension, Android,
> iOS, Rust) across every platform: their fetchers, functionality, what is real vs
> stubbed, what is missing, and separation from MasterWallet / Admin apps.

> **⚠️ STATUS UPDATE (2026-08-11):** The "broken/stub/orphan" state documented in
> the body of this file (the `:8105`/`:8080` split, dead `user_wallet_handler.go`,
> desktop route mismatch, Android not compiling, iOS placeholders, Rust fetchers
> dead, Math.random fake crypto) has been **resolved**. Current verified state:
> - All `user_wallet/*` clients target the canonical `go/wallet_api` (:8443) with
>   correct routes. No `:8105`/`:8080` split remains.
> - 0 actual `Math.random()` calls in any client (fake mnemonics/hashes/sigs/data
>   replaced with real backend calls or fail-closed throws).
> - `rust/userwallet_fetchers` builds clean and delegates all fetchers to wallet_api
>   (no stubs; fail-closed Err for absent endpoints).
> - `frontend/web_nextjs/app/wallet/lib/transactions.ts` EVM path fully wired via
>   proxy routes; dynamic `/api/v1/transactions/[txHash]` route created.
> - Light/dark theme: 0 `dark:` variants in web_nextjs; mobile has theme managers.
> - `permission_service`/`connection_api`/`monitoring_dashboard` build + vet clean;
>   permission_service uses bcrypt (was SHA-256). PostgreSQL + Redis (no SQLite).
> The body below is retained as the historical pre-fix record.

---

## Isolation Guarantee

The UserWallet apps are **separate** from MasterWallet and Admin apps:
- UserWallet apps **never** call/access MasterWallet fetchers or functionality.
- UserWallet apps **never** call/access Admin fetchers or functionality.
- Verified in analysis: no UserWallet component references `master_wallet` or
  `admin` fetchers. Each app family has its own backend and clients.

---

## 1. Broad Overview

| Piece | Path | Port | Status |
|-------|------|------|--------|
| Live user-wallet backend (CRUD + stubs) | `user_wallet/go/cmd/main.go` | **8105** | Served, DB/Redis + several stubs |
| Dead hardcoded backend | `user_wallet/go/handlers/user_wallet_handler.go` | — | **Not wired** (never served) |
| **Canonical REAL wallet backend** | `go/wallet_api` | **8443** | ✅ Real (RPC + BIP-32 + CoinGecko + Etherscan) |
| Legacy wallet service | `go/wallet_service` | 8001 | Mongo, separate |
| Web client (React) | `user_wallet/web` | — | Hits `:8105/api/v1` |
| Desktop (Electron) | `user_wallet/desktop` | — | **Route mismatch** (dead endpoints) |
| Extension | `user_wallet/extension` | — | Theme toggle only, no fetchers |
| Android (Kotlin) | `user_wallet/android` | — | Hits dead handler routes |
| iOS (Swift) | `user_wallet/ios` | — | Hits `:8105/api/v1` |
| Rust lib | `user_wallet/rust` | — | Local HD only, no network |
| Next.js wallet | `frontend/web_nextjs/app/wallet` | — | **Stubbed + route mismatch** |
| Production browser extension | `browser_extensions/chrome` | — | ✅ Real RPC + `api.tigerwallet.com` |

---

## 2. UserWallet Backend Fetchers (`user_wallet/go/cmd/main.go`, port 8105)

This is the **live-served** backend. Fetchers (all at `cmd/main.go:939-965`):

| Method | Path | Handler | Data source / status |
|--------|------|---------|----------------------|
| GET | `/health` | healthCheck | live |
| POST | `/api/v1/auth/register` | register | Postgres |
| POST | `/api/v1/auth/login` | login | Postgres + Redis |
| GET | `/api/v1/wallets` | getWallets | Postgres |
| POST | `/api/v1/wallets` | createWallet (760) | Postgres, **address = MOCK** (`cmd/main.go:779`) |
| GET | `/api/v1/transactions` | getTransactions (327) | Postgres |
| POST | `/api/v1/transactions` | createTransaction (709) | **DB insert only — no broadcast** (`main.go:748-755`) |
| GET | `/api/v1/balances` | getAllBalances (304) | Postgres + Redis 30s |
| GET | `/api/v1/balances/:wallet_id` | getBalance (280) | Postgres + Redis (**no RPC**) |
| GET | `/api/v1/prices/:token` | getTokenPrice (364) | **STUB** — "live token price provider is not configured" |
| GET | `/api/v1/networks` | getNetworks (235) | Postgres (seeded mainnet/testnet) |
| GET | `/api/v1/network/:network/status` | getNetworkStatus (376) | **STUB** — Redis only |
| GET | `/api/v1/network/:network/gas` | getGasPrice (388) | **STUB** — Redis only |
| GET | `/api/v1/tokens` | getTokens (251) | Postgres (seeded) |
| GET | `/api/v1/kyc/status` | getKYCStatus (401) | Postgres |

**Fetchers actually served:** balance, balances, transactions, tokens, networks,
KYC status — all from **seeded Postgres + Redis**, no on-chain RPC.
**Missing here:** NFTs, swap, stake, send/sign broadcast, portfolio, bridge,
gas/prices/network-status are all **stubs or absent**.

### Dead handler — `user_wallet/go/handlers/user_wallet_handler.go`
**Not wired** (zero references to `NewUserWalletHandler`). All hardcoded inline = **STUB**:
- `POST /api/v1/wallet` create (mock address), `GET /api/v1/wallet` list (2 hardcoded),
  `GET /api/v1/wallet/:id`, `GET /:id/balance` (ETH/USDT/WBTC hardcoded),
  `POST /:id/send` (fake `tx_hash`), `GET /:id/transactions`,
  `POST /wallet/swap` (hardcoded 0.95), `GET /wallet/swap/quote`,
  `POST /wallet/stake` (5% reward), `POST /wallet/unstake`, `GET /wallet/stakes`,
  `GET /wallet/nfts` (1 NFT hardcoded), `POST /wallet/nft/transfer` (fake hash),
  `GET /wallet/portfolio`, `GET /wallet/history`.

### ✅ The REAL canonical backend — `go/wallet_api` (port 8443)
Per repo memory this is the only service performing real key management + signing:
- Routes (`main.go:57-89`): `/health`, `/api/v1/chains`, `/api/v1/price`,
  `/api/v1/gas`, `/auth/register`, `/auth/login`, then wallet group
  `POST /wallets`, `GET /wallets`, `GET /balance`, `GET /tokens`,
  `GET /transactions`, `GET /nfts`, `POST /send`, `POST /sign`, plus public mirrors.
- Real fetchers (`fetchers.go`): `FetchNativeBalance` (eth_getBalance, L78),
  `FetchTransactionCount` (L93), `FetchGasPrice` (eth_gasPrice + priority, L108),
  `FetchChainID` (L135), ERC-20 `FetchERC20Balance` (L183) / `FetchERC20Metadata`
  (L192) / `FetchTokenBalances` (L231), CoinGecko `FetchTokenPrice` (L299) /
  `FetchETHPrice` (L324), explorer `FetchTransactionHistory` (L333).
- Real key management: `hd_derive.go` (BIP-39/BIP-32/BIP-44), `wallet_engine.go`.
- ⚠️ **BUT none of the `user_wallet/` frontends point at this backend** — they all
  use `:8105`.

### Legacy — `go/wallet_service` (port 8001)
register/login, `POST /wallets`, `POST /wallets/import`, `GET /users/:id/wallets`,
`GET /wallets/:id` (Balance), `GET /wallets/:id/transactions`, `POST /send`,
`POST /swap`, `POST /stake`, `GET /chains`. Uses **Mongo + Redis**. Not referenced
by any `user_wallet` frontend (defaults `:8001`).

---

## 3. Per-Platform Frontend Fetchers

### 3a. Web (`user_wallet/web`, React CRA) — targets `:8105/api/v1`
Fetchers (`web/src/services/api.ts`): `login/register/getProfile` (L24-38),
`getWallets/createWallet` (L43-53), `getTransactions/createTransaction` (L56-71),
`getBalances/getBalance` (L74-86), `getTokenPrice` (L89), `getNetworks` (L93),
`getGasPrice` (L97), `getNetworkStatus` (L101), `getKYCStatus` (L105).
UI pages: **Dashboard** (calls `getBalances`, Dashboard.tsx:14), **Transactions**
(Trans.tsx:15), **Wallets** (createWallet L27, getWallets L18), **Login, Settings**.
→ These hit the live stub backend (correct paths).

### 3b. Desktop (`user_wallet/desktop`, Electron) — targets `:8105/api/v1/wallet/*`
- `desktop/src/pages/Dashboard.jsx:3,10` fetch `${API_URL}/wallet/balances`
- `Transactions.jsx:3,10` fetch `/wallet/transactions`
⚠️ **ROUTE MISMATCH**: main.go serves `/api/v1/balances` & `/api/v1/transactions` —
there is **NO `/api/v1/wallet/balances` route**. Desktop hits dead endpoints.
UI: Dashboard, Transactions, Wallets, Login, Settings.

### 3c. Extension (`user_wallet/extension`)
**Not a crypto wallet** — theme toggle + links only. toggles theme
(`src/popup.js:4-8`), opens `http://localhost:8105` (L14), `/wallets` (L18),
`/transactions` (L22). **No fetchers.**

### 3d. Android (`user_wallet/android`, Kotlin) — targets `:8105/api/v1/wallet/*`
`UserWalletApiService.kt` fetchers (match the **DEAD handler**, not main.go):
`POST /wallet` create (L54), `GET /wallet` list (L68), `GET /wallet/:id` (L82),
`GET /:id/balance` (L96), `POST /:id/send` (L149), `GET /:id/transactions` (L163),
`GET /transactions/:id` (L177), `POST /wallet/swap` (L200),
`GET /wallet/swap/quote` (L225), `POST /wallet/stake` (L257),
`POST /wallet/unstake` (L283), `GET /wallet/stakes` (L307),
`GET /wallet/nfts` (L336), `POST /wallet/nft/transfer` (L367),
`GET /wallet/portfolio` (L383), `GET /wallet/history` (L417).
UI: `DashboardFragment`, `WalletsFragment`, `TransactionsFragment`, `SettingsFragment`.
These hit **hardcoded stubs** if/when reached.

### 3e. iOS (`user_wallet/ios`, Swift) — targets `:8105/api/v1`
`UserWalletApiService.swift:4` fetchers: `getBalances`, `getWallets`,
`createWallet`, `getTransactions`. Views: `DashboardView` (loadBalances),
`WalletsView`, `TransactionsView` (loadTransactions), `SettingsView`.
→ Correctly target main.go backend.

### 3f. Rust (`user_wallet/rust`) — local HD library, **no network fetchers**
`rust/src/lib.rs`: chain registry (L35-62), `create_wallet` (L253),
`import_wallet` (L285), `get_address` (L390), `get_blockchains` (L395),
`get_tokens` (L400), `sign_transaction` (L417). Real local BIP derivation, **zero HTTP/DB**.

---

## 4. Next.js Wallet (`frontend/web_nextjs/app/wallet`)

- **Feature claims** in `app/wallet/page.tsx`: 24-word seed generation (BIP-39 via
  `@scure/bip39`), send / swap / multi-sig UI.
- `app/wallet/lib/transactions.ts` — **STUB**: key derivation, signing,
  `broadcastTransaction`, gas, swap all **throw "unavailable until […] configured"**.
- API routes present (`app/api/v1/`): only **`transactions`** exists + a **`price`**
  route (proxies to `/price`). **NO `/api/wallet/create | send | swap` routes exist**
  — yet `page.tsx` calls them → the documented Create/Send/Swap flow would **404**.
- `app/master_wallet/page.tsx` is a **separate** MasterWallet (own theme context).

---

## 5. Production Browser Extension (`browser_extensions/chrome`) — 🟢 REALEST

- `wallet/wallet.js` — **"PRODUCTION-READY — NO STUBS"**. Real HD derivation
  (m/44'/60'/0'/0/0, L272), real `eth_getBalance` (L405), nonce (L435), gas (L450),
  chainId (L465), broadcast raw signed tx (L497,507) against real RPC:
  `eth.llamarpc.com`, `bsc-dataseed.binance.org`, `polygon-rpc.com` (L387-396).
- `wallet/stakingModule.js` — external `https://api.tigerwallet.com/v1/staking/*`
  (chains, validators, stake/unstake/claim, positions).
- `wallet/swap-nft-staking-bridge.js` — external `api.tigerwallet.com/v1/swap`
  (tokens/quote/execute), `/v1/nft` (collections/owners/listings), bridge.
- `services/price-service.js` — external `api.tigerwallet.com/v1/prices` +
  WebSocket `wss://api.tigerwallet.com/ws/prices`.
- `services/convert-service.js` — external `/v1/convert` (quote/execute/tokens/history).

---

## 6. Real vs Stub Matrix

| Fetcher | `user_wallet/go` main | dead handler | `wallet_api` (REAL) | web | desktop | android | ios | chrome ext |
|---------|----------------------|--------------|---------------------|-----|---------|---------|-----|------------|
| Balance | DB-backed stub | STUB | ✅ REAL RPC | DB | dead route | STUB | DB | ✅ RPC |
| Token balances | DB | STUB | ✅ REAL | DB | dead | STUB | — | ✅ RPC |
| Transactions | DB insert (no broadcast) | STUB | ✅ REAL explorer | DB | dead | STUB | DB | ✅ RPC |
| NFTs | — | STUB | ✅ REAL | — | — | STUB | — | ✅ ext API |
| Prices | STUB | — | ✅ CoinGecko | STUB | — | — | — | ✅ ext API |
| Gas | STUB | — | ✅ REAL | STUB | — | — | — | ✅ RPC |
| Send / broadcast | DB-only | STUB fake hash | ✅ REAL | DB-only | dead | STUB | — | ✅ REAL |
| Sign | — | — | ✅ REAL | — | — | — | — | dapp |
| Swap | — | STUB 0.95 | — | — | dead | STUB | — | ✅ ext API |
| Stake | — | STUB | — | — | — | STUB | — | ✅ ext API |
| Portfolio | — | STUB | — | — | — | STUB | — | — |
| KYC / Networks | DB | — | — | DB | — | — | — | — |

---

## 7. Key Findings & Gaps

1. **The genuinely real UserWallet backend is `go/wallet_api` (8443)** — but **no
   frontend in `user_wallet/` actually uses it**. All user-facing apps point at
   **`:8105`** (`user_wallet/go/cmd/main.go`), which is DB-CRUD + stubs.
2. **3 frontends hit non-existent routes:** desktop (`/wallet/balances`), android
   (`/api/v1/wallet/*` → only exists in the dead handler), web_nextjs
   (`/api/wallet/create|send|swap` — no such routes). Only `web` + `ios` correctly
   hit main.go routes.
3. **The only fully REAL frontends** are **`browser_extensions/chrome`** (direct
   RPC + `api.tigerwallet.com`) and **`user_wallet/production/react`** (ethers /
   Solana + `:8080/api/v1`, though `:8080` is claimed by multiple unrelated Go
   services, and its routes `/wallets/...`, `/chains/...`, `/swap`, `/stake` do
   **not** match `rpc_service`'s `/rpc/:chain`).
4. **No cross-contamination:** no UserWallet component references MasterWallet or
   Admin fetchers (separation respected), but the whole user-wallet data chain is
   largely **stub/mock or pointed at absent/remote endpoints**.

---

## 8. Recommended Fixes (priority)

1. **Point all `user_wallet` frontends at `go/wallet_api` (8443)** — the canonical
   real backend — instead of the stub `:8105` server.
2. **Fix route mismatches:** desktop `/wallet/balances`, android `/api/v1/wallet/*`,
   web_nextjs `/api/wallet/create|send|swap`.
3. **Implement real fetchers in `user_wallet/go/cmd/main.go`** for prices, gas,
   network status, NFTs, swap, stake, and on-chain send/sign broadcast (or proxy to
   `go/wallet_api`).
4. **Wire or delete the dead `user_wallet_handler.go`** to remove the trap for the
   Android client.
5. **Reconcile BASE_URL** so every client (web `:8105`, desktop, android, ios,
   next.js, production/react) points at one canonical backend.