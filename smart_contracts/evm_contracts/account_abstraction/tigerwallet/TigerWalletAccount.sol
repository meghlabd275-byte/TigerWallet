// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/**
 * @title TigerWalletAccount
 * @notice ERC-4337 smart account owned by a single EOA. Extends the canonical
 *         `BaseAccount` and validates UserOp signatures with real ECDSA
 *         (secp256k1) recovery over the EIP-191-prefixed userOpHash.
 *
 * Replaces the legacy `SimpleAccount` in `legacy_aa/AccountFactory.sol`, which
 * only checked `signature.length == 64` and did NOT recover or verify any
 * signer. This contract performs a real ecrecover against the configured owner.
 */
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "../BaseAccount.sol";
import "../interfaces/IEntryPoint.sol";

contract TigerWalletAccount is BaseAccount {
    using ECDSA for bytes32;

    error NotOwner(address caller, address owner);

    IEntryPoint internal immutable _entryPoint;
    address public owner;

    constructor(IEntryPoint entryPoint_, address _owner) {
        require(address(entryPoint_) != address(0), "entryPoint=0");
        require(_owner != address(0), "owner=0");
        _entryPoint = entryPoint_;
        owner = _owner;
    }

    /// @inheritdoc BaseAccount
    function entryPoint() public view override returns (IEntryPoint) {
        return _entryPoint;
    }

    /// @inheritdoc BaseAccount
    function _validateSignature(
        PackedUserOperation calldata userOp,
        bytes32 userOpHash
    ) internal override returns (uint256 validationData) {
        // userOpHash is the EIP-191 prefixed hash produced by EntryPoint.
        address signer = userOpHash.recover(userOp.signature);
        if (signer != owner) {
            return SIG_VALIDATION_FAILED;
        }
        return 0;
    }

    /**
     * @notice Allow the owner to execute arbitrary calls directly (bypassing
     *         the EntryPoint), e.g. for rescue or governance. Guarded by owner.
     */
    function executeFromOwner(address target, uint256 value, bytes calldata data)
        external
        returns (bool ok, bytes memory result)
    {
        if (msg.sender != owner) revert NotOwner(msg.sender, owner);
        (ok, result) = target.call{value: value}(data);
        if (!ok) {
            assembly ("memory-safe") {
                revert(add(result, 0x20), mload(result))
            }
        }
    }

    /**
     * @notice Transfer ownership of the smart account to a new EOA. Only the
     *         current owner may do this (directly, not via EntryPoint).
     */
    function transferOwnership(address newOwner) external {
        if (msg.sender != owner) revert NotOwner(msg.sender, owner);
        require(newOwner != address(0), "newOwner=0");
        address prev = owner;
        owner = newOwner;
        emit OwnershipTransferred(prev, newOwner);
    }

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    receive() external payable {}
}
