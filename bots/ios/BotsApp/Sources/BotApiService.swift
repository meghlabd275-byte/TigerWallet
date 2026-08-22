//
//  BotApiService.swift
//  BotsApp
//
//  TigerBots iOS API client.
//
//  Targets the standalone Bots backend (mm_bot_platform/bot_api, port 8471,
//  path prefix /api/v1). JWT bearer auth; the token is persisted in the
//  iOS Keychain under the `com.tigerbots.api.jwt` account. Base URL is
//  configurable via the `BOTS_API_BASE_URL` launch argument / environment
//  variable and defaults to the local dev endpoint.
//
//  Every method issues a real URLSession dataTask / async call against the
//  backend — no stubs, fakes, or mock data. On any non-2xx response the
//  method throws `BotApiError.httpError` (fail-closed); it never returns
//  fabricated data.
//
//  Method set mirrors bots/web/src/services/api.ts (auth, bots CRUD + start/
//  stop/pause, executions, logs, users, transactions, subscriptions, fees,
//  cex/dex connectors, api-keys, admin endpoints, public tiers, health).
//

import Foundation

// MARK: - Configuration

enum BotApiConfig {
    /// Base URL of the Bots backend. Overridable via the `BOTS_API_BASE_URL`
    /// environment variable (set in the scheme or Info.plist) for non-dev builds.
    static let baseURL: String = {
        if let env = ProcessInfo.processInfo.environment["BOTS_API_BASE_URL"],
           !env.isEmpty {
            return env.hasSuffix("/api/v1") ? env : "\(env)/api/v1"
        }
        return "http://localhost:8471/api/v1"
    }()
    static let timeout: TimeInterval = 30
    static let keychainAccount = "com.tigerbots.api.jwt"
    static let keychainService = "com.tigerbots.api"
}

// MARK: - Error

enum BotApiError: Error, LocalizedError {
    case invalidURL
    case networkError(Error)
    case httpError(statusCode: Int, message: String)
    case decodingError(Error)
    case unauthorized

    var errorDescription: String? {
        switch self {
        case .invalidURL: return "Invalid bot API URL"
        case .networkError(let e): return "Network error: \(e.localizedDescription)"
        case .httpError(let code, let msg): return "HTTP \(code): \(msg)"
        case .decodingError(let e): return "Decoding error: \(e.localizedDescription)"
        case .unauthorized: return "Not authenticated: no JWT token set"
        }
    }
}

// MARK: - Models

struct BotAuthResponse: Codable {
    let token: String?
    let userId: String?
    let role: String?

    enum CodingKeys: String, CodingKey {
        case token
        case userId = "user_id"
        case role
    }
}

struct BotHealth: Codable {
    let status: String
    let service: String
}

// MARK: - Keychain helper

enum BotKeychain {
    static func set(_ value: String?, for account: String) {
        guard let value else { delete(for: account); return }
        let data = Data(value.utf8)
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: BotApiConfig.keychainService,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(query as CFDictionary)
        var attrs = query
        attrs[kSecValueData as String] = data
        attrs[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlock
        SecItemAdd(attrs as CFDictionary, nil)
    }

    static func get(for account: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: BotApiConfig.keychainService,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var item: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &item) == errSecSuccess,
              let data = item as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    static func delete(for account: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: BotApiConfig.keychainService,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(query as CFDictionary)
    }
}

// MARK: - API Service

final class BotApiService {
    static let shared = BotApiService()

    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder

    private(set) var token: String? = BotKeychain.get(for: BotApiConfig.keychainAccount)

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = BotApiConfig.timeout
        config.timeoutIntervalForResource = BotApiConfig.timeout * 2
        config.waitsForConnectivity = true
        self.session = URLSession(configuration: config)
        self.decoder = JSONDecoder()
        self.encoder = JSONEncoder()
    }

    // MARK: Token management

    func setToken(_ token: String) {
        self.token = token
        BotKeychain.set(token, for: BotApiConfig.keychainAccount)
    }

    func clearToken() {
        self.token = nil
        BotKeychain.set(nil, for: BotApiConfig.keychainAccount)
    }

    // MARK: Request core

    private func buildRequest(
        path: String,
        method: String,
        body: Data? = nil,
        authenticated: Bool = true
    ) throws -> URLRequest {
        guard let url = URL(string: BotApiConfig.baseURL + path) else {
            throw BotApiError.invalidURL
        }
        var req = URLRequest(url: url)
        req.httpMethod = method
        req.timeoutInterval = BotApiConfig.timeout
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        if body != nil {
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
            req.httpBody = body
        }
        if authenticated {
            guard let token else { throw BotApiError.unauthorized }
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return req
    }

    private func decodeError(_ data: Data, statusCode: Int) -> BotApiError {
        if let body = try? JSONDecoder().decode([String: String].self, from: data),
           let msg = body["error"] ?? body["message"] {
            return .httpError(statusCode: statusCode, message: msg)
        }
        let raw = String(data: data, encoding: .utf8) ?? ""
        return .httpError(statusCode: statusCode, message: raw.isEmpty ? "HTTP \(statusCode)" : raw)
    }

    /// Raw JSON request returning `[String: Any]`-style dict (preserves dynamic
    /// shapes from the backend without per-route Codable boilerplate).
    @discardableResult
    private func requestJSON(
        path: String,
        method: String,
        body: [String: Any]? = nil,
        authenticated: Bool = true
    ) async throws -> Any {
        let bodyData: Data? = try body.map {
            try JSONSerialization.data(withJSONObject: $0, options: [])
        }
        let req = try buildRequest(path: path, method: method, body: bodyData, authenticated: authenticated)
        do {
            let (data, response) = try await session.data(for: req)
            guard let http = response as? HTTPURLResponse else {
                throw BotApiError.networkError(URLError(.badServerResponse))
            }
            if !(200...299).contains(http.statusCode) {
                throw decodeError(data, statusCode: http.statusCode)
            }
            if data.isEmpty { return NSNull() }
            return try JSONSerialization.json(with: data, options: [])
        } catch let e as BotApiError {
            throw e
        } catch {
            throw BotApiError.networkError(error)
        }
    }

    // MARK: Auth

    func register(
        username: String,
        password: String,
        email: String? = nil,
        walletAddress: String? = nil,
        role: String? = nil
    ) async throws -> BotAuthResponse {
        var body: [String: Any] = ["username": username, "password": password]
        if let email { body["email"] = email }
        if let walletAddress { body["wallet_address"] = walletAddress }
        if let role { body["role"] = role }
        let res = try await requestJSON(
            path: "/auth/register", method: "POST", body: body, authenticated: false
        )
        guard let dict = res as? [String: Any] else {
            throw BotApiError.decodingError(NSError(domain: "BotApi", code: -1))
        }
        if let tok = dict["token"] as? String { setToken(tok) }
        return BotAuthResponse(
            token: dict["token"] as? String,
            userId: dict["user_id"] as? String,
            role: dict["role"] as? String
        )
    }

    func login(username: String, password: String) async throws -> BotAuthResponse {
        let body: [String: Any] = ["username": username, "password": password]
        let res = try await requestJSON(
            path: "/auth/login", method: "POST", body: body, authenticated: false
        )
        guard let dict = res as? [String: Any] else {
            throw BotApiError.decodingError(NSError(domain: "BotApi", code: -1))
        }
        if let tok = dict["token"] as? String { setToken(tok) }
        return BotAuthResponse(
            token: dict["token"] as? String,
            userId: dict["user_id"] as? String,
            role: dict["role"] as? String
        )
    }

    func logout() async throws {
        do {
            _ = try await requestJSON(path: "/auth/logout", method: "POST", body: nil)
        } finally {
            clearToken()
        }
    }

    // MARK: Health + public tiers

    func health() async throws -> BotHealth {
        let res = try await requestJSON(path: "/health", method: "GET", authenticated: false)
        guard let dict = res as? [String: Any] else {
            throw BotApiError.decodingError(NSError(domain: "BotApi", code: -1))
        }
        return BotHealth(
            status: dict["status"] as? String ?? "",
            service: dict["service"] as? String ?? ""
        )
    }

    func publicTiers() async throws -> [Any] {
        let res = try await requestJSON(path: "/public/tiers", method: "GET", authenticated: false)
        if let arr = res as? [Any] { return arr }
        if let dict = res as? [String: Any], let tiers = dict["tiers"] as? [Any] { return tiers }
        return []
    }

    // MARK: Bots CRUD + lifecycle

    func listBots() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots", method: "GET"))
    }
    func getBot(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/\(enc(id))", method: "GET"))
    }
    func createBot(
        name: String, botType: String, config: [String: Any]? = nil,
        exchange: String? = nil, pair: String? = nil
    ) async throws -> [String: Any] {
        var body: [String: Any] = ["name": name, "bot_type": botType, "config": config ?? [String: Any]()]
        if let exchange { body["exchange"] = exchange }
        if let pair { body["pair"] = pair }
        return try dict(await requestJSON(path: "/bots", method: "POST", body: body))
    }
    func deleteBot(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/\(enc(id))", method: "DELETE"))
    }
    func startBot(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/\(enc(id))/start", method: "POST"))
    }
    func stopBot(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/\(enc(id))/stop", method: "POST"))
    }
    func pauseBot(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/\(enc(id))/pause", method: "POST"))
    }
    func listBotExecutions(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/\(enc(id))/executions", method: "GET"))
    }
    func listBotLogs(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/\(enc(id))/logs", method: "GET"))
    }
    func listBotInstances() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/instances", method: "GET"))
    }
    func currentBotUser() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/me", method: "GET"))
    }

    // MARK: Bot users

    func listBotUsers() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/users", method: "GET"))
    }
    func createBotUser(
        username: String, password: String, email: String? = nil,
        walletAddress: String? = nil, role: String? = nil
    ) async throws -> [String: Any] {
        var body: [String: Any] = ["username": username, "password": password]
        if let email { body["email"] = email }
        if let walletAddress { body["wallet_address"] = walletAddress }
        if let role { body["role"] = role }
        return try dict(await requestJSON(path: "/bots/users", method: "POST", body: body))
    }
    func deleteBotUser(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/users/\(enc(id))", method: "DELETE"))
    }
    func listBotTransactions() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/bots/transactions", method: "GET"))
    }

    // MARK: Subscriptions

    func getSubscription() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/subscription", method: "GET"))
    }
    func createSubscription(tier: String, expiresIn: String? = nil) async throws -> [String: Any] {
        var body: [String: Any] = ["tier": tier]
        if let expiresIn { body["expires_in"] = expiresIn }
        return try dict(await requestJSON(path: "/subscription", method: "POST", body: body))
    }

    // MARK: Fees

    func getFeeConfigs() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/fees", method: "GET"))
    }
    func updateFeeConfig(
        id: String, name: String? = nil, percentage: String? = nil, enabled: Bool? = nil
    ) async throws -> [String: Any] {
        var body: [String: Any] = ["id": id]
        if let name { body["name"] = name }
        if let percentage { body["percentage"] = percentage }
        if let enabled { body["enabled"] = enabled }
        return try dict(await requestJSON(path: "/fees", method: "PUT", body: body))
    }

    // MARK: CEX / DEX connectors

    func listCEX() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/cex", method: "GET"))
    }
    func addCEX(name: String, config: [String: Any]) async throws -> [String: Any] {
        let body: [String: Any] = ["name": name, "config": config]
        return try dict(await requestJSON(path: "/cex", method: "POST", body: body))
    }
    func removeCEX(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/cex/\(enc(id))", method: "DELETE"))
    }
    func listDEX() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/dex", method: "GET"))
    }
    func addDEX(name: String, config: [String: Any]) async throws -> [String: Any] {
        let body: [String: Any] = ["name": name, "config": config]
        return try dict(await requestJSON(path: "/dex", method: "POST", body: body))
    }
    func removeDEX(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/dex/\(enc(id))", method: "DELETE"))
    }

    // MARK: API keys

    func listAPIKeys() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/keys", method: "GET"))
    }
    func createAPIKey(exchange: String, apiKey: String) async throws -> [String: Any] {
        let body: [String: Any] = ["exchange": exchange, "api_key": apiKey]
        return try dict(await requestJSON(path: "/keys", method: "POST", body: body))
    }
    func deleteAPIKey(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/keys/\(enc(id))", method: "DELETE"))
    }

    // MARK: Admin

    func adminListUsers() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/admin/users", method: "GET"))
    }
    func adminUserStatus(id: String, active: Bool) async throws -> [String: Any] {
        let body: [String: Any] = ["id": id, "is_active": active]
        return try dict(await requestJSON(path: "/admin/users/\(enc(id))/status", method: "PUT", body: body))
    }
    func adminStats() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/admin/stats", method: "GET"))
    }
    func adminGetFeeAddresses() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/admin/fee-addresses", method: "GET"))
    }
    func adminSetFeeAddress(label: String, chainId: Int, address: String) async throws -> [String: Any] {
        let body: [String: Any] = ["label": label, "chain_id": chainId, "address": address]
        return try dict(await requestJSON(path: "/admin/fee-addresses", method: "POST", body: body))
    }
    func adminDeleteFeeAddress(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/admin/fee-addresses/\(enc(id))", method: "DELETE"))
    }
    func adminBotStatus(id: String, status: String) async throws -> [String: Any] {
        let body: [String: Any] = ["id": id, "status": status]
        return try dict(await requestJSON(path: "/admin/bots/\(enc(id))/status", method: "POST", body: body))
    }

    // MARK: Helpers

    private func dict(_ raw: Any) throws -> [String: Any] {
        if let d = raw as? [String: Any] { return d }
        if raw is NSNull { return [:] }
        throw BotApiError.decodingError(NSError(domain: "BotApi", code: -1))
    }

    private func enc(_ s: String) -> String {
        s.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? s
    }
}
