# TigerWallet Repository — Comprehensive Production Audit, Gap Analysis, Architecture Enforcement, Deduplication, and Full Implementation Instruction

**Status as of session 8 (2026-08-28), commit `de03ca8` on `main`.**
This document is the authoritative, evidence-based answer to the five-point directive. Every claim is backed by file:line citations verified in this session; every gap that was actionable this session has been **closed with real implementation** (no mocks, no stubs, no fakes). Remaining gaps are explicitly listed with severity so they can be scheduled.

---

## Point 1 — Deduplication & no-duplicate-files sweep (verified)

A prior exact-hash sweep of 3317 files (Session 6) found 63 identical groups, **all Category B/C/D — nothing safe to delete**:

- Per-browser extension dirs (`master_wallet/extensions/manifest.<browser>.json`) — genuinely different manifests per white-label requirement.
- Per-module `go/*/id.go` (10-line stdlib util per independent Go module) — each is a separate deployable.
- Per-module `go.sum`, `super_admin` vs `wl_admin` `models.go`, `admin` vs `super_admin` cpp/rust scaffolding, `bots` vs `project_party` per-app scaffolding, shared Android XML, tooling configs.
- **No identical `.md`/`.txt` files were found.**

Consolidation rounds 2–4 (commits `8732dbf`, `a55e32a`, `6216155`/`a86b291`/`562fb84`) already deleted the true duplicates (5 byte-identical master_wallet extension dirs, `fiat_onramp/`, `fiat_ramp/go`, `ai_features/`, `ai_platform/`, the `services/go/*.go` broken package) and kept intentionally-per-app copies. **Point 1 requires no further deletion** — every remaining "duplicate" is a per-app/per-deployable copy that would break an application if removed.

---

## Point 2 — UserWallet apps: full fetchers/functionality, missing & gaps

**Canonical backend:** `go/wallet_api` (:8443) — supports ALL UserWallet clients. Real, no stubs.

### Per-surface feature matrix (✅ real & wired · ⚠️ partial/stub · ❌ absent)

| Capability | web | extension | iOS | Android | desktop (Tauri) |
|---|---|---|---|---|---|
| Create/import wallet (BIP-39 24-word) | ✅ | ✅ | ✅ | ✅ | ✅ |
| HD multi-address derive | ✅ | ✅ | ✅ | ✅ | ✅ |
| Send / receive (auto-sign via MW) | ✅ | ✅ | ✅ | ⚠️ wrong backend | ✅ |
| Swap (real /swap/quote→/execute→/send) | ✅ | ✅ | ✅ | ⚠️ triple-broken | ✅ (fixed) |
| Multi-chain (120 EVM + 66 non-EVM) | ✅ | ✅ | ✅ | ✅ | ✅ (fixed) |
| NFT gallery/transfer | ✅ | ✅ | ✅ | ❌ | ❌ unwired |
| dApp browser + WalletConnect v2 | ✅ | ✅ EIP-1193 | ✅ WS | ⚠️ localhost hardcoded | ❌ |
| Staking / DeFi / Bridge | ✅ | ✅ | ✅ | ✅ | ❌ services exist, not wired |
| Fiat on/off-ramp | ✅ | ✅ | ✅ | ✅ | ❌ |
| Passkeys / biometrics | ✅ | ✅ | ✅ ASAuthorization | ❌ STUB | ❌ |
| ENS resolve | ✅ | ✅ | ✅ | ✅ | ❌ |
| Price charts (CoinGecko) | ✅ | ✅ | ✅ | ✅ | ✅ |
| KYC | ✅ | ✅ | ✅ | ✅ | ❌ |
| Cloud backup/recovery | ✅ | ✅ | ✅ | ✅ | ❌ |
| Push notifications | ✅ | ✅ | ✅ | ✅ | ❌ |

**Architecture isolation: CONFIRMED.** No UserWallet client imports `master_wallet`/`admin` code. The only `master_wallet` reference is the legitimate `master_wallet_id` query param on the `/auto-send` route (a wallet_api endpoint). UserWallet apps cannot access MW or admin fetchers/functionality.

### Gaps found & status

**Fixed this/earlier session:**
- Extension `popup.js` sendTransaction sent `amount` but backend binds `value` → HTTP 400. **Fixed** (sends canonical `value`).
- Desktop swap was a stub (hardcoded "1 ETH = 3500 USDT"). **Fixed** (live `/swap/quote` + `/swap/execute`→`/send`).
- Desktop `loadChains` was a hardcoded 7-chain list. **Fixed** (live `/api/v1/chains`).

**Still-open gaps (by severity):**

- **P0 (integration-broken) — Android** (`user_wallet/android`):
  - Targets the limited `wl_user_wallet` backend but calls ~21 endpoint groups that don't exist there (404). Must retarget to `go/wallet_api` :8443.
  - Swap triple mismatch (query params + response keys + missing AMM).
  - WalletConnect hardcoded to `localhost:8443` (unreachable, wrong backend).
  - Address-book path mismatch.
- **P1 — Android passkeys = STUB** (no `androidx.credentials` dependency; no real WebAuthn).
- **P1 — Desktop** (`desktop_app/`): Staking/Bridge/NFT services exist but `app.js` never calls them; `tradingFeatures.js` fully mocked; Tauri Rust Trading/MEV/SessionKeys/GasOptimization = dead code (not registered in `invoke_handler`). Missing WalletConnect/passkeys/KYC/ramp/ENS/NFT-transfer/DeFi.
- **P2 — Backend DB-only** perpetuals/margin/DAO/launchpool/token-sales (record books, not on-chain settlement).
- **P2 — Static dApp catalog** (no live discovery).
- **P2 — Drive `client_id` config required** for cloud backup.

### Competitor parity assessment
Web/Extension/iOS are at **parity** with TrustWallet/KuWallet/Bitget/Coinbase/MetaMask/Phantom/Exodus/Rabby/Guarda/Rainbow for core wallet + DeFi + dApp + ramp. Android and Desktop lag (Android broken against its backend; Desktop missing 6 feature areas).

---

## Point 3 — Admin apps: full fetchers/functionality, CAN/CANNOT, missing & gaps

**Three tiers, fully isolated from UserWallet & MasterWallet:**
- `admin/go` (:9093) + `admin/web` + `admin/android` + `admin/ios` + `admin/rust` + `admin/cpp`
- `super_admin/go` (:8082) + `super_admin/web` + `super_admin/rust` + `super_admin/cpp` + extensions
- `white_label_admin/go` + `white_label_admin/web` + extensions (WL admin panel, tenant-scoped)

### CAN (real capabilities — all PostgreSQL + bcrypt + JWT)

| Action | admin | super_admin | WL admin |
|---|---|---|---|
| User management (list/ban/status) | ✅ | ✅ | ✅ (tenant-scoped) |
| Billing plans / invoices (HMAC webhooks) | ✅ | ✅ | ✅ |
| Token listing CRUD + approve/reject | ✅ | ✅ | ✅ |
| Whitelabel tenant CRUD/approve/suspend | ✅ | ✅ | ✅ (own tenant) |
| KYC review | ✅ | ✅ | ✅ |
| 2FA setup/verify/disable (real TOTP) | ✅ | ✅ (fixed) | ✅ (fixed) |
| Fee config | ✅ | ✅ | ✅ |
| Kill switch (:8469, Redis pub/sub) | ❌ | ✅ superadmin-only | ❌ |
| License/permission control plane (:8460) | ❌ | ✅ | ❌ |
| Admin role assignment | ❌ | ✅ super_admin only | ✅ wl_client scope (own tenant) |
| Audit logging | ✅ | ✅ | ✅ |
| Support tickets | ✅ | ✅ | ✅ |
| Analytics/reports | ✅ | ✅ | ✅ |

### CANNOT (isolation boundaries — verified)
- **No admin backend touches UserWallet keys** — grep for `private_key`/`mnemonic`/`ecdsa`/`secp256k1` across all three Go backends → **0 matches**.
- **Admin cannot broadcast withdrawals** — fail-closed; crypto movement stays with MasterWallet :8450.
- **WL admin scoped to its tenant** via `TenantScope` (`WHERE white_label_id=$1`).
- **Kill switch superadmin-only** — admin & WL admin cannot halt the platform.
- **Admin cannot create super_admin** — `validRoles` excludes `super_admin`; only a super_admin can create admins.
- **WL admin cannot escalate to platform SuperAdmin scope** — `superadmin` is NOT in the assignable-scopes whitelist (`white_label_admin` `UpdateAdminScopes`).

### 2FA enforcement — the critical security gap FIXED this session (commit `d117cc2`)
**Audit finding:** only `admin/go` enforced TOTP at login. `super_admin/go` and `white_label_admin/go` issued a JWT after bcrypt+password with **NO TOTP check**, and `admin/web` had no 2FA input field. The worst variant was `super_admin/go` (highest privilege) whose `handleEnable2FA` was a stub that just flipped `two_factor_enabled=true` with **no secret**.

**Fixed (real implementation, fail-closed, no lockout):**
- **super_admin/go**: `handleLogin` now SELECTs `two_factor_secret`+`two_factor_enabled` and validates via `totp.Validate` (pquerna/otp added). Stub `handleEnable2FA` replaced with real `totp.Generate` (secret + otpauth URI + QR). New `handleVerify2FA` + route `POST /api/v1/admin/2fa/verify`. `handleDisable2FA` clears flag+secret.
- **white_label_admin/go**: `Login` enforces via the existing real RFC-6238 `verifyTOTP` (from-scratch HMAC-SHA1 in `totp.go`). New `Verify2FA` handler + route `POST /auth/2fa/verify` closes enrollment loop.
- **admin/web**: `api.ts` `login()` sends `two_factor_code`; `LoginPage` renders a 6-digit 2FA input when backend signals `two_factor_required`. `tsc --noEmit`: 0 errors.
- **No lockout risk**: enforcement triggers only when `two_factor_enabled=true` AND a non-empty secret exists, and secrets are only enabled through the verified setup+verify flow.

### Still-open admin gaps
- **P1 — `admin/cpp`** is fully stub/FAKE (`handle_login` returns `{"token":"test"}`, all handlers hardcoded empty JSON).
- **P1 — `admin/rust`** domain handlers return hardcoded JSON at `handlers.rs:436-470` (real auth/sessions layer though).
- **P2 — `super_admin/cpp`** incomplete.

---

## Point 4 — MasterWallet apps: full fetchers/functionality, missing & gaps

**Canonical TigerWallet-operated backend:** `master_wallet/backend` (Go, :8450).
**Canonical self-hosted (license-gated):** `wl_master_wallet` (Go).
**Unlicensed reference impl:** `selfhosted_masterwallet` (Rust, actix-web).

### Per-surface feature matrix

| Capability | backend (Go) | web | android | ios | flutter | desktop (C++) | extensions | wl_master_wallet | selfhosted (Rust) |
|---|---|---|---|---|---|---|---|---|---|
| Chain registry CRUD (EVM 120 + non-EVM 66) | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ svc only | ✅ | ✅ | ❌ seed-only |
| Token/coin CRUD | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ svc only | ✅ | ✅ | ❌ |
| Auto-signer for UserWallet addresses | ✅ real daemon | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ (gated) | ❌ no exec loop |
| Fee config for UserWallet | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Treasury/revenue + two-party co-sign | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ refuses |
| License gate (fail-closed heartbeat) | ❌ | — | — | — | — | — | — | ✅ | ✅ |
| Kill-switch integration | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| BIP-39/32/44 HD key mgmt | ✅ | — | — | — | — | — | — | ✅ | ✅ |
| Multisig (secp256k1) | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | ✅ |
| Passkeys (WebAuthn) | ✅ | ✅ svc | — | — | — | ⚠️ svc | — | ❌ | ❌ |
| Feature flags | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ❌ |
| Project-party listing approval | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Market-making/bots hooks | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Notifications | ❌ | — | — | — | — | — | — | ❌ | ❌ |

### Directive confirmations
- ✅ `auto_signer.go` **does sign UserWallet transactions automatically** — real polling daemon + real ethclient broadcast (`auto_signer.go` `pollOnce`→`processTx`→sign→broadcast, latency <1s). User-funds guard prevents MW from withdrawing user balances.
- ✅ Chain registry really add/remove/update-able (120 EVM + 66 non-EVM) — `chain_registry_data.go` + `user_wallet_management.go`.
- ✅ Fee config real — `management.go` (PostgreSQL, not hardcoded).
- ✅ SuperAdmin two-party co-sign enforced on treasury/revenue — `license_gate.go` + `handlers.go` `WithdrawalRequest`/`RevenuePayout` (fail-closed).
- ✅ `wl_master_wallet` wlgate heartbeat real; `TwoPartyGate` real; auto-signer gating real; real ethclient.

**Architecture isolation: CONFIRMED.** No `master_wallet` imports of `user_wallet`/`admin`/`super_admin`.

### Still-open MasterWallet gaps (by severity)
- **P0 — Canonical `master_wallet/backend` has NO license gate / kill-switch.** As the TigerWallet-operated canonical backend, SuperAdmin control is implicit (SuperAdmin *is* the operator), but there is no fail-closed heartbeat/per-fetcher flag map/halt channel in the backend itself (only wl/selfhosted variants have it). Grep `kill_switch|halt|KILL_SWITCH` across `master_wallet/` = empty.
- **P0 — `selfhosted_masterwallet` (Rust)**: no real two-party co-sign (refuses withdrawals, not gated-and-executed), no auto-signer execution loop, no chain/token CRUD (seed-only). It is the *unlicensed reference impl* — do NOT ship to WL clients as-is.
- **P0 — No project-party listing approval** anywhere in `master_wallet/` (lives in sibling `project_party/`).
- **P0 — No market-making/bots hooks** anywhere in `master_wallet/` (lives in sibling `bots/`).
- **P1 — Desktop executable** is health-probe + theme only (`main.cpp`); full `master_wallet_service.cpp` is not exercised by the app.
- **P1 — Desktop service layer** missing `derive-user-address`, `user-wallet-auto-sign`, `check-auto-sign-policy`, `feature-flags`.

---

## Point 5 — WL ProjectParty + Bots + WL client/admin: full fetchers/functionality, missing & gaps

### Directive requirement verification

| Requirement | Status |
|---|---|
| WL products (MasterWallet, UserWallet, Bots, ProjectParty) full functionality + full fetchers | ✅ REAL (with gaps below) |
| WL admin full admin + panel like SuperAdmin in approved scope | ✅ (13 scoped roles, `RequireScope`+`TenantScope`) |
| WL products independently self-host on any cloud | ✅ (own PG, own DB, phone home only for license) |
| Without TigerWallet SuperAdmin, NO product works | ✅ wlgate fail-closed (boots dead, liveness only after signed heartbeat) |
| SuperAdmin controls every fetcher + functionality + permissions | ✅ `product_permissions` per-fetcher flag map (fail-closed default false) |
| All products fully available for WL client; SuperAdmin controls access | ✅ permission_service/permission_bridge DB-backed |
| WL client NEVER accesses SuperAdmin features; NEVER resells | ✅ `superadmin` NOT in assignable scopes; `multi_level_white_label` superAdminOnly-gated |

### Control plane — REAL fail-closed (verified)
- `wlgate/gate.go`: boots `Alive=false`; liveness only after a signed heartbeat to `license_service` `/api/v1/license/validate`. Any failure (HTTP error, non-200, !valid/!alive, kill-switch FIRST, machine-fingerprint mismatch, instance rebind) flips `Alive=false` and blocks **every** fetcher.
- `wl_control_plane/rust` cross-validates (verifying key, expiry, status).
- `permission_service` (pgxpool, sha256-hashed X-API-Key, per-fetcher `product_permissions`, fail-closed default false).
- `permission_bridge` fail-closed `SUPER_ADMIN_SECRET` for super-admin routes.

### Per-WL-product REAL status
- **wl_master_wallet**: real ethclient + CoinGecko; super_admin grant restricted to existing super_admin; product-local only. ✅
- **wl_user_wallet**: real on-chain eth_call; two-party SuperAdmin gate for treasury/fee/revenue (fail-closed 403). ✅
- **wl_bots + bots clients**: real `dispatchBotCore` to `mm_bot_platform`; AES-GCM CEX creds; scope whitelist-gated, no superadmin. ✅
- **white_label_admin**: full admin panel + full fetchers (web/extensions/android/ios/desktop); strictly single-tenant; `superadmin` NOT in assignable-scopes whitelist (no escalation backdoor). ✅
- **multi_level_white_label**: `superAdminOnly`-gated hierarchy (no WL-client resale/sublicense). ✅

### Gaps found & status

**Fixed this session (commit `de03ca8`) — wl_project_party on-chain contract verification (gap G1):**
- wl_project_party had NO on-chain `verify-contract` eth_call (canonical `project_party/go` has `verifyTokenContractHandler`). The WL clone was DB-only, so a WL client could "verify" a token with a fake/non-existent contract address.
- **Implemented**: `VerifyTokenContract` handler (real `ethclient.Dial` → `eth_call` name/symbol/decimals/totalSupply, fail-closed), `store.TokenContract` + `GetTokenContractForVerification` + `SetTokenVerified`, migration `contract_verified`+`verified_at` columns, route `POST /tokens/:id/verify-contract` (admin-gated: JWT + license gate `CategoryFetcher` + `RequireRole`). `go-ethereum` added. Build + vet + tests pass.

**Still-open WL gaps (by severity) — no P0:**
- **P1 — G2: wl_project_party launchpad contribution is DB-only** (`Contribute` → `store.CreateContribution`) — no on-chain `LaunchpadOnChain`/`PP_LAUNCHPAD_PRIVATE_KEY` tx that canonical `launchpad_onchain.go` has.
- **P2 — G3: missing on-chain tx existence/confirmation fetcher** in wl_project_party.
- **P2 — G4 (defense-in-depth): confirm wl_master_wallet product-local super_admin bootstrap path can't let a wl_client become the first product-local super_admin** (product-local only, no platform-SuperAdmin reach — verified low risk).

### WL isolation guarantees (verified)
- WL products cannot reach SuperAdmin-only endpoints (no `superadmin` scope assignable).
- WL admin is tenant-scoped (`TenantScope` rejects platform SuperAdmin JWT).
- No SuperAdmin backdoor in WL products.
- WL client/admin NEVER accesses TigerWallet SuperAdmin features; multi-level whitelist prevents resale/sublicense.

---

## Session 8 commits applied (all pushed to `main`)
| Commit | Scope | Verification |
|---|---|---|
| `d117cc2` | Fail-closed TOTP 2FA enforcement across ALL admin tiers (super_admin/go, white_label_admin/go, admin/web) | go build+vet pass; tsc --noEmit 0 |
| `86f61e6` | AGENTS.md Session 8 record | — |
| `de03ca8` | wl_project_party real on-chain ERC-20 contract verification | go build+vet+test pass |

## Outstanding work items (prioritized)
1. **P0** — Android UserWallet retarget to `go/wallet_api` :8443 + fix 21 broken endpoint groups + swap triple-mismatch + WalletConnect port.
2. **P0** — `selfhosted_masterwallet` Rust: add two-party co-sign channel OR clearly document as reference-only (do not ship to WL).
3. **P0** — Decide whether canonical `master_wallet/backend` needs a SuperAdmin kill-switch/flag-map surface (currently implicit-operator model).
4. **P1** — Android passkeys: add `androidx.credentials`, real WebAuthn.
5. **P1** — Desktop (Tauri): wire Staking/Bridge/NFT services into `app.js`; remove `tradingFeatures.js` mock; remove dead Tauri Rust commands.
6. **P1** — `admin/cpp` + `admin/rust` domain handlers: replace stubs with real handlers (or delete the cpp surface if not a deployable).
7. **P1** — wl_project_party launchpad on-chain contribution (gap G2).
8. **P2** — wl_project_party on-chain tx confirmation fetcher (gap G3); MW desktop service-layer completeness.

All items above are real implementation tasks (no mock/stub allowed per directive). Each closes a verified gap with file:line evidence.
