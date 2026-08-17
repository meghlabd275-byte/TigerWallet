# MasterWallet Apps — Full Fetchers & Functionality Report

> Generated: 2026-08-17
> Scope: Complete inventory of every MasterWallet client app's fetchers,
> functionality, and gaps against the canonical backend contract.
> Methodology: automated source analysis of all 9 client apps across 7 platform
> families, cross-referenced against `master_wallet/CANONICAL_API_CONTRACT.md`
> and the 86 routes registered in `master_wallet/backend/main.go`.

---

## Architecture Overview

All MasterWallet apps talk to **ONE canonical backend**: `master_wallet/backend/`
(Go, port **8450**, module `github.com/tigerwallet/master-wallet-backend`). The
contract is documented in `master_wallet/CANONICAL_API_CONTRACT.md`. The backend
exposes **86 routes** across 18 functional domains.

There are **9 MasterWallet client apps** across 7 platform families:

| # | App | Path | Tech | Buildable? |
|---|-----|------|------|-----------|
| 1 | Web | `master_wallet/web/` | React 18 + TS + Vite | ✅ |
| 2 | Desktop | `master_wallet/desktop/` | C++20 + CMake (libcurl) | ✅ |
| 3 | Flutter | `master_wallet/flutter/` | Dart/Flutter | ✅ (pubspec present) |
| 4 | Rust core | `master_wallet/rust/` | Rust (reqwest) | ✅ |
| 5 | Android | `master_wallet/android/` | Kotlin (OkHttp) | ⚠️ source-only (no Gradle/manifest) |
| 6 | iOS | `master_wallet/ios/` | Swift (URLSession) | ⚠️ source-only (no Xcode project) |
| 7 | Chrome ext | `master_wallet/extensions/chrome/` | JS MV3 | ✅ |
| 8 | Brave/Edge/Firefox/Safari exts | `master_wallet/extensions/*_extension/` | JS MV3 | ✅ (byte-identical clones of Chrome) |

The top-level `master_wallet/main.go` is **not** the canonical backend — it is a
thin stdlib reverse-proxy deprecation shim that forwards legacy clients to
`http://localhost:8450`. The canonical backend lives in `master_wallet/backend/`.

### ✅ Separation Verified (the core requirement)

MasterWallet, UserWallet, and Admin apps must be **completely separated**: no
MasterWallet app may import or call UserWallet or Admin client fetchers, and vice
versa. This was verified by grepping **every** MasterWallet client source file for:

- imports of `user_wallet/` or `admin/` client code → **0 hits**
- references to non-canonical backend ports (`:8443`, `:8080`, `:8105`, `:8008`,
  `:8082`, `:9093`) → **0 hits**
- every client points only at `localhost:8450`

**The separation is clean.** MasterWallet apps do NOT access UserWallet or Admin
fetchers/functionality, and vice versa.

> Nuance: the MasterWallet backend *owns* a "UserWallet governance" surface
> (EVM/non-EVM chain CRUD, token CRUD, address derivation, auto-sign) because per
> the system design "one master wallet owns billions of UserWallet addresses" and
> "the master wallet auto-signs ALL UserWallet transactions." This is
> MasterWallet→UserWallet *governance over the backend*, NOT MasterWallet apps
> importing UserWallet-app client code. The apps themselves remain fully separated.

---

## The Canonical Backend Surface (86 routes)

This is the "full functionality" each app is expected to expose. All protected
routes require `Authorization: Bearer <JWT>`.

| Domain | Routes | Count |
|--------|--------|-------|
| Public (no auth) | `/health`, `/api/v1/health`, `/api/v1/chains`, `/api/v1/gas`, `/api/v1/price`, `/api/v1/transactions/history`, `/ws` | 7 |
| Auth | `POST /auth/register`, `POST /auth/login` | 2 |
| Master wallets | `GET/POST /master-wallet`, `GET/PUT/DELETE /:id`, `GET /:id/balance`, `POST /:id/sign`, `POST /:id/revenue-payout`, `POST /:id/withdrawal-request` | 9 |
| Sub-wallets | `GET/POST /:id/sub-wallets`, `GET /:id/sub-wallets/:sid/balance`, `POST /:id/sub-wallets/:sid/transfer` | 4 |
| Transactions | `GET /:id/transactions`, `GET /:id/transactions/:tid`, `POST /:id/transactions`, `POST /:id/transactions/:tid/approve`, `POST /:id/transactions/:tid/reject` | 5 |
| Passkey | `POST /:id/passkey/register`, `GET /:id/passkey/credentials`, `DELETE /:id/passkey/credentials/:credId`, `POST /:id/passkey/verify-assertion` | 4 |
| Policies | `GET/POST /:id/policies`, `PUT/DELETE /:id/policies/:pid` | 4 |
| Fees | `GET/POST /:id/fees`, `DELETE /:id/fees/:fid` | 3 |
| Auto-sign rules | `GET/POST /:id/auto-sign`, `DELETE /:id/auto-sign/:rid` | 3 |
| Users | `GET/POST /:id/users`, `DELETE /:id/users/:uid` | 3 |
| Audit | `GET /:id/audit` | 1 |
| Analytics | `GET /:id/analytics/{volume,transactions,wallets}` | 3 |
| Notifications | `GET/POST /:id/notifications` | 2 |
| Webhooks | `GET/POST /:id/webhooks`, `DELETE /:id/webhooks/:wid` | 3 |
| Treasury | `GET /:id/treasury`, `GET /:id/treasury/transactions`, `POST /:id/treasury/transfer`, `POST /:id/treasury/sweep` | 4 |
| Multisig | `GET/POST /:id/multisig/wallets`, `GET /:id/multisig/wallets/:wid`, `GET /:id/multisig/wallets/:wid/transactions`, `POST /:id/multisig/wallets/:wid/transactions`, `POST /:id/multisig/transactions/:tid/sign`, `POST /:id/multisig/transactions/:tid/execute` | 6 |
| UserWallet governance | EVM chains (4), non-EVM chains (4), tokens (4), derive-user-address (1), user-wallet-addresses (1), auto-sign-transaction (1), auto-sign-logs (1), user-wallet-auto-sign (1), check-auto-sign-policy (1) | 18 |
| Feature flags | `GET/POST /:id/feature-flags`, `PUT/DELETE /:id/feature-flags/:flagId` | 4 |
| **Total** | | **86** |

All routes are prefixed with `/api/v1` (except `/health` and `/ws`).

---

## Per-App Full Fetchers & Functionality

### 1. Web (`master_wallet/web/`) — 83/86 routes ✅ best-in-class

**Primary fetcher:** `src/api.ts` (80 real `fetch()` methods, all MATCH) +
`masterWalletService.ts` (real ethers.js mnemonic/derive helpers + delegations) +
`PasskeyService.ts` (real WebAuthn) + `webSocketService.ts` (real `/ws`).

**Covers:** auth, master wallets (incl. revenue-payout + withdrawal-request),
sub-wallets, transactions+approve/reject, passkey, policies, fees, auto-sign,
users, audit, analytics×3, notifications, webhooks, treasury×4, multisig×6, full
UWM governance (EVM/non-EVM chains, tokens, derive-address, addresses,
auto-sign-transaction, auto-sign-logs), feature-flags, public
(chains/gas/price/history/health), WebSocket.

**Non-canonical services (6 files, 32 methods):** AccountAbstraction,
Biometric, Paymaster, Privacy, SuperAdmin, TaxAnalytics — **none fetch**; all fail
loud with descriptive errors except `TaxAnalyticsService.getSummary()` which
returns a hardcoded **zeroed** TaxSummary (honest zeros, not fabricated financial
data — the only quasi-stub).

---

### 2. Android (`master_wallet/android/`) — 86/86 routes ✅ most complete

**Primary fetcher:** `MasterWalletApiService.kt` (~208 request refs) +
`MasterWalletService.kt` + 9 domain services.

**Covers:** ALL 86 canonical routes (the only client at 100%). Real OkHttp,
Bearer JWT, all `:8450`.

**Non-canonical extras (will 404 against `:8450`):**
- `AccountAbstractionService` `POST /api/aa/submit`
- `PaymasterService` `POST/GET /api/aa/paymaster/{sponsor,balance,fund}`

**1 MISMATCH:** `MasterWalletViewModel.deleteSubWallet` calls
`DELETE /master-wallet/:id/sub-wallets/:sid` — **no such backend route** (canonical
only has GET/POST sub-wallets). This call always 404s.

**12 fail-closed stubs** (PrivacyService×5, SuperAdminService×21 methods,
simulateValidation, importWallet, loadTokens) — all throw descriptive errors,
**no fake data**.

---

### 3. iOS (`master_wallet/ios/`) — 83/86 routes ✅

**Primary fetcher:** `MasterAPIService.swift` (~50 KB, URLSession) + 10 service
files.

**Covers:** same surface as web (83 routes). Real WebAuthn via CryptoKit P-256
in PasskeyService.

**Non-canonical but legitimate:** ERC-4337 bundler + paymaster sponsor endpoints
(configurable, fail-closed when unset) — not part of the canonical MasterWallet
backend but legitimate ERC-4337 integration.

All non-canonical ops (AA, Paymaster, Privacy ZK/stealth, SuperAdmin) are
**fail-closed throws** — no fabricated addresses/sigs/tx hashes.

---

### 4. Desktop C++ (`master_wallet/desktop/`) — 79/86 routes ⚠️ lowest coverage

**Primary fetcher:** `api_client.cpp` (libcurl, 51 HTTP refs) + 9 service pairs.

**Covers:** public(5 incl /ws), auth, master-wallets(10), sub-wallets(3/4),
transactions(5), passkey(4), policies(4), fees(3), auto-sign(3), users(3),
audit(1), analytics(3), notifications(2), webhooks(3), treasury(4), multisig(7
incl detail), UWM governance(16), feature-flags(4).

**Missing 7:** `GET /health`, `GET /api/v1/health`,
**`POST /master-wallet/:id/sub-wallets` (create sub-wallet)**,
`POST /user-wallet-auto-sign`, `POST /check-auto-sign-policy`, + 2 more.

Desktop is the **only client that can't create sub-wallets**.

Non-canonical ops (AA, paymaster, ZK/CoinJoin/stealth, HD derivation, message
signing, historical price) all **fail-closed throw / return honest empty**.

---

### 5. Flutter (`master_wallet/flutter/`) — 82/86 routes ✅

**Primary fetcher:** `master_wallet_service.dart` (66 HTTP method refs) + 15
service/feature files. Now a full runnable app (`main.dart` + AuthGate + 6-tab
dashboard).

**Covers:** 82 routes, all real `package:http` to `:8450`.

**Missing 4:** `POST /user-wallet-auto-sign`, `POST /check-auto-sign-policy`,
`GET /api/v1/health`, and a passkey DELETE collection variant.

All non-canonical ops (AA, paymaster sponsorship, Privacy ZK/stealth/CoinJoin,
SuperAdmin, tax engine, batch tx, treasury allocations/reports, multisig owner
mgmt, policy testing/logs, audit reports) **throw UnimplementedError** — no fakes.
`Random.secure()`/`FortunaRandom` used for all crypto (no insecure
`dart:math Random()`).

---

### 6. Rust core (`master_wallet/rust/`) — 82/86 routes ✅

**Single-file crate** `lib.rs` (1622 lines). `BackendClient` struct with 82 public
async reqwest methods + `MasterWalletService` crypto orchestrator (delegates all
I/O to BackendClient).

**Covers:** 82 routes, all MATCH. Real on-device crypto (BIP-39/32/44, secp256k1,
keccak, scrypt/AES-GCM, P-256 passkey) — verified real, not stubs.

**Missing 4:** `GET /api/v1/health`, `GET /ws` (reqwest is HTTP-only — no
WebSocket client), `POST /user-wallet-auto-sign`, `POST /check-auto-sign-policy`.

---

### 7. Chrome Extension (gold-standard reference) — 82/86 routes ✅

**Primary fetcher:** `services/apiClient.js` (authedFetch, Bearer JWT) +
`masterWalletService.js` (171 refs, 59 unique paths) + 7 domain services reusing
canonical routes.

**Covers:** 82/86 (95.3%). All fetchers go through `authedFetch`, fail-closed.

**Missing 4:** `GET /api/v1/health`, `GET /ws`, `POST /user-wallet-auto-sign`,
`POST /check-auto-sign-policy`.

**CRITICAL WIRING GAP:** **21 `masterWalletService` methods are implemented but
have 0 cases in `background.js`'s `MW_RELAY` switch** — so the popup UI cannot
invoke them. This affects the entire UWM-governance surface + feature-flags +
`updateMasterWallet`. The fetchers exist but are unreachable from the UI.

---

### 8. Brave / Edge / Firefox / Safari Extensions — 82/86 each ✅

Verified **byte-identical** `services/` directories to Chrome (`diff -rq` returns
empty). Same 82/86 coverage, same 4 missing routes, same `background.js` relay
gap. Only the manifest description string differs (Safari).

---

## Consolidated Gap Matrix

| App | Routes covered | Missing routes | Non-canonical/404 calls | Fake data | Cross-app contamination |
|-----|---------------|----------------|------------------------|-----------|------------------------|
| Web | 83/86 | 3 | 0 | 0 (1 zeroed TaxSummary) | None |
| Android | 86/86 | 0 | 5 (4× `/api/aa/*` + 1× `DELETE sub-wallets/:sid`) | 0 | None |
| iOS | 83/86 | 3 | 0 (2 configurable ERC-4337, fail-closed) | 0 | None |
| Desktop C++ | 79/86 | 7 | 0 | 0 | None |
| Flutter | 82/86 | 4 | 0 | 0 | None |
| Rust | 82/86 | 4 | 0 | 0 | None |
| Chrome ext | 82/86 | 4 | 0 | 0 | None |
| Brave/Edge/Firefox/Safari ext | 82/86 each | 4 each | 0 | 0 | None |

---

## Detailed Gaps (what's still missing per app)

### Gap A — Universal gaps (affect MOST clients)

These 3 routes are missing from **6 of 9** clients:

1. **`POST /master-wallet/:id/user-wallet-auto-sign`** — missing on Web, iOS,
   Desktop, Flutter, Rust, Chrome ext (×5). Present only on Android. This is the
   MasterWallet-owner auto-sign bridge (policy-based auto-approval of UserWallet
   txs).
2. **`POST /master-wallet/:id/check-auto-sign-policy`** — same distribution.
   Policy-only check (server-to-server).
3. **`GET /api/v1/health`** — missing on Web, iOS, Desktop, Flutter, Rust, Chrome
   ext. (Most wire `/health` but not the `/api/v1/health` alias — low impact.)

### Gap B — WebSocket (`/ws`) missing on 3 clients

- **Rust**: reqwest is HTTP-only; no WebSocket client at all.
- **Chrome/Brave/Edge/Firefox/Safari exts**: no WebSocket client wired.
- (Web, iOS, Desktop, Flutter, Android all have real `/ws` clients.)

### Gap C — Desktop C++ is the weakest (79/86)

Missing 7 routes — in addition to A+B above, it's also missing:
- **`POST /master-wallet/:id/sub-wallets`** (create sub-wallet) — a core
  master-wallet feature.
- 2 others.

Desktop is the only client that can't create sub-wallets.

### Gap D — Android non-canonical calls that will 404

- `DELETE /master-wallet/:id/sub-wallets/:sid` (in
  `MasterWalletViewModel.deleteSubWallet`) — no such backend route.
- 4× `/api/aa/*` (AccountAbstraction + Paymaster) — not in the canonical 86; will
  404 against `:8450`.

### Gap E — Chrome-family extension relay wiring gap (HIGH impact, UI-invisible)

21 `masterWalletService.js` methods (the **entire UserWallet-governance surface +
feature-flags + `updateMasterWallet`**) are implemented as fetchers but have
**zero cases in `background.js`'s `MW_RELAY` switch**. The popup cannot reach them
— they're dead code from the UI's perspective. This affects all 5 extensions
(brave/edge/firefox/safari are byte-identical clones, so the same gap propagates).

### Gap F — Web TaxAnalyticsService stub

`TaxAnalyticsService.getSummary()` returns a hardcoded zeroed `TaxSummary` instead
of fetching. Minor (it's honest zeros, not fabricated numbers), but it's the only
non-fail-closed stub on web.

### Gap G — Build-config gaps (not fetcher gaps, but app-availability gaps)

- **Android**: full Kotlin source but **no `build.gradle`, no
  `AndroidManifest.xml`, no Gradle wrapper** → cannot build an APK.
- **iOS**: full Swift source but **no `.xcodeproj`/`Info.plist`** → cannot build
  an IPA.
- Both are source-complete but not standalone-buildable. All other clients (web,
  desktop, flutter, rust, 5 extensions) are fully buildable.

---

## Bottom Line

**Separation: 100% clean.** No MasterWallet app imports or calls UserWallet-app
or Admin-app client fetchers. All point only at `:8450`. The separation
requirement is satisfied.

**Fetcher coverage: strong but uneven.**
- **Best:** Android (86/86, though with 5 non-canonical 404 calls), Web (83/86),
  iOS (83/86).
- **Good:** Flutter (82/86), Rust (82/86), Chrome ext (82/86).
- **Weakest:** Desktop C++ (79/86 — can't create sub-wallets).

**The 3 highest-priority real gaps to close:**
1. **Extension relay wiring (Gap E)** — 21 implemented-but-unreachable fetchers
   across all 5 browser extensions. Add the missing `MW_RELAY` cases in
   `background.js` (×5).
2. **Universal `user-wallet-auto-sign` + `check-auto-sign-policy` (Gap A)** — add
   these 2 methods to Web, iOS, Desktop, Flutter, Rust, and the 5 extensions
   (Android already has them).
3. **Desktop C++ sub-wallet creation (Gap C)** — add
   `POST /master-wallet/:id/sub-wallets` + the 6 other missing routes to reach
   parity.

**No fabricated/fake crypto or data exists** in any MasterWallet client — all
non-canonical paths fail-closed. The separation boundary is intact.
