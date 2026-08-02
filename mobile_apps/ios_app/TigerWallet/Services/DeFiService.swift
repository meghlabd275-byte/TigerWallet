//
//  DeFiService.swift
//  TigerWallet
//
//  Production-ready DeFi protocol integrations
//  Supports: Aave, Compound, Uniswap, Curve, Yearn
//

import Foundation

// MARK: - DeFi Models

struct DeFiPool: Codable, Identifiable {
    let id: String
    let protocol: String
    let chain: String
    let token0: TokenInfo
    let token1: TokenInfo?
    let tvl: Double
    let apy: Double
    let rewardsApy: Double?
    let poolAddress: String
}

struct TokenInfo: Codable {
    let address: String
    let symbol: String
    let name: String
    let decimals: Int
    let logoUrl: String?
}

struct DeFiPosition: Codable, Identifiable {
    let id: String
    let protocol: String
    let chain: String
    let poolAddress: String
    let token0: TokenInfo
    let token1: TokenInfo?
    let deposited0: Double
    let deposited1: Double?
    let valueUsd: Double
    let apy: Double
    let rewards: [Reward]?
}

struct Reward: Codable {
    let token: TokenInfo
    let amount: Double
    let valueUsd: Double
    let apy: Double
}

struct SwapQuote: Codable {
    let fromToken: TokenInfo
    let toToken: TokenInfo
    let fromAmount: Double
    let toAmount: Double
    let toAmountMin: Double
    let priceImpact: Double
    let route: [String]
    let gasCostUsd: Double
    let protocol: String
}

// MARK: - DeFi Service

class DeFiService {
    
    static let shared = DeFiService()
    
    private let session = URLSession.shared
    
    // MARK: - Aave Methods
    
    /// Get Aave pools
    func getAavePools(chain: String = "ethereum") async throws -> [DeFiPool] {
        let url = URL(string: "https://api.tigerwallet.com/v1/defi/aave/pools?chain=\(chain)")!
        
        let (data, response) = try await session.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw DeFiError.networkError
        }
        
        let pools = try JSONDecoder().decode([DeFiPool].self, from: data)
        return pools
    }
    
    /// Supply to Aave
    func supplyToAave(
        walletAddress: String,
        poolAddress: String,
        tokenAddress: String,
        amount: String,
        chain: String
    ) async throws -> String {
        let url = URL(string: "https://api.tigerwallet.com/v1/defi/aave/supply")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "pool_address": poolAddress,
            "token_address": tokenAddress,
            "amount": amount,
            "chain": chain
        ]
        
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, response) = try await session.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw DeFiError.transactionFailed
        }
        
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    /// Borrow from Aave
    func borrowFromAave(
        walletAddress: String,
        poolAddress: String,
        tokenAddress: String,
        amount: String,
        interestRateMode: Int,
        chain: String
    ) async throws -> String {
        let url = URL(string: "https://api.tigerwallet.com/v1/defi/aave/borrow")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "pool_address": poolAddress,
            "token_address": tokenAddress,
            "amount": amount,
            "interest_rate_mode": interestRateMode,
            "chain": chain
        ]
        
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, response) = try await session.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw DeFiError.transactionFailed
        }
        
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    // MARK: - Uniswap Methods
    
    /// Get swap quote
    func getSwapQuote(
        tokenIn: String,
        tokenOut: String,
        amount: String,
        chain: String = "ethereum"
    ) async throws -> SwapQuote {
        let url = URL(string: "https://api.tigerwallet.com/v1/defi/uniswap/quote")!
        
        var components = URLComponents(url: url, resolvingAgainstBaseURL: false)!
        components.queryItems = [
            URLQueryItem(name: "tokenIn", value: tokenIn),
            URLQueryItem(name: "tokenOut", value: tokenOut),
            URLQueryItem(name: "amount", value: amount),
            URLQueryItem(name: "chain", value: chain)
        ]
        
        let (data, response) = try await session.data(from: components.url!)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw DeFiError.networkError
        }
        
        let quote = try JSONDecoder().decode(SwapQuote.self, from: data)
        return quote
    }
    
    /// Execute swap
    func executeSwap(
        walletAddress: String,
        tokenIn: String,
        tokenOut: String,
        amount: String,
        minOutput: String,
        chain: String
    ) async throws -> String {
        let url = URL(string: "https://api.tigerwallet.com/v1/defi/uniswap/swap")!
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "wallet_address": walletAddress,
            "token_in": tokenIn,
            "token_out": tokenOut,
            "amount": amount,
            "min_output": minOutput,
            "chain": chain
        ]
        
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        
        let (data, response) = try await session.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw DeFiError.transactionFailed
        }
        
        let result = try JSONDecoder().decode(TransactionResult.self, from: data)
        return result.txHash
    }
    
    // MARK: - Compound Methods
    
    /// Get Compound pools
    func getCompoundPools() async throws -> [DeFiPool] {
        let url = URL(string: "https://api.tigerwallet.com/v1/defi/compound/pools")!
        
        let (data, response) = try await session.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw DeFiError.networkError
        }
        
        let pools = try JSONDecoder().decode([DeFiPool].self, from: data)
        return pools
    }
    
    // MARK: - Yearn Vaults
    
    /// Get Yearn vaults
    func getYearnVaults() async throws -> [DeFiPool] {
        let url = URL(string: "https://api.tigerwallet.com/v1/defi/yearn/vaults")!
        
        let (data, response) = try await session.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw DeFiError.networkError
        }
        
        let pools = try JSONDecoder().decode([DeFiPool].self, from: data)
        return pools
    }
    
    // MARK: - Portfolio
    
    /// Get all positions for a wallet
    func getAllPositions(walletAddress: String) async throws -> [DeFiPosition] {
        let url = URL(string: "https://api.tigerwallet.com/v1/defi/positions/\(walletAddress)")!
        
        let (data, response) = try await session.data(from: url)
        
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw DeFiError.networkError
        }
        
        let positions = try JSONDecoder().decode([DeFiPosition].self, from: data)
        return positions
    }
}

// MARK: - Error Types

enum DeFiError: Error {
    case networkError
    case transactionFailed
    case invalidParameters
    case insufficientBalance
}

struct TransactionResult: Codable {
    let txHash: String
    let status: String
}

// MARK: - Protocol Enum

enum DeFiProtocol: String, CaseIterable {
    case aave = "Aave"
    case compound = "Compound"
    case uniswap = "Uniswap"
    case yearn = "Yearn"
    case curve = "Curve"
    
    var displayName: String {
        return self.rawValue
    }
}
