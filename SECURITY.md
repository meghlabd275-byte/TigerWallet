# TigerWallet — Security Model

> Authoritative description of the key/seed architecture, the automatic-signing
> security model, the two-party co-sign boundary, the white-label license gate,
> the kill switch, and tenant isolation. Statuses are evidence-based.

---

## 1. Key & seed architecture (Phase 22)

The platform distinguishes and never mixes these key classes:

| Key class | Ownership | Storage | Exposed via |
|---|---|---|---|
| **User wallet keys** (24-word seed → private keys) | The user | AES-256-GCM encrypted seed in PostgreSQL; client-side encrypted backup (Google Drive / file / keystore) | Never returned after wallet creation; re-export only via explicit encrypted-seed export |
| **Platform operational keys** | TigerWallet | Secrets manager / env | Internal only |
| **MasterWallet treasury hot-wallet key** | MasterWallet owner | env `MASTER_WALLET_TREASURY_KEY_HEX` | Fail-closed 503 when unset |
| **License-signing Ed25519 private key** | SuperAdmin control plane only (`license_service/go/internal/crypto`) | Never leaves the control plane | Baked **public** key ships in every WL product |
| **HSM / MPC / AA signers** | Per-integration | HSM / MPC nodes / paymaster | `master_wallet/backend`, `go/mpc`, `account_abstraction/` |

**24-word seed = all chains.** One BIP-39 mnemonic derives addresses for every
supported chain (EVM BIP-44 `m/44'/60'/…`, Solana SLIP-10 Ed25519, Bitcoin
P2PKH, Cosmos bech32, etc.). If a user loses their seed, they lose control of
their wallet — by design (self-custody).

Seeds and private keys are **never** written to logs, analytics, API responses,
plaintext DB columns, frontend state, or telemetry. `user_wallet/web` clears the
mnemonic from memory immediately after backup confirmation.

---

## 2. Self-custody vs. delegated signing (Phase 23)

The architecture uses **two distinct signing models**, and they never mix:

1. **UserWallet — user-authorized signing.** UserWallet requests are forwarded
   to `go/wallet_api` `/api/v1/sign` + `/api/v1/send` (EVM) and
   `/api/v1/non_evm/*` (non-EVM). Keys are derived/decrypted server-side from
   the user's encrypted seed at the moment of signing.

2. **MasterWallet — policy-gated delegated auto-signing.** The MasterWallet
   auto-signer daemon (`master_wallet/backend/auto_signer.go`) resolves pending
   user-initiated requests end-to-end (approve → sign → broadcast → push)
   within ~1 second. This is a **centralized, policy-gated signing path** — a
   deliberate architectural conflict with pure self-custody that is resolved by
   hard, fail-closed policy guards (§3).

The auto-signer is **not** unrestricted. See §3.

---

## 3. Automatic-signing security controls (Phase 23 — VERIFIED)

`master_wallet/backend/auto_signer.go`:

| Control | Enforcement |
|---|---|
| Transaction classifier | `UserTransfer`, `Swap`, `Stake`, `NftTransfer`, `PersonalSign`, `TypedDataSign` are auto-approvable. `RevenuePayout`, `TreasuryTransfer`, `TreasurySweep`, `FeeWithdrawal` are **never** auto-approved |
| User-funds guard (`guardUserFunds`) | Daemon refuses to sign anything that moves funds out of a user sub-wallet to a destination not belonging to that same user |
| Velocity limits (`checkAutoSignRules`) | Per-rule `max_txs_per_hour` + `max_value_per_day` counted against the real `auto_sign_log`; exhausted rules fall through; query errors fail closed |
| Replay/nonce protection | EIP-1559 nonce management per sub-wallet in `signAndBroadcast` |
| Fail-closed | On any doubt the tx stays pending for manual review; if `MASTER_AUTO_SIGN_PASSWORD` is unset, approvals still record but broadcast is disabled |

**Residual risk to document:** the auto-signer keys are held server-side, so the
security of the delegated path rests on the backend's own key protection,
network isolation, and the velocity limits. This is an operator-accepted
trade-off per product requirements ("UserWallet always gets automatic sign and
automatic approval within a second").

---

## 4. Two-party SuperAdmin co-sign (Phase 25 — VERIFIED)

*"No one can withdraw any fund or revenue without TigerWallet SuperAdmin
collaboration."*

Enforced at the **broadcast boundary** in `master_wallet/backend/license_gate.go`:
every `FeeWithdrawal` / `RevenuePayout` / `TreasuryTransfer` / `TreasurySweep`
requires a valid SuperAdmin co-sign before broadcast, and **fails closed** when
the control plane URL is unset/unreachable. A compromised WL admin key or
MasterWallet owner key therefore cannot move funds alone.

---

## 5. White-label license gate (Phase 29/31 — VERIFIED)

- Licenses are **Ed25519-signed tokens** (`license_service/go/internal/crypto`).
- WL products embed the **public** key and heartbeat to
  `POST /api/v1/license/heartbeat`; the answer is `alive` or `halt`.
- **Fail-closed:** suspended/revoked/expired license, or any heartbeat failure,
  ⇒ the gate refuses to serve (403/503). Missing config ⇒ refuse, never permit.
- The gate is implemented in three languages in lockstep: C++
  (`wl_control_plane/cpp`), Rust (`wl_control_plane/rust`), Go
  (`wl_shared/go/wlgate`).
- **Self-resume is rejected** at the store layer — only SuperAdmin can
  reactivate a suspended product or license.

### WL boundary (Phases 30–32)

WL clients may only access their own tenant (users, wallets, bots, projects,
listings, config, admin). They **cannot**: access TigerWallet SuperAdmin
functionality, other tenants' data, TigerWallet secrets, mint/resell licenses,
or forge license tokens. Tenant isolation is enforced by a `white_label_id`
JWT claim + `TenantScope` middleware (not by convention).

---

## 6. Kill switch (Phase 29 — VERIFIED)

`kill_switch/` (:8469), SuperAdmin-auth only (HS256 JWT, role `superadmin`).
Four scopes — `global`, `client`, `product`, `fetcher` — with durable state in
PostgreSQL (`kill_state`, `kill_events`), sub-second propagation via Redis keys
+ `kill:events` pub/sub, and a 10-second self-healing republisher. The
`license_service` heartbeat consults the kill switch and fails closed with
`{"alive": false, "command": "halt"}`.

---

## 7. AuthN / AuthZ / secrets (Phase 41)

See `ADMIN_ARCHITECTURE.md` for the role hierarchy:

- **Platform admin RBAC** (`admin/go/internal/middleware/auth.go`):
  `super_admin` (full) > `admin` (read+write) > `support`/`analyst`/`moderator`
  (read-only). JWT validated; 401 on missing/invalid, 403 on insufficient role.
- **WL scoped roles (14)** (`white_label_admin/go/internal/roles/roles.go`):
  `wl_client` owner + 13 sub-admin scopes; every route gated by
  `RequireScope(...)`.
- **permission_bridge** fail-closed: `X-API-Key` must map to an enabled product;
  `SUPER_ADMIN_SECRET` bearer for super-admin routes.

**Secrets policy:** no hardcoded credentials anywhere. JWT/DB/Redis/provider
secrets are sourced from env (`JWT_SECRET`, `DATABASE_URL`, `REDIS_*`,
`COINGECKO_API_KEY`, `ETHERSCAN_API_KEY`, `EVM_RPC_URL`, etc.) and should be
injected by the deployment secrets manager. See `ENVIRONMENT.md`.

**Known residual security findings** (tracked in `GAPS.md`): see the P0–P2
register there, including unlicensed `selfhosted_masterwallet`, and the
hardcoded `admin/go` billing-plan seeding.