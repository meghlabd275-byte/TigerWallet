/**
 * WebSocketService - iOS Implementation
 * Real-time connection for Master Wallet
 */

import Foundation
import Starscream

public class WebSocketService: NSObject {
    
    // MARK: - Singleton
    public static let shared = WebSocketService()
    
    // MARK: - Properties
    private var socket: WebSocket?
    private var walletId: String?
    private var authToken: String?
    private var reconnectAttempts = 0
    private var heartbeatTimer: Timer?
    
    // Connection state
    public enum ConnectionState {
        case disconnected
        case connecting
        case connected
        case reconnecting
        case error
    }
    
    public var connectionState: ConnectionState = .disconnected
    
    // Callbacks
    public var onStateChange: ((ConnectionState) -> Void)?
    public var onMessage: ((String) -> Void)?
    public var onBalanceUpdate: ((BalanceUpdate) -> Void)?
    public var onTransactionUpdate: ((TransactionUpdate) -> Void)?
    
    // MARK: - Constants
    private let WS_BASE = "ws://localhost:8450"
    private let RECONNECT_DELAY: TimeInterval = 5.0
    private let MAX_RECONNECT_ATTEMPTS = 10
    
    // MARK: - Initialization
    private override init() {
        super.init()
    }
    
    // MARK: - Connection
    
    /// Connect to WebSocket server
    public func connect(walletId: String, token: String?) {
        self.walletId = walletId
        self.authToken = token
        connectionState = .connecting
        onStateChange?(.connecting)

        guard let url = buildWebSocketURL(walletId: walletId, token: token) else {
            connectionState = .error
            onStateChange?(.error)
            return
        }

        var request = URLRequest(url: url)
        request.timeoutInterval = 30

        socket = WebSocket(request: request)
        socket?.delegate = self
        socket?.connect()
    }

    private func buildWebSocketURL(walletId: String, token: String?) -> URL? {
        var components = URLComponents(string: "\(WS_BASE)/ws")
        var queryItems: [URLQueryItem] = [
            URLQueryItem(name: "master_wallet_id", value: walletId)
        ]
        if let token = token, !token.isEmpty {
            queryItems.append(URLQueryItem(name: "token", value: token))
        }
        components?.queryItems = queryItems
        return components?.url
    }
    
    /// Disconnect from server
    public func disconnect() {
        stopHeartbeat()
        socket?.disconnect()
        socket = nil
        connectionState = .disconnected
        onStateChange?(.disconnected)
    }
    
    // MARK: - Subscriptions
    
    /// Subscribe to balance updates
    public func subscribeToBalance(chainId: Int) {
        sendMessage(type: "subscribe", channel: "balance", data: ["chainId": chainId])
    }
    
    /// Unsubscribe from balance updates
    public func unsubscribeFromBalance(chainId: Int) {
        sendMessage(type: "unsubscribe", channel: "balance", data: ["chainId": chainId])
    }
    
    /// Subscribe to transaction updates
    public func subscribeToTransactions(address: String) {
        sendMessage(type: "subscribe", channel: "transactions", data: ["address": address])
    }
    
    /// Subscribe to ticker updates
    public func subscribeToTicker(pair: String) {
        sendMessage(type: "subscribe", channel: "ticker", data: ["pair": pair])
    }
    
    /// Subscribe to order book
    public func subscribeToOrderBook(pair: String) {
        sendMessage(type: "subscribe", channel: "orderbook", data: ["pair": pair])
    }
    
    // MARK: - Authentication
    
    /// Send authenticated message
    public func authenticate() {
        sendMessage(type: "auth", channel: "auth", data: [
            "walletId": walletId ?? "",
            "token": authToken ?? ""
        ])
    }
    
    // MARK: - Private Helpers
    
    private func sendMessage(type: String, channel: String, data: [String: Any]) {
        var message: [String: Any] = [
            "type": type,
            "channel": channel,
            "data": data,
            "timestamp": Int(Date().timeIntervalSince1970 * 1000)
        ]
        
        if let jsonData = try? JSONSerialization.data(withJSONObject: message),
           let jsonString = String(data: jsonData, encoding: .utf8) {
            socket?.write(string: jsonString)
        }
    }
    
    private func handleMessage(_ text: String) {
        guard let data = text.data(using: .utf8),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let channel = json["channel"] as? String,
              let payload = json["data"] as? [String: Any] else {
            return
        }
        
        onMessage?(text)
        
        switch channel {
        case "balance":
            handleBalanceUpdate(payload)
        case "transactions":
            handleTransactionUpdate(payload)
        default:
            break
        }
    }
    
    private func handleBalanceUpdate(_ data: [String: Any]) {
        let update = BalanceUpdate(
            chainId: data["chainId"] as? Int ?? 0,
            address: data["address"] as? String ?? "",
            balance: data["balance"] as? String ?? "0",
            token: data["token"] as? String ?? "ETH",
            timestamp: data["timestamp"] as? Int64 ?? 0
        )
        onBalanceUpdate?(update)
    }
    
    private func handleTransactionUpdate(_ data: [String: Any]) {
        let update = TransactionUpdate(
            txHash: data["txHash"] as? String ?? "",
            from: data["from"] as? String ?? "",
            to: data["to"] as? String ?? "",
            amount: data["amount"] as? String ?? "0",
            status: data["status"] as? String ?? "",
            timestamp: data["timestamp"] as? Int64 ?? 0
        )
        onTransactionUpdate?(update)
    }
    
    private func handleReconnect() {
        guard reconnectAttempts < MAX_RECONNECT_ATTEMPTS else {
            connectionState = .error
            onStateChange?(.error)
            return
        }
        
        reconnectAttempts += 1
        connectionState = .reconnecting
        onStateChange?(.reconnecting)
        
        DispatchQueue.main.asyncAfter(deadline: .now() + RECONNECT_DELAY * Double(reconnectAttempts)) { [weak self] in
            guard let self = self, let id = self.walletId else { return }
            self.connect(walletId: id, token: self.authToken)
        }
    }
    
    private func startHeartbeat() {
        stopHeartbeat()
        heartbeatTimer = Timer.scheduledTimer(withTimeInterval: 15, repeats: true) { [weak self] _ in
            self?.sendMessage(type: "ping", channel: "heartbeat", data: [:])
        }
    }
    
    private func stopHeartbeat() {
        heartbeatTimer?.invalidate()
        heartbeatTimer = nil
    }
}

// MARK: - WebSocketDelegate
extension WebSocketService: WebSocketDelegate {
    public func didReceive(event: WebSocketEvent, client: WebSocketClient) {
        switch event {
        case .connected:
            connectionState = .connected
            onStateChange?(.connected)
            reconnectAttempts = 0
            authenticate()
            startHeartbeat()
            
        case .disconnected(let reason, let code):
            connectionState = .disconnected
            onStateChange?(.disconnected)
            stopHeartbeat()
            handleReconnect()
            
        case .text(let text):
            handleMessage(text)
            
        case .error(let error):
            connectionState = .error
            onStateChange?(.error)
            stopHeartbeat()
            handleReconnect()
            
        default:
            break
        }
    }
}

// MARK: - Data Structures
public struct BalanceUpdate {
    public let chainId: Int
    public let address: String
    public let balance: String
    public let token: String
    public let timestamp: Int64
}

public struct TransactionUpdate {
    public let txHash: String
    public let from: String
    public let to: String
    public let amount: String
    public let status: String
    public let timestamp: Int64
}
