/**
 * TigerAdmin C++ Core - Transaction & Withdrawal Handler
 */

#ifndef TIGER_ADMIN_TRANSACTIONS_HPP
#define TIGER_ADMIN_TRANSACTIONS_HPP

#include <string>
#include <vector>
#include <optional>
#include "admin_models.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Transaction Service
// ============================================================================

class TransactionService {
public:
    static TransactionService& instance();
    
    void initialize();
    
    // List transactions
    struct TransactionListParams {
        std::optional<TransactionStatus> status;
        std::optional<UserID> user_id;
        std::optional<int> chain_id;
        std::string search;
        std::string start_date;
        std::string end_date;
        int page = 1;
        int page_size = 20;
    };
    
    struct TransactionListResult {
        std::vector<Transaction> transactions;
        int64_t total;
        int page;
        int page_size;
    };
    
    TransactionListResult list_transactions(const TransactionListParams& params);
    
    // Get single transaction
    std::optional<Transaction> get_transaction(TransactionID id);
    std::optional<Transaction> get_transaction_by_hash(const std::string& tx_hash);
    
    // Flag/unflag
    bool flag_transaction(TransactionID id, const std::string& reason,
                         AdminID admin_id);
    bool unflag_transaction(TransactionID id, AdminID admin_id);
    
    // Stats
    struct TransactionStats {
        int64_t total;
        int64_t pending;
        int64_t confirmed;
        int64_t failed;
        int64_t flagged;
        double total_volume;
    };
    
    TransactionStats get_transaction_stats(const std::string& start_date,
                                           const std::string& end_date);
    
    // Export
    std::string export_transactions(const TransactionListParams& params,
                                    const std::string& format);
    
private:
    TransactionService() = default;
};

// ============================================================================
// Withdrawal Service
// ============================================================================

class WithdrawalService {
public:
    static WithdrawalService& instance();
    
    void initialize();
    
    // List withdrawals
    struct WithdrawalListParams {
        std::optional<WithdrawalStatus> status;
        std::optional<UserID> user_id;
        std::string search;
        std::string start_date;
        std::string end_date;
        int page = 1;
        int page_size = 20;
    };
    
    struct WithdrawalListResult {
        std::vector<Withdrawal> withdrawals;
        int64_t total;
        int page;
        int page_size;
    };
    
    WithdrawalListResult list_withdrawals(const WithdrawalListParams& params);
    
    // Get single withdrawal
    std::optional<Withdrawal> get_withdrawal(WithdrawalID id);
    
    // Process withdrawal
    struct ApproveResult {
        bool success;
        std::string message;
        std::string tx_hash;
    };
    
    struct RejectResult {
        bool success;
        std::string message;
    };
    
    ApproveResult approve_withdrawal(WithdrawalID id, AdminID admin_id);
    RejectResult reject_withdrawal(WithdrawalID id, AdminID admin_id,
                                   const std::string& reason);
    
    // Process (execute on blockchain)
    bool process_withdrawal(WithdrawalID id, AdminID admin_id);
    
    // Bulk operations
    int bulk_approve(const std::vector<WithdrawalID>& ids, AdminID admin_id);
    int bulk_reject(const std::vector<WithdrawalID>& ids, AdminID admin_id,
                   const std::string& reason);
    
    // Stats
    struct WithdrawalStats {
        int64_t total;
        int64_t pending;
        int64_t approved;
        int64_t rejected;
        int64_t processing;
        int64_t completed;
        double total_amount;
    };
    
    WithdrawalStats get_withdrawal_stats(const std::string& start_date,
                                          const std::string& end_date);
    
private:
    WithdrawalService() = default;
    
    bool validate_withdrawal(const Withdrawal& withdrawal);
    std::string broadcast_transaction(const Withdrawal& withdrawal);
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_TRANSACTIONS_HPP
