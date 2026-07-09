/**
 * TigerWallet ERC-4337 Bundler - High Performance UserOperation Bundler
 * 
 * This is a production-grade C++ implementation of an ERC-4337 bundler that:
 * - Receives UserOperations from relayers
 * - Simulates validation and execution
 * - Batches and submits to EntryPoint contract
 * - Handles gas estimation and fee markets
 * 
 * Author: TigerWallet Development Team
 * License: MIT
 */

#ifndef TIGERWALLET_BUNDLER_HPP
#define TIGERWALLET_BUNDLER_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <optional>
#include <functional>
#include <mutex>
#include <atomic>
#include <thread>
#include <queue>
#include <unordered_map>
#include <chrono>

#include "user_operation.hpp"
#include "entry_point.hpp"
#include "gas_estimator.hpp"
#include "simulation.hpp"
#include " reputation.hpp"

namespace tigerwallet {
namespace bundler {

/**
 * Configuration for the bundler
 */
struct BundlerConfig {
    std::string entry_point_address;
    std::string chain_id;
    uint32_t max_batch_size;
    uint64_t max_gas_price_gwei;
    uint64_t priority_fee_gwei;
    uint32_t simulation_timeout_ms;
    uint32_t max_revert_reason_size;
    bool enable_reputation;
    std::vector<std::string> whitelisted_contracts;
    std::vector<std::string> blacklisted_addresses;
    std::string rpc_endpoint;
    std::string redis_url;
};

/**
 * Result of bundling operation
 */
struct BundleResult {
    std::string bundle_hash;
    std::vector<std::string> user_operation_hashes;
    uint64_t total_gas_used;
    uint64_t actual_gas_price;
    std::string transaction_hash;
    std::string error_message;
    bool success;
};

/**
 * Reputation status for an address
 */
struct ReputationStatus {
    reputation:: ReputationLevel level;
    uint64_t utxo_count;
    uint64_t op_count;
    uint64_t last_update_time;
    double reputation_score;
};

/**
 * Main Bundler class - Production Implementation
 */
class Bundler {
public:
    explicit Bundler(const BundlerConfig& config);
    ~Bundler();

    /**
     * Initialize the bundler and all dependencies
     */
    bool initialize();

    /**
     * Start the bundler main loop
     */
    void start();

    /**
     * Stop the bundler gracefully
     */
    void stop();

    /**
     * Submit a UserOperation for bundling
     * @param user_op The UserOperation to submit
     * @return The hash of the UserOperation
     */
    std::string submit_user_operation(const UserOperation& user_op);

    /**
     * Get pending UserOperations
     */
    std::vector<UserOperation> get_pending_operations();

    /**
     * Estimate gas for a UserOperation
     */
    std::optional<GasEstimate> estimate_gas(const UserOperation& user_op);

    /**
     * Simulate validation of a UserOperation
     */
    SimulationResult simulate_validation(const UserOperation& user_op);

    /**
     * Execute bundled operations
     */
    std::optional<BundleResult> execute_bundle();

    /**
     * Get reputation for an address
     */
    ReputationStatus get_reputation(const std::string& address);

    /**
     * Update reputation after bundle execution
     */
    void update_reputation(const std::string& address, bool success);

    /**
     * Get bundler statistics
     */
    BundlerStats get_stats() const;

private:
    // Configuration
    BundlerConfig config_;

    // Components
    std::unique_ptr<EntryPoint> entry_point_;
    std::unique_ptr<GasEstimator> gas_estimator_;
    std::unique_ptr<SimulationEngine> simulation_engine_;
    std::unique_ptr<ReputationManager> reputation_manager_;

    // Operation queues
    std::queue<UserOperation> pending_operations_;
    std::queue<UserOperation> bundle_queue_;
    mutable std::mutex operations_mutex_;

    // State
    std::atomic<bool> running_{false};
    std::atomic<uint64_t> total_bundles_{0};
    std::atomic<uint64_t> total_gas_saved_{0};
    std::chrono::steady_clock::time_point start_time_;

    // Threading
    std::vector<std::thread> worker_threads_;
    std::thread main_loop_thread_;

    // Internal methods
    bool validate_user_operation(const UserOperation& user_op);
    bool check_reputation(const std::string& address);
    std::vector<UserOperation> create_bundle();
    bool simulate_bundle(const std::vector<UserOperation>& bundle);
    std::string calculate_bundle_hash(const std::vector<UserOperation>& bundle);
    void cleanup_old_operations();
    void update_statistics(const BundleResult& result);
    
    // RPC helpers
    bool send_transaction(const std::vector<uint8_t>& calldata);
    std::string call_contract(const std::string& to, const std::vector<uint8_t>& calldata);
    
    // Logging
    void log_info(const std::string& message);
    void log_error(const std::string& message);
    void log_warning(const std::string& message);
};

/**
 * UserOperation hash calculation (EIP-4337)
 */
std::string hash_user_operation(
    const UserOperation& op,
    const std::string& entry_point,
    const std::string& chain_id
);

/**
 * Validate UserOperation fields
 */
bool validate_user_operation_fields(const UserOperation& op, std::string& error);

/**
 * Parse address from hex string
 */
std::optional<std::array<uint8_t, 20>> parse_address(const std::string& hex_addr);

/**
 * Validate signature format
 */
bool validate_signature(
    const UserOperation& op,
    const std::string& sender
);

} // namespace bundler
} // namespace tigerwallet

#endif // TIGERWALLET_BUNDLER_HPP
