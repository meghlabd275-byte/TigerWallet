# TigerWallet — Gap Analysis: Fake/Mock Scan, Feature Gaps, Priorities (Directive Phases 6, 14, 26, 46)

> Session 1 deliverable. Date: 2026-08-25.
> Marker scan (mock/fake/stub/placeholder/dummy/simulat*) across Go/Rust/TS/Py,
> excluding tests and node_modules, combined with previously verified gaps.
> Marker counts are leads, not verdicts — each requires classification per
> Phase 6 ("do not automatically assume every match is broken").

## 1. Marker-scan heat map (files containing ≥1 marker, excl. tests)

| Directory | Files w/ markers | Initial assessment |
|---|---|---|
| `frontend` | 87 | Largest cluster; needs per-file classification (session 2) |
| `go` | 46 | Mixed; `go/paper_trading` legitimately simulates (Category: intentional) |
| `super_admin` | 25 | Frontend demo data suspected |
| `white_label_admin` | 23 | Same |
| `master_wallet` | 22 | Desktop health-probe + partial web pages |
| `admin` | 22 | Rust auth stubs confirmed; billing hardcoded |
| `rust` | 21 | Needs classification |
| `user_wallet` | 15 | Needs classification |
| `core` | 14 | Needs classification |
| `project_party` | 11 | Scaffold-stage code |
| `white_label` | 10 | Needs classification |
| `hardware_wallet`, `bots`, `blockchain_layer` | 8 each | Needs classification |

## 2. Confirmed gaps (VERIFIED — from code inspection history)

### P0 — Critical / security
| Gap | Location | Evidence |
|---|---|---|
| Auth handlers stubbed | `admin/rust` | JWT_SECRET fail-closed at startup, but handler-level auth incomplete |
| Unlicensed self-hosted MasterWallet | `selfhosted_masterwallet` | No SuperAdmin license gate; reference impl only — do not ship to WL clients |

### P1 — Production blockers
| Gap | Location | Evidence |
|---|---|---|
| No Login page (manual localStorage token) | `admin/web`, `white_label_admin/web` | Page inventory — no login route |
| LoginActivity stub (`setupLoginForm` empty) | `admin/android` `MainActivity.kt` | Round-4 verification |
| Billing plans hardcoded | `admin/go` | Plan definitions in code, not DB |

### P2 — Major gaps
| Gap | Location | Evidence |
|---|---|---|
| Desktop app is a health probe only | `master_wallet/desktop` (C++ main) | Round-1 verification |
| Only 3 of 7 web pages implemented | `master_wallet/web/src/pages.tsx` | Single-file page stub set |
| Missing extension icon PNGs | (resolved round 4 for master_wallet) | Verify other domains' extension icons |
| No horizontal scaling design for billions of addresses | `database/`, `go/wallet_api` | Phase 21 — sharding/partitioning plan absent |
| user_wallet has no dedicated desktop dir | uses repo-root `desktop_app/` (Tauri) | Acceptable, but document ownership |

### P3 — Improvements
| Gap | Location |
|---|---|
| Committed 8.2 MB ELF binary | `master_wallet/main` |
| Solidity file in Go path | `fiat_gateway/go/fiat_gateway.go` |
| Orphan directory | `web3_browser/` |
| Shared Electron scaffold for bots/project_party | `bots/desktop`, `project_party/desktop` |

## 3. UserWallet competitive feature audit (Phase 14) — status snapshot

Benchmarked against Trust Wallet / MetaMask / Phantom / Rabby feature sets.

| Area | Status | Notes |
|---|---|---|
| Wallet creation/import/HD | VERIFIED COMPLETE | HD impls in 4 languages (intentional) |
| Multi-chain (EVM + non-EVM) | VERIFIED COMPLETE | 120 EVM + 66 non-EVM in chain registry |
| Extension dApp injection (EIP-1193) | VERIFIED COMPLETE | MV3, inpage MAIN world, no keys in extension |
| Signing/send backend | VERIFIED COMPLETE | `/api/v1/sign`, `/api/v1/send` |
| Swaps / bridges / staking / lending / P2P | VERIFIED PARTIAL | Services exist in `go/`; end-to-end consumer tracing pending (Phase 15, session 2) |
| NFT, portfolio, analytics | VERIFIED PARTIAL | `go/nft*`, `go/portfolio*` present; fetcher wiring unverified |
| Gasless / paymaster / AA | VERIFIED PARTIAL | `account_abstraction/`, `paymaster_sdk/`, `gasless_tx/` present; flow unverified |
| Hardware wallet | NOT VERIFIABLE | `hardware_wallet/` contains markers; needs device-integration test |
| Social recovery / MPC / multisig | VERIFIED PARTIAL | `go/mpc`, `go/multisig_service`, `go/social_recovery*` present |
| Transaction simulation / security warnings | VERIFIED PARTIAL | `transaction_simulator/`, `transaction_shield/` present |
| Biometric/passkeys | VERIFIED PARTIAL | `passkeys_auth/` present; client wiring unverified |

## 4. MasterWallet gap analysis (Phase 26) — status snapshot

| Capability | Status |
|---|---|
| Chain registry management (EVM + non-EVM) | VERIFIED COMPLETE (seeded data + backend) |
| Auto-signer with policy | VERIFIED PARTIAL — policy matrix documentation pending (Phase 23) |
| Revenue ops with SuperAdmin co-sign | VERIFIED COMPLETE |
| Fee configuration (Phase 24) | VERIFIED PARTIAL — per-fee audit trail/rollback verification pending |
| Token/coin management (Phase 20) | VERIFIED PARTIAL — `token_management/`, `listing_service` exist; admin UI flow unverified |
| Desktop client | VERIFIED BROKEN (health probe only) |
| Web client | VERIFIED PARTIAL (3/7 pages) |

## 5. Fake-implementation policy enforcement (Phase 6)

Confirmed real (not fake) integrations:
- `go/fiat_ramp` — HMAC-verified Stripe/MoonPay/Transak webhooks.
- `ai_agent` — real `eth_gasPrice` via `EVM_RPC_URL` (11 tests).
- `permission_bridge` — DB-backed handlers, fail-closed auth.

Known intentional simulation (allowed, must never be represented as live):
- `go/paper_trading` — paper trading service (Phase 34 requires clear
  distinction from live execution; verify API/UI labeling in session 2).

## 6. Next-session queue (priority order)

1. Classify the 87 `frontend/` + 46 `go/` marker files (Phase 6) → produce
   FAKE_IMPLEMENTATION_REGISTER.md.
2. Fetcher master audit (Phase 36): enumerate `go/full_fetchers` + per-domain
   fetchers with the 22-field template.
3. API audit (Phase 37) starting with `go/wallet_api`, `master_wallet/backend`,
   `admin/go`, `super_admin/go` route tables.
4. Fix P1: Login pages for admin/web + white_label_admin/web; wire
   admin/android LoginActivity.
5. Fix P0: complete `admin/rust` handler auth; document/keep
   `selfhosted_masterwallet` non-shippable status.
