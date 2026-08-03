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
        initializeTraders()
    }
    
    private func initializeTraders() {
        traders = [
            Trader(id: "trader_0", address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E", username: "TraderAlex", avatar: "🐋", winRate: 78.5, totalPnl: 125000, followers: 5420, copyCount: 2710, tradingPair: "BTC/USDT", monthlyPnl: 31250, weeklyPnl: 7500, dailyPnl: 1250, riskLevel: "MEDIUM", isFollowing: false, isVerified: true),
            Trader(id: "trader_1", address: "0x1234567890abcdef1234567890abcdef12345678", username: "CryptoKing", avatar: "👑", winRate: 72.3, totalPnl: 98500, followers: 3210, copyCount: 1605, tradingPair: "ETH/USDT", monthlyPnl: 24625, weeklyPnl: 5910, dailyPnl: 985, riskLevel: "HIGH", isFollowing: true, isVerified: true),
            Trader(id: "trader_2", address: "0xabcdef1234567890abcdef1234567890abcdef12", username: "DeFiMaster", avatar: "🎯", winRate: 85.0, totalPnl: 150000, followers: 8930, copyCount: 4465, tradingPair: "SOL/USDT", monthlyPnl: 37500, weeklyPnl: 9000, dailyPnl: 1500, riskLevel: "LOW", isFollowing: false, isVerified: true),
            Trader(id: "trader_3", address: "0x9876543210fedcba9876543210fedcba98765432", username: "AltSeason", avatar: "🚀", winRate: 65.0, totalPnl: 87000, followers: 1890, copyCount: 945, tradingPair: "XRP/USDT", monthlyPnl: 21750, weeklyPnl: 5220, dailyPnl: 870, riskLevel: "HIGH", isFollowing: true, isVerified: false),
            Trader(id: "trader_4", address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", username: "BitcoinWhale", avatar: "🐋", winRate: 82.0, totalPnl: 200000, followers: 6540, copyCount: 3270, tradingPair: "BTC/USDT", monthlyPnl: 50000, weeklyPnl: 12000, dailyPnl: 2000, riskLevel: "MEDIUM", isFollowing: false, isVerified: true)
        ]
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
