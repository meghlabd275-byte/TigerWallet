/**
 * TigerWallet Desktop - Staking Service
 * Staking operations for proof-of-stake blockchains
 */

#ifndef TIGER_WALLET_STAKING_SERVICE_H
#define TIGER_WALLET_STAKING_SERVICE_H

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
// Staking Service
// ============================================================================

class StakingService {
public:
    static std::shared_ptr<StakingService> getInstance();

    // Initialization
    void initialize();
    void shutdown();

    // Staking Quotes
    std::future<StakingQuote> getStakingQuote(
        const std::string& chainId,
        const std::string& token
    );

    std::future<std::vector<StakingQuote>> getStakingQuotes(const std::string& chainId);

    // Stake Operations
    std::future<StakeResponse> stake(
        const std::string& walletId,
        const std::string& token,
        const std::string& amount,
        const std::optional<std::string>& validator = std::nullopt
    );

    std::future<std::string> unstake(
        const std::string& walletId,
        const std::string& positionId,
        const std::string& amount
    );

    std::future<std::string> claimRewards(
        const std::string& walletId,
        const std::string& positionId
    );

    // Position Management
    std::future<std::vector<StakingPosition>> getStakingPositions(const std::string& walletId);
    std::future<StakingPosition> getStakingPosition(const std::string& walletId, const std::string& positionId);

    // Validators
    std::future<std::vector<std::string>> getValidators(const std::string& chainId);
    std::future<std::map<std::string, double>> getValidatorStats(const std::string& validatorAddress);

    // Rewards
    std::future<double> getPendingRewards(const std::string& walletId, const std::string& positionId);
    std::future<double> getTotalStaked(const std::string& walletId);
    std::future<double> getTotalRewards(const std::string& walletId);

private:
    StakingService(const StakingService&) = delete;
    StakingService& operator=(const StakingService&) = delete;

public:
    StakingService();
    ~StakingService();

    // API Calls
    std::string callStakingAPI(const std::string& endpoint, const std::string& body);
    std::string fetchFromAPI(const std::string& url);

    // Members
    static std::shared_ptr<StakingService> instance_;
    CURL* curl_;
    bool initialized_;
};

// ============================================================================
// Exception
// ============================================================================

class StakingServiceException : public std::runtime_error {
public:
    enum class ErrorCode {
        InsufficientFunds,
        ValidatorNotFound,
        PositionNotFound,
        LockPeriodActive,
        NetworkError,
        Unknown
    };

    StakingServiceException(ErrorCode code, const std::string& message);
    ErrorCode getErrorCode() const;

private:
    ErrorCode code_;
};

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_STAKING_SERVICE_H
