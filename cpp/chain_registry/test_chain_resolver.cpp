// Smoke + correctness test for the ultra-low-latency ChainResolver.
// Compiles standalone: g++ -std=c++20 -O2 -I. chain_registry_data.cpp test_chain_resolver.cpp -o test_chain_resolver
#include "ChainResolver.hpp"
#include <cassert>
#include <cstdio>

int main() {
    using namespace TigerWallet::ChainRegistry;
    ChainResolver r;

    // Minimums: >=100 EVM, >=50 non-EVM, total >=150.
    assert(r.evmCount() >= 100);
    assert(r.nonEvmCount() >= 50);
    assert(r.total() >= 150);
    std::printf("evm=%zu nonevm=%zu total=%zu\n", r.evmCount(), r.nonEvmCount(), r.total());

    // Ethereum lookup.
    const auto* eth = r.findById(1);
    assert(eth != nullptr);
    assert(eth->symbol == "ETH");
    assert(eth->type == ChainType::Evm);
    assert(eth->coin_type == 60);

    // Pi Network present.
    bool pi = false;
    for (const auto& c : r.all()) if (c.type == ChainType::Pi) { pi = true; break; }
    assert(pi);

    // Unknown id returns nullptr.
    assert(r.findById(999999999) == nullptr);

    // Search is case-insensitive.
    auto hits = r.search("bitcoin");
    assert(!hits.empty());

    // loadExtra merges an admin-added chain and is then resolvable.
    ChainEntry custom{123456789, "AdminChain", "ADM", ChainType::Evm,
                      "https://rpc.example", "https://explorer.example", 18, 60,
                      "m/44'/60'/0'/0/0"};
    r.loadExtra({custom});
    assert(r.findById(123456789) != nullptr);

    std::printf("OK: chain resolver verified\n");
    return 0;
}
