/**
 * Passkey Service - C++ Desktop Implementation
 * 
 * Complete Passkey/WebAuthn Features:
 * - Platform authenticator support
 * - Biometric integration
 * - Secure key storage
 * - Credential management
 * 
 * This service MUST be identical across ALL platforms.
 */

#ifndef PASSKEY_SERVICE_HPP
#define PASSKEY_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <ctime>
#include <cstdint>

namespace tigerwallet {

// Passkey credential
struct PasskeyCredential {
    std::string id;
    std::string publicKey;
    std::string algorithm;
    std::string counter;
    std::string rpId;
    std::string userId;
    time_t createdAt;
    time_t lastUsedAt;
    bool isActive;
};

// Passkey registration options
struct PasskeyRegistrationOptions {
    std::string rpId;
    std::string rpName;
    std::string userId;
    std::string userName;
    std::vector<std::string> pubKeyCredParams;
    uint32_t timeout;
    std::vector<std::string> excludeCredentials;
};

// Passkey assertion options
struct PasskeyAssertionOptions {
    std::string rpId;
    std::string challenge;
    uint32_t timeout;
    std::vector<std::string> allowedCredentials;
};

// Passkey verification result
struct PasskeyVerificationResult {
    bool success;
    std::string credentialId;
    std::string userId;
    std::string errorMessage;
    uint64_t signatureCount;
};

/**
 * Passkey Service - Production Implementation
 */
class PasskeyService {
public:
    static PasskeyService& getInstance();
    
    // Initialize passkey service
    bool initialize(const std::string& rpId, const std::string& rpName);
    
    // Check if passkey is available
    bool isPasskeyAvailable();
    
    // Check if passkey is enabled
    bool isPasskeyEnabled() const;
    
    // Enable/disable passkey
    bool setPasskeyEnabled(bool enabled);
    
    // Generate registration options
    PasskeyRegistrationOptions generateRegistrationOptions(
        const std::string& userId,
        const std::string& userName
    );
    
    // Generate assertion options
    PasskeyAssertionOptions generateAssertionOptions(
        const std::string& rpId,
        const std::string& challenge
    );
    
    // Register a new credential
    bool registerCredential(
        const std::string& credentialId,
        const std::string& publicKey,
        const std::string& algorithm,
        const std::string& rpId,
        const std::string& userId
    );
    
    // Authenticate with a credential
    PasskeyVerificationResult authenticate(
        const std::string& credentialId,
        const std::string& challenge,
        const std::vector<uint8_t>& clientDataHash,
        const std::vector<uint8_t>& authenticatorData,
        const std::vector<uint8_t>& signature
    );
    
    // Get all credentials for a relying party
    std::vector<PasskeyCredential> getCredentials(const std::string& rpId) const;
    
    // Get credential by ID
    PasskeyCredential* getCredential(const std::string& credentialId);
    
    // Remove a credential
    bool removeCredential(const std::string& credentialId);
    
    // Remove all credentials for a user
    bool removeAllCredentials();
    
    // Verify signature
    bool verifySignature(
        const std::string& credentialId,
        const std::vector<uint8_t>& clientDataHash,
        const std::vector<uint8_t>& authenticatorData,
        const std::vector<uint8_t>& signature
    );
    
    // Get relying party ID
    std::string getRpId() const;
    
    // Get relying party name
    std::string getRpName() const;
    
    // Get credential count
    size_t getCredentialCount() const;
    
private:
    PasskeyService();
    ~PasskeyService() = default;
    PasskeyService(const PasskeyService&) = delete;
    PasskeyService& operator=(const PasskeyService&) = delete;
    
    bool _initialized;
    bool _enabled;
    std::string _rpId;
    std::string _rpName;
    std::map<std::string, PasskeyCredential> _credentials;
    std::string _currentUserId;
    
    // Internal methods
    std::vector<uint8_t> generateChallenge();
    bool validateAuthenticatorData(
        const std::vector<uint8_t>& authenticatorData,
        const std::string& rpId
    );
    std::string computeSignatureHash(
        const std::vector<uint8_t>& authenticatorData,
        const std::vector<uint8_t>& clientDataHash
    );
};

} // namespace tigerwallet

#endif // PASSKEY_SERVICE_HPP
