// Live check for the DEX connectors against a real node.
//
// This test is disabled by default because it needs network access. Run it
// with an endpoint and the deployed router/factory for that chain:
//
//   TIGER_RPC_URL=https://polygon-rpc.com
//   TIGER_DEX_ROUTER=0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff
//   TIGER_DEX_FACTORY=0x5757371414417b8C6CAad45bAeF941aBc60d2C9E
//   TIGER_DEX_CHAIN_ID=137
//   TIGER_DEX_TOKEN_IN=0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270
//   TIGER_DEX_TOKEN_OUT=0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174
//   ctest -R dex_live --output-on-failure

#include <cstdio>
#include <cstdlib>
#include <string>

#include "dex_connector.hpp"

using namespace tigerwallet::dex;

namespace {

const char* env_or(const char* key, const char* fallback) {
    const char* v = std::getenv(key);
    return (v != nullptr && *v != '\0') ? v : fallback;
}

}  // namespace

int main() {
    const std::string rpc_url = env_or("TIGER_RPC_URL", "");
    if (rpc_url.empty()) {
        std::printf("TIGER_RPC_URL is not set; skipping\n");
        return 0;
    }

    const std::string router = env_or("TIGER_DEX_ROUTER", "");
    const std::string factory = env_or("TIGER_DEX_FACTORY", "");
    const std::string token_in = env_or("TIGER_DEX_TOKEN_IN", "");
    const std::string token_out = env_or("TIGER_DEX_TOKEN_OUT", "");
    const uint64_t chain_id = std::strtoull(env_or("TIGER_DEX_CHAIN_ID", "1"), nullptr, 10);

    if (router.empty() || factory.empty() || token_in.empty() || token_out.empty()) {
        std::printf("router/factory/token env vars are not set; skipping\n");
        return 0;
    }

    auto connector = make_v2_connector(DexProtocol::UNISWAP_V2, rpc_url, chain_id,
                                       router, factory);

    if (!connector->connect()) {
        std::printf("FAIL connect (chain id mismatch or node unreachable)\n");
        return 1;
    }
    std::printf("connected to chain %llu\n",
                static_cast<unsigned long long>(connector->chain_id()));

    Token info;
    if (!connector->get_token_info(token_in, info)) {
        std::printf("FAIL get_token_info(%s)\n", token_in.c_str());
        return 1;
    }
    std::printf("token in: %s (%s), %u decimals\n",
                info.symbol.c_str(), info.name.c_str(), info.decimals);

    Pool pool;
    if (!connector->get_pool(token_in, token_out, pool)) {
        std::printf("FAIL get_pool: no pair for the supplied tokens\n");
        return 1;
    }
    std::printf("pair %s reserves %s / %s\n",
                pool.address.c_str(), pool.reserve_a.c_str(), pool.reserve_b.c_str());

    // Quote one whole unit of the input token.
    const std::string amount = "1000000000000000000";
    Quote quote;
    if (!connector->get_quote(token_in, token_out, amount, quote)) {
        std::printf("FAIL get_quote\n");
        return 1;
    }
    std::printf("quote: %s in -> %s out\n",
                quote.from_amount.c_str(), quote.to_amount.c_str());

    std::printf("live dex connector checks passed\n");
    return 0;
}
