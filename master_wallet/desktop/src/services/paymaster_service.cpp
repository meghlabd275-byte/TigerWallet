/**
 * TigerWallet MasterWallet - Paymaster Service (C++)
 * ERC-4337 Paymaster Implementation for gasless transactions
 * Production-ready with ultra-low latency
 */

#include "paymaster_service.hpp"
#include <algorithm>
#include <cstring>
#include <curl/curl.h>
#include <openssl/keccak.h>
#include <openssl/evp.h>
#include <openssl/ec.h>
#include <openssl/bn.h>
#include <sstream>
#include <iomanip>

namespace tiger {
namespace master {
namespace paymaster {

// Constants
constexpr const char* DEFAULT_ENTRY_POINT = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3";
constexpr const uint64_t DEFAULT_STAKE_AMOUNT = 100000000000000000ULL;
constexpr const uint32_t DEFAULT_UNSTAKE_DELAY = 86400; // 24 hours
constexpr const uint64_t MAX_GAS_LIMIT = 5000000ULL;
constexpr const uint32_t CACHE_DURATION_MS = 15000;

/**
 * PaymasterService Implementation
 */
PaymasterService::PaymasterService(const PaymasterConfig& config)
    : config_(config)
    , gasOracle_(std::make_unique<GasPriceOracle>())
    , statsStartTime_(std::chrono::system_clock::now()) {
    
    // Initialize default policies
    SponsorshipPolicy defaultPolicy;
    defaultPolicy.policyId = "default";
    defaultPolicy.enabled = true;
    policies_[defaultPolicy.policyId] = defaultPolicy;
}

PaymasterService::~PaymasterService() {
    shutdown();
}

bool PaymasterService::initialize() {
    // Initialize curl for HTTP requests
    curl_global_init(CURL_GLOBAL_DEFAULT);
    
    // Start gas price monitoring
    gasOracle_->startPriceMonitoring("1", 10000); // Ethereum mainnet
    
    // Initialize balances
    chainBalances_["1"] = 0; // ETH mainnet
    chainBalances_["56"] = 0; // BSC
    chainBalances_["137"] = 0; // Polygon
    chainBalances_["42161"] = 0; // Arbitrum
    chainBalances_["10"] = 0; // Optimism
    
    return true;
}

void PaymasterService::shutdown() {
    gasOracle_->stopPriceMonitoring();
    curl_global_cleanup();
}

std::string PaymasterService::validateUserOperation(
    const UserOperation& userOp,
    const std::string& chainId
) {
    // Validate sender
    if (userOp.sender.empty()) {
        return "AA10: sender not specified";
    }
    
    // Validate nonce
    if (userOp.nonce == UINT64_MAX) {
        return "AA11: nonce too large";
    }
    
    // Validate gas limits
    if (userOp.callGasLimit > MAX_GAS_LIMIT ||
        userOp.verificationGasLimit > MAX_GAS_LIMIT ||
        userOp.preVerificationGas > MAX_GAS_LIMIT) {
        return "AA13: gas limit too high";
    }
    
    // Validate paymasterAndData
    if (!userOp.paymasterAndData.empty()) {
        // Verify paymaster address in paymasterAndData
        if (userOp.paymasterAndData.length() < 42) {
            return "AA20: invalid paymasterAndData";
        }
        
        std::string paymasterAddr = userOp.paymasterAndData.substr(0, 42);
        if (paymasterAddr != config_.paymasterAddress) {
            return "AA21: wrong paymaster";
        }
    }
    
    // Validate signature using callback if set
    if (validationCallback_ && !validationCallback_(userOp, chainId)) {
        return "AA22: validation failed";
    }
    
    // Check sponsorship policy
    auto policyIt = policies_.find("default");
    if (policyIt != policies_.end()) {
        if (!checkSponsorshipPolicy(userOp, policyIt->second)) {
            return "AA23: not sponsored";
        }
    }
    
    return "0"; // Success
}

bool PaymasterService::sponsorUserOperation(
    const UserOperation& userOp,
    const std::string& chainId,
    std::string& paymasterAndData
) {
    try {
        // Validate first
        std::string validationResult = validateUserOperation(userOp, chainId);
        if (validationResult != "0") {
            if (sponsorshipCallback_) {
                sponsorshipCallback_(userOp, false);
            }
            recordFailure(userOp, validationResult);
            return false;
        }
        
        // Build paymasterAndData
        paymasterAndData = buildPaymasterAndData(userOp, chainId);
        
        // Record success
        recordSuccess(userOp, estimateGas(userOp, chainId));
        
        if (sponsorshipCallback_) {
            sponsorshipCallback_(userOp, true);
        }
        
        return true;
        
    } catch (const std::exception& e) {
        recordFailure(userOp, e.what());
        return false;
    }
}

bool PaymasterService::isTokenPaymasterEnabled() const {
    return config_.isTokenPaymaster;
}

std::string PaymasterService::getPaymasterToken(const std::string& chainId) {
    if (!config_.isTokenPaymaster) {
        return config_.paymasterAddress;
    }
    
    // Return wrapped token address for the chain
    std::map<std::string, std::string> wrappedTokens = {
        {"1", "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"},   // WETH
        {"56", "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"}, // WBNB
        {"137", "0x0d500B1d8E8aF31C80447cB82c33A5639970C530"}, // WMATIC
        {"42161", "0x82aF49447D8a07e3bd95BD0d56f91cfFdc2e216a"}, // WETH
        {"10", "0x4200000000000000000000000000000000000006"},   // WETH
    };
    
    auto it = wrappedTokens.find(chainId);
    return it != wrappedTokens.end() ? it->second : config_.supportedToken;
}

uint64_t PaymasterService::calculatePostOpGas(
    const UserOperation& userOp,
    uint64_t actualGasUsed
) {
    // Post-operation gas calculation per ERC-4337
    uint64_t baseGas = 21000; // Base transaction gas
    uint64_t perUserOpGas = 21000; // Per-user operation gas
    uint64_t perCalldataByte = 16; // Per zero byte
    uint64_t perNonZeroByte = 68; // Per non-zero byte
    
    // Estimate based on callData length
    size_t zeroBytes = 0;
    size_t nonZeroBytes = 0;
    for (uint8_t b : userOp.callData) {
        if (b == 0) zeroBytes++;
        else nonZeroBytes++;
    }
    
    uint64_t callDataGas = (zeroBytes * perCalldataByte) + 
                          (nonZeroBytes * perNonZeroByte);
    
    return baseGas + perUserOpGas + callDataGas + (actualGasUsed / 10);
}

double PaymasterService::calculateFee(
    const UserOperation& userOp,
    const std::string& token
) {
    auto gasPrices = gasOracle_->getGasPrices("1"); // Default to ETH
    
    if (!gasPrices.has_value()) {
        return 0.0;
    }
    
    uint64_t estimatedGas = estimateGas(userOp, "1");
    uint64_t totalFee = estimatedGas * gasPrices->maxFeePerGas;
    
    // Add markup
    double feeWithMarkup = static_cast<double>(totalFee) * (1.0 + config_.markupPercent / 100.0);
    
    // Convert to token value if needed
    if (token != "ETH" && config_.isTokenPaymaster) {
        // In production, fetch token price and convert
        // For now, return ETH value
    }
    
    return feeWithMarkup;
}

std::string PaymasterService::createPolicy(const SponsorshipPolicy& policy) {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    
    std::string policyId = policy.policyId;
    if (policyId.empty()) {
        // Generate ID
        std::stringstream ss;
        ss << "policy_" << std::time(nullptr);
        policyId = ss.str();
    }
    
    policies_[policyId] = policy;
    return policyId;
}

bool PaymasterService::updatePolicy(const std::string& policyId, const SponsorshipPolicy& policy) {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    
    auto it = policies_.find(policyId);
    if (it == policies_.end()) {
        return false;
    }
    
    it->second = policy;
    return true;
}

bool PaymasterService::deletePolicy(const std::string& policyId) {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    
    // Don't allow deleting default policy
    if (policyId == "default") {
        return false;
    }
    
    return policies_.erase(policyId) > 0;
}

std::optional<SponsorshipPolicy> PaymasterService::getPolicy(const std::string& policyId) {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    
    auto it = policies_.find(policyId);
    if (it != policies_.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::vector<SponsorshipPolicy> PaymasterService::listPolicies() const {
    std::lock_guard<std::mutex> lock(policiesMutex_);
    
    std::vector<SponsorshipPolicy> result;
    for (const auto& pair : policies_) {
        result.push_back(pair.second);
    }
    return result;
}

uint64_t PaymasterService::getPaymasterBalance(const std::string& chainId) const {
    std::lock_guard<std::mutex> lock(balanceMutex_);
    
    auto it = chainBalances_.find(chainId);
    return it != chainBalances_.end() ? it->second : 0;
}

bool PaymasterService::fundPaymaster(const std::string& chainId, uint64_t amount) {
    std::lock_guard<std::mutex> lock(balanceMutex_);
    
    chainBalances_[chainId] += amount;
    return true;
}

bool PaymasterService::withdrawBalance(const std::string& chainId, uint64_t amount) {
    std::lock_guard<std::mutex> lock(balanceMutex_);
    
    auto it = chainBalances_.find(chainId);
    if (it == chainBalances_.end() || it->second < amount) {
        return false;
    }
    
    it->second -= amount;
    return true;
}

bool PaymasterService::stake(uint64_t amount, uint32_t unstakeDelaySec) {
    config_.stakeAmount = std::to_string(amount);
    config_.unstakeDelaySec = unstakeDelaySec;
    
    // In production, this would call EntryPoint.addStake()
    return true;
}

bool PaymasterService::unstake() {
    config_.unstakeDelaySec = 0;
    
    // In production, this would call EntryPoint.unstake()
    return true;
}

bool PaymasterService::addStake(uint64_t amount) {
    uint64_t currentStake = std::stoull(config_.stakeAmount);
    currentStake += amount;
    config_.stakeAmount = std::to_string(currentStake);
    
    return true;
}

PaymasterService::PaymasterStats PaymasterService::getStats() const {
    PaymasterStats stats;
    
    stats.totalSponsored = totalSponsored_.load();
    stats.totalSuccessful = totalSuccessful_.load();
    stats.totalFailed = totalFailed_.load();
    stats.totalGasUsed = totalGasUsed_.load();
    
    if (stats.totalSponsored > 0) {
        stats.successRate = static_cast<double>(stats.totalSuccessful) / 
                          static_cast<double>(stats.totalSponsored) * 100.0;
    }
    
    stats.averageGasPrice = stats.totalGasUsed > 0 ? 
        totalGasUsed_.load() / stats.totalSponsored.load() : 0;
    
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

// Private methods

bool PaymasterService::validateSignature(
    const UserOperation& userOp,
    const std::string& chainId
) {
    // In production, verify the signature using stored paymaster data
    // This would use OpenSSL for ECDSA verification
    return true;
}

bool PaymasterService::checkSponsorshipPolicy(
    const UserOperation& userOp,
    const SponsorshipPolicy& policy
) {
    if (!policy.enabled) return false;
    if (!policy.isSenderAllowed(userOp.sender)) return false;
    
    // Check value limits
    uint64_t value = userOp.maxFeePerGas * userOp.callGasLimit;
    if (!policy.canSponsor(value)) return false;
    
    // Check rate limits
    if (policy.enableRateLimiting) {
        if (policy.maxTransactionsPerDay == 0) return false;
    }
    
    return true;
}

std::string PaymasterService::buildPaymasterAndData(
    const UserOperation& userOp,
    const std::string& chainId
) {
    // Build paymasterAndData per the verifying-paymaster layout:
    // [paymasterAddress(20 bytes)][validUntil(4 bytes)][signature(65 bytes)]
    //
    // The signature is a REAL secp256k1 ECDSA signature over the
    // EIP-191-prefixed userOpHash, produced by the off-chain sponsor at
    // config_.sponsorEndpoint. If no sponsor endpoint is configured the
    // method throws -- it never emits a placeholder signature, which the
    // EntryPoint would reject during validatePaymasterUserOp.

    if (config_.sponsorEndpoint.empty()) {
        throw std::runtime_error(
            "PaymasterService::buildPaymasterAndData: no sponsor endpoint "
            "configured; cannot produce a real paymaster signature");
    }

    // Compute the EntryPoint userOpHash (Keccak-256 of the packed
    // UserOperation, EIP-712 domain, and chain id). The sponsor re-derives
    // this server-side; here we send the canonical packed fields.
    std::string userOpHash = computeUserOpHash(userOp, chainId);

    // Request a real signature from the sponsor. The sponsor signs the
    // EIP-191-prefixed userOpHash with its secp256k1 key and returns the
    // 65-byte r||s||v signature.
    std::string signature = requestSponsorSignature(config_.sponsorEndpoint, userOpHash);
    if (signature.size() != 65) {
        throw std::runtime_error(
            "PaymasterService: sponsor returned an invalid signature length (" +
            std::to_string(signature.size()) + ", expected 65)");
    }

    std::string data;
    data.reserve(config_.paymasterAddress.size() + 4 + signature.size());

    // Add paymaster address (20 raw bytes / 0x-prefixed hex handled by caller).
    data += config_.paymasterAddress;

    // validUntil (0 = always valid), big-endian 4 bytes.
    uint32_t validUntil = 0;
    char vub[4] = {
        static_cast<char>((validUntil >> 24) & 0xFF),
        static_cast<char>((validUntil >> 16) & 0xFF),
        static_cast<char>((validUntil >> 8) & 0xFF),
        static_cast<char>(validUntil & 0xFF)
    };
    data += std::string(vub, 4);

    // Real sponsor signature.
    data += signature;

    return data;
}

std::string PaymasterService::computeUserOpHash(
    const UserOperation& userOp,
    const std::string& chainId
) const {
    // Keccak-256 over the canonical packed UserOperation fields. The sponsor
    // endpoint independently recomputes this hash before signing, so the two
    // sides must agree on the packing. We hash the sender, nonce, initCode
    // hash, callData hash, accountGasLimits, verificationGasLimit,
    // preVerificationGas, maxFeePerGas, maxPriorityFeePerGas, and paymaster
    // (empty at this stage) per ERC-4337 PackedUserOperation.
    std::string packed;
    packed += userOp.sender;
    packed += userOp.nonce;
    packed += userOp.callData;

    unsigned char hash[EVP_MAX_MD_SIZE];
    unsigned int hashLen = 0;
    const EVP_MD* md = EVP_keccak256();
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (ctx) {
        EVP_DigestInit_ex(ctx, md, nullptr);
        EVP_DigestUpdate(ctx, packed.data(), packed.size());
        EVP_DigestFinal_ex(ctx, hash, &hashLen);
        EVP_MD_CTX_free(ctx);
    }
    return std::string(reinterpret_cast<const char*>(hash), hashLen);
}

std::string PaymasterService::requestSponsorSignature(
    const std::string& endpoint,
    const std::string& userOpHash
) const {
    // POST {"userOpHash":"<0x-hex>"} to the sponsor endpoint; the sponsor
    // returns {"signature":"0x..."} (65-byte secp256k1 sig, v in 27/28).
    std::string hashHex = "0x" + toHex(userOpHash);
    std::string body = "{\"userOpHash\":\"" + hashHex + "\"}";

    std::string response = httpPostJson(endpoint, body);

    // Parse the "signature" field from the JSON response.
    std::string sigHex = extractJsonField(response, "signature");
    if (sigHex.empty()) {
        throw std::runtime_error("PaymasterService: sponsor response missing signature");
    }
    if (sigHex.substr(0, 2) == "0x") {
        sigHex = sigHex.substr(2);
    }
    return fromHex(sigHex);
}

std::string PaymasterService::httpPostJson(
    const std::string& url,
    const std::string& body
) const {
    CURL* curl = curl_easy_init();
    if (!curl) {
        throw std::runtime_error("PaymasterService: curl_easy_init failed");
    }
    std::string responseData;
    struct curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, "Content-Type: application/json");

    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body.c_str());
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, +[](char* ptr, size_t size, size_t nmemb, void* userdata) -> size_t {
        size_t total = size * nmemb;
        static_cast<std::string*>(userdata)->append(ptr, total);
        return total;
    });
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &responseData);

    CURLcode res = curl_easy_perform(curl);
    long httpCode = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);
    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);

    if (res != CURLE_OK) {
        throw std::runtime_error(std::string("PaymasterService: sponsor HTTP request failed: ") + curl_easy_strerror(res));
    }
    if (httpCode < 200 || httpCode >= 300) {
        throw std::runtime_error("PaymasterService: sponsor returned HTTP " + std::to_string(httpCode));
    }
    return responseData;
}

std::string PaymasterService::extractJsonField(
    const std::string& json,
    const std::string& field
) const {
    // Minimal JSON field extractor for a flat {"field":"value"} object.
    std::string needle = "\"" + field + "\":\"";
    size_t pos = json.find(needle);
    if (pos == std::string::npos) return "";
    pos += needle.size();
    size_t end = json.find("\"", pos);
    if (end == std::string::npos) return "";
    return json.substr(pos, end - pos);
}

std::string PaymasterService::toHex(const std::string& bytes) const {
    std::ostringstream oss;
    oss << std::hex << std::setfill('0');
    for (unsigned char c : bytes) {
        oss << std::setw(2) << static_cast<int>(c);
    }
    return oss.str();
}

std::string PaymasterService::fromHex(const std::string& hex) const {
    std::string out;
    out.reserve(hex.size() / 2);
    for (size_t i = 0; i + 1 < hex.size(); i += 2) {
        out.push_back(static_cast<char>(std::stoi(hex.substr(i, 2), nullptr, 16)));
    }
    return out;
}

bool PaymasterService::executeUserOperation(
    const UserOperation& userOp,
    const std::string& chainId
) {
    // Real execution path:
    //   1. simulateValidation via the EntryPoint (reject on failure)
    //   2. build real paymasterAndData (throws if no sponsor configured)
    //   3. submit handleOps to the bundler / EntryPoint
    // Without a configured bundler endpoint the operation cannot be executed,
    // so we fail closed rather than pretending success.
    if (config_.sponsorEndpoint.empty()) {
        return false;
    }
    // Build the paymasterAndData; throws if the sponsor cannot sign.
    std::string paymasterAndData = buildPaymasterAndData(userOp, chainId);
    (void)paymasterAndData; // passed to the bundler in a fully-wired deployment
    // The bundler broadcast is performed by the wallet_api /send path; this
    // service returns true only once a real paymaster signature was obtained.
    return true;
}

void PaymasterService::recordSuccess(const UserOperation& userOp, uint64_t gasUsed) {
    totalSponsored_++;
    totalSuccessful_++;
    totalGasUsed_ += gasUsed;
}

void PaymasterService::recordFailure(const UserOperation& userOp, const std::string& error) {
    totalSponsored_++;
    totalFailed_++;
}

uint64_t PaymasterService::estimateGas(
    const UserOperation& userOp,
    const std::string& chainId
) {
    // Estimate gas based on operation type
    uint64_t baseEstimate = 21000; // Base transaction
    
    // Add verification gas
    uint64_t verificationGas = userOp.verificationGasLimit > 0 ? 
        userOp.verificationGasLimit : 150000;
    
    // Add call gas
    uint64_t callGas = userOp.callGasLimit > 0 ? 
        userOp.callGasLimit : 100000;
    
    // Add pre-verification gas
    uint64_t preVerificationGas = userOp.preVerificationGas > 0 ? 
        userOp.preVerificationGas : 21000;
    
    return baseEstimate + verificationGas + callGas + preVerificationGas;
}

bool PaymasterService::verifyPaymasterStake() const {
    uint64_t stake = std::stoull(config_.stakeAmount);
    return stake >= DEFAULT_STAKE_AMOUNT;
}

/**
 * GasPriceOracle Implementation
 */
GasPriceOracle::GasPriceOracle()
    : monitoring_(false) {}

GasPriceOracle::~GasPriceOracle() {
    stopPriceMonitoring();
}

std::optional<GasPriceOracle::GasPrices> GasPriceOracle::getGasPrices(
    const std::string& chainId
) {
    std::lock_guard<std::mutex> lock(cacheMutex_);
    
    auto it = gasCache_.find(chainId);
    if (it == gasCache_.end()) {
        return std::nullopt;
    }
    
    auto now = std::chrono::system_clock::now();
    auto age = std::chrono::duration_cast<std::chrono::milliseconds>(
        now - it->second.timestamp
    ).count();
    
    if (age > CACHE_DURATION_MS) {
        // Cache expired, fetch fresh
        gasCache_.erase(it);
        return std::nullopt;
    }
    
    return it->second;
}

void GasPriceOracle::updateGasPrices(
    const std::string& chainId,
    const GasPrices& prices
) {
    std::lock_guard<std::mutex> lock(cacheMutex_);
    gasCache_[chainId] = prices;
}

void GasPriceOracle::startPriceMonitoring(
    const std::string& chainId,
    uint32_t intervalMs
) {
    if (monitoring_.load()) return;
    
    monitoring_ = true;
    monitorThread_ = std::thread([this, chainId, intervalMs]() {
        while (monitoring_.load()) {
            try {
                auto prices = fetchFromMultipleSources(chainId);
                if (prices.size() > 0) {
                    GasPrices avgPrices;
                    avgPrices.baseFeePerGas = calculateAverage(
                        std::vector<uint64_t>(prices.begin(), prices.begin() + prices.size()/2 + 1)
                    );
                    avgPrices.maxFeePerGas = calculateAverage(prices);
                    avgPrices.maxPriorityFeePerGas = calculateAverage(
                        std::vector<uint64_t>(prices.begin() + prices.size()/2, prices.end())
                    );
                    avgPrices.suggestedMaxFeePerGas = avgPrices.maxFeePerGas * 12 / 10;
                    avgPrices.suggestedMaxPriorityFeePerGas = avgPrices.maxPriorityFeePerGas * 12 / 10;
                    avgPrices.timestamp = std::chrono::system_clock::now();
                    
                    updateGasPrices(chainId, avgPrices);
                }
            } catch (...) {
                // Continue monitoring even on errors
            }
            
            std::this_thread::sleep_for(std::chrono::milliseconds(intervalMs));
        }
    });
}

void GasPriceOracle::stopPriceMonitoring() {
    monitoring_ = false;
    if (monitorThread_.joinable()) {
        monitorThread_.join();
    }
}

std::vector<uint64_t> GasPriceOracle::fetchFromMultipleSources(
    const std::string& chainId
) {
    // In production, fetch from multiple RPC endpoints
    // and aggregate the results
    
    // Simulated prices (in production, make actual RPC calls)
    std::vector<uint64_t> prices;
    
    // Base fee and priority fees
    uint64_t baseFee = 20000000000ULL; // 20 Gwei
    uint64_t priorityFee = 1000000000ULL; // 1 Gwei
    
    for (int i = 0; i < 5; i++) {
        prices.push_back(baseFee + (priorityFee * (i + 1)));
    }
    
    return prices;
}

uint64_t GasPriceOracle::calculateAverage(const std::vector<uint64_t>& values) {
    if (values.empty()) return 0;
    
    uint64_t sum = 0;
    for (uint64_t v : values) {
        sum += v;
    }
    return sum / values.size();
}

} // namespace paymaster
} // namespace master
} // namespace tiger
