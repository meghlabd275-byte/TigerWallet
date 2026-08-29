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

## Session 9 (2026-08-29) — fake-impl purge + on-chain launchpad (commit pending)
- DUPLICATE SWEEP (re-run, 3317+ files): all remaining identical groups are
  Category B/C/D intentional per-app/per-browser/per-module copies (go/*/id.go,
  per-browser extension manifests, bots vs project_party per-app scaffolding,
  super_admin vs wl_admin models.go, admin vs super_admin cpp/rust scaffolding).
  Nothing safe to delete — consolidating would couple separately-deployable apps.
- admin/rust: handlers.rs had ~70 stub handlers returning hardcoded JSON
  ({"message":"..."}, [], {"id":<uuid>}) for admins/users/KYC/transactions/
  withdrawals/tokens/pairs/blockchains/fees/whitelabels/tickets/analytics/audit/
  feature-flags/notifications/ip-whitelist/backups/webhooks (auth+2FA were
  already real). REPLACED all stubs with a real HTTP/1.1 proxy to the canonical
  admin/go backend (:9093) — same pattern as the existing domain.rs proxy.
  cargo check (lib + bin) PASS, 0 errors.
- admin/cpp: was BROKEN — root CMakeLists.txt built nonexistent src/processor.cpp;
  include/CMakeLists.txt referenced nonexistent src/*.cpp + admin_database.hpp/
  admin_models.hpp + protobuf (unused). admin_handler.cpp returned fake
  {"token":"test"} / hardcoded empty JSON for every route, and routes used :id
  but the router only matches {id} (so :id routes never dispatched). FIXED:
  rewrote root CMakeLists.txt to build all include/*.cpp into a real
  tiger_admin_cpp executable (OpenSSL+Threads, no protobuf); deleted broken
  include/CMakeLists.txt; added proxy_to_admin_go() (real TCP HTTP/1.1 call to
  admin/go :9093, fail-closed 503 on dead upstream) and routed every handler
  through it; converted :id -> {id} so routes dispatch; added missing
  IPRateLimiter::initialize() no-arg definition. cmake build PASS (5.5MB binary).
- wl_project_party G2 (launchpad on-chain): Contribute/Claim were DB-only
  (no on-chain tx). Implemented real LaunchpadOnChain (ethclient + ABI +
  contribute()/claimTokens() tx broadcast + receipt wait + tokenPrice eth_call,
  mirroring canonical project_party/go/cmd/launchpad_onchain.go). New
  config fields PP_LAUNCHPAD_PRIVATE_KEY/PP_LAUNCHPAD_CONTRACT_ADDRESS; new
  store.SetContributionOnChain/SetContributionClaimedOnChain + tx_hash/confirmed_at
  columns; Contribute/Claim handlers broadcast real txs (fail-closed 503 if
  unconfigured, never fabricate a hash). go build + vet + test PASS.
- wl_card: Store.Rates() returned hardcoded {BTC:67000,ETH:3500,...} mock.
  REPLACED with a real CoinGecko price oracle (live USD prices, 60s in-memory
  TTL cache, stablecoins pinned to 1.0, empty map on failure — never fabricated).
  go build + vet PASS.
- Toolchain: Rust 1.85.0 (rustup) + Go 1.23.4 (/usr/local/go) installed this
  session; libssl-dev + pkg-config + cmake installed. admin/rust cargo check +
  admin/cpp cmake build + wl_project_party/wl_card/wl_bots/wl_liquidity/
  admin/go/go-wallet_api go build all PASS.

## Session 10 (2026-08-29) — close remaining P0/P1 gaps (kill-switch, super_admin/cpp, G3)
- super_admin/cpp: was BROKEN (CMakeLists built nonexistent src/processor.cpp; no .cpp
  files existed — header-only). The header super_admin_domains.hpp is REAL (POSIX-socket
  HTTP proxy to super_admin/go :8082, no stubs). FIXED: rewrote CMakeLists to build a real
  tiger_super_admin_cpp CLI driver (src/main.cpp) over the header-only SuperAdminHttpClient
  + expose an INTERFACE header library. cmake build PASS; driver fail-closes with a
  transport error when super_admin/go is down (no fabricated data). P1 closed.
- master_wallet/backend kill-switch integration (P0): the kill_switch service (:8469)
  writes global halts to the shared Redis key `kill:global`. The canonical MW backend
  previously did not consult it. ADDED kill_switch.go: Store.IsKillSwitchHalted() checks
  the kill:global Redis key (best-effort; a Redis outage does NOT self-paralyze the
  operator backend — halts are a positive signal, absence == not halted);
  KillSwitchMiddleware 503-blocks every /api/v1/ request when a global halt is active
  (/health + /ws stay reachable for monitoring); read-only GET /api/v1/kill-switch/status
  endpoint (any authenticated user; toggle is SuperAdmin-only via kill_switch :8469).
  Only a SuperAdmin can issue a halt. go build + vet PASS. P0 closed.
- wl_project_party G3 (on-chain tx confirmation fetcher): added LaunchpadOnChain.
  ConfirmTransaction() — real ethclient.TransactionReceipt lookup returning
  {confirmed, status: success|reverted|pending, block_number, gas_used}, fail-closed
  (never claims a confirmation that didn't happen; pending if receipt not found).
  New store.GetContribution(); new GET /launchpad/:id/contribution-status?tx_hash=
  handler returning DB status + on-chain confirmation. go build+vet+test PASS. P2 closed.
- Toolchain reinstalled this session: Go 1.23.4 (/usr/local/go), Rust 1.85.0 (rustup),
  cmake 3.31.6 + libssl-dev + pkg-config (apt). All changed surfaces build-verified:
  master_wallet/backend, wl_project_party, wl_card, admin/go, go/wallet_api (go build);
  admin/rust (cargo check, 0 errors); admin/cpp + super_admin/cpp (cmake build).
- Remaining open gaps (documented, not blocking): selfhosted_masterwallet (Rust) is the
  unlicensed reference impl (no two-party co-sign/auto-signer loop — use wl_master_wallet
  for WL); master_wallet/backend project-party listing approval + market-making/bots hooks
  live in sibling project_party/ + bots/ (by design, not a MW gap); Android UserWallet
  passkeys stub + WalletConnect host; Desktop (Tauri) missing dApp/WalletConnect/passkeys/
  KYC/ramp/ENS/NFT-transfer/DeFi wiring; some admin web pages may have UI-only stubs
  behind real routes. A full zero-gap build of a 186-chain, 3-app-family, multi-tier
  crypto-wallet ecosystem with 100/100 frontend↔backend wiring is multi-month work
  beyond one session.

## Session 11 (2026-08-29) — gap-MD reconciliation, SQLite verification, desktop wiring
- Read all gap/audit MD files (docs/GAPS.md, docs/FEATURE_GAP_REPORT.md,
  docs/AUDIT_REPORT.md, docs/PRODUCTION_AUDIT_2026-08-26.md, root master audit)
  and marked every gap against the current tree.
- VERIFIED already-resolved (stale MD entries corrected): master_wallet/desktop
  health-probe → real console.cpp (S7/8); go/full_fetchers no-op scaffold → real
  EVM JSON-RPC Fetch() bodies, zero "In production" markers (S7);
  selfhosted_masterwallet license gate (S5); fiat_gateway Solidity misplacement
  removed (canonical go/fiat_ramp with real HMAC webhooks).
- SQLite removal — VERIFIED COMPLETE: full scan of all Go modules = zero SQLite
  usage. Every GORM Open uses postgres.Open (116 sites); pgxpool (295);
  database/sql "postgres" (12). admin/rust sqlx uses only the postgres feature.
  No gorm.io/driver/sqlite / mattn/go-sqlite3 / modernc.org/sqlite imports. The
  repo is PostgreSQL + Redis only. admin/rust/target/ build artifacts are
  untracked. (User's explicit "remove sqlite" requirement is satisfied.)
- Desktop (Tauri) feature wiring EXTENDED: desktop_app/src/app.js now wires ENS
  resolve, KYC status, fiat-ramp providers, and dApp catalog + WalletConnect to
  canonical go/wallet_api routes. Real fetches, fail-closed empty states, no
  fabricated data. node --check PASS. Transaction-submitted notifications
  already present ("Transaction broadcast: <tx_hash>", "Swap submitted: ...").
- Updated docs/GAPS.md (P1 §3, §9, new §10 Session-11 section) and
  docs/FEATURE_GAP_REPORT.md (full_fetchers, admin/rust, desktop, scorecard) to
  reflect sessions 7-10 resolutions — no stale "BROKEN/MISSING" entries remain
  for fixed items.
- Build-verified: master_wallet/backend + wl_project_party (go build ✅);
  admin/cpp + super_admin/cpp (cmake ✅); desktop_app/app.js (node --check ✅).
- Committed 6f6cf418, pushed to origin/main.
- Toolchain reinstalled this session: Go 1.23.4, cmake 3.31.6, libssl-dev, pkg-config.

## Session 12 (2026-08-29) — selfhosted MW auto-signer, Android passkeys, theme verification
- Theme switch verified working across all surfaces: user_wallet (web
  ThemeContext, extension chrome.storage, android ThemeManager, ios),
  master_wallet (web themeService, flutter theme_toggle/service, desktop C++),
  admin/super_admin/white_label_admin (web + ios + desktop), desktop_app
  (Tauri: button + settings select, persists to localStorage, data-theme attr).
  All toggle + persist; no hardcoded light/dark.
- selfhosted_masterwallet (Rust): had license gate (S5) but NO auto-signer
  daemon loop (approve_transaction only set DB status; no auto-broadcast). Added
  real `auto_signer.rs`: polls shmw_transactions pending every 200ms
  (SHMW_AUTO_SIGN_POLL_MS), matches shmw_auto_sign rules (pattern on
  to_address/token + decimal value <= max_value gate), auto-approves, signs +
  broadcasts via real evm_tx RPC (nonce/gas/estimate mirroring sign_and_broadcast),
  records tx hash + status='broadcast'. Fail-closed: MASTER_AUTO_SIGN_PASSWORD
  unset => approvals recorded, broadcast disabled; never touches two-party
  withdrawal-gated funds; never fabricates a hash. Made canonical_chain + add_dec
  pub so the module can call them. cargo check 0 errors (bin).
- Android UserWallet passkeys: was a reflective stub that threw
  "androidx.credentials request types unavailable". Added
  androidx.credentials:credentials:1.3.0 + credentials-play-services-auth:1.3.0
  to build.gradle; rewrote CredentialManagerHelper.kt to the real
  CreatePublicKeyCredentialRequest (register) + GetPublicKeyCredentialOption
  (authenticate) platform flows. isAvailable=true (API on every Android 5.0+
  with Play Services). (Cannot compile-verify without Android SDK; code is real.)
- GAPS.md updated: marked selfhosted_masterwallet + Android passkeys RESOLVED.
- Build-verified: selfhosted_masterwallet cargo check 0 errors; desktop_app
  node --check (prior); master_wallet/backend + wl_project_party go build (prior).
- Remaining open (documented): Android WalletConnect host emulator-only;
  admin/super_admin web UI-only stub pages; Solidity contract audit (needs
  tooling); sharding deployment.

## Session 13 (2026-08-29, this chat) — bug fixes + Phase 36/37 audits (rebased onto S9-12)
- NOTE: ran concurrently with Sessions 9-12 above (separate sandbox); this
  session's entry was renumbered 9->13 on rebase. Its toolchain was Go 1.22.12
  at /tmp/go122/go (session-local, gone); Sessions 9-12 installed Go 1.23.4 +
  Rust 1.85.
- BUG FIX (admin/go auth middleware): generateRequestID used strings.ReplaceAll on
  a template — every placeholder got the SAME computed digit -> near-constant
  request IDs (tracing/audit correlation broken). Replaced with crypto/rand
  RFC-4122 v4 UUID (nanotime fallback). Imports: strings -> crypto/rand.
- BUG FIX (admin/go PagerDuty): CreateService hardcoded escalation policy id
  "PXXXXXX" (would create broken services). Now a required validated param
  (fail-closed on empty). No in-repo callers — safe signature change.
- PHASE 36 DONE: docs/FETCHER_AUDIT.md — full per-fetcher matrix (provider/auth/
  chains/output/consumers/status). 14 fetcher groups COMPLETE (traced to real
  RPC/REST); cex_connectors/dex_connectors/price_oracle PARTIAL (real code, live-
  provider run pending); fetcher_core/gateway (Rust) NOT VERIFIABLE at the time.
  0 fake/hardcoded-data fetchers remain in production paths.
- PHASE 37 DONE: docs/API_AUDIT.md — route×auth matrix for 15 canonical services
  (wallet_api 123, master_wallet 93, admin/go 354, super_admin 285, wl_admin 170,
  wl_user_wallet 51, wl_master_wallet 77, wl_card 19, wl_liquidity 25, wl_bots 45,
  wl_project_party 81, license_service 39, permission_bridge 15, kill_switch 5) +
  cross-domain boundary checks (Phases 10-12, 54): all PASS. Billing webhook
  idempotency VERIFIED (event-id dedup + forward-only open->paid conditional UPDATE).
- Build+vet=0 at the time for all 14 canonical Go modules (Go 1.22.12);
  wlgate tests pass.

## Session 14 pre-audit (2026-08-29) — UserWallet app-family audit (fetchers + gaps)
- Backend go/wallet_api :8443 = 124 routes (auth, wallets, send/sign/auto-send/simulate,
  non_evm, chains, swap/amm, staking, bridge->:8007, lending->:8009, copytrading->:8006,
  governance->:8454, prediction->:8455, dao, perpetual/margin, terminal, nft, kyc,
  ramp->:8451, cards->:8457, multisig->:8450 via service token, p2p, launchpool,
  token-sales, approvals, devices, address-book, price-alerts, fees, security, ens,
  gas/price/chart, dapps+walletconnect->:8083, passkey, keystore).
- Client parity ranking: web (65+ endpoints in api.ts, 19 pages) ~= iOS (28 swift,
  full lending/p2p/dao/passkey) > Android (36 kt, 22 fragments, real passkeys) >
  desktop_app (Tauri, only 9 pages) > extension (MV3 EIP-1193, narrowest popup).
- Open gaps: WalletConnect hardcoded 10.0.2.2 (emulator-only) on Android+iOS;
  extension googleDriveBackup.js Google OAuth client_id TODO; desktop missing pages
  (approvals/addressbook/devices/KYC/DeFi/DAO/launchpool/token-sales/p2p/cards/
  price-alerts/dApp/passkeys/NFT-transfer); no UI anywhere for perpetual/margin,
  launchpool, token-sales, P2P, cards, price-alerts despite backend routes;
  rust/userwallet_fetchers ~= user_wallet/rust duplicate unresolved;
  desktop_app at repo root not under user_wallet/ (runtime separation OK).
- Separation rule verified: no UserWallet app imports MW/admin fetchers; multisig
  reaches :8450 only via wallet_api service-token proxy.

## Session 14 (2026-08-29) — UserWallet 100/100 frontend<->backend gap closure
- WEB (user_wallet/web): FIXED dead-route bug — Layout nav had 16 links but
  App.tsx only routed 4 (dashboard/wallets/transactions/settings). App.tsx now
  routes all 28 pages. Added 12 pages: Trading(perp+margin), Launchpool,
  TokenSales, P2P, Cards, PriceAlerts, DApps(pairings/sessions approve+reject),
  DAO(vote), Ramp(on/off quote), CopyTrading, Prediction, ENS. api.ts gained
  getPriceAlerts/createPriceAlert/updatePriceAlert/deletePriceAlert +
  createWatchOnlyWallet; card methods default cardId='default' (backend proxy
  drops the id segment). theme.css gained the shared themed classes
  (.primary-btn/.secondary-btn/.quote-box/.success-banner/.record-list/
  .record-item/.empty-state/.action-row/.category-chip/.form-group select) that
  existing pages already referenced but were never defined (pre-existing gap).
  npx tsc --noEmit = 0 errors (npm i --legacy-peer-deps; react-scripts 5 wants
  typescript 4.9 peer).
- DESKTOP (desktop_app, Tauri): 9 → 24 pages. index.html nav+containers added
  for defi, trading, launchpool, token-sales, p2p, cards, price-alerts, dao,
  ens, ramp, dapps, approvals, address-book, devices, kyc + NFT-transfer form.
  app.js: titles map, navigateTo cases, 11 loaders + 12 actions all real
  fetches to :8443 (fail-closed empty states, escapeHtml everywhere,
  "submitted to the blockchain network" confirmations). node --check all 6 JS OK.
  (ens/kyc/ramp/dapps loaders existed but had NO html containers — now added.)
- EXTENSION (user_wallet/extension, MV3): 9 → 16 tabs (added NFTs, Bridge,
  History, Approvals, Contacts, Devices, Alerts). WalletAPI gained bridge
  (routes/quote/transfer/history) + price-alerts methods; popup.js got 7
  loaders + 4 handlers (createElement-based rendering, no innerHTML injection).
  node --check all 6 extension JS OK.
- ANDROID (user_wallet/android): FIXED dead-fragment bug — 12 feature
  fragments (Swap/Send/Receive/Staking/Bridge/NFTs/DeFi/Approvals/AddressBook/
  Devices/KYC/Keystore) were never instantiated anywhere; bottom nav even
  lacked a nav_send handler (Send fell through to Dashboard). Added nav_send +
  nav_more (FeaturesFragment hub w/ 17 buttons, addToBackStack navigation via
  MainActivity.navigateToFeature) + ic_more drawable. NEW fragments+layouts:
  RampFragment, CardsFragment, P2PFragment, PriceAlertsFragment, ENSFragment,
  FeaturesFragment. UserWalletApiService gained createP2POrder, bridge
  (getBridgeRoutes/getBridgeQuote/executeBridgeTransfer/getBridgeHistory),
  price alerts (get/create/delete). BridgeFragment rewritten from fake
  convert-quote+/send to real /bridge/quote+/bridge/transfer. WalletConnect
  fix: WalletConnectSocket.wsBase() now derives from the configured
  UserWalletApiService.apiBaseUrl() (was hardcoded 10.0.2.2); base URL is
  user-configurable+persisted (Settings → Backend Server, BASE_URL_KEY in
  SharedPreferences) so physical devices work.
- iOS (user_wallet/ios): TabView 4 → 5 tabs (added More → FeaturesView hub
  with NavigationLinks to all 17 feature views). NEW views: RampView,
  CardsView, P2PView, PriceAlertsView, ENSView. UserWalletApiService gained
  price-alert methods (get/create/delete); baseURL is now persisted+
  user-configurable (UserDefaults userwallet-base-url, Settings → Backend
  Server section). BridgeView rewritten from fake getConvertQuote+sendTransaction
  to real getBridgeQuote+bridgeTransfer. WalletConnectSocket.wsBase() derives
  from shared.baseURL (was hardcoded localhost).
- RUST duplicate RESOLVED (kept both, documented): user_wallet/rust = full
  typed SDK (126 fns, WalletConnect); rust/userwallet_fetchers = fetcher-
  manager registry crate parallel to rust/admin_fetchers +
  rust/masterwallet_fetchers (per-domain family, not an exact duplicate — do
  not consolidate).
- Verified: separation rule holds (all UserWallet surfaces only call :8443);
  every new action surfaces "submitted to the blockchain network"; all new
  pages inherit theme (web data-theme vars, desktop data-theme vars, extension
  [data-theme] vars, android ThemeManager, ios ThemeManager).
- NOT compile-verified this session: Kotlin (no Android SDK), Swift (no Xcode),
  Go/Rust toolchains absent (session-local in prior sessions).

## Session 15 (2026-08-29) — residual gap closure (Security/Terminal/watch-only)
- WEB: added Security.tsx (check-url/check-address/full-scan) + Terminal.tsx
  (live ticker + real canvas candle chart from /terminal/kline) + watch-only
  enroll form on Wallets; App.tsx now routes 30 pages. tsc clean.
- DESKTOP: added Security (check+scan) + Terminal (canvas candles) pages —
  24→26 nav items/containers. node --check clean.
- EXTENSION: added Security (check/scan) + Terminal (ticker+candle canvas)
  tabs — 16→18. node --check clean.
- ANDROID: added SecurityFragment (+layout), TerminalFragment with a real
  CandleChartView custom View (+layout); watch-only enroll on WalletsFragment
  (watchOnlyFormCard in layout + createWatchOnlyWallet API method). Security +
  Terminal also added to the Features hub. XML parse OK.
- iOS: SecurityView (check/scan), TerminalView (ticker + real Canvas candle
  chart, iOS 15+, with a list fallback), watch-only enroll sheet on
  WalletsView (WatchOnlyView + createWatchOnlyWallet API method). Security +
  Terminal added to FeaturesView. Brace balance OK.
- ALL: real backend fetches only; theme inherited on every new page. The ONLY
  remaining known gap is the extension Google OAuth client_id (deployment
  config, not code) — same as web/android/ios backup helpers which are
  complete.

## Session 16 (2026-08-29) — wl_user_wallet full flat-route parity with canonical backend
- ROUTE PARITY: wl_user_wallet backend originally exposed only ~44 routes;
  canonical go/wallet_api exposes 102 flat routes on the wallet group (auth,
  wallets/balances/transactions/send/sign/simulate/non-evm/chains/amm/swap/
  staking/bridge-prefs/defi/dao/prediction/launchpool/token-sales/copy-trading/
  margin/perp/dapps/ens/cards/kyc/p2p/pricing/fee/approvals/security/terminal).
- IMPLEMENTATION: added wl/store/features.go (new file, ~362 lines) with real
  pgx-backed SQL for price_alerts, p2p_adverts/orders, dao_proposals/votes,
  launchpool, token_sales/entry, token_approvals, fees, kyc_records,
  card_accounts/transactions, margin_positions, perp_positions. Extended
  store.migrate() extras with all full schema CREATE TABLE statements.
- HANDLERS: added wl/handlers/features.go (~277 lines) with gin handlers
  for List/Create/Delete on each new entity, exposing them under the same
  flat-route names as canonical (with per-fetcher license gate from
  middleware.Gate("user_wallet", middleware.CategoryFetcher), so TigerWallet
  SuperAdmin can add/remove/enable/disable each feature per white-label
  client). All queries filter by user_id etc.
- ROUTES: wl/main.go wires 20+ new flat-route handlers (/price-alerts,
  /p2p/adverts|orders, /dao/proposals|vote, /launchpool{,stake,stakes},
  /token-sales/:id/participate, /approvals|:id, /fees+revenue,
  /kyc/status/register/submit, /card{balance,transactions},
  /margin/positions(+close), /perp/positions(+close)) — parity
  improved from 77 missing to ~57 pos-duplicate/syntactic-only misses.
- COMPILE: /tmp/go/bin/go build ./... + go vet ./... = 0 (go1.22.5 session-local
  at /tmp/go; GOMODCACHE=/tmp/gomodcache).
## Session 17 (2026-08-29, this chat) — real-impl gap closure + P0 verification
- TOOLCHAIN (sandbox was reset): reinstalled Go 1.22.12 (/tmp/go122), Rust 1.85
  (rustup, ~/.cargo), cmake 4.4 (pip), libssl-dev+pkg-config+libcurl4-openssl-dev.
- P0 admin/rust auth VERIFIED: all 82 protected routes proxy through fail-closed
  bearer_token() -> admin/go AuthMiddleware (JWT validated upstream). cargo check
  0 errors, cargo test ok. The GAPS.md "handler auth completeness" P0 is CLOSED.
- launchpad_ecosystem: NEW launchpad_onchain.go — real OnChainClient (env
  LAUNCHPAD_RPC_URL/PRIVATE_KEY/CONTRACT_ADDRESS, EIP-155 sign+broadcast+receipt
  wait). ClaimTokens/ClaimRewards now broadcast real txs; allocation CLAIMED and
  stake pending->claimed only after confirmation. tx_hash columns added. 501
  not_implemented stubs REMOVED. go build+vet pass (go-ethereum v1.13.15 added).
- timelock: NEW go.mod (was unpackaged) + NEW onchain.go — real OnChainExecutor
  (TIMELOCK_RPC_URL/EXECUTOR_PRIVATE_KEY). ExecuteTransaction broadcasts the real
  matured tx; fixed ExecutionTxHash uniqueIndex empty-hash collision + latent
  go-redis v9 ZAdd pointer bug (file never compiled before). go build+vet pass.
- dex_connectors/top_20 (Rust): was a non-compiling fake demo (undefined
  get_mock_rate, fabricated tx hash from nanotime, mock router addresses,
  hardcoded rates, YOUR_API_KEY). REWRITTEN: canonical deployed router addresses,
  env RPC (RPC_URL_<chain>/INFURA_API_KEY, public fallback), real eth_chainId
  health check, real xy=k pool quotes + real CoinGecko market quotes, fail-closed
  build_swap (SwapRequest envelope, no fake hash). NEW Cargo.toml + MSRV-1.85
  Cargo.lock. cargo build 0 errors, 5/5 tests pass.
- master_wallet/desktop (C++): implemented PasskeyCredential encode/decode
  (versioned TPC1 format) and TaxAnalyticsService::exportToPDF (real minimal
  PDF 1.4 writer, correct xref + pagination). cmake build 100% pass.
- integrations/tax (C++): implemented SPECIFIC_IDENTIFICATION via designated
  lots (fail-closed without designation); fixed pre-existing 'double double
  longTermLoss' typo + missing <algorithm> + CSV string-concat so the header
  compiles (g++ -fsyntax-only clean).
- Verified real (no changes needed): price_oracle (0 fakes), cex_connectors
  (param credentials), UserWallet no-register onboarding (Onboarding.tsx choose/
  create/import/backup + BackupMnemonic Google Drive API v3 + clipboard), auto-
  sign (Send.tsx autoSendTransaction -> /auto-send + MW auto_signer daemon), MW
  chain/token/fee CRUD (user-chains EVM+non-EVM + /:id/fees; wallet_api
  applyAdminChainOverrides at boot).

