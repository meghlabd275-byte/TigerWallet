/**
 * TigerWallet - Bitcoin Ordinals & BRC-20 Implementation
 * High-performance C++ implementation for Bitcoin ordinals, inscriptions, and BRC-20 tokens
 * 
 * Production-ready with:
 * - Ordinal theory support
 * - BRC-20 token standard
 * - Inscription tooling
 * - Bitcoin wallet integration
 */

#ifndef TIGERWALLET_BITCOIN_ORDINALS_H
#define TIGERWALLET_BITCOIN_ORDINALS_H

#include <string>
#include <vector>
#include <map>
#include <array>
#include <variant>
#include <optional>
#include <cstdint>

namespace tiger {
namespace bitcoin {

// ============ Base Types ============

using Hash256 = std::array<uint8_t, 32>;
using Hash160 = std::array<uint8_t, 20>;
using PublicKey = std::array<uint8_t, 33>; // Compressed
using Signature = std::array<uint8_t, 64>;

struct BitcoinAddress {
    enum class Type {
        P2PKH,
        P2SH,
        P2WPKH,
        P2WSH,
        P2TR,  // Taproot
    };
    
    Type type;
    std::vector<uint8_t> data;
    
    std::string to_string() const;
    static BitcoinAddress from_string(const std::string& addr);
    static BitcoinAddress from_pubkey(const PublicKey& pubkey, Type type);
    
    Hash160 hash160() const;
};

struct Transaction {
    Hash256 txid;
    uint32_t version;
    std::vector<TxInput> inputs;
    std::vector<TxOutput> outputs;
    uint32_t locktime;
    
    std::vector<uint8_t> serialize() const;
    Hash256 hash() const;
};

struct TxInput {
    Hash256 previous_txid;
    uint32_t previous_vout;
    std::vector<uint8_t> script;
    uint32_t sequence;
};

struct TxOutput {
    int64_t value; // satoshis
    std::vector<uint8_t> script;
    
    // Get output type
    enum class OutputType {
        PUBKEY_HASH,
        SCRIPT_HASH,
        WITNESS_V0_KEYHASH,
        WITNESS_V0_SCRIPTHASH,
        WITNESS_V1_TAPROOT,
        NULL_DATA,
        NON_STANDARD,
    };
    
    OutputType get_type() const;
};

// ============ Ordinal Theory ============

/**
 * Ordinal numbers are a numbering scheme for satoshis.
 * Each satoshi gets a unique ordinal number.
 */
struct Ordinal {
    uint64_t cycle;      // 4 cycles of 6 sub-cycles
    uint8_t subcycle;    // 0-5
    uint8_t epoch;       // Difficulty adjustment period
    uint32_t block;      // Block height
    uint32_t tx_index;   // Transaction index in block
    uint16_t output_index; // Output index in transaction
    uint32_t satoshi;    // Sat index in output
    
    // Parse from satoshi number
    static Ordinal from_satoshi(uint64_t satoshi);
    
    // Get satoshi number
    uint64_t to_satoshi() const;
    
    // Get ordinal string (e.g., "0.0.0.0.0.0")
    std::string to_string() const;
    
    // Get rarity
    enum class Rarity {
        COMMON,     // Any satoshi not listed below
        UNCOMMON,   #1 of 6
        RARE,       #1 of 36
        EPIC,       #1 of 216
        LEGENDARY,  #1 of 1296
        MYTHIC,     #1 of 7776
    };
    
    Rarity get_rarity() const;
    
    // Get name
    std::string get_name() const;
};

/**
 * Inscriptions are satoshi metadata stored in witness data.
 */
struct Inscription {
    std::string content_type;    // MIME type
    std::vector<uint8_t> content; // Binary content
    std::string charset;        // Character set
    std::optional<uint64_t> parent_inscription_id;
    std::vector<std::string> tags;
    std::map<std::string, std::string> metadata;
    
    // Get inscription ID
    std::string get_id() const;
    
    // Serialize to envelope
    std::vector<uint8_t> serialize() const;
    
    // Deserialize from envelope
    static Inscription deserialize(const std::vector<uint8_t>& data);
};

// ============ BRC-20 Token Standard ============

/**
 * BRC-20 is a Bitcoin ordinal token standard inspired by ERC-20.
 * Uses JSON inscriptions for deploy, mint, and transfer.
 */
struct BRC20Token {
    std::string ticker;          // Token symbol (max 4 chars)
    std::string name;            // Full name
    uint64_t max_supply;         // Maximum supply in tokens
    uint64_t mint_limit;         // Maximum mint per transaction
    uint8_t decimals;            // Decimal places
    
    // Get inscription content
    std::string to_json() const;
    
    static BRC20Token from_json(const std::string& json);
};

struct BRC20Deploy {
    BRC20Token token;
    
    std::string to_json() const;
    static BRC20Deploy from_json(const std::string& json);
    
    // Create inscription
    std::vector<uint8_t> create_inscription() const;
};

struct BRC20Mint {
    std::string ticker;          // Token ticker
    uint64_t amount;            // Amount to mint
    
    std::string to_json() const;
    static BRC20Mint from_json(const std::string& json);
    
    // Create inscription
    std::vector<uint8_t> create_inscription() const;
};

struct BRC20Transfer {
    std::string ticker;          // Token ticker
    uint64_t amount;            // Amount to transfer
    
    std::string to_json() const;
    static BRC20Transfer from_json(const std::string& json);
    
    // Create inscription
    std::vector<uint8_t> create_inscription() const;
};

struct BRC20Balance {
    std::string ticker;
    std::string available;       // Available balance
    std::string transferable;    // Transferable balance
    std::string minted;          // Total minted
    
    // Parse from ordinals.app API
    static BRC20Balance from_json(const std::string& json);
};

// ============ Wallet Implementation ============

class BitcoinWallet {
public:
    explicit BitcoinWallet(const std::array<uint8_t, 32>& seed);
    
    // Get address
    BitcoinAddress get_address(BitcoinAddress::Type type = BitcoinAddress::Type::P2TR) const;
    
    // Get private key (WIF)
    std::string get_private_key_wif() const;
    
    // Sign message
    Signature sign_message(const std::vector<uint8_t>& message) const;
    
    // Verify signature
    bool verify_signature(
        const Signature& sig,
        const std::vector<uint8_t>& message
    ) const;
    
    // Get public key
    const PublicKey& get_public_key() const { return public_key_; }
    
private:
    std::array<uint8_t, 32> seed_;
    PublicKey public_key_;
    std::array<uint8_t, 32> private_key_;
    
    void derive_keys();
};

// ============ RPC Client ============

class BitcoinRPC {
public:
    struct Config {
        std::string url;
        std::string user;
        std::string password;
    };
    
    explicit BitcoinRPC(const Config& config);
    
    // Get block by height
    std::optional<std::string> get_block_hash(int64_t height) const;
    
    // Get block
    std::optional<std::string> get_block(const Hash256& block_hash) const;
    
    // Get transaction
    std::optional<Transaction> get_raw_transaction(const Hash256& txid) const;
    
    // Get UTXO
    struct UTXO {
        Hash256 txid;
        uint32_t vout;
        int64_t value;
        std::vector<uint8_t> script;
        bool is_ordinal;
    };
    
    std::vector<UTXO> get_utxos(const BitcoinAddress& address) const;
    
    // Send transaction
    std::optional<Hash256> send_raw_transaction(const Transaction& tx) const;
    
    // Get ordinal UTXOs for address
    std::vector<UTXO> get_ordinal_utxos(const BitcoinAddress& address) const;
    
    // Get inscription UTXOs
    std::vector<Inscription> get_inscriptions(
        const BitcoinAddress& address,
        int64_t limit = 50,
        int64_t offset = 0
    ) const;
    
    // Get BRC-20 balances
    std::map<std::string, BRC20Balance> get_brc20_balances(
        const BitcoinAddress& address
    ) const;
    
    // Get BRC-20 history
    std::vector<std::string> get_brc20_history(
        const BitcoinAddress& address,
        const std::string& ticker
    ) const;
    
private:
    Config config_;
    
    std::variant<std::string, nlohmann::json> call(
        const std::string& method,
        const nlohmann::json& params = {}
    ) const;
};

// ============ Inscription Builder ============

class InscriptionBuilder {
public:
    InscriptionBuilder();
    
    // Set content
    InscriptionBuilder& set_content_type(const std::string& type);
    InscriptionBuilder& set_content(const std::vector<uint8_t>& content);
    InscriptionBuilder& set_metadata(
        const std::string& key,
        const std::string& value
    );
    
    // Add parent
    InscriptionBuilder& set_parent(const std::string& parent_id);
    
    // Build
    Inscription build() const;
    
    // Create reveal transaction
    Transaction create_reveal_transaction(
        const BitcoinWallet& wallet,
        const BitcoinAddress& receiver,
        int64_t fee_rate  // sat/vB
    ) const;
    
private:
    std::string content_type_;
    std::vector<uint8_t> content_;
    std::map<std::string, std::string> metadata_;
    std::optional<uint64_t> parent_id_;
};

// ============ Ordinal Wallet ============

class OrdinalWallet {
public:
    explicit OrdinalWallet(const BitcoinWallet& wallet);
    
    // Get ordinal balance
    uint64_t get_ordinal_balance() const;
    
    // Get inscriptions
    std::vector<Inscription> get_inscriptions(int64_t limit = 50) const;
    
    // Get specific inscription
    std::optional<Inscription> get_inscription(const std::string& id) const;
    
    // Send inscription
    Transaction send_inscription(
        const std::string& id,
        const BitcoinAddress& receiver,
        int64_t fee_rate
    ) const;
    
    // Create inscription
    Transaction create_inscription(
        const InscriptionBuilder& builder,
        const BitcoinAddress& receiver,
        int64_t fee_rate
    ) const;
    
    // Transfer ordinal
    Transaction transfer_ordinal(
        const Ordinal& ordinal,
        const BitcoinAddress& receiver,
        int64_t fee_rate
    ) const;
    
    // Get BRC-20 balances
    std::map<std::string, BRC20Balance> get_brc20_balances() const;
    
    // Deploy BRC-20 token
    Transaction deploy_brc20(
        const BRC20Deploy& deploy,
        int64_t fee_rate
    ) const;
    
    // Mint BRC-20 token
    Transaction mint_brc20(
        const BRC20Mint& mint,
        int64_t fee_rate
    ) const;
    
    // Transfer BRC-20 token
    Transaction transfer_brc20(
        const BRC20Transfer& transfer,
        const BitcoinAddress& receiver,
        int64_t fee_rate
    ) const;
    
private:
    const BitcoinWallet& wallet_;
    BitcoinRPC rpc_;
    
    // Find ordinal UTXO
    std::optional<BitcoinRPC::UTXO> find_ordinal_utxo(
        const Ordinal& ordinal
    ) const;
};

// ============ Utilities ============

namespace utils {
    // Base58 encoding
    std::string encode_base58(const uint8_t* data, size_t len);
    std::vector<uint8_t> decode_base58(const std::string& str);
    
    // Bech32 encoding
    std::string encode_bech32(
        const std::string& hrp,
        const std::vector<uint8_t>& data
    );
    std::pair<std::string, std::vector<uint8_t>> decode_bech32(
        const std::string& addr
    );
    
    // PSBT (Partially Signed Bitcoin Transaction)
    struct PSBT {
        std::vector<uint8_t> global_tx_data;
        std::vector<PSBTInput> inputs;
        std::vector<PSBTOutput> outputs;
        
        static PSBT from_base64(const std::string& base64);
        std::string to_base64() const;
    };
    
    struct PSBTInput {
        std::vector<uint8_t> utxo;
        std::vector<uint8_t> witness_utxo;
        std::vector<Signature> partial_sigs;
        std::optional<BitcoinAddress> sighash_type;
    };
    
    struct PSBTOutput {
        std::vector<uint8_t> redeem_script;
        std::vector<uint8_t> witness_script;
    };
    
    // Fee estimation
    int64_t estimate_smart_fee(
        int64_t target_blocks,
        bool conservative
    );
    
    // Parse transaction vsize
    size_t calculate_vsize(const Transaction& tx);
}

// ============ Rune Protocol (BRC-42) ============

/**
 * Runes are a new Bitcoin ordinal token protocol.
 * More efficient than BRC-20.
 */
struct RuneToken {
    uint64_t id;
    std::string symbol;
    std::string name;
    uint128_t supply;
    uint128_t dividents;
    uint8_t decimals;
    std::optional<std::string> rune_id;
    
    std::string to_json() const;
};

struct RuneMint {
    std::string rune_id;
    uint64_t amount;
    
    std::vector<uint8_t> encode() const;
};

struct RuneTransfer {
    std::string rune_id;
    uint64_t amount;
    std::vector<Hash160> outputs;
    
    std::vector<uint8_t> encode() const;
};

} // namespace bitcoin
} // namespace tiger

#endif // TIGERWALLET_BITCOIN_ORDINALS_H
