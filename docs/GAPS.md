# TigerWallet — Resolved & Open Gaps Record

> Consolidated, evidence-based gap matrix. This is the file the other docs link
> as `GAPS.md`. It merges the session-1 `docs/audit/GAP_ANALYSIS.md` findings
> with the session-4 verification of the canonical apps (UserWallet, MasterWallet,
> Admin, SuperAdmin, White-Label, ProjectParty, Bots) and the current working
> tree. Statuses use the Phase-56 vocabulary: **VERIFIED COMPLETE / PARTIAL /
> BROKEN / MISSING / FAKE / SECURITY_RISK / NOT VERIFIABLE**.

---

## 0. Method & evidence

- Recursive inspection of the working tree (excluding `.git` and `node_modules`).
- Route/file counts measured directly (e.g. `admin/go/main.go` → 352 routes,
  `admin/web/src/pages/*.tsx` → 37 pages, `super_admin/web` → 41 pages,
  `white_label_admin/web` → 31 pages, `user_wallet/web` → 19 pages,
  `project_party/web` → 11 pages, `bots/web` → 9 pages).
- Chain counts read from `chains_evm_data.go` (`evmMainnetCount = 120`) and
  `chains_nonevm_data.go` (`nonEVMMainnetCount = 66`).
- "Real vs fake" determined by tracing fetch paths to real HTTP/RPC.

---

## 1. RESOLVED (since session 1 — do not redo)

| Item | Resolution |
|---|---|
| admin/web + white_label_admin/web had no Login page | Real Login pages added + auth gate (`LoginPage.tsx`, `Login.tsx`) |
| admin/android `LoginActivity` stub | Wired to real `POST /auth/login` via `AdminRepository`; `MainActivity` auth redirect added |
| `SessionManager.isTokenExpired()` placeholder | Real ISO-8601 expiry check (fail-closed) |
| `BaseActivity.isNetworkAvailable()` placeholder | Real `ConnectivityManager` check |
| Stubs → PostgreSQL (ProjectParty/Bots/super_admin/WL/import surfaces) | All handlers now pgx/GORM-pool-backed; no in-memory maps |
| Orphan `web3_browser/` | Removed from tree (no consumer; `dapp_browser/` is canonical) |
| `push_notifications/`, `payment_card/`, `bridge/`, `monitoring/` top-level dirs | Already consolidated (canonical: `go/push_notifications`, `go/card_service`, `go/bridge*`, `monitoring_dashboard/` + `observability/`) |
| master_wallet committed ELF (`master_wallet/main`) | Git-ignored; not tracked |

---

## 2. OPEN — P0 (security)

| Gap | Location | Evidence | Status |
|---|---|---|---|
| Unlicensed self-hosted MasterWallet | `selfhosted_masterwallet` (Rust) | No SuperAdmin license gate; reference impl only | SECURITY_RISK — do not ship to WL clients until gated |
| `admin/rust` handler auth completeness | `admin/rust/src/domain.rs` | JWT fail-closed at startup; remaining handler-level auth coverage to confirm | NOT VERIFIABLE (no Rust toolchain in this sandbox) |
| ~~Docker-compose WL host-port collision~~ | `docker-compose.yml` WL block | ~~8461/8462/8463 bound twice~~ | **RESOLVED 2026-08-25** — wl-admin :8456, wl-liquidity :8458, wl-card :8459 (host-side only; container ports unchanged). `docker compose config --quiet` passes |

---

## 3. OPEN — P1 (production blockers)

| Gap | Location | Status |
|---|---|---|
| Billing plans seeded in code, not DB | `admin/go` `billing_handler.go` | PARTIAL — `billing_plans` table exists; initial plans are hardcoded seed rows. Confirm admin CRUD + payment-processor callback for invoice `paid` state |
| MasterWallet desktop is health-probe only | `master_wallet/desktop/src/main.cpp` | BROKEN (as a full client) — probes `/health`, `/chains`; not a full console. Web/Android/iOS/Flutter are the full clients |
| `go/full_fetchers` is scaffold-only | `go/full_fetchers/fetchers.go` (1671 lines) | PARTIAL — 18 fetcher types registered, real data structures, but `Fetch()` bodies are no-op (`// In production, query blockchain`). Canonical live fetch lives in `go/wallet_api/fetchers.go` (real eth_getBalance/eth_call) and per-domain services |

---

## 4. OPEN — P2 (major gaps)

| Gap | Location | Status |
|---|---|---|
| No horizontal-scaling plan for billions of addresses | `database/`, `go/wallet_api` | MISSING — sharding/partitioning design (Phase 21) not documented |
| Fetcher master audit (Phase 36) | fleet-wide | NOT VERIFIABLE IN FULL — enumerated in `docs/audit/REPOSITORY_INVENTORY.md` §6; the 22-field per-fetcher template not yet completed |
| API audit (Phase 37) | `go/wallet_api`, `master_wallet/backend`, `admin/go`, `super_admin/go` | PARTIAL — canonical wallets verified; full route×auth×consumer matrix pending |
| Smart-contract security audit (Phase 42) | `smart_contracts/` (105 .sol) | NOT VERIFIABLE — needs Solidity tooling + auditor |
| `fiat_gateway/go/fiat_gateway.go` is actually Solidity | `fiat_gateway/` | PARTIAL — misplacement; relocated candidate |

---

## 5. UserWallet competitive audit (Phase 14) — verified snapshot

Benchmarked against Trust Wallet / MetaMask / Phantom / Rabby / Coinbase Web3
Wallet / Exodus / KuWallet feature sets.

| Area | Status |
|---|---|
| Wallet create/import/HD (BIP-39/32/44) | VERIFIED COMPLETE (go/wallet_api hd_derive, keystore_v3, crypto_core) |
| Multi-chain (120 EVM + 66 non-EVM) | VERIFIED COMPLETE |
| Extension dApp injection (EIP-1193, MV3) | VERIFIED COMPLETE |
| Signing/send backend (/sign, /send, /non_evm/*) | VERIFIED COMPLETE |
| Send/Receive/Swap/Bridge/Staking/NFT/DeFi/Claim | VERIFIED PARTIAL — full web page set + backend routes present; per-flow end-to-end consumer tracing pending |
| Gasless / paymaster / AA | VERIFIED PARTIAL — `account_abstraction/`, `paymaster_sdk/`, `gasless_tx/` present; flow unverified end-to-end |
| Hardware wallet | NOT VERIFIABLE — `hardware_wallet/` present; needs device test |
| Social recovery / MPC / multisig | VERIFIED PARTIAL — `go/mpc`, `go/multisig_service`, `go/social_recovery*` present |
| Transaction simulation / security warnings | VERIFIED PARTIAL — `transaction_simulator/`, `transaction_shield/` present |
| Biometric/passkeys | VERIFIED PARTIAL — `passkeys_auth/` present; client wiring in user_wallet web/android/ios present but untested on-device |
| Portfolio / NFT / analytics | VERIFIED PARTIAL — `go/nft*`, `go/portfolio*` present |

---

## 6. Platform client matrix — verified

`user_wallet` (Android, iOS, desktop via repo-root `desktop_app/`, MV3 extension,
web 19 pages). `master_wallet` (Android, iOS, C++ desktop health-probe, single-source
extension, web 13 pages, Flutter). `admin` (Android, iOS, desktop, per-browser
extension, web 37 pages, Flutter). `super_admin` (Android, iOS, desktop, extension,
web 41 pages). `white_label_admin` (Android, iOS, desktop, extension, web 31 pages).
`project_party` (web/desktop/android/ios/extension). `bots` (web/desktop/android/ios/extension).

See the domain READMEs and `ARCHITECTURE.md` for per-domain capability detail.

---

## 7. Known intentional simulation (allowed, never represented as live)

- `go/paper_trading` — paper-trading service (Phase 34). Must be clearly labeled
  as paper/simulation in UI/API — verify label.
- Bot backtesting/simulation paths in `mm_bot_platform` — distinct from live execution.

---

## 8. Next queue (priority order)

1. Apply the WL host-port fix to `docker-compose.yml` (P0).
2. Complete `admin/go` billing: move plan seeds to admin CRUD + wire a real
   payment-processor callback for invoice `paid`.
3. Verify/ship `selfhosted_masterwallet` only after adding a license gate.
4. Complete fetcher + API matrices (Phases 36–37).

---

## 9. Verified 2026-08-28 (session 6 — re-run of the master directive)

### Re-verified FIXES from prior sessions (no regression)

- `permission_service` audit SQL injection → FIXED (parameterized `client_id`,
  strict-parsed `LIMIT`; route behind `superAdminMiddleware`).
- `super_admin/go` open admin self-registration → FIXED (only `POST /auth/login`
  + `/auth/refresh`; admin creation is SuperAdmin-only).
- `frontend/web_nextjs` `/wallet/import` route → FIXED (no longer a byte-copy of
  `/wallet/create`; now targets the real import endpoint).

### Fixed in session 6

- `services/go/*.go` (9 standalone demo scripts): the directory was a **broken
  Go package** (8 `func main()` in one dir + 1 file with none). All 9 now carry
  a `//go:build ignore` tag, so each is still runnable via `go run <file>` while
  the directory no longer breaks sweep builds. `services/go/staking_service.go`
  placeholder `"your-opensea-key"` now reads `OPENSEA_API_KEY` from env.

### Duplicate audit (exact-hash sweep over 3317 files, excl. .git/node_modules)

63 identical-content groups found; **all classified Category B/C/D — safe to
delete: none** (Phase 0 safety rule; see `docs/PRODUCTION_AUDIT_2026-08-26.md`
for the full classification table):

- per-browser extension copies (admin / super_admin / white_label_admin) — KEPT
  (genuinely different manifests; browser↔chrome shims).
- `go/*/id.go` (byte-identical 10-line stdlib util per independent Go module) —
  KEPT (each module must be self-contained).
- identical `go.sum` files across `go/*` modules — KEPT (Go requires co-located
  go.sum).
- `super_admin` vs `white_label_admin` `internal/models/models.go` — KEPT
  (separate deployables).
- `admin` vs `super_admin` `cpp/processor.hpp` + CMakeLists, `rust/database/mod.rs`
  — KEPT (separate deployables).
- `bots` vs `project_party` scaffolding (Android res/xml, desktop Electron,
  extension popup/background/index) — KEPT (two independent apps).
- shared Android drawable XML across apps, tooling configs (postcss, tsconfig,
  .dockerignore) — KEPT (standard per-app boilerplate).

No identical `.md`/`.txt` files exist.

### Open gaps (carried over, unchanged)

- `master_wallet/desktop/src/main.cpp` — still a health-probe CLI (61 lines); C++
  services under `desktop/src/services` exist (auth/passkeys/AA/paymaster/privacy/
  super-admin/tax/ws) but the full console is Web/Android/iOS/Flutter/React.
- `go/full_fetchers` — orphan scaffold (19 no-op `Fetch()` bodies, **zero
  importers**). Canonical fetch path: `go/wallet_api/fetchers.go`. Kept as
  documented scaffold; implementing it would duplicate canonical fetchers.
- Billions-of-addresses sharding design — still missing (`database/`).
- Smart-contract security audit (`smart_contracts/` 105 sol) — needs auditor
  tooling; NOT VERIFIABLE here.
- `fiat_gateway/go/fiat_gateway.go` contains Solidity, not Go — relocation
  candidate (P2).