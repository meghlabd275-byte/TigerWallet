# TigerWallet Verified Gap Matrix

## Scope and evidence standard

This matrix compares the current `main` branch implementation with capabilities documented by major wallet platforms. A feature is marked **implemented** only when source code exists, its contract is connected to a client or service, and an automated build or test can exercise it. Repository claims and README checkmarks are not evidence by themselves.

| Capability | Benchmark evidence | TigerWallet main-branch finding | Priority | Acceptance test |
|---|---|---|---|---|
| Shared wallet core | Trust Wallet documents Wallet Core and cross-platform SDKs; MetaMask documents common provider integrations [1] [2] | Three overlapping Rust wallet-core crates exist; canonical ownership and bindings are not established. The primary crates also declare missing benchmark targets. | Critical | One versioned core API builds and is consumed by web, Flutter, desktop, and extension adapters. |
| Secure key derivation and signing | Trust Wallet self-custody model; MetaMask smart/embedded wallet security [1] [2] | Rust security crate had a syntax error, missing dependencies, and CLI import/API errors; these are now corrected and its four tests pass. | Critical | Known-answer tests, malformed-input tests, zeroization review, and FFI boundary tests pass. |
| Chain support | Trust Wallet claims 100+ chains; Coinbase documents Solana and EVM compatibility [1] [3] | README claims broad support, but current core validation contains simplified/incorrect chain handling and many SDK directories are not connected to a unified signer/fetcher contract. | Critical | Each supported chain has address derivation, validation, transaction encoding, signing, broadcast, balance, token, and history integration tests. |
| dApp/provider connectivity | TrustConnect documents EVM, Solana, Bitcoin, browser extension, mobile WalletConnect, and deep links; MetaMask documents extension/mobile multichain and Solana [1] [2] | Browser and mobile provider surfaces are fragmented; no verified cross-platform provider contract was found in the initial audit. | Critical | EIP-1193, Solana Wallet Standard, and Bitcoin provider conformance suites pass on every client. |
| Smart accounts | Trust Wallet documents Barz ERC-4337; MetaMask documents smart accounts and permissions [1] [2] | Smart-contract files exist, but production wiring, recovery/guardian policy, passkey validation, and security verification are not established. | High | ERC-4337 user-operation tests, passkey tests, guardian recovery tests, and paymaster policy tests pass on a local test chain. |
| Market/portfolio data | Coinbase documents large asset coverage and dApp wallet integrations [3] | Fetcher code is distributed across many Go/Rust modules; cache and persistence contracts are not unified. | High | Provider failover, rate-limit, freshness, reconciliation, and Redis cache tests pass. |
| Backend persistence | Repository tech docs declare PostgreSQL and Redis, while some code/docs retain SQLite fallbacks [4] | SQLite remains in Cargo manifests, iOS pods, Android code/comments, and a Go fallback path. | Critical | No production dependency or runtime path references SQLite; PostgreSQL migrations and Redis integration tests pass. |
| Frontend parity and themes | Leading wallets provide mobile and extension/web experiences [1] [2] [3] | Multiple web/admin/mobile/desktop surfaces exist, but there is no single shared feature contract or proof of complete parity. | High | Contract-generated API client and parity checklist cover every page and client; light/dark persistence tests pass. |
| Build/release quality | Benchmark projects publish SDKs and documented integrations [1] [2] [3] | 48 Go modules and 99 Rust manifests create a large fragmented build surface; one package manifest was invalid and is now repaired. | Critical | CI builds the canonical modules, checks all package manifests, runs security scanning, and rejects unconnected modules. |

## Canonical architecture decision

The repository will converge on one Rust wallet-core crate for key derivation, address encoding, transaction serialization, and signing; one C++ performance layer only for measured hot paths with a stable C ABI; Go services for API, fetchers, queues, and distributed workloads; PostgreSQL as the system of record; Redis for cache, rate limits, sessions, and idempotency; and generated typed contracts consumed by every client.

No private-key operation may execute in JavaScript, Python, or Go. No client may implement chain-specific signing independently. Unsupported chains must return an explicit typed error rather than accepting an address or transaction using permissive fallback validation.

## Current verified build blockers

The baseline environment initially lacked Rust and Go toolchains. After installing current stable toolchains, the security crate exposed and then resolved a malformed signer declaration, missing `hex` and `clap` dependencies, an incorrect crate import, a missing nested error import, an Ed25519 parsing API mismatch, and an HMAC trait ambiguity. The security crate now passes all four library tests and builds its CLI test target.

The primary wallet-core crates still require consolidation: they contain overlapping manifests, missing benchmark targets, and unverified chain implementations. These are the next implementation gate before broad client parity work.

## References

[1]: https://developer.trustwallet.com/developer "Trust Wallet official developer documentation"
[2]: https://docs.metamask.io/ "MetaMask official developer documentation"
[3]: https://www.coinbase.com/developer-platform/products/wallet-sdk "Coinbase Wallet official SDK documentation"
[4]: https://github.com/meghlabd275-byte/TigerWallet "TigerWallet main branch repository"
