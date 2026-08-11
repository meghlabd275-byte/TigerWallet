// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/**
 * @title TigerWalletPaymaster
 * @notice ERC-4337 paymaster that sponsors gas for whitelisted senders. Extends
 *         the canonical `BasePaymaster`. Replaces the legacy `Paymaster` in
 *         `legacy_aa/AccountFactory.sol` which used the old unpacked
 *         `UserOperation` struct and an incorrect on-chain stake map that never
 *         interacted with the EntryPoint's real stake/deposit accounting.
 *
 * Sponsorship policy: an owner-configurable set of senders is gasless. The
 * paymaster draws from its real EntryPoint deposit (funded via `deposit()`).
 * Non-whitelisted senders are rejected (fail-closed).
 */
import "@openzeppelin/contracts/access/Ownable.sol";
import "../BasePaymaster.sol";
import "../UserOperationLib.sol";

contract TigerWalletPaymaster is BasePaymaster {
    using UserOperationLib for PackedUserOperation;

    error SenderNotWhitelisted(address sender);
    error InsufficientDeposit(uint256 available, uint256 required);

    mapping(address => bool) public whitelistedSender;
    uint256 public constant MAX_SPONSORED_COST = 0.01 ether; // per-op cap

    event WhitelistUpdated(address indexed sender, bool status);

    constructor(IEntryPoint _entryPoint, address _owner) BasePaymaster(_entryPoint, _owner) {}

    function setWhitelistedSender(address sender, bool status) external onlyOwner {
        whitelistedSender[sender] = status;
        emit WhitelistUpdated(sender, status);
    }

    /// @inheritdoc BasePaymaster
    function _validatePaymasterUserOp(
        PackedUserOperation calldata userOp,
        bytes32 userOpHash,
        uint256 maxCost
    ) internal override returns (bytes memory context, uint256 validationData) {
        (userOpHash);
        if (!whitelistedSender[userOp.sender]) {
            revert SenderNotWhitelisted(userOp.sender);
        }
        if (maxCost > MAX_SPONSORED_COST) {
            revert InsufficientDeposit(0, maxCost);
        }

        // Context carries the sponsored sender so postOp can record spend.
        return (abi.encode(userOp.sender), 0);
    }

    /// @inheritdoc BasePaymaster
    function _postOp(
        PostOpMode mode,
        bytes calldata context,
        uint256 actualGasCost,
        uint256 actualUserOpFeePerGas
    ) internal override {
        (mode, context, actualGasCost, actualUserOpFeePerGas);
        // No additional post-processing; the EntryPoint deducts actualGasCost
        // from this paymaster's deposit automatically.
    }
}
