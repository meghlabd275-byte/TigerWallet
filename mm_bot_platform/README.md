# 🐯 MM Bot Platform — Canonical Bots Product

TigerWallet's **canonical** trading-bot platform: a Go control plane
(`bot_api`, :8471), a Rust execution engine (`bot_core`, :8472), and Solidity
contracts. This is the service the SuperAdmin operates. `wl_bots` is the
white-label clone clients run in their own clouds, and the top-level `bots/`
directory is a **deprecated shim** — real bot functionality lives here.

## Components

| Component | Path | Port | Purpose |
|---|---|---|---|
| Bot API (control plane) | `mm_bot_platform/bot_api` | `8471` (`PORT`) | REST API: auth, bots, tiers, subscriptions, fees, CEX/DEX connectors, keys, admin |
| Bot Core (execution engine) | `mm_bot_platform/bot_core` | `8472` (const) | Rust/Axum engine; bot_api dispatches start/stop/pause to it over HTTP (`BOT_CORE_URL`) |
| Contracts (Foundry) | `mm_bot_platform/bot_admin` | — | `TigerBotPlatform.sol`, `ProjectPartyLaunchpad.sol` (+ Forge tests) |
| Strategies | `mm_bot_platform/strategies` | — | `TigerBotStrategies.sol` |

## Tech stack

- **bot_api**: Go, Gin, PostgreSQL (`pgx/v5` pool), Redis (`go-redis/v9`),
  JWT (HS256), bcrypt
- **bot_core**: Rust, Axum, `sqlx` (PostgreSQL via `BOT_CORE_PG`), tracing
- **contracts**: Solidity, Foundry (`foundry.toml`, `src/`, `test/`)

## Key features (verified in `bot_api/main.go`)

- **18 bot types** (mirrored by the Rust `bot_core` BotType enum):
  `market_maker`, `liquidity_provider`, `sniper`, `front_run`, `mev`,
  `sandwich`, `flash_loan`, `cross_chain`, `perp_hedge`, `grid`, `dca`,
  `momentum`, `mean_reversion`, `scalping`, `ai_trading`, `signal`,
  `arbitrage`, `custom`.
- **4 subscription tiers** with per-tier limits — Free (1 bot), Basic (3),
  Pro (10), Enterprise (50), each with max DEX/CEX counts, latency, and
  monthly fee; public surface `GET /api/v1/public/tiers`.
- **Bot lifecycle**: create/list/get/delete, `start`/`stop`/`pause`/
  direct status set — lifecycle commands are **dispatched over HTTP to
  bot_core** (`/dispatch/*`) rather than executed in-process.
- **CEX/DEX connectors**: `/cex` and `/dex` connector CRUD, plus per-user
  exchange **API keys** (`/keys`) and market-making configs (`/mm-configs`).
- **Subscriptions & fees**: `/subscription`, `/fees`, admin-managed
  `/fee-addresses`.
- **JWT auth + RBAC**: register/login/logout, roles in JWT claims,
  `requireRole(...)` middleware; admin user management (`/users`,
  `/users/:id/status`) and platform `/stats`.
- Frontend-compatibility aliases: `/bots/instances`, `/bots/create`,
  `/bots/me`.

## Environment variables

| Variable | Purpose |
|---|---|
| `PORT` | bot_api listen port (default `8471`) |
| `DATABASE_URL` | PostgreSQL for bot_api |
| `REDIS_ADDR` | Redis for bot_api |
| `JWT_SECRET` | HS256 signing secret |
| `BOT_CORE_URL` | Where bot_api reaches bot_core (default `http://localhost:8472`) |
| `BOT_CORE_PG` | PostgreSQL DSN for bot_core |

## Run

```bash
cd mm_bot_platform/bot_api   # control plane :8471
go run .

cd mm_bot_platform/bot_core  # execution engine :8472
cargo run

cd mm_bot_platform/bot_admin # contracts
forge test
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md`: this is the canonical Bots product owned by the
SuperAdmin. `wl_bots` is the self-hosted clone shipped to WL clients (same
API surface, own DB, AES-GCM-encrypted keys, license-gated via
`wl_shared/go/wlgate`). The top-level `bots/` directory is a **deprecated
shim** — do not extend it; all real bot work belongs in `mm_bot_platform`.
