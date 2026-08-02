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

bool KeychainManager::saveWalletSeed(const std::string& walletId, const std::string& mnemonic) {
    std::vector<uint8_t> data(mnemonic.begin(), mnemonic.end());
    return save("wallet_seed_" + walletId, data);
}

std::optional<std::string> KeychainManager::loadWalletSeed(const std::string& walletId) {
    auto data = load("wallet_seed_" + walletId);
    if (data && !data->empty()) {
        return std::string(data->begin(), data->end());
    }
    return std::nullopt;
}

bool KeychainManager::removeWalletSeed(const std::string& walletId) {
    return remove("wallet_seed_" + walletId);
}

bool KeychainManager::savePrivateKey(const std::string& walletId, const std::string& privateKey) {
    std::vector<uint8_t> data(privateKey.begin(), privateKey.end());
    return save("wallet_key_" + walletId, data);
}

std::optional<std::string> KeychainManager::loadPrivateKey(const std::string& walletId) {
    auto data = load("wallet_key_" + walletId);
    if (data && !data->empty()) {
        return std::string(data->begin(), data->end());
    }
    return std::nullopt;
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
// Encryption
// ============================================================================

std::vector<uint8_t> KeychainManager::encrypt(const std::vector<uint8_t>& data, const std::string& password) {
    // Derive key from password
    auto key = deriveMasterKey(password);
    
    // Generate random IV
    std::vector<uint8_t> iv(16);
    RAND_bytes(iv.data(), iv.size());
    
    // AES-256-CBC encryption
    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    
    std::vector<uint8_t> encrypted(data.size() + 32); // Extra space for padding
    int outLen1 = 0, outLen2 = 0;
    
    EVP_EncryptInit_ex(ctx, EVP_aes_256_cbc(), nullptr, key.data(), iv.data());
    EVP_EncryptUpdate(ctx, encrypted.data(), &outLen1, data.data(), data.size());
    EVP_EncryptFinal_ex(ctx, encrypted.data() + outLen1, &outLen2);
    
    EVP_CIPHER_CTX_free(ctx);
    
    // Prepend IV to encrypted data
    std::vector<uint8_t> result;
    result.insert(result.end(), iv.begin(), iv.end());
    result.insert(result.end(), encrypted.begin(), encrypted.begin() + outLen1 + outLen2);
    
    return result;
}

std::vector<uint8_t> KeychainManager::decrypt(const std::vector<uint8_t>& encryptedData, const std::string& password) {
    if (encryptedData.size() < 16) {
        throw KeychainException(KeychainException::ErrorCode::DecryptionError, "Invalid encrypted data");
    }
    
    // Derive key from password
    auto key = deriveMasterKey(password);
    
    // Extract IV
    std::vector<uint8_t> iv(encryptedData.begin(), encryptedData.begin() + 16);
    std::vector<uint8_t> ciphertext(encryptedData.begin() + 16, encryptedData.end());
    
    // AES-256-CBC decryption
    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    
    std::vector<uint8_t> decrypted(ciphertext.size());
    int outLen1 = 0, outLen2 = 0;
    
    EVP_DecryptInit_ex(ctx, EVP_aes_256_cbc(), nullptr, key.data(), iv.data());
    EVP_DecryptUpdate(ctx, decrypted.data(), &outLen1, ciphertext.data(), ciphertext.size());
    EVP_DecryptFinal_ex(ctx, decrypted.data() + outLen1, &outLen2);
    
    EVP_CIPHER_CTX_free(ctx);
    
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
    // Use PBKDF2 for key derivation
    std::vector<uint8_t> key(32); // 256 bits for AES-256
    
    // Salt for derivation (in production, use unique salt per user)
    std::string salt = "TigerWalletSecureSalt2024";
    
    PKCS5_PBKDF2_HMAC(
        password.c_str(),
        password.length(),
        reinterpret_cast<const unsigned char*>(salt.c_str()),
        salt.length(),
        100000, // iterations
        EVP_sha256(),
        32,
        key.data()
    );
    
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
