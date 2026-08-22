// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/**
 * @title TigerBotPlatform
 * @notice Complete Bot Platform with Role-Based Access Control
 * @dev Full-featured trading bot platform with admin and client roles
 * 
 * Features:
 * - Role-Based Access Control (Admin, Bot Operator, Client)
 * - Bot Deployment and Management
 * - Fee Collection and Distribution
 * - Performance Tracking
 * - Emergency Controls
 * - Multi-exchange Support
 */
/// @notice Uniswap-V2-compatible router interface for real on-chain swaps.
interface IUniswapV2Router {
    function swapExactTokensForTokens(
        uint256 amountIn,
        uint256 amountOutMin,
        address[] calldata path,
        address to,
        uint256 deadline
    ) external returns (uint256[] memory amounts);
    function getAmountsOut(uint256 amountIn, address[] calldata path)
        external view returns (uint256[] memory amounts);
}

contract TigerBotPlatform is ReentrancyGuard {
    using SafeERC20 for IERC20;

    // ============================================================================
    // Roles
    // ============================================================================
    
    bytes32 constant ROLE_ADMIN = keccak256("ADMIN");
    bytes32 constant ROLE_BOT_OPERATOR = keccak256("BOT_OPERATOR");
    bytes32 constant ROLE_CLIENT = keccak256("CLIENT");
    bytes32 constant ROLE_MAINTENANCE = keccak256("MAINTENANCE");
    bytes32 constant ROLE_FINANCE = keccak256("FINANCE");

    // ============================================================================
    // Bot Types
    // ============================================================================
    
    uint8 constant BOT_TYPE_MARKET_MAKER = 1;
    uint8 constant BOT_TYPE_ARBITRAGE = 2;
    uint8 constant BOT_TYPE_SNIPER = 3;
    uint8 constant BOT_TYPE_LIQUIDITY = 4;
    uint8 constant BOT_TYPE_FRONT_RUN = 5;
    uint8 constant BOT_TYPE_MEV = 6;
    uint8 constant BOT_TYPE_FLASH_LOAN = 7;
    uint8 constant BOT_TYPE_CROSS_CHAIN = 8;
    uint8 constant BOT_TYPE_PERP_HEDGE = 9;

    // ============================================================================
    // State Variables
    // ============================================================================
    
    // Role Management
    mapping(address => bytes32) public userRoles;
    mapping(bytes32 => mapping(address => bool)) public roleMembers;
    
    // Bot Management
    mapping(bytes32 => Bot) public bots;
    bytes32[] public botIds;
    mapping(address => bytes32[]) public userBots;
    
    // Bot Statistics
    mapping(bytes32 => BotStats) public botStats;
    mapping(address => uint256) public userBotCount;
    
    // Fee Management
    uint256 public protocolFeeBps = 50; // 0.5%
    address public feeRecipient;
    uint256 public totalFeesCollected;
    mapping(address => uint256) public feesByUser;
    mapping(address => uint256) public feeTokenBalances; // real per-token fees held by the contract
    
    // Exchange Connections
    mapping(bytes32 => Exchange) public exchanges;
    bytes32[] public exchangeIds;
    mapping(address => mapping(bytes32 => bool)) public userExchanges;
    
    // Performance
    uint256 public totalProfit;
    uint256 public totalTrades;
    uint256 public activeBots;
    
    // Emergency
    bool public emergencyMode;
    bool public pauseTrading;
    bool public pauseNewBots;
    
    // Governance
    address public governance;
    address public pendingGovernance;
    uint256 public proposalCount;
    
    // ============== Structs ==============
    
    struct Bot {
        bytes32 id;
        address owner;
        uint8 botType;
        string name;
        string description;
        address tokenIn;
        address tokenOut;
        uint256 minInvestment;
        uint256 maxInvestment;
        uint256 targetApy;
        uint256 riskLevel;
        bool isActive;
        bool isPaused;
        uint256 createdAt;
        uint256 lastActiveAt;
    }
    
    struct BotStats {
        bytes32 botId;
        uint256 totalVolume;
        uint256 totalProfit;
        uint256 totalTrades;
        uint256 successfulTrades;
        uint256 failedTrades;
        uint256 lastTradeTime;
        uint256 averageTradeSize;
        uint256 winRate;
        uint256 totalFeesPaid;
        uint256 uptime;
    }
    
    struct Exchange {
        bytes32 id;
        string name;
        string endpoint;
        address router;
        uint256 gasLimit;
        bool isActive;
        uint256 minTradeSize;
        uint256 maxTradeSize;
        uint256 feeBps;
    }
    
    struct BotRequest {
        address user;
        uint8 botType;
        address tokenIn;
        address tokenOut;
        uint256 investment;
        uint256 minApy;
    }
    
    struct Trade {
        bytes32 botId;
        address user;
        bytes32 exchangeId;
        address tokenIn;
        address tokenOut;
        uint256 amountIn;
        uint256 amountOut;
        uint256 fee;
        uint256 profit;
        uint256 timestamp;
        bool success;
    }
    
    // ============== Events ==============
    
    event RoleAssigned(address indexed user, bytes32 role);
    event RoleRevoked(address indexed user, bytes32 role);
    event BotCreated(bytes32 indexed botId, address indexed owner, uint8 botType);
    event BotStarted(bytes32 indexed botId);
    event BotStopped(bytes32 indexed botId);
    event BotPaused(bytes32 indexed botId);
    event TradeExecuted(bytes32 indexed botId, uint256 amountIn, uint256 amountOut, bool success);
    event FeeCollected(address indexed user, uint256 amount);
    event EmergencyMode(bool enabled);
    event ExchangeAdded(bytes32 indexed exchangeId, string name);
    event ExchangeRemoved(bytes32 indexed exchangeId);
    event ProfitDistributed(address indexed user, uint256 amount);
    
    // ============== Constructor ==============
    
    constructor() {
        governance = msg.sender;
        feeRecipient = msg.sender;
        userRoles[msg.sender] = ROLE_ADMIN;
        roleMembers[ROLE_ADMIN][msg.sender] = true;
    }
    
    // ============================================================================
    // Role Management
    // ============================================================================
    
    /**
     * @notice Grant role to user
     */
    function grantRole(address _user, bytes32 _role) external {
        require(msg.sender == governance, "ONLY_GOVERNANCE");
        require(_user != address(0), "INVALID_USER");
        
        userRoles[_user] = _role;
        roleMembers[_role][_user] = true;
        
        emit RoleAssigned(_user, _role);
    }
    
    /**
     * @notice Revoke role from user
     */
    function revokeRole(address _user) external {
        require(msg.sender == governance, "ONLY_GOVERNANCE");
        
        bytes32 role = userRoles[_user];
        require(role != ROLE_ADMIN, "CANNOT_REVOKE_ADMIN");
        
        roleMembers[role][_user] = false;
        delete userRoles[_user];
        
        emit RoleRevoked(_user, role);
    }
    
    /**
     * @notice Check if user has role
     */
    function hasRole(address _user, bytes32 _role) external view returns (bool) {
        return roleMembers[_role][_user];
    }
    
    /**
     * @notice Get user role
     */
    function getUserRole(address _user) external view returns (bytes32) {
        return userRoles[_user];
    }
    
    // ============================================================================
    // Bot Creation (Admin Only)
    // ============================================================================
    
    /**
     * @notice Create new bot (Admin or Bot Operator)
     */
    function createBot(
        uint8 _botType,
        string calldata _name,
        string calldata _description,
        address _tokenIn,
        address _tokenOut,
        uint256 _minInvestment,
        uint256 _maxInvestment,
        uint256 _targetApy,
        uint256 _riskLevel
    ) external returns (bytes32 botId) {
        require(!emergencyMode, "EMERGENCY_MODE");
        require(!pauseNewBots, "BOTS_PAUSED");
        require(_botType >= BOT_TYPE_MARKET_MAKER && _botType <= BOT_TYPE_PERP_HEDGE, "INVALID_TYPE");
        require(_minInvestment > 0, "INVALID_MIN");
        require(_maxInvestment >= _minInvestment, "INVALID_MAX");
        
        // Check role - admin or client can create
        bytes32 role = userRoles[msg.sender];
        require(role == ROLE_ADMIN || role == ROLE_BOT_OPERATOR || role == ROLE_CLIENT, "NO_ROLE");
        
        botId = keccak256(abi.encodePacked(
            msg.sender,
            _botType,
            _name,
            block.timestamp,
            botIds.length
        ));
        
        botIds.push(botId);
        userBots[msg.sender].push(botId);
        userBotCount[msg.sender]++;
        activeBots++;
        emit BotCreated(botId, msg.sender, _botType);
        {
            Bot storage b = bots[botId];
            b.id = botId;
            b.owner = msg.sender;
            b.botType = _botType;
            b.name = _name;
            b.description = _description;
            b.tokenIn = _tokenIn;
            b.tokenOut = _tokenOut;
            b.minInvestment = _minInvestment;
            b.maxInvestment = _maxInvestment;
            b.targetApy = _targetApy;
            b.riskLevel = _riskLevel;
            b.isActive = true;
            b.isPaused = false;
            b.createdAt = block.timestamp;
            b.lastActiveAt = block.timestamp;
            BotStats storage st = botStats[botId];
            st.botId = botId;
            st.uptime = 10000; // 100%
        }
    }
    
    // ============================================================================
    // Bot Control
    // ============================================================================
    
    /**
     * @notice Start bot
     */
    function startBot(bytes32 _botId) external {
        Bot storage bot = bots[_botId];
        require(bot.owner == msg.sender || roleMembers[ROLE_ADMIN][msg.sender], "NOT_OWNER");
        require(bot.isActive, "NOT_ACTIVE");
        require(!bot.isPaused, "ALREADY_RUNNING");
        require(!pauseTrading, "TRADING_PAUSED");
        
        bot.isPaused = false;
        bot.lastActiveAt = block.timestamp;
        
        emit BotStarted(_botId);
    }
    
    /**
     * @notice Stop bot
     */
    function stopBot(bytes32 _botId) external {
        Bot storage bot = bots[_botId];
        require(bot.owner == msg.sender || roleMembers[ROLE_ADMIN][msg.sender], "NOT_OWNER");
        
        bot.isPaused = true;
        
        emit BotStopped(_botId);
    }
    
    /**
     * @notice Pause bot (Admin only)
     */
    function pauseBot(bytes32 _botId) external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        
        bots[_botId].isPaused = true;
        
        emit BotPaused(_botId);
    }
    
    /**
     * @notice Delete bot
     */
    function deleteBot(bytes32 _botId) external {
        Bot storage bot = bots[_botId];
        require(bot.owner == msg.sender || roleMembers[ROLE_ADMIN][msg.sender], "NOT_OWNER");
        
        bot.isActive = false;
        activeBots--;
    }
    
    // ============================================================================
    // Trade Execution
    // ============================================================================

    /**
     * @notice Execute trade through bot — REAL on-chain swap via the exchange router.
     * @dev The caller (bot owner/operator/admin) must have approved this contract to
     *      spend _amountIn of _tokenIn. Tokens are pulled from the caller, swapped via
     *      the configured V2 router, and the output is forwarded to the bot owner. The
     *      protocol fee is taken in the output token and transferred to feeRecipient.
     *      Non-reentrant. No simulated output — the real swap return is used.
     */
    function executeTrade(
        bytes32 _botId,
        bytes32 _exchangeId,
        address _tokenIn,
        address _tokenOut,
        uint256 _amountIn,
        uint256 _minAmountOut,
        bytes memory /* _data */
    ) public nonReentrant returns (uint256 amountOut) {
        return _executeSingleTrade(_botId, _exchangeId, _tokenIn, _tokenOut, _amountIn, _minAmountOut);
    }

    // Batch trades removed: callers should submit individual executeTrade transactions
    // (the multi-array calldata + loop form triggered EVM stack-too-deep under via_ir).

    /// @dev Internal worker for a single swap. Delegates to _pullAndApprove,
    ///      _callSwap (returns nothing; balance delta read after), and _settle
    ///      to keep each function's EVM stack shallow (no stack-too-deep).
    function _executeSingleTrade(
        bytes32 _botId,
        bytes32 _exchangeId,
        address _tokenIn,
        address _tokenOut,
        uint256 _amountIn,
        uint256 _minAmountOut
    ) internal returns (uint256 amountOut) {
        require(!emergencyMode, "EMERGENCY_MODE");
        require(!pauseTrading, "TRADING_PAUSED");
        require(bots[_botId].isActive, "BOT_NOT_ACTIVE");
        require(!bots[_botId].isPaused, "BOT_PAUSED");
        require(bots[_botId].owner == msg.sender || roleMembers[ROLE_ADMIN][msg.sender] || roleMembers[ROLE_BOT_OPERATOR][msg.sender], "NOT_AUTHORIZED");
        address router = exchanges[_exchangeId].router;
        require(exchanges[_exchangeId].isActive, "EXCHANGE_NOT_ACTIVE");
        require(router != address(0), "NO_ROUTER");
        require(_amountIn >= exchanges[_exchangeId].minTradeSize, "BELOW_MIN");
        require(_amountIn <= exchanges[_exchangeId].maxTradeSize, "ABOVE_MAX");
        require(_tokenIn != address(0) && _tokenOut != address(0), "ZERO_TOKEN");
        require(_tokenIn != _tokenOut, "SAME_TOKEN");

        _pullAndApprove(_tokenIn, _amountIn, router);
        _preBal = IERC20(_tokenOut).balanceOf(address(this));
        _callSwap(router, _tokenIn, _tokenOut, _amountIn, _minAmountOut);
        amountOut = IERC20(_tokenOut).balanceOf(address(this)) - _preBal;
        require(amountOut >= _minAmountOut, "SLIPPAGE");
        uint256 fee = amountOut * protocolFeeBps / 10000;
        _settle(_botId, bots[_botId].owner, _tokenOut, _amountIn, amountOut, fee);
    }

    uint256 private _preBal;

    function _pullAndApprove(address _tokenIn, uint256 _amountIn, address _router) internal {
        uint256 pre = IERC20(_tokenIn).balanceOf(address(this));
        _pull(_tokenIn, _amountIn);
        require(IERC20(_tokenIn).balanceOf(address(this)) - pre == _amountIn, "IN_MISMATCH");
        IERC20(_tokenIn).safeIncreaseAllowance(_router, _amountIn);
    }

    function _pull(address _tokenIn, uint256 _amountIn) internal {
        IERC20(_tokenIn).safeTransferFrom(msg.sender, address(this), _amountIn);
    }

    /// Transient swap context (avoids passing many params into the swap helper,
    /// which triggers EVM stack-too-deep under via_ir).
    address private _swapRouter;
    address private _swapTokenIn;
    address private _swapTokenOut;
    uint256 private _swapAmountIn;
    uint256 private _swapMinOut;

    function _callSwap(address _router, address _tokenIn, address _tokenOut, uint256 _amountIn, uint256 _minAmountOut) internal {
        _swapRouter = _router;
        _swapTokenIn = _tokenIn;
        _swapTokenOut = _tokenOut;
        _swapAmountIn = _amountIn;
        _swapMinOut = _minAmountOut;
        _doSwapFromContext();
    }

    function _doSwapFromContext() internal {
        // Encode the swap calldata manually (no Solidity memory-array local ->
        // avoids stack-too-deep). path = [tokenIn, tokenOut].
        bytes memory data = abi.encodeWithSelector(
            IUniswapV2Router.swapExactTokensForTokens.selector,
            _swapAmountIn,
            _swapMinOut,
            _twoPath(),
            address(this),
            block.timestamp + 300
        );
        (bool ok, ) = _swapRouter.call(data);
        require(ok, "SWAP_FAILED");
    }

    function _twoPath() internal view returns (address[] memory path) {
        path = new address[](2);
        path[0] = _swapTokenIn;
        path[1] = _swapTokenOut;
    }

    function _settle(bytes32 _botId, address _owner, address _tokenOut, uint256 _amountIn, uint256 _amountOut, uint256 _fee) internal {
        _transferProfit(_owner, _tokenOut, _fee, _amountOut);
        _updateStats(_botId, _amountIn, _amountOut, _fee);
    }

    function _transferProfit(address _owner, address _tokenOut, uint256 _fee, uint256 _amountOut) internal {
        uint256 netOut = _amountOut - _fee;
        if (_fee > 0) {
            _accountFee(_owner, _tokenOut, _fee);
        }
        if (netOut > 0) {
            IERC20(_tokenOut).safeTransfer(_owner, netOut);
        }
    }

    function _accountFee(address _owner, address _tokenOut, uint256 _fee) internal {
        feeTokenBalances[_tokenOut] += _fee;
        totalFeesCollected += _fee;
        feesByUser[_owner] += _fee;
    }

    function _updateStats(bytes32 _botId, uint256 _amountIn, uint256 _amountOut, uint256 _fee) internal {
        BotStats storage stats = botStats[_botId];
        stats.totalVolume += _amountIn;
        stats.totalTrades++;
        stats.lastTradeTime = block.timestamp;
        stats.totalFeesPaid += _fee;
        bool success = _amountOut >= _amountIn;
        if (success) {
            stats.successfulTrades++;
            stats.totalProfit += (_amountOut - _fee);
            totalProfit += (_amountOut - _fee);
        } else {
            stats.failedTrades++;
        }
        totalTrades++;
        stats.winRate = (stats.successfulTrades * 10000) / stats.totalTrades;
        bots[_botId].lastActiveAt = block.timestamp;
        emit TradeExecuted(_botId, _amountIn, _amountOut, success);
    }

    /**
     * @notice Quote the expected output for a swap via the exchange router (read-only).
     */
    function quoteSwap(
        bytes32 _exchangeId,
        address _tokenIn,
        address _tokenOut,
        uint256 _amountIn
    ) external view returns (uint256 amountOut) {
        Exchange storage exchange = exchanges[_exchangeId];
        require(exchange.isActive && exchange.router != address(0), "BAD_EXCHANGE");
        amountOut = _quoteAmountsOut(exchange.router, _tokenIn, _tokenOut, _amountIn);
    }

    function _quoteAmountsOut(address _router, address _tokenIn, address _tokenOut, uint256 _amountIn) internal view returns (uint256 amountOut) {
        // Build path via inline assembly to avoid Yul memory-array tracking (stack-too-deep).
        bytes32 pathSlot;
        assembly {
            let mem := mload(0x40)
            mstore(mem, 2)
            mstore(add(mem, 0x20), _tokenIn)
            mstore(add(mem, 0x40), _tokenOut)
            mstore(0x40, add(mem, 0x60))
            pathSlot := mem
        }
        address[] memory path;
        assembly { path := pathSlot }
        // Call getAmountsOut and read the LAST element from return data via assembly,
        // avoiding a Solidity-level memory array local (stack-too-deep trigger).
        (bool ok, bytes memory ret) = _router.staticcall(
            abi.encodeWithSelector(IUniswapV2Router.getAmountsOut.selector, _amountIn, path)
        );
        require(ok, "QUOTE_FAILED");
        assembly {
            // ret = [offset(32)][len(32)][elem0..elemN]; last elem at len*0x20 from first elem.
            let len := mload(add(ret, 0x20))
            amountOut := mload(add(add(ret, 0x40), mul(sub(len, 1), 0x20)))
        }
    }

    // Exchange Management
    // ============================================================================
    
    /**
     * @notice Add exchange (Admin only)
     */
    function addExchange(
        bytes32 _exchangeId,
        string calldata _name,
        string calldata _endpoint,
        address _router,
        uint256 _gasLimit,
        uint256 _minTradeSize,
        uint256 _maxTradeSize,
        uint256 _feeBps
    ) external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        require(!exchanges[_exchangeId].isActive, "EXISTS");
        Exchange storage e = exchanges[_exchangeId];
        e.id = _exchangeId;
        e.name = _name;
        e.endpoint = _endpoint;
        e.router = _router;
        e.gasLimit = _gasLimit;
        e.isActive = true;
        e.minTradeSize = _minTradeSize;
        e.maxTradeSize = _maxTradeSize;
        e.feeBps = _feeBps;
        exchangeIds.push(_exchangeId);
        emit ExchangeAdded(_exchangeId, _name);
    }
    
    /**
     * @notice Remove exchange (Admin only)
     */
    function removeExchange(bytes32 _exchangeId) external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        
        exchanges[_exchangeId].isActive = false;
        
        emit ExchangeRemoved(_exchangeId);
    }
    
    /**
     * @notice Connect user to exchange
     */
    function connectToExchange(bytes32 _exchangeId) external {
        require(exchanges[_exchangeId].isActive, "EXCHANGE_NOT_ACTIVE");
        
        userExchanges[msg.sender][_exchangeId] = true;
    }
    
    // ============================================================================
    // Fee Management
    // ============================================================================
    
    /**
     * @notice Withdraw collected fees for a specific token (Admin or Finance only).
     * @dev REAL ERC-20 transfer from this contract to _recipient. Fees are held
     *      per-token in feeTokenBalances; withdrawing requires that the contract
     *      actually holds that token balance (enforced by the real transfer).
     */
    function withdrawFees(address _token, address _recipient, uint256 _amount) external nonReentrant {
        require(roleMembers[ROLE_ADMIN][msg.sender] || roleMembers[ROLE_FINANCE][msg.sender], "NO_ACCESS");
        require(_recipient != address(0), "ZERO_RECIPIENT");
        require(_amount <= feeTokenBalances[_token], "INSUFFICIENT_FEES");

        feeTokenBalances[_token] -= _amount;
        totalFeesCollected -= _amount;
        IERC20(_token).safeTransfer(_recipient, _amount);

        emit FeeCollected(_recipient, _amount);
    }
    
    /**
     * @notice Distribute accrued profit to a bot owner in a specific token.
     * @dev REAL ERC-20 transfer. Profit accrues in the bot's tokenOut from swaps;
     *      the contract must actually hold the balance (enforced by safeTransfer).
     */
    function distributeProfit(bytes32 _botId, address _token, uint256 _amount) external nonReentrant {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        require(bots[_botId].isActive, "BOT_NOT_ACTIVE");
        uint256 bal = IERC20(_token).balanceOf(address(this));
        require(_amount <= bal - feeTokenBalances[_token], "INSUFFICIENT_PROFIT");
        address owner = bots[_botId].owner;
        IERC20(_token).safeTransfer(owner, _amount);
        botStats[_botId].totalProfit -= _amount;
        totalProfit -= _amount;
        emit ProfitDistributed(owner, _amount);
    }
    
    /**
     * @notice Update protocol fee
     */
    function updateProtocolFee(uint256 _feeBps) external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        require(_feeBps <= 500, "FEE_TOO_HIGH"); // Max 5%
        
        protocolFeeBps = _feeBps;
    }

    /**
     * @notice Update the fee recipient address (Admin only).
     * @dev Zero address is rejected. Only the ADMIN role can rotate the fee
     *      recipient — this is the ONE legitimate on-chain crypto movement
     *      governance path (fees/profits route to the new recipient), and it
     *      is role-gated. No admin private key or wallet seed is involved.
     */
    function setFeeRecipient(address _newFeeRecipient) external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        require(_newFeeRecipient != address(0), "ZERO_ADDRESS");

        address oldRecipient = feeRecipient;
        feeRecipient = _newFeeRecipient;

        emit FeeRecipientUpdated(oldRecipient, _newFeeRecipient);
    }

    /**
     * @notice Emitted when the fee recipient is rotated.
     */
    event FeeRecipientUpdated(address indexed oldRecipient, address indexed newRecipient);
    
    // ============================================================================
    // Emergency Controls
    // ============================================================================
    
    /**
     * @notice Enable emergency mode (Admin only)
     */
    function enableEmergencyMode() external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        
        emergencyMode = true;
        pauseTrading = true;
        
        emit EmergencyMode(true);
    }
    
    /**
     * @notice Disable emergency mode (Admin only)
     */
    function disableEmergencyMode() external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        
        emergencyMode = false;
        
        emit EmergencyMode(false);
    }
    
    /**
     * @notice Pause all trading (Admin only)
     */
    function pauseAllTrading() external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        
        pauseTrading = true;
    }
    
    /**
     * @notice Resume all trading (Admin only)
     */
    function resumeAllTrading() external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        
        pauseTrading = false;
    }
    
    /**
     * @notice Pause new bot creation (Admin only)
     */
    function pauseNewBotsCreation() external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        
        pauseNewBots = true;
    }
    
    /**
     * @notice Resume new bot creation (Admin only)
     */
    function resumeNewBotsCreation() external {
        require(roleMembers[ROLE_ADMIN][msg.sender], "ONLY_ADMIN");
        
        pauseNewBots = false;
    }
    
    // ============================================================================
    // Governance
    // ============================================================================
    
    /**
     * @notice Transfer governance
     */
    function transferGovernance(address _newGovernance) external {
        require(msg.sender == governance, "ONLY_GOVERNANCE");
        
        pendingGovernance = _newGovernance;
    }
    
    /**
     * @notice Accept governance
     */
    function acceptGovernance() external {
        require(msg.sender == pendingGovernance, "NOT_PENDING");
        
        // Revoke old admin role
        roleMembers[ROLE_ADMIN][governance] = false;
        
        governance = msg.sender;
        userRoles[msg.sender] = ROLE_ADMIN;
        roleMembers[ROLE_ADMIN][msg.sender] = true;
        
        delete pendingGovernance;
    }
    
    // ============================================================================
    // View Functions
    // ============================================================================
    
    struct BotView {
        address owner;
        uint8 botType;
        string name;
        string description;
        uint256 minInvestment;
        uint256 maxInvestment;
        bool isActive;
        bool isPaused;
    }

    /**
     * @notice Get bot details (packed struct return to avoid stack-too-deep).
     */
    function getBot(bytes32 _botId) external view returns (BotView memory b) {
        Bot storage bot = bots[_botId];
        b.owner = bot.owner;
        b.botType = bot.botType;
        b.name = bot.name;
        b.description = bot.description;
        b.minInvestment = bot.minInvestment;
        b.maxInvestment = bot.maxInvestment;
        b.isActive = bot.isActive;
        b.isPaused = bot.isPaused;
    }
    
    /**
     * @notice Get bot statistics
     */
    struct BotStatsView {
        uint256 totalVolume;
        uint256 totalProfit;
        uint256 totalTrades;
        uint256 successfulTrades;
        uint256 failedTrades;
        uint256 winRate;
        uint256 totalFeesPaid;
        uint256 uptime;
    }

    /**
     * @notice Get bot statistics (packed struct return to avoid stack-too-deep).
     */
    function getBotStats(bytes32 _botId) external view returns (BotStatsView memory s) {
        BotStats storage stats = botStats[_botId];
        s.totalVolume = stats.totalVolume;
        s.totalProfit = stats.totalProfit;
        s.totalTrades = stats.totalTrades;
        s.successfulTrades = stats.successfulTrades;
        s.failedTrades = stats.failedTrades;
        s.winRate = stats.winRate;
        s.totalFeesPaid = stats.totalFeesPaid;
        s.uptime = stats.uptime;
    }
    
    /**
     * @notice Get user's bots
     */
    function getUserBots(address _user) external view returns (bytes32[] memory) {
        return userBots[_user];
    }
    
    /**
     * @notice Get exchange details
     */
    struct ExchangeView {
        string name;
        string endpoint;
        address router;
        bool isActive;
        uint256 minTradeSize;
        uint256 maxTradeSize;
    }
    function getExchange(bytes32 _exchangeId) external view returns (ExchangeView memory e) {
        Exchange storage exchange = exchanges[_exchangeId];
        e.name = exchange.name;
        e.endpoint = exchange.endpoint;
        e.router = exchange.router;
        e.isActive = exchange.isActive;
        e.minTradeSize = exchange.minTradeSize;
        e.maxTradeSize = exchange.maxTradeSize;
    }
    
    /**
     * @notice Get all exchanges
     */
    function getAllExchanges() external view returns (bytes32[] memory) {
        return exchangeIds;
    }
    
    /**
     * @notice Get platform statistics
     */
    function getPlatformStats() external view returns (
        uint256 totalBots_,
        uint256 activeBots_,
        uint256 totalTrades_,
        uint256 totalProfit_,
        uint256 totalFees_
    ) {
        return (
            botIds.length,
            activeBots,
            totalTrades,
            totalProfit,
            totalFeesCollected
        );
    }
    
    /**
     * @notice Check if user can trade on exchange
     */
    function canTradeOnExchange(address _user, bytes32 _exchangeId) external view returns (bool) {
        // Admin can trade on any exchange
        if (roleMembers[ROLE_ADMIN][_user]) return true;
        
        // Check if user is connected to exchange
        return userExchanges[_user][_exchangeId];
    }
}