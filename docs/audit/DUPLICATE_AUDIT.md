# TigerWallet — Duplicate Code & File Audit (Directive Phase 5)

> Session 1 deliverable. Date: 2026-08-25.
> Method: content-hash (md5) scan of all source files >500 bytes, plus the
> previously verified consolidation history (rounds 1–4, see `docs/GAPS.md`
> and `AGENTS.md`). Classification per Phase 5 categories A–E.
> **No files were deleted in this session** — Phase 0 safety rules require
> dependency analysis before any consolidation.

## 1. Exact-duplicate groups found this session (hash-verified)

| # | Files | Category | Decision |
|---|---|---|---|
| 1 | `super_admin/extensions/{chrome,firefox,safari}/background.js` | C — platform-specific per-browser packaging | KEEP (manifests genuinely differ per browser) |
| 2 | `admin/extensions/{chrome,firefox,safari}/js/popup.js` | C | KEEP |
| 3 | `admin/extensions/{chrome,firefox,safari}/content.js` | C | KEEP |
| 4 | `super_admin/go/internal/models/models.go` = `white_label_admin/go/internal/models/models.go` | D — separate deployables (SuperAdmin vs WL-Admin) | KEEP (documented in round 2) |
| 5 | `bots/desktop/src/main/main.js` = `project_party/desktop/src/main/main.js` | D — separate products' desktop shells | KEEP for now; flag for shared Electron scaffold extraction (P3) |
| 6 | `admin/extensions/{chrome,firefox,safari}/background.js` | C | KEEP |
| 7 | `admin/extensions/{chrome,firefox,safari}/js/api.js` | C | KEEP |
| 8 | `white_label_admin/extensions/{chrome,firefox}/popup.js` (+safari) | C | KEEP |
| 9 | `bots/extension/src/popup.js` = `project_party/extension/src/popup.js` | D | KEEP; same scaffold note as #5 |
| 10 | `super_admin/extensions/{chrome,firefox,safari}/popup.js` | C | KEEP |
| 11 | `admin/rust/src/database/mod.rs` = `super_admin/rust/src/database/mod.rs` | D — 19-line sqlx pool wrapper, separate deployables | KEEP (documented round 3) |

**Result: 0 Category-A (safe-to-delete) duplicates confirmed this session.**
All hash-identical groups are per-browser packaging or per-deployable copies,
which Phase 5 explicitly says to keep.

## 2. Previously consolidated (rounds 1–4 — verified history, do not redo)

| Item | Resolution |
|---|---|
| `fiat_onramp/` vs `fiat_ramp/` | Canonical = `go/fiat_ramp` (:8451) with real HMAC webhooks; repo-root duplicates deleted |
| AI price prediction ×4 | Canonical = `ai_layer` + `ai_agent`; `ai_features/`, `ai_platform/` deleted |
| master_wallet extensions ×5 byte-identical | Single source `master_wallet/extensions/extension/` + `manifests/manifest.<browser>.json` + `build.sh` |
| push_notifications ⊂ notifications | Kept as distinct services (`go/push_notifications`, `go/notification_service`) — different consumers |
| `payment_card` vs `crypto_card` | Both retained; canonical card service = `go/card_service` |
| `bridge` vs `cross_chain_aggregator` | Both retained as distinct services in `go/` |
| monitoring/monitoring_dashboard/observability | Distinct: backend service, Grafana dashboard, instrumentation lib |
| 11 identical `go/*/id.go` | KEEP — 10-line stdlib util per independent Go module |
| `fiat_gateway/go/fiat_gateway.go` (actually Solidity) | Known misplacement; relocation candidate (P3) |

## 3. Intentionally kept look-alikes (Category B/D/E — do NOT merge)

| Pair | Reason |
|---|---|
| `wl_user_wallet/go` ≅ `go/wallet_api` | Per-tenant WL deployable with license gate |
| `user_wallet/rust` ≅ `rust/userwallet_fetchers` | Per-language native impl |
| HD crypto ×4 (`wallet_core` Rust, `cpp/wallet_core`, `go/wallet_api` Go, `wl_shared/wlcrypto`) | Per-language native implementations |
| `web3_browser` ≅ `dapp_browser` | `web3_browser` is an orphan dir — consolidation candidate pending consumer analysis (session 2+) |
| `bots/` ≡ `project_party/` scaffolding | Separate products sharing scaffold; extract shared scaffold only if a real third consumer appears |

## 4. Near-duplicate candidates queued for dependency analysis (Phase 4 prerequisite)

These need full import/consumer tracing before any action (Phase 0 rule):

1. `web3_browser/` (orphan) vs `dapp_browser/` — verify zero imports, then remove or fold in.
2. `bots/desktop` vs `project_party/desktop` Electron shells — extract shared scaffold.
3. `go/bridge`, `go/bridge_service`, `go/bridge_aggregator`, `go/cross_chain_aggregator` — four bridge-adjacent services; map routes/consumers to confirm distinct roles.
4. `monitoring/`, `monitoring_dashboard/`, `observability/` — confirm dashboard vs service vs library split.
5. `fiat_gateway/go/fiat_gateway.go` — Solidity file in a Go path; relocate to `smart_contracts/`.

## 5. Non-source duplication

- `master_wallet/main` — committed 8.2 MB ELF binary. Not a code duplicate but a
  VCS hygiene issue; remove and build via CI (P3).
- 146 `go.mod` / 111 `go.sum` files — expected for multi-module repo; verify
  each module still builds (Phase 44, session 2+).
