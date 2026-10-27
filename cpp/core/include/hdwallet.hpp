// Hierarchical deterministic wallets: BIP39 mnemonics and BIP32 derivation.
//
// BIP39 turns entropy into a mnemonic and a mnemonic into a 64-byte seed.
// BIP32 turns that seed into a tree of keys. Both are implemented here so the
// wallet does not depend on a third-party wallet library.
//
// Every entry point that can fail returns std::nullopt or false rather than
// guessing: a mnemonic with a bad checksum, an unknown word, or a wrong word
// count is rejected, because accepting one silently would derive a different
// wallet than the user expects.

#ifndef TIGERWALLET_HDWALLET_HPP
#define TIGERWALLET_HDWALLET_HPP

#include <array>
#include <cstdint>
#include <optional>
#include <string>
#include <vector>

namespace tigerwallet {
namespace hd {

// ---------------------------------------------------------------------------
// Base58 and Base58Check
// ---------------------------------------------------------------------------

std::string Base58Encode(const std::vector<uint8_t>& data);

// Returns false on an invalid character or a length that decodes to an
// implausible size.
bool Base58Decode(const std::string& text, std::vector<uint8_t>& out);

// Appends the first four bytes of double-SHA256.
std::string Base58CheckEncode(const std::vector<uint8_t>& payload);

// Verifies and strips the four-byte checksum.
bool Base58CheckDecode(const std::string& text, std::vector<uint8_t>& payload);

// ---------------------------------------------------------------------------
// RIPEMD-160
// ---------------------------------------------------------------------------

std::array<uint8_t, 20> Ripemd160(const uint8_t* data, size_t length);
std::array<uint8_t, 20> Ripemd160(const std::vector<uint8_t>& data);

// Hash160: RIPEMD-160 of SHA-256. This is what Bitcoin addresses use.
std::array<uint8_t, 20> Hash160(const std::vector<uint8_t>& data);

// ---------------------------------------------------------------------------
// BIP39
// ---------------------------------------------------------------------------

// Valid word counts for a BIP39 mnemonic.
bool IsValidWordCount(size_t word_count);

// Generates a mnemonic of `word_count` words (12, 15, 18, 21 or 24) from the
// operating system's random source.
std::optional<std::string> GenerateMnemonic(size_t word_count = 12);

// Converts entropy to a mnemonic. Entropy must be 16, 20, 24, 28 or 32 bytes.
std::optional<std::string> EntropyToMnemonic(const std::vector<uint8_t>& entropy);

// Converts a mnemonic back to entropy, verifying every word and the checksum.
bool MnemonicToEntropy(const std::string& mnemonic, std::vector<uint8_t>& entropy);

// Verifies the word list and the checksum without returning the entropy.
bool ValidateMnemonic(const std::string& mnemonic);

// PBKDF2-HMAC-SHA512 over the mnemonic with the BIP39 salt ("mnemonic" plus the
// optional passphrase), 2048 iterations, 64-byte output.
std::array<uint8_t, 64> MnemonicToSeed(const std::string& mnemonic,
                                       const std::string& passphrase = "");

// ---------------------------------------------------------------------------
// BIP32
// ---------------------------------------------------------------------------

// A node in the derivation tree.
struct ExtendedKey {
    std::vector<uint8_t> private_key;  // 32 bytes, empty for a public-only node
    std::vector<uint8_t> public_key;   // 33 bytes, compressed
    std::vector<uint8_t> chain_code;   // 32 bytes
    uint32_t depth = 0;
    uint32_t child_number = 0;
    uint32_t parent_fingerprint = 0;
};

// One step of a derivation path.
struct PathElement {
    uint32_t index = 0;
    bool hardened = false;
};

// Parses a path such as "m/44'/60'/0'/0/0". The leading "m" is optional, so
// both absolute ("m/0") and relative ("0") paths are accepted. A trailing
// apostrophe, h or H marks a hardened step, and the returned index already has
// bit 31 set for those, which is the form DeriveChild expects.
bool ParsePath(const std::string& path, std::vector<PathElement>& out);

// Builds a path from BIP44 components: m/purpose'/coin'/account'/change/index.
std::string Bip44Path(uint32_t purpose, uint32_t coin_type, uint32_t account,
                      uint32_t change, uint32_t index);

// The master key for a seed. The seed is the 64 bytes BIP39 produces, but BIP32
// itself accepts 16 to 64 bytes and the published vectors use those lengths, so
// the input is not restricted to 64.
std::optional<ExtendedKey> MasterKeyFromSeed(const std::vector<uint8_t>& seed);
std::optional<ExtendedKey> MasterKeyFromSeed(const std::array<uint8_t, 64>& seed);

// Derives one child. Hardened derivation requires a private key.
//
// `index` may be given either as the bare index (with bit 31 clear) or already
// marked with bit 31, as ParsePath returns it. Passing a marked index with
// `hardened` false is a contradiction and is rejected.
std::optional<ExtendedKey> DeriveChild(const ExtendedKey& parent, uint32_t index,
                                       bool hardened);

// Walks a full path from a master key.
std::optional<ExtendedKey> DerivePath(const ExtendedKey& master,
                                      const std::vector<PathElement>& path);
std::optional<ExtendedKey> DerivePath(const ExtendedKey& master, const std::string& path);

// The 4-byte fingerprint used as a parent identifier: the first four bytes of
// hash160 of the compressed public key.
uint32_t KeyFingerprint(const ExtendedKey& key);

// Serializes to the xprv / xpub textual form, for export and for comparison
// against the published BIP32 test vectors.
std::optional<std::string> SerializeExtendedPrivateKey(const ExtendedKey& key);
std::optional<std::string> SerializeExtendedPublicKey(const ExtendedKey& key);

// The 20-byte Ethereum address for a node's private key.
std::optional<std::array<uint8_t, 20>> EthereumAddress(const ExtendedKey& key);

// The 20-byte Bitcoin hash160 for a node's compressed public key.
std::optional<std::array<uint8_t, 20>> BitcoinHash160(const ExtendedKey& key);

// ---------------------------------------------------------------------------
// BIP44 coin types
// ---------------------------------------------------------------------------

// Registered SLIP-44 coin type for a chain, or std::nullopt if unregistered.
std::optional<uint32_t> CoinTypeForChain(const std::string& symbol);

}  // namespace hd
}  // namespace tigerwallet

#endif  // TIGERWALLET_HDWALLET_HPP
