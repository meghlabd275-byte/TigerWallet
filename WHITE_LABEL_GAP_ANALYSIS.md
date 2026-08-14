# TigerWallet White-Label Architecture — Complete Gap Analysis

> Full audit of the white-label operating model against the current codebase.
> Date: 2026-08-09. **Re-verified 2026-08-13** — see update notes below.

> **Update 2026-08-13.** A re-verification against the current source shows
> several Pillar-1 findings from the original audit are now **RESOLVED**:
> - `go/wallet_service` is no longer a mock-quote service — it is a stdlib
>   reverse-proxy shim to the canonical `go/wallet_api` (no fabricated quotes).
> - `fetcher_core/rust/src/blockchain/mod.rs` no longer hardcodes
>   `block_number: 18000000` / fake `0x1234…` addresses — it delegates to the
>   backend and is fail-closed.
> - `white_label_admin/go` is no longer a `status:ok` stub — it initializes a
>   real PostgreSQL DB (`database.Initialize`) and wires the admin router with
>   real handler logic (113 handlers).
> - MasterWallet + UserWallet clients (`master_wallet/*`, `user_wallet/*`,
>   `mobile_apps/*`) all target the real canonical backends
>   (`go/wallet_api`:8443, `master_wallet/backend`:8450) — the old
>   `api.tigerwallet.com` cloud hardcoding was removed repo-wide.
>
> Items still genuinely OPEN at re-verification:
> - `license_service` / `kill_switch` are not wired into `docker-compose.yml`
>   (Pillar 4 control-plane enforcement) — pending.
>
> **Update 2026-08-13 (2).** `selfhosted_masterwallet/rust/src/main.rs` is no
> longer a 5-line `println!` placeholder — it is now a real, standalone
> actix-web HTTP server (port 8470) with its own PostgreSQL schema (sqlx),
> real PBKDF2-HMAC-SHA256 password hashing (600k iters, constant-time verify),
> real HS256 JWT auth, and the full canonical MasterWallet REST contract
> (auth, master wallets, sub-wallets, transactions, approve/reject, fees,
> auto-sign, users, analytics, chains, health). `cargo check` exit 0.

## Target Operating Model (4 pillars)

1. White-level client/admin have approved White-level products (**MasterWallet, UserWallet, Bots + BotsClients, ProjectParty coin/token listing**). All products have full functionality with full fetchers.
2. White-level client/admin have full admin + admin panel with full fetchers — complete functionality like Super Admin, scoped to their approved white-level products.
3. All white-level products independently host + run on any cloud environment — **never** hosted or run on TigerWallet cloud.
4. **Without TigerWallet Super Admin, no product works, no functionality works, no fetcher works.** TigerWallet Super Admin completely manages each fetcher and each functionality of every white-level product, and can manage each permission and each fetcher of white-level clients/admins.

---

## Verdict Summary

| Pillar | Status |
|--------|--------|
| 1 — Full product functionality + full fetchers | ❌ **LARGELY MISSING** (scaffolding exists, live-data layer is mocked) |
| 2 — White-label admin = full super-admin (scoped) | ❌ **NOT IMPLEMENTED** (neither feature-complete nor scope-enforced) |
| 3 — Independent self-hosting (not TigerWallet cloud) | ⚠️ **PARTIAL / CONTRADICTED** (deploy docs self-host, but client endpoints target TigerWallet cloud) |
| 4 — Super Admin hard-dependency / control plane | ❌ **STRUCTURALLY NOT IMPLEMENTED** (biggest gap) |

---

## Pillar 1 — Full functionality + full fetchers: LARGELY MISSING

The product shells (services, UIs, DB) exist, but almost every fetcher that powers them is hardcoded/mocked.

### MasterWallet (`master_wallet/`)
- Real: Go service, RPC plumbing (`rpcCall`, real `eth_getTransactionCount`/`eth_sendTransaction`), white-label config/branding, HD wallet service, token registry.
- **Balance = hardcoded `"balance":"0"`** (`go/services/main.go:685-694`); sub-wallet balance `"0"`; token prices static in-memory.
- Key derivation is SHA-256-keyed, **not** BIP39/BIP32/secp256k1.
- White-label clients target TigerWallet cloud (`web/src/api.ts` → `api.tigerwallet.io`).

### UserWallet (`user_wallet/`)
- Real: pgxpool + Redis; wallet create/import/list, swap service.
- **Swap quote = mock** (`go/wallet_service.go:458` "Simplified: generate mock quote").
- Token-price/gas/network fetchers return `"provider is not configured"`.
- Rust `derive_address` = SHA-256(last-20-bytes), not BIP32.

### Bots / BotsClients (`bots/`, `mm_bot_platform/`, `bot_platform/`)
- Most real of the four: genuine Rust strategy math (grid/DCA/momentum/volatility/arbitrage) + real Solidity contracts (RBAC, fees, bot lifecycle).
- **Orders simulated** — `execute_order` instantly sets `filled`, no exchange/DEX call.
- `getMarketPrices` hardcoded (`bots/go/cmd/main.go:228` → `{"BTC":"67500","change":"3.2"}`).
- **No live exchange connections** — docs claim "20+ DEXs / 200+ CEXs" but code connects to zero.
- Prices consumed as injected `f64` only; no market-data feed.

### ProjectParty (coin/token listing) (`project_party/`, `token_creator/`, `token_management/`, `swap_and_dex/`, `dex_aggregator/`)
- Most stubbed. Postgres pool **opened but never used** (`project_party/go/cmd/main.go:26` — only `defer db.Close()`); handlers return fabricated/maps JSON.
- `listTokens` / `getToken` return hardcoded sample tokens; price `"1.50"`, volume `"100000"` regardless of token_id.
- `DeployToken` = `"Deploy (simulated)"` with fake contract address; **fabricated audit reports** (score 95/98 attributed to "TigerWallet Security Team").
- Rust nodes return `vec![]` / empty metadata.
- Only the DEX aggregator makes real HTTP calls (Uniswap/CoinGecko), with a naive 0.3%-fee `simulateQuote` fallback on any error; `GetPools` returns `[]` ("In production, query subgraph").

### Fetcher layer (`fetcher_core/`, `rust/userwallet_fetchers/`, `rust/full_fetchers/`, `rust/admin_fetchers/`)
- **Only two live fetchers in the whole repo**: EVM balance (`eth_getBalance`) and EVM gas (`eth_gasPrice`) in `rust/userwallet_fetchers/src/fetchers.rs:146,501`.
- BTC/Solana/Price/Token/NFT/Staking fetchers = `"In production, would…"` comments returning `"0.0"` / `0.0` (`fetchers.rs:168,180,285,557`).
- `fetcher_core/rust/src/blockchain/mod.rs:165` — explicit mock: hardcoded `block_number: 18000000`, fake `0x1234567890abcdef`, placeholder Infura URL.
- `rust/full_fetchers` — contains `rand()`-based data paths, not verified live fetchers.

### Gap
Implement real fetchers (balances, gas, prices, order books, pools, listings, KYC, transactions) and wire them into the **actual handler paths** — not the parallel `production/react` stacks that are disconnected from each product's `web/src/services/api.ts`.

---

## Pillar 2 — White-label admin with full super-admin functionality, scoped: NOT IMPLEMENTED

- **Route/feature parity is false**: `super_admin/go` = **189** handlers vs `white_label_admin/go` **116**, and the white-label backend is a **stub** (every handler returns `gin.H{"status":"ok"}`; `handleLogin` returns a literal string; JWT/db imports unused).
- **No per-product scoping in the admin layer.** The only real DB-backed admin (`admin/go`, 255 routers) has roles `super_admin / admin / support / analyst / moderator` — **there is no `white_label_admin` role** and **no middleware checks `product_permissions`**.
- The `WhiteLabel` model (`admin/go/internal/models/models.go:231`) has **no columns scoping products/fetchers** (only `name/slug/domain/logo/colors/status/contact/features/fee_structure`).
- The "white-label admin = full super-admin, scoped to approved products" promise exists **only in an in-memory stub** (`admin/go/internal/services/super_admin_service.go`: `IsFeatureEnabled:811`, `SetMasterAdminFeature:780`) — not wired to any route, DB, or admin UI.
- Only **one real polished admin frontend**: `admin/web` (React → `admin/go`). **No fetcher-configuration page, no product-permission page.**
- `frontend/admin_dashboard` (CompleteAdminDashboard.tsx) and `frontend/admin_panel` are static/mock UIs with **zero API calls**.
- `white_label/frontend` has real pages (ProductManagement, AdminManagement…) but is backed by `white_label/go/main.go` (in-memory `sync.Map`) and `main_postgres.go` (hardcoded `5000 users`/`$12.5M`, `user1@example.com`).

### Gap
- Real white-label admin backend with full functionality.
- Real RBAC layer binding each white-label admin to exactly their approved products and enabled fetchers.
- Admin UI to manage those permissions.

---

## Pillar 3 — Independent self-hosting (not TigerWallet cloud): PARTIAL / CONTRADICTED

- `docker-compose.yml` deploys 10 standalone services (postgres, redis, wallet-api, wallet-frontend, super-admin-api, white-label-frontend, permission-service, connection-api, fetcher-gateway, monitoring-dashboard). White-label deployments bring their own self-contained stacks (`master_wallet/BUILD_AND_DEPLOY.md` runs only postgres/redis/master-wallet-api/fetchers/web in its own network).
- **BUT** `master_wallet` clients still hardcode `https://api.tigerwallet.io/master` and `wss://master-ws.tigerwallet.com/ws` (Flutter, mobile, web) — wired to TigerWallet cloud, violating this pillar.

### Gap
De-couple all white-label product clients from `*.tigerwallet.io` / `*.tigerwallet.com` so they run standalone on the client's cloud.

---

## Pillar 4 — Super Admin control plane (hard dependency): STRUCTURALLY NOT IMPLEMENTED — BIGGEST GAP

This is the core of the model: a **centralized super-admin control plane** that white-label products and fetchers *must* call (license/heartbeat → fetch approval → kill-switch), failing closed when unreachable. None of this exists.

| Mechanism | Required | Actual |
|-----------|----------|--------|
| **License enforcement** | Products refuse to start/run without a valid super-admin license | `license_service` is **orphaned** — its client `NewLicenseService` (`sdks/go/client.go:258`) is defined but **never called**; always returns `"valid": true` (`license_service/go/cmd/main.go:99`); generates an RSA key that is **never used** to sign/verify |
| **Kill-switch enforcement** | Super admin can remotely shut down a white-label product | `kill_switch` has a **sender with zero receivers** — it writes `remote_commands`/Redis, but **no product daemon subscribes or polls**. Repo-wide grep for `kill_switch` outside its own file → nothing |
| **Fetcher control** | Super admin can enable/disable each fetcher per white-label; fetchers refuse to run otherwise | `permission_service` `CheckPermission`/`GrantPermission` are **local** (local Postgres + in-memory cache, no central super-admin round-trip). Worse: `fetcher_gateway`'s `check_permission` reads the **same local DB** and is **NOT even called by the `fetch_data` handler** (`fetcher_gateway/rust/src/main.rs:225-426` only enforces a local rate-limiter). The permission-gate route `/api/v1/permission/check` is **dead** |
| **Super-admin management of white-label clients/admins** | Super admin remotely provisions, scopes, and controls each white-label's products, fetchers, permissions | In `docker-compose.yml`, `super-admin-api` exists but **no service depends on it**; `license_service`/`kill_switch` are **not even in the compose**. `selfhosted_masterwallet/rust/src/main.rs` is a one-line `println!` placeholder |

### Consequences today
- White-label products run **fully standalone** — they don't call any super-admin service.
- Fetchers run **without any permission gate**.
- License and kill-switch are stubs with **no consumers**.

### Gap
Build the **super-admin control plane** (license server, heartbeat, kill-switch consumer, per-tenant fetcher permission) and make every white-label product + fetcher **call it and fail-closed** when unreachable/rejected. This is the largest single missing piece.

---

## Priority Gap Register

| # | Missing capability | Where | Category |
|---|--------------------|-------|----------|
| 1 | Super-admin control plane (license server, heartbeat, kill-switch consumer, per-tenant fetcher permission) that products actually depend on | `license_service`, `kill_switch`, `fetcher_gateway`, `permission_service` | **Missing foundation** |
| 2 | Live fetchers (balances, gas, prices, order books, pools, listings) wired into real handlers — only 2 EVM fetchers are real | all 4 products + `fetcher_core` | Large stubbing gap |
| 3 | White-label admin with full super-admin-equivalent functionality (real backend, not stubs) | `white_label_admin/go`, `super_admin/go` | Missing |
| 4 | Per-product / per-fetcher permission-scoped RBAC bound to each white-label admin, with admin UI | `admin/go`, role model, `permission_service` wiring | Missing |
| 5 | Remove TigerWallet-cloud coupling from white-label client endpoints | `master_wallet/web/src/api.ts` etc. → `api.tigerwallet.io` | Contradicts Pillar 3 |
| 6 | Real BIP-39/BIP-32/secp256k1 key derivation (currently SHA-256-keyed) | `master_wallet`, `user_wallet`, `admin/rust` signatures | Crypto correctness |
| 7 | Real token deploy + real audit (not simulated) | `token_creator` | Fake production data |
| 8 | Real DEX/CEX connectivity for bots (docs claim 20+ DEX / 200+ CEX; code connects to zero) | `bots`, `mm_bot_platform` | Contradicts docs |

---

## Bottom Line

The codebase has the **scaffolding** (product shells, admin UIs, a permission-service, a fetcher-gateway, license/kill-switch services) but **does not implement the core operating model described**:

- There is **no super-admin control plane** that white-label products and fetchers are wired to depend on.
- White-label admins **do not actually get full, product-scoped super-admin functionality** with live fetchers.
- Most fetchers that power the products **return hardcoded/mock data**.
- Some white-label clients **still target TigerWallet cloud**, contradicting the self-hosting requirement.

The documentation (root README, BOT_PLATFORM.md, USERWALLET_FEATURES.md, USERWALLET_DETAILED_COMPARISON.md, TIGERWALLET_WALLET_SYSTEM_SPECIFICATION.md) claims complete, production-grade functionality that substantially exceeds what the code implements.

---

*This document was prepared by an AI agent (OpenHands) on behalf of the TigerWallet project as part of a white-label architecture gap audit.*