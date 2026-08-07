/**
 * TigerWallet Admin - Models
 */

import Foundation

struct User: Codable {
    let id: String
    let email: String
    let username: String
    let role: UserRole
    var status: UserStatus
    var kycLevel: Int
    var createdAt: Date
    
    enum UserRole: String, Codable {
        case superAdmin = "super_admin"
        case admin = "admin"
        case support = "support"
        case analyst = "analyst"
        case viewer = "viewer"
    }
    
    enum UserStatus: String, Codable {
        case active
        case suspended
        case banned
        case pending
    }
}

struct KycRequest: Codable {
    let id: String
    let userId: String
    let level: Int
    let documentType: String
    let status: KycStatus
    let firstName: String
    let lastName: String
    let country: String
    let createdAt: Date
    
    enum KycStatus: String, Codable {
        case pending
        case approved
        case rejected
    }
}

struct Transaction: Codable {
    let id: String
    let userId: String
    let type: TransactionType
    let amount: String
    let currency: String
    let status: TransactionStatus
    let fromAddress: String?
    let toAddress: String?
    let txHash: String?
    let timestamp: Date
    
    enum TransactionType: String, Codable {
        case deposit
        case withdraw
        case transfer
        case swap
    }
    
    enum TransactionStatus: String, Codable {
        case pending
        case confirmed
        case failed
        case flagged
    }
}

struct Token: Codable {
    let id: String
    let name: String
    let symbol: String
    let contractAddress: String?
    let decimals: Int
    let isActive: Bool
    let chainId: Int
    let price: String?
    let volume24h: String?
}

struct TradingPair: Codable {
    let id: String
    let baseToken: Token
    let quoteToken: Token
    let pairName: String
    let price: String?
    let volume24h: String?
    let liquidity: String?
    let status: PairStatus
    
    enum PairStatus: String, Codable {
        case active
        case halted
        case suspended
    }
}

struct Blockchain: Codable {
    let id: String
    let name: String
    let symbol: String
    let chainId: Int
    let isEvm: Bool
    let rpcUrl: String
    let explorerUrl: String?
    let isActive: Bool
}

struct WhiteLabel: Codable {
    let id: String
    let companyName: String
    let domain: String
    let status: WhiteLabelStatus
    let maxUsers: Int
    let platformFeePercent: Double
    let customFeePercent: Double
    let contactEmail: String
    
    enum WhiteLabelStatus: String, Codable {
        case pending
        case active
        case suspended
    }
}

struct Ticket: Codable {
    let id: String
    let title: String
    let description: String
    let type: TicketType
    let priority: TicketPriority
    let status: TicketStatus
    let createdBy: String
    let assignedTo: String?
    let createdAt: Date
    
    enum TicketType: String, Codable {
        case withdrawal
        case kyc
        case account
        case transaction
        case other
    }
    
    enum TicketPriority: String, Codable {
        case low
        case medium
        case high
        case urgent
    }
    
    enum TicketStatus: String, Codable {
        case open
        case inProgress
        case resolved
        case closed
    }
}
