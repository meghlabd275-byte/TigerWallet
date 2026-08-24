# TigerWallet Repository Knowledge

## Architecture (verified 2026-08-24)
- Canonical backends: `master_wallet/backend` (Go, :8450), `go/wallet_api` (UserWallet, :8443),
  `admin/go` (:9093), `super_admin/go` (:8082), `license_service/go` (:8460), `kill_switch` (:8469),
  `project_party/go` (:8106).
- All client apps (android/ios/desktop/extensions/web/flutter) are thin REST/WS clients over these backends.
- White-label clones: `wl_master_wallet`, `wl_user_wallet`, `wl_bots`, `wl_project_party` (Go + `wl_shared/wlgate` license gate, fail-closed heartbeat to license_service).
- Chain registry: 120 EVM + 66 non-EVM seeded chains (`master_wallet/backend/chain_registry_data.go`).
- MasterWallet auto-signer: `master_wallet/backend/auto_signer.go`; revenue/treasury ops require SuperAdmin two-party co-sign (`license_gate.go`).

## Known gaps (verified)
- master_wallet/desktop: C++ main is a health-probe only; React UI has 3 of 7 pages.
- user_wallet has NO desktop dir (desktop app is repo-root `desktop_app/` Tauri).
- user_wallet/extension: localhost-only host permission, no window.ethereum injection, background.js stub.
- admin/web & white_label_admin/web: no Login page (manual localStorage token).
- admin/rust handlers stubbed auth; admin/go billing plans hardcoded.
- master_wallet extensions: 5 byte-identical copies; missing icon PNGs.
- Committed 8.2MB ELF binary at `master_wallet/main`.

## Confirmed duplicates
- fiat_onramp vs fiat_ramp (~480 shared lines); push_notifications ⊂ notifications;
  payment_card vs crypto_card (317 shared lines); bridge vs cross_chain_aggregator (253 shared);
  monitoring/monitoring_dashboard/observability; AI price prediction ×4 (ai_agent/ai_features/ai_layer/ai_platform);
  wl_user_wallet/go ≅ go/wallet_api; user_wallet/rust ≅ rust/userwallet_fetchers;
  wallet_core (Rust) ≅ cpp/wallet_core ≅ wl_shared/wlcrypto; web3_browser orphan dir ≅ dapp_browser;
  bots/ ≡ project_party/ scaffolding; 11 identical go/*/id.go; fiat_gateway/go/fiat_gateway.go is actually Solidity.
- docs/GAPS.md documents 3 consolidation rounds (PR #11); item 11 "no duplicates" only covered scanned dirs.

## Build environment notes (2026-08-24)
- Go toolchain lives at /tmp/go/bin (use GOPATH=/tmp/gopath GOMODCACHE=/tmp/gomodcache GOFLAGS=-mod=mod).
- Rust: system cargo/rustc 1.85 (apt). admin/rust Cargo.lock is pinned to MSRV-1.85-compatible versions (time 0.3.36, simple_asn1 0.6.2); do not run cargo update without re-checking MSRV. libssl-dev + pkg-config required.
- admin/rust requires JWT_SECRET env at startup (fail-closed, no default).
- admin/go billing: invoices are created in "open" status and only a real payment-processor callback should mark them paid.
- web_nextjs API proxies: canonical copy-trading backend routes are /api/v1/copytrading/{copiers,follow,stop-all,copiers/:id/stop}; staking backend group is /api/v1/staking (proxy paths must not double-prefix).
