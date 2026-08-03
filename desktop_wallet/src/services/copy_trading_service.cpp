// Copy Trading Service Implementation
#include "copy_trading_service.h"
#include <sstream>
#include <algorithm>
#include <cstdlib>

namespace tigerwallet {

CopyTradingService::CopyTradingService() {
    initializeTraders();
}

CopyTradingService::~CopyTradingService() {}

void CopyTradingService::initializeTraders() {
    traders_ = {
        {"0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E", "TraderAlex", "", 0.78, 45.5, 12.3, 5420, 1250, "BTC/USDT", 12.3, 3.5, 1.2, 15.2, "MEDIUM", false, true},
        {"0x1234567890abcdef1234567890abcdef12345678", "CryptoKing", "", 0.72, 32.8, 8.5, 3210, 890, "ETH/USDT", 8.5, 2.1, 0.8, 12.5, "HIGH", true, true},
        {"0xabcdef1234567890abcdef1234567890abcdef12", "DeFiMaster", "", 0.85, 68.2, 15.7, 8930, 2100, "SOL/USDT", 15.7, 4.2, 1.5, 18.3, "LOW", false, true},
        {"0x9876543210fedcba9876543210fedcba98765432", "AltSeason", "", 0.65, 18.3, 4.2, 1890, 560, "XRP/USDT", 4.2, 1.1, 0.5, 22.1, "HIGH", true, false},
        {"0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", "BitcoinWhale", "", 0.82, 52.1, 11.8, 6540, 1800, "BTC/USDT", 11.8, 3.2, 1.1, 14.8, "MEDIUM", false, true}
    };
    
    for (size_t i = 0; i < traders_.size(); i++) {
        traders_[i].id = "trader_" + std::to_string(i);
    }
}

std::vector<Trader> CopyTradingService::getTopTraders(int limit) {
    std::vector<Trader> sorted = traders_;
    std::sort(sorted.begin(), sorted.end(), [](const Trader& a, const Trader& b) {
        return a.totalPnL > b.totalPnL;
    });
    if (limit > 0 && limit < (int)sorted.size()) {
        sorted.resize(limit);
    }
    return sorted;
}

std::vector<Trader> CopyTradingService::searchTraders(const std::string& query) {
    std::vector<Trader> results;
    for (const auto& trader : traders_) {
        if (trader.username.find(query) != std::string::npos ||
            trader.address.find(query) != std::string::npos) {
            results.push_back(trader);
        }
    }
    return results;
}

std::vector<Trader> CopyTradingService::getFollowing(const std::string& userId) {
    if (following_.find(userId) != following_.end()) {
        return following_[userId];
    }
    return std::vector<Trader>();
}

void CopyTradingService::followTrader(const std::string& userId, const std::string& traderId) {
    for (auto& trader : traders_) {
        if (trader.id == traderId) {
            trader.followers++;
            trader.isFollowing = true;
            following_[userId].push_back(trader);
            break;
        }
    }
}

void CopyTradingService::unfollowTrader(const std::string& userId, const std::string& traderId) {
    for (auto& trader : traders_) {
        if (trader.id == traderId) {
            trader.followers--;
            trader.isFollowing = false;
            break;
        }
    }
    
    auto& userFollowing = following_[userId];
    userFollowing.erase(
        std::remove_if(userFollowing.begin(), userFollowing.end(),
            [&traderId](const Trader& t) { return t.id == traderId; }),
        userFollowing.end()
    );
}

std::vector<CopyPosition> CopyTradingService::getCopyPositions(const std::string& userId) {
    if (positions_.find(userId) != positions_.end()) {
        return positions_[userId];
    }
    return std::vector<CopyPosition>();
}

CopyPosition CopyTradingService::copyTrade(const std::string& userId, const std::string& traderId,
                                           const std::string& symbol, const std::string& side, double amount) {
    CopyPosition pos;
    std::stringstream ss;
    ss << "copy_" << time(nullptr);
    pos.id = ss.str();
    pos.traderId = traderId;
    pos.userId = userId;
    pos.symbol = symbol;
    pos.side = side;
    pos.size = amount;
    pos.entryPrice = 43250.0; // Mock price
    pos.currentPrice = pos.entryPrice;
    pos.pnl = 0;
    pos.pnlPercent = 0;
    pos.openTime = time(nullptr);
    pos.status = "OPEN";
    
    // Find trader name
    for (const auto& trader : traders_) {
        if (trader.id == traderId) {
            pos.traderName = trader.username;
            break;
        }
    }
    
    positions_[userId].push_back(pos);
    return pos;
}

CopySettings CopyTradingService::getSettings(const std::string& userId) {
    if (settings_.find(userId) != settings_.end()) {
        return settings_[userId];
    }
    CopySettings defaultSettings;
    defaultSettings.userId = userId;
    settings_[userId] = defaultSettings;
    return defaultSettings;
}

void CopyTradingService::updateSettings(const std::string& userId, const CopySettings& settings) {
    settings_[userId] = settings;
}

void CopyTradingService::closeCopyPosition(const std::string& userId, const std::string& positionId) {
    auto& userPositions = positions_[userId];
    for (auto& pos : userPositions) {
        if (pos.id == positionId) {
            pos.status = "CLOSED";
            pos.currentPrice = pos.entryPrice * (1 + ((rand() % 100 - 50) / 1000.0));
            pos.pnl = (pos.currentPrice - pos.entryPrice) * pos.size;
            pos.pnlPercent = ((pos.currentPrice - pos.entryPrice) / pos.entryPrice) * 100;
            break;
        }
    }
}

} // namespace tigerwallet
