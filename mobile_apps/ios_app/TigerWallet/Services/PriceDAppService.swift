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
        let url = URL(string: "https://api.tigerwallet.com/v1/prices")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body = try JSONEncoder().encode(tokens)
        request.httpBody = body
        
        let (data, _) = try await URLSession.shared.data(for: request)
        return try JSONDecoder().decode([String: PriceData].self, from: data)
    }
    
    func getHistoricalPrices(token: String, timeframe: String = "24h") async throws -> [PricePoint] {
        let url = URL(string: "https://api.tigerwallet.com/v1/prices/\(token)/history?timeframe=\(timeframe)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([PricePoint].self, from: data)
    }
    
    func connectWebSocket() {
        guard webSocket == nil else { return }
        
        let url = URL(string: "wss://api.tigerwallet.com/ws/prices")!
        webSocket = URLSession.shared.webSocketTask(with: url)
        
        webSocket?.resume()
        receiveMessage()
        
        webSocket?.resume()
    }
    
    private func receiveMessage() {
        webSocket?.receive { [weak self] result in
            switch result {
            case .success(let message):
                switch message {
                case .string(let text):
                    if let data = text.data(using: .utf8),
                       let priceUpdate = try? JSONDecoder().decode(PriceUpdate.self, from: data) {
                        self?.priceSubject.send(priceUpdate)
                    }
                case .data(let data):
                    if let priceUpdate = try? JSONDecoder().decode(PriceUpdate.self, from: data) {
                        self?.priceSubject.send(priceUpdate)
                    }
                @unknown default:
                    break
                }
                self?.receiveMessage()
                
            case .failure(let error):
                print("WebSocket error: \(error)")
                self?.reconnect()
            }
        }
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
    private let baseURL = "https://api.tigerwallet.com/v1/dapps"
    
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
