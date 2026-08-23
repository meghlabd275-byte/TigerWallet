# MasterWallet

The **MasterWallet** is TigerWallet's operator-grade wallet: one canonical
backend that owns treasury, chain/token management, policies, fees, multisig,
and the **auto-approve / auto-sign daemon** that gives every UserWallet
automatic approval and signing within a second — while being structurally
unable to move user funds or withdraw fees/revenue without TigerWallet
SuperAdmin collaboration.

## Platforms

| Client | Location |
|---|---|
| Canonical Go backend (:8450) | `master_wallet/backend` |
| Web console | `master_wallet/web` |
| Android | `master_wallet/android` |
| iOS | `master_wallet/ios` |
| Desktop | `master_wallet/desktop` |
| Browser extensions | `master_wallet/extensions` |
| Flutter | `master_wallet/flutter` |
| Rust core | `master_wallet/rust` |

All clients talk to the same canonical Go backend — there is one API surface,
defined in `CANONICAL_API_CONTRACT.md`.

## Canonical Go Backend (`backend/`, :8450)

A single canonical Gin server that replaces the old redundant backends. Real
crypto, PostgreSQL + Redis persistence, live chain fetchers, treasury,
multisig, WebSocket push. No stubs/fakes. Boots in degraded mode (with a
warning) if PostgreSQL/Redis are unreachable.

Key source files:

| File | Responsibility |
|---|---|
| `main.go` | Entrypoint, route registration, starts market ticker + auto-sign daemon |
| `config.go` | Env-driven config (`MASTER_WALLET_*` vars) |
| `handlers.go` | Core API handlers |
| `auto_signer.go` | Auto-approve/auto-sign daemon + tx classifier + user-funds guard |
| `crypto_core.go` | Real BIP-39/32/44, secp256k1 + keccak256, scrypt + AES-256-GCM seed encryption |
| `non_evm_crypto.go` | Non-EVM signers (e.g. SLIP-10 Ed25519 for Solana, BTC P2PKH, Cosmos) |
| `chains.go` + `chain_registry_data.go` | Curated mainnet chain config (RPC from env, fail-closed) |
| `management.go` | Policies, fees, auto-sign rules, users, whitelist, analytics, audit, notifications, webhooks, API keys |
| `treasury.go` | Treasury overview/transfers/sweeps/allocations — real balances, real broadcast |
| `multisig.go` | Threshold multisig: create, collect owner ECDSA signatures, execute at threshold |
| `user_wallet_management.go` | MasterWallet → UserWallet governance (chains, coins, derivation, auto-sign, fees, feature flags) |
| `license_gate.go` | Two-party SuperAdmin co-sign gate at the broadcast boundary |
| `fetchers.go` | Live chain/market fetchers |
| `websocket.go` | Real-time event hub |

## Auto-Approve / Auto-Sign Daemon (`auto_signer.go`)

Requirement (product owner):

1. *"UserWallet always gets automatic sign and automatic approval within a
   second from SuperAdmin or MasterWallet owner or Admin from admin panel."*
2. *"MasterWallet CANNOT withdraw users' funds of any UserWallet."*
3. *"MasterWallet owner cannot withdraw any fees or revenue without
   TigerWallet SuperAdmin permission."*

Design:

- A background goroutine polls the real `transactions` table every
  `MASTER_AUTO_SIGN_POLL_MS` (default **100 ms**) for pending, user-initiated
  approval requests and resolves them end-to-end: approve (real rows in
  `transaction_signatures` + `approval_requests`) → sign (EIP-1559 via
  `SignEVMTransaction`; non-EVM via `non_evm_crypto.go` signers) → broadcast
  (`eth_sendRawTransaction`) → websocket event so UIs update **within a
  second**.
- **Classifier** (Go mirror of `wl_control_plane/rust/src/classifier.rs`):
  `UserTransfer`, `Swap`, `Stake`, `NftTransfer`, `PersonalSign`,
  `TypedDataSign` are **auto-approvable**.
  `RevenuePayout`, `TreasuryTransfer`, `TreasurySweep`, `FeeWithdrawal` are
  **NEVER auto-approved** — they stay pending for the two-party SuperAdmin
  co-sign path (`license_gate.go`).
- **Velocity limits:** `checkAutoSignRules` enforces per-rule
  `max_txs_per_hour` / `max_value_per_day` from the rule `conditions` JSONB,
  counted against the real `auto_sign_log` in PostgreSQL. Exhausted rules
  fall through; query errors fail closed. Rules are managed via the auto-sign
  CRUD API — no schema change needed.

### User-Funds Guard

`guardUserFunds` is the critical security invariant: the daemon refuses to
sign anything that moves funds **out** of a user sub-wallet to a destination
not belonging to that same user. The MasterWallet can never pull user funds.
Fail-closed: on any doubt the transaction stays pending for manual review.

## Two-Party SuperAdmin Co-Sign (`license_gate.go`)

*"No one can withdraw any fund or revenue without TigerWallet SuperAdmin
collaboration."* Enforced at the **broadcast boundary** — the last point
before funds move — so even a compromised WL admin key cannot move funds
alone. Fail-closed: if the control plane URL is unset or unreachable, the
withdrawal is refused.

## Chain Management

- The full canonical registry is **120 EVM mainnets**
  (`go/wallet_api/chains_evm_data.go`, `evmMainnetCount = 120`) plus **66
  non-EVM chains** (`go/wallet_api/chains_nonevm_data.go`), served to every
  client via `GET /api/v1/chains`.
- `chains.go` holds the curated, mainnet-only subset the MasterWallet
  treasury/operator wallet needs. Each chain maps to a BIP-44 coin type +
  derivation path; RPC endpoints resolve from env vars at runtime and fail
  closed when unset (no fabricated endpoints).
- `user_wallet_management.go` lets the MasterWallet owner add/remove/update
  EVM + non-EVM chains and coins/tokens available to UserWallet users, and
  derive UserWallet addresses from a user's seed for any chain (EVM BIP-44
  `m/44'/60'/...`, Solana SLIP-10 Ed25519, Bitcoin P2PKH, Cosmos bech32).

## Fees, Policies, Treasury, Multisig, Passkeys

- **Fees** — UserWallet transaction fee management in `management.go` /
  `user_wallet_management.go`; fee balances accrue on-chain and any
  withdrawal is a `FeeWithdrawal` → never auto-approved, co-sign required.
- **Policies** — named policy records (type, conditions, actions, priority)
  persisted in PostgreSQL and enforced by the auto-sign path.
- **Treasury** — real on-chain balance queries + real broadcast for
  transfers/sweeps/allocations. The treasury hot-wallet key loads from env;
  when unset, write endpoints return 503 (fail-closed) instead of
  fabricating.
- **Multisig** — on-chain-style threshold wallets: create (threshold +
  owners), collect owner secp256k1 ECDSA signatures, execute once the
  threshold is gathered. Real `go-ethereum/crypto` verification.
- **Passkeys** — WebAuthn credential management:
  `POST /:id/passkey/register`, `GET /:id/passkey/credentials`,
  `DELETE /:id/passkey/credentials/:credId`,
  `POST /:id/passkey/verify-assertion`. Public keys (SPKI) are stored in a
  dedicated relying-party table (`store.go`).

## Environment Variables

| Variable | Purpose |
|---|---|
| `MASTER_WALLET_PORT` | Listen port (default `8450`) |
| `MASTER_WALLET_JWT_SECRET` | JWT signing secret |
| `MASTER_WALLET_DATABASE_URL` / DB vars | PostgreSQL |
| `MASTER_WALLET_REDIS_PASSWORD` | Redis auth |
| `MASTER_WALLET_TREASURY_KEY_HEX` | Treasury hot-wallet key (unset → treasury writes 503) |
| `MASTER_AUTO_SIGN_POLL_MS` | Auto-sign poll interval (default 100 ms) |
| `MASTER_WALLET_BUNDLER_URL` / `MASTER_WALLET_PAYMASTER_URL` | Account-abstraction bundler/paymaster |
| `COINGECKO_API_KEY`, `ETHERSCAN_API_KEY` | Market/explorer data |
| `ETH_RPC_URL`, `BSC_RPC_URL`, `POLYGON_RPC_URL`, `ARBITRUM_RPC_URL`, `OPTIMISM_RPC_URL`, `AVALANCHE_RPC_URL`, `BASE_RPC_URL`, … | Per-chain RPC endpoints (fail-closed when unset) |

## How to Run

```bash
# From source (requires PostgreSQL + Redis)
cd master_wallet/backend
go run .                 # listens on :8450

# Docker (backend has its own Dockerfile)
cd master_wallet/backend
docker build -t masterwallet-backend .
docker run -p 8450:8450 --env-file .env masterwallet-backend

# Or via the platform compose (service: master-wallet-backend)
cd ../.. && docker compose up master-wallet-backend
```

See `BUILD_AND_DEPLOY.md` for full deployment, and
`CANONICAL_API_CONTRACT.md` for the API surface every client implements.
