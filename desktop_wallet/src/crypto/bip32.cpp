/**
 * TigerWallet Desktop - BIP-32 HD Wallet Implementation
 *
 * Real BIP-32 using OpenSSL's EC API (secp256k1 group) for point operations
 * and modular arithmetic. No fake SHA512(seed||path) derivation.
 */

#include "crypto/bip32.h"
#include "crypto/keccak256.h"
#include <openssl/hmac.h>
#include <openssl/ec.h>
#include <openssl/bn.h>
#include <openssl/obj_mac.h>
#include <sstream>
#include <cstring>
#include <stdexcept>

namespace tiger {
namespace wallet {

// secp256k1 curve order n (big-endian, 32 bytes)
static const char* CURVE_ORDER_HEX =
    "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141";

std::vector<uint8_t> BIP32::hmacSha512(const std::vector<uint8_t>& key,
                                        const std::vector<uint8_t>& data) {
    std::vector<uint8_t> out(64);
    unsigned int len = 64;
    HMAC(EVP_sha512(),
         key.data(), (int)key.size(),
         data.data(), (int)data.size(),
         out.data(), &len);
    return out;
}

std::vector<uint8_t> BIP32::curveOrder() {
    std::vector<uint8_t> order(32);
    BN_CTX* ctx = BN_CTX_new();
    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);
    BIGNUM* n = BN_new();
    EC_GROUP_get_order(group, n, ctx);
    BN_bn2bin(n, order.data());
    BN_free(n);
    EC_GROUP_free(group);
    BN_CTX_free(ctx);
    return order;
}

std::vector<uint8_t> BIP32::modAdd(const std::vector<uint8_t>& a,
                                    const std::vector<uint8_t>& b) {
    BN_CTX* ctx = BN_CTX_new();
    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);
    BIGNUM* n = BN_new();
    EC_GROUP_get_order(group, n, ctx);

    BIGNUM* bnA = BN_bin2bn(a.data(), (int)a.size(), nullptr);
    BIGNUM* bnB = BN_bin2bn(b.data(), (int)b.size(), nullptr);
    BIGNUM* result = BN_new();

    BN_mod_add(result, bnA, bnB, n, ctx);

    std::vector<uint8_t> out(32);
    BN_bn2binpad(result, out.data(), 32);

    BN_free(bnA);
    BN_free(bnB);
    BN_free(result);
    BN_free(n);
    EC_GROUP_free(group);
    BN_CTX_free(ctx);
    return out;
}

HDKey BIP32::fromSeed(const std::vector<uint8_t>& seed) {
    std::vector<uint8_t> key("Bitcoin seed", "Bitcoin seed" + 12);
    auto I = hmacSha512(key, seed);

    HDKey master;
    master.private_key.assign(I.begin(), I.begin() + 32);
    master.chain_code.assign(I.begin() + 32, I.end());
    return master;
}

HDKey BIP32::deriveChild(const HDKey& parent, uint32_t index) {
    std::vector<uint8_t> data;

    if (index >= HARDENED_OFFSET) {
        // Hardened: 0x00 || ser256(kpar) || ser32(i)
        data.push_back(0x00);
        data.insert(data.end(), parent.private_key.begin(), parent.private_key.end());
    } else {
        // Normal: serP(point(kpar)) || ser32(i)
        auto pubkey = compressedPublicKey(parent.private_key);
        data.insert(data.end(), pubkey.begin(), pubkey.end());
    }

    // Append index as 4 bytes big-endian
    data.push_back((index >> 24) & 0xFF);
    data.push_back((index >> 16) & 0xFF);
    data.push_back((index >> 8) & 0xFF);
    data.push_back(index & 0xFF);

    auto I = hmacSha512(parent.chain_code, data);
    auto IL = std::vector<uint8_t>(I.begin(), I.begin() + 32);
    auto IR = std::vector<uint8_t>(I.begin() + 32, I.end());

    // child_key = (IL + parent_key) mod n
    auto childKey = modAdd(IL, parent.private_key);

    HDKey child;
    child.private_key = childKey;
    child.chain_code = IR;
    return child;
}

std::vector<uint32_t> BIP32::parsePath(const std::string& path) {
    std::vector<uint32_t> indices;
    std::istringstream iss(path);
    std::string segment;

    // Skip "m" prefix if present
    std::getline(iss, segment, '/');
    if (segment != "m" && !segment.empty()) {
        // No "m" prefix, parse first segment
        bool hardened = false;
        if (!segment.empty() && (segment.back() == '\'' || segment.back() == 'h' || segment.back() == 'H')) {
            hardened = true;
            segment.pop_back();
        }
        uint32_t idx = (uint32_t)std::stoul(segment);
        if (hardened) idx |= HARDENED_OFFSET;
        indices.push_back(idx);
    }

    while (std::getline(iss, segment, '/')) {
        if (segment.empty()) continue;
        bool hardened = false;
        if (segment.back() == '\'' || segment.back() == 'h' || segment.back() == 'H') {
            hardened = true;
            segment.pop_back();
        }
        uint32_t idx = (uint32_t)std::stoul(segment);
        if (hardened) idx |= HARDENED_OFFSET;
        indices.push_back(idx);
    }
    return indices;
}

HDKey BIP32::derivePath(const HDKey& master, const std::string& path) {
    auto indices = parsePath(path);
    HDKey current = master;
    for (uint32_t idx : indices) {
        current = deriveChild(current, idx);
    }
    return current;
}

std::vector<uint8_t> BIP32::publicKey(const std::vector<uint8_t>& privateKey) {
    BN_CTX* ctx = BN_CTX_new();
    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);

    BIGNUM* privBN = BN_bin2bn(privateKey.data(), (int)privateKey.size(), nullptr);
    EC_POINT* pub = EC_POINT_new(group);
    EC_POINT_mul(group, pub, privBN, nullptr, nullptr, ctx);

    // Get uncompressed public key (65 bytes: 0x04 + X + Y)
    std::vector<uint8_t> out(65);
    EC_POINT_point2oct(group, pub, POINT_CONVERSION_UNCOMPRESSED, out.data(), 65, ctx);

    EC_POINT_free(pub);
    BN_free(privBN);
    EC_GROUP_free(group);
    BN_CTX_free(ctx);
    return out;
}

std::vector<uint8_t> BIP32::compressedPublicKey(const std::vector<uint8_t>& privateKey) {
    BN_CTX* ctx = BN_CTX_new();
    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);

    BIGNUM* privBN = BN_bin2bn(privateKey.data(), (int)privateKey.size(), nullptr);
    EC_POINT* pub = EC_POINT_new(group);
    EC_POINT_mul(group, pub, privBN, nullptr, nullptr, ctx);

    std::vector<uint8_t> out(33);
    EC_POINT_point2oct(group, pub, POINT_CONVERSION_COMPRESSED, out.data(), 33, ctx);

    EC_POINT_free(pub);
    BN_free(privBN);
    EC_GROUP_free(group);
    BN_CTX_free(ctx);
    return out;
}

std::string BIP32::evmAddress(const std::vector<uint8_t>& privateKey) {
    // EVM address = last 20 bytes of keccak256(uncompressed_pubkey[1:65])
    auto pub = publicKey(privateKey);
    // Skip the 0x04 prefix
    std::vector<uint8_t> pubXY(pub.begin() + 1, pub.end());
    auto hash = Keccak256::hash(pubXY.data(), pubXY.size());

    // Address = hash[12:32]
    std::vector<uint8_t> addr(hash.begin() + 12, hash.end());
    return "0x" + Keccak256::hex(addr);
}

} // namespace wallet
} // namespace tiger
