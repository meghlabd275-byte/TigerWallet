# 🐯 WL-Bots — Standalone White-Label Bot Management Backend

A white-label clone of the canonical TigerWallet bot platform
(`mm_bot_platform/bot_api`). It runs **independently in the WL client's own
cloud** with its own PostgreSQL and phones home to the TigerWallet license
control plane on heartbeat. Every protected request is gated fail-closed via
`wl_shared/go/wlgate` — a suspended/revoked license or stale heartbeat 403s
the API.

## Port

`8471` (env `PORT`).

## Tech stack

- Go, Gin (+ gin-contrib/cors)
- PostgreSQL (`pgx/v5`) — users, bots, subscriptions, fees, API keys,
  CEX/DEX connector configs, executions, logs
- `wl_shared/go/wlgate` — JWT auth + fail-closed license gate + heartbeat
- `wl_shared/go/wlcrypto` — AES-GCM at-rest encryption
- bcrypt passwords, JWT auth

## Key features (verified in `go/main.go` + `go/internal/handlers`)

- **18 bot types** (domain constants mirrored from the canonical platform):
  `market_maker`, `liquidity_provider`, `sniper`, `front_run`, `mev`,
  `sandwich`, `flash_loan`, `cross_chain`, `perp_hedge`, `grid`, `dca`,
  `momentum`, `mean_reversion`, `scalping`, `ai_trading`, `signal`,
  `arbitrage`, `custom`.
- **4 subscription tiers** — Free / Basic / Pro / Enterprise — with per-tier
  bot/DEX/CEX limits; public surface `GET /api/v1/public/tiers`.
- **Full bot lifecycle**: create/list/get/delete, start/stop/pause, direct
  status set, executions and logs per bot; canonical aliases
  (`/bots/create`, `/bots/instances`) for frontend compatibility.
- **CEX/DEX connectors**: admin CRUD for CEX configs and DEX connection
  management, with connector secrets **AES-GCM encrypted at rest** via
  `wlcrypto` (scrypt(passphrase) → AES-256-GCM; the JWT secret is the
  passphrase) — same for per-user exchange API keys.
- **Subscriptions + fee configs** (create/list/update), platform stats and
  user management for `super_admin` / `finance_admin` / `bot_operator` roles.
- **License heartbeat fail-closed**: `gate.Middleware("bots",
  wlgate.CategoryFetcher)` on the protected group; the heartbeat loop to the
  control plane is required at boot (`TWO_PARTY_GATE_URL`, `WL_CLIENT_ID`,
  `WL_LICENSE_KEY` are mandatory).

## Environment variables

| Variable | Purpose |
|---|---|
| `PORT` | Listen port (default `8471`) |
| `DATABASE_URL` | WL client's own PostgreSQL |
| `JWT_SECRET` | Auth tokens **and** at-rest key-encryption passphrase |
| `TWO_PARTY_GATE_URL` | License control plane URL |
| `WL_CLIENT_ID` / `WL_LICENSE_KEY` | License identity issued by SuperAdmin |

## Run

```bash
cd wl_bots/go
go run .
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md`: the canonical Bots product is
`mm_bot_platform` (bot_api control plane + Rust bot_core execution engine).
`wl_bots` is the **self-hosted copy shipped to a WL client** — same API
surface, own database, own encrypted keys, but under license control: the
SuperAdmin control plane can halt the product or individual fetcher
categories at any time.
