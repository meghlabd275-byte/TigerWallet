/**
 * TigerWallet Desktop - EVM Transaction Signer Implementation
 *
 * Real EVM signing: RLP encoding, ECDSA secp256k1 with EIP-155,
 * personal_sign with Ethereum prefix. No stubs.
 */

#include "crypto/evm_signer.h"
#include "crypto/keccak256.h"
#include <openssl/ec.h>
#include <openssl/bn.h>
#include <openssl/obj_mac.h>
#include <openssl/evp.h>
#include <sstream>
#include <cstring>
#include <stdexcept>

namespace tiger {
namespace wallet {

void EvmSigner::encodeLength(std::vector<uint8_t>& out, size_t len, uint8_t offset) {
    if (len < 56) {
        out.push_back((uint8_t)(offset + len));
    } else {
        // Length of length
        uint8_t lenBytes[8];
        int n = 0;
        size_t tmp = len;
        while (tmp > 0) {
            lenBytes[n++] = (uint8_t)(tmp & 0xFF);
            tmp >>= 8;
        }
        out.push_back((uint8_t)(offset + 55 + n));
        for (int i = n - 1; i >= 0; i--) {
            out.push_back(lenBytes[i]);
        }
    }
}

std::vector<uint8_t> EvmSigner::rlpEncodeBytes(const std::vector<uint8_t>& data) {
    std::vector<uint8_t> out;
    if (data.size() == 1 && data[0] < 0x80) {
        out.push_back(data[0]);
        return out;
    }
    encodeLength(out, data.size(), 0x80);
    out.insert(out.end(), data.begin(), data.end());
    return out;
}

std::vector<uint8_t> EvmSigner::rlpEncodeList(const std::vector<std::vector<uint8_t>>& items) {
    std::vector<uint8_t> payload;
    for (const auto& item : items) {
        payload.insert(payload.end(), item.begin(), item.end());
    }
    std::vector<uint8_t> out;
    encodeLength(out, payload.size(), 0xC0);
    out.insert(out.end(), payload.begin(), payload.end());
    return out;
}

std::vector<uint8_t> EvmSigner::rlpEncodeUint(uint64_t value) {
    if (value == 0) {
        return rlpEncodeBytes({}); // empty string for zero
    }
    std::vector<uint8_t> bytes;
    while (value > 0) {
        bytes.insert(bytes.begin(), (uint8_t)(value & 0xFF));
        value >>= 8;
    }
    return rlpEncodeBytes(bytes);
}

std::vector<uint8_t> EvmSigner::hexToBytes(const std::string& hex) {
    std::string h = hex;
    if (h.substr(0, 2) == "0x") h = h.substr(2);
    std::vector<uint8_t> bytes;
    for (size_t i = 0; i < h.length(); i += 2) {
        std::string byteStr = h.substr(i, 2);
        if (byteStr.length() == 1) byteStr = "0" + byteStr;
        bytes.push_back((uint8_t)std::stoul(byteStr, nullptr, 16));
    }
    return bytes;
}

std::string EvmSigner::bytesToHex(const std::vector<uint8_t>& bytes) {
    static const char* hexchars = "0123456789abcdef";
    std::string out = "0x";
    for (auto b : bytes) {
        out += hexchars[b >> 4];
        out += hexchars[b & 0xf];
    }
    return out;
}

std::vector<uint8_t> EvmSigner::ecdsaSign(const std::vector<uint8_t>& hash,
                                           const std::vector<uint8_t>& privateKey) {
    BN_CTX* ctx = BN_CTX_new();
    EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);

    BIGNUM* privBN = BN_bin2bn(privateKey.data(), (int)privateKey.size(), nullptr);
    BIGNUM* hashBN = BN_bin2bn(hash.data(), (int)hash.size(), nullptr);
    BIGNUM* n = BN_new();
    EC_GROUP_get_order(group, n, ctx);

    // Generate deterministic k (RFC 6979 simplified: use hash of key||msg)
    // For production, use full RFC 6979. This uses random k which is also
    // cryptographically sound (non-repeating, from RAND_bytes).
    BIGNUM* k = BN_new();
    BIGNUM* kInv = BN_new();
    EC_POINT* R = EC_POINT_new(group);
    BIGNUM* r = BN_new();
    BIGNUM* s = BN_new();

    // Loop until we get a valid (r, s) with s < n/2 (low-s enforcement for EIP-2)
    int recoveryId = 0;
    while (true) {
        // Random k in [1, n-1]
        BN_rand_range(k, n);
        if (BN_is_zero(k)) continue;

        // R = k * G
        EC_POINT_mul(group, R, k, nullptr, nullptr, ctx);

        // r = R.x mod n
        EC_POINT_get_affine_coordinates(group, R, r, nullptr, ctx);
        BN_mod(r, r, n, ctx);
        if (BN_is_zero(r)) continue;

        // s = k^{-1} * (hash + r * privKey) mod n
        BN_mod_inverse(kInv, k, n, ctx);

        BIGNUM* rTimesPriv = BN_new();
        BN_mod_mul(rTimesPriv, r, privBN, n, ctx);

        BIGNUM* sum = BN_new();
        BN_mod_add(sum, hashBN, rTimesPriv, n, ctx);

        BN_mod_mul(s, kInv, sum, n, ctx);

        BN_free(rTimesPriv);
        BN_free(sum);

        if (BN_is_zero(s)) continue;

        // Low-s enforcement (EIP-2): if s > n/2, s = n - s, flip recoveryId
        BIGNUM* halfN = BN_new();
        BN_rshift1(halfN, n);
        if (BN_cmp(s, halfN) > 0) {
            BIGNUM* newS = BN_new();
            BN_sub(newS, n, s);
            BN_copy(s, newS);
            BN_free(newS);
            recoveryId ^= 1; // flip
        }
        BN_free(halfN);
        break;
    }

    // Determine recovery id from R's y parity
    BIGNUM* ry = BN_new();
    EC_POINT_get_affine_coordinates(group, R, nullptr, ry, ctx);
    bool yOdd = BN_is_odd(ry);
    recoveryId = (yOdd ? 1 : 0) ^ recoveryId;
    BN_free(ry);

    // Output: r(32) || s(32) || v(1)
    std::vector<uint8_t> result(65);
    BN_bn2binpad(r, result.data(), 32);
    BN_bn2binpad(s, result.data() + 32, 32);
    result[64] = (uint8_t)recoveryId;

    BN_free(k);
    BN_free(kInv);
    EC_POINT_free(R);
    BN_free(r);
    BN_free(s);
    BN_free(privBN);
    BN_free(hashBN);
    BN_free(n);
    EC_GROUP_free(group);
    BN_CTX_free(ctx);

    return result;
}

std::string EvmSigner::signLegacy(const EvmTxParams& params,
                                   const std::vector<uint8_t>& privateKey) {
    // EIP-155 legacy signing:
    // 1. RLP-encode unsigned tx with chainId: [nonce, gasPrice, gasLimit, to, value, data, chainId, 0, 0]
    // 2. keccak256(rlp_unsigned)
    // 3. ECDSA sign → r, s, v(recoveryId)
    // 4. v = recoveryId + 35 + 2*chainId
    // 5. RLP-encode signed: [nonce, gasPrice, gasLimit, to, value, data, v, r, s]

    auto toBytes = hexToBytes(params.toAddress);
    auto dataBytes = hexToBytes(params.data);
    auto gasPriceBytes = hexToBytes(params.gasPriceWei);
    auto valueBytes = hexToBytes(params.valueWei);

    std::vector<std::vector<uint8_t>> unsignedItems = {
        rlpEncodeUint(params.nonce),
        rlpEncodeBytes(gasPriceBytes),
        rlpEncodeUint(params.gasLimit),
        rlpEncodeBytes(toBytes),
        rlpEncodeBytes(valueBytes),
        rlpEncodeBytes(dataBytes),
        rlpEncodeUint(params.chainId),
        rlpEncodeBytes({}),  // 0
        rlpEncodeBytes({}),  // 0
    };

    auto unsignedRlp = rlpEncodeList(unsignedItems);
    auto hash = Keccak256::hash(unsignedRlp.data(), unsignedRlp.size());

    auto sig = ecdsaSign(hash, privateKey);
    uint8_t recoveryId = sig[64];

    // EIP-155: v = recoveryId + 35 + 2*chainId
    uint64_t v = recoveryId + 35 + 2 * params.chainId;

    std::vector<uint8_t> r(sig.begin(), sig.begin() + 32);
    std::vector<uint8_t> s(sig.begin() + 32, sig.begin() + 64);

    std::vector<std::vector<uint8_t>> signedItems = {
        rlpEncodeUint(params.nonce),
        rlpEncodeBytes(gasPriceBytes),
        rlpEncodeUint(params.gasLimit),
        rlpEncodeBytes(toBytes),
        rlpEncodeBytes(valueBytes),
        rlpEncodeBytes(dataBytes),
        rlpEncodeUint(v),
        rlpEncodeBytes(r),
        rlpEncodeBytes(s),
    };

    auto signedRlp = rlpEncodeList(signedItems);
    return bytesToHex(signedRlp);
}

std::string EvmSigner::signEIP1559(const EvmTxParams& params,
                                    const std::vector<uint8_t>& privateKey) {
    // EIP-1559 (type-2) signing:
    // 1. 0x02 || RLP([chainId, nonce, maxPrioFee, maxFee, gasLimit, to, value, data, accessList])
    // 2. keccak256(0x02 || rlp_unsigned)
    // 3. ECDSA sign → r, s, v(recoveryId)
    // 4. 0x02 || RLP([chainId, nonce, maxPrioFee, maxFee, gasLimit, to, value, data, accessList, v, r, s])
    //    where v = recoveryId (0 or 1, no +27)

    auto toBytes = hexToBytes(params.toAddress);
    auto dataBytes = hexToBytes(params.data);
    auto maxFeeBytes = hexToBytes(params.maxFeePerGas);
    auto maxPrioBytes = hexToBytes(params.maxPriorityFee);
    auto valueBytes = hexToBytes(params.valueWei);

    std::vector<std::vector<uint8_t>> unsignedItems = {
        rlpEncodeUint(params.chainId),
        rlpEncodeUint(params.nonce),
        rlpEncodeBytes(maxPrioBytes),
        rlpEncodeBytes(maxFeeBytes),
        rlpEncodeUint(params.gasLimit),
        rlpEncodeBytes(toBytes),
        rlpEncodeBytes(valueBytes),
        rlpEncodeBytes(dataBytes),
        rlpEncodeList({}),  // empty access list
    };

    auto unsignedRlp = rlpEncodeList(unsignedItems);

    // Prepend 0x02 (transaction type)
    std::vector<uint8_t> typePrefixed;
    typePrefixed.push_back(0x02);
    typePrefixed.insert(typePrefixed.end(), unsignedRlp.begin(), unsignedRlp.end());

    auto hash = Keccak256::hash(typePrefixed.data(), typePrefixed.size());
    auto sig = ecdsaSign(hash, privateKey);
    uint8_t recoveryId = sig[64];

    std::vector<uint8_t> r(sig.begin(), sig.begin() + 32);
    std::vector<uint8_t> s(sig.begin() + 32, sig.begin() + 64);

    std::vector<std::vector<uint8_t>> signedItems = {
        rlpEncodeUint(params.chainId),
        rlpEncodeUint(params.nonce),
        rlpEncodeBytes(maxPrioBytes),
        rlpEncodeBytes(maxFeeBytes),
        rlpEncodeUint(params.gasLimit),
        rlpEncodeBytes(toBytes),
        rlpEncodeBytes(valueBytes),
        rlpEncodeBytes(dataBytes),
        rlpEncodeList({}),  // empty access list
        rlpEncodeUint(recoveryId),
        rlpEncodeBytes(r),
        rlpEncodeBytes(s),
    };

    auto signedRlp = rlpEncodeList(signedItems);

    std::vector<uint8_t> result;
    result.push_back(0x02);
    result.insert(result.end(), signedRlp.begin(), signedRlp.end());
    return bytesToHex(result);
}

std::string EvmSigner::personalSign(const std::string& message,
                                     const std::vector<uint8_t>& privateKey) {
    // Ethereum personal sign: keccak256("\x19Ethereum Signed Message:\n" + len(msg) + msg)
    std::string prefix = "\x19""Ethereum Signed Message:\n" + std::to_string(message.size());
    std::string fullMessage = prefix + message;

    auto hash = Keccak256::hash(
        reinterpret_cast<const uint8_t*>(fullMessage.data()), fullMessage.size());

    auto sig = ecdsaSign(hash, privateKey);

    // v = recoveryId + 27 for personal_sign
    sig[64] += 27;

    return bytesToHex(sig);
}

} // namespace wallet
} // namespace tiger
