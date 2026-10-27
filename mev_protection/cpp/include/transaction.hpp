/**
 * Transaction data structures
 */

#ifndef TRANSACTION_HPP
#define TRANSACTION_HPP

#include <array>
#include <cstdint>
#include <cstring>
#include <string>
#include <vector>
#include <chrono>

namespace tiger {

// Minimal unsigned 256-bit integer used for wei-denominated values. Values are
// stored as four 64-bit limbs, least significant first, and the arithmetic the
// MEV module performs (comparison, addition, scaling) is defined here. This is
// deliberately small: it is not a general-purpose bignum, and the operators
// below are the ones callers actually use.
struct uint256 {
    std::array<uint64_t, 4> limbs{};  // little-endian limbs

    uint256() = default;
    uint256(uint64_t v) { limbs[0] = v; }

    // Construct from a decimal or 0x-prefixed hexadecimal string. Returns a
    // zero value for malformed input rather than throwing, so callers in
    // latency-sensitive paths are not forced into exception handling.
    static uint256 from_string(const std::string& text) {
        uint256 out;
        if (text.empty()) return out;
        if (text.size() > 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X')) {
            size_t limb_index = 0;
            uint64_t limb = 0;
            int shift = 0;
            for (size_t i = text.size(); i-- > 2 && limb_index < 4;) {
                char c = text[i];
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
                unsigned __int128 cur = static_cast<unsigned __int128>(out.limbs[i]) * 10 + carry;
                out.limbs[i] = static_cast<uint64_t>(cur);
                carry = static_cast<uint64_t>(cur >> 64);
            }
        }
        return out;
    }

    bool is_zero() const {
        return limbs[0] == 0 && limbs[1] == 0 && limbs[2] == 0 && limbs[3] == 0;
    }

    uint256 operator+(const uint256& other) const {
        uint256 out;
        unsigned __int128 carry = 0;
        for (size_t i = 0; i < 4; ++i) {
            unsigned __int128 cur = static_cast<unsigned __int128>(limbs[i]) + other.limbs[i] + carry;
            out.limbs[i] = static_cast<uint64_t>(cur);
            carry = cur >> 64;
        }
        return out;
    }

    uint256 operator-(const uint256& other) const {
        uint256 out;
        int64_t borrow = 0;
        for (size_t i = 0; i < 4; ++i) {
            unsigned __int128 cur = static_cast<unsigned __int128>(limbs[i]) - other.limbs[i] - static_cast<uint64_t>(borrow);
            out.limbs[i] = static_cast<uint64_t>(cur);
            borrow = (cur >> 64) ? 1 : 0;
        }
        return out;
    }

    uint256 operator*(uint64_t factor) const {
        uint256 out;
        unsigned __int128 carry = 0;
        for (size_t i = 0; i < 4; ++i) {
            unsigned __int128 cur = static_cast<unsigned __int128>(limbs[i]) * factor + carry;
            out.limbs[i] = static_cast<uint64_t>(cur);
            carry = cur >> 64;
        }
        return out;
    }

    uint256 operator/(uint64_t divisor) const {
        uint256 out;
        if (divisor == 0) return out;
        unsigned __int128 rem = 0;
        for (size_t i = 4; i-- > 0;) {
            unsigned __int128 cur = (rem << 64) | limbs[i];
            out.limbs[i] = static_cast<uint64_t>(cur / divisor);
            rem = cur % divisor;
        }
        return out;
    }

    // Truncating conversion, used only where the caller has already bounded
    // the value (gas prices and gas limits both fit comfortably in 64 bits).
    uint64_t to_uint64() const { return limbs[0]; }

    // Scaled fixed-point conversion to double for profit estimation.
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

struct Transaction {
    std::string hash;
    std::string from;
    std::string to;
    uint256 value;
    uint256 gas_price;
    uint64_t gas_limit;
    uint64_t gas_used;
    std::string data;
    uint64_t nonce;
    uint64_t chain_id;
    std::string v;
    std::string r;
    std::string s;
    
    // Additional metadata
    std::chrono::steady_clock::time_point received_at;
    std::string raw;
    bool is_private;
    std::string relayer;
    
    Transaction() 
        : gas_limit(21000),
          gas_used(0),
          nonce(0),
          chain_id(1),
          is_private(false) {
        received_at = std::chrono::steady_clock::now();
    }
};

struct Log {
    std::string address;
    std::vector<std::string> topics;
    std::string data;
    uint64_t log_index;
};

struct TransactionReceipt {
    std::string transaction_hash;
    std::string block_hash;
    uint64_t block_number;
    uint64_t gas_used;
    std::string status;
    std::vector<Log> logs;
    
    TransactionReceipt() : block_number(0), gas_used(0) {}
};

struct Block {
    uint64_t number;
    std::string hash;
    std::string parent_hash;
    std::string miner;
    uint256 base_fee_per_gas;
    uint64_t gas_limit;
    uint64_t gas_used;
    std::string timestamp;
    std::vector<Transaction> transactions;
    
    Block() : number(0), gas_limit(0), gas_used(0) {}
};

}  // namespace tiger

#endif  // TRANSACTION_HPP
