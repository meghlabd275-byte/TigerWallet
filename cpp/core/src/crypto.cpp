// Core cryptographic primitives for the wallet. See include/crypto.hpp.

#include "crypto.hpp"

#include <algorithm>
#include <cstring>
#include <stdexcept>

#include <openssl/evp.h>
#include <openssl/hmac.h>
#include <openssl/rand.h>
#include <secp256k1.h>
#include <secp256k1_recovery.h>

namespace tigerwallet {
namespace crypto {
namespace {

constexpr uint64_t kKeccakRoundConstants[24] = {
    0x0000000000000001ULL, 0x0000000000008082ULL, 0x800000000000808aULL,
    0x8000000080008000ULL, 0x000000000000808bULL, 0x0000000080000001ULL,
    0x8000000080008081ULL, 0x8000000000008009ULL, 0x000000000000008aULL,
    0x0000000000000088ULL, 0x0000000080008009ULL, 0x000000008000000aULL,
    0x000000008000808bULL, 0x800000000000008bULL, 0x8000000000008089ULL,
    0x8000000000008003ULL, 0x8000000000008002ULL, 0x8000000000000080ULL,
    0x000000000000800aULL, 0x800000008000000aULL, 0x8000000080008081ULL,
    0x8000000000008080ULL, 0x0000000080000001ULL, 0x8000000080008008ULL};

// rho rotation offsets, indexed [x][y].
constexpr int kKeccakRho[5][5] = {
    {0, 36, 3, 41, 18},
    {1, 44, 10, 45, 2},
    {62, 6, 43, 15, 61},
    {28, 55, 25, 21, 56},
    {27, 20, 39, 8, 14}};

inline uint64_t Rotl64(uint64_t x, int n) {
    return (x << n) | (x >> (64 - n));
}

void KeccakF1600(uint64_t a[25]) {
    // The lane at (x, y) is a[x + 5*y].
    for (int round = 0; round < 24; ++round) {
        // theta
        uint64_t c[5];
        for (int x = 0; x < 5; ++x) {
            c[x] = a[x] ^ a[x + 5] ^ a[x + 10] ^ a[x + 15] ^ a[x + 20];
        }
        uint64_t d[5];
        for (int x = 0; x < 5; ++x) {
            d[x] = c[(x + 4) % 5] ^ Rotl64(c[(x + 1) % 5], 1);
        }
        for (int x = 0; x < 5; ++x) {
            for (int y = 0; y < 5; ++y) {
                a[x + 5 * y] ^= d[x];
            }
        }

        // rho and pi: b[y][(2x + 3y) mod 5] = rotl(a[x][y], rho[x][y])
        uint64_t b[25];
        for (int x = 0; x < 5; ++x) {
            for (int y = 0; y < 5; ++y) {
                b[y + 5 * ((2 * x + 3 * y) % 5)] = Rotl64(a[x + 5 * y], kKeccakRho[x][y]);
            }
        }

        // chi
        for (int x = 0; x < 5; ++x) {
            for (int y = 0; y < 5; ++y) {
                a[x + 5 * y] = b[x + 5 * y] ^
                               ((~b[(x + 1) % 5 + 5 * y]) & b[(x + 2) % 5 + 5 * y]);
            }
        }

        // iota
        a[0] ^= kKeccakRoundConstants[round];
    }
}

inline uint64_t LoadLe64(const uint8_t* p) {
    uint64_t v = 0;
    for (int i = 0; i < 8; ++i) v |= static_cast<uint64_t>(p[i]) << (8 * i);
    return v;
}

inline void StoreLe64(uint8_t* p, uint64_t v) {
    for (int i = 0; i < 8; ++i) p[i] = static_cast<uint8_t>(v >> (8 * i));
}

}  // namespace

std::array<uint8_t, 32> Keccak256(const uint8_t* data, size_t length) {
    constexpr size_t kRate = 136;  // 1088 bits for Keccak-256

    uint64_t state[25] = {0};
    uint8_t block[kRate];

    size_t offset = 0;
    while (length - offset >= kRate) {
        for (size_t i = 0; i < kRate / 8; ++i) {
            state[i] ^= LoadLe64(data + offset + i * 8);
        }
        KeccakF1600(state);
        offset += kRate;
    }

    // Final partial block with the original Keccak padding: 0x01 ... 0x80.
    const size_t remaining = length - offset;
    std::memset(block, 0, sizeof(block));
    if (remaining > 0) std::memcpy(block, data + offset, remaining);
    block[remaining] = 0x01;
    block[kRate - 1] |= 0x80;

    for (size_t i = 0; i < kRate / 8; ++i) {
        state[i] ^= LoadLe64(block + i * 8);
    }
    KeccakF1600(state);

    std::array<uint8_t, 32> digest{};
    for (int i = 0; i < 4; ++i) {
        StoreLe64(digest.data() + i * 8, state[i]);
    }
    return digest;
}

std::array<uint8_t, 32> Keccak256(const std::vector<uint8_t>& data) {
    return Keccak256(data.data(), data.size());
}

std::array<uint8_t, 32> Sha256(const uint8_t* data, size_t length) {
    std::array<uint8_t, 32> out{};
    unsigned int out_len = 0;
    if (EVP_Digest(data, length, out.data(), &out_len, EVP_sha256(), nullptr) != 1 ||
        out_len != out.size()) {
        throw std::runtime_error("Sha256: EVP_Digest failed");
    }
    return out;
}

std::array<uint8_t, 32> Sha256(const std::vector<uint8_t>& data) {
    return Sha256(data.data(), data.size());
}

std::string ToHex(const uint8_t* data, size_t length) {
    static const char* kDigits = "0123456789abcdef";
    std::string out;
    out.reserve(length * 2);
    for (size_t i = 0; i < length; ++i) {
        out.push_back(kDigits[data[i] >> 4]);
        out.push_back(kDigits[data[i] & 0x0F]);
    }
    return out;
}

std::string ToHex(const std::vector<uint8_t>& data) {
    return ToHex(data.data(), data.size());
}

bool FromHex(const std::string& hex, std::vector<uint8_t>& out) {
    std::string clean;
    clean.reserve(hex.size());
    for (char c : hex) {
        if (c == ' ' || c == '\n' || c == '\t' || c == '\r') continue;
        clean.push_back(c);
    }
    if (clean.size() >= 2 && clean[0] == '0' && (clean[1] == 'x' || clean[1] == 'X')) {
        clean.erase(0, 2);
    }
    if (clean.size() % 2 != 0) return false;

    auto nibble = [](char c) -> int {
        if (c >= '0' && c <= '9') return c - '0';
        if (c >= 'a' && c <= 'f') return c - 'a' + 10;
        if (c >= 'A' && c <= 'F') return c - 'A' + 10;
        return -1;
    };

    std::vector<uint8_t> decoded;
    decoded.reserve(clean.size() / 2);
    for (size_t i = 0; i < clean.size(); i += 2) {
        const int hi = nibble(clean[i]);
        const int lo = nibble(clean[i + 1]);
        if (hi < 0 || lo < 0) return false;
        decoded.push_back(static_cast<uint8_t>((hi << 4) | lo));
    }

    out = std::move(decoded);
    return true;
}

std::vector<uint8_t> FromHex(const std::string& hex) {
    std::vector<uint8_t> out;
    if (!FromHex(hex, out)) return {};
    return out;
}

bool IsValidEvmAddress(const std::string& address) {
    if (address.size() != 42) return false;
    if (address[0] != '0' || (address[1] != 'x' && address[1] != 'X')) return false;
    for (size_t i = 2; i < address.size(); ++i) {
        const char c = address[i];
        const bool hex = (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') ||
                         (c >= 'A' && c <= 'F');
        if (!hex) return false;
    }
    return true;
}

std::string ToChecksumAddress(const std::string& address) {
    if (!IsValidEvmAddress(address)) return address;

    // EIP-55: lowercase the address, hash the lowercase hex without the 0x
    // prefix, then uppercase each hex letter whose corresponding hash nibble
    // is 8 or above.
    std::string lower = address.substr(2);
    for (char& c : lower) {
        if (c >= 'A' && c <= 'F') c = static_cast<char>(c - 'A' + 'a');
    }

    const auto hash = Keccak256(reinterpret_cast<const uint8_t*>(lower.data()), lower.size());

    std::string out = "0x";
    out.reserve(42);
    for (size_t i = 0; i < lower.size(); ++i) {
        const char c = lower[i];
        if (c >= 'a' && c <= 'f') {
            const uint8_t nibble = (i % 2 == 0) ? (hash[i / 2] >> 4) : (hash[i / 2] & 0x0F);
            out.push_back(nibble >= 8 ? static_cast<char>(c - 'a' + 'A') : c);
        } else {
            out.push_back(c);
        }
    }
    return out;
}

bool VerifyChecksumAddress(const std::string& address) {
    if (!IsValidEvmAddress(address)) return false;
    bool has_upper = false;
    bool has_lower = false;
    for (size_t i = 2; i < address.size(); ++i) {
        const char c = address[i];
        if (c >= 'A' && c <= 'F') has_upper = true;
        if (c >= 'a' && c <= 'f') has_lower = true;
    }
    // An all-lowercase or all-uppercase address carries no checksum information
    // and is accepted by convention.
    if (!has_upper || !has_lower) return true;
    return ToChecksumAddress(address) == address;
}

std::array<uint8_t, 20> PublicKeyToAddress(const uint8_t* public_key, size_t length) {
    // Accept the 64-byte uncompressed point, or 65 bytes with the 0x04 prefix.
    const uint8_t* point = public_key;
    if (length == 65 && public_key[0] == 0x04) {
        point = public_key + 1;
        length = 64;
    }
    if (length != 64) {
        throw std::invalid_argument("PublicKeyToAddress: expected a 64-byte public key");
    }

    const auto hash = Keccak256(point, 64);
    std::array<uint8_t, 20> address{};
    std::copy(hash.end() - 20, hash.end(), address.begin());
    return address;
}

std::string AddressToString(const std::array<uint8_t, 20>& address) {
    return ToChecksumAddress("0x" + ToHex(address.data(), address.size()));
}

bool PrivateKeyToAddress(const std::array<uint8_t, 32>& private_key,
                         std::array<uint8_t, 20>& address) {
    secp256k1_context* ctx =
        secp256k1_context_create(SECP256K1_CONTEXT_SIGN | SECP256K1_CONTEXT_VERIFY);
    if (ctx == nullptr) return false;

    secp256k1_pubkey pubkey;
    bool ok = secp256k1_ec_pubkey_create(ctx, &pubkey, private_key.data()) == 1;
    if (ok) {
        uint8_t serialized[65];
        size_t serialized_len = sizeof(serialized);
        ok = secp256k1_ec_pubkey_serialize(ctx, serialized, &serialized_len, &pubkey,
                                           SECP256K1_EC_UNCOMPRESSED) == 1;
        if (ok) address = PublicKeyToAddress(serialized, serialized_len);
    }

    secp256k1_context_destroy(ctx);
    return ok;
}

std::vector<uint8_t> SignHash(const std::array<uint8_t, 32>& digest,
                              const std::array<uint8_t, 32>& private_key,
                              uint8_t& recovery_id) {
    secp256k1_context* ctx = secp256k1_context_create(SECP256K1_CONTEXT_SIGN);
    if (ctx == nullptr) return {};

    secp256k1_ecdsa_recoverable_signature signature;
    std::vector<uint8_t> out;
    // A null nonce function selects libsecp256k1's RFC 6979 deterministic nonce.
    if (secp256k1_ecdsa_sign_recoverable(ctx, &signature, digest.data(), private_key.data(),
                                         nullptr, nullptr) == 1) {
        uint8_t compact[64];
        int recid = 0;
        secp256k1_ecdsa_recoverable_signature_serialize_compact(ctx, compact, &recid, &signature);
        recovery_id = static_cast<uint8_t>(recid);
        out.assign(compact, compact + 64);
    }

    secp256k1_context_destroy(ctx);
    return out;
}

bool RecoverAddress(const std::array<uint8_t, 32>& digest,
                    const std::vector<uint8_t>& signature,
                    uint8_t recovery_id,
                    std::array<uint8_t, 20>& address) {
    if (signature.size() != 64 || recovery_id > 3) return false;

    secp256k1_context* ctx = secp256k1_context_create(SECP256K1_CONTEXT_VERIFY);
    if (ctx == nullptr) return false;

    secp256k1_ecdsa_recoverable_signature sig;
    bool ok = secp256k1_ecdsa_recoverable_signature_parse_compact(
                  ctx, &sig, signature.data(), static_cast<int>(recovery_id)) == 1;
    if (ok) {
        secp256k1_pubkey pubkey;
        ok = secp256k1_ecdsa_recover(ctx, &pubkey, &sig, digest.data()) == 1;
        if (ok) {
            uint8_t serialized[65];
            size_t serialized_len = sizeof(serialized);
            ok = secp256k1_ec_pubkey_serialize(ctx, serialized, &serialized_len, &pubkey,
                                               SECP256K1_EC_UNCOMPRESSED) == 1;
            if (ok) address = PublicKeyToAddress(serialized, serialized_len);
        }
    }

    secp256k1_context_destroy(ctx);
    return ok;
}

std::array<uint8_t, 64> HmacSha512(const uint8_t* key, size_t key_len,
                                   const uint8_t* data, size_t data_len) {
    std::array<uint8_t, 64> out{};
    unsigned int out_len = 0;
    if (HMAC(EVP_sha512(), key, static_cast<int>(key_len), data, data_len, out.data(),
             &out_len) == nullptr ||
        out_len != out.size()) {
        throw std::runtime_error("HmacSha512: HMAC failed");
    }
    return out;
}

std::vector<uint8_t> Pbkdf2HmacSha512(const std::string& password,
                                      const std::vector<uint8_t>& salt,
                                      uint32_t iterations,
                                      size_t key_length) {
    std::vector<uint8_t> out(key_length);
    if (PKCS5_PBKDF2_HMAC(password.data(), static_cast<int>(password.size()), salt.data(),
                          static_cast<int>(salt.size()), static_cast<int>(iterations),
                          EVP_sha512(), static_cast<int>(key_length), out.data()) != 1) {
        throw std::runtime_error("Pbkdf2HmacSha512: derivation failed");
    }
    return out;
}

std::vector<uint8_t> SecureRandom(size_t length) {
    std::vector<uint8_t> out(length);
    if (length > 0 && RAND_bytes(out.data(), static_cast<int>(length)) != 1) {
        throw std::runtime_error("SecureRandom: the OS random source failed");
    }
    return out;
}

}  // namespace crypto
}  // namespace tigerwallet
