//
//  APIService.swift
//  TigerWallet - API Service
//

import Foundation

class APIService {
    private let baseURL: String
    private let session: URLSession
    
    init(baseURL: String = "https://api.tigerwallet.io") {
        self.baseURL = baseURL
        
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 60
        self.session = URLSession(configuration: config)
    }
    
    // MARK: - Generic Request
    func request<T: Decodable>(endpoint: String, method: String = "GET", body: Data? = nil, headers: [String: String]? = nil) async throws -> T {
        guard let url = URL(string: "\(baseURL)\(endpoint)") else {
            throw APIError(code: "INVALID_URL", message: "Invalid URL")
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        if let headers = headers {
            for (key, value) in headers {
                request.setValue(value, forHTTPHeaderField: key)
            }
        }
        
        if let body = body {
            request.httpBody = body
        }
        
        let (data, response) = try await session.data(for: request)
        
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
    
    // MARK: - Wallet APIs
    func getWallets() async throws -> [Wallet] {
        let response: APIResponse<[Wallet]> = try await request(endpoint: "/api/v1/wallets")
        return response.data ?? []
    }
    
    func getWallet(chain: Chain) async throws -> Wallet {
        let response: APIResponse<Wallet> = try await request(endpoint: "/api/v1/wallet/\(chain.rawValue)")
        guard let wallet = response.data else {
            throw APIError(code: "NO_DATA", message: "No wallet data")
        }
        return wallet
    }
    
    func createWallet(chain: Chain, type: String = "user") async throws -> Wallet {
        let body = try JSONEncoder().encode(["chain": chain.rawValue, "type": type])
        let response: APIResponse<Wallet> = try await request(endpoint: "/api/v1/wallets", method: "POST", body: body)
        guard let wallet = response.data else {
            throw APIError(code: "CREATE_FAILED", message: "Failed to create wallet")
        }
        return wallet
    }
    
    // MARK: - Balance APIs
    func getBalance(walletAddress: String, chain: Chain) async throws -> [Token] {
        let response: APIResponse<[Token]> = try await request(endpoint: "/api/v1/balance/\(chain.rawValue)/\(walletAddress)")
        return response.data ?? []
    }
    
    func getTokenPrice(tokenAddress: String, chain: Chain) async throws -> PriceData {
        let response: APIResponse<PriceData> = try await request(endpoint: "/api/v1/price/\(chain.rawValue)/\(tokenAddress)")
        guard let price = response.data else {
            throw APIError(code: "NO_DATA", message: "No price data")
        }
        return price
    }
    
    // MARK: - Transaction APIs
    func sendTransaction(to: String, amount: String, chain: Chain, data: String? = nil) async throws -> Transaction {
        let body = try JSONEncoder().encode([
            "to": to,
            "amount": amount,
            "chain": chain.rawValue,
            "data": data ?? "0x"
        ] as [String : Any])
        let response: APIResponse<Transaction> = try await request(endpoint: "/api/v1/transactions", method: "POST", body: body)
        guard let tx = response.data else {
            throw APIError(code: "SEND_FAILED", message: "Failed to send transaction")
        }
        return tx
    }
    
    func getTransactions(walletAddress: String, chain: Chain, limit: Int = 50) async throws -> [Transaction] {
        let response: APIResponse<[Transaction]> = try await request(endpoint: "/api/v1/transactions/\(chain.rawValue)/\(walletAddress)?limit=\(limit)")
        return response.data ?? []
    }
    
    func getTransactionStatus(hash: String, chain: Chain) async throws -> Transaction {
        let response: APIResponse<Transaction> = try await request(endpoint: "/api/v1/transaction/\(chain.rawValue)/\(hash)")
        guard let tx = response.data else {
            throw APIError(code: "NO_DATA", message: "No transaction data")
        }
        return tx
    }
    
    // MARK: - Swap APIs
    func getSwapQuote(fromToken: String, toToken: String, amount: String, chain: Chain) async throws -> SwapQuote {
        let response: APIResponse<SwapQuote> = try await request(
            endpoint: "/api/v1/swap/quote?from=\(fromToken)&to=\(toToken)&amount=\(amount)&chain=\(chain.rawValue)"
        )
        guard let quote = response.data else {
            throw APIError(code: "QUOTE_FAILED", message: "Failed to get swap quote")
        }
        return quote
    }
    
    func executeSwap(quoteId: String) async throws -> Transaction {
        let body = try JSONEncoder().encode(["quoteId": quoteId])
        let response: APIResponse<Transaction> = try await request(endpoint: "/api/v1/swap/execute", method: "POST", body: body)
        guard let tx = response.data else {
            throw APIError(code: "SWAP_FAILED", message: "Failed to execute swap")
        }
        return tx
    }
    
    // MARK: - Gas APIs
    func getGasPrice(chain: Chain) async throws -> GasData {
        let response: APIResponse<GasData> = try await request(endpoint: "/api/v1/gas/\(chain.rawValue)")
        guard let gas = response.data else {
            throw APIError(code: "NO_DATA", message: "No gas data")
        }
        return gas
    }
    
    // MARK: - Network APIs
    func getNetworkStatus(chain: Chain) async throws -> NetworkStatus {
        let response: APIResponse<NetworkStatus> = try await request(endpoint: "/api/v1/network/\(chain.rawValue)/status")
        guard let status = response.data else {
            throw APIError(code: "NO_DATA", message: "No network status")
        }
        return status
    }
    
    // MARK: - Token APIs
    func getTokens(chain: Chain, limit: Int = 100) async throws -> [Token] {
        let response: APIResponse<[Token]> = try await request(endpoint: "/api/v1/tokens/\(chain.rawValue)?limit=\(limit)")
        return response.data ?? []
    }
    
    // MARK: - Staking APIs
    func getStakingPositions(walletAddress: String, chain: Chain) async throws -> [StakingPosition] {
        let response: APIResponse<[StakingPosition]> = try await request(endpoint: "/api/v1/staking/\(chain.rawValue)/\(walletAddress)")
        return response.data ?? []
    }
    
    // MARK: - NFT APIs
    func getNFTs(walletAddress: String, chain: Chain) async throws -> [NFT] {
        let response: APIResponse<[NFT]> = try await request(endpoint: "/api/v1/nfts/\(chain.rawValue)/\(walletAddress)")
        return response.data ?? []
    }
}
