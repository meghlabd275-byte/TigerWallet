/**
 * TigerWallet DEX Connectors
 * High-Performance C++ Implementation for Real DEX Integration
 * Supports: Uniswap V2/V3, SushiSwap, PancakeSwap, QuickSwap, Curve, Balancer
 */

#ifndef DEX_CONNECTOR_HPP
#define DEX_CONNECTOR_HPP

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

// ============================================================================
// Configuration
// ============================================================================

constexpr size_t MAX_PENDING_ORDERS = 10000;
constexpr size_t MAX_RETRY_ATTEMPTS = 3;
constexpr auto DEFAULT_TIMEOUT = std::chrono::seconds(30);
constexpr auto PRICE_CACHE_TTL = std::chrono::milliseconds(500);

// ============================================================================
// Types
// ============================================================================

using Timestamp = int64_t;
using Millis = std::chrono::milliseconds;

enum class DexProtocol {
    UNISWAP_V2,
    UNISWAP_V3,
    SUSHISWAP,
    PANCAKESWAP,
    QUICKSWAP,
    CURVE,
    BALANCER,
    JUPITER,
    RAYDIUM,
    ORCA
};

enum class OrderType {
    MARKET,
    LIMIT,
    GTC,  // Good Till Cancel
    FOK,  // Fill Or Kill
    IOC   // Immediate Or Cancel
};

enum class OrderSide {
    BUY,
    SELL
};

enum class OrderStatus {
    PENDING,
    OPEN,
    FILLED,
    PARTIALLY_FILLED,
    CANCELLED,
    REJECTED,
    EXPIRED
};

// ============================================================================
// Data Structures
// ============================================================================

struct Token {
    std::string address;
    std::string symbol;
    std::string name;
    uint8_t decimals;
    
    Token() : decimals(18) {}
    Token(const std::string& addr, const std::string& sym, const std::string& n, uint8_t dec)
        : address(addr), symbol(sym), name(n), decimals(dec) {}
};

struct TokenAmount {
    std::string token_address;
    std::string amount;  // Raw amount as string (big number)
    double amount_usd;   // Approximate USD value
    
    TokenAmount() : amount_usd(0.0) {}
};

struct Pool {
    std::string address;
    Token token_a;
    Token token_b;
    std::string reserve_a;
    std::string reserve_b;
    double fee_bps;  // Fee in basis points (e.g., 30 = 0.3%)
    uint24_t pool_id;  // For Uniswap V3
    
    Pool() : fee_bps(0.0), pool_id(0) {}
};

struct Order {
    std::string order_id;
    std::string pool_address;
    OrderSide side;
    OrderType type;
    OrderStatus status;
    TokenAmount from_token;
    TokenAmount to_token;
    std::string limit_price;
    double fill_amount;
    double executed_amount;
    Timestamp created_at;
    Timestamp expires_at;
    
    Order() : side(OrderSide::BUY), type(OrderType::MARKET), 
              status(OrderStatus::PENDING), fill_amount(0), 
              executed_amount(0), created_at(0), expires_at(0) {}
};

struct SwapResult {
    bool success;
    std::string order_id;
    std::string from_token;
    std::string to_token;
    std::string from_amount;
    std::string to_amount;
    std::string tx_hash;
    double price_impact_bps;
    double gas_estimate;
    std::string error_message;
    
    SwapResult() : success(false), price_impact_bps(0), gas_estimate(0) {}
};

struct Quote {
    std::string from_token;
    std::string to_token;
    std::string from_amount;
    std::string to_amount;
    double price;
    double price_impact_bps;
    double gas_estimate;
    std::vector<std::string> route;  // DEX names
    
    Quote() : price(0), price_impact_bps(0), gas_estimate(0) {}
};

struct PriceInfo {
    std::string token_a;
    std::string token_b;
    double price;
    double bid;
    double ask;
    double spread_bps;
    Timestamp timestamp;
    
    PriceInfo() : price(0), bid(0), ask(0), spread_bps(0), timestamp(0) {}
};

struct LiquidityPosition {
    std::string pool_address;
    std::string token_a;
    std::string token_b;
    std::string liquidity_tokens;
    std::string token_a_amount;
    std::string token_b_amount;
    double value_usd;
    
    LiquidityPosition() : value_usd(0) {}
};

// ============================================================================
// DEX Connector Interface
// ============================================================================

class DexConnector {
public:
    virtual ~DexConnector() = default;
    
    // Connection management
    virtual bool connect() = 0;
    virtual void disconnect() = 0;
    virtual bool is_connected() const = 0;
    
    // Pool/Token discovery
    virtual std::vector<Pool> get_pools(const std::string& token_a, const std::string& token_b) = 0;
    virtual std::vector<Token> get_tokens() = 0;
    virtual Token get_token_info(const std::string& address) = 0;
    
    // Pricing
    virtual Quote get_quote(const std::string& from_token, const std::string& to_token, 
                           const std::string& amount) = 0;
    virtual std::vector<Quote> get_quotes(const std::string& from_token, 
                                          const std::string& to_token,
                                          const std::string& amount) = 0;
    virtual PriceInfo get_price(const std::string& token_a, const std::string& token_b) = 0;
    
    // Trading
    virtual SwapResult execute_swap(const Order& order) = 0;
    virtual bool cancel_order(const std::string& order_id) = 0;
    virtual Order get_order_status(const std::string& order_id) = 0;
    
    // Liquidity
    virtual std::vector<LiquidityPosition> get_liquidity_positions(const std::string& owner) = 0;
    virtual bool add_liquidity(const std::string& pool, const std::string& amount_a, 
                              const std::string& amount_b) = 0;
    virtual bool remove_liquidity(const std::string& position, const std::string& percent) = 0;
    
    // Getters
    virtual DexProtocol get_protocol() const = 0;
    virtual std::string get_name() const = 0;
    virtual uint64_t get_chain_id() const = 0;
};

// ============================================================================
// Factory
// ============================================================================

std::unique_ptr<DexConnector> create_dex_connector(DexProtocol protocol, 
                                                     const std::string& rpc_url,
                                                     const std::string& private_key = "");

#endif // DEX_CONNECTOR_HPP
