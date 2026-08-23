# White-Label Admin

The per-tenant admin console for white-label (WL) clients. A WL client who
licenses a TigerWallet product uses this app to run **their own** back
office: a Go backend (`white_label_admin/go`, Gin + PostgreSQL + Redis,
`SERVER_PORT`, default **8082**) and a React/MUI web UI with **30 pages**
(`white_label_admin/web/src/pages/`).

## Per-Tenant Isolation

Every WL-admin JWT carries a `white_label_id` claim
(`go/internal/middleware/`):

- `JWTAuth` validates the token and stashes `white_label_id` in the Gin
  context.
- `TenantScope` then stashes it as `tenant_id`; **every handler MUST filter
  its queries by this id**. A caller without a `white_label_id` cannot
  access tenant data.
- Even the `wl_client` owner role can only touch rows in their own
  `white_label_id` — tenant isolation is enforced at the middleware layer,
  not by convention.
- An optional `IPWhitelistMiddleware` restricts the console to configured
  IPs.

## License Gate (wlgate) — Fail-Closed

The entire admin route group is wrapped by
`gate.Middleware(cfg.Product, adminFetcher)` from the shared
`wl_shared/wlgate` Go package, before any auth middleware:

- `wlgate.New().WithAutoApprover(wlgate.NewAutoApprover())` builds the gate;
  `HeartbeatLoop` phones home to the SuperAdmin control plane
  (`cfg.TwoPartyGateURL`, the license_service heartbeat) on
  `cfg.HeartbeatInterval` using `cfg.LicenseKey` + `cfg.Product`.
- If the product license is suspended/revoked — or the control plane cannot
  confirm the product is live — the gate **refuses to serve (503)**. There
  is no graceful degradation for admin surfaces.
- **Per-trading-vertical fetcher granularity**: `adminFetcher` maps the
  request path (`/api/v1/admin/<domain>/...`) to the domain fetcher key
  (futures, options, copy-trading, convert, onramp, offramp, p2p-clients,
  partners, rewards, marketing, kyc, tokens, pairs, blockchains, fees,
  withdrawals, admin-roles, ...), so SuperAdmin can disable any single
  vertical (e.g. halt futures while options stays alive).

## 14 Scoped Roles (`go/internal/roles/roles.go`)

One owner plus 13 sub-admin roles, each restricted to its domain via
`middleware.RequireScope(...)` on every route:

| Scope | Purpose |
|---|---|
| `wl_client` | Tenant owner — can do everything in their tenancy EXCEPT withdraw funds/revenue (needs SuperAdmin co-sign); manages sub-admins (`/admin/admins`) |
| `trading_admin` | Product scope — WL futures, margin, options, copy, convert trading |
| `p2p_admin` | Product scope — WL P2P, on-ramp, off-ramp, P2P merchant+client |
| `bot_admin` | Product scope — all WL trading bots |
| `listing_admin` | Product scope — coin/token listing, trading pairs, listing/partner management |
| `liquidity_admin` | Product scope — all WL liquidity sources |
| `wallet_admin` | Product scope — WL MasterWallet + WL-UserWallet management |
| `customer_service_admin` | Customer service / support tickets |
| `marketing_admin` | Marketing & promotion |
| `kyc_admin` | KYC review |
| `card_admin` | WL-branded CryptoCard |
| `reward_admin` | Reward system |
| `security_admin` | Security (WL client only) — user ban/suspend, tx flagging |
| `compliance_admin` | Compliance / audit / reports |

6 product scopes (trading, p2p, bot, listing, liquidity, wallet) plus 7
other-services scopes, plus the owner. `ValidScopes` is a whitelist — a WL
client cannot invent arbitrary scope strings; `AllScopes()` feeds the
frontend role picker; `RequireScope` middleware enforces per-endpoint
authorization.

`GET /api/v1/scopes` enumerates all scopes and their human-readable groups
for the web UI.

## Web UI (30 pages)

`web/src/pages/`: Dashboard, Login, Users, WalletManagement, Transactions,
Trading, Futures, Options, Convert, CopyTrading, P2PFiat, Onramp, Offramp,
CryptoCard, BotsManagement, Liquidity, Listings, Tokens, Fees, Withdrawals,
KYC, Compliance, Security, Marketing, Rewards, CustomerService, Partners,
Admins, Settings (plus `index.tsx` / `_app.tsx` shells).

## What a WL Client CAN Do

- Full governance **of their own tenant**: users, wallets, KYC, compliance,
  security actions, every trading vertical, fees, withdrawals, marketing,
  rewards, support.
- CRUD their own sub-admin accounts and assign scopes (`/admin/admins`,
  `wl_client` scope only).
- 2FA, password changes, token refresh for their admin staff.

## What a WL Client CANNOT Do

- **Reactivate their own product/license** — `license_service` rejects
  self-resume: *"WL client cannot self-resume: only SuperAdmin may
  reactivate a product"* / *"license cannot self-resume: only SuperAdmin may
  reactivate"*.
- **Set their own profit-share** (0–50%) — SuperAdmin only.
- **Sign or mint licenses** — only the SuperAdmin control plane holds the
  Ed25519 private key.
- **See or touch other tenants** — enforced by `TenantScope`.
- **Override the kill switch or the license gate** — a halted product/fetcher
  serves 503 until SuperAdmin resumes it.
- **Co-sign fund withdrawals alone** — every `FeeWithdrawal` /
  `RevenuePayout` / `TreasuryTransfer` / `TreasurySweep` requires the
  SuperAdmin two-party co-sign at the MasterWallet broadcast boundary.

## Environment Variables

| Variable | Purpose |
|---|---|
| `SERVER_PORT` | Listen port (default `8082`) |
| `JWT_SECRET` | JWT signing secret (required at boot) |
| `LICENSE_KEY` | License key issued by SuperAdmin for this product |
| `PRODUCT` | Product identifier used with the gate |
| `TWO_PARTY_GATE_URL` / `TWO_PARTY_GATE_TOKEN` | Control-plane heartbeat endpoint + token |
| `HEARTBEAT_INTERVAL` | How often to phone home |
| Database env vars | PostgreSQL |
| Redis env vars | Shared feature-flag store (non-fatal if down) |

## How to Run

```bash
cd white_label_admin/go
JWT_SECRET=... LICENSE_KEY=... PRODUCT=<product> TWO_PARTY_GATE_URL=... go run .

cd white_label_admin/web
npm install && npm start
```

See `ADMIN_ARCHITECTURE.md` (repo root) for the full white-label control
plane.
