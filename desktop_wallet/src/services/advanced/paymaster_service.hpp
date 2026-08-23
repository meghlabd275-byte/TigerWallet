/**
 * Paymaster Service - C++ Desktop Implementation
 * 
 * Complete Paymaster Features:
 * - Gasless transactions
 * - Sponsored transactions
 * - Token paymaster
 * - Verifying paymaster
 * - User operation validation
 * 
 * This service MUST be identical across ALL platforms.
 */

#ifndef PAYMASTER_SERVICE_HPP
#define PAYMASTER_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>

namespace tigerwallet {

// Paymaster types
enum class PaymasterType {
    VERIFYING,      // Verifying paymaster - signature based
    TOKEN,         // Token paymaster - accepts ERC-20 tokens
    SPONSORED,     // Fully sponsored - free transactions
    HYBRID         // Hybrid - combination of above
};

// Paymaster sponsorship result
struct SponsorshipResult {
    bool success;
    std::string paymasterAddress;
    std::string preOpGas;
    std::string paymasterData;
    std::string signature;
    std::string errorMessage;
};

// User operation for gasless transactions
struct UserOperation {
    std::string sender;
    std::string nonce;
    std::string initCode;
    std::string callData;
    std::string callGasLimit;
    std::string verificationGasLimit;
    std::string preVerificationGas;
    std::string maxFeePerGas;
    std::string maxPriorityFeePerGas;
    std::string signature;
    std::string entryPoint;
};

// Token paymaster context
struct TokenPaymasterContext {
    std::string token;
    std::string exchangeRate;
    std::string decimals;
    std::string priceMarkup;
};

// Paymaster configuration
struct PaymasterConfig {
    std::string entryPointAddress;
    PaymasterType type;
    std::string paymasterAddress;
    std::string stakeAmount;
    uint32_t unstakeDelaySec;
    bool requiresSignature;
    std::vector<std::string> supportedTokens;
    std::map<std::string, TokenPaymasterContext> tokenContexts;
};

/**
 * Paymaster Service - Production Implementation
 */
class PaymasterService {
public:
    static PaymasterService& getInstance();
    
    // Initialize paymaster service
    bool initialize(const PaymasterConfig& config);
    
    // Check if paymaster is available for a user operation
    bool isOperationSponsored(const UserOperation& op);
    
    // Sponsor a user operation
    SponsorshipResult sponsorOperation(
        const UserOperation& op,
        const std::string& userAddress
    );
    
    // Validate user operation
    bool validateUserOperation(
        const UserOperation& op,
        const std::string& signature
    );
    
    // Get paymaster data for user operation
    std::string getPaymasterData(const UserOperation& op);
    
    // Calculate gas fees
    uint64_t calculateGasFees(const UserOperation& op);
    
    // Set paymaster type
    void setPaymasterType(PaymasterType type);
    
    // Get current paymaster type
    PaymasterType getPaymasterType() const;
    
    // Add supported token
    bool addSupportedToken(
        const std::string& tokenAddress,
        const TokenPaymasterContext& context
    );
    
    // Remove supported token
    bool removeSupportedToken(const std::string& tokenAddress);
    
    // Get supported tokens
    std::vector<std::string> getSupportedTokens() const;
    
    // Set entry point address
    void setEntryPoint(const std::string& entryPoint);
    
    // Get entry point address
    std::string getEntryPoint() const;
    
    // Check if service is initialized
    bool isInitialized() const;
    
private:
    PaymasterService();
    ~PaymasterService() = default;
    PaymasterService(const PaymasterService&) = delete;
    PaymasterService& operator=(const PaymasterService&) = delete;
    
    bool _initialized;
    PaymasterConfig _config;
    std::map<std::string, uint64_t> _userOperationCounts;
    
    // Internal methods
    bool validateSignature(
        const UserOperation& op,
        const std::string& signature
    );
    
    uint64_t calculatePreVerificationGas(const UserOperation& op);
    uint64_t calculateVerificationGas(const UserOperation& op);
    std::string generatePaymasterData(const UserOperation& op);
    std::string signUserOperation(const UserOperation& op);
};

} // namespace tigerwallet

#endif // PAYMASTER_SERVICE_HPP
