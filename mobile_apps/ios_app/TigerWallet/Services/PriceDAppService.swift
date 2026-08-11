//
//  PriceService.swift
//  TigerWallet
//
//  Real-time Price Service for iOS
//

import Foundation
import Combine

class PriceService {
    static let shared = PriceService()
    
    private var webSocket: URLSessionWebSocketTask?
    private var reconnectAttempts = 0
    private let maxReconnectAttempts = 5
    private var priceSubject = PassthroughSubject<PriceUpdate, Never>()
    
    var pricePublisher: AnyPublisher<PriceUpdate, Never> {
        priceSubject.eraseToAnyPublisher()
    }
    
    private init() {}
    
    func getPrices(tokens: [String] = ["BTC", "ETH", "SOL", "BNB", "MATIC", "AVAX"]) async throws -> [String: PriceData] {
        let url = URL(string: "http://localhost:8443/api/v1/prices")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body = try JSONEncoder().encode(tokens)
        request.httpBody = body
        
        let (data, _) = try await URLSession.shared.data(for: request)
        return try JSONDecoder().decode([String: PriceData].self, from: data)
    }
    
    func getHistoricalPrices(token: String, timeframe: String = "24h") async throws -> [PricePoint] {
        let url = URL(string: "http://localhost:8443/api/v1/prices/\(token)/history?timeframe=\(timeframe)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([PricePoint].self, from: data)
    }
    
    func connectWebSocket() {
        guard pollTimer == nil else { return }
        let interval: TimeInterval = 15
        func poll() {
            Task { [weak self] in
                guard let self = self else { return }
                do {
                    let url = URL(string: "http://localhost:8443/api/v1/prices")!
                    let (data, _) = try await URLSession.shared.data(from: url)
                    if let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                       let prices = object["prices"] as? [String: Any] ?? object as? [String: Any] {
                        for (token, info) in prices {
                            let price: Double; let change: Double
                            if let n = info as? Double { price = n; change = 0 }
                            else if let d = info as? [String: Any] {
                                price = (d["price"] as? Double) ?? 0
                                change = (d["change_24h"] as? Double) ?? (d["change24h"] as? Double) ?? 0
                            } else { continue }
                            self.priceSubject.send(PriceUpdate(token: token, price: price, change24h: change))
                        }
                        self.reconnectAttempts = 0
                    }
                } catch { print("Price poll failed: \(error)"); self.reconnect() }
            }
        }
        poll()
        let timer = DispatchSource.timer(flags: [])
        timer.schedule(deadline: .now() + interval, repeating: interval)
        timer.setEventHandler(handler: poll)
        timer.resume()
        pollTimer = timer
    }
    private var pollTimer: DispatchSourceTimer?
    private func receiveMessage() {
        // no-op: polling replaces WebSocket
    }
    
    private func reconnect() {
        guard reconnectAttempts < maxReconnectAttempts else { return }
        
        reconnectAttempts += 1
        DispatchQueue.main.asyncAfter(deadline: .now() + Double(reconnectAttempts)) { [weak self] in
            self?.webSocket = nil
            self?.connectWebSocket()
        }
    }
    
    func disconnect() {
        webSocket?.cancel(with: .goingAway, reason: nil)
        webSocket = nil
    }
    
    func subscribe(to token: String) {
        let message = ["action": "subscribe", "token": token]
        if let data = try? JSONEncoder().encode(message),
           let string = String(data: data, encoding: .utf8) {
            webSocket?.send(.string(string)) { _ in }
        }
    }
}

// DApp Browser Service
class DAppBrowserService {
    static let shared = DAppBrowserService()
    private let baseURL = "http://localhost:8443/api/v1/dapps"
    
    private init() {}
    
    func getDApps(category: String? = nil) async throws -> [DApp] {
        var urlString = baseURL
        if let category = category {
            urlString += "?category=\(category)"
        }
        
        let url = URL(string: urlString)!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([DApp].self, from: data)
    }
    
    func searchDApps(query: String) async throws -> [DApp] {
        let url = URL(string: "\(baseURL)/search?q=\(query)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([DApp].self, from: data)
    }
    
    func getCategories() async throws -> [String] {
        let url = URL(string: "\(baseURL)/categories")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([String].self, from: data)
    }
}

struct DApp: Codable, Identifiable {
    let id: String
    let name: String
    let description: String?
    let url: String
    let logoUrl: String?
    let category: String
    let chains: [String]
}

struct PriceUpdate: Codable {
    let token: String
    let price: Double
    let change24h: Double
    let timestamp: Int
}

struct PriceData: Codable {
    let price: Double
    let change24h: Double
    let high24h: Double
    let low24h: Double
    let volume24h: Double
    let marketCap: Double
}

struct PricePoint: Codable {
    let timestamp: Int
    let price: Double
}
