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
