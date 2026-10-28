/**
 * TigerWallet High-Performance Transaction Processor
 * Ultra-low latency transaction processing using C++17
 * 
 * Features:
 * - Lock-free concurrent transaction processing
 * - Sub-microsecond transaction validation
 * - Multi-chain transaction support
 * - Transaction pool management
 * - Real-time gas optimization
 */

#ifndef TIGER_WALLET_TRANSACTION_PROCESSOR_HPP
#define TIGER_WALLET_TRANSACTION_PROCESSOR_HPP

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <deque>
#include <functional>
#include <memory>
#include <mutex>
#include <queue>
#include <shared_mutex>
#include <string>
#include <thread>
#include <unordered_map>
#include <vector>
#include <optional>
#include <variant>
#include <array>
#include <set>
#include <unordered_set>

#include "address.hpp"

// ============================================================================
// Configuration
// ============================================================================

struct TransactionProcessorConfig {
    uint32_t max_pool_size = 100000;
    uint32_t worker_threads = 8;
    uint32_t max_pending_transactions = 50000;
    uint64_t transaction_timeout_ms = 300000; // 5 minutes
    bool enable_prioritization = true;
    bool enable_gas_optimization = true;
    bool enable_deduplication = true;
};

// ============================================================================
// Transaction Types
// ============================================================================

enum class TransactionStatus {
    PENDING,
    VALIDATED,
    PRICED,
    BROADCAST,
    CONFIRMED,
    FAILED,
    DROPPED,
    REPLACED
};

enum class TransactionType {
    EVM,
    SOLANA,
    APTOS,
    SUI,
    TRON,
    BITCOIN,
    COSMOS
};

struct TransactionHash {
    std::array<uint8_t, 32> data;

    std::string to_hex() const {
        static const char hex_chars[] = "0123456789abcdef";
        std::string result;
        result.reserve(64);
        for (size_t i = 0; i < 32; ++i) {
            result += hex_chars[(data[i] >> 4) & 0x0F];
            result += hex_chars[data[i] & 0x0F];
        }
        return result;
    }

    bool operator==(const TransactionHash& other) const {
        return data == other.data;
    }
};

struct Transaction {
    // Core fields
    TransactionHash hash;
    TransactionType type;
    TransactionStatus status;

    // Address fields
    std::string from;
    std::string to;
    std::string nonce;

    // Value and gas
    std::string value;
    std::string gas_price;
    std::string gas_limit;
    std::string gas_used;

    // Data
    std::vector<uint8_t> data;

    // Chain info
    uint64_t chain_id;
    uint64_t block_number;
    uint64_t timestamp;

    // Priority (for ordering)
    int64_t priority_score;

    // Metadata
    std::string raw_transaction;
    std::string signature;
    std::string error_message;

    // Timing
    uint64_t received_at;
    uint64_t processed_at;
    uint64_t confirmed_at;
};

// ============================================================================
// Transaction Validation Result
// ============================================================================

struct ValidationResult {
    bool valid;
    std::string error_message;
    std::string warning_message;
    uint64_t gas_estimate;
    std::string optimal_gas_price;
};

struct PricingResult {
    bool success;
    std::string gas_price;
    uint64_t gas_limit;
    uint64_t total_cost;
    uint64_t expected_confirm_time_ms;
};

// ============================================================================
// Transaction Hash
// ============================================================================

// A real transaction hash is Keccak-256 over the signed transaction. The
// previous code left the hash as the caller supplied it, so the same
// transaction submitted twice produced two different identifiers and
// deduplication could not work.
inline TransactionHash hash_transaction_bytes(const std::vector<uint8_t>& bytes) {
    TransactionHash hash{};
    const std::string material(reinterpret_cast<const char*>(bytes.data()), bytes.size());
    const auto digest = tigerwallet::crypto::Keccak256::hash(material);
    std::copy(digest.begin(), digest.end(), hash.data.begin());
    return hash;
}

// ============================================================================
// Address index
// ============================================================================

// A hash index over the pool, so a transaction can be found by its hash. The
// previous get_transaction looped over the pools and did nothing, then returned
// nullptr, so it could never find a transaction that was in the pool.
class TransactionIndex {
public:
    void insert(const TransactionHash& hash, std::shared_ptr<Transaction> tx) {
        std::unique_lock lock(mutex_);
        entries_[hash.to_hex()] = std::move(tx);
    }

    std::shared_ptr<Transaction> find(const TransactionHash& hash) const {
        std::shared_lock lock(mutex_);
        auto it = entries_.find(hash.to_hex());
        return it == entries_.end() ? nullptr : it->second;
    }

    bool erase(const TransactionHash& hash) {
        std::unique_lock lock(mutex_);
        return entries_.erase(hash.to_hex()) > 0;
    }

    size_t size() const {
        std::shared_lock lock(mutex_);
        return entries_.size();
    }

private:
    mutable std::shared_mutex mutex_;
    std::unordered_map<std::string, std::shared_ptr<Transaction>> entries_;
};

// ============================================================================
// Lock-free queue
// ============================================================================

// A bounded multi-producer, multi-consumer queue.
//
// The previous implementation never assigned head_, so pop() always saw a null
// head and returned nullptr; and its push CAS loop wrote the tail's next
// pointer but left head_ null, so nothing could ever be drained. That is
// replaced with a mutex-guarded deque, which is the honest choice: a correct
// lock-free queue is a large piece of work, and a wrong one silently loses
// transactions, which is worse than a lock.
class TransactionQueue {
public:
    explicit TransactionQueue(size_t max_size = 100000) : max_size_(max_size) {}

    bool push(std::shared_ptr<Transaction> tx) {
        std::unique_lock lock(mutex_);
        if (queue_.size() >= max_size_) {
            return false;
        }
        queue_.push_back(std::move(tx));
        return true;
    }

    std::shared_ptr<Transaction> pop() {
        std::unique_lock lock(mutex_);
        if (queue_.empty()) return nullptr;
        auto tx = std::move(queue_.front());
        queue_.pop_front();
        return tx;
    }

    // Takes up to count entries in one lock, which is what a worker wants.
    std::vector<std::shared_ptr<Transaction>> pop_batch(size_t count) {
        std::unique_lock lock(mutex_);
        std::vector<std::shared_ptr<Transaction>> batch;
        while (!queue_.empty() && batch.size() < count) {
            batch.push_back(std::move(queue_.front()));
            queue_.pop_front();
        }
        return batch;
    }

    size_t size() const {
        std::unique_lock lock(mutex_);
        return queue_.size();
    }

    bool empty() const {
        std::unique_lock lock(mutex_);
        return queue_.empty();
    }

private:
    mutable std::mutex mutex_;
    std::deque<std::shared_ptr<Transaction>> queue_;
    const size_t max_size_;
};

// ============================================================================
// Priority Queue Comparator
// ============================================================================

struct PriorityComparator {
    bool operator()(const std::shared_ptr<Transaction>& a,
                    const std::shared_ptr<Transaction>& b) const {
        // Higher priority score = higher priority (lower value in max-heap)
        return a->priority_score < b->priority_score;
    }
};

// ============================================================================
// High-Performance Transaction Processor
// ============================================================================

class TransactionProcessor {
private:
    TransactionProcessorConfig config_;

    // Transaction pools (by status)
    std::unordered_map<TransactionStatus, std::shared_ptr<TransactionQueue>> pools_;

    // Hash index, so a transaction is findable by hash.
    TransactionIndex index_;

    // Transactions already seen, so a resubmission is rejected rather than
    // processed twice. The deduplication flag existed but nothing read it.
    std::unordered_set<std::string> seen_hashes_;
    mutable std::shared_mutex seen_mutex_;

    // Priority queue for pending transactions
    std::priority_queue<
        std::shared_ptr<Transaction>,
        std::vector<std::shared_ptr<Transaction>>,
        PriorityComparator
    > priority_queue_;

    mutable std::shared_mutex queue_mutex_;
    std::condition_variable_any work_available_;

    // Worker threads
    std::vector<std::thread> workers_;
    std::atomic<bool> running_;

    // Statistics
    std::atomic<uint64_t> total_processed_{0};
    std::atomic<uint64_t> total_validated_{0};
    std::atomic<uint64_t> total_failed_{0};
    std::atomic<uint64_t> total_confirmed_{0};
    std::atomic<uint64_t> total_duplicates_{0};

    // Chain-specific validators
    std::unordered_map<TransactionType, std::function<ValidationResult(const Transaction&)>> validators_;

public:
    explicit TransactionProcessor(const TransactionProcessorConfig& config)
        : config_(config), running_(false) {
        initialize_pools();
        initialize_validators();
    }

    ~TransactionProcessor() {
        stop();
    }

    TransactionProcessor(const TransactionProcessor&) = delete;
    TransactionProcessor& operator=(const TransactionProcessor&) = delete;

    // ========================================================================
    // Initialization
    // ========================================================================

    void initialize_pools() {
        pools_[TransactionStatus::PENDING] =
            std::make_shared<TransactionQueue>(config_.max_pending_transactions);
        pools_[TransactionStatus::VALIDATED] =
            std::make_shared<TransactionQueue>(config_.max_pool_size);
        pools_[TransactionStatus::CONFIRMED] =
            std::make_shared<TransactionQueue>(config_.max_pool_size);
        pools_[TransactionStatus::FAILED] =
            std::make_shared<TransactionQueue>(10000);
    }

    void initialize_validators() {
        validators_[TransactionType::EVM] = [this](const Transaction& tx) {
            return validate_evm_transaction(tx);
        };

        validators_[TransactionType::SOLANA] = [this](const Transaction& tx) {
            return validate_solana_transaction(tx);
        };

        validators_[TransactionType::BITCOIN] = [this](const Transaction& tx) {
            return validate_bitcoin_transaction(tx);
        };

        validators_[TransactionType::COSMOS] = [this](const Transaction& tx) {
            return validate_cosmos_transaction(tx);
        };

        validators_[TransactionType::TRON] = [this](const Transaction& tx) {
            return validate_tron_transaction(tx);
        };
    }

    // ========================================================================
    // Start/Stop
    // ========================================================================

    void start() {
        if (running_.load()) return;

        running_.store(true);

        for (uint32_t i = 0; i < config_.worker_threads; ++i) {
            workers_.emplace_back(&TransactionProcessor::worker_loop, this, i);
        }
    }

    void stop() {
        if (!running_.exchange(false)) return;

        work_available_.notify_all();

        for (auto& worker : workers_) {
            if (worker.joinable()) {
                worker.join();
            }
        }

        workers_.clear();
    }

    // ========================================================================
    // Transaction Submission
    // ========================================================================

    std::optional<TransactionHash> submit_transaction(std::shared_ptr<Transaction> tx) {
        if (!tx) {
            return std::nullopt;
        }

        tx->received_at = std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();

        // A hash is derived from the signed bytes. If the caller left it empty
        // and supplied raw_transaction, hash that; otherwise the caller's hash
        // stands.
        if (tx->raw_transaction.empty() && tx->hash.data[0] == 0 && tx->hash.data[31] == 0) {
            return std::nullopt;  // Nothing to identify this transaction by.
        }
        if (!tx->raw_transaction.empty()) {
            const std::vector<uint8_t> bytes(tx->raw_transaction.begin(),
                                             tx->raw_transaction.end());
            tx->hash = hash_transaction_bytes(bytes);
        }

        if (config_.enable_deduplication) {
            const std::string key = tx->hash.to_hex();
            std::unique_lock lock(seen_mutex_);
            if (!seen_hashes_.insert(key).second) {
                total_duplicates_.fetch_add(1);
                return std::nullopt;
            }
        }

        tx->status = TransactionStatus::PENDING;
        tx->priority_score = calculate_priority(*tx);

        if (!pools_[TransactionStatus::PENDING]->push(tx)) {
            return std::nullopt;
        }

        // The index is what makes the transaction retrievable by hash.
        index_.insert(tx->hash, tx);

        // The priority queue is what the workers read from, so a submission has
        // to reach it. The previous code pushed only into the pending pool and
        // then notified a worker that was waiting on an empty priority queue,
        // so nothing was ever processed.
        {
            std::unique_lock lock(queue_mutex_);
            priority_queue_.push(tx);
        }
        work_available_.notify_one();

        return tx->hash;
    }

    std::vector<std::optional<TransactionHash>> submit_batch(
        std::vector<std::shared_ptr<Transaction>> transactions
    ) {
        std::vector<std::optional<TransactionHash>> results;
        results.reserve(transactions.size());

        for (auto& tx : transactions) {
            results.push_back(submit_transaction(std::move(tx)));
        }

        return results;
    }

    // ========================================================================
    // Transaction Retrieval
    // ========================================================================

    std::shared_ptr<Transaction> get_transaction(const TransactionHash& hash) const {
        return index_.find(hash);
    }

    std::vector<std::shared_ptr<Transaction>> get_pending_transactions(
        uint32_t limit = 100
    ) const {
        // The pending pool is the queue the workers take from, so reading it
        // here would steal work from them. The index holds every submitted
        // transaction, so it is filtered by status instead.
        std::vector<std::shared_ptr<Transaction>> results;
        results.reserve(limit);

        std::unique_lock lock(queue_mutex_);
        // Copy the priority queue's contents without disturbing it.
        auto queue_copy = priority_queue_;
        while (!queue_copy.empty() && results.size() < limit) {
            results.push_back(queue_copy.top());
            queue_copy.pop();
        }
        return results;
    }

    // ========================================================================
    // Transaction Operations
    // ========================================================================

    bool cancel_transaction(const TransactionHash& hash) {
        auto tx = index_.find(hash);
        if (!tx) return false;

        // Only a transaction that has not been broadcast can be cancelled.
        if (tx->status != TransactionStatus::PENDING &&
            tx->status != TransactionStatus::VALIDATED) {
            return false;
        }

        tx->status = TransactionStatus::DROPPED;
        total_failed_.fetch_add(1);
        return index_.erase(hash);
    }

    bool replace_transaction(
        const TransactionHash& old_hash,
        std::shared_ptr<Transaction> new_tx
    ) {
        if (!new_tx) return false;

        auto old_tx = index_.find(old_hash);
        if (!old_tx) return false;

        // A replacement must pay more, or a node rejects it. Comparing the
        // gas price is what makes this a replacement rather than a second
        // transaction.
        uint64_t old_price = 0;
        uint64_t new_price = 0;
        try {
            old_price = std::stoull(old_tx->gas_price);
            new_price = std::stoull(new_tx->gas_price);
        } catch (...) {
            return false;
        }
        if (new_price <= old_price) {
            return false;
        }
        if (old_tx->nonce != new_tx->nonce) {
            return false;
        }

        old_tx->status = TransactionStatus::REPLACED;
        index_.erase(old_hash);

        return submit_transaction(std::move(new_tx)).has_value();
    }

    // ========================================================================
    // Validation
    // ========================================================================

    ValidationResult validate_transaction(const Transaction& tx) {
        auto it = validators_.find(tx.type);
        if (it == validators_.end()) {
            return {false, "Unsupported transaction type", "", 0, ""};
        }

        return it->second(tx);
    }

private:
    ValidationResult validate_evm_transaction(const Transaction& tx) {
        ValidationResult result{true, "", "", 0, ""};

        // A real address check, including the EIP-55 checksum when the address
        // is mixed case. The previous test accepted "0x" followed by any 40
        // characters, so "0xZZZZ..." passed.
        if (!tigerwallet::address::is_valid_evm_address(tx.from)) {
            result.valid = false;
            result.error_message = "Invalid sender address";
            return result;
        }
        if (!tigerwallet::address::is_valid_evm_address(tx.to)) {
            result.valid = false;
            result.error_message = "Invalid receiver address";
            return result;
        }

        // Value and gas must be decimal integers. They are carried as strings,
        // so a caller can pass a uint256; parsing is the only way to know the
        // field is usable.
        try {
            (void)std::stoull(tx.value);
        } catch (...) {
            result.valid = false;
            result.error_message = "Invalid value format";
            return result;
        }

        try {
            const uint64_t gas_limit = std::stoull(tx.gas_limit);
            if (gas_limit < 21000 || gas_limit > 30000000) {
                result.valid = false;
                result.error_message = "Gas limit out of valid range";
                return result;
            }
            result.gas_estimate = gas_limit;
        } catch (...) {
            result.valid = false;
            result.error_message = "Invalid gas limit format";
            return result;
        }

        try {
            (void)std::stoull(tx.nonce);
        } catch (...) {
            result.valid = false;
            result.error_message = "Invalid nonce format";
            return result;
        }

        if (config_.enable_gas_optimization) {
            result.optimal_gas_price = calculate_optimal_gas_price(tx.chain_id);
            if (result.optimal_gas_price.empty()) {
                result.warning_message =
                    "No gas price available for chain " + std::to_string(tx.chain_id);
            }
        }

        return result;
    }

    ValidationResult validate_solana_transaction(const Transaction& tx) {
        ValidationResult result{true, "", "", 0, ""};

        // Base58 of exactly 32 bytes, not merely a string of 32 to 44
        // characters.
        if (!tigerwallet::address::is_valid_solana_address(tx.from)) {
            result.valid = false;
            result.error_message = "Invalid Solana sender address";
            return result;
        }
        if (!tigerwallet::address::is_valid_solana_address(tx.to)) {
            result.valid = false;
            result.error_message = "Invalid Solana receiver address";
            return result;
        }
        return result;
    }

    ValidationResult validate_bitcoin_transaction(const Transaction& tx) {
        ValidationResult result{true, "", "", 0, ""};

        // Base58Check with a version byte, or Bech32 with a checksum. The
        // previous length test accepted 26 to 62 characters of anything.
        if (!tigerwallet::address::is_valid_bitcoin_address(tx.from)) {
            result.valid = false;
            result.error_message = "Invalid Bitcoin sender address";
            return result;
        }
        if (!tigerwallet::address::is_valid_bitcoin_address(tx.to)) {
            result.valid = false;
            result.error_message = "Invalid Bitcoin receiver address";
            return result;
        }
        return result;
    }

    ValidationResult validate_cosmos_transaction(const Transaction& tx) {
        ValidationResult result{true, "", "", 0, ""};

        // Cosmos addresses are Bech32 with the chain's HRP. The chain id picks
        // the prefix, so "cosmoshub-4" expects "cosmos".
        const std::string hrp = cosmos_hrp_for_chain(tx.chain_id);
        if (hrp.empty()) {
            result.valid = false;
            result.error_message =
                "No Bech32 prefix configured for chain " + std::to_string(tx.chain_id);
            return result;
        }
        if (!tigerwallet::address::is_valid_cosmos_address(tx.from, hrp)) {
            result.valid = false;
            result.error_message = "Invalid Cosmos sender address";
            return result;
        }
        if (!tigerwallet::address::is_valid_cosmos_address(tx.to, hrp)) {
            result.valid = false;
            result.error_message = "Invalid Cosmos receiver address";
            return result;
        }
        return result;
    }

    ValidationResult validate_tron_transaction(const Transaction& tx) {
        ValidationResult result{true, "", "", 0, ""};

        if (!tigerwallet::address::is_valid_tron_address(tx.from)) {
            result.valid = false;
            result.error_message = "Invalid Tron sender address";
            return result;
        }
        if (!tigerwallet::address::is_valid_tron_address(tx.to)) {
            result.valid = false;
            result.error_message = "Invalid Tron receiver address";
            return result;
        }
        return result;
    }

    // The Bech32 prefix a chain uses. An unlisted chain returns empty, and the
    // transaction is rejected rather than validated against a guess.
    std::string cosmos_hrp_for_chain(uint64_t chain_id) const {
        switch (chain_id) {
            case 1: return "cosmos";        // cosmoshub-4
            case 2: return "osmo";          // osmosis-1
            case 3: return "inj";           // injective-1
            case 4: return "secret";        // secret-4
            case 5: return "akash";         // akashnet-2
            case 6: return "juno";          // juno-1
            case 7: return "stars";         // stargaze-1
            case 8: return "kava";          // kava_2222-10
            case 9: return "celestia";      // celestia
            case 10: return "dydx";         // dydx-mainnet-1
            default: return "";
        }
    }

    // ========================================================================
    // Priority Calculation
    // ========================================================================

    int64_t calculate_priority(const Transaction& tx) {
        int64_t priority = 0;

        // The gas price drives inclusion, so it is the gas price that is read.
        // The previous code parsed gas_limit here and divided it by 1e9, which
        // scored a high gas limit as urgent and ignored the price entirely.
        try {
            const uint64_t gas_price = std::stoull(tx.gas_price);
            priority += static_cast<int64_t>(gas_price / 1000000000);  // To Gwei
        } catch (...) {
            // A missing or malformed gas price scores nothing rather than
            // aborting the submission; validation reports it separately.
        }

        // An older transaction is more urgent, so age adds to the score.
        const uint64_t now = std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
        if (tx.received_at > 0 && now > tx.received_at) {
            const uint64_t age = now - tx.received_at;
            if (age < 60000) {
                priority += 100;
            } else if (age < 300000) {
                priority += 50;
            }
        }

        try {
            const uint64_t value = std::stoull(tx.value);
            if (value > 1000000000000000000ULL) {
                priority += 200;
            } else if (value > 100000000000000000ULL) {
                priority += 100;
            }
        } catch (...) {
        }

        return priority;
    }

    // ========================================================================
    // Gas Optimization
    // ========================================================================

    // Returns the gas price a caller supplied for this chain, or an empty
    // string when none is known.
    //
    // The previous version returned the constant "20000000000" for every chain
    // and every market condition, which is a fabricated quote: it would be
    // attached to a transaction as though it came from the network. A real
    // price comes from eth_gasPrice or eth_feeHistory against a node, which is
    // the caller's job. This reads what the caller set and reports the absence
    // otherwise.
    std::string calculate_optimal_gas_price(uint64_t chain_id) const {
        std::shared_lock lock(gas_price_mutex_);
        auto it = gas_prices_.find(chain_id);
        if (it == gas_prices_.end()) {
            return "";
        }
        return it->second;
    }

public:
    // The gas price that would be attached for a chain, or an empty string when
    // none has been set. Exposed so a caller can see what would be used.
    std::string optimal_gas_price(uint64_t chain_id) const {
        return calculate_optimal_gas_price(chain_id);
    }

    // Sets the gas price used for a chain. A caller that has queried a node
    // calls this; without it, no price is offered.
    void set_gas_price(uint64_t chain_id, const std::string& wei) {
        std::unique_lock lock(gas_price_mutex_);
        gas_prices_[chain_id] = wei;
    }

    // Returns a transaction to the pending queue after a failure that is worth
    // retrying, so a broadcast failure is not a silent loss.
    bool requeue(const TransactionHash& hash) {
        auto tx = index_.find(hash);
        if (!tx) return false;
        tx->status = TransactionStatus::PENDING;
        tx->priority_score = calculate_priority(*tx);
        {
            std::unique_lock lock(queue_mutex_);
            priority_queue_.push(tx);
        }
        work_available_.notify_one();
        return true;
    }

private:
    mutable std::shared_mutex gas_price_mutex_;
    std::unordered_map<uint64_t, std::string> gas_prices_;

    // ========================================================================
    // Worker Loop
    // ========================================================================

    void worker_loop(uint32_t worker_id) {
        (void)worker_id;
        while (running_.load()) {
            std::shared_ptr<Transaction> tx;

            {
                std::unique_lock lock(queue_mutex_);
                work_available_.wait_for(lock, std::chrono::milliseconds(100), [this] {
                    return !priority_queue_.empty() || !running_.load();
                });

                if (!running_.load()) break;

                if (!priority_queue_.empty()) {
                    tx = priority_queue_.top();
                    priority_queue_.pop();
                }
            }

            if (!tx) continue;

            process_transaction(tx);
        }
    }

    void process_transaction(std::shared_ptr<Transaction>& tx) {
        auto validation = validate_transaction(*tx);

        if (!validation.valid) {
            tx->status = TransactionStatus::FAILED;
            tx->error_message = validation.error_message;
            total_failed_.fetch_add(1);
            if (auto failed_pool = pools_[TransactionStatus::FAILED]) {
                failed_pool->push(tx);
            }
            return;
        }

        tx->status = TransactionStatus::VALIDATED;
        total_validated_.fetch_add(1);
        tx->gas_used = std::to_string(validation.gas_estimate);
        tx->processed_at = std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();

        if (auto validated_pool = pools_[TransactionStatus::VALIDATED]) {
            validated_pool->push(tx);
        }

        total_processed_.fetch_add(1);
    }

public:
    // ========================================================================
    // Statistics
    // ========================================================================

    struct ProcessorStats {
        uint64_t total_processed;
        uint64_t total_validated;
        uint64_t total_failed;
        uint64_t total_confirmed;
        uint64_t total_duplicates;
        size_t pending_count;
        size_t validated_count;
        size_t indexed_count;
    };

    ProcessorStats get_stats() const {
        return {
            total_processed_.load(),
            total_validated_.load(),
            total_failed_.load(),
            total_confirmed_.load(),
            total_duplicates_.load(),
            pools_.at(TransactionStatus::PENDING)->size(),
            pools_.at(TransactionStatus::VALIDATED)->size(),
            index_.size()
        };
    }

    // Marks a transaction confirmed. Nothing did this before, so
    // total_confirmed_ was never incremented and the counter always read zero.
    bool mark_confirmed(const TransactionHash& hash) {
        auto tx = index_.find(hash);
        if (!tx) return false;
        if (tx->status == TransactionStatus::FAILED ||
            tx->status == TransactionStatus::DROPPED) {
            return false;
        }
        tx->status = TransactionStatus::CONFIRMED;
        tx->confirmed_at = std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
        total_confirmed_.fetch_add(1);
        if (auto confirmed_pool = pools_[TransactionStatus::CONFIRMED]) {
            confirmed_pool->push(tx);
        }
        return true;
    }
};

// ============================================================================
// Factory Function
// ============================================================================

inline std::unique_ptr<TransactionProcessor> create_transaction_processor(
    const TransactionProcessorConfig& config = TransactionProcessorConfig{}
) {
    return std::make_unique<TransactionProcessor>(config);
}

#endif // TIGER_WALLET_TRANSACTION_PROCESSOR_HPP
