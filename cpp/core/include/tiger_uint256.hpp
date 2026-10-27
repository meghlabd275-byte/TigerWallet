// Minimal unsigned 256-bit integer for wei-denominated values.
//
// Values are four 64-bit limbs, least significant first. Only the operations
// the wallet actually performs are provided; this is not a general-purpose
// bignum. It exists because balances, gas prices and token amounts do not fit
// in 64 bits, and the previous code declared a uint256_t that was never
// defined anywhere.
//
// Prefer uint64_t for values that genuinely are 64-bit (gas limits, nonces,
// block numbers); using this type for those only adds cost.

#ifndef TIGER_UINT256_HPP
#define TIGER_UINT256_HPP

#include <array>
#include <cstdint>
#include <string>

namespace tiger {

struct uint256 {
    std::array<uint64_t, 4> limbs{};  // little-endian limbs

    uint256() = default;
    uint256(uint64_t v) { limbs[0] = v; }  // NOLINT: implicit by design

    // Parse a decimal string, or a 0x-prefixed hexadecimal string. Returns zero
    // for malformed input rather than throwing, so latency-sensitive callers
    // are not forced into exception handling. Callers that must reject bad
    // input should validate the string before calling.
    static uint256 from_string(const std::string& text) {
        uint256 out;
        if (text.empty()) return out;

        if (text.size() > 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X')) {
            size_t limb_index = 0;
            uint64_t limb = 0;
            int shift = 0;
            for (size_t i = text.size(); i-- > 2 && limb_index < 4;) {
                const char c = text[i];
                uint64_t digit;
                if (c >= '0' && c <= '9') digit = static_cast<uint64_t>(c - '0');
                else if (c >= 'a' && c <= 'f') digit = static_cast<uint64_t>(c - 'a' + 10);
                else if (c >= 'A' && c <= 'F') digit = static_cast<uint64_t>(c - 'A' + 10);
                else return {};
                limb |= digit << shift;
                shift += 4;
                if (shift == 64) {
                    out.limbs[limb_index++] = limb;
                    limb = 0;
                    shift = 0;
                }
            }
            if (shift > 0 && limb_index < 4) out.limbs[limb_index] = limb;
            return out;
        }

        for (char c : text) {
            if (c < '0' || c > '9') return {};
            uint64_t carry = static_cast<uint64_t>(c - '0');
            for (size_t i = 0; i < 4; ++i) {
                unsigned __int128 cur =
                    static_cast<unsigned __int128>(out.limbs[i]) * 10 + carry;
                out.limbs[i] = static_cast<uint64_t>(cur);
                carry = static_cast<uint64_t>(cur >> 64);
            }
        }
        return out;
    }

    // Lowercase 0x-prefixed hex, minimal width (no leading zero padding).
    std::string to_hex() const {
        static const char* digits = "0123456789abcdef";
        std::string out;
        bool started = false;
        for (size_t i = 4; i-- > 0;) {
            for (int shift = 60; shift >= 0; shift -= 4) {
                const uint8_t nibble = static_cast<uint8_t>((limbs[i] >> shift) & 0xF);
                if (!started && nibble == 0 && i != 0) continue;
                started = true;
                out.push_back(digits[nibble]);
            }
        }
        return "0x" + out;
    }

    // Decimal string, used where a human-readable amount is required.
    std::string to_string() const {
        if (is_zero()) return "0";
        std::string digits;
        uint256 value = *this;
        while (!value.is_zero()) {
            uint64_t remainder = 0;
            for (size_t i = 4; i-- > 0;) {
                unsigned __int128 cur = (static_cast<unsigned __int128>(remainder) << 64) | value.limbs[i];
                value.limbs[i] = static_cast<uint64_t>(cur / 10);
                remainder = static_cast<uint64_t>(cur % 10);
            }
            digits.push_back(static_cast<char>('0' + remainder));
        }
        return std::string(digits.rbegin(), digits.rend());
    }

    bool is_zero() const {
        return limbs[0] == 0 && limbs[1] == 0 && limbs[2] == 0 && limbs[3] == 0;
    }

    uint256 operator+(const uint256& other) const {
        uint256 out;
        unsigned __int128 carry = 0;
        for (size_t i = 0; i < 4; ++i) {
            unsigned __int128 cur =
                static_cast<unsigned __int128>(limbs[i]) + other.limbs[i] + carry;
            out.limbs[i] = static_cast<uint64_t>(cur);
            carry = cur >> 64;
        }
        return out;
    }

    uint256 operator-(const uint256& other) const {
        uint256 out;
        int64_t borrow = 0;
        for (size_t i = 0; i < 4; ++i) {
            unsigned __int128 cur = static_cast<unsigned __int128>(limbs[i]) -
                                    other.limbs[i] - static_cast<uint64_t>(borrow);
            out.limbs[i] = static_cast<uint64_t>(cur);
            borrow = (cur >> 64) ? 1 : 0;
        }
        return out;
    }

    uint256 operator*(uint64_t factor) const {
        uint256 out;
        unsigned __int128 carry = 0;
        for (size_t i = 0; i < 4; ++i) {
            unsigned __int128 cur =
                static_cast<unsigned __int128>(limbs[i]) * factor + carry;
            out.limbs[i] = static_cast<uint64_t>(cur);
            carry = cur >> 64;
        }
        return out;
    }

    uint256 operator/(uint64_t divisor) const {
        uint256 out;
        if (divisor == 0) return out;
        unsigned __int128 remainder = 0;
        for (size_t i = 4; i-- > 0;) {
            unsigned __int128 cur = (remainder << 64) | limbs[i];
            out.limbs[i] = static_cast<uint64_t>(cur / divisor);
            remainder = cur % divisor;
        }
        return out;
    }

    uint256& operator+=(const uint256& other) { return *this = *this + other; }
    uint256& operator-=(const uint256& other) { return *this = *this - other; }

    // Truncating conversion. Only valid where the caller has already bounded
    // the value to 64 bits.
    uint64_t to_uint64() const { return limbs[0]; }

    // Lossy above 2^53; intended for display and rough comparisons only.
    double to_double() const {
        double out = 0.0;
        for (size_t i = 4; i-- > 0;) {
            out = out * 18446744073709551616.0 + static_cast<double>(limbs[i]);
        }
        return out;
    }

    bool operator==(const uint256& o) const { return limbs == o.limbs; }
    bool operator!=(const uint256& o) const { return !(*this == o); }
    bool operator<(const uint256& o) const {
        for (size_t i = 4; i-- > 0;) {
            if (limbs[i] != o.limbs[i]) return limbs[i] < o.limbs[i];
        }
        return false;
    }
    bool operator>(const uint256& o) const { return o < *this; }
    bool operator<=(const uint256& o) const { return !(o < *this); }
    bool operator>=(const uint256& o) const { return !(*this < o); }
};

}  // namespace tiger

#endif  // TIGER_UINT256_HPP
