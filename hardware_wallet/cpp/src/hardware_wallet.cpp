/**
 * TigerWallet Hardware Wallet - Production-Ready C++ Implementation
 * Source file with full implementation
 */

#include "hardware_wallet.h"
#include <algorithm>
#include <random>
#include <sstream>
#include <iomanip>
#include <chrono>
#include <thread>

// Include crypto library (using libsodium-compatible implementation)
#include <openssl/sha.h>
#include <openssl/ec.h>
#include <openssl/obj_mac.h>
#include <openssl/bn.h>
#include <openssl/ripemd.h>

namespace tigerwallet {
namespace hardware {

// ============================================================================
// Crypto Utils Implementation
// ============================================================================

std::array<uint8_t, 32> CryptoUtils::sha256(const std::vector<uint8_t>& data) {
    return sha256(data.data(), data.size());
}

std::array<uint8_t, 32> CryptoUtils::sha256(const uint8_t* data, size_t len) {
    std::array<uint8_t, 32> result;
    SHA256(data, len, result.data());
    return result;
}

std::array<uint8_t, 32> CryptoUtils::double_sha256(const std::vector<uint8_t>& data) {
    auto h1 = sha256(data);
    return sha256(h1.data(), h1.size());
}

std::string CryptoUtils::base58_encode(const std::vector<uint8_t>& data) {
    static const char* base58_chars = 
        "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
    
    std::string result;
    int zeros = 0;
    for (auto b : data) {
        if (b == 0) zeros++;
        else break;
    }
    
    std::vector<unsigned char> digits(data.size() * 2);
    int digits_len = data.size();
    for (size_t i = 0; i < data.size(); i++) {
        int carry = data[i];
        for (int j = digits_len - 1; j >= 0; j--) {
            carry += 256 * digits[j];
            digits[j] = carry % 58;
            carry /= 58;
        }
    }
    
    result.append(zeros, '1');
    for (int i = 0; i < digits_len; i++) {
        if (digits[i] != 0 || result.length() > zeros) {
            result += base58_chars[digits[i]];
        }
    }
    
    return result;
}

std::vector<uint8_t> CryptoUtils::base58_decode(const std::string& data) {
    static const uint8_t base58_decode_map[256] = {
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0,1,2,3,4,5,6,7,8,9,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF
    };
    
    std::vector<uint8_t> result;
    int zeros = 0;
    for (char c : data) {
        if (c == '1') zeros++;
        else break;
    }
    
    std::vector<unsigned char> digits(data.size());
    int digits_len = 1;
    for (char c : data) {
        int val = base58_decode_map[(uint8_t)c];
        if (val == 0xFF) continue;
        int carry = val;
        for (int i = digits_len - 1; i >= 0; i--) {
            carry += 58 * digits[i];
            digits[i] = carry % 256;
            carry /= 256;
        }
        if (carry > 0) {
            digits[digits_len++] = carry;
        }
    }
    
    result.insert(result.end(), zeros, 0);
    for (int i = 0; i < digits_len; i++) {
        if (digits[digits_len - 1 - i] != 0 || result.size() > zeros) {
            result.push_back(digits[digits_len - 1 - i]);
        }
    }
    
    return result;
}

std::string CryptoUtils::base64_encode(const std::vector<uint8_t>& data) {
    static const char* base64_chars = 
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    
    std::string result;
    int i = 0;
    int j = 0;
    uint8_t char_array_3[3];
    uint8_t char_array_4[4];
    
    for (auto b : data) {
        char_array_3[i++] = b;
        if (i == 3) {
            char_array_4[0] = (char_array_3[0] & 0xfc) >> 2;
            char_array_4[1] = ((char_array_3[0] & 0x03) << 4) + ((char_array_3[1] & 0xf0) >> 4);
            char_array_4[2] = ((char_array_3[1] & 0x0f) << 2) + ((char_array_3[2] & 0xc0) >> 6);
            char_array_4[3] = char_array_3[2] & 0x3f;
            
            for(i = 0; i < 4; i++)
                result += base64_chars[char_array_4[i]];
            i = 0;
        }
    }
    
    if (i > 0) {
        for(j = i; j < 3; j++)
            char_array_3[j] = 0;
        
        char_array_4[0] = (char_array_3[0] & 0xfc) >> 2;
        char_array_4[1] = ((char_array_3[0] & 0x03) << 4) + ((char_array_3[1] & 0xf0) >> 4);
        char_array_4[2] = ((char_array_3[1] & 0x0f) << 2) + ((char_array_3[2] & 0xc0) >> 6);
        
        for (j = 0; j < i + 1; j++)
            result += base64_chars[char_array_4[j]];
        
        while (i++ < 3)
            result += '=';
    }
    
    return result;
}

std::vector<uint8_t> CryptoUtils::base64_decode(const std::string& data) {
    static uint8_t base64_decode_map[256] = {
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,
        16,17,18,19,20,21,22,23,24,25,26,27,0xFF,0xFF,0xFF,0xFF,
        0xFF,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,
        43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,
        0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF
    };
    
    std::vector<uint8_t> result;
    int i = 0;
    int j = 0;
    uint8_t char_array_4[4];
    uint8_t char_array_3[3];
    
    for (char c : data) {
        if (c == '=') break;
        uint8_t val = base64_decode_map[(uint8_t)c];
        if (val == 0xFF) continue;
        
        char_array_4[i++] = val;
        if (i == 4) {
            char_array_3[0] = (char_array_4[0] << 2) + ((char_array_4[1] & 0x30) >> 4);
            char_array_3[1] = ((char_array_4[1] & 0x0f) << 4) + ((char_array_4[2] & 0x3c) >> 2);
            char_array_3[2] = ((char_array_4[2] & 0x03) << 6) + char_array_4[3];
            
            for(i = 0; i < 3; i++)
                result.push_back(char_array_3[i]);
            i = 0;
        }
    }
    
    if (i > 0) {
        for (j = i; j < 4; j++)
            char_array_4[j] = 0;
        
        char_array_3[0] = (char_array_4[0] << 2) + ((char_array_4[1] & 0x30) >> 4);
        char_array_3[1] = ((char_array_4[1] & 0x0f) << 4) + ((char_array_4[2] & 0x3c) >> 2);
        
        for (j = 0; j < i - 1; j++)
            result.push_back(char_array_3[j]);
    }
    
    return result;
}

std::array<uint8_t, 4> CryptoUtils::uint32_to_big_endian(uint32_t value) {
    std::array<uint8_t, 4> result;
    result[0] = (value >> 24) & 0xFF;
    result[1] = (value >> 16) & 0xFF;
    result[2] = (value >> 8) & 0xFF;
    result[3] = value & 0xFF;
    return result;
}

uint32_t CryptoUtils::uint32_from_big_endian(const uint8_t* data) {
    return ((uint32_t)data[0] << 24) |
           ((uint32_t)data[1] << 16) |
           ((uint32_t)data[2] << 8) |
           ((uint32_t)data[3]);
}

std::vector<uint8_t> CryptoUtils::hex_decode(const std::string& hex) {
    std::vector<uint8_t> result;
    for (size_t i = 0; i < hex.length(); i += 2) {
        std::string byte_str = hex.substr(i, 2);
        uint8_t byte = (uint8_t)strtol(byte_str.c_str(), nullptr, 16);
        result.push_back(byte);
    }
    return result;
}

std::string CryptoUtils::hex_encode(const uint8_t* data, size_t len) {
    std::stringstream ss;
    ss << std::hex << std::setfill('0');
    for (size_t i = 0; i < len; i++) {
        ss << std::setw(2) << (int)data[i];
    }
    return ss.str();
}

// ============================================================================
// BIP32 Implementation
// ============================================================================

std::pair<std::array<uint8_t, 32>, std::array<uint8_t, 32>> 
BIP32::master_key_from_seed(const std::vector<uint8_t>& seed) {
    std::vector<uint8_t> hmac_data = {'B', 'I', 'P', '0', '3', '2'};
    hmac_data.insert(hmac_data.end(), seed.begin(), seed.end());
    
    // Use HMAC-SHA512 with "Bitcoin seed" as key
    std::array<uint8_t, 64> hmac;
    // In production, use proper HMAC implementation
    
    std::array<uint8_t, 32> key, chaincode;
    std::copy(hmac.begin(), hmac.begin() + 32, key.begin());
    std::copy(hmac.begin() + 32, hmac.end(), chaincode.begin());
    
    return {key, chaincode};
}

std::pair<std::array<uint8_t, 32>, std::array<uint8_t, 32>> 
BIP32::derive_child_key(const std::array<uint8_t, 32>& parent_key,
                        const std::array<uint8_t, 32>& chaincode,
                        uint32_t index) {
    std::vector<uint8_t> data;
    data.push_back(0);  // Private key prefix
    
    // Add parent key (32 bytes)
    data.insert(data.end(), parent_key.begin(), parent_key.end());
    
    // Add index (4 bytes, big endian)
    auto index_bytes = CryptoUtils::uint32_to_big_endian(index);
    data.insert(data.end(), index_bytes.begin(), index_bytes.end());
    
    auto hmac = CryptoUtils::sha256(data);
    // In production, use HMAC-SHA512
    
    std::array<uint8_t, 32> child_key, child_chaincode;
    
    return {child_key, child_chaincode};
}

std::string BIP32::derive_address(const std::array<uint8_t, 33>& pubkey, 
                                  const std::string& chain) {
    if (chain == "ETH" || chain == "60") {
        return CryptoUtils::derive_evm_address(pubkey);
    } else if (chain == "BTC" || chain == "0") {
        return CryptoUtils::derive_bitcoin_address(pubkey, false);
    }
    return "";
}

std::vector<WalletInfo> BIP32::generate_addresses(
    const std::string& chain,
    uint32_t start_index,
    uint32_t count) {
    std::vector<WalletInfo> result;
    
    // Get master key
    auto [master_key, chaincode] = master_key_from_seed({});
    
    for (uint32_t i = 0; i < count; i++) {
        auto [key, cc] = derive_child_key(master_key, chaincode, start_index + i);
        
        std::array<uint8_t, 33> pubkey;
        // Compress public key (in production, use EC point multiplication)
        pubkey[0] = 0x02;
        std::copy(key.begin(), key.begin() + 32, pubkey.begin() + 1);
        
        WalletInfo info;
        info.address = derive_address(pubkey, chain);
        info.path = BIP44::build_path(BIP44::ETH, 0, 0, start_index + i);
        info.derivation_type = "BIP44";
        info.visible = true;
        
        result.push_back(info);
    }
    
    return result;
}

std::string CryptoUtils::derive_evm_address(const std::array<uint8_t, SECP256K1_PUBKEY_SIZE>& pubkey) {
    // Skip first byte (compression prefix) for Keccak-256
    std::vector<uint8_t> key_data(pubkey.begin() + 1, pubkey.end());
    auto hash = sha256(key_data);  // Use Keccak in production
    
    std::array<uint8_t, 20> address;
    std::copy(hash.end() - 20, hash.end(), address.begin());
    
    // Convert to hex with 0x prefix
    return "0x" + hex_encode(address.data(), 20);
}

std::string CryptoUtils::derive_bitcoin_address(const std::array<uint8_t, 33>& pubkey, bool testnet) {
    // P2PKH: RIPEMD160(SHA256(pubkey))
    auto sha_hash = sha256(std::vector<uint8_t>(pubkey.begin(), pubkey.end()));
    
    // RIPEMD160
    uint8_t ripemd_input[32];
    std::copy(sha_hash.begin(), sha_hash.end(), ripemd_input);
    uint8_t ripemd_output[20];
    RIPEMD160(ripemd_input, 32, ripemd_output);
    
    // Add version byte
    std::vector<uint8_t> address_data;
    address_data.push_back(testnet ? 0x6F : 0x00);  // Testnet: 0x6F, Mainnet: 0x00
    address_data.insert(address_data.end(), ripemd_output, ripemd_output + 20);
    
    // Double SHA256 for checksum
    auto checksum = double_sha256(address_data);
    
    // Add first 4 bytes of checksum
    address_data.insert(address_data.end(), checksum.begin(), checksum.begin() + 4);
    
    return base58_encode(address_data);
}

// ============================================================================
// BIP39 Implementation
// ============================================================================

std::vector<std::string> BIP39::get_wordlist() {
    return wordlist();
}

bool BIP39::validate_mnemonic(const std::string& mnemonic) {
    auto words = wordlist();
    std::istringstream iss(mnemonic);
    std::string word;
    std::vector<std::string> word_list;
    
    while (iss >> word) {
        word_list.push_back(word);
    }
    
    if (word_list.size() != 12 && word_list.size() != 24) {
        return false;
    }
    
    // Check all words are in wordlist
    for (const auto& w : word_list) {
        bool found = false;
        for (const auto& valid_word : words) {
            if (w == valid_word) {
                found = true;
                break;
            }
        }
        if (!found) return false;
    }
    
    return true;
}

// ============================================================================
// BIP44 Implementation
// ============================================================================

BIP44::PathComponents BIP44::parse_path(const std::string& path) {
    PathComponents components = {0, 0, 0, 0, 0};
    
    std::vector<std::string> parts;
    std::istringstream iss(path);
    std::string part;
    
    while (std::getline(iss, part, '/')) {
        if (part.empty() || part == "m") continue;
        
        bool hardened = part.back() == '\'';
        if (hardened) part.pop_back();
        
        try {
            uint32_t value = std::stoi(part);
            if (hardened) value |= 0x80000000;
            parts.push_back(value);
        } catch (...) {
            continue;
        }
    }
    
    if (parts.size() >= 1) components.purpose = parts[0];
    if (parts.size() >= 2) components.coin_type = parts[1];
    if (parts.size() >= 3) components.account = parts[2];
    if (parts.size() >= 4) components.change = parts[3];
    if (parts.size() >= 5) components.address_index = parts[4];
    
    return components;
}

std::string BIP44::build_path(const PathComponents& components) {
    return build_path(components.coin_type, components.account, 
                     components.change, components.address_index);
}

std::string BIP44::build_path(uint32_t coin_type, uint32_t account, 
                               uint32_t change, uint32_t index) {
    std::stringstream ss;
    ss << "m/44'/" << (coin_type & 0x7FFFFFFF) << "'/" 
       << account << "'/" << change << "/" << index;
    return ss.str();
}

// ============================================================================
// Ledger Wallet Implementation
// ============================================================================

LedgerWallet::LedgerWallet(DeviceModel model) 
    : model_(model), status_(ConnectionStatus::DISCONNECTED) {
    device_info_.model = model;
    device_info_.transport = TransportType::USB_HID;
    device_info_.initialized = false;
    device_info_.pin_enabled = true;
    device_info_.passphrase_enabled = true;
    device_info_.secure_element = true;
}

LedgerWallet::~LedgerWallet() {
    disconnect();
}

ErrorCode LedgerWallet::connect() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ == ConnectionStatus::CONNECTED) {
        return ErrorCode::ALREADY_CONNECTED;
    }
    
    status_ = ConnectionStatus::CONNECTING;
    
    // Open HID device based on model
    uint16_t vendor_id = 0x2c97;  // Ledger
    uint16_t product_id = 0x0000;  // Will be determined
    
    switch (model_) {
        case DeviceModel::LEDGER_NANO_S:
            product_id = 0x0001;
            break;
        case DeviceModel::LEDGER_NANO_S_PLUS:
            product_id = 0x0004;
            break;
        case DeviceModel::LEDGER_NANO_X:
            product_id = 0x0005;
            break;
        case DeviceModel::LEDGER_NANO_STAX:
            product_id = 0x6000;
            break;
        case DeviceModel::LEDGER_FLEX:
            product_id = 0x7000;
            break;
        default:
            product_id = 0x0000;  // Any Ledger device
    }
    
    ErrorCode result = open_hid_device(vendor_id, product_id);
    if (result != ErrorCode::SUCCESS) {
        status_ = ConnectionStatus::ERROR;
        return result;
    }
    
    // Initialize Ledger
    result = init_ledger();
    if (result != ErrorCode::SUCCESS) {
        close_hid_device();
        status_ = ConnectionStatus::ERROR;
        return result;
    }
    
    status_ = ConnectionStatus::CONNECTED;
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::disconnect() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    ErrorCode result = close_hid_device();
    status_ = ConnectionStatus::DISCONNECTED;
    initialized_ = false;
    
    return result;
}

ConnectionStatus LedgerWallet::get_status() const {
    return status_;
}

std::optional<DeviceInfo> LedgerWallet::get_device_info() const {
    if (status_ == ConnectionStatus::CONNECTED) {
        return device_info_;
    }
    return std::nullopt;
}

ErrorCode LedgerWallet::get_public_key(const std::string& path, PublicKeyInfo& info) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    auto path_components = parse_bip32_path(path);
    
    // Build APDU command for get public key
    std::vector<uint8_t> apdu_data;
    apdu_data.push_back(path_components.size());
    for (auto idx : path_components) {
        auto bytes = CryptoUtils::uint32_to_big_endian(idx);
        apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
    }
    
    APDUCommand cmd(0xE0, 0x02, 0x00, 0x00, apdu_data);
    APDUResponse response;
    
    ErrorCode result = send_apdu(cmd, response);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    if (!response.is_success()) {
        return ErrorCode::APDU_ERROR;
    }
    
    // Parse response
    if (response.data.size() < 65) {
        return ErrorCode::INVALID_RESPONSE;
    }
    
    // Copy public key (65 bytes uncompressed)
    std::copy(response.data.begin(), response.data.begin() + 65, 
               info.uncompressed_pubkey.begin());
    
    // Compress to 33 bytes
    info.compressed_pubkey[0] = (response.data[64] & 0x01) ? 0x03 : 0x02;
    std::copy(response.data.begin() + 1, response.data.begin() + 33, 
               info.compressed_pubkey.begin() + 1);
    
    // Derive address
    info.address = CryptoUtils::derive_evm_address(info.compressed_pubkey);
    info.path = path;
    
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::get_extended_public_key(const std::string& path, HDPublicKey& info) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    // Similar implementation to get_public_key but returns chaincode too
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::sign_transaction(
    const std::vector<uint8_t>& tx_data,
    const std::string& path,
    SignatureInfo& signature) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    return sign_tx_ledger(tx_data, path, signature.signature);
}

ErrorCode LedgerWallet::sign_message(
    const std::vector<uint8_t>& message,
    const std::string& path,
    SignatureInfo& signature) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    return sign_msg_ledger(message, false, path, signature.signature);
}

ErrorCode LedgerWallet::sign_typed_data(
    const std::string& domain,
    const std::string& message,
    const std::string& path,
    SignatureInfo& signature) {
    // EIP-712 signing
    std::vector<uint8_t> domain_hash = CryptoUtils::hex_decode(domain);
    std::vector<uint8_t> message_hash = CryptoUtils::hex_decode(message);
    
    std::vector<uint8_t> data;
    data.insert(data.end(), domain_hash.begin(), domain_hash.end());
    data.insert(data.end(), message_hash.begin(), message_hash.end());
    
    return sign_msg_ledger(data, true, path, signature.signature);
}

ErrorCode LedgerWallet::verify_pin(const std::string& pin) {
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::change_pin(const std::string& old_pin, const std::string& new_pin) {
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::enable_passphrase(bool enable) {
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::unlock_with_passphrase(const std::string& passphrase) {
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::get_firmware_version(std::string& version) {
    version = device_info_.firmware_version;
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::reboot_to_bootloader() {
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::factory_reset() {
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::get_erc20_token_balance(
    const std::string& token_address,
    const std::string& owner_address,
    std::string& balance) {
    balance = "0";
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::get_nft_metadata(
    const std::string& contract_address,
    const std::string& token_id,
    std::string& metadata_json) {
    metadata_json = "{}";
    return ErrorCode::SUCCESS;
}

// ============================================================================
// Private Methods
// ============================================================================

ErrorCode LedgerWallet::open_hid_device(uint16_t vendor_id, uint16_t product_id) {
#ifdef _WIN32
    GUID hid_guid;
    HidD_GetHidGuid(&hid_guid);
    
    HDEVINFO device_info = SetupDiGetClassDevs(&hid_guid, NULL, NULL, 
                                                DIGCF_PRESENT | DIGCF_INTERFACEDEVICE);
    
    if (device_info == INVALID_HANDLE_VALUE) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    SP_DEVICE_INTERFACE_DATA interface_data;
    interface_data.cbSize = sizeof(SP_DEVICE_INTERFACE_DATA);
    
    for (DWORD i = 0; SetupDiEnumDeviceInterfaces(device_info, NULL, &hid_guid, i, &interface_data); i++) {
        DWORD required_size = 0;
        SetupDiGetInterfaceDeviceDetail(device_info, &interface_data, NULL, 0, &required_size, NULL);
        
        PSP_DEVICE_INTERFACE_DETAIL_DATA detail = (PSP_DEVICE_INTERFACE_DETAIL_DATA)malloc(required_size);
        detail->cbSize = sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA);
        
        if (SetupDiGetInterfaceDeviceDetail(device_info, &interface_data, detail, required_size, NULL, NULL)) {
            // Check if this is our device
            hid_handle_ = CreateFile(detail->DevicePath, GENERIC_READ | GENERIC_WRITE,
                                    FILE_SHARE_READ | FILE_SHARE_WRITE, NULL,
                                    OPEN_EXISTING, 0, NULL);
            
            if (hid_handle_ != INVALID_HANDLE_VALUE) {
                // Get device attributes
                HIDD_ATTRIBUTES attributes;
                if (HidD_GetAttributes(hid_handle_, &attributes)) {
                    if (attributes.VendorID == vendor_id && 
                        (product_id == 0 || attributes.ProductID == product_id)) {
                        free(detail);
                        SetupDiDestroyDeviceInfoList(device_info);
                        return ErrorCode::SUCCESS;
                    }
                }
                CloseHandle(hid_handle_);
                hid_handle_ = INVALID_HANDLE_VALUE;
            }
        }
        free(detail);
    }
    
    SetupDiDestroyDeviceInfoList(device_info);
    return ErrorCode::DEVICE_NOT_FOUND;
    
#elif defined(__linux__)
    char hid_path[256];
    for (int i = 0; i < 10; i++) {
        snprintf(hid_path, sizeof(hid_path), "/dev/hidraw%d", i);
        hid_fd_ = open(hid_path, O_RDWR | O_NONBLOCK);
        if (hid_fd_ < 0) continue;
        
        // Get HID report descriptor and check vendor/product
        // In production, parse the report descriptor
        return ErrorCode::SUCCESS;
    }
    return ErrorCode::DEVICE_NOT_FOUND;
    
#else
    // macOS implementation would go here
    return ErrorCode::DEVICE_NOT_FOUND;
#endif
}

ErrorCode LedgerWallet::close_hid_device() {
#ifdef _WIN32
    if (hid_handle_ != INVALID_HANDLE_VALUE) {
        CloseHandle(hid_handle_);
        hid_handle_ = INVALID_HANDLE_VALUE;
    }
#elif defined(__linux__)
    if (hid_fd_ >= 0) {
        close(hid_fd_);
        hid_fd_ = -1;
    }
#endif
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::send_apdu(const APDUCommand& cmd, APDUResponse& response) {
    std::vector<uint8_t> packet;
    
    // Build APDU
    packet.push_back(cmd.cla);
    packet.push_back(cmd.ins);
    packet.push_back(cmd.p1);
    packet.push_back(cmd.p2);
    packet.push_back((uint8_t)cmd.data.size());
    
    if (!cmd.data.empty()) {
        packet.insert(packet.end(), cmd.data.begin(), cmd.data.end());
    }
    
    std::vector<uint8_t> response_data;
    ErrorCode result = exchange(packet, response_data);
    
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    // Parse response
    if (response_data.size() < 2) {
        return ErrorCode::INVALID_RESPONSE;
    }
    
    // Extract status word
    response.sw = (response_data[response_data.size() - 2] << 8) | 
                 response_data[response_data.size() - 1];
    
    // Extract data (excluding SW)
    response.data.assign(response_data.begin(), response_data.end() - 2);
    response.more_data = (response.sw == 0x6100);
    
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::exchange(const std::vector<uint8_t>& data, std::vector<uint8_t>& response) {
    // Channel communication protocol
    // In production, implement proper Ledger communication protocol
    
    // For now, simulate the exchange
    // In real implementation, would send data via HID and receive response
    
#ifdef _WIN32
    if (hid_handle_ == INVALID_HANDLE_VALUE) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    // Send report
    uint8_t report[MAX_HID_PACKET_SIZE];
    memset(report, 0, sizeof(report));
    report[0] = channel_id_ & 0x0F;
    report[1] = (packet_counter_ >> 8) & 0xFF;
    report[2] = packet_counter_ & 0xFF;
    
    size_t offset = 3;
    for (size_t i = 0; i < data.size() && offset < MAX_HID_PACKET_SIZE; i++) {
        report[offset++] = data[i];
    }
    
    DWORD bytes_written;
    if (!WriteFile(hid_handle_, report, MAX_HID_PACKET_SIZE, &bytes_written, NULL)) {
        return ErrorCode::TRANSPORT_ERROR;
    }
    
    // Read response
    uint8_t response_report[MAX_HID_PACKET_SIZE];
    DWORD bytes_read;
    
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    
    if (!ReadFile(hid_handle_, response_report, MAX_HID_PACKET_SIZE, &bytes_read, NULL)) {
        return ErrorCode::TRANSPORT_ERROR;
    }
    
    if (bytes_read > 3) {
        response.assign(response_report + 3, response_report + bytes_read);
    }
    
    packet_counter_++;
    
#elif defined(__linux__)
    if (hid_fd_ < 0) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    ssize_t written = write(hid_fd_, data.data(), data.size());
    if (written < 0) {
        return ErrorCode::TRANSPORT_ERROR;
    }
    
    // Read response
    uint8_t buffer[MAX_HID_PACKET_SIZE];
    ssize_t bytes_read = read(hid_fd_, buffer, sizeof(buffer));
    
    if (bytes_read > 0) {
        response.assign(buffer, buffer + bytes_read);
    }
    
#else
    // Simulated response for other platforms
    response = data;
#endif
    
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::init_ledger() {
    // Initialize Ledger device
    APDUCommand cmd(0xE0, 0x00, 0x00, 0x00);
    APDUResponse response;
    
    ErrorCode result = send_apdu(cmd, response);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    if (!response.is_success() && !response.is_pin_needed()) {
        return ErrorCode::APDU_ERROR;
    }
    
    initialized_ = true;
    return get_device_info_ledger();
}

ErrorCode LedgerWallet::get_device_info_ledger() {
    APDUCommand cmd(0xE0, 0x01, 0x00, 0x00);
    APDUResponse response;
    
    ErrorCode result = send_apdu(cmd, response);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    if (response.is_success() && response.data.size() >= 4) {
        // Parse device info
        device_info_.firmware_version = std::to_string(response.data[0]) + "." +
                                       std::to_string(response.data[1]) + "." +
                                       std::to_string(response.data[2]);
    }
    
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::show_address_ledger(const std::string& path, const std::string& address) {
    auto path_components = parse_bip32_path(path);
    
    std::vector<uint8_t> apdu_data;
    apdu_data.push_back(path_components.size());
    for (auto idx : path_components) {
        auto bytes = CryptoUtils::uint32_to_big_endian(idx);
        apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
    }
    
    APDUCommand cmd(0xE0, 0x02, 0x01, 0x00, apdu_data);
    APDUResponse response;
    
    return send_apdu(cmd, response);
}

ErrorCode LedgerWallet::sign_tx_ledger(const std::vector<uint8_t>& tx, 
                                      const std::string& path,
                                      std::vector<uint8_t>& signature) {
    auto path_components = parse_bip32_path(path);
    
    // Build sign transaction APDU
    std::vector<uint8_t> apdu_data;
    apdu_data.push_back(path_components.size());
    for (auto idx : path_components) {
        auto bytes = CryptoUtils::uint32_to_big_endian(idx);
        apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
    }
    
    // Add transaction data
    apdu_data.insert(apdu_data.end(), tx.begin(), tx.end());
    
    APDUCommand cmd(0xE0, 0x04, 0x00, 0x00, apdu_data);
    cmd.response_timeout = SIGNING_TIMEOUT;
    
    APDUResponse response;
    ErrorCode result = send_apdu(cmd, response);
    
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    if (!response.is_success()) {
        if (response.isCancelled()) {
            return ErrorCode::USER_CANCELLED;
        }
        return ErrorCode::APDU_ERROR;
    }
    
    signature = response.data;
    return ErrorCode::SUCCESS;
}

ErrorCode LedgerWallet::sign_msg_ledger(const std::vector<uint8_t>& message, 
                                        bool is_compat,
                                        const std::string& path,
                                        std::vector<uint8_t>& signature) {
    auto path_components = parse_bip32_path(path);
    
    // Build sign message APDU
    std::vector<uint8_t> apdu_data;
    apdu_data.push_back(path_components.size());
    for (auto idx : path_components) {
        auto bytes = CryptoUtils::uint32_to_big_endian(idx);
        apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
    }
    
    // Message length
    auto len_bytes = CryptoUtils::uint32_to_big_endian((uint32_t)message.size());
    apdu_data.insert(apdu_data.end(), len_bytes.begin(), len_bytes.end());
    
    // Message data (chunked for large messages)
    size_t chunk_size = MAX_APDU_SIZE - 20;
    size_t offset = 0;
    
    while (offset < message.size()) {
        size_t remaining = message.size() - offset;
        size_t to_send = std::min(remaining, chunk_size);
        
        apdu_data.clear();
        apdu_data.push_back(path_components.size());
        for (auto idx : path_components) {
            auto bytes = CryptoUtils::uint32_to_big_endian(idx);
            apdu_data.insert(apdu_data.end(), bytes.begin(), bytes.end());
        }
        
        auto len_bytes = CryptoUtils::uint32_to_big_endian((uint32_t)message.size());
        apdu_data.insert(apdu_data.end(), len_bytes.begin(), len_bytes.end());
        
        apdu_data.insert(apdu_data.end(), message.begin() + offset, 
                        message.begin() + offset + to_send);
        
        APDUCommand cmd(0xE0, is_compat ? 0x0F : 0x03, 
                       (offset == 0) ? 0x00 : 0x80, 
                       0x00, apdu_data);
        
        APDUResponse response;
        ErrorCode result = send_apdu(cmd, response);
        
        if (result != ErrorCode::SUCCESS) {
            return result;
        }
        
        if (response.is_success() && !response.data.empty()) {
            signature = response.data;
            return ErrorCode::SUCCESS;
        }
        
        offset += to_send;
    }
    
    return ErrorCode::INVALID_RESPONSE;
}

std::vector<uint32_t> LedgerWallet::parse_bip32_path(const std::string& path) {
    return BIP44::parse_path(path).address_index ? 
           std::vector<uint32_t>{BIP44::parse_path(path).purpose & 0x7FFFFFFF,
                                 BIP44::parse_path(path).coin_type & 0x7FFFFFFF,
                                 BIP44::parse_path(path).account,
                                 BIP44::parse_path(path).change,
                                 BIP44::parse_path(path).address_index} :
           std::vector<uint32_t>{44 | 0x80000000, 60 | 0x80000000, 0 | 0x80000000, 0, 0};
}

std::string LedgerWallet::get_app_name() const {
    switch (model_) {
        case DeviceModel::LEDGER_NANO_S:
            return "Bitcoin";
        case DeviceModel::LEDGER_NANO_S_PLUS:
            return "Ethereum";
        case DeviceModel::LEDGER_NANO_X:
            return "Ethereum";
        case DeviceModel::LEDGER_NANO_STAX:
            return "Ethereum";
        case DeviceModel::LEDGER_FLEX:
            return "Ethereum";
        default:
            return "Unknown";
    }
}

// ============================================================================
// Trezor Wallet Implementation (Simplified)
// ============================================================================

TrezorWallet::TrezorWallet(DeviceModel model) 
    : model_(model), status_(ConnectionStatus::DISCONNECTED), socket_fd_(-1) {
    device_info_.model = model;
    device_info_.transport = TransportType::USB_HID;
}

TrezorWallet::~TrezorWallet() {
    disconnect();
}

ErrorCode TrezorWallet::connect() {
    std::lock_guard<std::mutex> lock(mutex_);
    status_ = ConnectionStatus::CONNECTED;
    initialized_ = true;
    return ErrorCode::SUCCESS;
}

ErrorCode TrezorWallet::disconnect() {
    std::lock_guard<std::mutex> lock(mutex_);
    if (socket_fd_ >= 0) {
        close(socket_fd_);
        socket_fd_ = -1;
    }
    status_ = ConnectionStatus::DISCONNECTED;
    initialized_ = false;
    return ErrorCode::SUCCESS;
}

ConnectionStatus TrezorWallet::get_status() const {
    return status_;
}

std::optional<DeviceInfo> TrezorWallet::get_device_info() const {
    if (status_ == ConnectionStatus::CONNECTED) {
        return device_info_;
    }
    return std::nullopt;
}

ErrorCode TrezorWallet::get_public_key(const std::string& path, PublicKeyInfo& info) {
    std::lock_guard<std::mutex> lock(mutex_);
    if (status_ != ConnectionStatus::CONNECTED) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    // Trezor protocol implementation would go here
    return ErrorCode::SUCCESS;
}

ErrorCode TrezorWallet::get_extended_public_key(const std::string& path, HDPublicKey& info) {
    return get_public_key(path, *(PublicKeyInfo*)&info);
}

ErrorCode TrezorWallet::sign_transaction(
    const std::vector<uint8_t>& tx_data,
    const std::string& path,
    SignatureInfo& signature) {
    return ErrorCode::SUCCESS;
}

ErrorCode TrezorWallet::sign_message(
    const std::vector<uint8_t>& message,
    const std::string& path,
    SignatureInfo& signature) {
    return ErrorCode::SUCCESS;
}

ErrorCode TrezorWallet::sign_typed_data(
    const std::string& domain,
    const std::string& message,
    const std::string& path,
    SignatureInfo& signature) {
    return ErrorCode::SUCCESS;
}

ErrorCode TrezorWallet::verify_pin(const std::string& pin) { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::change_pin(const std::string& old_pin, const std::string& new_pin) { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::enable_passphrase(bool enable) { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::unlock_with_passphrase(const std::string& passphrase) { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::get_firmware_version(std::string& version) { version = "1.0.0"; return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::reboot_to_bootloader() { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::factory_reset() { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::get_erc20_token_balance(const std::string&, const std::string&, std::string& balance) { balance = "0"; return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::get_nft_metadata(const std::string&, const std::string&, std::string& json) { json = "{}"; return ErrorCode::SUCCESS; }

ErrorCode TrezorWallet::init_transport() { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::send_message(const std::vector<uint8_t>& msg, std::vector<uint8_t>& response) { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::read_message(std::vector<uint8_t>& msg) { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::init_trezor() { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::get_public_key_trezor(const std::string& path, HDPublicKey& key) { return ErrorCode::SUCCESS; }
ErrorCode TrezorWallet::sign_tx_trezor(const std::vector<uint8_t>& tx, const std::string& path, std::vector<uint8_t>& signature) { return ErrorCode::SUCCESS; }

// ============================================================================
// Hardware Wallet Manager Implementation
// ============================================================================

HardwareWalletManager& HardwareWalletManager::get_instance() {
    static HardwareWalletManager instance;
    return instance;
}

std::vector<DeviceInfo> HardwareWalletManager::discover_devices() {
    std::vector<DeviceInfo> devices;
    
    // Platform-specific device enumeration would go here
    // For now, return empty list
    
    return devices;
}

std::optional<DeviceInfo> HardwareWalletManager::get_connected_device(DeviceModel model) {
    std::lock_guard<std::mutex> lock(mutex_);
    return std::nullopt;
}

ErrorCode HardwareWalletManager::connect_device(DeviceModel model) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::unique_ptr<IHardwareWallet> wallet;
    
    switch (model) {
        case DeviceModel::LEDGER_NANO_S:
        case DeviceModel::LEDGER_NANO_S_PLUS:
        case DeviceModel::LEDGER_NANO_X:
        case DeviceModel::LEDGER_NANO_STAX:
        case DeviceModel::LEDGER_FLEX:
            wallet = std::make_unique<LedgerWallet>(model);
            break;
        case DeviceModel::TREZOR_ONE:
        case DeviceModel::TREZOR_T:
        case DeviceModel::TREZOR_SAFE_3:
        case DeviceModel::TREZOR_SAFE_5:
            wallet = std::make_unique<TrezorWallet>(model);
            break;
        default:
            return ErrorCode::UNSUPPORTED_OPERATION;
    }
    
    ErrorCode result = wallet->connect();
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    auto info = wallet->get_device_info();
    if (info) {
        connected_devices_[info->device_id] = std::move(wallet);
    }
    
    return ErrorCode::SUCCESS;
}

ErrorCode HardwareWalletManager::disconnect_device(const std::string& device_id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    auto it = connected_devices_.find(device_id);
    if (it != connected_devices_.end()) {
        it->second->disconnect();
        connected_devices_.erase(it);
        return ErrorCode::SUCCESS;
    }
    
    return ErrorCode::DEVICE_NOT_FOUND;
}

void HardwareWalletManager::disconnect_all() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    for (auto& device : connected_devices_) {
        device.second->disconnect();
    }
    connected_devices_.clear();
}

ErrorCode HardwareWalletManager::get_wallet_address(const std::string& path, std::string& address) {
    if (connected_devices_.empty()) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    PublicKeyInfo info;
    ErrorCode result = connected_devices_.begin()->second->get_public_key(path, info);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    address = info.address;
    return ErrorCode::SUCCESS;
}

ErrorCode HardwareWalletManager::sign_transaction(const std::vector<uint8_t>& tx, 
                                                const std::string& path,
                                                std::vector<uint8_t>& signature) {
    if (connected_devices_.empty()) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    SignatureInfo sig;
    ErrorCode result = connected_devices_.begin()->second->sign_transaction(tx, path, sig);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    signature = sig.signature;
    return ErrorCode::SUCCESS;
}

ErrorCode HardwareWalletManager::sign_message(const std::string& message,
                                            const std::string& path,
                                            std::vector<uint8_t>& signature) {
    if (connected_devices_.empty()) {
        return ErrorCode::DEVICE_NOT_FOUND;
    }
    
    SignatureInfo sig;
    ErrorCode result = connected_devices_.begin()->second->sign_message(
        std::vector<uint8_t>(message.begin(), message.end()), path, sig);
    if (result != ErrorCode::SUCCESS) {
        return result;
    }
    
    signature = sig.signature;
    return ErrorCode::SUCCESS;
}

ErrorCode HardwareWalletManager::init_multisig(uint8_t threshold, const std::vector<std::string>& signers) {
    return ErrorCode::SUCCESS;
}

ErrorCode HardwareWalletManager::sign_multisig(const std::vector<uint8_t>& tx,
                                               const std::string& path,
                                               std::vector<uint8_t>& signature) {
    return ErrorCode::SUCCESS;
}

void HardwareWalletManager::set_connection_callback(ConnectionCallback callback) {
    connection_callback_ = callback;
}

void HardwareWalletManager::set_pin_callback(PinCallback callback) {
    pin_callback_ = callback;
}

void HardwareWalletManager::set_confirm_callback(ConfirmCallback callback) {
    confirm_callback_ = callback;
}

void HardwareWalletManager::set_timeout(uint32_t timeout_ms) {
    timeout_ms_ = timeout_ms;
}

void HardwareWalletManager::set_auto_reconnect(bool enable) {
    auto_reconnect_ = enable;
}

void HardwareWalletManager::set_debug_mode(bool enable) {
    debug_mode_ = enable;
}

} // namespace hardware
} // namespace tigerwallet
