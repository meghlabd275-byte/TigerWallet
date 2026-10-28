// Self-built RLP (Recursive Length Prefix) encoder.
//
// RLP is how Ethereum serialises transactions before hashing and signing. A
// transaction's hash is keccak256 of its RLP encoding, so getting this wrong
// produces a hash that no node recognises.
//
// The rules, from the Ethereum yellow paper:
//   - A single byte below 0x80 encodes as itself.
//   - A string of 0 to 55 bytes encodes as 0x80+len followed by the bytes.
//   - A string longer than 55 bytes encodes as 0xb7+len(len), then len, then
//     the bytes.
//   - A list encodes as 0xc0+len for short lists, or 0xf7+len(len) for long
//     ones, followed by the concatenated encodings of its items.
//
// Note that a byte string is NOT the same as an integer. 0x00 is the empty
// string under RLP, while 0x80 is the integer zero. Encoding an integer
// requires stripping leading zero bytes first.

#ifndef TIGERWALLET_COMMON_RLP_HPP
#define TIGERWALLET_COMMON_RLP_HPP

#include <cstddef>
#include <cstdint>
#include <string>
#include <vector>

namespace tigerwallet {
namespace rlp {

using Bytes = std::vector<uint8_t>;

class Encoder {
public:
    // A byte string, as given. An empty string encodes as 0x80.
    static Bytes encodeBytes(const uint8_t* data, size_t len) {
        Bytes out;
        if (len == 1 && data[0] < 0x80) {
            out.push_back(data[0]);
            return out;
        }
        appendLength(out, len, 0x80, 0xb7);
        out.insert(out.end(), data, data + len);
        return out;
    }

    static Bytes encodeBytes(const Bytes& data) {
        return encodeBytes(data.data(), data.size());
    }

    static Bytes encodeString(const std::string& data) {
        return encodeBytes(reinterpret_cast<const uint8_t*>(data.data()),
                           data.size());
    }

    // A big-endian unsigned integer with leading zeros stripped. Zero encodes
    // as the empty string, which is 0x80, not 0x00.
    static Bytes encodeUint(uint64_t value) {
        if (value == 0) {
            return Bytes{0x80};
        }
        Bytes big_endian;
        while (value > 0) {
            big_endian.insert(big_endian.begin(),
                              static_cast<uint8_t>(value & 0xff));
            value >>= 8;
        }
        return encodeBytes(big_endian);
    }

    // A 256-bit value held as four little-endian 64-bit words, the layout this
    // codebase uses. Encoded as a minimal big-endian integer.
    static Bytes encodeUint256(const uint64_t words[4]) {
        Bytes big_endian;
        for (int i = 3; i >= 0; --i) {
            for (int shift = 56; shift >= 0; shift -= 8) {
                big_endian.push_back(
                    static_cast<uint8_t>((words[i] >> shift) & 0xff));
            }
        }
        // Strip leading zeros so the integer form is minimal.
        size_t first = 0;
        while (first < big_endian.size() && big_endian[first] == 0x00) {
            ++first;
        }
        if (first == big_endian.size()) {
            return Bytes{0x80};  // The value is zero.
        }
        big_endian.erase(big_endian.begin(),
                         big_endian.begin() + static_cast<long>(first));
        return encodeBytes(big_endian);
    }

    // A list of already-encoded items.
    static Bytes encodeList(const std::vector<Bytes>& items) {
        Bytes payload;
        for (const Bytes& item : items) {
            payload.insert(payload.end(), item.begin(), item.end());
        }
        Bytes out;
        appendLength(out, payload.size(), 0xc0, 0xf7);
        out.insert(out.end(), payload.begin(), payload.end());
        return out;
    }

    static std::string toHex(const Bytes& data) {
        static const char* kHex = "0123456789abcdef";
        std::string out;
        out.reserve(data.size() * 2);
        for (uint8_t byte : data) {
            out.push_back(kHex[byte >> 4]);
            out.push_back(kHex[byte & 0x0f]);
        }
        return out;
    }

private:
    // Writes the length prefix. `short_base` is the offset for payloads of up
    // to 55 bytes (0x80 for strings, 0xc0 for lists) and `long_base` is the
    // offset used when the length itself needs encoding (0xb7, 0xf7).
    static void appendLength(Bytes& out, size_t len, uint8_t short_base,
                             uint8_t long_base) {
        if (len <= 55) {
            out.push_back(static_cast<uint8_t>(short_base + len));
            return;
        }
        Bytes len_bytes;
        size_t remaining = len;
        while (remaining > 0) {
            len_bytes.insert(len_bytes.begin(),
                             static_cast<uint8_t>(remaining & 0xff));
            remaining >>= 8;
        }
        out.push_back(static_cast<uint8_t>(long_base + len_bytes.size()));
        out.insert(out.end(), len_bytes.begin(), len_bytes.end());
    }
};

}  // namespace rlp
}  // namespace tigerwallet

#endif  // TIGERWALLET_COMMON_RLP_HPP
