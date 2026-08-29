/**
 * TigerAdmin C++ Core - Rate Limiter Implementation
 */

#include "admin_rate_limiter.hpp"
#include "admin_logger.hpp"
#include <chrono>

namespace tiger {
namespace admin {

RateLimiterService& RateLimiterService::instance() {
    static RateLimiterService service;
    return service;
}

void RateLimiterService::initialize() { LOG_INFO("Rate limiter service initialized"); }

RateLimiterService::RateLimitResult RateLimiterService::check_rate_limit(
    const std::string& identifier, const std::string& endpoint,
    int max_requests, int window_seconds) {
    
    RateLimitResult result;
    result.allowed = is_allowed(identifier, endpoint, max_requests, window_seconds);
    result.remaining = get_current_usage(identifier, endpoint, window_seconds);
    result.reset_time = get_current_time() + window_seconds;
    result.retry_after = std::to_string(window_seconds);
    return result;
}

bool RateLimiterService::is_allowed(const std::string& identifier,
    const std::string& endpoint, int max_requests, int window_seconds) {
    return get_current_usage(identifier, endpoint, window_seconds) < max_requests;
}

int RateLimiterService::get_current_usage(const std::string& identifier,
    const std::string& endpoint, int window_seconds) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto key = make_key(identifier, endpoint);
    auto it = entries_.find(key);
    if (it == entries_.end()) return 0;
    return it->second.count.load();
}

void RateLimiterService::reset(const std::string& identifier) {
    std::lock_guard<std::mutex> lock(mutex_);
    entries_.erase(identifier);
}

void RateLimiterService::reset_endpoint(const std::string& identifier, const std::string& endpoint) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto key = make_key(identifier, endpoint);
    entries_.erase(key);
}

void RateLimiterService::cleanup(int max_age_seconds) {
    std::lock_guard<std::mutex> lock(mutex_);
    int64_t now = get_current_time();
    for (auto it = entries_.begin(); it != entries_.end();) {
        if (now - it->second.window_start.load() > max_age_seconds) {
            it = entries_.erase(it);
        } else {
            ++it;
        }
    }
}

int64_t RateLimiterService::get_current_time() {
    return std::chrono::duration_cast<std::chrono::seconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
}

std::string RateLimiterService::make_key(const std::string& identifier, const std::string& endpoint) {
    return identifier + ":" + endpoint;
}

// IP Rate Limiter
IPRateLimiter& IPRateLimiter::instance() {
    static IPRateLimiter limiter;
    return limiter;
}

void IPRateLimiter::initialize(int max_requests, int window_seconds) {
    max_requests_ = max_requests;
    window_seconds_ = window_seconds;
    LOG_INFO("IP rate limiter initialized: " + std::to_string(max_requests) + " req/" + std::to_string(window_seconds) + "s");
}

void IPRateLimiter::initialize() {
    initialize(max_requests_, window_seconds_);
}

bool IPRateLimiter::is_allowed(const std::string& ip_address) {
    return check(ip_address).allowed;
}

RateLimiterService::RateLimitResult IPRateLimiter::check(const std::string& ip_address) {
    return RateLimiterService::instance().check_rate_limit(ip_address, "global", max_requests_, window_seconds_);
}

bool IPRateLimiter::is_blacklisted(const std::string& ip_address) {
    std::lock_guard<std::mutex> lock(mutex_);
    return std::find(blacklist_.begin(), blacklist_.end(), ip_address) != blacklist_.end();
}

void IPRateLimiter::blacklist_ip(const std::string& ip_address) {
    std::lock_guard<std::mutex> lock(mutex_);
    if (!is_blacklisted(ip_address)) {
        blacklist_.push_back(ip_address);
    }
}

void IPRateLimiter::unblacklist_ip(const std::string& ip_address) {
    std::lock_guard<std::mutex> lock(mutex_);
    blacklist_.erase(std::remove(blacklist_.begin(), blacklist_.end(), ip_address), blacklist_.end());
}

std::vector<std::string> IPRateLimiter::get_blacklist() {
    std::lock_guard<std::mutex> lock(mutex_);
    return blacklist_;
}

// Admin Action Rate Limiter
AdminActionRateLimiter& AdminActionRateLimiter::instance() {
    static AdminActionRateLimiter limiter;
    return limiter;
}

void AdminActionRateLimiter::initialize() { LOG_INFO("Admin action rate limiter initialized"); }

bool AdminActionRateLimiter::allow_action(AdminID admin_id, const std::string& action) {
    return true;
}

int AdminActionRateLimiter::get_remaining(AdminID admin_id, const std::string& action) {
    return 100;
}

void AdminActionRateLimiter::set_limits(const std::string& action, int max_per_hour, int max_per_day) {}

} // namespace admin
} // namespace tiger
