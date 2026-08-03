//
//  ConvertService.swift
//  TigerWallet
//
//  Convert Service - iOS Implementation
//

import Foundation

struct ConvertToken {
    var symbol: String
    var name: String
    var balance: Double
    var icon: String
}

struct ConvertPair {
    var from: String
    var to: String
    var rate: Double
    var inverseRate: Double
    var fee: Double
}

struct ConvertOrder {
    var id: String
    var fromToken: String
    var toToken: String
    var fromAmount: Double
    var toAmount: Double
    var rate: Double
    var fee: Double
    var status: String
    var timestamp: Date
}

class ConvertService {
    
    private var rates: [String: Double] = [:]
    
    init() {
        initializeRates()
    }
    
    private func initializeRates() {
        rates["BTC_USDT"] = 43250.0
        rates["ETH_USDT"] = 2280.0
        rates["BNB_USDT"] = 312.5
        rates["SOL_USDT"] = 98.75
        rates["XRP_USDT"] = 0.62
        rates["DOGE_USDT"] = 0.082
        rates["ADA_USDT"] = 0.58
        rates["BTC_ETH"] = 18.97
    }
    
    func getTokens() -> [ConvertToken] {
        return [
            ConvertToken(symbol: "BTC", name: "Bitcoin", balance: 1.5, icon: "₿"),
            ConvertToken(symbol: "ETH", name: "Ethereum", balance: 15.0, icon: "Ξ"),
            ConvertToken(symbol: "USDT", name: "Tether", balance: 50000.0, icon: "₮"),
            ConvertToken(symbol: "USDC", name: "USD Coin", balance: 25000.0, icon: "$"),
            ConvertToken(symbol: "BNB", name: "BNB", balance: 50.0, icon: "B"),
            ConvertToken(symbol: "SOL", name: "Solana", balance: 150.0, icon: "S"),
            ConvertToken(symbol: "XRP", name: "Ripple", balance: 10000.0, icon: "X"),
            ConvertToken(symbol: "ADA", name: "Cardano", balance: 5000.0, icon: "A"),
            ConvertToken(symbol: "DOGE", name: "Dogecoin", balance: 100000.0, icon: "D"),
            ConvertToken(symbol: "AVAX", name: "Avalanche", balance: 200.0, icon: "A")
        ]
    }
    
    func getRate(from: String, to: String) -> Double {
        if from == to { return 1.0 }
        
        let key = "\(from)_\(to)"
        if let rate = rates[key] { return rate }
        
        // Try through USDT
        if let fromToUsdt = rates["\(from)_USDT"],
           let toFromUsdt = rates["\(to)_USDT"] {
            return fromToUsdt / toFromUsdt
        }
        
        return 1.0
    }
    
    func convert(userId: String, from: String, to: String, amount: Double) -> ConvertOrder {
        let rate = getRate(from: from, to: to)
        return ConvertOrder(
            id: "convert_\(Int(Date().timeIntervalSince1970 * 1000))",
            fromToken: from,
            toToken: to,
            fromAmount: amount,
            toAmount: amount * rate,
            rate: rate,
            fee: amount * 0.001,
            status: "COMPLETED",
            timestamp: Date()
        )
    }
}
