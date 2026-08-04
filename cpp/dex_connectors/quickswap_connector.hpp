/**
 * TigerWallet QuickSwap DEX Connector
 * High-Performance C++ Implementation for Polygon Network
 * Ultra-low latency with connection pooling and caching
 */

#ifndef QUICKSWAP_CONNECTOR_HPP
#define QUICKSWAP_CONNECTOR_HPP

#include <atomic>
#include <chrono>
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>
#include <queue>
#include <thread>
#include <optional>
#include <variant>

// ============================================================================
// Configuration
// ============================================================================

namespace quickswap {

constexpr const char* QUICKSWAP_ROUTER_V2 = "0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff";
constexpr const char* QUICKSWAP_ROUTER_V3 = "0x1b02dA8Cb0d097eB8D57A175b88c7D8b47997506";
constexpr const char* QUICKSWAP_FACTORY_V2 = "0x5757371414417b8C6CAad45bAeF941aBc60d2C9E";
constexpr const char* QUICKSWAP_FACTORY_V3 = "0x411b0fAcC1e1fB9D1c93f5E5Ae7E3d4A1a3C31d4";
constexpr const char* QUICKSWAP_NFT_POSITION_MANAGER = "0x5e52e3a13f2e1a5890b5c7e2e3d4e5f6a7b8c9d0";

constexpr uint64_t POLYGON_CHAIN_ID = 137;
constexpr auto CACHE_TTL = std::chrono::milliseconds(500);
constexpr size_t MAX_POOL_CACHE = 10000;
constexpr size_t MAX_CONNECTION_POOL = 100;

// ============================================================================
// Types
// ============================================================================

using BigInt = std::string;
using Timestamp = int64_t;
using Millis = std::chrono::milliseconds;

// Token types
struct Token {
    std::string address;     // Contract address (lowercase)
    std::string symbol;     // Token symbol (e.g., "USDC")
    std::string name;       // Full name
    uint8_t decimals;       // Decimals
    uint256_t totalSupply;  // Total supply
    bool isNative;          // Is native MATIC
    std::string coingeckoId; // For price feeds

    Token() : decimals(18), isNative(false) {}
};

// Pool information
struct Pool {
    std::string address;           // Pool contract address
    Token token0;                  // Token 0 (sorted)
    Token token1;                  // Token 1 (sorted)
    BigInt reserve0;               // Reserve 0 (raw)
    BigInt reserve1;               // Reserve 1 (raw)
    double feeRate;                // Fee in basis points (V3: 100, 500, 3000)
    uint24_t poolTokenId;          // For V3 NFT positions
    uint128_t liquidity;           // Current liquidity
    int24_t tick;                  // Current tick (V3)
    double volume24h;              // 24h volume USD
    double tvl;                    // Total value locked USD

    Pool() : feeRate(0), liquidity(0), tick(0), volume24h(0), tvl(0) {}
};

// Swap quote
struct Quote {
    std::string fromToken;         // Input token address
    std::string toToken;          // Output token address
    BigInt fromAmount;             // Input amount (raw)
    BigInt toAmount;              // Output amount (raw)
    double priceImpact;           // Price impact in %
    double midPrice;              // Mid price
    double execPrice;             // Execution price
    BigInt gasEstimate;           // Gas estimate (wei)
    std::vector<std::string> path; // Swap path
    std::vector<std::string> pools; // Pool addresses used
    bool v3;                      // Uses V3

    Quote() : priceImpact(0), midPrice(0), execPrice(0), v3(false) {}
};

// Swap result
struct SwapResult {
    bool success;
    std::string txHash;
    BigInt fromAmount;
    BigInt toAmount;
    BigInt gasUsed;
    double priceImpact;
    std::string error;
    Timestamp timestamp;

    SwapResult() : success(false), priceImpact(0), timestamp(0) {}
};

// Position (for V3)
struct Position {
    uint256_t tokenId;
    std::string poolAddress;
    Token token0;
    Token token1;
    int24_t tickLower;
    int24_t tickUpper;
    uint128_t liquidity;
    BigInt collectedFees0;
    BigInt collectedFees1;

    Position() : tokenId(0), liquidity(0), tickLower(0), tickUpper(0) {}
};

// Order
struct Order {
    std::string orderId;
    std::string owner;
    std::string fromToken;
    std::string toToken;
    BigInt fromAmount;
    BigInt toAmountMin;
    BigInt toAmount;
    Timestamp deadline;
    Timestamp createdAt;
    bool v3;
    std::optional<Position> position;

    Order() : deadline(0), createdAt(0), v3(false) {}
};

// ============================================================================
// Connection Pool
// ============================================================================

class RPCConnection {
public:
    std::string url;
    std::atomic<int> activeRequests;
    std::atomic<int64_t> latencyUs;
    std::atomic<bool> healthy;
    std::chrono::steady_clock::time_point lastUsed;
    std::mutex mutex;

    RPCConnection(const std::string& u) 
        : url(u), activeRequests(0), latencyUs(0), healthy(true) {}
};

class ConnectionPool {
private:
    std::vector<std::shared_ptr<RPCConnection>> connections_;
    std::mutex mutex_;
    size_t currentIndex_;
    std::atomic<bool> running_;

public:
    ConnectionPool(const std::vector<std::string>& urls);
    ~ConnectionPool();

    std::shared_ptr<RPCConnection> getConnection();
    void returnConnection(std::shared_ptr<RPCConnection> conn);
    void healthCheck();
    size_t size() const { return connections_.size(); }
};

// ============================================================================
// HTTP Client (for RPC calls)
// ============================================================================

class RPCClient {
private:
    ConnectionPool& pool_;
    CURL* curl_;
    std::mutex curlMutex_;
    std::string apiKey_;

public:
    RPCClient(ConnectionPool& pool, const std::string& apiKey = "");
    ~RPCClient();

    // JSON-RPC methods
    std::optional<std::string> call(const std::string& method, const std::string& params);
    
    // Block data
    std::optional<uint64_t> getBlockNumber();
    std::optional<std::string> getBlockByNumber(uint64_t blockNum, bool fullTx = false);
    
    // Token info
    std::optional<Token> getTokenInfo(const std::string& address);
    std::optional<BigInt> balanceOf(const std::string& owner, const std::string& token);
    std::optional<BigInt> allowance(const std::string& owner, const std::string& spender, const std::string& token);
    
    // Pool info
    std::optional<Pool> getPoolByPair(const std::string& tokenA, const std::string& tokenB);
    std::optional<Pool> getPoolByAddress(const std::string& address);
    std::vector<Pool> getAllPools(uint32_t limit = 100);
    
    // Swap
    std::optional<Quote> getQuote(const std::string& fromToken, const std::string& toToken, const BigInt& amount);
    std::optional<Quote> getQuoteV3(const std::string& fromToken, const std::string& toToken, const BigInt& amount);
    
    // Transaction
    std::optional<std::string> sendRawTransaction(const std::string& signedTx);
    std::optional<std::string> getTransactionReceipt(const std::string& txHash);
    
    // Multicall
    std::vector<std::string> multicall(const std::vector<std::string>& calls);

private:
    std::string buildJSONRPC(const std::string& method, const std::string& params);
    std::optional<std::string> executeRequest(std::shared_ptr<RPCConnection> conn, const std::string& payload);
};

// ============================================================================
// Cache
// ============================================================================

template<typename T>
struct CacheEntry {
    T data;
    std::chrono::steady_clock::time_point expires;
    bool valid() const {
        return std::chrono::steady_clock::now() < expires;
    }
};

class PriceCache {
private:
    std::unordered_map<std::string, CacheEntry<double>> prices_;
    std::unordered_map<std::string, CacheEntry<Pool>> pools_;
    std::mutex mutex_;

public:
    void setPrice(const std::string& pair, double price, Millis ttl) {
        std::lock_guard<std::mutex> lock(mutex_);
        prices_[pair] = {price, std::chrono::steady_clock::now() + ttl};
    }

    std::optional<double> getPrice(const std::string& pair) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = prices_.find(pair);
        if (it != prices_.end() && it->second.valid()) {
            return it->second.data;
        }
        return std::nullopt;
    }

    void setPool(const std::string& address, const Pool& pool, Millis ttl) {
        std::lock_guard<std::mutex> lock(mutex_);
        pools_[address] = {pool, std::chrono::steady_clock::now() + ttl};
    }

    std::optional<Pool> getPool(const std::string& address) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = pools_.find(address);
        if (it != pools_.end() && it->second.valid()) {
            return it->second.data;
        }
        return std::nullopt;
    }

    void cleanup() {
        std::lock_guard<std::mutex> lock(mutex_);
        auto now = std::chrono::steady_clock::now();
        
        for (auto it = prices_.begin(); it != prices_.end(); ) {
            if (!it->second.valid()) {
                it = prices_.erase(it);
            } else {
                ++it;
            }
        }
        
        for (auto it = pools_.begin(); it != pools_.end(); ) {
            if (!it->second.valid()) {
                it = pools_.erase(it);
            } else {
                ++it;
            }
        }
    }
};

// ============================================================================
// QuickSwap Connector
// ============================================================================

class QuickSwapConnector {
private:
    ConnectionPool rpcPool_;
    std::unique_ptr<RPCClient> client_;
    PriceCache cache_;
    std::atomic<bool> connected_;
    std::mutex mutex_;
    
    // Token cache
    std::unordered_map<std::string, Token> tokens_;
    std::unordered_map<std::string, std::vector<Pool>> tokenPools_;
    
    // Private key for transactions
    std::string privateKey_;
    std::string walletAddress_;
    
    // Configuration
    bool useV3_;
    double slippageTolerance_;  // in %
    uint64_t deadlineSeconds_;

public:
    QuickSwapConnector(
        const std::vector<std::string>& rpcUrls,
        const std::string& privateKey = "",
        bool useV3 = true
    );
    ~QuickSwapConnector();

    // Connection
    bool connect();
    void disconnect();
    bool isConnected() const { return connected_.load(); }

    // Token operations
    Token getToken(const std::string& address);
    std::vector<Token> getTopTokens(uint32_t limit = 100);
    bool isValidToken(const std::string& address);
    
    // Pool operations
    Pool getPool(const std::string& tokenA, const std::string& tokenB);
    Pool getPoolByAddress(const std::string& address);
    std::vector<Pool> getPoolsForToken(const std::string& token, uint32_t limit = 50);
    std::vector<Pool> getAllPools(uint32_t limit = 100);
    double getTVL();
    double getVolume24h();
    
    // Pricing
    Quote getQuote(const std::string& fromToken, const std::string& toToken, const BigInt& amount);
    Quote getQuoteV2(const std::string& fromToken, const std::string& toToken, const BigInt& amount);
    Quote getQuoteV3(const std::string& fromToken, const std::string& toToken, const BigInt& amount);
    double getPrice(const std::string& tokenA, const std::string& tokenB);
    double getLPPrice(const std::string& poolAddress);
    
    // Trading
    SwapResult swap(const std::string& fromToken, const std::string& toToken, const BigInt& amount, const BigInt& minOutput);
    SwapResult swapV2(const std::string& fromToken, const std::string& toToken, const BigInt& amount, const BigInt& minOutput);
    SwapResult swapV3(const std::string& fromToken, const std::string& toToken, const BigInt& amount, const BigInt& minOutput, uint24_t fee);
    
    // Liquidity (V2)
    BigInt addLiquidity(
        const std::string& tokenA, 
        const std::string& tokenB, 
        const BigInt& amountADesired, 
        const BigInt& amountBDesired,
        const BigInt& amountAMin,
        const BigInt& amountBMin
    );
    
    BigInt removeLiquidity(
        const std::string& tokenA,
        const std::string& tokenB,
        const BigInt& liquidity,
        const BigInt& amountAMin,
        const BigInt& amountBMin
    );
    
    // Liquidity (V3)
    Position mintPosition(
        const std::string& token0,
        const std::string& token1,
        int24_t tickLower,
        int24_t tickUpper,
        uint128_t liquidity
    );
    
    Position increaseLiquidity(
        uint256_t tokenId,
        const BigInt& amount0Desired,
        const BigInt& amount1Desired
    );
    
    Position decreaseLiquidity(
        uint256_t tokenId,
        uint128_t liquidity
    );
    
    Position collectPosition(uint256_t tokenId);
    
    std::vector<Position> getPositions(const std::string& owner);
    
    // Approval
    bool approve(const std::string& token, const std::string& spender, const BigInt& amount);
    bool isApproved(const std::string& owner, const std::string& spender, const std::string& token);
    
    // Configuration
    void setSlippageTolerance(double tolerance) { slippageTolerance_ = tolerance; }
    void setUseV3(bool useV3) { useV3_ = useV3; }
    void setDeadline(uint64_t seconds) { deadlineSeconds_ = seconds; }
    
    // Getters
    std::string getName() const { return "QuickSwap"; }
    uint64_t getChainId() const { return POLYGON_CHAIN_ID; }

private:
    void initializeTokens();
    void initializePools();
    std::string sortTokens(const std::string& tokenA, const std::string& tokenB);
    std::string pairKey(const std::string& tokenA, const std::string& tokenB);
    BigInt calculateMinOutput(const BigInt& expected, double slippage);
    std::string signTransaction(const std::string& txData);
    void refreshCache();
};

// ============================================================================
// Factory Function
// ============================================================================

std::unique_ptr<QuickSwapConnector> createQuickSwapConnector(
    const std::vector<std::string>& rpcUrls,
    const std::string& privateKey = "",
    bool useV3 = true
);

// ============================================================================
// Smart Contract ABIs (Embedded)
// ============================================================================

namespace abi {
    const std::string ERC20_ABI = R"([
        {"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
        {"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
        {"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
        {"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"},
        {"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"type":"function"},
        {"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"},
        {"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"},
        {"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"}
    ])";

    const std::string ROUTER_V2_ABI = R"([
        {"inputs":[{"internalType":"address","name":"_factory","type":"address"},{"internalType":"address","name":"_WETH","type":"address"}],"stateMutability":"nonpayable","type":"constructor"},
        {"inputs":[],"name":"WETH","outputs":[{"internalType":"address","name":"","type":"address"}],"type":"function"},
        {"inputs":[{"internalType":"address","name":"tokenA","type":"address"},{"internalType":"address","name":"tokenB","type":"address"},{"internalType":"uint256","name":"amountADesired","type":"uint256"},{"internalType":"uint256","name":"amountBDesired","type":"uint256"},{"internalType":"uint256","name":"amountAMin","type":"uint256"},{"internalType":"uint256","name":"amountBMin","type":"uint256"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"addLiquidity","outputs":[{"internalType":"uint256","name":"amountA","type":"uint256"},{"internalType":"uint256","name":"amountB","type":"uint256"},{"internalType":"uint256","name":"liquidity","type":"uint256"}],"type":"function"},
        {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"amountTokenDesired","type":"uint256"},{"internalType":"uint256","name":"amountTokenMin","type":"uint256"},{"internalType":"uint256","name":"amountETHMin","type":"uint256"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"addLiquidityETH","outputs":[{"internalType":"uint256","name":"amountToken","type":"uint256"},{"internalType":"uint256","name":"amountETH","type":"uint256"},{"internalType":"uint256","name":"liquidity","type":"uint256"}],"type":"function"},
        {"inputs":[{"internalType":"address","name":"tokenA","type":"address"},{"internalType":"address","name":"tokenB","type":"address"},{"internalType":"uint256","name":"liquidity","type":"uint256"},{"internalType":"uint256","name":"amountAMin","type":"uint256"},{"internalType":"uint256","name":"amountBMin","type":"uint256"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"removeLiquidity","outputs":[{"internalType":"uint256","name":"amountA","type":"uint256"},{"internalType":"uint256","name":"amountB","type":"uint256"}],"type":"function"},
        {"inputs":[{"internalType":"address","name":"tokenA","type":"address"},{"internalType":"address","name":"tokenB","type":"address"},{"internalType":"uint256","name":"liquidity","type":"uint256"},{"internalType":"uint256","name":"amountAMin","type":"uint256"},{"internalType":"uint256","name":"amountBMin","type":"uint256"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"},{"internalType":"bool","name":"approveMax","type":"bool"},{"internalType":"uint8","name":"v","type":"uint8"},{"internalType":"bytes32","name":"r","type":"bytes32"},{"internalType":"bytes32","name":"s","type":"bytes32"}],"name":"removeLiquidityWithPermit","outputs":[{"internalType":"uint256","name":"amountA","type":"uint256"},{"internalType":"uint256","name":"amountB","type":"uint256"}],"type":"function"},
        {"inputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"},{"internalType":"uint256","name":"reserveIn","type":"uint256"},{"internalType":"uint256","name":"reserveOut","type":"uint256"}],"name":"getAmountIn","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"type":"function"},
        {"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"reserveIn","type":"uint256"},{"internalType":"uint256","name":"reserveOut","type":"uint256"}],"name":"getAmountOut","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"type":"function"},
        {"inputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"}],"name":"getAmountsIn","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"type":"function"},
        {"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"}],"name":"getAmountsOut","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"type":"function"},
        {"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactETHForTokens","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"type":"function"},
        {"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactTokensForETH","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"type":"function"},
        {"inputs":[{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"amountOutMin","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapExactTokensForTokens","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"type":"function"},
        {"inputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"},{"internalType":"uint256","name":"amountInMax","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapTokensForExactETH","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"type":"function"},
        {"inputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"},{"internalType":"uint256","name":"amountInMax","type":"uint256"},{"internalType":"address[]","name":"path","type":"address[]"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"deadline","type":"uint256"}],"name":"swapTokensForExactTokens","outputs":[{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"type":"function"}
    ])";
}

// ============================================================================
// Error Codes
// ============================================================================

enum class QuickSwapError {
    SUCCESS = 0,
    NOT_CONNECTED = 1,
    INVALID_TOKEN = 2,
    POOL_NOT_FOUND = 3,
    INSUFFICIENT_LIQUIDITY = 4,
    SLIPPAGE_EXCEEDED = 5,
    TRANSACTION_FAILED = 6,
    RPC_ERROR = 7,
    INVALID_PARAMS = 8,
    TIMEOUT = 9,
    INSUFFICIENT_BALANCE = 10,
    APPROVAL_FAILED = 11
};

} // namespace quickswap

#endif // QUICKSWAP_CONNECTOR_HPP
