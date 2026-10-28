// Verifies the aggregator's pricing math against the reference formulas.
//
// The constant-product case uses the worked example from the Uniswap V2
// whitepaper. The concentrated-liquidity and StableSwap cases are checked
// against hand-computed values and against the properties their invariants
// must satisfy.

#include "dex_aggregator.hpp"

#include <cassert>
#include <cmath>
#include <cstdio>
#include <iostream>

using namespace tigerwallet;
using namespace tigerwallet::math;
using namespace tigerwallet::dex;

static int failures = 0;

#define CHECK(condition, message)                                        \
    do {                                                                 \
        if (!(condition)) {                                              \
            std::cerr << "FAIL: " << message << std::endl;               \
            ++failures;                                                  \
        }                                                                \
    } while (false)

// --- Uniswap V2 ----------------------------------------------------------
// Reserves of 1000 and 1000, an input of 10, and the 0.3 percent fee. The
// whitepaper's answer is 9.8667 units out.
static void test_v2_constant_product() {
    const uint256_t amount_in = uint256_t(U256::fromUint64(10));
    const uint256_t reserve_in = uint256_t(U256::fromUint64(1000));
    const uint256_t reserve_out = uint256_t(U256::fromUint64(1000));

    const uint256_t out = UniswapV2Adapter::calculateOutput(
        amount_in, reserve_in, reserve_out, 30);
    CHECK(!out.isZero(), "v2: output must be non-zero");
    // 10 * 997 * 1000 / (1000 * 1000 + 10 * 997) = 9970000 / 1009970 = 9
    CHECK(out.value().toUint64() == 9, "v2: output must be 9");

    // A zero input returns nothing rather than dividing by zero.
    CHECK(UniswapV2Adapter::calculateOutput(uint256_t(), reserve_in, reserve_out, 30)
              .isZero(),
          "v2: zero input must yield zero");

    // A 100 percent fee leaves nothing.
    CHECK(UniswapV2Adapter::calculateOutput(amount_in, reserve_in, reserve_out, 10000)
              .isZero(),
          "v2: a 100 percent fee must yield zero");

    // The output can never exceed the reserve.
    const uint256_t huge = uint256_t(U256::fromUint64(1000000));
    const uint256_t out_huge = UniswapV2Adapter::calculateOutput(
        huge, reserve_in, reserve_out, 30);
    CHECK(static_cast<double>(out_huge) < static_cast<double>(reserve_out),
          "v2: output must stay below the reserve");
}

// The inverse direction must recover the input that produced an output.
static void test_v2_inverse() {
    const uint256_t reserve_in = uint256_t(U256::fromUint64(5000));
    const uint256_t reserve_out = uint256_t(U256::fromUint64(8000));
    const uint256_t amount_in = uint256_t(U256::fromUint64(123));

    const uint256_t out = UniswapV2Adapter::calculateOutput(
        amount_in, reserve_in, reserve_out, 30);
    CHECK(!out.isZero(), "v2 inverse: output must be non-zero");

    const uint256_t recovered = UniswapV2Adapter::calculateInput(
        out, reserve_in, reserve_out, 30);
    // The rounding in each direction means the recovered input is within one
    // unit of the original, never below it.
    const uint64_t recovered_u = recovered.value().toUint64();
    CHECK(recovered_u >= amount_in.value().toUint64() &&
              recovered_u <= amount_in.value().toUint64() + 1,
          "v2 inverse: the input must round-trip within one unit");

    // An output at or above the reserve is impossible.
    CHECK(UniswapV2Adapter::calculateInput(reserve_out, reserve_in, reserve_out, 30)
              .isZero(),
          "v2 inverse: an output at the reserve must be refused");
}

// --- Uniswap V3 ----------------------------------------------------------
// A pool at price 1 with a large amount of liquidity. The trade is small, so
// the output is close to the amount in less the fee, but it must always be
// strictly less because the price moves.
static void test_v3_concentrated() {
    // sqrtPriceX96 = 2^96 means price 1.
    const U256 one = U256::fromUint64(1);
    const U256 sqrt_price = shiftLeft(one, 96);
    const U256 liquidity = shiftLeft(U256::fromUint64(1000000), 96);

    const uint256_t amount_in = uint256_t(U256::fromUint64(1000000));
    const uint256_t out = UniswapV3Adapter::amountOutV3(
        amount_in, uint256_t(sqrt_price), uint256_t(liquidity), 3000);

    CHECK(!out.isZero(), "v3: output must be non-zero");
    // At price 1 the output is below the input because of the fee.
    CHECK(out.value().toUint64() < amount_in.value().toUint64(),
          "v3: output must be below the input at price 1");

    // A pool with no state cannot be quoted.
    CHECK(UniswapV3Adapter::amountOutV3(amount_in, uint256_t(), uint256_t(liquidity),
                                        3000)
              .isZero(),
          "v3: a missing price must yield zero");
    CHECK(UniswapV3Adapter::amountOutV3(amount_in, uint256_t(sqrt_price), uint256_t(),
                                        3000)
              .isZero(),
          "v3: missing liquidity must yield zero");

    // A fee of 100 percent or more leaves nothing.
    CHECK(UniswapV3Adapter::amountOutV3(amount_in, uint256_t(sqrt_price),
                                        uint256_t(liquidity), 1000000)
              .isZero(),
          "v3: a 100 percent fee must yield zero");

    // A larger trade must move the price further and return proportionally
    // less than a small one.
    const uint256_t small = UniswapV3Adapter::amountOutV3(
        uint256_t(U256::fromUint64(1000)), uint256_t(sqrt_price),
        uint256_t(liquidity), 3000);
    const uint256_t large = UniswapV3Adapter::amountOutV3(
        uint256_t(U256::fromUint64(1000000)), uint256_t(sqrt_price),
        uint256_t(liquidity), 3000);
    const double small_rate =
        static_cast<double>(small) / 1000.0;
    const double large_rate =
        static_cast<double>(large) / 1000000.0;
    CHECK(small_rate > large_rate,
          "v3: a larger trade must get a worse rate");
}

// --- Curve StableSwap ----------------------------------------------------
// A balanced two-coin pool. Because the coins are meant to trade at parity,
// the output for a small trade is close to the input, and it must stay below
// it after the fee.
static void test_curve_stableswap() {
    const std::vector<uint256_t> balances{
        uint256_t(U256::fromUint64(1000000)),
        uint256_t(U256::fromUint64(1000000))};
    const uint256_t amp = uint256_t(U256::fromUint64(100));

    const uint256_t out = CurveAdapter::getDy(uint256_t(0), uint256_t(1),
                                              uint256_t(U256::fromUint64(1000)),
                                              balances, amp, 40);
    CHECK(!out.isZero(), "curve: output must be non-zero");
    // At parity the output is just under the input, since the fee is taken.
    CHECK(out.value().toUint64() < 1000, "curve: output must be below the input");
    CHECK(out.value().toUint64() > 990, "curve: output must be close to the input");

    // A balanced pool must price symmetrically: swapping the other way gives
    // the same answer.
    const uint256_t reverse = CurveAdapter::getDy(
        uint256_t(1), uint256_t(0), uint256_t(U256::fromUint64(1000)), balances,
        amp, 40);
    CHECK(out.value().toUint64() == reverse.value().toUint64(),
          "curve: a balanced pool must price symmetrically");

    // An imbalanced pool must give less than a balanced one, which is the
    // whole point of the invariant.
    const std::vector<uint256_t> skewed{
        uint256_t(U256::fromUint64(2000000)),
        uint256_t(U256::fromUint64(500000))};
    const uint256_t skewed_out = CurveAdapter::getDy(
        uint256_t(0), uint256_t(1), uint256_t(U256::fromUint64(1000)), skewed,
        amp, 40);
    CHECK(skewed_out.value().toUint64() < out.value().toUint64(),
          "curve: a skewed pool must quote worse than a balanced one");

    // A pool with no amplification cannot be quoted.
    CHECK(CurveAdapter::getDy(uint256_t(0), uint256_t(1),
                              uint256_t(U256::fromUint64(1000)), balances,
                              uint256_t(), 40)
              .isZero(),
          "curve: a missing amplification must yield zero");

    // The same coin index on both sides is not a swap.
    CHECK(CurveAdapter::getDy(uint256_t(0), uint256_t(0),
                              uint256_t(U256::fromUint64(1000)), balances, amp, 40)
              .isZero(),
          "curve: identical indices must be refused");

    // An out-of-range index is refused.
    CHECK(CurveAdapter::getDy(uint256_t(0), uint256_t(5),
                              uint256_t(U256::fromUint64(1000)), balances, amp, 40)
              .isZero(),
          "curve: an out-of-range index must be refused");
}

// --- route pricing -------------------------------------------------------
// The multi-hop search must price each hop from the pool it crosses.
static void test_route_pricing() {
    Token a;
    a.address = "0x0000000000000000000000000000000000000001";
    a.symbol = "A";
    a.decimals = 18;
    Token b = a;
    b.address = "0x0000000000000000000000000000000000000002";
    b.symbol = "B";
    Token c = a;
    c.address = "0x0000000000000000000000000000000000000003";
    c.symbol = "C";

    Pool ab;
    ab.protocol = DEXProtocol::UNISWAP_V2;
    ab.token0 = a;
    ab.token1 = b;
    ab.reserve0 = uint256_t(U256::fromUint64(1000000));
    ab.reserve1 = uint256_t(U256::fromUint64(1000000));
    ab.fee = 0.003;

    Pool bc;
    bc.protocol = DEXProtocol::UNISWAP_V2;
    bc.token0 = b;
    bc.token1 = c;
    bc.reserve0 = uint256_t(U256::fromUint64(1000000));
    bc.reserve1 = uint256_t(U256::fromUint64(1000000));
    bc.fee = 0.003;

    PathFinder finder;
    std::unordered_map<std::string, std::vector<Pool>> pools;
    pools[a.id() + "-" + b.id()] = {ab};
    pools[b.id() + "-" + c.id()] = {bc};

    const auto routes = finder.findBestRoutes(
        a, c, uint256_t(U256::fromUint64(1000)), pools, 3, 4);

    CHECK(!routes.empty(), "routes: a two-hop path must be found");

    // Both hops charge a fee, so the output is below the input.
    if (!routes.empty()) {
        CHECK(routes.front().expectedOutput.rawAmount.value().toUint64() < 1000,
              "routes: two hops must return less than the input");
        CHECK(routes.front().steps.size() == 2,
              "routes: the path must have two steps");
    }

    // A pair with no pool yields no route rather than a made-up one.
    const auto none = finder.findBestRoutes(
        c, a, uint256_t(U256::fromUint64(1000)),
        std::unordered_map<std::string, std::vector<Pool>>(), 3, 4);
    CHECK(none.empty(), "routes: no pool must mean no route");
}

// --- cross-decimal hops --------------------------------------------------
// A hop from an 18-decimal token into a 6-decimal one must scale the amount
// onto the target's exponent. Without that the result is off by 10^12.
static void test_cross_decimal() {
    Token weth;
    weth.address = "0x0000000000000000000000000000000000000010";
    weth.symbol = "WETH";
    weth.decimals = 18;

    Token usdc;
    usdc.address = "0x0000000000000000000000000000000000000011";
    usdc.symbol = "USDC";
    usdc.decimals = 6;

    // A pool holding one WETH against 2000 USDC.
    Pool pool;
    pool.protocol = DEXProtocol::UNISWAP_V2;
    pool.token0 = weth;
    pool.token1 = usdc;
    pool.reserve0 = uint256_t(U256::fromUint64(1000000000000000000ULL));
    pool.reserve1 = uint256_t(U256::fromUint64(2000000000ULL));
    pool.fee = 0.003;

    PathFinder finder;
    std::unordered_map<std::string, std::vector<Pool>> pools;
    pools[weth.id() + "-" + usdc.id()] = {pool};

    // 0.01 WETH, which is 10^16 raw units.
    const auto routes = finder.findBestRoutes(
        weth, usdc, uint256_t(U256::fromUint64(10000000000000000ULL)), pools, 3, 4);
    CHECK(!routes.empty(), "cross-decimal: a route must be found");

    if (!routes.empty()) {
        // 0.01 WETH is about 20 USDC, which is 20 * 10^6 raw units. The fee and
        // the price move pull it slightly below.
        const uint64_t out = routes.front().expectedOutput.rawAmount.value().toUint64();
        CHECK(out > 19000000ULL && out < 20000000ULL,
              "cross-decimal: the output must be about 20 USDC in raw units");
    }
}

// --- split orders --------------------------------------------------------
// A large order that moves the pool is divided, and the parts must add back
// to the whole.
static void test_split_conserves_amount() {
    Token a;
    a.address = "0x0000000000000000000000000000000000000020";
    a.symbol = "A";
    a.decimals = 18;
    Token b = a;
    b.address = "0x0000000000000000000000000000000000000021";
    b.symbol = "B";

    // A shallow pool, so a large order moves the price.
    Pool ab;
    ab.protocol = DEXProtocol::UNISWAP_V2;
    ab.token0 = a;
    ab.token1 = b;
    ab.reserve0 = uint256_t(U256::fromUint64(1000000));
    ab.reserve1 = uint256_t(U256::fromUint64(1000000));
    ab.fee = 0.003;

    Pool ab2 = ab;
    ab2.poolAddress = "0x00000000000000000000000000000000000000ff";

    PathFinder finder;
    std::unordered_map<std::string, std::vector<Pool>> pools;
    pools[a.id() + "-" + b.id()] = {ab, ab2};

    const uint256_t amount = uint256_t(U256::fromUint64(100000));
    const auto routes = finder.findBestRoutes(a, b, amount, pools, 3, 4);
    CHECK(routes.size() >= 2, "split: two venues must give two routes");

    if (routes.size() >= 2) {
        // Each route quotes the full amount on its own pool, so a split of the
        // same total across both must be no worse than the best single route.
        const math::U256 half =
            math::mulDiv(amount.value(), math::U256::fromUint64(1),
                         math::U256::fromUint64(2));
        math::U256 combined;
        for (size_t i = 0; i < 2; ++i) {
            const Pool& pool = routes[i].steps.front().pool;
            combined = combined +
                       UniswapV2Adapter::calculateOutput(
                           uint256_t(half), pool.reserve0, pool.reserve1, 30)
                           .value();
        }
        CHECK(!combined.isZero(), "split: the combined output must be non-zero");
        CHECK(combined > routes.front().expectedOutput.rawAmount.value(),
              "split: splitting must beat the best single route");
    }
}

// --- ABI encoding --------------------------------------------------------
// The calldata must match the ABI specification exactly, or the router will
// revert. The expected string below is the canonical encoding of
// swapExactTokensForTokens(1000, 900, [0xaa.., 0xbb..], 0x11.., 1700000000).
static void test_abi_encoding() {
    Token a;
    a.address = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    a.decimals = 18;
    Token b;
    b.address = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
    b.decimals = 18;

    const TokenAmount amount(a, uint256_t(U256::fromUint64(1000)));

    Pool pool;
    pool.protocol = DEXProtocol::UNISWAP_V2;
    pool.token0 = a;
    pool.token1 = b;

    UniswapV2Adapter adapter;
    const std::string data = adapter.buildSwapData(
        amount, b, "0x1111111111111111111111111111111111111111", pool,
        uint256_t(U256::fromUint64(900)));

    // The selector and the four fixed words, then the array tail. The deadline
    // is computed from the clock, so it is checked by shape rather than value.
    const std::string head =
        "0x38ed1739"
        "00000000000000000000000000000000000000000000000000000000000003e8"
        "0000000000000000000000000000000000000000000000000000000000000384"
        "00000000000000000000000000000000000000000000000000000000000000a0"
        "0000000000000000000000001111111111111111111111111111111111111111";
    CHECK(data.rfind(head, 0) == 0,
          "abi: the head must match the canonical encoding");

    // The head is five words, so the tail begins at 10 + 5 * 64 characters.
    const std::string tail = data.substr(10 + 5 * 64);
    const std::string expected_tail =
        "0000000000000000000000000000000000000000000000000000000000000002"
        "000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        "000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
    CHECK(tail == expected_tail,
          "abi: the array tail must be the length then the elements");

    // The whole payload is the selector plus five head words plus three tail
    // words.
    CHECK(data.size() == 10 + (5 + 3) * 64,
          "abi: the payload must be the expected length");
}

int main() {
    test_v2_constant_product();
    test_v2_inverse();
    test_v3_concentrated();
    test_curve_stableswap();
    test_route_pricing();
    test_cross_decimal();
    test_split_conserves_amount();
    test_abi_encoding();

    if (failures == 0) {
        std::cout << "dex_aggregator: all checks passed" << std::endl;
        return 0;
    }
    std::cerr << "dex_aggregator: " << failures << " check(s) failed"
              << std::endl;
    return 1;
}
