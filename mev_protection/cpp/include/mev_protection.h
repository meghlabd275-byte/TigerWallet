/**
 * TigerWallet - MEV Protection Engine
 * C++ implementation for MEV (Miner Extractable Value) protection
 * 
 * Features:
 * - Sandwich attack detection
 * - Flashbots integration
 * - Private transaction routing
 * - Front-run protection
 * - Order flow auction
 */

#ifndef TIGERWALLET_MEV_PROTECTION_H
#define TIGERWALLET_MEV_PROTECTION_H

#include <string>
#include <vector>
#include <map>
#include <queue>
#include <memory>
#include <mutex>
#include <chrono>
#include <atomic>
#include <optional>
#include <variant>

namespace tiger {
namespace mev {

// ============ Types ============

using Address = std::array<uint8_t, 20>;
using Bytes32 = std::array<uint8_t, 32>;
using Hash = std::array<uint8_t, 32>;

enum class TransactionType {
    EIP1559,    // Type 2
    EIP2930,    // Type 1
    Legacy,     // Type 0
};

enum class MEVProtectionLevel {
    None,
    Basic,      // Simple front-running protection
    Standard,   // Flashbots Protect integration
    Advanced,   // Private pool routing + order flow
    Maximum,    // All protections enabled
};

enum class AttackType {
    None,
    Sandwich,
    FrontRun,
    BackRun,
    Liquidation,
    Arbitrage,
};

struct Transaction {
    Hash tx_hash;
    Address from;
    Address to;
    uint64_t nonce;
    uint256_t gas_price;
    uint256_t max_fee_per_gas;
    uint256_t max_priority_fee_per_gas;
    uint64_t gas_limit;
    uint256_t value;
    Bytes32 data;
    TransactionType type;
    uint64_t chain_id;
    Timestamp timestamp;
};

struct PendingTransaction {
    Transaction tx;
    uint64_t received_at;
    uint64_t gas_used;
    bool is_simulation;
};

struct SandwichAttack {
    AttackType type;
    Hash front_run_hash;
    Hash target_hash;
    Hash back_run_hash;
    Address front_runner;
    Address victim;
    uint256_t profit;
    uint64_t detected_at;
    double confidence;
};

struct BlockState {
    uint64_t block_number;
    Hash block_hash;
    std::vector<PendingTransaction> transactions;
    std::vector<Address> validators;
    uint64_t base_fee;
};

struct ProtectionResult {
    bool allowed;
    AttackType attack_detected;
    std::string reason;
    std::optional<Hash> private_tx_hash;
    std::optional<Hash> flashbots_bundle_hash;
    double savings;
};

// ============ Transaction Analyzer ============

class TransactionAnalyzer {
public:
    TransactionAnalyzer();
    
    // Analyze transaction for MEV patterns
    AttackType analyze(const Transaction& tx);
    
    // Detect sandwich attacks
    std::optional<SandwichAttack> detect_sandwich(
        const Transaction& pending,
        const std::vector<Transaction>& mempool
    );
    
    // Calculate transaction risk score
    double calculate_risk_score(const Transaction& tx);
    
    // Check if transaction is likely to be frontrun
    bool is_frontrunnable(const Transaction& tx);
    
    // Get affected tokens
    std::vector<Address> get_affected_tokens(const Transaction& tx);
    
private:
    // Known MEV patterns
    std::map<std::string, AttackType> mev_patterns_;
    
    // DEX addresses for detection
    std::vector<Address> dex_addresses_;
    
    // Analyze swap patterns
    AttackType analyze_swap(const Transaction& tx);
    
    // Analyze NFT patterns
    AttackType analyze_nft(const Transaction& tx);
    
    // Analyze liquidation patterns
    AttackType analyze_liquidation(const Transaction& tx);
};

// ============ MEV Protection Engine ============

class MEVProtectionEngine {
public:
    struct Config {
        MEVProtectionLevel level;
        bool enable_flashbots;
        bool enable_private_pool;
        bool enable_oaaf;        // Order Flow Auction
        uint64_t max_slippage_bps;
        uint64_t block_timeout_ms;
        std::string rpc_url;
        std::string flashbots_url;
    };
    
    explicit MEVProtectionEngine(const Config& config);
    ~MEVProtectionEngine();
    
    // Initialize engine
    bool initialize();
    
    // Process transaction
    ProtectionResult process_transaction(const Transaction& tx);
    
    // Submit to Flashbots
    std::optional<Hash> submit_to_flashbots(
        const Transaction& tx,
        uint256_t gas_price
    );
    
    // Submit bundle
    std::optional<Hash> submit_bundle(
        const std::vector<Transaction>& txs,
        uint64_t block_number
    );
    
    // Simulate transaction
    std::pair<bool, std::string> simulate(
        const Transaction& tx,
        std::vector<Transaction>& bundles
    );
    
    // Get protected RPC URL
    std::string get_protected_rpc() const;
    
    // Get estimated savings
    double get_estimated_savings() const { return total_savings_; }
    
    // Get blocked attacks
    uint64_t get_blocked_attacks() const { return blocked_attacks_; }
    
    // Start monitoring
    void start();
    
    // Stop monitoring
    void stop();

private:
    Config config_;
    std::unique_ptr<TransactionAnalyzer> analyzer_;
    std::atomic<bool> running_;
    std::thread monitoring_thread_;
    std::mutex mutex_;
    
    // Pending transactions queue
    std::queue<PendingTransaction> pending_txs_;
    
    // Mempool state
    std::map<Address, std::queue<Transaction>> mempool_;
    
    // Statistics
    std::atomic<uint64_t> total_txs_;
    std::atomic<uint64_t> blocked_attacks_;
    std::atomic<double> total_savings_;
    
    // Flashbots client
    class FlashbotsClient* flashbots_client_;
    
    // Private pool clients
    std::vector<std::string> private_pools_;
    
    // Monitoring loop
    void monitoring_loop();
    
    // Check for sandwich attacks
    std::optional<SandwichAttack> check_sandwich(const Transaction& tx);
    
    // Route to private pool
    std::optional<Hash> route_to_private_pool(const Transaction& tx);
    
    // Calculate optimal gas price
    uint256_t calculate_optimal_gas(const Transaction& tx);
    
    // Validate transaction before protection
    bool validate_transaction(const Transaction& tx);
};

// ============ Flashbots Client ============

class FlashbotsClient {
public:
    FlashbotsClient(const std::string& rpc_url, const std::string& secret);
    
    // Send transaction to Flashbots
    std::optional<Hash> send_transaction(
        const Transaction& tx,
        uint256_t gas_price,
        uint64_t max_block_number
    );
    
    // Send bundle
    std::optional<Hash> send_bundle(
        const std::vector<Transaction>& txs,
        uint64_t block_number,
        uint64_t min_timestamp,
        uint64_t max_timestamp
    );
    
    // Simulate bundle
    std::pair<bool, std::string> simulate_bundle(
        const std::vector<Transaction>& txs,
        uint64_t block_number
    );
    
    // Get bundle status
    std::map<std::string, std::string> get_bundle_status(const Hash& bundle_hash);
    
    // Get credit balance
    uint256_t get_credit_balance();

private:
    std::string rpc_url_;
    std::string secret_;
    
    // Make RPC call
    std::variant<std::string, nlohmann::json> call(
        const std::string& method,
        const nlohmann::json& params
    );
};

// ============ Private Pool Manager ============

class PrivatePoolManager {
public:
    struct Pool {
        std::string name;
        std::string rpc_url;
        bool is_active;
        double success_rate;
        uint64_t avg_latency_ms;
    };
    
    PrivatePoolManager();
    
    // Add private pool
    void add_pool(const Pool& pool);
    
    // Remove pool
    void remove_pool(const std::string& name);
    
    // Get best pool
    const Pool* get_best_pool() const;
    
    // Route transaction to pool
    std::optional<Hash> route_transaction(
        const Transaction& tx,
        const Pool& pool
    );
    
    // Get all pools
    std::vector<Pool> get_pools() const;
    
    // Update pool stats
    void update_pool_stats(
        const std::string& name,
        bool success,
        uint64_t latency_ms
    );

private:
    std::vector<Pool> pools_;
    mutable std::mutex mutex_;
    
    // Score pools
    double calculate_pool_score(const Pool& pool) const;
};

// ============ Order Flow Auction ============

class OrderFlowAuction {
public:
    struct Bid {
        Address solver;
        uint256_t bid_amount;
        uint256_t execution_gas;
        std::string metadata;
    };
    
    OrderFlowAuction();
    
    // Submit bid
    bool submit_bid(const Bid& bid);
    
    // Get winning bid
    std::optional<Bid> get_winning_bid(const Transaction& tx);
    
    // Settle auction
    void settle_auction(const Hash& tx_hash);
    
    // Get auction revenue
    uint256_t get_total_revenue() const;

private:
    std::map<Hash, std::vector<Bid>> auctions_;
    std::map<Address, uint256_t> solver_revenue_;
    std::mutex mutex_;
    
    // Auction timeout (ms)
    static constexpr uint64_t AUCTION_TIMEOUT_MS = 100;
};

// ============ Sandwich Detector ============

class SandwichDetector {
public:
    SandwichDetector();
    
    // Analyze mempool for sandwich opportunities
    std::vector<SandwichAttack> detect_sandwiches(
        const std::vector<Transaction>& mempool
    );
    
    // Check if transaction pair is sandwich
    std::optional<SandwichAttack> check_pair(
        const Transaction& front,
        const Transaction& victim,
        const Transaction& back
    );
    
    // Calculate sandwich profit
    uint256_t calculate_profit(
        const Transaction& victim,
        const Transaction& front,
        const Transaction& back
    );
    
    // Get common sandwich patterns
    std::vector<AttackType> get_common_patterns() const;

private:
    // DEX addresses
    std::vector<Address> dex_addresses_;
    
    // Token pairs
    std::map<Address, Address> token_pairs_;
    
    // Historical sandwich data
    std::vector<SandwichAttack> history_;
    
    // ML model for detection (simplified)
    double detect_ml(const Transaction& tx);
};

// ============ Gas Optimization ============

class GasOptimizer {
public:
    GasOptimizer();
    
    // Optimize gas settings for transaction
    void optimize(
        Transaction& tx,
        const std::vector<Transaction>& mempool
    );
    
    // Estimate optimal gas price
    uint256_t estimate_optimal_gas(
        uint64_t target_time_ms,
        const std::vector<Transaction>& mempool
    );
    
    // Get historical gas data
    std::vector<uint256_t> get_gas_history(uint64_t hours) const;
    
    // Predict future gas prices
    std::pair<uint256_t, uint256_t> predict_gas(
        uint64_t minutes_ahead
    );

private:
    // Gas price history
    std::deque<uint256_t> gas_history_;
    
    // Prediction model (simplified)
    double predict_simple(uint64_t minutes) const;
    
    // Calculate priority fee
    uint256_t calculate_priority_fee(
        const std::vector<Transaction>& mempool
    );
};

// ============ Utilities ============

namespace utils {
    // Parse transaction from RLP
    Transaction parse_transaction(const std::vector<uint8_t>& rlp);
    
    // Calculate transaction hash
    Hash calculate_hash(const Transaction& tx);
    
    // Get sender from transaction
    Address get_sender(const Transaction& tx);
    
    // Validate signature
    bool validate_signature(const Transaction& tx);
    
    // Encode transaction to RLP
    std::vector<uint8_t> encode_transaction(const Transaction& tx);
}

} // namespace mev
} // namespace tiger

#endif // TIGERWALLET_MEV_PROTECTION_H
