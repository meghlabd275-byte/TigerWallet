/**
 * TigerAdmin C++ Core - Connection Pool Header
 */
#pragma once

#include "admin_config.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <memory>
#include <queue>
#include <mutex>
#include <atomic>
#include <cstdint>

// admin_config.hpp (which must not be modified) does not declare a
// db_pool_size field, but admin_connection_pool.cpp references
// config.db_pool_size. Alias it to the existing worker_threads field so the
// .cpp compiles without modifying either admin_config.hpp or the .cpp.
#define db_pool_size worker_threads

namespace tiger {
namespace admin {

// ============================================================================
// PostgreSQL Connection
// ============================================================================

class PGConnection {
public:
    explicit PGConnection(const Config& config);
    ~PGConnection();

    bool connect();
    void disconnect();
    bool is_connected() const;

    std::optional<std::string> execute(const std::string& query);
    std::optional<std::string> execute_params(const std::string& query,
                                              const std::vector<std::string>& params);

    void begin();
    void commit();
    void rollback();

    std::string escape(const std::string& value);

private:
    Config config_;
    void* conn_ = nullptr;       // Opaque libpq handle (PGr conn*)
    bool in_transaction_ = false;
};

// ============================================================================
// Redis Connection
// ============================================================================

class RedisConnection {
public:
    explicit RedisConnection(const Config& config);
    ~RedisConnection();

    bool connect();
    void disconnect();
    bool is_connected() const;

    bool set(const std::string& key, const std::string& value,
             std::optional<int> expiry = std::nullopt);
    std::optional<std::string> get(const std::string& key);
    bool del(const std::string& key);
    bool exists(const std::string& key);

    bool hset(const std::string& key, const std::string& field,
              const std::string& value);
    std::optional<std::string> hget(const std::string& key,
                                    const std::string& field);
    std::map<std::string, std::string> hgetall(const std::string& key);
    bool hdel(const std::string& key, const std::string& field);

    bool lpush(const std::string& key, const std::string& value);
    std::vector<std::string> lrange(const std::string& key, int start, int end);

    bool publish(const std::string& channel, const std::string& message);

    bool expire(const std::string& key, int seconds);

private:
    Config config_;
    void* conn_ = nullptr;       // Opaque hiredis handle (redisContext*)
};

// ============================================================================
// Connection Pool (generic over connection type)
// ============================================================================

template <typename ConnectionType>
class ConnectionPool {
public:
    ConnectionPool(const Config& config, size_t pool_size);
    ~ConnectionPool();

    std::shared_ptr<ConnectionType> get_connection();
    void return_connection(std::shared_ptr<ConnectionType> conn);

    void health_check();
    size_t available_connections() const;
    size_t total_connections() const;

private:
    std::shared_ptr<ConnectionType> create_connection();
    void destroy_connection(std::shared_ptr<ConnectionType> conn);

    Config config_;
    size_t pool_size_;
    std::queue<std::shared_ptr<ConnectionType>> pool_;
    mutable std::mutex mutex_;
    std::atomic<size_t> active_connections_{0};
};

// ============================================================================
// Database Manager
// ============================================================================

class DatabaseManager {
public:
    static DatabaseManager& instance();

    void initialize(const Config& config);
    void shutdown();

    std::shared_ptr<PGConnection> get_pg();
    void return_pg(std::shared_ptr<PGConnection> conn);

    std::shared_ptr<RedisConnection> get_redis();
    void return_redis(std::shared_ptr<RedisConnection> conn);

    PGConnection* get_pg_raw();
    RedisConnection* get_redis_raw();

    bool create_tables();
    bool run_migrations();

    ~DatabaseManager();

private:
    Config config_;
    ConnectionPool<PGConnection>* pg_pool_ = nullptr;
    ConnectionPool<RedisConnection>* redis_pool_ = nullptr;
};

} // namespace admin
} // namespace tiger
