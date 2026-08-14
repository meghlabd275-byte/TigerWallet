# MasterWallet Canonical API Contract (Backend :8450)

Base URL: `http://localhost:8450` (dev) / `https://master-api.tigerwallet.com` (prod)
Set via `MASTER_WALLET_API_URL` env var on clients.

All protected routes require `Authorization: Bearer <JWT>` header.

## Auth
- `POST /api/v1/auth/register` — `{email, password, name}` → `{token, user_id, email, role}`
- `POST /api/v1/auth/login` — `{email, password}` → `{token, user_id, email, role}`

## Master Wallets (auth required)
- `GET  /api/v1/master-wallet` → `{wallets: [MasterWallet]}`
- `POST /api/v1/master-wallet` — `{name, password, chain_id}` → `MasterWallet` (creates HD wallet, returns mnemonic once)
- `GET  /api/v1/master-wallet/:id` → `MasterWallet`
- `DELETE /api/v1/master-wallet/:id`
- `GET  /api/v1/master-wallet/:id/balance` → `BalanceResponse` (real RPC native + token balances)
- `POST /api/v1/master-wallet/:id/sign` — `{to, amount, password, token?}` → `{transaction_hash, status}` (real secp256k1 sign + broadcast)

## Sub Wallets
- `GET  /api/v1/master-wallet/:id/sub-wallets`
- `POST /api/v1/master-wallet/:id/sub-wallets` — `{name, password, chain_id}`
- `GET  /api/v1/master-wallet/:id/sub-wallets/:sid/balance`
- `POST /api/v1/master-wallet/:id/sub-wallets/:sid/transfer` — `{to, amount, password, token?}`

## Transactions
- `GET  /api/v1/master-wallet/:id/transactions` → `{transactions: [...]}`
- `POST /api/v1/master-wallet/:id/transactions` — `{to, amount, password, token?}`
- `POST /api/v1/master-wallet/:id/transactions/:tid/approve`
- `POST /api/v1/master-wallet/:id/transactions/:tid/reject`

## Policies
- `GET  /api/v1/master-wallet/:id/policies`
- `POST /api/v1/master-wallet/:id/policies` — `{rule_type, threshold, ...}`
- `PUT  /api/v1/master-wallet/:id/policies/:pid`
- `DELETE /api/v1/master-wallet/:id/policies/:pid`

## Fees
- `GET  /api/v1/master-wallet/:id/fees`
- `POST /api/v1/master-wallet/:id/fees`
- `DELETE /api/v1/master-wallet/:id/fees/:fid`

## Auto-Sign Rules
- `GET  /api/v1/master-wallet/:id/auto-sign`
- `POST /api/v1/master-wallet/:id/auto-sign`
- `DELETE /api/v1/master-wallet/:id/auto-sign/:rid`

## Users
- `GET  /api/v1/master-wallet/:id/users`
- `POST /api/v1/master-wallet/:id/users`
- `DELETE /api/v1/master-wallet/:id/users/:uid`

## Audit + Analytics
- `GET  /api/v1/master-wallet/:id/audit`
- `GET  /api/v1/master-wallet/:id/analytics/volume`
- `GET  /api/v1/master-wallet/:id/analytics/transactions`
- `GET  /api/v1/master-wallet/:id/analytics/wallets`

## Notifications + Webhooks
- `GET  /api/v1/master-wallet/:id/notifications`
- `POST /api/v1/master-wallet/:id/notifications`
- `GET  /api/v1/master-wallet/:id/webhooks`
- `POST /api/v1/master-wallet/:id/webhooks`
- `DELETE /api/v1/master-wallet/:id/webhooks/:wid`

## Treasury
- `GET  /api/v1/master-wallet/:id/treasury` → overview (real balances)
- `GET  /api/v1/master-wallet/:id/treasury/transactions`
- `POST /api/v1/master-wallet/:id/treasury/transfer` — `{to, amount, password}`
- `POST /api/v1/master-wallet/:id/treasury/sweep` — `{to, password}`

## Multisig
- `GET  /api/v1/master-wallet/:id/multisig/wallets`
- `POST /api/v1/master-wallet/:id/multisig/wallets` — `{name, owners, threshold}`
- `GET  /api/v1/master-wallet/:id/multisig/wallets/:wid/transactions`
- `POST /api/v1/master-wallet/:id/multisig/wallets/:wid/transactions`
- `POST /api/v1/master-wallet/:id/multisig/transactions/:tid/sign`
- `POST /api/v1/master-wallet/:id/multisig/transactions/:tid/execute`

## Public (no auth)
- `GET /api/v1/chains` → `{chains: [...]}`
- `GET /api/v1/gas?chain_id=N` → `{gas_price, max_fee, priority_fee}`
- `GET /api/v1/price?coin_id=ethereum` → `{usd, usd_24h_change}`
- `GET /api/v1/transactions/history?address=&chain_id=` → `{transactions: [...]}`
- `GET /health`

## WebSocket
- `GET /ws?master_wallet_id=&token=` — live balance updates, tx confirmations, market ticker

## UserWallet Management (MasterWallet → UserWallet governance)

The MasterWallet owner governs the UserWallet ecosystem. One master wallet owns
billions of UserWallet addresses. Users control their wallets via their 24-word
seed (BIP-39) — losing the seed means losing control. The master wallet
auto-signs and auto-approves ALL UserWallet transactions (send/claim/swap/trade).
SuperAdmin governs feature flags; the master wallet owner has full control of
enabled features.

### EVM Chain Management
- `GET    /api/v1/master-wallet/:id/user-chains/evm` → `{chains: [...]}` (120 default mainnet chains seeded on first boot)
- `POST   /api/v1/master-wallet/:id/user-chains/evm` — `{chain_id, name, symbol, rpc_url, explorer_url, decimals, derivation_path}`
- `PUT    /api/v1/master-wallet/:id/user-chains/evm/:chainId` — same body + `is_active`
- `DELETE /api/v1/master-wallet/:id/user-chains/evm/:chainId`

### Non-EVM Chain Management
- `GET    /api/v1/master-wallet/:id/user-chains/nonevm` → `{chains: [...]}` (66 default mainnet chains seeded on first boot — Bitcoin, Litecoin, Solana, Cosmos, Osmosis, Polkadot, Cardano, Aptos, Near, Sui, Aptos, Algorand, Ripple, Stellar, Elrond, Filecoin, etc.)
- `POST   /api/v1/master-wallet/:id/user-chains/nonevm` — `{chain_id, name, symbol, chain_type, rpc_url, explorer_url, decimals, derivation_path, address_prefix}`
- `PUT    /api/v1/master-wallet/:id/user-chains/nonevm/:chainId`
- `DELETE /api/v1/master-wallet/:id/user-chains/nonevm/:chainId`

### Token/Coin Management
- `GET    /api/v1/master-wallet/:id/user-tokens?chain_id=N` → `{tokens: [...]}`
- `POST   /api/v1/master-wallet/:id/user-tokens` — `{chain_id, contract_address, symbol, name, decimals, logo_uri, is_native}`
- `PUT    /api/v1/master-wallet/:id/user-tokens/:tokenId`
- `DELETE /api/v1/master-wallet/:id/user-tokens/:tokenId`

### UserWallet Address Derivation (24-word seed → any chain)
- `POST   /api/v1/master-wallet/:id/derive-user-address` — `{mnemonic, chain_id, chain_type, derivation_path, account_index}` → `{address, chain_type, chain_id, derivation_path, account_index}`
- `GET    /api/v1/master-wallet/:id/user-wallet-addresses` → `{addresses: [...], count: N}`

Real crypto: EVM via BIP-44 `m/44'/60'/...` secp256k1 + keccak256, Solana via
SLIP-0010 Ed25519, Bitcoin via secp256k1 P2PKH base58check, Cosmos via
secp256k1 + bech32. Seed hash stored only (never the seed).

### Auto-Sign (automatically sign + approve ALL UserWallet transactions)
- `POST   /api/v1/master-wallet/:id/auto-sign-transaction` — `{mnemonic, chain_id, chain_type, derivation_path, account_index, tx_type, to_address, value, token_address, contract_address, data}` → `{tx_hash, status, seed_hash, tx_type}`
- `GET    /api/v1/master-wallet/:id/auto-sign-logs` → `{logs: [...], count: N}`

Supports ALL chain types: EVM (real secp256k1 + eth_sendRawTransaction broadcast),
Solana (real SLIP-0010 Ed25519 over transfer message), Bitcoin (real P2PKH tx —
fetches UTXOs from blockstream.info, builds legacy SIGHASH_ALL tx, signs with
secp256k1, returns raw hex for broadcast), Cosmos (real secp256k1 SIGN_MODE_LEGACY_AMINO_JSON
over SignDoc). ERC-20 token transfers fetch real decimals from the contract.
tx_type: `send`, `claim`, `swap`, `trade`. Status: `signed`, `broadcast`, `confirmed`, `failed`.

### SuperAdmin Feature-Flag Governance
- `GET    /api/v1/master-wallet/:id/feature-flags` → `{feature_flags: [...]}`
- `POST   /api/v1/master-wallet/:id/feature-flags` — `{flag_key, flag_value, description, is_enabled}` (SuperAdmin adds features)
- `PUT    /api/v1/master-wallet/:id/feature-flags/:flagId` — master wallet owner updates (full control)
- `DELETE /api/v1/master-wallet/:id/feature-flags/:flagId`
