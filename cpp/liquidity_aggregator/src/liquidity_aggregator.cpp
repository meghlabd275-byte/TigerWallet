/**
 * TigerWallet Liquidity Aggregator Implementation
 * High-Performance C++ Trading System
 */

#include "liquidity_aggregator.h"
#include <algorithm>
#include <cmath>
#include <numeric>
#include <random>

namespace tiger {
namespace liquidity {

// ============================================================================
// Quote Generation Implementation
// ============================================================================

std::vector<DexQuote> LiquidityAggregator::get_quotes(
    const std::string& from_token,
    const std::string& to_token,
    double amount,
    OrderSide side
) {
    std::shared_lock lock(mutex_);
    
    std::vector<DexQuote> quotes;
    
    // Check cache first
    std::string cache_key = get_cache_key(from_token, to_token, amount);
    if (auto cached = get_cached_quote(cache_key)) {
        quotes.push_back(*cached);
        return quotes;
    }
    
    // Get tokens
    auto from_it = tokens_.find(from_token);
    auto to_it = tokens_.find(to_token);
    
    if (from_it == tokens_.end() || to_it == tokens_.end()) {
        return quotes;
    }
    
    const Token& from = from_it->second;
    const Token& to = to_it->second;
    
    // Fetch quotes from each enabled DEX
    for (const auto& [dex_type, enabled] : dex_enabled_) {
        if (!enabled) continue;
        
        auto quote = fetch_quote_from_dex(dex_type, from, to, amount, side);
        if (quote.price > 0) {
            quotes.push_back(quote);
            
            // Cache the best quote
            cache_key = get_cache_key(from_token, to_token, amount);
            cache_quote(cache_key, quote);
        }
    }
    
    // Sort by total output (price - gas cost)
    std::sort(quotes.begin(), quotes.end(), 
        [](const DexQuote& a, const DexQuote& b) {
            return (a.price - a.gas_cost) > (b.price - b.gas_cost);
        }
    );
    
    total_quotes_++;
    
    if (quote_callback_) {
        quote_callback_(quotes);
    }
    
    return quotes;
}

// ============================================================================
// Route Finding Implementation
// ============================================================================

std::optional<TradeRoute> LiquidityAggregator::find_best_route(
    const std::string& from_token,
    const std::string& to_token,
    double amount,
    OrderSide side
) {
    auto routes = find_all_routes(from_token, to_token, amount, side);
    
    if (routes.empty()) {
        return std::nullopt;
    }
    
    // Return the route with best output
    return routes[0];
}

std::vector<TradeRoute> LiquidityAggregator::find_all_routes(
    const std::string& from_token,
    const std::string& to_token,
    double amount,
    OrderSide side
) {
    std::shared_lock lock(mutex_);
    
    std::vector<TradeRoute> routes;
    
    // Direct route
    auto quotes = get_quotes(from_token, to_token, amount, side);
    for (const auto& quote : quotes) {
        TradeRoute route;
        route.expected_output = amount * quote.price * (1.0 - quote.impact);
        route.total_gas = quote.gas_cost;
        route.total_slippage = quote.impact;
        route.deadline = quote.expires_at;
        routes.push_back(route);
    }
    
    // Multi-hop routes via common tokens (USDC, USDT, WETH)
    std::vector<std::string> intermediate_tokens = {"USDC", "USDT", "WETH", "DAI"};
    
    for (const auto& intermediate : intermediate_tokens) {
        if (intermediate == from_token || intermediate == to_token) continue;
        
        // Check if we have both legs
        auto from_quotes = get_quotes(from_token, intermediate, amount, side);
        if (from_quotes.empty()) continue;
        
        auto to_quotes = get_quotes(intermediate, to_token, 
            amount * from_quotes[0].price, side);
        if (to_quotes.empty()) continue;
        
        // Calculate combined route
        TradeRoute route;
        route.expected_output = amount * from_quotes[0].price * to_quotes[0].price;
        route.total_gas = from_quotes[0].gas_cost + to_quotes[0].gas_cost;
        route.total_slippage = from_quotes[0].impact + to_quotes[0].impact;
        
        routes.push_back(route);
    }
    
    // Sort by expected output
    std::sort(routes.begin(), routes.end(),
        [](const TradeRoute& a, const TradeRoute& b) {
            return a.expected_output > b.expected_output;
        }
    );
    
    return routes;
}

// ============================================================================
// Order Execution Implementation
// ============================================================================

OrderResult LiquidityAggregator::execute_order(
    const Order& order,
    const TradeRoute& route
) {
    OrderResult result;
    result.order_id = order.order_id;
    
    // Simulate execution (in production, this would call DEX APIs)
    double slippage = calculate_slippage(order.from_amount, route.total_slippage * 100);
    
    result.input_spent = order.from_amount;
    result.output_received = route.expected_output * (1.0 - slippage);
    result.gas_used = route.total_gas;
    result.total_cost = result.gas_used * 0.00002; // Estimated gas cost
    result.success = true;
    
    // Update stats
    total_trades_++;
    total_slippage_ += slippage;
    total_gas_cost_ += result.total_cost;
    total_volume_ += order.from_amount;
    
    if (trade_callback_) {
        trade_callback_(result);
    }
    
    return result;
}

void LiquidityAggregator::execute_order_async(
    const Order& order,
    const TradeRoute& route,
    TradeCallback callback
) {
    // Execute in background thread (simplified)
    auto result = execute_order(order, route);
    if (callback) {
        callback(result);
    }
}

std::vector<OrderResult> LiquidityAggregator::execute_split_order(
    const Order& order,
    int num_splits,
    bool optimize_for_slippage
) {
    std::vector<OrderResult> results;
    results.reserve(num_splits);
    
    double split_amount = order.from_amount / num_splits;
    
    for (int i = 0; i < num_splits; i++) {
        Order split_order = order;
        split_order.order_id = order.order_id + "_" + std::to_string(i);
        
        // Find best route for this split
        auto route_opt = find_best_route(
            order.from_token,
            order.to_token,
            split_amount,
            order.side
        );
        
        if (route_opt) {
            auto result = execute_order(split_order, *route_opt);
            results.push_back(result);
        }
    }
    
    return results;
}

// ============================================================================
// Price Impact Calculation
// ============================================================================

PriceImpact LiquidityAggregator::calculate_price_impact(
    const std::string& from_token,
    const std::string& to_token,
    double amount
) {
    std::shared_lock lock(mutex_);
    
    PriceImpact impact;
    
    // Get current liquidity
    double liquidity = get_total_liquidity(from_token, to_token);
    if (liquidity <= 0) {
        return impact;
    }
    
    impact.spot_price = 1.0; // Simplified
    impact.exec_price = impact.spot_price * (1.0 - (amount / liquidity));
    impact.impact_percent = ((impact.spot_price - impact.exec_price) / impact.spot_price) * 100;
    impact.gas_adjusted_impact = impact.impact_percent; // Would add gas cost adjustment
    
    return impact;
}

// ============================================================================
// Gas Estimation
// ============================================================================

double LiquidityAggregator::estimate_gas(const TradeRoute& route) {
    // Base gas cost
    double gas = 21000; // Base transaction
    
    // Add gas for each swap step
    gas += route.steps.size() * 50000;
    
    // Add gas for price impact
    gas += route.total_price_impact * 1000;
    
    return gas;
}

double LiquidityAggregator::estimate_gas_cost(const TradeRoute& route, double gas_price_gwei) {
    double gas = estimate_gas(route);
    return gas * gas_price_gwei * 0.000000001;
}

// ============================================================================
// Liquidity Queries
// ============================================================================

double LiquidityAggregator::get_liquidity(
    const std::string& token_a,
    const std::string& token_b,
    DexType dex
) {
    std::shared_lock lock(mutex_);
    
    std::string pool_key = token_a + "_" + token_b;
    auto it = pools_.find(pool_key);
    if (it != pools_.end()) {
        for (const auto& pool : it->second) {
            if (pool.dex == dex) {
                return pool.reserve_a + pool.reserve_b;
            }
        }
    }
    
    // Simulate liquidity (in production, would query DEX)
    return 1000000.0 + (rand() % 1000000);
}

double LiquidityAggregator::get_total_liquidity(
    const std::string& token_a,
    const std::string& token_b
) {
    double total = 0.0;
    
    for (int i = 0; i <= (int)DexType::PANCAKE; i++) {
        total += get_liquidity(token_a, token_b, (DexType)i);
    }
    
    return total;
}

// ============================================================================
// Helper Methods Implementation
// ============================================================================

std::string LiquidityAggregator::get_cache_key(
    const std::string& from,
    const std::string& to,
    double amount
) const {
    return from + "_" + to + "_" + std::to_string((int)amount);
}

std::optional<DexQuote> LiquidityAggregator::get_cached_quote(const std::string& key) const {
    auto it = quote_cache_.find(key);
    if (it == quote_cache_.end()) {
        return std::nullopt;
    }
    
    auto now = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    if (now - it->second.cached_at > config_.quote_cache_ttl_ms) {
        return std::nullopt;
    }
    
    return it->second.quote;
}

void LiquidityAggregator::cache_quote(const std::string& key, const DexQuote& quote) {
    auto now = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    // Evict old entries if cache is full
    if (quote_cache_.size() >= (size_t)config_.quote_cache_size) {
        quote_cache_.erase(quote_cache_.begin());
    }
    
    quote_cache_[key] = {quote, (uint64_t)now};
}

std::vector<std::string> LiquidityAggregator::find_token_path(
    const std::string& from_token,
    const std::string& to_token,
    int max_hops
) {
    // Simplified path finding
    std::vector<std::string> path = {from_token, to_token};
    return path;
}

double LiquidityAggregator::calculate_slippage(double amount, double reserve) {
    if (reserve <= 0) return 0.0;
    return amount / reserve;
}

double LiquidityAggregator::calculate_price_impact_single(double amount, double reserve) {
    if (reserve <= 0) return 0.0;
    return (amount / reserve) * (1.0 + (amount / reserve) / 2.0);
}

DexQuote LiquidityAggregator::fetch_quote_from_dex(
    DexType dex,
    const Token& from,
    const Token& to,
    double amount,
    OrderSide side
) {
    DexQuote quote;
    quote.dex_type = dex;
    
    // Get DEX name
    auto name_it = dex_names_.find(dex);
    if (name_it != dex_names_.end()) {
        quote.dex_name = name_it->second;
    }
    
    // Simulate quote (in production, would call DEX API)
    double base_price = 1.0;
    if (from.symbol == "WETH" && to.symbol == "USDC") {
        base_price = 3000.0;
    } else if (from.symbol == "USDC" && to.symbol == "WETH") {
        base_price = 1.0 / 3000.0;
    }
    
    // Add some randomness to simulate market
    double random_factor = 1.0 + ((rand() % 100) - 50) / 1000.0;
    quote.price = base_price * random_factor;
    
    // Calculate impact
    double liquidity = get_liquidity(from.address, to.address, dex);
    quote.impact = calculate_price_impact_single(amount, liquidity);
    
    // Gas cost estimation
    quote.gas_cost = 50000 + (quote.impact * 10000);
    
    // Other fields
    quote.amount_out_min = amount * quote.price * (1.0 - quote.impact - 0.003); // 0.3% fee
    quote.expires_at = std::chrono::duration_cast<std::chrono::seconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count() + 30;
    
    quote.route = from.symbol + " -> " + to.symbol;
    quote.confidence = 0.95 - (quote.impact * 10);
    
    return quote;
}

} // namespace liquidity
} // namespace tiger
