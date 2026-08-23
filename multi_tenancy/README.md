# 🐯 Multi-Tenancy — Tenant Management Library

A Go library module (`github.com/tigerwallet/multi-tenancy`) that provides
the core tenant lifecycle for TigerWallet's multi-tenant SaaS layer. It is a
**library, not a standalone service** — it has no `main.go`; embedding
services (admin, billing, white-label control plane) import it and call the
`TenantService` API directly.

## Tech stack

- Go 1.21
- PostgreSQL (`pgx/v5`) — tenants, tenant users, quotas, configs
- Redis (`go-redis/v9`) — available for caching/session use by embedders

## Key features (verified in `go/internal/services/tenant_service.go` + `models/models.go`)

- **CreateTenant flow** (`TenantService.CreateTenant`):
  1. Builds the tenant record with `status=active`, timezone `UTC`,
     language `en`, empty features/metadata.
  2. Sets **`trial_ends_at = now + 14 days`** — every new tenant gets a
     14-day trial window.
  3. Looks up the **`free` plan** (`SELECT id FROM plans WHERE tier='free'`)
     and attaches it — tenant creation fails if no free plan row exists.
  4. Inserts the tenant, then creates **default quotas** and **default
     config**.
- **Default quotas** (per-tenant rows, monthly reset period):
  `api_calls: 1000`, `storage_gb: 1`, `users: 1`, `wallets: 5`, `bots: 1`.
- **Quota enforcement API**: `GetQuota`, `IncrementQuota`,
  `CheckQuota` (returns whether usage is still under the limit).
- **Tenant lifecycle**: get by ID/slug, update fields, update status
  (`active`/`trial`/…), list with status filter + pagination, delete.
- **Tenant users**: `AddUserToTenant` with role, list users, update user
  role.
- **Tenant statuses** include `trial` and `active` (see
  `models.TenantStatus`).

## Data model (verified in `go/internal/models/models.go`)

- **Tenant**: `id`, `name`, `slug`, `email`, `status` (`trial`, `active`,
  ...), `plan_id`, timezone/language, features, metadata, `trial_ends_at`.
- **Quota**: per-tenant `(resource, limit, used)` with `period_start`,
  `period_end`, and `reset_at` — monthly reset window.
- **TenantUser**: tenant ↔ user membership with a role string.
- **Plan**: referenced by tier (`free` is the default attached at creation).

## How to use

```go
import "github.com/tigerwallet/multi-tenancy/internal/services"

svc := services.NewTenantService(...)
tenant, err := svc.CreateTenant(ctx, "Acme Wallet", "ops@acme.example", "acme")
```

There is no binary to run; tests/consumers live in the embedding services.
Build it with `cd multi_tenancy/go && go build ./...`.

## Architecture role

Per `ADMIN_ARCHITECTURE.md`: the tenant layer sits beneath the white-label
system — TigerWallet's SuperAdmin provisions tenants (WL clients), each on a
plan with quotas, and the license service + `wl_shared/go/wlgate` enforce the
resulting limits in the clients' self-hosted products. The 14-day trial +
free-plan default matches the onboarding flow in `client_onboarding`.
