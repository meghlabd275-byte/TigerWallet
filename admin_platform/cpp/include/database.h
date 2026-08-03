#pragma once

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <memory>
#include <functional>
#include <mutex>
#include <libpq-fe.h>

namespace tiger {

class Database {
public:
    Database(const std::string& host, int port, const std::string& dbname,
             const std::string& user, const std::string& password,
             int min_connections = 5, int max_connections = 50);
    ~Database();
    
    // Connection management
    bool connect();
    void disconnect();
    bool is_connected() const;
    
    // Query execution
    class QueryResult {
    public:
        QueryResult(PGresult* result);
        ~QueryResult();
        
        int rows() const;
        int columns() const;
        std::string column_name(int col) const;
        std::string get_value(int row, int col) const;
        std::string get_value(int row, const std::string& col) const;
        bool is_null(int row, int col) const;
        ExecStatusType status() const;
        std::string error_message() const;
        
    private:
        PGresult* result_;
    };
    
    std::optional<QueryResult> execute(const std::string& query);
    std::optional<QueryResult> execute_params(const std::string& query, const std::vector<std::string>& params);
    
    // Transaction support
    bool begin();
    bool commit();
    bool rollback();
    
    // Prepared statements
    bool prepare_statement(const std::string& name, const std::string& query);
    std::optional<QueryResult> execute_prepared(const std::string& name, const std::vector<std::string>& params);
    
    // Connection pool management
    void set_min_connections(int min);
    void set_max_connections(int max);
    int active_connections() const;
    int idle_connections() const;
    
    // Health check
    bool health_check();
    
    // Escape string
    std::string escape_string(const std::string& str);
    
private:
    struct Connection {
        PGconn* conn;
        bool in_use;
        std::chrono::steady_clock::time_point last_used;
    };
    
    std::vector<std::unique_ptr<Connection>> pool_;
    std::mutex pool_mutex_;
    std::condition_variable pool_cv_;
    
    std::string host_;
    int port_;
    std::string dbname_;
    std::string user_;
    std::string password_;
    int min_connections_;
    int max_connections_;
    bool connected_;
    
    PGconn* create_connection();
    Connection* acquire_connection();
    void release_connection(Connection* conn);
    void initialize_pool();
    void cleanup_pool();
    
    // Statement cache
    std::map<std::string, std::string> prepared_statements_;
    std::mutex statement_mutex_;
};

// RAII wrapper for database connections
class DBConnection {
public:
    DBConnection(Database* db);
    ~DBConnection();
    
    Database* operator->() { return db_; }
    Database* get() { return db_; }
    
private:
    Database* db_;
};

} // namespace tiger
