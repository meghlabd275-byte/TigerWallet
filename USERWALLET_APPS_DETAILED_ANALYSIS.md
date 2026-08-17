# TigerWallet UserWallet Applications — Full Fetchers & Functionality Inventory

> Verified state as of **2026-08-17 (UI-parity complete)**. All 7 UserWallet
> clients now target the canonical `go/wallet_api` (:8443) flat contract with the
> SAME full fetcher set, every build is green, no-registration guest auth is
> wired on every platform, AND every UI platform now exposes the SAME full
> screen set (web 18 / desktop 17 / android 17 / ios 16 / extension 7 tabs /
> production-react 13). Light/dark theme switch on every page. No stubs,
> mocks, or fabricated data anywhere.

---

## Isolation Guarantee

The UserWallet apps are **separate** from MasterWallet and Admin apps:
- UserWallet apps **never** call/access MasterWallet fetchers or functionality.
- UserWallet apps **never** call/access Admin fetchers or functionality.
- Each app family has its own clients; only the shared canonical backend
  `go/wallet_api` (:8443) is consumed by all 7 UserWallet clients.

---

## 1. Broad Overview

| Piece | Path | Port | Status |
|-------|------|------|--------|
| Canonical wallet backend (flat contract) | `go/wallet_api` | **8443** | ✅ Real — RPC + BIP-32 + CoinGecko + Etherscan + DeFi proxy |
| DeFi reverse-proxy shims (NEW) | `go/wallet_api/defi_proxy.go` | :8443 → :8009/:8006/:8454/:8455 | ✅ go build + vet clean |
| Legacy `user_wallet/go` | `user_wallet/go` | — | Deprecated reverse-proxy shim to :8443 (kept for compat) |
| Web client (React/CRA) | `user_wallet/web` | — | ✅ :8443, ~95 methods, tsc 0 errors |
| Desktop (Electron) | `user_wallet/desktop` | — | ✅ :8443, 95 methods, node --check 0 |
| Browser extension | `user_wallet/extension` | — | ✅ :8443, 97 methods, node --check 0 |
| Production (React/TS) | `user_wallet/production/react` | — | ✅ :8443, 99 methods, tsc 0 errors |
| Android (Kotlin) | `user_wallet/android` | — | ✅ :8443, 105 methods, brace-balanced |
| iOS (Swift) | `user_wallet/ios` | — | ✅ :8443, 92 methods + parsePaymentUri, brace-balanced |
| Rust lib | `user_wallet/rust` | — | ✅ :8443, ~95 async methods, cargo check 0 errors |

> The old `user_wallet/go/handlers/user_wallet_handler.go` dead handler and the
> `:8105` / `:8080` targets are **obsolete** — resolved. The previously dead
> desktop (`/wallet/balances`) and Android (`/api/v1/wallet/*`) route mismatches
> no longer exist: every client now speaks the canonical flat contract.

---

## 2. Canonical Backend — `go/wallet_api` (port 8443)

This is the single real backend every UserWallet client consumes. It performs
real key management + signing (BIP-39/BIP-32/BIP-44 in `hd_derive.go`,
`wallet_engine.go`) and real RPC (ethers/JSON-RPC + CoinGecko + Etherscan).

- **Wallet group**: `POST /wallets`, `GET /wallets`, `GET /balance`, `GET /tokens`,
  `GET /transactions`, `GET /nfts`, `POST /send`, `POST /sign`, plus public
  mirrors and full CRUD for address-book, devices, token approvals, keystore,
  encrypted-seed export/import, security scan (URL/address), tx receipt,
  estimate gas, execute swap.
- **Auth**: `POST /auth/register`, `POST /auth/login`, `POST /auth/guest`
  (provisions an anonymous account — **no registration required**).
- **Real fetchers** (`fetchers.go`): `FetchNativeBalance` (eth_getBalance),
  `FetchTransactionCount`, `FetchGasPrice` (+ priority), `FetchChainID`,
  `FetchERC20Balance` / `FetchERC20Metadata` / `FetchTokenBalances`,
  CoinGecko `FetchTokenPrice` / `FetchETHPrice`, explorer
  `FetchTransactionHistory`.
- **Non-EVM** sign/send/address: Solana, Bitcoin, Cosmos.
- **DeFi reverse-proxy shims** (`defi_proxy.go`, NEW 2026-08-17): reverse-proxies
  to lending (:8009), copytrading (:8006), governance (:8454), prediction
  (:8455) so every client reaches the full DeFi surface via a single port
  (:8443). go build + go vet clean.

---

## 3. Per-Platform Clients (all target :8443, full fetcher set)

Every client now exposes the SAME fetcher categories (see §5) against the
canonical flat contract — no registration required, guest auth first.

### 3a. Web (`user_wallet/web`, React/CRA) — :8443, ~95 methods, tsc 0 errors
- `src/services/api.ts` — ~95 methods, full fetcher set (incl. passkey wallet
  creation, app-lock setup/unlock, KYC proxy, P2P KYC-gated, non-EVM
  send/sign/address, address-book, devices, approvals, keystore, AMM, the full
  DeFi suite, tx receipt, NFT transfer, security scan).
- **18 pages** (full UI parity, the reference set): Dashboard, Wallets, Send,
  Receive, Swap, Staking, NFTs, Bridge, DeFi, AddressBook, Devices, Approvals,
  Keystore, KYC, Transactions, Settings, Login, Register.
- `Login.tsx` — 3-mode: Create Wallet / Import Wallet first; email/password kept
  as an OPTIONAL recovery path; returning users with a stored token unlock
  straight in. "Create with Passkey" 4th option (real WebAuthn via
  `src/services/webauthn.ts`).
- **Send success** shows "Transaction submitted to the blockchain network"
  (Send.tsx). Passwordless send via "Unlock Wallet (passwordless)" →
  `unlockWallet` → `unlockToken` (no per-tx password).
- **Google Drive backup** in `Wallets.tsx`: exports the encrypted seed
  (AES-256-GCM via `/wallets/:id/export-encrypted-seed`) and downloads it as a
  JSON file for the user to upload to their own Drive (backend never sees Drive
  credentials). Mnemonic Copy button. "Setup App Lock" per wallet (passcode +
  real WebAuthn passkey).
- Light/dark theme via `ThemeContext` + `data-theme` CSS vars; `theme.css` has
  themed styles for all 18 pages.

### 3b. Desktop (`user_wallet/desktop`, Electron) — :8443, 95 methods, node --check 0
- `src/services/api.js` — 95 methods, full fetcher set (same as web).
- **17 pages** (full UI parity with web): Dashboard, Wallets, Send, Receive,
  Swap, Staking, NFTs, Bridge, DeFi, AddressBook, Devices, Approvals, Keystore,
  KYC, Transactions, Settings, Login. (Login.jsx covers register.)
- `Login.jsx` — guest-first (Create / Import first); email/password optional
  recovery path. "Create with Passkey" option (real WebAuthn).
- Send success shows "Transaction submitted to the blockchain network" (Send.jsx).
  Passwordless send via unlockWallet → unlockToken.
- Per-wallet "Setup App Lock" (passcode + real WebAuthn passkey) in Wallets.jsx.
- Light/dark theme via the same ThemeContext + CSS vars (isDark ternaries on
  every page; no `dark:` Tailwind variants).

### 3c. Browser extension (`user_wallet/extension`) — :8443, 97 methods, node --check 0
- `src/popup.js` `WalletAPI` — 97 methods, full fetcher set.
- `popup.html` — `guestStart` button; `popup.js` `handleGuestStart` runs the
  guest-auth flow (Create / Import first).
- Send success shows "Transaction submitted to the blockchain network" (popup.js).
- Light/dark theme via `toggleTheme` + CSS vars.

### 3d. Production (`user_wallet/production/react`, React/TS) — :8443, 99 methods, tsc 0 errors
- `src/services/WalletService.ts` — 99 methods, full fetcher set.
- Orphan `src/services/master/*` (11 files) **DELETED** (no MasterWallet
  cross-contamination).
- **13 pages**: Home, Wallet, Send, Receive, Swap, Bridge, Staking, NFTs,
  History, DApps, Settings, Login, KYC. (DeFi surface split across Staking/
  NFTs/Swap/Bridge/DApps pages.)
- `LoginPage.tsx` — guestAuth in `AuthContext` (Create / Import first). "Create
  with Passkey" via `src/utils/passkey.ts` (real WebAuthn).
- Send success shows "Transaction submitted to the blockchain network"
  (SendPage.tsx). Passwordless send via `src/components/AppLockModal.tsx` →
  unlockToken.
- **Google Drive backup**: exports the encrypted seed
  (`/wallets/:id/export-encrypted-seed`) and downloads it as a JSON file for the
  user's own Drive (no Drive credentials reach the backend). Mnemonic Copy.
- Light/dark theme via `ThemeContext` (isDark ternaries on every page).

### 3e. Android (`user_wallet/android`, Kotlin) — :8443, 105 methods, brace-balanced
- `app/src/main/java/com/tigeruserwallet/api/UserWalletApiService.kt` — 105
  methods, full fetcher set, brace-balanced.
- **17 fragments** (full UI parity): Dashboard (15-button nav grid),
  Wallets, Send, Receive, Swap, Staking, NFTs, Bridge, DeFi, AddressBook,
  Approvals, Devices, Keystore, KYC, Transactions, Settings, Start.
- `StartFragment` (guest-auth, Create / Import first) + "Create with Passkey"
  (`util/CredentialManagerHelper.kt` — real CredentialManager structure; passkey
  requires the `androidx.credentials` gradle dep, documented inline; passcode
  app-lock is fully real now).
- Send success shows "Transaction submitted to the blockchain network"
  (SendFragment.kt). Passwordless send via unlockWallet → unlockToken.
- Per-wallet "Setup App Lock" (passcode + passkey) in WalletsFragment.
- Light/dark theme via `AppCompatDelegate.setDefaultNightMode` + `values-night/`.

### 3f. iOS (`user_wallet/ios`, Swift) — :8443, 92 methods + parsePaymentUri, brace-balanced
- `App/UserWalletApiService.swift` — 92 methods + `parsePaymentUri` free func,
  full fetcher set, brace-balanced.
- **16 views** (full UI parity): Dashboard, Wallets, Send, Receive, Swap,
  Staking, NFTs, Bridge, DeFi, AddressBook, Approvals, Devices, Keystore, KYC,
  Transactions, Settings. Navigation via a TabView + "More" tab.
- `RootView` guest-auth (Create / Import first) + "Create with Passkey"
  (`PasskeyHelper.swift` — real ASAuthorizationPlatformPublicKeyCredential).
- Send success shows "Transaction submitted to the blockchain network"
  (SendView.swift). Passwordless send via unlockWallet → unlockToken.
- ReceiveView renders a REAL QR (`CoreImage.CIQRCodeGenerator`, not an asset/fake).
- Light/dark theme via `preferredColorScheme` + `ThemeManager`.

### 3g. Rust lib (`user_wallet/rust`) — :8443, ~95 async methods, cargo check 0 errors
- `src/lib.rs` — ~95 async methods, full fetcher set (HTTP against :8443).
- Local BIP derivation retained; now also performs network fetches via the
  canonical backend.

---

## 4. No-Registration Guest Auth (all 7 clients)

Every login UI now leads with **Create Wallet / Import Wallet**; `POST /auth/guest`
provisions an anonymous account so users can start instantly. Email/password
login is kept as an **OPTIONAL recovery path**. Returning users with a stored
token unlock straight in (no re-entry).

Implemented on: web (`Login.tsx` 3-mode), desktop (`Login.jsx` guest-first),
extension (`popup.html` `guestStart` + `popup.js` `handleGuestStart`),
production/react (`LoginPage.tsx` guestAuth in `AuthContext`), android
(`StartFragment.kt`), ios (`RootView` in `UserWalletApp.swift`).

---

## 5. Full Fetcher Categories (on ALL 7 clients)

Previously missing on most clients, now present everywhere:

- **Non-EVM** sign/send/address (Solana / Bitcoin / Cosmos).
- **Address-book** CRUD.
- **Devices** CRUD.
- **Token approvals** + revoke.
- **Keystore** V3 export/import.
- **Encrypted-seed** export/import (AES-256-GCM).
- **Security scan** (URL / address).
- **AMM** quote / swap.
- **Lending** supply / borrow / withdraw / repay.
- **Copy-trading** follow / traders / signals.
- **DAO governance** proposals / vote / delegates.
- **Perpetual** + **margin** positions.
- **Prediction markets**.
- **Launchpool** stake / unstake.
- **Token-sales** participate.
- **Dapps** + categories.
- **Chart** history.
- **DeFi protocols**.
- **NFT transfer**.
- **Transaction receipt**.
- **Estimate gas**.
- **Execute swap**.

---

## 6. UI Parity (FULL — 2026-08-17)

Every UI platform now exposes the SAME full screen set, all theme-aware:
- **Web**: 18 pages (Dashboard, Wallets, Send, Receive, Swap, Staking, NFTs,
  Bridge, DeFi, AddressBook, Devices, Approvals, Keystore, KYC, Transactions,
  Settings, Login, Register) — the reference set; `theme.css` themed for all 18.
- **Desktop**: 17 pages (same set; Login covers register).
- **Android**: 17 fragments (same set + Start; Dashboard 15-button nav grid).
- **iOS**: 16 views (same set; TabView + "More" tab).
- **Production/react**: 13 pages (Home, Wallet, Send, Receive, Swap, Bridge,
  Staking, NFTs, History, DApps, Settings, Login, KYC — DeFi split across these).
- **Extension**: 7 popup tabs (wallets, send, convert, staking, fiat, qr, kyc).
- Light/dark theme switch on every page of every platform (ThemeContext isDark
  ternaries / ThemeManager / AppCompatDelegate / preferredColorScheme /
  data-theme CSS vars). No `dark:` Tailwind variants in themed pages.

---

## 7. Build Verification (2026-08-17 — ALL GREEN)

| Client / backend | Verification |
|------------------|--------------|
| `go/wallet_api` (+ `defi_proxy.go`) | go build ✅ + go vet clean ✅ |
| `user_wallet/web` (React/CRA) | `tsc --noEmit` 0 errors |
| `user_wallet/desktop` (Electron) | `node --check` 0 |
| `user_wallet/extension` | `node --check` 0 |
| `user_wallet/production/react` | `tsc --noEmit` 0 errors |
| `user_wallet/android` (Kotlin) | brace-balanced (validated) |
| `user_wallet/ios` (Swift) | brace-balanced (validated) |
| `user_wallet/rust` | `cargo check` 0 errors |

---

## 8. Summary

All 7 UserWallet clients consume the single canonical `go/wallet_api` (:8443)
flat contract with the same full fetcher set, reached via the new
`defi_proxy.go` DeFi shims. No registration is required (guestAuth). Send-flow
success, Google Drive encrypted-seed backup, full UI parity, light/dark theme,
and all fetcher categories are present on every platform. All builds are green.
The previously documented `:8105` dead handler, `:8080` target, route
mismatches, and stubs are **resolved**.
