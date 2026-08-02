/**
 * Account Abstraction Service - C++ Desktop
 * ERC-4337 Implementation
 */

#ifndef ACCOUNT_ABSTRACTION_SERVICE_HPP
#define ACCOUNT_ABSTRACTION_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>

namespace tigerwallet {

const std::string ENTRY_POINT_ADDRESS = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3";

struct SmartAccount {
    std::string address;
    std::string owner;
    int nonce;
    bool isDeployed;
    std::string entryPoint;
};

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
    std::string paymasterAndData;
    std::string signature;
};

struct SessionKey {
    std::string keyAddress;
    std::string dAppAddress;
    int64_t validUntil;
    std::vector<std::string> allowedContracts;
    std::vector<std::string> allowedSelectors;
    std::string spendingLimit;
    std::string spentAmount;
    bool isRevoked;
};

class AccountAbstractionService {
public:
    static AccountAbstractionService& getInstance();
    
    SmartAccount initialize(const std::string& ownerAddress);
    std::string getAccountAddress();
    
    std::string sendUserOp(const std::string& to, const std::string& value,
                          const std::vector<uint8_t>& data, bool paymaster = true);
    
    SessionKey createSessionKey(const std::string& dAppAddress, int64_t validUntil,
                                const std::vector<std::string>& allowedContracts,
                                const std::vector<std::string>& allowedSelectors,
                                const std::string& spendingLimit);
    bool revokeSessionKey(const std::string& keyAddress);
    std::vector<SessionKey> getActiveSessionKeys();
    std::string executeWithSessionKey(const std::string& keyAddress, const std::string& to,
                                       const std::vector<uint8_t>& data);

private:
    AccountAbstractionService();
    ~AccountAbstractionService() = default;
    AccountAbstractionService(const AccountAbstractionService&) = delete;
    AccountAbstractionService& operator=(const AccountAbstractionService&) = delete;
    
    std::unique_ptr<SmartAccount> smartAccount_;
    std::map<std::string, SessionKey> sessionKeys_;
    bool isInitialized_;
    
    std::string deriveSmartAccountAddress(const std::string& owner);
    std::string generateKeyAddress();
    UserOperation createUserOperation(const std::string& to, const std::string& value,
                                      const std::vector<uint8_t>& data, bool paymaster);
    std::string encodeCallData(const std::string& to, const std::string& value,
                               const std::vector<uint8_t>& data);
    std::string hashUserOperation(const UserOperation& userOp);
    std::string hash(const std::string& input);
};

} // namespace tigerwallet

#endif // ACCOUNT_ABSTRACTION_SERVICE_HPP
