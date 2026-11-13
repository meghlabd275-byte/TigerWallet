// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title EmergencyCouncil
 * @notice Multi-sig council for emergency decisions
 */
contract EmergencyCouncil {
    struct Proposal {
        uint256 id;
        string description;
        bytes data;
        uint256 votesFor;
        uint256 votesAgainst;
        uint256 startTime;
        uint256 endTime;
        bool executed;
        bool cancelled;
    }

    mapping(address => bool) public councilMembers;
    mapping(uint256 => Proposal) public proposals;
    mapping(uint256 => mapping(address => bool)) public voted;

    uint256 public proposalCount;
    uint256 public quorum;
    uint256 public votingPeriod;

    event ProposalCreated(uint256 indexed id, string description);
    event Voted(uint256 indexed id, address voter, bool support);
    event ProposalExecuted(uint256 indexed id);

    /// @notice Account allowed to add and remove council members.
    address public owner;

    event MemberUpdated(address indexed member, bool allowed);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event QuorumUpdated(uint256 quorum);
    event VotingPeriodUpdated(uint256 votingPeriod);

    constructor() {
        owner = msg.sender;
        councilMembers[msg.sender] = true;
        quorum = 3;
        votingPeriod = 86400; // 24 hours
    }

    modifier onlyMember() {
        require(councilMembers[msg.sender], "Not member");
        _;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    function createProposal(string memory description, bytes memory data) external onlyMember returns (uint256) {
        proposalCount++;
        uint256 id = proposalCount;

        proposals[id] = Proposal({
            id: id,
            description: description,
            data: data,
            votesFor: 0,
            votesAgainst: 0,
            startTime: block.timestamp,
            endTime: block.timestamp + votingPeriod,
            executed: false,
            cancelled: false
        });

        emit ProposalCreated(id, description);
        return id;
    }

    function vote(uint256 proposalId, bool support) external onlyMember {
        Proposal storage proposal = proposals[proposalId];
        require(proposal.startTime > 0, "Not found");
        require(!proposal.executed, "Executed");
        require(!proposal.cancelled, "Cancelled");
        require(block.timestamp < proposal.endTime, "Ended");
        require(!voted[proposalId][msg.sender], "Already voted");

        voted[proposalId][msg.sender] = true;

        if (support) {
            proposal.votesFor++;
        } else {
            proposal.votesAgainst++;
        }

        emit Voted(proposalId, msg.sender, support);
    }

    function executeProposal(uint256 proposalId) external onlyMember {
        Proposal storage proposal = proposals[proposalId];
        require(proposal.startTime > 0, "Not found");
        require(!proposal.executed, "Already executed");
        require(block.timestamp >= proposal.endTime, "Voting ongoing");
        require(proposal.votesFor >= quorum, "No quorum");

        proposal.executed = true;

        emit ProposalExecuted(proposalId);
    }

    /// @dev Membership is the entire security boundary of this contract: a
    ///      caller who can add themselves can reach quorum alone and execute
    ///      any proposal. Both mutations are therefore owner-only.
    function addMember(address member) external onlyOwner {
        require(member != address(0), "Zero address");
        councilMembers[member] = true;
        emit MemberUpdated(member, true);
    }

    function removeMember(address member) external onlyOwner {
        require(councilMembers[member], "Not member");
        delete councilMembers[member];
        emit MemberUpdated(member, false);
    }

    function setQuorum(uint256 _quorum) external onlyOwner {
        require(_quorum > 0, "Zero quorum");
        quorum = _quorum;
        emit QuorumUpdated(_quorum);
    }

    function setVotingPeriod(uint256 _votingPeriod) external onlyOwner {
        require(_votingPeriod > 0, "Zero period");
        votingPeriod = _votingPeriod;
        emit VotingPeriodUpdated(_votingPeriod);
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Zero address");
        address previous = owner;
        owner = newOwner;
        councilMembers[newOwner] = true;
        emit OwnershipTransferred(previous, newOwner);
    }
}