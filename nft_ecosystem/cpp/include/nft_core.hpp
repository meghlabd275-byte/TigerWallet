#ifndef NFT_CORE_HPP
#define NFT_CORE_HPP

#include <string>
#include <vector>
#include <unordered_map>
#include <memory>
#include <optional>
#include <cstdint>
#include <chrono>

namespace tigerwallet {
namespace nft {

// Forward declarations
class NFTToken;
class NFTCollection;
class NFTSigner;

// Types
using Address = std::string;
using TokenID = std::string;
using ContractAddress = std::string;
using ChainID = uint64_t;
using Amount = std::string;

// NFT Token metadata
struct NFTMetadata {
    TokenID token_id;
    std::string name;
    std::string description;
    std::string image_url;
    std::optional<std::string> animation_url;
    std::optional<std::string> external_url;
    std::vector<NFTTrait> attributes;
    std::string uri;
    ChainID chain_id;
    std::chrono::system_clock::time_point minted_at;
};

// NFT Trait/Attribute
struct NFTTrait {
    std::string trait_type;
    std::string value;
    std::optional<std::string> display_type;
};

// NFT Transfer info
struct TransferRequest {
    Address from;
    Address to;
    ContractAddress contract;
    TokenID token_id;
    Amount amount;
};

// NFT Mint request
struct MintRequest {
    Address to;
    ContractAddress contract;
    std::string uri;
    std::optional<std::string> name;
    std::optional<std::string> description;
    std::optional<std::string> image_url;
};

// NFT Offer
struct Offer {
    std::string offer_id;
    TokenID token_id;
    ContractAddress contract;
    Address seller;
    std::optional<Address> buyer;
    Amount price;
    Address price_token;
    ChainID chain_id;
    OfferStatus status;
    std::chrono::system_clock::time_point expires_at;
};

enum class OfferStatus : uint8_t {
    Open,
    Cancelled,
    Accepted,
    Completed,
    Expired
};

// Collection
struct Collection {
    ContractAddress contract;
    std::string name;
    std::string symbol;
    std::string description;
    std::string image_url;
    std::optional<std::string> external_url;
    Address creator;
    ChainID chain_id;
    uint64_t total_supply;
    bool is_verified;
};

// Result types
template<typename T>
struct Result {
    bool success;
    std::optional<T> value;
    std::optional<std::string> error;

    static Result<T> Ok(T val) {
        return {true, val, std::nullopt};
    }

    static Result<T> Err(std::string err) {
        return {false, std::nullopt, err};
    }
};

using TxHash = std::string;

// NFT Core interface
class IN FTCore {
public:
    virtual ~IN FTCore() = default;

    // Token operations
    virtual Result<TxHash> transfer(const TransferRequest& req) = 0;
    virtual Result<TxHash> mint(const MintRequest& req) = 0;
    virtual Result<TxHash> burn(const Address& owner, const ContractAddress& contract, const TokenID& token_id) = 0;

    // Query operations
    virtual Result<Address> ownerOf(const ContractAddress& contract, const TokenID& token_id) = 0;
    virtual Result<uint64_t> balanceOf(const ContractAddress& contract, const Address& owner) = 0;
    virtual Result<NFTMetadata> getMetadata(const ContractAddress& contract, const TokenID& token_id) = 0;

    // Approval operations
    virtual Result<TxHash> setApprovalForAll(const Address& owner, const Address& operator_, bool approved) = 0;
    virtual Result<bool> isApprovedForAll(const Address& owner, const Address& operator_) = 0;

    // Batch operations
    virtual std::vector<Result<TxHash>> batchTransfer(const std::vector<TransferRequest>& transfers) = 0;
    virtual std::vector<Result<TokenID>> batchMint(const std::vector<MintRequest>& mints) = 0;
};

// Ultra-low latency NFT signer (C++)
class NFTSignerCXX {
private:
    std::vector<uint8_t> private_key_;
    bool initialized_;

public:
    NFTSignerCXX();
    explicit NFTSignerCXX(const std::string& private_key_hex);
    ~NFTSignerCXX();

    // Initialize with private key
    bool initialize(const std::string& private_key_hex);

    // Sign NFT transfer transaction
    std::optional<std::string> signTransfer(
        const Address& from,
        const Address& to,
        const ContractAddress& contract,
        const TokenID& token_id
    );

    // Sign NFT mint transaction
    std::optional<std::string> signMint(
        const Address& to,
        const ContractAddress& contract,
        const std::string& uri
    );

    // Sign approval transaction
    std::optional<std::string> signApproval(
        const Address& owner,
        const Address& operator_,
        bool approved
    );

    // Get address from private key
    std::optional<Address> getAddress() const;

    // Check if initialized
    bool isInitialized() const { return initialized_; }

private:
    // Internal signing helpers
    std::vector<uint8_t> prepareTransferData(const Address& to, const TokenID& token_id) const;
    std::vector<uint8_t> prepareMintData(const Address& to, const std::string& uri) const;
    std::vector<uint8_t> prepareApprovalData(const Address& operator_, bool approved) const;

    // Keccak256 hash
    std::vector<uint8_t> keccak256(const std::vector<uint8_t>& data) const;

    // Sign with ECDSA
    std::vector<uint8_t> signECDSA(const std::vector<uint8_t>& message) const;
};

// NFT Cache for ultra-fast reads
class NFTCache {
private:
    struct CacheEntry {
        NFTMetadata metadata;
        std::chrono::steady_clock::time_point timestamp;
        uint64_t hits;
    };

    std::unordered_map<std::string, CacheEntry> cache_;
    size_t max_size_;
    uint64_t ttl_seconds_;

public:
    explicit NFTCache(size_t max_size = 10000, uint64_t ttl_seconds = 300);

    // Get cached metadata
    std::optional<NFTMetadata> get(const ContractAddress& contract, const TokenID& token_id);

    // Set cached metadata
    void set(const ContractAddress& contract, const TokenID& token_id, const NFTMetadata& metadata);

    // Invalidate cache entry
    void invalidate(const ContractAddress& contract, const TokenID& token_id);

    // Invalidate all for collection
    void invalidateCollection(const ContractAddress& contract);

    // Clear all cache
    void clear();

    // Get cache statistics
    struct Stats {
        size_t size;
        uint64_t total_hits;
        uint64_t total_misses;
    };
    Stats getStats() const;

private:
    std::string makeKey(const ContractAddress& contract, const TokenID& token_id) const;
    bool isExpired(const CacheEntry& entry) const;
};

// Batch processor for high throughput
class NFTBatchProcessor {
private:
    static constexpr size_t BATCH_SIZE = 100;

public:
    struct BatchResult {
        size_t total;
        size_t successful;
        size_t failed;
        std::vector<TxHash> tx_hashes;
        std::vector<std::string> errors;
    };

    // Process batch transfers
    static BatchResult processTransfers(
        const std::vector<TransferRequest>& transfers,
        NFTSignerCXX& signer
    );

    // Process batch mints
    static BatchResult processMints(
        const std::vector<MintRequest>& mints,
        NFTSignerCXX& signer
    );
};

// Security validators
class NFTValidator {
public:
    static bool isValidAddress(const Address& address);
    static bool isValidTokenID(const TokenID& token_id);
    static bool isValidContract(const ContractAddress& contract);
    static bool isValidURI(const std::string& uri);

    struct ValidationResult {
        bool valid;
        std::vector<std::string> errors;
        std::vector<std::string> warnings;
    };

    static ValidationResult validateMetadata(const NFTMetadata& metadata);
    static ValidationResult validateTransfer(const TransferRequest& req);
    static ValidationResult validateMint(const MintRequest& req);
};

// Floor price calculator
class FloorPriceCalculator {
private:
    std::vector<Offer> recent_sales_;

public:
    void addSale(const Offer& offer);
    std::optional<Amount> getFloorPrice() const;
    std::optional<Amount> getAveragePrice() const;
    Amount getVolume() const;
    std::optional<Amount> getMedianPrice() const;
};

} // namespace nft
} // namespace tigerwallet

#endif // NFT_CORE_HPP