#include "tiger_wallet.hpp"
#include <openssl/evp.h>
#include <openssl/ec.h>
#include <openssl/sha.h>
#include <openssl/ripemd.h>
#include <openssl/hmac.h>
#include <openssl/bn.h>
#include <openssl/rand.h>
#include <openssl/aes.h>
#include <sstream>
#include <iomanip>
#include <chrono>
#include <thread>
#include <random>
#include <algorithm>
#include <cstring>

namespace tiger {

// BIP39 word list (simplified - 2048 words)
static const std::vector<std::string> BIP39_WORD_LIST = {
    "abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract", "absurd", "abuse",
    "access", "accident", "account", "accuse", "achieve", "acid", "acoustic", "acquire", "across", "act",
    "action", "actor", "actress", "actual", "adapt", "add", "addict", "address", "adjust", "admit",
    "adult", "advance", "advice", "aerobic", "affair", "afford", "afraid", "again", "age", "agent",
    "agree", "ahead", "aim", "air", "airport", "aisle", "alarm", "album", "alcohol", "alert"
    // ... (complete 2048 words would be included in production)
};

// Entropy to mnemonic
std::vector<std::string> entropy_to_mnemonic(const std::vector<uint8_t>& entropy) {
    std::vector<std::string> words;
    
    // Calculate checksum
    uint8_t hash[32];
    SHA256(entropy.data(), entropy.size(), hash);
    
    size_t ent = entropy.size() * 8;
    size_t cs = ent / 32;
    size_t total = ent + cs;
    size_t indices = total / 11;
    
    std::vector<bool> bits;
    bits.reserve(total);
    
    for (size_t i = 0; i < entropy.size(); i++) {
        for (int j = 7; j >= 0; j--) {
            bits.push_back((entropy[i] >> j) & 1);
        }
    }
    
    for (size_t i = 0; i < cs; i++) {
        bits.push_back((hash[i / 8] >> (7 - (i % 8))) & 1);
    }
    
    for (size_t i = 0; i < indices; i++) {
        uint32_t index = 0;
        for (size_t j = 0; j < 11; j++) {
            index = (index << 1) | bits[i * 11 + j];
        }
        if (index < BIP39_WORD_LIST.size()) {
            words.push_back(BIP39_WORD_LIST[index]);
        }
    }
    
    return words;
}

// Mnemonic to entropy
std::vector<uint8_t> mnemonic_to_entropy(const std::vector<std::string>& words) {
    if (words.empty() || words.size() % 3 != 0) {
        return {};
    }
    
    std::vector<bool> bits;
    bits.reserve(words.size() * 11);
    
    for (const auto& word : words) {
        uint32_t index = 0;
        for (size_t i = 0; i < BIP39_WORD_LIST.size(); i++) {
            if (BIP39_WORD_LIST[i] == word) {
                index = static_cast<uint32_t>(i);
                break;
            }
        }
        
        for (int j = 10; j >= 0; j--) {
            bits.push_back((index >> j) & 1);
        }
    }
    
    size_t ent = (words.size() * 11) / 33 * 32;
    size_t cs = (words.size() * 11) - ent;
    
    std::vector<uint8_t> entropy;
    entropy.reserve(ent / 8);
    
    for (size_t i = 0; i < ent / 8; i++) {
        uint8_t b = 0;
        for (size_t j = 0; j < 8; j++) {
            b = (b << 1) | (bits[i * 8 + j] ? 1 : 0);
        }
        entropy.push_back(b);
    }
    
    // Verify checksum
    uint8_t hash[32];
    SHA256(entropy.data(), entropy.size(), hash);
    
    for (size_t i = 0; i < cs; i++) {
        bool expected = (hash[i / 8] >> (7 - (i % 8))) & 1;
        if (expected != bits[ent + i]) {
            return {};
        }
    }
    
    return entropy;
}

// Derive seed from mnemonic
std::vector<uint8_t> mnemonic_to_seed(const std::vector<std::string>& words, const std::string& passphrase = "") {
    std::string mnemonic;
    for (size_t i = 0; i < words.size(); i++) {
        if (i > 0) mnemonic += " ";
        mnemonic += words[i];
    }
    
    std::string salt = "mnemonic" + passphrase;
    
    std::vector<uint8_t> seed(64, 0);
    
    PKCS5_PBKDF2_HMAC(
        mnemonic.c_str(),
        mnemonic.size(),
        reinterpret_cast<const uint8_t*>(salt.c_str()),
        salt.size(),
        2048,
        EVP_sha512(),
        seed.data(),
        64
    );
    
    return seed;
}

// HD Key derivation
class HDKey {
public:
    std::vector<uint8_t> private_key;
    std::vector<uint8_t> chain_code;
    std::string public_key;
    std::string private_key_hex;
    std::string public_key_hex;
    
    static HDKey from_seed(const std::vector<uint8_t>& seed) {
        HDKey key;
        
        // HMAC-SHA512 with "Bitcoin seed"
        const char* prefix = "Bitcoin seed";
        uint8_t hmac[64];
        HMAC(EVP_sha512(), prefix, strlen(prefix), seed.data(), seed.size(), hmac, nullptr);
        
        key.private_key.assign(hmac, hmac + 32);
        key.chain_code.assign(hmac + 32, hmac + 64);
        
        // Derive public key
        key.public_key = derive_public_key(key.private_key);
        key.private_key_hex = bytes_to_hex(key.private_key);
        key.public_key_hex = bytes_to_hex(hex_to_bytes(key.public_key));
        
        return key;
    }
    
    HDKey derive(const std::string& path) {
        HDKey child;
        
        std::vector<uint32_t> indices = parse_path(path);
        
        HDKey current = *this;
        for (uint32_t idx : indices) {
            current = current.derive_child(idx);
        }
        
        return current;
    }
    
    HDKey derive_child(uint32_t index) {
        HDKey child;
        
        std::vector<uint8_t> data;
        
        // Hardened derivation
        if (index >= 0x80000000) {
            data.push_back(0);
            data.insert(data.end(), private_key.begin(), private_key.end());
        } else {
            std::vector<uint8_t> pub = hex_to_bytes(public_key);
            data.insert(data.end(), pub.begin(), pub.end());
        }
        
        // Append index
        data.push_back((index >> 24) & 0xFF);
        data.push_back((index >> 16) & 0xFF);
        data.push_back((index >> 8) & 0xFF);
        data.push_back(index & 0xFF);
        
        // HMAC-SHA512
        uint8_t hmac[64];
        HMAC(EVP_sha512(), chain_code.data(), chain_code.size(), 
             data.data(), data.size(), hmac, nullptr);
        
        // Parse IL
        BIGNUM* il = BN_bin2bn(hmac, 32, nullptr);
        BIGNUM* n = BN_new();
        EC_GROUP* group = EC_GROUP_new_by_curve_name(NID_secp256k1);
        EC_GROUP_get_order(group, n, nullptr);
        
        // Check if IL >= n
        if (BN_cmp(il, n) >= 0 || BN_is_zero(il)) {
            // Derivation failed, would need to try next index
            child.private_key = private_key;
            child.chain_code = chain_code;
            child.public_key = public_key;
            child.private_key_hex = private_key_hex;
            child.public_key_hex = public_key_hex;
        } else {
            // IL * G + IR
            EC_POINT* point = EC_POINT_new(group);
            EC_POINT_mul(group, point, il, nullptr, nullptr, nullptr);
            
            // Add parent public key
            std::vector<uint8_t> parent_pub = hex_to_bytes(public_key);
            if (!parent_pub.empty()) {
                EC_POINT* parent_point = EC_POINT_new(group);
                BN_CTX* ctx = BN_CTX_new();
                EC_POINT_oct2point(group, parent_point, parent_pub.data(), parent_pub.size(), ctx);
                EC_POINT_add(group, point, point, parent_point, ctx);
                BN_CTX_free(ctx);
                
                std::vector<uint8_t> result(65);
                EC_POINT_point2oct(group, point, POINT_CONVERSION_UNCOMPRESSED, 
                                   result.data(), result.size(), nullptr);
                child.public_key = bytes_to_hex(result);
            }
            
            EC_POINT_free(point);
            
            // IL + IR (private key addition)
            BIGNUM* ir = BN_bin2bn(hmac + 32, 32, nullptr);
            BIGNUM* priv = BN_bin2bn(private_key.data(), private_key.size(), nullptr);
            BN_mod_add(priv, priv, ir, n, nullptr);
            
            child.private_key.resize(32);
            BN_bn2bin(priv, child.private_key.data());
            
            child.chain_code.assign(hmac + 32, hmac + 64);
            child.private_key_hex = bytes_to_hex(child.private_key);
            child.public_key_hex = bytes_to_hex(hex_to_bytes(child.public_key));
            
            BN_free(il);
            BN_free(ir);
            BN_free(priv);
        }
        
        BN_free(n);
        EC_GROUP_free(group);
        
        return child;
    }
    
private:
    static std::string derive_public_key(const std::vector<uint8_t>& priv_key) {
        EC_KEY* key = EC_KEY_new_by_curve_name(NID_secp256k1);
        BIGNUM* priv = BN_bin2bn(priv_key.data(), priv_key.size(), nullptr);
        EC_KEY_set_private_key(key, priv);
        
        const EC_POINT* point = EC_KEY_get0_public_key(key);
        EC_GROUP* group = EC_KEY_get0_group(key);
        
        std::vector<uint8_t> pub(65);
        EC_POINT_point2oct(group, point, POINT_CONVERSION_UNCOMPRESSED, 
                          pub.data(), pub.size(), nullptr);
        
        BN_free(priv);
        EC_KEY_free(key);
        
        return bytes_to_hex(pub);
    }
    
    static std::vector<uint32_t> parse_path(const std::string& path) {
        std::vector<uint32_t> indices;
        
        std::stringstream ss(path);
        std::string segment;
        
        while (std::getline(ss, segment, '/')) {
            if (segment == "m") continue;
            
            bool hardened = segment.back() == '\'';
            if (hardened) segment.pop_back();
            
            uint32_t idx = std::stoi(segment);
            if (hardened) idx += 0x80000000;
            indices.push_back(idx);
        }
        
        return indices;
    }
    
    static std::string bytes_to_hex(const std::vector<uint8_t>& bytes) {
        std::stringstream ss;
        ss << std::hex << std::setfill('0');
        for (uint8_t b : bytes) {
            ss << std::setw(2) << static_cast<int>(b);
        }
        return ss.str();
    }
    
    static std::vector<uint8_t> hex_to_bytes(const std::string& hex) {
        std::vector<uint8_t> bytes;
        for (size_t i = 0; i < hex.size(); i += 2) {
            std::string byte_str = hex.substr(i, 2);
            uint8_t byte = static_cast<uint8_t>(std::stoi(byte_str, nullptr, 16));
            bytes.push_back(byte);
        }
        return bytes;
    }
};

// EVM address derivation from public key
std::string public_key_to_eth_address(const std::string& public_key_hex) {
    std::vector<uint8_t> pub = hex_to_bytes(public_key_hex);
    
    // Remove compression prefix if present
    if (pub.size() == 65 && pub[0] == 0x04) {
        pub.erase(pub.begin());
    }
    
    // Keccak256 hash
    uint8_t hash[32];
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_keccak256(), nullptr);
    EVP_DigestUpdate(ctx, pub.data(), pub.size());
    EVP_DigestFinal_ex(ctx, hash, nullptr);
    EVP_MD_CTX_free(ctx);
    
    // Last 20 bytes
    std::vector<uint8_t> address_bytes(hash + 12, hash + 32);
    
    // Add 0x prefix
    std::string address = "0x" + bytes_to_hex(address_bytes);
    
    return address;
}

// Generate random bytes
std::vector<uint8_t> generate_random_bytes(size_t count) {
    std::vector<uint8_t> bytes(count);
    RAND_bytes(bytes.data(), count);
    return bytes;
}

// Implementation class
class TigerWallet::Impl {
public:
    std::vector<std::string> mnemonic;
    std::vector<uint8_t> seed;
    HDKey master_key;
    std::map<ChainType, std::map<uint32_t, std::string>> addresses;
    std::map<ChainType, ChainConfig> chain_configs;
    std::map<std::string, TokenConfig> token_configs;
    std::map<ChainType, std::string> rpc_urls;
    std::map<ChainType, std::string> explorer_urls;
    std::map<ChainType, uint64_t> gas_limits;
    uint8_t slippage_tolerance = 50; // 0.5%
    ConfirmationCallback confirmation_callback;
    
    Impl() {
        initialize_default_chains();
    }
    
    void initialize_default_chains() {
        // EVM chains
        chain_configs[ChainType::ETHEREUM] = {
            ChainType::ETHEREUM, "Ethereum", "ETH", 1,
            "https://eth.llamarpc.com", "https://etherscan.io", "", 18, true, 21000, 12
        };
        chain_configs[ChainType::POLYGON] = {
            ChainType::POLYGON, "Polygon", "MATIC", 137,
            "https://polygon-rpc.com", "https://polygonscan.com", "", 18, true, 21000, 12
        };
        chain_configs[ChainType::ARBITRUM] = {
            ChainType::ARBITRUM, "Arbitrum", "ARB", 42161,
            "https://arb1.arbitrum.io/rpc", "https://arbiscan.io", "", 18, true, 21000, 12
        };
        chain_configs[ChainType::OPTIMISM] = {
            ChainType::OPTIMISM, "Optimism", "OP", 10,
            "https://mainnet.optimism.io", "https://optimistic.etherscan.io", "", 18, true, 21000, 12
        };
        chain_configs[ChainType::BASE] = {
            ChainType::BASE, "Base", "BASE", 8453,
            "https://mainnet.base.org", "https://basescan.org", "", 18, true, 21000, 12
        };
        chain_configs[ChainType::AVALANCHE] = {
            ChainType::AVALANCHE, "Avalanche", "AVAX", 43114,
            "https://api.avax.network/ext/bc/C/rpc", "https://snowtrace.io", "", 18, true, 21000, 12
        };
        chain_configs[ChainType::BINANCE_SMART_CHAIN] = {
            ChainType::BINANCE_SMART_CHAIN, "BNB Chain", "BNB", 56,
            "https://bsc-dataseed.binance.org", "https://bscscan.com", "", 18, true, 21000, 12
        };
        
        // Initialize RPC URLs
        for (const auto& [chain, config] : chain_configs) {
            rpc_urls[chain] = config.rpc_url;
            explorer_urls[chain] = config.explorer_url;
            gas_limits[chain] = config.gas_limit;
        }
        
        // Initialize default tokens
        initialize_default_tokens();
    }
    
    void initialize_default_tokens() {
        // Ethereum tokens
        token_configs["0x0000000000000000000000000000000000000000_1"] = {
            "0x0000000000000000000000000000000000000000", "ETH", "Ethereum",
            18, ChainType::ETHEREUM, TokenStandard::NATIVE, "", true, "", 0, true
        };
        token_configs["0xdAC17F958D2ee523a2206206994597C13D831ec7_1"] = {
            "0xdAC17F958D2ee523a2206206994597C13D831ec7", "USDT", "Tether USD",
            6, ChainType::ETHEREUM, TokenStandard::ERC20, "", true, "", 0, true
        };
        token_configs["0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48_1"] = {
            "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "USDC", "USD Coin",
            6, ChainType::ETHEREUM, TokenStandard::ERC20, "", true, "", 0, true
        };
        token_configs["0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599_1"] = {
            "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", "WBTC", "Wrapped Bitcoin",
            8, ChainType::ETHEREUM, TokenStandard::ERC20, "", true, "", 0, true
        };
        
        // BSC tokens
        token_configs["0x0000000000000000000000000000000000000000_56"] = {
            "0x0000000000000000000000000000000000000000", "BNB", "BNB",
            18, ChainType::BINANCE_SMART_CHAIN, TokenStandard::NATIVE, "", true, "", 0, true
        };
        token_configs["0x55d398326f99059fF775485246999027B3197955_56"] = {
            "0x55d398326f99059fF775485246999027B3197955", "USDT", "Tether USD",
            18, ChainType::BINANCE_SMART_CHAIN, TokenStandard::BEP20, "", true, "", 0, true
        };
    }
};

// Constructor
TigerWallet::TigerWallet() : pImpl(std::make_unique<Impl>()) {}

// Destructor
TigerWallet::~TigerWallet() = default;

// Generate mnemonic
Result<std::vector<std::string>> TigerWallet::generate_mnemonic() {
    try {
        // Generate 256-bit entropy (24 words)
        std::vector<uint8_t> entropy = generate_random_bytes(32);
        
        auto words = entropy_to_mnemonic(entropy);
        
        if (words.empty()) {
            return Result<std::vector<std::string>>::err(
                WalletError::MNEMONIC_INVALID,
                "Failed to generate mnemonic"
            );
        }
        
        pImpl->mnemonic = words;
        pImpl->seed = mnemonic_to_seed(words);
        pImpl->master_key = HDKey::from_seed(pImpl->seed);
        
        return Result<std::vector<std::string>>::ok(words);
    } catch (const std::exception& e) {
        return Result<std::vector<std::string>>::err(
            WalletError::UNKNOWN_ERROR,
            e.what()
        );
    }
}

// Recover from mnemonic
Result<void> TigerWallet::recover_from_mnemonic(const std::string& mnemonic, const std::string& password) {
    try {
        std::vector<std::string> words;
        std::stringstream ss(mnemonic);
        std::string word;
        
        while (ss >> word) {
            words.push_back(word);
        }
        
        if (words.size() != 12 && words.size() != 24) {
            return Result<void>::err(
                WalletError::MNEMONIC_INVALID,
                "Mnemonic must be 12 or 24 words"
            );
        }
        
        auto entropy = mnemonic_to_entropy(words);
        if (entropy.empty()) {
            return Result<void>::err(
                WalletError::MNEMONIC_INVALID,
                "Invalid mnemonic checksum"
            );
        }
        
        pImpl->mnemonic = words;
        pImpl->seed = mnemonic_to_seed(words, password);
        pImpl->master_key = HDKey::from_seed(pImpl->seed);
        
        // Derive addresses for default chains
        derive_addresses(ChainType::ETHEREUM, 5);
        derive_addresses(ChainType::BINANCE_SMART_CHAIN, 5);
        
        return Result<void>::ok();
    } catch (const std::exception& e) {
        return Result<void>::err(
            WalletError::UNKNOWN_ERROR,
            e.what()
        );
    }
}

// Recover from private key
Result<void> TigerWallet::recover_from_private_key(const std::string& private_key, ChainType chain) {
    try {
        auto key_bytes = hex_to_bytes(private_key);
        
        if (key_bytes.size() != 32) {
            return Result<void>::err(
                WalletError::INVALID_PARAMS,
                "Invalid private key length"
            );
        }
        
        // For EVM chains, derive address from private key
        if (chain == ChainType::ETHEREUM || chain == ChainType::POLYGON || 
            chain == ChainType::ARBITRUM || chain == ChainType::OPTIMISM ||
            chain == ChainType::BASE || chain == ChainType::AVALANCHE ||
            chain == ChainType::BINANCE_SMART_CHAIN) {
            
            EC_KEY* key = EC_KEY_new_by_curve_name(NID_secp256k1);
            BIGNUM* priv = BN_bin2bn(key_bytes.data(), key_bytes.size(), nullptr);
            EC_KEY_set_private_key(key, priv);
            
            const EC_POINT* point = EC_KEY_get0_public_key(key);
            EC_GROUP* group = EC_KEY_get0_group(key);
            
            std::vector<uint8_t> pub(65);
            EC_POINT_point2oct(group, point, POINT_CONVERSION_UNCOMPRESSED,
                              pub.data(), pub.size(), nullptr);
            
            std::string address = public_key_to_eth_address(bytes_to_hex(pub));
            
            BN_free(priv);
            EC_KEY_free(key);
            
            pImpl->addresses[chain][0] = address;
        }
        
        return Result<void>::ok();
    } catch (const std::exception& e) {
        return Result<void>::err(
            WalletError::UNKNOWN_ERROR,
            e.what()
        );
    }
}

// Get address
Result<std::string> TigerWallet::get_address(ChainType chain, uint32_t index) {
    auto it = pImpl->addresses.find(chain);
    if (it != pImpl->addresses.end()) {
        auto addr_it = it->second.find(index);
        if (addr_it != it->second.end()) {
            return Result<std::string>::ok(addr_it->second);
        }
    }
    
    // Derive if not exists
    auto derive_result = derive_addresses(chain, index + 1);
    if (!derive_result.success) {
        return Result<std::string>::err(derive_result.error, derive_result.error_message);
    }
    
    return Result<std::string>::ok(pImpl->addresses[chain][index]);
}

// Derive addresses
Result<void> TigerWallet::derive_addresses(ChainType chain, uint32_t count) {
    try {
        std::string path_prefix;
        uint32_t coin_type;
        
        switch (chain) {
            case ChainType::ETHEREUM:
            case ChainType::POLYGON:
            case ChainType::ARBITRUM:
            case ChainType::OPTIMISM:
            case ChainType::BASE:
            case ChainType::AVALANCHE:
            case ChainType::BINANCE_SMART_CHAIN:
                coin_type = 60; // Ethereum
                break;
            case ChainType::BITCOIN:
                coin_type = 0;
                break;
            case ChainType::SOLANA:
                coin_type = 501;
                break;
            case ChainType::TRON:
                coin_type = 195;
                break;
            case ChainType::COSMOS:
                coin_type = 118;
                break;
            default:
                coin_type = 60;
        }
        
        for (uint32_t i = 0; i < count; i++) {
            std::stringstream ss;
            ss << "m/44'/" << coin_type << "'/0'/0/" << i;
            
            auto child_key = pImpl->master_key.derive(ss.str());
            std::string address = public_key_to_eth_address(child_key.public_key_hex);
            
            pImpl->addresses[chain][i] = address;
        }
        
        return Result<void>::ok();
    } catch (const std::exception& e) {
        return Result<void>::err(
            WalletError::DERIVATION_FAILED,
            e.what()
        );
    }
}

// Get all addresses
Result<std::vector<AddressInfo>> TigerWallet::get_all_addresses() {
    std::vector<AddressInfo> result;
    
    for (const auto& [chain, addr_map] : pImpl->addresses) {
        for (const auto& [index, address] : addr_map) {
            AddressInfo info;
            info.address = address;
            info.chain = chain;
            info.derivation_path = "m/44'/60'/0'/0/" + std::to_string(index);
            info.is_derived = true;
            result.push_back(info);
        }
    }
    
    return Result<std::vector<AddressInfo>>::ok(result);
}

// Get native balance
Result<uint256_t> TigerWallet::get_native_balance(ChainType chain, const std::string& address) {
    // In production, this would query the RPC
    // For now, return placeholder
    return Result<uint256_t>::ok(0);
}

// Get token balances
Result<std::vector<TokenBalance>> TigerWallet::get_token_balances(ChainType chain, const std::string& address) {
    // In production, this would query token contracts
    return Result<std::vector<TokenBalance>>::ok({});
}

// Get full balance
Result<AddressInfo> TigerWallet::get_full_balance(ChainType chain, const std::string& address) {
    AddressInfo info;
    info.address = address;
    info.chain = chain;
    info.balance = 0;
    
    auto balance_result = get_native_balance(chain, address);
    if (balance_result.success) {
        info.balance = balance_result.data;
    }
    
    auto tokens_result = get_token_balances(chain, address);
    if (tokens_result.success) {
        info.tokens = tokens_result.data;
    }
    
    return Result<AddressInfo>::ok(info);
}

// Create transaction
Result<SignedTransaction> TigerWallet::create_transaction(const TransactionData& tx_data) {
    SignedTransaction tx;
    tx.data = tx_data;
    tx.timestamp = static_cast<uint64_t>(std::time(nullptr));
    tx.status = TxStatus::PENDING;
    
    return Result<SignedTransaction>::ok(tx);
}

// Sign transaction
Result<SignedTransaction> TigerWallet::sign_transaction(const TransactionData& tx_data) {
    auto tx_result = create_transaction(tx_data);
    if (!tx_result.success) {
        return Result<SignedTransaction>::err(tx_result.error, tx_result.error_message);
    }
    
    SignedTransaction signed_tx = tx_result.data;
    
    // Sign with private key
    // In production, this would properly sign the transaction
    
    return Result<SignedTransaction>::ok(signed_tx);
}

// Broadcast transaction
Result<std::string> TigerWallet::broadcast_transaction(const SignedTransaction& signed_tx) {
    // In production, this would broadcast to the network
    return Result<std::string>::ok(signed_tx.tx_hash);
}

// Get transaction status
Result<TxStatus> TigerWallet::get_transaction_status(const std::string& tx_hash, ChainType chain) {
    // In production, this would query the explorer
    return Result<TxStatus>::ok(TxStatus::PENDING);
}

// Get transaction history
Result<std::vector<SignedTransaction>> TigerWallet::get_transaction_history(
    const std::string& address,
    ChainType chain,
    uint32_t limit,
    uint32_t offset
) {
    // In production, this would query the database/explorer
    return Result<std::vector<SignedTransaction>>::ok({});
}

// Get swap quote
Result<SwapQuote> TigerWallet::get_swap_quote(
    const std::string& from_token,
    const std::string& to_token,
    uint256_t amount,
    ChainType chain
) {
    SwapQuote quote;
    quote.from_token = from_token;
    quote.to_token = to_token;
    quote.from_amount = amount;
    quote.to_amount = amount; // Simplified
    quote.price_impact = 0;
    quote.gas_estimate = 150000;
    quote.dex = "TigerSwap";
    quote.expires_at = static_cast<uint64_t>(std::time(nullptr)) + 300;
    
    return Result<SwapQuote>::ok(quote);
}

// Execute swap
Result<SignedTransaction> TigerWallet::execute_swap(
    const SwapQuote& quote,
    const std::string& from_address,
    uint256_t slippage_tolerance
) {
    TransactionData tx_data;
    tx_data.from = from_address;
    tx_data.value = quote.from_amount;
    tx_data.chain = ChainType::ETHEREUM;
    
    return sign_transaction(tx_data);
}

// Open perpetual position
Result<PerpetualPosition> TigerWallet::open_perpetual_position(
    const std::string& collateral_token,
    const std::string& index_token,
    bool is_long,
    uint256_t collateral_amount,
    uint64_t leverage
) {
    PerpetualPosition position;
    position.collateral_token = collateral_token;
    position.index_token = index_token;
    position.is_long = is_long;
    position.collateral = collateral_amount;
    position.leverage = leverage;
    position.size = collateral_amount * leverage;
    position.timestamp = static_cast<uint64_t>(std::time(nullptr));
    
    return Result<PerpetualPosition>::ok(position);
}

// Close perpetual position
Result<PerpetualPosition> TigerWallet::close_perpetual_position(const std::string& position_id) {
    return Result<PerpetualPosition>::ok({});
}

// Update perpetual position
Result<PerpetualPosition> TigerWallet::update_perpetual_position(
    const std::string& position_id,
    uint256_t new_collateral,
    uint64_t new_leverage
) {
    return Result<PerpetualPosition>::ok({});
}

// Get perpetual positions
Result<std::vector<PerpetualPosition>> TigerWallet::get_perpetual_positions(const std::string& trader) {
    return Result<std::vector<PerpetualPosition>>::ok({});
}

// Follow trader
Result<void> TigerWallet::follow_trader(const std::string& trader_address, uint64_t max_copy_amount) {
    return Result<void>::ok();
}

// Unfollow trader
Result<void> TigerWallet::unfollow_trader(const std::string& trader_address) {
    return Result<void>::ok();
}

// Get trading signals
Result<std::vector<TradingSignal>> TigerWallet::get_trading_signals(
    const std::string& trader_address,
    uint32_t limit
) {
    return Result<std::vector<TradingSignal>>::ok({});
}

// Execute copy trade
Result<void> TigerWallet::execute_copy_trade(const TradingSignal& signal, uint256_t amount) {
    return Result<void>::ok();
}

// Get top traders
Result<std::vector<TradingSignal>> TigerWallet::get_top_traders(uint32_t limit) {
    return Result<std::vector<TradingSignal>>::ok({});
}

// Add custom chain
Result<void> TigerWallet::add_custom_chain(const ChainConfig& config) {
    pImpl->chain_configs[config.type] = config;
    return Result<void>::ok();
}

// Remove chain
Result<void> TigerWallet::remove_chain(ChainType chain) {
    pImpl->chain_configs.erase(chain);
    return Result<void>::ok();
}

// Get supported chains
Result<std::vector<ChainConfig>> TigerWallet::get_supported_chains() {
    std::vector<ChainConfig> result;
    for (const auto& [chain, config] : pImpl->chain_configs) {
        result.push_back(config);
    }
    return Result<std::vector<ChainConfig>>::ok(result);
}

// Get chain config
Result<ChainConfig> TigerWallet::get_chain_config(ChainType chain) {
    auto it = pImpl->chain_configs.find(chain);
    if (it != pImpl->chain_configs.end()) {
        return Result<ChainConfig>::ok(it->second);
    }
    return Result<ChainConfig>::err(WalletError::CHAIN_NOT_SUPPORTED, "Chain not supported");
}

// Add custom token
Result<void> TigerWallet::add_custom_token(const TokenConfig& config) {
    std::string key = config.address + "_" + std::to_string(static_cast<uint32_t>(config.chain));
    pImpl->token_configs[key] = config;
    return Result<void>::ok();
}

// Remove token
Result<void> TigerWallet::remove_token(const std::string& address, ChainType chain) {
    std::string key = address + "_" + std::to_string(static_cast<uint32_t>(chain));
    pImpl->token_configs.erase(key);
    return Result<void>::ok();
}

// Get supported tokens
Result<std::vector<TokenConfig>> TigerWallet::get_supported_tokens(ChainType chain) {
    std::vector<TokenConfig> result;
    for (const auto& [key, config] : pImpl->token_configs) {
        if (config.chain == chain) {
            result.push_back(config);
        }
    }
    return Result<std::vector<TokenConfig>>::ok(result);
}

// Admin operations (simplified)
Result<void> TigerWallet::add_blockchain_admin(ChainType chain, const std::string& admin_address) {
    return Result<void>::ok();
}

Result<void> TigerWallet::remove_blockchain_admin(ChainType chain, const std::string& admin_address) {
    return Result<void>::ok();
}

Result<void> TigerWallet::add_token_admin(const std::string& token_address, ChainType chain, const std::string& admin_address) {
    return Result<void>::ok();
}

Result<void> TigerWallet::remove_token_admin(const std::string& token_address, ChainType chain, const std::string& admin_address) {
    return Result<void>::ok();
}

Result<bool> TigerWallet::is_super_admin(const std::string& address) {
    return Result<bool>::ok(false);
}

Result<void> TigerWallet::set_super_admin(const std::string& address, bool is_admin) {
    return Result<void>::ok();
}

// Settings
void TigerWallet::set_rpc_url(ChainType chain, const std::string& url) {
    pImpl->rpc_urls[chain] = url;
}

void TigerWallet::set_explorer_url(ChainType chain, const std::string& url) {
    pImpl->explorer_urls[chain] = url;
}

void TigerWallet::set_gas_limit(ChainType chain, uint64_t limit) {
    pImpl->gas_limits[chain] = limit;
}

void TigerWallet::set_slippage_tolerance(uint8_t tolerance) {
    pImpl->slippage_tolerance = tolerance;
}

void TigerWallet::set_confirmation_callback(ConfirmationCallback callback) {
    pImpl->confirmation_callback = callback;
}

// Helper function implementations
std::string DerivationPath::to_string() const {
    std::stringstream ss;
    ss << "m/" << purpose << "'/" << coin_type << "'/" << account << "'/" << change << "/" << address_index;
    return ss.str();
}

} // namespace tiger
