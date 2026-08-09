/**
 * TigerWallet Desktop - Keychain Manager Implementation
 * Cross-platform secure storage using file system with encryption
 */

#include "services/keychain_manager.h"
#include <iostream>
#include <fstream>
#include <sstream>
#include <random>
#include <openssl/evp.h>
#include <openssl/rand.h>
#include <openssl/sha.h>
#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>

namespace tiger {
namespace wallet {

// ============================================================================
// Static Instance
// ============================================================================

std::shared_ptr<KeychainManager> KeychainManager::instance_ = nullptr;

// ============================================================================
// Constructor/Destructor
// ============================================================================

KeychainManager::KeychainManager() : initialized_(false) {
    // Set default storage path
    const char* home = getenv("HOME");
    if (home) {
        masterKeyPath_ = std::string(home) + "/.tigerwallet";
    } else {
        masterKeyPath_ = "/tmp/tigerwallet";
    }
}

KeychainManager::~KeychainManager() {
    shutdown();
}

// ============================================================================
// Singleton
// ============================================================================

std::shared_ptr<KeychainManager> KeychainManager::getInstance() {
    if (!instance_) {
        instance_ = std::make_shared<KeychainManager>();
    }
    return instance_;
}

// ============================================================================
// Initialization
// ============================================================================

void KeychainManager::initialize() {
    if (initialized_) return;
    
    // Create storage directory if it doesn't exist
    mkdir(masterKeyPath_.c_str(), 0700);
    
    initialized_ = true;
    std::cout << "[KeychainManager] Initialized at: " << masterKeyPath_ << std::endl;
}

void KeychainManager::shutdown() {
    clearSession();
    initialized_ = false;
}

// ============================================================================
// Secure Storage
// ============================================================================

bool KeychainManager::save(const std::string& key, const std::vector<uint8_t>& data) {
    return saveToStorage(key, data);
}

std::optional<std::vector<uint8_t>> KeychainManager::load(const std::string& key) {
    return loadFromStorage(key);
}

bool KeychainManager::remove(const std::string& key) {
    return removeFromStorage(key);
}

bool KeychainManager::exists(const std::string& key) {
    std::string path = masterKeyPath_ + "/" + key;
    struct stat buffer;
    return (stat(path.c_str(), &buffer) == 0);
}

// ============================================================================
// Wallet-specific Storage
// ============================================================================

bool KeychainManager::saveWalletSeed(const std::string& walletId, const std::string& mnemonic, const std::string& password) {
    // Encrypt the mnemonic with AES-256-GCM using the user's password before
    // storing. Never store seeds in plaintext.
    std::vector<uint8_t> data(mnemonic.begin(), mnemonic.end());
    auto encrypted = encrypt(data, password);
    return save("wallet_seed_" + walletId, encrypted);
}

std::optional<std::string> KeychainManager::loadWalletSeed(const std::string& walletId, const std::string& password) {
    auto data = load("wallet_seed_" + walletId);
    if (!data || data->empty()) return std::nullopt;
    auto decrypted = decrypt(*data, password);
    return std::string(decrypted.begin(), decrypted.end());
}

bool KeychainManager::savePrivateKey(const std::string& walletId, const std::string& privateKey, const std::string& password) {
    // Encrypt private keys with AES-256-GCM before storing.
    std::vector<uint8_t> data(privateKey.begin(), privateKey.end());
    auto encrypted = encrypt(data, password);
    return save("wallet_key_" + walletId, encrypted);
}

std::optional<std::string> KeychainManager::loadPrivateKey(const std::string& walletId, const std::string& password) {
    auto data = load("wallet_key_" + walletId);
    if (!data || data->empty()) return std::nullopt;
    auto decrypted = decrypt(*data, password);
    return std::string(decrypted.begin(), decrypted.end());
}

bool KeychainManager::removePrivateKey(const std::string& walletId) {
    return remove("wallet_key_" + walletId);
}

// ============================================================================
// Session Management
// ============================================================================

void KeychainManager::setSessionToken(const std::string& token) {
    sessionToken_ = token;
    
    // Also save to file for persistence
    std::vector<uint8_t> data(token.begin(), token.end());
    save("session_token", data);
}

std::optional<std::string> KeychainManager::getSessionToken() {
    if (!sessionToken_.empty()) {
        return sessionToken_;
    }
    
    // Try loading from file
    auto data = load("session_token");
    if (data && !data->empty()) {
        sessionToken_ = std::string(data->begin(), data->end());
        return sessionToken_;
    }
    
    return std::nullopt;
}

void KeychainManager::clearSession() {
    sessionToken_.clear();
    remove("session_token");
}

// ============================================================================
// Encryption (AES-256-GCM, authenticated — replaces insecure CBC)
// ============================================================================

std::vector<uint8_t> KeychainManager::encrypt(const std::vector<uint8_t>& data, const std::string& password) {
    std::vector<uint8_t> key(32);
    std::vector<uint8_t> iv(12);
    std::vector<uint8_t> salt(16);
    RAND_bytes(iv.data(), (int)iv.size());
    RAND_bytes(salt.data(), (int)salt.size());

    PKCS5_PBKDF2_HMAC(password.c_str(), password.length(),
                      salt.data(), (int)salt.size(), 100000,
                      EVP_sha256(), 32, key.data());

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    EVP_EncryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, key.data(), iv.data());

    std::vector<uint8_t> ciphertext(data.size() + 16);
    int outLen1 = 0;
    EVP_EncryptUpdate(ctx, ciphertext.data(), &outLen1, data.data(), (int)data.size());
    int outLen2 = 0;
    EVP_EncryptFinal_ex(ctx, ciphertext.data() + outLen1, &outLen2);
    ciphertext.resize(outLen1 + outLen2);

    std::vector<uint8_t> tag(16);
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, 16, tag.data());
    EVP_CIPHER_CTX_free(ctx);

    std::vector<uint8_t> result;
    result.insert(result.end(), salt.begin(), salt.end());
    result.insert(result.end(), iv.begin(), iv.end());
    result.insert(result.end(), tag.begin(), tag.end());
    result.insert(result.end(), ciphertext.begin(), ciphertext.end());
    return result;
}

std::vector<uint8_t> KeychainManager::decrypt(const std::vector<uint8_t>& encryptedData, const std::string& password) {
    if (encryptedData.size() < 44) {
        throw KeychainException(KeychainException::ErrorCode::DecryptionError, "Invalid encrypted data");
    }

    std::vector<uint8_t> salt(encryptedData.begin(), encryptedData.begin() + 16);
    std::vector<uint8_t> iv(encryptedData.begin() + 16, encryptedData.begin() + 28);
    std::vector<uint8_t> tag(encryptedData.begin() + 28, encryptedData.begin() + 44);
    std::vector<uint8_t> ciphertext(encryptedData.begin() + 44, encryptedData.end());

    std::vector<uint8_t> key(32);
    PKCS5_PBKDF2_HMAC(password.c_str(), password.length(),
                      salt.data(), (int)salt.size(), 100000,
                      EVP_sha256(), 32, key.data());

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    EVP_DecryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, key.data(), iv.data());

    std::vector<uint8_t> decrypted(ciphertext.size());
    int outLen1 = 0;
    EVP_DecryptUpdate(ctx, decrypted.data(), &outLen1, ciphertext.data(), (int)ciphertext.size());

    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, 16, tag.data());
    int outLen2 = 0;
    int ret = EVP_DecryptFinal_ex(ctx, decrypted.data() + outLen1, &outLen2);
    EVP_CIPHER_CTX_free(ctx);

    if (ret <= 0) {
        throw KeychainException(KeychainException::ErrorCode::DecryptionError,
            "Decryption failed: wrong password or corrupted data");
    }

    decrypted.resize(outLen1 + outLen2);
    return decrypted;
}

// ============================================================================
// Private: Storage Operations
// ============================================================================

bool KeychainManager::saveToStorage(const std::string& key, const std::vector<uint8_t>& data) {
    std::string path = masterKeyPath_ + "/" + key;
    
    std::ofstream file(path, std::ios::binary);
    if (!file.is_open()) {
        return false;
    }
    
    file.write(reinterpret_cast<const char*>(data.data()), data.size());
    file.close();
    
    // Set permissions to owner only
    chmod(path.c_str(), 0600);
    
    return true;
}

std::optional<std::vector<uint8_t>> KeychainManager::loadFromStorage(const std::string& key) {
    std::string path = masterKeyPath_ + "/" + key;
    
    std::ifstream file(path, std::ios::binary | std::ios::ate);
    if (!file.is_open()) {
        return std::nullopt;
    }
    
    std::streamsize size = file.tellg();
    file.seekg(0, std::ios::beg);
    
    std::vector<uint8_t> data(size);
    if (!file.read(reinterpret_cast<char*>(data.data()), size)) {
        return std::nullopt;
    }
    
    return data;
}

bool KeychainManager::removeFromStorage(const std::string& key) {
    std::string path = masterKeyPath_ + "/" + key;
    return remove(path.c_str()) == 0;
}

// ============================================================================
// Private: Master Key Derivation
// ============================================================================

std::vector<uint8_t> KeychainManager::deriveMasterKey(const std::string& password) {
    std::vector<uint8_t> key(32);
    static std::vector<uint8_t> masterSalt;
    if (masterSalt.empty()) {
        masterSalt.resize(16);
        RAND_bytes(masterSalt.data(), 16);
    }
    PKCS5_PBKDF2_HMAC(password.c_str(), password.length(),
                      masterSalt.data(), (int)masterSalt.size(),
                      100000, EVP_sha256(), 32, key.data());
    return key;
}

// ============================================================================
// Exception
// ============================================================================

KeychainException::KeychainException(ErrorCode code, const std::string& message)
    : std::runtime_error(message), code_(code) {}

KeychainException::ErrorCode KeychainException::getErrorCode() const {
    return code_;
}

} // namespace wallet
} // namespace tiger
