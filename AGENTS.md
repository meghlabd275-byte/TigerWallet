# TigerWallet â€” Agent Memory

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
- **PORT COMPLETE (2026-08-11):** `account_abstraction/SimpleAccount.sol` now
  extends the canonical `BaseAccount` and implements `_validateSignature` with
  REAL ECDSA (OZ v5 `ECDSA.recover` + `MessageHashUtils.toEthSignedMessageHash`
  over the EntryPoint `userOpHash`). The legacy accept-any-64-byte-sig
  `validateUserOp` is replaced; non-65-byte/wrong-signer sigs return
  `SIG_VALIDATION_FAILED (1)`. `account_abstraction/AccountFactory.sol` deploys
  deterministic EIP-1167 clones via `Clones.cloneDeterministic` +
  `predictDeterministicAddress` (counterfactual address matches create output).
  `test_aa/AccountFactory.t.sol`: 5 passing Foundry tests (real `vm.sign`, no
  mocks). `legacy_aa/AccountFactory.sol` kept as historical reference only.
- **VerifyingPaymaster (2026-08-11):** `account_abstraction/VerifyingPaymaster.sol`
  extends the audited `BasePaymaster` and sponsors gas only when an off-chain
  sponsor's real ECDSA signature over the EIP-191-prefixed `userOpHash`
  recovers to the registered `signingSigner` (Pimlico/Stackup
  verifying-paymaster pattern â€” the GetGas-equivalent gas-subsidy product).
  Fail-closed sender whitelist, `validUntil`/`validAfter` time-range bounds,
  owner-gated signer rotation, inherited stake/deposit/withdraw via
  `Stakeable`/`BasePaymaster`. `test_aa/VerifyingPaymaster.t.sol`: 8 passing
  Foundry tests (real `vm.sign`, no mocks).
- **MultisigWallet (2026-08-11):** `account_abstraction/tigerwallet/MultisigWallet.sol`
  is a deployable Gnosis Safe-style on-chain threshold multisig (NOT an ERC-4337
  account). A tx executes only after `threshold` owner ECDSA signatures over an
  EIP-712 typed-data hash (`domain(chainId,verifyingContract) ||
  Transaction(to,value,dataHash,nonce)`) are collected. Verification uses OZ v5
  `ECDSA.recover` (real secp256k1, low-s) â€” NOT length checks. Sorted-sig
  convention (`recovered > lastOwner`) dedups without storage. `ReentrancyGuard`,
  nonce replay protection, threshold clamped to `[1, ownerCount]`, self-governed
  owner mgmt (add/remove/changeThreshold) via the wallet's own execute path,
  constructor rejects duplicate/zero owners + bad threshold. Pairs with the
  off-chain `go/multisig_service` relayer (already real secp256k1 + ethclient).
  `test_aa/MultisigWallet.t.sol`: 13 Foundry tests (real `vm.sign`). Full AA
  suite now 31/31 pass.
- `paymasterAndData` layout for the VerifyingPaymaster: `address(20) ||
  verificationGasLimit(16) || postOpGasLimit(16) || sponsorSignature(65) ||
  validUntil(6) || validAfter(6)` (the slice after the fixed 52-byte head is
  `UserOperationLib.PAYMASTER_DATA_OFFSET`).
- TigerWallet's pre-existing custom `AccountFactory.sol` (which used the old
  unpacked `UserOperation` + an on-chain `Bundler`) was relocated to
  `smart_contracts/evm_contracts/legacy_aa/` because it does not compile against
  the canonical packed types. It must be ported to `PackedUserOperation` +
  `UserOperationLib` and extend `BaseAccount`/`BasePaymaster` before reuse.

## Foundry setup

- Foundry is NOT preinstalled. Install on demand:
  `curl -L https://foundry.paradigm.xyz | bash` then `~/.foundry/bin/foundryup`.
  Binary lands at `~/.foundry/bin/forge` (v1.7.1). Add `$HOME/.foundry/bin`
  to PATH for the session.
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
  `cd smart_contracts/evm_contracts && forge build` â†’ "Compiler run successful!"
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
  NEVER return a fake all-zero `0x0000...` signature â€” if no signer key is
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

## go/wallet_api (canonical wallet backend â€” REAL)

- The canonical Go wallet backend at `go/wallet_api/` is the ONLY service that
  performs key management and signing. It replaces the old `go/wallet_service`
  (which used NIST P-256 + `sha512(seed)` â€” NOT secp256k1/BIP-32). Do NOT use
  `go/wallet_service` for signing.
- Real BIP-39 mnemonic generation (`tyler-smith/go-bip39`), real BIP-32 HD
  derivation (`hd_derive.go`: HMAC-SHA512 "Bitcoin seed" master + CKDpriv mod-n
  via secp256k1), BIP-44 path parsing (`m/44'/60'/0'/0/0`, `'`/`h`/`H` hardened
  suffixes). The **canonical BIP-44 test vector PASSES**: mnemonic
  "abandon abandon ... about" m/44'/60'/0'/0/0 â†’ `0x9858EfFD232B4033E47d90003D41EC34EcaEda94`.
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
- Curated dApp directory (`dapp_directory.go`): ~20 real protocol entries
  (Uniswap, Aave, OpenSea, Curve, 1inch, Jupiter, Stargate, Lido, ENS, Lens,
  Farcaster, etc.) with categories/chains/verified flag â€” NO fabricated metrics
  (no invented user counts/ratings). Public REST `GET /api/v1/dapps`
  (`?category=&chain=`), `/dapps/categories`, `/dapps/:id`. Frontend
  `dapp-store` + `dapp-browser` pages fetch this instead of hardcoding.
- Token asset registry (`token_registry.go` + `handleTokenRegistry`): real
  per-chain token lists (mainnet contract addresses/decimals/symbols for
  Ethereum/BSC/Polygon/Arbitrum/Optimism/Base). Public REST
  `GET /api/v1/tokens/registry` (full registry grouped by chain, or
  `?chain_id=N`; 404 for unknown chain â€” never fabricated).
- REST API (gin, port 8443): `/api/v1/auth/{register,login}`,
  `/api/v1/wallets` (POST create, GET list), `/api/v1/{balance,tokens,transactions,nfts}`,
  `/api/v1/send`, `/api/v1/sign`, `/api/v1/gas`, `/api/v1/price`, `/api/v1/chains`,
  `/api/v1/swap/{quote,execute}`, `/api/v1/staking/{quote,stake,unstake,claim}`,
  `/api/v1/transactions/:txHash`. Public read endpoints at
  `/api/v1/public/{balance,tokens,transactions,nfts}`. JWT (HS256, 24h) auth
  middleware on protected routes. The swap/quote uses a real CoinGecko
  cross-rate; swap/execute + staking stake/unstake/claim return the on-chain
  action to submit via the real `/api/v1/send` (no fabricated tx hashes);
  staking/quote returns supported native assets with APY 0 (no invented yield).
- Build: `cd go/wallet_api && go build ./...` (exit 0). Tests: `go test ./...`
  (11 tests pass, including BIP-44 test vector). `go vet` clean.
- Docker: `go/wallet_api/Dockerfile` (multi-stage, golang:1.23-alpine â†’ alpine).
- Frontend connects to this backend (port 8443, not 8080). All Next.js API
  routes updated to use `localhost:8443`. The `WalletService` class in
  `frontend/web_nextjs/app/api/service.ts` calls the real endpoints.
- Browser extension (`browser_extensions/chrome`) connects to the same backend
  via `BACKEND_URL = 'http://localhost:8443'`. All fake stubs removed
  (`generateMnemonic`, `deriveAddress`, `signTransaction`, `personalSign`,
  `signTypedData`, `exportPrivateKey` now call the backend or throw). No more
  hardcoded `"abandon "`Ă—12 or `0x`+fake signatures.

## docker-compose.yml (cleaned)

- Rewritten from 17 broken services â†’ 10 working services: `postgres`,
  `redis`, `wallet-api`, `wallet-frontend`, `super-admin-api`,
  `white-label-frontend`, `permission-service`, `connection-api`,
  `fetcher-gateway`, `monitoring-dashboard`. All build contexts have real
  Dockerfiles. `database/init.sql` creates the `tigerwallet` DB + schema on
  first boot. No SQLite.

- npm registry is reachable in this env (`npm ping` â†’ PONG). `npm install`
  works (installs the full tree). `@scure/bip39` + `@noble/hashes` +
  `@scure/base` were added to `frontend/web_nextjs/package.json`.

## Rust toolchain

- No Rust toolchain is preinstalled in this env. Install on demand with
  `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal`
  then `. "$HOME/.cargo/env"`. Build/check zk_infrastructure with
  `cargo check` / `cargo test --lib` from `core/rust/zk_infrastructure`.
  Toolchain installed 2026-08-11: cargo/rustc 1.97.1 (stable, minimal profile).

## hardware_wallet/rust (Ledger APDU layer â€” real, fail-closed)

- `hardware_wallet/rust/src/ledger/mod.rs` is a REAL Ledger Ethereum-app APDU
  protocol layer (NOT a fake). Implements: APDU constants (CLA 0xE0,
  INS_GET_PUBLIC_KEY 0x02 / INS_SIGN 0x04 / INS_GET_APP_CONFIGURATION 0x06,
  P1_FIRST/P1_MORE, P2_TRANSACTION/P2_TYPED_DATA/P2_MESSAGE, status words
  0x9000/0x6985/0x6A80/0x6700/0x6A86); real APDU builders
  (`build_get_public_key_apdu`, `build_sign_apdu`, `build_get_app_configuration_apdu`
  â€” BIP-32 path BE-encoded, Lc length); real response parsers
  (`parse_get_public_key_response` pubKey+address, `parse_sign_response`
  v||r||s with v normalized 0/1â†’27/28, `split_status_word`, `check_status`,
  `parse_app_config_response`); real EIP-191 host-side message prefixing
  (`eip191_personal_message`). `ApduTransport` trait + fail-closed `NoTransport`
  default â€” NO fake signature is ever produced. `derive_bip32_path` accepts
  `'`/`h`/`H` hardened suffixes, rejects oversized indices + empty paths.
- The fake `"02"+zeros` public key and `vec![0u8;64]` signature are GONE.
  `cargo test --lib` â†’ 20/20 pass (19 ledger via a canned-response transport).
- Trezor/OneKey/Ellipal/SafePal modules are now fail-closed (DeviceNotFound)
  â€” removed fake all-zero keys/sigs + the compile-broken `"0x" + &hex::encode()`
  string concat. AirGap is a legitimate QR-code air-gapped protocol (kept).
- Production next step: wire `ApduTransport` to a HID/BLE backend (hidapi /
  ledger-rs) â€” the protocol layer above is unchanged.

## wallet_core (Rust core) â€” keystore_v3

- `wallet_core/src/keystore_v3.rs` is a REAL Web3 Secret Storage v3
  implementation (Geth/MetaMask/MyCrypto keystore JSON format â€”
  https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition).
- Both real KDFs: `scrypt` (default N=131072,r=8,p=1,dklen=32) and
  `pbkdf2` (HMAC-SHA256, 262144 iters). Power-of-two N validation.
- Cipher AES-128-CTR (real `aes`+`ctr` crates). MAC =
  keccak256(derived_key[16:32]â€–ciphertext), constant-time compared via
  `subtle::ConstantTimeEq`. Wrong password / tampered ciphertext â†’
  `MacMismatch` (fail-closed). Derived material zeroized.
- serde structs match the spec field names exactly (`crypto.cipher`,
  `ciphertext`, `cipherparams.iv`, `kdf`, `kdfparams.{n,r,p,c,prf,dklen,
  salt}`, `mac`, `id`, `version`, `address`).
- API: `encrypt_key`/`encrypt_key_scrypt`/`encrypt_key_pbkdf2`/`decrypt_key`/
  `to_json`/`from_json`. Added deps `ctr = "0.9.2"`,
  `scrypt = { version = "0.11", default-features = false }`.
- `cargo test --lib keystore_v3` â†’ 12/12 pass (real crypto, no mocks):
  scrypt/pbkdf2 roundtrip, wrong-password fails, JSON roundtrip, invalid
  key length, non-power-of-two N rejected, unsupported cipher/KDF
  rejected, MAC non-constant (real randomness), both-KDFs cross-JSON
  interop, tampered-ciphertext MAC break, version-2 rejected.
- This is the Rust-core counterpart of `go/wallet_api/keystore_v3.go`
  (Go backend already had the scrypt variant + REST endpoints); both
  now produce spec-valid keystores importable across wallets.

## solana/rust (Solana core â€” real Ed25519 + PDA)

- New crate `tiger_solana_core` (`solana/rust/src/lib.rs`). Replaces the
  C++ `solana_core.cpp` fakes (SHA256-as-pubkey, unsalted-SHA256 PDA).
- `derive_public_key`: REAL Ed25519 via `ed25519-dalek`. Accepts 32-byte
  seed OR 64-byte expanded key (uses the seed half). The pubkey is
  `scalar_mult(seed)` via SHA-512 clamping, NOT SHA-256. Validates the
  result decompresses to a real on-curve point.
- `find_program_address` / `create_program_address`: the canonical Solana
  PDA algorithm â€” `sha256(seeds || program_id || bump)` with
  `PDA_MARKER = b"ProgramDerivedAddress"`, 255â†’1 bump-seed search, and
  curve25519 on-curve REJECTION (a PDA must NOT be a valid Ed25519
  pubkey). Matches `solana-program`.
- `sign_message` / `verify_signature`: real Ed25519.
- `pubkey_to_base58` / `pubkey_from_base58`: real base58, rejects wrong
  length (must be 32 bytes).
- `cargo test --lib` â†’ 12/12 pass (no mocks): pubkey derivation (seed +
  expanded), bad-length reject, sign+verify roundtrip + tamper, PDA is
  off-curve, PDA deterministic + idempotent bump, create-with-bump ==
  find, PDAs differ across program ids, long-seed reject, base58
  roundtrip, base58 wrong-length reject, pubkey non-zero.
- C++ side: the three fake SHA-256 derivations (`derive_public_key`,
  `TokenAddress::create`, `NFTMetadata::get_metadata_address`) are now
  fail-closed (return all-zero sentinel + comment). The legitimate
  `Message::hash` (SHA-256 of the serialized message â€” used for signing)
  is kept; that one is real.

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
  (dev pkgs: `libcurl4-openssl-dev libssl-dev`; `cmake` was not preinstalled â€”
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
- BUILD PREREQS (2026-08-11): cmake was NOT installed in a fresh env. Install
  with `sudo apt-get update && sudo apt-get install -y cmake libcurl4-openssl-dev
  libssl-dev` first (CURL + OpenSSL dev headers are required by the FindCURL
  / FindOpenSSL CMake modules). g++ 14.x is preinstalled. After install, the
  standard build command above works.
- NEW (2026-08-11): `src/services/swap_service.cpp` now calls the real on-chain
  AMM router (`GET /api/v1/amm/quote` for `getAmountsOut`, then the two-step
  `POST /api/v1/amm/swap` + `POST /api/v1/send` for real tx broadcast) â€” same
  backend path as the web frontend. NEW `include/services/multisig_service.h`
  + `src/services/multisig_service.cpp`: C++ service calling the
  `go/multisig_service` REST API (port 8450) for threshold multisig
  (createWallet/listWallets/addOwner/createTransaction/signTransaction/
  executeTransaction/revokeTransaction/pendingTransactions). Uses a dedicated
  `APIClient` instance (NOT the wallet_api singleton) pointed at the multisig
  port. Added to `CMakeLists.txt`. Singleton ctor moved to public (copy ctor
  stays deleted in private) so `std::make_shared` works â€” same fix pattern as
  the other desktop services.
- SINGLETON PATTERN FIX: service headers (`include/services/*` +
  `include/services/master/master_wallet_service.h`) used the singleton pattern
  with `private:` ctor/dtor, which breaks `std::make_shared`/`std::construct_at`
  ("... is private within this context"). Fix applied: keep deleted copy ctor
  + copy-assignment in `private:`, but MOVE the ctor/dtor to `public:`.
  Headers fixed: blockchain_service.h, price_service.h, swap_service.h,
  staking_service.h, nft_service.h, keychain_manager.h, api_client.h,
  master_wallet_service.h. Do NOT regress these back to private ctor/dtor.
- When a struct has a user-declared default constructor it is NOT an aggregate,
  so brace-init-list assignment like `tokens_ = { {"BTC","Bitcoin",1.5,"â‚ż"}, ... }`
  fails ("no match for operator= ... <brace-enclosed initializer list>"). Fix:
  add an explicit constructor taking all the initialized fields (done for
  `ConvertToken`/`ConvertPair` in `src/services/convert_service.h` and
  `Trader` in `src/services/copy_trading_service.h`). Include `<utility>` for
  `std::move` in those headers.
- `std::cerr` requires `#include <iostream>` â€” was missing in
  `src/models/wallet_models.cpp`; added. `std::remove_if` requires
  `#include <algorithm>` â€” was missing in `src/services/margin_trading_service.cpp`;
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
- Go toolchain: install on demand from go.dev (system Go is NOT installed).
  In this env: `cd /tmp && curl -sSfL https://go.dev/dl/go1.23.12.linux-amd64.tar.gz -o go.tar.gz && mkdir -p $HOME/.go-sdk && tar -C $HOME/.go-sdk -xzf go.tar.gz`,
  then `export PATH="$HOME/.go-sdk/go/bin:$PATH" && export GOPATH="$HOME/go" && export GOTOOLCHAIN=local`.
  Verify: `go version` â†’ `go1.23.12 linux/amd64`. Build any `go/<svc>` with
  `cd go/<svc> && go build ./...`. NOTE: `/usr/local` is read-only â€” install
  into `$HOME`.
- Two modules: go/multisig_service (github.com/tigerwallet/multisig-service), go/mpc (github.com/tigerwallet/mpc).
- All signing uses github.com/ethereum/go-ethereum/crypto (real ECDSA secp256k1, low-s). No sha256/sha3 fakes.
- multisig_service: broadcastTransaction (main.go) and broadcastRawTransaction (multisig_service.go) use ethclient.Dial + types.NewTx(DynamicFeeTx) + types.SignTx + client.SendTransaction. RPC from ETH_RPC_URL env.
- SignHash uses crypto.Sign(hash, privKey), v normalized to 27/28.
- mpc/enterprise.go: real Shamir + Lagrange over secp256k1; CombineShares reconstructs then crypto.Sign. Real crypto.Ecrecover verification. secp256k1 only (no P256 for crypto). The PolicyEngine now has REAL rule enforcement: daily_limit uses a per-wallet UTC-day spend counter (RecordExecution updates it); tx_limit parses the wei amount (decimal or 0x-hex); whitelist/blacklist parse comma/newline address lists and match case-insensitively; unknown rule types REJECT (fail-closed). The old fake evaluateRule (always-true daily_limit/whitelist, byte-length tx_limit, allow-fallthrough) is gone.
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
  transfer(address,uint256) â€” manual calldata (selector 0xa9059cbb + padded
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
  OTHER pages like hardware-wallet/launchpad/lending/bridge remain â€” not
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

## Android fake-crypto remediation (2026-08-10)
- Go MPC backend (go/mpc) default port is 9099 (env MPC_PORT), not 8085; wallet_api runs on 8443. Backend JSON uses base64 for publicKey/signature, ms-epoch ints for createdAt/signedAt, and 0x-hex 32-byte messageHash.
- mobile/android/.../services/mpc/MPCWalletService.java now calls only the 3 real MPC endpoints (/create, /sign, /wallet/{keyId}). keyId cached by address at create time; sign/getWalletInfo error if address unknown. SHA-256 used for messageHash with a keccak-256 TODO (real signature still from backend).
- trading/{MarginTrading,P2P,CryptoCard,Futures}Service.java: no real backend exists, so data-fabricating methods throw UnsupportedOperationException; pure-math helpers kept. Removed Math.random/Random card-PAN/CVV generators and hardcoded price arrays.


## Rust crate compile audit (2026-08-10)

- Toolchain: rustc 1.97.1 stable (minimal profile). libssl-dev / pkg-config are NOT
  installed â€” every crate depending on openssl-sys fails its build script.
  Installing libssl-dev + pkg-config is a precondition, not a code fix.
- 88 Rust crates total (no workspaces; each Cargo.toml is standalone). cargo check
  (offline first, online fallback), per-crate timeout ~240s: 23 clean, 65 fail (exit 101).
- Failure categories: (1) nonexistent dep names in Cargo.toml [timeout, msgpack, cbor,
  sha256, raft, usb, cryptography, apple-api, technical-analysis]; (2) openssl-sys build
  failures [ai_agent, starknet_sdk, substrate_sdk, zksync_sdk, fetcher_gateway,
  master_admin_management, rust/crypto, security/rust/mev_protection, services/rust/oracle,
  super_admin, white_label_admin]; (3) sqlx/zeroize/tower-http resolution failures;
  (4) manifest parse errors [missing benches, bad feature refs]; (5) missing bin targets
  [aptos_sdk, liquid_staking, rust/masterwallet_fetchers]; (6) genuine compile errors.
- Stub crate: blockchain_layer/zksync_sdk declares pub mod address/crypto/provider/
  transaction/types but those files do not exist.
- lib.rs < 25 lines: master_admin_management(8), services(8), super_admin(8),
  white_label_admin(8), ai_platform(10), api_gateway(10), backend_services(10),
  dapp_browser(10), fiat_ramp(10), security_platform(10), staking_hub(10), admin(13),
  core/rust/trading_engine(19), white_level_sdk(20), perpetuals_engine(22).
- Raw audit log: /tmp/crate_audit_results.txt (ephemeral).

## Compile-fix round (security-critical crates, real crypto only)
All 10 target crates now `cargo check` (lib) exit 0. No stubs/fakes/mocks; real
crypto crates only.
- multisig/rust: k256 0.13 (`SigningKey`/`Signer`) for threshold signing.
- security_engine/rust: serde/parking_lot/hex deps added.
- social_recovery/rust: ring 0.17 (`Ed25519KeyPair`/`ED25519`) + `InvalidAddress`
  variant; borrow/move fixes.
- blockchain_layer/solana_core/rust: real ed25519-dalek 2 verify.
- blockchain_layer/injective_sdk/rust: real secp256k1 0.24 ECDSA
  (sign_ecdsa/verify_ecdsa, `Message::from_slice`).
- blockchain_layer/sei_sdk/rust: real ed25519-dalek 2 sign/public_key/verify
  (ed25519-dalek already a dep; structs are [u8;32]/[u8;64]).
  NOTE: native Sei (Cosmos) keys are secp256k1; if strict secp256k1 is required,
  PublicKey must become 33 bytes (compressed) + custom serde. Ed25519 here is
  real & verifiable, not a fake SHA-512 hash.
- blockchain_layer/algorand_sdk/rust: real ed25519-dalek 2 sign/public_key/verify;
  from_seed keeps SHA-512/256 key derivation.
- blockchain_layer/aptos_sdk/rust: real ed25519-dalek 2 sign/verify.
- blockchain_layer/cardano_sdk/rust: real ed25519-zebra + blake2b-224 + bech32.
- blockchain_layer/near_sdk/rust: real ed25519-dalek 2 + BorshSerialize.

## Rust trading/latency/service crates compile-fix pass (2026-08-10)
All of the following pass cargo check --lib (or cargo check for bin-only):
api_gateway/rust, core/rust/{analytics_service,bot_platform_service,bridge_engine,indexer_service,intent_engine,intent_settlement,matching_engine,mm_engine,quote_engine,simulation_engine}, user_features/limit_orders/rust, fiat_ramp/rust, liquid_staking/rust, white_level_sdk/rust, admin/rust, rust/rbac_admin_backend, white_label/rust/high_speed, white_label_analytics_ai/rust, fetcher_core/rust, rust/full_fetchers, rust/high_performance_calculator, portfolio/rust, cloud_recovery/rust, hsm_integration/rust, master_wallet/rust, rust/transaction_engine.
Bin-only crates passing cargo check: launchpad_ecosystem/rust, mm_bot_platform/bot_core, white_label_templates/rust, trading_charts/rust.
Notes:
- liquid_staking/rust: added deps parking_lot/uuid/sha3 to Cargo.toml; use sha3::Digest; typo UnkakeStatus->UnstakeStatus; max_stake_amount widened to u128 (1000 ETH in wei = 1e21 overflows u64) with cast in comparison.
- transaction_engine/src/lib.rs: MultiChainTx derives Clone.
- portfolio/rust/src/lib.rs: * 100 -> * Decimal::from(100); &self.positions -> &mut self.positions.
- No stubs/todo!()/unimplemented!()/all-zero stubs introduced; real logic preserved.


## Security-primitive fixes (2026-08-09)
- go/social_recovery_service/main.go: AES-GCM key now derived via scrypt
  (N=32768,r=8,p=1) + per-ciphertext 32-byte salt (was bare sha256(passphrase)).
  Blob = salt(32)||nonce(12)||ciphertext. Build + vet clean.
- go/two_factor_auth/main.go: WebAuthn is now REAL. RegisterWebAuthn stores the
  browser-supplied SPKI P-256 public key (validates it parses). VerifyWebAuthn
  reconstructs authenticatorData||SHA256(clientDataJSON), parses the stored
  P-256 key, and verifies the ECDSA signature via ecdsa.VerifyASN1 (normalizes
  raw r||s to DER). Never returns true on a bad/missing signature. The handler
  requires `publicKey` in the register body. go.mod requires go 1.25 -> build
  with GOTOOLCHAIN=auto.
- go/mpc/enterprise.go: PolicyEngine rule enforcement is real (see Go services
  section above). RecordExecution updates the daily spend counter.

## wallet_api DeFi + tx-receipt endpoints (2026-08-09)
- go/wallet_api/defi_handlers.go (new): GET /api/v1/swap/quote (real CoinGecko
  cross-rate, honest price_impact/gas=0 indicative), POST /api/v1/swap/execute
  (returns on-chain action for /send, no fabricated hash), GET /api/v1/staking/quote
  (supported assets, APY=0 honestly), POST /api/v1/staking/{stake,unstake,claim}
  (route to /send), GET /api/v1/transactions/:txHash (real explorer proxy).
- go/wallet_api/amm_router.go (NEW, on-chain AMM): REAL `getAmountsOut` via
  `eth_call` to per-chain Uniswap-V2-compatible routers (Ethereum
  `0x7a250d56â€¦`, PancakeSwap BSC, QuickSwap Polygon, SushiSwap
  Arbitrum/Optimism, Base). REAL `swapExactTokensForTokens` calldata
  construction (selector `0x18cbafe5`, exact ABI encoding). Real on-chain
  `decimals()` per token for human<->wei. 0.5% slippage default from the
  live `getAmountsOut`. `GET /api/v1/amm/quote` + `POST /api/v1/amm/swap`
  (constructs calldata; client broadcasts via real `/api/v1/send`). 503 on
  RPC failure â€” never fabricates. 8 Go tests (byte-exact selector + encoding,
  decode roundtrip, short-return reject, router resolution, humanToWei +
  zero/negative reject). `go build` + `go vet` clean. Frontend parity:
  `app/api/v1/amm/{quote,swap}/route.ts` (wallet_api auth group), `tsc` clean.

## Mobile wallet-creation password contract (2026-08-09)
- The wallet backend requires a `password` (min 8) + `label` to create a wallet.
  iOS (mobile_apps/ios_app/TigerWallet/Models/Wallet.swift) createWallet/importWallet
  now thread `password`; backendCreateWalletMnemonic sends {label,password}.
  Android (Services.kt + WalletRepository.kt) generateMnemonic(password) +
  createWallet(name,password,mnemonic) (suspend). Flutter is unaffected: real
  on-device BIP-39/32/44 + flutter_secure_storage (self-custody, no backend pw).

## Frontend (web_nextjs) â€” light/dark theme switching

- `ThemeProvider` at `frontend/web_nextjs/app/components/ThemeProvider.tsx` exposes `useTheme()` returning `{ theme, isDark, colors, ... }`.
- Reference implementation: `app/convert/page.tsx` (zero `dark:` Tailwind variants; uses `isDark ? '...' : '...'` ternaries).
- Themed page pattern (all pages under `app/<route>/page.tsx`):
  1. `import { useTheme } from '../components/ThemeProvider';`
  2. `const { isDark } = useTheme();` near the top of the component body (after existing `useState`/`useEffect` hooks).
  3. Root container: `isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'` (keep layout classes like `min-h-screen`, `p-6`).
  4. Convert `dark:bg-*`/`dark:text-*`/`dark:border-*` Tailwind variants into `isDark ? 'dark-classes' : 'light-classes'` ternaries. Light-mode equivalents: bg â†’ `bg-white border border-gray-200`; text â†’ `text-gray-900`/`text-gray-600`; secondary text â†’ `text-gray-500`; border â†’ `border-gray-200`/`border-slate-200`.
- Pages already converted (all under `app/<route>/page.tsx`): themes, tx-simulation, gas-estimation, approvals, address_book, nft-marketplace, bug-bounty, defi, staking, token_sale, ieo, master_wallet, copy_trading, swap, walletconnect, fiat-ramps, price-feeds, protection-fund, kyc, history, social_recovery, gift_cards, biometric, insurance_fund, widgets, dao, perpetual, notifications_center, launchpool, fiat_onramp, i18n. Verify with `grep -rn "dark:" app/` (should be 0 in themed pages). Full `npx tsc --noEmit` passes with 0 errors after `npm install`.
- NOTE: `approvals/page.tsx` `RISK_COLORS` badges are intentionally light-tinted and not theme-dependent.

## Frontend (web_nextjs) â€” perpetual/margin_trading/token_sale/dao real backend (2026-08-11)

Converted four `app/<route>/page.tsx` pages from MOCK_* consts to real wallet_api
(:8443) PostgreSQL-backed fetch calls, following the launchpool/approvals/defi
`fetchAPI` pattern. All four pass `npx tsc --noEmit` (only pre-existing unrelated
axios errors in `src/lib/api/client.ts` remain).

Common helper added to each file:
```ts
const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');
const fetchAPI = async <T,>(endpoint, options?) => { ... returns response.data; Bearer localStorage 'tigerwallet-token'; }
```
- `import React, { useState, useEffect, useCallback } from 'react'`.
- `loading`/`error` state + UI banners; fail-closed empty arrays; useEffect loads
  on mount via useCallback.

Per-page:
- **perpetual/page.tsx**: removed `MOCK_POSITIONS`; GET `/perpetual/positions`;
  POST `/perpetual/positions` (create) + POST `/perpetual/positions/:id/close`
  replaced `setTimeout` simulators.
- **margin_trading/page.tsx**: `MOCK_PAIRS` (BTC/USDT, ETH/USDT â€” chain/reference
  config, NOT user data) renamed to local const `TRADING_PAIRS` (kept, not from
  backend); positions now come from GET `/margin/positions`; POST create +
  POST `/margin/positions/:id/close` are real fetchAPI calls.
- **token_sale/page.tsx**: removed `MOCK_SALES`; GET `/token-sales`; POST
  `/token-sales/:id/participate` real. Backend snake_case mapped to frontend
  camelCase interface (token_name/token_symbol/contract_address/chain_id/
  price_per_token/total_supply/sold_amount/min_allocation/max_allocation/
  start_time/end_time/status/description/website â†’ existing fields).
- **dao/page.tsx**: removed `MOCK_PROPOSALS` + `MOCK_DELEGATES`; GET
  `/dao/proposals` + GET `/dao/delegates`; POST `/dao/proposals` (create) +
  POST `/dao/proposals/:id/vote` real. Backend snake_case mapped to frontend
  camelCase (for_votes/against_votes/abstain_votes/start_time/end_time/executed/
  proposer/proposer_name). Stats block uses real data. Vote modal submit
  disabled while submitting/empty.
- JSX gotcha fixed in dao: nested ternary `{loading ? ... : empty ? ... : (
  arr.map(...) )}` â€” the `.map` branch MUST close with `})` then `)}` (arrow
  body + map call, then ternary `)` + JSX `}`); a stray extra `)` causes
  TS1005/TS1381.

### Pitfalls observed during conversion
- A half-converted file may already reference `isDark` in JSX but be **missing the import + hook declaration** â†’ compiles to "isDark is not defined". Always confirm both `import { useTheme }` and `const { isDark } = useTheme();` exist (token_sale and ieo hit this).
- When converting a static multi-class `className` to a template literal, keep ALL original classes inside the backticks; accidentally leaving trailing classes (e.g. `px-4 py-3`) outside the closing backtick produces an unbalanced template literal (syntax error). Verify backtick count is even after edits (gas-estimation error banner hit this).
- `master_wallet/page.tsx` contains REAL BIP-39/AES-GCM/PBKDF2/WebCrypto logic â€” only change its colors, never its crypto code.


## Go toolchain + real-NFT-fetcher + keystore V3 (2026-08-11)

- **Go toolchain install location**: The official Go tarball must be installed
  under `$HOME` (NOT `/usr/local`, which is read-only in this env). Working
  install: `cd /tmp && curl -sSfL https://go.dev/dl/go1.23.12.linux-amd64.tar.gz
  -o go.tar.gz && mkdir -p $HOME/.go-sdk && tar -C $HOME/.go-sdk -xzf go.tar.gz`,
  then `export PATH="$HOME/.go-sdk/go/bin:$PATH" && export GOPATH="$HOME/go" &&
  export GOTOOLCHAIN=local`. Verify: `go version` â†’ `go1.23.12 linux/amd64`.
- **go-ethereum version pinning**: go-ethereum v1.17.5 (latest) requires
  Go >= 1.24.0; with Go 1.23.12 you MUST pin `go-ethereum@v1.13.15` (the same
  version `go/wallet_api` uses) or `go get`/`go mod tidy` fail with "requires
  go >= 1.24.0 (running go 1.23.12; GOTOOLCHAIN=local)".
- **go/nft_service is the canonical NFT service**: it is wired into
  `docker-compose.yml` (port 8085), unlike the orphan `go/nft/` and
  `go/nft_prices/` dirs. `go/nft_service/fetcher.go` (new) does REAL on-chain
  ERC-721 reads via go-ethereum `ethclient` (balanceOf/tokenOfOwnerByIndex/
  ownerOf/tokenURI/name/symbol/totalSupply via `eth_call`), with HTTP metadata
  fetch + `ipfs://`â†’gateway resolution + Redis cache (60s TTL, capped 200
  tokens/owner). If `ETH_RPC_URL` is unset, `GetUserNFTs?contract=` returns 503
  "unavailable" â€” NEVER fabricates data. The old `initializeDefaultData()`
  (mock BAYC/CryptoPunks/Azuki/DeGods with `Owner:"0x000"`) was REMOVED;
  service starts empty. `NFT` struct has NO `Standard` field (only
  `NFTCollection` does) â€” don't set it in fetcher literals. `cfg.Port` is a
  string. go build + go vet clean.
- **go/wallet_api/keystore_v3.go (new)**: real Web3 Secret Storage V3 (scrypt
  variant) â€” `ExportKeystoreV3`/`ImportKeystoreV3`. scrypt N=1<<18/r=8/p=1/
  dklen=32, AES-128-CTR, MAC = keccak256(dk[16:32]â€–ciphertext), constant-time
  MAC compare, v4 UUID. 2 tests pass (round-trip + wrong-password MAC failure).
  REST: `POST /api/v1/keystore/{export,import}` (AuthMiddleware-protected) +
  Next.js proxy routes `app/api/v1/keystore/{export,import}/route.ts` (use
  `../../_proxy` import path, NOT `../_proxy`, because they're one dir deeper
  than the top-level v1 routes). handlers.go needed `crypto` (go-ethereum)
  import added. The Wallet persistence struct is `WalletRecord` (not `Wallet`)
  with `ChainID int64` (not `ChainType`); creation method is `store.SaveWallet`
  (not `CreateWallet`); `SaveWallet` assigns the UUID itself.
- **Frontend proxy route depth**: top-level `app/api/v1/<x>/route.ts` uses
  `../_proxy`; nested `app/api/v1/<a>/<b>/route.ts` uses `../../_proxy`. The
  `_proxy.ts` lives at `app/api/v1/_proxy.ts`. `proxyMutation(req, path, method)`
  for POST/PUT/DELETE; `proxyGet(req, path)` for GET.

## UserWallet gap-closure (2026-08-11)

- **rust/userwallet_fetchers** now has **22 fetchers** (was 21, all fail-closed).
  9 wallet-api (balance, transactions, tokens, nfts, gas, price, swap, staking,
  dapps) + **8 REAL DeFi-service fetchers** added via a multi-service
  `UserWalletClient::service_get()` that mirrors the Next.js `_proxy.ts` service
  map: lending→:8009, copy_trading→:8006, dao→:8454, futures→:8464,
  margin→:8464 (perpetual covers margin), prediction→:8455, nft_trading→:8085,
  fiat_ramp→:8008. 5 remain HONEST fail-closed (bridge=no HTTP server —
  go/bridge is a library; options/p2p/gift_card/price_alerts=no service). The
  `services` HashMap field is populated by `default_service_urls()` in
  `fetchers.rs`. `block_on(async {...})?` — the `?` is REQUIRED because
  `block_on` returns `Result<F::Output, String>` (double-wrapped). `cargo test`
  3/3 pass. NEVER regress the wired fetchers to `UnavailableFetcher`.
- **mobile/flutter** now has a `pubspec.yaml` (deps: http, crypto, path_provider,
  provider, shared_preferences — the 5 packages its lib imports). Flutter SDK
  is NOT installed in this env; the pubspec makes it buildable where Flutter is
  present. `AppConstants.baseUrl` default changed from `api.tigerwallet.io` →
  `http://localhost:8443` (configurable via `--dart-define=API_BASE_URL=...`),
  matching the canonical wallet_api. `wallet_service.dart` already calls real
  `/api/v1/auth/*`, `/wallets`, `/send`, `/sign`, `/transactions` on :8443.
- **user_wallet/production/react** retargeted :8080→:8443. `AuthService.ts` +
  `WalletService.ts` REWRITTEN to the canonical wallet_api flat contract (NOT
  the nested `/wallets/:id/send` RESTful design — that 404s against wallet_api).
  Real routes: `/auth/login`, `/auth/register`, `/wallets`, `/public/balance`,
  `/send`, `/sign`, `/swap/quote`, `/gas`, `/transactions`, `/nfts`,
  `/staking/*`. Features the backend doesn't expose (bridges, nft/transfer,
  dapp/connect, 2FA, refresh tokens, password reset, sessions) throw real
  errors — never fake success. Added `tsconfig.json`. The two service files
  compile clean (0 errors); the 35 remaining `tsc` errors are all in the
  PRE-EXISTING `src/services/master/*` files (HardwareWalletService etc.),
  outside UserWallet scope. Both files export BOTH named `{ AuthService }`/
  `{ WalletService }` AND default (contexts use named imports).
- **frontend/web_nextjs/app/wallet/lib/transactions.ts** rewritten: all 9
  "unavailable until the canonical Rust wallet-core bridge is configured"
  stubs now delegate to the backend via Next.js proxy routes (`/send`, `/sign`,
  `/transactions`, `/swap/quote`, `/gas`, `/bridge/quote`, `/lending/markets`).
  Proxy route bug fixed: `proxyGet(req, '/wallet/transactions')` →
  `proxyGet(req, '/transactions')`.
- **Theme switching verified on EVERY app**: web (ThemeProvider sets
  `data-theme` on documentElement + CSS vars in theme.css), desktop (same),
  iOS (ThemeManager @StateObject + preferredColorScheme + Toggle in SettingsView),
  Android (AppCompatDelegate.setDefaultNightMode), extension (data-theme attr +
  chrome.storage), Flutter (ThemeProvider ChangeNotifier), Next.js (isDark
  ternaries, 0 dark: variants in themed pages). All apply theme globally, not
  per-page.
- **Production-vs-dev target split (now clean)**: dev frontends (`user_wallet/*`,
  `frontend/web_nextjs`) → `localhost:8443` (canonical wallet_api); production
  frontends (`desktop_app`, `browser_extensions/chrome`) previously pointed at
  `api.tigerwallet.com` BUT origin/main commit `041bb49` "Repo-wide: remove ALL
  fake api.tigerwallet.com backend URLs" removed those — so all clients now use
  the real canonical backend. No orphan/stub targets remain.
- **Go DeFi services with main.go (real HTTP servers)**: lending_service
  (:8009, real Aave V3, group `/api/v1/lending`), copy_trading_service (:8006,
  `/api/v1/copytrading`), governance_service (:8454, `/api/v1/governance`),
  perpetual_service (:8464, `/api/v1/perpetual`, covers futures+margin),
  prediction_service (:8455, `/api/v1/prediction`), nft_prices (:8085
  canonical nft_service, `/api/v1/nft`), fiat (:8008), fiat_ramp (:8008,
  `/api/v1/ramp`). `bridge`/`red_packets_service`/`nft` (the dir, not
  nft_service) have NO main.go — they're libraries.

## Session 2026-08-11: UserWallet fake-crypto removal + theme + service builds

### Fake crypto / Math.random elimination (COMPLETE)
- **0 actual `Math.random()` calls remain** across all client code. All
  remaining `Math.random` mentions are comments. Key fixes: user_app/react
  LoginPage (fake 24-word mnemonic -> backend POST /wallets real BIP-39);
  walletApi.createWallet sends {label,password,chain_id,entropy_bits};
  CrossChainIntentRouter fabricated tx hash -> backend /swap/execute;
  user_wallet/production/react LoginPage dead generateMnemonic removed;
  mobile/tigerswap-wallet fabricated address -> backend /wallets;
  mobile_apps/tigerwallet device id -> crypto.getRandomValues + storage;
  DAppBrowser fake tx hashes/sigs -> honest throws; master_wallet extensions
  biometric throw + CSPRNG shuffle/random; blockchain_explorer mock blocks
  -> real JSON-RPC eth_getBlockByNumber; trading_terminal fabricated price
  -> backend; bitcoin_ordinals simulated inscription -> backend /ordinals/inscribe.
- Deleted unreferenced stub frontend/extensions/chrome.

### Next.js wallet lib/transactions.ts (EVM fully wired)
- EVM path (createTransaction, estimateGas, getTransactionReceipt,
  swap.findBestRoute/executeSwap, masterWallet.autoSign) all delegate to
  wallet_api via same-origin proxy routes. Solana/Bitcoin are honest
  fail-closed throws (not stubs).
- Created dynamic route app/api/v1/transactions/[txHash]/route.ts (import
  path `../../_proxy`; proxyGet auto-appends search params).

### Light/dark theme (web_nextjs: 0 dark: variants)
- All 5 remaining pages (passkey, biometric-auth, gas-tracker, app/page,
  login/page) converted to useTheme() + isDark ternaries. grep -rln "dark:"
  app/ -> 0 files. Mobile: Android ThemeManager.kt, iOS ThemeManager.swift,
  Flutter theme_provider.dart all exist.

### docker-compose Go service build-fix (3 services)
- permission_service, connection_api, monitoring_dashboard build + vet clean.
  go.mod/go.sum at SERVICE ROOT. docker-compose contexts retargeted to service
  roots with dockerfile: go/Dockerfile; Dockerfiles build ./go/cmd.
- permission_service: SHA-256 password hashing -> bcrypt; redis import ->
  redislib. connection_api: fixed unused jwt + missing SessionInfo fields.
  go mod tidy -e (transitive test dep needs Go>=1.25; not runtime). fetcher_gateway/rust
  still fails locally (openssl-sys/pkg-config missing; Docker build works).

### rust/userwallet_fetchers (FIXED, builds clean)
- cargo check --lib exit 0. Has Cargo.toml. Delegates ALL fetchers to wallet_api
  (:8443) via pooled async reqwest::Client. No stubs; fail-closed (Err).

### user_wallet/* client wiring (all verified -> :8443)
- web, desktop, ios, android, production/react all target :8443 with correct
  routes. Route mismatches fixed. mobile_apps/flutter_app + mobile/flutter
  have pubspec.yaml. user_wallet/android compiles.
- production/react: DAppsPage previously hardcoded a `popularDApps` list (fake
  data). Replaced with WalletService.getDapps()/getDappCategories() hitting
  GET /dapps + /dapps/categories. Added those methods to WalletService.ts.
  SwapPage + StakingPage already used real WalletService calls (getSwapQuote/
  swap/getStakingPositions/stake/claimRewards) — no fake data. production/
  react remaining tsc errors are all pre-existing & out-of-scope (services/
  master/* = MasterWallet product; missing Header/Sidebar/LoadingSpinner
  components; NFTsPage/Home/SendPage).

