/**
 * TigerWallet High-Frequency Trading Risk Engine
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - Real-time position monitoring
 * - Margin calculation
 * - Liquidation risk assessment
 * - Portfolio stress testing
 * - Value at Risk (VaR) calculation
 * - Exposure limits enforcement
 * - Order size validation
 * - Counterparty risk management
 */

#ifndef TIGER_RISK_ENGINE_H
#define TIGER_RISK_ENGINE_H

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

namespace tiger {
namespace risk {

// ============================================================================
// Configuration
// ============================================================================

struct RiskConfig {
    // Position limits
    double max_position_size = 10000000.0;      // Max position in USD
    double max_order_size = 1000000.0;          // Max single order
    double max_daily_volume = 50000000.0;       // Max daily trading volume
    
    // Margin requirements
    double initial_margin_rate = 0.10;          // 10% initial margin
    double maintenance_margin_rate = 0.05;      // 5% maintenance margin
    double liquidation_threshold = 0.03;         // 3% liquidation
    
    // Risk limits
    double max_leverage = 10.0;                 // Max 10x leverage
    double max_portfolio_var = 0.02;            // 2% VaR limit
    double max_single_trade_var = 0.005;        // 0.5% single trade VaR
    
    // Concentration limits
    double max_single_asset_concentration = 0.30; // 30% max in single asset
    double max_single_counterparty = 0.20;        // 20% max single counterparty
    
    // Trading halts
    double price_move_halt_threshold = 0.05;      // 5% price move halt
    double circuit_breaker_threshold = 0.10;      // 10% circuit breaker
    
    // Time windows
    int var_confidence_level = 99;              // 99% VaR confidence
    int var_lookback_days = 30;                  // 30 day lookback
    int stress_test_scenarios = 100;             // Monte Carlo scenarios
};

// ============================================================================
// Data Types
// ============================================================================

enum class OrderSide { BUY, SELL };
enum class OrderType { MARKET, LIMIT, STOP, STOP_LIMIT };
enum class PositionStatus { OPEN, CLOSED, LIQUIDATING, LIQUIDATED };
enum class RiskLevel { LOW, MEDIUM, HIGH, CRITICAL };
enum class OrderRejectReason { 
    NONE, 
    POSITION_LIMIT, 
    MARGIN_INSUFFICIENT, 
    ORDER_SIZE_EXCEEDED,
    LEVERAGE_EXCEEDED,
    CONCENTRATION_LIMIT,
    CIRCUIT_BREAKER,
    PRICE_MOVE_HALT,
    RISK_LIMIT_EXCEEDED
};

// Price data
struct PriceData {
    std::string symbol;
    double bid;
    double ask;
    double last;
    double volume_24h;
    double change_24h;
    uint64_t timestamp;
};

// Position
struct Position {
    std::string position_id;
    std::string symbol;
    std::string trader_id;
    double size;                     // Positive = long, negative = short
    double entry_price;
    double current_price;
    double unrealized_pnl;
    double realized_pnl;
    double margin_used;
    double maintenance_margin;
    PositionStatus status;
    uint64_t open_time;
    uint64_t last_update_time;
};

// Order
struct Order {
    std::string order_id;
    std::string symbol;
    std::string trader_id;
    OrderSide side;
    OrderType type;
    double size;
    double price;
    double filled_size;
    double avg_fill_price;
    uint64_t created_time;
    uint64_t updated_time;
};

// Account
struct Account {
    std::string account_id;
    double balance;
    double available_balance;
    double total_margin;
    double unrealized_pnl;
    double realized_pnl;
    double total_equity;
    std::vector<Position> positions;
    std::vector<Order> pending_orders;
};

// Counterparty info
struct Counterparty {
    std::string counterparty_id;
    std::string name;
    double exposure;
    double limit;
    double current_utilization;
    RiskLevel risk_rating;
};

// ============================================================================
// Risk Metrics
// ============================================================================

struct RiskMetrics {
    // Position metrics
    double total_exposure = 0.0;
    double net_exposure = 0.0;
    double long_exposure = 0.0;
    double short_exposure = 0.0;
    double gross_exposure = 0.0;
    
    // Margin metrics
    double total_margin_used = 0.0;
    double available_margin = 0.0;
    double margin_ratio = 0.0;
    double maintenance_margin_ratio = 0.0;
    
    // PnL
    double unrealized_pnl = 0.0;
    double realized_pnl = 0.0;
    double daily_pnl = 0.0;
    double max_drawdown = 0.0;
    
    // Risk metrics
    double value_at_risk = 0.0;          // VaR
    double conditional_var = 0.0;        // CVaR / Expected Shortfall
    double stress_loss = 0.0;
    
    // Concentration
    double largest_position_pct = 0.0;
    double largest_counterparty_pct = 0.0;
    
    // Counts
    int num_positions = 0;
    int num_pending_orders = 0;
    RiskLevel overall_risk_level = RiskLevel::LOW;
};

// ============================================================================
// Order Validation Result
// ============================================================================

struct OrderValidationResult {
    bool approved = true;
    OrderRejectReason reject_reason = OrderRejectReason::NONE;
    std::string message;
    double adjusted_size = 0.0;
    double adjusted_price = 0.0;
    double required_margin = 0.0;
    RiskMetrics projected_metrics;
};

// ============================================================================
// Stress Test Scenario
// ============================================================================

struct StressScenario {
    std::string name;
    double price_change_pct;
    double volatility_multiplier;
    double correlation_breakdown;
    double liquidity_shock;
};

// ============================================================================
// Risk Engine Core
// ============================================================================

class RiskEngine {
public:
    explicit RiskEngine(const RiskConfig& config);
    ~RiskEngine() = default;
    
    // Disable copying
    RiskEngine(const RiskEngine&) = delete;
    RiskEngine& operator=(const RiskEngine&) = delete;
    
    // Configuration
    void update_config(const RiskConfig& config);
    RiskConfig get_config() const;
    
    // Account management
    void register_account(const Account& account);
    void update_account(const Account& account);
    void remove_account(const std::string& account_id);
    std::optional<Account> get_account(const std::string& account_id);
    
    // Position management
    void update_position(const Position& position);
    void remove_position(const std::string& position_id);
    std::optional<Position> get_position(const std::string& position_id);
    std::vector<Position> get_all_positions();
    
    // Counterparty management
    void register_counterparty(const Counterparty& counterparty);
    void update_counterparty(const Counterparty& counterparty);
    double get_counterparty_exposure(const std::string& counterparty_id);
    
    // Price updates
    void update_price(const PriceData& price);
    PriceData get_price(const std::string& symbol) const;
    std::unordered_map<std::string, PriceData> get_all_prices() const;
    
    // Order validation
    OrderValidationResult validate_order(
        const Order& order,
        const Account& account
    );
    
    // Post-trade risk check
    bool check_post_trade_risk(const Order& order);
    
    // Calculate risk metrics
    RiskMetrics calculate_risk_metrics(
        const std::string& account_id
    );
    
    // VaR calculation
    double calculate_var(
        const std::vector<double>& returns,
        int confidence_level
    );
    
    double calculate_portfolio_var(
        const std::string& account_id,
        int confidence_level
    );
    
    // Stress testing
    double run_stress_test(
        const std::string& account_id,
        const StressScenario& scenario
    );
    
    std::vector<double> run_monte_carlo_stress(
        const std::string& account_id,
        int num_scenarios
    );
    
    // Liquidation check
    bool check_liquidation(const Position& position);
    std::vector<Position> get_positions_to_liquidate();
    
    // Circuit breaker
    bool is_circuit_breaker_triggered(const std::string& symbol);
    void reset_circuit_breaker(const std::string& symbol);
    
    // Daily reset
    void reset_daily_limits();
    
    // Statistics
    size_t get_account_count() const;
    size_t get_position_count() const;
    size_t get_order_count() const;
    
private:
    // Configuration
    RiskConfig config_;
    
    // Data storage
    std::unordered_map<std::string, Account> accounts_;
    std::unordered_map<std::string, Position> positions_;
    std::unordered_map<std::string, Counterparty> counterparties_;
    std::unordered_map<std::string, PriceData> prices_;
    std::unordered_map<std::string, std::deque<Order>> order_history_;
    
    // Circuit breaker state
    std::unordered_map<std::string, double> last_prices_;
    std::unordered_map<std::string, bool> circuit_breaker_triggered_;
    
    // Daily trading
    std::unordered_map<std::string, double> daily_volumes_;
    std::unordered_map<std::string, uint64_t> last_reset_date_;
    
    // Mutex for thread safety
    mutable std::shared_mutex mutex_;
    
    // Helper methods
    double calculate_position_value(const Position& position) const;
    double calculate_unrealized_pnl(const Position& position) const;
    double calculate_required_margin(const Position& position) const;
    double calculate_portfolio_exposure(const std::string& account_id) const;
    RiskLevel assess_risk_level(double margin_ratio) const;
    double calculate_correlation_adjustment() const;
    bool check_concentration_limits(
        const std::string& symbol,
        double order_size
    ) const;
    void update_daily_volume(
        const std::string& account_id,
        double volume
    );
    bool is_new_day() const;
};

// ============================================================================
// Risk Manager (Singleton)
// ============================================================================

class RiskManager {
public:
    static RiskManager& get_instance();
    
    // Engine access
    RiskEngine& get_engine() { return engine_; }
    const RiskEngine& get_engine() const { return engine_; }
    
    // Convenience methods
    OrderValidationResult validate_order(const Order& order);
    RiskMetrics get_account_risk(const std::string& account_id);
    double get_portfolio_var(const std::string& account_id);
    
    // Lifecycle
    void initialize(const RiskConfig& config);
    void shutdown();
    
private:
    RiskManager() = default;
    ~RiskManager() = default;
    
    RiskManager(const RiskManager&) = delete;
    RiskManager& operator=(const RiskManager&) = delete;
    
    RiskEngine engine_;
    bool initialized_ = false;
};

// ============================================================================
// Inline Implementations
// ============================================================================

inline RiskEngine::RiskEngine(const RiskConfig& config) 
    : config_(config) {}

inline void RiskEngine::update_config(const RiskConfig& config) {
    std::unique_lock lock(mutex_);
    config_ = config;
}

inline RiskConfig RiskEngine::get_config() const {
    std::shared_lock lock(mutex_);
    return config_;
}

inline void RiskEngine::register_account(const Account& account) {
    std::unique_lock lock(mutex_);
    accounts_[account.account_id] = account;
}

inline void RiskEngine::update_account(const Account& account) {
    std::unique_lock lock(mutex_);
    accounts_[account.account_id] = account;
}

inline void RiskEngine::remove_account(const std::string& account_id) {
    std::unique_lock lock(mutex_);
    accounts_.erase(account_id);
}

inline std::optional<Account> RiskEngine::get_account(const std::string& account_id) {
    std::shared_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it != accounts_.end()) {
        return it->second;
    }
    return std::nullopt;
}

inline void RiskEngine::update_position(const Position& position) {
    std::unique_lock lock(mutex_);
    positions_[position.position_id] = position;
}

inline void RiskEngine::remove_position(const std::string& position_id) {
    std::unique_lock lock(mutex_);
    positions_.erase(position_id);
}

inline std::optional<Position> RiskEngine::get_position(const std::string& position_id) {
    std::shared_lock lock(mutex_);
    auto it = positions_.find(position_id);
    if (it != positions_.end()) {
        return it->second;
    }
    return std::nullopt;
}

inline std::vector<Position> RiskEngine::get_all_positions() {
    std::shared_lock lock(mutex_);
    std::vector<Position> result;
    result.reserve(positions_.size());
    for (const auto& [id, pos] : positions_) {
        result.push_back(pos);
    }
    return result;
}

inline void RiskEngine::register_counterparty(const Counterparty& counterparty) {
    std::unique_lock lock(mutex_);
    counterparties_[counterparty.counterparty_id] = counterparty;
}

inline void RiskEngine::update_price(const PriceData& price) {
    std::unique_lock lock(mutex_);
    
    auto it = last_prices_.find(price.symbol);
    if (it != last_prices_.end()) {
        double price_change = std::abs(price.last - it->second) / it->second;
        
        // Check price move halt
        if (price_change > config_.price_move_halt_threshold) {
            circuit_breaker_triggered_[price.symbol] = true;
        }
        
        // Check circuit breaker
        if (price_change > config_.circuit_breaker_threshold) {
            circuit_breaker_triggered_[price.symbol] = true;
        }
    }
    
    last_prices_[price.symbol] = price.last;
    prices_[price.symbol] = price;
}

inline PriceData RiskEngine::get_price(const std::string& symbol) const {
    std::shared_lock lock(mutex_);
    auto it = prices_.find(symbol);
    if (it != prices_.end()) {
        return it->second;
    }
    return PriceData{};
}

inline std::unordered_map<std::string, PriceData> RiskEngine::get_all_prices() const {
    std::shared_lock lock(mutex_);
    return prices_;
}

inline size_t RiskEngine::get_account_count() const {
    std::shared_lock lock(mutex_);
    return accounts_.size();
}

inline size_t RiskEngine::get_position_count() const {
    std::shared_lock lock(mutex_);
    return positions_.size();
}

inline size_t RiskEngine::get_order_count() const {
    std::shared_lock lock(mutex_);
    size_t count = 0;
    for (const auto& [id, orders] : order_history_) {
        count += orders.size();
    }
    return count;
}

inline bool RiskEngine::is_circuit_breaker_triggered(const std::string& symbol) {
    std::shared_lock lock(mutex_);
    auto it = circuit_breaker_triggered_.find(symbol);
    return it != circuit_breaker_triggered_.end() && it->second;
}

inline void RiskEngine::reset_circuit_breaker(const std::string& symbol) {
    std::unique_lock lock(mutex_);
    circuit_breaker_triggered_[symbol] = false;
}

// Risk Manager inline methods
inline OrderValidationResult RiskManager::validate_order(const Order& order) {
    auto account_opt = engine_.get_account(order.trader_id);
    if (!account_opt) {
        return OrderValidationResult{
            false,
            OrderRejectReason::RISK_LIMIT_EXCEEDED,
            "Account not found"
        };
    }
    return engine_.validate_order(order, *account_opt);
}

inline RiskMetrics RiskManager::get_account_risk(const std::string& account_id) {
    return engine_.calculate_risk_metrics(account_id);
}

inline double RiskManager::get_portfolio_var(const std::string& account_id) {
    return engine_.calculate_portfolio_var(account_id, 99);
}

inline void RiskManager::initialize(const RiskConfig& config) {
    engine_ = RiskEngine(config);
    initialized_ = true;
}

inline void RiskManager::shutdown() {
    initialized_ = false;
}

} // namespace risk
} // namespace tiger

#endif // TIGER_RISK_ENGINE_H
