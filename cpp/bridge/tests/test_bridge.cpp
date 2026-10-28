// Tests for the cross-chain bridge quote layer.
//
// The point of these is the failure modes: an unregistered protocol, a token
// with no price, an invalid contract address, and the guarantee that a
// transaction identifier is never invented for a transaction that was not sent.

#include "bridge.hpp"

#include <cstdio>
#include <functional>
#include <string>

using namespace tigerwallet::bridge;

static int failures = 0;

#define CHECK(condition, label)                                            \
    do {                                                                   \
        if (!(condition)) {                                                \
            std::printf("FAIL: %s\n", label);                              \
            ++failures;                                                    \
        } else {                                                           \
            std::printf("ok:   %s\n", label);                              \
        }                                                                  \
    } while (0)

namespace {

const char* kUsdt = "0xdac17f958d2ee523a2206206994597c13d831ec7";
const char* kWeth = "0x0000000000000000000000000000000000000000";

// A price source with fixed values, so the arithmetic is checkable.
std::function<double(const std::string&, const std::string&)> fixed_oracle() {
    return [](const std::string& chain, const std::string& token) -> double {
        if (token == kUsdt) return 1.0;
        if (token == kWeth) return 2000.0;
        return 0.0;
    };
}

BridgeConfig make_config(BridgeProtocol protocol, const std::string& address,
                         double fee) {
    BridgeConfig config;
    config.protocol = protocol;
    config.name = "Test Bridge";
    config.api_base_url = "https://bridge.example.com";
    config.contract_address = address;
    config.fee_percentage = fee;
    config.min_amount = 0;
    config.max_amount = 1000000000;
    config.default_duration = 600;
    config.network_fee_usd = 2.0;
    return config;
}

// A bridge with no protocol registered produces no routes, rather than routes
// pointing at an invented contract.
void test_no_protocols_no_routes() {
    BridgeManager manager;
    manager.set_price_oracle(fixed_oracle());

    const auto quote = manager.get_quote("1", "56", kUsdt, kUsdt, 1000);
    CHECK(quote.has_value(), "a quote is returned even with no protocols");
    CHECK(quote->get_all_routes().empty(), "no protocol registered means no routes");

    CHECK(manager.get_protocols().empty(), "the protocol list starts empty");
    CHECK(manager.get_supported_chains().size() >= 8, "the chain registry is populated");
    CHECK(!manager.get_tokens("1").empty(), "the token registry is populated");
}

// A registered protocol produces a route, and the fee arithmetic is exact.
void test_quote_with_protocol() {
    BridgeManager manager;
    manager.set_price_oracle(fixed_oracle());
    manager.register_protocol(make_config(
        BridgeProtocol::Stargate, "0x1111111111111111111111111111111111111111", 0.01));

    const auto quote = manager.get_quote("1", "56", kUsdt, kUsdt, 10000);
    CHECK(quote.has_value(), "a quote is returned");
    const auto routes = quote->get_all_routes();
    CHECK(routes.size() == 1, "one registered protocol gives one route");

    const auto& route = routes[0];
    CHECK(route.bridge_fee == 100.0, "the bridge fee is one percent of the amount");
    // 10000 minus a 100 fee is 9900, and both tokens are a dollar.
    CHECK(route.to_amount == 9900.0, "the output amount accounts for the fee");
    // The network cost is 2 USD at a dollar per token.
    CHECK(route.network_fee == 2.0, "the network fee converts USD at the token price");
    CHECK(route.total_fee == 102.0, "the total fee is the sum");
    CHECK(route.bridge_address == "0x1111111111111111111111111111111111111111",
          "the route carries the configured contract address");
    CHECK(route.valid_until > 0, "the route carries an expiry");
}

// A cross-price route uses both prices.
void test_cross_price_route() {
    BridgeManager manager;
    manager.set_price_oracle(fixed_oracle());
    manager.register_protocol(make_config(
        BridgeProtocol::Portal, "0x2222222222222222222222222222222222222222", 0.0));

    // 2 WETH into USDT at 2000 per WETH is 4000 USDT. The fee is zero here.
    const auto quote = manager.get_quote("1", "56", kWeth, kUsdt, 2);
    CHECK(quote.has_value(), "a cross-price quote is returned");
    const auto routes = quote->get_all_routes();
    CHECK(routes.size() == 1, "the cross-price route is produced");
    CHECK(routes[0].to_amount == 4000.0, "the rate converts at the oracle price");
    CHECK(routes[0].network_fee == 0.001, "the network fee converts at 2000 per WETH");
}

// A token the oracle cannot price produces no route, not a guessed one.
void test_unpriced_token_no_route() {
    BridgeManager manager;
    manager.set_price_oracle(fixed_oracle());
    manager.register_protocol(make_config(
        BridgeProtocol::Stargate, "0x1111111111111111111111111111111111111111", 0.01));

    const char* unknown = "0x9999999999999999999999999999999999999999";
    const auto quote = manager.get_quote("1", "56", unknown, kUsdt, 100);
    if (quote.has_value()) {
        CHECK(quote->get_all_routes().empty(), "an unpriceable token produces no route");
    } else {
        CHECK(true, "an unpriceable token produces no route");
    }
}

// Without an oracle the module must not fall back to a stored price.
void test_no_oracle_no_fabricated_price() {
    BridgeManager manager;
    manager.register_protocol(make_config(
        BridgeProtocol::Stargate, "0x1111111111111111111111111111111111111111", 0.01));

    const auto quote = manager.get_quote("1", "56", kUsdt, kUsdt, 10000);
    if (quote.has_value()) {
        CHECK(quote->get_all_routes().empty(), "without an oracle no route is produced");
    } else {
        CHECK(true, "without an oracle no route is produced");
    }
}

// Registering a protocol validates its address and fee.
void test_protocol_validation() {
    BridgeManager manager;

    bool threw = false;
    try {
        manager.register_protocol(make_config(
            BridgeProtocol::Wormhole, "0xnotanaddress", 0.01));
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a malformed contract address is refused");

    threw = false;
    try {
        manager.register_protocol(make_config(
            BridgeProtocol::Wormhole, "0x1111111111111111111111111111111111111111", 1.5));
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a fee at or above 100 percent is refused");

    threw = false;
    try {
        BridgeConfig bad = make_config(
            BridgeProtocol::Wormhole, "0x1111111111111111111111111111111111111111", 0.01);
        bad.name.clear();
        manager.register_protocol(bad);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a protocol with no name is refused");

    threw = false;
    try {
        BridgeConfig bad = make_config(
            BridgeProtocol::Wormhole, "0x1111111111111111111111111111111111111111", 0.01);
        bad.min_amount = 100;
        bad.max_amount = 10;
        manager.register_protocol(bad);
    } catch (const std::exception&) {
        threw = true;
    }
    CHECK(threw, "a minimum above the maximum is refused");

    manager.register_protocol(make_config(
        BridgeProtocol::Wormhole, "0x1111111111111111111111111111111111111111", 0.01));
    CHECK(manager.get_protocols().size() == 1, "a valid protocol is registered");

    CHECK(manager.remove_protocol(BridgeProtocol::Wormhole), "a protocol is removable");
    CHECK(manager.get_protocols().empty(), "the removal takes effect");
    CHECK(!manager.remove_protocol(BridgeProtocol::Wormhole),
          "removing an absent protocol reports false");
}

// A transaction identifier is never invented.
void test_no_fabricated_hash() {
    BridgeManager manager;
    manager.set_price_oracle(fixed_oracle());
    manager.register_protocol(make_config(
        BridgeProtocol::Stargate, "0x1111111111111111111111111111111111111111", 0.01));

    const auto quote = manager.get_quote("1", "56", kUsdt, kUsdt, 10000);
    CHECK(quote.has_value(), "a quote is available");

    const auto route = quote->get_best_route();
    CHECK(route.has_value(), "a best route is available");

    const auto tx = manager.initiate_bridge(*route,
        "0x3333333333333333333333333333333333333333",
        "0x4444444444444444444444444444444444444444");
    CHECK(tx.has_value(), "the bridge transaction is created");
    CHECK(tx->tx_hash.empty(), "an unsent transaction carries no hash");
    CHECK(tx->status == BridgeStatus::Pending, "it starts pending");

    // The hash arrives only when the chain reports one.
    manager.update_transaction_status(
        tx->id, "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
        BridgeStatus::Confirmed);
    const auto updated = manager.get_transaction(tx->id);
    CHECK(updated.has_value(), "the transaction is retrievable");
    CHECK(updated->tx_hash.size() == 66, "the hash comes from the chain");
    CHECK(updated->confirmed_at > 0, "the confirmation time is recorded");
}

// Identifiers are unique even when two are created in quick succession.
void test_unique_ids() {
    BridgeManager manager;
    manager.set_price_oracle(fixed_oracle());
    manager.register_protocol(make_config(
        BridgeProtocol::Stargate, "0x1111111111111111111111111111111111111111", 0.01));

    const auto quote = manager.get_quote("1", "56", kUsdt, kUsdt, 10000);
    const auto route = quote->get_best_route();
    CHECK(route.has_value(), "a route is available");

    const auto first = manager.initiate_bridge(*route,
        "0x3333333333333333333333333333333333333333",
        "0x4444444444444444444444444444444444444444");
    const auto second = manager.initiate_bridge(*route,
        "0x3333333333333333333333333333333333333333",
        "0x4444444444444444444444444444444444444444");
    CHECK(first->id != second->id, "two transactions get distinct ids");

    const auto history = manager.get_transaction_history(
        "0x3333333333333333333333333333333333333333");
    CHECK(history.size() == 2, "both transactions appear in the history");
}

// The best-route selector picks the highest net output.
void test_best_route_selection() {
    BridgeQuote quote;
    BridgeRoute cheap;
    cheap.from_amount = 100.0;
    cheap.to_amount = 200.0;
    cheap.total_fee = 1.0;
    cheap.id = "cheap";

    BridgeRoute expensive;
    expensive.from_amount = 100.0;
    expensive.to_amount = 300.0;
    expensive.total_fee = 50.0;
    expensive.id = "expensive";

    quote.add_route(cheap);
    quote.add_route(expensive);

    const auto best = quote.get_best_route();
    CHECK(best.has_value(), "a best route is selected");
    // 300 - 50 is 250 against 200 - 1, so the expensive one nets more.
    CHECK(best->id == "expensive", "the best route is the highest net output");
}

}  // namespace

int main() {
    test_no_protocols_no_routes();
    test_quote_with_protocol();
    test_cross_price_route();
    test_unpriced_token_no_route();
    test_no_oracle_no_fabricated_price();
    test_protocol_validation();
    test_no_fabricated_hash();
    test_unique_ids();
    test_best_route_selection();

    if (failures == 0) {
        std::printf("\nAll bridge tests passed.\n");
        return 0;
    }
    std::printf("\n%d bridge test(s) failed.\n", failures);
    return 1;
}
