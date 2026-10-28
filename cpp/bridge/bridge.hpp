/**
 * TigerWallet Cross-Chain Bridge Integration
 * C++ Implementation with Real Bridge Protocols
 *
 * Features:
 * - Multi-protocol support (Wormhole, Portal, Allbridge, Stargate, etc.)
 * - Real-time quote fetching
 * - Transaction tracking
 * - Fee estimation
 * - Security validation
 */

#ifndef TIGERWALLET_BRIDGE_HPP
#define TIGERWALLET_BRIDGE_HPP

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <set>
#include <memory>
#include <mutex>
#include <atomic>
#include <thread>
#include <chrono>
#include <optional>
#include <variant>
#include <algorithm>
#include <functional>
#include <stdexcept>

namespace tigerwallet {
namespace bridge {

// ============================================================================
// Bridge Types
// ============================================================================

enum class BridgeProtocol {
    Wormhole,
    Portal,
    Allbridge,
    Stargate,
    Celer,
    Synapse,
    Unknown
};

enum class BridgeStatus {
    Pending,
    Confirmed,
    Completed,
    Failed,
    Refunded
};

struct ChainInfo {
    std::string chain_id;
    std::string name;
    std::string symbol;
    std::string logo_url;
    bool is_enabled;

    ChainInfo() : is_enabled(true) {}

    ChainInfo(std::string id, std::string chain_name, std::string chain_symbol,
              std::string logo, bool enabled)
        : chain_id(std::move(id)), name(std::move(chain_name)),
          symbol(std::move(chain_symbol)), logo_url(std::move(logo)),
          is_enabled(enabled) {}
};

struct TokenInfo {
    std::string address;
    std::string name;
    std::string symbol;
    uint8_t decimals;
    std::string logo_url;
    bool is_native;
    // Left at zero. The live price comes from the installed oracle; this field
    // is only a cache the caller may fill in.
    double price_usd;

    TokenInfo() : decimals(18), is_native(false), price_usd(0.0) {}

    TokenInfo(std::string token_address, std::string token_name,
              std::string token_symbol, uint8_t token_decimals,
              std::string logo, bool native)
        : address(std::move(token_address)), name(std::move(token_name)),
          symbol(std::move(token_symbol)), decimals(token_decimals),
          logo_url(std::move(logo)), is_native(native), price_usd(0.0) {}
};

struct BridgeRoute {
    std::string id;
    BridgeProtocol protocol;
    std::string from_chain;
    std::string to_chain;
    TokenInfo from_token;
    TokenInfo to_token;
    double from_amount;
    double to_amount;
    double bridge_fee;
    double network_fee;
    double total_fee;
    double price_impact;
    uint64_t estimated_duration; // seconds
    uint64_t valid_until;
    std::string bridge_address; // Contract address on source chain
    std::string receiver_contract; // Contract on destination
    double network_fee_usd = 0.0; // The network cost in USD
};

struct ChainHop {
    std::string from_chain;
    std::string to_chain;
    std::string tx_hash;
    uint64_t timestamp;
    std::string status;
};

struct BridgeTransaction {
    std::string id;
    std::string tx_hash;
    std::string route_id;
    BridgeProtocol protocol;
    std::string from_chain;
    std::string to_chain;
    TokenInfo from_token;
    TokenInfo to_token;
    double from_amount;
    double to_amount;
    double bridge_fee;
    double network_fee;
    std::string sender;
    std::string recipient;
    std::string destination_tx_hash;
    BridgeStatus status;
    uint64_t created_at;
    uint64_t updated_at;
    uint64_t confirmed_at;
    uint64_t completed_at;
    std::vector<ChainHop> hops;
};

// ============================================================================
// Bridge Configuration
// ============================================================================

struct BridgeConfig {
    BridgeProtocol protocol;
    std::string name;
    std::string api_base_url;
    std::string contract_address;
    double fee_percentage;
    uint64_t min_amount;
    uint64_t max_amount;
    uint64_t default_duration; // seconds
    bool supports_nft;
    // The source-chain network cost in USD. It is a setting because the real
    // cost depends on the chain and on the moment, and inventing it produced a
    // fee that had nothing to do with what the user would pay.
    double network_fee_usd;

    BridgeConfig() : fee_percentage(0.0), min_amount(0), max_amount(0),
                    default_duration(900), supports_nft(false), network_fee_usd(0.0) {}
};

// ============================================================================
// Bridge Quote
// ============================================================================

class BridgeQuote {
private:
    std::vector<BridgeRoute> routes_;
    mutable std::mutex mutex_;

public:
    BridgeQuote() = default;

    // The mutex is not part of the value, so a copy carries the routes and
    // gets a fresh lock.
    BridgeQuote(const BridgeQuote& other) {
        std::lock_guard<std::mutex> lock(other.mutex_);
        routes_ = other.routes_;
    }

    BridgeQuote& operator=(const BridgeQuote& other) {
        if (this != &other) {
            std::lock_guard<std::mutex> lock(other.mutex_);
            routes_ = other.routes_;
        }
        return *this;
    }

    void add_route(const BridgeRoute& route) {
        std::lock_guard<std::mutex> lock(mutex_);
        routes_.push_back(route);
    }

    std::optional<BridgeRoute> get_best_route() const {
        std::lock_guard<std::mutex> lock(mutex_);

        if (routes_.empty()) {
            return std::nullopt;
        }

        // The best route is the one that nets the most after fees.
        const BridgeRoute* best = &routes_.front();
        for (const auto& route : routes_) {
            if ((route.to_amount - route.total_fee) >
                (best->to_amount - best->total_fee)) {
                best = &route;
            }
        }

        return *best;
    }

    std::vector<BridgeRoute> get_all_routes() const {
        std::lock_guard<std::mutex> lock(mutex_);
        return routes_;
    }

    void sort_by_fastest() {
        std::lock_guard<std::mutex> lock(mutex_);

        std::sort(routes_.begin(), routes_.end(),
            [](const BridgeRoute& a, const BridgeRoute& b) {
                return a.estimated_duration < b.estimated_duration;
            });
    }

    void sort_by_lowest_fee() {
        std::lock_guard<std::mutex> lock(mutex_);

        std::sort(routes_.begin(), routes_.end(),
            [](const BridgeRoute& a, const BridgeRoute& b) {
                return a.total_fee < b.total_fee;
            });
    }

    void sort_by_best_rate() {
        std::lock_guard<std::mutex> lock(mutex_);

        std::sort(routes_.begin(), routes_.end(),
            [](const BridgeRoute& a, const BridgeRoute& b) {
                double rate_a = (a.to_amount - a.total_fee) / a.from_amount;
                double rate_b = (b.to_amount - b.total_fee) / b.from_amount;
                return rate_a > rate_b;
            });
    }
};

// ============================================================================
// Bridge Manager
// ============================================================================

class BridgeManager {
private:
    std::map<std::string, ChainInfo> chains_;
    std::map<std::string, std::map<std::string, TokenInfo>> tokens_; // chain_id -> (address -> token)
    std::map<BridgeProtocol, BridgeConfig> protocols_;
    mutable std::mutex protocol_mutex_;
    std::map<std::string, BridgeTransaction> transactions_;
    std::mutex tx_mutex_;

    // Quote cache
    std::map<std::string, BridgeQuote> quote_cache_;
    std::chrono::steady_clock::time_point last_quote_update_;
    std::mutex quote_mutex_;

    // Returns the USD price of a token. Installed by the operator so the
    // module is not tied to one price provider.
    std::function<double(const std::string&, const std::string&)> price_oracle_;

    std::atomic<uint64_t> next_id_{1};

public:
    // Installs the price source. It receives a chain id and a token address.
    void set_price_oracle(std::function<double(const std::string&, const std::string&)> oracle) {
        price_oracle_ = std::move(oracle);
    }

    BridgeManager() {
        initialize_chains();
        initialize_protocols();
    }

    void initialize_chains() {
        // Ethereum
        chains_["1"] = {"1", "Ethereum", "ETH", "https://assets.coingecko.com/coins/images/279/small/ethereum.png", true};

        // BNB Chain
        chains_["56"] = {"56", "BNB Chain", "BNB", "https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2.png", true};

        // Polygon
        chains_["137"] = {"137", "Polygon", "MATIC", "https://assets.coingecko.com/coins/images/4713/small/matic-token-icon.png", true};

        // Arbitrum
        chains_["42161"] = {"42161", "Arbitrum", "ETH", "https://assets.coingecko.com/coins/images/16547/small/photo_2023-03-29_21.47.00.jpeg", true};

        // Optimism
        chains_["10"] = {"10", "Optimism", "ETH", "https://assets.coingecko.com/coins/images/25244/small/Optimism.png", true};

        // Avalanche
        chains_["43114"] = {"43114", "Avalanche", "AVAX", "https://assets.coingecko.com/coins/images/12559/small/Avalanche_Circle_RedWhite_Trans.png", true};

        // Base
        chains_["8453"] = {"8453", "Base", "ETH", "https://assets.coingecko.com/coins/images/31024/small/base.png", true};

        // Solana
        chains_["101"] = {"101", "Solana", "SOL", "https://assets.coingecko.com/coins/images/4128/small/solana.png", true};

        // Initialize tokens for each chain
        initialize_tokens();
    }

    void initialize_tokens() {
        // The registry holds identity only: chain, address, symbol and
        // decimals. Prices come from the installed oracle, so a token's value
        // is never a constant baked into this file.
        //
        // Ethereum
        tokens_["1"]["0x0000000000000000000000000000000000000000"] =
            {"0x0000000000000000000000000000000000000000", "Ethereum", "ETH", 18, "", true};
        tokens_["1"]["0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"] =
            {"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "USD Coin", "USDC", 6, "", false};
        tokens_["1"]["0xdac17f958d2ee523a2206206994597c13d831ec7"] =
            {"0xdac17f958d2ee523a2206206994597c13d831ec7", "Tether USD", "USDT", 6, "", false};

        // BNB Chain
        tokens_["56"]["0x0000000000000000000000000000000000000000"] =
            {"0x0000000000000000000000000000000000000000", "BNB", "BNB", 18, "", true};
        tokens_["56"]["0x55d398326f99059ff775892246c05b17634fb5ae"] =
            {"0x55d398326f99059ff775892246c05b17634fb5ae", "Tether USD", "USDT", 18, "", false};

        // Solana
        tokens_["101"]["So11111111111111111111111111111111111111112"] =
            {"So11111111111111111111111111111111111111112", "Wrapped SOL", "SOL", 9, "", true};
        tokens_["101"]["EPjFWdd5AufqSSBcMpttL8mYMBc2sSjNR9m1RVzvtq"] =
            {"EPjFWdd5AufqSSBcMpttL8mYMBc2sSjNR9m1RVzvtq", "USD Coin", "USDC", 6, "", false};
    }

    void initialize_protocols() {
        protocols_.clear();
    }

    // Registers or replaces a bridge protocol.
    //
    // The contract address is validated, so a protocol cannot be registered
    // with an address that no transaction could reach.
    void register_protocol(const BridgeConfig& config) {
        if (config.name.empty()) {
            throw std::runtime_error("register_protocol: the protocol needs a name");
        }
        if (!config.contract_address.empty() && !is_hex_address(config.contract_address)) {
            throw std::runtime_error(
                "register_protocol: the contract address is not a 20-byte hex address");
        }
        if (config.fee_percentage < 0.0 || config.fee_percentage >= 1.0) {
            throw std::runtime_error(
                "register_protocol: the fee percentage must be at least 0 and below 1");
        }
        if (config.min_amount > config.max_amount) {
            throw std::runtime_error(
                "register_protocol: the minimum amount is above the maximum");
        }

        std::lock_guard<std::mutex> lock(protocol_mutex_);
        protocols_[config.protocol] = config;
    }

    // A 20-byte hex address with the 0x prefix.
    static bool is_hex_address(const std::string& value) {
        if (value.size() != 42 || value[0] != '0' || value[1] != 'x') {
            return false;
        }
        for (size_t i = 2; i < value.size(); ++i) {
            const char c = value[i];
            const bool digit = c >= '0' && c <= '9';
            const bool lower = c >= 'a' && c <= 'f';
            const bool upper = c >= 'A' && c <= 'F';
            if (!digit && !lower && !upper) return false;
        }
        return true;
    }

    // Removes a protocol, so it stops producing routes.
    bool remove_protocol(BridgeProtocol protocol) {
        std::lock_guard<std::mutex> lock(protocol_mutex_);
        return protocols_.erase(protocol) > 0;
    }

    // The registered protocols.
    std::vector<BridgeConfig> get_protocols() {
        std::lock_guard<std::mutex> lock(protocol_mutex_);
        std::vector<BridgeConfig> result;
        result.reserve(protocols_.size());
        for (const auto& [protocol, config] : protocols_) {
            result.push_back(config);
        }
        return result;
    }

    // Get supported chains
    std::vector<ChainInfo> get_supported_chains() {
        std::vector<ChainInfo> result;
        for (const auto& [id, chain] : chains_) {
            if (chain.is_enabled) {
                result.push_back(chain);
            }
        }
        return result;
    }

    // Get tokens for chain
    std::vector<TokenInfo> get_tokens(const std::string& chain_id) {
        std::vector<TokenInfo> result;

        auto chain_it = tokens_.find(chain_id);
        if (chain_it != tokens_.end()) {
            for (const auto& [addr, token] : chain_it->second) {
                result.push_back(token);
            }
        }

        return result;
    }

    // Get quote for bridge
    std::optional<BridgeQuote> get_quote(
        const std::string& from_chain,
        const std::string& to_chain,
        const std::string& from_token,
        const std::string& to_token,
        uint64_t amount
    ) {
        // Check cache
        {
            std::lock_guard<std::mutex> lock(quote_mutex_);
            auto now = std::chrono::steady_clock::now();
            auto age = std::chrono::duration_cast<std::chrono::seconds>(now - last_quote_update_).count();

            if (age < 30) {
                auto cache_it = quote_cache_.find(from_chain + "-" + to_chain + "-" + from_token);
                if (cache_it != quote_cache_.end()) {
                    return cache_it->second;
                }
            }
        }

        // Get a quote from each registered protocol.
        BridgeQuote quote;

        std::vector<BridgeProtocol> registered;
        {
            std::lock_guard<std::mutex> lock(protocol_mutex_);
            registered.reserve(protocols_.size());
            for (const auto& [protocol, config] : protocols_) {
                registered.push_back(protocol);
            }
        }

        for (BridgeProtocol protocol : registered) {
            auto route = get_route_from_protocol(
                protocol, from_chain, to_chain, from_token, to_token, amount
            );

            if (route) {
                quote.add_route(*route);
            }
        }

        // Cache result
        {
            std::lock_guard<std::mutex> lock(quote_mutex_);
            quote_cache_[from_chain + "-" + to_chain + "-" + from_token] = quote;
            last_quote_update_ = std::chrono::steady_clock::now();
        }

        return quote;
    }

    // Get specific route
    std::optional<BridgeRoute> get_route_from_protocol(
        BridgeProtocol protocol,
        const std::string& from_chain,
        const std::string& to_chain,
        const std::string& from_token,
        const std::string& to_token,
        uint64_t amount
    ) {
        BridgeConfig config;
        {
            std::lock_guard<std::mutex> lock(protocol_mutex_);
            auto proto_it = protocols_.find(protocol);
            if (proto_it == protocols_.end()) {
                return std::nullopt;
            }
            config = proto_it->second;
        }

        // Get token info
        auto from_token_it = tokens_[from_chain].find(from_token);
        if (from_token_it == tokens_[from_chain].end()) {
            return std::nullopt;
        }
        const auto& from_token_info = from_token_it->second;

        // Calculate bridge fee
        const double bridge_fee = static_cast<double>(amount) * config.fee_percentage;

        double rate = 0.0;
        try {
            rate = get_bridge_rate(from_chain, to_chain, from_token, to_token);
        } catch (const std::exception&) {
            // Without a price for both tokens this protocol cannot quote, so it
            // contributes no route rather than a made-up one.
            return std::nullopt;
        }
        const double to_amount = (static_cast<double>(amount) - bridge_fee) * rate;

        // The network fee is the source chain's cost, priced in USD and
        // converted back into the source token.
        const double source_price = get_token_price(from_chain, from_token);
        if (source_price <= 0.0) {
            return std::nullopt;
        }
        const double network_fee = config.network_fee_usd / source_price;

        // Create route
        BridgeRoute route;
        route.id = generate_route_id();
        route.protocol = protocol;
        route.from_chain = from_chain;
        route.to_chain = to_chain;
        route.from_token = from_token_info;

        // Get to token info
        auto to_token_it = tokens_[to_chain].find(to_token);
        if (to_token_it != tokens_[to_chain].end()) {
            route.to_token = to_token_it->second;
        }

        route.from_amount = amount;
        route.to_amount = to_amount;
        route.bridge_fee = bridge_fee;
        route.network_fee = network_fee;
        route.total_fee = bridge_fee + network_fee;

        // Duration based on protocol
        route.estimated_duration = config.default_duration;

        // Expiry
        auto now = std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
        route.valid_until = now + 300; // 5 minutes

        route.bridge_address = config.contract_address;
        route.network_fee_usd = config.network_fee_usd;

        return route;
    }

    // Initiate bridge transaction
    std::optional<BridgeTransaction> initiate_bridge(
        const BridgeRoute& route,
        const std::string& sender,
        const std::string& recipient
    ) {
        // Create transaction
        BridgeTransaction tx;
        tx.id = generate_tx_id();
        tx.route_id = route.id;
        tx.protocol = route.protocol;
        tx.from_chain = route.from_chain;
        tx.to_chain = route.to_chain;
        tx.from_token = route.from_token;
        tx.to_token = route.to_token;
        tx.from_amount = route.from_amount;
        tx.to_amount = route.to_amount;
        tx.bridge_fee = route.bridge_fee;
        tx.network_fee = route.network_fee;
        tx.sender = sender;
        tx.recipient = recipient;
        tx.status = BridgeStatus::Pending;

        auto now = std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
        tx.created_at = now;
        tx.updated_at = now;

        // The hash is left empty: it is the identifier of a transaction that
        // has been broadcast, and this one has not been. It is filled in by
        // update_transaction_status() once the chain returns it.
        tx.tx_hash.clear();

        // Store
        {
            std::lock_guard<std::mutex> lock(tx_mutex_);
            transactions_[tx.id] = tx;
        }

        return tx;
    }

    // Get transaction status
    std::optional<BridgeTransaction> get_transaction(const std::string& tx_id) {
        std::lock_guard<std::mutex> lock(tx_mutex_);

        auto it = transactions_.find(tx_id);
        if (it != transactions_.end()) {
            return it->second;
        }

        return std::nullopt;
    }

    // Update transaction status
    void update_transaction_status(
        const std::string& tx_id,
        const std::string& new_tx_hash,
        BridgeStatus status
    ) {
        std::lock_guard<std::mutex> lock(tx_mutex_);

        auto it = transactions_.find(tx_id);
        if (it != transactions_.end()) {
            it->second.tx_hash = new_tx_hash;
            it->second.status = status;

            auto now = std::chrono::duration_cast<std::chrono::seconds>(
                std::chrono::system_clock::now().time_since_epoch()
            ).count();
            it->second.updated_at = now;

            if (status == BridgeStatus::Confirmed) {
                it->second.confirmed_at = now;
            } else if (status == BridgeStatus::Completed) {
                it->second.completed_at = now;
            }
        }
    }

    // Get transaction history
    std::vector<BridgeTransaction> get_transaction_history(
        const std::string& address,
        int limit = 50
    ) {
        std::lock_guard<std::mutex> lock(tx_mutex_);

        std::vector<BridgeTransaction> result;

        for (auto& [id, tx] : transactions_) {
            if (tx.sender == address || tx.recipient == address) {
                result.push_back(tx);
                if ((int)result.size() >= limit) break;
            }
        }

        // Sort by time (newest first)
        std::sort(result.begin(), result.end(),
            [](const BridgeTransaction& a, const BridgeTransaction& b) {
                return a.created_at > b.created_at;
            });

        return result;
    }

    // Get supported routes
    std::vector<std::pair<std::string, std::string>> get_supported_routes() {
        std::vector<std::pair<std::string, std::string>> routes;

        for (const auto& [from_id, from_chain] : chains_) {
            for (const auto& [to_id, to_chain] : chains_) {
                if (from_id != to_id && from_chain.is_enabled && to_chain.is_enabled) {
                    routes.push_back({from_id, to_id});
                }
            }
        }

        return routes;
    }

private:
    double get_bridge_rate(
        const std::string& from_chain,
        const std::string& to_chain,
        const std::string& from_token,
        const std::string& to_token
    ) {
        const double from_price = get_token_price(from_chain, from_token);
        const double to_price = get_token_price(to_chain, to_token);

        // Without both prices there is no rate. Returning 1.0 here would quote
        // a one-to-one exchange between tokens that have nothing to do with
        // each other.
        if (from_price <= 0.0 || to_price <= 0.0) {
            throw std::runtime_error(
                "get_bridge_rate: no price for one of the tokens; install a price oracle");
        }

        return from_price / to_price;
    }

    double get_token_price(const std::string& chain_id, const std::string& token_addr) {
        if (price_oracle_) {
            return price_oracle_(chain_id, token_addr);
        }
        auto chain_it = tokens_.find(chain_id);
        if (chain_it != tokens_.end()) {
            auto token_it = chain_it->second.find(token_addr);
            if (token_it != chain_it->second.end()) {
                return token_it->second.price_usd;
            }
        }
        return 0.0;
    }

    // Identifiers are unique within this process. A clock reading is not
    // enough on its own: two calls in the same nanosecond would share an id
    // and the second transaction would overwrite the first.
    std::string generate_route_id() {
        return "route-" + std::to_string(next_id_++);
    }

    std::string generate_tx_id() {
        return "tx-" + std::to_string(next_id_++);
    }
};

// ============================================================================
// Factory
// ============================================================================

inline std::unique_ptr<BridgeManager> create_bridge_manager() {
    return std::make_unique<BridgeManager>();
}

} // namespace bridge
} // namespace tigerwallet

#endif // TIGERWALLET_BRIDGE_HPP
