#pragma once

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <memory>
#include <functional>
#include <mutex>
#include <hiredis/hiredis.h>

namespace tiger {

class RedisClient {
public:
    RedisClient(const std::string& host, int port, const std::string& password = "",
                int db = 0, int min_connections = 5, int max_connections = 50);
    ~RedisClient();
    
    // Connection management
    bool connect();
    void disconnect();
    bool is_connected() const;
    bool ping();
    
    // String operations
    bool set(const std::string& key, const std::string& value);
    bool setex(const std::string& key, const std::string& value, int64_t ttl_seconds);
    bool setnx(const std::string& key, const std::string& value);
    std::optional<std::string> get(const std::string& key);
    std::vector<std::string> mget(const std::vector<std::string>& keys);
    bool del(const std::string& key);
    bool del(const std::vector<std::string>& keys);
    bool exists(const std::string& key);
    int64_t incr(const std::string& key);
    int64_t incrby(const std::string& key, int64_t increment);
    int64_t decr(const std::string& key);
    int64_t decrby(const std::string& key, int64_t decrement);
    
    // Hash operations
    bool hset(const std::string& key, const std::string& field, const std::string& value);
    bool hmset(const std::string& key, const std::map<std::string, std::string>& values);
    std::optional<std::string> hget(const std::string& key, const std::string& field);
    std::map<std::string, std::string> hgetall(const std::string& key);
    std::vector<std::string> hmget(const std::string& key, const std::vector<std::string>& fields);
    bool hexists(const std::string& key, const std::string& field);
    int64_t hdel(const std::string& key, const std::string& field);
    int64_t hdel(const std::string& key, const std::vector<std::string>& fields);
    int64_t hlen(const std::string& key);
    int64_t h incr(const std::string& key, const std::string& field, int64_t increment);
    
    // List operations
    bool lpush(const std::string& key, const std::string& value);
    bool rpush(const std::string& key, const std::string& value);
    std::optional<std::string> lpop(const std::string& key);
    std::optional<std::string> rpop(const std::string& key);
    std::vector<std::string> lrange(const std::string& key, int64_t start, int64_t stop);
    int64_t llen(const std::string& key);
    bool ltrim(const std::string& key, int64_t start, int64_t stop);
    
    // Set operations
    bool sadd(const std::string& key, const std::string& member);
    bool sadd(const std::string& key, const std::vector<std::string>& members);
    bool srem(const std::string& key, const std::string& member);
    bool srem(const std::string& key, const std::vector<std::string>& members);
    std::vector<std::string> smembers(const std::string& key);
    bool sismember(const std::string& key, const std::string& member);
    int64_t scard(const std::string& key);
    std::vector<std::string> sinter(const std::vector<std::string>& keys);
    std::vector<std::string> sunion(const std::vector<std::string>& keys);
    
    // Sorted set operations
    bool zadd(const std::string& key, double score, const std::string& member);
    bool zadd(const std::string& key, const std::map<std::string, double>& members);
    bool zrem(const std::string& key, const std::string& member);
    std::vector<std::string> zrange(const std::string& key, int64_t start, int64_t stop);
    std::vector<std::string> zrevrange(const std::string& key, int64_t start, int64_t stop);
    std::vector<std::pair<std::string, double>> zrange_with_scores(const std::string& key, int64_t start, int64_t stop);
    std::vector<std::pair<std::string, double>> zrevrange_with_scores(const std::string& key, int64_t start, int64_t stop);
    std::optional<double> zscore(const std::string& key, const std::string& member);
    int64_t zrank(const std::string& key, const std::string& member);
    int64_t zrevrank(const std::string& key, const std::string& member);
    int64_t zcard(const std::string& key);
    int64_t zcount(const std::string& key, double min, double max);
    
    // Key operations
    bool expire(const std::string& key, int64_t ttl_seconds);
    bool persist(const std::string& key);
    int64_t ttl(const std::string& key);
    std::vector<std::string> keys(const std::string& pattern);
    std::vector<std::string> scan(const std::string& cursor, const std::string& pattern = "", int count = 100);
    
    // Pub/Sub
    bool publish(const std::string& channel, const std::string& message);
    class Subscriber {
    public:
        Subscriber(redisContext* ctx);
        ~Subscriber();
        
        bool subscribe(const std::string& channel);
        bool psubscribe(const std::string& pattern);
        bool unsubscribe(const std::string& channel);
        bool punsubscribe(const std::string& pattern);
        std::optional<std::pair<std::string, std::string>> listen();
        
    private:
        redisContext* ctx_;
    };
    std::unique_ptr<Subscriber> subscribe();
    
    // Pipeline
    class Pipeline {
    public:
        Pipeline(RedisClient* client);
        ~Pipeline();
        
        Pipeline& set(const std::string& key, const std::string& value);
        Pipeline& setex(const std::string& key, const std::string& value, int64_t ttl);
        Pipeline& get(const std::string& key);
        Pipeline& del(const std::string& key);
        Pipeline& hset(const std::string& key, const std::string& field, const std::string& value);
        Pipeline& hget(const std::string& key, const std::string& field);
        
        std::vector<std::optional<std::string>> execute();
        
    private:
        RedisClient* client_;
        std::vector<std::string> commands_;
    };
    std::unique_ptr<Pipeline> pipeline();
    
    // Lua scripting
    std::optional<std::string> eval(const std::string& script, const std::vector<std::string>& keys,
                                    const std::vector<std::string>& args);
    
    // Transaction
    bool multi();
    bool exec();
    bool discard();
    
    // Connection pool
    void set_min_connections(int min);
    void set_max_connections(int max);
    int active_connections() const;
    int idle_connections() const;
    
    // Health check
    bool health_check();
    
private:
    struct Connection {
        redisContext* ctx;
        bool in_use;
        std::chrono::steady_clock::time_point last_used;
    };
    
    std::vector<std::unique_ptr<Connection>> pool_;
    std::mutex pool_mutex_;
    std::condition_variable pool_cv_;
    
    std::string host_;
    int port_;
    std::string password_;
    int db_;
    int min_connections_;
    int max_connections_;
    bool connected_;
    
    Connection* acquire_connection();
    void release_connection(Connection* conn);
};

// Redis-based rate limiter
class RateLimiter {
public:
    RateLimiter(std::shared_ptr<RedisClient> redis);
    ~RateLimiter() = default;
    
    // Sliding window rate limiter
    bool allow(const std::string& key, int max_requests, int window_seconds);
    
    // Fixed window rate limiter
    bool allow_fixed(const std::string& key, int max_requests, int window_seconds);
    
    // Token bucket rate limiter
    bool allow_token(const std::string& key, int capacity, int refill_rate);
    
    // Get current count
    int64_t get_count(const std::string& key, int window_seconds);
    
    // Reset
    void reset(const std::string& key);
    
private:
    std::shared_ptr<RedisClient> redis_;
};

// Redis-based distributed lock
class DistributedLock {
public:
    DistributedLock(std::shared_ptr<RedisClient> redis);
    ~DistributedLock() = default;
    
    bool acquire(const std::string& key, int64_t ttl_seconds, const std::string& value = "");
    bool release(const std::string& key, const std::string& value = "");
    bool extend(const std::string& key, int64_t ttl_seconds, const std::string& value = "");
    
    class Guard {
    public:
        Guard(DistributedLock* lock, const std::string& key, int64_t ttl);
        ~Guard();
        
        bool is_acquired() const { return acquired_; }
        
    private:
        DistributedLock* lock_;
        std::string key_;
        std::string value_;
        bool acquired_;
    };
    
    std::unique_ptr<Guard> guard(const std::string& key, int64_t ttl_seconds);
    
private:
    std::shared_ptr<RedisClient> redis_;
};

} // namespace tiger
