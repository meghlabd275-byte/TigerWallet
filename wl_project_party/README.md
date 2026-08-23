# 🐯 WL-ProjectParty — Standalone White-Label Token Listing / Launchpad

A white-label clone of the TigerWallet `project_party` token-listing /
launchpad platform. It runs **independently in the WL client's own cloud**
with its own PostgreSQL, and phones home to the license control plane on
heartbeat. Every protected request is gated fail-closed via
`wl_shared/go/wlgate`.

## Port

`8106` (env `PORT`). Product identifier: `project_party` (env `WL_PRODUCT`).

## Tech stack

- Go, Gin (+ gin-contrib/cors)
- PostgreSQL (`pgx/v5`) — tokens, listings, launchpad projects, contributions,
  market-making orders, KYC/audit, fees, favorites
- `wl_shared/go/wlgate` — JWT auth + fail-closed license gate + heartbeat
- bcrypt passwords, JWT auth

## Key features (verified in `go/main.go`)

- **≥50 routes** (80 route registrations) covering the full canonical surface:
- **Token listing pipeline**: token CRUD, `submit` for review, listing-status
  lookup, admin approve/reject/feature — plus public discovery endpoints
  (`/coins`, `/search`, `/featured`, `/trending`, `/market`).
- **Launchpad**: project CRUD, participate/contribute, claim, cancel,
  contribution history, per-project participation lists.
- **Market making**: MM order lifecycle (list/create/update-status), MM
  config CRUD, market-maker status, liquidity add/remove and liquidity state;
  canonical aliases under `/marketmaking/*`.
- **KYC / audit (compliance)**: KYC submit + per-token KYC status, per-token
  audit status (read open), audit-log creation admin-gated — mirrored under
  both `/audit|/kyc` and `/compliance/*` shapes.
- **Fees**: fee listing, calculation, payment, and admin fee-config
  set/update (multiple canonical alias shapes).
- **Analytics**: volume, transaction count, holder count, pricing +
  price history (`/analytics/*` aliases included).
- **Admin governance**: token approve/reject/featured, fee management, audit
  creation, price set/update, admin-scope management — all role-gated and
  behind the license gate.

## Environment variables

| Variable | Purpose |
|---|---|
| `PORT` | Listen port (default `8106`) |
| `DATABASE_URL` | WL client's own PostgreSQL |
| `JWT_SECRET` | Tenant JWT auth |
| `TWO_PARTY_GATE_URL` | License control plane URL |
| `WL_CLIENT_ID` / `WL_LICENSE_KEY` | License identity issued by SuperAdmin |
| `WL_PRODUCT` | Product identifier (default `project_party`) |

## Run

```bash
cd wl_project_party/go
go run .
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md` (§3): the canonical `project_party` service is
TigerWallet's own launchpad; `wl_project_party` is the **self-hosted copy a
WL client runs in its own cloud**, with its own database and token listings,
under the same fail-closed license gate as every other WL product.
