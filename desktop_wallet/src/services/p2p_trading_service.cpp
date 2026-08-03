// P2P Trading Service Implementation
#include "p2p_trading_service.h"
#include <sstream>
#include <random>

namespace tigerwallet {

P2PTradingService::P2PTradingService() {}
P2PTradingService::~P2PTradingService() {}

std::vector<P2PAdvert> P2PTradingService::getAdverts(const std::string& token, 
                                                      const std::string& fiatCurrency,
                                                      const std::string& side,
                                                      const std::string& paymentMethod) {
    std::vector<P2PAdvert> results;
    
    // Generate mock adverts
    std::vector<std::string> users = {"CryptoTrader1", "BitSeller", "FastTrade", "P2PPro", "SecureDeal"};
    std::vector<std::string> avatars = {"🧑‍💼", "👨‍💻", "⚡", "🎯", "🔒"};
    std::vector<bool> online = {true, true, false, true, true};
    std::vector<std::string> payments = {"Bank Transfer", "PayPal", "AliPay", "WeChat Pay", "UPI"};
    
    double basePrices[] = {1.0, 43250.0, 2280.0, 1.0, 312.5};
    std::string tokens[] = {"USDT", "BTC", "ETH", "USDC", "BNB"};
    std::string fiats[] = {"USD", "EUR", "GBP", "CNY", "INR"};
    
    for (size_t i = 0; i < users.size(); i++) {
        P2PAdvert advert;
        std::stringstream ss;
        ss << "advert_" << i;
        advert.id = ss.str();
        advert.userId = "user_" + std::to_string(i);
        advert.username = users[i];
        advert.avatar = avatars[i];
        advert.side = side;
        advert.token = token.empty() ? tokens[i % 5] : token;
        advert.fiatCurrency = fiatCurrency.empty() ? fiats[i % 5] : fiatCurrency;
        advert.paymentMethod = payments[i];
        
        double priceVariation = (rand() % 10 - 5) / 1000.0;
        advert.price = basePrices[i % 5] * (1 + priceVariation);
        advert.minAmount = 10.0;
        advert.maxAmount = 5000.0;
        advert.availableAmount = basePrices[i % 5] * 10;
        advert.ordersCompleted = 50 + i * 10;
        advert.completionRate = 95.0 + (i % 5);
        advert.avgReleaseTime = 2.0 + (i % 10);
        advert.isOnline = online[i];
        advert.createTime = time(nullptr) - i * 86400;
        
        results.push_back(advert);
    }
    
    return results;
}

P2PAdvert P2PTradingService::createAdvert(const std::string& userId, const std::string& side,
                                         const std::string& token, const std::string& fiatCurrency,
                                         const std::string& paymentMethod, double price,
                                         double minAmount, double maxAmount) {
    P2PAdvert advert;
    std::stringstream ss;
    ss << "advert_" << time(nullptr);
    advert.id = ss.str();
    advert.userId = userId;
    advert.username = "MyUser";
    advert.avatar = "👤";
    advert.side = side;
    advert.token = token;
    advert.fiatCurrency = fiatCurrency;
    advert.paymentMethod = paymentMethod;
    advert.price = price;
    advert.minAmount = minAmount;
    advert.maxAmount = maxAmount;
    advert.availableAmount = maxAmount;
    advert.ordersCompleted = 0;
    advert.completionRate = 100.0;
    advert.avgReleaseTime = 5.0;
    advert.isOnline = true;
    advert.createTime = time(nullptr);
    
    adverts_[userId].push_back(advert);
    return advert;
}

P2POrder P2PTradingService::createOrder(const std::string& advertId, const std::string& takerId, double amount) {
    P2POrder order;
    std::stringstream ss;
    ss << "order_" << time(nullptr);
    order.id = ss.str();
    order.advertId = advertId;
    order.takerId = takerId;
    order.side = "BUY";
    order.token = "USDT";
    order.fiatCurrency = "USD";
    order.paymentMethod = "Bank Transfer";
    order.price = 1.0;
    order.amount = amount;
    order.fiatAmount = amount;
    order.status = "PENDING";
    order.createTime = time(nullptr);
    
    orders_[takerId].push_back(order);
    return order;
}

void P2PTradingService::markAsPaid(const std::string& userId, const std::string& orderId) {
    auto& userOrders = orders_[userId];
    for (auto& order : userOrders) {
        if (order.id == orderId) {
            order.status = "PAID";
            break;
        }
    }
}

void P2PTradingService::releaseCrypto(const std::string& userId, const std::string& orderId) {
    auto& userOrders = orders_[userId];
    for (auto& order : userOrders) {
        if (order.id == orderId) {
            order.status = "COMPLETED";
            break;
        }
    }
}

void P2PTradingService::cancelOrder(const std::string& userId, const std::string& orderId) {
    auto& userOrders = orders_[userId];
    for (auto& order : userOrders) {
        if (order.id == orderId) {
            order.status = "CANCELLED";
            break;
        }
    }
}

std::vector<P2POrder> P2PTradingService::getOrders(const std::string& userId) {
    if (orders_.find(userId) != orders_.end()) {
        return orders_[userId];
    }
    return std::vector<P2POrder>();
}

} // namespace tigerwallet
