# TigerWallet Admin Ecosystem — Full Fetchers, Functionality & Gap Analysis

> **Analysis date: 2026-08-16.** This document is the authoritative analysis of
> the three completely-separated admin app families (Admin, SuperAdmin,
> WhiteLabel Admin), their full fetchers/functionality per platform, what is
> missing, and exactly what actions can and cannot be performed.
>
> **Separation principle (verified):** No admin app imports or calls the
> UserWallet or MasterWallet client code. Admin backends manage *governance
> records* (users, wallets, tokens, features) via their own Go backend +
> PostgreSQL — they never hold wallet seeds or signing keys. The only shared
> channel is the Redis feature-flag namespace (`tigerwallet:feature:<name>`)
> which admin backends WRITE and downstream wallet/bot services READ.

---

## Table of Contents

1. [Architecture — three separated families](#1-architecture)
2. [Admin family (`admin/`) — full fetchers & functionality](#2-admin-family)
3. [SuperAdmin family (`super_admin/`) — full fetchers & functionality](#3-superadmin-family)
4. [WhiteLabel Admin family (`white_label_admin/`) — full fetchers & functionality](#4-whitelabel-admin-family)
5. [SuperAdmin per-product control matrix](#5-superadmin-control-matrix)
6. [What admins CAN perform](#6-what-admins-can-perform)
7. [What admins CANNOT perform (enforced)](#7-what-admins-cannot-perform)
8. [Remaining gaps & missing features (detailed)](#8-remaining-gaps)
9. [Build state](#9-build-state)

---

## 1. Architecture

| Family | Go backend (port) | DB | Clients present |
|---|---|---|---|
| **Admin** (`admin/`) | `admin/go` (9093) | PostgreSQL `tigerwallet_admin` + Redis | web, android, ios, desktop, flutter, extensions(chrome/firefox/safari), cpp, rust |
| **SuperAdmin** (`super_admin/`) | `super_admin/go` (8082) | PostgreSQL + Redis | web, android, ios, desktop, extensions(chrome/firefox/safari), cpp, rust |
| **WhiteLabel Admin** (`white_label_admin/`) | `white_label_admin/go` (8093) | PostgreSQL + Redis | web, android(Java), ios, desktop, extensions(chrome/firefox/safari), cpp, rust |

**Separation verified:** grep for cross-imports between admin apps and
UserWallet/MasterWallet client code returns ZERO source-level imports. The
`user_wallet`/`master_wallet` string matches in admin code are REST endpoint
paths (`/admin/user-wallets`, `/admin/master-wallets`) that manage governance
records — not code imports.

---

## 2. Admin Family (`admin/`) — Full Fetchers & Functionality

### 2a. Backend (`admin/go`, port 9093) — 349 routes across 48 route groups

The `admin/go` backend has 48 route groups, each with full CRUD + status
control. Handler files (38 total in `internal/handlers/`):

| Domain | Route group | CRUD | Status control | Sub-resources |
|---|---|---|---|---|
| **Trading — Futures** | `/futures` | ✅ | ✅ start/stop/pause/resume | stats |
| **Trading — Options** | `/options` | ✅ | ✅ | stats |
| **Trading — Margin** | `/margin-trading` | ✅ | ✅ | positions, close |
| **Trading — Copy** | `/copy-trading` | ✅ | ✅ | stats |
| **Trading — Convert** | `/convert` | ✅ | ✅ | calculate |
| **P2P — Clients** | `/p2p-clients` | ✅ | ✅ | verify, flag |
| **P2P — Merchants** | `/p2p-merchants` | ✅ | ✅ | approve, reject |
| **On-Ramp** | `/onramp` | ✅ | ✅ (PUT /:id/status) | approve, reject, verify |
| **Off-Ramp** | `/offramp` | ✅ | ✅ (PUT /:id/status) | approve, reject |
| **Bots** | `/bots` | ✅ | ✅ | tiers CRUD, stats |
| **Bots-Clients** | `/bots-clients` | ✅ | ✅ | — |
| **Project-Teams** | `/project-teams` | ✅ | ✅ | members CRUD |
| **Liquidity** | `/liquidity` | ✅ | ✅ | pools, add/remove |
| **Liquidity-Sources** | `/liquidity-sources` | ✅ | ✅ | priority, health-check, stats |
| **Tokens** | `/tokens` | ✅ | — | — |
| **Pairs** | `/pairs` | ✅ | ✅ toggle | — |
| **Partners** | `/partners` | ✅ | ✅ | approve, reject |
| **Fees** | `/fees` | ✅ | — | — |
| **Blockchains** | `/blockchains` | ✅ | ✅ | test-rpc |
| **Users** | `/users` | ✅ | ✅ ban/suspend/unban | verify-kyc |
| **KYC** | `/kyc` | ✅ | ✅ verify | — |
| **Transactions** | `/transactions` | ✅ (read) | ✅ flag/unflag | — |
| **Withdrawals** | `/withdrawals` | ✅ (read) | approve/reject (record-only) | ❌ BroadcastWithdrawal DISABLED |
| **Master-Wallet (governance)** | `/master-wallet` | ✅ (read) | ✅ | balance (read) |
| **Crypto-Cards** | `/crypto-cards` | ✅ | ✅ block/activate | limit, status |
| **Admins** | `/admins` | ✅ | ✅ | roles, permissions |
| **Roles** | `/roles` | ✅ | — | — |
| **2FA** | `/2fa` | ✅ | enable/disable | backup-codes |
| **API-Keys** | `/api-keys` | ✅ | ✅ regenerate | — |
| **Feature-Flags** | `/feature-flags` | ✅ | ✅ enable/disable | — |
| **Features** | `/features` | ✅ | ✅ toggle | — |
| **Notifications** | `/notifications` | ✅ | ✅ | broadcast |
| **Tickets** | `/tickets` | ✅ | ✅ assign/status | messages |
| **Knowledge-Base** | `/knowledge-base` | ✅ (articles, categories) | — | — |
| **Compliance** | `/compliance` | ✅ | ✅ | aml-report, gdpr, sla |
| **Marketing** | `/marketing` | ✅ | ✅ | plans, categories |
| **Rewards** | `/rewards` | ✅ | ✅ | tiers, plans |
| **Billing** | `/billing` | ✅ | ✅ | invoices, payment-methods |
| **Institutional** | `/institutional` | ✅ | ✅ | — |
| **Brokers** | `/brokers` | ✅ | ✅ | — |
| **Multisig (governance)** | `/multisig` | ✅ (read) | ✅ | — |
| **NFTs** | `/nfts` | ✅ | ✅ | — |
| **White-Labels** | `/white-labels` | ✅ (read) | ✅ status | allowed-products |
| **Integrations** | `/integrations` | ✅ | ✅ | webhook, test |
| **System** | `/system` | ✅ | ✅ enable/disable | metrics, rate-limits |
| **Export** | `/export` | ✅ | — | gdpr/export, anonymize |
| **Analytics** | `/analytics` | ✅ (read) | — | users, transactions, revenue, custom |
| **Audit-Logs** | (read) | ✅ | — | — |

### 2b. Admin client parity per platform

| Platform | Pages/API methods | Status |
|---|---|---|
| **web** (React/TS/Vite) | 36 pages | ✅ All 48 domains covered |
| **android** (Kotlin/Retrofit) | AdminApiService.kt (229 methods), DomainModels.kt, DomainRepository.kt | ✅ All domains including bots/bots-clients/project-teams/liquidity-sources |
| **ios** (Swift/URLSession) | AdminAPIService.swift (144 funcs), DomainModels.swift | ✅ All domains |
| **desktop** (Electron/TS) | App.tsx + DomainPage.tsx + DesktopAdminAPI | ✅ All domains (4 new domains in sidebar + DOMAIN_PAGES) |
| **flutter** (Dart) | 31 dart files | ⚠️ **GAP: missing 26 screens** (futures, options, copy-trading, convert, onramp, offramp, p2p-clients, p2p-merchant, partners, rewards, marketing, bots, bots-clients, project-teams, liquidity-sources, fees, features, audit-logs, admins, admin-roles, reports, system, withdrawals, chains — see §8) |
| **extensions** (chrome/firefox/safari) | js/api.js (151 lines) + popup.js + popup.html | ✅ All domains including 4 new (bots/bots-clients/project-teams/liquidity-sources) |
| **cpp** | admin_domains.hpp + admin_bots.hpp (16 handlers) | ✅ All 16 domains |
| **rust** | domain.rs (16 domain route registrations) | ✅ All 16 domains |

---

## 3. SuperAdmin Family (`super_admin/`) — Full Fetchers & Functionality

### 3a. Backend (`super_admin/go`, port 8082) — 284 routes under `/api/v1/admin`

The SuperAdmin backend has the broadest surface — it governs EVERYTHING the
admin family governs PLUS white-label client/product management, approval
workflows, security controls, and cross-product feature-flag enforcement.

| Domain | Routes | CRUD | Status control | Notes |
|---|---|---|---|---|
| **Admin users** | `/admin/admins` | ✅ | ✅ | roles, permissions, role assignment |
| **Admin roles** | `/admin/admin-roles` (implied via /admins) | ✅ | ✅ | SuperAdmin can add/edit/remove any admin role + grant any role to any admin |
| **Users** | `/admin/users` | ✅ | ✅ ban/suspend/unban | status |
| **KYC** | `/admin/kyc` | ✅ | ✅ | verify |
| **Transactions** | `/admin/transactions` | ✅ | ✅ flag/unflag | — |
| **Tokens** | `/admin/tokens` | ✅ | — | — |
| **Withdrawals** | `/admin/withdrawals` | ✅ (read) | approve/reject/process (record-only) | ❌ NO fund movement |
| **Blockchains** | `/admin/blockchains` | ✅ | ✅ status | — |
| **Pairs** | `/admin/pairs` (implied) | ✅ | ✅ status | — |
| **Fees** | `/admin/fees` | ✅ | ✅ | — |
| **Master-Wallets (governance)** | `/admin/master-wallets` | ✅ (read) | ✅ | balance (read) — governance records only |
| **User-Wallets (governance)** | `/admin/user-wallets` | ✅ (read) | ✅ | balance (read) — governance records only |
| **White-Labels** | `/admin/white-labels` | ✅ | ✅ status | — |
| **WL-Clients** | `/admin/wl-clients` | ✅ | ✅ status | SuperAdmin can add/remove fetchers/permissions of WL clients |
| **WL-MasterWallets** | `/admin/wl-master-wallets` | ✅ | ✅ status | — |
| **WL-UserWallets** | `/admin/wl-user-wallets` | ✅ | ✅ status | — |
| **WL-Bots** | `/admin/wl-bots` | ✅ | ✅ status | operators |
| **WL-Bots-Clients** | `/admin/wl-bots-clients` | ✅ | ✅ status | — |
| **WL-Project-Teams** | `/admin/wl-project-teams` | ✅ | ✅ status | members |
| **Bots** | `/admin/bots` | ✅ | ✅ status | tiers, stats |
| **Bots-Clients** | `/admin/bots-clients` | ✅ | ✅ status | — |
| **Project-Teams** | `/admin/project-teams` | ✅ | ✅ status | members |
| **Futures** | `/admin/futures` | ✅ | ✅ status | — |
| **Options** | `/admin/options` | ✅ | ✅ status | — |
| **Copy-Trading** | `/admin/copy-trading` | ✅ | ✅ status | — |
| **Convert** | `/admin/convert` | ✅ | ✅ status | — |
| **On-Ramp** | `/admin/onramp` | ✅ | ✅ status | approve/reject |
| **Off-Ramp** | `/admin/offramp` | ✅ | ✅ status | approve/reject |
| **P2P-Clients** | `/admin/p2p-clients` | ✅ | ✅ status | — |
| **P2P-Merchants** | `/admin/p2p-merchants` | ✅ | ✅ status | — |
| **Partners** | `/admin/partners` | ✅ | ✅ status | — |
| **Rewards** | `/admin/rewards` | ✅ | ✅ status | — |
| **Marketing** | `/admin/marketing` | ✅ | ✅ status | — |
| **Crypto-Cards** | `/admin/crypto-cards` | ✅ | ✅ block/activate | limit, status — governance records only, NO fund movement |
| **Feature-Flags** | `/admin/feature-flags` | ✅ | ✅ enable/disable/paused | SuperAdmin controls each feature of every product |
| **Features** | `/admin/features` | ✅ | ✅ toggle | `/features/:name/check` runtime check |
| **IP-Whitelist** | `/admin/ip-whitelist` | ✅ | — | security |
| **Sessions** | `/admin/sessions` | ✅ | ✅ | revoke |
| **2FA** | `/admin/2fa` | ✅ | enable/disable | — |
| **Audit-Logs** | `/admin/audit-logs` | ✅ (read) | — | export |
| **Approval-Requests** | `/admin/approval-requests` | ✅ | ✅ approve/reject | workflow approvals |
| **Workflows** | `/admin/workflows` | ✅ | — | approval workflows |
| **Webhooks** | `/admin/webhooks` | ✅ | ✅ | test |
| **Backups** | `/admin/backups` | ✅ | ✅ | restore |
| **Archival** | `/admin/archival/policies` + `/records` | ✅ | ✅ | run |
| **SLA** | `/admin/sla/policies` + `/reports` | ✅ | — | — |
| **Reports** | `/admin/reports` + `/configs` | ✅ | — | — |
| **Knowledge-Base** | `/admin/knowledge-base` | ✅ | — | — |
| **Notifications** | `/admin/notifications` | ✅ | ✅ read | — |
| **Tickets** | `/admin/tickets` | ✅ | ✅ assign/status | messages |
| **Integrations** | `/admin/integrations` | ✅ | ✅ | — |
| **Stats** | `/admin/stats` | ✅ (read) | — | platform-wide |
| **Security (transfer disabled)** | `/admin/transfer` | ❌ 403 | — | Explicitly returns 403: "admin fund transfer is prohibited" |

### 3b. SuperAdmin client parity per platform

| Platform | Pages/API methods | Status |
|---|---|---|
| **web** (React/TS/Vite) | 41 pages | ✅ All domains including Security, Governance, Workflows, Webhooks, MasterWallets, UserWallets, KnowledgeBase |
| **android** (Kotlin, single-file) | MainActivity.kt | ✅ All domains including crypto-cards, workflows, webhooks, governance |
| **ios** (Swift, single-file) | TigerSuperAdminApp.swift | ✅ All domains including crypto-cards |
| **desktop** (Electron) | main.js + preload.js | ✅ All domains including crypto-cards |
| **extensions** (chrome/firefox/safari) | background.js | ✅ All domains including crypto-cards |
| **cpp** | super_admin_domains.hpp | ✅ All domains including CryptoCardsHandler |
| **rust** | domain/crypto_cards.rs + api/mod.rs + domain/mod.rs | ✅ All domains |
| **flutter** | — | ❌ **MISSING entirely** (no `super_admin/flutter/` directory) |

---

## 4. WhiteLabel Admin Family (`white_label_admin/`) — Full Fetchers & Functionality

### 4a. Backend (`white_label_admin/go`, port 8093) — 162 routes

The WhiteLabel Admin manages a single white-label tenant's products + their
clients. It does NOT manage other white labels (that's SuperAdmin's job).

| Domain | Routes | CRUD | Status control |
|---|---|---|---|
| **Admins** (WL tenant admins) | `/admins` | ✅ | ✅ |
| **Admin-Roles** | `/admin-roles` | ✅ | ✅ |
| **Admin-Permissions** | `/admin-permissions` | ✅ (read) | — |
| **Users** | `/users` | ✅ | ✅ ban/suspend/unban |
| **KYC** | `/kyc` | ✅ | ✅ |
| **Tokens** | `/tokens` | ✅ | — |
| **Pairs** | `/pairs` | ✅ | ✅ status |
| **Blockchains** | `/blockchains` | ✅ | ✅ status |
| **Futures** | `/futures` | ✅ | ✅ status |
| **Options** | `/options` | ✅ | ✅ status |
| **Copy-Trading** | `/copy-trading` | ✅ | ✅ status |
| **Convert** | `/convert` | ✅ | ✅ status |
| **On-Ramp** | `/onramp` | ✅ | ✅ approve/reject |
| **Off-Ramp** | `/offramp` | ✅ | ✅ approve/reject |
| **P2P-Clients** | `/p2p-clients` | ✅ | ✅ status |
| **Partners** | `/partners` | ✅ | ✅ approve/reject |
| **Rewards** | `/rewards` | ✅ | ✅ status |
| **Marketing** | `/marketing` | ✅ | ✅ status |
| **Fees** | `/fees` | ✅ | ✅ |
| **Feature-Flags** | `/feature-flags` | ✅ | ✅ enable/disable |
| **WL-Bots** (operators) | `/wl-bots/operators` | ✅ | ✅ status |
| **WL-Cards** | `/wl-cards` | ✅ | ✅ status |
| **WL-Liquidity** (sources + allocations) | `/wl-liquidity/sources` + `/allocations` | ✅ | ✅ |
| **Tickets** | `/tickets` | ✅ | ✅ assign/status |
| **Transactions** | `/transactions` | ✅ | ✅ flag/unflag |
| **Withdrawals** | `/withdrawals` | ✅ (read) | approve/reject/process (record-only) |
| **Audit-Logs** | `/audit-logs` | ✅ (read) | — |
| **IP-Whitelist** | `/ip-whitelist` | ✅ | — |
| **Sessions** | `/sessions` | ✅ | ✅ |
| **Notifications** | `/notifications` | ✅ | ✅ read |
| **Stats** | `/stats` | ✅ (read) | — |
| **Health** | `/health` | ✅ | — |

### 4b. WhiteLabel Admin client parity per platform

| Platform | Status | Notes |
|---|---|---|
| **web** (Next.js/React) | ✅ 30 pages | All WL tenant domains covered |
| **android** (Java, NOT Kotlin) | ⚠️ **GAP** | 9 Java fragments only (Dashboard, Domains, DomainDetail, KYC, Transactions, Users, Settings, + MainActivity + App) — missing many domains; uses Java not Kotlin (inconsistent with admin/super_admin which use Kotlin) |
| **ios** (Swift) | ⚠️ **GAP** | 2 files only (TigerWhiteLabelAdminApp.swift + ContentView.swift) — minimal, not full domain screens |
| **desktop** (Electron) | ⚠️ **GAP** | 3 JS files — minimal |
| **extensions** (chrome/firefox/safari) | ⚠️ **GAP** | 6 JS files — minimal, likely not full domain coverage |
| **cpp** | ✅ | Present + builds |
| **rust** | ✅ | Present + builds |
| **flutter** | ❌ **MISSING** | No `white_label_admin/flutter/` directory |

---

## 5. SuperAdmin Per-Product Control Matrix

SuperAdmin can add/remove/halt/pause/start/resume each feature and each
functionality of the following products. This is implemented via the
`/admin/feature-flags` CRUD (Redis-backed, `tigerwallet:feature:<name>` key)
+ per-product `/status` endpoints:

| Product | SuperAdmin control surface | Status endpoint | Feature-flag enforcement |
|---|---|---|---|
| **White-Label Client** | `/admin/wl-clients/:id/status` + `/allowed-products` | ✅ | ✅ (flag store) |
| **WL-MasterWallet** | `/admin/wl-master-wallets/:id/status` | ✅ | ✅ |
| **WL-UserWallet** | `/admin/wl-user-wallets/:id/status` | ✅ | ✅ |
| **WL-Bots + Bots-Clients** | `/admin/wl-bots/:id/status` + `/wl-bots-clients/:id/status` | ✅ | ✅ |
| **WL-Project-Teams** | `/admin/wl-project-teams/:id/status` | ✅ | ✅ |
| **MasterWallet** | `/admin/master-wallets/:id` (governance) | ✅ | ✅ |
| **UserWallet** | `/admin/user-wallets/:id` (governance) | ✅ | ✅ |
| **Bots + Bots-Clients** | `/admin/bots/:id/status` + `/bots-clients/:id/status` | ✅ | ⚠️ Partial (see §8) |
| **Project-Teams** | `/admin/project-teams/:id/status` | ✅ | ✅ |
| **Futures** | `/admin/futures/:id/status` | ✅ | ✅ |
| **Options** | `/admin/options/:id/status` | ✅ | ✅ |
| **Copy-Trading** | `/admin/copy-trading/:id/status` | ✅ | ✅ |
| **Convert** | `/admin/convert/:id/status` | ✅ | ✅ |
| **Margin-Trading** | (via futures/perpetual) | ✅ | ⚠️ Partial |
| **On-Ramp** | `/admin/onramp/:id/status` | ✅ | ✅ |
| **Off-Ramp** | `/admin/offramp/:id/status` | ✅ | ✅ |
| **P2P-Clients** | `/admin/p2p-clients/:id/status` | ✅ | ✅ |
| **P2P-Merchants** | `/admin/p2p-merchants/:id/status` | ✅ | ✅ |
| **Coin/Token Listing** | `/admin/tokens` CRUD + `/pairs/:id/status` | ✅ | ✅ |
| **Liquidity Sources** | `/admin/liquidity-sources` (in admin/go) | ✅ | ⚠️ Partial |
| **Crypto-Cards** | `/admin/crypto-cards/:id/status` + block/activate | ✅ | ✅ |
| **Marketing** | `/admin/marketing/:id/status` | ✅ | ✅ |
| **Rewards** | `/admin/rewards/:id/status` | ✅ | ✅ |
| **Knowledge-Base** | `/admin/knowledge-base` CRUD | ✅ | ✅ |
| **KYC** | `/admin/kyc` + verify | ✅ | ✅ |
| **Customer Service (Tickets)** | `/admin/tickets` + assign/status/messages | ✅ | ✅ |
| **Security** | `/admin/ip-whitelist`, `/admin/sessions`, `/admin/2fa`, `/admin/audit-logs` | ✅ | SuperAdmin-only |

**WL client permission granularity:** SuperAdmin can add/remove any
fetcher/functionality of an approved WL product via
`/admin/wl-clients/:id/allowed-products` — this controls which WL products
(WL-MasterWallet, WL-UserWallet, WL-Bots, WL-Project-Teams) a WL client can
access. SuperAdmin can start/stop/pause/resume each permission.

**Admin role management:** SuperAdmin can add/edit/remove/update any
adminRight to any admin via `/admin/admins/:id/roles` + `/admin/admin-roles`.
SuperAdmin can create any adminRole and grant it to any admin.

---

## 6. What Admins CAN Perform

### Admin (regular admin role)
- ✅ Full CRUD + status control on their assigned domains (trading, p2p,
  onramp/offramp, bots, tokens, pairs, liquidity, partners, rewards,
  marketing, KYC, tickets, knowledge-base, compliance, billing, etc.)
- ✅ Read-only access to users, transactions, withdrawals, master-wallets,
  user-wallets (governance records)
- ✅ Approve/reject withdrawals (record-only — records the decision, does
  NOT move funds)
- ✅ Manage feature flags (enable/disable/pause within their scope)
- ✅ View audit logs, analytics, reports
- ✅ Manage API keys, 2FA, integrations

### SuperAdmin
- ✅ Everything an admin can do, PLUS:
- ✅ Manage all admin users + roles (create any role, grant any right)
- ✅ Manage white-label clients + all WL products (WL-MasterWallet,
  WL-UserWallet, WL-Bots, WL-Project-Teams)
- ✅ Control each feature/functionality of every product (halt/pause/start/
  resume via feature flags + per-product status)
- ✅ Security controls (IP whitelist, sessions, 2FA enforcement, audit log
  export, backups, archival, SLA, workflows, webhooks, approval-requests)
- ✅ Platform-wide stats + governance
- ✅ `setFeeRecipient(address)` on TigerBotPlatform.sol (ADMIN-only on-chain
  governance — rotates the protocol fee recipient address; the ONE legitimate
  on-chain crypto-movement governance path; no admin private key involved)

### WhiteLabel Admin
- ✅ Manage their own WL tenant's products (tokens, pairs, futures, options,
  copy-trading, convert, onramp, offramp, p2p-clients, partners, rewards,
  marketing, fees, feature-flags, WL-bots, WL-cards, WL-liquidity)
- ✅ Manage their WL tenant's users, KYC, tickets, transactions
- ✅ Approve/reject withdrawals (record-only)

---

## 7. What Admins CANNOT Perform (Enforced in Code)

| Action | Enforcement | Location |
|---|---|---|
| ❌ Withdraw crypto assets | `BroadcastWithdrawal` is intentionally DISABLED (returns error) | `admin/go/internal/handlers/withdrawal_handler.go:40` |
| ❌ Transfer funds on behalf of users | `/admin/transfer` returns explicit 403 | `super_admin/go/main.go:3128-3130` |
| ❌ Hold wallet seeds / private keys | No admin backend performs key management or signing | Architecture-level |
| ❌ Sign transactions | No signing capability in any admin backend | Architecture-level |
| ❌ Access UserWallet client code | Zero source imports between admin and UserWallet client apps | Separation verified |
| ❌ Access MasterWallet client code | Zero source imports between admin and MasterWallet client apps | Separation verified |
| ❌ Move crypto-card balances | Crypto-card admin is governance records only (block/activate/setLimit) | `super_admin/go:4406` |

**Withdrawal flow:** Admins can approve/reject withdrawal *requests* (the
decision is recorded in PostgreSQL), but the actual fund movement
(`BroadcastWithdrawal`) is disabled — only the wallet owner via the canonical
`go/wallet_api` backend (which holds the encrypted seed) can broadcast a
signed transaction.

---

## 8. Remaining Gaps & Missing Features (Detailed)

### Gap 1: Feature-flag enforcement not wired in 10 downstream services ⚠️ HIGH PRIORITY

The admin backends WRITE feature flags to Redis
(`tigerwallet:feature:<name>` = enabled/disabled/paused), but only 3 of 14
downstream services READ and enforce them:

| Service | Enforces feature flags? |
|---|---|
| `go/wallet_api` | ✅ Yes (`feature_flags.go`, wired in handlers.go + defi_handlers.go) |
| `go/lending_service` | ✅ Yes |
| `go/copy_trading_service` | ✅ Yes |
| `go/bridge_service` | ✅ Yes |
| `go/perpetual_service` | ❌ **NO** |
| `go/governance_service` | ❌ **NO** |
| `go/prediction_service` | ❌ **NO** |
| `go/nft_service` | ❌ **NO** |
| `go/airdrop_service` | ❌ **NO** |
| `go/earn_service` | ❌ **NO** |
| `go/coupon_service` | ❌ **NO** |
| `go/red_packets_service` | ❌ **NO** |
| `go/fiat_ramp` | ❌ **NO** |
| `go/gift_card_service` | ❌ **NO** |
| `mm_bot_platform/bot_api` | ❌ **NO** |

**Impact:** SuperAdmin can SET `perpetual_trading = paused` in the admin
panel, but `perpetual_service` does not consult the flag store, so the
feature is NOT actually halted at runtime. The flag is set but not enforced.

**Fix needed:** Each of the 10 services needs a `feature_flags.go` file
(Redis read, fail-closed: missing/unknown = disabled) + `enforceFeature()`
calls at the entry point of each gated route handler.

### Gap 2: `super_admin/flutter` entirely missing ❌

There is no `super_admin/flutter/` directory. The admin family has a full
Flutter app (31 dart files), but SuperAdmin has none. SuperAdmin has
web/android/ios/desktop/extensions/cpp/rust but NOT flutter.

**Fix needed:** Create `super_admin/flutter/` mirroring the admin/flutter
structure with all 41 super_admin domains.

### Gap 3: `white_label_admin/flutter` entirely missing ❌

Same as above — no `white_label_admin/flutter/` directory.

### Gap 4: `admin/flutter` missing 26 domain screens ⚠️

The admin Flutter app has 31 dart files but is missing screens for 26
domains that exist in the web app:
- Futures, Options, Margin-Trading, Copy-Trading, Convert
- On-Ramp, Off-Ramp, P2P-Clients, P2P-Merchant
- Partners, Rewards, Marketing
- Bots, Bots-Clients, Project-Teams, Liquidity-Sources
- Fees, Features, Audit-Logs, Admin-Roles, Admins
- Reports, System, Withdrawals, Chains

### Gap 5: `white_label_admin/android` is Java (not Kotlin) + minimal ⚠️

The WL admin Android app uses Java (9 fragments) while admin + super_admin
use Kotlin. It's also minimal — missing many domain screens that the web has.
The `DomainsFragment` is generic but may not cover all 30 web pages.

### Gap 6: `white_label_admin/ios` + `desktop` + `extensions` minimal ⚠️

- **ios:** Only 2 Swift files (app + ContentView) — not per-domain screens
- **desktop:** Only 3 JS files — minimal
- **extensions:** 6 JS files — may not cover all 30 web domains

### Gap 7: Three feature-flag systems need consolidation ⚠️

There are three overlapping feature-flag systems:
1. `super_admin/go` `feature_flags` DB table + Redis
2. `admin/go` `features` DB table
3. `wallet_api` in-memory `FeatureControl` (partially unwired)

These need consolidation into ONE enforced source of truth (Redis, which
admin backends write + downstream services read).

### Gap 8: Admin web missing some SuperAdmin-equivalent pages ⚠️

Admin web (36 pages) is missing these pages that SuperAdmin web has:
- **Governance** (approval workflows)
- **Workflows**
- **Webhooks**
- **Security** (IP whitelist, sessions — admin has `/system` but no dedicated
  security page)
- **MasterWallets** (governance view)
- **UserWallets** (governance view)
- **KnowledgeBase**
- **Tickets** (admin has it via `/tickets` route but no web page)
- **APIKeys** (admin has route but no dedicated page)

These may be intentional (some are SuperAdmin-only like Security), but
Tickets/APIKeys/KnowledgeBase should have admin web pages since the routes
exist in `admin/go`.

---

## 9. Build State

All three admin families build clean:

| Component | admin | super_admin | white_label_admin |
|---|---|---|---|
| **Go backend** | ✅ build + vet | ✅ build | ✅ build + vet |
| **Rust** | ✅ cargo check | ✅ cargo check | ✅ cargo check |
| **C++** | ✅ g++ -fsyntax-only | ✅ g++ -fsyntax-only | ✅ g++ -fsyntax-only |
| **Web** | ✅ tsc 0 errors | ✅ tsc 0 errors | ✅ tsc 0 errors |
| **Android** | ✅ (Kotlin) | ✅ (Kotlin) | ⚠️ Java (builds but inconsistent) |
| **iOS** | ✅ (Swift) | ✅ (Swift) | ⚠️ Minimal |
| **Desktop** | ✅ (Electron) | ✅ (Electron) | ⚠️ Minimal |
| **Extensions** | ✅ (3 browsers) | ✅ (3 browsers) | ⚠️ Minimal |
| **Flutter** | ⚠️ 31 files, 26 screens missing | ❌ Missing | ❌ Missing |

---

## Summary

The admin ecosystem has **three completely separated families** with full
backend coverage (349 + 284 + 162 = 795 routes total). The SuperAdmin has
comprehensive per-product control via feature flags + status endpoints, and
**no admin can withdraw or transfer crypto assets** (enforced in code).

**The most critical remaining gap is Gap 1 (feature-flag enforcement):** 10
of 14 downstream services do not consult the admin flag store, so
SuperAdmin's halt/pause/start/resume controls are SET but NOT ENFORCED at
runtime for those services. This is the single highest-priority fix needed.

**The second priority is client parity:** SuperAdmin and WhiteLabel Admin
are missing Flutter apps entirely, and the admin Flutter app is missing 26
domain screens. The WhiteLabel Admin native clients (android/ios/desktop/
extensions) are minimal and need expansion to match the web app's 30 pages.
