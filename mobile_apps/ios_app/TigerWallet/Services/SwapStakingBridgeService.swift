//
//  SwapService.swift
//  TigerWallet
//
//  Token Swap Service for iOS
//

import Foundation

class SwapService {
    static let shared = SwapService()
    private let baseURL = "https://api.tigerwallet.com/v1/swap"
    
    private init() {}
    
    func getTokens(chain: String = "ethereum") async throws -> [TokenInfo] {
        let url = URL(string: "\(baseURL)/tokens?chain=\(chain)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([TokenInfo].self, from: data)
    }
    
    func getQuote(fromToken: String, toToken: String, amount: String, slippage: Double = 0.5) async throws -> SwapQuote? {
        var request = URLRequest(url: URL(string: "\(baseURL)/quote")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "fromToken": fromToken,
            "toToken": toToken,
            "amount": amount,
            "slippage": slippage
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        return try JSONDecoder().decode(SwapQuote.self, from: data)
    }
    
    func executeSwap(walletId: String, fromToken: String, toToken: String, amount: String, minReceived: String, route: [String]) async throws -> String {
        var request = URLRequest(url: URL(string: "\(baseURL)/execute")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "walletId": walletId,
            "fromToken": fromToken,
            "toToken": toToken,
            "amount": amount,
            "minReceived": minReceived,
            "route": route
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
}

class StakingService {
    static let shared = StakingService()
    private let baseURL = "https://api.tigerwallet.com/v1/staking"
    
    private init() {}
    
    func getValidators(chain: String = "ethereum") async throws -> [Validator] {
        let url = URL(string: "\(baseURL)/validators?chain=\(chain)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([Validator].self, from: data)
    }
    
    func delegate(walletAddress: String, validatorAddress: String, amount: String, chain: String = "ethereum") async throws -> String {
        var request = URLRequest(url: URL(string: "\(baseURL)/delegate")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "validator_address": validatorAddress,
            "amount": amount,
            "chain": chain
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    func undelegate(walletAddress: String, amount: String, chain: String = "ethereum") async throws -> String {
        var request = URLRequest(url: URL(string: "\(baseURL)/undelegate")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "amount": amount,
            "chain": chain
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    func getRewards(walletAddress: String, chain: String = "ethereum") async throws -> StakingRewards? {
        let url = URL(string: "\(baseURL)/rewards/\(walletAddress)?chain=\(chain)")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode(StakingRewards.self, from: data)
    }
}

class BridgeService {
    static let shared = BridgeService()
    private let baseURL = "https://api.tigerwallet.com/v1/bridge"
    
    private init() {}
    
    func getQuotes(fromChain: String, toChain: String, fromToken: String, toToken: String, amount: String) async throws -> [BridgeQuote] {
        var request = URLRequest(url: URL(string: "\(baseURL)/quotes")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "from_chain": fromChain,
            "to_chain": toChain,
            "from_token": fromToken,
            "to_token": toToken,
            "amount": amount
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        return try JSONDecoder().decode([BridgeQuote].self, from: data)
    }
    
    func executeBridge(walletId: String, fromChain: String, toChain: String, fromToken: String, toToken: String, amount: String, bridgeRoute: [String]) async throws -> String {
        var request = URLRequest(url: URL(string: "\(baseURL)/execute")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_id": walletId,
            "from_chain": fromChain,
            "to_chain": toChain,
            "from_token": fromToken,
            "to_token": toToken,
            "amount": amount,
            "bridge_route": bridgeRoute
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, _) = try await URLSession.shared.data(for: request)
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    func getSupportedChains() async throws -> [ChainInfo] {
        let url = URL(string: "\(baseURL)/chains")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([ChainInfo].self, from: data)
    }
}

// Models
struct TokenInfo: Codable {
    let address: String
    let symbol: String
    let name: String
    let decimals: Int
    let logoUrl: String?
}

struct SwapQuote: Codable {
    let fromToken: String
    let toToken: String
    let fromAmount: String
    let toAmount: String
    let toAmountMin: String
    let priceImpact: Double
    let route: [String]
    let gasCost: String
}

struct Validator: Codable, Identifiable {
    let id: String
    let address: String
    let name: String
    let commission: Double
    let apy: Double
    let totalStaked: String
    let delegators: Int
    let chain: String
}

struct StakingRewards: Codable {
    let pendingRewards: String
    let lastClaimed: String?
    let totalClaimed: String
}

struct BridgeQuote: Codable {
    let bridge: String
    let fromChain: String
    let toChain: String
    let fromToken: String
    let toToken: String
    let fromAmount: String
    let toAmount: String
    let estimatedTime: Int
    let fee: String
}

struct ChainInfo: Codable, Identifiable {
    let id: String
    let name: String
    let symbol: String
    let chainId: Int
}

struct TransactionResult: Codable {
    let txHash: String
}
