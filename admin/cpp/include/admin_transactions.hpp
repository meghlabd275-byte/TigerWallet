/**
 * TigerAdmin C++ Core - Transactions Header
 */
#pragma once

#include "admin_security.hpp"
#include "admin_kyc.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>

namespace tiger {
namespace admin {

using TransactionID = uint64_t;
using WithdrawalID = uint64_t;

enum class TransactionType {
    DEPOSIT = 0,
    WITHDRAWAL = 1,
    TRADE = 2,
    TRANSFER = 3,
    FEE = 4,
    REFUND = 5,
    STAKE = 6,
    UNSTAKE = 7
};

enum class TransactionStatus {
    PENDING = 0,
    CONFIRMED = 1,
    COMPLETED = 2,
    FAILED = 3,
    CANCELLED = 4,
    FLAGGED = 5
};

enum class WithdrawalStatus {
    PENDING = 0,
    APPROVED = 1,
    REJECTED = 2,
    PROCESSING = 3,
    COMPLETED = 4,
    FAILED = 5,
    CANCELLED = 6
};

struct Transaction {
    TransactionID id = 0;
    std::string tx_hash;
    UserID user_id = 0;
    TransactionType type = TransactionType::DEPOSIT;
    TransactionStatus status = TransactionStatus::PENDING;
    std::string from_address;
    std::string to_address;
    double amount = 0.0;
    std::string currency;
    double fee = 0.0;
    std::string blockchain;
    int confirmations = 0;
    bool is_flagged = false;
    std::string flag_reason;
    int64_t created_at = 0;
    int64_t updated_at = 0;
};

struct Withdrawal {
    WithdrawalID id = 0;
    UserID user_id = 0;
    std::string to_address;
    double amount = 0.0;
    std::string currency;
    std::string blockchain;
    double fee = 0.0;
    WithdrawalStatus status = WithdrawalStatus::PENDING;
    std::string tx_hash;
    std::string rejection_reason;
    AdminID approved_by = 0;
    int64_t created_at = 0;
    int64_t processed_at = 0;
};

class TransactionService {
public:
    struct TransactionListParams {
        int page = 1;
        int page_size = 20;
        std::optional<TransactionType> type;
        std::optional<TransactionStatus> status;
        std::optional<UserID> user_id;
        std::optional<std::string> currency;
        std::optional<std::string> start_date;
        std::optional<std::string> end_date;
        std::optional<bool> flagged_only;
    };

    struct TransactionListResult {
        int page = 1;
        int page_size = 20;
        int64_t total = 0;
        std::vector<Transaction> transactions;
    };

    struct TransactionStats {
        int64_t total = 0;
        int64_t pending = 0;
        int64_t completed = 0;
        int64_t failed = 0;
        int64_t flagged = 0;
        double total_volume = 0.0;
    };

    static TransactionService& instance();

    void initialize();

    TransactionListResult list_transactions(const TransactionListParams& params);
    std::optional<Transaction> get_transaction(TransactionID id);
    std::optional<Transaction> get_transaction_by_hash(const std::string& tx_hash);

    bool flag_transaction(TransactionID id, const std::string& reason,
                          AdminID admin_id);
    bool unflag_transaction(TransactionID id, AdminID admin_id);

    TransactionStats get_transaction_stats(const std::string& start_date,
                                          const std::string& end_date);
    std::string export_transactions(const TransactionListParams& params,
                                   const std::string& format);
};

class WithdrawalService {
public:
    struct WithdrawalListParams {
        int page = 1;
        int page_size = 20;
        std::optional<WithdrawalStatus> status;
        std::optional<UserID> user_id;
        std::optional<std::string> currency;
        std::optional<std::string> start_date;
        std::optional<std::string> end_date;
    };

    struct WithdrawalListResult {
        int page = 1;
        int page_size = 20;
        int64_t total = 0;
        std::vector<Withdrawal> withdrawals;
    };

    struct ApproveResult {
        bool success = false;
        std::string message;
        std::string tx_hash;
    };

    struct RejectResult {
        bool success = false;
        std::string message;
    };

    struct WithdrawalStats {
        int64_t total = 0;
        int64_t pending = 0;
        int64_t approved = 0;
        int64_t rejected = 0;
        int64_t completed = 0;
        int64_t failed = 0;
        double total_amount = 0.0;
    };

    static WithdrawalService& instance();

    void initialize();

    WithdrawalListResult list_withdrawals(const WithdrawalListParams& params);
    std::optional<Withdrawal> get_withdrawal(WithdrawalID id);

    ApproveResult approve_withdrawal(WithdrawalID id, AdminID admin_id);
    RejectResult reject_withdrawal(WithdrawalID id, AdminID admin_id,
                                   const std::string& reason);
    bool process_withdrawal(WithdrawalID id, AdminID admin_id);
    int bulk_approve(const std::vector<WithdrawalID>& ids, AdminID admin_id);
    int bulk_reject(const std::vector<WithdrawalID>& ids, AdminID admin_id,
                    const std::string& reason);

    WithdrawalStats get_withdrawal_stats(const std::string& start_date,
                                         const std::string& end_date);
    bool validate_withdrawal(const Withdrawal& withdrawal);
    std::string broadcast_transaction(const Withdrawal& withdrawal);
};

} // namespace admin
} // namespace tiger
