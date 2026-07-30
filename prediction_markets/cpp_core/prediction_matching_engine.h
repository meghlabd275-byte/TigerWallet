/**
 * TigerWallet Prediction Markets - High-Performance Matching Engine
 * Ultra-low latency C++ implementation for prediction market order matching
 * 
 * Features:
 * - Sub-microsecond order matching
 * - Deterministic ordering with timestamp priority
 * - Multi-outcome support
 * - Real-time market resolution
 * - Atomic trade execution
 */

#ifndef TIGER_PREDICTION_MATCHING_ENGINE_H
#define TIGER_PREDICTION_MATCHING_ENGINE_H

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

// Platform-specific optimizations
#if defined(__linux__)
    #include <sched.h>
    #define TIGER_CPU_PIN(cpu) sched_setaffinity(0, sizeof(cpu_set_t), &cpu)
#elif defined(__APPLE__)
    #define TIGER_CPU_PIN(cpu)
#else
    #define TIGER_CPU_PIN(cpu)
#endif

namespace tiger {
namespace prediction {

// Constants
constexpr size_t MAX_OUTCOMES_PER_MARKET = 256;
constexpr size_t MAX_ORDERS_PER_MARKET = 1000000;
constexpr uint64_t NANOSECONDS_PER_SECOND = 1000000000ULL;
constexpr uint64_t MAX_PRICE = 1000000;  // 0.000001 precision
constexpr uint64_t MIN_PRICE = 1;

// Order types
enum class OrderType : uint8_t {
    MARKET = 0,
    LIMIT = 1,
    STOP_LOSS = 2,
    TAKE_PROFIT = 3
};

enum class OrderSide : uint8_t {
    BUY = 0,
    SELL = 1
};

enum class MarketStatus : uint8_t {
    ACTIVE = 0,
    RESOLVING = 1,
    RESOLVED = 2,
    CANCELLED = 3,
    PAUSED = 4
};

enum class OutcomeType : uint8_t {
    BINARY = 0,
    CATEGORICAL = 1,
    SCALAR = 2
};

// Price precision: 6 decimal places
constexpr uint64_t PRICE_SCALE = 1000000ULL;
constexpr uint64_t AMOUNT_SCALE = 1000000000ULL;  // 9 decimal places

// Order structure - cache-line optimized (64 bytes)
struct alignas(64) Order {
    uint64_t order_id;
    uint64_t market_id;
    uint32_t outcome_id;
    uint32_t user_id;
    OrderType order_type;
    OrderSide side;
    uint64_t price;
    uint64_t amount;
    uint64_t filled_amount;
    uint64_t timestamp;
    uint64_t expires_at;
    uint64_t stop_price;  // For stop orders
    uint64_t reserved[4];  // Padding
    
    Order() : order_id(0), market_id(0), outcome_id(0), user_id(0),
              order_type(OrderType::LIMIT), side(OrderSide::BUY),
              price(0), amount(0), filled_amount(0), timestamp(0),
              expires_at(0), stop_price(0) {
        reserved[0] = reserved[1] = reserved[2] = reserved[3] = 0;
    }
};

// Trade execution result
struct Trade {
    uint64_t trade_id;
    uint64_t order_id;
    uint64_t market_id;
    uint32_t outcome_id;
    uint64_t maker_order_id;
    uint64_t taker_order_id;
    uint64_t price;
    uint64_t amount;
    uint64_t timestamp;
    uint64_t fees;
};

// Market structure
struct Market {
    uint64_t market_id;
    std::string question;
    std::string description;
    OutcomeType outcome_type;
    uint32_t num_outcomes;
    std::vector<std::string> outcome_names;
    std::vector<uint64_t> outcome_prices;
    std::vector<uint64_t> outcome_volumes;
    MarketStatus status;
    uint64_t resolution_time;
    uint32_t resolved_outcome;
    uint64_t created_at;
    uint64_t updated_at;
    uint64_t volume24h;
    uint64_t total_volume;
    bool featured;
    std::string category;
    std::string image_url;
    
    Market() : market_id(0), outcome_type(OutcomeType::BINARY),
               num_outcomes(2), status(MarketStatus::ACTIVE),
               resolution_time(0), resolved_outcome(0),
               created_at(0), updated_at(0), volume24h(0),
               total_volume(0), featured(false) {}
};

// Order book for a single outcome
class OrderBook {
public:
    static constexpr size_t PRICE_LEVELS = 10000;
    
private:
    struct PriceLevel {
        uint64_t price;
        uint64_t total_amount;
        uint64_t order_count;
        std::deque<uint64_t> orders;  // Order IDs
        
        PriceLevel() : price(0), total_amount(0), order_count(0) {}
    };
    
    std::array<PriceLevel, PRICE_LEVELS> bid_levels_;
    std::array<PriceLevel, PRICE_LEVELS> ask_levels_;
    std::unordered_map<uint64_t, Order> orders_;
    uint64_t best_bid_price_;
    uint64_t best_ask_price_;
    uint32_t outcome_id_;
    mutable std::shared_mutex mutex_;
    
    inline size_t price_to_index(uint64_t price) const {
        // Convert price to index (price from 0.000001 to 1.0)
        return (price * PRICE_LEVELS) / MAX_PRICE;
    }
    
public:
    OrderBook(uint32_t outcome_id);
    ~OrderBook() = default;
    
    // Non-copyable
    OrderBook(const OrderBook&) = delete;
    OrderBook& operator=(const OrderBook&) = delete;
    
    // Add order to book
    bool add_order(const Order& order);
    
    // Remove order from book
    bool remove_order(uint64_t order_id);
    
    // Match orders - returns vector of trades
    std::vector<Trade> match_orders(uint64_t timestamp);
    
    // Get best bid/ask
    std::pair<uint64_t, uint64_t> get_best_prices() const;
    
    // Get market depth
    std::vector<std::pair<uint64_t, uint64_t>> get_depth(uint32_t levels, bool bids) const;
    
    // Get order by ID
    std::optional<Order> get_order(uint64_t order_id) const;
    
    // Get total volume at price
    uint64_t get_volume_at_price(uint64_t price, bool bids) const;
    
    // Clear expired orders
    void clear_expired(uint64_t current_time);
    
    // Cancel all orders for user
    void cancel_user_orders(uint32_t user_id, std::vector<uint64_t>& cancelled);
    
    // Get statistics
    uint64_t get_total_bid_volume() const;
    uint64_t get_total_ask_volume() const;
    uint64_t get_order_count() const { return orders_.size(); }
};

// Market manager - handles all markets
class MarketManager {
private:
    std::unordered_map<uint64_t, Market> markets_;
    std::unordered_map<uint64_t, std::unique_ptr<OrderBook>> order_books_;
    std::unordered_map<std::string, uint64_t> market_id_by_question_;
    std::unordered_set<uint64_t> featured_markets_;
    std::deque<Market> recent_markets_;
    
    mutable std::shared_mutex mutex_;
    uint64_t next_market_id_;
    uint64_t next_order_id_;
    uint64_t next_trade_id_;
    
    // Market categories
    std::unordered_map<std::string, std::vector<uint64_t>> markets_by_category_;
    
public:
    MarketManager();
    ~MarketManager() = default;
    
    // Create new market
    uint64_t create_market(
        std::string question,
        OutcomeType outcome_type,
        std::vector<std::string> outcome_names,
        uint64_t resolution_time,
        std::string category = "General"
    );
    
    // Get market by ID
    std::optional<Market> get_market(uint64_t market_id) const;
    
    // Get market by question
    std::optional<Market> get_market_by_question(std::string_view question) const;
    
    // Update market prices
    void update_market_prices(uint64_t market_id, const std::vector<uint64_t>& prices);
    
    // Resolve market
    bool resolve_market(uint64_t market_id, uint32_t outcome);
    
    // Pause/resume market
    void pause_market(uint64_t market_id);
    void resume_market(uint64_t market_id);
    void cancel_market(uint64_t market_id);
    
    // Featured markets
    void set_featured(uint64_t market_id, bool featured);
    std::vector<uint64_t> get_featured_markets() const;
    
    // Get markets by category
    std::vector<uint64_t> get_markets_by_category(std::string_view category) const;
    
    // Get all categories
    std::vector<std::string> get_categories() const;
    
    // Get active markets with pagination
    std::vector<Market> get_markets(
        MarketStatus status,
        uint32_t offset,
        uint32_t limit,
        std::string_view category = ""
    ) const;
    
    // Get order book for market outcome
    OrderBook* get_order_book(uint64_t market_id, uint32_t outcome_id);
    
    // Get market statistics
    struct MarketStats {
        uint64_t total_markets;
        uint64_t active_markets;
        uint64_t total_volume_24h;
        uint64_t total_volume_all;
    };
    MarketStats get_stats() const;
};

// Order matching engine - main class
class PredictionMatchingEngine {
private:
    std::unique_ptr<MarketManager> market_manager_;
    
    // Thread pools for parallel processing
    std::vector<std::thread> worker_threads_;
    std::atomic<bool> running_;
    uint32_t num_threads_;
    
    // Order queue
    std::deque<Order> order_queue_;
    std::mutex queue_mutex_;
    std::condition_variable queue_cv_;
    
    // Statistics
    std::atomic<uint64_t> total_orders_processed_;
    std::atomic<uint64_t> total_trades_executed_;
    std::atomic<uint64_t> total_volume_;
    std::atomic<uint64_t> last_match_time_;
    
    // Performance monitoring
    struct LatencyStats {
        std::atomic<uint64_t> min_latency_ns;
        std::atomic<uint64_t> max_latency_ns;
        std::atomic<uint64_t> total_latency_ns;
        std::atomic<uint64_t> count;
        
        LatencyStats() : min_latency_ns(UINT64_MAX), max_latency_ns(0),
                        total_latency_ns(0), count(0) {}
    };
    
    LatencyStats order_latency_;
    LatencyStats match_latency_;
    
    // Worker thread function
    void worker_thread();
    
    // Process single order
    std::vector<Trade> process_order(const Order& order);
    
public:
    PredictionMatchingEngine(uint32_t num_threads = std::thread::hardware_concurrency());
    ~PredictionMatchingEngine();
    
    // Non-copyable
    PredictionMatchingEngine(const PredictionMatchingEngine&) = delete;
    PredictionMatchingEngine& operator=(const PredictionMatchingEngine&) = delete;
    
    // Start/stop engine
    void start();
    void stop();
    
    // Submit order
    uint64_t submit_order(const Order& order);
    
    // Cancel order
    bool cancel_order(uint64_t market_id, uint32_t outcome_id, uint64_t order_id, uint32_t user_id);
    
    // Cancel all orders for user
    uint32_t cancel_all_orders(uint32_t user_id);
    
    // Get market manager
    MarketManager* get_market_manager() { return market_manager_.get(); }
    
    // Get market by ID
    std::optional<Market> get_market(uint64_t market_id) const {
        return market_manager_->get_market(market_id);
    }
    
    // Create market
    uint64_t create_market(
        std::string question,
        OutcomeType outcome_type,
        std::vector<std::string> outcome_names,
        uint64_t resolution_time,
        std::string category = "General"
    ) {
        return market_manager_->create_market(
            std::move(question), outcome_type, std::move(outcome_names),
            resolution_time, std::move(category)
        );
    }
    
    // Resolve market
    bool resolve_market(uint64_t market_id, uint32_t outcome) {
        return market_manager_->resolve_market(market_id, outcome);
    }
    
    // Get order book
    OrderBook* get_order_book(uint64_t market_id, uint32_t outcome_id) {
        return market_manager_->get_order_book(market_id, outcome_id);
    }
    
    // Get performance statistics
    struct EngineStats {
        uint64_t total_orders;
        uint64_t total_trades;
        uint64_t total_volume;
        uint64_t last_match_ms;
        uint64_t avg_order_latency_us;
        uint64_t avg_match_latency_us;
        uint64_t min_order_latency_us;
        uint64_t max_order_latency_us;
        uint64_t min_match_latency_us;
        uint64_t max_match_latency_us;
        uint64_t queue_size;
    };
    
    EngineStats get_stats() const;
    
    // Get all markets
    std::vector<Market> get_markets(
        MarketStatus status = MarketStatus::ACTIVE,
        uint32_t offset = 0,
        uint32_t limit = 100,
        std::string_view category = ""
    ) const {
        return market_manager_->get_markets(status, offset, limit, category);
    }
    
    // Get featured markets
    std::vector<Market> get_featured_markets() const;
    
    // Get market depth
    std::vector<std::pair<uint64_t, uint64_t>> get_market_depth(
        uint64_t market_id,
        uint32_t outcome_id,
        uint32_t levels = 10,
        bool bids = true
    ) const;
};

// Inline implementations for performance
inline OrderBook::OrderBook(uint32_t outcome_id)
    : best_bid_price_(0), best_ask_price_(MAX_PRICE), outcome_id_(outcome_id) {
    // Initialize bid levels in descending order
    for (size_t i = 0; i < PRICE_LEVELS; ++i) {
        bid_levels_[i].price = MAX_PRICE - i;
        ask_levels_[i].price = 1 + i;
    }
}

inline bool OrderBook::add_order(const Order& order) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    auto price_idx = price_to_index(order.price);
    
    if (order.side == OrderSide::BUY) {
        auto& level = bid_levels_[price_idx];
        level.orders.push_back(order.order_id);
        level.total_amount += order.amount;
        level.order_count++;
        
        if (order.price > best_bid_price_) {
            best_bid_price_ = order.price;
        }
    } else {
        auto& level = ask_levels_[price_idx];
        level.orders.push_back(order.order_id);
        level.total_amount += order.amount;
        level.order_count++;
        
        if (order.price < best_ask_price_) {
            best_ask_price_ = order.price;
        }
    }
    
    orders_[order.order_id] = order;
    return true;
}

inline bool OrderBook::remove_order(uint64_t order_id) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    auto it = orders_.find(order_id);
    if (it == orders_.end()) return false;
    
    const auto& order = it->second;
    auto price_idx = price_to_index(order.price);
    
    if (order.side == OrderSide::BUY) {
        auto& level = bid_levels_[price_idx];
        level.total_amount -= order.amount - order.filled_amount;
        level.order_count--;
        
        // Find and remove order ID from deque
        auto& deque = level.orders;
        for (auto dit = deque.begin(); dit != deque.end(); ++dit) {
            if (*dit == order_id) {
                deque.erase(dit);
                break;
            }
        }
        
        // Update best bid
        if (level.total_amount == 0 && order.price == best_bid_price_) {
            // Find next best bid
            for (size_t i = price_idx; i < PRICE_LEVELS; --i) {
                if (bid_levels_[i].total_amount > 0) {
                    best_bid_price_ = bid_levels_[i].price;
                    break;
                }
            }
        }
    } else {
        auto& level = ask_levels_[price_idx];
        level.total_amount -= order.amount - order.filled_amount;
        level.order_count--;
        
        auto& deque = level.orders;
        for (auto dit = deque.begin(); dit != deque.end(); ++dit) {
            if (*dit == order_id) {
                deque.erase(dit);
                break;
            }
        }
        
        if (level.total_amount == 0 && order.price == best_ask_price_) {
            for (size_t i = price_idx; i < PRICE_LEVELS; ++i) {
                if (ask_levels_[i].total_amount > 0) {
                    best_ask_price_ = ask_levels_[i].price;
                    break;
                }
            }
        }
    }
    
    orders_.erase(it);
    return true;
}

inline std::pair<uint64_t, uint64_t> OrderBook::get_best_prices() const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    return {best_bid_price_, best_ask_price_};
}

inline std::vector<std::pair<uint64_t, uint64_t>> OrderBook::get_depth(
    uint32_t levels, bool bids
) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    std::vector<std::pair<uint64_t, uint64_t>> depth;
    depth.reserve(levels);
    
    if (bids) {
        size_t start_idx = price_to_index(best_bid_price_);
        for (size_t i = 0; i < levels && i < PRICE_LEVELS; ++i) {
            auto idx = start_idx + i;
            if (idx >= PRICE_LEVELS) break;
            if (bid_levels_[idx].total_amount > 0) {
                depth.emplace_back(bid_levels_[idx].price, bid_levels_[idx].total_amount);
            }
        }
    } else {
        size_t start_idx = price_to_index(best_ask_price_);
        for (size_t i = 0; i < levels && i < PRICE_LEVELS; ++i) {
            auto idx = start_idx + i;
            if (idx >= PRICE_LEVELS) break;
            if (ask_levels_[idx].total_amount > 0) {
                depth.emplace_back(ask_levels_[idx].price, ask_levels_[idx].total_amount);
            }
        }
    }
    
    return depth;
}

inline std::optional<Order> OrderBook::get_order(uint64_t order_id) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    auto it = orders_.find(order_id);
    if (it != orders_.end()) {
        return it->second;
    }
    return std::nullopt;
}

inline uint64_t OrderBook::get_volume_at_price(uint64_t price, bool bids) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    auto idx = price_to_index(price);
    if (idx >= PRICE_LEVELS) return 0;
    
    return bids ? bid_levels_[idx].total_amount : ask_levels_[idx].total_amount;
}

inline void OrderBook::clear_expired(uint64_t current_time) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    std::vector<uint64_t> to_remove;
    
    for (const auto& [order_id, order] : orders_) {
        if (order.expires_at > 0 && order.expires_at < current_time) {
            to_remove.push_back(order_id);
        }
    }
    
    for (auto order_id : to_remove) {
        remove_order(order_id);
    }
}

inline void OrderBook::cancel_user_orders(uint32_t user_id, std::vector<uint64_t>& cancelled) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    for (const auto& [order_id, order] : orders_) {
        if (order.user_id == user_id) {
            cancelled.push_back(order_id);
        }
    }
    
    lock.unlock();
    
    for (auto order_id : cancelled) {
        remove_order(order_id);
    }
}

inline uint64_t OrderBook::get_total_bid_volume() const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    uint64_t total = 0;
    for (const auto& level : bid_levels_) {
        total += level.total_amount;
    }
    return total;
}

inline uint64_t OrderBook::get_total_ask_volume() const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    uint64_t total = 0;
    for (const auto& level : ask_levels_) {
        total += level.total_amount;
    }
    return total;
}

// MarketManager inline implementations
inline MarketManager::MarketManager() 
    : next_market_id_(1), next_order_id_(1), next_trade_id_(1) {}

inline uint64_t MarketManager::create_market(
    std::string question,
    OutcomeType outcome_type,
    std::vector<std::string> outcome_names,
    uint64_t resolution_time,
    std::string category
) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    uint64_t market_id = next_market_id_++;
    
    Market market;
    market.market_id = market_id;
    market.question = std::move(question);
    market.outcome_type = outcome_type;
    market.outcome_names = std::move(outcome_names);
    market.num_outcomes = static_cast<uint32_t>(market.outcome_names.size());
    market.outcome_prices.resize(market.num_outcomes, MAX_PRICE / 2);  // Initialize at 0.5
    market.outcome_volumes.resize(market.num_outcomes, 0);
    market.status = MarketStatus::ACTIVE;
    market.resolution_time = resolution_time;
    market.created_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    market.updated_at = market.created_at;
    market.category = std::move(category);
    
    markets_[market_id] = market;
    market_id_by_question_[market.question] = market_id;
    
    // Create order books for each outcome
    for (uint32_t i = 0; i < market.num_outcomes; ++i) {
        order_books_[market_id * 1000 + i] = std::make_unique<OrderBook>(i);
    }
    
    // Add to category
    markets_by_category_[market.category].push_back(market_id);
    
    // Add to recent markets
    recent_markets_.push_front(market);
    if (recent_markets_.size() > 100) {
        recent_markets_.pop_back();
    }
    
    return market_id;
}

inline std::optional<Market> MarketManager::get_market(uint64_t market_id) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    auto it = markets_.find(market_id);
    if (it != markets_.end()) {
        return it->second;
    }
    return std::nullopt;
}

inline std::optional<Market> MarketManager::get_market_by_question(std::string_view question) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    auto it = market_id_by_question_.find(std::string(question));
    if (it != market_id_by_question_.end()) {
        return get_market(it->second);
    }
    return std::nullopt;
}

inline void MarketManager::update_market_prices(
    uint64_t market_id,
    const std::vector<uint64_t>& prices
) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    auto it = markets_.find(market_id);
    if (it == markets_.end()) return;
    
    auto& market = it->second;
    if (prices.size() != market.outcome_prices.size()) return;
    
    market.outcome_prices = prices;
    market.updated_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
}

inline bool MarketManager::resolve_market(uint64_t market_id, uint32_t outcome) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    auto it = markets_.find(market_id);
    if (it == markets_.end()) return false;
    
    auto& market = it->second;
    if (market.status != MarketStatus::ACTIVE) return false;
    if (outcome >= market.num_outcomes) return false;
    
    market.status = MarketStatus::RESOLVED;
    market.resolved_outcome = outcome;
    market.updated_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    return true;
}

inline void MarketManager::pause_market(uint64_t market_id) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    auto it = markets_.find(market_id);
    if (it != markets_.end()) {
        it->second.status = MarketStatus::PAUSED;
    }
}

inline void MarketManager::resume_market(uint64_t market_id) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    auto it = markets_.find(market_id);
    if (it != markets_.end()) {
        it->second.status = MarketStatus::ACTIVE;
    }
}

inline void MarketManager::cancel_market(uint64_t market_id) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    auto it = markets_.find(market_id);
    if (it != markets_.end()) {
        it->second.status = MarketStatus::CANCELLED;
    }
}

inline void MarketManager::set_featured(uint64_t market_id, bool featured) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    if (featured) {
        featured_markets_.insert(market_id);
    } else {
        featured_markets_.erase(market_id);
    }
    
    auto it = markets_.find(market_id);
    if (it != markets_.end()) {
        it->second.featured = featured;
    }
}

inline std::vector<uint64_t> MarketManager::get_featured_markets() const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    return std::vector<uint64_t>(featured_markets_.begin(), featured_markets_.end());
}

inline std::vector<uint64_t> MarketManager::get_markets_by_category(
    std::string_view category
) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    auto it = markets_by_category_.find(std::string(category));
    if (it != markets_by_category_.end()) {
        return it->second;
    }
    return {};
}

inline std::vector<std::string> MarketManager::get_categories() const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    std::vector<std::string> categories;
    categories.reserve(markets_by_category_.size());
    
    for (const auto& [category, _] : markets_by_category_) {
        categories.push_back(category);
    }
    
    return categories;
}

inline std::vector<Market> MarketManager::get_markets(
    MarketStatus status,
    uint32_t offset,
    uint32_t limit,
    std::string_view category
) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    std::vector<Market> result;
    result.reserve(limit);
    
    uint32_t count = 0;
    uint32_t skipped = 0;
    
    for (const auto& [_, market] : markets_) {
        if (market.status != status) continue;
        
        if (!category.empty() && market.category != category) continue;
        
        if (skipped < offset) {
            skipped++;
            continue;
        }
        
        if (count >= limit) break;
        
        result.push_back(market);
        count++;
    }
    
    return result;
}

inline OrderBook* MarketManager::get_order_book(uint64_t market_id, uint32_t outcome_id) {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    auto it = order_books_.find(market_id * 1000 + outcome_id);
    if (it != order_books_.end()) {
        return it->second.get();
    }
    return nullptr;
}

inline MarketManager::MarketStats MarketManager::get_stats() const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    MarketStats stats = {0, 0, 0, 0};
    
    for (const auto& [_, market] : markets_) {
        stats.total_markets++;
        if (market.status == MarketStatus::ACTIVE) {
            stats.active_markets++;
        }
        stats.total_volume_24h += market.volume24h;
        stats.total_volume_all += market.total_volume;
    }
    
    return stats;
}

} // namespace prediction
} // namespace tiger

#endif // TIGER_PREDICTION_MATCHING_ENGINE_H
