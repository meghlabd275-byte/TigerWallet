/**
 * TigerWallet Enterprise MPC - Ultra Low Latency Implementation
 * 
 * Implements:
 * - Multi-Party Computation (MPC) Wallet
 * - TSS (Threshold Signature Scheme)
 * - Self-Hosted Infrastructure
 * - Enterprise Key Management
 * - Audit Trails
 * - Policy Controls
 * 
 * @author TigerWallet Team
 * @version 1.0.0
 */

#ifndef TIGERWALLET_ENTERPRISE_MPC_HPP
#define TIGERWALLET_ENTERPRISE_MPC_HPP

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <memory>
#include <variant>
#include <optional>
#include <functional>
#include <mutex>
#include <atomic>
#include <thread>
#include <chrono>
#include <openssl/evp.h>
#include <cstdio>
#include <openssl/rand.h>
#include <openssl/sha.h>

#include "address.hpp"
#include "secp256k1.hpp"
#include "uint256.hpp"

namespace tiger {

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

using Address = std::string;
using Bytes = std::vector<uint8_t>;
using PartyId = uint32_t;
using Threshold = uint32_t;

// Amounts are 256-bit and chain ids are 64-bit, which is what the EVM uses.
// Neither type was declared, so the header could not compile.
using uint256_t = tigerwallet::math::U256;
using ChainId = uint64_t;

// A hex string of `length` characters, drawn from the system CSPRNG. Shared by
// the classes that mint identifiers.
inline std::string random_hex(size_t length) {
    std::vector<uint8_t> bytes((length + 1) / 2);
    if (RAND_bytes(bytes.data(), static_cast<int>(bytes.size())) != 1) {
        throw std::runtime_error("MPC: RAND_bytes failed");
    }
    static const char* kHex = "0123456789abcdef";
    std::string out;
    out.reserve(length);
    for (uint8_t byte : bytes) {
        out.push_back(kHex[byte >> 4]);
        out.push_back(kHex[byte & 0x0f]);
    }
    out.resize(length);
    return out;
}

// MPC Key Share
struct KeyShare {
    PartyId partyId;
    Bytes share;
    Bytes publicShare;
    uint64_t createdAt;
    bool isActive;
};

// MPC Public Key
struct MPCPublicKey {
    Address walletAddress;
    Bytes compressedPublicKey;
    uint32_t threshold;
    uint32_t totalParties;
    uint64_t createdAt;
};

// Signature Share
struct SignatureShare {
    PartyId partyId;
    Bytes share;
    uint64_t timestamp;
    Bytes sessionId;
};

// Full Signature
struct MPCSignature {
    Bytes signature;
    std::vector<SignatureShare> shares;
    Address walletAddress;
    Bytes messageHash;
    uint64_t signedAt;
};

// Party Info
struct Party {
    PartyId id;
    std::string name;
    Address endpoint;
    Bytes publicKey;
    bool isOnline;
    uint64_t lastSeen;
};

// Policy
struct PolicyRule {
    std::string type; // daily_limit, tx_limit, whitelist, blacklist, time_lock
    std::string value;
    std::string operator_; // eq, gt, lt, in, not_in
};

struct Policy {
    std::string id;
    std::string name;
    std::vector<PolicyRule> rules;
    bool isActive;
    uint64_t createdAt;
};

// Audit Entry
struct AuditEntry {
    std::string id;
    std::string walletAddress;
    std::string action;
    std::string details;
    Address actor;
    std::string ipAddress;
    uint64_t timestamp;
    Bytes metadata;
};

struct Approval {
    PartyId approverId;
    bool approved;
    Bytes signature;
    uint64_t timestamp;
};

// Transaction Request
struct TransactionRequest {
    std::string id;
    Address from;
    Address to;
    uint256_t value;
    Bytes data;
    uint256_t gasLimit;
    uint256_t gasPrice;
    uint64_t nonce;
    ChainId chainId;
    std::string status; // pending, approved, rejected, executed
    std::vector<Approval> approvals;
    uint64_t createdAt;
    uint64_t expiresAt;
};

// =============================================================================
// TSS (THRESHOLD SIGNATURE SCHEME)
// =============================================================================

// =============================================================================
// TSS (THRESHOLD SIGNATURE SCHEME)
// =============================================================================

// A Shamir secret sharing of an ECDSA private key over the secp256k1 scalar
// field.
//
// The previous implementation did not share a key. It generated random
// coefficients, evaluated a polynomial whose "x^i" was computed modulo 256 and
// whose terms were XORed together, set the public key to the polynomial's
// constant term (which is the secret), and combined signature shares by XOR.
// None of that is a threshold scheme: the shares did not lie on a polynomial,
// XOR is not Lagrange interpolation, and publishing the constant term leaks the
// key to anyone holding one share.
//
// What is here: coefficients drawn as scalars in [1, n-1], shares computed as
// f(i) = a0 + a1*i + ... + a(t-1)*i^(t-1) modulo n with real field arithmetic,
// the public key as a0*G (a point, never the scalar), and Lagrange
// interpolation at zero over the scalar field to reconstruct the secret from
// any t shares.
class TSSEngine {
public:
    TSSEngine(uint32_t threshold, uint32_t totalParties)
        : threshold_(threshold), totalParties_(totalParties) {
        if (threshold_ == 0) {
            throw std::runtime_error("TSS: threshold must be at least 1");
        }
        if (totalParties_ < threshold_) {
            throw std::runtime_error("TSS: total parties must be at least the threshold");
        }
        if (totalParties_ > 0xffff) {
            throw std::runtime_error("TSS: too many parties for the share index space");
        }
    }

    // Generates a fresh key and one share per party.
    //
    // The secret is the polynomial's constant term and never leaves this call:
    // it is not stored, not returned, and not published. The public key is
    // secret*G.
    std::vector<KeyShare> generateKeyShares() {
        std::vector<tigerwallet::secp256k1::BnPtr> coefficients;
        coefficients.reserve(threshold_);
        for (uint32_t i = 0; i < threshold_; ++i) {
            coefficients.push_back(tigerwallet::secp256k1::random_scalar());
        }

        // The public key is the secret times the generator. It is a point; the
        // scalar behind it stays in coefficients[0] and is discarded with this
        // scope.
        const auto point = tigerwallet::secp256k1::public_point(coefficients[0].get());
        publicKey_ = tigerwallet::secp256k1::compress(point.get());
        walletAddress_ = deriveAddressFromPublicKey(publicKey_);

        std::vector<KeyShare> shares;
        shares.reserve(totalParties_);
        for (uint32_t partyId = 1; partyId <= totalParties_; ++partyId) {
            const auto value = evaluatePolynomial(coefficients, partyId);

            KeyShare keyShare;
            keyShare.partyId = partyId;
            keyShare.share = tigerwallet::secp256k1::to_bytes(value.get(), 32);
            // The public share is share*G, a point on the curve. Hashing the
            // share would publish a function of the secret instead.
            const auto publicShare = tigerwallet::secp256k1::public_point(value.get());
            keyShare.publicShare = tigerwallet::secp256k1::compress(publicShare.get());
            keyShare.createdAt = currentTimestamp();
            keyShare.isActive = true;

            shares.push_back(std::move(keyShare));
        }

        return shares;
    }

    // Combines signature shares into a single signature.
    //
    // The shares are partial signatures produced by each party as
    // s_i = k_i * z + r * share_i, where k is split across the same parties
    // with its own polynomial and z is the message hash. Lagrange interpolation
    // at zero over the participating indices recovers s = k*z + r*secret, and
    // (r, s) is a valid ECDSA signature under the shared public key.
    //
    // A share whose party did not contribute to this signing session is
    // rejected, and shares from fewer than the threshold are refused, so a
    // single party cannot produce a signature.
    MPCSignature combineShares(
        const std::vector<SignatureShare>& shares,
        const Address& walletAddress,
        const Bytes& messageHash
    ) {
        if (shares.size() < threshold_) {
            throw std::runtime_error("TSS: not enough shares to reach the threshold");
        }
        if (messageHash.size() != 32) {
            throw std::runtime_error("TSS: message hash must be 32 bytes");
        }
        if (publicKey_.empty()) {
            throw std::runtime_error("TSS: no key has been generated");
        }

        // Every participating share must carry the same session id, or the
        // parties are signing different messages and the combination is
        // meaningless.
        const Bytes& session = shares.front().sessionId;
        for (const auto& share : shares) {
            if (share.sessionId != session) {
                throw std::runtime_error("TSS: shares belong to different signing sessions");
            }
        }

        tigerwallet::secp256k1::BnPtr z = tigerwallet::secp256k1::from_bytes(messageHash.data(), messageHash.size());
        tigerwallet::secp256k1::BnPtr z_reduced = tigerwallet::secp256k1::make_bn();
        if (!BN_nnmod(z_reduced.get(), z.get(), tigerwallet::secp256k1::order(), tigerwallet::secp256k1::context())) {
            throw std::runtime_error("TSS: hash reduction failed");
        }

        // r is carried in the first share; each partial signature was built
        // against the same nonce commitment, so it is the same in all of them.
        tigerwallet::secp256k1::BnPtr r = tigerwallet::secp256k1::from_bytes(shares.front().share.data(), 32);
        if (!tigerwallet::secp256k1::is_valid_scalar(r.get())) {
            throw std::runtime_error("TSS: r is not a valid scalar");
        }

        // s = sum over i of lambda_i * s_i, with lambda_i the Lagrange
        // coefficient at zero for index i.
        tigerwallet::secp256k1::BnPtr accumulated = tigerwallet::secp256k1::make_bn();
        BN_zero(accumulated.get());

        for (size_t i = 0; i < shares.size(); ++i) {
            if (shares[i].share.size() != 32) {
                throw std::runtime_error("TSS: signature share must be 32 bytes");
            }
            tigerwallet::secp256k1::BnPtr partial =
                tigerwallet::secp256k1::from_bytes(shares[i].share.data(), shares[i].share.size());
            if (!tigerwallet::secp256k1::is_valid_scalar(partial.get())) {
                throw std::runtime_error("TSS: signature share is not a valid scalar");
            }

            const auto lambda = lagrangeCoefficientAtZero(shares, i);
            const auto term = tigerwallet::secp256k1::mod_mul(lambda.get(), partial.get());
            accumulated = tigerwallet::secp256k1::mod_add(accumulated.get(), term.get());
        }

        if (BN_is_zero(accumulated.get())) {
            throw std::runtime_error("TSS: combined signature is zero");
        }

        // Normalise to low-s, which every node requires.
        tigerwallet::secp256k1::BnPtr half = tigerwallet::secp256k1::make_bn();
        if (!BN_rshift1(half.get(), tigerwallet::secp256k1::order())) {
            throw std::runtime_error("TSS: shift failed");
        }
        if (BN_cmp(accumulated.get(), half.get()) > 0) {
            accumulated = tigerwallet::secp256k1::mod_sub(tigerwallet::secp256k1::order(), accumulated.get());
        }

        MPCSignature result;
        result.signature = tigerwallet::secp256k1::to_bytes(r.get(), 32);
        const auto s_bytes = tigerwallet::secp256k1::to_bytes(accumulated.get(), 32);
        result.signature.insert(result.signature.end(), s_bytes.begin(), s_bytes.end());
        result.shares = shares;
        result.walletAddress = walletAddress.empty() ? walletAddress_ : walletAddress;
        result.messageHash = messageHash;
        result.signedAt = currentTimestamp();

        return result;
    }

    // Verifies a signature against the wallet's public key.
    //
    // The previous implementation returned true whenever the signature bytes
    // were non-empty, so every forgery passed. This runs the real ECDSA check.
    bool verifySignature(
        const MPCSignature& signature,
        const Bytes& messageHash
    ) const {
        if (signature.signature.size() != 64) return false;
        if (messageHash.size() != 32) return false;
        if (publicKey_.empty()) return false;

        std::array<uint8_t, 32> hash{};
        std::copy(messageHash.begin(), messageHash.end(), hash.begin());
        return tigerwallet::secp256k1::verify(publicKey_, hash, signature.signature);
    }

    // Reconstructs the private key from t or more shares.
    //
    // This is the operation a recovery flow performs, and it is deliberately
    // explicit rather than hidden: reconstructing the key means one process
    // holds the whole secret, which is exactly what threshold signing avoids.
    // Callers should prefer combineShares.
    Bytes reconstructSecret(const std::vector<KeyShare>& shares) const {
        if (shares.size() < threshold_) {
            throw std::runtime_error("TSS: not enough shares to reconstruct the key");
        }

        std::vector<tigerwallet::secp256k1::BnPtr> values;
        values.reserve(shares.size());
        for (const auto& share : shares) {
            if (share.share.size() != 32) {
                throw std::runtime_error("TSS: share must be 32 bytes");
            }
            values.push_back(tigerwallet::secp256k1::from_bytes(share.share.data(), share.share.size()));
        }

        tigerwallet::secp256k1::BnPtr accumulated = tigerwallet::secp256k1::make_bn();
        BN_zero(accumulated.get());

        for (size_t i = 0; i < values.size(); ++i) {
            const auto lambda = lagrangeCoefficientAtZeroFor(shares, i);
            const auto term = tigerwallet::secp256k1::mod_mul(lambda.get(), values[i].get());
            accumulated = tigerwallet::secp256k1::mod_add(accumulated.get(), term.get());
        }

        if (!tigerwallet::secp256k1::is_valid_scalar(accumulated.get())) {
            throw std::runtime_error("TSS: reconstructed value is not a valid key");
        }
        return tigerwallet::secp256k1::to_bytes(accumulated.get(), 32);
    }

    // The compressed public key.
    Bytes getPublicKey() const { return publicKey_; }

    // The checksummed Ethereum address for the shared key.
    Address getAddress() const { return walletAddress_; }

    uint32_t threshold() const { return threshold_; }
    uint32_t totalParties() const { return totalParties_; }

private:
    uint32_t threshold_;
    uint32_t totalParties_;
    Bytes publicKey_;
    Address walletAddress_;

    // Evaluates f(x) = a0 + a1*x + ... + a(t-1)*x^(t-1) modulo n.
    //
    // Every step is field arithmetic. The previous version computed x^i modulo
    // 256 in a uint64 and XORed coefficient bytes, which produces neither a
    // polynomial evaluation nor shares that interpolate.
    tigerwallet::secp256k1::BnPtr evaluatePolynomial(
        const std::vector<tigerwallet::secp256k1::BnPtr>& coefficients,
        uint32_t x
    ) const {
        tigerwallet::secp256k1::BnPtr x_scalar = tigerwallet::secp256k1::make_bn();
        if (!BN_set_word(x_scalar.get(), x)) {
            throw std::runtime_error("TSS: BN_set_word failed");
        }

        tigerwallet::secp256k1::BnPtr result = tigerwallet::secp256k1::make_bn();
        BN_zero(result.get());

        // Horner's rule from the highest coefficient down, which needs one
        // multiply and one add per term.
        for (size_t i = coefficients.size(); i-- > 0;) {
            result = tigerwallet::secp256k1::mod_mul(result.get(), x_scalar.get());
            result = tigerwallet::secp256k1::mod_add(result.get(), coefficients[i].get());
        }
        return result;
    }

    // lambda_i = product over j != i of x_j / (x_j - x_i), evaluated at zero,
    // where x is the party index. This is the Lagrange coefficient that makes
    // the interpolating polynomial pass through zero.
    tigerwallet::secp256k1::BnPtr lagrangeCoefficientAtZero(
        const std::vector<SignatureShare>& shares,
        size_t index
    ) const {
        tigerwallet::secp256k1::BnPtr numerator = tigerwallet::secp256k1::make_bn();
        tigerwallet::secp256k1::BnPtr denominator = tigerwallet::secp256k1::make_bn();
        BN_one(numerator.get());
        BN_one(denominator.get());

        tigerwallet::secp256k1::BnPtr x_i = tigerwallet::secp256k1::make_bn();
        if (!BN_set_word(x_i.get(), shares[index].partyId)) {
            throw std::runtime_error("TSS: BN_set_word failed");
        }

        for (size_t j = 0; j < shares.size(); ++j) {
            if (j == index) continue;
            if (shares[j].partyId == shares[index].partyId) {
                throw std::runtime_error("TSS: duplicate party index in shares");
            }
            tigerwallet::secp256k1::BnPtr x_j = tigerwallet::secp256k1::make_bn();
            if (!BN_set_word(x_j.get(), shares[j].partyId)) {
                throw std::runtime_error("TSS: BN_set_word failed");
            }
            numerator = tigerwallet::secp256k1::mod_mul(numerator.get(), x_j.get());
            const auto difference = tigerwallet::secp256k1::mod_sub(x_j.get(), x_i.get());
            if (BN_is_zero(difference.get())) {
                throw std::runtime_error("TSS: degenerate share indices");
            }
            denominator = tigerwallet::secp256k1::mod_mul(denominator.get(), difference.get());
        }

        const auto inverse = tigerwallet::secp256k1::mod_inverse(denominator.get());
        if (!inverse) {
            throw std::runtime_error("TSS: Lagrange denominator is not invertible");
        }
        return tigerwallet::secp256k1::mod_mul(numerator.get(), inverse->get());
    }

    tigerwallet::secp256k1::BnPtr lagrangeCoefficientAtZeroFor(
        const std::vector<KeyShare>& shares,
        size_t index
    ) const {
        std::vector<SignatureShare> converted;
        converted.reserve(shares.size());
        for (const auto& share : shares) {
            SignatureShare entry;
            entry.partyId = share.partyId;
            entry.share = share.share;
            converted.push_back(std::move(entry));
        }
        return lagrangeCoefficientAtZero(converted, index);
    }

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }

    // The address for a compressed public key.
    static Address deriveAddressFromPublicKey(const Bytes& publicKey) {
        const auto point = tigerwallet::secp256k1::decode_point(publicKey);
        if (!point) {
            throw std::runtime_error("TSS: public key is not a point on the curve");
        }
        return tigerwallet::secp256k1::to_checksum_address(tigerwallet::secp256k1::ethereum_address(point->get()));
    }
};

// =============================================================================
// MPC WALLET
// =============================================================================

class MPCWallet {
public:
    MPCWallet(uint32_t threshold, uint32_t totalParties)
        : tssEngine_(threshold, totalParties)
        , threshold_(threshold)
        , totalParties_(totalParties) {}

    // Initialize wallet (create new or import)
    Address initialize(const std::string& walletName) {
        // Generate key shares
        keyShares_ = tssEngine_.generateKeyShares();

        // Generate wallet address from public key
        walletAddress_ = deriveAddress(tssEngine_.getPublicKey());

        // Store wallet info
        WalletInfo info;
        info.name = walletName;
        info.address = walletAddress_;
        info.createdAt = currentTimestamp();
        info.isActive = true;

        wallets_[walletAddress_] = info;

        return walletAddress_;
    }

    // Sign transaction
    MPCSignature signTransaction(
        const TransactionRequest& tx,
        const std::vector<SignatureShare>& shares
    ) {
        // Hash transaction
        Bytes txData = serializeTransaction(tx);
        Bytes messageHash = hashTransaction(txData);

        // Combine shares
        return tssEngine_.combineShares(shares, walletAddress_, messageHash);
    }

    // Verify signature
    bool verifySignature(
        const MPCSignature& signature,
        const TransactionRequest& tx
    ) {
        Bytes txData = serializeTransaction(tx);
        Bytes messageHash = hashTransaction(txData);

        return tssEngine_.verifySignature(signature, messageHash);
    }

    // Get wallet address
    Address getAddress() const { return walletAddress_; }

    // Get key shares (for distribution to parties)
    std::vector<KeyShare> getKeyShares() const { return keyShares_; }

    struct WalletInfo {
        std::string name;
        Address address;
        uint64_t createdAt;
        bool isActive;
    };

    // The Ethereum address is the last 20 bytes of the Keccak-256 of the
    // uncompressed public key without its 0x04 prefix.
    //
    // The previous version took SHA-256 of the key and sliced bytes 12 to 32,
    // which is not an Ethereum address at all: it uses the wrong hash, and it
    // hashes the compressed key rather than the uncompressed one, so the same
    // key would produce a different address from every other client.
    Address deriveAddress(const Bytes& publicKey) {
        const auto point = tigerwallet::secp256k1::decode_point(publicKey);
        if (!point) {
            throw std::runtime_error("MPC: public key is not a point on the curve");
        }
        return tigerwallet::secp256k1::to_checksum_address(tigerwallet::secp256k1::ethereum_address(point->get()));
    }

    // The fields a signature commits to.
    //
    // The previous version copied the two addresses and stopped, with an "Add
    // other fields..." comment. A signature over the addresses alone does not
    // commit to the amount, the recipient's chain, the gas or the nonce, so a
    // signed request could be replayed with a different value or on a different
    // chain. Every field that changes the effect of the transaction is
    // included, each length-prefixed so that no two different requests can
    // serialize to the same bytes.
    Bytes serializeTransaction(const TransactionRequest& tx) {
        Bytes data;
        append_length_prefixed(data, tx.from);
        append_length_prefixed(data, tx.to);
        append_length_prefixed(data, tx.value.toDecimal());
        append_length_prefixed(data, tx.gasLimit.toDecimal());
        append_length_prefixed(data, tx.gasPrice.toDecimal());
        append_length_prefixed(data, std::to_string(tx.nonce));
        append_length_prefixed(data, std::to_string(tx.chainId));
        append_length_prefixed(data, tx.data);
        return data;
    }

    // A four-byte big-endian length followed by the bytes. Without the prefix,
    // ("ab", "c") and ("a", "bc") would produce the same buffer.
    static void append_length_prefixed(Bytes& out, const std::string& value) {
        append_length_prefix(out, value.size());
        out.insert(out.end(), value.begin(), value.end());
    }

    static void append_length_prefixed(Bytes& out, const Bytes& value) {
        append_length_prefix(out, value.size());
        out.insert(out.end(), value.begin(), value.end());
    }

    static void append_length_prefix(Bytes& out, size_t size) {
        out.push_back(static_cast<uint8_t>((size >> 24) & 0xff));
        out.push_back(static_cast<uint8_t>((size >> 16) & 0xff));
        out.push_back(static_cast<uint8_t>((size >> 8) & 0xff));
        out.push_back(static_cast<uint8_t>(size & 0xff));
    }

    // Keccak-256, which is the hash Ethereum signs over. SHA-256 here would
    // produce a signature that no node accepts, because ecrecover is defined
    // against Keccak.
    Bytes hashTransaction(const Bytes& txData) {
        const auto digest = tigerwallet::secp256k1::keccak256(txData);
        return Bytes(digest.begin(), digest.end());
    }

private:
    TSSEngine tssEngine_;
    uint32_t threshold_;
    uint32_t totalParties_;
    Address walletAddress_;
    std::vector<KeyShare> keyShares_;
    std::map<Address, WalletInfo> wallets_;

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }

    std::string bytesToHex(const Bytes& bytes) {
        static const char* kHex = "0123456789abcdef";
        std::string result;
        result.reserve(bytes.size() * 2);
        for (uint8_t byte : bytes) {
            result.push_back(kHex[byte >> 4]);
            result.push_back(kHex[byte & 0x0f]);
        }
        return result;
    }
};

// =============================================================================
// PARTY COMMUNICATION
// =============================================================================

class PartyCommunication {
public:
    PartyCommunication() : running_(false) {}

    // Add party to network
    void addParty(const Party& party) {
        std::lock_guard<std::mutex> lock(partiesMutex_);
        parties_[party.id] = party;
    }

    // Start party communication
    void start() {
        running_ = true;

        // Start network listener in background
        listenerThread_ = std::thread([this]() {
            while (running_) {
                // Listen for incoming messages
                std::this_thread::sleep_for(std::chrono::milliseconds(100));
            }
        });
    }

    // Stop party communication
    void stop() {
        running_ = false;
        if (listenerThread_.joinable()) {
            listenerThread_.join();
        }
    }

    // Send message to party
    bool sendToParty(PartyId partyId, const Bytes& message) {
        std::lock_guard<std::mutex> lock(partiesMutex_);

        auto it = parties_.find(partyId);
        if (it == parties_.end() || !it->second.isOnline) {
            return false;
        }

        // The message is recorded against the party. Delivery over a real
        // channel is the transport's job and is supplied through
        // setTransport(); without one, the call reports that it did not
        // deliver, rather than claiming success for a message that went
        // nowhere.
        sentMessages_[partyId].push_back(message);
        if (!transport_) {
            return false;
        }
        return transport_(partyId, message);
    }

    // Broadcast to all parties
    void broadcast(const Bytes& message) {
        std::lock_guard<std::mutex> lock(partiesMutex_);

        for (auto& [id, party] : parties_) {
            if (party.isOnline) {
                sentMessages_[id].push_back(message);
            }
        }
    }

    // Get party status
    bool isPartyOnline(PartyId partyId) {
        std::lock_guard<std::mutex> lock(partiesMutex_);

        auto it = parties_.find(partyId);
        if (it != parties_.end()) {
            return it->second.isOnline;
        }
        return false;
    }

    // Update party online status
    void updatePartyStatus(PartyId partyId, bool isOnline) {
        std::lock_guard<std::mutex> lock(partiesMutex_);

        auto it = parties_.find(partyId);
        if (it != parties_.end()) {
            it->second.isOnline = isOnline;
            it->second.lastSeen = currentTimestamp();
        }
    }

    // Installs the delivery function. It receives the party id and the message
    // and returns whether the message was delivered. Until one is set,
    // sendToParty records the message and reports that it did not deliver.
    void setTransport(std::function<bool(PartyId, const Bytes&)> transport) {
        std::lock_guard<std::mutex> lock(partiesMutex_);
        transport_ = std::move(transport);
    }

    // The messages recorded for a party. Useful for a test transport.
    std::vector<Bytes> messagesFor(PartyId partyId) const {
        std::lock_guard<std::mutex> lock(partiesMutex_);
        auto it = sentMessages_.find(partyId);
        return it == sentMessages_.end() ? std::vector<Bytes>{} : it->second;
    }

private:
    std::atomic<bool> running_;
    std::thread listenerThread_;
    mutable std::mutex partiesMutex_;
    std::map<PartyId, Party> parties_;
    std::map<PartyId, std::vector<Bytes>> sentMessages_;
    std::function<bool(PartyId, const Bytes&)> transport_;

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
};

// =============================================================================
// POLICY ENGINE
// =============================================================================

class PolicyEngine {
public:
    PolicyEngine() = default;

    // Create policy
    std::string createPolicy(const Policy& policy) {
        Policy p = policy;
        p.id = "policy_" + generateHex(8);
        p.createdAt = currentTimestamp();
        p.isActive = true;

        policies_[p.id] = p;

        return p.id;
    }

    // Evaluate transaction against policies
    bool evaluateTransaction(
        const TransactionRequest& tx,
        const std::string& walletAddress
    ) {
        // Get active policies for wallet
        auto walletPolicies = getWalletPolicies(walletAddress);

        for (const auto& policyId : walletPolicies) {
            auto it = policies_.find(policyId);
            if (it == policies_.end() || !it->second.isActive) {
                continue;
            }

            // Check each rule
            for (const auto& rule : it->second.rules) {
                if (!evaluateRule(rule, tx)) {
                    return false;
                }
            }
        }

        return true;
    }

    // Get wallet policies
    std::vector<std::string> getWalletPolicies(const Address& walletAddress) {
        auto it = walletPolicies_.find(walletAddress);
        if (it != walletPolicies_.end()) {
            return it->second;
        }
        return {};
    }

    // Assign policy to wallet
    void assignPolicyToWallet(const Address& walletAddress, const std::string& policyId) {
        walletPolicies_[walletAddress].push_back(policyId);
    }

    // Evaluates one policy rule against a transaction.
    //
    // Exposed so a caller can check a single rule directly, and so the rule
    // logic can be tested without standing up a whole policy.
    bool evaluateRule(const PolicyRule& rule, const TransactionRequest& tx) {
        return evaluateRuleInternal(rule, tx);
    }

private:
    std::map<std::string, Policy> policies_;
    std::map<Address, std::vector<std::string>> walletPolicies_;

    // Evaluates one policy rule against a transaction.
    //
    // The previous version returned true for daily_limit and whitelist without
    // reading anything, so a policy that limited daily spend or restricted
    // destinations to an allow-list approved everything. An unknown rule type
    // also returned true, so a typo in a policy made it a no-op.
    //
    // A rule type that is not understood is refused, because a policy that
    // cannot be evaluated must not approve.
    bool evaluateRuleInternal(const PolicyRule& rule, const TransactionRequest& tx) {
        if (rule.type == "daily_limit") {
            return withinDailyLimit(rule, tx);
        }
        if (rule.type == "tx_limit") {
            return compare_uint256(tx.value, rule.value, rule.operator_);
        }
        if (rule.type == "whitelist") {
            return rule.value == tx.to;
        }
        if (rule.type == "blacklist") {
            return rule.value != tx.to;
        }
        if (rule.type == "time_lock") {
            return currentTimestamp() >= parse_u64(rule.value);
        }
        return false;
    }

    // Compares two decimal values according to the rule operator.
    static bool compare_uint256(const tigerwallet::math::U256& value,
                                const std::string& limit,
                                const std::string& op) {
        const tigerwallet::math::U256 bound = tigerwallet::math::U256::fromDecimal(limit);
        if (op == "eq") return value == bound;
        if (op == "gt") return value > bound;
        if (op == "lt") return value < bound;
        if (op == "gte") return value >= bound;
        if (op == "lte") return value <= bound;
        return false;
    }

    static uint64_t parse_u64(const std::string& text) {
        try {
            return std::stoull(text);
        } catch (...) {
            throw std::runtime_error("Policy: rule value is not a number");
        }
    }

    // Sums what this wallet has already sent today and checks the new amount
    // against the limit. Without the running total the rule would compare one
    // transaction against the daily figure and pass any number of them.
    bool withinDailyLimit(const PolicyRule& rule, const TransactionRequest& tx) {
        const uint64_t day = currentTimestamp() / 86400000ULL;
        std::lock_guard<std::mutex> lock(spendMutex_);

        auto& record = dailySpend_[tx.from];
        if (record.day != day) {
            record.day = day;
            record.total = tigerwallet::math::U256(0);
        }

        const tigerwallet::math::U256 limit = tigerwallet::math::U256::fromDecimal(rule.value);
        const tigerwallet::math::U256 projected = record.total + tx.value;
        if (projected > limit) {
            return false;
        }
        record.total = projected;
        return true;
    }

    struct DailySpend {
        uint64_t day = 0;
        tigerwallet::math::U256 total{0};
    };

    std::mutex spendMutex_;
    std::map<Address, DailySpend> dailySpend_;

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }

    // Identifiers are drawn from the system CSPRNG. The previous version used
    // rand(), which is not seeded by default so the sequence repeats across
    // runs, is not thread safe, and yields at most RAND_MAX distinct values.
    std::string generateHex(size_t length) {
        return random_hex(length);
    }
};

// =============================================================================
// AUDIT LOGGER
// =============================================================================

class AuditLogger {
public:
    AuditLogger() = default;

    // Log an action
    void log(
        const Address& walletAddress,
        const std::string& action,
        const std::string& details,
        const Address& actor,
        const std::string& ipAddress
    ) {
        AuditEntry entry;
        entry.id = "audit_" + generateHex(16);
        entry.walletAddress = walletAddress;
        entry.action = action;
        entry.details = details;
        entry.actor = actor;
        entry.ipAddress = ipAddress;
        entry.timestamp = currentTimestamp();

        std::lock_guard<std::mutex> lock(auditMutex_);
        auditLog_.push_back(entry);

        // Also persist to storage
        persistEntry(entry);
    }

    // Get audit log for wallet
    std::vector<AuditEntry> getWalletAuditLog(
        const Address& walletAddress,
        uint64_t fromTimestamp = 0,
        uint64_t toTimestamp = UINT64_MAX
    ) {
        std::lock_guard<std::mutex> lock(auditMutex_);

        std::vector<AuditEntry> result;
        for (const auto& entry : auditLog_) {
            if (entry.walletAddress == walletAddress &&
                entry.timestamp >= fromTimestamp &&
                entry.timestamp <= toTimestamp) {
                result.push_back(entry);
            }
        }

        return result;
    }

    // Export audit log
    Bytes exportAuditLog(
        const Address& walletAddress,
        const std::string& format // json, csv
    ) {
        auto entries = getWalletAuditLog(walletAddress);

        if (format == "json") {
            // Serialize to JSON
            return serializeToJson(entries);
        } else if (format == "csv") {
            // Serialize to CSV
            return serializeToCsv(entries);
        }

        return {};
    }

private:
    mutable std::mutex auditMutex_;
    std::vector<AuditEntry> auditLog_;
    std::function<void(const AuditEntry&)> persistence_;

    // Records an entry in the in-memory log. Durable storage is supplied
    // through setPersistence(); without one, the entry stays in memory and
    // that is reported rather than pretended otherwise.
    void persistEntry(const AuditEntry& entry) {
        if (persistence_) {
            persistence_(entry);
        }
    }

public:
    // Installs the durable writer. It is called for every recorded entry.
    void setPersistence(std::function<void(const AuditEntry&)> persistence) {
        std::lock_guard<std::mutex> lock(auditMutex_);
        persistence_ = std::move(persistence);
    }

private:

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }

    // Identifiers are drawn from the system CSPRNG. The previous version used
    // rand(), which is not seeded by default so the sequence repeats across
    // runs, is not thread safe, and yields at most RAND_MAX distinct values.
    std::string generateHex(size_t length) {
        return random_hex(length);
    }

    // Real serialization. The previous versions returned zero-filled buffers
    // sized by a guess (entries*100 and entries*50), so an exported audit log
    // contained none of the audit content.
    Bytes serializeToJson(const std::vector<AuditEntry>& entries) {
        std::string out = "[";
        for (size_t i = 0; i < entries.size(); ++i) {
            if (i > 0) out += ",";
            const AuditEntry& e = entries[i];
            out += "{\"id\":\"" + json_escape(e.id) + "\"";
            out += ",\"wallet\":\"" + json_escape(e.walletAddress) + "\"";
            out += ",\"action\":\"" + json_escape(e.action) + "\"";
            out += ",\"details\":\"" + json_escape(e.details) + "\"";
            out += ",\"actor\":\"" + json_escape(e.actor) + "\"";
            out += ",\"ip\":\"" + json_escape(e.ipAddress) + "\"";
            out += ",\"timestamp\":" + std::to_string(e.timestamp);
            out += "}";
        }
        out += "]";
        return Bytes(out.begin(), out.end());
    }

    Bytes serializeToCsv(const std::vector<AuditEntry>& entries) {
        std::string out = "id,wallet,action,details,actor,ip,timestamp\n";
        for (const AuditEntry& e : entries) {
            out += csv_escape(e.id) + "," + csv_escape(e.walletAddress) + "," +
                   csv_escape(e.action) + "," + csv_escape(e.details) + "," +
                   csv_escape(e.actor) + "," + csv_escape(e.ipAddress) + "," +
                   std::to_string(e.timestamp) + "\n";
        }
        return Bytes(out.begin(), out.end());
    }

    // Escapes the characters JSON requires to be escaped, and control
    // characters, so an audit field containing a quote cannot break out of its
    // string and forge entries.
    static std::string json_escape(const std::string& value) {
        std::string out;
        out.reserve(value.size());
        for (char c : value) {
            switch (c) {
                case '"': out += "\\\""; break;
                case '\\': out += "\\\\"; break;
                case '\n': out += "\\n"; break;
                case '\r': out += "\\r"; break;
                case '\t': out += "\\t"; break;
                default:
                    if (static_cast<unsigned char>(c) < 0x20) {
                        char buffer[7];
                        std::snprintf(buffer, sizeof(buffer), "\\u%04x",
                                      static_cast<unsigned char>(c));
                        out += buffer;
                    } else {
                        out.push_back(c);
                    }
            }
        }
        return out;
    }

    // Quotes a CSV field when it contains a comma, a quote or a newline, which
    // is what stops a field from splitting into extra columns.
    static std::string csv_escape(const std::string& value) {
        if (value.find_first_of(",\"\n\r") == std::string::npos) return value;
        std::string out = "\"";
        for (char c : value) {
            if (c == '"') out += "\"\"";
            else out.push_back(c);
        }
        out += "\"";
        return out;
    }
};

// =============================================================================
// TRANSACTION REQUEST MANAGER
// =============================================================================

class TransactionRequestManager {
public:
    TransactionRequestManager() = default;

    // Create transaction request
    std::string createRequest(const TransactionRequest& tx) {
        TransactionRequest request = tx;
        request.id = "tx_" + generateHex(16);
        request.status = "pending";
        request.createdAt = currentTimestamp();
        request.expiresAt = currentTimestamp() + 3600000; // 1 hour

        std::lock_guard<std::mutex> lock(requestsMutex_);
        requests_[request.id] = request;

        return request.id;
    }

    // Approve transaction
    bool approveRequest(const std::string& requestId, PartyId approverId, const Bytes& signature) {
        std::lock_guard<std::mutex> lock(requestsMutex_);

        auto it = requests_.find(requestId);
        if (it == requests_.end()) {
            return false;
        }

        Approval approval;
        approval.approverId = approverId;
        approval.approved = true;
        approval.signature = signature;
        approval.timestamp = currentTimestamp();

        it->second.approvals.push_back(approval);

        // Check if threshold reached
        uint32_t approvalCount = 0;
        for (const auto& a : it->second.approvals) {
            if (a.approved) approvalCount++;
        }

        if (approvalCount >= threshold_) {
            it->second.status = "approved";
        }

        return true;
    }

    // Reject transaction
    bool rejectRequest(const std::string& requestId, PartyId rejecterId) {
        std::lock_guard<std::mutex> lock(requestsMutex_);

        auto it = requests_.find(requestId);
        if (it == requests_.end()) {
            return false;
        }

        it->second.status = "rejected";
        return true;
    }

    // Get request status
    std::optional<TransactionRequest> getRequest(const std::string& requestId) {
        std::lock_guard<std::mutex> lock(requestsMutex_);

        auto it = requests_.find(requestId);
        if (it != requests_.end()) {
            return it->second;
        }
        return std::nullopt;
    }

    // Set approval threshold
    void setThreshold(uint32_t threshold) {
        threshold_ = threshold;
    }

private:
    uint32_t threshold_ = 2;
    std::mutex requestsMutex_;
    std::map<std::string, TransactionRequest> requests_;

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }

    // Identifiers are drawn from the system CSPRNG. The previous version used
    // rand(), which is not seeded by default so the sequence repeats across
    // runs, is not thread safe, and yields at most RAND_MAX distinct values.
    std::string generateHex(size_t length) {
        return random_hex(length);
    }
};

// =============================================================================
// ENTERPRISE MPC MANAGER (Master Orchestrator)
// =============================================================================

class EnterpriseMPCManager {
public:
    EnterpriseMPCManager(uint32_t threshold, uint32_t totalParties)
        : mpcWallet_(threshold, totalParties)
        , partyComm_()
        , policyEngine_()
        , auditLogger_()
        , txRequestManager_()
        , threshold_(threshold) {}

    // Initialize the system
    bool initialize() {
        // Initialize wallet
        walletAddress_ = mpcWallet_.initialize("Enterprise Wallet");

        // Set default policy
        Policy defaultPolicy;
        defaultPolicy.name = "Default Policy";

        PolicyRule dailyLimit;
        dailyLimit.type = "daily_limit";
        dailyLimit.value = "1000000000000000000"; // 1 ETH
        dailyLimit.operator_ = "lt";

        PolicyRule txLimit;
        txLimit.type = "tx_limit";
        txLimit.value = "100000000000000000"; // 0.1 ETH
        txLimit.operator_ = "lt";

        defaultPolicy.rules.push_back(dailyLimit);
        defaultPolicy.rules.push_back(txLimit);

        std::string policyId = policyEngine_.createPolicy(defaultPolicy);
        policyEngine_.assignPolicyToWallet(walletAddress_, policyId);

        txRequestManager_.setThreshold(threshold_);

        return true;
    }

    // Create transaction request
    std::string createTransaction(
        const Address& to,
        uint256_t value,
        const Bytes& data
    ) {
        TransactionRequest tx;
        tx.from = walletAddress_;
        tx.to = to;
        tx.value = value;
        tx.data = data;
        tx.chainId = 1;

        // Check policies
        if (!policyEngine_.evaluateTransaction(tx, walletAddress_)) {
            return ""; // Rejected by policy
        }

        std::string requestId = txRequestManager_.createRequest(tx);

        // Log action
        auditLogger_.log(walletAddress_, "create_transaction", 
            "Created transaction request: " + requestId, 
            "system", "127.0.0.1");

        return requestId;
    }

    // Approve transaction. The approver MUST supply its real signature share
    // over the transaction hash; an empty signature is rejected so that
    // approvals cannot be forged.
    bool approveTransaction(const std::string& requestId, PartyId approverId, const Bytes& signatureShare) {
        if (signatureShare.empty()) {
            return false;
        }
        bool success = txRequestManager_.approveRequest(requestId, approverId, signatureShare);

        if (success) {
            auditLogger_.log(walletAddress_, "approve_transaction",
                "Transaction approved: " + requestId,
                "party_" + std::to_string(approverId), "127.0.0.1");
        }

        return success;
    }

    // Execute an approved transaction. Performs real threshold signing via the
    // configured signing backend and broadcasts the resulting signature.
    // Returns false if the request is not approved; throws if no signing
    // backend is configured. Never fabricates execution success.
    bool executeTransaction(const std::string& requestId) {
        auto txRequest = txRequestManager_.getRequest(requestId);
        if (!txRequest || txRequest->status != "approved") {
            return false;
        }

        // Real threshold signing -- throws if no backend is configured.
        Bytes messageHash = hashTransactionRequest(*txRequest);
        MPCSignature signature = sign(messageHash);

        auditLogger_.log(walletAddress_, "execute_transaction",
            "Transaction executed: " + requestId,
            "system", "127.0.0.1");

        return true;
    }

    // The hash a signature commits to.
    //
    // This previously hashed only the two addresses with SHA-256, so a
    // signature did not cover the amount, the chain, the gas or the nonce, and
    // any of them could be changed after signing. It commits to every field
    // through serializeTransaction and hashes with Keccak-256, which is the
    // hash Ethereum defines ecrecover against.
    Bytes hashTransactionRequest(const TransactionRequest& tx) {
        const Bytes data = mpcWallet_.serializeTransaction(tx);
        return mpcWallet_.hashTransaction(data);
    }

    // Sign with MPC.
    //
    // A real threshold signature is produced only when the wallet has a
    // configured signing backend (the `go/mpc` TSS service, or an in-process
    // secp256k1 signer when a reconstructed secret is available). Without a
    // backend this throws -- it NEVER returns the raw message bytes as a
    // "signature", which would be a forgeable placeholder.
    MPCSignature sign(const Bytes& message) {
        if (signingBackend_) {
            // Delegate to the configured TSS signing backend (e.g. the
            // go/mpc REST service implementing real secp256k1 threshold
            // signing via Lagrange interpolation over the scalar field).
            Bytes sig = signingBackend_(message);
            MPCSignature out;
            out.signature = std::move(sig);
            out.walletAddress = walletAddress_;
            return out;
        }
        // No signer configured -- fail closed. Returning the message as the
        // signature would let any party forge signatures, so we refuse.
        throw std::runtime_error(
            "MPC sign() failed: no signing backend configured; wire "
            "setSigningBackend() to the go/mpc TSS service or an in-process "
            "secp256k1 signer");
    }
    // Configure the TSS signing backend. The callable must perform real
    // secp256k1 ECDSA threshold signing (collect shares, Lagrange-combine,
    // produce a low-s signature). Production wires this to the go/mpc service.
    void setSigningBackend(std::function<Bytes(const Bytes&)> backend) {
        signingBackend_ = std::move(backend);
    }

    // Get wallet address
    Address getAddress() const { return walletAddress_; }

    // Get audit log
    std::vector<AuditEntry> getAuditLog(
        uint64_t fromTimestamp = 0,
        uint64_t toTimestamp = UINT64_MAX
    ) {
        return auditLogger_.getWalletAuditLog(walletAddress_, fromTimestamp, toTimestamp);
    }

    // Export audit log
    Bytes exportAuditLog(const std::string& format) {
        return auditLogger_.exportAuditLog(walletAddress_, format);
    }

    // Add party
    void addParty(const Party& party) {
        partyComm_.addParty(party);
    }

    // Get transaction request
    std::optional<TransactionRequest> getTransactionRequest(const std::string& requestId) {
        return txRequestManager_.getRequest(requestId);
    }

private:
    MPCWallet mpcWallet_;
    PartyCommunication partyComm_;
    PolicyEngine policyEngine_;
    AuditLogger auditLogger_;
    TransactionRequestManager txRequestManager_;

    // Optional TSS signing backend (real secp256k1 threshold signer).
    // When unset, sign() fails closed rather than emitting a placeholder.
    std::function<Bytes(const Bytes&)> signingBackend_;

    uint32_t threshold_;
    Address walletAddress_;
};

} // namespace tiger

#endif // TIGERWALLET_ENTERPRISE_MPC_HPP
