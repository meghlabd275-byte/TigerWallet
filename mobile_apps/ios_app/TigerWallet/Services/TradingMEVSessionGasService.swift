//
//  TradingService.swift
//  TigerWallet
//
//  Complete Trading Service - Order Book, Charts, Positions
//

import Foundation

class TradingService {
    static let shared = TradingService()
    private let baseURL = "http://localhost:8443/api/v1/trading"
    
    private init() {}
    
    // Order Book
    struct OrderBook: Codable {
        let bids: [[Double]]
        let asks: [[Double]]
        let timestamp: Int64
        let symbol: String
    }
    
    func getOrderBook(symbol: String, limit: Int = 50) async throws -> OrderBook? {
        let url = URL(string: "\(baseURL)/orderbook?symbol=\(symbol)&limit=\(limit)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode(OrderBook.self, from: data)
    }
    
    // Candlesticks
    struct Candlestick: Codable {
        let timestamp: Int64
        let open: Double
        let high: Double
        let low: Double
        let close: Double
        let volume: Double
    }
    
    func getCandlesticks(symbol: String, interval: String = "1h", limit: Int = 100) async throws -> [Candlestick] {
        let url = URL(string: "\(baseURL)/klines?symbol=\(symbol)&interval=\(interval)&limit=\(limit)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([Candlestick].self, from: data)
    }
    
    // Positions
    struct Position: Codable, Identifiable {
        let id: String
        let symbol: String
        let side: String
        let amount: Double
        let entryPrice: Double
        let currentPrice: Double
        let unrealizedPnl: Double
        let leverage: Int
        let liquidationPrice: Double
        let margin: Double
    }
    
    func getPositions(walletAddress: String) async throws -> [Position] {
        let url = URL(string: "\(baseURL)/positions/\(walletAddress)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([Position].self, from: data)
    }
    
    // Open Orders
    struct OpenOrder: Codable, Identifiable {
        let id: String
        let symbol: String
        let side: String
        let type: String
        let price: Double
        let amount: Double
        let filledAmount: Double
        let status: String
        let createdAt: Int64
    }
    
    func getOpenOrders(walletAddress: String) async throws -> [OpenOrder] {
        let url = URL(string: "\(baseURL)/orders/\(walletAddress)?status=open")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([OpenOrder].self, from: data)
    }
    
    // Place Order
    func placeMarketOrder(walletAddress: String, symbol: String, side: String, amount: Double, leverage: Int = 1) async throws -> String {
        var request = URLRequest(url: URL(string: "\(baseURL)/orders")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "symbol": symbol,
            "side": side,
            "type": "market",
            "amount": amount,
            "leverage": leverage
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    func placeLimitOrder(walletAddress: String, symbol: String, side: String, price: Double, amount: Double, leverage: Int = 1) async throws -> String {
        var request = URLRequest(url: URL(string: "\(baseURL)/orders")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "symbol": symbol,
            "side": side,
            "type": "limit",
            "price": price,
            "amount": amount,
            "leverage": leverage
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    func cancelOrder(walletAddress: String, orderId: String) async throws -> Bool {
        var request = URLRequest(url: URL(string: "\(baseURL)/orders/\(orderId)")!)
        request.httpMethod = "DELETE"
        
        let (_, response) = try await URLSession.shared.data(for: request)
        return (response as? HTTPURLResponse)?.statusCode == 200
    }
    
    func closePosition(walletAddress: String, positionId: String) async throws -> String {
        var request = URLRequest(url: URL(string: "\(baseURL)/positions/\(positionId)/close")!)
        request.httpMethod = "POST"
        
        let (data, _) = try await URLSession.shared.data(for: request)
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
}

// MEV Protection
class MEVProtectionService {
    static let shared = MEVProtectionService()
    
    struct SandwichDetection: Codable {
        let detected: Bool
        let frontRunTx: String?
        let backRunTx: String?
        let profit: Double?
        let severity: String
    }
    
    func detectSandwichAttack(txHash: String) async throws -> SandwichDetection {
        let url = URL(string: "http://localhost:8443/api/v1/mev/detect-sandwich?tx=\(txHash)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode(SandwichDetection.self, from: data)
    }
    
    struct SimulationResult: Codable {
        let success: Bool
        let gasUsed: Int64
        let error: String?
    }
    
    func simulateTransaction(from: String, to: String, data: String, value: String) async throws -> SimulationResult {
        var request = URLRequest(url: URL(string: "http://localhost:8443/api/v1/mev/simulate")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["from": from, "to": to, "data": data, "value": value]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        return try JSONDecoder().decode(SimulationResult.self, from: data)
    }
    
    func submitWithProtection(signedTx: String, protectionLevel: String = "medium") async throws -> String {
        var request = URLRequest(url: URL(string: "http://localhost:8443/api/v1/mev/submit")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["signed_tx": signedTx, "protection_level": protectionLevel]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
}

// Session Keys
class SessionKeysService {
    static let shared = SessionKeysService()
    
    struct SessionKey: Codable, Identifiable {
        let id: String
        let key: String
        let dapp: String
        let permissions: [String]
        let expiresAt: Int64
        let createdAt: Int64
    }
    
    func generateSessionKey(walletAddress: String, dappUrl: String, permissions: [String], expiresIn: Int64 = 86400) async throws -> SessionKey {
        var request = URLRequest(url: URL(string: "http://localhost:8443/api/v1/session-keys")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "dapp_url": dappUrl,
            "permissions": permissions,
            "expires_in": expiresIn
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        return try JSONDecoder().decode(SessionKey.self, from: data)
    }
    
    func getSessionKeys(walletAddress: String) async throws -> [SessionKey] {
        let url = URL(string: "http://localhost:8443/api/v1/session-keys/\(walletAddress)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([SessionKey].self, from: data)
    }
    
    func revokeSessionKey(walletAddress: String, sessionKeyId: String) async throws -> Bool {
        var request = URLRequest(url: URL(string: "http://localhost:8443/api/v1/session-keys/\(sessionKeyId)")!
        request.httpMethod = "DELETE"
        
        let (_, response) = try await URLSession.shared.data(for: request)
        return (response as? HTTPURLResponse)?.statusCode == 200
    }
}

// Gas Optimization
class GasOptimizationService {
    static let shared = GasOptimizationService()
    
    struct GasPrice: Codable {
        let slow: Int64
        let standard: Int64
        let fast: Int64
        let instant: Int64
    }
    
    func getGasPrices(chain: String = "ethereum") async throws -> GasPrice? {
        let url = URL(string: "http://localhost:8443/api/v1/gas/prices?chain=\(chain)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode(GasPrice.self, from: data)
    }
    
    struct OptimizationSuggestion: Codable {
        let type: String
        let potentialSavings: Double
        let recommendation: String
    }
    
    func getOptimizationSuggestions(from: String, to: String, data: String) async throws -> [OptimizationSuggestion] {
        var request = URLRequest(url: URL(string: "http://localhost:8443/api/v1/gas/optimize")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = ["from": from, "to": to, "data": data]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        return try JSONDecoder().decode([OptimizationSuggestion].self, from: data)
    }
}

struct TransactionResult: Codable {
    let txHash: String
}
