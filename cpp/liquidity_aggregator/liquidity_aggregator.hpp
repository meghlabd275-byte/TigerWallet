/**
 * TigerWallet High-Performance Liquidity Aggregator
 * Ultra-low latency C++17 liquidity aggregation for DEX
 * 
 * Features:
 * - Multi-source liquidity aggregation
 * - Smart order routing
 * - Best price execution
 * - Slippage optimization
 */

#ifndef TIGER_WALLET_LIQUIDITY_AGGREGATOR_HPP
#define TIGER_WALLET_LIQUIDITY_AGGREGATOR_HPP

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
#include <unordered_set>
#include <vector>
#include <algorithm>
#include <cmath>
#include <sstream>
#include <curl/curl.h>
#include <nlohmann/json.hpp>

// ============================================================================
// Configuration
// ============================================================================

struct LiquidityConfig {
    uint32_t max_routes = 10;
    uint32_t max_liquidity_sources = 100;
    double max_slippage_tolerance = 0.05;
    uint64_t quote_timeout_ms = 5000;
    uint64_t refresh_interval_ms = 1000;
    bool enable_split_routing = true;
    bool enable_smart_routing = true;
    double min_liquidity_threshold = 100.0;
};

// ============================================================================
// Types
// ============================================================================

using Price = double;
using Quantity = double;
using Timestamp = std::chrono::milliseconds;

enum class LiquiditySource {
    UNISWAP_V2,
    UNISWAP_V3,
    SUSHISWAP,
    CURVE,
    BALANCER,
    PANCAKESWAP,
    RAYDIUM,
    ORCA,
    JUPITER,
    SERUM,
    DEX_SCREENER,
    AGGREGATOR
};

enum class OrderSide {
    BUY,
    SELL
};

struct TokenPair {
    std::string base_token;
    std::string quote_token;
    
    std::string to_string() const {
        return base_token + "/" + quote_token;
    }
    
    bool operator==(const TokenPair& other) const {
        return base_token == other.base_token && quote_token == other.quote_token;
    }
};

struct TokenPairHash {
    size_t operator()(const TokenPair& pair) const {
        return std::hash<std::string>()(pair.base_token + pair.quote_token);
    }
};

struct Quote {
    std::string id;
    TokenPair pair;
    LiquiditySource source;
    Price price;
    Quantity available_quantity;
    Quantity estimated_quantity;
    double slippage;
    double fee;
    Timestamp timestamp;
    Timestamp expires_at;
    
    Price total_value() const {
        return price * estimated_quantity;
    }
    
    bool is_valid() const {
        auto now = std::chrono::duration_cast<Timestamp>(
            std::chrono::system_clock::now().time_since_epoch());
        return now < expires_at && available_quantity > 0;
    }
};

struct Route {
    LiquiditySource source;
    TokenPair pair;
    Price price;
    Quantity quantity;
    double proportion;
};

struct ExecutionPlan {
    std::string id;
    TokenPair pair;
    Quantity input_amount;
    Quantity output_amount;
    double expected_output;
    double min_output;
    double price_impact;
    double total_slippage;
    double total_fee;
    std::vector<Route> routes;
    Timestamp created_at;
    
    bool is_optimal() const {
        return routes.size() == 1 && routes[0].proportion >= 0.99;
    }
};

struct LiquidityPool {
    std::string id;
    TokenPair pair;
    LiquiditySource source;
    Price token0_price;
    Price token1_price;
    Quantity reserve0;
    Quantity reserve1;
    Quantity volume_24h;
    double apy;
    Timestamp last_update;
};

// ============================================================================
// Liquidity Aggregator
// ============================================================================

class LiquidityAggregator {
public:
    using QuoteCallback = std::function<void(const Quote&)>;
    using PlanCallback = std::function<void(const ExecutionPlan&)>;

private:
    LiquidityConfig config_;
    std::string backend_url_;
    double slippage_from_backend_ = 0.0;
    std::unordered_map<TokenPair, std::vector<LiquidityPool>, TokenPairHash> pools_;
    mutable std::shared_mutex pools_mutex_;
    std::unordered_map<TokenPair, std::vector<Quote>, TokenPairHash> recent_quotes_;
    mutable std::shared_mutex quotes_mutex_;
    std::unordered_set<std::string> pending_requests_;
    mutable std::mutex requests_mutex_;
    std::vector<QuoteCallback> quote_callbacks_;
    std::vector<PlanCallback> plan_callbacks_;
    std::atomic<bool> running_;
    std::thread refresh_thread_;
    std::atomic<uint64_t> total_quotes_{0};
    std::atomic<uint64_t> successful_quotes_{0};

public:
    explicit LiquidityAggregator(const LiquidityConfig& config = LiquidityConfig())
        : config_(config), running_(false) {
        initialize_sources();
    }
    
    ~LiquidityAggregator() {
        stop();
    }
    
    std::optional<Quote> get_quote(const TokenPair& pair, Quantity amount, OrderSide side) {
        std::vector<Quote> quotes = fetch_quotes_from_all_sources(pair, amount, side);
        if (quotes.empty()) {
            return std::nullopt;
        }
        
        Quote best_quote = side == OrderSide::BUY ? 
            *std::min_element(quotes.begin(), quotes.end(),
                [](const Quote& a, const Quote& b) { return a.price < b.price; }) :
            *std::max_element(quotes.begin(), quotes.end(),
                [](const Quote& a, const Quote& b) { return a.price < b.price; });
        
        std::unique_lock lock(quotes_mutex_);
        recent_quotes_[pair].push_back(best_quote);
        
        total_quotes_++;
        successful_quotes_++;
        
        return best_quote;
    }
    
    std::vector<Quote> get_quotes(const TokenPair& pair, Quantity amount, OrderSide side, uint32_t limit = 5) {
        auto quotes = fetch_quotes_from_all_sources(pair, amount, side);
        
        if (side == OrderSide::BUY) {
            std::sort(quotes.begin(), quotes.end(),
                [](const Quote& a, const Quote& b) { return a.price < b.price; });
        } else {
            std::sort(quotes.begin(), quotes.end(),
                [](const Quote& a, const Quote& b) { return a.price > b.price; });
        }
        
        if (quotes.size() > limit) {
            quotes.resize(limit);
        }
        
        return quotes;
    }
    
    std::optional<ExecutionPlan> create_execution_plan(const TokenPair& pair, Quantity input_amount, OrderSide side) {
        auto quotes = fetch_quotes_from_all_sources(pair, input_amount, side);
        if (quotes.empty()) {
            return std::nullopt;
        }
        
        ExecutionPlan plan;
        plan.id = generate_id();
        plan.pair = pair;
        plan.input_amount = input_amount;
        plan.created_at = std::chrono::duration_cast<Timestamp>(
            std::chrono::system_clock::now().time_since_epoch());
        
        if (config_.enable_split_routing && quotes.size() > 1) {
            plan = create_split_plan(quotes, input_amount, side);
        } else {
            plan = create_single_plan(quotes[0], input_amount, side);
        }
        
        return plan;
    }
    
    void add_pool(const LiquidityPool& pool) {
        std::unique_lock lock(pools_mutex_);
        pools_[pool.pair].push_back(pool);
    }
    
    void remove_pool(const std::string& pool_id) {
        std::unique_lock lock(pools_mutex_);
        for (auto& [pair, pools] : pools_) {
            pools.erase(
                std::remove_if(pools.begin(), pools.end(),
                    [&pool_id](const LiquidityPool& p) { return p.id == pool_id; }),
                pools.end());
        }
    }
    
    std::vector<LiquidityPool> get_pools(const TokenPair& pair) const {
        std::shared_lock lock(pools_mutex_);
        auto it = pools_.find(pair);
        if (it != pools_.end()) {
            return it->second;
        }
        return {};
    }
    
    void on_quote(QuoteCallback callback) {
        quote_callbacks_.push_back(callback);
    }
    
    void on_plan(PlanCallback callback) {
        plan_callbacks_.push_back(callback);
    }
    
    void start() {
        running_ = true;
        if (!refresh_thread_.joinable()) {
            refresh_thread_ = std::thread([this]() { refresh_loop(); });
        }
    }
    
    void stop() {
        running_ = false;
        if (refresh_thread_.joinable()) {
            refresh_thread_.join();
        }
    }
    
    uint64_t total_quotes() const { return total_quotes_; }
    uint64_t successful_quotes() const { return successful_quotes_; }
    double success_rate() const {
        uint64_t total = total_quotes_;
        return total > 0 ? static_cast<double>(successful_quotes_) / total : 0.0;
    }

private:
    void initialize_sources() {}
    
    std::vector<Quote> fetch_quotes_from_all_sources(const TokenPair& pair, Quantity amount, OrderSide side) {
        std::vector<Quote> quotes;
        for (const auto& source : get_supported_sources()) {
            auto quote = fetch_quote_from_source(source, pair, amount, side);
            if (quote) {
                quotes.push_back(*quote);
            }
        }
        return quotes;
    }
    
    std::optional<Quote> fetch_quote_from_source(LiquiditySource source, const TokenPair& pair, Quantity amount, OrderSide side) {
        // Fetch a real price from the canonical wallet_api AMM router
        // (GET /api/v1/amm/quote does a real on-chain getAmountsOut). If the
        // backend is unreachable or the pair is unknown, return std::nullopt
        // (fail-closed) — never fabricate a price.
        auto price = fetch_real_price(pair, source);
        if (!price) return std::nullopt;

        Quote quote;
        quote.id = generate_id();
        quote.pair = pair;
        quote.source = source;
        quote.price = *price;
        quote.available_quantity = amount * 10;
        quote.estimated_quantity = amount;
        quote.slippage = slippage_from_backend_;
        quote.fee = amount * 0.003;
        quote.timestamp = std::chrono::duration_cast<Timestamp>(
            std::chrono::system_clock::now().time_since_epoch());
        quote.expires_at = Timestamp(quote.timestamp.count() + config_.quote_timeout_ms);
        return quote;
    }

    static size_t curlWriteCb(char* ptr, size_t size, size_t nmemb, void* userdata) {
        auto* buf = static_cast<std::string*>(userdata);
        buf->append(ptr, size * nmemb);
        return size * nmemb;
    }

    static std::string sourceToString(LiquiditySource s) {
        switch (s) {
            case LiquiditySource::UNISWAP_V2: return "uniswap_v2";
            case LiquiditySource::UNISWAP_V3: return "uniswap_v3";
            case LiquiditySource::SUSHISWAP: return "sushiswap";
            case LiquiditySource::CURVE: return "curve";
            case LiquiditySource::BALANCER: return "balancer";
            case LiquiditySource::PANCAKESWAP: return "pancakeswap";
            default: return "uniswap_v2";
        }
    }

    std::optional<Price> fetch_real_price(const TokenPair& pair, LiquiditySource source) {
        std::string base = backend_url_.empty() ? "http://localhost:8443" : backend_url_;
        std::string url = base + "/api/v1/amm/quote?from=" + pair.base_token +
                          "&to=" + pair.quote_token + "&amount=1&source=" + sourceToString(source);
        CURL* curl = curl_easy_init();
        if (!curl) return std::nullopt;
        std::string resp;
        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curlWriteCb);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &resp);
        curl_easy_setopt(curl, CURLOPT_TIMEOUT, 5L);
        curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 3L);
        CURLcode rc = curl_easy_perform(curl);
        long http = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http);
        curl_easy_cleanup(curl);
        if (rc != CURLE_OK || http != 200 || resp.empty()) return std::nullopt;
        try {
            auto j = nlohmann::json::parse(resp);
            if (j.contains("price_out") && j["price_out"].is_number()) {
                if (j.contains("slippage") && j["slippage"].is_number())
                    slippage_from_backend_ = j["slippage"].get<double>();
                return j["price_out"].get<double>();
            }
            if (j.contains("expected_out") && j["expected_out"].is_number())
                return j["expected_out"].get<double>();
        } catch (...) {
            return std::nullopt;
        }
        return std::nullopt;
    }
    
    ExecutionPlan create_single_plan(const Quote& quote, Quantity input_amount, OrderSide side) {
        ExecutionPlan plan;
        plan.id = generate_id();
        plan.pair = quote.pair;
        plan.input_amount = input_amount;
        plan.expected_output = input_amount / quote.price;
        plan.min_output = plan.expected_output * (1.0 - quote.slippage);
        plan.price_impact = quote.slippage;
        plan.total_slippage = quote.slippage;
        plan.total_fee = quote.fee;
        plan.output_amount = plan.expected_output;
        
        Route route;
        route.source = quote.source;
        route.pair = quote.pair;
        route.price = quote.price;
        route.quantity = plan.output_amount;
        route.proportion = 1.0;
        
        plan.routes.push_back(route);
        return plan;
    }
    
    ExecutionPlan create_split_plan(const std::vector<Quote>& quotes, Quantity input_amount, OrderSide side) {
        ExecutionPlan plan;
        plan.id = generate_id();
        plan.pair = quotes[0].pair;
        plan.input_amount = input_amount;
        
        Quantity remaining = input_amount;
        double total_expected = 0;
        
        for (size_t i = 0; i < quotes.size() && remaining > 0; i++) {
            const auto& quote = quotes[i];
            Quantity qty = std::min(remaining, quote.available_quantity);
            double proportion = qty / input_amount;
            
            Route route;
            route.source = quote.source;
            route.pair = quote.pair;
            route.price = quote.price;
            route.quantity = qty / quote.price;
            route.proportion = proportion;
            
            plan.routes.push_back(route);
            total_expected += route.quantity;
            remaining -= qty;
        }
        
        plan.expected_output = total_expected;
        plan.min_output = total_expected * 0.99;
        plan.price_impact = 0;
        for (const auto& route : plan.routes) {
            plan.price_impact += route.proportion * 0.001;
        }
        plan.total_slippage = plan.price_impact;
        plan.total_fee = input_amount * 0.003;
        plan.output_amount = total_expected;
        
        return plan;
    }
    
    void refresh_loop() {
        while (running_) {
            refresh_pools();
            std::this_thread::sleep_for(
                std::chrono::milliseconds(config_.refresh_interval_ms));
        }
    }
    
    void refresh_pools() {}
    
    std::vector<LiquiditySource> get_supported_sources() const {
        return {
            LiquiditySource::UNISWAP_V3,
            LiquiditySource::SUSHISWAP,
            LiquiditySource::CURVE,
            LiquiditySource::BALANCER,
        };
    }
    
    std::string generate_id() {
        static uint64_t counter = 0;
        return "quote_" + std::to_string(++counter) + "_" + 
               std::to_string(std::chrono::system_clock::now().time_since_epoch().count());
    }
};

inline std::unique_ptr<LiquidityAggregator> create_liquidity_aggregator(
    const LiquidityConfig& config = LiquidityConfig()) {
    return std::make_unique<LiquidityAggregator>(config);
}

#endif
