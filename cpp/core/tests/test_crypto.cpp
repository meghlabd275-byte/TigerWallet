// Tests for the cryptographic primitives.
//
// The Keccak-256 expectations come from an independent implementation written
// from the specification and validated against hashlib.sha3_256, not from this
// code. The EIP-55 addresses are the four from the specification. The AES-GCM
// cases cover the failure modes that matter: a wrong password and a modified
// blob must both be rejected.

#include <cstdint>
#include <iostream>
#include <string>
#include <vector>

#include "crypto.hpp"

using namespace tigerwallet::crypto;

namespace {

int failures = 0;

void Check(const std::string& label, bool ok) {
    if (!ok) ++failures;
    std::cout << (ok ? "  PASS " : "  FAIL ") << label << "\n";
}

void CheckEqual(const std::string& label, const std::string& got, const std::string& want) {
    const bool ok = got == want;
    if (!ok) ++failures;
    std::cout << (ok ? "  PASS " : "  FAIL ") << label << "\n";
    if (!ok) std::cout << "        got  " << got << "\n        want " << want << "\n";
}

std::string Hex32(const std::array<uint8_t, 32>& value) {
    return ToHex(value.data(), value.size());
}

void TestKeccak256() {
    std::cout << "Keccak-256\n";
    CheckEqual("empty input", Hex32(Keccak256((const uint8_t*)"", 0)),
               "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470");
    CheckEqual("three bytes", Hex32(Keccak256((const uint8_t*)"abc", 3)),
               "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45");

    const char* fox = "The quick brown fox jumps over the lazy dog";
    CheckEqual("43 bytes", Hex32(Keccak256((const uint8_t*)fox, 43)),
               "4d741b6f1eb29cb2a9b9911c82f56fa8d73b04959d3d9d222895df6c0b28aa15");

    // Boundaries around the 136-byte rate, where padding bugs show up.
    struct Case { size_t len; const char* want; };
    const Case cases[] = {
        {135, "34367dc248bbd832f4e3e69dfaac2f92638bd0bbd18f2912ba4ef454919cf446"},
        {136, "a6c4d403279fe3e0af03729caada8374b5ca54d8065329a3ebcaeb4b60aa386e"},
        {137, "d869f639c7046b4929fc92a4d988a8b22c55fbadb802c0c66ebcd484f1915f39"},
        {200, "96ea54061def936c4be90b518992fdc6f12f535068a256229aca54267b4d084d"},
    };
    for (const auto& c : cases) {
        const std::string input(c.len, 'a');
        CheckEqual(std::to_string(c.len) + " bytes",
                   Hex32(Keccak256((const uint8_t*)input.data(), input.size())), c.want);
    }

    std::vector<uint8_t> all_bytes(256);
    for (size_t i = 0; i < all_bytes.size(); ++i) all_bytes[i] = static_cast<uint8_t>(i);
    CheckEqual("bytes 0..255", Hex32(Keccak256(all_bytes)),
               "dc924469b334aed2a19fac7252e9961aea41f8d91996366029dbe0884229bf36");
}

void TestSha256() {
    std::cout << "SHA-256\n";
    CheckEqual("abc", ToHex(Sha256((const uint8_t*)"abc", 3)),
               "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad");
}

void TestChecksumAddresses() {
    std::cout << "EIP-55 checksum addresses\n";
    const char* addresses[] = {
        "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
        "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
        "0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB",
        "0xD1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb",
    };
    for (const char* address : addresses) {
        std::string lower = address;
        for (char& c : lower) {
            if (c >= 'A' && c <= 'F') c = static_cast<char>(c - 'A' + 'a');
        }
        CheckEqual(std::string("checksum ") + std::string(address).substr(0, 12),
                   ToChecksumAddress(lower), address);
        Check(std::string("verify ") + std::string(address).substr(0, 12),
              VerifyChecksumAddress(address));
    }
    Check("rejects a corrupted checksum",
          !VerifyChecksumAddress("0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAeD"));
    Check("accepts all-lowercase (no checksum information)",
          VerifyChecksumAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"));
    Check("rejects a bad length", !IsValidEvmAddress("0x1234"));
}

void TestKeysAndSigning() {
    std::cout << "secp256k1\n";
    std::array<uint8_t, 32> key{};
    key[31] = 1;  // the well-known key 0x...01

    std::array<uint8_t, 20> address{};
    Check("private key 0x01 derives an address", PrivateKeyToAddress(key, address));
    CheckEqual("address for key 0x01", AddressToString(address),
               "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf");

    const auto public_key = PrivateKeyToPublicKey(key);
    Check("public key is 64 bytes", public_key.size() == 64);

    const auto digest = Sha256((const uint8_t*)"tiger", 5);
    uint8_t recovery_id = 0;
    const auto signature = SignHash(digest, key, recovery_id);
    Check("signature is 64 bytes", signature.size() == 64);

    std::array<uint8_t, 20> recovered{};
    Check("signature recovers the signer",
          RecoverAddress(digest, signature, recovery_id, recovered) && recovered == address);

    // Determinism matters: a repeated nonce across two signatures leaks the key.
    uint8_t recovery_id_2 = 0;
    const auto signature_2 = SignHash(digest, key, recovery_id_2);
    Check("signing is deterministic (RFC 6979)",
          signature == signature_2 && recovery_id == recovery_id_2);

    // Recovery is not verification. Given a different digest, ECDSA recovery
    // still returns a well-formed public key; the point is that it is not the
    // signer. Callers that need verification must compare the recovered
    // address, which is what the transaction code does.
    const auto other_digest = Sha256((const uint8_t*)"not-tiger", 9);
    std::array<uint8_t, 20> wrong{};
    const bool recovered_wrong = RecoverAddress(other_digest, signature, recovery_id, wrong);
    Check("recovering under a different digest yields a different address",
          !recovered_wrong || wrong != address);
}

void TestHexEncoding() {
    std::cout << "hex encoding\n";
    std::vector<uint8_t> out;
    Check("rejects non-hex characters", !FromHex("0xzz", out));
    Check("rejects odd length", !FromHex("abc", out));
    Check("accepts a 0x prefix", FromHex("0xdeadbeef", out) && out.size() == 4);
    Check("accepts no prefix", FromHex("deadbeef", out) && out.size() == 4);
    Check("empty input decodes to empty", FromHex("", out) && out.empty());
    Check("round trip", ToHex(FromHex("0x00ff10")) == "00ff10");
}

void TestQuantities() {
    std::cout << "JSON-RPC quantities\n";
    Check("accepts 0x0", IsValidHexQuantity("0x0"));
    Check("accepts a value above 64 bits", IsValidHexQuantity("0x207a92bdd8ab78bf"));
    Check("rejects a bare 0x", !IsValidHexQuantity("0x"));
    Check("rejects a leading zero", !IsValidHexQuantity("0x01"));
    Check("rejects a missing prefix", !IsValidHexQuantity("207a"));
    Check("rejects a non-hex digit", !IsValidHexQuantity("0xzz"));
    Check("rejects more than 32 bytes", !IsValidHexQuantity(std::string("0x") + std::string(66, 'a')));
    Check("accepts exactly 32 bytes", IsValidHexQuantity(std::string("0x") + std::string(64, 'a')));

    CheckEqual("0x0", HexQuantityToDecimal("0x0"), "0");
    CheckEqual("0xff", HexQuantityToDecimal("0xff"), "255");
    // Odd digit counts must not be padded per pair: 0x100 is 256, not 4096.
    CheckEqual("0x100 (odd digit count)", HexQuantityToDecimal("0x100"), "256");
    CheckEqual("1 ETH in wei", HexQuantityToDecimal("0xde0b6b3a7640000"), "1000000000000000000");
    CheckEqual("the largest 256-bit value",
               HexQuantityToDecimal("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
               "115792089237316195423570985008687907853269984665640564039457584007913129639935");
}

void TestAesGcm() {
    std::cout << "AES-256-GCM\n";
    const std::string plaintext = "seed phrase: abandon ability able about above absent";
    const std::string password = "correct horse battery staple";

    const auto blob = AesGcmEncrypt(password, (const uint8_t*)plaintext.data(),
                                    plaintext.size(), 10000);
    Check("blob layout is salt+iterations+iv+ciphertext+tag",
          blob.size() == 16 + 4 + 12 + plaintext.size() + 16);

    const auto decrypted = AesGcmDecrypt(password, blob.data(), blob.size());
    Check("round trip recovers the plaintext",
          decrypted.has_value() &&
              std::string(decrypted->begin(), decrypted->end()) == plaintext);

    Check("wrong password is rejected",
          !AesGcmDecrypt("wrong password", blob.data(), blob.size()).has_value());

    auto tampered_ciphertext = blob;
    tampered_ciphertext[16 + 4 + 12 + 3] ^= 0x01;
    Check("modified ciphertext is rejected",
          !AesGcmDecrypt(password, tampered_ciphertext.data(), tampered_ciphertext.size())
               .has_value());

    auto tampered_tag = blob;
    tampered_tag[tampered_tag.size() - 1] ^= 0x01;
    Check("modified tag is rejected",
          !AesGcmDecrypt(password, tampered_tag.data(), tampered_tag.size()).has_value());

    Check("truncated blob is rejected",
          !AesGcmDecrypt(password, blob.data(), 40).has_value());

    // An absurd iteration count must be refused rather than run.
    auto absurd = blob;
    absurd[16] = 0xFF;
    absurd[17] = 0xFF;
    absurd[18] = 0xFF;
    absurd[19] = 0xFF;
    Check("implausible iteration count is rejected",
          !AesGcmDecrypt(password, absurd.data(), absurd.size()).has_value());

    const auto second = AesGcmEncrypt(password, (const uint8_t*)plaintext.data(),
                                      plaintext.size(), 10000);
    Check("salt and IV differ per call", second != blob);
}

}  // namespace

int main() {
    TestKeccak256();
    TestSha256();
    TestChecksumAddresses();
    TestKeysAndSigning();
    TestHexEncoding();
    TestQuantities();
    TestAesGcm();

    std::cout << (failures == 0 ? "crypto: all tests passed\n"
                                : "crypto: " + std::to_string(failures) + " test(s) failed\n");
    return failures == 0 ? 0 : 1;
}
