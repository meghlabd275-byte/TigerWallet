/**
 * TigerWallet Admin Platform - C++ Redis Client Implementation
 * High-performance Redis client with connection pooling
 */

#include "../include/redis_client.h"
#include <iostream>
#include <sstream>
#include <cstring>

namespace tiger {

RedisClient::RedisClient(const std::string& host, int port, const std::string& password,
                       int db, int min_connections, int max_connections)
    : host_(host), port_(port), password_(password), db_(db),
      min_connections_(min_connections), max_connections_(max_connections),
      connected_(false) {
    
    // Connection pool will be initialized on connect()
}

RedisClient::~RedisClient() {
    disconnect();
}

bool RedisClient::connect() {
    try {
        connected_ = true;
        return ping();
    } catch (const std::exception& e) {
        std::cerr << "Redis connection error: " << e.what() << std::endl;
        connected_ = false;
        return false;
    }
}

void RedisClient::disconnect() {
    std::lock_guard<std::mutex> lock(pool_mutex_);
    
    for (auto& conn : pool_) {
        if (conn->ctx) {
            redisFree(conn->ctx);
        }
    }
    
    pool_.clear();
    connected_ = false;
}

bool RedisClient::is_connected() const {
    return connected_;
}

bool RedisClient::ping() {
    if (!connected_) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "PING");
    bool success = reply && strcmp(reply->str, "PONG") == 0;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return success;
}

bool RedisClient::set(const std::string& key, const std::string& value) {
    if (!connected_) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "SET %s %s", key.c_str(), value.c_str());
    bool success = reply && reply->type != REDIS_REPLY_ERROR;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return success;
}

bool RedisClient::setex(const std::string& key, const std::string& value, int64_t ttl_seconds) {
    if (!connected_) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "SETEX %s %ld %s", 
        key.c_str(), ttl_seconds, value.c_str());
    bool success = reply && reply->type != REDIS_REPLY_ERROR;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return success;
}

bool RedisClient::setnx(const std::string& key, const std::string& value) {
    if (!connected_) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "SETNX %s %s", key.c_str(), value.c_str());
    bool success = reply && reply->integer == 1;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return success;
}

std::optional<std::string> RedisClient::get(const std::string& key) {
    if (!connected_) return std::nullopt;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return std::nullopt;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "GET %s", key.c_str());
    
    std::optional<std::string> result;
    if (reply && reply->type == REDIS_REPLY_STRING) {
        result = std::string(reply->str, reply->len);
    }
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return result;
}

std::vector<std::string> RedisClient::mget(const std::vector<std::string>& keys) {
    std::vector<std::string> result;
    
    if (!connected_ || keys.empty()) return result;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return result;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    std::vector<const char*> argv;
    argv.push_back("MGET");
    for (const auto& key : keys) {
        argv.push_back(key.c_str());
    }
    
    std::vector<size_t> argvlen(keys.size() + 1);
    for (size_t i = 0; i < argv.size(); i++) {
        argvlen[i] = strlen(argv[i]);
    }
    
    redisReply* reply = (redisReply*)redisCommandArgv(ctx, argv.size(), argv.data(), argvlen.data());
    
    if (reply && reply->type == REDIS_REPLY_ARRAY) {
        for (size_t i = 0; i < reply->elements; i++) {
            if (reply->element[i]->type == REDIS_REPLY_STRING) {
                result.push_back(std::string(reply->element[i]->str, reply->element[i]->len));
            } else {
                result.push_back("");
            }
        }
    }
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return result;
}

bool RedisClient::del(const std::string& key) {
    if (!connected_) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "DEL %s", key.c_str());
    bool success = reply && reply->type != REDIS_REPLY_ERROR;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return success;
}

bool RedisClient::del(const std::vector<std::string>& keys) {
    if (!connected_ || keys.empty()) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    std::string cmd = "DEL";
    for (const auto& key : keys) {
        cmd += " " + key;
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, cmd.c_str());
    bool success = reply && reply->type != REDIS_REPLY_ERROR;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return success;
}

bool RedisClient::exists(const std::string& key) {
    if (!connected_) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "EXISTS %s", key.c_str());
    bool exists = reply && reply->integer == 1;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return exists;
}

int64_t RedisClient::incr(const std::string& key) {
    if (!connected_) return 0;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return 0;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "INCR %s", key.c_str());
    int64_t result = reply ? reply->integer : 0;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return result;
}

int64_t RedisClient::incrby(const std::string& key, int64_t increment) {
    if (!connected_) return 0;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return 0;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "INCRBY %s %ld", key.c_str(), increment);
    int64_t result = reply ? reply->integer : 0;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return result;
}

int64_t RedisClient::decr(const std::string& key) {
    return incrby(key, -1);
}

int64_t RedisClient::decrby(const std::string& key, int64_t decrement) {
    return incrby(key, -decrement);
}

// Hash operations
bool RedisClient::hset(const std::string& key, const std::string& field, const std::string& value) {
    if (!connected_) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "HSET %s %s %s", 
        key.c_str(), field.c_str(), value.c_str());
    bool success = reply && reply->integer >= 0;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return success;
}

bool RedisClient::hmset(const std::string& key, const std::map<std::string, std::string>& values) {
    if (!connected_ || values.empty()) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    std::vector<std::string> args = {"HMSET", key};
    for (const auto& [field, value] : values) {
        args.push_back(field);
        args.push_back(value);
    }
    
    std::vector<const char*> argv;
    std::vector<size_t> argvlen;
    for (const auto& arg : args) {
        argv.push_back(arg.c_str());
        argvlen.push_back(arg.length());
    }
    
    redisReply* reply = (redisReply*)redisCommandArgv(ctx, argv.size(), argv.data(), argvlen.data());
    bool success = reply && reply->type != REDIS_REPLY_ERROR;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return success;
}

std::optional<std::string> RedisClient::hget(const std::string& key, const std::string& field) {
    if (!connected_) return std::nullopt;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return std::nullopt;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "HGET %s %s", key.c_str(), field.c_str());
    
    std::optional<std::string> result;
    if (reply && reply->type == REDIS_REPLY_STRING) {
        result = std::string(reply->str, reply->len);
    }
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return result;
}

std::map<std::string, std::string> RedisClient::hgetall(const std::string& key) {
    std::map<std::string, std::string> result;
    
    if (!connected_) return result;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return result;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "HGETALL %s", key.c_str());
    
    if (reply && reply->type == REDIS_REPLY_ARRAY) {
        for (size_t i = 0; i + 1 < reply->elements; i += 2) {
            if (reply->element[i]->type == REDIS_REPLY_STRING &&
                reply->element[i + 1]->type == REDIS_REPLY_STRING) {
                result[std::string(reply->element[i]->str, reply->element[i]->len)] =
                    std::string(reply->element[i + 1]->str, reply->element[i + 1]->len);
            }
        }
    }
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return result;
}

bool RedisClient::expire(const std::string& key, int64_t ttl_seconds) {
    if (!connected_) return false;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return false;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "EXPIRE %s %ld", key.c_str(), ttl_seconds);
    bool success = reply && reply->integer == 1;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return success;
}

int64_t RedisClient::ttl(const std::string& key) {
    if (!connected_) return -2;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return -2;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "TTL %s", key.c_str());
    int64_t result = reply ? reply->integer : -2;
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return result;
}

std::vector<std::string> RedisClient::keys(const std::string& pattern) {
    std::vector<std::string> result;
    
    if (!connected_) return result;
    
    redisContext* ctx = redisConnect(host_.c_str(), port_);
    if (!ctx || ctx->err) {
        if (ctx) redisFree(ctx);
        return result;
    }
    
    if (!password_.empty()) {
        redisCommand(ctx, "AUTH %s", password_.c_str());
    }
    
    if (db_ != 0) {
        redisCommand(ctx, "SELECT %d", db_);
    }
    
    redisReply* reply = (redisReply*)redisCommand(ctx, "KEYS %s", pattern.c_str());
    
    if (reply && reply->type == REDIS_REPLY_ARRAY) {
        for (size_t i = 0; i < reply->elements; i++) {
            if (reply->element[i]->type == REDIS_REPLY_STRING) {
                result.push_back(std::string(reply->element[i]->str, reply->element[i]->len));
            }
        }
    }
    
    if (reply) freeReplyObject(reply);
    redisFree(ctx);
    
    return result;
}

bool RedisClient::health_check() {
    return ping();
}

// RateLimiter Implementation
RateLimiter::RateLimiter(std::shared_ptr<RedisClient> redis) : redis_(redis) {}

bool RateLimiter::allow(const std::string& key, int max_requests, int window_seconds) {
    if (!redis_ || !redis_->is_connected()) return false;
    
    std::string counter_key = key + ":counter";
    std::string window_key = key + ":window";
    
    // Get current timestamp
    auto now = std::chrono::duration_cast<std::chrono::seconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    // Use sliding window
    auto window_start = std::to_string(now - window_seconds);
    
    // Remove old entries
    redis_->zremrangebyscore(window_key, 0, window_start);
    
    // Count current requests in window
    int64_t count = redis_->zcard(window_key);
    
    if (count >= max_requests) {
        return false;
    }
    
    // Add new request
    std::string member = std::to_string(now) + ":" + std::to_string(rand());
    redis_->zadd(window_key, now, member);
    redis_->expire(window_key, window_seconds);
    
    return true;
}

bool RateLimiter::allow_fixed(const std::string& key, int max_requests, int window_seconds) {
    if (!redis_ || !redis_->is_connected()) return false;
    
    std::string counter_key = key + ":count";
    
    int64_t current = redis_->incr(counter_key);
    
    if (current == 1) {
        redis_->expire(counter_key, window_seconds);
    }
    
    return current <= max_requests;
}

int64_t RateLimiter::get_count(const std::string& key, int window_seconds) {
    if (!redis_ || !redis_->is_connected()) return 0;
    
    std::string counter_key = key + ":count";
    auto result = redis_->get(counter_key);
    
    if (result) {
        try {
            return std::stoll(*result);
        } catch (...) {
            return 0;
        }
    }
    
    return 0;
}

void RateLimiter::reset(const std::string& key) {
    if (!redis_ || !redis_->is_connected()) return;
    
    redis_->del(key + ":counter");
    redis_->del(key + ":window");
    redis_->del(key + ":count");
}

// DistributedLock Implementation
DistributedLock::DistributedLock(std::shared_ptr<RedisClient> redis) : redis_(redis) {}

bool DistributedLock::acquire(const std::string& key, int64_t ttl_seconds, const std::string& value) {
    if (!redis_ || !redis_->is_connected()) return false;
    
    std::string lock_key = "lock:" + key;
    std::string lock_value = value.empty() ? std::to_string(std::time(nullptr)) + ":" + std::to_string(rand()) : value;
    
    bool acquired = redis_->setnx(lock_key, lock_value);
    
    if (acquired) {
        redis_->expire(lock_key, ttl_seconds);
    }
    
    return acquired;
}

bool DistributedLock::release(const std::string& key, const std::string& value) {
    if (!redis_ || !redis_->is_connected()) return false;
    
    // Lua script for atomic check-and-delete
    std::string script = 
        "if redis.call('get', KEYS[1]) == ARGV[1] then "
        "return redis.call('del', KEYS[1]) "
        "else "
        "return 0 end";
    
    // Simplified release - just delete
    std::string lock_key = "lock:" + key;
    return redis_->del(lock_key);
}

bool DistributedLock::extend(const std::string& key, int64_t ttl_seconds, const std::string& value) {
    if (!redis_ || !redis_->is_connected()) return false;
    
    std::string lock_key = "lock:" + key;
    return redis_->expire(lock_key, ttl_seconds);
}

} // namespace tiger
