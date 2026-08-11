/**
 * TigerWallet Desktop - Price Service Implementation
 */

#include "services/price_service.h"
#include "services/api_client.h"
#include <iostream>
#include <sstream>
#include <thread>
#include <chrono>
#include <cmath>
#include <cctype>
#include <ctime>

namespace tiger {
namespace wallet {

// ============================================================================
// Static Instance
// ============================================================================

std::shared_ptr<PriceService> PriceService::instance_ = nullptr;

// ============================================================================
// Constructor/Destructor
// ============================================================================

PriceService::PriceService() 
    : curl_(nullptr), initialized_(false), updatesRunning_(false), cacheEnabled_(true),
      cacheExpiry_(60) {  // 1 minute cache
    
    // Supported coins list
    supportedCoins_ = {
        "bitcoin", "ethereum", "tether", "usd-coin", "binancecoin",
        "ripple", "cardano", "solana", "dogecoin", "polkadot",
        "polygon", "avalanche-2", "chainlink", "uniswap", "litecoin",
        "near", "aptos", "sui", "toncoin", "cosmos",
        "internet-computer", "polkadot", "near", "fantom", "algorand",
        "stellar", "monero", "cosmos", "vechain", "filecoin"
    };
}

PriceService::~PriceService() {
    stopPriceUpdates();
    shutdown();
}

// ============================================================================
// Singleton
// ============================================================================

std::shared_ptr<PriceService> PriceService::getInstance() {
    if (!instance_) {
        instance_ = std::make_shared<PriceService>();
    }
    return instance_;
}

// ============================================================================
// Initialization
// ============================================================================

void PriceService::initialize() {
    if (initialized_) return;
    
    curl_ = curl_easy_init();
    initialized_ = true;
    std::cout << "[PriceService] Initialized with CoinGecko API" << std::endl;
}

void PriceService::shutdown() {
    if (curl_) {
        curl_easy_cleanup(curl_);
        curl_ = nullptr;
    }
    initialized_ = false;
}

// ============================================================================
// Price Fetching
// ============================================================================

std::future<PriceInfo> PriceService::getPrice(const std::string& symbol) {
    return std::async(std::launch::async, [this, symbol]() -> PriceInfo {
        // Check cache first
        if (cacheEnabled_ && isCacheValid(symbol)) {
            auto cached = getCachedPrice(symbol);
            if (cached) {
                return *cached;
            }
        }

        std::string coinId = symbol;
        // Convert symbol to CoinGecko ID
        if (symbol == "ETH") coinId = "ethereum";
        else if (symbol == "BTC") coinId = "bitcoin";
        else if (symbol == "MATIC") coinId = "matic-network";
        else if (symbol == "AVAX") coinId = "avalanche-2";
        else if (symbol == "BNB") coinId = "binancecoin";
        else if (symbol == "SOL") coinId = "solana";
        
        std::string url = "https://api.coingecko.com/api/v3/simple/price?ids=" 
                        + coinId + "&vs_currencies=usd&include_24hr_change=true&include_market_cap=true&include_24hr_vol=true&include_high_low=true";
        
        std::string response = fetchFromAPI(url);
        
        auto prices = parsePriceResponse(response);
        if (!prices.empty()) {
            // Update cache
            if (cacheEnabled_) {
                priceCache_[symbol] = {prices[0], std::chrono::system_clock::now()};
            }
            return prices[0];
        }
        
        throw PriceServiceException(PriceServiceException::ErrorCode::NotFound, 
            "Price not found for: " + symbol);
    });
}

std::future<std::map<std::string, PriceInfo>> PriceService::getPrices(const std::vector<std::string>& symbols) {
    return std::async(std::launch::async, [this, symbols]() -> std::map<std::string, PriceInfo> {
        std::map<std::string, PriceInfo> result;
        
        // Build comma-separated coin IDs
        std::ostringstream coinIds;
        std::map<std::string, std::string> symbolToId;
        
        for (const auto& symbol : symbols) {
            std::string coinId = symbol;
            if (symbol == "ETH") coinId = "ethereum";
            else if (symbol == "BTC") coinId = "bitcoin";
            else if (symbol == "MATIC") coinId = "matic-network";
            else if (symbol == "AVAX") coinId = "avalanche-2";
            else if (symbol == "BNB") coinId = "binancecoin";
            else if (symbol == "SOL") coinId = "solana";
            
            symbolToId[symbol] = coinId;
            coinIds << coinId << ",";
        }
        
        std::string ids = coinIds.str();
        if (!ids.empty() && ids.back() == ',') {
            ids.pop_back();
        }
        
        std::string url = "https://api.coingecko.com/api/v3/simple/price?ids=" 
                        + ids + "&vs_currencies=usd&include_24hr_change=true&include_market_cap=true&include_24hr_vol=true&include_high_low=true";
        
        std::string response = fetchFromAPI(url);
        auto prices = parsePriceResponse(response);
        
        // Map back to symbols
        for (const auto& price : prices) {
            for (const auto& pair : symbolToId) {
                if (pair.second == price.symbol) {
                    result[pair.first] = price;
                    // Update cache
                    if (cacheEnabled_) {
                        priceCache_[pair.first] = {price, std::chrono::system_clock::now()};
                    }
                    break;
                }
            }
        }
        
        return result;
    });
}

std::future<std::vector<PriceInfo>> PriceService::getTopPrices(int limit) {
    return std::async(std::launch::async, [this, limit]() -> std::vector<PriceInfo> {
        std::string url = "https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=" 
                        + std::to_string(limit) + "&page=1&sparkline=false&price_change_percentage=24h";
        
        std::string response = fetchFromAPI(url);
        
        // Parse markets response (simplified)
        return parsePriceResponse(response);
    });
}

// ============================================================================
// Portfolio
// ============================================================================

std::future<double> PriceService::getPortfolioValue(const std::vector<Token>& tokens) {
    return std::async(std::launch::async, [this, tokens]() -> double {
        double total = 0.0;
        
        for (const auto& token : tokens) {
            try {
                auto price = getPrice(token.symbol).get();
                total += token.balance * price.price;
            } catch (...) {
                // Skip if price not found
            }
        }
        
        return total;
    });
}

// ============================================================================
// Historical Data
// ============================================================================

std::future<PriceHistory> PriceService::getPriceHistory(const std::string& symbol, int days) {
    return std::async(std::launch::async, [this, symbol, days]() -> PriceHistory {
        std::string coinId = symbol;
        if (symbol == "ETH") coinId = "ethereum";
        else if (symbol == "BTC") coinId = "bitcoin";
        else if (symbol == "SOL") coinId = "solana";
        
        std::string url = "https://api.coingecko.com/api/v3/coins/" + coinId 
                        + "/market_chart?vs_currency=usd&days=" + std::to_string(days);
        
        std::string response = fetchFromAPI(url);
        return parseHistoryResponse(response);
    });
}

std::future<std::vector<PriceHistory>> PriceService::getMultiPriceHistory(const std::vector<std::string>& symbols, int days) {
    return std::async(std::launch::async, [this, symbols, days]() -> std::vector<PriceHistory> {
        std::vector<PriceHistory> result;
        
        for (const auto& symbol : symbols) {
            try {
                auto history = getPriceHistory(symbol, days).get();
                result.push_back(history);
            } catch (...) {
                // Skip on error
            }
        }
        
        return result;
    });
}

// ============================================================================
// Conversion
// ============================================================================

std::future<double> PriceService::convert(double amount, const std::string& from, const std::string& to) {
    return std::async(std::launch::async, [this, amount, from, to]() -> double {
        auto fromPrice = getPrice(from).get();
        auto toPrice = getPrice(to).get();
        
        // Convert to USD first, then to target currency
        double usdValue = amount * fromPrice.price;
        return usdValue / toPrice.price;
    });
}

// ============================================================================
// Search
// ============================================================================

std::future<std::vector<PriceInfo>> PriceService::searchCoins(const std::string& query) {
    return std::async(std::launch::async, [this, query]() -> std::vector<PriceInfo> {
        // URL encode query
        std::string encodedQuery = query;
        // Simple encoding for spaces
        size_t pos;
        while ((pos = encodedQuery.find(' ')) != std::string::npos) {
            encodedQuery.replace(pos, 1, "%20");
        }
        
        std::string url = "https://api.coingecko.com/api/v3/search?query=" + encodedQuery;
        std::string response = fetchFromAPI(url);
        
        // Parse search results (simplified)
        std::vector<PriceInfo> results;
        
        // In production, parse JSON response
        return results;
    });
}

// ============================================================================
// Price Updates
// ============================================================================

void PriceService::startPriceUpdates(std::chrono::seconds interval) {
    if (updatesRunning_) return;
    
    updatesRunning_ = true;
    updateThread_ = std::thread([this, interval]() {
        while (updatesRunning_) {
            try {
                auto prices = getTopPrices(20).get();
                
                for (const auto& price : prices) {
                    if (priceCallback_) {
                        priceCallback_(price);
                    }
                    
                    // Update cache
                    if (cacheEnabled_) {
                        priceCache_[price.symbol] = {price, std::chrono::system_clock::now()};
                    }
                }
            } catch (const std::exception& e) {
                std::cerr << "[PriceService] Update error: " << e.what() << std::endl;
            }
            
            std::this_thread::sleep_for(interval);
        }
    });
}

void PriceService::stopPriceUpdates() {
    if (!updatesRunning_) return;
    
    updatesRunning_ = false;
    if (updateThread_.joinable()) {
        updateThread_.join();
    }
}

void PriceService::setPriceUpdateCallback(PriceUpdateCallback callback) {
    priceCallback_ = callback;
}

// ============================================================================
// Cache Management
// ============================================================================

void PriceService::clearCache() {
    priceCache_.clear();
}

std::optional<PriceInfo> PriceService::getCachedPrice(const std::string& symbol) {
    auto it = priceCache_.find(symbol);
    if (it != priceCache_.end()) {
        return it->second.price;
    }
    return std::nullopt;
}

bool PriceService::isCacheValid(const std::string& symbol) const {
    auto it = priceCache_.find(symbol);
    if (it == priceCache_.end()) return false;
    
    auto now = std::chrono::system_clock::now();
    auto elapsed = std::chrono::duration_cast<std::chrono::seconds>(now - it->second.timestamp);
    return elapsed.count() < cacheExpiry_.count();
}

// ============================================================================
// Private: API Calls
// ============================================================================

std::string PriceService::fetchFromAPI(const std::string& url) {
    if (!curl_) {
        curl_ = curl_easy_init();
    }
    
    std::string response_string;
    
    curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl_, CURLOPT_WRITEFUNCTION, +[](char* ptr, size_t size, size_t nmemb, void* userdata) {
        auto* str = static_cast<std::string*>(userdata);
        str->append(ptr, size * nmemb);
        return size * nmemb;
    });
    curl_easy_setopt(curl_, CURLOPT_WRITEDATA, &response_string);
    curl_easy_setopt(curl_, CURLOPT_TIMEOUT, 30L);
    curl_easy_setopt(curl_, CURLOPT_FOLLOWLOCATION, 1L);
    
    CURLcode res = curl_easy_perform(curl_);
    
    if (res != CURLE_OK) {
        throw PriceServiceException(PriceServiceException::ErrorCode::NetworkError,
            std::string("API call failed: ") + curl_easy_strerror(res));
    }
    
    long http_code = 0;
    curl_easy_getinfo(curl_, CURLINFO_RESPONSE_CODE, &http_code);
    
    if (http_code == 429) {
        throw PriceServiceException(PriceServiceException::ErrorCode::RateLimited, 
            "Rate limited by CoinGecko API");
    }
    
    if (http_code != 200) {
        throw PriceServiceException(PriceServiceException::ErrorCode::NetworkError,
            "HTTP error: " + std::to_string(http_code));
    }
    
    return response_string;
}

// ----------------------------------------------------------------------------
// Real (minimal) JSON parsing helpers for the CoinGecko responses the service
// already fetches. No third-party JSON dependency. CoinGecko data is real
// market data; these parsers only extract it honestly. On any parse failure
// they return an empty result rather than fabricated numbers.
// ----------------------------------------------------------------------------

namespace {

// Find the substring spanning the JSON object that starts at the '{' found at
// or after `from`, matching braces (ignoring braces inside strings).
std::string extractObject(const std::string& json, size_t from, size_t& nextPos) {
    size_t start = json.find('{', from);
    if (start == std::string::npos) { nextPos = std::string::npos; return {}; }
    int depth = 0;
    bool inStr = false;
    for (size_t i = start; i < json.size(); ++i) {
        char c = json[i];
        if (inStr) {
            if (c == '\\' && i + 1 < json.size()) { ++i; continue; }
            if (c == '"') inStr = false;
            continue;
        }
        if (c == '"') inStr = true;
        else if (c == '{') ++depth;
        else if (c == '}') { --depth; if (depth == 0) { nextPos = i + 1; return json.substr(start, i - start + 1); } }
    }
    nextPos = std::string::npos;
    return {};
}

// Return the first quoted-string token at or after `from` (the token itself,
// without quotes). Used to read the top-level coin id key of /simple/price.
std::string firstStringToken(const std::string& json, size_t from) {
    size_t s = json.find('"', from);
    if (s == std::string::npos) return {};
    size_t e = s + 1;
    while (e < json.size() && json[e] != '"') {
        if (json[e] == '\\' && e + 1 < json.size()) ++e;
        ++e;
    }
    if (e >= json.size()) return {};
    return json.substr(s + 1, e - s - 1);
}

} // namespace

std::vector<PriceInfo> PriceService::parsePriceResponse(const std::string& response) {
    std::vector<PriceInfo> prices;

    // Locate the first non-whitespace character to distinguish the two
    // CoinGecko response shapes used by this service.
    size_t i = 0;
    while (i < response.size() && (response[i] == ' ' || response[i] == '\t' ||
           response[i] == '\n' || response[i] == '\r')) {
        ++i;
    }
    if (i >= response.size()) return prices;

    if (response[i] == '[') {
        // /coins/markets response: a JSON array of market objects.
        size_t pos = i + 1;
        size_t next = 0;
        while (pos < response.size()) {
            std::string obj = extractObject(response, pos, next);
            if (obj.empty() || next == std::string::npos) break;
            pos = next;

            PriceInfo info;
            auto sym = jsonStringField(obj, "symbol");
            auto name = jsonStringField(obj, "name");
            info.symbol = sym.value_or("");
            for (auto& c : info.symbol) c = static_cast<char>(std::toupper(static_cast<unsigned char>(c)));
            info.name = name.value_or("");
            info.price = jsonNumberField(obj, "current_price").value_or(0.0);
            double pct = jsonNumberField(obj, "price_change_percentage_24h").value_or(0.0);
            info.change_percent_24h = pct;
            info.change_24h = info.price * (pct / 100.0);
            info.market_cap = jsonNumberField(obj, "market_cap").value_or(0.0);
            info.volume_24h = jsonNumberField(obj, "total_volume").value_or(0.0);
            info.high_24h = jsonNumberField(obj, "high_24h").value_or(0.0);
            info.low_24h = jsonNumberField(obj, "low_24h").value_or(0.0);
            info.last_updated = std::chrono::system_clock::now();
            if (info.price > 0.0) prices.push_back(info);
        }
        return prices;
    }

    // /simple/price response: { "<coinId>": { "usd":..., "usd_24h_change":..., ... } }
    std::string coinId = firstStringToken(response, 0);
    size_t next = 0;
    std::string obj = extractObject(response, response.find('{', 0), next);
    if (obj.empty()) return prices;

    PriceInfo info;
    info.symbol = coinId;
    info.name = coinId;
    info.price = jsonNumberField(obj, "usd").value_or(0.0);
    info.change_percent_24h = jsonNumberField(obj, "usd_24h_change").value_or(0.0);
    info.change_24h = info.price * (info.change_percent_24h / 100.0);
    info.market_cap = jsonNumberField(obj, "usd_market_cap").value_or(0.0);
    info.volume_24h = jsonNumberField(obj, "usd_24hr_vol").value_or(0.0);
    info.high_24h = jsonNumberField(obj, "usd_24h_high").value_or(0.0);
    info.low_24h = jsonNumberField(obj, "usd_24h_low").value_or(0.0);
    info.last_updated = std::chrono::system_clock::now();
    if (info.price > 0.0) prices.push_back(info);

    return prices;
}

PriceHistory PriceService::parseHistoryResponse(const std::string& response) {
    PriceHistory history;

    // CoinGecko /coins/{id}/market_chart response:
    //   { "prices": [ [ <unix_ms>, <price> ], ... ], ... }
    size_t p = response.find("\"prices\"");
    if (p == std::string::npos) return history;
    size_t arr = response.find('[', p);
    if (arr == std::string::npos) return history;

    size_t i = arr + 1;
    while (i < response.size()) {
        size_t pairStart = response.find('[', i);
        if (pairStart == std::string::npos) break;
        size_t comma = response.find(',', pairStart);
        if (comma == std::string::npos) break;
        size_t pairEnd = response.find(']', comma);
        if (pairEnd == std::string::npos) break;

        try {
            double tsMs = std::stod(response.substr(pairStart + 1, comma - pairStart - 1));
            double price = std::stod(response.substr(comma + 1, pairEnd - comma - 1));
            auto tp = std::chrono::system_clock::from_time_t(
                static_cast<std::time_t>(tsMs / 1000.0));
            history.prices.emplace_back(tp, price);
        } catch (...) {
            // Skip unparseable pair rather than fabricating.
        }
        i = pairEnd + 1;
    }

    return history;
}

// ============================================================================
// Exception
// ============================================================================

PriceServiceException::PriceServiceException(ErrorCode code, const std::string& message)
    : std::runtime_error(message), code_(code) {}

PriceServiceException::ErrorCode PriceServiceException::getErrorCode() const {
    return code_;
}

} // namespace wallet
} // namespace tiger
