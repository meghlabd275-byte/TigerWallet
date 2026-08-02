/**
 * Account Abstraction Service Implementation - C++ Desktop
 */

#include "account_abstraction_service.hpp"
#include <chrono>
#include <random>
#include <sstream>
#include <iomanip>

namespace tigerwallet {

AccountAbstractionService& AccountAbstractionService::getInstance() {
    static AccountAbstractionService instance;
    return instance;
}

AccountAbstractionService::AccountAbstractionService() : isInitialized_(false) {}

SmartAccount AccountAbstractionService::initialize(const std::string& ownerAddress) {
    SmartAccount account;
    account.address = deriveSmartAccountAddress(ownerAddress);
    account.owner = ownerAddress;
    account.nonce = 0;
    account.isDeployed = false;
    account.entryPoint = ENTRY_POINT_ADDRESS;
    smartAccount_ = std::make_unique<SmartAccount>(account);
    isInitialized_ = true;
    return account;
}

std::string AccountAbstractionService::getAccountAddress() {
    return smartAccount_ ? smartAccount_->address : "";
}

std::string AccountAbstractionService::sendUserOp(const std::string& to, const std::string& value,
                                                    const std::vector<uint8_t>& data, bool paymaster) {
    auto userOp = createUserOperation(to, value, data, paymaster);
    std::string hashResult = hashUserOperation(userOp);
    return "0x" + hashResult + std::to_string(std::chrono::system_clock::now().time_since_epoch().count());
}

SessionKey AccountAbstractionService::createSessionKey(const std::string& dAppAddress, int64_t validUntil,
                                                       const std::vector<std::string>& allowedContracts,
                                                       const std::vector<std::string>& allowedSelectors,
                                                       const std::string& spendingLimit) {
    SessionKey key;
    key.keyAddress = generateKeyAddress();
    key.dAppAddress = dAppAddress;
    key.validUntil = validUntil;
    key.allowedContracts = allowedContracts;
    key.allowedSelectors = allowedSelectors;
    key.spendingLimit = spendingLimit;
    key.spentAmount = "0";
    key.isRevoked = false;
    sessionKeys_[key.keyAddress] = key;
    return key;
}

bool AccountAbstractionService::revokeSessionKey(const std::string& keyAddress) {
    auto it = sessionKeys_.find(keyAddress);
    if (it != sessionKeys_.end()) {
        it->second.isRevoked = true;
        return true;
    }
    return false;
}

std::vector<SessionKey> AccountAbstractionService::getActiveSessionKeys() {
    int64_t now = std::chrono::system_clock::now().time_since_epoch().count();
    std::vector<SessionKey> active;
    for (const auto& pair : sessionKeys_) {
        if (!pair.second.isRevoked && pair.second.validUntil > now) {
            active.push_back(pair.second);
        }
    }
    return active;
}

std::string AccountAbstractionService::executeWithSessionKey(const std::string& keyAddress, const std::string& to,
                                                             const std::vector<uint8_t>& data) {
    auto it = sessionKeys_.find(keyAddress);
    if (it == sessionKeys_.end()) throw std::runtime_error("Session key not found");
    if (it->second.isRevoked) throw std::runtime_error("Session key revoked");
    
    int64_t now = std::chrono::system_clock::now().time_since_epoch().count();
    if (now > it->second.validUntil) throw std::runtime_error("Session key expired");
    
    std::string dataStr(data.begin(), data.end());
    return "0x" + hash(to + dataStr);
}

std::string AccountAbstractionService::deriveSmartAccountAddress(const std::string& owner) {
    std::string hashResult = hash(owner + "_smart_account");
    return "0x" + hashResult.substr(0, 40);
}

std::string AccountAbstractionService::generateKeyAddress() {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 255);
    std::string bytes;
    for (int i = 0; i < 32; ++i) {
        bytes += static_cast<char>(dis(gen));
    }
    return "0x" + hash(bytes).substr(0, 40);
}

UserOperation AccountAbstractionService::createUserOperation(const std::string& to, const std::string& value,
                                                            const std::vector<uint8_t>& data, bool paymaster) {
    UserOperation op;
    op.sender = smartAccount_ ? smartAccount_->address : "";
    op.nonce = smartAccount_ ? std::to_string(smartAccount_->nonce) : "0";
    op.initCode = (smartAccount_ && !smartAccount_->isDeployed) ? "0x" : "0x";
    op.callData = encodeCallData(to, value, data);
    op.callGasLimit = "0x5208";
    op.verificationGasLimit = "0x186A0";
    op.preVerificationGas = "0x5208";
    op.maxFeePerGas = "0x3B9ACA00";
    op.maxPriorityFeePerGas = "0x3B9ACA00";
    op.paymasterAndData = paymaster ? "0xPaymasterAddress" : "0x";
    op.signature = "0x";
    return op;
}

std::string AccountAbstractionService::encodeCallData(const std::string& to, const std::string& value,
                                                       const std::vector<uint8_t>& data) {
    std::string toClean = to.substr(0, 2) == "0x" ? to.substr(2) : to;
    std::string result = "0x" + toClean;
    result += std::string(64 - value.length(), '0') + value;
    result += std::string(64, '0'); // data length placeholder
    return result;
}

std::string AccountAbstractionService::hashUserOperation(const UserOperation& userOp) {
    return hash(userOp.sender + userOp.nonce + userOp.callData);
}

std::string AccountAbstractionService::hash(const std::string& input) {
    uint64_t h = 0;
    for (char c : input) {
        h = h * 31 + static_cast<uint8_t>(c);
    }
    std::stringstream ss;
    ss << std::hex << std::setfill('0') << std::setw(16) << h;
    return ss.str();
}

} // namespace tigerwallet
