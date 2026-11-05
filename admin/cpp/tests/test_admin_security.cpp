/**
 * Tests for the admin C++ security primitives.
 *
 * These exercise the real implementations in admin_security.cpp: the password
 * KDF, the AES-256-GCM envelope, the token generator, and the IP validators.
 * No mocks: a failure here means the production code is wrong.
 */

#include "admin_security.hpp"

#include <cstdlib>
#include <iostream>
#include <set>
#include <string>

using namespace tiger::admin;

namespace {

int g_checks = 0;
int g_failures = 0;

void check(bool condition, const std::string& what) {
    g_checks++;
    if (!condition) {
        g_failures++;
        std::cerr << "FAIL: " << what << "\n";
    }
}

void set_encryption_key(const char* value) {
    if (value == nullptr) {
        unsetenv("TIGERADMIN_ENCRYPTION_KEY");
    } else {
        setenv("TIGERADMIN_ENCRYPTION_KEY", value, 1);
    }
    SecurityService::instance().initialize();
}

// --- password hashing -------------------------------------------------------

void test_password_hash_is_salted() {
    set_encryption_key("0123456789abcdef0123456789abcdef");

    const std::string a = SecurityService::instance().hash_password("correct horse");
    const std::string b = SecurityService::instance().hash_password("correct horse");

    check(!a.empty(), "hash_password returns a value");
    check(a != b, "the same password hashes differently each time (random salt)");
    check(a.rfind("pbkdf2-sha256:", 0) == 0, "hash records the algorithm");
    check(a.find("correct horse") == std::string::npos,
          "the plaintext does not appear in the hash");
}

void test_password_verify() {
    set_encryption_key("0123456789abcdef0123456789abcdef");

    const std::string hash = SecurityService::instance().hash_password("s3cret-Pass!");

    check(SecurityService::instance().verify_password("s3cret-Pass!", hash),
          "the correct password verifies");
    check(!SecurityService::instance().verify_password("s3cret-Pass", hash),
          "a truncated password does not verify");
    check(!SecurityService::instance().verify_password("S3cret-Pass!", hash),
          "a case-changed password does not verify");
    check(!SecurityService::instance().verify_password("", hash),
          "an empty password does not verify");
}

void test_password_verify_rejects_malformed_records() {
    set_encryption_key("0123456789abcdef0123456789abcdef");

    // A bare SHA-256 digest was the old format. It must not verify, because
    // accepting it would keep the unsalted scheme alive.
    const std::string legacy = SecurityService::instance().hash_sha256("password");
    check(!SecurityService::instance().verify_password("password", legacy),
          "a bare SHA-256 digest does not verify");

    check(!SecurityService::instance().verify_password("x", ""),
          "an empty record does not verify");
    check(!SecurityService::instance().verify_password("x", "pbkdf2-sha256:notanumber:AA:AA"),
          "a non-numeric iteration count does not verify");
    check(!SecurityService::instance().verify_password("x", "pbkdf2-sha256:210000:!!:AA"),
          "an invalid base64 salt does not verify");
    check(!SecurityService::instance().verify_password("x", "md5:1:AA:AA"),
          "an unknown algorithm does not verify");
}

// --- encryption -------------------------------------------------------------

void test_encryption_round_trip() {
    set_encryption_key("0123456789abcdef0123456789abcdef");

    const std::string plaintext = "seed phrase and api credentials";
    const std::string ciphertext = SecurityService::instance().encrypt(plaintext);

    check(!ciphertext.empty(), "encrypt produces output");
    check(ciphertext != plaintext, "ciphertext differs from plaintext");
    check(ciphertext.find(plaintext) == std::string::npos,
          "the plaintext does not appear in the ciphertext");
    check(SecurityService::instance().decrypt(ciphertext) == plaintext,
          "decrypt recovers the plaintext");
}

void test_encryption_is_randomised() {
    set_encryption_key("0123456789abcdef0123456789abcdef");

    const std::string a = SecurityService::instance().encrypt("same input");
    const std::string b = SecurityService::instance().encrypt("same input");

    check(a != b, "encrypting the same input twice gives different ciphertext (fresh IV)");
    check(SecurityService::instance().decrypt(a) == "same input", "first decrypts");
    check(SecurityService::instance().decrypt(b) == "same input", "second decrypts");
}

void test_encryption_detects_tampering() {
    set_encryption_key("0123456789abcdef0123456789abcdef");

    const std::string ciphertext = SecurityService::instance().encrypt("transfer 100");

    // Flip one character of the payload. GCM must reject it rather than
    // returning altered plaintext.
    std::string tampered = ciphertext;
    tampered[tampered.size() - 1] =
        (tampered[tampered.size() - 1] == 'A') ? 'B' : 'A';
    check(SecurityService::instance().decrypt(tampered).empty(),
          "a tampered ciphertext is rejected");

    check(SecurityService::instance().decrypt("garbage").empty(),
          "a malformed envelope is rejected");
    check(SecurityService::instance().decrypt("").empty(),
          "an empty envelope is rejected");
}

void test_encryption_fails_closed_without_key() {
    set_encryption_key(nullptr);

    check(SecurityService::instance().encrypt("secret").empty(),
          "encrypt refuses to run with no key configured");
    check(SecurityService::instance().decrypt("AA:AA:AA").empty(),
          "decrypt refuses to run with no key configured");

    // A key shorter than 32 bytes is not an AES-256 key.
    set_encryption_key("tooshort");
    check(SecurityService::instance().encrypt("secret").empty(),
          "encrypt refuses a key shorter than 32 bytes");

    set_encryption_key("0123456789abcdef0123456789abcdef");
}

void test_decrypt_with_wrong_key_fails() {
    set_encryption_key("0123456789abcdef0123456789abcdef");
    const std::string ciphertext = SecurityService::instance().encrypt("sensitive");

    set_encryption_key("fedcba9876543210fedcba9876543210");
    check(SecurityService::instance().decrypt(ciphertext).empty(),
          "a ciphertext from another key does not decrypt");

    set_encryption_key("0123456789abcdef0123456789abcdef");
    check(SecurityService::instance().decrypt(ciphertext) == "sensitive",
          "the original key still decrypts");
}

// --- token generation -------------------------------------------------------

void test_token_length_and_uniqueness() {
    // generate_api_key() calls generate_token(48). The old implementation read
    // 16 bytes past a 32-byte buffer at that length.
    const std::string key = SecurityService::instance().generate_api_key();
    check(key.rfind("twak_", 0) == 0, "the api key carries its prefix");
    check(key.size() == 5 + 48, "the api key body is the requested length");

    std::set<std::string> seen;
    for (int i = 0; i < 200; i++) {
        seen.insert(SecurityService::instance().generate_token(32));
    }
    check(seen.size() == 200, "200 generated tokens are all distinct");

    for (const std::string& token : seen) {
        check(token.size() == 32, "each token has the requested length");
    }

    check(SecurityService::instance().generate_token(0).empty(),
          "a zero-length request returns nothing");
    check(SecurityService::instance().generate_token(-5).empty(),
          "a negative-length request returns nothing");
}

void test_token_alphabet() {
    const std::string token = SecurityService::instance().generate_token(256);
    const std::string allowed =
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    check(token.find_first_not_of(allowed) == std::string::npos,
          "tokens only use the documented alphabet");
}

// --- hashes -----------------------------------------------------------------

void test_known_digests() {
    // Fixed vectors so a change to the digest or the hex encoding is caught.
    check(SecurityService::instance().hash_sha256("abc") ==
              "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
          "sha256 matches the known digest for \"abc\"");
    check(SecurityService::instance().hash_sha256("").size() == 64,
          "sha256 of the empty string is still 64 hex characters");
    check(SecurityService::instance().hash_sha512("abc").size() == 128,
          "sha512 is 128 hex characters");
}

// --- IP validation ----------------------------------------------------------

void test_ip_validation() {
    SecurityService& s = SecurityService::instance();

    check(s.is_valid_ip("127.0.0.1"), "loopback is a valid IPv4");
    check(s.is_valid_ip("8.8.8.8"), "a public IPv4 is valid");
    check(s.is_valid_ip("255.255.255.255"), "the top of the IPv4 range is valid");
    check(!s.is_valid_ip("256.1.1.1"), "an octet above 255 is invalid");
    check(!s.is_valid_ip("1.2.3"), "a three-octet address is invalid");
    check(!s.is_valid_ip("1.2.3.4.5"), "a five-octet address is invalid");
    check(!s.is_valid_ip("01.2.3.4"), "a leading zero is invalid");
    check(!s.is_valid_ip(""), "an empty address is invalid");
    check(!s.is_valid_ip("not an ip"), "arbitrary text is invalid");
    check(s.is_valid_ip("::1"), "IPv6 loopback is valid");
    check(s.is_valid_ip("2001:db8::1"), "a documentation IPv6 is valid");
    check(!s.is_valid_ip("2001:db8:::1"), "a malformed IPv6 is invalid");
}

void test_private_ip_detection() {
    SecurityService& s = SecurityService::instance();

    check(s.is_private_ip("10.0.0.1"), "10/8 is private");
    check(s.is_private_ip("172.16.5.4"), "172.16/12 is private");
    check(s.is_private_ip("172.31.255.255"), "the top of 172.16/12 is private");
    check(!s.is_private_ip("172.32.0.1"), "just above 172.16/12 is public");
    check(s.is_private_ip("192.168.1.1"), "192.168/16 is private");
    check(s.is_private_ip("127.0.0.1"), "loopback is private");
    check(s.is_private_ip("169.254.1.1"), "link-local is private");
    check(!s.is_private_ip("8.8.8.8"), "a public address is not private");
    check(s.is_private_ip("::1"), "IPv6 loopback is private");
    check(s.is_private_ip("fd00::1"), "a unique local IPv6 is private");
}

// --- password policy and email ---------------------------------------------

void test_password_policy() {
    SecurityService& s = SecurityService::instance();

    check(s.is_strong_password("Str0ng!Pass"), "a mixed password is strong");
    check(!s.is_strong_password("alllowercase1!"), "no uppercase is weak");
    check(!s.is_strong_password("ALLUPPERCASE1!"), "no lowercase is weak");
    check(!s.is_strong_password("NoDigits!!x"), "no digit is weak");
    check(!s.is_strong_password("NoSpecial123"), "no special character is weak");
    check(!s.is_strong_password("Ab1!"), "short is weak even when mixed");
}

void test_email_validation() {
    SecurityService& s = SecurityService::instance();

    check(s.is_valid_email("admin@example.com"), "a normal address is valid");
    check(!s.is_valid_email("no-at-sign.com"), "a missing @ is invalid");
    check(!s.is_valid_email("@example.com"), "a missing local part is invalid");
    check(!s.is_valid_email("admin@"), "a missing domain is invalid");
    check(!s.is_valid_email("admin@example"), "a domain without a dot is invalid");
    check(!s.is_valid_email("a@b@c.com"), "two @ signs are invalid");
    check(!s.is_valid_email("admin @example.com"), "a space is invalid");
}

// --- feature flags ----------------------------------------------------------

void test_feature_flags_fail_closed() {
    // is_enabled used to return true for every name, including ones that were
    // never defined, so a flag gate could not turn anything off.
    check(!FeatureFlagService::instance().is_enabled("any-flag"),
          "an unknown flag is not reported as enabled");
    check(!FeatureFlagService::instance().is_enabled_for_user("any-flag", 1),
          "an unknown flag is not enabled for a user");
    check(!FeatureFlagService::instance().get_flag("any-flag").has_value(),
          "an unknown flag is not returned");
    check(!FeatureFlagService::instance().set_flag("x", "y", true, 100),
          "set_flag reports that it did not apply");
    check(!FeatureFlagService::instance().delete_flag("x"),
          "delete_flag reports that it did not apply");
}

}  // namespace

int main() {
    test_password_hash_is_salted();
    test_password_verify();
    test_password_verify_rejects_malformed_records();
    test_encryption_round_trip();
    test_encryption_is_randomised();
    test_encryption_detects_tampering();
    test_encryption_fails_closed_without_key();
    test_decrypt_with_wrong_key_fails();
    test_token_length_and_uniqueness();
    test_token_alphabet();
    test_known_digests();
    test_ip_validation();
    test_private_ip_detection();
    test_password_policy();
    test_email_validation();
    test_feature_flags_fail_closed();

    std::cout << (g_checks - g_failures) << "/" << g_checks << " checks passed\n";
    if (g_failures != 0) {
        std::cout << g_failures << " FAILED\n";
        return 1;
    }
    return 0;
}
