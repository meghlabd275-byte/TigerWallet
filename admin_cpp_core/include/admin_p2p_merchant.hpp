/**
 * TigerWallet Admin - P2P Merchant Handler (C++ Ultra-Low Latency)
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

struct P2PMerchant {
    uint64_t id;
    std::string business_name;
    std::string email;
    std::string phone;
    std::string country;
    std::string status; // pending, approved, rejected, suspended
    bool verified;
    double total_volume;
    uint32_t transaction_count;
    double rating;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
};

struct P2PTransaction {
    uint64_t id;
    uint64_t merchant_id;
    uint64_t buyer_id;
    uint64_t seller_id;
    double amount;
    std::string currency;
    std::string status;
    std::string payment_method;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point completed_at;
};

class P2PMerchantHandler : public AdminHandler {
public:
    P2PMerchantHandler();
    ~P2PMerchantHandler() override = default;
    
    void Initialize(ConnectionPool* pool) override;
    
    // Merchant Management
    std::vector<P2PMerchant> GetMerchants(const std::string& status = "", int page = 1, int limit = 20);
    std::optional<P2PMerchant> GetMerchantById(uint64_t id);
    bool ApproveMerchant(uint64_t id);
    bool RejectMerchant(uint64_t id, const std::string& reason);
    bool SuspendMerchant(uint64_t id);
    
    // Transactions
    std::vector<P2PTransaction> GetTransactions(uint64_t merchant_id, int page = 1, int limit = 20);
    
    // Statistics
    MerchantStats GetStats();

private:
    ConnectionPool* pool_;
    void PrepareStatements();
    P2PMerchant ParseMerchantRow(const Row& row);
    P2PTransaction ParseTransactionRow(const Row& row);
};

inline std::vector<P2PMerchant> P2PMerchantHandler::GetMerchants(const std::string& status, int page, int limit) {
    std::vector<P2PMerchant> result;
    std::string query = "SELECT * FROM p2p_merchants";
    
    if (!status.empty() && status != "all") {
        query += " WHERE status = $1";
    }
    query += " ORDER BY created_at DESC LIMIT $" + std::to_string(limit) + " OFFSET $" + std::to_string((page - 1) * limit);
    
    auto stmt = pool_->Prepare(query);
    auto rows = (!status.empty() && status != "all") ? stmt->Execute(status) : stmt->Execute();
    
    while (rows->Next()) {
        result.push_back(ParseMerchantRow(*rows));
    }
    
    return result;
}

inline bool P2PMerchantHandler::ApproveMerchant(uint64_t id) {
    auto stmt = pool_->Prepare(
        "UPDATE p2p_merchants SET status = 'approved', verified = true, updated_at = NOW() WHERE id = $1"
    );
    auto result = stmt->Execute(id);
    return result->AffectedRows() > 0;
}

inline bool P2PMerchantHandler::RejectMerchant(uint64_t id, const std::string& reason) {
    auto stmt = pool_->Prepare(
        "UPDATE p2p_merchants SET status = 'rejected', updated_at = NOW() WHERE id = $1"
    );
    auto result = stmt->Execute(id);
    return result->AffectedRows() > 0;
}

inline std::vector<P2PTransaction> P2PMerchantHandler::GetTransactions(uint64_t merchant_id, int page, int limit) {
    std::vector<P2PTransaction> result;
    
    auto stmt = pool_->Prepare(
        "SELECT * FROM p2p_transactions WHERE merchant_id = $1 "
        "ORDER BY created_at DESC LIMIT $" + std::to_string(limit) + " OFFSET $" + std::to_string((page - 1) * limit)
    );
    auto rows = stmt->Execute(merchant_id);
    
    while (rows->Next()) {
        result.push_back(ParseTransactionRow(*rows));
    }
    
    return result;
}

} // namespace admin
} // namespace tiger
