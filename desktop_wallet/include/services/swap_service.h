/**
 * TigerWallet Desktop - Swap Service
 * Token swapping via DEX aggregators
 */

#ifndef TIGER_WALLET_SWAP_SERVICE_H
#define TIGER_WALLET_SWAP_SERVICE_H

#include "models/wallet_models.h"
#include <memory>
#include <string>
#include <vector>
#include <future>
#include <optional>
#include <curl/curl.h>

namespace tiger {
namespace wallet {

// ============================================================================
// Swap Service
// ============================================================================

class SwapService {
public:
    static std::shared_ptr<SwapService> getInstance();

    // Initialization
    void initialize();
    void shutdown();

    // Swap Operations
    std::future<SwapQuote> getQuote(
        const std::string& fromToken,
        const std::string& toToken,
        const std::string& amount,
        const std::string& chainId
    );

    std::future<SwapResponse> executeSwap(
        const std::string& walletId,
        const SwapQuote& quote
    );

    // DEX Management
    std::vector<std::string> getSupportedDEXes(const std::string& chainId);
    void setPreferredDEX(const std::string& dex);

    // Approval
    std::future<std::string> approveToken(
        const std::string& walletId,
        const std::string& tokenAddress,
        const std::string& amount,
        const std::string& chainId
    );

    std::future<bool> isTokenApproved(
        const std::string& walletId,
        const std::string& tokenAddress,
        const std::string& chainId
    );

private:
    SwapService(const SwapService&) = delete;
    SwapService& operator=(const SwapService&) = delete;

public:
    SwapService();
    ~SwapService();

    // API Calls
    std::string callAggregatorAPI(const std::string& endpoint, const std::string& body);
    std::string fetchFromAPI(const std::string& url);

    // Members
    static std::shared_ptr<SwapService> instance_;
    CURL* curl_;
    bool initialized_;
    std::string preferredDEX_;
};

// ============================================================================
// Exception
// ============================================================================

class SwapServiceException : public std::runtime_error {
public:
    enum class ErrorCode {
        InsufficientLiquidity,
        PriceImpactTooHigh,
        SlippageExceeded,
        TokenNotSupported,
        NetworkError,
        Unknown
    };

    SwapServiceException(ErrorCode code, const std::string& message);
    ErrorCode getErrorCode() const;

private:
    ErrorCode code_;
};

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_SWAP_SERVICE_H
