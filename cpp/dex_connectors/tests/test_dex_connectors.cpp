// Unit tests for the DEX connector ABI encoders and decimal arithmetic.
//
// The vectors are taken from the Ethereum ABI specification and from the
// keccak-256 test vectors, so a regression in the encoders is caught here
// rather than by a rejected transaction on chain.

#include <cstdio>
#include <string>
#include <vector>

#include "dex_connector.hpp"

using namespace tigerwallet::dex;

namespace {

int failures = 0;

void check(bool condition, const std::string& what) {
    if (!condition) {
        std::printf("FAIL %s\n", what.c_str());
        ++failures;
    }
}

void check_eq(const std::string& got, const std::string& want, const std::string& what) {
    if (got != want) {
        std::printf("FAIL %s\n  got:  %s\n  want: %s\n", what.c_str(), got.c_str(), want.c_str());
        ++failures;
    }
}

void test_selectors() {
    // Known selectors, from the canonical function signatures.
    check_eq(function_selector("balanceOf(address)"), "70a08231", "selector balanceOf");
    check_eq(function_selector("transfer(address,uint256)"), "a9059cbb", "selector transfer");
    check_eq(function_selector("approve(address,uint256)"), "095ea7b3", "selector approve");
    check_eq(function_selector("getReserves()"), "0902f1ac", "selector getReserves");
    check_eq(function_selector("decimals()"), "313ce567", "selector decimals");
    check_eq(function_selector("symbol()"), "95d89b41", "selector symbol");
    check_eq(function_selector("name()"), "06fdde03", "selector name");
    check_eq(function_selector("getAmountsOut(uint256,address[])"),
             "d06ca61f", "selector getAmountsOut");
}

void test_address_encoding() {
    check_eq(encode_address("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
             "000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7",
             "encode_address USDT");
    check_eq(encode_address("0x0000000000000000000000000000000000000000"),
             "0000000000000000000000000000000000000000000000000000000000000000",
             "encode_address zero");

    bool threw = false;
    try {
        encode_address("0x1234");
    } catch (const std::exception&) {
        threw = true;
    }
    check(threw, "encode_address rejects short input");
}

void test_uint_encoding() {
    std::string word;
    check(encode_uint256("0", word), "encode 0");
    check_eq(word, std::string(64, '0'), "encode 0 word");

    check(encode_uint256("1", word), "encode 1");
    check_eq(word, std::string(63, '0') + "1", "encode 1 word");

    check(encode_uint256("255", word), "encode 255");
    check_eq(word, std::string(62, '0') + "ff", "encode 255 word");

    // 2^256 - 1 is the largest encodable value.
    const std::string max256 =
        "115792089237316195423570985008687907853269984665640564039457584007913129639935";
    check(encode_uint256(max256, word), "encode 2^256-1");
    check_eq(word, std::string(64, 'f'), "encode 2^256-1 word");

    // 2^256 does not fit.
    const std::string too_big =
        "115792089237316195423570985008687907853269984665640564039457584007913129639936";
    check(!encode_uint256(too_big, word), "encode 2^256 rejected");

    check(!encode_uint256("-1", word), "encode rejects negative");
    check(!encode_uint256("12a", word), "encode rejects non-digit");

    // One ether in wei.
    check(encode_uint256("1000000000000000000", word), "encode 1e18");
    check_eq(word, "0000000000000000000000000000000000000000000000000de0b6b3a7640000",
             "encode 1e18 word");
}

void test_uint_decoding() {
    check_eq(decode_uint256(std::string(64, '0')), "0", "decode zero");
    check_eq(decode_uint256(std::string(63, '0') + "1"), "1", "decode one");
    check_eq(decode_uint256(std::string(62, '0') + "ff"), "255", "decode 255");
    check_eq(decode_uint256(std::string(64, 'f')),
             "115792089237316195423570985008687907853269984665640564039457584007913129639935",
             "decode 2^256-1");
    check_eq(decode_uint256("0x" + std::string(62, '0') + "ff"), "255", "decode with 0x prefix");
    check_eq(decode_uint256("abcd"), "", "decode rejects short word");
}

void test_round_trip() {
    const std::vector<std::string> values = {
        "0", "1", "255", "256", "1000000", "1000000000000000000",
        "123456789012345678901234567890",
        "115792089237316195423570985008687907853269984665640564039457584007913129639935",
    };
    for (const auto& v : values) {
        std::string word;
        check(encode_uint256(v, word), "round trip encode " + v);
        check_eq(decode_uint256(word), v, "round trip decode " + v);
    }
}

void test_decimal_arithmetic() {
    std::string out;

    check(decimal_add("1", "2", out), "add 1+2");
    check_eq(out, "3", "add 1+2 value");

    check(decimal_add("999999999999999999999", "1", out), "add carry");
    check_eq(out, "1000000000000000000000", "add carry value");

    check(decimal_add("0", "0", out), "add zero");
    check_eq(out, "0", "add zero value");

    check(decimal_sub("10", "3", out), "sub 10-3");
    check_eq(out, "7", "sub 10-3 value");

    check(decimal_sub("1000", "999", out), "sub borrow");
    check_eq(out, "1", "sub borrow value");

    check(!decimal_sub("3", "10", out), "sub underflow rejected");

    check(decimal_mul_div("1000", "30", "10000", out), "mul_div fee");
    check_eq(out, "3", "mul_div fee value");

    // 1e18 * 997 / 1000, the V2 fee adjustment.
    check(decimal_mul_div("1000000000000000000", "997", "1000", out), "mul_div 1e18");
    check_eq(out, "997000000000000000", "mul_div 1e18 value");

    check(!decimal_mul_div("1", "1", "0", out), "mul_div rejects zero divisor");

    check_eq(apply_fee("1000000", 30), "997000", "apply_fee 0.30%");
    check_eq(apply_fee("1000000", 0), "1000000", "apply_fee 0%");
    check_eq(apply_fee("1000000", 10000), "0", "apply_fee 100%");
}

void test_amounts_array_decoding() {
    // getAmountsOut returns a dynamic uint256[]:
    //   word 0: offset (0x20)
    //   word 1: length (2)
    //   word 2: amounts[0]
    //   word 3: amounts[1]
    const std::string data =
        std::string(64, '0').replace(62, 2, "20") +
        std::string(64, '0').replace(63, 1, "2") +
        std::string(64, '0').replace(62, 2, "ff") +      // 255 in
        std::string(64, '0').replace(62, 2, "9f");       // 159 out

    std::vector<std::string> amounts;
    check(decode_amounts_array(data, amounts), "decode amounts array");
    check_eq(std::to_string(amounts.size()), "2", "amounts length");
    check_eq(amounts[0], "255", "amounts[0]");
    check_eq(amounts[1], "159", "amounts[1]");

    check(!decode_amounts_array("0x1234", amounts), "decode rejects short data");
    check(!decode_amounts_array(std::string(128, '0'), amounts), "decode rejects zero length");
}

void test_string_decoding() {
    // symbol() returning the string "WETH".
    //   word 0: offset 0x20
    //   word 1: length 4
    //   word 2: "WETH" padded
    const std::string data =
        std::string(64, '0').replace(62, 2, "20") +
        std::string(64, '0').replace(63, 1, "4") +
        "5745544800000000000000000000000000000000000000000000000000000000";

    check_eq(decode_string(data), "WETH", "decode string WETH");

    // A longer name. The length word counts the real bytes, so "USD Coin" is
    // encoded with length 8 and the decoder returns exactly those 8 bytes.
    const std::string usdc_data =
        std::string(64, '0').replace(62, 2, "20") +
        std::string(64, '0').replace(63, 1, "8") +
        "55534420436f696e000000000000000000000000000000000000000000000000";
    check_eq(decode_string(usdc_data), "USD Coin", "decode string USD Coin");

    // Padding beyond the declared length must not be returned.
    const std::string padded_data =
        std::string(64, '0').replace(62, 2, "20") +
        std::string(64, '0').replace(63, 1, "4") +
        "5553444300000000000000000000000000000000000000000000000000000000";
    check_eq(decode_string(padded_data), "USDC", "decode string ignores padding");

    check_eq(decode_string("0x00"), "", "decode rejects short data");
}

void test_address_array_decoding() {
    // A two-element address[] return value.
    const std::string data =
        std::string(64, '0').replace(62, 2, "20") +
        std::string(64, '0').replace(63, 1, "2") +
        std::string(24, '0') + "dac17f958d2ee523a2206206994597c13d831ec7" +
        std::string(24, '0') + "a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48";

    const auto addresses = decode_address_array(data);
    check_eq(std::to_string(addresses.size()), "2", "address array length");
    check_eq(addresses[0], "0xdac17f958d2ee523a2206206994597c13d831ec7", "address array[0]");
    check_eq(addresses[1], "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "address array[1]");
}

void test_connector_requires_matching_chain() {
    // Constructing a connector must not fabricate an address or a connection.
    auto connector = make_v2_connector(DexProtocol::QUICKSWAP,
                                       "http://127.0.0.1:1",  // nothing listens here
                                       137,
                                       "0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff",
                                       "0x5757371414417b8C6CAad45bAeF941aBc60d2C9E");
    check(connector != nullptr, "connector constructed");
    check(connector->chain_id() == 137, "connector chain id");
    check_eq(std::string(connector->name()), "QuickSwap", "connector name");
    // An unreachable node must report failure, not a fake success.
    check(!connector->connect(), "connect fails without a node");
    check(!connector->is_connected(), "not connected after failure");

    std::string balance;
    check(!connector->get_balance("0xdAC17F958D2ee523a2206206994597C13D831ec7",
                                  "0x0000000000000000000000000000000000000000", balance),
          "get_balance fails without a node");

    Quote quote;
    check(!connector->get_quote("0xdAC17F958D2ee523a2206206994597C13D831ec7",
                                "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270",
                                "1000000", quote),
          "get_quote fails without a node");
}

}  // namespace

int main() {
    test_selectors();
    test_address_encoding();
    test_uint_encoding();
    test_uint_decoding();
    test_round_trip();
    test_decimal_arithmetic();
    test_amounts_array_decoding();
    test_string_decoding();
    test_address_array_decoding();
    test_connector_requires_matching_chain();

    if (failures == 0) {
        std::printf("all dex connector tests passed\n");
        return 0;
    }
    std::printf("%d failure(s)\n", failures);
    return 1;
}
