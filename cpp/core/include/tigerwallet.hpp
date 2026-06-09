#ifndef TIGERWALLET_HPP
#define TIGERWALLET_HPP

#include <string>
#include <vector>
#include <memory>
#include <chrono>
#include <optional>
#include <variant>
#include <map>
#include <unordered_map>

namespace tigerwallet {

// ============================================================================
// Core Types
// ============================================================================

using Bytes = std::vector<uint8_t>;
using Address = std::string;
using ChainId = uint64_t;

// Transaction types
struct Transaction {
    ChainId chain_id;
    Address from;
    Address to;
    Bytes data;
    std::string value;  // Wei as string
    uint64_t gas_limit;
    std::string gas_price;
    uint64_t nonce;
    uint8_t v;
    Bytes r;
    Bytes s;
};

struct TransactionReceipt {
    std::string hash;
    uint64_t block_number;
    bool success;
    uint64_t gas_used;
    std::vector<Log> logs;
};

struct Log {
    Address address;
    std::vector<Bytes> topics;
    Bytes data;
};

// Wallet types
struct Wallet {
    std::string id;
    std::string name;
    Address address;
    Bytes public_key;
    Bytes private_key_encrypted;
    std::vector<ChainId> supported_chains;
    bool is_connected;
};

// ============================================================================
// Core Engine Interface
// ============================================================================

class CoreEngine {
public:
    static CoreEngine& get();
    
    // Initialization
    bool initialize(const std::string& config_path);
    void shutdown();
    
    // Wallet operations
    Wallet create_wallet(const std::string& name, const std::string& password);
    Wallet import_wallet(const std::string& seed_phrase, const std::string& password);
    bool unlock_wallet(const std::string& wallet_id, const std::string& password);
    void lock_wallet(const std::string& wallet_id);
    
    // Transaction operations
    Transaction create_transaction(ChainId chain_id, const Address& from, 
                               const Address& to, const std::string& value,
                               const Bytes& data = {});
    
    TransactionReceipt send_transaction(const Transaction& tx);
    TransactionReceipt send_transaction_sync(const Transaction& tx, uint32_t timeout_ms = 30000);
    
    // Batch operations
    std::vector<TransactionReceipt> send_batch(const std::vector<Transaction>& txs);
    
    // Signing
    Bytes sign_message(const Bytes& message, const std::string& wallet_id);
    Transaction sign_transaction(const Transaction& tx, const std::string& wallet_id);
    
    // Chain operations
    bool add_chain(ChainId chain_id, const std::string& rpc_url, 
                  const std::string& explorer);
    bool switch_chain(ChainId chain_id);
    ChainId get_current_chain() const;
    
    // Balance operations
    std::string get_balance(const Address& address, ChainId chain_id);
    std::map<std::string, std::string> get_all_balances(const Address& address);
    
    // Nonce management
    uint64_t get_nonce(const Address& address, ChainId chain_id);
    void set_nonce(const Address& address, ChainId chain_id, uint64_t nonce);
    
    // Gas operations
    std::string estimate_gas(const Transaction& tx);
    std::string get_gas_price(ChainId chain_id);
    
private:
    CoreEngine() = default;
    ~CoreEngine() = default;
    CoreEngine(const CoreEngine&) = delete;
    CoreEngine& operator=(const CoreEngine&) = delete;
};

// ============================================================================
// High-Performance Transaction Pool
// ============================================================================

class TransactionPool {
public:
    TransactionPool() = default;
    
    // Pool operations
    void add_pending(const Transaction& tx);
    void confirm(const std::string& hash, uint64_t nonce);
    void replace(const std::string& old_hash, const Transaction& new_tx);
    void cancel(const Address& from, uint64_t nonce);
    
    // Queries
    std::optional<Transaction> get_pending(const std::string& hash) const;
    std::vector<Transaction> get_pending_for(const Address& from) const;
    size_t pending_count() const;
    
    // Optimization
    void sort_by_gas_price();
    void prune_old(uint64_t max_age_blocks);

private:
    struct PendingTx {
        Transaction tx;
        uint64_t added_at;
        uint64_t replaced_by;
    };
    std::unordered_map<std::string, PendingTx> pool_;
    std::multimap<uint64_t, std::string> by_nonce_;
    std::multimap<uint64_t, std::string> by_gas_;
};

// ============================================================================
// RPC Client (High-Performance)
// ============================================================================

class RPCClient {
public:
    RPCClient(const std::string& url, uint32_t max_connections = 100);
    ~RPCClient();
    
    // Synchronous calls
    std::variant<TransactionReceipt, std::string> 
    send_raw_transaction(const Bytes& tx);
    
    std::variant<std::string, std::string>
    call(const std::string& method, const std::vector<std::string>& params);
    
    std::variant<std::string, std::string>
    get_balance(const Address& address);
    
    std::variant<uint64_t, std::string>
    get_transaction_count(const Address& address);
    
    std::variant<std::string, std::string>
    get_gas_price();
    
    // Batch operations
    // std::vector<std::variant<json::Value, std::string>>
    // batch_call(const std::vector<std::pair<std::string, std::vector<std::string>>& calls);

private:
    class Impl;
    std::unique_ptr<Impl> impl_;
};

// ============================================================================
// Utility Functions
// ============================================================================

std::string to_hex(const Bytes& data);
Bytes from_hex(const std::string& hex);
std::string to_wei(const std::string& eth, uint8_t decimals = 18);
std::string from_wei(const std::string& wei, uint8_t decimals = 18);

bool is_valid_address(const Address& address, ChainId chain_id);
Address create_address(const Bytes& public_key);

} // namespace tigerwallet

#endif // TIGERWALLET_HPP
