# 🐯 License Service — TigerWallet Control Plane

The central licensing authority for the TigerWallet white-label (WL) platform.
It is the **only** service that can authorize a WL product to run: it issues
Ed25519-signed license tokens, receives heartbeats from every WL deployment,
and answers the fail-closed `IsProductAlive` check that all WL backends
consult before serving a single request.

See `ADMIN_ARCHITECTURE.md` (§1.2, §3) for how this service fits the admin
plane: SuperAdmin signs/renews licenses, operates the kill switch, sets
profit-share, and co-signs withdrawals — all of which route through here.

## Port

`9008` (env `PORT`).

## Tech stack

- Go, Gin HTTP framework
- PostgreSQL (`pgx/v5` pool) — clients, licenses, heartbeats, withdrawals,
  feature flags, profit-share, audit log
- Redis (`go-redis/v9`) — published feature-flag / kill-switch snapshots
- Ed25519 (`crypto/ed25519`) — license token signing and verification
- JWT (`golang-jwt/v5`) — SuperAdmin/operator service auth

## Key features (verified in `go/internal/handlers` + `go/internal/crypto`)

- **Ed25519 `LicenseToken`** (`internal/crypto/license.go`): SuperAdmin signs
  a token with a **5-minute TTL**. WL products renew it continuously via the
  heartbeat loop; an expired token fails verification, so a disconnected WL
  product goes dead within minutes.
- **Client/license statuses**: `active`, `suspended`, `revoked`, `expired`,
  `halted`. A license can only transition back to `active` from `suspended`
  via resume.
- **Self-resume is SuperAdmin-only**: a WL client cannot lift its own
  suspension — only a SuperAdmin-signed request may resume a client/license.
- **`IsProductAlive`** = client approved **and** license active **and**
  heartbeat fresh. All three must hold; any failure returns dead (fail-closed).
- **Kill switch**: global / per-client / per-product / per-fetcher halt,
  consulted on every heartbeat validation (`Hub.Killed()`).
- **Two-party withdrawal co-sign**: WL clients create withdrawal requests in
  `wl_approved` state; funds move only after
  `SuperAdminApproveWithdrawal` — enforced at the master-wallet broadcast
  boundary as well.
- **Feature-flag policy snapshot**: the heartbeat response pushes the current
  flag snapshot (treasury addresses, auto-sign rules, per-fetcher enables)
  into each WL product, so policy changes propagate live.
- **Branding updates** with audit logging; short-TTL (5 min) auth tokens for
  service-to-service calls.

## Environment variables

| Variable | Purpose |
|---|---|
| `PORT` | Listen port (default `9008`) |
| `DATABASE_URL` / `DB_*` | PostgreSQL connection |
| `REDIS_ADDR` | Redis for flag/kill-switch snapshots |
| `JWT_SECRET` | Shared with `super_admin` for service auth |
| `LICENSE_SIGNING_KEY` / Ed25519 keypair | Token signing |

## Run

```bash
cd license_service/go
go run .
```

## Architecture role

WL products (`wl_master_wallet`, `wl_user_wallet`, `wl_bots`, `wl_card`,
`wl_liquidity`, `wl_project_party`, `white_label_admin`) run standalone in
each WL client's own cloud but phone home here via heartbeat. Without a valid
license + live heartbeat they return 403/503 fail-closed. New WL clients are
created by `client_onboarding` (intake → review → approve →
`CreateWLClient` here).
