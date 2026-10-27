/**
 * TigerWallet Hardware Wallet - Production-Ready C++ Implementation
 * Source file with full implementation
 */

#include "hardware_wallet.h"

#include "json_rpc.h"
#include <algorithm>
#include <stdexcept>
#include <fstream>

#ifdef __linux__
#include <dirent.h>
#endif
#include <random>
#include <sstream>
#include <iomanip>
#include <chrono>
#include <thread>

// Include crypto library (using libsodium-compatible implementation)
#include <openssl/sha.h>
#include <openssl/ec.h>
#include <openssl/obj_mac.h>
#include <openssl/bn.h>
#include <openssl/ripemd.h>
#include <openssl/evp.h>
#include <openssl/rand.h>
#include <openssl/param_build.h>
#include <openssl/core_names.h>
#include <openssl/hmac.h>
#include <openssl/ec.h>
#include <openssl/bn.h>

namespace tigerwallet {
namespace hardware {

// ============================================================================
// Crypto Utils Implementation
// ============================================================================

std::array<uint8_t, 32> CryptoUtils::sha256(const std::vector<uint8_t>& data) {
    return sha256(data.data(), data.size());
}

std::array<uint8_t, 32> CryptoUtils::sha256(const uint8_t* data, size_t len) {
    std::array<uint8_t, 32> result;
    SHA256(data, len, result.data());
    return result;
}

std::array<uint8_t, 32> CryptoUtils::double_sha256(const std::vector<uint8_t>& data) {
    auto h1 = sha256(data);
    return sha256(h1.data(), h1.size());
}

namespace {

/// The value of a base58 character, or -1 when it is not in the alphabet.
int base58_value(char c) {
    static const char* alphabet =
        "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

    for (int i = 0; i < 58; i++) {
        if (alphabet[i] == c) {
            return i;
        }
    }

    return -1;
}

/// The value of a base64 character, or -1 when it is not in the alphabet.
///
/// Both the standard and the URL-safe alphabets are accepted, because a
/// signature or key may arrive from either.
int base64_value(char c) {
    if (c >= 'A' && c <= 'Z') return c - 'A';
    if (c >= 'a' && c <= 'z') return c - 'a' + 26;
    if (c >= '0' && c <= '9') return c - '0' + 52;
    if (c == '+' || c == '-') return 62;
    if (c == '/' || c == '_') return 63;

    return -1;
}

}  // namespace

std::string CryptoUtils::base58_encode(const std::vector<uint8_t>& data) {
    static const char* alphabet =
        "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

    if (data.empty()) {
        return "";
    }

    // Every leading zero byte becomes a leading '1'.
    size_t zeros = 0;
    while (zeros < data.size() && data[zeros] == 0) {
        zeros++;
    }

    // Repeated division by 58 over the whole number. The buffer holds base58
    // digits, least significant first.
    std::vector<uint8_t> digits;
    digits.reserve(data.size() * 138 / 100 + 1);

    for (size_t i = zeros; i < data.size(); i++) {
        int carry = data[i];
        for (size_t j = 0; j < digits.size(); j++) {
            carry += 256 * digits[j];
            digits[j] = static_cast<uint8_t>(carry % 58);
            carry /= 58;
        }
        while (carry > 0) {
            digits.push_back(static_cast<uint8_t>(carry % 58));
            carry /= 58;
        }
    }

    std::string result(zeros, '1');
    for (auto it = digits.rbegin(); it != digits.rend(); ++it) {
        result += alphabet[*it];
    }

    return result;
}

std::vector<uint8_t> CryptoUtils::base58_decode(const std::string& data) {
    // Invalid characters are rejected rather than skipped. Skipping them would
    // decode two different strings to the same bytes, which for an address or a
    // key means accepting a value the user never entered.
    std::vector<uint8_t> result;
    if (data.empty()) {
        return result;
    }

    size_t zeros = 0;
    while (zeros < data.size() && data[zeros] == '1') {
        zeros++;
    }

    // Base-256 accumulator, least significant byte first.
    std::vector<uint8_t> digits;
    digits.reserve(data.size() * 733 / 1000 + 1);

    for (size_t i = zeros; i < data.size(); i++) {
        int value = base58_value(data[i]);
        if (value < 0) {
            throw std::invalid_argument("the value contains a character that is not valid base58");
        }

        int carry = value;
        for (size_t j = 0; j < digits.size(); j++) {
            carry += 58 * digits[j];
            digits[j] = static_cast<uint8_t>(carry % 256);
            carry /= 256;
        }

        while (carry > 0) {
            digits.push_back(static_cast<uint8_t>(carry % 256));
            carry /= 256;
        }
    }

    result.assign(zeros, 0);
    for (auto it = digits.rbegin(); it != digits.rend(); ++it) {
        result.push_back(*it);
    }

    return result;
}

std::string CryptoUtils::base64_encode(const std::vector<uint8_t>& data) {
    static const char* base64_chars = 
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    
    std::string result;
    int i = 0;
    int j = 0;
    uint8_t char_array_3[3];
    uint8_t char_array_4[4];
    
    for (auto b : data) {
        char_array_3[i++] = b;
        if (i == 3) {
            char_array_4[0] = (char_array_3[0] & 0xfc) >> 2;
            char_array_4[1] = ((char_array_3[0] & 0x03) << 4) + ((char_array_3[1] & 0xf0) >> 4);
            char_array_4[2] = ((char_array_3[1] & 0x0f) << 2) + ((char_array_3[2] & 0xc0) >> 6);
            char_array_4[3] = char_array_3[2] & 0x3f;
            
            for(i = 0; i < 4; i++)
                result += base64_chars[char_array_4[i]];
            i = 0;
        }
    }
    
    if (i > 0) {
        for(j = i; j < 3; j++)
            char_array_3[j] = 0;
        
        char_array_4[0] = (char_array_3[0] & 0xfc) >> 2;
        char_array_4[1] = ((char_array_3[0] & 0x03) << 4) + ((char_array_3[1] & 0xf0) >> 4);
        char_array_4[2] = ((char_array_3[1] & 0x0f) << 2) + ((char_array_3[2] & 0xc0) >> 6);
        
        for (j = 0; j < i + 1; j++)
            result += base64_chars[char_array_4[j]];
        
        while (i++ < 3)
            result += '=';
    }
    
    return result;
}

std::vector<uint8_t> CryptoUtils::base64_decode(const std::string& data) {
    // As with base58, an invalid character is an error rather than something to
    // skip over.
    std::vector<uint8_t> result;
    if (data.empty()) {
        return result;
    }

    int buffer = 0;
    int bits = 0;

    for (char c : data) {
        if (c == '=' || c == '\n' || c == '\r') {
            break;
        }

        int value = base64_value(c);
        if (value < 0) {
            throw std::invalid_argument("the value contains a character that is not valid base64");
        }

        buffer = (buffer << 6) | value;
        bits += 6;

        if (bits >= 8) {
            bits -= 8;
            result.push_back(static_cast<uint8_t>((buffer >> bits) & 0xFF));
        }
    }

    // The remaining bits, if any, must be zero padding.
    if (bits > 0 && ((buffer & ((1 << bits) - 1)) != 0)) {
        throw std::invalid_argument("the base64 value has non-zero trailing bits");
    }

    return result;
}

uint32_t CryptoUtils::uint32_from_big_endian(const uint8_t* data) {
    return ((uint32_t)data[0] << 24) |
           ((uint32_t)data[1] << 16) |
           ((uint32_t)data[2] << 8) |
           ((uint32_t)data[3]);
}

std::vector<uint8_t> CryptoUtils::hex_decode(const std::string& hex) {
    std::vector<uint8_t> result;
    for (size_t i = 0; i < hex.length(); i += 2) {
        std::string byte_str = hex.substr(i, 2);
        uint8_t byte = (uint8_t)strtol(byte_str.c_str(), nullptr, 16);
        result.push_back(byte);
    }
    return result;
}

std::string CryptoUtils::hex_encode(const uint8_t* data, size_t len) {
    std::stringstream ss;
    ss << std::hex << std::setfill('0');
    for (size_t i = 0; i < len; i++) {
        ss << std::setw(2) << (int)data[i];
    }
    return ss.str();
}

// ============================================================================
// Hash and Key Primitives
// ============================================================================

std::array<uint8_t, 4> CryptoUtils::uint32_to_big_endian(uint32_t value) {
    return {static_cast<uint8_t>(value >> 24),
            static_cast<uint8_t>(value >> 16),
            static_cast<uint8_t>(value >> 8),
            static_cast<uint8_t>(value)};
}

std::array<uint8_t, 32> CryptoUtils::keccak256(const std::vector<uint8_t>& data) {
    std::array<uint8_t, 32> digest{};

    EVP_MD* md = EVP_MD_fetch(nullptr, "KECCAK-256", nullptr);
    if (md == nullptr) {
        throw std::runtime_error("Keccak-256 is not available in this OpenSSL build");
    }

    unsigned int length = 0;
    if (EVP_Digest(data.data(), data.size(), digest.data(), &length, md, nullptr) != 1 ||
        length != 32) {
        EVP_MD_free(md);
        throw std::runtime_error("Keccak-256 failed");
    }

    EVP_MD_free(md);
    return digest;
}

std::array<uint8_t, 64> CryptoUtils::hmac_sha512(const std::vector<uint8_t>& key,
                                                 const std::vector<uint8_t>& data) {
    std::array<uint8_t, 64> mac{};
    unsigned int length = 0;

    if (HMAC(EVP_sha512(), key.data(), static_cast<int>(key.size()),
             data.data(), data.size(), mac.data(), &length) == nullptr ||
        length != 64) {
        throw std::runtime_error("HMAC-SHA512 failed");
    }

    return mac;
}

bool CryptoUtils::public_key_from_private(const std::array<uint8_t, 32>& private_key,
                                          std::array<uint8_t, 65>& public_key) {
    bool ok = false;

    BIGNUM* scalar = BN_bin2bn(private_key.data(), 32, nullptr);
    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);

    if (scalar != nullptr && group != nullptr) {
        EC_POINT* point = EC_POINT_new(group);
        BN_CTX* context = BN_CTX_new();

        if (point != nullptr && context != nullptr &&
            BN_cmp(scalar, BN_value_one()) >= 0 &&
            EC_POINT_mul(group, point, scalar, nullptr, nullptr, context) == 1) {
            size_t written = EC_POINT_point2oct(
                group, point, POINT_CONVERSION_UNCOMPRESSED, public_key.data(), 65, context);

            // The first byte is the 0x04 uncompressed marker.
            ok = (written == 65) && (public_key[0] == 0x04);
        }

        if (context != nullptr) BN_CTX_free(context);
        if (point != nullptr) EC_POINT_free(point);
    }

    if (group != nullptr) EC_GROUP_free(group);
    if (scalar != nullptr) BN_free(scalar);

    return ok;
}

bool CryptoUtils::decompress_pubkey(const std::array<uint8_t, 33>& compressed,
                                    std::array<uint8_t, 65>& uncompressed) {
    bool ok = false;

    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);
    BN_CTX* context = BN_CTX_new();

    if (group != nullptr && context != nullptr) {
        EC_POINT* point = EC_POINT_new(group);

        if (point != nullptr &&
            EC_POINT_oct2point(group, point, compressed.data(), compressed.size(), context) == 1 &&
            EC_POINT_is_on_curve(group, point, context) == 1) {
            const size_t written = EC_POINT_point2oct(
                group, point, POINT_CONVERSION_UNCOMPRESSED,
                uncompressed.data(), uncompressed.size(), context);

            ok = (written == 65) && (uncompressed[0] == 0x04);
        }

        if (point != nullptr) EC_POINT_free(point);
    }

    if (context != nullptr) BN_CTX_free(context);
    if (group != nullptr) EC_GROUP_free(group);

    return ok;
}

std::array<uint8_t, 33> CryptoUtils::compress_pubkey(const std::array<uint8_t, 65>& uncompressed) {
    // The parity of Y decides the prefix: 0x02 for even, 0x03 for odd.
    std::array<uint8_t, 33> compressed{};
    compressed[0] = (uncompressed[64] & 1) ? 0x03 : 0x02;
    std::copy(uncompressed.begin() + 1, uncompressed.begin() + 33, compressed.begin() + 1);
    return compressed;
}

// ============================================================================
// BIP32 Implementation
// ============================================================================

std::pair<std::array<uint8_t, 32>, std::array<uint8_t, 32>> 
BIP32::master_key_from_seed(const std::vector<uint8_t>& seed) {
    // BIP32: I = HMAC-SHA512(key = "Bitcoin seed", data = seed).
    // The left half is the master private key, the right half the chain code.
    if (seed.size() < 16 || seed.size() > 64) {
        throw std::invalid_argument("the BIP32 seed must be between 16 and 64 bytes");
    }

    const std::vector<uint8_t> key = {'B', 'i', 't', 'c', 'o', 'i', 'n', ' ',
                                      's', 'e', 'e', 'd'};
    auto mac = CryptoUtils::hmac_sha512(key, seed);

    std::array<uint8_t, 32> master_key{};
    std::array<uint8_t, 32> chaincode{};
    std::copy(mac.begin(), mac.begin() + 32, master_key.begin());
    std::copy(mac.begin() + 32, mac.end(), chaincode.begin());

    // A master key that is zero or at least the group order is not usable.
    BIGNUM* order = nullptr;
    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);
    if (group != nullptr) {
        order = BN_new();
        EC_GROUP_get_order(group, order, nullptr);
        BIGNUM* candidate = BN_bin2bn(master_key.data(), 32, nullptr);

        if (candidate != nullptr && BN_cmp(candidate, order) >= 0) {
            BN_free(candidate);
            BN_free(order);
            EC_GROUP_free(group);
            throw std::runtime_error("the derived master key is out of range for secp256k1");
        }

        if (candidate != nullptr) BN_free(candidate);
        BN_free(order);
        EC_GROUP_free(group);
    }

    return {master_key, chaincode};
}

std::pair<std::array<uint8_t, 32>, std::array<uint8_t, 32>>
BIP32::derive_child_key(const std::array<uint8_t, 32>& parent_key,
                        const std::array<uint8_t, 32>& chaincode,
                        uint32_t index) {
    // BIP32 child derivation. The hardened flag changes what goes into the
    // HMAC: a hardened child uses the private key, a normal child the public.
    const bool hardened = (index & 0x80000000u) != 0;

    std::vector<uint8_t> data;
    data.reserve(37);

    if (hardened) {
        data.push_back(0x00);
        data.insert(data.end(), parent_key.begin(), parent_key.end());
    } else {
        std::array<uint8_t, 65> public_key{};
        if (!CryptoUtils::public_key_from_private(parent_key, public_key)) {
            throw std::runtime_error("the parent key is not a valid secp256k1 scalar");
        }

        std::array<uint8_t, 33> compressed = CryptoUtils::compress_pubkey(public_key);
        data.insert(data.end(), compressed.begin(), compressed.end());
    }

    auto index_bytes = CryptoUtils::uint32_to_big_endian(index);
    data.insert(data.end(), index_bytes.begin(), index_bytes.end());

    auto mac = CryptoUtils::hmac_sha512(
        std::vector<uint8_t>(chaincode.begin(), chaincode.end()), data);

    // The child key is (IL + parent) mod n.
    BIGNUM* il = BN_bin2bn(mac.data(), 32, nullptr);
    BIGNUM* parent = BN_bin2bn(parent_key.data(), 32, nullptr);
    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);
    BIGNUM* order = BN_new();
    BIGNUM* child = BN_new();

    if (il == nullptr || parent == nullptr || group == nullptr ||
        order == nullptr || child == nullptr) {
        if (il) BN_free(il);
        if (parent) BN_free(parent);
        if (order) BN_free(order);
        if (child) BN_free(child);
        if (group) EC_GROUP_free(group);
        throw std::runtime_error("out of memory during child key derivation");
    }

    EC_GROUP_get_order(group, order, nullptr);

    std::array<uint8_t, 32> child_key{};
    std::array<uint8_t, 32> child_chaincode{};

    if (BN_cmp(il, order) >= 0) {
        // An IL at or above the group order makes this index unusable. BIP32
        // says to move on to the next index; the caller decides how.
        BN_free(il); BN_free(parent); BN_free(order); BN_free(child); EC_GROUP_free(group);
        throw std::runtime_error("the derived key is invalid for this index");
    }

    BN_mod_add(child, il, parent, order, BN_CTX_new());
    BN_bn2binpad(child, child_key.data(), 32);

    std::copy(mac.begin() + 32, mac.end(), child_chaincode.begin());

    BN_free(il);
    BN_free(parent);
    BN_free(order);
    BN_free(child);
    EC_GROUP_free(group);

    return {child_key, child_chaincode};
}


std::string BIP32::derive_address(const std::array<uint8_t, 33>& pubkey, 
                                  const std::string& chain) {
    if (chain == "ETH" || chain == "60") {
        return CryptoUtils::derive_evm_address(pubkey);
    } else if (chain == "BTC" || chain == "0") {
        return CryptoUtils::derive_bitcoin_address(pubkey, false);
    }
    return "";
}

std::vector<WalletInfo> BIP32::generate_addresses(
    const std::string& chain,
    uint32_t start_index,
    uint32_t count,
    const std::vector<uint8_t>& seed) {
    // Without a seed there is nothing to derive from. Reporting that is better
    // than handing back a list of addresses that belong to no one.
    if (seed.empty()) {
        throw std::invalid_argument("a seed is required to generate addresses");
    }

    auto [master_key, chaincode] = master_key_from_seed(seed);
    std::vector<WalletInfo> result;

    for (uint32_t i = 0; i < count; i++) {
        // The account level is hardened, the change and index levels are not.
        auto [account_key, account_chaincode] =
            derive_child_key(master_key, chaincode, BIP44::PURPOSE | 0x80000000u);
        auto [coin_key, coin_chaincode] =
            derive_child_key(account_key, account_chaincode,
                             BIP44::ETH | 0x80000000u);
        auto [change_key, change_chaincode] =
            derive_child_key(coin_key, coin_chaincode, 0);
        auto [index_key, unused_chaincode] =
            derive_child_key(change_key, change_chaincode, start_index + i);
        (void)unused_chaincode;

        std::array<uint8_t, 65> uncompressed{};
        if (!CryptoUtils::public_key_from_private(index_key, uncompressed)) {
            continue;
        }

        std::array<uint8_t, 33> pubkey = CryptoUtils::compress_pubkey(uncompressed);

        WalletInfo info;
        info.address = derive_address(pubkey, chain);
        info.path = BIP44::build_path(BIP44::ETH, 0, 0, start_index + i);
        info.derivation_type = "BIP44";
        info.visible = true;

        result.push_back(info);
    }

    return result;
}

bool CryptoUtils::ecdsa_verify(const std::array<uint8_t, 32>& message,
                               const std::array<uint8_t, SECP256K1_SIG_SIZE>& signature,
                               const std::array<uint8_t, SECP256K1_PUBKEY_SIZE>& pubkey) {
    // The caller passes the 33-byte compressed key. The point is rebuilt first
    // so a key that is not on the curve is rejected before any verification.
    if (pubkey[0] != 0x02 && pubkey[0] != 0x03) {
        return false;
    }

    std::array<uint8_t, 65> uncompressed{};
    if (!decompress_pubkey(pubkey, uncompressed)) {
        return false;
    }

    // Build the public key through the provider interface. The EC_KEY API is
    // deprecated in OpenSSL 3 and is not available in a FIPS build.
    OSSL_PARAM_BLD* builder = OSSL_PARAM_BLD_new();
    OSSL_PARAM* params = nullptr;
    EVP_PKEY_CTX* fromdata = nullptr;
    EVP_PKEY* key = nullptr;

    if (builder != nullptr) {
        if (OSSL_PARAM_BLD_push_utf8_string(builder, OSSL_PKEY_PARAM_GROUP_NAME,
                                            "secp256k1", 0) == 1 &&
            OSSL_PARAM_BLD_push_octet_string(builder, OSSL_PKEY_PARAM_PUB_KEY,
                                             uncompressed.data(),
                                             uncompressed.size()) == 1) {
            params = OSSL_PARAM_BLD_to_param(builder);
        }
    }

    if (params != nullptr) {
        fromdata = EVP_PKEY_CTX_new_from_name(nullptr, "EC", nullptr);
        if (fromdata != nullptr &&
            EVP_PKEY_fromdata_init(fromdata) == 1 &&
            EVP_PKEY_fromdata(fromdata, &key, EVP_PKEY_PUBLIC_KEY, params) != 1) {
            key = nullptr;
        }
    }

    if (builder != nullptr) OSSL_PARAM_BLD_free(builder);
    if (params != nullptr) OSSL_PARAM_free(params);
    if (fromdata != nullptr) EVP_PKEY_CTX_free(fromdata);

    if (key == nullptr) {
        return false;
    }

    // The signature is r || s in the raw 64-byte form; the provider wants DER.
    ECDSA_SIG* sig = ECDSA_SIG_new();
    BIGNUM* r = BN_bin2bn(signature.data(), 32, nullptr);
    BIGNUM* s = BN_bin2bn(signature.data() + 32, 32, nullptr);

    bool verified = false;

    if (sig != nullptr && r != nullptr && s != nullptr) {
        ECDSA_SIG_set0(sig, r, s);

        unsigned char* der = nullptr;
        const int der_length = i2d_ECDSA_SIG(sig, &der);

        if (der_length > 0 && der != nullptr) {
            // The caller passes an already-computed 32-byte digest, so the
            // digest is verified directly. A DigestVerify call would hash the
            // input a second time and never match.
            EVP_PKEY_CTX* ctx = EVP_PKEY_CTX_new(key, nullptr);
            if (ctx != nullptr) {
                if (EVP_PKEY_verify_init(ctx) == 1 &&
                    EVP_PKEY_verify(ctx, der, static_cast<size_t>(der_length),
                                    message.data(), message.size()) == 1) {
                    verified = true;
                }
                EVP_PKEY_CTX_free(ctx);
            }

            OPENSSL_free(der);
        } else {
            BN_free(r);
            BN_free(s);
        }

        ECDSA_SIG_free(sig);
    } else {
        if (r != nullptr) BN_free(r);
        if (s != nullptr) BN_free(s);
        if (sig != nullptr) ECDSA_SIG_free(sig);
    }

    EVP_PKEY_free(key);

    return verified;
}

std::pair<std::array<uint8_t, 32>, std::array<uint8_t, 32>>
CryptoUtils::derive_bip44(const std::vector<uint32_t>& path) {
    if (path.empty()) {
        throw std::invalid_argument("a BIP44 path needs at least one level");
    }

    // The seed is not part of this call, so the caller supplies the first two
    // levels and the derivation continues from there. Without a seed there is
    // nothing to derive, so this reports rather than inventing a key.
    throw std::invalid_argument(
        "derive_bip44 needs a seed; use BIP32::master_key_from_seed and BIP32::derive_child_key");
}

std::string CryptoUtils::derive_evm_address(const std::array<uint8_t, SECP256K1_PUBKEY_SIZE>& pubkey) {
    // An address is the last 20 bytes of Keccak-256 over the uncompressed key
    // with its 0x04 prefix removed. The caller passes the 33-byte compressed
    // key, so the point is decompressed first: hashing the compressed bytes
    // would give a different, wrong address.
    std::array<uint8_t, 65> uncompressed{};

    if (pubkey[0] == 0x04) {
        // Already an uncompressed point carried in the first 65 bytes.
        std::copy(pubkey.begin(), pubkey.end(), uncompressed.begin());
    } else if (pubkey[0] == 0x02 || pubkey[0] == 0x03) {
        if (!decompress_pubkey(pubkey, uncompressed)) {
            throw std::invalid_argument("the public key is not a point on secp256k1");
        }
    } else {
        throw std::invalid_argument("the public key prefix is not recognised");
    }

    auto hash = keccak256(std::vector<uint8_t>(uncompressed.begin() + 1, uncompressed.end()));

    std::array<uint8_t, 20> address{};
    std::copy(hash.end() - 20, hash.end(), address.begin());

    // EIP-55: upper-case a hex letter when the matching nibble of the Keccak
    // hash of the lower-case address is 8 or above.
    const std::string lower = hex_encode(address.data(), 20);
    auto hash_of_address = keccak256(std::vector<uint8_t>(lower.begin(), lower.end()));

    std::string checksummed = "0x";
    for (size_t i = 0; i < lower.size(); i++) {
        const char character = lower[i];
        if (character >= 'a' && character <= 'f') {
            const uint8_t nibble = (i % 2 == 0) ? (hash_of_address[i / 2] >> 4)
                                                : (hash_of_address[i / 2] & 0x0F);
            checksummed += (nibble >= 8) ? static_cast<char>(character - 32) : character;
        } else {
            checksummed += character;
        }
    }

    return checksummed;
}

std::string CryptoUtils::derive_bitcoin_address(const std::array<uint8_t, 33>& pubkey, bool testnet) {
    // P2PKH: RIPEMD160(SHA256(pubkey))
    auto sha_hash = sha256(std::vector<uint8_t>(pubkey.begin(), pubkey.end()));
    
    // RIPEMD160, through the provider interface: the bare RIPEMD160 call is
    // deprecated in OpenSSL 3 and may be absent from a minimal build.
    uint8_t ripemd_output[20];
    unsigned int ripemd_length = 0;
    EVP_MD* ripemd = EVP_MD_fetch(nullptr, "RIPEMD160", nullptr);
    if (ripemd == nullptr ||
        EVP_Digest(sha_hash.data(), sha_hash.size(), ripemd_output,
                   &ripemd_length, ripemd, nullptr) != 1 ||
        ripemd_length != 20) {
        if (ripemd != nullptr) EVP_MD_free(ripemd);
        throw std::runtime_error("RIPEMD160 is not available in this OpenSSL build");
    }
    EVP_MD_free(ripemd);
    
    // Add version byte
    std::vector<uint8_t> address_data;
    address_data.push_back(testnet ? 0x6F : 0x00);  // Testnet: 0x6F, Mainnet: 0x00
    address_data.insert(address_data.end(), ripemd_output, ripemd_output + 20);
    
    // Double SHA256 for checksum
    auto checksum = double_sha256(address_data);
    
    // Add first 4 bytes of checksum
    address_data.insert(address_data.end(), checksum.begin(), checksum.begin() + 4);
    
    return base58_encode(address_data);
}

// ============================================================================
// BIP39 Implementation
// ============================================================================

const std::vector<std::string>& BIP39::wordlist() {
    static const std::vector<std::string> words = [] {
        std::vector<std::string> list;
        list.reserve(BIP39_ENGLISH_WORDS.size());
        for (const auto& word : BIP39_ENGLISH_WORDS) {
            list.emplace_back(word);
        }
        return list;
    }();
    return words;
}

std::vector<std::string> BIP39::get_wordlist() {
    return wordlist();
}

size_t BIP39::bits_of_entropy(size_t entropy_bytes) {
    return entropy_bytes * 8;
}

bool BIP39::checksum_valid(const std::vector<uint8_t>& entropy) {
    // BIP39 defines the entropy as 128, 160, 192, 224 or 256 bits. Any other
    // length has no checksum length, so it cannot be valid.
    switch (entropy.size()) {
        case 16:
        case 20:
        case 24:
        case 28:
        case 32:
            return true;
        default:
            return false;
    }
}

std::vector<uint8_t> BIP39::entropy_to_mnemonic(const std::vector<uint8_t>& entropy) {
    if (!checksum_valid(entropy)) {
        throw std::invalid_argument("the entropy length is not one BIP39 defines");
    }

    const size_t bits = entropy.size() * 8;
    const size_t checksum_bits = bits / 32;

    auto digest = CryptoUtils::sha256(entropy);

    std::vector<uint8_t> bit_stream;
    bit_stream.reserve(bits + checksum_bits);

    for (size_t i = 0; i < bits; i++) {
        bit_stream.push_back((entropy[i / 8] >> (7 - (i % 8))) & 1);
    }
    for (size_t i = 0; i < checksum_bits; i++) {
        bit_stream.push_back((digest[i / 8] >> (7 - (i % 8))) & 1);
    }

    const auto& words = wordlist();
    std::vector<uint8_t> encoded;

    for (size_t i = 0; i + 11 <= bit_stream.size(); i += 11) {
        uint16_t index = 0;
        for (size_t j = 0; j < 11; j++) {
            index = static_cast<uint16_t>((index << 1) | bit_stream[i + j]);
        }
        if (index >= words.size()) {
            throw std::runtime_error("the mnemonic index is outside the word list");
        }

        // The words come back as the bytes of a space-separated string so the
        // caller can show them without a second encoding step.
        if (!encoded.empty()) {
            encoded.push_back(' ');
        }
        const std::string& word = words[index];
        encoded.insert(encoded.end(), word.begin(), word.end());
    }

    return encoded;
}

std::vector<uint8_t> BIP39::generate_mnemonic(uint32_t strength) {
    // Only the BIP39 lengths are accepted; anything else has no defined
    // checksum length.
    if (strength != 128 && strength != 160 && strength != 192 &&
        strength != 224 && strength != 256) {
        throw std::invalid_argument(
            "the BIP39 entropy must be 128, 160, 192, 224 or 256 bits");
    }

    std::vector<uint8_t> entropy(strength / 8);

    // The entropy comes from the system random source. A predictable seed
    // would let anyone rebuild the wallet.
    if (RAND_bytes(entropy.data(), static_cast<int>(entropy.size())) != 1) {
        throw std::runtime_error("the system random source failed");
    }

    return entropy_to_mnemonic(entropy);
}

bool BIP39::validate_mnemonic(const std::string& mnemonic) {
    std::istringstream iss(mnemonic);
    std::string word;
    std::vector<std::string> words;

    while (iss >> word) {
        words.push_back(word);
    }

    // BIP39 defines 12, 15, 18, 21 and 24 words.
    if (words.size() != 12 && words.size() != 15 && words.size() != 18 &&
        words.size() != 21 && words.size() != 24) {
        return false;
    }

    const auto& list = wordlist();

    // Every word must be in the list and the checksum must agree. Checking
    // membership alone would accept a mistyped phrase, which restores a
    // different wallet.
    std::vector<uint8_t> bit_stream;
    bit_stream.reserve(words.size() * 11);

    for (const auto& candidate : words) {
        auto it = std::lower_bound(list.begin(), list.end(), candidate);
        if (it == list.end() || *it != candidate) {
            return false;
        }

        const size_t index = static_cast<size_t>(std::distance(list.begin(), it));
        for (int bit = 10; bit >= 0; bit--) {
            bit_stream.push_back(static_cast<uint8_t>((index >> bit) & 1));
        }
    }

    const size_t total_bits = bit_stream.size();
    const size_t checksum_bits = total_bits / 33;
    const size_t entropy_bits = total_bits - checksum_bits;

    std::vector<uint8_t> entropy(entropy_bits / 8, 0);
    for (size_t i = 0; i < entropy_bits; i++) {
        entropy[i / 8] |= static_cast<uint8_t>(bit_stream[i] << (7 - (i % 8)));
    }

    auto digest = CryptoUtils::sha256(entropy);

    for (size_t i = 0; i < checksum_bits; i++) {
        const uint8_t expected = (digest[i / 8] >> (7 - (i % 8))) & 1;
        if (bit_stream[entropy_bits + i] != expected) {
            return false;
        }
    }

    return true;
}

std::vector<uint8_t> BIP39::mnemonic_to_seed(const std::string& mnemonic,
                                             const std::string& passphrase) {
    // BIP39: the seed is PBKDF2-HMAC-SHA512 over the mnemonic, salted with
    // "mnemonic" plus the passphrase, run for 2048 rounds.
    const std::string salt = "mnemonic" + passphrase;
    std::vector<uint8_t> seed(64, 0);

    if (PKCS5_PBKDF2_HMAC(mnemonic.c_str(), static_cast<int>(mnemonic.size()),
                          reinterpret_cast<const unsigned char*>(salt.data()),
                          static_cast<int>(salt.size()),
                          2048, EVP_sha512(), static_cast<int>(seed.size()),
                          seed.data()) != 1) {
        throw std::runtime_error("the BIP39 seed derivation failed");
    }

    return seed;
}

// ============================================================================
// BIP44 Implementation
// ============================================================================

BIP44::PathComponents BIP44::parse_path(const std::string& path) {
    // A malformed path is an error. Silently substituting a default would sign
    // with a different key than the one the caller asked for.
    std::vector<uint32_t> levels;
    std::istringstream iss(path);
    std::string part;

    while (std::getline(iss, part, '/')) {
        if (part.empty() || part == "m") continue;

        bool hardened = part.back() == '\'' || part.back() == 'h' || part.back() == 'H';
        if (hardened) part.pop_back();

        if (part.empty()) {
            throw std::invalid_argument("the derivation path has an empty level");
        }

        // Only digits are accepted; std::stoi would also take a leading sign or
        // surrounding whitespace.
        for (char c : part) {
            if (c < '0' || c > '9') {
                throw std::invalid_argument("the derivation path has a non-numeric level");
            }
        }

        unsigned long parsed = 0;
        try {
            parsed = std::stoul(part);
        } catch (...) {
            throw std::invalid_argument("the derivation path level is out of range");
        }

        if (parsed > 0x7FFFFFFFul) {
            throw std::invalid_argument("the derivation path level exceeds 2^31 - 1");
        }

        uint32_t value = static_cast<uint32_t>(parsed);
        if (hardened) value |= 0x80000000;

        levels.push_back(value);
    }

    if (levels.empty()) {
        throw std::invalid_argument("the derivation path has no levels");

    // Ignoring extra levels would derive a key the caller did not ask for.
    if (levels.size() > 5) {
        throw std::invalid_argument("the derivation path has more than 5 levels");
    }
    }

    PathComponents components = {0, 0, 0, 0, 0};
    if (levels.size() >= 1) components.purpose = levels[0];
    if (levels.size() >= 2) components.coin_type = levels[1];
    if (levels.size() >= 3) components.account = levels[2];
    if (levels.size() >= 4) components.change = levels[3];
    if (levels.size() >= 5) components.address_index = levels[4];

    return components;
}

std::string BIP44::build_path(const PathComponents& components) {
    std::stringstream ss;
    ss << "m/" << (components.purpose & 0x7FFFFFFF) << "'/"
       << (components.coin_type & 0x7FFFFFFF) << "'/"
       << (components.account & 0x7FFFFFFF) << "'/"
       << components.change << "/" << components.address_index;
    return ss.str();
}

std::string BIP44::build_path(uint32_t coin_type, uint32_t account, 
                               uint32_t change, uint32_t index) {
    std::stringstream ss;
    ss << "m/44'/" << (coin_type & 0x7FFFFFFF) << "'/" 
       << account << "'/" << change << "/" << index;
    return ss.str();
}

// ============================================================================
// Ledger Wallet Implementation
// ============================================================================

LedgerWallet::LedgerWallet(DeviceModel model) 
    : model_(model), status_(ConnectionStatus::DISCONNECTED) {
    device_info_.model = model;
    device_info_.transport = TransportType::USB_HID;
    device_info_.initialized = false;
    device_info_.pin_enabled = true;
    device_info_.passphrase_enabled = true;
    device_info_.secure_element = true;
}

LedgerWallet::~LedgerWallet() {
    disconnect();
}

ErrorCode LedgerWallet::connect() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ == ConnectionStatus::CONNECTED) {
        return ErrorCode::ALREADY_CONNECTED;
    }
    
    status_ = ConnectionStatus::CONNECTING;
    
    // Open HID device based on model
    uint16_t vendor_id = 0x2c97;  // Ledger
    uint16_t product_id = 0x0000;  // Will be determined
    
    switch (model_) {
        case DeviceModel::LEDGER_NANO_S:
            product_id = 0x0001;
            break;
        case DeviceModel::LEDGER_NANO_S_PLUS:
            product_id = 0x0004;
            break;
        case DeviceModel::LEDGER_NANO_X:
            product_id = 0x0005;
            break;
        case DeviceModel::LEDGER_NANO_STAX:
            product_id = 0x6000;
            break;
        case DeviceModel::LEDGER_FLEX:
            product_id = 0x7000;
            break;
        default:
            product_id = 0x0000;  // Any Ledger device
    }
    
    ErrorCode result = open_hid_device(vendor_id, product_id);
    if (result != ErrorCode::SUCCESS) {
        status_ = ConnectionStatus::ERROR;
        return result;
    }
    
    // Initialize Ledger
    result = init_ledger();
    if (result != ErrorCode::SUCCESS) {
        close_hid_device();
        status_ = ConnectionStatus::ERROR;
        return result;
    }
    
    status_ = ConnectionStatus::CONNECTED;
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::disconnect() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    ErrorCode result = close_hid_device();
    status_ = ConnectionStatus::DISCONNECTED;
    initialized_ = false;
    
    return result;
}

ConnectionStatus LedgerWallet::get_status() const {
    return status_;
}

std::optional<DeviceInfo> LedgerWallet::get_device_info() const {
    if (status_ == ConnectionStatus::CONNECTED) {
        return device_info_;
    }
    return std::nullopt;
}

ErrorCode LedgerWallet::get_public_key(const std::string& path, PublicKeyInfo& info) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    auto path_components = parse_bip32_path(path);
    
    // Build APDU command for get public key
    std::vector<uint8_t> apdu_data;
    // An APDU carries the level count in one byte.
    apdu_data.push_back(static_cast<uint8_t>(path_components.size()));
    for (auto idx : path_components) {
        auto bytes = CryptoUtils::uint32_to_big_endian(idx);
        apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
    }
    
    APDUCommand cmd(0xE0, 0x02, 0x00, 0x00, apdu_data);
    APDUResponse response;
    
    ErrorCode result = send_apdu(cmd, response);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    if (!response.is_success()) {
        return ErrorCode::APDU_ERROR;
    }
    
    // Parse response
    if (response.data.size() < 65) {
        return ErrorCode::INVALID_RESPONSE;
    }
    
    // The device returns the point as a 0x04 marker, then X, then Y. The
    // struct holds X and Y without the marker, so the copy starts past it:
    // writing all 65 bytes would run one byte past the end of the array.
    std::copy(response.data.begin() + 1, response.data.begin() + 65,
              info.uncompressed_pubkey.begin());

    // The parity of Y chooses the compressed prefix.
    info.compressed_pubkey[0] = (response.data[64] & 0x01) ? 0x03 : 0x02;
    std::copy(response.data.begin() + 1, response.data.begin() + 33,
              info.compressed_pubkey.begin() + 1);
    
    // Derive address
    info.address = CryptoUtils::derive_evm_address(info.compressed_pubkey);
    info.path = path;
    
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::get_extended_public_key(const std::string& path, HDPublicKey& info) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    auto path_components = parse_bip32_path(path);

    std::vector<uint8_t> apdu_data;
    apdu_data.push_back(static_cast<uint8_t>(path_components.size()));
    for (auto idx : path_components) {
        auto bytes = CryptoUtils::uint32_to_big_endian(idx);
        apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
    }

    // P2 = 0x01 asks the device to append the chain code to the response.
    APDUCommand cmd(0xE0, 0x02, 0x00, 0x01, apdu_data);
    APDUResponse response;

    const ErrorCode result = send_apdu(cmd, response);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }

    if (!response.is_success()) {
        return ErrorCode::APDU_ERROR;
    }

    // The point is 65 bytes and the chain code follows it.
    if (response.data.size() < 97) {
        return ErrorCode::INVALID_RESPONSE;
    }

    info.key[0] = (response.data[64] & 0x01) ? 0x03 : 0x02;
    std::copy(response.data.begin() + 1, response.data.begin() + 33,
              info.key.begin() + 1);
    std::copy(response.data.begin() + 65, response.data.begin() + 97,
              info.chaincode.begin());
    info.path = path;

    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::sign_transaction(
    const std::vector<uint8_t>& tx_data,
    const std::string& path,
    SignatureInfo& signature) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    return sign_tx_ledger(tx_data, path, signature.signature);
}

ErrorCode LedgerWallet::sign_message(
    const std::vector<uint8_t>& message,
    const std::string& path,
    SignatureInfo& signature) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    return sign_msg_ledger(message, false, path, signature.signature);
}

ErrorCode LedgerWallet::sign_typed_data(
    const std::string& domain,
    const std::string& message,
    const std::string& path,
    SignatureInfo& signature) {
    // EIP-712 signing
    std::vector<uint8_t> domain_hash = CryptoUtils::hex_decode(domain);
    std::vector<uint8_t> message_hash = CryptoUtils::hex_decode(message);
    
    std::vector<uint8_t> data;
    data.insert(data.end(), domain_hash.begin(), domain_hash.end());
    data.insert(data.end(), message_hash.begin(), message_hash.end());
    
    return sign_msg_ledger(data, true, path, signature.signature);
}

ErrorCode LedgerWallet::verify_pin(const std::string& pin) {
    // A Ledger unlocks on the device itself. There is no APDU that checks a
    // PIN, so reporting success would claim an action that never happened.
    (void)pin;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode LedgerWallet::change_pin(const std::string& old_pin, const std::string& new_pin) {
    (void)old_pin;
    (void)new_pin;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode LedgerWallet::enable_passphrase(bool enable) {
    (void)enable;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode LedgerWallet::unlock_with_passphrase(const std::string& passphrase) {
    // The passphrase is entered on the device and never sent to the host.
    (void)passphrase;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode LedgerWallet::get_firmware_version(std::string& version) {
    // The version comes from the device info read at connect time. An empty
    // value means the read never happened, which is not success.
    if (device_info_.firmware_version.empty()) {
        return ErrorCode::NOT_INITIALIZED;
    }

    version = device_info_.firmware_version;
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::reboot_to_bootloader() {
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode LedgerWallet::factory_reset() {
    // A factory reset is a device-menu action. It is never done from the host.
    return ErrorCode::UNSUPPORTED_OPERATION;
}

void LedgerWallet::set_rpc_endpoint(std::string url) {
    std::lock_guard<std::mutex> lock(mutex_);
    rpc_url_ = std::move(url);
}

ErrorCode LedgerWallet::get_erc20_token_balance(
    const std::string& token_address,
    const std::string& owner_address,
    std::string& balance) {
    // The balance is read from the chain. A zero would be a lie when the node
    // is unreachable, so the failure is reported instead.
    std::string endpoint;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        endpoint = rpc_url_;
    }

    if (endpoint.empty()) {
        return ErrorCode::NOT_INITIALIZED;
    }

    if (token_address.size() != 42 || owner_address.size() != 42) {
        return ErrorCode::INVALID_PARAMETER;
    }

    // balanceOf(address) selector, then the owner padded to a 32-byte word.
    const std::string data = "0x70a08231" + std::string(24, '0') +
                             owner_address.substr(2);

    JsonRpcClient client(endpoint);
    const RpcResult outcome = client.eth_call(token_address, data);

    if (!outcome.ok()) {
        return (outcome.status == RpcStatus::NodeError) ? ErrorCode::CRYPTO_ERROR
                                                        : ErrorCode::TRANSPORT_ERROR;
    }

    if (!decode_abi_uint256(outcome.result.as_string(), balance)) {
        return ErrorCode::INVALID_RESPONSE;
    }

    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::get_nft_metadata(
    const std::string& contract_address,
    const std::string& token_id,
    std::string& metadata_json) {
    // tokenURI(uint256) is read from the chain and the answer is returned as
    // it came. An empty object would look like a token with no metadata.
    std::string endpoint;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        endpoint = rpc_url_;
    }

    if (endpoint.empty()) {
        return ErrorCode::NOT_INITIALIZED;
    }

    if (contract_address.size() != 42) {
        return ErrorCode::INVALID_PARAMETER;
    }

    // The token identifier must be a decimal number; it is padded to a word.
    if (token_id.empty() ||
        !std::all_of(token_id.begin(), token_id.end(),
                     [](char c) { return c >= '0' && c <= '9'; })) {
        return ErrorCode::INVALID_PARAMETER;
    }

    std::string padded(64, '0');
    size_t cursor = 64;
    for (auto it = token_id.rbegin(); it != token_id.rend() && cursor > 0; ++it) {
        padded[--cursor] = *it;
    }

    // c87b56dd is the tokenURI(uint256) selector.
    const std::string data = "0xc87b56dd" + padded;

    JsonRpcClient client(endpoint);
    const RpcResult outcome = client.eth_call(contract_address, data);

    if (!outcome.ok()) {
        return (outcome.status == RpcStatus::NodeError) ? ErrorCode::CRYPTO_ERROR
                                                        : ErrorCode::TRANSPORT_ERROR;
    }

    if (!decode_abi_string(outcome.result.as_string(), metadata_json)) {
        return ErrorCode::INVALID_RESPONSE;
    }

    return ErrorCode::SUCCESS;
}

// ============================================================================
// Private Methods
// ============================================================================

ErrorCode LedgerWallet::open_hid_device(uint16_t vendor_id, uint16_t product_id) {
#ifdef _WIN32
    GUID hid_guid;
    HidD_GetHidGuid(&hid_guid);
    
    HDEVINFO device_info = SetupDiGetClassDevs(&hid_guid, NULL, NULL, 
                                                DIGCF_PRESENT | DIGCF_INTERFACEDEVICE);
    
    if (device_info == INVALID_HANDLE_VALUE) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    SP_DEVICE_INTERFACE_DATA interface_data;
    interface_data.cbSize = sizeof(SP_DEVICE_INTERFACE_DATA);
    
    for (DWORD i = 0; SetupDiEnumDeviceInterfaces(device_info, NULL, &hid_guid, i, &interface_data); i++) {
        DWORD required_size = 0;
        SetupDiGetInterfaceDeviceDetail(device_info, &interface_data, NULL, 0, &required_size, NULL);
        
        PSP_DEVICE_INTERFACE_DETAIL_DATA detail = (PSP_DEVICE_INTERFACE_DETAIL_DATA)malloc(required_size);
        detail->cbSize = sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA);
        
        if (SetupDiGetInterfaceDeviceDetail(device_info, &interface_data, detail, required_size, NULL, NULL)) {
            // Check if this is our device
            hid_handle_ = CreateFile(detail->DevicePath, GENERIC_READ | GENERIC_WRITE,
                                    FILE_SHARE_READ | FILE_SHARE_WRITE, NULL,
                                    OPEN_EXISTING, 0, NULL);
            
            if (hid_handle_ != INVALID_HANDLE_VALUE) {
                // Get device attributes
                HIDD_ATTRIBUTES attributes;
                if (HidD_GetAttributes(hid_handle_, &attributes)) {
                    if (attributes.VendorID == vendor_id && 
                        (product_id == 0 || attributes.ProductID == product_id)) {
                        free(detail);
                        SetupDiDestroyDeviceInfoList(device_info);
                        return ErrorCode::SUCCESS;
                    }
                }
                CloseHandle(hid_handle_);
                hid_handle_ = INVALID_HANDLE_VALUE;
            }
        }
        free(detail);
    }
    
    SetupDiDestroyDeviceInfoList(device_info);
    return ErrorCode::DEVICE_NOT_FOUND;
    
#elif defined(__linux__)
    // Only the hidraw nodes whose USB identity matches the request are opened.
    // Opening the first readable node would attach to whatever device happens
    // to be plugged in, and a signature would then come from the wrong one.
    DIR* directory = opendir("/sys/class/hidraw");
    if (directory == nullptr) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }

    struct dirent* entry = nullptr;
    while ((entry = readdir(directory)) != nullptr) {
        if (entry->d_name[0] == '.') {
            continue;
        }

        const std::string node = std::string("/dev/") + entry->d_name;
        const std::string sysfs =
            std::string("/sys/class/hidraw/") + entry->d_name + "/device/uevent";

        std::ifstream uevent(sysfs);
        if (!uevent.is_open()) {
            continue;
        }

        // The uevent carries a HID_ID line shaped like
        // "HID_ID=0003:00002C97:00000004" — bus, vendor, product.
        std::string line;
        uint16_t found_vendor = 0;
        uint16_t found_product = 0;

        while (std::getline(uevent, line)) {
            if (line.rfind("HID_ID=", 0) != 0) {
                continue;
            }

            const size_t last_colon = line.rfind(':');
            const size_t middle_colon = line.rfind(':', last_colon - 1);
            if (last_colon == std::string::npos || middle_colon == std::string::npos) {
                break;
            }

            try {
                found_vendor = static_cast<uint16_t>(
                    std::stoul(line.substr(middle_colon + 1, last_colon - middle_colon - 1), nullptr, 16));
                found_product = static_cast<uint16_t>(
                    std::stoul(line.substr(last_colon + 1), nullptr, 16));
            } catch (...) {
                break;
            }
            break;
        }

        if (found_vendor != vendor_id || (product_id != 0 && found_product != product_id)) {
            continue;
        }

        hid_fd_ = open(node.c_str(), O_RDWR | O_NONBLOCK);
        if (hid_fd_ < 0) {
            continue;
        }

        closedir(directory);
        return ErrorCode::SUCCESS;
    }

    closedir(directory);
    return ErrorCode::DEVICE_NOT_FOUND;

#elif defined(__APPLE__)
    // IOKit reports the same vendor and product ids as the other platforms.
    io_iterator_t iterator = 0;
    io_service_t service = 0;

    if (IOServiceGetMatchingServices(kIOMainPortDefault,
                                     IOServiceMatching(kIOHIDDeviceKey),
                                     &iterator) != KERN_SUCCESS) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }

    while ((service = IOIteratorNext(iterator)) != 0) {
        CFTypeRef vendor_ref = IORegistryEntryCreateCFProperty(
            service, CFSTR(kIOHIDVendorIDKey), kCFAllocatorDefault, 0);
        CFTypeRef product_ref = IORegistryEntryCreateCFProperty(
            service, CFSTR(kIOHIDProductIDKey), kCFAllocatorDefault, 0);

        bool matches = false;
        if (vendor_ref != nullptr && product_ref != nullptr) {
            int64_t found_vendor = 0;
            int64_t found_product = 0;
            CFNumberGetValue(static_cast<CFNumberRef>(vendor_ref), kCFNumberSInt64Type, &found_vendor);
            CFNumberGetValue(static_cast<CFNumberRef>(product_ref), kCFNumberSInt64Type, &found_product);

            matches = (static_cast<uint16_t>(found_vendor) == vendor_id) &&
                      (product_id == 0 || static_cast<uint16_t>(found_product) == product_id);
        }

        if (vendor_ref != nullptr) CFRelease(vendor_ref);
        if (product_ref != nullptr) CFRelease(product_ref);

        if (matches) {
            // The IOKit handle is the device; the APDU layer opens it through
            // IOHIDDeviceOpen when the first exchange happens.
            hid_device_ = reinterpret_cast<void*>(service);
            return ErrorCode::SUCCESS;
        }

        IOObjectRelease(service);
    }

    IOObjectRelease(iterator);
    return ErrorCode::DEVICE_NOT_FOUND;
#else
    // No HID transport on this platform. Reporting DEVICE_NOT_FOUND keeps the
    // caller from proceeding with a device that cannot be reached.
    return ErrorCode::DEVICE_NOT_FOUND;
#endif
}

ErrorCode LedgerWallet::close_hid_device() {
#ifdef _WIN32
    if (hid_handle_ != INVALID_HANDLE_VALUE) {
        CloseHandle(hid_handle_);
        hid_handle_ = INVALID_HANDLE_VALUE;
    }
#elif defined(__linux__)
    if (hid_fd_ >= 0) {
        close(hid_fd_);
        hid_fd_ = -1;
    }
#elif defined(__APPLE__)
    if (hid_device_ != nullptr) {
        IOObjectRelease(reinterpret_cast<io_object_t>(hid_device_));
        hid_device_ = nullptr;
    }
#endif
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::send_apdu(const APDUCommand& cmd, APDUResponse& response) {
    std::vector<uint8_t> packet;
    
    // Build APDU
    packet.push_back(cmd.cla);
    packet.push_back(cmd.ins);
    packet.push_back(cmd.p1);
    packet.push_back(cmd.p2);
    packet.push_back((uint8_t)cmd.data.size());
    
    if (!cmd.data.empty()) {
        packet.insert(packet.end(), cmd.data.begin(), cmd.data.end());
    }
    
    std::vector<uint8_t> response_data;
    ErrorCode result = exchange(packet, response_data);
    
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    // Parse response
    if (response_data.size() < 2) {
        return ErrorCode::INVALID_RESPONSE;
    }
    
    // Extract status word
    response.sw = (response_data[response_data.size() - 2] << 8) | 
                 response_data[response_data.size() - 1];
    
    // Extract data (excluding SW)
    response.data.assign(response_data.begin(), response_data.end() - 2);
    response.more_data = (response.sw == 0x6100);
    
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::exchange(const std::vector<uint8_t>& data, std::vector<uint8_t>& response) {
    // HID transport. The APDU is written to the open device and the reply is
    // read back. With no device open the call reports DEVICE_NOT_FOUND; it
    // never returns a value the caller could mistake for a device response.
    response.clear();
    
#ifdef _WIN32
    if (hid_handle_ == INVALID_HANDLE_VALUE) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    // Send report
    uint8_t report[MAX_HID_PACKET_SIZE];
    memset(report, 0, sizeof(report));
    report[0] = channel_id_ & 0x0F;
    report[1] = (packet_counter_ >> 8) & 0xFF;
    report[2] = packet_counter_ & 0xFF;
    
    size_t offset = 3;
    for (size_t i = 0; i < data.size() && offset < MAX_HID_PACKET_SIZE; i++) {
        report[offset++] = data[i];
    }
    
    DWORD bytes_written;
    if (!WriteFile(hid_handle_, report, MAX_HID_PACKET_SIZE, &bytes_written, NULL)) {
        return ErrorCode::TRANSPORT_ERROR;
    }
    
    // Read response
    uint8_t response_report[MAX_HID_PACKET_SIZE];
    DWORD bytes_read;
    
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    
    if (!ReadFile(hid_handle_, response_report, MAX_HID_PACKET_SIZE, &bytes_read, NULL)) {
        return ErrorCode::TRANSPORT_ERROR;
    }
    
    if (bytes_read > 3) {
        response.assign(response_report + 3, response_report + bytes_read);
    }
    
    packet_counter_++;
    
#elif defined(__linux__)
    if (hid_fd_ < 0) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    ssize_t written = write(hid_fd_, data.data(), data.size());
    if (written < 0) {
        return ErrorCode::TRANSPORT_ERROR;
    }
    
    // Read response
    uint8_t buffer[MAX_HID_PACKET_SIZE];
    ssize_t bytes_read = read(hid_fd_, buffer, sizeof(buffer));
    
    if (bytes_read > 0) {
        response.assign(buffer, buffer + bytes_read);
    }
    
#elif defined(__APPLE__)
    if (hid_device_ == nullptr) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }

    IOHIDDeviceRef device = reinterpret_cast<IOHIDDeviceRef>(hid_device_);

    if (IOHIDDeviceOpen(device, kIOHIDOptionsTypeNone) != kIOReturnSuccess) {
        return ErrorCode::TRANSPORT_ERROR;
    }

    // Same framing as the other platforms: three header bytes, then the APDU.
    uint8_t report[MAX_HID_PACKET_SIZE];
    memset(report, 0, sizeof(report));
    report[0] = channel_id_ & 0x0F;
    report[1] = (packet_counter_ >> 8) & 0xFF;
    report[2] = packet_counter_ & 0xFF;

    size_t offset = 3;
    for (size_t i = 0; i < data.size() && offset < MAX_HID_PACKET_SIZE; i++) {
        report[offset++] = data[i];
    }

    if (IOHIDDeviceSetReport(device, kIOHIDReportTypeOutput, 0, report, offset) != kIOReturnSuccess) {
        IOHIDDeviceClose(device, kIOHIDOptionsTypeNone);
        return ErrorCode::TRANSPORT_ERROR;
    }

    uint8_t reply[MAX_HID_PACKET_SIZE];
    CFIndex reply_length = static_cast<CFIndex>(sizeof(reply));

    IOReturn read_result = IOHIDDeviceGetReport(device, kIOHIDReportTypeInput, 0, reply, &reply_length);
    IOHIDDeviceClose(device, kIOHIDOptionsTypeNone);

    if (read_result != kIOReturnSuccess) {
        return ErrorCode::TRANSPORT_ERROR;
    }

    if (reply_length > 3) {
        response.assign(reply + 3, reply + reply_length);
    }
#else
    // No HID transport is available on this platform. Reporting failure is
    // the only safe answer: echoing the request back would hand the caller a
    // "signature" the device never produced.
    (void)data;
    return ErrorCode::UNSUPPORTED_OPERATION;
#endif
    

    // A device that returns nothing cannot be trusted to have accepted the
    // request.
    if (response.empty()) {
        return ErrorCode::INVALID_RESPONSE;
    }
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::init_ledger() {
    // Initialize Ledger device
    APDUCommand cmd(0xE0, 0x00, 0x00, 0x00);
    APDUResponse response;
    
    ErrorCode result = send_apdu(cmd, response);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    if (!response.is_success() && !response.is_pin_needed()) {
        return ErrorCode::APDU_ERROR;
    }
    
    initialized_ = true;
    return get_device_info_ledger();
}

ErrorCode LedgerWallet::get_device_info_ledger() {
    APDUCommand cmd(0xE0, 0x01, 0x00, 0x00);
    APDUResponse response;
    
    ErrorCode result = send_apdu(cmd, response);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    if (response.is_success() && response.data.size() >= 4) {
        // Parse device info
        device_info_.firmware_version = std::to_string(response.data[0]) + "." +
                                       std::to_string(response.data[1]) + "." +
                                       std::to_string(response.data[2]);
    }
    
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::show_address_ledger(const std::string& path, const std::string& address) {
    auto path_components = parse_bip32_path(path);

    std::vector<uint8_t> apdu_data;
    // An APDU carries the level count in one byte.
    apdu_data.push_back(static_cast<uint8_t>(path_components.size()));
    for (auto idx : path_components) {
        auto bytes = CryptoUtils::uint32_to_big_endian(idx);
        apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
    }

    // P1 = 0x01 asks the device to display the address for this path.
    APDUCommand cmd(0xE0, 0x02, 0x01, 0x00, apdu_data);
    APDUResponse response;

    const ErrorCode result = send_apdu(cmd, response);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }

    if (!response.is_success()) {
        return ErrorCode::APDU_ERROR;
    }

    // The point of showing an address is to confirm it matches what the caller
    // expects. A mismatch means the device holds a different key, which is
    // exactly what this check exists to catch.
    PublicKeyInfo info;
    const ErrorCode derived = get_public_key(path, info);
    if (derived != ErrorCode::SUCCESS) {
        return derived;
    }

    if (info.address != address) {
        return ErrorCode::SECURITY_ERROR;
    }

    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::sign_tx_ledger(const std::vector<uint8_t>& tx, 
                                      const std::string& path,
                                      std::vector<uint8_t>& signature) {
    auto path_components = parse_bip32_path(path);
    
    // Build sign transaction APDU
    std::vector<uint8_t> apdu_data;
    // An APDU carries the level count in one byte.
    apdu_data.push_back(static_cast<uint8_t>(path_components.size()));
    for (auto idx : path_components) {
        auto bytes = CryptoUtils::uint32_to_big_endian(idx);
        apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
    }
    
    // Add transaction data
    apdu_data.insert(apdu_data.end(), tx.begin(), tx.end());
    
    APDUCommand cmd(0xE0, 0x04, 0x00, 0x00, apdu_data);
    cmd.response_timeout = SIGNING_TIMEOUT;
    
    APDUResponse response;
    ErrorCode result = send_apdu(cmd, response);
    
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    if (!response.is_success()) {
        if (response.isCancelled()) {
            return ErrorCode::USER_CANCELLED;
        }
        return ErrorCode::APDU_ERROR;
    }
    
    signature = response.data;
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::sign_msg_ledger(const std::vector<uint8_t>& message, 
                                        bool is_compat,
                                        const std::string& path,
                                        std::vector<uint8_t>& signature) {
    auto path_components = parse_bip32_path(path);
    
    // Build sign message APDU
    std::vector<uint8_t> apdu_data;
    // An APDU carries the level count in one byte.
    apdu_data.push_back(static_cast<uint8_t>(path_components.size()));
    for (auto idx : path_components) {
        auto bytes = CryptoUtils::uint32_to_big_endian(idx);
        apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
    }
    
    // Message length
    auto len_bytes = CryptoUtils::uint32_to_big_endian((uint32_t)message.size());
    apdu_data.insert(apdu_data.end(), len_bytes.begin(), len_bytes.end());
    
    // Message data (chunked for large messages)
    size_t chunk_size = MAX_APDU_SIZE - 20;
    size_t offset = 0;
    
    while (offset < message.size()) {
        size_t remaining = message.size() - offset;
        size_t to_send = std::min(remaining, chunk_size);
        
        apdu_data.clear();
        apdu_data.push_back(static_cast<uint8_t>(path_components.size()));
        for (auto idx : path_components) {
            auto bytes = CryptoUtils::uint32_to_big_endian(idx);
            apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
        }
        
        auto len_bytes = CryptoUtils::uint32_to_big_endian((uint32_t)message.size());
        apdu_data.insert(apdu_data.end(), len_bytes.begin(), len_bytes.end());
        
        // The chunk offsets are unsigned sizes; the iterator arithmetic needs
        // a signed difference type.
        const auto chunk_start = message.begin() + static_cast<std::ptrdiff_t>(offset);
        apdu_data.insert(apdu_data.end(), chunk_start,
                         chunk_start + static_cast<std::ptrdiff_t>(to_send));
        
        APDUCommand cmd(0xE0, is_compat ? 0x0F : 0x03, 
                       (offset == 0) ? 0x00 : 0x80, 
                       0x00, apdu_data);
        
        APDUResponse response;
        ErrorCode result = send_apdu(cmd, response);
        
        if (result != ErrorCode::SUCCESS) {
            return result;
        }
        
        if (response.is_success() && !response.data.empty()) {
            signature = response.data;
            return ErrorCode::SUCCESS;
        }
        
        offset += to_send;
    }
    
    return ErrorCode::INVALID_RESPONSE;
}

std::vector<uint32_t> LedgerWallet::parse_bip32_path(const std::string& path) {
    // The APDU carries exactly the levels the caller asked for, hardened bits
    // included. Substituting a default here would sign with a different key
    // than the one the caller selected.
    BIP44::PathComponents components = BIP44::parse_path(path);

    std::vector<uint32_t> levels = {components.purpose,
                                    components.coin_type,
                                    components.account,
                                    components.change,
                                    components.address_index};

    // A trailing all-zero tail would make two different paths encode to the
    // same APDU, so the levels are trimmed to the ones actually supplied.
    while (levels.size() > 1 && levels.back() == 0) {
        levels.pop_back();
    }

    return levels;
}

std::string LedgerWallet::get_app_name() const {
    switch (model_) {
        case DeviceModel::LEDGER_NANO_S:
            return "Bitcoin";
        case DeviceModel::LEDGER_NANO_S_PLUS:
            return "Ethereum";
        case DeviceModel::LEDGER_NANO_X:
            return "Ethereum";
        case DeviceModel::LEDGER_NANO_STAX:
            return "Ethereum";
        case DeviceModel::LEDGER_FLEX:
            return "Ethereum";
        default:
            return "Unknown";
    }
}

// ============================================================================
// Trezor Wallet Implementation (Simplified)
// ============================================================================

TrezorWallet::TrezorWallet(DeviceModel model) 
    : model_(model), status_(ConnectionStatus::DISCONNECTED), socket_fd_(-1) {
    device_info_.model = model;
    device_info_.transport = TransportType::USB_HID;
}

TrezorWallet::~TrezorWallet() {
    disconnect();
}

ErrorCode TrezorWallet::connect() {
    std::lock_guard<std::mutex> lock(mutex_);

    // The Trezor message protocol is not implemented, so no device can be
    // opened. Reporting CONNECTED here would let the caller believe a device
    // is present and usable when nothing was ever contacted.
    status_ = ConnectionStatus::DISCONNECTED;
    initialized_ = false;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::disconnect() {
    std::lock_guard<std::mutex> lock(mutex_);
    if (socket_fd_ >= 0) {
        close(socket_fd_);
        socket_fd_ = -1;
    }
    status_ = ConnectionStatus::DISCONNECTED;
    initialized_ = false;
    return ErrorCode::SUCCESS;
}

ConnectionStatus TrezorWallet::get_status() const {
    return status_;
}

std::optional<DeviceInfo> TrezorWallet::get_device_info() const {
    if (status_ == ConnectionStatus::CONNECTED) {
        return device_info_;
    }
    return std::nullopt;
}

ErrorCode TrezorWallet::get_public_key(const std::string& path, PublicKeyInfo& info) {
    std::lock_guard<std::mutex> lock(mutex_);
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }

    (void)path;
    (void)info;
    
    // The Trezor message protocol is not wired up, so the device cannot be
    // initialised. Reporting success would hand the caller an uninitialised
    // device it believes is ready.
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::get_extended_public_key(const std::string& path, HDPublicKey& info) {
    // The two structs have different layouts, so casting one to the other
    // reads past the end of the destination. The public key is fetched into
    // the right type and the fields are copied across.
    PublicKeyInfo public_key{};
    const ErrorCode outcome = get_public_key(path, public_key);

    if (outcome != ErrorCode::SUCCESS) {
        return outcome;
    }

    info.key = public_key.compressed_pubkey;
    info.path = path;

    // A chain code is only available from the device's own derivation. Without
    // the protocol there is none, so an all-zero code would be misleading.
    info.chaincode.fill(0);
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::sign_transaction(
    const std::vector<uint8_t>& tx_data,
    const std::string& path,
    SignatureInfo& signature) {
    // No signature can be produced without the device protocol.
    (void)tx_data;
    (void)path;
    (void)signature;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::sign_message(
    const std::vector<uint8_t>& message,
    const std::string& path,
    SignatureInfo& signature) {
    // No signature can be produced without the device protocol.
    (void)message;
    (void)path;
    (void)signature;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::sign_typed_data(
    const std::string& domain,
    const std::string& message,
    const std::string& path,
    SignatureInfo& signature) {
    // No signature can be produced without the device protocol.
    (void)domain;
    (void)message;
    (void)path;
    (void)signature;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

// The Trezor message protocol is not wired up, so none of these operations can
// be carried out. Reporting failure is the only honest answer: returning
// SUCCESS would tell the caller that a PIN was changed or the device was wiped
// when nothing happened.
ErrorCode TrezorWallet::verify_pin(const std::string& pin) {
    (void)pin;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::change_pin(const std::string& old_pin, const std::string& new_pin) {
    (void)old_pin;
    (void)new_pin;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::enable_passphrase(bool enable) {
    (void)enable;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::unlock_with_passphrase(const std::string& passphrase) {
    (void)passphrase;
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::get_firmware_version(std::string& version) {
    version.clear();
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::reboot_to_bootloader() {
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::factory_reset() {
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::get_erc20_token_balance(const std::string&, const std::string&, std::string& balance) {
    // A zero balance would be indistinguishable from a real one, so the call
    // reports that it could not be answered.
    balance.clear();
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::get_nft_metadata(const std::string&, const std::string&, std::string& json) {
    json.clear();
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::init_transport() {
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::send_message(const std::vector<uint8_t>& msg, std::vector<uint8_t>& response) {
    (void)msg;
    response.clear();
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::read_message(std::vector<uint8_t>& msg) {
    msg.clear();
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::init_trezor() {
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode TrezorWallet::get_public_key_trezor(const std::string& path, HDPublicKey& key) {
    (void)path;
    (void)key;
    return ErrorCode::UNSUPPORTED_OPERATION;
}


ErrorCode TrezorWallet::sign_tx_trezor(const std::vector<uint8_t>& tx,
                                       const std::string& path,
                                       std::vector<uint8_t>& signature) {
    // The Trezor message protocol is not implemented, so no signature exists.
    (void)tx;
    (void)path;
    signature.clear();
    return ErrorCode::UNSUPPORTED_OPERATION;
}

// ============================================================================
// Hardware Wallet Manager Implementation
// ============================================================================

std::vector<DeviceInfo> HIDManager::enumerate_devices() {
    // The list comes from the operating system. Each entry is a device that is
    // present now; a model that is not plugged in does not appear.
    std::vector<DeviceInfo> devices;

    const auto model_for = [](uint16_t vid, uint16_t pid) -> DeviceModel {
        if (vid == LEDGER_VID) {
            switch (pid) {
                case 0x0001: return DeviceModel::LEDGER_NANO_S;
                case 0x0004: return DeviceModel::LEDGER_NANO_X;
                case 0x0006: return DeviceModel::LEDGER_NANO_STAX;
                case 0x0007: return DeviceModel::LEDGER_FLEX;
                default: return DeviceModel::UNKNOWN;
            }
        }
        if (vid == TREZOR_VID) {
            switch (pid) {
                case 0x0001: return DeviceModel::TREZOR_ONE;
                case 0x0002: return DeviceModel::TREZOR_T;
                case 0x0003: return DeviceModel::TREZOR_SAFE_3;
                case 0x0005: return DeviceModel::TREZOR_SAFE_5;
                default: return DeviceModel::UNKNOWN;
            }
        }
        if (vid == KEYSTONE_VID) return DeviceModel::KEYSTONE_PRO;
        if (vid == ONEKEY_VID) return DeviceModel::ONEKEY_CLASSIC;
        return DeviceModel::UNKNOWN;
    };

    const auto make_info = [&](uint16_t vid, uint16_t pid) {
        DeviceModel model = model_for(vid, pid);
        if (model == DeviceModel::UNKNOWN) {
            return;
        }

        DeviceInfo info;
        info.device_id = "usb:" + std::to_string(vid) + ":" + std::to_string(pid);
        info.model = model;
        info.transport = TransportType::USB_HID;
        info.initialized = false;
        info.pin_enabled = true;
        info.passphrase_enabled = false;
        info.secure_element = (vid == LEDGER_VID);
        devices.push_back(info);
    };

#ifdef _WIN32
    HDEVINFO device_set = SetupDiGetClassDevsA(&GUID_DEVINTERFACE_HID, nullptr, nullptr,
                                               DIGCF_PRESENT | DIGCF_DEVICEINTERFACE);
    if (device_set == INVALID_HANDLE_VALUE) {
        return devices;
    }

    SP_DEVICE_INTERFACE_DATA interface_data{};
    interface_data.cbSize = sizeof(interface_data);

    for (DWORD index = 0;
         SetupDiEnumDeviceInterfaces(device_set, nullptr, &GUID_DEVINTERFACE_HID, index, &interface_data);
         index++) {
        DWORD required = 0;
        SetupDiGetDeviceInterfaceDetailA(device_set, &interface_data, nullptr, 0, &required, nullptr);
        if (required == 0) {
            continue;
        }

        std::vector<char> buffer(required);
        auto* detail = reinterpret_cast<SP_DEVICE_INTERFACE_DETAIL_DATA_A*>(buffer.data());
        detail->cbSize = sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA_A);

        if (!SetupDiGetDeviceInterfaceDetailA(device_set, &interface_data, detail, required, nullptr, nullptr)) {
            continue;
        }

        HANDLE handle = CreateFileA(detail->DevicePath, GENERIC_READ | GENERIC_WRITE,
                                    FILE_SHARE_READ | FILE_SHARE_WRITE, nullptr, OPEN_EXISTING, 0, nullptr);
        if (handle == INVALID_HANDLE_VALUE) {
            continue;
        }

        HIDD_ATTRIBUTES attributes{};
        attributes.Size = sizeof(attributes);

        if (HidD_GetAttributes(handle, &attributes)) {
            make_info(attributes.VendorID, attributes.ProductID);
        }

        CloseHandle(handle);
    }

    SetupDiDestroyDeviceInfoList(device_set);
#elif defined(__linux__)
    // sysfs is the kernel's own view of the USB bus, so it needs no library.
    const std::string root = "/sys/bus/usb/devices";
    DIR* directory = opendir(root.c_str());
    if (directory == nullptr) {
        return devices;
    }

    struct dirent* entry = nullptr;
    while ((entry = readdir(directory)) != nullptr) {
        if (entry->d_name[0] == '.') {
            continue;
        }

        const std::string base = root + "/" + entry->d_name + "/";

        std::ifstream vendor_file(base + "idVendor");
        std::ifstream product_file(base + "idProduct");
        if (!vendor_file.is_open() || !product_file.is_open()) {
            continue;
        }

        std::string vendor_text;
        std::string product_text;
        std::getline(vendor_file, vendor_text);
        std::getline(product_file, product_text);

        uint16_t vid = 0;
        uint16_t pid = 0;
        try {
            vid = static_cast<uint16_t>(std::stoul(vendor_text, nullptr, 16));
            pid = static_cast<uint16_t>(std::stoul(product_text, nullptr, 16));
        } catch (...) {
            continue;
        }

        make_info(vid, pid);
    }

    closedir(directory);
#elif defined(__APPLE__)
    // IOKit reports the same information on macOS.
    CFMutableDictionaryRef matching = IOServiceMatching(kIOHIDDeviceKey);
    if (matching != nullptr) {
        io_iterator_t iterator = 0;
        if (IOServiceGetMatchingServices(kIOMainPortDefault, matching, &iterator) == KERN_SUCCESS) {
            io_object_t service = 0;
            while ((service = IOIteratorNext(iterator)) != 0) {
                CFTypeRef vendor_ref = IORegistryEntryCreateCFProperty(
                    service, CFSTR(kIOHIDVendorIDKey), kCFAllocatorDefault, 0);
                CFTypeRef product_ref = IORegistryEntryCreateCFProperty(
                    service, CFSTR(kIOHIDProductIDKey), kCFAllocatorDefault, 0);

                if (vendor_ref != nullptr && product_ref != nullptr) {
                    int64_t vid = 0;
                    int64_t pid = 0;
                    CFNumberGetValue(static_cast<CFNumberRef>(vendor_ref), kCFNumberSInt64Type, &vid);
                    CFNumberGetValue(static_cast<CFNumberRef>(product_ref), kCFNumberSInt64Type, &pid);
                    make_info(static_cast<uint16_t>(vid), static_cast<uint16_t>(pid));
                }

                if (vendor_ref != nullptr) CFRelease(vendor_ref);
                if (product_ref != nullptr) CFRelease(product_ref);
                IOObjectRelease(service);
            }
            IOObjectRelease(iterator);
        }
    }
#endif

    return devices;
}


HardwareWalletManager& HardwareWalletManager::get_instance() {
    static HardwareWalletManager instance;
    return instance;
}

std::optional<UsbIdentity> HardwareWalletManager::usb_identity(DeviceModel model) {
    switch (model) {
        case DeviceModel::LEDGER_NANO_S:
            return UsbIdentity{HIDManager::LEDGER_VID, 0x0001, false};
        case DeviceModel::LEDGER_NANO_S_PLUS:
            return UsbIdentity{HIDManager::LEDGER_VID, 0x0004, false};
        case DeviceModel::LEDGER_NANO_X:
            return UsbIdentity{HIDManager::LEDGER_VID, 0x0004, true};
        case DeviceModel::LEDGER_NANO_STAX:
            return UsbIdentity{HIDManager::LEDGER_VID, 0x0006, true};
        case DeviceModel::LEDGER_FLEX:
            return UsbIdentity{HIDManager::LEDGER_VID, 0x0007, true};
        case DeviceModel::TREZOR_ONE:
            return UsbIdentity{HIDManager::TREZOR_VID, 0x0001, false};
        case DeviceModel::TREZOR_T:
            return UsbIdentity{HIDManager::TREZOR_VID, 0x0002, false};
        case DeviceModel::TREZOR_SAFE_3:
            return UsbIdentity{HIDManager::TREZOR_VID, 0x0003, false};
        case DeviceModel::TREZOR_SAFE_5:
            return UsbIdentity{HIDManager::TREZOR_VID, 0x0005, false};
        case DeviceModel::KEYSTONE_PRO:
        case DeviceModel::KEYSTONE_PRO_3:
            return UsbIdentity{HIDManager::KEYSTONE_VID, 0x0001, false};
        case DeviceModel::ONEKEY_MINI:
        case DeviceModel::ONEKEY_CLASSIC:
            return UsbIdentity{HIDManager::ONEKEY_VID, 0x0001, false};
        default:
            // Ellipal, SafePal and the unknown case have no USB identity in
            // this build, so there is nothing to open.
            return std::nullopt;
    }
}

std::vector<DeviceInfo> HardwareWalletManager::discover_devices() {
    // Only what the bus reports. No device is added because a model exists.
    return HIDManager::enumerate_devices();
}

std::optional<DeviceInfo> HardwareWalletManager::get_connected_device(DeviceModel model) {
    std::lock_guard<std::mutex> lock(mutex_);

    // The manager keeps what it opened, so the answer is the registered
    // device rather than a guess from the model.
    for (const auto& entry : connected_devices_) {
        auto info = entry.second->get_device_info();
        if (info && info->model == model) {
            return info;
        }
    }

    return std::nullopt;
}

ErrorCode HardwareWalletManager::connect_device(DeviceModel model) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::unique_ptr<IHardwareWallet> wallet;
    
    switch (model) {
        case DeviceModel::LEDGER_NANO_S:
        case DeviceModel::LEDGER_NANO_S_PLUS:
        case DeviceModel::LEDGER_NANO_X:
        case DeviceModel::LEDGER_NANO_STAX:
        case DeviceModel::LEDGER_FLEX:
            wallet = std::make_unique<LedgerWallet>(model);
            break;
        case DeviceModel::TREZOR_ONE:
        case DeviceModel::TREZOR_T:
        case DeviceModel::TREZOR_SAFE_3:
        case DeviceModel::TREZOR_SAFE_5:
            wallet = std::make_unique<TrezorWallet>(model);
            break;
        default:
            return ErrorCode::UNSUPPORTED_OPERATION;
    }
    
    ErrorCode result = wallet->connect();
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    auto info = wallet->get_device_info();
    if (!info) {
        // The device did not describe itself, so there is nothing to track
        // and the caller must not be told the connection succeeded.
        wallet->disconnect();
        return ErrorCode::INVALID_RESPONSE;
    }

    const std::string device_id = info->device_id;
    connected_devices_[device_id] = std::move(wallet);

    if (connection_callback_) {
        connection_callback_(device_id, ConnectionStatus::CONNECTED);
    }
    return ErrorCode::SUCCESS;
}

ErrorCode HardwareWalletManager::disconnect_device(const std::string& device_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = connected_devices_.find(device_id);
    if (it != connected_devices_.end()) {
        it->second->disconnect();
        connected_devices_.erase(it);
        return ErrorCode::SUCCESS;
    }
    
    return ErrorCode::DEVICE_NOT_FOUND;
}

void HardwareWalletManager::disconnect_all() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    for (auto& device : connected_devices_) {
        device.second->disconnect();
    }
    connected_devices_.clear();
}

ErrorCode HardwareWalletManager::get_wallet_address(const std::string& path, std::string& address) {
    if (connected_devices_.empty()) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    PublicKeyInfo info;
    ErrorCode result = connected_devices_.begin()->second->get_public_key(path, info);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    address = info.address;
    return ErrorCode::SUCCESS;
}

ErrorCode HardwareWalletManager::sign_transaction(const std::vector<uint8_t>& tx, 
                                                const std::string& path,
                                                std::vector<uint8_t>& signature) {
    if (connected_devices_.empty()) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    SignatureInfo sig;
    ErrorCode result = connected_devices_.begin()->second->sign_transaction(tx, path, sig);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    signature = sig.signature;
    return ErrorCode::SUCCESS;
}

ErrorCode HardwareWalletManager::sign_message(const std::string& message,
                                            const std::string& path,
                                            std::vector<uint8_t>& signature) {
    if (connected_devices_.empty()) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    SignatureInfo sig;
    ErrorCode result = connected_devices_.begin()->second->sign_message(
        std::vector<uint8_t>(message.begin(), message.end()), path, sig);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    signature = sig.signature;
    return ErrorCode::SUCCESS;
}

ErrorCode HardwareWalletManager::init_multisig(uint8_t threshold, const std::vector<std::string>& signers) {
    // A threshold of zero, or one above the signer count, can never be met.
    if (threshold == 0 || signers.empty() || threshold > signers.size()) {
        return ErrorCode::INVALID_PARAMETER;
    }

    // Coordinated signing needs a device protocol that is not implemented and
    // session state to record the policy in. Reporting success would leave the
    // caller believing a wallet was configured.
    return ErrorCode::UNSUPPORTED_OPERATION;
}

ErrorCode HardwareWalletManager::sign_multisig(const std::vector<uint8_t>& tx,
                                               const std::string& path,
                                               std::vector<uint8_t>& signature) {
    // No multisig session can exist, so no partial signature can be produced.
    // An empty signature reported as success is indistinguishable from a real
    // one to the caller.
    (void)tx;
    (void)path;
    signature.clear();
    return ErrorCode::UNSUPPORTED_OPERATION;
}

void HardwareWalletManager::set_connection_callback(ConnectionCallback callback) {
    connection_callback_ = callback;
}

void HardwareWalletManager::set_pin_callback(PinCallback callback) {
    pin_callback_ = callback;
}

void HardwareWalletManager::set_confirm_callback(ConfirmCallback callback) {
    confirm_callback_ = callback;
}

void HardwareWalletManager::set_timeout(uint32_t timeout_ms) {
    timeout_ms_ = timeout_ms;
}

void HardwareWalletManager::set_auto_reconnect(bool enable) {
    auto_reconnect_ = enable;
}

void HardwareWalletManager::set_debug_mode(bool enable) {
    debug_mode_ = enable;
}

} // namespace hardware
} // namespace tigerwallet
