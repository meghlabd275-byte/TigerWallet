/**
 * TigerWallet Desktop - NFT Service
 * NFT operations including viewing, transferring, and metadata
 */

#ifndef TIGER_WALLET_NFT_SERVICE_H
#define TIGER_WALLET_NFT_SERVICE_H

#include "models/wallet_models.h"
#include <memory>
#include <string>
#include <vector>
#include <future>
#include <optional>
#include <curl/curl.h>

namespace tiger {
namespace wallet {

// ============================================================================
// NFT Service
// ============================================================================

class NFTService {
public:
    static std::shared_ptr<NFTService> getInstance();

    // Initialization
    void initialize();
    void shutdown();

    // NFT Fetching
    std::future<std::vector<NFT>> getNFTs(const std::string& walletAddress, const std::string& chainId);
    std::future<NFT> getNFTMetadata(const std::string& contractAddress, const std::string& tokenId, const std::string& chainId);
    std::future<std::vector<NFT>> getNFTsByCollection(const std::string& collectionAddress, const std::string& chainId);

    // NFT Operations
    std::future<std::string> transferNFT(
        const std::string& walletId,
        const std::string& contractAddress,
        const std::string& tokenId,
        const std::string& toAddress,
        const std::string& chainId
    );

    std::future<std::string> mintNFT(
        const std::string& walletId,
        const std::string& contractAddress,
        const std::string& metadataUrl,
        const std::string& chainId
    );

    // Collection Info
    struct CollectionInfo {
        std::string address;
        std::string name;
        std::string symbol;
        int total_supply;
        std::optional<std::string> description;
        std::optional<std::string> image_url;
        std::optional<std::string> floor_price;
    };

    std::future<CollectionInfo> getCollectionInfo(const std::string& collectionAddress, const std::string& chainId);

    // IPFS Support
    std::string resolveIPFS(const std::string& ipfsUrl);

private:
    NFTService(const NFTService&) = delete;
    NFTService& operator=(const NFTService&) = delete;

public:
    NFTService();
    ~NFTService();

    // API Calls
    std::string fetchFromAPI(const std::string& url);
    NFT parseNFTFromJSON(const std::string& json);

    // Members
    static std::shared_ptr<NFTService> instance_;
    CURL* curl_;
    bool initialized_;
};

// ============================================================================
// Exception
// ============================================================================

class NFTServiceException : public std::runtime_error {
public:
    enum class ErrorCode {
        CollectionNotFound,
        TokenNotFound,
        TransferFailed,
        NetworkError,
        UnsupportedChain,
        Unknown
    };

    NFTServiceException(ErrorCode code, const std::string& message);
    ErrorCode getErrorCode() const;

private:
    ErrorCode code_;
};

} // namespace wallet
} // namespace tiger

#endif // TIGER_WALLET_NFT_SERVICE_H
