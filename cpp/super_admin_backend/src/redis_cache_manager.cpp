/**
 * TigerWallet Super Admin - Redis Cache Manager Implementation
 * High-performance in-memory caching with Redis backend
 */

#include "redis_cache_manager.hpp"
#include <iostream>
#include <sstream>
#include <thread>
#include <chrono>
#include <cstring>

namespace tigerwallet {
namespace super_admin {

// ============================================================================
// LRUCache Implementation
// ============================================================================

LRUCache::LRUCache(size_t max_size, size_t max_memory_bytes)
    : max_size_(max_size), max_memory_bytes_(max_memory_bytes), current_memory_(0) {
    head_ = new Node{"", "", {}, {}, 0, nullptr, nullptr};
    tail_ = new Node{"", "", {}, {}, 0, nullptr, nullptr};
    head_->next = tail_;
    tail_->prev = head_;
}

LRUCache::~LRUCache() {
    clear();
    delete head_;
    delete tail_;
}

void LRUCache::detach(Node* node) {
    node->prev->next = node->next;
    node->next->prev = node->prev;
    current_memory_ -= node->size();
}

void LRUCache::attach_to_head(Node* node) {
    node->next = head_->next;
    node->prev = head_;
    head_->next->prev = node;
    head_->next = node;
    current_memory_ += node->size();
}

void LRUCache::remove_lru() {
    if (tail_->prev == head_) return;
    
    Node* lru = tail_->prev;
    detach(lru);
    cache_.erase(lru->key);
    delete lru;
}

bool LRUCache::put(const std::string& key, const std::string& value, int ttl) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = cache_.find(key);
    if (it != cache_.end()) {
        detach(it->second);
        delete it->second;
        cache_.erase(it);
    }
    
    while (cache_.size() >= max_size_ || (current_memory_ + key.size() + value.size() > max_memory_bytes_)) {
        if (cache_.empty()) break;
        remove_lru();
    }
    
    Node* node = new Node{key, value, std::chrono::steady_clock::now(), 
                          std::chrono::steady_clock::now(), ttl, nullptr, nullptr};
    cache_[key] = node;
    attach_to_head(node);
    
    return true;
}

std::optional<std::string> LRUCache::get(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = cache_.find(key);
    if (it == cache_.end()) {
        return std::nullopt;
    }
    
    Node* node = it->second;
    
    if (node->is_expired()) {
        detach(node);
        delete node;
        cache_.erase(it);
        return std::nullopt;
    }
    
    detach(node);
    attach_to_head(node);
    node->last_accessed = std::chrono::steady_clock::now();
    
    return node->value;
}

bool LRUCache::remove(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = cache_.find(key);
    if (it == cache_.end()) {
        return false;
    }
    
    detach(it->second);
    delete it->second;
    cache_.erase(it);
    
    return true;
}

bool LRUCache::exists(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = cache_.find(key);
    if (it == cache_.end()) {
        return false;
    }
    
    if (it->second->is_expired()) {
        detach(it->second);
        delete it->second;
        cache_.erase(it);
        return false;
    }
    
    return true;
}

void LRUCache::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    for (auto& [key, node] : cache_) {
        delete node;
    }
    cache_.clear();
    current_memory_ = 0;
    
    head_->next = tail_;
    tail_->prev = head_;
}

void LRUCache::cleanup_expired() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::vector<std::string> expired_keys;
    for (auto& [key, node] : cache_) {
        if (node->is_expired()) {
            expired_keys.push_back(key);
        }
    }
    
    for (const auto& key : expired_keys) {
        auto it = cache_.find(key);
        if (it != cache_.end()) {
            detach(it->second);
            delete it->second;
            cache_.erase(it);
        }
    }
}

std::vector<std::string> LRUCache::get_keys(const std::string& pattern) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::vector<std::string> keys;
    for (auto& [key, node] : cache_) {
        if (pattern == "*" || key.find(pattern.substr(0, pattern.length() - 1)) == 0) {
            keys.push_back(key);
        }
    }
    return keys;
}

int LRUCache::get_ttl(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = cache_.find(key);
    if (it == cache_.end()) {
        return -2;
    }
    
    return it->second->ttl;
}

bool LRUCache::set_ttl(const std::string& key, int ttl) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = cache_.find(key);
    if (it == cache_.end()) {
        return false;
    }
    
    it->second->ttl = ttl;
    return true;
}

// ============================================================================
// RedisConnection Implementation
// ============================================================================

RedisConnection::RedisConnection(redisContext* ctx, uint64_t id)
    : ctx_(ctx), id_(id) {}

RedisConnection::~RedisConnection() {
    if (ctx_) {
        redisFree(ctx_);
        ctx_ = nullptr;
    }
}

RedisConnection::RedisConnection(RedisConnection&& other) noexcept
    : ctx_(other.ctx_), id_(other.id_) {
    other.ctx_ = nullptr;
}

RedisConnection& RedisConnection::operator=(RedisConnection&& other) noexcept {
    if (this != &other) {
        if (ctx_) {
            redisFree(ctx_);
        }
        ctx_ = other.ctx_;
        id_ = other.id_;
        other.ctx_ = nullptr;
    }
    return *this;
}

bool RedisConnection::is_connected() const {
    return ctx_ && ctx_->err == REDIS_OK;
}

redisReply* RedisConnection::command(const char* format, ...) {
    if (!ctx_) return nullptr;
    
    va_list args;
    va_start(args, format);
    redisReply* reply = (redisReply*)redisvCommand(ctx_, format, args);
    va_end(args);
    
    return reply;
}

redisReply* RedisConnection::command_argv(int argc, const char** argv, const size_t* argvlen) {
    if (!ctx_) return nullptr;
    return (redisReply*)redisCommandArgv(ctx_, argc, argv, argvlen);
}

// ============================================================================
// RedisConnectionPool Implementation
// ============================================================================

RedisConnectionPool::RedisConnectionPool(const RedisConfig& config)
    : config_(config), initialized_(false), next_id_(1) {}

RedisConnectionPool::~RedisConnectionPool() {
    shutdown();
}

bool RedisConnectionPool::initialize() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (initialized_) {
        return true;
    }
    
    for (int i = 0; i < config_.pool_size; ++i) {
        auto conn = create_connection();
        if (conn) {
            auto wrapped = std::make_shared<RedisConnection>(conn, next_id_++);
            connections_.push_back(wrapped);
            available_ids_.push_back(wrapped->id());
        }
    }
    
    if (connections_.empty()) {
        std::cerr << "[Redis Pool] Failed to create initial connections" << std::endl;
        return false;
    }
    
    initialized_ = true;
    std::cout << "[Redis Pool] Initialized with " << connections_.size() << " connections" << std::endl;
    return true;
}

std::shared_ptr<RedisConnection> RedisConnectionPool::acquire() {
    std::unique_lock<std::mutex> lock(mutex_);
    
    if (!initialized_) {
        return nullptr;
    }
    
    for (auto& conn : connections_) {
        if (conn->is_connected()) {
            return conn;
        }
    }
    
    // Try to create a new connection
    auto conn = create_connection();
    if (conn) {
        auto wrapped = std::make_shared<RedisConnection>(conn, next_id_++);
        connections_.push_back(wrapped);
        return wrapped;
    }
    
    // Wait for connection
    cv_.wait_for(lock, std::chrono::seconds(5), [this] { 
        return !available_ids_.empty(); 
    });
    
    for (auto& conn : connections_) {
        if (conn->is_connected()) {
            return conn;
        }
    }
    
    return nullptr;
}

void RedisConnectionPool::release(std::shared_ptr<RedisConnection> conn) {
    if (!conn) return;
    
    std::lock_guard<std::mutex> lock(mutex_);
    available_ids_.push_back(conn->id());
    cv_.notify_one();
}

size_t RedisConnectionPool::get_active_connections() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return connections_.size();
}

size_t RedisConnectionPool::get_idle_connections() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return available_ids_.size();
}

bool RedisConnectionPool::health_check() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    for (auto& conn : connections_) {
        if (conn->is_connected()) {
            auto reply = conn->command("PING");
            if (reply) {
                free(reply);
                return true;
            }
        }
    }
    return false;
}

void RedisConnectionPool::shutdown() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    for (auto& conn : connections_) {
        // Connection destructor handles cleanup
    }
    
    connections_.clear();
    available_ids_.clear();
    initialized_ = false;
}

redisContext* RedisConnectionPool::create_connection() {
    redisContext* ctx = redisConnect(config_.host.c_str(), config_.port);
    
    if (!ctx || ctx->err) {
        if (ctx) {
            std::cerr << "[Redis Pool] Connection error: " << ctx->errstr << std::endl;
            redisFree(ctx);
        }
        return nullptr;
    }
    
    if (!config_.password.empty()) {
        redisReply* reply = (redisReply*)redisCommand(ctx, "AUTH %s", config_.password.c_str());
        if (!reply || reply->type == REDIS_REPLY_ERROR) {
            std::cerr << "[Redis Pool] Auth error" << std::endl;
            if (reply) free(reply);
            redisFree(ctx);
            return nullptr;
        }
        free(reply);
    }
    
    if (config_.database != 0) {
        redisReply* reply = (redisReply*)redisCommand(ctx, "SELECT %d", config_.database);
        if (reply) free(reply);
    }
    
    return ctx;
}

bool RedisConnectionPool::validate_connection(redisContext* ctx) {
    if (!ctx || ctx->err) return false;
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "PING");
    if (!reply) return false;
    
    bool valid = (reply->type == REDIS_REPLY_STATUS && 
                  strcmp(reply->str, "PONG") == 0);
    free(reply);
    
    return valid;
}

// ============================================================================
// RedisCacheManager Implementation
// ============================================================================

RedisCacheManager& RedisCacheManager::getInstance() {
    static RedisCacheManager instance;
    return instance;
}

RedisCacheManager::RedisCacheManager() : initialized_(false) {}

RedisCacheManager::~RedisCacheManager() {
    shutdown();
}

void RedisCacheManager::configure(const RedisConfig& redis_config) {
    redis_config_ = redis_config;
}

void RedisCacheManager::configure_cache(const CacheConfig& cache_config) {
    cache_config_ = cache_config;
}

bool RedisCacheManager::initialize() {
    if (initialized_) {
        return true;
    }
    
    pool_ = std::make_unique<RedisConnectionPool>(redis_config_);
    
    if (!pool_->initialize()) {
        std::cerr << "[Redis] Failed to initialize connection pool" << std::endl;
        return false;
    }
    
    size_t max_memory = cache_config_.max_memory_mb * 1024 * 1024;
    lru_cache_ = std::make_unique<LRUCache>(cache_config_.max_entries, max_memory);
    
    initialized_ = true;
    std::cout << "[Redis] Cache manager initialized successfully" << std::endl;
    return true;
}

void RedisCacheManager::shutdown() {
    if (pool_) {
        pool_->shutdown();
        pool_.reset();
    }
    
    if (lru_cache_) {
        lru_cache_->clear();
        lru_cache_.reset();
    }
    
    initialized_ = false;
}

std::string RedisCacheManager::build_key(const std::string& key) const {
    return cache_config_.persistence_prefix + key;
}

std::string RedisCacheManager::extract_key(const std::string& full_key) const {
    size_t prefix_len = cache_config_.persistence_prefix.length();
    if (full_key.length() > prefix_len) {
        return full_key.substr(prefix_len);
    }
    return full_key;
}

void RedisCacheManager::update_stats(bool hit) {
    if (hit) {
        hit_count_++;
    } else {
        miss_count_++;
    }
}

bool RedisCacheManager::set(const std::string& key, const std::string& value, int ttl) {
    if (!initialized_) return false;
    
    // Always set in LRU cache first
    int effective_ttl = ttl > 0 ? ttl : cache_config_.ttl_default;
    lru_cache_->put(key, value, effective_ttl);
    
    // Optionally persist to Redis
    if (cache_config_.enable_persistence && pool_) {
        auto conn = pool_->acquire();
        if (conn && conn->is_connected()) {
            std::string full_key = build_key(key);
            if (ttl > 0) {
                conn->command("SETEX %s %d %s", full_key.c_str(), ttl, value.c_str());
            } else {
                conn->command("SET %s %s", full_key.c_str(), value.c_str());
            }
            pool_->release(conn);
        }
    }
    
    return true;
}

bool RedisCacheManager::set(const std::string& key, const std::vector<std::string>& values, int ttl) {
    std::string joined;
    for (size_t i = 0; i < values.size(); ++i) {
        if (i > 0) joined += "\n";
        joined += values[i];
    }
    return set(key, joined, ttl);
}

std::optional<std::string> RedisCacheManager::get(const std::string& key) {
    if (!initialized_) return std::nullopt;
    
    // Try LRU cache first
    auto lru_value = lru_cache_->get(key);
    if (lru_value) {
        update_stats(true);
        return lru_value;
    }
    
    update_stats(false);
    
    // Try Redis
    if (cache_config_.enable_persistence && pool_) {
        auto conn = pool_->acquire();
        if (conn && conn->is_connected()) {
            std::string full_key = build_key(key);
            redisReply* reply = conn->command("GET %s", full_key.c_str());
            
            if (reply && reply->type == REDIS_REPLY_STRING) {
                std::string value(reply->str, reply->len);
                
                // Populate LRU cache
                int ttl = this->ttl(key);
                if (ttl > 0) {
                    lru_cache_->put(key, value, ttl);
                }
                
                free(reply);
                pool_->release(conn);
                return value;
            }
            
            if (reply) free(reply);
            pool_->release(conn);
        }
    }
    
    return std::nullopt;
}

std::vector<std::optional<std::string>> RedisCacheManager::get(const std::vector<std::string>& keys) {
    std::vector<std::optional<std::string>> results;
    results.reserve(keys.size());
    
    for (const auto& key : keys) {
        results.push_back(get(key));
    }
    
    return results;
}

bool RedisCacheManager::exists(const std::string& key) {
    if (!initialized_) return false;
    
    if (lru_cache_->exists(key)) {
        return true;
    }
    
    if (cache_config_.enable_persistence && pool_) {
        auto conn = pool_->acquire();
        if (conn && conn->is_connected()) {
            std::string full_key = build_key(key);
            redisReply* reply = conn->command("EXISTS %s", full_key.c_str());
            
            bool exists = (reply && reply->type == REDIS_REPLY_INTEGER && reply->integer > 0);
            
            if (reply) free(reply);
            pool_->release(conn);
            return exists;
        }
    }
    
    return false;
}

bool RedisCacheManager::remove(const std::string::size_type key) {
    if (!initialized_) return false;
    
    lru_cache_->remove(key);
    
    if (cache_config_.enable_persistence && pool_) {
        auto conn = pool_->acquire();
        if (conn && conn->is_connected()) {
            std::string full_key = build_key(key);
            conn->command("DEL %s", full_key.c_str());
            pool_->release(conn);
        }
    }
    
    return true;
}

int RedisCacheManager::remove(const std::vector<std::string>& keys) {
    int count = 0;
    for (const auto& key : keys) {
        if (remove(key)) count++;
    }
    return count;
}

void RedisCacheManager::clear() {
    lru_cache_->clear();
    
    if (pool_) {
        auto conn = pool_->acquire();
        if (conn && conn->is_connected()) {
            std::string pattern = build_key("*");
            redisReply* reply = conn->command("KEYS %s", pattern.c_str());
            
            if (reply && reply->type == REDIS_REPLY_ARRAY) {
                for (size_t i = 0; i < reply->element.size(); ++i) {
                    conn->command("DEL %s", reply->element[i]->str);
                }
            }
            
            if (reply) free(reply);
            pool_->release(conn);
        }
    }
}

bool RedisCacheManager::expire(const std::string& key, int ttl) {
    lru_cache_->set_ttl(key, ttl);
    
    if (pool_) {
        auto conn = pool_->acquire();
        if (conn && conn->is_connected()) {
            std::string full_key = build_key(key);
            redisReply* reply = conn->command("EXPIRE %s %d", full_key.c_str(), ttl);
            bool result = (reply && reply->type == REDIS_REPLY_INTEGER && reply->integer > 0);
            if (reply) free(reply);
            pool_->release(conn);
            return result;
        }
    }
    
    return false;
}

int RedisCacheManager::ttl(const std::string& key) {
    int lru_ttl = lru_cache_->get_ttl(key);
    if (lru_ttl > 0) {
        return lru_ttl;
    }
    
    if (pool_) {
        auto conn = pool_->acquire();
        if (conn && conn->is_connected()) {
            std::string full_key = build_key(key);
            redisReply* reply = conn->command("TTL %s", full_key.c_str());
            
            int ttl = -1;
            if (reply && reply->type == REDIS_REPLY_INTEGER) {
                ttl = static_cast<int>(reply->integer);
            }
            
            if (reply) free(reply);
            pool_->release(conn);
            return ttl;
        }
    }
    
    return -2;
}

bool RedisCacheManager::persist(const std::string& key) {
    lru_cache_->set_ttl(key, -1);
    
    if (pool_) {
        auto conn = pool_->acquire();
        if (conn && conn->is_connected()) {
            std::string full_key = build_key(key);
            redisReply* reply = conn->command("PERSIST %s", full_key.c_str());
            bool result = (reply && reply->type == REDIS_REPLY_INTEGER && reply->integer > 0);
            if (reply) free(reply);
            pool_->release(conn);
            return result;
        }
    }
    
    return false;
}

std::vector<std::string> RedisCacheManager::keys(const std::string& pattern) {
    return lru_cache_->get_keys(pattern);
}

std::vector<std::string> RedisCacheManager::scan(const std::string& pattern, size_t count) {
    std::vector<std::string> results;
    
    if (pool_) {
        auto conn = pool_->acquire();
        if (conn && conn->is_connected()) {
            std::string full_pattern = build_key(pattern);
            redisReply* reply = conn->command("SCAN 0 MATCH %s COUNT %zu", full_pattern.c_str(), count);
            
            if (reply && reply->type == REDIS_REPLY_ARRAY && reply->element.size() >= 2) {
                redisReply* items = reply->element[1];
                for (size_t i = 0; i < items->element.size(); ++i) {
                    results.push_back(extract_key(std::string(items->element[i]->str, items->element[i]->len)));
                }
            }
            
            if (reply) free(reply);
            pool_->release(conn);
        }
    }
    
    return results;
}

// Hash operations
bool RedisCacheManager::hset(const std::string& key, const std::string& field, const std::string& value) {
    if (!pool_) return false;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return false;
    }
    
    std::string full_key = build_key(key);
    redisReply* reply = conn->command("HSET %s %s %s", full_key.c_str(), field.c_str(), value.c_str());
    
    bool result = (reply && reply->type == REDIS_REPLY_INTEGER);
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

bool RedisCacheManager::hset(const std::string& key, const std::map<std::string, std::string>& values) {
    if (!pool_) return false;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return false;
    }
    
    std::string full_key = build_key(key);
    for (const auto& [field, value] : values) {
        conn->command("HSET %s %s %s", full_key.c_str(), field.c_str(), value.c_str());
    }
    
    pool_->release(conn);
    return true;
}

std::optional<std::string> RedisCacheManager::hget(const std::string& key, const std::string& field) {
    if (!pool_) return std::nullopt;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return std::nullopt;
    }
    
    std::string full_key = build_key(key);
    redisReply* reply = conn->command("HGET %s %s", full_key.c_str(), field.c_str());
    
    std::optional<std::string> result;
    if (reply && reply->type == REDIS_REPLY_STRING) {
        result = std::string(reply->str, reply->len);
    }
    
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

std::map<std::string, std::string> RedisCacheManager::hgetall(const std::string& key) {
    std::map<std::string, std::string> result;
    
    if (!pool_) return result;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return result;
    }
    
    std::string full_key = build_key(key);
    redisReply* reply = conn->command("HGETALL %s", full_key.c_str());
    
    if (reply && reply->type == REDIS_REPLY_ARRAY) {
        for (size_t i = 0; i + 1 < reply->element.size(); i += 2) {
            std::string field(reply->element[i]->str, reply->element[i]->len);
            std::string value(reply->element[i + 1]->str, reply->element[i + 1]->len);
            result[field] = value;
        }
    }
    
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

std::vector<std::optional<std::string>> RedisCacheManager::hmget(const std::string& key, const std::vector<std::string>& fields) {
    std::vector<std::optional<std::string>> result;
    
    if (!pool_) return result;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return result;
    }
    
    std::string full_key = build_key(key);
    std::string cmd = "HMGET " + full_key;
    for (const auto& field : fields) {
        cmd += " " + field;
    }
    
    redisReply* reply = conn->command(cmd.c_str());
    
    if (reply && reply->type == REDIS_REPLY_ARRAY) {
        for (size_t i = 0; i < reply->element.size(); ++i) {
            if (reply->element[i]->type == REDIS_REPLY_STRING) {
                result.push_back(std::string(reply->element[i]->str, reply->element[i]->len));
            } else {
                result.push_back(std::nullopt);
            }
        }
    }
    
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

bool RedisCacheManager::hexists(const std::string& key, const std::string& field) {
    if (!pool_) return false;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return false;
    }
    
    std::string full_key = build_key(key);
    redisReply* reply = conn->command("HEXISTS %s %s", full_key.c_str(), field.c_str());
    
    bool result = (reply && reply->type == REDIS_REPLY_INTEGER && reply->integer > 0);
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

int RedisCacheManager::hdel(const std::string& key, const std::vector<std::string>& fields) {
    if (!pool_) return 0;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return 0;
    }
    
    std::string full_key = build_key(key);
    std::string cmd = "HDEL " + full_key;
    for (const auto& field : fields) {
        cmd += " " + field;
    }
    
    redisReply* reply = conn->command(cmd.c_str());
    
    int result = 0;
    if (reply && reply->type == REDIS_REPLY_INTEGER) {
        result = static_cast<int>(reply->integer);
    }
    
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

int RedisCacheManager::hlen(const std::string& key) {
    if (!pool_) return 0;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return 0;
    }
    
    std::string full_key = build_key(key);
    redisReply* reply = conn->command("HLEN %s", full_key.c_str());
    
    int result = 0;
    if (reply && reply->type == REDIS_REPLY_INTEGER) {
        result = static_cast<int>(reply->integer);
    }
    
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

// Counter operations
int64_t RedisCacheManager::incr(const std::string& key) {
    if (!pool_) return 0;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return 0;
    }
    
    std::string full_key = build_key(key);
    redisReply* reply = conn->command("INCR %s", full_key.c_str());
    
    int64_t result = 0;
    if (reply && reply->type == REDIS_REPLY_INTEGER) {
        result = reply->integer;
    }
    
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

int64_t RedisCacheManager::incrby(const std::string& key, int64_t increment) {
    if (!pool_) return 0;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return 0;
    }
    
    std::string full_key = build_key(key);
    redisReply* reply = conn->command("INCRBY %s %lld", full_key.c_str(), (long long)increment);
    
    int64_t result = 0;
    if (reply && reply->type == REDIS_REPLY_INTEGER) {
        result = reply->integer;
    }
    
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

int64_t RedisCacheManager::decr(const std::string& key) {
    return incrby(key, -1);
}

int64_t RedisCacheManager::decrby(const std::string& key, int64_t decrement) {
    return incrby(key, -decrement);
}

// Pub/Sub
bool RedisCacheManager::publish(const std::string& channel, const std::string& message) {
    if (!pool_) return false;
    
    auto conn = pool_->acquire();
    if (!conn || !conn->is_connected()) {
        if (conn) pool_->release(conn);
        return false;
    }
    
    redisReply* reply = conn->command("PUBLISH %s %s", channel.c_str(), message.c_str());
    
    bool result = (reply && reply->type == REDIS_REPLY_INTEGER);
    if (reply) free(reply);
    pool_->release(conn);
    
    return result;
}

// Pipeline
RedisCacheManager::Pipeline::Pipeline(RedisCacheManager& manager) : manager_(manager) {}

RedisCacheManager::Pipeline& RedisCacheManager::Pipeline::set(const std::string& key, const std::string& value, int ttl) {
    commands_.push_back("SET " + key + " " + value);
    return *this;
}

RedisCacheManager::Pipeline& RedisCacheManager::Pipeline::get(const std::string& key) {
    commands_.push_back("GET " + key);
    return *this;
}

RedisCacheManager::Pipeline& RedisCacheManager::Pipeline::del(const std::string& key) {
    commands_.push_back("DEL " + key);
    return *this;
}

RedisCacheManager::Pipeline& RedisCacheManager::Pipeline::hset(const std::string& key, const std::string& field, const std::string& value) {
    commands_.push_back("HSET " + key + " " + field + " " + value);
    return *this;
}

RedisCacheManager::Pipeline& RedisCacheManager::Pipeline::hget(const std::string& key, const std::string& field) {
    commands_.push_back("HGET " + key + " " + field);
    return *this;
}

std::vector<std::optional<std::string>> RedisCacheManager::Pipeline::execute() {
    std::vector<std::optional<std::string>> results;
    // Pipeline execution would go here
    return results;
}

RedisCacheManager::Pipeline RedisCacheManager::pipeline() {
    return Pipeline(*this);
}

// Stats
RedisCacheManager::Stats RedisCacheManager::get_stats() const {
    Stats stats;
    
    if (lru_cache_) {
        stats.memory_usage = lru_cache_->memory_usage();
        stats.entry_count = lru_cache_->size();
    }
    
    stats.hit_count = hit_count_.load();
    stats.miss_count = miss_count_.load();
    stats.hit_rate = (stats.hit_count + stats.miss_count) > 0 
        ? static_cast<double>(stats.hit_count) / (stats.hit_count + stats.miss_count) 
        : 0.0;
    
    if (pool_) {
        stats.redis_connected = pool_->get_active_connections();
    }
    
    return stats;
}

void RedisCacheManager::reset_stats() {
    hit_count_ = 0;
    miss_count_ = 0;
}

}  // namespace super_admin
}  // namespace tigerwallet
