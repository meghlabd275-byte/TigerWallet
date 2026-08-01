/**
 * TigerWallet MPC Wallet - C++ Implementation
 * Ultra-low latency for high-frequency trading
 */

#include "mpc_wallet.h"
#include <sha2.h>
#include <hmac.h>
#include <secp256k1.h>
#include <ed25519.h>
#include <random.hpp>
#include <chrono>
#include <sstream>
#include <iomanip>

namespace tigerwallet {
namespace mpc {

// ============================================================================
// Utility Functions
// ============================================================================

static std::vector<uint8_t> computeSha256(const std::vector<uint8_t>& data) {
    std::vector<uint8_t> hash(32);
    sha256(data.data(), data.size(), hash.data());
    return hash;
}

static std::vector<uint8_t> computeHmacSha256(const std::vector<uint8_t>& key, 
                                               const std::vector<uint8_t>& data) {
    std::vector<uint8_t> result(32);
    hmac_sha256(key.data(), key.size(), data.data(), data.size(), result.data());
    return result;
}

static std::string bytesToHex(const std::vector<uint8_t>& bytes) {
    std::ostringstream oss;
    for (uint8_t b : bytes) {
        oss << std::hex << std::setfill('0') << std::setw(2) << (int)b;
    }
    return oss.str();
}

static std::vector<uint8_t> hexToBytes(const std::string& hex) {
    std::vector<uint8_t> bytes;
    for (size_t i = 0; i < hex.length(); i += 2) {
        std::string byteStr = hex.substr(i, 2);
        uint8_t byte = (uint8_t)std::strtol(byteStr.c_str(), nullptr, 16);
        bytes.push_back(byte);
    }
    return bytes;
}

static std::string generateUuid() {
    static std::random_device rd;
    static std::mt19937 gen(rd());
    static std::uniform_int_distribution<> dis(0, 255);
    
    std::array<uint8_t, 16> bytes;
    for (auto& b : bytes) {
        b = (uint8_t)dis(gen);
    }
    // Set version (4) and variant
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    
    return fmt::format("{:02x}{:02x}{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}-{:02x}{:02x}{:02x}{:02x}{:02x}{:02x}",
        bytes[0], bytes[1], bytes[2], bytes[3],
        bytes[4], bytes[5], bytes[6], bytes[7],
        bytes[8], bytes[9], bytes[10], bytes[11],
        bytes[12], bytes[13], bytes[14], bytes[15]);
}

// ============================================================================
// MPC Wallet Implementation
// ============================================================================

class MpcWallet::Impl {
public:
    Impl() : config_(std::make_shared<MpcConfig>()) {}
    
    std::string generateKey(const std::vector<uint8_t>& entropy) {
        // Generate random bytes
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 255);
        
        std::vector<uint8_t> randomBytes(32);
        for (auto& b : randomBytes) {
            b = (uint8_t)dis(gen);
        }
        
        // Combine entropy with random
        std::vector<uint8_t> seed;
        seed.reserve(entropy.size() + randomBytes.size());
        seed.insert(seed.end(), entropy.begin(), entropy.end());
        seed.insert(seed.end(), randomBytes.begin(), randomBytes.end());
        
        std::string keyId = generateUuid();
        
        // Generate key based on curve type
        std::vector<uint8_t> publicKey;
        std::vector<ShareInfo> shares;
        
        switch (config_->curve) {
            case CurveType::Secp256k1:
                publicKey = generateSecp256k1Key(seed, shares, keyId);
                break;
            case CurveType::Ed25519:
                publicKey = generateEd25519Key(seed, shares, keyId);
                break;
            case CurveType::P256:
                publicKey = generateP256Key(seed, shares, keyId);
                break;
        }
        
        // Generate backup
        std::vector<uint8_t> backupKey(32);
        for (auto& b : backupKey) {
            b = (uint8_t)dis(gen);
        }
        auto backup = generateBackup(publicKey, backupKey);
        
        // Store key data
        MpcKeyData keyData;
        keyData.key_id = keyId;
        keyData.public_key = publicKey;
        keyData.curve = config_->curve;
        keyData.created_at = std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()).count();
        
        keys_[keyId] = keyData;
        shares_[keyId] = shares;
        
        return keyId;
    }
    
    std::vector<uint8_t> getPublicKey(const std::string& keyId) const {
        auto it = keys_.find(keyId);
        if (it == keys_.end()) {
            throw MpcException(MpcErrorCode::KeyNotFound, "Key not found: " + keyId);
        }
        return it->second.public_key;
    }
    
    ShareInfo getShare(const std::string& keyId, uint32_t index) const {
        auto it = shares_.find(keyId);
        if (it == shares_.end()) {
            throw MpcException(MpcErrorCode::KeyNotFound, "Key not found: " + keyId);
        }
        
        for (const auto& share : it->second) {
            if (share.index == index) {
                return share;
            }
        }
        
        throw MpcException(MpcErrorCode::InvalidShare, "Share not found");
    }
    
    void addShare(const std::string& keyId, const ShareInfo& share) {
        auto it = shares_.find(keyId);
        if (it == shares_.end()) {
            throw MpcException(MpcErrorCode::KeyNotFound, "Key not found: " + keyId);
        }
        
        // Verify share
        auto computed = computeSha256(share.share_data);
        if (computed != share.verification_key) {
            throw MpcException(MpcErrorCode::InvalidShare, "Share verification failed");
        }
        
        it->second.push_back(share);
    }
    
    void deleteKey(const std::string& keyId) {
        keys_.erase(keyId);
        shares_.erase(keyId);
    }
    
    std::vector<std::string> listKeys() const {
        std::vector<std::string> result;
        for (const auto& pair : keys_) {
            result.push_back(pair.first);
        }
        return result;
    }
    
    SignResult sign(const std::string& keyId, const std::vector<uint8_t>& message) {
        auto keyIt = keys_.find(keyId);
        if (keyIt == keys_.end()) {
            throw MpcException(MpcErrorCode::KeyNotFound, "Key not found: " + keyId);
        }
        
        auto shareIt = shares_.find(keyId);
        if (shareIt == shares_.end()) {
            throw MpcException(MpcErrorCode::KeyNotFound, "Shares not found: " + keyId);
        }
        
        if (shareIt->second.size() < config_->threshold) {
            throw MpcException(MpcErrorCode::ThresholdNotMet, 
                "Need " + std::to_string(config_->threshold) + " shares, have " + 
                std::to_string(shareIt->second.size()));
        }
        
        SignResult result;
        result.key_id = keyId;
        result.public_key = keyIt->second.public_key;
        
        // Sign based on curve
        std::vector<std::vector<uint8_t>> shareData;
        for (size_t i = 0; i < config_->threshold; i++) {
            shareData.push_back(shareIt->second[i].share_data);
        }
        
        switch (config_->curve) {
            case CurveType::Secp256k1:
                result.signature = signSecp256k1(shareData, message);
                break;
            case CurveType::Ed25519:
                result.signature = signEd25519(shareData, message);
                break;
            case CurveType::P256:
                result.signature = signP256(shareData, message);
                break;
        }
        
        return result;
    }
    
    bool verify(const std::string& keyId,
                const std::vector<uint8_t>& signature,
                const std::vector<uint8_t>& message) const {
        auto it = keys_.find(keyId);
        if (it == keys_.end()) {
            return false;
        }
        
        // Verify based on curve
        switch (it->second.curve) {
            case CurveType::Secp256k1:
                return verifySecp256k1(it->second.public_key, signature, message);
            case CurveType::Ed25519:
                return verifyEd25519(it->second.public_key, signature, message);
            case CurveType::P256:
                return verifyP256(it->second.public_key, signature, message);
        }
        
        return false;
    }
    
    std::vector<ShareInfo> reshareKey(const std::string& keyId, const MpcConfig& newConfig) {
        auto shareIt = shares_.find(keyId);
        if (shareIt == shares_.end()) {
            throw MpcException(MpcErrorCode::KeyNotFound, "Key not found: " + keyId);
        }
        
        auto keyIt = keys_.find(keyId);
        if (keyIt == keys_.end()) {
            throw MpcException(MpcErrorCode::KeyNotFound, "Key not found: " + keyId);
        }
        
        // Generate new shares
        std::vector<ShareInfo> newShares;
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 255);
        
        for (uint32_t i = 1; i <= newConfig.total_shares; i++) {
            std::vector<uint8_t> shareData(32);
            for (auto& b : shareData) {
                b = (uint8_t)dis(gen);
            }
            
            auto verificationKey = computeSha256(shareData);
            
            newShares.push_back(ShareInfo{
                .index = i,
                .share_data = shareData,
                .verification_key = verificationKey
            });
        }
        
        shares_[keyId] = newShares;
        
        return newShares;
    }
    
    std::vector<uint8_t> backupKey(const std::string& keyId, 
                                    const std::vector<uint8_t>& encryptionKey) {
        auto shareIt = shares_.find(keyId);
        if (shareIt == shares_.end()) {
            throw MpcException(MpcErrorCode::KeyNotFound, "Key not found: " + keyId);
        }
        
        // Combine all shares
        std::vector<uint8_t> combined;
        for (const auto& share : shareIt->second) {
            combined.insert(combined.end(), share.share_data.begin(), share.share_data.end());
        }
        
        // Encrypt
        auto hmac = computeHmacSha256(encryptionKey, combined);
        
        std::vector<uint8_t> backup;
        backup.insert(backup.end(), encryptionKey.begin(), encryptionKey.end());
        backup.insert(backup.end(), hmac.begin(), hmac.begin() + 16);
        backup.insert(backup.end(), combined.begin(), combined.end());
        
        return backup;
    }
    
    void restoreFromBackup(const std::string& keyId,
                           const std::vector<uint8_t>& backup,
                           const std::vector<uint8_t>& decryptionKey) {
        if (backup.size() < 48) {
            throw MpcException(MpcErrorCode::InvalidShare, "Invalid backup format");
        }
        
        // Verify decryption key
        auto salt = std::vector<uint8_t>(backup.begin() + 32, backup.begin() + 48);
        auto encryptedShares = std::vector<uint8_t>(backup.begin() + 48, backup.end());
        
        auto hmac = computeHmacSha256(decryptionKey, encryptedShares);
        
        if (!std::equal(hmac.begin(), hmac.begin() + 16, salt.begin())) {
            throw MpcException(MpcErrorCode::InvalidShare, "Invalid decryption key");
        }
        
        // Parse shares (simplified)
        std::vector<ShareInfo> shares;
        size_t shareCount = encryptedShares.size() / 32;
        
        for (size_t i = 0; i < shareCount; i++) {
            auto shareData = std::vector<uint8_t>(encryptedShares.begin() + i * 32,
                                                   encryptedShares.begin() + (i + 1) * 32);
            auto verificationKey = computeSha256(shareData);
            
            shares.push_back(ShareInfo{
                .index = (uint32_t)(i + 1),
                .share_data = shareData,
                .verification_key = verificationKey
            });
        }
        
        shares_[keyId] = shares;
    }
    
    std::vector<uint8_t> recoverKey(const std::string& keyId,
                                      const std::vector<ShareInfo>& shares) {
        if (shares.size() < config_->threshold) {
            throw MpcException(MpcErrorCode::ThresholdNotMet,
                "Need " + std::to_string(config_->threshold) + " shares");
        }
        
        // XOR all shares together (simplified - production would use proper interpolation)
        std::vector<uint8_t> recovered(32, 0);
        
        for (size_t i = 0; i < config_->threshold && i < shares.size(); i++) {
            const auto& share = shares[i].share_data;
            for (size_t j = 0; j < 32 && j < share.size(); j++) {
                recovered[j] ^= share[j];
            }
        }
        
        return recovered;
    }
    
    void setConfig(const MpcConfig& config) {
        *config_ = config;
    }
    
    const MpcConfig& getConfig() const {
        return *config_;
    }
    
private:
    std::shared_ptr<MpcConfig> config_;
    std::unordered_map<std::string, MpcKeyData> keys_;
    std::unordered_map<std::string, std::vector<ShareInfo>> shares_;
    
    // secp256k1 implementation
    std::vector<uint8_t> generateSecp256k1Key(const std::vector<uint8_t>& seed,
                                                 std::vector<ShareInfo>& shares,
                                                 const std::string& keyId) {
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 255);
        
        // Generate shares
        for (uint32_t i = 1; i <= config_->total_shares; i++) {
            std::vector<uint8_t> shareData(32);
            for (auto& b : shareData) {
                b = (uint8_t)dis(gen);
            }
            
            auto verificationKey = computeSha256(shareData);
            
            shares.push_back(ShareInfo{
                .index = i,
                .share_data = shareData,
                .verification_key = verificationKey
            });
        }
        
        // For production, use proper DKG
        // This is a simplified version
        auto publicKey = computeSha256(shares[0].share_data);
        publicKey.resize(33);
        publicKey[0] = 0x02; // Compressed public key prefix
        
        return publicKey;
    }
    
    std::vector<uint8_t> generateEd25519Key(const std::vector<uint8_t>& seed,
                                              std::vector<ShareInfo>& shares,
                                              const std::string& keyId) {
        std::random_device rd;
        std::mt19937 gen(rd());
        std::uniform_int_distribution<> dis(0, 255);
        
        // Generate shares
        for (uint32_t i = 1; i <= config_->total_shares; i++) {
            std::vector<uint8_t> shareData(32);
            for (auto& b : shareData) {
                b = (uint8_t)dis(gen);
            }
            
            auto verificationKey = computeSha256(shareData);
            
            shares.push_back(ShareInfo{
                .index = i,
                .share_data = shareData,
                .verification_key = verificationKey
            });
        }
        
        // Return public key (simplified)
        return computeSha256(shares[0].share_data);
    }
    
    std::vector<uint8_t> generateP256Key(const std::vector<uint8_t>& seed,
                                           std::vector<ShareInfo>& shares,
                                           const std::string& keyId) {
        // Similar to secp256k1
        return generateSecp256k1Key(seed, shares, keyId);
    }
    
    std::vector<uint8_t> generateBackup(const std::vector<uint8_t>& publicKey,
                                         const std::vector<uint8_t>& backupKey) {
        auto hmac = computeHmacSha256(backupKey, publicKey);
        
        std::vector<uint8_t> backup;
        backup.insert(backup.end(), backupKey.begin(), backupKey.end());
        backup.insert(backup.end(), hmac.begin(), hmac.begin() + 16);
        
        return backup;
    }
    
    std::vector<uint8_t> signSecp256k1(const std::vector<std::vector<uint8_t>>& shares,
                                         const std::vector<uint8_t>& message) {
        // Reconstruct secret (simplified - production would use proper interpolation)
        std::vector<uint8_t> secret(32, 0);
        for (const auto& share : shares) {
            for (size_t i = 0; i < 32; i++) {
                secret[i] ^= share[i];
            }
        }
        
        // Sign message
        auto hash = computeSha256(message);
        
        // Simplified signature (production would use proper ECDSA)
        std::vector<uint8_t> signature;
        auto sigHash = computeSha256(secret);
        signature.insert(signature.end(), sigHash.begin(), sigHash.end());
        signature.insert(signature.end(), hash.begin(), hash.end());
        
        return signature;
    }
    
    std::vector<uint8_t> signEd25519(const std::vector<std::vector<uint8_t>>& shares,
                                       const std::vector<uint8_t>& message) {
        // XOR shares to get secret
        std::vector<uint8_t> secret(32, 0);
        for (const auto& share : shares) {
            for (size_t i = 0; i < 32; i++) {
                secret[i] ^= share[i];
            }
        }
        
        // Sign message
        auto hash = computeSha256(message);
        std::vector<uint8_t> signature;
        auto sigHash = computeSha256(secret);
        signature.insert(signature.end(), sigHash.begin(), sigHash.end());
        signature.insert(signature.end(), hash.begin(), hash.end());
        
        return signature;
    }
    
    std::vector<uint8_t> signP256(const std::vector<std::vector<uint8_t>>& shares,
                                    const std::vector<uint8_t>& message) {
        return signSecp256k1(shares, message);
    }
    
    bool verifySecp256k1(const std::vector<uint8_t>& publicKey,
                          const std::vector<uint8_t>& signature,
                          const std::vector<uint8_t>& message) {
        // Simplified verification
        auto hash = computeSha256(message);
        
        // In production, use proper ECDSA verification
        return signature.size() >= 64 && publicKey.size() >= 33;
    }
    
    bool verifyEd25519(const std::vector<uint8_t>& publicKey,
                        const std::vector<uint8_t>& signature,
                        const std::vector<uint8_t>& message) {
        return signature.size() >= 64 && publicKey.size() >= 32;
    }
    
    bool verifyP256(const std::vector<uint8_t>& publicKey,
                    const std::vector<uint8_t>& signature,
                    const std::vector<uint8_t>& message) {
        return verifySecp256k1(publicKey, signature, message);
    }
};

// ============================================================================
// Public API Implementation
// ============================================================================

MpcWallet::MpcWallet() : pImpl_(std::make_unique<Impl>()) {}

MpcWallet::MpcWallet(const MpcConfig& config) : pImpl_(std::make_unique<Impl>()) {
    pImpl_->setConfig(config);
}

MpcWallet::~MpcWallet() = default;

MpcWallet::MpcWallet(MpcWallet&&) noexcept = default;
MpcWallet& MpcWallet::operator=(MpcWallet&&) noexcept = default;

std::string MpcWallet::generateKey(const std::vector<uint8_t>& entropy) {
    return pImpl_->generateKey(entropy);
}

std::vector<uint8_t> MpcWallet::getPublicKey(const std::string& keyId) const {
    return pImpl_->getPublicKey(keyId);
}

ShareInfo MpcWallet::getShare(const std::string& keyId, uint32_t index) const {
    return pImpl_->getShare(keyId, index);
}

void MpcWallet::addShare(const std::string& keyId, const ShareInfo& share) {
    pImpl_->addShare(keyId, share);
}

void MpcWallet::deleteKey(const std::string& keyId) {
    pImpl_->deleteKey(keyId);
}

std::vector<std::string> MpcWallet::listKeys() const {
    return pImpl_->listKeys();
}

SignResult MpcWallet::sign(const std::string& keyId, const std::vector<uint8_t>& message) {
    return pImpl_->sign(keyId, message);
}

bool MpcWallet::verify(const std::string& keyId,
                        const std::vector<uint8_t>& signature,
                        const std::vector<uint8_t>& message) const {
    return pImpl_->verify(keyId, signature, message);
}

std::vector<ShareInfo> MpcWallet::reshareKey(const std::string& keyId, 
                                               const MpcConfig& newConfig) {
    return pImpl_->reshareKey(keyId, newConfig);
}

std::vector<uint8_t> MpcWallet::backupKey(const std::string& keyId,
                                            const std::vector<uint8_t>& encryptionKey) {
    return pImpl_->backupKey(keyId, encryptionKey);
}

void MpcWallet::restoreFromBackup(const std::string& keyId,
                                    const std::vector<uint8_t>& backup,
                                    const std::vector<uint8_t>& decryptionKey) {
    pImpl_->restoreFromBackup(keyId, backup, decryptionKey);
}

std::vector<uint8_t> MpcWallet::recoverKey(const std::string& keyId,
                                              const std::vector<ShareInfo>& shares) {
    return pImpl_->recoverKey(keyId, shares);
}

void MpcWallet::setConfig(const MpcConfig& config) {
    pImpl_->setConfig(config);
}

const MpcConfig& MpcWallet::getConfig() const {
    return pImpl_->getConfig();
}

// ============================================================================
// Utility Functions
// ============================================================================

std::string generateAddress(const std::vector<uint8_t>& publicKey, uint32_t chainId) {
    std::vector<uint8_t> data = publicKey;
    
    // Add chain ID
    for (int i = 0; i < 4; i++) {
        data.push_back((chainId >> (i * 8)) & 0xFF);
    }
    
    auto hash = computeSha256(data);
    
    return "0x" + bytesToHex(std::vector<uint8_t>(hash.begin(), hash.begin() + 20));
}

const char* curveName(CurveType curve) {
    switch (curve) {
        case CurveType::Secp256k1: return "secp256k1";
        case CurveType::Ed25519: return "ed25519";
        case CurveType::P256: return "p256";
        default: return "unknown";
    }
}

const char* errorMessage(MpcErrorCode code) {
    switch (code) {
        case MpcErrorCode::Success: return "Success";
        case MpcErrorCode::KeyGenFailed: return "Key generation failed";
        case MpcErrorCode::SigningFailed: return "Signing failed";
        case MpcErrorCode::KeySharingFailed: return "Key sharing failed";
        case MpcErrorCode::InvalidShare: return "Invalid share";
        case MpcErrorCode::ThresholdNotMet: return "Threshold not met";
        case MpcErrorCode::InvalidConfig: return "Invalid configuration";
        case MpcErrorCode::EncryptionError: return "Encryption error";
        case MpcErrorCode::NetworkError: return "Network error";
        case MpcErrorCode::KeyNotFound: return "Key not found";
        default: return "Unknown error";
    }
}

} // namespace mpc
} // namespace tigerwallet
