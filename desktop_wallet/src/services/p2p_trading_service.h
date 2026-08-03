// P2P Trading Service - C++ Desktop Implementation

#ifndef P2P_TRADING_SERVICE_H
#define P2P_TRADING_SERVICE_H

#include <string>
#include <vector>
#include <map>
#include <ctime>

namespace tigerwallet {

struct P2PAdvert {
    std::string id;
    std::string userId;
    std::string username;
    std::string avatar;
    std::string side;
    std::string token;
    std::string fiatCurrency;
    std::string paymentMethod;
    double price;
    double minAmount;
    double maxAmount;
    double availableAmount;
    int ordersCompleted;
    double completionRate;
    double avgReleaseTime;
    bool isOnline;
    time_t createTime;
    
    P2PAdvert() : price(0), minAmount(0), maxAmount(0), availableAmount(0), 
                  ordersCompleted(0), completionRate(0), avgReleaseTime(0), isOnline(false), createTime(0) {}
};

struct P2POrder {
    std::string id;
    std::string advertId;
    std::string makerId;
    std::string takerId;
    std::string side;
    std::string token;
    std::string fiatCurrency;
    std::string paymentMethod;
    double price;
    double amount;
    double fiatAmount;
    std::string status;
    time_t createTime;
    
    P2POrder() : price(0), amount(0), fiatAmount(0), createTime(0) {}
};

class P2PTradingService {
private:
    std::map<std::string, std::vector<P2PAdvert>> adverts_;
    std::map<std::string, std::vector<P2POrder>> orders_;
    
public:
    P2PTradingService();
    ~P2PTradingService();
    
    // Get adverts
    std::vector<P2PAdvert> getAdverts(const std::string& token, const std::string& fiatCurrency,
                                      const std::string& side, const std::string& paymentMethod);
    
    // Create advert
    P2PAdvert createAdvert(const std::string& userId, const std::string& side,
                          const std::string& token, const std::string& fiatCurrency,
                          const std::string& paymentMethod, double price,
                          double minAmount, double maxAmount);
    
    // Create order
    P2POrder createOrder(const std::string& advertId, const std::string& takerId, double amount);
    
    // Mark as paid
    void markAsPaid(const std::string& userId, const std::string& orderId);
    
    // Release crypto
    void releaseCrypto(const std::string& userId, const std::string& orderId);
    
    // Cancel order
    void cancelOrder(const std::string& userId, const std::string& orderId);
    
    // Get orders
    std::vector<P2POrder> getOrders(const std::string& userId);
};

} // namespace tigerwallet

#endif // P2P_TRADING_SERVICE_H
