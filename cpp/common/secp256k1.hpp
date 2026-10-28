/**
 * TigerWallet secp256k1 primitives.
 *
 * The MPC engine previously derived a "public key" by SHA-256 hashing a private
 * share, combined "signature shares" by XOR, and verified a signature by
 * checking that it was not empty. All three are replaced here with the real
 * curve arithmetic they were standing in for.
 *
 * The arithmetic is built on OpenSSL's BIGNUM and EC_POINT. OpenSSL is a
 * general-purpose cryptography library, not a wallet SDK: no third-party wallet
 * supplies any part of this.
 *
 * What is here: the group, scalar arithmetic modulo the curve order, public key
 * derivation, compressed and uncompressed encodings, ECDSA signing and
 * verification with the low-s rule and the range checks EIP-2 and the
 * ecosystem require, and Ethereum address derivation.
 *
 * The BN_CTX and the group are thread_local. BN_CTX carries scratch state and
 * is not safe to share between threads, and building the group per call showed
 * up in profiles.
 */

#ifndef TIGER_WALLET_SECP256K1_HPP
#define TIGER_WALLET_SECP256K1_HPP

#include <openssl/bn.h>
#include <openssl/ec.h>
#include <openssl/ecdsa.h>
#include <openssl/obj_mac.h>
#include <openssl/rand.h>

#include <array>
#include <cstdint>
#include <cstring>
#include <memory>
#include <optional>
#include <stdexcept>
#include <string>
#include <vector>

#include "keccak256.hpp"

namespace tigerwallet {
namespace secp256k1 {

// ==================== Ownership ====================

struct BnDeleter {
    void operator()(BIGNUM* value) const { BN_free(value); }
};
struct CtxDeleter {
    void operator()(BN_CTX* value) const { BN_CTX_free(value); }
};
struct GroupDeleter {
    void operator()(EC_GROUP* value) const { EC_GROUP_free(value); }
};
struct PointDeleter {
    void operator()(EC_POINT* value) const { EC_POINT_free(value); }
};

using BnPtr = std::unique_ptr<BIGNUM, BnDeleter>;
using CtxPtr = std::unique_ptr<BN_CTX, CtxDeleter>;
using GroupPtr = std::unique_ptr<EC_GROUP, GroupDeleter>;
using PointPtr = std::unique_ptr<EC_POINT, PointDeleter>;

// ==================== Curve ====================

// The curve and the scratch context, built once per thread. BN_CTX holds
// scratch state and must not be shared between threads.
inline BN_CTX* context() {
    thread_local CtxPtr ctx(BN_CTX_new());
    if (!ctx) throw std::runtime_error("secp256k1: BN_CTX_new failed");
    return ctx.get();
}

inline const EC_GROUP* curve() {
    thread_local GroupPtr group(EC_GROUP_new_by_curve_name(NID_secp256k1));
    if (!group) throw std::runtime_error("secp256k1: curve unavailable");
    return group.get();
}

// The curve order n. Every scalar is reduced modulo this.
inline const BIGNUM* order() {
    thread_local BnPtr n([] {
        BnPtr value(BN_new());
        if (!value) throw std::runtime_error("secp256k1: BN_new failed");
        if (!EC_GROUP_get_order(curve(), value.get(), context())) {
            throw std::runtime_error("secp256k1: order unavailable");
        }
        return value.release();
    }());
    return n.get();
}

// ==================== Scalars ====================

inline BnPtr make_bn() {
    BnPtr value(BN_new());
    if (!value) throw std::runtime_error("secp256k1: BN_new failed");
    return value;
}

inline BnPtr from_bytes(const uint8_t* data, size_t size) {
    BnPtr value(BN_bin2bn(data, static_cast<int>(size), nullptr));
    if (!value) throw std::runtime_error("secp256k1: BN_bin2bn failed");
    return value;
}

inline BnPtr from_bytes(const std::vector<uint8_t>& data) {
    return from_bytes(data.data(), data.size());
}

// Fixed-width big-endian output. A signature component is always 32 bytes, so
// the width is stated rather than inferred: a value with a leading zero byte
// would otherwise be emitted one byte short, and BN_bn2bin alone does exactly
// that.
inline std::vector<uint8_t> to_bytes(const BIGNUM* value, size_t width) {
    std::vector<uint8_t> out(width, 0);
    if (BN_bn2binpad(value, out.data(), static_cast<int>(width)) < 0) {
        throw std::runtime_error("secp256k1: value does not fit the requested width");
    }
    return out;
}

// A scalar is valid when it is in [1, n-1]. Zero and anything at or above the
// order have no inverse and no valid public key, and a signature carrying one
// is rejected by every node.
inline bool is_valid_scalar(const BIGNUM* value) {
    if (BN_is_zero(value) || BN_is_negative(value)) return false;
    return BN_cmp(value, order()) < 0;
}

inline BnPtr mod_mul(const BIGNUM* a, const BIGNUM* b) {
    BnPtr out = make_bn();
    if (!BN_mod_mul(out.get(), a, b, order(), context())) {
        throw std::runtime_error("secp256k1: BN_mod_mul failed");
    }
    return out;
}

inline BnPtr mod_add(const BIGNUM* a, const BIGNUM* b) {
    BnPtr out = make_bn();
    if (!BN_mod_add(out.get(), a, b, order(), context())) {
        throw std::runtime_error("secp256k1: BN_mod_add failed");
    }
    return out;
}

inline BnPtr mod_sub(const BIGNUM* a, const BIGNUM* b) {
    BnPtr out = make_bn();
    if (!BN_mod_sub(out.get(), a, b, order(), context())) {
        throw std::runtime_error("secp256k1: BN_mod_sub failed");
    }
    return out;
}

// The modular inverse. Returns nullopt when the value is not invertible, which
// is the case for zero and for anything that shares a factor with the order.
inline std::optional<BnPtr> mod_inverse(const BIGNUM* value) {
    BnPtr out = make_bn();
    if (!BN_mod_inverse(out.get(), value, order(), context())) return std::nullopt;
    return out;
}

// A uniformly random scalar in [1, n-1]. Drawing a random 32-byte value and
// reducing it modulo n is biased; BN_rand_range is not.
inline BnPtr random_scalar() {
    BnPtr out = make_bn();
    if (!BN_rand_range(out.get(), order())) {
        throw std::runtime_error("secp256k1: BN_rand_range failed");
    }
    if (BN_is_zero(out.get())) return random_scalar();
    return out;
}

// ==================== Points ====================

inline PointPtr make_point() {
    PointPtr point(EC_POINT_new(curve()));
    if (!point) throw std::runtime_error("secp256k1: EC_POINT_new failed");
    return point;
}

inline PointPtr generator() {
    PointPtr point = make_point();
    if (!EC_POINT_copy(point.get(), EC_GROUP_get0_generator(curve()))) {
        throw std::runtime_error("secp256k1: generator copy failed");
    }
    return point;
}

inline PointPtr point_mul(const BIGNUM* scalar) {
    PointPtr point = make_point();
    if (!EC_POINT_mul(curve(), point.get(), scalar, nullptr, nullptr, context())) {
        throw std::runtime_error("secp256k1: EC_POINT_mul failed");
    }
    return point;
}

inline PointPtr point_add(const EC_POINT* a, const EC_POINT* b) {
    PointPtr point = make_point();
    if (!EC_POINT_add(curve(), point.get(), a, b, context())) {
        throw std::runtime_error("secp256k1: EC_POINT_add failed");
    }
    return point;
}

// The public key for a secret scalar. The previous code returned SHA-256 of the
// secret share, which is not a point on the curve at all.
inline PointPtr public_point(const BIGNUM* secret) {
    if (!is_valid_scalar(secret)) {
        throw std::runtime_error("secp256k1: secret is not a valid scalar");
    }
    return point_mul(secret);
}

inline std::vector<uint8_t> compress(const EC_POINT* point) {
    std::vector<uint8_t> out(33);
    const size_t written = EC_POINT_point2oct(curve(), point, POINT_CONVERSION_COMPRESSED,
                                              out.data(), out.size(), context());
    if (written != out.size()) {
        throw std::runtime_error("secp256k1: compression failed");
    }
    return out;
}

inline std::vector<uint8_t> uncompressed(const EC_POINT* point) {
    std::vector<uint8_t> out(65);
    const size_t written = EC_POINT_point2oct(curve(), point, POINT_CONVERSION_UNCOMPRESSED,
                                              out.data(), out.size(), context());
    if (written != out.size()) {
        throw std::runtime_error("secp256k1: encoding failed");
    }
    return out;
}

// Decodes a public key in compressed or uncompressed form. Returns nullopt for
// anything that is not a point on the curve, which is what stops a caller from
// verifying against a fabricated key.
inline std::optional<PointPtr> decode_point(const std::vector<uint8_t>& encoded) {
    PointPtr point = make_point();
    if (!EC_POINT_oct2point(curve(), point.get(), encoded.data(), encoded.size(), context())) {
        return std::nullopt;
    }
    if (EC_POINT_is_at_infinity(curve(), point.get()) == 1) return std::nullopt;
    if (EC_POINT_is_on_curve(curve(), point.get(), context()) != 1) return std::nullopt;
    return point;
}

// ==================== ECDSA ====================

struct Signature {
    std::array<uint8_t, 32> r{};
    std::array<uint8_t, 32> s{};
    uint8_t recovery_id = 0;
    bool has_recovery_id = false;

    // r and s concatenated, which is the form a raw transaction carries.
    std::vector<uint8_t> to_bytes() const {
        std::vector<uint8_t> out(64);
        std::copy(r.begin(), r.end(), out.begin());
        std::copy(s.begin(), s.end(), out.begin() + 32);
        return out;
    }

    // The DER form, which is what a general-purpose verifier expects.
    std::vector<uint8_t> to_der() const {
        BnPtr r_bn = from_bytes(r.data(), r.size());
        BnPtr s_bn = from_bytes(s.data(), s.size());
        ECDSA_SIG* sig = ECDSA_SIG_new();
        if (!sig) throw std::runtime_error("secp256k1: ECDSA_SIG_new failed");
        if (ECDSA_SIG_set0(sig, r_bn.release(), s_bn.release()) != 1) {
            ECDSA_SIG_free(sig);
            throw std::runtime_error("secp256k1: ECDSA_SIG_set0 failed");
        }
        const int length = i2d_ECDSA_SIG(sig, nullptr);
        if (length <= 0) {
            ECDSA_SIG_free(sig);
            throw std::runtime_error("secp256k1: DER encoding failed");
        }
        std::vector<uint8_t> out(static_cast<size_t>(length));
        uint8_t* cursor = out.data();
        i2d_ECDSA_SIG(sig, &cursor);
        ECDSA_SIG_free(sig);
        return out;
    }
};

// Signs a 32-byte message hash.
//
// The nonce is a uniformly random scalar in [1, n-1]. A predictable or reused
// nonce reveals the private key, so it comes from BN_rand_range over the curve
// order rather than from a counter or a hash of the message alone.
//
// The signature is normalised to low-s. A high-s signature is valid
// mathematically but is rejected or silently mutated by nodes, and accepting
// both forms gives a third party a way to produce a second valid signature for
// the same transaction, which changes its hash.
inline Signature sign(const BIGNUM* secret, const std::array<uint8_t, 32>& hash) {
    if (!is_valid_scalar(secret)) {
        throw std::runtime_error("secp256k1: private key is not a valid scalar");
    }

    // z is the hash reduced modulo n, as ECDSA specifies.
    BnPtr z = from_bytes(hash.data(), hash.size());
    BnPtr z_reduced = make_bn();
    if (!BN_nnmod(z_reduced.get(), z.get(), order(), context())) {
        throw std::runtime_error("secp256k1: hash reduction failed");
    }

    for (int attempt = 0; attempt < 64; ++attempt) {
        BnPtr nonce = random_scalar();

        PointPtr r_point = point_mul(nonce.get());
        BnPtr x = make_bn();
        if (!EC_POINT_get_affine_coordinates(curve(), r_point.get(), x.get(), nullptr, context())) {
            throw std::runtime_error("secp256k1: coordinate read failed");
        }

        BnPtr r = make_bn();
        if (!BN_nnmod(r.get(), x.get(), order(), context())) {
            throw std::runtime_error("secp256k1: r reduction failed");
        }
        // r is zero only for the vanishingly rare nonce whose point has an
        // x-coordinate divisible by the order. Draw another.
        if (BN_is_zero(r.get())) continue;

        BnPtr nonce_inverse = make_bn();
        if (!BN_mod_inverse(nonce_inverse.get(), nonce.get(), order(), context())) continue;

        // s = k^-1 (z + r*d)
        BnPtr rd = mod_mul(r.get(), secret);
        BnPtr sum = mod_add(z_reduced.get(), rd.get());
        BnPtr s = mod_mul(nonce_inverse.get(), sum.get());
        if (BN_is_zero(s.get())) continue;

        // Normalise to low-s.
        BnPtr half_order = make_bn();
        if (!BN_rshift1(half_order.get(), order())) {
            throw std::runtime_error("secp256k1: shift failed");
        }
        if (BN_cmp(s.get(), half_order.get()) > 0) {
            BnPtr negated = mod_sub(order(), s.get());
            s = std::move(negated);
        }

        Signature out;
        const auto r_bytes = to_bytes(r.get(), 32);
        const auto s_bytes = to_bytes(s.get(), 32);
        std::copy(r_bytes.begin(), r_bytes.end(), out.r.begin());
        std::copy(s_bytes.begin(), s_bytes.end(), out.s.begin());
        return out;
    }

    throw std::runtime_error("secp256k1: could not produce a signature");
}

// Verifies a signature against a public key.
//
// The previous implementation in the MPC engine returned true whenever the
// signature bytes were non-empty, which accepts every forgery. This performs
// the real check: both components are in [1, n-1], the public key is a point on
// the curve, and the ECDSA equation holds. A high-s signature is rejected, so
// only one of the two malleable forms of a given signature is accepted.
inline bool verify(const std::vector<uint8_t>& public_key,
                   const std::array<uint8_t, 32>& hash,
                   const Signature& signature) {
    const auto point = decode_point(public_key);
    if (!point) return false;

    BnPtr r = from_bytes(signature.r.data(), signature.r.size());
    BnPtr s = from_bytes(signature.s.data(), signature.s.size());
    if (!is_valid_scalar(r.get()) || !is_valid_scalar(s.get())) return false;

    BnPtr half_order = make_bn();
    if (!BN_rshift1(half_order.get(), order())) return false;
    if (BN_cmp(s.get(), half_order.get()) > 0) return false;

    BnPtr z = from_bytes(hash.data(), hash.size());
    BnPtr z_reduced = make_bn();
    if (!BN_nnmod(z_reduced.get(), z.get(), order(), context())) return false;

    BnPtr s_inverse = make_bn();
    if (!BN_mod_inverse(s_inverse.get(), s.get(), order(), context())) return false;

    // u1 = z/s, u2 = r/s
    BnPtr u1 = mod_mul(z_reduced.get(), s_inverse.get());
    BnPtr u2 = mod_mul(r.get(), s_inverse.get());

    // R = u1*G + u2*Q
    PointPtr u1_g = point_mul(u1.get());
    PointPtr u2_q = make_point();
    if (!EC_POINT_mul(curve(), u2_q.get(), nullptr, point->get(), u2.get(), context())) {
        return false;
    }
    PointPtr sum = point_add(u1_g.get(), u2_q.get());
    if (EC_POINT_is_at_infinity(curve(), sum.get()) == 1) return false;

    BnPtr x = make_bn();
    if (!EC_POINT_get_affine_coordinates(curve(), sum.get(), x.get(), nullptr, context())) {
        return false;
    }
    BnPtr x_reduced = make_bn();
    if (!BN_nnmod(x_reduced.get(), x.get(), order(), context())) return false;

    return BN_cmp(x_reduced.get(), r.get()) == 0;
}

inline bool verify(const std::vector<uint8_t>& public_key,
                   const std::array<uint8_t, 32>& hash,
                   const std::vector<uint8_t>& raw_signature) {
    if (raw_signature.size() != 64) return false;
    Signature signature;
    std::copy(raw_signature.begin(), raw_signature.begin() + 32, signature.r.begin());
    std::copy(raw_signature.begin() + 32, raw_signature.end(), signature.s.begin());
    return verify(public_key, hash, signature);
}

// ==================== Ethereum addresses ====================

// The Ethereum address is the last 20 bytes of the Keccak-256 of the
// uncompressed public key with its 0x04 prefix removed. This is the derivation
// the previous code approximated by SHA-256 hashing a secret share.
inline std::vector<uint8_t> ethereum_address(const EC_POINT* point) {
    const std::vector<uint8_t> encoded = uncompressed(point);
    const std::string material(reinterpret_cast<const char*>(encoded.data() + 1), 64);
    const auto digest = crypto::Keccak256::hash(material);
    return std::vector<uint8_t>(digest.begin() + 12, digest.end());
}

inline std::string to_checksum_address(const std::vector<uint8_t>& address) {
    static const char* kHex = "0123456789abcdef";
    std::string lower;
    lower.reserve(40);
    for (uint8_t byte : address) {
        lower.push_back(kHex[byte >> 4]);
        lower.push_back(kHex[byte & 0x0f]);
    }
    const auto digest = crypto::Keccak256::hash(lower);
    std::string out = "0x";
    for (size_t i = 0; i < lower.size(); ++i) {
        const char c = lower[i];
        if (c >= '0' && c <= '9') {
            out.push_back(c);
            continue;
        }
        const uint8_t nibble = (i % 2 == 0) ? (digest[i / 2] >> 4) : (digest[i / 2] & 0x0f);
        out.push_back(nibble >= 8 ? static_cast<char>(std::toupper(static_cast<unsigned char>(c))) : c);
    }
    return out;
}

// ==================== Keccak helpers ====================

inline std::array<uint8_t, 32> keccak256(const uint8_t* data, size_t size) {
    const std::string material(reinterpret_cast<const char*>(data), size);
    return crypto::Keccak256::hash(material);
}

inline std::array<uint8_t, 32> keccak256(const std::vector<uint8_t>& data) {
    return keccak256(data.data(), data.size());
}

}  // namespace secp256k1
}  // namespace tigerwallet

#endif  // TIGER_WALLET_SECP256K1_HPP
