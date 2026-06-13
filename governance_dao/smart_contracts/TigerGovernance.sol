// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title TigerGovernance
 * @dev Multi-sig treasury, proposal system, voting, delegation
 */
contract TigerGovernance {
    event ProposalCreated(uint256 indexed id, address indexed proposer, string description);
    event VoteCast(address indexed voter, uint256 indexed proposal, uint256 votes, bool support);
    event ProposalExecuted(uint256 indexed id);
    event DelegationChanged(address indexed from, address indexed to, uint256 votes);
    event TreasuryUpdated(address indexed newTreasury);
    
    struct Proposal {
        address proposer;
        string description;
        uint256 votesFor;
        uint256 votesAgainst;
        uint256 startTime;
        uint256 endTime;
        bool executed;
        bool cancelled;
        address[] targets;
        uint256[] values;
        bytes[] calldatas;
    }
    
    mapping(uint256 => Proposal) public proposals;
    mapping(uint256 => mapping(address => bool)) public hasVoted;
    mapping(address => address) public delegates;
    mapping(address => uint256) public votingPower;
    mapping(address => bool) public members;
    
    uint256 public proposalCount;
    address public treasury;
    uint256 public votingPeriod = 3 days;
    uint256 public proposalThreshold = 1e18;
    uint256 public quorum = 1e20;
    
    /**
     * @dev Create proposal
     */
    function createProposal(
        string calldata description,
        address[] calldata targets,
        uint256[] calldata values,
        bytes[] calldata calldatas_
    ) external returns (uint256) {
        require(votingPower[msg.sender] >= proposalThreshold, "Below threshold");
        
        uint256 id = proposalCount++;
        
        proposals[id] = Proposal({
            proposer: msg.sender,
            description: description,
            votesFor: 0,
            votesAgainst: 0,
            startTime: block.timestamp,
            endTime: block.timestamp + votingPeriod,
            executed: false,
            cancelled: false,
            targets: targets,
            values: values,
            calldatas: calldatas_
        });
        
        emit ProposalCreated(id, msg.sender, description);
        return id;
    }
    
    /**
     * @dev Cast vote
     */
    function castVote(uint256 proposalId, bool support) external {
        require(!hasVoted[proposalId][msg.sender], "Already voted");
        
        Proposal storage p = proposals[proposalId];
        require(block.timestamp <= p.endTime, "Ended");
        require(!p.executed, "Executed");
        
        uint256 votes = votingPower[msg.sender];
        
        if (support) {
            p.votesFor += votes;
        } else {
            p.votesAgainst += votes;
        }
        
        hasVoted[proposalId][msg.sender] = true;
        
        emit VoteCast(msg.sender, proposalId, votes, support);
    }
    
    /**
     * @dev Execute proposal
     */
    function executeProposal(uint256 proposalId) external {
        Proposal storage p = proposals[proposalId];
        require(block.timestamp > p.endTime, "Not ended");
        require(!p.executed, "Executed");
        require(p.votesFor >= quorum, "No quorum");
        
        p.executed = true;
        
        // Execute calls
        for (uint256 i = 0; i < p.targets.length; ) {
            (bool success, ) = p.targets[i].call{value: p.values[i]}(p.calldatas[i]);
            require(success, "Call failed");
            unchecked { i++; }
        }
        
        emit ProposalExecuted(proposalId);
    }
    
    /**
     * @dev Delegate votes
     */
    function delegate(address to) external {
        require(to != address(0), "Zero address");
        
        votingPower[msg.sender] = 0;
        delegates[msg.sender] = to;
        
        emit DelegationChanged(msg.sender, to, votingPower[msg.sender]);
    }
    
    /**
     * @dev Add member
     */
    function addMember(address member) external {
        members[member] = true;
    }
    
    /**
     * @dev Set treasury
     */
    function setTreasury(address newTreasury) external {
        treasury = newTreasury;
        emit TreasuryUpdated(newTreasury);
    }
}