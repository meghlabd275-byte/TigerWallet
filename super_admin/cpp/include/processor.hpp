/**
 * TigerWallet Admin - High Performance Transaction Processor
 * Ultra-low latency C++ implementation
 */
#ifndef TIGER_ADMIN_PROCESSOR_HPP
#define TIGER_ADMIN_PROCESSOR_HPP

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <functional>
#include <memory>
#include <mutex>
#include <queue>
#include <shared_mutex>
#include <string>
#include <unordered_map>
#include <vector>
#include <optional>
#include <variant>
#include <cstdint>
#include <cstring>
#include <thread>

namespace tiger {
namespace admin {
namespace processor {

using Timestamp = std::chrono::time_point<std::chrono::steady_clock>;
using Duration = std::chrono::microseconds;

enum class TransactionType { DEPOSIT, WITHDRAWAL, TRANSFER, SWAP, TRADE, FEE };
enum class TransactionStatus { PENDING, PROCESSING, COMPLETED, FAILED, FLAGGED, CANCELLED };
enum class RiskLevel { LOW, MEDIUM, HIGH, CRITICAL };

struct alignas(64) Transaction {
    uint64_t id;
    uint64_t user_id;
    uint64_t chain_id;
    TransactionType type;
    TransactionStatus status;
    RiskLevel risk_level;
    int64_t amount;
    int64_t fee;
    uint32_t token_id;
    uint32_t from_chain_id;
    uint32_t to_chain_id;
    uint64_t from_address_hash;
    uint64_t to_address_hash;
    uint64_t created_at_ns;
    uint64_t processed_at_ns;
    float risk_score;
    uint8_t flags;
    char tx_hash[72];
    char tx_data[256];
};

struct alignas(64) StatsCounters {
    std::atomic<uint64_t> total_transactions{0};
    std::atomic<uint64_t> processed_transactions{0};
    std::atomic<uint64_t> failed_transactions{0};
    std::atomic<uint64_t> flagged_transactions{0};
    std::atomic<uint64_t> total_volume{0};
    std::atomic<uint64_t> total_fees{0};
    std::atomic<uint64_t> processing_time_us{0};
    std::atomic<uint64_t> max_processing_time_us{0};
};

struct RiskAssessment {
    RiskLevel level;
    float score;
    std::vector<std::string> reasons;
};

template<typename T, size_t Capacity = 65536>
class LockFreeQueue {
private:
    struct Node { T data; std::atomic<Node*> next; };
    alignas(64) std::atomic<Node*> head_;
    alignas(64) std::atomic<Node*> tail_;
public:
    LockFreeQueue() {
        Node* dummy = new Node();
        dummy->next.store(nullptr, std::memory_order_relaxed);
        head_.store(dummy, std::memory_order_relaxed);
        tail_.store(dummy, std::memory_order_relaxed);
    }
    ~LockFreeQueue() {
        while (dequeue(nullptr)) {}
        delete head_.load();
    }
    bool enqueue(const T& data) {
        Node* node = new Node();
        node->data = data;
        node->next.store(nullptr, std::memory_order_relaxed);
        Node* old_tail = tail_.load(std::memory_order_relaxed);
        while (!tail_.compare_exchange_weak(old_tail, node, std::memory_order_release, std::memory_order_relaxed)) {}
        old_tail->next.store(node, std::memory_order_release);
        return true;
    }
    bool dequeue(T* out) {
        Node* old_head = head_.load(std::memory_order_relaxed);
        Node* next = old_head->next.load(std::memory_order_acquire);
        if (next == nullptr) return false;
        if (out) *out = next->data;
        head_.store(next, std::memory_order_release);
        delete old_head;
        return true;
    }
};

class TransactionCache {
private:
    static constexpr size_t MAX_CACHE_SIZE = 100000;
    static constexpr size_t HASH_BUCKETS = 131072;
    struct Entry { Transaction txn; Timestamp cached_at; };
    std::unordered_map<uint64_t, Entry> cache_;
    std::vector<std::unordered_map<uint64_t, Entry>> hash_table_;
    mutable std::shared_mutex mutex_;
public:
    TransactionCache() : hash_table_(HASH_BUCKETS) {}
    void put(const Transaction& txn) {
        std::unique_lock lock(mutex_);
        size_t bucket = txn.id % HASH_BUCKETS;
        hash_table_[bucket][txn.id] = {txn, std::chrono::steady_clock::now()};
        cache_[txn.id] = {txn, std::chrono::steady_clock::now()};
    }
    std::optional<Transaction> get(uint64_t id) const {
        std::shared_lock lock(mutex_);
        size_t bucket = id % HASH_BUCKETS;
        auto it = hash_table_[bucket].find(id);
        if (it != hash_table_[bucket].end()) return it->second.txn;
        return std::nullopt;
    }
};

class RiskEngine {
private:
    float high_risk_threshold_ = 0.7f;
    float critical_risk_threshold_ = 0.9f;
    int64_t large_transaction_threshold_ = 1000000000;
public:
    RiskAssessment assess(const Transaction& txn) {
        RiskAssessment result;
        result.score = 0.0f;
        if (txn.amount > large_transaction_threshold_) {
            result.score += 0.3f;
            result.reasons.push_back("Large transaction amount");
        }
        if (txn.from_chain_id != txn.to_chain_id) {
            result.score += 0.15f;
            result.reasons.push_back("Cross-chain transaction");
        }
        if (result.score >= critical_risk_threshold_) result.level = RiskLevel::CRITICAL;
        else if (result.score >= high_risk_threshold_) result.level = RiskLevel::HIGH;
        else if (result.score >= 0.3f) result.level = RiskLevel::MEDIUM;
        else result.level = RiskLevel::LOW;
        return result;
    }
};

class TransactionProcessor {
private:
    LockFreeQueue<Transaction> queue_;
    TransactionCache cache_;
    RiskEngine risk_engine_;
    StatsCounters stats_;
    std::atomic<bool> running_{false};
    std::vector<std::thread> worker_threads_;
    std::vector<std::function<void(const Transaction&)>> on_complete_;
    std::vector<std::function<void(const Transaction&, const RiskAssessment&)>> on_high_risk_;
public:
    TransactionProcessor() {}
    ~TransactionProcessor() { stop(); }
    
    void start(uint32_t num_threads = 4) {
        if (running_.load()) return;
        running_.store(true);
        for (uint32_t i = 0; i < num_threads; ++i) {
            worker_threads_.emplace_back([this]() { process_loop(); });
        }
    }
    
    void stop() {
        if (!running_.load()) return;
        running_.store(false);
        for (auto& t : worker_threads_) {
            if (t.joinable()) t.join();
        }
        worker_threads_.clear();
    }
    
    bool submit(const Transaction& txn) {
        if (!running_.load()) return false;
        return queue_.enqueue(txn);
    }
    
    RiskAssessment process_one(Transaction& txn) {
        auto start = std::chrono::steady_clock::now();
        RiskAssessment risk = risk_engine_.assess(txn);
        txn.risk_score = risk.score;
        txn.risk_level = risk.level;
        
        if (risk.level == RiskLevel::CRITICAL || risk.level == RiskLevel::HIGH) {
            txn.status = TransactionStatus::FLAGGED;
            stats_.flagged_transactions.fetch_add(1, std::memory_order_relaxed);
            for (auto& cb : on_high_risk_) cb(txn, risk);
        } else {
            txn.status = TransactionStatus::COMPLETED;
            stats_.processed_transactions.fetch_add(1, std::memory_order_relaxed);
        }
        
        cache_.put(txn);
        
        auto end = std::chrono::steady_clock::now();
        auto duration = std::chrono::duration_cast<Duration>(end - start).count();
        stats_.total_transactions.fetch_add(1, std::memory_order_relaxed);
        stats_.processing_time_us.fetch_add(duration, std::memory_order_relaxed);
        
        for (auto& cb : on_complete_) cb(txn);
        return risk;
    }
    
    std::optional<Transaction> get_transaction(uint64_t id) const { return cache_.get(id); }
    
    struct ProcessorStats {
        uint64_t total_transactions;
        uint64_t processed_transactions;
        uint64_t flagged_transactions;
        double avg_processing_time_us;
    };
    
    ProcessorStats get_stats() const {
        ProcessorStats stats;
        stats.total_transactions = stats_.total_transactions.load();
        stats.processed_transactions = stats_.processed_transactions.load();
        stats.flagged_transactions = stats_.flagged_transactions.load();
        uint64_t t = stats_.processing_time_us.load();
        uint64_t p = stats_.processed_transactions.load();
        stats.avg_processing_time_us = p > 0 ? static_cast<double>(t) / p : 0;
        return stats;
    }
    
    void on_complete(std::function<void(const Transaction&)> cb) { on_complete_.push_back(cb); }
    void on_high_risk(std::function<void(const Transaction&, const RiskAssessment&)> cb) { on_high_risk_.push_back(cb); }

private:
    void process_loop() {
        while (running_.load()) {
            Transaction txn;
            if (queue_.dequeue(&txn)) process_one(txn);
            else std::this_thread::yield();
        }
    }
};

inline std::unique_ptr<TransactionProcessor> create_processor() {
    return std::make_unique<TransactionProcessor>();
}

}  // namespace processor
}  // namespace admin
}  // namespace tiger

#endif
