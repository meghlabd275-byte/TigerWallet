# TigerWallet — Agent Memory

## ERC-4337 / Account Abstraction contracts

- The canonical, audited ERC-4337 reference implementation (eth-infinitism,
  `develop` branch, audited by OpenZeppelin) lives in
  `smart_contracts/evm_contracts/account_abstraction/` and is the ONLY
  EntryPoint used by the project.
  - Root: `EntryPoint.sol`, `EntryPointSimulations.sol`, `SenderCreator.sol`,
    `StakeManager.sol`, `NonceManager.sol`, `UserOperationLib.sol`,
    `Helpers.sol`, `Eip7702Support.sol`, `Stakeable.sol`, `BaseAccount.sol`,
    `BasePaymaster.sol`.
  - `interfaces/`: `IEntryPoint.sol`, `IAccount.sol`, `IAccountExecute.sol`,
    `IAggregator.sol`, `IPaymaster.sol`, `INonceManager.sol`,
    `IStakeManager.sol`, `ISenderCreator.sol`, `IEntryPointSimulations.sol`,
    `PackedUserOperation.sol`.
  - `utils/`: `Exec.sol`.
  - Uses `PackedUserOperation` (packed fields), NOT the legacy unpacked
    `UserOperation` struct. Pragma `^0.8.28`, EVM `cancun`.
- The old duplicate custom contracts under `account_abstraction/smart_contracts/`
  were removed; that dir now only has a `README.md` pointing to the canonical
  location. Do NOT re-add custom EntryPoint/SmartAccount/Paymaster there.
- TigerWallet's pre-existing custom `AccountFactory.sol` (which used the old
  unpacked `UserOperation` + an on-chain `Bundler`) was relocated to
  `smart_contracts/evm_contracts/legacy_aa/` because it does not compile against
  the canonical packed types. It must be ported to `PackedUserOperation` +
  `UserOperationLib` and extend `BaseAccount`/`BasePaymaster` before reuse.

## Foundry setup

- `smart_contracts/evm_contracts/foundry.toml`: `src = "account_abstraction"`,
  `solc_version = "0.8.28"`, `evm_version = "cancun"`, `libs = ["lib"]`.
  `test`/`script` are pointed at nonexistent `test_aa`/`script_aa` so `forge
  build` only compiles the canonical AA source root and does NOT pick up the
  legacy Hardhat TigerSwap DEX tree (`contracts/`, `interfaces/`, `test/`),
  whose interfaces collide with OpenZeppelin's `IERC20`.
- OpenZeppelin contracts v5.7.0 installed at
  `smart_contracts/evm_contracts/lib/openzeppelin-contracts` (via
  `forge install OpenZeppelin/openzeppelin-contracts --no-git`). Auto-remapping
  `@openzeppelin/contracts/=lib/openzeppelin-contracts/contracts/`.
- Forge binary: `~/.foundry/bin/forge` (v1.7.1). Build:
  `cd smart_contracts/evm_contracts && forge build` → "Compiler run successful!"
- `smart_contracts/evm_contracts/contracts/AccountAbstraction.sol` and
  `AccountAbstraction_Upgraded.sol` only use an `onlyEntryPoint`
  (`msg.sender == entryPoint`) pattern and do NOT depend on the `UserOperation`
  struct, so they remain compatible with the canonical EntryPoint (they are in
  the legacy Hardhat tree, not compiled by this Foundry project).
- A `hardhat.config.ts` also exists at `smart_contracts/evm_contracts/`; the
  repo historically used Hardhat, the Foundry project was added for the
  canonical AA stack.

## dapp_browser/go (WalletConnect service)

- Module: `tigerwallet/dapp_browser` (go.mod created in `dapp_browser/go/`).
- `walletconnect.go`: `handlePersonalSign` / `handleEthSignTypedData_v4` now do
  REAL signing via `github.com/ethereum/go-ethereum/crypto` (ECDSA secp256k1).
  NEVER return a fake all-zero `0x0000...` signature — if no signer key is
  configured, reject the request with JSON-RPC error `-32000`
  ("Signing not available: wallet not connected").
- The signer ECDSA private key is loaded from env `SIGNER_PRIVATE_KEY` (hex,
  optional) into `WalletConnectService.signer`. Personal_sign prefixes with
  keccak256("\x19Ethereum Signed Message:\n" + len + msg); typed data uses
  `apitypes.TypedDataAndHash` (EIP-712).
- gorilla/websocket is v1.5.3: `Upgrader.CheckOrigin` takes `*http.Request`
  (NOT `*websocket.HandshakeRequest`), and `SetPongHandler` takes
  `func(appData string) error` (NOT `func() error`). `net/http` must be
  imported for the CheckOrigin signature.
- Build: `cd dapp_browser/go && GOFLAGS=-mod=mod go build ./...` (exit 0).

## frontend/web_nextjs (Next.js app)

- `app/master_wallet/page.tsx` generates a VALID 24-word BIP-39 mnemonic via
  `@scure/bip39` `generateMnemonic(wordlist, 256)` (256-bit entropy + checksum).
  Import uses `validateMnemonic` (wordlist + checksum), not just word count.
  NEVER pick words from only the first 24 BIP-39 words.
- `@scure/bip39` subpath import MUST include the `.js` extension:
  `import { wordlist } from '@scure/bip39/wordlists/english.js'` (the package
  `exports` map keys require it; `moduleResolution: bundler` in tsconfig
  resolves the types either way, but Node ESM at runtime needs the `.js`).
- The wallet mnemonic is NOT stored in plaintext localStorage. It is encrypted
  with AES-GCM (256-bit) using a PBKDF2-derived key (600k iters, SHA-256) from
  a user password; only the `{v, salt, iv, ciphertext}` blob is persisted
  (`masterWallet` key). Mnemonic lives in React state (memory) for the session;
  `unlockWallet` decrypts on demand. NOTE comment: production should use a
  hardware wallet / secure enclave / HSM-backed KMS.
- `app/wallet/TigerWallet.tsx` has a PRE-EXISTING syntax error (line ~64:
  `isEVM: true explorer:` missing comma) unrelated to wallet work; `npx tsc
  --noEmit` reports it but it is not from these changes.

## Network / package install

## go/wallet_api (canonical wallet backend — REAL)

- The canonical Go wallet backend at `go/wallet_api/` is the ONLY service that
  performs key management and signing. It replaces the old `go/wallet_service`
  (which used NIST P-256 + `sha512(seed)` — NOT secp256k1/BIP-32). Do NOT use
  `go/wallet_service` for signing.
- Real BIP-39 mnemonic generation (`tyler-smith/go-bip39`), real BIP-32 HD
  derivation (`hd_derive.go`: HMAC-SHA512 "Bitcoin seed" master + CKDpriv mod-n
  via secp256k1), BIP-44 path parsing (`m/44'/60'/0'/0/0`, `'`/`h`/`H` hardened
  suffixes). The **canonical BIP-44 test vector PASSES**: mnemonic
  "abandon abandon ... about" m/44'/60'/0'/0/0 → `0x9858EfFD232B4033E47d90003D41EC34EcaEda94`.
- Real EVM transaction signing via `go-ethereum/core/types.SignTx` +
  `NewLondonSigner` (EIP-1559 DynamicFeeTx + legacy LegacyTx), real
  `eth_sendRawTransaction` broadcast. `MarshalBinary()` gives the signed RLP.
- Real ECDSA personal_sign (`crypto.Sign` with Ethereum prefix
  `keccak256("\x19Ethereum Signed Message:\n" + len + msg)`), recovery byte 27/28.
- Seed encryption: AES-256-GCM with scrypt-derived key (N=32768, r=8, p=1).
  Password never stored; wrong password fails (GCM auth tag).
- PostgreSQL persistence (`store.go`, pgx/v5 pool): `users`, `wallets` (stores
  `encrypted_seed`), `address_book`, `transaction_log`. Redis cache for
  balances/prices/gas (30-60s TTL). No SQLite anywhere.
- Real fetchers (`fetchers.go`): native balance via `eth_getBalance`, ERC-20
  via `balanceOf` eth_call, tx history via Etherscan API, NFTs via Etherscan,
  prices via CoinGecko, gas via `eth_feeHistory`/`eth_gasPrice`.
- REST API (gin, port 8443): `/api/v1/auth/{register,login}`,
  `/api/v1/wallets` (POST create, GET list), `/api/v1/{balance,tokens,transactions,nfts}`,
  `/api/v1/send`, `/api/v1/sign`, `/api/v1/gas`, `/api/v1/price`, `/api/v1/chains`.
  Public read endpoints at `/api/v1/public/{balance,tokens,transactions,nfts}`.
  JWT (HS256, 24h) auth middleware on protected routes.
- Build: `cd go/wallet_api && go build ./...` (exit 0). Tests: `go test ./...`
  (11 tests pass, including BIP-44 test vector). `go vet` clean.
- Docker: `go/wallet_api/Dockerfile` (multi-stage, golang:1.23-alpine → alpine).
- Frontend connects to this backend (port 8443, not 8080). All Next.js API
  routes updated to use `localhost:8443`. The `WalletService` class in
  `frontend/web_nextjs/app/api/service.ts` calls the real endpoints.
- Browser extension (`browser_extensions/chrome`) connects to the same backend
  via `BACKEND_URL = 'http://localhost:8443'`. All fake stubs removed
  (`generateMnemonic`, `deriveAddress`, `signTransaction`, `personalSign`,
  `signTypedData`, `exportPrivateKey` now call the backend or throw). No more
  hardcoded `"abandon "`×12 or `0x`+fake signatures.

## docker-compose.yml (cleaned)

- Rewritten from 17 broken services → 10 working services: `postgres`,
  `redis`, `wallet-api`, `wallet-frontend`, `super-admin-api`,
  `white-label-frontend`, `permission-service`, `connection-api`,
  `fetcher-gateway`, `monitoring-dashboard`. All build contexts have real
  Dockerfiles. `database/init.sql` creates the `tigerwallet` DB + schema on
  first boot. No SQLite.

- npm registry is reachable in this env (`npm ping` → PONG). `npm install`
  works (installs the full tree). `@scure/bip39` + `@noble/hashes` +
  `@scure/base` were added to `frontend/web_nextjs/package.json`.

## Rust toolchain

- No Rust toolchain is preinstalled in this env. Install on demand with
  `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal`
  then `. "$HOME/.cargo/env"`. Build/check zk_infrastructure with
  `cargo check` / `cargo test --lib` from
  `core/rust/zk_infrastructure`.

## zk_infrastructure / ZK prover

- `zk_infrastructure` uses a REAL Fiat-Shamir Schnorr proof of knowledge of a
  discrete log over Ristretto255 (curve25519-dalek + sha2). No all-zero proof
  stubs, no always-true verification. Prover serializes `R || s || Y`;
  verifier recomputes the challenge and checks `s*G == R + e*Y` and rejects
  the identity point / malformed encodings. Do NOT regress these to stubs.
- The Schnorr scheme has no trusted setup; `setup()` only marks the circuit
  keyed so `prove` refuses unset-up circuits (preserves the existing API).

## desktop_wallet (C++)

- CMake project at `desktop_wallet/`, C++20. Depends on CURL + OpenSSL
  (dev pkgs: `libcurl4-openssl-dev libssl-dev`; `cmake` was not preinstalled —
  `sudo apt-get install -y cmake`). `g++` 14.x is available.
- UI is NOT a GUI toolkit (no ImGui/Qt/wx). UI components under
  `src/ui/components/**.hpp` are header-only HTML-string generators. Themes
  are therefore applied by injecting CSS color strings, not by styling a
  native widget set. `ThemeManager` (`include/ui/theme.hpp` +
  `src/ui/theme.cpp`, namespace `TigerWallet`) is a singleton that owns the
  dark/light palettes and persists preference to a JSON file.
- BUILD STATUS (fixed 2026-08-08): `cd desktop_wallet && rm -rf build && mkdir
  build && cd build && cmake .. && make -j4` now succeeds with exit 0, building
  `tigerwallet_core` (static lib) + `tigerwallet_test` targets. p2p_trading
  service + theme code compile clean.
- SINGLETON PATTERN FIX: service headers (`include/services/*` +
  `include/services/master/master_wallet_service.h`) used the singleton pattern
  with `private:` ctor/dtor, which breaks `std::make_shared`/`std::construct_at`
  ("... is private within this context"). Fix applied: keep deleted copy ctor
  + copy-assignment in `private:`, but MOVE the ctor/dtor to `public:`.
  Headers fixed: blockchain_service.h, price_service.h, swap_service.h,
  staking_service.h, nft_service.h, keychain_manager.h, api_client.h,
  master_wallet_service.h. Do NOT regress these back to private ctor/dtor.
- When a struct has a user-declared default constructor it is NOT an aggregate,
  so brace-init-list assignment like `tokens_ = { {"BTC","Bitcoin",1.5,"₿"}, ... }`
  fails ("no match for operator= ... <brace-enclosed initializer list>"). Fix:
  add an explicit constructor taking all the initialized fields (done for
  `ConvertToken`/`ConvertPair` in `src/services/convert_service.h` and
  `Trader` in `src/services/copy_trading_service.h`). Include `<utility>` for
  `std::move` in those headers.
- `std::cerr` requires `#include <iostream>` — was missing in
  `src/models/wallet_models.cpp`; added. `std::remove_if` requires
  `#include <algorithm>` — was missing in `src/services/margin_trading_service.cpp`;
  added.
- Note: the bundled `cpp/rpc_manager/json.hpp` referenced as
  `<nlohmann/json.hpp>` is a stub; not on the current build's include path for
  the core/test targets, so it doesn't block the build.

## rust/key_management crypto tests (src/main.rs)

- `keccak256` uses `tiny_keccak::Keccak::v256` (correct Ethereum keccak256, NOT
  sha3-256). The production `derive_address` EIP-55 path already hashes the
  LOWERCASE hex address and is correct.
- BIP-32 `ckd_priv` + `master_key_from_seed` are CORRECT per official BIP-32
  test vector 1 (verified against `bip32utils`):
  - seed 000102...0f -> m/0' priv
    edb2e14f9ee77d26dd93b4ecede8d16ed408ce149b6cd80b0715a2d911a0afea, cc
    47fdacbd0f1097043b78c63c20c34ef4ed9a111d980047ad16282c7ae6236141;
    m/0'/1 priv 3c6cb8d0..., cc 2a7857631386ba23dacac34180dd1983734e444fdbf774041578e9b6adb37c19;
    m/0'/1/2'/2 priv 0f479245fb19a38a1954c5c7c0ebab2f9bdfd96a17563ef28a6a4b1a2a764ef4.
- BIP-39 abandon...about / empty passphrase -> m/44'/60'/0'/0/0 priv
  1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727, address
  0x9858EfFD232B4033E47d90003D41EC34EcaEda94 (both correct). The
  Hardhat/test-junk key ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
  derives address 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 -- do NOT use it as
  the expected private key for the abandon mnemonic; it is a different vector.
- TWO test-helper bugs were fixed (NOT expected-value changes):
  1. test_address_eip55_checksum hashed the MIXED-CASE address instead of the
     lowercase one -> fixed to hash `lower`.
  2. base58_decode consumed leading zero bytes as the LSB of the big integer
     (prepending them BEFORE accumulation) -> fixed to prepend them AFTER
     accumulation. Bitcoin P2PKH addresses are 25 bytes (1 version + 20 hash160
     + 4 double-SHA256 checksum).
- The 4 BIP-32/BIP-44 tests (test_bip32_ckdpriv_h0, test_bip32_ckdpriv_h0_1,
  test_bip32_derive_path, test_bip44_ethereum_known_vector) have WRONG expected
  values in the test file that conflict with the official BIP-32 vectors the
  code (correctly) produces. They cannot pass without either corrupting correct
  crypto or correcting the expected values. The test_bip44_ethereum_known_vector
  priv-key assertion is also internally inconsistent with its own address
  assertion. Left unchanged per explicit user constraint not to edit expected
  values -- needs user decision.

## core/rust AMM/orderbook/mev crates (fixed 2026-08-09)

- `num-bigint` `BigUint` has **no** `to_f64()` method. Use
  `num_traits::ToPrimitive` (`biguint.to_f64()`), or fall back to
  `to_string().parse::<f64>()`.
- `is_even()`/`is_odd()` on BigUint come from the **`num-integer`** crate
  (`num_integer::Integer`), NOT `num_traits::Integer` (which doesn't exist).
  Add `num-integer = "0.1"` to Cargo.toml.
- Const evaluation of `BigUint::from(1u64) << 96` fails in const context (used
  for Q96/Q128/MAX_UINT128). Use `std::sync::LazyLock` with
  `LazyLock::new(|| BigUint::from(1u64) << 96)`.
- `&LazyLock<BigUint>` does NOT auto-deref in arithmetic comparisons/divisions.
  Use `&*Q96` (or `&*MAX_UINT128`) explicitly.
- `serde::Deserialize` derive needs `serde` with the `derive` feature in
  Cargo.toml + `use serde::Deserialize;`.
- `ahash::AHashMap` requires the `ahash` crate as an explicit dep.
- `amm`: `PoolCore::liquidity()` added (reads `gross_liquidity`); `swap()`
  signature changed to `swap(&self, amount_in, zero_for_one, sqrt_price_limit:
  Option<&BigUint>) -> Result<SwapResult, String>` (was 2-arg -> SwapResult);
  `PoolState` fields made `pub`. Cargo.toml gained `num-integer` + `ahash`.
- `mev`: `#[derive(Deserialize)]` added to `MEVAttackType`; `pool_address` type
  mismatch fixed.
- `orderbook`: borrow/mutability fixes (clone where needed, reorder statements).
- Build: `source "$HOME/.cargo/env"` then `cargo check` per crate. All three
  crates now compile with 0 errors (warnings only).


## Go services (multisig_service, mpc) - crypto/build notes
- Go binary: /home/openhands/go/bin/go (add to PATH via `export PATH="/home/openhands/go/bin:$PATH"`). System go not installed. NOTE: /home/openhands/.local/go/bin does NOT contain go — use /home/openhands/go/bin.
- Two modules: go/multisig_service (github.com/tigerwallet/multisig-service), go/mpc (github.com/tigerwallet/mpc).
- All signing uses github.com/ethereum/go-ethereum/crypto (real ECDSA secp256k1, low-s). No sha256/sha3 fakes.
- multisig_service: broadcastTransaction (main.go) and broadcastRawTransaction (multisig_service.go) use ethclient.Dial + types.NewTx(DynamicFeeTx) + types.SignTx + client.SendTransaction. RPC from ETH_RPC_URL env.
- SignHash uses crypto.Sign(hash, privKey), v normalized to 27/28.
- mpc/enterprise.go: real Shamir + Lagrange over secp256k1; CombineShares reconstructs then crypto.Sign. Real crypto.Ecrecover verification. secp256k1 only (no P256 for crypto).
- mpc/server.go: REAL HTTP service exposing the TSS engine. Endpoints:
  POST /api/v1/mpc/create ({threshold,totalShards} -> {keyId,address,publicKey});
  POST /api/v1/mpc/sign ({keyId,messageHash(hex)} -> {signature}); GET
  /api/v1/mpc/wallet/:keyId; GET /api/v1/health. Port from MPC_PORT (default
  9099). VERIFIED end-to-end: Ecrecover recovers the wallet address from the
  produced signature. The old demo print-based main() was removed from
  enterprise.go. go build + go vet clean.
- Frontend MPCLogin (src/components/mpc/MPCLogin.tsx) now calls the real MPC
  backend (createMPCWallet POSTs to /api/v1/mpc/create). login() does a real
  OIDC redirect (throws if no client_id). loginWithEmail requests a real
  magic-link. NO MORE simulateOAuthFlow / random-32-byte local key.
- pkg/threshold/sign.go: HashMessage = Keccak-256; CombineSignatures/SignWithTSS use lagrangeCoefficients + crypto.Ecrecover; low-s normalization. Structs carry PublicKey []byte; SigningSession has GroupPublicKey []byte.
- Build/vet both clean: cd go/<dir> && go build ./... && go vet ./...

## Go services fixed 2026-08-09 (wallet_service, swap, staking, payment, ens)
- go/wallet_service/main.go: REWRITTEN as a deprecation reverse-proxy shim to
  canonical wallet_api (stdlib net/http/httputil only). The old P-256 +
  sha512(seed) broken crypto is GONE. go.mod trimmed to no external deps
  (no go.sum). Build: `cd go/wallet_service && go build ./... && go vet ./...`
  (OK). Set WALLET_API_URL env (default http://localhost:8443), PORT (default 8001).
- go/swap_service/main.go: ExecuteSwap no longer fabricates TxHash; status
  "quote_ready" + `action_required` -> wallet_api POST /api/v1/send. Fixed
  pre-existing build breaks: big.Float.String() returns 1 value (not 2),
  removed unused context/encoding/json imports, fixed mangled AddLiquidity
  (req.B_token -> req.TokenB, removed duplicated dangling block). Build+vet OK.
- go/staking_service/: moved staking_service.go into staking/ subpackage
  (package staking) to resolve the "two packages in one dir" build break
  (main.go is package main). main.go: fake validators (0x1234...) -> empty
  Address + Verified:false + Status:"sample"; no-op JWT (c.Set user_id
  "user-123") -> real golang-jwt/v5 HMAC validation with JWT_SECRET env.
  staking/staking.go: SetString returns (*Int,bool) not (*Int,error); added
  missing LastClaimTime field to UserStake. Build+vet OK.
- go/payment/main.go: processWithdrawal now does a REAL on-chain ERC-20
  transfer(address,uint256) — manual calldata (selector 0xa9059cbb + padded
  addr + amount) + types.NewTx(DynamicFeeTx) + types.SignTx(NewLondonSigner)
  + ethclient.SendTransaction. No more `0x%x` of timestamp. If no hot-wallet
  key/RPC, status="requires_signing" (not failed, not faked).
  generatePaymentAddress returns getHotWalletAddress() (no fabricated sha256
  deposit address). Removed unused aes/cipher/elliptic/rand/sha256/hex
  imports. Fixed receipt.BlockNumber (*big.Int) confirmation math. Build+vet OK.
- go/ens_service/main.go: nameHash/labelHash now use crypto.Keccak256 (EIP-137
  recursive namehash; was sha256). Resolve/ReverseResolve do real on-chain
  CallContract against ENSRegistry (0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e):
  resolver(bytes32)=0x0178b8bf, addr(bytes32)=0x3b3b57de, name(bytes32)=0x691f3431.
  No more hardcoded 0x0000...0001/0x0000...0000. Added ethclient to struct +
  NewENSService. Created go.mod (github.com/tigerwallet/ens-service) with
  go-ethereum + gin + go-redis/v8. Build+vet OK.

## frontend/web_nextjs fixes 2026-08-09
- app/wallet/page.tsx: REWRITTEN to use real WalletService (createWallet,
  sendTransaction, getBalance, getTransactions) + auth (login/register). No
  more Math.random() mnemonic / fabricated 0x addresses.
- app/api/service.ts: API_CONFIG.baseURL is same-origin '' in browser (talks
  to Next.js proxy routes) and BACKEND_URL on server. BlockchainService
  delegates to backend (ethers removed).
- app/api/v1/_proxy.ts: unchanged (proxyGet/proxyMutation, BACKEND_URL).
- Fixed 30 route.ts files: _proxy import had one extra ../ (resolved).
- Added 15 missing proxy routes: app/api/v1/{wallets,send,sign,balance,
  tokens,transactions,nfts,gas,chains,auth/login,auth/register,
  public/balance,public/tokens,public/transactions,public/nfts}/route.ts.
- tsc --noEmit: 0 errors in all changed files (57 pre-existing errors in
  OTHER pages like hardware-wallet/launchpad/lending/bridge remain — not
  from these changes).

## frontend/web_nextjs fixes 2026-08-09/10 (TS strict + wallet hook)
- ALL TypeScript errors fixed (48 -> 0 via `npx tsc --noEmit`). Key fixes:
  - ThemeProvider (app/components/ThemeProvider.tsx): added full ThemeColors
    palette (bgPrimary/Secondary/Tertiary/Card, textPrimary/Secondary/Tertiary,
    border, accent, success, error, warning, overlay) + LIGHT_COLORS/DARK_COLORS
    + `colors` in context, so listing/admin pages render in both themes.
  - launchpad: `?? 0` on optional vesting fields, `|| '#'` on optional href.
  - master_wallet: cast `Uint8Array` -> `BufferSource` for WebCrypto
    deriveKey/decrypt (TS 5.7 typing of `Uint8Array<ArrayBufferLike>`).
  - bug-bounty: added `'verified'` to status union.
  - protection-fund: real `claims` state (not MOCK_CLAIMS), field names
    claimId/userAddress, same-origin API_BASE.
  - fiat-ramp: `address || undefined` coercion.
  - farming/pool/twap: MUI v5 `InputAdornment` requires `position` prop.
  - passkey: removed duplicate local BrowserAdapter class; fixed
    PublicKeyCredentialParameters type; typeof checks for WebAuthn support.
  - client.ts: removed duplicate startCopying method.
- app/wallet.ts: REWROTE the mock `useWallet` hook (hardcoded fake address
  0x742d35Cc... + balance '1.5') as a REAL EIP-1193 injected-provider hook:
  connect() -> eth_requestAccounts; balance via eth_getBalance; listens to
  accountsChanged/chainChanged; silent reconnect on mount via eth_accounts.
  The canonical `window.ethereum` (Eip1193Provider) global type lives here;
  TigerWalletKit.tsx's duplicate declaration was removed.


## Go services audit (go/ directory)

- The Go toolchain is at /home/openhands/go/bin/go (NOT /home/openhands/.local/go/bin/go as some prompts suggest). Set `export PATH=$PATH:/home/openhands/go/bin` and `export GOPATH=/home/openhands/.gopath` (the latter avoids the "GOPATH set to GOROOT has no effect" warning).
- ~90 immediate subdirs of go/ contain .go files; 27 have a go.mod.
- Many "service" dirs are orphans: they have .go files but NO go.mod, so `go build ./...` fails with "directory prefix . does not contain main module or its selected dependencies".
- 5 dirs have a package-name conflict (two distinct package declarations, one main + one named): copy_trading_service, fiat, monitoring_service, perpetual_service, rpc_service. These also lack a go.mod, so the conflict only surfaces after a go.mod is added.
- admin_service and super_admin_service HAVE go.mod but fail to build: both have config.MinConns (uint32) assigned to an int32 field (database/db.go); super_admin_service also has unused imports in main.go (encoding/base64, encoding/json, golang-jwt/jwt/v5).
- Clean builds (has go.mod, ec=0): blockchain_rpc, ens_service, lending_service, payment, staking_service, swap_service, wallet_service, walletconnect.

## Go services build-fix sweep (35 services under go/)

- All 35 target services build clean (go build ./... exit 0): advanced_analytics_service, airdrop_service, api_gateway, approval_manager, blockchain_registry, bridge, bridge_aggregator, bug_bounty_service, cdn_service, cex_connector, coupon_service, cross_chain_aggregator, distributed_trading, earn_service, enterprise_service, fiat_offramp, fiat_onramp, fiat_ramp, full_fetchers, liquidity, nft_prices, notifications, oracle, paper_trading, portfolio_tracker, rate_limiter_service, rbac_admin_service, real_time_charts, red_packets_service, rpc_node_manager, sdk, tax_reports, two_factor_auth, webhook_service, white_label_service.
- Key recurring fix patterns: SetString capture bool; math.random->rand.Float64; re-add math import; uint64 cast for FormatUint; use exported struct field names + add missing fields; io.ReadAll for io.ReadCloser->w.Write; cex_connector restructured into subpackages; syntax fixes for bad struct literals/switch; earn_service bool err->real error; remove unused vars; package-level config var (white_label_service); two_factor_auth uses totp.Generate(totp.GenerateOpts{...}) no Skew; tax_reports Lot type moved to package level + dedup cases + hLots typo; distributed_trading no atomic.LoadFloat64; full_fetchers TokenAmount.Address.
- Did NOT run go mod tidy, did NOT commit, did NOT delete go.mod.
