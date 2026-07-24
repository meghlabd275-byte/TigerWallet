/**
 * TigerWallet Mempool Analyzer
 * Ultra-Low Latency C++ Implementation
 * 
 * Features:
 * - Transaction ordering analysis
 * - MEV detection
 * - Gas price forecasting
 * - Front-running protection
 */

#ifndef TIGER_MEMPOOL_H
#define TIGER_MEMPOOL_H

#include <atomic>
#include <chrono>
#include <functional>
#include <mutex>
#include <queue>
#include <string>
#include <unordered_map>
#include <vector>

namespace tiger {
namespace mempool {

struct PendingTransaction {
    std::string hash;
    std::string from;
    std::string to;
    uint256_t value;
    uint256_t gas_price;
    uint64_t gas_limit;
    uint64_t nonce;
    std::string data;
    std::string chain;
    uint64_t received_at;
    uint64_t timestamp;
    std::vector<std::string> ancestors;
    std::vector<std::string> descendants;
};

struct MempoolStats {
    uint64_t pending_txs;
    uint64_t total_gas;
    double avg_gas_price;
    uint64_t unique_senders;
    uint64_t bundle_opportunities;
};

struct MEVOpportunity {
    std::string type; // arbitrage, sandwich, liquidation
    double estimated_profit;
    std::vector<std::string> transactions;
    std::string attack_vector;
    uint64_t timestamp;
};

class MempoolAnalyzer {
private:
    std::unordered_map<std::string, PendingTransaction> transactions_;
    std::mutex mutex_;
    std::atomic<uint64_t> total_analyzed_{0};

public:
    MempoolAnalyzer();
    ~MempoolAnalyzer() = default;

    void add_transaction(const PendingTransaction& tx);
    void remove_transaction(const std::string& hash);
    std::vector<PendingTransaction> get_transactions(const std::string& from);
    MempoolStats get_stats();
    std::vector<MEVOpportunity> detect_mev();
    std::vector<PendingTransaction> get_sandwich_victims();
    double calculate_gas_optimal();
};

inline MempoolAnalyzer::MempoolAnalyzer() {
}

inline void MempoolAnalyzer::add_transaction(const PendingTransaction& tx) {
    std::lock_guard<std::mutex> lock(mutex_);
    transactions_[tx.hash] = tx;
    total_analyzed_++;
}

inline void MempoolAnalyzer::remove_transaction(const std::string& hash) {
    std::lock_guard<std::mutex> lock(mutex_);
    transactions_.erase(hash);
}

inline std::vector<PendingTransaction> MempoolAnalyzer::get_transactions(const std::string& from) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<PendingTransaction> result;
    for (const auto& tx : transactions_) {
        if (tx.second.from == from) {
            result.push_back(tx.second);
        }
    }
    return result;
}

inline MempoolStats MempoolAnalyzer::get_stats() {
    std::lock_guard<std::mutex> lock(mutex_);
    MempoolStats stats;
    stats.pending_txs = transactions_.size();
    
    uint64_t total_gas = 0;
    uint64_t gas_price_sum = 0;
    std::unordered_set<std::string> senders;
    
    for (const auto& tx : transactions_) {
        total_gas += tx.second.gas_limit;
        gas_price_sum += tx.second.gas_price;
        senders.insert(tx.second.from);
    }
    
    stats.total_gas = total_gas;
    stats.avg_gas_price = senders.size() > 0 ? gas_price_sum / senders.size() : 0;
    stats.unique_senders = senders.size();
    stats.bundle_opportunities = detect_mev().size();
    
    return stats;
}

inline std::vector<MEVOpportunity> MempoolAnalyzer::detect_mev() {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<MEVOpportunity> opportunities;
    
    // Simplified MEV detection
    for (const auto& tx : transactions_) {
        // Check for large swap
        if (tx.second.data.find("swap") != std::string::npos) {
            MEVOpportunity opp;
            opp.type = "sandwich";
            opp.estimated_profit = 0.05; // 5% estimate
            opp.transactions.push_back(tx.first);
            opp.attack_vector = "uniswap_v3";
            opp.timestamp = std::chrono::duration_cast<std::chrono::seconds>(
                std::chrono::system_clock::now().time_since_epoch()
            ).count();
            opportunities.push_back(opp);
        }
    }
    
    return opportunities;
}

inline std::vector<PendingTransaction> MempoolAnalyzer::get_sandwich_victims() {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<PendingTransaction> victims;
    
    for (const auto& tx : transactions_) {
        if (tx.second.gas_price > 100) { // High gas = likely urgent
            victims.push_back(tx.second);
        }
    }
    
    return victims;
}

inline double MempoolAnalyzer::calculate_gas_optimal() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (transactions_.empty()) {
        return 20.0; // Default 20 gwei
    }
    
    // Calculate optimal gas based on mempool saturation
    uint64_t count = transactions_.size();
    double base_gas = 20.0;
    
    if (count > 10000) {
        return base_gas * 3.0;
    } else if (count > 5000) {
        return base_gas * 2.0;
    } else if (count > 1000) {
        return base_gas * 1.5;
    }
    
    return base_gas;
}

} // namespace mempool
} // namespace tiger

#endif
