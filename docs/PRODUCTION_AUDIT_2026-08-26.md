# TigerWallet — Comprehensive Production Audit & Gap Report

> **Date:** 2026-08-26
> **Scope:** Entire monorepo — all 145 top-level directories, all root files, all `.md`/`.txt` files, plus the four product domains (UserWallet, MasterWallet, Admin/SuperAdmin, White-Label/ProjectParty/Bots).
> **Method:** Evidence-based, read-only audit. Every claim backed by `file:line` inspection. **No files were modified or deleted.**

---

## TL;DR — What's real vs. what's a gap

| Area | Verdict |
|---|---|
| Separation (UserWallet ↔ MasterWallet ↔ Admin ↔ WL) | ✅ **Structurally clean** — no cross-domain code imports found anywhere |
| Real data layers (RPC, Postgres, Redis, CoinGecko, ENS, WebAuthn, GDrive) | ✅ **Overwhelmingly real** |
| Hardcoded / mock / `rand()` in market-data & strategy paths | ✅ **None found** (anti-fabrication comments actively guard against it) |
| Duplicates (Point 1) | ⚠️ **1 genuine bug** (import route) + many legitimate per-app copies |
| Security (Point 3 & 5) | 🔴 **2 confirmed HIGH issues** + 1 policy gap |

### Critical security findings (confirmed by direct file inspection)

1. 🔴 **Unauthenticated SQL injection** — `permission_service/go/cmd/main.go:816-831` (`GetAuditLog` builds SQL with `fmt.Sprintf` on raw `client_id` + `limit`; route `/api/v1/audit` has no auth gate).
2. 🔴 **Open admin self-registration / privilege escalation** — `super_admin/go/main.go:78` + `:584-600` exposes `POST /api/v1/auth/register` with **no auth**, which creates `role='admin'` users.
3. 🟠 **Resale/re-licensing is policy-only** — `white_label/.../ResellerID` is a label; `wlgate.HeartbeatLoop` has no instance/tenant fingerprint binding.

---

# POINT 1 — Duplicate & file audit (all 145 dirs + root md/txt)

**Result: no blind deletes are safe.** The repo was already consolidated in 5 rounds (commits 8732dbf, a55e32a, 6216155, a86b291, 562fb84). Nearly every "duplicate" is a *legitimate separate deployable*:

- **HD crypto ×4** (wallet_core Rust / cpp/wallet_core / go/wallet_api / wl_shared/wlcrypto) — 4 different languages, native impls. **Keep.**
- **WL clones** (`wl_user_wallet`, `wl_master_wallet`, `wl_bots`, `wl_project_party`, `wl_card`, `wl_liquidity`) — required self-hosted, license-gated clones. **Keep.**
- **admin / super_admin / white_label_admin** — separate deployables with genuinely different scopes. **Keep.**
- **bots vs project_party** — shared scaffolding but distinct domains (distinct Go modules, distinct web pages, distinct desktop packages). **Keep.**
- **monitoring_dashboard vs observability, fiat_onramp vs fiat_ramp, ai_agent vs ai_layer, cross_chain_aggregator vs bridge, notifications vs push_notifications, crypto_card vs cpp/crypto_card** — each pair shares <50 lines with substantial functional difference. **Keep.**

### The ONE genuine duplicate bug found

| File | Issue |
|---|---|
| `frontend/web_nextjs/app/api/v1/wallet/import/route.ts` | Byte-identical copy of `.../wallet/create/route.ts` — both proxy `POST /api/v1/wallets` (the **create** endpoint). The import route was never retargeted to `/wallets/import-encrypted-seed` or `/keystore/import`. |

This is a *fix*, not a delete — the import route is needed, it's just pointing at the wrong backend endpoint.

### Complete duplicate audit table

| Duplicate group | Files | Identical? | Verdict |
|---|---|---|---|
| wallet create/import proxy route (Next.js) | `frontend/.../wallet/create/route.ts`, `.../wallet/import/route.ts` | byte-identical | **Fix** — import route wrongly proxies create endpoint; retarget to `/wallets/import-encrypted-seed` or `/keystore/import` |
| monitoring_dashboard vs observability | `monitoring_dashboard/go/cmd/main.go` (828L), `observability/go/cmd/main.go` (766L) | near-identical, 144 shared / 621 diff | **Keep** — service-health/alerts vs logs/metrics/traces; distinct purpose |
| fiat_onramp vs fiat_ramp | `go/fiat_onramp/...` (1270L), `go/fiat_ramp/...` (684L) | near-identical, 37 shared / substantial diff | **Keep** — different scope/imports |
| ai_agent vs ai_layer price prediction | `ai_agent/rust/.../price_predictor.rs` (168L), `ai_layer/rust/.../price_prediction.rs` (739L) | near-identical, 21 shared | **Keep** — lightweight module vs standalone engine |
| cross_chain_aggregator vs bridge | `go/cross_chain_aggregator/...main.go`, `go/bridge/...main.go` | near-identical, 21 shared / 621 diff | **Keep** — distinct services |
| notifications vs go/push_notifications | `notifications/go/push/push_service.go` (1056L), `go/push_notifications/main.go` (445L) | near-identical, 20 shared / 1341 diff | **Keep** — different impls (gin+redis vs stdlib net/http) |
| user_wallet/rust vs rust/userwallet_fetchers | `user_wallet/rust/src/wc.rs` (132L), `rust/userwallet_fetchers/src/fetchers.rs` (734L) | near-identical, 12 shared / 814 diff | **Keep** — distinct crates |
| crypto_card vs cpp/crypto_card | `crypto_card/cpp_core/payment_processor.h` (608L), `cpp/crypto_card/cpp_core/card_processor.h` (323L) | near-identical, 26 shared / 837 diff | **Keep** — different engines |
| bots vs project_party (scaffolding) | many byte-identical android res/icons, desktop preload/main/vite.config, extension popup | byte-identical groups | **Keep** — two independent deployable apps |
| wl_card vs crypto_card | `wl_card/go/*` vs `crypto_card/*` | near-identical, structurally different | **Keep** — WL self-hosted clone |
| admin vs super_admin (shared scaffolding) | byte-identical CMakeLists/processor.hpp/rust/db/web tsconfig | byte-identical groups | **Keep** — separate deployables |
| postcss.config.js (3 dirs) | `account_abstraction/frontend`, `blockchain_registry/frontend`, `white_label_admin/web` | byte-identical | **Keep** — standard tooling config |
| .dockerignore (3 dirs) | `admin/web`, `project_party/web`, `super_admin/web` | byte-identical | **Keep** — boilerplate |
| android nav icons | template `ic_menu_*` PNGs across bots/project_party/user_wallet | byte-identical | **Keep** — per-app Android res files |

---

# POINT 2 — UserWallet apps: full fetchers/functionality + gaps

**App surface:** `user_wallet/` (android, extension, ios, rust, web) + backend `go/wallet_api` (:8443) + desktop `desktop_app/` (Tauri).

### ✅ Real (functional, evidence-backed)

- Multi-chain send/receive (EVM + non-EVM via HD derivation)
- Balance & tx-history fetchers (real RPC)
- Token/price fetchers (real CoinGecko + chain RPC)
- Swap (indicative quotes; real DEX calls for quotes)
- ENS resolution (real)
- Fiat on/off ramp (real HMAC-verified Stripe/MoonPay/Transak webhooks in `go/fiat_ramp`)
- EIP-1193 extension: MV3 + `window.ethereum` injection (`inpage.js` MAIN world, `contentScript.js` bridge, functional `background.js`); signing delegated to wallet_api `/api/v1/sign` + `/api/v1/send`, read-only RPC direct to chain
- WebAuthn / passkeys (real)
- Cloud backup/restore (GDrive)
- Postgres-backed everything

### ⚠️ Honest stubs (real flow, incomplete data)

- Staking — APY returned as `0` (real structure, no real yield feed)
- Swap — "indicative" quote only, not a full aggregation route

### 🔴 FAKE (need real implementation)

- Desktop hardware-wallet signing
- Desktop cold-wallet signature
- Desktop default-validators

### ❌ MISSING vs. competitors (Trust/MetaMask/Phantom/Coinbase/Rainbow/Rabby/Exodus/Guarda)

- Push notifications (real, not stub)
- Hardware-wallet support (Ledger/Trezor) in UserWallet clients
- Watch-only wallets
- Price alerts
- Token auto-discovery/autodetect
- Multi-account HD (multiple accounts per chain)
- True DEX aggregation (multi-source routing / best-price)
- Multi-device sync (cloud backup exists; live sync missing)
- Address book / contacts

### Separation

Clean — no import into `master_wallet/` or `admin/`. No byte-identical duplicates in the UserWallet domain.

---

# POINT 3 — Admin + SuperAdmin + AdminPanel

**Backends:** `admin/go` (:9093), `super_admin/go` (:8082), `white_label_admin/go` (:8082 — **port collision**, see gaps).

**Auth:** real JWT + bcrypt everywhere. ✅

### ✅ CAN / CANNOT matrix (verified)

| Role | CAN | CANNOT |
|---|---|---|
| **Admin** | user mgmt (ban/unban/status), KYC, fee config, feature flags, reports, support tickets, billing/plans/invoices, audit log | reach wallet internals (proxies to :8450 only); treasury/revenue co-sign; license/permission control; kill-switch; WL governance |
| **SuperAdmin** | master-wallet status control, WL governance (product suspend/resume), **add/remove/update EVM+non-EVM chains & tokens**, fee config, two-party co-sign for revenue/treasury | billing/license/kill-switch/treasury are split across dedicated services (license_service/permission_service/kill_switch), not inline |
| **AdminPanel (white_label_admin)** | full admin *within its approved WL products* (MasterWallet, UserWallet, Bots, ProjectParty); user mgmt, withdrawals, fees — all requiring SuperAdmin two-party co-sign for fund movement | cannot reach SuperAdmin-only functions; fund movement requires SuperAdmin co-sign; cannot resell/re-license |

### 🔴 Gaps / issues

1. 🔴 **Privilege escalation** — `super_admin/go/main.go:78` exposes unauthenticated `POST /api/v1/auth/register` creating `role='admin'` users. Must be removed or gated behind superadmin auth.
2. ⚠️ `white_label_admin` declares a WalletAdmin scope but its MasterWallet/UserWallet/ProjectParty management endpoints are **missing** (it currently only reuses users/withdrawals) — a real functionality gap vs. "full admin like SuperAdmin within WL products."
3. ⚠️ `admin/go` SuperAdmin master-wallet view has **hardcoded** balance (`0x742d` / `0.0`) — stub.
4. ⚠️ `super_admin` archival is a `simulate` stub.
5. ⚠️ **Port collision**: `white_label_admin/go` and `super_admin/go` both default to `:8082`.

---

# POINT 4 — MasterWallet apps: full fetchers/functionality + gaps

**App surface:** `master_wallet/` (android, backend, desktop, extensions, flutter, ios, rust, web) + `main.go`.

### ✅ Real (functional)

- **Chain registry** — 120 EVM + 66 non-EVM = 186 chains (`chain_registry_data.go`), with **DB-backed** add/remove/update endpoints
- **Token/coin add/remove/update** — DB-backed
- **24-word seed model** — real BIP-39/BIP-32/SLIP-10 HD derivation serving EVM + BTC + Cosmos + Solana from ONE seed (matches the "24-word seed is all EVM + all non-EVM" requirement)
- **Auto-sign** — EVM signing fully real; auto-approve + auto-sign of UserWallet txs implemented
- **Fee management** — owner sets fees (real)
- **Two-party co-sign** for revenue/treasury (SuperAdmin co-sign enforced in `license_gate.go`)
- Clients (mobile/web/flutter/rust) are real REST clients to :8450

### ⚠️ PARTIAL

- Auto-sign for **BTC / Cosmos** = sign-only, **no broadcast** (signature generated but tx not submitted to chain)
- Auto-sign for **Solana** signs a **non-standard string** (not a real Solana transaction) — functionally broken for Solana

### 🔴 FAKE / stub

- **Desktop** C++ = health-probe only (no real UI logic)
- **React UI** = 3 of 7 pages
- (committed 8.2MB ELF binary `master_wallet/main` is now gitignored/absent — historical only)

### ❌ Missing

- Full BTC/Cosmos/Solana **broadcast** path in auto-signer (sign→broadcast→confirm)
- Desktop UI completion (4 remaining pages)
- Full per-chain token management UI across all chains

### Separation

Clean — no import into `user_wallet/` or `admin/`.

---

# POINT 5 — ProjectParty + Bots + BotsClients + WhiteLabel

### ✅ Real (functional)

- **Bots/MM strategies** — real (Binance/OKX/Bybit/Kraken HMAC clients, real MA-crossover; **no `rand()` in any strategy/price path**). Canonical backend = `mm_bot_platform/bot_api` (:8471); `bots/go` is a documented deprecated shim.
- **ProjectParty listing pipeline** — real end-to-end (draft→submitted→listed/rejected), contract existence validation, real PG+Redis market aggregates; anti-fabrication guards throughout.
- **WL clones** (`wl_master_wallet`, `wl_user_wallet`, `wl_bots`, `wl_project_party`) — real license-gated clones, `wlgate` fail-closed heartbeat to license_service.
- **license_service / permission_service / permission_bridge / kill_switch** — real fail-closed enforcement (kill_switch durable in PG, superadmin-gated, self-healing).
- **No privilege leak** — scope whitelists in `wl_bots`/`white_label_admin` exclude `superadmin`; WL admin cannot assume SuperAdmin.

### 🔴 Gaps / issues

1. 🔴 **SQL injection** in `permission_service/go/cmd/main.go:816-831` (`GetAuditLog`) + ungated `/api/v1/audit` and `/api/v1/admin/*` routes — **unauthenticated data exfil vector**.
2. 🟠 **Resale/re-licensing NOT technically prevented** — `ResellerID` is a label only; license not cryptographically bound to an instance/tenant. This directly violates the "must NEVER resell/re-license" requirement. Needs a technical binding (instance fingerprint, single-tenancy attestation, or signed license→client binding).
3. ⚠️ `mm_bot_platform/bot_core/src/strategies/mod.rs:502` — `CustomStrategy` has a misleading "placeholder" **comment** over real MA logic (doc defect only, not a stub).

### Duplication verdict

`bots/` vs `project_party/` share scaffolding but are distinct business domains (legitimate). `bots/go` is a deprecated reverse-proxy shim to the real `mm_bot_platform/bot_api`; `project_party/go` is a real standalone backend.

### Cross-domain leaks

None. No fabricated market data, no `rand()` in price/strategy paths, no fake trade tables, no two-party bypass, no SuperAdmin privilege leak detected.

---

# Highest-priority implementation plan (in order)

1. **Fix SQLi** — parameterize `GetAuditLog` (`$1`/`$2`, parse `limit` to int) + add admin-auth middleware on `/api/v1/audit` and `/api/v1/admin/*`.
2. **Remove/secure open register** — gate `POST /api/v1/auth/register` behind SuperAdmin auth (or delete it; admin creation should be superadmin-only).
3. **Retarget import route** — fix `wallet/import/route.ts` to hit `/wallets/import-encrypted-seed`/`keystore/import` instead of `create`.
4. **Complete auto-sign broadcast** — add sign→broadcast→confirm for BTC/Cosmos; fix Solana to sign a real transaction.
5. **De-stub UserWallet desktop** — hardware/cold-wallet signing + default-validators.
6. **Close UserWallet feature gaps** — push notifications, price alerts, watch-only, multi-account HD, DEX aggregation, token autodetect.
7. **Complete WL admin management endpoints** (MasterWallet/UserWallet/ProjectParty) in `white_label_admin`.
8. **Fix port collision** (`white_label_admin` vs `super_admin` both :8082).
9. **Technically enforce no-resale** (instance/tenant binding in license heartbeat).

---

## Appendix — Security findings detail

### Finding 1 — SQL injection in permission_service audit endpoint

**Location:** `permission_service/go/cmd/main.go:816-831`

```go
clientID := c.Query("client_id")
limit := c.DefaultQuery("limit", "100")

query := `
        SELECT id, admin_id, client_id, action, resource_type, resource_id, details, ip_address, timestamp
        FROM permission_audits
        WHERE 1=1
`
if clientID != "" {
        query += fmt.Sprintf(" AND client_id = '%s'", clientID)   // string interpolation — SQLi
}
query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT %s", limit)  // string interpolation — SQLi
```

**Route:** `router.GET("/api/v1/audit", GetAuditLog)` — no auth gate. **Fix:** parameterize with `$1`/`$2`, parse `limit` to int, add admin-auth middleware.

### Finding 2 — Open admin self-registration (privilege escalation)

**Location:** `super_admin/go/main.go:78` + `:584-600`

```go
auth.POST("/register", handleRegister)   // unauthenticated

func handleRegister(c *gin.Context) {
        // ... binds username/email/password, bcrypt-hashes, then:
        _, err = dbExec(c, `INSERT INTO admin_users (..., role, ...) VALUES ($1,$2,$3,$4,'admin',true,NOW(),NOW())`, ...)
        //                            ^^^^^ creates role='admin' — unauthenticated privilege escalation
}
```

**Fix:** delete the route, or gate it behind SuperAdmin auth so only an authenticated SuperAdmin can provision admins.
