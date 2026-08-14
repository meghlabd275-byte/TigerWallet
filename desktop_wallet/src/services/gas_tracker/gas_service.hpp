#ifndef TIGERWALLET_GAS_SERVICE_HPP
#define TIGERWALLET_GAS_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <functional>
#include <chrono>
#include <thread>
#include <mutex>
#include <atomic>
#include <cmath>
#include <algorithm>
#include <sstream>
#include <iomanip>
#include <cstdlib>

// Forward declaration for curl
struct Curl;
#include <curl/curl.h>

// =============================================================================
// GAS PRICE STRUCTURES
// =============================================================================

namespace tigerwallet {

struct GasPrice {
    std::string chain;
    uint64_t slow_gas_price;
    uint64_t standard_gas_price;
    uint64_t fast_gas_price;
    uint64_t base_fee;
    uint64_t priority_fee;
    double usd_per_gwei;
    uint64_t timestamp;
    bool is_eip1559;
    
    GasPrice() : chain(""), slow_gas_price(0), standard_gas_price(0), 
                 fast_gas_price(0), base_fee(0), priority_fee(0),
                 usd_per_gwei(0.0), timestamp(0), is_eip1559(false) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"chain\":\"" << chain << "\",";
        oss << "\"slow\":" << slow_gas_price << ",";
        oss << "\"standard\":" << standard_gas_price << ",";
        oss << "\"fast\":" << fast_gas_price << ",";
        oss << "\"baseFee\":" << base_fee << ",";
        oss << "\"priorityFee\":" << priority_fee << ",";
        oss << "\"usdPerGwei\":" << std::fixed << std::setprecision(6) << usd_per_gwei << ",";
        oss << "\"timestamp\":" << timestamp << ",";
        oss << "\"isEip1559\":" << (is_eip1559 ? "true" : "false");
        oss << "}";
        return oss.str();
    }
};

struct GasEstimate {
    std::string from;
    std::string to;
    std::string value;
    std::string data;
    uint64_t estimated_gas;
    uint64_t gas_price;
    uint64_t total_cost;
    double total_cost_usd;
    bool success;
    std::string error_message;
    
    GasEstimate() : estimated_gas(0), gas_price(0), total_cost(0), 
                    total_cost_usd(0.0), success(true), error_message("") {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"from\":\"" << from << "\",";
        oss << "\"to\":\"" << to << "\",";
        oss << "\"value\":\"" << value << "\",";
        oss << "\"estimatedGas\":" << estimated_gas << ",";
        oss << "\"gasPrice\":" << gas_price << ",";
        oss << "\"totalCost\":" << total_cost << ",";
        oss << "\"totalCostUsd\":" << std::fixed << std::setprecision(2) << total_cost_usd << ",";
        oss << "\"success\":" << (success ? "true" : "false") << ",";
        oss << "\"errorMessage\":\"" << error_message << "\"";
        oss << "}";
        return oss.str();
    }
};

// =============================================================================
// CHAIN CONFIGURATION
// =============================================================================

struct ChainGasConfig {
    std::string name;
    std::string rpc_url;
    std::string chain_id;
    bool supports_eip1559;
    uint64_t default_gas_limit;
    uint64_t min_gas_price;
    uint64_t max_gas_price;
    double usd_price;
    
    ChainGasConfig() : name(""), rpc_url(""), chain_id(""), 
                       supports_eip1559(false), default_gas_limit(21000),
                       min_gas_price(1), max_gas_price(1000), usd_price(0.0) {}
    
    static std::map<std::string, ChainGasConfig> getDefaultConfigs() {
        std::map<std::string, ChainGasConfig> configs;
        
        configs["ethereum"] = []{
            ChainGasConfig c;
            c.name = "Ethereum";
            // Env-overridable RPC; default to a public endpoint (not a shared
            // Alchemy demo key, which is rate-limited and unreliable).
            const char* rpc = std::getenv("ETH_RPC_URL");
            c.rpc_url = rpc ? rpc : "https://ethereum-rpc.publicnode.com";
            c.chain_id = "1";
            c.supports_eip1559 = true;
            c.default_gas_limit = 21000;
            c.min_gas_price = 1;
            c.max_gas_price = 500;
            c.usd_price = 3500.0;
            return c;
        }();
        
        configs["polygon"] = []{
            ChainGasConfig c;
            c.name = "Polygon";
            const char* rpc = std::getenv("POLYGON_RPC_URL");
            c.rpc_url = rpc ? rpc : "https://polygon-bor-rpc.publicnode.com";
            c.chain_id = "137";
            c.supports_eip1559 = true;
            c.default_gas_limit = 21000;
            c.min_gas_price = 1;
            c.max_gas_price = 200;
            c.usd_price = 0.85;
            return c;
        }();
        
        configs["bsc"] = []{
            ChainGasConfig c;
            c.name = "BNB Chain";
            c.rpc_url = "https://bsc-dataseed.binance.org";
            c.chain_id = "56";
            c.supports_eip1559 = false;
            c.default_gas_limit = 21000;
            c.min_gas_price = 1;
            c.max_gas_price = 100;
            c.usd_price = 600.0;
            return c;
        }();
        
        configs["avalanche"] = []{
            ChainGasConfig c;
            c.name = "Avalanche";
            c.rpc_url = "https://api.avax.network/ext/bc/C/rpc";
            c.chain_id = "43114";
            c.supports_eip1559 = true;
            c.default_gas_limit = 21000;
            c.min_gas_price = 1;
            c.max_gas_price = 200;
            c.usd_price = 35.0;
            return c;
        }();
        
        configs["arbitrum"] = []{
            ChainGasConfig c;
            c.name = "Arbitrum One";
            c.rpc_url = "https://arb1.arbitrum.io/rpc";
            c.chain_id = "42161";
            c.supports_eip1559 = true;
            c.default_gas_limit = 21000;
            c.min_gas_price = 1;
            c.max_gas_price = 10;
            c.usd_price = 3500.0;
            return c;
        }();
        
        configs["optimism"] = []{
            ChainGasConfig c;
            c.name = "Optimism";
            c.rpc_url = "https://mainnet.optimism.io";
            c.chain_id = "10";
            c.supports_eip1559 = true;
            c.default_gas_limit = 21000;
            c.min_gas_price = 1;
            c.max_gas_price = 10;
            c.usd_price = 3500.0;
            return c;
        }();
        
        configs["fantom"] = []{
            ChainGasConfig c;
            c.name = "Fantom Opera";
            c.rpc_url = "https://rpc.fantom.network";
            c.chain_id = "250";
            c.supports_eip1559 = false;
            c.default_gas_limit = 21000;
            c.min_gas_price = 1;
            c.max_gas_price = 100;
            c.usd_price = 0.8;
            return c;
        }();
        
        return configs;
    }
};

// =============================================================================
// GAS SERVICE - MAIN IMPLEMENTATION (No external dependencies)
// =============================================================================

class GasService {
private:
    std::map<std::string, ChainGasConfig> chain_configs;
    std::map<std::string, GasPrice> cached_prices;
    std::mutex cache_mutex;
    std::atomic<uint64_t> last_update{0};
    bool initialized;
    
    static constexpr uint64_t CACHE_TTL_SECONDS = 15;
    
    uint64_t parseHexToUint64(const std::string& hex) const {
        if (hex.empty() || hex.length() < 3) return 0;
        std::string clean_hex = hex;
        if (clean_hex.substr(0, 2) == "0x") {
            clean_hex = clean_hex.substr(2);
        }
        return std::stoull(clean_hex, nullptr, 16);
    }
    
    bool isCacheValid() const {
        uint64_t now = std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
        return (now - last_update.load()) < CACHE_TTL_SECONDS;
    }
    
    void updateCache(const std::string& chain, const GasPrice& price) {
        std::lock_guard<std::mutex> lock(cache_mutex);
        cached_prices[chain] = price;
        last_update = std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
    
    GasPrice fetchGasPriceInternal(const std::string& chain) {
        auto it = chain_configs.find(chain);
        if (it == chain_configs.end()) {
            throw std::runtime_error("Unsupported chain: " + chain);
        }
        
        const ChainGasConfig& config = it->second;
        GasPrice price;
        price.chain = chain;
        price.timestamp = std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
        
        // Fetch REAL gas prices from the wallet_api backend (which calls
        // eth_feeHistory / eth_gasPrice over RPC). The previous implementation
        // fabricated "realistic" prices from the static chain config
        // (base*1.2, base*1.5) - that was a fake. On any failure we throw
        // rather than return fabricated values.
        const char* baseEnv = std::getenv("WALLET_API_URL");
        std::string baseUrl = baseEnv && baseEnv[0] ? std::string(baseEnv) : std::string("http://localhost:8443");
        std::string url = baseUrl + "/api/v1/gas?chain_id=" + config.chain_id;
        std::string response;
        CURL* curl = curl_easy_init();
        if (!curl) throw std::runtime_error("gas: curl init failed");
        auto writeCb = +[](char* ptr, size_t size, size_t nmemb, std::string* data) -> size_t {
            data->append(ptr, size * nmemb);
            return size * nmemb;
        };
        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_TIMEOUT, 8L);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeCb);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
        CURLcode res = curl_easy_perform(curl);
        long httpCode = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);
        curl_easy_cleanup(curl);
        if (res != CURLE_OK || httpCode < 200 || httpCode >= 300) {
            throw std::runtime_error("gas: backend unavailable for chain " + chain);
        }
        auto extractStr = [&](const std::string& key) -> std::string {
            std::string k = "\"" + key + "\":";
            auto pos = response.find(k);
            if (pos == std::string::npos) return "";
            auto st = pos + k.size();
            while (st < response.size() && (response[st] == ' ' || response[st] == '"')) st++;
            auto en = st;
            while (en < response.size() && response[en] != ',' && response[en] != '}' && response[en] != '"') en++;
            return response.substr(st, en - st);
        };
        auto extractDbl = [&](const std::string& key) -> double {
            std::string v = extractStr(key);
            return v.empty() ? 0.0 : std::stod(v);
        };
        double gwei = extractDbl("gas_price_gwei");
        double maxFeeGwei = extractDbl("max_fee_per_gas") / 1e9;
        double prioGwei = extractDbl("max_priority_fee") / 1e9;
        if (gwei <= 0 && maxFeeGwei <= 0) {
            throw std::runtime_error("gas: no price returned for chain " + chain);
        }
        double baseGwei = (gwei > 0 ? gwei : maxFeeGwei);
        price.slow_gas_price = static_cast<uint64_t>(baseGwei * 0.9);
        price.standard_gas_price = static_cast<uint64_t>(baseGwei);
        price.fast_gas_price = static_cast<uint64_t>(baseGwei * 1.2);
        if (config.supports_eip1559 && maxFeeGwei > 0) {
            price.is_eip1559 = true;
            price.base_fee = static_cast<uint64_t>(maxFeeGwei);
            price.priority_fee = static_cast<uint64_t>(prioGwei > 0 ? prioGwei : baseGwei * 0.1);
        }
        price.slow_gas_price = std::min(price.slow_gas_price, config.max_gas_price);
        price.standard_gas_price = std::min(price.standard_gas_price, config.max_gas_price);
        price.fast_gas_price = std::min(price.fast_gas_price, config.max_gas_price);
        price.usd_per_gwei = config.usd_price / 1e9;
        return price;
    }
public:
    GasService() : initialized(false) {
        chain_configs = ChainGasConfig::getDefaultConfigs();
    }
    
    ~GasService() {}
    
    bool initialize() {
        if (initialized) return true;
        
        // Pre-populate cache for all chains
        for (const auto& config : chain_configs) {
            try {
                GasPrice price = fetchGasPriceInternal(config.first);
                updateCache(config.first, price);
            } catch (...) {
                // Continue with other chains
            }
        }
        
        initialized = true;
        return true;
    }
    
    // Public API - Get current gas price for a chain
    GasPrice getGasPrice(const std::string& chain) {
        std::string lower_chain = chain;
        std::transform(lower_chain.begin(), lower_chain.end(), lower_chain.begin(), ::tolower);
        
        // Check cache first
        {
            std::lock_guard<std::mutex> lock(cache_mutex);
            auto it = cached_prices.find(lower_chain);
            if (it != cached_prices.end() && isCacheValid()) {
                return it->second;
            }
        }
        
        // Fetch fresh price
        try {
            GasPrice price = fetchGasPriceInternal(lower_chain);
            updateCache(lower_chain, price);
            return price;
        } catch (const std::exception& e) {
            // Return cached if available, even if expired
            std::lock_guard<std::mutex> lock(cache_mutex);
            auto it = cached_prices.find(lower_chain);
            if (it != cached_prices.end()) {
                return it->second;
            }
            throw;
        }
    }
    
    // Get gas price with speed option (slow, standard, fast)
    uint64_t getGasPriceBySpeed(const std::string& chain, const std::string& speed) {
        GasPrice price = getGasPrice(chain);
        
        if (speed == "slow" || speed == "SLOW") {
            return price.slow_gas_price;
        } else if (speed == "fast" || speed == "FAST") {
            return price.fast_gas_price;
        }
        return price.standard_gas_price;
    }
    
    // Estimate gas for a transaction
    GasEstimate estimateGas(const std::string& chain, 
                          const std::string& from,
                          const std::string& to,
                          const std::string& value,
                          const std::string& data = "") {
        GasEstimate estimate;
        estimate.from = from;
        estimate.to = to;
        estimate.value = value;
        estimate.data = data;
        
        try {
            auto it = chain_configs.find(chain);
            if (it == chain_configs.end()) {
                estimate.success = false;
                estimate.error_message = "Unsupported chain: " + chain;
                return estimate;
            }
            
            const ChainGasConfig& config = it->second;
            GasPrice gas_price = getGasPrice(chain);
            
            // For simple ETH transfers, use default gas limit
            if (data.empty() && value != "0x0") {
                estimate.estimated_gas = 21000;
            } else {
                // For contract interactions, use higher gas limit
                estimate.estimated_gas = config.default_gas_limit;
            }
            
            // Add 20% buffer for safety
            estimate.estimated_gas = static_cast<uint64_t>(estimate.estimated_gas * 1.2);
            
            // Calculate costs
            estimate.gas_price = gas_price.standard_gas_price * 1000000000;
            estimate.total_cost = estimate.estimated_gas * estimate.gas_price;
            estimate.total_cost_usd = (estimate.total_cost / 1e18) * config.usd_price;
            
            estimate.success = true;
            
        } catch (const std::exception& e) {
            estimate.success = false;
            estimate.error_message = e.what();
            estimate.estimated_gas = 21000;
        }
        
        return estimate;
    }
    
    // Get estimated cost in USD for a standard transaction
    double estimateTransactionCostUsd(const std::string& chain) {
        try {
            GasPrice price = getGasPrice(chain);
            auto it = chain_configs.find(chain);
            if (it != chain_configs.end()) {
                uint64_t gas_wei = price.standard_gas_price * 1000000000 * 21000;
                return (gas_wei / 1e18) * it->second.usd_price;
            }
        } catch (...) {}
        return 0.0;
    }
    
    // Get all supported chains
    std::vector<std::string> getSupportedChains() const {
        std::vector<std::string> chains;
        for (const auto& config : chain_configs) {
            chains.push_back(config.first);
        }
        return chains;
    }
    
    // Add custom chain
    void addChain(const ChainGasConfig& config) {
        chain_configs[config.name] = config;
    }
    
    // Get chain configuration
    ChainGasConfig getChainConfig(const std::string& chain) const {
        auto it = chain_configs.find(chain);
        if (it != chain_configs.end()) {
            return it->second;
        }
        return ChainGasConfig();
    }
    
    // Refresh all gas prices
    void refreshAllPrices() {
        for (const auto& config : chain_configs) {
            try {
                GasPrice price = fetchGasPriceInternal(config.first);
                updateCache(config.first, price);
            } catch (...) {
                // Continue with other chains
            }
        }
    }
};

} // namespace tigerwallet

#endif // TIGERWALLET_GAS_SERVICE_HPP
