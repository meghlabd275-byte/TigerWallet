# TigerWallet — API Audit (Phase 37)

> Verified 2026-08-29 against working tree (HEAD c2b645c7). Route counts measured
> directly from each service `main.go` via `.(GET|POST|PUT|DELETE|PATCH)(`
> registrations. Auth wiring read from the same files.

## Route inventory (canonical services)

| Service | Port | Routes | Auth model | Status |
|---|---|---|---|---|
| `go/wallet_api` (UserWallet) | 8443 | 123 | `AuthMiddleware(JWT)` on `/api/v1` group; public read-only group (`/api/v1/public/*`, `/api/v1/chains`, `/health`, price/gas/ens) unauthenticated by design; rate-limited signing subgroup (`signLimited`); `/admin` subgroup behind admin check | COMPLETE |
| `master_wallet/backend` (MasterWallet) | 8450 | 93 | `AuthMiddleware(JWT)` on `/api/v1`; wallet-scoped groups `/master-wallet/:id/{treasury,multisig,feature-flags,...}`; treasury/revenue require SuperAdmin two-party co-sign (`license_gate.go`) | COMPLETE |
| `admin/go` | 9093 | 354 | Global `RequestID`, `SecurityHeaders`, `CORS`; `protected.Use(AuthMiddleware)`; `SuperAdminMiddleware()` gates admins/whiteLabels/sysConfig/features; `AdminMiddleware()` gates autoApprovals; `DomainScopeMiddleware(...)` per-domain (liquidity, p2p, futures, ...); structured RBAC handler | COMPLETE |
| `super_admin/go` | 8082 | 285 | `JWTAuth(cfg)` + `IPWhitelistMiddleware(cfg)` on the admin group; `RoleAuth("super_admin")` on walletMgmt + adminAdmins; TOTP 2FA enforced at login (Session 8); no self-registration | COMPLETE |
| `white_label_admin/go` | — | 170 | JWT + TOTP 2FA at login (Session 8); tenant-scoped (`WHERE white_label_id=$1`) | COMPLETE |
| `wl_user_wallet/go` | 8461 | 51 | `JWTAuth` + `middleware.Gate("user_wallet", CategoryFetcher)` (fail-closed license heartbeat) | COMPLETE |
| `wl_master_wallet/go` | 8462 | 77 | JWT + fail-closed `wlgate` license gate | COMPLETE |
| `wl_card/go` | 8459 | 19 | JWT + wlgate; full card lifecycle (apply/activate/block/cancel/limits/topup/rates/stats) | COMPLETE |
| `wl_liquidity/go` | 8458 | 25 | JWT + wlgate; P2P orders/trades/messages/users | COMPLETE |
| `wl_bots/go` | 8463 | 45 | JWT + wlgate | COMPLETE |
| `wl_project_party/go` | — | 81 | JWT + wlgate; real on-chain ERC-20 verification | COMPLETE |
| `project_party/go` | 8106 | — | JWT_SECRET fail-closed at boot (Session 7) | COMPLETE |
| `permission_bridge/go` | — | 15 | `X-API-Key` must map to an enabled product (fail-closed); `SUPER_ADMIN_SECRET` bearer for `/super-admin/*` (403 if unset) | COMPLETE |
| `license_service/go` | 8460 | 39 | license validation + heartbeat + machine-fingerprint binding; records `wl_fingerprint_violations` on drift | COMPLETE |
| `kill_switch` | 8469 | 5 | superadmin-only; platform emergency halt | COMPLETE |

## Cross-domain boundary checks (Phases 10–12, 54)

| Boundary | Evidence | Status |
|---|---|---|
| UserWallet → MasterWallet internals | `go/wallet_api` has **no** import of `master_wallet/*`; admin routes inside wallet_api are behind the `/admin` subgroup + admin check | PASS |
| MasterWallet → UserWallet internals | `master_wallet/backend` keeps its own fetcher copy (`fetchers.go`); no import of `go/wallet_api` | PASS |
| Admin → crypto movement | grep for `private_key|mnemonic|ecdsa|secp256k1` across `admin/go`, `super_admin/go`, `white_label_admin/go` = **0 matches**; withdrawals are approve/reject records only — broadcasting stays with MasterWallet :8450 | PASS (fail-closed) |
| WL tenant → SuperAdmin/TigerWallet internals | WL services expose only tenant routes behind `wlgate`; heartbeat is one-way client→license_service; no SuperAdmin handler registered in any `wl_*` service | PASS |
| Kill switch | registered superadmin-only (`kill_switch`, 5 routes) | PASS |

## Sensitive-endpoint exposure scan

- Unauthenticated groups contain only read-only public data (chains registry,
  prices, gas, ENS lookup, dApp catalog, public balance/nft/tx proxies) — no key
  material, no signing, no admin surface.
- Signing endpoints (`/sign`, `/send`, `/auto-send`, `/keystore/*`,
  `/wallets/:id/export-encrypted-seed`) are all behind `AuthMiddleware`; the
  signing subgroup is additionally rate-limited (`signLimited`).
- Seed export is password-verified + AES-256-GCM encrypted client-side blob;
  plaintext seed never appears in any response.

## Known follow-ups

| Item | Priority | Note |
|---|---|---|
| Per-route request/response schema matrix (123+93+354+285+170 WL routes) | P2 | Route-level table above is authoritative for method/path/auth; field-level schema docs live in `API_DOCUMENTATION.md` — keep in sync when handlers change |
| ~~`idempotency-key` enforcement on payment/billing callbacks~~ | — | **VERIFIED 2026-08-29** — `billing_webhook.go` `PaymentCallback` is HMAC-verified, forward-only `open→paid` (conditional `UPDATE ... WHERE status='open'`), and idempotent on replay (event-id dedup → "already recorded" 200 with no side effects) |
| Rate-limit coverage per route (vs. per signing group) | P2 | Global + signing-group limits present; per-route limits partial |
