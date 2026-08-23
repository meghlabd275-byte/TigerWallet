# UserWallet

The end-user wallet of TigerWallet: a **no-registration** multichain wallet
where the very first screen is *Create Wallet* or *Import Wallet* — never a
login/email wall. Backed by the canonical Go wallet API (`go/wallet_api`,
port **8443**) which performs real on-chain RPC, real BIP-39/32/44 HD
derivation, real secp256k1 signing + broadcast, and AES-256-GCM
encrypted-seed persistence (PostgreSQL + Redis). It is the **only** service
in the platform that performs key management and signing for users.

## Platforms

| Client | Location |
|---|---|
| Web (React/CRA) | `user_wallet/web` |
| Android | `user_wallet/android` |
| iOS | `user_wallet/ios` |
| Desktop (Tauri) | `desktop_app/` |
| Browser extension | `user_wallet/extension` |
| Rust core | `user_wallet/rust` |

All clients share the same canonical API: `go/wallet_api` on :8443. The
deprecated `user_wallet/go` reverse-proxy (:8105 → :8443) and the duplicate
`react_app`, `production/react`, and Electron `desktop` clients have been
removed; point all clients directly at :8443.

## No-Registration UX

`user_wallet/web/src/pages/Onboarding.tsx` — the no-registration landing page:

1. **First open** — the user sees exactly two choices: **Create Wallet** or
   **Import Wallet**. No login/register/email wall.
2. **Transparent guest identity** — behind the scenes the onboarding context
   provisions an ephemeral device identity via
   `POST /api/v1/auth/guest` (`go/wallet_api/handlers.go: handleGuestAuth`;
   optional `device_id`, empty → random) so the JWT-backed backend is
   satisfied, but the user only ever interacts with the wallet.

### Create with password, then back up

- Create: pick a label, a password (min 8 chars) and a home chain
  (Ethereum / BNB Chain / Polygon / Arbitrum / Optimism / Base are offered on
  the page; the backend serves 120 EVM + 66 non-EVM chains via
  `GET /api/v1/chains`) → `POST /api/v1/wallets` (`handleCreateWallet`).
- Backup: `BackupMnemonic.tsx` shows the 12/24-word recovery phrase **once**
  (the backend returns it only on create) and offers:
  1. **Copy to clipboard** (Web Clipboard API).
  2. **Google Drive backup** — real Google Drive API v3 upload via Google
     Identity Services + gapi; requires `REACT_APP_GOOGLE_CLIENT_ID`. If no
     client ID is configured the button is disabled with an honest message —
     never a fake success.
  3. **Download as encrypted file** — AES-GCM via WebCrypto with a
     password-derived key, as a real offline fallback.
  The user must confirm *"I've backed up my recovery phrase"* before
  proceeding; the mnemonic is then cleared from memory.

### Import

Import an existing wallet with a **12/24-word seed phrase** from the
Onboarding page (`importWallet`). The backend also supports
`POST /api/v1/wallets/import-encrypted-seed` (seed encrypted client-side with
the wallet password before transit) and
`POST /api/v1/keystore/import` for keystore imports. Wallets can later be
re-locked/unlocked (`/wallets/:id/lock`, `/wallets/:id/unlock`) and the
encrypted seed re-exported (`/wallets/:id/export-encrypted-seed`).

## Every Outgoing Transaction Shows "Transaction submitted to the blockchain network"

`user_wallet/web/src/components/TxSubmittedBanner.tsx` renders the banner
**"Transaction submitted to the blockchain network"** after every outgoing
transaction (send, swap, bridge, NFT transfer, auto-send, etc.) so the user
always has clear, immediate confirmation that the tx left the wallet.

## Automatic Sign + Approval Within a Second

UserWallet transactions are auto-approved and auto-signed by the
**MasterWallet auto-signer daemon** (`master_wallet/backend/auto_signer.go`):
pending user-initiated requests (transfers, swaps, stakes, NFT transfers,
message signing) are resolved end-to-end — approve → sign (EIP-1559
secp256k1 / Ed25519) → broadcast → websocket push — **within a second**.

Critical safety rules enforced there (and in `license_gate.go`):

- The MasterWallet can **never** withdraw user funds (`guardUserFunds`).
- `RevenuePayout`, `TreasuryTransfer`, `TreasurySweep`, `FeeWithdrawal` are
  **never** auto-approved — they require a two-party SuperAdmin co-sign.
- Per-rule velocity limits (`max_txs_per_hour`, `max_value_per_day`) counted
  against the real `auto_sign_log`; failures fail closed.

## Features

From the web pages and the canonical `go/wallet_api` surface:

- **Send / Receive** — real signing + broadcast on 120 EVM mainnets; non-EVM
  address derivation (Solana, Bitcoin, Cosmos, …).
- **Swap** — AMM routing / DEX aggregation.
- **Bridge** — full DeFi bridge surface proxied through :8443.
- **Staking** — staking positions and flows.
- **NFTs** — NFT gallery + transfers.
- **Claim** — token/airdrop claim pages.
- **DeFi** — DeFi protocol integrations and proxies.
- Approvals management, address book, devices, keystore, transactions history,
  settings, KYC status.

Web pages (`user_wallet/web/src/pages/`): Onboarding, Dashboard, Send,
Receive, Swap, Bridge, Staking, NFTs, DeFi, Transactions, Approvals,
AddressBook, Devices, Keystore, Wallets, Settings, KYC, Login, Register.

## Environment Variables

Frontend (`user_wallet/web`, CRA):

| Variable | Purpose |
|---|---|
| `REACT_APP_API_URL` | Canonical API base (default `http://localhost:8443/api/v1`) |
| `REACT_APP_GOOGLE_CLIENT_ID` | Google OAuth client ID for Drive backup (unset → Drive button disabled) |

Backend (`go/wallet_api`):

| Variable | Purpose |
|---|---|
| `PORT` / listen config | API port (default **8443**) |
| `JWT_SECRET` | JWT signing secret |
| Database env vars | PostgreSQL (encrypted seeds, users, tx log) |
| Redis env vars | Session/cache |
| `COINGECKO_API_KEY`, `ETHERSCAN_API_KEY` | Market/explorer data |
| `WALLET_API_URL` (shim only) | Upstream for the deprecated :8105 proxy |

## How to Run

```bash
# Canonical backend (requires PostgreSQL + Redis)
cd go/wallet_api
go run .                       # listens on :8443

# Web frontend
cd user_wallet/web
npm install
REACT_APP_API_URL=http://localhost:8443/api/v1 npm start

# Optional legacy shim (:8105 -> :8443)
cd user_wallet/go
WALLET_API_URL=http://localhost:8443 PORT=8105 go run .
```

See also: `master_wallet/README.md` (the auto-signer daemon this wallet
depends on) and `ADMIN_ARCHITECTURE.md` (approval/co-sign security model).
