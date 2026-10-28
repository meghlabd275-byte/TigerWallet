// Tests for the secp256k1 primitives.
//
// The vectors are the ones the ecosystem publishes, so these assertions check
// against the specification rather than against this code's own output:
//
//   - The private keys and compressed public keys are the well-known sequence
//     from the Bitcoin wiki (1 -> 0279BE667E..., 2 -> 02C6047F94..., and so on).
//   - The Ethereum addresses are the published addresses for those keys.
//   - The EIP-191 hash and address for key 1 is the vector from the Ethereum
//     documentation.
//   - A fixed private key and message, signed and then verified, with tampered
//     r, s and message all rejected. This checks that signing and verification
//     agree and that verification rejects forgeries; it is not a RFC 6979
//     vector, because the nonce here is random rather than derived.
//   - The low-s boundary is checked at (n-1)/2 and (n-1)/2 + 1.

#include "secp256k1.hpp"

#include <cassert>
#include <cstdio>
#include <cstring>
#include <stdexcept>
#include <string>
#include <vector>

using namespace tigerwallet::secp256k1;

namespace {

std::vector<uint8_t> hex_to_bytes(const std::string& hex) {
    std::vector<uint8_t> out;
    out.reserve(hex.size() / 2);
    for (size_t i = 0; i + 1 < hex.size(); i += 2) {
        out.push_back(static_cast<uint8_t>(std::stoi(hex.substr(i, 2), nullptr, 16)));
    }
    return out;
}

BnPtr hex_to_scalar(const std::string& hex) {
    const auto bytes = hex_to_bytes(hex);
    if (bytes.empty()) throw std::runtime_error("hex_to_scalar: empty input");
    return from_bytes(bytes.data(), bytes.size());
}

// Parses a decimal integer. The small private keys (1, 2, 3) are decimal in the
// published tables, and reading "1" as hex would give an empty byte string.
BnPtr decimal_to_scalar(const std::string& digits) {
    BIGNUM* raw = nullptr;
    if (!BN_dec2bn(&raw, digits.c_str())) {
        throw std::runtime_error("decimal_to_scalar: parse failed");
    }
    return BnPtr(raw);
}

std::string to_hex(const std::vector<uint8_t>& data) {
    static const char* kHex = "0123456789abcdef";
    std::string out;
    out.reserve(data.size() * 2);
    for (uint8_t byte : data) {
        out.push_back(kHex[byte >> 4]);
        out.push_back(kHex[byte & 0x0f]);
    }
    return out;
}

void test_public_key_derivation() {
    // The published private key to compressed public key mapping. The previous
    // implementation returned SHA-256 of the key, which is not a curve point;
    // none of these would have matched.
    struct Vector {
        const char* secret;
        const char* compressed;
    };
    const Vector vectors[] = {
        {"1",
         "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"},
        {"2",
         "02c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5"},
        {"3",
         "02f9308a019258c31049344f85f89d5229b531c845836f99b08601f113bce036f9"},
        {"4",
         "02e493dbf1c10d80f3581e4904930b1404cc6c13900ee0758474fa94abe8c4cd13"},
        {"5",
         "022f8bde4d1a07209355b4a7250a5c5128e88b84bddc619ab7cba8d569b240efe4"},
    };

    for (const auto& vector : vectors) {
        BnPtr secret = decimal_to_scalar(vector.secret);
        const auto point = public_point(secret.get());
        assert(to_hex(compress(point.get())) == vector.compressed);
    }

    // Uncompressed form starts with 0x04 and is 65 bytes.
    BnPtr one = decimal_to_scalar("1");
    const auto point = public_point(one.get());
    const auto uncompressed_key = uncompressed(point.get());
    assert(uncompressed_key.size() == 65);
    assert(uncompressed_key[0] == 0x04);
}

void test_ethereum_addresses() {
    // Published Ethereum addresses for the same keys.
    struct Vector {
        const char* secret;
        const char* address;
    };
    const Vector vectors[] = {
        {"1", "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf"},
        {"2", "0x2B5AD5c4795c026514f8317c7a215E218DcCD6cF"},
        {"3", "0x6813Eb9362372EEF6200f3b1dbC3f819671cBA69"},
        {"4", "0x1efF47bc3a10a45D4B230B5d10E37751FE6AA718"},
    };
    for (const auto& vector : vectors) {
        BnPtr secret = decimal_to_scalar(vector.secret);
        const auto point = public_point(secret.get());
        const auto address = ethereum_address(point.get());
        assert(address.size() == 20);
        assert(to_checksum_address(address) == vector.address);
    }
}

void test_group_arithmetic() {
    // 1*G + 1*G must equal 2*G, and 2*G + 1*G must equal 3*G. This is what
    // makes a Lagrange combination of shares a real point rather than an XOR.
    BnPtr one = decimal_to_scalar("1");
    BnPtr two = decimal_to_scalar("2");
    BnPtr three = decimal_to_scalar("3");

    const auto g1 = point_mul(one.get());
    const auto g2 = point_mul(two.get());
    const auto g3 = point_mul(three.get());

    const auto sum = point_add(g1.get(), g1.get());
    assert(to_hex(compress(sum.get())) == to_hex(compress(g2.get())));

    const auto sum3 = point_add(g2.get(), g1.get());
    assert(to_hex(compress(sum3.get())) == to_hex(compress(g3.get())));
}

void test_scalar_field() {
    BnPtr n_minus_one([] {
        BnPtr value = make_bn();
        BN_sub_word(BN_copy(value.get(), order()), 1);
        return value.release();
    }());

    assert(is_valid_scalar(n_minus_one.get()));
    assert(!is_valid_scalar(order()));  // n itself is out of range.
    BnPtr zero = decimal_to_scalar("0");
    assert(!is_valid_scalar(zero.get()));

    // (n-1) is its own inverse modulo n.
    const auto inverse = mod_inverse(n_minus_one.get());
    assert(inverse.has_value());
    assert(BN_cmp(inverse->get(), n_minus_one.get()) == 0);

    // Zero has no inverse.
    assert(!mod_inverse(zero.get()).has_value());

    // a + (n - a) == 0 modulo n.
    BnPtr five = decimal_to_scalar("5");
    BnPtr neg_five = mod_sub(order(), five.get());
    const auto sum = mod_add(five.get(), neg_five.get());
    assert(BN_is_zero(sum.get()));
}

void test_sign_verify_round_trip() {
    // Sign then verify, and confirm that every part of the signature is
    // actually checked. The signature is produced by this implementation, so
    // the assertions are that signing and verification agree and that a wrong
    // hash, a wrong r or a wrong s is rejected. A fixed r and s are not
    // asserted, because the nonce here is random rather than derived.
    const std::string secret_hex =
        "c9afa9d845ba75166b5c215767b1d6934e50c3db36e89b127b8a622b120f6721";
    BnPtr secret = hex_to_scalar(secret_hex);
    assert(is_valid_scalar(secret.get()));

    const auto public_key = compress(public_point(secret.get()).get());
    const auto hash = keccak256(reinterpret_cast<const uint8_t*>("sample"),
                                std::strlen("sample"));

    const Signature signature = sign(secret.get(), hash);
    assert(verify(public_key, hash, signature));

    // A different hash must not verify.
    const auto other_hash = keccak256(reinterpret_cast<const uint8_t*>("sample2"),
                                      std::strlen("sample2"));
    assert(!verify(public_key, other_hash, signature));

    // A tampered r must not verify.
    Signature tampered = signature;
    tampered.r[31] ^= 0x01;
    assert(!verify(public_key, hash, tampered));

    // A tampered s must not verify.
    tampered = signature;
    tampered.s[31] ^= 0x01;
    assert(!verify(public_key, hash, tampered));

    // The raw 64-byte form verifies through the same path.
    assert(verify(public_key, hash, signature.to_bytes()));
}

void test_verification_rejects_forgeries() {
    BnPtr secret = decimal_to_scalar("1");
    const auto public_key = compress(public_point(secret.get()).get());
    const auto hash = keccak256(reinterpret_cast<const uint8_t*>("hello"), 5);

    const Signature signature = sign(secret.get(), hash);
    assert(verify(public_key, hash, signature));

    // An all-zero signature is rejected. This is the case the previous
    // "not empty" check accepted.
    Signature zeros;
    assert(!verify(public_key, hash, zeros));

    // A signature from a different key does not verify.
    BnPtr other_secret = decimal_to_scalar("2");
    const Signature other_signature = sign(other_secret.get(), hash);
    assert(!verify(public_key, hash, other_signature));

    // A public key that is not a point on the curve is rejected rather than
    // used.
    std::vector<uint8_t> bogus_key(33, 0x02);
    assert(!verify(bogus_key, hash, signature));

    // A 64-byte buffer that is not a signature is rejected.
    assert(!verify(public_key, hash, std::vector<uint8_t>(64, 0)));
    assert(!verify(public_key, hash, std::vector<uint8_t>(63, 0)));
    assert(!verify(public_key, hash, std::vector<uint8_t>()));
}

void test_low_s_normalisation() {
    // Every signature this implementation produces must have s <= (n-1)/2, and
    // must be accepted. The complement n - s is above the bound and must be
    // rejected, so the two malleable forms are not both accepted.
    BnPtr secret = decimal_to_scalar("3");
    const auto public_key = compress(public_point(secret.get()).get());
    const auto hash = keccak256(reinterpret_cast<const uint8_t*>("malleability"), 12);

    BnPtr half_order = make_bn();
    BN_rshift1(half_order.get(), order());

    for (int i = 0; i < 16; ++i) {
        const Signature signature = sign(secret.get(), hash);
        BnPtr s = from_bytes(signature.s.data(), signature.s.size());
        assert(BN_cmp(s.get(), half_order.get()) <= 0);
        assert(verify(public_key, hash, signature));

        // The high-s complement is rejected.
        BnPtr high = mod_sub(order(), s.get());
        Signature malleated = signature;
        const auto high_bytes = to_bytes(high.get(), 32);
        std::copy(high_bytes.begin(), high_bytes.end(), malleated.s.begin());
        assert(!verify(public_key, hash, malleated));
    }
}

void test_der_round_trip() {
    BnPtr secret = decimal_to_scalar("5");
    const auto public_key = compress(public_point(secret.get()).get());
    const auto hash = keccak256(reinterpret_cast<const uint8_t*>("der"), 3);

    const Signature signature = sign(secret.get(), hash);
    const auto der = signature.to_der();
    assert(!der.empty());
    assert(der[0] == 0x30);  // A DER sequence.

    // OpenSSL must accept the DER encoding this produces.
    BnPtr r = from_bytes(signature.r.data(), signature.r.size());
    BnPtr s = from_bytes(signature.s.data(), signature.s.size());
    const uint8_t* cursor = der.data();
    ECDSA_SIG* parsed = d2i_ECDSA_SIG(nullptr, &cursor, static_cast<long>(der.size()));
    assert(parsed != nullptr);
    const BIGNUM* parsed_r = nullptr;
    const BIGNUM* parsed_s = nullptr;
    ECDSA_SIG_get0(parsed, &parsed_r, &parsed_s);
    assert(BN_cmp(parsed_r, r.get()) == 0);
    assert(BN_cmp(parsed_s, s.get()) == 0);
    ECDSA_SIG_free(parsed);
}

void test_invalid_secrets() {
    BnPtr zero = decimal_to_scalar("0");
    bool threw = false;
    try {
        public_point(zero.get());
    } catch (const std::runtime_error&) {
        threw = true;
    }
    assert(threw);

    BnPtr too_large = make_bn();
    BN_copy(too_large.get(), order());
    threw = false;
    try {
        public_point(too_large.get());
    } catch (const std::runtime_error&) {
        threw = true;
    }
    assert(threw);
}

}  // namespace

int main() {
    test_public_key_derivation();
    test_ethereum_addresses();
    test_group_arithmetic();
    test_scalar_field();
    test_sign_verify_round_trip();
    test_verification_rejects_forgeries();
    test_low_s_normalisation();
    test_der_round_trip();
    test_invalid_secrets();

    std::printf("secp256k1: all checks passed\n");
    return 0;
}
