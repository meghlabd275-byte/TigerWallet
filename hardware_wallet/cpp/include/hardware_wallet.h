/**
 * TigerWallet Hardware Wallet - Production-Ready C++ Implementation
 * Supports Ledger, Trezor, Keystone, and other hardware wallets
 * Ultra-low latency design with thread-safe operations
 * 
 * Copyright (c) 2026 TigerWallet
 * License: MIT
 */

#ifndef TIGERWALLET_HARDWARE_WALLET_H
#define TIGERWALLET_HARDWARE_WALLET_H

#include <cstdint>
#include <cstring>
#include <string>
#include <vector>
#include <array>
#include <memory>
#include <mutex>
#include <thread>
#include <atomic>
#include <functional>
#include <optional>
#include <future>
#include <variant>
#include <unordered_map>

// Platform-specific headers
#ifdef _WIN32
#include <windows.h>
#include <setupapi.h>
#include <hid.h>
#pragma comment(lib, "setupapi.lib")
#pragma comment(lib, "hid.lib")
#elif defined(__APPLE__)
#include <IOKit/hid/IOHIDManager.h>
#include <CoreFoundation/CoreFoundation.h>
#elif defined(__linux__)
#include <linux/hidraw.h>
#include <linux/input.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/ioctl.h>
#endif

namespace tigerwallet {
namespace hardware {

// ============================================================================
// Constants
// ============================================================================

constexpr size_t MAX_HID_PACKET_SIZE = 64;
constexpr size_t MAX_APDU_SIZE = 255;
constexpr size_t ED25519_PUBKEY_SIZE = 32;
constexpr size_t ED25519_SIG_SIZE = 64;
constexpr size_t SECP256K1_PUBKEY_SIZE = 33;
constexpr size_t SECP256K1_SIG_SIZE = 64;
constexpr size_t BIP39_SEED_SIZE = 64;
constexpr size_t CHAINCODE_SIZE = 32;

// Timeout values (milliseconds)
constexpr uint32_t DEFAULT_TIMEOUT = 30000;
constexpr uint32_t SHORT_TIMEOUT = 5000;
constexpr uint32_t SIGNING_TIMEOUT = 120000;

// ============================================================================
// Error Types
// ============================================================================

enum class ErrorCode : int32_t {
    SUCCESS = 0,
    DEVICE_NOT_FOUND = -1,
    DEVICE_BUSY = -2,
    TRANSPORT_ERROR = -3,
    APDU_ERROR = -4,
    USER_CANCELLED = -5,
    TIMEOUT = -6,
    INVALID_RESPONSE = -7,
    INVALID_PARAMETER = -8,
    SECURITY_ERROR = -9,
    NOT_INITIALIZED = -10,
    ALREADY_CONNECTED = -11,
    PIN_RETRY_EXCEEDED = -12,
    WRONG_PIN = -13,
    FIRMWARE_ERROR = -14,
    CRYPTO_ERROR = -15,
    UNSUPPORTED_OPERATION = -16
};

// ============================================================================
// Device Models
// ============================================================================

enum class DeviceModel : uint8_t {
    UNKNOWN = 0,
    // Ledger models
    LEDGER_NANO_S = 1,
    LEDGER_NANO_S_PLUS = 2,
    LEDGER_NANO_X = 3,
    LEDGER_NANO_STAX = 4,
    LEDGER_FLEX = 5,
    // Trezor models
    TREZOR_ONE = 10,
    TREZOR_T = 11,
    TREZOR_SAFE_3 = 12,
    TREZOR_SAFE_5 = 13,
    // Other models
    KEYSTONE_PRO = 20,
    KEYSTONE_PRO_3 = 21,
    ONEKEY_MINI = 30,
    ONEKEY_CLASSIC = 31,
    ELLIPAL_TITAN = 40,
    SAFEPAL_S1 = 50
};

enum class TransportType : uint8_t {
    USB_HID = 0,
    BLUETOOTH = 1,
    BLE = 1,
    NFC = 2
};

enum class ConnectionStatus : uint8_t {
    DISCONNECTED = 0,
    CONNECTING = 1,
    CONNECTED = 2,
    ERROR = 3
};

// ============================================================================
// Data Structures
// ============================================================================

struct DeviceInfo {
    std::string device_id;
    std::string serial_number;
    DeviceModel model;
    TransportType transport;
    std::string firmware_version;
    std::string ble_version;
    bool initialized;
    bool pin_enabled;
    bool passphrase_enabled;
    bool secure_element;
    uint32_t capabilities;
};

struct PublicKeyInfo {
    std::array<uint8_t, SECP256K1_PUBKEY_SIZE> compressed_pubkey;
    std::array<uint8_t, 64> uncompressed_pubkey;
    std::string address;
    std::string path;
};

struct SignatureInfo {
    std::vector<uint8_t> signature;
    std::string tx_hash;
    uint64_t timestamp;
};

struct HDPublicKey {
    std::array<uint8_t, SECP256K1_PUBKEY_SIZE> key;
    std::array<uint8_t, CHAINCODE_SIZE> chaincode;
    std::string path;
};

struct WalletInfo {
    std::string address;
    std::string path;
    std::string derivation_type;
    bool visible;
};

// ============================================================================
// APDU Commands
// ============================================================================

struct APDUCommand {
    uint8_t cla;
    uint8_t ins;
    uint8_t p1;
    uint8_t p2;
    std::vector<uint8_t> data;
    uint32_t response_timeout;

    APDUCommand(uint8_t c, uint8_t i, uint8_t p1_, uint8_t p2_, 
                const std::vector<uint8_t>& d = {})
        : cla(c), ins(i), p1(p1_), p2(p2_), data(d), response_timeout(DEFAULT_TIMEOUT) {}
};

struct APDUResponse {
    std::vector<uint8_t> data;
    uint16_t sw;
    bool more_data;

    bool is_success() const { return sw == 0x9000; }
    bool is_pin_needed() const { return sw == 0x6982; }
    bool isCancelled() const { return sw == 0x6985; }
};

// ============================================================================
// Crypto Utilities
// ============================================================================

class CryptoUtils {
public:
    static std::array<uint8_t, 32> sha256(const std::vector<uint8_t>& data);
    static std::array<uint8_t, 32> sha256(const uint8_t* data, size_t len);
    static std::array<uint8_t, 32> double_sha256(const std::vector<uint8_t>& data);
    static std::string base58_encode(const std::vector<uint8_t>& data);
    static std::vector<uint8_t> base58_decode(const std::string& data);
    static std::string base64_encode(const std::vector<uint8_t>& data);
    static std::vector<uint8_t> base64_decode(const std::string& data);
    static std::array<uint8_t, 4> uint32_to_big_endian(uint32_t value);
    static uint32_t uint32_from_big_endian(const uint8_t* data);
    static std::vector<uint8_t> hex_decode(const std::string& hex);
    static std::string hex_encode(const uint8_t* data, size_t len);
    
    static bool ecdsa_verify(const std::array<uint8_t, 32>& message,
                             const std::array<uint8_t, SECP256K1_SIG_SIZE>& signature,
                             const std::array<uint8_t, SECP256K1_PUBKEY_SIZE>& pubkey);
    
    static std::pair<std::array<uint8_t, 32>, std::array<uint8_t, 32>> 
    derive_child_key(const std::array<uint8_t, 32>& parent_key,
                    const std::array<uint8_t, 32>& chaincode,
                    uint32_t index);
    
    static std::pair<std::array<uint8_t, 32>, std::array<uint8_t, 32>> 
    derive_bip44(const std::vector<uint32_t>& path);
    
    static std::string derive_evm_address(const std::array<uint8_t, SECP256K1_PUBKEY_SIZE>& pubkey);
    static std::string derive_bitcoin_address(const std::array<uint8_t, 33>& pubkey, bool testnet = false);
};

// ============================================================================
// Hardware Wallet Interface
// ============================================================================

class IHardwareWallet {
public:
    virtual ~IHardwareWallet() = default;
    
    virtual ErrorCode connect() = 0;
    virtual ErrorCode disconnect() = 0;
    virtual ConnectionStatus get_status() const = 0;
    virtual std::optional<DeviceInfo> get_device_info() const = 0;
    
    virtual ErrorCode get_public_key(const std::string& path, PublicKeyInfo& info) = 0;
    virtual ErrorCode get_extended_public_key(const std::string& path, HDPublicKey& info) = 0;
    
    virtual ErrorCode sign_transaction(
        const std::vector<uint8_t>& tx_data,
        const std::string& path,
        SignatureInfo& signature) = 0;
    
    virtual ErrorCode sign_message(
        const std::vector<uint8_t>& message,
        const std::string& path,
        SignatureInfo& signature) = 0;
    
    virtual ErrorCode sign_typed_data(
        const std::string& domain,
        const std::string& message,
        const std::string& path,
        SignatureInfo& signature) = 0;
    
    virtual ErrorCode verify_pin(const std::string& pin) = 0;
    virtual ErrorCode change_pin(const std::string& old_pin, const std::string& new_pin) = 0;
    
    virtual ErrorCode enable_passphrase(bool enable) = 0;
    virtual ErrorCode unlock_with_passphrase(const std::string& passphrase) = 0;
    
    virtual ErrorCode get_firmware_version(std::string& version) = 0;
    virtual ErrorCode reboot_to_bootloader() = 0;
    virtual ErrorCode factory_reset() = 0;
    
    virtual ErrorCode get_erc20_token_balance(
        const std::string& token_address,
        const std::string& owner_address,
        std::string& balance) = 0;
    
    virtual ErrorCode get_nft_metadata(
        const std::string& contract_address,
        const std::string& token_id,
        std::string& metadata_json) = 0;
};

// ============================================================================
// Hardware Wallet Manager
// ============================================================================

class HardwareWalletManager {
public:
    static HardwareWalletManager& get_instance();
    
    std::vector<DeviceInfo> discover_devices();
    std::optional<DeviceInfo> get_connected_device(DeviceModel model);
    
    ErrorCode connect_device(DeviceModel model);
    ErrorCode disconnect_device(const std::string& device_id);
    void disconnect_all();
    
    ErrorCode get_wallet_address(const std::string& path, std::string& address);
    ErrorCode sign_transaction(const std::vector<uint8_t>& tx, 
                              const std::string& path,
                              std::vector<uint8_t>& signature);
    ErrorCode sign_message(const std::string& message,
                          const std::string& path,
                          std::vector<uint8_t>& signature);
    
    ErrorCode init_multisig(uint8_t threshold, const std::vector<std::string>& signers);
    ErrorCode sign_multisig(const std::vector<uint8_t>& tx,
                           const std::string& path,
                           std::vector<uint8_t>& signature);
    
    using ConnectionCallback = std::function<void(const std::string&, ConnectionStatus)>;
    using PinCallback = std::function<bool(std::string&)>;
    using ConfirmCallback = std::function<bool(const std::string&, const std::string&)>;
    
    void set_connection_callback(ConnectionCallback callback);
    void set_pin_callback(PinCallback callback);
    void set_confirm_callback(ConfirmCallback callback);
    
    void set_timeout(uint32_t timeout_ms);
    void set_auto_reconnect(bool enable);
    void set_debug_mode(bool enable);
    
private:
    HardwareWalletManager() = default;
    ~HardwareWalletManager() = default;
    HardwareWalletManager(const HardwareWalletManager&) = delete;
    HardwareWalletManager& operator=(const HardwareWalletManager&) = delete;
    
    std::mutex mutex_;
    std::unordered_map<std::string, std::unique_ptr<IHardwareWallet>> connected_devices_;
    ConnectionCallback connection_callback_;
    PinCallback pin_callback_;
    ConfirmCallback confirm_callback_;
    uint32_t timeout_ms_ = DEFAULT_TIMEOUT;
    bool auto_reconnect_ = true;
    bool debug_mode_ = false;
};

// ============================================================================
// Concrete Implementations
// ============================================================================

class LedgerWallet : public IHardwareWallet {
public:
    explicit LedgerWallet(DeviceModel model = DeviceModel::LEDGER_NANO_X);
    ~LedgerWallet() override;
    
    ErrorCode connect() override;
    ErrorCode disconnect() override;
    ConnectionStatus get_status() const override;
    std::optional<DeviceInfo> get_device_info() const override;
    
    ErrorCode get_public_key(const std::string& path, PublicKeyInfo& info) override;
    ErrorCode get_extended_public_key(const std::string& path, HDPublicKey& info) override;
    
    ErrorCode sign_transaction(
        const std::vector<uint8_t>& tx_data,
        const std::string& path,
        SignatureInfo& signature) override;
    
    ErrorCode sign_message(
        const std::vector<uint8_t>& message,
        const std::string& path,
        SignatureInfo& signature) override;
    
    ErrorCode sign_typed_data(
        const std::string& domain,
        const std::string& message,
        const std::string& path,
        SignatureInfo& signature) override;
    
    ErrorCode verify_pin(const std::string& pin) override;
    ErrorCode change_pin(const std::string& old_pin, const std::string& new_pin) override;
    
    ErrorCode enable_passphrase(bool enable) override;
    ErrorCode unlock_with_passphrase(const std::string& passphrase) override;
    
    ErrorCode get_firmware_version(std::string& version) override;
    ErrorCode reboot_to_bootloader() override;
    ErrorCode factory_reset() override;
    
    ErrorCode get_erc20_token_balance(
        const std::string& token_address,
        const std::string& owner_address,
        std::string& balance) override;
    
    ErrorCode get_nft_metadata(
        const std::string& contract_address,
        const std::string& token_id,
        std::string& metadata_json) override;

private:
    ErrorCode open_hid_device(uint16_t vendor_id, uint16_t product_id);
    ErrorCode close_hid_device();
    ErrorCode send_apdu(const APDUCommand& cmd, APDUResponse& response);
    ErrorCode exchange(const std::vector<uint8_t>& data, std::vector<uint8_t>& response);
    
    ErrorCode init_ledger();
    ErrorCode get_device_info_ledger();
    ErrorCode show_address_ledger(const std::string& path, const std::string& address);
    ErrorCode sign_tx_ledger(const std::vector<uint8_t>& tx, const std::string& path, 
                            std::vector<uint8_t>& signature);
    ErrorCode sign_msg_ledger(const std::vector<uint8_t>& message, bool is_compat,
                             const std::string& path, std::vector<uint8_t>& signature);
    
    std::vector<uint32_t> parse_bip32_path(const std::string& path);
    std::string get_app_name() const;
    
    DeviceModel model_;
    ConnectionStatus status_ = ConnectionStatus::DISCONNECTED;
    DeviceInfo device_info_;
    std::mutex mutex_;
    
#ifdef _WIN32
    HANDLE hid_handle_ = INVALID_HANDLE_VALUE;
#elif defined(__APPLE__)
    void* hid_manager_ = nullptr;
    void* hid_device_ = nullptr;
#elif defined(__linux__)
    int hid_fd_ = -1;
#endif
    
    uint8_t channel_id_ = 0;
    uint16_t packet_counter_ = 0;
    bool initialized_ = false;
    std::string wallet_id_;
};

class TrezorWallet : public IHardwareWallet {
public:
    explicit TrezorWallet(DeviceModel model = DeviceModel::TREZOR_T);
    ~TrezorWallet() override;
    
    ErrorCode connect() override;
    ErrorCode disconnect() override;
    ConnectionStatus get_status() const override;
    std::optional<DeviceInfo> get_device_info() const override;
    
    ErrorCode get_public_key(const std::string& path, PublicKeyInfo& info) override;
    ErrorCode get_extended_public_key(const std::string& path, HDPublicKey& info) override;
    
    ErrorCode sign_transaction(
        const std::vector<uint8_t>& tx_data,
        const std::string& path,
        SignatureInfo& signature) override;
    
    ErrorCode sign_message(
        const std::vector<uint8_t>& message,
        const std::string& path,
        SignatureInfo& signature) override;
    
    ErrorCode sign_typed_data(
        const std::string& domain,
        const std::string& message,
        const std::string& path,
        SignatureInfo& signature) override;
    
    ErrorCode verify_pin(const std::string& pin) override;
    ErrorCode change_pin(const std::string& old_pin, const std::string& new_pin) override;
    
    ErrorCode enable_passphrase(bool enable) override;
    ErrorCode unlock_with_passphrase(const std::string& passphrase) override;
    
    ErrorCode get_firmware_version(std::string& version) override;
    ErrorCode reboot_to_bootloader() override;
    ErrorCode factory_reset() override;
    
    ErrorCode get_erc20_token_balance(
        const std::string& token_address,
        const std::string& owner_address,
        std::string& balance) override;
    
    ErrorCode get_nft_metadata(
        const std::string& contract_address,
        const std::string& token_id,
        std::string& metadata_json) override;

private:
    ErrorCode init_transport();
    ErrorCode send_message(const std::vector<uint8_t>& msg, std::vector<uint8_t>& response);
    ErrorCode read_message(std::vector<uint8_t>& msg);
    
    ErrorCode init_trezor();
    ErrorCode get_public_key_trezor(const std::string& path, HDPublicKey& key);
    ErrorCode sign_tx_trezor(const std::vector<uint8_t>& tx, const std::string& path,
                            std::vector<uint8_t>& signature);
    
    DeviceModel model_;
    ConnectionStatus status_ = ConnectionStatus::DISCONNECTED;
    DeviceInfo device_info_;
    std::mutex mutex_;
    int socket_fd_ = -1;
    bool initialized_ = false;
    uint32_t session_id_ = 0;
};

// ============================================================================
// BIP32/39/44 Implementation
// ============================================================================

class BIP32 {
public:
    static std::pair<std::array<uint8_t, 32>, std::array<uint8_t, 32>> 
    master_key_from_seed(const std::vector<uint8_t>& seed);
    
    static std::pair<std::array<uint8_t, 32>, std::array<uint8_t, 32>> 
    derive_child_key(const std::array<uint8_t, 32>& key,
                    const std::array<uint8_t, 32>& chaincode,
                    uint32_t index);
    
    static std::string derive_address(const std::array<uint8_t, 33>& pubkey, 
                                     const std::string& chain);
    
    static std::vector<WalletInfo> generate_addresses(
        const std::string& chain,
        uint32_t start_index,
        uint32_t count);
};

class BIP39 {
public:
    static std::vector<uint8_t> generate_mnemonic(uint32_t strength = 256);
    static std::vector<uint8_t> mnemonic_to_seed(const std::string& mnemonic,
                                                 const std::string& passphrase = "");
    static bool validate_mnemonic(const std::string& mnemonic);
    static std::vector<std::string> get_wordlist();
    
private:
    static const std::vector<std::string>& wordlist();
    static bool checksum_valid(const std::vector<uint8_t>& entropy);
};

class BIP44 {
public:
    struct PathComponents {
        uint32_t purpose;
        uint32_t coin_type;
        uint32_t account;
        uint32_t change;
        uint32_t address_index;
    };
    
    static PathComponents parse_path(const std::string& path);
    static std::string build_path(const PathComponents& components);
    static std::string build_path(uint32_t coin_type, uint32_t account = 0, 
                                   uint32_t change = 0, uint32_t index = 0);
    
    static constexpr uint32_t BTC = 0;
    static constexpr uint32_t ETH = 60;
    static constexpr uint32_t SOL = 501;
    static constexpr uint32_t TRX = 195;
    static constexpr uint32_t ATOM = 118;
    static constexpr uint32_t DOT = 354;
    static constexpr uint32_t NEAR = 397;
    static constexpr uint32_t APTOS = 637;
    static constexpr uint32_t SUI = 784;
};

// ============================================================================
// Cross-Platform HID Manager
// ============================================================================

class HIDManager {
public:
    static std::vector<DeviceInfo> enumerate_devices();
    static std::unique_ptr<IHardwareWallet> create_wallet(const DeviceInfo& info);
    
    static constexpr uint16_t LEDGER_VID = 0x2c97;
    static constexpr uint16_t TREZOR_VID = 0x1209;
    static constexpr uint16_t KEYSTONE_VID = 0x2d47;
    static constexpr uint16_t ONEKEY_VID = 0x2a2c;
};

} // namespace hardware
} // namespace tigerwallet

#endif // TIGERWALLET_HARDWARE_WALLET_H
