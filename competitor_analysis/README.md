# TigerWallet — Competitive Analysis & Feature-Gap Audit

**Date:** 2026-08-09
**Scope:** TigerWallet (`main` branch, commit `20b001e`) vs 11 leading self-custody competitors.

| File | Contents |
|------|----------|
| [`01-COMPETITOR-FEATURES.md`](./01-COMPETITOR-FEATURES.md) | Full feature/functionality inventory of each of the 11 competing wallets (official-sourced). |
| [`02-COMPETITOR-MATRIX.md`](./02-COMPETITOR-MATRIX.md) | Master feature matrix comparing all 11 competitors side-by-side. |
| [`03-TIGERWALLET-AUDIT.md`](./03-TIGERWALLET-AUDIT.md) | **Honest** audit of TigerWallet: what is genuinely implemented (REAL) vs STUB / MOCK / PLACEHOLDER / FAKE across the whole repo, with code evidence. |
| [`04-GAP-ANALYSIS.md`](./04-GAP-ANALYSIS.md) | What is still missing / incomplete in TigerWallet relative to the competitors, and what is only partially implemented/degraded. |
| [`05-SECURITY-REVIEW.md`](./05-SECURITY-REVIEW.md) | Security-focused review — high-grade storage/signing audit, and where "real" security is NOT in the repo. |

---

## Progress implemented (2026-08-11)

Gaps closed and pushed to `main` (commit `f660821`, parent `8f5915f`):

1. **MIT `LICENSE`** added (maturity/trust gap #1; README already declared MIT).
2. **ERC-4337 smart-wallet contracts** (gap #3): real `TigerWalletAccount` (extends `BaseAccount`, ECDSA owner validation), `TigerWalletAccountFactory` (real CREATE2), `TigerWalletPaymaster` (extends `BasePaymaster`) under `account_abstraction/tigerwallet/`; OpenZeppelin vendored; `forge build` green; 5 Foundry tests pass. Supersedes the quarantined `legacy_aa/AccountFactory.sol`.
3. **`perpetuals_engine/rust`** (gap #13) now compiles (was 27 errors); 7/11 tests pass — see `04-GAP-ANALYSIS.md` for details on the 4 remaining test-assertion mismatches.
4. **Light/dark theme on all 83 `web_nextjs` pages** (UX gap): 32 hardcoded-dark pages now use the global `useTheme()` `isDark` conditionals; the MUI `admin_wallet` page wraps in a theme-synced MUI `ThemeProvider` with a header toggle; `npx tsc --noEmit` is 0 errors; `master_wallet` real crypto untouched.
5. **Removed committed ELF build artifacts** (~94 MB of compiled Go binaries) from git + `.gitignore` updated.
6. **Real on-chain NFT fetcher** (`go/nft_service/fetcher.go`): replaced in-memory mock NFT data with real `eth_call` ERC-721 reads (balanceOf/tokenOfOwnerByIndex/ownerOf/tokenURI/name/symbol/totalSupply) + IPFS metadata + Redis cache; 503 "unavailable" (never mock) when `ETH_RPC_URL` unset. `go build`+`go vet` clean.
7. **Web3 Secret Storage V3 keystore interop** (`go/wallet_api/keystore_v3.go`): real scrypt+AES-128-CTR+keccak256-MAC V3 keystore export/import (geth/MetaMask-compatible); 2 passing tests; REST routes `POST /api/v1/keystore/{export,import}` + Next.js proxy routes.
8. **Real curated dApp directory** (`go/wallet_api/dapp_directory.go`): ~20 real protocol entries (Uniswap, Aave, OpenSea, Curve, 1inch, Jupiter, Stargate, Lido, ENS, Lens, Farcaster, …) with categories/chains/verified flag — no fabricated metrics. Public REST `GET /api/v1/dapps` (+`/categories`, `/:id`) + Next.js proxy routes; both `dapp-store` and `dapp-browser` pages now `fetch('/api/v1/dapps')` (removed hardcoded `SAMPLE_DAPPS`/`POPULAR_DAPPS`); 3 backend tests pass.
9. **Token asset registry endpoint** (`GET /api/v1/tokens/registry`): the curated per-chain token list (real mainnet contract addresses for Ethereum/BSC/Polygon/Arbitrum/Optimism/Base) is now public — the Trust-Wallet-assets-repo equivalent; 404 for unknown chain (never fabricated); Next.js proxy route added.

Items 1, 2, 4, 5, 6, 7, 8 in the gap-analysis action list remain open.

---

## Executive summary (one page)

TigerWallet's repository is **very large (~2,858 tracked files; 47 Go modules, 88 Cargo crates, 11 CMake projects)** and its **`go/wallet_api/` Go backend is genuinely real** — real BIP-39, real BIP-32/44 HD derivation over secp256k1, real EIP-1559/191/712 signing, AES-256-GCM seed encryption, real `eth_sendRawTransaction`, PostgreSQL + Redis. The WalletConnect v2 relay (`go/walletconnect/`), a real Fiat-Shamir Schnorr ZK prover (`core/rust/zk_infrastructure/`), the Chrome browser extension, and the `master_wallet` Next.js mnemonic page are also genuinely real.

**However, the overwhelming majority of the 100+ marketing-feature directories are stubs, mocks, or fake implementations** — hand-rolled non-BIP-39 wordlists, XOR "encryption", SHA-256-as-KDF, P-256 used where secp256k1 is required, fabricated transaction hashes (`return "txhash", nil`), "In production…" placeholder bodies, and a main web wallet UI that posts to nonexistent endpoints and fabricates `0x…` random addresses.

**Bottom line.** The cryptographic plumbing is real and high-effort, but almost every *user-facing product feature* sold in the spec (swaps, staking, DeFi, copy-trading, MPC, hardware wallet, fiat on-ramp, mobile apps, smart-contract deployment) is still a stub. The single biggest gap versus every competitor is **a complete, real, end-to-end product flow in at least one frontend** — not more plumbing.