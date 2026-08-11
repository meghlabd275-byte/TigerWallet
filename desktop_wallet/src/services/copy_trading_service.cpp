// Copy Trading Service Implementation
#include "copy_trading_service.h"
#include <sstream>
#include <algorithm>

namespace tigerwallet {

CopyTradingService::CopyTradingService() {}

CopyTradingService::~CopyTradingService() {}

void CopyTradingService::initializeTraders() {
    // The trader list is sourced from the backend copy-trading service. There
    // is currently no such endpoint on the wallet_api backend, so we start
    // with an EMPTY list rather than seeding fabricated "demo" traders. Never
    // seed mock traders with invented addresses/PnL.
    traders_.clear();
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
    pos.entryPrice = 0.0; // No fabricated price; sourced from the real market via the backend
    pos.currentPrice = 0.0;
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
            // Closing PnL must be computed from the real exit price sourced
            // from the backend/market. Without it we leave the PnL fields
            // unchanged (honest "pending") rather than inventing a random
            // outcome. Final settlement is performed by the backend.
            break;
        }
    }
}

} // namespace tigerwallet
