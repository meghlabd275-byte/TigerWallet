/**
 * TigerWallet Privacy Features Implementation
 * Ultra-low latency C++ implementation for ZK proofs, CoinJoin, and privacy operations
 */

#include "privacy.hpp"
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
#include <queue>
#include <cstring>
#include <openssl/sha.h>
#include <openssl/ripemd.h>
#include <openssl/ec.h>
#include <openssl/bn.h>
#include <openssl/obj_mac.h>

using namespace tigerwallet::privacy;

// ============================================================================
// Cryptographic Primitives Implementation
// ============================================================================

class PoseidonHash::Impl {
public:
    static constexpr int STATE_SIZE = 8;
    
    std::string hash(const std::vector<std::string>& inputs, int rate) {
        // Simplified Poseidon-like hash
        std::vector<uint8_t> state(STATE_SIZE * 32, 0);
        
        for (const auto& input : inputs) {
            // Input mixing
            for (size_t i = 0; i < std::min(inputs.size(), (size_t)rate); i++) {
                const uint8_t* input_bytes = reinterpret_cast<const uint8_t*>(input.c_str());
                size_t input_len = input.length();
                for (size_t j = 0; j < 32 && (i * 32 + j) < state.size(); j++) {
                    state[i * 32 + j] ^= input_bytes[j % input_len];
                }
            }
            
            // Apply sponge permutation (simplified)
            for (int round = 0; round < 8; round++) {
                // Non-linear layer (SubBytes equivalent)
                for (auto& byte : state) {
                    byte = byte ^ (255 - byte);
                }
                
                // Linear layer (ShiftRows equivalent)
                for (size_t i = 0; i < STATE_SIZE; i++) {
                    for (size_t j = 0; j < 32; j++) {
                        size_t idx = (i * 32 + j + i * 5) % state.size();
                        state[idx] ^= state[i * 32 + j];
                    }
                }
            }
        }
        
        // Output
        std::string result;
        for (size_t i = 0; i < 32; i++) {
            char hex[3];
            snprintf(hex, sizeof(hex), "%02x", state[i]);
            result += hex;
        }
        
        return result;
    }
    
    static std::string hash_single(const std::string& input) {
        Impl impl;
        return impl.hash({input}, 4);
    }
    
    static std::string hash_pair(const std::string& a, const std::string& b) {
        Impl impl;
        return impl.hash({a, b}, 4);
    }
};

std::string PoseidonHash::hash(const std::vector<std::string>& inputs, int rate) {
    Impl impl;
    return impl.hash(inputs, rate);
}

std::string PoseidonHash::hash_single(const std::string& input) {
    return Impl::hash_single(input);
}

std::string PoseidonHash::hash_pair(const std::string& a, const std::string& b) {
    return Impl::hash_pair(a, b);
}

// ============================================================================
// Merkle Tree Implementation
// ============================================================================

class MerkleTree::Impl {
public:
    std::vector<std::string> tree;
    int depth;
    size_t leaf_count;
    std::mutex mtx;
    
    Impl(int depth_) : depth(depth_), leaf_count(0) {
        tree.resize(1 << (depth_ + 1));
    }
    
    std::string hash_pair(const std::string& a, const std::string& b) {
        return PoseidonHash::hash_pair(a, b);
    }
    
    std::string insert(const std::string& leaf) {
        std::lock_guard<std::mutex> lock(mtx);
        
        size_t leaf_index = leaf_count;
        tree[leaf_count + (1 << depth)] = leaf;
        leaf_count++;
        
        // Update parents
        size_t current = leaf_index + (1 << depth);
        while (current > 1) {
            current /= 2;
            tree[current] = hash_pair(tree[current * 2], tree[current * 2 + 1]);
        }
        
        return tree[1]; // Return root
    }
    
    std::string get_root() const {
        std::lock_guard<std::mutex> lock(mtx);
        return tree[1];
    }
    
    std::vector<std::string> get_path(size_t index) const {
        std::lock_guard<std::mutex> lock(mtx);
        
        std::vector<std::string> path;
        size_t leaf_index = index + (1 << depth);
        
        while (leaf_index > 1) {
            size_t sibling = (leaf_index % 2 == 0) ? leaf_index + 1 : leaf_index - 1;
            path.push_back(tree[sibling]);
            leaf_index /= 2;
        }
        
        return path;
    }
    
    size_t size() const {
        return leaf_count;
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

size_t MerkleTree::size() const {
    return pimpl_->size();
}

void MerkleTree::reset() {
    std::lock_guard<std::mutex> lock(pimpl_->mtx);
    pimpl_->leaf_count = 0;
    pimpl_->tree.clear();
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
        // Generate placeholder CRS
        crs_ = "CRS_" + curve + "_" + protocol;
    }
    
    ZKProof generate_range_proof(const std::string& value, const std::string& commitment) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        ZKProof proof;
        auto start = std::chrono::high_resolution_clock::now();
        
        // Simplified range proof generation
        // In production, this would use actual ZK-SNARK libraries
        
        std::string proof_data = "RANGE_PROOF_" + commitment + "_" + value;
        proof.proof_data = proof_data;
        proof.public_inputs = commitment + "," + value;
        proof.verification_key = "VK_" + curve_;
        proof.protocol = protocol_;
        
        // Simulate proof generation
        proof.is_valid = true;
        
        auto end = std::chrono::high_resolution_clock::now();
        proof.generation_time = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
        
        return proof;
    }
    
    ZKProof generate_merkle_proof(const std::string& leaf, const std::vector<std::string>& path, const std::string& root) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        ZKProof proof;
        auto start = std::chrono::high_resolution_clock::now();
        
        proof.proof_data = "MERKLE_PROOF_" + leaf + "_" + root;
        proof.public_inputs = leaf + "," + root;
        proof.verification_key = "VK_MERKLE_" + curve_;
        proof.protocol = protocol_;
        
        // Verify path (simplified)
        std::string computed = leaf;
        for (const auto& node : path) {
            computed = PoseidonHash::hash_pair(computed, node);
        }
        proof.is_valid = (computed == root);
        
        auto end = std::chrono::high_resolution_clock::now();
        proof.generation_time = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
        
        return proof;
    }
    
    bool verify_proof(const ZKProof& proof) {
        // Simplified verification
        return proof.is_valid;
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
        
        std::string round_id = "ROUND_" + std::to_string(time(nullptr)) + "_" + 
                              std::to_string(rounds_.size());
        
        CoinJoinRound round;
        round.id = round_id;
        round.round_number = rounds_.size() + 1;
        round.status = "pending";
        round.required_participants = anonymity_set_;
        round.current_participants = 0;
        round.started_at = std::chrono::system_clock::now();
        
        rounds_[round_id] = round;
        
        return round_id;
    }
    
    std::string join_round(const std::string& round_id, const std::string& input, const std::string& proof) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        auto it = rounds_.find(round_id);
        if (it == rounds_.end()) {
            return "";
        }
        
        CoinJoinRound& round = it->second;
        
        if (round.status == "completed" || round.status == "failed") {
            return "";
        }
        
        round.participants.push_back(input);
        round.current_participants++;
        
        // Check if we have enough participants
        if (round.current_participants >= round.required_participants) {
            round.status = "collecting";
        }
        
        return round_id;
    }
    
    CoinJoinRound get_round_status(const std::string& round_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        return rounds_.at(round_id);
    }
    
    std::vector<MixedOutput> process_round(const std::string& round_id) {
        std::lock_guard<std::mutex> lock(mtx_);
        
        auto it = rounds_.find(round_id);
        if (it == rounds_.end()) {
            return {};
        }
        
        CoinJoinRound& round = it->second;
        round.status = "processing";
        
        // Simulate CoinJoin processing
        std::vector<MixedOutput> outputs;
        
        // Generate mixed outputs
        for (size_t i = 0; i < round.participants.size(); i++) {
            MixedOutput output;
            output.transaction_hash = "TX_" + round_id + "_" + std::to_string(i);
            output.output_index = std::to_string(i);
            output.amount = "100"; // Simplified
            output.denomination = "1";
            output.status = "confirmed";
            output.timestamp = std::chrono::system_clock::now();
            
            outputs.push_back(output);
        }
        
        round.status = "completed";
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
        tx.commitment = PedersenCommitment::commit(amount, randomness);
        
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
        
        tx.nullifier_hash = PoseidonHash::hash_single(tx.id);
        
        unshield_transactions_[tx.id] = tx;
        
        return tx;
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
    return false; // Simplified
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
    std::mutex mtx_;
    
    Impl(const std::string& seed, const PrivacyConfig& config)
        : seed_(seed), config_(config), tx_manager_(config), merkle_tree_(32) {
        // Derive keys from seed (simplified)
        viewing_key_ = PoseidonHash::hash_single(seed + "_viewing");
        spending_key_ = PoseidonHash::hash_single(seed + "_spending");
        
        // Generate shielded address
        shielded_address_ = "shield_" + viewing_key_.substr(0, 40);
    }
    
    std::string get_viewing_key() const { return viewing_key_; }
    std::string get_spending_key() const { return spending_key_; }
    std::string get_shielded_address() const { return shielded_address_; }
    
    std::string get_shielded_balance() const { return "0"; }
    
    std::map<std::string, std::string> get_all_balances() const {
        return {{"shielded", "0"}, {"transparent", "0"}};
    }
    
    std::string shield_funds(const std::string& to_address, const std::string& amount, const std::string& currency) {
        auto tx = tx_manager_.create_shield_transaction(
            "from_address", 
            shielded_address_, 
            amount, 
            currency, 
            ""
        );
        
        merkle_tree_.insert(tx.commitment);
        
        return tx.id;
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

std::string PrivacyWallet::unshield_funds(const std::string& to_address, const std::string& amount, const std::string& currency) {
    return "";
}

std::string PrivacyWallet::transfer_to_shielded(const std::string& to_shielded, const std::string& amount, const std::string& memo) {
    return "";
}

std::string PrivacyWallet::transfer_between_shielded(const std::string& to_shielded, const std::string& amount, const std::string& memo) {
    return "";
}

std::vector<PrivacyNote> PrivacyWallet::scan_for_notes(const std::string& start_height, const std::string& end_height) {
    return {};
}

void PrivacyWallet::mark_note_spent(const std::string& nullifier) {}

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
        pool_address_ = "POOL_" + PoseidonHash::hash_single("pool_address");
    }
    
    bool initialize_pool(const std::string& pool_parameters) {
        pool_address_ = "POOL_" + PoseidonHash::hash_single(pool_parameters);
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
    return PoseidonHash::hash_pair(spending_key, note_commitment);
}

std::string compute_commitment(const std::string& value, const std::string& randomness) {
    return PedersenCommitment::commit(value, randomness);
}

std::string compute_merkle_root(const std::vector<std::string>& commitments) {
    MerkleTree tree(32);
    for (const auto& c : commitments) {
        tree.insert(c);
    }
    return tree.get_root();
}

std::string encrypt_note(const std::string& note, const std::string& recipient_viewing_key) {
    return "ENCRYPTED_" + note + "_FOR_" + recipient_viewing_key;
}

std::string decrypt_note(const std::string& encrypted_note, const std::string& viewing_key) {
    return encrypted_note; // Simplified
}

std::map<std::string, std::string> parse_shielded_address(const std::string& address) {
    std::map<std::string, std::string> result;
    result["type"] = "shielded";
    result["diversifier"] = address.substr(8, 16);
    result["pk"] = address.substr(24);
    return result;
}

std::string create_shielded_address(const std::string& viewing_key, uint32_t index) {
    return "shield_" + PoseidonHash::hash_single(viewing_key + std::to_string(index));
}

// Pedersen Commitment stub
std::string PedersenCommitment::commit(const std::string& value, const std::string& randomness) {
    return PoseidonHash::hash_pair(value, randomness);
}

std::string PedersenCommitment::commit_with_asset(const std::string& value, const std::string& asset, const std::string& randomness) {
    return PoseidonHash::hash({value, asset, randomness}, 3);
}

bool PedersenCommitment::verify(const std::string& commitment, const std::string& value, const std::string& randomness) {
    return commitment == commit(value, randomness);
}
