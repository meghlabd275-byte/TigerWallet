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

## Executive summary (one page)

TigerWallet's repository is **very large (~2,858 tracked files; 47 Go modules, 88 Cargo crates, 11 CMake projects)** and its **`go/wallet_api/` Go backend is genuinely real** — real BIP-39, real BIP-32/44 HD derivation over secp256k1, real EIP-1559/191/712 signing, AES-256-GCM seed encryption, real `eth_sendRawTransaction`, PostgreSQL + Redis. The WalletConnect v2 relay (`go/walletconnect/`), a real Fiat-Shamir Schnorr ZK prover (`core/rust/zk_infrastructure/`), the Chrome browser extension, and the `master_wallet` Next.js mnemonic page are also genuinely real.

**However, the overwhelming majority of the 100+ marketing-feature directories are stubs, mocks, or fake implementations** — hand-rolled non-BIP-39 wordlists, XOR "encryption", SHA-256-as-KDF, P-256 used where secp256k1 is required, fabricated transaction hashes (`return "txhash", nil`), "In production…" placeholder bodies, and a main web wallet UI that posts to nonexistent endpoints and fabricates `0x…` random addresses.

**Bottom line.** The cryptographic plumbing is real and high-effort, but almost every *user-facing product feature* sold in the spec (swaps, staking, DeFi, copy-trading, MPC, hardware wallet, fiat on-ramp, mobile apps, smart-contract deployment) is still a stub. The single biggest gap versus every competitor is **a complete, real, end-to-end product flow in at least one frontend** — not more plumbing.