// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/* TigerWallet — ERC-4337 account factory.
 *
 * Deploys deterministic (counterfactual) EIP-1167 minimal-proxy clones of
 * `SimpleAccount` using OpenZeppelin v5 `Clones.cloneDeterministic`. The
 * counterfactual address is computed off-chain the same way via
 * `Clones.predictDeterministicAddress`, so a client can show the account
 * address before it is deployed. Replaces the legacy `legacy_aa/
 * AccountFactory.sol` which used a hand-rolled CREATE2 calc and a broken
 * `initCode` assembly.
 */

import {Clones} from "@openzeppelin/contracts/proxy/Clones.sol";
import {SimpleAccount} from "./SimpleAccount.sol";
import {IEntryPoint} from "./interfaces/IEntryPoint.sol";

contract AccountFactory {
    /// The canonical `SimpleAccount` implementation (deployed once, then cloned).
    SimpleAccount public immutable accountImplementation;

    event AccountCreated(address indexed account, address indexed owner, bytes32 salt);

    error AccountAlreadyDeployed(address account);

    constructor(IEntryPoint anEntryPoint) {
        accountImplementation = new SimpleAccount(anEntryPoint);
    }

    /// @notice Create (or return the existing) account for `owner` + `salt`.
    /// @dev `salt` is combined with `owner` before being used as the CREATE2
    ///      salt, so two different owners using the same salt get different accounts.
    /// @return account The deployed account address.
    function createAccount(address owner, uint256 salt) external returns (address account) {
        bytes32 resolvedSalt = keccak256(abi.encode(owner, salt));
        account = getAddress(owner, salt);
        if (account.code.length > 0) {
            return account;
        }
        SimpleAccount cloned = SimpleAccount(payable(Clones.cloneDeterministic(address(accountImplementation), resolvedSalt)));
        cloned.initialize(owner);
        emit AccountCreated(account, owner, bytes32(salt));
    }

    /// @notice Counterfactual address for `owner` + salt (matches createAccount).
    function getAddress(address owner, uint256 salt) public view returns (address) {
        bytes32 resolvedSalt = keccak256(abi.encode(owner, salt));
        return Clones.predictDeterministicAddress(
            address(accountImplementation),
            resolvedSalt
        );
    }
}
