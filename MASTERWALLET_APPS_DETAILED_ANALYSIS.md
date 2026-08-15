# TigerWallet MasterWallet Applications — Full Fetchers & Functionality Inventory

> Comprehensive analysis of all MasterWallet apps (web, flutter, desktop, Android,
> iOS, extensions, Rust) across every platform: their fetchers, functionality,
> what is real vs stubbed, what is missing, the database schema, and separation
> from UserWallet / Admin apps.

---

## Isolation Guarantee

The MasterWallet apps are **separate** from UserWallet and Admin apps:
- MasterWallet apps **never** call/access UserWallet fetchers or functionality.
- MasterWallet apps **never** call/access Admin fetchers or functionality.
- The MasterWallet backend services are independent and only touch
  MasterWallet-owned resources.

---

## 1. Broad Overview

| Piece | Path | Status |
|-------|------|--------|
| Most complete server | `master_wallet/go/services/main.go` (2,283 lines) | Auth + master/sub wallet + tx + fees + analytics + audit, WS |
| Thin GORM server | `master_wallet/go/cmd/main.go` (port 8080) | Master-wallet CRUD + stubs |
| Thin variant | `master_wallet/go/cmd/master_wallet_service/main.go` | Duplicate of above |
| External treasury service | `go/services/master_wallet_service/main.go` | Treasury APIs — **entirely stubbed** |
| Web (React) | `master_wallet/web` | Local in-memory HD wallet + live RPC |
| Flutter | `master_wallet/flutter` | Mirror of web HD wallet + feature REST fetchers |
| Desktop (C++/Qt) | `master_wallet/desktop` | **Balance/tx stubs** |
| Android (Kotlin) | `master_wallet/android` | ETH balance REAL (web3j), token STUB |
| iOS (Swift) | `master_wallet/ios` | Balance **stub** |
| Extensions (chrome/firefox/brave/edge) | `master_wallet/extensions` | Simulated handlers |
| Rust | `master_wallet/rust` | In-memory lib, **no network** |
| DB schema | `master_wallet/database/schema.sql` | **Real & complete** (17 tables) |

---

## 2. Backend Services

### 2A. `go/services/main.go` (2,283 lines) — most complete

**Public auth (`/api/v1/auth`, main.go:304-311):**
`POST /auth/register` (412), `POST /auth/login` (458), `POST /auth/logout` (521),
`POST /auth/refresh` (526).

**Master wallet (`/api/v1/master-wallet`, protected, main.go:320-329):**
- `GET` (list) `GetMasterWallets` (553)
- `POST` (create) `CreateMasterWallet` (579)
- `GET /:id` `GetMasterWallet` (629)
- `PUT /:id` `UpdateMasterWallet` (647)
- `DELETE /:id` `DeleteMasterWallet` (673)
- `GET /:id/balance` `GetMasterWalletBalance` (685) — **STUB** hardcoded
  `"balance":"0"`, `"tokens":[]` (*"In production, would query blockchain"*)
- `POST /:id/sign` `SignTransaction` (696) — **REAL broadcast** via the chain's
  RPC node (resolves `address, chain_id` from `master_wallets`)

**Sub wallet (`/api/v1/sub-wallet`, main.go:332-342):**
- `GET /:id/balance` `GetSubWalletBalance` (864) — **STUB** hardcoded `"0"`
- `POST /:id/transfer` `TransferFromSubWallet` (873) — **REAL broadcast** via RPC

**Transactions (`/api/v1/transactions`, main.go:344-353):** `GET`, `GET /:id`,
`POST`, `POST /:id/approve`, `POST /:id/reject`.

**Auto-sign (`/api/v1/auto-sign`, main.go:354-361):** full CRUD.
**Fees (`/api/v1/fees`, main.go:363-370):** full CRUD.
**Policies, Users, Analytics, Audit (main.go:371-405):** full CRUD + `GET
/analytics/volume`, `/transactions`, `/wallets`, audit logs, WebSocket
(`HandleWebSocket` :1626).

**Fetcher summary (2A):** balances/token fetchers are **STUBS**; transaction
broadcasts (sign/transfer) are **REAL** RPC.

### 2B. `go/cmd/main.go` (port 8080) — thin GORM version
Routes (`SetupRoutes`, :469-489): `POST /api/v1/master-wallet/generate`,
`GET /api/v1/master-wallet/addresses`,
`GET /api/v1/master-wallet/balance/:chainID` → `GetBalance` (:505) — **STUB**
(*"In production, query actual blockchain"*, always `"0"`), `POST /transaction`,
`GET /transactions`, `GET /transaction/:id`, `POST /fees`, `GET /fees`,
`POST /blockchains`, `GET /blockchains`.

### 2C. `go/cmd/master_wallet_service/main.go`
Same `/api/v1/master-wallet` group, GORM-backed, near-identical to 2B.

### 2D. External `go/services/master_wallet_service/main.go` — treasury (ENTIRELY STUBBED)
- `GET /treasury/overview`, `/treasury/balances`, `/treasury/transactions`, `/treasury/report`
- `POST /treasury/allocations` (+ get/update/delete), `POST /treasury/transfer`, `POST /treasury/sweep`
- ⚠️ All return hardcoded zeros / empty arrays / canned `gin.H{"status":"completed"}` —
  **no RPC, no broadcast**.

### Top-level root files (`master_wallet/` root)
`master_wallet_service.go` (in-memory), `hd_wallet_service.go`,
`custom_branding_service.go`, `admin_api_service.go`, `trading_admin_service.go`,
`blockchain_registry.go`, `token_registry.go`, `tiger_wallet_service.go` —
mostly in-memory/config with hardcoded blockchain/token lists.

---

## 3. Per-Platform Frontend Fetchers

### 3A. Web (`master_wallet/web`, React/TS) — `API_BASE_URL = https://api.tigerwallet.io/master`
- `web/src/services/masterWalletService.ts` — **LOCAL in-memory HD wallet** (ethers):
  - `generateWallet` / `importWallet` — real BIP-39 derivation
  - `getBalance(walletId, chainId)` (:152) — **REAL live RPC** via
    `ethers.JsonRpcProvider(chainConfig.rpcUrl)` → `provider.getBalance()` (:172)
  - `getTokenBalance` (:189), `sendTransaction` (:211) — real RPC
  - Chain configs (:5-11) real public RPC URLs (`eth.llamarpc.com`,
    `polygon-rpc.com`, etc.); wallet store is in-memory `Map` (not persisted).
- `web/src/api.ts` (`MasterWalletAPI`) — REST client to `go/services/main.go`.
- `web/src/App.tsx` — UI: Dashboard, Users, Settings (mock stats at :88).

### 3B. Flutter — `API_BASE = https://master-api.tigerwallet.com/api/v1`
- `lib/services/master_wallet_service.dart` — **mirror** of the web HD wallet
  (bip39/bip32, in-memory `_wallets` map; live RPC via `Web3Client`).
- Feature services (backend REST, all → `master-api.tigerwallet.com/api/v1`):
  `treasury/treasury_service.dart`, `policy/policy_service.dart`,
  `multisig/multisig_service.dart`, `batch_tx/batch_transaction_service.dart`,
  `audit/audit_service.dart`.

### 3C. Desktop (`desktop/`, C++/Qt)
`src/services/master_wallet_service.cpp`:
- `getBalance` (:260) — **STUB** mock `"0"` from in-memory `balanceCache_`
  (*"In production, query RPC"*); TTL cache per wallet/chain.
- `getAllBalances` (:293) — loops chains through stub `getBalance`.
- `createTransaction` (:304) — **STUB**: fabricates `0x...` hash via `RAND_bytes`.
- `signAndBroadcast` (:328) — delegates to stub `createTransaction`.
- `estimateGas` (`paymaster_service.cpp:443`) — **STUB** (*"In production, fetch
  from multiple RPC endpoints"*; simulated prices).
- Other services: account_abstraction, passkey (RAND_bytes), paymaster, privacy,
  super_admin, tax_analytics, web_socket.
- UI: Dashboard, Wallets, Settings (mock stats :88).

### 3D. Android (`android/.../com/tigermaster/`)
`MasterWalletService.kt`: **ETH balance REAL RPC** via web3j
`ethGetBalance(...)` on `Dispatchers.IO` (:208-214); **Token balance STUB**
(`"0"`, *"In production, call token contract"*, :224-233).
`AccountAbstractionService.kt` — smart-account address placeholder `0x` +
`"0".repeat(40)` (:432), `"0"` default balance (:214).
`MasterWalletViewModel.kt:20` → `api.tigerwallet.io/master`;
`MasterWalletApiService.kt` (OkHttp) CRUD client complete.

### 3E. iOS (`ios/.../Sources/Services/`)
`MasterAPIService.swift` → `api.tigerwallet.io/master` (matches `go/services/main.go`
routes: auth, `/api/v1/master`, approve/reject/tx).
`MasterWalletService.swift` — balance fetcher comments *"In production, make
actual…"* → **STUB**.

### 3F. Extensions (chrome / firefox / brave / edge) — four virtual copies
`*/services/masterWalletService.js` — `API_BASE = https://master-api.tigerwallet.com/api/v1` (:6).
`background.js` handlers **mostly simulated** (activate/approve/rejectTransaction,
createSubWallet return canned in-memory results).

### 3G. Rust (`rust/src/lib.rs`) — **pure in-memory, zero network/RPC**
Fetchers are local-only: `create_master_wallet(seed_phrase)` (:110),
`derive_address` (:124), `set_fees` (:210), user management
(add/register/get/suspend/block/activate, :225-284), blockchain/token
add/remove (:294-312), `sign_user_transaction` (:330), `transfer_from_user`
(:347), `get_master_info` (:384). **No on-chain balance fetcher.**

---

## 4. Database Schema (`master_wallet/database/schema.sql`) — ✅ Real & Complete

17 tables, treasury-grade:
`master_wallets` (46), `sub_wallets` (90), `signers` (136), `transactions` (185),
`transaction_signatures` (248), `approval_requests` (284), `wallet_users` (316),
`whitelist` (357), `policies` (391), `audit_logs` (420), `fee_config` (463),
`token_balances` (503), `notifications` (537), `api_keys` (573), `webhooks` (611),
`webhook_delivery_logs` (650), `sessions` (680) + 25 indexes.

⚠️ Most backend services auto-migrate only a subset and don't wire the full schema;
the external `master_wallet_service` auto-migrates
MasterWallet/FeeConfig/BlockchainConfig/SupportedToken/RevenueRecord/WhiteLabel.

---

## 5. Real vs Stub Matrix

| Fetcher | `go/services` | external `master_wallet_service` | Web | Flutter | Android | iOS | Desktop cpp | Rust |
|---------|---------------|----------------------------------|-----|---------|---------|-----|-------------|------|
| Native ETH balance | **STUB** `"0"` | ❌ (SQL only) | **REAL RPC** | **REAL RPC** | **REAL** (web3j) | **STUB** | **STUB** `"0"` | ❌ |
| Token balance | ❌ | ❌ | REAL RPC | REAL RPC | **STUB** | STUB | STUB | ❌ |
| Tx broadcast / sign | **REAL RPC** | ❌ | REAL RPC | REAL RPC | partial | STUB | **STUB** (`RAND_bytes`) | signs locally |
| Gas / estimate | ❌ | `CalculateFee` (SQL) | — | — | ❌ | ❌ | **STUB** simulated | fee config |
| Treasury balances | ❌ | **STUB** (empty/0) | — | — | — | — | — | ❌ |
| Treasury transfer / sweep | ❌ | **STUB** (canned) | — | — | — | — | — | ❌ |
| Revenue stats / report | ✅ `/analytics/*` (SQL) | SQL `GetRevenueStats` | — | — | — | — | — | ❌ |

---

## 6. Key Findings & Gaps

1. **No treasury/custody fetcher is functional.** The richest treasury API
   (`go/services/master_wallet_service/main.go`: overview/balances/transactions/
   allocations/transfer/sweep/report) is **entirely stubbed** — returns
   zeros/empty JSON, never touches RPC or the 17-table schema.
2. **No backend balance fetcher ever queries a chain** — every Go `/balance`,
   `/:id/balance`, `/getBalance` returns `"0"`. Only client-side (web/flutter
   local, Android ETH) hit RPC directly.
3. **No gas-price fetcher** — neither backend implements a real `eth_gasPrice`
   fetch; desktop `paymaster_service` simulates prices; `GetGasPrice` stubs return
   hardcoded static values.
4. **No fee-per-chain RPC integration** — fees are SQL-config (`fee_config`) or
   hardcoded defaults only.
5. **No custody/withdrawal broadcast in the external treasury service** —
   `TreasuryTransfer` / `SweepToCold` fabricate results; only `go/services/main.go`
   has real broadcast logic.
6. **Three competing Go backends** with overlapping routes and **no canonical
   wiring** — frontends point at `master-api.tigerwallet.com/api/v1`
   (chrome/flutter) or `api.tigerwallet.io/master` (web/android/ios), neither of
   which corresponds to a single local server's route set → **no frontend is
   guaranteed to reach a real backend today**.
7. **No persistence in client HD wallets** — web/flutter wallets live in in-memory
   `Map`s and are lost on restart.

---

## 7. Recommended Fixes (priority)

1. **Implement chain-facing balance & treasury fetchers** in the Go backends
   (currently all return `"0"`/empty), backed by the 17-table schema.
2. **Wire the external treasury service** (`master_wallet_service/main.go`) to real
   RPC broadcast for `TreasuryTransfer` / `SweepToCold` / allocations.
3. **Pick one canonical backend** (recommend `go/services/main.go`) and point all 9
   clients at it, reconciling BASE_URLs.
4. **Persist client HD wallets** (web/flutter) to the backend instead of in-memory.
5. **Remove duplicate thin GORM servers** (2B/2C) that add no functionality.