# Super Admin

The **Super Admin** app is the **top authority** of TigerWallet: the only
place that can sign white-label licenses, set profit-share, operate the kill
switch, publish feature flags, and co-sign fund/revenue withdrawals. Go
backend (`super_admin/go`, Gin + PostgreSQL + Redis, `SERVER_PORT`, default
**8082**) plus a React/MUI web console with **41 pages**
(`super_admin/web/src/pages/`).

## Powers Exclusive to SuperAdmin

### License signing (Ed25519)

SuperAdmin issues and manages white-label product licenses through
`license_service/go` — the sole authority that can authorize an
externally-hosted WL product to run. Licenses are Ed25519-signed tokens
(`license_service/go/internal/crypto`); the public key is baked into every WL
product, the private key never leaves this control plane. WL products
heartbeat to `POST /api/v1/license/heartbeat`; a suspended/revoked license
halts the product **fail-closed** (every route refuses to serve).

Self-resume restriction: a WL client **cannot** reactivate itself.
`license_service/go/internal/store/store.go` rejects self-resume in both
places:

- `"WL client cannot self-resume: only SuperAdmin may reactivate a product"`
- `"license cannot self-resume: only SuperAdmin may reactivate"`

Only a SuperAdmin can move a suspended/revoked product or license back to
active.

### Profit-share (0–50%)

SuperAdmin sets each WL client's revenue share via
`admin/go/internal/services/super_admin_service.go`
(`ProfitShareConfig`: `WhiteLabelID`, `SuperAdminWallet`,
`ProfitPercentage` default 20%, `AutoTransfer`):

- Enforced bounds: `percentage < 0 || percentage > 50` is rejected with
  `"percentage must be between 0 and 50"`; `MaxPercentage` is 50%.
- Authorization: `"unauthorized: only super admin can set profit share"`.
- Payout transactions are classified `RevenuePayout` and therefore require
  the two-party co-sign (below) — they can never ride the auto-sign path.

### Kill switch

`kill_switch/` (Go service, port **8469**). SuperAdmin-only auth (HS256 JWT,
role `superadmin`, shared `JWT_SECRET` with `license_service`; 401/403
fail-closed). Four scopes — `global`, `client`, `product`, `fetcher` — with
durable state + audit trail in PostgreSQL (`kill_state`, `kill_events`) and
sub-second propagation through Redis keys (`kill:global`, `kill:client:<id>`,
`kill:product:<id>:<product>`, `kill:fetcher:<id>:<product>:<fetcher>`) and
the `kill:events` pub/sub channel. A self-healing loop republishes active
halts from PG into Redis every 10s. The `license_service` heartbeat consults
the kill switch and fails closed with `{"alive": false, "command": "halt"}`,
so a halt reaches every WL product within one heartbeat interval.

UI: the Governance page (`super_admin/web/src/pages/Governance.tsx`) has a
GlobalKillBar (platform-wide HALT/RESUME, live state, 15s refresh) and
per-client Halt All / Resume buttons wired through `killApi.ts` → Vite
`/kill-api` proxy → :8469.

### Feature flags

SuperAdmin CRUDs feature flags (`feature_flags` in PostgreSQL) and publishes
them live to shared Redis keys that downstream services read — enabling
instant, per-fetcher enable/disable across the fleet (futures, options,
copy-trading, convert, on-ramp, off-ramp, p2p, partners, rewards, marketing,
kyc, tokens, pairs, blockchains, fees, withdrawals, admin-roles, ...).

### Two-party co-sign for withdrawals

*"No one can withdraw any fund or revenue without TigerWallet SuperAdmin
collaboration."* Enforced at the broadcast boundary in
`master_wallet/backend/license_gate.go`: any `FeeWithdrawal`,
`RevenuePayout`, `TreasuryTransfer`, or `TreasurySweep` requires a valid
SuperAdmin co-sign before the MasterWallet will broadcast, and fails closed
when the control plane is unset/unreachable. The SuperAdmin web console's
Withdrawals page surfaces the approve/reject/process review flow
(`super_admin/go` withdrawal routes).

### WL client & product lifecycle

Create/suspend/revoke licenses, and start/stop/pause/resume WL products via
the product status field (`super_admin/go` + `license_service` store).

## Web Pages (41)

`super_admin/web/src/pages/`: Dashboard, Governance, WhiteLabels, Users,
Admins, AdminRoles, MasterWallets, UserWallets, Transactions, Withdrawals,
Fees, Tokens, Blockchains, TradingPairs, Futures, Options, CopyTrading,
Convert, P2PClients, P2PMerchants, OnRamp, OffRamp, Bots, BotsClients,
Partners, ProjectTeams, Rewards, Marketing, KYC, CryptoCards, KnowledgeBase,
Tickets, Reports, AuditLogs, APIKeys, Webhooks, Security, System, Workflows,
Settings, Login.

## Environment Variables

| Variable | Purpose |
|---|---|
| `SERVER_PORT` | Listen port (default `8082`) |
| `JWT_SECRET` | HS256 JWT secret (required; shared with `license_service` and `kill_switch`) |
| Database env vars | PostgreSQL (pgx) |
| Redis env vars | Feature-flag publication, kill-switch state |
| `TRUSTED_PROXIES` | Optional reverse-proxy CIDRs |

## How to Run

```bash
cd super_admin/go
JWT_SECRET=... go run .        # :8082 (needs PostgreSQL + Redis)

cd super_admin/web
npm install
npm start                      # Vite dev server, /kill-api proxy -> :8469
```

Or via the platform compose (`docker compose up super-admin`).

See `ADMIN_ARCHITECTURE.md` (repo root) for the full control-plane diagram
and how SuperAdmin interacts with `license_service`, `kill_switch`, and the
MasterWallet co-sign gate.
