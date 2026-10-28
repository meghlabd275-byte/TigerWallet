// Self-built 256-bit unsigned integer arithmetic.
//
// EVM amounts are 256-bit. Computing them in 64-bit words and ignoring the
// upper words produces a wrong answer without any indication: the value wraps
// silently, and a swap quote built that way is simply incorrect.
//
// Only what the wallet needs is here: multiply with a full 512-bit product,
// and division of that product by a 256-bit divisor. That combination is
// enough for the constant-product AMM formula
//     amountOut = (amountIn * 997 * reserveOut) / (reserveIn * 1000 + amountIn * 997)
// without losing precision to truncation.

#ifndef TIGERWALLET_COMMON_UINT256_HPP
#define TIGERWALLET_COMMON_UINT256_HPP

#include <array>
#include <cstdint>
#include <string>

namespace tigerwallet {
namespace math {

// Four 64-bit words, least significant first. This matches the layout the rest
// of the codebase uses for 256-bit values.
struct U256 {
    std::array<uint64_t, 4> word{{0, 0, 0, 0}};

    U256() = default;
    explicit U256(uint64_t value) { word = {value, 0, 0, 0}; }

    bool isZero() const {
        return word[0] == 0 && word[1] == 0 && word[2] == 0 && word[3] == 0;
    }

    // Truncating conversion, for values known to fit.
    uint64_t toUint64() const { return word[0]; }

    static U256 fromWords(uint64_t w0, uint64_t w1, uint64_t w2, uint64_t w3) {
        U256 v;
        v.word = {w0, w1, w2, w3};
        return v;
    }

    static U256 fromUint64(uint64_t value) { return U256(value); }

    static U256 fromDecimal(const std::string& digits);

    std::string toDecimal() const;

    // -1, 0 or 1.
    int compare(const U256& other) const {
        for (int i = 3; i >= 0; --i) {
            if (word[i] < other.word[i]) return -1;
            if (word[i] > other.word[i]) return 1;
        }
        return 0;
    }

    bool operator==(const U256& other) const { return compare(other) == 0; }
    bool operator!=(const U256& other) const { return compare(other) != 0; }
    bool operator<(const U256& other) const { return compare(other) < 0; }
    bool operator>(const U256& other) const { return compare(other) > 0; }
    bool operator<=(const U256& other) const { return compare(other) <= 0; }
    bool operator>=(const U256& other) const { return compare(other) >= 0; }

    U256 operator+(const U256& other) const {
        U256 out;
        unsigned __int128 carry = 0;
        for (int i = 0; i < 4; ++i) {
            const unsigned __int128 sum =
                static_cast<unsigned __int128>(word[i]) + other.word[i] + carry;
            out.word[i] = static_cast<uint64_t>(sum);
            carry = sum >> 64;
        }
        return out;  // Wraps on overflow, matching EVM unchecked addition.
    }

    U256 operator-(const U256& other) const {
        U256 out;
        unsigned __int128 borrow = 0;
        for (int i = 0; i < 4; ++i) {
            const unsigned __int128 diff =
                static_cast<unsigned __int128>(word[i]) - other.word[i] - borrow;
            out.word[i] = static_cast<uint64_t>(diff);
            borrow = (diff >> 64) ? 1 : 0;
        }
        return out;
    }

    U256 operator*(const U256& other) const {
        U256 out;
        for (int i = 0; i < 4; ++i) {
            unsigned __int128 carry = 0;
            for (int j = 0; i + j < 4; ++j) {
                const unsigned __int128 product =
                    static_cast<unsigned __int128>(word[i]) * other.word[j] +
                    out.word[i + j] + carry;
                out.word[i + j] = static_cast<uint64_t>(product);
                carry = product >> 64;
            }
        }
        return out;
    }

    // Long division by a single 64-bit word. Used by the decimal conversion.
    U256 divmodWord(uint64_t divisor, uint64_t& remainder) const {
        U256 quotient;
        unsigned __int128 rem = 0;
        for (int i = 3; i >= 0; --i) {
            const unsigned __int128 current = (rem << 64) | word[i];
            quotient.word[i] = static_cast<uint64_t>(current / divisor);
            rem = current % divisor;
        }
        remainder = static_cast<uint64_t>(rem);
        return quotient;
    }
};

// The 512-bit product of two 256-bit values, as eight 64-bit words.
struct U512 {
    std::array<uint64_t, 8> word{{0, 0, 0, 0, 0, 0, 0, 0}};

    bool isZero() const {
        for (uint64_t w : word) {
            if (w != 0) return false;
        }
        return true;
    }

    int compare(const U512& other) const {
        for (int i = 7; i >= 0; --i) {
            if (word[i] < other.word[i]) return -1;
            if (word[i] > other.word[i]) return 1;
        }
        return 0;
    }
};

inline U512 multiplyFull(const U256& a, const U256& b) {
    U512 out;
    for (int i = 0; i < 4; ++i) {
        unsigned __int128 carry = 0;
        for (int j = 0; j < 4; ++j) {
            const unsigned __int128 product =
                static_cast<unsigned __int128>(a.word[i]) * b.word[j] +
                out.word[i + j] + carry;
            out.word[i + j] = static_cast<uint64_t>(product);
            carry = product >> 64;
        }
        // Propagate the final carry into the high words.
        int k = i + 4;
        while (carry != 0 && k < 8) {
            const unsigned __int128 sum =
                static_cast<unsigned __int128>(out.word[k]) + carry;
            out.word[k] = static_cast<uint64_t>(sum);
            carry = sum >> 64;
            ++k;
        }
    }
    return out;
}

// Divides a 512-bit value by a 256-bit divisor, returning the 256-bit
// quotient and the remainder. Binary long division, which is straightforward
// to verify even though it is not the fastest possible method.
inline U256 divideFull(const U512& numerator, const U256& divisor,
                       U256* remainder_out = nullptr) {
    U512 remainder;
    U256 quotient;

    if (divisor.isZero()) {
        // Division by zero has no result. Callers must check the divisor; a
        // zero here means the caller did not.
        if (remainder_out) *remainder_out = U256();
        return U256();
    }

    // Walk the numerator from the most significant bit down.
    for (int bit = 511; bit >= 0; --bit) {
        // remainder <<= 1
        for (int i = 7; i >= 0; --i) {
            remainder.word[i] <<= 1;
            if (i > 0 && (remainder.word[i - 1] >> 63) & 1ULL) {
                remainder.word[i] |= 1ULL;
            }
        }
        // Bring the next numerator bit in.
        const uint64_t next = (numerator.word[bit / 64] >> (bit % 64)) & 1ULL;
        remainder.word[0] |= next;

        // If the remainder now holds the divisor, subtract and record the bit.
        U512 divisor_extended;
        for (int i = 0; i < 4; ++i) {
            divisor_extended.word[i] = divisor.word[i];
        }
        if (remainder.compare(divisor_extended) >= 0) {
            unsigned __int128 borrow = 0;
            for (int i = 0; i < 8; ++i) {
                const unsigned __int128 diff =
                    static_cast<unsigned __int128>(remainder.word[i]) -
                    divisor_extended.word[i] - borrow;
                remainder.word[i] = static_cast<uint64_t>(diff);
                borrow = (diff >> 64) ? 1 : 0;
            }
            quotient.word[bit / 64] |= (1ULL << (bit % 64));
        }
    }

    if (remainder_out) {
        remainder_out->word = {remainder.word[0], remainder.word[1],
                               remainder.word[2], remainder.word[3]};
    }
    return quotient;
}

// a * b / c computed with a 512-bit intermediate, so the multiplication cannot
// overflow before the division brings the result back into range. This is the
// shape every AMM quote needs.
inline U256 mulDiv(const U256& a, const U256& b, const U256& c) {
    if (c.isZero()) return U256();
    return divideFull(multiplyFull(a, b), c);
}

inline std::string toHex(const U256& value) {
    static const char* kHex = "0123456789abcdef";
    std::string out;
    bool started = false;
    for (int i = 3; i >= 0; --i) {
        for (int shift = 60; shift >= 0; shift -= 4) {
            const uint8_t nibble =
                static_cast<uint8_t>((value.word[i] >> shift) & 0xf);
            if (nibble != 0 || started || i == 0) {
                started = true;
                out.push_back(kHex[nibble]);
            }
        }
    }
    return out.empty() ? "0" : out;
}

inline U256 U256::fromDecimal(const std::string& digits) {
    U256 value;
    const U256 ten(10);
    for (char c : digits) {
        if (c < '0' || c > '9') break;
        value = value * ten + U256(static_cast<uint64_t>(c - '0'));
    }
    return value;
}

inline std::string U256::toDecimal() const {
    if (isZero()) return "0";
    std::string digits;
    U256 current = *this;
    while (!current.isZero()) {
        uint64_t remainder = 0;
        current = current.divmodWord(10, remainder);
        digits.insert(digits.begin(), static_cast<char>('0' + remainder));
    }
    return digits;
}

}  // namespace math
}  // namespace tigerwallet

#endif  // TIGERWALLET_COMMON_UINT256_HPP
