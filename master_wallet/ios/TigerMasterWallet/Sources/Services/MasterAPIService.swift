//
//  MasterAPIService.swift
//  TigerMasterWallet - Master Wallet API Service
//

import Foundation

class MasterAPIService {
    private let baseURL: String
    
    init(baseURL: String = "https://api.tigerwallet.io/master") {
        self.baseURL = baseURL
    }
    
    // MARK: - Master Wallet APIs
    func getMasterWallet() async throws -> MasterWallet {
        // Implementation for master wallet
        return try await request(endpoint: "/api/v1/master/wallet")
    }
    
    func createSubWallet(name: String, chain: String, permissions: [String]) async throws -> SubWallet {
        let body = try JSONEncoder().encode(["name": name, "chain": chain, "permissions": permissions])
        return try await request(endpoint: "/api/v1/master/wallets", method: "POST", body: body)
    }
    
    func getSubWallets() async throws -> [SubWallet] {
        return try await request(endpoint: "/api/v1/master/wallets")
    }
    
    func getSubWallet(id: String) async throws -> SubWallet {
        return try await request(endpoint: "/api/v1/master/wallets/\(id)")
    }
    
    func updateSubWallet(id: String, name: String, permissions: [String]) async throws -> SubWallet {
        let body = try JSONEncoder().encode(["name": name, "permissions": permissions])
        return try await request(endpoint: "/api/v1/master/wallets/\(id)", method: "PUT", body: body)
    }
    
    func deleteSubWallet(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master/wallets/\(id)", method: "DELETE")
    }
    
    // MARK: - Auto-Sign APIs
    func getAutoSignRules() async throws -> [AutoSignRule] {
        return try await request(endpoint: "/api/v1/master/auto-sign/rules")
    }
    
    func createAutoSignRule(rule: AutoSignRule) async throws -> AutoSignRule {
        let body = try JSONEncoder().encode(rule)
        return try await request(endpoint: "/api/v1/master/auto-sign/rules", method: "POST", body: body)
    }
    
    func updateAutoSignRule(id: String, rule: AutoSignRule) async throws -> AutoSignRule {
        let body = try JSONEncoder().encode(rule)
        return try await request(endpoint: "/api/v1/master/auto-sign/rules/\(id)", method: "PUT", body: body)
    }
    
    func deleteAutoSignRule(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master/auto-sign/rules/\(id)", method: "DELETE")
    }
    
    func approveTransaction(id: String) async throws -> Transaction {
        return try await request(endpoint: "/api/v1/master/transactions/\(id)/approve", method: "POST")
    }
    
    func rejectTransaction(id: String, reason: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master/transactions/\(id)/reject", method: "POST", body: try JSONEncoder().encode(["reason": reason]))
    }
    
    // MARK: - User Management APIs
    func getUsers() async throws -> [MasterUser] {
        return try await request(endpoint: "/api/v1/master/users")
    }
    
    func getUser(id: String) async throws -> MasterUser {
        return try await request(endpoint: "/api/v1/master/users/\(id)")
    }
    
    func createUser(user: CreateUserRequest) async throws -> MasterUser {
        let body = try JSONEncoder().encode(user)
        return try await request(endpoint: "/api/v1/master/users", method: "POST", body: body)
    }
    
    func updateUserPermissions(userId: String, permissions: MasterPermissions) async throws -> MasterUser {
        let body = try JSONEncoder().encode(permissions)
        return try await request(endpoint: "/api/v1/master/users/\(userId)/permissions", method: "PUT", body: body)
    }
    
    // MARK: - Transaction APIs
    func getTransactions(status: String? = nil, limit: Int = 50) async throws -> [MasterTransaction] {
        var endpoint = "/api/v1/master/transactions?limit=\(limit)"
        if let status = status {
            endpoint += "&status=\(status)"
        }
        return try await request(endpoint: endpoint)
    }
    
    func getTransaction(id: String) async throws -> MasterTransaction {
        return try await request(endpoint: "/api/v1/master/transactions/\(id)")
    }
    
    // MARK: - Analytics APIs
    func getAnalytics() async throws -> MasterAnalytics {
        return try await request(endpoint: "/api/v1/master/analytics")
    }
    
    func getVolumeHistory(period: String) async throws -> [VolumeData] {
        return try await request(endpoint: "/api/v1/master/analytics/volume?period=\(period)")
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

struct MasterWallet: Codable {
    let id: String
    let address: String
    let publicKey: String
    let name: String
    let createdAt: Date
    var totalValueUSD: Double
}

struct SubWallet: Codable, Identifiable {
    let id: String
    let name: String
    let address: String
    var balanceUSD: Double
    var status: String
    var permissions: [String]
}

struct AutoSignRule: Codable {
    let id: String
    let name: String
    let maxAmount: Double
    let chain: String
    let enabled: Bool
    let createdAt: Date
}

struct MasterUser: Codable {
    let id: String
    let email: String
    let name: String
    let permissions: MasterPermissions
    let createdAt: Date
    var lastLoginAt: Date?
}

struct CreateUserRequest: Codable {
    let email: String
    let name: String
    let permissions: MasterPermissions
}

struct MasterPermissions: Codable {
    var canAutoSign: Bool
    var canAirdrop: Bool
    var canClaim: Bool
    var canAdjustFees: Bool
    var maxTransactionLimit: Double
}

struct MasterTransaction: Codable, Identifiable {
    let id: String
    let subWalletId: String
    let from: String
    let to: String
    let amount: String
    let chain: String
    let status: String
    let type: String
    let createdAt: Date
    var approvedAt: Date?
}

struct MasterAnalytics: Codable {
    let totalWallets: Int
    let totalVolumeUSD: Double
    let totalTransactions: Int
    let pendingTransactions: Int
}

struct VolumeData: Codable {
    let date: Date
    let volumeUSD: Double
}

// MARK: - Wallet Service
class MasterWalletService {
    private let apiService: MasterAPIService
    
    init(apiService: MasterAPIService) {
        self.apiService = apiService
    }
    
    func getMasterWallet() async throws -> MasterWallet {
        return try await apiService.getMasterWallet()
    }
    
    func getSubWallets() async throws -> [SubWallet] {
        return try await apiService.getSubWallets()
    }
    
    func createSubWallet(name: String, chain: String, permissions: [String]) async throws -> SubWallet {
        return try await apiService.createSubWallet(name: name, chain: chain, permissions: permissions)
    }
}

// MARK: - Auto Sign Service
class AutoSignService {
    private let apiService: MasterAPIService
    
    init(apiService: MasterAPIService) {
        self.apiService = apiService
    }
    
    func getRules() async throws -> [AutoSignRule] {
        return try await apiService.getAutoSignRules()
    }
    
    func createRule(rule: AutoSignRule) async throws -> AutoSignRule {
        return try await apiService.createAutoSignRule(rule: rule)
    }
    
    func approveTransaction(id: String) async throws -> MasterTransaction {
        return try await apiService.approveTransaction(id: id)
    }
    
    func rejectTransaction(id: String, reason: String) async throws {
        try await apiService.rejectTransaction(id: id, reason: reason)
    }
}
