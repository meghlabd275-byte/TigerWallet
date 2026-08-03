#pragma once

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <memory>
#include <functional>
#include <chrono>
#include <mutex>
#include <shared_mutex>
#include <atomic>

namespace tiger {

// Forward declarations
class Database;
class RedisClient;
class SessionManager;

namespace models {
struct Admin;
struct User;
struct KYCRecord;
struct Token;
struct TradingPair;
struct Transaction;
struct WhiteLabel;
struct FeeStructure;
struct Blockchain;
struct MarketMakerBot;
struct Withdrawal;
struct AuditLog;
}

namespace admin {

// Permission types
enum class Permission {
    // User management
    USERS_READ,
    USERS_WRITE,
    USERS_DELETE,
    USERS_BAN,
    
    // Admin management
    ADMINS_READ,
    ADMINS_WRITE,
    ADMINS_DELETE,
    
    // KYC
    KYC_READ,
    KYC_WRITE,
    KYC_APPROVE,
    KYC_REJECT,
    
    // Wallets
    WALLETS_READ,
    WALLETS_WRITE,
    
    // Transactions
    TRANSACTIONS_READ,
    TRANSACTIONS_WRITE,
    TRANSACTIONS_CANCEL,
    
    // Pairs
    PAIRS_READ,
    PAIRS_WRITE,
    PAIRS_DELETE,
    
    // Liquidity
    LIQUIDITY_READ,
    LIQUIDITY_WRITE,
    LIQUIDITY_DELETE,
    
    // Fees
    FEES_READ,
    FEES_WRITE,
    
    // White label
    WHITELABEL_READ,
    WHITELABEL_WRITE,
    WHITELABEL_DELETE,
    
    // Withdrawals
    WITHDRAWALS_READ,
    WITHDRAWALS_WRITE,
    WITHDRAWALS_APPROVE,
    WITHDRAWALS_REJECT,
    
    // API
    API_READ,
    API_WRITE,
    API_DELETE,
    
    // Analytics
    ANALYTICS_READ,
    
    // Settings
    SETTINGS_READ,
    SETTINGS_WRITE,
    
    // NFT
    NFT_READ,
    NFT_WRITE,
    
    // Tokens
    TOKENS_READ,
    TOKENS_WRITE,
    TOKENS_DELETE,
    
    // Multisend
    MULTISEND_READ,
    MULTISEND_WRITE
};

// Admin roles
enum class AdminRole {
    SUPER_ADMIN,
    ADMIN,
    MANAGER,
    SUPPORT,
    ANALYST,
    MODERATOR
};

// Admin status
enum class AdminStatus {
    ACTIVE,
    SUSPENDED,
    INACTIVE,
    PENDING
};

// KYC status
enum class KYCStatus {
    NONE,
    PENDING,
    LEVEL1,
    LEVEL2,
    LEVEL3,
    REJECTED
};

// User status
enum class UserStatus {
    ACTIVE,
    SUSPENDED,
    BANNED,
    PENDING
};

// Token status
enum class TokenStatus {
    ACTIVE,
    INACTIVE,
    SUSPENDED,
    DELISTED
};

// Pair status
enum class PairStatus {
    ACTIVE,
    SUSPENDED,
    HALTED,
    DELISTED
};

// White label status
enum class WhiteLabelStatus {
    ACTIVE,
    SUSPENDED,
    PENDING,
    EXPIRED,
    CANCELLED
};

// Withdrawal status
enum class WithdrawalStatus {
    PENDING,
    PROCESSING,
    COMPLETED,
    FAILED,
    CANCELLED,
    REJECTED
};

// Transaction status
enum class TransactionStatus {
    PENDING,
    PROCESSING,
    COMPLETED,
    FAILED,
    CANCELLED
};

// Transaction type
enum class TransactionType {
    DEPOSIT,
    WITHDRAWAL,
    TRANSFER,
    SWAP,
    TRADE,
    FEE,
    REWARD,
    INTERNAL
};

// Pagination
struct Pagination {
    int page = 1;
    int limit = 20;
    std::string sort_by;
    bool descending = true;
};

// Filter options
struct UserFilter {
    std::optional<std::string> status;
    std::optional<std::string> kyc_status;
    std::optional<std::string> search;
    std::optional<std::string> white_label_id;
    std::optional<std::string> date_from;
    std::optional<std::string> date_to;
};

struct TransactionFilter {
    std::optional<std::string> status;
    std::optional<std::string> type;
    std::optional<std::string> user_id;
    std::optional<std::string> token;
    std::optional<std::string> chain;
    std::optional<std::string> date_from;
    std::optional<std::string> date_to;
    std::optional<double> min_amount;
    std::optional<double> max_amount;
};

struct TokenFilter {
    std::optional<std::string> status;
    std::optional<std::string> chain;
    std::optional<std::string> search;
    std::optional<bool> verified;
};

struct PairFilter {
    std::optional<std::string> status;
    std::optional<std::string> base_asset;
    std::optional<std::string> quote_asset;
    std::optional<std::string> chain;
};

struct KYCFilter {
    std::optional<std::string> status;
    std::optional<int> level;
    std::optional<std::string> date_from;
    std::optional<std::string> date_to;
};

struct WithdrawalFilter {
    std::optional<std::string> status;
    std::optional<std::string> user_id;
    std::optional<std::string> token;
    std::optional<std::string> chain;
    std::optional<std::string> date_from;
    std::optional<std::string> date_to;
};

// Result types
template<typename T>
struct Result {
    bool success;
    std::optional<T> data;
    std::string error;
    int status_code;
    
    static Result<T> Ok(T value) {
        return {true, value, "", 200};
    }
    
    static Result<T> Error(const std::string& err, int code = 400) {
        return {false, std::nullopt, err, code};
    }
};

template<typename T>
struct PaginatedResult {
    std::vector<T> items;
    int total;
    int page;
    int limit;
    int total_pages;
};

// Response helpers
using Json = nlohmann::json;

Json success_response(const Json& data);
Json error_response(const std::string& message, int code = 400);
Json paginated_response(const Json& items, int total, int page, int limit);

// Admin Service interface
class AdminService {
public:
    AdminService(std::shared_ptr<Database> db, 
                std::shared_ptr<RedisClient> redis,
                std::shared_ptr<SessionManager> session);
    ~AdminService();
    
    // Admin management
    Result<models::Admin> create_admin(const Json& data);
    Result<models::Admin> get_admin(const std::string& id);
    Result<models::Admin> update_admin(const std::string& id, const Json& data);
    Result<bool> delete_admin(const std::string& id);
    PaginatedResult<models::Admin> list_admins(const Pagination& pagination, const std::string& filter = "");
    
    // Permission management
    Result<bool> update_admin_permissions(const std::string& id, const std::vector<Permission>& permissions);
    Result<bool> has_permission(const std::string& admin_id, Permission permission);
    
    // Authentication
    Result<Json> authenticate(const std::string& email, const std::string& password, const std::string& ip = "");
    Result<Json> authenticate_2fa(const std::string& admin_id, const std::string& code);
    Result<bool> logout(const std::string& admin_id, const std::string& token);
    Result<Json> refresh_token(const std::string& token);
    
    // Session management
    Result<bool> validate_session(const std::string& admin_id, const std::string& token);
    Result<std::vector<Json>> get_active_sessions(const std::string& admin_id);
    Result<bool> revoke_session(const std::string& admin_id, const std::string& session_id);
    Result<bool> revoke_all_sessions(const std::string& admin_id);
    
    // 2FA management
    Result<Json> enable_2fa(const std::string& admin_id);
    Result<bool> disable_2fa(const std::string& admin_id, const std::string& code);
    Result<bool> verify_2fa(const std::string& admin_id, const std::string& code);
    
    // IP whitelist
    Result<bool> add_ip_whitelist(const std::string& admin_id, const std::string& cidr, const std::string& description);
    Result<bool> remove_ip_whitelist(const std::string& admin_id, const std::string& id);
    Result<std::vector<Json>> get_ip_whitelist(const std::string& admin_id);
    Result<bool> check_ip_allowed(const std::string& admin_id, const std::string& ip);
    
    // Audit logs
    Result<bool> log_action(const std::string& admin_id, const std::string& action, 
                           const std::string& resource, const std::string& resource_id,
                           const Json& details = Json::object());
    PaginatedResult<Json> get_audit_logs(const std::string& admin_id, 
                                         const Pagination& pagination,
                                         const std::string& action = "",
                                         const std::string& resource = "");
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::shared_ptr<SessionManager> session_;
    std::mutex mutex_;
    
    std::string hash_password(const std::string& password);
    bool verify_password(const std::string& password, const std::string& hash);
    std::string generate_token();
    std::string generate_2fa_secret();
};

// User Service interface
class UserService {
public:
    UserService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~UserService();
    
    Result<models::User> create_user(const Json& data);
    Result<models::User> get_user(const std::string& id);
    Result<models::User> get_user_by_email(const std::string& email);
    Result<models::User> get_user_by_wallet(const std::string& wallet_address);
    Result<models::User> update_user(const std::string& id, const Json& data);
    Result<bool> delete_user(const std::string& id);
    PaginatedResult<models::User> list_users(const Pagination& pagination, const UserFilter& filter = {});
    
    Result<bool> suspend_user(const std::string& id, const std::string& reason);
    Result<bool> ban_user(const std::string& id, const std::string& reason);
    Result<bool> activate_user(const std::string& id);
    Result<bool> reset_user_2fa(const std::string& id);
    
    Result<Json> get_user_transactions(const std::string& user_id, const Pagination& pagination);
    Result<Json> get_user_wallets(const std::string& user_id);
    Result<Json> get_user_kyc(const std::string& user_id);
    
    Result<bool> verify_email(const std::string& user_id);
    Result<bool> verify_phone(const std::string& user_id);
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

// KYC Service interface
class KYCService {
public:
    KYCService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~KYCService();
    
    Result<models::KYCRecord> create_kyc(const Json& data);
    Result<models::KYCRecord> get_kyc(const std::string& id);
    Result<models::KYCRecord> get_user_kyc(const std::string& user_id);
    Result<models::KYCRecord> update_kyc(const std::string& id, const Json& data);
    
    PaginatedResult<models::KYCRecord> list_kyc(const Pagination& pagination, const KYCFilter& filter = {});
    
    Result<bool> approve_kyc(const std::string& id, const std::string& admin_id, const std::string& notes = "");
    Result<bool> reject_kyc(const std::string& id, const std::string& admin_id, const std::string& reason);
    Result<bool> request_info(const std::string& id, const std::string& admin_id, const std::string& message);
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

// Token Service interface
class TokenService {
public:
    TokenService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~TokenService();
    
    Result<models::Token> create_token(const Json& data);
    Result<models::Token> get_token(const std::string& id);
    Result<models::Token> get_token_by_address(const std::string& address, const std::string& chain);
    Result<models::Token> update_token(const std::string& id, const Json& data);
    Result<bool> delete_token(const std::string& id);
    PaginatedResult<models::Token> list_tokens(const Pagination& pagination, const TokenFilter& filter = {});
    
    Result<bool> verify_token(const std::string& id);
    Result<bool> unverify_token(const std::string& id);
    Result<bool> suspend_token(const std::string& id);
    Result<bool> activate_token(const std::string& id);
    
    Result<Json> get_token_analytics(const std::string& id, const std::string& period = "24h");
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

// Pair Service interface
class PairService {
public:
    PairService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~PairService();
    
    Result<models::TradingPair> create_pair(const Json& data);
    Result<models::TradingPair> get_pair(const std::string& id);
    Result<models::TradingPair> get_pair_by_symbol(const std::string& symbol);
    Result<models::TradingPair> update_pair(const std::string& id, const Json& data);
    Result<bool> delete_pair(const std::string& id);
    PaginatedResult<models::TradingPair> list_pairs(const Pagination& pagination, const PairFilter& filter = {});
    
    Result<bool> enable_trading(const std::string& id);
    Result<bool> disable_trading(const std::string& id);
    Result<bool> update_pair_fees(const std::string& id, double maker_fee, double taker_fee);
    
    Result<Json> get_pair_liquidity(const std::string& id);
    Result<Json> get_pair_analytics(const std::string& id, const std::string& period = "24h");
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

// Fee Service interface
class FeeService {
public:
    FeeService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~FeeService();
    
    Result<models::FeeStructure> create_fee(const Json& data);
    Result<models::FeeStructure> get_fee(const std::string& id);
    Result<models::FeeStructure> update_fee(const std::string& id, const Json& data);
    Result<bool> delete_fee(const std::string& id);
    std::vector<models::FeeStructure> list_fees(const std::string& type = "", const std::string& chain_id = "");
    
    Result<double> calculate_fee(const std::string& type, double amount, const std::string& chain_id = "", const std::string& token_id = "");
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

// Chain Service interface
class ChainService {
public:
    ChainService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~ChainService();
    
    Result<models::Blockchain> create_chain(const Json& data);
    Result<models::Blockchain> get_chain(const std::string& id);
    Result<models::Blockchain> get_chain_by_chain_id(int64_t chain_id);
    Result<models::Blockchain> update_chain(const std::string& id, const Json& data);
    Result<bool> delete_chain(const std::string& id);
    std::vector<models::Blockchain> list_chains(bool include_testnet = false);
    
    Result<bool> enable_chain(const std::string& id);
    Result<bool> disable_chain(const std::string& id);
    Result<bool> set_maintenance(const std::string& id, bool maintenance);
    
    Result<Json> get_chain_stats(const std::string& id);
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

// Wallet Service interface
class WalletService {
public:
    WalletService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~WalletService();
    
    Result<Json> get_master_wallet(const std::string& id);
    Result<Json> get_all_master_wallets(const Pagination& pagination);
    Result<Json> create_master_wallet(const Json& data);
    Result<Json> update_master_wallet(const std::string& id, const Json& data);
    
    Result<Json> get_wallet_balance(const std::string& wallet_id);
    Result<Json> get_wallet_transactions(const std::string& wallet_id, const Pagination& pagination);
    Result<Json> get_wallet_analytics(const std::string& wallet_id, const std::string& period = "24h");
    
    Result<bool> cold_wallet_transfer(const std::string& from_wallet, const std::string& to_address, 
                                      const std::string& amount, const std::string& token);
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

// White Label Service interface
class WhiteLabelService {
public:
    WhiteLabelService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~WhiteLabelService();
    
    Result<models::WhiteLabel> create_whitelabel(const Json& data);
    Result<models::WhiteLabel> get_whitelabel(const std::string& id);
    Result<models::WhiteLabel> get_whitelabel_by_domain(const std::string& domain);
    Result<models::WhiteLabel> update_whitelabel(const std::string& id, const Json& data);
    Result<bool> delete_whitelabel(const std::string& id);
    PaginatedResult<models::WhiteLabel> list_whitelabels(const Pagination& pagination, const std::string& status = "");
    
    Result<bool> approve_whitelabel(const std::string& id, const std::string& admin_id);
    Result<bool> suspend_whitelabel(const std::string& id, const std::string& reason);
    Result<bool> resume_whitelabel(const std::string& id);
    Result<bool> update_branding(const std::string& id, const Json& branding);
    Result<bool> update_fee(const std::string& id, double fee_percent);
    
    Result<Json> get_whitelabel_users(const std::string& whitelabel_id, const Pagination& pagination);
    Result<Json> get_whitelabel_analytics(const std::string& whitelabel_id, const std::string& period = "24h");
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

// Analytics Service interface
class AnalyticsService {
public:
    AnalyticsService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~AnalyticsService();
    
    Result<Json> get_dashboard_stats();
    Result<Json> get_user_analytics(const std::string& period = "24h");
    Result<Json> get_transaction_analytics(const std::string& period = "24h");
    Result<Json> get_revenue_analytics(const std::string& period = "24h");
    Result<Json> get_volume_analytics(const std::string& period = "24h");
    
    Result<Json> get_realtime_stats();
    
    Result<Json> export_report(const std::string& type, const std::string& format, 
                              const std::string& date_from, const std::string& date_to);
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
    
    std::map<std::string, std::string> get_analytics_query(const std::string& period);
};

// Withdrawal Service interface
class WithdrawalService {
public:
    WithdrawalService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~WithdrawalService();
    
    Result<models::Withdrawal> get_withdrawal(const std::string& id);
    PaginatedResult<models::Withdrawal> list_withdrawals(const Pagination& pagination, const WithdrawalFilter& filter = {});
    
    Result<bool> approve_withdrawal(const std::string& id, const std::string& admin_id);
    Result<bool> reject_withdrawal(const std::string& id, const std::string& admin_id, const std::string& reason);
    Result<bool> process_withdrawal(const std::string& id);
    
    Result<Json> get_withdrawal_analytics(const std::string& period = "24h");
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

// Transaction Service interface
class TransactionService {
public:
    TransactionService(std::shared_ptr<Database> db, std::shared_ptr<RedisClient> redis);
    ~TransactionService();
    
    Result<models::Transaction> get_transaction(const std::string& id);
    Result<models::Transaction> get_transaction_by_hash(const std::string& hash);
    PaginatedResult<models::Transaction> list_transactions(const Pagination& pagination, const TransactionFilter& filter = {});
    
    Result<bool> cancel_transaction(const std::string& id, const std::string& admin_id);
    Result<bool> flag_transaction(const std::string& id, const std::string& reason);
    
    Result<Json> get_transaction_analytics(const std::string& period = "24h");
    
private:
    std::shared_ptr<Database> db_;
    std::shared_ptr<RedisClient> redis_;
    std::mutex mutex_;
};

} // namespace admin
} // namespace tiger
