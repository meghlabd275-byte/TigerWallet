# 🐯 White Label — White-Label Management System

The white-label management layer of TigerWallet: it provisions and governs
**WL clients** (companies running their own branded wallet stack), their
admins, products, trading pairs, liquidity pools, and tokens. This is the
operator-facing side of the white-label architecture — the WL client
self-hosted backends live in `wl_*` and are license-gated via `wl_shared`.

## Components & ports

| Component | Path | Port | Purpose |
|---|---|---|---|
| White-label management API | `white_label/go/main.go` | `8095` (`PORT`) | CRUD over clients, admins, products, trading pairs, liquidity pools, tokens |
| White-label system | `white_label/system/main.go` | `8090` (`PORT`) | Super Admin authorization, 2FA auth flow, profit sharing, dashboards |
| Marketplace | `white_label/marketplace/go/main.go` | `8085` (`PORT`) | White-label marketplace |
| Structured service | `white_label/go/cmd/white_label_service` | `8090` (`PORT`) | Repository-layer service (config/models/repository) |
| Frontend / portal / sdk / rust / cpp | `white_label/{frontend,portal,sdk,rust,cpp}` | — | Client-facing surfaces |

## Tech stack

- Go, Gin
- PostgreSQL (`pgx`) — all entities persisted via pgx
- Redis — session / admin caching
- bcrypt password hashing, JWT auth, zerolog logging

## Key features (verified in `go/main.go`, `system/main.go`, `cmd/white_label_service`)

- **CreateClient + client lifecycle** (`POST/GET/PUT/DELETE /clients`,
  `/clients/:id/suspend|resume|halt`) — full provisioning and state control
  of WL clients.
- **Admin provisioning** (`/admins` CRUD + `/admins/:id/permissions`):
  - passwords hashed with **bcrypt** (`bcrypt.GenerateFromPassword`);
  - **2FA** — `two_factor_enabled` / `two_factor_secret` columns, plus
    `/auth/2fa/setup` and `/auth/2fa/verify` in `system/main.go`;
  - **lockout** — `login_attempts` / `locked_until` columns; the structured
    service defaults to `MaxLoginAttempts: 3` with a
    `LockoutDuration: 15 minutes`, resetting attempts on successful login.
- **Role model**: `super_admin`, `admin`, `manager`, `support`
  (`system/main.go`); super-admin sessions cached in Redis, admin actions
  written to an audit log (`logTransaction`).
- **Product management**: products CRUD; trading-pairs CRUD + import +
  suspend/resume; liquidity-pool CRUD + import; token CRUD + `createNewToken`.
- **Business rules** (per `system/main.go` header): Super Admin authorization
  over all WL clients, 20% profit sharing with the Super Admin, and a WL
  client may not resell the platform as its own product.

## Environment variables

| Variable | Purpose |
|---|---|
| `PORT` | Listen port (`8095` main API, `8090` system, `8085` marketplace) |
| `DATABASE_URL` / `DB_*` | PostgreSQL |
| `REDIS_URL` | Redis |
| `JWT_SECRET` | Admin auth tokens |

## Run

```bash
cd white_label/go        # management API :8095
go run .

cd white_label/system    # system service :8090
go run .

cd white_label/marketplace/go  # marketplace :8085
go run .
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md`: `white_label/` is where WL clients are
**created and administered**. The pipeline is: intake in
`client_onboarding` (:8101) → `license_service.CreateWLClient` issues an
Ed25519-signed license → the client's self-hosted `wl_*` products are
governed by that license through `wl_shared/go/wlgate`. `multi_tenancy`
supplies plans/quotas and `multi_level_white_label` models reseller
hierarchies on top.
