# TigerWallet — Missing Pieces & Implementation Gaps (Detailed)

> Companion document to `PROJECT_PARTY.md`, `BOTS_CLIENTS.md`, and
> `LIQUIDITY_TRADING_PAIRS.md`.
>
> This file documents **what is missing and what the gaps are** in the current
> implementation, verified by direct source inspection. It is grouped by product
> area and by cross-cutting concern, with **severity**, **evidence**, and **what
> needs to be built** for each item.

---

## ⚠️ Summary of Critical Gaps (Top 10)

| # | Gap | Location | Severity |
|---|-----|----------|----------|
| 1 | `project_party` API is **100% stubs** — no DB writes | `project_party/go/cmd/main.go` | 🔴 Critical |
| 2 | `super_admin` Go API is **100% stubs** — no DB writes | `super_admin/go/main.go` | 🔴 Critical |
| 3 | `master_admin_management` Go API is **100% stubs** — no DB writes | `master_admin_management/go/main.go` | 🔴 Critical |
| 4 | `mm_bot_platform` bot API is **in-memory demo only** | `mm_bot_platform/bot_api/bot_api_server.go` | 🔴 Critical |
| 5 | No **admin panel API** in `docker-compose.yml` + no Dockerfile for `admin/` | `docker-compose.yml`, `admin/` | 🔴 Critical |
| 6 | **Port mismatches** between web UIs and backend services | multiple | 🟠 High |
| 7 | **JSON field mismatch** (camelCase vs snake_case) breaks pair/liquidity create & update | `admin/web`, `admin/go` | 🟠 High |
| 8 | White-label service has **DB tables but no routes/methods** for liquidity, MM bots, tokens | `white_label_service` | 🟠 High |
| 9 | `importTradingPairs` / `importLiquidity` **routes are stubs** (return fake counts) | `white_label/go/main*.go`, `super_admin` | 🟠 High |
| 10 | Frontends hardcode `localhost` URLs; no auth wiring in many admin UIs | all web apps | 🟡 Medium |

---

## 1. ProjectParty Missing Pieces

### 1.1 🔴 The entire backend API has no persistence
- **File:** `project_party/go/cmd/main.go`
- **Evidence:** A `pgxpool` is created and pinged (`initDatabase`, line ~874), and every
  handler receives `db *pgxpool.Pool` as a parameter — but **zero** `db.Exec`,
  `db.Query`, or `db.QueryRow` calls exist in the file. All handlers build in-memory
  structs / maps and `c.JSON` them.
- **Impact:** Nothing created (tokens, listings, launchpads, orders, KYC, audits, fees)
  is ever saved. Listing, market-making, and pricing would lose all data on restart.
- **What's missing:**
  - Actual SQL INSERT/UPDATE/SELECT/DELETE for every table (tokens, listgs, launchpads,
    contributions, maker_orders, prices, audits, KYC).
  - Automigrations or a schema/DDL for ProjectParty tables.
  - Atomic contributions + claims + refunds (with transaction/rollback).
  - Auth middleware (the handlers take no auth; `tenant_id` is merely a JSON field).
  - Error mapping (`gorm.ErrRecordNotFound` → 404, etc.).

### 1.2 ❌ No admin review workflow implemented
- Approval/reject/submit endpoints exist but only return static messages — there is no
  reviewer assignment, review queue, or `rejection_reason` persistence.

### 1.3 ❌ Fees & payments are stubs
- `calculateFeesHandler` computes a total but nothing is persisted; `payFeesHandler`
  returns a fake `transaction_id`. No real payment integration (fiat/crypto), no
  payment status tracking, no invoice records.

### 1.4 ❌ No real market-making / liquidity engine
- `createMakerOrdersHandler`, `addLiquidityHandler`, `removeLiquidityHandler` return
  hardcoded values. ProjectParty does not actually place orders or interact with a DEX
  router / on-chain liquidity. It is only an API surface that should delegate to a real
  engine (none is wired).

### 1.5 ❌ Port/URL config drift
- **Frontend:** `project_party/web/src/services/api.ts` → `http://localhost:8106/api/v1`.
- **Backend (code default):** `PROJECT_PARTY_PORT` default `9006`.
- **Deployment compose:** `deployments/project_party/docker/docker-compose.yml` exposes
  `8004` and sets `REACT_APP_API_URL=http://project-party-backend:8004`.
- **Impact:** the web frontend cannot reach the backend as configured out of the box.
- **Fix:** single source of truth for the port (env), and align all three.

### 1.6 ❌ ProjectParty not in main `docker-compose.yml`
- `deployments/project_party/docker/docker-compose.yml` exists, but the top-level
  `docker-compose.yml` does not include a ProjectParty service.

---

## 2. BotsClients & Bots Platform Gaps

### 2.1 🔴 Bot API server is an in-memory demo
- **File:** `mm_bot_platform/bot_api/bot_api_server.go`
- **Evidence:** comment `// DATABASE (In-Memory for Demo)` (line ~199). All state is held
  in Go maps: users, botTiers, botInstances, botSubscriptions, feeConfigs,
  adminFeeAddresses, userCEXConnections, userDEXConnections, apiKeys, sessions.
- **Impact:** no persistence, no horizontal scaling, data lost on restart; CEX/DEX API
  credentials are stored in a comment-noted insecure placeholder ("in production, store
  in secure DB").
- **What's missing:** PostgreSQL/SQLite backing, secret storage (KMS/Vault), durable
  bot/instance/subscription state.

### 2.2 ❌ Bot engine execution not connected to API
- The Rust `bot_core` (bot_types.rs, strategies/mod.rs) implements strategies and a
  `BotManager`, but the Go `bot_api_server` does **not** call it. There is no bridge
  between the API server and the actual Rust strategy engine or the Solidity contracts.
- Fix: a gRPC/REST integration between `bot_api` ↔ `bot_core` ↔ the on-chain admin
  contract.

### 2.3 ❌ On-chain admin contracts are not deployed/wired
- `TigerBotPlatform.sol` and `TigerBotStrategies.sol` exist but there is **no deployment
  script, no addresses config, and no off-chain client** that invokes them. No
  subscription payment flow is wired to them.

### 2.4 ❌ Super Admin BotsClients handlers are stubs
- `super_admin/go/main.go` `handleGetBotsClients` / `handleCreateBotsClient` /
  `handleUpdateBotsClientStatus` etc. all return empty responses. No `bots_clients`
  table or CRUD persistence exists in `super_admin`.

### 2.5 ❌ No admin UI pages for BotsClients / ProjectParty
- The Super Admin web (`super_admin/web/src/pages/`) has `TradingPairs.tsx`, `Tokens.tsx`,
  etc., but **no** `BotsClients.tsx` or `ProjectParty.tsx` page, even though the API routes
  and `WhiteLevelProduct` enums define them.

### 2.6 ❌ Bot tier enforcement is partial
- `handleCreateBot` checks `user.MaxBots` but concurrency/per-user enforcement of DEX/CEX
  counts and latency tiers is not actually measured/guaranteed in the API server.

---

## 3. Liquidity & Trading-Pair Management Gaps

### 3.1 🟠 Pair create / update JSON mismatch (broken in current UI/API)
- **Frontend** (`admin/web/src/services/api.ts` `createPair`) sends:
  `{ baseToken, quoteToken, minTradeAmount, maxTradeAmount, makerFee, takerFee }`
  (camelCase, no `pair_name`, no `chain`).
- **Backend** (`admin/go/internal/handlers/pair_handler.go` `CreatePair`) **requires**:
  `pair_name`, `base_token`, `quote_token`, `chain` (snake_case, `binding:"required"`),
  plus float fields.
- **Impact:** `createPair` from the admin UI will fail validation (missing required
  `pair_name`/`chain`, and it sends strings for float fields). `updatePairStatus`
  frontend sends a boolean (`status: boolean`) but backend expects a string
  `"active"|"suspended"|"halted"`.
- **Fix:** align the JSON contract (rename to snake_case, add `pair_name` + `chain`,
  type the amounts as numbers), or add a camelCase binding.

### 3.2 ❌ `/pairs/import` endpoint missing on the admin backend
- `admin/web` calls `POST /api/v1/pairs/import` (`importPairs`), but `admin/go/main.go`
  does **not** register any `/pairs/import` route (and `PairHandler` has no import
  method). The admin text references it, but the backend doesn't implement it.

### 3.3 ❌ Pair stats vs model mismatch
- `GetPairStats` filters by `status = 'active'|'suspended'|'halted'`, but the
  `TradingPair` creation sets `is_active` and default `status='active'`. The status-set
  path (`UpdatePairStatus`) supports the three string values, but `CreatePair` has no
  `status` field. Minor but worth aligning.

### 3.4 🟠 Liquidity import is a stub everywhere
- `white_label/go/main_postgres.go` `wlHandleImportPairs` and the in-memory
  `white_label/go/main.go` `importLiquidity` both work on in-memory maps or return fake
  `{"imported": 10}` counts.
- The **real** white-label service (`white_label_service`) has a `wl_liquidity_pools`
  table but **no repository methods or HTTP routes** to create/import/list liquidity.

### 3.5 ❌ Liquidity & MM-bot persistence missing in white-label service
- `white_label_service/migrations/001_initial_schema.sql` defines:
  `wl_liquidity_pools`, `wl_market_maker_bots`, `wl_token_configs`, `wl_blockchains`,
  `wl_products`, `wl_api_keys`, `wl_audit_logs`, `wl_notifications`, `wl_sessions`,
  `wl_analytics_daily`.
- But the repository (`repository.go`) **only** implements CRUD for:
  clients, admins, products, **trading pairs**, and blockchains (list/update), plus
  dashboard stats, audit logs, notifications.
- **Missing methods:** liquidity-pool create/import/list/get/remove,
  market-maker-bot create/start/stop/delete, token-config CRUD, API-key management,
  product enable/disable.
- **Missing routes:** `white_label_service/main.go` only exposes `/clients`, `/domain`,
  `/admin/login`, subscriptions, and stats. There are **no** `/pairs`, `/liquidity`,
  `/bots`, `/tokens`, `/products`, or `/import` routes in the real service.

### 3.6 🟠 Liquidity math is simplified
- `admin/go` `AddLiquidity` computes `lpTokens = (AmountA + AmountB) / 2` — no constant
  product pricing, no slippage, no fee-on-mint, and no on-chain interaction. It is a
  **mock** of a DEX pool, not a real AMM.
- **Fix:** either wire to an actual AMM (Uniswap V2/V3-style) via the `cpp/liquidity_aggregator`
  or `swap_and_dex`, or document it as an off-chain approximation.

### 3.7 🟡 Not authenticated / no multi-tenant isolation
- The `admin` liquidity/pair handlers rely on GORM but there is no per-tenant filtering;
  `tenant_id` exists in ProjectParty models but is never scoped in queries.

---

## 4. Cross-Cutting Gaps (all areas)

### 4.1 🔴 No `admin` service in `docker-compose.yml` + no Dockerfile
- **`admin/`** (the most complete panel) has **no Dockerfile** and is **absent from all
  docker-compose files**. The super-admin API is included, but the `admin` panel API
  (port 9093, GORM, real persistence) is never deployed.
- **Fix:** add `admin/go/Dockerfile` + a `admin-api` service in `docker-compose.yml`.

### 4.2 🔴 Port configuration is inconsistent
| Web frontend | Backend actually runs on | Mismatch |
|--------------|--------------------------|----------|
| `admin` web → `:8080` | admin API default `9093` (compose: n/a) | ❌ |
| `super_admin` web → `:9090` | super-admin API `8082` | ❌ |
| `project_party` web → `:8106` | backend default `9006`; deploy `8004` | ❌ |
| `mm_bot_platform` → `:8080` | same `8080` | ✅ |

### 4.3 ❌ Auth is not consistently wired
- `project_party` handlers accept requests with no JWT/role checks.
- Admin UI login pages exist, but several pages call APIs without demonstrating token
  plumbing. The permission/connection services are real, but the product APIs
  (`project_party`, `super_admin`, `mm_bot_platform`) do not call them.
- **Fix:** integrate a shared auth middleware (JWT) + call `permission_service` for
  fetcher-level checks on every product route.

### 4.4 ❌ No integration tests / live end-to-end flows
- No tests exercise the admin API, ProjectParty, or bot flows against a real DB.
- The only verified-real backend (`go/wallet_api`) has tests; these product services do not.

### 4.5 ❌ No on-chain / exchange execution wiring
- `cpp/liquidity_aggregator`, `swap_and_dex`, `dex_connectors` exist as engines/connectors,
  but none of the admin-liquidity, ProjectParty market-making, or MM-bot flows call them.

---

## 5. What Is Actually Working / Complete (for calibration)

| Area | Status |
|------|--------|
| **`admin` panel (GORM)** — pairs CRUD + status + price + stats, liquidity pool add/remove/stats | ✅ Real persistence + audit logging |
| `white_label_service` — **trading-pair** CRUD (INSERT/UPDATE via `wl_trading_pairs`) | ✅ Real persistence (but only pairs; see 3.5) |
| `connection_api` — connect/heartbeat/disconnect sessions in PostgreSQL | ✅ Real (INSERT/UPDATE `connection_sessions`) |
| `permission_service` — client registration, permissions, audit log | ✅ Real DB |
| `mm_bot_platform` **Rust core** — 18 bot types + strategy engines | ✅ Implemented (but not bridged to API, see 2.2) |
| Solidity admin/strategy **contracts** — role-gated bot admin, fees | ✅ Implemented (not deployed/wired, see 2.3) |
| `go/wallet_api` — real wallet/signing/broadcast backend | ✅ Fully functional (per repo memory) |
| Super Admin web client (`api.ts`) — full method surface | ✅ UI client exists (backend stubbed) |

---

## 6. Recommended Build Order (to close the gaps)

1. **Add real persistence to `project_party`** — DDL + SQL in handlers (biggest impact,
   unblocks tokens/listings/launchpad/MM/pricing).
2. **Implement `super_admin` / `master_admin_management` handlers** against the existing
   PostgreSQL schema (DDL already present in `master_admin_management/.../postgres.go`).
3. **Fix admin pair JSON contract** + add `/pairs/import` route; add `admin/go/Dockerfile`
   and an `admin-api` compose service.
4. **Align ports** via a single env-driven config per service; update web `API_BASE_URL`s.
5. **Wire real liquidity import/CRUD** in `white_label_service` (add repo methods +
   routes for `wl_liquidity_pools`, `wl_market_maker_bots`, `wl_token_configs`).
6. **Bridge `bot_api` ↔ `bot_core` ↔ on-chain contracts**, and add persistence to the bot
   API server.
7. **Add auth** (JWT middleware + `permission_service` checks) to all product APIs.
8. **Add end-to-end integration tests** with a real PostgreSQL for the admin API,
   ProjectParty, and bot flows.

---

## 7. Legend
- 🔴 **Critical** — blocks the feature from working at all (stubs, no persistence).
- 🟠 **High** — breaks a specific flow (contract mismatch, port/config drift, missing route).
- 🟡 **Medium** — quality/robustness/scale issue (simplified math, missing tests, minor drift).
- ❌ marks an unimplemented capability; ✅ marks what is implemented.