// Core cryptographic primitives for the wallet.
//
// Keccak-256 is implemented here because Ethereum uses the original Keccak
// padding (0x01), not the FIPS-202 SHA3 padding (0x06), and OpenSSL's EVP
// interface only exposes the latter. Everything else uses OpenSSL: secp256k1
// for public key creation and recovery, and its digest and KDF routines.

#ifndef TIGERWALLET_CORE_CRYPTO_HPP
#define TIGERWALLET_CORE_CRYPTO_HPP

#include <array>
#include <cstdint>
#include <string>
#include <vector>

namespace tigerwallet {
namespace crypto {

// Ethereum's Keccak-256 (original padding, not SHA3-256).
std::array<uint8_t, 32> Keccak256(const uint8_t* data, size_t length);
std::array<uint8_t, 32> Keccak256(const std::vector<uint8_t>& data);

std::array<uint8_t, 32> Sha256(const uint8_t* data, size_t length);
std::array<uint8_t, 32> Sha256(const std::vector<uint8_t>& data);

std::string ToHex(const uint8_t* data, size_t length);
std::string ToHex(const std::vector<uint8_t>& data);

template <size_t N>
std::string ToHex(const std::array<uint8_t, N>& data) {
    return ToHex(data.data(), data.size());
}

// Decodes hex with or without a 0x prefix. Returns false on odd length or a
// non-hex character; `out` is only written on success.
bool FromHex(const std::string& hex, std::vector<uint8_t>& out);
std::vector<uint8_t> FromHex(const std::string& hex);

bool IsValidEvmAddress(const std::string& address);
std::string ToChecksumAddress(const std::string& address);
bool VerifyChecksumAddress(const std::string& address);

// Accepts a 64-byte public key, or 65 bytes with the 0x04 uncompressed prefix.
// Throws std::invalid_argument for any other length.
std::array<uint8_t, 20> PublicKeyToAddress(const uint8_t* public_key, size_t length);
std::string AddressToString(const std::array<uint8_t, 20>& address);

bool PrivateKeyToAddress(const std::array<uint8_t, 32>& private_key,
                         std::array<uint8_t, 20>& address);

// ECDSA over secp256k1. Returns an empty vector on failure. `recovery_id` is
// the raw 0..3 value; callers that build an Ethereum signature add 27.
std::vector<uint8_t> SignHash(const std::array<uint8_t, 32>& digest,
                              const std::array<uint8_t, 32>& private_key,
                              uint8_t& recovery_id);

bool RecoverAddress(const std::array<uint8_t, 32>& digest,
                    const std::vector<uint8_t>& signature,
                    uint8_t recovery_id,
                    std::array<uint8_t, 20>& address);

std::array<uint8_t, 64> HmacSha512(const uint8_t* key, size_t key_len,
                                   const uint8_t* data, size_t data_len);

std::vector<uint8_t> Pbkdf2HmacSha512(const std::string& password,
                                      const std::vector<uint8_t>& salt,
                                      uint32_t iterations,
                                      size_t key_length);

// Throws std::runtime_error if the OS random source fails.
std::vector<uint8_t> SecureRandom(size_t length);

}  // namespace crypto
}  // namespace tigerwallet

#endif  // TIGERWALLET_CORE_CRYPTO_HPP
