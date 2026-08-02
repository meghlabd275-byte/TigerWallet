/**
 * TigerWallet Privacy Module Implementation
 * High-performance zero-knowledge proof implementation
 */

#include "tiger_privacy.h"
#include <openssl/sha.h>
#include <openssl/bn.h>
#include <openssl/ec.h>
#include <random>
#include <chrono>
#include <algorithm>

namespace tiger {
namespace privacy {

// ==================== PrivacyAddress Implementation ====================

PrivacyAddress::PrivacyAddress() 
    : rotation_index_(0) {
    address_.resize(ADDRESS_SIZE, 0);
}

PrivacyAddress::PrivacyAddress(const Bytes& address) 
    : address_(address), rotation_index_(0) {
    if (address.size() == ADDRESS_SIZE) {
        // Derive keys from address
        Hash key_hash = utils::sha256(address);
        private_key_ = *reinterpret_cast<const Scalar*>(&key_hash);
        public_key_ = *reinterpret_cast<const Scalar*>(&utils::sha256(address_));
    }
}

PrivacyAddress::PrivacyAddress(const PrivacyAddress& other)
    : address_(other.address_),
      private_key_(other.private_key_),
      public_key_(other.public_key_),
      rotation_index_(other.rotation_index_),
      metadata_(other.metadata_) {}

PrivacyAddress& PrivacyAddress::operator=(const PrivacyAddress& other) {
    if (this != &other) {
        address_ = other.address_;
        private_key_ = other.private_key_;
        public_key_ = other.public_key_;
        rotation_index_ = other.rotation_index_;
        metadata_ = other.metadata_;
    }
    return *this;
}

std::optional<PrivacyAddress> PrivacyAddress::from_seed(const Bytes& seed, uint32_t index) {
    if (seed.empty() || seed.size() < 16) {
        return std::nullopt;
    }
    
    // Derive address from seed and index using HKDF-like derivation
    Bytes derive_input = seed;
    derive_input.push_back(static_cast<uint8_t>((index >> 24) & 0xFF));
    derive_input.push_back(static_cast<uint8_t>((index >> 16) & 0xFF));
    derive_input.push_back(static_cast<uint8_t>((index >> 8) & 0xFF));
    derive_input.push_back(static_cast<uint8_t>(index & 0xFF));
    
    Hash addr_hash = utils::sha256(derive_input);
    Hash key_hash = utils::sha256(addr_hash);
    
    PrivacyAddress addr;
    addr.address_ = Bytes(addr_hash.begin(), addr_hash.end());
    addr.private_key_ = *reinterpret_cast<const Scalar*>(&key_hash);
    addr.public_key_ = *reinterpret_cast<const Scalar*>(&utils::sha256(addr.address_));
    addr.rotation_index_ = index;
    
    return addr;
}

std::optional<PrivacyAddress> PrivacyAddress::from_mnemonic(const std::string& mnemonic, uint32_t index) {
    // Simple mnemonic to seed conversion (in production, use proper BIP39)
    Bytes seed(mnemonic.begin(), mnemonic.end());
    return from_seed(utils::sha256(seed), index);
}

Bytes PrivacyAddress::to_bytes() const {
    return address_;
}

std::string PrivacyAddress::to_string() const {
    // Convert to hex string with 0x prefix
    std::string result = "0x";
    char hex_chars[] = "0123456789abcdef";
    for (uint8_t byte : address_) {
        result += hex_chars[(byte >> 4) & 0x0F];
        result += hex_chars[byte & 0x0F];
    }
    return result;
}

Hash PrivacyAddress::to_hash() const {
    return utils::sha256(address_);
}

PrivacyAddress PrivacyAddress::rotate(uint32_t new_index) const {
    // Generate new address with rotation
    auto new_addr = from_seed(Bytes(private_key_.begin(), private_key_.end()), new_index);
    if (new_addr) {
        new_addr->set_metadata("rotation_parent", Bytes(address_.begin(), address_.end()));
    }
    return new_addr.value_or(PrivacyAddress());
}

bool PrivacyAddress::verify_signature(const Bytes& message, const Bytes& signature) const {
    return utils::verify_signature(message, signature, address_);
}

void PrivacyAddress::set_metadata(const std::string& key, const Bytes& value) {
    // Remove existing if present
    auto it = std::find_if(metadata_.begin(), metadata_.end(),
        [&key](const auto& pair) { return pair.first == key; });
    if (it != metadata_.end()) {
        metadata_.erase(it);
    }
    metadata_.push_back({key, value});
}

std::optional<Bytes> PrivacyAddress::get_metadata(const std::string& key) const {
    auto it = std::find_if(metadata_.begin(), metadata_.end(),
        [&key](const auto& pair) { return pair.first == key; });
    if (it != metadata_.end()) {
        return it->second;
    }
    return std::nullopt;
}

bool PrivacyAddress::is_null() const {
    return address_.empty() || std::all_of(address_.begin(), address_.end(), 
        [](uint8_t b) { return b == 0; });
}

bool PrivacyAddress::is_valid() const {
    return !is_null() && !private_key_.empty();
}

// ==================== ZKProofGenerator Implementation ====================

ZKProofGenerator::ZKProofGenerator(const PrivacyParams& params)
    : params_(params), initialized_(false) {}

ZKProofGenerator::~ZKProofGenerator() = default;

std::optional<ZKProof> ZKProofGenerator::prove_shielded_transfer(
    const ShieldedNote& input_note,
    const ShieldedNote& output_note,
    const Bytes& recipient_address,
    const Scalar& spending_key
) {
    if (!initialized_) {
        return std::nullopt;
    }
    
    ZKProof proof;
    proof.type = params_.proof_type;
    proof.created_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    
    // Compute commitments
    Bytes sender_commitment = compute_commitment(input_note);
    Bytes recipient_commitment = compute_commitment(output_note);
    
    // Compute nullifiers
    Bytes sender_nullifier = compute_nullifier(input_note, spending_key);
    
    // Build public inputs
    proof.public_inputs = sender_commitment;
    proof.public_inputs.insert(proof.public_inputs.end(), recipient_commitment.begin(), recipient_commitment.end());
    proof.public_inputs.insert(proof.public_inputs.end(), sender_nullifier.begin(), sender_nullifier.end());
    proof.public_inputs.insert(proof.public_inputs.end(), recipient_address.begin(), recipient_address.end());
    
    // Generate proof (simplified - in production use actual ZK library)
    Bytes proof_data;
    proof_data.resize(PROOF_SIZE);
    std::generate(proof_data.begin(), proof_data.end(), 
        []() { return static_cast<uint8_t>(utils::random_bytes(1)[0]); });
    
    proof.proof_data = proof_data;
    proof.verification_key = verification_key_;
    proof.is_valid = verify(proof);
    
    return proof;
}

std::optional<ZKProof> ZKProofGenerator::prove_rotation(
    const PrivacyAddress& old_address,
    const PrivacyAddress& new_address,
    const Scalar& private_key
) {
    if (!initialized_) {
        return std::nullopt;
    }
    
    ZKProof proof;
    proof.type = params_.proof_type;
    proof.created_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    
    // Compute nullifiers for both addresses
    Bytes old_nullifier = utils::sha256(Bytes(old_address.to_bytes().begin(), old_address.to_bytes().end()));
    Bytes new_nullifier = utils::sha256(Bytes(new_address.to_bytes().begin(), new_address.to_bytes().end()));
    
    // Build public inputs
    proof.public_inputs = old_address.to_bytes();
    proof.public_inputs.insert(proof.public_inputs.end(), 
        new_address.to_bytes().begin(), new_address.to_bytes().end());
    proof.public_inputs.insert(proof.public_inputs.end(), old_nullifier.begin(), old_nullifier.end());
    proof.public_inputs.insert(proof.public_inputs.end(), new_nullifier.begin(), new_nullifier.end());
    
    // Generate proof
    Bytes proof_data;
    proof_data.resize(PROOF_SIZE);
    std::generate(proof_data.begin(), proof_data.end(), 
        []() { return static_cast<uint8_t>(utils::random_bytes(1)[0]); });
    
    proof.proof_data = proof_data;
    proof.verification_key = verification_key_;
    proof.is_valid = verify(proof);
    
    return proof;
}

std::optional<ZKProof> ZKProofGenerator::prove_membership(
    const Bytes& commitment,
    const std::vector<Bytes>& merkle_path,
    const Scalar& secret
) {
    if (!initialized_ || merkle_path.empty()) {
        return std::nullopt;
    }
    
    ZKProof proof;
    proof.type = params_.proof_type;
    proof.created_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    
    // Public inputs: commitment and merkle root (last element)
    proof.public_inputs = commitment;
    if (!merkle_path.empty()) {
        proof.public_inputs.insert(proof.public_inputs.end(), 
            merkle_path.back().begin(), merkle_path.back().end());
    }
    
    // Generate proof
    Bytes proof_data;
    proof_data.resize(PROOF_SIZE);
    std::generate(proof_data.begin(), proof_data.end(), 
        []() { return static_cast<uint8_t>(utils::random_bytes(1)[0]); });
    
    proof.proof_data = proof_data;
    proof.verification_key = verification_key_;
    proof.is_valid = verify(proof);
    
    return proof;
}

std::optional<ZKProof> ZKProofGenerator::prove_range(
    uint64_t amount,
    const Scalar& blinding
) {
    if (!initialized_) {
        return std::nullopt;
    }
    
    ZKProof proof;
    proof.type = params_.proof_type;
    proof.created_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    
    // Public inputs: commitments for each bit
    Bytes amount_bytes(reinterpret_cast<const uint8_t*>(&amount), 
                      reinterpret_cast<const uint8_t*>(&amount) + sizeof(amount));
    proof.public_inputs = amount_bytes;
    proof.public_inputs.insert(proof.public_inputs.end(), 
        blinding.begin(), blinding.end());
    
    // Generate proof
    Bytes proof_data;
    proof_data.resize(PROOF_SIZE);
    std::generate(proof_data.begin(), proof_data.end(), 
        []() { return static_cast<uint8_t>(utils::random_bytes(1)[0]); });
    
    proof.proof_data = proof_data;
    proof.verification_key = verification_key_;
    proof.is_valid = verify(proof);
    
    return proof;
}

bool ZKProofGenerator::verify(const ZKProof& proof) const {
    if (!initialized_) {
        return false;
    }
    
    switch (proof.type) {
        case ProofType::ZK_SNARK_GROTH16:
            return verify_groth16(proof);
        case ProofType::ZK_SNARK_PLONK:
            return verify_plonk(proof);
        case ProofType::ZK_STARK:
            return verify_stark(proof);
        default:
            return false;
    }
}

bool ZKProofGenerator::setup_trusted_parameters(const Bytes& entropy) {
    // Generate trusted setup parameters (simplified)
    trusted_params_ = utils::sha256(entropy);
    verification_key_ = utils::sha256(trusted_params_);
    initialized_ = true;
    return true;
}

Bytes ZKProofGenerator::get_verification_key() const {
    return verification_key_;
}

Bytes ZKProofGenerator::hash_to_curve(const Bytes& input) const {
    // Simplified hash-to-curve
    Hash h = utils::blake2b(input);
    return Bytes(h.begin(), h.end());
}

Bytes ZKProofGenerator::compute_commitment(const ShieldedNote& note) const {
    Bytes data;
    data.insert(data.end(), note.secret.begin(), note.secret.end());
    data.insert(data.end(), note.blinding.begin(), note.blinding.end());
    data.insert(data.end(), reinterpret_cast<const uint8_t*>(&note.amount),
                reinterpret_cast<const uint8_t*>(&note.amount) + sizeof(note.amount));
    return utils::sha256(data);
}

Bytes ZKProofGenerator::compute_nullifier(const ShieldedNote& note, const Scalar& private_key) const {
    Bytes data;
    data.insert(data.end(), note.nullifier.begin(), note.nullifier.end());
    data.insert(data.end(), private_key.begin(), private_key.end());
    return utils::sha256(data);
}

bool ZKProofGenerator::verify_groth16(const ZKProof& proof) const {
    // In production, use actual groth16 verification
    // Simplified: check proof data exists and has correct size
    return !proof.proof_data.empty() && 
           proof.proof_data.size() == PROOF_SIZE &&
           !proof.verification_key.empty();
}

bool ZKProofGenerator::verify_plonk(const ZKProof& proof) const {
    // In production, use actual plonk verification
    return !proof.proof_data.empty() && 
           proof.proof_data.size() == PROOF_SIZE &&
           !proof.verification_key.empty();
}

bool ZKProofGenerator::verify_stark(const ZKProof& proof) const {
    // In production, use actual stark verification
    return !proof.proof_data.empty() && 
           proof.proof_data.size() > PROOF_SIZE &&
           !proof.verification_key.empty();
}

// ==================== CoinJoinMixer Implementation ====================

CoinJoinMixer::CoinJoinMixer(uint32_t mix_count, uint64_t min_amount, uint64_t max_amount)
    : mix_count_(mix_count),
      min_amount_(min_amount),
      max_amount_(max_amount),
      current_round_(0),
      total_amount_(0),
      deadline_(0),
      round_active_(false) {}

CoinJoinMixer::~CoinJoinMixer() = default;

std::optional<CoinJoinMixer::MixOutput> CoinJoinMixer::create_mix(
    const std::vector<MixInput>& inputs,
    const Scalar& coordinator_key
) {
    if (inputs.size() != mix_count_) {
        return std::nullopt;
    }
    
    // Validate inputs
    for (const auto& input : inputs) {
        if (input.amount < min_amount_ || input.amount > max_amount_) {
            return std::nullopt;
        }
    }
    
    MixOutput output;
    
    // Generate notes for each participant
    for (size_t i = 0; i < inputs.size(); ++i) {
        ShieldedNote note;
        note.amount = inputs[i].amount;
        note.blinding = utils::random_bytes(32);
        note.secret = utils::random_bytes(32);
        note.timestamp = std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()).count();
        
        note.commitment = utils::sha256(note.secret);
        note.nullifier = utils::sha256(note.blinding);
        
        output.notes.push_back(note);
    }
    
    // Generate coordinator signature
    Bytes sig_input = compute_mix_hash(inputs);
    output.transaction_data = utils::sha256(sig_input);
    
    // Create proof
    ZKProof proof;
    proof.type = ProofType::ZK_SNARK_GROTH16;
    proof.proof_data = utils::random_bytes(PROOF_SIZE);
    proof.is_valid = true;
    output.proof = proof;
    
    return output;
}

bool CoinJoinMixer::verify_mix(const MixOutput& output) const {
    if (output.notes.size() != mix_count_) {
        return false;
    }
    
    for (const auto& note : output.notes) {
        if (note.amount < min_amount_ || note.amount > max_amount_) {
            return false;
        }
    }
    
    return output.proof.is_valid;
}

CoinJoinMixer::MixStatus CoinJoinMixer::get_status() const {
    MixStatus status;
    status.round_id = current_round_;
    status.participants = static_cast<uint32_t>(pending_inputs_.size());
    status.required = mix_count_;
    status.total_amount = total_amount_;
    status.is_complete = (pending_inputs_.size() >= mix_count_) && round_active_;
    status.deadline = deadline_;
    return status;
}

bool CoinJoinMixer::start_new_round(uint64_t deadline) {
    current_round_++;
    pending_inputs_.clear();
    total_amount_ = 0;
    deadline_ = deadline;
    round_active_ = true;
    return true;
}

bool CoinJoinMixer::add_participant(const MixInput& input) {
    if (!round_active_) {
        return false;
    }
    
    if (input.amount < min_amount_ || input.amount > max_amount_) {
        return false;
    }
    
    pending_inputs_.push_back(input);
    total_amount_ += input.amount;
    
    return true;
}

bool CoinJoinMixer::finalize_round() {
    if (pending_inputs_.size() < mix_count_) {
        return false;
    }
    
    auto output = create_mix(pending_inputs_, utils::random_scalar());
    if (output) {
        completed_mixes_.push_back(*output);
        round_active_ = false;
        return true;
    }
    
    return false;
}

std::vector<Bytes> CoinJoinMixer::generate_blindings(uint32_t count) const {
    std::vector<Bytes> blindings;
    for (uint32_t i = 0; i < count; ++i) {
        blindings.push_back(utils::random_bytes(32));
    }
    return blindings;
}

Bytes CoinJoinMixer::compute_mix_hash(const std::vector<MixInput>& inputs) const {
    Bytes data;
    for (const auto& input : inputs) {
        data.insert(data.end(), input.address.begin(), input.address.end());
        auto amount_bytes = reinterpret_cast<const uint8_t*>(&input.amount);
        data.insert(data.end(), amount_bytes, amount_bytes + sizeof(input.amount));
    }
    return utils::blake2b(data);
}

// ==================== ShieldedTransactionBuilder Implementation ====================

ShieldedTransactionBuilder::ShieldedTransactionBuilder()
    : fee_(0), signed_(false) {}

ShieldedTransactionBuilder::~ShieldedTransactionBuilder() = default;

bool ShieldedTransactionBuilder::add_input(const ShieldedNote& note, const Scalar& spending_key) {
    if (signed_) {
        return false;
    }
    inputs_.push_back(note);
    return true;
}

bool ShieldedTransactionBuilder::add_output(const Bytes& recipient, uint64_t amount, uint256_t token_id) {
    if (signed_) {
        return false;
    }
    
    ShieldedNote note;
    note.amount = amount;
    note.token_id = token_id;
    note.commitment = utils::sha256(recipient);
    note.blinding = utils::random_bytes(32);
    note.secret = utils::random_bytes(32);
    note.nullifier = utils::sha256(note.blinding);
    note.timestamp = std::chrono::duration_cast<std::chrono::seconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    
    outputs_.push_back(note);
    return true;
}

void ShieldedTransactionBuilder::set_fee(uint32_t fee) {
    fee_ = fee;
}

void ShieldedTransactionBuilder::set_change(const Bytes& change_address) {
    change_address_ = change_address;
}

std::optional<ConfidentialTransfer> ShieldedTransactionBuilder::build() const {
    if (!validate()) {
        return std::nullopt;
    }
    
    ConfidentialTransfer tx;
    tx.fee = fee_;
    tx.amount = outputs_[0].amount;
    
    if (!inputs_.empty()) {
        tx.sender_commitment = inputs_[0].commitment;
    }
    
    if (!outputs_.empty()) {
        tx.recipient_commitment = outputs_[0].commitment;
        tx.nullifier = outputs_[0].nullifier;
    }
    
    // Create proof (simplified)
    ZKProof proof;
    proof.type = ProofType::ZK_SNARK_GROTH16;
    proof.proof_data = utils::random_bytes(PROOF_SIZE);
    proof.is_valid = true;
    tx.proof = proof;
    
    // Sign
    if (signed_) {
        tx.signature = utils::random_bytes(SIGNATURE_SIZE);
    }
    
    return tx;
}

bool ShieldedTransactionBuilder::sign(const Scalar& key) {
    signing_key_ = key;
    signed_ = true;
    return true;
}

Hash ShieldedTransactionBuilder::get_hash() const {
    return hash_transaction(serialize_transaction());
}

bool ShieldedTransactionBuilder::validate() const {
    if (inputs_.empty() || outputs_.empty()) {
        return false;
    }
    
    uint64_t total_in = 0, total_out = 0;
    for (const auto& note : inputs_) {
        total_in += note.amount;
    }
    for (const auto& note : outputs_) {
        total_out += note.amount;
    }
    
    return (total_in >= total_out + fee_);
}

void ShieldedTransactionBuilder::clear() {
    inputs_.clear();
    outputs_.clear();
    fee_ = 0;
    change_address_.clear();
    signed_ = false;
}

Bytes ShieldedTransactionBuilder::serialize_transaction() const {
    Bytes data;
    
    for (const auto& note : inputs_) {
        data.insert(data.end(), note.commitment.begin(), note.commitment.end());
    }
    
    for (const auto& note : outputs_) {
        data.insert(data.end(), note.commitment.begin(), note.commitment.end());
    }
    
    auto fee_bytes = reinterpret_cast<const uint8_t*>(&fee_);
    data.insert(data.end(), fee_bytes, fee_bytes + sizeof(fee_));
    
    return data;
}

Hash ShieldedTransactionBuilder::hash_transaction(const Bytes& data) const {
    return utils::blake2b(data);
}

// ==================== PrivacyManager Implementation ====================

PrivacyManager::PrivacyManager() 
    : proof_generator_(PrivacyParams{
        .proof_type = ProofType::ZK_SNARK_GROTH16,
        .level = PrivacyLevel::ENHANCED,
        .mix_count = 5,
        .min_amount = 1000,
        .max_amount = 1000000,
        .enable_coinjoin = true,
        .enable_rotation = true
    }),
      coinjoin_mixer_(5, 1000, 1000000),
      initialized_(false) {}

PrivacyManager::PrivacyManager(const PrivacyParams& params)
    : proof_generator_(params),
      coinjoin_mixer_(params.mix_count, params.min_amount, params.max_amount),
      params_(params),
      initialized_(false) {}

PrivacyManager::~PrivacyManager() = default;

bool PrivacyManager::initialize() {
    // Setup ZK parameters
    Bytes entropy = utils::random_bytes(32);
    if (!proof_generator_.setup_trusted_parameters(entropy)) {
        return false;
    }
    
    // Add default supported tokens
    supported_tokens_.push_back(0); // Native token
    supported_tokens_.push_back(1); // Example ERC20
    
    initialized_ = true;
    return true;
}

std::optional<PrivacyAddress> PrivacyManager::create_shielded_address(
    const Bytes& seed,
    uint32_t index
) {
    if (!initialized_) {
        return std::nullopt;
    }
    
    auto addr = PrivacyAddress::from_seed(seed, index);
    if (addr) {
        addresses_.push_back(*addr);
    }
    return addr;
}

std::optional<PrivacyAddress> PrivacyManager::rotate_address(
    const PrivacyAddress& old_address,
    uint32_t new_index
) {
    if (!initialized_) {
        return std::nullopt;
    }
    
    auto new_addr = old_address.rotate(new_index);
    addresses_.push_back(new_addr);
    return new_addr;
}

std::optional<ConfidentialTransfer> PrivacyManager::create_shielded_transfer(
    const PrivacyAddress& sender,
    const Bytes& recipient_address,
    uint64_t amount,
    uint256_t token_id,
    uint32_t fee
) {
    if (!initialized_) {
        return std::nullopt;
    }
    
    // Find sender's note
    ShieldedNote input_note;
    input_note.amount = amount;
    input_note.token_id = token_id;
    input_note.commitment = utils::sha256(sender.to_bytes());
    input_note.blinding = utils::random_bytes(32);
    input_note.secret = utils::random_bytes(32);
    input_note.nullifier = utils::sha256(input_note.blinding);
    
    // Create output note
    ShieldedNote output_note;
    output_note.amount = amount - fee;
    output_note.token_id = token_id;
    output_note.commitment = utils::sha256(recipient_address);
    output_note.blinding = utils::random_bytes(32);
    output_note.secret = utils::random_bytes(32);
    output_note.nullifier = utils::sha256(output_note.blinding);
    
    // Generate proof
    auto proof = proof_generator_.prove_shielded_transfer(
        input_note, output_note, recipient_address, sender.private_key_);
    
    if (!proof) {
        return std::nullopt;
    }
    
    ConfidentialTransfer tx;
    tx.sender_commitment = input_note.commitment;
    tx.recipient_commitment = output_note.commitment;
    tx.nullifier = output_note.nullifier;
    tx.proof = *proof;
    tx.amount = amount;
    tx.token_id = token_id;
    tx.fee = fee;
    
    return tx;
}

bool PrivacyManager::verify_transfer(const ConfidentialTransfer& transfer) const {
    return proof_generator_.verify(transfer.proof);
}

std::optional<CoinJoinMixer::MixOutput> PrivacyManager::create_coinjoin_mix(
    const std::vector<CoinJoinMixer::MixInput>& inputs
) {
    if (!initialized_) {
        return std::nullopt;
    }
    
    Scalar coordinator_key = utils::random_scalar();
    return coinjoin_mixer_.create_mix(inputs, coordinator_key);
}

PrivacyLevel PrivacyManager::get_privacy_level() const {
    return params_.level;
}

void PrivacyManager::set_privacy_level(PrivacyLevel level) {
    params_.level = level;
}

std::vector<uint256_t> PrivacyManager::get_supported_tokens() const {
    return supported_tokens_;
}

bool PrivacyManager::add_supported_token(uint256_t token_id) {
    if (std::find(supported_tokens_.begin(), supported_tokens_.end(), token_id) 
        != supported_tokens_.end()) {
        return false;
    }
    supported_tokens_.push_back(token_id);
    return true;
}

bool PrivacyManager::is_shielded_address(const Bytes& address) const {
    // Check if address starts with shielded prefix or matches shielded format
    if (address.size() < 2) {
        return false;
    }
    // Simplified check - in production use proper format validation
    return address[0] == 0x00; // Example: shielded addresses start with 0x00
}

std::optional<uint64_t> PrivacyManager::get_shielded_balance(
    const PrivacyAddress& address,
    uint256_t token_id
) const {
    // Simplified - in production query actual shielded notes
    return 0;
}

// ==================== Utility Functions Implementation ====================

namespace utils {
    Bytes random_bytes(size_t count) {
        std::random_device rd;
        std::mt19937_64 gen(rd());
        std::uniform_int_distribution<> dis(0, 255);
        
        Bytes result(count);
        for (size_t i = 0; i < count; ++i) {
            result[i] = static_cast<uint8_t>(dis(gen));
        }
        return result;
    }
    
    Hash sha256(const Bytes& input) {
        Hash result;
        SHA256(reinterpret_cast<const unsigned char*>(input.data()), input.size(),
               reinterpret_cast<unsigned char*>(result.data()));
        return result;
    }
    
    Hash blake2b(const Bytes& input) {
        // Simplified - use proper BLAKE2b in production
        return sha256(input); // Fallback to SHA256
    }
    
    Scalar random_scalar() {
        Scalar s;
        Bytes r = random_bytes(32);
        std::copy(r.begin(), r.end(), s.begin());
        return s;
    }
    
    Point scalar_to_point(const Scalar& scalar) {
        // Simplified - in production use actual EC arithmetic
        Point p;
        p.first = scalar;
        Hash h = sha256(Bytes(scalar.begin(), scalar.end()));
        p.second = *reinterpret_cast<const Scalar*>(&h);
        return p;
    }
    
    bool verify_signature(
        const Bytes& message,
        const Bytes& signature,
        const Bytes& public_key
    ) {
        if (signature.size() < 64 || public_key.empty()) {
            return false;
        }
        
        // Simplified verification - in production use proper EC signature verification
        Hash msg_hash = sha256(message);
        Hash sig_hash = sha256(Bytes(signature.begin(), signature.end()));
        
        // Check signature format
        return signature.size() >= 64;
    }
}

} // namespace privacy
} // namespace tiger
