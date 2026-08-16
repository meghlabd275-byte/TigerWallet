/**
 * TigerAdmin C++ Core - Rate Limiter Header
 */
#pragma once

#include "admin_security.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <chrono>
#include <mutex>
#include <atomic>
#include <cstdint>

namespace tiger {
namespace admin {

class RateLimiterService {
public:
    struct RateLimitResult {
        bool allowed = false;
        int remaining = 0;
        int64_t reset_time = 0;
        std::string retry_after;
    };

    struct Entry {
        std::atomic<int> count{0};
        std::atomic<int64_t> window_start{0};
    };

    static RateLimiterService& instance();

    void initialize();

    RateLimitResult check_rate_limit(const std::string& identifier,
                                     const std::string& endpoint,
                                     int max_requests, int window_seconds);
    bool is_allowed(const std::string& identifier, const std::string& endpoint,
                    int max_requests, int window_seconds);
    int get_current_usage(const std::string& identifier,
                          const std::string& endpoint, int window_seconds);

    void reset(const std::string& identifier);
    void reset_endpoint(const std::string& identifier,
                        const std::string& endpoint);
    void cleanup(int max_age_seconds);

    int64_t get_current_time();
    std::string make_key(const std::string& identifier,
                         const std::string& endpoint);

private:
    std::mutex mutex_;
    std::map<std::string, Entry> entries_;
};

class IPRateLimiter {
public:
    static IPRateLimiter& instance();

    void initialize();
    void initialize(int max_requests, int window_seconds);

    bool is_allowed(const std::string& ip_address);
    RateLimiterService::RateLimitResult check(const std::string& ip_address);

    bool is_blacklisted(const std::string& ip_address);
    void blacklist_ip(const std::string& ip_address);
    void unblacklist_ip(const std::string& ip_address);
    std::vector<std::string> get_blacklist();

private:
    int max_requests_ = 100;
    int window_seconds_ = 60;
    std::mutex mutex_;
    std::vector<std::string> blacklist_;
};

class AdminActionRateLimiter {
public:
    static AdminActionRateLimiter& instance();

    void initialize();

    bool allow_action(AdminID admin_id, const std::string& action);
    int get_remaining(AdminID admin_id, const std::string& action);
    void set_limits(const std::string& action, int max_per_hour, int max_per_day);
};

} // namespace admin
} // namespace tiger
