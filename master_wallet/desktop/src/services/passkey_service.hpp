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
};

// Inline implementations

inline PasskeyCredential::PasskeyCredential()
    : createdAt(0)
    , lastUsedAt(0)
    , isResident(false) {}

inline std::vector<uint8_t> PasskeyCredential::encode() const {
    // Credential serialization must be implemented with an explicit, versioned
    // format. Returning empty silently would lose data; fail closed.
    throw std::runtime_error("PasskeyCredential serialization is not implemented");
}

inline PasskeyCredential PasskeyCredential::decode(const std::vector<uint8_t>& /*data*/) {
    throw std::runtime_error("PasskeyCredential deserialization is not implemented");
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
