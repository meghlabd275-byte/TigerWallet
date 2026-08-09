/**
 * TigerWallet Desktop - Keccak-256 Implementation
 *
 * Real Keccak-256 (the Ethereum hash function, NOT NIST FIPS 202 SHA3-256).
 * The difference: Keccak uses the original padding (0x01) while SHA3 uses
 * 0x06. Ethereum uses Keccak-256 with the 0x01 padding.
 */

#ifndef TIGERWALLET_KECCAK256_H
#define TIGERWALLET_KECCAK256_H

#include <cstdint>
#include <cstring>
#include <vector>
#include <string>

namespace tiger {
namespace wallet {

class Keccak256 {
public:
    Keccak256() { reset(); }

    void reset() {
        memset(state_, 0, sizeof(state_));
        rate_ = 136;
        pos_ = 0;
    }

    void update(const uint8_t* data, size_t len) {
        while (len > 0) {
            size_t toAbsorb = rate_ - pos_;
            if (toAbsorb > len) toAbsorb = len;
            absorb(data, toAbsorb);
            data += toAbsorb;
            len -= toAbsorb;
        }
    }

    void update(const std::string& data) {
        update(reinterpret_cast<const uint8_t*>(data.data()), data.size());
    }

    std::vector<uint8_t> finalize() {
        buf_[pos_++] = 0x01;
        if (pos_ > rate_) {
            while (pos_ < rate_) buf_[pos_++] = 0x00;
            absorbBlock();
        }
        while (pos_ < rate_) buf_[pos_++] = 0x00;
        buf_[rate_ - 1] |= 0x80;
        absorbBlock();

        std::vector<uint8_t> out(32);
        memcpy(out.data(), state_, 32);
        return out;
    }

    static std::vector<uint8_t> hash(const uint8_t* data, size_t len) {
        Keccak256 k;
        k.update(data, len);
        return k.finalize();
    }

    static std::vector<uint8_t> hash(const std::string& data) {
        return hash(reinterpret_cast<const uint8_t*>(data.data()), data.size());
    }

    static std::string hex(const std::vector<uint8_t>& bytes) {
        static const char* hexchars = "0123456789abcdef";
        std::string out;
        out.reserve(bytes.size() * 2);
        for (auto b : bytes) {
            out += hexchars[b >> 4];
            out += hexchars[b & 0xf];
        }
        return out;
    }

private:
    uint64_t state_[25];
    uint8_t buf_[136];
    size_t rate_;
    size_t pos_;

    void absorb(const uint8_t* data, size_t len) {
        memcpy(buf_ + pos_, data, len);
        pos_ += len;
        if (pos_ == rate_) absorbBlock();
    }

    void absorbBlock() {
        for (size_t i = 0; i < rate_ / 8; i++) {
            uint64_t lane = 0;
            memcpy(&lane, buf_ + i * 8, 8);
            state_[i] ^= lane;
        }
        keccakF();
        pos_ = 0;
    }

    void keccakF() {
        static const uint64_t RC[24] = {
            0x0000000000000001ULL, 0x0000000000008082ULL, 0x800000000000808AULL,
            0x8000000080008000ULL, 0x000000000000808BULL, 0x0000000080000001ULL,
            0x8000000080008081ULL, 0x8000000000008009ULL, 0x000000000000008AULL,
            0x0000000000000088ULL, 0x0000000080008009ULL, 0x000000008000000AULL,
            0x000000008000808BULL, 0x800000000000008BULL, 0x8000000000008089ULL,
            0x8000000000008003ULL, 0x8000000000008002ULL, 0x8000000000000080ULL,
            0x000000000000800AULL, 0x800000008000000AULL, 0x8000000080008081ULL,
            0x8000000000008080ULL, 0x0000000080000001ULL, 0x8000000080008008ULL
        };
        static const int ROTC[24] = {
            1, 3, 6, 10, 15, 21, 28, 36, 45, 55, 2, 14,
            27, 41, 56, 8, 25, 43, 62, 18, 39, 61, 20, 44
        };
        static const int PILN[24] = {
            10, 7, 11, 17, 18, 3, 5, 16, 8, 21, 24, 4,
            15, 23, 19, 13, 12, 2, 20, 14, 22, 9, 6, 1
        };

        uint64_t t;
        uint64_t A[25];
        memcpy(A, state_, sizeof(A));

        for (int round = 0; round < 24; round++) {
            uint64_t C[5];
            for (int i = 0; i < 5; i++)
                C[i] = A[i] ^ A[i+5] ^ A[i+10] ^ A[i+15] ^ A[i+20];
            uint64_t D[5];
            for (int i = 0; i < 5; i++)
                D[i] = C[(i+4)%5] ^ rotl(C[(i+1)%5], 1);
            for (int i = 0; i < 25; i++)
                A[i] ^= D[i%5];

            t = A[1];
            for (int i = 0; i < 24; i++) {
                int j = PILN[i];
                uint64_t tmp = A[j];
                A[j] = rotl(t, ROTC[i]);
                t = tmp;
            }

            for (int j = 0; j < 25; j += 5) {
                uint64_t a0=A[j], a1=A[j+1], a2=A[j+2], a3=A[j+3], a4=A[j+4];
                A[j]   = a0 ^ (~a1 & a2);
                A[j+1] = a1 ^ (~a2 & a3);
                A[j+2] = a2 ^ (~a3 & a4);
                A[j+3] = a3 ^ (~a4 & a0);
                A[j+4] = a4 ^ (~a0 & a1);
            }

            A[0] ^= RC[round];
        }
        memcpy(state_, A, sizeof(A));
    }

    static uint64_t rotl(uint64_t x, int n) {
        return (x << n) | (x >> (64 - n));
    }
};

} // namespace wallet
} // namespace tiger

#endif // TIGERWALLET_KECCAK256_H
