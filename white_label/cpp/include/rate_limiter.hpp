#pragma once

#include <chrono>
#include <mutex>
#include <unordered_map>
#include <string>
#include <memory>
#include <atomic>
#include <optional>
#include <vector>
#include <functional>

namespace tigerwallet {
namespace highspeed {

/**
 * Token Bucket Rate Limiter - Ultra Low Latency Implementation
 * Uses lock-free operations where possible for maximum throughput
 */
class TokenBucketRateLimiter {
public:
    struct BucketConfig {
        size_t max_tokens;
        size_t refill_rate;  // tokens per second
        size_t burst_size;
    };

    explicit TokenBucketLimiter(const BucketConfig& config);
    ~TokenBucketLimiter() = default;

    // Non-blocking check and consume
    bool try_consume(const std::string& key, size_t tokens = 1);
    
    // Blocking consume with timeout
    bool consume(const std::string& key, size_t tokens = 1, 
                std::chrono::milliseconds timeout = std::chrono::milliseconds(100));
    
    // Get remaining tokens
    size_t available(const std::string& key) const;
    
    // Reset bucket for key
    void reset(const std::string& key);
    
    // Get current usage stats
    struct UsageStats {
        size_t total_requests;
        size_t allowed_requests;
        size_t rejected_requests;
        double current_rate;
    };
    
    UsageStats get_stats() const;
    UsageStats get_stats(const std::string& key) const;

private:
    struct Bucket {
        std::atomic<size_t> tokens;
        std::atomic<int64_t> last_refill;
        std::atomic<size_t> requests;
        std::atomic<size_t> allowed;
        std::atomic<size_t> rejected;
    };

    BucketConfig config_;
    mutable std::mutex mutex_;
    std::unordered_map<std::string, std::unique_ptr<Bucket>> buckets_;
    
    Bucket& get_or_create_bucket(const std::string& key);
    void refill_bucket(Bucket& bucket);
    int64_t current_timestamp_ms() const;
};

/**
 * Sliding Window Rate Limiter - More Accurate Rate Limiting
 */
class SlidingWindowRateLimiter {
public:
    struct WindowConfig {
        size_t max_requests;
        std::chrono::milliseconds window_size;
        size_t max_buckets;
    };

    explicit SlidingWindowRateLimiter(const WindowConfig& config);
    ~SlidingWindowRateLimiter() = default;

    bool is_allowed(const std::string& key);
    size_t get_request_count(const std::string& key) const;
    void clear();

private:
    struct Window {
        std::vector<int64_t> timestamps;
        size_t max_size;
    };

    WindowConfig config_;
    mutable std::mutex mutex_;
    std::unordered_map<std::string, Window> windows_;
    
    void cleanup_old_requests(Window& window, int64_t now);
};

/**
 * Connection Pool for High-Speed Database Connections
 */
class ConnectionPool {
public:
    struct ConnectionConfig {
        std::string host;
        uint16_t port;
        std::string database;
        std::string username;
        std::string password;
        size_t min_connections;
        size_t max_connections;
        std::chrono::milliseconds connection_timeout;
        std::chrono::milliseconds idle_timeout;
    };

    explicit ConnectionPool(const ConnectionConfig& config);
    ~ConnectionPool();

    // Connection management
    class Connection {
    public:
        ~Connection();
        
        bool execute(const std::string& query);
        std::optional<std::string> query(const std::string& query);
        bool is_connected() const;
        void reset();
        
    private:
        friend class ConnectionPool;
        Connection(int fd, const ConnectionConfig& config);
        int fd_;
        const ConnectionConfig& config_;
        std::chrono::steady_clock::time_point last_used_;
    };

    std::shared_ptr<Connection> acquire();
    void release(std::shared_ptr<Connection> conn);
    
    // Pool stats
    struct PoolStats {
        size_t total_connections;
        size_t active_connections;
        size_t idle_connections;
        size_t waiting_requests;
    };
    
    PoolStats get_stats() const;

private:
    ConnectionConfig config_;
    mutable std::mutex mutex_;
    std::vector<std::shared_ptr<Connection>> idle_connections_;
    std::atomic<size_t> total_connections_{0};
    std::atomic<size_t> waiting_requests_{0};
};

/**
 * LRU Cache - High Performance In-Memory Cache
 */
class LRUCache {
public:
    struct CacheConfig {
        size_t max_size;
        std::chrono::milliseconds default_ttl;
        size_t shard_count;
    };

    explicit LRUCache(const CacheConfig& config);
    ~LRUCache() = default;

    // Basic operations
    void put(const std::string& key, const std::string& value);
    void put(const std::string& key, const std::string& value, std::chrono::milliseconds ttl);
    
    std::optional<std::string> get(const std::string& key);
    bool exists(const std::string& key) const;
    
    void remove(const std::string& key);
    void clear();
    
    // Stats
    struct CacheStats {
        size_t size;
        size_t hits;
        size_t misses;
        double hit_rate;
        size_t evictions;
    };
    
    CacheStats get_stats() const;

private:
    struct CacheEntry {
        std::string value;
        int64_t expiry;
        std::atomic<size_t> access_count;
        std::atomic<int64_t> last_access;
    };

    CacheConfig config_;
    mutable std::mutex mutex_;
    std::unordered_map<std::string, std::unique_ptr<CacheEntry>> cache_;
    
    // Stats
    std::atomic<size_t> hits_{0};
    std::atomic<size_t> misses_{0};
    std::atomic<size_t> evictions_{0};
    
    void evict_if_needed();
    bool is_expired(const CacheEntry& entry) const;
};

/**
 * Bloom Filter - High-Speed Membership Testing
 */
class BloomFilter {
public:
    explicit BloomFilter(size_t expected_items, double false_positive_rate);
    ~BloomFilter() = default;

    void add(const std::string& key);
    bool might_contain(const std::string& key) const;
    void clear();
    
    size_t size() const { return size_; }
    double estimated_false_positive_rate() const;

private:
    size_t size_;
    size_t hash_count_;
    std::vector<bool> bits_;
    mutable std::mutex mutex_;
    
    std::vector<size_t> get_hash_positions(const std::string& key) const;
};

/**
 * High-Speed Circular Buffer for Analytics
 */
class CircularBuffer {
public:
    explicit CircularBuffer(size_t capacity);
    ~CircularBuffer() = default;

    void push(double value);
    double get(size_t index) const;
    size_t size() const { return size_; }
    size_t capacity() const { return capacity_; }
    
    // Aggregate functions
    double sum() const;
    double avg() const;
    double min() const;
    double max() const;
    double percentile(double p) const;
    double stddev() const;

private:
    size_t capacity_;
    size_t head_{0};
    size_t size_{0};
    std::vector<double> buffer_;
    mutable std::mutex mutex_;
};

/**
 * Lock-Free Queue for Async Processing
 */
template<typename T>
class LockFreeQueue {
public:
    explicit LockFreeQueue(size_t capacity);
    ~LockFreeQueue();

    bool push(const T& value);
    bool pop(T& value);
    bool is_empty() const;
    bool is_full() const;
    size_t size() const;

private:
    struct Node {
        T data;
        std::atomic<Node*> next;
    };

    size_t capacity_;
    std::atomic<Node*> head_;
    std::atomic<Node*> tail_;
    std::atomic<size_t> count_;
};

} // namespace highspeed
} // namespace tigerwallet
