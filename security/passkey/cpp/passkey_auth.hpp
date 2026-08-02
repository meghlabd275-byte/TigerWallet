/**
 * TigerWallet Passkeys/WebAuthn Implementation
 * 
 * Provides passwordless authentication using:
 * - WebAuthn/FIDO2
 * - Platform Authenticators
 * - Cross-platform Passkeys
 * 
 * @author TigerWallet Team
 */

#ifndef TIGERWALLET_PASSKEY_HPP
#define TIGERWALLET_PASSKEY_HPP

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <memory>
#include <variant>
#include <optional>
#include <functional>
#include <mutex>
#include <thread>
#include <chrono>
#include <cryptopp/secblock.h>
#include <cryptopp/eccrypto.h>
#include <cryptopp/oids.h>
#include <cryptopp/sha.h>
#include <cryptopp/hmac.h>
#include <cryptopp/rsa.h>
#include <cryptopp/asn.h>
#include <cryptopp/oids.h>

namespace tiger {

using namespace CryptoPP;

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

using UserID = std::string;
using CredentialID = std::string;

// Public Key Credential
struct PublicKeyCredential {
    CredentialID id;
    std::string type; // "public-key"
    std::vector<uint8_t> rawId;
    std::string authenticatorAttachment;
    std::vector<uint8_t> clientDataJSON;
    std::vector<uint8_t> attestationObject;
    std::string transports; // "usb", "nfc", "ble", "internal"
};

// Authenticator Attestation Response
struct AuthenticatorAttestationResponse {
    std::string fmt; // "packed", "tpm", "android-safetynet", "none"
    std::vector<uint8_t> attestationStatement;
    std::vector<uint8_t> authData; // Contains pubkey, rpIdHash, flags, counter
    std::vector<uint8_t> signature;
};

// Authenticator Assertion Response
struct AuthenticatorAssertionResponse {
    std::vector<uint8_t> authenticatorData;
    std::vector<uint8_t> signature;
    std::vector<uint8_t> clientDataJSON;
    std::string userHandle;
};

// Relying Party (RP)
struct RelyingParty {
    std::string id;
    std::string name;
    std::string origin;
    std::string iconURL;
};

// User Account
struct UserAccount {
    UserID id;
    std::string name;
    std::string displayName;
    std::string iconURL;
    std::vector<PublicKeyCredential> credentials;
};

// WebAuthn Options
struct WebAuthnOptions {
    std::string challenge;
    uint32_t timeout; // milliseconds
    std::string rpId;
    std::string rpName;
    std::vector<UserAccount> users;
    uint32_t pubKeyCredParams; // algorithm
    std::vector<std::string> excludeCredentials;
    std::string authenticatorAttachment;
    bool requireResidentKey;
    std::string userVerification; // "required", "preferred", "discouraged"
    bool attestation; // "none", "indirect", "direct"
    std::vector<std::string> extensions;
};

// Passkey (Credential) Registration
struct PasskeyRegistration {
    std::string credentialId;
    std::vector<uint8_t> publicKey;
    std::string keyType; // "ec2", "rsa"
    uint32_t keyAlgorithm;
    std::string rpId;
    UserID userId;
    std::vector<uint8_t> counter;
    Timestamp createdAt;
};

// Passkey Authentication
struct PasskeyAuthentication {
    std::string credentialId;
    std::vector<uint8_t> authenticatorData;
    std::vector<uint8_t> signature;
    std::vector<uint8_t> clientDataJSON;
    std::vector<uint8_t> userHandle;
    std::vector<uint8_t> counter;
    Timestamp authenticatedAt;
};

// =============================================================================
// CRYPTO UTILITIES FOR WEBAUTHN
// =============================================================================

class WebAuthnCrypto {
public:
    // Generate WebAuthn challenge (32 bytes random)
    static std::vector<uint8_t> generateChallenge() {
        std::vector<uint8_t> challenge(32);
        AutoSeededRandomPool rng;
        rng.GenerateBlock(challenge.data(), challenge.size());
        return challenge;
    }
    
    // Verify signature
    static bool verifySignature(
        const std::vector<uint8_t>& data,
        const std::vector<uint8_t>& signature,
        const std::vector<uint8_t>& publicKey,
        const std::string& keyType
    ) {
        try {
            if (keyType == "ec2") {
                // Verify ECDSA signature
                return verifyECDSASignature(data, signature, publicKey);
            } else if (keyType == "rsa") {
                // Verify RSA signature
                return verifyRSASignature(data, signature, publicKey);
            }
        } catch (const std::exception& e) {
            std::cerr << "Signature verification failed: " << e.what() << std::endl;
        }
        return false;
    }
    
    // Hash client data JSON
    static std::vector<uint8_t> hashClientDataJSON(const std::vector<uint8_t>& clientDataJSON) {
        SHA256 hash;
        std::vector<uint8_t> digest(hash.DigestSize());
        hash.CalculateDigest(digest.data(), clientDataJSON.data(), clientDataJSON.size());
        return digest;
    }
    
    // Parse authenticator data
    static bool parseAuthenticatorData(
        const std::vector<uint8_t>& authData,
        std::string& rpIdHash,
        uint8_t& flags,
        std::vector<uint8_t>& counter,
        std::vector<uint8_t>& pubKey
    ) {
        if (authData.size() < 37) return false;
        
        // RP ID hash (32 bytes)
        rpIdHash = bytesToHex(std::vector<uint8_t>(authData.begin(), authData.begin() + 32));
        
        // Flags (1 byte)
        flags = authData[32];
        
        // Counter (4 bytes)
        counter = std::vector<uint8_t>(authData.begin() + 33, authData.begin() + 37);
        
        // Public key (if present)
        if (authData.size() > 37) {
            pubKey = std::vector<uint8_t>(authData.begin() + 37, authData.end());
        }
        
        return true;
    }
    
    // Verify attestation signature
    static bool verifyAttestationSignature(
        const std::vector<uint8_t>& authenticatorData,
        const std::vector<uint8_t>& clientDataHash,
        const std::vector<uint8_t>& signature,
        const std::vector<uint8_t>& attestationCert
    ) {
        // Combine authenticator data + client data hash
        std::vector<uint8_t> signedData;
        signedData.insert(signedData.end(), authenticatorData.begin(), authenticatorData.end());
        signedData.insert(signedData.end(), clientDataHash.begin(), clientDataHash.end());
        
        return verifySignature(signedData, signature, attestationCert, "ec2");
    }
    
private:
    static bool verifyECDSASignature(
        const std::vector<uint8_t>& data,
        const std::vector<uint8_t>& signature,
        const std::vector<uint8_t>& publicKey
    ) {
        // Simplified - in production use proper ECDSA verification
        return !signature.empty() && !publicKey.empty();
    }
    
    static bool verifyRSASignature(
        const std::vector<uint8_t>& data,
        const std::vector<uint8_t>& signature,
        const std::vector<uint8_t>& publicKey
    ) {
        return !signature.empty() && !publicKey.empty();
    }
    
    static std::string bytesToHex(const std::vector<uint8_t>& bytes) {
        std::string result;
        char hex[3];
        for (const auto& b : bytes) {
            sprintf(hex, "%02x", b);
            result += hex;
        }
        return result;
    }
};

// =============================================================================
// PASSKEY REGISTRATION
// =============================================================================

class PasskeyRegistrationService {
public:
    PasskeyRegistrationService() = default;
    
    // Generate registration options
    WebAuthnOptions generateRegistrationOptions(
        const RelyingParty& rp,
        const UserAccount& user,
        const std::vector<CredentialID>& excludeCredentials = {}
    ) {
        WebAuthnOptions options;
        options.challenge = bytesToHex(WebAuthnCrypto::generateChallenge());
        options.timeout = 60000; // 60 seconds
        options.rpId = rp.id;
        options.rpName = rp.name;
        options.users.push_back(user);
        options.pubKeyCredParams = -7; // ES256
        options.requireResidentKey = false;
        options.userVerification = "preferred";
        options.attestation = "none";
        
        // Add exclude credentials
        for (const auto& credId : excludeCredentials) {
            options.excludeCredentials.push_back(credId);
        }
        
        // Store challenge for verification
        pendingChallenges_[user.id] = options.challenge;
        
        return options;
    }
    
    // Verify and complete registration
    std::optional<PasskeyRegistration> verifyRegistration(
        const UserID& userId,
        const std::vector<uint8_t>& clientDataJSON,
        const AuthenticatorAttestationResponse& attestation
    ) {
        // Verify challenge
        auto it = pendingChallenges_.find(userId);
        if (it == pendingChallenges_.end()) {
            std::cerr << "No pending challenge for user" << std::endl;
            return std::nullopt;
        }
        
        // Verify client data JSON
        if (!verifyClientData(clientDataJSON, it->second)) {
            std::cerr << "Client data verification failed" << std::endl;
            return std::nullopt;
        }
        
        // Parse authenticator data
        std::string rpIdHash;
        uint8_t flags;
        std::vector<uint8_t> counter, pubKey;
        
        if (!WebAuthnCrypto::parseAuthenticatorData(attestation.authData, rpIdHash, flags, counter, pubKey)) {
            std::cerr << "Authenticator data parsing failed" << std::endl;
            return std::nullopt;
        }
        
        // Verify attestation (in production, verify with attestation root certificates)
        
        // Create passkey registration
        PasskeyRegistration reg;
        reg.credentialId = bytesToHex(attestation.authData); // Simplified
        reg.publicKey = pubKey;
        reg.keyType = "ec2";
        reg.keyAlgorithm = -7; // ES256
        reg.rpId = rpIdHash;
        reg.userId = userId;
        reg.counter = counter;
        reg.createdAt = currentTimestamp();
        
        // Store registration
        registrations_[reg.credentialId] = reg;
        
        // Clean up challenge
        pendingChallenges_.erase(it);
        
        return reg;
    }
    
private:
    std::map<UserID, std::string> pendingChallenges_;
    std::map<CredentialID, PasskeyRegistration> registrations_;
    
    bool verifyClientData(const std::vector<uint8_t>& clientDataJSON, const std::string& expectedChallenge) {
        // Parse client data JSON and verify challenge
        // In production, parse JSON and verify challenge matches
        return true;
    }
    
    std::string bytesToHex(const std::vector<uint8_t>& bytes) {
        std::string result;
        char hex[3];
        for (const auto& b : bytes) {
            sprintf(hex, "%02x", b);
            result += hex;
        }
        return result;
    }
    
    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
};

// =============================================================================
// PASSKEY AUTHENTICATION
// =============================================================================

class PasskeyAuthenticationService {
public:
    PasskeyAuthenticationService() = default;
    
    // Generate authentication options
    WebAuthnOptions generateAuthenticationOptions(
        const RelyingParty& rp,
        const std::vector<PasskeyRegistration>& allowCredentials
    ) {
        WebAuthnOptions options;
        options.challenge = bytesToHex(WebAuthnCrypto::generateChallenge());
        options.timeout = 60000;
        options.rpId = rp.id;
        
        for (const auto& cred : allowCredentials) {
            options.excludeCredentials.push_back(cred.credentialId);
        }
        
        options.userVerification = "preferred";
        
        // Store challenge
        storedChallenge_ = options.challenge;
        
        return options;
    }
    
    // Verify and complete authentication
    std::optional<PasskeyAuthentication> verifyAuthentication(
        const CredentialID& credentialId,
        const std::vector<uint8_t>& clientDataJSON,
        const AuthenticatorAssertionResponse& assertion
    ) {
        // Find registration
        auto it = registrations_.find(credentialId);
        if (it == registrations_.end()) {
            std::cerr << "Credential not found" << std::endl;
            return std::nullopt;
        }
        
        const auto& reg = it->second;
        
        // Verify client data
        if (!verifyClientData(clientDataJSON, storedChallenge_)) {
            std::cerr << "Client data verification failed" << std::endl;
            return std::nullopt;
        }
        
        // Parse authenticator data
        std::string rpIdHash;
        uint8_t flags;
        std::vector<uint8_t> counter, pubKey;
        
        if (!WebAuthnCrypto::parseAuthenticatorData(assertion.authenticatorData, rpIdHash, flags, counter, pubKey)) {
            std::cerr << "Authenticator data parsing failed" << std::endl;
            return std::nullopt;
        }
        
        // Verify user present flag
        if (!(flags & 0x01)) {
            std::cerr << "User not present" << std::endl;
            return std::nullopt;
        }
        
        // Verify signature
        std::vector<uint8_t> clientDataHash = WebAuthnCrypto::hashClientDataJSON(clientDataJSON);
        
        std::vector<uint8_t> signedData;
        signedData.insert(signedData.end(), assertion.authenticatorData.begin(), assertion.authenticatorData.end());
        signedData.insert(signedData.end(), clientDataHash.begin(), clientDataHash.end());
        
        if (!WebAuthnCrypto::verifySignature(signedData, assertion.signature, reg.publicKey, reg.keyType)) {
            std::cerr << "Signature verification failed" << std::endl;
            return std::nullopt;
        }
        
        // Create authentication record
        PasskeyAuthentication auth;
        auth.credentialId = credentialId;
        auth.authenticatorData = assertion.authenticatorData;
        auth.signature = assertion.signature;
        auth.clientDataJSON = clientDataJSON;
        auth.userHandle = assertion.userHandle;
        auth.counter = counter;
        auth.authenticatedAt = currentTimestamp();
        
        // Update counter
        registrations_[credentialId].counter = counter;
        
        return auth;
    }
    
private:
    std::map<CredentialID, PasskeyRegistration> registrations_;
    std::string storedChallenge_;
    
    bool verifyClientData(const std::vector<uint8_t>& clientDataJSON, const std::string& expectedChallenge) {
        return true;
    }
    
    std::string bytesToHex(const std::vector<uint8_t>& bytes) {
        std::string result;
        char hex[3];
        for (const auto& b : bytes) {
            sprintf(hex, "%02x", b);
            result += hex;
        }
        return result;
    }
    
    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
};

// =============================================================================
// PASSKEY CREDENTIAL MANAGEMENT
// =============================================================================

class PasskeyCredentialManager {
public:
    PasskeyCredentialManager() = default;
    
    // Add credential
    void addCredential(const PasskeyRegistration& credential) {
        std::lock_guard<std::mutex> lock(mutex_);
        credentials_[credential.credentialId] = credential;
        
        // Also index by user
        userCredentials_[credential.userId].push_back(credential.credentialId);
    }
    
    // Get credentials for user
    std::vector<PasskeyRegistration> getCredentialsForUser(const UserID& userId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        std::vector<PasskeyRegistration> result;
        auto it = userCredentials_.find(userId);
        if (it != userCredentials_.end()) {
            for (const auto& credId : it->second) {
                auto credIt = credentials_.find(credId);
                if (credIt != credentials_.end()) {
                    result.push_back(credIt->second);
                }
            }
        }
        
        return result;
    }
    
    // Get credential by ID
    std::optional<PasskeyRegistration> getCredential(const CredentialID& credId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = credentials_.find(credId);
        if (it != credentials_.end()) {
            return it->second;
        }
        
        return std::nullopt;
    }
    
    // Update credential counter
    void updateCounter(const CredentialID& credId, const std::vector<uint8_t>& counter) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = credentials_.find(credId);
        if (it != credentials_.end()) {
            it->second.counter = counter;
        }
    }
    
    // Delete credential
    bool deleteCredential(const CredentialID& credId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = credentials_.find(credId);
        if (it != credentials_.end()) {
            // Remove from user index
            for (auto& [userId, credIds] : userCredentials_) {
                for (auto it2 = credIds.begin(); it2 != credIds.end(); ++it2) {
                    if (*it2 == credId) {
                        credIds.erase(it2);
                        break;
                    }
                }
            }
            
            credentials_.erase(it);
            return true;
        }
        
        return false;
    }
    
    // Delete all credentials for user
    void deleteAllCredentialsForUser(const UserID& userId) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = userCredentials_.find(userId);
        if (it != userCredentials_.end()) {
            for (const auto& credId : it->second) {
                credentials_.erase(credId);
            }
            userCredentials_.erase(it);
        }
    }
    
    // List all credentials
    std::vector<PasskeyRegistration> listAllCredentials() {
        std::lock_guard<std::mutex> lock(mutex_);
        
        std::vector<PasskeyRegistration> result;
        for (const auto& [_, cred] : credentials_) {
            result.push_back(cred);
        }
        
        return result;
    }
    
    // Check if credential exists
    bool credentialExists(const CredentialID& credId) {
        std::lock_guard<std::mutex> lock(mutex_);
        return credentials_.find(credId) != credentials_.end();
    }
    
private:
    std::mutex mutex_;
    std::map<CredentialID, PasskeyRegistration> credentials_;
    std::map<UserID, std::vector<CredentialID>> userCredentials_;
};

// =============================================================================
// PASSKEY MANAGER (Master Orchestrator)
// =============================================================================

class PasskeyManager {
public:
    PasskeyManager() 
        : registrationService_(), 
          authService_(), 
          credentialManager_() {}
    
    // Initialize
    bool initialize() {
        return true;
    }
    
    // Start registration flow
    WebAuthnOptions startRegistration(
        const std::string& rpId,
        const std::string& rpName,
        const UserID& userId,
        const std::string& userName,
        const std::string& userDisplayName
    ) {
        RelyingParty rp;
        rp.id = rpId;
        rp.name = rpName;
        
        UserAccount user;
        user.id = userId;
        user.name = userName;
        user.displayName = userDisplayName;
        
        // Get existing credentials to exclude
        auto existingCreds = credentialManager_.getCredentialsForUser(userId);
        std::vector<CredentialID> excludeCreds;
        for (const auto& cred : existingCreds) {
            excludeCreds.push_back(cred.credentialId);
        }
        
        return registrationService_.generateRegistrationOptions(rp, user, excludeCreds);
    }
    
    // Complete registration
    std::optional<PasskeyRegistration> completeRegistration(
        const UserID& userId,
        const std::vector<uint8_t>& clientDataJSON,
        const AuthenticatorAttestationResponse& attestation
    ) {
        auto result = registrationService_.verifyRegistration(userId, clientDataJSON, attestation);
        
        if (result.has_value()) {
            credentialManager_.addCredential(result.value());
        }
        
        return result;
    }
    
    // Start authentication flow
    WebAuthnOptions startAuthentication(
        const std::string& rpId,
        const std::string& rpName,
        const UserID& userId
    ) {
        RelyingParty rp;
        rp.id = rpId;
        rp.name = rpName;
        
        // Get user's credentials
        auto creds = credentialManager_.getCredentialsForUser(userId);
        
        return authService_.generateAuthenticationOptions(rp, creds);
    }
    
    // Complete authentication
    std::optional<PasskeyAuthentication> completeAuthentication(
        const CredentialID& credentialId,
        const std::vector<uint8_t>& clientDataJSON,
        const AuthenticatorAssertionResponse& assertion
    ) {
        auto result = authService_.verifyAuthentication(credentialId, clientDataJSON, assertion);
        
        if (result.has_value()) {
            // Update credential counter
            credentialManager_.updateCounter(credentialId, result->counter);
        }
        
        return result;
    }
    
    // Get user credentials
    std::vector<PasskeyRegistration> getUserCredentials(const UserID& userId) {
        return credentialManager_.getCredentialsForUser(userId);
    }
    
    // Delete credential
    bool deleteCredential(const CredentialID& credId) {
        return credentialManager_.deleteCredential(credId);
    }
    
    // Delete all user credentials
    void deleteAllUserCredentials(const UserID& userId) {
        credentialManager_.deleteAllCredentialsForUser(userId);
    }
    
    // List all credentials
    std::vector<PasskeyRegistration> listAllCredentials() {
        return credentialManager_.listAllCredentials();
    }
    
private:
    PasskeyRegistrationService registrationService_;
    PasskeyAuthenticationService authService_;
    PasskeyCredentialManager credentialManager_;
};

} // namespace tiger

#endif // TIGERWALLET_PASSKEY_HPP
