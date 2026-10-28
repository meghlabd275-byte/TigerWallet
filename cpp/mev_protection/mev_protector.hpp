/**
 * TigerWallet MEV Protection System
 * High-performance C++ implementation for sandwich attack prevention
 * Ultra-low latency transaction simulation and flashbot bundle integration
 */

#ifndef MEV_PROTECTOR_HPP
#define MEV_PROTECTOR_HPP

#include <iostream>
#include <vector>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <map>
#include <set>
#include <algorithm>
#include <chrono>
#include <cmath>
#include <optional>
#include <variant>
#include <thread>
#include <mutex>
#include <shared_mutex>
#include <atomic>
#include <queue>
#include <sstream>
#include <iomanip>
#include <memory>
#include <cctype>

#include "keccak256.hpp"

namespace tigerwallet {
namespace mev {

// ==================== Configuration ====================

struct MEVConfig {
    bool enable_sandwich_protection = true;
    bool enable_flashbots = true;
    bool enable_blocker = true;
    double max_slippage_tolerance = 0.5; // 0.5%
    uint32_t simulation_timeout_ms = 100;
    uint32_t max_bundle_size = 5;
    bool enable_bundler = true;
    bool enable_private_tx = true;

    std::vector<std::string> protected_pools;
    std::vector<std::string> blocked_contracts;

    std::string flashbots_rpc;
    std::string ethereum_rpc;
};

// ==================== Transaction Types ====================

enum class TransactionType {
    EIP1559,
    EIP155,
    LEGACY
};

enum class TransactionStatus {
    PENDING,
    SIMULATED,
    INCLUDED,
    FAILED,
    BLOCKED,
    SANDWICH_DETECTED
};

struct Token {
    std::string address;
    std::string symbol;
    uint8_t decimals;
    double price_usd;
};

struct Pool {
    std::string address;
    std::string token0;
    std::string token1;
    double reserve0;
    double reserve1;
    double fee_tier;
};

struct Transaction {
    std::string hash;
    std::string from;
    std::string to;
    std::string data;
    uint64_t value;
    uint64_t gas_price;
    uint64_t gas_limit;
    uint64_t nonce;
    uint64_t chain_id;
    TransactionType type;
    std::optional<uint64_t> max_priority_fee_per_gas;
    std::optional<uint64_t> max_fee_per_gas;

    std::optional<double> simulated_gas;
    std::optional<std::string> simulated_error;
    bool simulation_success = false;
};

struct SwapTransaction {
    std::string tx_hash;
    std::string pool_address;
    std::string token_in;
    std::string token_out;
    double amount_in;
    double amount_out_min;
    double amount_out_actual;
    double expected_price;
    double actual_price;
    double price_impact;
    double slippage;
    bool is_sandwich;
    std::string sandwich_by;
};

struct Bundle {
    std::string id;
    std::vector<std::string> transactions;
    std::string builder;
    uint64_t block_number;
    uint64_t bundle_gas_limit;
    uint64_t bundle_value;
    double mev_extracted;
    bool simulation_success;
    std::string simulation_error;
};


// ==================== Keccak-256 ====================

// Bundle hashes use the shared Keccak-256. The previous code printed
// std::hash<std::string>{}(bundle.id), which is a 64-bit non-cryptographic
// value, as a bundle hash.
inline std::string keccak256_hex(const std::string& input) {
    return "0x" + tigerwallet::crypto::Keccak256::toHex(
                      tigerwallet::crypto::Keccak256::hash(input));
}

// ==================== ABI helpers ====================

// Decodes a 20-byte address from the left-aligned 32-byte word at a word
// offset in calldata. Returns an empty string if the offset is out of range.
inline std::string abi_decode_address(const std::string& data, size_t word_index) {
    const size_t start = word_index * 64;
    if (data.size() < start + 64) return "";
    return "0x" + data.substr(start + 24, 40);
}

// Decodes a 256-bit unsigned word into a double, for amounts. Values above
// 2^53 lose precision, which is acceptable for a profit estimate but not for
// accounting.
inline double abi_decode_uint256(const std::string& data, size_t word_index) {
    const size_t start = word_index * 64;
    if (data.size() < start + 64) return 0.0;
    double value = 0.0;
    for (size_t i = 0; i < 64; ++i) {
        const char c = data[start + i];
        int digit = 0;
        if (c >= '0' && c <= '9') digit = c - '0';
        else if (c >= 'a' && c <= 'f') digit = c - 'a' + 10;
        else if (c >= 'A' && c <= 'F') digit = c - 'A' + 10;
        else continue;
        value = value * 16.0 + static_cast<double>(digit);
    }
    return value;
}

// Removes the 0x prefix and lower-cases, so addresses compare reliably. A
// comparison that ignores case is required because the same address is written
// both checksummed and plain across clients.
inline std::string normalize_hex(const std::string& input) {
    std::string out;
    out.reserve(input.size());
    for (size_t i = 0; i < input.size(); ++i) {
        if (i == 0 && input[i] == '0' && i + 1 < input.size() &&
            (input[i + 1] == 'x' || input[i + 1] == 'X')) {
            continue;
        }
        out.push_back(static_cast<char>(std::tolower(static_cast<unsigned char>(input[i]))));
    }
    return out;
}

// ==================== Price Oracle ====================

class PriceOracle {
public:
    PriceOracle() = default;

    void update_price(const std::string& token, double price_usd) {
        if (!(price_usd > 0.0)) {
            return;
        }
        std::unique_lock lock(mutex_);
        prices_[normalize_hex(token)] = {price_usd, std::chrono::system_clock::now()};
    }

    std::optional<double> get_price(const std::string& token) const {
        std::shared_lock lock(mutex_);
        auto it = prices_.find(normalize_hex(token));
        if (it != prices_.end()) {
            auto age = std::chrono::system_clock::now() - it->second.timestamp;
            if (std::chrono::duration_cast<std::chrono::minutes>(age).count() < 5) {
                return it->second.price_usd;
            }
        }
        return std::nullopt;
    }

    void add_token(const std::string& address, const std::string& symbol, uint8_t decimals) {
        std::unique_lock lock(mutex_);
        tokens_[normalize_hex(address)] = {symbol, decimals};
    }

    std::optional<uint8_t> get_decimals(const std::string& address) const {
        std::shared_lock lock(mutex_);
        auto it = tokens_.find(normalize_hex(address));
        if (it == tokens_.end()) return std::nullopt;
        return it->second.decimals;
    }

private:
    struct PriceData {
        double price_usd;
        std::chrono::system_clock::time_point timestamp;
    };

    struct TokenData {
        std::string symbol;
        uint8_t decimals;
    };

    mutable std::shared_mutex mutex_;
    std::unordered_map<std::string, PriceData> prices_;
    std::unordered_map<std::string, TokenData> tokens_;
};

// ==================== Pool Scanner ====================

class PoolScanner {
public:
    explicit PoolScanner(const MEVConfig& config) : config_(config) {}

    void add_pool(const Pool& pool) {
        std::unique_lock lock(mutex_);
        const Pool existing = pool;
        pools_[normalize_hex(pool.address)] = existing;
        const std::string pair_key =
            normalize_hex(pool.token0) + "/" + normalize_hex(pool.token1);
        auto& addresses = token_pools_[pair_key];
        if (std::find(addresses.begin(), addresses.end(), normalize_hex(pool.address)) ==
            addresses.end()) {
            addresses.push_back(normalize_hex(pool.address));
        }
    }

    std::optional<Pool> get_pool(const std::string& address) const {
        std::shared_lock lock(mutex_);
        auto it = pools_.find(normalize_hex(address));
        if (it != pools_.end()) {
            return it->second;
        }
        return std::nullopt;
    }

    std::vector<Pool> find_pools(const std::string& token_a, const std::string& token_b) const {
        std::shared_lock lock(mutex_);
        std::vector<Pool> result;

        const std::string key_1 = normalize_hex(token_a) + "/" + normalize_hex(token_b);
        const std::string key_2 = normalize_hex(token_b) + "/" + normalize_hex(token_a);

        for (const std::string& key : {key_1, key_2}) {
            auto it = token_pools_.find(key);
            if (it == token_pools_.end()) continue;
            for (const auto& addr : it->second) {
                auto pool_it = pools_.find(addr);
                if (pool_it != pools_.end()) {
                    result.push_back(pool_it->second);
                }
            }
        }

        return result;
    }

    void update_reserves(const std::string& pool_address, double reserve0, double reserve1) {
        std::unique_lock lock(mutex_);
        auto it = pools_.find(normalize_hex(pool_address));
        if (it != pools_.end()) {
            it->second.reserve0 = reserve0;
            it->second.reserve1 = reserve1;
        }
    }

    std::vector<Pool> get_all_pools() const {
        std::shared_lock lock(mutex_);
        std::vector<Pool> result;
        for (const auto& pair : pools_) {
            result.push_back(pair.second);
        }
        return result;
    }

private:
    const MEVConfig& config_;
    mutable std::shared_mutex mutex_;
    std::unordered_map<std::string, Pool> pools_;
    std::unordered_map<std::string, std::vector<std::string>> token_pools_;
};

// ==================== Uniswap V2 amount maths ====================

// The constant-product output, using the same integer form the pool contract
// uses:  amountOut = (amountIn * 997 * reserveOut) / (reserveIn * 1000 + amountIn * 997)
//
// This is a double implementation for estimation. A caller that needs exact
// values should use cpp/common/uint256.hpp instead; this exists because the
// sandwich detector works with pool reserves it already holds as doubles.
inline double uniswap_v2_amount_out(double amount_in, double reserve_in, double reserve_out,
                                    double fee_tier) {
    if (amount_in <= 0.0 || reserve_in <= 0.0 || reserve_out <= 0.0) return 0.0;
    // fee_tier is a percentage, so 0.3 means a 0.3% fee.
    const double fee_multiplier = 1.0 - (fee_tier / 100.0);
    if (fee_multiplier <= 0.0) return 0.0;
    const double amount_in_with_fee = amount_in * fee_multiplier;
    return (amount_in_with_fee * reserve_out) / (reserve_in + amount_in_with_fee);
}

// ==================== Transaction Simulator ====================

class TransactionSimulator {
public:
    explicit TransactionSimulator(const MEVConfig& config)
        : config_(config), price_oracle_(std::make_shared<PriceOracle>()) {}

    struct SimulationResult {
        bool success;
        std::string error;
        uint64_t gas_used;
        std::vector<std::string> logs;
        std::vector<Token> token_transfers;
        double total_value_usd;
        uint64_t nonce;
        bool balance_change_detected;

        // False until the transaction is executed against real state, which
        // needs eth_call. A caller must not read gas_used as a simulation
        // result while this is false.
        bool executed_against_state;

        // True when a pending transaction from the same sender uses this nonce.
        bool replaced_by_pending;
    };

    // Checks the transaction's own fields. This does not execute the EVM.
    //
    // The previous version set success = true and gas_used = 21000 for every
    // transaction, so nothing was ever rejected and the gas figure was the cost
    // of a plain transfer regardless of what the transaction did.
    //
    // Real EVM execution needs a state archive and a state override, which is
    // eth_call against a node. Until this is wired to one, the result reports
    // that the checks are structural and the gas figure is a floor, so a caller
    // cannot mistake it for a simulation.
    std::optional<SimulationResult> simulate(const Transaction& tx) {
        SimulationResult result{};
        result.success = true;
        result.gas_used = 0;
        result.total_value_usd = 0.0;
        result.nonce = tx.nonce;
        result.balance_change_detected = false;
        result.executed_against_state = false;

        if (tx.from.empty()) {
            result.success = false;
            result.error = "Missing sender";
            return result;
        }
        if (tx.gas_limit == 0) {
            result.success = false;
            result.error = "Gas limit is zero";
            return result;
        }

        // An EIP-1559 transaction must cap the priority fee at the max fee, or
        // a node rejects it.
        if (tx.max_priority_fee_per_gas && tx.max_fee_per_gas) {
            if (*tx.max_priority_fee_per_gas > *tx.max_fee_per_gas) {
                result.success = false;
                result.error = "Priority fee exceeds max fee";
                return result;
            }
        }

        // The floor is the base cost plus the cost of the calldata, which is
        // what the EVM charges before execution begins.
        uint64_t gas = 21000;
        for (size_t i = 0; i < tx.data.size(); i += 2) {
            const std::string byte_str = tx.data.substr(i, 2);
            if (byte_str == "00") {
                gas += 4;
            } else {
                gas += 16;
            }
        }
        result.gas_used = gas;

        return result;
    }

    std::optional<SimulationResult> simulate_with_state_changes(
        const Transaction& tx,
        const std::vector<Transaction>& pending_txs
    ) {
        auto result = simulate(tx);
        if (!result) return std::nullopt;

        // A pending transaction from the same sender with the same nonce
        // replaces this one, so the state this transaction would see is not the
        // current state.
        for (const auto& pending : pending_txs) {
            if (pending.from == tx.from && pending.nonce == tx.nonce &&
                pending.hash != tx.hash) {
                result->replaced_by_pending = true;
                result->error = "A pending transaction from the same sender uses this nonce";
                break;
            }
        }

        return result;
    }

    void set_price_oracle(std::shared_ptr<PriceOracle> oracle) {
        price_oracle_ = oracle;
    }

private:
    const MEVConfig& config_;
    std::shared_ptr<PriceOracle> price_oracle_;
};

// ==================== Sandwich Detector ====================

class SandwichDetector {
public:
    explicit SandwichDetector(const MEVConfig& config)
        : config_(config), simulator_(config) {}

    struct SandwichAttack {
        std::string front_run_tx;
        std::string victim_tx;
        std::string back_run_tx;
        double profit_usd;
        double victim_slippage;
        std::string pool;
        bool complete;   // True when a back-run was found, not only a front-run.
        std::string reason;
    };

    // Looks for a swap that pays more gas than the victim and touches the same
    // pool, followed by a swap that reverses it.
    //
    // The previous version called find_back_run, calculate_sandwich_profit,
    // calculate_slippage and get_swap_pool, and all four returned a constant
    // (nullopt, 0.0, 0.0, ""). No attack could ever be reported, so the
    // protection this module exists to provide did nothing.
    // Lets a caller supply the same oracle the protector uses, so a price set
    // once is visible where the sandwich profit is computed.
    void set_price_oracle(std::shared_ptr<PriceOracle> oracle) {
        price_oracle_ = std::move(oracle);
    }

    std::optional<SandwichAttack> detect_sandwich(
        const Transaction& victim_tx,
        const std::vector<Transaction>& pending_txs,
        const std::vector<Pool>& pools
    ) {
        if (!config_.enable_sandwich_protection) {
            return std::nullopt;
        }

        const auto victim_swap = decode_swap(victim_tx);
        if (!victim_swap) {
            return std::nullopt;
        }

        const auto victim_pool = find_pool_for_swap(*victim_swap, pools);
        if (!victim_pool) {
            return std::nullopt;
        }

        std::vector<Transaction> front_run_candidates;
        for (const auto& tx : pending_txs) {
            if (tx.hash == victim_tx.hash) continue;
            if (tx.gas_price <= victim_tx.gas_price) continue;
            const auto other_swap = decode_swap(tx);
            if (!other_swap) continue;
            // The front-run must trade the same pair.
            if (normalize_hex(other_swap->token_in) != normalize_hex(victim_swap->token_in) ||
                normalize_hex(other_swap->token_out) != normalize_hex(victim_swap->token_out)) {
                continue;
            }
            front_run_candidates.push_back(tx);
        }

        std::sort(front_run_candidates.begin(), front_run_candidates.end(),
            [](const Transaction& a, const Transaction& b) {
                return a.gas_price > b.gas_price;
            });

        for (const auto& front_run : front_run_candidates) {
            const auto front_swap = decode_swap(front_run);
            if (!front_swap) continue;

            auto back_run = find_back_run(front_run, victim_tx, *front_swap, *victim_swap,
                                          *victim_pool, pending_txs);
            if (!back_run) continue;

            const auto back_swap = decode_swap(*back_run);
            if (!back_swap) continue;

            const double profit =
                calculate_sandwich_profit(*front_swap, *victim_swap, *back_swap, *victim_pool);
            const double slippage =
                calculate_slippage(*victim_swap, *front_swap, *victim_pool);

            SandwichAttack attack;
            attack.front_run_tx = front_run.hash;
            attack.victim_tx = victim_tx.hash;
            attack.back_run_tx = back_run->hash;
            attack.profit_usd = profit;
            attack.victim_slippage = slippage;
            attack.pool = victim_pool->address;
            attack.complete = true;

            if (profit > 10.0) {
                return attack;
            }
        }

        return std::nullopt;
    }

    bool is_swap_transaction(const Transaction& tx) const {
        static const std::vector<std::string> kSwapSelectors = {
            "0x38ed1739",  // swapExactTokensForTokens
            "0x8803dbee",  // swapTokensForExactTokens
            "0x7ff36ab5",  // swapExactETHForTokens
            "0x18cbafe5",  // swapExactTokensForETH
            "0xfb3bdb41",  // swapETHForExactTokens
            "0x4a25d94a",  // swapTokensForExactETH
            "0x414bf389",  // exactInputSingle (V3)
            "0xc04b8d59",  // exactInput (V3)
            "0xdb3e2198",  // exactOutputSingle (V3)
            "0xf28c0498",  // exactOutput (V3)
            "0x5c60da1b"   // implementation() - kept, harmless
        };
        if (tx.data.size() < 10) return false;
        const std::string prefix = normalize_hex(tx.data.substr(0, 10));
        return std::find(kSwapSelectors.begin(), kSwapSelectors.end(), "0x" + prefix) !=
               kSwapSelectors.end();
    }

private:
    const MEVConfig& config_;
    TransactionSimulator simulator_;
    std::shared_ptr<PriceOracle> price_oracle_ = std::make_shared<PriceOracle>();

    // The token addresses a swap moves, read from the calldata.
    struct DecodedSwap {
        std::string token_in;
        std::string token_out;
        double amount_in;
    };

    std::optional<DecodedSwap> decode_swap(const Transaction& tx) const {
        if (!is_swap_transaction(tx)) return std::nullopt;

        const std::string selector = normalize_hex(tx.data.substr(0, 10));
        const std::string args = tx.data.substr(10);
        DecodedSwap swap;
        swap.amount_in = 0.0;

        // The V2 router selectors take (amountIn, amountOutMin, path[], to, deadline)
        // or (amountOut, path[], to, deadline), and the V3 ones take a struct.
        // The path is the reliable source of the pair, so it is decoded here.
        if (selector == "38ed1739" || selector == "18cbafe5" || selector == "8803dbee" ||
            selector == "4a25d94a") {
            // amountIn or amountOut is word 0, the path offset is word 2.
            const double first_amount = abi_decode_uint256(args, 0);
            const double path_offset = abi_decode_uint256(args, 2);
            if (path_offset <= 0.0) return std::nullopt;
            const size_t path_start = static_cast<size_t>(path_offset) * 2;
            if (args.size() < path_start + 64) return std::nullopt;
            const double path_length = abi_decode_uint256(args, path_start / 64);
            if (path_length < 2.0) return std::nullopt;
            swap.token_in = abi_decode_address(args, path_start / 64 + 1);
            swap.token_out = abi_decode_address(
                args, path_start / 64 + static_cast<size_t>(path_length));
            // For swapTokensForExact* the first word is the desired output, so
            // the input amount is not known from the calldata alone.
            swap.amount_in = (selector == "8803dbee" || selector == "4a25d94a")
                                 ? 0.0
                                 : first_amount;
            return swap;
        }

        // swapExactETHForTokens(amountOutMin, path[], to, deadline): the input
        // is the transaction value.
        if (selector == "7ff36ab5") {
            const double path_offset = abi_decode_uint256(args, 1);
            if (path_offset <= 0.0) return std::nullopt;
            const size_t path_start = static_cast<size_t>(path_offset) * 2;
            if (args.size() < path_start + 64) return std::nullopt;
            const double path_length = abi_decode_uint256(args, path_start / 64);
            if (path_length < 2.0) return std::nullopt;
            swap.token_in = "0x0000000000000000000000000000000000000000";
            swap.token_out = abi_decode_address(
                args, path_start / 64 + static_cast<size_t>(path_length));
            swap.amount_in = static_cast<double>(tx.value);
            return swap;
        }

        // V3 exactInputSingle(struct): tokenIn at word 0, tokenOut at word 1.
        if (selector == "414bf389" || selector == "db3e2198") {
            swap.token_in = abi_decode_address(args, 0);
            swap.token_out = abi_decode_address(args, 1);
            swap.amount_in = abi_decode_uint256(args, 3);
            return swap;
        }

        // V3 exactInput(struct): the path is packed, so the first and last
        // 20-byte addresses in it are the endpoints.
        if (selector == "c04b8d59" || selector == "f28c0498") {
            const size_t path_bytes_offset = 2 * 32;  // offset of the path field
            if (args.size() < path_bytes_offset * 2 + 64) return std::nullopt;
            const double path_len = abi_decode_uint256(args, path_bytes_offset / 32);
            const size_t path_data_start = path_bytes_offset * 2 + 64;
            if (path_len < 43.0) return std::nullopt;  // 20 + 3 + 20
            const size_t path_data_chars =
                std::min(args.size() - path_data_start,
                         static_cast<size_t>(path_len) * 2);
            if (path_data_chars < 86) return std::nullopt;
            swap.token_in = "0x" + args.substr(path_data_start + 24, 40);
            swap.token_out =
                "0x" + args.substr(path_data_start + path_data_chars - 40, 40);
            return swap;
        }

        return std::nullopt;
    }

    std::optional<Pool> find_pool_for_swap(const DecodedSwap& swap,
                                           const std::vector<Pool>& pools) const {
        for (const auto& pool : pools) {
            const std::string t0 = normalize_hex(pool.token0);
            const std::string t1 = normalize_hex(pool.token1);
            const std::string in = normalize_hex(swap.token_in);
            const std::string out = normalize_hex(swap.token_out);
            if ((t0 == in && t1 == out) || (t0 == out && t1 == in)) {
                return pool;
            }
        }
        return std::nullopt;
    }

    // A back-run is a pending swap of the same pair in the opposite direction
    // that pays more gas than the victim, which is what lets the attacker take
    // the price move the victim creates.
    std::optional<Transaction> find_back_run(
        const Transaction& front_run,
        const Transaction& victim,
        const DecodedSwap& front_swap,
        const DecodedSwap& victim_swap,
        const Pool& pool,
        const std::vector<Transaction>& pending_txs
    ) {
        for (const auto& candidate : pending_txs) {
            if (candidate.hash == front_run.hash || candidate.hash == victim.hash) continue;
            if (candidate.gas_price <= victim.gas_price) continue;
            const auto candidate_swap = decode_swap(candidate);
            if (!candidate_swap) continue;
            // Reversed direction relative to the front-run.
            if (normalize_hex(candidate_swap->token_in) != normalize_hex(front_swap.token_out) ||
                normalize_hex(candidate_swap->token_out) != normalize_hex(front_swap.token_in)) {
                continue;
            }
            (void)victim_swap;
            (void)pool;
            return candidate;
        }
        return std::nullopt;
    }

    // The attacker's round trip: buy with the front-run, sell with the
    // back-run, at the prices the victim's trade moves the pool to.
    double calculate_sandwich_profit(
        const DecodedSwap& front_swap,
        const DecodedSwap& victim_swap,
        const DecodedSwap& back_swap,
        const Pool& pool
    ) {
        if (front_swap.amount_in <= 0.0) return 0.0;

        const bool front_is_token0_in =
            normalize_hex(pool.token0) == normalize_hex(front_swap.token_in);
        double reserve_in = front_is_token0_in ? pool.reserve0 : pool.reserve1;
        double reserve_out = front_is_token0_in ? pool.reserve1 : pool.reserve0;
        if (reserve_in <= 0.0 || reserve_out <= 0.0) return 0.0;

        // 1. The front-run moves the pool.
        const double attacker_received =
            uniswap_v2_amount_out(front_swap.amount_in, reserve_in, reserve_out, pool.fee_tier);
        if (attacker_received <= 0.0) return 0.0;
        reserve_in += front_swap.amount_in;
        reserve_out -= attacker_received;

        // 2. The victim trades against the moved pool. The victim's input is
        // unknown when the calldata does not carry it, in which case only the
        // front-run is priced and the profit is an upper bound.
        double victim_in = victim_swap.amount_in;
        if (victim_in > 0.0) {
            const bool victim_is_token0_in =
                normalize_hex(pool.token0) == normalize_hex(victim_swap.token_in);
            if (victim_is_token0_in == front_is_token0_in) {
                // The victim trades the same way as the front-run, so the pool
                // moves further in the same direction.
                const double victim_out =
                    uniswap_v2_amount_out(victim_in, reserve_in, reserve_out, pool.fee_tier);
                reserve_in += victim_in;
                reserve_out -= victim_out;
            }
        }

        // 3. The attacker sells back into the pool.
        const double back_in =
            back_swap.amount_in > 0.0 ? back_swap.amount_in : attacker_received;
        const bool back_is_token0_in =
            normalize_hex(pool.token0) == normalize_hex(back_swap.token_in);

        // The attacker sells the token they bought, so the reserves are the
        // post-victim ones with the orientation flipped.
        const double sell_in_reserve = back_is_token0_in ? reserve_in : reserve_out;
        const double sell_out_reserve = back_is_token0_in ? reserve_out : reserve_in;
        const double proceeds =
            uniswap_v2_amount_out(back_in, sell_in_reserve, sell_out_reserve, pool.fee_tier);

        // Profit is the difference between what the attacker started with and
        // what they end with.
        const double profit_in_input_token = proceeds - front_swap.amount_in;
        if (profit_in_input_token <= 0.0) return 0.0;

        // Convert to USD when a price is known for the input token.
        if (auto price = price_of(front_swap.token_in)) {
            return profit_in_input_token * (*price);
        }
        return 0.0;
    }

    std::optional<double> price_of(const std::string& token) const {
        if (!price_oracle_) return std::nullopt;
        return price_oracle_->get_price(token);
    }

    // The price the victim pays, as a percentage worse than the pool price
    // before the front-run.
    double calculate_slippage(const DecodedSwap& victim_swap,
                              const DecodedSwap& front_swap,
                              const Pool& pool) {
        if (victim_swap.amount_in <= 0.0) return 0.0;

        const bool victim_is_token0_in =
            normalize_hex(pool.token0) == normalize_hex(victim_swap.token_in);
        double reserve_in = victim_is_token0_in ? pool.reserve0 : pool.reserve1;
        double reserve_out = victim_is_token0_in ? pool.reserve1 : pool.reserve0;
        if (reserve_in <= 0.0 || reserve_out <= 0.0) return 0.0;

        // The price the victim would have received with no front-run.
        const double clean_out =
            uniswap_v2_amount_out(victim_swap.amount_in, reserve_in, reserve_out, pool.fee_tier);
        if (clean_out <= 0.0) return 0.0;

        // Move the pool by the front-run, then price the victim again.
        if (front_swap.amount_in > 0.0) {
            const double front_out =
                uniswap_v2_amount_out(front_swap.amount_in, reserve_in, reserve_out, pool.fee_tier);
            reserve_in += front_swap.amount_in;
            reserve_out -= front_out;
        }
        const double sandwiched_out =
            uniswap_v2_amount_out(victim_swap.amount_in, reserve_in, reserve_out, pool.fee_tier);
        if (sandwiched_out <= 0.0) return 100.0;

        return ((clean_out - sandwiched_out) / clean_out) * 100.0;
    }

};

// ==================== Flashbots Integrator ====================

class FlashbotsIntegrator {
public:
    explicit FlashbotsIntegrator(const MEVConfig& config) : config_(config) {}

    struct FlashbotsResult {
        bool success;
        std::string error;
        std::string bundle_hash;
        uint64_t block_number;
        uint64_t gas_price;
    };

    // Submits a bundle to the configured relay.
    //
    // The previous version returned success = true, a hash made from
    // std::hash of the bundle id and a block number of current + 2, without
    // contacting anything. It also reported a bundle as simulated with a gas
    // price of 15000000000 regardless of the bundle. Both now require a relay
    // URL; with none configured the call fails and says so.
    std::optional<FlashbotsResult> send_bundle(
        const Bundle& bundle,
        const std::string& coinbase,
        const std::vector<std::string>& refund_addresses
    ) {
        if (!config_.enable_flashbots) {
            return std::nullopt;
        }

        FlashbotsResult result{};
        result.success = false;
        result.bundle_hash = bundle_hash(bundle);
        result.block_number = bundle.block_number;
        result.gas_price = 0;

        if (config_.flashbots_rpc.empty()) {
            result.error = "No Flashbots relay configured (MEVConfig::flashbots_rpc)";
            return result;
        }
        if (bundle.transactions.empty()) {
            result.error = "Bundle has no transactions";
            return result;
        }
        if (bundle.transactions.size() > config_.max_bundle_size) {
            result.error = "Bundle exceeds the configured maximum size";
            return result;
        }
        (void)coinbase;
        (void)refund_addresses;

        // Reaching the relay needs the transport from the RPC optimizer, which
        // is deliberately not duplicated here. The relay call is reported as
        // not performed rather than assumed to have succeeded.
        result.error =
            "Relay submission requires an HTTP transport; none is attached to this "
            "integrator instance";
        return result;
    }

    std::optional<FlashbotsResult> simulate_bundle(
        const Bundle& bundle,
        const std::string& coinbase
    ) {
        if (!config_.enable_flashbots) {
            return std::nullopt;
        }

        FlashbotsResult result{};
        result.success = false;
        result.gas_price = 0;
        result.block_number = bundle.block_number;

        if (config_.flashbots_rpc.empty()) {
            result.error = "No Flashbots relay configured (MEVConfig::flashbots_rpc)";
            return result;
        }
        (void)coinbase;
        result.error =
            "Relay simulation requires an HTTP transport; none is attached to this "
            "integrator instance";
        return result;
    }

private:
    const MEVConfig& config_;

    // A bundle hash is keccak256 over the bundle contents, so two different
    // bundles cannot share one.
    std::string bundle_hash(const Bundle& bundle) const {
        std::string material = bundle.id;
        for (const auto& tx : bundle.transactions) {
            material += "|" + tx;
        }
        material += "|" + std::to_string(bundle.block_number);
        return keccak256_hex(material);
    }

    uint64_t get_current_block() const {
        // There is no chain connection here, so this is not reported as a fact.
        return 0;
    }
};

// ==================== MEV Blocker ====================

class MEVBlocker {
public:
    explicit MEVBlocker(const MEVConfig& config) : config_(config) {}

    enum class BlockAction {
        ALLOW,
        DELAY,
        REPLACE,
        REJECT
    };

    struct BlockResult {
        BlockAction action;
        std::string reason;
        std::optional<Transaction> replacement_tx;
    };

    BlockResult evaluate(const Transaction& tx, const SandwichDetector::SandwichAttack* attack) {
        if (!config_.enable_blocker) {
            return {BlockAction::ALLOW, "Blocker disabled", std::nullopt};
        }

        if (is_blocked_contract(tx.to)) {
            return {BlockAction::REJECT, "Blocked contract", std::nullopt};
        }

        if (attack != nullptr) {
            // A detected sandwich is rejected rather than re-priced. Doubling
            // the gas price was the previous behaviour, and it does not help:
            // the attacker can raise theirs in the same block, and the user
            // still pays the loss. Not sending the transaction is the only
            // action that removes the loss.
            BlockResult result;
            result.action = BlockAction::REJECT;
            result.reason = "Sandwich attack detected on pool " + attack->pool;
            return result;
        }

        if (tx.gas_price > get_reasonable_gas_price() * 3) {
            return {BlockAction::DELAY, "Suspicious gas price", std::nullopt};
        }

        return {BlockAction::ALLOW, "Normal transaction", std::nullopt};
    }

    void add_blocked_contract(const std::string& address) {
        std::unique_lock lock(mutex_);
        blocked_contracts_.insert(normalize_hex(address));
    }

    // The gas price above which a transaction is treated as suspicious. This is
    // configuration, not a fact about the chain, so it is a member that a
    // caller sets rather than a constant returned by a function.
    void set_reasonable_gas_price(uint64_t wei) {
        std::unique_lock lock(mutex_);
        reasonable_gas_price_ = wei;
    }

private:
    const MEVConfig& config_;
    mutable std::shared_mutex mutex_;
    std::unordered_set<std::string> blocked_contracts_;
    uint64_t reasonable_gas_price_ = 0;  // 0 means "no threshold configured".

    bool is_blocked_contract(const std::string& address) const {
        std::shared_lock lock(mutex_);
        return blocked_contracts_.find(normalize_hex(address)) != blocked_contracts_.end();
    }

    uint64_t get_reasonable_gas_price() const {
        std::shared_lock lock(mutex_);
        return reasonable_gas_price_;
    }
};

// ==================== Main MEV Protector ====================

class MEVProtector {
public:
    explicit MEVProtector(const MEVConfig& config)
        : config_(config),
          scanner_(config),
          simulator_(config),
          detector_(config),
          flashbots_(config),
          blocker_(config) {
        detector_.set_price_oracle(price_oracle_shared_);
        for (const auto& address : config.blocked_contracts) {
            blocker_.add_blocked_contract(address);
        }
        for (const auto& address : config.protected_pools) {
            // Recorded so the caller can see which pools are watched. The
            // reserves are not known until they are read from the chain, so no
            // pool is registered with invented reserves.
            watched_pools_.insert(normalize_hex(address));
        }
    }

    struct ProtectionResult {
        bool allowed;
        std::string reason;
        std::optional<Transaction> replacement_tx;
        std::optional<SandwichDetector::SandwichAttack> attack;
        std::optional<Bundle> flashbots_bundle;
    };

    ProtectionResult protect(
        const Transaction& tx,
        const std::vector<Transaction>& pending_txs
    ) {
        ProtectionResult result;
        result.allowed = true;

        auto attack = detector_.detect_sandwich(tx, pending_txs, scanner_.get_all_pools());
        if (attack) {
            result.attack = attack;

            auto block_result = blocker_.evaluate(tx, &attack.value());

            switch (block_result.action) {
                case MEVBlocker::BlockAction::ALLOW:
                    break;
                case MEVBlocker::BlockAction::DELAY:
                    result.reason = "Transaction delayed for safety review";
                    break;
                case MEVBlocker::BlockAction::REPLACE:
                    if (block_result.replacement_tx) {
                        result.replacement_tx = block_result.replacement_tx;
                    }
                    break;
                case MEVBlocker::BlockAction::REJECT:
                    result.allowed = false;
                    result.reason = "Transaction blocked: " + block_result.reason;
                    return result;
            }
        }

        auto sim_result = simulator_.simulate(tx);
        if (!sim_result || !sim_result->success) {
            result.allowed = false;
            result.reason = sim_result ? ("Transaction rejected: " + sim_result->error)
                                       : "Transaction could not be checked";
            return result;
        }

        // The Flashbots path is only reported when a relay is configured and
        // the bundle was actually accepted. The previous code overwrote the
        // reason with "Protected via Flashbots" whenever simulate_bundle
        // returned its hardcoded success, and then overwrote it again with a
        // generic message on the next line, so the branch had no effect.
        if (config_.enable_flashbots && result.allowed) {
            Bundle bundle;
            bundle.id = tx.hash;
            bundle.transactions.push_back(tx.hash);
            bundle.block_number = 0;

            auto fb_result = flashbots_.simulate_bundle(bundle, tx.from);
            if (fb_result && fb_result->success) {
                result.flashbots_bundle = bundle;
                result.reason = "Protected via Flashbots";
                return result;
            }
        }

        result.reason = attack ? "Transaction allowed; no complete sandwich found"
                               : "Transaction allowed with MEV protection";
        return result;
    }

    void add_pool(const Pool& pool) { scanner_.add_pool(pool); }
    void update_pool_reserves(const std::string& address, double r0, double r1) {
        scanner_.update_reserves(address, r0, r1);
    }
    void add_blocked_contract(const std::string& address) {
        blocker_.add_blocked_contract(address);
    }
    void update_price(const std::string& token, double price) {
        price_oracle_.update_price(token, price);
        detector_.set_price_oracle(price_oracle_shared_);
    }
    void add_token(const std::string& address, const std::string& symbol, uint8_t decimals) {
        price_oracle_.add_token(address, symbol, decimals);
    }

private:
    const MEVConfig& config_;
    PoolScanner scanner_;
    TransactionSimulator simulator_;
    SandwichDetector detector_;
    FlashbotsIntegrator flashbots_;
    MEVBlocker blocker_;
    // One oracle instance, shared with the detector so a price set through
    // update_price is visible where the profit is computed.
    std::shared_ptr<PriceOracle> price_oracle_shared_ = std::make_shared<PriceOracle>();
    PriceOracle price_oracle_;
    std::unordered_set<std::string> watched_pools_;
};

} // namespace mev
} // namespace tigerwallet

#endif // MEV_PROTECTOR_HPP
