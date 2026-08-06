/**
 * TigerAdmin C++ Core - Rate Limiter
 */

#ifndef TIGER_ADMIN_RATE_LIMITER_HPP
#define TIGER_ADMIN_RATE_LIMITER_HPP

#include <string>
#include <map>
#include <mutex>
#include <atomic>
#include <chrono>
#include <optional>

namespace tiger {
namespace admin {

// ============================================================================
// Rate Limiter
// ============================================================================

class RateLimiterService {
public:
    static RateLimiterService& instance();
    
    void initialize();
    
    // Check if request is allowed
    struct RateLimitResult {
        bool allowed;
        int remaining;
        int64_t reset_time;
        std::string retry_after;
    };
    
    RateLimitResult check_rate_limit(const std::string& identifier,
                                     const std::string& endpoint,
                                     int max_requests,
                                     int window_seconds);
    
    // Simple check (returns just allowed/not allowed)
    bool is_allowed(const std::string& identifier,
                    const std::string& endpoint,
                    int max_requests,
                    int window_seconds);
    
    // Get current usage
    int get_current_usage(const std::string& identifier,
                          const std::string& endpoint,
                          int window_seconds);
    
    // Reset
    void reset(const std::string& identifier);
    void reset_endpoint(const std::string& identifier,
                        const std::string& endpoint);
    
    // Cleanup old entries
    void cleanup(int max_age_seconds = 3600);
    
private:
    RateLimiterService() = default;
    
    struct RateLimitEntry {
        std::atomic<int> count{0};
        std::atomic<int64_t> window_start{0};
    };
    
    std::mutex mutex_;
    std::map<std::string, RateLimitEntry> entries_;
    
    int64_t get_current_time();
    std::string make_key(const std::string& identifier,
                        const std::string& endpoint);
};

// ============================================================================
// IP Rate Limiter (per IP)
// ============================================================================

class IPRateLimiter {
public:
    static IPRateLimiter& instance();
    
    void initialize(int max_requests = 100, int window_seconds = 60);
    
    bool is_allowed(const std::string& ip_address);
    RateLimiterService::RateLimitResult check(const std::string& ip_address);
    
    // Blacklist
    bool is_blacklisted(const std::string& ip_address);
    void blacklist_ip(const std::string& ip_address);
    void unblacklist_ip(const std::string& ip_address);
    std::vector<std::string> get_blacklist();
    
private:
    IPRateLimiter() = default;
    
    int max_requests_ = 100;
    int window_seconds_ = 60;
    
    std::mutex mutex_;
    std::map<std::string, RateLimiterService::RateLimitResult> ip_stats_;
    std::vector<std::string> blacklist_;
};

// ============================================================================
// Admin Action Rate Limiter
// ============================================================================

class AdminActionRateLimiter {
public:
    static AdminActionRateLimiter& instance();
    
    void initialize();
    
    // Limit sensitive admin actions
    bool allow_action(AdminID admin_id, const std::string& action);
    
    // Get remaining actions
    int get_remaining(AdminID admin_id, const std::string& action);
    
    // Config per action
    void set_limits(const std::string& action, int max_per_hour,
                   int max_per_day);
    
private:
    AdminActionRateLimiter() = default;
    
    struct ActionLimit {
        int max_per_hour;
        int max_per_day;
        int used_this_hour;
        int used_this_day;
        int64_t hour_reset;
        int64_t day_reset;
    };
    
    std::mutex mutex_;
    std::map<std::string, ActionLimit> limits_;
    std::map<std::pair<AdminID, std::string>, ActionLimit> admin_usage_;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_RATE_LIMITER_HPP
