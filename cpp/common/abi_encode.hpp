// Self-built Solidity ABI encoding for contract calldata.
//
// The wallet builds calldata for token transfers, approvals and NFT moves. A
// mistake here moves real funds: an argument written into the wrong word, or
// silently dropped, sends value to the wrong place. Every entry point is
// therefore total and fallible — malformed input yields nullopt rather than a
// zero-filled argument, and the caller is expected to refuse to sign.
//
// Layout reference: the ABI encodes each argument as a 32-byte word. Values of
// static type are placed in the head in order. Values of dynamic type (bytes,
// arrays) put an offset in the head, measured in bytes from the start of the
// argument block, and their contents in the tail.

#ifndef TIGERWALLET_COMMON_ABI_ENCODE_HPP
#define TIGERWALLET_COMMON_ABI_ENCODE_HPP

#include "keccak256.hpp"
#include "uint256.hpp"

#include <array>
#include <cctype>
#include <cstddef>
#include <cstdint>
#include <optional>
#include <string>
#include <vector>

namespace tigerwallet {
namespace abi {

using Bytes = std::vector<uint8_t>;
using Word = std::array<uint8_t, 32>;
using Address = std::array<uint8_t, 20>;

// ---------------------------------------------------------------------------
// Primitives
// ---------------------------------------------------------------------------

inline bool is_hex_digit(char c) {
    return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F');
}

inline uint8_t hex_value(char c) {
    if (c >= '0' && c <= '9') return static_cast<uint8_t>(c - '0');
    if (c >= 'a' && c <= 'f') return static_cast<uint8_t>(c - 'a' + 10);
    return static_cast<uint8_t>(c - 'A' + 10);
}

// Strips an optional 0x prefix and decodes an even-length hex string.
inline std::optional<Bytes> hex_to_bytes(const std::string& input) {
    std::string hex = input;
    if (hex.size() >= 2 && hex[0] == '0' && (hex[1] == 'x' || hex[1] == 'X')) {
        hex = hex.substr(2);
    }
    if (hex.empty() || (hex.size() % 2) != 0) return std::nullopt;

    Bytes out;
    out.reserve(hex.size() / 2);
    for (size_t i = 0; i < hex.size(); i += 2) {
        if (!is_hex_digit(hex[i]) || !is_hex_digit(hex[i + 1])) return std::nullopt;
        out.push_back(static_cast<uint8_t>((hex_value(hex[i]) << 4) | hex_value(hex[i + 1])));
    }
    return out;
}

// Left-pads up to 32 bytes into one word. Anything longer cannot be
// represented and must be rejected by the caller.
inline std::optional<Word> left_pad(const Bytes& bytes) {
    if (bytes.size() > 32) return std::nullopt;
    Word word{};
    for (size_t i = 0; i < bytes.size(); ++i) {
        word[32 - bytes.size() + i] = bytes[i];
    }
    return word;
}

inline Word u64_word(uint64_t value) {
    Word word{};
    for (int i = 0; i < 8; ++i) {
        word[31 - i] = static_cast<uint8_t>((value >> (8 * i)) & 0xff);
    }
    return word;
}

// Big-endian 32-byte encoding of a 256-bit value.
inline Word u256_word(const math::U256& value) {
    Word word{};
    for (int limb = 0; limb < 4; ++limb) {
        const uint64_t bits = value.word[static_cast<size_t>(limb)];
        for (int byte = 0; byte < 8; ++byte) {
            word[31 - (limb * 8 + byte)] = static_cast<uint8_t>((bits >> (8 * byte)) & 0xff);
        }
    }
    return word;
}

inline Word address_word(const Address& address) {
    Word word{};
    for (size_t i = 0; i < address.size(); ++i) {
        word[12 + i] = address[i];
    }
    return word;
}

inline Bytes selector(const std::string& signature) {
    const uint32_t value = crypto::Keccak256::selector(signature);
    return Bytes{
        static_cast<uint8_t>((value >> 24) & 0xff),
        static_cast<uint8_t>((value >> 16) & 0xff),
        static_cast<uint8_t>((value >> 8) & 0xff),
        static_cast<uint8_t>(value & 0xff),
    };
}

inline void append_word(Bytes& out, const Word& word) {
    out.insert(out.end(), word.begin(), word.end());
}

// Appends a dynamic `bytes` value: a length word followed by the contents
// padded with zeroes to a whole number of words.
inline void append_dynamic_bytes(Bytes& out, const Bytes& data) {
    append_word(out, u64_word(data.size()));
    out.insert(out.end(), data.begin(), data.end());
    const size_t remainder = data.size() % 32;
    if (remainder != 0) {
        out.insert(out.end(), 32 - remainder, 0);
    }
}

// ---------------------------------------------------------------------------
// Parsing
// ---------------------------------------------------------------------------

// Accepts a 0x-prefixed 20-byte address. The prefix is required because a bare
// 40-character string is also a valid hex number, and accepting both would make
// the meaning of an argument depend on its length.
inline std::optional<Address> parse_address(const std::string& input) {
    if (input.size() != 42) return std::nullopt;
    if (!(input[0] == '0' && (input[1] == 'x' || input[1] == 'X'))) return std::nullopt;

    auto bytes = hex_to_bytes(input);
    if (!bytes || bytes->size() != 20) return std::nullopt;

    Address address{};
    for (size_t i = 0; i < address.size(); ++i) {
        address[i] = (*bytes)[i];
    }
    return address;
}

// Accepts either decimal digits or a 0x-prefixed hex value of up to 32 bytes.
// Returns the big-endian word so the result can be appended directly.
inline std::optional<Word> parse_uint256(const std::string& raw) {
    std::string value = raw;
    while (!value.empty() && std::isspace(static_cast<unsigned char>(value.front()))) {
        value.erase(value.begin());
    }
    while (!value.empty() && std::isspace(static_cast<unsigned char>(value.back()))) {
        value.pop_back();
    }
    if (value.empty()) return std::nullopt;

    if (value.size() >= 2 && value[0] == '0' && (value[1] == 'x' || value[1] == 'X')) {
        std::string digits = value.substr(2);
        if (digits.empty()) return std::nullopt;
        // An odd number of digits is padded on the left, so "0x1" is one byte.
        if (digits.size() % 2 != 0) digits.insert(digits.begin(), '0');
        auto bytes = hex_to_bytes(digits);
        if (!bytes) return std::nullopt;
        return left_pad(*bytes);
    }

    // Decimal: every character must be a digit, because U256::fromDecimal stops
    // at the first non-digit and would otherwise accept "100garbage" as 100.
    for (char c : value) {
        if (c < '0' || c > '9') return std::nullopt;
    }
    return u256_word(math::U256::fromDecimal(value));
}

// ---------------------------------------------------------------------------
// ERC-20
// ---------------------------------------------------------------------------

// transfer(address,uint256)
inline std::optional<Bytes> erc20_transfer(const std::string& to, const std::string& amount) {
    auto recipient = parse_address(to);
    auto value = parse_uint256(amount);
    if (!recipient || !value) return std::nullopt;

    Bytes out = selector("transfer(address,uint256)");
    append_word(out, address_word(*recipient));
    append_word(out, *value);
    return out;
}

// approve(address,uint256)
inline std::optional<Bytes> erc20_approve(const std::string& spender, const std::string& amount) {
    auto spender_address = parse_address(spender);
    auto value = parse_uint256(amount);
    if (!spender_address || !value) return std::nullopt;

    Bytes out = selector("approve(address,uint256)");
    append_word(out, address_word(*spender_address));
    append_word(out, *value);
    return out;
}

// transferFrom(address,address,uint256)
inline std::optional<Bytes> erc20_transfer_from(const std::string& from,
                                                const std::string& to,
                                                const std::string& amount) {
    auto sender = parse_address(from);
    auto recipient = parse_address(to);
    auto value = parse_uint256(amount);
    if (!sender || !recipient || !value) return std::nullopt;

    Bytes out = selector("transferFrom(address,address,uint256)");
    append_word(out, address_word(*sender));
    append_word(out, address_word(*recipient));
    append_word(out, *value);
    return out;
}

// ---------------------------------------------------------------------------
// ERC-721
// ---------------------------------------------------------------------------

// safeTransferFrom(address,address,uint256)
inline std::optional<Bytes> erc721_safe_transfer_from(const std::string& from,
                                                      const std::string& to,
                                                      const std::string& token_id) {
    auto sender = parse_address(from);
    auto recipient = parse_address(to);
    auto id = parse_uint256(token_id);
    if (!sender || !recipient || !id) return std::nullopt;

    Bytes out = selector("safeTransferFrom(address,address,uint256)");
    append_word(out, address_word(*sender));
    append_word(out, address_word(*recipient));
    append_word(out, *id);
    return out;
}

// safeTransferFrom(address,address,uint256,bytes)
inline std::optional<Bytes> erc721_safe_transfer_from_with_data(const std::string& from,
                                                                const std::string& to,
                                                                const std::string& token_id,
                                                                const Bytes& data) {
    auto sender = parse_address(from);
    auto recipient = parse_address(to);
    auto id = parse_uint256(token_id);
    if (!sender || !recipient || !id) return std::nullopt;

    Bytes out = selector("safeTransferFrom(address,address,uint256,bytes)");
    append_word(out, address_word(*sender));
    append_word(out, address_word(*recipient));
    append_word(out, *id);
    // Three static words precede the offset, so the tail starts at word four.
    append_word(out, u64_word(4 * 32));
    append_dynamic_bytes(out, data);
    return out;
}

// setApprovalForAll(address,bool)
inline std::optional<Bytes> erc721_set_approval_for_all(const std::string& operator_address,
                                                        bool approved) {
    auto op = parse_address(operator_address);
    if (!op) return std::nullopt;

    Bytes out = selector("setApprovalForAll(address,bool)");
    append_word(out, address_word(*op));
    append_word(out, u64_word(approved ? 1 : 0));
    return out;
}

// ---------------------------------------------------------------------------
// ERC-1155
// ---------------------------------------------------------------------------

// safeTransferFrom(address,address,uint256,uint256,bytes)
inline std::optional<Bytes> erc1155_safe_transfer_from(const std::string& from,
                                                       const std::string& to,
                                                       const std::string& token_id,
                                                       const std::string& amount,
                                                       const Bytes& data) {
    auto sender = parse_address(from);
    auto recipient = parse_address(to);
    auto id = parse_uint256(token_id);
    auto value = parse_uint256(amount);
    if (!sender || !recipient || !id || !value) return std::nullopt;

    Bytes out = selector("safeTransferFrom(address,address,uint256,uint256,bytes)");
    append_word(out, address_word(*sender));
    append_word(out, address_word(*recipient));
    append_word(out, *id);
    append_word(out, *value);
    // Four static words precede the offset.
    append_word(out, u64_word(4 * 32));
    append_dynamic_bytes(out, data);
    return out;
}

// safeBatchTransferFrom(address,address,uint256[],uint256[],bytes)
inline std::optional<Bytes> erc1155_safe_batch_transfer_from(
    const std::string& from,
    const std::string& to,
    const std::vector<std::string>& token_ids,
    const std::vector<std::string>& amounts,
    const Bytes& data) {
    auto sender = parse_address(from);
    auto recipient = parse_address(to);
    if (!sender || !recipient) return std::nullopt;
    if (token_ids.empty() || token_ids.size() != amounts.size()) return std::nullopt;

    std::vector<Word> ids;
    ids.reserve(token_ids.size());
    for (const auto& id : token_ids) {
        auto parsed = parse_uint256(id);
        if (!parsed) return std::nullopt;
        ids.push_back(*parsed);
    }

    std::vector<Word> values;
    values.reserve(amounts.size());
    for (const auto& amount : amounts) {
        auto parsed = parse_uint256(amount);
        if (!parsed) return std::nullopt;
        values.push_back(*parsed);
    }

    // Head: from, to, ids offset, amounts offset, data offset (five words).
    const size_t head_size = 5 * 32;
    const size_t ids_size = 32 + ids.size() * 32;
    const size_t amounts_size = 32 + values.size() * 32;

    Bytes out = selector("safeBatchTransferFrom(address,address,uint256[],uint256[],bytes)");
    append_word(out, address_word(*sender));
    append_word(out, address_word(*recipient));
    append_word(out, u64_word(head_size));
    append_word(out, u64_word(head_size + ids_size));
    append_word(out, u64_word(head_size + ids_size + amounts_size));

    append_word(out, u64_word(ids.size()));
    for (const auto& id : ids) append_word(out, id);

    append_word(out, u64_word(values.size()));
    for (const auto& value : values) append_word(out, value);

    append_dynamic_bytes(out, data);
    return out;
}

}  // namespace abi
}  // namespace tigerwallet

#endif  // TIGERWALLET_COMMON_ABI_ENCODE_HPP
