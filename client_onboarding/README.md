# 🐯 Client Onboarding

The intake pipeline for new white-label (WL) customers. A prospective WL
client submits company details and the products it wants; an operator reviews
the application; on approval the service calls
`license_service.CreateWLClient`, which creates the client record and makes
the client eligible for license issuance. Until that approval happens, no WL
product can ever go live for the client.

## Port

`8101` (env `CLIENT_ONBOARDING_PORT`).

## Tech stack

- Go, standard `net/http` (no framework)
- PostgreSQL (`pgx/v5` pool) — applications table, schema auto-created on boot
- `google/uuid` — application IDs

## Flow (verified in `go/cmd/main.go`)

1. **Intake** — `POST /api/v1/onboarding/apply` (public). Stores company name,
   contact email, website, requested products, and notes with status
   `pending`.
2. **Review** — operators list applications
   (`GET /api/v1/onboarding/applications`, optional `?status=` filter) and
   inspect a single one (`GET /api/v1/onboarding/applications/{id}`).
3. **Approve** — `POST /api/v1/onboarding/applications/{id}/approve`. The
   service calls `license_service` (`POST /api/v1/license/clients`) with the
   service token, then marks the application `approved` and stores the
   returned `wl_client_id`. License issuance itself is a separate
   SuperAdmin-signed step in `license_service`.
4. **Reject** — `POST /api/v1/onboarding/applications/{id}/reject` marks the
   application `rejected`.

`GET /health` reports service liveness.

## Environment variables

| Variable | Purpose |
|---|---|
| `CLIENT_ONBOARDING_PORT` | Listen port (default `8101`) |
| `DATABASE_URL` | PostgreSQL connection string |
| `LICENSE_SERVICE_URL` | Base URL of `license_service` (default `http://localhost:9008`) |
| `LICENSE_SERVICE_ADMIN_TOKEN` | Service token used for `CreateWLClient` |

## Run

```bash
cd client_onboarding/go
go run ./cmd
```

## Architecture role

This is the front door of the WL pipeline described in
`ADMIN_ARCHITECTURE.md`: onboarding creates the WL client in the license
control plane; SuperAdmin then signs an Ed25519 license; the client deploys
the standalone WL backends (`wl_master_wallet`, `wl_user_wallet`, `wl_bots`,
`wl_card`, `wl_liquidity`, `wl_project_party`) in its own cloud, and those
phone home to `license_service` via heartbeat.
