#ifndef MASTER_WALLET_PAYMASTER_SERVICE_HPP
#define MASTER_WALLET_PAYMASTER_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <chrono>
#include <mutex>
#include <atomic>
#include <optional>
#include <stdexcept>
#include <algorithm>
#include <thread>
#include <cstdint>

namespace tiger {
namespace master {
namespace paymaster {

// Forward declarations
class PaymasterService;
class GasPriceOracle;
class UserOperation;
class PaymasterConfig;

/**
 * UserOperation - ERC-4337 User Operation structure
 */
struct UserOperation {
    std::string sender;
    uint64_t nonce;
    std::string initCode;
    std::string callData;
    uint64_t callGasLimit;
    uint64_t verificationGasLimit;
    uint64_t preVerificationGas;
    uint64_t maxFeePerGas;
    uint64_t maxPriorityFeePerGas;
    std::string paymasterAndData;
    std::string signature;
    
    std::string hash() const;
    std::vector<uint8_t> encode() const;
    static UserOperation decode(const std::vector<uint8_t>& data);
};

/**
 * PaymasterConfig - Configuration for paymaster operations
 */
struct PaymasterConfig {
    std::string entryPointAddress;
    std::string paymasterAddress;
    std::string stakeAmount;
    uint32_t unstakeDelaySec;
    bool isVerifyingPaymaster;
    bool isTokenPaymaster;
    std::string supportedToken;
    uint64_t minBalance;
    uint64_t maxGasLimit;
    double markupPercent;
    bool enableBundlerIntegration;
    // Off-chain sponsor endpoint that returns a real ECDSA signature over the
    // EIP-191-prefixed userOpHash (the Pimlico/Stackup verifying-paymaster
    // pattern). When empty, buildPaymasterAndData fails closed rather than
    // emitting a placeholder signature.
    std::string sponsorEndpoint;
    
    PaymasterConfig();
};

/**
 * GasPriceOracle - Real-time gas price fetching
 */
class GasPriceOracle {
public:
    GasPriceOracle();
    ~GasPriceOracle();
    
    struct GasPrices {
        uint64_t baseFeePerGas;
        uint64_t maxFeePerGas;
        uint64_t maxPriorityFeePerGas;
        uint64_t suggestedMaxFeePerGas;
        uint64_t suggestedMaxPriorityFeePerGas;
        std::chrono::system_clock::time_point timestamp;
    };
    
    std::optional<GasPrices> getGasPrices(const std::string& chainId);
    void updateGasPrices(const std::string& chainId, const GasPrices& prices);
    void startPriceMonitoring(const std::string& chainId, uint32_t intervalMs);
    void stopPriceMonitoring();
    
private:
    std::map<std::string, GasPrices> gasCache_;
    std::mutex cacheMutex_;
    std::atomic<bool> monitoring_;
    std::thread monitorThread_;
    
    GasPrices fetchFromMultipleSources(const std::string& chainId);
    // Fetch real gas prices from the canonical backend GET /api/v1/gas.
    bool fetchFromBackend(const std::string& chainId, GasPrices& out);
    uint64_t calculateAverage(const std::vector<uint64_t>& values);
};

/**
 * SponsorshipPolicy - Defines conditions for gas sponsorship
 */
struct SponsorshipPolicy {
    std::string policyId;
    bool enabled;
    std::vector<std::string> allowedSenders;
    std::vector<std::string> blockedSenders;
    uint64_t minTransactionValue;
    uint64_t maxTransactionValue;
    uint64_t dailyBudget;
    uint64_t dailyUsed;
    std::chrono::system_clock::time_point dailyReset;
    std::map<std::string, uint64_t> tokenBudgets;
    bool requireWhitelist;
    bool enableRateLimiting;
    uint32_t maxTransactionsPerDay;
    uint32_t maxTransactionsPerHour;
    
    SponsorshipPolicy();
    
    bool isSenderAllowed(const std::string& sender) const;
    bool canSponsor(uint64_t value) const;
    void recordSponsorship(uint64_t value);
    void resetDailyBudget();
};

/**
 * PaymasterService - ERC-4337 Paymaster implementation
 */
class PaymasterService {
public:
    explicit PaymasterService(const PaymasterConfig& config);
    ~PaymasterService();
    
    // Core paymaster operations
    bool initialize();
    void shutdown();
    
    // UserOperation validation and execution
    std::string validateUserOperation(
        const UserOperation& userOp,
        const std::string& chainId
    );
    
    bool sponsorUserOperation(
        const UserOperation& userOp,
        const std::string& chainId,
        std::string& paymasterAndData
    );
    
    // Token paymaster operations
    bool isTokenPaymasterEnabled() const;
    std::string getPaymasterToken(const std::string& chainId);
    
    // Fee management
    uint64_t calculatePostOpGas(
        const UserOperation& userOp,
        uint64_t actualGasUsed
    );
    
    double calculateFee(
        const UserOperation& userOp,
        const std::string& token
    );
    
    // Sponsorship policies
    std::string createPolicy(const SponsorshipPolicy& policy);
    bool updatePolicy(const std::string& policyId, const SponsorshipPolicy& policy);
    bool deletePolicy(const std::string& policyId);
    std::optional<SponsorshipPolicy> getPolicy(const std::string& policyId);
    std::vector<SponsorshipPolicy> listPolicies() const;
    
    // Balance management
    uint64_t getPaymasterBalance(const std::string& chainId) const;
    bool fundPaymaster(const std::string& chainId, uint64_t amount);
    bool withdrawBalance(const std::string& chainId, uint64_t amount);
    
    // Stake management
    bool stake(uint64_t amount, uint32_t unstakeDelaySec);
    bool unstake();
    bool addStake(uint64_t amount);
    
    // Monitoring
    struct PaymasterStats {
        uint64_t totalSponsored;
        uint64_t totalSuccessful;
        uint64_t totalFailed;
        uint64_t totalGasUsed;
        double successRate;
        uint64_t averageGasPrice;
        std::map<std::string, uint64_t> byChain;
        std::chrono::system_clock::time_point lastUpdated;
    };
    
    PaymasterStats getStats() const;
    void resetStats();
    
    // Event callbacks
    using ValidationCallback = std::function<bool(const UserOperation&, const std::string&)>;
    using SponsorshipCallback = std::function<void(const UserOperation&, bool)>;
    
    void setValidationCallback(ValidationCallback callback);
    void setSponsorshipCallback(SponsorshipCallback callback);

private:
    PaymasterConfig config_;
    std::unique_ptr<GasPriceOracle> gasOracle_;
    std::map<std::string, SponsorshipPolicy> policies_;
    mutable std::mutex policiesMutex_;
    
    // Balance tracking
    std::map<std::string, uint64_t> chainBalances_;
    mutable std::mutex balanceMutex_;
    
    // Statistics
    std::atomic<uint64_t> totalSponsored_{0};
    std::atomic<uint64_t> totalSuccessful_{0};
    std::atomic<uint64_t> totalFailed_{0};
    std::atomic<uint64_t> totalGasUsed_{0};
    std::chrono::system_clock::time_point statsStartTime_;
    
    // Callbacks
    ValidationCallback validationCallback_;
    SponsorshipCallback sponsorshipCallback_;
    
    // Private methods
    bool validateSignature(
        const UserOperation& userOp,
        const std::string& chainId
    );
    
    bool checkSponsorshipPolicy(
        const UserOperation& userOp,
        const SponsorshipPolicy& policy
    );
    
    std::string buildPaymasterAndData(
        const UserOperation& userOp,
        const std::string& chainId
    );
    
    bool executeUserOperation(
        const UserOperation& userOp,
        const std::string& chainId
    );

    // Compute the Keccak-256 userOpHash over the packed UserOperation fields.
    std::string computeUserOpHash(const UserOperation& userOp, const std::string& chainId) const;
    // Request a real ECDSA signature from the off-chain sponsor endpoint.
    std::string requestSponsorSignature(const std::string& endpoint, const std::string& userOpHash) const;
    // Minimal HTTP POST (JSON) via libcurl.
    std::string httpPostJson(const std::string& url, const std::string& body) const;
    // Minimal flat-JSON field extractor.
    std::string extractJsonField(const std::string& json, const std::string& field) const;
    std::string toHex(const std::string& bytes) const;
    std::string fromHex(const std::string& hex) const;
    
    void recordSuccess(const UserOperation& userOp, uint64_t gasUsed);
    void recordFailure(const UserOperation& userOp, const std::string& error);
    
    uint64_t estimateGas(
        const UserOperation& userOp,
        const std::string& chainId
    );
    
    bool verifyPaymasterStake() const;
};

// Inline implementations

inline std::string UserOperation::hash() const {
    // Requires keccak256 over an ABI-encoded payload; not available client-side.
    throw std::runtime_error(
        "UserOperation hash computation requires keccak256 and is not "
        "available client-side");
}

inline std::vector<uint8_t> UserOperation::encode() const {
    // Requires ERC-4337 ABI encoding; not available client-side.
    throw std::runtime_error(
        "UserOperation ABI encoding is not available client-side");
}

inline UserOperation UserOperation::decode(const std::vector<uint8_t>& /*data*/) {
    // Requires ERC-4337 ABI decoding; not available client-side.
    throw std::runtime_error(
        "UserOperation ABI decoding is not available client-side");
}

inline PaymasterConfig::PaymasterConfig()
    : stakeAmount("100000000000000000")  // 0.1 ETH
    , unstakeDelaySec(0)
    , isVerifyingPaymaster(true)
    , isTokenPaymaster(false)
    , minBalance(0)
    , maxGasLimit(5000000)
    , markupPercent(10.0)
    , enableBundlerIntegration(true) {}

inline SponsorshipPolicy::SponsorshipPolicy()
    : enabled(true)
    , minTransactionValue(0)
    , maxTransactionValue(UINT64_MAX)
    , dailyBudget(UINT64_MAX)
    , dailyUsed(0)
    , requireWhitelist(false)
    , enableRateLimiting(true)
    , maxTransactionsPerDay(UINT32_MAX)
    , maxTransactionsPerHour(UINT32_MAX) {}

inline bool SponsorshipPolicy::isSenderAllowed(const std::string& sender) const {
    if (!enabled) return false;
    if (requireWhitelist && allowedSenders.empty()) return false;
    if (!allowedSenders.empty()) {
        return std::find(allowedSenders.begin(), allowedSenders.end(), sender) 
               != allowedSenders.end();
    }
    return std::find(blockedSenders.begin(), blockedSenders.end(), sender) 
           == blockedSenders.end();
}

inline bool SponsorshipPolicy::canSponsor(uint64_t value) const {
    if (value < minTransactionValue || value > maxTransactionValue) return false;
    if (dailyUsed >= dailyBudget) return false;
    return true;
}

inline void SponsorshipPolicy::recordSponsorship(uint64_t value) {
    dailyUsed += value;
}

inline void SponsorshipPolicy::resetDailyBudget() {
    auto now = std::chrono::system_clock::now();
    if (now >= dailyReset) {
        dailyUsed = 0;
        dailyReset = now + std::chrono::hours(24);
    }
}

} // namespace paymaster
} // namespace master
} // namespace tiger

#endif // MASTER_WALLET_PAYMASTER_SERVICE_HPP
