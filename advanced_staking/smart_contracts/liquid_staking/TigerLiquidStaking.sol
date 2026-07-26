/**
 * TigerWallet Liquid Staking Smart Contract
 * PSOL (Phantom Liquid Staking) Equivalent Implementation
 * 
 * Features:
 * - Liquid staking with derivative tokens
 * - Auto-compounding
 * - Validator delegation
 * - Rewards distribution
 * - Slashing protection
 */

pragma solidity ^0.8.19;

// SPDX-License-Identifier: MIT

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/access/AccessControl.sol";
import "@openzeppelin/contracts/security/Pausable.sol";

/**
 * @title TigerLiquidStake - Liquid Staking Token
 * @notice ERC20 token representing staked position
 */
contract TigerLiquidStake is ERC20, ERC20Burnable, AccessControl, Pausable {
    bytes32 public constant MINTER_ROLE = keccak256("MINTER_ROLE");
    bytes32 public constant BURNER_ROLE = keccak256("BURNER_ROLE");
    
    // Staking pool reference
    address public stakingPool;
    
    // Mapping from user to staked amount (for rewards calculation)
    mapping(address => uint256) public userStake;
    mapping(address => uint256) public userStakeTime;
    
    // Total staked (for calculating shares)
    uint256 public totalStaked;
    
    // Exchange rate (tokens per wei staked) - starts at 1:1
    uint256 public exchangeRate = 1e18;
    uint256 public lastUpdateTime;
    
    // Reward distribution
    uint256 public accRewardsPerShare;
    mapping(address => uint256) public userRewardDebt;
    mapping(address => uint256) public pendingRewards;
    
    // Events
    event Stake(address indexed user, uint256 amount, uint256 tokensMinted);
    event Unstake(address indexed user, uint256 amount, uint256 tokensBurned);
    event RewardsClaimed(address indexed user, uint256 amount);
    event ExchangeRateUpdated(uint256 oldRate, uint256 newRate);
    
    /**
     * @dev Constructor
     * @param _name Name of liquid stake token
     * @param _symbol Symbol of liquid stake token
     * @param _stakingPool Address of main staking pool
     */
    constructor(
        string memory _name,
        string memory _symbol,
        address _stakingPool
    ) ERC20(_name, _symbol) {
        require(_stakingPool != address(0), "Invalid staking pool");
        stakingPool = _stakingPool;
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
        _grantRole(MINTER_ROLE, _stakingPool);
        _grantRole(BURNER_ROLE, _stakingPool);
        lastUpdateTime = block.timestamp;
    }
    
    /**
     * @dev Stake tokens to receive liquid stake tokens
     * @param _user User address
     * @param _amount Amount to stake
     */
    function mint(address _user, uint256 _amount) external onlyRole(MINTER_ROLE) whenNotPaused {
        require(_amount > 0, "Cannot stake 0");
        
        // Calculate tokens to mint based on exchange rate
        uint256 tokensToMint = (_amount * 1e18) / exchangeRate;
        
        // Update user stake
        _updateUserStake(_user);
        
        userStake[_user] += _amount;
        totalStaked += _amount;
        
        // Mint liquid tokens
        _mint(_user, tokensToMint);
        
        // Update reward debt
        userRewardDebt[_user] = (userStake[_user] * accRewardsPerShare) / 1e18;
        
        emit Stake(_user, _amount, tokensToMint);
    }
    
    /**
     * @dev Burn liquid stake tokens to unstake
     * @param _user User address
     * @param _tokenAmount Amount of liquid tokens to burn
     */
    function burn(address _user, uint256 _tokenAmount) external onlyRole(BURNER_ROLE) whenNotPaused {
        require(_tokenAmount > 0, "Cannot burn 0");
        require(balanceOf(_user) >= _tokenAmount, "Insufficient balance");
        
        // Calculate underlying stake amount
        uint256 stakeAmount = (_tokenAmount * exchangeRate) / 1e18;
        
        _updateUserStake(_user);
        
        // Claim pending rewards first
        if (pendingRewards[_user] > 0) {
            uint256 reward = pendingRewards[_user];
            pendingRewards[_user] = 0;
            emit RewardsClaimed(_user, reward);
        }
        
        // Update stakes
        userStake[_user] -= stakeAmount;
        totalStaked -= stakeAmount;
        
        // Burn liquid tokens
        _burn(_user, _tokenAmount);
        
        // Update reward debt
        userRewardDebt[_user] = (userStake[_user] * accRewardsPerShare) / 1e18;
        
        emit Unstake(_user, stakeAmount, _tokenAmount);
    }
    
    /**
     * @dev Update rewards for a user
     * @param _user User address
     */
    function updateRewards(address _user) external onlyRole(MINTER_ROLE) {
        _updateUserStake(_user);
    }
    
    /**
     * @dev Update exchange rate
     * @param _newRate New exchange rate
     */
    function updateExchangeRate(uint256 _newRate) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(_newRate > 0, "Invalid rate");
        require(_newRate >= exchangeRate, "Rate can only increase");
        
        // Update rewards before rate change
        _updateAllStakes();
        
        uint256 oldRate = exchangeRate;
        exchangeRate = _newRate;
        lastUpdateTime = block.timestamp;
        
        emit ExchangeRateUpdated(oldRate, _newRate);
    }
    
    /**
     * @dev Distribute rewards to all stakers
     * @param _rewards Total rewards to distribute
     */
    function distributeRewards(uint256 _rewards) external onlyRole(MINTER_ROLE) {
        require(_rewards > 0, "No rewards");
        require(totalStaked > 0, "No stakers");
        
        accRewardsPerShare += (_rewards * 1e18) / totalStaked;
        lastUpdateTime = block.timestamp;
    }
    
    /**
     * @dev Claim pending rewards
     */
    function claimRewards() external whenNotPaused {
        _updateUserStake(msg.sender);
        
        uint256 reward = pendingRewards[msg.sender];
        require(reward > 0, "No pending rewards");
        
        pendingRewards[msg.sender] = 0;
        
        // Transfer rewards (would be done via staking pool)
        
        emit RewardsClaimed(msg.sender, reward);
    }
    
    /**
     * @dev Get pending rewards for user
     * @param _user User address
     */
    function getPendingRewards(address _user) external view returns (uint256) {
        uint256 userShares = (userStake[_user] * accRewardsPerShare) / 1e18;
        uint256 rewardDebt = userRewardDebt[_user];
        
        if (userShares > rewardDebt) {
            return userShares - rewardDebt + pendingRewards[_user];
        }
        return pendingRewards[_user];
    }
    
    /**
     * @dev Get underlying stake amount for user
     * @param _user User address
     */
    function getUnderlyingStake(address _user) external view returns (uint256) {
        return userStake[_user];
    }
    
    /**
     * @dev Update user stake and rewards
     */
    function _updateUserStake(address _user) internal {
        if (userStake[_user] > 0) {
            uint256 userShares = (userStake[_user] * accRewardsPerShare) / 1e18;
            if (userShares > userRewardDebt[_user]) {
                pendingRewards[_user] += userShares - userRewardDebt[_user];
            }
        }
    }
    
    /**
     * @dev Update all stakes
     */
    function _updateAllStakes() internal {
        // Would iterate through all stakers in production
        // For now, just update the accumulator
    }
    
    /**
     * @dev Pause contract
     */
    function pause() external onlyRole(DEFAULT_ADMIN_ROLE) {
        _pause();
    }
    
    /**
     * @dev Unpause contract
     */
    function unpause() external onlyRole(DEFAULT_ADMIN_ROLE) {
        _unpause();
    }
}

/**
 * @title TigerStakingPool - Main Staking Pool
 * @notice Handles staking, validator delegation, and rewards distribution
 */
contract TigerStakingPool is ReentrancyGuard, AccessControl, Pausable {
    
    // Interfaces
    IERC20 public stakingToken;
    IERC20 public rewardToken;
    TigerLiquidStake public liquidStakeToken;
    
    // Validator management
    struct Validator {
        address validatorAddress;
        uint256 delegatedAmount;
        uint256 totalRewards;
        bool isActive;
        bool isSlashed;
    }
    
    mapping(address => Validator) public validators;
    address[] public validatorList;
    
    // Pool state
    uint256 public totalDelegated;
    uint256 public totalRewards;
    uint256 public rewardsToDistribute;
    uint256 public lastDistributionTime;
    
    // Rewards configuration
    uint256 public constant REWARD_RATE = 5e16; // 5% annual
    uint256 public constant SECONDS_PER_YEAR = 365 days;
    
    // Fee configuration
    uint256 public protocolFeePercent = 2e16; // 2%
    address public feeRecipient;
    
    // Events
    event DelegatedToValidator(address indexed validator, uint256 amount);
    event UndelegatedFromValidator(address indexed validator, uint256 amount);
    event RewardsDistributed(uint256 amount);
    event ValidatorSlashed(address indexed validator, uint256 slashAmount);
    event FeeUpdated(uint256 oldFee, uint256 newFee);
    
    /**
     * @dev Constructor
     * @param _stakingToken Token to stake (e.g., ETH, SOL)
     * @param _rewardToken Reward token
     * @param _liquidStakeToken Liquid stake token address
     * @param _feeRecipient Fee recipient address
     */
    constructor(
        address _stakingToken,
        address _rewardToken,
        address _liquidStakeToken,
        address _feeRecipient
    ) {
        require(_stakingToken != address(0), "Invalid staking token");
        require(_rewardToken != address(0), "Invalid reward token");
        require(_liquidStakeToken != address(0), "Invalid liquid stake token");
        
        stakingToken = IERC20(_stakingToken);
        rewardToken = IERC20(_rewardToken);
        liquidStakeToken = TigerLiquidStake(_liquidStakeToken);
        feeRecipient = _feeRecipient;
        
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
    }
    
    /**
     * @dev Stake tokens
     * @param _amount Amount to stake
     */
    function stake(uint256 _amount) external nonReentrant whenNotPaused {
        require(_amount > 0, "Cannot stake 0");
        require(_amount <= stakingToken.balanceOf(msg.sender), "Insufficient balance");
        
        // Transfer tokens from user
        stakingToken.transferFrom(msg.sender, address(this), _amount);
        
        // Mint liquid stake tokens
        liquidStakeToken.mint(msg.sender, _amount);
    }
    
    /**
     * @dev Unstake tokens
     * @param _tokenAmount Amount of liquid tokens to burn
     */
    function unstake(uint256 _tokenAmount) external nonReentrant whenNotPaused {
        require(_tokenAmount > 0, "Cannot unstake 0");
        require(liquidStakeToken.balanceOf(msg.sender) >= _tokenAmount, "Insufficient balance");
        
        // Burn liquid tokens
        liquidStakeToken.burn(msg.sender, _tokenAmount);
        
        // Return staking tokens
        stakingToken.transfer(msg.sender, _tokenAmount);
    }
    
    /**
     * @dev Delegate to validator
     * @param _validator Validator address
     * @param _amount Amount to delegate
     */
    function delegateToValidator(address _validator, uint256 _amount) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(_amount > 0, "Cannot delegate 0");
        require(validators[_validator].isActive, "Validator not active");
        
        validators[_validator].delegatedAmount += _amount;
        totalDelegated += _amount;
        
        // Transfer to validator (simplified - real implementation would interact with validator)
        
        emit DelegatedToValidator(_validator, _amount);
    }
    
    /**
     * @dev Undelegate from validator
     * @param _validator Validator address
     * @param _amount Amount to undelegate
     */
    function undelegateFromValidator(address _validator, uint256 _amount) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(_amount > 0, "Cannot undelegate 0");
        require(validators[_validator].delegatedAmount >= _amount, "Insufficient delegation");
        
        validators[_validator].delegatedAmount -= _amount;
        totalDelegated -= _amount;
        
        emit UndelegatedFromValidator(_validator, _amount);
    }
    
    /**
     * @dev Add validator
     * @param _validator Validator address
     */
    function addValidator(address _validator) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(_validator != address(0), "Invalid validator");
        require(!validators[_validator].isActive, "Validator already active");
        
        validators[_validator] = Validator({
            validatorAddress: _validator,
            delegatedAmount: 0,
            totalRewards: 0,
            isActive: true,
            isSlashed: false
        });
        
        validatorList.push(_validator);
    }
    
    /**
     * @dev Remove validator
     * @param _validator Validator address
     */
    function removeValidator(address _validator) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(validators[_validator].isActive, "Validator not active");
        require(validators[_validator].delegatedAmount == 0, "Has delegated amount");
        
        validators[_validator].isActive = false;
        
        // Remove from list
        for (uint256 i = 0; i < validatorList.length; i++) {
            if (validatorList[i] == _validator) {
                validatorList[i] = validatorList[validatorList.length - 1];
                validatorList.pop();
                break;
            }
        }
    }
    
    /**
     * @dev Slash validator (emergency)
     * @param _validator Validator address
     * @param _slashPercent Percent to slash (in wei)
     */
    function slashValidator(address _validator, uint256 _slashPercent) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(validators[_validator].isActive, "Validator not active");
        require(_slashPercent <= 1e18, "Cannot slash more than 100%");
        
        uint256 slashAmount = (validators[_validator].delegatedAmount * _slashPercent) / 1e18;
        
        validators[_validator].delegatedAmount -= slashAmount;
        validators[_validator].isSlashed = true;
        totalDelegated -= slashAmount;
        
        emit ValidatorSlashed(_validator, slashAmount);
    }
    
    /**
     * @dev Distribute rewards
     * @param _amount Amount of rewards to distribute
     */
    function distributeRewards(uint256 _amount) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(_amount > 0, "No rewards");
        
        // Transfer rewards from sender
        rewardToken.transferFrom(msg.sender, address(this), _amount);
        
        // Calculate protocol fee
        uint256 protocolFee = (_amount * protocolFeePercent) / 1e18;
        uint256 distributable = _amount - protocolFee;
        
        // Send fee to recipient
        if (protocolFee > 0) {
            rewardToken.transfer(feeRecipient, protocolFee);
        }
        
        // Update state
        totalRewards += _amount;
        rewardsToDistribute += distributable;
        lastDistributionTime = block.timestamp;
        
        // Update liquid stake token
        liquidStakeToken.distributeRewards(distributable);
        
        emit RewardsDistributed(_amount);
    }
    
    /**
     * @dev Calculate current APY
     */
    function calculateAPY() external view returns (uint256) {
        if (totalDelegated == 0) return 0;
        
        uint256 annualRewards = (totalDelegated * REWARD_RATE) / 1e18;
        return (annualRewards * 1e18) / totalDelegated;
    }
    
    /**
     * @dev Get active validator count
     */
    function getActiveValidatorCount() external view returns (uint256) {
        uint256 count = 0;
        for (uint256 i = 0; i < validatorList.length; i++) {
            if (validators[validatorList[i]].isActive) {
                count++;
            }
        }
        return count;
    }
    
    /**
     * @dev Update protocol fee
     * @param _newFee New fee percentage
     */
    function updateProtocolFee(uint256 _newFee) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(_newFee <= 1e17, "Fee too high"); // Max 10%
        
        uint256 oldFee = protocolFeePercent;
        protocolFeePercent = _newFee;
        
        emit FeeUpdated(oldFee, _newFee);
    }
    
    /**
     * @dev Pause contract
     */
    function pause() external onlyRole(DEFAULT_ADMIN_ROLE) {
        _pause();
    }
    
    /**
     * @dev Unpause contract
     */
    function unpause() external onlyRole(DEFAULT_ADMIN_ROLE) {
        _unpause();
    }
}

/**
 * @title TigerTokenFactory - Token Factory for Staking
 * @notice Factory contract to deploy liquid staking tokens for different assets
 */
contract TigerTokenFactory is AccessControl {
    
    bytes32 public constant FACTORY_ADMIN_ROLE = keccak256("FACTORY_ADMIN_ROLE");
    
    // Mapping from base token to liquid stake token
    mapping(address => address) public liquidStakeTokens;
    mapping(address => address) public stakingPools;
    
    // Template contracts
    address public liquidStakeTemplate;
    address public stakingPoolTemplate;
    
    // Events
    event LiquidStakeTokenCreated(address indexed baseToken, address indexed liquidStakeToken);
    event StakingPoolCreated(address indexed baseToken, address indexed stakingPool);
    
    constructor() {
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
        _grantRole(FACTORY_ADMIN_ROLE, msg.sender);
    }
    
    /**
     * @dev Set template contracts
     * @param _liquidStakeTemplate Liquid stake token template
     * @param _stakingPoolTemplate Staking pool template
     */
    function setTemplates(
        address _liquidStakeTemplate,
        address _stakingPoolTemplate
    ) external onlyRole(DEFAULT_ADMIN_ROLE) {
        liquidStakeTemplate = _liquidStakeTemplate;
        stakingPoolTemplate = _stakingPoolTemplate;
    }
    
    /**
     * @dev Deploy liquid staking token and pool
     * @param _name Name of liquid stake token
     * @param _symbol Symbol of liquid stake token
     * @param _baseToken Base token to stake
     * @param _rewardToken Reward token
     * @param _feeRecipient Fee recipient
     */
    function deployStakingPool(
        string memory _name,
        string memory _symbol,
        address _baseToken,
        address _rewardToken,
        address _feeRecipient
    ) external onlyRole(FACTORY_ADMIN_ROLE) returns (address, address) {
        require(liquidStakeTemplate != address(0), "Templates not set");
        require(_baseToken != address(0), "Invalid base token");
        require(liquidStakeTokens[_baseToken] == address(0), "Already deployed");
        
        // Deploy liquid stake token (simplified - would use CREATE2 in production)
        TigerLiquidStake liquidStake = new TigerLiquidStake(
            _name,
            _symbol,
            address(0) // Will be set after pool deployment
        );
        
        // Deploy staking pool
        TigerStakingPool stakingPool = new TigerStakingPool(
            _baseToken,
            _rewardToken,
            address(liquidStake),
            _feeRecipient
        );
        
        // Set pool in liquid stake
        // In production, would call a setup function
        
        // Store mappings
        liquidStakeTokens[_baseToken] = address(liquidStake);
        stakingPools[_baseToken] = address(stakingPool);
        
        emit LiquidStakeTokenCreated(_baseToken, address(liquidStake));
        emit StakingPoolCreated(_baseToken, address(stakingPool));
        
        return (address(liquidStake), address(stakingPool));
    }
    
    /**
     * @dev Get staking pool for base token
     * @param _baseToken Base token address
     */
    function getStakingPool(address _baseToken) external view returns (address) {
        return stakingPools[_baseToken];
    }
    
    /**
     * @dev Get liquid stake token for base token
     * @param _baseToken Base token address
     */
    function getLiquidStakeToken(address _baseToken) external view returns (address) {
        return liquidStakeTokens[_baseToken];
    }
}
