// Self-built Keccak-256 (Ethereum's keccak, not NIST SHA3-256).
//
// This is the permutation used for transaction hashes, addresses, event
// signatures and function selectors. It is implemented here rather than taken
// from a library so the wallet does not depend on one for a primitive this
// small, and so every module that needs it hashes the same way.
//
// The implementation was validated against the Keccak-256 reference vectors
// and against `openssl dgst -sha3-256` (with the 0x06 domain byte, which is
// the only difference between the two).

#ifndef TIGERWALLET_COMMON_KECCAK256_HPP
#define TIGERWALLET_COMMON_KECCAK256_HPP

#include <array>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <string>
#include <vector>

namespace tigerwallet {
namespace crypto {

class Keccak256 {
public:
    static constexpr size_t kDigestSize = 32;
    static constexpr size_t kRate = 136;  // 1088 bits.

    using Digest = std::array<uint8_t, kDigestSize>;

    // Hashes `len` bytes at `data`.
    static Digest hash(const uint8_t* data, size_t len) {
        uint64_t state[25] = {0};

        // Absorb whole blocks.
        const uint8_t* p = data;
        const uint8_t* const end = data + len;
        while (static_cast<size_t>(end - p) >= kRate) {
            absorb(state, p);
            p += kRate;
        }

        // Pad the final block: Keccak pads with 0x01 ... 0x80. SHA3-256 would
        // use 0x06, which produces different output for the same input.
        uint8_t last[kRate];
        const size_t remaining = static_cast<size_t>(end - p);
        if (remaining > 0) {
            std::memcpy(last, p, remaining);
        }
        last[remaining] = 0x01;
        for (size_t i = remaining + 1; i < kRate; ++i) {
            last[i] = 0x00;
        }
        last[kRate - 1] |= 0x80;
        absorb(state, last);

        Digest digest{};
        // Squeeze 32 bytes: four 64-bit lanes, little-endian.
        for (size_t i = 0; i < 4; ++i) {
            for (size_t j = 0; j < 8; ++j) {
                digest[i * 8 + j] =
                    static_cast<uint8_t>((state[i] >> (8 * j)) & 0xff);
            }
        }
        return digest;
    }

    static Digest hash(const std::string& data) {
        return hash(reinterpret_cast<const uint8_t*>(data.data()), data.size());
    }

    static Digest hash(const std::vector<uint8_t>& data) {
        return hash(data.data(), data.size());
    }

    // Lowercase hex, no 0x prefix.
    static std::string toHex(const Digest& digest) {
        static const char* kHex = "0123456789abcdef";
        std::string out;
        out.reserve(digest.size() * 2);
        for (uint8_t byte : digest) {
            out.push_back(kHex[byte >> 4]);
            out.push_back(kHex[byte & 0x0f]);
        }
        return out;
    }

    // Function selector: the first four bytes of the hash of a signature such
    // as "transfer(address,uint256)".
    static uint32_t selector(const std::string& signature) {
        const Digest digest = hash(signature);
        return (static_cast<uint32_t>(digest[0]) << 24) |
               (static_cast<uint32_t>(digest[1]) << 16) |
               (static_cast<uint32_t>(digest[2]) << 8) |
               static_cast<uint32_t>(digest[3]);
    }

private:
    static uint64_t rotl(uint64_t x, int n) {
        // n is never 0 at the call sites below; the rho offsets that are zero
        // are handled by the table entry of 0 producing a no-op shift, which
        // would be undefined behaviour. Rotating by 0 returns x unchanged.
        if (n == 0) return x;
        return (x << n) | (x >> (64 - n));
    }

    static void absorb(uint64_t state[25], const uint8_t* block) {
        for (size_t i = 0; i < kRate / 8; ++i) {
            uint64_t lane = 0;
            for (size_t j = 0; j < 8; ++j) {
                lane |= static_cast<uint64_t>(block[i * 8 + j]) << (8 * j);
            }
            state[i] ^= lane;
        }
        permute(state);
    }

    static void permute(uint64_t state[25]) {
        static const uint64_t kRoundConstants[24] = {
            0x0000000000000001ULL, 0x0000000000008082ULL, 0x800000000000808aULL,
            0x8000000080008000ULL, 0x000000000000808bULL, 0x0000000080000001ULL,
            0x8000000080008081ULL, 0x8000000000008009ULL, 0x000000000000008aULL,
            0x0000000000000088ULL, 0x0000000080008009ULL, 0x000000008000000aULL,
            0x000000008000808bULL, 0x800000000000008bULL, 0x8000000000008089ULL,
            0x8000000000008003ULL, 0x8000000000008002ULL, 0x8000000000000080ULL,
            0x000000000000800aULL, 0x800000008000000aULL, 0x8000000080008081ULL,
            0x8000000000008080ULL, 0x0000000080000001ULL, 0x8000000080008008ULL
        };
        // Rho offsets, indexed by lane.
        static const int kRotation[25] = {
            0,  1, 62, 28, 27,
            36, 44,  6, 55, 20,
            3, 10, 43, 25, 39,
            41, 45, 15, 21,  8,
            18,  2, 61, 56, 14
        };
        // Pi step as a permutation of lane indices.
        static const int kPi[25] = {
             0, 10, 20,  5, 15,
            16,  1, 11, 21,  6,
             7, 17,  2, 12, 22,
            23,  8, 18,  3, 13,
            14, 24,  9, 19,  4
        };

        for (int round = 0; round < 24; ++round) {
            // Theta.
            uint64_t column[5];
            for (int i = 0; i < 5; ++i) {
                column[i] = state[i] ^ state[i + 5] ^ state[i + 10] ^
                            state[i + 15] ^ state[i + 20];
            }
            uint64_t diff[5];
            for (int i = 0; i < 5; ++i) {
                diff[i] = column[(i + 4) % 5] ^ rotl(column[(i + 1) % 5], 1);
            }
            for (int i = 0; i < 25; ++i) {
                state[i] ^= diff[i % 5];
            }

            // Rho and Pi.
            uint64_t moved[25];
            for (int i = 0; i < 25; ++i) {
                moved[kPi[i]] = rotl(state[i], kRotation[i]);
            }

            // Chi.
            for (int j = 0; j < 5; ++j) {
                for (int i = 0; i < 5; ++i) {
                    state[i + 5 * j] =
                        moved[i + 5 * j] ^
                        ((~moved[(i + 1) % 5 + 5 * j]) & moved[(i + 2) % 5 + 5 * j]);
                }
            }

            // Iota.
            state[0] ^= kRoundConstants[round];
        }
    }
};

}  // namespace crypto
}  // namespace tigerwallet

#endif  // TIGERWALLET_COMMON_KECCAK256_HPP
