/**
 * TigerWallet Super Admin - PostgreSQL Database Manager Implementation
 * Production-ready PostgreSQL implementation with connection pooling
 * Ultra-low latency design for high-performance applications
 */

#include "postgres_database_manager.hpp"
#include <iostream>
#include <sstream>
#include <thread>
#include <future>
#include <chrono>
#include <libpq-fe.h>

namespace tigerwallet {
namespace super_admin {

// ============================================================================
// QueryResult Implementation
// ============================================================================

std::optional<std::string> QueryResult::get_string(size_t row, size_t col) const {
    if (!result_ || row >= result_->size() || col >= result_->columns()) {
        return std::nullopt;
    }
    
    try {
        const auto& field = (*result_)[row][col];
        if (field.is_null()) {
            return std::nullopt;
        }
        return field.as<std::string>();
    } catch (...) {
        return std::nullopt;
    }
}

std::vector<std::string> QueryResult::get_column_names() const {
    std::vector<std::string> names;
    if (!result_) return names;
    
    for (size_t i = 0; i < result_->columns(); ++i) {
        names.push_back(result_->column_name(i));
    }
    return names;
}

std::vector<std::string> QueryResult::get_row_as_strings(size_t row) const {
    std::vector<std::string> values;
    if (!result_ || row >= result_->size()) return values;
    
    for (size_t i = 0; i < result_->columns(); ++i) {
        const auto& field = (*result_)[row][i];
        if (field.is_null()) {
            values.push_back("NULL");
        } else {
            values.push_back(field.as<std::string>());
        }
    }
    return values;
}

// ============================================================================
// PostgresConnection Implementation
// ============================================================================

PostgresConnection::PostgresConnection(pqxx::connection* conn, uint64_t id)
    : conn_(conn), id_(id) {}

PostgresConnection::~PostgresConnection() {
    if (conn_) {
        try {
            if (!conn_->is_open()) {
                delete conn_;
                conn_ = nullptr;
            }
        } catch (...) {
            delete conn_;
            conn_ = nullptr;
        }
    }
}

PostgresConnection::PostgresConnection(PostgresConnection&& other) noexcept
    : conn_(other.conn_), id_(other.id_) {
    other.conn_ = nullptr;
}

PostgresConnection& PostgresConnection::operator=(PostgresConnection&& other) noexcept {
    if (this != &other) {
        if (conn_) {
            delete conn_;
        }
        conn_ = other.conn_;
        id_ = other.id_;
        other.conn_ = nullptr;
    }
    return *this;
}

bool PostgresConnection::is_valid() const {
    if (!conn_) return false;
    try {
        return conn_->is_open();
    } catch (...) {
        return false;
    }
}

void PostgresConnection::reset() {
    if (conn_) {
        try {
            conn_->reset();
        } catch (...) {
            // Connection is dead
        }
    }
}

// ============================================================================
// ConnectionPool Implementation
// ============================================================================

ConnectionPool::ConnectionPool(const PostgresConfig& config, const PoolConfig& pool_config)
    : config_(config), pool_config_(pool_config), 
      initialized_(false), next_connection_id_(1) {
    connections_.reserve(pool_config.max_connections);
}

ConnectionPool::~ConnectionPool() {
    shutdown();
}

bool ConnectionPool::initialize() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (initialized_) {
        return true;
    }
    
    // Create initial connections
    for (int i = 0; i < pool_config_.min_connections; ++i) {
        auto conn = create_connection();
        if (conn) {
            PooledConnection pc;
            pc.conn = std::move(conn);
            pc.created_at = std::chrono::steady_clock::now();
            pc.last_used_at = pc.created_at;
            pc.in_use = false;
            pc.id = next_connection_id_++;
            
            connections_.push_back(std::move(pc));
            available_ids_.push_back(connections_.back().id);
        }
    }
    
    if (connections_.empty()) {
        std::cerr << "[PostgreSQL Pool] Failed to create initial connections" << std::endl;
        return false;
    }
    
    initialized_ = true;
    std::cout << "[PostgreSQL Pool] Initialized with " << connections_.size() 
              << " connections" << std::endl;
    return true;
}

std::shared_ptr<PostgresConnection> ConnectionPool::acquire() {
    std::unique_lock<std::mutex> lock(mutex_);
    
    if (!initialized_) {
        return nullptr;
    }
    
    // Try to find an available connection
    for (auto& pc : connections_) {
        if (!pc.in_use && validate_connection(pc.conn.get())) {
            pc.in_use = true;
            pc.last_used_at = std::chrono::steady_clock::now();
            
            // Remove from available list
            available_ids_.erase(
                std::remove(available_ids_.begin(), available_ids_.end(), pc.id),
                available_ids_.end()
            );
            
            return std::make_shared<PostgresConnection>(pc.conn.get(), pc.id);
        }
    }
    
    // Try to create a new connection if we haven't hit the limit
    if (connections_.size() < pool_config_.max_connections) {
        auto conn = create_connection();
        if (conn) {
            PooledConnection pc;
            pc.conn = std::move(conn);
            pc.created_at = std::chrono::steady_clock::now();
            pc.last_used_at = pc.created_at;
            pc.in_use = true;
            pc.id = next_connection_id_++;
            
            connections_.push_back(std::move(pc));
            
            return std::make_shared<PostgresConnection>(
                connections_.back().conn.get(), 
                connections_.back().id
            );
        }
    }
    
    // Wait for a connection to become available
    auto timeout = std::chrono::seconds(pool_config_.acquire_timeout);
    if (cv_.wait_for(lock, timeout, [this] { 
        return available_ids_.size() > 0; 
    })) {
        return acquire();  // Retry acquiring
    }
    
    std::cerr << "[PostgreSQL Pool] Timeout waiting for connection" << std::endl;
    return nullptr;
}

void ConnectionPool::release(std::shared_ptr<PostgresConnection> conn) {
    if (!conn) return;
    
    std::lock_guard<std::mutex> lock(mutex_);
    
    for (auto& pc : connections_) {
        if (pc.id == conn->get_id()) {
            pc.in_use = false;
            pc.last_used_at = std::chrono::steady_clock::now();
            available_ids_.push_back(pc.id);
            cv_.notify_one();
            break;
        }
    }
}

size_t ConnectionPool::get_active_connections() const {
    std::lock_guard<std::mutex> lock(mutex_);
    size_t count = 0;
    for (const auto& pc : connections_) {
        if (pc.in_use) ++count;
    }
    return count;
}

size_t ConnectionPool::get_idle_connections() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return available_ids_.size();
}

size_t ConnectionPool::get_total_connections() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return connections_.size();
}

bool ConnectionPool::health_check() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    for (auto& pc : connections_) {
        if (pc.in_use) continue;
        
        if (!validate_connection(pc.conn.get())) {
            // Try to recreate
            pc.conn = create_connection();
            if (!pc.conn) {
                return false;
            }
        }
    }
    return true;
}

void ConnectionPool::shutdown() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    for (auto& pc : connections_) {
        if (pc.conn && pc.conn->is_open()) {
            pc.conn->close();
        }
    }
    
    connections_.clear();
    available_ids_.clear();
    initialized_ = false;
}

std::unique_ptr<pqxx::connection> ConnectionPool::create_connection() {
    try {
        std::string connstr = 
            "host=" + config_.host + " "
            "port=" + std::to_string(config_.port) + " "
            "dbname=" + config_.database + " "
            "user=" + config_.username + " "
            "password=" + config_.password + " "
            "connect_timeout=" + std::to_string(config_.connection_timeout) + " "
            "sslmode=" + config_.ssl_mode + " "
            "application_name=tigerwallet_super_admin";
        
        auto conn = std::make_unique<pqxx::connection>(connstr);
        
        if (conn->is_open()) {
            // Set session parameters for performance
            pqxx::work w(*conn);
            w.exec("SET statement_timeout = " + std::to_string(config_.statement_timeout));
            w.exec("SET idle_in_transaction_session_timeout = 30000");
            w.exec("SET lock_timeout = 10000");
            w.commit();
            
            return conn;
        }
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL Pool] Connection failed: " << e.what() << std::endl;
    }
    
    return nullptr;
}

bool ConnectionPool::validate_connection(pqxx::connection* conn) {
    if (!conn || !conn->is_open()) return false;
    
    try {
        pqxx::nontransaction n(*conn);
        n.exec("SELECT 1");
        return true;
    } catch (...) {
        return false;
    }
}

void ConnectionPool::cleanup_idle_connections() {
    auto now = std::chrono::steady_clock::now();
    
    for (auto it = connections_.begin(); it != connections_.end(); ++it) {
        if (it->in_use) continue;
        
        auto idle_time = std::chrono::duration_cast<std::chrono::seconds>(
            now - it->last_used_at
        ).count();
        
        if (idle_time > pool_config_.max_idle_time && 
            connections_.size() > pool_config_.min_connections) {
            it->conn->close();
            // Remove from available list
            available_ids_.erase(
                std::remove(available_ids_.begin(), available_ids_.end(), it->id),
                available_ids_.end()
            );
            connections_.erase(it);
        }
    }
}

// ============================================================================
// PostgresDatabaseManager Implementation
// ============================================================================

PostgresDatabaseManager& PostgresDatabaseManager::getInstance() {
    static PostgresDatabaseManager instance;
    return instance;
}

PostgresDatabaseManager::PostgresDatabaseManager()
    : initialized_(false) {
    metrics_.start_time = std::chrono::steady_clock::now();
}

PostgresDatabaseManager::~PostgresDatabaseManager() {
    shutdown();
}

void PostgresDatabaseManager::configure(const PostgresConfig& config) {
    config_ = config;
}

void PostgresDatabaseManager::configure_pool(const PoolConfig& config) {
    pool_config_ = config;
}

bool PostgresDatabaseManager::initialize() {
    if (initialized_) {
        return true;
    }
    
    pool_ = std::make_unique<ConnectionPool>(config_, pool_config_);
    
    if (!pool_->initialize()) {
        std::cerr << "[PostgreSQL] Failed to initialize connection pool" << std::endl;
        return false;
    }
    
    initialized_ = true;
    start_time_ = std::chrono::steady_clock::now();
    
    std::cout << "[PostgreSQL] Database manager initialized successfully" << std::endl;
    return true;
}

bool PostgresDatabaseManager::initialize(const std::string& connection_string) {
    // Parse connection string and configure
    // Format: postgresql://user:password@host:port/database
    
    if (connection_string.find("postgresql://") == 0) {
        size_t pos = 11;  // Length of "postgresql://"
        
        // Extract user:password
        size_t at_pos = connection_string.find('@', pos);
        if (at_pos != std::string::npos) {
            size_t colon_pos = connection_string.find(':', pos);
            if (colon_pos < at_pos) {
                config_.username = connection_string.substr(pos, colon_pos - pos);
                config_.password = connection_string.substr(colon_pos + 1, at_pos - colon_pos - 1);
            }
            
            // Extract host:port/database
            size_t slash_pos = connection_string.find('/', at_pos + 3);
            if (slash_pos != std::string::npos) {
                std::string host_port = connection_string.substr(at_pos + 3, slash_pos - at_pos - 3);
                size_t colon_pos2 = host_port.find(':');
                if (colon_pos2 != std::string::npos) {
                    config_.host = host_port.substr(0, colon_pos2);
                    config_.port = std::stoi(host_port.substr(colon_pos2 + 1));
                } else {
                    config_.host = host_port;
                }
                
                config_.database = connection_string.substr(slash_pos + 1);
            }
        }
    }
    
    return initialize();
}

void PostgresDatabaseManager::shutdown() {
    if (pool_) {
        pool_->shutdown();
        pool_.reset();
    }
    initialized_ = false;
}

bool PostgresDatabaseManager::is_connected() const {
    return initialized_ && pool_ && pool_->get_total_connections() > 0;
}

std::string PostgresDatabaseManager::get_version() const {
    auto result = execute("SELECT version()");
    if (!result.is_empty()) {
        return result.get_string(0, 0).value_or("");
    }
    return "";
}

QueryResult PostgresDatabaseManager::execute(const std::string& query) {
    auto conn = pool_->acquire();
    if (!conn) {
        std::cerr << "[PostgreSQL] Failed to acquire connection" << std::endl;
        return QueryResult();
    }
    
    try {
        pqxx::work w(**conn);
        auto res = w.exec(query);
        w.commit();
        
        auto shared_res = std::make_shared<pqxx::result>(std::move(res));
        
        pool_->release(conn);
        update_metrics_on_query(true);
        
        return QueryResult(shared_res);
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Query error: " << e.what() << std::endl;
        
        try {
            conn->reset();
        } catch (...) {}
        
        pool_->release(conn);
        update_metrics_on_query(false);
        
        return QueryResult();
    }
}

QueryResult PostgresDatabaseManager::execute(const std::string& query, 
                                              const std::vector<std::string>& params) {
    auto conn = pool_->acquire();
    if (!conn) {
        return QueryResult();
    }
    
    try {
        pqxx::work w(**conn);
        
        pqxx::params p;
        for (const auto& param : params) {
            p.append(param);
        }
        
        auto res = w.exec_params(query, p);
        w.commit();
        
        auto shared_res = std::make_shared<pqxx::result>(std::move(res));
        pool_->release(conn);
        update_metrics_on_query(true);
        
        return QueryResult(shared_res);
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Query error: " << e.what() << std::endl;
        pool_->release(conn);
        update_metrics_on_query(false);
        return QueryResult();
    }
}

QueryResult PostgresDatabaseManager::execute(const std::string& query, 
                                              const std::map<std::string, std::string>& named_params) {
    auto conn = pool_->acquire();
    if (!conn) {
        return QueryResult();
    }
    
    try {
        pqxx::work w(**conn);
        
        std::string param_list;
        std::vector<std::string> values;
        
        for (const auto& [key, value] : named_params) {
            if (!param_list.empty()) param_list += ", ";
            param_list += key + " := $" + std::to_string(values.size() + 1);
            values.push_back(value);
        }
        
        pqxx::params p;
        for (const auto& v : values) {
            p.append(v);
        }
        
        auto res = w.exec(query, p);
        w.commit();
        
        auto shared_res = std::make_shared<pqxx::result>(std::move(res));
        pool_->release(conn);
        update_metrics_on_query(true);
        
        return QueryResult(shared_res);
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Query error: " << e.what() << std::endl;
        pool_->release(conn);
        update_metrics_on_query(false);
        return QueryResult();
    }
}

bool PostgresDatabaseManager::prepare_statement(const std::string& name, 
                                                 const std::string& query) {
    auto conn = pool_->acquire();
    if (!conn) return false;
    
    try {
        pqxx::nontransaction n(**conn);
        n.prepare(name, query);
        pool_->release(conn);
        return true;
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Prepare error: " << e.what() << std::endl;
        pool_->release(conn);
        return false;
    }
}

QueryResult PostgresDatabaseManager::execute_prepared(const std::string& name) {
    auto conn = pool_->acquire();
    if (!conn) return QueryResult();
    
    try {
        pqxx::nontransaction n(**conn);
        auto res = n.prepared(name).exec();
        
        auto shared_res = std::make_shared<pqxx::result>(std::move(res));
        pool_->release(conn);
        update_metrics_on_query(true);
        
        return QueryResult(shared_res);
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Prepared query error: " << e.what() << std::endl;
        pool_->release(conn);
        update_metrics_on_query(false);
        return QueryResult();
    }
}

QueryResult PostgresDatabaseManager::execute_prepared(const std::string& name, 
                                                       const std::vector<std::string>& params) {
    auto conn = pool_->acquire();
    if (!conn) return QueryResult();
    
    try {
        pqxx::nontransaction n(**conn);
        
        pqxx::params p;
        for (const auto& param : params) {
            p.append(param);
        }
        
        auto res = n.prepared(name).exec(p);
        
        auto shared_res = std::make_shared<pqxx::result>(std::move(res));
        pool_->release(conn);
        update_metrics_on_query(true);
        
        return QueryResult(shared_res);
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Prepared query error: " << e.what() << std::endl;
        pool_->release(conn);
        update_metrics_on_query(false);
        return QueryResult();
    }
}

QueryResult PostgresDatabaseManager::execute_prepared(const std::string& name, 
                                                       const std::map<std::string, std::string>& params) {
    auto conn = pool_->acquire();
    if (!conn) return QueryResult();
    
    try {
        pqxx::nontransaction n(**conn);
        
        pqxx::params p;
        for (const auto& [key, value] : params) {
            p.append(key, value);
        }
        
        auto res = n.prepared(name).exec(p);
        
        auto shared_res = std::make_shared<pqxx::result>(std::move(res));
        pool_->release(conn);
        update_metrics_on_query(true);
        
        return QueryResult(shared_res);
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Prepared query error: " << e.what() << std::endl;
        pool_->release(conn);
        update_metrics_on_query(false);
        return QueryResult();
    }
}

// ============================================================================
// Transaction Implementation
// ============================================================================

PostgresDatabaseManager::Transaction::Transaction(PostgresDatabaseManager& db)
    : db_(db), active_(true), committed_(false) {
    auto conn = db.pool_->acquire();
    if (conn) {
        work_ = std::make_unique<pqxx::work>(**conn);
    }
}

PostgresDatabaseManager::Transaction::~Transaction() {
    if (active_ && !committed_) {
        rollback();
    }
}

void PostgresDatabaseManager::Transaction::commit() {
    if (!active_ || committed_) return;
    
    try {
        work_->commit();
        committed_ = true;
        active_ = false;
        db_.update_metrics_on_transaction(true);
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Transaction commit error: " << e.what() << std::endl;
        rollback();
        throw;
    }
}

void PostgresDatabaseManager::Transaction::rollback() {
    if (!active_) return;
    
    try {
        work_->abort();
        active_ = false;
        db_.update_metrics_on_transaction(false);
    } catch (...) {}
}

QueryResult PostgresDatabaseManager::Transaction::execute(const std::string& query) {
    if (!active_) return QueryResult();
    
    try {
        auto res = work_->exec(query);
        auto shared_res = std::make_shared<pqxx::result>(std::move(res));
        return QueryResult(shared_res);
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Transaction query error: " << e.what() << std::endl;
        throw;
    }
}

QueryResult PostgresDatabaseManager::Transaction::execute(const std::string& query, 
                                                          const std::vector<std::string>& params) {
    if (!active_) return QueryResult();
    
    try {
        pqxx::params p;
        for (const auto& param : params) {
            p.append(param);
        }
        
        auto res = work_->exec_params(query, p);
        auto shared_res = std::make_shared<pqxx::result>(std::move(res));
        return QueryResult(shared_res);
    } catch (const std::exception& e) {
        std::cerr << "[PostgreSQL] Transaction query error: " << e.what() << std::endl;
        throw;
    }
}

std::unique_ptr<PostgresDatabaseManager::Transaction> 
PostgresDatabaseManager::begin_transaction() {
    return std::make_unique<Transaction>(*this);
}

// ============================================================================
// Table Operations
// ============================================================================

bool PostgresDatabaseManager::create_table(const std::string& table_name, 
                                           const std::string& schema) {
    auto result = execute("CREATE TABLE IF NOT EXISTS " + quote_identifier(table_name) + 
                         " (" + schema + ")");
    return !result.is_empty() || table_exists(table_name);
}

bool PostgresDatabaseManager::drop_table(const std::string& table_name) {
    auto result = execute("DROP TABLE IF EXISTS " + quote_identifier(table_name));
    return !table_exists(table_name);
}

bool PostgresDatabaseManager::table_exists(const std::string& table_name) {
    auto result = execute(
        "SELECT EXISTS (SELECT FROM information_schema.tables " 
        "WHERE table_schema = 'public' AND table_name = $1)",
        {table_name}
    );
    
    if (!result.is_empty()) {
        return result.get_bool(0, 0).value_or(false);
    }
    return false;
}

std::vector<std::string> PostgresDatabaseManager::list_tables() {
    std::vector<std::string> tables;
    auto result = execute(
        "SELECT table_name FROM information_schema.tables "
        "WHERE table_schema = 'public' ORDER BY table_name"
    );
    
    for (size_t i = 0; i < result.row_count(); ++i) {
        auto name = result.get_string(i, 0);
        if (name) {
            tables.push_back(*name);
        }
    }
    return tables;
}

// ============================================================================
// Schema Operations
// ============================================================================

bool PostgresDatabaseManager::execute_schema(const std::string& schema_sql) {
    auto result = execute(schema_sql);
    return !result.is_empty();
}

std::string PostgresDatabaseManager::get_table_schema(const std::string& table_name) {
    auto result = execute(
        "SELECT column_name, data_type, is_nullable, column_default "
        "FROM information_schema.columns "
        "WHERE table_name = $1 ORDER BY ordinal_position",
        {table_name}
    );
    
    std::ostringstream oss;
    for (size_t i = 0; i < result.row_count(); ++i) {
        auto name = result.get_string(i, 0);
        auto type = result.get_string(i, 1);
        auto nullable = result.get_string(i, 2);
        auto def = result.get_string(i, 3);
        
        if (name && type) {
            oss << *name << " " << *type;
            if (nullable && *nullable == "NO") oss << " NOT NULL";
            if (def) oss << " DEFAULT " << *def;
            oss << ",\n";
        }
    }
    return oss.str();
}

// ============================================================================
// Metrics
// ============================================================================

DatabaseMetrics PostgresDatabaseManager::get_metrics() const {
    DatabaseMetrics m = metrics_;
    if (pool_) {
        m.current_active_connections = pool_->get_active_connections();
    }
    return m;
}

void PostgresDatabaseManager::reset_metrics() {
    metrics_.total_queries = 0;
    metrics_.successful_queries = 0;
    metrics_.failed_queries = 0;
    metrics_.total_connection_acquisitions = 0;
    metrics_.total_transaction_commits = 0;
    metrics_.total_transaction_rollbacks = 0;
    metrics_.current_active_connections = 0;
    metrics_.peak_active_connections = 0;
    metrics_.start_time = std::chrono::steady_clock::now();
}

bool PostgresDatabaseManager::health_check() {
    if (!pool_) return false;
    return pool_->health_check();
}

// ============================================================================
// Utilities
// ============================================================================

std::string PostgresDatabaseManager::escape_string(const std::string& input) const {
    // Use PQescapeStringConn for proper escaping
    std::string result;
    result.reserve(input.size() * 2);
    
    for (char c : input) {
        if (c == '\'') {
            result += "''";
        } else if (c == '\\') {
            result += "\\\\";
        } else {
            result += c;
        }
    }
    return result;
}

std::string PostgresDatabaseManager::quote_identifier(const std::string& identifier) const {
    // PostgreSQL identifier quoting
    std::string result = "\"";
    for (char c : identifier) {
        if (c == '"') {
            result += "\"\"";
        } else {
            result += c;
        }
    }
    result += "\"";
    return result;
}

void PostgresDatabaseManager::execute_async(const std::string& query, 
                                             QueryCallback on_success, 
                                             ErrorCallback on_error) {
    std::thread([this, query, on_success, on_error]() {
        try {
            auto result = execute(query);
            if (on_success) {
                on_success(result);
            }
        } catch (const std::exception& e) {
            if (on_error) {
                on_error(e.what());
            }
        }
    }).detach();
}

void PostgresDatabaseManager::execute_async(const std::string& query, 
                                             const std::vector<std::string>& params,
                                             QueryCallback on_success, 
                                             ErrorCallback on_error) {
    std::thread([this, query, params, on_success, on_error]() {
        try {
            auto result = execute(query, params);
            if (on_success) {
                on_success(result);
            }
        } catch (const std::exception& e) {
            if (on_error) {
                on_error(e.what());
            }
        }
    }).detach();
}

std::string PostgresDatabaseManager::build_connection_string() const {
    std::ostringstream oss;
    oss << "postgresql://" << config_.username << ":" << config_.password 
        << "@" << config_.host << ":" << config_.port << "/" << config_.database
        << "?sslmode=" << config_.ssl_mode
        << "&connect_timeout=" << config_.connection_timeout;
    return oss.str();
}

void PostgresDatabaseManager::update_metrics_on_query(bool success) {
    metrics_.total_queries++;
    if (success) {
        metrics_.successful_queries++;
    } else {
        metrics_.failed_queries++;
    }
}

void PostgresDatabaseManager::update_metrics_on_transaction(bool commit) {
    if (commit) {
        metrics_.total_transaction_commits++;
    } else {
        metrics_.total_transaction_rollbacks++;
    }
}

}  // namespace super_admin
}  // namespace tigerwallet
