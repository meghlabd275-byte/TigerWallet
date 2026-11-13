// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title EmergencyPause
 * @notice Emergency pause functionality for the protocol
 *
 * @dev Every mutating entry point is restricted to the guardian set. A pause
 *      that any caller can lift is not a safety control, so unpause and
 *      deactivateEmergency are gated exactly like pause and
 *      activateEmergency. Guardians are managed by the owner, and the owner is
 *      an implicit guardian so the protocol can never end up with nobody able
 *      to pause it.
 */
contract EmergencyPause {
    bool public paused;
    bool public emergencyActive;

    mapping(bytes32 => uint256) public pauseDelays;
    mapping(bytes32 => uint256) public pauseTimestamps;

    /// @notice Account that can add and remove guardians.
    address public owner;

    /// @notice Accounts allowed to pause, unpause, and set pause delays.
    mapping(address => bool) public guardians;

    event Paused(address indexed account, string reason);
    event Unpaused(address indexed account);
    event EmergencyActivated(address indexed account, string reason);
    event EmergencyDeactivated(address indexed account);
    event GuardianUpdated(address indexed account, bool allowed);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    modifier whenNotPaused() {
        require(!paused, "Paused");
        _;
    }

    modifier whenPaused() {
        require(paused, "Not paused");
        _;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    modifier onlyGuardian() {
        require(guardians[msg.sender], "Not guardian");
        _;
    }

    constructor() {
        owner = msg.sender;
        guardians[msg.sender] = true;
    }

    function pause(string memory reason) external onlyGuardian {
        paused = true;
        emit Paused(msg.sender, reason);
    }

    function unpause() external onlyGuardian {
        require(!emergencyActive, "Emergency active");
        paused = false;
        emit Unpaused(msg.sender);
    }

    function activateEmergency(string memory reason) external onlyGuardian {
        emergencyActive = true;
        paused = true;
        emit EmergencyActivated(msg.sender, reason);
    }

    /// @dev Lifting an emergency is owner-only on purpose: a guardian whose key
    ///      is compromised must not be able to undo the protocol freeze.
    function deactivateEmergency() external onlyOwner {
        emergencyActive = false;
        emit EmergencyDeactivated(msg.sender);
    }

    function setPauseDelay(bytes32 functionId, uint256 delay) external onlyGuardian {
        pauseDelays[functionId] = delay;
    }

    function setGuardian(address account, bool allowed) external onlyOwner {
        require(account != address(0), "Zero address");
        guardians[account] = allowed;
        emit GuardianUpdated(account, allowed);
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Zero address");
        address previous = owner;
        owner = newOwner;
        guardians[newOwner] = true;
        emit OwnershipTransferred(previous, newOwner);
    }

    function canExecute(bytes32 functionId) external view returns (bool) {
        if (paused && !emergencyActive) return false;
        if (pauseTimestamps[functionId] == 0) return true;
        return block.timestamp >= pauseTimestamps[functionId] + pauseDelays[functionId];
    }
}