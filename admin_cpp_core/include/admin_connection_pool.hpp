/**
 * TigerAdmin C++ Core - Connection Pool
 * Thread-safe connection pooling for PostgreSQL and Redis
 */

#ifndef TIGER_ADMIN_CONNECTION_POOL_HPP
#define TIGER_ADMIN_CONNECTION_POOL_HPP

#include <string>
#include <vector>
#include <queue>
#include <mutex>
#include <memory>
#include <atomic>
#include <chrono>
#include <functional>
#include <optional>
#include <future>
#include "admin_config.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// PostgreSQL Connection
// ============================================================================

class PGConnection {
public:
    PGConnection(const Config& config);
    ~PGConnection();
    
    bool connect();
    void disconnect();
    bool is_connected() const;
    
    // Query execution
    std::optional<std::string> execute(const std::string& query);
    std::optional<std::string> execute_params(const std::string& query, 
                                               const std::vector<std::string>& params);
    
    // Transaction
    void begin();
    void commit();
    void rollback();
    
    // Escape string
    std::string escape(const std::string& value);
    
private:
    Config config_;
    void* conn_ = nullptr; // PGconn*
    bool in_transaction_ = false;
};

// ============================================================================
// Redis Connection
// ============================================================================

class RedisConnection {
public:
    RedisConnection(const Config& config);
    ~RedisConnection();
    
    bool connect();
    void disconnect();
    bool is_connected() const;
    
    // String operations
    bool set(const std::string& key, const std::string& value, 
             std::optional<int> expiry = std::nullopt);
    std::optional<std::string> get(const std::string& key);
    bool del(const std::string& key);
    bool exists(const std::string& key);
    
    // Hash operations
    bool hset(const std::string& key, const std::string& field, 
              const std::string& value);
    std::optional<std::string> hget(const std::string& key, 
                                      const std::string& field);
    std::map<std::string, std::string> hgetall(const std::string& key);
    bool hdel(const std::string& key, const std::string& field);
    
    // List operations
    bool lpush(const std::string& key, const std::string& value);
    std::vector<std::string> lrange(const std::string& key, 
                                      int start, int end);
    
    // Pub/Sub
    bool publish(const std::string& channel, const std::string& message);
    
    // Expiry
    bool expire(const std::string& key, int seconds);
    
private:
    Config config_;
    void* conn_ = nullptr; // redisContext*
};

// ============================================================================
// Connection Pool
// ============================================================================

template<typename ConnectionType>
class ConnectionPool {
public:
    ConnectionPool(const Config& config, size_t pool_size);
    ~ConnectionPool();
    
    // Get a connection from the pool
    std::shared_ptr<ConnectionType> get_connection();
    
    // Return a connection to the pool
    void return_connection(std::shared_ptr<ConnectionType> conn);
    
    // Health check
    void health_check();
    
    // Stats
    size_t available_connections() const;
    size_t total_connections() const;
    
private:
    Config config_;
    size_t pool_size_;
    std::queue<std::shared_ptr<ConnectionType>> pool_;
    std::mutex mutex_;
    std::atomic<size_t> active_connections_{0};
    
    std::shared_ptr<ConnectionType> create_connection();
    void destroy_connection(std::shared_ptr<ConnectionType> conn);
};

// ============================================================================
// Database Manager
// ============================================================================

class DatabaseManager {
public:
    static DatabaseManager& instance();
    
    void initialize(const Config& config);
    void shutdown();
    
    // PostgreSQL
    std::shared_ptr<PGConnection> get_pg();
    void return_pg(std::shared_ptr<PGConnection> conn);
    
    // Redis
    std::shared_ptr<RedisConnection> get_redis();
    void return_redis(std::shared_ptr<RedisConnection> conn);
    
    // Raw pointers for performance-critical code
    PGConnection* get_pg_raw();
    RedisConnection* get_redis_raw();
    
    // Admin operations
    bool create_tables();
    bool run_migrations();
    
private:
    DatabaseManager() = default;
    ~DatabaseManager();
    DatabaseManager(const DatabaseManager&) = delete;
    DatabaseManager& operator=(const DatabaseManager&) = delete;
    
    Config config_;
    ConnectionPool<PGConnection>* pg_pool_ = nullptr;
    ConnectionPool<RedisConnection>* redis_pool_ = nullptr;
};

// Convenience macros
#define PG_DB tiger::admin::DatabaseManager::instance().get_pg_raw()
#define REDIS tiger::admin::DatabaseManager::instance().get_redis_raw()

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_CONNECTION_POOL_HPP
