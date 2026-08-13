#ifndef MASTER_WALLET_ACCOUNT_ABSTRACTION_SERVICE_HPP
#define MASTER_WALLET_ACCOUNT_ABSTRACTION_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <optional>
#include <chrono>
#include <mutex>
#include <atomic>

namespace tiger {
namespace master {
namespace account_abstraction {

// Forward declarations
class AccountAbstractionService;
class SmartWalletFactory;
class EntryPoint;

/**
 * SmartWallet - ERC-4337 Smart Contract Wallet
 */
struct SmartWallet {
    std::string address;
    std::string owner;
    std::string entryPoint;
    uint256_t nonce;
    std::string implementation;
    bool initialized;
    std::vector<std::string> guardians;
    uint64_t createdAt;
};

/**
 * UserOperation - ERC-4337 User Operation
 */
struct UserOperation {
    std::string sender;
    uint256_t nonce;
    std::string initCode;
    std::string callData;
    uint64_t callGasLimit;
    uint64_t verificationGasLimit;
    uint64_t preVerificationGas;
    uint64_t maxFeePerGas;
    uint64_t maxPriorityFeePerGas;
    std::string paymasterAndData;
    std::string signature;
    
    std::string hash(const std::string& entryPoint, uint64_t chainId) const;
};

/**
 * SessionKey - Session key for automatic transactions
 */
struct SessionKey {
    std::string key;
    std::string permission;
    std::vector<std::string> allowedContracts;
    std::vector<std::string> allowedTokens;
    uint64_t spendingLimit;
    uint64_t spentAmount;
    std::chrono::system_clock::time_point expiresAt;
    bool isActive;
};

/**
 * Guardian - Social recovery guardian
 */
struct Guardian {
    std::string address;
    std::string name;
    uint8_t threshold;
    std::chrono::system_clock::time_point addedAt;
    bool confirmed;
};

/**
 * SocialRecoveryConfig - Social recovery configuration
 */
struct SocialRecoveryConfig {
    std::vector<Guardian> guardians;
    uint8_t threshold;
    uint8_t guardianCount;
    bool isSetup;
    std::chrono::system_clock::time_point lastRecoveryAttempt;
};

/**
 * BatchedUserOperation - Batched operations
 */
struct BatchedUserOperation {
    std::string batchId;
    std::vector<UserOperation> operations;
    std::string status;  // pending, executing, completed, failed
    uint64_t totalGas;
    std::chrono::system_clock::time_point createdAt;
    std::chrono::system_clock::time_point executedAt;
};

/**
 * AccountAbstractionService - ERC-4337 Account Abstraction
 */
class AccountAbstractionService {
public:
    AccountAbstractionService(const std::string& masterWalletId);
    ~AccountAbstractionService();
    
    // Service lifecycle
    bool initialize();
    void shutdown();
    
    // Smart wallet management
    std::string createSmartWallet(
        const std::string& owner,
        const std::string& entryPoint = ""
    );
    
    bool initializeSmartWallet(
        const std::string& walletAddress,
        const std::string& initData
    );
    
    std::optional<SmartWallet> getSmartWallet(const std::string& owner);
    std::vector<SmartWallet> listSmartWallets();
    
    // User operations
    std::string sendUserOperation(
        const UserOperation& userOp,
        const std::string& chainId
    );
    
    std::vector<std::string> sendBatchOperations(
        const std::vector<UserOperation>& operations,
        const std::string& chainId
    );
    
    bool simulateValidation(
        const UserOperation& userOp,
        const std::string& chainId,
        std::string& validationResult
    );
    
    // Entry point management
    std::string getEntryPoint(const std::string& chainId);
    bool addEntryPoint(
        const std::string& chainId,
        const std::string& entryPointAddress
    );
    
    // Paymaster integration
    bool setPaymaster(
        const std::string& chainId,
        const std::string& paymasterAddress
    );
    
    std::string getPaymaster(const std::string& chainId);

    // Bundler endpoint (ERC-4337 eth_sendUserOperation). Required for any
    // UserOperation execution; without it executeUserOperation fails closed.
    bool setBundlerUrl(const std::string& chainId, const std::string& bundlerUrl);
    std::string getBundlerUrl(const std::string& chainId);

    // Session keys
    std::string addSessionKey(
        const std::string& walletAddress,
        const SessionKey& sessionKey
    );
    
    bool removeSessionKey(
        const std::string& walletAddress,
        const std::string& key
    );
    
    std::vector<SessionKey> getSessionKeys(const std::string& walletAddress);
    
    bool isSessionKeyValid(
        const std::string& walletAddress,
        const std::string& key,
        const std::string& contract,
        const std::string& token,
        uint64_t amount
    );
    
    // Social recovery
    bool setupSocialRecovery(
        const std::string& walletAddress,
        const std::vector<Guardian>& guardians,
        uint8_t threshold
    );
    
    bool addGuardian(
        const std::string& walletAddress,
        const Guardian& guardian
    );
    
    bool removeGuardian(
        const std::string& walletAddress,
        const std::string& guardianAddress
    );
    
    bool confirmGuardian(
        const std::string& walletAddress,
        const std::string& guardianAddress
    );
    
    bool initiateRecovery(
        const std::string& walletAddress,
        const std::string& newOwner
    );
    
    bool completeRecovery(
        const std::string& walletAddress,
        const std::string& newOwner,
        const std::vector<std::string>& guardianSignatures
    );
    
    std::optional<SocialRecoveryConfig> getSocialRecoveryConfig(
        const std::string& walletAddress
    );
    
    // Batched operations
    std::string createBatch(
        const std::vector<UserOperation>& operations
    );
    
    bool addToBatch(
        const std::string& batchId,
        const UserOperation& operation
    );
    
    bool executeBatch(
        const std::string& batchId,
        const std::string& chainId
    );
    
    std::optional<BatchedUserOperation> getBatch(const std::string& batchId);
    std::vector<BatchedUserOperation> getPendingBatches();
    
    // Factory
    std::string getFactoryAddress(const std::string& chainId);
    // Set the on-chain factory address for `chainId` (must be a real deployed
    // ERC-4337 account factory address, not invented here).
    bool setFactoryAddress(const std::string& chainId, const std::string& factoryAddress);
    bool deployFactory(const std::string& chainId);
    
    // Statistics
    struct AAStats {
        uint64_t totalSmartWallets;
        uint64_t totalUserOperations;
        uint64_t successfulOperations;
        uint64_t failedOperations;
        uint64_t totalGasUsed;
        double successRate;
        uint64_t pendingBatches;
    };
    
    AAStats getStats() const;
    void resetStats();

private:
    std::string masterWalletId_;
    std::map<std::string, SmartWallet> smartWallets_;
    std::map<std::string, std::string> entryPoints_;  // chainId -> entryPoint
    std::map<std::string, std::string> paymasters_;    // chainId -> paymaster
    std::map<std::string, std::vector<SessionKey>> sessionKeys_;
    std::map<std::string, SocialRecoveryConfig> socialRecoveryConfigs_;
    std::map<std::string, BatchedUserOperation> batches_;
    std::map<std::string, std::string> factories_;    // chainId -> factory
    // chainId -> bundler RPC endpoint (ERC-4337 eth_sendUserOperation). A
    // UserOperation can only be executed when a real bundler is configured;
    // without one executeUserOperation fails closed (returns "").
    std::map<std::string, std::string> bundlerUrls_;
    
    mutable std::mutex dataMutex_;
    
    std::atomic<uint64_t> totalUserOperations_{0};
    std::atomic<uint64_t> successfulOperations_{0};
    std::atomic<uint64_t> failedOperations_{0};
    std::atomic<uint64_t> totalGasUsed_{0};
    
    // Private methods
    std::string generateWalletAddress(
        const std::string& owner,
        const std::string& salt
    );
    
    std::string encodeInitializeCall(
        const std::string& owner,
        const std::string& entryPoint
    );
    
    bool validateUserOperation(
        const UserOperation& userOp,
        const std::string& chainId
    );
    
    uint64_t estimateUserOperationGas(
        const UserOperation& userOp,
        const std::string& chainId
    );
    
    std::string signUserOperation(
        const UserOperation& userOp,
        const std::string& privateKey
    );
    
    std::string executeUserOperation(
        const UserOperation& userOp,
        const std::string& chainId
    );
};

} // namespace account_abstraction
} // namespace master
} // namespace tiger

#endif // MASTER_WALLET_ACCOUNT_ABSTRACTION_SERVICE_HPP
