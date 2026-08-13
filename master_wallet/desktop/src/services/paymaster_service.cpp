/**
 * TigerWallet MasterWallet - Paymaster Service (C++)
 * ERC-4337 Paymaster client.
 *
 * The canonical backend (CANONICAL_API_CONTRACT.md) does not expose ERC-4337
 * paymaster endpoints. Real paymaster operations (signature generation for
 * paymasterAndData, on-chain staking, balance deposit/withdraw,
 * EntryPoint.handleOps execution) require the paymaster's private key and an
 * on-chain contract, neither of which are available client-side. Those
 * operations therefore THROW fail-closed rather than fabricate signatures,
 * balances, or transaction outcomes.
 *
 * Gas-price information IS available via the public backend route
 * GET /api/v1/gas?chain_id=N, so the GasPriceOracle uses it directly (no
 * fabricated gas prices).
 */

#include "paymaster_service.hpp"
#include "api_client.hpp"

#include <algorithm>
#include <chrono>
#include <cstring>
#include <ctime>
#include <sstream>
#include <thread>

namespace tiger {
namespace master {
namespace paymaster {

constexpr const char* DEFAULT_ENTRY_POINT = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3";
constexpr const uint64_t DEFAULT_STAKE_AMOUNT = 100000000000000000ULL;
constexpr const uint32_t CACHE_DURATION_MS = 15000;

PaymasterService::PaymasterService(const PaymasterConfig& config)
    : config_(config)
    , gasOracle_(std::make_unique<GasPriceOracle>())
    , statsStartTime_(std::chrono::system_clock::now()) {
    SponsorshipPolicy defaultPolicy;
    defaultPolicy.policyId = "default";
    defaultPolicy.enabled = true;
    policies_[defaultPolicy.policyId] = defaultPolicy;
}

PaymasterService::~PaymasterService() { shutdown(); }

bool PaymasterService::initialize() {
    // Gas price monitoring uses the real backend /api/v1/gas endpoint.
    gasOracle_->startPriceMonitoring("1", 10000);
    return true;
}

void PaymasterService::shutdown() {
    gasOracle_->stopPriceMonitoring();
}

std::string PaymasterService::validateUserOperation(
    const UserOperation& userOp,
    const std::string& chainId
) {
    // Structural validation only; no on-chain or signature claims.
    if (userOp.sender.empty()) return "AA10: sender not specified";
    if (userOp.nonce == UINT64_MAX) return "AA11: nonce too large";
    if (userOp.callGasLimit > 5000000ULL ||
        userOp.verificationGasLimit > 5000000ULL ||
        userOp.preVerificationGas > 5000000ULL) {
        return "AA13: gas limit too high";
    }
    if (!userOp.paymasterAndData.empty()) {
        if (userOp.paymasterAndData.length() < 42) return "AA20: invalid paymasterAndData";
        std::string paymasterAddr = userOp.paymasterAndData.substr(0, 42);
        if (paymasterAddr != config_.paymasterAddress) return "AA21: wrong paymaster";
    }
    if (validationCallback_ && !validationCallback_(userOp, chainId)) {
        return "AA22: validation failed";
    }
    std::lock_guard<std::mutex> lock(policiesMutex_);
    auto policyIt = policies_.find("default");
    if (policyIt != policies_.end() && !checkSponsorshipPolicy(userOp, policyIt->second)) {
        return "AA23: not sponsored";
    }
    return "0";
}

bool PaymasterService::sponsorUserOperation(
    const UserOperation& userOp,
    const std::string& chainId,
    std::string& paymasterAndData
) {
    // Sponsoring requires the paymaster to sign the UserOperation hash with
    // its private key and (in production) submit to a bundler. The backend
    // does not expose this, so we cannot honestly produce paymasterAndData.
    // Fail closed.
    (void)userOp; (void)chainId; (void)paymasterAndData;
    throw std::runtime_error(
        "Sponsoring an ERC-4337 UserOperation requires the paymaster private "
        "key and bundler support, which are not exposed by the canonical "
        "backend");
}

bool PaymasterService::isTokenPaymasterEnabled() const {
    return config_.isTokenPaymaster;
}

std::string PaymasterService::getPaymasterToken(const std::string& chainId) {
    // Well-known canonical wrapped-token contract addresses (public constants).
    static const std::map<std::string, std::string> wrappedTokens = {
        {"1", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"},    // WETH
        {"56", "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"},  // WBNB
        {"137", "0x0d500B1d8E8aF31C80447cB82c33A5639970C530"}, // WMATIC
        {"42161", "0x82aF49447D8a07e3bd95BD0d56f91cfFdc2e216a"},// WETH
        {"10", "0x4200000000000000000000000000000000000006"},   // WETH
    };
    if (!config_.isTokenPaymaster) return config_.paymasterAddress;
    auto it = wrappedTokens.find(chainId);
    return it != wrappedTokens.end() ? it->second : config_.supportedToken;
}

uint64_t PaymasterService::calculatePostOpGas(
    const UserOperation& userOp,
    uint64_t actualGasUsed
) {
    // Local estimate derived from the supplied UserOperation and actual gas
    // usage; makes no claim about the actual on-chain postOp cost.
    uint64_t baseGas = 21000;
    uint64_t perUserOpGas = 21000;
    uint64_t perCalldataByte = 16;
    uint64_t perNonZeroByte = 68;
    size_t zeroBytes = 0;
    size_t nonZeroBytes = 0;
    for (uint8_t b : userOp.callData) {
        if (b == 0) zeroBytes++; else nonZeroBytes++;
    }
    uint64_t callDataGas = (zeroBytes * perCalldataByte) + (nonZeroBytes * perNonZeroByte);
    return baseGas + perUserOpGas + callDataGas + (actualGasUsed / 10);
}

double PaymasterService::calculateFee(
    const UserOperation& userOp,
    const std::string& token
) {
    auto gasPrices = gasOracle_->getGasPrices("1");
    if (!gasPrices.has_value()) return 0.0;
    uint64_t estimatedGas = estimateGas(userOp, "1");
    uint64_t totalFee = estimatedGas * gasPrices->maxFeePerGas;
    double feeWithMarkup = static_cast<double>(totalFee) * (1.0 + config_.markupPercent / 100.0);
    (void)token;
    return feeWithMarkup;
}

// ==================== Sponsorship policies (local config) ====================

std::string PaymasterService::createPolicy(const SponsorshipPolicy& policy) {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    std::string policyId = policy.policyId;
    if (policyId.empty()) {
        std::stringstream ss;
        ss << "policy_" << static_cast<uint64_t>(std::time(nullptr));
        policyId = ss.str();
    }
    policies_[policyId] = policy;
    return policyId;
}

bool PaymasterService::updatePolicy(const std::string& policyId, const SponsorshipPolicy& policy) {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    auto it = policies_.find(policyId);
    if (it == policies_.end()) return false;
    it->second = policy;
    return true;
}

bool PaymasterService::deletePolicy(const std::string& policyId) {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    if (policyId == "default") return false;
    return policies_.erase(policyId) > 0;
}

std::optional<SponsorshipPolicy> PaymasterService::getPolicy(const std::string& policyId) {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    auto it = policies_.find(policyId);
    return it != policies_.end() ? std::make_optional(it->second) : std::nullopt;
}

std::vector<SponsorshipPolicy> PaymasterService::listPolicies() const {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    std::vector<SponsorshipPolicy> result;
    for (const auto& pair : policies_) result.push_back(pair.second);
    return result;
}

// ==================== Balance management ====================

uint64_t PaymasterService::getPaymasterBalance(const std::string& chainId) const {
    // Real paymaster balance lives on-chain in the EntryPoint deposit. The
    // backend does not expose it, so we return nothing (no fabricated balance).
    (void)chainId;
    return 0;
}

bool PaymasterService::fundPaymaster(const std::string& /*chainId*/, uint64_t /*amount*/) {
    // Funding the paymaster is an on-chain deposit to the EntryPoint; not
    // available client-side.
    throw std::runtime_error(
        "Funding the paymaster is an on-chain deposit not exposed by the "
        "canonical backend");
}

bool PaymasterService::withdrawBalance(const std::string& /*chainId*/, uint64_t /*amount*/) {
    throw std::runtime_error(
        "Withdrawing paymaster balance is an on-chain operation not exposed "
        "by the canonical backend");
}

// ==================== Stake management ====================

bool PaymasterService::stake(uint64_t /*amount*/, uint32_t /*unstakeDelaySec*/) {
    // Staking is an on-chain EntryPoint.addStake() transaction; not available
    // client-side.
    throw std::runtime_error(
        "Paymaster staking is an on-chain operation not exposed by the "
        "canonical backend");
}

bool PaymasterService::unstake() {
    throw std::runtime_error(
        "Paymaster unstaking is an on-chain operation not exposed by the "
        "canonical backend");
}

bool PaymasterService::addStake(uint64_t /*amount*/) {
    throw std::runtime_error(
        "Adding paymaster stake is an on-chain operation not exposed by the "
        "canonical backend");
}

// ==================== Statistics ====================

PaymasterService::PaymasterStats PaymasterService::getStats() const {
    PaymasterStats stats{};
    stats.totalSponsored = totalSponsored_.load();
    stats.totalSuccessful = totalSuccessful_.load();
    stats.totalFailed = totalFailed_.load();
    stats.totalGasUsed = totalGasUsed_.load();
    if (stats.totalSponsored > 0) {
        stats.successRate = static_cast<double>(stats.totalSuccessful) /
                            static_cast<double>(stats.totalSponsored) * 100.0;
        stats.averageGasPrice = stats.totalGasUsed > 0 ?
            totalGasUsed_.load() / stats.totalSponsored : 0;
    }
    stats.lastUpdated = std::chrono::system_clock::now();
    return stats;
}

void PaymasterService::resetStats() {
    totalSponsored_ = 0;
    totalSuccessful_ = 0;
    totalFailed_ = 0;
    totalGasUsed_ = 0;
    statsStartTime_ = std::chrono::system_clock::now();
}

void PaymasterService::setValidationCallback(ValidationCallback callback) {
    validationCallback_ = callback;
}

void PaymasterService::setSponsorshipCallback(SponsorshipCallback callback) {
    sponsorshipCallback_ = callback;
}

// ==================== Private methods ====================

bool PaymasterService::validateSignature(
    const UserOperation& /*userOp*/,
    const std::string& /*chainId*/
) {
    // Signature verification requires the paymaster's verification logic on a
    // chain node; not available client-side. Fail closed.
    throw std::runtime_error(
        "Paymaster signature verification is not available client-side");
}

bool PaymasterService::checkSponsorshipPolicy(
    const UserOperation& userOp,
    const SponsorshipPolicy& policy
) {
    if (!policy.enabled) return false;
    if (!policy.isSenderAllowed(userOp.sender)) return false;
    uint64_t value = userOp.maxFeePerGas * userOp.callGasLimit;
    if (!policy.canSponsor(value)) return false;
    if (policy.enableRateLimiting && policy.maxTransactionsPerDay == 0) return false;
    return true;
}

std::string PaymasterService::buildPaymasterAndData(
    const UserOperation& /*userOp*/,
    const std::string& /*chainId*/
) {
    // paymasterAndData must end with a valid paymaster signature over the
    // UserOperation hash. Producing it requires the paymaster's private key,
    // which is not available client-side. Fail closed.
    throw std::runtime_error(
        "Building paymasterAndData requires the paymaster private key, which "
        "is not available client-side");
}

bool PaymasterService::executeUserOperation(
    const UserOperation& /*userOp*/,
    const std::string& /*chainId*/
) {
    // handleOps is an on-chain EntryPoint call; not available client-side.
    throw std::runtime_error(
        "Executing an ERC-4337 UserOperation requires EntryPoint/bundler "
        "support not exposed by the canonical backend");
}

void PaymasterService::recordSuccess(const UserOperation& /*userOp*/, uint64_t gasUsed) {
    totalSponsored_++;
    totalSuccessful_++;
    totalGasUsed_ += gasUsed;
}

void PaymasterService::recordFailure(const UserOperation& /*userOp*/, const std::string& /*error*/) {
    totalSponsored_++;
    totalFailed_++;
}

uint64_t PaymasterService::estimateGas(
    const UserOperation& userOp,
    const std::string& /*chainId*/
) {
    uint64_t baseEstimate = 21000;
    uint64_t verificationGas = userOp.verificationGasLimit > 0 ? userOp.verificationGasLimit : 150000;
    uint64_t callGas = userOp.callGasLimit > 0 ? userOp.callGasLimit : 100000;
    uint64_t preVerificationGas = userOp.preVerificationGas > 0 ? userOp.preVerificationGas : 21000;
    return baseEstimate + verificationGas + callGas + preVerificationGas;
}

bool PaymasterService::verifyPaymasterStake() const {
    // Real stake verification reads on-chain EntryPoint storage; not available
    // client-side. Fail closed.
    return false;
}

// ==================== GasPriceOracle (real backend /api/v1/gas) ====================

GasPriceOracle::GasPriceOracle() : monitoring_(false) {}

GasPriceOracle::~GasPriceOracle() { stopPriceMonitoring(); }

std::optional<GasPriceOracle::GasPrices> GasPriceOracle::getGasPrices(const std::string& chainId) {
    std::lock_guard<std::mutex> lock(cacheMutex_);
    auto it = gasCache_.find(chainId);
    if (it == gasCache_.end()) return std::nullopt;
    auto age = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now() - it->second.timestamp).count();
    if (age > CACHE_DURATION_MS) {
        gasCache_.erase(it);
        return std::nullopt;
    }
    return it->second;
}

void GasPriceOracle::updateGasPrices(const std::string& chainId, const GasPrices& prices) {
    std::lock_guard<std::mutex> lock(cacheMutex_);
    gasCache_[chainId] = prices;
}

void GasPriceOracle::startPriceMonitoring(const std::string& chainId, uint32_t intervalMs) {
    if (monitoring_.load()) return;
    monitoring_ = true;
    monitorThread_ = std::thread([this, chainId, intervalMs]() {
        while (monitoring_.load()) {
            try {
                GasPrices prices{};
                if (fetchFromBackend(chainId, prices)) {
                    prices.timestamp = std::chrono::system_clock::now();
                    updateGasPrices(chainId, prices);
                }
            } catch (...) {
                // Continue monitoring even on backend errors.
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(intervalMs));
        }
    });
}

void GasPriceOracle::stopPriceMonitoring() {
    monitoring_ = false;
    if (monitorThread_.joinable()) monitorThread_.join();
}

bool GasPriceOracle::fetchFromBackend(const std::string& chainId, GasPrices& out) {
    // Real gas prices from the canonical backend: GET /api/v1/gas?chain_id=N.
    // Backend returns {gas_price, max_fee, priority_fee} (wei as numbers).
    std::map<std::string, std::string> params = {{"chain_id", chainId}};
    std::string body = api::backendGet("/api/v1/gas", params);

    auto gasPrice = api::jsonNumberField(body, "gas_price");
    auto maxFee = api::jsonNumberField(body, "max_fee");
    auto priorityFee = api::jsonNumberField(body, "priority_fee");

    if (!gasPrice && !maxFee) return false;  // backend unreachable / no data

    uint64_t gp = gasPrice.has_value() ? static_cast<uint64_t>(*gasPrice) : 0;
    uint64_t mf = maxFee.has_value() ? static_cast<uint64_t>(*maxFee) : gp;
    uint64_t pf = priorityFee.has_value() ? static_cast<uint64_t>(*priorityFee) : 0;

    out.baseFeePerGas = gp;
    out.maxFeePerGas = mf;
    out.maxPriorityFeePerGas = pf;
    out.suggestedMaxFeePerGas = mf;
    out.suggestedMaxPriorityFeePerGas = pf;
    out.timestamp = std::chrono::system_clock::now();
    return true;
}

uint64_t GasPriceOracle::calculateAverage(const std::vector<uint64_t>& values) {
    if (values.empty()) return 0;
    uint64_t sum = 0;
    for (uint64_t v : values) sum += v;
    return sum / values.size();
}

} // namespace paymaster
} // namespace master
} // namespace tiger
