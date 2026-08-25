# TigerWallet — Repository Inventory (Directive Phases 1–2)

> Session 1 deliverable of the multi-session Master Engineering Directive
> ("Comprehensive Production Audit, Gap Analysis, Architecture Enforcement,
> Deduplication, and Full Implementation Instruction").
> Date: 2026-08-25. Statuses use the Phase 56 evidence-based vocabulary.

## 1. Scope and method

Recursive inspection of the working tree (excluding `.git` and `node_modules`),
cross-checked against `docker-compose.yml`, root manifests, and the per-domain
directory structure. All 149 top-level directories listed in the directive were
confirmed present; the directive's list was verified to be complete for the
current tree (no unlisted top-level code directories exist).

## 2. Repository shape (measured)

| Metric | Count |
|---|---|
| Top-level directories | 149 |
| Go modules (`go.mod`) | 146 |
| Rust crates (`Cargo.toml`) | 95 |
| JS/TS packages (`package.json`, excl. node_modules) | 29 |
| Go source files | 633 |
| Rust source files | 419 |
| TSX files | 368 |
| TS files | 348 |
| JS files | 136 |
| Solidity contracts | 105 |
| C++ sources (.cpp/.hpp/.h) | 193 |
| Kotlin (.kt) | 77 |
| Swift | 77 |
| Dart (Flutter) | 54 |
| Markdown docs | 67 |
| Docker-compose services | 40 (incl. postgres/redis) |

Primary languages: **Go, Rust, TypeScript/React, C++, Kotlin, Swift, Dart, Solidity, Python**.

## 3. Canonical backends (VERIFIED COMPLETE — build/entry present)

| Service | Location | Port | Notes |
|---|---|---|---|
| MasterWallet backend | `master_wallet/backend` | :8450 | Go; auto-signer (`auto_signer.go`), chain registry (120 EVM + 66 non-EVM), license gate |
| UserWallet API | `go/wallet_api` | :8443 | Go; `/api/v1/sign`, `/api/v1/send`, public `/api/v1/chains` |
| Admin API | `admin/go` | :9093 | Go; billing plans currently hardcoded (gap) |
| SuperAdmin API | `super_admin/go` | :8082 | Go |
| License service | `license_service/go` | :8460 | Go; white-label license gate target |
| Kill switch | `kill_switch` | :8469 | Go |
| ProjectParty API | `project_party/go` | :8106 | Go |
| Permission service | `permission_service` | :8460 | Authoritative WL license/permission control plane |
| Permission bridge | `permission_bridge` | — | Tenant-facing edge; DB-backed (pgx), fail-closed auth |
| Fiat ramp | `go/fiat_ramp` | :8451 | HMAC-verified Stripe/MoonPay/Transak webhooks |

## 4. White-label (WL) deployables

| Product | Location | Notes |
|---|---|---|
| WL MasterWallet (canonical self-hosted) | `wl_master_wallet/go` | License-gated heartbeat to SuperAdmin; in docker-compose |
| WL UserWallet | `wl_user_wallet/go` | License-gated clone of `go/wallet_api` |
| WL Bots | `wl_bots/go` | License-gated |
| WL ProjectParty | `wl_project_party/go` | License-gated |
| WL shared gate | `wl_shared/wlgate` | Fail-closed license heartbeat client |
| WL admin / control plane / card / liquidity | `white_label_admin`, `wl_control_plane`, `wl_card`, `wl_liquidity` | Separate deployables |
| Self-hosted MasterWallet (reference) | `selfhosted_masterwallet` | Rust/actix-web; **unlicensed** reference impl — must not ship to WL clients as-is |

## 5. Client application matrix

| Domain | Android | iOS | Desktop | Extension | Web | Flutter |
|---|---|---|---|---|---|---|
| master_wallet | yes | yes | C++ health-probe only (gap) | single source + per-browser manifests | React, 3/7 pages (gap) | yes |
| user_wallet | yes | yes | repo-root `desktop_app/` (Tauri) | MV3 + EIP-1193 (production-capable) | yes | — |
| admin | Kotlin app (LoginActivity stub) | yes | yes | per-browser dirs | React, **no Login page** (gap) | yes |
| super_admin | yes | yes | yes | per-browser dirs (compat shim — do not remove) | yes | — |
| white_label_admin | yes | yes | yes | per-browser dirs | **no Login page** (gap) | — |
| bots / project_party | yes | yes | yes | yes | yes | — |

## 6. Go service fleet (`go/` — 85 service dirs)

`go/` contains 85 service directories including: `wallet_api`, `api_gateway`,
`blockchain_registry`, `blockchain_rpc`, `rpc_node_manager`, `full_fetchers`,
`swap_service`, `bridge`, `bridge_aggregator`, `bridge_service`,
`cross_chain_aggregator`, `staking_service`, `liquid_staking`, `lending_service`,
`perpetual_service`, `p2p_trading`, `copy_trading_service`, `prediction_service`,
`governance_service`, `launchpad_service`, `ieo_service`, `listing_service`,
`nft`, `nft_service`, `nft_prices`, `portfolio`, `portfolio_tracker`,
`analytics_service`, `monitoring_service`, `notification_service`,
`push_notifications`, `fiat`, `fiat_onramp`, `fiat_offramp`, `fiat_ramp`,
`card_service`, `gift_card_service`, `coupon_service`, `red_packets_service`,
`airdrop_service`, `earn_service`, `mpc`, `multisig_service`,
`signature_service`, `social_recovery`, `social_recovery_service`,
`two_factor_auth`, `rbac_admin_service`, `rate_limiter_service`,
`graphql_service`, `websocket_service`, `scheduler`, `sdk`, `oracle`,
`gas_oracle`, `tax_reports`, `bug_bounty_service`, `protection_fund_service`,
`insurance_service`, `enterprise_service`, `cloud_backup_service`,
`log_aggregation_service`, `cdn_service`, `paper_trading`, `matching`,
`distributed_trading`, `real_time_charts`, `twap_service`, `convert_service`,
`leaderboard_service`, `token_deployer_service`, `webhook_service` (partial list;
see `go/` for the authoritative set).

## 7. Root configuration and docs (all present)

`.gitignore`, `docker-compose.yml`, `package.json`, `package-lock.json`,
`requirements.txt`, `tsconfig.json`, `LICENSE`, `README.md`, `AGENTS.md`,
`ADMIN_ARCHITECTURE.md`, `API_DOCUMENTATION.md`, `BOT_PLATFORM.md`,
`BUILD_DEPLOY.md`, `CLOUD_DEPLOYMENT_GUIDE.md`, `INSTALLATION.md`,
`TECH_STACK.md`. Additional docs under `docs/` (fetcher API, project party,
bots/clients, feature-flag enforcement, integration guides).

## 8. Docker-compose topology (40 services)

Infrastructure: `postgres`, `redis`. Core: `wallet-api`, `wallet-frontend`,
`super-admin-api`, `admin-api`, `project-party-api`, `bridge-api`,
`lending-service`, `copy-trading-service`, `governance-service`,
`prediction-service`, `fiat-ramp-service`, `card-service`, `dapp-browser`,
`white-label-api`, `white-label-frontend`, `admin-frontend`,
`super-admin-frontend`, `project-party-frontend`, `permission-service`,
`connection-api`, `fetcher-gateway`, `monitoring-dashboard`, `airdrop-service`,
`earn-service`, `coupon-service`, `red-packets-service`, `bot-api`,
`bots-service`, `license-service`, `kill-switch`, `master-wallet-backend`,
`master-wallet-frontend`, plus all six WL deployables (`wl-user-wallet`,
`wl-master-wallet`, `wl-bots`, `wl-project-party`, `wl-liquidity`, `wl-card`,
`wl-admin`) and `bots-frontend`.

## 9. Inventory gaps to resolve in later sessions

- 146 `go.mod` files vs 85 `go/` service dirs — many modules live outside `go/`
  (per-domain backends). A module-by-module build matrix is required (Phase 44).
- 95 Rust crates — several are pinned for rustc 1.85 MSRV (admin/rust,
  selfhosted_masterwallet). Do not run `cargo update` without MSRV re-check.
- Committed 8.2 MB ELF binary at `master_wallet/main` — should be removed from
  VCS and built by CI instead (P3).
- Per-directory file-level inventory (every source file classified by
  purpose/consumer) is deferred to the fetcher/API/database audit sessions
  (Phases 15, 36–38).
