#ifndef MASTER_WALLET_PASSKEY_SERVICE_HPP
#define MASTER_WALLET_PASSKEY_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <optional>
#include <chrono>
#include <mutex>
#include <atomic>
#include <stdexcept>
#include <cstdint>

namespace tiger {
namespace master {
namespace passkey {

// Forward declarations
class PasskeyService;
class WebAuthenticator;

/**
 * PasskeyCredential - FIDO2/WebAuthn credential
 */
struct PasskeyCredential {
    std::string id;
    std::string publicKey;
    std::string privateKey;  // Encrypted
    std::string counter;
    std::vector<std::string> transports;
    std::string aaguid;
    std::string label;
    int64_t createdAt;
    int64_t lastUsedAt;
    bool isResident;
    
    PasskeyCredential();
    
    std::vector<uint8_t> encode() const;
    static PasskeyCredential decode(const std::vector<uint8_t>& data);
};

/**
 * PublicKeyCredentialRpEntity - Relying Party information
 */
struct PublicKeyCredentialRpEntity {
    std::string id;
    std::string name;
    std::optional<std::string> icon;
    
    std::map<std::string, std::string> toJSON() const;
};

/**
 * PublicKeyCredentialUserEntity - User information
 */
struct PublicKeyCredentialUserEntity {
    std::vector<uint8_t> id;
    std::string name;
    std::optional<std::string> displayName;
    std::optional<std::string> icon;
    
    std::map<std::string, std::string> toJSON() const;
};

/**
 * PublicKeyCredentialParameters - Algorithm parameters
 */
struct PublicKeyCredentialParameters {
    std::string type;
    int alg;
    
    static std::vector<PublicKeyCredentialParameters> defaultAlgorithms();
    
    std::map<std::string, std::string> toJSON() const;
};

/**
 * AuthenticatorSelectionCriteria - Authenticator selection options
 */
struct AuthenticatorSelectionCriteria {
    std::optional<std::string> authenticatorAttachment;
    bool requireResidentKey;
    std::optional<std::string> userVerification;
    std::optional<std::string> residentKey;
    std::optional<std::string> attestation;
    
    std::map<std::string, std::string> toJSON() const;
};

/**
 * PublicKeyCredentialDescriptor - Credential descriptor
 */
struct PublicKeyCredentialDescriptor {
    std::string type;
    std::vector<uint8_t> id;
    std::vector<std::string> transports;
    
    std::map<std::string, std::string> toJSON() const;
};

/**
 * PublicKeyCredentialRequestOptions - Authentication options
 */
struct PublicKeyCredentialRequestOptions {
    std::vector<uint8_t> challenge;
    uint64_t timeout;
    std::string rpId;
    std::vector<PublicKeyCredentialDescriptor> allowCredentials;
    std::string userVerification;
    std::map<std::string, std::string> extensions;
    
    std::map<std::string, std::string> toJSON() const;
};

/**
 * PublicKeyCredentialCreationOptions - Registration options
 */
struct PublicKeyCredentialCreationOptions {
    PublicKeyCredentialRpEntity rp;
    PublicKeyCredentialUserEntity user;
    std::vector<PublicKeyCredentialParameters> pubKeyCredParams;
    std::vector<uint8_t> challenge;
    uint64_t timeout;
    std::optional<AuthenticatorSelectionCriteria> authenticatorSelection;
    std::vector<std::string> attestation;
    std::map<std::string, std::string> extensions;
    
    std::map<std::string, std::string> toJSON() const;
};

/**
 * AuthenticatorAttestationResponse - Registration response
 */
struct AuthenticatorAttestationResponse {
    std::vector<uint8_t> clientDataJSON;
    std::vector<uint8_t> attestationObject;
    std::optional<std::vector<std::string>> transports;
    std::optional<std::string> publicKeyAlgorithm;
    std::optional<std::vector<uint8_t>> publicKey;
    
    bool verify(const std::vector<uint8_t>& expectedChallenge) const;
    std::map<std::string, std::string> toJSON() const;
};

/**
 * AuthenticatorAssertionResponse - Authentication response
 */
struct AuthenticatorAssertionResponse {
    std::vector<uint8_t> clientDataJSON;
    std::vector<uint8_t> authenticatorData;
    std::vector<uint8_t> signature;
    std::optional<std::vector<uint8_t>> userHandle;
    
    bool verify(
        const std::vector<uint8_t>& publicKey,
        const std::vector<uint8_t>& expectedChallenge,
        const std::string& rpIdHash
    ) const;
    
    std::map<std::string, std::string> toJSON() const;
};

/**
 * PasskeyService - WebAuthn/FIDO2 Passkey implementation
 */
class PasskeyService {
public:
    explicit PasskeyService(const std::string& masterWalletId);
    ~PasskeyService();
    
    // Service lifecycle
    bool initialize();
    void shutdown();
    
    // Credential registration
    std::map<std::string, std::string> generateRegistrationOptions(
        const std::string& relyingPartyId,
        const std::string& relyingPartyName,
        const std::string& userId,
        const std::string& userName
    );
    
    bool registerPasskey(
        const std::map<std::string, std::string>& attestationResponse,
        std::string& credentialId
    );
    
    // Credential authentication
    std::map<std::string, std::string> generateAuthenticationOptions(
        const std::string& relyingPartyId
    );
    
    bool authenticateWithPasskey(
        const std::map<std::string, std::string>& assertionResponse,
        std::string& userId
    );

    // ==================== Backend-backed passkey flow ====================
    // register() generates a real P-256 (ES256) ECDSA keypair with OpenSSL,
    // derives a random 32-byte credential id (RAND_bytes), base64url-encodes
    // the SPKI public key + credential id, and POSTs them to the canonical
    // backend at /api/v1/master-wallet/:id/passkey/register. The backend is
    // authoritative for storing the credential; we never fabricate success.
    // On success, credentialId is set to the base64url credential id and
    // passkeyId to the backend-issued passkey id.
    struct RegisterResult {
        bool success = false;
        std::string credentialId;  // base64url
        std::string passkeyId;     // backend id
        std::string error;
    };
    RegisterResult register_(const std::string& label,
                             const std::vector<std::string>& transports);

    // verifyAssertion() forwards a WebAuthn assertion to the backend at
    // /api/v1/master-wallet/:id/passkey/verify-assertion. All fields are
    // base64url. As a defense-in-depth local fallback (NOT a substitute for
    // the backend), if the backend cannot be reached the assertion is
    // verified locally with real OpenSSL ECDSA P-256 verification against a
    // stored public key for the given credential id. Verification NEVER
    // succeeds without a real check.
    struct AssertionInput {
        std::string credentialId;      // base64url
        std::string authenticatorData; // base64url
        std::string clientDataJson;    // base64url
        std::string signature;         // base64url
    };
    struct VerifyAssertionResult {
        bool verified = false;
        std::string credentialId;
        std::string source;  // "backend" or "local"
        std::string error;
    };
    VerifyAssertionResult verifyAssertion(const AssertionInput& input);
    
    // Credential management
    std::vector<PasskeyCredential> listCredentials() const;
    bool deleteCredential(const std::string& credentialId);
    bool deleteAllCredentials();
    bool updateCredentialLabel(
        const std::string& credentialId,
        const std::string& label
    );
    
    // Device support
    bool isSupported() const;
    std::vector<std::string> getAvailableTransports() const;
    
    // Platform integration
    bool isPlatformAuthenticatorAvailable() const;
    bool isCrossPlatformAuthenticatorAvailable() const;
    
    // Statistics
    struct PasskeyStats {
        uint64_t totalCredentials;
        uint64_t totalRegistrations;
        uint64_t totalAuthentications;
        uint64_t failedAuthentications;
        double successRate;
        int64_t lastRegistrationTime;
        int64_t lastAuthenticationTime;
    };
    
    PasskeyStats getStats() const;
    void resetStats();

private:
    std::string masterWalletId_;
    std::vector<PasskeyCredential> credentials_;
    mutable std::mutex credentialsMutex_;
    
    std::atomic<uint64_t> totalRegistrations_{0};
    std::atomic<uint64_t> totalAuthentications_{0};
    std::atomic<uint64_t> failedAuthentications_{0};
    
    std::chrono::system_clock::time_point lastRegistrationTime_;
    std::chrono::system_clock::time_point lastAuthenticationTime_;
    
    std::string encryptionKey_;
    
    // Private methods
    std::vector<uint8_t> generateChallenge(size_t length);
    
    bool verifyAttestation(
        const AuthenticatorAttestationResponse& response,
        const std::vector<uint8_t>& expectedChallenge
    );
    
    bool verifyAssertion(
        const AuthenticatorAssertionResponse& response,
        const PasskeyCredential& credential,
        const std::vector<uint8_t>& expectedChallenge,
        const std::string& rpId
    );
    
    bool storeCredential(const PasskeyCredential& credential);
    bool removeCredential(const std::string& credentialId);
    
    std::optional<PasskeyCredential> getCredential(const std::string& credentialId) const;
    
    std::string encrypt(const std::string& data);
    std::string decrypt(const std::string& encryptedData);
    
    std::string computeSHA256(const std::vector<uint8_t>& data);
    std::string computeSHA256(const std::string& data);
    
    bool verifyECDSASignature(
        const std::vector<uint8_t>& publicKey,
        const std::vector<uint8_t>& message,
        const std::vector<uint8_t>& signature
    );

    // Lookup of the stored SPKI public key bytes for a credential id, used by
    // the local ECDSA fallback in verifyAssertion(). Returns std::nullopt when
    // the credential is unknown or has no public key.
    std::optional<std::vector<uint8_t>> credentialPublicKeyBytes(
        const std::string& credentialId) const;
};

// Inline implementations

inline PasskeyCredential::PasskeyCredential()
    : createdAt(0)
    , lastUsedAt(0)
    , isResident(false) {}

// Versioned binary format v1 ("TPC1"):
//   magic[4] | for each string field: u32be len + bytes | transports: u32be
//   count + items | createdAt i64be | lastUsedAt i64be | isResident u8.
// Field order is fixed: id, publicKey, privateKey, counter, aaguid, label.
namespace detail {
inline void putU32(std::vector<uint8_t>& out, uint32_t v) {
    out.push_back(static_cast<uint8_t>((v >> 24) & 0xff));
    out.push_back(static_cast<uint8_t>((v >> 16) & 0xff));
    out.push_back(static_cast<uint8_t>((v >> 8) & 0xff));
    out.push_back(static_cast<uint8_t>(v & 0xff));
}
inline void putI64(std::vector<uint8_t>& out, int64_t v) {
    for (int i = 7; i >= 0; --i)
        out.push_back(static_cast<uint8_t>((static_cast<uint64_t>(v) >> (i * 8)) & 0xff));
}
inline void putStr(std::vector<uint8_t>& out, const std::string& s) {
    putU32(out, static_cast<uint32_t>(s.size()));
    out.insert(out.end(), s.begin(), s.end());
}
struct Reader {
    const std::vector<uint8_t>& d;
    size_t pos = 0;
    explicit Reader(const std::vector<uint8_t>& v) : d(v) {}
    uint32_t u32() {
        if (pos + 4 > d.size()) throw std::runtime_error("passkey blob truncated");
        uint32_t v = (static_cast<uint32_t>(d[pos]) << 24) |
                     (static_cast<uint32_t>(d[pos + 1]) << 16) |
                     (static_cast<uint32_t>(d[pos + 2]) << 8) |
                     static_cast<uint32_t>(d[pos + 3]);
        pos += 4;
        return v;
    }
    int64_t i64() {
        if (pos + 8 > d.size()) throw std::runtime_error("passkey blob truncated");
        uint64_t v = 0;
        for (int i = 0; i < 8; ++i) v = (v << 8) | d[pos + i];
        pos += 8;
        return static_cast<int64_t>(v);
    }
    uint8_t u8() {
        if (pos + 1 > d.size()) throw std::runtime_error("passkey blob truncated");
        return d[pos++];
    }
    std::string str() {
        uint32_t n = u32();
        if (pos + n > d.size()) throw std::runtime_error("passkey blob truncated");
        std::string s(reinterpret_cast<const char*>(d.data() + pos), n);
        pos += n;
        return s;
    }
};
} // namespace detail

inline std::vector<uint8_t> PasskeyCredential::encode() const {
    std::vector<uint8_t> out{'T', 'P', 'C', '1'};
    detail::putStr(out, id);
    detail::putStr(out, publicKey);
    detail::putStr(out, privateKey);
    detail::putStr(out, counter);
    detail::putStr(out, aaguid);
    detail::putStr(out, label);
    detail::putU32(out, static_cast<uint32_t>(transports.size()));
    for (const auto& t : transports) detail::putStr(out, t);
    detail::putI64(out, createdAt);
    detail::putI64(out, lastUsedAt);
    out.push_back(isResident ? 1 : 0);
    return out;
}

inline PasskeyCredential PasskeyCredential::decode(const std::vector<uint8_t>& data) {
    if (data.size() < 4 || data[0] != 'T' || data[1] != 'P' || data[2] != 'C' ||
        data[3] != '1')
        throw std::runtime_error("unsupported passkey blob version");
    detail::Reader r(data);
    r.pos = 4;
    PasskeyCredential c;
    c.id = r.str();
    c.publicKey = r.str();
    c.privateKey = r.str();
    c.counter = r.str();
    c.aaguid = r.str();
    c.label = r.str();
    uint32_t n = r.u32();
    c.transports.reserve(n);
    for (uint32_t i = 0; i < n; ++i) c.transports.push_back(r.str());
    c.createdAt = r.i64();
    c.lastUsedAt = r.i64();
    c.isResident = r.u8() != 0;
    return c;
}

inline std::vector<PublicKeyCredentialParameters> 
PublicKeyCredentialParameters::defaultAlgorithms() {
    return {
        {"public-key", -7},  // ES256
        {"public-key", -257}, // RS256
    };
}

} // namespace passkey
} // namespace master
} // namespace tiger

#endif // MASTER_WALLET_PASSKEY_SERVICE_HPP
