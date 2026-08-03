/**
 * TigerWallet Admin Platform - C++ PostgreSQL Database Implementation
 * High-performance, ultra-low latency database layer
 */

#include "../include/database.h"
#include <iostream>
#include <sstream>
#include <stdexcept>

namespace tiger {

// Database Implementation
Database::Database(const std::string& host, int port, const std::string& dbname,
                 const std::string& user, const std::string& password,
                 int min_connections, int max_connections)
    : host_(host), port_(port), dbname_(dbname), user_(user), 
      password_(password), min_connections_(min_connections), 
      max_connections_(max_connections), connected_(false) {
    
    initialize_pool();
}

Database::~Database() {
    cleanup_pool();
}

bool Database::connect() {
    try {
        for (int i = 0; i < min_connections_; ++i) {
            PGconn* conn = PQsetdbLogin(
                host_.c_str(), 
                std::to_string(port_).c_str(),
                nullptr,  // options
                nullptr,  // tty
                dbname_.c_str(),
                user_.c_str(),
                password_.c_str()
            );
            
            if (PQstatus(conn) == CONNECTION_OK) {
                auto conn_obj = std::make_unique<Connection>();
                conn_obj->conn = conn;
                conn_obj->in_use = false;
                conn_obj->last_used = std::chrono::steady_clock::now();
                pool_.push_back(std::move(conn_obj));
            } else {
                std::cerr << "Failed to connect: " << PQerrorMessage(conn) << std::endl;
                PQfinish(conn);
            }
        }
        
        connected_ = !pool_.empty();
        return connected_;
    } catch (const std::exception& e) {
        std::cerr << "Database connection error: " << e.what() << std::endl;
        return false;
    }
}

void Database::disconnect() {
    cleanup_pool();
    connected_ = false;
}

bool Database::is_connected() const {
    return connected_;
}

Database::QueryResult::QueryResult(PGresult* result) : result_(result) {}

Database::QueryResult::~QueryResult() {
    if (result_) {
        PQclear(result_);
    }
}

int Database::QueryResult::rows() const {
    return result_ ? PQntuples(result_) : 0;
}

int Database::QueryResult::columns() const {
    return result_ ? PQnfields(result_) : 0;
}

std::string Database::QueryResult::column_name(int col) const {
    return result_ ? std::string(PQfname(result_, col)) : "";
}

std::string Database::QueryResult::get_value(int row, int col) const {
    if (!result_ || row >= rows() || col >= columns()) {
        return "";
    }
    
    if (PQgetisnull(result_, row, col)) {
        return "";
    }
    
    return std::string(PQgetvalue(result_, row, col));
}

std::string Database::QueryResult::get_value(int row, const std::string& col) const {
    for (int i = 0; i < columns(); ++i) {
        if (column_name(i) == col) {
            return get_value(row, i);
        }
    }
    return "";
}

bool Database::QueryResult::is_null(int row, int col) const {
    return result_ && PQgetisnull(result_, row, col);
}

ExecStatusType Database::QueryResult::status() const {
    return result_ ? PQresultStatus(result_) : PGRES_FATAL_ERROR;
}

std::string Database::QueryResult::error_message() const {
    return result_ ? std::string(PQerrorMessage(result_)) : "";
}

std::optional<Database::QueryResult> Database::execute(const std::string& query) {
    Connection* conn = acquire_connection();
    if (!conn) {
        return std::nullopt;
    }
    
    PGresult* result = PQexec(conn->conn, query.c_str());
    
    if (PQresultStatus(result) != PGRES_TUPLES_OK && 
        PQresultStatus(result) != PGRES_COMMAND_OK) {
        std::cerr << "Query error: " << PQerrorMessage(conn->conn) << std::endl;
        PQclear(result);
        release_connection(conn);
        return std::nullopt;
    }
    
    release_connection(conn);
    return QueryResult(result);
}

std::optional<Database::QueryResult> Database::execute_params(
    const std::string& query, 
    const std::vector<std::string>& params) {
    
    Connection* conn = acquire_connection();
    if (!conn) {
        return std::nullopt;
    }
    
    std::vector<const char*> param_values;
    for (const auto& param : params) {
        param_values.push_back(param.c_str());
    }
    
    PGresult* result = PQexecParams(
        conn->conn,
        query.c_str(),
        params.size(),
        nullptr,
        param_values.data(),
        nullptr,
        nullptr,
        0
    );
    
    if (PQresultStatus(result) != PGRES_TUPLES_OK && 
        PQresultStatus(result) != PGRES_COMMAND_OK) {
        std::cerr << "Query error: " << PQerrorMessage(conn->conn) << std::endl;
        PQclear(result);
        release_connection(conn);
        return std::nullopt;
    }
    
    release_connection(conn);
    return QueryResult(result);
}

bool Database::begin() {
    auto result = execute("BEGIN");
    return result.has_value();
}

bool Database::commit() {
    auto result = execute("COMMIT");
    return result.has_value();
}

bool Database::rollback() {
    auto result = execute("ROLLBACK");
    return result.has_value();
}

bool Database::prepare_statement(const std::string& name, const std::string& query) {
    std::lock_guard<std::mutex> lock(statement_mutex_);
    prepared_statements_[name] = query;
    return true;
}

std::optional<Database::QueryResult> Database::execute_prepared(
    const std::string& name, 
    const std::vector<std::string>& params) {
    
    std::string query;
    {
        std::lock_guard<std::mutex> lock(statement_mutex_);
        auto it = prepared_statements_.find(name);
        if (it == prepared_statements_.end()) {
            return std::nullopt;
        }
        query = it->second;
    }
    
    return execute_params(query, params);
}

void Database::set_min_connections(int min) {
    min_connections_ = min;
}

void Database::set_max_connections(int max) {
    max_connections_ = max;
}

int Database::active_connections() const {
    int count = 0;
    for (const auto& conn : pool_) {
        if (conn->in_use) count++;
    }
    return count;
}

int Database::idle_connections() const {
    int count = 0;
    for (const auto& conn : pool_) {
        if (!conn->in_use) count++;
    }
    return count;
}

bool Database::health_check() {
    auto result = execute("SELECT 1");
    return result.has_value();
}

std::string Database::escape_string(const std::string& str) {
    Connection* conn = acquire_connection();
    if (!conn) {
        return str;
    }
    
    char* escaped = PQescapeString(conn->conn, str.c_str(), str.length());
    std::string result(escaped);
    PQfreemem(escaped);
    release_connection(conn);
    
    return result;
}

PGconn* Database::create_connection() {
    return PQsetdbLogin(
        host_.c_str(),
        std::to_string(port_).c_str(),
        nullptr, nullptr,
        dbname_.c_str(),
        user_.c_str(),
        password_.c_str()
    );
}

Database::Connection* Database::acquire_connection() {
    std::unique_lock<std::mutex> lock(pool_mutex_);
    
    pool_cv_.wait(lock, [this] {
        for (const auto& conn : pool_) {
            if (!conn->in_use) {
                return true;
            }
        }
        return pool_.size() < static_cast<size_t>(max_connections_);
    });
    
    for (auto& conn : pool_) {
        if (!conn->in_use) {
            conn->in_use = true;
            conn->last_used = std::chrono::steady_clock::now();
            
            if (PQstatus(conn->conn) != CONNECTION_OK) {
                PQfinish(conn->conn);
                conn->conn = create_connection();
            }
            
            return conn.get();
        }
    }
    
    // Create new connection if under max
    if (pool_.size() < static_cast<size_t>(max_connections_)) {
        auto conn_obj = std::make_unique<Connection>();
        conn_obj->conn = create_connection();
        conn_obj->in_use = true;
        conn_obj->last_used = std::chrono::steady_clock::now();
        
        Connection* result = conn_obj.get();
        pool_.push_back(std::move(conn_obj));
        return result;
    }
    
    return nullptr;
}

void Database::release_connection(Connection* conn) {
    if (!conn) return;
    
    std::lock_guard<std::mutex> lock(pool_mutex_);
    conn->in_use = false;
    conn->last_used = std::chrono::steady_clock::now();
    pool_cv_.notify_one();
}

void Database::initialize_pool() {
    // Pool will be initialized on connect()
}

void Database::cleanup_pool() {
    std::lock_guard<std::mutex> lock(pool_mutex_);
    
    for (auto& conn : pool_) {
        if (conn->conn) {
            PQfinish(conn->conn);
        }
    }
    
    pool_.clear();
}

} // namespace tiger
