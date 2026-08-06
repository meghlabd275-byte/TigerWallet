/**
 * TigerAdmin C++ Core - Cache Manager
 */

#ifndef TIGER_ADMIN_CACHE_HPP
#define TIGER_ADMIN_CACHE_HPP

#include <string>
#include <optional>
#include <chrono>
#include <memory>

namespace tiger {
namespace admin {

// ============================================================================
// Cache Service
// ============================================================================

class CacheService {
public:
    static CacheService& instance();
    
    void initialize();
    void shutdown();
    
    // String cache
    bool set(const std::string& key, const std::string& value,
             std::optional<int> ttl_seconds = std::nullopt);
    std::optional<std::string> get(const std::string& key);
    bool del(const std::string& key);
    bool exists(const std::string& key);
    
    // Hash cache
    bool hset(const std::string& key, const std::string& field,
              const std::string& value);
    std::optional<std::string> hget(const std::string& key,
                                     const std::string& field);
    std::map<std::string, std::string> hgetall(const std::string& key);
    bool hdel(const std::string& key, const std::string& field);
    bool hdel_all(const std::string& key);
    
    // List cache
    bool lpush(const std::string& key, const std::string& value);
    std::vector<std::string> lrange(const std::string& key,
                                    int start, int end);
    
    // Sorted set
    bool zadd(const std::string& key, const std::string& member, double score);
    std::vector<std::string> zrange(const std::string& key,
                                     int start, int end);
    std::vector<std::string> zrevrange(const std::string& key,
                                        int start, int end);
    
    // Expiry
    bool expire(const std::string& key, int seconds);
    int64_t ttl(const std::string& key);
    
    // Flush
    bool flush_db();
    bool flush_pattern(const std::string& pattern);
    
    // Keys
    std::vector<std::string> keys(const std::string& pattern);
    
private:
    CacheService() = default;
};

// ============================================================================
// Rate Limiter
// ============================================================================

class RateLimiter {
public:
    static RateLimiter& instance();
    
    void initialize();
    
    // Check if request is allowed
    bool is_allowed(const std::string& identifier,
                    int max_requests,
                    int window_seconds);
    
    // Get remaining requests
    int get_remaining(const std::string& identifier,
                      int max_requests,
                      int window_seconds);
    
    // Get reset time
    int64_t get_reset_time(const std::string& identifier,
                           int window_seconds);
    
    // Reset
    void reset(const std::string& identifier);
    
    // Clear all
    void clear();
    
private:
    RateLimiter() = default;
    
    struct RateLimitEntry {
        int count;
        int64_t window_start;
    };
    
    std::mutex mutex_;
    std::map<std::string, RateLimitEntry> entries_;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_CACHE_HPP
