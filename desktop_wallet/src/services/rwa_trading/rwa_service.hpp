#ifndef TIGERWALLET_RWA_SERVICE_HPP
#define TIGERWALLET_RWA_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <mutex>
#include <chrono>
#include <atomic>
#include <functional>
#include <sstream>
#include <iomanip>
#include <algorithm>

namespace tigerwallet {

// =============================================================================
// RWA TYPES
// =============================================================================

enum class RWAType {
    REAL_ESTATE,
    COMMODITIES,
    ART,
    VINTAGE_WINE,
    PRIVATE_EQUITY,
    FUND,
    BOND,
    STOCK,
    SECURITY_TOKEN,
    OTHER
};

enum class RWAOrderType {
    BUY,
    SELL
};

enum class RWAOrderStatus {
    PENDING,
    OPEN,
    FILLED,
    PARTIALLY_FILLED,
    CANCELLED,
    REJECTED,
    EXPIRED
};

enum class RWAListingStatus {
    ACTIVE,
    PAUSED,
    SOLD,
    CANCELLED
};

// =============================================================================
// RWA DATA STRUCTURES
// =============================================================================

struct RWAAsset {
    std::string id;
    std::string name;
    std::string symbol;
    RWAType type;
    std::string description;
    std::string asset_address;
    std::string contract_address;
    std::string image_url;
    std::string metadata_uri;
    double total_supply;
    double circulating_supply;
    double price;
    double price_change_24h;
    double volume_24h;
    double market_cap;
    std::string currency;
    std::string jurisdiction;
    std::vector<std::string> features;
    bool is_fractional;
    bool is_verified;
    uint64_t created_at;
    uint64_t updated_at;
    
    RWAAsset() : type(RWAType::OTHER), total_supply(0), circulating_supply(0),
                 price(0), price_change_24h(0), volume_24h(0), market_cap(0),
                 is_fractional(false), is_verified(false), created_at(0), updated_at(0) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"id\":\"" << id << "\",";
        oss << "\"name\":\"" << name << "\",";
        oss << "\"symbol\":\"" << symbol << "\",";
        oss << "\"type\":\"" << static_cast<int>(type) << "\",";
        oss << "\"description\":\"" << description << "\",";
        oss << "\"price\":" << std::fixed << std::setprecision(2) << price << ",";
        oss << "\"priceChange24h\":" << price_change_24h << ",";
        oss << "\"volume24h\":" << volume_24h << ",";
        oss << "\"marketCap\":" << market_cap << ",";
        oss << "\"totalSupply\":" << total_supply << ",";
        oss << "\"isFractional\":" << (is_fractional ? "true" : "false") << ",";
        oss << "\"isVerified\":" << (is_verified ? "true" : "false");
        oss << "}";
        return oss.str();
    }
};

struct RWAOrder {
    std::string order_id;
    std::string asset_id;
    std::string user_address;
    RWAOrderType order_type;
    double amount;
    double price;
    double filled_amount;
    double total_value;
    RWAOrderStatus status;
    std::string payment_token;
    uint64_t created_at;
    uint64_t expires_at;
    uint64_t updated_at;
    
    RWAOrder() : order_type(RWAOrderType::BUY), amount(0), price(0),
                 filled_amount(0), total_value(0), status(RWAOrderStatus::PENDING),
                 created_at(0), expires_at(0), updated_at(0) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"orderId\":\"" << order_id << "\",";
        oss << "\"assetId\":\"" << asset_id << "\",";
        oss << "\"orderType\":\"" << (order_type == RWAOrderType::BUY ? "BUY" : "SELL") << "\",";
        oss << "\"amount\":" << std::fixed << std::setprecision(8) << amount << ",";
        oss << "\"price\":" << price << ",";
        oss << "\"filledAmount\":" << filled_amount << ",";
        oss << "\"totalValue\":" << total_value << ",";
        oss << "\"status\":\"" << static_cast<int>(status) << "\"";
        oss << "}";
        return oss.str();
    }
};

struct RWAListing {
    std::string listing_id;
    std::string asset_id;
    std::string seller_address;
    double amount;
    double price_per_unit;
    double total_value;
    RWAListingStatus status;
    std::string payment_token;
    uint64_t created_at;
    uint64_t expires_at;
    
    RWAListing() : amount(0), price_per_unit(0), total_value(0),
                   status(RWAListingStatus::ACTIVE), created_at(0), expires_at(0) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"listingId\":\"" << listing_id << "\",";
        oss << "\"assetId\":\"" << asset_id << "\",";
        oss << "\"amount\":" << amount << ",";
        oss << "\"pricePerUnit\":" << price_per_unit << ",";
        oss << "\"totalValue\":" << total_value << ",";
        oss << "\"status\":\"" << static_cast<int>(status) << "\"";
        oss << "}";
        return oss.str();
    }
};

struct RWAHolding {
    std::string asset_id;
    std::string owner_address;
    double balance;
    double locked_balance;
    double available_balance;
    double cost_basis;
    double current_value;
    double pnl;
    double pnl_percent;
    
    RWAHolding() : balance(0), locked_balance(0), available_balance(0),
                   cost_basis(0), current_value(0), pnl(0), pnl_percent(0) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"assetId\":\"" << asset_id << "\",";
        oss << "\"balance\":" << std::fixed << std::setprecision(8) << balance << ",";
        oss << "\"availableBalance\":" << available_balance << ",";
        oss << "\"currentValue\":" << current_value << ",";
        oss << "\"pnl\":" << pnl << ",";
        oss << "\"pnlPercent\":" << std::setprecision(2) << pnl_percent << "%";
        oss << "}";
        return oss.str();
    }
};

struct RWATransaction {
    std::string tx_id;
    std::string order_id;
    std::string asset_id;
    std::string from_address;
    std::string to_address;
    double amount;
    double price;
    double total_value;
    double fee;
    std::string status;
    uint64_t timestamp;
    
    RWATransaction() : amount(0), price(0), total_value(0), fee(0), timestamp(0) {}
    
    std::string toJson() const {
        std::ostringstream oss;
        oss << "{";
        oss << "\"txId\":\"" << tx_id << "\",";
        oss << "\"assetId\":\"" << asset_id << "\",";
        oss << "\"from\":\"" << from_address << "\",";
        oss << "\"to\":\"" << to_address << "\",";
        oss << "\"amount\":" << std::fixed << std::setprecision(8) << amount << ",";
        oss << "\"totalValue\":" << total_value << ",";
        oss << "\"fee\":" << fee << ",";
        oss << "\"status\":\"" << status << "\",";
        oss << "\"timestamp\":" << timestamp;
        oss << "}";
        return oss.str();
    }
};

// =============================================================================
// RWA SERVICE IMPLEMENTATION
// =============================================================================

class RWAService {
private:
    std::map<std::string, RWAAsset> assets;
    std::map<std::string, std::vector<RWAOrder>> orders;
    std::map<std::string, std::vector<RWAListing>> listings;
    std::map<std::string, std::vector<RWAHolding>> holdings;
    std::map<std::string, std::vector<RWATransaction>> transactions;
    
    std::mutex assets_mutex;
    std::mutex orders_mutex;
    std::mutex listings_mutex;
    std::mutex holdings_mutex;
    std::mutex transactions_mutex;
    
    std::atomic<uint64_t> order_counter{1};
    std::atomic<uint64_t> listing_counter{1};
    std::atomic<uint64_t> tx_counter{1};
    
    bool initialized;
    
    uint64_t getCurrentTimestamp() const {
        return std::chrono::duration_cast<std::chrono::seconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count();
    }
    
    std::string generateOrderId() {
        std::ostringstream oss;
        oss << "RWA-ORD-" << order_counter.fetch_add(1) << "-" << getCurrentTimestamp();
        return oss.str();
    }
    
    std::string generateListingId() {
        std::ostringstream oss;
        oss << "RWA-LST-" << listing_counter.fetch_add(1) << "-" << getCurrentTimestamp();
        return oss.str();
    }
    
    std::string generateTxId() {
        std::ostringstream oss;
        oss << "0x" << std::hex << tx_counter.fetch_add(1) << getCurrentTimestamp();
        return oss.str();
    }
    
    void initializeDefaultAssets() {
        // Fail-closed: do NOT seed fabricated "verified" RWA assets with
        // placeholder contract addresses. A real RWA marketplace must fetch
        // verified assets from the canonical backend / on-chain registry.
        // The asset map starts empty and is populated by real fetches.
    }
    
public:
    RWAService() : initialized(false) {}
    
    ~RWAService() {}
    
    bool initialize() {
        if (initialized) return true;
        
        std::lock_guard<std::mutex> lock(assets_mutex);
        initializeDefaultAssets();
        
        initialized = true;
        return true;
    }
    
    // Get all available RWA assets
    std::vector<RWAAsset> getAssets() {
        std::lock_guard<std::mutex> lock(assets_mutex);
        std::vector<RWAAsset> result;
        for (const auto& pair : assets) {
            result.push_back(pair.second);
        }
        return result;
    }
    
    // Get asset by ID
    RWAAsset getAsset(const std::string& asset_id) {
        std::lock_guard<std::mutex> lock(assets_mutex);
        auto it = assets.find(asset_id);
        if (it != assets.end()) {
            return it->second;
        }
        return RWAAsset();
    }
    
    // Get assets by type
    std::vector<RWAAsset> getAssetsByType(RWAType type) {
        std::lock_guard<std::mutex> lock(assets_mutex);
        std::vector<RWAAsset> result;
        for (const auto& pair : assets) {
            if (pair.second.type == type) {
                result.push_back(pair.second);
            }
        }
        return result;
    }
    
    // Create buy order
    RWAOrder createBuyOrder(const std::string& asset_id, const std::string& user_address,
                           double amount, double price) {
        RWAOrder order;
        order.order_id = generateOrderId();
        order.asset_id = asset_id;
        order.user_address = user_address;
        order.order_type = RWAOrderType::BUY;
        order.amount = amount;
        order.price = price;
        order.total_value = amount * price;
        order.status = RWAOrderStatus::OPEN;
        order.payment_token = "USDC";
        order.created_at = getCurrentTimestamp();
        order.expires_at = order.created_at + (7 * 24 * 60 * 60); // 7 days
        order.updated_at = order.created_at;
        
        std::lock_guard<std::mutex> lock(orders_mutex);
        orders[user_address].push_back(order);
        
        return order;
    }
    
    // Create sell order
    RWAOrder createSellOrder(const std::string& asset_id, const std::string& user_address,
                            double amount, double price) {
        RWAOrder order;
        order.order_id = generateOrderId();
        order.asset_id = asset_id;
        order.user_address = user_address;
        order.order_type = RWAOrderType::SELL;
        order.amount = amount;
        order.price = price;
        order.total_value = amount * price;
        order.status = RWAOrderStatus::OPEN;
        order.payment_token = "USDC";
        order.created_at = getCurrentTimestamp();
        order.expires_at = order.created_at + (7 * 24 * 60 * 60);
        order.updated_at = order.created_at;
        
        std::lock_guard<std::mutex> lock(orders_mutex);
        orders[user_address].push_back(order);
        
        return order;
    }
    
    // Get user's orders
    std::vector<RWAOrder> getUserOrders(const std::string& user_address) {
        std::lock_guard<std::mutex> lock(orders_mutex);
        auto it = orders.find(user_address);
        if (it != orders.end()) {
            return it->second;
        }
        return {};
    }
    
    // Get open orders for an asset
    std::vector<RWAOrder> getOpenOrders(const std::string& asset_id) {
        std::lock_guard<std::mutex> lock(orders_mutex);
        std::vector<RWAOrder> result;
        for (const auto& pair : orders) {
            for (const auto& order : pair.second) {
                if (order.asset_id == asset_id && order.status == RWAOrderStatus::OPEN) {
                    result.push_back(order);
                }
            }
        }
        return result;
    }
    
    // Cancel order
    bool cancelOrder(const std::string& order_id, const std::string& user_address) {
        std::lock_guard<std::mutex> lock(orders_mutex);
        auto it = orders.find(user_address);
        if (it != orders.end()) {
            for (auto& order : it->second) {
                if (order.order_id == order_id && order.status == RWAOrderStatus::OPEN) {
                    order.status = RWAOrderStatus::CANCELLED;
                    order.updated_at = getCurrentTimestamp();
                    return true;
                }
            }
        }
        return false;
    }
    
    // Fill order (execute trade)
    RWAOrder fillOrder(const std::string& order_id, const std::string& filler_address, double amount) {
        std::lock_guard<std::mutex> lock(orders_mutex);
        
        for (auto& orders_pair : orders) {
            for (auto& order : orders_pair.second) {
                if (order.order_id == order_id && order.status == RWAOrderStatus::OPEN) {
                    double fill_amount = std::min(amount, order.amount - order.filled_amount);
                    order.filled_amount += fill_amount;
                    
                    if (order.filled_amount >= order.amount) {
                        order.status = RWAOrderStatus::FILLED;
                    } else {
                        order.status = RWAOrderStatus::PARTIALLY_FILLED;
                    }
                    
                    order.updated_at = getCurrentTimestamp();
                    
                    // Create transaction record
                    RWATransaction tx;
                    tx.tx_id = generateTxId();
                    tx.order_id = order_id;
                    tx.asset_id = order.asset_id;
                    tx.from_address = order.user_address;
                    tx.to_address = filler_address;
                    tx.amount = fill_amount;
                    tx.price = order.price;
                    tx.total_value = fill_amount * order.price;
                    tx.fee = tx.total_value * 0.0025; // 0.25% fee
                    tx.status = "COMPLETED";
                    tx.timestamp = getCurrentTimestamp();
                    
                    std::lock_guard<std::mutex> tx_lock(transactions_mutex);
                    transactions[filler_address].push_back(tx);
                    
                    return order;
                }
            }
        }
        
        return RWAOrder();
    }
    
    // Get user's holdings
    std::vector<RWAHolding> getUserHoldings(const std::string& user_address) {
        std::lock_guard<std::mutex> lock(holdings_mutex);
        auto it = holdings.find(user_address);
        if (it != holdings.end()) {
            return it->second;
        }
        return {};
    }
    
    // Get transaction history
    std::vector<RWATransaction> getUserTransactions(const std::string& user_address) {
        std::lock_guard<std::mutex> lock(transactions_mutex);
        auto it = transactions.find(user_address);
        if (it != transactions.end()) {
            return it->second;
        }
        return {};
    }
    
    // Get all transactions for an asset
    std::vector<RWATransaction> getAssetTransactions(const std::string& asset_id) {
        std::lock_guard<std::mutex> lock(transactions_mutex);
        std::vector<RWATransaction> result;
        for (const auto& pair : transactions) {
            for (const auto& tx : pair.second) {
                if (tx.asset_id == asset_id) {
                    result.push_back(tx);
                }
            }
        }
        return result;
    }
    
    // Get market statistics for an asset
    std::map<std::string, double> getMarketStats(const std::string& asset_id) {
        std::map<std::string, double> stats;
        
        std::lock_guard<std::mutex> lock(orders_mutex);
        double total_volume = 0;
        double highest_price = 0;
        double lowest_price = 0;
        int order_count = 0;
        
        for (const auto& pair : orders) {
            for (const auto& order : pair.second) {
                if (order.asset_id == asset_id && order.status == RWAOrderStatus::FILLED) {
                    total_volume += order.total_value;
                    if (order.price > highest_price || highest_price == 0) {
                        highest_price = order.price;
                    }
                    if (order.price < lowest_price || lowest_price == 0) {
                        lowest_price = order.price;
                    }
                    order_count++;
                }
            }
        }
        
        stats["totalVolume"] = total_volume;
        stats["highestPrice"] = highest_price;
        stats["lowestPrice"] = lowest_price;
        stats["orderCount"] = order_count;
        
        return stats;
    }
    
    // Search assets
    std::vector<RWAAsset> searchAssets(const std::string& query) {
        std::lock_guard<std::mutex> lock(assets_mutex);
        std::vector<RWAAsset> result;
        std::string lower_query = query;
        std::transform(lower_query.begin(), lower_query.end(), lower_query.begin(), ::tolower);
        
        for (const auto& pair : assets) {
            const RWAAsset& asset = pair.second;
            std::string lower_name = asset.name;
            std::transform(lower_name.begin(), lower_name.end(), lower_name.begin(), ::tolower);
            
            if (lower_name.find(lower_query) != std::string::npos ||
                asset.symbol.find(query) != std::string::npos) {
                result.push_back(asset);
            }
        }
        
        return result;
    }
};

} // namespace tigerwallet

#endif // TIGERWALLET_RWA_SERVICE_HPP
