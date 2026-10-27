// BIP39 and BIP32. See include/hdwallet.hpp.

#include "hdwallet.hpp"

#include "crypto.hpp"

#include <algorithm>
#include <cstring>
#include <map>
#include <stdexcept>
#include <string>

#include <openssl/bn.h>
#include <openssl/ec.h>
#include <openssl/evp.h>
#include <openssl/obj_mac.h>
#include <secp256k1.h>

namespace tigerwallet {
namespace hd {

// Provided by bip39_wordlist.cpp. Declared at namespace scope, not inside the
// anonymous namespace below, so this refers to the same entity that file
// defines.
const char* const* Bip39Wordlist();
size_t Bip39WordlistSize();

namespace {

constexpr const char* kBase58Alphabet =
    "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

// BIP32 serialization versions.
constexpr uint32_t kXprvVersion = 0x0488ADE4;
constexpr uint32_t kXpubVersion = 0x0488B21E;

constexpr size_t kChainCodeLength = 32;
constexpr size_t kPrivateKeyLength = 32;
constexpr size_t kCompressedPublicKeyLength = 33;

std::array<uint8_t, 32> DoubleSha256(const std::vector<uint8_t>& data) {
    const auto first = crypto::Sha256(data);
    return crypto::Sha256(first.data(), first.size());
}

// The secp256k1 group order, as a BIGNUM. The caller owns the result.
BIGNUM* CurveOrder() {
    BIGNUM* order = BN_new();
    if (order == nullptr) return nullptr;
    if (BN_hex2bn(&order,
                  "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141") == 0) {
        BN_free(order);
        return nullptr;
    }
    return order;
}

// True when `value` is in [1, n-1].
bool IsValidScalar(const BIGNUM* value, const BIGNUM* order) {
    return !BN_is_zero(value) && !BN_is_negative(value) && BN_cmp(value, order) < 0;
}

// Serializes an extended key's 78-byte payload and applies Base58Check.
std::string SerializeExtended(const ExtendedKey& key, uint32_t version, bool is_private) {
    std::vector<uint8_t> payload;
    payload.reserve(78);

    payload.push_back(static_cast<uint8_t>((version >> 24) & 0xFF));
    payload.push_back(static_cast<uint8_t>((version >> 16) & 0xFF));
    payload.push_back(static_cast<uint8_t>((version >> 8) & 0xFF));
    payload.push_back(static_cast<uint8_t>(version & 0xFF));

    payload.push_back(static_cast<uint8_t>(key.depth & 0xFF));

    payload.push_back(static_cast<uint8_t>((key.parent_fingerprint >> 24) & 0xFF));
    payload.push_back(static_cast<uint8_t>((key.parent_fingerprint >> 16) & 0xFF));
    payload.push_back(static_cast<uint8_t>((key.parent_fingerprint >> 8) & 0xFF));
    payload.push_back(static_cast<uint8_t>(key.parent_fingerprint & 0xFF));

    payload.push_back(static_cast<uint8_t>((key.child_number >> 24) & 0xFF));
    payload.push_back(static_cast<uint8_t>((key.child_number >> 16) & 0xFF));
    payload.push_back(static_cast<uint8_t>((key.child_number >> 8) & 0xFF));
    payload.push_back(static_cast<uint8_t>(key.child_number & 0xFF));

    payload.insert(payload.end(), key.chain_code.begin(), key.chain_code.end());

    if (is_private) {
        payload.push_back(0x00);
        payload.insert(payload.end(), key.private_key.begin(), key.private_key.end());
    } else {
        payload.insert(payload.end(), key.public_key.begin(), key.public_key.end());
    }

    return Base58CheckEncode(payload);
}

// Converts a word list index bit sequence into entropy, checking the trailing
// checksum bits. Returns false when the checksum does not match.
bool BitsToEntropy(const std::vector<uint8_t>& bits, std::vector<uint8_t>& entropy) {
    if (bits.size() % 33 != 0) return false;
    const size_t entropy_bits = bits.size() - bits.size() / 33;
    const size_t checksum_bits = bits.size() - entropy_bits;

    std::vector<uint8_t> candidate(entropy_bits / 8, 0);
    for (size_t i = 0; i < entropy_bits; ++i) {
        if (bits[i]) candidate[i / 8] |= static_cast<uint8_t>(1u << (7 - (i % 8)));
    }

    const auto digest = crypto::Sha256(candidate);
    for (size_t i = 0; i < checksum_bits; ++i) {
        const bool expected = (digest[i / 8] & (1u << (7 - (i % 8)))) != 0;
        if (bits[entropy_bits + i] != expected) return false;
    }

    entropy = std::move(candidate);
    return true;
}

}  // namespace

// ---------------------------------------------------------------------------
// Base58
// ---------------------------------------------------------------------------

std::string Base58Encode(const std::vector<uint8_t>& data) {
    // Leading zero bytes encode as leading '1' characters.
    size_t zeros = 0;
    while (zeros < data.size() && data[zeros] == 0) ++zeros;

    std::vector<uint8_t> digits;  // base-256 input, little-endian base-58 digits
    for (size_t i = zeros; i < data.size(); ++i) {
        int carry = data[i];
        for (size_t j = 0; j < digits.size(); ++j) {
            carry += 256 * digits[j];
            digits[j] = static_cast<uint8_t>(carry % 58);
            carry /= 58;
        }
        while (carry > 0) {
            digits.push_back(static_cast<uint8_t>(carry % 58));
            carry /= 58;
        }
    }

    std::string out(zeros, '1');
    for (size_t i = digits.size(); i-- > 0;) {
        out.push_back(kBase58Alphabet[digits[i]]);
    }
    return out;
}

bool Base58Decode(const std::string& text, std::vector<uint8_t>& out) {
    if (text.empty()) {
        out.clear();
        return true;
    }
    // The alphabet omits lookalike characters, so the map is exact.
    static const std::map<char, int> values = [] {
        std::map<char, int> m;
        for (int i = 0; i < 58; ++i) m[kBase58Alphabet[i]] = i;
        return m;
    }();

    size_t zeros = 0;
    while (zeros < text.size() && text[zeros] == '1') ++zeros;

    std::vector<uint8_t> bytes;
    for (size_t i = zeros; i < text.size(); ++i) {
        const auto it = values.find(text[i]);
        if (it == values.end()) return false;  // invalid character
        int carry = it->second;
        for (size_t j = 0; j < bytes.size(); ++j) {
            carry += 58 * bytes[j];
            bytes[j] = static_cast<uint8_t>(carry & 0xFF);
            carry >>= 8;
        }
        while (carry > 0) {
            bytes.push_back(static_cast<uint8_t>(carry & 0xFF));
            carry >>= 8;
        }
    }

    out.assign(zeros, 0);
    for (size_t i = bytes.size(); i-- > 0;) {
        out.push_back(bytes[i]);
    }
    return true;
}

std::string Base58CheckEncode(const std::vector<uint8_t>& payload) {
    const auto checksum = DoubleSha256(payload);
    std::vector<uint8_t> with_checksum = payload;
    with_checksum.insert(with_checksum.end(), checksum.begin(), checksum.begin() + 4);
    return Base58Encode(with_checksum);
}

bool Base58CheckDecode(const std::string& text, std::vector<uint8_t>& payload) {
    std::vector<uint8_t> decoded;
    if (!Base58Decode(text, decoded)) return false;
    if (decoded.size() < 5) return false;  // four checksum bytes plus one payload byte

    std::vector<uint8_t> body(decoded.begin(), decoded.end() - 4);
    const auto checksum = DoubleSha256(body);
    for (size_t i = 0; i < 4; ++i) {
        if (decoded[body.size() + i] != checksum[i]) return false;
    }
    payload = std::move(body);
    return true;
}

// ---------------------------------------------------------------------------
// RIPEMD-160
// ---------------------------------------------------------------------------

std::array<uint8_t, 20> Ripemd160(const uint8_t* data, size_t length) {
    std::array<uint8_t, 20> out{};

    // RIPEMD-160 is in the default provider on OpenSSL 3.x, but it is fetched
    // by name so that a build without it fails here rather than returning
    // uninitialised bytes.
    EVP_MD* md = EVP_MD_fetch(nullptr, "RIPEMD160", nullptr);
    if (md == nullptr) {
        throw std::runtime_error("Ripemd160: the RIPEMD160 digest is unavailable");
    }

    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    unsigned int out_len = 0;
    const bool ok = ctx != nullptr && EVP_DigestInit_ex(ctx, md, nullptr) == 1 &&
                    EVP_DigestUpdate(ctx, data, length) == 1 &&
                    EVP_DigestFinal_ex(ctx, out.data(), &out_len) == 1 &&
                    out_len == out.size();

    EVP_MD_CTX_free(ctx);
    EVP_MD_free(md);

    if (!ok) throw std::runtime_error("Ripemd160: digest computation failed");
    return out;
}

std::array<uint8_t, 20> Ripemd160(const std::vector<uint8_t>& data) {
    return Ripemd160(data.data(), data.size());
}

std::array<uint8_t, 20> Hash160(const std::vector<uint8_t>& data) {
    const auto sha = crypto::Sha256(data);
    return Ripemd160(sha.data(), sha.size());
}

// ---------------------------------------------------------------------------
// BIP39
// ---------------------------------------------------------------------------

bool IsValidWordCount(size_t word_count) {
    return word_count == 12 || word_count == 15 || word_count == 18 ||
           word_count == 21 || word_count == 24;
}

std::optional<std::string> EntropyToMnemonic(const std::vector<uint8_t>& entropy) {
    const size_t bits = entropy.size() * 8;
    if (bits != 128 && bits != 160 && bits != 192 && bits != 224 && bits != 256) {
        return std::nullopt;
    }

    const auto digest = crypto::Sha256(entropy);
    const size_t checksum_bits = bits / 32;

    // The bit string is the entropy followed by the leading checksum bits.
    std::vector<uint8_t> all_bits;
    all_bits.reserve(bits + checksum_bits);
    for (size_t i = 0; i < bits; ++i) {
        all_bits.push_back(static_cast<uint8_t>((entropy[i / 8] >> (7 - (i % 8))) & 1));
    }
    for (size_t i = 0; i < checksum_bits; ++i) {
        all_bits.push_back(static_cast<uint8_t>((digest[i / 8] >> (7 - (i % 8))) & 1));
    }

    const char* const* words = Bip39Wordlist();
    std::string mnemonic;
    for (size_t i = 0; i < all_bits.size(); i += 11) {
        uint32_t index = 0;
        for (size_t j = 0; j < 11; ++j) index = (index << 1) | all_bits[i + j];
        if (index >= 2048) return std::nullopt;
        if (!mnemonic.empty()) mnemonic.push_back(' ');
        mnemonic += words[index];
    }
    return mnemonic;
}

std::optional<std::string> GenerateMnemonic(size_t word_count) {
    if (!IsValidWordCount(word_count)) return std::nullopt;
    // Eleven bits per word, and one checksum bit per 32 entropy bits.
    const size_t total_bits = word_count * 11;
    const size_t entropy_bytes = (total_bits - total_bits / 33) / 8;
    return EntropyToMnemonic(crypto::SecureRandom(entropy_bytes));
}

bool MnemonicToEntropy(const std::string& mnemonic, std::vector<uint8_t>& entropy) {
    // BIP39 separates words with exactly one space.
    std::vector<std::string> words;
    std::string current;
    for (char c : mnemonic) {
        if (c == ' ') {
            if (current.empty()) return false;  // a leading or doubled space
            words.push_back(current);
            current.clear();
        } else {
            current.push_back(c);
        }
    }
    if (!current.empty()) words.push_back(current);
    // A trailing space is not part of the canonical form. Accepting one here
    // while MnemonicToSeed hashes the raw string would derive a different
    // wallet than the user's input implies, so it is rejected.
    if (!mnemonic.empty() && mnemonic.back() == ' ') return false;
    if (!IsValidWordCount(words.size())) return false;

    // An index map rather than a scan, so validating a 24-word mnemonic is 24
    // lookups instead of 24 * 2048 comparisons.
    static const std::map<std::string, uint32_t> index_of = [] {
        std::map<std::string, uint32_t> m;
        const char* const* list = Bip39Wordlist();
        for (uint32_t i = 0; i < Bip39WordlistSize(); ++i) m[list[i]] = i;
        return m;
    }();

    std::vector<uint8_t> bits;
    bits.reserve(words.size() * 11);
    for (const auto& word : words) {
        const auto it = index_of.find(word);
        if (it == index_of.end()) return false;  // not in the word list
        for (int shift = 10; shift >= 0; --shift) {
            bits.push_back(static_cast<uint8_t>((it->second >> shift) & 1));
        }
    }

    return BitsToEntropy(bits, entropy);
}

bool ValidateMnemonic(const std::string& mnemonic) {
    std::vector<uint8_t> entropy;
    return MnemonicToEntropy(mnemonic, entropy);
}

std::array<uint8_t, 64> MnemonicToSeed(const std::string& mnemonic,
                                       const std::string& passphrase) {
    // BIP39 requires the mnemonic and the passphrase to be NFKD-normalized
    // before hashing. With the English word list the mnemonic is already NFKD,
    // so only a non-ASCII passphrase needs normalizing, and that is the
    // caller's responsibility. It is not silently approximated here.
    const std::string salt = "mnemonic" + passphrase;
    const auto seed = crypto::Pbkdf2HmacSha512(
        mnemonic, std::vector<uint8_t>(salt.begin(), salt.end()), 2048, 64);

    std::array<uint8_t, 64> out{};
    std::copy(seed.begin(), seed.end(), out.begin());
    return out;
}

// ---------------------------------------------------------------------------
// BIP32
// ---------------------------------------------------------------------------

bool ParsePath(const std::string& path, std::vector<PathElement>& out) {
    out.clear();
    if (path.empty()) return false;

    // Consume an optional leading "m". The separator after it is skipped here
    // so the loop below only has to deal with one shape: a step, then
    // optionally a slash and another step.
    size_t pos = 0;
    if (path[0] == 'm' || path[0] == 'M') {
        pos = 1;
        if (pos == path.size()) return true;  // just "m" means the master key
        if (path[pos] != '/') return false;
        ++pos;
    }

    while (pos < path.size()) {
        uint64_t index = 0;
        bool any_digits = false;
        while (pos < path.size() && path[pos] >= '0' && path[pos] <= '9') {
            index = index * 10 + static_cast<uint64_t>(path[pos] - '0');
            any_digits = true;
            ++pos;
            if (index > 0xFFFFFFFFull) return false;
        }
        if (!any_digits) return false;

        bool hardened = false;
        if (pos < path.size() &&
            (path[pos] == '\'' || path[pos] == 'h' || path[pos] == 'H')) {
            hardened = true;
            ++pos;
        }

        if (hardened) {
            // A hardened index must fit in 31 bits: bit 31 is the marker.
            if (index > 0x7FFFFFFFull) return false;
            index |= 0x80000000ull;
        } else if (index > 0x7FFFFFFFull) {
            // The top half of the range is reserved for hardened steps.
            return false;
        }

        PathElement element;
        element.index = static_cast<uint32_t>(index);
        element.hardened = hardened;
        out.push_back(element);

        if (pos == path.size()) break;
        if (path[pos] != '/') return false;
        ++pos;
        if (pos == path.size()) return false;  // a trailing slash
    }
    return !out.empty();
}

std::string Bip44Path(uint32_t purpose, uint32_t coin_type, uint32_t account,
                      uint32_t change, uint32_t index) {
    return "m/" + std::to_string(purpose) + "'/" + std::to_string(coin_type) + "'/" +
           std::to_string(account) + "'/" + std::to_string(change) + "/" +
           std::to_string(index);
}

std::optional<ExtendedKey> MasterKeyFromSeed(const std::vector<uint8_t>& seed) {
    // BIP32 accepts seeds of 16 to 64 bytes. An empty or oversized seed is
    // rejected rather than hashed into something plausible-looking.
    if (seed.size() < 16 || seed.size() > 64) return std::nullopt;

    const std::string key = "Bitcoin seed";
    const auto digest = crypto::HmacSha512(
        reinterpret_cast<const uint8_t*>(key.data()), key.size(), seed.data(), seed.size());

    BIGNUM* order = CurveOrder();
    BIGNUM* secret = BN_bin2bn(digest.data(), 32, nullptr);
    const bool valid = order != nullptr && secret != nullptr && IsValidScalar(secret, order);
    BN_free(secret);
    BN_free(order);
    if (!valid) return std::nullopt;

    ExtendedKey master;
    master.private_key.assign(digest.begin(), digest.begin() + kPrivateKeyLength);
    master.chain_code.assign(digest.begin() + kPrivateKeyLength, digest.end());
    master.depth = 0;
    master.child_number = 0;
    master.parent_fingerprint = 0;

    std::array<uint8_t, 32> secret_bytes{};
    std::copy(master.private_key.begin(), master.private_key.end(), secret_bytes.begin());
    const auto public_key = crypto::PrivateKeyToPublicKey(secret_bytes);
    if (public_key.size() != 64) return std::nullopt;

    // BIP32 stores the compressed form: 0x02 or 0x03, then the x coordinate.
    master.public_key.push_back((public_key[63] & 1) != 0 ? 0x03 : 0x02);
    master.public_key.insert(master.public_key.end(), public_key.begin(),
                             public_key.begin() + 32);
    return master;
}

std::optional<ExtendedKey> MasterKeyFromSeed(const std::array<uint8_t, 64>& seed) {
    return MasterKeyFromSeed(std::vector<uint8_t>(seed.begin(), seed.end()));
}

std::optional<ExtendedKey> DeriveChild(const ExtendedKey& parent, uint32_t index,
                                       bool hardened) {
    if (parent.chain_code.size() != kChainCodeLength) return std::nullopt;
    if (hardened && parent.private_key.size() != kPrivateKeyLength) return std::nullopt;
    if (parent.private_key.size() != kPrivateKeyLength &&
        parent.public_key.size() != kCompressedPublicKeyLength) {
        return std::nullopt;
    }
    // ParsePath returns hardened indices with bit 31 already set, and a caller
    // may also pass the bare index. Both are accepted for a hardened step.
    // A bare index at or above 2^31 with `hardened` false is contradictory:
    // that range is reserved for hardened derivation.
    if (!hardened && (index & 0x80000000u) != 0) return std::nullopt;

    const uint32_t child_number = hardened ? (index | 0x80000000u) : index;

    // The HMAC input is the parent private key for a hardened step, otherwise
    // the parent compressed public key, followed by the child index.
    std::vector<uint8_t> data;
    data.reserve(kCompressedPublicKeyLength + 4);
    if (hardened) {
        data.push_back(0x00);
        data.insert(data.end(), parent.private_key.begin(), parent.private_key.end());
    } else {
        data.insert(data.end(), parent.public_key.begin(), parent.public_key.end());
    }
    data.push_back(static_cast<uint8_t>((child_number >> 24) & 0xFF));
    data.push_back(static_cast<uint8_t>((child_number >> 16) & 0xFF));
    data.push_back(static_cast<uint8_t>((child_number >> 8) & 0xFF));
    data.push_back(static_cast<uint8_t>(child_number & 0xFF));

    const auto digest = crypto::HmacSha512(parent.chain_code.data(), parent.chain_code.size(),
                                           data.data(), data.size());

    BIGNUM* order = CurveOrder();
    BIGNUM* left = BN_bin2bn(digest.data(), 32, nullptr);
    if (order == nullptr || left == nullptr || !IsValidScalar(left, order)) {
        // An out-of-range IL is astronomically unlikely; BIP32 says to skip to
        // the next index, so the failure is reported rather than papered over.
        BN_free(left);
        BN_free(order);
        return std::nullopt;
    }

    ExtendedKey child;
    child.chain_code.assign(digest.begin() + 32, digest.end());
    child.depth = parent.depth + 1;
    child.child_number = child_number;
    child.parent_fingerprint = KeyFingerprint(parent);

    if (parent.private_key.size() == kPrivateKeyLength) {
        // child = (IL + k_par) mod n
        BIGNUM* parent_secret = BN_bin2bn(parent.private_key.data(), 32, nullptr);
        BIGNUM* sum = BN_new();
        BN_CTX* ctx = BN_CTX_new();
        bool ok = parent_secret != nullptr && sum != nullptr && ctx != nullptr &&
                  BN_mod_add(sum, left, parent_secret, order, ctx) == 1 && !BN_is_zero(sum);

        std::array<uint8_t, 32> secret{};
        if (ok) {
            // BN_bn2binpad writes exactly 32 bytes, left-padded with zeros.
            ok = BN_bn2binpad(sum, secret.data(), 32) == 32;
        }
        BN_CTX_free(ctx);
        BN_free(sum);
        BN_free(parent_secret);
        BN_free(left);
        BN_free(order);
        if (!ok) return std::nullopt;

        child.private_key.assign(secret.begin(), secret.end());
        const auto public_key = crypto::PrivateKeyToPublicKey(secret);
        if (public_key.size() != 64) return std::nullopt;
        child.public_key.push_back((public_key[63] & 1) != 0 ? 0x03 : 0x02);
        child.public_key.insert(child.public_key.end(), public_key.begin(),
                                public_key.begin() + 32);
        return child;
    }

    // Public-only derivation: the child public key is point(IL) plus the parent
    // point.
    BN_free(left);
    BN_free(order);

    secp256k1_context* ctx =
        secp256k1_context_create(SECP256K1_CONTEXT_VERIFY | SECP256K1_CONTEXT_SIGN);
    if (ctx == nullptr) return std::nullopt;

    secp256k1_pubkey pubkey;
    bool ok = secp256k1_ec_pubkey_parse(ctx, &pubkey, parent.public_key.data(),
                                        parent.public_key.size()) == 1;
    std::array<uint8_t, 32> tweak{};
    std::copy(digest.begin(), digest.begin() + 32, tweak.begin());
    if (ok) ok = secp256k1_ec_pubkey_tweak_add(ctx, &pubkey, tweak.data()) == 1;

    std::vector<uint8_t> serialized(kCompressedPublicKeyLength);
    size_t serialized_len = serialized.size();
    if (ok) {
        ok = secp256k1_ec_pubkey_serialize(ctx, serialized.data(), &serialized_len, &pubkey,
                                           SECP256K1_EC_COMPRESSED) == 1;
    }
    secp256k1_context_destroy(ctx);
    if (!ok) return std::nullopt;

    child.public_key = std::move(serialized);
    return child;
}

std::optional<ExtendedKey> DerivePath(const ExtendedKey& master,
                                      const std::vector<PathElement>& path) {
    ExtendedKey current = master;
    for (const auto& element : path) {
        const auto next = DeriveChild(current, element.index, element.hardened);
        if (!next) return std::nullopt;
        current = *next;
    }
    return current;
}

std::optional<ExtendedKey> DerivePath(const ExtendedKey& master, const std::string& path) {
    std::vector<PathElement> elements;
    if (!ParsePath(path, elements)) return std::nullopt;
    return DerivePath(master, elements);
}

uint32_t KeyFingerprint(const ExtendedKey& key) {
    if (key.public_key.size() != kCompressedPublicKeyLength) return 0;
    const auto hash = Hash160(key.public_key);
    return (static_cast<uint32_t>(hash[0]) << 24) | (static_cast<uint32_t>(hash[1]) << 16) |
           (static_cast<uint32_t>(hash[2]) << 8) | static_cast<uint32_t>(hash[3]);
}

std::optional<std::string> SerializeExtendedPrivateKey(const ExtendedKey& key) {
    if (key.private_key.size() != kPrivateKeyLength) return std::nullopt;
    if (key.chain_code.size() != kChainCodeLength) return std::nullopt;
    return SerializeExtended(key, kXprvVersion, true);
}

std::optional<std::string> SerializeExtendedPublicKey(const ExtendedKey& key) {
    if (key.public_key.size() != kCompressedPublicKeyLength) return std::nullopt;
    if (key.chain_code.size() != kChainCodeLength) return std::nullopt;
    return SerializeExtended(key, kXpubVersion, false);
}

std::optional<std::array<uint8_t, 20>> EthereumAddress(const ExtendedKey& key) {
    if (key.private_key.size() != kPrivateKeyLength) return std::nullopt;
    std::array<uint8_t, 32> secret{};
    std::copy(key.private_key.begin(), key.private_key.end(), secret.begin());

    std::array<uint8_t, 20> address{};
    if (!crypto::PrivateKeyToAddress(secret, address)) return std::nullopt;
    return address;
}

std::optional<std::array<uint8_t, 20>> BitcoinHash160(const ExtendedKey& key) {
    if (key.public_key.size() != kCompressedPublicKeyLength) return std::nullopt;
    return Hash160(key.public_key);
}

// ---------------------------------------------------------------------------
// BIP44 coin types
// ---------------------------------------------------------------------------

std::optional<uint32_t> CoinTypeForChain(const std::string& symbol) {
    // SLIP-44 registered values. An unregistered chain has no canonical coin
    // type, and inventing one would derive addresses nobody else reproduces, so
    // the absence is reported instead.
    static const std::map<std::string, uint32_t> coin_types = {
        {"BTC", 0},        {"LTC", 2},        {"DOGE", 3},       {"DASH", 5},
        {"ETH", 60},       {"ETC", 61},       {"BNB", 714},      {"SOL", 501},
        {"TRX", 195},      {"ATOM", 118},     {"XRP", 144},      {"XLM", 148},
        {"ADA", 1815},     {"DOT", 354},      {"KSM", 434},      {"NEAR", 397},
        {"TON", 607},      {"APT", 637},      {"SUI", 784},      {"SEI", 19000118},
        {"INJ", 60},       {"OSMO", 118},     {"AVAX", 9000},    {"MATIC", 966},
        {"POL", 966},      {"FTM", 1007},     {"CELO", 52752},   {"KAVA", 459},
        {"ONE", 1023},     {"ALGO", 283},     {"XTZ", 1729},     {"EGLD", 508},
        {"HBAR", 3030},    {"FIL", 461},      {"ICP", 223},      {"VET", 818},
        {"ZEC", 133},      {"BCH", 145},      {"BSV", 236},      {"GRS", 17},
        {"ZIL", 119},      {"WAVES", 5741564},{"KAS", 111111},   {"NANO", 165},
        {"CKB", 309},      {"FLOW", 539},     {"NEO", 888},      {"AR", 472},
        {"ROSE", 474},     {"STX", 5757},
    };
    const auto it = coin_types.find(symbol);
    if (it == coin_types.end()) return std::nullopt;
    return it->second;
}

}  // namespace hd
}  // namespace tigerwallet
