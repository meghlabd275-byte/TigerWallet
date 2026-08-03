/**
 * TigerWallet Super Admin - PostgreSQL Database Manager
 * Production-ready PostgreSQL implementation with connection pooling
 * Ultra-low latency design for high-performance applications
 * 
 * Features:
 * - Connection pooling
 * - Prepared statements
 * - Async queries
 * - Automatic reconnection
 * - SSL support
 * - Transaction management
 */

#ifndef POSTGRES_DATABASE_MANAGER_HPP
#define POSTGRES_DATABASE_MANAGER_HPP

#include <string>
#include <vector>
#include <map>
#include <mutex>
#include <memory>
#include <atomic>
#include <chrono>
#include <functional>
#include <optional>
#include <pqxx/pqxx>

namespace tigerwallet {
namespace super_admin {

// Configuration structures
struct PostgresConfig {
    std::string host = "localhost";
    int port = 5432;
    std::string database = "tigerwallet";
    std::string username = "tigerwallet";
    std::string password = "";
    int pool_size = 20;
    int max_connections = 100;
    int connection_timeout = 10;
    int statement_timeout = 30000;  // 30 seconds
    bool use_ssl = true;
    std::string ssl_mode = "require";
};

struct PoolConfig {
    int min_connections = 5;
    int max_connections = 50;
    int max_idle_time = 300;  // seconds
    int max_lifetime = 3600;   // seconds
    int acquire_timeout = 30;   // seconds
    bool enable_ipv6 = false;
};

struct DatabaseMetrics {
    std::atomic<uint64_t> total_queries{0};
    std::atomic<uint64_t> successful_queries{0};
    std::atomic<uint64_t> failed_queries{0};
    std::atomic<uint64_t> total_connection_acquisitions{0};
    std::atomic<uint64_t> total_transaction_commits{0};
    std::atomic<uint64_t> total_transaction_rollbacks{0};
    std::atomic<uint64_t> current_active_connections{0};
    std::atomic<uint64_t> peak_active_connections{0};
    std::chrono::steady_clock::time_point start_time;
    
    double get_average_query_time_ms() const;
    double get_queries_per_second() const;
    double get_connection_pool_utilization() const;
};

// Query result wrapper
class QueryResult {
public:
    QueryResult() = default;
    explicit QueryResult(std::shared_ptr<pqxx::result> res);
    
    bool is_empty() const { return result_ == nullptr || result_->empty(); }
    size_t row_count() const { return result_ ? result_->size() : 0; }
    size_t column_count() const { return result_ ? result_->columns() : 0; }
    
    template<typename T>
    std::optional<T> get_value(size_t row, size_t col) const {
        if (!result_ || row >= result_->size() || col >= result_->columns()) {
            return std::nullopt;
        }
        
        try {
            const auto& field = (*result_)[row][col];
            if (field.is_null()) {
                return std::nullopt;
            }
            return field.as<T>();
        } catch (...) {
            return std::nullopt;
        }
    }
    
    std::optional<std::string> get_string(size_t row, size_t col) const;
    std::optional<int> get_int(size_t row, size_t col) const;
    std::optional<int64_t> get_int64(size_t row, size_t col) const;
    std::optional<double> get_double(size_t row, size_t col) const;
    std::optional<bool> get_bool(size_t row, size_t col) const;
    
    std::vector<std::string> get_column_names() const;
    std::vector<std::string> get_row_as_strings(size_t row) const;
    
private:
    std::shared_ptr<pqxx::result> result_;
};

// Connection wrapper with automatic cleanup
class PostgresConnection {
public:
    PostgresConnection(pqxx::connection* conn, uint64_t id);
    ~PostgresConnection();
    
    PostgresConnection(const PostgresConnection&) = delete;
    PostgresConnection& operator=(const PostgresConnection&) = delete;
    
    PostgresConnection(PostgresConnection&& other) noexcept;
    PostgresConnection& operator=(PostgresConnection&& other) noexcept;
    
    pqxx::connection* get() { return conn_; }
    pqxx::connection& operator*() { return *conn_; }
    pqxx::connection* operator->() { return conn_; }
    
    uint64_t get_id() const { return id_; }
    bool is_valid() const;
    void reset();
    
private:
    pqxx::connection* conn_;
    uint64_t id_;
};

// Connection pool implementation
class ConnectionPool {
public:
    ConnectionPool(const PostgresConfig& config, const PoolConfig& pool_config);
    ~ConnectionPool();
    
    // Initialize the pool
    bool initialize();
    
    // Acquire a connection from the pool
    std::shared_ptr<PostgresConnection> acquire();
    
    // Release a connection back to the pool
    void release(std::shared_ptr<PostgresConnection> conn);
    
    // Get pool statistics
    size_t get_active_connections() const;
    size_t get_idle_connections() const;
    size_t get_total_connections() const;
    
    // Health check
    bool health_check();
    
    // Shutdown the pool
    void shutdown();
    
private:
    struct PooledConnection {
        std::unique_ptr<pqxx::connection> conn;
        std::chrono::steady_clock::time_point created_at;
        std::chrono::steady_clock::time_point last_used_at;
        bool in_use;
        uint64_t id;
    };
    
    PostgresConfig config_;
    PoolConfig pool_config_;
    std::vector<PooledConnection> connections_;
    std::vector<uint64_t> available_ids_;
    std::mutex mutex_;
    std::condition_variable cv_;
    std::atomic<bool> initialized_;
    std::atomic<uint64_t> next_connection_id_;
    
    std::unique_ptr<pqxx::connection> create_connection();
    bool validate_connection(pqxx::connection* conn);
    void cleanup_idle_connections();
};

// Main database manager class
class PostgresDatabaseManager {
public:
    static PostgresDatabaseManager& getInstance();
    
    // Configuration
    void configure(const PostgresConfig& config);
    void configure_pool(const PoolConfig& config);
    
    // Initialization
    bool initialize();
    bool initialize(const std::string& connection_string);
    void shutdown();
    
    // Connection status
    bool is_connected() const;
    std::string get_version() const;
    
    // Query methods
    QueryResult execute(const std::string& query);
    QueryResult execute(const std::string& query, const std::vector<std::string>& params);
    QueryResult execute(const std::string& query, const std::map<std::string, std::string>& named_params);
    
    // Prepared statements
    bool prepare_statement(const std::string& name, const std::string& query);
    QueryResult execute_prepared(const std::string& name);
    QueryResult execute_prepared(const std::string& name, const std::vector<std::string>& params);
    QueryResult execute_prepared(const std::string& name, const std::map<std::string, std::string>& params);
    
    // Transaction support
    class Transaction {
    public:
        Transaction(PostgresDatabaseManager& db);
        ~Transaction();
        
        void commit();
        void rollback();
        
        QueryResult execute(const std::string& query);
        QueryResult execute(const std::string& query, const std::vector<std::string>& params);
        
        bool is_active() const { return active_; }
        
    private:
        PostgresDatabaseManager& db_;
        std::unique_ptr<pqxx::work> work_;
        bool active_;
        bool committed_;
    };
    
    std::unique_ptr<Transaction> begin_transaction();
    
    // Table operations
    bool create_table(const std::string& table_name, const std::string& schema);
    bool drop_table(const std::string& table_name);
    bool table_exists(const std::string& table_name);
    std::vector<std::string> list_tables();
    
    // Schema operations
    bool execute_schema(const std::string& schema_sql);
    std::string get_table_schema(const std::string& table_name);
    
    // Metrics
    DatabaseMetrics get_metrics() const;
    void reset_metrics();
    
    // Health check
    bool health_check();
    
    // Escape/quote utilities
    std::string escape_string(const std::string& input) const;
    std::string quote_identifier(const std::string& identifier) const;
    
    // Batch operations
    template<typename Func>
    void batch_execute(const std::vector<std::string>& queries, Func callback);
    
    // Async query support
    using QueryCallback = std::function<void(QueryResult)>;
    using ErrorCallback = std::function<void(const std::string&)>;
    
    void execute_async(const std::string& query, QueryCallback on_success, ErrorCallback on_error);
    void execute_async(const std::string& query, const std::vector<std::string>& params, 
                       QueryCallback on_success, ErrorCallback on_error);

private:
    PostgresDatabaseManager();
    ~PostgresDatabaseManager();
    
    PostgresDatabaseManager(const PostgresDatabaseManager&) = delete;
    PostgresDatabaseManager& operator=(const PostgresDatabaseManager&) = delete;
    
    PostgresConfig config_;
    PoolConfig pool_config_;
    std::unique_ptr<ConnectionPool> pool_;
    DatabaseMetrics metrics_;
    std::atomic<bool> initialized_;
    std::mutex query_mutex_;
    std::chrono::steady_clock::time_point start_time_;
    
    std::string build_connection_string() const;
    void update_metrics_on_query(bool success);
    void update_metrics_on_transaction(bool commit);
};

// Inline implementations

inline QueryResult::QueryResult(std::shared_ptr<pqxx::result> res) 
    : result_(std::move(res)) {}

inline std::optional<std::string> QueryResult::get_string(size_t row, size_t col) const {
    return get_value<std::string>(row, col);
}

inline std::optional<int> QueryResult::get_int(size_t row, size_t col) const {
    return get_value<int>(row, col);
}

inline std::optional<int64_t> QueryResult::get_int64(size_t row, size_t col) const {
    return get_value<int64_t>(row, col);
}

inline std::optional<double> QueryResult::get_double(size_t row, size_t col) const {
    return get_value<double>(row, col);
}

inline std::optional<bool> QueryResult::get_bool(size_t row, size_t col) const {
    return get_value<bool>(row, col);
}

inline double DatabaseMetrics::get_average_query_time_ms() const {
    return 0.0;  // Would need timing implementation
}

inline double DatabaseMetrics::get_queries_per_second() const {
    auto elapsed = std::chrono::steady_clock::now() - start_time;
    auto seconds = std::chrono::duration_cast<std::chrono::seconds>(elapsed).count();
    return seconds > 0 ? static_cast<double>(total_queries.load()) / seconds : 0.0;
}

inline double DatabaseMetrics::get_connection_pool_utilization() const {
    return 0.0;  // Would need pool size tracking
}

template<typename Func>
void PostgresDatabaseManager::batch_execute(const std::vector<std::string>& queries, Func callback) {
    auto txn = begin_transaction();
    try {
        for (const auto& query : queries) {
            auto result = txn->execute(query);
            if (callback) {
                callback(result);
            }
        }
        txn->commit();
    } catch (...) {
        txn->rollback();
        throw;
    }
}

}  // namespace super_admin
}  // namespace tigerwallet

#endif  // POSTGRES_DATABASE_MANAGER_HPP
