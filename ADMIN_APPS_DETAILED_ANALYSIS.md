# TigerWallet Admin Applications — Full Inventory, Gaps & Role Capabilities

> Comprehensive analysis of all Admin, SuperAdmin, and AdminPanel applications:
> their fetchers, functionality, what is missing, what is still in gaps, and
> which actions each role (Admin / SuperAdmin / AdminPanel) can and cannot perform.

---

## Executive Overview

There are **three administratively separate control planes** in this repository,
each with its own Go backend plus scattered frontend/native clients:

| Surface | Backend | Default Port | Primary Clients |
|---------|---------|-------------|-----------------|
| **Admin** | `admin/go` (Gin) | **9093** | `admin/web`, `admin/flutter`, `admin/android`, `admin/ios`, `admin/desktop`, `admin/extensions`, `admin/rust` (stub), `admin/cpp` (stub) |
| **SuperAdmin** | `super_admin/go` (Gin) | **8082** | `super_admin/web`, `super_admin/android`, `super_admin/desktop`, `super_admin/extensions`, `super_admin/rust`, `super_admin/cpp` |
| **adminPanel** | `go/admin_service` + `go/rbac_admin_service` + `api_gateway/rest_api/tiger_admin_api.go` | 8080/— | `frontend/admin_panel` (React) — calls `/api/v1/admin/*` |

They share Postgres (`tigerwallet` / `admin` schemas) and Redis — they are
**not** fully isolated data-stores, but each is a separate deployable service
with separate handlers.

> **Reminder (isolation requirement):** UserWallet, MasterWallet, and Admin apps
> must never access each other's fetchers or functionality. Each of the three
> admin surfaces below is also intended to be separately scoped. The gaps section
> highlights where this separation is currently **not** enforced (especially the
> RBAC middleware hole in `admin/go`).

---

## 1. ADMIN (`admin/`) — 223 routes, full RBAC middleware (port 9093)

### Fetchers / Functionality (per route group)

- **Auth / Profile**: login, refresh, logout, profile get/update, change-password.
- **2FA (TOTP)**: setup, verify, enable, disable, status, backup-codes regen, list-2FA-users, stats.
- **Admin management** *(SuperAdmin-gated)*: create / list / get / update / delete / suspend / activate admins, admin activity log.
- **Dashboard & Analytics**: dashboard, user/transaction/revenue analytics, custom-range analytics, system metrics.
- **Users**: list, get, update, delete, verify-KYC.
- **KYC**: list, get, approve, reject, stats.
- **Transactions**: list, get, flag.
- **Tokens**: full CRUD + activate / deactivate / verify + price update + stats.
- **Withdrawals**: list, get, approve, reject, process, bulk-approve, stats.
- **White labels**: CRUD + approve / suspend + stats.
- **Pairs**: CRUD + status/price update + stats.
- **Fees**: CRUD + calculate + stats.
- **API Keys**: CRUD + revoke / reactivate / regenerate + stats.
- **System config** *(SuperAdmin-gated)*: config get/update, rate-limits get/update, master-wallet list/get/balance.
- **Feature flags**: CRUD *(only Auth-gated — **NOT SuperAdmin-restricted**)*.
- **Notifications**: list/get/read/delete/send/broadcast/template/stats.
- **Tickets**: CRUD + messages + close + SLA-violations + stats.
- **Integrations**: CRUD + test + Slack / PagerDuty / Datadog / webhook senders + stats.
- **Brokers**: CRUD + approve / suspend + commission + clients + stats.
- **Institutional clients**: CRUD + approve / suspend + limits + account-manager + stats.
- **Compliance**: AML report, tax report, GDPR process / export / anonymize, reports list, stats.
- **Knowledge base**: categories + articles CRUD + stats.
- **Multisig**: CRUD multisig wallets.
- **NFTs**: list / get / delete / flag / stats.
- **Master wallet**: stats, balances, transactions.
- **Billing**: plans CRUD, subscription get/create/cancel, invoices, payment-methods CRUD + set-default.
- **Crypto cards**: CRUD + block / activate + set-limit.
- **Features**: CRUD + toggle + rollout + check.
- **Liquidity**: pools list/get/create + add/remove liquidity + stats.
- **Margin trading**: positions list/open/close + liquidation-price + stats.
- **P2P merchants**: CRUD + approve / reject + transactions.
- **Audit logs**: get *(only Auth-gated — not SuperAdmin restricted)*.

### What is missing / gaps — Admin

1. **ACCESS-CONTROL HOLE (critical):** Only `/admins` and `/system` are
   SuperAdmin-gated. All other ~200 endpoints — including **feature-flags,
   audit-logs, master-wallet, billing, crypto-cards, liquidity, margin, p2p,
   integrations, compliance, KYC, withdrawals** — are open to **any authenticated
   non-superadmin**. `RoleMiddleware`, `AdminMiddleware`,
   `PermissionMiddleware`, and `RateLimitMiddleware` exist in `auth.go` but are
   **never wired** into the router (dead code). The `rate_limiter.go` Redis
   service is also not attached.
2. **Two parallel ticket systems** — `compliance_service.go`'s `TicketService`
   and `admin_ticket_system.go`'s `AdminTicketService` are duplicative;
   `compliance_service` is in-memory only ("In real implementation, save to
   database").
3. **Services that are simulated / in-memory, disconnected from handlers:**
   - `archival.go` — archival is simulated ("In production, this would actually archive").
   - `chain_management.go` / `listing_management.go` — in-memory defaults, no DB.
   - `report_service.go` — CSV/JSON only; PDF/Excel **stubbed**; uses sample data.
   - `sla.go` — metrics simplified/hardcoded.
   - `super_admin_service.go` — profit-sharing transfers **simulated**.
4. **Billing handler is a hardcoded stub** — `NewBillingHandler()` takes no DB
   (nil), returns hardcoded Basic/Pro/Enterprise plans.
5. **Frontend breakage (web):** `Features.tsx`, `MarginTrading.tsx`,
   `P2PMerchant.tsx`, `Liquidity.tsx`, `CryptoCards.tsx` import non-existent APIs
   → **won't compile**. The web client doesn't cover most blocks (billing,
   crypto-cards, liquidity, margin, p2p, master-wallet, integrations, brokers,
   institutional, compliance, KB, multisig, NFTs, notifications, tickets).
6. **Android:** 3 incoherent trees; corrupted Retrofit annotation
   `@POST("kyc/{id}/request-more-info")`; **no `AndroidManifest.xml`** → not
   buildable; package/fragment mismatches.
7. **Flutter:** `AppConstants.baseUrl = 'https://api.tigerwallet.com/v1'`
   (mismatch with `:9093/api/v1`); 7 orphan un-registered mock screens; unresolved
   `moreRoute` → crash.
8. **iOS:** baseURL `https://api.tigerwallet.com/admin/v1` mismatch; 8 placeholder
   More-screen controllers; CryptoCards hardcoded mock + un-wired Margin.
9. **Desktop:** `package.json` "main" points to missing `src/main/main.js`
   (actual is `src/main.js`); two uncompiled renderers → not runnable.
10. **Base-URL chaos across clients:** backend `9093` vs web `8080` (default),
    extensions `9093` ✓, flutter/ios `api.tigerwallet.com`, android
    `api.tigerwallet.io`.
11. **Rust & C++ are pure scaffolding/stubs** (empty JSON, mock returns) — no real
    fetchers.

---

## 2. SUPERADMIN (`super_admin/`) — ~90+ routes (port 8082)

### Fetchers / Functionality

- **Auth**: login, register, refresh, logout, change-password, 2FA enable/disable.
- **Users**: list, get, update status, **ban / unban**, suspend.
- **KYC**: list, approve, reject.
- **Transactions**: list, get, **flag / unflag**.
- **Withdrawals**: list, approve, reject, **process**.
- **Tokens**: list, create, update, delete.
- **Pairs**: list, create, update status.
- **Blockchains**: list, create, update, **status toggle**.
- **Fees**: list, create, update.
- **Webhooks**: list, create, **test**, delete.
- **Notifications**: list, mark-read, send, **broadcast**.
- **Audit logs**: list, **export**.
- **Sessions**: list, **revoke one / revoke all**.
- **Feature flags**: CRUD.
- **IP whitelist**: list, add, remove.
- **Tickets**: list/get/create/update-status/messages/**assign**.
- **White labels**: CRUD.
- **Stats**: aggregated stats.
- **Bots & Bot-Tiers**: CRUD + status + stats + tiers CRUD.
- **Bots-clients**: CRUD + status.
- **Project teams**: CRUD + members add/remove.
- **WL clients / WL master-wallets / WL user-wallets / WL bots / WL bots-clients /
  WL project-teams**: full CRUD + status for the **white-label tenant ecosystem**.
- **Master wallets**: list/get/create/update/delete + **balance** + **transfer**.
- **User wallets**: list/get/create/update/delete + balance.
- **Admins**: list/get/update/delete/suspend/activate.
- **Workflows**: CRUD.
- **Approval requests**: list, approve, reject.
- **Backups**: list, create, **restore**, delete.
- **Knowledge base**: CRUD.
- **Archival**: policies CRUD + run archive + records.
- **Reports**: configs CRUD + list + **generate**.
- **SLA**: policies CRUD + reports.

### What is missing / gaps — SuperAdmin

1. **Frontend drastically incomplete:** of 19 page files, **14 are only 2 lines**
   (stubs — `APIKeys, Admins, AuditLogs, Blockchains, Fees, KnowledgeBase,
   Reports, Settings, System, Tickets, Tokens, TradingPairs, Webhooks,
   WhiteLabels, Withdrawals, Workflows`). Only Dashboard / Users / KYC /
   Transactions / Security / Login have real UI. The 2008-line `api.ts` defines
   many methods (incl. reports download, WL entities, bots) but most pages don't
   consume them.
2. **SuperAdmin web points at `:9090`** while backend default is **`:8082`** →
   won't connect without env override.
3. **Web/backend route gaps:** `api.ts` references endpoints (e.g.
   `/reports/:id/download`, backups export, sessions) that need a matching backend
   route verified for every one — some backend routes (e.g. `/reports` GET vs
   `/reports/generate`) are present, but download-by-id is the most likely mismatch.
4. **No frontend for many backend capabilities** (approval-requests, backup
   restore UI, archival UI, workflows UI, project-team members beyond the 2-line
   stubs).

---

## 3. ADMIN PANEL (`frontend/admin_panel` + `go/admin_service` +
`go/rbac_admin_service` + `api_gateway/rest_api/tiger_admin_api.go`)

This is a **separate "operations" admin panel**. Its frontend pages call
`/api/v1/admin/*` and `/api/v1/2fa/*`.

### Frontend pages & what they fetch

- **Dashboard, Users, Transactions, Analytics, Fees, Integrations, Compliance,
  Notifications, Security, Settings, Support, Bots, Bridges, Chains, DEXs,
  MarketMaker, Pools, Treasury**.
- Endpoint targets seen:
  - `2fa/*`
  - `admin/bridges`, `admin/chains`, `admin/dexs`, `admin/fees`,
    `admin/integrations`, `admin/market-makers`, `admin/notifications`,
    `admin/pools`, `admin/security/ip-rules(+stats)`, `admin/support/tickets(+stats)`,
    `admin/transactions(+stats)`, `admin/treasury(+stats)`,
    `admin/analytics/{dashboard,revenue,volume}`,
    `admin/compliance/{aml,tax,reports,stats}`,
    `integrations/slack/notify`.

### What the candidate backends actually serve

- **`go/admin_service`** (`/api/v1/*`, **no `/admin` segment**): login, admins,
  users + KYC/suspend, blockchains, tokens, pairs + import, white-labels,
  withdrawals + approve/reject, analytics, activities, bulk ops
  (users/tokens/withdrawals), CSV exports, fees (trading/withdrawal/deposit),
  api-keys. **This is the `admin_panel`'s main intended backend** — but its routes
  are **NOT** under `/admin/*`, so the frontend (which prefixes `/api/v1/admin/...`)
  **will 404** against it.
- **`go/rbac_admin_service`** (`/api/v1/admin/*`): users (search), kyc,
  transactions, pairs, fees, blockchains, bots, bot-tiers, api-keys, stats.
  **Partially matches** the frontend but **no bridges / dexs / pools / treasury /
  market-makers / notifications / compliance / support / security / 2fa**.
- **`api_gateway/rest_api/tiger_admin_api.go`** (`/api/v1/admin/*`): blockchains,
  fees (configs/addresses/collections), bots (tiers/instances), external
  connections, listings. **Different subsets.**

### What is missing / gaps — adminPanel (the largest structural gap)

1. **No single backend implements the full `/api/v1/admin/*` surface the frontend
   expects.** The panel pages for **Bridges, Chains, DEXs, Pools, MarketMaker,
   Treasury** have **virtually no matching Go route** anywhere (bridges live only
   in `cross_chain_aggregator`; dexs only in `external_trading.go` of the gateway;
   treasury in `master_wallet_service` — all under different paths/prefixes).
   These panels are **frontend-only mock/spec** in practice.
2. **Bridges / Chains / DEXs / MarketMaker / Treasury pages** — no authoritative
   backend; likely 404 or empty in production.
3. **`2fa/*`** endpoints exist only in the **Admin** backend (`admin/go`), not in
   any of these three admin_panel backends — so the panel's Security/2FA page
   points at a service that doesn't serve it.
4. **`integrations/slack/notify`** is a separate path from the `Admin` backend's
   `/integrations/slack`.
5. **RBAC separation:** `go/admin_service` has no role middleware beyond login
   token; the "admin vs superadmin" distinction is not enforced on the panel
   operations surface.

---

## Which actions each role CAN perform vs CANNOT

### Super Admin (highest), in `admin/go`

**CAN (exclusive):** manage admins (create/delete/suspend/activate/see
activities), system config, rate-limits, master-wallet system access.
**CAN (shared, unrestricted due to the middleware gap):** *everything else* —
users / KYC / withdrawals / tokens / fees / feature-flags / **audit-logs** /
billing / crypto-cards / compliance / liquidity / margin / p2p etc.
**CANNOT (in the SuperAdmin app `super_admin/go`):** run trades, access
user-wallet signing keys, or send on-chain transactions — its scope is
tenant/wallet/bot/report management.

### Admin, in `admin/go`

**CAN:** users, KYC approve/reject, transaction flagging, withdrawals
approve/process, tokens/pairs/fees CRUD, white-label oversight, tickets,
notifications, integrations, brokers, analytics, dashboard.
**SHOULD NOT be able to:** admin management, system config, rate-limit config,
master-wallet system access, feature-flags, audit-logs, billing, crypto-cards,
compliance GDPR, liquidity, margin, p2p merchant approval,
master-wallet/billing permissions — **BUT due to the middleware hole, it
currently CAN access all of these.** That is the single biggest security gap in
the Admin app.

### adminPanel operator

**CAN (per `go/admin_service` backend):** users (KYC updates, suspend),
blockchains, tokens, pairs (+import), white-labels, withdrawals (approve/reject),
analytics, activities, bulk suspend/activate/delete users/tokens, bulk
approve/reject withdrawals, CSV exports, fee updates, api-keys. Admin CRUD.
**CANNOT:** access master-wallet private keys, transaction broadcasting/signing,
on-chain fund movement, SuperAdmin-only WL-tenant/bot/master-wallet management,
audit-log management, 2FA management (its 2FA calls hit a different backend).

---

## Cross-cutting gaps summary

1. **Isolation:** All three share Postgres/Redis and are only *logically*
   separate — no per-app DB segregation / tenancy enforcement in the shared
   `tigerwallet` / `admin` databases.
2. **AuthZ enforcement:** `admin/go`'s role/permission/rate-limit middleware is
   dead code — the "can / can't" matrix is not enforced in practice.
3. **Frontend ↔ backend port / BASE_URL mismatches everywhere**
   (9093 / 8080 / 8082 / 9090 / api.tigerwallet.com / api.tigerwallet.io).
4. **adminPanel is the least complete:** several pages (bridges / dexs / pools /
   treasury / market-makers / 2fa) have no matching backend; it depends on
   endpoints scattered across services.
5. **Stubs everywhere:** Rust & C++ backends in all three apps are scaffolding;
   archival / reports / SLA / billing services are simulated; iOS and Android have
   mock screens and broken builds.

---

## Recommended priority fixes

1. **Wire the RBAC middleware** (`RoleMiddleware` / `AdminMiddleware` /
   `RateLimitMiddleware`) into `admin/go/main.go` so the role restrictions
   actually hold. Highest impact / security-critical.
2. **Repair the broken web page imports** in `admin/web` (Features, MarginTrading,
   P2PMerchant, Liquidity, CryptoCards).
3. **Add a unified `/api/v1/admin/*` backend** for the adminPanel pages that have
   no route (bridges / dexs / pools / treasury / market-makers / 2fa /
   notifications / support / security / compliance).
4. **Reconcile BASE_URL across all clients** to the actual backend ports.
5. **Remove duplicate ticket systems** and connect in-memory services (archival,
   chain_management, listing_management, report, sla, billing) to real DB storage.