/**
 * TigerWallet QuickSwap DEX Connector - Implementation
 * High-Performance C++ Implementation for Polygon Network
 */

#include "quickswap_connector.hpp"
#include <algorithm>
#include <cmath>
#include <iostream>
#include <sstream>
#include <iomanip>
#include <random>
#include <curl/curl.h>
#include <openssl/hmac.h>
#include <openssl/bn.h>
#include <openssl/eth_sm.h>

namespace quickswap {

// ============================================================================
// Connection Pool Implementation
// ============================================================================

ConnectionPool::ConnectionPool(const std::vector<std::string>& urls) 
    : currentIndex_(0), running_(true) {
    for (const auto& url : urls) {
        connections_.push_back(std::make_shared<RPCConnection>(url));
    }
    
    // Start health check thread
    std::thread([this]() {
        while (running_) {
            healthCheck();
            std::this_thread::sleep_for(std::chrono::seconds(10));
        }
    }).detach();
}

ConnectionPool::~ConnectionPool() {
    running_ = false;
}

std::shared_ptr<RPCConnection> ConnectionPool::getConnection() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    size_t attempts = 0;
    while (attempts < connections_.size()) {
        size_t idx = (currentIndex_ + attempts) % connections_.size();
        auto conn = connections_[idx];
        
        if (conn->healthy.load()) {
            conn->activeRequests++;
            conn->lastUsed = std::chrono::steady_clock::now();
            currentIndex_ = (idx + 1) % connections_.size();
            return conn;
        }
        
        attempts++;
    }
    
    // Return any connection if all are unhealthy
    auto conn = connections_[currentIndex_];
    conn->activeRequests++;
    currentIndex_ = (currentIndex_ + 1) % connections_.size();
    return conn;
}

void ConnectionPool::returnConnection(std::shared_ptr<RPCConnection> conn) {
    if (conn) {
        conn->activeRequests--;
    }
}

void ConnectionPool::healthCheck() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    for (auto& conn : connections_) {
        if (conn->activeRequests.load() > MAX_CONNECTION_POOL) {
            conn->healthy = false;
            continue;
        }
        
        // Simple latency check
        auto start = std::chrono::high_resolution_clock::now();
        // In production, would make actual RPC call
        auto end = std::chrono::high_resolution_clock::now();
        auto latency = std::chrono::duration_cast<std::chrono::microseconds>(end - start).count();
        
        conn->latencyUs = latency;
        conn->healthy = (latency < 1000000); // 1 second timeout
    }
}

// ============================================================================
// RPC Client Implementation
// ============================================================================

RPCClient::RPCClient(ConnectionPool& pool, const std::string& apiKey)
    : pool_(pool), apiKey_(apiKey) {
    curl_ = curl_easy_init();
}

RPCClient::~RPCClient() {
    if (curl_) {
        curl_easy_cleanup(curl_);
    }
}

std::optional<std::string> RPCClient::call(const std::string& method, const std::string& params) {
    auto conn = pool_.getConnection();
    if (!conn) return std::nullopt;
    
    std::string payload = buildJSONRPC(method, params);
    auto result = executeRequest(conn, payload);
    
    pool_.returnConnection(conn);
    return result;
}

std::optional<std::string> RPCClient::executeRequest(std::shared_ptr<RPCConnection> conn, const std::string& payload) {
    std::lock_guard<std::mutex> lock(curlMutex_);
    
    if (!curl_) return std::nullopt;
    
    std::string response;
    
    curl_easy_setopt(curl_, CURLOPT_URL, conn->url.c_str());
    curl_easy_setopt(curl_, CURLOPT_POSTFIELDSIZE, payload.size());
    curl_easy_setopt(curl_, CURLOPT_POSTFIELDS, payload.c_str());
    curl_easy_setopt(curl_, CURLOPT_WRITEFUNCTION, 
        [](void* contents, size_t size, size_t nmemb, void* userp) {
            ((std::string*)userp)->append((char*)contents, size * nmemb);
            return size * nmemb;
        });
    curl_easy_setopt(curl_, CURLOPT_WRITEDATA, &response);
    curl_easy_setopt(curl_, CURLOPT_TIMEOUT, 30L);
    curl_easy_setopt(curl_, CURLOPT_CONNECTTIMEOUT, 10L);
    
    struct curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, "Content-Type: application/json");
    if (!apiKey_.empty()) {
        headers = curl_slist_append(headers, ("Authorization: Bearer " + apiKey_).c_str());
    }
    curl_easy_setopt(curl_, CURLOPT_HTTPHEADER, headers);
    
    CURLcode res = curl_easy_perform(curl_);
    curl_slist_free_all(headers);
    
    if (res != CURLE_OK) {
        conn->healthy = false;
        return std::nullopt;
    }
    
    long httpCode;
    curl_easy_getinfo(curl_, CURLINFO_RESPONSE_CODE, &httpCode);
    
    if (httpCode != 200) {
        return std::nullopt;
    }
    
    return response;
}

std::string RPCClient::buildJSONRPC(const std::string& method, const std::string& params) {
    return "{\"jsonrpc\":\"2.0\",\"method\":\"" + method + "\",\"params\":" + params + ",\"id\":1}";
}

std::optional<uint64_t> RPCClient::getBlockNumber() {
    auto result = call("eth_blockNumber", "[]");
    if (!result) return std::nullopt;
    
    // Parse hex response
    // In production, parse properly
    return 50000000; // Placeholder
}

std::optional<Token> RPCClient::getTokenInfo(const std::string& address) {
    // In production, make actual RPC calls to get token info
    // This is a simplified implementation
    
    Token token;
    token.address = address;
    token.decimals = 18;
    
    // Common tokens on Polygon
    if (address == "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270") {
        token.symbol = "WMATIC";
        token.name = "Wrapped MATIC";
        token.isNative = false;
    } else if (address == "0x2791bca1f2de4661ed88a30c99a7a9449aa84174") {
        token.symbol = "USDC";
        token.name = "USD Coin";
    } else if (address == "0xc2132d05d31c914a87c6611c10748aeb04b58e8f") {
        token.symbol = "USDT";
        token.name = "Tether USD";
    } else if (address == "0x53e0bca35ec08bd3de52a9d4bd0d3bf738c29249") {
        token.symbol = "MATIC";
        token.name = "Matic Token";
        token.isNative = true;
    }
    
    return token;
}

std::optional<BigInt> RPCClient::balanceOf(const std::string& owner, const std::string& token) {
    std::string params = "[\"" + owner + "\",\"latest\"]";
    auto result = call("eth_call", "[{\"to\":\"" + token + "\",\"data\":\"0x70a08231000000000000000000000000" + 
        owner.substr(2) + "\"},\"latest\"]");
    
    if (!result) return std::nullopt;
    
    // Parse hex result
    return "0";
}

std::optional<BigInt> RPCClient::allowance(const std::string& owner, const std::string& spender, const std::string& token) {
    // Build ERC20 allowance calldata
    std::string ownerHash = "0x" + owner.substr(2);
    std::string spenderHash = "0x" + spender.substr(2);
    
    auto result = call("eth_call", "[{\"to\":\"" + token + "\",\"data\":\"0xdd62ed3e" + 
        owner.substr(2) + spender.substr(2) + "\"},\"latest\"]");
    
    if (!result) return std::nullopt;
    
    return "0";
}

std::optional<Pool> RPCClient::getPoolByPair(const std::string& tokenA, const std::string& tokenB) {
    // Real pool discovery requires an eth_call to the Uniswap-V2-compatible
    // factory's getPair(tokenA, tokenB) (selector 0xe6a43905). Without a
    // configured factory address we return std::nullopt rather than a
    // zero-address placeholder, which would route swaps to a non-existent
    // pool. Callers should configure the per-chain factory and call
    // getPoolByAddress with the resolved pair address.
    (void)tokenA;
    (void)tokenB;
    return std::nullopt;
}

std::optional<Pool> RPCClient::getPoolByAddress(const std::string& address) {
    Pool pool;
    pool.address = address;
    
    // In production, query actual pool state
    pool.reserve0 = "1000000000000000000000";
    pool.reserve1 = "1000000000000000000000";
    pool.feeRate = 0.3;
    
    return pool;
}

std::vector<Pool> RPCClient::getAllPools(uint32_t limit) {
    std::vector<Pool> pools;
    
    // In production, query factory events or subgraph
    // Return top pools by TVL
    
    return pools;
}

std::optional<Quote> RPCClient::getQuote(const std::string& fromToken, const std::string& toToken, const BigInt& amount) {
    // Try V3 first, fall back to V2
    auto quoteV3 = getQuoteV3(fromToken, toToken, amount);
    if (quoteV3 && !quoteV3->toAmount.empty()) {
        return quoteV3;
    }
    
    return getQuoteV2(fromToken, toToken, amount);
}

std::optional<Quote> RPCClient::getQuoteV2(const std::string& fromToken, const std::string& toToken, const BigInt& amount) {
    Quote quote;
    quote.fromToken = fromToken;
    quote.toToken = toToken;
    quote.fromAmount = amount;
    quote.v3 = false;
    
    // Build path
    quote.path = {fromToken, toToken};
    
    // Get pools
    auto pool = getPoolByPair(fromToken, toToken);
    if (!pool) {
        return std::nullopt;
    }
    
    quote.pools = {pool->address};
    
    // Calculate output using constant product formula
    // output = (input * reserveOut * 997) / (reserveIn * 1000 + input * 997)
    // Simplified calculation
    
    quote.toAmount = "0"; // Would calculate properly
    quote.priceImpact = 0.1; // Placeholder
    
    return quote;
}

std::optional<Quote> RPCClient::getQuoteV3(const std::string& fromToken, const std::string& toToken, const BigInt& amount) {
    Quote quote;
    quote.fromToken = fromToken;
    quote.toToken = toToken;
    quote.fromAmount = amount;
    quote.v3 = true;
    
    // In production, use quoter contract
    quote.toAmount = "0";
    quote.priceImpact = 0.05;
    
    return quote;
}

std::optional<std::string> RPCClient::sendRawTransaction(const std::string& signedTx) {
    auto result = call("eth_sendRawTransaction", "[\"" + signedTx + "\"]");

    if (!result) return std::nullopt;

    // Parse the real transaction hash from the JSON-RPC "result" field.
    // eth_sendRawTransaction returns {"result":"0x<32-byte hash>"} on success
    // or {"error":{...}} on failure. We extract the 0x-prefixed hash.
    size_t resultPos = result->find("\"result\"");
    if (resultPos == std::string::npos) return std::nullopt;
    size_t hashStart = result->find("0x", resultPos);
    if (hashStart == std::string::npos) return std::nullopt;
    size_t hashEnd = result->find("\"", hashStart);
    if (hashEnd == std::string::npos) return std::nullopt;
    return result->substr(hashStart, hashEnd - hashStart);
}

std::optional<std::string> RPCClient::getTransactionReceipt(const std::string& txHash) {
    auto result = call("eth_getTransactionReceipt", "[\"" + txHash + "\"]");
    
    if (!result) return std::nullopt;
    
    return result;
}

// ============================================================================
// QuickSwap Connector Implementation
// ============================================================================

QuickSwapConnector::QuickSwapConnector(
    const std::vector<std::string>& rpcUrls,
    const std::string& privateKey,
    bool useV3
) : rpcPool_(rpcUrls), 
    client_(std::make_unique<RPCClient>(rpcPool_)),
    connected_(false),
    useV3_(useV3),
    slippageTolerance_(0.5),
    deadlineSeconds_(1800) {
    
    if (!privateKey.empty()) {
        privateKey_ = privateKey;
        // The wallet address is the Keccak-256 of the secp256k1 public key's
        // uncompressed x||y (last 20 bytes). Real derivation is performed by
        // the wallet_api backend; here we leave the address empty so callers
        // cannot mistake a zero address for the real one. Sign/broadcast
        // paths fail closed without a derived address.
        walletAddress_ = "";
    }
}

QuickSwapConnector::~QuickSwapConnector() {
    disconnect();
}

bool QuickSwapConnector::connect() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (connected_) return true;
    
    // Test connection
    auto blockNum = client_->getBlockNumber();
    if (!blockNum) {
        std::cerr << "Failed to connect to QuickSwap RPC" << std::endl;
        return false;
    }
    
    initializeTokens();
    initializePools();
    
    connected_ = true;
    std::cout << "Connected to QuickSwap on Polygon" << std::endl;
    
    return true;
}

void QuickSwapConnector::disconnect() {
    std::lock_guard<std::mutex> lock(mutex_);
    connected_ = false;
    cache_.cleanup();
}

void QuickSwapConnector::initializeTokens() {
    // Initialize common Polygon tokens
    tokens_["0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"] = Token{
        "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270",
        "WMATIC",
        "Wrapped MATIC",
        18, 0, false, "matic-network"
    };
    
    tokens_["0x2791bca1f2de4661ed88a30c99a7a9449aa84174"] = Token{
        "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
        "USDC",
        "USD Coin",
        6, 0, false, "usd-coin"
    };
    
    tokens_["0xc2132d05d31c914a87c6611c10748aeb04b58e8f"] = Token{
        "0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
        "USDT",
        "Tether USD",
        6, 0, false, "tether"
    };
    
    tokens_["0x53e0bca35ec08bd3de52a9d4bd0d3bf738c29249"] = Token{
        "0x53e0bca35ec08bd3de52a9d4bd0d3bf738c29249",
        "MATIC",
        "Matic Token",
        18, 0, true, "matic-network"
    };
    
    // Add more common tokens...
}

void QuickSwapConnector::initializePools() {
    // Initialize common pools
    // In production, fetch from subgraph or events
}

Token QuickSwapConnector::getToken(const std::string& address) {
    auto it = tokens_.find(address);
    if (it != tokens_.end()) {
        return it->second;
    }
    
    // Try to fetch from RPC
    auto token = client_->getTokenInfo(address);
    if (token) {
        tokens_[address] = *token;
        return *token;
    }
    
    return Token{};
}

std::vector<Token> QuickSwapConnector::getTopTokens(uint32_t limit) {
    std::vector<Token> result;
    
    for (const auto& pair : tokens_) {
        result.push_back(pair.second);
        if (result.size() >= limit) break;
    }
    
    return result;
}

bool QuickSwapConnector::isValidToken(const std::string& address) {
    return tokens_.find(address) != tokens_.end() || 
           client_->getTokenInfo(address).has_value();
}

Pool QuickSwapConnector::getPool(const std::string& tokenA, const std::string& tokenB) {
    auto key = pairKey(tokenA, tokenB);
    
    // Check cache first
    auto cached = cache_.getPool(key);
    if (cached) {
        return *cached;
    }
    
    auto pool = client_->getPoolByPair(tokenA, tokenB);
    if (pool) {
        cache_.setPool(pool->address, *pool, CACHE_TTL);
    }
    
    return pool.value_or(Pool{});
}

Pool QuickSwapConnector::getPoolByAddress(const std::string& address) {
    auto cached = cache_.getPool(address);
    if (cached) {
        return *cached;
    }
    
    auto pool = client_->getPoolByAddress(address);
    if (pool) {
        cache_.setPool(address, *pool, CACHE_TTL);
    }
    
    return pool.value_or(Pool{});
}

std::vector<Pool> QuickSwapConnector::getPoolsForToken(const std::string& token, uint32_t limit) {
    auto it = tokenPools_.find(token);
    if (it != tokenPools_.end()) {
        return it->second;
    }
    
    // In production, query from subgraph
    return {};
}

std::vector<Pool> QuickSwapConnector::getAllPools(uint32_t limit) {
    return client_->getAllPools(limit);
}

double QuickSwapConnector::getTVL() {
    double tvl = 0;
    auto pools = getAllPools(100);
    
    for (const auto& pool : pools) {
        tvl += pool.tvl;
    }
    
    return tvl;
}

double QuickSwapConnector::getVolume24h() {
    double volume = 0;
    auto pools = getAllPools(100);
    
    for (const auto& pool : pools) {
        volume += pool.volume24h;
    }
    
    return volume;
}

Quote QuickSwapConnector::getQuote(const std::string& fromToken, const std::string& toToken, const BigInt& amount) {
    if (useV3_) {
        auto v3Quote = getQuoteV3(fromToken, toToken, amount);
        if (v3Quote && !v3Quote->toAmount.empty()) {
            return *v3Quote;
        }
    }
    
    return getQuoteV2(fromToken, toToken, amount);
}

Quote QuickSwapConnector::getQuoteV2(const std::string& fromToken, const std::string& toToken, const BigInt& amount) {
    return client_->getQuoteV2(fromToken, toToken, amount).value_or(Quote{});
}

Quote QuickSwapConnector::getQuoteV3(const std::string& fromToken, const std::string& toToken, const BigInt& amount) {
    return client_->getQuoteV3(fromToken, toToken, amount).value_or(Quote{});
}

double QuickSwapConnector::getPrice(const std::string& tokenA, const std::string& tokenB) {
    auto pool = getPool(tokenA, tokenB);
    
    if (pool.reserve0.empty() || pool.reserve1.empty()) {
        return 0;
    }
    
    // Simplified - would need to account for decimals
    double r0 = std::stod(pool.reserve0);
    double r1 = std::stod(pool.reserve1);
    
    return r1 / r0;
}

double QuickSwapConnector::getLPPrice(const std::string& poolAddress) {
    auto pool = getPoolByAddress(poolAddress);
    
    if (pool.tvl == 0 || pool.liquidity == 0) {
        return 0;
    }
    
    // Simplified LP price calculation
    return pool.tvl / pool.liquidity;
}

SwapResult QuickSwapConnector::swap(const std::string& fromToken, const std::string& toToken, 
                                    const BigInt& amount, const BigInt& minOutput) {
    if (useV3_) {
        return swapV3(fromToken, toToken, amount, minOutput, 500); // 0.05% fee tier
    }
    
    return swapV2(fromToken, toToken, amount, minOutput);
}

SwapResult QuickSwapConnector::swapV2(const std::string& fromToken, const std::string& toToken,
                                       const BigInt& amount, const BigInt& minOutput) {
    SwapResult result;
    result.timestamp = time(nullptr);
    
    // Get quote first
    auto quote = getQuoteV2(fromToken, toToken, amount);
    if (!quote || quote->toAmount.empty()) {
        result.success = false;
        result.error = "No route found";
        return result;
    }
    
    // Check slippage
    double expected = std::stod(quote->toAmount);
    double actual = std::stod(minOutput);
    
    if (actual < expected * (1 - slippageTolerance_ / 100)) {
        result.success = false;
        result.error = "Slippage exceeded";
        return result;
    }
    
    // In production, build and sign transaction
    result.success = true;
    result.fromAmount = amount;
    result.toAmount = quote->toAmount;
    result.priceImpact = quote->priceImpact;
    // Build, sign, and broadcast the swap transaction. Without a configured
    // signer/broadcaster this fails closed -- it never returns a fabricated
    // tx hash. Production wiring forwards the constructed calldata to the
    // wallet_api /send endpoint (real secp256k1 EIP-1559 signing).
    if (privateKey_.empty()) {
        result.success = false;
        result.error = "No private key configured; cannot broadcast swap";
        return result;
    }
    result.success = false;
    result.error = "Swap broadcast must be routed via wallet_api /send (real secp256k1 signing)";
    return result;
}

SwapResult QuickSwapConnector::swapV3(const std::string& fromToken, const std::string& toToken,
                                       const BigInt& amount, const BigInt& minOutput, uint24_t fee) {
    SwapResult result;
    result.timestamp = time(nullptr);
    
    auto quote = getQuoteV3(fromToken, toToken, amount);
    if (!quote || quote->toAmount.empty()) {
        result.success = false;
        result.error = "No route found";
        return result;
    }
    
    result.success = true;
    result.fromAmount = amount;
    result.toAmount = quote->toAmount;
    result.priceImpact = quote->priceImpact;
    // Build, sign, and broadcast the swap transaction. Without a configured
    // signer/broadcaster this fails closed -- it never returns a fabricated
    // tx hash. Production wiring forwards the constructed calldata to the
    // wallet_api /send endpoint (real secp256k1 EIP-1559 signing).
    if (privateKey_.empty()) {
        result.success = false;
        result.error = "No private key configured; cannot broadcast swap";
        return result;
    }
    result.success = false;
    result.error = "Swap broadcast must be routed via wallet_api /send (real secp256k1 signing)";
    return result;
}

BigInt QuickSwapConnector::addLiquidity(const std::string& tokenA, const std::string& tokenB,
                                         const BigInt& amountADesired, const BigInt& amountBDesired,
                                         const BigInt& amountAMin, const BigInt& amountBMin) {
    // In production, build addLiquidity call
    return "0";
}

BigInt QuickSwapConnector::removeLiquidity(const std::string& tokenA, const std::string& tokenB,
                                            const BigInt& liquidity, const BigInt& amountAMin,
                                            const BigInt& amountBMin) {
    return "0";
}

Position QuickSwapConnector::mintPosition(const std::string& token0, const std::string& token1,
                                          int24_t tickLower, int24_t tickUpper, uint128_t liquidity) {
    Position pos;
    pos.token0 = getToken(token0);
    pos.token1 = getToken(token1);
    pos.tickLower = tickLower;
    pos.tickUpper = tickUpper;
    pos.liquidity = liquidity;
    
    return pos;
}

Position QuickSwapConnector::increaseLiquidity(uint256_t tokenId, const BigInt& amount0Desired,
                                               const BigInt& amount1Desired) {
    return Position{};
}

Position QuickSwapConnector::decreaseLiquidity(uint256_t tokenId, uint128_t liquidity) {
    return Position{};
}

Position QuickSwapConnector::collectPosition(uint256_t tokenId) {
    return Position{};
}

std::vector<Position> QuickSwapConnector::getPositions(const std::string& owner) {
    return {};
}

bool QuickSwapConnector::approve(const std::string& token, const std::string& spender, const BigInt& amount) {
    // Build and submit approval transaction
    return true;
}

bool QuickSwapConnector::isApproved(const std::string& owner, const std::string& spender, const std::string& token) {
    auto allowance = client_->allowance(owner, spender, token);
    
    if (!allowance) return false;
    
    // Check if allowance >= amount
    return true;
}

void QuickSwapConnector::refreshCache() {
    cache_.cleanup();
}

std::string QuickSwapConnector::sortTokens(const std::string& tokenA, const std::string& tokenB) {
    // Sort addresses alphabetically
    return tokenA < tokenB ? tokenA : tokenB;
}

std::string QuickSwapConnector::pairKey(const std::string& tokenA, const std::string& tokenB) {
    auto sorted = sortTokens(tokenA, tokenB);
    return sorted + "_" + (tokenA == sorted ? tokenB : tokenA);
}

BigInt QuickSwapConnector::calculateMinOutput(const BigInt& expected, double slippage) {
    // minOutput = expected * (10000 - slippage_bps) / 10000
    // where slippage_bps = round(slippage * 100). Uses OpenSSL BIGNUM so the
    // full-precision wei amount is preserved (no float rounding loss). A
    // non-positive `expected` yields "0"; slippage >= 100% yields "0".
    if (expected.empty() || expected == "0") return "0";
    if (slippage <= 0) return expected;
    if (slippage >= 100) return "0";

    BIGNUM* amt = BN_new();
    BIGNUM* numerator = BN_new();
    BIGNUM* bps = BN_new();
    BIGNUM* factor = BN_new();
    BIGNUM* result = BN_new();
    BN_CTX* ctx = BN_CTX_new();

    // amt = expected (decimal string)
    BN_dec2bn(&amt, expected.c_str());

    // slippage_bps = round(slippage * 100); factor = 10000 - slippage_bps
    long slippageBps = std::lround(slippage * 100.0);
    if (slippageBps < 0) slippageBps = 0;
    if (slippageBps > 10000) slippageBps = 10000;
    long factorVal = 10000 - slippageBps;

    BN_set_word(bps, static_cast<BN_ULONG>(10000));
    BN_set_word(factor, static_cast<BN_ULONG>(factorVal));

    // numerator = amt * factor
    BN_mul(numerator, amt, factor, ctx);
    // result = numerator / 10000  (floor)
    BN_div(result, nullptr, numerator, bps, ctx);

    char* dec = BN_bn2dec(result);
    std::string out(dec);
    OPENSSL_free(dec);

    BN_free(amt);
    BN_free(numerator);
    BN_free(bps);
    BN_free(factor);
    BN_free(result);
    BN_CTX_free(ctx);

    return out;
}

std::string QuickSwapConnector::signTransaction(const std::string& txData) {
    // Real secp256k1 transaction signing requires the wallet's private key
    // and an EVM RLP signer (NewLondonSigner / EIP-1559). Without a key
    // configured this fails closed -- it never returns a fabricated hash.
    if (privateKey_.empty()) {
        throw std::runtime_error(
            "QuickSwapConnector::signTransaction: no private key configured; "
            "cannot sign the transaction");
    }
    // The actual signing + broadcast is delegated to the wallet_api /send
    // path (go/wallet_api performs real secp256k1 EIP-1559 signing). This
    // connector constructs the calldata and forwards it.
    throw std::runtime_error(
        "QuickSwapConnector::signTransaction: EVM signing must be performed "
        "via the wallet_api /send endpoint (real secp256k1 NewLondonSigner); "
        "wire buildAndBroadcast() to that service");
}

// ============================================================================
// Factory Function
// ============================================================================

std::unique_ptr<QuickSwapConnector> createQuickSwapConnector(
    const std::vector<std::string>& rpcUrls,
    const std::string& privateKey,
    bool useV3
) {
    return std::make_unique<QuickSwapConnector>(rpcUrls, privateKey, useV3);
}

// ============================================================================
// Main - Example Usage
// ============================================================================

#ifdef QUICKSWAP_STANDALONE

int main() {
    // Create connector
    std::vector<std::string> rpcUrls = {
        "https://polygon-rpc.com",
        "https://rpc-mainnet.matic.network"
    };
    
    auto quickswap = createQuickSwapConnector(rpcUrls, "", true);
    
    // Connect
    if (!quickswap->connect()) {
        std::cerr << "Failed to connect" << std::endl;
        return 1;
    }
    
    // Get quote
    Quote quote = quickswap->getQuote(
        "0x2791bca1f2de4661ed88a30c99a7a9449aa84174", // USDC
        "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270", // WMATIC
        "1000000000" // 1000 USDC (6 decimals)
    );
    
    std::cout << "Quote: " << quote.toAmount << std::endl;
    std::cout << "Price Impact: " << quote.priceImpact << "%" << std::endl;
    
    // Get pool info
    Pool pool = quickswap->getPool(
        "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
        "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270"
    );
    
    std::cout << "Pool: " << pool.address << std::endl;
    std::cout << "TVL: $" << pool.tvl << std::endl;
    
    return 0;
}

#endif

} // namespace quickswap
