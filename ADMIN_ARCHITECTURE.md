# TigerWallet Admin Architecture

This document describes the administrative plane of TigerWallet: the three admin
applications, the role hierarchy, and the control-plane security mechanisms
(license gate, kill switch, two-party withdrawal co-sign, feature flags,
profit-share). It is the canonical reference for "who can do what, and how the
platform enforces it."

All claims below are backed by the implementation in this repository:

| Component | Canonical location |
|---|---|
| Platform Admin app | `admin/` (backend `admin/go`, frontend `admin/web`) |
| Super Admin app | `super_admin/` (backend `super_admin/go`, frontend `super_admin/web`) |
| White-Label Admin app | `white_label_admin/` (backend `white_label_admin/go`, frontend `white_label_admin/web`) |
| License service (control plane) | `license_service/go` |
| Kill switch | `kill_switch/` |
| WL control plane (C++/Rust) | `wl_control_plane/cpp`, `wl_control_plane/rust` |
| WL gate shared library (Go) | `wl_shared/go/wlgate` |
| MasterWallet (execution, funds) | `master_wallet/backend` |

---

## 1. The Three Admin Apps

### 1.1 Platform Admin — `admin/`

The day-to-day governance console for the whole platform. Go backend
(`admin/go`, Gin + PostgreSQL/GORM + Redis, port `9093` via `ADMIN_PORT`) with
**46 domain handler files** in `admin/go/internal/handlers/` registering
~340 routes in `main.go`, and a React web UI with 36 pages
(`admin/web/src/pages/`). Platforms: web, android, ios, desktop, extensions
(`admin/{web,android,ios,desktop,extensions}`).

Governance domains include: users, tokens, pairs, liquidity, blockchains, fees,
withdrawals, KYC/compliance, support tickets, knowledge base, notifications,
marketing, rewards, billing, exports, RBAC, API keys, 2FA, auto-approvals,
white labels, partners, brokers, bots, P2P, futures/options/margin trading,
institutional, crypto cards, convert, on/off-ramps, NFTs, multisig, and
analytics.

### 1.2 Super Admin — `super_admin/`

The top authority. Go backend (`super_admin/go`, Gin + PostgreSQL + Redis,
port `SERVER_PORT`, default `8082`) and a React/MUI web UI with 39 pages
(`super_admin/web/src/pages/`). SuperAdmin is the **only** role that can:

- Sign and issue white-label **licenses** (Ed25519 — see `license_service/go/internal/crypto`).
- Set **profit-share** percentage per WL client (0–50%).
- Operate the **kill switch** (`kill_switch/` on :8469) — halt/resume at
  global / client / product / fetcher scope.
- Manage **feature flags** (CRUD + live publish to shared Redis keys).
- **Co-sign withdrawals** — every fee/revenue/treasury withdrawal requires
  SuperAdmin collaboration at the broadcast boundary
  (`master_wallet/backend/license_gate.go`).
- Manage WL client lifecycle (create/suspend/revoke licenses, start/stop/
  pause/resume products).

### 1.3 White-Label Admin — `white_label_admin/`

The per-tenant admin console for a white-label (WL) client. Go backend
(`white_label_admin/go`, Gin + PostgreSQL + Redis, `SERVER_PORT`) with a
React/MUI web UI of 28 pages (`white_label_admin/web/src/pages/`). Every
request is scoped to a `white_label_id` claim in the JWT — a WL admin can
never see or touch another tenant's data. The WL client (`wl_client` role)
can create **14 scoped sub-admin roles** (see §2.2) whose permissions are
enforced per endpoint by the `RequireScope` middleware.

The WL admin backend itself runs **behind the license gate** (fail-closed):
at boot it starts a heartbeat loop to the SuperAdmin control plane and returns
503 on all routes if the product license is suspended/revoked or the matching
fetcher has been disabled
(`white_label_admin/go/main.go`, using `wl_shared/go/wlgate`).

---

## 2. Role Hierarchy

```
super_admin            Top authority. Licenses, kill switch, profit-share,
   │                   feature flags, withdrawal co-sign, WL lifecycle.
   ▼
admin                  Full read+write on the 11+ governance domains in the
   │                   platform admin panel. Cannot sign licenses, set
   │                   profit-share, co-sign withdrawals, or use kill switch.
   ▼
support / analyst / moderator
                       Read-only (GET) on governance domains; writes denied
                       (403) via DomainScopeMiddleware
                       (admin/go/internal/middleware/auth.go).
```

### 2.1 Platform-side RBAC

Enforced in `admin/go/internal/middleware/auth.go`:

- `AuthMiddleware` — validates the JWT (`Authorization` header), sets
  `admin_id` / `admin_email` / `admin_role` in context. 401 on missing or
  invalid token.
- `RoleMiddleware(...)` / `AdminMiddleware()` / `SuperAdminMiddleware()` —
  per-route role allow-lists; 403 on insufficient role.
- `DomainScopeMiddleware(scope)` — per-domain RBAC for the governance
  domains: `super_admin` gets full access, `admin` gets read+write,
  `support`/`analyst`/`moderator` get read-only (GET); all other writes are
  denied.

### 2.2 White-label scoped roles (14)

Defined in `white_label_admin/go/internal/roles/roles.go` (authoritative
list) — the WL client owner role plus 13 scoped sub-admin roles a WL client
can assign:

- `wl_client` — the WL client owner; can do everything in their tenancy
  **except** withdraw funds/revenue (that needs SuperAdmin co-sign).
- 6 product scopes: `trading_admin` (futures/margin/options/copy/convert),
  `p2p_admin` (p2p, on-ramp, off-ramp, merchant), `bot_admin`,
  `listing_admin` (coin/token listing, trading pairs, partners),
  `liquidity_admin`, `wallet_admin` (WL MasterWallet + WL UserWallet).
- 7 other-services scopes: `customer_service_admin`, `marketing_admin`,
  `kyc_admin`, `card_admin`, `reward_admin`, `security_admin`,
  `compliance_admin`.

Every WL admin route is protected by `middleware.RequireScope(...)` in
`white_label_admin/go/main.go`, mapping each endpoint to exactly the role
allowed to call it. Sub-admin management itself (create/update/suspend/delete
admins) requires the `WLClient` scope — only the WL client owner can manage
its sub-admins.

---

## 3. License Gate (Ed25519, fail-closed, heartbeat)

Authority: `license_service/go` — "the ONLY authority that can authorize an
externally-hosted white-label product to run" (see its `main.go` header).

- **Signing:** licenses are Ed25519-signed tokens
  (`license_service/go/internal/crypto`). The public key is baked into every
  WL product; the private key never leaves the SuperAdmin control plane.
- **Heartbeat:** WL products phone home to
  `POST /api/v1/license/heartbeat` on a configurable interval. The heartbeat
  answer is the lifecycle command (`alive` / `halt`).
- **Fail-closed:** if the license is suspended/revoked, expired, or the
  heartbeat fails, the embedded gate refuses to serve (HTTP 403/503
  depending on the layer). Missing config = refuse, never permit.
- **Layers:** the same invariant is enforced in three languages, kept in
  lockstep:
  - C++ shared library for ultra-low-latency services (`wl_control_plane/cpp`)
  - Rust crate (`wl_control_plane/rust`, `license.rs` — Ed25519 verification,
    fail-closed)
  - Go shared library (`wl_shared/go/wlgate`) used by Go WL services such as
    `white_label_admin`, `wl_user_wallet`, `wl_master_wallet`.

## 4. Kill Switch

Service: `kill_switch/` (Go, port `8469`). SuperAdmin-only auth (HS256 JWT,
role `superadmin`, shared `JWT_SECRET` with `license_service`).

- Four scopes: `global` (whole platform), `client`, `product`, `fetcher`.
- Durable state + full audit trail in PostgreSQL (`kill_state`, `kill_events`).
- Sub-second propagation via Redis keys
  (`kill:global`, `kill:client:<id>`, `kill:product:<id>:<product>`,
  `kill:fetcher:<id>:<product>:<fetcher>`) + pub/sub channel `kill:events`.
- Self-healing: a loop republishes active halts from PostgreSQL into Redis
  every 10 s (a halt is a positive signal, never inferred from missing data).
- The `license_service` heartbeat consults the kill switch (`Hub.Killed()`,
  MGET on scope keys) and fails closed with
  `{"alive": false, "command": "halt"}` before any other lifecycle check, so
  a halt reaches every WL product within one heartbeat interval.

## 5. Two-Party Co-Sign for Withdrawals

Requirement: *"No one can withdraw any fund or revenue without TigerWallet
SuperAdmin collaboration."* Enforced at the **broadcast boundary** — the last
point before funds move — in `master_wallet/backend/license_gate.go`:

- Any fee/revenue/treasury withdrawal requires a valid SuperAdmin co-sign
  obtained from the control plane before the MasterWallet will broadcast.
- **Fail-closed:** if the control plane URL is unset or unreachable, the
  withdrawal is refused.
- Consequence: even a compromised WL admin key or MasterWallet owner key
  cannot move funds alone.

The transaction classifier (see §6) routes these kinds to MANUAL, so they can
never slip through the auto-sign path:

- `RevenuePayout`, `TreasuryTransfer`, `TreasurySweep`, `FeeWithdrawal`.

## 6. Auto-Approval & Per-Fetcher Feature Flags

- The MasterWallet auto-approve/auto-sign daemon
  (`master_wallet/backend/auto_signer.go`) resolves pending user-initiated
  transactions (UserTransfer, Swap, Stake, NftTransfer, PersonalSign,
  TypedDataSign) end-to-end **within a second**: approve → sign (EIP-1559
  secp256k1 / Ed25519) → broadcast → websocket push.
- **Velocity limits:** `checkAutoSignRules` enforces per-rule
  `max_txs_per_hour` and `max_value_per_day` (conditions JSONB, counted
  against the real `auto_sign_log` in PostgreSQL). Exhausted rules fall
  through; query errors fail closed.
- **Per-fetcher granularity:** SuperAdmin can disable any single trading
  vertical (futures, options, copy-trading, convert, onramp, offramp,
  p2p-clients, partners, rewards, marketing, kyc, tokens, pairs, blockchains,
  fees, withdrawals, admin-roles, …). The WL admin backend maps
  `/api/v1/admin/<domain>/...` to the domain fetcher key, so disabling
  `futures` leaves `options` alive (`white_label_admin/go/main.go`).
- **Feature flags** are authored in `super_admin/go` (CRUD over
  `feature_flags`, PostgreSQL) and published live to shared Redis keys that
  downstream services read.

## 7. Profit-Share Configuration

- SuperAdmin sets a per-WL-client profit share in the range **0–50%**
  (`super_admin/` governance UI + `admin/go/internal/services/super_admin_service.go`,
  `ProfitShareConfig`: `WhiteLabelID`, `SuperAdminWallet`, `ProfitPercentage`,
  `AutoTransfer`; default 20% in the model).
- Collected revenue flows through the fee/revenue/treasury accounting; actual
  payout transactions are classified `RevenuePayout` and therefore always
  require the two-party SuperAdmin co-sign (§5).

## 8. What the MasterWallet Cannot Do

Enforced by `guardUserFunds` in `master_wallet/backend/auto_signer.go`:

- The MasterWallet can **never** withdraw user funds — the daemon refuses to
  sign anything that moves funds out of a user sub-wallet to a destination
  not belonging to that same user. Fail-closed: on any doubt the transaction
  stays pending for manual review.
- The MasterWallet owner **cannot** withdraw fees or revenue without
  SuperAdmin permission (two-party co-sign, §5).

## 9. Architecture Diagram

```
                        ┌────────────────────────────────────────────┐
                        │         SUPER ADMIN  (top authority)       │
                        │  super_admin/web (39 pages)  →  super_admin│
                        │  /go :8082 (PostgreSQL + Redis)            │
                        │  • sign WL licenses (Ed25519)              │
                        │  • profit-share 0–50%                      │
                        │  • feature flags → Redis                   │
                        │  • kill switch (halt/resume, 4 scopes)     │
                        │  • co-sign withdrawals                     │
                        └───────┬───────────────┬────────────────────┘
                                │               │
                 ┌──────────────▼───┐     ┌─────▼──────────────┐
                 │ license_service  │     │   kill_switch :8469│
                 │ (Ed25519 signing,│◄────│ PG kill_state +    │
                 │ heartbeat, halt) │MGET │ Redis kill:* keys  │
                 └───────┬──────────┘     └────────────────────┘
                         │ heartbeat (alive / halt), fail-closed
        ┌────────────────┼───────────────────────────┐
        │                │                           │
┌───────▼────────┐ ┌─────▼──────────────┐  ┌─────────▼──────────────┐
│ PLATFORM ADMIN │ │ WHITE-LABEL ADMIN  │  │   MASTERWALLET :8450   │
│ admin/web (36) │ │ white_label_admin/ │  │ master_wallet/backend  │
│ admin/go :9093 │ │ web (28 pages)     │  │ • auto-signer daemon   │
│ RBAC:          │ │ white_label_id JWT │  │   (<1s, user txs only) │
│  super_admin   │ │ scope; 14 scoped   │  │ • guardUserFunds: can  │
│  admin         │ │ sub-admin roles;   │  │   NEVER move user funds│
│  support/      │ │ runs BEHIND the    │  │ • license_gate: every  │
│  analyst/      │ │ license gate (wlgate│ │   fee/revenue/treasury │
│  moderator =RO │ │  heartbeat, 503 on │  │   withdrawal needs     │
│                │ │  revoked license)  │  │   SuperAdmin co-sign   │
└───────┬────────┘ └─────┬──────────────┘  └─────────┬──────────────┘
        │                │                           │
        └────────────────┴────────► UserWallets (user_wallet/, wl_user_wallet/)
                                     auto-approved + auto-signed within a
                                     second; outgoing tx always shows
                                     "transaction submitted to the
                                     blockchain network"
```

## 10. Separation of Concerns

- Canonical functionality lives in `admin/`, `super_admin/`, `master_wallet/`.
  Dead, unreferenced copies of master/admin services inside user apps
  (Android, Flutter, desktop) were removed; user apps call the canonical
  backends over HTTP.
- The platform admin app **cannot** authorize MasterWallet admins, set
  profit-share, sign licenses, operate the kill switch, or co-sign
  withdrawals — those powers exist only in `super_admin/` +
  `license_service` + `kill_switch`.
- The WL admin app can only act inside its own `white_label_id` tenant and
  only while its license is alive.

See also: `admin/README.md`, `super_admin/README.md`,
`white_label_admin/README.md`, `master_wallet/README.md`,
`wl_control_plane/README.md`.
