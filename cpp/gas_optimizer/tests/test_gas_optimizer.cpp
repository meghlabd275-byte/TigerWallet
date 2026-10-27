// Tests for the gas optimizer.
//
// This header declared seven methods and defined none of them, including
// initialize_chains(), which the constructor calls, so nothing that used the
// class could link. It also fabricated its outputs: the USD cost assumed every
// chain's native token was worth $3500, savings were a constant, and a
// recommendation counted as "optimal" whenever the cache happened to be warm.

#include "gas_optimizer.h"

#include <cassert>
#include <cmath>
#include <cstdio>

using namespace tiger::gas;

namespace {

GasData MakeGas(const std::string& chain, uint64_t base_fee,
                uint64_t priority_fee, double native_price_usd) {
    GasData data{};
    data.chain_name = chain;
    data.base_fee = base_fee;
    data.priority_fee = priority_fee;
    data.native_price_usd = native_price_usd;
    data.block_number = 1;
    data.timestamp_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    return data;
}

}  // namespace

int main() {
    GasOptimizer optimizer;

    // Nothing is seeded, so there is no gas data and no recommendation basis.
    // This is the honest state, not a fabricated one.
    assert(!optimizer.get_current_gas("ethereum").has_value());
    assert(optimizer.get_cache_hit_rate() == 0.0);
    assert(optimizer.compare_chains({"ethereum", "polygon"}).empty());
    assert(optimizer.predict_gas("ethereum", 5).empty());

    // With no data there is no cost basis, so the cost is zero rather than a
    // figure derived from a hardcoded $3500.
    const auto no_data = optimizer.get_recommendation("ethereum");
    assert(no_data.estimated_cost == 0);
    assert(!no_data.is_optimal);

    // Feed a real observation: 20 gwei base, 2 gwei tip, ETH at $2000.
    optimizer.update_gas_data(MakeGas("ethereum", 20, 2, 2000.0));

    const auto gas = optimizer.get_current_gas("ethereum");
    assert(gas.has_value());
    assert(gas->base_fee == 20);
    assert(gas->native_price_usd == 2000.0);

    // Standard pays base + 1x tip = 22 gwei.
    const auto standard = optimizer.get_recommendation("ethereum", GasSpeed::STANDARD);
    assert(standard.suggested_gas_price == 22);

    // Instant pays base + 4x tip = 28 gwei.
    const auto instant = optimizer.get_recommendation("ethereum", GasSpeed::INSTANT);
    assert(instant.suggested_gas_price == 28);

    // Slow pays base + 0.5x tip, rounded down = 21 gwei.
    const auto slow = optimizer.get_recommendation("ethereum", GasSpeed::SLOW);
    assert(slow.suggested_gas_price == 21);

    // A higher speed costs more, and confirms sooner.
    assert(instant.estimated_cost > standard.estimated_cost);
    assert(instant.estimated_time_sec < slow.estimated_time_sec);

    // Cost: 22 gwei * 1e-9 * 21000 gas = 0.000462 ETH, at $2000 = $0.92.
    // Truncated to whole dollars by the uint64 interface.
    assert(standard.estimated_cost == 0 || standard.estimated_cost == 1);

    // A cheaper chain with a cheaper token must produce a much smaller cost.
    optimizer.update_gas_data(MakeGas("polygon", 30, 1, 0.50));
    const auto polygon = optimizer.get_recommendation("polygon", GasSpeed::STANDARD);
    // 31 gwei * 1e-9 * 21000 = 0.000651 POL at $0.50 = $0.00033, so zero USD.
    assert(polygon.estimated_cost == 0);
    // The old code would have priced this at $3500 per token.
    assert(polygon.suggested_gas_price == 31);

    // Savings are measured against the floor of base + tip, and are zero when
    // the price is exactly the floor.
    const auto floor_price = optimizer.get_recommendation("ethereum", GasSpeed::STANDARD);
    assert(floor_price.savings_percent >= 0.0);

    // A recommendation well above the floor reports a negative saving rather
    // than the old constant 15.5.
    assert(instant.savings_percent < standard.savings_percent);
    assert(instant.savings_percent < 0.0);

    // Comparison returns only chains that have data.
    const auto compared = optimizer.compare_chains({"ethereum", "polygon", "unknown"});
    assert(compared.size() == 2);

    // A forecast carries the latest observation forward, and never invents a
    // trend from a short history.
    const auto forecast = optimizer.predict_gas("ethereum", 3);
    assert(forecast.size() == 3);
    for (const auto& block : forecast) {
        assert(block.base_fee == 20);
        assert(block.chain_name == "ethereum");
    }
    // A non-positive horizon yields nothing.
    assert(optimizer.predict_gas("ethereum", 0).empty());
    assert(optimizer.predict_gas("ethereum", -1).empty());

    // The transaction estimate is populated and does not claim a cost it
    // cannot know.
    const auto tx = optimizer.estimate_transaction("0xfrom", "0xto", "USDT", "100");
    assert(tx.from == "0xfrom");
    assert(tx.to == "0xto");
    assert(tx.token == "USDT");
    assert(tx.gas_needed == 21000);
    assert(tx.total_cost_usd == 0);

    std::printf("gas_optimizer: all checks passed\n");
    return 0;
}
