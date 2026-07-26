/**
 * Transaction data structures
 */

#ifndef TRANSACTION_HPP
#define TRANSACTION_HPP

#include <array>
#include <cstdint>
#include <string>
#include <vector>
#include <chrono>

namespace tiger {

// 256-bit integer type (simplified)
using uint256 = std::array<uint8_t, 32>;

struct Transaction {
    std::string hash;
    std::string from;
    std::string to;
    uint256 value;
    uint256 gas_price;
    uint64_t gas_limit;
    uint64_t gas_used;
    std::string data;
    uint64_t nonce;
    uint64_t chain_id;
    std::string v;
    std::string r;
    std::string s;
    
    // Additional metadata
    std::chrono::steady_clock::time_point received_at;
    std::string raw;
    bool is_private;
    std::string relayer;
    
    Transaction() 
        : gas_limit(21000),
          gas_used(0),
          nonce(0),
          chain_id(1),
          is_private(false) {
        value.fill(0);
        gas_price.fill(0);
        received_at = std::chrono::steady_clock::now();
    }
};

struct TransactionReceipt {
    std::string transaction_hash;
    std::string block_hash;
    uint64_t block_number;
    uint64_t gas_used;
    std::string status;
    std::vector<Log> logs;
    
    TransactionReceipt() : block_number(0), gas_used(0) {}
};

struct Log {
    std::string address;
    std::vector<std::string> topics;
    std::string data;
    uint64_t log_index;
};

struct Block {
    uint64_t number;
    std::string hash;
    std::string parent_hash;
    std::string miner;
    uint256 base_fee_per_gas;
    uint64_t gas_limit;
    uint64_t gas_used;
    std::string timestamp;
    std::vector<Transaction> transactions;
    
    Block() : number(0), gas_limit(0), gas_used(0) {}
};

}  // namespace tiger

#endif  // TRANSACTION_HPP
