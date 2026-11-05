/**
 * TigerAdmin C++ Core - Security Implementation
 */

#include "admin_security.hpp"
#include "admin_logger.hpp"
#include <openssl/evp.h>
#include <openssl/rand.h>
#include <openssl/crypto.h>
#include <arpa/inet.h>
#include <cctype>
#include <cstdlib>
#include <cstring>
#include <vector>

namespace tiger {
namespace admin {

namespace {

constexpr int kPasswordSaltBytes = 16;
constexpr int kPasswordKeyBytes = 32;
constexpr int kPasswordIterations = 210000;  // OWASP guidance for PBKDF2-HMAC-SHA256
constexpr int kGcmIvBytes = 12;
constexpr int kGcmTagBytes = 16;

// base64url without padding, used to serialise binary blobs into the
// colon-separated text form the column-oriented storage expects. Padding is
// omitted (not emitted as '='), which is the URL-safe convention and the form
// the decoder below accepts.
std::string base64_url_encode(const unsigned char* data, size_t len) {
    static const char table[] =
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
    std::string out;
    out.reserve((len + 2) / 3 * 4);
    for (size_t i = 0; i < len; i += 3) {
        const size_t remaining = len - i;
        const unsigned int b0 = data[i];
        const unsigned int b1 = (remaining > 1) ? data[i + 1] : 0;
        const unsigned int b2 = (remaining > 2) ? data[i + 2] : 0;
        const unsigned int triple = (b0 << 16) | (b1 << 8) | b2;
        out.push_back(table[(triple >> 18) & 0x3F]);
        out.push_back(table[(triple >> 12) & 0x3F]);
        if (remaining > 1) out.push_back(table[(triple >> 6) & 0x3F]);
        if (remaining > 2) out.push_back(table[triple & 0x3F]);
    }
    return out;
}

int base64_url_value(char c) {
    if (c >= 'A' && c <= 'Z') return c - 'A';
    if (c >= 'a' && c <= 'z') return c - 'a' + 26;
    if (c >= '0' && c <= '9') return c - '0' + 52;
    if (c == '-') return 62;
    if (c == '_') return 63;
    return -1;
}

// Strict: rejects any character outside the alphabet.
bool base64_url_decode(const std::string& in, std::string& out) {
    out.clear();
    unsigned int buffer = 0;
    int bits = 0;
    for (char c : in) {
        const int v = base64_url_value(c);
        if (v < 0) return false;
        buffer = (buffer << 6) | static_cast<unsigned int>(v);
        bits += 6;
        if (bits >= 8) {
            bits -= 8;
            out.push_back(static_cast<char>((buffer >> bits) & 0xFF));
        }
    }
    return true;
}

std::string hex_encode(const unsigned char* data, size_t len) {
    static const char digits[] = "0123456789abcdef";
    std::string out;
    out.reserve(len * 2);
    for (size_t i = 0; i < len; i++) {
        out.push_back(digits[(data[i] >> 4) & 0x0F]);
        out.push_back(digits[data[i] & 0x0F]);
    }
    return out;
}

std::vector<std::string> split_on(const std::string& s, char sep) {
    std::vector<std::string> parts;
    size_t start = 0;
    while (true) {
        const size_t pos = s.find(sep, start);
        if (pos == std::string::npos) {
            parts.push_back(s.substr(start));
            break;
        }
        parts.push_back(s.substr(start, pos - start));
        start = pos + 1;
    }
    return parts;
}

// Derives a key with PBKDF2-HMAC-SHA256.
bool derive_key(const std::string& password, const std::string& salt,
                int iterations, unsigned char* out, int out_len) {
    return PKCS5_PBKDF2_HMAC(password.data(), static_cast<int>(password.size()),
                             reinterpret_cast<const unsigned char*>(salt.data()),
                             static_cast<int>(salt.size()), iterations,
                             EVP_sha256(), out_len, out) == 1;
}

// Constant-time comparison. Both operands are hashed first so that a length
// difference cannot leak through an early exit in CRYPTO_memcmp.
bool constant_time_equal(const std::string& a, const std::string& b) {
    unsigned char ha[32];
    unsigned char hb[32];
    unsigned int ha_len = 0;
    unsigned int hb_len = 0;
    EVP_Digest(a.data(), a.size(), ha, &ha_len, EVP_sha256(), nullptr);
    EVP_Digest(b.data(), b.size(), hb, &hb_len, EVP_sha256(), nullptr);
    return CRYPTO_memcmp(ha, hb, sizeof(ha)) == 0;
}

}  // namespace

SecurityService& SecurityService::instance() {
    static SecurityService service;
    return service;
}

void SecurityService::initialize() {
    // The key comes from the environment. A key committed to the repository
    // would be known to everyone with a copy of the source, so there is no
    // fallback: without TIGERADMIN_ENCRYPTION_KEY, encryption refuses to run
    // and the operator sees an explicit error instead of silently weak
    // ciphertext.
    const char* key = std::getenv("TIGERADMIN_ENCRYPTION_KEY");
    if (key != nullptr && std::strlen(key) >= 32) {
        encryption_key_ = key;
        encryption_ready_ = true;
        LOG_INFO("Security service initialized (encryption key present)");
    } else {
        encryption_key_.clear();
        encryption_ready_ = false;
        LOG_ERROR("TIGERADMIN_ENCRYPTION_KEY is unset or shorter than 32 bytes; "
                  "encryption is disabled and will fail closed");
    }
}

std::string SecurityService::encrypt(const std::string& plaintext) {
    return aes_encrypt(plaintext);
}

std::string SecurityService::decrypt(const std::string& ciphertext) {
    return aes_decrypt(ciphertext);
}

std::string SecurityService::hash_sha256(const std::string& data) {
    unsigned char hash[32];
    unsigned int hash_len = 0;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (ctx == nullptr) return "";
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    EVP_DigestUpdate(ctx, data.c_str(), data.size());
    EVP_DigestFinal_ex(ctx, hash, &hash_len);
    EVP_MD_CTX_free(ctx);
    return hex_encode(hash, hash_len);
}

std::string SecurityService::hash_sha512(const std::string& data) {
    unsigned char hash[64];
    unsigned int hash_len = 0;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (ctx == nullptr) return "";
    EVP_DigestInit_ex(ctx, EVP_sha512(), nullptr);
    EVP_DigestUpdate(ctx, data.c_str(), data.size());
    EVP_DigestFinal_ex(ctx, hash, &hash_len);
    EVP_MD_CTX_free(ctx);
    return hex_encode(hash, hash_len);
}

// Passwords use PBKDF2-HMAC-SHA256 with a per-password random salt. A bare
// SHA-256 of the password (what this used to return) is reversible with a
// rainbow table and identical for every admin who chose the same password.
std::string SecurityService::hash_password(const std::string& password) {
    unsigned char salt[kPasswordSaltBytes];
    if (RAND_bytes(salt, sizeof(salt)) != 1) {
        LOG_ERROR("hash_password: RAND_bytes failed");
        return "";
    }

    const std::string salt_str(reinterpret_cast<char*>(salt), sizeof(salt));

    unsigned char key[kPasswordKeyBytes];
    if (!derive_key(password, salt_str, kPasswordIterations, key, sizeof(key))) {
        LOG_ERROR("hash_password: PBKDF2 failed");
        return "";
    }

    return "pbkdf2-sha256:" + std::to_string(kPasswordIterations) + ":" +
           base64_url_encode(salt, sizeof(salt)) + ":" +
           base64_url_encode(key, sizeof(key));
}

bool SecurityService::verify_password(const std::string& password,
                                      const std::string& hash) {
    const std::vector<std::string> parts = split_on(hash, ':');
    if (parts.size() != 4 || parts[0] != "pbkdf2-sha256") {
        return false;
    }

    int iterations = 0;
    try {
        iterations = std::stoi(parts[1]);
    } catch (...) {
        return false;
    }
    if (iterations <= 0 || iterations > 10000000) return false;

    std::string salt;
    std::string expected;
    if (!base64_url_decode(parts[2], salt)) return false;
    if (!base64_url_decode(parts[3], expected)) return false;
    if (salt.empty() || expected.empty()) return false;

    std::vector<unsigned char> key(expected.size());
    if (!derive_key(password, salt, iterations, key.data(),
                    static_cast<int>(key.size()))) {
        return false;
    }

    const std::string computed(key.begin(), key.end());
    return constant_time_equal(computed, expected);
}

std::string SecurityService::generate_token(int length) {
    if (length <= 0) return "";

    static const char chars[] =
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    constexpr unsigned int kAlphabet = sizeof(chars) - 1;

    // Rejection sampling: a plain modulo of random bytes is biased whenever
    // the alphabet does not divide 256. The old code also indexed a 32-byte
    // buffer with `i` up to `length`, so generate_api_key() (length 48) read
    // 16 bytes past the end of the array.
    const unsigned int limit = 256 - (256 % kAlphabet);

    std::string token;
    token.reserve(static_cast<size_t>(length));
    unsigned char buf[64];
    while (token.size() < static_cast<size_t>(length)) {
        if (RAND_bytes(buf, sizeof(buf)) != 1) {
            LOG_ERROR("generate_token: RAND_bytes failed");
            return "";
        }
        for (unsigned char b : buf) {
            if (b >= limit) continue;
            token.push_back(chars[b % kAlphabet]);
            if (token.size() == static_cast<size_t>(length)) break;
        }
    }
    return token;
}

std::string SecurityService::generate_api_key() {
    return "twak_" + generate_token(48);
}

bool SecurityService::is_valid_email(const std::string& email) {
    const size_t at = email.find('@');
    if (at == std::string::npos || at == 0 || at + 1 >= email.size()) {
        return false;
    }
    if (email.find('@', at + 1) != std::string::npos) return false;
    if (email.find(' ') != std::string::npos) return false;
    const size_t dot = email.find('.', at + 1);
    return dot != std::string::npos && dot > at + 1 && dot + 1 < email.size();
}

bool SecurityService::is_valid_password(const std::string& password) {
    return password.length() >= 8;
}

bool SecurityService::is_strong_password(const std::string& password) {
    bool has_upper = false, has_lower = false, has_digit = false, has_special = false;
    for (unsigned char c : password) {
        if (std::isupper(c)) has_upper = true;
        if (std::islower(c)) has_lower = true;
        if (std::isdigit(c)) has_digit = true;
        if (!std::isalnum(c)) has_special = true;
    }
    return password.length() >= 8 && has_upper && has_lower && has_digit &&
           has_special;
}

std::string SecurityService::sanitize_string(const std::string& input) {
    std::string output;
    output.reserve(input.size());
    for (unsigned char c : input) {
        if (c >= 32 && c < 127) output += static_cast<char>(c);
    }
    return output;
}

// Escaping quotes is not a defence against SQL injection on its own: it does
// nothing about numeric context or comment sequences, and it depends on the
// backend treating backslash escapes the same way. Every query in this module
// goes through the upstream service, which uses bound parameters; this helper
// only makes a value safe to embed in an already-fixed statement template.
std::string SecurityService::sanitize_sql(const std::string& input) {
    std::string output;
    output.reserve(input.size());
    for (char c : input) {
        if (c == '\'') output += "''";
        else if (c == '\\') output += "\\\\";
        else if (c == '\0') continue;
        else output += c;
    }
    return output;
}

bool SecurityService::is_valid_ip(const std::string& ip) {
    if (ip.empty() || ip.size() > 45) return false;

    // inet_pton is the authority for both families; it rejects leading zeros,
    // out-of-range octets and malformed groups.
    unsigned char buf[16];
    if (inet_pton(AF_INET, ip.c_str(), buf) == 1) return true;
    if (inet_pton(AF_INET6, ip.c_str(), buf) == 1) return true;
    return false;
}

bool SecurityService::is_private_ip(const std::string& ip) {
    // RFC 1918 plus loopback and link-local, so an operator cannot whitelist
    // an address that is not routable from the internet and then believe
    // remote traffic is being filtered.
    unsigned char buf[16];
    if (inet_pton(AF_INET, ip.c_str(), buf) == 1) {
        return buf[0] == 10 ||
               (buf[0] == 172 && buf[1] >= 16 && buf[1] <= 31) ||
               (buf[0] == 192 && buf[1] == 168) ||
               buf[0] == 127 ||
               (buf[0] == 169 && buf[1] == 254);
    }
    if (inet_pton(AF_INET6, ip.c_str(), buf) == 1) {
        if (buf[0] == 0xFE && (buf[1] & 0xC0) == 0x80) return true;  // fe80::/10
        bool loopback = true;
        for (int i = 0; i < 15; i++) {
            if (buf[i] != 0) { loopback = false; break; }
        }
        if (loopback && buf[15] == 1) return true;  // ::1
        return (buf[0] & 0xFE) == 0xFC;             // fc00::/7
    }
    return false;
}

// AES-256-GCM. The output is iv:tag:ciphertext, each part base64url, so the
// three can be stored in separate columns and the IV is never reused.
std::string SecurityService::aes_encrypt(const std::string& plaintext) {
    if (!encryption_ready_) {
        LOG_ERROR("encrypt: no TIGERADMIN_ENCRYPTION_KEY; refusing to encrypt");
        return "";
    }

    unsigned char iv[kGcmIvBytes];
    if (RAND_bytes(iv, sizeof(iv)) != 1) {
        LOG_ERROR("encrypt: RAND_bytes failed");
        return "";
    }

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (ctx == nullptr) return "";

    std::string ciphertext;
    std::string tag;
    bool ok = false;

    do {
        if (EVP_EncryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) break;
        if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, sizeof(iv), nullptr) != 1) break;
        if (EVP_EncryptInit_ex(
                ctx, nullptr, nullptr,
                reinterpret_cast<const unsigned char*>(encryption_key_.data()),
                iv) != 1) break;

        std::vector<unsigned char> out(plaintext.size() + EVP_MAX_BLOCK_LENGTH);
        int written = 0;
        if (EVP_EncryptUpdate(ctx, out.data(), &written,
                              reinterpret_cast<const unsigned char*>(plaintext.data()),
                              static_cast<int>(plaintext.size())) != 1) break;
        int total = written;

        if (EVP_EncryptFinal_ex(ctx, out.data() + total, &written) != 1) break;
        total += written;

        unsigned char tag_buf[kGcmTagBytes];
        if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, sizeof(tag_buf), tag_buf) != 1) break;

        ciphertext.assign(reinterpret_cast<char*>(out.data()), static_cast<size_t>(total));
        tag.assign(reinterpret_cast<char*>(tag_buf), sizeof(tag_buf));
        ok = true;
    } while (false);

    EVP_CIPHER_CTX_free(ctx);

    if (!ok) {
        LOG_ERROR("encrypt: AES-256-GCM failed");
        return "";
    }

    return base64_url_encode(iv, sizeof(iv)) + ":" +
           base64_url_encode(reinterpret_cast<const unsigned char*>(tag.data()), tag.size()) + ":" +
           base64_url_encode(reinterpret_cast<const unsigned char*>(ciphertext.data()), ciphertext.size());
}

std::string SecurityService::aes_decrypt(const std::string& ciphertext) {
    if (!encryption_ready_) {
        LOG_ERROR("decrypt: no TIGERADMIN_ENCRYPTION_KEY; refusing to decrypt");
        return "";
    }

    const std::vector<std::string> parts = split_on(ciphertext, ':');
    if (parts.size() != 3) {
        LOG_ERROR("decrypt: ciphertext is not in iv:tag:data form");
        return "";
    }

    std::string iv;
    std::string tag;
    std::string data;
    if (!base64_url_decode(parts[0], iv)) return "";
    if (!base64_url_decode(parts[1], tag)) return "";
    if (!base64_url_decode(parts[2], data)) return "";
    if (iv.size() != kGcmIvBytes || tag.size() != kGcmTagBytes) {
        LOG_ERROR("decrypt: malformed iv or tag length");
        return "";
    }

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (ctx == nullptr) return "";

    std::string plaintext;
    bool ok = false;

    do {
        if (EVP_DecryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) break;
        if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN,
                                static_cast<int>(iv.size()), nullptr) != 1) break;
        if (EVP_DecryptInit_ex(
                ctx, nullptr, nullptr,
                reinterpret_cast<const unsigned char*>(encryption_key_.data()),
                reinterpret_cast<const unsigned char*>(iv.data())) != 1) break;

        std::vector<unsigned char> out(data.size() + EVP_MAX_BLOCK_LENGTH);
        int written = 0;
        if (EVP_DecryptUpdate(ctx, out.data(), &written,
                              reinterpret_cast<const unsigned char*>(data.data()),
                              static_cast<int>(data.size())) != 1) break;
        int total = written;

        // The tag is checked in DecryptFinal: a tampered ciphertext, IV or tag
        // fails here rather than returning altered plaintext.
        if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG,
                                static_cast<int>(tag.size()),
                                const_cast<char*>(tag.data())) != 1) break;
        if (EVP_DecryptFinal_ex(ctx, out.data() + total, &written) != 1) {
            LOG_ERROR("decrypt: authentication tag mismatch");
            break;
        }
        total += written;

        plaintext.assign(reinterpret_cast<char*>(out.data()), static_cast<size_t>(total));
        ok = true;
    } while (false);

    EVP_CIPHER_CTX_free(ctx);

    if (!ok) return "";
    return plaintext;
}

// FeatureFlag Service
//
// Flags are stored by the upstream admin backend. This process holds no flag
// store, so reads report "not found" and writes report "not applied" instead
// of returning success for an operation that did nothing.
FeatureFlagService& FeatureFlagService::instance() {
    static FeatureFlagService service;
    return service;
}

void FeatureFlagService::initialize() {
    LOG_INFO("Feature flag service initialized (flags are served by the upstream "
             "admin backend)");
}

std::optional<FeatureFlag> FeatureFlagService::get_flag(const std::string& name) {
    (void)name;
    return std::nullopt;
}

std::vector<FeatureFlag> FeatureFlagService::list_flags() { return {}; }

bool FeatureFlagService::set_flag(const std::string& name, const std::string& description,
    bool is_enabled, int rollout_percentage) {
    (void)name;
    (void)description;
    (void)is_enabled;
    (void)rollout_percentage;
    LOG_ERROR("set_flag: feature flags are written by the upstream admin backend");
    return false;
}

bool FeatureFlagService::update_flag(const std::string& name, const std::optional<bool>& is_enabled,
    const std::optional<int>& rollout_percentage) {
    (void)name;
    (void)is_enabled;
    (void)rollout_percentage;
    LOG_ERROR("update_flag: feature flags are written by the upstream admin backend");
    return false;
}

bool FeatureFlagService::delete_flag(const std::string& name) {
    (void)name;
    LOG_ERROR("delete_flag: feature flags are written by the upstream admin backend");
    return false;
}

bool FeatureFlagService::is_enabled(const std::string& name) {
    // A flag whose value is unknown must not report "on". This returned true
    // unconditionally, so a caller that gates a feature on is_enabled() would
    // enable it even when the flag did not exist.
    (void)name;
    return false;
}

bool FeatureFlagService::is_enabled_for_user(const std::string& name, AdminID admin_id) {
    (void)name;
    (void)admin_id;
    return false;
}

} // namespace admin
} // namespace tiger
