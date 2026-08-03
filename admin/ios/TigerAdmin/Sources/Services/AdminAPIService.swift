//
//  AdminAPIService.swift
//  TigerAdmin - Admin API Service
//

import Foundation

class AdminAPIService {
    private let baseURL: String
    
    init(baseURL: String = "https://api.tigerwallet.io/admin") {
        self.baseURL = baseURL
    }
    
    // MARK: - User Management APIs
    func getUsers(status: String? = nil, limit: Int = 50) async throws -> [User] {
        var endpoint = "/api/v1/admin/users?limit=\(limit)"
        if let status = status {
            endpoint += "&status=\(status)"
        }
        return try await request(endpoint: endpoint)
    }
    
    func getUser(id: String) async throws -> User {
        return try await request(endpoint: "/api/v1/admin/users/\(id)")
    }
    
    func updateUserKYC(userId: String, status: String) async throws -> User {
        let body = try JSONEncoder().encode(["kyc_status": status])
        return try await request(endpoint: "/api/v1/admin/users/\(userId)/kyc", method: "PUT", body: body)
    }
    
    func deleteUser(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/admin/users/\(id)", method: "DELETE")
    }
    
    // MARK: - Transaction APIs
    func getTransactions(status: String? = nil, limit: Int = 50) async throws -> [AdminTransaction] {
        var endpoint = "/api/v1/admin/transactions?limit=\(limit)"
        if let status = status {
            endpoint += "&status=\(status)"
        }
        return try await request(endpoint: endpoint)
    }
    
    func getTransaction(id: String) async throws -> AdminTransaction {
        return try await request(endpoint: "/api/v1/admin/transactions/\(id)")
    }
    
    func approveTransaction(id: String) async throws -> AdminTransaction {
        return try await request(endpoint: "/api/v1/admin/transactions/\(id)/approve", method: "POST")
    }
    
    func rejectTransaction(id: String, reason: String) async throws -> AdminTransaction {
        let body = try JSONEncoder().encode(["reason": reason])
        return try await request(endpoint: "/api/v1/admin/transactions/\(id)/reject", method: "POST", body: body)
    }
    
    func cancelTransaction(id: String) async throws -> AdminTransaction {
        return try await request(endpoint: "/api/v1/admin/transactions/\(id)/cancel", method: "POST")
    }
    
    // MARK: - Analytics APIs
    func getAnalytics() async throws -> AdminAnalytics {
        return try await request(endpoint: "/api/v1/admin/analytics")
    }
    
    func getUserAnalytics(period: String) async throws -> [AnalyticsData] {
        return try await request(endpoint: "/api/v1/admin/analytics/users?period=\(period)")
    }
    
    func getVolumeAnalytics(period: String) async throws -> [AnalyticsData] {
        return try await request(endpoint: "/api/v1/admin/analytics/volume?period=\(period)")
    }
    
    // MARK: - System APIs
    func getSystemStatus() async throws -> SystemStatus {
        return try await request(endpoint: "/api/v1/admin/system/status")
    }
    
    func getServiceStatus(service: String) async throws -> ServiceStatus {
        return try await request(endpoint: "/api/v1/admin/system/services/\(service)")
    }
    
    func restartService(service: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/admin/system/services/\(service)/restart", method: "POST")
    }
    
    // MARK: - Fee Configuration APIs
    func getFeeConfig() async throws -> FeeConfig {
        return try await request(endpoint: "/api/v1/admin/fees")
    }
    
    func updateFeeConfig(config: FeeConfig) async throws -> FeeConfig {
        let body = try JSONEncoder().encode(config)
        return try await request(endpoint: "/api/v1/admin/fees", method: "PUT", body: body)
    }
    
    // MARK: - Token Management APIs
    func getTokens() async throws -> [Token] {
        return try await request(endpoint: "/api/v1/admin/tokens")
    }
    
    func listToken(address: String, listing: Bool) async throws -> Token {
        let body = try JSONEncoder().encode(["listed": listing])
        return try await request(endpoint: "/api/v1/admin/tokens/\(address)/listing", method: "PUT", body: body)
    }
    
    // MARK: - Admin User APIs
    func getAdminUsers() async throws -> [AdminUser] {
        return try await request(endpoint: "/api/v1/admin/admins")
    }
    
    func createAdminUser(user: CreateAdminUserRequest) async throws -> AdminUser {
        let body = try JSONEncoder().encode(user)
        return try await request(endpoint: "/api/v1/admin/admins", method: "POST", body: body)
    }
    
    func updateAdminPermissions(adminId: String, permissions: [String]) async throws -> AdminUser {
        let body = try JSONEncoder().encode(["permissions": permissions])
        return try await request(endpoint: "/api/v1/admin/admins/\(adminId)/permissions", method: "PUT", body: body)
    }
    
    func deleteAdminUser(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/admin/admins/\(id)", method: "DELETE")
    }
    
    // MARK: - Generic Request
    private func request<T: Decodable>(_ endpoint: String, method: String = "GET", body: Data? = nil) async throws -> T {
        guard let url = URL(string: "\(baseURL)\(endpoint)") else {
            throw APIError(code: "INVALID_URL", message: "Invalid URL")
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        if let body = body {
            request.httpBody = body
        }
        
        let (data, response) = try await URLSession.shared.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError(code: "INVALID_RESPONSE", message: "Invalid response")
        }
        
        guard (200...299).contains(httpResponse.statusCode) else {
            throw APIError(code: "HTTP_\(httpResponse.statusCode)", message: "HTTP error")
        }
        
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        
        return try decoder.decode(T.self, from: data)
    }
}

struct APIError: Error {
    let code: String
    let message: String
}

struct EmptyResponse: Codable {}

struct User: Codable {
    let id: String
    let email: String
    let name: String
    let kycStatus: String
    let createdAt: Date
}

struct AdminTransaction: Codable {
    let id: String
    let hash: String
    let from: String
    let to: String
    let amount: String
    let chain: String
    let type: String
    let status: String
    let createdAt: Date
    var processedAt: Date?
}

struct AdminAnalytics: Codable {
    let totalUsers: Int
    let totalVolumeUSD: Double
    let totalTransactions: Int
    let pendingKYC: Int
    let systemHealth: Double
    let activeUsers24h: Int
    let volume24h: Double
}

struct AnalyticsData: Codable {
    let date: Date
    let value: Double
    let change: Double
}

struct SystemStatus: Codable {
    let uptime: Double
    let services: [ServiceStatus]
    let database: DatabaseStatus
    let cache: CacheStatus
}

struct ServiceStatus: Codable {
    let name: String
    let status: String
    let uptime: Double
    let lastCheck: Date
}

struct DatabaseStatus: Codable {
    let postgres: ComponentStatus
    let redis: ComponentStatus
}

struct ComponentStatus: Codable {
    let status: String
    let connections: Int
    let latency: Double
}

struct CacheStatus: Codable {
    let status: String
    let hitRate: Double
    let memory: Int64
}

struct FeeConfig: Codable {
    var tradingFee: Double
    var withdrawalFee: Double
    var depositFee: Double
    var networkFee: Double
}

struct Token: Codable {
    let address: String
    let name: String
    let symbol: String
    let decimals: Int
    let isListed: Bool
    let marketCap: Double
    let volume24h: Double
}

struct AdminUser: Codable {
    let id: String
    let email: String
    let name: String
    let role: String
    let permissions: [String]
    let createdAt: Date
    var lastLogin: Date?
}

struct CreateAdminUserRequest: Codable {
    let email: String
    let name: String
    let role: String
    let permissions: [String]
}
