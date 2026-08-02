#include "rate_limiter.hpp"
#include <cmath>
#include <algorithm>
#include <numeric>
#include <thread>
#include <sstream>

namespace tigerwallet {
namespace highspeed {

// ============================================================================
// Token Bucket Rate Limiter Implementation
// ============================================================================

TokenBucketLimiter::TokenBucketLimiter(const BucketConfig& config)
    : config_(config) {}

TokenBucketLimiter::Bucket& TokenBucketLimiter::get_or_create_bucket(const std::string& key) {
    auto it = buckets_.find(key);
    if (it != buckets_.end()) {
        return *it->second;
    }
    
    auto bucket = std::make_unique<Bucket>();
    bucket->tokens.store(config_.max_tokens);
    bucket->last_refill.store(current_timestamp_ms());
    bucket->requests.store(0);
    bucket->allowed.store(0);
    bucket->rejected.store(0);
    
    auto [new_it, inserted] = buckets_.emplace(key, std::move(bucket));
    return *new_it->second;
}

bool TokenBucketLimiter::try_consume(const std::string& key, size_t tokens) {
    Bucket& bucket = get_or_create_bucket(key);
    
    // Refill if needed
    refill_bucket(bucket);
    
    // Try to consume tokens
    size_t current = bucket.tokens.load(std::memory_order_relaxed);
    while (current >= tokens) {
        if (bucket.tokens.compare_exchange_weak(current, current - tokens,
            std::memory_order_release, std::memory_order_relaxed)) {
            bucket.requests.fetch_add(1, std::memory_order_relaxed);
            bucket.allowed.fetch_add(1, std::memory_order_relaxed);
            return true;
        }
    }
    
    bucket.requests.fetch_add(1, std::memory_order_relaxed);
    bucket.rejected.fetch_add(1, std::memory_order_relaxed);
    return false;
}

bool TokenBucketLimiter::consume(const std::string& key, size_t tokens, 
                                std::chrono::milliseconds timeout) {
    auto start = std::chrono::steady_clock::now();
    
    while (std::chrono::steady_clock::now() - start < timeout) {
        if (try_consume(key, tokens)) {
            return true;
        }
        std::this_thread::yield();
    }
    
    return false;
}

size_t TokenBucketLimiter::available(const std::string& key) const {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = buckets_.find(key);
    if (it == buckets_.end()) {
        return config_.max_tokens;
    }
    return it->second->tokens.load(std::memory_order_relaxed);
}

void TokenBucketLimiter::reset(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    buckets_.erase(key);
}

TokenBucketLimiter::UsageStats TokenBucketLimiter::get_stats() const {
    UsageStats stats{};
    stats.total_requests = 0;
    stats.allowed_requests = 0;
    stats.rejected_requests = 0;
    stats.current_rate = 0.0;
    
    std::lock_guard<std::mutex> lock(mutex_);
    for (const auto& [key, bucket] : buckets_) {
        stats.total_requests += bucket->requests.load(std::memory_order_relaxed);
        stats.allowed_requests += bucket->allowed.load(std::memory_order_relaxed);
        stats.rejected_requests += bucket->rejected.load(std::memory_order_relaxed);
    }
    
    return stats;
}

TokenBucketLimiter::UsageStats TokenBucketLimiter::get_stats(const std::string& key) const {
    std::lock_guard<std::mutex> lock(mutex_);
    UsageStats stats{};
    
    auto it = buckets_.find(key);
    if (it != buckets_.end()) {
        stats.total_requests = it->second->requests.load(std::memory_order_relaxed);
        stats.allowed_requests = it->second->allowed.load(std::memory_order_relaxed);
        stats.rejected_requests = it->second->rejected.load(std::memory_order_relaxed);
    }
    
    return stats;
}

void TokenBucketLimiter::refill_bucket(Bucket& bucket) {
    int64_t now = current_timestamp_ms();
    int64_t last = bucket.last_refill.load(std::memory_order_relaxed);
    
    if (now > last) {
        int64_t elapsed = now - last;
        size_t refill = (elapsed * config_.refill_rate) / 1000;
        
        if (refill > 0) {
            size_t current = bucket.tokens.load(std::memory_order_relaxed);
            size_t new_tokens = std::min(current + refill, config_.max_tokens);
            bucket.tokens.store(new_tokens, std::memory_order_release);
            bucket.last_refill.store(now, std::memory_order_release);
        }
    }
}

int64_t TokenBucketLimiter::current_timestamp_ms() const {
    return std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
}

// ============================================================================
// Sliding Window Rate Limiter Implementation
// ============================================================================

SlidingWindowRateLimiter::SlidingWindowRateLimiter(const WindowConfig& config)
    : config_(config) {}

bool SlidingWindowRateLimiter::is_allowed(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    int64_t now = current_timestamp_ms();
    auto& window = windows_[key];
    
    cleanup_old_requests(window, now);
    
    if (window.timestamps.size() < config_.max_requests) {
        window.timestamps.push_back(now);
        return true;
    }
    
    return false;
}

size_t SlidingWindowRateLimiter::get_request_count(const std::string& key) const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = windows_.find(key);
    if (it == windows_.end()) {
        return 0;
    }
    
    int64_t now = current_timestamp_ms();
    Window temp = it->second;
    cleanup_old_requests(temp, now);
    
    return temp.timestamps.size();
}

void SlidingWindowRateLimiter::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    windows_.clear();
}

void SlidingWindowRateLimiter::cleanup_old_requests(Window& window, int64_t now) {
    auto cutoff = now - config_.window_size.count();
    
    window.timestamps.erase(
        std::remove_if(window.timestamps.begin(), window.timestamps.end(),
            [cutoff](int64_t ts) { return ts < cutoff; }),
        window.timestamps.end()
    );
    
    // Cleanup old buckets if we have too many
    if (windows_.size() > config_.max_buckets) {
        // Remove buckets with no recent activity
        for (auto it = windows_.begin(); it != windows_.end();) {
            cleanup_old_requests(it->second, now);
            if (it->second.timestamps.empty()) {
                it = windows_.erase(it);
            } else {
                ++it;
            }
        }
    }
}

int64_t SlidingWindowRateLimiter::current_timestamp_ms() const {
    return std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
}

// ============================================================================
// LRU Cache Implementation
// ============================================================================

LRUCache::LRUCache(const CacheConfig& config) : config_(config) {}

void LRUCache::put(const std::string& key, const std::string& value) {
    put(key, value, config_.default_ttl);
}

void LRUCache::put(const std::string& key, const std::string& value, std::chrono::milliseconds ttl) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    int64_t expiry = current_timestamp_ms() + ttl.count();
    
    auto it = cache_.find(key);
    if (it != cache_.end()) {
        it->second->value = value;
        it->second->expiry = expiry;
        it->second->access_count.fetch_add(1, std::memory_order_relaxed);
        it->second->last_access.store(expiry, std::memory_order_relaxed);
    } else {
        evict_if_needed();
        
        auto entry = std::make_unique<CacheEntry>();
        entry->value = value;
        entry->expiry = expiry;
        entry->access_count.store(1, std::memory_order_relaxed);
        entry->last_access.store(expiry, std::memory_order_relaxed);
        
        cache_[key] = std::move(entry);
    }
}

std::optional<std::string> LRUCache::get(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = cache_.find(key);
    if (it == cache_.end()) {
        misses_.fetch_add(1, std::memory_order_relaxed);
        return std::nullopt;
    }
    
    if (is_expired(*it->second)) {
        cache_.erase(it);
        misses_.fetch_add(1, std::memory_order_relaxed);
        return std::nullopt;
    }
    
    hits_.fetch_add(1, std::memory_order_relaxed);
    it->second->access_count.fetch_add(1, std::memory_order_relaxed);
    it->second->last_access.store(current_timestamp_ms(), std::memory_order_relaxed);
    
    return it->second->value;
}

bool LRUCache::exists(const std::string& key) const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = cache_.find(key);
    if (it == cache_.end()) {
        return false;
    }
    
    return !is_expired(*it->second);
}

void LRUCache::remove(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    cache_.erase(key);
}

void LRUCache::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    cache_.clear();
    hits_.store(0, std::memory_order_relaxed);
    misses_.store(0, std::memory_order_relaxed);
    evictions_.store(0, std::memory_order_relaxed);
}

LRUCache::CacheStats LRUCache::get_stats() const {
    CacheStats stats{};
    stats.size = cache_.size();
    stats.hits = hits_.load(std::memory_order_relaxed);
    stats.misses = misses_.load(std::memory_order_relaxed);
    stats.evictions = evictions_.load(std::memory_order_relaxed);
    
    size_t total = stats.hits + stats.misses;
    stats.hit_rate = total > 0 ? static_cast<double>(stats.hits) / total : 0.0;
    
    return stats;
}

void LRUCache::evict_if_needed() {
    while (cache_.size() >= config_.max_size) {
        // Find LRU entry
        auto lru_it = cache_.begin();
        int64_t oldest_time = std::numeric_limits<int64_t>::max();
        
        for (auto it = cache_.begin(); it != cache_.end(); ++it) {
            int64_t last_access = it->second->last_access.load(std::memory_order_relaxed);
            if (last_access < oldest_time) {
                oldest_time = last_access;
                lru_it = it;
            }
        }
        
        if (lru_it != cache_.end()) {
            cache_.erase(lru_it);
            evictions_.fetch_add(1, std::memory_order_relaxed);
        }
    }
}

bool LRUCache::is_expired(const CacheEntry& entry) const {
    return current_timestamp_ms() > entry.expiry;
}

int64_t LRUCache::current_timestamp_ms() const {
    return std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
}

// ============================================================================
// Bloom Filter Implementation
// ============================================================================

BloomFilter::BloomFilter(size_t expected_items, double false_positive_rate)
    : size_(static_cast<size_t>(-expected_items * std::log(false_positive_rate) / (std::log(2) * std::log(2))))
    , hash_count_(static_cast<size_t>(size_ / expected_items * std::log(2)))
    , bits_(size_, false) {}

void BloomFilter::add(const std::string& key) {
    auto positions = get_hash_positions(key);
    
    std::lock_guard<std::mutex> lock(mutex_);
    for (size_t pos : positions) {
        bits_[pos] = true;
    }
}

bool BloomFilter::might_contain(const std::string& key) const {
    auto positions = get_hash_positions(key);
    
    for (size_t pos : positions) {
        if (!bits_[pos]) {
            return false;
        }
    }
    
    return true;
}

void BloomFilter::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    std::fill(bits_.begin(), bits_.end(), false);
}

double BloomFilter::estimated_false_positive_rate() const {
    size_t set_bits = 0;
    for (bool bit : bits_) {
        if (bit) set_bits++;
    }
    
    double p = static_cast<double>(set_bits) / size_;
    return std::pow(p, hash_count_);
}

std::vector<size_t> BloomFilter::get_hash_positions(const std::string& key) const {
    // Use FNV-1a hash split into multiple values
    std::vector<size_t> positions;
    positions.reserve(hash_count_);
    
    uint64_t hash = 1469598103934665603ULL; // FNV offset basis
    
    for (char c : key) {
        hash ^= static_cast<uint64_t>(c);
        hash *= 1099511628211ULL; // FNV prime
    }
    
    for (size_t i = 0; i < hash_count_; i++) {
        hash ^= i;
        hash *= 1099511628211ULL;
        positions.push_back(hash % size_);
    }
    
    return positions;
}

// ============================================================================
// Circular Buffer Implementation
// ============================================================================

CircularBuffer::CircularBuffer(size_t capacity)
    : capacity_(capacity)
    , buffer_(capacity, 0.0) {}

void CircularBuffer::push(double value) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    buffer_[head_] = value;
    head_ = (head_ + 1) % capacity_;
    
    if (size_ < capacity_) {
        size_++;
    }
}

double CircularBuffer::get(size_t index) const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (index >= size_) {
        return 0.0;
    }
    
    size_t actual_index = (head_ + capacity_ - size_ + index) % capacity_;
    return buffer_[actual_index];
}

double CircularBuffer::sum() const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    double total = 0.0;
    size_t count = 0;
    
    for (size_t i = 0; i < size_; i++) {
        size_t idx = (head_ + capacity_ - size_ + i) % capacity_;
        if (buffer_[idx] != 0.0) {
            total += buffer_[idx];
            count++;
        }
    }
    
    return total;
}

double CircularBuffer::avg() const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (size_ == 0) return 0.0;
    
    return sum() / size_;
}

double CircularBuffer::min() const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (size_ == 0) return 0.0;
    
    double min_val = std::numeric_limits<double>::max();
    
    for (size_t i = 0; i < size_; i++) {
        size_t idx = (head_ + capacity_ - size_ + i) % capacity_;
        min_val = std::min(min_val, buffer_[idx]);
    }
    
    return min_val;
}

double CircularBuffer::max() const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (size_ == 0) return 0.0;
    
    double max_val = std::numeric_limits<double>::lowest();
    
    for (size_t i = 0; i < size_; i++) {
        size_t idx = (head_ + capacity_ - size_ + i) % capacity_;
        max_val = std::max(max_val, buffer_[idx]);
    }
    
    return max_val;
}

double CircularBuffer::percentile(double p) const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (size_ == 0) return 0.0;
    
    std::vector<double> sorted;
    sorted.reserve(size_);
    
    for (size_t i = 0; i < size_; i++) {
        size_t idx = (head_ + capacity_ - size_ + i) % capacity_;
        sorted.push_back(buffer_[idx]);
    }
    
    std::sort(sorted.begin(), sorted.end());
    
    size_t index = static_cast<size_t>((p / 100.0) * (sorted.size() - 1));
    return sorted[index];
}

double CircularBuffer::stddev() const {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (size_ <= 1) return 0.0;
    
    double mean = avg();
    double sum_sq = 0.0;
    
    for (size_t i = 0; i < size_; i++) {
        size_t idx = (head_ + capacity_ - size_ + i) % capacity_;
        double diff = buffer_[idx] - mean;
        sum_sq += diff * diff;
    }
    
    return std::sqrt(sum_sq / (size_ - 1));
}

} // namespace highspeed
} // namespace tigerwallet
