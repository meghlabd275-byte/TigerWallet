/**
 * TigerWallet Hardware Wallet Service - C++ Ultra-Low Latency Implementation
 * Production-ready integration with Ledger and Trezor hardware wallets
 * 
 * This service provides:
 * - Hardware wallet detection and management
 * - Transaction signing with hardware security
 * - Address derivation (BIP-44)
 * - Multi-chain support
 * - Real-time communication with minimal latency
 */

#ifndef HARDWARE_WALLET_SERVICE_HPP
#define HARDWARE_WALLET_SERVICE_HPP

#include <string>
#include <vector>
#include <memory>
#include <functional>
#include <mutex>
#include <atomic>
#include <thread>
#include <chrono>
#include <array>
#include <optional>
#include <variant>
#include <unordered_map>
#include <functional>

// Crypto libraries
#include <openssl/ec.h>
#include <openssl/evp.h>
#include <openssl/bn.h>
#include <openssl/hmac.h>
#include <openssl/sha.h>
#include <openssl/ripemd.h>

// Platform-specific includes
#ifdef __linux__
#include <libusb.h>
#elif _WIN32
#include <windows.h>
#include <setupapi.h>
#include <hidapi.h>
#elif __APPLE__
#include <IOKit/hid/IOHIDManager.h>
#endif

namespace tigerwallet {
namespace hardware {

// ============================================================================
// CONSTANTS & CONFIGURATION
// ============================================================================

constexpr size_t MAX_DEVICES = 16;
constexpr size_t HID_BUFFER_SIZE = 64;
constexpr size_t MAX_PENDING_TRANSACTIONS = 1000;
constexpr auto DEVICE_POLL_INTERVAL = std::chrono::milliseconds(100);
constexpr auto SIGN_TIMEOUT = std::chrono::seconds(30);
constexpr auto CONNECT_TIMEOUT = std::chrono::seconds(5);

// BIP-44 Purpose constant
constexpr uint32_t BIP44_PURPOSE = 0x8000002C;  // 44'
constexpr uint32_t BIP44_ETH_COIN = 0x8000003C; // 60' for Ethereum
constexpr uint32_t BIP44_BTC_COIN = 0x80000000; // 0' for Bitcoin

// ============================================================================
// ERROR CODES
// ============================================================================

enum class HardwareError {
    SUCCESS = 0,
    NOT_INITIALIZED = 1,
    DEVICE_NOT_FOUND = 2,
    DEVICE_LOCKED = 3,
    DEVICE_BUSY = 4,
    CONNECTION_FAILED = 5,
    TRANSACTIONRejected = 6,
    INVALID_PARAMETER = 7,
    SIGNING_FAILED = 8,
    TIMEOUT = 9,
    MEMORY_ERROR = 10,
    CRYPTO_ERROR = 11,
    PERMISSION_DENIED = 12,
    FIRMWARE_OUTDATED = 13,
    UNSUPPORTED_FEATURE = 14,
    USER_CANCELLED = 15,
    INVALID_SIGNATURE = 16,
    COMMUNICATION_ERROR = 17
};

// ============================================================================
// DATA STRUCTURES
// ============================================================================

/**
 * Supported blockchain types
 */
enum class BlockchainType : uint8_t {
    BITCOIN = 0,
    ETHEREUM = 1,
    POLYGON = 2,
    BSC = 3,
    AVALANCHE = 4,
    SOLANA = 5,
    APTOS = 6,
    SUI = 7,
    NEAR = 8,
    ALGORAND = 9,
    CARDANO = 10,
    NEAR_CHAIN = 11,
    INJECTIVE = 12,
    SEI = 13,
    STARKNET = 14,
    SUBSTRATE = 15,
    ZKSYNC = 16,
    COUNT
};

/**
 * Hardware wallet device types
 */
enum class DeviceType : uint8_t {
    UNKNOWN = 0,
    LEDGER_NANO_S = 1,
    LEDGER_NANO_X = 2,
    LEDGER_NANO_SP = 3,
    TREZOR_ONE = 4,
    TREZOR_T = 5,
    TREZOR_MODEL_T = 6,
    COLDCARD = 7,
    BITBOX02 = 8,
    KEEPKEY = 9
};

/**
 * Transaction data structure
 */
struct TransactionData {
    std::string tx_hash;
    std::vector<uint8_t> raw_tx;
    BlockchainType chain;
    uint64_t nonce;
    uint64_t gas_limit;
    uint64_t gas_price;
    uint64_t value;
    std::string to_address;
    std::vector<uint8_t> data;
    uint64_t chain_id;
};

/**
 * Signed transaction result
 */
struct SignedTransaction {
    std::vector<uint8_t> signature;
    std::vector<uint8_t> public_key;
    std::string signature_hex;
    bool success;
    HardwareError error_code;
    std::string error_message;
    int64_t timestamp_ms;
};

/**
 * Device information
 */
struct DeviceInfo {
    DeviceType type;
    std::string model;
    std::string serial;
    std::string firmware_version;
    bool is_initialized;
    bool is_unlocked;
    bool has_screen;
    bool has_ble;
    uint16_t vendor_id;
    uint16_t product_id;
    std::string path;
};

/**
 * Wallet address information
 */
struct WalletAddress {
    std::string address;
    std::string public_key;
    std::string path;
    BlockchainType chain;
    std::string derivation_index;
};

/**
 * Device event types
 */
enum class DeviceEventType {
    CONNECTED,
    DISCONNECTED,
    LOCKED,
    UNLOCKED,
    BUTTON_PRESS,
    FIRMWARE_UPDATE
};

/**
 * Device event callback
 */
using DeviceEventCallback = std::function<void(DeviceEventType, const DeviceInfo&)>;
using SigningCompleteCallback = std::function<void(const SignedTransaction&)>;
using ProgressCallback = std::function<void(int percentage, const std::string& message)>;

// ============================================================================
// CRYPTO HELPERS (Ultra-Low Latency)
// ============================================================================

class CryptoUtils {
public:
    // Prevent instantiation
    CryptoUtils() = delete;
    
    /**
     * Compute SHA-256 hash (optimized inline)
     */
    static inline std::array<uint8_t, 32> sha256(const std::vector<uint8_t>& data) {
        std::array<uint8_t, 32> hash{};
        SHA256(data.data(), data.size(), hash.data());
        return hash;
    }
    
    /**
     * Compute SHA-256 hash for string
     */
    static inline std::array<uint8_t, 32> sha256(const std::string& str) {
        std::vector<uint8_t> data(str.begin(), str.end());
        return sha256(data);
    }
    
    /**
     * Compute RIPEMD-160 hash
     */
    static inline std::array<uint8_t, 20> ripemd160(const std::vector<uint8_t>& data) {
        std::array<uint8_t, 20> hash{};
        RIPEMD160(data.data(), data.size(), hash.data());
        return hash;
    }
    
    /**
     * Compute double SHA-256 (Bitcoin style)
     */
    static inline std::array<uint8_t, 32> doubleSha256(const std::vector<uint8_t>& data) {
        auto first = sha256(data);
        return sha256(std::vector<uint8_t>(first.begin(), first.end()));
    }
    
    /**
     * HMAC-SHA512
     */
    static inline std::array<uint8_t, 64> hmacSha512(const std::vector<uint8_t>& key, 
                                                        const std::vector<uint8_t>& data) {
        std::array<uint8_t, 64> result{};
        unsigned int len = 64;
        HMAC(EVP_sha512(), key.data(), key.size(), data.data(), data.size(), result.data(), &len);
        return result;
    }
    
    /**
     * PBKDF2-HMAC-SHA512 (for BIP-39)
     */
    static inline std::vector<uint8_t> pbkdf2Sha512(const std::vector<uint8_t>& password,
                                                       const std::vector<uint8_t>& salt,
                                                       uint32_t iterations) {
        std::vector<uint8_t> result(64, 0);
        PKCS5_PBKDF2_HMAC(
            reinterpret_cast<const char*>(password.data()),
            password.size(),
            salt.data(),
            salt.size(),
            iterations,
            EVP_sha512(),
            64,
            result.data()
        );
        return result;
    }
    
    /**
     * Base58 encoding (Bitcoin style)
     */
    static std::string base58Encode(const std::vector<uint8_t>& data);
    
    /**
     * Base58 decoding
     */
    static std::vector<uint8_t> base58Decode(const std::string& encoded);
    
    /**
     * Convert bytes to hex string
     */
    static std::string toHex(const std::vector<uint8_t>& data);
    
    /**
     * Convert hex string to bytes
     */
    static std::vector<uint8_t> fromHex(const std::string& hex);
    
    /**
     * Verify ECDSA signature
     */
    static bool verifySignature(const std::vector<uint8_t>& public_key,
                                const std::vector<uint8_t>& message,
                                const std::vector<uint8_t>& signature);
    
    /**
     * Compress public key
     */
    static std::vector<uint8_t> compressPublicKey(const std::vector<uint8_t>& uncompressed);
    
    /**
     * Decompress public key
     */
    static std::vector<uint8_t> decompressPublicKey(const std::vector<uint8_t>& compressed);
};

// ============================================================================
// HARDWARE WALLET INTERFACE
// ============================================================================

/**
 * Abstract hardware wallet interface
 */
class IHardwareWallet {
public:
    virtual ~IHardwareWallet() = default;
    
    virtual HardwareError initialize() = 0;
    virtual HardwareError connect() = 0;
    virtual void disconnect() = 0;
    virtual bool isConnected() const = 0;
    virtual bool isUnlocked() const = 0;
    
    virtual DeviceInfo getDeviceInfo() const = 0;
    virtual std::vector<WalletAddress> getAddresses(uint32_t start, uint32_t count, BlockchainType chain) = 0;
    
    virtual HardwareError signTransaction(const TransactionData& tx, SignedTransaction& result) = 0;
    virtual HardwareError signMessage(const std::vector<uint8_t>& message, std::vector<uint8_t>& signature) = 0;
    virtual HardwareError signHash(const std::vector<uint8_t>& hash, std::vector<uint8_t>& signature) = 0;
    
    virtual std::string getPublicKey(const std::string& path) = 0;
    virtual std::string getAddress(const std::string& path, BlockchainType chain) = 0;
};

// ============================================================================
// LEDGER HARDWARE WALLET IMPLEMENTATION
// ============================================================================

/**
 * Ledger hardware wallet implementation
 * Supports Nano S, Nano X, and Nano SP
 */
class LedgerWallet : public IHardwareWallet {
public:
    explicit LedgerWallet(DeviceType type);
    ~LedgerWallet() override;
    
    HardwareError initialize() override;
    HardwareError connect() override;
    void disconnect() override;
    bool isConnected() const override;
    bool isUnlocked() const override;
    
    DeviceInfo getDeviceInfo() const override;
    std::vector<WalletAddress> getAddresses(uint32_t start, uint32_t count, BlockchainType chain) override;
    
    HardwareError signTransaction(const TransactionData& tx, SignedTransaction& result) override;
    HardwareError signMessage(const std::vector<uint8_t>& message, std::vector<uint8_t>& signature) override;
    HardwareError signHash(const std::vector<uint8_t>& hash, std::vector<uint8_t>& signature) override;
    
    std::string getPublicKey(const std::string& path) override;
    std::string getAddress(const std::string& path, BlockchainType chain) override;

private:
    DeviceType device_type_;
    DeviceInfo device_info_;
    std::atomic<bool> connected_;
    std::atomic<bool> unlocked_;
    mutable std::mutex mutex_;
    
    // HID communication
    int device_handle_;
    std::thread event_thread_;
    std::atomic<bool> running_;
    
    // APDU commands
    static constexpr uint8_t CLA = 0xE0;
    static constexpr uint8_t INS_GET_APP_VERSION = 0x00;
    static constexpr uint8_t INS_GET_PUBLIC_KEY = 0x02;
    static constexpr uint8_t INS_SIGN_TX = 0x04;
    static constexpr uint8_t INS_SIGN_MESSAGE = 0x08;
    static constexpr uint8_t INS_GET_ADDRESS = 0x0A;
    static constexpr uint8_t INS_SIGN_HASH = 0x0C;
    static constexpr uint8_t INS_GET_ETH_CHAIN_CODE = 0x10;
    
    // Internal methods
    HardwareError sendAPDU(uint8_t cla, uint8_t ins, uint8_t p1, uint8_t p2,
                          const std::vector<uint8_t>& data, std::vector<uint8_t>& response);
    HardwareError exchange(const std::vector<uint8_t>& command, std::vector<uint8_t>& response);
    std::string deriveAddress(const std::vector<uint8_t>& public_key, BlockchainType chain);
    std::string formatBip32Path(const std::vector<uint32_t>& path);
    void startEventLoop();
    void eventLoop();
    bool verifyCertificate(const std::vector<uint8_t>& cert);
};

// ============================================================================
// TREZOR HARDWARE WALLET IMPLEMENTATION
// ============================================================================

/**
 * Trezor hardware wallet implementation
 * Supports Trezor One, Trezor T, and Model T
 */
class TrezorWallet : public IHardwareWallet {
public:
    explicit TrezorWallet(DeviceType type);
    ~TrezorWallet() override;
    
    HardwareError initialize() override;
    HardwareError connect() override;
    void disconnect() override;
    bool isConnected() const override;
    bool isUnlocked() const override;
    
    DeviceInfo getDeviceInfo() const override;
    std::vector<WalletAddress> getAddresses(uint32_t start, uint32_t count, BlockchainType chain) override;
    
    HardwareError signTransaction(const TransactionData& tx, SignedTransaction& result) override;
    HardwareError signMessage(const std::vector<uint8_t>& message, std::vector<uint8_t>& signature) override;
    HardwareError signHash(const std::vector<uint8_t>& hash, std::vector<uint8_t>& signature) override;
    
    std::string getPublicKey(const std::string& path) override;
    std::string getAddress(const std::string& path, BlockchainType chain) override;

private:
    DeviceType device_type_;
    DeviceInfo device_info_;
    std::atomic<bool> connected_;
    std::atomic<bool> unlocked_;
    mutable std::mutex mutex_;
    
    int device_handle_;
    std::thread event_thread_;
    std::atomic<bool> running_;
    uint16_t protocol_version_;
    
    // Trezor message types
    static constexpr uint16_t MSG_INITIALIZE = 0x00;
    static constexpr uint16_t MSG_GET_PUBLIC_KEY = 0x11;
    static constexpr uint16_t MSG_PUBLIC_KEY = 0x11;
    static constexpr uint16_t MSG_SIGN_TX = 0x17;
    static constexpr uint16_t MSG_SIGNED_TX = 0x17;
    static constexpr uint16_t MSG_SIGN_MESSAGE = 0x1A;
    static constexpr uint16_t MSG_SIGNED_MESSAGE = 0x1A;
    static constexpr uint16_t MSG_GET_ADDRESS = 0x21;
    static constexpr uint16_t MSG_ADDRESS = 0x21;
    static constexpr uint16_t MSG_PINMATRIX_REQUEST = 0x2B;
    static constexpr uint16_t MSG_PINMATRIX_ACK = 0x2C;
    static constexpr uint16_t MSG_BUTTON_REQUEST = 0x26;
    static constexpr uint16_t MSG_BUTTON_ACK = 0x27;
    static constexpr uint16_t MSG_PASSPHRASE_REQUEST = 0x2D;
    static constexpr uint16_t MSG_PASSPHRASE_ACK = 0x2E;
    
    // Internal methods
    HardwareError sendMessage(const std::vector<uint8_t>& message, std::vector<uint8_t>& response);
    HardwareError sendMessageWithSession(uint32_t session_id, const std::vector<uint8_t>& message, 
                                         std::vector<uint8_t>& response);
    std::string deriveAddress(const std::vector<uint8_t>& public_key, BlockchainType chain);
    std::vector<uint8_t> serializeMessage(uint16_t msg_type, const std::vector<uint8_t>& data);
    std::pair<uint16_t, std::vector<uint8_t>> parseMessage(const std::vector<uint8_t>& data);
    std::string formatBip32Path(const std::vector<uint32_t>& path);
    void startEventLoop();
    void eventLoop();
};

// ============================================================================
// HARDWARE WALLET MANAGER
// ============================================================================

/**
 * Hardware wallet manager - manages all connected devices
 * Thread-safe, ultra-low latency design
 */
class HardwareWalletManager {
public:
    /**
     * Get singleton instance
     */
    static HardwareWalletManager& getInstance();
    
    /**
     * Initialize the manager
     */
    HardwareError initialize();
    
    /**
     * Shutdown the manager
     */
    void shutdown();
    
    /**
     * Start device discovery
     */
    HardwareError startDeviceDiscovery();
    
    /**
     * Stop device discovery
     */
    void stopDeviceDiscovery();
    
    /**
     * Get all connected devices
     */
    std::vector<DeviceInfo> getConnectedDevices();
    
    /**
     * Get device by type
     */
    std::shared_ptr<IHardwareWallet> getDevice(DeviceType type);
    
    /**
     * Get device by serial
     */
    std::shared_ptr<IHardwareWallet> getDeviceBySerial(const std::string& serial);
    
    /**
     * Register device event callback
     */
    void registerDeviceEventCallback(DeviceEventCallback callback);
    
    /**
     * Check if any device is connected
     */
    bool hasConnectedDevice() const;
    
    /**
     * Check if any device is unlocked
     */
    bool hasUnlockedDevice() const;
    
    /**
     * Get the first available unlocked device
     */
    std::shared_ptr<IHardwareWallet> getFirstAvailableDevice();
    
    /**
     * Set signing timeout
     */
    void setSigningTimeout(std::chrono::seconds timeout);
    
    /**
     * Enable/disable debug mode
     */
    void setDebugMode(bool enabled);

private:
    HardwareWalletManager();
    ~HardwareWalletManager();
    
    // Prevent copying
    HardwareWalletManager(const HardwareWalletManager&) = delete;
    HardwareWalletManager& operator=(const HardwareWalletManager&) = delete;
    
    void discoverDevices();
    void processDevice(DeviceType type, const std::string& path);
    void removeDevice(const std::string& serial);
    void notifyDeviceEvent(DeviceEventType event, const DeviceInfo& info);
    
    // Device management
    std::unordered_map<std::string, std::shared_ptr<IHardwareWallet>> devices_;
    std::unordered_map<std::string, DeviceInfo> device_info_cache_;
    std::vector<DeviceEventCallback> event_callbacks_;
    
    // Threading
    mutable std::mutex mutex_;
    std::thread discovery_thread_;
    std::atomic<bool> running_;
    std::atomic<bool> debug_mode_;
    
    // Configuration
    std::chrono::seconds signing_timeout_;
    
    // Counters for metrics
    std::atomic<uint64_t> total_transactions_signed_;
    std::atomic<uint64_t> total_bytes_transferred_;
    std::atomic<uint64_t> total_errors_;
};

// ============================================================================
// BIP-32 / BIP-44 DERIVATION
// ============================================================================

/**
 * BIP-32 key derivation utility
 */
class Bip32Derivation {
public:
    /**
     * Derive child key from parent key
     */
    static std::pair<std::vector<uint8_t>, std::vector<uint8_t>> deriveChildKey(
        const std::vector<uint8_t>& parent_key,
        const std::vector<uint8_t>& parent_chain_code,
        uint32_t index
    );
    
    /**
     * Derive key from path
     */
    static std::pair<std::vector<uint8_t>, std::vector<uint8_t>> deriveFromPath(
        const std::vector<uint8_t>& seed,
        const std::string& path
    );
    
    /**
     * Parse BIP-32 path string
     */
    static std::vector<uint32_t> parsePath(const std::string& path);
    
    /**
     * Format path to string
     */
    static std::string formatPath(const std::vector<uint32_t>& path);
    
    /**
     * Get Ethereum address path (m/44'/60'/0'/0/0)
     */
    static std::string getEthAddressPath(uint32_t address_index);
    
    /**
     * Get Bitcoin address path (m/44'/0'/0'/0/0)
     */
    static std::string getBtcAddressPath(uint32_t address_index, bool change = false);
    
    /**
     * Validate path
     */
    static bool isValidPath(const std::string& path);
};

// ============================================================================
// TRANSACTION BUILDER
// ============================================================================

/**
 * Transaction builder for hardware wallet signing
 */
class TransactionBuilder {
public:
    TransactionBuilder() = default;
    ~TransactionBuilder() = default;
    
    /**
     * Set blockchain type
     */
    TransactionBuilder& setChain(BlockchainType chain);
    
    /**
     * Set from address
     */
    TransactionBuilder& setFrom(const std::string& from);
    
    /**
     * Set to address
     */
    TransactionBuilder& setTo(const std::string& to);
    
    /**
     * Set value (in wei/satoshi)
     */
    TransactionBuilder& setValue(uint64_t value);
    
    /**
     * Set gas limit
     */
    TransactionBuilder& setGasLimit(uint64_t gas_limit);
    
    /**
     * Set gas price
     */
    TransactionBuilder& setGasPrice(uint64_t gas_price);
    
    /**
     * Set nonce
     */
    TransactionBuilder& setNonce(uint64_t nonce);
    
    /**
     * Set data
     */
    TransactionBuilder& setData(const std::vector<uint8_t>& data);
    
    /**
     * Set chain ID
     */
    TransactionBuilder& setChainId(uint64_t chain_id);
    
    /**
     * Set EIP-1559 parameters
     */
    TransactionBuilder& setEIP1559(uint64_t max_priority_fee, uint64_t max_fee);
    
    /**
     * Build transaction data for signing
     */
    TransactionData build();
    
    /**
     * Encode transaction to RLP (Ethereum)
     */
    std::vector<uint8_t> encodeRLP() const;
    
    /**
     * Get transaction hash
     */
    std::string getTransactionHash() const;

private:
    TransactionData tx_data_;
    std::vector<uint8_t> rlp_encoded_;
    
    void encodeEIP1559();
    void encodeLegacy();
};

// ============================================================================
// INLINE IMPLEMENTATIONS
// ============================================================================

inline HardwareError toError(int code) {
    return static_cast<HardwareError>(code);
}

inline bool isSuccess(HardwareError error) {
    return error == HardwareError::SUCCESS;
}

inline bool isRecoverable(HardwareError error) {
    return error == HardwareError::DEVICE_BUSY || 
           error == HardwareError::TIMEOUT ||
           error == HardwareError::COMMUNICATION_ERROR;
}

} // namespace hardware
} // namespace tigerwallet

#endif // HARDWARE_WALLET_SERVICE_HPP
