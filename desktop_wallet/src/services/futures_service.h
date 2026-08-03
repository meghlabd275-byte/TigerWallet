// Futures Trading Service - C++ Desktop Implementation
// Perpetual futures trading

#ifndef FUTURES_SERVICE_H
#define FUTURES_SERVICE_H

#include <string>
#include <vector>
#include <map>
#include <ctime>

namespace tigerwallet {

struct FuturesPair {
    std::string id;
    std::string base;
    std::string quote;
    std::string symbol;
    double price;
    double change24h;
    double volume24h;
    double high24h;
    double low24h;
    double makerFee;
    double takerFee;
    
    FuturesPair() : price(0), change24h(0), volume24h(0), high24h(0), low24h(0), makerFee(0.02), takerFee(0.04) {}
};

struct FuturesPosition {
    std::string id;
    std::string userId;
    std::string symbol;
    std::string side;
    double size;
    double entryPrice;
    double markPrice;
    int leverage;
    double margin;
    std::string marginMode;
    double pnl;
    double pnlPercent;
    double liquidationPrice;
    time_t openTime;
    
    FuturesPosition() : size(0), entryPrice(0), markPrice(0), leverage(1), margin(0), pnl(0), pnlPercent(0), liquidationPrice(0), openTime(0) {}
};

struct FuturesOrder {
    std::string id;
    std::string userId;
    std::string symbol;
    std::string side;
    std::string type;
    double size;
    double price;
    double filled;
    std::string status;
    int leverage;
    std::string marginMode;
    double* stopPrice;
    time_t createTime;
    
    FuturesOrder() : size(0), price(0), filled(0), leverage(1), createTime(0), stopPrice(nullptr) {}
};

class FuturesService {
private:
    std::vector<FuturesPair> pairs_;
    std::map<std::string, std::vector<FuturesPosition>> positions_;
    std::map<std::string, std::vector<FuturesOrder>> orders_;
    
public:
    FuturesService();
    ~FuturesService();
    
    // Initialize pairs
    void initializePairs();
    
    // Get all pairs
    std::vector<FuturesPair> getPairs();
    
    // Get positions
    std::vector<FuturesPosition> getPositions(const std::string& userId);
    
    // Get orders
    std::vector<FuturesOrder> getOrders(const std::string& userId);
    
    // Open position
    FuturesOrder openPosition(const std::string& userId, const std::string& symbol,
                              const std::string& side, double size, double price,
                              int leverage, const std::string& marginMode);
    
    // Close position
    void closePosition(const std::string& userId, const std::string& positionId);
    
    // Place order
    FuturesOrder placeOrder(const std::string& userId, const std::string& symbol,
                           const std::string& side, const std::string& type,
                           double size, double price, int leverage,
                           const std::string& marginMode, double stopPrice = 0);
    
    // Cancel order
    void cancelOrder(const std::string& userId, const std::string& orderId);
    
    // Calculate liquidation
    double calculateLiquidation(double entryPrice, int leverage, const std::string& side);
};

} // namespace tigerwallet

#endif // FUTURES_SERVICE_H
