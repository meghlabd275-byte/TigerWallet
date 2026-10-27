// Tests for the ultra-low-latency trading engine.
//
// There was no build file and no test for this header, which is why it had
// never been compiled: it did not compile, and constructing the engine
// crashed because it held 32 MB of tables and a 4 MB ring buffer by value.

#include "trading_engine.hpp"

#include <cassert>
#include <cstdio>
int main() {
    UltraLowLatencyTradingEngine engine;
    // Nothing is pre-registered, so the engine starts empty.
    assert(engine.getTotalPairs() == 0);
    assert(engine.getPreInstalledPairs().empty());
    assert(engine.getTradingPair("BTC/USDT") == nullptr);
    assert(engine.getPrice("BTC/USDT") == 0.0);
    // Register a pair with a real price and read it back.
    engine.addTradingPair("BTC/USDT", "BTC", "USDT", 43000.0, true);
    assert(engine.getTotalPairs() == 1);
    assert(engine.getPreInstalledPairs().size() == 1);
    assert(engine.getTradingPair("BTC/USDT") != nullptr);
    assert(engine.getPrice("BTC/USDT") == 43000.0);
    engine.updatePrice("BTC/USDT", 44000.0);
    assert(engine.getPrice("BTC/USDT") == 44000.0);
    // Positions were unreachable before openPosition existed.
    assert(engine.getPosition(7, "BTC/USDT") == nullptr);
    auto* p = engine.openPosition(7, "BTC/USDT", PositionSide::LONG, 2.0, 40000.0, 4);
    assert(p != nullptr);
    assert(engine.getPosition(7, "BTC/USDT") == p);
    assert(engine.getUserPositions(7).size() == 1);
    assert(engine.getUserPositions(8).empty());
    // 2 * 40000 / 4 = 20000 margin.
    assert(p->margin == 20000.0);
    // Order lifecycle.
    auto id = engine.createOrder(7, "BTC/USDT", OrderSide::BUY,
                                 OrderType::LIMIT, 1.0, 43000.0, 1);
    assert(id != 0);
    assert(engine.cancelOrder(id));
    assert(!engine.cancelOrder(id));

    // The engine object must be small enough to construct on a normal stack.
    // Before the tables and the ring buffer moved to the heap this was 32 MB.
    assert(sizeof(UltraLowLatencyTradingEngine) < 64 * 1024);

    std::printf("trading_engine: all checks passed\n");
    return 0;
}
