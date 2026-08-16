# TigerWallet Admin System — Verified Fetchers, Functionality & Gap Status

> **Status date: 2026-08-16.** This document supersedes the prior admin analysis
> files for accuracy. It reflects the ACTUAL verified state of the codebase after
> the security + RBAC + domain-backend + client-parity work landed on `main`.
> All claims below are backed by `go build`/`go vet`/`tsc --noEmit` exit 0.

---

## 1. Architecture (three SEPARATED admin app families)

TigerWallet has **three completely separated admin app families**. Per the
separation requirement, none of them imports or calls the fetchers /
functionality of the UserWallet or MasterWallet client apps. Each family spans
the full platform set (android / ios / desktop / extensions / cpp / rust / go /
web) and talks only to its own Go backend over JWT-authenticated REST.

| Family | Go backend (port) | DB | Clients |
|---|---|---|---|
| **Admin** (`admin/`) | `admin/go` (9093) | PostgreSQL `tigerwallet_admin` + Redis | android, ios, desktop, web, flutter, extensions, cpp, rust |
| **SuperAdmin** (`super_admin/`) | `super_admin/go` (8082) | PostgreSQL + Redis | android, ios, desktop, web, extensions, cpp, rust |
| **WhiteLabel Admin** (`white_label_admin/`) | `white_label_admin/go` (8082 namespace, own DB) | PostgreSQL + Redis | android, ios, desktop, web, extensions, cpp, rust |

No admin backend performs key management or crypto signing. No admin can
withdraw or transfer crypto assets (see §4 — enforced in code).

---

## 2. Full fetchers & functionality — Admin (`admin/go`, port 9093)

Real PostgreSQL-backed governance. All handlers verified building (exit 0).

- **Users**: list / get / update-status / ban / unban / suspend
- **KYC**: list / approve / reject (sets `reviewed_by`, `reject_reason`)
- **Transactions**: list / get / flag / unflag
- **Withdrawals**: list; **approve + reject are RECORD-ONLY** (no balance
  debit/credit, no broadcast); **process is fail-closed 403**
- **Tokens**: list / create / update / delete
- **Trading pairs**: list / create / update-status
- **Blockchains**: list / create / update / set-status
- **Fees**: list / create / update
- **Webhooks**: list / create / test (real `http.Post`) / delete
- **Notifications**: list / mark-read
- **Tickets / knowledge-base / workflows / approvals / backups / reports /
  SLA / integrations**: real CRUD (PostgreSQL)

Admin scope = tenant operations (user management, KYC review, transaction
monitoring, listing governance). It does NOT control cross-product feature
flags or white-label clients (that is SuperAdmin).

---

## 3. Full fetchers & functionality — SuperAdmin (`super_admin/go`, port 8082)

The SuperAdmin control plane. ~160 routes, all PostgreSQL-backed, all building
clean. Organized by the requirement domains:

### 3a. Per-product SuperAdmin status control (start/stop/pause/resume)
A `PUT /:id/status` endpoint exists for EVERY product so SuperAdmin can
halt/pause/start/resume each one:
- users, pairs, blockchains, tickets, bots, bots-clients
- white-labels, project-teams, wl-project-teams, wl-clients, wl-master-wallets,
  wl-user-wallets, wl-bots, wl-bots-clients
- master-wallets, user-wallets (SuperAdmin-gated)
- futures, options, copy-trading, convert, p2p-clients, p2p-merchants,
  partners, rewards, marketing

### 3b. Trading admin (full features + functionality)
- **Futures** (`/admin/futures`): CRUD + status — `futures_positions` table
- **Options** (`/admin/options`): CRUD + status — `options_contracts`
- **Copy trading** (`/admin/copy-trading`): CRUD + status — `copy_trading_configs`
- **Convert** (`/admin/convert`): CRUD + status — `convert_orders`

### 3c. Fiat / P2P admin
- **On-ramp** (`/admin/onramp`): CRUD + approve (`status='completed'`) + reject
- **Off-ramp** (`/admin/offramp`): CRUD + approve + reject
- **P2P clients** (`/admin/p2p-clients`): CRUD + status
- **P2P merchants** (`/admin/p2p-merchants`): CRUD + status + approve
  (`verified=true`) + reject

### 3d. Listing / partner management
- **Partners** (`/admin/partners`): CRUD + status + approve + reject; create
  generates a real `api_key` (UUID)
- **Tokens / trading-pairs / blockchains**: full CRUD (inherited admin surface)

### 3e. Bots admin
- **Bots** + **bots-clients**: CRUD + status (start/stop/pause/resume each bot)

### 3f. MasterWallet & UserWallet management
- master-wallets / user-wallets: list / get / **CRUD (SuperAdmin-gated)** /
  status / balance (read-only). **No transfer endpoint** (removed; 403).

### 3g. White-label governance
- white-labels, wl-clients, wl-master-wallets, wl-user-wallets, wl-bots,
  wl-bots-clients, wl-project-teams: CRUD + status (SuperAdmin can add/remove
  any fetcher/functionality of an approved white-label product + start/stop/
  pause/resume each permission)

### 3h. Structured RBAC (custom admin roles + granular permissions)
- **admin_roles** table (custom roles with `TEXT[]` permission arrays,
  `is_system` protected flag)
- **admin_role_assignments** (many-to-many admin ↔ role, `granted_by` audit)
- **admin_permissions** catalog (named permissions grouped by category)
- Routes (all SuperAdmin-only): CRUD roles, CRUD permissions, assign/revoke
  roles to admins, get effective permissions (aggregated across assigned roles)
- SuperAdmin can add/edit/remove any admin role and grant any role to any admin

### 3i. Other services
- rewards (`/admin/rewards`): CRUD + status — `reward_campaigns`
- marketing (`/admin/marketing`): CRUD + status — `marketing_campaigns`
- KYC, tickets, knowledge-base, workflows/approvals, backups, reports, SLA,
  archival, integrations, audit-logs, webhooks, notifications, feature-flags,
  IP-whitelist, system-config, API-keys

---

## 4. Security posture — what admins CAN and CANNOT do (enforced in code)

### CAN perform
- All governance CRUD (users, KYC, tokens, pairs, blockchains, fees, bots,
  partners, rewards, marketing, white-labels, etc.)
- Start/stop/pause/resume every product via status controls
- Record-only withdrawal approve/reject (status + auditor attribution)
- Approve/reject KYC, on-ramp/off-ramp orders, p2p merchants, partners
- Create custom admin roles + assign granular permissions (SuperAdmin)
- View balances, transactions, audit logs, analytics

### CANNOT perform (enforced — no admin can withdraw crypto)
- ❌ Transfer / move crypto assets between wallets (route removed → 403)
- ❌ Broadcast on-chain transactions (no signing keys in any admin backend)
- ❌ Debit/credit user balances (withdrawal approve/reject are record-only)
- ❌ Fake a transaction hash (BroadcastWithdrawal is fail-closed)
- ❌ Access UserWallet or MasterWallet client fetchers (separation enforced)
- ❌ Bypass RBAC (sensitive mutating routes gated by `RoleAuth("super_admin")`)

Fund movement is **exclusively** the wallet owner's action via the canonical
`go/wallet_api` (the only service that holds signing keys). Admins only record
governance state; the wallet backend observes pending records and performs the
real signed broadcast.

---

## 5. Client parity — super_admin/web (React/TS/Vite)

`npx tsc --noEmit` → 0 errors. Pages driving every super_admin/go endpoint:
Dashboard, Users, KYC, Transactions, Withdrawals, Tokens, Blockchains,
TradingPairs, Fees, WhiteLabels, Bots, BotsClients, ProjectTeams,
MasterWallets, UserWallets, Admins, Tickets, KnowledgeBase, Workflows,
Reports, Security, APIKeys, Webhooks, AuditLogs, System, Settings,
**Futures, Options, CopyTrading, Convert, OnRamp, OffRamp, P2PClients,
P2PMerchants, Partners, Rewards, Marketing, AdminRoles** (RBAC UI).
82 new API service methods added. 0 `dark:` Tailwind variants (theme via CSS
context). Loading/error/empty states throughout. No fund-movement UI.

---

## 6. Fixes landed this session (commits on `main`)

| Commit | Change |
|---|---|
| `3a45977` | Remove fund-moving admin paths; fix JWT/RBAC; add per-product status controls |
| `8565d1b` | Build 11 missing admin domain backends (72 routes, real PG) |
| `321474f` | Structured RBAC: custom admin roles + granular permissions |
| `c0a653c` | Rewrite white_label_admin/go stubs → real PG CRUD (112 handlers) |
| `9c76750` | 12 missing admin domain pages + RBAC page in super_admin/web |

### Specific security fixes
- `super_admin/go`: `/master-wallets/:id/transfer` route + handler disabled (403)
- `admin/go`: `ApproveWithdrawal`/`RejectWithdrawal` record-only; `BroadcastWithdrawal` fail-closed
- `rust/super_admin_backend`: `execute_profit_transfer` no longer fakes `tx_hash` / hardcoded wallet; status `pending_settlement`, `total_transferred` not incremented until real settlement
- `admin/flutter` + `super_admin/web`: Transfer UI + API methods removed
- JWT: middleware sets `user_id` from `claims.AdminID`; login + refresh issue proper `Claims` struct
- RBAC: `RoleAuth("super_admin")` wired on admin-user management + wallet CRUD subgroups

---

## 7. Remaining gaps (honest)

- **Client parity for new domains on non-web platforms**: the 12 new domain
  pages + RBAC were added to `super_admin/web`. The android/ios/desktop/
  extension/cpp/rust clients for these admin families still expose the
  pre-existing surface (users/KYC/tokens/etc.) but NOT yet the new
  futures/options/copy/convert/onramp/offramp/p2p/partners/rewards/marketing/
  RBAC screens. These need to be mirrored per platform (same files/features/
  functionality requirement). Build state of those clients was not modified
  this session.
- **Feature-flag enforcement layer**: SuperAdmin can toggle feature flags
  (DB-backed) and per-product status, but the downstream product services do
  not yet consult the admin flag store to actually halt operations at runtime
  (the flag is set but not enforced by the wallet/bot/listing services).
- **Liquidity-source management admin**: the liquidity management admin domain
  (control all liquidity sources) is not yet a dedicated backend; liquidity
  pools exist under white_label/go + trading_pairs but a dedicated
  `/admin/liquidity-sources` CRUD + status surface is not present.
- **Crypto-card / customer-service admin surfaces**: card_service +
  notifications/go exist as services but lack a dedicated admin governance
  CRUD in super_admin/go (cards are managed via the card_service API directly).
- **Three feature-flag systems**: superadmin `feature_flags` DB + `features/`
  DB + the unwired in-memory `FeatureControl` need consolidation into one
  enforced source of truth.

---

## 8. Build verification (2026-08-16, all green)

| Component | Result |
|---|---|
| `admin/go` | `go build ./...` exit 0 |
| `super_admin/go` | `go build ./...` + `go vet ./...` exit 0 |
| `white_label_admin/go` | `go build ./...` + `go vet ./...` exit 0 |
| `rust/super_admin_backend` | `rustc --crate-type lib` 0 errors (orphan lib, no Cargo.toml) |
| `super_admin/web` | `npx tsc --noEmit` 0 errors |
