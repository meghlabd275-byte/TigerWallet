// Tests for the Solidity ABI calldata encoder.
//
// The expected values are taken from the ABI specification and from the
// calldata real contracts dispatch on, not from this implementation. A
// regression here would send funds with the wrong arguments, so the encodings
// are pinned byte for byte.

#include "abi_encode.hpp"

#include <cstdio>
#include <string>
#include <vector>

using namespace tigerwallet::abi;

static int failures = 0;

static void check(bool condition, const char* label) {
    if (!condition) {
        std::printf("FAIL %s\n", label);
        ++failures;
    } else {
        std::printf("PASS %s\n", label);
    }
}

static std::string to_hex(const Bytes& data) {
    static const char* kHex = "0123456789abcdef";
    std::string out;
    out.reserve(data.size() * 2);
    for (uint8_t byte : data) {
        out.push_back(kHex[byte >> 4]);
        out.push_back(kHex[byte & 0x0f]);
    }
    return out;
}

static void check_hex(const Bytes& data, const std::string& expected, const char* label) {
    const std::string actual = to_hex(data);
    if (actual != expected) {
        std::printf("FAIL %s\n  expected %s\n  actual   %s\n", label, expected.c_str(), actual.c_str());
        ++failures;
    } else {
        std::printf("PASS %s\n", label);
    }
}

int main() {
    // --- Selectors ---------------------------------------------------------
    check_hex(selector("transfer(address,uint256)"), "a9059cbb", "selector transfer");
    check_hex(selector("approve(address,uint256)"), "095ea7b3", "selector approve");
    check_hex(selector("transferFrom(address,address,uint256)"), "23b872dd", "selector transferFrom");
    check_hex(selector("balanceOf(address)"), "70a08231", "selector balanceOf");
    check_hex(selector("totalSupply()"), "18160ddd", "selector totalSupply");
    check_hex(selector("decimals()"), "313ce567", "selector decimals");
    check_hex(selector("setApprovalForAll(address,bool)"), "a22cb465", "selector setApprovalForAll");
    check_hex(selector("safeTransferFrom(address,address,uint256)"), "42842e0e",
              "selector erc721 safeTransferFrom");
    check_hex(selector("safeTransferFrom(address,address,uint256,bytes)"), "b88d4fde",
              "selector erc721 safeTransferFrom with data");
    check_hex(selector("safeTransferFrom(address,address,uint256,uint256,bytes)"), "f242432a",
              "selector erc1155 safeTransferFrom");
    check_hex(selector("safeBatchTransferFrom(address,address,uint256[],uint256[],bytes)"), "2eb2c2d6",
              "selector erc1155 safeBatchTransferFrom");

    // Two signatures sharing a prefix must not collide.
    check(selector("transfer(address,uint256)") != selector("transfer(address,uint8)"),
          "selectors differ for same-prefix signatures");

    // --- ERC-20 transfer ---------------------------------------------------
    {
        auto data = erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "1000");
        check(data.has_value(), "erc20 transfer encodes");
        if (data) {
            // 4-byte selector, then the address right-aligned in one word, then
            // the amount 1000 (0x3e8) in the next word.
            check_hex(*data,
                      "a9059cbb"
                      "000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"
                      "00000000000000000000000000000000000000000000000000000000000003e8",
                      "erc20 transfer calldata");
            check(data->size() == 4 + 32 + 32, "erc20 transfer length");
        }
    }

    // Decimal and hex forms of the same amount must produce the same calldata.
    {
        auto decimal = erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "1000");
        auto hex = erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "0x3e8");
        check(decimal && hex && *decimal == *hex, "decimal and hex amounts agree");
    }

    // A full-width amount must occupy exactly 32 bytes.
    {
        auto data = erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
                                   "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff");
        check(data && data->size() == 68, "max uint256 encodes");
        if (data) {
            // Byte 36 starts the amount word; byte 35 is the address word's
            // last byte and must still hold the address.
            const bool all_ones = data->at(36) == 0xff && data->at(67) == 0xff;
            check(all_ones, "max uint256 bytes");
            check(data->at(35) == 0x45, "address word not overwritten");
        }
    }

    // --- ERC-20 approve ----------------------------------------------------
    {
        auto data = erc20_approve("0x1111111254EEB25477B68fb85Ed929f73A960582", "0");
        check(data.has_value(), "erc20 approve encodes");
        if (data) {
            check_hex(*data,
                      "095ea7b3"
                      "0000000000000000000000001111111254eeb25477b68fb85ed929f73a960582"
                      "0000000000000000000000000000000000000000000000000000000000000000",
                      "erc20 approve calldata");
        }
    }

    // --- ERC-20 transferFrom ----------------------------------------------
    {
        auto data = erc20_transfer_from("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
                                        "0x1111111254EEB25477B68fb85Ed929f73A960582", "5");
        check(data.has_value() && data->size() == 4 + 32 * 3, "erc20 transferFrom length");
        if (data) {
            check_hex(*data,
                      "23b872dd"
                      "000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"
                      "0000000000000000000000001111111254eeb25477b68fb85ed929f73a960582"
                      "0000000000000000000000000000000000000000000000000000000000000005",
                      "erc20 transferFrom calldata");
        }
    }

    // --- ERC-721 -----------------------------------------------------------
    {
        auto data = erc721_safe_transfer_from("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
                                              "0x1111111254EEB25477B68fb85Ed929f73A960582", "1");
        check(data.has_value() && data->size() == 4 + 32 * 3, "erc721 safeTransferFrom length");
        if (data) {
            check(data->at(3) == 0x0e, "erc721 selector byte");
            check(data->at(99) == 1, "erc721 token id in third word");
        }
    }

    {
        auto data = erc721_set_approval_for_all("0x1111111254EEB25477B68fb85Ed929f73A960582", true);
        check(data.has_value() && data->size() == 4 + 32 * 2, "erc721 setApprovalForAll length");
        if (data) {
            check_hex(*data,
                      "a22cb465"
                      "0000000000000000000000001111111254eeb25477b68fb85ed929f73a960582"
                      "0000000000000000000000000000000000000000000000000000000000000001",
                      "erc721 setApprovalForAll calldata");
        }
    }

    {
        auto data = erc721_set_approval_for_all("0x1111111254EEB25477B68fb85Ed929f73A960582", false);
        check(data && data->at(67) == 0, "erc721 setApprovalForAll false");
    }

    // --- ERC-721 with data (dynamic argument) ------------------------------
    {
        auto data = erc721_safe_transfer_from_with_data("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
                                                        "0x1111111254EEB25477B68fb85Ed929f73A960582",
                                                        "1", Bytes{0xde, 0xad});
        check(data.has_value(), "erc721 safeTransferFrom with data encodes");
        if (data) {
            // 4 selector + 4 head words + length word + one padded data word.
            check(data->size() == 4 + 32 * 4 + 32 + 32, "erc721 with data length");
            check(data->at(3) == 0xde, "erc721 with data selector byte");
            // The data offset counts from the start of the argument block.
            check_hex(Bytes(data->begin() + 4 + 96, data->begin() + 4 + 128),
                      "0000000000000000000000000000000000000000000000000000000000000080",
                      "erc721 with data offset");
            // Length 2, then the bytes padded to 32.
            check_hex(Bytes(data->begin() + 4 + 128, data->begin() + 4 + 160),
                      "0000000000000000000000000000000000000000000000000000000000000002",
                      "erc721 with data length word");
            check(data->at(4 + 160) == 0xde && data->at(4 + 161) == 0xad,
                  "erc721 with data payload");
        }
    }

    // --- ERC-1155 ----------------------------------------------------------
    {
        auto data = erc1155_safe_transfer_from("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
                                               "0x1111111254EEB25477B68fb85Ed929f73A960582",
                                               "7", "10", Bytes{});
        check(data.has_value(), "erc1155 safeTransferFrom encodes");
        if (data) {
            // 4 selector + 4 static words + offset word + length word (empty data).
            check(data->size() == 4 + 32 * 4 + 32 + 32, "erc1155 safeTransferFrom length");
            check(data->at(3) == 0x2a, "erc1155 selector byte");
            check_hex(Bytes(data->begin() + 4 + 64, data->begin() + 4 + 96),
                      "0000000000000000000000000000000000000000000000000000000000000007",
                      "erc1155 token id");
            check_hex(Bytes(data->begin() + 4 + 96, data->begin() + 4 + 128),
                      "000000000000000000000000000000000000000000000000000000000000000a",
                      "erc1155 amount");
        }
    }

    {
        auto data = erc1155_safe_batch_transfer_from(
            "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
            "0x1111111254EEB25477B68fb85Ed929f73A960582",
            {"1", "2"}, {"10", "20"}, Bytes{});
        check(data.has_value(), "erc1155 safeBatchTransferFrom encodes");
        if (data) {
            // 4 selector + 5 head words, then ids (length + 2), amounts
            // (length + 2), then the empty data (length + no payload).
            check(data->size() == 4 + 32 * 5 + 32 * 3 + 32 * 3 + 32, "erc1155 batch length");
            // Offsets: ids at 160, amounts at 160+96=256, data at 256+96=352.
            check_hex(Bytes(data->begin() + 4 + 64, data->begin() + 4 + 96),
                      "00000000000000000000000000000000000000000000000000000000000000a0",
                      "erc1155 batch ids offset");
            check_hex(Bytes(data->begin() + 4 + 96, data->begin() + 4 + 128),
                      "0000000000000000000000000000000000000000000000000000000000000100",
                      "erc1155 batch amounts offset");
            check_hex(Bytes(data->begin() + 4 + 128, data->begin() + 4 + 160),
                      "0000000000000000000000000000000000000000000000000000000000000160",
                      "erc1155 batch data offset");
            // Array length words.
            check(data->at(4 + 160 + 31) == 2, "erc1155 batch ids length");
            check(data->at(4 + 256 + 31) == 2, "erc1155 batch amounts length");
        }
    }

    // --- Fail-closed behaviour --------------------------------------------
    check(!erc20_transfer("0xnothex", "1").has_value(), "reject non-hex address");
    check(!erc20_transfer("d8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "1").has_value(),
          "reject address without 0x prefix");
    check(!erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA9604", "1").has_value(),
          "reject short address");
    check(!erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045ff", "1").has_value(),
          "reject long address");
    check(!erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "").has_value(),
          "reject empty amount");
    check(!erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "12a").has_value(),
          "reject malformed decimal amount");
    check(!erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "0xzz").has_value(),
          "reject malformed hex amount");
    check(!erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "0x").has_value(),
          "reject empty hex amount");
    // A value wider than one word cannot be represented.
    check(!erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
                          "0x" + std::string(66, 'f')).has_value(),
          "reject over-wide amount");
    check(!erc1155_safe_batch_transfer_from(
              "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
              "0x1111111254EEB25477B68fb85Ed929f73A960582", {"1"}, {"1", "2"}, Bytes{})
              .has_value(),
          "reject mismatched batch lengths");
    check(!erc1155_safe_batch_transfer_from(
              "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
              "0x1111111254EEB25477B68fb85Ed929f73A960582", {}, {}, Bytes{})
              .has_value(),
          "reject empty batch");

    // Whitespace around a decimal amount is tolerated.
    check(erc20_transfer("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", " 42 ").has_value(),
          "tolerate surrounding whitespace");

    if (failures == 0) {
        std::printf("\nAll ABI encoder checks passed.\n");
        return 0;
    }
    std::printf("\n%d ABI encoder checks FAILED.\n", failures);
    return 1;
}
