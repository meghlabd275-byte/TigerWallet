//
//  TigerWallet Admin Models - iOS Implementation
//  Complete data models for admin operations
//

import Foundation

// MARK: - Response Wrappers

struct UsersListResponse: Codable {
    let data: [User]
    let total: Int
    let page: Int
    let pageSize: Int
}

struct KYCListResponse: Codable {
    let data: [KYCRequest]
    let total: Int
    let page: Int
    let pageSize: Int
}

struct TransactionsListResponse: Codable {
    let data: [Transaction]
    let total: Int
    let page: Int
    let pageSize: Int
}

struct WithdrawalsListResponse: Codable {
    let data: [Withdrawal]
    let total: Int
    let page: Int
    let pageSize: Int
}

struct TokensListResponse: Codable {
    let data: [Token]
    let total: Int
    let page: Int
    let pageSize: Int
}

struct FeeRulesListResponse: Codable {
    let data: [FeeRule]
    let total: Int
    let page: Int
    let pageSize: Int
}

struct BotsListResponse: Codable {
    let data: [Bot]
    let total: Int
    let page: Int
    let pageSize: Int
}

struct TicketsListResponse: Codable {
    let data: [SupportTicket]
    let total: Int
    let page: Int
    let pageSize: Int
}

// MARK: - Auth Models

struct LoginResponse: Codable {
    let token: String
    let refreshToken: String?
    let admin: Admin?
}

struct Admin: Codable {
    let id: Int
    let email: String
    let username: String
    let role: String
    let twoFactorEnabled: Bool
    let isActive: Bool
    let createdAt: String?
}

// MARK: - User Models

struct User: Codable, Identifiable {
    let id: Int
    let email: String
    let username: String
    let walletAddress: String?
    let status: String
    let kycStatus: String
    let twoFactorEnabled: Bool
    let balance: [String: Double]?
    let country: String?
    let createdAt: String
    let lastLogin: String?
}

struct UserBalance: Codable {
    let token: String
    let available: Double
    let locked: Double
    let total: Double
}

// MARK: - KYC Models

struct KYCRequest: Codable, Identifiable {
    let id: Int
    let userId: Int
    let documentType: String
    let status: String
    let documentUrl: String?
    let submittedAt: String
    let reviewedAt: String?
    let reviewedBy: Int?
}

enum KYCStatus: String, CaseIterable {
    case pending = "pending"
    case submitted = "submitted"
    case inReview = "in_review"
    case approved = "approved"
    case rejected = "rejected"
}

enum DocumentType: String, CaseIterable {
    case passport = "passport"
    case nationalId = "national_id"
    case driversLicense = "drivers_license"
    case utilityBill = "utility_bill"
}

// MARK: - Transaction Models

struct Transaction: Codable, Identifiable {
    let id: Int
    let txHash: String
    let userId: Int
    let type: String
    let amount: Double
    let fee: Double
    let token: String
    let chain: String
    let status: String
    let fromAddress: String?
    let toAddress: String?
    let blockNumber: Int?
    let createdAt: String
    let confirmedAt: String?
}

enum TransactionType: String, CaseIterable {
    case transfer = "transfer"
    case swap = "swap"
    case deposit = "deposit"
    case withdrawal = "withdrawal"
    case trade = "trade"
    case fee = "fee"
}

enum TransactionStatus: String, CaseIterable {
    case pending = "pending"
    case processing = "processing"
    case completed = "completed"
    case failed = "failed"
    case cancelled = "cancelled"
}

// MARK: - Withdrawal Models

struct Withdrawal: Codable, Identifiable {
    let id: Int
    let userId: Int
    let amount: Double
    let token: String
    let address: String
    let status: String
    let fee: Double
    let txHash: String?
    let createdAt: String
    let processedAt: String?
}

enum WithdrawalStatus: String, CaseIterable {
    case pending = "pending"
    case approved = "approved"
    case processing = "processing"
    case completed = "completed"
    case rejected = "rejected"
    case failed = "failed"
}

// MARK: - Token Models

struct Token: Codable, Identifiable {
    let id: Int
    let name: String
    let symbol: String
    let decimals: Int
    let contractAddress: String?
    let isActive: Bool
    let isVerified: Bool
    let logoUrl: String?
    let price: Double?
    let marketCap: Double?
    let createdAt: String
}

struct TokenRequest: Encodable {
    let name: String
    let symbol: String
    let decimals: Int
    let contractAddress: String?
    
    func toDictionary() -> [String: Any] {
        var dict: [String: Any] = [
            "name": name,
            "symbol": symbol,
            "decimals": decimals
        ]
        if let address = contractAddress {
            dict["contract_address"] = address
        }
        return dict
    }
}

// MARK: - Fee Models

struct FeeRule: Codable, Identifiable {
    let id: Int
    let name: String
    let feeType: String
    let feeValue: Double
    let minAmount: Double?
    let maxAmount: Double?
    let isActive: Bool
    let createdAt: String
}

enum FeeType: String, CaseIterable {
    case percentage = "percentage"
    case fixed = "fixed"
    case tiered = "tiered"
}

struct FeeRuleRequest: Encodable {
    let name: String
    let feeType: String
    let feeValue: Double
    let minAmount: Double?
    let maxAmount: Double?
    
    func toDictionary() -> [String: Any] {
        var dict: [String: Any] = [
            "name": name,
            "fee_type": feeType,
            "fee_value": feeValue
        ]
        if let min = minAmount { dict["min_amount"] = min }
        if let max = maxAmount { dict["max_amount"] = max }
        return dict
    }
}

// MARK: - Bot Models

struct Bot: Codable, Identifiable {
    let id: Int
    let name: String
    let botType: String
    let status: String
    let totalPnl: Double
    let totalVolume: Double
    let config: BotConfig?
    let createdAt: String
    let startedAt: String?
}

struct BotConfig: Codable {
    let maxPosition: Double?
    let stopLoss: Double?
    let takeProfit: Double?
    let tradingPairs: [String]?
}

enum BotStatus: String, CaseIterable {
    case stopped = "stopped"
    case running = "running"
    case paused = "paused"
    case error = "error"
}

enum BotType: String, CaseIterable {
    case arbitrage = "arbitrage"
    case grid = "grid"
    case dca = "dca"
    case momentum = "momentum"
    case marketMaker = "market_maker"
}

struct BotRequest: Encodable {
    let name: String
    let botType: String
    let config: [String: Any]?
    
    func toDictionary() -> [String: Any] {
        var dict: [String: Any] = [
            "name": name,
            "bot_type": botType
        ]
        if let config = config {
            dict["config"] = config
        }
        return dict
    }
}

// MARK: - Support Models

struct SupportTicket: Codable, Identifiable {
    let id: Int
    let ticketId: String
    let userId: Int
    let subject: String
    let status: String
    let priority: String
    let category: String?
    let messages: [TicketMessage]?
    let createdAt: String
    let updatedAt: String?
    let closedAt: String?
}

struct TicketMessage: Codable, Identifiable {
    let id: Int
    let userId: Int?
    let adminId: Int?
    let content: String
    let isInternal: Bool
    let attachments: [String]?
    let createdAt: String
}

enum TicketStatus: String, CaseIterable {
    case open = "open"
    case inProgress = "in_progress"
    case pending = "pending"
    case resolved = "resolved"
    case closed = "closed"
}

enum TicketPriority: String, CaseIterable {
    case low = "low"
    case medium = "medium"
    case high = "high"
    case urgent = "urgent"
}

// MARK: - Analytics Models

struct DashboardStats: Codable {
    let totalUsers: Int
    let activeUsers: Int
    let totalVolume: Double
    let todayVolume: Double
    let totalTransactions: Int
    let todayTransactions: Int
    let pendingWithdrawals: Int
    let pendingKYC: Int
    let activeBots: Int
    let totalBots: Int
}

struct VolumeAnalytics: Codable {
    let dailyVolumes: [DailyVolume]
    let totalVolume: Double
    let averageVolume: Double
}

struct DailyVolume: Codable {
    let date: String
    let volume: Double
    let count: Int
}

struct RevenueAnalytics: Codable {
    let totalRevenue: Double
    let period: String
    let byType: [RevenueByType]?
}

struct RevenueByType: Codable {
    let type: String
    let amount: Double
    let percentage: Double
}

// MARK: - Notification Models

struct BroadcastResponse: Codable {
    let totalUsers: Int
    let notified: Int
    let failed: Int
}

// MARK: - Error Response

struct ErrorResponse: Codable {
    let error: String
    let message: String?
    let code: String?
}

// MARK: - Pagination

struct PaginationInfo {
    let currentPage: Int
    let totalPages: Int
    let totalItems: Int
    let itemsPerPage: Int
    
    var hasNextPage: Bool {
        currentPage < totalPages
    }
    
    var hasPreviousPage: Bool {
        currentPage > 1
    }
}

// MARK: - Filter Options

struct UserFilters: Equatable {
    var search: String?
    var status: String?
    var kycStatus: String?
    var dateFrom: String?
    var dateTo: String?
    
    var isEmpty: Bool {
        search == nil && status == nil && kycStatus == nil && dateFrom == nil && dateTo == nil
    }
}

struct TransactionFilters: Equatable {
    var status: String?
    var type: String?
    var token: String?
    var chain: String?
    var dateFrom: String?
    var dateTo: String?
    
    var isEmpty: Bool {
        status == nil && type == nil && token == nil && chain == nil && dateFrom == nil && dateTo == nil
    }
}
