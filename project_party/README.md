# 🐯 ProjectParty — Canonical Token Listing & Launchpad Platform

TigerWallet's **canonical** token-listing / launchpad / market-making
service. Unlike the white-label clone (`wl_project_party`), this service
runs in TigerWallet's own cloud and executes **real on-chain launchpad
operations** against the `ProjectPartyLaunchpad` Solidity contract via
go-ethereum's `ethclient` — contributions, claims, and cancellations are
actual EVM transactions, never mocked or fabricated.

## Port

`8106` (env `PROJECT_PARTY_PORT`).

## Tech stack

- Go, Gin
- PostgreSQL (`pgx/v5` pool) — tokens, listings, launchpad projects,
  contributions, market-making orders/configs, KYC/audit, fees, pricing,
  analytics, favorites
- Redis (`go-redis/v9`) — caching for public coin/search/market endpoints
- **go-ethereum `ethclient` + `abi/bind`** — real on-chain launchpad
  transactions (`launchpad_onchain.go`)
- bcrypt passwords, JWT auth

## Key features (verified in `go/cmd/main.go` + `go/cmd/launchpad_onchain.go`)

- **Real on-chain launchpad** (`LaunchpadOnChain`): dials `PP_RPC_URL`,
  loads the operator key from `PP_LAUNCHPAD_PRIVATE_KEY` and the contract
  from `PP_LAUNCHPAD_CONTRACT_ADDRESS`, parses the launchpad ABI inline, and
  sends real transactions. **Fail-closed**: if the on-chain path is
  unconfigured or the RPC errors, contribute/claim return
  `503 on-chain not configured` — the service never fabricates a tx hash.
- **Token pipeline**: token CRUD + `submit` for review, admin
  approve/reject, contract verification; listings CRUD, status updates,
  admin feature flags; public browse: `/coins`, `/search`, `/featured`,
  `/trending`, `/market`.
- **Launchpad lifecycle**: create project, contribute, claim tokens, cancel.
- **Market making**: maker orders (create/list/update-status), MM configs
  CRUD, liquidity add/remove, market-maker status per token.
- **Compliance**: KYC submit/status, audit request/status per token.
- **Fees**: listing fee list, calculate, pay, admin verify.
- **Pricing & analytics**: price set/update + history; trading volume,
  liquidity, holder count, transaction count.
- **Favorites** per authenticated user.

## Environment variables

| Variable | Purpose |
|---|---|
| `PROJECT_PARTY_PORT` | Listen port (default `8106`) |
| `DATABASE_URL` | PostgreSQL |
| `REDIS_ADDR` / `REDIS_PASSWORD` | Redis cache |
| `JWT_SECRET` | Auth tokens |
| `PP_RPC_URL` | EVM RPC endpoint for the on-chain launchpad |
| `PP_LAUNCHPAD_PRIVATE_KEY` | Operator wallet key (on-chain signer) |
| `PP_LAUNCHPAD_CONTRACT_ADDRESS` | Deployed launchpad contract |

## Run

```bash
cd project_party/go/cmd
go run .
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md`: `project_party` is the canonical launchpad
product owned and operated by TigerWallet (SuperAdmin). `wl_project_party`
is its white-label clone that WL clients run in their own clouds under the
fail-closed license gate. If a WL client's license is revoked only their
clone halts — this canonical service is unaffected.
