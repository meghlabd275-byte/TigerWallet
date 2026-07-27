/**
 * TigerWallet High-Frequency Market Data Processor
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - Real-time market data ingestion
 * - Order book management
 * - Price aggregation
 * - Volume analysis
 * - VWAP calculation
 * - Market depth analysis
 * - Statistical indicators
 * - WebSocket streaming
 */

#ifndef TIGER_MARKET_DATA_PROCESSOR_H
#define TIGER_MARKET_DATA_PROCESSOR_H

#include <atomic>
#include <chrono>
#include <deque>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <queue>
#include <shared_mutex>
#include <string>
#include <unordered_map>
#include <vector>
#include <sstream>
#include <iomanip>

namespace tiger {
namespace market_data {

// ============================================================================
// Configuration
// ============================================================================

struct MarketDataConfig {
    // Processing
    int max_order_book_depth = 100;
    int price_precision = 8;
    int volume_precision = 8;
    
    // Aggregation
    int aggregation_window_ms = 1000;
    int vwap_window_seconds = 300;
    
    // Buffering
    int max_tick_buffer = 10000;
    int max_trade_buffer = 10000;
    int max_order_book_history = 1000;
    
    // Performance
    bool enable_lock_free = true;
    int worker_threads = 4;
    
    // Data retention
    int tick_retention_minutes = 60;
    int trade_retention_minutes = 1440;
};

// ============================================================================
// Data Types
// ============================================================================

enum class MarketDataType {
    TRADE,
    BID,
    ASK,
    BIDS_ASK,
    TICKER,
    DEPTH,
    Candle
};

enum class Side {
    BUY,
    SELL,
    UNKNOWN
};

enum class OrderStatus {
    PENDING,
    OPEN,
    PARTIALLY_FILLED,
    FILLED,
    CANCELLED
};

// Price level in order book
struct PriceLevel {
    double price;
    double quantity;
    int order_count;
    uint64_t last_update;
};

// Order book entry
struct OrderBookEntry {
    std::string order_id;
    std::string symbol;
    Side side;
    double price;
    double quantity;
    double filled_quantity;
    OrderStatus status;
    uint64_t timestamp;
    uint64_t update_time;
};

// Trade
struct Trade {
    std::string trade_id;
    std::string symbol;
    Side side;
    double price;
    double quantity;
    double fee;
    bool is_buyer_maker;
    uint64_t timestamp;
    uint64_t trade_time;
};

// Tick data
struct Tick {
    std::string symbol;
    double bid_price;
    double bid_quantity;
    double ask_price;
    double ask_quantity;
    double last_price;
    double volume_24h;
    double quote_volume_24h;
    double price_change_24h;
    double price_change_percent_24h;
    double high_24h;
    double low_24h;
    uint64_t timestamp;
};

// Candlestick
struct Candle {
    std::string symbol;
    uint64_t timestamp;
    double open;
    double high;
    double low;
    double close;
    double volume;
    double quote_volume;
    int trades;
};

// Order book
struct OrderBook {
    std::string symbol;
    uint64_t last_update_id;
    std::vector<PriceLevel> bids;
    std::vector<PriceLevel> asks;
    uint64_t timestamp;
    bool is_consistent;
};

// ============================================================================
// Aggregated Data
// ============================================================================

struct AggregatedPrice {
    double vwap = 0.0;
    double twap = 0.0;
    double volume_weighted_bid = 0.0;
    double volume_weighted_ask = 0.0;
    double spread = 0.0;
    double mid_price = 0.0;
    double best_bid = 0.0;
    double best_ask = 0.0;
    double total_bid_volume = 0.0;
    double total_ask_volume = 0.0;
    uint64_t timestamp = 0;
};

struct MarketStatistics {
    double open = 0.0;
    double high = 0.0;
    double low = 0.0;
    double close = 0.0;
    double volume = 0.0;
    double quote_volume = 0.0;
    int trade_count = 0;
    double vwap = 0.0;
    double avg_trade_size = 0.0;
    double max_slippage = 0.0;
    double volatility = 0.0;
    uint64_t start_time = 0;
    uint64_t end_time = 0;
};

// ============================================================================
// Statistical Indicators
// ============================================================================

struct BollingerBands {
    double upper = 0.0;
    double middle = 0.0;
    double lower = 0.0;
};

struct RSI {
    double value = 0.0;
    bool is_oversold = false;
    bool is_overbought = false;
};

struct MACD {
    double macd_line = 0.0;
    double signal_line = 0.0;
    double histogram = 0.0;
};

struct MovingAverage {
    double sma = 0.0;
    double ema = 0.0;
    double wma = 0.0;
};

// ============================================================================
// Callbacks
// ============================================================================

using TickCallback = std::function<void(const Tick&)>;
using TradeCallback = std::function<void(const Trade&)>;
using OrderBookCallback = std::function<void(const OrderBook&)>;
using CandleCallback = std::function<void(const Candle&)>;
using StatisticsCallback = std::function<void(const MarketStatistics&)>;

// ============================================================================
// Market Data Processor Core
// ============================================================================

class MarketDataProcessor {
public:
    explicit MarketDataProcessor(const MarketDataConfig& config);
    ~MarketDataProcessor() = default;
    
    // Disable copying
    MarketDataProcessor(const MarketDataProcessor&) = delete;
    MarketDataProcessor& operator=(const MarketDataProcessor&) = delete;
    
    // Configuration
    void update_config(const MarketDataConfig& config);
    MarketDataConfig get_config() const;
    
    // Symbol management
    void register_symbol(const std::string& symbol);
    void unregister_symbol(const std::string& symbol);
    bool is_symbol_registered(const std::string& symbol) const;
    
    // Data ingestion
    void process_tick(const Tick& tick);
    void process_trade(const Trade& trade);
    void process_order_book(const OrderBook& order_book);
    void process_candle(const Candle& candle);
    
    // Batch processing
    void process_tick_batch(const std::vector<Tick>& ticks);
    void process_trade_batch(const std::vector<Trade>& trades);
    
    // Order book access
    std::optional<OrderBook> get_order_book(const std::string& symbol) const;
    std::vector<OrderBook> get_all_order_books() const;
    
    // Aggregated data
    AggregatedPrice get_aggregated_price(const std::string& symbol) const;
    
    // Statistics
    MarketStatistics get_market_statistics(
        const std::string& symbol,
        uint64_t start_time,
        uint64_t end_time
    );
    
    // Historical data
    std::vector<Tick> get_ticks(
        const std::string& symbol,
        uint64_t start_time,
        uint64_t end_time,
        int limit = 1000
    );
    
    std::vector<Trade> get_trades(
        const std::string& symbol,
        uint64_t start_time,
        uint64_t end_time,
        int limit = 1000
    );
    
    std::vector<Candle> get_candles(
        const std::string& symbol,
        uint64_t start_time,
        uint64_t end_time,
        const std::string& interval = "1m"
    );
    
    // Callbacks
    void set_tick_callback(TickCallback callback);
    void set_trade_callback(TradeCallback callback);
    void set_order_book_callback(OrderBookCallback callback);
    void set_candle_callback(CandleCallback callback);
    void set_statistics_callback(StatisticsCallback callback);
    
    // Indicators
    BollingerBands calculate_bollinger_bands(
        const std::string& symbol,
        int period = 20,
        double std_dev = 2.0
    );
    
    RSI calculate_rsi(const std::string& symbol, int period = 14);
    MACD calculate_macd(const std::string& symbol, int fast = 12, int slow = 26, int signal = 9);
    MovingAverage calculate_moving_averages(const std::string& symbol, int period = 20);
    
    // Market analysis
    double calculate_vwap(const std::string& symbol, int window_seconds = 300);
    double calculate_slippage(const std::string& symbol, double order_size);
    double calculate_market_impact(const std::string& symbol, double order_size);
    double calculate_liquidity(const std::string& symbol, double price_range_percent = 1.0);
    
    // Statistics
    size_t get_tick_count(const std::string& symbol) const;
    size_t get_trade_count(const std::string& symbol) const;
    size_t get_order_book_count() const;
    
    // Cleanup
    void clear_symbol_data(const std::string& symbol);
    void clear_all_data();
    
    // Performance metrics
    struct PerformanceMetrics {
        uint64_t total_ticks_processed = 0;
        uint64_t total_trades_processed = 0;
        uint64_t total_order_books_processed = 0;
        double avg_tick_processing_time_us = 0.0;
        double avg_trade_processing_time_us = 0.0;
        double avg_order_book_processing_time_us = 0.0;
        uint64_t last_processing_timestamp = 0;
    };
    
    PerformanceMetrics get_performance_metrics() const;
    void reset_performance_metrics();

private:
    // Configuration
    MarketDataConfig config_;
    
    // Symbol data
    struct SymbolData {
        // Order book
        std::map<double, PriceLevel, std::greater<double>> bid_levels;
        std::map<double, PriceLevel> ask_levels;
        OrderBook current_order_book;
        
        // Ticks
        std::deque<Tick> tick_buffer;
        std::deque<Tick> tick_history;
        
        // Trades
        std::deque<Trade> trade_buffer;
        std::deque<Trade> trade_history;
        
        // Candles
        std::map<uint64_t, Candle> candles;
        
        // Aggregated data
        AggregatedPrice aggregated_price;
        
        // Statistics
        MarketStatistics statistics;
        
        // Performance
        uint64_t ticks_processed = 0;
        uint64_t trades_processed = 0;
        uint64_t order_books_processed = 0;
    };
    
    std::unordered_map<std::string, std::shared_ptr<SymbolData>> symbol_data_;
    mutable std::shared_mutex mutex_;
    
    // Callbacks
    TickCallback tick_callback_;
    TradeCallback trade_callback_;
    OrderBookCallback order_book_callback_;
    CandleCallback candle_callback_;
    StatisticsCallback statistics_callback_;
    
    // Performance tracking
    std::atomic<uint64_t> total_ticks_processed_{0};
    std::atomic<uint64_t> total_trades_processed_{0};
    std::atomic<uint64_t> total_order_books_processed_{0};
    
    // Helper methods
    std::shared_ptr<SymbolData> get_or_create_symbol(const std::string& symbol);
    void update_order_book(std::shared_ptr<SymbolData>& data, const OrderBook& order_book);
    void update_statistics(std::shared_ptr<SymbolData>& data, const Trade& trade);
    void calculate_aggregated_price(std::shared_ptr<SymbolData>& data);
    void cleanup_old_data(std::shared_ptr<SymbolData>& data);
    double calculate_volatility(const std::vector<double>& prices) const;
    double calculate_std_dev(const std::vector<double>& values) const;
};

// ============================================================================
// Inline Implementations
// ============================================================================

inline MarketDataProcessor::MarketDataProcessor(const MarketDataConfig& config) 
    : config_(config) {}

inline void MarketDataProcessor::update_config(const MarketDataConfig& config) {
    std::unique_lock lock(mutex_);
    config_ = config;
}

inline MarketDataProcessor::MarketDataConfig MarketDataProcessor::get_config() const {
    std::shared_lock lock(mutex_);
    return config_;
}

inline void MarketDataProcessor::register_symbol(const std::string& symbol) {
    std::unique_lock lock(mutex_);
    if (symbol_data_.find(symbol) == symbol_data_.end()) {
        symbol_data_[symbol] = std::make_shared<SymbolData>();
    }
}

inline void MarketDataProcessor::unregister_symbol(const std::string& symbol) {
    std::unique_lock lock(mutex_);
    symbol_data_.erase(symbol);
}

inline bool MarketDataProcessor::is_symbol_registered(const std::string& symbol) const {
    std::shared_lock lock(mutex_);
    return symbol_data_.find(symbol) != symbol_data_.end();
}

inline void MarketDataProcessor::set_tick_callback(TickCallback callback) {
    tick_callback_ = callback;
}

inline void MarketDataProcessor::set_trade_callback(TradeCallback callback) {
    trade_callback_ = callback;
}

inline void MarketDataProcessor::set_order_book_callback(OrderBookCallback callback) {
    order_book_callback_ = callback;
}

inline void MarketDataProcessor::set_candle_callback(CandleCallback callback) {
    candle_callback_ = callback;
}

inline void MarketDataProcessor::set_statistics_callback(StatisticsCallback callback) {
    statistics_callback_ = callback;
}

inline size_t MarketDataProcessor::get_tick_count(const std::string& symbol) const {
    std::shared_lock lock(mutex_);
    auto it = symbol_data_.find(symbol);
    if (it != symbol_data_.end()) {
        return it->second->tick_history.size();
    }
    return 0;
}

inline size_t MarketDataProcessor::get_trade_count(const std::string& symbol) const {
    std::shared_lock lock(mutex_);
    auto it = symbol_data_.find(symbol);
    if (it != symbol_data_.end()) {
        return it->second->trade_history.size();
    }
    return 0;
}

inline size_t MarketDataProcessor::get_order_book_count() const {
    std::shared_lock lock(mutex_);
    return symbol_data_.size();
}

inline void MarketDataProcessor::clear_all_data() {
    std::unique_lock lock(mutex_);
    symbol_data_.clear();
}

inline MarketDataProcessor::PerformanceMetrics MarketDataProcessor::get_performance_metrics() const {
    PerformanceMetrics metrics;
    metrics.total_ticks_processed = total_ticks_processed_.load();
    metrics.total_trades_processed = total_trades_processed_.load();
    metrics.total_order_books_processed = total_order_books_processed_.load();
    return metrics;
}

inline void MarketDataProcessor::reset_performance_metrics() {
    total_ticks_processed_ = 0;
    total_trades_processed_ = 0;
    total_order_books_processed_ = 0;
}

} // namespace market_data
} // namespace tiger

#endif // TIGER_MARKET_DATA_PROCESSOR_H
