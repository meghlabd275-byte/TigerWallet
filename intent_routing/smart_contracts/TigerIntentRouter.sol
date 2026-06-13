// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title TigerIntentRouter
 * @dev Intent-Based Cross-Chain Router
 * @dev UniswapX/CoW Swap style intent execution
 */
contract TigerIntentRouter {
    // Events
    event IntentCreated(bytes32 indexed intentHash, address indexed user, Intent intent);
    event IntentFilled(bytes32 indexed intentHash, address indexed solver, uint256 filledAmount);
    event IntentCancelled(bytes32 indexed intentHash);
    event SolverRegistered(address indexed solver, uint256 stake);
    event SolverDeregistered(address indexed solver);
    event OrderFilled(
        bytes32 indexed orderHash,
        address indexed filler,
        uint256 filledAmount,
        uint256 price
    );
    event FeeUpdated(uint256 fee);
    event SettlementExecuted(
        address indexed user,
        address indexed solver,
        uint256 amountIn,
        uint256 amountOut
    );

    // Constants
    uint256 public constant FEE_DENOMINATOR = 10000;
    uint256 public constant MIN_STAKE = 1e18;

    // Structs
    struct Intent {
        address owner;
        address[] tokensIn;
        address[] tokensOut;
        uint256[] amountsIn;
        uint256[] amountsOutMin;
        uint256[] prices;
        uint256 expiry;
        uint256 nonce;
        uint256 auctionId;
        bool filled;
    }

    struct Order {
        address owner;
        address sellToken;
        address buyToken;
        uint256 sellAmount;
        uint256 buyAmount;
        uint256 filledAmount;
        uint256 fee;
        uint256 auctionId;
        uint256 deadline;
        bytes data;
        bool filled;
    }

    struct Solver {
        address solver;
        uint256 stake;
        uint256 filledAmounts;
        uint256 totalVolume;
        bool active;
    }

    // State
    mapping(bytes32 => Intent) public intents;
    mapping(bytes32 => Order) public orders;
    mapping(address => Solver) public solvers;
    mapping(address => uint256) public solverIndex;
    address[] public solverList;

    // Fee
    uint256 public fee = 30; // 0.3%

    // Fill amount
    uint256 public fillAmount;

    // Auction
    uint256 public auctionId;
    mapping(uint256 => uint256) public auctionPrices;

    // Order book
    mapping(address => mapping(address => bytes32[])) public orderBook;

    // Modifiers
    modifier onlySolver() {
        require(solvers[msg.sender].active, "Not solver");
        _;
    }

    /**
     * @dev Create intent
     */
    function createIntent(
        address[] memory tokensIn,
        address[] memory tokensOut,
        uint256[] memory amountsIn,
        uint256[] memory amountsOutMin,
        uint256[] memory prices,
        uint256 expiry
    ) external returns (bytes32 intentHash) {
        require(tokensIn.length == tokensOut.length, "Length mismatch");
        require(tokensIn.length == amountsIn.length, "Length mismatch");
        
        Intent memory intent = Intent({
            owner: msg.sender,
            tokensIn: tokensIn,
            tokensOut: tokensOut,
            amountsIn: amountsIn,
            amountsOutMin: amountsOutMin,
            prices: prices,
            expiry: expiry,
            nonce: 0,
            auctionId: auctionId,
            filled: false
        });
        
        intentHash = keccak256(abi.encode(intent));
        intents[intentHash] = intent;
        
        emit IntentCreated(intentHash, msg.sender, intent);
    }

    /**
     * @dev Fill intent
     */
    function fillIntent(
        bytes32 intentHash,
        uint256 fillAmount_
    ) external onlySolver returns (uint256 amountOut) {
        Intent storage intent = intents[intentHash];
        require(intent.owner != address(0), "Not found");
        require(!intent.filled, "Already filled");
        require(block.timestamp <= intent.expiry, "Expired");
        
        // Calculate output
        amountOut = fillAmount_ * intent.prices[0] / 1e18;
        
        // Transfer tokens
        IERC20(intent.tokensIn[0]).transferFrom(msg.sender, address(this), fillAmount_);
        IERC20(intent.tokensOut[0]).transfer(msg.sender, amountOut);
        
        // Update fill amount
        fillAmount += fillAmount_;
        intent.filled = true;
        
        emit IntentFilled(intentHash, msg.sender, fillAmount_);
    }

    /**
     * @dev Cancel intent
     */
    function cancelIntent(bytes32 intentHash) external {
        Intent storage intent = intents[intentHash];
        require(intent.owner == msg.sender, "Not owner");
        
        intent.filled = true;
        
        emit IntentCancelled(intentHash);
    }

    /**
     * @dev Create order
     */
    function createOrder(
        address sellToken,
        address buyToken,
        uint256 sellAmount,
        uint256 buyAmount,
        uint256 deadline,
        bytes memory data
    ) external returns (bytes32 orderHash) {
        Order memory order = Order({
            owner: msg.sender,
            sellToken: sellToken,
            buyToken: buyToken,
            sellAmount: sellAmount,
            buyAmount: buyAmount,
            filledAmount: 0,
            fee: fee,
            auctionId: auctionId,
            deadline: deadline,
            data: data,
            filled: false
        });
        
        orderHash = keccak256(abi.encode(order));
        orders[orderHash] = order;
        
        // Add to order book
        orderBook[sellToken][buyToken].push(orderHash);
    }

    /**
     * @dev Fill order
     */
    function fillOrder(
        bytes32 orderHash,
        uint256 fillAmount_
    ) external onlySolver returns (uint256 amountOut) {
        Order storage order = orders[orderHash];
        require(order.owner != address(0), "Not found");
        require(!order.filled, "Already filled");
        require(block.timestamp <= order.deadline, "Expired");
        
        // Calculate output
        amountOut = (fillAmount_ * order.buyAmount) / order.sellAmount;
        
        // Apply fee
        uint256 feeAmount = amountOut * order.fee / FEE_DENOMINATOR;
        amountOut -= feeAmount;
        
        // Transfer tokens
        IERC20(order.sellToken).transferFrom(msg.sender, order.owner, fillAmount_);
        IERC20(order.buyToken).transfer(msg.sender, amountOut);
        
        // Update filled amount
        order.filledAmount += fillAmount_;
        
        if (order.filledAmount >= order.sellAmount) {
            order.filled = true;
        }
        
        emit OrderFilled(orderHash, msg.sender, fillAmount_, amountOut);
    }

    /**
     * @dev Register solver
     */
    function registerSolver() external payable {
        require(msg.value >= MIN_STAKE, "Stake too low");
        require(!solvers[msg.sender].active, "Already registered");
        
        solvers[msg.sender] = Solver({
            solver: msg.sender,
            stake: msg.value,
            filledAmounts: 0,
            totalVolume: 0,
            active: true
        });
        
        solverList.push(msg.sender);
        
        emit SolverRegistered(msg.sender, msg.value);
    }

    /**
     * @dev Deregister solver
     */
    function deregisterSolver() external {
        require(solvers[msg.sender].active, "Not registered");
        
        solvers[msg.sender].active = false;
        
        // Return stake
        payable(msg.sender).transfer(solvers[msg.sender].stake);
        
        emit SolverDeregistered(msg.sender);
    }

    /**
     * @dev Update fee
     */
    function setFee(uint256 _fee) external {
        require(_fee <= 1000, "Fee too high");
        fee = _fee;
        
        emit FeeUpdated(_fee);
    }

    /**
     * @dev Start new auction
     */
    function startAuction() external {
        auctionId++;
    }

    /**
     * @dev Get order book
     */
    function getOrderBook(
        address sellToken,
        address buyToken
    ) external view returns (bytes32[] memory) {
        return orderBook[sellToken][buyToken];
    }

    /**
     * @dev Get active solvers
     */
    function getActiveSolvers() external view returns (address[] memory) {
        uint256 count;
        for (uint256 i = 0; i < solverList.length; ) {
            if (solvers[solverList[i]].active) {
                count++;
            }
            unchecked {
                i++;
            }
        }
        
        address[] memory result = new address[](count);
        uint256 index;
        for (uint256 i = 0; i < solverList.length; ) {
            if (solvers[solverList[i]].active) {
                result[index] = solverList[i];
                index++;
            }
            unchecked {
                i++;
            }
        }
        
        return result;
    }

    receive() external payable {}
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

    function transfer(
        address to,
        uint256 amount
    ) external returns (bool);

    function balanceOf(address account) external view returns (uint256);
}