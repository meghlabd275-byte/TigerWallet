/**
 * TigerAdmin C++ Core - Data Models
 * Ultra-low latency admin operations
 */

#ifndef TIGER_ADMIN_MODELS_HPP
#define TIGER_ADMIN_MODELS_HPP

#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <optional>
#include <chrono>
#include <variant>
#include <memory>
#include <shared_mutex>
#include <atomic>

namespace tiger {
namespace admin {

// ============================================================================
// Core Types
// ============================================================================

using AdminID = uint64_t;
using UserID = uint64_t;
using TransactionID = uint64_t;
using TokenID = uint64_t;
using PairID = uint64_t;
using BlockchainID = uint64_t;
using KYCRequestID = uint64_t;
using WithdrawalID = uint64_t;
using TicketID = uint64_t;
using WhiteLabelID = uint64_t;

using JSON = std::string;
using Timestamp = int64_t;
using UnixTime = std::chrono::seconds;

// ============================================================================
// Enums
// ============================================================================

enum class AdminRole {
    SUPER_ADMIN,
    ADMIN,
    SUPPORT,
    ANALYST,
    VIEWER,
    WHITE_LABEL_ADMIN,
    MASTER_ADMIN
};

enum class UserStatus {
    ACTIVE,
    SUSPENDED,
    BANNED,
    PENDING
};

enum class KYCStatus {
    NONE,
    PENDING,
    APPROVED,
    REJECTED
};

enum class TransactionStatus {
    PENDING,
    CONFIRMED,
    FAILED,
    FLAGGED
};

enum class WithdrawalStatus {
    PENDING,
    APPROVED,
    REJECTED,
    PROCESSING,
    COMPLETED,
    FAILED
};

enum class TokenStatus {
    ACTIVE,
    INACTIVE,
    SUSPENDED,
    DELETED
};

enum class PairStatus {
    ACTIVE,
    HALTED,
    SUSPENDED
};

enum class BlockchainStatus {
    ACTIVE,
    INACTIVE,
    MAINTENANCE
};

enum class TicketStatus {
    OPEN,
    IN_PROGRESS,
    RESOLVED,
    CLOSED
};

enum class TicketPriority {
    LOW,
    MEDIUM,
    HIGH,
    URGENT
};

enum class WhiteLabelStatus {
    ACTIVE,
    SUSPENDED,
    PENDING,
    TERMINATED
};

// ============================================================================
// Data Models
// ============================================================================

struct Admin {
    AdminID id;
    std::string username;
    std::string email;
    std::string password_hash;
    AdminRole role;
    std::vector<std::string> permissions;
    bool is_active;
    bool two_factor_enabled;
    std::string two_factor_secret;
    std::string ip_whitelist;
    Timestamp created_at;
    Timestamp updated_at;
    Timestamp last_login_at;
    uint32_t failed_attempts;
    std::optional<Timestamp> locked_until;
};

struct Session {
    uint64_t id;
    AdminID admin_id;
    std::string token_hash;
    std::string ip_address;
    std::string user_agent;
    Timestamp expires_at;
    Timestamp created_at;
};

struct User {
    UserID id;
    std::string user_id;
    std::string username;
    std::string email;
    std::string phone;
    std::string password_hash;
    std::string wallet_address;
    UserStatus status;
    int tier;
    bool email_verified;
    bool phone_verified;
    KYCStatus kyc_status;
    int kyc_level;
    std::optional<WhiteLabelID> white_label_id;
    std::string referrer_id;
    std::string referral_code;
    Timestamp created_at;
    Timestamp updated_at;
    std::optional<Timestamp> last_login_at;
    uint32_t failed_login_count;
    std::string country;
    std::string ip_address;
};

struct KYCRequest {
    KYCRequestID id;
    UserID user_id;
    int level;
    std::string document_type;
    std::string document_number;
    std::string document_front;
    std::string document_back;
    std::string selfie_image;
    std::string first_name;
    std::string last_name;
    std::string date_of_birth;
    std::string country;
    std::string address;
    KYCStatus status;
    std::string reject_reason;
    AdminID reviewed_by;
    Timestamp reviewed_at;
    Timestamp created_at;
};

struct Transaction {
    TransactionID id;
    UserID user_id;
    std::string type;
    std::string amount;
    std::string currency;
    TransactionStatus status;
    std::string from_address;
    std::string to_address;
    std::string tx_hash;
    std::string fee;
    int chain_id;
    bool is_flagged;
    std::string flag_reason;
    Timestamp timestamp;
};

struct Withdrawal {
    WithdrawalID id;
    UserID user_id;
    std::string amount;
    std::string currency;
    WithdrawalStatus status;
    std::string address;
    std::string tx_hash;
    AdminID approved_by;
    Timestamp processed_at;
    Timestamp created_at;
};

struct Token {
    TokenID id;
    std::string token_id;
    std::string name;
    std::string symbol;
    std::string contract_address;
    int decimals;
    bool is_active;
    bool is_verified;
    std::string total_supply;
    int chain_id;
    std::string logo_url;
    std::string website;
    std::string description;
    TokenStatus status;
    Timestamp created_at;
    Timestamp updated_at;
};

struct TradingPair {
    PairID id;
    TokenID base_token_id;
    TokenID quote_token_id;
    std::string pair_name;
    std::string price;
    std::string volume_24h;
    std::string liquidity;
    PairStatus status;
    int chain_id;
    Timestamp created_at;
    Timestamp updated_at;
};

struct Blockchain {
    BlockchainID id;
    std::string name;
    std::string symbol;
    int chain_id;
    bool is_evm;
    std::string rpc_url;
    std::string explorer_url;
    std::string native_token;
    int decimals;
    bool is_active;
    std::string avg_gas_price_gwei;
    BlockchainStatus status;
    Timestamp created_at;
};

struct FeeStructure {
    uint64_t id;
    std::string fee_type;
    std::string asset;
    std::string fee_percent;
    std::string fee_fixed;
    std::string min_fee;
    std::string max_fee;
    std::string tier;
    bool is_active;
    int chain_id;
    Timestamp created_at;
    Timestamp updated_at;
};

struct Webhook {
    uint64_t id;
    std::string name;
    std::string url;
    std::string secret;
    std::vector<std::string> events;
    bool is_active;
    Timestamp created_at;
    AdminID created_by;
};

struct WhiteLabel {
    WhiteLabelID id;
    std::string client_id;
    std::string company_name;
    std::string domain;
    bool domain_verified;
    AdminID admin_user_id;
    WhiteLabelStatus status;
    std::string logo_url;
    std::string primary_color;
    std::string secondary_color;
    std::string theme_mode;
    JSON features;
    int max_users;
    double max_daily_volume;
    double platform_fee_percent;
    double custom_fee_percent;
    std::string liquidity_source;
    std::string trading_pairs_import;
    std::string contact_email;
    std::string contact_phone;
    Timestamp activated_at;
    Timestamp expires_at;
    Timestamp created_at;
};

struct Ticket {
    TicketID id;
    std::string title;
    std::string description;
    std::string ticket_type;
    TicketPriority priority;
    TicketStatus status;
    UserID created_by;
    AdminID assigned_to;
    Timestamp created_at;
    Timestamp updated_at;
    Timestamp resolved_at;
};

struct TicketMessage {
    uint64_t id;
    TicketID ticket_id;
    std::string message;
    bool is_internal;
    UserID created_by;
    AdminID admin_id;
    Timestamp created_at;
};

struct AuditLog {
    uint64_t id;
    AdminID admin_id;
    std::string action;
    std::string resource_type;
    std::string resource_id;
    JSON details;
    std::string ip_address;
    std::string user_agent;
    bool success;
    std::string error_message;
    Timestamp created_at;
};

struct FeatureFlag {
    uint64_t id;
    std::string name;
    std::string description;
    bool is_enabled;
    int rollout_percentage;
    AdminID updated_by;
    Timestamp created_at;
    Timestamp updated_at;
};

struct IPWhitelist {
    uint64_t id;
    std::string ip_address;
    std::string description;
    bool is_active;
    AdminID created_by;
    Timestamp created_at;
};

struct Notification {
    uint64_t id;
    AdminID admin_id;
    std::string title;
    std::string message;
    std::string notification_type;
    bool is_read;
    Timestamp created_at;
};

struct PlatformStats {
    int64_t total_users;
    int64_t active_users;
    int64_t suspended_users;
    double total_volume;
    int64_t total_transactions;
    double total_fees;
    int active_bots;
    int total_bots;
    int64_t pending_kyc;
    int64_t approved_kyc;
    int64_t rejected_kyc;
};

struct ProfitShareConfig {
    std::string id;
    std::string white_label_id;
    std::string super_admin_wallet;
    std::string master_wallet_address;
    double profit_percentage;
    double min_percentage;
    double max_percentage;
    bool is_active;
    bool auto_transfer;
    std::string transfer_frequency;
    Timestamp last_transfer;
    double total_transferred;
    Timestamp created_at;
    Timestamp updated_at;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_MODELS_HPP
