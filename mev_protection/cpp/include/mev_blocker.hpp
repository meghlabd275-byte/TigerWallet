/**
 * TigerWallet MEV Protection System
 * Ultra-low latency C++ implementation for MEV blocking
 * 
 * This provides:
 * - Front-running protection
 * - Back-running protection
 * - Sandwich attack prevention
 * - Transaction reordering optimization
 * - Flashbots Protect integration
 */

#ifndef MEV_BLOCKER_HPP
#define MEV_BLOCKER_HPP

#include <array>
#include <chrono>
#include <cstdint>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <thread>
#include <unordered_map>
#include <vector>

#include "keccak.hpp"
#include "transaction.hpp"

namespace tiger {

// Maximum transactions per block
constexpr size_t kMaxTransactionsPerBlock = 2000;
constexpr size_t kMempoolSize = 10000;
constexpr auto kBlockTime = std::chrono::seconds(12);
constexpr auto kBundleCheckInterval = std::chrono::milliseconds(100);

// MEV Protection levels
enum class ProtectionLevel {
    kNone = 0,
    kBasic = 1,
    kAdvanced = 2,
    kMaximum = 3
};

// Transaction classification
enum class TransactionType {
    kRegular = 0,
    kSwap = 1,
    kTransfer = 2,
    kNFT = 3,
    kContractInteraction = 4
};

// Risk assessment result
struct RiskAssessment {
    bool is_profitable;
    float profit_extracted;
    std::string attack_type;
    float risk_score;
    std::vector<std::string> recommendations;
    
    RiskAssessment() : is_profitable(false), profit_extracted(0.0f), 
                      risk_score(0.0f) {}
};

// Bundle for MEV extraction
struct MEVBundle {
    uint64_t bundle_id;
    std::vector<Transaction> transactions;
    uint256_t bundle_gas_price;
    uint256_t miner_reward;
    float priority_fee;
    std::chrono::steady_clock::time_point submitted_at;
    bool backrunnable;
    
    MEVBundle() : bundle_id(0), bundle_gas_price(0), 
                  miner_reward(0), priority_fee(0.0f), 
                  backrunnable(false) {}
};

// Gas oracle data
struct GasOracle {
    uint256_t base_fee;
    uint256_t priority_fee_slow;
    uint256_t priority_fee_standard;
    uint256_t priority_fee_fast;
    uint256_t priority_fee_instant;
    std::chrono::steady_clock::time_point updated_at;
    
    GasOracle() : base_fee(0), priority_fee_slow(0), 
                   priority_fee_standard(0), priority_fee_fast(0),
                   priority_fee_instant(0) {}
};

// Block proposal
struct BlockProposal {
    uint64_t block_number;
    uint64_t slot;
    std::vector<Transaction> transactions;
    uint256_t base_fee;
    uint256_t gas_limit;
    std::chrono::steady_clock::time_point received_at;
    
    BlockProposal() : block_number(0), slot(0), base_fee(0), gas_limit(0) {}
};

class MEVBlocker {
public:
    /**
     * Construct MEV blocker with configuration
     */
    explicit MEVBlocker(ProtectionLevel level = ProtectionLevel::kAdvanced);
    
    /**
     * Destructor - cleanup resources
     */
    ~MEVBlocker();
    
    // Disable copy and move
    MEVBlocker(const MEVBlocker&) = delete;
    MEVBlocker& operator=(const MEVBlocker&) = delete;
    MEVBlocker(MEVBlocker&&) = delete;
    MEVBlocker& operator=(MEVBlocker&&) = delete;
    
    /**
     * Initialize the MEV blocker
     */
    bool Initialize(const std::string& config_path);
    
    /**
     * Start the MEV protection service
     */
    bool Start();
    
    /**
     * Stop the MEV protection service
     */
    void Stop();
    
    /**
     * Submit a transaction for MEV protection
     * Returns: Transaction hash if protected, empty if blocked
     */
    std::string SubmitTransaction(const Transaction& tx);
    
    /**
     * Submit a private transaction (Flashbots)
     */
    std::string SubmitPrivateTransaction(const Transaction& tx);
    
    /**
     * Analyze a transaction for MEV risk
     */
    RiskAssessment AnalyzeTransaction(const Transaction& tx);
    
    /**
     * Get current gas prices
     */
    GasOracle GetGasOracle() const;
    
    /**
     * Get block proposals for validation
     */
    std::vector<BlockProposal> GetBlockProposals() const;
    
    /**
     * Set protection level
     */
    void SetProtectionLevel(ProtectionLevel level);
    
    /**
     * Get statistics
     */
    struct Stats {
        uint64_t total_transactions;
        uint64_t protected_transactions;
        uint64_t blocked_transactions;
        uint64_t bundles_submitted;
        uint64_t bundles_included;
        uint256_t total_profit;
        float avg_protection_time_ms;
    };
    
    Stats GetStats() const;
    
    /**
     * Add trusted relayer
     */
    bool AddRelayer(const std::string& relayer_url);
    
    /**
     * Remove trusted relayer
     */
    bool RemoveRelayer(const std::string& relayer_url);
    
    /**
     * Check if transaction is private
     */
    bool IsPrivateTransaction(const std::string& tx_hash) const;
    
    /**
     * Get flashbots protect status
     */
    bool IsFlashbotsProtected(const std::string& tx_hash) const;

private:
    // Configuration
    ProtectionLevel protection_level_;
    bool running_;
    bool initialized_;
    
    // Gas oracle data
    GasOracle gas_oracle_;
    mutable std::mutex gas_mutex_;
    
    // Transaction pool
    std::vector<Transaction> mempool_;
    mutable std::mutex mempool_mutex_;
    
    // Private transactions (Flashbots)
    std::unordered_map<std::string, Transaction> private_txs_;
    mutable std::mutex private_mutex_;
    
    // Block proposals
    std::vector<BlockProposal> proposals_;
    mutable std::mutex proposals_mutex_;
    
    // Relayers
    std::vector<std::string> relayers_;
    mutable std::mutex relayers_mutex_;
    
    // Statistics
    Stats stats_;
    mutable std::mutex stats_mutex_;
    
    // Worker threads
    std::vector<std::thread> workers_;
    
    // Methods
    void RunGasOracleUpdater();
    void RunMempoolCleaner();
    void RunBundleProcessor();
    
    bool ShouldBlockTransaction(const Transaction& tx);
    bool IsSandwichAttack(const Transaction& tx);
    bool IsFrontRunning(const Transaction& tx);
    bool IsBackRunning(const Transaction& tx);
    
    std::string SubmitToFlashbots(const Transaction& tx);
    std::string SubmitToRelayer(const Transaction& tx, const std::string& relayer);
    
    uint256_t CalculateOptimalGasPrice(const Transaction& tx);
    float EstimateMEVProfit(const Transaction& tx);
    
    void UpdateStatistics(bool protected_, bool blocked_);
    
    // Sandwich detection
    struct SandwichOpportunity {
        Transaction front_run;
        Transaction victim;
        Transaction back_run;
        uint256_t potential_profit;
    };
    
    std::optional<SandwichOpportunity> DetectSandwich(const Transaction& tx);
    
    // Transaction classification
    TransactionType ClassifyTransaction(const Transaction& tx);
    
    // Simulate transaction
    bool SimulateTransaction(const Transaction& tx, std::string& output);
};

// Factory function
std::unique_ptr<MEVBlocker> CreateMEVBlocker(ProtectionLevel level);

}  // namespace tiger

#endif  // MEV_BLOCKER_HPP
