import Foundation

// MARK: - Admin User Models

struct AdminUser: Codable, Identifiable {
    let id: Int64
    let username: String
    let email: String
    let firstName: String?
    let lastName: String?
    let role: AdminRole
    let permissions: [String]
    let status: AdminStatus
    let twoFactorEnabled: Bool
    let lastLoginAt: String?
    let createdAt: String
    let updatedAt: String
    let avatarUrl: String?
    let phone: String?
    let department: String?
    
    var fullName: String {
        if let firstName = firstName, let lastName = lastName {
            return "\(firstName) \(lastName)"
        }
        return firstName ?? username
    }
}

enum AdminRole: String, Codable, CaseIterable {
    case superAdmin = "super_admin"
    case admin = "admin"
    case support = "support"
    case analyst = "analyst"
    case moderator = "moderator"
    
    var displayName: String {
        switch self {
        case .superAdmin: return "Super Admin"
        case .admin: return "Admin"
        case .support: return "Support"
        case .analyst: return "Analyst"
        case .moderator: return "Moderator"
        }
    }
    
    var permissions: [String] {
        switch self {
        case .superAdmin: return ["all"]
        case .admin: return ["users", "transactions", "kyc", "tokens", "fees", "analytics"]
        case .support: return ["users.view", "kyc.review"]
        case .analyst: return ["analytics.view", "reports.view"]
        case .moderator: return ["users.view", "transactions.view"]
        }
    }
}

enum AdminStatus: String, Codable {
    case active
    case suspended
    case inactive
}

// MARK: - Platform User Models

struct PlatformUser: Codable, Identifiable {
    let id: Int64
    let email: String
    let username: String?
    let walletAddress: String?
    let status: UserStatus
    let kycStatus: KYCStatus
    let kycLevel: Int
    let riskScore: Int
    let createdAt: String
    let lastLoginAt: String?
    let registrationIp: String?
    let tags: [String]
    let referredBy: String?
    let whiteLabelId: Int64?
}

enum UserStatus: String, Codable, CaseIterable {
    case active
    case pending
    case suspended
    case banned
    
    var displayName: String {
        rawValue.capitalized
    }
}

enum KYCStatus: String, Codable, CaseIterable {
    case none
    case pending
    case level1 = "level1"
    case level2 = "level2"
    case level3 = "level3"
    case rejected
    
    var displayName: String {
        switch self {
        case .none: return "Not Verified"
        case .pending: return "Pending"
        case .level1: return "Level 1"
        case .level2: return "Level 2"
        case .level3: return "Level 3"
        case .rejected: return "Rejected"
        }
    }
    
    var level: Int {
        switch self {
        case .none: return 0
        case .pending: return 0
        case .level1: return 1
        case .level2: return 2
        case .level3: return 3
        case .rejected: return 0
        }
    }
}

// MARK: - Transaction Models

struct Transaction: Codable, Identifiable {
    let id: Int64
    let hash: String
    let type: TransactionType
    let chain: String
    let fromAddress: String
    let toAddress: String
    let amount: String
    let token: String
    let tokenAmount: String?
    let status: TransactionStatus
    let blockNumber: Int64?
    let gasUsed: String?
    let gasPrice: String?
    let timestamp: String
    let flagged: Bool
    let flagReason: String?
    let userId: Int64
    
    var shortHash: String {
        let hash = self.hash
        if hash.count > 16 {
            return String(hash.prefix(8)) + "..." + String(hash.suffix(8))
        }
        return hash
    }
}

enum TransactionType: String, Codable, CaseIterable {
    case transfer
    case swap
    case stake
    case unstake
    case bridge
    case withdraw
    case deposit
    case mint
    case burn
    
    var displayName: String {
        rawValue.capitalized
    }
    
    var icon: String {
        switch self {
        case .transfer: return "arrow.left.arrow.right"
        case .swap: return "arrow.triangle.2.circlepath"
        case .stake: return "lock.fill"
        case .unstake: return "lock.open.fill"
        case .bridge: return "link"
        case .withdraw: return "arrow.down.circle"
        case .deposit: return "arrow.up.circle"
        case .mint: return "plus.circle"
        case .burn: return "minus.circle"
        }
    }
}

enum TransactionStatus: String, Codable, CaseIterable {
    case pending
    case confirmed
    case failed
    
    var displayName: String {
        rawValue.capitalized
    }
    
    var color: String {
        switch self {
        case .pending: return "orange"
        case .confirmed: return "green"
        case .failed: return "red"
        }
    }
}

// MARK: - KYC Models

struct KYCApplication: Codable, Identifiable {
    let id: Int64
    let userId: Int64
    let userEmail: String
    let level: Int
    let status: KYCApplicationStatus
    let submittedAt: String
    let reviewedAt: String?
    let reviewedBy: String?
    let rejectionReason: String?
    let documents: [KYCDocument]
    let ipAddress: String?
    let notes: String?
}

enum KYCApplicationStatus: String, Codable, CaseIterable {
    case pending
    case approved
    case rejected
    
    var displayName: String {
        rawValue.capitalized
    }
    
    var color: String {
        switch self {
        case .pending: return "orange"
        case .approved: return "green"
        case .rejected: return "red"
        }
    }
}

struct KYCDocument: Codable {
    let type: String
    let url: String
    let status: String
    let verifiedAt: String?
}

// MARK: - Token Models

struct Token: Codable, Identifiable {
    let id: Int64
    let name: String
    let symbol: String
    let contractAddress: String
    let chain: String
    let decimals: Int
    let totalSupply: String
    let logoUrl: String?
    let website: String?
    let description: String?
    let price: String?
    let marketCap: String?
    let volume24h: String?
    let priceChange24h: String?
    let isActive: Bool
    let isVerified: Bool
    let listingFee: String?
    let listedAt: String?
}

struct TokenListingRequest: Codable, Identifiable {
    let id: Int64
    let tokenSymbol: String
    let tokenName: String
    let contractAddress: String
    let chainId: Int64
    let tier: ListingTier
    let status: ListingStatus
    let requesterAddress: String
    let requesterEmail: String
    let oneTimeFee: String
    let monthlyFee: String
    let requestedAt: String
}

enum ListingTier: String, Codable {
    case basic
    case standard
    case premium
    case premiumPlus = "premium_plus"
    
    var displayName: String {
        switch self {
        case .basic: return "Basic"
        case .standard: return "Standard"
        case .premium: return "Premium"
        case .premiumPlus: return "Premium Plus"
        }
    }
}

enum ListingStatus: String, Codable, CaseIterable {
    case pending
    case approved
    case rejected
    
    var displayName: String {
        rawValue.capitalized
    }
}

// MARK: - Withdrawal Models

struct WithdrawalRequest: Codable, Identifiable {
    let id: Int64
    let userId: Int64
    let userEmail: String
    let amount: String
    let token: String
    let chain: String
    let toAddress: String
    let status: WithdrawalStatus
    let approvedAt: String?
    let approvedBy: String?
    let rejectedAt: String?
    let rejectionReason: String?
    let processedAt: String?
    let txHash: String?
    let fee: String?
    let createdAt: String
}

enum WithdrawalStatus: String, Codable, CaseIterable {
    case pending
    case approved
    case rejected
    case processing
    case completed
    case failed
    
    var displayName: String {
        rawValue.capitalized
    }
    
    var color: String {
        switch self {
        case .pending: return "orange"
        case .approved: return "blue"
        case .rejected: return "red"
        case .processing: return "purple"
        case .completed: return "green"
        case .failed: return "red"
        }
    }
}

// MARK: - White Label Models

struct WhiteLabel: Codable, Identifiable {
    let id: Int64
    let name: String
    let slug: String
    let domain: String?
    let logoUrl: String?
    let faviconUrl: String?
    let primaryColor: String
    let secondaryColor: String?
    let status: WhiteLabelStatus
    let contactEmail: String?
    let contactPhone: String?
    let address: String?
    let description: String?
    let features: [String]
    let feeStructure: FeeStructure?
    let createdAt: String
    let expiresAt: String?
}

enum WhiteLabelStatus: String, Codable {
    case active
    case suspended
    case pending
    
    var displayName: String {
        rawValue.capitalized
    }
}

struct FeeStructure: Codable {
    let tradingFee: String
    let withdrawalFee: String
    let depositFee: String
    let listingFee: String
}

// MARK: - Fee Configuration Models

struct FeeConfig: Codable, Identifiable {
    let id: Int64
    let feeType: String
    let chainId: Int64?
    let tokenSymbol: String?
    let feeAmountUsd: String
    let feePercentage: String
    let minFeeUsd: String
    let maxFeeUsd: String?
    let isActive: Bool
}

// MARK: - System Models

struct SystemStatus: Codable {
    let serviceName: String
    let status: String
    let uptime: String
    let latency: String
    let lastCheck: String
    
    var isHealthy: Bool {
        status == "running" || status == "healthy"
    }
}

struct SystemHealth: Codable {
    let status: String
    let uptime: String
    let memoryUsage: String
    let cpuUsage: String
    let diskUsage: String
}

// MARK: - Analytics Models

struct AnalyticsData: Codable {
    let totalUsers: Int64
    let activeUsers: Int64
    let totalVolume: String
    let dailyTransactions: Int64
    let totalFees: String
    let pendingKyc: Int64
    let systemHealth: String
    let timestamp: String
}

// MARK: - Bot Models

struct BotInstance: Codable, Identifiable {
    let id: Int64
    let userId: Int64
    let userEmail: String
    let botType: String
    let name: String
    let status: BotStatus
    let connectedDexs: Int
    let connectedCexs: Int
    let totalPnl: String
    let totalVolume: String
    let totalOrders: Int64
    let avgLatencyUs: Int64
    let createdAt: String
    let lastTradeAt: String?
}

enum BotStatus: String, Codable {
    case running
    case stopped
    case error
    case paused
    
    var displayName: String {
        rawValue.capitalized
    }
    
    var color: String {
        switch self {
        case .running: return "green"
        case .stopped: return "gray"
        case .error: return "red"
        case .paused: return "orange"
        }
    }
}

// MARK: - API Key Models

struct APIKey: Codable, Identifiable {
    let id: Int64
    let name: String
    let key: String
    let adminId: Int64
    let permissions: APIKeyPermissions
    let rateLimit: Int
    let lastUsedAt: String?
    let expiresAt: String?
    let status: String
    let createdAt: String
}

struct APIKeyPermissions: Codable {
    let trading: Bool
    let reading: Bool
    let withdrawal: Bool
}

// MARK: - Blockchain Models

struct Blockchain: Codable, Identifiable {
    let id: String
    let name: String
    let symbol: String
    let chainId: Int64
    let chainIdHex: String?
    let isEvm: Bool
    let isActive: Bool
    let explorerUrl: String?
    let rpcUrl: String?
    let nativeTokenSymbol: String
    let avgGasPriceGwei: Double
}

// MARK: - Pagination

struct PaginatedResponse<T: Codable>: Codable {
    let data: [T]
    let pagination: Pagination
}

struct Pagination: Codable {
    let page: Int
    let limit: Int
    let total: Int64
    let totalPages: Int
    
    var hasNextPage: Bool {
        page < totalPages
    }
    
    var hasPreviousPage: Bool {
        page > 1
    }
}

// MARK: - Notification Models

struct AdminNotification: Codable, Identifiable {
    let id: Int64
    let title: String
    let message: String
    let type: String
    let read: Bool
    let createdAt: String
    let data: [String: String]?
}
