/**
 * TigerWallet High-Performance DEX Aggregator
 * C++ Implementation with Ultra-Low Latency
 *
 * COMPLETE PRODUCTION IMPLEMENTATION - NO STUBS
 * Features:
 * - Multi-hop route optimization with Dijkstra/A*
 * - Real-time price aggregation from multiple DEXs
 * - Gas optimization algorithms
 * - Slippage protection mechanisms
 * - MEV protection
 * - Cross-DEX routing
 * - Smart order routing
 * - Split routing for large orders
 */

#ifndef TIGERWALLET_DEX_AGGREGATOR_HPP
#define TIGERWALLET_DEX_AGGREGATOR_HPP

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <queue>
#include <set>
#include <memory>
#include <mutex>
#include <shared_mutex>
#include <atomic>
#include <thread>
#include <future>
#include <functional>
#include <chrono>
#include <optional>
#include <variant>
#include <algorithm>
#include <cmath>
#include <sstream>
#include <iomanip>
#include <regex>
#include <limits>
#include <random>
#include <unordered_map>
#include <unordered_set>

#include "uint256.hpp"
#include "keccak256.hpp"
#include "secp256k1.hpp"
#include "rlp.hpp"
#include "address.hpp"
#include "rpc.hpp"

// Thread pool for parallel execution
#include <condition_variable>

// Networking
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <unistd.h>
#include <fcntl.h>

// JSON handling - using simple internal JSON
namespace tigerwallet {
namespace dex {

// ============================================================================
// Constants
// ============================================================================

constexpr int MAX_HOPS = 4;
constexpr int MAX_ROUTES = 10;
constexpr double DEFAULT_SLIPPAGE = 0.003;
// The most routes one order is divided across.
constexpr int MAX_SPLITS = 4;
// The most quotes held in memory before the stale ones are swept.
constexpr size_t MAX_CACHED_QUOTES = 1000;
// An order is split only when the best pool would move this far. Below it the
// extra gas of a second route costs more than the price improvement.
constexpr double SPLIT_PRICE_IMPACT = 0.01;
constexpr double MIN_LIQUIDITY = 1000.0;
constexpr uint64_t ROUTE_CACHE_TTL_MS = 5000;

// ============================================================================
// Simple JSON Implementation
// ============================================================================

class JsonValue {
public:
    enum Type { NULL_T, BOOL, NUMBER, STRING, ARRAY, OBJECT };

    JsonValue() : type(NULL_T) {}
    JsonValue(bool v) : type(BOOL), boolVal(v) {}
    JsonValue(double v) : type(NUMBER), numVal(v) {}
    JsonValue(const std::string& v) : type(STRING), strVal(v) {}

    Type type;
    bool boolVal = false;
    double numVal = 0.0;
    std::string strVal;
    std::vector<JsonValue> arrayVal;
    std::map<std::string, JsonValue> objVal;

    bool isNull() const { return type == NULL_T; }
    bool isBool() const { return type == BOOL; }
    bool isNumber() const { return type == NUMBER; }
    bool isString() const { return type == STRING; }
    bool isArray() const { return type == ARRAY; }
    bool isObject() const { return type == OBJECT; }

    bool asBool() const { return boolVal; }
    double asNumber() const { return numVal; }
    std::string asString() const { return strVal; }

    JsonValue& operator[](const std::string& key) { return objVal[key]; }
    const JsonValue& operator[](const std::string& key) const {
        static JsonValue null_val;
        auto it = objVal.find(key);
        return (it != objVal.end()) ? it->second : null_val;
    }

    JsonValue& operator[](size_t idx) { return arrayVal[idx]; }
    const JsonValue& operator[](size_t idx) const { return arrayVal[idx]; }

    size_t size() const { return arrayVal.size(); }

    static JsonValue parse(const std::string& json);
    std::string stringify() const;
};

class JsonObject : public JsonValue {
public:
    JsonObject() { type = OBJECT; }

    template<typename T>
    JsonObject& set(const std::string& key, const T& val) {
        objVal[key] = JsonValue(val);
        return *this;
    }

    JsonObject& set(const std::string& key, const std::string& val) {
        objVal[key] = JsonValue(val);
        return *this;
    }

    JsonObject& setNull(const std::string& key) {
        objVal[key] = JsonValue();
        return *this;
    }
};

class JsonArray : public JsonValue {
public:
    JsonArray() { type = ARRAY; }

    template<typename T>
    JsonArray& add(const T& val) {
        arrayVal.push_back(JsonValue(val));
        return *this;
    }

    JsonArray& add(const std::string& val) {
        arrayVal.push_back(JsonValue(val));
        return *this;
    }
};

// ============================================================================
// Enums
// ============================================================================

enum class SwapKind {
    EXACT_IN,
    EXACT_OUT
};

enum class RouteType {
    DIRECT,
    MULTI_HOP,
    SPLIT,
    ARBITRAGE
};

enum class DEXProtocol {
    UNISWAP_V2,
    UNISWAP_V3,
    CURVE,
    SUSHISWAP,
    BALANCER,
    PANCAKESWAP,
    DODO,
    BANCOR,
    KYBER,
    HOOT
};

enum class OrderStatus {
    PENDING,
    ROUTING,
    PREPARING,
    SIGNING,
    BROADCASTING,
    CONFIRMED,
    FAILED
};

// ============================================================================
// 256-bit Integer Implementation
// ============================================================================

// A 256-bit unsigned integer, built on the shared implementation so every
// module in the wallet computes the same way.
class uint256_t {
private:
    math::U256 value_;

public:
    uint256_t() = default;
    uint256_t(uint64_t v) : value_(math::U256::fromUint64(v)) {}
    explicit uint256_t(const math::U256& v) : value_(v) {}

    // Parses decimal digits. A non-digit ends the parse.
    explicit uint256_t(const std::string& decimal)
        : value_(math::U256::fromDecimal(decimal)) {}

    const math::U256& value() const { return value_; }

    // The 64-bit words, least significant first.
    uint64_t operator[](size_t i) const { return value_.word[i]; }

    uint256_t operator+(const uint256_t& other) const {
        return uint256_t(value_ + other.value_);
    }
    uint256_t operator-(const uint256_t& other) const {
        return uint256_t(value_ - other.value_);
    }
    uint256_t operator*(const uint256_t& other) const {
        return uint256_t(value_ * other.value_);
    }
    // Division and remainder with a 512-bit intermediate, so a product is not
    // truncated before the divide.
    uint256_t operator/(const uint256_t& other) const {
        if (other.value_.isZero()) return uint256_t();
        return uint256_t(math::mulDiv(value_, math::U256::fromUint64(1),
                                      other.value_));
    }
    uint256_t operator%(const uint256_t& other) const {
        if (other.value_.isZero()) return uint256_t();
        math::U256 remainder;
        math::divideFull(math::multiplyFull(value_, math::U256::fromUint64(1)),
                         other.value_, &remainder);
        return uint256_t(remainder);
    }

    bool operator==(const uint256_t& other) const { return value_ == other.value_; }
    bool operator!=(const uint256_t& other) const { return value_ != other.value_; }
    bool operator<(const uint256_t& other) const { return value_ < other.value_; }
    bool operator>(const uint256_t& other) const { return value_ > other.value_; }
    bool operator<=(const uint256_t& other) const { return value_ <= other.value_; }
    bool operator>=(const uint256_t& other) const { return value_ >= other.value_; }

    uint256_t& operator+=(const uint256_t& other) { value_ = value_ + other.value_; return *this; }
    uint256_t& operator-=(const uint256_t& other) { value_ = value_ - other.value_; return *this; }
    uint256_t& operator*=(const uint256_t& other) { value_ = value_ * other.value_; return *this; }
    uint256_t& operator/=(const uint256_t& other) { *this = *this / other; return *this; }

    bool isZero() const { return value_.isZero(); }

    std::string convert_to_string() const { return value_.toDecimal(); }

    // A lossy conversion for display and for values that fit a double.
    explicit operator double() const {
        return static_cast<double>(value_.word[0]) +
               static_cast<double>(value_.word[1]) * 18446744073709551616.0;
    }
};

// ============================================================================
// Structures
// ============================================================================

struct Token {
    std::string address;
    std::string symbol;
    std::string name;
    uint8_t decimals;
    std::string chain;

    bool operator==(const Token& other) const {
        return address == other.address && chain == other.chain;
    }

    std::string id() const {
        return chain + ":" + address;
    }
};

struct TokenAmount {
    Token token;
    uint256_t rawAmount;
    double decimalAmount;

    TokenAmount() : rawAmount(0), decimalAmount(0.0) {}

    TokenAmount(const Token& t, uint256_t raw) : token(t), rawAmount(raw) {
        double divisor = 1.0;
        for (int i = 0; i < t.decimals; i++) divisor *= 10.0;
        decimalAmount = static_cast<double>(raw) / divisor;
    }

    std::string toString() const {
        return rawAmount.convert_to_string();
    }
};

struct Pool {
    std::string id;
    DEXProtocol protocol;
    Token token0;
    Token token1;
    uint256_t reserve0;
    uint256_t reserve1;
    double fee;
    uint64_t timestamp;
    std::string poolAddress;

    // Uniswap V3 state. sqrtPriceX96 is the square root of the price, scaled by
    // 2^96, and liquidity is the in-range liquidity. Both are read from the
    // pool contract; a pool without them cannot be quoted and is refused.
    uint256_t sqrtPriceX96;
    uint256_t liquidity;

    // Uniswap V3 fee tier in hundredths of a basis point, which is how the
    // pool identifies itself. 3000 is the 0.3 percent tier.
    uint32_t feeTier = 0;

    // Curve state: the amplification coefficient and the pool balances, one
    // per coin in the pool.
    uint256_t amplification;
    std::vector<uint256_t> balances;

    // True when the fields the protocol needs are present.
    bool hasV3State() const {
        return !sqrtPriceX96.isZero() && !liquidity.isZero();
    }

    bool hasCurveState() const {
        return !amplification.isZero() && balances.size() >= 2;
    }

    double getPrice(bool token0In) const {
        if (reserve0.isZero() || reserve1.isZero()) return 0;
        const double r0 = static_cast<double>(reserve0);
        const double r1 = static_cast<double>(reserve1);
        if (token0In) {
            return r1 / std::max(1.0, r0);
        }
        return r0 / std::max(1.0, r1);
    }
};

struct RouteStep {
    Pool pool;
    Token fromToken;
    Token toToken;
    DEXProtocol protocol;
    double fee;
    double expectedOutput;
    double priceImpact;

    RouteStep() : fee(0.0), expectedOutput(0.0), priceImpact(0.0) {}
};

struct SwapRoute {
    std::vector<RouteStep> steps;
    TokenAmount input;
    TokenAmount expectedOutput;
    double totalGas;
    double totalFee;
    double priceImpact;
    RouteType type;
    std::string routeId;

    SwapRoute() : totalGas(0.0), totalFee(0.0), priceImpact(0.0), type(RouteType::DIRECT) {}

    double getOutputAmount() const {
        return expectedOutput.decimalAmount;
    }

    double getTotalValue() const {
        return expectedOutput.decimalAmount;
    }
};

struct SwapQuote {
    std::string quoteId;
    TokenAmount inputToken;
    TokenAmount outputToken;
    std::vector<SwapRoute> routes;
    SwapRoute bestRoute;
    uint64_t validUntil;
    double gasPrice;
    uint64_t estimatedGas;
    std::string transactionData;

    SwapQuote() : validUntil(0), gasPrice(0.0), estimatedGas(0) {}
};

struct Order {
    std::string orderId;
    std::string userAddress;
    TokenAmount inputAmount;
    TokenAmount outputAmount;
    double slippageTolerance;
    std::string recipient;
    uint64_t deadline;
    OrderStatus status;
    std::string txHash;
    std::string errorMessage;

    Order() : slippageTolerance(DEFAULT_SLIPPAGE), deadline(0), status(OrderStatus::PENDING) {}
};

struct MarketData {
    std::string pairId;
    Token token0;
    Token token1;
    double price0;
    double price1;
    double volume24h;
    double liquidity;
    uint64_t lastUpdate;

    MarketData() : price0(0.0), price1(0.0), volume24h(0.0), liquidity(0.0), lastUpdate(0) {}
};

struct GasEstimate {
    uint64_t gasLimit;
    uint64_t gasUsed;
    double gasPrice;
    double totalGasCost;
    uint64_t confirmationBlocks;

    GasEstimate() : gasLimit(0), gasUsed(0), gasPrice(0.0), totalGasCost(0.0), confirmationBlocks(0) {}
};

// ============================================================================
// Custom Hash Functions
// ============================================================================

struct TokenHash {
    size_t operator()(const Token& token) const {
        std::hash<std::string> hasher;
        return hasher(token.address + token.chain);
    }
};

struct PoolHash {
    size_t operator()(const Pool& pool) const {
        std::hash<std::string> hasher;
        return hasher(pool.id);
    }
};

// ============================================================================
// Exception Classes
// ============================================================================

class DEXAggregatorException : public std::runtime_error {
public:
    explicit DEXAggregatorException(const std::string& msg) : std::runtime_error(msg) {}
};

class InsufficientLiquidityException : public DEXAggregatorException {
public:
    InsufficientLiquidityException() : DEXAggregatorException("Insufficient liquidity for swap") {}
};

class NoRouteFoundException : public DEXAggregatorException {
public:
    NoRouteFoundException() : DEXAggregatorException("No valid route found for swap") {}
};

class PriceTooHighException : public DEXAggregatorException {
public:
    explicit PriceTooHighException(double impact)
        : DEXAggregatorException("Price impact too high: " + std::to_string(impact * 100) + "%") {}
};

class OrderFailedException : public DEXAggregatorException {
public:
    explicit OrderFailedException(const std::string& reason)
        : DEXAggregatorException("Order failed: " + reason) {}
};

// ============================================================================
// Path Finding Algorithm Classes
// ============================================================================

class PathFinder {
public:
    struct PathNode {
        Token token;
        double bestScore;
        std::vector<RouteStep> path;

        // The queue is a max-heap, so the largest output is popped first and
        // the search follows the best route found so far.
        bool operator<(const PathNode& other) const {
            return bestScore < other.bestScore;
        }
    };

    std::vector<SwapRoute> findBestRoutes(
        const Token& fromToken,
        const Token& toToken,
        uint256_t amount,
        const std::unordered_map<std::string, std::vector<Pool>>& pools,
        int maxHops = MAX_HOPS,
        int maxRoutes = MAX_ROUTES
    );

private:
    double calculateRouteScore(
        const std::vector<RouteStep>& route,
        double inputAmount,
        double outputAmount
    );

    std::vector<RouteStep> dijkstra(
        const Token& start,
        const Token& end,
        uint256_t amount,
        const std::unordered_map<std::string, std::vector<Pool>>& pools,
        int maxHops
    );
};

// ============================================================================
// Price Oracle
// ============================================================================

class PriceOracle {
public:
    PriceOracle();

    // The node and the feed contract the oracle reads from. Both must be set
    // before a price can be fetched.
    void setSource(tigerwallet::RpcClient* client, std::string feed_address);

    // Reads the feed for a token pair. The pair key is "<base>/<quote>".
    // Returns nullopt when no feed is configured or the read fails.
    std::optional<double> fetchPrice(const std::string& tokenPair);

    // The most recent value, refreshed from the feed when it is older than the
    // refresh interval.
    double getPrice(const std::string& tokenPair);

    // Forces a read, ignoring the cache.
    std::optional<double> getPriceWithRefresh(const std::string& tokenPair);

    std::map<std::string, double> getAllPrices();
    void setRefreshInterval(uint64_t ms);

private:
    struct PriceData {
        double price;
        uint64_t timestamp;
        uint64_t updateCount;
    };

    // Reads latestAnswer() from the feed contract.
    std::optional<double> readFeed(const std::string& feed_address);

    std::unordered_map<std::string, PriceData> prices_;
    std::unordered_map<std::string, std::string> feeds_;
    std::mutex mutex_;
    uint64_t refreshIntervalMs_;
    std::chrono::steady_clock::time_point lastUpdate_;
    tigerwallet::RpcClient* rpc_ = nullptr;
    std::string feed_address_;
};

// ============================================================================
// Gas Estimator
// ============================================================================

class GasEstimator {
public:
    GasEstimator();

    GasEstimate estimateGas(
        const SwapRoute& route,
        const Token& fromToken,
        const Token& toToken,
        double gasPriceWei
    );

    void updateGasPrice(double gasPriceWei);
    double getCurrentGasPrice();
    uint64_t estimateConfirmationTime(uint64_t gasLimit);

private:
    std::atomic<double> currentGasPrice_;
    std::mutex mutex_;
    std::vector<double> gasPriceHistory_;
    uint64_t lastUpdate_;
};

// ============================================================================
// DEX Adapter Base Class
// ============================================================================

class DEXAdapter {
public:
    virtual ~DEXAdapter() = default;

    // The node the adapter reads pools from, and the factory it asks. Both are
    // installed by the aggregator; an adapter without them reports no pools
    // rather than inventing one.
    void setNode(tigerwallet::RpcClient* client, std::string factory_address) {
        rpc_ = client;
        factory_address_ = std::move(factory_address);
    }

    virtual std::string getName() const = 0;
    virtual DEXProtocol getProtocol() const = 0;
    virtual std::vector<Pool> getPools(const Token& tokenA, const Token& tokenB) = 0;
    virtual double getAmountOut(
        const TokenAmount& amountIn,
        const Token& toToken,
        const Pool& pool
    ) = 0;
    virtual double getAmountIn(
        const TokenAmount& amountOut,
        const Token& fromToken,
        const Pool& pool
    ) = 0;
    virtual std::string buildSwapData(
        const TokenAmount& amountIn,
        const Token& toToken,
        const std::string& to,
        const Pool& pool,
        uint256_t amountOutMin
    ) = 0;
    virtual bool supportsProtocol(DEXProtocol protocol) const = 0;

protected:
    // Reads a pool's reserves and derives the pool address. Returns nullopt
    // when the pair does not exist.
    std::optional<Pool> discoverV2Pool(const Token& tokenA, const Token& tokenB,
                                       DEXProtocol protocol, double fee);

    tigerwallet::RpcClient* rpc_ = nullptr;
    std::string factory_address_;

    // Encodes swapExactTokensForTokens for a V2-style router. Both Uniswap V2
    // and PancakeSwap use this interface.
    static std::string encodeV2Swap(const TokenAmount& amountIn,
                                    const Token& toToken, const std::string& to,
                                    uint256_t amountOutMin, uint64_t deadline);

    // Decodes the first ABI word of a return payload as an address.
    static std::optional<std::string> decode_address(const std::string& hex);
    // Decodes the first two ABI words as unsigned integers.
    static std::optional<std::pair<math::U256, math::U256>> decode_two_uints(
        const std::string& hex);
};

// ============================================================================
// Concrete DEX Adapters
// ============================================================================

class UniswapV2Adapter : public DEXAdapter {
public:
    std::string getName() const override { return "Uniswap V2"; }
    DEXProtocol getProtocol() const override { return DEXProtocol::UNISWAP_V2; }

    std::vector<Pool> getPools(const Token& tokenA, const Token& tokenB) override;

    double getAmountOut(
        const TokenAmount& amountIn,
        const Token& toToken,
        const Pool& pool
    ) override;

    double getAmountIn(
        const TokenAmount& amountOut,
        const Token& fromToken,
        const Pool& pool
    ) override;

    std::string buildSwapData(
        const TokenAmount& amountIn,
        const Token& toToken,
        const std::string& to,
        const Pool& pool,
        uint256_t amountOutMin
    ) override;

    bool supportsProtocol(DEXProtocol protocol) const override {
        return protocol == DEXProtocol::UNISWAP_V2 || protocol == DEXProtocol::SUSHISWAP;
    }

    // The constant-product output for an exact input.
    //
    //     amountOut = (amountIn * (10000 - feeBps) * reserveOut)
    //                 / (reserveIn * 10000 + amountIn * (10000 - feeBps))
    //
    // The product is 512-bit, so it cannot overflow before the divide.
    static uint256_t calculateOutput(uint256_t amountIn, uint256_t reserveIn,
                                     uint256_t reserveOut, uint32_t feeBps);

    // The input needed for an exact output, rounding up.
    static uint256_t calculateInput(uint256_t amountOut, uint256_t reserveIn,
                                    uint256_t reserveOut, uint32_t feeBps);
};

class UniswapV3Adapter : public DEXAdapter {
public:
    std::string getName() const override { return "Uniswap V3"; }
    DEXProtocol getProtocol() const override { return DEXProtocol::UNISWAP_V3; }

    std::vector<Pool> getPools(const Token& tokenA, const Token& tokenB) override;

    double getAmountOut(
        const TokenAmount& amountIn,
        const Token& toToken,
        const Pool& pool
    ) override;

    double getAmountIn(
        const TokenAmount& amountOut,
        const Token& fromToken,
        const Pool& pool
    ) override;

    std::string buildSwapData(
        const TokenAmount& amountIn,
        const Token& toToken,
        const std::string& to,
        const Pool& pool,
        uint256_t amountOutMin
    ) override;

    bool supportsProtocol(DEXProtocol protocol) const override {
        return protocol == DEXProtocol::UNISWAP_V3;
    }

    // The exact single-tick output for an exact input, from the pool's current
    // sqrt price and in-range liquidity:
    //
    //     sqrtNext = (liquidity * sqrtPrice * 1e6)
    //                / (liquidity * 1e6 + amountIn * sqrtPrice)
    //     amountOut = liquidity * (sqrtPrice - sqrtNext) / 1e6
    //
    // Returns zero when the pool carries no V3 state.
    static uint256_t amountOutV3(uint256_t amountIn, uint256_t sqrtPriceX96,
                                 uint256_t liquidity, uint32_t feePpm);

    // The fee tiers this adapter will look up, in hundredths of a basis point.
    // A V3 factory can hold one pool per tier for a pair.
    static const std::vector<uint32_t>& feeTiers();
};

class CurveAdapter : public DEXAdapter {
public:
    // The StableSwap invariant and its exact output, by Newton iteration on
    //
    //     A * n^n * sum(x) + D = A * D * n^n + D^(n+1) / (n^n * prod(x))
    //
    // Returns zero when the pool carries no Curve state.
    static uint256_t getDy(uint256_t i, uint256_t j, uint256_t dx,
                           const std::vector<uint256_t>& balances,
                           uint256_t amplification, uint32_t feePpm);

    // The Curve registry the adapter asks for a pool address. Unlike the V2
    // factory, the registry is a separate contract.
    void setRegistry(std::string registry_address) {
        registry_address_ = std::move(registry_address);
    }

private:
    std::string registry_address_;
    std::vector<Pool> discoverCurvePools(const Token& tokenA, const Token& tokenB);

public:

public:
    std::string getName() const override { return "Curve"; }
    DEXProtocol getProtocol() const override { return DEXProtocol::CURVE; }

    std::vector<Pool> getPools(const Token& tokenA, const Token& tokenB) override;

    double getAmountOut(
        const TokenAmount& amountIn,
        const Token& toToken,
        const Pool& pool
    ) override;

    double getAmountIn(
        const TokenAmount& amountOut,
        const Token& fromToken,
        const Pool& pool
    ) override;

    std::string buildSwapData(
        const TokenAmount& amountIn,
        const Token& toToken,
        const std::string& to,
        const Pool& pool,
        uint256_t amountOutMin
    ) override;

    bool supportsProtocol(DEXProtocol protocol) const override {
        return protocol == DEXProtocol::CURVE;
    }
};

class PancakeSwapAdapter : public DEXAdapter {
public:
    std::string getName() const override { return "PancakeSwap"; }
    DEXProtocol getProtocol() const override { return DEXProtocol::PANCAKESWAP; }

    std::vector<Pool> getPools(const Token& tokenA, const Token& tokenB) override;

    double getAmountOut(
        const TokenAmount& amountIn,
        const Token& toToken,
        const Pool& pool
    ) override;

    double getAmountIn(
        const TokenAmount& amountOut,
        const Token& fromToken,
        const Pool& pool
    ) override;

    std::string buildSwapData(
        const TokenAmount& amountIn,
        const Token& toToken,
        const std::string& to,
        const Pool& pool,
        uint256_t amountOutMin
    ) override;

    bool supportsProtocol(DEXProtocol protocol) const override {
        return protocol == DEXProtocol::PANCAKESWAP;
    }
};

// ============================================================================
// Main DEX Aggregator Class
// ============================================================================

class DEXAggregator {
public:
    static DEXAggregator& getInstance();

    // Initialization
    void initialize(
        const std::vector<std::string>& rpcEndpoints,
        const std::string& chainId,
        const std::string& routerAddress
    );

    // Quote fetching
    SwapQuote getQuote(
        const Token& fromToken,
        const Token& toToken,
        uint256_t amountIn,
        double slippageTolerance = DEFAULT_SLIPPAGE,
        SwapKind kind = SwapKind::EXACT_IN
    );

    std::vector<SwapQuote> getQuotes(
        const Token& fromToken,
        const Token& toToken,
        uint256_t amountIn,
        int maxResults = 5
    );

    // Order execution
    Order executeSwap(
        const SwapQuote& quote,
        const std::string& privateKey,
        const std::string& recipient
    );

    // The router that executes a protocol's swaps. Each protocol has its own,
    // and a call built for one sent to another reverts.
    void registerRouter(DEXProtocol protocol, const std::string& router_address);
    std::string routerFor(DEXProtocol protocol) const;

    // The contract a route's calldata must be sent to. A Curve route targets
    // the pool itself; the others target their router.
    std::string transactionTarget(const SwapRoute& route) const;

    // Registers the factory an adapter asks for pools, and the Curve registry.
    // Both are per-chain and per-deployment, so the operator supplies them.
    void registerFactory(DEXProtocol protocol, const std::string& factory_address);
    void registerCurveRegistry(const std::string& registry_address);

    // The feed the price oracle reads, and the registry it uses.
    void registerPriceFeed(const std::string& feed_address);

    // Pool management
    void addPool(const Pool& pool);
    void removePool(const std::string& poolId);
    void updatePool(const Pool& pool);
    std::vector<Pool> getPools(const Token& tokenA, const Token& tokenB);

    // Token management
    void addToken(const Token& token);
    void removeToken(const std::string& tokenAddress);
    std::vector<Token> getTokens();

    // Market data
    MarketData getMarketData(const Token& tokenA, const Token& tokenB);
    std::map<std::string, MarketData> getAllMarketData();

    // Gas management
    GasEstimate estimateGas(
        const SwapRoute& route,
        const Token& fromToken,
        const Token& toToken
    );
    void updateGasPrice(double gasPriceWei);

    // Monitoring
    struct AggregatorStats {
        uint64_t totalQuotes;
        uint64_t totalSwaps;
        uint64_t failedSwaps;
        double totalVolumeUSD;
        double averageSlippage;
        uint64_t averageExecutionTimeMs;
        std::map<std::string, uint64_t> dexUsage;
    };

    AggregatorStats getStats() const;
    void resetStats();

    // Configuration
    // Installs the signer used by executeSwap. It receives the unsigned
    // transaction as JSON and returns the signed raw transaction hex.
    void setTransactionSigner(
        std::function<std::string(const std::string&)> signer);

    void setMaxSlippage(double slippage);
    void setDeadline(uint64_t seconds);
    void enableMEVProtection(bool enable);
    void enableGasOptimization(bool enable);

    // Destructor
    ~DEXAggregator();

private:
    DEXAggregator();

    // Prevent copying
    DEXAggregator(const DEXAggregator&) = delete;
    DEXAggregator& operator=(const DEXAggregator&) = delete;

    // The factory registered for a protocol, or empty.
    std::string factoryFor(DEXProtocol protocol) const;

    // The adapter that handles a protocol, or nullptr.
    DEXAdapter* adapterFor(DEXProtocol protocol) const;

    // Core routing
    std::vector<SwapRoute> findRoutes(
        const Token& fromToken,
        const Token& toToken,
        uint256_t amount,
        int maxRoutes,
        SwapKind kind = SwapKind::EXACT_IN
    );

    // Route optimization
    SwapRoute optimizeRoute(
        const std::vector<RouteStep>& steps,
        const TokenAmount& input,
        double slippage
    );

    // Split routing for large orders
    std::vector<SwapRoute> createSplitRoutes(
        const Token& fromToken,
        const Token& toToken,
        uint256_t amountIn,
        const std::vector<SwapRoute>& routes
    );

    // Re-prices a route for a smaller input by walking its steps again. Used
    // when an order is split across routes.
    SwapRoute repriceRoute(const SwapRoute& route, const TokenAmount& input);

    // Transaction building
    std::string buildTransaction(
        const SwapRoute& route,
        const TokenAmount& input,
        const std::string& to,
        uint256_t amountOutMin,
        uint64_t deadline
    );

    // RPC communication. The client talks to the operator's own node.
    std::string callRPC(const std::string& method, const std::string& params_json);
    std::string broadcastTransaction(const std::string& signedTx);

    // The node client, built from the configured endpoint list.
    std::unique_ptr<tigerwallet::RpcClient> rpcClient_;

    // The signer for executeSwap. Installed by the operator so the module
    // never handles a key it was not given.
    std::function<std::string(const std::string& /*unsigned_tx_json*/)>
        transaction_signer_;

    // Caching
    std::optional<SwapQuote> getCachedQuote(const std::string& key);
    void cacheQuote(const std::string& key, const SwapQuote& quote);

    // Thread safety
    mutable std::shared_mutex mutex_;

    // State
    std::string chainId_;
    std::string routerAddress_;
    std::vector<std::string> rpcEndpoints_;
    size_t currentEndpoint_;

    // Token and pool storage
    std::unordered_map<std::string, Token> tokens_;
    std::unordered_map<std::string, std::vector<Pool>> poolsByPair_;
    std::unordered_map<std::string, Pool> poolsById_;

    // DEX adapters
    std::vector<std::unique_ptr<DEXAdapter>> dexAdapters_;

    // Per-protocol factory addresses and the price feed, supplied by the
    // operator.
    std::unordered_map<DEXProtocol, std::string> factories_;
    std::unordered_map<DEXProtocol, std::string> routers_;
    std::string priceFeedAddress_;

    // Path finder
    std::unique_ptr<PathFinder> pathFinder_;

    // Price oracle
    std::unique_ptr<PriceOracle> priceOracle_;

    // Gas estimator
    std::unique_ptr<GasEstimator> gasEstimator_;

    // Configuration
    double maxSlippage_;
    uint64_t deadline_;
    bool mevProtectionEnabled_;
    bool gasOptimizationEnabled_;

    // Statistics
    std::atomic<uint64_t> totalQuotes_;
    std::atomic<uint64_t> totalSwaps_;
    std::atomic<uint64_t> failedSwaps_;
    std::atomic<double> totalVolumeUSD_;

    // Thread pool for async operations
    std::vector<std::thread> workerThreads_;
    std::queue<std::function<void()>> taskQueue_;
    std::condition_variable taskCV_;
    std::mutex taskMutex_;
    std::atomic<bool> running_;

    // Quote cache
    struct CachedQuote {
        SwapQuote quote;
        std::chrono::steady_clock::time_point timestamp;
    };
    std::unordered_map<std::string, CachedQuote> quoteCache_;
    // The quote cache is written from getQuote while the routing lock is held
    // in shared mode, so it carries its own lock.
    mutable std::mutex cacheMutex_;
};

// ============================================================================
// Utility Functions
// ============================================================================

std::string generateUUID();
std::string toLower(const std::string& str);
std::string toHex(const std::vector<uint8_t>& data);
std::vector<uint8_t> fromHex(const std::string& hex);
uint256_t parseUnits(const std::string& amount, uint8_t decimals);
std::string formatUnits(uint256_t amount, uint8_t decimals);

// ============================================================================
// Inline Implementations
// ============================================================================

inline std::string generateUUID() {
    static std::random_device rd;
    static std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 15);
    std::uniform_int_distribution<> dis2(8, 11);

    std::stringstream ss;
    ss << std::hex;
    for (int i = 0; i < 8; i++) ss << dis(gen);
    ss << "-";
    for (int i = 0; i < 4; i++) ss << dis(gen);
    ss << "-4";
    for (int i = 0; i < 3; i++) ss << dis(gen);
    ss << "-";
    ss << dis2(gen);
    for (int i = 0; i < 3; i++) ss << dis(gen);
    ss << "-";
    for (int i = 0; i < 12; i++) ss << dis(gen);
    return ss.str();
}

inline std::string toLower(const std::string& str) {
    std::string result = str;
    std::transform(result.begin(), result.end(), result.begin(), ::tolower);
    return result;
}

inline std::string toHex(const std::vector<uint8_t>& data) {
    std::stringstream ss;
    ss << "0x";
    for (auto b : data) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)b;
    }
    return ss.str();
}

inline std::vector<uint8_t> fromHex(const std::string& hex) {
    std::vector<uint8_t> result;
    std::string hexStr = hex;
    if (hexStr.length() >= 2 && hexStr.substr(0, 2) == "0x") {
        hexStr = hexStr.substr(2);
    }

    for (size_t i = 0; i < hexStr.length(); i += 2) {
        std::string byteStr = hexStr.substr(i, 2);
        uint8_t byte = static_cast<uint8_t>(std::stoi(byteStr, nullptr, 16));
        result.push_back(byte);
    }
    return result;
}

inline uint256_t parseUnits(const std::string& amount, uint8_t decimals) {
    uint256_t result = 0;
    size_t dotPos = amount.find('.');
    std::string intPart = amount;
    std::string fracPart = "";

    if (dotPos != std::string::npos) {
        intPart = amount.substr(0, dotPos);
        fracPart = amount.substr(dotPos + 1);
    }

    if (!intPart.empty()) {
        result = result + uint256_t(std::stoull(intPart));
    }

    if (!fracPart.empty()) {
        uint256_t multiplier = 1;
        for (size_t i = fracPart.length(); i < decimals; i++) {
            multiplier *= 10;
        }
        for (char c : fracPart) {
            if (c >= '0' && c <= '9') {
                result = result + uint256_t(c - '0') * multiplier;
            }
            multiplier /= 10;
        }
    }

    return result;
}

inline std::string formatUnits(uint256_t amount, uint8_t decimals) {
    std::string result = amount.convert_to_string();
    if (decimals == 0) return result;

    if (result.length() <= decimals) {
        result = std::string(decimals - result.length() + 1, '0') + result;
    }
    result.insert(result.length() - decimals, ".");
    return result;
}

} // namespace dex
} // namespace tigerwallet

#endif // TIGERWALLET_DEX_AGGREGATOR_HPP
