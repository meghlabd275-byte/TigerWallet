// Futures Service Implementation
#include "futures_service.h"
#include <sstream>
#include <algorithm>
#include <cmath>

namespace tigerwallet {

FuturesService::FuturesService() {
    initializePairs();
}

FuturesService::~FuturesService() {}

void FuturesService::initializePairs() {
    std::vector<std::string> bases = {
        "BTC", "ETH", "BNB", "SOL", "XRP", "DOGE", "ADA", "AVAX", "DOT", "LINK",
        "MATIC", "LTC", "UNI", "ATOM", "XLM", "NEAR", "APT", "ARB", "OP", "INJ"
    };
    
    std::map<std::string, double> prices = {
        {"BTC", 43250.0}, {"ETH", 2280.0}, {"BNB", 312.5}, {"SOL", 98.75}, {"XRP", 0.62},
        {"DOGE", 0.082}, {"ADA", 0.58}, {"AVAX", 38.2}, {"DOT", 7.85}, {"LINK", 14.50},
        {"MATIC", 0.92}, {"LTC", 72.30}, {"UNI", 6.25}, {"ATOM", 10.45}, {"XLM", 0.125},
        {"NEAR", 3.25}, {"APT", 9.80}, {"ARB", 1.12}, {"OP", 2.45}, {"INJ", 35.50}
    };
    
    for (size_t i = 0; i < bases.size(); i++) {
        FuturesPair pair;
        pair.id = "futures_" + std::to_string(i);
        pair.base = bases[i];
        pair.quote = "USDT";
        pair.symbol = bases[i] + "/USDT";
        pair.price = prices[bases[i]];
        pair.change24h = ((rand() % 100) - 50) / 10.0;
        pair.volume24h = prices[bases[i]] * 1000000;
        pair.high24h = prices[bases[i]] * 1.05;
        pair.low24h = prices[bases[i]] * 0.95;
        pair.makerFee = 0.02;
        pair.takerFee = 0.04;
        pairs_.push_back(pair);
    }
}

std::vector<FuturesPair> FuturesService::getPairs() {
    return pairs_;
}

std::vector<FuturesPosition> FuturesService::getPositions(const std::string& userId) {
    if (positions_.find(userId) != positions_.end()) {
        return positions_[userId];
    }
    return std::vector<FuturesPosition>();
}

std::vector<FuturesOrder> FuturesService::getOrders(const std::string& userId) {
    if (orders_.find(userId) != orders_.end()) {
        return orders_[userId];
    }
    return std::vector<FuturesOrder>();
}

FuturesOrder FuturesService::openPosition(const std::string& userId, const std::string& symbol,
                                         const std::string& side, double size, double price,
                                         int leverage, const std::string& marginMode) {
    FuturesOrder order;
    std::stringstream ss;
    ss << "futures_order_" << time(nullptr);
    order.id = ss.str();
    order.userId = userId;
    order.symbol = symbol;
    order.side = side;
    order.type = "MARKET";
    order.size = size;
    order.price = price;
    order.filled = 0;
    order.status = "PENDING";
    order.leverage = leverage;
    order.marginMode = marginMode;
    order.createTime = time(nullptr);
    
    orders_[userId].push_back(order);
    return order;
}

void FuturesService::closePosition(const std::string& userId, const std::string& positionId) {
    auto& userPositions = positions_[userId];
    userPositions.erase(
        std::remove_if(userPositions.begin(), userPositions.end(),
            [&positionId](const FuturesPosition& pos) { return pos.id == positionId; }),
        userPositions.end()
    );
}

FuturesOrder FuturesService::placeOrder(const std::string& userId, const std::string& symbol,
                                       const std::string& side, const std::string& type,
                                       double size, double price, int leverage,
                                       const std::string& marginMode, double stopPrice) {
    FuturesOrder order;
    std::stringstream ss;
    ss << "futures_order_" << time(nullptr) << "_" << (rand() % 10000);
    order.id = ss.str();
    order.userId = userId;
    order.symbol = symbol;
    order.side = side;
    order.type = type;
    order.size = size;
    order.price = price;
    order.filled = 0;
    order.status = "PENDING";
    order.leverage = leverage;
    order.marginMode = marginMode;
    order.createTime = time(nullptr);
    
    orders_[userId].push_back(order);
    return order;
}

void FuturesService::cancelOrder(const std::string& userId, const std::string& orderId) {
    auto& userOrders = orders_[userId];
    for (auto& order : userOrders) {
        if (order.id == orderId) {
            order.status = "CANCELLED";
            break;
        }
    }
}

double FuturesService::calculateLiquidation(double entryPrice, int leverage, const std::string& side) {
    double liquidationPercent = 1.0 / leverage;
    if (side == "LONG") {
        return entryPrice * (1 - liquidationPercent);
    } else {
        return entryPrice * (1 + liquidationPercent);
    }
}

} // namespace tigerwallet
