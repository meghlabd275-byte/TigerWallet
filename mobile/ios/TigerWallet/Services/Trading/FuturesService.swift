//
//  FuturesService.swift
//  TigerWallet
//
//  Futures Trading Service - iOS Implementation
//

import Foundation

struct FuturesPair {
    var id: String
    var base: String
    var quote: String
    var symbol: String
    var price: Double
    var change24h: Double
    var volume24h: Double
    var high24h: Double
    var low24h: Double
    var makerFee: Double
    var takerFee: Double
}

struct FuturesPosition {
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

struct FuturesOrder {
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
    var createTime: Date
}

class FuturesService {
    
    private let bases = ["BTC", "ETH", "BNB", "SOL", "XRP", "DOGE", "ADA", "AVAX", "DOT", "LINK", "MATIC", "LTC", "UNI", "ATOM", "XLM", "NEAR", "APT", "ARB", "OP", "INJ"]
    private let prices: [String: Double] = [
        "BTC": 43250.0, "ETH": 2280.0, "BNB": 312.5, "SOL": 98.75, "XRP": 0.62,
        "DOGE": 0.082, "ADA": 0.58, "AVAX": 38.2, "DOT": 7.85, "LINK": 14.50,
        "MATIC": 0.92, "LTC": 72.30, "UNI": 6.25, "ATOM": 10.45, "XLM": 0.125,
        "NEAR": 3.25, "APT": 9.80, "ARB": 1.12, "OP": 2.45, "INJ": 35.50
    ]
    
    func getPairs() -> [FuturesPair] {
        var pairs: [FuturesPair] = []
        let quotes = ["USDT", "USDC"]
        
        for (index, base) in bases.enumerated() {
            for quote in quotes {
                guard base != quote else { continue }
                
                let price = prices[base] ?? 10.0
                let pair = FuturesPair(
                    id: "futures_\(index)",
                    base: base,
                    quote: quote,
                    symbol: "\(base)/\(quote)",
                    price: price,
                    change24h: Double.random(in: -5...5),
                    volume24h: price * 1000000,
                    high24h: price * 1.05,
                    low24h: price * 0.95,
                    makerFee: 0.02,
                    takerFee: 0.04
                )
                pairs.append(pair)
            }
        }
        return pairs
    }
    
    func openPosition(userId: String, symbol: String, side: String, size: Double, price: Double, leverage: Int, marginMode: String) -> FuturesOrder {
        return FuturesOrder(
            id: "futures_order_\(Int(Date().timeIntervalSince1970 * 1000))",
            userId: userId,
            symbol: symbol,
            side: side,
            type: "MARKET",
            size: size,
            price: price,
            filled: 0,
            status: "PENDING",
            leverage: leverage,
            marginMode: marginMode,
            createTime: Date()
        )
    }
    
    func placeOrder(userId: String, symbol: String, side: String, type: String, size: Double, price: Double, leverage: Int, marginMode: String) -> FuturesOrder {
        return FuturesOrder(
            id: "futures_order_\(Int(Date().timeIntervalSince1970 * 1000))",
            userId: userId,
            symbol: symbol,
            side: side,
            type: type,
            size: size,
            price: price,
            filled: 0,
            status: "PENDING",
            leverage: leverage,
            marginMode: marginMode,
            createTime: Date()
        )
    }
    
    func cancelOrder(order: inout FuturesOrder) {
        order.status = "CANCELLED"
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
