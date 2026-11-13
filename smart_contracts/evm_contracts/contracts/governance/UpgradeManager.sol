// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title UpgradeManager
 * @notice Manages contract upgrades
 */
contract UpgradeManager {
    struct Upgrade {
        bytes32 id;
        address implementation;
        uint256 votes;
        uint256 startTime;
        bool executed;
        bool cancelled;
    }

    mapping(bytes32 => Upgrade) public upgrades;
    mapping(bytes32 => mapping(address => bool)) public voters;

    uint256 public quorum;
    uint256 public votingPeriod;

    /// @notice Account allowed to administer voters and protocol parameters.
    address public owner;

    /// @notice Accounts allowed to propose, vote on, and execute upgrades.
    mapping(address => bool) public governors;

    event UpgradeProposed(bytes32 indexed id, address implementation);
    event UpgradeVoted(bytes32 indexed id, address voter);
    event UpgradeExecuted(bytes32 indexed id);
    event GovernorUpdated(address indexed account, bool allowed);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event QuorumUpdated(uint256 quorum);

    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    modifier onlyGovernor() {
        require(governors[msg.sender], "Not governor");
        _;
    }

    constructor() {
        owner = msg.sender;
        governors[msg.sender] = true;
        quorum = 3;
        votingPeriod = 86400;
    }

    /// @dev Proposing is governor-only. An unauthenticated caller could
    ///      otherwise register an upgrade proposal against an implementation of
    ///      their choosing, and votes alone would then carry it.
    function proposeUpgrade(address implementation) external onlyGovernor returns (bytes32) {
        require(implementation != address(0), "Zero implementation");

        bytes32 id = keccak256(abi.encode(implementation, block.timestamp));

        upgrades[id] = Upgrade({
            id: id,
            implementation: implementation,
            votes: 0,
            startTime: block.timestamp,
            executed: false,
            cancelled: false
        });

        emit UpgradeProposed(id, implementation);
        return id;
    }

    function vote(bytes32 upgradeId) external onlyGovernor {
        Upgrade storage upgrade = upgrades[upgradeId];
        require(upgrade.startTime > 0, "Not found");
        require(!upgrade.executed, "Executed");
        require(!upgrade.cancelled, "Cancelled");
        require(!voters[upgradeId][msg.sender], "Already voted");

        voters[upgradeId][msg.sender] = true;
        upgrade.votes++;

        emit UpgradeVoted(upgradeId, msg.sender);
    }

    function executeUpgrade(bytes32 upgradeId) external onlyGovernor {
        Upgrade storage upgrade = upgrades[upgradeId];
        require(upgrade.startTime > 0, "Not found");
        require(!upgrade.executed, "Already executed");
        require(block.timestamp >= upgrade.startTime + votingPeriod, "Voting ongoing");
        require(upgrade.votes >= quorum, "No quorum");

        upgrade.executed = true;

        emit UpgradeExecuted(upgradeId);
    }

    function setQuorum(uint256 _quorum) external onlyOwner {
        require(_quorum > 0, "Zero quorum");
        quorum = _quorum;
        emit QuorumUpdated(_quorum);
    }

    function setVotingPeriod(uint256 _votingPeriod) external onlyOwner {
        require(_votingPeriod > 0, "Zero period");
        votingPeriod = _votingPeriod;
    }

    function setGovernor(address account, bool allowed) external onlyOwner {
        require(account != address(0), "Zero address");
        governors[account] = allowed;
        emit GovernorUpdated(account, allowed);
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Zero address");
        address previous = owner;
        owner = newOwner;
        governors[newOwner] = true;
        emit OwnershipTransferred(previous, newOwner);
    }
}