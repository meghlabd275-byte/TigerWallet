//
//  TokensViewModel.swift
//  TigerWalletAdmin
//
//  Tokens ViewModel with real API integration
//

import Foundation
import Combine

class TokensViewModel: ObservableObject {
    @Published var tokens: [Token] = []
    @Published var pendingTokens: [Token] = []
    @Published var isLoading = false
    @Published var errorMessage: String?
    
    private let apiService = APIService.shared
    
    func loadTokens() {
        isLoading = true
        errorMessage = nil
        
        apiService.getTokens { [weak self] result in
            DispatchQueue.main.async {
                self?.isLoading = false
                
                switch result {
                case .success(let tokens):
                    self?.tokens = tokens
                case .failure(let error):
                    self?.errorMessage = error.localizedDescription
                    self?.loadSampleTokens()
                }
            }
        }
        
        // Load pending tokens
        apiService.getPendingTokens { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success(let pending):
                    self?.pendingTokens = pending
                case .failure:
                    break
                }
            }
        }
    }
    
    func filteredTokens(searchText: String) -> [Token] {
        if searchText.isEmpty {
            return tokens
        }
        
        return tokens.filter { token in
            token.name.localizedCaseInsensitiveContains(searchText) ||
            token.symbol.localizedCaseInsensitiveContains(searchText)
        }
    }
    
    func toggleTokenStatus(_ token: Token) {
        let newStatus = token.status == "active" ? "paused" : "active"
        
        apiService.updateTokenStatus(tokenId: token.id, status: newStatus) { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success:
                    if let index = self?.tokens.firstIndex(where: { $0.id == token.id }) {
                        let updated = Token(
                            id: token.id,
                            name: token.name,
                            symbol: token.symbol,
                            chain: token.chain,
                            price: token.price,
                            change24h: token.change24h,
                            marketCap: token.marketCap,
                            status: newStatus,
                            isVerified: token.isVerified,
                            decimals: token.decimals,
                            contractAddress: token.contractAddress
                        )
                        self?.tokens[index] = updated
                    }
                case .failure(let error):
                    self?.errorMessage = error.localizedDescription
                }
            }
        }
    }
    
    func approveToken(_ token: Token) {
        apiService.approveToken(tokenId: token.id) { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success:
                    self?.pendingTokens.removeAll { $0.id == token.id }
                    self?.tokens.append(token)
                case .failure(let error):
                    self?.errorMessage = error.localizedDescription
                }
            }
        }
    }
    
    func rejectToken(_ token: Token, reason: String) {
        apiService.rejectToken(tokenId: token.id, reason: reason) { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success:
                    self?.pendingTokens.removeAll { $0.id == token.id }
                case .failure(let error):
                    self?.errorMessage = error.localizedDescription
                }
            }
        }
    }
    
    private func loadSampleTokens() {
        tokens = [
            Token(id: "1", name: "Bitcoin", symbol: "BTC", chain: "Bitcoin", price: 67500.00, change24h: 2.5, marketCap: 1300000000000, status: "active", isVerified: true, decimals: 8, contractAddress: nil),
            Token(id: "2", name: "Ethereum", symbol: "ETH", chain: "Ethereum", price: 3450.00, change24h: -1.2, marketCap: 415000000000, status: "active", isVerified: true, decimals: 18, contractAddress: "0x..."),
            Token(id: "3", name: "Tether", symbol: "USDT", chain: "Ethereum", price: 1.00, change24h: 0.01, marketCap: 95000000000, status: "active", isVerified: true, decimals: 6, contractAddress: "0x..."),
            Token(id: "4", name: "BNB", symbol: "BNB", chain: "BNB Chain", price: 580.00, change24h: 1.8, marketCap: 87000000000, status: "active", isVerified: true, decimals: 18, contractAddress: "0x..."),
            Token(id: "5", name: "Solana", symbol: "SOL", chain: "Solana", price: 145.00, change24h: 5.2, marketCap: 65000000000, status: "active", isVerified: true, decimals: 9, contractAddress: nil)
        ]
    }
}
