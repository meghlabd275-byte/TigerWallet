/**
 * TigerWallet Crypto Core - Production-Ready Cryptographic Library
 * 
 * Implements real cryptographic operations with ultra-low latency:
 * - Ed25519 (Edwards-curve Digital Signature)
 * - ECDSA (Elliptic Curve Digital Signature)
 * - BIP-32/39/44 Key Derivation
 * - SHA-256/SHA-3/Keccak
 * - AES-256-GCM Encryption
 * - ChaCha20-Poly1305
 * 
 * Author: TigerWallet Team
 * License: MIT
 */

#ifndef TIGER_CRYPTO_H
#define TIGER_CRYPTO_H

#include <cstdint>
#include <cstddef>
#include <vector>
#include <string>
#include <array>
#include <memory>
#include <optional>

namespace tiger {
namespace crypto {

// ============================================================================
// Error Types
// ============================================================================

enum class CryptoError {
    Success = 0,
    InvalidKey,
    InvalidSignature,
    InvalidMnemonic,
    InvalidDerivationPath,
    EncryptionFailed,
    DecryptionFailed,
    RandomGenerationFailed,
    InvalidAddress,
    HashMismatch,
    BufferTooSmall,
    InvalidParameter
};

class CryptoException : public std::exception {
public:
    explicit CryptoException(CryptoError error, const std::string& message)
        : error_(error), message_(message) {}
    
    const char* what() const noexcept override {
        return message_.c_str();
    }
    
    CryptoError code() const { return error_; }
    
private:
    CryptoError error_;
    std::string message_;
};

// ============================================================================
// Secure Random Number Generation
// ============================================================================

class SecureRandom {
public:
    static std::vector<uint8_t> generate(size_t length);
    static uint32_t generate32();
    static uint64_t generate64();
    
private:
    SecureRandom() = default;
};

// ============================================================================
// SHA-256 Implementation
// ============================================================================

class Sha256 {
public:
    Sha256();
    void update(const uint8_t* data, size_t length);
    void update(const std::string& str);
    std::array<uint8_t, 32> finalize();
    std::string finalize_hex();
    
    static std::array<uint8_t, 32> hash(const uint8_t* data, size_t length);
    static std::array<uint8_t, 32> hash(const std::string& str);
    static std::string hash_hex(const uint8_t* data, size_t length);
    static std::string hash_hex(const std::string& str);
    
private:
    uint64_t length_;
    std::array<uint32_t, 8> state_;
    std::vector<uint8_t> buffer_;
};

// ============================================================================
// SHA-3 / Keccak Implementation
// ============================================================================

class Sha3_256 {
public:
    Sha3_256();
    void update(const uint8_t* data, size_t length);
    std::array<uint8_t, 32> finalize();
    
    static std::array<uint8_t, 32> hash(const uint8_t* data, size_t length);
    
private:
    void keccakf(std::array<uint64_t, 25>& state);
    
    std::array<uint64_t, 25> state_;
    size_t rate_;
    size_t capacity_;
    size_t input_offset_;
};

class Keccak256 {
public:
    Keccak256();
    void update(const uint8_t* data, size_t length);
    std::array<uint8_t, 32> finalize();
    
    static std::array<uint8_t, 32> hash(const uint8_t* data, size_t length);
    static std::string hash_hex(const uint8_t* data, size_t length);
    
private:
    Sha3_256 hasher_;
    bool finalized_;
};

// ============================================================================
// HMAC Implementation
// ============================================================================

class HmacSha256 {
public:
    HmacSha256(const uint8_t* key, size_t key_length);
    void update(const uint8_t* data, size_t length);
    std::array<uint8_t, 32> finalize();
    
    static std::array<uint8_t, 32> compute(const uint8_t* key, size_t key_len,
                                             const uint8_t* data, size_t data_len);
    
private:
    std::array<uint8_t, 64> o_key_pad_;
    std::array<uint8_t, 64> i_key_pad_;
    Sha256 inner_hasher_;
};

// ============================================================================
// PBKDF2 Key Derivation
// ============================================================================

class Pbkdf2 {
public:
    static std::vector<uint8_t> derive(const uint8_t* password, size_t password_len,
                                        const uint8_t* salt, size_t salt_len,
                                        uint32_t iterations,
                                        size_t key_length);
    
    static std::vector<uint8_t> derive(const std::string& password,
                                        const std::string& salt,
                                        uint32_t iterations,
                                        size_t key_length);
};

// ============================================================================
// Ed25519 Digital Signature (Pure C++ Implementation)
// ============================================================================

class Ed25519KeyPair {
public:
    std::array<uint8_t, 32> private_key;
    std::array<uint8_t, 32> public_key;
};

class Ed25519 {
public:
    static Ed25519KeyPair generate_keypair();
    static Ed25519KeyPair keypair_from_seed(const uint8_t* seed, size_t seed_len);
    
    static std::array<uint8_t, 64> sign(const uint8_t* message, size_t message_len,
                                          const uint8_t* private_key);
    static std::array<uint8_t, 64> sign(const std::string& message,
                                          const std::array<uint8_t, 32>& private_key);
    
    static bool verify(const uint8_t* message, size_t message_len,
                       const uint8_t* signature,
                       const uint8_t* public_key);
    static bool verify(const std::string& message,
                       const std::array<uint8_t, 64>& signature,
                       const std::array<uint8_t, 32>& public_key);
    
    // Edwards curve operations (for internal use)
private:
    static void scalar_multiply(const uint8_t* scalar, const uint8_t* point, uint8_t* result);
    static void point_add(const uint8_t* p1, const uint8_t* p2, uint8_t* result);
};

// ============================================================================
// secp256k1 ECDSA (Bitcoin/Ethereum signature)
// ============================================================================

class Secp256k1KeyPair {
public:
    std::array<uint8_t, 32> private_key;
    std::array<uint8_t, 33> public_key;  // Compressed format
};

class EcdsaSecp256k1 {
public:
    static Secp256k1KeyPair generate_keypair();
    static Secp256k1KeyPair keypair_from_seed(const uint8_t* seed, size_t seed_len);
    
    // Sign with RFC6979 deterministic k
    static std::array<uint8_t, 64> sign(const uint8_t* message, size_t message_len,
                                          const uint8_t* private_key);
    static std::array<uint8_t, 64> sign(const std::string& message,
                                          const std::array<uint8_t, 32>& private_key);
    
    static bool verify(const uint8_t* message, size_t message_len,
                       const uint8_t* signature,
                       const uint8_t* public_key);
    static bool verify(const std::string& message,
                       const std::array<uint8_t, 64>& signature,
                       const std::array<uint8_t, 33>& public_key);
    
    // Ethereum address from public key
    static std::array<uint8_t, 20> public_key_to_eth_address(const uint8_t* public_key);
    static std::string eth_address_to_string(const std::array<uint8_t, 20>& address);
};

// ============================================================================
// AES-256-GCM Encryption
// ============================================================================

class Aes256Gcm {
public:
    static const size_t KEY_SIZE = 32;
    static const size_t IV_SIZE = 12;
    static const size_t TAG_SIZE = 16;
    
    Aes256Gcm(const uint8_t* key);
    ~Aes256Gcm();
    
    // Encrypt plaintext, output includes IV + ciphertext + tag
    std::vector<uint8_t> encrypt(const uint8_t* plaintext, size_t plaintext_len,
                                  const uint8_t* aad, size_t aad_len);
    
    // Decrypt (input includes IV + ciphertext + tag)
    std::vector<uint8_t> decrypt(const uint8_t* ciphertext, size_t ciphertext_len,
                                  const uint8_t* aad, size_t aad_len);
    
    static std::vector<uint8_t> encrypt(const uint8_t* key, const uint8_t* plaintext, 
                                         size_t plaintext_len);
    static std::vector<uint8_t> decrypt(const uint8_t* key, const uint8_t* ciphertext,
                                         size_t ciphertext_len);
    
private:
    void gcm_init(const uint8_t* key);
    void gcm_crypt(const uint8_t* input, uint8_t* output, size_t length,
                   const uint8_t* iv, size_t iv_len, const uint8_t* aad, size_t aad_len,
                   uint8_t* tag);
    
    std::array<uint8_t, 32> key_;
    std::array<uint8_t, 16> h_;
    std::array<uint8_t, 12> j0_;
};

// ============================================================================
// ChaCha20-Poly1305
// ============================================================================

class ChaCha20Poly1305 {
public:
    static const size_t KEY_SIZE = 32;
    static const size_t NONCE_SIZE = 12;
    static const size_t TAG_SIZE = 16;
    
    static std::vector<uint8_t> encrypt(const uint8_t* key, const uint8_t* nonce,
                                          const uint8_t* plaintext, size_t plaintext_len,
                                          const uint8_t* aad, size_t aad_len);
    
    static std::optional<std::vector<uint8_t>> decrypt(const uint8_t* key, const uint8_t* nonce,
                                                          const uint8_t* ciphertext, size_t ciphertext_len,
                                                          const uint8_t* aad, size_t aad_len);
};

// ============================================================================
// Base58 Encoding
// ============================================================================

class Base58 {
public:
    static std::string encode(const uint8_t* data, size_t length);
    static std::string encode(const std::vector<uint8_t>& data);
    static std::vector<uint8_t> decode(const std::string& encoded);
    
    // Bitcoin-specific with checksum
    static std::string encode_check(const uint8_t* data, size_t length);
    static std::vector<uint8_t> decode_check(const std::string& encoded);
};

// ============================================================================
// Base32 Encoding
// ============================================================================

class Base32 {
public:
    static std::string encode(const uint8_t* data, size_t length);
    static std::vector<uint8_t> decode(const std::string& encoded);
};

// ============================================================================
// RIPEMD-160
// ============================================================================

class Ripemd160 {
public:
    Ripemd160();
    void update(const uint8_t* data, size_t length);
    std::array<uint8_t, 20> finalize();
    
    static std::array<uint8_t, 20> hash(const uint8_t* data, size_t length);
    
private:
    uint64_t length_;
    std::array<uint32_t, 5> h_;
    std::vector<uint8_t> buffer_;
};

// ============================================================================
// Bitcoin Address Generation
// ============================================================================

class BitcoinAddress {
public:
    enum class Type {
        P2PKH,      // Legacy
        P2SH,       // Pay to Script Hash
        P2WPKH,     // Native SegWit
        P2WSH       // SegWit Script Hash
    };
    
    static std::string create_p2pkh(const uint8_t* public_key, bool mainnet = true);
    static std::string create_p2sh(const uint8_t* script_hash, bool mainnet = true);
    static std::string create_p2wpkh(const uint8_t* public_key, bool mainnet = true);
    static std::string create_p2wsh(const uint8_t* script_hash, bool mainnet = true);
    
    // From WIF (Wallet Import Format)
    static std::optional<std::vector<uint8_t>> wif_to_private_key(const std::string& wif);
    static std::string private_key_to_wif(const uint8_t* private_key, bool mainnet = true, bool compressed = true);
    
    // Validate address
    static bool validate(const std::string& address);
    static Type detect_type(const std::string& address);
};

// ============================================================================
// BIP-39 Mnemonic Phrase
// ============================================================================

class Bip39Mnemonic {
public:
    static const std::vector<std::string>& get_wordlist();
    
    // Generate mnemonic from random entropy
    static std::string generate(uint32_t word_count = 24);
    
    // Validate mnemonic
    static bool validate(const std::string& mnemonic);
    static bool validate(const std::string& mnemonic, std::string& error);
    
    // Convert mnemonic to seed (with optional passphrase)
    static std::array<uint8_t, 64> mnemonic_to_seed(const std::string& mnemonic,
                                                      const std::string& passphrase = "");
    
    // Get entropy from mnemonic
    static std::vector<uint8_t> mnemonic_to_entropy(const std::string& mnemonic);
    
    // Create mnemonic from entropy
    static std::string entropy_to_mnemonic(const std::vector<uint8_t>& entropy);
    
private:
    static bool is_word_valid(const std::string& word);
    static std::vector<uint8_t> compute_checksum(const std::vector<uint8_t>& entropy);
};

// ============================================================================
// BIP-32 HD Key Derivation
// ============================================================================

class Bip32Path {
public:
    Bip32Path() : hardened_count_(0) {}
    
    static Bip32Path parse(const std::string& path);
    static Bip32Path from_string(const std::string& path);
    
    Bip32Path& add(uint32_t index, bool hardened = false);
    
    std::string to_string() const;
    std::vector<uint8_t> to_bytes() const;
    
    uint32_t hardened_count() const { return hardened_count_; }
    bool is_hardened(uint32_t index) const;
    
private:
    std::vector<std::pair<uint32_t, bool>> path_;  // (index, hardened)
    uint32_t hardened_count_;
};

class Bip32Key {
public:
    std::array<uint8_t, 32> key;       // Private key or public key
    std::array<uint8_t, 32> chain_code;
    std::vector<uint8_t> public_key;   // Derived public key (for private key derivation)
    uint32_t depth;
    uint32_t child_number;
    std::array<uint8_t, 4> parent_fingerprint;
    
    bool is_private() const { return key[0] != 0x00; }
    bool is_public() const { return key[0] == 0x00; }
};

class Bip32 {
public:
    // Master key from seed
    static Bip32Key master_key(const uint8_t* seed, size_t seed_len);
    static Bip32Key master_key(const std::array<uint8_t, 64>& seed);
    
    // Derive child key
    static Bip32Key derive(const Bip32Key& parent, uint32_t index);
    static Bip32Key derive(const Bip32Key& parent, const Bip32Path& path);
    static Bip32Key derive(const Bip32Key& parent, const std::string& path);
    
    // Neutered key (public key only)
    static Bip32Key neuter(const Bip32Key& private_key);
    
    // Get Ethereum address from key
    static std::string eth_address(const Bip32Key& key);
    static std::array<uint8_t, 20> eth_address_bytes(const Bip32Key& key);
    
    // Get Bitcoin address from key
    static std::string btc_address(const Bip32Key& key, BitcoinAddress::Type type = BitcoinAddress::Type::P2WPKH);
    
    // Get Solana address from key
    static std::string sol_address(const Bip32Key& key);
};

// ============================================================================
// BIP-44 Multi-Account Hierarchy
// ============================================================================

namespace bip44 {

enum class CoinType : uint32_t {
    Bitcoin = 0,
    Litecoin = 2,
    Dogecoin = 3,
    Ethereum = 60,
    Polygon = 966,
    BNBChain = 714,
    Arbitrum = 11021,
    Optimism = 10,
    AvalancheC = 9000,
    Solana = 501,
    Aptos = 637,
    Sui = 784,
    Ton = 607,
    Near = 397,
    Cosmos = 118,
    Algorand = 283,
    Hedera = 3030,
    Starknet = 8864,
    ZkSync = 324,
    NearProtocol = 397,
    MultiVerse = 10000
};

struct Purpose {
    uint32_t value;
    explicit Purpose(uint32_t v) : value(v) {}
};

struct CoinType_ {
    uint32_t value;
    explicit CoinType_(CoinType v) : value(static_cast<uint32_t>(v)) {}
    explicit CoinType_(uint32_t v) : value(v) {}
};

struct Account {
    uint32_t value;
    explicit Account(uint32_t v) : value(v) {}
};

struct Change {
    uint32_t value;  // 0 = external, 1 = internal
    explicit Change(uint32_t v) : value(v) {}
};

struct AddressIndex {
    uint32_t value;
    explicit AddressIndex(uint32_t v) : value(v) {}
};

class Path {
public:
    Path(Purpose p, CoinType_ c, Account a, Change ch, AddressIndex i)
        : purpose_(p.value), coin_(c.value), account_(a.value), 
          change_(ch.value), index_(i.value) {}
    
    static Path eth_default() { return Path(Purpose(44), CoinType_(CoinType::Ethereum), Account(0), Change(0), AddressIndex(0)); }
    static Path btc_default() { return Path(Purpose(44), CoinType_(CoinType::Bitcoin), Account(0), Change(0), AddressIndex(0)); }
    static Path sol_default() { return Path(Purpose(44), CoinType_(CoinType::Solana), Account(0), Change(0), AddressIndex(0)); }
    
    std::string to_string() const;
    std::vector<uint8_t> to_bytes() const;
    
private:
    uint32_t purpose_;
    uint32_t coin_;
    uint32_t account_;
    uint32_t change_;
    uint32_t index_;
};

} // namespace bip44

// ============================================================================
// BIP-85 Deterministic Entropy
// ============================================================================

class Bip85 {
public:
    // Derive entropy for different applications
    static std::array<uint8_t, 32> derive(const Bip32Key& master_key, 
                                            uint32_t application,
                                            uint32_t index);
    
    // Applications
    static constexpr uint32_t APP_BTC = 0;
    static constexpr uint32_t APP_ETH = 1;
    static constexpr uint32_t APP_SLIP21 = 21;
    static constexpr uint32_t APP_HEX = 0x54584700; // "HEX"
    static constexpr uint32_t APP_PASSWORD = 0x50415353; // "PASS"
};

// ============================================================================
// Address Formatting
// ============================================================================

class Address {
public:
    // Ethereum
    static std::string eth_from_public_key(const uint8_t* public_key);
    static std::string eth_from_private_key(const uint8_t* private_key);
    static bool is_valid_eth(const std::string& address);
    
    // Bitcoin
    static std::string btc_from_public_key(const uint8_t* public_key, BitcoinAddress::Type type = BitcoinAddress::Type::P2WPKH);
    static bool is_valid_btc(const std::string& address);
    
    // Solana
    static std::string sol_from_public_key(const uint8_t* public_key);
    static bool is_valid_sol(const std::string& address);
    
    // TRON
    static std::string tron_from_eth_address(const std::string& eth_address);
    static bool is_valid_tron(const std::string& address);
    
    // Aptos
    static std::string apt_from_public_key(const uint8_t* public_key);
    static bool is_valid_apt(const std::string& address);
    
    // Sui
    static std::string sui_from_public_key(const uint8_t* public_key);
    static bool is_valid_sui(const std::string& address);
    
    // TON
    static std::string ton_from_public_key(const uint8_t* public_key);
    static bool is_valid_ton(const std::string& address);
};

// ============================================================================
// Utilities
// ============================================================================

namespace utils {

std::string to_hex(const uint8_t* data, size_t length);
std::string to_hex(const std::vector<uint8_t>& data);
std::vector<uint8_t> from_hex(const std::string& hex);

std::string to_base64(const uint8_t* data, size_t length);
std::vector<uint8_t> from_base64(const std::string& base64);

void secure_zero(void* ptr, size_t length);

} // namespace utils

} // namespace crypto
} // namespace tiger

#endif // TIGER_CRYPTO_H
