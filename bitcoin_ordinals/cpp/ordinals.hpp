/**
 * TigerWallet Bitcoin Ordinals Implementation
 * C++ core for ordinal inscription and BRC-20 token support
 */

#ifndef TIGERWALLET_ORDINALS_HPP
#define TIGERWALLET_ORDINALS_HPP

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>
#include <memory>

namespace tigerwallet {
namespace ordinals {

// ============================================================================
// Data Types
// ============================================================================

/// Ordinal number (satoshi position in Bitcoin)
using Ordinal = uint64_t;

/// Satributes - ordinal rarity
enum class Rarity {
    Common,     // 0-1,000
    Uncommon,   // 1,001-10,000
    Rare,       // 10,001-100,000
    Epic,       // 100,001-1,000,000
    Legendary,  // 1,000,001-2,100,000
    Mythic      // Every 10th ordinal at 6666 (after 2,100,000)
};

/// Inscription content type
enum class ContentType {
    Text,
    Image,
    Video,
    Audio,
    Application,
    JSON,
    HTML,
    CSS,
    JavaScript,
    Unknown
};

/// Ordinal inscription
struct Inscription {
    std::string id;              // Unique inscription ID
    Ordinal ordinal;             // Starting ordinal number
    std::string content;         // Content data (hex encoded)
    ContentType content_type;    // MIME type
    std::string mimetype;        // Full MIME type string
    uint32_t content_length;     // Size in bytes
    std::string parent_id;      // Parent inscription (for collections)
    std::map<std::string, std::string> metadata;
    std::string owner;           // Owner address
    uint32_t inscription_number; // Monotonic inscription number
    bool revealed;              // Whether content is revealed
    uint64_t timestamp;         // Creation timestamp
};

/// BRC-20 token
struct BRC20Token {
    std::string tick;            // Token ticker (4 chars)
    std::string name;          // Full name
    uint64_t max_supply;        // Maximum supply
    uint64_t minted;            // Amount minted
    uint32_t decimals;          // Decimal places
    std::string deployer;       // Deployer address
    bool transferable;         // Whether transferable
    uint64_t timestamp;         // Deployment timestamp
};

/// BRC-20 balance
struct BRC20Balance {
    std::string tick;           // Token ticker
    std::string address;       // Owner address
    uint64_t balance;          // Available balance
    uint64_t balance_bigint;   // BigInt representation
    uint64_t transferable;     // Transferable amount
};

/// BRC-20 transfer
struct BRC20Transfer {
    std::string id;            // Transfer ID
    std::string tick;         // Token ticker
    std::string from;          // Sender
    std::string to;           // Receiver
    uint64_t amount;           // Amount
    std::string tx_hash;      // Bitcoin transaction
    uint64_t timestamp;       // Transfer timestamp
    bool confirmed;           // Whether confirmed
};

/// Inscription UTXO
struct InscriptionUTXO {
    std::string tx_hash;       // Transaction hash
    uint32_t index;            // Output index
    std::string owner;         // Owner address
    std::vector<std::string> inscription_ids;
    uint64_t satoshi_value;    // Satoshi value
};

// ============================================================================
// Ordinal Engine
// ============================================================================

class OrdinalEngine {
public:
    OrdinalEngine();
    ~OrdinalEngine();

    /// Calculate rarity of an ordinal
    static Rarity get_rarity(Ordinal ordinal);

    /// Get rarity name
    static std::string get_rarity_name(Rarity rarity);

    /// Create inscription
    std::optional<Inscription> create_inscription(
        const std::string& content,
        ContentType content_type,
        const std::string& owner,
        std::optional<std::string> parent_id = std::nullopt
    );

    /// Parse inscription from content
    static ContentType parse_content_type(const std::string& mimetype);

    /// Get inscription metadata
    std::map<std::string, std::string> get_inscription_metadata(
        const Inscription& inscription
    ) const;

    /// Calculate inscription fees
    uint64_t calculate_inscription_fee(
        uint32_t content_size,
        uint64_t fee_rate
    ) const;

    /// Generate inscription reveal tx
    std::vector<uint8_t> generate_reveal_tx(
        const Inscription& inscription,
        const std::string& reveal_address,
        uint64_t satoshi_value
    ) const;

    /// Parse ordinal from UTXO
    static std::optional<Ordinal> parse_ordinal_from_utxo(
        const std::string& tx_hash,
        uint32_t output_index,
        uint64_t satoshi
    );

    /// Check if satoshi is first in block
    static bool is_first_satoshi_of_block(Ordinal ordinal);

    /// Get block for ordinal
    static uint64_t get_block_height(Ordinal ordinal);

private:
    uint64_t next_inscription_number_;
    std::map<std::string, Inscription> inscriptions_;
};

// ============================================================================
// BRC-20 Engine
// ============================================================================

class BRC20Engine {
public:
    BRC20Engine();
    ~BRC20Engine();

    /// Deploy BRC-20 token
    std::optional<BRC20Token> deploy_token(
        const std::string& tick,
        const std::string& name,
        uint64_t max_supply,
        uint32_t decimals,
        const std::string& deployer,
        bool transferable
    );

    /// Mint BRC-20 tokens
    std::optional<BRC20Transfer> mint(
        const std::string& tick,
        const std::string& to,
        uint64_t amount,
        const std::string& tx_hash
    );

    /// Transfer BRC-20 tokens
    std::optional<BRC20Transfer> transfer(
        const std::string& tick,
        const std::string& from,
        const std::string& to,
        uint64_t amount,
        const std::string& tx_hash
    );

    /// Get token info
    std::optional<BRC20Token> get_token(const std::string& tick) const;

    /// Get balance for address
    std::optional<BRC20Balance> get_balance(
        const std::string& tick,
        const std::string& address
    ) const;

    /// Get all balances for address
    std::vector<BRC20Balance> get_all_balances(const std::string& address) const;

    /// Parse BRC-20 operation from inscription content
    static std::optional<std::map<std::string, std::string>> parse_operation(
        const std::string& content
    );

    /// Validate transfer inscription
    bool validate_transfer(
        const std::string& tick,
        const std::string& from,
        uint64_t amount
    ) const;

    /// Get holders for token
    std::vector<BRC20Balance> get_holders(const std::string& tick) const;

    /// Calculate transferable balance
    uint64_t calculate_transferable(
        const BRC20Balance& balance,
        uint64_t current_block
    ) const;

private:
    std::map<std::string, BRC20Token> tokens_;
    std::map<std::string, std::vector<BRC20Balance>> balances_;
    std::vector<BRC20Transfer> transfers_;

    bool token_exists(const std::string& tick) const;
    bool has_sufficient_balance(
        const std::string& tick,
        const std::string& address,
        uint64_t amount
    ) const;
};

// ============================================================================
// Bitcoin UTXO Parser
// ============================================================================

class UTXOParser {
public:
    UTXOParser();
    ~UTXOParser();

    /// Parse transaction hex to get outputs
    std::vector<InscriptionUTXO> parse_transaction(
        const std::string& tx_hex,
        const std::string& block_hash
    );

    /// Find ordinal outputs in transaction
    std::vector<Ordinal> find_ordinals(
        const std::string& tx_hash,
        uint64_t starting_satoshi
    );

    /// Check if output contains inscription
    bool has_inscription(const std::vector<uint8_t>& script) const;

    /// Extract inscription from output script
    std::optional<std::string> extract_inscription(
        const std::vector<uint8_t>& script
    ) const;

    /// Get satoshi ranges for output
    std::vector<std::pair<Ordinal, Ordinal>> get_satoshi_ranges(
        uint64_t output_value,
        Ordinal start_ordinal
    ) const;

private:
    uint64_t chain_tip_;
};

// ============================================================================
// Wallet Integration
// ============================================================================

class OrdinalWallet {
public:
    OrdinalWallet(
        const std::string& mnemonic,
        const std::string& password
    );
    ~OrdinalWallet();

    /// Generate address for ordinal
    std::string get_ordinal_address() const;

    /// Get private key for ordinal
    std::string get_ordinal_private_key() const;

    /// Sign message with ordinal key
    std::vector<uint8_t> sign_message(const std::string& message) const;

    /// Verify signature
    bool verify_signature(
        const std::string& message,
        const std::vector<uint8_t>& signature
    ) const;

    /// Derive ordinal-specific address
    std::string derive_ordinal_address(uint32_t index) const;

    /// Get all ordinal UTXOs
    std::vector<InscriptionUTXO> get_ordinal_utxos() const;

    /// Create inscription transaction
    std::string create_inscription_tx(
        const Inscription& inscription,
        uint64_t fee_rate
    );

    /// Broadcast transaction
    std::string broadcast_transaction(const std::vector<uint8_t>& tx_hex) const;

private:
    std::string master_seed_;
    std::string ordinal_key_;
    std::string ordinal_address_;
};

// ============================================================================
// Helper Functions
// ============================================================================

/// Convert decimal to hex
std::string to_hex(const std::vector<uint8_t>& data);

/// Convert hex to decimal
std::vector<uint8_t> from_hex(const std::string& hex);

/// Calculate Bitcoin address from public key
std::string public_key_to_address(
    const std::vector<uint8_t>& pubkey,
    bool is_mainnet
);

/// Create Pay-to-Ordinal (P2O) script
std::vector<uint8_t> create_p2o_script(
    const std::string& ordinal_address,
    const std::string& inscription_id
);

/// Create inscription reveal script
std::vector<uint8_t> create_inscription_script(
    const std::string& content,
    ContentType content_type,
    const std::string& reveal_address
);

} // namespace ordinals
} // namespace tigerwallet

#endif // TIGERWALLET_ORDINALS_HPP
