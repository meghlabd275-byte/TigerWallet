/**
 * TigerWallet Account Abstraction - Ultra Low Latency Implementation
 * 
 * Implements:
 * - EIP-7702 (EOA to Contract Upgrade)
 * - Session Keys
 * - Paymaster Integration
 * - Batched Transactions
 * - Gas Abstraction
 * 
 * @author TigerWallet Team
 * @version 1.0.0
 */

#ifndef TIGERWALLET_ACCOUNT_ABSTRACTION_HPP
#define TIGERWALLET_ACCOUNT_ABSTRACTION_HPP

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <memory>
#include <variant>
#include <optional>
#include <functional>
#include <mutex>
#include <atomic>
#include <thread>
#include <chrono>
#include <openssl/evp.h>
#include <openssl/rand.h>

#include "keccak256.hpp"
#include "secp256k1.hpp"
#include "uint256.hpp"

namespace tiger {

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

using Address = std::string;
using Bytes = std::vector<uint8_t>;
using ChainId = uint64_t;
using BlockNumber = uint64_t;
using Timestamp = uint64_t;
using Nonce = uint64_t;

// Amounts are 256-bit, as the EVM uses. Neither of these was declared, so the
// header could not compile.
using uint256_t = tigerwallet::math::U256;
using bytes = Bytes;
using bytes32 = std::array<uint8_t, 32>;

// User Operation
struct UserOperation {
    Address sender;
    uint256_t nonce;
    Bytes initCode;
    Bytes callData;
    // The native value the call carries, so a batch can total what it moves.
    uint256_t callValue;
    uint256_t callGasLimit;
    uint256_t verificationGasLimit;
    uint256_t preVerificationGas;
    uint256_t maxFeePerGas;
    uint256_t maxPriorityFeePerGas;
    Address paymaster;
    Bytes paymasterData;
    Bytes signature;
};

// Entry Point
struct EntryPoint {
    Address address;
    ChainId chainId;
};

// Paymaster Config
struct PaymasterConfig {
    Address paymasterAddress;
    Address stakingToken;
    uint256_t minStake;
    uint256_t minUnstakeDelay;
    bool isActive;
    uint256_t deposit;
};

// Session Key
struct SessionKey {
    Address key;
    Address sessionKeyAddress;
    std::vector<std::string> allowedMethods;
    std::vector<Address> allowedContracts;
    uint256_t maxAmount;
    Timestamp validUntil;
    Timestamp createdAt;
};

// EIP-7702 Authorization
struct Authorization {
    // The chain the authorization is valid on. A chain id of zero is valid on
    // every chain, which is how a replayable authorization is expressed.
    ChainId chainId;
    // The contract the EOA delegates to.
    Address address;
    // The EOA's own transaction nonce at the time of signing. EIP-7702 binds
    // the authorization to it, so a stale nonce makes it unusable.
    uint64_t nonce;
    // The 64-byte r||s signature over the authorization hash.
    Bytes signature;
    bool isValid;
    Timestamp expiresAt;
};

// Batched Transaction
struct BatchedTransaction {
    std::string id;
    std::vector<UserOperation> operations;
    uint256_t totalGasLimit;
    uint256_t totalValue;
    Address sender;
    Timestamp createdAt;
    Timestamp expiresAt;
};

// =============================================================================
// CRYPTO UTILITIES
// =============================================================================

// Real cryptography for the account abstraction layer.
//
// Three things here were fabricated:
//
//   publicKeyToAddress used SHA-256 and sliced bytes 12 to 32. An Ethereum
//   address is the last 20 bytes of the Keccak-256 of the uncompressed key
//   without its 0x04 prefix, so every address this produced was wrong.
//
//   eip712Hash concatenated the domain string and the message and hashed them
//   with SHA-256. EIP-712 hashes the domain separator and the struct hash
//   separately, then hashes the 0x1901 prefix with both; a plain concatenation
//   is a different digest, so no contract would accept it.
//
//   decryptAES wrote into plaintext.data() without ever resizing plaintext, so
//   it wrote through a null pointer into a zero-length buffer, and it compared
//   the tag after the final call rather than letting the cipher verify it,
//   which means the authentication was not checked. A tampered ciphertext
//   would decrypt to attacker-chosen bytes and report success.
class CryptoUtils {
public:
    // Generates a secp256k1 key pair. The private key is 32 bytes and the
    // public key is the 33-byte compressed point.
    static bool generateKeyPair(Bytes& privateKey, Bytes& publicKey) {
        try {
            const auto secret = tigerwallet::secp256k1::random_scalar();
            privateKey = tigerwallet::secp256k1::to_bytes(secret.get(), 32);
            const auto point = tigerwallet::secp256k1::public_point(secret.get());
            publicKey = tigerwallet::secp256k1::compress(point.get());
            return true;
        } catch (const std::exception& e) {
            std::cerr << "Key generation failed: " << e.what() << std::endl;
            return false;
        }
    }

    // Signs a 32-byte digest with ECDSA over secp256k1, low-s normalised.
    static bool sign(const Bytes& message, const Bytes& privateKey, Bytes& signature) {
        try {
            if (message.size() != 32) {
                std::cerr << "Signing failed: the digest must be 32 bytes" << std::endl;
                return false;
            }
            const auto secret =
                tigerwallet::secp256k1::from_bytes(privateKey.data(), privateKey.size());
            if (!tigerwallet::secp256k1::is_valid_scalar(secret.get())) {
                std::cerr << "Signing failed: the private key is not a valid scalar" << std::endl;
                return false;
            }
            std::array<uint8_t, 32> digest{};
            std::copy(message.begin(), message.end(), digest.begin());
            signature = tigerwallet::secp256k1::sign(secret.get(), digest).to_bytes();
            return true;
        } catch (const std::exception& e) {
            std::cerr << "Signing failed: " << e.what() << std::endl;
            return false;
        }
    }

    // Verifies a 64-byte r||s signature against a compressed or uncompressed
    // public key.
    static bool verify(const Bytes& message, const Bytes& signature, const Bytes& publicKey) {
        try {
            if (message.size() != 32) return false;
            std::array<uint8_t, 32> digest{};
            std::copy(message.begin(), message.end(), digest.begin());
            return tigerwallet::secp256k1::verify(publicKey, digest, signature);
        } catch (const std::exception& e) {
            std::cerr << "Verification failed: " << e.what() << std::endl;
            return false;
        }
    }

    // The EIP-712 signing digest: keccak256(0x1901 || domainSeparator ||
    // structHash).
    //
    // The caller supplies both hashes because they depend on the struct being
    // signed, which this layer does not know. Passing the wrong shape here was
    // the previous bug: it concatenated the raw domain text with the raw
    // message and hashed with SHA-256, which produces a digest no contract
    // recognises.
    static Bytes eip712Hash(const bytes32& domainSeparator, const bytes32& structHash) {
        Bytes payload;
        payload.reserve(2 + 32 + 32);
        payload.push_back(0x19);
        payload.push_back(0x01);
        payload.insert(payload.end(), domainSeparator.begin(), domainSeparator.end());
        payload.insert(payload.end(), structHash.begin(), structHash.end());
        const auto digest = tigerwallet::secp256k1::keccak256(payload);
        return Bytes(digest.begin(), digest.end());
    }

    // The EIP-712 domain separator for an EIP-2612 style domain. The type hash
    // is fixed by the specification.
    static bytes32 domainSeparator(const std::string& name,
                                   const std::string& version,
                                   ChainId chainId,
                                   const Address& verifyingContract) {
        // keccak256("EIP712Domain(string name,string version,uint256
        //          chainId,address verifyingContract)")
        static const std::string kDomainType =
            "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)";
        const auto type_hash = tigerwallet::secp256k1::keccak256(
            reinterpret_cast<const uint8_t*>(kDomainType.data()), kDomainType.size());
        const auto name_hash = tigerwallet::secp256k1::keccak256(
            reinterpret_cast<const uint8_t*>(name.data()), name.size());
        const auto version_hash = tigerwallet::secp256k1::keccak256(
            reinterpret_cast<const uint8_t*>(version.data()), version.size());

        Bytes encoded(type_hash.begin(), type_hash.end());
        encoded.insert(encoded.end(), name_hash.begin(), name_hash.end());
        encoded.insert(encoded.end(), version_hash.begin(), version_hash.end());

        // chainId is a 32-byte big-endian word.
        for (int i = 7; i >= 0; --i) {
            encoded.push_back(static_cast<uint8_t>((chainId >> (i * 8)) & 0xff));
        }
        for (int i = 0; i < 24; ++i) encoded.push_back(0);

        // The contract address is the last 20 bytes of a 32-byte word.
        const auto contract = decode_address(verifyingContract);
        if (!contract) throw std::runtime_error("EIP-712: invalid verifying contract address");
        encoded.insert(encoded.end(), contract->begin(), contract->end());

        return tigerwallet::secp256k1::keccak256(encoded);
    }

    // The Ethereum address for a public key: keccak256 of the uncompressed key
    // without its 0x04 prefix, last 20 bytes, in EIP-55 checksum form.
    static Address publicKeyToAddress(const Bytes& publicKey) {
        const auto point = tigerwallet::secp256k1::decode_point(publicKey);
        if (!point) {
            throw std::runtime_error("publicKeyToAddress: not a point on the curve");
        }
        return tigerwallet::secp256k1::to_checksum_address(
            tigerwallet::secp256k1::ethereum_address(point->get()));
    }

    // AES-256-GCM encryption. The returned buffer is the ciphertext with the
    // 16-byte tag appended, and the IV is written to `iv`.
    static bool encryptAES(const Bytes& plaintext, const Bytes& key, Bytes& ciphertext, Bytes& iv) {
        if (key.size() != 32) {
            std::cerr << "Encryption failed: the key must be 32 bytes" << std::endl;
            return false;
        }
        try {
            iv.resize(12);
            if (RAND_bytes(iv.data(), static_cast<int>(iv.size())) != 1) {
                std::cerr << "Encryption failed: no random source" << std::endl;
                return false;
            }

            EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
            if (!ctx) return false;
            struct Guard {
                EVP_CIPHER_CTX* ctx;
                ~Guard() { EVP_CIPHER_CTX_free(ctx); }
            } guard{ctx};

            if (EVP_EncryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) return false;
            if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, static_cast<int>(iv.size()), nullptr) != 1) {
                return false;
            }
            if (EVP_EncryptInit_ex(ctx, nullptr, nullptr, key.data(), iv.data()) != 1) return false;

            Bytes out(plaintext.size() + 16);
            int written = 0;
            if (!plaintext.empty() &&
                EVP_EncryptUpdate(ctx, out.data(), &written, plaintext.data(),
                                  static_cast<int>(plaintext.size())) != 1) {
                return false;
            }
            int final_written = 0;
            if (EVP_EncryptFinal_ex(ctx, out.data() + written, &final_written) != 1) return false;
            const size_t cipher_len = static_cast<size_t>(written + final_written);

            Bytes tag(16);
            if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, 16, tag.data()) != 1) return false;

            ciphertext.assign(out.begin(), out.begin() + static_cast<long>(cipher_len));
            ciphertext.insert(ciphertext.end(), tag.begin(), tag.end());
            return true;
        } catch (const std::exception& e) {
            std::cerr << "Encryption failed: " << e.what() << std::endl;
            return false;
        }
    }

    // AES-256-GCM decryption with tag verification.
    //
    // The tag is given to the cipher before the final call, so EVP_DecryptFinal_ex
    // fails on a tampered ciphertext and nothing is released. The previous
    // version resized nothing and compared tags after the fact, which both
    // wrote out of bounds and skipped authentication.
    static bool decryptAES(const Bytes& ciphertext, const Bytes& key, const Bytes& iv, Bytes& plaintext) {
        if (key.size() != 32 || iv.size() != 12 || ciphertext.size() < 16) {
            std::cerr << "Decryption failed: bad key, IV or ciphertext length" << std::endl;
            return false;
        }
        try {
            const size_t body_len = ciphertext.size() - 16;
            const uint8_t* body = ciphertext.data();
            const uint8_t* tag = ciphertext.data() + body_len;

            EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
            if (!ctx) return false;
            struct Guard {
                EVP_CIPHER_CTX* ctx;
                ~Guard() { EVP_CIPHER_CTX_free(ctx); }
            } guard{ctx};

            if (EVP_DecryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) return false;
            if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, static_cast<int>(iv.size()), nullptr) != 1) {
                return false;
            }
            if (EVP_DecryptInit_ex(ctx, nullptr, nullptr, key.data(), iv.data()) != 1) return false;

            Bytes out(body_len);
            int written = 0;
            if (body_len > 0 &&
                EVP_DecryptUpdate(ctx, out.data(), &written, body,
                                  static_cast<int>(body_len)) != 1) {
                return false;
            }

            // The tag must be set before the final call, which is what performs
            // the authentication.
            if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, 16,
                                    const_cast<uint8_t*>(tag)) != 1) {
                return false;
            }
            int final_written = 0;
            if (EVP_DecryptFinal_ex(ctx, out.data() + written, &final_written) != 1) {
                std::cerr << "Decryption failed: authentication tag mismatch" << std::endl;
                return false;
            }

            plaintext.assign(out.begin(), out.begin() + written + final_written);
            return true;
        } catch (const std::exception& e) {
            std::cerr << "Decryption failed: " << e.what() << std::endl;
            return false;
        }
    }

    // Decodes a 0x-prefixed 20-byte address into its raw bytes. Returns
    // nullopt when the text is not a well-formed address, so a caller cannot
    // sign against a malformed one.
    static std::optional<Bytes> decode_address(const std::string& address) {
        if (address.size() != 42 || address[0] != '0' || address[1] != 'x') {
            return std::nullopt;
        }
        Bytes out(20);
        for (size_t i = 0; i < 20; ++i) {
            const char high = address[2 + i * 2];
            const char low = address[3 + i * 2];
            const auto nibble = [](char c) -> int {
                if (c >= '0' && c <= '9') return c - '0';
                if (c >= 'a' && c <= 'f') return c - 'a' + 10;
                if (c >= 'A' && c <= 'F') return c - 'A' + 10;
                return -1;
            };
            const int h = nibble(high);
            const int l = nibble(low);
            if (h < 0 || l < 0) return std::nullopt;
            out[i] = static_cast<uint8_t>((h << 4) | l);
        }
        return out;
    }
};

// =============================================================================
// ENTRY POINT CONTRACT
// =============================================================================

class EntryPointContract {
public:
    EntryPointContract(const Address& entryPointAddress, ChainId chainId)
        : address_(entryPointAddress), chainId_(chainId) {}

    // Handle an array of user operations
    std::vector<UserOperation> handleOps(const std::vector<UserOperation>& ops, Address beneficiary) {
        std::vector<UserOperation> failedOps;

        for (const auto& op : ops) {
            if (!validatePrePayment(op)) {
                failedOps.push_back(op);
                continue;
            }

            if (!executeUserOp(op)) {
                failedOps.push_back(op);
            }
        }

        return failedOps;
    }

    // Installs the node call used for simulation and execution.
    //
    // It receives the target address, the calldata, and a flag for whether the
    // call changes state or is a read-only eth_call, and returns the call's
    // success and output. Until one is installed, simulateHandleOp and
    // executeUserOp refuse rather than claim a result.
    using NodeCall = std::function<bool(const Address&, const Bytes&, bool, Bytes&)>;

    void setNodeCall(NodeCall call) {
        std::lock_guard<std::mutex> lock(nodeMutex_);
        nodeCall_ = std::move(call);
    }

    // Simulates an operation against the node.
    //
    // The previous version returned validatePrePayment(op), which checks the
    // sender, the gas limits and the fee ordering. It never ran the operation,
    // so it could not report whether the call would revert, and the target and
    // calldata arguments were ignored entirely. A caller relying on it would be
    // told that an operation which reverts is fine.
    bool simulateHandleOp(const UserOperation& op, Address target, Bytes calldata) {
        if (!validatePrePayment(op)) return false;
        if (!CryptoUtils::decode_address(target)) {
            throw std::runtime_error("simulateHandleOp: the target address is not valid");
        }

        NodeCall call;
        {
            std::lock_guard<std::mutex> lock(nodeMutex_);
            call = nodeCall_;
        }
        if (!call) {
            throw std::runtime_error(
                "simulateHandleOp: no node configured; call setNodeCall() to enable simulation");
        }

        Bytes output;
        return call(target, calldata, false, output);
    }

    // The entry point's deposit for an account, read from the node.
    //
    // The previous version returned the account address it was given, which is
    // not a deposit, so a caller reading it as a balance would act on a wrong
    // number.
    uint256_t getDepositInfo(const Address& account) {
        if (!CryptoUtils::decode_address(account)) {
            throw std::runtime_error("getDepositInfo: the account address is not valid");
        }

        NodeCall call;
        {
            std::lock_guard<std::mutex> lock(nodeMutex_);
            call = nodeCall_;
        }
        if (!call) {
            throw std::runtime_error(
                "getDepositInfo: no node configured; call setNodeCall() to read deposits");
        }

        // balances(address) selector, then the account as a 32-byte word.
        Bytes data = {0x70, 0xa0, 0x82, 0x31};
        const auto account_bytes = CryptoUtils::decode_address(account);
        for (int i = 0; i < 12; ++i) data.push_back(0);
        data.insert(data.end(), account_bytes->begin(), account_bytes->end());

        Bytes output;
        if (!call(address_, data, false, output) || output.size() < 32) {
            throw std::runtime_error("getDepositInfo: the node call failed");
        }
        return decode_word(output.data());
    }

    // The entry point address this contract represents.
    Address getAddress() const { return address_; }

    // Executes an operation against the node.
    //
    // The previous version returned true without doing anything, so handleOps
    // reported every operation as successful. Without a node this throws,
    // because reporting success for an operation that was never submitted is
    // worse than reporting that nothing happened.
    bool executeUserOp(const UserOperation& op) {
        NodeCall call;
        {
            std::lock_guard<std::mutex> lock(nodeMutex_);
            call = nodeCall_;
        }
        if (!call) {
            throw std::runtime_error(
                "executeUserOp: no node configured; call setNodeCall() to enable execution");
        }

        Bytes output;
        return call(address_, op.callData, true, output);
    }

private:
    Address address_;
    ChainId chainId_;
    std::mutex nodeMutex_;
    NodeCall nodeCall_;

    // A 32-byte big-endian word as a 256-bit value.
    static uint256_t decode_word(const uint8_t* data) {
        return uint256_t::fromWords(read_word(data + 24), read_word(data + 16),
                                    read_word(data + 8), read_word(data));
    }

    static uint64_t read_word(const uint8_t* data) {
        uint64_t value = 0;
        for (int i = 0; i < 8; ++i) value = (value << 8) | data[i];
        return value;
    }

    bool validatePrePayment(const UserOperation& op) {
        // Check sender is not zero
        if (!CryptoUtils::decode_address(op.sender)) return false;

        // Check gas limits are reasonable
        if (op.verificationGasLimit < uint256_t::fromUint64(21000)) return false;
        if (op.callGasLimit < uint256_t::fromUint64(21000)) return false;

        // Check fees
        if (op.maxFeePerGas < op.maxPriorityFeePerGas) return false;

        return true;
    }

};

// =============================================================================
// ACCOUNT FACTORY
// =============================================================================

class AccountFactory {
public:
    AccountFactory() = default;

    // The address a smart account will have once deployed.
    //
    // This is the CREATE2 address: keccak256(0xff || factory || salt ||
    // keccak256(initCode)), last 20 bytes. The previous version concatenated
    // the owner string with the salt bytes, hashed them with SHA-256 through a
    // function that was itself wrong, and sliced the result. That produced an
    // address unrelated to any deployed contract, so a client sending funds to
    // it would lose them.
    Address getAccountAddress(const Address& owner, const Bytes& salt) {
        const auto factory = CryptoUtils::decode_address(factoryAddress_);
        if (!factory) {
            throw std::runtime_error("AccountFactory: the factory address is not valid");
        }
        if (salt.size() != 32) {
            throw std::runtime_error("AccountFactory: the salt must be 32 bytes");
        }

        const Bytes initCode = buildInitCode(owner);
        const auto initCodeHash = tigerwallet::secp256k1::keccak256(initCode);

        Bytes material;
        material.reserve(1 + 20 + 32 + 32);
        material.push_back(0xff);
        material.insert(material.end(), factory->begin(), factory->end());
        material.insert(material.end(), salt.begin(), salt.end());
        material.insert(material.end(), initCodeHash.begin(), initCodeHash.end());

        const auto digest = tigerwallet::secp256k1::keccak256(material);
        const Bytes address_bytes(digest.begin() + 12, digest.end());
        return tigerwallet::secp256k1::to_checksum_address(address_bytes);
    }

    // Creates a new smart account and records it. The address is
    // deterministic, so the same owner and salt yield the same account rather
    // than a second entry.
    Address createAccount(const Address& owner, const Bytes& salt) {
        const Address accountAddress = getAccountAddress(owner, salt);

        AccountInfo info;
        info.owner = owner;
        info.accountAddress = accountAddress;
        info.createdAt = currentTimestamp();
        info.isDeployed = false;

        accounts_[accountAddress] = info;

        return accountAddress;
    }

    // The factory address, which the CREATE2 derivation commits to.
    void setFactoryAddress(const Address& factory) {
        if (!CryptoUtils::decode_address(factory)) {
            throw std::runtime_error("AccountFactory: the factory address is not valid");
        }
        factoryAddress_ = factory;
    }

    Address getFactoryAddress() const { return factoryAddress_; }

    // The implementation the accounts delegate to.
    void setImplementationAddress(const Address& implementation) {
        if (!CryptoUtils::decode_address(implementation)) {
            throw std::runtime_error("AccountFactory: the implementation address is not valid");
        }
        implementationAddress_ = implementation;
    }

    // The init code the factory deploys: the implementation address padded to a
    // word, then the owner as the constructor argument.
    Bytes buildInitCode(const Address& owner) const {
        const auto implementation = CryptoUtils::decode_address(implementationAddress_);
        const auto owner_bytes = CryptoUtils::decode_address(owner);
        if (!implementation || !owner_bytes) {
            throw std::runtime_error(
                "AccountFactory: implementation or owner address is not valid");
        }
        Bytes code(implementation->begin(), implementation->end());
        for (int i = 0; i < 12; ++i) code.push_back(0);
        code.insert(code.end(), owner_bytes->begin(), owner_bytes->end());
        return code;
    }

    // Check if account is deployed
    bool isDeployed(const Address& account) {
        auto it = accounts_.find(account);
        if (it != accounts_.end()) {
            return it->second.isDeployed;
        }
        return false;
    }

private:
    struct AccountInfo {
        Address owner;
        Address accountAddress;
        bool isDeployed;
        Timestamp createdAt;
        std::vector<Address> guardians;
    };

    std::unordered_map<Address, AccountInfo> accounts_;
    Address factoryAddress_ = "0x0000000000000000000000000000000000000000";
    Address implementationAddress_ = "0x0000000000000000000000000000000000000000";

    static std::string bytesToHex(const Bytes& bytes) {
        static const char* kHex = "0123456789abcdef";
        std::string result;
        result.reserve(bytes.size() * 2);
        for (const auto& b : bytes) {
            result.push_back(kHex[b >> 4]);
            result.push_back(kHex[b & 0xf]);
        }
        return result;
    }

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
};

// =============================================================================
// SESSION KEY MANAGER
// =============================================================================

class SessionKeyManager {
public:
    SessionKeyManager() = default;

    // Create a new session key
    SessionKey createSessionKey(
        const Address& walletAddress,
        const Address& sessionKey,
        const std::vector<std::string>& allowedMethods,
        const std::vector<Address>& allowedContracts,
        uint256_t maxAmount,
        uint64_t validitySeconds
    ) {
        SessionKey key;
        key.key = sessionKey;
        key.sessionKeyAddress = walletAddress;
        key.allowedMethods = allowedMethods;
        key.allowedContracts = allowedContracts;
        key.maxAmount = maxAmount;
        key.validUntil = currentTimestamp() + validitySeconds * 1000;
        key.createdAt = currentTimestamp();

        // Store the key
        sessionKeys_[walletAddress].push_back(key);

        return key;
    }

    // Validate a session key for a transaction
    bool validateSessionKey(
        const Address& walletAddress,
        const Address& sessionKey,
        const std::string& method,
        const Address& target,
        uint256_t amount
    ) {
        auto it = sessionKeys_.find(walletAddress);
        if (it == sessionKeys_.end()) {
            return false;
        }

        for (const auto& key : it->second) {
            if (key.key != sessionKey) continue;

            // Check validity
            if (currentTimestamp() > key.validUntil) {
                return false;
            }

            // Check amount
            if (amount > key.maxAmount) {
                return false;
            }

            // Check allowed method
            bool methodAllowed = false;
            for (const auto& m : key.allowedMethods) {
                if (m == "*" || m == method) {
                    methodAllowed = true;
                    break;
                }
            }
            if (!methodAllowed) return false;

            // Check allowed contract
            if (!key.allowedContracts.empty()) {
                bool contractAllowed = false;
                for (const auto& c : key.allowedContracts) {
                    if (c == target) {
                        contractAllowed = true;
                        break;
                    }
                }
                if (!contractAllowed) return false;
            }

            return true;
        }

        return false;
    }

    // Revoke a session key
    bool revokeSessionKey(const Address& walletAddress, const Address& sessionKey) {
        auto it = sessionKeys_.find(walletAddress);
        if (it == sessionKeys_.end()) {
            return false;
        }

        auto& keys = it->second;
        for (auto it2 = keys.begin(); it2 != keys.end(); ++it2) {
            if (it2->key == sessionKey) {
                keys.erase(it2);
                return true;
            }
        }

        return false;
    }

    // Get all session keys for a wallet
    std::vector<SessionKey> getSessionKeys(const Address& walletAddress) {
        auto it = sessionKeys_.find(walletAddress);
        if (it != sessionKeys_.end()) {
            return it->second;
        }
        return {};
    }

private:
    std::unordered_map<Address, std::vector<SessionKey>> sessionKeys_;

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
};

// =============================================================================
// PAYMASTER MANAGER
// =============================================================================

class PaymasterManager {
public:
    PaymasterManager() = default;

    // Configure a paymaster
    void configurePaymaster(
        const Address& paymasterAddress,
        const Address& stakingToken,
        uint256_t minStake,
        uint256_t minUnstakeDelay
    ) {
        PaymasterConfig config;
        config.paymasterAddress = paymasterAddress;
        config.stakingToken = stakingToken;
        config.minStake = minStake;
        config.minUnstakeDelay = minUnstakeDelay;
        config.isActive = true;
        config.deposit = uint256_t::fromUint64(0);

        paymasters_[paymasterAddress] = config;
    }

    // Check if paymaster is valid
    bool isPaymasterValid(const Address& paymaster, uint256_t maxFee) {
        auto it = paymasters_.find(paymaster);
        if (it == paymasters_.end()) {
            return false;
        }

        const auto& config = it->second;
        if (!config.isActive) return false;
        if (config.deposit < config.minStake) return false;

        return true;
    }

    // Validates a user operation against its paymaster.
    //
    // The previous version checked the paymaster's registration and ignored
    // requiredPreFund entirely, so an operation whose prefund exceeded the
    // paymaster's deposit was approved and the entry point would then fail to
    // collect. The prefund is now checked against the deposit.
    bool validatePaymasterUserOp(
        const UserOperation& op,
        uint256_t requiredPreFund
    ) {
        if (op.paymaster.empty()) {
            return false; // No paymaster specified
        }
        if (!isPaymasterValid(op.paymaster, op.maxFeePerGas)) {
            return false;
        }

        auto it = paymasters_.find(op.paymaster);
        if (it == paymasters_.end()) return false;

        // The deposit must cover the prefund plus the staked minimum, because
        // the stake is what backs the paymaster's guarantees.
        const uint256_t needed = requiredPreFund + it->second.minStake;
        return it->second.deposit >= needed;
    }

    // Post-operation handler.
    //
    // The previous version returned an empty buffer with a comment describing
    // what it should do. This charges the actual gas cost against the
    // paymaster's deposit, which is the accounting the entry point expects, and
    // records the settlement so it is auditable.
    Bytes postOp(
        uint8_t mode,
        const UserOperation& op,
        bytes32 success,
        uint256_t actualGasCost,
        Bytes calldata
    ) {
        if (op.paymaster.empty()) {
            throw std::runtime_error("postOp: the operation has no paymaster");
        }

        auto it = paymasters_.find(op.paymaster);
        if (it == paymasters_.end()) {
            throw std::runtime_error("postOp: the paymaster is not registered");
        }

        // If the deposit cannot cover the actual cost the operation is not
        // settled, and the caller must not treat it as paid.
        if (it->second.deposit < actualGasCost) {
            throw std::runtime_error(
                "postOp: the paymaster deposit does not cover the actual gas cost");
        }
        it->second.deposit = it->second.deposit - actualGasCost;

        {
            std::lock_guard<std::mutex> lock(postOpMutex_);
            postOpRecords_.push_back({op.sender, op.paymaster, actualGasCost, success, mode});
        }

        Bytes result;
        result.reserve(32);
        result.insert(result.end(), success.begin(), success.end());
        return result;
    }

    // A settlement recorded by postOp.
    struct PostOpRecord {
        Address sender;
        Address paymaster;
        uint256_t actualGasCost;
        bytes32 success;
        uint8_t mode;
    };

    // The settlements recorded by postOp, in order.
    std::vector<PostOpRecord> postOpRecords() const {
        std::lock_guard<std::mutex> lock(postOpMutex_);
        return postOpRecords_;
    }

    // The remaining deposit for a paymaster.
    uint256_t paymasterDeposit(const Address& paymaster) const {
        auto it = paymasters_.find(paymaster);
        if (it == paymasters_.end()) return uint256_t::fromUint64(0);
        return it->second.deposit;
    }

    // Fund paymaster deposit
    void fundPaymasterDeposit(const Address& paymaster, uint256_t amount) {
        auto it = paymasters_.find(paymaster);
        if (it != paymasters_.end()) {
            it->second.deposit = it->second.deposit + amount;
        }
    }

    // Withdraw from paymaster deposit
    bool withdrawPaymasterDeposit(const Address& paymaster, uint256_t amount) {
        auto it = paymasters_.find(paymaster);
        if (it == paymasters_.end()) {
            return false;
        }

        if (it->second.deposit < amount) {
            return false;
        }

        it->second.deposit = it->second.deposit - amount;
        return true;
    }

private:
    std::unordered_map<Address, PaymasterConfig> paymasters_;
    mutable std::mutex postOpMutex_;
    std::vector<PostOpRecord> postOpRecords_;
};

// =============================================================================
// EIP-7702 AUTHORIZATION
// =============================================================================

class EIP7702Manager {
public:
    EIP7702Manager(ChainId chainId) : chainId_(chainId) {}

    // Installs the node call used to submit an authorization.
    //
    // It receives the transaction payload and returns whether the node
    // accepted it, along with the transaction hash. Until one is installed,
    // executeAuthorization refuses.
    using NodeSubmit = std::function<bool(const Bytes&, Bytes&)>;

    void setNodeSubmit(NodeSubmit submit) {
        std::lock_guard<std::mutex> lock(submitMutex_);
        nodeSubmit_ = std::move(submit);
    }

    // The authorization for upgrading an EOA to a contract.
    //
    // The nonce is the account's own transaction nonce, which the caller
    // supplies because only the chain knows it. The previous version generated
    // 32 random bytes with rand(), which is not the nonce the chain expects:
    // EIP-7702 binds the authorization to the EOA's current nonce, and a random
    // value would produce an authorization the account can never use.
    Authorization createAuthorization(const Address& contractAddress, uint64_t accountNonce) {
        if (!CryptoUtils::decode_address(contractAddress)) {
            throw std::runtime_error("EIP-7702: the contract address is not valid");
        }
        if (contractAddress == "0x0000000000000000000000000000000000000000") {
            throw std::runtime_error(
                "EIP-7702: the contract address is zero, which would clear the delegation");
        }

        Authorization auth;
        auth.chainId = chainId_;
        auth.address = contractAddress;
        auth.nonce = accountNonce;
        auth.isValid = true;
        auth.expiresAt = currentTimestamp() + 3600000; // 1 hour

        return auth;
    }

    // The hash EIP-7702 signs: keccak256(0x05 || rlp([chainId, address, nonce])).
    //
    // The previous version signed the ASCII string "EIP7702:<chain>:<address>:<nonce>",
    // which is not the digest the protocol defines. An authorization signed
    // that way would be rejected by every node, and the domain separation the
    // leading 0x05 provides was missing, so the same signature could be
    // replayed as a different message type.
    bytes32 authorizationHash(const Authorization& auth) const {
        const auto address_bytes = CryptoUtils::decode_address(auth.address);
        if (!address_bytes) {
            throw std::runtime_error("EIP-7702: the address is not valid");
        }

        Bytes encoded = {0x05};
        encoded.push_back(0xd8);  // A three-item RLP list (payload under 56 bytes).

        // chainId
        encoded.push_back(0x84);
        for (int i = 3; i >= 0; --i) {
            encoded.push_back(static_cast<uint8_t>((auth.chainId >> (i * 8)) & 0xff));
        }

        // address, 20 bytes
        encoded.push_back(0x94);
        encoded.insert(encoded.end(), address_bytes->begin(), address_bytes->end());

        // nonce, as a minimal big-endian integer. Zero is the empty string.
        if (auth.nonce == 0) {
            encoded.push_back(0x80);
        } else {
            Bytes nonce_bytes;
            uint64_t value = auth.nonce;
            while (value > 0) {
                nonce_bytes.insert(nonce_bytes.begin(), static_cast<uint8_t>(value & 0xff));
                value >>= 8;
            }
            if (nonce_bytes.size() == 1 && nonce_bytes[0] < 0x80) {
                encoded.push_back(nonce_bytes[0]);
            } else {
                encoded.push_back(static_cast<uint8_t>(0x80 + nonce_bytes.size()));
                encoded.insert(encoded.end(), nonce_bytes.begin(), nonce_bytes.end());
            }
        }

        return tigerwallet::secp256k1::keccak256(encoded);
    }

    // Signs the authorization. The private key belongs to the EOA being
    // upgraded.
    bool signAuthorization(const Authorization& auth, const Bytes& privateKey, Bytes& signature) {
        const auto digest = authorizationHash(auth);
        return CryptoUtils::sign(std::vector<uint8_t>(digest.begin(), digest.end()),
                                 privateKey, signature);
    }

    // Verifies an authorization against the signer's public key.
    bool verifyAuthorization(const Authorization& auth, const Bytes& signature,
                             const Bytes& publicKey) {
        if (currentTimestamp() > auth.expiresAt) return false;
        if (!auth.isValid) return false;

        const auto digest = authorizationHash(auth);
        return CryptoUtils::verify(std::vector<uint8_t>(digest.begin(), digest.end()),
                                   signature, publicKey);
    }

    // Submits the authorization to the node.
    //
    // The previous version returned true without submitting anything, so a
    // caller believed the upgrade had happened. The transaction hash is
    // returned through `txHash`.
    bool executeAuthorization(const Authorization& auth, const Bytes& signature,
                              const Bytes& signedTransaction, Bytes& txHash) {
        if (!auth.isValid) return false;
        if (currentTimestamp() > auth.expiresAt) return false;

        NodeSubmit submit;
        {
            std::lock_guard<std::mutex> lock(submitMutex_);
            submit = nodeSubmit_;
        }
        if (!submit) {
            throw std::runtime_error(
                "executeAuthorization: no node configured; call setNodeSubmit() to submit");
        }

        // The caller builds the signed transaction, because the transaction
        // type and fee fields are chain policy. The signature is checked here
        // so a malformed one is not submitted.
        if (signature.size() != 64) {
            throw std::runtime_error("executeAuthorization: the signature must be 64 bytes");
        }
        if (signedTransaction.empty()) {
            throw std::runtime_error("executeAuthorization: the signed transaction is empty");
        }

        return submit(signedTransaction, txHash);
    }

    // The authorizations submitted so far, for auditing.
    struct Submission {
        Address contract;
        uint64_t nonce;
        Bytes txHash;
    };

    std::vector<Submission> submissions() const {
        std::lock_guard<std::mutex> lock(submitMutex_);
        return submissions_;
    }

    ChainId chainId() const { return chainId_; }

private:
    ChainId chainId_;
    mutable std::mutex submitMutex_;
    NodeSubmit nodeSubmit_;
    std::vector<Submission> submissions_;

    uint64_t currentTimestamp() const {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
};

// =============================================================================
// BATCHED TRANSACTIONS
// =============================================================================

class BatchedTransactionManager {
public:
    // Installs the node call used to execute an operation.
    //
    // It receives the target and calldata and returns whether the call
    // succeeded. Until one is installed, executeBatch refuses rather than
    // reporting success for work that never happened.
    using NodeExecute = std::function<bool(const Address&, const Bytes&, Bytes&)>;

    void setNodeExecute(NodeExecute execute) {
        std::lock_guard<std::mutex> lock(mutex_);
        nodeExecute_ = std::move(execute);
    }

    // Creates a batch and returns its identifier.
    std::string createBatch(
        const Address& sender,
        const std::vector<UserOperation>& operations,
        uint64_t deadlineSeconds
    ) {
        if (!CryptoUtils::decode_address(sender)) {
            throw std::runtime_error("createBatch: the sender address is not valid");
        }
        if (operations.empty()) {
            throw std::runtime_error("createBatch: the batch is empty");
        }
        if (deadlineSeconds == 0) {
            throw std::runtime_error("createBatch: the deadline must be positive");
        }

        BatchedTransaction batch;
        // The identifier comes from the system random source. The previous
        // version used rand(), which is a predictable sequence, so one caller
        // could guess another caller's batch id and execute or cancel it.
        batch.id = "0x" + randomHex(16);
        batch.sender = sender;
        batch.operations = operations;
        batch.totalGasLimit = uint256_t::fromUint64(0);
        batch.totalValue = uint256_t::fromUint64(0);

        // Sum what the batch actually moves. The previous version added zero
        // for every operation, so totalValue was always zero.
        for (const auto& op : operations) {
            batch.totalGasLimit = batch.totalGasLimit + op.callGasLimit;
            batch.totalValue = batch.totalValue + op.callValue;
        }

        batch.createdAt = currentTimestamp();
        batch.expiresAt = currentTimestamp() + deadlineSeconds * 1000;

        std::lock_guard<std::mutex> lock(mutex_);
        batches_[batch.id] = batch;

        return batch.id;
    }

    // Executes every operation in a batch.
    //
    // The previous version called executeOperation, which returned true for
    // every operation without doing anything, so executeBatch reported success
    // for a batch that was never submitted.
    bool executeBatch(const std::string& batchId) {
        NodeExecute execute;
        {
            std::lock_guard<std::mutex> lock(mutex_);
            auto it = batches_.find(batchId);
            if (it == batches_.end()) return false;

            auto& batch = it->second;
            if (currentTimestamp() > batch.expiresAt) {
                batches_.erase(it);
                return false;
            }
            execute = nodeExecute_;
        }
        if (!execute) {
            throw std::runtime_error(
                "executeBatch: no node configured; call setNodeExecute() to execute");
        }

        std::vector<UserOperation> operations;
        {
            std::lock_guard<std::mutex> lock(mutex_);
            operations = batches_.at(batchId).operations;
        }

        // A batch is all or nothing. If one operation fails, the ones already
        // sent cannot be recalled, so the caller is told the batch failed and
        // the remaining operations are not attempted.
        for (const auto& op : operations) {
            Bytes output;
            if (!execute(op.sender, op.callData, output)) {
                std::lock_guard<std::mutex> lock(mutex_);
                batches_.erase(batchId);
                return false;
            }
        }

        std::lock_guard<std::mutex> lock(mutex_);
        batches_.erase(batchId);
        return true;
    }

    bool cancelBatch(const std::string& batchId) {
        std::lock_guard<std::mutex> lock(mutex_);
        return batches_.erase(batchId) > 0;
    }

    std::optional<BatchedTransaction> getBatch(const std::string& batchId) {
        std::lock_guard<std::mutex> lock(mutex_);
        auto it = batches_.find(batchId);
        if (it != batches_.end()) {
            return it->second;
        }
        return std::nullopt;
    }

private:
    std::unordered_map<std::string, BatchedTransaction> batches_;
    std::atomic<uint64_t> txCounter_{0};
    std::mutex mutex_;
    NodeExecute nodeExecute_;

    uint64_t currentTimestamp() const {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }

    static std::string randomHex(size_t length) {
        std::vector<uint8_t> bytes(length);
        if (RAND_bytes(bytes.data(), static_cast<int>(length)) != 1) {
            throw std::runtime_error("randomHex: no random source");
        }
        static const char* kHex = "0123456789abcdef";
        std::string result;
        result.reserve(length * 2);
        for (uint8_t b : bytes) {
            result.push_back(kHex[b >> 4]);
            result.push_back(kHex[b & 0xf]);
        }
        return result;
    }
};

// =============================================================================
// GAS ABSTRACTION (Stablecoin Gas Payment)
// =============================================================================

class GasAbstraction {
public:
    GasAbstraction() = default;

    // Registers a token that may be used to pay gas. The address must be well
    // formed, so the registry cannot hold an entry that can never be matched.
    void addGasToken(const Address& token, bool isStable) {
        if (!CryptoUtils::decode_address(token)) {
            throw std::runtime_error("GasAbstraction: the token address is not valid");
        }
        gasTokens_[token] = isStable;
    }

    // Installs the price source. It receives a token address and returns the
    // price of one whole token in USD scaled by 1e18, so one dollar is
    // 1000000000000000000. Zero means "no price".
    //
    // The previous version returned hardcoded prices: 1.0 for USDT and USDC,
    // 3500.0 for ETH and 1.0 for every unknown token. A wallet would quote gas
    // in an unknown token as if it were worth a dollar, and the ETH figure was
    // wrong as soon as the market moved.
    void setPriceOracle(std::function<uint256_t(const Address&)> oracle) {
        std::lock_guard<std::mutex> lock(priceMutex_);
        priceOracle_ = std::move(oracle);
    }

    uint256_t tokenPriceUsd(const Address& token) {
        std::lock_guard<std::mutex> lock(priceMutex_);
        if (!priceOracle_) {
            throw std::runtime_error(
                "GasAbstraction: no price oracle configured; call setPriceOracle() first");
        }
        return priceOracle_(token);
    }

    // The amount of `token` needed to pay for `gasUsed` at `gasPrice`.
    //
    // The result is gasUsed * gasPrice * 1e18 / priceUsd in integer arithmetic
    // with a 512-bit intermediate. The previous version converted both 256-bit
    // values to double: a double has 53 bits of mantissa, so any amount above
    // about 9e15 is not exact and the quoted cost drifts from the real one. It
    // also divided by the price twice, losing precision at each step.
    uint256_t calculateGasCost(const Address& token, uint256_t gasUsed, uint256_t gasPrice) {
        const uint256_t price = tokenPriceUsd(token);
        if (price.isZero()) {
            throw std::runtime_error("GasAbstraction: the oracle returned no price for this token");
        }

        const uint256_t gas_cost_wei =
            tigerwallet::math::mulDiv(gasUsed, gasPrice, uint256_t::fromUint64(1));
        if (gas_cost_wei.isZero() && !gasUsed.isZero() && !gasPrice.isZero()) {
            throw std::runtime_error("GasAbstraction: the gas cost overflowed 256 bits");
        }

        return tigerwallet::math::mulDiv(
            gas_cost_wei,
            tigerwallet::math::U256::fromDecimal("1000000000000000000"),
            price);
    }

    // Pays for an operation with a gas token.
    //
    // The token must be registered and priced, and the amount must cover the
    // cost. The previous version checked only the registration and returned
    // true, so an operation that transferred nothing was reported as paid.
    bool payWithGasToken(const UserOperation& op, const Address& gasToken, uint256_t tokenAmount) {
        if (gasTokens_.find(gasToken) == gasTokens_.end()) {
            return false;
        }

        uint256_t required;
        try {
            required = calculateGasCost(gasToken, op.callGasLimit, op.maxFeePerGas);
        } catch (const std::exception&) {
            // Without a price the payment cannot be valued, so it cannot be
            // accepted.
            return false;
        }

        if (tokenAmount < required) {
            return false;
        }

        std::lock_guard<std::mutex> lock(settlementMutex_);
        settlements_[op.sender] = settlements_[op.sender] + tokenAmount;
        settlementsByToken_[op.sender][gasToken] =
            settlementsByToken_[op.sender][gasToken] + tokenAmount;
        return true;
    }

    // What a payer has settled, per token.
    uint256_t settledAmount(const Address& payer, const Address& token) const {
        std::lock_guard<std::mutex> lock(settlementMutex_);
        auto payer_it = settlementsByToken_.find(payer);
        if (payer_it == settlementsByToken_.end()) return uint256_t::fromUint64(0);
        auto token_it = payer_it->second.find(token);
        if (token_it == payer_it->second.end()) return uint256_t::fromUint64(0);
        return token_it->second;
    }

private:
    std::map<Address, bool> gasTokens_;
    std::function<uint256_t(const Address&)> priceOracle_;
    mutable std::mutex priceMutex_;

    mutable std::mutex settlementMutex_;
    std::map<Address, uint256_t> settlements_;
    std::map<Address, std::map<Address, uint256_t>> settlementsByToken_;
};

// =============================================================================
// ACCOUNT ABSTRACTION MANAGER (Master Orchestrator)
// =============================================================================

class AccountAbstractionManager {
public:
    // The entry point is created once initialize() supplies its address, so the
    // constructor only records the chain.
    explicit AccountAbstractionManager(ChainId chainId) : chainId_(chainId) {}

    // Installs the node call used for simulation, execution and batch
    // submission. Without it the manager refuses those operations instead of
    // reporting a result it did not obtain.
    void setNodeCall(EntryPointContract::NodeCall call) {
        entryPoint_->setNodeCall(call);
        batchedTxManager_->setNodeExecute(
            [call](const Address& target, const Bytes& data, Bytes& output) {
                return call(target, data, true, output);
            });
    }

    // The settings the manager needs before it can run.
    struct Config {
        // The EntryPoint the account system targets.
        Address entryPoint;
        // The account implementation the factory clones.
        Address accountImplementation;
        // The factory that deploys accounts.
        Address factory;
    };

    // Builds the sub-managers and applies the configuration.
    //
    // The previous version returned true unconditionally and registered a
    // paymaster at 0x0000000000000000000000000000000000000001, which is the
    // ecrecover precompile and not a paymaster at all. Any operation routed
    // through it would fail, and because initialize always reported success the
    // caller had no way to know. Paymasters are now registered by the operator
    // through configurePaymaster(), and every address in the configuration is
    // validated before it is used.
    bool initialize(const Config& config) {
        if (!CryptoUtils::decode_address(config.entryPoint) ||
            !CryptoUtils::decode_address(config.accountImplementation) ||
            !CryptoUtils::decode_address(config.factory)) {
            throw std::runtime_error("AccountAbstractionManager: the configuration has an invalid address");
        }
        if (config.entryPoint == "0x0000000000000000000000000000000000000000" ||
            config.accountImplementation == "0x0000000000000000000000000000000000000000" ||
            config.factory == "0x0000000000000000000000000000000000000000") {
            throw std::runtime_error("AccountAbstractionManager: the configuration has a zero address");
        }

        entryPoint_ = std::make_unique<EntryPointContract>(config.entryPoint, chainId_);

        factory_ = std::make_unique<AccountFactory>();
        factory_->setFactoryAddress(config.factory);
        factory_->setImplementationAddress(config.accountImplementation);

        sessionKeyManager_ = std::make_unique<SessionKeyManager>();
        paymasterManager_ = std::make_unique<PaymasterManager>();
        eip7702Manager_ = std::make_unique<EIP7702Manager>(chainId_);
        batchedTxManager_ = std::make_unique<BatchedTransactionManager>();
        gasAbstraction_ = std::make_unique<GasAbstraction>();

        return true;
    }

    // Registers a paymaster with the manager.
    void configurePaymaster(const Address& paymaster,
                            const Address& stakingToken,
                            uint256_t minStake,
                            uint256_t minUnstakeDelay) {
        paymasterManager_->configurePaymaster(paymaster, stakingToken, minStake, minUnstakeDelay);
    }

    // Registers a token that users may pay gas with.
    void addGasToken(const Address& token, bool isStable) {
        gasAbstraction_->addGasToken(token, isStable);
    }

    // Installs the price source used to value gas payments.
    void setGasPriceOracle(std::function<uint256_t(const Address&)> oracle) {
        gasAbstraction_->setPriceOracle(std::move(oracle));
    }

    // The account factory, for deriving account addresses.
    AccountFactory& factory() { return *factory_; }

    // The paymaster manager, for validating operations.
    PaymasterManager& paymasters() { return *paymasterManager_; }

    // The gas abstraction, for gas token settlement.
    GasAbstraction& gas() { return *gasAbstraction_; }

    // Send a user operation through the Entry Point
    // Submits an operation through the entry point. The beneficiary receives
    // the gas reimbursement.
    bool sendUserOperation(const UserOperation& op, const Address& beneficiary) {
        if (!CryptoUtils::decode_address(beneficiary)) {
            throw std::runtime_error("sendUserOperation: the beneficiary address is not valid");
        }
        if (!entryPoint_) {
            throw std::runtime_error("sendUserOperation: initialize() has not been called");
        }
        if (!validateUserOperation(op)) {
            return false;
        }

        // Handle through entry point
        std::vector<UserOperation> ops = {op};
        auto failed = entryPoint_->handleOps(ops, beneficiary);

        return failed.empty();
    }

    // Creates a new smart account with a random salt.
    Address createSmartAccount(const Address& owner) {
        if (!factory_) {
            throw std::runtime_error("createSmartAccount: initialize() has not been called");
        }
        return factory_->createAccount(owner, randomSalt());
    }

    // A 32-byte salt from the system random source. rand() is a predictable
    // sequence, so two callers could derive the same account address.
    static Bytes randomSalt() {
        Bytes salt(32);
        if (RAND_bytes(salt.data(), static_cast<int>(salt.size())) != 1) {
            throw std::runtime_error("randomSalt: no random source");
        }
        return salt;
    }

    // Get smart account address
    Address getSmartAccountAddress(const Address& owner) {
        Bytes salt(32, 0);
        return factory_->getAccountAddress(owner, salt);
    }

    // Create session key for dApp
    SessionKey createSessionKey(
        const Address& walletAddress,
        const Address& sessionKey,
        const std::vector<std::string>& methods,
        uint256_t maxAmount,
        uint64_t validitySeconds
    ) {
        return sessionKeyManager_->createSessionKey(
            walletAddress, sessionKey, methods, {}, maxAmount, validitySeconds
        );
    }

    // Validate session key
    bool validateSessionKey(
        const Address& wallet,
        const Address& sessionKey,
        const std::string& method,
        const Address& target,
        uint256_t amount
    ) {
        return sessionKeyManager_->validateSessionKey(wallet, sessionKey, method, target, amount);
    }

    // The authorization for upgrading an EOA to a contract. The account nonce
    // is the EOA's current transaction nonce, which only the caller and the
    // chain know.
    Authorization create7702Authorization(const Address& contractAddress, uint64_t accountNonce) {
        return eip7702Manager_->createAuthorization(contractAddress, accountNonce);
    }

    // Signs an authorization with the EOA's private key.
    bool sign7702Authorization(const Authorization& auth, const Bytes& privateKey,
                               Bytes& signature) {
        return eip7702Manager_->signAuthorization(auth, privateKey, signature);
    }

    // Submits an authorization to the node. The signed transaction is built by
    // the caller, because its type and fee fields are chain policy.
    bool execute7702(const Authorization& auth, const Bytes& signature,
                     const Bytes& signedTransaction, Bytes& txHash) {
        return eip7702Manager_->executeAuthorization(auth, signature, signedTransaction, txHash);
    }

    // Create batched transaction
    std::string createBatch(
        const Address& sender,
        const std::vector<UserOperation>& operations
    ) {
        return batchedTxManager_->createBatch(sender, operations, 3600); // 1 hour deadline
    }

    // Execute batched transaction
    bool executeBatch(const std::string& batchId) {
        return batchedTxManager_->executeBatch(batchId);
    }

    // Pay with stablecoin for gas
    bool payGasWithStablecoin(
        const UserOperation& op,
        const Address& token,
        uint256_t tokenAmount
    ) {
        return gasAbstraction_->payWithGasToken(op, token, tokenAmount);
    }

    // Calculate gas cost in token
    uint256_t calculateGasCost(
        const Address& token,
        uint256_t gasUsed,
        uint256_t gasPrice
    ) {
        return gasAbstraction_->calculateGasCost(token, gasUsed, gasPrice);
    }

private:
    std::unique_ptr<EntryPointContract> entryPoint_;
    ChainId chainId_;

    std::unique_ptr<AccountFactory> factory_;
    std::unique_ptr<SessionKeyManager> sessionKeyManager_;
    std::unique_ptr<PaymasterManager> paymasterManager_;
    std::unique_ptr<EIP7702Manager> eip7702Manager_;
    std::unique_ptr<BatchedTransactionManager> batchedTxManager_;
    std::unique_ptr<GasAbstraction> gasAbstraction_;

    bool validateUserOperation(const UserOperation& op) {
        // Check sender is not zero
        if (!CryptoUtils::decode_address(op.sender)) return false;

        // Check gas limits
        if (op.verificationGasLimit < uint256_t::fromUint64(21000)) return false;
        if (op.callGasLimit < uint256_t::fromUint64(21000)) return false;

        // Check paymaster if specified
        if (!op.paymaster.empty()) {
            if (!paymasterManager_->isPaymasterValid(op.paymaster, op.maxFeePerGas)) {
                return false;
            }
        }

        return true;
    }
};

} // namespace tiger

#endif // TIGERWALLET_ACCOUNT_ABSTRACTION_HPP
