// P2P Trading Service - C++ Desktop Implementation

#include "p2p_trading_service.h"
#include <algorithm>
#include <chrono>
#include <sstream>
#include <iomanip>
#include <random>

namespace tigerwallet {

static std::string generateUUID() {
    static std::mt19937_64 rng(std::chrono::steady_clock::now().time_since_epoch().count());
    std::uniform_int_distribution<uint64_t> dist;
    std::ostringstream ss;
    ss << std::hex << std::setfill('0') << std::setw(16) << dist(rng) << dist(rng);
    return ss.str();
}

P2PTradingService::P2PTradingService() {}
P2PTradingService::~P2PTradingService() {}

std::vector<P2PAdvert> P2PTradingService::getAdverts(const std::string& token,
                                                      const std::string& fiatCurrency,
                                                      const std::string& side,
                                                      const std::string& paymentMethod) {
    std::vector<P2PAdvert> result;
    auto it = adverts_.find(token);
    if (it == adverts_.end()) return result;
    for (const auto& ad : it->second) {
        if (!fiatCurrency.empty() && ad.fiatCurrency != fiatCurrency) continue;
        if (!side.empty() && ad.side != side) continue;
        if (!paymentMethod.empty() && ad.paymentMethod != paymentMethod) continue;
        result.push_back(ad);
    }
    return result;
}

P2PAdvert P2PTradingService::createAdvert(const std::string& userId, const std::string& side,
                                          const std::string& token, const std::string& fiatCurrency,
                                          const std::string& paymentMethod, double price,
                                          double minAmount, double maxAmount) {
    P2PAdvert ad;
    ad.id = generateUUID();
    ad.userId = userId;
    ad.side = side;
    ad.token = token;
    ad.fiatCurrency = fiatCurrency;
    ad.paymentMethod = paymentMethod;
    ad.price = price;
    ad.minAmount = minAmount;
    ad.maxAmount = maxAmount;
    ad.availableAmount = maxAmount;
    ad.createTime = std::time(nullptr);
    adverts_[token].push_back(ad);
    return ad;
}

P2POrder P2PTradingService::createOrder(const std::string& advertId, const std::string& takerId, double amount) {
    P2POrder order;
    order.id = generateUUID();
    order.advertId = advertId;
    order.takerId = takerId;
    order.amount = amount;
    order.status = "pending";
    order.createTime = std::time(nullptr);

    for (auto& [token, ads] : adverts_) {
        for (auto& ad : ads) {
            if (ad.id == advertId) {
                order.makerId = ad.userId;
                order.side = ad.side;
                order.token = ad.token;
                order.fiatCurrency = ad.fiatCurrency;
                order.paymentMethod = ad.paymentMethod;
                order.price = ad.price;
                order.fiatAmount = amount * ad.price;
                ad.availableAmount -= amount;
                break;
            }
        }
    }

    orders_[takerId].push_back(order);
    return order;
}

void P2PTradingService::markAsPaid(const std::string& userId, const std::string& orderId) {
    auto it = orders_.find(userId);
    if (it == orders_.end()) return;
    for (auto& order : it->second) {
        if (order.id == orderId) {
            order.status = "paid";
            return;
        }
    }
}

void P2PTradingService::releaseCrypto(const std::string& userId, const std::string& orderId) {
    for (auto& [uid, orders] : orders_) {
        for (auto& order : orders) {
            if (order.id == orderId && (order.makerId == userId || order.takerId == userId)) {
                order.status = "released";
                return;
            }
        }
    }
}

void P2PTradingService::cancelOrder(const std::string& userId, const std::string& orderId) {
    auto it = orders_.find(userId);
    if (it == orders_.end()) return;
    for (auto& order : it->second) {
        if (order.id == orderId) {
            order.status = "cancelled";
            for (auto& [token, ads] : adverts_) {
                for (auto& ad : ads) {
                    if (ad.id == order.advertId) {
                        ad.availableAmount += order.amount;
                        break;
                    }
                }
            }
            return;
        }
    }
}

std::vector<P2POrder> P2PTradingService::getOrders(const std::string& userId) {
    auto it = orders_.find(userId);
    if (it == orders_.end()) return {};
    return it->second;
}

} // namespace tigerwallet
