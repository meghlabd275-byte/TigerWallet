// Tests for TigerSmartChain support.
//
// These tests cover the pure logic - asset metadata, address handling, decimal
// formatting, pool parsing and price derivation - with real values. The pool
// tests build the exact ABI payload a Uniswap-V2 pool returns, so the parser is
// exercised against the real wire format rather than a simplified stand-in.
//
// One test reads tiger_smart_chain/chain/predeploy.go and fails if the
// addresses here disagree with the chain's own source, because the two are
// written in different languages and cannot share a constant.

#include "platform_chain.hpp"

#include <cstdint>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

namespace {

int g_failures = 0;

void Check(bool condition, const std::string& what) {
    if (!condition) {
        std::cerr << "FAIL: " << what << "\n";
        ++g_failures;
    }
}

void CheckEq(const std::string& actual, const std::string& expected,
             const std::string& what) {
    if (actual != expected) {
        std::cerr << "FAIL: " << what << "\n  expected: " << expected
                  << "\n  actual:   " << actual << "\n";
        ++g_failures;
    }
}

// Left-pads a hex string to a 32-byte ABI word.
std::string Word(const std::string& hex) {
    return std::string(64 - hex.size(), '0') + hex;
}

// Encodes a getReserves() return value: two uint112 words plus a uint32.
std::string ReservesPayload(const std::string& reserve0_hex,
                            const std::string& reserve1_hex) {
    return "0x" + Word(reserve0_hex) + Word(reserve1_hex) +
           Word("00000000");
}

// Encodes an address as an ABI word.
std::string AddressWord(const std::string& address) {
    const std::string hex = address.substr(2);
    return "0x" + std::string(24, '0') + hex;
}

std::string SetEnv(const char* name, const std::string& value) {
    const char* previous = std::getenv(name);
    const std::string saved = previous == nullptr ? std::string() : std::string(previous);
    setenv(name, value.c_str(), 1);
    return saved;
}

void UnsetEnv(const char* name) { unsetenv(name); }

void RestoreEnv(const char* name, const std::string& saved, bool had_value) {
    if (had_value) {
        setenv(name, saved.c_str(), 1);
    } else {
        unsetenv(name);
    }
}

void TestAssets() {
    using namespace tigerwallet::platform_chain;

    const auto assets = PlatformAssets();
    Check(assets.size() == 3, "three platform assets: TGR, WTGR, RUSD");

    const auto& tgr = assets[0];
    CheckEq(tgr.symbol, "TGR", "first asset is the native coin");
    Check(tgr.kind == AssetKind::kNative, "TGR is native");
    Check(!tgr.is_contract(), "TGR has no contract");
    Check(tgr.address.empty(), "TGR address is empty");
    Check(!tgr.pegged_usd, "TGR is not pegged");

    const auto wtgr = AssetBySymbol("wtgr");
    Check(wtgr.has_value(), "WTGR is found case-insensitively");
    CheckEq(wtgr->address, kWtgrAddress, "WTGR is the wrapped-native predeploy");
    Check(wtgr->is_contract(), "WTGR is a contract");
    Check(!wtgr->pegged_usd, "WTGR is not pegged");

    const auto rusd = AssetBySymbol("RUSD");
    Check(rusd.has_value(), "RUSD is registered");
    CheckEq(rusd->address, kRusdAddress, "RUSD is the stablecoin predeploy");
    Check(rusd->pegged_usd, "RUSD is pegged to USD");
    CheckEq(rusd->name, "Royal Tiger USD", "RUSD display name");

    Check(!AssetBySymbol("").has_value(), "empty symbol finds nothing");
    Check(!AssetBySymbol("ETH").has_value(), "a non-platform symbol finds nothing");

    // The native coin has no address, so address lookup must not find it.
    const auto by_address = AssetByAddress(kRusdAddress);
    Check(by_address.has_value(), "RUSD is found by address");
    CheckEq(by_address->symbol, "RUSD", "address lookup agrees with symbol lookup");
    Check(AssetByAddress("0x0000000000000000000000000000000000000000") == std::nullopt,
          "the zero address is not a platform asset");
    Check(AssetByAddress("not-an-address") == std::nullopt, "malformed address finds nothing");
}

void TestAddressValidation() {
    using namespace tigerwallet::platform_chain;

    Check(IsHexAddress("0x0000000000000000000000000000000000000000"), "zero address is valid");
    Check(IsHexAddress("0xABCDEF0123456789abcdef0123456789ABCDEF01"), "mixed case is valid");
    Check(!IsHexAddress(""), "empty is invalid");
    Check(!IsHexAddress("0x"), "bare prefix is invalid");
    Check(!IsHexAddress("0x1234"), "short address is invalid");
    Check(!IsHexAddress("0x" + std::string(39, '1')), "39 digits is invalid");
    Check(!IsHexAddress("0x" + std::string(41, '1')), "41 digits is invalid");
    Check(!IsHexAddress("0x" + std::string(40, 'z')), "non-hex is invalid");
    Check(!IsHexAddress(std::string(40, '1')), "missing prefix is invalid");

    const auto decoded = AddressFromAbiWord(AddressWord(kRusdAddress));
    Check(decoded.has_value(), "ABI word decodes");
    CheckEq(*decoded, kRusdAddress, "ABI word keeps the low 20 bytes");
    Check(!AddressFromAbiWord("0x1234").has_value(), "short ABI word is rejected");
    // A zero ABI word decodes to the zero address, which is a valid address
    // value; it simply never matches a platform asset.
    const auto zero_word = AddressFromAbiWord("0x" + std::string(64, '0'));
    Check(zero_word.has_value(), "a zero ABI word decodes to the zero address");
    CheckEq(*zero_word, "0x0000000000000000000000000000000000000000",
            "the zero word is the zero address");
}

void TestFormatUnits() {
    using namespace tigerwallet::platform_chain;

    CheckEq(FormatUnits("0", 18), "0", "zero");
    CheckEq(FormatUnits("1", 18), "0.000000000000000001", "one wei");
    CheckEq(FormatUnits("1000000000000000000", 18), "1", "one ether");
    CheckEq(FormatUnits("1500000000000000000", 18), "1.5", "one and a half");
    CheckEq(FormatUnits("1230000000000000000", 18), "1.23", "trailing zeros trimmed");
    CheckEq(FormatUnits("100", 0), "100", "zero decimals");
    CheckEq(FormatUnits("1234", 2), "12.34", "two decimals");
    CheckEq(FormatUnits("0001", 0), "1", "leading zeros trimmed");
    CheckEq(FormatUnits("", 18), "0", "empty becomes zero");
    CheckEq(FormatUnits("12a4", 2), "0", "non-numeric becomes zero");
    CheckEq(FormatUnits("5", 2), "0.05", "value below one unit");

    // A full-width uint256 value must survive: 78 decimal digits is more than
    // any machine integer here can hold.
    const std::string max_u256 =
        "115792089237316195423570985008687907853269984665640564039457584007913129639935";
    const std::string formatted = FormatUnits(max_u256, 18);
    Check(formatted.rfind("115792089237316195423570985008687907853269984665640564039457.584007913129639935", 0) == 0,
          "a uint256 balance keeps every digit");
}

void TestReserveParsing() {
    using namespace tigerwallet::platform_chain;

    // 1000 TGR and 2500 RUSD, both with 18 decimals.
    const std::string tgr = "3635c9adc5dea00000";        // 1000e18
    const std::string rusd = "878678326eac900000";       // 2500e18
    const auto payload = ReservesPayload(tgr, rusd);

    const auto parsed = ParseReservesHex(payload);
    Check(parsed.has_value(), "a well-formed reserves payload parses");
    CheckEq(parsed->reserve0, "1000000000000000000000", "reserve0 decoded exactly");
    CheckEq(parsed->reserve1, "2500000000000000000000", "reserve1 decoded exactly");

    // A uint112 must not use the top 8 bytes of its word.
    const std::string too_large = "0x" + std::string(24, 'f') + std::string(40, '0') +
                                  Word(rusd) + Word("00000000");
    Check(!ParseReservesHex(too_large).has_value(),
          "a reserve above uint112 is rejected, not truncated");

    Check(!ParseReservesHex("0x").has_value(), "empty payload is rejected");
    Check(!ParseReservesHex("0x" + std::string(64, '0')).has_value(),
          "a single word is too short");
    Check(!ParseReservesHex("0x" + std::string(192, 'z')).has_value(),
          "non-hex payload is rejected");
}

void TestPoolPrice() {
    using namespace tigerwallet::platform_chain;

    const std::string wtgr = kWtgrAddress;
    const std::string rusd = kRusdAddress;

    // 1000 WTGR against 2500 RUSD -> 2.50 USD per WTGR.
    const auto price = PoolPriceUsd("1000000000000000000000", "2500000000000000000000",
                                    wtgr, rusd);
    Check(price.has_value(), "a funded pool yields a price");
    Check(price.has_value() && *price > 2.49 && *price < 2.51, "price is 2.50 USD");

    // The token order must not change the answer: the parser reads the pool's
    // own ordering rather than assuming one.
    const auto reversed = PoolPriceUsd("2500000000000000000000",
                                       "1000000000000000000000", rusd, wtgr);
    Check(reversed.has_value(), "reversed token order still prices");
    Check(reversed.has_value() && *reversed > 2.49 && *reversed < 2.51,
          "token order does not invert the price");

    // An empty pool has no price. That is not a price of zero.
    Check(PoolPriceUsd("0", "2500000000000000000000", wtgr, rusd) == std::nullopt,
          "an empty asset reserve yields no price");
    Check(PoolPriceUsd("1000000000000000000000", "0", wtgr, rusd) == std::nullopt,
          "an empty RUSD reserve yields no price");

    // A pool with no RUSD leg says nothing about the dollar price.
    const std::string other = "0x0000000000000000000000000000000000001234";
    Check(PoolPriceUsd("1000", "2000", wtgr, other) == std::nullopt,
          "a pool without RUSD is refused");
    Check(PoolPriceUsd("1000", "2000", rusd, rusd) == std::nullopt,
          "a degenerate pool with two RUSD legs is refused");
    Check(PoolPriceUsd("1000", "2000", "nonsense", rusd) == std::nullopt,
          "a malformed token address is refused");
}

void TestEnvironmentConfiguration() {
    using namespace tigerwallet::platform_chain;

    const char* rpc = kRpcEnv;
    const char* pair = kPairEnv;
    const char* prev_rpc = std::getenv(rpc);
    const char* prev_pair = std::getenv(pair);
    const std::string saved_rpc = prev_rpc == nullptr ? "" : prev_rpc;
    const std::string saved_pair = prev_pair == nullptr ? "" : prev_pair;

    UnsetEnv(rpc);
    CheckEq(RpcUrlFromEnv(), "", "an unset TSC_RPC_URL yields no endpoint");
    const auto unset_entry = ChainConfigEntry();
    CheckEq(unset_entry.name, "TigerSmartChain", "the chain is described even when unconfigured");
    Check(unset_entry.rpc_url.empty(), "no endpoint is invented");
    Check(unset_entry.chain_id == 7710, "chain id is 7710");
    CheckEq(unset_entry.native_symbol, "TGR", "native symbol is TGR");
    Check(unset_entry.explorer_url.empty(), "no explorer is invented");

    SetEnv(rpc, "http://tsc.internal:8545");
    CheckEq(RpcUrlFromEnv(), "http://tsc.internal:8545", "the endpoint is read from the env");
    CheckEq(ChainConfigEntry().rpc_url, "http://tsc.internal:8545",
            "the registry entry carries the configured endpoint");

    UnsetEnv(pair);
    Check(!PairAddressFromEnv().has_value(), "an unset pool is not configured");
    SetEnv(pair, "not-an-address");
    Check(!PairAddressFromEnv().has_value(), "a malformed pool is rejected");
    SetEnv(pair, "0x00000000000000000000000000000000000000aa");
    Check(PairAddressFromEnv().has_value(), "a valid pool is accepted");

    RestoreEnv(rpc, saved_rpc, prev_rpc != nullptr);
    RestoreEnv(pair, saved_pair, prev_pair != nullptr);
}

// Reads tiger_smart_chain/chain/predeploy.go and checks that the addresses and
// metadata in this module match it. The two modules are in different languages
// and cannot share a constant, so the test is what keeps them in step.
void TestPredeploySourceAgrees() {
    using namespace tigerwallet::platform_chain;

    std::vector<std::string> candidates = {
        "chain/predeploy.go",
        "../chain/predeploy.go",
        "../../tiger_smart_chain/chain/predeploy.go",
        "../../../tiger_smart_chain/chain/predeploy.go",
    };

    std::ifstream file;
    std::string used;
    for (const auto& candidate : candidates) {
        file.open(candidate);
        if (file.good()) {
            used = candidate;
            break;
        }
        file.close();
        file.clear();
    }
    if (!file.is_open()) {
        // Running from an installed location with no source tree next to it.
        // Nothing to cross-check, and failing here would be wrong.
        std::cout << "SKIP: tiger_smart_chain/chain/predeploy.go not found next to the build\n";
        return;
    }

    std::stringstream buffer;
    buffer << file.rdbuf();
    const std::string source = buffer.str();
    file.close();

    // The chain's own source must contain the same two addresses. They are
    // matched case-insensitively because the Go source may use EIP-55 checksums.
    const auto contains_address = [&source](const std::string& address) {
        std::string haystack = source;
        std::string needle = address;
        for (char& c : haystack) c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
        for (char& c : needle) c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
        return haystack.find(needle) != std::string::npos;
    };

    Check(contains_address(kRusdAddress),
          "tiger_smart_chain/chain/predeploy.go declares the RUSD address this module uses");
    Check(contains_address(kWtgrAddress),
          "tiger_smart_chain/chain/predeploy.go declares the WTGR address this module uses");
    Check(source.find("RUSD") != std::string::npos, "the chain source names RUSD");
    Check(source.find("WTGR") != std::string::npos, "the chain source names WTGR");

    std::cout << "cross-checked against " << used << "\n";
}

// Exercises the whole read path against a real TigerSmartChain node. It runs
// only when TSC_RPC_URL is set. A node that answers but is the wrong network is
// a failure rather than a skip: passing while the chain is misconfigured would
// defeat the purpose of the check.
void TestLiveNode() {
    using namespace tigerwallet::platform_chain;

    const char* url = std::getenv(kRpcEnv);
    if (url == nullptr || *url == '\0') {
        std::cout << "SKIP: set " << kRpcEnv << " to run the live node check\n";
        return;
    }

    const tigerwallet::RpcClient client(url, 15);
    const auto chain_id = client.ChainId();
    Check(chain_id.ok, "the node answers eth_chainId");
    if (!chain_id.ok) return;
    std::cout << "live node chain id " << chain_id.value << "\n";

    Check(chain_id.value == kChainId || chain_id.value == kTestnetChainId,
          "the node reports a TigerSmartChain id (7710 mainnet or 7711 testnet)");

    // Reads an ABI-encoded string return value. Empty when the payload is not
    // one, which the caller treats as a failed read.
    const auto decode_abi_string = [](const std::string& hex_result) {
        std::string hex = hex_result;
        if (hex.size() >= 2 && hex[0] == '0' && (hex[1] == 'x' || hex[1] == 'X')) hex = hex.substr(2);
        if (hex.size() < 128) return std::string();
        size_t length = 0;
        try {
            length = static_cast<size_t>(std::stoul(hex.substr(64, 64), nullptr, 16));
        } catch (...) {
            return std::string();
        }
        std::string text;
        for (size_t i = 0; i < length && 128 + i * 2 + 2 <= hex.size(); ++i) {
            try {
                text.push_back(static_cast<char>(std::stoul(hex.substr(128 + i * 2, 2), nullptr, 16)));
            } catch (...) {
                return std::string();
            }
        }
        return text;
    };

    // The genesis predeploys must be present and must report the metadata the
    // wallet advertises. A name() that reverts means the contract is not there.
    for (const auto& asset : PlatformAssets()) {
        if (!asset.is_contract()) continue;

        const auto name = client.EthCall(asset.address, "0x06fdde03");
        Check(name.ok, "name() answers for " + asset.symbol);
        if (name.ok) {
            CheckEq(decode_abi_string(name.value), asset.name,
                    "name() on chain matches the wallet metadata for " + asset.symbol);
        }

        const auto decimals = client.EthCall(asset.address, "0x313ce567");
        Check(decimals.ok, "decimals() answers for " + asset.symbol);
        if (decimals.ok) {
            std::string hex = decimals.value;
            if (hex.size() >= 2 && hex[0] == '0' && (hex[1] == 'x' || hex[1] == 'X')) hex = hex.substr(2);
            if (hex.size() >= 64) {
                const auto dec = std::stoul(hex.substr(0, 64), nullptr, 16);
                Check(dec == asset.decimals, "decimals() on chain matches for " + asset.symbol);
            }
        }
    }

    // A balance read must work end to end, including for an address that holds
    // nothing.
    const std::string nobody = "0x00000000000000000000000000000000000000aa";
    const auto holdings = ReadHoldings(client, nobody, std::nullopt);
    Check(holdings.ok, "balances read from the live node");
    if (holdings.ok) {
        Check(holdings.holdings.size() == PlatformAssets().size(),
              "every platform asset is reported");
        for (const auto& holding : holdings.holdings) {
            Check(!holding.balance.empty(), "a balance was formatted for " + holding.asset.symbol);
        }
    }
}

}  // namespace

int main() {
    TestAssets();
    TestAddressValidation();
    TestFormatUnits();
    TestReserveParsing();
    TestPoolPrice();
    TestEnvironmentConfiguration();
    TestPredeploySourceAgrees();
    TestLiveNode();

    if (g_failures != 0) {
        std::cerr << g_failures << " check(s) failed\n";
        return 1;
    }
    std::cout << "all platform_chain checks passed\n";
    return 0;
}
