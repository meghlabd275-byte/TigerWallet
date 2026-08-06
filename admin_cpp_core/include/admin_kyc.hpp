/**
 * TigerAdmin C++ Core - KYC Handler
 * Know Your Customer processing
 */

#ifndef TIGER_ADMIN_KYC_HPP
#define TIGER_ADMIN_KYC_HPP

#include <string>
#include <vector>
#include <optional>
#include "admin_models.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// KYC Service
// ============================================================================

class KYCService {
public:
    static KYCService& instance();
    
    void initialize();
    
    // List KYC requests
    struct KYCListParams {
        std::optional<KYCStatus> status;
        std::optional<int> level;
        std::string search;
        int page = 1;
        int page_size = 20;
    };
    
    struct KYCListResult {
        std::vector<KYCRequest> requests;
        int64_t total;
        int page;
        int page_size;
    };
    
    KYCListResult list_kyc_requests(const KYCListParams& params);
    
    // Get single request
    std::optional<KYCRequest> get_kyc_request(KYCRequestID id);
    std::optional<KYCRequest> get_user_kyc(UserID user_id);
    
    // Process KYC
    struct ApproveResult {
        bool success;
        std::string message;
    };
    
    struct RejectResult {
        bool success;
        std::string message;
    };
    
    ApproveResult approve_kyc(KYCRequestID id, AdminID admin_id);
    RejectResult reject_kyc(KYCRequestID id, AdminID admin_id, 
                           const std::string& reason);
    
    // Bulk operations
    int bulk_approve(const std::vector<KYCRequestID>& ids, AdminID admin_id);
    int bulk_reject(const std::vector<KYCRequestID>& ids, AdminID admin_id,
                    const std::string& reason);
    
    // Stats
    struct KYCStats {
        int64_t pending;
        int64_t approved;
        int64_t rejected;
        int64_t total;
    };
    
    KYCStats get_kyc_stats();
    
    // Auto-review (for low-risk cases)
    bool auto_review(KYCRequestID id);
    
private:
    KYCService() = default;
    
    bool validate_documents(const KYCRequest& request);
    bool check_fraud Indicators(const KYCRequest& request);
    void send_notification(UserID user_id, KYCStatus status);
};

// ============================================================================
// User Service
// ============================================================================

class UserService {
public:
    static UserService& instance();
    
    void initialize();
    
    // List users
    struct UserListParams {
        std::optional<UserStatus> status;
        std::optional<KYCStatus> kyc_status;
        std::string search;
        std::optional<WhiteLabelID> white_label_id;
        int page = 1;
        int page_size = 20;
    };
    
    struct UserListResult {
        std::vector<User> users;
        int64_t total;
        int page;
        int page_size;
    };
    
    UserListResult list_users(const UserListParams& params);
    
    // Get user
    std::optional<User> get_user(UserID id);
    std::optional<User> get_user_by_email(const std::string& email);
    std::optional<User> get_user_by_wallet(const std::string& wallet_address);
    
    // Update user
    bool update_user_status(UserID id, UserStatus status);
    bool suspend_user(UserID id, const std::string& reason);
    bool ban_user(UserID id, const std::string& reason);
    bool unban_user(UserID id);
    
    // User actions
    bool verify_email(UserID id);
    bool verify_phone(UserID id);
    bool update_kyc_level(UserID id, int level);
    
    // Bulk operations
    int bulk_suspend(const std::vector<UserID>& ids, 
                     const std::string& reason);
    int bulk_ban(const std::vector<UserID>& ids, 
                 const std::string& reason);
    
    // Stats
    struct UserStats {
        int64_t total;
        int64_t active;
        int64_t suspended;
        int64_t banned;
    };
    
    UserStats get_user_stats();
    
private:
    UserService() = default;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_KYC_HPP
