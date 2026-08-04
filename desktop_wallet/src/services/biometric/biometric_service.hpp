/**
 * TigerWallet Desktop Biometric Service - C++ Implementation
 * Production-ready biometric authentication using Windows Hello / macOS Touch ID / Linux
 * 
 * Features:
 * - Fingerprint/Face authentication
 * - Platform-specific implementations
 * - Secure key storage
 * - Multi-factor authentication
 */

#ifndef BIOMETRIC_SERVICE_HPP
#define BIOMETRIC_SERVICE_HPP

#include <string>
#include <vector>
#include <memory>
#include <functional>
#include <mutex>
#include <atomic>
#include <optional>
#include <array>

// Platform-specific includes
#ifdef _WIN32
#include <windows.h>
#include <winbio.h>
#pragma comment(lib, "winbio.lib")
#elif __APPLE__
#include <Security/Security.h>
#include <LocalAuthentication/LocalAuthentication.h>
#elif __linux__
#include <pthread.h>
#include <unistd.h>
#endif

namespace tigerwallet {
namespace security {

// ============================================================================
// CONSTANTS
// ============================================================================

constexpr size_t MAX_BIOMETRIC_RETRY = 3;
constexpr size_t BIOMETRIC_TIMEOUT_SECONDS = 30;

// Biometric types
enum class BiometricType : uint8_t {
    NONE = 0,
    FINGERPRINT = 1,
    FACE = 2,
    IRIS = 3,
    VOICE = 4,
    MULTI = 5
};

// Authentication result
enum class BiometricResult : uint8_t {
    SUCCESS = 0,
    NOT_AVAILABLE = 1,
    NOT_ENROLLED = 2,
    LOCKOUT = 3,
    CANCELLED = 4,
    FAILED = 5,
    TIMEOUT = 6,
    ERROR = 7
};

// ============================================================================
// DATA STRUCTURES
// ============================================================================

struct BiometricCredential {
    std::string credential_id;
    std::string user_id;
    BiometricType type;
    std::string public_key;
    std::string encrypted_private_key;
    bool is_active;
    uint64_t created_at;
    uint64_t last_used_at;
};

struct BiometricAuthContext {
    std::string session_id;
    std::string user_id;
    BiometricType type;
    bool is_authenticated;
    uint64_t authenticated_at;
    uint64_t expires_at;
};

struct BiometricEnrollment {
    std::string user_id;
    std::vector<BiometricType> available_types;
    std::vector<BiometricType> enrolled_types;
    bool is_enrolled;
};

// ============================================================================
// BIOMETRIC SERVICE INTERFACE
// ============================================================================

class IBiometricService {
public:
    virtual ~IBiometricService() = default;
    
    virtual bool initialize() = 0;
    virtual void shutdown() = 0;
    
    virtual BiometricResult isAvailable() = 0;
    virtual BiometricEnrollment getEnrollment(const std::string& user_id) = 0;
    
    virtual BiometricResult enroll(
        const std::string& user_id,
        BiometricType type,
        const std::string& credential_data
    ) = 0;
    
    virtual BiometricResult authenticate(
        const std::string& credential_id,
        BiometricAuthContext& context
    ) = 0;
    
    virtual BiometricResult verify(
        const std::string& session_id,
        const std::vector<uint8_t>& signature
    ) = 0;
    
    virtual bool remove(const std::string& credential_id) = 0;
};

// ============================================================================
// WINDOWS HELLO IMPLEMENTATION
// ============================================================================

#ifdef _WIN32

class WindowsHelloBiometricService : public IBiometricService {
public:
    WindowsHelloBiometricService();
    ~WindowsHelloBiometricService() override;
    
    bool initialize() override;
    void shutdown() override;
    
    BiometricResult isAvailable() override;
    BiometricEnrollment getEnrollment(const std::string& user_id) override;
    
    BiometricResult enroll(
        const std::string& user_id,
        BiometricType type,
        const std::string& credential_data
    ) override;
    
    BiometricResult authenticate(
        const std::string& credential_id,
        BiometricAuthContext& context
    ) override;
    
    BiometricResult verify(
        const std::string& session_id,
        const std::vector<uint8_t>& signature
    ) override;
    
    bool remove(const std::string& credential_id) override;

private:
    HANDLE session_handle_;
    bool initialized_;
    std::mutex mutex_;
    std::vector<BiometricCredential> credentials_;
    
    BiometricResult enrollFingerprint(const std::string& user_id);
    BiometricResult enrollFace(const std::string& user_id);
    BiometricResult authenticateInternal(BiometricType type, BiometricAuthContext& context);
};

#endif

// ============================================================================
// MACOS TOUCH ID IMPLEMENTATION
// ============================================================================

#ifdef __APPLE__

class TouchIDBiometricService : public IBiometricService {
public:
    TouchIDBiometricService();
    ~TouchIDBiometricService() override;
    
    bool initialize() override;
    void shutdown() override;
    
    BiometricResult isAvailable() override;
    BiometricEnrollment getEnrollment(const std::string& user_id) override;
    
    BiometricResult enroll(
        const std::string& user_id,
        BiometricType type,
        const std::string& credential_data
    ) override;
    
    BiometricResult authenticate(
        const std::string& credential_id,
        BiometricAuthContext& context
    ) override;
    
    BiometricResult verify(
        const std::string& session_id,
        const std::vector<uint8_t>& signature
    ) override;
    
    bool remove(const std::string& credential_id) override;

private:
    bool initialized_;
    std::mutex mutex_;
    std::vector<BiometricCredential> credentials_;
    
    bool evaluatePolicy(void* policy, std::string& error);
};

#endif

// ============================================================================
// LINUX FINGERPRINT IMPLEMENTATION
// ============================================================================

#ifdef __linux__

class LinuxFingerprintService : public IBiometricService {
public:
    LinuxFingerprintService();
    ~LinuxFingerprintService() override;
    
    bool initialize() override;
    void shutdown() override;
    
    BiometricResult isAvailable() override;
    BiometricEnrollment getEnrollment(const std::string& user_id) override;
    
    BiometricResult enroll(
        const std::string& user_id,
        BiometricType type,
        const std::string& credential_data
    ) override;
    
    BiometricResult authenticate(
        const std::string& credential_id,
        BiometricAuthContext& context
    ) override;
    
    BiometricResult verify(
        const std::string& session_id,
        const std::vector<uint8_t>& signature
    ) override;
    
    bool remove(const std::string& credential_id) override;

private:
    bool initialized_;
    std::mutex mutex_;
    int device_fd_;
    std::vector<BiometricCredential> credentials_;
    
    bool openDevice();
    bool closeDevice();
    BiometricResult scanAndEnroll(const std::string& user_id);
    BiometricResult scanAndVerify(BiometricAuthContext& context);
};

#endif

// ============================================================================
// BIOMETRIC MANAGER
// ============================================================================

class BiometricManager {
public:
    static BiometricManager& getInstance();
    
    bool initialize();
    void shutdown();
    
    std::shared_ptr<IBiometricService> getService();
    
    BiometricResult authenticate(
        const std::string& user_id,
        BiometricAuthContext& context
    );
    
    BiometricResult enrollBiometric(
        const std::string& user_id,
        BiometricType type
    );
    
    bool isPlatformSupported() const;
    BiometricType getSupportedType() const;
    
    // Credential management
    bool saveCredential(const BiometricCredential& credential);
    std::optional<BiometricCredential> getCredential(const std::string& credential_id);
    std::vector<BiometricCredential> getUserCredentials(const std::string& user_id);
    bool deleteCredential(const std::string& credential_id);

private:
    BiometricManager();
    ~BiometricManager();
    
    BiometricManager(const BiometricManager&) = delete;
    BiometricManager& operator=(const BiometricManager&) = delete;
    
    std::shared_ptr<IBiometricService> service_;
    std::mutex mutex_;
    bool initialized_;
    BiometricType supported_type_;
    
    // Platform detection
    void detectPlatform();
};

// ============================================================================
// KEYCHAIN/STORAGE
// ============================================================================

class SecureKeyStorage {
public:
    static SecureKeyStorage& getInstance();
    
    bool initialize(const std::string& app_name);
    void shutdown();
    
    bool storeKey(
        const std::string& key_id,
        const std::vector<uint8_t>& key_data,
        const std::string& user_id
    );
    
    std::optional<std::vector<uint8_t>> retrieveKey(
        const std::string& key_id,
        const std::string& user_id
    );
    
    bool deleteKey(const std::string& key_id, const std::string& user_id);
    
    bool exists(const std::string& key_id, const std::string& user_id) const;

private:
    SecureKeyStorage() = default;
    ~SecureKeyStorage() = default;
    
    SecureKeyStorage(const SecureKeyStorage&) = delete;
    SecureKeyStorage& operator=(const SecureKeyStorage&) = delete;
    
    std::string app_name_;
    std::mutex mutex_;
};

// ============================================================================
// INLINE HELPERS
// ============================================================================

inline bool isSuccess(BiometricResult result) {
    return result == BiometricResult::SUCCESS;
}

inline bool isRecoverable(BiometricResult result) {
    return result == BiometricResult::LOCKOUT || 
           result == BiometricResult::TIMEOUT ||
           result == BiometricResult::CANCELLED;
}

} // namespace security
} // namespace tigerwallet

#endif // BIOMETRIC_SERVICE_HPP
