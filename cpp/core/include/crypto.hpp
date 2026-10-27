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
#include <optional>
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

// Uncompressed public key without the 0x04 prefix (64 bytes). Returns an empty
// vector if the key is not a valid secp256k1 scalar.
std::vector<uint8_t> PrivateKeyToPublicKey(const std::array<uint8_t, 32>& private_key);

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

// A JSON-RPC quantity is a 0x-prefixed hexadecimal integer with no leading
// zeros ("0x0" for zero). Validates the shape and that the value fits in
// `max_bytes`. Needed because balances and gas prices are 256-bit and must not
// be range-checked as 64-bit values.
bool IsValidHexQuantity(const std::string& text, size_t max_bytes = 32);

// Converts a JSON-RPC quantity to a decimal string, for display and for the
// decimal fields on Transaction. Returns an empty string if `text` is not a
// valid quantity within `max_bytes`.
std::string HexQuantityToDecimal(const std::string& text, size_t max_bytes = 32);

// AES-256-GCM. The output layout is
//   salt(16) || iterations(4, big-endian) || iv(12) || ciphertext || tag(16)
// The key is derived from `password` with PBKDF2-HMAC-SHA512 over the salt.
// The iteration count travels in the blob so that decryption does not have to
// assume the default, and is range-checked on the way back in.
// Decryption returns std::nullopt when authentication fails, which is what
// makes a tampered or wrong-password blob detectable rather than producing
// garbage plaintext.
std::vector<uint8_t> AesGcmEncrypt(const std::string& password,
                                   const uint8_t* plaintext, size_t plaintext_len,
                                   uint32_t iterations = 210000);

std::optional<std::vector<uint8_t>> AesGcmDecrypt(const std::string& password,
                                                  const uint8_t* blob, size_t blob_len);

// ---------------------------------------------------------------------------
// Ed25519 (RFC 8032)
// ---------------------------------------------------------------------------
// Used by the non-EVM chains: Solana, Cardano, Polkadot, Stellar, Near, Sui,
// Aptos, TON, Algorand and others all sign with Ed25519. The 32-byte private
// key is the seed; OpenSSL derives the scalar and the nonce from it, so the
// seed is the only secret that has to be stored.

// Derives the 32-byte public key from a 32-byte seed.
bool Ed25519PublicKeyFromSeed(const std::array<uint8_t, 32>& seed,
                              std::array<uint8_t, 32>& public_key);

// Signs a message. The signature is 64 bytes.
bool Ed25519Sign(const std::array<uint8_t, 32>& seed, const uint8_t* message,
                 size_t message_length, std::array<uint8_t, 64>& signature);

// Verifies a signature. Returns false for a malformed key or signature.
bool Ed25519Verify(const std::array<uint8_t, 32>& public_key, const uint8_t* message,
                   size_t message_length, const std::array<uint8_t, 64>& signature);

// ---------------------------------------------------------------------------
// Base32 (RFC 4648)
// ---------------------------------------------------------------------------
// Solana addresses and Stellar and Algorand account identifiers are Base32, and
// Solana's transaction wire format carries a Base58-encoded blockhash.

std::string Base32Encode(const uint8_t* data, size_t length, bool pad = true);
std::string Base32Encode(const std::vector<uint8_t>& data, bool pad = true);

// Accepts padded or unpadded input and is case-insensitive on the way in, since
// addresses are written both ways. Returns false on an invalid character.
bool Base32Decode(const std::string& text, std::vector<uint8_t>& out);

}  // namespace crypto
}  // namespace tigerwallet

#endif  // TIGERWALLET_CORE_CRYPTO_HPP
