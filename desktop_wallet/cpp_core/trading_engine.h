/**
 * TigerWallet Desktop Trading Terminal - High-Performance C++ Core
 * Ultra-low latency trading engine for professional traders
 */

#ifndef TIGER_DESKTOP_TRADING_ENGINE_H
#define TIGER_DESKTOP_TRADING_ENGINE_H

#include <array>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <deque>
#include <memory>
#include <mutex>
#include <optional>
#include <shared_mutex>
#include <string>
#include <string_view>
#include <thread>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#if defined(__linux__)
    #include <sched.h>
    #define TIGER_CPU_PIN(cpu) sched_setaffinity(0, sizeof(cpu_set_t), &cpu)
#elif defined(_WIN32)
    #include <windows.h>
    #define TIGER_CPU_PIN(cpu)
#else
    #define TIGER_CPU_PIN(cpu)
#endif

namespace tiger {
namespace desktop {

// ============================================================================
// Constants
// ============================================================================

constexpr size_t MAX_ORDERS = 1000000;
constexpr size_t MAX_POSITIONS = 10000;
constexpr size_t MAX_MARKETS = 5000;
constexpr uint64_t NANOSECONDS_PER_MS = 1000000ULL;

// Order types
enum class OrderType : uint8_t {
    MARKET = 0,
    LIMIT = 1,
    STOP_MARKET = 2,
    STOP_LIMIT = 3,
    TRAILING_STOP = 4,
    OCO = 5,  // One Cancels Other
    OTO = 6,   // One Triggers Other
};

enum class OrderSide : uint8_t {
    BUY = 0,
    SELL = 1
};

enum class OrderStatus : uint8_t {
    PENDING = 0,
    OPEN = 1,
    PARTIALLY_FILLED = 2,
    FILLED = 3,
    CANCELLED = 4,
    REJECTED = 5,
    EXPIRED = 6,
};

enum class PositionSide : uint8_t {
    LONG = 0,
    SHORT = 1,
    BOTH = 2,
};

enum class MarketType : uint8_t {
    SPOT = 0,
    PERPETUAL = 1,
    FUTURE = 2,
    OPTION = 3,
};

// ============================================================================
// Price Precision
// ============================================================================

constexpr uint64_t PRICE_SCALE = 1000000000ULL;  // 9 decimals
constexpr uint64_t QTY_SCALE = 1000000000ULL;     // 9 decimals
constexpr uint64_t PERCENT_SCALE = 10000ULL;      // 4 decimals (basis points)

// ============================================================================
// Data Structures
// ============================================================================

struct Order {
    uint64_t order_id;
    uint64_t market_id;
    uint64_t client_order_id;
    uint32_t user_id;
    OrderType order_type;
    OrderSide side;
    PositionSide position_side;
    uint64_t price;
    uint64_t quantity;
    uint64_t filled_quantity;
    uint64_t avg_fill_price;
    uint64_t stop_price;
    uint64_t trailing_percent;
    uint64_t timestamp;
    uint64_t expires_at;
    uint64_t last_update_time;
    uint8_t time_in_force;  // 0=GTC, 1=IOC, 2=FOK, 3=GTX
    bool reduce_only;
    bool post_only;
    OrderStatus status;
    std::string reject_reason;
    
    Order() : order_id(0), market_id(0), client_order_id(0), user_id(0),
              order_type(OrderType::LIMIT), side(OrderSide::BUY),
              position_side(PositionSide::BOTH), price(0), quantity(0),
              filled_quantity(0), avg_fill_price(0), stop_price(0),
              trailing_percent(0), timestamp(0), expires_at(0),
              last_update_time(0), time_in_force(0), reduce_only(false),
              post_only(false), status(OrderStatus::PENDING) {}
};

struct Position {
    uint64_t position_id;
    uint64_t market_id;
    uint32_t user_id;
    PositionSide side;
    uint64_t quantity;
    uint64_t entry_price;
    uint64_t mark_price;
    uint64_t liquidation_price;
    uint64_t margin_ratio;
    int64_t unrealized_pnl;
    int64_t realized_pnl;
    uint64_t margin_used;
    uint64_t margin_balance;
    uint64_t leverage;
    uint64_t last_update_time;
    bool auto_add_margin;
    bool is_isolated;
    
    Position() : position_id(0), market_id(0), user_id(0), side(PositionSide::BOTH),
                 quantity(0), entry_price(0), mark_price(0), liquidation_price(0),
                 margin_ratio(0), unrealized_pnl(0), realized_pnl(0),
                 margin_used(0), margin_balance(0), leverage(100),
                 last_update_time(0), auto_add_margin(false), is_isolated(false) {}
};

struct Market {
    uint64_t market_id;
    std::string symbol;
    std::string base_asset;
    std::string quote_asset;
    MarketType market_type;
    uint64_t price_precision;
    uint64_t quantity_precision;
    uint64_t min_quantity;
    uint64_t max_quantity;
    uint64_t tick_size;
    uint64_t step_size;
    uint64_t maker_fee;
    uint64_t taker_fee;
    uint64_t settlement_fee;
    uint64_t funding_rate;
    uint64_t next_funding_time;
    uint64_t index_price;
    uint64_t mark_price;
    uint64_t last_price;
    uint64_t last_price_change;
    uint64_t volume24h;
    uint64_t quote_volume24h;
    uint64_t open_interest;
    bool is_trading;
    bool is_margin_enabled;
    uint64_t max_leverage;
    uint64_t implied_leverage;
    
    Market() : market_id(0), market_type(MarketType::SPOT),
               price_precision(8), quantity_precision(8),
               min_quantity(1), max_quantity(UINT64_MAX),
               tick_size(1), step_size(1),
               maker_fee(1000), taker_fee(1500), settlement_fee(500),
               funding_rate(0), next_funding_time(0),
               index_price(0), mark_price(0), last_price(0),
               last_price_change(0), volume24h(0), quote_volume24h(0),
               open_interest(0), is_trading(true), is_margin_enabled(true),
               max_leverage(125), implied_leverage(100) {}
};

struct Trade {
    uint64_t trade_id;
    uint64_t order_id;
    uint64_t market_id;
    uint32_t user_id;
    OrderSide side;
    uint64_t price;
    uint64_t quantity;
    uint64_t fee;
    uint64_t realized_pnl;
    uint64_t timestamp;
    bool is_maker;
    std::string fee_asset;
    
    Trade() : trade_id(0), order_id(0), market_id(0), user_id(0),
              side(OrderSide::BUY), price(0), quantity(0), fee(0),
              realized_pnl(0), timestamp(0), is_maker(false) {}
};

struct Account {
    uint32_t user_id;
    std::string account_id;
    uint64_t total_equity;
    uint64_t total_margin;
    uint64_t available_margin;
    uint64_t total_unrealized_pnl;
    uint64_t total_realized_pnl;
    uint64_t total_fee;
    uint64_t total_position_value;
    uint64_t max_withdrawable;
    std::vector<Position> positions;
    uint64_t last_update_time;
    
    Account() : user_id(0), total_equity(0), total_margin(0),
                available_margin(0), total_unrealized_pnl(0),
                total_realized_pnl(0), total_fee(0),
                total_position_value(0), max_withdrawable(0),
                last_update_time(0) {}
};

struct OrderBook {
    uint64_t market_id;
    uint64_t last_update_id;
    
    struct PriceLevel {
        uint64_t price;
        uint64_t quantity;
        uint32_t orders;
        
        PriceLevel() : price(0), quantity(0), orders(0) {}
    };
    
    std::vector<PriceLevel> bids;
    std::vector<PriceLevel> asks;
    
    OrderBook() : market_id(0), last_update_id(0) {}
};

struct RiskLimit {
    uint64_t position_value;
    uint64_t margin_ratio;
    uint64_t max_leverage;
};

struct UserRiskStatus {
    uint32_t user_id;
    uint64_t total_exposure;
    uint64_t margin_ratio;
    bool is_liquidated;
    bool is_margin_called;
    uint64_t margin_call_level;
    uint64_t liquidation_level;
};

// ============================================================================
// Risk Engine
// ============================================================================

class RiskEngine {
private:
    std::vector<RiskLimit> risk_limits_;
    std::unordered_map<uint64_t, uint64_t> position_limits_;
    std::unordered_map<uint32_t, UserRiskStatus> user_risk_;
    mutable std::shared_mutex mutex_;
    
    uint64_t default_margin_ratio_;
    uint64_t liquidation_buffer_;
    uint64_t max_leverage_;
    
public:
    RiskEngine();
    ~RiskEngine() = default;
    
    // Risk checks
    bool check_order_risk(const Order& order, const Account& account, const Market& market);
    bool check_position_risk(const Position& position, const Market& market);
    bool check_margin_sufficient(const Account& account, uint64_t required_margin);
    bool check_max_leverage(uint64_t leverage, const Market& market);
    bool check_max_position_size(uint32_t user_id, uint64_t market_id, uint64_t quantity);
    
    // Position calculations
    uint64_t calculate_margin_required(const Order& order, const Market& market);
    uint64_t calculate_liquidation_price(const Position& position);
    uint64_t calculate_margin_ratio(const Position& position, const Market& market);
    int64_t calculate_unrealized_pnl(const Position& position, uint64_t current_price);
    
    // Risk updates
    void update_position_risk(const Position& position);
    void update_account_risk(Account& account, const std::vector<Position>& positions);
    void check_liquidation(Account& account, std::vector<Position>& positions, const Market& market);
    
    // Risk limits
    void set_risk_limits(const std::vector<RiskLimit>& limits);
    void set_position_limit(uint32_t user_id, uint64_t market_id, uint64_t limit);
    
    // Get risk status
    std::optional<UserRiskStatus> get_user_risk_status(uint32_t user_id);
};

// ============================================================================
// Matching Engine
// ============================================================================

class MatchingEngine {
private:
    std::unordered_map<uint64_t, std::unique_ptr<OrderBook>> order_books_;
    std::unordered_map<uint64_t, std::unordered_map<uint64_t, Order>> orders_;
    std::unordered_map<uint64_t, std::vector<Trade>> trades_;
    
    std::atomic<uint64_t> next_order_id_;
    std::atomic<uint64_t> next_trade_id_;
    
    mutable std::shared_mutex mutex_;
    
    // Market data
    std::unordered_map<uint64_t, Market> markets_;
    
public:
    MatchingEngine();
    ~MatchingEngine() = default;
    
    // Market management
    void add_market(const Market& market);
    std::optional<Market> get_market(uint64_t market_id);
    std::vector<Market> get_all_markets();
    void update_market_price(uint64_t market_id, uint64_t price);
    
    // Order operations
    uint64_t submit_order(const Order& order);
    bool cancel_order(uint64_t order_id);
    bool modify_order(uint64_t order_id, uint64_t new_price, uint64_t new_quantity);
    
    // Order retrieval
    std::optional<Order> get_order(uint64_t order_id);
    std::vector<Order> get_user_orders(uint32_t user_id, uint64_t market_id = 0);
    std::vector<Order> get_open_orders(uint32_t user_id);
    
    // Market data
    std::optional<OrderBook> get_order_book(uint64_t market_id);
    std::vector<Trade> get_market_trades(uint64_t market_id, uint32_t limit);
    std::vector<Trade> get_user_trades(uint32_t user_id, uint32_t limit);
    
    // Position management
    std::vector<Position> get_user_positions(uint32_t user_id);
    std::optional<Position> get_position(uint32_t user_id, uint64_t market_id);
    
    // Account
    Account get_account(uint32_t user_id);
    
    // Stats
    uint64_t get_market_volume(uint64_t market_id);
};

// ============================================================================
// Trading Engine
// ============================================================================

class DesktopTradingEngine {
private:
    std::unique_ptr<MatchingEngine> matching_engine_;
    std::unique_ptr<RiskEngine> risk_engine_;
    
    // Worker threads
    std::vector<std::thread> worker_threads_;
    std::atomic<bool> running_;
    uint32_t num_threads_;
    
    // Order queue
    std::deque<Order> order_queue_;
    std::mutex queue_mutex_;
    std::condition_variable queue_cv_;
    
    // Statistics
    struct EngineStats {
        std::atomic<uint64_t> total_orders;
        std::atomic<uint64_t> filled_orders;
        std::atomic<uint64_t> cancelled_orders;
        std::atomic<uint64_t> rejected_orders;
        std::atomic<uint64_t> total_volume;
        std::atomic<uint64_t> total_fees;
        std::atomic<uint64_t> avg_latency_us;
        std::atomic<uint64_t> min_latency_us;
        std::atomic<uint64_t> max_latency_us;
        
        EngineStats() : total_orders(0), filled_orders(0), cancelled_orders(0),
                       rejected_orders(0), total_volume(0), total_fees(0),
                       avg_latency_us(0), min_latency_us(UINT64_MAX), max_latency_us(0) {}
    } stats_;
    
    void worker_thread();
    bool process_order(const Order& order);
    
public:
    DesktopTradingEngine(uint32_t num_threads = std::thread::hardware_concurrency());
    ~DesktopTradingEngine();
    
    void start();
    void stop();
    
    // Order operations
    uint64_t place_order(const Order& order);
    bool cancel_order(uint64_t order_id);
    bool modify_order(uint64_t order_id, uint64_t new_price, uint64_t new_quantity);
    
    // Market operations
    void add_market(const Market& market);
    std::vector<Market> get_markets();
    std::optional<Market> get_market(uint64_t market_id);
    
    // Position and account
    std::vector<Position> get_positions(uint32_t user_id);
    Account get_account(uint32_t user_id);
    
    // Order and trade history
    std::vector<Order> get_orders(uint32_t user_id, uint64_t market_id = 0);
    std::vector<Trade> get_trades(uint32_t user_id, uint32_t limit = 100);
    
    // Statistics
    EngineStats get_stats() const;
    
    // Accessors
    MatchingEngine* matching_engine() { return matching_engine_.get(); }
    RiskEngine* risk_engine() { return risk_engine_.get(); }
};

// ============================================================================
// Inline Implementations
// ============================================================================

inline RiskEngine::RiskEngine()
    : default_margin_ratio_(1000),  // 10%
      liquidation_buffer_(500),       // 5%
      max_leverage_(125)            // 125x
{
    // Default risk limits
    risk_limits_ = {
        {10000000000ULL, 10000, 125},   // Up to 10M: 100% margin, 125x leverage
        {50000000000ULL, 2000, 100},   // Up to 50M: 20% margin, 100x
        {100000000000ULL, 2500, 50},   // Up to 100M: 25% margin, 50x
        {500000000000ULL, 5000, 20},   // Up to 500M: 50% margin, 20x
        {UINT64_MAX, 10000, 10},       // Above: 100% margin, 10x
    };
}

inline bool RiskEngine::check_order_risk(const Order& order, const Account& account, const Market& market) {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    // Check margin sufficiency
    uint64_t required_margin = calculate_margin_required(order, market);
    if (required_margin > account.available_margin) {
        return false;
    }
    
    // Check max leverage
    if (!check_max_leverage(order.quantity * order.price / QTY_SCALE, market)) {
        return false;
    }
    
    // Check position limit
    if (!check_max_position_size(order.user_id, order.market_id, order.quantity)) {
        return false;
    }
    
    return true;
}

inline uint64_t RiskEngine::calculate_margin_required(const Order& order, const Market& market) {
    uint64_t order_value = (order.quantity * order.price) / PRICE_SCALE;
    uint64_t margin_ratio = market.implied_leverage > 0 ? market.implied_leverage : max_leverage_;
    
    return (order_value * PERCENT_SCALE) / margin_ratio;
}

inline uint64_t RiskEngine::calculate_liquidation_price(const Position& position) {
    if (position.quantity == 0 || position.leverage == 0) return 0;
    
    uint64_t margin_ratio = PERCENT_SCALE / position.leverage;  // e.g., 10000/100 = 100 (1%)
    
    if (position.side == PositionSide::LONG) {
        // Liquidation price for long: entry * (1 - margin_ratio + buffer)
        return position.entry_price * (PERCENT_SCALE - margin_ratio + liquidation_buffer_) / PERCENT_SCALE;
    } else {
        // Liquidation price for short: entry * (1 + margin_ratio - buffer)
        return position.entry_price * (PERCENT_SCALE + margin_ratio - liquidation_buffer_) / PERCENT_SCALE;
    }
}

inline uint64_t RiskEngine::calculate_margin_ratio(const Position& position, const Market& market) {
    if (position.quantity == 0 || position.entry_price == 0) return PERCENT_SCALE;
    
    uint64_t position_value = (position.quantity * market.mark_price) / PRICE_SCALE;
    if (position_value == 0) return PERCENT_SCALE;
    
    return (position.margin_balance * PERCENT_SCALE * PERCENT_SCALE) / position_value;
}

inline int64_t RiskEngine::calculate_unrealized_pnl(const Position& position, uint64_t current_price) {
    if (position.quantity == 0) return 0;
    
    int64_t price_diff = 0;
    if (position.side == PositionSide::LONG) {
        price_diff = (int64_t)current_price - (int64_t)position.entry_price;
    } else {
        price_diff = (int64_t)position.entry_price - (int64_t)current_price;
    }
    
    return (int64_t)position.quantity * price_diff / PRICE_SCALE;
}

inline void RiskEngine::check_liquidation(Account& account, std::vector<Position>& positions, const Market& market) {
    for (auto& position : positions) {
        if (position.quantity == 0) continue;
        
        uint64_t margin_ratio = calculate_margin_ratio(position, market);
        
        // Update unrealized PnL
        position.unrealized_pnl = calculate_unrealized_pnl(position, market.mark_price);
        
        // Check liquidation
        uint64_t liquidation_price = calculate_liquidation_price(position);
        
        if (market.mark_price <= liquidation_price && position.side == PositionSide::LONG) {
            // Liquidate long position
            position.quantity = 0;
            account.available_margin -= position.margin_used;
        } else if (market.mark_price >= liquidation_price && position.side == PositionSide::SHORT) {
            // Liquidate short position
            position.quantity = 0;
            account.available_margin -= position.margin_used;
        }
    }
}

} // namespace desktop
} // namespace tiger

#endif // TIGER_DESKTOP_TRADING_ENGINE_H
