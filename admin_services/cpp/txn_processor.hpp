/**
 * TigerWallet Ultra-Low Latency Transaction Processor
 * C++ High-Performance Transaction Processing Engine
 */

#ifndef TIGER_WALLET_TXN_PROCESSOR_HPP
#define TIGER_WALLET_TXN_PROCESSOR_HPP

#include <atomic>
#include <chrono>
#include <cstdint>
#include <memory>
#include <string>
#include <vector>
#include <unordered_map>
#include <queue>
#include <mutex>
#include <thread>
#include <optional>
#include <array>
#include <functional>
#include <span>

#if defined(__linux__)
    #define likely(x)       __builtin_expect(!!(x), 1)
    #define unlikely(x)     __builtin_expect(!!(x), 0)
    #define hot             __attribute__((hot))
#else
    #define likely(x)       (x)
    #define unlikely(x)     (x)
    #define hot
#endif

namespace tigerwallet {
namespace txn {

enum class TransactionType : uint8_t {
    TRANSFER = 0, SWAP = 1, STAKE = 2, UNSTAKE = 3,
    MINT = 4, BURN = 5, APPROVE = 6, TRANSFER_FROM = 7,
    BRIDGE = 8, NFT_TRANSFER = 9, UNKNOWN = 255
};

enum class TransactionStatus : uint8_t {
    PENDING = 0, CONFIRMED = 1, FAILED = 2, FLAGGED = 3, CANCELLED = 4
};

enum class Chain : uint8_t {
    ETHEREUM = 0, POLYGON = 1, BSC = 2, AVALANCHE = 3,
    SOLANA = 4, ARBITRUM = 5, OPTIMISM = 6, BASE = 7, BITCOIN = 8
};

struct Address {
    std::array<uint8_t, 32> data{};
    Address() = default;
    explicit Address(const std::string& hex);
    void fromHex(const std::string& hex);
    std::string toHex() const;
    bool isZero() const;
};

struct TxHash {
    std::array<uint8_t, 32> data{};
    TxHash() = default;
    explicit TxHash(const std::string& hex);
    void fromHex(const std::string& hex);
    std::string toHex() const;
    struct hash { size_t operator()(const TxHash& h) const; };
};

struct HighPrecisionTimestamp {
    uint64_t nanoseconds;
    static HighPrecisionTimestamp now() {
        auto now = std::chrono::steady_clock::now().time_since_epoch().count();
        return {static_cast<uint64_t>(now)};
    }
    uint64_t toMicroseconds() const { return nanoseconds / 1000; }
};

class uint256_t {
private:
    std::array<uint64_t, 4> words_;
public:
    uint256_t() : words_{0, 0, 0, 0} {}
    explicit uint256_t(uint64_t val) : words_{val, 0, 0, 0} {}
    uint256_t(uint64_t w0, uint64_t w1, uint64_t w2, uint64_t w3) : words_{w0, w1, w2, w3} {}
    
    uint256_t operator+(const uint256_t& o) const;
    uint256_t operator-(const uint256_t& o) const;
    uint256_t operator*(const uint256_t& o) const;
    bool operator==(const uint256_t& o) const;
    bool operator!=(const uint256_t& o) const;
    bool operator<(const uint256_t& o) const;
    uint64_t low64() const { return words_[0]; }
    std::string toString() const;
};

struct Transaction {
    TxHash hash;
    Address from;
    Address to;
    uint256_t amount;
    uint256_t fee;
    Chain chain;
    TransactionType type;
    TransactionStatus status;
    uint64_t nonce;
    uint64_t block_number;
    HighPrecisionTimestamp created_at;
    HighPrecisionTimestamp processed_at;
    HighPrecisionTimestamp confirmed_at;
    uint64_t gas_limit;
    uint64_t gas_used;
    uint64_t gas_price;
    std::vector<uint8_t> data;
    std::string memo;
    std::atomic<bool> verified;
    std::atomic<bool> processed;
    
    Transaction() : amount(0), fee(0), type(TransactionType::UNKNOWN), status(TransactionStatus::PENDING),
        nonce(0), block_number(0), gas_limit(0), gas_used(0), gas_price(0), verified(false), processed(false) {}
};

class TransactionPool {
private:
    struct PendingTransaction {
        Transaction txn;
        std::atomic<uint64_t> priority;
        std::atomic<uint64_t> timestamp;
    };
    std::vector<PendingTransaction> pending_;
    std::unordered_map<TxHash, size_t, TxHash::hash> index_;
    std::mutex mutex_;
    static constexpr size_t MAX_POOL_SIZE = 100000;
public:
    TransactionPool();
    ~TransactionPool();
    bool add(const Transaction& txn);
    bool remove(const TxHash& hash);
    std::optional<Transaction> getNext();
    std::optional<Transaction> get(const TxHash& hash);
    size_t size() const { return pending_.size(); }
    size_t capacity() const { return MAX_POOL_SIZE; }
    void clear();
    std::vector<Transaction> getByAddress(const Address& addr);
};

struct SignatureResult {
    bool valid;
    Address signer;
    std::string error;
};

class SignatureVerifier {
public:
    virtual ~SignatureVerifier() = default;
    virtual SignatureResult verify(const Transaction& txn, const std::vector<uint8_t>& signature) = 0;
    virtual bool batchVerify(const std::vector<Transaction>& txns, std::vector<uint8_t>& results) = 0;
};

class EVMSignatureVerifier : public SignatureVerifier {
public:
    SignatureResult verify(const Transaction& txn, const std::vector<uint8_t>& signature) override;
    bool batchVerify(const std::vector<Transaction>& txns, std::vector<uint8_t>& results) override;
private:
    Address recoverSigner(const std::vector<uint8_t>& msg, const std::vector<uint8_t>& sig);
};

struct ProcessorConfig {
    uint32_t num_workers;
    uint32_t queue_size;
    uint64_t max_gas_price;
    uint64_t min_gas_price;
    uint64_t block_time_ms;
    bool enable_verification;
    bool enable_deduplication;
    std::vector<Chain> supported_chains;
};

struct ProcessingResult {
    bool success;
    TransactionStatus status;
    std::string error;
    HighPrecisionTimestamp processed_at;
    uint64_t gas_used;
};

class TransactionProcessor {
private:
    ProcessorConfig config_;
    std::unique_ptr<TransactionPool> pool_;
    std::unique_ptr<SignatureVerifier> verifier_;
    std::vector<std::thread> workers_;
    std::atomic<bool> running_;
    std::atomic<uint64_t> processed_count_;
    std::atomic<uint64_t> failed_count_;
    
    struct Stats {
        std::atomic<uint64_t> total_processed;
        std::atomic<uint64_t> total_failed;
        std::atomic<uint64_t> total_gas_used;
        std::atomic<uint64_t> avg_latency_ns;
        std::atomic<uint64_t> max_latency_ns;
        std::atomic<uint64_t> min_latency_ns;
    } stats_;
    
public:
    explicit TransactionProcessor(const ProcessorConfig& config);
    ~TransactionProcessor();
    void start();
    void stop();
    ProcessingResult submit(const Transaction& txn);
    std::vector<ProcessingResult> submitBatch(const std::vector<Transaction>& txns);
    std::optional<Transaction> getTransaction(const TxHash& hash);
    
    struct ProcessorStats {
        uint64_t total_processed;
        uint64_t total_failed;
        uint64_t total_gas_used;
        uint64_t avg_latency_ns;
        uint64_t max_latency_ns;
        uint64_t min_latency_ns;
        size_t pending_count;
    };
    
    ProcessorStats getStats() const;
    bool isHealthy() const;
    
private:
    void workerLoop();
    ProcessingResult processTransaction(Transaction& txn);
    bool validateTransaction(const Transaction& txn);
    bool deduplicateTransaction(const Transaction& txn);
};

class TransactionEventEmitter {
public:
    using EventCallback = std::function<void(const Transaction&)>;
    void onTransactionConfirmed(EventCallback cb);
    void onTransactionFailed(EventCallback cb);
    void onTransactionFlagged(EventCallback cb);
    void emitConfirmed(const Transaction& txn);
    void emitFailed(const Transaction& txn);
    void emitFlagged(const Transaction& txn);
private:
    std::vector<EventCallback> confirmed_, failed_, flagged_;
    std::mutex mutex_;
};

class RateLimiter {
private:
    uint64_t max_requests_per_second_;
    std::atomic<uint64_t> current_count_;
    std::chrono::steady_clock::time_point window_start_;
    std::mutex mutex_;
public:
    explicit RateLimiter(uint64_t max_rps);
    bool allow();
    void reset();
    uint64_t currentCount() const;
};

class BlockBuilder {
private:
    std::vector<Transaction> transactions_;
    uint64_t gas_limit_;
    uint64_t gas_used_;
    uint64_t block_number_;
    Address miner_;
public:
    BlockBuilder(uint64_t block_number, uint64_t gas_limit);
    bool addTransaction(const Transaction& txn);
    void removeTransaction(size_t index);
    std::vector<Transaction> build();
    uint64_t getGasUsed() const { return gas_used_; }
    size_t getTransactionCount() const { return transactions_.size(); }
    uint64_t getBlockNumber() const { return block_number_; }
};

class MempoolMonitor {
private:
    std::unordered_map<TxHash, Transaction, TxHash::hash> transactions_;
    std::mutex mutex_;
public:
    void addTransaction(const Transaction& txn);
    void removeTransaction(const TxHash& hash);
    std::vector<Transaction> getHighValueTransactions(uint64_t threshold);
    std::vector<Transaction> getByAddress(const Address& addr);
    size_t size() const;
};

} // namespace txn
} // namespace tigerwallet

#endif
