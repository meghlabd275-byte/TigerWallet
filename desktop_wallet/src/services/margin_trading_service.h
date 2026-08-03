// Margin Trading Service - C++ Desktop Implementation
// Supports Cross/Isolated Margin, Long/Short, Leverage 1-125x

#ifndef MARGIN_TRADING_SERVICE_H
#define MARGIN_TRADING_SERVICE_H

#include <string>
#include <vector>
#include <map>
#include <ctime>

namespace tigerwallet {

struct MarginPair {
    std::string id;
    std::string base;
    std::string quote;
    std::string symbol;
    double price;
    double change24h;
    double volume24h;
    double borrowable;
    double interestRate;
    bool isActive;
    
    MarginPair() : price(0), change24h(0), volume24h(0), borrowable(0), interestRate(0), isActive(true) {}
};

struct MarginPosition {
    std::string id;
    std::string userId;
    std::string symbol;
    std::string side;
    double size;
    double entryPrice;
    double markPrice;
    int leverage;
    double margin;
    double pnl;
    double pnlPercent;
    double liquidationPrice;
    std::string marginMode;
    time_t openTime;
    
    MarginPosition() : size(0), entryPrice(0), markPrice(0), leverage(1), margin(0), pnl(0), pnlPercent(0), liquidationPrice(0), openTime(0) {}
};

struct MarginOrder {
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
    time_t createTime;
    
    MarginOrder() : size(0), price(0), filled(0), leverage(1), createTime(0) {}
};

struct MarginAccount {
    std::string userId;
    double totalAssets;
    double totalLiabilities;
    double netAssets;
    double availableBalance;
    double totalBorrowed;
    double marginRatio;
    std::string riskLevel;
    
    MarginAccount() : totalAssets(0), totalLiabilities(0), netAssets(0), availableBalance(0), totalBorrowed(0), marginRatio(0) {}
};

class MarginTradingService {
private:
    std::map<std::string, std::vector<MarginPosition>> positions_;
    std::map<std::string, std::vector<MarginOrder>> orders_;
    std::map<std::string, MarginAccount> accounts_;
    std::vector<MarginPair> pairs_;
    
public:
    MarginTradingService();
    ~MarginTradingService();
    
    // Initialize trading pairs
    void initializePairs();
    
    // Get all trading pairs
    std::vector<MarginPair> getPairs();
    
    // Get account info
    MarginAccount getAccount(const std::string& userId);
    
    // Get open positions
    std::vector<MarginPosition> getPositions(const std::string& userId);
    
    // Get order history
    std::vector<MarginOrder> getOrders(const std::string& userId);
    
    // Open a position
    MarginOrder openPosition(const std::string& userId, const std::string& symbol,
                            const std::string& side, double size, double price,
                            int leverage, const std::string& marginMode);
    
    // Close a position
    void closePosition(const std::string& userId, const std::string& positionId);
    
    // Place an order
    MarginOrder placeOrder(const std::string& userId, const std::string& symbol,
                          const std::string& side, const std::string& type,
                          double size, double price, int leverage,
                          const std::string& marginMode);
    
    // Cancel an order
    void cancelOrder(const std::string& userId, const std::string& orderId);
    
    // Borrow funds
    void borrow(const std::string& userId, const std::string& symbol, double amount);
    
    // Repay funds
    void repay(const std::string& userId, const std::string& borrowId);
    
    // Calculate liquidation price
    double calculateLiquidationPrice(double entryPrice, int leverage,
                                    const std::string& side, double margin);
    
    // Calculate PnL
    double calculatePnL(double entryPrice, double closePrice,
                       double size, const std::string& side);
};

} // namespace tigerwallet

#endif // MARGIN_TRADING_SERVICE_H
