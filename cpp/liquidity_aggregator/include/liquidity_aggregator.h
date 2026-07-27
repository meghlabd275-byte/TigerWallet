/**
 * TigerWallet High-Frequency Liquidity Aggregator
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - Multi-DEX liquidity aggregation
 * - Smart order routing
 * - Slippage optimization
 * - Gas optimization
 * - Split order execution
 * - MEV protection
 */

#ifndef TIGER_LIQUIDITY_AGGREGATOR_H
#define TIGER_LIQUIDITY_AGGREGATOR_H

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
#include <cmath>

namespace tiger {
namespace liquidity {

// ============================================================================
// Configuration
// ============================================================================

struct AggregatorConfig {
    // Routing
    int max_hops = 3;
    double max_slippage = 0.03;
    double gas_price_multiplier = 1.2;
    bool enable_split_orders = true;
    int max_split_parts = 5;
    
    // Caching
    int quote_cache_ttl_ms = 1000;
    int quote_cache_size = 1000;
    
    // Execution
    int max_retry = 3;
    int retry_delay_ms = 100;
    int deadline_seconds = 300;
    
    // MEV Protection
    bool enable_mev_protection = true;
    bool use_private_rpc = false;
};

// ============================================================================
// Data Types
// ============================================================================

enum class OrderSide { BUY, SELL };
enum class OrderType { MARKET, LIMIT, GTC, FOK, IOC };
enum class DexType { UNISWAP, SUSHISWAP, CURVE, BALANCER, DODO, PANCAKE, BISWAP };

struct Token {
    std::string address;
    std::string symbol;
    std::string name;
    int decimals;
};

struct DexQuote {
    std::string dex_name;
    DexType dex_type;
    double price;
    double gas_cost;
    double impact;
    double amount_out_min;
    uint64_t expires_at;
    std::string route;
    double confidence;
};

struct RouteStep {
    Token from_token;
    Token to_token;
    std::string pool_address;
    DexType dex;
    double swap_fee;
};

struct TradeRoute {
    std::vector<RouteStep> steps;
    double total_gas;
    double total_price_impact;
    double total_slippage;
    double expected_output;
    double deadline;
};

struct Order {
    std::string order_id;
    std::string from_token;
    std::string to_token;
    double from_amount;
    OrderSide side;
    OrderType type;
    double limit_price;
    std::string recipient;
    uint64_t created_at;
};

struct OrderResult {
    std::string order_id;
    bool success;
    std::string tx_hash;
    double input_spent;
    double output_received;
    double gas_used;
    double total_cost;
    std::string error;
};

// ============================================================================
// Price Impact Calculator
// ============================================================================

struct PriceImpact {
    double spot_price = 0.0;
    double exec_price = 0.0;
    double impact_percent = 0.0;
    double gas_adjusted_impact = 0.0;
};

// ============================================================================
// Liquidity Pool
// ============================================================================

struct LiquidityPool {
    std::string pool_id;
    DexType dex;
    Token token_a;
    Token token_b;
    double reserve_a;
    double reserve_b;
    double fee;
    std::string pool_address;
};

// ============================================================================
// Callbacks
// ============================================================================

using QuoteCallback = std::function<void(const std::vector<DexQuote>&)>;
using TradeCallback = std::function<void(const OrderResult&)>;
using ErrorCallback = std::function<void(const std::string&)>;

// ============================================================================
// Liquidity Aggregator Core
// ============================================================================

class LiquidityAggregator {
public:
    explicit LiquidityAggregator(const AggregatorConfig& config);
    ~LiquidityAggregator() = default;
    
    // Configuration
    void update_config(const AggregatorConfig& config);
    AggregatorConfig get_config() const;
    
    // Token management
    void register_token(const Token& token);
    void register_tokens(const std::vector<Token>& tokens);
    std::optional<Token> get_token(const std::string& address) const;
    std::vector<Token> get_all_tokens() const;
    
    // DEX management
    void register_dex(DexType dex_type, const std::string& name, const std::string& rpc_url);
    void update_dex(DexType dex_type, bool enabled);
    std::vector<std::string> get_enabled_dexs() const;
    
    // Quote generation
    std::vector<DexQuote> get_quotes(
        const std::string& from_token,
        const std::string& to_token,
        double amount,
        OrderSide side
    );
    
    // Route finding
    std::optional<TradeRoute> find_best_route(
        const std::string& from_token,
        const std::string& to_token,
        double amount,
        OrderSide side
    );
    
    std::vector<TradeRoute> find_all_routes(
        const std::string& from_token,
        const std::string& to_token,
        double amount,
        OrderSide side
    );
    
    // Order execution
    OrderResult execute_order(const Order& order, const TradeRoute& route);
    void execute_order_async(const Order& order, const TradeRoute& route, TradeCallback callback);
    
    // Split order execution
    std::vector<OrderResult> execute_split_order(
        const Order& order,
        int num_splits,
        bool optimize_for_slippage
    );
    
    // Price impact calculation
    PriceImpact calculate_price_impact(
        const std::string& from_token,
        const std::string& to_token,
        double amount
    );
    
    // Gas estimation
    double estimate_gas(const TradeRoute& route);
    double estimate_gas_cost(const TradeRoute& route, double gas_price_gwei);
    
    // Liquidity queries
    double get_liquidity(
        const std::string& token_a,
        const std::string& token_b,
        DexType dex
    );
    
    double get_total_liquidity(
        const std::string& token_a,
        const std::string& token_b
    );
    
    // Callbacks
    void set_quote_callback(QuoteCallback callback);
    void set_trade_callback(TradeCallback callback);
    void set_error_callback(ErrorCallback callback);
    
    // Analytics
    struct AggregatorStats {
        uint64_t total_quotes = 0;
        uint64_t total_trades = 0;
        uint64_t failed_trades = 0;
        double avg_slippage = 0.0;
        double avg_gas_cost = 0.0;
        double total_volume = 0.0;
    };
    
    AggregatorStats get_stats() const;
    void reset_stats();

private:
    AggregatorConfig config_;
    
    // Data
    std::unordered_map<std::string, Token> tokens_;
    std::unordered_map<DexType, std::string> dex_names_;
    std::unordered_map<DexType, std::string> dex_rpc_urls_;
    std::unordered_map<DexType, bool> dex_enabled_;
    std::unordered_map<std::string, std::vector<LiquidityPool>> pools_;
    
    // Quote cache
    struct CachedQuote {
        DexQuote quote;
        uint64_t cached_at;
    };
    std::unordered_map<std::string, CachedQuote> quote_cache_;
    
    // Stats
    std::atomic<uint64_t> total_quotes_{0};
    std::atomic<uint64_t> total_trades_{0};
    std::atomic<uint64_t> failed_trades_{0};
    std::atomic<double> total_slippage_{0.0};
    std::atomic<double> total_gas_cost_{0.0};
    std::atomic<double> total_volume_{0.0};
    
    // Callbacks
    QuoteCallback quote_callback_;
    TradeCallback trade_callback_;
    ErrorCallback error_callback_;
    
    mutable std::shared_mutex mutex_;
    
    // Helper methods
    std::string get_cache_key(const std::string& from, const std::string& to, double amount) const;
    std::optional<DexQuote> get_cached_quote(const std::string& key) const;
    void cache_quote(const std::string& key, const DexQuote& quote);
    std::vector<std::string> find_token_path(
        const std::string& from_token,
        const std::string& to_token,
        int max_hops
    );
    double calculate_slippage(double amount, double reserve);
    double calculate_price_impact_single(double amount, double reserve);
    DexQuote fetch_quote_from_dex(
        DexType dex,
        const Token& from,
        const Token& to,
        double amount,
        OrderSide side
    );
};

// ============================================================================
// Inline Implementations
// ============================================================================

inline LiquidityAggregator::LiquidityAggregator(const AggregatorConfig& config) 
    : config_(config) {}

inline void LiquidityAggregator::update_config(const AggregatorConfig& config) {
    std::unique_lock lock(mutex_);
    config_ = config;
}

inline AggregatorConfig LiquidityAggregator::get_config() const {
    std::shared_lock lock(mutex_);
    return config_;
}

inline void LiquidityAggregator::register_token(const Token& token) {
    std::unique_lock lock(mutex_);
    tokens_[token.address] = token;
}

inline void LiquidityAggregator::register_tokens(const std::vector<Token>& tokens) {
    std::unique_lock lock(mutex_);
    for (const auto& token : tokens) {
        tokens_[token.address] = token;
    }
}

inline std::optional<Token> LiquidityAggregator::get_token(const std::string& address) const {
    std::shared_lock lock(mutex_);
    auto it = tokens_.find(address);
    if (it != tokens_.end()) {
        return it->second;
    }
    return std::nullopt;
}

inline std::vector<Token> LiquidityAggregator::get_all_tokens() const {
    std::shared_lock lock(mutex_);
    std::vector<Token> result;
    result.reserve(tokens_.size());
    for (const auto& [addr, token] : tokens_) {
        result.push_back(token);
    }
    return result;
}

inline void LiquidityAggregator::register_dex(DexType dex_type, const std::string& name, const std::string& rpc_url) {
    std::unique_lock lock(mutex_);
    dex_names_[dex_type] = name;
    dex_rpc_urls_[dex_type] = rpc_url;
    dex_enabled_[dex_type] = true;
}

inline void LiquidityAggregator::update_dex(DexType dex_type, bool enabled) {
    std::unique_lock lock(mutex_);
    dex_enabled_[dex_type] = enabled;
}

inline std::vector<std::string> LiquidityAggregator::get_enabled_dexs() const {
    std::shared_lock lock(mutex_);
    std::vector<std::string> result;
    for (const auto& [dex, enabled] : dex_enabled_) {
        if (enabled) {
            auto it = dex_names_.find(dex);
            if (it != dex_names_.end()) {
                result.push_back(it->second);
            }
        }
    }
    return result;
}

inline void LiquidityAggregator::set_quote_callback(QuoteCallback callback) {
    quote_callback_ = callback;
}

inline void LiquidityAggregator::set_trade_callback(TradeCallback callback) {
    trade_callback_ = callback;
}

inline void LiquidityAggregator::set_error_callback(ErrorCallback callback) {
    error_callback_ = callback;
}

inline LiquidityAggregator::AggregatorStats LiquidityAggregator::get_stats() const {
    AggregatorStats stats;
    stats.total_quotes = total_quotes_.load();
    stats.total_trades = total_trades_.load();
    stats.failed_trades = failed_trades_.load();
    
    uint64_t trades = total_trades_.load();
    if (trades > 0) {
        stats.avg_slippage = total_slippage_.load() / trades;
        stats.avg_gas_cost = total_gas_cost_.load() / trades;
    }
    
    stats.total_volume = total_volume_.load();
    return stats;
}

inline void LiquidityAggregator::reset_stats() {
    total_quotes_ = 0;
    total_trades_ = 0;
    failed_trades_ = 0;
    total_slippage_ = 0.0;
    total_gas_cost_ = 0.0;
    total_volume_ = 0.0;
}

} // namespace liquidity
} // namespace tiger

#endif // TIGER_LIQUIDITY_AGGREGATOR_H
