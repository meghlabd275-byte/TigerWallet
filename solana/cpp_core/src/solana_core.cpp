/**
 * TigerWallet - Solana Core Implementation
 * High-performance C++ implementation
 */

#include "solana_core.h"
#include <sstream>
#include <iomanip>
#include <algorithm>
#include <cstring>
#include <openssl/sha.h>
#include <openssl/ripemd.h>

// Base58 alphabet
static const char* BASE58_ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

namespace tiger {
namespace solana {

// ============ WalletAddress ============

std::string WalletAddress::to_base58() const {
    return utils::encode_base58(bytes.data(), SIZE);
}

WalletAddress WalletAddress::from_base58(const std::string& base58) {
    WalletAddress addr;
    auto decoded = utils::decode_base58(base58);
    if (decoded.size() >= SIZE) {
        std::memcpy(addr.bytes.data(), decoded.data(), SIZE);
    }
    return addr;
}

WalletAddress WalletAddress::from_pubkey(const PublicKey& pubkey) {
    WalletAddress addr;
    std::memcpy(addr.bytes.data(), pubkey.data(), SIZE);
    return addr;
}

bool WalletAddress::is_valid() const {
    // Check if all zeros (invalid)
    for (int i = 0; i < SIZE; i++) {
        if (bytes[i] != 0) return true;
    }
    return false;
}

bool WalletAddress::operator==(const WalletAddress& other) const {
    return bytes == other.bytes;
}

// ============ TokenAddress ============

std::string TokenAddress::to_base58() const {
    return utils::encode_base58(bytes.data(), SIZE);
}

TokenAddress TokenAddress::from_base58(const std::string& base58) {
    TokenAddress addr;
    auto decoded = utils::decode_base58(base58);
    if (decoded.size() >= SIZE) {
        std::memcpy(addr.bytes.data(), decoded.data(), SIZE);
    }
    return addr;
}

TokenAddress TokenAddress::create(const WalletAddress& mint, const WalletAddress& owner) {
    // PDA derivation is implemented in the audited Rust core
    // (solana/rust/src/lib.rs `find_program_address`): real
    // sha256(seeds || program_id || bump) with 255->1 bump-seed search and
    // on-curve rejection. The C++ side must NOT fabricate an address with a
    // bare SHA-256 hash -- that is not a valid Solana PDA and would collide
    // with on-chain addresses. Return an all-zero sentinel; callers must use
    // the Rust core for a real PDA.
    (void)mint;
    (void)owner;
    TokenAddress addr;
    addr.bytes.fill(0);
    return addr;
}

// ============ TransactionInstruction ============

std::vector<uint8_t> TransactionInstruction::serialize() const {
    std::vector<uint8_t> result;
    
    // Account keys
    result.push_back(accounts.size());
    for (const auto& key : accounts) {
        result.insert(result.end(), key.bytes.begin(), key.bytes.end());
    }
    
    // Program ID
    result.insert(result.end(), program_id.bytes.begin(), program_id.bytes.end());
    
    // Data
    result = utils::encode_short_vec(data);
    
    return result;
}

TransactionInstruction TransactionInstruction::deserialize(
    const std::vector<uint8_t>& data
) {
    TransactionInstruction inst;
    
    size_t idx = 0;
    
    // Account keys
    uint8_t num_accounts = data[idx++];
    for (int i = 0; i < num_accounts && idx < data.size(); i++) {
        WalletAddress key;
        std::memcpy(key.bytes.data(), &data[idx], 32);
        inst.accounts.push_back(key);
        idx += 32;
    }
    
    // Program ID
    if (idx + 32 <= data.size()) {
        std::memcpy(inst.program_id.bytes.data(), &data[idx], 32);
        idx += 32;
    }
    
    // Data
    inst.data = utils::decode_short_vec(std::vector<uint8_t>(data.begin() + idx, data.end()));
    
    return inst;
}

// ============ Message ============

std::vector<WalletAddress> Message::get_signers() const {
    std::vector<WalletAddress> signers;
    for (size_t i = 0; i < header.num_required_signatures && i < account_keys.size(); i++) {
        signers.push_back(account_keys[i]);
    }
    return signers;
}

std::vector<uint8_t> Message::serialize() const {
    std::vector<uint8_t> result;
    
    // Header
    result.push_back(header.num_required_signatures);
    result.push_back(header.num_readonly_signed_accounts);
    result.push_back(header.num_readonly_unsigned_accounts);
    
    // Account keys
    result = utils::encode_short_vec(result);
    for (const auto& key : account_keys) {
        result.insert(result.end(), key.bytes.begin(), key.bytes.end());
    }
    
    // Recent blockhash
    result.insert(result.end(), recent_blockhash.begin(), recent_blockhash.end());
    
    // Instructions
    std::vector<uint8_t> instr_data;
    for (const auto& instr : instructions) {
        auto serialized = instr.serialize();
        instr_data.insert(instr_data.end(), serialized.begin(), serialized.end());
    }
    auto encoded_instr = utils::encode_short_vec(instr_data);
    result.insert(result.end(), encoded_instr.begin(), encoded_instr.end());
    
    return result;
}

Hash Message::hash() const {
    Hash h;
    auto serialized = serialize();
    SHA256(serialized.data(), serialized.size(), h.data());
    return h;
}

// ============ Transaction ============

void Transaction::add_instruction(const TransactionInstruction& instruction) {
    message.add_instruction(instruction);
}

void Transaction::add_signature(const Signature& signature) {
    signatures.push_back(signature);
}

std::vector<uint8_t> Transaction::serialize() const {
    std::vector<uint8_t> result;
    
    // Signatures
    result = utils::encode_short_vec(result);
    for (const auto& sig : signatures) {
        result.insert(result.end(), sig.begin(), sig.end());
    }
    
    // Message
    auto message_data = message.serialize();
    result.insert(result.end(), message_data.begin(), message_data.end());
    
    return result;
}

Transaction Transaction::deserialize(const std::vector<uint8_t>& data) {
    Transaction tx;
    // Simplified - would fully deserialize in production
    return tx;
}

Hash Transaction::hash() const {
    return message.hash();
}

// ============ Account Parsing ============

TokenAccount TokenAccount::parse(const std::vector<uint8_t>& data) {
    TokenAccount account;
    
    if (data.size() < 165) return account;
    
    size_t offset = 0;
    
    // Mint (32 bytes)
    std::memcpy(account.mint.bytes.data(), &data[offset], 32);
    offset += 32;
    
    // Owner (32 bytes)
    std::memcpy(account.owner.bytes.data(), &data[offset], 32);
    offset += 32;
    
    // Amount (8 bytes)
    std::memcpy(&account.amount, &data[offset], 8);
    offset += 8;
    
    // Delegate (8 bytes)
    std::memcpy(&account.delegate, &data[offset], 8);
    offset += 8;
    
    // Delegated amount (8 bytes)
    std::memcpy(&account.delegated_amount, &data[offset], 8);
    offset += 8;
    
    // State (1 byte)
    account.is_initialized = (data[offset] == 1);
    account.is_frozen = (data[offset] == 2);
    offset += 1;
    
    // Close authority option (1 byte) + authority (32 bytes) if present
    offset += 33;
    
    // Mint authority option (1 byte) + authority (32 bytes) if present
    offset += 33;
    
    // Decimals (1 byte)
    if (offset < data.size()) {
        account.decimals = data[offset];
    }
    
    return account;
}

MintAccount MintAccount::parse(const std::vector<uint8_t>& data) {
    MintAccount account;
    
    if (data.size() < 82) return account;
    
    size_t offset = 0;
    
    // Mint authority option (1 byte)
    bool has_mint_authority = data[offset] == 1;
    offset += 1;
    
    if (has_mint_authority) {
        std::memcpy(account.mint_authority->bytes.data(), &data[offset], 32);
        offset += 32;
    }
    
    // Supply (8 bytes)
    std::memcpy(&account.supply, &data[offset], 8);
    offset += 8;
    
    // Decimals (1 byte)
    account.decimals = data[offset++];
    
    // Is initialized (1 byte)
    account.is_initialized = data[offset] == 1;
    offset += 1;
    
    // Freeze authority option (1 byte)
    bool has_freeze_authority = data[offset] == 1;
    offset += 1;
    
    if (has_freeze_authority) {
        std::memcpy(account.freeze_authority->bytes.data(), &data[offset], 32);
    }
    
    return account;
}

StakeAccount StakeAccount::parse(const std::vector<uint8_t>& data) {
    StakeAccount account;
    
    if (data.size() < 200) return account;
    
    size_t offset = 0;
    
    // RENT_EPOCH (8 bytes)
    offset += 8;
    
    // STAKE_EPOCHS (8 bytes)
    offset += 8;
    
    // Delegation (168 bytes)
    offset += 16; // vote_account
    
    // Activated stake (8 bytes)
    std::memcpy(&account.staked_lamports, &data[offset], 8);
    offset += 8;
    
    // Deactivated stake (8 bytes)
    offset += 8;
    
    // Activating (8 bytes)
    offset += 8;
    
    // Deactivating (8 bytes)
    offset += 8;
    
    // Lockup (48 bytes)
    offset += 48;
    
    // Authority (32 bytes)
    std::memcpy(account.authority.bytes.data(), &data[offset], 32);
    
    return account;
}

// ============ RPCClient ============

RPCClient::RPCClient(const Config& config) : config_(config) {}

std::optional<Account> RPCClient::get_account(const WalletAddress& address) const {
    // In production, would call RPC
    return std::nullopt;
}

std::optional<TokenAccount> RPCClient::get_token_account(
    const TokenAddress& address
) const {
    auto account = get_account(WalletAddress::from_base58(address.to_base58()));
    if (account && !account->data.empty()) {
        return TokenAccount::parse(account->data);
    }
    return std::nullopt;
}

std::vector<TokenAccount> RPCClient::get_token_accounts_by_owner(
    const WalletAddress& owner
) const {
    // Would call getTokenAccountsByOwner RPC
    return {};
}

std::optional<MintAccount> RPCClient::get_mint_info(
    const WalletAddress& mint
) const {
    auto account = get_account(mint);
    if (account && !account->data.empty()) {
        return MintAccount::parse(account->data);
    }
    return std::nullopt;
}

uint64_t RPCClient::get_balance(const WalletAddress& address) const {
    auto account = get_account(address);
    return account ? account->lamports : 0;
}

std::pair<Hash, uint64_t> RPCClient::get_recent_blockhash() const {
    // Would call getRecentBlockhash RPC
    Hash blockhash = {};
    return {blockhash, 0};
}

uint64_t RPCClient::get_minimum_balance_for_rent_exemption(
    size_t data_size
) const {
    // Simplified - would calculate based on current rent schedule
    return data_size * 1000;
}

std::optional<Hash> RPCClient::send_transaction(
    const Transaction& transaction
) const {
    // Would serialize and send transaction
    return std::nullopt;
}

std::pair<bool, std::string> RPCClient::simulate_transaction(
    const Transaction& transaction
) const {
    // Would simulate transaction
    return {true, ""};
}

std::optional<RPCClient::SignatureStatus> RPCClient::get_signature_status(
    const Hash& signature
) const {
    return std::nullopt;
}

StakeAccount RPCClient::get_stake_info(const WalletAddress& address) const {
    auto account = get_account(address);
    if (account && !account->data.empty()) {
        return StakeAccount::parse(account->data);
    }
    return {};
}

std::vector<double> RPCClient::get_staking_rewards(
    const WalletAddress& address,
    uint64_t from_epoch,
    uint64_t to_epoch
) const {
    return {};
}

std::variant<std::string, nlohmann::json> RPCClient::call(
    const std::string& method,
    const nlohmann::json& params
) const {
    // Would make HTTP request to RPC endpoint
    return "";
}

// ============ TransactionBuilder ============

TransactionBuilder::TransactionBuilder() {}

TransactionBuilder& TransactionBuilder::add_instruction(
    const TransactionInstruction& instruction
) {
    instructions_.push_back(instruction);
    return *this;
}

TransactionBuilder& TransactionBuilder::set_fee_payer(const WalletAddress& fee_payer) {
    fee_payer_ = fee_payer;
    return *this;
}

TransactionBuilder& TransactionBuilder::set_recent_blockhash(const Hash& blockhash) {
    recent_blockhash_ = blockhash;
    return *this;
}

Transaction TransactionBuilder::build() const {
    Transaction tx;
    Message msg;
    
    // Build account keys from instructions
    std::set<WalletAddress> account_set;
    
    for (const auto& instr : instructions_) {
        account_set.insert(instr.program_id);
        for (const auto& key : instr.accounts) {
            account_set.insert(key);
        }
    }
    
    msg.account_keys.assign(account_set.begin(), account_set.end());
    
    // Set header
    msg.header.num_required_signatures = fee_payer_ ? 1 : 0;
    msg.header.num_readonly_unsigned_accounts = msg.account_keys.size() - msg.header.num_required_signatures;
    msg.header.num_readonly_signed_accounts = 0;
    
    // Set recent blockhash
    if (recent_blockhash_) {
        msg.recent_blockhash = *recent_blockhash_;
    }
    
    // Set instructions
    msg.instructions = instructions_;
    
    tx.message = msg;
    
    return tx;
}

Transaction TransactionBuilder::build_and_sign(
    const std::vector<PrivateKey>& signers
) const {
    auto tx = build();
    
    // Sign with all provided signers
    for (const auto& sk : signers) {
        Wallet wallet(sk);
        auto sig = wallet.sign_transaction(tx);
        tx.add_signature(sig);
    }
    
    return tx;
}

// ============ Wallet ============

Wallet::Wallet(const PrivateKey& private_key) : private_key_(private_key) {
    public_key_ = derive_public_key(private_key);
    address_ = WalletAddress::from_pubkey(public_key_);
}

PublicKey Wallet::get_public_key() const {
    return public_key_;
}

WalletAddress Wallet::get_address() const {
    return address_;
}

Signature Wallet::sign_message(const std::vector<uint8_t>& message) const {
    Signature sig = {};
    
    // Simplified Ed25519 signing
    // In production, would use proper Ed25519 signing
    SHA512(reinterpret_cast<const unsigned char*>(private_key_.data()), 64, sig.data());
    
    return sig;
}

Signature Wallet::sign_transaction(const Transaction& transaction) const {
    auto message = transaction.message.serialize();
    return sign_message(message);
}

bool Wallet::verify_signature(
    const Signature& signature,
    const std::vector<uint8_t>& message
) const {
    // Simplified - would verify properly
    return true;
}

PublicKey Wallet::derive_public_key(const PrivateKey& private_key) {
    // Solana uses Ed25519. The public key is `scalar_mult(seed)` derived via
    // SHA-512 of the 32-byte seed -- NOT SHA-256 of the 64-byte expanded key.
    // The real derivation lives in the audited Rust core
    // (solana/rust/src/lib.rs `derive_public_key`, using ed25519-dalek).
    // Fabricating a key with SHA-256 here would be a security bug, so this
    // returns an all-zero sentinel; callers must use the Rust core.
    (void)private_key;
    PublicKey pubkey = {};
    return pubkey;
}

// ============ Utilities ============

namespace utils {
    std::string encode_base58(const uint8_t* data, size_t len) {
        std::string result;
        
        // Count leading zeros
        size_t zeros = 0;
        for (size_t i = 0; i < len; i++) {
            if (data[i] == 0) zeros++;
            else break;
        }
        
        // Convert to base58
        std::vector<unsigned int> digits(len * 138 / 100 + 1);
        size_t digits_len = 1;
        
        for (size_t i = 0; i < len; i++) {
            int carry = data[i];
            for (size_t j = digits_len - 1; j < digits_len; j--) {
                carry += digits[j] << 8;
                digits[j] = carry % 58;
                carry /= 58;
            }
            while (carry > 0) {
                digits[digits_len++] = carry % 58;
                carry /= 58;
            }
        }
        
        // Add leading zeros
        result.append(zeros, '1');
        
        // Convert to string
        for (size_t i = digits_len; i > 0; i--) {
            result += BASE58_ALPHABET[digits[i - 1]];
        }
        
        return result;
    }
    
    std::vector<uint8_t> decode_base58(const std::string& str) {
        std::vector<uint8_t> result;
        
        // Count leading ones
        size_t zeros = 0;
        for (char c : str) {
            if (c == '1') zeros++;
            else break;
        }
        
        // Convert from base58
        std::vector<unsigned int> digits(str.size());
        size_t digits_len = 1;
        
        for (char c : str) {
            int val = -1;
            for (int i = 0; i < 58; i++) {
                if (BASE58_ALPHABET[i] == c) {
                    val = i;
                    break;
                }
            }
            if (val == -1) continue;
            
            for (size_t j = digits_len - 1; j < digits_len; j--) {
                val += digits[j] * 58;
                digits[j] = val & 0xFF;
                val >>= 8;
            }
            while (val > 0) {
                digits[digits_len++] = val & 0xFF;
                val >>= 8;
            }
        }
        
        // Add leading zeros
        result.assign(zeros, 0);
        
        // Convert to bytes
        for (size_t i = 0; i < digits_len; i++) {
            result.push_back(digits[digits_len - 1 - i]);
        }
        
        return result;
    }
    
    std::vector<uint8_t> encode_short_vec(const std::vector<uint8_t>& data) {
        std::vector<uint8_t> result;
        
        if (data.size() < 192) {
            result.push_back(data.size() - 128);
        } else if (data.size() < 16320) {
            uint16_t len = data.size() - 192;
            result.push_back((len >> 8) + 192);
            result.push_back(len & 0xFF);
        } else {
            uint32_t len = data.size() - 16320;
            result.push_back(254);
            result.push_back(len >> 24);
            result.push_back((len >> 16) & 0xFF);
            result.push_back((len >> 8) & 0xFF);
            result.push_back(len & 0xFF);
        }
        
        result.insert(result.end(), data.begin(), data.end());
        
        return result;
    }
    
    std::vector<uint8_t> decode_short_vec(const std::vector<uint8_t>& data) {
        if (data.empty()) return {};
        
        std::vector<uint8_t> result;
        size_t offset = 0;
        
        uint8_t first = data[0];
        
        if (first < 192) {
            uint8_t len = first + 128;
            offset = 1;
            if (data.size() > len) {
                result.assign(data.begin() + 1, data.begin() + 1 + len);
            }
        } else if (first < 240) {
            uint16_t len = ((first - 192) << 8) + data[1] + 192;
            offset = 2;
            if (data.size() > len + 2) {
                result.assign(data.begin() + 2, data.begin() + 2 + len);
            }
        } else {
            uint32_t len = (data[1] << 24) + (data[2] << 16) + (data[3] << 8) + data[4] + 16320;
            offset = 5;
            if (data.size() > len + 5) {
                result.assign(data.begin() + 5, data.begin() + 5 + len);
            }
        }
        
        return result;
    }
    
    std::string get_cluster_url(const std::string& cluster) {
        if (cluster == "mainnet") {
            return "https://api.mainnet-beta.solana.com";
        } else if (cluster == "testnet") {
            return "https://api.testnet.solana.com";
        } else if (cluster == "devnet") {
            return "https://api.devnet.solana.com";
        }
        return "";
    }
}

// ============ SPL Token Program IDs ============

const WalletAddress SPLTokenClient::TOKEN_PROGRAM_ID = 
    WalletAddress::from_base58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA");
const WalletAddress SPLTokenClient::ASSOCIATED_TOKEN_PROGRAM_ID = 
    WalletAddress::from_base58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL");

// ============ Staking Program IDs ============

const WalletAddress StakingClient::STAKE_PROGRAM_ID = 
    WalletAddress::from_base58("Stake11111111111111111111111111111111111111");
const WalletAddress StakingClient::VOTE_PROGRAM_ID = 
    WalletAddress::from_base58("Vote111111111111111111111111111111111111111");

// ============ SPLTokenClient ============

SPLTokenClient::SPLTokenClient(RPCClient& rpc_client) : rpc_client_(rpc_client) {}

uint64_t SPLTokenClient::get_balance(
    const WalletAddress& owner,
    const WalletAddress& mint
) const {
    auto token_addr = TokenAddress::create(mint, owner);
    auto account = rpc_client_.get_token_account(token_addr);
    return account ? account->amount : 0;
}

std::vector<TokenAccount> SPLTokenClient::get_all_tokens(
    const WalletAddress& owner
) const {
    return rpc_client_.get_token_accounts_by_owner(owner);
}

Transaction SPLTokenClient::create_token(
    const Wallet& authority,
    uint8_t decimals,
    std::optional<WalletAddress> freeze_authority
) {
    auto mint = WalletAddress{}; // Would generate new address
    
    // Use system program for creating mint
    return TransactionBuilder()
        .set_fee_payer(authority.get_address())
        .add_instruction({
            TOKEN_PROGRAM_ID,
            {},
            {0x09} // Create new mint
        })
        .build();
}

Transaction SPLTokenClient::mint_to(
    const Wallet& authority,
    const WalletAddress& mint,
    const WalletAddress& destination,
    uint64_t amount
) {
    TransactionBuilder builder;
    builder.set_fee_payer(authority.get_address());
    
    // MintTo instruction
    TransactionInstruction instr;
    instr.program_id = TOKEN_PROGRAM_ID;
    instr.accounts = {mint, destination, authority.get_address()};
    instr.data = {0x09}; // MintTo instruction
    
    builder.add_instruction(instr);
    
    return builder.build();
}

Transaction SPLTokenClient::transfer(
    const Wallet& authority,
    const WalletAddress& source,
    const WalletAddress& destination,
    uint64_t amount
) {
    TransactionBuilder builder;
    builder.set_fee_payer(authority.get_address());
    
    // Transfer instruction
    TransactionInstruction instr;
    instr.program_id = TOKEN_PROGRAM_ID;
    instr.accounts = {source, destination, authority.get_address()};
    
    // Encode amount as data
    instr.data = {0x03}; // Transfer instruction
    for (int i = 7; i >= 0; i--) {
        instr.data.push_back((amount >> (i * 8)) & 0xFF);
    }
    
    builder.add_instruction(instr);
    
    return builder.build();
}

Transaction SPLTokenClient::burn(
    const Wallet& authority,
    const WalletAddress& mint,
    uint64_t amount
) {
    TransactionBuilder builder;
    builder.set_fee_payer(authority.get_address());
    
    // Burn instruction
    TransactionInstruction instr;
    instr.program_id = TOKEN_PROGRAM_ID;
    instr.accounts = {mint, authority.get_address()};
    instr.data = {0x0b}; // Burn instruction
    
    builder.add_instruction(instr);
    
    return builder.build();
}

// ============ StakingClient ============

StakingClient::StakingClient(RPCClient& rpc_client) : rpc_client_(rpc_client) {}

std::vector<StakingClient::ValidatorInfo> StakingClient::get_validators() const {
    // Would call getVoteAccounts RPC
    return {};
}

uint64_t StakingClient::get_stake_balance(const WalletAddress& stake_account) const {
    auto account = rpc_client_.get_stake_info(stake_account);
    return account.staked_lamports;
}

std::vector<StakingClient::DelegationInfo> StakingClient::get_delegations(
    const WalletAddress& stake_account
) const {
    return {};
}

Transaction StakingClient::create_stake_account(
    const Wallet& from,
    uint64_t lamports
) {
    return TransactionBuilder()
        .set_fee_payer(from.get_address())
        .build();
}

Transaction StakingClient::delegate_stake(
    const Wallet& authority,
    const WalletAddress& stake_account,
    const WalletAddress& vote_account
) {
    return TransactionBuilder()
        .set_fee_payer(authority.get_address())
        .build();
}

Transaction StakingClient::deactivate_stake(
    const Wallet& authority,
    const WalletAddress& stake_account
) {
    return TransactionBuilder()
        .set_fee_payer(authority.get_address())
        .build();
}

Transaction StakingClient::withdraw_stake(
    const Wallet& authority,
    const WalletAddress& stake_account,
    uint64_t lamports
) {
    return TransactionBuilder()
        .set_fee_payer(authority.get_address())
        .build();
}

double StakingClient::get_rewards(const WalletAddress& stake_account) const {
    return 0.0;
}

// ============ NFTClient ============

NFTClient::NFTClient(RPCClient& rpc_client) : rpc_client_(rpc_client) {}

std::optional<NFTMetadata> NFTClient::get_metadata(const WalletAddress& mint) const {
    auto metadata_addr = NFTMetadata::get_metadata_address(mint);
    auto account = rpc_client_.get_account(metadata_addr);
    
    if (account && !account->data.empty()) {
        // Parse metadata
        // Simplified - would fully parse Metaplex metadata
        NFTMetadata metadata;
        return metadata;
    }
    
    return std::nullopt;
}

std::vector<WalletAddress> NFTClient::get_nfts_by_owner(
    const WalletAddress& owner
) const {
    auto accounts = rpc_client_.get_token_accounts_by_owner(owner);
    
    std::vector<WalletAddress> nfts;
    for (const auto& acc : accounts) {
        // Check if NFT (decimals = 0)
        if (acc.decimals == 0 && acc.amount > 0) {
            nfts.push_back(acc.mint);
        }
    }
    
    return nfts;
}

std::optional<NFTMetadata> NFTClient::get_nft_by_mint(
    const WalletAddress& mint
) const {
    return get_metadata(mint);
}

std::vector<WalletAddress> NFTClient::get_collection_nfts(
    const WalletAddress& collection
) const {
    return {};
}

// Static
WalletAddress NFTMetadata::get_metadata_address(const WalletAddress& mint) {
    // Metaplex metadata PDA derivation must use the real find_program_address
    // (solana/rust/src/lib.rs): sha256(["metadata" || mint] || METAPLEX_PROGRAM
    // || bump) with on-curve rejection. A bare SHA-256 of the mint is NOT a
    // valid PDA. Return an all-zero sentinel; callers must use the Rust core.
    (void)mint;
    WalletAddress addr;
    return addr;
}

NFTMetadata NFTMetadata::parse_from_json(const std::string& json) {
    NFTMetadata metadata;
    // Would parse JSON
    return metadata;
}

} // namespace solana
} // namespace tiger
