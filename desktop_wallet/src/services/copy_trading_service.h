// Copy Trading Service - C++ Desktop Implementation
// Follow expert traders

#ifndef COPY_TRADING_SERVICE_H
#define COPY_TRADING_SERVICE_H

#include <string>
#include <vector>
#include <map>
#include <ctime>
#include <utility>

namespace tigerwallet {

struct Trader {
    std::string id;
    std::string address;
    std::string username;
    std::string avatar;
    double winRate;
    double totalPnL;
    double pnlPercent;
    int followers;
    int copyCount;
    std::string tradingPair;
    double monthlyPnL;
    double weeklyPnL;
    double dailyPnL;
    double maxDrawdown;
    std::string avgHoldingTime;
    std::string riskLevel;
    bool isFollowing;
    bool isVerified;
    
    Trader() : winRate(0), totalPnL(0), pnlPercent(0), followers(0), copyCount(0),
               monthlyPnL(0), weeklyPnL(0), dailyPnL(0), maxDrawdown(0), isFollowing(false), isVerified(false) {}
    Trader(std::string address_, std::string username_, std::string avatar_,
           double winRate_, double totalPnL_, double pnlPercent_, int followers_, int copyCount_,
           std::string tradingPair_, double monthlyPnL_, double weeklyPnL_, double dailyPnL_,
           double maxDrawdown_, std::string riskLevel_, bool isFollowing_, bool isVerified_)
        : address(std::move(address_)), username(std::move(username_)), avatar(std::move(avatar_)),
          winRate(winRate_), totalPnL(totalPnL_), pnlPercent(pnlPercent_),
          followers(followers_), copyCount(copyCount_), tradingPair(std::move(tradingPair_)),
          monthlyPnL(monthlyPnL_), weeklyPnL(weeklyPnL_), dailyPnL(dailyPnL_),
          maxDrawdown(maxDrawdown_), riskLevel(std::move(riskLevel_)),
          isFollowing(isFollowing_), isVerified(isVerified_) {}
};

struct CopyPosition {
    std::string id;
    std::string traderId;
    std::string traderName;
    std::string userId;
    std::string symbol;
    std::string side;
    double size;
    double entryPrice;
    double currentPrice;
    double pnl;
    double pnlPercent;
    time_t openTime;
    std::string status;
    
    CopyPosition() : size(0), entryPrice(0), currentPrice(0), pnl(0), pnlPercent(0), openTime(0) {}
};

struct CopySettings {
    std::string userId;
    double copyAmount;
    int copyLeverage;
    double stopLossPercent;
    double takeProfitPercent;
    bool autoCopy;
    
    CopySettings() : copyAmount(1000), copyLeverage(1), stopLossPercent(10), takeProfitPercent(20), autoCopy(true) {}
};

class CopyTradingService {
private:
    std::vector<Trader> traders_;
    std::map<std::string, std::vector<CopyPosition>> positions_;
    std::map<std::string, std::vector<Trader>> following_;
    std::map<std::string, CopySettings> settings_;
    
public:
    CopyTradingService();
    ~CopyTradingService();
    
    // Initialize mock traders
    void initializeTraders();
    
    // Get top traders
    std::vector<Trader> getTopTraders(int limit = 10);
    
    // Search traders
    std::vector<Trader> searchTraders(const std::string& query);
    
    // Get following traders
    std::vector<Trader> getFollowing(const std::string& userId);
    
    // Follow trader
    void followTrader(const std::string& userId, const std::string& traderId);
    
    // Unfollow trader
    void unfollowTrader(const std::string& userId, const std::string& traderId);
    
    // Get copy positions
    std::vector<CopyPosition> getCopyPositions(const std::string& userId);
    
    // Copy trade
    CopyPosition copyTrade(const std::string& userId, const std::string& traderId,
                          const std::string& symbol, const std::string& side, double amount);
    
    // Get settings
    CopySettings getSettings(const std::string& userId);
    
    // Update settings
    void updateSettings(const std::string& userId, const CopySettings& settings);
    
    // Close copy position
    void closeCopyPosition(const std::string& userId, const std::string& positionId);
};

} // namespace tigerwallet

#endif // COPY_TRADING_SERVICE_H
