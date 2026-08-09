/**
 * TigerWallet Desktop - BIP-32 HD Wallet Implementation
 *
 * Real BIP-32 hierarchical deterministic key derivation:
 * - Master key: HMAC-SHA512("Bitcoin seed", seed) → IL=private, IR=chaincode
 * - CKDpriv: HMAC-SHA512(chaincode, data) → IL+parent_key mod n = child_key
 * - Uses OpenSSL secp256k1 EC group for point operations and modular arithmetic
 */

#ifndef TIGERWALLET_BIP32_H
#define TIGERWALLET_BIP32_H

#include <string>
#include <vector>
#include <cstdint>
#include <optional>

namespace tiger {
namespace wallet {

struct HDKey {
    std::vector<uint8_t> private_key;  // 32 bytes
    std::vector<uint8_t> chain_code;   // 32 bytes
};

class BIP32 {
public:
    // Derive master key from a BIP-39 seed (64 bytes from mnemonicToSeed).
    static HDKey fromSeed(const std::vector<uint8_t>& seed);

    // Derive a child key from a parent key at the given index.
    // If index >= 0x80000000, it's a hardened child.
    static HDKey deriveChild(const HDKey& parent, uint32_t index);

    // Derive a key at a BIP-32 path like "m/44'/60'/0'/0/0".
    static HDKey derivePath(const HDKey& master, const std::string& path);

    // Parse a derivation path into a list of indices.
    // Hardened indices use ' suffix (e.g. "m/44'/60'/0'/0/0").
    static std::vector<uint32_t> parsePath(const std::string& path);

    // Get the secp256k1 public key (uncompressed, 65 bytes) from a private key.
    static std::vector<uint8_t> publicKey(const std::vector<uint8_t>& privateKey);

    // Get compressed public key (33 bytes) from a private key.
    static std::vector<uint8_t> compressedPublicKey(const std::vector<uint8_t>& privateKey);

    // Derive an EVM address from a private key: keccak256(pubkey)[12:32].
    static std::string evmAddress(const std::vector<uint8_t>& privateKey);

    // HMAC-SHA512 (key, data) → 64 bytes.
    static std::vector<uint8_t> hmacSha512(const std::vector<uint8_t>& key,
                                            const std::vector<uint8_t>& data);

    // The secp256k1 curve order n (as big-endian bytes).
    static std::vector<uint8_t> curveOrder();

    // Add two scalaries mod n (for CKDpriv: child_key = (IL + parent_key) mod n).
    static std::vector<uint8_t> modAdd(const std::vector<uint8_t>& a,
                                        const std::vector<uint8_t>& b);

private:
    static constexpr uint32_t HARDENED_OFFSET = 0x80000000;
};

} // namespace wallet
} // namespace tiger

#endif // TIGERWALLET_BIP32_H
