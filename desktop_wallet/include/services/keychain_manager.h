/**
 * TigerWallet Desktop - Keychain Manager
 * Secure storage for sensitive data (private keys, mnemonics)
 */

#ifndef TIGER_WALLET_KEYCHAIN_MANAGER_H
#define TIGER_WALLET_KEYCHAIN_MANAGER_H

#include <string>
#include <vector>
#include <optional>
#include <memory>
#include <stdexcept>

namespace tiger {
namespace wallet {

// ============================================================================
// Keychain Manager
// ============================================================================

class KeychainManager {
public:
    static std::shared_ptr<KeychainManager> getInstance();

    // Initialization
    void initialize();
    void shutdown();

    // Secure Storage
    bool save(const std::string& key, const std::vector<uint8_t>& data);
    std::optional<std::vector<uint8_t>> load(const std::string& key);
    bool remove(const std::string& key);
    bool exists(const std::string& key);

    // Wallet-specific storage (encrypted with AES-256-GCM using user password)
    bool saveWalletSeed(const std::string& walletId, const std::string& mnemonic, const std::string& password);
    std::optional<std::string> loadWalletSeed(const std::string& walletId, const std::string& password);
    bool removeWalletSeed(const std::string& walletId);

    bool savePrivateKey(const std::string& walletId, const std::string& privateKey, const std::string& password);
    std::optional<std::string> loadPrivateKey(const std::string& walletId, const std::string& password);
    bool removePrivateKey(const std::string& walletId);

    // Session Management
    void setSessionToken(const std::string& token);
    std::optional<std::string> getSessionToken();
    void clearSession();

    // Encryption
    std::vector<uint8_t> encrypt(const std::vector<uint8_t>& data, const std::string& password);
    std::vector<uint8_t> decrypt(const std::vector<uint8_t>& encryptedData, const std::string& password);

private:
    KeychainManager(const KeychainManager&) = delete;
    KeychainManager& operator=(const KeychainManager&) = delete;

public:
    KeychainManager();
    ~KeychainManager();

    // Platform-specific storage
    bool saveToStorage(const std::string& key, const std::vector<uint8_t>& data);
    std::optional<std::vector<uint8_t>> loadFromStorage(const std::string& key);
    bool removeFromStorage(const std::string& key);

    // Master key derivation
    std::vector<uint8_t> deriveMasterKey(const std::string& password);

    // Members
    static std::shared_ptr<KeychainManager> instance_;
    bool initialized_;
    std::string sessionToken_;
    std::string masterKeyPath_;
};

// ============================================================================
// Exception
// ============================================================================

class KeychainException : public std::runtime_error {
public:
    enum class ErrorCode {
        StorageError,
        EncryptionError,
        DecryptionError,
        NotFound,
        AccessDenied,
        Unknown
    };

    KeychainException(ErrorCode code, const std::string& message);
    ErrorCode getErrorCode() const;

private:
    ErrorCode code_;
};

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_KEYCHAIN_MANAGER_H
