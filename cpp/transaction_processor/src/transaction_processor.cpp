/**
 * TigerWallet - High-Performance Transaction Processor Implementation
 * C++ Implementation for Ultra-Low Latency
 */

#include "transaction_processor.hpp"
#include "hasher.hpp"
#include "signature.hpp"
#include <iostream>
#include <sstream>
#include <iomanip>

namespace tiger {

// ============================================================================
// Transaction Processor Implementation
// ============================================================================

TransactionProcessor::TransactionProcessor(size_t worker_threads) {
    // Pre-allocate memory for transaction pool
    pool_.transactions.reserve(constants::MAX_POOL_SIZE);
    
    // Pre-allocate worker threads
    for (size_t i = 0; i < worker_threads; ++i) {
        workers_.emplace_back(&TransactionProcessor::worker_loop, this);
    }
    
    std::cout << "[TigerTxProc] Initialized with " << worker_threads << " workers" << std::endl;
}

TransactionProcessor::~TransactionProcessor() {
    stop();
}

// ============================================================================
// Transaction Submission
// ============================================================================

std::optional<Hash> TransactionProcessor::submit_transaction(const Transaction& tx) {
    if (!running_.load()) {
        return std::nullopt;
    }
    
    // Validate transaction first
    auto validation_result = validate_transaction(tx);
    if (validation_result != ValidationResult::VALID) {
        stats_.total_failed.fetch_add(1);
        return std::nullopt;
    }
    
    // Calculate hash
    auto hash = tx.calculate_hash();
    
    // Add to pool
    {
        std::unique_lock lock(pool_.mutex);
        if (pool_.transactions.size() >= constants::MAX_POOL_SIZE) {
            return std::nullopt;
        }
        
        Transaction tx_copy = tx;
        tx_copy.hash = hash;
        pool_.transactions[hash] = std::move(tx_copy);
        pool_.pending_queue.push(pool_.transactions[hash]);
    }
    
    stats_.pending_count.fetch_add(1);
    stats_.pool_size.fetch_add(1);
    
    return hash;
}

std::vector<std::optional<Hash>> TransactionProcessor::submit_transactions(
    const std::vector<Transaction>& txs
) {
    std::vector<std::optional<Hash>> results;
    results.reserve(txs.size());
    
    // Batch submission with parallel validation
    std::vector<std::future<std::optional<Hash>>> futures;
    
    for (const auto& tx : txs) {
        futures.push_back(std::async(std::launch::async, [this, &tx]() {
            return submit_transaction(tx);
        }));
    }
    
    for (auto& future : futures) {
        results.push_back(future.get());
    }
    
    return results;
}

// ============================================================================
// Transaction Processing
// ============================================================================

size_t TransactionProcessor::process_pending(size_t max_transactions) {
    if (!running_.load()) {
        return 0;
    }
    
    size_t processed = 0;
    auto start_time = std::chrono::high_resolution_clock::now();
    
    // Get top transactions from pool
    std::vector<Transaction> txs_to_process;
    {
        std::unique_lock lock(pool_.mutex);
        txs_to_process = pool_.get_top(std::min(max_transactions, (size_t)100));
    }
    
    // Process in parallel
    std::vector<std::future<ValidationResult>> futures;
    for (auto& tx : txs_to_process) {
        futures.push_back(std::async(std::launch::async, [this, &tx]() {
            return process_transaction(tx);
        }));
    }
    
    // Collect results
    for (size_t i = 0; i < futures.size(); ++i) {
        auto result = futures[i].get();
        
        if (result == ValidationResult::VALID) {
            // Execute transaction
            auto receipt = execute_transaction(txs_to_process[i]);
            if (receipt) {
                {
                    std::unique_lock lock(receipts_mutex_);
                    receipts_[txs_to_process[i].hash] = *receipt;
                }
                
                // Apply state changes
                apply_state_changes(txs_to_process[i], *receipt);
                processed++;
            }
        }
        
        // Remove from pool
        {
            std::unique_lock lock(pool_.mutex);
            pool_.remove(txs_to_process[i].hash);
        }
    }
    
    auto end_time = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end_time - start_time);
    
    update_stats(duration.count(), processed > 0);
    
    return processed;
}

ValidationResult TransactionProcessor::process_transaction(const Transaction& tx) {
    auto start = std::chrono::high_resolution_clock::now();
    
    // Validation
    auto result = validate_transaction(tx);
    if (result != ValidationResult::VALID) {
        return result;
    }
    
    // Execute
    auto receipt = execute_transaction(tx);
    if (!receipt) {
        return ValidationResult::INVALID_SENDER;
    }
    
    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);
    
    tx.validation_time = duration.count();
    
    stats_.total_validated.fetch_add(1);
    
    return ValidationResult::VALID;
}

// ============================================================================
// Validation
// ============================================================================

ValidationResult TransactionProcessor::validate_transaction(const Transaction& tx) const {
    // Check sender exists and has sufficient balance
    {
        std::shared_lock lock(state_mutex_);
        auto it = state_db_.find(tx.from);
        if (it == state_db_.end()) {
            return ValidationResult::INVALID_SENDER;
        }
        
        const auto& account = it->second;
        
        // Check balance
        uint256_t total_cost = tx.value + (tx.gas_limit * tx.gas_price);
        if (account.balance < total_cost) {
            return ValidationResult::INSUFFICIENT_BALANCE;
        }
        
        // Check nonce
        if (tx.nonce < account.nonce) {
            return ValidationResult::NONCE_TOO_LOW;
        }
    }
    
    // Check gas limits
    if (tx.gas_limit < 21000 || tx.gas_limit > 30000000) {
        return ValidationResult::GAS_LIMIT_TOO_HIGH;
    }
    
    if (tx.gas_price < constants::MIN_GAS_PRICE) {
        return ValidationResult::GAS_PRICE_TOO_LOW;
    }
    
    // Verify signature (simplified)
    // In production, would use full signature verification
    auto sender = tx.get_sender();
    if (sender == Address{}) {
        return ValidationResult::INVALID_SIGNATURE;
    }
    
    return ValidationResult::VALID;
}

// ============================================================================
// Execution
// ============================================================================

std::optional<TransactionReceipt> TransactionProcessor::execute_transaction(
    const Transaction& tx
) {
    TransactionReceipt receipt;
    receipt.transaction_hash = tx.hash;
    receipt.block_number = 0; // Would be set by block builder
    receipt.gas_used = tx.gas_limit;
    receipt.status = 1; // Success
    
    // Simplified execution - in production would run EVM
    // For now, just record the transaction
    
    receipt.effective_gas_price = tx.gas_price;
    
    // Calculate bloom filter for logs
    // Simplified - would calculate real bloom
    
    return receipt;
}

void TransactionProcessor::apply_state_changes(
    const Transaction& tx,
    const TransactionReceipt& receipt
) {
    std::unique_lock lock(state_mutex_);
    
    // Deduct value and gas from sender
    auto& sender_state = state_db_[tx.from];
    uint256_t gas_cost = receipt.gas_used * receipt.effective_gas_price;
    sender_state.balance -= (tx.value + gas_cost);
    sender_state.nonce = tx.nonce + 1;
    
    // Add value to receiver (or create contract)
    if (tx.to != Address{}) {
        auto& receiver_state = state_db_[tx.to];
        receiver_state.balance += tx.value;
    }
    
    stats_.total_gas_used.fetch_add(receipt.gas_used);
    stats_.total_processed.fetch_add(1);
}

void TransactionProcessor::update_stats(uint64_t processing_time_us, bool success) {
    if (!success) {
        stats_.total_failed.fetch_add(1);
        return;
    }
    
    // Update average processing time
    auto current_avg = stats_.avg_processing_time_us.load();
    auto total = stats_.total_validated.load();
    if (total > 0) {
        auto new_avg = ((current_avg * (total - 1)) + processing_time_us) / total;
        stats_.avg_processing_time_us.store(new_avg);
    }
}

// ============================================================================
// State Management
// ============================================================================

std::optional<AccountState> TransactionProcessor::get_account_state(
    const Address& address
) const {
    std::shared_lock lock(state_mutex_);
    
    auto it = state_db_.find(address);
    if (it != state_db_.end()) {
        return it->second;
    }
    
    return std::nullopt;
}

void TransactionProcessor::update_account_state(
    const Address& address,
    const AccountState& state
) {
    std::unique_lock lock(state_mutex_);
    state_db_[address] = state;
}

// ============================================================================
// Query Methods
// ============================================================================

std::optional<Transaction> TransactionProcessor::get_transaction(
    const Hash& hash
) const {
    std::shared_lock lock(pool_.mutex);
    
    auto it = pool_.transactions.find(hash);
    if (it != pool_.transactions.end()) {
        return it->second;
    }
    
    return std::nullopt;
}

std::optional<TransactionReceipt> TransactionProcessor::get_receipt(
    const Hash& hash
) const {
    std::shared_lock lock(receipts_mutex_);
    
    auto it = receipts_.find(hash);
    if (it != receipts_.end()) {
        return it->second;
    }
    
    return std::nullopt;
}

size_t TransactionProcessor::pending_count() const {
    return stats_.pending_count.load();
}

// ============================================================================
// Block Building
// ============================================================================

Block TransactionProcessor::build_block(
    const Address& coinbase,
    uint64_t gas_limit
) {
    Block block;
    block.number = 0; // Would get from chain
    block.coinbase = coinbase;
    block.timestamp = std::chrono::duration_cast<std::chrono::seconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    // Get transactions within gas limit
    uint64_t used_gas = 0;
    std::vector<Transaction> selected_txs;
    
    {
        std::unique_lock lock(pool_.mutex);
        
        auto txs = pool_.get_top(100);
        for (const auto& tx : txs) {
            if (used_gas + tx.gas_limit > gas_limit) {
                break;
            }
            
            // Re-validate
            if (validate_transaction(tx) == ValidationResult::VALID) {
                selected_txs.push_back(tx);
                used_gas += tx.gas_limit;
            }
        }
    }
    
    block.transactions = std::move(selected_txs);
    
    // Set gas used
    std::string gas_used_str = std::to_string(used_gas);
    std::memcpy(block.gas_used.data(), gas_used_str.data(), 
                std::min(gas_used_str.size(), (size_t)32));
    
    // Calculate roots
    block.transactions_root = block.get_transactions_root();
    block.parent_hash.fill(0); // Would get from previous block
    
    return block;
}

// ============================================================================
// Worker Loop
// ============================================================================

void TransactionProcessor::worker_loop() {
    while (running_.load()) {
        // Process pending transactions
        process_pending(10);
        
        // Sleep briefly to prevent busy waiting
        std::this_thread::sleep_for(std::chrono::microseconds(100));
    }
}

// ============================================================================
// Lifecycle
// ============================================================================

void TransactionProcessor::start() {
    running_.store(true);
    std::cout << "[TigerTxProc] Started processing" << std::endl;
}

void TransactionProcessor::stop() {
    running_.store(false);
    
    // Wait for workers to finish
    for (auto& worker : workers_) {
        if (worker.joinable()) {
            worker.join();
        }
    }
    
    std::cout << "[TigerTxProc] Stopped processing" << std::endl;
}

// ============================================================================
// Transaction Pool Implementation
// ============================================================================

void TransactionProcessor::TransactionPool::add(const Transaction& tx) {
    transactions[tx.hash] = tx;
    pending_queue.push(tx);
}

bool TransactionProcessor::TransactionPool::remove(const Hash& hash) {
    return transactions.erase(hash) > 0;
}

std::vector<Transaction> TransactionProcessor::TransactionPool::get_top(size_t count) {
    std::vector<Transaction> result;
    result.reserve(count);
    
    std::unordered_set<Hash, std::hash<Hash>> seen;
    
    while (!pending_queue.empty() && result.size() < count) {
        auto tx = pending_queue.top();
        pending_queue.pop();
        
        if (seen.find(tx.hash) == seen.end()) {
            seen.insert(tx.hash);
            result.push_back(tx);
        }
    }
    
    return result;
}

size_t TransactionProcessor::TransactionPool::size() const {
    return transactions.size();
}

} // namespace tiger

// ============================================================================
// Main Entry Point
// ============================================================================

int main() {
    std::cout << "TigerWallet High-Performance Transaction Processor" << std::endl;
    std::cout << "===============================================" << std::endl;
    
    // Create processor with optimal thread count
    size_t optimal_threads = std::thread::hardware_concurrency();
    if (optimal_threads < 4) optimal_threads = 4;
    if (optimal_threads > 32) optimal_threads = 32;
    
    tiger::TransactionProcessor processor(optimal_threads);
    
    // Start processing
    processor.start();
    
    // Create test transaction
    tiger::Transaction tx;
    tx.chain_id = 1;
    tx.nonce = 0;
    tx.gas_price = 20000000000; // 20 gwei
    tx.gas_limit = 21000;
    tx.value = 1000000000000000000; // 1 ETH
    tx.type = tiger::TransactionType::EIP1559;
    tx.data = {0x00};
    tx.timestamp = std::time(nullptr);
    
    // Submit transaction
    auto hash = processor.submit_transaction(tx);
    
    if (hash) {
        std::cout << "Transaction submitted: 0x" << std::endl;
        for (auto b : *hash) {
            std::cout << std::hex << std::setw(2) << std::setfill('0') << (int)b;
        }
        std::cout << std::dec << std::endl;
    }
    
    // Process pending
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    auto processed = processor.process_pending(10);
    std::cout << "Processed " << processed << " transactions" << std::endl;
    
    // Print stats
    const auto& stats = processor.get_stats();
    std::cout << "Total processed: " << stats.total_processed.load() << std::endl;
    std::cout << "Total validated: " << stats.total_validated.load() << std::endl;
    std::cout << "Average processing time: " << stats.avg_processing_time_us.load() << " us" << std::endl;
    
    // Stop processor
    processor.stop();
    
    return 0;
}
