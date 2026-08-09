// ============================================================================
// TIGERWALLET HIGH-PERFORMANCE C++ CORE
// Ultra-low latency cryptographic operations and data structures
// ============================================================================

#ifndef TIGER_WALLET_CORE_HPP
#define TIGER_WALLET_CORE_HPP

#include <array>
#include <cstdint>
#include <cstring>
#include <string>
#include <vector>
#include <optional>
#include <memory>
#include <functional>
#include <atomic>
#include <mutex>
#include <shared_mutex>

namespace tiger {

// ============================================================================
// Constants
// ============================================================================

constexpr size_t ADDRESS_LENGTH = 40;
constexpr size_t ADDRESS_LENGTH_WITH_PREFIX = 42;
constexpr size_t PRIVATE_KEY_LENGTH = 64;
constexpr size_t MNEMONIC_WORD_COUNT = 24;
constexpr size_t MAX_CHAINS = 100;

// Chain IDs
constexpr int64_t CHAIN_ETHEREUM = 1;
constexpr int64_t CHAIN_BSC = 56;
constexpr int64_t CHAIN_POLYGON = 137;
constexpr int64_t CHAIN_ARBITRUM = 42161;
constexpr int64_t CHAIN_OPTIMISM = 10;
constexpr int64_t CHAIN_BASE = 8453;
constexpr int64_t CHAIN_AVALANCHE = 43114;

// ============================================================================
// Error Types
// ============================================================================

enum class ErrorCode : uint32_t {
    SUCCESS = 0,
    INVALID_ADDRESS = 1,
    INVALID_PRIVATE_KEY = 2,
    INVALID_MNEMONIC = 3,
    ENCRYPTION_FAILED = 4,
    DECRYPTION_FAILED = 5,
    SIGNATURE_FAILED = 6,
    KEY_DERIVATION_FAILED = 7,
    VALIDATION_FAILED = 8,
    NETWORK_ERROR = 9,
    TIMEOUT = 10,
    NOT_FOUND = 11,
    INSUFFICIENT_FUNDS = 12,
    INVALID_CHAIN = 13,
    BUFFER_OVERFLOW = 14,
    INVALID_PARAMETER = 15,
};

struct Result {
    ErrorCode code;
    std::string message;
    
    Result() : code(ErrorCode::SUCCESS) {}
    Result(ErrorCode c) : code(c) {}
    Result(ErrorCode c, const std::string& msg) : code(c), message(msg) {}
    
    bool isSuccess() const { return code == ErrorCode::SUCCESS; }
    operator bool() const { return isSuccess(); }
};

// ============================================================================
// Chain Configuration
// ============================================================================

struct ChainConfig {
    int64_t chain_id;
    char name[32];
    char symbol[8];
    char rpc_url[256];
    char explorer_url[128];
    char explorer_api[256];
    uint32_t block_time_ms;
    uint8_t is_evm : 1;
    uint8_t is_active : 1;
};

// ============================================================================
// Address Types
// ============================================================================

struct alignas(32) Address {
    std::array<uint8_t, 20> bytes;
    
    Address() { bytes.fill(0); }
    
    explicit Address(const std::array<uint8_t, 20>& b) : bytes(b) {}
    
    std::string toHexString() const {
        std::string result = "0x";
        result.reserve(42);
        for (const auto& byte : bytes) {
            const char hex_chars[] = "0123456789abcdef";
            result += hex_chars[byte >> 4];
            result += hex_chars[byte & 0x0F];
        }
        return result;
    }
    
    bool isZero() const {
        for (const auto& byte : bytes) {
            if (byte != 0) return false;
        }
        return true;
    }
    
    bool operator==(const Address& other) const {
        return bytes == other.bytes;
    }
};

// ============================================================================
// Transaction Types
// ============================================================================

struct alignas(32) Transaction {
    Address to;
    uint256_t value;
    uint256_t nonce;
    uint256_t gas_price;
    uint256_t max_fee;
    uint256_t max_priority_fee;
    uint64_t gas_limit;
    uint64_t chain_id;
    std::vector<uint8_t> data;
    std::array<uint8_t, 32> tx_hash;
    std::array<uint8_t, 65> signature;
    
    Transaction() : value(0), nonce(0), gas_price(0), max_fee(0), max_priority_fee(0), gas_limit(21000), chain_id(1) {}
};

// Safe unsigned 256-bit integer
struct uint256_t {
    std::array<uint64_t, 4> words;
    
    uint256_t() { words.fill(0); }
    uint256_t(uint64_t v) { words.fill(0); words[0] = v; }
    
    uint64_t low64() const { return words[0]; }
    uint64_t high64() const { return words[3]; }
    
    bool isZero() const {
        for (const auto& w : words) {
            if (w != 0) return false;
        }
        return true;
    }
    
    std::string toHexString() const {
        std::string result = "0x";
        bool started = false;
        for (int i = 3; i >= 0; --i) {
            if (!started && words[i] == 0) continue;
            started = true;
            char buf[17];
            snprintf(buf, sizeof(buf), "%016lx", words[i]);
            result += buf;
        }
        if (!started) result += "0";
        return result;
    }
    
    double toDouble() const {
        double result = 0;
        double multiplier = 1;
        for (int i = 0; i < 4; ++i) {
            result += static_cast<double>(words[i]) * multiplier;
            multiplier *= 18446744073709551616.0;
        }
        return result;
    }
};

// ============================================================================
// Token Types
// ============================================================================

struct Token {
    Address address;
    char symbol[8];
    char name[32];
    uint8_t decimals;
    int64_t chain_id;
    double price_usd;
    
    Token() : decimals(18), chain_id(1), price_usd(0) {
        symbol[0] = 0;
        name[0] = 0;
    }
};

// ============================================================================
// Balance Types
// ============================================================================

struct Balance {
    Address address;
    int64_t chain_id;
    double native_balance;
    double native_balance_usd;
    char symbol[8];
    std::vector<TokenBalance> tokens;
    
    Balance() : chain_id(1), native_balance(0), native_balance_usd(0) {
        symbol[0] = 0;
    }
};

struct TokenBalance {
    Token token;
    double balance;
    double balance_usd;
};

// ============================================================================
// Staking Types
// ============================================================================

struct StakingPool {
    std::string id;
    std::string name;
    std::string token;
    int64_t chain_id;
    double apy;
    double min_stake;
    double total_staked;
    double tvl;
    std::string description;
};

struct StakingPosition {
    std::string id;
    std::string pool_id;
    std::string pool_name;
    std::string token;
    double amount;
    double reward;
    double apy;
    std::string status;  // "active", "unbonding", "claimed"
    int64_t start_time;
    int64_t unlock_time;
};

// ============================================================================
// Lending Types
// ============================================================================

struct LendingMarket {
    int64_t id;
    Address asset;
    char symbol[8];
    int64_t chain_id;
    double total_supply;
    double total_borrow;
    double supply_apy;
    double borrow_apy;
    double utilization;
    double ltv;
    double liquidation_threshold;
};

struct LendingPosition {
    double health_factor;
    double total_collateral;
    double total_debt;
    std::vector<LendingSupply> supplies;
    std::vector<LendingBorrow> borrows;
};

struct LendingSupply {
    char asset[8];
    double amount;
    double apy;
    double value_usd;
};

struct LendingBorrow {
    char asset[8];
    double amount;
    double apy;
    double value_usd;
};

// ============================================================================
// Bridge Types
// ============================================================================

struct BridgeRoute {
    std::string id;
    std::string name;
    std::string logo;
    double fee;
    std::string time;
    double min_amount;
    double max_amount;
    double reliability;
    int64_t from_chain;
    int64_t to_chain;
};

struct BridgeQuote {
    int64_t from_chain;
    int64_t to_chain;
    std::string token;
    double amount;
    double received_amount;
    double bridge_fee;
    double network_fee;
    std::string estimated_time;
    double rate;
    std::string route;
};

struct BridgeTransfer {
    std::string id;
    int64_t from_chain;
    int64_t to_chain;
    std::string token;
    double amount;
    std::string status;
    int64_t timestamp;
    std::string source_tx_hash;
    std::string dest_tx_hash;
};

// ============================================================================
// Swap Types
// ============================================================================

struct SwapToken {
    Address address;
    char symbol[8];
    char name[32];
    uint8_t decimals;
    int64_t chain_id;
    bool is_native;
    bool is_stable;
    double price_usd;
    std::string logo_uri;
};

struct SwapQuote {
    std::string input_token;
    std::string output_token;
    double input_amount;
    double output_amount;
    double minimum_out;
    double price_impact;
    double gas_estimate;
    double gas_fee_usd;
    double exchange_rate;
    int64_t expires_at;
    std::vector<SwapRouteStep> route;
};

struct SwapRouteStep {
    std::string dex;
    std::string pool_address;
    uint32_t fee;
    double amount_in;
    double amount_out;
};

// ============================================================================
// NFT Types
// ============================================================================

struct NFTCollection {
    std::string id;
    std::string name;
    std::string symbol;
    Address contract_address;
    int64_t chain_id;
    int32_t total_supply;
    double floor_price;
    double volume_24h;
    std::string image_url;
};

struct NFTItem {
    std::string id;
    std::string token_id;
    Address contract_address;
    std::string name;
    std::string description;
    std::string image_url;
    std::string animation_url;
    std::vector<NFTAttribute> attributes;
    Address owner;
    double price;
    std::string price_token;
    int64_t chain_id;
};

struct NFTAttribute {
    std::string trait_type;
    std::string value;
    std::string rarity;
};

// ============================================================================
// Core Interface
// ============================================================================

class WalletCore {
public:
    static WalletCore& getInstance();
    
    // Initialization
    Result initialize();
    Result shutdown();
    
    // Address operations
    Result validateAddress(const char* address, int64_t chain_id);
    Result parseAddress(const char* address, Address& out);
    std::string addressToHex(const Address& addr);
    
    // Balance operations
    Result getBalance(const Address& address, int64_t chain_id, Balance& balance);
    Result getTokenBalances(const Address& address, int64_t chain_id, std::vector<TokenBalance>& tokens);
    
    // Transaction operations
    Result createTransaction(
        const Address& from,
        const Address& to,
        uint64_t chain_id,
        const char* value,
        const char* data,
        uint64_t gas_limit,
        Transaction& tx
    );
    
    Result signTransaction(Transaction& tx, const uint8_t* private_key, size_t key_len);
    Result encodeTransaction(const Transaction& tx, std::vector<uint8_t>& encoded);
    
    // Staking operations
    Result getStakingPools(std::vector<StakingPool>& pools);
    Result getStakingPositions(const Address& address, std::vector<StakingPosition>& positions);
    Result stake(const Address& address, const char* pool_id, double amount, std::string& tx_hash);
    Result unstake(const char* position_id, double amount, std::string& tx_hash);
    Result claimRewards(const char* position_id, std::string& tx_hash);
    
    // Lending operations
    Result getLendingMarkets(int64_t chain_id, std::vector<LendingMarket>& markets);
    Result getLendingPosition(const Address& address, LendingPosition& position);
    Result lendingSupply(const Address& address, const char* asset, double amount, std::string& tx_hash);
    Result lendingWithdraw(const Address& address, const char* asset, double amount, std::string& tx_hash);
    Result lendingBorrow(const Address& address, const char* asset, double amount, std::string& tx_hash);
    Result lendingRepay(const Address& address, const char* asset, double amount, std::string& tx_hash);
    
    // Bridge operations
    Result getBridgeRoutes(int64_t from_chain, int64_t to_chain, std::vector<BridgeRoute>& routes);
    Result getBridgeQuote(int64_t from_chain, int64_t to_chain, const char* token, double amount, BridgeQuote& quote);
    Result executeBridge(const Address& address, const BridgeQuote& quote, const char* dest_address, std::string& transfer_id);
    Result getBridgeHistory(const Address& address, std::vector<BridgeTransfer>& history);
    
    // Swap operations
    Result getSwapTokens(int64_t chain_id, std::vector<SwapToken>& tokens);
    Result getSwapQuote(const char* token_in, const char* token_out, double amount, int64_t chain_id, SwapQuote& quote);
    Result executeSwap(const Address& address, const SwapQuote& quote, std::string& tx_hash);
    
    // NFT operations
    Result getNFTCollections(int64_t chain_id, std::vector<NFTCollection>& collections);
    Result getNFTItems(const char* collection_id, int64_t chain_id, std::vector<NFTItem>& items);
    Result buyNFT(const Address& address, const char* nft_id, double price, const char* price_token, std::string& tx_hash);
    Result sellNFT(const Address& address, const char* nft_id, double price, const char* price_token, std::string& tx_hash);
    
    // Utility
    Result hashData(const uint8_t* data, size_t len, std::array<uint8_t, 32>& hash);
    uint64_t getCurrentTimeMs();
    
private:
    WalletCore() = default;
    ~WalletCore() = default;
    WalletCore(const WalletCore&) = delete;
    WalletCore& operator=(const WalletCore&) = delete;
    
    std::atomic<bool> initialized_{false};
    mutable std::shared_mutex mutex_;
    
    ChainConfig chains_[MAX_CHAINS];
    size_t chain_count_ = 0;
    
    void initChains();
    ChainConfig* findChain(int64_t chain_id);
};

// ============================================================================
// Inline Implementations
// ============================================================================

inline WalletCore& WalletCore::getInstance() {
    static WalletCore instance;
    return instance;
}

inline uint64_t WalletCore::getCurrentTimeMs() {
    auto now = std::chrono::system_clock::now();
    auto duration = now.time_since_epoch();
    return std::chrono::duration_cast<std::chrono::milliseconds>(duration).count();
}

} // namespace tiger

#endif // TIGER_WALLET_CORE_HPP
