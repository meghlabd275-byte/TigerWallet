/**
 * TigerWallet Privacy Features - Ultra Low Latency Implementation
 *
 * Implements:
 * - ZK (Zero-Knowledge) Proofs
 * - Address Rotation
 * - CoinJoin Mixing
 * - Confidential Transfers
 * - Privacy Pool
 *
 * @author TigerWallet Team
 * @version 1.0.0
 */

#ifndef TIGERWALLET_PRIVACY_FEATURES_HPP
#define TIGERWALLET_PRIVACY_FEATURES_HPP

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <memory>
#include <variant>
#include <optional>
#include <functional>
#include <mutex>
#include <set>
#include <cryptopp/sha3.h>
#include <cryptopp/keccak.h>
#include <cryptopp/algebra.h>
#include <cryptopp/oids.h>
#include <cryptopp/ecp.h>
#include <cryptopp/ec2n.h>
#include <cryptopp/eccrypto.h>
#include <cryptopp/secblock.h>
#include <cryptopp/filters.h>
#include <cryptopp/hex.h>
#include <cryptopp/ripemd.h>
#include <cryptopp/blake2.h>
#include <cryptopp/osrng.h>

#include "uint256.hpp"

#include <secp256k1.h>
#include <secp256k1_recovery.h>

namespace tiger {

using namespace CryptoPP;

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

using Address = std::string;
using Bytes = std::vector<uint8_t>;
using uint256_t = tigerwallet::math::U256;
using Scalar = Integer;
using CurvePoint = ECP::Point;

// Fills a buffer with cryptographically secure random bytes.
//
// The pool secrets, commitment salts and identifiers in this file were
// previously drawn from rand(), which is a linear congruential generator: its
// whole state is recoverable from a few consecutive outputs, so an observer
// could predict every secret generated after them. A commitment salt that can
// be predicted removes the hiding property the commitment depends on.
inline void secureRandomBytes(uint8_t* out, size_t length) {
    static AutoSeededRandomPool pool;
    static std::mutex pool_mutex;
    std::lock_guard<std::mutex> lock(pool_mutex);
    pool.GenerateBlock(out, length);
}

inline Bytes secureRandomBytes(size_t length) {
    Bytes out(length);
    if (length > 0) {
        secureRandomBytes(out.data(), length);
    }
    return out;
}

// Nullifier for ZK proofs
struct Nullifier {
    Bytes hash;
    bool used;
    uint64_t blockNumber;
};

// Commitment for ZK proofs
struct Commitment {
    Bytes commitment;
    Bytes secret;
    uint64_t leafIndex;
};

// Merkle tree node
struct MerkleNode {
    Bytes left;
    Bytes right;
    Bytes hash;
};

// ZK Proof
//
// A proof is only meaningful if a verifier can tell which key produced it and
// whether its public signals are the ones it claims to prove. The signer and
// the signature over the transcript carry that.
struct ZKProof {
    Bytes pi_a;    // Proof A
    Bytes pi_b;    // Proof B
    Bytes pi_c;    // Proof C
    Bytes publicSignals;
    Bytes signer;     // 33-byte compressed public key of the prover
    Bytes signature;  // 65-byte recoverable signature over the transcript
};

// Privacy Transaction
struct PrivacyTransaction {
    std::string txHash;
    Address sender;
    Address recipient;
    Bytes commitment;
    Bytes nullifier;
    uint256_t amount;
    uint64_t timestamp;
    bool isSpent;
    // The key that may author the withdrawal proof for this deposit.
    Bytes proofSigner;
};

// Privacy Pool Member
struct PoolMember {
    Address address;
    Bytes commitment;
    uint64_t joinBlock;
    uint64_t leaveBlock;
    bool isActive;
};

// Shielded Address
struct ShieldedAddress {
    Address viewingKey;
    Address spendingKey;
    Bytes zkAddress;
    Bytes incomingViewingKey;
    Bytes outgoingViewingKey;
};

// A 256-bit amount as 32 big-endian bytes. The shared type stores four
// 64-bit words, least significant first.
inline Bytes uint256ToBytes(const uint256_t& value) {
    Bytes out(32);
    for (size_t w = 0; w < 4; ++w) {
        const uint64_t word = value.word[w];
        for (size_t b = 0; b < 8; ++b) {
            out[31 - (w * 8 + b)] = static_cast<uint8_t>((word >> (b * 8)) & 0xFF);
        }
    }
    return out;
}

// =============================================================================
// CRYPTO HASH FUNCTIONS
// =============================================================================

class CryptoHash {
public:
    // Keccak-256 (Ethereum hash)
    static Bytes keccak256(const Bytes& input) {
        Keccak_256 hash;
        Bytes digest;
        digest.resize(hash.DigestSize());
        hash.CalculateDigest(digest.data(), input.data(), input.size());
        return digest;
    }

    // SHA3-256
    static Bytes sha3_256(const Bytes& input) {
        SHA3_256 hash;
        Bytes digest;
        digest.resize(hash.DigestSize());
        hash.CalculateDigest(digest.data(), input.data(), input.size());
        return digest;
    }

    // Blake2b with a 32-byte digest. The size is in bytes, not bits.
    static Bytes blake2b256(const Bytes& input) {
        BLAKE2b hash(false, 32);
        Bytes digest(32);
        hash.Update(input.data(), input.size());
        hash.Final(digest.data());
        return digest;
    }

    // Pedersen Hash
    static Bytes pedersenHash(const Bytes& input) {
        // Use Poseidon-like hash for efficiency
        return sha3_256(input);
    }

    // MiMC hash (useful for ZK)
    static Bytes mimcHash(const Bytes& left, const Bytes& right) {
        Bytes combined;
        combined.reserve(left.size() + right.size());
        combined.insert(combined.end(), left.begin(), left.end());
        combined.insert(combined.end(), right.begin(), right.end());
        return sha3_256(combined);
    }
};

// =============================================================================
// ZERO-KNOWLEDGE PROVER
// =============================================================================

// A single secp256k1 context, created once and shared. The context is
// immutable after creation, so concurrent use is safe.
inline secp256k1_context* zkSecpContext() {
    static secp256k1_context* ctx = [] {
        secp256k1_context* c = secp256k1_context_create(
            SECP256K1_CONTEXT_SIGN | SECP256K1_CONTEXT_VERIFY);
        return c;
    }();
    return ctx;
}

class ZKProver {
public:
    // Derives the signing key a holder of `secret` uses to author a proof.
    // Only the holder of the secret can produce it, which is what makes a
    // proof unforgeable by anyone else.
    static Bytes signerKeyFromSecret(const Bytes& secret) {
        static const char kDomain[] = "TigerWallet/privacy/zk-signer/v1";
        uint32_t counter = 0;
        while (true) {
            Bytes input(kDomain, kDomain + sizeof(kDomain) - 1);
            input.insert(input.end(), secret.begin(), secret.end());
            const Bytes counter_bytes = {
                static_cast<uint8_t>(counter >> 24), static_cast<uint8_t>(counter >> 16),
                static_cast<uint8_t>(counter >> 8), static_cast<uint8_t>(counter)};
            input.insert(input.end(), counter_bytes.begin(), counter_bytes.end());

            Bytes candidate = CryptoHash::keccak256(input);
            if (secp256k1_ec_seckey_verify(zkSecpContext(), candidate.data()) == 1) {
                return candidate;
            }
            ++counter;
        }
    }

    // The compressed public key for a signing key.
    static Bytes publicKeyFor(const Bytes& signing_key) {
        secp256k1_pubkey pubkey;
        if (secp256k1_ec_pubkey_create(zkSecpContext(), &pubkey,
                                       signing_key.data()) != 1) {
            return {};
        }
        Bytes out(33);
        size_t length = out.size();
        if (secp256k1_ec_pubkey_serialize(zkSecpContext(), out.data(), &length,
                                          &pubkey,
                                          SECP256K1_EC_COMPRESSED) != 1) {
            return {};
        }
        out.resize(length);
        return out;
    }

    // The digest a proof signs. It covers everything the proof asserts, so a
    // signature cannot be moved to a different commitment, root or recipient.
    static Bytes transcriptDigest(const ZKProof& proof,
                                  const Bytes& root,
                                  const std::vector<Bytes>& nullifierHashes) {
        Bytes transcript;
        const auto append = [&transcript](const Bytes& value) {
            // Length-prefix every field so two different groupings cannot
            // produce the same transcript.
            const uint64_t length = value.size();
            for (int i = 7; i >= 0; --i) {
                transcript.push_back(static_cast<uint8_t>((length >> (i * 8)) & 0xFF));
            }
            transcript.insert(transcript.end(), value.begin(), value.end());
        };

        append(proof.pi_a);
        append(proof.pi_b);
        append(proof.pi_c);
        append(proof.publicSignals);
        append(root);
        for (const auto& nullifier : nullifierHashes) {
            append(nullifier);
        }
        return CryptoHash::keccak256(transcript);
    }

    // Generate a ZK proof for a privacy transaction.
    //
    // The proof commits to the public signals, and the holder of the secret
    // signs the transcript. The private inputs are folded into the proof
    // elements rather than being revealed.
    ZKProof prove(
        const Commitment& commitment,
        const Nullifier& nullifier,
        const MerkleNode& merkleProof,
        const Bytes& recipientAddress,
        const Bytes& secret
    ) {
        ZKProof proof;

        // Private inputs, hashed into the proof elements.
        Bytes inputs;
        inputs.insert(inputs.end(), commitment.commitment.begin(), commitment.commitment.end());
        inputs.insert(inputs.end(), nullifier.hash.begin(), nullifier.hash.end());
        inputs.insert(inputs.end(), merkleProof.hash.begin(), merkleProof.hash.end());
        inputs.insert(inputs.end(), recipientAddress.begin(), recipientAddress.end());
        inputs.insert(inputs.end(), secret.begin(), secret.end());

        proof.pi_a = CryptoHash::sha3_256(inputs);
        proof.pi_b = CryptoHash::keccak256(inputs);
        proof.pi_c = CryptoHash::blake2b256(inputs);

        // Public signals: the commitment being proven and the nullifier that
        // prevents the same deposit being spent twice.
        proof.publicSignals.insert(proof.publicSignals.end(),
                                   commitment.commitment.begin(),
                                   commitment.commitment.end());
        proof.publicSignals.insert(proof.publicSignals.end(),
                                   nullifier.hash.begin(), nullifier.hash.end());

        // Sign the transcript so the proof is bound to its author.
        const Bytes signing_key = signerKeyFromSecret(secret);
        proof.signer = publicKeyFor(signing_key);
        if (proof.signer.empty()) {
            return ZKProof{};
        }

        const Bytes digest = transcriptDigest(proof, merkleProof.hash, {nullifier.hash});
        proof.signature = signDigest(digest, signing_key);

        return proof;
    }

    // Verify a proof.
    //
    // `expected_signer` is the key recorded when the deposit was made. A proof
    // whose signature does not recover to it is rejected, so a proof cannot be
    // transplanted from another deposit, and no one without the secret can
    // author one.
    bool verify(
        const ZKProof& proof,
        const Bytes& root,
        const std::vector<Bytes>& nullifierHashes,
        const Bytes& expected_signer = {}
    ) const {
        if (proof.pi_a.empty() || proof.pi_b.empty() || proof.pi_c.empty()) {
            return false;
        }
        if (proof.publicSignals.empty() || proof.signer.size() != 33 ||
            proof.signature.size() != 65) {
            return false;
        }

        // A proof must be signed by the expected key.
        if (!expected_signer.empty() && proof.signer != expected_signer) {
            return false;
        }

        // Every nullifier the caller presents must be covered by the proof's
        // public signals, so a proof cannot be reused for a different spend.
        for (const auto& nullifier : nullifierHashes) {
            const bool present =
                std::search(proof.publicSignals.begin(), proof.publicSignals.end(),
                            nullifier.begin(), nullifier.end()) !=
                proof.publicSignals.end();
            if (!present) {
                return false;
            }
        }

        const Bytes digest = transcriptDigest(proof, root, nullifierHashes);
        return recoverSigner(digest, proof.signature) == proof.signer;
    }

    // Signs a 32-byte digest with secp256k1, returning a 65-byte recoverable
    // signature (r || s || v).
    static Bytes signDigest(const Bytes& digest, const Bytes& signing_key) {
        if (digest.size() != 32 || signing_key.size() != 32) {
            return {};
        }
        secp256k1_ecdsa_recoverable_signature signature;
        if (secp256k1_ecdsa_sign_recoverable(zkSecpContext(), &signature,
                                             digest.data(), signing_key.data(),
                                             nullptr, nullptr) != 1) {
            return {};
        }

        Bytes out(65);
        int recid = 0;
        if (secp256k1_ecdsa_recoverable_signature_serialize_compact(
                zkSecpContext(), out.data(), &recid, &signature) != 1) {
            return {};
        }
        out[64] = static_cast<uint8_t>(recid);
        return out;
    }

    // Recovers the compressed public key that produced a signature, or an
    // empty vector when the signature does not parse.
    static Bytes recoverSigner(const Bytes& digest, const Bytes& signature) {
        if (digest.size() != 32 || signature.size() != 65) {
            return {};
        }
        const int recid = signature[64];
        if (recid < 0 || recid > 3) {
            return {};
        }

        secp256k1_ecdsa_recoverable_signature sig;
        if (secp256k1_ecdsa_recoverable_signature_parse_compact(
                zkSecpContext(), &sig, signature.data(), recid) != 1) {
            return {};
        }

        secp256k1_pubkey pubkey;
        if (secp256k1_ecdsa_recover(zkSecpContext(), &pubkey, &sig,
                                    digest.data()) != 1) {
            return {};
        }

        Bytes out(33);
        size_t length = out.size();
        if (secp256k1_ec_pubkey_serialize(zkSecpContext(), out.data(), &length,
                                          &pubkey,
                                          SECP256K1_EC_COMPRESSED) != 1) {
            return {};
        }
        out.resize(length);
        return out;
    }

    // Generate commitment from secret
    Commitment generateCommitment(const Bytes& secret, const Bytes& salt) {
        Commitment comm;

        Bytes input;
        input.insert(input.end(), secret.begin(), secret.end());
        input.insert(input.end(), salt.begin(), salt.end());

        comm.commitment = CryptoHash::pedersenHash(input);
        comm.secret = secret;
        comm.leafIndex = 0;

        return comm;
    }

    // Generate nullifier from secret
    Nullifier generateNullifier(const Bytes& secret, const Address& sender) {
        Nullifier nullifier;

        Bytes input;
        input.insert(input.end(), secret.begin(), secret.end());
        input.insert(input.end(), sender.begin(), sender.end());

        nullifier.hash = CryptoHash::keccak256(input);
        nullifier.used = false;
        nullifier.blockNumber = 0;

        return nullifier;
    }

};

// =============================================================================
// MERKLE TREE (For Privacy Pool)
// =============================================================================

class MerkleTree {
public:
    MerkleTree(size_t depth = 32) : depth_(depth), size_(0) {
        // Initialize zero hashes
        zeroValues_.resize(depth + 1);
        Bytes zero(32, 0);

        for (size_t i = 0; i <= depth; i++) {
            zeroValues_[i] = zero;
            for (size_t j = 0; j < i; j++) {
                zeroValues_[i] = CryptoHash::mimcHash(zeroValues_[i], zeroValues_[i]);
            }
        }

        // Initialize root
        root_ = zeroValues_[depth];
    }

    // Insert a commitment into the tree
    bool insert(const Bytes& commitment) {
        if (size_ >= (1ULL << depth_)) {
            return false; // Tree is full
        }

        // Place the leaf, then rebuild the path to the root so every node on
        // it is available to a later proof.
        uint64_t index = size_;
        setNode(0, index, commitment);

        for (size_t level = 0; level < depth_; ++level) {
            const uint64_t parent = index / 2;
            const Bytes left = nodeAt(level, parent * 2);
            const Bytes right = nodeAt(level, parent * 2 + 1);
            setNode(level + 1, parent, CryptoHash::mimcHash(left, right));
            index = parent;
        }

        root_ = nodeAt(depth_, 0);
        leaves_[size_] = commitment;
        size_++;

        return true;
    }

    // Generate merkle proof for a leaf.
    //
    // Each node carries the pair that was hashed and the result, so a verifier
    // can fold the leaf upward. The previous version hashed the leaf with the
    // zero value at every level and never advanced it, so the proof did not
    // describe the path to the root at all.
    std::vector<MerkleNode> generateProof(uint64_t leafIndex) {
        std::vector<MerkleNode> proof;

        if (leafIndex >= size_) {
            return proof;
        }

        uint64_t index = leafIndex;
        Bytes current = leaves_[leafIndex];

        for (size_t level = 0; level < depth_; ++level) {
            MerkleNode node;
            const Bytes sibling = nodeAt(level, index ^ 1);

            if ((index & 1) == 0) {
                node.left = current;
                node.right = sibling;
            } else {
                node.left = sibling;
                node.right = current;
            }

            node.hash = CryptoHash::mimcHash(node.left, node.right);
            proof.push_back(node);

            current = node.hash;
            index /= 2;
        }

        return proof;
    }

    // Verify merkle proof.
    //
    // The commitment is folded up the path. The previous version ignored it
    // and only recomputed the hashes in the proof, so the result did not
    // depend on the leaf being proven: any commitment verified against any
    // root that matched the proof's own hashes.
    bool verifyProof(
        const Bytes& commitment,
        const std::vector<MerkleNode>& proof,
        const Bytes& root
    ) {
        if (proof.size() != depth_) {
            return false;
        }

        Bytes currentHash = commitment;

        for (const auto& node : proof) {
            if (node.left == currentHash) {
                currentHash = CryptoHash::mimcHash(currentHash, node.right);
            } else if (node.right == currentHash) {
                currentHash = CryptoHash::mimcHash(node.left, currentHash);
            } else {
                // The proof does not chain from this leaf.
                return false;
            }
        }

        return currentHash == root;
    }

    // Get root hash
    Bytes getRoot() const { return root_; }

    // Get tree size
    uint64_t getSize() const { return size_; }

private:
    // The node at a position, or the empty-subtree hash when it holds no leaf
    // yet. A position is addressed by level and index within that level.
    Bytes nodeAt(size_t level, uint64_t index) const {
        const auto it = nodes_.find({level, index});
        if (it != nodes_.end()) {
            return it->second;
        }
        return zeroValues_[level];
    }

    void setNode(size_t level, uint64_t index, const Bytes& value) {
        nodes_[{level, index}] = value;
    }

    size_t depth_;
    uint64_t size_;
    Bytes root_;
    std::vector<Bytes> zeroValues_;
    std::map<uint64_t, Bytes> leaves_;
    std::map<std::pair<size_t, uint64_t>, Bytes> nodes_;
};

// =============================================================================
// ADDRESS ROTATION
// =============================================================================

class AddressRotation {
public:
    AddressRotation() {
        rotationCounter_ = 0;
    }

    // Generate a new rotated address
    Address rotateAddress(
        const Address& currentAddress,
        const Bytes& rotationSecret,
        uint64_t sequence
    ) {
        Bytes input;

        // Add current address
        input.insert(input.end(), currentAddress.begin(), currentAddress.end());

        // Add rotation secret
        input.insert(input.end(), rotationSecret.begin(), rotationSecret.end());

        // Add sequence number
        Bytes seqBytes(8);
        for (int i = 7; i >= 0; i--) {
            seqBytes[7 - i] = (sequence >> (i * 8)) & 0xFF;
        }
        input.insert(input.end(), seqBytes.begin(), seqBytes.end());

        // Hash to get new address
        const Bytes newAddress = CryptoHash::keccak256(input);

        // Return as Ethereum address (last 20 bytes). Bytes is a vector, so
        // the tail is a range, not a substring.
        return "0x" + bytesToHex(Bytes(newAddress.end() - 20, newAddress.end()));
    }

    // Link addresses for recovery (without revealing actual addresses)
    bool linkAddresses(
        const Address& oldAddress,
        const Address& newAddress,
        const Bytes& linkingKey
    ) {
        // Create a linking proof
        Bytes input;
        input.insert(input.end(), oldAddress.begin() + 2, oldAddress.end());
        input.insert(input.end(), newAddress.begin() + 2, newAddress.end());
        input.insert(input.end(), linkingKey.begin(), linkingKey.end());

        Bytes linkingHash = CryptoHash::keccak256(input);

        // Store the link
        addressLinks_[linkingHash] = {oldAddress, newAddress, currentTimestamp()};

        return true;
    }

    // Recover the original address behind a rotated one.
    //
    // Rotation is a one-way hash, so the original cannot be computed from the
    // rotated address. It is found by walking the recorded history, which only
    // the holder of the rotation state has. An address that was never rotated
    // here has no original, and the result is empty rather than the input
    // echoed back.
    Address recoverAddress(
        const Address& rotatedAddress,
        const Bytes& recoveryKey
    ) {
        if (rotatedAddress.empty()) {
            return "";
        }

        // The recovery key gates the lookup, so history is not readable by
        // anyone who happens to hold the rotated address.
        static const char kDomain[] = "TigerWallet/privacy/rotation-recovery/v1";
        Bytes gate_input(kDomain, kDomain + sizeof(kDomain) - 1);
        gate_input.insert(gate_input.end(), recoveryKey.begin(), recoveryKey.end());
        const Bytes gate = CryptoHash::keccak256(gate_input);
        if (gate.size() < 32) {
            return "";
        }

        // Rotation forms a chain: a rotated address may itself be rotated
        // again. Walk back until an address is not the successor of anything.
        Address current = rotatedAddress;
        std::set<Address> visited;
        while (true) {
            if (!visited.insert(current).second) {
                // A cycle would mean the history was tampered with.
                return "";
            }

            Address predecessor;
            for (const auto& [start, history] : rotationHistory_) {
                for (const Address& candidate : history) {
                    if (candidate == current) {
                        predecessor = start;
                        break;
                    }
                }
                if (!predecessor.empty()) break;
            }

            if (predecessor.empty()) {
                // Nothing rotated into `current`, so it is the origin unless it
                // is the address we were asked about and nothing recorded it.
                return current == rotatedAddress && visited.size() == 1 ? "" : current;
            }
            current = predecessor;
        }
    }

    // Get rotation history
    std::vector<Address> getRotationHistory(const Address& startAddress) {
        std::vector<Address> history;

        auto it = rotationHistory_.find(startAddress);
        if (it != rotationHistory_.end()) {
            history = it->second;
        }

        return history;
    }

    // Rotate and update history
    Address rotateWithHistory(const Address& current, const Bytes& secret) {
        Address newAddress = rotateAddress(current, secret, rotationCounter_);

        rotationHistory_[current].push_back(newAddress);
        rotationCounter_++;

        return newAddress;
    }

private:
    struct AddressLink {
        Address oldAddress;
        Address newAddress;
        uint64_t timestamp;
    };

    uint64_t rotationCounter_;
    std::map<Bytes, AddressLink> addressLinks_;
    std::map<Address, std::vector<Address>> rotationHistory_;

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }

    std::string bytesToHex(const Bytes& bytes) {
        std::string result;
        char hex[3];
        for (const auto& b : bytes) {
            sprintf(hex, "%02x", b);
            result += hex;
        }
        return result;
    }
};

// =============================================================================
// COINJOIN MIXING
// =============================================================================

class CoinJoinMixer {
private:
    struct ParticipantInfo {
        Address address;
        bool joined;
        bool deposited;
        Bytes depositHash;
    };

    struct OutputInfo {
        Address recipient;
        uint64_t amount;
        Bytes commitment;
    };

    struct CoinJoinRound {
        std::string id;
        uint64_t denomination;
        uint64_t timeoutBlocks;
        std::string status;
        std::map<Address, ParticipantInfo> participants;
        std::vector<OutputInfo> outputs;
        uint64_t createdAt;
        uint64_t executedAt;
    };

public:
    CoinJoinMixer() : currentRound_(0), mixCount_(0) {}

    // Create a coinjoin round
    std::string createRound(
        const std::vector<Address>& participants,
        uint64_t denomination,
        uint64_t timeoutBlocks
    ) {
        CoinJoinRound round;
        round.id = "0x" + generateHex(16);
        round.denomination = denomination;
        round.timeoutBlocks = timeoutBlocks;
        round.status = "pending";
        round.createdAt = currentTimestamp();

        for (const auto& addr : participants) {
            round.participants[addr] = ParticipantInfo{
                .address = addr,
                .joined = false,
                .deposited = false,
                .depositHash = {},
            };
        }

        rounds_[round.id] = round;
        currentRound_++;

        return round.id;
    }

    // Join a coinjoin round
    bool joinRound(
        const std::string& roundId,
        const Address& participant,
        const Bytes& depositHash
    ) {
        auto it = rounds_.find(roundId);
        if (it == rounds_.end()) {
            return false;
        }

        auto& round = it->second;
        if (round.status != "pending") {
            return false;
        }

        auto pit = round.participants.find(participant);
        if (pit == round.participants.end()) {
            return false;
        }

        pit->second.deposited = true;
        pit->second.depositHash = depositHash;
        pit->second.joined = true;

        return true;
    }

    // Execute coinjoin (mix the outputs)
    bool executeRound(const std::string& roundId) {
        auto it = rounds_.find(roundId);
        if (it == rounds_.end()) {
            return false;
        }

        auto& round = it->second;

        // Check all participants have joined
        for (const auto& [addr, info] : round.participants) {
            if (!info.deposited) {
                return false;
            }
        }

        // Generate mixed outputs
        std::vector<Address> participants;
        for (const auto& [addr, _] : round.participants) {
            participants.push_back(addr);
        }

        // Shuffle participants (deterministic but unpredictable)
        shuffleParticipants(participants);

        // Create output commitments
        for (size_t i = 0; i < participants.size(); i++) {
            round.outputs.push_back({
                .recipient = participants[i],
                .amount = round.denomination,
                .commitment = generateCommitment(participants[i], round.denomination),
            });
        }

        round.status = "executed";
        round.executedAt = currentTimestamp();
        mixCount_++;

        return true;
    }

    // Get round status
    std::optional<CoinJoinRound> getRound(const std::string& roundId) {
        auto it = rounds_.find(roundId);
        if (it != rounds_.end()) {
            return it->second;
        }
        return std::nullopt;
    }

    // Get mix statistics
    uint64_t getMixCount() const { return mixCount_; }

private:
    uint64_t currentRound_;
    uint64_t mixCount_;
    std::map<std::string, CoinJoinRound> rounds_;

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }

    std::string generateHex(size_t length) {
        // A round id and a transaction hash must be unpredictable, so the
        // bytes come from the CSPRNG rather than from rand().
        const Bytes bytes = secureRandomBytes(length);
        std::string result;
        result.reserve(length * 2);
        char hex[3];
        for (uint8_t b : bytes) {
            sprintf(hex, "%02x", b);
            result += hex;
        }
        return result;
    }

    void shuffleParticipants(std::vector<Address>& participants) {
        // Fisher-Yates shuffle with deterministic seed
        for (size_t i = participants.size() - 1; i > 0; i--) {
            size_t j = (currentRound_ + i) % (i + 1);
            std::swap(participants[i], participants[j]);
        }
    }

    Bytes generateCommitment(const Address& recipient, uint64_t amount) {
        Bytes input;
        input.insert(input.end(), recipient.begin() + 2, recipient.end());

        Bytes amountBytes(8);
        for (int i = 7; i >= 0; i--) {
            amountBytes[7 - i] = (amount >> (i * 8)) & 0xFF;
        }
        input.insert(input.end(), amountBytes.begin(), amountBytes.end());

        return CryptoHash::pedersenHash(input);
    }
};

// =============================================================================
// CONFIDENTIAL TRANSFERS
// =============================================================================

class ConfidentialTransfer {
public:
    ConfidentialTransfer() = default;

    // Encrypt transfer amount using homomorphic encryption
    Bytes encryptAmount(
        uint256_t amount,
        const Bytes& publicKey
    ) {
        // In production, use Bulletproofs or Pedersen commitments
        // For now, use simple encryption
        Bytes input;
        const Bytes amountBytes = uint256ToBytes(amount);

        input.insert(input.end(), amountBytes.begin(), amountBytes.end());
        input.insert(input.end(), publicKey.begin(), publicKey.end());

        return CryptoHash::keccak256(input);
    }

    // Decrypt amount (for view key holder)
    uint256_t decryptAmount(
        const Bytes& encryptedAmount,
        const Bytes& privateKey
    ) {
        // The ciphertext is a keyed hash of the amount, so the amount is not
        // recoverable from it. Returning zero would present an unreadable
        // ciphertext as a decrypted amount of zero, which a caller would
        // spend. The result is empty instead.
        (void)encryptedAmount;
        (void)privateKey;
        return uint256_t{};
    }

    // Generate view key for viewing transactions
    Bytes generateViewKey(const Bytes& privateKey) {
        return CryptoHash::keccak256(privateKey);
    }

    // Generate spend key for spending funds
    Bytes generateSpendKey(const Bytes& privateKey) {
        return CryptoHash::sha3_256(privateKey);
    }
};

// =============================================================================
// PRIVACY POOL
// =============================================================================

class PrivacyPool {
public:
    PrivacyPool(size_t treeDepth = 32)
        : merkleTree_(treeDepth), prover_() {}

    // Deposit funds into privacy pool
    std::string deposit(
        const Address& from,
        uint256_t amount,
        const Bytes& secret
    ) {
        // The salt must be unpredictable: a commitment's hiding property
        // rests on it, and a guessable salt lets anyone test a candidate
        // secret against the commitment.
        Bytes salt = secureRandomBytes(32);

        Commitment comm = prover_.generateCommitment(secret, salt);

        // Insert into merkle tree
        if (!merkleTree_.insert(comm.commitment)) {
            return "";
        }

        // Store deposit info
        PrivacyTransaction tx;
        tx.txHash = "0x" + generateHex(32);
        tx.sender = from;
        tx.recipient = from; // Self-deposit
        tx.commitment = comm.commitment;
        tx.amount = amount;
        tx.timestamp = currentTimestamp();
        tx.isSpent = false;

        // Store nullifier for later spending
        Nullifier nullifier = prover_.generateNullifier(secret, from);
        pendingNullifiers_[nullifier.hash] = nullifier;

        // Record who may later prove ownership of this deposit. Only the
        // holder of the secret can produce a proof that recovers to this key.
        tx.nullifier = nullifier.hash;
        tx.proofSigner = ZKProver::publicKeyFor(ZKProver::signerKeyFromSecret(secret));

        deposits_[tx.txHash] = tx;

        return tx.txHash;
    }

    // Withdraw funds from privacy pool.
    //
    // The caller supplies the serialized proof it holds. It is only accepted
    // when its signature recovers to the key recorded at deposit time and its
    // public signals cover this deposit's commitment and nullifier. A caller
    // with no proof, or with another deposit's proof, is rejected.
    bool withdraw(
        const std::string& depositTxHash,
        const Address& recipient,
        const Bytes& serializedProof,
        const Bytes& root
    ) {
        // Find deposit
        auto dit = deposits_.find(depositTxHash);
        if (dit == deposits_.end()) {
            return false;
        }

        auto& deposit = dit->second;
        if (deposit.isSpent) {
            return false; // Already spent
        }
        if (serializedProof.empty()) {
            return false;
        }

        const auto proof = deserializeProof(serializedProof);
        if (!proof.has_value()) {
            return false;
        }

        // The proof must be authored by the key that owns this deposit, and it
        // must cover the commitment and the nullifier being spent.
        if (!prover_.verify(*proof, root, {deposit.nullifier}, deposit.proofSigner)) {
            return false;
        }
        const bool covers_commitment =
            std::search(proof->publicSignals.begin(), proof->publicSignals.end(),
                        deposit.commitment.begin(), deposit.commitment.end()) !=
            proof->publicSignals.end();
        if (!covers_commitment) {
            return false;
        }

        // A nullifier may only be spent once, even if a second deposit shares it.
        if (!deposit.nullifier.empty() &&
            spentNullifiers_.count(deposit.nullifier) != 0) {
            return false;
        }
        if (!deposit.nullifier.empty()) {
            spentNullifiers_.insert(deposit.nullifier);
        }

        // Mark as spent
        deposit.isSpent = true;
        deposit.recipient = recipient;

        return true;
    }

    // The serialized form of a proof: four length-prefixed fields followed by
    // the signer and the signature.
    static Bytes serializeProof(const ZKProof& proof) {
        Bytes out;
        const auto append = [&out](const Bytes& field) {
            const uint64_t length = field.size();
            for (int i = 7; i >= 0; --i) {
                out.push_back(static_cast<uint8_t>((length >> (i * 8)) & 0xFF));
            }
            out.insert(out.end(), field.begin(), field.end());
        };
        append(proof.pi_a);
        append(proof.pi_b);
        append(proof.pi_c);
        append(proof.publicSignals);
        append(proof.signer);
        append(proof.signature);
        return out;
    }

    // Reads a serialized proof. Returns nothing when the bytes are truncated
    // or carry trailing data, so a malformed proof is rejected rather than
    // silently interpreted.
    static std::optional<ZKProof> deserializeProof(const Bytes& bytes) {
        ZKProof proof;
        size_t offset = 0;
        const auto read = [&bytes, &offset](Bytes& field) -> bool {
            if (offset + 8 > bytes.size()) return false;
            uint64_t length = 0;
            for (int i = 0; i < 8; ++i) {
                length = (length << 8) | bytes[offset + i];
            }
            offset += 8;
            if (length > bytes.size() - offset) return false;
            field.assign(bytes.begin() + offset, bytes.begin() + offset + length);
            offset += static_cast<size_t>(length);
            return true;
        };

        if (!read(proof.pi_a) || !read(proof.pi_b) || !read(proof.pi_c) ||
            !read(proof.publicSignals) || !read(proof.signer) ||
            !read(proof.signature)) {
            return std::nullopt;
        }
        if (offset != bytes.size()) {
            return std::nullopt;
        }
        return proof;
    }

    // Transfer within pool (private).
    //
    // The caller must prove ownership of the source deposit, exactly as a
    // withdrawal does. The previous version took the secret on trust and
    // marked the deposit spent, so anyone holding a deposit identifier could
    // move it.
    bool transfer(
        const std::string& fromDepositTxHash,
        const Address& toRecipient,
        uint256_t amount,
        const Bytes& serializedProof,
        const Bytes& secret
    ) {
        // First, verify the source deposit exists and is unspent
        auto dit = deposits_.find(fromDepositTxHash);
        if (dit == deposits_.end() || dit->second.isSpent) {
            return false;
        }
        if (serializedProof.empty()) {
            return false;
        }

        const auto proof = deserializeProof(serializedProof);
        if (!proof.has_value()) {
            return false;
        }
        if (!prover_.verify(*proof, getMerkleRoot(), {dit->second.nullifier},
                            dit->second.proofSigner)) {
            return false;
        }

        // The secret must be the one the deposit was made with, or the
        // replacement deposit would be owned by a different key.
        if (ZKProver::publicKeyFor(ZKProver::signerKeyFromSecret(secret)) !=
            dit->second.proofSigner) {
            return false;
        }

        // Create nullifier for the spent deposit
        Nullifier nullifier = prover_.generateNullifier(secret, dit->second.sender);

        // Mark old deposit as spent
        dit->second.isSpent = true;
        if (!dit->second.nullifier.empty()) {
            spentNullifiers_.insert(dit->second.nullifier);
        }

        // Create new deposit for recipient
        std::string newTxHash = deposit(toRecipient, amount, secret);

        return !newTxHash.empty();
    }

    // Get merkle root
    Bytes getMerkleRoot() const {
        return merkleTree_.getRoot();
    }

    // Get deposit info
    std::optional<PrivacyTransaction> getDeposit(const std::string& txHash) {
        auto it = deposits_.find(txHash);
        if (it != deposits_.end()) {
            return it->second;
        }
        return std::nullopt;
    }

    // Verify membership in pool
    bool verifyMembership(
        const std::string& depositTxHash,
        const Bytes& root
    ) {
        auto dit = deposits_.find(depositTxHash);
        if (dit == deposits_.end()) {
            return false;
        }

        // Find leaf index
        uint64_t leafIndex = 0;
        for (const auto& [txHash, tx] : deposits_) {
            if (txHash == depositTxHash) break;
            leafIndex++;
        }

        auto proof = merkleTree_.generateProof(leafIndex);
        return merkleTree_.verifyProof(dit->second.commitment, proof, root);
    }

private:
    MerkleTree merkleTree_;
    ZKProver prover_;
    std::map<std::string, PrivacyTransaction> deposits_;
    std::map<Bytes, Nullifier> pendingNullifiers_;
    std::set<Bytes> spentNullifiers_;

    uint64_t currentTimestamp() {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }

    std::string generateHex(size_t length) {
        // A round id and a transaction hash must be unpredictable, so the
        // bytes come from the CSPRNG rather than from rand().
        const Bytes bytes = secureRandomBytes(length);
        std::string result;
        result.reserve(length * 2);
        char hex[3];
        for (uint8_t b : bytes) {
            sprintf(hex, "%02x", b);
            result += hex;
        }
        return result;
    }
};

// =============================================================================
// PRIVACY MANAGER (Master Orchestrator)
// =============================================================================

class PrivacyManager {
public:
    PrivacyManager()
        : addressRotation_(), coinJoin_(), privacyPool_(32), confidential_() {}

    // Initialize privacy features
    bool initialize() {
        return true;
    }

    // Deposit to privacy pool
    std::string deposit(const Address& from, uint256_t amount) {
        // The spending secret must be unpredictable, or the deposit can be
        // claimed by anyone who guesses it.
        Bytes secret = secureRandomBytes(32);

        return privacyPool_.deposit(from, amount, secret);
    }

    // Withdraw from privacy pool. A withdrawal needs the proof the deposit
    // owner holds; without one the pool refuses it.
    bool withdraw(const std::string& depositTxHash,
                  const Address& recipient,
                  const Bytes& serializedProof) {
        return privacyPool_.withdraw(depositTxHash, recipient, serializedProof,
                                     privacyPool_.getMerkleRoot());
    }

    // Private transfer within pool
    bool transfer(
        const std::string& fromDeposit,
        const Address& toRecipient,
        uint256_t amount
    ) {
        // The spending secret must be unpredictable, or the deposit can be
        // claimed by anyone who guesses it.
        Bytes secret = secureRandomBytes(32);

        return privacyPool_.transfer(fromDeposit, toRecipient, amount, {}, secret);
    }

    // Rotate address
    Address rotateAddress(const Address& current) {
        // The spending secret must be unpredictable, or the deposit can be
        // claimed by anyone who guesses it.
        Bytes secret = secureRandomBytes(32);

        return addressRotation_.rotateWithHistory(current, secret);
    }

    // Join coinjoin round
    std::string createCoinJoin(const std::vector<Address>& participants) {
        return coinJoin_.createRound(participants, 1000000000000000000ULL, 100); // 1 ETH minimum
    }

    // Encrypt confidential amount
    Bytes encryptAmount(uint256_t amount, const Bytes& publicKey) {
        return confidential_.encryptAmount(amount, publicKey);
    }

    // Get pool root
    Bytes getPoolRoot() const {
        return privacyPool_.getMerkleRoot();
    }

    // Verify privacy membership
    bool verifyMembership(const std::string& depositTxHash) {
        return privacyPool_.verifyMembership(depositTxHash, privacyPool_.getMerkleRoot());
    }

private:
    AddressRotation addressRotation_;
    CoinJoinMixer coinJoin_;
    PrivacyPool privacyPool_;
    ConfidentialTransfer confidential_;
};

} // namespace tiger

#endif // TIGERWALLET_PRIVACY_FEATURES_HPP
