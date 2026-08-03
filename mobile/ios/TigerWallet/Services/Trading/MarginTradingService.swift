//
//  MarginTradingService.swift
//  TigerWallet
//
//  Margin Trading Service - iOS Implementation
//

import Foundation

struct MarginPair {
    var id: String
    var base: String
    var quote: String
    var symbol: String
    var price: Double
    var change24h: Double
    var volume24h: Double
    var borrowable: Double
    var interestRate: Double
    var isActive: Bool
}

struct MarginPosition {
    var id: String
    var userId: String
    var symbol: String
    var side: String
    var size: Double
    var entryPrice: Double
    var markPrice: Double
    var leverage: Int
    var margin: Double
    var pnl: Double
    var pnlPercent: Double
    var liquidationPrice: Double
    var marginMode: String
    var openTime: Date
}

struct MarginOrder {
    var id: String
    var userId: String
    var symbol: String
    var side: String
    var type: String
    var size: Double
    var price: Double
    var filled: Double
    var status: String
    var leverage: Int
    var marginMode: String
}

struct MarginAccount {
    var userId: String
    var totalAssets: Double
    var totalLiabilities: Double
    var netAssets: Double
    var availableBalance: Double
    var totalBorrowed: Double
    var marginRatio: Double
    var riskLevel: String
}

class MarginTradingService {
    
    func generatePairs() -> [MarginPair] {
        let bases = ["BTC", "ETH", "BNB", "SOL", "XRP", "DOGE", "ADA", "AVAX", "DOT", "LINK"]
        let prices: [String: Double] = [
            "BTC": 43250.0, "ETH": 2280.0, "BNB": 312.5, "SOL": 98.75, "XRP": 0.62,
            "DOGE": 0.082, "ADA": 0.58, "AVAX": 38.2, "DOT": 7.85, "LINK": 14.50
        ]
        
        var pairs: [MarginPair] = []
        for (index, base) in bases.enumerated() {
            let price = prices[base] ?? 10.0
            let pair = MarginPair(
                id: "margin_\(index)",
                base: base,
                quote: "USDT",
                symbol: "\(base)/USDT",
                price: price,
                change24h: Double.random(in: -5...5),
                volume24h: price * 1000000,
                borrowable: price * 50000000,
                interestRate: 0.0001,
                isActive: true
            )
            pairs.append(pair)
        }
        return pairs
    }
    
    func getAccount(userId: String) -> MarginAccount {
        return MarginAccount(
            userId: userId,
            totalAssets: 50000.0,
            totalLiabilities: 5000.0,
            netAssets: 45000.0,
            availableBalance: 40000.0,
            totalBorrowed: 5000.0,
            marginRatio: 9.0,
            riskLevel: "SAFE"
        )
    }
    
    func openPosition(userId: String, symbol: String, side: String, size: Double, price: Double, leverage: Int, marginMode: String) -> MarginOrder {
        return MarginOrder(
            id: "margin_order_\(Int(Date().timeIntervalSince1970 * 1000))",
            userId: userId,
            symbol: symbol,
            side: side,
            type: "MARKET",
            size: size,
            price: price,
            filled: 0,
            status: "PENDING",
            leverage: leverage,
            marginMode: marginMode
        )
    }
    
    func calculateLiquidationPrice(entryPrice: Double, leverage: Int, side: String) -> Double {
        let liquidationPercent = 1.0 / Double(leverage)
        if side == "LONG" {
            return entryPrice * (1 - liquidationPercent)
        } else {
            return entryPrice * (1 + liquidationPercent)
        }
    }
    
    func calculatePnL(entryPrice: Double, closePrice: Double, size: Double, side: String) -> Double {
        if side == "LONG" {
            return (closePrice - entryPrice) * size
        } else {
            return (entryPrice - closePrice) * size
        }
    }
}
