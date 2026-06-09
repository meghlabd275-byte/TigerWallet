#include "nft_core.hpp"
#include <algorithm>
#include <sstream>
#include <iomanip>
#include <cstring>

namespace tigerwallet {
namespace nft {

// NFTSignerCXX Implementation
NFTSignerCXX::NFTSignerCXX() : initialized_(false) {}

NFTSignerCXX::NFTSignerCXX(const std::string& private_key_hex) : initialized_(false) {
    initialize(private_key_hex);
}

NFTSignerCXX::~NFTSignerCXX() {
    // Secure clear private key
    std::fill(private_key_.begin(), private_key_.end(), 0);
}

bool NFTSignerCXX::initialize(const std::string& private_key_hex) {
    if (private_key_hex.length() != 64) {
        return false;
    }

    // Parse hex string
    private_key_.resize(32);
    for (size_t i = 0; i < 32; i++) {
        std::string byte_str = private_key_hex.substr(i * 2, 2);
        private_key_[i] = static_cast<uint8_t>(std::stoi(byte_str, nullptr, 16));
    }

    initialized_ = true;
    return true;
}

std::optional<std::string> NFTSignerCXX::signTransfer(
    const Address& from,
    const Address& to,
    const ContractAddress& contract,
    const TokenID& token_id
) {
    if (!initialized_) {
        return std::nullopt;
    }

    auto data = prepareTransferData(to, token_id);
    auto signature = signECDSA(data);

    // Return signed transaction data
    std::ostringstream oss;
    for (const auto& b : signature) {
        oss << std::hex << std::setw(2) << std::setfill('0') << static_cast<int>(b);
    }

    return oss.str();
}

std::optional<std::string> NFTSignerCXX::signMint(
    const Address& to,
    const ContractAddress& contract,
    const std::string& uri
) {
    if (!initialized_) {
        return std::nullopt;
    }

    auto data = prepareMintData(to, uri);
    auto signature = signECDSA(data);

    std::ostringstream oss;
    for (const auto& b : signature) {
        oss << std::hex << std::setw(2) << std::setfill('0') << static_cast<int>(b);
    }

    return oss.str();
}

std::optional<std::string> NFTSignerCXX::signApproval(
    const Address& owner,
    const Address& operator_,
    bool approved
) {
    if (!initialized_) {
        return std::nullopt;
    }

    auto data = prepareApprovalData(operator_, approved);
    auto signature = signECDSA(data);

    std::ostringstream oss;
    for (const auto& b : signature) {
        oss << std::hex << std::setw(2) << std::setfill('0') << static_cast<int>(b);
    }

    return oss.str();
}

std::optional<Address> NFTSignerCXX::getAddress() const {
    if (!initialized_) {
        return std::nullopt;
    }

    // Derive address from public key (simplified)
    std::vector<uint8_t> pubkey = private_key_; // In production, derive from private key
    auto hash = keccak256(pubkey);

    // Take last 20 bytes as address
    std::ostringstream oss;
    oss << "0x";
    for (size_t i = hash.size() - 20; i < hash.size(); i++) {
        oss << std::hex << std::setw(2) << std::setfill('0') << static_cast<int>(hash[i]);
    }

    return oss.str();
}

std::vector<uint8_t> NFTSignerCXX::prepareTransferData(const Address& to, const TokenID& token_id) const {
    std::vector<uint8_t> data;

    // Function selector for safeTransferFrom(address,address,uint256)
    std::vector<uint8_t> selector = {0x4, 0x2, 0x3, 0x2}; // First 4 bytes of keccak256
    data.insert(data.end(), selector.begin(), selector.end());

    // Pad from address (32 bytes of zero)
    data.resize(data.size() + 32, 0);

    // Pad to address
    std::vector<uint8_t> to_bytes(to.begin() + 2, to.end()); // Skip 0x
    data.insert(data.end(), to_bytes.begin(), to_bytes.end());
    while (data.size() % 32 != 20) {} // Pad to 32 bytes

    // Pad token ID
    std::vector<uint8_t> token_bytes(token_id.begin(), token_id.end());
    data.insert(data.end(), token_bytes.begin(), token_bytes.end());
    while (data.size() % 32 != 0) {
        data.push_back(0);
    }

    return data;
}

std::vector<uint8_t> NFTSignerCXX::prepareMintData(const Address& to, const std::string& uri) const {
    std::vector<uint8_t> data;

    // Function selector for mint(address,string)
    std::vector<uint8_t> selector = {0x9, 0x5, 0xc1, 0x8}; // Placeholder
    data.insert(data.end(), selector.begin(), selector.end());

    // Pad to address
    std::vector<uint8_t> to_bytes(to.begin() + 2, to.end());
    data.insert(data.end(), to_bytes.begin(), to_bytes.end());
    while (data.size() % 32 != 20) {
        data.push_back(0);
    }

    // String offset
    data.resize(data.size() + 32, 0);
    data[data.size() - 32] = 0x40; // Offset to string data

    // String length
    std::vector<uint8_t> len_bytes(32, 0);
    uint32_t uri_len = static_cast<uint32_t>(uri.size());
    len_bytes[28] = static_cast<uint8_t>((uri_len >> 24) & 0xFF);
    len_bytes[29] = static_cast<uint8_t>((uri_len >> 16) & 0xFF);
    len_bytes[30] = static_cast<uint8_t>((uri_len >> 8) & 0xFF);
    len_bytes[31] = static_cast<uint8_t>(uri_len & 0xFF);
    data.insert(data.end(), len_bytes.begin(), len_bytes.end());

    // String data
    std::vector<uint8_t> uri_bytes(uri.begin(), uri.end());
    data.insert(data.end(), uri_bytes.begin(), uri_bytes.end());

    return data;
}

std::vector<uint8_t> NFTSignerCXX::prepareApprovalData(const Address& operator_, bool approved) const {
    std::vector<uint8_t> data;

    // Function selector for setApprovalForAll(address,bool)
    std::vector<uint8_t> selector = {0x1, 0x6, 0x2, 0x3}; // Placeholder
    data.insert(data.end(), selector.begin(), selector.end());

    // Pad operator address
    std::vector<uint8_t> op_bytes(operator_.begin() + 2, operator_.end());
    data.insert(data.end(), op_bytes.begin(), op_bytes.end());
    while (data.size() % 32 != 20) {
        data.push_back(0);
    }

    // Approved bool
    data.resize(data.size() + 31, 0);
    data.push_back(approved ? 1 : 0);

    return data;
}

std::vector<uint8_t> NFTSignerCXX::keccak256(const std::vector<uint8_t>& data) const {
    // Simplified Keccak256 implementation
    std::vector<uint8_t> hash(32, 0);

    // Simple hash for demonstration
    // In production, use proper Keccak256
    for (size_t i = 0; i < data.size(); i++) {
        hash[i % 32] ^= data[i];
        hash[(i + 1) % 32] += data[i];
    }

    return hash;
}

std::vector<uint8_t> NFTSignerCXX::signECDSA(const std::vector<uint8_t>& message) const {
    // Simplified ECDSA signature
    // In production, use proper secp256k1

    std::vector<uint8_t> signature(64, 0);

    // Create deterministic signature from message hash
    auto hash = keccak256(message);

    // Fill with hash-derived values
    for (size_t i = 0; i < 32; i++) {
        signature[i] = private_key_[i] ^ hash[i];
        signature[i + 32] = hash[i];
    }

    return signature;
}

// NFTCache Implementation
NFTCache::NFTCache(size_t max_size, uint64_t ttl_seconds)
    : max_size_(max_size), ttl_seconds_(ttl_seconds) {}

std::optional<NFTMetadata> NFTCache::get(const ContractAddress& contract, const TokenID& token_id) {
    auto key = makeKey(contract, token_id);
    auto it = cache_.find(key);

    if (it == cache_.end()) {
        return std::nullopt;
    }

    if (isExpired(it->second)) {
        cache_.erase(it);
        return std::nullopt;
    }

    it->second.hits++;
    return it->second.metadata;
}

void NFTCache::set(const ContractAddress& contract, const TokenID& token_id, const NFTMetadata& metadata) {
    auto key = makeKey(contract, token_id);

    // Evict oldest if at capacity
    if (cache_.size() >= max_size_) {
        auto oldest = cache_.begin();
        for (auto it = cache_.begin(); it != cache_.end(); ++it) {
            if (it->second.timestamp < oldest->second.timestamp) {
                oldest = it;
            }
        }
        cache_.erase(oldest);
    }

    CacheEntry entry;
    entry.metadata = metadata;
    entry.timestamp = std::chrono::steady_clock::now();
    entry.hits = 0;

    cache_[key] = entry;
}

void NFTCache::invalidate(const ContractAddress& contract, const TokenID& token_id) {
    auto key = makeKey(contract, token_id);
    cache_.erase(key);
}

void NFTCache::invalidateCollection(const ContractAddress& contract) {
    for (auto it = cache_.begin(); it != cache_.end();) {
        if (it->first.find(contract) == 0) {
            it = cache_.erase(it);
        } else {
            ++it;
        }
    }
}

void NFTCache::clear() {
    cache_.clear();
}

NFTCache::Stats NFTCache::getStats() const {
    Stats stats;
    stats.size = cache_.size();
    stats.total_hits = 0;
    stats.total_misses = 0;

    for (const auto& entry : cache_.second) {
        stats.total_hits += entry.second.hits;
    }

    return stats;
}

std::string NFTCache::makeKey(const ContractAddress& contract, const TokenID& token_id) const {
    return contract + ":" + token_id;
}

bool NFTCache::isExpired(const CacheEntry& entry) const {
    auto age = std::chrono::steady_clock::now() - entry.timestamp;
    return std::chrono::duration_cast<std::chrono::seconds>(age).count() > ttl_seconds_;
}

// NFTBatchProcessor Implementation
NFTBatchProcessor::BatchResult NFTBatchProcessor::processTransfers(
    const std::vector<TransferRequest>& transfers,
    NFTSignerCXX& signer
) {
    BatchResult result;
    result.total = transfers.size();

    for (size_t i = 0; i < transfers.size(); i += BATCH_SIZE) {
        size_t batch_end = std::min(i + BATCH_SIZE, transfers.size());

        for (size_t j = i; j < batch_end; j++) {
            const auto& req = transfers[j];

            auto signature = signer.signTransfer(req.from, req.to, req.contract, req.token_id);
            if (signature) {
                result.successful++;
                result.tx_hashes.push_back(*signature);
            } else {
                result.failed++;
                result.errors.push_back("Transfer failed for token: " + req.token_id);
            }
        }
    }

    return result;
}

NFTBatchProcessor::BatchResult NFTBatchProcessor::processMints(
    const std::vector<MintRequest>& mints,
    NFTSignerCXX& signer
) {
    BatchResult result;
    result.total = mints.size();

    for (size_t i = 0; i < mints.size(); i += BATCH_SIZE) {
        size_t batch_end = std::min(i + BATCH_SIZE, mints.size());

        for (size_t j = i; j < batch_end; j++) {
            const auto& req = mints[j];

            auto signature = signer.signMint(req.to, req.contract, req.uri);
            if (signature) {
                result.successful++;
                result.tx_hashes.push_back(*signature);
            } else {
                result.failed++;
                result.errors.push_back("Mint failed for URI: " + req.uri);
            }
        }
    }

    return result;
}

// NFTValidator Implementation
bool NFTValidator::isValidAddress(const Address& address) {
    if (address.size() != 42) return false;
    if (!address.starts_with("0x")) return false;

    // Check hex characters
    for (size_t i = 2; i < address.size(); i++) {
        char c = address[i];
        if (!((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F'))) {
            return false;
        }
    }

    return true;
}

bool NFTValidator::isValidTokenID(const TokenID& token_id) {
    if (token_id.empty()) return false;

    // Try as number
    if (std::all_of(token_id.begin(), token_id.end(), [](char c) { return c >= '0' && c <= '9'; })) {
        return true;
    }

    // Try as hex
    if (token_id.starts_with("0x")) {
        for (size_t i = 2; i < token_id.size(); i++) {
            char c = token_id[i];
            if (!((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F'))) {
                return false;
            }
        }
        return true;
    }

    return false;
}

bool NFTValidator::isValidContract(const ContractAddress& contract) {
    return isValidAddress(contract);
}

bool NFTValidator::isValidURI(const std::string& uri) {
    return uri.starts_with("http://") || uri.starts_with("https://") || uri.starts_with("ipfs://");
}

NFTValidator::ValidationResult NFTValidator::validateMetadata(const NFTMetadata& metadata) {
    ValidationResult result;
    result.valid = true;

    // Validate name
    if (metadata.name.empty()) {
        result.errors.push_back("Name is required");
        result.valid = false;
    } else if (metadata.name.length() > 100) {
        result.errors.push_back("Name exceeds maximum length");
        result.valid = false;
    }

    // Validate image URL
    if (!metadata.image_url.empty()) {
        if (!isValidURI(metadata.image_url)) {
            result.warnings.push_back("Image URL may be invalid");
        }
    } else {
        result.warnings.push_back("Image URL is recommended");
    }

    // Validate animation URL
    if (metadata.animation_url && !metadata.animation_url->empty()) {
        if (!isValidURI(*metadata.animation_url)) {
            result.warnings.push_back("Animation URL may be invalid");
        }
    }

    return result;
}

NFTValidator::ValidationResult NFTValidator::validateTransfer(const TransferRequest& req) {
    ValidationResult result;
    result.valid = true;

    if (!isValidAddress(req.from)) {
        result.errors.push_back("Invalid from address");
        result.valid = false;
    }

    if (!isValidAddress(req.to)) {
        result.errors.push_back("Invalid to address");
        result.valid = false;
    }

    if (!isValidContract(req.contract)) {
        result.errors.push_back("Invalid contract address");
        result.valid = false;
    }

    if (!isValidTokenID(req.token_id)) {
        result.errors.push_back("Invalid token ID");
        result.valid = false;
    }

    // Prevent transfers to zero address
    if (req.to == "0x0000000000000000000000000000000000000000") {
        result.errors.push_back("Cannot transfer to zero address");
        result.valid = false;
    }

    return result;
}

NFTValidator::ValidationResult NFTValidator::validateMint(const MintRequest& req) {
    ValidationResult result;
    result.valid = true;

    if (!isValidAddress(req.to)) {
        result.errors.push_back("Invalid to address");
        result.valid = false;
    }

    if (!isValidContract(req.contract)) {
        result.errors.push_back("Invalid contract address");
        result.valid = false;
    }

    if (!isValidURI(req.uri)) {
        result.errors.push_back("Invalid URI");
        result.valid = false;
    }

    return result;
}

// FloorPriceCalculator Implementation
void FloorPriceCalculator::addSale(const Offer& offer) {
    recent_sales_.push_back(offer);

    // Keep only last 100 sales
    if (recent_sales_.size() > 100) {
        recent_sales_.erase(recent_sales_.begin());
    }
}

std::optional<Amount> FloorPriceCalculator::getFloorPrice() const {
    if (recent_sales_.empty()) {
        return std::nullopt;
    }

    std::optional<Amount> min_price;
    for (const auto& offer : recent_sales_) {
        if (offer.status == OfferStatus::Completed) {
            if (!min_price || offer.price < *min_price) {
                min_price = offer.price;
            }
        }
    }

    return min_price;
}

std::optional<Amount> FloorPriceCalculator::getAveragePrice() const {
    if (recent_sales_.empty()) {
        return std::nullopt;
    }

    uint64_t sum = 0;
    uint64_t count = 0;

    for (const auto& offer : recent_sales_) {
        if (offer.status == OfferStatus::Completed) {
            try {
                sum += std::stoull(offer.price);
                count++;
            } catch (...) {}
        }
    }

    if (count == 0) {
        return std::nullopt;
    }

    return std::to_string(sum / count);
}

Amount FloorPriceCalculator::getVolume() const {
    uint64_t total = 0;

    for (const auto& offer : recent_sales_) {
        if (offer.status == OfferStatus::Completed) {
            try {
                total += std::stoull(offer.price);
            } catch (...) {}
        }
    }

    return std::to_string(total);
}

std::optional<Amount> FloorPriceCalculator::getMedianPrice() const {
    std::vector<std::string> prices;

    for (const auto& offer : recent_sales_) {
        if (offer.status == OfferStatus::Completed) {
            prices.push_back(offer.price);
        }
    }

    if (prices.empty()) {
        return std::nullopt;
    }

    std::sort(prices.begin(), prices.end());
    size_t mid = prices.size() / 2;

    if (prices.size() % 2 == 0) {
        uint64_t p1 = std::stoull(prices[mid - 1]);
        uint64_t p2 = std::stoull(prices[mid]);
        return std::to_string((p1 + p2) / 2);
    }

    return prices[mid];
}

} // namespace nft
} // namespace tigerwallet