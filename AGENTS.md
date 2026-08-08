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
