// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/* TigerWallet — ported, canonical ERC-4337 account.
 *
 * This replaces the legacy `legacy_aa/AccountFactory.sol` whose `SimpleAccount`
 * validated signatures by `signature.length == 64` (accept-anything) and used the
 * pre-packed `UserOperation` struct, so it did not compile against the canonical
 * eth-infinitism EntryPoint. This contract extends the audited `BaseAccount`
 * (which implements the `onlyEntryPoint` guard, nonce delegation, and
 * execute/executeBatch) and adds real owner-based EIP-173 ownership + real
 * ECDSA signature validation via OpenZeppelin v5 `ECDSA` /
 * `MessageHashUtils.toEthSignedMessageHash`. No accept-anything signature check.
 */

import {BaseAccount} from "./BaseAccount.sol";
import {IEntryPoint} from "./interfaces/IEntryPoint.sol";
import {PackedUserOperation} from "./interfaces/PackedUserOperation.sol";
import {UserOperationLib} from "./UserOperationLib.sol";
import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";
import {Initializable} from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

contract SimpleAccount is BaseAccount, Initializable {
    using ECDSA for bytes;
    using MessageHashUtils for bytes32;

    // SIG_VALIDATION_FAILED from IAccount semantics (uint256(1)).
    uint256 internal constant SIG_VALIDATION_FAILED = 1;

    address public owner;

    event SimpleAccountInitialized(address indexed entryPoint, address indexed owner);
    event OwnerChanged(address indexed oldOwner, address indexed newOwner);

    error NotOwner(address sender);

    /// @dev Factory clones this implementation; the constructor only guards
    ///      against direct (non-proxy) deployment so the implementation cannot
    ///      be initialized by an attacker that front-runs a clone deploy.
    constructor(IEntryPoint anEntryPoint) {
        if (address(anEntryPoint) == address(0)) revert NotOwner(address(0));
        _entryPoint = anEntryPoint;
        _disableInitializers();
    }

    /// @notice Initializer called by the factory right after the clone is created.
    function initialize(address anOwner) external virtual initializer {
        owner = anOwner;
        emit SimpleAccountInitialized(address(entryPoint()), anOwner);
    }

    /// @inheritdoc BaseAccount
    function entryPoint() public view virtual override returns (IEntryPoint) {
        return _entryPoint;
    }

    IEntryPoint internal immutable _entryPoint;

    /// @notice Returns the next sequential nonce for the default key.
    function getNonce() public view virtual override returns (uint256) {
        return entryPoint().getNonce(address(this), 0);
    }

    /// @inheritdoc BaseAccount
    /// @dev Real ECDSA validation of the EIP-191-prefixed userOpHash against
    ///      the stored owner. A 65-byte ECDSA signature is expected; anything
    ///      else or a recover mismatch returns SIG_VALIDATION_FAILED (1).
    function _validateSignature(
        PackedUserOperation calldata userOp,
        bytes32 userOpHash
    ) internal virtual override returns (uint256 validationData) {
        bytes memory sig = userOp.signature;
        if (sig.length != 65) {
            return SIG_VALIDATION_FAILED;
        }
        address recovered = ECDSA.recover(MessageHashUtils.toEthSignedMessageHash(userOpHash), sig);
        if (recovered != owner) {
            return SIG_VALIDATION_FAILED;
        }
        return 0;
    }

    /// @notice Owner-gated execution (for direct, non-4337 calls). The 4337
    ///         path routes through the inherited BaseAccount.execute which is
    ///         already gated by `_requireFromEntryPoint`.
    function execute(address dest, uint256 value, bytes calldata func) external override {
        if (msg.sender != address(entryPoint()) && msg.sender != owner) revert NotOwner(msg.sender);
        _call(dest, value, func);
    }

    /// @notice Owner-gated batched execution.
    function executeBatch(address[] calldata dest, uint256[] calldata values, bytes[] calldata funcs) external {
        if (msg.sender != address(entryPoint()) && msg.sender != owner) revert NotOwner(msg.sender);
        require(dest.length == funcs.length, "wrong length");
        for (uint256 i = 0; i < dest.length; i++) {
            _call(dest[i], values.length > 0 ? values[i] : 0, funcs[i]);
        }
    }

    function _call(address dest, uint256 value, bytes memory func) internal {
        (bool ok, bytes memory result) = dest.call{value: value}(func);
        if (!ok) {
            assembly ("memory-safe") {
                revert(add(result, 0x20), mload(result))
            }
        }
    }

    /// @notice Transfer ownership (EIP-173-ish, owner-gated).
    function transferOwnership(address newOwner) external {
        if (msg.sender != owner) revert NotOwner(msg.sender);
        if (newOwner == address(0)) revert NotOwner(newOwner);
        address old = owner;
        owner = newOwner;
        emit OwnerChanged(old, newOwner);
    }

    /// @notice Withdraw the account's EntryPoint deposit back to the owner.
    function withdrawDepositTo(address payable withdrawAddress) external {
        if (msg.sender != owner) revert NotOwner(msg.sender);
        entryPoint().withdrawTo(withdrawAddress, entryPoint().balanceOf(address(this)));
    }

    receive() external payable {}
}
