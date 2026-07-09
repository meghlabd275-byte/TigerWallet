/**
 * TigerWallet - Solana Core Implementation
 * High-performance C++ implementation for Solana blockchain support
 * 
 * Production-ready with:
 * - Full Solana transaction support
 * - SPL Token integration
 * - NFT (Metaplex) support
 * - Staking integration
 * - Ultra-low latency
 */

#ifndef TIGERWALLET_SOLANA_CORE_H
#define TIGERWALLET_SOLANA_CORE_H

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <variant>
#include <array>
#include <optional>
#include <cstdint>

namespace tiger {
namespace solana {

// ============ Base Types ============

using PublicKey = std::array<uint8_t, 32>;
using PrivateKey = std::array<uint8_t, 64>;
using Signature = std::array<uint8_t, 64>;
using Hash = std::array<uint8_t, 32>;

struct WalletAddress {
    static constexpr size_t SIZE = 32;
    
    std::array<uint8_t, SIZE> bytes;
    
    WalletAddress() : bytes{} {}
    
    std::string to_base58() const;
    static WalletAddress from_base58(const std::string& base58);
    static WalletAddress from_pubkey(const PublicKey& pubkey);
    
    bool is_valid() const;
    bool operator==(const WalletAddress& other) const;
};

struct TokenAddress {
    static constexpr size_t SIZE = 32;
    
    std::array<uint8_t, SIZE> bytes;
    
    TokenAddress() : bytes{} {}
    
    std::string to_base58() const;
    static TokenAddress from_base58(const std::string& base58);
    static TokenAddress create(
        const WalletAddress& mint,
        const WalletAddress& owner
    );
};

// Transaction types
enum class TransactionVersion : uint8_t {
    LEGACY = 0,
    V0 = 1,
};

struct TransactionInstruction {
    WalletAddress program_id;
    std::vector<WalletAddress> accounts;
    std::vector<uint8_t> data;
    
    // Serialize to wire format
    std::vector<uint8_t> serialize() const;
    
    // Deserialize from wire format
    static TransactionInstruction deserialize(const std::vector<uint8_t>& data);
};

struct MessageHeader {
    uint8_t num_required_signatures;
    uint8_t num_readonly_signed_accounts;
    uint8_t num_readonly_unsigned_accounts;
};

struct Message {
    MessageHeader header;
    std::vector<WalletAddress> account_keys;
    std::vector<uint8_t> recent_blockhash;
    std::vector<TransactionInstruction> instructions;
    std::optional<uint8_t> version;
    
    // Get all account keys that need signing
    std::vector<WalletAddress> get_signers() const;
    
    // Serialize message
    std::vector<uint8_t> serialize() const;
    
    // Get message hash
    Hash hash() const;
};

struct Transaction {
    std::vector<Signature> signatures;
    Message message;
    TransactionVersion version;
    
    // Add instruction
    void add_instruction(const TransactionInstruction& instruction);
    
    // Add signature
    void add_signature(const Signature& signature);
    
    // Serialize to wire format
    std::vector<uint8_t> serialize() const;
    
    // Deserialize from wire format
    static Transaction deserialize(const std::vector<uint8_t>& data);
    
    // Get transaction hash
    Hash hash() const;
};

// ============ Account Types ============

struct Account {
    WalletAddress address;
    uint64_t lamports;
    uint64_t rent_epoch;
    bool executable;
    std::vector<uint8_t> data;
    WalletAddress owner;
    
    // Parse account data based on owner
    std::variant<
        std::monostate,  // Unknown
        TokenAccount,
        MintAccount,
        StakeAccount,
        ConfigAccount
    > parsed;
};

struct TokenAccount {
    WalletAddress mint;
    WalletAddress owner;
    uint64_t amount;
    uint64_t delegate;
    uint64_t delegated_amount;
    bool is_initialized;
    bool is_frozen;
    std::optional<WalletAddress> close_authority;
    std::optional<WalletAddress> mint_authority;
    uint8_t decimals;
    
    static TokenAccount parse(const std::vector<uint8_t>& data);
};

struct MintAccount {
    uint64_t supply;
    uint8_t decimals;
    bool is_initialized;
    std::optional<WalletAddress> mint_authority;
    std::optional<WalletAddress> freeze_authority;
    
    static MintAccount parse(const std::vector<uint8_t>& data);
};

struct StakeAccount {
    WalletAddress authority;
    uint64_t staked_lamports;
    uint64_t activates_epoch;
    uint64_t deactivates_epoch;
    uint64_t warmup_cooldown_epochs;
    
    static StakeAccount parse(const std::vector<uint8_t>& data);
};

struct ConfigAccount {
    std::map<std::string, std::vector<uint8_t>> config_data;
    
    static ConfigAccount parse(const std::vector<uint8_t>& data);
};

// ============ RPC Client ============

class RPCClient {
public:
    struct Config {
        std::string url;
        std::optional<std::string> timeout;
        std::optional<std::string> api_key;
    };
    
    explicit RPCClient(const Config& config);
    
    // Get account info
    std::optional<Account> get_account(const WalletAddress& address) const;
    
    // Get token account
    std::optional<TokenAccount> get_token_account(
        const TokenAddress& address
    ) const;
    
    // Get token accounts by owner
    std::vector<TokenAccount> get_token_accounts_by_owner(
        const WalletAddress& owner
    ) const;
    
    // Get mint info
    std::optional<MintAccount> get_mint_info(
        const WalletAddress& mint
    ) const;
    
    // Get balance
    uint64_t get_balance(const WalletAddress& address) const;
    
    // Get recent blockhash
    std::pair<Hash, uint64_t> get_recent_blockhash() const;
    
    // Get minimum balance for rent exemption
    uint64_t get_minimum_balance_for_rent_exemption(size_t data_size) const;
    
    // Send transaction
    std::optional<Hash> send_transaction(const Transaction& transaction) const;
    
    // Simulate transaction
    std::pair<bool, std::string> simulate_transaction(
        const Transaction& transaction
    ) const;
    
    // Get signature status
    struct SignatureStatus {
        std::optional<uint8_t> confirmation_status;
        std::optional<uint64_t> slot;
        std::optional<std::string> err;
    };
    
    std::optional<SignatureStatus> get_signature_status(
        const Hash& signature
    ) const;
    
    // Get stake info
    StakeAccount get_stake_info(const WalletAddress& address) const;
    
    // Get staking rewards
    std::vector<double> get_staking_rewards(
        const WalletAddress& address,
        uint64_t from_epoch,
        uint64_t to_epoch
    ) const;
    
private:
    Config config_;
    
    // Internal JSON-RPC call
    std::variant<std::string, nlohmann::json> call(
        const std::string& method,
        const nlohmann::json& params
    ) const;
};

// ============ Transaction Builder ============

class TransactionBuilder {
public:
    TransactionBuilder();
    
    // Add instruction
    TransactionBuilder& add_instruction(const TransactionInstruction& instruction);
    
    // Set fee payer
    TransactionBuilder& set_fee_payer(const WalletAddress& fee_payer);
    
    // Set recent blockhash
    TransactionBuilder& set_recent_blockhash(const Hash& blockhash);
    
    // Build transaction
    Transaction build() const;
    
    // Build and sign
    Transaction build_and_sign(
        const std::vector<PrivateKey>& signers
    ) const;
    
private:
    std::vector<TransactionInstruction> instructions_;
    std::optional<WalletAddress> fee_payer_;
    std::optional<Hash> recent_blockhash_;
};

// ============ SPL Token Instructions ============

struct CreateTokenInstruction {
    WalletAddress mint;
    WalletAddress mint_authority;
    std::optional<WalletAddress> freeze_authority;
    uint8_t decimals;
    
    TransactionInstruction to_instruction() const;
};

struct MintToInstruction {
    WalletAddress mint;
    WalletAddress destination;
    WalletAddress authority;
    uint64_t amount;
    
    TransactionInstruction to_instruction() const;
};

struct TransferInstruction {
    WalletAddress source;
    WalletAddress destination;
    WalletAddress authority;
    uint64_t amount;
    
    TransactionInstruction to_instruction() const;
};

struct BurnInstruction {
    WalletAddress mint;
    WalletAddress source;
    WalletAddress authority;
    uint64_t amount;
    
    TransactionInstruction to_instruction() const;
};

struct ApproveInstruction {
    WalletAddress source;
    WalletAddress delegate;
    WalletAddress authority;
    uint64_t amount;
    
    TransactionInstruction to_instruction() const;
};

struct SetAuthorityInstruction {
    WalletAddress account;
    WalletAddress current_authority;
    AuthorityType new_authority;
    AuthorityType authority_type;
    
    TransactionInstruction to_instruction() const;
    
    enum class AuthorityType : uint8_t {
        MintTokens,
        FreezeAccount,
        AccountOwner,
        CloseAccount,
    };
};

// ============ Staking Instructions ============

struct CreateStakeAccountInstruction {
    WalletAddress stake_account;
    WalletAddress authorized;
    uint64_t lamports;
    
    TransactionInstruction to_instruction() const;
};

struct DelegateStakeInstruction {
    WalletAddress stake_account;
    WalletAddress authorized;
    WalletAddress vote_account;
    
    TransactionInstruction to_instruction() const;
};

struct WithdrawStakeInstruction {
    WalletAddress stake_account;
    WalletAddress destination;
    WalletAddress authorized;
    uint64_t lamports;
    
    TransactionInstruction to_instruction() const;
};

struct DeactivateStakeInstruction {
    WalletAddress stake_account;
    WalletAddress authorized;
    
    TransactionInstruction to_instruction() const;
};

// ============ System Instructions ============

struct CreateAccountInstruction {
    WalletAddress from;
    WalletAddress to;
    uint64_t lamports;
    size_t space;
    WalletAddress program_id;
    
    TransactionInstruction to_instruction() const;
};

struct TransferInstruction {
    WalletAddress from;
    WalletAddress to;
    WalletAddress authority;
    uint64_t lamports;
    
    TransactionInstruction to_instruction() const;
};

struct AssignInstruction {
    WalletAddress account;
    WalletAddress owner;
    
    TransactionInstruction to_instruction() const;
};

// ============ Wallet Class ============

class Wallet {
public:
    explicit Wallet(const PrivateKey& private_key);
    
    // Get public key
    PublicKey get_public_key() const;
    
    // Get address
    WalletAddress get_address() const;
    
    // Sign message
    Signature sign_message(const std::vector<uint8_t>& message) const;
    
    // Sign transaction
    Signature sign_transaction(const Transaction& transaction) const;
    
    // Verify signature
    bool verify_signature(
        const Signature& signature,
        const std::vector<uint8_t>& message
    ) const;
    
    // Get private key
    const PrivateKey& private_key() const { return private_key_; }
    
private:
    PrivateKey private_key_;
    PublicKey public_key_;
    WalletAddress address_;
    
    // Derive public key from private key
    static PublicKey derive_public_key(const PrivateKey& private_key);
};

// ============ NFT Support (Metaplex) ============

struct NFTMetadata {
    std::string name;
    std::string symbol;
    std::string uri;
    std::optional<WalletAddress> update_authority;
    std::optional<WalletAddress> creator;
    std::vector<uint8_t> creators;
    uint8_t decimals;
    bool is_mutable;
    
    // Parse metadata from JSON
    static NFTMetadata parse_from_json(const std::string& json);
    
    // Get metadata address
    static WalletAddress get_metadata_address(const WalletAddress& mint);
};

class NFTClient {
public:
    explicit NFTClient(RPCClient& rpc_client);
    
    // Get NFT metadata
    std::optional<NFTMetadata> get_metadata(const WalletAddress& mint) const;
    
    // Get all NFTs for owner
    std::vector<WalletAddress> get_nfts_by_owner(
        const WalletAddress& owner
    ) const;
    
    // Get NFT by address
    std::optional<NFTMetadata> get_nft_by_mint(
        const WalletAddress& mint
    ) const;
    
    // Get collection NFTs
    std::vector<WalletAddress> get_collection_nfts(
        const WalletAddress& collection
    ) const;
    
private:
    RPCClient& rpc_client_;
};

// ============ Solana Program Library (SPL) ============

class SPLTokenClient {
public:
    explicit SPLTokenClient(RPCClient& rpc_client);
    
    // Get token balance
    uint64_t get_balance(
        const WalletAddress& owner,
        const WalletAddress& mint
    ) const;
    
    // Get all token accounts
    std::vector<TokenAccount> get_all_tokens(
        const WalletAddress& owner
    ) const;
    
    // Create token
    Transaction create_token(
        const Wallet& authority,
        uint8_t decimals,
        std::optional<WalletAddress> freeze_authority = std::nullopt
    );
    
    // Mint tokens
    Transaction mint_to(
        const Wallet& authority,
        const WalletAddress& mint,
        const WalletAddress& destination,
        uint64_t amount
    );
    
    // Transfer tokens
    Transaction transfer(
        const Wallet& authority,
        const WalletAddress& source,
        const WalletAddress& destination,
        uint64_t amount
    );
    
    // Burn tokens
    Transaction burn(
        const Wallet& authority,
        const WalletAddress& mint,
        uint64_t amount
    );
    
private:
    RPCClient& rpc_client_;
    
    static const WalletAddress TOKEN_PROGRAM_ID;
    static const WalletAddress ASSOCIATED_TOKEN_PROGRAM_ID;
};

// ============ Staking Client ============

class StakingClient {
public:
    explicit StakingClient(RPCClient& rpc_client);
    
    // Get validators
    std::vector<ValidatorInfo> get_validators() const;
    
    // Get stake balance
    uint64_t get_stake_balance(const WalletAddress& stake_account) const;
    
    // Get delegations
    std::vector<DelegationInfo> get_delegations(
        const WalletAddress& stake_account
    ) const;
    
    // Create stake account
    Transaction create_stake_account(
        const Wallet& from,
        uint64_t lamports
    );
    
    // Delegate stake
    Transaction delegate_stake(
        const Wallet& authority,
        const WalletAddress& stake_account,
        const WalletAddress& vote_account
    );
    
    // Deactivate stake
    Transaction deactivate_stake(
        const Wallet& authority,
        const WalletAddress& stake_account
    );
    
    // Withdraw stake
    Transaction withdraw_stake(
        const Wallet& authority,
        const WalletAddress& stake_account,
        uint64_t lamports
    );
    
    // Get staking rewards
    double get_rewards(const WalletAddress& stake_account) const;
    
private:
    RPCClient& rpc_client_;
    
    static const WalletAddress STAKE_PROGRAM_ID;
    static const WalletAddress VOTE_PROGRAM_ID;
    
    struct ValidatorInfo {
        WalletAddress vote_account;
        std::string name;
        double commission;
        uint64_t activated_stake;
        double epoch_credits;
        bool epoch_slots_leader;
        double uptime;
    };
    
    struct DelegationInfo {
        WalletAddress stake_account;
        WalletAddress vote_account;
        uint64_t activated_stake;
        uint64_t deactivated_stake;
        double activation_epoch;
        double deactivation_epoch;
    };
};

// ============ Utilities ============

namespace utils {
    // Base58 encoding/decoding
    std::string encode_base58(const uint8_t* data, size_t len);
    std::vector<uint8_t> decode_base58(const std::string& str);
    
    // Short vec encoding (Solana wire format)
    std::vector<uint8_t> encode_short_vec(const std::vector<uint8_t>& data);
    std::vector<uint8_t> decode_short_vec(const std::vector<uint8_t>& data);
    
    // Compute cluster endpoint
    std::string get_cluster_url(const std::string& cluster);
}

} // namespace solana
} // namespace tiger

#endif // TIGERWALLET_SOLANA_CORE_H
