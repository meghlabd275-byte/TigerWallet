// Tests for the WlGate hot-path checker. Compiles + runs with g++ -std=c++20.
#include "wl_gate.hpp"
#include <cassert>
#include <cstdio>
#include <string>

using namespace tigerwallet::wl;

static void test_default_dead() {
    // A fresh gate must be dead (fail-closed) until a successful validate.
    WlGate& g = WlGate::instance();
    g.set_alive(false);
    assert(!g.is_alive());
    assert(!g.fetcher_enabled("user_wallet", "balances"));
    printf("ok: default dead (fail-closed)\n");
}

static void test_alive_permits_absent_flags() {
    WlGate& g = WlGate::instance();
    g.set_flags({});
    g.set_alive(true);
    assert(g.is_alive());
    assert(g.fetcher_enabled("user_wallet", "balances"));
    assert(g.fetcher_enabled("user_wallet", "transactions"));
    printf("ok: alive + absent flags => enabled (default-permit)\n");
}

static void test_whole_product_disable() {
    WlGate& g = WlGate::instance();
    g.set_flags({{"user_wallet", "*", false}});
    g.set_alive(true);
    assert(!g.fetcher_enabled("user_wallet", "balances"));
    assert(!g.fetcher_enabled("user_wallet", "transactions"));
    // other product still ok
    assert(g.fetcher_enabled("master_wallet", "balance"));
    printf("ok: whole-product ('*') disable blocks all fetchers of that product\n");
}

static void test_specific_fetcher_disable() {
    WlGate& g = WlGate::instance();
    g.set_flags({{"user_wallet", "swap_quote", false}});
    g.set_alive(true);
    assert(!g.fetcher_enabled("user_wallet", "swap_quote"));
    assert(g.fetcher_enabled("user_wallet", "balances"));
    printf("ok: specific fetcher disable blocks only that one\n");
}

static void test_dead_overrides_flags() {
    WlGate& g = WlGate::instance();
    g.set_flags({}); // all enabled
    g.set_alive(false, "halted by SuperAdmin");
    assert(!g.fetcher_enabled("user_wallet", "balances"));
    assert(g.reason() == "halted by SuperAdmin");
    printf("ok: dead => no fetcher serves, reason propagated\n");
}

static void test_c_abi() {
    wl_gate_set_alive(0, nullptr);
    assert(wl_gate_is_alive() == 0);
    assert(wl_gate_fetcher_enabled("user_wallet", "balances") == 0);

    wl_gate_set_flags("[{\"product\":\"user_wallet\",\"fetcher\":\"*\",\"enabled\":false}]");
    wl_gate_set_alive(1, nullptr);
    assert(wl_gate_is_alive() == 1);
    assert(wl_gate_fetcher_enabled("user_wallet", "balances") == 0);
    assert(wl_gate_fetcher_enabled("master_wallet", "balance") == 1);

    wl_gate_set_flags("[{\"product\":\"user_wallet\",\"fetcher\":\"swap_quote\",\"enabled\":false}]");
    assert(wl_gate_fetcher_enabled("user_wallet", "swap_quote") == 0);
    assert(wl_gate_fetcher_enabled("user_wallet", "balances") == 1);
    printf("ok: C ABI roundtrip\n");
}

int main() {
    test_default_dead();
    test_alive_permits_absent_flags();
    test_whole_product_disable();
    test_specific_fetcher_disable();
    test_dead_overrides_flags();
    test_c_abi();
    printf("ALL WLGATE TESTS PASSED\n");
    return 0;
}
