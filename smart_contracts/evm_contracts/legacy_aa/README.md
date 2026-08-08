# Legacy Account Abstraction Contracts

This directory holds TigerWallet's **custom, pre-existing** account-abstraction
contracts that are **not** part of the canonical ERC-4337 reference stack and
are **not** compiled by the Foundry project in `../foundry.toml`
(`src = "account_abtraction"`).

## Why these were moved out of `account_abstraction/`

`AccountFactory.sol` was written against TigerWallet's *old custom*
`EntryPoint.sol`, which defined an unpacked `UserOperation` struct and an
on-chain `Bundler` contract. Both concepts were removed when the repo adopted
the **canonical, audited ERC-4337 EntryPoint** from eth-infinitism
(`account_abstraction/EntryPoint.sol`), which uses the packed
`PackedUserOperation` format and treats bundlers as off-chain components.

As a result `AccountFactory.sol` no longer compiles as-is. It has been
preserved here rather than deleted.

## What needs to happen before this code can be used

To revive `AccountFactory.sol` against the canonical stack it must be ported to:

- `PackedUserOperation` (see `account_abstraction/interfaces/PackedUserOperation.sol`)
  and `UserOperationLib` (`account_abstraction/UserOperationLib.sol`) for field
  access, instead of the unpacked `UserOperation` struct.
- The canonical `IEntryPoint` interface
  (`account_abstraction/interfaces/IEntryPoint.sol`) for paymaster/account hooks.
- Extend `BaseAccount` / `BasePaymaster` rather than reimplementing stake logic
  (the canonical `StakeManager` already provides deposit/stake/unstake).

Do **not** drop this directory back into `account_abstraction/` without
completing that port — it would break `forge build`.
