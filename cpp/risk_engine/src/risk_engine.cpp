/**
 * TigerWallet Risk Engine Implementation
 * High-Performance C++ Risk Management
 */

#include "risk_engine.h"
#include <algorithm>
#include <cmath>
#include <numeric>
#include <random>

namespace tiger {
namespace risk {

// ============================================================================
// Order Validation Implementation
// ============================================================================

OrderValidationResult RiskEngine::validate_order(
    const Order& order,
    const Account& account
) {
    std::unique_lock lock(mutex_);
    
    OrderValidationResult result;
    result.adjusted_size = order.size;
    result.adjusted_price = order.price;
    
    // Check if account exists
    auto account_it = accounts_.find(order.trader_id);
    if (account_it == accounts_.end()) {
        result.approved = false;
        result.reject_reason = OrderRejectReason::RISK_LIMIT_EXCEEDED;
        result.message = "Account not registered";
        return result;
    }
    
    const Account& acc = account_it->second;
    
    // 1. Check circuit breaker
    if (is_circuit_breaker_triggered(order.symbol)) {
        result.approved = false;
        result.reject_reason = OrderRejectReason::CIRCUIT_BREAKER;
        result.message = "Circuit breaker triggered for symbol";
        return result;
    }
    
    // 2. Check order size limits
    if (order.size > config_.max_order_size) {
        result.approved = false;
        result.reject_reason = OrderRejectReason::ORDER_SIZE_EXCEEDED;
        result.message = "Order size exceeds maximum";
        result.adjusted_size = config_.max_order_size;
    }
    
    // 3. Check position limits after trade
    double current_position = 0.0;
    for (const auto& [id, pos] : positions_) {
        if (pos.symbol == order.symbol && pos.trader_id == order.trader_id) {
            current_position = pos.size;
            break;
        }
    }
    
    double new_position = current_position + 
        (order.side == OrderSide::BUY ? order.size : -order.size);
    
    if (std::abs(new_position) * order.price > config_.max_position_size) {
        result.approved = false;
        result.reject_reason = OrderRejectReason::POSITION_LIMIT;
        result.message = "Would exceed position limit";
        return result;
    }
    
    // 4. Check concentration limits
    if (!check_concentration_limits(order.symbol, order.size)) {
        result.approved = false;
        result.reject_reason = OrderRejectReason::CONCENTRATION_LIMIT;
        result.message = "Would exceed concentration limit";
        return result;
    }
    
    // 5. Calculate required margin
    double order_value = order.size * order.price;
    double required_margin = order_value * config_.initial_margin_rate;
    result.required_margin = required_margin;
    
    // 6. Check margin availability
    double available_margin = acc.available_balance;
    if (required_margin > available_margin) {
        result.approved = false;
        result.reject_reason = OrderRejectReason::MARGIN_INSUFFICIENT;
        result.message = "Insufficient margin";
        return result;
    }
    
    // 7. Check leverage limits
    double total_exposure = 0.0;
    for (const auto& [id, pos] : positions_) {
        if (pos.trader_id == order.trader_id) {
            total_exposure += std::abs(pos.size * pos.current_price);
        }
    }
    
    double new_exposure = total_exposure + order_value;
    double leverage = new_exposure / acc.balance;
    if (leverage > config_.max_leverage) {
        result.approved = false;
        result.reject_reason = OrderRejectReason::LEVERAGE_EXCEEDED;
        result.message = "Would exceed leverage limit";
        return result;
    }
    
    // 8. Check daily volume limits
    double daily_volume = daily_volumes_[order.trader_id];
    if (daily_volume + order_value > config_.max_daily_volume) {
        result.approved = false;
        result.reject_reason = OrderRejectReason::RISK_LIMIT_EXCEEDED;
        result.message = "Would exceed daily volume limit";
        return result;
    }
    
    // If we adjusted the size, update the result
    if (result.adjusted_size != order.size) {
        result.approved = false;
        return result;
    }
    
    // Update daily volume
    update_daily_volume(order.trader_id, order_value);
    
    return result;
}

// ============================================================================
// Risk Metrics Calculation
// ============================================================================

RiskMetrics RiskEngine::calculate_risk_metrics(const std::string& account_id) {
    std::shared_lock lock(mutex_);
    
    RiskMetrics metrics;
    
    auto account_it = accounts_.find(account_id);
    if (account_it == accounts_.end()) {
        return metrics;
    }
    
    const Account& account = account_it->second;
    
    // Calculate position metrics
    double total_long = 0.0;
    double total_short = 0.0;
    double total_margin = 0.0;
    double total_unrealized_pnl = 0.0;
    double largest_position = 0.0;
    
    int position_count = 0;
    
    for (const auto& [id, position] : positions_) {
        if (position.trader_id == account_id && 
            position.status == PositionStatus::OPEN) {
            
            double position_value = std::abs(position.size * position.current_price);
            
            if (position.size > 0) {
                total_long += position_value;
            } else {
                total_short += position_value;
            }
            
            total_margin += position.margin_used;
            total_unrealized_pnl += position.unrealized_pnl;
            
            if (position_value > largest_position) {
                largest_position = position_value;
            }
            
            position_count++;
        }
    }
    
    // Set metrics
    metrics.total_exposure = total_long + total_short;
    metrics.net_exposure = total_long - total_short;
    metrics.long_exposure = total_long;
    metrics.short_exposure = total_short;
    metrics.gross_exposure = metrics.total_exposure;
    
    metrics.total_margin_used = total_margin;
    metrics.available_margin = account.balance - total_margin;
    metrics.margin_ratio = account.balance > 0 ? 
        total_margin / account.balance : 0.0;
    
    metrics.unrealized_pnl = total_unrealized_pnl;
    metrics.num_positions = position_count;
    
    // Calculate concentration
    if (account.balance > 0) {
        metrics.largest_position_pct = largest_position / account.balance;
    }
    
    // Assess risk level
    metrics.overall_risk_level = assess_risk_level(metrics.margin_ratio);
    
    return metrics;
}

// ============================================================================
// VaR Calculation
// ============================================================================

double RiskEngine::calculate_var(
    const std::vector<double>& returns,
    int confidence_level
) {
    if (returns.empty()) {
        return 0.0;
    }
    
    // Sort returns
    std::vector<double> sorted_returns = returns;
    std::sort(sorted_returns.begin(), sorted_returns.end());
    
    // Calculate index for confidence level
    int index = static_cast<int>(
        returns.size() * (100 - confidence_level) / 100.0
    );
    
    // Return the VaR (absolute value of loss)
    return std::abs(sorted_returns[index]);
}

double RiskEngine::calculate_portfolio_var(
    const std::string& account_id,
    int confidence_level
) {
    std::shared_lock lock(mutex_);
    
    // Collect historical returns
    std::vector<double> returns;
    
    for (const auto& [id, position] : positions_) {
        if (position.trader_id == account_id) {
            // Simulate returns based on price changes
            double return_pct = (position.current_price - position.entry_price) 
                / position.entry_price;
            returns.push_back(return_pct);
        }
    }
    
    if (returns.empty()) {
        return 0.0;
    }
    
    return calculate_var(returns, confidence_level);
}

// ============================================================================
// Stress Testing
// ============================================================================

double RiskEngine::run_stress_test(
    const std::string& account_id,
    const StressScenario& scenario
) {
    std::shared_lock lock(mutex_);
    
    double total_loss = 0.0;
    
    for (const auto& [id, position] : positions_) {
        if (position.trader_id == account_id && 
            position.status == PositionStatus::OPEN) {
            
            // Calculate loss under stress scenario
            double price_change = scenario.price_change_pct * 
                scenario.volatility_multiplier;
            
            double new_price = position.current_price * (1.0 + price_change);
            double position_value = position.size * new_price;
            double entry_value = position.size * position.entry_price;
            
            double pnl = position_value - entry_value;
            
            // Apply correlation breakdown
            if (scenario.correlation_breakdown > 0) {
                pnl *= (1.0 + scenario.correlation_breakdown);
            }
            
            total_loss += pnl;
        }
    }
    
    return total_loss;
}

std::vector<double> RiskEngine::run_monte_carlo_stress(
    const std::string& account_id,
    int num_scenarios
) {
    std::shared_lock lock(mutex_);
    
    std::vector<double> results;
    results.reserve(num_scenarios);
    
    std::random_device rd;
    std::mt19937 gen(rd());
    std::normal_distribution<> dist(0.0, 1.0);
    
    // Get current portfolio
    std::vector<std::pair<std::string, double>> positions;
    for (const auto& [id, pos] : positions_) {
        if (pos.trader_id == account_id) {
            positions.push_back({pos.symbol, pos.size});
        }
    }
    
    // Run simulations
    for (int i = 0; i < num_scenarios; i++) {
        double scenario_loss = 0.0;
        
        for (const auto& [symbol, size] : positions) {
            // Generate random return
            double random_return = dist(gen);
            
            // Get current price
            auto price_it = prices_.find(symbol);
            if (price_it != prices_.end()) {
                double new_price = price_it->second.last * (1.0 + random_return * 0.1);
                scenario_loss += size * (new_price - price_it->second.last);
            }
        }
        
        results.push_back(scenario_loss);
    }
    
    return results;
}

// ============================================================================
// Liquidation
// ============================================================================

bool RiskEngine::check_liquidation(const Position& position) {
    if (position.status != PositionStatus::OPEN) {
        return false;
    }
    
    double margin_ratio = position.maintenance_margin / 
        (std::abs(position.size) * position.current_price);
    
    return margin_ratio < config_.liquidation_threshold;
}

std::vector<Position> RiskEngine::get_positions_to_liquidate() {
    std::shared_lock lock(mutex_);
    
    std::vector<Position> to_liquidate;
    
    for (const auto& [id, position] : positions_) {
        if (check_liquidation(position)) {
            to_liquidate.push_back(position);
        }
    }
    
    return to_liquidate;
}

// ============================================================================
// Post-Trade Risk Check
// ============================================================================

bool RiskEngine::check_post_trade_risk(const Order& order) {
    auto account_opt = get_account(order.trader_id);
    if (!account_opt) {
        return false;
    }
    
    auto result = validate_order(order, *account_opt);
    return result.approved;
}

// ============================================================================
// Helper Methods
// ============================================================================

double RiskEngine::calculate_position_value(const Position& position) const {
    return std::abs(position.size * position.current_price);
}

double RiskEngine::calculate_unrealized_pnl(const Position& position) const {
    return position.size * (position.current_price - position.entry_price);
}

double RiskEngine::calculate_required_margin(const Position& position) const {
    return std::abs(position.size * position.current_price) * 
        config_.initial_margin_rate;
}

double RiskEngine::calculate_portfolio_exposure(
    const std::string& account_id
) const {
    double exposure = 0.0;
    
    for (const auto& [id, position] : positions_) {
        if (position.trader_id == account_id) {
            exposure += std::abs(position.size * position.current_price);
        }
    }
    
    return exposure;
}

RiskLevel RiskEngine::assess_risk_level(double margin_ratio) const {
    if (margin_ratio < config_.liquidation_threshold) {
        return RiskLevel::CRITICAL;
    } else if (margin_ratio < config_.maintenance_margin_rate) {
        return RiskLevel::HIGH;
    } else if (margin_ratio < config_.initial_margin_rate * 0.7) {
        return RiskLevel::MEDIUM;
    }
    return RiskLevel::LOW;
}

double RiskEngine::calculate_correlation_adjustment() const {
    // Simplified correlation adjustment
    return 0.1;
}

bool RiskEngine::check_concentration_limits(
    const std::string& symbol,
    double order_size
) const {
    // Simplified concentration check
    return true;
}

void RiskEngine::update_daily_volume(
    const std::string& account_id,
    double volume
) {
    if (is_new_day()) {
        daily_volumes_.clear();
    }
    
    daily_volumes_[account_id] += volume;
}

bool RiskEngine::is_new_day() const {
    auto now = std::chrono::system_clock::now();
    auto today = std::chrono::duration_cast<std::chrono::days>(
        now.time_since_epoch()
    ).count();
    
    return false; // Simplified
}

void RiskEngine::reset_daily_limits() {
    std::unique_lock lock(mutex_);
    daily_volumes_.clear();
}

// ============================================================================
// Counterparty Methods
// ============================================================================

void RiskEngine::register_counterparty(const Counterparty& counterparty) {
    std::unique_lock lock(mutex_);
    counterparties_[counterparty.counterparty_id] = counterparty;
}

void RiskEngine::update_counterparty(const Counterparty& counterparty) {
    std::unique_lock lock(mutex_);
    counterparties_[counterparty.counterparty_id] = counterparty;
}

double RiskEngine::get_counterparty_exposure(
    const std::string& counterparty_id
) {
    std::shared_lock lock(mutex_);
    
    double exposure = 0.0;
    
    for (const auto& [id, position] : positions_) {
        // Simplified - would need counterparty association
        exposure += 0.0;
    }
    
    return exposure;
}

} // namespace risk
} // namespace tiger
