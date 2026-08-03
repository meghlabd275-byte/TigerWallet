// Fiat Ramp Service Implementation
#include "fiat_ramp_service.h"
#include <sstream>

namespace tigerwallet {

FiatRampService::FiatRampService() {
    initializeProviders();
}

FiatRampService::~FiatRampService() {}

void FiatRampService::initializeProviders() {
    providers_.clear();
    
    // MoonPay
    FiatProvider p1;
    p1.id = "provider_1";
    p1.name = "MoonPay";
    p1.logo = "🌙";
    p1.supportedFiat = {"USD", "EUR", "GBP", "AUD", "CAD"};
    p1.supportedCrypto = {"BTC", "ETH", "USDT", "BNB", "SOL"};
    p1.minAmount = 30;
    p1.maxAmount = 50000;
    p1.feePercent = 2.5;
    p1.processingTime = "5-30 minutes";
    p1.isAvailable = true;
    providers_.push_back(p1);
    
    // Simplex
    FiatProvider p2;
    p2.id = "provider_2";
    p2.name = "Simplex";
    p2.logo = "💳";
    p2.supportedFiat = {"USD", "EUR", "GBP"};
    p2.supportedCrypto = {"BTC", "ETH", "USDT"};
    p2.minAmount = 50;
    p2.maxAmount = 25000;
    p2.feePercent = 3.5;
    p2.processingTime = "10-60 minutes";
    p2.isAvailable = true;
    providers_.push_back(p2);
    
    // Transak
    FiatProvider p3;
    p3.id = "provider_3";
    p3.name = "Transak";
    p3.logo = "🔄";
    p3.supportedFiat = {"USD", "EUR", "GBP", "INR"};
    p3.supportedCrypto = {"BTC", "ETH", "USDT", "MATIC", "AVAX"};
    p3.minAmount = 20;
    p3.maxAmount = 100000;
    p3.feePercent = 2.0;
    p3.processingTime = "15-45 minutes";
    p3.isAvailable = true;
    providers_.push_back(p3);
    
    // OnRamper
    FiatProvider p4;
    p4.id = "provider_4";
    p4.name = "OnRamper";
    p4.logo = "📱";
    p4.supportedFiat = {"USD", "EUR", "GBP", "AUD"};
    p4.supportedCrypto = {"BTC", "ETH", "USDT", "ADA", "DOT"};
    p4.minAmount = 25;
    p4.maxAmount = 75000;
    p4.feePercent = 1.8;
    p4.processingTime = "5-20 minutes";
    p4.isAvailable = true;
    providers_.push_back(p4);
}

std::vector<FiatProvider> FiatRampService::getProviders() {
    return providers_;
}

FiatProvider FiatRampService::getProvider(const std::string& providerId) {
    for (const auto& p : providers_) {
        if (p.id == providerId) return p;
    }
    return providers_[0];
}

double FiatRampService::calculateRate(const std::string& providerId, const std::string& fiatCurrency,
                                      const std::string& cryptoCurrency, double fiatAmount) {
    FiatProvider provider = getProvider(providerId);
    double fee = fiatAmount * (provider.feePercent / 100.0);
    double netAmount = fiatAmount - fee;
    
    double baseRates[] = {43250.0, 2280.0, 1.0, 312.5, 98.75, 0.58, 0.92};
    std::string cryptos[] = {"BTC", "ETH", "USDT", "BNB", "SOL", "ADA", "MATIC"};
    
    double baseRate = 1.0;
    for (int i = 0; i < 7; i++) {
        if (cryptoCurrency == cryptos[i]) {
            baseRate = baseRates[i];
            break;
        }
    }
    
    return netAmount / baseRate;
}

FiatOrder FiatRampService::createBuyOrder(const std::string& userId, const std::string& providerId,
                                          const std::string& fiatCurrency, const std::string& cryptoCurrency,
                                          double fiatAmount, const std::string& paymentMethod,
                                          const std::string& walletAddress) {
    FiatProvider provider = getProvider(providerId);
    
    FiatOrder order;
    std::stringstream ss;
    ss << "fiat_order_" << time(nullptr);
    order.id = ss.str();
    order.userId = userId;
    order.providerId = providerId;
    order.providerName = provider.name;
    order.side = "BUY";
    order.fiatCurrency = fiatCurrency;
    order.cryptoCurrency = cryptoCurrency;
    order.fiatAmount = fiatAmount;
    order.exchangeRate = calculateRate(providerId, fiatCurrency, cryptoCurrency, fiatAmount);
    order.cryptoAmount = (fiatAmount * (1 - provider.feePercent/100)) / order.exchangeRate;
    order.fee = fiatAmount * (provider.feePercent / 100);
    order.paymentMethod = paymentMethod;
    order.status = "PENDING";
    order.walletAddress = walletAddress;
    order.createTime = time(nullptr);
    
    orders_[userId].push_back(order);
    return order;
}

FiatOrder FiatRampService::createSellOrder(const std::string& userId, const std::string& providerId,
                                           const std::string& fiatCurrency, const std::string& cryptoCurrency,
                                           double cryptoAmount, const std::string& paymentMethod) {
    FiatProvider provider = getProvider(providerId);
    
    double baseRates[] = {43250.0, 2280.0, 1.0, 312.5, 98.75, 0.58, 0.92};
    std::string cryptos[] = {"BTC", "ETH", "USDT", "BNB", "SOL", "ADA", "MATIC"};
    double baseRate = 1.0;
    for (int i = 0; i < 7; i++) {
        if (cryptoCurrency == cryptos[i]) {
            baseRate = baseRates[i];
            break;
        }
    }
    
    double fiatAmount = cryptoAmount * baseRate * (1 - provider.feePercent/100);
    
    FiatOrder order;
    std::stringstream ss;
    ss << "fiat_order_" << time(nullptr);
    order.id = ss.str();
    order.userId = userId;
    order.providerId = providerId;
    order.providerName = provider.name;
    order.side = "SELL";
    order.fiatCurrency = fiatCurrency;
    order.cryptoCurrency = cryptoCurrency;
    order.fiatAmount = fiatAmount;
    order.cryptoAmount = cryptoAmount;
    order.exchangeRate = baseRate;
    order.fee = fiatAmount * (provider.feePercent / 100);
    order.paymentMethod = paymentMethod;
    order.status = "PENDING";
    order.createTime = time(nullptr);
    
    orders_[userId].push_back(order);
    return order;
}

std::vector<FiatOrder> FiatRampService::getOrders(const std::string& userId) {
    if (orders_.find(userId) != orders_.end()) {
        return orders_[userId];
    }
    return std::vector<FiatOrder>();
}

FiatOrder* FiatRampService::getOrder(const std::string& userId, const std::string& orderId) {
    if (orders_.find(userId) != orders_.end()) {
        for (auto& order : orders_[userId]) {
            if (order.id == orderId) return &order;
        }
    }
    return nullptr;
}

void FiatRampService::confirmPayment(const std::string& userId, const std::string& orderId) {
    if (orders_.find(userId) != orders_.end()) {
        for (auto& order : orders_[userId]) {
            if (order.id == orderId) {
                order.status = "AWAITING_CONFIRMATION";
                break;
            }
        }
    }
}

void FiatRampService::completeOrder(const std::string& userId, const std::string& orderId, const std::string& txHash) {
    if (orders_.find(userId) != orders_.end()) {
        for (auto& order : orders_[userId]) {
            if (order.id == orderId) {
                order.status = "COMPLETED";
                order.txHash = txHash;
                break;
            }
        }
    }
}

void FiatRampService::cancelOrder(const std::string& userId, const std::string& orderId) {
    if (orders_.find(userId) != orders_.end()) {
        for (auto& order : orders_[userId]) {
            if (order.id == orderId) {
                order.status = "CANCELLED";
                break;
            }
        }
    }
}

} // namespace tigerwallet
