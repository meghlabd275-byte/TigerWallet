// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/interfaces/IERC1271.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "@openzeppelin/contracts/utils/cryptography/SignatureChecker.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title TigerAccountAbstraction
 * @notice EIP-4337 Account Abstraction Implementation
 * @dev Supports UserOperations from EntryPoint with flexible validation
 */
contract TigerAccountAbstraction is IERC1271, Ownable {
    using ECDSA for bytes32;

    // Constants
    uint256 public constant VALIDATION_SUCCESS = 0;
    uint256 public constant VALIDATION_FAILED = 1;

    // Events
    event ExecutedUserOp(
        bytes32 indexed userOpHash,
        address indexed sender,
        uint256 nonce,
        bool success
    );

    event OwnershipTransferred(
        address indexed previousOwner,
        address indexed newOwner
    );

    // State
    address public entryPoint;
    uint256 public nonce;
    mapping(bytes32 => bool) public executedOps;
    mapping(address => bool) public authorized;

    // Modifiers
    modifier onlyEntryPoint() {
        require(msg.sender == entryPoint, "AA30: not from EntryPoint");
        _;
    }

    constructor(address _entryPoint, address _owner) Ownable(_owner) {
        entryPoint = _entryPoint;
        authorized[_owner] = true;
    }

    /**
     * @notice Validate user operation (called by EntryPoint)
     * @dev Must implement IAccount interface
     */
    function validateUserOp(
        bytes calldata userOp,
        bytes32 userOpHash,
        uint256 missingWalletFunds
    ) external onlyEntryPoint returns (uint256 validationData) {
        (address sender, uint256 nonce, bytes memory signature) = parseUserOp(userOp);

        // Verify nonce
        require(nonce == this.nonce, "AA25: invalid nonce");

        // Verify signature
        bytes32 hash = userOpHash.toEthSignedMessageHash();
        address recovered = hash.recover(signature);

        if (recovered != owner() && !authorized[recovered]) {
            return VALIDATION_FAILED;
        }

        // Increment nonce
        nonce++;

        // Handle missing funds (optional)
        if (missingWalletFunds > 0) {
            (bool success, ) = entryPoint.call{value: missingWalletFunds}("");
            require(success, "AA24: failed to send funds");
        }

        return VALIDATION_SUCCESS;
    }

    /**
     * @notice Execute user operation
     */
    function executeUserOp(
        address to,
        uint256 value,
        bytes calldata data
    ) external onlyEntryPoint returns (bool success) {
        (success, ) = to.call{value: value}(data);
        require(success, "AA13: failed to execute user op");
    }

    /**
     * @notice Execute batch user operations
     */
    function executeBatch(
        address[] calldata targets,
        uint256[] calldata values,
        bytes[] calldata data
    ) external onlyEntryPoint {
        require(targets.length == values.length && targets.length == data.length, "AA14: mismatched array lengths");

        for (uint256 i = 0; i < targets.length; i++) {
            (bool success, ) = targets[i].call{value: values[i]}(data[i]);
            require(success, "AA13: batch execution failed");
        }
    }

    /**
     * @notice ERC-1271: Validate signature
     */
    function isValidSignature(
        bytes32 hash,
        bytes calldata signature
    ) external view override returns (bytes4) {
        address signer = hash.recover(signature);
        if (signer == owner() || authorized[signer]) {
            return IERC1271.isValidSignature.selector;
        }
        return bytes4(0xffffffff);
    }

    /**
     * @notice Add authorized signer
     */
    function addAuthorized(address _authorized) external onlyOwner {
        authorized[_authorized] = true;
    }

    /**
     * @notice Remove authorized signer
     */
    function removeAuthorized(address _authorized) external onlyOwner {
        authorized[_authorized] = false;
    }

    /**
     * @notice Set new entry point
     */
    function setEntryPoint(address _entryPoint) external onlyOwner {
        entryPoint = _entryPoint;
    }

    /**
     * @notice Receive ETH
     */
    receive() external payable {}

    /**
     * @notice Parse user operation
     */
    function parseUserOp(
        bytes calldata userOp
    ) internal pure returns (address sender, uint256 nonce, bytes memory signature) {
        // Decode packed user operation
        (sender, nonce, signature) = abi.decode(
            userOp,
            (address, uint256, bytes)
        );
    }
}
