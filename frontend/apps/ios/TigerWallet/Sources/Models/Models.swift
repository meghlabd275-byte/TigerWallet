//
//  Models.swift
//  TigerWallet - Data Models
//

import Foundation

// MARK: - Chain
enum Chain: String, CaseIterable, Codable {
    case ethereum = "ethereum"
    case bsc = "bsc"
    case polygon = "polygon"
    case avalanche = "avalanche"
    case arbitrum = "arbitrum"
    case optimism = "optimism"
    case base = "base"
    
    var name: String {
        switch self {
        case .ethereum: return "Ethereum"
        case .bsc: return "BNB Chain"
        case .polygon: return "Polygon"
        case .avalanche: return "Avalanche"
        case .arbitrum: return "Arbitrum"
        case .optimism: return "Optimism"
        case .base: return "Base"
        }
    }
    
    var symbol: String {
        switch self {
        case .ethereum: return "ETH"
        case .bsc: return "BNB"
        case .polygon: return "MATIC"
        case .avalanche: return "AVAX"
        case .arbitrum, .optimism, .base: return "ETH"
        }
    }
    
    var icon: String {
        switch self {
        case .ethereum: return "⬡"
        case .bsc: return "🟡"
        case .polygon: return "🟣"
        case .avalanche: return "🔺"
        case .arbitrum: return "🔵"
        case .optimism: return "🔴"
        case .base: return "⚪"
        }
    }
    
    var chainId: Int {
        switch self {
        case .ethereum: return 1
        case .bsc: return 56
        case .polygon: return 137
        case .avalanche: return 43114
        case .arbitrum: return 42161
        case .optimism: return 10
        case .base: return 8453
        }
    }
}

// MARK: - Wallet
struct Wallet: Codable, Identifiable {
    let id: String
    let address: String
    let publicKey: String
    var tokens: [Token]
    let chain: Chain
    var totalBalance: Double
    var nativeBalance: String
    
    init(id: String = UUID().uuidString, address: String, publicKey: String = "", tokens: [Token] = [], chain: Chain, totalBalance: Double = 0, nativeBalance: String = "0.0") {
        self.id = id
        self.address = address
        self.publicKey = publicKey
        self.tokens = tokens
        self.chain = chain
        self.totalBalance = totalBalance
        self.nativeBalance = nativeBalance
    }
}

// MARK: - Token
struct Token: Codable, Identifiable {
    let id: String
    let address: String
    let name: String
    let symbol: String
    let decimals: Int
    let logoURL: String
    var balance: String
    var balanceUSD: Double
    var price: Double
    var change24h: Double
    
    init(id: String = UUID().uuidString, address: String, name: String, symbol: String, decimals: Int = 18, logoURL: String = "", balance: String = "0", balanceUSD: Double = 0, price: Double = 0, change24h: Double = 0) {
        self.id = id
        self.address = address
        self.name = name
        self.symbol = symbol
        self.decimals = decimals
        self.logoURL = logoURL
        self.balance = balance
        self.balanceUSD = balanceUSD
        self.price = price
        self.change24h = change24h
    }
}

// MARK: - Transaction
struct Transaction: Codable, Identifiable {
    let id: String
    let hash: String
    let from: String
    let to: String
    let amount: String
    let amountUSD: Double
    let fee: String
    let status: TransactionStatus
    let type: TransactionType
    let timestamp: Date
    let chain: Chain
    
    enum TransactionStatus: String, Codable {
        case pending
        case confirmed
        case failed
    }
    
    enum TransactionType: String, Codable {
        case send
        case receive
        case swap
        case stake
        case unstake
        case bridge
        case approve
    }
}

// MARK: - DApp
struct DApp: Identifiable {
    let id: String
    let name: String
    let icon: String
    let category: String
    let url: URL?
    
    init(id: String = UUID().uuidString, name: String, icon: String, category: String, url: URL? = nil) {
        self.id = id
        self.name = name
        self.icon = icon
        self.category = category
        self.url = url
    }
}

// MARK: - Swap Quote
struct SwapQuote: Codable {
    let id: String
    let fromToken: Token
    let toToken: Token
    let fromAmount: String
    let toAmount: String
    let priceImpact: Double
    let gasLimit: Int
    let route: [SwapRoute]
    let expiresAt: Date
    
    struct SwapRoute: Codable {
        let protocol: String
        let fromToken: String
        let toToken: String
        let poolAddress: String
        let poolFee: Int
    }
}

// MARK: - Gas Data
struct GasData: Codable {
    let chainId: Int
    let gasPriceGwei: Double
    let gasLimit: Int
    let estimatedGas: Int
    let maxFeePerGas: Int
    let maxPriorityFeePerGas: Int
    let networkCongestion: String
    let timestamp: Date
}

// MARK: - Network Status
struct NetworkStatus: Codable {
    let chain: Chain
    let name: String
    let symbol: String
    let blockNumber: Int
    let blockTimeMs: Int
    let gasLimit: Int
    let networkStatus: String
    let lastSynced: Date
}

// MARK: - API Response
struct APIResponse<T: Codable>: Codable {
    let success: Bool
    let data: T?
    let error: APIError?
}

struct APIError: Codable {
    let code: String
    let message: String
}

// MARK: - Price Data
struct PriceData: Codable {
    let tokenAddress: String
    let priceUSD: Double
    let priceETH: Double
    let change24h: Double
    let volume24h: Double
    let marketCap: Double
    let timestamp: Date
}

// MARK: - Staking Position
struct StakingPosition: Codable, Identifiable {
    let id: String
    let validator: String
    let amount: String
    let rewardAmount: String
    let apy: Double
    let status: StakingStatus
    let createdAt: Date
    
    enum StakingStatus: String, Codable {
        case active
        case unbonding
        case withdrawn
    }
}

// MARK: - NFT
struct NFT: Codable, Identifiable {
    let id: String
    let collectionAddress: String
    let tokenId: String
    let name: String
    let description: String?
    let imageURL: String
    let owner: String
    let standard: String
    let isListed: Bool
    let listingPrice: String?
}

// MARK: - Notification
struct Notification: Codable, Identifiable {
    let id: String
    let title: String
    let body: String
    let type: NotificationType
    let timestamp: Date
    let isRead: Bool
    
    enum NotificationType: String, Codable {
        case transaction
        case price
        case security
        case system
    }
}
