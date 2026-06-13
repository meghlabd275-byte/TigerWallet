// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title Paymaster
 * @dev ERC-4337 Paymaster Contract
 * @dev Supports gas sponsoring and token payments
 */
contract Paymaster {
    // Events
    event Deposited(address indexed account, uint256 amount);
    event Withdrawn(address indexed account, uint256 amount);
    event Sponsorship(
        address indexed sender,
        address indexed paymaster,
        uint256 gasUsed,
        bytes data
    );
    event TokenRatesUpdated(
        address indexed token,
        uint256 exchangeRate,
        uint256 decimals
    );
    event WhitelistUpdated(
        address indexed user,
        bool allowed
    );
    event Sponsored(
        address indexed sender,
        uint256 sponsoredAmount
    );

    // Struct for token configuration
    struct TokenConfig {
        address token;
        uint256 exchangeRate;
        uint256 decimals;
        bool accepted;
    }

    // Struct for sponsor policy
    struct SponsorPolicy {
        uint256 maxGasLimit;
        uint256 maxSponsorAmount;
        uint256 gasBufferPercent;
        uint256 cooldownPeriod;
        mapping(address => uint256) lastSponsored;
    }

    // Entry point
    IEntryPoint public entryPoint;

    // Deposits
    mapping(address => uint256) public deposits;

    // Accepted tokens
    mapping(address => TokenConfig) public tokens;

    // Token list
    address[] public tokenList;

    // Whitelisted users
    mapping(address => bool) public whitelisted;

    // Sponsor policies
    SponsorPolicy[] public policies;

    // Current policy
    uint256 public currentPolicy;

    // Minimum stake
    uint256 public minStake = 1e18;

    // Minimum deposit
    uint256 public minDeposit = 1e17;

    // Maximum sponsorship per user
    mapping(address => uint256) public userSponsorshipLimits;

    // Total sponsored
    uint256 public totalSponsored;

    // Owner
    address public owner;

    // Paused state
    bool public paused;

    // Revert reasons
    string internal constant REVERT_REASON_ZERO_ADDRESS = "Zero address";
    string internal constant REVERT_REASON_INSUFFICIENT_DEPOSIT = "Insufficient deposit";
    string internal constant REVERT_REASON_TOKEN_NOT_ACCEPTED = "Token not accepted";
    string internal constant REVERT_REASON_INVALID_RATE = "Invalid rate";
    string internal constant REVERT_REASON_POLICY_VIOLATION = "Policy violation";
    string internal constant REVERT_REASON_PAUSED = "Paused";
    string internal constant REVERT_REASON_STAKE_TOO_LOW = "Stake too low";
    string internal constant REVERT_REASON_COOLDOWN = "Cooldown active";

    // Modifiers
    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    modifier whenNotPaused() {
        require(!paused, REVERT_REASON_PAUSED);
        _;
    }

    /**
     * @dev Constructor
     * @param _entryPoint Entry point address
     */
    constructor(address _entryPoint) {
        require(_entryPoint != address(0), REVERT_REASON_ZERO_ADDRESS);
        entryPoint = IEntryPoint(_entryPoint);
        owner = msg.sender;
    }

    /**
     * @dev Deposit for sponsored user
     */
    function deposit() external payable {
        deposits[msg.sender] += msg.value;
        emit Deposited(msg.sender, msg.value);
    }

    /**
     * @dev Withdraw deposit
     * @param amount Amount to withdraw
     * @param recipient Recipient address
     */
    function withdraw(
        uint256 amount,
        address payable recipient
    ) external onlyOwner {
        require(recipient != address(0), REVERT_REASON_ZERO_ADDRESS);
        require(deposits[address(this)] >= amount, REVERT_REASON_INSUFFICIENT_DEPOSIT);
        
        deposits[address(this)] -= amount;
        recipient.transfer(amount);
        
        emit Withdrawn(address(this), amount);
    }

    /**
     * @dev Add accepted token
     * @param token Token address
     * @param exchangeRate Exchange rate (1 ETH = rate units of token)
     * @param decimals Token decimals
     */
    function addToken(
        address token,
        uint256 exchangeRate,
        uint256 decimals
    ) external onlyOwner {
        require(token != address(0), REVERT_REASON_ZERO_ADDRESS);
        require(exchangeRate > 0, REVERT_REASON_INVALID_RATE);
        
        tokens[token] = TokenConfig(
            token,
            exchangeRate,
            decimals,
            true
        );
        
        tokenList.push(token);
        
        emit TokenRatesUpdated(token, exchangeRate, decimals);
    }

    /**
     * @dev Remove token
     * @param token Token address
     */
    function removeToken(address token) external onlyOwner {
        require(tokens[token].accepted, REVERT_REASON_TOKEN_NOT_ACCEPTED);
        tokens[token].accepted = false;
    }

    /**
     * @dev Whitelist user
     * @param user User address
     * @param allowed Allow status
     */
    function setWhitelist(
        address user,
        bool allowed
    ) external onlyOwner {
        whitelisted[user] = allowed;
        emit WhitelistUpdated(user, allowed);
    }

    /**
     * @dev Set sponsor policy
     * @param policy SponsorPolicy struct
     */
    function setPolicy(
        SponsorPolicy memory policy
    ) external onlyOwner {
        policies.push(policy);
        currentPolicy = policies.length - 1;
    }

    /**
     * @dev Set user sponsorship limit
     * @param user User address
     * @param limit Max sponsorship amount
     */
    function setUserLimit(
        address user,
        uint256 limit
    ) external onlyOwner {
        userSponsorshipLimits[user] = limit;
    }

    /**
     * @dev Validate paymaster user operation
     */
    function validatePaymasterUserOp(
        bytes32,
        bytes calldata,
        uint256
    ) external view returns (bytes memory) {
        require(!paused, REVERT_REASON_PAUSED);
        
        SponsorPolicy storage policy = policies[currentPolicy];
        
        // Check whitelist if enabled
        if (policy.maxGasLimit > 0) {
            require(
                tx.gaslimit <= policy.maxGasLimit,
                REVERT_REASON_POLICY_VIOLATION
            );
        }
        
        return abi.encode(0);
    }

    /**
     * @dev Post-operation handler
     */
    function postOp(
        PostOpMode mode,
        bytes calldata context,
        uint256 actualGasCost
    ) external {
        require(!paused, REVERT_REASON_PAUSED);
        
        if (mode == PostOpMode.postOpReverted) {
            return;
        }
        
        // Decode context
        (address sender) = abi.decode(context, (address));
        
        // Calculate sponsorship amount
        uint256 sponsorship = actualGasCost;
        
        // Check limits
        uint256 userLimit = userSponsorshipLimits[sender];
        if (userLimit > 0) {
            require(
                sponsorship <= userLimit,
                REVERT_REASON_POLICY_VIOLATION
            );
        }
        
        // Deduct from deposit
        require(
            deposits[sender] >= sponsorship,
            REVERT_REASON_INSUFFICIENT_DEPOSIT
        );
        
        deposits[sender] -= sponsorship;
        totalSponsored += sponsorship;
        
        emit Sponsored(sender, sponsorship);
    }

    /**
     * @dev Fund with tokens (for token-based gas sponsorship)
     * @param token Token address
     * @param amount Amount
     */
    function fundWithToken(
        address token,
        uint256 amount
    ) external whenNotPaused {
        require(tokens[token].accepted, REVERT_REASON_TOKEN_NOT_ACCEPTED);
        
        // Transfer tokens from user (requires approval)
        IERC20(token).transferFrom(msg.sender, address(this), amount);
        
        // Calculate ETH equivalent
        TokenConfig storage config = tokens[token];
        uint256 ethValue = (amount * config.exchangeRate) / (10 ** config.decimals);
        
        deposits[msg.sender] += ethValue;
        
        emit Deposited(msg.sender, ethValue);
    }

    /**
     * @dev Get deposit for user
     * @param user User address
     * @return Deposit amount
     */
    function getDeposit(address user) external view returns (uint256) {
        return deposits[user];
    }

    /**
     * @dev Get token count
     * @return Number of accepted tokens
     */
    function getTokenCount() external view returns (uint256) {
        return tokenList.length;
    }

    /**
     * @dev Get token at index
     * @param index Token index
     * @return Token address
     */
    function getToken(uint256 index) external view returns (address) {
        require(index < tokenList.length, "Invalid index");
        return tokenList[index];
    }

    /**
     * @dev Pause
     */
    function pause() external onlyOwner {
        paused = true;
    }

    /**
     * @dev Unpause
     */
    function unpause() external onlyOwner {
        paused = false;
    }

    /**
     * @dev Set owner
     * @param newOwner New owner address
     */
    function setOwner(address newOwner) external onlyOwner {
        require(newOwner != address(0), REVERT_REASON_ZERO_ADDRESS);
        owner = newOwner;
    }

    // Receive ETH
    receive() external payable {
        deposits[address(this)] += msg.value;
    }
}

/**
 * @title IEntryPoint
 * @dev Entry point interface
 */
interface IEntryPoint {
    function getDeposit(address account) external view returns (uint256);
}

/**
 * @title IERC20
 * @dev ERC20 interface
 */
interface IERC20 {
    function transferFrom(
        address from,
        address to,
        uint256 amount
    ) external returns (bool);

    function decimals() external view returns (uint8);
}

/**
 * @title PostOpMode
 * @dev Post operation mode enum
 */
enum PostOpMode {
    postOpReverted,
    postOpSuccess,
    postOpFailed
}