// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title TigerEmbeddedWallet
 * @dev React/Vue SDK, Mobile SDK, Game Plugin, WebComponent
 */

// React Component Interface
interface ITigerWalletWidget {
    function connect() external;
    function disconnect() external;
    function signTransaction(bytes calldata tx) external returns (bytes memory);
    function sendTransaction(bytes calldata tx) external returns (bytes32);
    function getBalance(address token) external view returns (uint256);
    function getChainId() external view returns (uint256);
}

// Core embedded wallet contract
contract TigerEmbeddedWallet {
    event WalletConnected(address indexed user);
    event TransactionSigned(address indexed user, bytes32 indexed hash);
    event TransactionSent(address indexed user, bytes32 indexed hash);
    
    mapping(address => bool) public connected;
    mapping(address => bytes32[]) public userTxs;
    
    /**
     * @dev Connect wallet
     */
    function connectWallet(address user) external {
        connected[user] = true;
        emit WalletConnected(user);
    }
    
    /**
     * @dev Sign transaction
     */
    function signTransaction(address user, bytes calldata tx) external returns (bytes32) {
        bytes32 hash = keccak256(tx);
        emit TransactionSigned(user, hash);
        return hash;
    }
    
    /**
     * @dev Send transaction
     */
    function sendTransaction(address user, bytes calldata tx) external returns (bytes32) {
        bytes32 hash = keccak256(tx);
        userTxs[user].push(hash);
        emit TransactionSent(user, hash);
        return hash;
    }
    
    /**
     * @dev Get transaction count
     */
    function getTxCount(address user) external view returns (uint256) {
        return userTxs[user].length;
    }
}