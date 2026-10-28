/**
 * TigerWallet address validation.
 *
 * Every check here decodes the address and verifies its checksum. A length test
 * is not validation: "0x" followed by 40 characters of anything passed the
 * previous checks, as did any 32-to-44 character string for Solana and any
 * 26-to-62 character string for Bitcoin.
 *
 * No third-party wallet or address library is used. Base58, Base58Check,
 * Bech32 and EIP-55 are implemented here, over the shared Keccak-256 and
 * OpenSSL's SHA-256.
 */

#ifndef TIGER_WALLET_ADDRESS_HPP
#define TIGER_WALLET_ADDRESS_HPP

#include <openssl/sha.h>

#include <array>
#include <cctype>
#include <cstdint>
#include <optional>
#include <string>
#include <vector>

#include "keccak256.hpp"

namespace tigerwallet {
namespace address {

// ==================== Hex ====================

inline bool is_hex(const std::string& input) {
    if (input.empty()) return false;
    for (char c : input) {
        if (!std::isxdigit(static_cast<unsigned char>(c))) return false;
    }
    return true;
}

// ==================== Base58 ====================

// The Bitcoin alphabet. Base58 omits 0, O, I and l, which are the characters
// people misread.
inline const char* base58_alphabet() {
    return "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
}

// Decodes Base58. Returns nullopt on an invalid character. Leading '1'
// characters are leading zero bytes, which the carry loop does not produce on
// its own.
inline std::optional<std::vector<uint8_t>> base58_decode(const std::string& input) {
    if (input.empty()) return std::nullopt;

    const std::string alphabet = base58_alphabet();
    std::array<int8_t, 256> map{};
    map.fill(-1);
    for (size_t i = 0; i < alphabet.size(); ++i) {
        map[static_cast<unsigned char>(alphabet[i])] = static_cast<int8_t>(i);
    }

    size_t zeros = 0;
    while (zeros < input.size() && input[zeros] == '1') ++zeros;

    std::vector<uint8_t> result;
    result.reserve(input.size());

    for (size_t i = zeros; i < input.size(); ++i) {
        const int8_t value = map[static_cast<unsigned char>(input[i])];
        if (value < 0) return std::nullopt;

        int carry = value;
        for (size_t j = 0; j < result.size(); ++j) {
            carry += 58 * static_cast<int>(result[j]);
            result[j] = static_cast<uint8_t>(carry % 256);
            carry /= 256;
        }
        while (carry > 0) {
            result.push_back(static_cast<uint8_t>(carry % 256));
            carry /= 256;
        }
    }

    // The digits were produced least-significant first.
    std::vector<uint8_t> decoded;
    decoded.reserve(zeros + result.size());
    decoded.insert(decoded.end(), zeros, 0);
    decoded.insert(decoded.end(), result.rbegin(), result.rend());
    return decoded;
}

inline std::string base58_encode(const std::vector<uint8_t>& input) {
    const std::string alphabet = base58_alphabet();

    size_t zeros = 0;
    while (zeros < input.size() && input[zeros] == 0) ++zeros;

    std::vector<uint8_t> digits;
    for (size_t i = zeros; i < input.size(); ++i) {
        int carry = input[i];
        for (size_t j = 0; j < digits.size(); ++j) {
            carry += 256 * static_cast<int>(digits[j]);
            digits[j] = static_cast<uint8_t>(carry % 58);
            carry /= 58;
        }
        while (carry > 0) {
            digits.push_back(static_cast<uint8_t>(carry % 58));
            carry /= 58;
        }
    }

    std::string result(zeros, '1');
    for (auto it = digits.rbegin(); it != digits.rend(); ++it) {
        result.push_back(alphabet[*it]);
    }
    return result;
}

// SHA-256, used for Base58Check and for Bitcoin's version bytes.
inline std::array<uint8_t, 32> sha256(const uint8_t* data, size_t size) {
    std::array<uint8_t, 32> out{};
    SHA256(data, size, out.data());
    return out;
}

inline std::array<uint8_t, 32> sha256(const std::vector<uint8_t>& data) {
    return sha256(data.data(), data.size());
}

// ==================== Base58Check ====================

// A Base58Check payload is the version byte(s) and the data, followed by the
// first four bytes of SHA256(SHA256(payload)).
inline std::optional<std::vector<uint8_t>> base58check_decode(const std::string& input) {
    const auto decoded = base58_decode(input);
    if (!decoded || decoded->size() < 5) return std::nullopt;

    const size_t payload_size = decoded->size() - 4;
    const std::vector<uint8_t> payload(decoded->begin(), decoded->begin() + payload_size);
    const std::array<uint8_t, 32> first = sha256(payload);
    const std::array<uint8_t, 32> second = sha256(first.data(), first.size());

    for (size_t i = 0; i < 4; ++i) {
        if ((*decoded)[payload_size + i] != second[i]) return std::nullopt;
    }
    return payload;
}

inline std::string base58check_encode(const std::vector<uint8_t>& payload) {
    const std::array<uint8_t, 32> first = sha256(payload);
    const std::array<uint8_t, 32> second = sha256(first.data(), first.size());

    std::vector<uint8_t> full = payload;
    full.insert(full.end(), second.begin(), second.begin() + 4);
    return base58_encode(full);
}

// ==================== Bech32 ====================

namespace bech32_detail {

inline const char* charset() {
    return "qpzry9x8gf2tvdw0s3jn54khce6mua7l";
}

inline uint32_t polymod(const std::vector<uint8_t>& values) {
    static const uint32_t kGenerators[5] = {0x3b6a57b2, 0x26508e6d, 0x1ea119fa,
                                            0x3d4233dd, 0x2a1462b3};
    uint32_t checksum = 1;
    for (uint8_t value : values) {
        const uint8_t top = static_cast<uint8_t>(checksum >> 25);
        checksum = ((checksum & 0x1ffffff) << 5) ^ value;
        for (int i = 0; i < 5; ++i) {
            if ((top >> i) & 1) checksum ^= kGenerators[i];
        }
    }
    return checksum;
}

inline std::vector<uint8_t> expand_hrp(const std::string& hrp) {
    std::vector<uint8_t> out;
    out.reserve(hrp.size() * 2 + 1);
    for (char c : hrp) out.push_back(static_cast<uint8_t>(c) >> 5);
    out.push_back(0);
    for (char c : hrp) out.push_back(static_cast<uint8_t>(c) & 31);
    return out;
}

// Converts 8-bit groups to 5-bit groups for encoding.
inline std::vector<uint8_t> to_five_bits(const std::vector<uint8_t>& data) {
    std::vector<uint8_t> out;
    uint32_t accumulator = 0;
    int bits = 0;
    for (uint8_t byte : data) {
        accumulator = (accumulator << 8) | byte;
        bits += 8;
        while (bits >= 5) {
            bits -= 5;
            out.push_back(static_cast<uint8_t>((accumulator >> bits) & 31));
        }
    }
    if (bits > 0) {
        out.push_back(static_cast<uint8_t>((accumulator << (5 - bits)) & 31));
    }
    return out;
}

// Converts 5-bit groups back to 8-bit groups. Fails on a group that is out of
// range or on leftover bits that are not zero, which would mean the data was
// not a whole number of bytes.
inline std::optional<std::vector<uint8_t>> from_five_bits(const std::vector<uint8_t>& data,
                                                          size_t from, size_t to) {
    std::vector<uint8_t> out;
    uint32_t accumulator = 0;
    int bits = 0;
    for (size_t i = from; i < to; ++i) {
        if (data[i] > 31) return std::nullopt;
        accumulator = (accumulator << 5) | data[i];
        bits += 5;
        if (bits >= 8) {
            bits -= 8;
            out.push_back(static_cast<uint8_t>((accumulator >> bits) & 0xff));
        }
    }
    // Any remaining bits must be zero padding.
    if (bits >= 5) return std::nullopt;
    if ((accumulator & ((1u << bits) - 1)) != 0) return std::nullopt;
    return out;
}

}  // namespace bech32_detail

struct Bech32Decoded {
    std::string hrp;
    std::vector<uint8_t> data;  // The 8-bit payload, witness version removed.
    uint8_t witness_version = 0;
    bool has_witness_version = false;
};

// Decodes a Bech32 or Bech32m string and verifies its checksum.
inline std::optional<Bech32Decoded> bech32_decode(const std::string& input) {
    if (input.size() < 8 || input.size() > 90) return std::nullopt;

    // Mixed case is invalid.
    bool has_lower = false;
    bool has_upper = false;
    for (char c : input) {
        if (c >= 'a' && c <= 'z') has_lower = true;
        if (c >= 'A' && c <= 'Z') has_upper = true;
    }
    if (has_lower && has_upper) return std::nullopt;

    std::string lowered = input;
    for (char& c : lowered) c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));

    const size_t separator = lowered.rfind('1');
    if (separator == std::string::npos || separator == 0 ||
        separator + 7 > lowered.size()) {
        return std::nullopt;
    }

    const std::string hrp = lowered.substr(0, separator);
    const std::string charset = bech32_detail::charset();
    std::array<int8_t, 256> map{};
    map.fill(-1);
    for (size_t i = 0; i < charset.size(); ++i) {
        map[static_cast<unsigned char>(charset[i])] = static_cast<int8_t>(i);
    }

    std::vector<uint8_t> values;
    values.reserve(lowered.size() - separator - 1);
    for (size_t i = separator + 1; i < lowered.size(); ++i) {
        const int8_t value = map[static_cast<unsigned char>(lowered[i])];
        if (value < 0) return std::nullopt;
        values.push_back(static_cast<uint8_t>(value));
    }

    std::vector<uint8_t> checksum_input = bech32_detail::expand_hrp(hrp);
    checksum_input.insert(checksum_input.end(), values.begin(), values.end());
    const uint32_t checksum = bech32_detail::polymod(checksum_input);

    const bool is_bech32 = (checksum == 1);
    const bool is_bech32m = (checksum == 0x2bc830a3);
    if (!is_bech32 && !is_bech32m) return std::nullopt;

    // Strip the six checksum characters.
    const size_t data_end = values.size() - 6;

    Bech32Decoded decoded;
    decoded.hrp = hrp;

    // Only the SegWit prefixes carry a witness version. A Cosmos address also
    // begins with a 5-bit zero, so without this restriction a cosmos1... string
    // would be read as a witness program and its payload would be misread.
    const bool segwit_hrp = (hrp == "bc" || hrp == "tb" || hrp == "bcrt" ||
                             hrp == "ltc" || hrp == "tltc");

    // A witness program starts with a version value of 0 to 16. In that form
    // Bech32 is used for version 0 and Bech32m for versions 1 to 16, so a
    // mismatch means the address is not valid.
    if (segwit_hrp && data_end > 0 && values[0] <= 16) {
        decoded.has_witness_version = true;
        decoded.witness_version = values[0];
        if (decoded.witness_version == 0 && !is_bech32) return std::nullopt;
        if (decoded.witness_version != 0 && !is_bech32m) return std::nullopt;

        auto payload = bech32_detail::from_five_bits(values, 1, data_end);
        if (!payload) return std::nullopt;
        decoded.data = *payload;
        return decoded;
    }

    auto payload = bech32_detail::from_five_bits(values, 0, data_end);
    if (!payload) return std::nullopt;
    decoded.data = *payload;
    return decoded;
}

// ==================== EIP-55 ====================

// The EIP-55 checksum: the address is lower-cased, hashed with Keccak-256, and
// each letter is upper-cased when the corresponding hash nibble is 8 or above.
inline std::string eip55_checksum(const std::string& address_hex) {
    std::string lower;
    for (char c : address_hex) {
        lower.push_back(static_cast<char>(std::tolower(static_cast<unsigned char>(c))));
    }
    const auto digest = crypto::Keccak256::hash(lower);

    static const char* kHex = "0123456789abcdef";
    std::string out = "0x";
    for (size_t i = 0; i < lower.size(); ++i) {
        const char c = lower[i];
        if (c >= '0' && c <= '9') {
            out.push_back(c);
            continue;
        }
        const uint8_t nibble = (i % 2 == 0) ? (digest[i / 2] >> 4) : (digest[i / 2] & 0x0f);
        if (nibble >= 8) {
            out.push_back(static_cast<char>(std::toupper(static_cast<unsigned char>(c))));
        } else {
            out.push_back(kHex[c - 'a' + 10]);
        }
    }
    return out;
}

// ==================== Validation ====================

// An EVM address is 20 bytes of hex. A mixed-case address carries an EIP-55
// checksum and is rejected when it does not match; an all-lower or all-upper
// address carries no checksum, which is how every EVM client treats it.
inline bool is_valid_evm_address(const std::string& address) {
    if (address.size() != 42) return false;
    if (address[0] != '0' || (address[1] != 'x' && address[1] != 'X')) return false;

    const std::string body = address.substr(2);
    if (!is_hex(body)) return false;

    bool has_upper = false;
    bool has_lower = false;
    for (char c : body) {
        if (c >= 'A' && c <= 'F') has_upper = true;
        if (c >= 'a' && c <= 'f') has_lower = true;
    }
    if (has_upper && has_lower) {
        return eip55_checksum(body) == address;
    }
    return true;
}

// Bitcoin Base58Check P2PKH and P2SH, plus Bech32 and Bech32m SegWit.
inline bool is_valid_bitcoin_address(const std::string& address) {
    // SegWit: bc1q (v0) or bc1p (v1), with a valid Bech32 or Bech32m checksum.
    if (address.size() > 3 &&
        (address.compare(0, 3, "bc1") == 0 || address.compare(0, 3, "BC1") == 0)) {
        const auto decoded = bech32_decode(address);
        if (!decoded || decoded->hrp != "bc") return false;
        if (!decoded->has_witness_version) return false;
        // A witness program is 2 to 40 bytes.
        if (decoded->data.size() < 2 || decoded->data.size() > 40) return false;
        // Version 0 is 20 or 32 bytes.
        if (decoded->witness_version == 0 &&
            decoded->data.size() != 20 && decoded->data.size() != 32) {
            return false;
        }
        return true;
    }

    // Legacy P2PKH (version 0x00) and P2SH (version 0x05).
    const auto payload = base58check_decode(address);
    if (!payload || payload->size() != 21) return false;
    return (*payload)[0] == 0x00 || (*payload)[0] == 0x05;
}

// Litecoin: Base58Check version 0x30 (L...) and 0x32 (M...), and the ltc1
// Bech32 prefix.
inline bool is_valid_litecoin_address(const std::string& address) {
    if (address.size() > 4 && address.compare(0, 4, "ltc1") == 0) {
        const auto decoded = bech32_decode(address);
        return decoded.has_value() && decoded->hrp == "ltc" &&
               decoded->has_witness_version;
    }
    const auto payload = base58check_decode(address);
    if (!payload || payload->size() != 21) return false;
    return (*payload)[0] == 0x30 || (*payload)[0] == 0x32;
}

// Dogecoin: Base58Check version 0x1e (D...).
inline bool is_valid_dogecoin_address(const std::string& address) {
    const auto payload = base58check_decode(address);
    if (!payload || payload->size() != 21) return false;
    return (*payload)[0] == 0x1e;
}

// Solana: Base58 of exactly 32 bytes. This is the check the previous code
// lacked; any 32-to-44 character string was accepted.
inline bool is_valid_solana_address(const std::string& address) {
    const auto decoded = base58_decode(address);
    return decoded.has_value() && decoded->size() == 32;
}

// Tron: Base58Check of a 21-byte payload with version 0x41, which renders as
// a T-prefixed address.
inline bool is_valid_tron_address(const std::string& address) {
    const auto payload = base58check_decode(address);
    if (!payload || payload->size() != 21) return false;
    return (*payload)[0] == 0x41;
}

// Cosmos and the chains that share its address format: Bech32 over 20 bytes of
// hash. The expected HRP is passed in because each chain has its own.
inline bool is_valid_cosmos_address(const std::string& address, const std::string& hrp) {
    const auto decoded = bech32_decode(address);
    if (!decoded) return false;
    if (decoded->hrp != hrp) return false;
    if (decoded->has_witness_version) return false;
    return decoded->data.size() == 20;
}

}  // namespace address
}  // namespace tigerwallet

#endif  // TIGER_WALLET_ADDRESS_HPP
