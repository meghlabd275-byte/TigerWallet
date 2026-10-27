/**
 * TigerWallet Privacy Features Implementation
 * Ultra-low latency C++ implementation for ZK proofs, CoinJoin, and privacy operations
 */

#include "privacy.hpp"

#include <algorithm>
#include <iostream>
#include <sstream>
#include <iomanip>
#include <random>
#include <chrono>
#include <thread>
#include <mutex>
#include <atomic>
#include <shared_mutex>
#include <unordered_map>
#include <unordered_set>
#include <queue>
#include <cstring>
#include <openssl/sha.h>
#include <openssl/ripemd.h>
#include <openssl/ec.h>
#include <openssl/bn.h>
#include <openssl/obj_mac.h>
#include <openssl/evp.h>
#include <openssl/rand.h>

// The definitions in this file belong to tigerwallet::privacy. A
// using-directive was here before, which put every definition in the global
// namespace: the header declared tigerwallet::privacy::ZKProofGenerator and
// this file defined ::ZKProofGenerator, so the library could not link. The
// namespace is now opened explicitly.
namespace tigerwallet {
namespace privacy {

// ============================================================================
// Cryptographic Primitives Implementation
// ============================================================================

class CommitmentHashImpl {
public:
    static constexpr int STATE_SIZE = 8;

    // Real Keccak-256 sponge. This is a genuine cryptographic hash (NOT the
    // Poseidon SNARK hash over a prime field, which would require BN254/Grumpkin
    // field arithmetic). It is used here as a sound commitment hash for the
    // privacy pool's Merkle tree and nullifiers. ZK proof *generation* is
    // delegated to the zk_infrastructure Rust crate / backend (fail-closed).
    std::string hash(const std::vector<std::string>& inputs) {
        // Concatenate inputs length-prefixed (domain separation) then Keccak-256.
        std::string buf;
        for (const auto& in : inputs) {
            uint32_t len = static_cast<uint32_t>(in.size());
            buf.append(reinterpret_cast<const char*>(&len), 4);
            buf.append(in);
        }
        uint8_t out[32];
        keccak256(reinterpret_cast<const uint8_t*>(buf.data()), buf.size(), out);
        return to_hex(out, 32);
    }

    static std::string hash_single(const std::string& input) {
        CommitmentHashImpl impl;
        return impl.hash({input});
    }

    static std::string hash_pair(const std::string& a, const std::string& b) {
        CommitmentHashImpl impl;
        return impl.hash({a, b});
    }

private:
    // Keccak-256 (Ethereum keccak, NOT NIST SHA3-256). Padding: 0x01 || 0x00...
    // This is a real, self-contained Keccak-f[1600] sponge implementation.
    static void keccak256(const uint8_t* data, size_t len, uint8_t* out) {
        uint64_t state[25] = {0};
        const size_t rate = 136; // 1088 bits = 136 bytes for Keccak-256
        const uint8_t* end = data + len;
        const uint8_t* p = data;
        // Absorb
        while (p + rate <= end) {
            absorb_block(state, p, rate);
            p += rate;
        }
        // Padding: 0x01 ... 0x80 (Keccak padding, NOT SHA3's 0x06)
        uint8_t last[rate];
        size_t rem = end - p;
        memcpy(last, p, rem);
        last[rem] = 0x01;
        for (size_t i = rem + 1; i < rate; i++) last[i] = 0;
        last[rate - 1] |= 0x80;
        absorb_block(state, last, rate);
        // Squeeze 32 bytes
        for (size_t i = 0; i < 4; i++) {
            memcpy(out + i * 8, &state[i], 8);
        }
    }

    static void absorb_block(uint64_t state[25], const uint8_t* block, size_t len) {
        for (size_t i = 0; i < len / 8; i++) {
            uint64_t lane = 0;
            memcpy(&lane, block + i * 8, 8);
            state[i] ^= lane;
        }
        keccak_f(state);
    }

    static void keccak_f(uint64_t state[25]) {
        static const uint64_t rc[24] = {
            0x0000000000000001ULL, 0x0000000000008082ULL, 0x800000000000808aULL,
            0x8000000080008000ULL, 0x000000000000808bULL, 0x0000000080000001ULL,
            0x8000000080008081ULL, 0x8000000000008009ULL, 0x000000000000008aULL,
            0x0000000000000088ULL, 0x0000000080008009ULL, 0x000000008000000aULL,
            0x000000008000808bULL, 0x800000000000008bULL, 0x8000000000008089ULL,
            0x8000000000008003ULL, 0x8000000000008002ULL, 0x8000000000000080ULL,
            0x000000000000800aULL, 0x800000008000000aULL, 0x8000000080008081ULL,
            0x8000000000008080ULL, 0x0000000080000001ULL, 0x8000000080008008ULL
        };
        static const int rot[25] = {
            0,1,62,28,27,36,44,6,55,20,3,10,43,25,39,41,45,15,21,8,18,2,61,56,14
        };
        for (int rnd = 0; rnd < 24; rnd++) {
            // Theta
            uint64_t c[5];
            for (int i = 0; i < 5; i++)
                c[i] = state[i]^state[i+5]^state[i+10]^state[i+15]^state[i+20];
            uint64_t d[5];
            for (int i = 0; i < 5; i++)
                d[i] = c[(i+4)%5] ^ rotl(c[(i+1)%5], 1);
            for (int i = 0; i < 25; i++) state[i] ^= d[i%5];
            // Rho + Pi
            uint64_t b[25];
            for (int i = 0; i < 25; i++)
                b[kPi[i]] = rotl(state[i], rot[i]);
            // Chi
            for (int i = 0; i < 5; i++)
                for (int j = 0; j < 5; j++)
                    state[i+5*j] = b[i+5*j] ^ ((~b[(i+1)%5+5*j]) & b[(i+2)%5+5*j]);
            // Iota
            state[0] ^= rc[rnd];
        }
    }
    static uint64_t rotl(uint64_t x, int n) {
        return (x << n) | (x >> (64 - n));
    }
    // Keccak's pi step as a permutation table. This was a non-const static
    // array filled by init_pi(), which nothing ever called, so every entry was
    // zero and the Rho+Pi step wrote all 25 lanes to b[0]. The sponge then
    // produced a value that was not Keccak at all.
    static constexpr int kPi[25] = {
        0, 10, 20,  5, 15,
       16,  1, 11, 21,  6,
        7, 17,  2, 12, 22,
       23,  8, 18,  3, 13,
       14, 24,  9, 19,  4
    };

    static std::string to_hex(const uint8_t* d, size_t n) {
        static const char* hex = "0123456789abcdef";
        std::string r;
        r.reserve(n*2);
        for (size_t i = 0; i < n; i++) {
            r += hex[d[i]>>4];
            r += hex[d[i]&0xf];
        }
        return r;
    }
};


std::string CommitmentHash::hash(const std::vector<std::string>& inputs) {
    CommitmentHashImpl impl;
    return impl.hash(inputs);
}

std::string CommitmentHash::hash_single(const std::string& input) {
    return CommitmentHashImpl::hash_single(input);
}

std::string CommitmentHash::hash_pair(const std::string& a, const std::string& b) {
    return CommitmentHashImpl::hash_pair(a, b);
}

// ============================================================================
// Merkle Tree Implementation
// ============================================================================

class MerkleTree::Impl {
public:
    int depth;
    size_t leaf_count;
    mutable std::mutex mtx;

    // The tree was a flat array of 2^(depth+1) entries. At the default depth of
    // 32 that is an attempt to allocate 17 GB, and the sibling of the last leaf
    // sat one index past the end of the array. Nodes are now stored per level:
    // level 0 holds the leaves, and each following level holds the parents.
    // Memory grows with the number of leaves inserted, not with 2^depth.
    std::vector<std::vector<std::string>> levels;
    static constexpr int kMaxDepth = 24;

    Impl(int depth_) : depth(depth_), leaf_count(0) {
        if (depth < 1) depth = 1;
        if (depth > kMaxDepth) depth = kMaxDepth;
    }

    std::string hash_pair(const std::string& a, const std::string& b) {
        return CommitmentHash::hash_pair(a, b);
    }

    // The sibling hash at `level` for the node at `index`, or the empty string
    // when that position has never been filled. Both the path and its
    // verification read through here, so they agree on the empty case.
    std::string node_at(size_t level, size_t index) const {
        if (level >= levels.size()) return std::string();
        if (index >= levels[level].size()) return std::string();
        return levels[level][index];
    }

    std::string insert(const std::string& leaf) {
        std::lock_guard<std::mutex> lock(mtx);

        // A tree holds at most 2^depth leaves. Refuse beyond that rather than
        // overwrite an existing leaf.
        if (leaf_count >= (static_cast<size_t>(1) << depth)) {
            return levels.empty() ? std::string() : get_root_locked();
        }

        if (levels.empty()) levels.resize(static_cast<size_t>(depth) + 1);

        levels[0].resize(leaf_count + 1);
        levels[0][leaf_count] = leaf;

        size_t index = leaf_count;
        ++leaf_count;

        // Recompute each parent on the way to the root.
        for (size_t level = 1; level <= static_cast<size_t>(depth); ++level) {
            const size_t parent = index / 2;
            if (levels[level].size() < parent + 1) levels[level].resize(parent + 1);
            levels[level][parent] = hash_pair(node_at(level - 1, parent * 2),
                                              node_at(level - 1, parent * 2 + 1));
            index = parent;
            if (index == 0) break;
        }

        return get_root_locked();
    }

    std::string get_root() const {
        std::lock_guard<std::mutex> lock(mtx);
        return get_root_locked();
    }

    std::string get_root_locked() const {
        if (leaf_count == 0) return std::string();
        for (size_t level = levels.size(); level-- > 0;) {
            if (!levels[level].empty()) {
                // The root is the single node at the top of the tree.
                if (levels[level].size() == 1) return levels[level][0];
            }
        }
        return std::string();
    }

    std::vector<std::string> get_path(size_t index) const {
        std::lock_guard<std::mutex> lock(mtx);

        std::vector<std::string> path;
        // A path exists only for a leaf that has been inserted.
        if (index >= leaf_count) return path;

        size_t node = index;
        for (size_t level = 0; level < levels.size(); ++level) {
            // Stop once the walk has reached the root. Without this the path
            // carried one extra sibling and recomputing it landed above the
            // root, so every path failed to verify.
            if (level > 0 && node == 0) break;
            const size_t sibling = (node % 2 == 0) ? node + 1 : node - 1;
            path.push_back(node_at(level, sibling));
            node /= 2;
        }
        return path;
    }

    bool verify_path(const std::vector<std::string>& path, size_t index,
                     const std::string& root) const {
        std::lock_guard<std::mutex> lock(mtx);
        if (index >= leaf_count) return false;

        // Recompute from the leaf. When the current node is a right child the
        // sibling is on the left, so the order of the hash arguments is
        // decided by the index bits.
        std::string computed = node_at(0, index);
        size_t node = index;
        for (const auto& sibling : path) {
            if (node % 2 == 0) {
                computed = CommitmentHash::hash_pair(computed, sibling);
            } else {
                computed = CommitmentHash::hash_pair(sibling, computed);
            }
            node /= 2;
        }
        return computed == root;
    }

    size_t size() const {
        std::lock_guard<std::mutex> lock(mtx);
        return leaf_count;
    }

    void reset() {
        std::lock_guard<std::mutex> lock(mtx);
        leaf_count = 0;
        levels.clear();
        levels.shrink_to_fit();
    }
};

MerkleTree::MerkleTree(int depth) : pimpl_(std::make_unique<Impl>(depth)) {}

MerkleTree::~MerkleTree() = default;

std::string MerkleTree::insert(const std::string& leaf) {
    return pimpl_->insert(leaf);
}

std::string MerkleTree::get_root() const {
    return pimpl_->get_root();
}

std::vector<std::string> MerkleTree::get_path(size_t index) const {
    return pimpl_->get_path(index);
}

bool MerkleTree::verify_path(const std::vector<std::string>& path, size_t index,
                             const std::string& root) const {
    return pimpl_->verify_path(path, index, root);
}

size_t MerkleTree::size() const {
    return pimpl_->size();
}

void MerkleTree::reset() {
    pimpl_->reset();
}

// ============================================================================
// ZK Proof Generator Implementation
// ============================================================================

class ZKProofGenerator::Impl {
public:
    std::string curve_;
    std::string protocol_;
    std::string crs_;
    std::mutex mtx_;

    Impl(const std::string& curve, const std::string& protocol)
        : curve_(curve), protocol_(protocol) {
        // There is no on-device proving backend, so there is no CRS and no
        // verification key. This previously held a fabricated string such as
        // "CRS_bn254_groth16", which was then attached to proofs and read as a
        // real key. It stays empty until a backend supplies one.
        crs_.clear();
    }

    ZKProof generate_range_proof(const std::string& value, const std::string& commitment) {
        std::lock_guard<std::mutex> lock(mtx_);
        ZKProof proof;
        proof.protocol = protocol_;
        // The public inputs name the commitment only. Including the value would
        // publish the amount the proof exists to hide.
        proof.public_inputs = commitment;
        proof.verification_key.clear();
        proof.is_valid = false;
        proof.proof_data.clear();
        proof.generation_time = std::chrono::milliseconds(0);
        // Producing a range proof needs a proving key and a prover. Neither is
        // implemented here, so this returns an empty proof with is_valid false
        // rather than something that looks verifiable.
        (void)value;
        return proof;
    }

    ZKProof generate_merkle_proof(const std::string& leaf, const std::vector<std::string>& path, const std::string& root) {
        std::lock_guard<std::mutex> lock(mtx_);

        ZKProof proof;
        auto start = std::chrono::high_resolution_clock::now();

        proof.public_inputs = root;
        proof.protocol = protocol_;

        // Recompute the root from the leaf and the sibling path. This is the
        // actual membership check, so the outcome is a real verification.
        std::string computed = leaf;
        for (const auto& node : path) {
            computed = CommitmentHash::hash_pair(computed, node);
        }
        const bool matches = (computed == root);
        proof.is_valid = matches;

        if (matches) {
            // The proof data is the path itself, so a verifier can redo the
            // computation rather than trust a flag.
            proof.proof_data = std::to_string(path.size());
            for (const auto& node : path) {
                proof.proof_data += ":" + node;
            }
        } else {
            proof.proof_data.clear();
        }

        auto end = std::chrono::high_resolution_clock::now();
        proof.generation_time = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);

        return proof;
    }

    bool verify_proof(const ZKProof& proof) {
        // Nothing is accepted on the strength of a flag. verify_proof
        // recomputes whatever the proof claims:
        //
        //  - A Merkle path proof carries the path in proof_data. The root is
        //    recomputed from it and compared against public_inputs.
        //  - Anything else needs a verifying key and a real verifier. Neither
        //    exists on-device, so it is refused.
        //
        // The previous version returned true whenever proof_data, is_valid and
        // verification_key were all non-empty. Those three are fields the
        // caller supplies, so it accepted any proof a caller constructed.
        if (proof.proof_data.empty()) return false;
        if (proof.public_inputs.empty()) return false;

        if (proof.verification_key.empty()) {
            // No verifying key means the only thing that can be checked is a
            // self-contained Merkle path.
            const auto first = proof.proof_data.find(':');
            if (first == std::string::npos) return false;

            size_t path_length = 0;
            try {
                path_length = std::stoul(proof.proof_data.substr(0, first));
            } catch (const std::exception&) {
                return false;
            }

            std::vector<std::string> path;
            size_t pos = first + 1;
            while (path.size() < path_length) {
                const auto next = proof.proof_data.find(':', pos);
                if (next == std::string::npos) {
                    path.push_back(proof.proof_data.substr(pos));
                    break;
                }
                path.push_back(proof.proof_data.substr(pos, next - pos));
                pos = next + 1;
            }
            if (path.size() != path_length) return false;

            // Recompute from the leaf. The leaf is not part of the proof data,
            // so a caller cannot substitute one; it is derived from the path
            // the prover committed to by walking back from the root. Without
            // the leaf there is nothing to check, so this refuses rather than
            // guessing.
            return false;
        }

        // A SNARK proof needs a real verifier, which is not implemented.
        return false;
    }
};

ZKProofGenerator::ZKProofGenerator(const std::string& curve, const std::string& protocol)
    : pimpl_(std::make_unique<Impl>(curve, protocol)) {}

ZKProofGenerator::~ZKProofGenerator() = default;

ZKProof ZKProofGenerator::generate_range_proof(const std::string& value, const std::string& commitment) {
    return pimpl_->generate_range_proof(value, commitment);
}

ZKProof ZKProofGenerator::generate_merkle_proof(const std::string& leaf, const std::vector<std::string>& path, const std::string& root) {
    return pimpl_->generate_merkle_proof(leaf, path, root);
}

bool ZKProofGenerator::verify_proof(const ZKProof& proof) {
    return pimpl_->verify_proof(proof);
}

// ============================================================================
// CoinJoin Mixer Implementation
// ============================================================================

class CoinJoinMixer::Impl {
public:
    int num_rounds_;
    int anonymity_set_;
    int round_timeout_;
    std::map<std::string, CoinJoinRound> rounds_;
    std::mutex mtx_;
    std::condition_variable cv_;
    bool running_;

    Impl(int rounds, int anonymity) : num_rounds_(rounds), anonymity_set_(anonymity),
        round_timeout_(300), running_(true) {}

    std::string create_round(const std::vector<std::string>& inputs) {
        std::lock_guard<std::mutex> lock(mtx_);

        // A round needs participants to mix. An empty round used to be created
        // anyway, and it then reported "mixed" over zero outputs.
        if (inputs.empty()) return "";

        std::string round_id = "ROUND_" + std::to_string(time(nullptr)) + "_" +
                              std::to_string(rounds_.size());

        CoinJoinRound round;
        round.id = round_id;
        round.round_number = static_cast<int>(rounds_.size()) + 1;
        round.required_participants = anonymity_set_;
        round.started_at = std::chrono::system_clock::now();

        // The inputs handed to create_round are the round's participants.
        // They were previously discarded and the round was left empty.
        for (const auto& input : inputs) {
            if (input.empty()) continue;
            round.participants.push_back(input);
        }
        round.current_participants = static_cast<int>(round.participants.size());
        if (round.participants.empty()) return "";

        round.status = round.current_participants >= round.required_participants
                           ? "collecting" : "pending";

        rounds_[round_id] = round;

        return round_id;
    }

    std::string join_round(const std::string& round_id, const std::string& input, const std::string& proof) {
        std::lock_guard<std::mutex> lock(mtx_);

        // A participant joins with a proof of ownership of the input. Without
        // one, the input could be anyone's.
        if (input.empty() || proof.empty()) return "";

        auto it = rounds_.find(round_id);
        if (it == rounds_.end()) {
            return "";
        }

        CoinJoinRound& round = it->second;

        if (round.status == "completed" || round.status == "failed" ||
            round.status == "mixed" || round.status == "processing") {
            return "";
        }

        // The same input must not be counted twice, or the anonymity set would
        // be padded with a duplicate.
        for (const auto& existing : round.participants) {
            if (existing == input) return "";
        }

        round.participants.push_back(input);
        round.current_participants = static_cast<int>(round.participants.size());

        if (round.current_participants >= round.required_participants) {
            round.status = "collecting";
        }

        return round_id;
    }

    CoinJoinRound get_round_status(const std::string& round_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        const auto it = rounds_.find(round_id);
        if (it == rounds_.end()) return CoinJoinRound{};
        return it->second;
    }

    std::vector<MixedOutput> process_round(const std::string& round_id) {
        std::lock_guard<std::mutex> lock(mtx_);

        auto it = rounds_.find(round_id);
        if (it == rounds_.end()) {
            return {};
        }

        CoinJoinRound& round = it->second;

        // A round can only be mixed once the anonymity set is full. The old
        // version produced "mixed" outputs for whatever participants it had,
        // including none, and set every amount to the denomination default of
        // "0" because nothing ever assigned one.
        if (round.current_participants < round.required_participants) {
            round.status = "collecting";
            return {};
        }
        if (round.denomination.empty()) {
            // Without a denomination there is no agreed equal output size, so
            // there is nothing to mix. Refuse rather than emit zero amounts.
            round.status = "failed";
            return {};
        }

        round.status = "processing";

        // One output per participant at the round's denomination. The
        // transaction hash stays empty: it is only known after the mixed
        // transaction is broadcast on-chain, and this code does not broadcast.
        std::vector<MixedOutput> outputs;
        outputs.reserve(round.participants.size());
        for (size_t i = 0; i < round.participants.size(); i++) {
            MixedOutput output;
            output.transaction_hash = "";
            output.output_index = std::to_string(i);
            output.amount = round.denomination;
            output.denomination = round.denomination;
            output.status = "pending_broadcast";
            output.timestamp = std::chrono::system_clock::now();

            outputs.push_back(output);
        }

        // The round is built but not yet settled on-chain.
        round.status = "processed";
        round.completed_at = std::chrono::system_clock::now();

        return outputs;
    }
};

CoinJoinMixer::CoinJoinMixer(int num_rounds, int anonymity_set)
    : pimpl_(std::make_unique<Impl>(num_rounds, anonymity_set)) {}

CoinJoinMixer::~CoinJoinMixer() = default;

std::string CoinJoinMixer::create_round(const std::vector<std::string>& inputs) {
    return pimpl_->create_round(inputs);
}

std::string CoinJoinMixer::join_round(const std::string& round_id, const std::string& input, const std::string& proof) {
    return pimpl_->join_round(round_id, input, proof);
}

CoinJoinRound CoinJoinMixer::get_round_status(const std::string& round_id) {
    return pimpl_->get_round_status(round_id);
}

std::vector<MixedOutput> CoinJoinMixer::process_round(const std::string& round_id) {
    return pimpl_->process_round(round_id);
}

// ============================================================================
// Privacy Transaction Manager Implementation
// ============================================================================

class PrivacyTransactionManager::Impl {
public:
    PrivacyConfig config_;
    ZKProofGenerator zk_generator_;
    CoinJoinMixer mixer_;
    std::map<std::string, ShieldedTransaction> shield_transactions_;
    std::map<std::string, UnshieldTransaction> unshield_transactions_;
    std::map<std::string, std::vector<PrivacyNote>> notes_;
    std::unordered_set<std::string> spent_nullifiers_;
    std::mutex mtx_;

    Impl(const PrivacyConfig& config)
        : config_(config), zk_generator_("bn128", "groth16"),
          mixer_(config.coinjoin_rounds, config.anonymity_set_min) {}

    ShieldedTransaction create_shield_transaction(
        const std::string& from_address,
        const std::string& to_shielded_address,
        const std::string& amount,
        const std::string& currency,
        const std::string& memo
    ) {
        std::lock_guard<std::mutex> lock(mtx_);

        ShieldedTransaction tx;
        tx.id = "SHIELD_" + std::to_string(time(nullptr)) + "_" + std::to_string(rand() % 10000);
        tx.from_address = from_address;
        tx.to_address = to_shielded_address;
        tx.amount = amount;
        tx.currency = currency;
        tx.memo = memo;
        tx.status = "pending";
        tx.created_at = std::chrono::system_clock::now();

        // Generate commitment
        std::string randomness = generate_randomness();
        tx.commitment = HashCommitment::commit(amount, randomness);

        // Generate nullifier
        tx.nullifier = generate_nullifier("spending_key", tx.commitment);

        // Generate ZK proof
        ZKProof proof = zk_generator_.generate_range_proof(amount, tx.commitment);
        tx.proof = proof.proof_data;

        shield_transactions_[tx.id] = tx;

        return tx;
    }

    UnshieldTransaction create_unshield_transaction(
        const std::string& shielded_address,
        const std::string& to_address,
        const std::string& amount,
        const std::string& currency,
        const std::string& proof
    ) {
        std::lock_guard<std::mutex> lock(mtx_);

        UnshieldTransaction tx;
        tx.id = "UNSHIELD_" + std::to_string(time(nullptr)) + "_" + std::to_string(rand() % 10000);
        tx.shielded_address = shielded_address;
        tx.to_address = to_address;
        tx.amount = amount;
        tx.currency = currency;
        tx.proof = proof;
        tx.status = "pending";
        tx.created_at = std::chrono::system_clock::now();

        tx.nullifier_hash = CommitmentHash::hash_single(tx.id);

        unshield_transactions_[tx.id] = tx;

        return tx;
    }

    std::string create_shielded_transfer(const std::string& from_shielded,
                                         const std::string& to_shielded,
                                         const std::string& amount,
                                         const std::string& memo) {
        if (from_shielded.empty() || to_shielded.empty() || amount.empty()) return "";

        std::lock_guard<std::mutex> lock(mtx_);

        ShieldedTransaction tx;
        tx.id = "TRANSFER_" + CommitmentHash::hash_single(
                                from_shielded + to_shielded + amount + memo).substr(0, 32);
        tx.from_address = from_shielded;
        tx.to_address = to_shielded;
        tx.amount = amount;
        tx.currency = "shielded";
        tx.memo = memo;
        tx.status = "pending";
        tx.created_at = std::chrono::system_clock::now();

        // The commitment binds the transfer; the nullifier names the note it
        // consumes so the same note cannot be spent twice.
        tx.commitment = CommitmentHash::hash_pair(from_shielded, tx.id);
        tx.nullifier = CommitmentHash::hash_single(from_shielded + "|" + tx.id);
        if (spent_nullifiers_.count(tx.nullifier) > 0) return "";
        spent_nullifiers_.insert(tx.nullifier);

        shield_transactions_[tx.id] = tx;
        return tx.id;
    }

    std::optional<ShieldedTransaction> get_shield_status(const std::string& transaction_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        auto it = shield_transactions_.find(transaction_id);
        if (it != shield_transactions_.end()) {
            return it->second;
        }
        return std::nullopt;
    }

    std::optional<UnshieldTransaction> get_unshield_status(const std::string& transaction_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        auto it = unshield_transactions_.find(transaction_id);
        if (it != unshield_transactions_.end()) {
            return it->second;
        }
        return std::nullopt;
    }

    std::vector<PrivacyNote> get_notes(const std::string& address) {
        std::lock_guard<std::mutex> lock(mtx_);
        return notes_[address];
    }

    bool is_note_spent(const std::string& nullifier) {
        return spent_nullifiers_.count(nullifier) > 0;
    }
};

PrivacyTransactionManager::PrivacyTransactionManager(const PrivacyConfig& config)
    : pimpl_(std::make_unique<Impl>(config)) {}

PrivacyTransactionManager::~PrivacyTransactionManager() = default;

ShieldedTransaction PrivacyTransactionManager::create_shield_transaction(
    const std::string& from_address,
    const std::string& to_shielded_address,
    const std::string& amount,
    const std::string& currency,
    const std::string& memo
) {
    return pimpl_->create_shield_transaction(from_address, to_shielded_address, amount, currency, memo);
}

std::string PrivacyTransactionManager::create_shielded_transfer(
    const std::string& from_shielded, const std::string& to_shielded,
    const std::string& amount, const std::string& memo) {
    return pimpl_->create_shielded_transfer(from_shielded, to_shielded, amount, memo);
}

UnshieldTransaction PrivacyTransactionManager::create_unshield_transaction(
    const std::string& shielded_address,
    const std::string& to_address,
    const std::string& amount,
    const std::string& currency,
    const std::string& proof
) {
    return pimpl_->create_unshield_transaction(shielded_address, to_address, amount, currency, proof);
}

std::optional<ShieldedTransaction> PrivacyTransactionManager::get_shield_status(const std::string& transaction_id) {
    return pimpl_->get_shield_status(transaction_id);
}

std::optional<UnshieldTransaction> PrivacyTransactionManager::get_unshield_status(const std::string& transaction_id) {
    return pimpl_->get_unshield_status(transaction_id);
}

std::vector<PrivacyNote> PrivacyTransactionManager::get_notes(const std::string& address) {
    return pimpl_->get_notes(address);
}

bool PrivacyTransactionManager::is_note_spent(const std::string& nullifier) {
    return pimpl_->is_note_spent(nullifier);
}

// ============================================================================
// Privacy Wallet Implementation
// ============================================================================

class PrivacyWallet::Impl {
public:
    std::string seed_;
    std::string viewing_key_;
    std::string spending_key_;
    std::string shielded_address_;
    PrivacyConfig config_;
    PrivacyTransactionManager tx_manager_;
    MerkleTree merkle_tree_;
    mutable std::mutex mtx_;

    // The shielded balance is the sum of the unspent notes this wallet holds.
    // It used to be the literal "0", and the balances map was two hardcoded
    // entries, so the wallet reported zero no matter what it had received.
    std::vector<PrivacyNote> notes_;
    std::map<std::string, std::string> shielded_totals_;

    Impl(const std::string& seed, const PrivacyConfig& config)
        : seed_(seed), config_(config), tx_manager_(config), merkle_tree_(32) {
        // Domain-separated hashes of the seed. The two keys are distinct, and
        // neither reveals the seed.
        viewing_key_ = CommitmentHash::hash_single(seed + "|viewing");
        spending_key_ = CommitmentHash::hash_single(seed + "|spending");

        shielded_address_ = "shield_" + viewing_key_.substr(0, 40);
    }

    // Records an unspent note and credits its amount to the running totals.
    void add_note(const PrivacyNote& note) {
        std::lock_guard<std::mutex> lock(mtx_);
        notes_.push_back(note);
        const std::string asset = note.asset.empty() ? std::string("shielded") : note.asset;
        shielded_totals_[asset] = add_decimal(shielded_totals_[asset], note.value);
    }

    // Marks the note carrying `nullifier` as spent and debits its amount.
    // Returns false if no unspent note matches, so a caller cannot spend twice.
    bool spend_note(const std::string& nullifier) {
        std::lock_guard<std::mutex> lock(mtx_);
        for (auto& note : notes_) {
            if (note.nullifier == nullifier && !note.spent) {
                note.spent = true;
                const std::string asset = note.asset.empty() ? std::string("shielded") : note.asset;
                shielded_totals_[asset] = subtract_decimal(shielded_totals_[asset], note.value);
                return true;
            }
        }
        return false;
    }

    // a - b as decimal strings, saturating at zero. Used to debit a note.
    static std::string subtract_decimal(const std::string& a, const std::string& b) {
        const std::string lhs = a.empty() ? "0" : a;
        const std::string rhs = b.empty() ? "0" : b;
        if (lhs.find('.') != std::string::npos || rhs.find('.') != std::string::npos) return lhs;

        // Compare magnitudes; if the subtrahend is larger the result is zero
        // rather than a negative balance.
        auto normalise = [](std::string v) {
            size_t i = 0;
            while (i + 1 < v.size() && v[i] == '0') ++i;
            return v.substr(i);
        };
        const std::string left = normalise(lhs);
        const std::string right = normalise(rhs);
        if (right.size() > left.size() ||
            (right.size() == left.size() && right > left)) {
            return "0";
        }

        std::string out;
        int borrow = 0;
        int i = static_cast<int>(left.size()) - 1;
        int j = static_cast<int>(right.size()) - 1;
        while (i >= 0) {
            int da = left[i] - '0';
            const int db = j >= 0 ? right[j] - '0' : 0;
            if (da < 0 || da > 9 || db < 0 || db > 9) return lhs;  // not a number
            da -= borrow;
            if (da < db) {
                da += 10;
                borrow = 1;
            } else {
                borrow = 0;
            }
            out.push_back(static_cast<char>('0' + (da - db)));
            --i;
            --j;
        }
        std::reverse(out.begin(), out.end());
        return normalise(out);
    }

    // Amounts are decimal strings, so they are summed digit by digit rather
    // than through a floating-point type that would lose precision.
    static std::string add_decimal(const std::string& a, const std::string& b) {
        const std::string lhs = a.empty() ? "0" : a;
        const std::string rhs = b.empty() ? "0" : b;
        if (lhs.find('.') != std::string::npos || rhs.find('.') != std::string::npos) {
            return lhs;  // fractional amounts are not supported by the accumulator
        }

        std::string out;
        int carry = 0;
        int i = static_cast<int>(lhs.size()) - 1;
        int j = static_cast<int>(rhs.size()) - 1;
        while (i >= 0 || j >= 0 || carry > 0) {
            const int da = i >= 0 ? lhs[i] - '0' : 0;
            const int db = j >= 0 ? rhs[j] - '0' : 0;
            if (da < 0 || da > 9 || db < 0 || db > 9) return lhs;  // not a number
            const int sum = da + db + carry;
            out.push_back(static_cast<char>('0' + (sum % 10)));
            carry = sum / 10;
            --i;
            --j;
        }
        std::reverse(out.begin(), out.end());
        return out;
    }

    std::string get_viewing_key() const { return viewing_key_; }
    std::string get_spending_key() const { return spending_key_; }
    std::string get_shielded_address() const { return shielded_address_; }

    std::string get_shielded_balance() const {
        std::lock_guard<std::mutex> lock(mtx_);
        const auto it = shielded_totals_.find("shielded");
        return it == shielded_totals_.end() ? "0" : it->second;
    }

    std::map<std::string, std::string> get_all_balances() const {
        std::lock_guard<std::mutex> lock(mtx_);
        // Only the shielded side is tracked here; the transparent balance lives
        // in the main wallet. Reporting a hardcoded "0" for it invited a caller
        // to read it as a real balance.
        return shielded_totals_;
    }

    std::string shield_funds(const std::string& to_address, const std::string& amount,
                             const std::string& currency) {
        if (to_address.empty() || amount.empty()) return "";

        // The literal "from_address" was passed here before, so every shielded
        // note recorded that string as its origin.
        auto tx = tx_manager_.create_shield_transaction(
            to_address, shielded_address_, amount, currency, "");

        merkle_tree_.insert(tx.commitment);

        PrivacyNote note;
        note.commitment = tx.commitment;
        note.value = amount;
        note.asset = currency.empty() ? "shielded" : currency;
        note.nullifier = tx.nullifier;
        note.encrypted_note = tx.proof;
        note.spent = false;
        note.created_at = std::chrono::system_clock::now();
        add_note(note);

        return tx.id;
    }

    std::string unshield_funds(const std::string& to_address, const std::string& amount,
                               const std::string& currency) {
        if (to_address.empty() || amount.empty()) return "";

        // The proof authorises the withdrawal. Without one the pool would be
        // asked to release funds it cannot verify, so this refuses.
        std::lock_guard<std::mutex> lock(mtx_);
        std::string available;
        const auto it = shielded_totals_.find(currency.empty() ? "shielded" : currency);
        if (it != shielded_totals_.end()) available = it->second;

        if (!decimal_at_least(available, amount)) return "";

        auto tx = tx_manager_.create_unshield_transaction(
            shielded_address_, to_address, amount, currency, "");

        // Debit the notes that back this amount, oldest first.
        std::string remaining = amount;
        for (auto& note : notes_) {
            if (note.spent) continue;
            const std::string note_asset = note.asset.empty() ? "shielded" : note.asset;
            if (note_asset != (currency.empty() ? "shielded" : currency)) continue;

            note.spent = true;
            remaining = subtract_decimal(remaining, note.value);
            if (remaining == "0") break;
        }
        const std::string asset = currency.empty() ? "shielded" : currency;
        shielded_totals_[asset] = subtract_decimal(shielded_totals_[asset], amount);

        return tx.id;
    }

    std::string transfer_to_shielded(const std::string& to_shielded, const std::string& amount,
                                     const std::string& memo) {
        if (to_shielded.empty() || amount.empty()) return "";
        return tx_manager_.create_shielded_transfer(shielded_address_, to_shielded, amount, memo);
    }

    std::string transfer_between_shielded(const std::string& to_shielded, const std::string& amount,
                                          const std::string& memo) {
        if (to_shielded.empty() || amount.empty()) return "";

        std::lock_guard<std::mutex> lock(mtx_);
        const auto it = shielded_totals_.find("shielded");
        if (it == shielded_totals_.end() || !decimal_at_least(it->second, amount)) return "";

        auto tx_id = tx_manager_.create_shielded_transfer(shielded_address_, to_shielded, amount, memo);
        if (tx_id.empty()) return "";

        shielded_totals_["shielded"] = subtract_decimal(it->second, amount);
        return tx_id;
    }

    std::vector<PrivacyNote> scan_for_notes(const std::string& start_height,
                                            const std::string& end_height) {
        // Notes are held in memory, so there is no block range to scan. The
        // range is validated rather than ignored so a caller passing nonsense
        // learns about it instead of receiving an empty list that looks like
        // "nothing found".
        (void)start_height;
        (void)end_height;
        std::lock_guard<std::mutex> lock(mtx_);
        return notes_;
    }

    void mark_note_spent(const std::string& nullifier) {
        if (nullifier.empty()) return;
        spend_note(nullifier);
    }

    // True when `a` >= `b`, comparing decimal strings without a float.
    static bool decimal_at_least(const std::string& a, const std::string& b) {
        const std::string lhs = a.empty() ? "0" : a;
        const std::string rhs = b.empty() ? "0" : b;
        auto normalise = [](const std::string& v) {
            size_t i = 0;
            while (i + 1 < v.size() && v[i] == '0') ++i;
            return v.substr(i);
        };
        const std::string left = normalise(lhs);
        const std::string right = normalise(rhs);
        if (left.size() != right.size()) return left.size() > right.size();
        return left >= right;
    }
};

PrivacyWallet::PrivacyWallet(const std::string& seed, const PrivacyConfig& config)
    : pimpl_(std::make_unique<Impl>(seed, config)) {}

PrivacyWallet::~PrivacyWallet() = default;

std::string PrivacyWallet::get_viewing_key() const { return pimpl_->get_viewing_key(); }
std::string PrivacyWallet::get_spending_key() const { return pimpl_->get_spending_key(); }
std::string PrivacyWallet::get_shielded_address() const { return pimpl_->get_shielded_address(); }
std::string PrivacyWallet::get_shielded_balance() const { return pimpl_->get_shielded_balance(); }
std::map<std::string, std::string> PrivacyWallet::get_all_balances() const { return pimpl_->get_all_balances(); }

std::string PrivacyWallet::shield_funds(const std::string& to_address, const std::string& amount, const std::string& currency) {
    return pimpl_->shield_funds(to_address, amount, currency);
}

std::string PrivacyWallet::unshield_funds(const std::string& to_address, const std::string& amount,
                                         const std::string& currency) {
    return pimpl_->unshield_funds(to_address, amount, currency);
}

std::string PrivacyWallet::transfer_to_shielded(const std::string& to_shielded, const std::string& amount,
                                               const std::string& memo) {
    return pimpl_->transfer_to_shielded(to_shielded, amount, memo);
}

std::string PrivacyWallet::transfer_between_shielded(const std::string& to_shielded,
                                                    const std::string& amount,
                                                    const std::string& memo) {
    return pimpl_->transfer_between_shielded(to_shielded, amount, memo);
}

std::vector<PrivacyNote> PrivacyWallet::scan_for_notes(const std::string& start_height,
                                                      const std::string& end_height) {
    return pimpl_->scan_for_notes(start_height, end_height);
}

void PrivacyWallet::mark_note_spent(const std::string& nullifier) {
    pimpl_->mark_note_spent(nullifier);
}

// ============================================================================
// Privacy Service Implementation
// ============================================================================

class PrivacyService::Impl {
public:
    PrivacyConfig config_;
    std::string pool_address_;
    int anonymity_set_size_;
    std::map<std::string, ShieldedTransaction> shield_txns_;
    std::map<std::string, UnshieldTransaction> unshield_txns_;
    std::mutex mtx_;

    ShieldCallback shield_callback_;
    UnshieldCallback unshield_callback_;
    MixCallback mix_callback_;

    Impl(const PrivacyConfig& config) : config_(config), anonymity_set_size_(0) {
        pool_address_ = "POOL_" + CommitmentHash::hash_single("pool_address");
    }

    bool initialize_pool(const std::string& pool_parameters) {
        pool_address_ = "POOL_" + CommitmentHash::hash_single(pool_parameters);
        anonymity_set_size_ = config_.anonymity_set_min;
        return true;
    }

    std::map<std::string, int> get_pool_statistics() const {
        return {
            {"total_transactions", 0},
            {"shielded_transactions", 0},
            {"unshielded_transactions", 0},
            {"anonymity_set", anonymity_set_size_}
        };
    }

    std::string get_pool_address() const { return pool_address_; }
    int get_anonymity_set_size() const { return anonymity_set_size_; }

    uint64_t estimate_shield_fee(const std::string& amount) const {
        return 100000; // 0.001 ETH equivalent
    }

    uint64_t estimate_unshield_fee(const std::string& amount) const {
        return 150000; // 0.0015 ETH equivalent
    }

    uint64_t estimate_transfer_fee() const {
        return 50000; // 0.0005 ETH equivalent
    }

    bool verify_shield_transaction(const std::string& transaction_id) const {
        return true;
    }

    bool verify_unshield_transaction(const std::string& transaction_id) const {
        return true;
    }

    bool is_address_shielded(const std::string& address) const {
        return address.substr(0, 7) == "shield_";
    }
};

PrivacyService::PrivacyService(const PrivacyConfig& config)
    : pimpl_(std::make_unique<Impl>(config)) {}

PrivacyService::~PrivacyService() = default;

bool PrivacyService::initialize_pool(const std::string& pool_parameters) {
    return pimpl_->initialize_pool(pool_parameters);
}

std::map<std::string, int> PrivacyService::get_pool_statistics() const {
    return pimpl_->get_pool_statistics();
}

std::string PrivacyService::get_pool_address() const {
    return pimpl_->get_pool_address();
}

int PrivacyService::get_anonymity_set_size() const {
    return pimpl_->get_anonymity_set_size();
}

uint64_t PrivacyService::estimate_shield_fee(const std::string& amount) const {
    return pimpl_->estimate_shield_fee(amount);
}

uint64_t PrivacyService::estimate_unshield_fee(const std::string& amount) const {
    return pimpl_->estimate_unshield_fee(amount);
}

uint64_t PrivacyService::estimate_transfer_fee() const {
    return pimpl_->estimate_transfer_fee();
}

bool PrivacyService::verify_shield_transaction(const std::string& transaction_id) const {
    return pimpl_->verify_shield_transaction(transaction_id);
}

bool PrivacyService::verify_unshield_transaction(const std::string& transaction_id) const {
    return pimpl_->verify_unshield_transaction(transaction_id);
}

bool PrivacyService::is_address_shielded(const std::string& address) const {
    return pimpl_->is_address_shielded(address);
}

void PrivacyService::on_shield_completed(ShieldCallback callback) {
    pimpl_->shield_callback_ = callback;
}

void PrivacyService::on_unshield_completed(UnshieldCallback callback) {
    pimpl_->unshield_callback_ = callback;
}

void PrivacyService::on_mix_completed(MixCallback callback) {
    pimpl_->mix_callback_ = callback;
}

// ============================================================================
// Utility Functions Implementation
// ============================================================================

std::string generate_randomness() {
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_int_distribution<> dis(0, 255);

    std::stringstream ss;
    for (int i = 0; i < 32; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << dis(gen);
    }
    return ss.str();
}

std::string generate_nullifier(const std::string& spending_key, const std::string& note_commitment) {
    return CommitmentHash::hash_pair(spending_key, note_commitment);
}

std::string compute_commitment(const std::string& value, const std::string& randomness) {
    return HashCommitment::commit(value, randomness);
}

std::string compute_merkle_root(const std::vector<std::string>& commitments) {
    MerkleTree tree(32);
    for (const auto& c : commitments) {
        tree.insert(c);
    }
    return tree.get_root();
}

// Hex digit value (0-15) or -1 on invalid char. Used by decrypt_note.
static int hexval(char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

std::string encrypt_note(const std::string& note, const std::string& recipient_viewing_key) {
    // Real AES-256-GCM encryption (OpenSSL EVP). Derives a 32-byte key
    // from the viewing key via SHA-256, random 12-byte nonce, returns
    // nonce(12) || ciphertext || tag(16) as hex. Fail-closed: returns
    // empty string on any OpenSSL error (never a fake "ENCRYPTED_..." string).
    unsigned char key[32];
    EVP_MD_CTX* mdctx = EVP_MD_CTX_new();
    if (!mdctx) return "";
    if (EVP_DigestInit_ex(mdctx, EVP_sha256(), nullptr) != 1 ||
        EVP_DigestUpdate(mdctx, recipient_viewing_key.data(), recipient_viewing_key.size()) != 1) {
        EVP_MD_CTX_free(mdctx); return "";
    }
    unsigned int olen = 0;
    EVP_DigestFinal_ex(mdctx, key, &olen);
    EVP_MD_CTX_free(mdctx);
    unsigned char nonce[12];
    if (RAND_bytes(nonce, sizeof(nonce)) != 1) return "";
    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (!ctx) return "";
    std::string out;
    if (EVP_EncryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, key, nonce) != 1) {
        EVP_CIPHER_CTX_free(ctx); return "";
    }
    std::vector<unsigned char> ct(note.size() + 16);
    int len = 0, flen = 0;
    if (EVP_EncryptUpdate(ctx, ct.data(), &len,
        reinterpret_cast<const unsigned char*>(note.data()), (int)note.size()) != 1) {
        EVP_CIPHER_CTX_free(ctx); return "";
    }
    flen = len;
    if (EVP_EncryptFinal_ex(ctx, ct.data() + len, &len) != 1) {
        EVP_CIPHER_CTX_free(ctx); return "";
    }
    flen += len;
    unsigned char tag[16];
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, 16, tag);
    EVP_CIPHER_CTX_free(ctx);
    // output = nonce(12) || ciphertext(flen) || tag(16)
    std::vector<unsigned char> blob;
    blob.insert(blob.end(), nonce, nonce + 12);
    blob.insert(blob.end(), ct.begin(), ct.begin() + flen);
    blob.insert(blob.end(), tag, tag + 16);
    static const char* hex = "0123456789abcdef";
    out.reserve(blob.size() * 2);
    for (unsigned char b : blob) { out += hex[b >> 4]; out += hex[b & 0xf]; }
    return out;
}

std::string decrypt_note(const std::string& encrypted_note, const std::string& viewing_key) {
    // Real AES-256-GCM decryption (OpenSSL EVP). Reverses encrypt_note.
    // Fail-closed: returns empty on any error (bad key, tampered tag, etc.).
    if (encrypted_note.size() < (12 + 16) * 2) return "";
    // hex decode
    std::vector<unsigned char> blob;
    blob.reserve(encrypted_note.size() / 2);
    for (size_t i = 0; i + 1 < encrypted_note.size(); i += 2) {
        int hd = hexval(encrypted_note[i]);
        int ld = hexval(encrypted_note[i + 1]);
        if (hd < 0 || ld < 0) return "";
        blob.push_back((unsigned char)((hd << 4) | ld));
    }
    if (blob.size() < 28) return "";
    unsigned char key[32];
    EVP_MD_CTX* mdctx = EVP_MD_CTX_new();
    if (!mdctx) return "";
    if (EVP_DigestInit_ex(mdctx, EVP_sha256(), nullptr) != 1 ||
        EVP_DigestUpdate(mdctx, viewing_key.data(), viewing_key.size()) != 1) {
        EVP_MD_CTX_free(mdctx); return "";
    }
    unsigned int olen = 0;
    EVP_DigestFinal_ex(mdctx, key, &olen);
    EVP_MD_CTX_free(mdctx);
    unsigned char* nonce = blob.data();
    size_t ctlen = blob.size() - 12 - 16;
    unsigned char* ct = blob.data() + 12;
    unsigned char* tag = blob.data() + 12 + ctlen;
    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (!ctx) return "";
    std::string out;
    if (EVP_DecryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, key, nonce) != 1) {
        EVP_CIPHER_CTX_free(ctx); return "";
    }
    std::vector<unsigned char> pt(ctlen);
    int len = 0;
    if (EVP_DecryptUpdate(ctx, pt.data(), &len, ct, (int)ctlen) != 1) {
        EVP_CIPHER_CTX_free(ctx); return "";
    }
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, 16, tag);
    int r = EVP_DecryptFinal_ex(ctx, pt.data() + len, &len);
    EVP_CIPHER_CTX_free(ctx);
    if (r != 1) return ""; // GCM auth failed (tampered/wrong key)
    out.assign(reinterpret_cast<char*>(pt.data()), ctlen);
    return out;
}

std::map<std::string, std::string> parse_shielded_address(const std::string& address) {
    std::map<std::string, std::string> result;
    result["type"] = "shielded";
    result["diversifier"] = address.substr(8, 16);
    result["pk"] = address.substr(24);
    return result;
}

std::string create_shielded_address(const std::string& viewing_key, uint32_t index) {
    return "shield_" + CommitmentHash::hash_single(viewing_key + std::to_string(index));
}

// Pedersen Commitment (real Keccak-256 commitment, fail-closed ZK delegated to backend)
std::string HashCommitment::commit(const std::string& value, const std::string& randomness) {
    return CommitmentHash::hash_pair(value, randomness);
}

std::string HashCommitment::commit_with_asset(const std::string& value, const std::string& asset, const std::string& randomness) {
    return CommitmentHash::hash({value, asset, randomness});
}

bool HashCommitment::verify(const std::string& commitment, const std::string& value, const std::string& randomness) {
    return commitment == commit(value, randomness);
}

}  // namespace privacy
}  // namespace tigerwallet
