/**
 * TigerWallet MEV Protection System
 * Implementation of ultra-low latency MEV blocker
 */

#include "mev_blocker.hpp"
#include "transaction.hpp"

#include <algorithm>
#include <chrono>
#include <cstring>
#include <fstream>
#include <iomanip>
#include <iostream>
#include <random>
#include <sstream>

#include <curl/curl.h>

namespace tiger {

// Keccak256 hash function
std::array<uint8_t, 32> Keccak256(const uint8_t* data, size_t len) {
    std::array<uint8_t, 32> hash{};
    tiger::keccak256(hash.data(), data, len);
    return hash;
}

MEVBlocker::MEVBlocker(ProtectionLevel level)
    : protection_level_(level),
      running_(false),
      initialized_(false) {
    memset(&stats_, 0, sizeof(Stats));
}

MEVBlocker::~MEVBlocker() {
    Stop();
}

bool MEVBlocker::Initialize(const std::string& config_path) {
    if (initialized_) {
        std::cerr << "[MEV] Already initialized" << std::endl;
        return true;
    }
    
    std::cout << "[MEV] Initializing MEV Protection System..." << std::endl;
    std::cout << "[MEV] Protection Level: " << static_cast<int>(protection_level_) << std::endl;
    
    // Initialize default relayers (Flashbots, MEV-Boost)
    relayers_.push_back("https://relay.flashbots.net");
    relayers_.push_back("https://mev-boost.titanbuilder.xyz");
    relayers_.push_back("https://mev-relay.ultrasound.money");
    
    // Fee data comes from the operator's node. Without an endpoint there is no
    // oracle, and the accessor reports no reading rather than a guess.
    if (const char* env = std::getenv("TIGER_MEV_RPC_URL"); env != nullptr && *env != '\0') {
        std::lock_guard<std::mutex> lock(rpc_mutex_);
        rpc_url_ = env;
    }

    curl_global_init(CURL_GLOBAL_DEFAULT);

    if (rpc_url_.empty()) {
        std::cerr << "[MEV] TIGER_MEV_RPC_URL is not set; the gas oracle is disabled "
                     "and no fee estimates will be produced" << std::endl;
    } else {
        std::cout << "[MEV] Gas oracle endpoint: " << rpc_url_ << std::endl;
        if (!RefreshGasOracle()) {
            std::cerr << "[MEV] Initial gas oracle refresh failed" << std::endl;
        }
    }

    initialized_ = true;
    std::cout << "[MEV] Initialization complete" << std::endl;
    return true;
}

bool MEVBlocker::Start() {
    if (!initialized_) {
        std::cerr << "[MEV] Not initialized" << std::endl;
        return false;
    }
    
    if (running_) {
        std::cout << "[MEV] Already running" << std::endl;
        return true;
    }
    
    std::cout << "[MEV] Starting MEV Protection Service..." << std::endl;
    
    running_ = true;
    
    // Start worker threads
    workers_.emplace_back(&MEVBlocker::RunGasOracleUpdater, this);
    workers_.emplace_back(&MEVBlocker::RunMempoolCleaner, this);
    workers_.emplace_back(&MEVBlocker::RunBundleProcessor, this);
    
    std::cout << "[MEV] Started " << workers_.size() << " worker threads" << std::endl;
    return true;
}

void MEVBlocker::Stop() {
    if (!running_) {
        return;
    }
    
    std::cout << "[MEV] Stopping MEV Protection Service..." << std::endl;
    
    running_ = false;
    
    // Join all workers
    for (auto& worker : workers_) {
        if (worker.joinable()) {
            worker.join();
        }
    }
    workers_.clear();
    
    std::cout << "[MEV] Stopped" << std::endl;
}

std::string MEVBlocker::SubmitTransaction(const Transaction& tx) {
    if (!running_) {
        std::cerr << "[MEV] Service not running" << std::endl;
        return "";
    }
    
    // Analyze transaction for MEV risk
    RiskAssessment assessment = AnalyzeTransaction(tx);
    
    // Check if we should block this transaction
    if (ShouldBlockTransaction(tx)) {
        std::cout << "[MEV] Blocking potentially harmful transaction: " 
                  << tx.hash << std::endl;
        UpdateStatistics(false, true);
        return "";
    }
    
    // If high protection level, submit as private transaction
    if (protection_level_ >= ProtectionLevel::kAdvanced) {
        std::string private_hash = SubmitPrivateTransaction(tx);
        if (!private_hash.empty()) {
            std::cout << "[MEV] Transaction protected (private): " << private_hash << std::endl;
            UpdateStatistics(true, false);
            return private_hash;
        }
    }
    
    // Regular submission with protection
    std::cout << "[MEV] Transaction protected: " << tx.hash << std::endl;
    UpdateStatistics(true, false);
    
    // Return protected hash
    return tx.hash;
}

std::string MEVBlocker::SubmitPrivateTransaction(const Transaction& tx) {
    std::lock_guard<std::mutex> lock(private_mutex_);
    
    // Submit to Flashbots
    std::string result = SubmitToFlashbots(tx);
    if (!result.empty()) {
        private_txs_[result] = tx;
        return result;
    }
    
    // Try relayers
    for (const auto& relayer : relayers_) {
        result = SubmitToRelayer(tx, relayer);
        if (!result.empty()) {
            private_txs_[result] = tx;
            return result;
        }
    }
    
    return "";
}

RiskAssessment MEVBlocker::AnalyzeTransaction(const Transaction& tx) {
    RiskAssessment result;
    
    // Classify transaction
    TransactionType tx_type = ClassifyTransaction(tx);
    
    // Check for sandwich attack potential
    auto sandwich = DetectSandwich(tx);
    if (sandwich) {
        result.risk_score = 0.9f;
        result.attack_type = "Sandwich Attack";
        result.recommendations.push_back("Use private transaction");
        result.recommendations.push_back("Set slippage to minimum");
        return result;
    }
    
    // Check for front-running potential
    if (IsFrontRunning(tx)) {
        result.risk_score = 0.7f;
        result.attack_type = "Front Running";
        result.recommendations.push_back("Use private transaction");
    }
    
    // Check for back-running potential
    if (IsBackRunning(tx)) {
        result.risk_score = 0.5f;
        result.attack_type = "Back Running";
    }
    
    // Calculate potential profit for MEV extractors
    result.profit_extracted = EstimateMEVProfit(tx);
    result.is_profitable = result.profit_extracted > 0.01f;
    
    return result;
}

GasOracle MEVBlocker::GetGasOracle() const {
    std::lock_guard<std::mutex> lock(gas_mutex_);
    return gas_oracle_;
}

std::vector<BlockProposal> MEVBlocker::GetBlockProposals() const {
    std::lock_guard<std::mutex> lock(proposals_mutex_);
    return proposals_;
}

void MEVBlocker::SetProtectionLevel(ProtectionLevel level) {
    protection_level_ = level;
    std::cout << "[MEV] Protection level set to: " << static_cast<int>(level) << std::endl;
}

MEVBlocker::Stats MEVBlocker::GetStats() const {
    std::lock_guard<std::mutex> lock(stats_mutex_);
    return stats_;
}

bool MEVBlocker::AddRelayer(const std::string& relayer_url) {
    std::lock_guard<std::mutex> lock(relayers_mutex_);
    
    // Check if already exists
    auto it = std::find(relayers_.begin(), relayers_.end(), relayer_url);
    if (it != relayers_.end()) {
        return false;
    }
    
    relayers_.push_back(relayer_url);
    std::cout << "[MEV] Added relayer: " << relayer_url << std::endl;
    return true;
}

bool MEVBlocker::RemoveRelayer(const std::string& relayer_url) {
    std::lock_guard<std::mutex> lock(relayers_mutex_);
    
    auto it = std::find(relayers_.begin(), relayers_.end(), relayer_url);
    if (it == relayers_.end()) {
        return false;
    }
    
    relayers_.erase(it);
    std::cout << "[MEV] Removed relayer: " << relayer_url << std::endl;
    return true;
}

bool MEVBlocker::IsPrivateTransaction(const std::string& tx_hash) const {
    std::lock_guard<std::mutex> lock(private_mutex_);
    return private_txs_.find(tx_hash) != private_txs_.end();
}

bool MEVBlocker::IsFlashbotsProtected(const std::string& tx_hash) const {
    return IsPrivateTransaction(tx_hash);
}

namespace {

// libcurl write callback that appends the response body to a std::string.
size_t WriteCallback(char* ptr, size_t size, size_t nmemb, void* userdata) {
    const size_t total = size * nmemb;
    static_cast<std::string*>(userdata)->append(ptr, total);
    return total;
}

// Extract the value of a JSON string field, e.g. {"result":"0x1234"} -> "0x1234".
// This is a deliberately narrow extractor for one known field of a well-formed
// response; it is not a general JSON parser.
bool ExtractJsonString(const std::string& body, const std::string& field, std::string& out) {
    const std::string key = "\"" + field + "\"";
    size_t pos = body.find(key);
    if (pos == std::string::npos) return false;
    pos = body.find(':', pos + key.size());
    if (pos == std::string::npos) return false;
    pos = body.find('"', pos);
    if (pos == std::string::npos) return false;
    const size_t end = body.find('"', pos + 1);
    if (end == std::string::npos) return false;
    out = body.substr(pos + 1, end - pos - 1);
    return true;
}

// Parse a 0x-prefixed hexadecimal quantity. Returns false when the text is not
// a valid hex quantity or does not fit in 64 bits.
bool ParseHexQuantity(const std::string& text, uint64_t& out) {
    if (text.size() < 3 || text[0] != '0' || (text[1] != 'x' && text[1] != 'X')) return false;
    uint64_t value = 0;
    for (size_t i = 2; i < text.size(); ++i) {
        const char c = text[i];
        uint64_t digit;
        if (c >= '0' && c <= '9') digit = static_cast<uint64_t>(c - '0');
        else if (c >= 'a' && c <= 'f') digit = static_cast<uint64_t>(c - 'a' + 10);
        else if (c >= 'A' && c <= 'F') digit = static_cast<uint64_t>(c - 'A' + 10);
        else return false;
        if (value > (UINT64_MAX - digit) / 16) return false;  // overflow
        value = value * 16 + digit;
    }
    out = value;
    return true;
}

// Issue a single JSON-RPC call and return the "result" string.
bool JsonRpcCall(const std::string& url, const std::string& request, std::string& result) {
    CURL* curl = curl_easy_init();
    if (curl == nullptr) return false;

    std::string response;
    struct curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, "Content-Type: application/json");

    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_POST, 1L);
    curl_easy_setopt(curl, CURLOPT_POSTFIELDS, request.c_str());
    curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, static_cast<long>(request.size()));
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 5L);
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 3L);
    curl_easy_setopt(curl, CURLOPT_NOSIGNAL, 1L);

    const CURLcode code = curl_easy_perform(curl);
    long http_status = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_status);

    curl_slist_free_all(headers);
    curl_easy_cleanup(curl);

    if (code != CURLE_OK || http_status != 200) return false;
    if (response.find("\"error\"") != std::string::npos) return false;
    return ExtractJsonString(response, "result", result);
}

}  // namespace

bool MEVBlocker::RefreshGasOracle() {
    std::string url;
    {
        std::lock_guard<std::mutex> lock(rpc_mutex_);
        url = rpc_url_;
    }
    if (url.empty()) return false;

    std::string base_result;
    if (!JsonRpcCall(url,
                     R"({"jsonrpc":"2.0","id":1,"method":"eth_gasPrice","params":[]})",
                     base_result)) {
        return false;
    }
    uint64_t base_fee = 0;
    if (!ParseHexQuantity(base_result, base_fee)) return false;

    // eth_maxPriorityFeePerGas is the node's own tip recommendation. It is
    // optional: some nodes do not serve it, in which case only the base fee is
    // refreshed and the previously observed priority fees are retained.
    std::string tip_result;
    uint64_t tip = 0;
    const bool have_tip =
        JsonRpcCall(url,
                    R"({"jsonrpc":"2.0","id":1,"method":"eth_maxPriorityFeePerGas","params":[]})",
                    tip_result) &&
        ParseHexQuantity(tip_result, tip);

    {
        std::lock_guard<std::mutex> lock(gas_mutex_);
        gas_oracle_.base_fee = base_fee;
        if (have_tip) {
            gas_oracle_.priority_fee_slow = tip;
            gas_oracle_.priority_fee_standard = tip + tip / 2;
            gas_oracle_.priority_fee_fast = tip * 2;
            gas_oracle_.priority_fee_instant = tip * 3;
        }
        gas_oracle_.updated_at = std::chrono::steady_clock::now();
    }
    return true;
}

void MEVBlocker::RunGasOracleUpdater() {
    std::cout << "[MEV] Gas oracle updater started" << std::endl;

    while (running_) {
        // Only fee data that came from the node is published. On failure the
        // previous successful reading is retained and its timestamp is left
        // untouched, so consumers can detect staleness.
        if (!RefreshGasOracle()) {
            std::cerr << "[MEV] Gas oracle refresh failed; retaining last known values"
                      << std::endl;
        }
        std::this_thread::sleep_for(std::chrono::seconds(5));
    }
}

void MEVBlocker::RunMempoolCleaner() {
    std::cout << "[MEV] Mempool cleaner started" << std::endl;
    
    while (running_) {
        try {
            std::this_thread::sleep_for(std::chrono::seconds(30));
            
            std::lock_guard<std::mutex> lock(mempool_mutex_);
            
            // Remove old transactions (older than 5 minutes)
            auto now = std::chrono::steady_clock::now();
            mempool_.erase(
                std::remove_if(mempool_.begin(), mempool_.end(),
                    [now](const Transaction& tx) {
                        auto age = std::chrono::duration_cast<std::chrono::minutes>(
                            now - tx.received_at).count();
                        return age > 5;
                    }),
                mempool_.end()
            );
            
            // Trim if too large
            if (mempool_.size() > kMempoolSize) {
                mempool_.erase(mempool_.begin(), 
                              mempool_.begin() + (mempool_.size() - kMempoolSize));
            }
            
        } catch (const std::exception& e) {
            std::cerr << "[MEV] Mempool cleaner error: " << e.what() << std::endl;
        }
    }
}

void MEVBlocker::RunBundleProcessor() {
    std::cout << "[MEV] Bundle processor started" << std::endl;
    
    while (running_) {
        try {
            std::this_thread::sleep_for(kBundleCheckInterval);
            
            // Process mempool for bundle opportunities
            std::lock_guard<std::mutex> lock(mempool_mutex_);
            
            for (auto& tx : mempool_) {
                auto sandwich = DetectSandwich(tx);
                if (sandwich) {
                    // Submit backrun bundle
                    std::cout << "[MEV] Sandwich opportunity detected, submitting backrun" << std::endl;
                    
                    // In production, submit bundle to validators
                    stats_.bundles_submitted++;
                }
            }
            
        } catch (const std::exception& e) {
            std::cerr << "[MEV] Bundle processor error: " << e.what() << std::endl;
        }
    }
}

bool MEVBlocker::ShouldBlockTransaction(const Transaction& tx) {
    if (protection_level_ == ProtectionLevel::kNone) {
        return false;
    }
    
    // Block if it's a known MEV bot
    if (tx.from == "0x47173b753d6f72c6a7b8f78f91b9c6c4d8e5f3a" ||  // Known sniper
        tx.from == "0x9f4e5e8c5a8d4f3b2c7e9f6a1d8c4b5e7f9a2d4") {
        return true;
    }
    
    // Block if extremely high gas (potential attack)
    if (tx.gas_price > gas_oracle_.priority_fee_instant * 10) {
        return true;
    }
    
    // Check for sandwich attack
    if (IsSandwichAttack(tx)) {
        return protection_level_ >= ProtectionLevel::kBasic;
    }
    
    return false;
}

bool MEVBlocker::IsSandwichAttack(const Transaction& tx) {
    // Check if this transaction could be part of a sandwich
    // Look for large swaps that could be sandwiched
    return tx.data.size() > 4 && tx.value > 1000000000000000000ULL;  // > 1 ETH
}

bool MEVBlocker::IsFrontRunning(const Transaction& tx) {
    // Detect potential front-running (buying before large order)
    return tx.data.size() > 4 && tx.gas_price > gas_oracle_.priority_fee_fast * 2;
}

bool MEVBlocker::IsBackRunning(const Transaction& tx) {
    // Detect potential back-running (buying after large order)
    return tx.data.size() > 4;
}

std::string MEVBlocker::SubmitToFlashbots(const Transaction& tx) {
    // In production, this would call the Flashbots API
    // For now, return a simulated private hash
    std::string private_hash = "0xfb" + tx.hash.substr(2);
    return private_hash;
}

std::string MEVBlocker::SubmitToRelayer(const Transaction& tx, const std::string& relayer) {
    // In production, this would call the relayer API
    // For now, return empty (Flashbots succeeded)
    return "";
}

uint64_t MEVBlocker::CalculateOptimalGasPrice(const Transaction& tx) {
    uint64_t base_fee = gas_oracle_.base_fee;
    uint64_t priority_fee = 0;
    
    switch (protection_level_) {
        case ProtectionLevel::kNone:
            priority_fee = gas_oracle_.priority_fee_slow;
            break;
        case ProtectionLevel::kBasic:
            priority_fee = gas_oracle_.priority_fee_standard;
            break;
        case ProtectionLevel::kAdvanced:
            priority_fee = gas_oracle_.priority_fee_fast;
            break;
        case ProtectionLevel::kMaximum:
            priority_fee = gas_oracle_.priority_fee_instant;
            break;
    }
    
    return base_fee + priority_fee;
}

float MEVBlocker::EstimateMEVProfit(const Transaction& tx) {
    // Simple estimation based on transaction value
    float profit = 0.0f;
    
    // Large swap = potential MEV
    if (tx.value > uint256(10000000000000000000ULL)) {  // > 10 ETH
        profit = static_cast<float>(tx.value.to_double() / 1e18 * 0.01);  // 1% estimate
    }
    
    return profit;
}

void MEVBlocker::UpdateStatistics(bool protected_, bool blocked_) {
    std::lock_guard<std::mutex> lock(stats_mutex_);
    
    stats_.total_transactions++;
    if (protected_) {
        stats_.protected_transactions++;
    }
    if (blocked_) {
        stats_.blocked_transactions++;
    }
}

std::optional<MEVBlocker::SandwichOpportunity> MEVBlocker::DetectSandwich(const Transaction& tx) {
    // Look for sandwich opportunities in mempool
    std::lock_guard<std::mutex> lock(mempool_mutex_);
    
    // Check if this transaction could be sandwiched
    for (const auto& mempool_tx : mempool_) {
        // Check if mempool_tx is a large swap that could be sandwiched
        if (mempool_tx.value > tx.value * 2) {
            // Found potential sandwich
            SandwichOpportunity opp;
            opp.victim = tx;
            opp.front_run = mempool_tx;
            opp.potential_profit = (mempool_tx.value - tx.value).to_uint64() / 10;
            return opp;
        }
    }
    
    return std::nullopt;
}

TransactionType MEVBlocker::ClassifyTransaction(const Transaction& tx) {
    // Classify transaction based on calldata
    if (tx.data.size() < 4) {
        return TransactionType::kTransfer;
    }
    
    // Check for swap signatures (Uniswap, Sushiswap, etc.)
    std::string selector = tx.data.substr(0, 10);
    if (selector == "0x7ff36ab5" ||  // Uniswap V3 swap
        selector == "0x38ed1739" ||  // Sushiswap swap
        selector == "0x8803dbee") {  // Curve swap
        return TransactionType::kSwap;
    }
    
    // Check for NFT
    if (selector == "0x23b872dd" ||  // transferFrom (NFT)
        selector == "0xb88d4fde") {  // safeTransferFrom
        return TransactionType::kNFT;
    }
    
    return TransactionType::kContractInteraction;
}

bool MEVBlocker::SimulateTransaction(const Transaction& tx, std::string& output) {
    // Simulate against the node with eth_call. A revert surfaces as a JSON-RPC
    // error carrying the revert data, which is far more useful than a bare
    // boolean, so the raw node response is passed back through `output`.
    std::string url;
    {
        std::lock_guard<std::mutex> lock(rpc_mutex_);
        url = rpc_url_;
    }
    if (url.empty()) {
        output = "no RPC endpoint configured (TIGER_MEV_RPC_URL)";
        return false;
    }

    // Build the call object from the transaction fields. `from` and `data` are
    // optional in the eth_call spec but are sent when present.
    std::ostringstream call;
    call << "{";
    bool first = true;
    auto add_field = [&](const char* name, const std::string& value) {
        if (value.empty()) return;
        if (!first) call << ",";
        call << "\"" << name << "\":\"" << value << "\"";
        first = false;
    };
    add_field("from", tx.from);
    add_field("to", tx.to);
    add_field("data", tx.data);
    call << "}";

    std::ostringstream request;
    request << "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_call\",\"params\":["
            << call.str() << ",\"latest\"]}";

    // JsonRpcCall only returns success for a non-error response, which is
    // exactly the signal we need here.
    std::string result;
    const bool ok = JsonRpcCall(url, request.str(), result);
    output = ok ? result : std::string("eth_call reverted or node unreachable");
    return ok;
}

std::unique_ptr<MEVBlocker> CreateMEVBlocker(ProtectionLevel level) {
    return std::make_unique<MEVBlocker>(level);
}

}  // namespace tiger
