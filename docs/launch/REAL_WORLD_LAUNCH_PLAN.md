# TigerWallet Real-World Launch Plan

This plan converts the Trust Wallet / Bitget Wallet readiness gaps into launch gates. A gate must be green before TigerWallet handles unrestricted mainnet user funds.

## Phase 0 — hard blockers

1. **Remove mock wallet generation**: backend APIs must not return generated mnemonics or mock wallet addresses. Wallet creation must be client-side encrypted, MPC-backed, or smart-account backed with explicit custody rules.
2. **Replace placeholder cryptography**: BIP32/BIP39/BIP44, address derivation, scalar arithmetic, signing, transaction serialization, and signature verification require audited implementations.
3. **Replace mock market data**: chart, portfolio, history, price, token, and NFT data must come from configured providers with failover.
4. **NFT metadata pipeline**: support ERC-721, ERC-1155, Solana NFTs, Ordinals metadata, IPFS gateways, media proxying, spam hiding, and verified collections.
5. **Transaction safety**: every signing flow requires simulation, approval warnings, Permit2 analysis, scam domain checks, DApp risk scoring, and human-readable previews.
6. **CI/security baseline**: add secret scanning, dependency audits, SAST, smart-contract coverage, mobile/extension build checks, fuzzing for parsers/signers, SBOM, and release signing.

## Phase 1 — minimum viable mainnet wallet

- Secure wallet creation/import and encrypted backup.
- Send/receive on priority EVM chains plus one priority non-EVM chain.
- Real token balances, real prices, spam-token hiding, custom token support, and verified token lists.
- WalletConnect v2 and DApp browser with domain risk checks.
- Swap routes from verified providers with slippage and MEV warnings.
- NFT display with spam filtering and malicious metadata protection.
- Fiat on-ramp/off-ramp with at least two providers and clear KYC/regional restrictions.
- Crash reporting, privacy-safe analytics, user support flow, incident escalation.

## Phase 2 — competitive Trust Wallet / Bitget Wallet layer

- ERC-4337 smart accounts with session keys, key rotation, spending limits, multi-owner accounts, and account factories.
- Gasless transaction sponsorship with EIP-2771 trusted forwarders, relayer budgets, sponsor policies, and replay protection.
- Guardian social recovery with threshold approvals, timelocks, emergency contacts, cancellation, and anti-scam notifications.
- Intent-based cross-chain swap/bridge routing with solver/RFQ integrations and settlement tracking.
- MEV protection through private RPCs, Flashbots/MEV Blocker integrations, smart slippage, and order-flow protection.
- Institutional custody with MPC ceremonies, RBAC, audit logs, approval workflows, AML/KYC hooks, reporting, and sub-accounts.

## New code in this change

- `rust/security/src/account_abstraction.rs`: smart-account policy engine for ERC-4337 user-operation validation, multi-owner quorum, spending limits, session keys, and key rotation.
- `rust/security/src/gasless.rs`: gasless relayer policy engine for forward requests, sponsor budgets, caller/target allowlists, deadlines, signatures, and nonces.
- `rust/security/src/social_recovery.rs`: guardian recovery engine with threshold approvals, timelock execution, cancellation, and emergency contact storage.
- `rust/security/src/launch_readiness.rs`: launch gates that product, security, legal, infrastructure, and operations must satisfy before full mainnet release.
- `rust/security/src/transaction_preview.rs`: pre-signing transaction classification and DApp risk scoring for approvals, Permit2 flows, blocked domains, malicious contracts, and simulation-required decisions.
- `backend_services/api_gateway/main.go`: wallet creation now proxies to the configured wallet service, strips returned secrets, and refuses to generate mock wallets in the gateway.

## Mainnet release rule

TigerWallet should launch unrestricted mainnet access only when all `required_for_mainnet` gates in `launch_readiness::default_launch_gates()` are marked passed by real tests, audits, and operational sign-off.
