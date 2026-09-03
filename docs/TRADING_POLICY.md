# TigerWallet Trading Policy & Built-in Architecture Specification

## 1. Executive Summary & Policy Guarantees

TigerWallet is an autonomous, self-contained multi-chain cryptocurrency ecosystem. All trading, swaps, derivatives, liquidity, copy trading, and options infrastructure are **built-in** directly into TigerWallet backends and client applications. 

### Core Guarantees:
1. **Zero External Dependency for Core Execution**: TigerWallet does not depend on third-party centralized brokers, centralized exchanges (CEX), or external custodial APIs to execute user trades, swaps, margin positions, futures contracts, options series, or copy-trading strategies.
2. **Seamless & Continuous Trading for Users**: All registered and guest users can continuously and seamlessly execute swaps, liquidity operations, margin trades, futures, perpetuals, and options without manual bootstrap steps. Feature flags are default-enabled with blacklist control semantics.
3. **Multi-Tier Administrative Governance**:
   - **SuperAdmin**: Global lifecycle control (`create`, `add`, `stop`, `resume`, `remove`, `halt`, `unhalt`) across all trading contracts, liquidity pools, trading pairs, margin markets, options series, and copy-trading engines across the entire platform.
   - **White-Label Client (Tenant)**: Tenant-isolated lifecycle management across their private user base and custom liquidity/trading products (`trading:control:<tenant_id>:*`).
   - **RBAC Admin**: Granular domain-scoped administrative management (`trading_control`, `listing_admin`, `liquidity_admin`) through dedicated admin dashboards.
   - **MasterWallet Enterprise Operator**: Enterprise liquidity and auto-signing treasury controls linked with multi-signature governance.
4. **66 Non-EVM Blockchain SDK Completeness**: Full native cryptographic derivation, signing, and keyless explorer integration across all 66 non-EVM chains (UTXO families, Cosmos families, Substrate, Solana, Algorand, Cardano, Aptos, Sui, TON, TRON, Near, Stellar, etc.) alongside 120 EVM chains.

---

## 2. Trading Vertical Lifecycle Control Matrix

| Vertical | Lifecycle Actions Supported | Governance Tiers | Enforcement Mechanism | Failure Semantics |
| :--- | :--- | :--- | :--- | :--- |
| **Spot Swap / AMM** | Create Pair, Stop Pair, Resume Pair, Remove Pair, Halt Spot | SuperAdmin, White-Label, RBAC Admin, MasterWallet | Redis `trading:control:*:pair:*` + Postgres `managed_trading_pairs` | Blacklist (Unmanaged pairs trade freely; outage fails open) |
| **Liquidity Pools** | Create Pool, Stop Pool, Resume Pool, Remove Pool, Halt Liquidity | SuperAdmin, White-Label, RBAC Admin, MasterWallet | Redis `trading:control:*:pool:*` + Postgres `liquidity_pools` | Blacklist; user position withdrawals/exits never blocked |
| **Futures & Perpetuals** | Add Contract, Stop Contract, Resume Contract, Remove Contract, Halt Perpetuals | SuperAdmin, White-Label, RBAC Admin, MasterWallet | Redis `trading:control:*:contract:*` + Postgres `trading_contracts` | Position creation gated; Position close ALWAYS allowed |
| **Margin Trading** | Add Market, Set Leverage, Stop Market, Resume Market, Remove Market, Halt Margin | SuperAdmin, White-Label, RBAC Admin, MasterWallet | Redis `trading:control:*:margin_market:*` + Postgres `margin_markets` | Borrowing/creation gated; Margin close/repay ALWAYS allowed |
| **Options Trading** | Create Series (Calls/Puts), Price (Black-Scholes), Stop Series, Resume Series, Halt Options | SuperAdmin, White-Label, RBAC Admin, MasterWallet | Redis `trading:control:*:options_series:*` + Postgres `options_series` | Position open gated; Exercise & close ALWAYS allowed |
| **Copy Trading** | Register Trader, Stop Trader, Resume Trader, Remove Config, Halt Copy | SuperAdmin, White-Label, RBAC Admin, MasterWallet | Redis `trading:control:*:copy_trader:*` + Reverse Proxy Guard | Strategy following gated; Unfollow/stop copying ALWAYS allowed |

---

## 3. Architecture & Service Breakdown

### 3.1 Backend Endpoints & Control Surface
- `go/wallet_api` (`:8443`): Core engine enforcing default-enabled blacklist checks on `/swap/quote`, `/swap/execute`, `/amm/swap`, `/perpetual/positions`, `/margin/positions`, `/options/positions`, `/copytrading/*`.
- `master_wallet/backend` (`:8450`): Enterprise control plane with PostgreSQL audit logging and multi-sig authorization hooks.
- `super_admin/go` (`:8082`): Global platform management radiating events across Redis channel and key structures.
- `white_label_admin/go` (`:9093`): Tenant-isolated administrative endpoints enforcing tenant UUID filters.
- `wl_user_wallet/go` (`:8461`): Independent self-hosted white-label wallet backend querying both SuperAdmin global stops and tenant-specific stops.

### 3.2 Frontend Interfaces
- `master_wallet/web`: Enterprise Trading Control-Plane (`TradingControlPage` under `/trading-control`).
- `super_admin/web`: Platform-wide trading management dashboard (`TradingControl.tsx`).
- `white_label_admin/web`: Tenant-scoped trading administration dashboard (`TradingControl.tsx`).
- `admin/web`: RBAC domain-scoped trading management dashboard (`TradingControl.tsx`).
- `desktop_app`, `user_wallet/web`, `user_wallet/extension`, `user_wallet/android`, `user_wallet/ios`, `user_wallet/flutter`: Full native client support for user swaps, liquidity, margin, perpetuals, options, and copy trading.

---

## 4. 66 Non-EVM Blockchain Management

All 66 non-EVM chains are natively supported with SDKs for key derivation, address encoding, and transaction construction without external SDK gaps:
- **UTXO Chains**: Bitcoin, Litecoin, Dogecoin, Dash, Bitcoin Cash, Zcash, Groestlcoin.
- **Cosmos SDK Chains**: Cosmos Hub, Osmosis, Injective, Sei, Cronos, Kava, Secret Network, Terra, Celestia, etc.
- **Substrate Chains**: Polkadot, Kusama, Substrate native SS58.
- **High-Throughput Modern L1s**: Solana (Ed25519), Aptos, Sui, Algorand (RFC 4648 Base32 + SHA-512/256), Cardano (BIP32-Ed25519 soft-derivation), TRON (Base58Check), Near, Stellar/Pi Network (Strkey SEP-0023), Ripple XRP, Tezos (tz1 Base58), Elrond/MultiversX (bech32 erd1), Kaspa (bech32 kaspa:), Nervos CKB, Filecoin, Nano.

---

## 5. Security & Isolation Safeguards

1. **Non-Custodial Integrity**: Admin tiers can configure parameters, market availability, leverage caps, and halt/resume operations, but **never** possess private keys to user funds or direct asset withdrawal capabilities.
2. **Auditability**: Every administrative action (`create`, `stop`, `resume`, `remove`, `halt`) creates an immutable record in `trading_control_audit` / `wl_trading_control_audit` storing the actor UUID, IP, action, entity, timestamp, and details.
3. **Fail-Safe Exit Guarantee**: Even during global or per-market emergency halts, position closure, debt repayment, collateral withdrawal, and fund release endpoints remain operational to ensure zero user-fund entrapment.
