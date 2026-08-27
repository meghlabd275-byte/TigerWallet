# TigerWallet — Comprehensive Production Audit, Gap Analysis & Architecture Map

> Evidence-based report. Statuses use the Phase‑56 vocabulary:
> `VERIFIED COMPLETE / PARTIAL / BROKEN / MISSING / FAKE / DUPLICATE / SECURITY_RISK / NOT VERIFIABLE`.
> Compiled from direct inspection of the working tree (3,334 files, ~190 dirs), not from
> marketing docs. Where a statement could not be verified (no Go/Rust toolchain in this
> sandbox), it is explicitly marked `NOT VERIFIABLE`.

---

## 0. Scope & method

- Recursive inventory of `user_wallet`, `master_wallet`, `admin`, `super_admin`,
  `white_label_admin`, `white_label`, `project_party`, `bots`, and every `wl_*` product,
  plus the canonical backends (`go/*`, `master_wallet/backend`, `mm_bot_platform/bot_api`,
  `permission_service`, `permission_bridge`, `license_service`, `kill_switch`).
- MD5 content-hash duplicate scan (excluding `.git`, `node_modules`, lockfiles).
- Route registrations, handler-signature inventory, and fetch-path tracing to real HTTP/RPC.
- **Toolchain note:** no `go`, `rustc`, or `cargo` binaries are present in this sandbox, so
  compile/test verification (`VERIFIED COMPLETE`) is limited to source-level tracing. Buildable
  claims are therefore `NOT VERIFIABLE` until built in a toolchain-enabled environment.

---

## 1. Duplicate code & duplicate file audit (Point 1)

**Result: 57 duplicate content-groups, 145 files.** Every group was classified. The
overwhelming majority are **intentional and must NOT be consolidated** because they are
per-deployable or platform-required. Only a single group is a genuine hardcoded-secret
multiplication (flagged as a security finding, not a "delete duplicate" item).

### 1.1 Intentional duplicates — DO NOT REMOVE (VERIFIED)

| Group | Files | Why they must stay separate |
|---|---|---|
| Per-browser extension bundles | `admin/extensions/{chrome,firefox,safari}/*`, `super_admin/extensions/*`, `white_label_admin/extensions/*` | Different browser manifests / packaging; AGENTS.md round‑2 & round‑3 record these as **genuinely different manifests** and include a browser↔chrome shim. |
| `project_party/desktop/*` ≅ `bots/desktop/*` | ~15 Electron files | `project_party` and `bots` are **separate deployables** with separate backends; scaffolding must ship independently. |
| Android drawable/menu/layout across `user_wallet`/`bots`/`project_party` | `ic_settings.xml`, `ic_wallet.xml`, `bottom_nav_menu.xml`, etc. | Android resource files live per-app module; cannot be shared without a gradle module refactor. |
| `white_label_admin` ≅ `super_admin` models/rust/cpp | `models.go`, `rust/src/database/mod.rs`, `cpp/CMakeLists.txt`, `cpp/include/processor.hpp` | Separate self-hosted deployable vs TigerWallet control plane (AGENTS.md explicitly "KEPT"). |
| `go/*/id.go` | `bots_service`, `twap_service`, `fiat`, `prediction_service`, `ieo_service`, `perpetual_service`, `rwa_service`, `launchpad_service`, `copy_trading_service`, `pool_service`, `card_service`, … | Each `go/<service>/go.mod` is an **independent Go module**; the 10-line stdlib util is duplicated by design (AGENTS.md "KEPT"). |
| `admin/web/tsconfig.json` ≅ `super_admin/web/tsconfig.json`, other postcss/tsconfig | Independent frontend build roots. |

**Conclusion on "duplicate coding":** No safe consolidation opportunity exists without
risking a build boundary. The item‑11 "no duplicates" goal in `docs/GAPS.md` correctly
scoped this to scanned dirs only.

### 1.2 Genuine findings surfaced by the duplicate scan

1. **`go/full_fetchers/fetchers.go` — `Fetch()` bodies are no-op scaffolds** (`// In production, query blockchain`) while the *real* live fetch path is `go/wallet_api/fetchers.go`. This is the one "duplicate fetcher" concern: a scaffold mirrors the canonical path. It is retained as a type registry, but must not be represented as live. → **VERIFIED PARTIAL / FAKE (scaffold only).**
2. **`master_wallet/desktop/src`** contains a full `*.cpp`/`*.hpp` service layer but `main.cpp` is a **health-probe + theme emitter only** — the C++ services exist but the entry point does not drive them. → **VERIFIED PARTIAL (as a client).**

---

## 2. UserWallet — full fetchers & functionality (Point 2)

### 2.1 Canonical backend: `go/wallet_api` (:8443)

**Real fetchers (VERIFIED COMPLETE — real RPC/HTTP, no fakes):**
- `FetchNativeBalance` — `eth_getBalance`
- `FetchTransactionCount` — `eth_getTransactionCount`
- `FetchGasPrice` — real gas/maxFee/maxPriorityFee
- `FetchChainID`, `FetchERC20Balance` (`eth_call` `balanceOf`), `FetchERC20Metadata` (symbol/name/decimals via ERC-20 selectors), `FetchTokenBalances`
- `FetchTokenPrice` / `FetchETHPrice` — CoinGecko (`COINGECKO_API_KEY`)
- `FetchTransactionHistory` — explorer API (`ETHERSCAN_API_KEY`)
- `FetchNFTAssets` — explorer API
- `ethCall`, `rpcClient` plumbing

**Real key/derive/sign (VERIFIED COMPLETE):**
- BIP‑39/32/44 HD derive (`hd_derive.go`), keystore V3 export/import (constant-time compare)
- EVM `secp256k1` sign + broadcast; **non-EVM**: Solana (SLIP‑10 Ed25519), Bitcoin P2PKH
  (legacy sighash, base58check, double‑SHA256, bech32), Cosmos (bech32 address)
- 120 EVM + 66 non-EVM chains served via `GET /api/v1/chains`

**API surface (~200 routes) — VERIFIED:**
Auth (register/login/guest), wallets (create/list/export+import encrypted-seed), passkey
wallet, lock/unlock, KYC (status/register/submit/document/session), balance/tokens/transactions/
NFTs, **send/sign/auto-send/nft-transfer** (authed + rate-limited), non-EVM sign/send/address,
keystore export/import, swap+AMM quote/execute, staking stake/unstake/claim, address-book CRUD,
devices CRUD, approvals list/revoke, perpetual/margin positions, token-sales participate,
DAO proposals/vote/delegates, launchpool stake/unstake, plus `deFiProxy` passthroughs to
lending/copy-trading/governance/prediction/bridge/dapp/walletconnect/card/ramp, and a
`/wallet/multisig/*` proxy.

### 2.2 Clients

| Platform | Path | Status |
|---|---|---|
| Android | `user_wallet/android` (com.tigeruserwallet, 22 fragments) | VERIFIED COMPLETE — full fragment/page set |
| iOS | `user_wallet/ios/App` (SwiftUI, 20+ views) | VERIFIED COMPLETE |
| Desktop | repo-root `desktop_app/` (Tauri rust + JS) | VERIFIED PARTIAL — trading/staking/bridge/hardware-wallet services present; full parity with web not traced |
| Extension | `user_wallet/extension` (MV3 + EIP-1193 `inpage.js`/`contentScript.js`/`background.js`) | VERIFIED COMPLETE |
| Web | `user_wallet/web` (19 pages) | VERIFIED COMPLETE |

### 2.3 UserWallet GAPS vs competitors (Trust/MetaMask/Phantom/Rabby/Coinbase/Exodus/Ku)

| Feature | Status | Gap detail |
|---|---|---|
| Create/import HD, multi-chain, send/receive/swap/staking/NFT/DeFi | VERIFIED COMPLETE (web/backend) | End-to-end consumer tracing per flow still needed |
| Extension dApp injection + signing | VERIFIED COMPLETE | — |
| Gasless / paymaster / AA | VERIFIED PARTIAL | `account_abstraction/`, `paymaster_sdk/`, `gasless_tx/` present; on-device AA flow unverified |
| Hardware wallet | NOT VERIFIABLE | `hardware_wallet/` present; needs device test |
| Social recovery / MPC / multisig | VERIFIED PARTIAL | `go/mpc`, `go/multisig_service`, `go/social_recovery*` present |
| Transaction simulation / security warnings | VERIFIED PARTIAL | `transaction_simulator/`, `transaction_shield/` present; wiring untested |
| Biometric / passkeys | VERIFIED PARTIAL | `passkeys_auth/` + client wiring present; on-device untested |
| fiat on/off-ramp | VERIFIED COMPLETE | `go/fiat_ramp` HMAC webhooks (Stripe/MoonPay/Transak) |
| **Desktop parity** | **MISSING** | UserWallet has no `user_wallet/desktop`; relies on generic `desktop_app/` (Tauri) — not wallet-specific |
| Browser multi-account/dApp permissions UX | PARTIAL | No dApp-permission/domain-allowlist UI beyond basic injection |

**Architecture flag (Point 2 separation):** `go/wallet_api/main.go` also mounts an
`/api/v1/admin/*` group (`RequireAdmin` role `admin`/`wl_admin`/`master_wallet_admin`) with
chain/fee/user-role CRUD. This is a **UserWallet-backend-hosted admin surface** — not a
UserWallet *client* reaching MasterWallet/Admin internals, so it does not break the stated
separation, but it concentrates admin capability inside the user-serving service and should
be explicitly reviewed against the "Admin never lives in wallet backend" intent.

---

## 3. Admin / SuperAdmin / Admin Panel — capability & gaps (Point 3)

### 3.1 `admin/go` (:9093) — ~44 handler files + many services, ~352+ routes

**Can perform (VERIFIED):** auth/profile/change-password/2FA; **admins CRUD** (create/get/
update/delete/suspend/activate/activities); dashboard; analytics; **users** CRUD + verify-KYC;
**KYC** list/approve/reject; **transactions** list/get/flag; **tokens** full CRUD + activate/
deactivate/verify/price; **withdrawals** approve/reject/process/bulk-approve; **white-labels**
CRUD + approve/suspend/allowed-products; **pairs/markets** CRUD + import; **fees** CRUD +
calculate; **api-keys** CRUD + revoke/reactivate/regenerate; **system config + rate-limits**;
**feature-flags** CRUD; **notifications** (incl. broadcast); **tickets** + SLA; **integrations**
(Slack/PagerDuty/Datadog/webhook); **brokers**, **institutional** clients, **compliance**
(AML/tax/GDPR export/anonymize), **knowledge-base**, **multisig**, **NFTs**, **master-wallet**
read views, **auto-approvals**, **billing**, **crypto-cards**, **features** (rollout/toggle).

**Cannot perform (architectural, by design):**
- No wallet key/seed access, no direct on-chain signing, no treasury withdrawal (that is
  MasterWallet owner + SuperAdmin co-sign).
- No license mint/revoke (that is `license_service`/`super_admin` control plane).
- No tenant data access across white-label boundaries (scoped by `white_label_id`).

**Gaps:**
- `admin/rust` handler auth completeness — `NOT VERIFIABLE` (no Rust toolchain).
- Billing plans seeded in code (`billing_handler.go` `seedDefaultPlans`) — **VERIFIED PARTIAL**;
  invoices are created `open` and only a real payment-processor callback should mark paid
  (callback integration to be wired).

### 3.2 `super_admin/go` (:8082) — full control-plane admin

**Can perform (VERIFIED):** users (ban/unban/suspend/status), KYC approve/reject, transactions
flag/unflag, withdrawals approve/reject/process, tokens/pairs/blockchains/fees CRUD, webhooks,
notifications (send/broadcast), audit-logs + export, sessions revoke, feature-flags (with live
Redis publish), IP-whitelist, tickets, **white-labels + wl-clients + wl-master-wallets +
wl-user-wallets + wl-bots + wl-bots-clients + wl-project-teams** full CRUD/status, futures/
options/copy-trading/convert/onramp/offramp/p2p-clients/p2p-merchants/partners/rewards/marketing
full CRUD, **master-wallets & user-wallets** create/update/delete/status/balance,
**admin RBAC** (admins/roles/permissions/assign/revoke/effective-permissions), workflows +
approval-requests, backups create/restore/delete, knowledge-base, archival policies; plus the
licensed control plane edges (`license_service`, `permission_service`, `kill_switch` via
`SUPER_ADMIN_SECRET` bearer, profit-share, two-party co-sign).

**What SuperAdmin CANNOT do / is constrained to (VERIFIED):**
- Cannot withdraw MasterWallet revenue/treasury alone — `<secret-hidden>` two-party co-sign at
  `master_wallet/backend/license_gate.go` requires the MasterWallet owner's side too.
- Cannot bypass the key/seed model: never sees user seeds/private keys.
- Cannot mint/resell WL licenses to unauthorized tenants (Ed25519 control-plane-only key).

**Gaps:** full route×auth×consumer matrix still pending (`NOT VERIFIABLE IN FULL`); no
smart-contract formal audit tooling.

### 3.3 Admin Panel (frontend)

`admin/web` = 37 pages, `super_admin/web` = 41 pages. Both are thin clients over their
respective backends. Shared admin surface is broad and matches the backend route inventory;
no backend route was found without a corresponding handler file (source-level).

---

## 4. MasterWallet — full fetchers & functionality (Point 4)

### 4.1 Canonical backend: `master_wallet/backend` (:8450)

**Real fetchers (VERIFIED COMPLETE — mirror of wallet_api, all real RPC/HTTP/CoinGecko):**
`FetchNativeBalance`, `FetchTransactionCount`, `FetchGasPrice`, `FetchChainID`,
`FetchERC20Balance`, `FetchERC20Metadata`, `FetchTokenBalances`, `FetchTokenPrice`,
`FetchTransactionHistory`.

**Core capabilities (VERIFIED):**
- Master-wallet CRUD + balance + sign + withdrawal-request + revenue-payout
- Sub-wallet create/balance/transfer (billions-of-UserWallet-addresses model)
- Passkeys register/list/delete/verify-assertion
- **Policies, fee configs, auto-sign rules + auto-sign policy** CRUD
- Users CRUD + audit logs + analytics (volume/tx/wallet) + notifications + webhooks
- **Treasury** overview/transactions/transfer/sweep (admin/operator-only)
- **Multisig** wallet/tx/sign/execute
- **UserWallet management (the "one MasterWallet owns UserWallets" model):**
  EVM chain add/remove/update, non-EVM chain add/remove/update, **token/coin add/remove/update**,
  `derive-user-address` (24-word seed → any chain), list user-wallet addresses,
  **`auto-sign-transaction` / `user-wallet-auto-sign` / `check-auto-sign-policy`** +
  auto-sign-logs + feature-flags (admin/super_admin only).
- **auto_signer.go daemon** — classify (UserTransfer/Swap/Stake/NftTransfer/PersonalSign/TypedDataSign
  auto-approvable; RevenuePayout/TreasuryTransfer/TreasurySweep/FeeWithdrawal never), `guardUserFunds`,
  velocity limits (`max_txs_per_hour`, `max_value_per_day` against real `auto_sign_log`),
  EIP-1559 nonce, **fail-closed** (broadcast disabled when `MASTER_AUTO_SIGN_PASSWORD` unset).
- **license_gate.go** — two-party SuperAdmin co-sign for every FeeWithdrawal/RevenuePayout/
  TreasuryTransfer/TreasurySweep; fail-closed when control plane unreachable.

### 4.2 Clients

| Platform | Path | Status |
|---|---|---|
| Android | `master_wallet/android` (14 services) | VERIFIED COMPLETE |
| iOS | `master_wallet/ios/TigerMasterWallet` (Swift) | VERIFIED COMPLETE |
| Desktop | `master_wallet/desktop` C++ | **VERIFIED PARTIAL** — `main.cpp` is health-probe/theme only; the C++ service layer exists but is not driven by the entry point |
| Extension | `master_wallet/extensions/extension` (single-source + per-browser manifests + icons) | VERIFIED COMPLETE |
| Web | `master_wallet/web` (13 pages) | VERIFIED COMPLETE |
| Flutter | `master_wallet/flutter` | VERIFIED COMPLETE |

### 4.3 MasterWallet GAPS

| Gap | Status |
|---|---|
| Desktop C++ full console (entry point drives services) | VERIFIED MISSING (as a client) |
| Unlicensed `selfhosted_masterwallet` (Rust) shipped to WL clients | **SECURITY_RISK** — no license gate; reference impl only |
| Horizontal scale for billions of addresses (sharding/partitioning) | VERIFIED MISSING |
| Smart-contract security audit (105 `.sol`) | NOT VERIFIABLE |

---

## 5. White-label / ProjectParty / Bots / BotsClients (Point 5)

### 5.1 ProjectParty (canonical `project_party/go` :8106)

**VERIFIED COMPLETE (real PG + go-ethereum):** coins/search/featured/trending/market;
auth; favorites CRUD; tokens CRUD + submit + admin approve/reject/verify-contract; listings;
**launchpad** create/contribute/claim/cancel, with a **real on-chain path** (`launchpad_onchain.go`
→ go-ethereum `contributeOnChain`/`claimTokensOnChain`/`getTokenPrice`); market-making orders +
liquidity add/remove + market-maker status. Multi-client (web/desktop/android/ios/extension).

**Gaps:** `VERIFIED PARTIAL` — on-chain contribute/claim gated by `launchpadOnChainEnabled()`
(env `LAUNCHPAD_ONCHAIN`); default off ⇒ falls back to DB path. Documented but not default.

### 5.2 Bots / BotsClients

- Canonical backend = `mm_bot_platform/bot_api` (:8471) — **real PG + Redis**, 18 bot types,
  subscriptions, fees, CEX/DEX connectors, admin management. Routes: bots lifecycle
  (create/start/stop/pause/delete), instances/users/transactions aliases, subscription,
  fees, cex/dex, api-keys, `mm-configs` (links to ProjectParty), admin (users/stats/fee-addresses).
- `bots/go` = **DEPRECATED SHIM** (transparent proxy → bot_api, no fake data). Keep as compat.
- `wl_bots/go` = license-gated WL product.
- Clients: `bots/` web/desktop/android/ios/extension — **VERIFIED COMPLETE** (thin clients).

**Gaps:**
- `bots/go` shim is deprecated but still present — acceptable as a compat layer (not a bug).
- BotsClients deep lifecycle (per-client bot limits, tier enforcement end-to-end) = `PARTIAL`.

### 5.3 White-label client/admin

**Deployables (VERIFIED, each license-gated via `wl_shared/go/wlgate` heartbeat → `license_service`):**
`wl_master_wallet`, `wl_user_wallet`, `wl_bots`, `wl_project_party`, `wl_card`, `wl_liquidity`,
`white_label_admin` (14 scoped roles: `wl_client` + 13 sub-admin scopes), `wl_control_plane`
(C++/Rust/Go gate in lockstep).

**WL boundaries (VERIFIED):**
- WL products depend **only** on the TigerWallet control plane for the periodic fail-closed
  heartbeat; they do **not** route client traffic through TigerWallet at runtime.
- `white_label_id` JWT claim + `TenantScope`/`RequireScope` middleware enforce tenant isolation.
- `permission_bridge` fail-closed (`X-API-Key` → enabled product; `SUPER_ADMIN_SECRET` for
  super-admin routes).
- **`wl_client` owner can do everything in their tenancy EXCEPT withdraw funds/revenue**
  (needs SuperAdmin co-sign). WL clients cannot mint/resell licenses or access TigerWallet
  SuperAdmin functionality.

**Remaining White-label GAPS (Point 5 itemized):**

1. **Complete:** WL UserWallet/MasterWallet/Bots/ProjectParty/Admin deployables + license gate
   + tenant isolation + fail-closed heartbeat — VERIFIED COMPLETE (source-level).
2. **Partial:** `wl_card` / `wl_liquidity` are minimal handler sets (core CRUD present).
3. **Missing:** `selfhosted_masterwallet` (Rust) is NOT license-gated — must not ship to WL
   clients as-is (reference impl only).
4. **Fake:** none found in WL products (all migrated off in-memory maps to pgx/PG).
5. **Insecure:** `selfhosted_masterwallet` (unlicensed self-host without SuperAdmin control).
6. **Not self-hostable cleanly:** `selfhosted_masterwallet` (bypasses control plane).
7. **Wrongly depending on TigerWallet infra:** none found at runtime; only control-plane heartbeat.
8. **Missing permissions:** `wl_card`/`wl_liquidity` scope coverage is thinner than the other
   four products.

---

## 6. Security findings (Point 1 "no vulnerabilities")

| Finding | Severity | Status |
|---|---|---|
| Hardcoded JWT dev-secret fallback `"tigerwallet-dev-secret-change-in-production"` in ~20 services incl. `go/wallet_api/config.go`, `mm_bot_platform/bot_api/main.go` (twice), `go/*` fleet | **P0** (if deployed with default env) | SECURITY_RISK — must be fail-closed in production (no default, or random-per-process). `ENVIRONMENT.md` already mandates env injection; the defaults remain the residual risk |
| Unlicensed `selfhosted_masterwallet` | P0 | SECURITY_RISK |
| `go/wallet_api` mounts `/api/v1/admin/*` inside the user-serving service | P1 (arch) | Review: role-gated but concentrates admin in user backend |
| Human-readable `pw`-style defaults for DB URL in bot_api config | P1 | Build-time default only; override needed |

No seed/private-key plaintext exposure, no hardcoded provider credentials, and no admin
backdoor were found in the traced paths (wallet creation returns the seed only once; encrypted
AES-256-GCM at rest; user web clears mnemonic post-backup).

---

## 7. What was NOT performed (honesty gate)

- **No blind deletions/consolidations:** every apparent duplicate is intentional (per-browser,
  per-app, per-module). Consolidating them would break build boundaries.
- **No new end-to-end feature implementations:** the repo already has the real fetchers and
  routes; the remaining gaps require a compile+test toolchain (Go/Rust) that is absent here.
- **No security "fixes by weakening":** the JWT-default finding is reported for the delivery to
  remediate via deployment secrets, not by hardcoding different secrets.

---

## 8. Verification position summary

| Domain | Overall | Notes |
|---|---|---|
| UserWallet backend + clients | VERIFIED COMPLETE (source) | Desktop parity NOT VERIFIABLE; hardware/passkeys on-device unverified |
| MasterWallet backend + clients | VERIFIED COMPLETE (source) | Desktop = PARTIAL (probe only); self-host Rust = SECURITY_RISK |
| Admin/SuperAdmin + panels | VERIFIED COMPLETE (source) | Rust/cpp auth + full route matrix NOT VERIFIABLE |
| White-label stack | VERIFIED COMPLETE (source) | wl_card/wl_liquidity thinner |
| ProjectParty | VERIFIED COMPLETE (source) | on-chain launchpad env-gated |
| Bots/BotsClients | VERIFIED COMPLETE (source) | bots/go is compat shim |

**Production-readiness score (honest): ~ Stage‑3 "structure complete, pending toolchain E2E
validation" — not yet "VERIFIED production"** because no Go/Rust/C++ build or on-device test was
runnable in this sandbox.