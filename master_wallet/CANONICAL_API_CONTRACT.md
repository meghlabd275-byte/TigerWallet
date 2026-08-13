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
