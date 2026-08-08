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
