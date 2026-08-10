# go/services/wallet_service (DEPRECATED — removed)

This duplicate wallet service was removed because it fabricated transaction
hashes (`0x` + sha256 of timestamp) and derived addresses with sha256 instead
of secp256k1/keccak256 — a security issue. It had no `go.mod` and was not
referenced by any other module or the frontend.

## Where the features now live (real, no fakes)

All wallet functionality is provided by the **canonical** services:

- **go/wallet_api/** — the ONLY key-management + signing backend. Real
  BIP-39/BIP-32/BIP-44 HD derivation, secp256k1, keccak256 address derivation,
  EIP-1559/191/712 signing, real `eth_sendRawTransaction` broadcast,
  AES-256-GCM seed encryption with scrypt, PostgreSQL + Redis persistence.
  REST API on port 8443: `/api/v1/auth/*`, `/api/v1/wallets`, `/api/v1/send`,
  `/api/v1/{balance,tokens,transactions,nfts}`, `/api/v1/{gas,price,chains}`.
- **go/wallet_service/** — a transparent reverse-proxy shim that forwards
  to `go/wallet_api` for any legacy callers (stdlib only, no key management).

If you need multi-chain (Solana/Tron) send beyond EVM, extend `go/wallet_api`
rather than reintroducing fake-hash stubs here.
