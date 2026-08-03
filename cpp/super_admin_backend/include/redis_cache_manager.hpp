/**
 * TigerWallet Super Admin - Redis Cache Manager
 * High-performance in-memory caching with Redis backend
 * Ultra-low latency design for distributed systems
 * 
 * Features:
 * - Redis connection pooling
 * - In-memory LRU cache with Redis persistence
 * - Pub/Sub support for real-time updates
 * - Cluster support
 * - Automatic failover
 */

#ifndef REDIS_CACHE_MANAGER_HPP
#define REDIS_CACHE_MANAGER_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <mutex>
#include <atomic>
#include <chrono>
#include <functional>
#include <optional>
#include <hiredis/hiredis.h>
#include <hiredis/async.h>

namespace tigerwallet {
namespace super_admin {

// ============================================================================
// Configuration
// ============================================================================

struct RedisConfig {
    std::string host = "localhost";
    int port = 6379;
    std::string password = "";
    int database = 0;
    int pool_size = 10;
    int connection_timeout = 5;
    int command_timeout = 5;
    bool use_ssl = false;
    bool enable_cluster = false;
    std::vector<std::pair<std::string, int>> cluster_nodes;
};

struct CacheConfig {
    size_t max_memory_mb = 512;
    int max_entries = 100000;
    int ttl_default = 3600;  // 1 hour
    int cleanup_interval = 300;  // 5 minutes
    bool enable_persistence = true;
    std::string persistence_prefix = "tigerwallet:";
};

// ============================================================================
// Cache Entry
// ============================================================================

struct CacheEntry {
    std::string key;
    std::string value;
    std::chrono::steady_clock::time_point created_at;
    std::chrono::steady_clock::time_point last_accessed;
    int ttl;
    bool is_persistent;
    
    bool is_expired() const {
        if (ttl <= 0) return false;
        auto now = std::chrono::steady_clock::now();
        auto age = std::chrono::duration_cast<std::chrono::seconds>(
            now - created_at
        ).count();
        return age > ttl;
    }
    
    size_t size() const {
        return key.size() + value.size() + sizeof(CacheEntry);
    }
};

// ============================================================================
// LRU Cache Implementation
// ============================================================================

class LRUCache {
public:
    LRUCache(size_t max_size, size_t max_memory_bytes);
    ~LRUCache();
    
    bool put(const std::string& key, const std::string& value, int ttl = -1);
    std::optional<std::string> get(const std::string& key);
    bool remove(const std::string& key);
    bool exists(const std::string& key);
    void clear();
    
    size_t size() const;
    size_t memory_usage() const;
    void cleanup_expired();
    
    std::vector<std::string> get_keys(const std::string& pattern = "*");
    int get_ttl(const std::string& key);
    bool set_ttl(const std::string& key, int ttl);
    
private:
    struct Node {
        std::string key;
        std::string value;
        std::chrono::steady_clock::time_point created_at;
        std::chrono::steady_clock::time_point last_accessed;
        int ttl;
        Node* prev;
        Node* next;
    };
    
    size_t max_size_;
    size_t max_memory_bytes_;
    size_t current_memory_;
    
    std::map<std::string, Node*> cache_;
    Node* head_;
    Node* tail_;
    std::mutex mutex_;
    
    void detach(Node* node);
    void attach_to_head(Node* node);
    void remove_lru();
};

// ============================================================================
// Redis Connection
// ============================================================================

class RedisConnection {
public:
    RedisConnection(redisContext* ctx, uint64_t id);
    ~RedisConnection();
    
    RedisConnection(const RedisConnection&) = delete;
    RedisConnection& operator=(const RedisConnection&) = delete;
    
    RedisConnection(RedisConnection&& other) noexcept;
    RedisConnection& operator=(RedisConnection&& other) noexcept;
    
    redisContext* get() { return ctx_; }
    uint64_t id() const { return id_; }
    bool is_connected() const;
    
    // Sync commands
    redisReply* command(const char* format, ...);
    redisReply* command_argv(int argc, const char** argv, const size_t* argvlen);
    
private:
    redisContext* ctx_;
    uint64_t id_;
};

// ============================================================================
// Redis Connection Pool
// ============================================================================

class RedisConnectionPool {
public:
    RedisConnectionPool(const RedisConfig& config);
    ~RedisConnectionPool();
    
    bool initialize();
    std::shared_ptr<RedisConnection> acquire();
    void release(std::shared_ptr<RedisConnection> conn);
    
    size_t get_active_connections() const;
    size_t get_idle_connections() const;
    bool health_check();
    void shutdown();
    
private:
    RedisConfig config_;
    std::vector<std::shared_ptr<RedisConnection>> connections_;
    std::vector<uint64_t> available_ids_;
    std::mutex mutex_;
    std::condition_variable cv_;
    std::atomic<bool> initialized_;
    std::atomic<uint64_t> next_id_;
    
    redisContext* create_connection();
    bool validate_connection(redisContext* ctx);
};

// ============================================================================
// Cache Manager
// ============================================================================

class RedisCacheManager {
public:
    static RedisCacheManager& getInstance();
    
    // Configuration
    void configure(const RedisConfig& redis_config);
    void configure_cache(const CacheConfig& cache_config);
    
    // Initialization
    bool initialize();
    void shutdown();
    
    // Cache operations
    bool set(const std::string& key, const std::string& value, int ttl = -1);
    bool set(const std::string& key, const std::vector<std::string>& values, int ttl = -1);
    std::optional<std::string> get(const std::string& key);
    std::vector<std::optional<std::string>> get(const std::vector<std::string>& keys);
    bool exists(const std::string& key);
    bool remove(const std::string::size_type key);
    int remove(const std::vector<std::string>& keys);
    void clear();
    
    // Expire operations
    bool expire(const std::string& key, int ttl);
    int ttl(const std::string& key);
    bool persist(const std::string& key);
    
    // Key patterns
    std::vector<std::string> keys(const std::string& pattern);
    std::vector<std::string> scan(const std::string& pattern, size_t count = 100);
    
    // Hash operations
    bool hset(const std::string& key, const std::string& field, const std::string& value);
    bool hset(const std::string& key, const std::map<std::string, std::string>& values);
    std::optional<std::string> hget(const std::string& key, const std::string& field);
    std::map<std::string, std::string> hgetall(const std::string& key);
    std::vector<std::optional<std::string>> hmget(const std::string& key, const std::vector<std::string>& fields);
    bool hexists(const std::string& key, const std::string& field);
    int hdel(const std::string& key, const std::vector<std::string>& fields);
    int hlen(const std::string& key);
    
    // List operations
    int lpush(const std::string& key, const std::string& value);
    int rpush(const std::string& key, const std::string& value);
    std::optional<std::string> lpop(const std::string& key);
    std::optional<std::string> rpop(const std::string& key);
    std::vector<std::string> lrange(const std::string& key, int start, int stop);
    int llen(const std::string& key);
    
    // Set operations
    bool sadd(const std::string& key, const std::string& member);
    int sadd(const std::string& key, const std::vector<std::string>& members);
    bool sismember(const std::string& key, const std::string& member);
    std::vector<std::string> smembers(const std::string& key);
    int srem(const std::string& key, const std::vector<std::string>& members);
    int scard(const std::string& key);
    
    // Sorted set operations
    bool zadd(const std::string& key, double score, const std::string& member);
    int zadd(const std::string& key, const std::map<double, std::string>& members);
    std::vector<std::string> zrange(const std::string& key, int start, int stop);
    std::vector<std::string> zrevrange(const std::string& key, int start, int stop);
    std::optional<double> zscore(const std::string& key, const std::string& member);
    int zrem(const std::string& key, const std::vector<std::string>& members);
    std::optional<int> zrank(const std::string& key, const std::string& member);
    
    // Counter operations
    int64_t incr(const std::string& key);
    int64_t incrby(const std::string& key, int64_t increment);
    int64_t decr(const std::string& key);
    int64_t decrby(const std::string& key, int64_t decrement);
    
    // Pub/Sub
    using MessageCallback = std::function<void(const std::string& channel, const std::string& message)>;
    bool publish(const std::string& channel, const std::string& message);
    bool subscribe(const std::string& channel, MessageCallback callback);
    bool unsubscribe(const std::string& channel);
    
    // Pipeline
    class Pipeline {
    public:
        Pipeline(RedisCacheManager& manager);
        Pipeline& set(const std::string& key, const std::string& value, int ttl = -1);
        Pipeline& get(const std::string& key);
        Pipeline& del(const std::string& key);
        Pipeline& hset(const std::string& key, const std::string& field, const std::string& value);
        Pipeline& hget(const std::string& key, const std::string& field);
        std::vector<std::optional<std::string>> execute();
        
    private:
        RedisCacheManager& manager_;
        std::vector<std::string> commands_;
    };
    
    Pipeline pipeline();
    
    // Stats
    struct Stats {
        size_t memory_usage;
        size_t entry_count;
        size_t hit_count;
        size_t miss_count;
        double hit_rate;
        size_t redis_connected;
    };
    
    Stats get_stats() const;
    void reset_stats();

private:
    RedisCacheManager();
    ~RedisCacheManager();
    
    RedisCacheManager(const RedisCacheManager&) = delete;
    RedisCacheManager& operator=(const RedisCacheManager&) = delete;
    
    RedisConfig redis_config_;
    CacheConfig cache_config_;
    std::unique_ptr<RedisConnectionPool> pool_;
    std::unique_ptr<LRUCache> lru_cache_;
    std::atomic<bool> initialized_;
    
    std::atomic<size_t> hit_count_{0};
    std::atomic<size_t> miss_count_{0};
    
    std::string build_key(const std::string& key) const;
    std::string extract_key(const std::string& full_key) const;
    void update_stats(bool hit);
};

// Inline implementations

inline LRUCache::LRUCache(size_t max_size, size_t max_memory_bytes)
    : max_size_(max_size), max_memory_bytes_(max_memory_bytes), current_memory_(0) {
    head_ = new Node{"", "", {}, {}, 0, nullptr, nullptr};
    tail_ = new Node{"", "", {}, {}, 0, nullptr, nullptr};
    head_->next = tail_;
    tail_->prev = head_;
}

inline LRUCache::~LRUCache() {
    clear();
    delete head_;
    delete tail_;
}

inline size_t LRUCache::size() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return cache_.size();
}

inline size_t LRUCache::memory_usage() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return current_memory_;
}

}  // namespace super_admin
}  // namespace tigerwallet

#endif  // REDIS_CACHE_MANAGER_HPP
