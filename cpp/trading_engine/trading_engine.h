/**
 * TigerWallet High-Frequency Trading Engine
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - Sub-microsecond order execution
 * - Market making strategy
 * - Arbitrage detection
 * - Risk management
 * - Order matching engine
 */

#ifndef TIGER_TRADING_ENGINE_H
#define TIGER_TRADING_ENGINE_H

#include <atomic>
#include <chrono>
#include <deque>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <queue>
#include <string>
#include <unordered_map>
#include <vector>

namespace tiger {
namespace trading {

// Order types
enum class OrderType {
    MARKET,
    LIMIT,
    STOP_LOSS,
    TAKE_PROFIT,
    STOP_LIMIT
};

enum class OrderSide {
    BUY,
    SELL
};

enum class OrderStatus {
    PENDING,
    OPEN,
    PARTIALLY_FILLED,
    FILLED,
    CANCELLED,
    REJECTED
};

enum class TimeInForce {
    GTC,  // Good Till Cancel
    IOC,  // Immediate Or Cancel
    FOK,  // Fill Or Kill
    GTD   // Good Till Date
};

// Order structure
struct Order {
    std::string order_id;
    std::string user_id;
    std::string symbol;
    OrderType type;
    OrderSide side;
    double quantity;
    double price;
    double filled_quantity;
    double avg_fill_price;
    OrderStatus status;
    TimeInForce tif;
    uint64_t timestamp;
    uint64_t expire_time;
    std::string client_order_id;
};

// Trade execution
struct Trade {
    std::string trade_id;
    std::string order_id;
    std::string symbol;
    OrderSide side;
    double quantity;
    double price;
    double fee;
    uint64_t timestamp;
    std::string maker_order_id;
    std::string taker_order_id;
};

// Market data
struct MarketTicker {
    std::string symbol;
    double bid_price;
    double ask_price;
    double last_price;
    double volume_24h;
    double change_24h;
    uint64_t timestamp;
};

struct OrderBookLevel {
    double price;
    double quantity;
    int orders;
};

struct OrderBook {
    std::string symbol;
    std::vector<OrderBookLevel> bids;
    std::vector<OrderBookLevel> asks;
    double mid_price;
    double spread;
    uint64_t timestamp;
};

// Position
struct Position {
    std::string symbol;
    double quantity;
    double avg_entry_price;
    double unrealized_pnl;
    double realized_pnl;
    double margin_used;
};

// Risk metrics
struct RiskMetrics {
    double total_exposure;
    double max_exposure;
    double portfolio_value;
    double leverage;
    double margin_utilization;
    double var_95;  // Value at Risk 95%
};

// High-frequency order book
class OrderBookManager {
private:
    std::unordered_map<std::string, OrderBook> order_books_;
    std::mutex mutex_;
    std::atomic<uint64_t> last_update_{0};

public:
    void update_order_book(const std::string& symbol, const OrderBook& ob);
    std::optional<OrderBook> get_order_book(const std::string& symbol) const;
    double get_mid_price(const std::string& symbol) const;
    double get_spread(const std::string& symbol) const;
    bool is_stale(const std::string& symbol, uint64_t max_age_ms = 1000) const;
};

// Order matching engine
class MatchingEngine {
private:
    std::unordered_map<std::string, std::deque<Order>> bid_orders_;
    std::unordered_map<std::string, std::deque<Order>> ask_orders_;
    std::unordered_map<std::string, std::vector<Trade>> trades_;
    std::mutex mutex_;
    std::atomic<uint64_t> order_counter_{0};
    std::atomic<uint64_t> trade_counter_{0};

public:
    std::optional<Order> match_order(Order order);
    std::vector<Trade> get_trades(const std::string& symbol) const;
    std::vector<Order> get_open_orders(const std::string& symbol) const;
    bool cancel_order(const std::string& order_id);
    void clear();

private:
    std::optional<Order> match_market_order(Order& order);
    std::optional<Order> match_limit_order(Order& order);
    void execute_trade(Order& bid, Order& ask, double price, double quantity);
};

// Position manager
class PositionManager {
private:
    std::unordered_map<std::string, Position> positions_;
    std::mutex mutex_;

public:
    void update_position(const std::string& symbol, double quantity, double price);
    std::optional<Position> get_position(const std::string& symbol);
    std::vector<Position> get_all_positions() const;
    double get_total_unrealized_pnl() const;
    void close_position(const std::string& symbol);
    void close_all_positions();
};

// Risk manager
class RiskManager {
private:
    double max_position_size_;
    double max_leverage_;
    double max_daily_loss_;
    double margin_ratio_;
    std::atomic<double> daily_pnl_{0};

public:
    RiskManager(double max_leverage = 10.0, double max_position_size = 1000000.0);
    
    bool check_order_risk(const Order& order, const PositionManager& positions);
    bool check_margin_requirement(const PositionManager& positions);
    RiskMetrics calculate_risk_metrics(const PositionManager& positions, double portfolio_value);
    void record_trade(const Trade& trade);
    void reset_daily();
    
    double get_leverage() const { return max_leverage_; }
    double get_margin_ratio() const { return margin_ratio_; }
};

// Trading strategy base
class Strategy {
public:
    virtual ~Strategy() = default;
    virtual std::optional<Order> generate_signal(
        const MarketTicker& ticker,
        const OrderBook& order_book
    ) = 0;
    virtual std::string get_name() const = 0;
};

// Market making strategy
class MarketMakingStrategy : public Strategy {
private:
    double spread_bps_;  // Base spread in basis points
    double size_;
    double refresh_ms_;

public:
    MarketMakingStrategy(double spread_bps = 5.0, double size = 1000.0);
    
    std::optional<Order> generate_signal(
        const MarketTicker& ticker,
        const OrderBook& order_book
    ) override;
    
    std::string get_name() const override { return "MarketMaking"; }
};

// Arbitrage strategy
class ArbitrageStrategy : public Strategy {
private:
    double min_profit_bps_;
    std::vector<std::string> symbols_;

public:
    ArbitrageStrategy(double min_profit_bps = 2.0);
    
    std::optional<Order> generate_signal(
        const MarketTicker& ticker,
        const OrderBook& order_book
    ) override;
    
    std::string get_name() const override { return "Arbitrage"; }
};

// Main trading engine
class TradingEngine {
private:
    std::unique_ptr<OrderBookManager> order_book_manager_;
    std::unique_ptr<MatchingEngine> matching_engine_;
    std::unique_ptr<PositionManager> position_manager_;
    std::unique_ptr<RiskManager> risk_manager_;
    
    std::vector<std::unique_ptr<Strategy>> strategies_;
    std::atomic<bool> running_{false};
    
    // Performance metrics
    std::atomic<uint64_t> orders_processed_{0};
    std::atomic<uint64_t> orders_filled_{0};
    std::atomic<uint64_t> latency_sum_ns_{0};
    std::atomic<uint64_t> max_latency_ns_{0};

public:
    TradingEngine();
    ~TradingEngine();
    
    // Engine control
    void start();
    void stop();
    bool is_running() const { return running_.load(); }
    
    // Order operations
    std::optional<Order> submit_order(Order order);
    bool cancel_order(const std::string& order_id);
    std::vector<Order> get_open_orders() const;
    std::vector<Trade> get_trades(const std::string& symbol) const;
    
    // Market data
    void update_market_data(const MarketTicker& ticker);
    void update_order_book(const OrderBook& order_book);
    
    // Position & risk
    std::vector<Position> get_positions() const;
    RiskMetrics get_risk_metrics(double portfolio_value) const;
    
    // Strategies
    void add_strategy(std::unique_ptr<Strategy> strategy);
    void remove_strategy(const std::string& name);
    
    // Performance metrics
    double get_avg_latency_ns() const;
    uint64_t get_max_latency_ns() const;
    double get_fill_rate() const;
    uint64_t get_orders_processed() const { return orders_processed_.load(); }

private:
    void process_strategies();
    void cleanup_stale_orders();
};

// Inline implementations for performance

inline std::optional<Order> MatchingEngine::match_order(Order order) {
    auto start = std::chrono::high_resolution_clock::now();
    
    if (order.type == OrderType::MARKET) {
        auto result = match_market_order(order);
        auto end = std::chrono::high_resolution_clock::now();
        auto latency = std::chrono::duration_cast<std::chrono::nanoseconds>(end - start).count();
        return result;
    }
    
    auto result = match_limit_order(order);
    return result;
}

inline bool RiskManager::check_order_risk(const Order& order, const PositionManager& positions) {
    auto pos = positions.get_position(order.symbol);
    double current_exposure = pos ? pos->quantity * pos->avg_entry_price : 0;
    double new_exposure = current_exposure + order.quantity * order.price;
    
    if (new_exposure > max_position_size_) {
        return false;
    }
    
    return true;
}

inline bool RiskManager::check_margin_requirement(const PositionManager& positions) {
    auto all_positions = positions.get_all_positions();
    double total_margin = 0;
    double total_value = 0;
    
    for (const auto& pos : all_positions) {
        total_margin += pos.margin_used;
        total_value += pos.quantity * pos.avg_entry_price;
    }
    
    double utilization = total_value > 0 ? total_margin / total_value : 0;
    return utilization < (1.0 / max_leverage_);
}

inline RiskMetrics RiskManager::calculate_risk_metrics(
    const PositionManager& positions, 
    double portfolio_value
) {
    auto all_positions = positions.get_all_positions();
    
    double total_exposure = 0;
    for (const auto& pos : all_positions) {
        total_exposure += pos.quantity * pos.avg_entry_price;
    }
    
    return RiskMetrics{
        .total_exposure = total_exposure,
        .max_exposure = max_position_size_,
        .portfolio_value = portfolio_value,
        .leverage = portfolio_value > 0 ? total_exposure / portfolio_value : 0,
        .margin_utilization = total_exposure > 0 ? (total_exposure / portfolio_value) / max_leverage_ : 0,
        .var_95 = portfolio_value * 0.02  // Simplified 2% VaR
    };
}

} // namespace trading
} // namespace tiger

#endif // TIGER_TRADING_ENGINE_H
