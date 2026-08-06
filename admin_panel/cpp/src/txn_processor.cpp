/**
 * TigerWallet Admin Panel - Transaction Processor Implementation
 * High-performance C++ implementation
 */

#include "txn_processor.hpp"
#include <iostream>
#include <fstream>
#include <thread>
#include <chrono>

namespace tiger {
namespace admin {
namespace processor {

// Example usage and test
void run_processor_test() {
    auto processor = create_processor();
    
    // Start with 4 worker threads
    processor->start(4);
    
    // Register callbacks
    processor->on_transaction_complete([](const Transaction& txn) {
        std::cout << "Transaction completed: " << txn.id << std::endl;
    });
    
    processor->on_high_risk([](const Transaction& txn, const RiskAssessment& risk) {
        std::cout << "HIGH RISK Transaction: " << txn.id 
                  << " (score: " << risk.score << ")" << std::endl;
        for (const auto& reason : risk.reasons) {
            std::cout << "  - " << reason << std::endl;
        }
    });
    
    // Submit test transactions
    for (int i = 0; i < 100; ++i) {
        Transaction txn;
        txn.id = i + 1;
        txn.user_id = (i % 10) + 1;
        txn.amount = (i + 1) * 1000000;
        txn.fee = 1000;
        txn.type = TransactionType::TRANSFER;
        txn.chain_id = 1;
        txn.from_chain_id = 1;
        txn.to_chain_id = 1;
        txn.created_at_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
            std::chrono::steady_clock::now().time_since_epoch()
        ).count();
        
        processor->submit(txn);
    }
    
    // Wait for processing
    std::this_thread::sleep_for(std::chrono::seconds(2));
    
    // Get statistics
    auto stats = processor->get_stats();
    std::cout << "\n=== Processor Statistics ===" << std::endl;
    std::cout << "Total transactions: " << stats.total_transactions << std::endl;
    std::cout << "Processed: " << stats.processed_transactions << std::endl;
    std::cout << "Flagged: " << stats.flagged_transactions << std::endl;
    std::cout << "Avg processing time: " << stats.avg_processing_time_us << " us" << std::endl;
    std::cout << "Max processing time: " << stats.max_processing_time_us << " us" << std::endl;
    
    // Get recent transactions
    auto recent = processor->get_recent_transactions(10);
    std::cout << "\n=== Recent Transactions ===" << std::endl;
    for (const auto& txn : recent) {
        std::cout << "ID: " << txn.id 
                  << ", Amount: " << txn.amount 
                  << ", Status: " << static_cast<int>(txn.status)
                  << ", Risk: " << txn.risk_score << std::endl;
    }
    
    // Stop processor
    processor->stop();
}

}  // namespace processor
}  // namespace admin
}  // namespace tiger

int main() {
    tiger::admin::processor::run_processor_test();
    return 0;
}
