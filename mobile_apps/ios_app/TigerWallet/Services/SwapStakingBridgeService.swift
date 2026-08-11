//
//  SwapService.swift
//  TigerWallet
//
//  Token Swap Service for iOS
//

import Foundation

class SwapService {
    static let shared = SwapService()
    private let baseURL = "http://localhost:8443/api/v1"

    private init() {}

    func getTokens(chain: String = "ethereum") async throws -> [TokenInfo] {
        // The backend exposes token holdings per wallet; there is no global
        // token-list endpoint, so return an honest empty list rather than
        // fabricating token metadata. The quote/swap endpoints accept any
        // 0x token address.
        return []
    }

    func getQuote(fromToken: String, toToken: String, amount: String, chainId: Int = 1) async throws -> SwapQuote? {
        // Real on-chain AMM quote: GET /api/v1/amm/quote runs a Uniswap-V2
        // getAmountsOut eth_call. 503 on RPC failure - never fabricates a number.
        var components = URLComponents(string: "\(baseURL)/amm/quote")!
        components.queryItems = [
            URLQueryItem(name: "chain_id", value: String(chainId)),
            URLQueryItem(name: "token_in", value: fromToken),
            URLQueryItem(name: "token_out", value: toToken),
            URLQueryItem(name: "amount_in", value: amount)
        ]
        let (data, response) = try await URLSession.shared.data(from: components.url!)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else { return nil }
        let amm = try JSONDecoder().decode(AMMQuoteResponse.self, from: data)
        return SwapQuote(
            fromToken: amm.tokenIn,
            toToken: amm.tokenOut,
            fromAmount: amm.amountIn,
            toAmount: amm.amountOut,
            toAmountMin: amm.amountOutWei,
            priceImpact: 0.0,
            route: amm.path ?? [],
            gasCost: "0"
        )
    }

    func executeSwap(walletId: String, fromToken: String, toToken: String, amount: String, minReceived: String, route: [String], chainId: Int = 1) async throws -> String {
        // Step 1: build the swap calldata via the AMM router.
        var swapReq = URLRequest(url: URL(string: "\(baseURL)/amm/swap")!)
        swapReq.httpMethod = "POST"
        swapReq.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let swapBody: [String: Any] = [
            "from": walletId,
            "chain_id": chainId,
            "token_in": fromToken,
            "token_out": toToken,
            "amount_in": amount
        ]
        swapReq.httpBody = try JSONSerialization.data(withJSONObject: swapBody)
        let (swapData, swapResp) = try await URLSession.shared.data(for: swapReq)
        guard let swapHttp = swapResp as? HTTPURLResponse, swapHttp.statusCode == 200 else { return "" }
        let swapResult = try JSONSerialization.jsonObject(with: swapData) as? [String: Any] ?? [:]
        let tx = swapResult["tx"] as? [String: Any] ?? swapResult
        guard let txTo = tx["to"] as? String, let txData = tx["data"] as? String, !txTo.isEmpty, !txData.isEmpty else { return "" }

        // Step 2: broadcast the assembled tx via /api/v1/send (real
        // eth_sendRawTransaction). Returns the REAL tx hash, or "" on failure.
        var sendReq = URLRequest(url: URL(string: "\(baseURL)/send")!)
        sendReq.httpMethod = "POST"
        sendReq.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let sendBody: [String: Any] = [
            "walletId": walletId,
            "to": txTo,
            "data": txData,
            "value": "0",
            "type": "swap"
        ]
        sendReq.httpBody = try JSONSerialization.data(withJSONObject: sendBody)
        let (sendData, _) = try await URLSession.shared.data(for: sendReq)
        let sendResult = try JSONSerialization.jsonObject(with: sendData) as? [String: Any] ?? [:]
        return (sendResult["tx_hash"] as? String) ?? (sendResult["txHash"] as? String) ?? ""
    }
}

private struct AMMQuoteResponse: Codable {
    let tokenIn: String
    let tokenOut: String
    let amountIn: String
    let amountOut: String
    let amountOutWei: String
    let path: [String]?

    enum CodingKeys: String, CodingKey {
        case tokenIn = "token_in"
        case tokenOut = "token_out"
        case amountIn = "amount_in"
        case amountOut = "amount_out"
        case amountOutWei = "amount_out_wei"
        case path
    }
}

class StakingService {
    static let shared = StakingService()
    private let baseURL = "http://localhost:8443/api/v1/staking"
    
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
    private let baseURL = "http://localhost:8443/api/v1/bridge"
    
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
