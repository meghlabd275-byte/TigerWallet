/**
 * TigerWallet Admin - Crypto Cards Handler (C++ Ultra-Low Latency)
 * High-performance implementation for critical crypto card operations
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

// Crypto Card Model
struct CryptoCard {
    uint64_t id;
    uint64_t user_id;
    std::string user_name;
    std::string card_number;
    std::string currency;
    double balance;
    double limit;
    std::string status; // active, blocked, pending
    std::string card_type; // virtual, physical
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
};

// Request/Response
struct CryptoCardRequest {
    uint64_t user_id;
    std::string currency;
    double limit;
    std::string card_type;
};

struct CryptoCardResponse {
    bool success;
    std::string message;
    CryptoCard card;
};

// Crypto Cards Handler - Ultra-low latency implementation
class CryptoCardsHandler : public AdminHandler {
public:
    CryptoCardsHandler();
    ~CryptoCardsHandler() override = default;

    // Initialize handler with connection pool
    void Initialize(ConnectionPool* pool) override;

    // CRUD Operations - Optimized for ultra-low latency
    std::vector<CryptoCard> GetAll(const std::string& status = "", int page = 1, int limit = 20);
    std::optional<CryptoCard> GetById(uint64_t id);
    CryptoCardResponse Create(const CryptoCardRequest& request);
    bool Update(uint64_t id, const CryptoCardRequest& request);
    bool Delete(uint64_t id);

    // Card Operations
    bool BlockCard(uint64_t id);
    bool ActivateCard(uint64_t id);
    bool SetLimit(uint64_t id, double limit);

    // Statistics
    CryptoCardStats GetStats();

private:
    ConnectionPool* pool_;
    CacheManager* cache_;
    
    // Prepared statements for fast execution
    std::unique_ptr<PreparedStatement> stmt_get_all_;
    std::unique_ptr<PreparedStatement> stmt_get_by_id_;
    std::unique_ptr<PreparedStatement> stmt_create_;
    std::unique_ptr<PreparedStatement> stmt_update_;
    std::unique_ptr<PreparedStatement> stmt_block_;
    std::unique_ptr<PreparedStatement> stmt_activate_;
    
    // In-memory cache for hot data
    std::unordered_map<uint64_t, CryptoCard> card_cache_;
    std::mutex cache_mutex_;

    // Ultra-low latency helpers
    void PrepareStatements();
    CryptoCard ParseRow(const Row& row);
    void UpdateCache(const CryptoCard& card);
    void InvalidateCache(uint64_t id);
};

// Inline implementations for performance
inline std::vector<CryptoCard> CryptoCardsHandler::GetAll(const std::string& status, int page, int limit) {
    auto start = std::chrono::high_resolution_clock::now();
    
    std::vector<CryptoCard> result;
    std::string query = "SELECT * FROM crypto_cards";
    
    if (!status.empty() && status != "all") {
        query += " WHERE status = $1";
    }
    query += " ORDER BY created_at DESC LIMIT $" + std::to_string(limit) + " OFFSET $" + std::to_string((page - 1) * limit);
    
    auto stmt = pool_->Prepare(query);
    auto rows = stmt->Execute(status.empty() ? std::nullopt : std::make_optional(status));
    
    while (rows->Next()) {
        result.push_back(ParseRow(*rows));
    }
    
    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);
    log::Debug("GetAll took {} microseconds", duration.count());
    
    return result;
}

inline std::optional<CryptoCard> CryptoCardsHandler::GetById(uint64_t id) {
    // Check cache first - O(1) lookup
    {
        std::shared_lock lock(cache_mutex_);
        auto it = card_cache_.find(id);
        if (it != card_cache_.end()) {
            return it->second;
        }
    }
    
    auto stmt = pool_->Prepare("SELECT * FROM crypto_cards WHERE id = $1");
    auto rows = stmt->Execute(id);
    
    if (rows->Next()) {
        auto card = ParseRow(*rows);
        UpdateCache(card);
        return card;
    }
    
    return std::nullopt;
}

inline CryptoCardResponse CryptoCardsHandler::Create(const CryptoCardRequest& request) {
    // Generate card number (in production, use proper Luhn algorithm)
    std::string card_number = "4532" + GenerateRandomDigits(12);
    
    auto stmt = pool_->Prepare(
        "INSERT INTO crypto_cards (user_id, card_number, currency, limit, status, card_type, created_at, updated_at) "
        "VALUES ($1, $2, $3, $4, 'pending', $5, NOW(), NOW()) RETURNING *"
    );
    
    auto rows = stmt->Execute(request.user_id, card_number, request.currency, 
                              request.limit, request.card_type);
    
    if (rows->Next()) {
        auto card = ParseRow(*rows);
        UpdateCache(card);
        return {true, "Card created successfully", card};
    }
    
    return {false, "Failed to create card", {}};
}

inline bool CryptoCardsHandler::BlockCard(uint64_t id) {
    auto stmt = pool_->Prepare("UPDATE crypto_cards SET status = 'blocked', updated_at = NOW() WHERE id = $1");
    auto result = stmt->Execute(id);
    
    if (result->AffectedRows() > 0) {
        InvalidateCache(id);
        return true;
    }
    return false;
}

inline bool CryptoCardsHandler::ActivateCard(uint64_t id) {
    auto stmt = pool_->Prepare("UPDATE crypto_cards SET status = 'active', updated_at = NOW() WHERE id = $1");
    auto result = stmt->Execute(id);
    
    if (result->AffectedRows() > 0) {
        InvalidateCache(id);
        return true;
    }
    return false;
}

inline bool CryptoCardsHandler::SetLimit(uint64_t id, double limit) {
    auto stmt = pool_->Prepare("UPDATE crypto_cards SET limit = $2, updated_at = NOW() WHERE id = $1");
    auto result = stmt->Execute(id, limit);
    
    if (result->AffectedRows() > 0) {
        InvalidateCache(id);
        return true;
    }
    return false;
}

} // namespace admin
} // namespace tiger
