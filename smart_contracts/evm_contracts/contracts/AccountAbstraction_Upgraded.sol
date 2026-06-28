// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/interfaces/IERC1271.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title TigerAccountAbstraction
 * @notice EIP-4337 Account Abstraction - UPGRADED
 * @dev UserOperations, batching, ERC-1271, paymaster support
 */
contract TigerAccountAbstraction is IERC1271, Ownable {
    using ECDSA for bytes32;

    uint256 public constant VALIDATION_SUCCESS = 0;
    uint256 public constant VALIDATION_FAILED = 1;

    event ExecutedUserOp(
        bytes32 indexed userOpHash,
        address indexed sender,
        uint256 nonce,
        bool success
    );

    address public entryPoint;
    uint256 public nonce;
    mapping(bytes32 => bool) public executedOps;
    mapping(address => bool) public authorized;

    modifier onlyEntryPoint() {
        require(msg.sender == entryPoint, "AA30");
        _;
    }

    constructor(address _entryPoint, address _owner) Ownable(_owner) {
        entryPoint = _entryPoint;
        authorized[_owner] = true;
    }

    /**
     * @notice Validate user operation (EntryPoint callback)
     */
    function validateUserOp(
        bytes calldata userOp,
        bytes32 userOpHash,
        uint256 missingWalletFunds
    ) external onlyEntryPoint returns (uint256 validationData) {
        (address sender, uint256 nonce_op, bytes memory signature) = parseUserOp(userOp);

        require(nonce_op == nonce, "AA25");

        bytes32 hash = userOpHash.toEthSignedMessageHash();
        address recovered = hash.recover(signature);

        if (recovered != owner() && !authorized[recovered]) {
            return VALIDATION_FAILED;
        }

        nonce++;

        if (missingWalletFunds > 0) {
            (bool success, ) = entryPoint.call{value: missingWalletFunds}("");
            require(success, "AA24");
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
        require(success, "AA13");
    }

    /**
     * @notice Execute batch operations
     */
    function executeBatch(
        address[] calldata targets,
        uint256[] calldata values,
        bytes[] calldata data
    ) external onlyEntryPoint {
        require(targets.length == values.length && targets.length == data.length, "AA14");
        for (uint256 i = 0; i < targets.length; i++) {
            (bool success, ) = targets[i].call{value: values[i]}(data[i]);
            require(success, "AA13");
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
    function addAuthorized(address _addr) external onlyOwner {
        authorized[_addr] = true;
    }

    /**
     * @notice Remove authorized signer
     */
    function removeAuthorized(address _addr) external onlyOwner {
        authorized[_addr] = false;
    }

    /**
     * @notice Set entry point
     */
    function setEntryPoint(address _entryPoint) external onlyOwner {
        entryPoint = _entryPoint;
    }

    receive() external payable {}

    function parseUserOp(
        bytes calldata userOp
    ) internal pure returns (address sender, uint256 nonce_op, bytes memory signature) {
        (sender, nonce_op, signature) = abi.decode(
            userOp,
            (address, uint256, bytes)
        );
    }
}
