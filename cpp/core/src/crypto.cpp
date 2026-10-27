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

std::vector<uint8_t> PrivateKeyToPublicKey(const std::array<uint8_t, 32>& private_key) {
    secp256k1_context* ctx =
        secp256k1_context_create(SECP256K1_CONTEXT_SIGN | SECP256K1_CONTEXT_VERIFY);
    if (ctx == nullptr) return {};

    secp256k1_pubkey pubkey;
    std::vector<uint8_t> out;
    if (secp256k1_ec_pubkey_create(ctx, &pubkey, private_key.data()) == 1) {
        uint8_t serialized[65];
        size_t serialized_len = sizeof(serialized);
        if (secp256k1_ec_pubkey_serialize(ctx, serialized, &serialized_len, &pubkey,
                                          SECP256K1_EC_UNCOMPRESSED) == 1 &&
            serialized_len == 65) {
            out.assign(serialized + 1, serialized + 65);  // drop the 0x04 prefix
        }
    }

    secp256k1_context_destroy(ctx);
    return out;
}

bool PrivateKeyToAddress(const std::array<uint8_t, 32>& private_key,
                         std::array<uint8_t, 20>& address) {
    const auto public_key = PrivateKeyToPublicKey(private_key);
    if (public_key.size() != 64) return false;
    address = PublicKeyToAddress(public_key.data(), public_key.size());
    return true;
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

bool IsValidHexQuantity(const std::string& text, size_t max_bytes) {
    if (text.size() < 3 || text[0] != '0' || (text[1] != 'x' && text[1] != 'X')) return false;
    const std::string digits = text.substr(2);
    if (digits.empty()) return false;
    for (char c : digits) {
        const bool hex = (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') ||
                         (c >= 'A' && c <= 'F');
        if (!hex) return false;
    }
    // Canonical form has no leading zeros, except for the single digit "0".
    if (digits.size() > 1 && digits[0] == '0') return false;
    if (digits.size() > max_bytes * 2) return false;
    return true;
}

std::string HexQuantityToDecimal(const std::string& text, size_t max_bytes) {
    if (!IsValidHexQuantity(text, max_bytes)) return {};

    std::string digits = text.substr(2);
    // Pad the whole digit string to an even length. Padding each pair
    // separately would treat "100" as bytes 0x10,0x00 (4096) instead of 256.
    if (digits.size() % 2 != 0) digits = "0" + digits;

    // Big-endian bytes, then long division by 10 from the most significant end.
    std::vector<uint8_t> value;
    value.reserve(max_bytes);
    for (size_t i = 0; i < digits.size(); i += 2) {
        value.push_back(static_cast<uint8_t>(std::stoul(digits.substr(i, 2), nullptr, 16)));
    }

    if (value.empty()) return "0";

    std::string decimal;
    while (!value.empty()) {
        unsigned int remainder = 0;
        std::vector<uint8_t> quotient;
        quotient.reserve(value.size());
        for (uint8_t byte : value) {
            const unsigned int current = remainder * 256 + byte;
            const unsigned int q = current / 10;
            remainder = current % 10;
            if (!quotient.empty() || q != 0) quotient.push_back(static_cast<uint8_t>(q));
        }
        decimal.push_back(static_cast<char>('0' + remainder));
        value.swap(quotient);
    }
    return std::string(decimal.rbegin(), decimal.rend());
}

std::vector<uint8_t> AesGcmEncrypt(const std::string& password,
                                   const uint8_t* plaintext, size_t plaintext_len,
                                   uint32_t iterations) {
    constexpr size_t kSaltLen = 16;
    constexpr size_t kIvLen = 12;
    constexpr size_t kTagLen = 16;

    const auto salt = SecureRandom(kSaltLen);
    const auto iv = SecureRandom(kIvLen);
    const auto key = Pbkdf2HmacSha512(password, salt, iterations, 32);

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (ctx == nullptr) throw std::runtime_error("AesGcmEncrypt: context allocation failed");

    std::vector<uint8_t> ciphertext(plaintext_len);
    std::vector<uint8_t> tag(kTagLen);
    int out_len = 0;
    int total = 0;

    bool ok = EVP_EncryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) == 1 &&
              EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, static_cast<int>(kIvLen),
                                  nullptr) == 1 &&
              EVP_EncryptInit_ex(ctx, nullptr, nullptr, key.data(), iv.data()) == 1;

    if (ok && plaintext_len > 0) {
        ok = EVP_EncryptUpdate(ctx, ciphertext.data(), &out_len, plaintext,
                               static_cast<int>(plaintext_len)) == 1;
        total = out_len;
    }
    if (ok) {
        ok = EVP_EncryptFinal_ex(ctx, ciphertext.data() + total, &out_len) == 1;
        total += out_len;
        ciphertext.resize(static_cast<size_t>(total));
    }
    if (ok) {
        ok = EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, static_cast<int>(kTagLen),
                                 tag.data()) == 1;
    }

    EVP_CIPHER_CTX_free(ctx);
    if (!ok) throw std::runtime_error("AesGcmEncrypt: encryption failed");

    // The iteration count is stored in the blob. It has to be: a caller that
    // encrypts with a non-default count would otherwise be unable to decrypt,
    // and silently assuming a constant is a correctness bug waiting to happen.
    std::vector<uint8_t> blob;
    blob.reserve(kSaltLen + 4 + kIvLen + ciphertext.size() + kTagLen);
    blob.insert(blob.end(), salt.begin(), salt.end());
    for (int shift = 24; shift >= 0; shift -= 8) {
        blob.push_back(static_cast<uint8_t>((iterations >> shift) & 0xFF));
    }
    blob.insert(blob.end(), iv.begin(), iv.end());
    blob.insert(blob.end(), ciphertext.begin(), ciphertext.end());
    blob.insert(blob.end(), tag.begin(), tag.end());
    return blob;
}

std::optional<std::vector<uint8_t>> AesGcmDecrypt(const std::string& password,
                                                  const uint8_t* blob, size_t blob_len) {
    constexpr size_t kSaltLen = 16;
    constexpr size_t kIterationsLen = 4;
    constexpr size_t kIvLen = 12;
    constexpr size_t kTagLen = 16;
    // Bounds on the stored iteration count. The lower bound rejects a weakened
    // blob; the upper bound keeps a corrupt value from turning into a very long
    // PBKDF2 run.
    constexpr uint32_t kMinIterations = 1000;
    constexpr uint32_t kMaxIterations = 10000000;

    if (blob_len < kSaltLen + kIterationsLen + kIvLen + kTagLen) return std::nullopt;

    const uint8_t* salt = blob;
    const uint8_t* iterations_bytes = blob + kSaltLen;
    const uint8_t* iv = blob + kSaltLen + kIterationsLen;
    const uint8_t* ciphertext = blob + kSaltLen + kIterationsLen + kIvLen;
    const size_t ciphertext_len = blob_len - kSaltLen - kIterationsLen - kIvLen - kTagLen;
    const uint8_t* tag = blob + blob_len - kTagLen;

    uint32_t iterations = 0;
    for (size_t i = 0; i < kIterationsLen; ++i) {
        iterations = (iterations << 8) | iterations_bytes[i];
    }
    if (iterations < kMinIterations || iterations > kMaxIterations) return std::nullopt;

    const std::vector<uint8_t> salt_vec(salt, salt + kSaltLen);
    const auto key = Pbkdf2HmacSha512(password, salt_vec, iterations, 32);

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (ctx == nullptr) return std::nullopt;

    std::vector<uint8_t> plaintext(ciphertext_len);
    int out_len = 0;
    int total = 0;

    bool ok = EVP_DecryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) == 1 &&
              EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, static_cast<int>(kIvLen),
                                  nullptr) == 1 &&
              EVP_DecryptInit_ex(ctx, nullptr, nullptr, key.data(), iv) == 1;

    if (ok && ciphertext_len > 0) {
        ok = EVP_DecryptUpdate(ctx, plaintext.data(), &out_len, ciphertext,
                               static_cast<int>(ciphertext_len)) == 1;
        total = out_len;
    }
    if (ok) {
        // The tag check happens in DecryptFinal. A wrong password or a modified
        // blob fails here, which is why the result is an optional.
        ok = EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, static_cast<int>(kTagLen),
                                 const_cast<uint8_t*>(tag)) == 1;
    }
    if (ok) {
        ok = EVP_DecryptFinal_ex(ctx, plaintext.data() + total, &out_len) == 1;
        total += out_len;
    }

    EVP_CIPHER_CTX_free(ctx);
    if (!ok) return std::nullopt;

    plaintext.resize(static_cast<size_t>(total));
    return plaintext;
}

// ---------------------------------------------------------------------------
// Ed25519
// ---------------------------------------------------------------------------

bool Ed25519PublicKeyFromSeed(const std::array<uint8_t, 32>& seed,
                              std::array<uint8_t, 32>& public_key) {
    EVP_PKEY* key = EVP_PKEY_new_raw_private_key(EVP_PKEY_ED25519, nullptr, seed.data(),
                                                 seed.size());
    if (key == nullptr) return false;

    size_t length = public_key.size();
    const bool ok = EVP_PKEY_get_raw_public_key(key, public_key.data(), &length) == 1 &&
                    length == public_key.size();
    EVP_PKEY_free(key);
    return ok;
}

bool Ed25519Sign(const std::array<uint8_t, 32>& seed, const uint8_t* message,
                 size_t message_length, std::array<uint8_t, 64>& signature) {
    EVP_PKEY* key = EVP_PKEY_new_raw_private_key(EVP_PKEY_ED25519, nullptr, seed.data(),
                                                 seed.size());
    if (key == nullptr) return false;

    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    // Ed25519 is a one-shot signature: EVP_DigestSign takes the whole message
    // and the digest is not selected, because Ed25519 hashes internally.
    const bool ok = ctx != nullptr && EVP_DigestSignInit(ctx, nullptr, nullptr, nullptr, key) == 1;

    size_t length = signature.size();
    const bool signed_ok = ok && EVP_DigestSign(ctx, signature.data(), &length, message,
                                                message_length) == 1 &&
                           length == signature.size();

    EVP_MD_CTX_free(ctx);
    EVP_PKEY_free(key);
    return signed_ok;
}

bool Ed25519Verify(const std::array<uint8_t, 32>& public_key, const uint8_t* message,
                   size_t message_length, const std::array<uint8_t, 64>& signature) {
    EVP_PKEY* key = EVP_PKEY_new_raw_public_key(EVP_PKEY_ED25519, nullptr,
                                                public_key.data(), public_key.size());
    if (key == nullptr) return false;

    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    const bool ok = ctx != nullptr && EVP_DigestVerifyInit(ctx, nullptr, nullptr, nullptr, key) == 1;
    const bool verified = ok && EVP_DigestVerify(ctx, signature.data(), signature.size(),
                                                 message, message_length) == 1;

    EVP_MD_CTX_free(ctx);
    EVP_PKEY_free(key);
    return verified;
}

// ---------------------------------------------------------------------------
// Base32 (RFC 4648)
// ---------------------------------------------------------------------------

std::string Base32Encode(const uint8_t* data, size_t length, bool pad) {
    static const char* kAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

    std::string out;
    out.reserve((length + 4) / 5 * 8);

    size_t i = 0;
    while (i < length) {
        // Accumulate up to five bytes (40 bits) into eight 5-bit groups.
        uint64_t buffer = 0;
        int bits = 0;
        for (int j = 0; j < 5 && i < length; ++j, ++i) {
            buffer = (buffer << 8) | data[i];
            bits += 8;
        }
        while (bits > 0) {
            // Take the top five bits still in the buffer.
            const int shift = bits - 5;
            const uint8_t index =
                shift >= 0 ? static_cast<uint8_t>((buffer >> shift) & 0x1F)
                           : static_cast<uint8_t>((buffer << (-shift)) & 0x1F);
            out.push_back(kAlphabet[index]);
            bits -= 5;
        }
    }

    if (pad) {
        while (out.size() % 8 != 0) out.push_back('=');
    }
    return out;
}

std::string Base32Encode(const std::vector<uint8_t>& data, bool pad) {
    return Base32Encode(data.data(), data.size(), pad);
}

bool Base32Decode(const std::string& text, std::vector<uint8_t>& out) {
    // Strip padding; the decoder does not need it because the byte count
    // follows from the number of data characters.
    std::string clean;
    clean.reserve(text.size());
    for (char c : text) {
        if (c == '=') continue;
        if (c == ' ' || c == '\n' || c == '\t' || c == '\r') continue;
        clean.push_back(c);
    }
    if (clean.empty()) {
        out.clear();
        return true;
    }

    // A Base32 group is eight characters carrying five bytes. Any other count
    // is malformed, so the trailing group must be one of the five valid sizes.
    const size_t remainder = clean.size() % 8;
    if (remainder == 1 || remainder == 3 || remainder == 6) return false;

    std::vector<uint8_t> decoded;
    uint64_t buffer = 0;
    int bits = 0;
    for (char raw : clean) {
        // Addresses are written in both cases, so the decoder accepts either.
        const char c = (raw >= 'a' && raw <= 'z') ? static_cast<char>(raw - 'a' + 'A') : raw;
        int value;
        if (c >= 'A' && c <= 'Z') {
            value = c - 'A';
        } else if (c >= '2' && c <= '7') {
            value = c - '2' + 26;
        } else {
            return false;  // not in the alphabet
        }

        buffer = (buffer << 5) | static_cast<uint64_t>(value);
        bits += 5;
        if (bits >= 8) {
            bits -= 8;
            decoded.push_back(static_cast<uint8_t>((buffer >> bits) & 0xFF));
        }
    }

    // Any leftover bits must be zero, otherwise the input encoded data that was
    // truncated rather than padded.
    if (bits > 0 && (buffer & ((1ull << bits) - 1)) != 0) return false;

    out = std::move(decoded);
    return true;
}

}  // namespace crypto
}  // namespace tigerwallet
