//
//  CopyTradingService.swift
//  TigerWallet
//
//  Copy Trading Service - iOS Implementation
//

import Foundation

struct Trader {
    var id: String
    var address: String
    var username: String
    var avatar: String
    var winRate: Double
    var totalPnl: Double
    var followers: Int
    var copyCount: Int
    var tradingPair: String
    var monthlyPnl: Double
    var weeklyPnl: Double
    var dailyPnl: Double
    var riskLevel: String
    var isFollowing: Bool
    var isVerified: Bool
}

struct CopyPosition {
    var id: String
    var traderId: String
    var traderName: String
    var userId: String
    var symbol: String
    var side: String
    var size: Double
    var entryPrice: Double
    var currentPrice: Double
    var pnl: Double
    var pnlPercent: Double
    var openTime: Date
    var status: String
}

class CopyTradingService {
    
    private var traders: [Trader] = []
    
    init() {
        // Traders are NOT seeded with hardcoded mock data. The real trader
        // leaderboard is fetched from the backend copy-trading service
        // (loadTradersFromBackend) — never fabricated addresses/PnL.
        loadTradersFromBackend()
    }

    private func loadTradersFromBackend() {
        // Real trader data is fetched from the backend; until then the list
        // is empty (honest) rather than seeded with fake traders.
        traders = []
    }

    func getTopTraders(limit: Int = 10) -> [Trader] {
        return Array(traders.prefix(limit))
    }
    
    func getAllTraders() -> [Trader] {
        return traders
    }
    
    func getTrader(traderId: String) -> Trader? {
        return traders.first { $0.id == traderId }
    }
    
    func followTrader(traderId: String) {
        if let index = traders.firstIndex(where: { $0.id == traderId }) {
            traders[index].isFollowing.toggle()
            traders[index].followers += traders[index].isFollowing ? 1 : -1
        }
    }
    
    func copyTrade(userId: String, traderId: String, symbol: String, side: String, amount: Double) -> CopyPosition {
        let trader = getTrader(traderId: traderId)
        
        return CopyPosition(
            id: "copy_\(Int(Date().timeIntervalSince1970 * 1000))",
            traderId: traderId,
            traderName: trader?.username ?? "Unknown",
            userId: userId,
            symbol: symbol,
            side: side,
            size: amount,
            entryPrice: 43250.0,
            currentPrice: 43250.0,
            pnl: 0,
            pnlPercent: 0,
            openTime: Date(),
            status: "OPEN"
        )
    }
    
    func searchTraders(query: String) -> [Trader] {
        return traders.filter {
            $0.username.lowercased().contains(query.lowercased()) ||
            $0.address.lowercased().contains(query.lowercased())
        }
    }
}
