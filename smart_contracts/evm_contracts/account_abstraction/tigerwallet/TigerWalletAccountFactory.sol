// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/**
 * @title TigerWalletAccountFactory
 * @notice ERC-4337 account factory that deploys EOA-owned smart accounts with
 *         CREATE2-deterministic addresses. Produces `initCode`-compatible
 *         factory calldata so the canonical EntryPoint can deploy accounts on
 *         demand via its SenderCreator.
 *
 * This replaces the legacy `legacy_aa/AccountFactory.sol` which was built
 * against the old unpacked `UserOperation` struct and validated signatures by
 * `signature.length == 64`. It now uses the canonical `PackedUserOperation`
 * stack and real ECDSA signature validation (EIP-191 over the userOpHash).
 */
import "@openzeppelin/contracts/utils/Create2.sol";
import "./TigerWalletAccount.sol";
import "../interfaces/IEntryPoint.sol";

contract TigerWalletAccountFactory {
    event AccountCreated(address indexed account, address indexed owner, uint256 indexed key);

    IEntryPoint public immutable entryPoint;
    TigerWalletAccount public immutable accountImplementation;

    constructor(IEntryPoint _entryPoint) {
        require(address(_entryPoint) != address(0), "entryPoint=0");
        entryPoint = _entryPoint;
        accountImplementation = new TigerWalletAccount(_entryPoint, address(this));
    }

    /**
     * @notice Create an account deterministically for `(owner, key)`.
     *         Called by the EntryPoint's SenderCreator as the `factory` portion
     *         of `initCode` (first 20 bytes), receiving the trailing calldata
     *         `createAccount(address,uint256)`.
     */
    function createAccount(address owner, uint256 key) public returns (TigerWalletAccount account) {
        bytes32 salt = _salt(owner, key);
        bytes memory initCode = _initCode(owner);

        address predicted = Create2.computeAddress(salt, keccak256(initCode), address(this));

        if (predicted.code.length > 0) {
            return TigerWalletAccount(payable(predicted));
        }

        account = TigerWalletAccount(payable(Create2.deploy(0, salt, initCode)));
        emit AccountCreated(address(account), owner, key);
    }

    /**
     * @notice Counterfactual address computation (no deploy).
     */
    function getAccountAddress(address owner, uint256 key) external view returns (address) {
        return Create2.computeAddress(_salt(owner, key), keccak256(_initCode(owner)), address(this));
    }

    function _salt(address owner, uint256 key) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(owner, key));
    }

    function _initCode(address owner) internal view returns (bytes memory) {
        return abi.encodePacked(
            type(TigerWalletAccount).creationCode,
            abi.encode(entryPoint, owner)
        );
    }
}
