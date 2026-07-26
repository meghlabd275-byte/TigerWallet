/**
 * Keccak-256 hash implementation
 * Optimized for ultra-low latency
 */

#ifndef KECCAK_HPP
#define KECCAK_HPP

#include <array>
#include <cstdint>
#include <cstring>

namespace tiger {

constexpr size_t kKeccakRate = 136;  // 1600 - 512 = 1088 bits for Keccak-256
constexpr size_t kKeccakCapacity = 256;
constexpr size_t kKeccakOutput = 32;

constexpr uint64_t keccak_round_constants[24] = {
    0x0000000000000001ULL, 0x0000000000008082ULL, 0x800000000000808aULL,
    0x8000000080008000ULL, 0x000000000000808bULL, 0x0000000080000001ULL,
    0x8000000080008081ULL, 0x8000000000008009ULL, 0x000000000000008aULL,
    0x0000000000000088ULL, 0x0000000080008009ULL, 0x000000008000000aULL,
    0x000000008000808bULL, 0x800000000000008bULL, 0x8000000000008089ULL,
    0x8000000000008003ULL, 0x8000000000008002ULL, 0x8000000000000080ULL,
    0x000000000000800aULL, 0x800000008000000aULL, 0x8000000080008081ULL,
    0x8000000000008080ULL, 0x0000000080000001ULL, 0x8000000080008008ULL
};

constexpr size_t keccak_rotation_offsets[5][5] = {
    {0, 36, 3, 41, 18},
    {1, 44, 10, 45, 2},
    {62, 6, 43, 15, 61},
    {28, 55, 25, 21, 56},
    {27, 20, 39, 8, 14}
};

inline size_t rotate_left(size_t x, size_t n) {
    return (x << n) | (x >> (64 - n));
}

inline void keccak_f(uint64_t* state) {
    for (size_t round = 0; round < 24; round++) {
        // Theta
        uint64_t c[5];
        for (size_t x = 0; x < 5; x++) {
            c[x] = state[x] ^ state[x + 5] ^ state[x + 10] ^ 
                   state[x + 15] ^ state[x + 20];
        }
        
        uint64_t d;
        for (size_t x = 0; x < 5; x++) {
            d = c[(x + 4) % 5] ^ rotate_left(c[(x + 1) % 5], 1);
            for (size_t y = 0; y < 5; y++) {
                state[x + 5 * y] ^= d;
            }
        }
        
        // Rho and Pi
        uint64_t b[25];
        for (size_t x = 0; x < 5; x++) {
            for (size_t y = 0; y < 5; y++) {
                size_t idx = x + 5 * y;
                b[y + 5 * ((2 * x + 3 * y) % 5)] = rotate_left(state[idx], 
                    keccak_rotation_offsets[y][x]);
            }
        }
        
        // Chi
        for (size_t x = 0; x < 5; x++) {
            for (size_t y = 0; y < 5; y++) {
                state[x + 5 * y] = b[x + 5 * y] ^ 
                    ((~b[(x + 1) % 5 + 5 * y]) & b[(x + 2) % 5 + 5 * y]);
            }
        }
        
        // Iota
        state[0] ^= keccak_round_constants[round];
    }
}

inline void keccak256(uint8_t* output, const uint8_t* input, size_t input_len) {
    uint64_t state[25] = {0};
    
    // Absorb
    size_t rate = kKeccakRate;
    size_t offset = 0;
    
    while (input_len > 0) {
        size_t block_size = (input_len < rate) ? input_len : rate;
        
        for (size_t i = 0; i < block_size; i++) {
            state[i / 8] ^= static_cast<uint64_t>(input[offset + i]) << (8 * (i % 8));
        }
        
        offset += block_size;
        input_len -= block_size;
        
        if (block_size == rate || input_len == 0) {
            keccak_f(state);
            rate = kKeccakRate;
        }
    }
    
    // Squeeze
    state[rate / 8] ^= 0x01;
    state[kKeccakRate / 8 - 1] ^= 0x80;
    keccak_f(state);
    
    // Output
    for (size_t i = 0; i < kKeccakOutput; i++) {
        output[i] = (state[i / 8] >> (8 * (i % 8))) & 0xFF;
    }
}

}  // namespace tiger

#endif  // KECCAK_HPP
