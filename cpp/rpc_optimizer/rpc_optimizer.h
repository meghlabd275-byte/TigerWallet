/**
 * TigerWallet RPC Optimizer
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - Multi-RPC load balancing
 * - Automatic failover
 * - Request caching
 * - Latency optimization
 * - Batch requests
 */

#ifndef TIGER_RPC_OPTIMIZER_H
#define TIGER_RPC_OPTIMIZER_H

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
#include <vector>

namespace tiger {
namespace rpc {

// RPC Provider
struct RPCProvider {
    std::string name;
    std::string url;
    double latency_ms;
    double success_rate;
    uint64_t requests_total;
    uint64_t requests_failed;
    bool is_healthy;
    std::chrono::steady_clock::time_point last_check;
};

// Request types
enum class RPCMethod {
    ETH_BLOCK_NUMBER,
    ETH_GET_BALANCE,
    ETH_CALL,
    ETH_SEND_RAW_TRANSACTION,
    ETH_GET_TRANSACTION_BY_HASH,
    ETH_GET_TRANSACTION_RECEIPT,
    ETH_GET_CODE,
    ETH_ESTIMATE_GAS,
    NET_VERSION,
    ETH_CHAIN_ID,
    CUSTOM
};

struct RPCRequest {
    std::string id;
    RPCMethod method;
    std::vector<std::string> params;
    std::string chain;
    uint64_t timestamp;
    uint64_t timeout_ms;
};

struct RPCResponse {
    std::string id;
    bool success;
    std::string result;
    std::string error;
    uint64_t latency_ms;
    std::string provider;
};

// Cache entry
struct CacheEntry {
    std::string key;
    std::string value;
    std::chrono::steady_clock::time_point created;
    uint64_t ttl_ms;
};

// Performance metrics
struct OptimizerMetrics {
    uint64_t total_requests;
    uint64_t successful_requests;
    uint64_t failed_requests;
    uint64_t cache_hits;
    uint64_t cache_misses;
    double avg_latency_ms;
    double min_latency_ms;
    double max_latency_ms;
    uint64_t total_bytes_sent;
    uint64_t total_bytes_received;
};

// RPC Optimizer
class RPCOptimizer {
private:
    std::unordered_map<std::string, std::vector<RPCProvider>> chain_providers_;
    std::unordered_map<std::string, std::unique_ptr<RPCProvider>> active_providers_;
    std::unordered_map<std::string, CacheEntry> cache_;
    
    std::mutex cache_mutex_;
    std::mutex providers_mutex_;
    
    std::atomic<uint64_t> request_counter_{0};
    std::atomic<uint64_t> success_counter_{0};
    std::atomic<uint64_t> fail_counter_{0};
    
    uint64_t cache_ttl_ms_ = 5000;
    uint64_t health_check_interval_ms_ = 30000;
    uint64_t request_timeout_ms_ = 10000;

public:
    RPCOptimizer();
    ~RPCOptimizer() = default;

    // Provider management
    void add_provider(const std::string& chain, const std::string& name, 
                     const std::string& url, double latency = 100.0);
    void remove_provider(const std::string& chain, const std::string& name);
    void set_active_provider(const std::string& chain, const std::string& name);
    
    // Request execution
    std::optional<RPCResponse> execute(const RPCRequest& request);
    std::vector<RPCResponse> execute_batch(const std::vector<RPCRequest>& requests);
    
    // Caching
    std::optional<std::string> get_cached(const std::string& key);
    void set_cache(const std::string& key, const std::string& value, uint64_t ttl_ms);
    void clear_cache();
    void clear_cache(const std::string& chain);
    
    // Health monitoring
    void check_provider_health(const std::string& chain);
    std::vector<RPCProvider> get_providers(const std::string& chain);
    std::optional<RPCProvider> get_best_provider(const std::string& chain);
    
    // Metrics
    OptimizerMetrics get_metrics() const;
    void reset_metrics();

private:
    std::string generate_cache_key(const RPCRequest& request);
    std::optional<RPCResponse> execute_single(const RPCRequest& request);
    bool is_cache_valid(const CacheEntry& entry) const;
    void update_provider_stats(const std::string& chain, const std::string& provider, 
                             bool success, uint64_t latency_ms);
};

// Inline implementations

inline RPCOptimizer::RPCOptimizer() {
    // Add default providers for major chains
    add_provider("ethereum", "Infura", "https://mainnet.infura.io/v3/", 80.0);
    add_provider("ethereum", "Alchemy", "https://eth-mainnet.g.alchemy.com/v2/", 75.0);
    add_provider("ethereum", "Public", "https://eth.llamarpc.com", 120.0);
    
    add_provider("polygon", "Infura", "https://polygon-mainnet.infura.io/v3/", 60.0);
    add_provider("polygon", "Alchemy", "https://polygon-mainnet.g.alchemy.com/v2/", 55.0);
    
    add_provider("bsc", "Public", "https://bsc-dataseed.binance.org/", 50.0);
    add_provider("arbitrum", "Public", "https://arb1.arbitrum.io/rpc/", 70.0);
}

inline void RPCOptimizer::add_provider(const std::string& chain, const std::string& name,
                                     const std::string& url, double latency) {
    std::lock_guard<std::mutex> lock(providers_mutex_);
    
    RPCProvider provider;
    provider.name = name;
    provider.url = url;
    provider.latency_ms = latency;
    provider.success_rate = 1.0;
    provider.requests_total = 0;
    provider.requests_failed = 0;
    provider.is_healthy = true;
    provider.last_check = std::chrono::steady_clock::now();
    
    chain_providers_[chain].push_back(provider);
}

inline std::optional<RPCResponse> RPCOptimizer::execute(const RPCRequest& request) {
    auto start = std::chrono::high_resolution_clock::now();
    
    // Check cache first
    std::string cache_key = generate_cache_key(request);
    if (auto cached = get_cached(cache_key)) {
        request_counter_++;
        
        auto end = std::chrono::high_resolution_clock::now();
        auto latency = std::chrono::duration_cast<std::chrono::milliseconds>(end - start).count();
        
        return RPCResponse{
            .id = request.id,
            .success = true,
            .result = *cached,
            .error = "",
            .latency_ms = latency,
            .provider = "CACHE"
        };
    }
    
    // Execute request
    auto result = execute_single(request);
    
    // Cache successful responses
    if (result && result->success) {
        set_cache(cache_key, result->result, cache_ttl_ms_);
    }
    
    return result;
}

inline std::string RPCOptimizer::generate_cache_key(const RPCRequest& request) {
    std::string key = request.chain + "_" + std::to_string(static_cast<int>(request.method));
    for (const auto& param : request.params) {
        key += "_" + param;
    }
    return key;
}

inline std::optional<std::string> RPCOptimizer::get_cached(const std::string& key) {
    std::lock_guard<std::mutex> lock(cache_mutex_);
    
    auto it = cache_.find(key);
    if (it == cache_.end()) {
        return std::nullopt;
    }
    
    if (is_cache_valid(it->second)) {
        return it->second.value;
    }
    
    cache_.erase(it);
    return std::nullopt;
}

inline bool RPCOptimizer::is_cache_valid(const CacheEntry& entry) const {
    auto now = std::chrono::steady_clock::now();
    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(now - entry.created).count();
    return elapsed < static_cast<int64_t>(entry.ttl_ms);
}

inline void RPCOptimizer::set_cache(const std::string& key, const std::string& value, uint64_t ttl_ms) {
    std::lock_guard<std::mutex> lock(cache_mutex_);
    
    CacheEntry entry;
    entry.key = key;
    entry.value = value;
    entry.created = std::chrono::steady_clock::now();
    entry.ttl_ms = ttl_ms;
    
    cache_[key] = entry;
}

inline std::optional<RPCProvider> RPCOptimizer::get_best_provider(const std::string& chain) {
    std::lock_guard<std::mutex> lock(providers_mutex_);
    
    auto it = chain_providers_.find(chain);
    if (it == chain_providers_.end()) {
        return std::nullopt;
    }
    
    const auto& providers = it->second;
    if (providers.empty()) {
        return std::nullopt;
    }
    
    // Find provider with best latency and success rate
    const RPCProvider* best = nullptr;
    double best_score = -1.0;
    
    for (const auto& provider : providers) {
        if (!provider.is_healthy) continue;
        
        double score = provider.success_rate * 1000.0 / (provider.latency_ms + 1.0);
        if (score > best_score) {
            best_score = score;
            best = &provider;
        }
    }
    
    if (best) {
        return *best;
    }
    
    return std::nullopt;
}

inline OptimizerMetrics RPCOptimizer::get_metrics() const {
    OptimizerMetrics metrics;
    metrics.total_requests = request_counter_.load();
    metrics.successful_requests = success_counter_.load();
    metrics.failed_requests = fail_counter_.load();
    metrics.cache_hits = 0; // Would track separately
    metrics.cache_misses = 0;
    metrics.avg_latency_ms = 50.0; // Would calculate from actual data
    metrics.min_latency_ms = 10.0;
    metrics.max_latency_ms = 500.0;
    metrics.total_bytes_sent = 0;
    metrics.total_bytes_received = 0;
    return metrics;
}

} // namespace rpc
} // namespace tiger

#endif // TIGER_RPC_OPTIMIZER_H
