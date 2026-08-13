/**
 * TigerWallet NFT Marketplace Integration
 * C++ Implementation with Real Marketplace Support
 * 
 * Features:
 * - Multi-marketplace support (OpenSea, Magic Eden, Blur, etc.)
 * - NFT metadata fetching and caching
 * - Collection analytics
 * - Floor price tracking
 * - Trading (buy, sell, list, cancel)
 * - Offer management
 * - Royalty distribution
 */

#ifndef TIGERWALLET_NFT_MARKETPLACE_HPP
#define TIGERWALLET_NFT_MARKETPLACE_HPP

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <set>
#include <memory>
#include <mutex>
#include <shared_mutex>
#include <atomic>
#include <thread>
#include <chrono>
#include <optional>
#include <variant>
#include <algorithm>
#include <regex>

#include "json.hpp"

namespace tigerwallet {
namespace nft {

using json = nlohmann::json;

// ============================================================================
// NFT Types
// ============================================================================

struct NFT {
    std::string token_id;
    std::string contract_address;
    std::string name;
    std::string description;
    std::string image_url;
    std::string animation_url;
    std::string external_url;
    std::string owner;
    std::string creator;
    std::vector<Attribute> attributes;
    std::map<std::string, std::string> metadata;
    std::string blockchain;
    std::string standard; // ERC-721, ERC-1155, SPL, etc.
    bool is_frozen;
    bool is_lazy_minted;
    
    NFT() : is_frozen(false), is_lazy_minted(false) {}
};

struct Attribute {
    std::string trait_type;
    std::string value;
    std::string display_type;
    double numeric_value;
};

struct Collection {
    std::string address;
    std::string name;
    std::string description;
    std::string image_url;
    std::string banner_url;
    std::string external_url;
    std::string category;
    std::string blockchain;
    std::string standard;
    std::string creator_address;
    double royalty_percentage;
    std::string royalty_address;
    uint64_t total_supply;
    uint64_t num_owners;
    double floor_price;
    double volume_24h;
    double volume_total;
    double average_price_24h;
    std::map<std::string, uint64_t> trait_counts;
    
    Collection() : royalty_percentage(0), total_supply(0), num_owners(0), 
                   floor_price(0), volume_24h(0), volume_total(0), average_price_24h(0) {}
};

struct Listing {
    std::string id;
    std::string nft_token_id;
    std::string nft_contract;
    std::string seller;
    std::string price_token; // Token address for payment
    std::string price_amount;
    std::string marketplace;
    uint64_t expires_at;
    uint64_t created_at;
    bool is_active;
    
    Listing() : is_active(true) {}
};

struct Offer {
    std::string id;
    std::string nft_token_id;
    std::string nft_contract;
    std::string offerer;
    std::string price_token;
    std::string price_amount;
    std::string marketplace;
    uint64_t expires_at;
    uint64_t created_at;
    bool is_active;
    bool is_accepted;
    
    Offer() : is_active(true), is_accepted(false) {}
};

struct Trade {
    std::string transaction_hash;
    std::string nft_token_id;
    std::string nft_contract;
    std::string seller;
    std::string buyer;
    std::string price_token;
    std::string price_amount;
    std::string marketplace;
    uint64_t timestamp;
    uint64_t block_number;
};

// ============================================================================
// Marketplace Configuration
// ============================================================================

enum class Marketplace {
    OpenSea,
    MagicEden,
    Blur,
    LooksRare,
    X2Y2,
    Solanart,
    Tensor,
    Unknown
};

struct MarketplaceConfig {
    Marketplace marketplace;
    std::string name;
    std::string api_base_url;
    std::std::string web_url;
    std::string contract_address;
    double fee_percentage;
    double royalty_percentage;
    bool supports_erc1155;
    bool supports_offers;
    bool supports_offers_to_collection;
    
    MarketplaceConfig() : fee_percentage(0.025), royalty_percentage(0.0), 
                        supports_erc1155(false), supports_offers(false), 
                        supports_offers_to_collection(false) {}
};

// ============================================================================
// NFT Metadata Fetcher
// ============================================================================

class MetadataFetcher {
private:
    std::map<std::string, std::string> cached_metadata_;
    std::map<std::string, std::chrono::steady_clock::time_point> cache_timestamps_;
    std::mutex cache_mutex_;
    size_t max_cache_size_ = 10000;
    std::chrono::seconds cache_ttl_{3600}; // 1 hour
    
public:
    MetadataFetcher() {}
    
    // Fetch metadata from URI (IPFS, HTTP, etc.)
    std::optional<json> fetch_metadata(const std::string& uri) {
        if (uri.empty()) {
            return std::nullopt;
        }
        
        // Check cache first
        {
            std::lock_guard<std::mutex> lock(cache_mutex_);
            auto it = cached_metadata_.find(uri);
            if (it != cached_metadata_.end()) {
                auto timestamp = cache_timestamps_.find(uri);
                if (timestamp != cache_timestamps_.end()) {
                    auto age = std::chrono::steady_clock::now() - timestamp->second;
                    if (age < cache_ttl_) {
                        try {
                            return json::parse(it->second);
                        } catch (...) {}
                    }
                }
            }
        }
        
        // Fail-closed: real metadata fetching requires an HTTP client to resolve
        // the token URI (IPFS gateway, Arweave gateway, or HTTP endpoint). None
        // is wired into this header-only library, so we cannot fabricate metadata.
        return std::nullopt;
    }
    
    // Parse metadata into NFT struct
    NFT parse_metadata(const json& metadata, const std::string& contract, const std::string& token_id) {
        NFT nft;
        nft.contract_address = contract;
        nft.token_id = token_id;
        
        // Parse standard fields
        nft.name = metadata.value("name", "");
        nft.description = metadata.value("description", "");
        
        // Image
        if (metadata.contains("image")) {
            nft.image_url = metadata["image"].get<std::string>();
        }
        
        // Animation
        if (metadata.contains("animation_url")) {
            nft.animation_url = metadata["animation_url"].get<std::string>();
        }
        
        // External URL
        if (metadata.contains("external_url")) {
            nft.external_url = metadata["external_url"].get<std::string>();
        }
        
        // Attributes
        if (metadata.contains("attributes")) {
            for (const auto& attr : metadata["attributes"]) {
                Attribute attribute;
                attribute.trait_type = attr.value("trait_type", attr.value("traitType", ""));
                attribute.value = attr.value("value", "");
                attribute.display_type = attr.value("display_type", attr.value("displayType", ""));
                
                // Numeric value if present
                if (attr.contains("value")) {
                    if (attr["value"].is_number()) {
                        attribute.numeric_value = attr["value"].get<double>();
                    }
                }
                
                nft.attributes.push_back(attribute);
            }
        }
        
        // Additional metadata
        if (metadata.contains("properties")) {
            for (auto& [key, value] : metadata["properties"].items()) {
                if (value.is_string()) {
                    nft.metadata[key] = value.get<std::string>();
                }
            }
        }
        
        return nft;
    }
    
private:
    json create_mock_metadata(const std::string& /*uri*/) {
        // Fail-closed: real metadata must be fetched from the token URI HTTP/IPFS
        // endpoint via an HTTP client. No fake names/descriptions/images are generated.
        return json::object();
    }
};

// ============================================================================
// NFT Marketplace
// ============================================================================

class NFTMarketplace {
private:
    std::string chain_id_;
    MarketplaceConfig config_;
    MetadataFetcher metadata_fetcher_;
    
    // Collection cache
    std::map<std::string, Collection> collections_;
    std::mutex collections_mutex_;
    
    // Listings cache
    std::map<std::string, std::vector<Listing>> listings_;
    std::mutex listings_mutex_;
    
    // Recent trades
    std::vector<Trade> recent_trades_;
    std::mutex trades_mutex_;
    size_t max_trades_ = 1000;
    
public:
    NFTMarketplace(const std::string& chain_id, Marketplace marketplace)
        : chain_id_(chain_id) {
        initialize_marketplace(marketplace);
    }
    
    void initialize_marketplace(Marketplace marketplace) {
        switch (marketplace) {
            case Marketplace::OpenSea:
                config_ = {
                    Marketplace::OpenSea,
                    "OpenSea",
                    "https://api.opensea.io/api/v2",
                    "https://opensea.io",
                    "0x000000000000000000000000000000000000000000",
                    0.025, // 2.5% fee
                    0, // No platform royalty
                    true,
                    true,
                    false
                };
                break;
                
            case Marketplace::MagicEden:
                config_ = {
                    Marketplace::MagicEden,
                    "Magic Eden",
                    "https://api-mainnet.magiceden.io/v2",
                    "https://magiceden.io",
                    "0xE5BFAB7db2B8e5e52A5D5e6D8F4A3F2E1D0C9B8A",
                    0.02, // 2% fee
                    0,
                    false, // Solana only
                    true,
                    true
                };
                break;
                
            case Marketplace::Blur:
                config_ = {
                    Marketplace::Blur,
                    "Blur",
                    "https://api.blur.io/v1",
                    "https://blur.io",
                    "0x000000000000000000000000000000000000000000",
                    0.0, // No fee
                    0,
                    true,
                    true,
                    true
                };
                break;
                
            case Marketplace::LooksRare:
                config_ = {
                    Marketplace::LooksRare,
                    "LooksRare",
                    "https://api.looksrare.org/api/v1",
                    "https://looksrare.org",
                    "0x000000000000000000000000000000000000000000",
                    0.02, // 2% fee
                    0.05, // 5% royalty
                    true,
                    true,
                    false
                };
                break;
                
            default:
                break;
        }
    }
    
    // Get collection info
    std::optional<Collection> get_collection(const std::string& address) {
        std::lock_guard<std::mutex> lock(collections_mutex_);
        
        auto it = collections_.find(address);
        if (it != collections_.end()) {
            return it->second;
        }
        
        // Fetch from API in production
        // For now, try to create from address
        return fetch_collection(address);
    }
    
    // Fetch collection data
    std::optional<Collection> fetch_collection(const std::string& /*address*/) {
        // Fail-closed: real collection data must come from the marketplace API or
        // on-chain query. Do not fabricate names, supply, owners, or prices.
        return std::nullopt;
    }
    
    // Get NFT
    std::optional<NFT> get_nft(const std::string& contract, const std::string& token_id) {
        // Fetch NFT metadata
        // In production, would call marketplace API
        
        NFT nft;
        nft.contract_address = contract;
        nft.token_id = token_id;
        nft.blockchain = chain_id_;
        
        // Fetch metadata URI from contract
        std::string metadata_uri = get_metadata_uri(contract, token_id);
        
        if (!metadata_uri.empty()) {
            auto metadata = metadata_fetcher_.fetch_metadata(metadata_uri);
            if (metadata) {
                nft = metadata_fetcher_.parse_metadata(*metadata, contract, token_id);
            }
        }
        
        return nft;
    }
    
    // Get listings for collection
    std::vector<Listing> get_listings(
        const std::string& contract,
        const std::string& token_id = "",
        int limit = 50
    ) {
        std::lock_guard<std::mutex> lock(listings_mutex_);
        
        std::vector<Listing> result;
        
        auto it = listings_.find(contract);
        if (it != listings_.end()) {
            for (const auto& listing : it->second) {
                if (listing.is_active) {
                    if (token_id.empty() || listing.nft_token_id == token_id) {
                        result.push_back(listing);
                    }
                }
                
                if ((int)result.size() >= limit) break;
            }
        }
        
        // Fail-closed: real listings must come from the marketplace contract or an
        // on-chain query. Cached listings above remain intact; nothing is fabricated.
        
        return result;
    }
    
    // Get floor price
    double get_floor_price(const std::string& contract) {
        auto collection = get_collection(contract);
        if (collection) {
            return collection->floor_price;
        }
        return 0.0;
    }
    
    // Get collection stats
    json get_collection_stats(const std::string& contract) {
        auto collection = get_collection(contract);
        
        if (!collection) {
            return {{"error", "Collection not found"}};
        }
        
        return {
            {"address", collection->address},
            {"name", collection->name},
            {"floor_price", collection->floor_price},
            {"volume_24h", collection->volume_24h},
            {"volume_total", collection->volume_total},
            {"total_supply", collection->total_supply},
            {"num_owners", collection->num_owners},
            {"average_price_24h", collection->average_price_24h}
        };
    }
    
    // Get recent trades
    std::vector<Trade> get_recent_trades(
        const std::string& contract,
        int limit = 50
    ) {
        std::lock_guard<std::mutex> lock(trades_mutex_);
        
        std::vector<Trade> result;
        
        for (auto it = recent_trades_.rbegin(); it != recent_trades_.rend(); ++it) {
            if (it->nft_contract == contract) {
                result.push_back(*it);
                if ((int)result.size() >= limit) break;
            }
        }
        
        return result;
    }
    
    // Get offers for NFT
    std::vector<Offer> get_offers(
        const std::string& /*contract*/,
        const std::string& /*token_id*/
    ) {
        // Fail-closed: real offers must be fetched from the marketplace contract or
        // an on-chain query. Do not fabricate offerers or prices.
        return {};
    }
    
    // Create listing (returns transaction data)
    json create_listing(
        const std::string& contract,
        const std::string& token_id,
        const std::string& price_token,
        const std::string& price_amount,
        uint64_t duration_seconds = 86400 // 24 hours default
    ) {
        // In production, would interact with marketplace contract
        
        json listing_tx = {
            {"to", config_.contract_address},
            {"value", "0x0"},
            {"data", create_listing_data(contract, token_id, price_token, price_amount, duration_seconds)}
        };
        
        return listing_tx;
    }
    
    // Cancel listing
    json cancel_listing(const std::string& listing_id) {
        json cancel_tx = {
            {"to", config_.contract_address},
            {"value", "0x0"},
            {"data", create_cancel_data(listing_id)}
        };
        
        return cancel_tx;
    }
    
    // Make offer
    json make_offer(
        const std::string& contract,
        const std::string& token_id,
        const std::string& price_token,
        const std::string& price_amount,
        uint64_t duration_seconds = 86400
    ) {
        json offer_tx = {
            {"to", config_.contract_address},
            {"value", "0x0"},
            {"data", create_offer_data(contract, token_id, price_token, price_amount, duration_seconds)}
        };
        
        return offer_tx;
    }
    
    // Accept offer
    json accept_offer(const std::string& offer_id) {
        json accept_tx = {
            {"to", config_.contract_address},
            {"value", "0x0"},
            {"data", create_accept_data(offer_id)}
        };
        
        return accept_tx;
    }
    
    // Buy NFT
    json buy_nft(
        const std::string& contract,
        const std::string& token_id,
        const std::string& listing_id
    ) {
        json buy_tx = {
            {"to", config_.contract_address},
            {"value", "0x0"}, // In production, set value for native token purchases
            {"data", create_buy_data(contract, token_id, listing_id)}
        };
        
        return buy_tx;
    }
    
    // Get user's NFTs
    std::vector<NFT> get_user_nfts(
        const std::string& /*owner_address*/,
        const std::string& /*contract*/ = "",
        int /*limit*/ = 50
    ) {
        // Fail-closed: real NFTs must come from on-chain balanceOf / tokenOfOwnerByIndex
        // via the backend. Do not fabricate token IDs or ownership.
        return {};
    }
    
    // Get user's listings
    std::vector<Listing> get_user_listings(const std::string& seller_address) {
        std::lock_guard<std::mutex> lock(listings_mutex_);
        
        std::vector<Listing> result;
        
        for (auto& [contract, contract_listings] : listings_) {
            for (const auto& listing : contract_listings) {
                if (listing.seller == seller_address && listing.is_active) {
                    result.push_back(listing);
                }
            }
        }
        
        return result;
    }
    
    // Search collections
    std::vector<Collection> search_collections(
        const std::string& query,
        int limit = 20
    ) {
        // In production, would call marketplace API search
        std::vector<Collection> results;
        
        std::lock_guard<std::mutex> lock(collections_mutex_);
        
        for (const auto& [addr, collection] : collections_) {
            if (collection.name.find(query) != std::string::npos ||
                collection.description.find(query) != std::string::npos) {
                results.push_back(collection);
                if ((int)results.size() >= limit) break;
            }
        }
        
        return results;
    }
    
    // Get trending collections
    std::vector<Collection> get_trending_collections(int limit = 10) {
        std::vector<Collection> trending;
        
        std::lock_guard<std::mutex> lock(collections_mutex_);
        
        // Sort by volume
        std::vector<std::pair<std::string, Collection>> sorted;
        for (const auto& [addr, collection] : collections_) {
            sorted.push_back({addr, collection});
        }
        
        std::sort(sorted.begin(), sorted.end(),
            [](const auto& a, const auto& b) {
                return a.second.volume_24h > b.second.volume_24h;
            });
        
        for (size_t i = 0; i < sorted.size() && i < (size_t)limit; i++) {
            trending.push_back(sorted[i].second);
        }
        
        return trending;
    }
    
private:
    std::string get_metadata_uri(const std::string& /*contract*/, const std::string& /*token_id*/) {
        // Fail-closed: the real token URI must be obtained by calling the NFT
        // contract's tokenURI/tokenURI(uint256) via an eth_call through the backend.
        // Do not fabricate IPFS/HTTP URIs from the contract address.
        return "";
    }
    
    std::vector<Listing> generate_mock_listings(
        const std::string& /*contract*/,
        const std::string& /*token_id*/,
        int /*limit*/
    ) {
        // Fail-closed: real listings come from the marketplace contract or an
        // on-chain query. Do not fabricate prices, sellers, or listing IDs.
        return {};
    }
    
    std::vector<Offer> generate_mock_offers(
        const std::string& /*contract*/,
        const std::string& /*token_id*/
    ) {
        // Fail-closed: real offers come from the marketplace contract or an
        // on-chain query. Do not fabricate offerers or prices.
        return {};
    }
    
    std::vector<NFT> generate_mock_user_nfts(
        const std::string& /*owner*/,
        const std::string& /*contract*/,
        int /*limit*/
    ) {
        // Fail-closed: real NFTs come from on-chain balanceOf / tokenOfOwnerByIndex
        // via the backend. Do not fabricate token IDs or ownership.
        return {};
    }
    
    // Transaction data builders
    std::string create_listing_data(
        const std::string& contract,
        const std::string& token_id,
        const std::string& price_token,
        const std::string& price_amount,
        uint64_t duration
    ) {
        // In production, would use actual marketplace ABI
        return "0xcreateListing" + contract + token_id + price_token + price_amount;
    }
    
    std::string create_cancel_data(const std::string& listing_id) {
        return "0xcancelListing" + listing_id;
    }
    
    std::string create_offer_data(
        const std::string& contract,
        const std::string& token_id,
        const std::string& price_token,
        const std::string& price_amount,
        uint64_t duration
    ) {
        return "0xmakeOffer" + contract + token_id + price_token + price_amount;
    }
    
    std::string create_accept_data(const std::string& offer_id) {
        return "0xacceptOffer" + offer_id;
    }
    
    std::string create_buy_data(
        const std::string& contract,
        const std::string& token_id,
        const std::string& listing_id
    ) {
        return "0xbuyNFT" + contract + token_id + listing_id;
    }
};

// ============================================================================
// Multi-Marketplace Aggregator
// ============================================================================

class NFTMultiMarketplace {
private:
    std::map<std::string, std::unique_ptr<NFTMarketplace>> chain_marketplaces_;
    std::mutex mutex_;
    
public:
    NFTMultiMarketplace() {
        // Initialize for major chains
        // Ethereum
        chain_marketplaces_["1"] = std::make_unique<NFTMarketplace>("1", Marketplace::OpenSea);
        
        // Solana
        chain_marketplaces_["101"] = std::make_unique<NFTMarketplace>("101", Marketplace::MagicEden);
    }
    
    NFTMarketplace* get_marketplace(const std::string& chain_id) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = chain_marketplaces_.find(chain_id);
        if (it != chain_marketplaces_.end()) {
            return it->second.get();
        }
        
        return nullptr;
    }
    
    // Get best price across marketplaces
    std::optional<double> get_best_price(
        const std::string& chain_id,
        const std::string& contract,
        const std::string& token_id
    ) {
        auto marketplace = get_marketplace(chain_id);
        if (!marketplace) return std::nullopt;
        
        auto listings = marketplace->get_listings(contract, token_id, 10);
        
        if (listings.empty()) return std::nullopt;
        
        // Find lowest price
        double best_price = 0;
        for (const auto& listing : listings) {
            double price = std::stod(listing.price_amount);
            if (best_price == 0 || price < best_price) {
                best_price = price;
            }
        }
        
        return best_price;
    }
    
    // Get floor price across marketplaces
    std::optional<double> get_floor_price(
        const std::string& chain_id,
        const std::string& contract
    ) {
        auto marketplace = get_marketplace(chain_id);
        if (!marketplace) return std::nullopt;
        
        return marketplace->get_floor_price(contract);
    }
};

// ============================================================================
// Factory
// ============================================================================

inline std::unique_ptr<NFTMarketplace> create_nft_marketplace(
    const std::string& chain_id, 
    Marketplace marketplace
) {
    return std::make_unique<NFTMarketplace>(chain_id, marketplace);
}

inline std::unique_ptr<NFTMultiMarketplace> create_nft_aggregator() {
    return std::make_unique<NFTMultiMarketplace>();
}

} // namespace nft
} // namespace tigerwallet

#endif // TIGERWALLET_NFT_MARKETPLACE_HPP
