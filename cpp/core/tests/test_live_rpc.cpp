// Live check against a real Ethereum-compatible node.
//
// This is excluded from the default test run: it needs network access and a
// working endpoint. Set TIGER_RPC_URL and run
//   ctest -R live_rpc --output-on-failure
// or invoke the binary directly with the URL as the first argument.
//
// The point of this test is that it asserts against values the node actually
// returns. A node that is unreachable is reported as a failure, not skipped,
// because a silently skipped integration check is worse than no check.

#include <cstdint>
#include <iostream>
#include <string>

#include "crypto.hpp"
#include "rpc.hpp"

using namespace tigerwallet;

namespace {

int failures = 0;

void Check(const std::string& label, bool ok) {
    if (!ok) ++failures;
    std::cout << (ok ? "  PASS " : "  FAIL ") << label << "\n";
}

}  // namespace

int main(int argc, char** argv) {
    std::string url;
    if (argc > 1) {
        url = argv[1];
    } else if (const char* env = std::getenv("TIGER_RPC_URL")) {
        url = env;
    } else {
        std::cout << "live_rpc: set TIGER_RPC_URL or pass a URL; skipping\n";
        return 0;
    }

    std::cout << "live RPC against " << url << "\n";
    const RpcClient client(url, 20);

    const auto chain_id = client.ChainId();
    Check("eth_chainId succeeds", chain_id.ok);
    if (!chain_id.ok) {
        std::cout << "        error: " << chain_id.error.message << "\n";
        std::cout << "live_rpc: FAILED\n";
        return 1;
    }
    std::cout << "        chain id " << chain_id.value << "\n";

    const auto block = client.BlockNumber();
    Check("eth_blockNumber succeeds", block.ok);
    if (block.ok) std::cout << "        block " << block.value << "\n";

    const auto gas_price = client.GetGasPrice();
    Check("eth_gasPrice succeeds", gas_price.ok);
    Check("gas price is a valid quantity",
          gas_price.ok && crypto::IsValidHexQuantity(gas_price.value));
    if (gas_price.ok) {
        std::cout << "        gas price " << crypto::HexQuantityToDecimal(gas_price.value)
                  << " wei\n";
    }

    // The Eth2 deposit contract holds far more than 2^64 wei, so this is the
    // case that a 64-bit balance parser gets wrong.
    const std::string deposit_contract = "0x00000000219ab540356cBB839Cbe05303d7705Fa";
    const auto balance = client.GetBalance(deposit_contract);
    Check("eth_getBalance succeeds for a balance above 2^64", balance.ok);
    if (balance.ok) {
        const auto decimal = crypto::HexQuantityToDecimal(balance.value);
        Check("the balance converts to decimal", !decimal.empty());
        std::cout << "        balance " << decimal << " wei\n";
        // More than 18.4 ETH, i.e. beyond what a 64-bit wei value can hold.
        Check("the balance exceeds the 64-bit range", decimal.size() > 20);
    }

    // The node must reject a malformed address rather than this client
    // accepting one silently.
    const auto bad = client.GetBalance("not-an-address");
    Check("a malformed address is rejected", !bad.ok);

    // An unknown method must surface the node's error.
    const auto unknown = client.Call("eth_noSuchMethodExists", "[]");
    Check("an unknown method is reported as an error", !unknown.ok);

    std::cout << (failures == 0 ? "live_rpc: all tests passed\n"
                                : "live_rpc: " + std::to_string(failures) + " test(s) failed\n");
    return failures == 0 ? 0 : 1;
}
