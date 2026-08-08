// Margin Trading Service Implementation
#include "margin_trading_service.h"
#include <sstream>
#include <iomanip>
#include <cmath>
#include <algorithm>
#include <cstdlib>

namespace tigerwallet {

MarginTradingService::MarginTradingService() {
    initializePairs();
}

MarginTradingService::~MarginTradingService() {}

void MarginTradingService::initializePairs() {
    pairs_.clear();
    
    std::vector<std::string> bases = {"BTC", "ETH", "BNB", "SOL", "XRP", "DOGE", "ADA", "AVAX", "DOT", "LINK"};
    std::map<std::string, double> prices = {
        {"BTC", 43250.0}, {"ETH", 2280.0}, {"BNB", 312.5}, {"SOL", 98.75}, {"XRP", 0.62},
        {"DOGE", 0.082}, {"ADA", 0.58}, {"AVAX", 38.2}, {"DOT", 7.85}, {"LINK", 14.50}
    };
    
    for (size_t i = 0; i < bases.size(); i++) {
        MarginPair pair;
        pair.id = "margin_" + std::to_string(i);
        pair.base = bases[i];
        pair.quote = "USDT";
        pair.symbol = bases[i] + "/USDT";
        pair.price = prices[bases[i]];
        pair.change24h = ((rand() % 100) - 50) / 10.0;
        pair.volume24h = prices[bases[i]] * 1000000;
        pair.borrowable = prices[bases[i]] * 50000000;
        pair.interestRate = 0.0001;
        pair.isActive = true;
        pairs_.push_back(pair);
    }
}

std::vector<MarginPair> MarginTradingService::getPairs() {
    return pairs_;
}

MarginAccount MarginTradingService::getAccount(const std::string& userId) {
    if (accounts_.find(userId) != accounts_.end()) {
        return accounts_[userId];
    }
    
    MarginAccount account;
    account.userId = userId;
    account.totalAssets = 50000.0;
    account.totalLiabilities = 5000.0;
    account.netAssets = 45000.0;
    account.availableBalance = 40000.0;
    account.totalBorrowed = 5000.0;
    account.marginRatio = 9.0;
    account.riskLevel = "SAFE";
    accounts_[userId] = account;
    return account;
}

std::vector<MarginPosition> MarginTradingService::getPositions(const std::string& userId) {
    if (positions_.find(userId) != positions_.end()) {
        return positions_[userId];
    }
    return std::vector<MarginPosition>();
}

std::vector<MarginOrder> MarginTradingService::getOrders(const std::string& userId) {
    if (orders_.find(userId) != orders_.end()) {
        return orders_[userId];
    }
    return std::vector<MarginOrder>();
}

MarginOrder MarginTradingService::openPosition(const std::string& userId, const std::string& symbol,
                                                const std::string& side, double size, double price,
                                                int leverage, const std::string& marginMode) {
    MarginOrder order;
    std::stringstream ss;
    ss << "margin_order_" << time(nullptr);
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

void MarginTradingService::closePosition(const std::string& userId, const std::string& positionId) {
    auto& userPositions = positions_[userId];
    userPositions.erase(
        std::remove_if(userPositions.begin(), userPositions.end(),
            [&positionId](const MarginPosition& pos) { return pos.id == positionId; }),
        userPositions.end()
    );
}

MarginOrder MarginTradingService::placeOrder(const std::string& userId, const std::string& symbol,
                                              const std::string& side, const std::string& type,
                                              double size, double price, int leverage,
                                              const std::string& marginMode) {
    MarginOrder order;
    std::stringstream ss;
    ss << "margin_order_" << time(nullptr) << "_" << (rand() % 10000);
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

void MarginTradingService::cancelOrder(const std::string& userId, const std::string& orderId) {
    auto& userOrders = orders_[userId];
    for (auto& order : userOrders) {
        if (order.id == orderId) {
            order.status = "CANCELLED";
            break;
        }
    }
}

void MarginTradingService::borrow(const std::string& userId, const std::string& symbol, double amount) {
    // Borrow implementation
}

void MarginTradingService::repay(const std::string& userId, const std::string& borrowId) {
    // Repay implementation
}

double MarginTradingService::calculateLiquidationPrice(double entryPrice, int leverage,
                                                        const std::string& side, double margin) {
    double liquidationPercent = 1.0 / leverage;
    if (side == "LONG") {
        return entryPrice * (1 - liquidationPercent);
    } else {
        return entryPrice * (1 + liquidationPercent);
    }
}

double MarginTradingService::calculatePnL(double entryPrice, double closePrice,
                                          double size, const std::string& side) {
    if (side == "LONG") {
        return (closePrice - entryPrice) * size;
    } else {
        return (entryPrice - closePrice) * size;
    }
}

} // namespace tigerwallet
