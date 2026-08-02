/**
 * TigerWallet Desktop - Price Service
 * Real-time cryptocurrency price fetching from CoinGecko API
 */

#ifndef TIGER_WALLET_PRICE_SERVICE_H
#define TIGER_WALLET_PRICE_SERVICE_H

#include "models/wallet_models.h"
#include <memory>
#include <functional>
#include <string>
#include <map>
#include <vector>
#include <future>
#include <chrono>
#include <optional>
#include <curl/curl.h>

namespace tiger {
namespace wallet {

// ============================================================================
// Price Service
// ============================================================================

class PriceService {
public:
    static std::shared_ptr<PriceService> getInstance();

    // Initialization
    void initialize();
    void shutdown();

    // Price Fetching
    std::future<PriceInfo> getPrice(const std::string& symbol);
    std::future<std::map<std::string, PriceInfo>> getPrices(const std::vector<std::string>& symbols);
    std::future<std::vector<PriceInfo>> getTopPrices(int limit = 100);

    // Portfolio
    std::future<double> getPortfolioValue(const std::vector<Token>& tokens);

    // Historical Data
    std::future<PriceHistory> getPriceHistory(const std::string& symbol, int days);
    std::future<std::vector<PriceHistory>> getMultiPriceHistory(const std::vector<std::string>& symbols, int days);

    // Conversion
    std::future<double> convert(double amount, const std::string& from, const std::string& to);

    // Search
    std::future<std::vector<PriceInfo>> searchCoins(const std::string& query);

    // Price Updates
    using PriceUpdateCallback = std::function<void(const PriceInfo&)>;
    void startPriceUpdates(std::chrono::seconds interval = std::chrono::seconds(30));
    void stopPriceUpdates();
    void setPriceUpdateCallback(PriceUpdateCallback callback);

    // Cache Management
    void clearCache();
    std::optional<PriceInfo> getCachedPrice(const std::string& symbol);
    bool isCacheValid(const std::string& symbol) const;

private:
    PriceService();
    ~PriceService();
    PriceService(const PriceService&) = delete;
    PriceService& operator=(const PriceService&) = delete;

    // API Calls
    std::string fetchFromAPI(const std::string& url);
    std::vector<PriceInfo> parsePriceResponse(const std::string& response);
    PriceHistory parseHistoryResponse(const std::string& response);

    // Caching
    struct CacheEntry {
        PriceInfo price;
        std::chrono::system_clock::time_point timestamp;
    };
    
    std::map<std::string, CacheEntry> priceCache_;
    std::chrono::seconds cacheExpiry_;
    bool cacheEnabled_;
    
    // Members
    static std::shared_ptr<PriceService> instance_;
    CURL* curl_;
    bool initialized_;
    bool updatesRunning_;
    std::thread updateThread_;
    PriceUpdateCallback priceCallback_;
    
    // Supported coins
    std::vector<std::string> supportedCoins_;
};

// ============================================================================
// Exception
// ============================================================================

class PriceServiceException : public std::runtime_error {
public:
    enum class ErrorCode {
        NetworkError,
        InvalidResponse,
        RateLimited,
        NotFound,
        Unknown
    };

    PriceServiceException(ErrorCode code, const std::string& message);
    ErrorCode getErrorCode() const;

private:
    ErrorCode code_;
};

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_PRICE_SERVICE_H
