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

#include <algorithm>
#include <atomic>
#include <cctype>
#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <functional>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_map>
#include <vector>

#include <curl/curl.h>

// The environment block, read once at construction to discover the configured
// node endpoints. POSIX declares this in unistd.h; the extern is repeated here
// so the header stands on its own.
#include <unistd.h>
#if !defined(_POSIX_VERSION) || _POSIX_VERSION < 200112L
extern char** environ;
#endif

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

// Performance metrics.
//
// Every field is measured. The previous version reported avg_latency_ms = 50,
// min = 10 and max = 500 as constants and left the byte counts and cache
// counters at zero, so the numbers did not depend on any request that was ever
// made.
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

// Sends one JSON-RPC request over HTTP and returns the response body.
//
// This is the transport the optimizer needs. Without it execute_single cannot
// be implemented, and it was in fact declared and never defined, so the whole
// request path failed to link.
class HttpTransport {
public:
    struct Result {
        bool success = false;
        long http_status = 0;
        std::string body;
        std::string error;
    };

    HttpTransport() { curl_global_init(CURL_GLOBAL_DEFAULT); }
    ~HttpTransport() { curl_global_cleanup(); }

    HttpTransport(const HttpTransport&) = delete;
    HttpTransport& operator=(const HttpTransport&) = delete;

    Result post(const std::string& url, const std::string& payload,
                const std::string& content_type, uint64_t timeout_ms) {
        Result result;
        CURL* curl = curl_easy_init();
        if (curl == nullptr) {
            result.error = "curl_easy_init failed";
            return result;
        }

        struct curl_slist* headers = nullptr;
        const std::string type_header = "Content-Type: " + content_type;
        headers = curl_slist_append(headers, type_header.c_str());

        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_POST, 1L);
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, payload.c_str());
        curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE,
                         static_cast<long>(payload.size()));
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, &HttpTransport::write_cb);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &result.body);
        curl_easy_setopt(curl, CURLOPT_TIMEOUT_MS,
                         static_cast<long>(timeout_ms == 0 ? 10000 : timeout_ms));
        curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT_MS, 3000L);
        curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
        curl_easy_setopt(curl, CURLOPT_MAXREDIRS, 3L);
        // TLS verification stays on. A node reached over a certificate that
        // does not check out is not a node to send signed transactions to.
        curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 1L);
        curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 2L);
        curl_easy_setopt(curl, CURLOPT_NOSIGNAL, 1L);
        curl_easy_setopt(curl, CURLOPT_USERAGENT, "TigerWallet/1.0");

        const CURLcode code = curl_easy_perform(curl);
        if (code != CURLE_OK) {
            result.error = curl_easy_strerror(code);
        } else {
            curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &result.http_status);
            result.success = true;
        }

        curl_slist_free_all(headers);
        curl_easy_cleanup(curl);
        return result;
    }

private:
    static size_t write_cb(char* data, size_t size, size_t nmemb, void* userdata) {
        const size_t total = size * nmemb;
        static_cast<std::string*>(userdata)->append(data, total);
        return total;
    }
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
    std::atomic<uint64_t> cache_hit_counter_{0};
    std::atomic<uint64_t> cache_miss_counter_{0};
    std::atomic<uint64_t> bytes_sent_counter_{0};
    std::atomic<uint64_t> bytes_received_counter_{0};

    // Latency of every completed request, so the average and the extremes come
    // from real measurements rather than constants.
    mutable std::mutex latency_mutex_;
    std::vector<double> latencies_ms_;

    HttpTransport transport_;

    uint64_t cache_ttl_ms_ = 5000;
    uint64_t health_check_interval_ms_ = 30000;
    uint64_t request_timeout_ms_ = 10000;

    // Bounded so a long-running process cannot grow without limit.
    static constexpr size_t kMaxLatencySamples = 100000;
    static constexpr size_t kMaxCacheEntries = 10000;

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
    void record_latency(double latency_ms);
};

// Inline implementations

// Maps the method enum to the JSON-RPC method name. The previous code had no
// such mapping, so a request had no way to reach a node.
inline const char* rpc_method_name(RPCMethod method) {
    switch (method) {
        case RPCMethod::ETH_BLOCK_NUMBER: return "eth_blockNumber";
        case RPCMethod::ETH_GET_BALANCE: return "eth_getBalance";
        case RPCMethod::ETH_CALL: return "eth_call";
        case RPCMethod::ETH_SEND_RAW_TRANSACTION: return "eth_sendRawTransaction";
        case RPCMethod::ETH_GET_TRANSACTION_BY_HASH: return "eth_getTransactionByHash";
        case RPCMethod::ETH_GET_TRANSACTION_RECEIPT: return "eth_getTransactionReceipt";
        case RPCMethod::ETH_GET_CODE: return "eth_getCode";
        case RPCMethod::ETH_ESTIMATE_GAS: return "eth_estimateGas";
        case RPCMethod::NET_VERSION: return "net_version";
        case RPCMethod::ETH_CHAIN_ID: return "eth_chainId";
        case RPCMethod::CUSTOM: return "custom";
    }
    return "custom";
}

// Escapes a string for inclusion in a JSON document.
inline std::string json_escape(const std::string& in) {
    std::string out;
    out.reserve(in.size() + 8);
    for (unsigned char c : in) {
        switch (c) {
            case '"': out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\n': out += "\\n"; break;
            case '\r': out += "\\r"; break;
            case '\t': out += "\\t"; break;
            case '\b': out += "\\b"; break;
            case '\f': out += "\\f"; break;
            default:
                if (c < 0x20) {
                    char buf[7];
                    std::snprintf(buf, sizeof(buf), "\\u%04x", c);
                    out += buf;
                } else {
                    out += static_cast<char>(c);
                }
        }
    }
    return out;
}

// Reads the host from a URL, for naming a provider after its endpoint.
inline std::string url_host(const std::string& url) {
    size_t start = url.find("://");
    start = (start == std::string::npos) ? 0 : start + 3;
    size_t end = url.find('/', start);
    if (end == std::string::npos) end = url.size();
    return url.substr(start, end - start);
}

inline RPCOptimizer::RPCOptimizer() {
    // Endpoints come from the environment. The previous constructor registered
    // "https://mainnet.infura.io/v3/" and
    // "https://eth-mainnet.g.alchemy.com/v2/" with no key, which cannot serve a
    // request, and reported latencies of 80 ms and 75 ms that were guesses.
    //
    // TIGER_RPC_<CHAIN> is a comma-separated list of URLs. The chain name is
    // lower-cased, so TIGER_RPC_ETHEREUM feeds the "ethereum" chain.
    if (const char* timeout = std::getenv("TIGER_RPC_TIMEOUT_MS")) {
        const uint64_t parsed = std::strtoull(timeout, nullptr, 10);
        if (parsed > 0) request_timeout_ms_ = parsed;
    }
    if (const char* ttl = std::getenv("TIGER_RPC_CACHE_TTL_MS")) {
        const uint64_t parsed = std::strtoull(ttl, nullptr, 10);
        if (parsed > 0) cache_ttl_ms_ = parsed;
    }

    for (char** env = environ; env != nullptr && *env != nullptr; ++env) {
        const std::string entry(*env);
        const std::string prefix = "TIGER_RPC_";
        if (entry.rfind(prefix, 0) != 0) continue;
        const size_t eq = entry.find('=');
        if (eq == std::string::npos) continue;

        std::string chain = entry.substr(prefix.size(), eq - prefix.size());
        std::transform(chain.begin(), chain.end(), chain.begin(),
                       [](unsigned char c) { return std::tolower(c); });
        if (chain == "timeout_ms" || chain == "cache_ttl_ms") continue;

        std::string urls = entry.substr(eq + 1);
        size_t pos = 0;
        while (pos <= urls.size()) {
            size_t comma = urls.find(',', pos);
            if (comma == std::string::npos) comma = urls.size();
            std::string url = urls.substr(pos, comma - pos);
            while (!url.empty() && std::isspace(static_cast<unsigned char>(url.front()))) url.erase(url.begin());
            while (!url.empty() && std::isspace(static_cast<unsigned char>(url.back()))) url.pop_back();
            if (!url.empty()) {
                // Latency starts at zero rather than a guess. The first
                // successful request records the real figure, and until then
                // providers are ranked by success rate alone.
                add_provider(chain, url_host(url), url, 0.0);
            }
            if (comma == urls.size()) break;
            pos = comma + 1;
        }
    }
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

    auto& providers = chain_providers_[chain];
    for (auto& existing : providers) {
        if (existing.name == name || existing.url == url) {
            existing = provider;
            return;
        }
    }
    providers.push_back(provider);
}

inline void RPCOptimizer::remove_provider(const std::string& chain, const std::string& name) {
    std::lock_guard<std::mutex> lock(providers_mutex_);
    auto it = chain_providers_.find(chain);
    if (it == chain_providers_.end()) return;
    auto& providers = it->second;
    providers.erase(std::remove_if(providers.begin(), providers.end(),
                                   [&](const RPCProvider& p) { return p.name == name; }),
                    providers.end());
    auto active = active_providers_.find(chain);
    if (active != active_providers_.end() && active->second && active->second->name == name) {
        active_providers_.erase(active);
    }
}

inline void RPCOptimizer::set_active_provider(const std::string& chain, const std::string& name) {
    std::lock_guard<std::mutex> lock(providers_mutex_);
    auto it = chain_providers_.find(chain);
    if (it == chain_providers_.end()) return;
    for (const auto& provider : it->second) {
        if (provider.name == name) {
            active_providers_[chain] = std::make_unique<RPCProvider>(provider);
            return;
        }
    }
}

inline void RPCOptimizer::record_latency(double latency_ms) {
    std::lock_guard<std::mutex> lock(latency_mutex_);
    if (latencies_ms_.size() >= kMaxLatencySamples) {
        latencies_ms_.erase(latencies_ms_.begin());
    }
    latencies_ms_.push_back(latency_ms);
}

inline std::optional<RPCResponse> RPCOptimizer::execute(const RPCRequest& request) {
    const auto start = std::chrono::high_resolution_clock::now();

    const std::string cache_key = generate_cache_key(request);

    // Only reads are cached. Caching a submission or a gas estimate keyed on
    // its arguments would hand back a stale answer, and for
    // eth_sendRawTransaction it would mean never broadcasting.
    const bool cacheable = request.method != RPCMethod::ETH_SEND_RAW_TRANSACTION &&
                           request.method != RPCMethod::ETH_ESTIMATE_GAS;

    if (cacheable) {
        if (auto cached = get_cached(cache_key)) {
            request_counter_++;
            cache_hit_counter_++;
            const auto end = std::chrono::high_resolution_clock::now();
            const double latency =
                std::chrono::duration<double, std::milli>(end - start).count();
            record_latency(latency);
            return RPCResponse{
                .id = request.id,
                .success = true,
                .result = *cached,
                .error = "",
                .latency_ms = static_cast<uint64_t>(latency),
                .provider = "CACHE"
            };
        }
        cache_miss_counter_++;
    }

    auto result = execute_single(request);

    if (result && result->success && cacheable) {
        set_cache(cache_key, result->result, cache_ttl_ms_);
    }

    return result;
}

inline std::vector<RPCResponse> RPCOptimizer::execute_batch(const std::vector<RPCRequest>& requests) {
    std::vector<RPCResponse> responses;
    responses.reserve(requests.size());
    for (const auto& request : requests) {
        if (auto response = execute(request)) {
            responses.push_back(*response);
        } else {
            RPCResponse failure;
            failure.id = request.id;
            failure.success = false;
            failure.error = "No provider available for chain " + request.chain;
            responses.push_back(failure);
        }
    }
    return responses;
}

inline std::optional<RPCResponse> RPCOptimizer::execute_single(const RPCRequest& request) {
    std::optional<RPCProvider> provider;
    {
        // The caller may have pinned a provider for this chain.
        std::lock_guard<std::mutex> lock(providers_mutex_);
        auto pinned = active_providers_.find(request.chain);
        if (pinned != active_providers_.end() && pinned->second) {
            provider = *pinned->second;
        }
    }
    if (!provider) {
        provider = get_best_provider(request.chain);
    }
    if (!provider) {
        return std::nullopt;
    }

    // Build the JSON-RPC envelope.
    std::string payload = "{\"jsonrpc\":\"2.0\",\"id\":\"";
    payload += json_escape(request.id);
    payload += "\",\"method\":\"";
    payload += json_escape(rpc_method_name(request.method));
    payload += "\",\"params\":[";
    for (size_t i = 0; i < request.params.size(); ++i) {
        if (i > 0) payload += ",";
        payload += "\"";
        payload += json_escape(request.params[i]);
        payload += "\"";
    }
    payload += "]}";

    const uint64_t timeout =
        request.timeout_ms > 0 ? request.timeout_ms : request_timeout_ms_;

    const auto start = std::chrono::high_resolution_clock::now();
    const HttpTransport::Result http =
        transport_.post(provider->url, payload, "application/json", timeout);
    const auto end = std::chrono::high_resolution_clock::now();
    const double latency_ms =
        std::chrono::duration<double, std::milli>(end - start).count();

    request_counter_++;
    bytes_sent_counter_ += payload.size();
    bytes_received_counter_ += http.body.size();
    record_latency(latency_ms);

    RPCResponse response;
    response.id = request.id;
    response.latency_ms = static_cast<uint64_t>(latency_ms);
    response.provider = provider->name;

    if (!http.success) {
        response.success = false;
        response.error = http.error;
        fail_counter_++;
        update_provider_stats(request.chain, provider->name, false,
                              static_cast<uint64_t>(latency_ms));
        return response;
    }

    if (http.http_status < 200 || http.http_status >= 300) {
        response.success = false;
        response.error = "HTTP " + std::to_string(http.http_status);
        fail_counter_++;
        update_provider_stats(request.chain, provider->name, false,
                              static_cast<uint64_t>(latency_ms));
        return response;
    }

    // A JSON-RPC reply carries either a result or an error. Both arrive with
    // HTTP 200, so the status code alone does not say whether the call worked.
    const std::string& body = http.body;
    if (body.find("\"error\"") != std::string::npos &&
        body.find("\"result\"") == std::string::npos) {
        response.success = false;
        const size_t message = body.find("\"message\"");
        response.error = (message == std::string::npos)
                             ? "JSON-RPC error"
                             : body.substr(message, 200);
        fail_counter_++;
        update_provider_stats(request.chain, provider->name, false,
                              static_cast<uint64_t>(latency_ms));
        return response;
    }

    const size_t result_pos = body.find("\"result\"");
    if (result_pos == std::string::npos) {
        response.success = false;
        response.error = "Malformed JSON-RPC reply";
        fail_counter_++;
        update_provider_stats(request.chain, provider->name, false,
                              static_cast<uint64_t>(latency_ms));
        return response;
    }

    size_t value_start = body.find(':', result_pos);
    if (value_start == std::string::npos) {
        response.success = false;
        response.error = "Malformed JSON-RPC reply";
        fail_counter_++;
        update_provider_stats(request.chain, provider->name, false,
                              static_cast<uint64_t>(latency_ms));
        return response;
    }
    ++value_start;
    while (value_start < body.size() &&
           std::isspace(static_cast<unsigned char>(body[value_start]))) {
        ++value_start;
    }

    if (value_start < body.size() && body[value_start] == '"') {
        // A string result. Copy until the closing quote, honouring escapes.
        ++value_start;
        std::string value;
        for (size_t i = value_start; i < body.size(); ++i) {
            if (body[i] == '\\' && i + 1 < body.size()) {
                value += body[i + 1];
                ++i;
                continue;
            }
            if (body[i] == '"') break;
            value += body[i];
        }
        response.result = value;
    } else if (value_start < body.size() && body[value_start] == 'n' &&
               body.compare(value_start, 4, "null") == 0) {
        // eth_getTransactionReceipt returns null while a transaction is
        // pending. That is a successful call with an empty result, not a
        // failure.
        response.result = "";
    } else {
        // A number, or an object or array for calls that return structure.
        // Objects and arrays are returned as written so the caller can decode
        // them, rather than being silently discarded.
        size_t value_end = value_start;
        if (value_start < body.size() && (body[value_start] == '{' || body[value_start] == '[')) {
            const char open = body[value_start];
            const char close = (open == '{') ? '}' : ']';
            int depth = 0;
            bool in_string = false;
            for (size_t i = value_start; i < body.size(); ++i) {
                const char c = body[i];
                if (in_string) {
                    if (c == '\\') { ++i; continue; }
                    if (c == '"') in_string = false;
                    continue;
                }
                if (c == '"') { in_string = true; continue; }
                if (c == open) ++depth;
                else if (c == close) {
                    --depth;
                    if (depth == 0) { value_end = i + 1; break; }
                }
            }
        } else {
            while (value_end < body.size() && body[value_end] != ',' &&
                   body[value_end] != '}' &&
                   !std::isspace(static_cast<unsigned char>(body[value_end]))) {
                ++value_end;
            }
        }
        response.result = body.substr(value_start, value_end - value_start);
    }

    response.success = true;
    success_counter_++;
    update_provider_stats(request.chain, provider->name, true,
                          static_cast<uint64_t>(latency_ms));
    return response;
}

inline std::string RPCOptimizer::generate_cache_key(const RPCRequest& request) {
    // The method name is used rather than the enum's integer value, so the key
    // stays meaningful if the enum is reordered.
    std::string key = request.chain + "|" + rpc_method_name(request.method);
    for (const auto& param : request.params) {
        key += "|" + param;
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
    return elapsed >= 0 && elapsed < static_cast<int64_t>(entry.ttl_ms);
}

inline void RPCOptimizer::set_cache(const std::string& key, const std::string& value, uint64_t ttl_ms) {
    std::lock_guard<std::mutex> lock(cache_mutex_);

    // Drop expired entries before adding, and evict the oldest if still full,
    // so the cache cannot grow without bound.
    if (cache_.size() >= kMaxCacheEntries) {
        for (auto it = cache_.begin(); it != cache_.end();) {
            if (!is_cache_valid(it->second)) {
                it = cache_.erase(it);
            } else {
                ++it;
            }
        }
        while (cache_.size() >= kMaxCacheEntries) {
            auto oldest = cache_.begin();
            for (auto it = cache_.begin(); it != cache_.end(); ++it) {
                if (it->second.created < oldest->second.created) oldest = it;
            }
            cache_.erase(oldest);
        }
    }

    CacheEntry entry;
    entry.key = key;
    entry.value = value;
    entry.created = std::chrono::steady_clock::now();
    entry.ttl_ms = ttl_ms;

    cache_[key] = entry;
}

inline void RPCOptimizer::clear_cache() {
    std::lock_guard<std::mutex> lock(cache_mutex_);
    cache_.clear();
}

inline void RPCOptimizer::clear_cache(const std::string& chain) {
    std::lock_guard<std::mutex> lock(cache_mutex_);
    for (auto it = cache_.begin(); it != cache_.end();) {
        if (it->first.rfind(chain + "|", 0) == 0) {
            it = cache_.erase(it);
        } else {
            ++it;
        }
    }
}

inline void RPCOptimizer::check_provider_health(const std::string& chain) {
    std::vector<RPCProvider> snapshot;
    {
        std::lock_guard<std::mutex> lock(providers_mutex_);
        auto it = chain_providers_.find(chain);
        if (it == chain_providers_.end()) return;
        snapshot = it->second;
    }

    // Ask each endpoint for its chain id. A provider that cannot answer is
    // marked unhealthy so it stops receiving requests.
    for (const auto& provider : snapshot) {
        RPCRequest probe;
        probe.id = "health-" + provider.name;
        probe.method = RPCMethod::ETH_CHAIN_ID;
        probe.chain = chain;
        probe.timeout_ms = 5000;

        const std::string payload =
            "{\"jsonrpc\":\"2.0\",\"id\":\"health\",\"method\":\"eth_chainId\",\"params\":[]}";
        const HttpTransport::Result http =
            transport_.post(provider.url, payload, "application/json", 5000);
        const bool healthy = http.success && http.http_status >= 200 &&
                             http.http_status < 300 &&
                             http.body.find("\"result\"") != std::string::npos;

        std::lock_guard<std::mutex> lock(providers_mutex_);
        auto it = chain_providers_.find(chain);
        if (it == chain_providers_.end()) continue;
        for (auto& entry : it->second) {
            if (entry.name == provider.name) {
                entry.is_healthy = healthy;
                entry.last_check = std::chrono::steady_clock::now();
            }
        }
    }
}

inline std::vector<RPCProvider> RPCOptimizer::get_providers(const std::string& chain) {
    std::lock_guard<std::mutex> lock(providers_mutex_);
    auto it = chain_providers_.find(chain);
    if (it == chain_providers_.end()) return {};
    return it->second;
}

inline std::optional<RPCProvider> RPCOptimizer::get_best_provider(const std::string& chain) {
    std::lock_guard<std::mutex> lock(providers_mutex_);

    auto it = chain_providers_.find(chain);
    if (it == chain_providers_.end() || it->second.empty()) {
        return std::nullopt;
    }

    // Rank by success rate, and use latency to separate providers that are
    // otherwise equal. A provider with no latency recorded yet (0.0) is not
    // treated as infinitely fast; it is scored on success rate, and once a
    // request completes its real latency replaces the zero.
    const RPCProvider* best = nullptr;
    double best_score = -1.0;

    for (const auto& provider : it->second) {
        if (!provider.is_healthy) continue;

        double score = provider.success_rate;
        if (provider.latency_ms > 0.0) {
            score = provider.success_rate * 1000.0 / (provider.latency_ms + 1.0);
        }
        if (score > best_score) {
            best_score = score;
            best = &provider;
        }
    }

    if (best == nullptr) {
        return std::nullopt;
    }
    return *best;
}

inline void RPCOptimizer::update_provider_stats(const std::string& chain, const std::string& provider,
                                             bool success, uint64_t latency_ms) {
    std::lock_guard<std::mutex> lock(providers_mutex_);
    auto it = chain_providers_.find(chain);
    if (it == chain_providers_.end()) return;

    for (auto& entry : it->second) {
        if (entry.name != provider) continue;
        entry.requests_total++;
        if (!success) {
            entry.requests_failed++;
        }
        // The success rate is the observed one, not an assumption.
        entry.success_rate =
            static_cast<double>(entry.requests_total - entry.requests_failed) /
            static_cast<double>(entry.requests_total);
        if (success && latency_ms > 0) {
            // A running average, weighted by the number of requests seen.
            const double observed = static_cast<double>(latency_ms);
            if (entry.latency_ms <= 0.0) {
                entry.latency_ms = observed;
            } else {
                entry.latency_ms =
                    entry.latency_ms + (observed - entry.latency_ms) /
                                           static_cast<double>(entry.requests_total);
            }
        }
        // Three consecutive failures take a provider out of rotation.
        if (!success && entry.requests_failed > 0 && entry.requests_total > 0) {
            const uint64_t recent = entry.requests_failed;
            if (recent >= 3 &&
                static_cast<double>(recent) / static_cast<double>(entry.requests_total) > 0.5) {
                entry.is_healthy = false;
            }
        }
        break;
    }
}

inline OptimizerMetrics RPCOptimizer::get_metrics() const {
    OptimizerMetrics metrics{};
    metrics.total_requests = request_counter_.load();
    metrics.successful_requests = success_counter_.load();
    metrics.failed_requests = fail_counter_.load();
    metrics.cache_hits = cache_hit_counter_.load();
    metrics.cache_misses = cache_miss_counter_.load();
    metrics.total_bytes_sent = bytes_sent_counter_.load();
    metrics.total_bytes_received = bytes_received_counter_.load();

    // The latency figures are the measured ones. The previous version returned
    // 50.0, 10.0 and 500.0 as constants regardless of traffic.
    std::lock_guard<std::mutex> lock(latency_mutex_);
    if (latencies_ms_.empty()) {
        metrics.avg_latency_ms = 0.0;
        metrics.min_latency_ms = 0.0;
        metrics.max_latency_ms = 0.0;
    } else {
        double total = 0.0;
        double lowest = latencies_ms_.front();
        double highest = latencies_ms_.front();
        for (double value : latencies_ms_) {
            total += value;
            if (value < lowest) lowest = value;
            if (value > highest) highest = value;
        }
        metrics.avg_latency_ms = total / static_cast<double>(latencies_ms_.size());
        metrics.min_latency_ms = lowest;
        metrics.max_latency_ms = highest;
    }
    return metrics;
}

inline void RPCOptimizer::reset_metrics() {
    request_counter_ = 0;
    success_counter_ = 0;
    fail_counter_ = 0;
    cache_hit_counter_ = 0;
    cache_miss_counter_ = 0;
    bytes_sent_counter_ = 0;
    bytes_received_counter_ = 0;
    std::lock_guard<std::mutex> lock(latency_mutex_);
    latencies_ms_.clear();
}

} // namespace rpc
} // namespace tiger

#endif // TIGER_RPC_OPTIMIZER_H
