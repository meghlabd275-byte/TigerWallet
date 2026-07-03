# TigerWallet Production Readiness Gates

TigerWallet must not launch with synthetic success paths. Any subsystem that is not connected to a real provider must fail closed and return an explicit configuration error.

## Priority 0 gates

- Frontend production build passes.
- Smart contracts compile and tests pass.
- Workspace tests run without missing test runners.
- Production APIs do not return mock trade fills, synthetic swaps, synthetic liquidity positions, fake transaction hashes, fake signatures, or simulated confirmations.
- Wallet cryptography uses audited BIP39/BIP32/BIP44 implementations or validated test-vector-compatible implementations.

## Production adapters required before enabling endpoints

- Trading: configured CEX/DEX adapter with authentication, order submission, status polling, cancellation, idempotency keys, and audit logs.
- Swap/liquidity: configured router/aggregator with quote verification, calldata construction, gas estimation, slippage checks, and receipt tracking.
- Master wallet: configured signer, nonce manager, RPC broadcaster, receipt polling, reorg handling, and secure key custody.
- Hardware wallet: configured Ledger/Trezor/AirGap transport or signed raw transaction ingestion plus real broadcaster.

## Chain launch policy

Launch only chains that pass native transfer, token transfer, balance, history, fee estimation, signing, broadcasting, receipt tracking, reorg handling, and explorer-link tests. Do not advertise 100+ chains until every listed chain passes those gates.
