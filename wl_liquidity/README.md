# 🐯 WL-Liquidity — Standalone White-Label Liquidity Aggregator

A white-label clone of the TigerWallet liquidity aggregator. It runs
**independently in the WL client's own cloud** with its own PostgreSQL, and
phones home to the license control plane on heartbeat. Every protected
request is gated fail-closed via `wl_shared/go/wlgate`.

## Port

`8462` (env `LIQ_PORT`, fallback `PORT`).

## Tech stack

- Go, Gin (+ gin-contrib/cors)
- PostgreSQL (`pgx/v5`) — liquidity sources, routes, pools
- `wl_shared/go/wlgate` — JWT auth + fail-closed license gate + heartbeat
- bcrypt passwords, JWT auth

## Key features (verified in `go/main.go` + `go/internal/handlers`)

- **Liquidity aggregator**: admin-managed liquidity sources and routes; the
  service starts empty (no fabricated pool data) and is populated by admin
  CRUD — `POST/PUT/DELETE /sources`, `POST/DELETE /routes`
  (role-gated to admin roles).
- **Real constant-product (x·y=k) quote math**: `GET /quote` computes output
  amounts from actual pool reserves using the x·y=k formula; `GET /best_dex`
  compares sources and returns the best venue. When no pool covers a pair the
  API returns an honest 404/empty result instead of a fabricated quote.
- **Depth / pools endpoints**: `GET /depth`, `GET /pools` expose real stored
  pool state.
- **Read endpoints** (`/sources`, `/routes`, `/quote`, `/depth`, `/pools`,
  `/best_dex`) require JWT + live license; **mutations** additionally require
  an admin role.
- **License heartbeat fail-closed**: the gate is dead until the first
  successful heartbeat validates the license against the SuperAdmin control
  plane.

## Environment variables

| Variable | Purpose |
|---|---|
| `LIQ_PORT` / `PORT` | Listen port (default `8462`) |
| `DATABASE_URL` | WL client's own PostgreSQL |
| `JWT_SECRET` | Tenant JWT auth |
| `TWO_PARTY_GATE_URL` | License control plane URL |
| `WL_CLIENT_ID` / `WL_LICENSE_KEY` | License identity issued by SuperAdmin |

## Run

```bash
cd wl_liquidity/go
go run .
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md` (§3): `wl_liquidity` is the self-hosted copy of
the liquidity aggregator shipped to a WL client. It serves quotes for the
client's own swap/DEX features while remaining under license control — the
control plane can disable the product or individual fetcher categories via
the kill switch / feature flags pushed on heartbeat.
