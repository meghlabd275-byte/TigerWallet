// Tests for the risk engine.
//
// This module had no build file and no test, and did not compile. Three of its
// checks were placeholders: concentration limits always passed, counterparty
// exposure was always zero, and the correlation adjustment was a constant.

#include "risk_engine.h"

#include <cassert>
#include <cmath>
#include <cstdio>

using namespace tiger::risk;

namespace {

Account MakeAccount(const std::string& id, double balance) {
    Account account;
    account.account_id = id;
    account.balance = balance;
    return account;
}

Position MakePosition(const std::string& id, const std::string& trader,
                      const std::string& symbol, double size, double price) {
    Position position;
    position.position_id = id;
    position.trader_id = trader;
    position.symbol = symbol;
    position.size = size;
    position.current_price = price;
    position.entry_price = price;
    return position;
}

}  // namespace

int main() {
    RiskConfig config;
    config.max_single_asset_concentration = 0.30;
    config.max_daily_volume = 1000.0;
    config.max_position_size = 10000000.0;
    config.max_order_size = 1000000.0;
    config.max_leverage = 10.0;
    RiskEngine engine(config);

    // Counterparty exposure must reflect real positions, not a constant zero.
    assert(engine.get_counterparty_exposure("cp-1") == 0.0);
    engine.update_position(MakePosition("p1", "cp-1", "BTC/USDT", 1.0, 50000.0));
    assert(engine.get_counterparty_exposure("cp-1") == 50000.0);
    // A different counterparty is unaffected.
    assert(engine.get_counterparty_exposure("cp-2") == 0.0);

    // Concentration is enforced through validate_order, which is the public
    // entry point. The book below is 50k of BTC against 3k of ETH, so a
    // further 1 BTC would be 100k of a 103k book.
    engine.update_price(PriceData{"BTC/USDT", 50000.0});
    engine.update_price(PriceData{"ETH/USDT", 3000.0});
    engine.update_position(MakePosition("p3", "cp-2", "ETH/USDT", 1.0, 3000.0));

    Account account = MakeAccount("t1", 10000000.0);
    account.available_balance = 10000000.0;
    engine.register_account(account);

    Order heavy;
    heavy.order_id = "o1";
    heavy.symbol = "BTC/USDT";
    heavy.trader_id = "t1";
    heavy.side = OrderSide::BUY;
    heavy.type = OrderType::LIMIT;
    heavy.size = 1.0;
    heavy.price = 50000.0;

    const auto rejected = engine.validate_order(heavy, account);
    // Before the fix check_concentration_limits returned true unconditionally,
    // so this order was approved.
    assert(!rejected.approved);
    assert(rejected.reject_reason == OrderRejectReason::CONCENTRATION_LIMIT);

    // A book where the symbol is a small share of the total accepts the same
    // order, so the check discriminates rather than rejecting everything.
    RiskEngine wide(config);
    wide.update_price(PriceData{"BTC/USDT", 100.0});
    wide.update_price(PriceData{"ETH/USDT", 100.0});
    wide.update_position(MakePosition("w1", "cp-1", "ETH/USDT", 1.0, 100.0));
    wide.update_position(MakePosition("w2", "cp-2", "ETH/USDT", 1.0, 100.0));
    wide.update_position(MakePosition("w3", "cp-3", "ETH/USDT", 1.0, 100.0));
    Account wide_account = MakeAccount("t2", 10000000.0);
    wide_account.available_balance = 10000000.0;
    wide.register_account(wide_account);

    Order small = heavy;
    small.trader_id = "t2";
    small.size = 1.0;
    small.price = 100.0;
    const auto accepted = wide.validate_order(small, wide_account);
    // 100 of BTC against a 400 gross book is 25%, under the 30% allowance.
    assert(accepted.approved);

    // RiskManager hands out a usable engine rather than a dangling one.
    RiskManager& manager = RiskManager::get_instance();
    manager.initialize(config);
    assert(manager.get_engine().get_counterparty_exposure("nobody") == 0.0);
    manager.shutdown();

    std::printf("risk_engine: all checks passed\n");
    return 0;
}
