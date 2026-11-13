# TigerWallet — Global Commercial Launch Gap Report

Audit date: 2026-11-13
Commit audited: `8eabd2e` (main)
Method: direct inspection of the working tree, live service runs, real builds
and test execution. Every claim below was verified against the tree; where a
claim comes from a prior session note it was re-checked, and stale notes were
corrected.

## 0. Repository at a glance

| Metric | Count |
| --- | --- |
| Tracked files | 3,967 |
| Go modules (`go.mod`) | 175 |
| Rust crates (`Cargo.toml`) | 107 |
| Dockerfiles | 77 |
| Compose files | 3 (root, `infrastructure/docker`, `deployments/docker`) |
| Go source files | 852 |
| TypeScript/TSX files | 1,112 |
| Kotlin files | 101 |
| Swift files | 103 |
| Dart files | 65 |
| Solidity files | 103 |
| Go test files | 137 |
| Rust test files | 18 |
| Frontend test files | 1 |
| Kotlin test files | 0 |
| Solidity test files | 6 |

---

## 1. What works smoothly (verified)

### 1.1 Core custody and signing backends

- **`go/wallet_api` (:8443)** — the UserWallet backend. 124 routes covering
  auth, wallets, balances, transactions, send/sign/auto-send/simulate, non-EVM,
  chains, swap/AMM, staking, bridge, lending, copy-trading, governance,
  prediction, perpetual/margin, NFT, KYC, ramp, cards, multisig, P2P,
  launchpool, token-sales, approvals, devices, address book, price alerts, fees,
  security, ENS, dApps/WalletConnect, passkeys, keystore. 27 tests pass.
- **`master_wallet/backend` (:8450)** — the custodial MasterWallet. 93 routes in
  the `master-wallet` group. 16 tests pass.
- **`admin/go` (:9093)** — 354 routes, 8 tests pass. TOTP two-factor is enforced
  at login (fail-closed).
- **`super_admin/go` (:8082)** — 285 routes, 7 tests pass.
- **`white_label_admin/go`** — 170 routes, 1 test passes.
- **`license_service/go` (:8460)**, **`kill_switch` (:8469)**,
  **`permission_bridge/go`** — build and test clean.
- **`mm_bot_platform/bot_api` (:8471)** + **`bot_core` (:8472)** — 10 of 18 bot
  types have real execution runners (market_maker, arbitrage, sniper, grid, dca,
  momentum, mean_reversion, scalping, perp_hedge, liquidity_provider). The other
  8 fail closed at start with an explicit error rather than faking execution.
- **`project_party/go` (:8106)** — launchpad with real on-chain
  contribute/claim, order-matching engine, ownership gates.

### 1.2 Security posture (verified)

- **No hardcoded secrets.** A targeted sweep of the four canonical backends for
  high-entropy string literals assigned to `secret` / `password` / `api_key` /
  `private_key` returned nothing outside tests and documented placeholders.
- **JWT secrets fail closed.** `master_wallet/backend/config.go:61` calls
  `log.Fatal` on a missing or placeholder secret; `go/wallet_api/main.go:31` and
  `white_label_admin/go` do the same; `admin/go` and `super_admin/go` use a
  `getRequiredSecret("JWT_SECRET", 32)` helper that refuses an empty or short
  value.
- **Two-factor authentication is enforced at login** for admin, super admin and
  white-label admin, and only when a verified secret exists (no lockout risk).
- **Separation of concerns holds.** No UserWallet client source references the
  MasterWallet (:8450), admin (:9093) or super-admin (:8082) backends. The only
  bridge is the wallet_api service-token proxy to MasterWallet multisig.
- **Kill switch is SuperAdmin-only**; every client exposes a read-only status
  view.
- **Governance contracts are access-controlled** and covered by 53 tests whose
  point is the rejection cases (a non-privileged caller is blocked).

### 1.3 Deployment

- **All 70 build services in the root compose resolve to an existing
  Dockerfile.** No missing build contexts.
- **All three compose files pass `docker compose config --quiet`.**
- **No duplicate host ports remain** in any compose file.
- **CI is fully green** at the audited commit. The `TigerWallet CI/CD Pipeline`
  runs 30 jobs (Code Quality, Security Scan, Integration Tests, and build
  matrices for Go, Rust, C++, and 8 frontends), plus `Launch Readiness` (8 jobs)
  and `MasterWallet CI/CD` (11 jobs). All pass.

### 1.4 Chain and token coverage

- **263 chains seeded** in the MasterWallet registry (198 EVM + 65 non-EVM
  across 43 families); `go/wallet_api` carries 196 EVM + 113 non-EVM entries.
- **UserWallet can build real send transactions for 16 non-EVM families**
  (Solana, Cardano, Algorand, Aptos, Sui, Near, Stellar, Nano, Tezos,
  MultiversX, Tron, Ripple, VET, UTXO/BTC-LTC, Cosmos), all with address-format
  and signing tests. Unknown chains fail closed.
- **TigerSmartChain (chain id 7710)** is consistently configured across Go, C++,
  Rust, TypeScript, Dart, Kotlin, Swift and the generator tooling, and was
  live-verified end to end (balance read, RUSD/WTGR predeploys, token list).

---

## 2. What is missing — gaps by priority

### P0 — Blocks a commercial launch

#### 2.1 `user_wallet/ios` had no project file at all (FIXED this session)

The flagship iOS app shipped **50 Swift sources** with no `.xcodeproj` and no
`Package.swift`. Nothing could build it: not Xcode, not `swift build`, not CI.
It is not in the CI frontend matrix, which is why the gap survived.

**Fixed:** added `user_wallet/ios/Package.swift`. Dependency repos and product
names were verified against the upstream manifests — two of the three
dependencies I first wrote were wrong and are corrected in the committed
version (`gtm-oauth2` ships no `Package.swift`, so `GTMAppAuth` is used; the
Drive SPM product is `GoogleAPIClientForREST_Drive`, not `GTLRDrive`).

**Remaining:** the manifest is committed and syntax-validated, but compiling the
target needs Xcode on macOS. Add the iOS job to CI on a macOS runner.

#### 2.2 Five app surfaces cannot be built (orphaned shells)

| Path | Swift/Kotlin files | Build files | Problem |
| --- | --- | --- | --- |
| `bots/android` | 1 Kotlin | none | Manifest declares `.MainActivity`; **the file does not exist**. Layouts are wallet-themed (`fragment_wallets.xml`, `dialog_add_wallet.xml`) in a bots app. |
| `bots/ios` | 2 Swift | none | No project. |
| `project_party/android` | 1 Kotlin | none | Same pattern as bots. |
| `project_party/ios` | 2 Swift | none | No project. |
| `admin/ios` | 24 Swift | none | No project — 24 sources that cannot build. |
| `super_admin/ios`, `white_label_admin/ios`, `frontend/apps/ios`, `privacy_features/ios` | 1–4 Swift each | none | Placeholder shells. |

These are not "nearly done"; they are unbuildable. For a launch comparable to
Trust Wallet or Atomic Wallet, every advertised platform must produce an
installable artifact. **Decision needed:** finish them, or remove them from the
release surface so the claim matches reality.

#### 2.3 Contracts are written but never deployed or wired

`smart_contracts/evm_contracts` holds 103 Solidity files with 6 test files
across the account-abstraction, DEX and governance suites (the DEX and
governance suites are run through the separate `dex` foundry profile). However:

- **No deployed contract address is configured anywhere** in any compose file or
  environment template. A grep for contract/launchpad/bot addresses across
  `deployments/` returns nothing.
- The TigerBotPlatform and ProjectPartyLaunchpad contracts therefore have **no
  effect on the running system** — the Go services implement their own logic.

**Impact:** the on-chain half of the product is test-complete but
deployment-incomplete. You need a deploy script, a verified-address registry,
and the services reading those addresses.

#### 2.4 Eight of eighteen bot types do not execute

`ai_trading`, `signal`, `cross_chain`, `flash_loan`, `sandwich`, `front_run`,
`mev` and `custom` have no execution runner. They fail closed at start (400 +
list of executable types), which is the correct behaviour, but they are
advertised as bot types. Either build the runners (MEV/sandwich/front-run need
Flashbots-style bundle infrastructure) or stop offering them.

### P1 — Blocks scale or specific markets

#### 2.5 MasterWallet custodial broadcast covers only 4 non-EVM families

This is the single biggest commercial gap. The MasterWallet can sign and
broadcast:

- EVM (all 198 chains)
- Cosmos (all 23 seeded cosmos chains)
- Bitcoin + Litecoin (UTXO via esplora)
- Solana

That is roughly **25 of 65 non-EVM chains**. The remaining ~40 — Tron, Cardano,
XRP, Near, Ton, Sui, Aptos, Polkadot, Algorand, Hedera, Filecoin, Flow, ICP,
Kaspa, Nano, Nervos, VeChain, Waves, Zilliqa, Aleo, MultiversX, Pi and the
fork-id BTC derivatives (BCH, BSV, eCash, Zcash, Groestl, Doge, Dash) — **fail
closed with an explicit error**.

UserWallet is better (16 families) but still short of the MasterWallet's
custodial need. A custodial wallet that cannot sweep or sign on Tron, Cardano or
XRP is not competitive with Trust Wallet. Each family needs a chain-specific
signer.

#### 2.6 Frontend test coverage is effectively zero

**One** frontend test file exists in the entire repository
(`super_admin/web/scripts/urlguard.test.tsx`). Of 33 `package.json` files, 5
declare a `test` script, and the root one is a placeholder that echoes
`'Run from frontend/web_nextjs'`.

**Zero Kotlin tests. One Swift test file.** With ~1,100 TypeScript files, 101
Kotlin files and 103 Swift files, this is the largest verification gap in the
repo. Backend coverage is good (137 Go test files, 18 Rust test files).

#### 2.7 Observability is one config file

There is a single `infrastructure/docker/monitoring/prometheus.yml`. No Grafana
dashboards, no alert rules, no alertmanager config, no tracing, no log
aggregation. For a financial product handling other people's funds this is not
enough. You need: dashboards for each backend, alerts on signing failures /
kill-switch trips / auto-sign backlog / DB and Redis saturation, and distributed
tracing across the wallet → signing → broadcast path.

#### 2.8 Kubernetes covers only two services

Three manifests exist (`k8s/wallet-api.yaml`, `deploy/k8s/masterwallet-backend.yaml`,
`infrastructure/kubernetes/k8s.yaml`) against 77 services. Only wallet-api and
masterwallet-backend have real Deployment/HPA/PDB definitions. The remaining
services are compose-only.

### P2 — Important but not launch-blocking

#### 2.9 Documentation drift

- `README.md:129` maps Notifications to `notifications/` → `notification/` →
  `go/cmd/gateway/`. There is **no `notification/` directory**.
- `notifications/go` contains **two entrypoints**: `cmd/main.go` (the one the
  Dockerfile builds, with FCM support) and `cmd/gateway/main.go` (SMTP/Twilio).
  Only the former is deployed. This duplication should be resolved so there is
  one canonical notification service.

#### 2.10 Infrastructure services had no tests (partially fixed)

`multi_tenancy/go`, `billing_subscription/go` and `notifications/go` had **no
tests** and were **not in the CI matrix** — which is why the bugs in §3 survived.

**Fixed:** 5 integration tests added for multi-tenancy, and all three modules
are now in the CI unit-test matrix, with multi-tenancy also in the PostgreSQL
integration job.

#### 2.11 Non-EVM transaction history needs paid API keys

EVM history uses Etherscan V2 (needs `ETHERSCAN_API_KEY`) with a keyless
Blockscout fallback that only covers chains that run Blockscout. Non-EVM history
is not implemented per chain.

#### 2.12 Mobile wallet connectivity is environment-bound

WalletConnect endpoints were hardcoded to `10.0.2.2` (the Android emulator
alias). This is fixed in this tree — the base URL is now user-configurable and
persisted — but it has never been validated on a physical device.

#### 2.13 Deployment configuration that must be supplied by the operator

- Extension Google Drive backup needs an OAuth `client_id`.
- Push notifications need `google-services.json` (FCM).
- Every service needs its secrets supplied via environment; all are fail-closed.

---

## 3. Bugs found and fixed during this audit

| # | Component | Bug | Impact | Status |
| --- | --- | --- | --- | --- |
| 1 | `multi_tenancy` | `CreateTenant` ran tenant + quota + config inserts as three independent statements | A failure after the tenant insert left a committed tenant with no quota rows; every later quota call failed permanently | **Fixed** — single transaction |
| 2 | `multi_tenancy` | Nothing ever wrote to `tenant_audit_logs` | The audit trail was permanently empty — no record of tenant creation, status change or role change | **Fixed** — `audit()` helper wired to all lifecycle mutations |
| 3 | `multi_tenancy` | Lookups issued SQL with a missing parameter binding | `GetTenantByID` / `GetTenantBySlug` could not resolve the identifier they were given | **Fixed** |
| 4 | `user_wallet/ios` | No project file of any kind | 50 Swift sources could not be built by anything | **Fixed** — `Package.swift` added |
| 5 | `billing_subscription` | `main.go` read only `DATABASE_URL` while compose supplies `BILLING_DB_*` | The service could not reach its database in the shipped compose stack | **Fixed** — `resolveDatabaseURL()` honours both; default port corrected 8100 → 9001 |
| 6 | `notifications` | Database name typo `tigerwallet_notification` | Mismatched the schema created by `database/init.sql` (`tigerwallet_notifications`) | **Fixed** in both entrypoints |
| 7 | `deployments/docker` | `api-gateway` and `ipfs` both published host port 8080 | One of the two could not start | **Fixed** — IPFS moved to 8098 |
| 8 | `notifications` compose | Service pointed at the wrong database | Would have failed at first write | **Fixed** — own DB `tigerwallet_billing` |

Verification: `go build` and `gofmt` clean on every changed module;
`go test ./internal/services/` passes 5/5 against a live PostgreSQL 17;
`docker compose config --quiet` passes on all three compose files; CI green
(30 + 8 + 11 jobs) on the pushed commit `8eabd2e`.

---

## 4. Honest launch-readiness scorecard

| Area | State | Verdict |
| --- | --- | --- |
| Backend logic (Go) | 175 modules, 137 test files, all canonical backends build and test clean | **Strong** |
| Security (secrets, 2FA, RBAC, isolation) | Fail-closed secrets, enforced TOTP, verified tenant isolation | **Strong** |
| Custody / signing | EVM + 4 non-EVM families; ~40 chains fail closed | **Partial** |
| UserWallet clients (web, desktop, extension, Android, iOS, Flutter, Rust) | Feature-complete per route audit; iOS had no build file until today | **Partial** |
| MasterWallet clients | 93/93 route parity on all 7 surfaces | **Strong** |
| Bots / ProjectParty | Backends real; 8 bot types non-executing; 4 app surfaces unbuildable | **Partial** |
| Contracts | Written and tested; never deployed or wired | **Not deployed** |
| Tests (frontend/mobile) | 1 frontend test, 0 Kotlin, 1 Swift | **Weak** |
| Observability | One Prometheus config | **Weak** |
| Orchestration | Compose complete; K8s for 2 of 77 services | **Partial** |
| CI | Fully green, 49 jobs | **Strong** |

---

## 5. Recommended sequence to a commercial launch

1. **Decide the platform surface.** Either build the 5 orphaned mobile shells
   (`bots/android`, `bots/ios`, `project_party/android`, `project_party/ios`,
   `admin/ios`) or remove them. Do not ship a claim you cannot build.
2. **Add the iOS job to CI** on a macOS runner so `user_wallet/ios` and
   `master_wallet/ios` are compiled, not just committed.
3. **Deploy the contracts** and wire the addresses into the services, or
   document clearly that the on-chain layer is not part of the shipped product.
4. **Extend custodial non-EVM signing** — prioritise Tron, Cardano, XRP, Near
   and Ton, which are the chains users actually hold.
5. **Stand up observability** — dashboards, alerts, tracing. This is the
   prerequisite for operating a financial service, not a nice-to-have.
6. **Start frontend and mobile testing** from zero. A single smoke test per
   surface, wired into CI, is the highest-value next step.
7. **Resolve the two notification entrypoints** into one canonical service and
   fix the README mapping.

---

## 6. Corrections to prior session notes

Notes from earlier sessions that the current tree contradicts:

- "Android UserWallet passkeys are a stub" — the tree has a real
  `CredentialManager` implementation.
- "selfhosted_masterwallet is unlicensed" — it has a license gate plus an
  auto-signer daemon. It is still an alternative to the canonical
  `wl_master_wallet`, not a replacement.
- "`master_wallet/desktop` is a health probe only" — it is a real C++ service
  plus a Vite GUI.
- "`go/full_fetchers` is a no-op scaffold" — it performs real EVM JSON-RPC calls.
- "`admin/web` and `white_label_admin/web` have no login page" — both do.
- "`admin/rust` auth is stubbed" — it has real bcrypt + lockout + JWT and
  proxies all protected routes through fail-closed bearer validation.
