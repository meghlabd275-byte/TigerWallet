// Convert Service Implementation
#include "convert_service.h"
#include <sstream>
#include <algorithm>

namespace tigerwallet {

ConvertService::ConvertService() {
    initialize();
}

ConvertService::~ConvertService() {}

void ConvertService::initialize() {
    // Initialize tokens
    tokens_ = {
        {"BTC", "Bitcoin", 1.5, "₿"},
        {"ETH", "Ethereum", 15.0, "Ξ"},
        {"USDT", "Tether", 50000.0, "₮"},
        {"USDC", "USD Coin", 25000.0, "$"},
        {"BNB", "BNB", 50.0, "B"},
        {"SOL", "Solana", 150.0, "S"},
        {"XRP", "Ripple", 10000.0, "X"},
        {"ADA", "Cardano", 5000.0, "A"},
        {"DOGE", "Dogecoin", 100000.0, "D"},
        {"AVAX", "Avalanche", 200.0, "A"}
    };
    
    // Initialize pairs
    pairs_ = {
        {"BTC", "USDT", 43250.0, 0.00002312, 0.1, true},
        {"ETH", "USDT", 2280.0, 0.0004386, 0.1, true},
        {"BNB", "USDT", 312.5, 0.0032, 0.1, true},
        {"SOL", "USDT", 98.75, 0.01013, 0.1, true},
        {"XRP", "USDT", 0.62, 1.6129, 0.1, true},
        {"BTC", "ETH", 18.97, 0.0527, 0.1, true},
        {"ETH", "BTC", 0.0527, 18.97, 0.1, true}
    };
}

std::vector<ConvertToken> ConvertService::getTokens() {
    return tokens_;
}

double ConvertService::getRate(const std::string& from, const std::string& to) {
    if (from == to) return 1.0;
    
    for (const auto& pair : pairs_) {
        if (pair.from == from && pair.to == to) {
            return pair.rate;
        }
    }
    
    // Try through USDT
    double fromToUsdt = 1.0;
    double toFromUsdt = 1.0;
    
    for (const auto& pair : pairs_) {
        if (pair.from == from && pair.to == "USDT") {
            fromToUsdt = pair.rate;
        }
        if (pair.from == to && pair.to == "USDT") {
            toFromUsdt = pair.rate;
        }
    }
    
    if (fromToUsdt != 1.0 && toFromUsdt != 1.0) {
        return fromToUsdt / toFromUsdt;
    }
    
    return 1.0;
}

double ConvertService::getBalance(const std::string& userId, const std::string& symbol) {
    if (balances_.find(userId) != balances_.end()) {
        if (balances_[userId].find(symbol) != balances_[userId].end()) {
            return balances_[userId][symbol];
        }
    }
    return 0.0;
}

ConvertOrder ConvertService::convert(const std::string& userId, const std::string& from,
                                     const std::string& to, double amount) {
    ConvertOrder order;
    std::stringstream ss;
    ss << "convert_" << time(nullptr);
    order.id = ss.str();
    order.userId = userId;
    order.fromToken = from;
    order.toToken = to;
    order.fromAmount = amount;
    order.rate = getRate(from, to);
    order.toAmount = amount * order.rate;
    order.fee = amount * 0.001;
    order.status = "COMPLETED";
    order.createTime = time(nullptr);
    
    // Update balances
    balances_[userId][from] -= amount;
    balances_[userId][to] += order.toAmount;
    
    orders_[userId].push_back(order);
    return order;
}

std::vector<ConvertOrder> ConvertService::getHistory(const std::string& userId) {
    if (orders_.find(userId) != orders_.end()) {
        return orders_[userId];
    }
    return std::vector<ConvertOrder>();
}

std::vector<ConvertPair> ConvertService::getPopularPairs() {
    return pairs_;
}

} // namespace tigerwallet
