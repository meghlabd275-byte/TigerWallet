/**
 * TigerAdmin C++ Core - Cache Header
 */
#pragma once

#include "admin_security.hpp"
#include "admin_config.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>

namespace tiger {
namespace admin {

class CacheService {
public:
    static CacheService& instance();

    void initialize();
    void shutdown();

    bool set(const std::string& key, const std::string& value,
             std::optional<int> ttl_seconds = std::nullopt);
    std::optional<std::string> get(const std::string& key);
    bool del(const std::string& key);
    bool exists(const std::string& key);

    bool hset(const std::string& key, const std::string& field,
              const std::string& value);
    std::optional<std::string> hget(const std::string& key,
                                    const std::string& field);
    std::map<std::string, std::string> hgetall(const std::string& key);
    bool hdel(const std::string& key, const std::string& field);
    bool hdel_all(const std::string& key);

    bool lpush(const std::string& key, const std::string& value);
    std::vector<std::string> lrange(const std::string& key, int start, int end);

    bool zadd(const std::string& key, const std::string& member, double score);
    std::vector<std::string> zrange(const std::string& key, int start, int end);
    std::vector<std::string> zrevrange(const std::string& key, int start, int end);

    bool expire(const std::string& key, int seconds);
    int64_t ttl(const std::string& key);

    bool flush_db();
    bool flush_pattern(const std::string& pattern);
    std::vector<std::string> keys(const std::string& pattern);
};

class RateLimiter {
public:
    static RateLimiter& instance();

    void initialize();

    bool is_allowed(const std::string& identifier, int max_requests,
                    int window_seconds);
    int get_remaining(const std::string& identifier, int max_requests,
                      int window_seconds);
    int64_t get_reset_time(const std::string& identifier, int window_seconds);
    void reset(const std::string& identifier);
    void clear();
};

} // namespace admin
} // namespace tiger
