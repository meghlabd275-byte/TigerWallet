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
- user_wallet/extension: PRODUCTION-CAPABLE (MV3 + EIP-1193 window.ethereum via src/inpage.js MAIN world + src/contentScript.js bridge + functional src/background.js) as of Round 4. Earlier "no window.ethereum injection, background.js stub" note is STALE.
- admin/web & white_label_admin/web: HAVE real Login pages (LoginPage.tsx / Login.tsx) as of 2026-08-28 audit — earlier "no Login page" note is stale.
- admin/rust has REAL DB-backed auth (bcrypt+lockout, JWT_SECRET fail-closed) — earlier "stubbed auth" note is stale.
- admin/go billing: invoices created "open", marked "paid" only via real Stripe webhook signature verification (billing_webhook.go). Plans may still be hardcoded — verify before relying.
- selfhosted_masterwallet (Rust) is NOW license-gated (license_gate.rs + heartbeat + require_auth on all protected routes) — Session 6 note "unlicensed reference impl" is STALE.
- master_wallet extensions: single source at master_wallet/extensions/extension/ with manifest.<browser>.json variants (chrome/brave/edge/firefox/safari) + build.sh (Round 2). The old "5 byte-identical copies + missing icons" is RESOLVED.
- The 8.2MB ELF binary at `master_wallet/main` has been REMOVED (Round 2). Resolved.

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

## Consolidation round 2 (2026-08-24, commit 8732dbf)
- Canonical fiat ramp: go/fiat_ramp (:8451) — now has real HMAC-verified Stripe/MoonPay/Transak webhooks (webhooks.go); repo-root fiat_onramp/ and fiat_ramp/go deleted (were unbuildable duplicates); fiat_ramp/rust SDK kept.
- Canonical AI: ai_layer (python + rust; rust crate created/fixed, 3 tests) + ai_agent (real eth_gasPrice via EVM_RPC_URL, 11 tests). ai_features/ and ai_platform/ deleted (were 100% stubs / rand() fakes).
- MasterWallet extension: single source at master_wallet/extensions/extension/ with manifests/manifest.<browser>.json variants + build.sh (firefox=MV2+gecko id). The 5 identical per-browser dirs were deleted.
- super_admin + white_label_admin extensions have a browser<->chrome compat shim at top of background.js/popup.js — do not remove.
- Intentionally KEPT as per-app copies (separate deployables per white-label requirement): wl_user_wallet/go (license-gated clone), go/*/id.go (10-line stdlib util per independent Go module), super_admin vs white_label_admin models.go, admin/super_admin/wl_admin extension per-browser dirs (genuinely different manifests).

## Consolidation round 3 (2026-08-24, commit a55e32a)
- permission_service (:8460, docker-compose) = authoritative WL license/permission control plane. permission_bridge = thin tenant-facing edge, now real: pb_products/pb_permissions schema, all 15 handlers DB-backed (pgx), fail-closed auth (X-API-Key must map to enabled product; SUPER_ADMIN_SECRET bearer for super-admin routes). Previously it was unbuildable + all mocks + auth bypass.
- permission_bridge requires SUPER_ADMIN_SECRET env for /super-admin/* routes (fail-closed 403 if unset).
- KEPT (not duplicates): HD crypto 4x = per-language native impls (wallet_core Rust, cpp/wallet_core C++, go/wallet_api Go, wl_shared/wlcrypto Go-WL); admin/rust vs super_admin/rust 19-line sqlx pool wrapper (separate deployables).

## Canonical MasterWallet decision (2026-08-24)
- TigerWallet-operated: master_wallet/backend (Go, :8450) is canonical.
- White-label self-hosted: wl_master_wallet (Go, license-gated heartbeat to SuperAdmin) is the CANONICAL self-hosted MasterWallet — it is the only self-hosted variant with the mandatory SuperAdmin license gate, and it is in docker-compose.
- selfhosted_masterwallet (Rust, actix-web) = alternative unlicensed reference implementation (narrower: no treasury/passkeys/notifications/feature flags/multisig routing, NO license gate). Cargo.lock pinned for rustc 1.85 (actix-web 4.9, url 2.5.0, icu 1.5.0, time 0.3.36). Do NOT ship to WL clients as-is — it bypasses the SuperAdmin control model until a gate is added.

## Round 4 (2026-08-24, commits 6216155, a86b291, 562fb84)
- UserWallet extension is production-capable: MV3 + EIP-1193 window.ethereum injection (src/inpage.js MAIN world, src/contentScript.js bridge, functional src/background.js). Signing delegated to wallet_api /sign + /send (paths are /api/v1/sign, /api/v1/send — NOT under /wallets); read-only RPC goes direct to the chain node from public GET /api/v1/chains registry. No keys in extension.
- master_wallet extension icons (icon16/48/128.png) now exist; build.sh copies them.
- Canonical self-hosted MasterWallet = wl_master_wallet (Go, license-gated). selfhosted_masterwallet (Rust) = unlicensed reference impl; Cargo.lock pinned for rustc 1.85.
- admin/android: single canonical Kotlin app at com.tigerwallet.admin (gradle namespace). Mock Java stubs + mock Compose layer deleted; AndroidManifest + full res tree (23 files) generated. LoginActivity inside MainActivity.kt is a stub (empty setupLoginForm) — needs real wiring if used.

## Round 5 (2026-08-25) — documentation normalization
- New top-level docs (created): `ARCHITECTURE.md` (domain map + ports), `SECURITY.md`
  (key/seed model, auto-sign + co-sign + license gate + kill switch + tenant isolation),
  `ENVIRONMENT.md` (env-var reference, no hardcoded secrets), `docs/GAPS.md`
  (authoritative consolidated gap register — restores the broken `GAPS.md` links in
  5 existing docs), `docs/FEATURE_GAP_REPORT.md` (5-point domain feature/gap matrix).
- Root docs rebranded: removed stale "TigerSwap"/`tigerswap.io` references from
  `INSTALLATION.md`, `API_DOCUMENTATION.md`, `BOT_PLATFORM.md`, `requirements.txt`,
  `tsconfig.json`; corrected chain counts in `README.md` (120 EVM + 66 non-EVM, not
  "200+ / 500–1000+"); fixed stale `.env.example`/`database/schemas/extended_schema.sql`
  refs in `INSTALLATION.md` (no `.env.example` is committed; use `main_schema.sql`).
- VERIFIED gap (P0) **RESOLVED 2026-08-25**: `docker-compose.yml` WL block had
  host-port collisions (8461 wl-user-wallet+wl-admin, 8462 wl-master-wallet+
  wl-liquidity, 8463 wl-bots+wl-card) → `docker compose up` would fail. Fixed
  host-side only: wl-admin :8456, wl-liquidity :8458, wl-card :8459. Container
  ports unchanged. `docker compose config --quiet` passes.
- VERIFIED: `go/full_fetchers/fetchers.go` (1671 lines) registers 18 fetcher types with
  correct structs but no-op `Fetch()` bodies (`// In production, query blockchain`).
  It is scaffold, NOT the canonical live fetch path (that is `go/wallet_api/fetchers.go`).
- VERIFIED: root `package.json` scripts are echo placeholders only (real apps live in
  their own dirs). `master_wallet/web` = 13 pages (not "3/7"); `admin/web` = 37 pages;
  `super_admin/web` = 41 pages; `white_label_admin/web` = 31 pages; `user_wallet/web` = 19.
- Chain registry counts are authoritative at 120 EVM (`chains_evm_data.go`) + 66 non-EVM
  (`chains_nonevm_data.go`). Do not restore the marketing "500–1000+" claims.

## Session 6 (2026-08-28) — master-directive re-audit
- Re-verified P0 fixes (no regressions): permission_service audit SQL parameterized +
  superAdminMiddleware-gated; super_admin has NO self-registration; web_nextjs
  wallet/import no longer proxies create.
- Fixed: services/go/*.go (9 standalone demos) — was a broken package (8 func main in
  one dir + treasury_service.go with none). All now carry `//go:build ignore` tags
  (still runnable via `go run <file>`). staking_service.go: placeholder
  "your-opensea-key" -> OPENSEA_API_KEY env; unused math/big import removed.
- Exact-hash sweep (3317 files): 63 identical groups, ALL Category B/C/D — nothing
  safe to delete (per-browser extension dirs, per-module go/*/id.go, per-module go.sum,
  super_admin vs wl_admin models.go, admin vs super_admin cpp/rust scaffolding,
  bots vs project_party per-app scaffolding, shared Android XML, tooling configs).
  No identical .md/.txt files.
- Go toolchain is NOT present in this sandbox (/tmp/go was session-local); Go compile
  verification requires installing a toolchain first.
- master_wallet/desktop/src/main.cpp remains a 61-line health-probe; real MW clients
  are web (13 pages)/android(14kt)/ios(13 swift)/flutter(20 dart). Full console gap open.

## Session 7 (2026-08-28) — gap implementation round 1
- `go/full_fetchers` (1667 lines, 19 fetchers): previously 100% no-op scaffold
  (`// In production, query blockchain`). Rewritten to REAL EVM JSON-RPC +
  public price APIs via stdlib-only `rpc.go` — build/vet PASS.
- billing payment-callback wired: `admin/go` now exposes
  `POST /api/v1/billing/payment-callback` — HMAC-SHA256-verified webhook
  (`BILLING_PAYMENT_WEBHOOK_SECRET`, fail-closed) — the only path that moves
  an invoice to "paid". admin/go build/vet PASS.
- WL license machine-fingerprint binding (no-resale hardening, Point 5):
  `wl_shared/go/wlgate/fingerprint.go` hashes /etc/machine-id|dmi uuid (no
  secrets); heartbeat sends it via X-Machine-Fingerprint over
  license/validate + /heartbeat; `license_service` binds the fingerprint to
  the instance and records `wl_fingerprint_violations` on drift. wl tests pass.
- Billions-of-addresses sharding (Phase 21 DESIGNED+SCHEMA):
  `database/schemas/user_wallet_sharding.sql` (unchecked additive; uses PG
  hash partitions over chain_id) + `docs/USER_WALLET_SHARDING.md`.
- master_wallet/desktop UI: now all 7 pages (Users, Transactions, Auto-Sign,
  Analytics added); also fixed a pre-existing broken template-literal comment
  that made App.tsx never compile. tsc reports 0 syntax errors.
- Go toolchain re-pinned in-session at /tmp/go/bin (go1.22.5).

## Session 7 (2026-08-28) — build-real gap fixes (UserWallet clients + admin 2FA + WL hardening)
- Go toolchain now verified at /tmp/goroot/go (GOROOT=/tmp/goroot/go; builds pass).
- USERWALLET EXTENSION BUG FIXED: user_wallet/extension/src/popup.js sendTransaction
  + autoSendTransaction POSTed `amount` but wallet_api sendTxReq binds `json:"value"`
  required -> every send returned HTTP 400. Now sends `value` (canonical field).
- USERWALLET DESKTOP (desktop_app/, Tauri) gaps CLOSED:
  - Swap page was a STUB (hardcoded "Rate: 1 ETH = 3500 USDT", no swap-btn listener).
    Now fetches real indicative quote from /api/v1/swap/quote (live CoinGecko) on input
    change + executes via /swap/execute -> /send (on-chain, no fabricated tx hash).
  - loadChains was a hardcoded 7-chain list. Now fetches live /api/v1/chains (fallback
    only when backend unreachable).
  - ADDED import flow (showImportWallet -> POST /wallets with mnemonic, server-side
    BIP-39 re-derive; mnemonic never persisted client-side).
  - ADDED auto-sign toggle on send page (checkbox -> /auto-send when checked).
  - ADDED cloud backup export (exportEncryptedBackup -> POST /wallets/:id/export-
    encrypted-seed, password-verified AES-256-GCM blob downloaded for Google Drive/iCloud).
  - node --check passes on desktop_app/src/app.js.
- ADMIN 2FA ENFORCEMENT (admin/go): login SKIPPED 2FA (empty no-op). Now fail-closed:
  AdminLoginRequest gained TwoFactorCode; Login validates via totp.Validate against
  the TwoFactorAuth record (user_type=admin) when admin.TwoFactorEnabled. Enable2FA
  now syncs two_factor_enabled=true onto the Admin record (was divergent -> bypass).
  Disable2FA: fixed broken sha256-vs-bcrypt password comparison (always failed -> 2FA
  un-disableable); now bcrypt.CompareHashAndPassword + admin pepper; syncs flag off.
  TwoFactorService now holds *config.Config (pepper); wired via NewTwoFactorHandler(db,
  redis, cfg) in main.go. go build + go vet pass (admin/go).
- PROJECT_PARTY JWT HARDENING (project_party/go): jwtSecret() returned a hardcoded
  dev default 'project-party-dev-secret-change-in-production' when JWT_SECRET unset
  (WL clone log.Fatal'd). Now fail-closed log.Fatal. go build + go vet pass.
- WL PRODUCT AUDIT (all 6 wl_*/go): ALL REAL with fail-closed wlgate license gates
  (heartbeat to permission_service/license_service; 503 if not alive, 403 if fetcher
  disabled by SuperAdmin). No auth bypass, no stubs, no forbidden SuperAdmin exposure
  to WL tenants. Independently self-hostable (separate Go modules, no canonical imports).
  wl_user_wallet is fully standalone (own wlcrypto+middleware, no wl-shared dep).
- MASTERWALLET DESKTOP C++ BUILDS: cmake 4.4 (pip cmake) + libcurl4-openssl-dev +
  libssl-dev installed; master_wallet/desktop builds clean (libmasterwallet_core.a +
  master_wallet_desktop binary). main.cpp is a 61-line health-probe but the C++ services
  (1535-line master_wallet_service.cpp, real OpenSSL) are real and compile. Real MW UI
  = React web App.tsx (380 lines, 18 pages not 13).

## Session 8 (2026-08-28) — admin-tier fail-closed 2FA enforcement (commit d117cc2)
- AUDIT FINDING: only admin/go enforced TOTP at login; super_admin/go and
  white_label_admin/go issued a JWT after bcrypt+password with NO TOTP check,
  and admin/web had no 2FA input field. All three tiers had TOTP infra but
  login never read it. SUPER_ADMIN (highest privilege) had the worst variant:
  its handleEnable2FA was a stub that just flipped two_factor_enabled=true
  with NO secret, so even adding login enforcement would have been meaningless.
- super_admin/go (:8082): handleLogin now SELECTs two_factor_secret +
  two_factor_enabled and, when enabled AND secret present, validates via
  totp.Validate (pquerna/otp added to go.mod). Stub handleEnable2FA replaced
  with real TOTP setup (totp.Generate, persist secret, return otpauth URI +
  QR). New handleVerify2FA (validate -> enable flag). New route
  POST /api/v1/admin/2fa/verify. handleDisable2FA clears flag+secret.
  go build + vet pass.
- white_label_admin/go: Login now enforces via the existing real RFC-6238
  verifyTOTP (from-scratch HMAC-SHA1 in totp.go). New Verify2FA handler +
  route POST /auth/2fa/verify closes the enrollment loop. go build + vet pass.
- admin/web: api.ts login() sends two_factor_code; LoginPage renders a
  6-digit 2FA input when backend signals two_factor_required. tsc --noEmit: 0.
- NO LOCKOUT RISK: enforcement triggers only when two_factor_enabled=true AND
  a non-empty secret exists, and secrets are only enabled through the
  verified setup+verify flow (stub-set flags had no secret => not enforced
  until real re-enrollment).
- VERIFIED ISOLATION (admin apps): grep for private_key/mnemonic/ecdsa/
  secp256k1 across admin/go, super_admin/go, white_label_admin/go => 0
  matches. Admin cannot broadcast withdrawals (fail-closed; crypto movement
  stays with MasterWallet :8450). WL admin scoped via TenantScope
  (WHERE white_label_id=$1). Kill switch superadmin-only. Admin cannot create
  super_admin.

## Session 7 — hygiene (route-scope fix + sqlite scrub + theme audit)
- master_wallet/desktop App.tsx: 4 new pages resolve via firstWalletId() to
  canonical /api/v1/master-wallet/:id/... (audit confirmed; no flat endpoints).
- scrub-stale sqlite hashes removed from 4 go.sum files (defi_service,
  blockchain_rpc_service, staking_hub, white_label) via go mod tidy; no
  applications import SQLite. Theme audit found no hardcoded 'light'/'dark'
  and single ThemeContext provider per surface; extension uses its own
  chrome.storage.local 'theme' key (separate surface, acceptable).

## Session 8 (2026-08-28) — Android P0 retarget + desktop mock purge
- ANDROID P0 RETARGET (commit 3e0df1b, pushed): user_wallet/android
  UserWalletApiService.kt pointed at wl_user_wallet :8461 (44 routes) but called
  ~21 endpoint groups that don't exist there -> 404s. Retargeted to canonical
  go/wallet_api :8443 (70+ routes) + aligned every field shape: DEFAULT_BASE_URL
  + healthUrl 8461->8443; WalletConnectSocket localhost:8443->10.0.2.2:8443
  (emulator can't resolve localhost, dApp pairing was dead); getBalance
  /wallets/:id/balance -> /balance?address=&chain_id= (resolve walletId first);
  getTransactions -> /transactions?address=&chain_id=; sendTransaction
  /wallets/:id/send -> /send (flat, wallet_id in body, amount->value, read tx_hash);
  autoSendTransaction -> /auto-send (flat); signMessage -> /sign (flat);
  ammSwap -> /amm/swap with from/chain_id/token_in/token_out/amount_in;
  getAmmQuote -> /amm/quote?token_in=&token_out=&amount_in= (was wrong keys);
  stakingAction asset->token; exportKeystore add export_password; importKeystore
  keystore->keystore_json. /wallets/:id/unlock + /lock remain (canonical).
- BACKEND BALANCE ALIAS (same commit): go/wallet_api BalanceResult now returns
  BOTH `balance` and `balance_wei` (alias) so web+android (read balance_wei)
  work without touching every client. Backward compatible. go build+vet=0.
- DESKTOP MOCK PURGE (commit 5c25a32): desktop_app staking/bridge/NFT/trading
  pages were 100% hardcoded mock UI (Bored Ape #1234@45.5ETH, ETH2.0 APY 4.2%,
  50k fake TOKEN pairs w/ fabricated prices, 505 fake copy-traders) with NO
  data loaders. Replaced ALL with real canonical-backend fetches: index.html
  mock cards -> empty dynamic containers; app.js added loadStakingAssets
  (/staking/quote), stakingAction (/staking/* token binding), loadBridgeRoutes
  (/chains), fetchBridgeQuote+executeBridge (/bridge/*), loadNFTs (/public/nfts),
  nav loaders + button listeners + escapeHtml. tradingFeatures.js: Futures
  loadPairs from /chains+/price + real /perpetual/* positions; Options real
  /price catalog + empty chain (no fabricated greeks); CopyTrading loadTraders
  from /copytrading/traders + follow(); Convert via /swap/quote. bridgeService
  getSupportedRoutes from /bridge/routes. stakingService asset->token. node
  --check passes all 6 desktop JS. No fabricated data remains.
- Toolchain: Go 1.22.5 installed at /tmp/go this session (GOPATH=/tmp/gopath
  GOMODCACHE=/tmp/gomodcache GOFLAGS=-mod=mod). Rust/cmake still NOT installed.
- CANONICAL ROUTE SHAPES (verified, authoritative for all clients):
  /balance?address=&chain_id= -> {chain_id,symbol,address,balance,balance_wei,
  balance_f,usd_value}; /transactions?address=&chain_id= -> {transactions:[
  {hash,to,value,...}]}; /send {wallet_id,password,to,value,chain_id} ->
  {tx_hash,raw_tx,chain_id,nonce}; /auto-send (flat, wallet_id in body);
  /sign {wallet_id,message,password}; /amm/quote?chain_id=&token_in=&token_out=
  &amount_in=; /amm/swap {from,chain_id,token_in,token_out,amount_in};
  /staking/{quote,stake,unstake,claim} binds `token` (NOT asset);
  /keystore/export {wallet_id,password,export_password}; /keystore/import
  {keystore_json,password,label} -> {wallet_id,address,...}; /perpetual/positions
  {+POST, +:id/close}; /copytrading/{traders,follow} (proxied to :8006);
  /bridge/{routes,quote,transfer} (proxied to bridge_service).

## Session 7 (2026-08-28) — wl_liquidity P2P trade surface
- wl_liquidity/go extended from 14 -> 25 routes with the canonical p2p_trading
  surface, all PostgreSQL-persisted (own DB, wl_shared/wlgate unchanged):
  POST/GET /api/v1/orders + GET /orders/:id; POST /trades, GET /trades/:id,
  POST /trades/:id/{confirm,release,dispute}, GET/POST /trades/:id/messages,
  GET /users/:address (uuid-or-email lookup + real trade counters).
- New tables (idempotent CREATE TABLE IF NOT EXISTS in store.Migrate):
  p2p_orders (buyer_id/seller_id nullable FK users, asset, amount/price NUMERIC,
  status open->pending->completed), p2p_trades (order_id FK, buyer/seller FK,
  status open->confirmed->released / disputed), p2p_messages (trade_id FK,
  from_user, body). Trade create runs in a tx that also marks the order pending;
  release marks the parent order completed.
