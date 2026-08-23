# 🐯 WL-Card — Standalone White-Label Crypto Card Backend

A white-label clone of the TigerWallet CryptoCard service. It runs
**independently in the WL client's own cloud** with its own PostgreSQL, and
phones home to the license control plane on heartbeat. Every protected
request is gated fail-closed via `wl_shared/go/wlgate` — the gate starts dead
and only opens after the first successful heartbeat validates the license.

## Port

`8463` (env `CARD_PORT`, fallback `PORT`).

## Tech stack

- Go, Gin (+ gin-contrib/cors)
- PostgreSQL (`pgx/v5`) — users, cards, transactions
- `wl_shared/go/wlgate` — JWT auth + fail-closed license gate + heartbeat
- AES-GCM at-rest encryption for PAN/CVV (key from `CARD_ENC_KEY`)
- bcrypt passwords, JWT auth

## Key features (verified in `go/main.go` + `go/internal/store`)

- **Crypto card product**: card issuance, card status management
  (freeze/unfreeze), balances, and transaction history — real DB rows only,
  no fabricated balances.
- **AES-GCM PAN/CVV at rest**: card numbers and CVVs are stored encrypted;
  `CARD_ENC_KEY` is mandatory at boot.
- **Atomic ledger**: recording a transaction and updating the card balance
  happen in a **single DB transaction** with `SELECT ... FOR UPDATE` on the
  card row — a debit that would overdraw fails atomically, never leaving a
  half-applied ledger entry.
- **Role-gated admin operations**: card issuance and status changes require
  the `admin`/`super_admin` role read from the real `users.role` column;
  cardholders only see their own cards.
- **License heartbeat fail-closed**: `gate.Middleware` guards all
  `/api/v1/cards*` routes; license invalid/suspended or control plane
  unreachable ⇒ 503 on protected routes.

## Environment variables

| Variable | Purpose |
|---|---|
| `CARD_PORT` / `PORT` | Listen port (default `8463`) |
| `DATABASE_URL` | WL client's own PostgreSQL |
| `JWT_SECRET` | Tenant JWT auth |
| `CARD_ENC_KEY` | AES-GCM at-rest key for PAN/CVV (required) |
| `TWO_PARTY_GATE_URL` | License control plane URL (default `http://localhost:8460`) |
| `WL_CLIENT_ID` / `WL_LICENSE_KEY` | License identity issued by SuperAdmin |

## Run

```bash
cd wl_card/go
go run .
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md` (§3): the card product is one of the standalone
WL backends a client deploys in its own cloud after onboarding and license
issuance. Card data and keys never leave the client's infrastructure; the
TigerWallet control plane only sees heartbeats and can halt the product via
the kill switch.
