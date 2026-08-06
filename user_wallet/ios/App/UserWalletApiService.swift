import Foundation

class UserWalletApiService {
    private let baseURL = "http://localhost:8105/api/v1"
    
    func getBalances() async throws -> [Balance] {
        // Implement API call
        return []
    }
    
    func getWallets() async throws -> [Wallet] {
        // Implement API call
        return []
    }
    
    func createWallet(name: String, network: String, tokens: [String]) async throws -> Wallet {
        // Implement API call
        return Wallet(name: name, walletType: network, address: "")
    }
    
    func getTransactions() async throws -> [Transaction] {
        // Implement API call
        return []
    }
}
