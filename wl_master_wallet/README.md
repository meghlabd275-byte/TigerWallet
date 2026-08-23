# 🐯 WL-MasterWallet — Standalone White-Label Master Wallet

A white-label clone of the canonical `master_wallet/backend` (:8450). It runs
entirely in the **WL client's own cloud** with its **own PostgreSQL** and its
own signing keys — no custody or key material is delegated to TigerWallet
cloud. It exposes the **same REST contract** under `/api/v1/master-wallet/...`
so the existing frontends work unchanged against a WL deployment.

The whole service runs **behind the license gate**: at boot it starts a
heartbeat loop to the TigerWallet SuperAdmin control plane and returns 403 on
every protected route if the license is suspended/revoked/halted or the
heartbeat goes stale (fail-closed).

## Port

`8450` (env `WL_MASTER_WALLET_PORT`).

## Tech stack

- Go, Gin
- PostgreSQL (`pgx/v5` pool) — full master-wallet schema, auto-created on boot
- `wl_shared/go/wlgate` — fail-closed license gate, heartbeat, `TwoPartyGate`
- `wl_shared/go/wlcrypto` — local key custody, EIP-1559 signing
- go-ethereum `ethclient` — real on-chain RPC for balances/broadcasts
- gorilla/websocket — market-data streaming

## Key features (verified in `go/main.go` + `go/internal/handlers`)

- **Same REST contract as canonical master_wallet**: `/api/v1/master-wallet/...`
  including sub-wallet balance/transfer, transaction workflow
  (create/approve/reject/execute/sign), users CRUD, fees/policies/auto-sign,
  analytics (real SQL aggregates), notifications, webhooks, audit, market data
  (chains/gas/price/history), and UserWallet management
  (chains/tokens/addresses/derive) plus feature-flag CRUD
  (`internal/handlers/handlers_routes.go`).
- **License heartbeat fail-closed**: `gate.Middleware("master_wallet",
  wlgate.CategoryFetcher)` guards the API group; dead license ⇒ 403 with the
  gate reason.
- **Two-party withdrawal gate**: withdrawal requests classified as
  `fee`, `revenue`, or `treasury` require a **SuperAdmin co-sign**
  (`wlgate.TwoPartyGate` against the control plane). Requests are created in
  `wl_approved` state and are only executed after SuperAdmin approval;
  `MarkWithdrawalExecuted` records the broadcast tx hash. User withdrawals
  follow the transaction workflow without the co-sign.
- **Auto-approver policy snapshot**: treasury addresses and auto-sign rules
  pushed by the heartbeat feed the `AutoApprover` classifier; anything outside
  the rules falls back to manual (two-party) approval.
- **Real on-chain execution**: EIP-1559 sign + broadcast via `ethclient`;
  non-EVM chains handled in `non_evm_crypto.go`.

## Environment variables

| Variable | Purpose |
|---|---|
| `WL_MASTER_WALLET_PORT` | Listen port (default `8450`) |
| `DATABASE_URL` | WL client's own PostgreSQL |
| `CONTROL_PLANE_URL` | license_service base URL for heartbeat + co-sign |
| `WL_CLIENT_ID` / `WL_LICENSE_KEY` | License identity issued by SuperAdmin |
| `JWT_SECRET` | Tenant JWT auth |

## Run

```bash
cd wl_master_wallet/go
go run .
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md`: the canonical `master_wallet/backend` serves
TigerWallet's own platform; this package is the **same product shipped to a WL
client**, running in the client's infrastructure under license control. Funds
movement at the broadcast boundary always passes the two-party SuperAdmin
gate — the WL operator cannot move fee/revenue/treasury funds unilaterally.
