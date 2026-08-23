# Platform Admin

The **Platform Admin** app is TigerWallet's day-to-day governance console:
a Go backend (`admin/go`) and a React web UI (`admin/web`) that let
platform administrators manage the entire product surface — users, chains,
tokens, trading verticals, fees, withdrawals, KYC, support, and more — under
a strict role hierarchy where true control-plane powers remain
**SuperAdmin-only**.

## Platforms Covered

| Client | Location |
|---|---|
| Web (React/MUI) | `admin/web` |
| Backend (Go) | `admin/go` |
| Android | `admin/android` |
| iOS | `admin/ios` |
| Desktop | `admin/desktop` |
| Browser extensions | `admin/extensions` |
| C++ core | `admin/cpp` |
| Rust core | `admin/rust` |

## Backend (`admin/go`)

Gin + PostgreSQL (GORM) + Redis. Entry point `main.go` registers ~340 routes
across **46 domain handler files** in `internal/handlers/`. Listens on
`ADMIN_PORT` (default **9093**); health probes `/healthz` and `/readyz`.
Proxy-aware with optional `TRUSTED_PROXIES`.

Handler domains (one file each in `internal/handlers/`):

admins, analytics, API keys, auto-approvals, billing, blockchains,
bots/clients, bots, brokers, compliance, convert, copy-trading, crypto cards,
exports, features, fees, futures, institutional, integrations, knowledge
base, KYC, liquidity, liquidity sources, margin trading, marketing,
MasterWallet, multisig, NFTs, notifications, off-ramps, on-ramps, options,
P2P clients, P2P merchants, trading pairs, partners, project teams, RBAC,
rewards, SuperAdmin bridge, support, tokens, 2FA, users, white labels,
withdrawals.

## Web UI (`admin/web`)

React/MUI console with **36 pages** (`src/pages/`), one (or more) per
governance domain listed above. Calls the Go API via the `API_URL` env var
(default `http://localhost:9093/api/v1`).

## RBAC & Middleware (`admin/go/internal/middleware/auth.go`)

- **`AuthMiddleware`** — validates the JWT from the `Authorization` header;
  sets `admin_id`, `admin_email`, `admin_role` in context. 401 on missing or
  invalid/expired token.
- **`RoleMiddleware(...)`** / **`AdminMiddleware()`** /
  **`SuperAdminMiddleware()`** — per-route role allow-lists; 403 when the
  caller's role is not permitted.
- **`DomainScopeMiddleware(scope)`** — per-domain RBAC for the governance
  domains (`futures`, `options`, `onramp`, `offramp`, `p2p`, `bots`, …):

  | Role | Access |
  |---|---|
  | `super_admin` | Full (unrestricted) access to all domains |
  | `admin` | Read + write |
  | `support`, `analyst`, `moderator` | Read-only (GET only); writes denied (403) |

So the practical hierarchy is **super_admin > admin > support / analyst /
moderator (read-only)**.

## What an Admin CAN Do

Full CRUD across the governance domains, including:

- **Users** — view/manage user accounts, wallets, devices, sessions.
- **Blockchains & tokens** — add/edit/update/remove chains, tokens, trading
  pairs, liquidity sources dynamically.
- **Fees** — configure and adjust fee structures per domain.
- **Withdrawals** — review/approve/reject/process user withdrawal requests
  (the final fund-moving co-sign is not an admin power — see below).
- **KYC / compliance** — review onboarding submissions, risk routing.
- **Trading verticals** — futures, options, margin, copy-trading, convert,
  bots, P2P merchants/clients, on/off-ramps.
- **Operations** — support tickets, knowledge base, notifications,
  marketing, rewards, billing, exports, analytics.
- **Admin org** — manage admin accounts/roles (RBAC), API keys, 2FA,
  auto-approval policies for user transactions.

## What an Admin CANNOT Do (SuperAdmin-Only Powers)

The platform admin app deliberately does **not** contain control-plane
authority. An admin **cannot**:

- **Authorize MasterWallet admins / owners** — that authority lives in
  `super_admin/` (and the MasterWallet's own governance).
- **Set profit-share** (0–50% per WL client) — SuperAdmin only
  (`super_admin/`).
- **Sign or issue white-label licenses** — Ed25519 signing lives in
  `license_service/go/internal/crypto`, driven by SuperAdmin.
- **Operate the kill switch** (global / client / product / fetcher halt &
  resume) — `kill_switch/` on :8469 accepts only `superadmin`-role JWTs.
- **Co-sign fee / revenue / treasury withdrawals** — enforced at the
  MasterWallet broadcast boundary (`master_wallet/backend/license_gate.go`);
  every such withdrawal requires SuperAdmin collaboration and fails closed
  without it. `RevenuePayout`, `TreasuryTransfer`, `TreasurySweep`,
  `FeeWithdrawal` are **never** auto-approved.

Separation note: dead, unreferenced copies of master/admin services that
once existed inside user apps were removed — the canonical implementations
live here and in `super_admin/` / `master_wallet/`; clients call them over
HTTP.

## Environment Variables

| Variable | Purpose |
|---|---|
| `ADMIN_PORT` | Listen port (default `9093`) |
| `JWT_SECRET` | JWT signing secret (required) |
| Database env vars | PostgreSQL (GORM) |
| Redis env vars | Session/cache |
| `TRUSTED_PROXIES` | Optional reverse-proxy CIDRs |

## How to Run

```bash
cd admin/go
go run .                       # :9093 (needs PostgreSQL + Redis)

cd admin/web
npm install
API_URL=http://localhost:9093/api/v1 npm start
```

See `ADMIN_ARCHITECTURE.md` (repo root) for how this app fits the overall
admin/control-plane architecture.
