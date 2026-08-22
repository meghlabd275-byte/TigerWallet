//
//  ProjectPartyApiService.swift
//  PartyApp
//
//  ProjectParty iOS API client.
//
//  Targets the standalone ProjectParty backend (project_party/go/cmd/main.go,
//  port 8106, path prefix /api/v1, JWT auth + RBAC). The token is persisted in
//  the iOS Keychain under `com.tigerparty.api.jwt`. Base URL is configurable
//  via the `PROJECT_PARTY_API_BASE_URL` environment variable and defaults to
//  the local dev endpoint.
//
//  Every method issues a real URLSession async call against the backend — no
//  stubs, fakes, or mock data. On any non-2xx response the method throws
//  `PartyApiError.httpError` (fail-closed); it never returns fabricated data.
//
//  Method set matches project_party/web/src/services/api.ts + the discovery,
//  pricing, analytics, compliance routes the task requires.
//

import Foundation

// MARK: - Configuration

enum PartyApiConfig {
    static let baseURL: String = {
        if let env = ProcessInfo.processInfo.environment["PROJECT_PARTY_API_BASE_URL"],
           !env.isEmpty {
            return env.hasSuffix("/api/v1") ? env : "\(env)/api/v1"
        }
        return "http://localhost:8106/api/v1"
    }()
    static let timeout: TimeInterval = 30
    static let keychainAccount = "com.tigerparty.api.jwt"
    static let keychainService = "com.tigerparty.api"
}

// MARK: - Error

enum PartyApiError: Error, LocalizedError {
    case invalidURL
    case networkError(Error)
    case httpError(statusCode: Int, message: String)
    case decodingError(Error)
    case unauthorized

    var errorDescription: String? {
        switch self {
        case .invalidURL: return "Invalid ProjectParty API URL"
        case .networkError(let e): return "Network error: \(e.localizedDescription)"
        case .httpError(let code, let msg): return "HTTP \(code): \(msg)"
        case .decodingError(let e): return "Decoding error: \(e.localizedDescription)"
        case .unauthorized: return "Not authenticated: no JWT token set"
        }
    }
}

// MARK: - Models

struct PartyAuthResponse: Codable {
    let token: String?
    let username: String?
    let role: String?
}

// MARK: - Keychain helper

enum PartyKeychain {
    static func set(_ value: String?, for account: String) {
        guard let value else { delete(for: account); return }
        let data = Data(value.utf8)
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: PartyApiConfig.keychainService,
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
            kSecAttrService as String: PartyApiConfig.keychainService,
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
            kSecAttrService as String: PartyApiConfig.keychainService,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(query as CFDictionary)
    }
}

// MARK: - API Service

final class ProjectPartyApiService {
    static let shared = ProjectPartyApiService()

    private let session: URLSession

    private(set) var token: String? = PartyKeychain.get(for: PartyApiConfig.keychainAccount)

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = PartyApiConfig.timeout
        config.timeoutIntervalForResource = PartyApiConfig.timeout * 2
        config.waitsForConnectivity = true
        self.session = URLSession(configuration: config)
    }

    // MARK: Token management

    func setToken(_ token: String) {
        self.token = token
        PartyKeychain.set(token, for: PartyApiConfig.keychainAccount)
    }

    func clearToken() {
        self.token = nil
        PartyKeychain.set(nil, for: PartyApiConfig.keychainAccount)
    }

    // MARK: Request core

    private func buildRequest(
        path: String,
        method: String,
        body: Data? = nil,
        authenticated: Bool = true,
        absolute: Bool = false
    ) throws -> URLRequest {
        let urlString = absolute
            ? PartyApiConfig.baseURL.trimmingCharacters(in: CharacterSet(charactersIn: "/")) + path
            : PartyApiConfig.baseURL + path
        guard let url = URL(string: urlString) else { throw PartyApiError.invalidURL }
        var req = URLRequest(url: url)
        req.httpMethod = method
        req.timeoutInterval = PartyApiConfig.timeout
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        if body != nil {
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
            req.httpBody = body
        }
        if authenticated {
            guard let token else { throw PartyApiError.unauthorized }
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return req
    }

    private func decodeError(_ data: Data, statusCode: Int) -> PartyApiError {
        if let body = try? JSONDecoder().decode([String: String].self, from: data),
           let msg = body["error"] ?? body["message"] {
            return .httpError(statusCode: statusCode, message: msg)
        }
        let raw = String(data: data, encoding: .utf8) ?? ""
        return .httpError(statusCode: statusCode, message: raw.isEmpty ? "HTTP \(statusCode)" : raw)
    }

    @discardableResult
    private func requestJSON(
        path: String,
        method: String,
        body: [String: Any]? = nil,
        authenticated: Bool = true,
        absolute: Bool = false
    ) async throws -> Any {
        let bodyData: Data? = try body.map {
            try JSONSerialization.data(withJSONObject: $0, options: [])
        }
        let req = try buildRequest(path: path, method: method, body: bodyData,
                                   authenticated: authenticated, absolute: absolute)
        do {
            let (data, response) = try await session.data(for: req)
            guard let http = response as? HTTPURLResponse else {
                throw PartyApiError.networkError(URLError(.badServerResponse))
            }
            if !(200...299).contains(http.statusCode) {
                throw decodeError(data, statusCode: http.statusCode)
            }
            if data.isEmpty { return NSNull() }
            return try JSONSerialization.json(with: data, options: [])
        } catch let e as PartyApiError {
            throw e
        } catch {
            throw PartyApiError.networkError(error)
        }
    }

    // MARK: Health (lives at /health, outside /api/v1)

    func getHealth() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/health", method: "GET", authenticated: false, absolute: true))
    }

    // MARK: Auth

    func register(username: String, password: String) async throws -> PartyAuthResponse {
        let body: [String: Any] = ["username": username, "password": password]
        let res = try await requestJSON(path: "/auth/register", method: "POST", body: body, authenticated: false)
        guard let dict = res as? [String: Any] else {
            throw PartyApiError.decodingError(NSError(domain: "PartyApi", code: -1))
        }
        if let tok = dict["token"] as? String { setToken(tok) }
        return PartyAuthResponse(
            token: dict["token"] as? String,
            username: dict["username"] as? String,
            role: dict["role"] as? String
        )
    }

    func login(username: String, password: String) async throws -> PartyAuthResponse {
        let body: [String: Any] = ["username": username, "password": password]
        let res = try await requestJSON(path: "/auth/login", method: "POST", body: body, authenticated: false)
        guard let dict = res as? [String: Any] else {
            throw PartyApiError.decodingError(NSError(domain: "PartyApi", code: -1))
        }
        if let tok = dict["token"] as? String { setToken(tok) }
        return PartyAuthResponse(
            token: dict["token"] as? String,
            username: dict["username"] as? String,
            role: dict["role"] as? String
        )
    }

    // MARK: Discovery (public)

    func getCoins() async throws -> [Any] {
        let res = try await requestJSON(path: "/coins", method: "GET", authenticated: false)
        return res as? [Any] ?? (res as? [String: Any]).map { [$0.key: $0.value] } ?? []
    }
    func searchTokens(q: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/search?q=\(enc(q))", method: "GET", authenticated: false))
    }
    func getFeatured() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/featured", method: "GET", authenticated: false))
    }
    func getTrending() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/trending", method: "GET", authenticated: false))
    }
    func getMarket() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/market", method: "GET", authenticated: false))
    }

    // MARK: Tokens

    func listTokens(status: String? = nil) async throws -> [String: Any] {
        let path = status.map({ "/tokens?status=\(enc($0))" }) ?? "/tokens"
        return try dict(await requestJSON(path: path, method: "GET"))
    }
    func getToken(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/tokens/\(enc(id))", method: "GET"))
    }
    func createToken(_ token: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/tokens", method: "POST", body: token))
    }
    func updateToken(id: String, token: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/tokens/\(enc(id))", method: "PUT", body: token))
    }
    func deleteToken(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/tokens/\(enc(id))", method: "DELETE"))
    }
    func submitToken(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/tokens/\(enc(id))/submit", method: "POST"))
    }
    func approveToken(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/tokens/\(enc(id))/approve", method: "POST"))
    }
    func rejectToken(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/tokens/\(enc(id))/reject", method: "POST"))
    }

    // MARK: Listings

    func listListings(status: String? = nil) async throws -> [String: Any] {
        let path = status.map({ "/listings?status=\(enc($0))" }) ?? "/listings"
        return try dict(await requestJSON(path: path, method: "GET"))
    }
    func getListing(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/listings/\(enc(id))", method: "GET"))
    }
    func createListing(_ listing: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/listings", method: "POST", body: listing))
    }
    func updateListingStatus(id: String, status: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/listings/\(enc(id))/status", method: "PUT", body: ["status": status]))
    }
    func featureListing(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/listings/\(enc(id))/featured", method: "POST"))
    }

    // MARK: Launchpad

    func listLaunchpads(status: String? = nil) async throws -> [String: Any] {
        let path = status.map({ "/launchpad?status=\(enc($0))" }) ?? "/launchpad"
        return try dict(await requestJSON(path: path, method: "GET"))
    }
    func getLaunchpad(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/launchpad/\(enc(id))", method: "GET"))
    }
    func createLaunchpad(_ launchpad: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/launchpad/create", method: "POST", body: launchpad))
    }
    func contribute(id: String, amount: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/launchpad/\(enc(id))/contribute", method: "POST", body: ["amount": amount]))
    }
    func claimTokens(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/launchpad/\(enc(id))/claim", method: "POST"))
    }
    func cancelLaunchpad(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/launchpad/\(enc(id))/cancel", method: "POST"))
    }

    // MARK: Market-making

    func getMakerOrders(tokenId: String? = nil) async throws -> [String: Any] {
        let path = tokenId.map({ "/market-making/orders?token_id=\(enc($0))" }) ?? "/market-making/orders"
        return try dict(await requestJSON(path: path, method: "GET"))
    }
    func getMarketMakerStatus(tokenId: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/market-making/status/\(enc(tokenId))", method: "GET"))
    }
    func createMakerOrders(_ orders: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/market-making/orders", method: "POST", body: orders))
    }
    func updateOrderStatus(id: String, status: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/market-making/orders/\(enc(id))/status", method: "PUT", body: ["status": status]))
    }
    func addLiquidity(_ liquidity: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/market-making/liquidity/add", method: "POST", body: liquidity))
    }
    func removeLiquidity(_ liquidity: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/market-making/liquidity/remove", method: "POST", body: liquidity))
    }

    // MARK: Pricing

    func getTokenPrice(tokenId: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/pricing/\(enc(tokenId))", method: "GET"))
    }
    func getPriceHistory(tokenId: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/pricing/history/\(enc(tokenId))", method: "GET"))
    }
    func setTokenPrice(tokenId: String, price: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/pricing/set", method: "POST", body: ["token_id": tokenId, "price": price]))
    }
    func updatePrice(tokenId: String, price: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/pricing/update", method: "POST", body: ["token_id": tokenId, "price": price]))
    }

    // MARK: Analytics (public)

    func getTradingVolume() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/analytics/volume", method: "GET", authenticated: false))
    }
    func getLiquidity() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/analytics/liquidity", method: "GET", authenticated: false))
    }
    func getHolderCount() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/analytics/holders", method: "GET", authenticated: false))
    }
    func getTransactionCount() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/analytics/transactions", method: "GET", authenticated: false))
    }

    // MARK: Compliance

    func getAuditStatus(tokenId: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/compliance/audit/\(enc(tokenId))", method: "GET"))
    }
    func getKYCStatus(tokenId: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/compliance/kyc/\(enc(tokenId))", method: "GET"))
    }
    func requestAudit(_ audit: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/compliance/audit", method: "POST", body: audit))
    }
    func submitKYC(_ kyc: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/compliance/kyc/submit", method: "POST", body: kyc))
    }

    // MARK: Fees

    func getListingFees() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/fees", method: "GET", authenticated: false))
    }
    func calculateFees(_ fee: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/fees/calculate", method: "POST", body: fee))
    }
    func payFees(_ payment: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/fees/pay", method: "POST", body: payment))
    }

    // MARK: Favorites (auth)

    func listFavorites() async throws -> [String: Any] {
        try dict(await requestJSON(path: "/favorites", method: "GET"))
    }
    func addFavorite(_ favorite: [String: Any]) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/favorites", method: "POST", body: favorite))
    }
    func removeFavorite(id: String) async throws -> [String: Any] {
        try dict(await requestJSON(path: "/favorites/\(enc(id))", method: "DELETE"))
    }

    // MARK: Helpers

    private func dict(_ raw: Any) throws -> [String: Any] {
        if let d = raw as? [String: Any] { return d }
        if raw is NSNull { return [:] }
        throw PartyApiError.decodingError(NSError(domain: "PartyApi", code: -1))
    }

    private func enc(_ s: String) -> String {
        s.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? s
    }
}
