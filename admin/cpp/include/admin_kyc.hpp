/**
 * TigerAdmin C++ Core - KYC & User Header
 */
#pragma once

#include "admin_security.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>

namespace tiger {
namespace admin {

using UserID = uint64_t;
using KYCRequestID = uint64_t;

enum class KYCStatus {
    PENDING = 0,
    APPROVED = 1,
    REJECTED = 2,
    EXPIRED = 3,
    IN_REVIEW = 4
};

enum class UserStatus {
    ACTIVE = 0,
    SUSPENDED = 1,
    BANNED = 2,
    PENDING_VERIFICATION = 3
};

struct KYCRequest {
    KYCRequestID id = 0;
    UserID user_id = 0;
    std::string full_name;
    std::string date_of_birth;
    std::string nationality;
    std::string id_type;
    std::string id_number;
    std::string id_front_image;
    std::string id_back_image;
    std::string selfie_image;
    std::string proof_of_address;
    KYCStatus status = KYCStatus::PENDING;
    std::string rejection_reason;
    AdminID reviewed_by = 0;
    int64_t submitted_at = 0;
    int64_t reviewed_at = 0;
};

struct User {
    UserID id = 0;
    std::string email;
    std::string username;
    std::string phone;
    std::string country;
    UserStatus status = UserStatus::ACTIVE;
    bool is_verified = false;
    bool kyc_verified = false;
    bool two_factor_enabled = false;
    int64_t created_at = 0;
    int64_t last_login = 0;
};

class KYCService {
public:
    struct KYCListParams {
        int page = 1;
        int page_size = 20;
        std::optional<KYCStatus> status;
        std::optional<std::string> start_date;
        std::optional<std::string> end_date;
    };

    struct KYCListResult {
        int page = 1;
        int page_size = 20;
        int64_t total = 0;
        std::vector<KYCRequest> requests;
    };

    struct ApproveResult {
        bool success = false;
        std::string message;
    };

    struct RejectResult {
        bool success = false;
        std::string message;
    };

    struct KYCStats {
        int64_t total = 0;
        int64_t pending = 0;
        int64_t approved = 0;
        int64_t rejected = 0;
    };

    static KYCService& instance();

    void initialize();

    KYCListResult list_kyc_requests(const KYCListParams& params);
    std::optional<KYCRequest> get_kyc_request(KYCRequestID id);
    std::optional<KYCRequest> get_user_kyc(UserID user_id);

    ApproveResult approve_kyc(KYCRequestID id, AdminID admin_id);
    RejectResult reject_kyc(KYCRequestID id, AdminID admin_id,
                            const std::string& reason);
    int bulk_approve(const std::vector<KYCRequestID>& ids, AdminID admin_id);
    int bulk_reject(const std::vector<KYCRequestID>& ids, AdminID admin_id,
                    const std::string& reason);

    KYCStats get_kyc_stats();
    bool auto_review(KYCRequestID id);
    bool validate_documents(const KYCRequest& request);
    bool check_fraud_indicators(const KYCRequest& request);
    void send_notification(UserID user_id, KYCStatus status);
};

class UserService {
public:
    struct UserListParams {
        int page = 1;
        int page_size = 20;
        std::optional<UserStatus> status;
        std::optional<std::string> search;
        std::optional<std::string> country;
        std::optional<bool> kyc_verified;
    };

    struct UserListResult {
        int page = 1;
        int page_size = 20;
        int64_t total = 0;
        std::vector<User> users;
    };

    struct UserStats {
        int64_t total = 0;
        int64_t active = 0;
        int64_t suspended = 0;
        int64_t banned = 0;
    };

    static UserService& instance();

    void initialize();

    UserListResult list_users(const UserListParams& params);
    std::optional<User> get_user(UserID id);
    std::optional<User> get_user_by_email(const std::string& email);
    std::optional<User> get_user_by_wallet(const std::string& wallet_address);

    bool update_user_status(UserID id, UserStatus status);
    bool suspend_user(UserID id, const std::string& reason);
    bool ban_user(UserID id, const std::string& reason);
    bool unban_user(UserID id);
    bool verify_email(UserID id);
    bool verify_phone(UserID id);
    bool update_kyc_level(UserID id, int level);

    int bulk_suspend(const std::vector<UserID>& ids, const std::string& reason);
    int bulk_ban(const std::vector<UserID>& ids, const std::string& reason);

    UserStats get_user_stats();
};

} // namespace admin
} // namespace tiger
