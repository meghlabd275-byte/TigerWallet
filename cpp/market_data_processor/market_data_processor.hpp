/**
 * TigerWallet High-Performance Market Data Processor
 * Ultra-low latency C++17 market data processing for trading
 * 
 * Features:
 * - Real-time price feed processing
 * - Order book aggregation
 * - Market statistics calculation
 * - WebSocket streaming
 */

#ifndef TIGER_WALLET_MARKET_DATA_PROCESSOR_HPP
#define TIGER_WALLET_MARKET_DATA_PROCESSOR_HPP

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <queue>
#include <shared_mutex>
#include <string>
#include <thread>
#include <unordered_map>
#include <vector>
#include <cmath>
#include <algorithm>
#include <sstream>
#include <iomanip>

// ============================================================================
// Configuration
// ============================================================================

struct MarketDataConfig {
    uint32_t max_order_book_levels = 100;
    uint32_t processing_thread_count = 4;
    uint64_t price_update_interval_us = 100; // microseconds
    uint64_t stats_aggregation_interval_ms = 1000;
    bool enable_real_time_stats = true;
    bool enable_historical_data = true;
    uint32_t max_historical_points = 10000;
};

// ============================================================================
// Types
// ============================================================================

using Price = double;
using Quantity = double;
using Timestamp = std::chrono::milliseconds;

// Price level in order book
struct PriceLevel {
    Price price;
    Quantity quantity;
    uint32_t order_count;
    
    bool operator<(const PriceLevel& other) const {
        return price < other.price;
    }
    
    bool operator>(const PriceLevel& other) const {
        return price > other.price;
    }
};

// Order book side
struct OrderBookSide {
    std::vector<PriceLevel> levels;
    
    void update(const Price& price, const Quantity& quantity) {
        auto it = std::find_if(levels.begin(), levels.end(),
            [&price](const PriceLevel& p) { return p.price == price; });
        
        if (quantity == 0) {
            if (it != levels.end()) {
                levels.erase(it);
            }
        } else {
            if (it != levels.end()) {
                it->quantity = quantity;
            } else {
                levels.push_back({price, quantity, 1});
            }
        }
        
        std::sort(levels.begin(), levels.end());
    }
    
    Price top_price(bool ascending = true) const {
        if (levels.empty()) return 0;
        return ascending ? levels.front().price : levels.back().price;
    }
    
    Quantity total_quantity() const {
        Quantity total = 0;
        for (const auto& level : levels) {
            total += level.quantity;
        }
        return total;
    }
};

// Order book
struct OrderBook {
    std::string symbol;
    OrderBookSide bids;
    OrderBookSide asks;
    Timestamp last_update;
    uint64_t sequence;
    
    Price best_bid() const { return bids.top_price(true); }
    Price best_ask() const { return asks.top_price(false); }
    Price mid_price() const { 
        if (bids.levels.empty() || asks.levels.empty()) return 0;
        return (best_bid() + best_ask()) / 2.0;
    }
    
    Quantity spread() const {
        if (bids.levels.empty() || asks.levels.empty()) return 0;
        return best_ask() - best_bid();
    }
    
    double spread_percentage() const {
        Price mid = mid_price();
        if (mid == 0) return 0;
        return (spread() / mid) * 100.0;
    }
};

// Trade
struct Trade {
    std::string id;
    std::string symbol;
    Price price;
    Quantity quantity;
    Timestamp timestamp;
    OrderSide side;
    std::string maker_order_id;
    std::string taker_order_id;
};

// OHLCV Candle
struct Candle {
    std::string symbol;
    Timestamp open_time;
    Timestamp close_time;
    Price open;
    Price high;
    Price low;
    Price close;
    Quantity volume;
    uint32_t trade_count;
    
    bool is_complete() const {
        return close_time.time_since_epoch().count() > 0;
    }
};

// Market statistics
struct MarketStats {
    std::string symbol;
    Price last_price;
    Price open_24h;
    Price high_24h;
    Price low_24h;
    Price close_24h;
    Quantity volume_24h;
    Quantity quote_volume_24h;
    Price vwap_24h;
    Price price_change_24h;
    double price_change_pct_24h;
    Quantity trades_24h;
    Timestamp timestamp;
};

// Ticker data
struct Ticker {
    std::string symbol;
    Price bid_price;
    Quantity bid_quantity;
    Price ask_price;
    Quantity ask_quantity;
    Price last_price;
    Quantity volume_24h;
    Quantity quote_volume_24h;
    Timestamp timestamp;
};

// ============================================================================
// Market Data Processor
// ============================================================================

class MarketDataProcessor {
public:
    using PriceUpdateCallback = std::function<void(const std::string&, Price)>;
    using TradeCallback = std::function<void(const Trade&)>;
    using StatsCallback = std::function<void(const MarketStats&)>;
    using OrderBookCallback = std::function<void(const OrderBook&)>;

private:
    MarketDataConfig config_;
    
    // Order books by symbol
    std::unordered_map<std::string, OrderBook> order_books_;
    mutable std::shared_mutex order_books_mutex_;
    
    // Market statistics
    std::unordered_map<std::string, MarketStats> stats_;
    mutable std::shared_mutex stats_mutex_;
    
    // Recent trades
    std::unordered_map<std::string, std::vector<Trade>> recent_trades_;
    mutable std::shared_mutex trades_mutex_;
    
    // Candles
    std::unordered_map<std::string, std::vector<Candle>> candles_;
    mutable std::shared_mutex candles_mutex_;
    
    // Callbacks
    std::vector<PriceUpdateCallback> price_callbacks_;
    std::vector<TradeCallback> trade_callbacks_;
    std::vector<StatsCallback> stats_callbacks_;
    std::vector<OrderBookCallback> orderbook_callbacks_;
    
    // Processing thread
    std::atomic<bool> running_;
    std::thread processing_thread_;
    
    // Sequence counter
    std::atomic<uint64_t> sequence_{0};
    
public:
    explicit MarketDataProcessor(const MarketDataConfig& config = MarketDataConfig())
        : config_(config), running_(false) {
        initialize_processing_thread();
    }
    
    ~MarketDataProcessor() {
        stop();
    }
    
    // =========================================================================
    // Order Book Management
    // =========================================================================
    
    void update_order_book(const std::string& symbol, 
                          const std::vector<std::pair<Price, Quantity>>& bids,
                          const std::vector<std::pair<Price, Quantity>>& asks) {
        std::unique_lock lock(order_books_mutex_);
        
        OrderBook& book = order_books_[symbol];
        book.symbol = symbol;
        book.last_update = std::chrono::duration_cast<Timestamp>(
            std::chrono::system_clock::now().time_since_epoch());
        book.sequence = sequence_++;
        
        // Update bids
        for (const auto& [price, quantity] : bids) {
            book.bids.update(price, quantity);
        }
        
        // Update asks
        for (const auto& [price, quantity] : asks) {
            book.asks.update(price, quantity);
        }
        
        // Limit order book size
        if (book.bids.levels.size() > config_.max_order_book_levels) {
            book.bids.levels.resize(config_.max_order_book_levels);
        }
        if (book.asks.levels.size() > config_.max_order_book_levels) {
            book.asks.levels.resize(config_.max_order_book_levels);
        }
        
        lock.unlock();
        
        // Notify callbacks
        for (auto& callback : orderbook_callbacks_) {
            callback(book);
        }
    }
    
    std::optional<OrderBook> get_order_book(const std::string& symbol) const {
        std::shared_lock lock(order_books_mutex_);
        auto it = order_books_.find(symbol);
        if (it != order_books_.end()) {
            return it->second;
        }
        return std::nullopt;
    }
    
    // =========================================================================
    // Trade Processing
    // =========================================================================
    
    void process_trade(const Trade& trade) {
        std::unique_lock lock(trades_mutex_);
        
        auto& trades = recent_trades_[trade.symbol];
        trades.push_back(trade);
        
        // Keep only recent trades
        if (trades.size() > 1000) {
            trades.erase(trades.begin(), trades.begin() + trades.size() - 1000);
        }
        
        lock.unlock();
        
        // Update statistics
        update_stats(trade);
        
        // Notify callbacks
        for (auto& callback : trade_callbacks_) {
            callback(trade);
        }
    }
    
    std::vector<Trade> get_recent_trades(const std::string& symbol, 
                                         uint32_t limit = 100) const {
        std::shared_lock lock(trades_mutex_);
        auto it = recent_trades_.find(symbol);
        if (it != recent_trades_.end()) {
            uint32_t count = std::min(limit, static_cast<uint32_t>(it->second.size()));
            return std::vector<Trade>(it->second.end() - count, it->second.end());
        }
        return {};
    }
    
    // =========================================================================
    // Market Statistics
    // =========================================================================
    
    void update_stats(const Trade& trade) {
        std::unique_lock lock(stats_mutex_);
        
        auto& stats = stats_[trade.symbol];
        stats.symbol = trade.symbol;
        stats.last_price = trade.price;
        stats.timestamp = trade.timestamp;
        
        // Update 24h stats
        auto now = std::chrono::duration_cast<Timestamp>(
            std::chrono::system_clock::now().time_since_epoch());
        
        if (stats.timestamp.time_since_epoch().count() == 0 ||
            now - stats.timestamp > std::chrono::hours(24)) {
            // Reset 24h stats
            stats.open_24h = trade.price;
            stats.high_24h = trade.price;
            stats.low_24h = trade.price;
            stats.volume_24h = 0;
            stats.quote_volume_24h = 0;
            stats.trades_24h = 0;
        }
        
        // Update high/low
        if (trade.price > stats.high_24h) stats.high_24h = trade.price;
        if (trade.price < stats.low_24h) stats.low_24h = trade.price;
        
        // Update volume
        stats.volume_24h += trade.quantity;
        stats.quote_volume_24h += trade.price * trade.quantity;
        stats.trades_24h++;
        
        // Calculate VWAP
        if (stats.volume_24h > 0) {
            stats.vwap_24h = stats.quote_volume_24h / stats.volume_24h;
        }
        
        // Calculate price change
        if (stats.open_24h > 0) {
            stats.price_change_24h = stats.last_price - stats.open_24h;
            stats.price_change_pct_24h = (stats.price_change_24h / stats.open_24h) * 100.0;
        }
        
        lock.unlock();
        
        // Notify callbacks
        for (auto& callback : stats_callbacks_) {
            callback(stats);
        }
    }
    
    std::optional<MarketStats> get_market_stats(const std::string& symbol) const {
        std::shared_lock lock(stats_mutex_);
        auto it = stats_.find(symbol);
        if (it != stats_.end()) {
            return it->second;
        }
        return std::nullopt;
    }
    
    // =========================================================================
    // Candle Management
    // =========================================================================
    
    void update_candle(const std::string& symbol, 
                      Timestamp timestamp,
                      Price price,
                      Quantity volume) {
        std::unique_lock lock(candles_mutex_);
        
        auto& candle_vec = candles_[symbol];
        
        // Find or create current candle
        if (candle_vec.empty() || 
            timestamp - candle_vec.back().open_time >= std::chrono::minutes(1)) {
            // Create new candle
            Candle candle;
            candle.symbol = symbol;
            candle.open_time = timestamp;
            candle.close_time = Timestamp::max();
            candle.open = price;
            candle.high = price;
            candle.low = price;
            candle.close = price;
            candle.volume = volume;
            candle.trade_count = 1;
            candle_vec.push_back(candle);
        } else {
            // Update existing candle
            auto& candle = candle_vec.back();
            candle.close = price;
            candle.high = std::max(candle.high, price);
            candle.low = std::min(candle.low, price);
            candle.volume += volume;
            candle.trade_count++;
        }
        
        // Keep only recent candles
        if (candle_vec.size() > config_.max_historical_points) {
            candle_vec.erase(candle_vec.begin(), 
                           candle_vec.begin() + candle_vec.size() - config_.max_historical_points);
        }
    }
    
    std::vector<Candle> get_candles(const std::string& symbol, 
                                    uint32_t limit = 100) const {
        std::shared_lock lock(candles_mutex_);
        auto it = candles_.find(symbol);
        if (it != candles_.end()) {
            uint32_t count = std::min(limit, static_cast<uint32_t>(it->second.size()));
            return std::vector<Candle>(it->second.end() - count, it->second.end());
        }
        return {};
    }
    
    // =========================================================================
    // Callbacks
    // =========================================================================
    
    void on_price_update(PriceUpdateCallback callback) {
        price_callbacks_.push_back(callback);
    }
    
    void on_trade(TradeCallback callback) {
        trade_callbacks_.push_back(callback);
    }
    
    void on_stats_update(StatsCallback callback) {
        stats_callbacks_.push_back(callback);
    }
    
    void on_order_book_update(OrderBookCallback callback) {
        orderbook_callbacks_.push_back(callback);
    }
    
    // =========================================================================
    // Processing
    // =========================================================================
    
    void start() {
        running_ = true;
        if (!processing_thread_.joinable()) {
            initialize_processing_thread();
        }
    }
    
    void stop() {
        running_ = false;
        if (processing_thread_.joinable()) {
            processing_thread_.join();
        }
    }

private:
    void initialize_processing_thread() {
        processing_thread_ = std::thread([this]() {
            while (running_) {
                // Process any pending data
                std::this_thread::sleep_for(
                    std::chrono::microseconds(config_.price_update_interval_us));
            }
        });
    }
};

// ============================================================================
// Order Side Enum
// ============================================================================

enum class OrderSide {
    BUY,
    SELL
};

// ============================================================================
// Factory
// ============================================================================

inline std::unique_ptr<MarketDataProcessor> create_market_data_processor(
    const MarketDataConfig& config = MarketDataConfig()) {
    return std::make_unique<MarketDataProcessor>(config);
}

#endif // TIGER_WALLET_MARKET_DATA_PROCESSOR_HPP
