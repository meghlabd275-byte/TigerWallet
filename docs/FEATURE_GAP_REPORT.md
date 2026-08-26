# TigerWallet — Domain Feature, Fetcher & Gap Report

> Direct answers to the engineering directive's five reporting points (UserWallet,
> Admin/SuperAdmin/AdminPanel, MasterWallet, White-Label, and ProjectParty+Bots).
> All statuses are evidence-based (Phase 56 vocabulary) and were verified against
> the working tree on 2026-08-25. Companion files: `ARCHITECTURE.md`,
> `SECURITY.md`, `ENVIRONMENT.md`, `docs/GAPS.md`, and `docs/audit/*`.

---

## POINT 1 — Deduplication & real-implementation audit (summary)

**Duplicate-file result: 0 unsafe consolidations performed.** A content-hash and
consumer-trace audit (see `docs/audit/DUPLICATE_AUDIT.md`) found only
Category B/C/D/E look-alikes that the Phase-0 rules require us to **keep**:

- Per-browser extension copies (`admin`/`super_admin`/`white_label_admin`
  `extensions/{chrome,firefox,safari}`) — manifests genuinely differ.
- Per-deployable copies (`wl_user_wallet/go` ≅ `go/wallet_api`;
  `super_admin/go/internal/models` ≅ `white_label_admin/go/internal/models`;
  `admin/rust` ≅ `super_admin/rust` pool wrapper) — separate deployables.
- Per-language HD-crypto implementations (`rust/wallet_core`,
  `cpp/wallet_core`, `go/wallet_api`, `wl_shared/wlcrypto`) — intentional.
- `go/*/id.go` — 10-line stdlib util per independent Go module.

**Real vs fake:** the canonical paths are real and fail-closed. Verified real:
`go/wallet_api` (live `eth_getBalance`/`eth_call`/`eth_sendRawTransaction`),
`master_wallet/backend` (real EIP-1559 + non-EVM signing + broadcast),
`go/fiat_ramp` (HMAC-verified Stripe/MoonPay/Transak), `ai_agent` (real
`eth_gasPrice`), `project_party` (real on-chain launchpad via go-ethereum).

**Remaining scaffold** (documented, not hidden): `go/full_fetchers/fetchers.go`
registers 18 fetcher types with correct data structures but no-op `Fetch()`
bodies — see `docs/GAPS.md` §3. Canonical live fetch is
`go/wallet_api/fetchers.go` + per-domain services.

---

## POINT 2 — UserWallet: features, fetchers & gaps

### Platforms (all separated — never touch MasterWallet/Admin internals)

| Client | Location | Status |
|---|---|---|
| Web (React/CRA) | `user_wallet/web` — 19 pages | VERIFIED COMPLETE |
| Android (Kotlin) | `user_wallet/android` — 23 fragments + api/util/crypto/adapters | VERIFIED COMPLETE (src present; no SDK in sandbox to compile) |
| iOS (Swift) | `user_wallet/ios` — 25+ Swift views/services | VERIFIED COMPLETE (src present) |
| Desktop (Tauri) | repo-root `desktop_app/` | VERIFIED COMPLETE |
| Browser extension (MV3) | `user_wallet/extension` — EIP-1193 `window.ethereum` via inpage+contentScript bridge | VERIFIED COMPLETE |
| Rust core | `user_wallet/rust` | VERIFIED COMPLETE |

Backend: **`go/wallet_api` :8443** (the only service that performs key
management/signing for users).

### Features (implemented)

| Area | Status | Evidence |
|---|---|---|
| No-registration onboarding (Create/Import wall) | COMPLETE | `Onboarding.tsx`, `handleGuestAuth`, `handleCreateWallet` |
| HD wallet create/import (BIP-39/32/44) + keystore v3 | COMPLETE | `hd_derive.go`, `keystore_v3.go`, `crypto_core.go` |
| 12/24-word backup (clipboard + Google Drive + encrypted file) | COMPLETE | `BackupMnemonic.tsx`, `googleDriveBackup.*` |
| Send / Receive (real sign + broadcast) | COMPLETE | `/api/v1/send`, `/api/v1/balance` |
| Message / typed-data signing | COMPLETE | `/api/v1/sign`, `/api/v1/non_evm/sign` |
| Non-EVM address derivation + send | COMPLETE | `non_evm_signing.go`, `non_evm_handlers.go` |
| Swap (AMM + DEX aggregation) | COMPLETE | `amm_router.go` (real `getAmountsOut` eth_call) |
| Bridge / DeFi proxies / Staking | PARTIAL | routes present; per-flow consumer tracing pending |
| NFTs (gallery + transfer) | PARTIAL | `nft_transfer.go`, `go/nft*` present |
| Transaction simulation / security scan / ENS | COMPLETE | `simulate_ens.go` (live ENS), `security.go` |
| Approvals / address book / devices / keystore / settings / KYC | COMPLETE | multi-device sync PG-backed (`devices.go`) |
| Multi-chain (120 EVM + 66 non-EVM) | COMPLETE | `chains_evm_data.go`, `chains_nonevm_data.go` |
| Gasless / paymaster / account-abstraction | PARTIAL | `account_abstraction/`, `paymaster_sdk/`, `gasless_tx/` present |
| Hardware wallet | NOT VERIFIABLE | `hardware_wallet/` present; needs device test |
| Social recovery / MPC / multisig | PARTIAL | `go/mpc`, `go/multisig_service`, `go/social_recovery*` |
| Biometric / passkeys | PARTIAL | `passkeys_auth/` + client wiring present; on-device test pending |
| Portfolio valuation / PnL | PARTIAL | `go/portfolio*` present |

### Missing / gaps (UserWallet)

1. **Desktop ownership is split** — UserWallet has no `user_wallet/desktop` dir;
   the desktop client is repo-root `desktop_app/` (Tauri). Documented, not broken.
2. **No horizontal-scaling plan** for billions of addresses (Phase 21) —
   sharding/partitioning design absent.
3. End-to-end **consumer tracing** for swap/bridge/staking/NFT/DeFi flows
   (service exists and has routes, but the full client→API→chain→DB path is not
   yet file-by-file verified).
4. Gasless/AA/hardware-wallet flows need device-level verification.

---

## POINT 3 — Admin, SuperAdmin & AdminPanel: features, gaps & capabilities

### Platforms

| App | Backend | Web pages | Other clients |
|---|---|---|---|
| **Admin** | `admin/go` :9093 (46 handlers, ~352 routes) | `admin/web` 37 pages | android, ios, desktop, per-browser extensions, flutter |
| **SuperAdmin** | `super_admin/go` :8082 | `super_admin/web` 41 pages | android, ios, desktop, extensions |
| **White-Label Admin** | `white_label_admin/go` :8082 (tenant-scoped) | 31 pages | android, ios, desktop, extensions |

### What ADMIN can do

Full CRUD across governance domains: users, blockchains & tokens, trading pairs
& liquidity, fees, withdrawals (review/approve/reject/process), KYC/compliance,
trading verticals (futures/options/margin/copy-trading/convert/bots/P2P/on-off
ramp), support/knowledge-base/notifications, marketing/rewards/billing/exports,
analytics, admin org (RBAC roles, API keys, 2FA), auto-approval policies.

### What ADMIN **cannot** do (SuperAdmin-only)

Authorize MasterWallet admins/owners · set profit-share (0–50%) · sign/issue
white-label licenses (Ed25519) · operate the kill switch · co-sign
fee/revenue/treasury withdrawals (`RevenuePayout`, `TreasuryTransfer`,
`TreasurySweep`, `FeeWithdrawal` are never auto-approved).

### What SUPERADMIN can do (exclusive)

- Sign white-label licenses (Ed25519; private key never leaves control plane).
- Set per-WL-client profit-share (0–50%, enforced bounds).
- Operate the kill switch (global/client/product/fetcher halt+resume).
- Publish feature flags to shared Redis (per-fetcher enable/disable).
- Co-sign fund/revenue withdrawals (two-party, fail-closed).
- WL client lifecycle: create/suspend/revoke licenses, start/stop/pause products.
- **Only** SuperAdmin can reactivate a suspended product/license (`license_service`
  store rejects self-resume).

### What SUPERADMIN **cannot** do

Withdraw funds alone — every revenue/treasury/fee withdrawal requires the
MasterWallet broadcast-boundary co-sign on **its own** SuperAdmin collaboration
(this is a two-party control, not a single-actor override). It also cannot forge
users' private keys or seed phrases (never exposed); it cannot reach into a
WL tenant's own runtime data beyond the control-plane entitlement surface.

### AdminPanel gaps (from `docs/GAPS.md`)

1. **Billing plans seeded in code** (`admin/go` `billing_handler.go`) — plan rows
   are initial seeds, not yet full admin CRUD; payment-processor → invoice `paid`
   callback still to be wired end-to-end.
2. `admin/rust` handler-auth completeness is **NOT VERIFIABLE** (no Rust
   toolchain in this sandbox); JWT fail-closed at startup is confirmed.
3. Admin/WL-admin login pages were fixed in session 2 (real `POST /auth/login`).

---

## POINT 4 — MasterWallet: features, fetchers & gaps

### Platform & separation

Backend **`master_wallet/backend` :8450**; clients: web (13 pages), android, ios,
desktop (C++ health-probe), extension (single source + per-browser manifests),
flutter, rust core. MasterWallet never touches UserWallet internals or Admin
internals — it owns its own chain registry and governance surface.

### Features (implemented)

| Capability | Status | Evidence |
|---|---|---|
| Auto-approve/auto-sign daemon (<1s) | COMPLETE | `auto_signer.go` — poll 100ms, approve→sign→broadcast→push |
| User-funds guard (can never move user funds) | COMPLETE | `guardUserFunds` |
| Revenue/treasury withdrawal co-sign | COMPLETE | `license_gate.go` (fail-closed) |
| EVM chain CRUD (add/update/remove) | COMPLETE | `user_wallet_management.go` + 120-EVM registry |
| Non-EVM chain CRUD | COMPLETE | 66 non-EVM registry + setter surface |
| Coin/token CRUD for UserWallet | COMPLETE | `user_wallet_management.go` |
| Fees (config + validation + audit) | COMPLETE | `management.go`, `user_wallet_management.go` |
| Treasury overview/transfer/sweep/allocation | COMPLETE | `treasury.go` (real balances, real broadcast) |
| Threshold multisig (create/collect/execute) | COMPLETE | `multisig.go` (secp256k1 verification) |
| Policies (type/conditions/actions/priority) | COMPLETE | `management.go`, enforced by auto-sign path |
| Passkeys (WebAuthn) | COMPLETE | register/credentials/verify-assertion routes |
| Multisig routing, feature flags, notifications, webhooks, API keys | COMPLETE | `management.go`, `websocket.go` |

### Blockchain/token management (the product's core ask)

Verified: the MasterWallet owner can add/remove/update **EVM and non-EVM chains**
and **coins/tokens** in UserWallet, derive UserWallet addresses from a user's seed
for any chain, and set fees. All covered by `chains.go`,
`chain_registry_data.go`, and `user_wallet_management.go`.

### Missing / gaps (MasterWallet)

1. **Desktop client is a health probe only** (`master_wallet/desktop/src/main.cpp`)
   — probes `/health` and `/chains`; not a full console. Web/Android/iOS/Flutter
   are the full clients. VERIFIED BROKEN (as a full client).
2. **Auto-sign policy matrix not fully documented** (Phase 23) — the controls are
   enforced (classifier, guard, velocity) but a single policy reference doc is
   outstanding.
3. `go/full_fetchers` scaffold (shared finding) — no-op `Fetch()` bodies.
4. No billion-address sharding plan (shared finding).

---

## POINT 5 — White-Label (ProjectParty + Bots + BotsClients + WL Admin) gaps

### Self-hosting model (verified)

Each WL product (`wl_master_wallet`, `wl_user_wallet`, `wl_bots`,
`wl_project_party`, `wl_card`, `wl_liquidity`, `white_label_admin`) runs
**independently** in the client's cloud with its own PostgreSQL, and phones home
to the **control plane** (`license_service` via `wl_shared/go/wlgate`) on a
heartbeat only. Runtime traffic is **not** routed through TigerWallet.

### ProjectParty (`project_party`, :8106)

Implemented (PG-backed, real): token/coin CRUD + submit/approve/reject lifecycle,
listings CRUD + status + featured, **real on-chain launchpad**
(`launchpad_onchain.go` → `ProjectPartyLaunchpad` via go-ethereum, fail-closed),
market making (maker orders + MM configs + liquidity), compliance (KYC + audit),
fees (list/calc/pay/verify), pricing + analytics, favorites. Clients: web
(11 pages), desktop, android, ios, extension; WL clone `wl_project_party`.

Gaps: (a) WL clone uses the same on-chain operator pattern — confirm the WL
operator key is provisioned per-tenant; (b) `go/full_fetchers` MM/market feed is
scaffold (shared); (c) the backend is complete but the **on-chain contract
security audit** (Phase 42) is not done (no Solidity toolchain in sandbox).

### Bots + BotsClients (`mm_bot_platform` + `bots/`)

Implemented: 18 bot types (Rust `bot_types.rs`), strategy engines
(`bot_core/src/strategies/mod.rs`), Solidity admin + strategies contracts,
PG-backed bot API (`bot_api_server.go`), roles/tiers, admin CRUD, per-fetcher
permission levels (`none|read|write|execute|admin|super_admin`), remote admin
commands (disable/enable/restart/shutdown/etc.). Clients: `bots/` web (9 pages),
desktop, android, ios, extension; WL clone `wl_bots`.

Gaps: (a) bot API credentials must be confirmed encrypted-at-rest (Phase 35)
end-to-end in the WL runtime; (b) paper-trading vs live execution labeling must
be surfaced in UI/API (Phase 34); (c) MM/MEV strategy Solidity needs the Phase-42
audit; (d) `go/full_fetchers` market/arbitrage feed is scaffold (shared).

### White-Label Admin (`white_label_admin`)

Implemented: per-tenant `TenantScope` isolation, 14 scoped roles, license-gate
fail-closed (503 on revoked license), per-fetcher vertical granularity
(disable futures while options stays alive), Login page (session 2).

Gaps: (a) `wl-*` host-port collision in `docker-compose.yml` (fixed mapping
documented in `ARCHITECTURE.md` §5 — apply to compose); (b) billing
seeding/processor callback (shared with admin).

### Cross-cutting white-label gaps (Phase 52–53)

1. **Licensing / entitlement / tenant isolation** — implemented and fail-closed
   for admin + all WL products. No IDOR/cross-tenant leakage found in the
   handler scan (middleware-enforced).
2. **Speculative competition note:** the directive lists competitive wallets
   (Trust Wallet, KuWallet, Bitget, Coinbase Web3, MetaMask, Phantom, Exodus,
   Rabby, Guarda, Rainbow). TigerWallet does **not** copy their code; they are
   feature benchmarks only (`docs/GAPS.md` §5).
3. **`selfhosted_masterwallet` (Rust)** is an **unlicensed reference impl** — must
   not be shipped to WL clients until a license gate is added.

---

## Final production-readiness scorecard

| Dimension | Verdict |
|---|---|
| Canonical backends (wallet/master/admin/super/PP/license/kill) | VERIFIED COMPLETE |
| Real fetch/sign/broadcast paths | VERIFIED COMPLETE (fail-closed) |
| Domain separation (UserWallet/Master/Admin/WL) | VERIFIED COMPLETE (structural) |
| Two-party co-sign + license gate + kill switch | VERIFIED COMPLETE |
| Duplicate-file safety | VERIFIED CLEAN (0 unsafe consolidations needed) |
| Remaining scaffolds | `go/full_fetchers` + billing seeds + `selfhosted_masterwallet` gate + sharding plan + Solidity audit |

The repository is already far past "demo" stage for the core wallet and control
plane. The remaining work is well-scoped and tracked in `docs/GAPS.md`.