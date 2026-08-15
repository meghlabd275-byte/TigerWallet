# TigerWallet Admin Applications — Complete Fetcher & Functionality Inventory

> Complete analysis of all admin control planes (Admin, SuperAdmin, adminPanel):
> their full fetchers and functionality, what is real vs stubbed, what is missing,
> what gaps remain, and what actions each role CAN or CANNOT perform.
>
> **Isolation reminder:** Admin apps never access MasterWallet or UserWallet fetchers.
> UserWallet apps never access Admin or MasterWallet fetchers. MasterWallet apps
> never access Admin or UserWallet fetchers. Each family is completely separated.

---

## 1. Overview — Three Separate Admin Control Planes

| Surface | Backend | Default Port | Clients |
|---------|---------|-------------|---------|
| **Admin** | `admin/go` (Gin) | **9093** | web, flutter, android, ios, desktop, extensions, rust, cpp |
| **SuperAdmin** | `super_admin/go` (Gin) | **8082** | web, android, desktop, extensions, rust, cpp |
| **adminPanel** | `go/admin_service` + `go/rbac_admin_service` + `api_gateway/rest_api/tiger_admin_api.go` | 8080/— | React (`frontend/admin_panel`) |

They share Postgres (`tigerwallet`/`admin` schemas) and Redis — logically separate
services but not fully DB-isolated.

---

## 2. ADMIN (`admin/go`) — 223 Routes, Port 9093

### 2.1 Full Route Map by Area

#### Auth / Profile (all authenticated)
```
POST /auth/login
POST /auth/refresh
POST /auth/logout
GET  /auth/profile
PUT  /auth/profile
POST /auth/change-password
```

#### 2FA / Security
```
POST /2fa/setup
POST /2fa/verify
POST /2fa/enable
POST /2fa/disable
GET  /2fa/status
POST /2fa/backup-codes
GET  /2fa/users
GET  /2fa/stats
```

#### Admin Management ⚠️ SUPERADMIN-GATED
```
GET  /admins
POST /admins
GET  /admins/:id
PUT  /admins/:id
DELETE /admins/:id
POST /admins/:id/suspend
POST /admins/:id/activate
GET  /admins/:id/activities
```

#### Dashboard & Analytics (all authenticated)
```
GET /dashboard
GET /analytics/users
GET /analytics/transactions
GET /analytics/revenue
GET /analytics/custom
GET /system/metrics
```

#### Users
```
GET  /users
GET  /users/:id
PUT  /users/:id
DELETE /users/:id
POST /users/:id/verify-kyc
```

#### KYC
```
GET  /kyc
GET  /kyc/:id
POST /kyc/:id/approve
POST /kyc/:id/reject
GET  /kyc/stats
```

#### Transactions
```
GET  /transactions
GET  /transactions/:id
POST /transactions/:id/flag
```

#### Tokens
```
GET  /tokens
POST /tokens
GET  /tokens/:id
PUT  /tokens/:id
DELETE /tokens/:id
POST /tokens/:id/activate
POST /tokens/:id/deactivate
POST /tokens/:id/verify
PUT  /tokens/:id/price
GET  /tokens/stats
```

#### Withdrawals
```
GET  /withdrawals
GET  /withdrawals/:id
POST /withdrawals/:id/approve
POST /withdrawals/:id/reject
POST /withdrawals/:id/process
POST /withdrawals/bulk-approve
GET  /withdrawals/stats
```

#### White Labels
```
GET  /white-labels
POST /white-labels
GET  /white-labels/:id
PUT  /white-labels/:id
DELETE /white-labels/:id
POST /white-labels/:id/approve
POST /white-labels/:id/suspend
GET  /white-labels/stats
```

#### Trading Pairs
```
GET  /pairs
POST /pairs
GET  /pairs/:id
PUT  /pairs/:id
DELETE /pairs/:id
PUT  /pairs/:id/status
PUT  /pairs/:id/price
GET  /pairs/stats
```

#### Fees
```
GET  /fees
POST /fees
GET  /fees/:id
PUT  /fees/:id
DELETE /fees/:id
POST /fees/calculate
GET  /fees/stats
```

#### API Keys
```
GET  /api-keys
POST /api-keys
GET  /api-keys/:id
PUT  /api-keys/:id
DELETE /api-keys/:id
POST /api-keys/:id/revoke
POST /api-keys/:id/reactivate
POST /api-keys/:id/regenerate
GET  /api-keys/stats
```

#### System Config ⚠️ SUPERADMIN-GATED
```
GET  /system/config
PUT  /system/config
GET  /system/rate-limits
PUT  /system/rate-limits
GET  /system/master-wallets
GET  /system/master-wallets/:id
GET  /system/master-wallets/:id/balance
```

#### Feature Flags (NOT gated — open to any authenticated)
```
GET  /feature-flags
POST /feature-flags
PUT  /feature-flags/:id
DELETE /feature-flags/:id
```

#### Notifications
```
GET  /notifications
GET  /notifications/:id
PUT  /notifications/:id/read
DELETE /notifications/:id
GET  /notifications/stats
POST /notifications/send
POST /notifications/broadcast
POST /notifications/template
```

#### Support Tickets
```
GET  /tickets
POST /tickets
GET  /tickets/:id
PUT  /tickets/:id
POST /tickets/:id/messages
POST /tickets/:id/close
GET  /tickets/stats
GET  /tickets/sla-violations
```

#### Integrations
```
GET  /integrations
POST /integrations
PUT  /integrations/:id
DELETE /integrations/:id
POST /integrations/:id/test
POST /integrations/slack
POST /integrations/pagerduty
POST /integrations/datadog
POST /integrations/webhook
GET  /integrations/stats
```

#### Brokers
```
GET  /brokers
POST /brokers
GET  /brokers/:id
PUT  /brokers/:id
DELETE /brokers/:id
POST /brokers/:id/approve
POST /brokers/:id/suspend
PUT  /brokers/:id/commission
GET  /brokers/:id/clients
GET  /brokers/stats
```

#### Institutional Clients
```
GET  /institutional
POST /institutional
GET  /institutional/:id
PUT  /institutional/:id
DELETE /institutional/:id
POST /institutional/:id/approve
POST /institutional/:id/suspend
PUT  /institutional/:id/limits
PUT  /institutional/:id/account-manager
GET  /institutional/stats
```

#### Compliance
```
POST /compliance/aml-report
POST /compliance/tax-report
POST /compliance/gdpr
GET  /compliance/reports
GET  /compliance/stats
POST /compliance/gdpr/export
POST /compliance/gdpr/anonymize
```

#### Knowledge Base
```
GET  /knowledge-base/categories
POST /knowledge-base/categories
PUT  /knowledge-base/categories/:id
DELETE /knowledge-base/categories/:id
GET  /knowledge-base/articles
POST /knowledge-base/articles
GET  /knowledge-base/articles/:id
PUT  /knowledge-base/articles/:id
DELETE /knowledge-base/articles/:id
GET  /knowledge-base/stats
```

#### Multisig
```
GET  /multisig
POST /multisig
GET  /multisig/:id
PUT  /multisig/:id
DELETE /multisig/:id
```

#### NFTs
```
GET  /nfts
GET  /nfts/:id
DELETE /nfts/:id
POST /nfts/:id/flag
GET  /nfts/stats
```

#### Master Wallet
```
GET  /master-wallet/stats
GET  /master-wallet/balances
GET  /master-wallet/transactions
```

#### Billing
```
GET  /billing/plans
POST /billing/plans
PUT  /billing/plans/:id
DELETE /billing/plans/:id
GET  /billing/subscription
POST /billing/subscription
DELETE /billing/subscription
GET  /billing/invoices
GET  /billing/payment-methods
POST /billing/payment-methods
DELETE /billing/payment-methods/:id
PUT  /billing/payment-methods/:id/default
```

#### Crypto Cards
```
GET  /crypto-cards
POST /crypto-cards
GET  /crypto-cards/:id
POST /crypto-cards/:id/block
POST /crypto-cards/:id/activate
PUT  /crypto-cards/:id/limit
```

#### Features (per-feature toggles)
```
GET  /features
POST /features
GET  /features/:id
PUT  /features/:id
POST /features/:id/toggle
PUT  /features/:id/rollout
DELETE /features/:id
GET  /features/:id/check
```

#### Liquidity
```
GET  /liquidity/pools
GET  /liquidity/pools/:id
POST /liquidity/pools
POST /liquidity/pools/:id/add
POST /liquidity/pools/:id/remove
GET  /liquidity/stats
```

#### Margin Trading
```
GET  /margin/positions
POST /margin/positions
POST /margin/positions/:id/close
PUT  /margin/positions/:id/liquidation
GET  /margin/stats
```

#### P2P Merchants
```
GET  /p2p-merchants
POST /p2p-merchants
GET  /p2p-merchants/:id
PUT  /p2p-merchants/:id
POST /p2p-merchants/:id/approve
POST /p2p-merchants/:id/reject
GET  /p2p-merchants/:id/transactions
```

#### Audit Logs (NOT gated — open to any authenticated)
```
GET /audit-logs
```

### 2.2 What Admin CAN and CANNOT Do (Role Gating)

The RBAC middleware exists (`auth.go`) but is **NOT wired** into the router:

| Action | SHOULD be gated | ACTUALLY gated | Status |
|--------|----------------|----------------|--------|
| Manage admins | ✅ SuperAdmin only | ✅ Only `/admins` group | Works |
| System config / rate-limits / master-wallet | ✅ SuperAdmin only | ✅ Only `/system` group | Works |
| Feature flags | ? | ❌ Any authenticated | **GAP** |
| Audit logs | ? | ❌ Any authenticated | **GAP** |
| Master wallet (stats/balances/txs) | ? | ❌ Any authenticated | **GAP** |
| Billing | ? | ❌ Any authenticated | **GAP** |
| Crypto cards | ? | ❌ Any authenticated | **GAP** |
| Compliance GDPR export/anonymize | ? | ❌ Any authenticated | **GAP** |
| Liquidity / Margin / P2P | ? | ❌ Any authenticated | **GAP** |
| KYC approve/reject | ? | ❌ Any authenticated | **GAP** |
| Withdrawals approve/process | ? | ❌ Any authenticated | **GAP** |

Dead code in `auth.go`: `RoleMiddleware`, `AdminMiddleware`, `PermissionMiddleware`,
`RateLimitMiddleware` — defined but never applied to any route group.

### 2.3 Admin App — Missing / Gaps

1. **Access control broken** — ~200 endpoints open to non-superadmin (see table above).
2. **Two parallel ticket systems** — `compliance_service.go`'s `TicketService` (in-memory)
   vs `admin_ticket_system.go`'s `AdminTicketService` (real). Only the latter is used.
3. **Simulated / in-memory services disconnected from handlers:**
   - `archival.go` — simulated ("In production, this would actually archive")
   - `chain_management.go` / `listing_management.go` — in-memory, no DB
   - `report_service.go` — CSV/JSON only; PDF/Excel stubbed
   - `sla.go` — hardcoded simplified metrics
   - `super_admin_service.go` — profit-sharing transfers simulated
4. **Billing handler is a hardcoded stub** — `NewBillingHandler()` has nil DB, returns
   hardcoded Basic/Pro/Enterprise plans.
5. **Web frontend:** `Features.tsx`, `MarginTrading.tsx`, `P2PMerchant.tsx`,
   `Liquidity.tsx`, `CryptoCards.tsx` import **non-existent APIs** → won't compile.
   Web doesn't cover billing, crypto-cards, liquidity, margin, p2p, master-wallet,
   integrations, brokers, institutional, compliance, KB, multisig, NFTs,
   notifications, tickets.
6. **Android:** 3 incoherent trees; corrupted Retrofit annotation; no `AndroidManifest.xml`;
   package/fragment mismatches → not buildable.
7. **Flutter:** wrong `baseUrl`; 7 orphan un-registered mock screens; unresolved
   `moreRoute` → crash.
8. **iOS:** wrong baseURL; 8 placeholder More-screen VCs; mock CryptoCards not wired.
9. **Desktop:** `package.json` main entry wrong; two uncompiled renderers → not runnable.
10. **Base-URL chaos:** extensions `9093`, web `8080`, flutter/ios `api.tigerwallet.com`,
    android `api.tigerwallet.io` — none point at the actual `9093` backend.
11. **Rust & C++:** pure scaffolding (empty JSON, mock returns).

---

## 3. SUPERADMIN (`super_admin/go`) — 186 Routes, Port 8082

### 3.1 Full Route Map by Area

#### Auth / Profile
```
POST /api/v1/auth/login
POST /api/v1/auth/register
POST /api/v1/auth/refresh
POST /api/v1/logout
POST /api/v1/change-password
POST /api/v1/2fa/enable       ← stub
POST /api/v1/2fa/disable      ← stub
```

#### Users
```
GET  /api/v1/users
GET  /api/v1/users/:id
PUT  /api/v1/users/:id/status
POST /api/v1/users/:id/ban
POST /api/v1/users/:id/unban
POST /api/v1/users/:id/suspend
```

#### KYC
```
GET  /api/v1/kyc
POST /api/v1/kyc/:id/approve
POST /api/v1/kyc/:id/reject
```

#### Transactions
```
GET  /api/v1/transactions
GET  /api/v1/transactions/:id
POST /api/v1/transactions/:id/flag
POST /api/v1/transactions/:id/unflag
```

#### Withdrawals
```
GET  /api/v1/withdrawals
POST /api/v1/withdrawals/:id/approve
POST /api/v1/withdrawals/:id/reject
POST /api/v1/withdrawals/:id/process
```

#### Tokens
```
GET  /api/v1/tokens
POST /api/v1/tokens
PUT  /api/v1/tokens/:id
DELETE /api/v1/tokens/:id
```

#### Trading Pairs
```
GET  /api/v1/pairs
POST /api/v1/pairs
PUT  /api/v1/pairs/:id/status
```

#### Blockchains
```
GET  /api/v1/blockchains
POST /api/v1/blockchains
PUT  /api/v1/blockchains/:id
PUT  /api/v1/blockchains/:id/status
```

#### Fees
```
GET  /api/v1/fees
POST /api/v1/fees
PUT  /api/v1/fees/:id
```

#### Webhooks
```
GET  /api/v1/webhooks
POST /api/v1/webhooks
POST /api/v1/webhooks/:id/test
DELETE /api/v1/webhooks/:id
```

#### Notifications
```
GET  /api/v1/notifications
PUT  /api/v1/notifications/:id/read
POST /api/v1/notifications/send
POST /api/v1/notifications/broadcast
```

#### Audit Logs
```
GET  /api/v1/audit-logs
POST /api/v1/audit-logs/export
```

#### Sessions
```
GET  /api/v1/sessions
DELETE /api/v1/sessions/:id
DELETE /api/v1/sessions
```

#### Feature Flags
```
GET  /api/v1/feature-flags
POST /api/v1/feature-flags
PUT  /api/v1/feature-flags/:id
DELETE /api/v1/feature-flags/:id
```
⚠️ All 4 handlers return empty arrays / mock responses — not wired to any real flag store.

#### IP Whitelist
```
GET  /api/v1/ip-whitelist
POST /api/v1/ip-whitelist
DELETE /api/v1/ip-whitelist/:id
```

#### Support Tickets
```
GET  /api/v1/tickets
GET  /api/v1/tickets/:id
POST /api/v1/tickets
PUT  /api/v1/tickets/:id/status
POST /api/v1/tickets/:id/messages
PUT  /api/v1/tickets/:id/assign
```

#### White Labels
```
GET  /api/v1/white-labels
POST /api/v1/white-labels
PUT  /api/v1/white-labels/:id
DELETE /api/v1/white-labels/:id
```

#### Stats
```
GET /api/v1/stats
```

#### Bots & Bot-Tiers
```
GET  /api/v1/bots
GET  /api/v1/bots/:id
POST /api/v1/bots
PUT  /api/v1/bots/:id
DELETE /api/v1/bots/:id
PUT  /api/v1/bots/:id/status
GET  /api/v1/bots/:id/stats
GET  /api/v1/bots/tiers
POST /api/v1/bots/tiers
PUT  /api/v1/bots/tiers/:id
DELETE /api/v1/bots/tiers/:id
```

#### Bot-Clients
```
GET  /api/v1/bots-clients
GET  /api/v1/bots-clients/:id
POST /api/v1/bots-clients
PUT  /api/v1/bots-clients/:id
DELETE /api/v1/bots-clients/:id
PUT  /api/v1/bots-clients/:id/status
```

#### Project Teams
```
GET  /api/v1/project-teams
GET  /api/v1/project-teams/:id
POST /api/v1/project-teams
PUT  /api/v1/project-teams/:id
DELETE /api/v1/project-teams/:id
GET  /api/v1/project-teams/:id/members
POST /api/v1/project-teams/:id/members
DELETE /api/v1/project-teams/:id/members/:memberId
```

#### WL Clients (White Label Tenants) ⭐
```
GET  /api/v1/wl-clients
GET  /api/v1/wl-clients/:id
POST /api/v1/wl-clients
PUT  /api/v1/wl-clients/:id
DELETE /api/v1/wl-clients/:id
PUT  /api/v1/wl-clients/:id/status   ← start/stop/pause/resume
```

#### WL Master Wallets ⭐
```
GET  /api/v1/wl-master-wallets
GET  /api/v1/wl-master-wallets/:id
POST /api/v1/wl-master-wallets
PUT  /api/v1/wl-master-wallets/:id
DELETE /api/v1/wl-master-wallets/:id
PUT  /api/v1/wl-master-wallets/:id/status   ← start/stop/pause/resume
```

#### WL User Wallets ⭐
```
GET  /api/v1/wl-user-wallets
GET  /api/v1/wl-user-wallets/:id
POST /api/v1/wl-user-wallets
PUT  /api/v1/wl-user-wallets/:id
DELETE /api/v1/wl-user-wallets/:id
PUT  /api/v1/wl-user-wallets/:id/status   ← start/stop/pause/resume
```

#### WL Bots ⭐
```
GET  /api/v1/wl-bots
GET  /api/v1/wl-bots/:id
POST /api/v1/wl-bots
PUT  /api/v1/wl-bots/:id
DELETE /api/v1/wl-bots/:id
PUT  /api/v1/wl-bots/:id/status   ← start/stop/pause/resume
```

#### WL Bot-Clients ⭐
```
GET  /api/v1/wl-bots-clients
GET  /api/v1/wl-bots-clients/:id
POST /api/v1/wl-bots-clients
PUT  /api/v1/wl-bots-clients/:id
DELETE /api/v1/wl-bots-clients/:id
PUT  /api/v1/wl-bots-clients/:id/status   ← start/stop/pause/resume
```

#### WL Project Teams ⭐
```
GET  /api/v1/wl-project-teams
GET  /api/v1/wl-project-teams/:id
POST /api/v1/wl-project-teams
PUT  /api/v1/wl-project-teams/:id
DELETE /api/v1/wl-project-teams/:id
```

#### Master Wallets (system-wide)
```
GET  /api/v1/master-wallets
GET  /api/v1/master-wallets/:id
POST /api/v1/master-wallets
PUT  /api/v1/master-wallets/:id
DELETE /api/v1/master-wallets/:id
GET  /api/v1/master-wallets/:id/balance
POST /api/v1/master-wallets/:id/transfer
```

#### User Wallets
```
GET  /api/v1/user-wallets
GET  /api/v1/user-wallets/:id
POST /api/v1/user-wallets
PUT  /api/v1/user-wallets/:id
DELETE /api/v1/user-wallets/:id
GET  /api/v1/user-wallets/:id/balance
```

#### Admins
```
GET  /api/v1/admins
GET  /api/v1/admins/:id
PUT  /api/v1/admins/:id
DELETE /api/v1/admins/:id
POST /api/v1/admins/:id/suspend
POST /api/v1/admins/:id/activate
```

#### Workflows
```
GET  /api/v1/workflows
POST /api/v1/workflows
PUT  /api/v1/workflows/:id
DELETE /api/v1/workflows/:id
```

#### Approval Requests
```
GET  /api/v1/approval-requests
POST /api/v1/approval-requests/:id/approve
POST /api/v1/approval-requests/:id/reject
```

#### Backups
```
GET  /api/v1/backups
POST /api/v1/backups
POST /api/v1/backups/:id/restore
DELETE /api/v1/backups/:id
```

#### Knowledge Base
```
GET  /api/v1/knowledge-base
GET  /api/v1/knowledge-base/:id
POST /api/v1/knowledge-base
PUT  /api/v1/knowledge-base/:id
DELETE /api/v1/knowledge-base/:id
```

#### Archival
```
GET  /api/v1/archival/policies
POST /api/v1/archival/policies
PUT  /api/v1/archival/policies/:id
DELETE /api/v1/archival/policies/:id
POST /api/v1/archival/policies/:id/run
GET  /api/v1/archival/records
```

#### Reports
```
GET  /api/v1/reports/configs
POST /api/v1/reports/configs
GET  /api/v1/reports
POST /api/v1/reports/generate
```

#### SLA
```
GET  /api/v1/sla/policies
POST /api/v1/sla/policies
PUT  /api/v1/sla/policies/:id
DELETE /api/v1/sla/policies/:id
GET  /api/v1/sla/reports
POST /api/v1/sla/reports/generate
```

### 3.2 SuperAdmin — What IS and IS NOT Implemented

#### ✅ FULLY IMPLEMENTED
- Users / KYC / Transactions / Withdrawals / Tokens / Pairs / Blockchains / Fees CRUD
- Webhooks CRUD + test
- Notifications / broadcast
- Audit logs + export
- Sessions (revoke one / all)
- Feature flags CRUD (but all stub handlers)
- IP whitelist
- Tickets CRUD + assign
- White labels CRUD
- Bots CRUD + tiers + bot-clients + status
- Project teams CRUD + members
- **WL clients** CRUD + status
- **WL master-wallets** CRUD + status
- **WL user-wallets** CRUD + status
- **WL bots** CRUD + status
- **WL bots-clients** CRUD + status
- **WL project-teams** CRUD
- Master wallets CRUD + balance + transfer
- User wallets CRUD + balance
- Admins CRUD + suspend/activate
- Workflows CRUD
- Approval requests
- Backups CRUD + restore
- Knowledge base CRUD
- Archival policies + run
- Reports configs + generate
- SLA policies + reports + generate

#### ❌ NOT IMPLEMENTED (stubs / missing)
- **2FA enable/disable** → returns `{"message": "2FA enabled"}` hardcoded, no TOTP
- **Feature flags** → all 4 handlers return empty arrays / mock, no real flag store
- **Feature-level WL product control** → no per-feature permission toggle (e.g. disable
  only WL-user-wallet swap, but keep send). The `/status` endpoints exist but affect
  the entire WL product, not individual fetchers/permissions within it.
- **Individual fetcher toggles for WL products** → no granular control per fetcher
  (e.g. can't disable only the "getBalance" fetcher for a WL client while keeping
  "createWallet"). Only entity-level status.
- **Pause/resume of WL product permissions** → not implemented; only `/status`
  toggle (binary on/off, not pause/resume).
- **WL product-specific fee config per WL client** → not implemented
- **WL bot-specific tiers/config override per WL client** → not implemented
- **Coin/token listing approval workflow per WL client** → WL product has no
  listing-approval sub-workflow
- **SLA/reports generation** → returns `"SLA report generation started"` stub

### 3.3 SuperAdmin — Missing / Gaps

1. **Web frontend drastically incomplete:** 14 of 19 page files are **2-line stubs**
   (APIKeys, Admins, AuditLogs, Blockchains, Fees, KnowledgeBase, Reports, Settings,
   System, Tickets, Tokens, TradingPairs, Webhooks, WhiteLabels, Withdrawals,
   Workflows). Only Dashboard / Users / KYC / Transactions / Security / Login have
   real UI. The 2008-line `api.ts` defines many methods not consumed by any page.
2. **Port mismatch:** web targets `:9090`, backend defaults to **`:8082`** → won't
   connect without env override.
3. **Feature flag handlers are stubs** — CRUD endpoints exist but all return empty
   data; no real feature-toggle store.
4. **No granular WL fetcher-level permissions** — only entity status (on/off); no
   per-fetcher/per-permission pause/resume/start/stop within a WL product.
5. **No WL product-specific config overrides** (fees, gas, RPC endpoints per WL
   client).
6. **No backend route for** `pauseToken` / `unpauseToken` / `resumeTradingPair`
   methods defined in `api.ts` (L479-484, 590-591) — these call
   `/api/v1/tokens/{id}/pause`, `/api/v1/tokens/{id}/unpause`,
   `/api/v1/pairs/{id}/resume` but **no such routes exist in main.go**.

---

## 4. adminPanel (`frontend/admin_panel` + Go backends) — Port 8080/—

### 4.1 Frontend Pages & Intended Endpoints

The React panel calls `/api/v1/admin/*` and `/api/v1/2fa/*`:

| Page | Intended endpoints | Backend match? |
|------|-------------------|---------------|
| Dashboard | `admin/analytics/{dashboard,revenue,volume}` | ❌ No match |
| Users | `admin/users`, `admin/transactions` | Partial (rbac_admin_service) |
| Transactions | `admin/transactions`, `admin/transactions/stats` | Partial |
| Analytics | `admin/analytics/dashboard` etc | ❌ |
| Fees | `admin/fees` | ✅ Partial |
| Integrations | `admin/integrations` | Partial |
| Compliance | `admin/compliance/{aml,tax,reports,stats}` | ❌ |
| Notifications | `admin/notifications`, `broadcast`, `read-all`, `stats` | ❌ |
| Security | `admin/security/{ip-rules,stats}`, `2fa/*` | ❌ |
| Support | `admin/support/{tickets,stats}` | ❌ |
| Bots | `admin/bots` | Partial (rbac_admin_service) |
| **Bridges** | `admin/bridges` | ❌ |
| **Chains** | `admin/chains` | ✅ Partial |
| **DEXs** | `admin/dexs` | ❌ |
| **MarketMaker** | `admin/market-makers` | ❌ |
| **Pools** | `admin/pools` | ❌ |
| **Treasury** | `admin/treasury`, `admin/treasury/stats` | ❌ |

### 4.2 Candidate Backends

**`go/admin_service`** (`/api/v1/*`, no `/admin` segment, port 8080):
login, users, blockchains, tokens, pairs, white-labels, withdrawals, analytics,
activities, bulk ops, CSV exports, fees, api-keys. → **Frontend hits wrong path** —
prefixes `/api/v1/admin/` but this backend has **no `/admin` segment**.

**`go/rbac_admin_service`** (`/api/v1/admin/*`, port 8080/—):
users(search), kyc, transactions, pairs, fees, blockchains, bots, bot-tiers,
api-keys, stats. → **Partial match** — covers Bots/Chains/Users but no Bridges/DEXs/
Pools/Treasury/MarketMaker/Notifications/Compliance/Support/Security/2fa.

**`api_gateway/rest_api/tiger_admin_api.go`** (`/api/v1/admin/*`):
blockchains, fees (configs/addresses/collections), bots (tiers/instances), external
connections, listings. → **Different subset**.

### 4.3 adminPanel — Missing / Gaps

1. **No single backend implements the full `/api/v1/admin/*` surface.** Pages for
   Bridges, DEXs, Pools, Treasury, MarketMaker, 2FA, Notifications, Compliance,
   Support, Security have **no matching Go route** anywhere.
2. **Route prefix mismatch:** the main intended backend (`admin_service`) serves
   `/api/v1/*` — the frontend prefixes `/api/v1/admin/*` → 404s.
3. **`2fa/*`** endpoints served only by `admin/go` (port 9093), not by any
   admin_panel backend.
4. **RBAC:** `admin_service` has no role middleware beyond login token — admin vs
   superadmin distinction not enforced.
5. **Compliance, Support, Security pages** call endpoints that don't exist in any
   of the three candidate backends.

---

## 5. White Label Admin (`white_label_admin/`) — Separate Per-Tenant Admin

This is a **per-white-label-client admin panel** (each WL client gets their own
deployment). Go backend (port varies) with 8 web pages.

### 5.1 Full Route Map

```
GET/POST   /admin/users
GET        /admin/users/:id
PUT        /admin/users/:id/status
POST       /admin/users/:id/ban
POST       /admin/users/:id/unban
POST       /admin/users/:id/suspend
GET/POST   /admin/kyc
POST       /admin/kyc/:id/approve
POST       /admin/kyc/:id/reject
GET/POST   /admin/transactions
GET        /admin/transactions/:id
POST       /admin/transactions/:id/flag
POST       /admin/transactions/:id/unflag
GET/POST   /admin/withdrawals
POST       /admin/withdrawals/:id/approve
POST       /admin/withdrawals/:id/reject
POST       /admin/withdrawals/:id/process
GET/POST   /admin/tokens
POST       /admin/tokens
PUT        /admin/tokens/:id
DELETE     /admin/tokens/:id
GET/POST   /admin/pairs
POST       /admin/pairs
PUT        /admin/pairs/:id/status
GET/POST   /admin/blockchains
POST       /admin/blockchains
PUT        /admin/blockchains/:id
PUT        /admin/blockchains/:id/status
GET/POST   /admin/fees
POST       /admin/fees
PUT        /admin/fees/:id
GET/POST   /admin/webhooks
POST       /admin/webhooks
POST       /admin/webhooks/:id/test
DELETE     /admin/webhooks/:id
GET/POST   /admin/notifications
PUT        /admin/notifications/:id/read
POST       /admin/notifications/send
POST       /admin/notifications/broadcast
GET        /admin/audit-logs
POST       /admin/audit-logs/export
GET        /admin/sessions
DELETE     /admin/sessions/:id
DELETE     /admin/sessions
GET/POST   /admin/tickets
GET        /admin/tickets/:id
PUT        /admin/tickets/:id/status
POST       /admin/tickets/:id/messages
PUT        /admin/tickets/:id/assign
GET/POST   /admin/white-labels
PUT        /admin/white-labels/:id
DELETE     /admin/white-labels/:id
GET        /admin/stats
POST       /admin/logout
POST       /admin/change-password
POST       /admin/2fa/enable
POST       /admin/2fa/disable
GET        /admin/admins
GET        /admin/admins/:id
PUT        /admin/admins/:id
DELETE     /admin/admins/:id
POST       /admin/admins/:id/suspend
POST       /admin/admins/:id/activate
GET/POST   /admin/workflows
PUT        /admin/workflows/:id
DELETE     /admin/workflows/:id
GET        /admin/approval-requests
POST       /admin/approval-requests/:id/approve
POST       /admin/approval-requests/:id/reject
GET        /admin/backups
POST       /admin/backups
POST       /admin/backups/:id/restore
DELETE     /admin/backups/:id
GET        /admin/knowledge-base
GET        /admin/knowledge-base/:id
POST       /admin/knowledge-base
PUT        /admin/knowledge-base/:id
DELETE     /admin/knowledge-base/:id
GET        /admin/archival/policies
POST       /admin/archival/policies
PUT        /admin/archival/policies/:id
DELETE     /admin/archival/policies/:id
POST       /admin/archival/policies/:id/run
GET        /admin/archival/records
GET        /admin/reports/configs
POST       /admin/reports/configs
GET        /admin/reports
POST       /admin/reports/generate
GET        /admin/sla/policies
POST       /admin/sla/policies
PUT        /admin/sla/policies/:id
DELETE     /admin/sla/policies/:id
GET        /admin/sla/reports
POST       /admin/sla/reports/generate
GET/POST   /admin/integrations
PUT        /admin/integrations/:id
DELETE     /admin/integrations/:id
POST       /admin/integrations/:id/test
```

### 5.2 White Label Admin — Missing / Gaps

1. **Web pages (8):** Dashboard, Users, KYC, Transactions, Fees, Settings, Tokens,
   Withdrawals. No pages for: Webhooks, Notifications, Audit logs, Sessions, Tickets,
   White labels, Stats, 2FA, Admins, Workflows, Approval requests, Backups,
   Knowledge base, Archival, Reports, SLA, Integrations — even though routes exist.
2. **CRUD only, no granular per-fetcher permissions** — cannot disable specific
   fetchers or features within the WL product (only disable the whole WL client
   via SuperAdmin's `/wl-clients/:id/status`).
3. **No WL product-specific config** — cannot set custom fees/RPCs per WL product
   without SuperAdmin-level override.

---

## 6. MasterAdmin Management (`master_admin_management/`) — Separate Tier

This is a **MasterAdmin control plane** (port 8082). 8 web pages (Android/Rust/C++
clients). Routes mirror SuperAdmin with some differences.

### 6.1 Full Route Map

```
GET/POST   /admin/users              GET /admin/users/:id
PUT        /admin/users/:id/status   POST /admin/users/:id/{ban,unban,suspend}
GET/POST   /admin/kyc               POST /admin/kyc/:id/{approve,reject}
GET/POST   /admin/transactions       GET /admin/transactions/:id
POST       /admin/transactions/:id/{flag,unflag}
GET/POST   /admin/withdrawals        POST /admin/withdrawals/:id/{approve,reject,process}
GET/POST   /admin/tokens             PUT /admin/tokens/:id        DELETE /admin/tokens/:id
GET/POST   /admin/pairs              PUT /admin/pairs/:id/status
GET/POST   /admin/blockchains         POST /admin/blockchains       PUT /admin/blockchains/:id
PUT        /admin/blockchains/:id/status
GET/POST   /admin/fees               PUT /admin/fees/:id
GET/POST   /admin/webhooks           POST /admin/webhooks/:id/test  DELETE /admin/webhooks/:id
GET/POST   /admin/notifications      POST /admin/notifications/{send,broadcast}
PUT        /admin/notifications/:id/read
GET        /admin/audit-logs         POST /admin/audit-logs/export
GET        /admin/sessions           DELETE /admin/sessions/{:id,}
GET/POST   /admin/tickets            PUT /admin/tickets/:id/status
POST       /admin/tickets/:id/{messages,assign}
GET/POST   /admin/feature-flags      PUT /admin/feature-flags/:id
DELETE     /admin/feature-flags/:id
GET/POST   /admin/ip-whitelist       DELETE /admin/ip-whitelist/:id
GET/POST   /admin/white-labels       PUT /admin/white-labels/:id   DELETE /admin/white-labels/:id
GET        /admin/stats
POST       /admin/logout             POST /admin/change-password
POST       /admin/2fa/{enable,disable}
GET/POST   /admin/admins             PUT /admin/admins/:id         DELETE /admin/admins/:id
POST       /admin/admins/:id/{suspend,activate}
```

### 6.2 MasterAdmin — Missing / Gaps

1. **Android:** fully built with bottom nav, fragments, layouts — seems most complete
   Android admin client. But `package com.tigermasteradmin` has no DB/API wiring
   confirmed.
2. **No WL-specific routes** — unlike SuperAdmin, no `/wl-*` routes, no bots/tiers,
   no master-wallets, no workflows, no approval requests, no backups, no archival,
   no reports, no SLA, no integrations.
3. **Rust:** full axum router but all handlers return **hardcoded stub JSON**.
4. **C++:** partial tx processor (sign/verify/broadcast) — transaction-level only.

---

## 7. What Each Role CAN and CANNOT Do — Complete Matrix

### SuperAdmin (port 8082)

| Action | CAN ✅ | CANNOT ❌ | Notes |
|--------|-------|-----------|-------|
| Manage admins | ✅ Create/delete/suspend/activate | | |
| Manage all WL clients | ✅ Full CRUD + status toggle | | |
| Manage WL-MasterWallets | ✅ Full CRUD + status toggle | | |
| Manage WL-UserWallets | ✅ Full CRUD + status toggle | | |
| Manage WL-Bots | ✅ Full CRUD + status toggle | | |
| Manage WL-BotClients | ✅ Full CRUD + status toggle | | |
| Manage WL-ProjectTeams | ✅ Full CRUD | | |
| Manage master-wallets | ✅ Create/update/delete + balance + transfer | Cannot withdraw to external | |
| Manage user-wallets | ✅ Create/update/delete + balance | Cannot withdraw to external | |
| Manage users | ✅ List/ban/unban/suspend | Cannot access private keys | |
| Manage KYC | ✅ Approve/reject | | |
| Manage transactions | ✅ List/flag/unflag | Cannot reverse/rollback on-chain | |
| Manage withdrawals | ✅ Approve/reject/process | Cannot reverse once broadcast | |
| Manage tokens | ✅ Create/update/delete | Cannot pause (no route) | |
| Manage trading pairs | ✅ Create/update/delete + status | Cannot resume (no route) | |
| Manage blockchains | ✅ Add/update + status | | |
| Manage fees | ✅ CRUD | | |
| Feature flags | ✅ CRUD (stub handlers) | Cannot toggle per-WL-product | Backend handlers return empty |
| Bot management | ✅ Full CRUD + tiers + clients | | |
| Project teams | ✅ CRUD + members | | |
| Workflows | ✅ CRUD | | |
| Approval requests | ✅ Approve/reject | | |
| Backups | ✅ Create/restore/delete | | |
| Knowledge base | ✅ CRUD | | |
| Archival | ✅ Policies CRUD + run | | |
| Reports | ✅ Config + generate | | SLA reports stubbed |
| SLA policies | ✅ CRUD | SLA report generation stubbed | |
| Webhooks | ✅ CRUD + test | | |
| Notifications | ✅ Send/broadcast | | |
| Audit logs | ✅ Read + export | | |
| Sessions | ✅ Revoke one/all | | |
| IP whitelist | ✅ CRUD | | |
| Tickets | ✅ CRUD + assign | | |
| White labels (platform) | ✅ CRUD | | |
| **Granular WL fetcher toggles** | ❌ Not implemented | Only entity-level on/off | No per-fetcher permission |
| **WL product per-feature pause/resume** | ❌ Not implemented | Only binary status | |
| **Per-WL-client fee config** | ❌ Not implemented | | |
| **Per-WL-client RPC override** | ❌ Not implemented | | |
| **Real 2FA** | ❌ Stub only | | Returns hardcoded message |
| **Real system restart** | ❌ No route | | `restartService` in api.ts calls absent route |

### Admin (port 9093)

| Action | SHOULD CAN | ACTUALLY CAN | Gap |
|--------|-----------|--------------|-----|
| Admin management | SuperAdmin only | ✅ Only via /admins | Works |
| System config | SuperAdmin only | ✅ Only via /system | Works |
| User management | Admin | ❌ ✅ Any authenticated | **RBAC broken** |
| KYC | Admin | ❌ ✅ Any authenticated | **RBAC broken** |
| Transactions | Admin | ❌ ✅ Any authenticated | **RBAC broken** |
| Withdrawals | Admin | ❌ ✅ Any authenticated | **RBAC broken** |
| Tokens | Admin | ❌ ✅ Any authenticated | **RBAC broken** |
| Pairs | Admin | ❌ ✅ Any authenticated | **RBAC broken** |
| Fees | Admin | ❌ ✅ Any authenticated | **RBAC broken** |
| Feature flags | SuperAdmin? | ❌ ✅ Any authenticated | **RBAC broken** |
| Audit logs | SuperAdmin? | ❌ ✅ Any authenticated | **RBAC broken** |
| Master wallet | SuperAdmin? | ❌ ✅ Any authenticated | **RBAC broken** |
| Billing | SuperAdmin? | ❌ ✅ Any authenticated | **RBAC broken** |
| Compliance GDPR | SuperAdmin? | ❌ ✅ Any authenticated | **RBAC broken** |
| Notifications | Admin | ✅ Any authenticated | | |
| Tickets | Admin | ✅ Any authenticated | | |
| Integrations | Admin | ✅ Any authenticated | | |
| Brokers | Admin | ✅ Any authenticated | | |
| Institutional | Admin | ✅ Any authenticated | | |
| Multisig | Admin | ✅ Any authenticated | | |
| NFTs | Admin | ✅ Any authenticated | | |
| Crypto cards | SuperAdmin? | ❌ ✅ Any authenticated | **RBAC broken** |
| Liquidity | SuperAdmin? | ❌ ✅ Any authenticated | **RBAC broken** |
| Margin trading | SuperAdmin? | ❌ ✅ Any authenticated | **RBAC broken** |
| P2P merchants | SuperAdmin? | ❌ ✅ Any authenticated | **RBAC broken** |

### adminPanel operator (port 8080)

| Action | CAN ✅ | CANNOT ❌ | Notes |
|--------|-------|-----------|-------|
| Users management | ✅ KYC status update, suspend | Cannot ban? (no /ban route) | |
| Blockchains | ✅ CRUD | | |
| Tokens | ✅ CRUD + status toggle | | |
| Pairs | ✅ CRUD + status toggle | | |
| White labels | ✅ CRUD | | |
| Withdrawals | ✅ Approve/reject + bulk | | |
| Analytics | ✅ Dashboard/revenue/volume | | |
| Activities | ✅ Read | | |
| Bulk ops | ✅ Users/tokens/withdrawals | | |
| CSV exports | ✅ Users/tx/tokens/withdrawals | | |
| Fees | ✅ Trading/withdrawal/deposit update | | |
| API Keys | ✅ Create/list/revoke | | |
| Bridges | ❌ No backend route | | 404 |
| DEXs | ❌ No backend route | | 404 |
| Pools | ❌ No backend route | | 404 |
| Treasury | ❌ No backend route | | 404 |
| MarketMaker | ❌ No backend route | | 404 |
| 2FA | ❌ No backend route (served by admin:9093) | | 404 |
| Notifications | ❌ No backend route | | 404 |
| Compliance | ❌ No backend route | | 404 |
| Support | ❌ No backend route | | 404 |
| Security | ❌ No backend route | | 404 |
| Feature flags | ❌ No route | | |
| Bots | ✅ Partial (via rbac_admin_service) | | |

### White Label Admin operator (per-WL-client)

| Action | CAN ✅ | CANNOT ❌ | Notes |
|--------|-------|-----------|-------|
| Own WL client users | ✅ CRUD + ban/suspend | Access other WL clients | |
| Own WL KYC | ✅ Approve/reject | | |
| Own WL transactions | ✅ List/flag | Cannot reverse | |
| Own WL withdrawals | ✅ Approve/reject/process | | |
| Own WL tokens | ✅ Create/update/delete | | |
| Own WL pairs | ✅ Create/update/status | | |
| Own WL blockchains | ✅ Add/update/status | | |
| Own WL fees | ✅ CRUD | | |
| Own WL webhooks | ✅ CRUD + test | | |
| Own WL notifications | ✅ Send/broadcast | | |
| Own WL audit logs | ✅ Read + export | | |
| Own WL sessions | ✅ Revoke one/all | | |
| Own WL tickets | ✅ CRUD + assign | | |
| Own WL workflows | ✅ CRUD | | |
| Own WL approvals | ✅ Approve/reject | | |
| Own WL backups | ✅ Create/restore/delete | | |
| Own WL knowledge base | ✅ CRUD | | |
| Own WL archival | ✅ Policies + run | | |
| Own WL reports | ✅ Config + generate | | |
| Own WL SLA | ✅ CRUD | SLA report gen stubbed | |
| Own WL integrations | ✅ CRUD + test | | |
| **Granular fetcher toggle** | ❌ No route | | Cannot disable single fetcher |
| **Per-fetcher fee override** | ❌ No route | | | |

---

## 8. Summary — Most Critical Gaps

### Critical (must fix)
1. **Admin RBAC is broken** — `RoleMiddleware`/`AdminMiddleware`/`PermissionMiddleware`
   are dead code; ~200 endpoints accessible to any authenticated admin.
2. **adminPanel has no backend for 9 of 19 pages** — Bridges/DEXs/Pools/Treasury/
   MarketMaker/2FA/Notifications/Compliance/Support/Security all 404.
3. **admin_service route prefix mismatch** — serves `/api/v1/*`, frontend calls
   `/api/v1/admin/*`.

### High
4. **SuperAdmin feature flag handlers are stubs** — CRUD endpoints return empty data.
5. **SuperAdmin: no granular WL product fetcher-level permissions** — only entity
   status (on/off); cannot pause/resume/start individual fetchers within a WL
   product (e.g. disable swap but keep send for a WL client).
6. **SuperAdmin: no per-WL-client fee/RPC/config overrides**.
7. **SuperAdmin web points at wrong port** (`:9090` vs `:8082`).
8. **SuperAdmin: 14 of 19 web pages are 2-line stubs**.
9. **Billing handler in Admin is hardcoded** — no real DB, returns fixed plans.
10. **No real 2FA in SuperAdmin** — enable/disable returns hardcoded message.

### Medium
11. Multiple backend services are in-memory/simulated (archival, chain_management,
    listing_management, report_service, sla, super_admin_service transfers).
12. Base-URL chaos across all admin clients.
13. Rust & C++ backends in all admin apps are pure scaffolding.
14. White Label Admin web only has 8 pages for 50+ routes.

### Recommended Priority Fixes
1. Wire `RoleMiddleware` / `AdminMiddleware` / `PermissionMiddleware` into
   `admin/go/main.go` — highest security impact.
2. Build missing admin_panel backend routes (Bridges/DEXs/Pools/Treasury/MarketMaker/
   2FA/Notifications/Compliance/Support/Security).
3. Fix SuperAdmin web port and complete stub pages.
4. Implement real feature flag store and granular WL fetcher-level permissions.
5. Wire real DB into billing and simulated services.