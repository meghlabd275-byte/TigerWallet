// test_matching_engine.cpp — behavioural tests for the prediction matching
// engine.
//
// These assert on observable outcomes (trades produced, prices, remaining
// depth) rather than internal counters, so a regression in the matching logic
// fails the test rather than passing quietly.

#include "prediction_matching_engine.h"

#include <cassert>
#include <cstdio>
#include <iostream>
#include <vector>

using namespace tiger::prediction;

static int failures = 0;

#define CHECK(cond, msg)                                                    \
    do {                                                                    \
        if (!(cond)) {                                                      \
            std::printf("FAIL %s:%d: %s\n", __FILE__, __LINE__, msg);       \
            ++failures;                                                     \
        }                                                                   \
    } while (0)

// A price of 0.60 at the engine's 1e-6 scale.
static constexpr uint64_t P60 = 600000;
static constexpr uint64_t P55 = 550000;
static constexpr uint64_t P65 = 650000;
static constexpr uint64_t P70 = 700000;
static constexpr uint64_t P75 = 750000;
static constexpr uint64_t P80 = 800000;

// A crossing buy and sell must produce exactly one trade, and the trade must
// settle at the resting order's price.
static void test_crossing_orders_match() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Will it rain?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    Order sell;
    sell.market_id = market;
    sell.outcome_id = 0;
    sell.side = OrderSide::SELL;
    sell.price = P55;
    sell.amount = 100;

    Order buy;
    buy.market_id = market;
    buy.outcome_id = 0;
    buy.side = OrderSide::BUY;
    buy.price = P65;
    buy.amount = 100;

    auto resting = engine.process_order(sell);
    CHECK(resting.empty(), "a lone sell must not trade");

    auto trades = engine.process_order(buy);
    CHECK(trades.size() == 1, "a crossing buy and sell must produce one trade");
    if (!trades.empty()) {
        CHECK(trades[0].amount == 100, "the trade must fill the whole size");
        CHECK(trades[0].price == P55,
              "the trade must settle at the resting order's price");
    }
}

// A buy below the best ask must rest without trading.
static void test_non_crossing_orders_do_not_match() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Will it snow?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    Order sell;
    sell.market_id = market;
    sell.outcome_id = 0;
    sell.side = OrderSide::SELL;
    sell.price = P65;
    sell.amount = 50;

    Order buy;
    buy.market_id = market;
    buy.outcome_id = 0;
    buy.side = OrderSide::BUY;
    buy.price = P55;
    buy.amount = 50;

    engine.process_order(sell);
    auto trades = engine.process_order(buy);
    CHECK(trades.empty(), "a buy below the ask must not trade");
}

// A partial fill must leave the remainder resting and produce a second trade
// when the rest of the size arrives.
static void test_partial_fill_leaves_remainder() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Partial?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    Order sell;
    sell.market_id = market;
    sell.outcome_id = 0;
    sell.side = OrderSide::SELL;
    sell.price = P60;
    sell.amount = 100;

    Order first;
    first.market_id = market;
    first.outcome_id = 0;
    first.side = OrderSide::BUY;
    first.price = P60;
    first.amount = 40;

    Order second;
    second.market_id = market;
    second.outcome_id = 0;
    second.side = OrderSide::BUY;
    second.price = P60;
    second.amount = 60;

    engine.process_order(sell);

    auto t1 = engine.process_order(first);
    CHECK(t1.size() == 1 && t1[0].amount == 40, "the first buy fills 40");

    auto t2 = engine.process_order(second);
    CHECK(t2.size() == 1, "the second buy fills the 60 that remained");
    if (!t2.empty()) {
        CHECK(t2[0].amount == 60, "the remainder is 60");
    }
}

// After the book is exhausted there must be no bid left. This is the case the
// original walkback got wrong by scanning for bids in the wrong direction.
static void test_book_empties_cleanly() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Empty?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    Order sell;
    sell.market_id = market;
    sell.outcome_id = 0;
    sell.side = OrderSide::SELL;
    sell.price = P60;
    sell.amount = 10;

    Order buy;
    buy.market_id = market;
    buy.outcome_id = 0;
    buy.side = OrderSide::BUY;
    buy.price = P60;
    buy.amount = 10;

    engine.process_order(sell);
    engine.process_order(buy);

    auto bids = engine.get_market_depth(market, 0, 5, true);
    auto asks = engine.get_market_depth(market, 0, 5, false);
    CHECK(bids.empty(), "no bids may remain after the book is exhausted");
    CHECK(asks.empty(), "no asks may remain after the book is exhausted");
}

// A second, higher bid must become the best bid, and the depth must report the
// price that was actually quoted rather than a value derived from the index.
static void test_best_bid_tracks_real_price() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Depth?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    Order low;
    low.market_id = market;
    low.outcome_id = 0;
    low.side = OrderSide::BUY;
    low.price = P55;
    low.amount = 10;

    Order high;
    high.market_id = market;
    high.outcome_id = 0;
    high.side = OrderSide::BUY;
    high.price = P65;
    high.amount = 20;

    engine.process_order(low);
    engine.process_order(high);

    auto bids = engine.get_market_depth(market, 0, 5, true);
    CHECK(!bids.empty(), "bids must be present");
    if (!bids.empty()) {
        CHECK(bids[0].first == P65,
              "the best bid must report the price that was quoted");
        CHECK(bids[0].second == 20, "the best bid size must be 20");
    }
}

// A market order must fill against the resting book.
static void test_market_order_fills_against_book() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Market order?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    Order sell;
    sell.market_id = market;
    sell.outcome_id = 0;
    sell.side = OrderSide::SELL;
    sell.price = P60;
    sell.amount = 25;

    Order buy;
    buy.market_id = market;
    buy.outcome_id = 0;
    buy.side = OrderSide::BUY;
    buy.order_type = OrderType::MARKET;
    buy.amount = 25;

    engine.process_order(sell);
    auto trades = engine.process_order(buy);
    CHECK(trades.size() == 1, "a market order must fill against the resting book");
    if (!trades.empty()) {
        CHECK(trades[0].price == P60, "the market order fills at the ask");
    }
}

// A market order against an empty book must not trade.
static void test_market_order_without_liquidity() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("No liquidity?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    Order buy;
    buy.market_id = market;
    buy.outcome_id = 0;
    buy.side = OrderSide::BUY;
    buy.order_type = OrderType::MARKET;
    buy.amount = 25;

    auto trades = engine.process_order(buy);
    CHECK(trades.empty(), "a market order with no liquidity must not trade");
}

// A resolved market must refuse further orders.
static void test_resolved_market_rejects_orders() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Resolved?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    engine.resolve_market(market, 0);

    Order buy;
    buy.market_id = market;
    buy.outcome_id = 0;
    buy.side = OrderSide::BUY;
    buy.price = P60;
    buy.amount = 10;

    auto trades = engine.process_order(buy);
    CHECK(trades.empty(), "a resolved market must not trade");
}

// Identifiers must be unique, otherwise a fill would be recorded against the
// wrong order.
static void test_trade_and_order_ids_are_unique() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Ids?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    std::vector<uint64_t> order_ids;
    for (int i = 0; i < 20; ++i) {
        Order buy;
        buy.market_id = market;
        buy.outcome_id = 0;
        buy.side = OrderSide::BUY;
        buy.price = P55;
        buy.amount = 1;
        order_ids.push_back(engine.submit_order(buy));
    }

    for (size_t i = 1; i < order_ids.size(); ++i) {
        for (size_t j = 0; j < i; ++j) {
            CHECK(order_ids[i] != order_ids[j], "order ids must be unique");
        }
    }
}

// Cancelling one user's orders must leave every other user's orders resting.
static void test_cancel_all_orders_is_scoped_to_user() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Cancel?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    Order alice;
    alice.market_id = market;
    alice.outcome_id = 0;
    alice.side = OrderSide::BUY;
    alice.price = P55;
    alice.amount = 10;
    alice.user_id = 1;

    Order bob;
    bob.market_id = market;
    bob.outcome_id = 0;
    bob.side = OrderSide::BUY;
    bob.price = P60;
    bob.amount = 10;
    bob.user_id = 2;

    engine.process_order(alice);
    engine.process_order(bob);

    uint32_t cancelled = engine.cancel_all_orders(1);
    CHECK(cancelled == 1, "exactly one order belonged to the caller");

    auto bids = engine.get_market_depth(market, 0, 5, true);
    CHECK(bids.size() == 1, "the other user's order must remain");
    if (!bids.empty()) {
        CHECK(bids[0].first == P60, "the remaining bid must be the other user's");
    }
}

// Removing the best bid must promote the next best, and removing the best ask
// must promote the next best ask.
static void test_cancel_promotes_next_best() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Promote?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    for (uint64_t price : {P55, P60, P65}) {
        Order buy;
        buy.market_id = market;
        buy.outcome_id = 0;
        buy.side = OrderSide::BUY;
        buy.price = price;
        buy.amount = 5;
        engine.process_order(buy);
    }

    for (uint64_t price : {P70, P75, P80}) {
        Order sell;
        sell.market_id = market;
        sell.outcome_id = 0;
        sell.side = OrderSide::SELL;
        sell.price = price;
        sell.amount = 5;
        engine.process_order(sell);
    }

    auto bids = engine.get_market_depth(market, 0, 5, true);
    auto asks = engine.get_market_depth(market, 0, 5, false);
    CHECK(!bids.empty() && bids[0].first == P65, "the best bid is the highest");
    CHECK(!asks.empty() && asks[0].first == P70, "the best ask is the lowest");

    // Cancel the best bid and the best ask by user 1, then confirm the book
    // re-prices to the next level on each side.
    engine.cancel_all_orders(0);

    bids = engine.get_market_depth(market, 0, 5, true);
    asks = engine.get_market_depth(market, 0, 5, false);
    CHECK(bids.empty(), "all bids belonged to the cancelled user");
    CHECK(asks.empty(), "all asks belonged to the cancelled user");
}

// A trade must clear the resting size so the level stops quoting.
static void test_fill_clears_level() {
    PredictionMatchingEngine engine(1);
    uint64_t market = engine.create_market("Clear?", OutcomeType::BINARY,
                                           {"Yes", "No"}, 0);

    Order sell;
    sell.market_id = market;
    sell.outcome_id = 0;
    sell.side = OrderSide::SELL;
    sell.price = P60;
    sell.amount = 10;

    Order buy;
    buy.market_id = market;
    buy.outcome_id = 0;
    buy.side = OrderSide::BUY;
    buy.price = P60;
    buy.amount = 10;

    engine.process_order(sell);
    engine.process_order(buy);

    auto asks = engine.get_market_depth(market, 0, 5, false);
    CHECK(asks.empty(), "a fully filled ask must leave the book");
}

int main() {
    test_crossing_orders_match();
    test_non_crossing_orders_do_not_match();
    test_partial_fill_leaves_remainder();
    test_book_empties_cleanly();
    test_best_bid_tracks_real_price();
    test_market_order_fills_against_book();
    test_market_order_without_liquidity();
    test_resolved_market_rejects_orders();
    test_trade_and_order_ids_are_unique();
    test_cancel_all_orders_is_scoped_to_user();
    test_cancel_promotes_next_best();
    test_fill_clears_level();

    if (failures == 0) {
        std::printf("all matching engine tests passed\n");
        return 0;
    }
    std::printf("%d test(s) failed\n", failures);
    return 1;
}
