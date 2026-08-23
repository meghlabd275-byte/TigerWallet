# 🐯 WL-UserWallet — Standalone White-Label User Wallet Backend

A white-label clone of the canonical UserWallet backend with **route parity**
to `go/wallet_api` (:8443). It runs in the **WL client's own cloud** with its
**own PostgreSQL** and its **own transaction signing** (BIP-39/32/44 keys
derived and held locally via `wl_shared/go/wlcrypto` — never delegated to
TigerWallet cloud).

All wallet routes run **behind the license gate**: an in-process fail-closed
gate (`internal/middleware`) is fed by a heartbeat loop to the SuperAdmin
control plane; a suspended/revoked license or a stale heartbeat turns every
protected route into 403.

## Port

`8443` (env `WL_USER_WALLET_PORT`).

## Tech stack

- Go, Gin
- PostgreSQL (`pgx/v5` pool) — users, wallets, transactions, settings
- go-ethereum `ethclient` — chain RPC
- `wl_shared/go/wlcrypto` — BIP-39 mnemonics, BIP-32/44 derivation, EIP-1559
  signing, AES-GCM seed encryption at rest
- Internal Go mirror of the C++ `WlGate` (`internal/middleware/middleware.go`)

## Key features (verified in `go/main.go` + `go/internal/handlers`)

- **Route parity with `go/wallet_api:8443`**: `/api/v1/wallet/...` — auth,
  wallet creation/import, balances, send/receive, transaction history,
  settings — so existing UserWallet frontends work unchanged.
- **Own signing**: mnemonics generated locally (24-word BIP-39), keys derived
  at `m/44'/60'/0'/0/i`, EIP-1559 transactions signed and broadcast from the
  WL deployment itself; seeds encrypted at rest with scrypt + AES-GCM.
- **License heartbeat fail-closed**: `middleware.Gate("user_wallet",
  middleware.CategoryFetcher)` derives a per-route fetcher key from the first
  functional path segment, so SuperAdmin can disable individual features
  (e.g. `swap`) without killing the whole product.
- **WebSocket support** (`internal/handlers/ws.go`) for live updates.

## Environment variables

| Variable | Purpose |
|---|---|
| `WL_USER_WALLET_PORT` | Listen port (default `8443`) |
| `DATABASE_URL` | WL client's own PostgreSQL |
| `CONTROL_PLANE_URL` | license_service base URL for heartbeat |
| `WL_CLIENT_ID` / `WL_LICENSE_KEY` | License identity issued by SuperAdmin |
| `JWT_SECRET` | Tenant JWT auth |

## Run

```bash
cd wl_user_wallet/go
go run .
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md` (§3): UserWallets sit at the edge of the platform
(`user_wallet/` for TigerWallet itself, `wl_user_wallet/` for WL clients).
The WL variant is self-hosted and self-custodial but remains under license
control — the SuperAdmin control plane can halt the product or individual
fetchers at any time via the kill switch / feature flags pushed on the
heartbeat.
