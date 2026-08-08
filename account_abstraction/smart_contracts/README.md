# Smart Contracts (Reference)

The custom ERC-4337 smart contracts that previously lived in this directory
(`EntryPoint.sol`, `SmartAccount.sol`, `AccountFactory.sol`, `Paymaster.sol`)
were custom reimplementations of the ERC-4337 stack and have been removed.

TigerWallet now uses the **canonical, audited ERC-4337 reference implementation**
from the ERC-4337 team (eth-infinitism), which has been audited by OpenZeppelin
and is used in production.

## Canonical location

All ERC-4337 core contracts, interfaces, and utilities now live in:

```
smart_contracts/evm_contracts/account_abstraction/
```

That directory is a Foundry source root and contains:

- `EntryPoint.sol`, `EntryPointSimulations.sol`, `SenderCreator.sol`,
  `StakeManager.sol`, `NonceManager.sol`, `UserOperationLib.sol`,
  `Helpers.sol`, `Eip7702Support.sol`, `Stakeable.sol`,
  `BaseAccount.sol`, `BasePaymaster.sol`
- `interfaces/` — `IEntryPoint.sol`, `IAccount.sol`, `IAccountExecute.sol`,
  `IAggregator.sol`, `IPaymaster.sol`, `INonceManager.sol`, `IStakeManager.sol`,
  `ISenderCreator.sol`, `IEntryPointSimulations.sol`, `PackedUserOperation.sol`
- `utils/` — `Exec.sol`

## Source

Fetched from:
https://github.com/eth-infinitism/account-abstraction (branch: `develop`)

Do not re-add custom EntryPoint/SmartAccount/Paymaster implementations here.
