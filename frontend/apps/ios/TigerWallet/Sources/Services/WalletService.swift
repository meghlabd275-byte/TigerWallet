//
//  WalletService.swift
//  TigerWallet - Wallet Service
//

import Foundation

class WalletService {
    private let apiService: APIService
    
    init(apiService: APIService) {
        self.apiService = apiService
    }
    
    // MARK: - Wallet Management
    func getWallet(for chain: Chain) async throws -> Wallet {
        // First try to get existing wallet
        do {
            return try await apiService.getWallet(chain: chain)
        } catch {
            // If no wallet exists, create one
            let wallet = try await apiService.createWallet(chain: chain)
            return wallet
        }
    }
    
    func getAllWallets() async throws -> [Wallet] {
        return try await apiService.getWallets()
    }
    
    // MARK: - Balance
    func getBalance(wallet: Wallet) async throws -> [Token] {
        return try await apiService.getBalance(walletAddress: wallet.address, chain: wallet.chain)
    }
    
    func getTotalBalance(wallet: Wallet) async throws -> Double {
        let tokens = try await getBalance(wallet: wallet)
        return tokens.reduce(0) { $0 + $1.balanceUSD }
    }
    
    // MARK: - Transactions
    func send(to: String, amount: String, chain: Chain) async throws -> Transaction {
        return try await apiService.sendTransaction(to: to, amount: amount, chain: chain)
    }
    
    func getTransactions(wallet: Wallet, limit: Int = 50) async throws -> [Transaction] {
        return try await apiService.getTransactions(walletAddress: wallet.address, chain: wallet.chain, limit: limit)
    }
    
    func getTransactionStatus(hash: String, chain: Chain) async throws -> Transaction {
        return try await apiService.getTransactionStatus(hash: hash, chain: chain)
    }
    
    // MARK: - Swap
    func getSwapQuote(fromToken: Token, toToken: Token, amount: String, chain: Chain) async throws -> SwapQuote {
        return try await apiService.getSwapQuote(fromToken: fromToken.address, toToken: toToken.address, amount: amount, chain: chain)
    }
    
    func executeSwap(quote: SwapQuote) async throws -> Transaction {
        return try await apiService.executeSwap(quoteId: quote.id)
    }
    
    // MARK: - Gas
    func getGasPrice(chain: Chain) async throws -> GasData {
        return try await apiService.getGasPrice(chain: chain)
    }
    
    // MARK: - Network
    func getNetworkStatus(chain: Chain) async throws -> NetworkStatus {
        return try await apiService.getNetworkStatus(chain: chain)
    }
}

// MARK: - Price Service
class PriceService {
    private let apiService: APIService
    
    init(apiService: APIService) {
        self.apiService = apiService
    }
    
    func getPrice(tokenAddress: String, chain: Chain) async throws -> PriceData {
        return try await apiService.getTokenPrice(tokenAddress: tokenAddress, chain: chain)
    }
    
    func getPrices(tokens: [Token], chain: Chain) async throws -> [String: PriceData] {
        var prices: [String: PriceData] = [:]
        
        for token in tokens {
            if let price = try? await getPrice(tokenAddress: token.address, chain: chain) {
                prices[token.address] = price
            }
        }
        
        return prices
    }
}
