import Foundation

class ApiService {
    static let shared = ApiService()
    
    private let baseURL = "https://admin-api.tigerwallet.com"
    private var token: String? {
        get { UserDefaults.standard.string(forKey: "auth_token") }
        set { UserDefaults.standard.set(newValue, forKey: "auth_token") }
    }
    
    private var headers: [String: String] {
        var headers = ["Content-Type": "application/json"]
        if let token = token {
            headers["Authorization"] = "Bearer \(token)"
        }
        return headers
    }
    
    private init() {}
    
    // MARK: - Auth
    
    func login(email: String, password: String) async throws -> LoginResponse {
        let body: [String: Any] = ["email": email, "password": password]
        let response: LoginResponse = try await post("/api/v1/auth/login", body: body)
        token = response.token
        return response
    }
    
    func logout() async throws {
        let _: EmptyResponse = try await post("/api/v1/auth/logout", body: [:])
        token = nil
    }
    
    func getCurrentAdmin() async throws -> Admin {
        return try await get("/api/v1/auth/me")
    }
    
    // MARK: - Users
    
    func getUsers(page: Int = 1, limit: Int = 20, status: String? = nil, search: String? = nil) async throws -> UsersResponse {
        var params: [String: String] = [
            "page": String(page),
            "limit": String(limit)
        ]
        if let status = status { params["status"] = status }
        if let search = search { params["search"] = search }
        return try await get("/api/v1/users", params: params)
    }
    
    func getUser(id: String) async throws -> User {
        return try await get("/api/v1/users/\(id)")
    }
    
    func updateUser(id: String, data: [String: Any]) async throws -> User {
        return try await put("/api/v1/users/\(id)", body: data)
    }
    
    func suspendUser(id: String, reason: String) async throws {
        let body: [String: Any] = ["reason": reason]
        let _: EmptyResponse = try await post("/api/v1/users/\(id)/suspend", body: body)
    }
    
    func banUser(id: String, reason: String) async throws {
        let body: [String: Any] = ["reason": reason]
        let _: EmptyResponse = try await post("/api/v1/users/\(id)/ban", body: body)
    }
    
    // MARK: - KYC
    
    func getKYCSubmissions(page: Int = 1, limit: Int = 20, status: String? = nil, level: Int? = nil) async throws -> KYCListResponse {
        var params: [String: String] = [
            "page": String(page),
            "limit": String(limit)
        ]
        if let status = status { params["status"] = status }
        if let level = level { params["level"] = String(level) }
        return try await get("/api/v1/kyc", params: params)
    }
    
    func approveKYC(id: String, notes: String? = nil) async throws {
        let body: [String: Any] = notes != nil ? ["notes": notes!] : [:]
        let _: EmptyResponse = try await post("/api/v1/kyc/\(id)/approve", body: body)
    }
    
    func rejectKYC(id: String, reason: String) async throws {
        let body: [String: Any] = ["reason": reason]
        let _: EmptyResponse = try await post("/api/v1/kyc/\(id)/reject", body: body)
    }
    
    // MARK: - Tokens
    
    func getTokens(page: Int = 1, limit: Int = 20, status: String? = nil, chain: String? = nil) async throws -> TokensResponse {
        var params: [String: String] = [
            "page": String(page),
            "limit": String(limit)
        ]
        if let status = status { params["status"] = status }
        if let chain = chain { params["chain"] = chain }
        return try await get("/api/v1/tokens", params: params)
    }
    
    func createToken(data: [String: Any]) async throws -> Token {
        return try await post("/api/v1/tokens", body: data)
    }
    
    func verifyToken(id: String) async throws {
        let _: EmptyResponse = try await post("/api/v1/tokens/\(id)/verify", body: [:])
    }
    
    func deleteToken(id: String) async throws {
        let _: EmptyResponse = try await delete("/api/v1/tokens/\(id)")
    }
    
    // MARK: - Pairs
    
    func getPairs(page: Int = 1, limit: Int = 20, status: String? = nil, chain: String? = nil) async throws -> PairsResponse {
        var params: [String: String] = [
            "page": String(page),
            "limit": String(limit)
        ]
        if let status = status { params["status"] = status }
        if let chain = chain { params["chain"] = chain }
        return try await get("/api/v1/pairs", params: params)
    }
    
    func updatePair(id: String, data: [String: Any]) async throws -> TradingPair {
        return try await put("/api/v1/pairs/\(id)", body: data)
    }
    
    // MARK: - Transactions
    
    func getTransactions(page: Int = 1, limit: Int = 20, status: String? = nil, type: String? = nil) async throws -> TransactionsResponse {
        var params: [String: String] = [
            "page": String(page),
            "limit": String(limit)
        ]
        if let status = status { params["status"] = status }
        if let type = type { params["type"] = type }
        return try await get("/api/v1/transactions", params: params)
    }
    
    // MARK: - Withdrawals
    
    func getWithdrawals(page: Int = 1, limit: Int = 20, status: String? = nil) async throws -> WithdrawalsResponse {
        var params: [String: String] = [
            "page": String(page),
            "limit": String(limit)
        ]
        if let status = status { params["status"] = status }
        return try await get("/api/v1/withdrawals", params: params)
    }
    
    func approveWithdrawal(id: String) async throws {
        let _: EmptyResponse = try await post("/api/v1/withdrawals/\(id)/approve", body: [:])
    }
    
    func rejectWithdrawal(id: String, reason: String) async throws {
        let body: [String: Any] = ["reason": reason]
        let _: EmptyResponse = try await post("/api/v1/withdrawals/\(id)/reject", body: body)
    }
    
    // MARK: - Chains
    
    func getChains() async throws -> ChainsResponse {
        return try await get("/api/v1/chains")
    }
    
    func createChain(data: [String: Any]) async throws -> Chain {
        return try await post("/api/v1/chains", body: data)
    }
    
    func updateChain(id: String, data: [String: Any]) async throws -> Chain {
        return try await put("/api/v1/chains/\(id)", body: data)
    }
    
    // MARK: - Fees
    
    func getFees() async throws -> FeesResponse {
        return try await get("/api/v1/fees")
    }
    
    func createFee(data: [String: Any]) async throws -> Fee {
        return try await post("/api/v1/fees", body: data)
    }
    
    func updateFee(id: String, data: [String: Any]) async throws -> Fee {
        return try await put("/api/v1/fees/\(id)", body: data)
    }
    
    // MARK: - White Labels
    
    func getWhiteLabels(page: Int = 1, limit: Int = 20, status: String? = nil) async throws -> WhiteLabelsResponse {
        var params: [String: String] = [
            "page": String(page),
            "limit": String(limit)
        ]
        if let status = status { params["status"] = status }
        return try await get("/api/v1/white-labels", params: params)
    }
    
    func approveWhiteLabel(id: String) async throws {
        let _: EmptyResponse = try await post("/api/v1/white-labels/\(id)/approve", body: [:])
    }
    
    func suspendWhiteLabel(id: String, reason: String) async throws {
        let body: [String: Any] = ["reason": reason]
        let _: EmptyResponse = try await post("/api/v1/white-labels/\(id)/suspend", body: body)
    }
    
    // MARK: - Dashboard
    
    func getDashboard() async throws -> DashboardStats {
        return try await get("/api/v1/dashboard")
    }
    
    // MARK: - HTTP Methods
    
    private func get<T: Decodable>(_ endpoint: String, params: [String: String] = [:]) async throws -> T {
        var urlString = baseURL + endpoint
        if !params.isEmpty {
            let queryString = params.map { "\($0.key)=\($0.value)" }.joined(separator: "&")
            urlString += "?" + queryString
        }
        
        guard let url = URL(string: urlString) else {
            throw ApiError.invalidURL
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        headers.forEach { request.setValue($0.value, forHTTPHeaderField: $0.key) }
        
        let (data, response) = try await URLSession.shared.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse else {
            throw ApiError.invalidResponse
        }
        
        guard httpResponse.statusCode >= 200 && httpResponse.statusCode < 300 else {
            throw ApiError.httpError(httpResponse.statusCode)
        }
        
        let decoder = JSONDecoder()
        return try decoder.decode(T.self, from: data)
    }
    
    private func post<T: Decodable>(_ endpoint: String, body: [String: Any]) async throws -> T {
        guard let url = URL(string: baseURL + endpoint) else {
            throw ApiError.invalidURL
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        headers.forEach { request.setValue($0.value, forHTTPHeaderField: $0.key) }
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, response) = try await URLSession.shared.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse else {
            throw ApiError.invalidResponse
        }
        
        guard httpResponse.statusCode >= 200 && httpResponse.statusCode < 300 else {
            throw ApiError.httpError(httpResponse.statusCode)
        }
        
        let decoder = JSONDecoder()
        return try decoder.decode(T.self, from: data)
    }
    
    private func put<T: Decodable>(_ endpoint: String, body: [String: Any]) async throws -> T {
        guard let url = URL(string: baseURL + endpoint) else {
            throw ApiError.invalidURL
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        headers.forEach { request.setValue($0.value, forHTTPHeaderField: $0.key) }
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, response) = try await URLSession.shared.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse else {
            throw ApiError.invalidResponse
        }
        
        guard httpResponse.statusCode >= 200 && httpResponse.statusCode < 300 else {
            throw ApiError.httpError(httpResponse.statusCode)
        }
        
        let decoder = JSONDecoder()
        return try decoder.decode(T.self, from: data)
    }
    
    private func delete<T: Decodable>(_ endpoint: String) async throws -> T {
        guard let url = URL(string: baseURL + endpoint) else {
            throw ApiError.invalidURL
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        headers.forEach { request.setValue($0.value, forHTTPHeaderField: $0.key) }
        
        let (data, response) = try await URLSession.shared.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse else {
            throw ApiError.invalidResponse
        }
        
        guard httpResponse.statusCode >= 200 && httpResponse.statusCode < 300 else {
            throw ApiError.httpError(httpResponse.statusCode)
        }
        
        let decoder = JSONDecoder()
        return try decoder.decode(T.self, from: data)
    }
}

// MARK: - Error Handling

enum ApiError: Error, LocalizedError {
    case invalidURL
    case invalidResponse
    case httpError(Int)
    case decodingError
    
    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "Invalid URL"
        case .invalidResponse:
            return "Invalid response"
        case .httpError(let code):
            return "HTTP error: \(code)"
        case .decodingError:
            return "Failed to decode response"
        }
    }
}

struct EmptyResponse: Codable {}
