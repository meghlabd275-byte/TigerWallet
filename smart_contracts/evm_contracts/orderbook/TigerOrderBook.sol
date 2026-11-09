// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../libraries/SafeMath.sol";
import "../interfaces/ITigerPriceOracle.sol";

/**
 * @title TigerOrderBook
 * @notice Collateralised central limit order book for perpetual markets.
 * @dev This contract holds user collateral. Three properties are load bearing:
 *
 *      1. Every price used for a fill, a position valuation or a liquidation is
 *         read from a configured oracle and rejected if it is missing, stale or
 *         non-positive. Settling against a constant would fill orders at a rate
 *         that does not exist and hand the difference to whoever traded against
 *         it.
 *      2. The matching loop always makes progress. A resting level is either
 *         filled, which consumes quantity, or removed, which shrinks the book,
 *         so the loop cannot spin on a single level.
 *      3. Balances are held internally and withdrawn by their owner. A push
 *         transfer would let a reverting or gas-griefing recipient block the
 *         match that pays them, and would re-enter this contract mid-fill.
 *
 *      Prices and amounts are 18-decimal fixed point throughout. Size is signed:
 *      positive is long, negative is short.
 */

interface IERC20 {
    function transfer(address to, uint256 amount) external returns (bool);

    function transferFrom(address from, address to, uint256 amount) external returns (bool);

    function balanceOf(address account) external view returns (uint256);
}

contract TigerOrderBook {
    using SafeMath for uint256;
    using SafeMath for uint128;

    // ============== Constants ==============

    uint8 internal constant ORDER_TYPE_LIMIT = 1;
    uint8 internal constant ORDER_TYPE_MARKET = 2;
    uint8 internal constant ORDER_TYPE_STOP_LOSS = 3;

    uint8 internal constant SIDE_BUY = 1;
    uint8 internal constant SIDE_SELL = 2;

    uint8 internal constant STATUS_OPEN = 1;
    uint8 internal constant STATUS_FILLED = 2;
    uint8 internal constant STATUS_CANCELLED = 3;
    uint8 internal constant STATUS_EXPIRED = 4;

    /// @dev Fee and ratio denominators, in basis points.
    uint256 internal constant FEE_DENOMINATOR = 10000;

    /// @dev Fixed-point scale for prices and quantities.
    uint256 internal constant PRICE_SCALE = 1e18;

    /// @dev A position is liquidatable when its equity falls below the
    ///      maintenance margin requirement. 500 bps = 5% of notional.
    uint256 internal constant MAINTENANCE_MARGIN_BPS = 500;

    /// @dev Upper bound on the fee the governance address may set, so a
    ///      misconfigured or compromised key cannot take the whole margin.
    uint256 internal constant MAX_FEE_BPS = 500;

    /// @dev Ceiling on resting levels per side per market. Removal is linear in
    ///      the number of levels, so an unbounded book would make cancelling
    ///      too expensive to complete.
    uint256 internal constant MAX_BOOK_LEVELS = 500;

    // ============== Types ==============

    struct Market {
        address indexToken;
        address quoteToken;
        uint256 maxPriceAge;
        uint256 maxLeverage;
        uint256 liquidationPenaltyBps;
        bool active;
    }

    struct Order {
        address owner;
        bytes32 marketId;
        uint8 orderType;
        uint8 side;
        uint256 price;
        uint256 quantity;
        uint256 filledQuantity;
        uint256 timestamp;
        uint256 expiry;
        bool reduceOnly;
        uint8 status;
        /// @dev Quote collateral escrowed against this order. It funds the
        ///      position margin as the order fills and is returned on cancel.
        uint256 margin;
    }

    struct BookLevel {
        uint256 price;
        bytes32 orderHash;
    }

    /// @dev Levels are kept sorted by price ascending for both sides. The best
    ///      bid is the last level and the best ask is the first, so one
    ///      comparator serves both.
    struct Book {
        BookLevel[] levels;
        /// @dev orderHash => index + 1; zero means the order is not resting.
        mapping(bytes32 => uint256) positionOf;
    }

    struct Position {
        address owner;
        bytes32 marketId;
        int256 size;
        uint256 entryPrice;
        uint256 margin;
        uint256 openNotional;
        uint256 lastUpdated;
    }

    struct Fill {
        address maker;
        address taker;
        bytes32 marketId;
        uint256 price;
        uint256 quantity;
        uint256 fee;
        bytes32 makerOrderHash;
        bytes32 takerOrderHash;
        uint256 timestamp;
    }

    // ============== State ==============

    address public governance;
    address public insuranceFund;
    address public feeRecipient;
    address public priceOracle;

    uint256 public makerFeeBps = 10;   // 0.10%
    uint256 public takerFeeBps = 30;   // 0.30%

    bool public paused;

    /// @dev Quote collateral held for each account, withdrawn by its owner.
    mapping(address => uint256) public collateralBalance;

    /// @dev Fees collected for `feeRecipient`, and penalties for `insuranceFund`.
    uint256 public feePool;
    uint256 public insuranceBalance;

    mapping(bytes32 => Market) public markets;
    bytes32[] public marketIds;

    mapping(bytes32 => Order) internal orders;
    uint256 public orderCount;

    mapping(bytes32 => Book) internal bids;
    mapping(bytes32 => Book) internal asks;

    mapping(address => mapping(bytes32 => Position)) internal positions;
    mapping(address => bytes32[]) public userOrderIds;
    mapping(bytes32 => Fill[]) public fillHistory;
    uint256 public fillCount;

    uint256 private _reentrancyState = 1;

    // ============== Events ==============

    event CollateralDeposited(address indexed account, address indexed token, uint256 amount);
    event CollateralWithdrawn(address indexed account, address indexed token, uint256 amount);
    event OrderCreated(
        bytes32 indexed orderHash, address indexed owner, bytes32 marketId,
        uint8 orderType, uint8 side, uint256 price, uint256 quantity
    );
    event OrderFilled(
        bytes32 indexed orderHash, address indexed owner, bytes32 marketId,
        uint256 price, uint256 quantity, uint256 filledQuantity
    );
    event OrderCancelled(bytes32 indexed orderHash, address indexed owner, bytes32 marketId, string reason);
    event PositionOpened(address indexed owner, bytes32 marketId, uint8 side, uint256 size, uint256 entryPrice, uint256 margin);
    event PositionClosed(address indexed owner, bytes32 marketId, int256 realizedPnl, uint256 exitPrice);
    event PositionLiquidated(address indexed owner, bytes32 marketId, uint256 liquidationPrice, uint256 penalty);
    event MarketCreated(bytes32 indexed marketId, address indexToken, address quoteToken, uint256 maxLeverage);
    event MarketUpdated(bytes32 indexed marketId, uint256 maxLeverage, uint256 liquidationPenaltyBps, bool active);
    event PriceOracleUpdated(address indexed oracle);
    event FeesUpdated(uint256 makerFeeBps, uint256 takerFeeBps);
    event TreasuryUpdated(address indexed insuranceFund, address indexed feeRecipient);

    // ============== Modifiers ==============

    modifier onlyGovernance() {
        require(msg.sender == governance, "ONLY_GOVERNANCE");
        _;
    }

    modifier nonReentrant() {
        require(_reentrancyState == 1, "REENTRANT");
        _reentrancyState = 2;
        _;
        _reentrancyState = 1;
    }

    modifier whenNotPaused() {
        require(!paused, "PAUSED");
        _;
    }

    // ============== Construction ==============

    constructor(address _priceOracle) {
        require(_priceOracle != address(0), "INVALID_ORACLE");
        governance = msg.sender;
        insuranceFund = msg.sender;
        feeRecipient = msg.sender;
        priceOracle = _priceOracle;
    }

    // ============== Governance ==============

    function setPriceOracle(address _priceOracle) external onlyGovernance {
        require(_priceOracle != address(0), "INVALID_ORACLE");
        priceOracle = _priceOracle;
        emit PriceOracleUpdated(_priceOracle);
    }

    function setFees(uint256 _makerFeeBps, uint256 _takerFeeBps) external onlyGovernance {
        require(_makerFeeBps <= MAX_FEE_BPS && _takerFeeBps <= MAX_FEE_BPS, "FEE_TOO_HIGH");
        makerFeeBps = _makerFeeBps;
        takerFeeBps = _takerFeeBps;
        emit FeesUpdated(_makerFeeBps, _takerFeeBps);
    }

    function setTreasury(address _insuranceFund, address _feeRecipient) external onlyGovernance {
        require(_insuranceFund != address(0) && _feeRecipient != address(0), "INVALID_ADDRESS");
        insuranceFund = _insuranceFund;
        feeRecipient = _feeRecipient;
        emit TreasuryUpdated(_insuranceFund, _feeRecipient);
    }

    function setPaused(bool _paused) external onlyGovernance {
        paused = _paused;
    }

    function transferGovernance(address _newGovernance) external onlyGovernance {
        require(_newGovernance != address(0), "INVALID_ADDRESS");
        governance = _newGovernance;
    }

    // ============== Market Management ==============

    function createMarket(
        bytes32 _marketId,
        address _indexToken,
        address _quoteToken,
        uint256 _maxLeverage,
        uint256 _maxPriceAge,
        uint256 _liquidationPenaltyBps
    ) external onlyGovernance {
        require(_marketId != bytes32(0), "INVALID_MARKET");
        require(!markets[_marketId].active, "MARKET_EXISTS");
        require(_indexToken != address(0) && _quoteToken != address(0), "INVALID_TOKEN");
        require(_maxLeverage >= 1, "INVALID_LEVERAGE");
        require(_maxPriceAge > 0, "INVALID_PRICE_AGE");
        require(_liquidationPenaltyBps <= MAX_FEE_BPS, "PENALTY_TOO_HIGH");

        markets[_marketId] = Market({
            indexToken: _indexToken,
            quoteToken: _quoteToken,
            maxPriceAge: _maxPriceAge,
            maxLeverage: _maxLeverage,
            liquidationPenaltyBps: _liquidationPenaltyBps,
            active: true
        });
        marketIds.push(_marketId);

        emit MarketCreated(_marketId, _indexToken, _quoteToken, _maxLeverage);
    }

    function updateMarket(
        bytes32 _marketId,
        uint256 _maxLeverage,
        uint256 _liquidationPenaltyBps,
        bool _active
    ) external onlyGovernance {
        require(markets[_marketId].indexToken != address(0), "UNKNOWN_MARKET");
        require(_maxLeverage >= 1, "INVALID_LEVERAGE");
        require(_liquidationPenaltyBps <= MAX_FEE_BPS, "PENALTY_TOO_HIGH");

        Market storage market = markets[_marketId];
        market.maxLeverage = _maxLeverage;
        market.liquidationPenaltyBps = _liquidationPenaltyBps;
        market.active = _active;

        emit MarketUpdated(_marketId, _maxLeverage, _liquidationPenaltyBps, _active);
    }

    // ============== Collateral ==============

    /**
     * @notice Deposit quote collateral.
     * @dev The amount credited is the balance change, not the requested amount,
     *      so a fee-on-transfer or rebasing token cannot credit more than it
     *      actually delivered.
     */
    function depositCollateral(bytes32 _marketId, uint256 _amount) external nonReentrant whenNotPaused {
        Market storage market = markets[_marketId];
        require(market.quoteToken != address(0), "UNKNOWN_MARKET");
        require(_amount > 0, "INVALID_AMOUNT");

        IERC20 token = IERC20(market.quoteToken);
        uint256 before = token.balanceOf(address(this));
        require(token.transferFrom(msg.sender, address(this), _amount), "TRANSFER_FAILED");
        uint256 received = token.balanceOf(address(this)) - before;
        require(received > 0, "NO_TOKENS_RECEIVED");

        collateralBalance[msg.sender] += received;
        emit CollateralDeposited(msg.sender, market.quoteToken, received);
    }

    function withdrawCollateral(bytes32 _marketId, uint256 _amount) external nonReentrant {
        Market storage market = markets[_marketId];
        require(market.quoteToken != address(0), "UNKNOWN_MARKET");
        require(_amount > 0, "INVALID_AMOUNT");
        require(collateralBalance[msg.sender] >= _amount, "INSUFFICIENT_COLLATERAL");

        collateralBalance[msg.sender] -= _amount;
        require(IERC20(market.quoteToken).transfer(msg.sender, _amount), "TRANSFER_FAILED");

        emit CollateralWithdrawn(msg.sender, market.quoteToken, _amount);
    }

    function claimFees(bytes32 _marketId) external nonReentrant {
        require(msg.sender == feeRecipient, "NOT_FEE_RECIPIENT");
        Market storage market = markets[_marketId];
        require(market.quoteToken != address(0), "UNKNOWN_MARKET");

        uint256 amount = feePool;
        require(amount > 0, "NOTHING_TO_CLAIM");
        feePool = 0;
        require(IERC20(market.quoteToken).transfer(feeRecipient, amount), "TRANSFER_FAILED");
    }

    function claimInsurance(bytes32 _marketId) external nonReentrant {
        require(msg.sender == insuranceFund, "NOT_INSURANCE_FUND");
        Market storage market = markets[_marketId];
        require(market.quoteToken != address(0), "UNKNOWN_MARKET");

        uint256 amount = insuranceBalance;
        require(amount > 0, "NOTHING_TO_CLAIM");
        insuranceBalance = 0;
        require(IERC20(market.quoteToken).transfer(insuranceFund, amount), "TRANSFER_FAILED");
    }

    // ============== Order Creation ==============

    /**
     * @notice Place a resting limit order.
     * @param _margin Quote collateral to escrow against the order. It is
     *        released back to the owner as the order fills or is cancelled.
     */
    function createLimitOrder(
        bytes32 _marketId,
        uint8 _side,
        uint256 _price,
        uint256 _quantity,
        uint256 _expiry,
        uint256 _margin
    ) external nonReentrant whenNotPaused returns (bytes32 orderHash) {
        Market storage market = markets[_marketId];
        require(market.active, "MARKET_NOT_ACTIVE");
        require(_side == SIDE_BUY || _side == SIDE_SELL, "INVALID_SIDE");
        require(_price > 0, "INVALID_PRICE");
        require(_quantity > 0, "INVALID_QUANTITY");
        require(_expiry == 0 || _expiry > block.timestamp, "INVALID_EXPIRY");
        require(_margin > 0, "INVALID_MARGIN");

        // Require enough escrow to margin the whole order at the configured
        // leverage. Without this an order could rest that can never be filled
        // without leaving the position under-margined.
        uint256 required = _quantity * _price / PRICE_SCALE / market.maxLeverage;
        require(_margin >= required, "INSUFFICIENT_MARGIN");

        orderHash = _createOrder(_marketId, _side, _price, _quantity, _expiry, _margin, ORDER_TYPE_LIMIT, false);

        Book storage book = _bookFor(_marketId, _side);
        require(book.levels.length < MAX_BOOK_LEVELS, "BOOK_FULL");
        _insertLevel(book, _price, orderHash);
    }

    /**
     * @notice Place an order that fills immediately against the book.
     * @param _slippageBps Maximum deviation from the oracle price, in bps.
     */
    function createMarketOrder(
        bytes32 _marketId,
        uint8 _side,
        uint256 _quantity,
        uint256 _slippageBps,
        uint256 _margin
    ) external nonReentrant whenNotPaused returns (bytes32 orderHash) {
        Market storage market = markets[_marketId];
        require(market.active, "MARKET_NOT_ACTIVE");
        require(_side == SIDE_BUY || _side == SIDE_SELL, "INVALID_SIDE");
        require(_quantity > 0, "INVALID_QUANTITY");
        require(_slippageBps <= 5000, "SLIPPAGE_TOO_HIGH");
        require(_margin > 0, "INVALID_MARGIN");

        uint256 oraclePrice = _getOraclePrice(_marketId);
        orderHash = _createOrder(_marketId, _side, oraclePrice, _quantity, 0, _margin, ORDER_TYPE_MARKET, false);

        uint256 worstPrice = _side == SIDE_BUY
            ? oraclePrice * (FEE_DENOMINATOR + _slippageBps) / FEE_DENOMINATOR
            : oraclePrice * (FEE_DENOMINATOR - _slippageBps) / FEE_DENOMINATOR;

        _match(orderHash, worstPrice);

        Order storage order = orders[orderHash];
        if (order.filledQuantity < order.quantity) {
            // An immediate order cannot rest, so any unfilled remainder is
            // cancelled and its escrow returned rather than left locked.
            order.status = STATUS_CANCELLED;
            _releaseEscrow(orderHash);
            emit OrderCancelled(orderHash, msg.sender, _marketId, "INSUFFICIENT_LIQUIDITY");
        }
    }

    /**
     * @notice Place a stop order. It is not resting liquidity: it becomes a
     *         market order once the oracle reaches `_stopPrice`.
     */
    function createStopOrder(
        bytes32 _marketId,
        uint8 _side,
        uint256 _stopPrice,
        uint256 _quantity,
        uint256 _margin
    ) external nonReentrant whenNotPaused returns (bytes32 orderHash) {
        Market storage market = markets[_marketId];
        require(market.active, "MARKET_NOT_ACTIVE");
        require(_side == SIDE_BUY || _side == SIDE_SELL, "INVALID_SIDE");
        require(_stopPrice > 0, "INVALID_STOP_PRICE");
        require(_quantity > 0, "INVALID_QUANTITY");
        require(_margin > 0, "INVALID_MARGIN");

        orderHash = _createOrder(_marketId, _side, _stopPrice, _quantity, 0, _margin, ORDER_TYPE_STOP_LOSS, true);
    }

    /**
     * @notice Trigger a stop order once its price condition holds.
     * @dev The fill price is the oracle price at trigger time, never the stop
     *      price, so a gap cannot be traded at a price that no longer exists.
     */
    function triggerStopOrder(bytes32 _orderHash, uint256 _slippageBps) external nonReentrant whenNotPaused {
        Order storage order = orders[_orderHash];
        require(order.owner != address(0), "UNKNOWN_ORDER");
        require(order.status == STATUS_OPEN, "NOT_OPEN");
        require(order.orderType == ORDER_TYPE_STOP_LOSS, "NOT_STOP_ORDER");
        require(_slippageBps <= 5000, "SLIPPAGE_TOO_HIGH");

        uint256 oraclePrice = _getOraclePrice(order.marketId);
        if (order.side == SIDE_BUY) {
            require(oraclePrice >= order.price, "STOP_NOT_TRIGGERED");
        } else {
            require(oraclePrice <= order.price, "STOP_NOT_TRIGGERED");
        }

        order.orderType = ORDER_TYPE_MARKET;
        uint256 worstPrice = order.side == SIDE_BUY
            ? oraclePrice * (FEE_DENOMINATOR + _slippageBps) / FEE_DENOMINATOR
            : oraclePrice * (FEE_DENOMINATOR - _slippageBps) / FEE_DENOMINATOR;

        _match(_orderHash, worstPrice);
        if (order.filledQuantity < order.quantity) {
            order.status = STATUS_CANCELLED;
            _releaseEscrow(_orderHash);
        }
    }

    function cancelOrder(bytes32 _orderHash) external nonReentrant {
        Order storage order = orders[_orderHash];
        require(order.owner == msg.sender, "NOT_OWNER");
        require(order.status == STATUS_OPEN, "NOT_OPEN");

        order.status = STATUS_CANCELLED;

        if (order.orderType == ORDER_TYPE_LIMIT) {
            _removeLevel(_bookFor(order.marketId, order.side), _orderHash);
        }
        _releaseEscrow(_orderHash);

        emit OrderCancelled(_orderHash, msg.sender, order.marketId, "USER_CANCEL");
    }

    // ============== Positions ==============

    function openPosition(
        bytes32 _marketId,
        uint8 _side,
        uint256 _size,
        uint256 _margin
    ) external nonReentrant whenNotPaused returns (int256) {
        Market storage market = markets[_marketId];
        require(market.active, "MARKET_NOT_ACTIVE");
        require(_side == SIDE_BUY || _side == SIDE_SELL, "INVALID_SIDE");
        require(_size > 0, "INVALID_SIZE");
        require(_margin > 0, "INVALID_MARGIN");

        uint256 currentPrice = _getOraclePrice(_marketId);
        uint256 notional = _size * currentPrice / PRICE_SCALE;
        require(_margin >= notional / market.maxLeverage, "INSUFFICIENT_MARGIN");
        require(collateralBalance[msg.sender] >= _margin, "INSUFFICIENT_COLLATERAL");

        collateralBalance[msg.sender] -= _margin;

        int256 realised = _applyFill(msg.sender, _marketId, _side, _size, currentPrice, _margin, false);

        Position storage position = positions[msg.sender][_marketId];
        emit PositionOpened(msg.sender, _marketId, _side, _size, position.entryPrice, position.margin);
        return realised;
    }

    function closePosition(bytes32 _marketId, uint256 _size) external nonReentrant returns (int256) {
        Position storage position = positions[msg.sender][_marketId];
        require(position.size != 0, "NO_POSITION");

        uint256 currentPrice = _getOraclePrice(_marketId);
        uint256 openSize = position.size > 0 ? uint256(position.size) : uint256(-position.size);
        uint256 closing = _size == 0 || _size > openSize ? openSize : _size;

        uint8 side = position.size > 0 ? SIDE_SELL : SIDE_BUY;
        int256 realised = _applyFill(msg.sender, _marketId, side, closing, currentPrice, 0, true);

        emit PositionClosed(msg.sender, _marketId, realised, currentPrice);
        return realised;
    }

    /**
     * @notice Liquidate a position whose equity has fallen below maintenance.
     * @dev Anyone may call. The penalty funds the insurance fund and the caller
     *      is paid a share of it for doing the work.
     */
    function liquidate(address _owner, bytes32 _marketId) external nonReentrant {
        Position storage position = positions[_owner][_marketId];
        require(position.size != 0, "NO_POSITION");

        Market storage market = markets[_marketId];
        uint256 currentPrice = _getOraclePrice(_marketId);

        int256 pnl = _unrealizedPnl(position, currentPrice);
        int256 equitySigned = int256(position.margin) + pnl;
        require(equitySigned <= int256(_maintenanceMargin(position, currentPrice)), "ABOVE_MAINTENANCE");

        uint256 equity = equitySigned > 0 ? uint256(equitySigned) : 0;
        uint256 notional = _absNotional(position, currentPrice);
        uint256 penalty = notional * market.liquidationPenaltyBps / FEE_DENOMINATOR;
        if (penalty > equity) {
            penalty = equity;
        }

        uint256 keeperReward = penalty / 2;
        uint256 toInsurance = penalty - keeperReward;

        insuranceBalance += toInsurance;
        uint256 ownerRefund = equity - penalty;
        if (ownerRefund > 0) {
            collateralBalance[_owner] += ownerRefund;
        }
        if (keeperReward > 0) {
            collateralBalance[msg.sender] += keeperReward;
        }

        delete positions[_owner][_marketId];

        emit PositionLiquidated(_owner, _marketId, currentPrice, penalty);
    }

    // ============== Matching ==============

    /**
     * @notice Match a taker order against the resting book.
     * @dev Each iteration either consumes quantity from the taker or removes a
     *      level, so the loop terminates even if the book is inconsistent.
     */
    function _match(bytes32 _takerHash, uint256 _worstPrice) internal {
        Order storage taker = orders[_takerHash];
        Market storage market = markets[taker.marketId];
        Book storage book = taker.side == SIDE_BUY
            ? asks[taker.marketId]
            : bids[taker.marketId];

        uint256 remaining = taker.quantity - taker.filledQuantity;

        while (remaining > 0 && book.levels.length > 0) {
            uint256 bestIndex = taker.side == SIDE_BUY ? 0 : book.levels.length - 1;
            uint256 price = book.levels[bestIndex].price;

            if (taker.side == SIDE_BUY && price > _worstPrice) break;
            if (taker.side == SIDE_SELL && price < _worstPrice) break;

            bytes32 makerHash = book.levels[bestIndex].orderHash;
            Order storage maker = orders[makerHash];
            uint256 available = maker.quantity - maker.filledQuantity;

            if (available == 0) {
                _removeLevel(book, makerHash);
                continue;
            }

            uint256 fillQuantity = remaining < available ? remaining : available;
            _settleFill(_takerHash, makerHash, market, price, fillQuantity);

            remaining -= fillQuantity;

            if (maker.filledQuantity == maker.quantity) {
                _removeLevel(book, makerHash);
            }
        }

        taker.filledQuantity = taker.quantity - remaining;
    }

    /**
     * @dev Books the two sides of a fill. Margin required for the filled
     *      quantity moves out of each order's escrow and into the position; the
     *      fee is charged to the position margin.
     */
    function _settleFill(
        bytes32 _takerHash,
        bytes32 _makerHash,
        Market storage market,
        uint256 price,
        uint256 fillQuantity
    ) internal {
        Order storage taker = orders[_takerHash];
        Order storage maker = orders[_makerHash];

        uint256 notional = fillQuantity * price / PRICE_SCALE;
        uint256 required = notional / market.maxLeverage;

        uint256 takerUse = required < taker.margin ? required : taker.margin;
        taker.margin -= takerUse;

        uint256 makerUse = required < maker.margin ? required : maker.margin;
        maker.margin -= makerUse;

        _applyFill(taker.owner, taker.marketId, taker.side, fillQuantity, price, takerUse, false);
        _applyFill(maker.owner, maker.marketId, maker.side, fillQuantity, price, makerUse, false);

        // Filled quantity must advance, otherwise the level is never considered
        // consumed and the same resting order is filled repeatedly.
        maker.filledQuantity += fillQuantity;
        taker.filledQuantity += fillQuantity;
        if (maker.filledQuantity >= maker.quantity) {
            maker.status = STATUS_FILLED;
        }
        if (taker.filledQuantity >= taker.quantity) {
            taker.status = STATUS_FILLED;
        }

        uint256 totalFee = notional * (makerFeeBps + takerFeeBps) / FEE_DENOMINATOR;
        _chargeFee(taker.owner, taker.marketId, totalFee);

        fillHistory[taker.marketId].push(Fill({
            maker: maker.owner,
            taker: taker.owner,
            marketId: taker.marketId,
            price: price,
            quantity: fillQuantity,
            fee: totalFee,
            makerOrderHash: _makerHash,
            takerOrderHash: _takerHash,
            timestamp: block.timestamp
        }));
        fillCount++;

        emit OrderFilled(_makerHash, maker.owner, taker.marketId, price, fillQuantity, maker.filledQuantity);
        emit OrderFilled(_takerHash, taker.owner, taker.marketId, price, fillQuantity, taker.filledQuantity);
    }

    /**
     * @dev Charges a fee against a position's margin, and the shortfall against
     *      free collateral. A position is never left with negative margin.
     */
    function _chargeFee(address _trader, bytes32 _marketId, uint256 _fee) internal {
        if (_fee == 0) return;

        Position storage position = positions[_trader][_marketId];
        if (position.margin >= _fee) {
            position.margin -= _fee;
            feePool += _fee;
            return;
        }

        uint256 fromPosition = position.margin;
        position.margin = 0;
        uint256 fromCollateral = _fee - fromPosition;
        uint256 free = collateralBalance[_trader];
        if (fromCollateral > free) {
            fromCollateral = free;
        }
        collateralBalance[_trader] = free - fromCollateral;
        feePool += fromPosition + fromCollateral;
    }

    /**
     * @dev Applies a fill to a trader's position.
     *
     *      `_marginIn` is collateral escrowed for this fill. It is used when the
     *      fill opens or increases a position, and returned to the trader's free
     *      collateral when the fill reduces one.
     */
    function _applyFill(
        address _trader,
        bytes32 _marketId,
        uint8 _side,
        uint256 _quantity,
        uint256 _price,
        uint256 _marginIn,
        bool _forceReduce
    ) internal returns (int256) {
        Position storage position = positions[_trader][_marketId];
        int256 delta = _side == SIDE_BUY ? int256(_quantity) : -int256(_quantity);

        if (position.size == 0) {
            require(!_forceReduce, "NO_POSITION");
            position.owner = _trader;
            position.marketId = _marketId;
            position.size = delta;
            position.entryPrice = _price;
            position.margin = _marginIn;
            position.openNotional = _quantity * _price / PRICE_SCALE;
            position.lastUpdated = block.timestamp;
            return 0;
        }

        if ((position.size > 0) == (delta > 0) && !_forceReduce) {
            _increasePosition(position, _quantity, _price, delta, _marginIn);
            return 0;
        }

        int256 realised = _reducePosition(_trader, position, _quantity, _price, delta);

        // Whatever remains of the position belongs to the trader as free
        // collateral once the position is gone; leaving it on a deleted
        // position would strand it in the contract.
        if (position.size == 0) {
            collateralBalance[_trader] += position.margin + _marginIn;
            delete positions[_trader][_marketId];
        } else {
            position.margin += _marginIn;
            position.lastUpdated = block.timestamp;
        }

        return realised;
    }

    function _increasePosition(
        Position storage _position,
        uint256 _quantity,
        uint256 _price,
        int256 _delta,
        uint256 _marginIn
    ) internal {
        uint256 newNotional = _position.openNotional + _quantity * _price / PRICE_SCALE;
        int256 newSize = _position.size + _delta;
        uint256 absSize = newSize > 0 ? uint256(newSize) : uint256(-newSize);

        _position.entryPrice = newNotional * PRICE_SCALE / absSize;
        _position.size = newSize;
        _position.openNotional = newNotional;
        _position.margin += _marginIn;
        _position.lastUpdated = block.timestamp;
    }

    /**
     * @dev Closes the open part of a position against a fill, then flips the
     *      remainder into a position in the new direction.
     */
    function _reducePosition(
        address _trader,
        Position storage _position,
        uint256 _quantity,
        uint256 _price,
        int256 _delta
    ) internal returns (int256 realisedPnl) {
        uint256 openSize = _position.size > 0 ? uint256(_position.size) : uint256(-_position.size);
        uint256 closing = _quantity < openSize ? _quantity : openSize;

        realisedPnl = _position.size > 0
            ? int256(closing) * (int256(_price) - int256(_position.entryPrice)) / int256(PRICE_SCALE)
            : int256(closing) * (int256(_position.entryPrice) - int256(_price)) / int256(PRICE_SCALE);

        _settleClosedMargin(_trader, _position, closing, openSize, realisedPnl);
        _position.openNotional -= _position.openNotional * closing / openSize;
        _position.size = _position.size > 0
            ? _position.size - int256(closing)
            : _position.size + int256(closing);

        if (_quantity > closing) {
            uint256 remainder = _quantity - closing;
            _position.size = _delta > 0 ? int256(remainder) : -int256(remainder);
            _position.entryPrice = _price;
            _position.openNotional = remainder * _price / PRICE_SCALE;
        }
    }

    /**
     * @dev Releases the margin that backed the closed quantity and settles the
     *      realised result.
     *
     *      The released margin plus the realised gain is credited to the
     *      trader's free collateral, so it can be withdrawn. A realised loss is
     *      taken out of it, and any shortfall beyond it comes out of the margin
     *      still backing the rest of the position.
     */
    function _settleClosedMargin(
        address _trader,
        Position storage _position,
        uint256 _closing,
        uint256 _openSize,
        int256 _realisedPnl
    ) internal {
        uint256 released = _position.margin * _closing / _openSize;
        _position.margin -= released;

        int256 equity = int256(released) + _realisedPnl;
        if (equity > 0) {
            collateralBalance[_trader] += uint256(equity);
            return;
        }

        uint256 shortfall = uint256(-equity);
        _position.margin = shortfall >= _position.margin ? 0 : _position.margin - shortfall;
    }

    // ============== Book Maintenance ==============

    function _bookFor(bytes32 _marketId, uint8 _side) internal view returns (Book storage) {
        return _side == SIDE_BUY ? bids[_marketId] : asks[_marketId];
    }

    /// @dev Inserts keeping the levels sorted by price ascending.
    function _insertLevel(Book storage _book, uint256 _price, bytes32 _orderHash) internal {
        uint256 low = 0;
        uint256 high = _book.levels.length;
        while (low < high) {
            uint256 mid = (low + high) / 2;
            if (_book.levels[mid].price < _price) {
                low = mid + 1;
            } else {
                high = mid;
            }
        }

        _book.levels.push(BookLevel({price: _price, orderHash: _orderHash}));
        for (uint256 i = _book.levels.length - 1; i > low; --i) {
            _book.levels[i] = _book.levels[i - 1];
        }
        _book.levels[low] = BookLevel({price: _price, orderHash: _orderHash});
        _book.positionOf[_orderHash] = low + 1;
    }

    function _removeLevel(Book storage _book, bytes32 _orderHash) internal {
        uint256 stored = _book.positionOf[_orderHash];
        if (stored == 0) return;

        uint256 index = stored - 1;
        for (uint256 i = index; i + 1 < _book.levels.length; ++i) {
            _book.levels[i] = _book.levels[i + 1];
            _book.positionOf[_book.levels[i].orderHash] = i + 1;
        }
        _book.levels.pop();
        delete _book.positionOf[_orderHash];
    }

    /// @dev Returns any escrow still attached to an order to its owner.
    function _releaseEscrow(bytes32 _orderHash) internal {
        Order storage order = orders[_orderHash];
        if (order.margin == 0) return;
        uint256 amount = order.margin;
        order.margin = 0;
        collateralBalance[order.owner] += amount;
    }

    function _createOrder(
        bytes32 _marketId,
        uint8 _side,
        uint256 _price,
        uint256 _quantity,
        uint256 _expiry,
        uint256 _margin,
        uint8 _orderType,
        bool _reduceOnly
    ) internal returns (bytes32 orderHash) {
        require(collateralBalance[msg.sender] >= _margin, "INSUFFICIENT_COLLATERAL");
        collateralBalance[msg.sender] -= _margin;

        orderHash = keccak256(abi.encodePacked(
            msg.sender, _marketId, _orderType, _side, _price, _quantity, block.timestamp, orderCount
        ));
        require(orders[orderHash].owner == address(0), "ORDER_EXISTS");

        orders[orderHash] = Order({
            owner: msg.sender,
            marketId: _marketId,
            orderType: _orderType,
            side: _side,
            price: _price,
            quantity: _quantity,
            filledQuantity: 0,
            timestamp: block.timestamp,
            expiry: _expiry,
            reduceOnly: _reduceOnly,
            status: STATUS_OPEN,
            margin: _margin
        });

        userOrderIds[msg.sender].push(orderHash);
        orderCount++;

        emit OrderCreated(orderHash, msg.sender, _marketId, _orderType, _side, _price, _quantity);
    }

    // ============== Pricing ==============

    /**
     * @notice Reads the oracle price, scaled to 18 decimals.
     * @dev Reverts when the oracle is unset, the answer is non-positive, or the
     *      answer is older than the market's staleness bound. Returning a
     *      constant here would value every position and settle every fill at a
     *      rate that does not exist.
     */
    function _getOraclePrice(bytes32 _marketId) internal view returns (uint256) {
        require(priceOracle != address(0), "ORACLE_NOT_CONFIGURED");

        Market storage market = markets[_marketId];
        require(market.maxPriceAge > 0, "UNKNOWN_MARKET");

        (, int256 answer, , uint256 updatedAt, ) = ITigerPriceOracle(priceOracle).latestRoundData();
        require(answer > 0, "INVALID_ORACLE_PRICE");
        require(updatedAt > 0 && block.timestamp - updatedAt <= market.maxPriceAge, "STALE_ORACLE_PRICE");

        uint8 feedDecimals = ITigerPriceOracle(priceOracle).decimals();
        uint256 scaled = uint256(answer);
        if (feedDecimals < 18) {
            scaled = scaled * (10 ** (18 - feedDecimals));
        } else if (feedDecimals > 18) {
            scaled = scaled / (10 ** (feedDecimals - 18));
        }
        return scaled;
    }

    function getPrice(bytes32 _marketId) external view returns (uint256) {
        return _getOraclePrice(_marketId);
    }

    // ============== Position Maths ==============

    function _unrealizedPnl(Position storage _position, uint256 _price) internal view returns (int256) {
        int256 delta = int256(_price) - int256(_position.entryPrice);
        return _position.size * delta / int256(PRICE_SCALE);
    }

    function _absNotional(Position storage _position, uint256 _price) internal view returns (uint256) {
        uint256 size = _position.size > 0 ? uint256(_position.size) : uint256(-_position.size);
        return size * _price / PRICE_SCALE;
    }

    function _maintenanceMargin(Position storage _position, uint256 _price) internal view returns (uint256) {
        return _absNotional(_position, _price) * MAINTENANCE_MARGIN_BPS / FEE_DENOMINATOR;
    }

    // ============== Views ==============

    function getOrder(bytes32 _orderHash) external view returns (
        address owner,
        bytes32 marketId,
        uint8 orderType,
        uint8 side,
        uint256 price,
        uint256 quantity,
        uint256 filledQuantity,
        uint256 margin,
        uint8 status
    ) {
        Order storage order = orders[_orderHash];
        return (
            order.owner, order.marketId, order.orderType, order.side,
            order.price, order.quantity, order.filledQuantity, order.margin, order.status
        );
    }

    function getPosition(address _owner, bytes32 _marketId) external view returns (
        int256 size,
        uint256 entryPrice,
        int256 unrealizedPnl,
        uint256 margin,
        uint256 openNotional
    ) {
        Position storage position = positions[_owner][_marketId];
        uint256 mark = _getOraclePrice(_marketId);
        return (
            position.size,
            position.entryPrice,
            _unrealizedPnl(position, mark),
            position.margin,
            position.openNotional
        );
    }

    function isLiquidatable(address _owner, bytes32 _marketId) external view returns (bool) {
        Position storage position = positions[_owner][_marketId];
        if (position.size == 0) return false;

        uint256 mark = _getOraclePrice(_marketId);
        int256 equity = int256(position.margin) + _unrealizedPnl(position, mark);
        return equity <= int256(_maintenanceMargin(position, mark));
    }

    function getBestBid(bytes32 _marketId) external view returns (uint256 price, uint256 available) {
        // Bids are sorted ascending, so the highest price is the last level.
        return _bestOf(bids[_marketId], true);
    }

    function getBestAsk(bytes32 _marketId) external view returns (uint256 price, uint256 available) {
        // Asks are sorted ascending, so the lowest price is the first level.
        return _bestOf(asks[_marketId], false);
    }

    function _bestOf(Book storage _book, bool _highest) internal view returns (uint256 price, uint256 available) {
        if (_book.levels.length == 0) return (0, 0);

        uint256 index = _highest ? _book.levels.length - 1 : 0;
        BookLevel storage level = _book.levels[index];
        Order storage order = orders[level.orderHash];
        return (level.price, order.quantity - order.filledQuantity);
    }

    function getOrderBookDepth(
        bytes32 _marketId,
        uint8 _side,
        uint256 _levels
    ) external view returns (uint256[] memory prices, uint256[] memory quantities) {
        Book storage book = _bookFor(_marketId, _side);
        uint256 count = _levels < book.levels.length ? _levels : book.levels.length;

        prices = new uint256[](count);
        quantities = new uint256[](count);

        for (uint256 i = 0; i < count; ++i) {
            // Walk outwards from the best price: the last level for bids, the
            // first for asks.
            uint256 index = _side == SIDE_BUY ? book.levels.length - 1 - i : i;
            BookLevel storage level = book.levels[index];
            Order storage order = orders[level.orderHash];
            prices[i] = level.price;
            quantities[i] = order.quantity - order.filledQuantity;
        }
    }

    function getFillCount(bytes32 _marketId) external view returns (uint256) {
        return fillHistory[_marketId].length;
    }

    function getFill(bytes32 _marketId, uint256 _index) external view returns (Fill memory) {
        return fillHistory[_marketId][_index];
    }

    function getUserOrders(address _user) external view returns (bytes32[] memory) {
        return userOrderIds[_user];
    }

    function getMarketCount() external view returns (uint256) {
        return marketIds.length;
    }
}
