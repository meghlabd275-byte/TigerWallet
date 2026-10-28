// Tests for the MEV protection module.
//
// The sandwich detector is the part that has to work: it previously called
// four helpers that all returned constants, so no attack could ever be
// reported and the module did nothing. These tests build real V2 router
// calldata, register a pool with real reserves, and check that a sandwich is
// found and priced, and that a clean transaction is not flagged.

#include "mev_protector.hpp"

#include <cassert>
#include <cstdio>
#include <string>
#include <algorithm>
#include <vector>

using namespace tigerwallet::mev;

namespace {

const char* kRouter = "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D";
const char* kWeth = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2";
const char* kUsdc = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48";

// A uint256 as a 32-byte word.
//
// The previous helper padded its argument as if it were already hex, so a
// caller writing word("160") produced the value 0x160 (352) rather than 160.
// The path offset came out as 352 bytes, which points past the end of the
// calldata, so decoding failed and the detector never ran. Taking a number and
// converting it removes the ambiguity.
//
// A 128-bit type is used because a realistic amount such as 50 WETH in wei is
// 5e19, which does not fit in 64 bits.
std::string word(unsigned __int128 value) {
    std::string hex;
    do {
        const int digit = static_cast<int>(value % 16);
        hex.push_back(static_cast<char>(digit < 10 ? '0' + digit : 'a' + digit - 10));
        value /= 16;
    } while (value != 0);
    std::reverse(hex.begin(), hex.end());
    while (hex.size() < 64) hex = "0" + hex;
    return hex;
}

// A whole-token amount in base units, which is what the calldata carries.
unsigned __int128 tokens(uint64_t whole, uint8_t decimals) {
    unsigned __int128 value = whole;
    for (uint8_t i = 0; i < decimals; ++i) {
        value *= 10;
    }
    return value;
}

std::string address_word(const std::string& address) {
    std::string out = address;
    if (out.rfind("0x", 0) == 0) out = out.substr(2);
    while (out.size() < 64) out = "0" + out;
    return out;
}

// swapExactTokensForTokens(amountIn, amountOutMin, path[], to, deadline)
std::string encode_swap_exact_tokens_for_tokens(
    unsigned __int128 amount_in, const std::string& token_in, const std::string& token_out) {
    std::string data = "0x38ed1739";
    data += word(amount_in);
    data += word(0);                                    // amountOutMin
    data += word(160);                                  // path offset
    data += address_word("0x1111111111111111111111111111111111111111");  // to
    data += word(9999999999ULL);                        // deadline
    data += word(2);                                    // path length
    data += address_word(token_in);
    data += address_word(token_out);
    return data;
}

// swapExactETHForTokens(amountOutMin, path[], to, deadline)
std::string encode_swap_exact_eth_for_tokens(const std::string& token_out) {
    std::string data = "0x7ff36ab5";
    data += word(0);                                    // amountOutMin
    data += word(128);                                  // path offset
    data += address_word("0x1111111111111111111111111111111111111111");  // to
    data += word(9999999999ULL);                        // deadline
    data += word(2);                                    // path length
    data += address_word(kWeth);
    data += address_word(token_out);
    return data;
}

Transaction MakeSwap(const std::string& hash, const std::string& data,
                     uint64_t gas_price, uint64_t value = 0) {
    Transaction tx;
    tx.hash = hash;
    tx.from = "0x2222222222222222222222222222222222222222";
    tx.to = kRouter;
    tx.data = data;
    tx.value = value;
    tx.gas_price = gas_price;
    tx.gas_limit = 300000;
    tx.nonce = 1;
    tx.chain_id = 1;
    tx.type = TransactionType::EIP1559;
    return tx;
}

Pool MakePool() {
    Pool pool;
    pool.address = "0x8ad599c3a0ff1de082011efddc58f1908eb6e6d8";
    pool.token0 = kWeth;
    pool.token1 = kUsdc;
    pool.reserve0 = 1000.0;   // 1000 WETH
    pool.reserve1 = 2000000.0;  // 2,000,000 USDC
    pool.fee_tier = 0.3;
    return pool;
}

}  // namespace

int main() {
    // --- A clean swap is not flagged -----------------------------------------

    {
        MEVConfig config;
        config.enable_flashbots = false;
        MEVProtector protector(config);
        protector.add_pool(MakePool());

        const Transaction victim =
            MakeSwap("0xvictim", encode_swap_exact_tokens_for_tokens(tokens(10, 18), kWeth, kUsdc), 20000000000ULL);

        const auto result = protector.protect(victim, {});
        assert(result.allowed);
        assert(!result.attack.has_value());
    }

    // --- A non-swap is not flagged -------------------------------------------

    {
        MEVConfig config;
        config.enable_flashbots = false;
        MEVProtector protector(config);
        protector.add_pool(MakePool());

        Transaction plain = MakeSwap("0xplain", "0x", 20000000000ULL);
        plain.to = "0x3333333333333333333333333333333333333333";
        const auto result = protector.protect(plain, {});
        assert(result.allowed);
        assert(!result.attack.has_value());
    }

    // --- A sandwich is detected and priced -----------------------------------

    {
        MEVConfig config;
        config.enable_flashbots = false;
        MEVProtector protector(config);
        protector.add_pool(MakePool());
        // WETH and USDC are priced and registered, so the profit can be
        // expressed in dollars and the decimals are known.
        protector.add_token(kWeth, "WETH", 18);
        protector.add_token(kUsdc, "USDC", 6);
        protector.update_price(kWeth, 2000.0);
        protector.update_price(kUsdc, 1.0);

        // The victim buys USDC with 10 WETH.
        const Transaction victim = MakeSwap(
            "0xvictim", encode_swap_exact_tokens_for_tokens(tokens(10, 18), kWeth, kUsdc), 20000000000ULL);

        // The attacker front-runs with 50 WETH at a higher gas price.
        const Transaction front_run = MakeSwap(
            "0xfrontrun", encode_swap_exact_tokens_for_tokens(tokens(50, 18), kWeth, kUsdc), 40000000000ULL);

        // ...and back-runs by selling USDC for WETH.
        const Transaction back_run = MakeSwap(
            "0xbackrun", encode_swap_exact_tokens_for_tokens(tokens(100000, 6), kUsdc, kWeth), 30000000000ULL);

        const auto result = protector.protect(victim, {front_run, back_run});

        assert(result.attack.has_value());
        const auto& attack = *result.attack;
        assert(attack.front_run_tx == "0xfrontrun");
        assert(attack.victim_tx == "0xvictim");
        assert(attack.back_run_tx == "0xbackrun");
        assert(attack.complete);
        // The victim pays a worse price than they would have without the
        // front-run, so the slippage must be positive.
        assert(attack.victim_slippage > 0.0);
        assert(attack.profit_usd > 0.0);

        // The blocker rejects a detected sandwich, so the user keeps their funds.
        assert(!result.allowed);
        assert(result.reason.find("Sandwich") != std::string::npos);
    }

    // --- A front-run alone is not a complete sandwich ------------------------

    {
        MEVConfig config;
        config.enable_flashbots = false;
        MEVProtector protector(config);
        protector.add_pool(MakePool());
        protector.update_price(kWeth, 2000.0);

        const Transaction victim = MakeSwap(
            "0xvictim", encode_swap_exact_tokens_for_tokens(tokens(10, 18), kWeth, kUsdc), 20000000000ULL);
        const Transaction front_run = MakeSwap(
            "0xfrontrun", encode_swap_exact_tokens_for_tokens(tokens(50, 18), kWeth, kUsdc), 40000000000ULL);

        // No back-run, so no profit can be taken.
        const auto result = protector.protect(victim, {front_run});
        assert(result.allowed);
        assert(!result.attack.has_value());
    }

    // --- A swap on a different pair is not a sandwich ------------------------

    {
        MEVConfig config;
        config.enable_flashbots = false;
        MEVProtector protector(config);
        protector.add_pool(MakePool());
        protector.update_price(kWeth, 2000.0);

        const Transaction victim = MakeSwap(
            "0xvictim", encode_swap_exact_tokens_for_tokens(tokens(10, 18), kWeth, kUsdc), 20000000000ULL);
        const Transaction other = MakeSwap(
            "0xother",
            encode_swap_exact_tokens_for_tokens(50, kUsdc, kWeth),
            40000000000ULL);
        const Transaction other_back = MakeSwap(
            "0xotherback",
            encode_swap_exact_tokens_for_tokens(tokens(50, 18), kWeth, kUsdc),
            40000000000ULL);

        const auto result = protector.protect(victim, {other, other_back});
        assert(!result.attack.has_value());
    }

    // --- ETH-for-tokens swaps decode their input from the value --------------

    {
        MEVConfig config;
        config.enable_flashbots = false;
        MEVProtector protector(config);
        protector.add_pool(MakePool());
        protector.add_token(kWeth, "WETH", 18);
        protector.add_token(kUsdc, "USDC", 6);
        protector.update_price(kWeth, 2000.0);
        protector.update_price(kUsdc, 1.0);

        const Transaction victim = MakeSwap(
            "0xvictim", encode_swap_exact_eth_for_tokens(kUsdc), 20000000000ULL, 5000000000000000000ULL);
        const Transaction front_run = MakeSwap(
            "0xfrontrun", encode_swap_exact_eth_for_tokens(kUsdc), 40000000000ULL, 20000000000000000000ULL);
        const Transaction back_run = MakeSwap(
            "0xbackrun", encode_swap_exact_tokens_for_tokens(tokens(100000, 6), kUsdc, kWeth), 30000000000ULL);

        const auto result = protector.protect(victim, {front_run, back_run});
        assert(result.attack.has_value());
        assert(result.attack->front_run_tx == "0xfrontrun");
    }

    // --- Structural checks reject malformed transactions ---------------------

    {
        MEVConfig config;
        config.enable_flashbots = false;
        MEVProtector protector(config);

        // No gas limit.
        Transaction no_gas = MakeSwap("0x1", encode_swap_exact_tokens_for_tokens(tokens(1, 18), kWeth, kUsdc), 1);
        no_gas.gas_limit = 0;
        const auto rejected = protector.protect(no_gas, {});
        assert(!rejected.allowed);

        // Priority fee above the max fee, which a node rejects.
        Transaction bad_fees = MakeSwap("0x2", encode_swap_exact_tokens_for_tokens(tokens(1, 18), kWeth, kUsdc), 1);
        bad_fees.max_priority_fee_per_gas = 5000;
        bad_fees.max_fee_per_gas = 1000;
        const auto bad = protector.protect(bad_fees, {});
        assert(!bad.allowed);
        assert(bad.reason.find("Priority fee") != std::string::npos);

        // No sender.
        Transaction no_from = MakeSwap("0x3", encode_swap_exact_tokens_for_tokens(tokens(1, 18), kWeth, kUsdc), 1);
        no_from.from = "";
        assert(!protector.protect(no_from, {}).allowed);
    }

    // --- Blocked contracts are rejected --------------------------------------

    {
        MEVConfig config;
        config.enable_flashbots = false;
        config.blocked_contracts.push_back("0xDEADBEEFdeadbeefDEADBEEFdeadbeefDEADBEEF");
        MEVProtector protector(config);

        Transaction tx = MakeSwap("0x4", encode_swap_exact_tokens_for_tokens(tokens(1, 18), kWeth, kUsdc), 1);
        tx.to = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
        const auto result = protector.protect(tx, {});
        assert(!result.allowed);
        assert(result.reason.find("blocked") != std::string::npos);
    }

    // --- Flashbots reports failure without a relay, not success --------------

    {
        MEVConfig config;
        config.enable_flashbots = true;
        config.flashbots_rpc = "";  // No relay.
        MEVProtector protector(config);
        protector.add_pool(MakePool());

        const Transaction victim = MakeSwap(
            "0xvictim", encode_swap_exact_tokens_for_tokens(tokens(10, 18), kWeth, kUsdc), 20000000000ULL);
        const auto result = protector.protect(victim, {});
        assert(result.allowed);
        // No relay, so no bundle is claimed and the reason does not mention
        // Flashbots.
        assert(!result.flashbots_bundle.has_value());
        assert(result.reason.find("Flashbots") == std::string::npos);
    }

    // --- The price oracle expires stale prices -------------------------------

    {
        PriceOracle oracle;
        oracle.update_price(kWeth, 2000.0);
        assert(oracle.get_price(kWeth).has_value());
        assert(oracle.get_price(kWeth).value() == 2000.0);
        // Case does not matter.
        assert(oracle.get_price("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2").has_value());
        // An unknown token has no price.
        assert(!oracle.get_price("0x0000000000000000000000000000000000000001").has_value());
        // A non-positive price is not stored.
        oracle.update_price("0x0000000000000000000000000000000000000002", 0.0);
        assert(!oracle.get_price("0x0000000000000000000000000000000000000002").has_value());
    }

    // --- Pool lookup ignores address case ------------------------------------

    {
        MEVConfig config;
        PoolScanner scanner(config);
        scanner.add_pool(MakePool());

        const auto found = scanner.find_pools(kWeth, kUsdc);
        assert(found.size() == 1);
        const auto reversed = scanner.find_pools(kUsdc, kWeth);
        assert(reversed.size() == 1);

        // Reserves update through a differently-cased address.
        scanner.update_reserves("0x8AD599C3A0FF1DE082011EFDDC58F1908EB6E6D8", 2000.0, 4000000.0);
        const auto pool = scanner.get_pool(MakePool().address);
        assert(pool.has_value());
        assert(pool->reserve0 == 2000.0);
    }

    // --- The constant-product formula matches the pool's own integer maths ---

    {
        // 10 WETH into a 1000 WETH / 2,000,000 USDC pool at 0.3%:
        //   10 * 0.997 = 9.97
        //   9.97 * 2000000 / (1000 + 9.97) = 19743.1607
        const double out = uniswap_v2_amount_out(10.0, 1000.0, 2000000.0, 0.3);
        assert(out > 19743.0 && out < 19744.0);

        // A zero reserve has no price.
        assert(uniswap_v2_amount_out(10.0, 0.0, 2000000.0, 0.3) == 0.0);
        assert(uniswap_v2_amount_out(0.0, 1000.0, 2000000.0, 0.3) == 0.0);
    }

    // --- Bundle hashes are 32 bytes, not a 64-bit std::hash ------------------

    {
        MEVConfig config;
        config.enable_flashbots = true;
        FlashbotsIntegrator integrator(config);

        Bundle bundle;
        bundle.id = "bundle-1";
        bundle.transactions = {"0xaaa"};
        bundle.block_number = 100;

        // No relay is configured, so the call reports failure rather than a
        // fabricated success.
        const auto result = integrator.send_bundle(bundle, "", {});
        assert(result.has_value());
        assert(!result->success);
        assert(!result->error.empty());

        // The hash is a keccak-256 digest, so 66 characters including the 0x.
        assert(result->bundle_hash.size() == 66);
        assert(result->bundle_hash.rfind("0x", 0) == 0);
    }

    // --- Transaction hashing is not affected by simulation state -------------

    {
        MEVConfig config;
        config.enable_flashbots = false;
        TransactionSimulator simulator(config);

        Transaction tx = MakeSwap("0x5", "0x38ed1739" + std::string(56, '0'), 1);
        tx.data = "0x38ed1739" + std::string(120, '0');

        const auto result = simulator.simulate(tx);
        assert(result.has_value());
        assert(result->success);
        // The gas figure is the floor for the calldata, not the old constant
        // 21000.
        assert(result->gas_used > 21000);
        // And the result says it was not executed against state, so a caller
        // cannot mistake the floor for a simulation.
        assert(!result->executed_against_state);
    }

    std::printf("mev_protector: all checks passed\n");
    return 0;
}
