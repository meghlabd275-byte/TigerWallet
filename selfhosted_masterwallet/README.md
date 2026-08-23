# Self-Hosted MasterWallet (Rust)

**Pillar 3 of the white-label operating model**: a standalone HTTP service
that lets a white-label client self-host the MasterWallet control plane on
their own cloud. It owns its own PostgreSQL database — no dependency on the
TigerWallet cloud — and exposes the **same canonical MasterWallet REST
contract as `master_wallet/backend`**, so the existing client UIs work
unchanged when pointed at this self-hosted endpoint.

Implementation: a single Rust service (`rust/src/main.rs`, ~1500 lines) on
actix-web with modules `chains_data`, `crypto`, `evm_tx`, `multisig`,
`non_evm`, `rlp`.

## Real Only — No Stubs

- JWT (HS256) auth on protected routes.
- PBKDF2-HMAC-SHA256 password hashing (600k iterations) with constant-time
  compare.
- Real `sqlx` queries against PostgreSQL; fail-closed (`401/404/500`) on any
  error — no mocks, no fabricated data.

## API Surface (canonical MasterWallet contract)

Under `/api/v1`:

- **Auth**: `POST /auth/register`, `POST /auth/login`.
- **Master-wallet CRUD**: `POST/GET /master-wallet`, `GET/PUT/DELETE
  /master-wallet/{id}`.
- **Funds view**: `GET /master-wallet/{id}/balance`, `GET/POST
  /master-wallet/{id}/transactions`.
- **Approval workflow**: `POST .../transactions/{tid}/approve`, `POST
  .../transactions/{tid}/reject`.
- **Signing**: `POST .../sign` (sign + broadcast), `POST .../sign-message`.
- **User-wallet derivation**: `POST .../derive-user-address`, `GET
  .../user-wallet-addresses`.
- **Fee management**: `GET/POST .../fees`, `DELETE .../fees/{fid}`.
- **Auto-sign rules**: `GET/POST .../auto-sign`, `DELETE .../auto-sign/{rid}`.
- **Access control**: `GET/POST .../users`, `DELETE .../users/{uid}`;
  `GET/POST .../sub-wallets`.
- **Analytics**: `GET .../analytics/transactions`, `GET
  .../analytics/wallets`.
- **Chain data**: `GET /chains`, `GET /gas`, `GET /price`, `GET /health`.
- **Multisig**: `POST/GET /wallets`, `GET /wallets/{id}/transactions`, `POST
  /transactions` (module `multisig.rs`).

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | (dev fallback) | Own PostgreSQL connection string |
| `JWT_SECRET` | `dev-only-change-me-in-prod` | HS256 secret — **set in production** |
| `BIND_ADDR` | `0.0.0.0:8470` | Listen address |
| `CHAIN_RPC_URL_<chain_id>` | — | Per-chain RPC endpoints for sign/broadcast |

## How to Run

```bash
cd selfhosted_masterwallet/rust
DATABASE_URL=postgres://user:pass@host/db JWT_SECRET=... cargo run --release
# serves on 0.0.0.0:8470
```

Point any canonical MasterWallet client UI at this endpoint instead of the
cloud service; the REST contract is identical.
