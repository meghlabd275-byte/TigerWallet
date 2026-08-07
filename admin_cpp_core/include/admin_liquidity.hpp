/**
 * TigerWallet Admin - Liquidity Handler (C++ Ultra-Low Latency)
 * High-performance implementation for liquidity pool operations
 */

#pragma once

#include <string>
#include <vector>
#include <memory>
#include <chrono>
#include <optional>
#include "admin_handler.hpp"

namespace tiger {
namespace admin {

// Liquidity Pool Model
struct LiquidityPool {
    uint64_t id;
    std::string pair;
    std::string token_a;
    std::string token_b;
    double reserve_a;
    double reserve_b;
    double total_supply;
    double apr;
    double volume_24h;
    double fees_24h;
    std::string status; // active, inactive
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
};

struct LiquidityPosition {
    uint64_t id;
    uint64_t pool_id;
    uint64_t user_id;
    double lp_token_amount;
    double reserve_a;
    double reserve_b;
    std::chrono::system_clock::time_point created_at;
};

struct LiquidityStats {
    uint64_t total_pools;
    double total_value_locked;
    double volume_24h;
    double fees_24h;
};

// Liquidity Handler - Ultra-low latency implementation
class LiquidityHandler : public AdminHandler {
public:
    LiquidityHandler();
    ~LiquidityHandler() override = default;

    void Initialize(ConnectionPool* pool) override;

    // Pool Management
    std::vector<LiquidityPool> GetPools();
    std::optional<LiquidityPool> GetPoolById(uint64_t id);
    LiquidityPool CreatePool(const std::string& pair, const std::string& token_a, const std::string& token_b);
    bool UpdatePool(uint64_t id, const std::string& status);
    bool DeletePool(uint64_t id);

    // Liquidity Operations
    bool AddLiquidity(uint64_t pool_id, uint64_t user_id, double amount_a, double amount_b);
    bool RemoveLiquidity(uint64_t pool_id, uint64_t user_id, double lp_amount);

    // Statistics
    LiquidityStats GetStats();

private:
    ConnectionPool* pool_;
    CacheManager* cache_;
    std::mutex pool_mutex_;

    void PrepareStatements();
    LiquidityPool ParsePoolRow(const Row& row);
    LiquidityPosition ParsePositionRow(const Row& row);
    double CalculateLPTokens(double amount_a, double amount_b, double total_supply, double reserve_a, double reserve_b);
};

// Inline implementations for ultra-low latency
inline std::vector<LiquidityPool> LiquidityHandler::GetPools() {
    auto start = std::chrono::high_resolution_clock::now();
    
    std::vector<LiquidityPool> result;
    auto stmt = pool_->Prepare("SELECT * FROM liquidity_pools ORDER BY created_at DESC");
    auto rows = stmt->Execute();
    
    while (rows->Next()) {
        result.push_back(ParsePoolRow(*rows));
    }
    
    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);
    log::Debug("GetPools took {} microseconds", duration.count());
    
    return result;
}

inline std::optional<LiquidityPool> LiquidityHandler::GetPoolById(uint64_t id) {
    auto stmt = pool_->Prepare("SELECT * FROM liquidity_pools WHERE id = $1");
    auto rows = stmt->Execute(id);
    
    if (rows->Next()) {
        return ParsePoolRow(*rows);
    }
    return std::nullopt;
}

inline bool LiquidityHandler::AddLiquidity(uint64_t pool_id, uint64_t user_id, double amount_a, double amount_b) {
    std::lock_guard<std::mutex> lock(pool_mutex_);
    
    // Get current pool state
    auto pool = GetPoolById(pool_id);
    if (!pool) return false;
    
    // Calculate LP tokens to mint
    double lp_tokens = CalculateLPTokens(amount_a, amount_b, pool->total_supply, pool->reserve_a, pool->reserve_b);
    
    // Update pool reserves
    auto update_stmt = pool_->Prepare(
        "UPDATE liquidity_pools SET "
        "reserve_a = reserve_a + $2, "
        "reserve_b = reserve_b + $3, "
        "total_supply = total_supply + $4, "
        "volume_24h = volume_24h + $5 "
        "WHERE id = $1"
    );
    
    double volume = amount_a + amount_b;
    update_stmt->Execute(pool_id, amount_a, amount_b, lp_tokens, volume);
    
    // Create liquidity position
    auto pos_stmt = pool_->Prepare(
        "INSERT INTO liquidity_positions (pool_id, user_id, lp_token_amount, reserve_a, reserve_b, created_at) "
        "VALUES ($1, $2, $3, $4, $5, NOW())"
    );
    pos_stmt->Execute(pool_id, user_id, lp_tokens, amount_a, amount_b);
    
    return true;
}

inline bool LiquidityHandler::RemoveLiquidity(uint64_t pool_id, uint64_t user_id, double lp_amount) {
    std::lock_guard<std::mutex> lock(pool_mutex_);
    
    // Get user's position
    auto pos_stmt = pool_->Prepare(
        "SELECT * FROM liquidity_positions WHERE pool_id = $1 AND user_id = $2"
    );
    auto pos_rows = pos_stmt->Execute(pool_id, user_id);
    
    if (!pos_rows->Next()) return false;
    
    auto position = ParsePositionRow(*pos_rows);
    
    // Calculate proportional amounts
    double ratio = lp_amount / position.lp_token_amount;
    double amount_a = position.reserve_a * ratio;
    double amount_b = position.reserve_b * ratio;
    
    // Update pool
    auto pool_stmt = pool_->Prepare(
        "UPDATE liquidity_pools SET "
        "reserve_a = reserve_a - $2, "
        "reserve_b = reserve_b - $3, "
        "total_supply = total_supply - $4 "
        "WHERE id = $1"
    );
    pool_stmt->Execute(pool_id, amount_a, amount_b, lp_amount);
    
    // Update position
    auto update_pos_stmt = pool_->Prepare(
        "UPDATE liquidity_positions SET "
        "lp_token_amount = lp_token_amount - $2, "
        "reserve_a = reserve_a - $3, "
        "reserve_b = reserve_b - $4 "
        "WHERE pool_id = $1 AND user_id = $5"
    );
    update_pos_stmt->Execute(pool_id, lp_amount, amount_a, amount_b, user_id);
    
    return true;
}

inline LiquidityStats LiquidityHandler::GetStats() {
    LiquidityStats stats{};
    
    auto stmt1 = pool_->Prepare("SELECT COUNT(*) FROM liquidity_pools");
    auto row1 = stmt1->Execute();
    if (row1->Next()) {
        stats.total_pools = row1->GetUInt64(0);
    }
    
    auto stmt2 = pool_->Prepare(
        "SELECT COALESCE(SUM(reserve_a + reserve_b), 0), "
        "COALESCE(SUM(volume_24h), 0), COALESCE(SUM(fees_24h), 0) "
        "FROM liquidity_pools"
    );
    auto row2 = stmt2->Execute();
    if (row2->Next()) {
        stats.total_value_locked = row2->GetDouble(0);
        stats.volume_24h = row2->GetDouble(1);
        stats.fees_24h = row2->GetDouble(2);
    }
    
    return stats;
}

} // namespace admin
} // namespace tiger
