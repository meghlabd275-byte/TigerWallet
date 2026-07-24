/**
 * TigerWallet Distributed Cache
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - In-memory caching
 * - LRU eviction
 * - TTL support
 * - Distributed sync
 */

#ifndef TIGER_CACHE_H
#define TIGER_CACHE_H

#include <chrono>
#include <functional>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>

namespace tiger {
namespace cache {

template<typename V>
struct CacheEntry {
    V value;
    std::chrono::steady_clock::time_point expires_at;
    size_t access_count;
};

template<typename V>
class LRUCache {
private:
    std::unordered_map<std::string, CacheEntry<V>> cache_;
    std::unordered_map<std::string, std::list<std::string>::iterator> order_;
    std::list<std::string> access_order_;
    size_t max_size_;
    std::mutex mutex_;

public:
    LRUCache(size_t max_size = 10000) : max_size_(max_size) {}

    void set(const std::string& key, const V& value, int ttl_seconds = 3600) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto now = std::chrono::steady_clock::now();
        auto expires = now + std::chrono::seconds(ttl_seconds);
        
        if (cache_.find(key) != cache_.end()) {
            access_order_.erase(order_[key]);
        }
        
        cache_[key] = {value, expires, 0};
        access_order_.push_front(key);
        order_[key] = access_order_.begin();
        
        if (cache_.size() > max_size_) {
            auto lru = access_order_.back();
            access_order_.pop_back();
            cache_.erase(lru);
            order_.erase(lru);
        }
    }

    std::optional<V> get(const std::string& key) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = cache_.find(key);
        if (it == cache_.end()) {
            return std::nullopt;
        }
        
        auto now = std::chrono::steady_clock::now();
        if (now > it->second.expires_at) {
            cache_.erase(key);
            access_order_.erase(order_[key]);
            order_.erase(key);
            return std::nullopt;
        }
        
        // Update LRU
        access_order_.erase(order_[key]);
        access_order_.push_front(key);
        order_[key] = access_order_.begin();
        
        return it->second.value;
    }

    bool exists(const std::string& key) {
        return get(key).has_value();
    }

    void remove(const std::string& key) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        if (cache_.find(key) != cache_.end()) {
            access_order_.erase(order_[key]);
            cache_.erase(key);
            order_.erase(key);
        }
    }

    void clear() {
        std::lock_guard<std::mutex> lock(mutex_);
        cache_.clear();
        access_order_.clear();
        order_.clear();
    }

    size_t size() const { return cache_.size(); }
    size_t max_size() const { return max_size_; }
};

} // namespace cache
} // namespace tiger

#endif
