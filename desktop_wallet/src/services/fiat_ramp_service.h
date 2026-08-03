// Fiat On-Ramp Service - C++ Desktop Implementation

#ifndef FIAT_RAMP_SERVICE_H
#define FIAT_RAMP_SERVICE_H

#include <string>
#include <vector>
#include <map>
#include <ctime>

namespace tigerwallet {

struct FiatProvider {
    std::string id;
    std::string name;
    std::string logo;
    std::vector<std::string> supportedFiat;
    std::vector<std::string> supportedCrypto;
    double minAmount;
    double maxAmount;
    double feePercent;
    std::string processingTime;
    bool isAvailable;
    
    FiatProvider() : minAmount(0), maxAmount(0), feePercent(0), isAvailable(false) {}
};

struct FiatOrder {
    std::string id;
    std::string userId;
    std::string providerId;
    std::string providerName;
    std::string side;
    std::string fiatCurrency;
    std::string cryptoCurrency;
    double fiatAmount;
    double cryptoAmount;
    double exchangeRate;
    double fee;
    std::string paymentMethod;
    std::string status;
    std::string walletAddress;
    std::string txHash;
    time_t createTime;
    
    FiatOrder() : fiatAmount(0), cryptoAmount(0), exchangeRate(0), fee(0), createTime(0) {}
};

class FiatRampService {
private:
    std::vector<FiatProvider> providers_;
    std::map<std::string, std::vector<FiatOrder>> orders_;
    
public:
    FiatRampService();
    ~FiatRampService();
    
    // Initialize providers
    void initializeProviders();
    
    // Get providers
    std::vector<FiatProvider> getProviders();
    
    // Get provider by ID
    FiatProvider getProvider(const std::string& providerId);
    
    // Calculate exchange rate
    double calculateRate(const std::string& providerId, const std::string& fiatCurrency,
                        const std::string& cryptoCurrency, double fiatAmount);
    
    // Create buy order
    FiatOrder createBuyOrder(const std::string& userId, const std::string& providerId,
                            const std::string& fiatCurrency, const std::string& cryptoCurrency,
                            double fiatAmount, const std::string& paymentMethod,
                            const std::string& walletAddress);
    
    // Create sell order
    FiatOrder createSellOrder(const std::string& userId, const std::string& providerId,
                             const std::string& fiatCurrency, const std::string& cryptoCurrency,
                             double cryptoAmount, const std::string& paymentMethod);
    
    // Get orders
    std::vector<FiatOrder> getOrders(const std::string& userId);
    
    // Get order
    FiatOrder* getOrder(const std::string& userId, const std::string& orderId);
    
    // Confirm payment
    void confirmPayment(const std::string& userId, const std::string& orderId);
    
    // Complete order
    void completeOrder(const std::string& userId, const std::string& orderId, const std::string& txHash);
    
    // Cancel order
    void cancelOrder(const std::string& userId, const std::string& orderId);
};

} // namespace tigerwallet

#endif // FIAT_RAMP_SERVICE_H
