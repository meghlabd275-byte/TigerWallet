//
//  TigerWallet Admin Network Service - iOS Implementation
//  Complete networking layer for admin operations
//

import Foundation

// MARK: - API Configuration

struct APIConfig {
    static let baseURL = "https://api.tigerwallet.com"
    static let adminPath = "/api/v1/admin"
    static let timeout: TimeInterval = 30
}

// MARK: - API Client

class AdminAPIClient {
    static let shared = AdminAPIClient()
    
    private let session: URLSession
    private var authToken: String?
    
    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = APIConfig.timeout
        config.timeoutIntervalForResource = APIConfig.timeout
        self.session = URLSession(configuration: config)
    }
    
    // MARK: - Authentication
    
    func setAuthToken(_ token: String?) {
        self.authToken = token
    }
    
    // MARK: - Request Building
    
    private func buildRequest(
        endpoint: String,
        method: HTTPMethod,
        body: [String: Any]? = nil,
        queryParams: [String: String]? = nil
    ) -> URLRequest? {
        var urlString = APIConfig.baseURL + APIConfig.adminPath + endpoint
        
        if let params = queryParams, !params.isEmpty {
            let queryString = params.map { "\($0.key)=\($0.value)" }.joined(separator: "&")
            urlString += "?\(queryString)"
        }
        
        guard let url = URL(string: urlString) else { return nil }
        
        var request = URLRequest(url: url)
        request.httpMethod = method.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        
        if let token = authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        
        if let body = body {
            request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        }
        
        return request
    }
    
    // MARK: - Generic Request
    
    func request<T: Decodable>(
        endpoint: String,
        method: HTTPMethod = .get,
        body: [String: Any]? = nil,
        queryParams: [String: String]? = nil,
        responseType: T.Type,
        completion: @escaping (Result<T, APIError>) -> Void
    ) {
        guard let request = buildRequest(endpoint: endpoint, method: method, body: body, queryParams: queryParams) else {
            completion(.failure(.invalidURL))
            return
        }
        
        session.dataTask(with: request) { data, response, error in
            DispatchQueue.main.async {
                if let error = error {
                    completion(.failure(.networkError(error.localizedDescription)))
                    return
                }
                
                guard let httpResponse = response as? HTTPURLResponse else {
                    completion(.failure(.invalidResponse))
                    return
                }
                
                guard (200...299).contains(httpResponse.statusCode) else {
                    completion(.failure(.httpError(httpResponse.statusCode)))
                    return
                }
                
                guard let data = data else {
                    completion(.failure(.noData))
                    return
                }
                
                do {
                    let decoder = JSONDecoder()
                    decoder.keyDecodingStrategy = .convertFromSnakeCase
                    let result = try decoder.decode(T.self, from: data)
                    completion(.success(result))
                } catch {
                    completion(.failure(.decodingError(error.localizedDescription)))
                }
            }
        }.resume()
    }
    
    // MARK: - Void Request
    
    func voidRequest(
        endpoint: String,
        method: HTTPMethod = .get,
        body: [String: Any]? = nil,
        completion: @escaping (Result<Void, APIError>) -> Void
    ) {
        guard let request = buildRequest(endpoint: endpoint, method: method, body: body) else {
            completion(.failure(.invalidURL))
            return
        }
        
        session.dataTask(with: request) { _, response, error in
            DispatchQueue.main.async {
                if let error = error {
                    completion(.failure(.networkError(error.localizedDescription)))
                    return
                }
                
                guard let httpResponse = response as? HTTPURLResponse else {
                    completion(.failure(.invalidResponse))
                    return
                }
                
                guard (200...299).contains(httpResponse.statusCode) else {
                    completion(.failure(.httpError(httpResponse.statusCode)))
                    return
                }
                
                completion(.success(()))
            }
        }.resume()
    }
}

// MARK: - HTTP Method

enum HTTPMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
    case patch = "PATCH"
}

// MARK: - API Error

enum APIError: Error, LocalizedError {
    case invalidURL
    case networkError(String)
    case invalidResponse
    case httpError(Int)
    case noData
    case decodingError(String)
    
    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "Invalid URL"
        case .networkError(let message):
            return "Network error: \(message)"
        case .invalidResponse:
            return "Invalid response"
        case .httpError(let code):
            return "HTTP error: \(code)"
        case .noData:
            return "No data received"
        case .decodingError(let message):
            return "Decoding error: \(message)"
        }
    }
}

// MARK: - Admin Service

class AdminService {
    static let shared = AdminService()
    private let client = AdminAPIClient.shared
    
    private init() {}
    
    // MARK: - Auth
    
    func login(email: String, password: String, completion: @escaping (Result<LoginResponse, APIError>) -> Void) {
        client.request(
            endpoint: "/login",
            method: .post,
            body: ["email": email, "password": password],
            responseType: LoginResponse.self,
            completion: { result in
                if case .success(let response) = result {
                    self.client.setAuthToken(response.token)
                }
                completion(result)
            }
        )
    }
    
    func logout(completion: @escaping (Result<Void, APIError>) -> Void) {
        client.voidRequest(endpoint: "/logout", method: .post) { [weak self] result in
            if case .success = result {
                self?.client.setAuthToken(nil)
            }
            completion(result)
        }
    }
    
    // MARK: - Users
    
    func getUsers(page: Int = 1, search: String? = nil, status: String? = nil, 
                  kycStatus: String? = nil, completion: @escaping (Result<UsersListResponse, APIError>) -> Void) {
        var params: [String: String] = [
            "page": String(page),
            "page_size": "20"
        ]
        if let search = search { params["search"] = search }
        if let status = status { params["status"] = status }
        if let kycStatus = kycStatus { params["kyc_status"] = kycStatus }
        
        client.request(endpoint: "/users", queryParams: params, responseType: UsersListResponse.self, completion: completion)
    }
    
    func getUser(id: Int, completion: @escaping (Result<User, APIError>) -> Void) {
        client.request(endpoint: "/users/\(id)", responseType: User.self, completion: completion)
    }
    
    func updateUser(id: Int, updates: [String: Any], completion: @escaping (Result<User, APIError>) -> Void) {
        client.request(endpoint: "/users/\(id)", method: .put, body: updates, responseType: User.self, completion: completion)
    }
    
    func banUser(id: Int, reason: String, completion: @escaping (Result<User, APIError>) -> Void) {
        client.request(endpoint: "/users/\(id)/ban", method: .put, body: ["reason": reason], responseType: User.self, completion: completion)
    }
    
    func unbanUser(id: Int, completion: @escaping (Result<User, APIError>) -> Void) {
        client.request(endpoint: "/users/\(id)/unban", method: .put, body: [:], responseType: User.self, completion: completion)
    }
    
    // MARK: - KYC
    
    func getKYCRequests(page: Int = 1, status: String? = nil, completion: @escaping (Result<KYCListResponse, APIError>) -> Void) {
        var params: [String: String] = ["page": String(page)]
        if let status = status { params["status"] = status }
        
        client.request(endpoint: "/kyc", queryParams: params, responseType: KYCListResponse.self, completion: completion)
    }
    
    func approveKYC(id: Int, completion: @escaping (Result<KYCRequest, APIError>) -> Void) {
        client.request(endpoint: "/kyc/\(id)/approve", method: .put, body: [:], responseType: KYCRequest.self, completion: completion)
    }
    
    func rejectKYC(id: Int, reason: String, completion: @escaping (Result<KYCRequest, APIError>) -> Void) {
        client.request(endpoint: "/kyc/\(id)/reject", method: .put, body: ["reason": reason], responseType: KYCRequest.self, completion: completion)
    }
    
    // MARK: - Transactions
    
    func getTransactions(page: Int = 1, status: String? = nil, token: String? = nil, 
                        chain: String? = nil, completion: @escaping (Result<TransactionsListResponse, APIError>) -> Void) {
        var params: [String: String] = ["page": String(page)]
        if let status = status { params["status"] = status }
        if let token = token { params["token"] = token }
        if let chain = chain { params["chain"] = chain }
        
        client.request(endpoint: "/transactions", queryParams: params, responseType: TransactionsListResponse.self, completion: completion)
    }
    
    func getTransaction(id: Int, completion: @escaping (Result<Transaction, APIError>) -> Void) {
        client.request(endpoint: "/transactions/\(id)", responseType: Transaction.self, completion: completion)
    }
    
    // MARK: - Withdrawals
    
    func getWithdrawals(page: Int = 1, status: String? = nil, completion: @escaping (Result<WithdrawalsListResponse, APIError>) -> Void) {
        var params: [String: String] = ["page": String(page)]
        if let status = status { params["status"] = status }
        
        client.request(endpoint: "/withdrawals", queryParams: params, responseType: WithdrawalsListResponse.self, completion: completion)
    }
    
    func approveWithdrawal(id: Int, completion: @escaping (Result<Withdrawal, APIError>) -> Void) {
        client.request(endpoint: "/withdrawals/\(id)/approve", method: .post, body: [:], responseType: Withdrawal.self, completion: completion)
    }
    
    func rejectWithdrawal(id: Int, reason: String, completion: @escaping (Result<Withdrawal, APIError>) -> Void) {
        client.request(endpoint: "/withdrawals/\(id)/reject", method: .post, body: ["reason": reason], responseType: Withdrawal.self, completion: completion)
    }
    
    // MARK: - Tokens
    
    func getTokens(completion: @escaping (Result<TokensListResponse, APIError>) -> Void) {
        client.request(endpoint: "/tokens", responseType: TokensListResponse.self, completion: completion)
    }
    
    func createToken(token: TokenRequest, completion: @escaping (Result<Token, APIError>) -> Void) {
        client.request(endpoint: "/tokens", method: .post, body: token.toDictionary(), responseType: Token.self, completion: completion)
    }
    
    func updateToken(id: Int, updates: [String: Any], completion: @escaping (Result<Token, APIError>) -> Void) {
        client.request(endpoint: "/tokens/\(id)", method: .put, body: updates, responseType: Token.self, completion: completion)
    }
    
    func activateToken(id: Int, completion: @escaping (Result<Token, APIError>) -> Void) {
        client.request(endpoint: "/tokens/\(id)/activate", method: .put, body: [:], responseType: Token.self, completion: completion)
    }
    
    func deactivateToken(id: Int, completion: @escaping (Result<Token, APIError>) -> Void) {
        client.request(endpoint: "/tokens/\(id)/deactivate", method: .put, body: [:], responseType: Token.self, completion: completion)
    }
    
    // MARK: - Fees
    
    func getFeeRules(completion: @escaping (Result<FeeRulesListResponse, APIError>) -> Void) {
        client.request(endpoint: "/fees", responseType: FeeRulesListResponse.self, completion: completion)
    }
    
    func createFeeRule(rule: FeeRuleRequest, completion: @escaping (Result<FeeRule, APIError>) -> Void) {
        client.request(endpoint: "/fees", method: .post, body: rule.toDictionary(), responseType: FeeRule.self, completion: completion)
    }
    
    func updateFeeRule(id: Int, updates: [String: Any], completion: @escaping (Result<FeeRule, APIError>) -> Void) {
        client.request(endpoint: "/fees/\(id)", method: .put, body: updates, responseType: FeeRule.self, completion: completion)
    }
    
    func deleteFeeRule(id: Int, completion: @escaping (Result<Void, APIError>) -> Void) {
        client.voidRequest(endpoint: "/fees/\(id)", method: .delete, completion: completion)
    }
    
    // MARK: - Bots
    
    func getBots(completion: @escaping (Result<BotsListResponse, APIError>) -> Void) {
        client.request(endpoint: "/bots", responseType: BotsListResponse.self, completion: completion)
    }
    
    func createBot(bot: BotRequest, completion: @escaping (Result<Bot, APIError>) -> Void) {
        client.request(endpoint: "/bots", method: .post, body: bot.toDictionary(), responseType: Bot.self, completion: completion)
    }
    
    func startBot(id: Int, completion: @escaping (Result<Bot, APIError>) -> Void) {
        client.request(endpoint: "/bots/\(id)/start", method: .post, body: [:], responseType: Bot.self, completion: completion)
    }
    
    func stopBot(id: Int, completion: @escaping (Result<Bot, APIError>) -> Void) {
        client.request(endpoint: "/bots/\(id)/stop", method: .post, body: [:], responseType: Bot.self, completion: completion)
    }
    
    // MARK: - Analytics
    
    func getDashboardStats(completion: @escaping (Result<DashboardStats, APIError>) -> Void) {
        client.request(endpoint: "/analytics/dashboard", responseType: DashboardStats.self, completion: completion)
    }
    
    func getVolumeAnalytics(period: String = "7d", completion: @escaping (Result<VolumeAnalytics, APIError>) -> Void) {
        client.request(endpoint: "/analytics/volume?period=\(period)", responseType: VolumeAnalytics.self, completion: completion)
    }
    
    func getRevenueAnalytics(period: String = "30d", completion: @escaping (Result<RevenueAnalytics, APIError>) -> Void) {
        client.request(endpoint: "/analytics/revenue?period=\(period)", responseType: RevenueAnalytics.self, completion: completion)
    }
    
    // MARK: - Support
    
    func getTickets(status: String? = nil, completion: @escaping (Result<TicketsListResponse, APIError>) -> Void) {
        var params: [String: String] = [:]
        if let status = status { params["status"] = status }
        
        client.request(endpoint: "/support/tickets", queryParams: params.isEmpty ? nil : params, responseType: TicketsListResponse.self, completion: completion)
    }
    
    func addTicketMessage(ticketId: Int, message: String, isInternal: Bool = false, 
                         completion: @escaping (Result<TicketMessage, APIError>) -> Void) {
        client.request(endpoint: "/support/tickets/\(ticketId)/messages", method: .post, 
                      body: ["content": message, "is_internal": isInternal], 
                      responseType: TicketMessage.self, completion: completion)
    }
    
    func closeTicket(ticketId: Int, completion: @escaping (Result<SupportTicket, APIError>) -> Void) {
        client.request(endpoint: "/support/tickets/\(ticketId)/close", method: .post, body: [:], 
                      responseType: SupportTicket.self, completion: completion)
    }
    
    // MARK: - Notifications
    
    func sendNotification(title: String, message: String, userId: Int? = nil, 
                         sendEmail: Bool = false, sendPush: Bool = false,
                         completion: @escaping (Result<Void, APIError>) -> Void) {
        var body: [String: Any] = [
            "title": title,
            "message": message,
            "send_email": sendEmail,
            "send_push": sendPush
        ]
        if let userId = userId { body["user_id"] = userId }
        
        client.voidRequest(endpoint: "/notifications", method: .post, body: body, completion: completion)
    }
    
    func broadcastNotification(title: String, message: String, completion: @escaping (Result<BroadcastResponse, APIError>) -> Void) {
        client.request(endpoint: "/notifications/broadcast", method: .post, 
                      body: ["title": title, "message": message], 
                      responseType: BroadcastResponse.self, completion: completion)
    }
}
