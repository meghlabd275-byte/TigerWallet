/**
 * TigerWallet Hardware Wallet Integration - C++ Core
 * Secure hardware wallet support for Ledger and Trezor devices
 */

#ifndef TIGER_HARDWARE_WALLET_H
#define TIGER_HARDWARE_WALLET_H

#include <array>
#include <atomic>
#include <cstdint>
#include <memory>
#include <mutex>
#include <optional>
#include <shared_mutex>
#include <string>
#include <string_view>
#include <thread>
#include <vector>

// Platform-specific includes
#if defined(__linux__)
    #include <hidapi.h>
#elif defined(_WIN32)
    #include <windows.h>
    #include <hid.h>
#elif defined(__APPLE__)
    #include <IOKit/hid/IOHIDManager.h>
#endif

namespace tiger {
namespace hardware {

// ============================================================================
// Constants
// ============================================================================

constexpr size_t MAX_DEVICES = 16;
constexpr size_t MAX_PENDING_TRANSACTIONS = 100;
constexpr size_t HID_BUFFER_SIZE = 4096;
constexpr uint32_t HID_REPORT_SIZE = 64;

// Device types
enum class DeviceType : uint8_t {
    UNKNOWN = 0,
    LEDGER_NANO_S = 1,
    LEDGER_NANO_X = 2,
    LEDGER_NANO_SP = 3,
    TREZOR_ONE = 4,
    TREZOR_T = 5,
    TREZOR_MODEL_T = 6,
};

// Device status
enum class DeviceStatus : uint8_t {
    DISCONNECTED = 0,
    CONNECTED = 1,
    UNLOCKED = 2,
    BUSY = 3,
    ERROR = 4,
};

// Signing status
enum class SignStatus : uint8_t {
    IDLE = 0,
    PENDING = 1,
    CONFIRMING = 2,
    SIGNED = 3,
    REJECTED = 4,
    FAILED = 5,
};

// ============================================================================
// Data Structures
// ============================================================================

struct DeviceInfo {
    std::string device_id;
    std::string serial_number;
    std::string manufacturer;
    std::string product_name;
    DeviceType device_type;
    DeviceStatus status;
    uint16_t vendor_id;
    uint16_t product_id;
    uint8_t firmware_version[4];
    bool supports_eth;
    bool supports_btc;
    bool supports_solana;
    bool initialized;
    uint64_t last_seen;
    
    DeviceInfo() : device_type(DeviceType::UNKNOWN), status(DeviceStatus::DISCONNECTED),
                  vendor_id(0), product_id(0), supports_eth(false),
                  supports_btc(false), supports_solana(false), initialized(false), last_seen(0) {
        firmware_version[0] = firmware_version[1] = firmware_version[2] = firmware_version[3] = 0;
    }
};

struct PublicKey {
    std::vector<uint8_t> data;
    std::string address;
    uint32_t chain_id;
    
    PublicKey() : chain_id(0) {}
};

struct Signature {
    std::vector<uint8_t> r;
    std::vector<uint8_t> s;
    uint8_t v;
    std::vector<uint8_t> serialized;
    bool is_derivation_successful;
    std::string error_message;
    
    Signature() : v(0), is_derivation_successful(false) {}
};

struct TransactionData {
    uint64_t chain_id;
    std::vector<uint8_t> nonce;
    std::vector<uint8_t> gas_price;
    std::vector<uint8_t> gas_limit;
    std::string to;
    std::vector<uint8_t> value;
    std::vector<uint8_t> data;
    uint64_t transaction_type;
    uint64_t access_list;
    
    TransactionData() : chain_id(1), transaction_type(0), access_list(0) {}
};

struct SignRequest {
    uint64_t request_id;
    uint64_t account_id;
    std::string device_id;
    TransactionData transaction;
    SignStatus status;
    std::string status_message;
    uint64_t created_at;
    uint64_t confirmed_at;
    std::optional<Signature> signature;
    
    SignRequest() : request_id(0), account_id(0), status(SignStatus::IDLE),
                    created_at(0), confirmed_at(0) {}
};

struct AccountPath {
    uint32_t purpose;     // 44' for BIP44
    uint32_t coin_type;    // 60' for Ethereum
    uint32_t account;     // Account index
    uint32_t change;       // 0 for external, 1 for internal
    uint32_t address_idx;  // Address index
    
    AccountPath() : purpose(44), coin_type(60), account(0), change(0), address_idx(0) {}
    
    std::string to_string() const {
        return "m/" + std::to_string(purpose) + "'/" + 
               std::to_string(coin_type) + "'/" + 
               std::to_string(account) + "'/" + 
               std::to_string(change) + "/" + 
               std::to_string(address_idx);
    }
};

// ============================================================================
// HID Communication
// ============================================================================

class HIDDevice {
private:
#if defined(__linux__) || defined(__APPLE__)
    hid_device* device_;
#elif defined(_WIN32)
    HANDLE device_;
#endif
    DeviceInfo info_;
    std::mutex write_mutex_;
    std::mutex read_mutex_;
    
public:
    HIDDevice();
    ~HIDDevice();
    
    bool open(uint16_t vendor_id, uint16_t product_id, const std::string& serial = "");
    void close();
    bool is_open() const;
    
    int write(const uint8_t* data, size_t length);
    int read(uint8_t* data, size_t length, int timeout_ms = 1000);
    
    DeviceInfo get_info() const { return info_; }
    void set_info(const DeviceInfo& info) { info_ = info; }
};

// ============================================================================
// Hardware Wallet Manager
// ============================================================================

class HardwareWalletManager {
private:
    std::vector<std::unique_ptr<HIDDevice>> devices_;
    std::unordered_map<std::string, DeviceInfo> connected_devices_;
    std::unordered_map<std::string, SignRequest> pending_signatures_;
    
    std::atomic<uint64_t> next_request_id_;
    std::atomic<bool> running_;
    
    mutable std::shared_mutex mutex_;
    std::thread monitor_thread_;
    
    // Device callbacks
    std::function<void(const DeviceInfo&)> on_device_connected_;
    std::function<void(const std::string&)> on_device_disconnected_;
    std::function<void(const SignRequest&)> on_signature_request_;
    std::function<void(const SignRequest&)> on_signature_complete_;
    
    void monitor_devices();
    bool detect_device_type(HIDDevice& device, DeviceInfo& info);
    bool communicate_with_ledger(HIDDevice& device, const std::vector<uint8_t>& command, std::vector<uint8_t>& response);
    bool communicate_with_trezor(HIDDevice& device, const std::vector<uint8_t>& command, std::vector<uint8_t>& response);
    
public:
    HardwareWalletManager();
    ~HardwareWalletManager();
    
    void start_monitoring();
    void stop_monitoring();
    
    // Device management
    std::vector<DeviceInfo> list_devices();
    std::optional<DeviceInfo> get_device(const std::string& device_id);
    bool connect_device(const std::string& device_id);
    void disconnect_device(const std::string& device_id);
    
    // Public key operations
    std::optional<PublicKey> get_public_key(
        const std::string& device_id,
        const AccountPath& path,
        bool display = false
    );
    
    std::vector<PublicKey> get_multiple_addresses(
        const std::string& device_id,
        const AccountPath& base_path,
        uint32_t count,
        bool display = false
    );
    
    // Signing operations
    uint64_t sign_transaction(
        const std::string& device_id,
        const AccountPath& path,
        const TransactionData& transaction
    );
    
    std::optional<Signature> get_signature(uint64_t request_id);
    bool cancel_sign_request(uint64_t request_id);
    
    // Message signing
    std::optional<Signature> sign_message(
        const std::string& device_id,
        const AccountPath& path,
        const std::string& message,
        bool is_typed_data = false
    );
    
    // Device operations
    bool verify_address(const std::string& device_id, const AccountPath& path, const std::string& address);
    bool get_device_info(const std::string& device_id, DeviceInfo& info);
    bool initialize_device(const std::string& device_id, const std::string& pin);
    bool factory_reset(const std::string& device_id);
    
    // Callbacks
    void set_on_device_connected(std::function<void(const DeviceInfo&)> callback);
    void set_on_device_disconnected(std::function<void(const std::string&)> callback);
    void set_on_signature_request(std::function<void(const SignRequest&)> callback);
    void set_on_signature_complete(std::function<void(const SignRequest&)> callback);
    
    // Static device detection
    static std::vector<DeviceInfo> enumerate_devices();
};

// ============================================================================
// Ledger Protocol
// ============================================================================

class LedgerProtocol {
private:
    static constexpr uint8_t CLA = 0xE0;
    static constexpr uint8_t INS_GET_PUBLIC_KEY = 0x02;
    static constexpr uint8_t INS_SIGN = 0x04;
    static constexpr uint8_t INS_SIGN_TX = 0x02;
    static constexpr uint8_t INS_GET_APP_VERSION = 0x00;
    static constexpr uint8_t INS_GET_CONFIG = 0x06;
    
public:
    static std::vector<uint8_t> build_get_public_key_cmd(
        const AccountPath& path,
        bool display = false,
        bool chain_code = false
    );
    
    static std::vector<uint8_t> build_sign_transaction_cmd(
        const TransactionData& transaction,
        const AccountPath& path
    );
    
    static std::vector<uint8_t> build_sign_message_cmd(
        const std::vector<uint8_t>& message_hash,
        const AccountPath& path
    );
    
    static std::vector<uint8_t> build_get_app_version_cmd();
    
    static bool parse_public_key_response(
        const std::vector<uint8_t>& response,
        PublicKey& public_key
    );
    
    static bool parse_signature_response(
        const std::vector<uint8_t>& response,
        Signature& signature
    );
};

// ============================================================================
// Trezor Protocol
// ============================================================================

class TrezorProtocol {
private:
    static constexpr uint16_t TREZOR_VENDOR_ID = 0x534C;
    static constexpr uint16_t TREZOR_ONE_PRODUCT_ID = 0x0001;
    static constexpr uint16_t TREZOR_T_PRODUCT_ID = 0x0002;
    
    static constexpr uint8_t MSG_GET_PUBLIC_KEY = 0x11;
    static constexpr uint8_t MSG_SIGN_TX = 0x1D;
    static constexpr uint8_t MSG_SIGN_MESSAGE = 0x1A;
    static constexpr uint8_t MSG_GET_FEATURES = 0x0C;
    
public:
    static std::vector<uint8_t> build_get_public_key_msg(
        const AccountPath& path,
        bool display = false
    );
    
    static std::vector<uint8_t> build_sign_tx_msg(
        const TransactionData& transaction,
        const AccountPath& path
    );
    
    static std::vector<uint8_t> build_sign_message_msg(
        const std::string& message,
        const AccountPath& path
    );
    
    static std::vector<uint8_t> build_get_features_msg();
    
    static bool parse_public_key_message(
        const std::vector<uint8_t>& message,
        PublicKey& public_key
    );
    
    static bool parse_signature_message(
        const std::vector<uint8_t>& message,
        Signature& signature
    );
};

// ============================================================================
// Inline Implementations
// ============================================================================

inline HardwareWalletManager::HardwareWalletManager()
    : next_request_id_(1), running_(false) {}

inline HardwareWalletManager::~HardwareWalletManager() {
    stop_monitoring();
}

inline void HardwareWalletManager::start_monitoring() {
    if (running_.load()) return;
    running_.store(true);
    monitor_thread_ = std::thread([this]() { monitor_devices(); });
}

inline void HardwareWalletManager::stop_monitoring() {
    if (!running_.load()) return;
    running_.store(false);
    if (monitor_thread_.joinable()) {
        monitor_thread_.join();
    }
}

inline std::vector<DeviceInfo> HardwareWalletManager::list_devices() {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    std::vector<DeviceInfo> result;
    for (const auto& [id, info] : connected_devices_) {
        result.push_back(info);
    }
    return result;
}

inline std::optional<DeviceInfo> HardwareWalletManager::get_device(const std::string& device_id) {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    auto it = connected_devices_.find(device_id);
    if (it != connected_devices_.end()) {
        return it->second;
    }
    return std::nullopt;
}

inline uint64_t HardwareWalletManager::sign_transaction(
    const std::string& device_id,
    const AccountPath& path,
    const TransactionData& transaction
) {
    uint64_t request_id = next_request_id_++;
    
    SignRequest request;
    request.request_id = request_id;
    request.device_id = device_id;
    request.transaction = transaction;
    request.status = SignStatus::PENDING;
    request.created_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
    
    {
        std::unique_lock<std::shared_mutex> lock(mutex_);
        pending_signatures_[device_id] = request;
    }
    
    // Notify callback
    if (on_signature_request_) {
        on_signature_request_(request);
    }
    
    return request_id;
}

inline std::optional<Signature> HardwareWalletManager::get_signature(uint64_t request_id) {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    for (const auto& [device_id, request] : pending_signatures_) {
        if (request.request_id == request_id && request.signature.has_value()) {
            return request.signature;
        }
    }
    return std::nullopt;
}

inline void HardwareWalletManager::set_on_device_connected(
    std::function<void(const DeviceInfo&)> callback
) {
    on_device_connected_ = callback;
}

inline void HardwareWalletManager::set_on_device_disconnected(
    std::function<void(const std::string&)> callback
) {
    on_device_disconnected_ = callback;
}

inline void HardwareWalletManager::set_on_signature_request(
    std::function<void(const SignRequest&)> callback
) {
    on_signature_request_ = callback;
}

inline void HardwareWalletManager::set_on_signature_complete(
    std::function<void(const SignRequest&)> callback
) {
    on_signature_complete_ = callback;
}

// Static method
inline std::vector<DeviceInfo> HardwareWalletManager::enumerate_devices() {
    std::vector<DeviceInfo> devices;
    
    // Known device IDs
    const std::vector<std::pair<uint16_t, uint16_t>> known_devices = {
        {0x2C97, 0x0001},  // Ledger Nano S
        {0x2C97, 0x0004},  // Ledger Nano X
        {0x2C97, 0x0010},  // Ledger Nano SP
        {0x534C, 0x0001},  // Trezor One
        {0x534C, 0x0002},  // Trezor T / Model T
    };
    
    for (const auto& [vid, pid] : known_devices) {
        DeviceInfo info;
        info.vendor_id = vid;
        info.product_id = pid;
        
        if (vid == 0x2C97) {
            info.manufacturer = "Ledger";
            info.device_type = (pid == 0x0001) ? DeviceType::LEDGER_NANO_S :
                            (pid == 0x0004) ? DeviceType::LEDGER_NANO_X :
                            DeviceType::LEDGER_NANO_SP;
            info.supports_eth = true;
            info.supports_btc = true;
        } else if (vid == 0x534C) {
            info.manufacturer = "SatoshiLabs";
            info.device_type = (pid == 0x0001) ? DeviceType::TREZOR_ONE : DeviceType::TREZOR_MODEL_T;
            info.supports_eth = true;
            info.supports_btc = true;
        }
        
        devices.push_back(info);
    }
    
    return devices;
}

} // namespace hardware
} // namespace tiger

#endif // TIGER_HARDWARE_WALLET_H
