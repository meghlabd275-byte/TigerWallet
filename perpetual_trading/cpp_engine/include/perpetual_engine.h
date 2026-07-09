/**
 * TigerWallet - Perpetual Trading Engine
 * Ultra-low latency C++ engine for perpetual futures trading
 * 
 * Features:
 * - Order book management
 * - Position management
 * - Liquidation engine
 * - Funding rate calculation
 * - Risk management
 * - Margin calculation
 */

#ifndef TIGERWALLET_PERPETUAL_ENGINE_H
#define TIGERWALLET_PERPETUAL_ENGINE_H

#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <memory>
#include <mutex>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <optional>
#include <variant>

namespace tiger {
namespace perpetual {

// ============ Types ============

using OrderID = uint64_t;
using UserID = uint64_t;
using Timestamp = uint64_t;
using Price = int64_t;  // Scaled by 1e8
using Quantity = int64_t; // Scaled by 1e8
using Decimal = __int128; // For precise calculations

// Market types
enum class Market : uint8_t {
    BTC_PERP,
    ETH_PERP,
    SOL_PERP,
    LINK_PERP,
    MATIC_PERP,
    Custom,
};

enum class Side : uint8_t {
    Buy = 0,
    Sell = 1,
};

enum class OrderType : uint8_t {
    Market,
    Limit,
    StopMarket,
    StopLimit,
    TakeProfit,
    TakeProfitLimit,
};

enum class OrderStatus : uint8_t {
    Pending,
    Open,
    PartiallyFilled,
    Filled,
    Cancelled,
    Rejected,
    Expired,
};

enum class PositionSide : uint8_t {
    Long,
    Short,
};

enum class PositionStatus : uint8_t {
    Open,
    Partial,
    Closed,
    Liquidated,
    Settlement,
};

// Margin mode
enum class MarginMode : uint8_t {
    Cross,
    Isolated,
};

// ============ Order ============

struct Order {
    OrderID order_id;
    UserID user_id;
    Market market;
    Side side;
    OrderType order_type;
    OrderStatus status;
    Price price;
    Quantity quantity;
    Quantity filled_quantity;
    Quantity remaining_quantity;
    Price stop_price;
    Price trigger_price;
    bool reduce_only;
    bool post_only;
    Timestamp created_at;
    Timestamp updated_at;
    Timestamp expires_at;
    std::string client_order_id;
    
    // Calculate order value
    Decimal get_notional() const;
    
    // Check if order is active
    bool is_active() const;
};

// ============ Position ============

struct Position {
    UserID user_id;
    Market market;
    PositionSide side;
    PositionStatus status;
    Quantity size;           // Position size (positive = long, negative = short)
    Quantity open_size;      // Open position size
    Price entry_price;      // Average entry price
    Price liquidation_price;
    Price mark_price;
    Price stop_loss;
    Price take_profit;
    Decimal unrealized_pnl; // Unrealized P&L
    Decimal realized_pnl;   // Realized P&L
    Timestamp opened_at;
    Timestamp updated_at;
    
    // Calculate margin requirement
    Decimal get_margin_required(Decimal margin_ratio) const;
    
    // Calculate P&L
    Decimal calculate_pnl(Price current_price) const;
    
    // Check liquidation
    bool should_liquidate(Price current_price) const;
};

// ============ Order Book ============

struct OrderBookLevel {
    Price price;
    Quantity quantity;
    OrderID order_id;
};

struct OrderBook {
    Market market;
    std::vector<OrderBookLevel> bids;  // Buy orders (sorted by price desc)
    std::vector<OrderBookLevel> asks;  // Sell orders (sorted by price asc)
    
    // Get best bid/ask
    std::optional<Price> best_bid() const;
    std::optional<Price> best_ask() const;
    
    // Get spread
    Price get_spread() const;
    
    // Get depth
    std::vector<std::pair<Price, Quantity>> get_depth(int levels) const;
    
    // Add order to book
    void add_order(const Order& order);
    
    // Remove order from book
    void remove_order(OrderID order_id);
    
    // Match orders
    std::vector<Order> match(Order& incoming);
};

// ============ Market Data ============

struct MarketData {
    Market market;
    Price index_price;      // Index price
    Price mark_price;      // Mark price (for P&L)
    Price last_price;      // Last traded price
    Price best_bid;
    Price best_ask;
    Price funding_rate;
    Timestamp funding_time;
    Decimal24h volume;
    Decimal24h open_interest;
    Decimal24h turnover;
    
    // Calculate funding
    Decimal calculate_funding(Decimal position_size) const;
};

struct Trade {
    OrderID trade_id;
    OrderID maker_order_id;
    OrderID taker_order_id;
    UserID maker_user_id;
    UserID taker_user_id;
    Market market;
    Side side;
    Price price;
    Quantity quantity;
    Timestamp timestamp;
    Decimal maker_fee;
    Decimal taker_fee;
};

// ============ Risk Management ============

struct RiskLimits {
    Decimal max_position_size;
    Decimal max_order_size;
    Decimal max_leverage;
    Decimal min_margin_ratio;
    Decimal liquidation_ratio;
    Decimal max_loss_per_trade;
    Decimal daily_loss_limit;
    Decimal max_open_orders;
};

struct AccountRisk {
    UserID user_id;
    Decimal total_collateral;
    Decimal total_margin_used;
    Decimal available_margin;
    Decimal unrealized_pnl;
    Decimal realized_pnl_24h;
    Decimal total_position_value;
    Decimal leverage;
    
    // Calculate margin ratio
    Decimal get_margin_ratio() const;
    
    // Check if account is under-margined
    bool is_undermargined(Decimal min_ratio) const;
    
    // Check if account should be liquidated
    bool should_liquidate(Decimal liq_ratio) const;
};

// ============ Liquidation Engine ============

struct LiquidationResult {
    UserID user_id;
    Market market;
    PositionSide side;
    Quantity liquidated_size;
    Price liquidation_price;
    Decimal penalty;
    Decimal remaining_collateral;
    bool partial;
};

class LiquidationEngine {
public:
    LiquidationEngine(const RiskLimits& limits);
    
    // Check positions for liquidation
    std::vector<LiquidationResult> check_liquidations(
        const std::map<UserID, Position>& positions,
        const std::map<Market, MarketData>& markets
    );
    
    // Execute liquidation
    LiquidationResult execute_liquidation(
        const Position& position,
        const MarketData& market
    );
    
    // Calculate liquidation price
    Price calculate_liquidation_price(
        const Position& position,
        Decimal collateral,
        Decimal margin_ratio
    );
    
private:
    RiskLimits limits_;
    std::mutex mutex_;
    
    // Get liquidation penalty
    Decimal get_liquidation_penalty(Decimal position_value);
};

// ============ Matching Engine ============

class MatchingEngine {
public:
    MatchingEngine();
    
    // Process new order
    std::variant<Order, std::string> process_order(
        const Order& order,
        OrderBook& book,
        AccountRisk& account
    );
    
    // Cancel order
    bool cancel_order(OrderID order_id, OrderBook& book);
    
    // Get market data
    MarketData get_market_data(Market market) const;
    
    // Update market data
    void update_market_data(Market market, const MarketData& data);
    
    // Get trades
    const std::vector<Trade>& get_trades() const { return trades_; }
    
    // Clear trades
    void clear_trades() { trades_.clear(); }
    
private:
    std::map<Market, MarketData> market_data_;
    std::vector<Trade> trades_;
    std::mutex mutex_;
    
    // Match incoming order with book
    std::vector<Trade> match_order(
        Order& order,
        OrderBook& book
    );
    
    // Calculate fees
    Decimal calculate_maker_fee(Decimal notional);
    Decimal calculate_taker_fee(Decimal notional);
};

// ============ Position Manager ============

class PositionManager {
public:
    PositionManager();
    
    // Open position
    Position open_position(
        UserID user_id,
        Market market,
        PositionSide side,
        Quantity size,
        Price entry_price,
        Decimal margin
    );
    
    // Close position
    Position close_position(
        UserID user_id,
        Market market,
        Quantity close_size,
        Price exit_price
    );
    
    // Update position
    void update_position(
        UserID user_id,
        Market market,
        const Position& update
    );
    
    // Get position
    std::optional<Position> get_position(UserID user_id, Market market);
    
    // Get all positions for user
    std::vector<Position> get_user_positions(UserID user_id);
    
    // Realize P&L
    void realize_pnl(UserID user_id, Market market, Decimal pnl);
    
    // Update unrealized P&L
    void update_unrealized_pnl(
        UserID user_id,
        Market market,
        Price mark_price
    );
    
    // Check margin requirements
    bool check_margin(
        UserID user_id,
        Decimal required_margin
    );
    
    // Liquidate position
    void liquidate_position(
        UserID user_id,
        Market market,
        Price liquidation_price
    );

private:
    struct PositionKey {
        UserID user_id;
        Market market;
        
        bool operator<(const PositionKey& other) const {
            return std::tie(user_id, market) < std::tie(other.user_id, other.market);
        }
    };
    
    std::map<PositionKey, Position> positions_;
    std::map<UserID, Decimal> realized_pnl_;
    std::mutex mutex_;
};

// ============ Account Manager ============

class AccountManager {
public:
    AccountManager();
    
    // Deposit collateral
    void deposit(UserID user_id, Decimal amount);
    
    // Withdraw collateral
    bool withdraw(UserID user_id, Decimal amount);
    
    // Get account info
    AccountRisk get_account_risk(UserID user_id);
    
    // Update margin
    void update_margin(UserID user_id, Decimal delta);
    
    // Freeze margin
    void freeze_margin(UserID user_id, Decimal amount);
    
    // Unfreeze margin
    void unfreeze_margin(UserID user_id, Decimal amount);
    
    // Check if can open position
    bool can_open_position(
        UserID user_id,
        Decimal required_margin,
        Decimal order_value
    );

private:
    struct Account {
        UserID user_id;
        Decimal balance;
        Decimal frozen_balance;
        Decimal total_pnl;
        Decimal total_realized_pnl;
        Timestamp last_updated;
    };
    
    std::map<UserID, Account> accounts_;
    std::mutex mutex_;
    
    Account& get_or_create_account(UserID user_id);
};

// ============ Funding Manager ============

struct FundingRate {
    Market market;
    Decimal funding_rate;
    Decimal interest_rate;
    Decimal premium_rate;
    Timestamp next_funding_time;
    Timestamp last_funding_time;
};

class FundingManager {
public:
    FundingManager();
    
    // Calculate funding rate
    FundingRate calculate_funding_rate(
        Market market,
        Price index_price,
        Price mark_price
    );
    
    // Process funding payment
    void process_funding_payment(
        PositionManager& position_mgr,
        UserID user_id,
        Market market,
        Decimal position_size,
        const FundingRate& funding
    );
    
    // Get current funding rate
    const FundingRate& get_funding_rate(Market market) const;
    
    // Update funding rate
    void update_funding_rate(Market market, const FundingRate& rate);

private:
    std::map<Market, FundingRate> funding_rates_;
    std::mutex mutex_;
    
    // Calculate premium
    Decimal calculate_premium(
        Price index_price,
        Price mark_price
    );
};

// ============ Order Manager ============

class OrderManager {
public:
    OrderManager(MatchingEngine& matching_engine);
    
    // Place order
    std::variant<OrderID, std::string> place_order(
        const Order& order
    );
    
    // Cancel order
    bool cancel_order(OrderID order_id);
    
    // Modify order
    bool modify_order(
        OrderID order_id,
        Price new_price,
        Quantity new_quantity
    );
    
    // Get order
    std::optional<Order> get_order(OrderID order_id) const;
    
    // Get user orders
    std::vector<Order> get_user_orders(UserID user_id) const;
    
    // Process expired orders
    void process_expired_orders();
    
    // Get open orders count
    size_t get_open_orders_count(UserID user_id) const;

private:
    MatchingEngine& matching_engine_;
    std::map<OrderID, Order> orders_;
    std::map<UserID, std::vector<OrderID>> user_orders_;
    std::atomic<OrderID> next_order_id_;
    std::mutex mutex_;
    
    // Validate order
    std::optional<std::string> validate_order(const Order& order);
};

// ============ Trading Engine (Main) ============

class TradingEngine {
public:
    TradingEngine();
    ~TradingEngine();
    
    // Initialize
    void initialize(const RiskLimits& limits);
    
    // Start engine
    void start();
    
    // Stop engine
    void stop();
    
    // Place order
    std::variant<OrderID, std::string> place_order(
        UserID user_id,
        Market market,
        Side side,
        OrderType order_type,
        Price price,
        Quantity quantity,
        const std::string& client_order_id = ""
    );
    
    // Cancel order
    bool cancel_order(OrderID order_id);
    
    // Close position
    std::variant<OrderID, std::string> close_position(
        UserID user_id,
        Market market,
        Quantity quantity
    );
    
    // Set stop loss / take profit
    bool set_risk_controls(
        UserID user_id,
        Market market,
        Price stop_loss,
        Price take_profit
    );
    
    // Get position
    std::optional<Position> get_position(UserID user_id, Market market);
    
    // Get account risk
    AccountRisk get_account_risk(UserID user_id);
    
    // Get market data
    MarketData get_market_data(Market market);
    
    // Get trades
    const std::vector<Trade>& get_trades() const;
    
    // Process market data update
    void on_market_data_update(Market market, const MarketData& data);
    
    // Liquidate positions
    std::vector<LiquidationResult> liquidate_positions();

private:
    std::unique_ptr<MatchingEngine> matching_engine_;
    std::unique_ptr<PositionManager> position_manager_;
    std::unique_ptr<AccountManager> account_manager_;
    std::unique_ptr<OrderManager> order_manager_;
    std::unique_ptr<LiquidationEngine> liquidation_engine_;
    std::unique_ptr<FundingManager> funding_manager_;
    
    RiskLimits limits_;
    std::atomic<bool> running_;
    std::thread processing_thread_;
    std::mutex mutex_;
    
    // Processing loop
    void processing_loop();
    
    // Update positions with market data
    void update_positions();
};

// ============ Utilities ============

namespace utils {
    // Price conversion
    Price price_from_double(double price);
    double price_to_double(Price price);
    
    // Quantity conversion
    Quantity quantity_from_double(double qty);
    double quantity_to_double(Quantity qty);
    
    // Decimal helpers
    Decimal decimal_from_double(double value, int decimals = 8);
    double decimal_to_double(Decimal value, int decimals = 8);
    
    // Timestamp helpers
    Timestamp get_current_timestamp();
    Timestamp timestamp_from_datetime(int year, int month, int day, int hour, int min, int sec);
}

} // namespace perpetual
} // namespace tiger

#endif // TIGERWALLET_PERPETUAL_ENGINE_H
