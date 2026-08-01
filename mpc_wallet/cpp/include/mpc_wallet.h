/**
 * TigerWallet MPC (Multi-Party Computation) Wallet - C++ Implementation
 * 
 * Ultra-low latency implementation for high-frequency trading
 * Supports threshold signature scheme (TSS) like Bitget Wallet
 * 
 * Features:
 * - Distributed key generation (DKG)
 * - Threshold signing
 * - Key resharing
 * - Key recovery
 * - Hardware security module (HSM) integration ready
 */

#ifndef TIGER_MPC_WALLET_H
#define TIGER_MPC_WALLET_H

#include <string>
#include <vector>
#include <memory>
#include <array>
#include <optional>
#include <functional>
#include <unordered_map>

namespace tigerwallet {
namespace mpc {

// ============================================================================
// Configuration
// ============================================================================

enum class CurveType {
    Secp256k1,  // Bitcoin, Ethereum
    Ed25519,    // Solana, Aptos
    P256
};

struct MpcConfig {
    uint32_t threshold;      // t (minimum shares for signing)
    uint32_t total_shares;  // n (total shares)
    CurveType curve;
    std::string key_id;
    
    MpcConfig() : threshold(2), total_shares(3), curve(CurveType::Secp256k1) {}
};

// ============================================================================
// Data Types
// ============================================================================

struct ShareInfo {
    uint32_t index;                    // Share index (1-based)
    std::vector<uint8_t> share_data;  // Share data
    std::vector<uint8_t> verification_key;  // Verification key
    
    ShareInfo() : index(0) {}
};

struct KeyGenResult {
    std::string key_id;
    std::vector<uint8_t> public_key;
    std::vector<ShareInfo> shares;
    std::vector<uint8_t> backup;  // Encrypted backup
};

struct SignResult {
    std::vector<uint8_t> signature;
    std::vector<uint8_t> public_key;
    std::string key_id;
};

struct MpcKeyData {
    std::string key_id;
    std::vector<uint8_t> public_key;
    CurveType curve;
    uint64_t created_at;
    std::unordered_map<std::string, std::string> metadata;
};

// ============================================================================
// Error Types
// ============================================================================

enum class MpcErrorCode {
    Success = 0,
    KeyGenFailed,
    SigningFailed,
    KeySharingFailed,
    InvalidShare,
    ThresholdNotMet,
    InvalidConfig,
    EncryptionError,
    NetworkError,
    KeyNotFound
};

class MpcException : public std::exception {
public:
    MpcException(MpcErrorCode code, const std::string& message)
        : code_(code), message_(message) {}
    
    const char* what() const noexcept override {
        return message_.c_str();
    }
    
    MpcErrorCode code() const { return code_; }
    
private:
    MpcErrorCode code_;
    std::string message_;
};

// ============================================================================
// MPC Wallet Class
// ============================================================================

class MpcWallet {
public:
    /**
     * Create new MPC wallet
     */
    MpcWallet();
    
    /**
     * Create with custom configuration
     */
    explicit MpcWallet(const MpcConfig& config);
    
    ~MpcWallet();
    
    // Disable copying
    MpcWallet(const MpcWallet&) = delete;
    MpcWallet& operator=(const MpcWallet&) = delete;
    
    // Allow moving
    MpcWallet(MpcWallet&&) noexcept;
    MpcWallet& operator=(MpcWallet&&) noexcept;
    
    // ========================================================================
    // Key Management
    // ========================================================================
    
    /**
     * Generate new MPC key with threshold signature
     * @param entropy Random entropy for key generation
     * @return Key ID
     */
    std::string generateKey(const std::vector<uint8_t>& entropy);
    
    /**
     * Get public key for a key ID
     */
    std::vector<uint8_t> getPublicKey(const std::string& keyId) const;
    
    /**
     * Get share for distribution to participant
     */
    ShareInfo getShare(const std::string& keyId, uint32_t index) const;
    
    /**
     * Add share from distributed participant
     */
    void addShare(const std::string& keyId, const ShareInfo& share);
    
    /**
     * Delete key from wallet
     */
    void deleteKey(const std::string& keyId);
    
    /**
     * List all key IDs
     */
    std::vector<std::string> listKeys() const;
    
    // ========================================================================
    // Signing
    // ========================================================================
    
    /**
     * Sign message with threshold signature
     * Requires threshold number of shares to be available
     */
    SignResult sign(const std::string& keyId, const std::vector<uint8_t>& message);
    
    /**
     * Verify signature
     */
    bool verify(const std::string& keyId, 
                const std::vector<uint8_t>& signature,
                const std::vector<uint8_t>& message) const;
    
    // ========================================================================
    // Key Resharing
    // ========================================================================
    
    /**
     * Reshare key - change threshold or number of participants
     * without changing the underlying key
     */
    std::vector<ShareInfo> reshareKey(const std::string& keyId, 
                                        const MpcConfig& newConfig);
    
    // ========================================================================
    // Backup & Recovery
    // ========================================================================
    
    /**
     * Backup key to encrypted format
     */
    std::vector<uint8_t> backupKey(const std::string& keyId,
                                   const std::vector<uint8_t>& encryptionKey);
    
    /**
     * Restore key from backup
     */
    void restoreFromBackup(const std::string& keyId,
                           const std::vector<uint8_t>& backup,
                           const std::vector<uint8_t>& decryptionKey);
    
    /**
     * Recover key from shares (emergency recovery)
     */
    std::vector<uint8_t> recoverKey(const std::string& keyId,
                                      const std::vector<ShareInfo>& shares);
    
    // ========================================================================
    // Configuration
    // ========================================================================
    
    void setConfig(const MpcConfig& config);
    const MpcConfig& getConfig() const;
    
private:
    // Internal implementation
    class Impl;
    std::unique_ptr<Impl> pImpl_;
};

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Generate address from public key
 */
std::string generateAddress(const std::vector<uint8_t>& publicKey, uint32_t chainId);

/**
 * Get curve name
 */
const char* curveName(CurveType curve);

/**
 * Get error message
 */
const char* errorMessage(MpcErrorCode code);

} // namespace mpc
} // namespace tigerwallet

#endif // TIGER_MPC_WALLET_H
