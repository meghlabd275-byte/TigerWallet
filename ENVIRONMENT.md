# TigerWallet — Environment Variables Reference

> Canonical environment-variable reference for the platform backends and the
> white-label products. **No credentials are hardcoded** — every secret is
> injected via the deployment's secrets manager / `.env` (both are git-ignored).
> No `.env.example` is committed; copy the service-specific variable group
> below and set real values.

Provider credentials (RPC URLs, explorer keys, price-feed keys, payment/KYC/AML
keys) are set in the Admin Panel / SuperAdmin dashboard or via env and are read
at runtime — they are never embedded in source.

---

## Canonical backends

### MasterWallet backend (`master_wallet/backend`, :8450)

| Variable | Purpose |
|---|---|
| `MASTER_WALLET_PORT` | Listen port (default `8450`) |
| `MASTER_WALLET_JWT_SECRET` | JWT signing secret |
| `MASTER_WALLET_DATABASE_URL` / `DB_*` | PostgreSQL |
| `MASTER_WALLET_REDIS_PASSWORD` / `REDIS_*` | Redis |
| `MASTER_WALLET_TREASURY_KEY_HEX` | Treasury hot-wallet key (unset → treasury writes 503) |
| `MASTER_AUTO_SIGN_POLL_MS` | Auto-sign poll interval (default 100 ms) |
| `MASTER_AUTO_SIGN_PASSWORD` | Auto-sign unlock password (unset → broadcast disabled, fail-closed) |
| `MASTER_WALLET_BUNDLER_URL` / `MASTER_WALLET_PAYMASTER_URL` | Account-abstraction bundler/paymaster |
| `COINGECKO_API_KEY`, `ETHERSCAN_API_KEY` | Market/explorer data |
| `ETH_RPC_URL`, `BSC_RPC_URL`, `POLYGON_RPC_URL`, … | Per-chain RPC (fail-closed when unset) |

### UserWallet API (`go/wallet_api`, :8443)

| Variable | Purpose |
|---|---|
| `PORT` | Listen port (default `8443`) |
| `JWT_SECRET` | JWT signing secret |
| Database env vars | PostgreSQL (encrypted seeds, users, tx log) |
| Redis env vars | Session/cache |
| `COINGECKO_API_KEY`, `ETHERSCAN_API_KEY` | Market/explorer data |
| `REACT_APP_API_URL` (frontend) | Canonical API base (default `http://localhost:8443/api/v1`) |
| `REACT_APP_GOOGLE_CLIENT_ID` (frontend) | Google OAuth client ID for Drive backup (unset → Drive button disabled) |

### Admin API (`admin/go`, :9093)

| Variable | Purpose |
|---|---|
| `ADMIN_PORT` | Listen port (default `9093`) |
| `JWT_SECRET` | JWT signing secret (required) |
| Database env vars | PostgreSQL (GORM) |
| Redis env vars | Session/cache |
| `TRUSTED_PROXIES` | Optional reverse-proxy CIDRs |

### SuperAdmin API (`super_admin/go`, :8082)

| Variable | Purpose |
|---|---|
| `SERVER_PORT` | Listen port (default `8082`) |
| `JWT_SECRET` | HS256 JWT secret (shared with `license_service` + `kill_switch`) |
| Database env vars | PostgreSQL (pgx) |
| Redis env vars | Feature-flag publication, kill-switch state |
| `TRUSTED_PROXIES` | Optional reverse-proxy CIDRs |

### ProjectParty API (`project_party/go`, :8106)

| Variable | Purpose |
|---|---|
| `PROJECT_PARTY_PORT` | Listen port (default `8106`) |
| `DATABASE_URL` | PostgreSQL |
| `REDIS_ADDR` / `REDIS_PASSWORD` | Redis cache |
| `JWT_SECRET` | Auth tokens |
| `PP_RPC_URL` | EVM RPC endpoint for the on-chain launchpad |
| `PP_LAUNCHPAD_PRIVATE_KEY` | Operator wallet key (on-chain signer) |
| `PP_LAUNCHPAD_CONTRACT_ADDRESS` | Deployed launchpad contract |

### Control-plane services

| Service | Variable | Default |
|---|---|---|
| License service (`license_service/go`) | `LICENSE_PORT` | 9008 (compose overrides to 8460) |
| | `GATE_ED25519_PUBLIC_KEY` | — |
| Kill switch (`kill_switch`) | `PORT` | 8469 |
| Permission service (`permission_service`) | `PERMISSION_PORT` | 8091 (compose maps 8085) |
| Permission bridge (`permission_bridge`) | `PERMISSION_BRIDGE_PORT` | 9007 |
| Connection API (`connection_api`) | `CONNECTION_PORT` | 8092 |
| Fetcher gateway (`fetcher_gateway`, Rust) | `FETCHER_PORT` | 8093 |

---

## White-label products (self-hosted)

All WL products share the license-gate / heartbeat variables (via
`wl_shared/go/wlgate`):

| Variable | Purpose |
|---|---|
| `LICENSE_KEY` | License key issued by SuperAdmin for this product |
| `PRODUCT` | Product identifier used with the gate |
| `TWO_PARTY_GATE_URL` / `TWO_PARTY_GATE_TOKEN` | Control-plane heartbeat endpoint + token |
| `HEARTBEAT_INTERVAL` | How often to phone home |
| `JWT_SECRET` | JWT signing secret (required at boot) |
| Database env vars | PostgreSQL (own tenant DB) |
| Redis env vars | Cache / feature-flag store (non-fatal if down) |

Per-product listen ports (container-internal defaults):

| Product | Variable | Default |
|---|---|---|
| WL MasterWallet | `PORT` | 8450 |
| WL UserWallet | `PORT` | 8443 |
| WL Bots | `PORT` | 8471 |
| WL ProjectParty | `PORT` | 8106 |
| WL Card | `CARD_PORT` / `PORT` | 8463 |
| WL Liquidity | `LIQ_PORT` / `PORT` | 8462 |
| WL Admin | `SERVER_PORT` | 8082 |

> In the platform `docker-compose.yml`, the WL host-side mappings are 8461–8464
> plus 8456 (wl-admin), 8458 (wl-liquidity), 8459 (wl-card) — see `ARCHITECTURE.md`.

---

## Provider credential provisioning (no hardcoded keys)

When a feature needs an external provider (RPC, explorer, price feed, CEX/DEX
API, fiat/payment/KYC/AML, notifications, cloud storage, email/SMS/push, AI),
configure the credential through the **Admin Panel / SuperAdmin dashboard** (or
env) — never in source. The implementation already supports real integration
for:

- `go/fiat_ramp` — Stripe/MoonPay/Transak (HMAC-verified webhooks).
- `ai_agent` — real `eth_gasPrice` via `EVM_RPC_URL`.
- `go/wallet_api` fetchers — real `eth_getBalance` / `eth_call` /
  `eth_sendRawTransaction` via go-ethereum RPC.
- `project_party` — real on-chain launchpad via go-ethereum `ethclient`.

See `GAPS.md` for the fetchers that remain scaffold-only (`go/full_fetchers`).