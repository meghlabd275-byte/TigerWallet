# MASTER FEATURE MATRIX — 11 Competitors

Legend: **Yes / No / Partial / n.c.** (not confirmed) + one-word descriptor. *(Companion to `01-COMPETITOR-FEATURES.md` for details.)*

| Feature | Trust | MetaMask | Bitget | OKX | Phantom | Coinbase | Atomic | TokenPocket | KuCoin | CoinEx | Math |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **Type** | non-cust. multi-chain | non-cust. EVM+Sol | non-cust. multi | self-cust. multi | self-cust. Sol+EVM | dApp + smart (AA) | non-cust. no-KYC | DeFi multi | exch-integrated | exch-integrated | multi-chain |
| **Platforms** | iOS/And/Ext | Ext/iOS/And | iOS/And/Ext | App/Ext/Web | iOS/And/Ext | iOS/And/Ext | Desk/And/iOS/Ext | Mobile/Ext/HW/Card | App | iOS/And | Ext/Mobile |
| **Chains** | 100+ | EVM + Solana | 130+ | major multi | Sol + EVM | multi (+Base) | 1000+ coins | multi | multi | 55+/1M tokens | multi/100+ |
| **Hardware wallet** | Yes (Ledger/Trezor) (n.c.) | Yes (Ledger/Trezor…) (n.c.) | No (n.c.) | No (n.c.) | Yes (Ledger) | No (n.c.) | No | Yes (KeyPal) | No (n.c.) | No (n.c.) | No (n.c.) |
| **dApp browser** | Yes | Yes (ext) | Yes | Yes | Yes | Yes | No | Yes (Store) | Yes | Yes (Explorer) | Yes (DApp Store) |
| **WalletConnect** | Yes | Yes | Yes | Yes | Yes (n.c.) | Yes | Partial | Yes | Yes | Partial | Partial |
| **Injected EIP-1193** | Yes | **Yes (industry std)** | Yes (n.c.) | Yes | Solana-first | Yes | No | Yes | n.c. | n.c. | Yes |
| **Seed model** | BIP-39 | BIP-39 | BIP-39 | seed+pk+biometric | seed | seed (+MPC/passkey) | 12-word seed | BIP-39 | seed | seed (multi-wallet) | BIP-39/44 |
| **Swap/DEX aggregator** | Yes (1M pairs) | Yes (Swaps) | Yes (gasless n.c.) | Yes (100+ pools) | Yes | Yes (cross-net) | Yes (60+ pairs) | Yes (Transit) | Yes | Yes (aggregated) | Yes (MathSwap) |
| **Perps** | Yes (100×) | Yes (50×) | n.c. | n.c. | Yes | n.c. | No | n.c. | n.c. | No | No |
| **Limit orders** | Yes | Partial (n.c.) | n.c. | n.c. | n.c. | n.c. | No | n.c. | n.c. | n.c. | n.c. |
| **Staking/DeFi** | Yes | Yes (+RWA) | Yes | Yes (Earn) | Yes | Yes | Yes (5–20%) | Yes | Yes | Yes | Partial |
| **Fiat on-ramp** | Yes | Yes | Yes (Bitget) | Yes (exchange) | Yes | Yes (Apple Pay) | Yes (card) | Yes | Yes (exchange) | Yes (exchange) | Partial |
| **Debit/crypto card** | Partial (n.c.) | **Yes (MetaMask Card)** | No (n.c.) | No (n.c.) | **Yes (Visa)** | Partial (Coinbase Card*) | No | **Yes (TP Card)** | No (n.c.) | No | No |
| **NFT support** | Yes (600M+) | Partial (portfolio) | Partial (n.c.) | Yes (store) | Yes | Yes (gallery) | Yes (hold only) | Yes (manager) | Partial (n.c.) | Yes (mgmt) | Partial |
| **Custom tokens** | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| **Security scanner / scam alerts** | Yes (Security Scanner) | Partial | Yes (n.c.) | Yes | **Yes (instant)** | Yes | Partial | Yes (token security) | n.c. | n.c. | n.c. |
| **MEV / tx shielding** | n.c. | Partial | n.c. | n.c. | n.c. | n.c. | No | n.c. | n.c. | n.c. | n.c. |
| **Custom cloud backup** | Yes (Encrypted) | No | n.c. | n.c. | n.c. | n.c. | No | n.c. | n.c. | n.c. | n.c. |
| **Hardware-key / enclave backup** | n.c. | Partial | n.c. | n.c. | n.c. | n.c. | No | n.c. | n.c. | n.c. | n.c. |
| **SSO / extensibility** | Dev platform | **Snaps (flagship)** | Partial (n.c.) | Partial (SDK) | Partial | Yes (dev SDK) | No | Partial | No | No | Partial (DApp store) |
| **WaaS / MPC / embedded** | Yes (WaaS) | **Yes (Embedded/MPC)** | Partial (n.c.) | Yes (MPC/Node) | No | **Yes (Smart Wallet)** | No | Partial | No | No | Partial |
| **RWA / prediction mkts** | n.c. | Yes (RWA+mUSD) | n.c. | n.c. | Yes (pred mkts) | n.c. | No | n.c. | n.c. | n.c. | n.c. |
| **On-chain analytics terminal** | Partial | Partial | n.c. | **Yes** | Yes (Explore) | n.c. | No | n.c. | n.c. | Market data | Partial |
| **Unique differentiator** | Perps+limit+Cloud backup | Snaps+Embedded+RWA | Gasless+Launchpad | 1,000 sub-accts + terminal | Visa+card+cash+pred mkts | **AA/gasless smart wallet** | No-KYC desktop+cashback | Toolkit+KeyPal+card | Exchange + airdrops | Accelerator + CSC chain | MathAgentics AI + MathVerse |

---

## Consensus baseline: features essentially EVERY major competitor ships

These are the realistic "table-stakes" a competitor-grade wallet must have, all present in most of the 11:

1. **Real multi-chain HD wallet** (BIP-39/44, secp256k1 + chain-specific paths) — Ethereum + Solana + BTC + EVM L2s.
2. **dApp browser + WalletConnect + injected EIP-1193 provider** (browser ext + mobile).
3. **Hardware-wallet support** (Ledger/Trezor) — at least the 5 leaders.
4. **Swap / DEX aggregator** with best-price routing.
5. **Staking / DeFi yield.**
6. **Fiat on-ramp** and (increasingly) a **debit card**.
7. **NFT** view/transfer (and usually mint/marketplace).
8. **Custom token** import; **portfolio analytics**.
9. **Security**: scam/risk scanning, phishing protection, verified signing flow, optional cloud backup, biometric/face unlock.
10. **Multi-platform parity** — the same product on iOS/Android/extension/desktop actually *working*.

MetaMask/Coinbase/OKX add an **enterprise tier**: MPC/embedded/smart wallets & wallet-as-a-service — where the industry is heading.