/**
 * TigerWallet MasterWallet - Account Abstraction Service (C++)
 * ERC-4337 Smart Wallet client.
 *
 * The canonical backend (CANONICAL_API_CONTRACT.md) does not expose ERC-4337
 * endpoints. ERC-4337 operations (CREATE2 address derivation, UserOperation
 * signing, bundler submission, EntryPoint simulation, factory deployment)
 * require on-chain interaction and the wallet's private key, which is held by
 * the backend. Therefore every operation that would need real crypto or
 * broadcast THROWS a fail-closed error rather than fabricating addresses,
 * signatures, or transaction hashes.
 *
 * Operations that are purely local configuration (entry-point/paymaster
 * addresses, session-key policy, social-recovery guardian sets, batch
 * metadata) are stored honestly client-side; they hold no secrets and make no
 * on-chain claims.
 */

#include "account_abstraction_service.hpp"
#include <algorithm>
#include <chrono>
#include <ctime>
#include <sstream>

namespace tiger {
namespace master {
namespace account_abstraction {

// Constants
constexpr const char* DEFAULT_ENTRY_POINT = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3";
constexpr uint64_t DEFAULT_CALL_GAS_LIMIT = 100000;
constexpr uint64_t DEFAULT_VERIFICATION_GAS_LIMIT = 150000;
constexpr uint64_t DEFAULT_PRE_VERIFICATION_GAS = 21000;

AccountAbstractionService::AccountAbstractionService(const std::string& masterWalletId)
    : masterWalletId_(masterWalletId) {
    // Ethereum mainnet EntryPoint address (public, well-known constant, not
    // fabricated data).
    entryPoints_["1"] = DEFAULT_ENTRY_POINT;
}

AccountAbstractionService::~AccountAbstractionService() { shutdown(); }

bool AccountAbstractionService::initialize() { return true; }

void AccountAbstractionService::shutdown() {
    std::lock_guard<std::mutex> lock(dataMutex_);
    smartWallets_.clear();
    batches_.clear();
}

// ==================== Smart wallet management ====================

std::string AccountAbstractionService::createSmartWallet(
    const std::string& owner,
    const std::string& entryPoint
) {
    if (owner.empty()) return "";

    // The contract wallet address is derived via CREATE2 from the factory,
    // salt, and init code on-chain. The backend does not expose an AA endpoint,
    // so the address cannot be computed client-side. Fail closed.
    throw std::runtime_error(
        "Smart wallet creation requires ERC-4337 bundler/factory interaction, "
        "which is not exposed by the canonical backend");
    (void)entryPoint;
}

bool AccountAbstractionService::initializeSmartWallet(
    const std::string& /*walletAddress*/,
    const std::string& /*initData*/
) {
    // Initialising a smart wallet is an on-chain transaction; not available
    // client-side.
    throw std::runtime_error(
        "Smart wallet initialization is an on-chain operation not exposed by "
        "the canonical backend");
}

std::optional<SmartWallet> AccountAbstractionService::getSmartWallet(const std::string& owner) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = smartWallets_.find(owner);
    if (it != smartWallets_.end()) return it->second;
    return std::nullopt;
}

std::vector<SmartWallet> AccountAbstractionService::listSmartWallets() {
    std::lock_guard<std::mutex> lock(dataMutex_);
    std::vector<SmartWallet> result;
    for (const auto& pair : smartWallets_) result.push_back(pair.second);
    return result;
}

// ==================== User operations ====================

std::string AccountAbstractionService::sendUserOperation(
    const UserOperation& /*userOp*/,
    const std::string& /*chainId*/
) {
    // Submitting a UserOperation requires signing with the wallet's private
    // key (held by the backend) and submitting to a bundler. Not available
    // client-side; fail closed instead of returning a fake hash.
    throw std::runtime_error(
        "Sending an ERC-4337 UserOperation requires backend/bundler support, "
        "which is not exposed by the canonical backend");
}

std::vector<std::string> AccountAbstractionService::sendBatchOperations(
    const std::vector<UserOperation>& /*operations*/,
    const std::string& /*chainId*/
) {
    throw std::runtime_error(
        "Batched ERC-4337 UserOperations require backend/bundler support, "
        "which is not exposed by the canonical backend");
}

bool AccountAbstractionService::simulateValidation(
    const UserOperation& /*userOp*/,
    const std::string& /*chainId*/,
    std::string& validationResult
) {
    // simulateValidation is an EntryPoint pre-flight call against a chain node.
    // Not available client-side; fail closed.
    validationResult = "simulateValidation not available client-side";
    return false;
}

// ==================== Entry point management (local config) ====================

std::string AccountAbstractionService::getEntryPoint(const std::string& chainId) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = entryPoints_.find(chainId);
    if (it != entryPoints_.end()) return it->second;
    return DEFAULT_ENTRY_POINT;
}

bool AccountAbstractionService::addEntryPoint(
    const std::string& chainId,
    const std::string& entryPointAddress
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    entryPoints_[chainId] = entryPointAddress;
    return true;
}

// ==================== Paymaster integration (local config) ====================

bool AccountAbstractionService::setPaymaster(
    const std::string& chainId,
    const std::string& paymasterAddress
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    paymasters_[chainId] = paymasterAddress;
    return true;
}

std::string AccountAbstractionService::getPaymaster(const std::string& chainId) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = paymasters_.find(chainId);
    return it != paymasters_.end() ? it->second : std::string();
}

// ==================== Session keys (local policy) ====================

std::string AccountAbstractionService::addSessionKey(
    const std::string& walletAddress,
    const SessionKey& sessionKey
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    sessionKeys_[walletAddress].push_back(sessionKey);
    return sessionKey.key;
}

bool AccountAbstractionService::removeSessionKey(
    const std::string& walletAddress,
    const std::string& key
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = sessionKeys_.find(walletAddress);
    if (it == sessionKeys_.end()) return false;
    auto& keys = it->second;
    auto keyIt = std::remove_if(keys.begin(), keys.end(),
        [&key](const SessionKey& sk) { return sk.key == key; });
    bool removed = keyIt != keys.end();
    keys.erase(keyIt, keys.end());
    return removed;
}

std::vector<SessionKey> AccountAbstractionService::getSessionKeys(const std::string& walletAddress) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = sessionKeys_.find(walletAddress);
    if (it != sessionKeys_.end()) return it->second;
    return {};
}

bool AccountAbstractionService::isSessionKeyValid(
    const std::string& walletAddress,
    const std::string& key,
    const std::string& contract,
    const std::string& token,
    uint64_t amount
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = sessionKeys_.find(walletAddress);
    if (it == sessionKeys_.end()) return false;
    for (const auto& sk : it->second) {
        if (sk.key != key || !sk.isActive) continue;
        if (std::chrono::system_clock::now() > sk.expiresAt) return false;
        if (sk.spentAmount + amount > sk.spendingLimit) return false;
        if (!sk.allowedContracts.empty() &&
            std::find(sk.allowedContracts.begin(), sk.allowedContracts.end(), contract) ==
                sk.allowedContracts.end()) return false;
        if (!sk.allowedTokens.empty() &&
            std::find(sk.allowedTokens.begin(), sk.allowedTokens.end(), token) ==
                sk.allowedTokens.end()) return false;
        return true;
    }
    return false;
}

// ==================== Social recovery (local config) ====================

bool AccountAbstractionService::setupSocialRecovery(
    const std::string& walletAddress,
    const std::vector<Guardian>& guardians,
    uint8_t threshold
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    SocialRecoveryConfig config;
    config.guardians = guardians;
    config.threshold = threshold;
    config.guardianCount = static_cast<uint8_t>(guardians.size());
    config.isSetup = true;
    socialRecoveryConfigs_[walletAddress] = config;
    return true;
}

bool AccountAbstractionService::addGuardian(
    const std::string& walletAddress,
    const Guardian& guardian
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = socialRecoveryConfigs_.find(walletAddress);
    if (it == socialRecoveryConfigs_.end()) return false;
    it->second.guardians.push_back(guardian);
    it->second.guardianCount = static_cast<uint8_t>(it->second.guardians.size());
    return true;
}

bool AccountAbstractionService::removeGuardian(
    const std::string& walletAddress,
    const std::string& guardianAddress
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = socialRecoveryConfigs_.find(walletAddress);
    if (it == socialRecoveryConfigs_.end()) return false;
    auto& guardians = it->second.guardians;
    auto gIt = std::remove_if(guardians.begin(), guardians.end(),
        [&guardianAddress](const Guardian& g) { return g.address == guardianAddress; });
    guardians.erase(gIt, guardians.end());
    it->second.guardianCount = static_cast<uint8_t>(guardians.size());
    return true;
}

bool AccountAbstractionService::confirmGuardian(
    const std::string& walletAddress,
    const std::string& guardianAddress
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = socialRecoveryConfigs_.find(walletAddress);
    if (it == socialRecoveryConfigs_.end()) return false;
    for (auto& guardian : it->second.guardians) {
        if (guardian.address == guardianAddress) { guardian.confirmed = true; return true; }
    }
    return false;
}

bool AccountAbstractionService::initiateRecovery(
    const std::string& walletAddress,
    const std::string& /*newOwner*/
) {
    // Recovering ownership is an on-chain operation requiring guardian
    // signatures and the wallet contract. Fail closed.
    throw std::runtime_error(
        "Social recovery initiation is an on-chain operation not exposed by "
        "the canonical backend");
}

bool AccountAbstractionService::completeRecovery(
    const std::string& /*walletAddress*/,
    const std::string& /*newOwner*/,
    const std::vector<std::string>& /*guardianSignatures*/
) {
    throw std::runtime_error(
        "Social recovery completion requires on-chain guardian signature "
        "verification not available client-side");
}

std::optional<SocialRecoveryConfig> AccountAbstractionService::getSocialRecoveryConfig(
    const std::string& walletAddress
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = socialRecoveryConfigs_.find(walletAddress);
    if (it != socialRecoveryConfigs_.end()) return it->second;
    return std::nullopt;
}

// ==================== Batched operations (local metadata) ====================

std::string AccountAbstractionService::createBatch(const std::vector<UserOperation>& operations) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    std::string batchId = "batch_" + std::to_string(static_cast<uint64_t>(std::time(nullptr)));
    BatchedUserOperation batch;
    batch.batchId = batchId;
    batch.operations = operations;
    batch.status = "pending";
    batch.totalGas = 0;
    batch.createdAt = std::chrono::system_clock::now();
    batches_[batchId] = batch;
    return batchId;
}

bool AccountAbstractionService::addToBatch(
    const std::string& batchId,
    const UserOperation& operation
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = batches_.find(batchId);
    if (it == batches_.end() || it->second.status != "pending") return false;
    it->second.operations.push_back(operation);
    return true;
}

bool AccountAbstractionService::executeBatch(
    const std::string& /*batchId*/,
    const std::string& /*chainId*/
) {
    // Executing a batch submits UserOperations to a bundler; not available
    // client-side. Fail closed.
    throw std::runtime_error(
        "Batch execution requires ERC-4337 bundler support, which is not "
        "exposed by the canonical backend");
}

std::optional<BatchedUserOperation> AccountAbstractionService::getBatch(const std::string& batchId) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = batches_.find(batchId);
    if (it != batches_.end()) return it->second;
    return std::nullopt;
}

std::vector<BatchedUserOperation> AccountAbstractionService::getPendingBatches() {
    std::lock_guard<std::mutex> lock(dataMutex_);
    std::vector<BatchedUserOperation> result;
    for (const auto& pair : batches_) if (pair.second.status == "pending") result.push_back(pair.second);
    return result;
}

// ==================== Factory ====================

std::string AccountAbstractionService::getFactoryAddress(const std::string& chainId) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auto it = factories_.find(chainId);
    return it != factories_.end() ? it->second : std::string();
}

bool AccountAbstractionService::deployFactory(const std::string& /*chainId*/) {
    // Deploying a factory is an on-chain deployment; not available client-side.
    throw std::runtime_error(
        "Factory deployment is an on-chain operation not exposed by the "
        "canonical backend");
}

// ==================== Statistics ====================

AccountAbstractionService::AAStats AccountAbstractionService::getStats() const {
    AAStats stats{};
    {
        std::lock_guard<std::mutex> lock(dataMutex_);
        stats.totalSmartWallets = smartWallets_.size();
        for (const auto& pair : batches_) if (pair.second.status == "pending") stats.pendingBatches++;
    }
    stats.totalUserOperations = totalUserOperations_.load();
    stats.successfulOperations = successfulOperations_.load();
    stats.failedOperations = failedOperations_.load();
    stats.totalGasUsed = totalGasUsed_.load();
    if (stats.totalUserOperations > 0)
        stats.successRate = static_cast<double>(stats.successfulOperations) /
                           static_cast<double>(stats.totalUserOperations) * 100.0;
    return stats;
}

void AccountAbstractionService::resetStats() {
    totalUserOperations_ = 0;
    successfulOperations_ = 0;
    failedOperations_ = 0;
    totalGasUsed_ = 0;
}

// ==================== Private methods ====================

std::string AccountAbstractionService::generateWalletAddress(
    const std::string& /*owner*/,
    const std::string& /*salt*/
) {
    // CREATE2 derivation needs keccak256 and the factory/init-code bytes, all
    // held on-chain. Not available client-side.
    throw std::runtime_error(
        "CREATE2 smart wallet address derivation is not available client-side");
}

std::string AccountAbstractionService::encodeInitializeCall(
    const std::string& /*owner*/,
    const std::string& /*entryPoint*/
) {
    // ABI-encoding the initializer requires the wallet's ABI, which the client
    // does not have. Fail closed.
    throw std::runtime_error(
        "Initializer ABI encoding is not available client-side");
}

bool AccountAbstractionService::validateUserOperation(
    const UserOperation& userOp,
    const std::string& /*chainId*/
) {
    // Only structural validation; no on-chain claims.
    return !userOp.sender.empty() && userOp.callGasLimit > 0;
}

uint64_t AccountAbstractionService::estimateUserOperationGas(
    const UserOperation& userOp,
    const std::string& /*chainId*/
) {
    // This is a local estimate only, clearly derived from the supplied limits;
    // it makes no claim about the actual gas a bundler will charge.
    uint64_t baseGas = 21000;
    uint64_t verificationGas = userOp.verificationGasLimit > 0 ?
        userOp.verificationGasLimit : DEFAULT_VERIFICATION_GAS_LIMIT;
    uint64_t callGas = userOp.callGasLimit > 0 ? userOp.callGasLimit : DEFAULT_CALL_GAS_LIMIT;
    uint64_t preVerificationGas = userOp.preVerificationGas > 0 ?
        userOp.preVerificationGas : DEFAULT_PRE_VERIFICATION_GAS;
    return baseGas + verificationGas + callGas + preVerificationGas;
}

std::string AccountAbstractionService::signUserOperation(
    const UserOperation& /*userOp*/,
    const std::string& /*privateKey*/
) {
    // Signing requires the wallet's private key (held by the backend) and
    // keccak256. Fail closed rather than fabricate a signature.
    throw std::runtime_error(
        "UserOperation signing is not available client-side; it must be "
        "performed by the backend");
}

std::string AccountAbstractionService::executeUserOperation(
    const UserOperation& /*userOp*/,
    const std::string& /*chainId*/
) {
    // Execution submits to a bundler. Fail closed instead of a fake hash.
    throw std::runtime_error(
        "UserOperation execution requires bundler support, which is not "
        "exposed by the canonical backend");
}

// UserOperation::hash requires keccak256 over an ABI-encoded payload, which is
// not available client-side. Fail closed rather than fabricate a digest.
std::string UserOperation::hash(const std::string& /*entryPoint*/, uint64_t /*chainId*/) const {
    throw std::runtime_error(
        "UserOperation hash computation requires keccak256 and is not "
        "available client-side");
}

} // namespace account_abstraction
} // namespace master
} // namespace tiger
