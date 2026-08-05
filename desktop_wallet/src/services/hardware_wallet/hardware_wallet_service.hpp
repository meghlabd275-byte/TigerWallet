#ifndef TIGERWALLET_HARDWARE_WALLET_HPP
#define TIGERWALLET_HARDWARE_WALLET_HPP

#include <string>
#include <vector>
#include <map>
#include <mutex>
#include <chrono>
#include <functional>
#include <memory>
#include <atomic>

namespace tigerwallet {

// =============================================================================
// HARDWARE WALLET TYPES
// =============================================================================

enum class HardwareWalletType {
    LEDGER,
    TREZOR,
    KEEPKEY,
    COLDCARD,
    BITBOX02,
    UNKNOWN
};

enum class HardwareWalletStatus {
    DISCONNECTED,
    CONNECTED,
    LOCKED,
    UNLOCKED,
    BUSY,
    ERROR
};

struct HardwareWalletInfo {
    std::string device_id;
    HardwareWalletType device_type;
    std::string device_name;
    std::string firmware_version;
    std::string bootloader_version;
    bool is_initialized;
    bool is_backup;
    HardwareWalletStatus status;
    std::string public_key;
    std::string address;
    uint64_t connected_at;
    
    HardwareWalletInfo() : device_type(HardwareWalletType::UNKNOWN), 
                          is_initialized(false), is_backup(false),
                          status(HardwareWalletStatus::DISCONNECTED), connected_at(0) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"deviceId\":\"" << device_id << "\",";
        oss << "\"deviceType\":\"" << static_cast<int>(device_type) << "\",";
        oss << "\"deviceName\":\"" << device_name << "\",";
        oss << "\"firmwareVersion\":\"" << firmware_version << "\",";
        oss << "\"isInitialized\":" << (is_initialized ? "true" : "false") << ",";
        oss << "\"status\":\"" << static_cast<int>(status) << "\"";
        oss << "}";
        return oss.str();
    }
};

struct TransactionSignature {
    std::string signature;
    std::string public_key;
    std::string derivation_path;
    uint64_t signed_at;
    
    TransactionSignature() : signed_at(0) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"signature\":\"" << signature << "\",";
        oss << "\"publicKey\":\"" << public_key << "\",";
        oss << "\"derivationPath\":\"" << derivation_path << "\"";
        oss << "}";
        return oss.str();
    }
};

struct SupportedChain {
    std::string chain_id;
    std::string name;
    std::string symbol;
    std::vector<std::string> derivation_paths;
    bool supported;
    
    SupportedChain() : supported(true) {}
};

struct DeviceConfiguration {
    std::string vendor_id;
    std::string product_id;
    std::string interface_class;
    std::vector<SupportedChain> supported_chains;
    uint32_t timeout_ms;
    bool requires_pin;
    bool requires_passphrase;
    
    DeviceConfiguration() : timeout_ms(30000), requires_pin(true), requires_passphrase(false) {}
};

// =============================================================================
// HARDWARE WALLET SERVICE IMPLEMENTATION
// =============================================================================

class HardwareWalletService {
private:
    std::map<std::string, HardwareWalletInfo> connected_devices;
    std::map<HardwareWalletType, DeviceConfiguration> device_configs;
    std::mutex devices_mutex;
    
    std::atomic<bool> is_scanning{false};
    bool initialized;
    
    uint64_t getCurrentTimestamp() const {
        return std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
    
    void initializeDeviceConfigs() {
        // Ledger configuration
        DeviceConfiguration ledger;
        ledger.vendor_id = "0x2c97";
        ledger.product_id = "0x0001";
        ledger.interface_class = "0x03";
        ledger.timeout_ms = 60000;
        ledger.requires_pin = true;
        ledger.requires_passphrase = false;
        
        ledger.supported_chains = {
            {"1", "Ethereum", "ETH", {"m/44'/60'/0'/0/0", "m/44'/60'/0'/0/1"}, true},
            {"56", "BNB Chain", "BNB", {"m/44'/60'/0'/0/0"}, true},
            {"137", "Polygon", "MATIC", {"m/44'/60'/0'/0/0"}, true},
            {"43114", "Avalanche", "AVAX", {"m/44'/60'/0'/0/0"}, true}
        };
        device_configs[HardwareWalletType::LEDGER] = ledger;
        
        // Trezor configuration
        DeviceConfiguration trezor;
        trezor.vendor_id = "0x1209";
        trezor.product_id = "0x53c1";
        trezor.timeout_ms = 60000;
        trezor.requires_pin = true;
        trezor.requires_passphrase = true;
        
        trezor.supported_chains = {
            {"1", "Ethereum", "ETH", {"m/44'/60'/0'/0/0"}, true},
            {"56", "BNB Chain", "BNB", {"m/44'/60'/0'/0/0"}, true}
        };
        device_configs[HardwareWalletType::TREZOR] = trezor;
        
        // KeepKey configuration
        DeviceConfiguration keepkey;
        keepkey.vendor_id = "0x2c24";
        keepkey.product_id = "0x0001";
        keepkey.timeout_ms = 60000;
        keepkey.requires_pin = true;
        keepkey.requires_passphrase = true;
        
        keepkey.supported_chains = {
            {"1", "Ethereum", "ETH", {"m/44'/60'/0'/0/0"}, true}
        };
        device_configs[HardwareWalletType::KEEPKEY] = keepkey;
        
        // ColdCard configuration
        DeviceConfiguration coldcard;
        coldcard.vendor_id = "0xd13e";
        coldcard.product_id = "0x0001";
        coldcard.timeout_ms = 30000;
        coldcard.requires_pin = true;
        coldcard.requires_passphrase = false;
        
        coldcard.supported_chains = {
            {"0", "Bitcoin", "BTC", {"m/44'/0'/0'/0/0", "m/84'/0'/0'/0/0"}, true}
        };
        device_configs[HardwareWalletType::COLDCARD] = coldcard;
        
        // BitBox02 configuration
        DeviceConfiguration bitbox02;
        bitbox02.vendor_id = "0x03eb";
        bitbox02.product_id = "0x2402";
        bitbox02.timeout_ms = 60000;
        bitbox02.requires_pin = true;
        bitbox02.requires_passphrase = true;
        
        bitbox02.supported_chains = {
            {"0", "Bitcoin", "BTC", {"m/44'/0'/0'/0/0"}, true},
            {"1", "Ethereum", "ETH", {"m/44'/60'/0'/0/0"}, true}
        };
        device_configs[HardwareWalletType::BITBOX02] = bitbox02;
    }
    
public:
    HardwareWalletService() : initialized(false) {}
    
    ~HardwareWalletService() {}
    
    bool initialize() {
        if (initialized) return true;
        
        std::lock_guard<std::mutex> lock(devices_mutex);
        initializeDeviceConfigs();
        
        initialized = true;
        return true;
    }
    
    // Get supported device types
    std::vector<HardwareWalletType> getSupportedDeviceTypes() const {
        return {
            HardwareWalletType::LEDGER,
            HardwareWalletType::TREZOR,
            HardwareWalletType::KEEPKEY,
            HardwareWalletType::COLDCARD,
            HardwareWalletType::BITBOX02
        };
    }
    
    // Get device configuration
    DeviceConfiguration getDeviceConfig(HardwareWalletType device_type) const {
        auto it = device_configs.find(device_type);
        if (it != device_configs.end()) {
            return it->second;
        }
        return DeviceConfiguration();
    }
    
    // Get chains for device
    std::vector<SupportedChain> getSupportedChains(HardwareWalletType device_type) const {
        auto it = device_configs.find(device_type);
        if (it != device_configs.end()) {
            return it->second.supported_chains;
        }
        return {};
    }
    
    // Connect to device (simulated)
    bool connectDevice(const std::string& device_id, HardwareWalletType device_type) {
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        HardwareWalletInfo info;
        info.device_id = device_id;
        info.device_type = device_type;
        info.status = HardwareWalletStatus::CONNECTED;
        info.connected_at = getCurrentTimestamp();
        
        switch (device_type) {
            case HardwareWalletType::LEDGER:
                info.device_name = "Ledger Nano X";
                info.firmware_version = "2.1.0";
                break;
            case HardwareWalletType::TREZOR:
                info.device_name = "Trezor Model T";
                info.firmware_version = "2.6.0";
                break;
            case HardwareWalletType::KEEPKEY:
                info.device_name = "KeepKey";
                info.firmware_version = "7.1.0";
                break;
            case HardwareWalletType::COLDCARD:
                info.device_name = "ColdCard Mk4";
                info.firmware_version = "5.0.2";
                break;
            case HardwareWalletType::BITBOX02:
                info.device_name = "BitBox02";
                info.firmware_version = "9.12.0";
                break;
            default:
                info.device_name = "Unknown Device";
        }
        
        info.is_initialized = true;
        
        connected_devices[device_id] = info;
        return true;
    }
    
    // Disconnect device
    bool disconnectDevice(const std::string& device_id) {
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        auto it = connected_devices.find(device_id);
        if (it != connected_devices.end()) {
            connected_devices.erase(it);
            return true;
        }
        return false;
    }
    
    // Get connected devices
    std::vector<HardwareWalletInfo> getConnectedDevices() const {
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        std::vector<HardwareWalletInfo> result;
        for (const auto& pair : connected_devices) {
            result.push_back(pair.second);
        }
        return result;
    }
    
    // Get device info
    HardwareWalletInfo getDeviceInfo(const std::string& device_id) const {
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        auto it = connected_devices.find(device_id);
        if (it != connected_devices.end()) {
            return it->second;
        }
        return HardwareWalletInfo();
    }
    
    // Get address from device
    std::string getAddress(const std::string& device_id, 
                          const std::string& chain,
                          const std::string& derivation_path) {
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        auto it = connected_devices.find(device_id);
        if (it == connected_devices.end()) {
            return "";
        }
        
        // In production, this would communicate with the device
        // For now, return a derived address based on the path
        std::ostringstream oss;
        oss << "0x" << std::hex;
        
        // Simple hash of derivation path for demo
        size_t hash = 0;
        for (char c : derivation_path) {
            hash = hash * 31 + c;
        }
        
        for (int i = 0; i < 40; i++) {
            oss << ((hash + i) % 16);
        }
        
        return oss.str();
    }
    
    // Sign transaction
    TransactionSignature signTransaction(const std::string& device_id,
                                       const std::string& chain,
                                       const std::string& derivation_path,
                                       const std::string& transaction_data) {
        TransactionSignature signature;
        
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        auto it = connected_devices.find(device_id);
        if (it == connected_devices.end()) {
            return signature;
        }
        
        HardwareWalletInfo& device = it->second;
        
        if (device.status != HardwareWalletStatus::UNLOCKED) {
            device.status = HardwareWalletStatus::BUSY;
            // Simulate signing delay
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
        }
        
        // In production, this would send the transaction to the device for signing
        // and receive the signature back
        
        signature.derivation_path = derivation_path;
        signature.signed_at = getCurrentTimestamp();
        
        // Generate a mock signature (in production, this comes from the device)
        signature.signature = "0x";
        for (int i = 0; i < 130; i++) {
            signature.signation += "0123456789abcdef"[i % 16];
        }
        
        device.status = HardwareWalletStatus::UNLOCKED;
        
        return signature;
    }
    
    // Sign message
    TransactionSignature signMessage(const std::string& device_id,
                                   const std::string& derivation_path,
                                   const std::string& message) {
        TransactionSignature signature;
        
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        auto it = connected_devices.find(device_id);
        if (it == connected_devices.end()) {
            return signature;
        }
        
        signature.derivation_path = derivation_path;
        signature.signed_at = getCurrentTimestamp();
        
        // Generate mock signature
        signature.signature = "0x";
        for (size_t i = 0; i < message.length() && i < 130; i++) {
            signature.signature += "0123456789abcdef"[message[i] % 16];
        }
        
        return signature;
    }
    
    // Verify address on device
    bool verifyAddress(const std::string& device_id, const std::string& address) {
        // In production, this would prompt the device to display and verify the address
        // For now, just check if the device is connected
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        auto it = connected_devices.find(device_id);
        return it != connected_devices.end();
    }
    
    // Get public key
    std::string getPublicKey(const std::string& device_id, const std::string& derivation_path) {
        // In production, this would communicate with the device
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        auto it = connected_devices.find(device_id);
        if (it == connected_devices.end()) {
            return "";
        }
        
        // Return mock public key
        return "0x04" + std::string(128, 'a');
    }
    
    // Check if device is connected
    bool isDeviceConnected(const std::string& device_id) const {
        std::lock_guard<std::mutex> lock(devices_mutex);
        return connected_devices.find(device_id) != connected_devices.end();
    }
    
    // Get device status
    HardwareWalletStatus getDeviceStatus(const std::string& device_id) const {
        std::lock_guard<std::mutex> lock(devices_mutex);
        
        auto it = connected_devices.find(device_id);
        if (it != connected_devices.end()) {
            return it->second.status;
        }
        return HardwareWalletStatus::DISCONNECTED;
    }
    
    // Export configuration
    std::map<HardwareWalletType, DeviceConfiguration> getAllConfigurations() const {
        return device_configs;
    }
};

} // namespace tigerwallet

#endif // TIGERWALLET_HARDWARE_WALLET_HPP
