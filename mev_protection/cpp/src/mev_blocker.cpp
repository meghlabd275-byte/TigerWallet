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

    // Every submitted transaction is recorded locally so the bundle processor
    // can look for sandwich exposure around it.
    RecordMempoolEntry(tx);

    // Analyze transaction for MEV risk
    RiskAssessment assessment = AnalyzeTransaction(tx);

    // Check if we should block this transaction
    if (ShouldBlockTransaction(tx)) {
        std::cout << "[MEV] Blocking potentially harmful transaction: "
                  << tx.hash << std::endl;
        UpdateStatistics(false, true);
        return "";
    }

    // At advanced and above the transaction is broadcast through a private
    // channel so it never appears in the public mempool.
    if (protection_level_ >= ProtectionLevel::kAdvanced) {
        std::string private_hash = SubmitPrivateTransaction(tx);
        if (!private_hash.empty()) {
            std::cout << "[MEV] Transaction protected (private): " << private_hash << std::endl;
            UpdateStatistics(true, false);
            return private_hash;
        }
        // The private channel could not accept the transaction. Falling back
        // to a public broadcast would defeat the protection the caller asked
        // for, so the submission fails instead.
        std::cerr << "[MEV] Private submission unavailable at protection level "
                  << static_cast<int>(protection_level_)
                  << "; refusing to broadcast publicly" << std::endl;
        UpdateStatistics(false, false);
        return "";
    }

    // Protection level is none or basic: broadcast the signed payload.
    const std::string broadcast = BroadcastRawTransaction(tx.raw);
    if (broadcast.empty()) {
        UpdateStatistics(false, false);
        return "";
    }

    UpdateStatistics(true, false);
    return broadcast;
}

std::string MEVBlocker::SubmitPrivateTransaction(const Transaction& tx) {
    // A private submission carries the signed raw transaction to a relay. An
    // empty payload cannot be submitted anywhere, so this fails closed rather
    // than returning a hash for something that was never sent.
    if (tx.raw.empty()) {
        std::cerr << "[MEV] Private submission requires a signed raw transaction" << std::endl;
        return "";
    }

    // Relayers are copied under the lock; the submission itself happens without
    // it so a slow relay cannot block relayer configuration.
    std::vector<std::string> relayers;
    {
        std::lock_guard<std::mutex> lock(relayers_mutex_);
        relayers = relayers_;
    }

    std::string result = SubmitToFlashbots(tx);
    if (!result.empty()) {
        std::lock_guard<std::mutex> lock(private_mutex_);
        private_txs_[result] = tx;
        return result;
    }

    // Try the operator's configured relayers in turn.
    for (const auto& relayer : relayers) {
        result = SubmitToRelayer(tx, relayer);
        if (!result.empty()) {
            std::lock_guard<std::mutex> lock(private_mutex_);
            private_txs_[result] = tx;
            return result;
        }
    }

    return "";
}

RiskAssessment MEVBlocker::AnalyzeTransaction(const Transaction& tx) {
    RiskAssessment result;
    const TransactionType tx_type = ClassifyTransaction(tx);

    // Only a value-bearing swap exposes a victim to sandwiching, so the
    // analysis is scoped to that case. Any other transaction is reported as
    // carrying no extractable MEV rather than being given a fabricated score.
    if (tx_type != TransactionType::kSwap || tx.value.is_zero()) {
        return result;
    }

    // The victim's own exposure: a swap large enough to move the pool is worth
    // front-running, and its value bounds what an attacker could take. This is
    // a stated heuristic, not a measurement of a specific attack.
    result.attack_type = "Sandwich exposure";
    result.risk_score = 0.6f;
    result.recommendations.push_back("Broadcast through a private channel");
    result.recommendations.push_back("Set slippage tolerance to the minimum");

    // Upper bound on extractable value, reported so callers can rank the
    // transaction. It is explicitly an upper bound: the realised extraction
    // depends on pool depth and competing searchers, which this module does
    // not observe.
    result.profit_extracted = EstimateMEVProfit(tx);
    result.is_profitable = result.profit_extracted > 0.0f;

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

bool MEVBlocker::AddBlockedSender(const std::string& address) {
    if (address.empty()) return false;

    std::lock_guard<std::mutex> lock(blocked_mutex_);
    return blocked_senders_.insert(address).second;
}

bool MEVBlocker::RemoveBlockedSender(const std::string& address) {
    std::lock_guard<std::mutex> lock(blocked_mutex_);
    return blocked_senders_.erase(address) > 0;
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

std::string MEVBlocker::BroadcastRawTransaction(const std::string& raw_tx) {
    // Broadcasting needs a signed payload and a node. Either missing means no
    // transaction hash exists, so the caller gets an empty string rather than a
    // locally invented one.
    if (raw_tx.empty()) return "";

    std::string url;
    {
        std::lock_guard<std::mutex> lock(rpc_mutex_);
        url = rpc_url_;
    }
    if (url.empty()) {
        std::cerr << "[MEV] No RPC endpoint configured (TIGER_MEV_RPC_URL); "
                     "cannot broadcast" << std::endl;
        return "";
    }

    std::ostringstream request;
    request << "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_sendRawTransaction\","
            << "\"params\":[\"" << raw_tx << "\"]}";

    std::string result;
    if (!JsonRpcCall(url, request.str(), result)) {
        return "";
    }
    return result;
}

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

            // Copy the candidate transactions under the lock, then analyse them
            // without it. DetectSandwich takes mempool_mutex_ itself, so
            // holding it across the call would deadlock on the non-recursive
            // mutex and permanently stall this worker.
            std::vector<Transaction> candidates;
            {
                std::lock_guard<std::mutex> lock(mempool_mutex_);
                candidates.assign(mempool_.begin(), mempool_.end());
            }

            for (const auto& tx : candidates) {
                if (DetectSandwich(tx)) {
                    // The opportunity is reported, never acted on. Turning it
                    // into a bundle would require the pool state, an unsigned
                    // attacker transaction, and a private key to sign with,
                    // none of which belong to a protection service.
                    std::lock_guard<std::mutex> lock(stats_mutex_);
                    stats_.sandwich_opportunities++;
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

    // An operator-supplied denylist of sender addresses. This is the only
    // deterministic block signal available: everything else here is a heuristic
    // and blocking on a heuristic would reject legitimate traffic.
    if (!tx.from.empty()) {
        std::lock_guard<std::mutex> lock(blocked_mutex_);
        if (blocked_senders_.count(tx.from) > 0) {
            return true;
        }
    }

    return false;
}

// The detectors below answer one narrow question: does this transaction carry
// the shape of the pattern in question? They inspect calldata and the fee the
// sender attached. They do not claim that an attack is occurring, because that
// would require pool state and ordering information this module does not have.
// Callers use them to decide whether to route a transaction privately.
bool MEVBlocker::IsSandwichAttack(const Transaction& tx) {
    return ClassifyTransaction(tx) == TransactionType::kSwap && !tx.value.is_zero();
}

bool MEVBlocker::IsFrontRunning(const Transaction& tx) {
    // A swap priced far above the current base fee is bidding for priority,
    // which is what a front-run does. The base fee comes from the node; with
    // no reading there is nothing to compare against, so this reports false
    // rather than guessing a threshold.
    if (ClassifyTransaction(tx) != TransactionType::kSwap) return false;

    uint64_t base_fee = 0;
    {
        std::lock_guard<std::mutex> lock(gas_mutex_);
        base_fee = gas_oracle_.base_fee;
    }
    if (base_fee == 0) return false;

    return tx.gas_price.to_double() > static_cast<double>(base_fee) * 2.0;
}

bool MEVBlocker::IsBackRunning(const Transaction& tx) {
    // A back-run is a swap sent immediately after a large swap. Without a
    // block-ordering view the only honest signal is that this is a swap at
    // all, which is what the caller needs to decide on private routing.
    return ClassifyTransaction(tx) == TransactionType::kSwap;
}

std::string MEVBlocker::SubmitToFlashbots(const Transaction& tx) {
    // Flashbots accepts an eth_sendBundle over its relay JSON-RPC endpoint. The
    // bundle holds the signed raw transaction. blockNumber is omitted, which
    // the relay reads as "the next block". A relay that rejects the bundle
    // produces an error and no hash, surfaced here as an empty result.
    if (tx.raw.empty()) return "";

    const std::string endpoint = flashbots_endpoint_.empty()
                                     ? std::string("https://relay.flashbots.net")
                                     : flashbots_endpoint_;

    std::ostringstream bundle;
    bundle << "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_sendBundle\",\"params\":[{"
           << "\"txs\":[\"" << tx.raw << "\"]"
           << "}]}";

    std::string result;
    if (!JsonRpcCall(endpoint, bundle.str(), result)) {
        return "";
    }
    return result;
}

std::string MEVBlocker::SubmitToRelayer(const Transaction& tx, const std::string& relayer) {
    // A relayer speaks the same eth_sendRawTransaction JSON-RPC method as a
    // node, so the signed payload is forwarded verbatim and the node's own
    // transaction hash is returned. Nothing is synthesised locally.
    if (tx.raw.empty() || relayer.empty()) return "";

    std::ostringstream request;
    request << "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_sendRawTransaction\","
            << "\"params\":[\"" << tx.raw << "\"]}";

    std::string result;
    if (!JsonRpcCall(relayer, request.str(), result)) {
        return "";
    }
    return result;
}

uint64_t MEVBlocker::CalculateOptimalGasPrice(const Transaction& tx) {
    // Snapshot the oracle under its lock; the worker thread updates these
    // fields concurrently.
    GasOracle oracle;
    {
        std::lock_guard<std::mutex> lock(gas_mutex_);
        oracle = gas_oracle_;
    }

    uint64_t base_fee = oracle.base_fee;
    uint64_t priority_fee = 0;

    switch (protection_level_) {
        case ProtectionLevel::kNone:
            priority_fee = oracle.priority_fee_slow;
            break;
        case ProtectionLevel::kBasic:
            priority_fee = oracle.priority_fee_standard;
            break;
        case ProtectionLevel::kAdvanced:
            priority_fee = oracle.priority_fee_fast;
            break;
        case ProtectionLevel::kMaximum:
            priority_fee = oracle.priority_fee_instant;
            break;
    }

    return base_fee + priority_fee;
}

float MEVBlocker::EstimateMEVProfit(const Transaction& tx) {
    // Upper bound on extractable value: a searcher cannot take more than the
    // victim's own swap is worth. The realised figure depends on pool depth and
    // competing searchers, which this module does not observe, so the bound is
    // reported as such rather than presented as a prediction.
    if (tx.value.is_zero()) return 0.0f;

    const double value_eth = tx.value.to_double() / 1e18;
    return static_cast<float>(value_eth);
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

void MEVBlocker::RecordMempoolEntry(const Transaction& tx) {
    std::lock_guard<std::mutex> lock(mempool_mutex_);
    mempool_.push_back(tx);
    if (mempool_.size() > kMempoolSize) {
        mempool_.erase(mempool_.begin());
    }
}

std::optional<MEVBlocker::SandwichOpportunity> MEVBlocker::DetectSandwich(const Transaction& tx) {
    // A sandwich needs a victim swap of its own. Without one there is nothing
    // to wrap, so this reports no opportunity rather than pairing the caller's
    // transaction with an unrelated mempool entry.
    if (ClassifyTransaction(tx) != TransactionType::kSwap || tx.value.is_zero()) {
        return std::nullopt;
    }

    // Look for a larger swap already in the mempool that this transaction would
    // be sandwiched against. Pool state is not available here, so the ordering
    // is reported as an observation for the caller to act on, never as an
    // executed bundle.
    std::lock_guard<std::mutex> lock(mempool_mutex_);
    for (const auto& mempool_tx : mempool_) {
        if (mempool_tx.hash == tx.hash) continue;
        if (ClassifyTransaction(mempool_tx) != TransactionType::kSwap) continue;
        if (mempool_tx.value.is_zero()) continue;

        // The counterpart has to be the larger leg for the pair to be worth
        // wrapping.
        if (!(mempool_tx.value > tx.value)) continue;

        SandwichOpportunity opp;
        opp.victim = tx;
        opp.front_run = mempool_tx;
        // Difference in wei between the two legs, which bounds the value the
        // pair exposes. Reported as an upper bound, not a realised profit.
        opp.potential_profit = mempool_tx.value.to_uint64() - tx.value.to_uint64();
        return opp;
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
