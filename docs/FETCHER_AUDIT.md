# TigerWallet — Fetcher Master Audit (Phase 36)

> Verified 2026-08-29 against the working tree (HEAD c2b645c7). Statuses use the
> Phase-56 vocabulary. A fetcher is only **COMPLETE** when its full runtime path
> (trigger → fetch → normalize → persist/serve → consumer) was traced to real
> network I/O — never merely because the function exists.

## Method

- Enumerated every `Fetch*` / `Fetcher` implementation in the Go/Rust/Python tree.
- Traced each to its provider (EVM JSON-RPC, explorer REST, price API, chain node).
- Verified buildability with Go 1.22.12 (`go build ./...` + `go vet ./...` = 0 for
  every canonical module listed).
- Consumers traced via route registration in each service `main.go`.

---

## 1. Canonical EVM fetch path — `go/wallet_api/fetchers.go` (UserWallet, :8443)

| Fetcher | Purpose | Provider | Auth | Chains | Output | Consumers | Status |
|---|---|---|---|---|---|---|---|
| `FetchNativeBalance` | Native coin balance | EVM JSON-RPC `eth_getBalance` | none (public RPC) | all 120 EVM | `*big.Int` wei | `/balance` route, android/web/desktop/extension | COMPLETE |
| `FetchTransactionCount` | Nonce | `eth_getTransactionCount` | none | all EVM | uint64 | `/send` nonce mgmt | COMPLETE |
| `FetchGasPrice` | Legacy + EIP-1559 fees | `eth_gasPrice`, `eth_maxPriorityFeePerGas` | none | all EVM | gasPrice/maxFee/prioFee | `/send`, `/simulate`, gas UI | COMPLETE |
| `FetchChainID` | Chain verification | `eth_chainId` | none | all EVM | chain id | chain registry validation | COMPLETE |
| `FetchERC20Balance` | Token balance | `eth_call` `balanceOf` | none | all EVM | `*big.Int` | `/balance`, token list | COMPLETE |
| `FetchERC20Metadata` | symbol/name/decimals | `eth_call` (3 selectors) | none | all EVM | metadata | token import, display | COMPLETE |
| `FetchTokenBalances` | Batch token balances | `eth_call` loop | none | all EVM | `[]TokenBalance` | portfolio | COMPLETE |
| `FetchTokenPrice` | USD price | CoinGecko REST | none (public) | coin-id mapped | price/24h change | portfolio valuation, swap quote | COMPLETE |
| `FetchETHPrice` | ETH USD reference | CoinGecko REST | none | ETH | float64 | gas USD conversion | COMPLETE |
| `FetchTransactionHistory` | Tx history | Explorer REST (Etherscan-compatible) | `apiKey` param (per-chain env) | explorer-supported EVM | `[]TransactionHistory` | `/transactions` | COMPLETE |
| `FetchNFTAssets` | NFT holdings | Explorer REST `tokennfttx` | `apiKey` param | explorer-supported EVM | `[]NFTAsset` | `/public/nfts`, NFT page | COMPLETE |
| `FetchOHLC` / `FetchMarketChart` (`chart_fetchers.go`) | Candle data | CoinGecko REST | none | coin-id mapped | `[]OHLCPoint` | trading charts | COMPLETE |

Retry/timeout: HTTP clients carry explicit timeouts; RPC dials are context-bound.
Caching: Redis cache layer in front of hot reads (`store.go`). Fallback: per-chain
RPC list from the chain registry (120 EVM + 66 non-EVM seeded).

## 2. MasterWallet fetch path — `master_wallet/backend/fetchers.go` (:8450)

Same shape as §1 (9 `Fetch*` funcs: native balance, nonce, gas, chain id, ERC-20
balance/metadata/batch, CoinGecko price, explorer tx history) but consumed only by
MasterWallet routes under `/api/v1/master-wallet/:id/...` behind `AuthMiddleware`.
Treasury/revenue ops additionally require SuperAdmin two-party co-sign
(`license_gate.go`). **Status: COMPLETE.** Kept as a separate copy from
`go/wallet_api` by design (Phase 11 — MasterWallet must not import UserWallet
internals; separate deployable).

## 3. Non-EVM signing/broadcast — `go/wallet_api/non_evm_signing.go`,
##    `master_wallet/backend/{btc_helpers,solana_broadcast}.go`

| Fetcher | Chain family | Mechanism | Status |
|---|---|---|---|
| BTC helpers | Bitcoin/UTXO | real base58/bech32, tx build, JSON-RPC broadcast | COMPLETE |
| Solana broadcast | Solana | Ed25519 sign, message build, JSON-RPC broadcast | COMPLETE |
| non-EVM signing | 3 mainnet families | real key derivation + sign, no testnet fakes | COMPLETE |

## 4. Extended fetch scaffold — `go/full_fetchers/fetchers.go` (19 fetchers, 128 funcs)

Rewritten Session 7 from 100% no-op scaffold to **real EVM JSON-RPC + public price
APIs** via stdlib-only `rpc.go` (`rpcCall`, `eth_blockNumber`, `eth_chainId`).
Zero in-repo importers — it is a standalone reference/library module; the canonical
live path remains `go/wallet_api/fetchers.go`. **Status: COMPLETE (as a library),
UNUSED (by canonical services) — kept intentionally, not duplicated live state.**

## 5. Domain fetcher services (`go/*`)

| Service | Fetcher | Provider | Status |
|---|---|---|---|
| `go/nft_service/fetcher.go` | ERC-721/1155 ownership enumeration | EVM `eth_call` (own keccak, no geth dep) | COMPLETE |
| `go/gas_oracle/main.go` | Multi-chain gas price oracle + USD conversion + cost estimate | per-chain RPC + price feed | COMPLETE |
| `go/fiat_ramp` | Stripe/MoonPay/Transak ramps | HMAC-verified webhooks (`webhooks.go`) | COMPLETE |
| `ai_agent` (rust+py) | Gas price, tx analysis | real `eth_gasPrice` via `EVM_RPC_URL` | COMPLETE (11 tests) |
| `ai_layer` (rust+py) | Price prediction inputs | market data ingestion | COMPLETE (3 tests) |
| `cex_connectors/connectors.rs` | CEX market data/execution | per-exchange REST/WS, env creds | PARTIAL — needs live-exchange credential test |
| `dex_connectors/{base/dex.go,top_20/connectors.rs}` | DEX quotes/routes | on-chain router calls | PARTIAL — route coverage unverified end-to-end |
| `price_oracle/main.go` | Price aggregation | multi-source | PARTIAL — provider failover unverified |

## 6. Fetchers NOT verifiable in this sandbox

| Item | Reason |
|---|---|
| `fetcher_core/rust`, `fetcher_gateway/rust` | No Rust toolchain in this sandbox (Session 6 note). Code present; build unverified. |
| Smart-contract event indexers (`smart_contracts/` 105 sol) | Needs Solidity toolchain + auditor (Phase 42, open). |
| Hardware-wallet fetch paths | Needs physical device test. |

## 7. Summary counts

- COMPLETE (traced to real I/O): 14 fetcher groups across wallet_api, master_wallet,
  non-EVM, nft_service, gas_oracle, fiat_ramp, ai_agent, ai_layer, full_fetchers.
- PARTIAL (real code, live-provider verification pending): cex_connectors,
  dex_connectors, price_oracle.
- NOT VERIFIABLE here: fetcher_core/gateway (Rust), contract indexers, hardware wallet.
- FAKE / hardcoded-data fetchers in production paths: **0 found** (Session 8 mock
  purge removed the last fabricated-data consumers; Session 9 removed the PagerDuty
  placeholder policy ID and the near-constant request-ID generator).
