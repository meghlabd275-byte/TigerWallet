// ABI encoding and decimal arithmetic for the DEX connectors.
//
// The encoders cover the small set of shapes the connectors need: a selector
// plus address/uint256 words, and decoding of router return values (a uint256
// array, a string and an address array). The test suite checks them against
// vectors from the Ethereum ABI specification.
//
// Big integers are held as plain digit strings. The connectors only need
// multiplication, division and base conversion, and a string is easier to
// verify by eye than a packed vector.

#include "dex_connector.hpp"

#include <algorithm>
#include <cctype>
#include <cstddef>
#include <stdexcept>

#include "crypto.hpp"

namespace tigerwallet {
namespace dex {
namespace {

const char* kHexDigits = "0123456789abcdef";

int hex_value(char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

std::string to_hex(const uint8_t* data, size_t length) {
    std::string out;
    out.reserve(length * 2);
    for (size_t i = 0; i < length; ++i) {
        out.push_back(kHexDigits[data[i] >> 4]);
        out.push_back(kHexDigits[data[i] & 0x0f]);
    }
    return out;
}

bool normalise_hex(const std::string& in, std::string& out) {
    std::string s = in;
    if (s.size() >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X')) {
        s = s.substr(2);
    }
    if (s.empty() || s.size() % 2 != 0) return false;
    for (char c : s) {
        if (hex_value(c) < 0) return false;
    }
    out = s;
    return true;
}

std::vector<uint8_t> hex_to_bytes(const std::string& hex) {
    std::vector<uint8_t> out;
    out.reserve(hex.size() / 2);
    for (size_t i = 0; i + 1 < hex.size(); i += 2) {
        out.push_back(static_cast<uint8_t>((hex_value(hex[i]) << 4) | hex_value(hex[i + 1])));
    }
    return out;
}

// --- decimal string arithmetic ----------------------------------------------

std::string strip_leading_zeroes(const std::string& s) {
    size_t i = 0;
    while (i + 1 < s.size() && s[i] == '0') ++i;
    return s.substr(i);
}

bool is_digits(const std::string& s) {
    if (s.empty()) return false;
    for (char c : s) {
        if (c < '0' || c > '9') return false;
    }
    return true;
}

std::string dec_mul(const std::string& a, const std::string& b) {
    const std::string na = strip_leading_zeroes(a);
    const std::string nb = strip_leading_zeroes(b);
    if (na == "0" || nb == "0") return "0";

    std::vector<int> acc(na.size() + nb.size(), 0);
    for (size_t i = na.size(); i-- > 0;) {
        for (size_t j = nb.size(); j-- > 0;) {
            acc[i + j + 1] += (na[i] - '0') * (nb[j] - '0');
        }
    }
    for (size_t i = acc.size(); i-- > 1;) {
        acc[i - 1] += acc[i] / 10;
        acc[i] %= 10;
    }
    std::string out;
    out.reserve(acc.size());
    for (int d : acc) out.push_back(static_cast<char>('0' + d));
    return strip_leading_zeroes(out);
}

int dec_cmp(const std::string& a, const std::string& b) {
    const std::string na = strip_leading_zeroes(a);
    const std::string nb = strip_leading_zeroes(b);
    if (na.size() != nb.size()) return na.size() < nb.size() ? -1 : 1;
    if (na == nb) return 0;
    return na < nb ? -1 : 1;
}

std::string dec_sub(const std::string& a, const std::string& b) {
    std::string result;
    int borrow = 0;
    size_t i = a.size(), j = b.size();
    while (i > 0) {
        int digit = (a[--i] - '0') - borrow;
        borrow = 0;
        if (j > 0) digit -= (b[--j] - '0');
        if (digit < 0) {
            digit += 10;
            borrow = 1;
        }
        result.push_back(static_cast<char>('0' + digit));
    }
    std::reverse(result.begin(), result.end());
    return strip_leading_zeroes(result);
}

// Long division: returns the quotient and reports the remainder.
std::string dec_divmod(const std::string& a, const std::string& b, std::string& remainder) {
    const std::string nb = strip_leading_zeroes(b);
    remainder = "0";
    std::string quotient;
    quotient.reserve(a.size());

    for (char c : strip_leading_zeroes(a)) {
        remainder = (remainder == "0") ? std::string(1, c) : remainder + c;
        remainder = strip_leading_zeroes(remainder);

        int digit = 0;
        while (dec_cmp(remainder, nb) >= 0) {
            remainder = dec_sub(remainder, nb);
            ++digit;
        }
        quotient.push_back(static_cast<char>('0' + digit));
    }
    return strip_leading_zeroes(quotient);
}

// --- base 256 <-> base 10 ----------------------------------------------------

std::vector<uint8_t> dec_to_be_bytes(const std::string& decimal) {
    std::string value = strip_leading_zeroes(decimal);
    std::vector<uint8_t> bytes;
    if (value == "0") return bytes;

    while (value != "0") {
        std::string remainder;
        value = dec_divmod(value, "256", remainder);
        bytes.push_back(static_cast<uint8_t>(std::stoi(remainder)));
    }
    std::reverse(bytes.begin(), bytes.end());
    return bytes;
}

std::string be_bytes_to_dec(const uint8_t* data, size_t length) {
    std::string value = "0";
    for (size_t i = 0; i < length; ++i) {
        value = dec_mul(value, "256");
        std::string sum = value;
        std::string add = std::to_string(data[i]);
        // value += data[i]
        std::string out;
        int carry = 0;
        size_t x = sum.size(), y = add.size();
        while (x > 0 || y > 0 || carry > 0) {
            int digit = carry;
            if (x > 0) digit += sum[--x] - '0';
            if (y > 0) digit += add[--y] - '0';
            out.push_back(static_cast<char>('0' + (digit % 10)));
            carry = digit / 10;
        }
        std::reverse(out.begin(), out.end());
        value = strip_leading_zeroes(out);
    }
    return value;
}

bool word_at(const std::string& hex, size_t index, std::string& out) {
    if (hex.size() < (index + 1) * 64) return false;
    const std::vector<uint8_t> bytes = hex_to_bytes(hex.substr(index * 64, 64));
    out = be_bytes_to_dec(bytes.data(), bytes.size());
    return true;
}

}  // namespace

const char* to_string(DexProtocol protocol) {
    switch (protocol) {
        case DexProtocol::UNISWAP_V2:  return "Uniswap V2";
        case DexProtocol::UNISWAP_V3:  return "Uniswap V3";
        case DexProtocol::SUSHISWAP:   return "SushiSwap";
        case DexProtocol::PANCAKESWAP: return "PancakeSwap";
        case DexProtocol::QUICKSWAP:   return "QuickSwap";
        case DexProtocol::CURVE:       return "Curve";
        case DexProtocol::BALANCER:    return "Balancer";
    }
    return "Unknown";
}

std::string function_selector(const std::string& signature) {
    const auto digest = crypto::Keccak256(
        reinterpret_cast<const uint8_t*>(signature.data()), signature.size());
    return to_hex(digest.data(), 4);
}

std::string encode_address(const std::string& address) {
    std::string hex;
    if (!normalise_hex(address, hex) || hex.size() != 40) {
        throw std::invalid_argument("address must be 20 bytes of hex");
    }
    // Lowercase is the canonical form for calldata, so the encoding of an
    // address does not depend on how the caller happened to capitalise it.
    for (char& c : hex) c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
    return std::string(24, '0') + hex;
}

bool encode_uint256(const std::string& decimal, std::string& out_word) {
    if (!is_digits(decimal)) return false;
    const std::vector<uint8_t> bytes = dec_to_be_bytes(decimal);
    if (bytes.size() > 32) return false;
    std::string word(64, '0');
    word.replace(64 - bytes.size() * 2, bytes.size() * 2, to_hex(bytes.data(), bytes.size()));
    out_word = word;
    return true;
}

std::string decode_uint256(const std::string& word32) {
    std::string hex;
    if (!normalise_hex(word32, hex) || hex.size() != 64) return "";
    const std::vector<uint8_t> bytes = hex_to_bytes(hex);
    return be_bytes_to_dec(bytes.data(), bytes.size());
}

std::string decode_string(const std::string& return_data) {
    std::string hex;
    if (!normalise_hex(return_data, hex) || hex.size() < 128) return "";

    std::string offset_dec;
    if (!word_at(hex, 0, offset_dec)) return "";
    const size_t offset = static_cast<size_t>(std::stoull(offset_dec));
    if (offset % 32 != 0 || hex.size() < (offset + 32) * 2) return "";

    std::string length_dec;
    if (!word_at(hex, offset / 32, length_dec)) return "";
    const size_t length = static_cast<size_t>(std::stoull(length_dec));
    const size_t data_start = (offset + 32) * 2;
    if (hex.size() < data_start + length * 2) return "";

    const std::vector<uint8_t> bytes = hex_to_bytes(hex.substr(data_start, length * 2));
    return std::string(bytes.begin(), bytes.end());
}

std::vector<std::string> decode_address_array(const std::string& return_data) {
    std::vector<std::string> out;
    std::string hex;
    if (!normalise_hex(return_data, hex) || hex.size() < 128) return out;

    std::string offset_dec;
    if (!word_at(hex, 0, offset_dec)) return out;
    const size_t offset = static_cast<size_t>(std::stoull(offset_dec));
    if (offset % 32 != 0) return out;

    std::string count_dec;
    if (!word_at(hex, offset / 32, count_dec)) return out;
    const size_t count = static_cast<size_t>(std::stoull(count_dec));

    for (size_t i = 0; i < count; ++i) {
        const size_t word_index = offset / 32 + 1 + i;
        if (hex.size() < (word_index + 1) * 64) return {};
        out.push_back("0x" + hex.substr(word_index * 64 + 24, 40));
    }
    return out;
}

bool decode_amounts_array(const std::string& return_data, std::vector<std::string>& out) {
    out.clear();
    std::string hex;
    if (!normalise_hex(return_data, hex) || hex.size() < 128) return false;

    std::string offset_dec;
    if (!word_at(hex, 0, offset_dec)) return false;
    const size_t offset = static_cast<size_t>(std::stoull(offset_dec));
    if (offset % 32 != 0) return false;

    std::string count_dec;
    if (!word_at(hex, offset / 32, count_dec)) return false;
    const size_t count = static_cast<size_t>(std::stoull(count_dec));
    if (count == 0 || count > 64) return false;

    for (size_t i = 0; i < count; ++i) {
        std::string value;
        if (!word_at(hex, offset / 32 + 1 + i, value)) return false;
        out.push_back(value);
    }
    return true;
}

std::string apply_fee(const std::string& amount, uint32_t fee_bps) {
    if (fee_bps > 10000 || !is_digits(amount)) return "0";
    std::string product;
    if (!decimal_mul_div(amount, std::to_string(10000u - fee_bps), "10000", product)) {
        return "0";
    }
    return product;
}

bool decimal_add(const std::string& a, const std::string& b, std::string& out) {
    if (!is_digits(a) || !is_digits(b)) return false;
    std::string result;
    int carry = 0;
    size_t i = a.size(), j = b.size();
    while (i > 0 || j > 0 || carry > 0) {
        int sum = carry;
        if (i > 0) sum += a[--i] - '0';
        if (j > 0) sum += b[--j] - '0';
        result.push_back(static_cast<char>('0' + (sum % 10)));
        carry = sum / 10;
    }
    std::reverse(result.begin(), result.end());
    out = strip_leading_zeroes(result);
    return true;
}

bool decimal_sub(const std::string& a, const std::string& b, std::string& out) {
    if (!is_digits(a) || !is_digits(b)) return false;
    if (dec_cmp(a, b) < 0) return false;
    out = dec_sub(a, b);
    return true;
}

bool decimal_mul_div(const std::string& a, const std::string& b,
                     const std::string& divisor, std::string& out) {
    if (!is_digits(a) || !is_digits(b) || !is_digits(divisor)) return false;
    if (strip_leading_zeroes(divisor) == "0") return false;
    std::string remainder;
    out = dec_divmod(dec_mul(a, b), divisor, remainder);
    return true;
}

}  // namespace dex
}  // namespace tigerwallet
