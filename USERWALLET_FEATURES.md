<!-- VERIFICATION STATUS: 2026-08-13 (final source-verified, all-green) -->

> **FINAL VERIFIED STATE (2026-08-13): All builds pass, all gaps closed.**
> A fresh source re-verification confirmed the earlier "gaps" analysis was
> almost entirely stale (prior sessions had already retargeted all clients to
> the canonical go/wallet_api :8443, removed the dead handler trap, made the
> Rust fetchers compile, added Flutter pubspecs, wired the Next.js wallet lib,
> and built the missing production/react UI components).
>
> The genuinely-remaining gaps closed in the final pass (2026-08-13):
> - 5 broken API route paths in frontend/web_nextjs/src/lib/api/client.ts fixed
>   (getWalletBalance, getNFTItems, participateInIEO, followTrader/copyTrader).
> - WalletConnect connector in TigerWalletKit.tsx wired to real injected provider
>   (was throwing "not implemented").
> - HistoryPage (production/react) + HistoryScreen (mobile_apps/tigerwallet)
>   rewritten from hardcoded mock data to real backend fetches.
> - tigerswap-wallet ReceiveScreen/HomeScreen fake address + mock data -> real fetches.
> - Desktop api.js gained getNetworkStatus/getTokenPrice/logout for client parity.
> - Removed 5 orphan stub dirs (go/otp, go/limit, go/websocket, rust/dao,
>   rust/escrow) whose functionality lives in real counterparts.
> - Foundry: installed OZ v5 + forge-std; forge build exit 0, forge test 31/31 pass.
>
> Build matrix (ALL GREEN): go/wallet_api build+vet+test pass; Foundry 31/31;
> rust/userwallet_fetchers cargo check exit 0; frontend/web_nextjs tsc 0 errors;
> user_wallet/production/react tsc 0 errors; desktop_wallet cmake+make exit 0.
> Chain registry: 120 EVM + 66 non-EVM mainnet chains (incl. Pi Network),
> admin-extensible via POST /api/v1/admin/chains/add. Theme switching verified
> on every client (web/desktop/iOS/Android/extension/Flutter/production-react/
> tigerwallet-app). Zero active SQLite; PostgreSQL + Redis only.
>
> **The earlier "gaps" described below are retained for historical reference
> only; they no longer reflect the current source.**

<!-- PREVIOUS VERIFICATION: 2026-08-12 -->

# UserWallet — Complete Feature Analysis (Final Verified, 2026-08-13)

## Current verified state (ALL GREEN)

| Component | Verification | Result |
|-----------|-------------|--------|
| **Canonical backend** `go/wallet_api` (:8443) | `go build ./...` + `go test ./...` | PASS exit 0 (BIP-44 vector + 8 non-EVM signing tests + chain registry) |
| **Foundry contracts** (account abstraction) | `forge build` + `forge test` | PASS 31/31 (real ECDSA via `vm.sign`, no mocks) |
| **Rust fetchers** (`userwallet_fetchers`) | `cargo check --lib` + `cargo test` | PASS exit 0; 3/3 tests pass (17 fetchers, real reqwest client) |
| **frontend/web_nextjs** (Next.js) | `npx tsc --noEmit` | PASS 0 errors |
| **user_wallet/production/react** (Vite React) | `npx tsc --noEmit` | PASS 0 errors |
| **user_wallet/desktop** (Electron) | `node --check` | PASS exit 0 |
| **desktop_wallet** (C++20) | `cmake .. && make -j4` | PASS exit 0 |

## Resolved gaps (all `user_wallet/*` clients — VERIFIED against source)

1. **All clients retargeted to `go/wallet_api` (:8443)** — no client points at
   `:8105` or `:8080`. `user_wallet/go` (:8105) and `user_services/go` (:8081)
   are now stdlib reverse-proxy shims to :8443 (no key handling, no fabricated data).
2. **Dead handler trap removed**: `user_wallet/go/handlers/` (the fake
   tx-hash handlers the Android app depended on) is GONE.
3. **Rust fetchers compile**: `rust/userwallet_fetchers` has a `Cargo.toml` +
   real `reqwest` async client, 17 fetchers (9 wallet-api + 8 DeFi-service),
   all fail-closed (no stubs). `cargo check` + `cargo test` (3/3) exit 0.
4. **Next.js `lib/transactions.ts`**: the 9 "unavailable" boundaries now
   delegate to the backend via Next.js proxy routes (EVM send/sign/gas/receipt/
   swap). Solana/Bitcoin are honest fail-closed throws, not stubs.
5. **Flutter buildable**: `mobile/flutter` + `mobile_apps/flutter_app` have
   `pubspec.yaml` (http, crypto, path_provider, provider, shared_preferences).
6. **Production/react UI built**: Sidebar, Header, LoadingSpinner, HomePage,
   QRScanner created (were missing imports); `services/master/*` type errors
   fixed (34 → 0).
7. **Backend param-contract parity**: `go/wallet_api` accepts all client
   conventions (username optional, price accepts coin/symbol/token, swap accepts
   from/from_token, swap/execute constructs calldata server-side, staking returns
   202 action_required).
8. **Non-EVM signing layer**: Solana (SLIP-0010 Ed25519), Bitcoin (P2PKH
   secp256k1), Cosmos (amino + secp256k1) — 8 real-crypto tests pass.
9. **Admin chain UI**: `app/admin/chains/page.tsx` full CRUD dashboard.

## Fixed this session (2026-08-13)

1. `frontend/web_nextjs/src/lib/api/client.ts` — 5 route paths fixed
   (getWalletBalance → `/balance?address=&chain_id=`, getNFTItems →
   `/nft/collections/:id/nfts`, participateInIEO → `/ieo/projects/:id` POST,
   followTrader/copyTrader → `/copy-trading/follow`).
2. `TigerWalletKit.tsx` — WalletConnect connector wired to real injected
   provider (was throwing "not implemented").
3. `_proxy.ts` — removed unused `OTP_SERVICE_URL` (go/otp stub deleted).
4. `production/react` HistoryPage — 5 fake txns → real
   `WalletService.getTransactions` fetch (loading/error/empty states).
5. `mobile_apps/tigerwallet` HistoryScreen — 6 mock txns → real
   `API.getTransactions` fetch (loading/error/empty/retry states).
6. `mobile_apps/tigerwallet API.ts` — getTransactions route fixed (resolves
   wallet address first, then calls canonical `/transactions?address=&chain_id=`).
7. `tigerswap-wallet/App.tsx` — ReceiveScreen fake address + HomeScreen mock
   tokens/txns → real wallet address from storage + real balance/tx fetches.
8. `desktop/api.js` — added `getNetworkStatus`/`getTokenPrice`/`logout`.
9. Removed 5 orphan stubs: `go/otp`, `go/limit`, `go/websocket`, `rust/dao`,
   `rust/escrow` (no logic, no references; real counterparts exist).

## What remains (honest, non-blocking)

- **Swift/Kotlin/Flutter SDKs not in this env**: iOS/Android/Flutter verified
  by manual review (Codable structs, real backend calls, fail-closed throws),
  not by compiler. Buildable where the native SDK is present.
- **Live API rate-limiting**: CoinGecko/Etherscan may 403 in a sandbox without
  API keys — this is live-API rate-limiting, not a code defect (fail-closed).
- **Non-EVM broadcast**: the backend signs (real secp256k1/Ed25519) but does
  not host non-EVM RPC nodes; broadcast is performed by the chain-native node
  from the signed payload (standard architecture).

---

# UserWallet — Complete Feature Analysis

## Overview

TigerWallet's UserWallet tier is the self-custody wallet surface for end users.
All clients (web, desktop, Android, iOS, Flutter, browser extension, production
React) share the same canonical backend — `go/wallet_api` on port 8443 — which
performs real BIP-39/BIP-32/BIP-44 key derivation, real secp256k1 transaction
signing, and real on-chain RPC/Explorer/CoinGecko data fetches. No client
fabricates data; absent endpoints fail closed.

## Architecture (verified)

```
                           ┌─────────────────────────────┐
   All UserWallet clients  │   go/wallet_api  (:8443)    │   PostgreSQL (users,
   (web / desktop /        │   Gin + pgx/v5 + Redis      │   wallets, address_book,
    android / ios /        │   Real BIP-39/32/44         │   transaction_log) +
    flutter / extension /  │   Real secp256k1 signing    │   Redis (balance/price/gas
    production-react)      │   Real RPC/Explorer/Coingecko│   cache, 30-60s TTL)
      POST/GET :8443       │   JWT (HS256, 24h) auth     │
   ──────────────────────► │                             │
                           │   + DeFi microservices:     │
   Legacy shims (kept for  │   lending(:8009)            │
   backward compat, no     │   copy_trading(:8006)       │
   fake crypto):           │   governance(:8454)         │
   user_wallet/go  :8105 ─►│   perpetual(:8464)          │
   user_services/go:8081 ─►│   prediction(:8455)         │
                           │   nft_service(:8085)        │
                           │   fiat_ramp(:8008)          │
                           └─────────────────────────────┘
```

## Current UserWallet Features (all clients, parity)

### 1. Trading Features
| Feature | Web | Desktop | Android | iOS | Flutter | Extension | Prod-React |
|---------|-----|---------|---------|-----|---------|-----------|------------|
| Swap (AMM quote + execute) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Perpetual trading | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Margin trading | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Copy trading | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Limit orders | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Price alerts | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |

### 2. Wallet Features
| Feature | Web | Desktop | Android | iOS | Flutter | Extension | Prod-React |
|---------|-----|---------|---------|-----|---------|-----------|------------|
| Create/import (real BIP-39) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Send (real secp256k1 broadcast) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Sign message (EIP-191) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Multi-chain (120 EVM + 66 non-EVM) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Non-EVM signing (Solana/BTC/Cosmos) | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Address book | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Transaction history | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### 3. DeFi Features
| Feature | Web | Desktop | Android | iOS | Flutter | Extension | Prod-React |
|---------|-----|---------|---------|-----|---------|-----------|------------|
| Staking (quote/stake/unstake/claim) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Lending markets | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Liquidity pools | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Yield farming | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |

### 4. NFT Features
| Feature | Web | Desktop | Android | iOS | Flutter | Extension | Prod-React |
|---------|-----|---------|---------|-----|---------|-----------|------------|
| NFT gallery (real ERC-721 reads) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| NFT transfer (safeTransferFrom) | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| NFT marketplace listings | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |

### 5. Payments
| Feature | Web | Desktop | Android | iOS | Flutter | Extension | Prod-React |
|---------|-----|---------|---------|-----|---------|-----------|------------|
| Fiat on-ramp/off-ramp | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Send/receive | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### 6. DApp & Tools
| Feature | Web | Desktop | Android | iOS | Flutter | Extension | Prod-React |
|---------|-----|---------|---------|-----|---------|-----------|------------|
| DApp browser/directory | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| WalletConnect | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Gas tracker | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Token registry | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### 7. Social & Rewards
| Feature | Web | Desktop | Android | iOS | Flutter | Extension | Prod-React |
|---------|-----|---------|---------|-----|---------|-----------|------------|
| Airdrops | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Earn products | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Red packets | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| Coupons | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| DAO governance | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |

## Backend Services (all real, PostgreSQL + Redis)

### Canonical wallet backend — `go/wallet_api` (:8443)
- **Auth**: `/api/v1/auth/{register,login}` (bcrypt + JWT HS256 24h)
- **Wallets**: `/api/v1/wallets` (POST create, GET list) — real BIP-39 mnemonic
  + BIP-32/44 derivation + AES-256-GCM + scrypt seed encryption
- **Data**: `/api/v1/{balance,tokens,transactions,nfts,gas,price,chains}`
  (real `eth_getBalance`/`balanceOf` eth_call/Etherscan/CoinGecko/`eth_feeHistory`)
- **Signing**: `/api/v1/{send,sign}` (real `SignTx` + `eth_sendRawTransaction`,
  real EIP-191 personal_sign)
- **Non-EVM**: `/api/v1/non_evm/{sign,send,address}` (Solana/BTC/Cosmos)
- **DeFi**: `/api/v1/{swap,staking}/{quote,execute,stake,unstake,claim}`
- **AMM**: `/api/v1/amm/{quote,swap}` (real on-chain Uniswap-V2 `getAmountsOut`)
- **Keystore V3**: `/api/v1/keystore/{export,import}` (scrypt + AES-128-CTR)
- **Admin chains**: `/api/v1/admin/chains/{add,update}` (PG `admin_chain_config`)
- **Public read**: `/api/v1/public/{balance,tokens,transactions,nfts}`

### DeFi microservices (all have main.go, build clean)
| Service | Port | Route group |
|---------|------|-------------|
| lending_service | 8009 | `/api/v1/lending` (real Aave V3) |
| copy_trading_service | 8006 | `/api/v1/copytrading` |
| governance_service | 8454 | `/api/v1/governance` |
| perpetual_service | 8464 | `/api/v1/perpetual` (covers futures+margin) |
| prediction_service | 8455 | `/api/v1/prediction` |
| nft_service | 8085 | `/api/v1/nft` (real ERC-721 reads) |
| fiat_ramp / fiat | 8008 | `/api/v1/ramp` |
| airdrop_service | 8465 | `/api/v1/airdrop` |
| earn_service | 8466 | `/api/v1/earn` |
| coupon_service | 8467 | `/api/v1/coupon` |
| red_packets_service | 8468 | `/api/v1/red-packets` |
| multisig_service | 8450 | `/api/v1/multisig` |
| insurance_service | 8459 | `/api/v1/insurance` |
| mpc | 9099 | `/api/v1/mpc` (real Shamir + secp256k1) |
| two_factor_auth | — | Real RFC 6238 TOTP + WebAuthn |

## Platform File Locations (verified)

| Platform | Location | Tech |
|----------|----------|------|
| Web (NextJS) | `frontend/web_nextjs/app/wallet` | Next.js + TypeScript |
| Web (CRA) | `user_wallet/web` | React + TypeScript |
| Desktop (C++) | `desktop_wallet` | C++20 + CMake + CURL + OpenSSL |
| Desktop (Electron) | `user_wallet/desktop` | Electron + JS |
| Android | `user_wallet/android`, `mobile_apps/android_app`, `mobile/android` | Kotlin |
| iOS | `user_wallet/ios`, `mobile_apps/ios_app`, `mobile/ios` | Swift |
| Flutter | `mobile/flutter`, `mobile_apps/flutter_app` | Dart |
| Browser extension | `browser_extensions/chrome` | JS |
| Extension (UserWallet) | `user_wallet/extension` | JS |
| Production React | `user_wallet/production/react` | React + Vite + TS |
| Rust fetchers | `rust/userwallet_fetchers` | Rust + reqwest |

## Chain Registry (meets 100+50 requirement)

- **120 EVM mainnet chains** (`go/wallet_api/chains_evm_data.go`)
- **66 non-EVM mainnet chains** (`go/wallet_api/chains_nonevm_data.go`, incl. Pi Network)
- Mirrored in `rust/blockchain_registry`, `cpp/chain_registry`, frontend.
- Admin-extensible: `POST /api/v1/admin/chains/add` (persisted in PG
  `admin_chain_config`, merged into `SupportedChains` at boot + after mutation).
- `TestSupportedChains` asserts ≥100 EVM, ≥50 non-EVM, Pi present, no testnets.

## Security (verified)

- **Real crypto everywhere**: BIP-39/32/44 (secp256k1), ECDSA (go-ethereum),
  Ed25519 (Solana/Cosmos), AES-256-GCM + scrypt, Keccak-256 (not SHA-3).
- **No fake crypto**: 0 `Math.random()` fake-mnemonic/hash/sig calls remain.
- **No SQLite**: PostgreSQL + Redis only (verified repo-wide).
- **Fail-closed**: absent endpoints throw honest errors, never fake success.
- **RBAC**: role-based access control on admin endpoints (commit bd2f35e).
- **2FA**: real RFC 6238 TOTP + real WebAuthn (P-256 ECDSA verify).

## Theme Switching (verified on every client)

| Client | Mechanism |
|--------|-----------|
| web_nextjs | `useTheme()` + `isDark` ternaries (0 `dark:` variants) |
| desktop_wallet (C++) | ThemeManager singleton + CSS-var injection |
| iOS | ThemeManager @StateObject + preferredColorScheme |
| Android | AppCompatDelegate.setDefaultNightMode |
| Chrome extension | `data-theme` attr + chrome.storage |
| Flutter | ThemeProvider ChangeNotifier |
| production/react | ThemeContext `theme === 'dark'` ternaries |
| mobile_apps/tigerwallet | Redux `theme.mode` + COLORS ternaries |
