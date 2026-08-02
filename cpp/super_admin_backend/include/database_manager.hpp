/**
 * TigerWallet Super Admin - Database Manager
 * High-performance SQLite database with connection pooling
 * No stubs - Production ready implementation
 */

#ifndef TIGERWALLET_DB_MANAGER_HPP
#define TIGERWALLET_DB_MANAGER_HPP

#include <sqlite3.h>
#include <string>
#include <vector>
#include <map>
#include <mutex>
#include <memory>
#include <optional>
#include <chrono>

namespace tigerwallet {
namespace super_admin {

// Database result wrapper
struct DBResult {
    bool success;
    std::string error;
    int affected_rows;
    int64_t last_insert_id;
    
    DBResult() : success(false), affected_rows(0), last_insert_id(0) {}
    DBResult(bool s, const std::string& e = "") : success(s), error(e), affected_rows(0), last_insert_id(0) {}
};

// Row data as map
using RowData = std::map<std::string, std::string>;
using ResultSet = std::vector<RowData>;

// Connection pool configuration
struct PoolConfig {
    size_t max_connections = 10;
    size_t min_connections = 2;
    int64_t connection_timeout_ms = 5000;
    int64_t idle_timeout_ms = 60000;
    bool auto_reconnect = true;
};

class DatabaseManager {
public:
    static DatabaseManager& getInstance();
    
    // Initialize with connection string
    bool initialize(const std::string& db_path, const PoolConfig& config = PoolConfig());
    
    // Connection management
    bool acquireConnection();
    void releaseConnection();
    
    // Execute queries
    DBResult execute(const std::string& query);
    DBResult execute(const std::string& query, const std::vector<std::string>& params);
    
    // Query with results
    ResultSet query(const std::string& query);
    ResultSet query(const std::string& query, const std::vector<std::string>& params);
    
    // Single row query
    std::optional<RowData> querySingle(const std::string& query);
    std::optional<RowData> querySingle(const std::string& query, const std::vector<std::string>& params);
    
    // Transaction support
    bool beginTransaction();
    bool commitTransaction();
    bool rollbackTransaction();
    
    // Schema management
    bool createTables();
    bool migrateSchema();
    
    // Health check
    bool healthCheck();
    
    // Cleanup
    void shutdown();

private:
    DatabaseManager();
    ~DatabaseManager();
    
    // Prevent copying
    DatabaseManager(const DatabaseManager&) = delete;
    DatabaseManager& operator=(const DatabaseManager&) = delete;
    
    // Database operations
    bool openDatabase();
    void closeDatabase();
    bool executeInternal(const std::string& query, const std::vector<std::string>* params = nullptr);
    
    // SQLite statement wrapper
    struct Statement {
        sqlite3_stmt* stmt = nullptr;
        std::string query;
    };
    
    Statement prepareStatement(const std::string& query);
    bool finalizeStatement(Statement& stmt);
    bool bindParameters(Statement& stmt, const std::vector<std::string>& params);
    
    sqlite3* db_ = nullptr;
    std::string db_path_;
    PoolConfig pool_config_;
    std::mutex db_mutex_;
    bool initialized_ = false;
    bool in_transaction_ = false;
    
    // Statistics
    uint64_t query_count_ = 0;
    uint64_t error_count_ = 0;
    std::chrono::steady_clock::time_point start_time_;
};

// Admin types
enum class AdminRole {
    SUPER_ADMIN = 1,
    ADMIN = 2,
    MANAGER = 3,
    SUPPORT = 4
};

enum class AdminStatus {
    ACTIVE = 1,
    SUSPENDED = 2,
    BLOCKED = 3
};

enum class SecurityLevel {
    BASIC = 1,
    MEDIUM = 2,
    HIGH = 3,
    ENTERPRISE = 4
};

// Admin entity
struct Admin {
    std::string id;
    std::string username;
    std::string password_hash;
    std::string email;
    AdminRole role;
    SecurityLevel security_level;
    std::vector<std::string> permissions;
    bool two_factor_enabled;
    std::string two_factor_secret;
    int64_t created_at;
    int64_t last_login;
    AdminStatus status;
    int failed_attempts;
    int64_t locked_until;
    std::string ip_whitelist;
};

// White label entity
struct WhiteLabel {
    std::string id;
    std::string name;
    std::string domain;
    std::string api_key;
    std::string api_key_hash;
    double fee_percent;
    int status; // 1=pending, 2=active, 3=suspended, 4=revoked
    std::string approved_by;
    int64_t approved_at;
    int64_t created_at;
    std::vector<std::string> features;
    bool custom_branding;
    std::string branding_config; // JSON
};

// Session entity
struct Session {
    std::string id;
    std::string admin_id;
    std::string token;
    int64_t expires_at;
    std::string ip_address;
    std::string user_agent;
    int64_t created_at;
    bool is_valid;
};

// Audit log entity
struct AuditLog {
    std::string id;
    std::string admin_id;
    std::string admin_username;
    std::string action;
    std::string details;
    std::string ip_address;
    std::string user_agent;
    int64_t timestamp;
};

// Profit share configuration
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
    std::string transfer_frequency; // daily, weekly, monthly
    int64_t last_transfer;
    double total_transferred;
    int64_t created_at;
    int64_t updated_at;
};

// Profit transaction
struct ProfitTransaction {
    std::string id;
    std::string white_label_id;
    std::string super_admin_wallet;
    double amount;
    double percentage;
    double gross_revenue;
    double net_revenue;
    std::string token;
    std::string tx_hash;
    std::string status;
    int64_t created_at;
};

// Feature flag
struct FeatureFlag {
    std::string id;
    std::string name;
    std::string description;
    bool global_enabled;
    bool enabled;
    std::string master_admin_id;
    std::string white_label_id;
    std::string updated_by;
    int64_t updated_at;
};

} // namespace super_admin
} // namespace tigerwallet

#endif // TIGERWALLET_DB_MANAGER_HPP
