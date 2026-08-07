/**
 * TigerWallet Admin - Margin Trading Handler (C++ Ultra-Low Latency)
 * High-performance implementation for margin trading operations
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

// Margin Position Model
struct MarginPosition {
    uint64_t id;
    uint64_t user_id;
    std::string user_name;
    std::string pair;
    std::string side; // long, short
    double size;
    int leverage;
    double entry_price;
    double current_price;
    double pnl;
    double liquidation_price;
    std::string status; // open, liquidated, closed
    std::chrono::system_clock::time_point opened_at;
    std::chrono::system_clock::time_point closed_at;
};

struct LiquidationStats {
    uint64_t total_positions;
    double total_volume;
    uint64_t liquidations_today;
    double liquidated_volume;
};

// Margin Trading Handler
class MarginTradingHandler : public AdminHandler {
public:
    MarginTradingHandler();
    ~MarginTradingHandler() override = default;

    void Initialize(ConnectionPool* pool) override;

    // Position Management
    std::vector<MarginPosition> GetPositions(const std::string& status = "", int page = 1, int limit = 20);
    std::vector<MarginPosition> GetHistory(int page = 1, int limit = 20);
    std::optional<MarginPosition> GetPositionById(uint64_t id);
    
    // Liquidation
    bool LiquidatePosition(uint64_t id);
    bool UpdatePrices(const std::string& pair, double price);
    
    // Statistics
    LiquidationStats GetLiquidationStats();

private:
    ConnectionPool* pool_;
    CacheManager* cache_;
    
    void PrepareStatements();
    MarginPosition ParseRow(const Row& row);
    void UpdatePositionPnL(MarginPosition& pos, double current_price);
};

// Inline implementations
inline std::vector<MarginPosition> MarginTradingHandler::GetPositions(const std::string& status, int page, int limit) {
    auto start = std::chrono::high_resolution_clock::now();
    
    std::vector<MarginPosition> result;
    std::string query = "SELECT * FROM margin_positions";
    
    if (!status.empty() && status != "all") {
        query += " WHERE status = $1";
    }
    query += " ORDER BY opened_at DESC LIMIT $" + std::to_string(limit) + " OFFSET $" + std::to_string((page - 1) * limit);
    
    auto stmt = pool_->Prepare(query);
    auto rows = (!status.empty() && status != "all") ? stmt->Execute(status) : stmt->Execute();
    
    while (rows->Next()) {
        result.push_back(ParseRow(*rows));
    }
    
    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);
    log::Debug("GetPositions took {} microseconds", duration.count());
    
    return result;
}

inline bool MarginTradingHandler::LiquidatePosition(uint64_t id) {
    auto now = std::chrono::system_clock::now();
    auto stmt = pool_->Prepare(
        "UPDATE margin_positions SET status = 'liquidated', closed_at = $2, updated_at = NOW() "
        "WHERE id = $1 AND status = 'open'"
    );
    auto result = stmt->Execute(id, now);
    
    // Notify liquidation service
    if (result->AffectedRows() > 0) {
        NotifyLiquidation(id);
        return true;
    }
    return false;
}

inline bool MarginTradingHandler::UpdatePrices(const std::string& pair, double price) {
    auto start = std::chrono::high_resolution_clock::now();
    
    // Get all open positions for this pair
    auto get_stmt = pool_->Prepare("SELECT * FROM margin_positions WHERE pair = $1 AND status = 'open'");
    auto rows = get_stmt->Execute(pair);
    
    std::vector<MarginPosition> positions;
    while (rows->Next()) {
        positions.push_back(ParseRow(*rows));
    }
    
    // Batch update for performance
    auto update_stmt = pool_->Prepare(
        "UPDATE margin_positions SET current_price = $3, pnl = "
        "CASE WHEN side = 'long' THEN ($3 - entry_price) * size "
        "ELSE (entry_price - $3) * size END "
        "WHERE id = $1 AND status = 'open'"
    );
    
    for (const auto& pos : positions) {
        update_stmt->Execute(pos.id, pair, price);
    }
    
    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);
    log::Debug("UpdatePrices for {} positions took {} microseconds", positions.size(), duration.count());
    
    return true;
}

inline LiquidationStats MarginTradingHandler::GetLiquidationStats() {
    LiquidationStats stats{};
    
    auto today = std::chrono::system_clock::now();
    
    auto stmt1 = pool_->Prepare("SELECT COUNT(*), COALESCE(SUM(size * entry_price), 0) FROM margin_positions WHERE status = 'open'");
    auto row1 = stmt1->Execute();
    if (row1->Next()) {
        stats.total_positions = row1->GetUInt64(0);
        stats.total_volume = row1->GetDouble(1);
    }
    
    auto stmt2 = pool_->Prepare(
        "SELECT COUNT(*), COALESCE(SUM(size * entry_price), 0) FROM margin_positions "
        "WHERE status = 'liquidated' AND closed_at >= $1"
    );
    auto row2 = stmt2->Execute(today);
    if (row2->Next()) {
        stats.liquidations_today = row2->GetUInt64(0);
        stats.liquidated_volume = row2->GetDouble(1);
    }
    
    return stats;
}

} // namespace admin
} // namespace tiger
