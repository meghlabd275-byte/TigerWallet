import Foundation

// MARK: - Session Manager

class SessionManager {
    static let shared = SessionManager()
    
    private let defaults = UserDefaults.standard
    
    private enum Keys {
        static let authToken = "admin_auth_token"
        static let refreshToken = "admin_refresh_token"
        static let expiresAt = "admin_expires_at"
        static let adminUser = "admin_user"
        static let isLoggedIn = "admin_is_logged_in"
    }
    
    var authToken: String? {
        get { defaults.string(forKey: Keys.authToken) }
        set { defaults.set(newValue, forKey: Keys.authToken) }
    }
    
    var refreshToken: String? {
        get { defaults.string(forKey: Keys.refreshToken) }
        set { defaults.set(newValue, forKey: Keys.refreshToken) }
    }
    
    var expiresAt: String? {
        get { defaults.string(forKey: Keys.expiresAt) }
        set { defaults.set(newValue, forKey: Keys.expiresAt) }
    }
    
    var isLoggedIn: Bool {
        get { defaults.bool(forKey: Keys.isLoggedIn) }
        set { defaults.set(newValue, forKey: Keys.isLoggedIn) }
    }
    
    var adminUser: AdminUser? {
        get {
            guard let data = defaults.data(forKey: Keys.adminUser) else { return nil }
            return try? JSONDecoder().decode(AdminUser.self, from: data)
        }
        set {
            if let newValue = newValue {
                let data = try? JSONEncoder().encode(newValue)
                defaults.set(data, forKey: Keys.adminUser)
            } else {
                defaults.removeObject(forKey: Keys.adminUser)
            }
        }
    }
    
    private init() {}
    
    func saveSession(authToken: String, refreshToken: String, expiresAt: String, admin: AdminUser) {
        self.authToken = authToken
        self.refreshToken = refreshToken
        self.expiresAt = expiresAt
        self.adminUser = admin
        self.isLoggedIn = true
    }
    
    func clearSession() {
        authToken = nil
        refreshToken = nil
        expiresAt = nil
        adminUser = nil
        isLoggedIn = false
    }
    
    func isSessionValid() -> Bool {
        guard isLoggedIn, let _ = authToken else { return false }
        // In production, check if token is expired
        return true
    }
}

// MARK: - Cache Manager

class CacheManager {
    static let shared = CacheManager()
    
    private let defaults = UserDefaults.standard
    private let cache = NSCache<NSString, AnyObject>()
    
    private enum Keys {
        static let usersCache = "users_cache"
        static let transactionsCache = "transactions_cache"
        static let tokensCache = "tokens_cache"
        static let kycCache = "kyc_cache"
    }
    
    // Cache expiration times
    private let cacheShort = 5 * 60      // 5 minutes
    private let cacheMedium = 15 * 60    // 15 minutes
    private let cacheLong = 60 * 60     // 1 hour
    
    private init() {
        cache.countLimit = 100
    }
    
    func save<T: Encodable>(_ data: T, forKey key: String, expirationTime: Int = cacheMedium) {
        let entry = CacheEntry(data: data, timestamp: Date(), expirationTime: TimeInterval(expirationTime))
        if let encoded = try? JSONEncoder().encode(entry) {
            defaults.set(encoded, forKey: key)
        }
    }
    
    func get<T: Decodable>(_ type: T.Type, forKey key: String) -> T? {
        guard let data = defaults.data(forKey: key),
              let entry = try? JSONDecoder().decode(CacheEntry<T>.self, from: data) else {
            return nil
        }
        
        let isExpired = Date().timeIntervalSince(entry.timestamp) > entry.expirationTime
        if isExpired {
            defaults.removeObject(forKey: key)
            return nil
        }
        
        return entry.data
    }
    
    func remove(forKey key: String) {
        defaults.removeObject(forKey: key)
    }
    
    func clearAll() {
        let domain = Bundle.main.bundleIdentifier!
        defaults.removePersistentDomain(forName: domain)
        cache.removeAllObjects()
    }
    
    func clearExpired() {
        // Clear all cached items
        for key in [Keys.usersCache, Keys.transactionsCache, Keys.tokensCache, Keys.kycCache] {
            if let _ = get([Any].self, forKey: key) {
                // Entry expired, remove it
                remove(forKey: key)
            }
        }
    }
}

struct CacheEntry<T: Codable>: Codable {
    let data: T
    let timestamp: Date
    let expirationTime: TimeInterval
}

// MARK: - Notification Manager

class NotificationManager {
    static let shared = NotificationManager()
    
    private init() {}
    
    enum NotificationType: String {
        case newUser = "new_user"
        case newTransaction = "new_transaction"
        case kycSubmitted = "kyc_submitted"
        case kycApproved = "kyc_approved"
        case kycRejected = "kyc_rejected"
        case largeTransaction = "large_transaction"
        case suspiciousActivity = "suspicious_activity"
        case systemAlert = "system_alert"
        case withdrawalRequest = "withdrawal_request"
        case tokenListing = "token_listing"
    }
    
    func sendLocalNotification(title: String, body: String, type: NotificationType, data: [String: String]? = nil) {
        // In a real app, use UNUserNotificationCenter
        print("[\(type.rawValue)] \(title): \(body)")
    }
    
    func handleNotificationTap(notificationId: String) {
        // Handle navigation based on notification type
    }
}

// MARK: - WebSocket Manager

class WebSocketManager: NSObject {
    static let shared = WebSocketManager()
    
    private var webSocketTask: URLSessionWebSocketTask?
    private var session: URLSession!
    private var isConnected = false
    private var reconnectAttempts = 0
    private let maxReconnectAttempts = 5
    
    var onConnect: (() -> Void)?
    var onDisconnect: (() -> Void)?
    var onError: ((Error) -> Void)?
    var onMessage: ((String) -> Void)?
    
    private override init() {
        super.init()
        session = URLSession(configuration: .default, delegate: self, delegateQueue: .main)
    }
    
    func connect(token: String) {
        guard let url = URL(string: "wss://ws.tigerwallet.io/admin?token=\(token)") else { return }
        
        webSocketTask = session.webSocketTask(with: url)
        webSocketTask?.resume()
        isConnected = true
        receiveMessage()
        
        onConnect?()
    }
    
    func disconnect() {
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        isConnected = false
        onDisconnect?()
    }
    
    func send(message: String) {
        guard isConnected else { return }
        webSocketTask?.send(.string(message)) { error in
            if let error = error {
                self.onError?(error)
            }
        }
    }
    
    func subscribe(event: String, data: [String: Any]? = nil) {
        var message: [String: Any] = [
            "action": "subscribe",
            "event": event
        ]
        if let data = data {
            message["data"] = data
        }
        
        if let jsonData = try? JSONSerialization.data(withJSONObject: message),
           let jsonString = String(data: jsonData, encoding: .utf8) {
            send(message: jsonString)
        }
    }
    
    private func receiveMessage() {
        webSocketTask?.receive { [weak self] result in
            switch result {
            case .success(let message):
                switch message {
                case .string(let text):
                    self?.onMessage?(text)
                case .data(let data):
                    if let text = String(data: data, encoding: .utf8) {
                        self?.onMessage?(text)
                    }
                @unknown default:
                    break
                }
                self?.receiveMessage()
                
            case .failure(let error):
                self?.onError?(error)
                self?.attemptReconnect()
            }
        }
    }
    
    private func attemptReconnect() {
        guard reconnectAttempts < maxReconnectAttempts else {
            print("Max reconnect attempts reached")
            return
        }
        
        reconnectAttempts += 1
        let delay = Double(reconnectAttempts) * 2.0
        
        DispatchQueue.main.asyncAfter(deadline: .now() + delay) { [weak self] in
            if let token = SessionManager.shared.authToken {
                self?.connect(token: token)
            }
        }
    }
}

extension WebSocketManager: URLSessionWebSocketDelegate {
    func urlSession(_ session: URLSession, webSocketTask: URLSessionWebSocketTask, didOpenWithProtocol protocol: String?) {
        isConnected = true
        reconnectAttempts = 0
        onConnect?()
    }
    
    func urlSession(_ session: URLSession, webSocketTask: URLSessionWebSocketTask, didCloseWith closeCode: URLSessionWebSocketTask.CloseCode, reason: Data?) {
        isConnected = false
        onDisconnect?()
        attemptReconnect()
    }
}

// MARK: - Analytics Manager

class AnalyticsManager {
    static let shared = AnalyticsManager()
    
    private init() {}
    
    func trackScreenView(screenName: String) {
        // Track screen view event
        print("Screen view: \(screenName)")
    }
    
    func trackAction(action: String, details: [String: Any]? = nil) {
        // Track user action
        print("Action: \(action), Details: \(String(describing: details))")
    }
    
    func trackError(error: String, stackTrace: String? = nil) {
        // Track error
        print("Error: \(error)")
    }
    
    func trackPerformance(metric: String, value: Double) {
        // Track performance metric
        print("Performance: \(metric) = \(value)")
    }
    
    func setUserProperties(_ properties: [String: Any]) {
        // Set user properties
        print("User properties: \(properties)")
    }
}

// MARK: - API Key Manager

class APIKeyManager {
    static let shared = APIKeyManager()
    
    private init() {}
    
    func generateAPIKey() -> String {
        // Generate a secure API key
        let characters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
        return String((0..<32).map { _ in characters.randomElement()! })
    }
    
    func hashAPIKey(_ key: String) -> String {
        // Hash the API key for storage
        // In production, use proper hashing
        return key
    }
}

// MARK: - Security Manager

class SecurityManager {
    static let shared = SecurityManager()
    
    private init() {}
    
    func validatePassword(_ password: String) -> PasswordValidationResult {
        var errors: [String] = []
        
        if password.count < 8 {
            errors.append("Password must be at least 8 characters")
        }
        
        if !password.contains(where: { $0.isUppercase }) {
            errors.append("Password must contain uppercase letter")
        }
        
        if !password.contains(where: { $0.isLowercase }) {
            errors.append("Password must contain lowercase letter")
        }
        
        if !password.contains(where: { $0.isNumber }) {
            errors.append("Password must contain number")
        }
        
        if errors.isEmpty {
            return .valid
        } else {
            return .invalid(errors)
        }
    }
    
    func encryptData(_ data: Data, key: String) -> Data? {
        // In production, use proper encryption
        return data
    }
    
    func decryptData(_ data: Data, key: String) -> Data? {
        // In production, use proper decryption
        return data
    }
}

enum PasswordValidationResult {
    case valid
    case invalid([String])
}

// MARK: - Network Manager

class NetworkManager {
    static let shared = NetworkManager()
    
    private init() {}
    
    var isNetworkAvailable: Bool {
        // Check network connectivity
        return true
    }
    
    func checkConnectivity() async -> Bool {
        // Check if network is available
        return true
    }
}

// MARK: - Database Manager

class DatabaseManager {
    static let shared = DatabaseManager()
    
    private init() {}
    
    // Data is persisted in the PostgreSQL backend (admin/go :9093); no local DB
    // For this implementation, we use UserDefaults as a placeholder
    
    func save<T: Codable>(_ object: T, forKey key: String) {
        if let data = try? JSONEncoder().encode(object) {
            UserDefaults.standard.set(data, forKey: key)
        }
    }
    
    func load<T: Codable>(_ type: T.Type, forKey key: String) -> T? {
        guard let data = UserDefaults.standard.data(forKey: key) else { return nil }
        return try? JSONDecoder().decode(type, from: data)
    }
    
    func delete(forKey key: String) {
        UserDefaults.standard.removeObject(forKey: key)
    }
}
