/**
 * TigerWallet Transaction Processor Implementation
 * High-performance C++17 transaction processing
 */

#include "transaction_processor.hpp"
#include <iostream>
#include <random>
#include <cstring>

namespace tiger_wallet {

// ============================================================================
// Main Function - Example Usage
// ============================================================================

int main() {
    std::cout << "TigerWallet High-Performance Transaction Processor" << std::endl;
    std::cout << "==================================================" << std::endl;
    
    // Configure processor
    TransactionProcessorConfig config;
    config.max_pool_size = 100000;
    config.worker_threads = 8;
    config.max_pending_transactions = 50000;
    config.enable_prioritization = true;
    config.enable_gas_optimization = true;
    config.enable_deduplication = true;
    
    // Create processor
    auto processor = create_transaction_processor(config);
    
    // Start processing
    processor->start();
    
    std::cout << "Processor started with " << config.worker_threads << " workers" << std::endl;
    
    // Example: Submit sample transactions
    for (int i = 0; i < 10; ++i) {
        auto tx = std::make_shared<Transaction>();
        
        // Generate random hash
        std::random_device rd;
        std::mt19937_64 gen(rd());
        std::uniform_int_distribution<uint64_t> dis(0, UINT64_MAX);
        
        std::array<uint8_t, 32> hash_data{};
        for (int j = 0; j < 32; ++j) {
            hash_data[j] = static_cast<uint8_t>(dis(gen) & 0xFF);
        }
        tx->hash = {hash_data};
        
        tx->type = TransactionType::EVM;
        tx->status = TransactionStatus::PENDING;
        tx->from = "0x742d35Cc6634C0532925a3b844Bc9e7595f8a1E1";
        tx->to = "0x8Ba1f109551bD432803012645Ac136ddd64DBA72";
        tx->nonce = std::to_string(i);
        tx->value = "1000000000000000000"; // 1 ETH
        tx->gas_price = "20000000000"; // 20 Gwei
        tx->gas_limit = "21000";
        tx->chain_id = 1;
        
        auto result = processor->submit_transaction(tx);
        if (result) {
            std::cout << "Submitted transaction: " << result->to_hex() << std::endl;
        }
    }
    
    // Wait for processing
    std::this_thread::sleep_for(std::chrono::seconds(2));
    
    // Get statistics
    auto stats = processor->get_stats();
    std::cout << "\nProcessor Statistics:" << std::endl;
    std::cout << "  Total Processed: " << stats.total_processed << std::endl;
    std::cout << "  Total Validated: " << stats.total_validated << std::endl;
    std::cout << "  Total Failed: " << stats.total_failed << std::endl;
    std::cout << "  Pending Count: " << stats.pending_count << std::endl;
    std::cout << "  Validated Count: " << stats.validated_count << std::endl;
    
    // Stop processor
    processor->stop();
    std::cout << "\nProcessor stopped." << std::endl;
    
    return 0;
}

} // namespace tiger_wallet
