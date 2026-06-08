// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @title TigerDAO Governance Contract
/// @notice On-chain governance for TigerWallet DAO
contract TigerDAO {
    /// @notice Proposal states
    enum ProposalState {
        Pending,
        Active,
        Cancelled,
        Defeated,
        Succeeded,
        Executed,
        Expired
    }

    /// @notice Vote types
    enum VoteType {
        For,
        Against,
        Abstain
    }

    /// @notice Proposal structure
    struct Proposal {
        uint256 id;
        address proposer;
        string title;
        string description;
        address target;
        uint256 value;
        bytes data;
        uint256 startTime;
        uint256 endTime;
        uint256 forVotes;
        uint256 againstVotes;
        uint256 abstainVotes;
        ProposalState state;
        bool executed;
    }

    /// @notice Vote structure
    struct Vote {
        address voter;
        VoteType voteType;
        uint256 weight;
        string reason;
    }

    /// @notice Proposal created event
    event ProposalCreated(
        uint256 indexed id,
        address indexed proposer,
        string title,
        uint256 startTime,
        uint256 endTime
    );

    /// @notice Vote cast event
    event VoteCast(
        address indexed voter,
        uint256 indexed proposalId,
        VoteType voteType,
        uint256 weight
    );

    /// @notice Proposal executed event
    event ProposalExecuted(uint256 indexed id);

    /// @notice Governance parameters
    uint256 public constant QUORUM_THRESHOLD = 4e17; // 40%
    uint256 public constant MAJORITY_THRESHOLD = 5e17; // 50%
    uint256 public constant PROPOSAL_THRESHOLD = 1e22; // 10,000 tokens
    uint256 public constant VOTING_PERIOD = 3 days;
    uint256 public constant EXECUTION_PERIOD = 2 days;

    /// @notice Token for governance
    IERC20 public governanceToken;

    /// @notice Proposals mapping
    mapping(uint256 => Proposal) public proposals;

    /// @notice Votes mapping (proposalId => voter => Vote)
    mapping(uint256 => mapping(address => Vote)) public votes;

    /// @notice Proposal count
    uint256 public proposalCount;

    /// @notice Delegatees (user => delegatee)
    mapping(address => address) public delegates;

    /// @notice Checkpoint for vote weight
    mapping(address => uint256) public checkpoints;

    /// @notice Constructor
    constructor(address _governanceToken) {
        governanceToken = IERC20(_governanceToken);
    }

    /// @notice Create a new proposal
    function propose(
        string memory _title,
        string memory _description,
        address _target,
        uint256 _value,
        bytes memory _data
    ) external returns (uint256) {
        require(
            governanceToken.balanceOf(msg.sender) >= PROPOSAL_THRESHOLD,
            "Below proposal threshold"
        );

        uint256 id = ++proposalCount;
        uint256 startTime = block.timestamp;
        uint256 endTime = startTime + VOTING_PERIOD;

        Proposal storage proposal = proposals[id];
        proposal.id = id;
        proposal.proposer = msg.sender;
        proposal.title = _title;
        proposal.description = _description;
        proposal.target = _target;
        proposal.value = _value;
        proposal.data = _data;
        proposal.startTime = startTime;
        proposal.endTime = endTime;
        proposal.state = ProposalState.Active;

        emit ProposalCreated(id, msg.sender, _title, startTime, endTime);

        return id;
    }

    /// @notice Cast a vote on a proposal
    function castVote(
        uint256 _proposalId,
        VoteType _voteType,
        string memory _reason
    ) external {
        Proposal storage proposal = proposals[_proposalId];
        require(proposal.state == ProposalState.Active, "Proposal not active");
        require(block.timestamp <= proposal.endTime, "Voting period ended");
        require(
            votes[_proposalId][msg.sender].weight == 0,
            "Already voted"
        );

        uint256 weight = getVotes(msg.sender);

        if (_voteType == VoteType.For) {
            proposal.forVotes += weight;
        } else if (_voteType == VoteType.Against) {
            proposal.againstVotes += weight;
        } else {
            proposal.abstainVotes += weight;
        }

        votes[_proposalId][msg.sender] = Vote({
            voter: msg.sender,
            voteType: _voteType,
            weight: weight,
            reason: _reason
        });

        emit VoteCast(msg.sender, _proposalId, _voteType, weight);
    }

    /// @notice Execute a proposal
    function execute(uint256 _proposalId) external {
        Proposal storage proposal = proposals[_proposalId];
        require(proposal.state == ProposalState.Succeeded, "Proposal not succeeded");
        require(
            block.timestamp <= proposal.endTime + EXECUTION_PERIOD,
            "Execution period expired"
        );

        uint256 totalVotes = proposal.forVotes +
            proposal.againstVotes +
            proposal.abstainVotes;
        require(
            proposal.forVotes >= (totalVotes * QUORUM_THRESHOLD) / 1e18,
            "Quorum not reached"
        );
        require(
            proposal.forVotes > proposal.againstVotes,
            "Majority not reached"
        );

        proposal.state = ProposalState.Executed;
        proposal.executed = true;

        // Execute the proposal
        (bool success, ) = proposal.target.call{value: proposal.value}(
            proposal.data
        );
        require(success, "Execution failed");

        emit ProposalExecuted(_proposalId);
    }

    /// @notice Get vote weight for an account
    function getVotes(address _account) public view returns (uint256) {
        return governanceToken.balanceOf(_account) + checkpoints[_account];
    }

    /// @notice Delegate votes to another account
    function delegate(address _to) external {
        delegates[msg.sender] = _to;
    }

    /// @notice Get proposal state
    function state(uint256 _proposalId) external view returns (ProposalState) {
        Proposal storage proposal = proposals[_proposalId];

        if (proposal.state == ProposalState.Cancelled) {
            return ProposalState.Cancelled;
        }
        if (proposal.executed) {
            return ProposalState.Executed;
        }
        if (block.timestamp < proposal.startTime) {
            return ProposalState.Pending;
        }
        if (block.timestamp <= proposal.endTime) {
            return ProposalState.Active;
        }

        uint256 totalVotes = proposal.forVotes +
            proposal.againstVotes +
            proposal.abstainVotes;

        if (proposal.forVotes > proposal.againstVotes &&
            proposal.forVotes >= (totalVotes * QUORUM_THRESHOLD) / 1e18
        ) {
            return ProposalState.Succeeded;
        }

        return ProposalState.Defeated;
    }
}

interface IERC20 {
    function balanceOf(address account) external view returns (uint256);
}