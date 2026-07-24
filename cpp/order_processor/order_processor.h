/**
 * TigerWallet High-Frequency Order Processor
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - Sub-microsecond order processing
 * - Order prioritization
 * - Market data aggregation
 * - Risk checks
 * - Trade execution
 */

#ifndef TIGER_ORDER_PROCESSOR_H
#define TIGER_ORDER_PROCESSOR_H

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <queue>
#include <string>
#include <unordered_map>
#include <vector>

namespace tiger {
namespace order {

// Order types
enum class OrderType { MARKET, LIMIT, STOP_LOSS, TAKE_PROFIT, STOP_LIMIT };
enum class OrderSide { BUY, SELL };
enum class OrderStatus { PENDING, OPEN, PARTIALLY_FILLED, FILLED, CANCELLED, REJECTED };

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
    uint64_t timestamp;
    uint64_t priority;
    std::string client_order_id;
};

struct Trade {
    std::string trade_id;
    std::string order_id;
    std::string symbol;
    OrderSide side;
    double quantity;
    double price;
    double fee;
    uint64_t timestamp;
};

struct PriceLevel {
    double price;
    double quantity;
    int orders;
};

struct MarketData {
    std::string symbol;
    double bid;
    double ask;
    double last;
    double volume_24h;
    double change_24h;
    std::vector<PriceLevel> bids;
    std::vector<PriceLevel> asks;
    uint64_t timestamp;
};

// Priority queue comparator
struct OrderComparator {
    bool operator()(const Order* a, const Order* b) const {
        // Higher priority first
        if (a->priority != b->priority) {
            return a->priority < b->priority;
        }
        // Earlier timestamp first
        return a->timestamp > b->timestamp;
    }
};

class OrderProcessor {
private:
    // Order queues by priority
    std::priority_queue<Order*, std::vector<Order*>, OrderComparator> order_queue_;
    
    // Order storage
    std::unordered_map<std::string, std::unique_ptr<Order>> orders_;
    
    // Market data cache
    std::unordered_map<std::string, MarketData> market_data_;
    
    // Trade history
    std::vector<Trade> trades_;
    
    // Counters
    std::atomic<uint64_t> orders_processed_{0};
    std::atomic<uint64_t> orders_filled_{0};
    std::atomic<uint64_t> total_latency_ns_{0};
    
    std::mutex mutex_;
    std::mutex market_mutex_;

public:
    OrderProcessor();
    ~OrderProcessor() = default;

    // Order operations
    std::string submit_order(Order order);
    bool cancel_order(const std::string& order_id);
    bool modify_order(const std::string& order_id, double new_quantity, double new_price);
    std::optional<Order> get_order(const std::string& order_id);
    std::vector<Order> get_open_orders(const std::string& user_id);
    
    // Market data
    void update_market_data(const MarketData& data);
    std::optional<MarketData> get_market_data(const std::string& symbol);
    
    // Processing
    void process_orders();
    std::optional<Trade> match_order(Order& order);
    
    // Statistics
    uint64_t get_orders_processed() const { return orders_processed_.load(); }
    uint64_t get_orders_filled() const { return orders_filled_.load(); }
    double get_avg_latency_us() const;
    double get_fill_rate() const;

private:
    bool validate_order(const Order& order);
    bool check_risk(const Order& order);
    std::optional<Trade> execute_market_order(Order& order);
    std::optional<Trade> execute_limit_order(Order& order);
    void update_order_status(Order& order, OrderStatus status);
    double calculate_fee(const Trade& trade);
};

// Inline implementations

inline OrderProcessor::OrderProcessor() {
    // Initialize with some market data
    std::vector<std::string> symbols = {"ETH/USDT", "BTC/USDT", "BNB/USDT"};
    for (const auto& symbol : symbols) {
        market_data_[symbol] = MarketData{
            .symbol = symbol,
            .bid = 3500.0,
            .ask = 3501.0,
            .last = 3500.5,
            .volume_24h = 1000000.0,
            .change_24h = 2.5,
            .timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
                std::chrono::system_clock::now().time_since_epoch()
            ).count()
        };
    }
}

inline std::string OrderProcessor::submit_order(Order order) {
    auto start = std::chrono::high_resolution_clock::now();
    
    // Validate order
    if (!validate_order(order)) {
        order.status = OrderStatus::REJECTED;
        orders_[order.order_id] = std::make_unique<Order>(order);
        return order.order_id;
    }
    
    // Check risk
    if (!check_risk(order)) {
        order.status = OrderStatus::REJECTED;
        orders_[order.order_id] = std::make_unique<Order>(order);
        return order.order_id;
    }
    
    // Set initial status
    order.status = OrderStatus::OPEN;
    order.filled_quantity = 0;
    order.avg_fill_price = 0;
    
    // Calculate priority (higher = more urgent)
    order.priority = order.type == OrderType::MARKET ? 1000 : 500;
    
    // Store order
    orders_[order.order_id] = std::make_unique<Order>(order);
    
    // Add to queue
    order_queue_.push(orders_[order.order_id].get());
    
    orders_processed_++;
    
    auto end = std::chrono::high_resolution_clock::now();
    auto latency = std::chrono::duration_cast<std::chrono::nanoseconds>(end - start).count();
    total_latency_ns_ += latency;
    
    return order.order_id;
}

inline bool OrderProcessor::validate_order(const Order& order) {
    if (order.quantity <= 0 || order.price <= 0) {
        return false;
    }
    
    if (order.symbol.empty() || order.user_id.empty()) {
        return false;
    }
    
    return true;
}

inline bool OrderProcessor::check_risk(const Order& order) {
    // Simple risk check - would integrate with risk manager
    // Check position limits, exposure, etc.
    return true;
}

inline void OrderProcessor::process_orders() {
    std::vector<Order*> to_process;
    
    // Get orders to process
    while (!order_queue_.empty()) {
        auto* order = order_queue_.top();
        
        if (order->status != OrderStatus::OPEN) {
            order_queue_.pop();
            continue;
        }
        
        // Try to match
        auto trade = match_order(*order);
        if (!trade) {
            break; // Can't fill immediately
        }
        
        order_queue_.pop();
        
        // Store trade
        trades_.push_back(*trade);
        orders_filled_++;
    }
}

inline std::optional<Trade> OrderProcessor::match_order(Order& order) {
    if (order.type == OrderType::MARKET) {
        return execute_market_order(order);
    }
    return execute_limit_order(order);
}

inline std::optional<Trade> OrderProcessor::execute_market_order(Order& order) {
    auto market_it = market_data_.find(order.symbol);
    if (market_it == market_data_.end()) {
        return std::nullopt;
    }
    
    const auto& market = market_it->second;
    double fill_price = order.side == OrderSide::BUY ? market.ask : market.bid;
    
    Trade trade;
    trade.trade_id = "trade_" + std::to_string(trades_.size() + 1);
    trade.order_id = order.order_id;
    trade.symbol = order.symbol;
    trade.side = order.side;
    trade.quantity = order.quantity;
    trade.price = fill_price;
    trade.fee = calculate_fee(trade);
    trade.timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    // Update order
    order.filled_quantity = order.quantity;
    order.avg_fill_price = fill_price;
    order.status = OrderStatus::FILLED;
    
    return trade;
}

inline std::optional<Trade> OrderProcessor::execute_limit_order(Order& order) {
    auto market_it = market_data_.find(order.symbol);
    if (market_it == market_data_.end()) {
        return std::nullopt;
    }
    
    const auto& market = market_it->second;
    
    bool can_fill = (order.side == OrderSide::BUY && order.price >= market.ask) ||
                   (order.side == OrderSide::SELL && order.price <= market.bid);
    
    if (!can_fill) {
        return std::nullopt; // Limit order waiting
    }
    
    double fill_price = order.price;
    
    Trade trade;
    trade.trade_id = "trade_" + std::to_string(trades_.size() + 1);
    trade.order_id = order.order_id;
    trade.symbol = order.symbol;
    trade.side = order.side;
    trade.quantity = order.quantity;
    trade.price = fill_price;
    trade.fee = calculate_fee(trade);
    trade.timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    order.filled_quantity = order.quantity;
    order.avg_fill_price = fill_price;
    order.status = OrderStatus::FILLED;
    
    return trade;
}

inline double OrderProcessor::calculate_fee(const Trade& trade) {
    // 0.1% fee
    return trade.quantity * trade.price * 0.001;
}

inline void OrderProcessor::update_market_data(const MarketData& data) {
    std::lock_guard<std::mutex> lock(market_mutex_);
    market_data_[data.symbol] = data;
}

inline std::optional<MarketData> OrderProcessor::get_market_data(const std::string& symbol) {
    std::lock_guard<std::mutex> lock(market_mutex_);
    auto it = market_data_.find(symbol);
    if (it != market_data_.end()) {
        return it->second;
    }
    return std::nullopt;
}

inline double OrderProcessor::get_avg_latency_us() const {
    auto total = total_latency_ns_.load();
    auto processed = orders_processed_.load();
    if (processed == 0) return 0;
    return (double)total / (double)processed / 1000.0;
}

inline double OrderProcessor::get_fill_rate() const {
    auto filled = orders_filled_.load();
    auto processed = orders_processed_.load();
    if (processed == 0) return 0;
    return (double)filled / (double)processed * 100.0;
}

} // namespace order
} // namespace tiger

#endif // TIGER_ORDER_PROCESSOR_H
