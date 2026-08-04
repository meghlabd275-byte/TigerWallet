import Foundation

// MARK: - User Models
struct User: Codable, Identifiable {
    let id: String
    let userId: String
    let username: String
    let email: String
    let phone: String?
    var status: String
    var tier: Int
    var kycStatus: String
    var kycLevel: Int
    let isEmailVerified: Bool
    let isPhoneVerified: Bool
    let whiteLabelId: String?
    let referralCode: String?
    let createdAt: String
    let lastLogin: String?
    
    enum CodingKeys: String, CodingKey {
        case id, username, email, phone, status, tier
        case userId = "user_id"
        case kycStatus = "kyc_status"
        case kycLevel = "kyc_level"
        case isEmailVerified = "is_email_verified"
        case isPhoneVerified = "is_phone_verified"
        case whiteLabelId = "white_label_id"
        case referralCode = "referral_code"
        case createdAt = "created_at"
        case lastLogin = "last_login"
    }
}

struct UsersResponse: Codable {
    let data: [User]
    let meta: PaginationMeta
}

// MARK: - KYC Models
struct KYCSubmission: Codable, Identifiable {
    let id: String
    let userId: String
    let level: Int
    let documentType: String?
    let documentNumber: String?
    let firstName: String?
    let lastName: String?
    let country: String?
    let address: String?
    var status: String
    let rejectReason: String?
    let reviewedBy: String?
    let reviewedAt: String?
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, level, country, address, status
        case userId = "user_id"
        case documentType = "document_type"
        case documentNumber = "document_number"
        case firstName = "first_name"
        case lastName = "last_name"
        case rejectReason = "reject_reason"
        case reviewedBy = "reviewed_by"
        case reviewedAt = "reviewed_at"
        case createdAt = "created_at"
    }
}

struct KYCListResponse: Codable {
    let data: [KYCSubmission]
    let meta: PaginationMeta
}

// MARK: - Token Models
struct Token: Codable, Identifiable {
    let id: String
    let tokenId: String
    let name: String
    let symbol: String
    let contractAddr: String
    let decimals: Int
    let chainId: String
    let chainName: String
    var isActive: Bool
    var isVerified: Bool
    let isNativeToken: Bool
    let logoUrl: String?
    let website: String?
    var price: Double?
    var priceChange24h: Double?
    var volume24h: Double?
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, name, symbol, decimals, status, price
        case tokenId = "token_id"
        case contractAddr = "contract_addr"
        case chainId = "chain_id"
        case chainName = "chain_name"
        case isActive = "is_active"
        case isVerified = "is_verified"
        case isNativeToken = "is_native_token"
        case logoUrl = "logo_url"
        case website
        case priceChange24h = "price_change_24h"
        case volume24h = "volume_24h"
        case createdAt = "created_at"
    }
}

struct TokensResponse: Codable {
    let data: [Token]
    let meta: PaginationMeta
}

// MARK: - Trading Pair Models
struct TradingPair: Codable, Identifiable {
    let id: String
    let pairId: String
    let baseSymbol: String
    let quoteSymbol: String
    let pairName: String
    let chainId: String
    var status: String
    var tradingEnabled: Bool
    let makerFee: Double
    let takerFee: Double
    var currentPrice: Double?
    var priceChange24h: Double?
    var volume24h: Double?
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, status, price
        case pairId = "pair_id"
        case baseSymbol = "base_symbol"
        case quoteSymbol = "quote_symbol"
        case pairName = "pair_name"
        case chainId = "chain_id"
        case tradingEnabled = "trading_enabled"
        case makerFee = "maker_fee"
        case takerFee = "taker_fee"
        case currentPrice = "current_price"
        case priceChange24h = "price_change_24h"
        case volume24h = "volume_24h"
        case createdAt = "created_at"
    }
}

struct PairsResponse: Codable {
    let data: [TradingPair]
    let meta: PaginationMeta
}

// MARK: - Transaction Models
struct Transaction: Codable, Identifiable {
    let id: String
    let txHash: String?
    let userId: String
    let type: String
    var status: String
    let chainId: String
    let fromAddress: String?
    let toAddress: String?
    let amount: String
    let fee: String
    let blockNumber: Int?
    let createdAt: String
    let completedAt: String?
    
    enum CodingKeys: String, CodingKey {
        case id, type, status, amount, fee
        case txHash = "tx_hash"
        case userId = "user_id"
        case chainId = "chain_id"
        case fromAddress = "from_address"
        case toAddress = "to_address"
        case blockNumber = "block_number"
        case createdAt = "created_at"
        case completedAt = "completed_at"
    }
}

struct TransactionsResponse: Codable {
    let data: [Transaction]
    let meta: PaginationMeta
}

// MARK: - Withdrawal Models
struct Withdrawal: Codable, Identifiable {
    let id: String
    let userId: String
    let walletAddress: String
    let chainId: String
    let token: String
    let amount: String
    let fee: String
    let total: String
    var status: String
    let txHash: String?
    let rejectionReason: String?
    let createdAt: String
    let processedAt: String?
    
    enum CodingKeys: String, CodingKey {
        case id, status, token, amount, fee, total
        case userId = "user_id"
        case walletAddress = "wallet_address"
        case chainId = "chain_id"
        case txHash = "tx_hash"
        case rejectionReason = "rejection_reason"
        case createdAt = "created_at"
        case processedAt = "processed_at"
    }
}

struct WithdrawalsResponse: Codable {
    let data: [Withdrawal]
    let meta: PaginationMeta
}

// MARK: - Chain Models
struct Chain: Codable, Identifiable {
    let id: String
    let chainId: Int
    let name: String
    let symbol: String
    let type: String
    let rpcUrls: [String]
    let explorerUrls: [String]
    var isActive: Bool
    let isTestnet: Bool
    let confirmations: Int
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, name, symbol, type, status
        case chainId = "chain_id"
        case rpcUrls = "rpc_urls"
        case explorerUrls = "explorer_urls"
        case isActive = "is_active"
        case isTestnet = "is_testnet"
        case confirmations
        case createdAt = "created_at"
    }
}

struct ChainsResponse: Codable {
    let data: [Chain]
}

// MARK: - Fee Models
struct Fee: Codable, Identifiable {
    let id: String
    let feeType: String
    let chainId: String?
    let tokenId: String?
    var feePercent: Double
    var feeFixed: Double
    var minFee: Double
    var maxFee: Double?
    var isActive: Bool
    
    enum CodingKeys: String, CodingKey {
        case id, status
        case feeType = "fee_type"
        case chainId = "chain_id"
        case tokenId = "token_id"
        case feePercent = "fee_percent"
        case feeFixed = "fee_fixed"
        case minFee = "min_fee"
        case maxFee = "max_fee"
        case isActive = "is_active"
    }
}

struct FeesResponse: Codable {
    let data: [Fee]
}

// MARK: - White Label Models
struct WhiteLabel: Codable, Identifiable {
    let id: String
    let clientId: String
    let name: String
    let domain: String
    var domainVerified: Bool
    var status: String
    let primaryColor: String?
    let secondaryColor: String?
    var platformFeePercent: Double
    let maxUsers: Int
    var currentUsers: Int
    let createdAt: String
    let activatedAt: String?
    let expiresAt: String?
    
    enum CodingKeys: String, CodingKey {
        case id, name, domain, status
        case clientId = "client_id"
        case domainVerified = "domain_verified"
        case primaryColor = "primary_color"
        case secondaryColor = "secondary_color"
        case platformFeePercent = "platform_fee_percent"
        case maxUsers = "max_users"
        case currentUsers = "current_users"
        case createdAt = "created_at"
        case activatedAt = "activated_at"
        case expiresAt = "expires_at"
    }
}

struct WhiteLabelsResponse: Codable {
    let data: [WhiteLabel]
    let meta: PaginationMeta
}

// MARK: - Dashboard Models
struct DashboardStats: Codable {
    let totalUsers: Int
    let activeUsers: Int
    let suspendedUsers: Int
    let kycPending: Int
    let totalTransactions: Int
    let volume24h: Double
    let revenue24h: Double
    let newUsers24h: Int
    let newTransactions24h: Int
    
    enum CodingKeys: String, CodingKey {
        case totalUsers = "total_users"
        case activeUsers = "active_users"
        case suspendedUsers = "suspended_users"
        case kycPending = "kyc_pending"
        case totalTransactions = "total_transactions"
        case volume24h = "volume_24h"
        case revenue24h = "revenue_24h"
        case newUsers24h = "new_users_24h"
        case newTransactions24h = "new_transactions_24h"
    }
}

// MARK: - Common Models
struct PaginationMeta: Codable {
    let page: Int
    let limit: Int
    let total: Int
    let totalPages: Int
    
    enum CodingKeys: String, CodingKey {
        case page, limit, total
        case totalPages = "total_pages"
    }
}

struct ApiResponse<T: Codable>: Codable {
    let success: Bool
    let data: T?
    let error: String?
    let meta: PaginationMeta?
}

// MARK: - Admin Models
struct Admin: Codable, Identifiable {
    let id: String
    let username: String
    let email: String
    let role: String
    var status: String
    var twoFactorEnabled: Bool
    var securityLevel: Int
    let createdAt: String
    let lastLogin: String?
    
    enum CodingKeys: String, CodingKey {
        case id, username, email, role, status
        case twoFactorEnabled = "two_factor_enabled"
        case securityLevel = "security_level"
        case createdAt = "created_at"
        case lastLogin = "last_login"
    }
}

struct LoginResponse: Codable {
    let token: String
    let refreshToken: String
    let expiresIn: Int
    let admin: Admin
    
    enum CodingKeys: String, CodingKey {
        case token
        case refreshToken = "refresh_token"
        case expiresIn = "expires_in"
        case admin
    }
}

// MARK: - Ticket Models
struct Ticket: Codable, Identifiable {
    let id: String
    let ticketId: String
    let subject: String
    let description: String
    let category: String
    let priority: String
    var status: String
    let userId: String
    let adminId: String?
    let createdAt: String
    let updatedAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, subject, description, category, priority, status
        case ticketId = "ticket_id"
        case userId = "user_id"
        case adminId = "admin_id"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct TicketsResponse: Codable {
    let data: [Ticket]
    let meta: PaginationMeta
}

// MARK: - Knowledge Base Models
struct Article: Codable, Identifiable {
    let id: String
    let articleId: String
    let title: String
    let content: String
    let category: String
    let tags: [String]
    let status: String
    let viewCount: Int
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, title, content, category, tags, status
        case articleId = "article_id"
        case viewCount = "view_count"
        case createdAt = "created_at"
    }
}

struct ArticlesResponse: Codable {
    let data: [Article]
}

// MARK: - Workflow Models
struct Workflow: Codable, Identifiable {
    let id: String
    let workflowId: String
    let name: String
    let description: String
    let resourceType: String
    let approvers: [String]
    let minApprovals: Int
    let status: String
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, name, description, resourceType, approvers, minApprovals, status
        case workflowId = "workflow_id"
        case resourceType = "resource_type"
        case createdAt = "created_at"
    }
}

struct WorkflowsResponse: Codable {
    let data: [Workflow]
}

struct ApprovalRequest: Codable, Identifiable {
    let id: String
    let workflowId: String
    let requesterId: String
    let resourceType: String
    let resourceId: String
    let status: String
    let approvedBy: [String]
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, requesterId, resourceType, resourceId, status, approvedBy
        case workflowId = "workflow_id"
        case createdAt = "created_at"
    }
}

struct ApprovalRequestsResponse: Codable {
    let data: [ApprovalRequest]
    let meta: PaginationMeta
}

// MARK: - Dashboard Models
struct ComplianceDashboard: Codable {
    let totalKYC: Int
    let pendingKYC: Int
    let approvedKYC: Int
    let rejectedKYC: Int
    let highRiskUsers: Int
    let suspiciousActivity: Int
    let transactionsFlagged: Int
    let complianceScore: Double
    
    enum CodingKeys: String, CodingKey {
        case totalKYC, pendingKYC, approvedKYC, rejectedKYC
        case highRiskUsers, suspiciousActivity, transactionsFlagged, complianceScore
    }
}

struct FinanceDashboard: Codable {
    let totalRevenue: Double
    let revenueToday: Double
    let revenueThisMonth: Double
    let tradingVolume: Double
    let feesCollected: Double
    let pendingWithdrawals: Double
    
    enum CodingKeys: String, CodingKey {
        case totalRevenue, revenueToday, revenueThisMonth, tradingVolume, feesCollected, pendingWithdrawals
    }
}

struct SecurityDashboard: Codable {
    let failedLogins: Int
    let activeSessions: Int
    let suspiciousIPs: Int
    let securityEvents: Int
    let blockedIPs: Int
    let twoFactorEnabled: Int
    let securityScore: Double
    
    enum CodingKeys: String, CodingKey {
        case failedLogins, activeSessions, suspiciousIPs, securityEvents, blockedIPs, twoFactorEnabled, securityScore
    }
}

// MARK: - Notification Models
struct Notification: Codable, Identifiable {
    let id: String
    let title: String
    let message: String
    let type: String
    var isRead: Bool
    let priority: String
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, title, message, type, priority
        case isRead = "is_read"
        case createdAt = "created_at"
    }
}

struct NotificationsResponse: Codable {
    let data: [Notification]
    let meta: PaginationMeta
}

// MARK: - API Key Models
struct APIKey: Codable, Identifiable {
    let id: String
    let keyId: String
    let keyPrefix: String
    let name: String
    let permissions: [String]
    let rateLimit: Int
    var isActive: Bool
    let expiresAt: String?
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, name, permissions, rateLimit, isActive, expiresAt, createdAt
        case keyId = "key_id"
        case keyPrefix = "key_prefix"
    }
}

struct APIKeysResponse: Codable {
    let data: [APIKey]
}

// MARK: - Webhook Models
struct Webhook: Codable, Identifiable {
    let id: String
    let webhookId: String
    let name: String
    let url: String
    let events: [String]
    var isActive: Bool
    let retryCount: Int
    let lastStatus: Int?
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, name, url, events, isActive, retryCount, lastStatus, createdAt
        case webhookId = "webhook_id"
    }
}

struct WebhooksResponse: Codable {
    let data: [Webhook]
}

// MARK: - Market Maker Bot Models
struct MarketMakerBot: Codable, Identifiable {
    let id: String
    let botName: String
    let botType: String
    var status: String
    let userId: String
    let connectedDEXs: Int
    let totalPnL: Double
    let totalVolume: Double
    let totalOrders: Int
    let avgLatency: Double
    let createdAt: String
    
    enum CodingKeys: String, CodingKey {
        case id, botName, botType, status, userId, connectedDEXs, totalPnL, totalVolume, totalOrders, avgLatency, createdAt
    }
}

struct BotsResponse: Codable {
    let data: [MarketMakerBot]
    let meta: PaginationMeta
}
