import Foundation
import Combine
import CryptoKit

// MARK: - Wallet Models

struct Wallet: Identifiable, Codable {
    let id: String
    var name: String
    var address: String
    var publicKey: String
    var chainAddresses: [String: String] // chainId -> address
    var createdAt: Date
    var isBackedUp: Bool
    
    init(id: String = UUID().uuidString, name: String, address: String = "", publicKey: String = "", createdAt: Date = Date()) {
        self.id = id
        self.name = name
        self.address = address
        self.publicKey = publicKey
        self.chainAddresses = [:]
        self.createdAt = createdAt
        self.isBackedUp = false
    }
}

struct TokenBalance: Identifiable, Codable {
    let id: String
    let symbol: String
    let name: String
    let address: String?
    let decimals: Int
    var balance: Double
    var price: Double
    var chainId: Int64
    var logoURL: String?
    
    var usdValue: Double {
        return balance * price
    }
}

struct NFT: Identifiable, Codable {
    let id: String
    let tokenId: String
    let contractAddress: String
    let name: String
    let description: String?
    let imageURL: String?
    let chainId: Int64
    let collectionName: String?
}

struct Transaction: Identifiable, Codable {
    let id: String
    let hash: String
    let from: String
    let to: String
    let amount: Double
    let symbol: String
    let decimals: Int
    let chainId: Int64
    let status: TransactionStatus
    let timestamp: Date
    let type: TransactionType
    let gasUsed: Double?
    let gasPrice: Double?
    
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
        case approve
        case contractInteraction
    }
}

// MARK: - Wallet Store

class WalletStore: ObservableObject {
    static let shared = WalletStore()
    
    @Published var wallets: [Wallet] = []
    @Published var currentWallet: Wallet?
    @Published var tokens: [TokenBalance] = []
    @Published var nfts: [NFT] = []
    @Published var transactions: [Transaction] = []
    @Published var isLoading: Bool = false
    @Published var error: String?
    
    private let userDefaultsKey = "tigerwallet_wallets"
    
    private init() {
        loadWallets()
    }
    
    // MARK: - Wallet Operations
    
    func createWallet(name: String, seedPhrase: [String]? = nil) -> Wallet? {
        // Generate or use provided seed phrase
        let mnemonic = seedPhrase ?? generateMnemonic()
        
        // Derive wallet address from seed
        guard let walletData = deriveWalletFromSeed(mnemonic) else {
            error = "Failed to create wallet"
            return nil
        }
        
        let wallet = Wallet(
            name: name,
            address: walletData.address,
            publicKey: walletData.publicKey
        )
        
        // Generate addresses for all supported chains
        var updatedWallet = wallet
        for network in NetworkManager.shared.networks {
            if let address = deriveAddress(from: mnemonic, chainId: network.chainId, isEVM: network.isEVM) {
                updatedWallet.chainAddresses[String(network.chainId)] = address
            }
        }
        
        wallets.append(updatedWallet)
        currentWallet = updatedWallet
        saveWallets()
        
        // Fetch balances
        Task {
            await fetchBalances()
        }
        
        return updatedWallet
    }
    
    func importWallet(seedPhrase: [String], name: String) -> Wallet? {
        guard validateMnemonic(seedPhrase) else {
            error = "Invalid seed phrase"
            return nil
        }
        return createWallet(name: name, seedPhrase: seedPhrase)
    }
    
    func deleteWallet(_ wallet: Wallet) {
        wallets.removeAll { $0.id == wallet.id }
        if currentWallet?.id == wallet.id {
            currentWallet = wallets.first
        }
        saveWallets()
    }
    
    func selectWallet(_ wallet: Wallet) {
        currentWallet = wallet
        Task {
            await fetchBalances()
        }
    }
    
    // MARK: - Balance Operations
    
    func fetchBalances() async {
        guard let wallet = currentWallet else { return }
        
        isLoading = true
        
        // Fetch balances for all chains
        var allTokens: [TokenBalance] = []
        
        for (chainId, address) in wallet.chainAddresses {
            guard let chainIdInt = Int64(chainId) else { continue }
            
            // Fetch native token balance
            if let balance = await fetchNativeBalance(address: address, chainId: chainIdInt) {
                let token = TokenBalance(
                    id: "\(chainIdInt)_native",
                    symbol: getChainSymbol(chainId: chainIdInt),
                    name: getChainName(chainId: chainIdInt),
                    address: nil,
                    decimals: getChainDecimals(chainId: chainIdInt),
                    balance: balance,
                    price: getTokenPrice(symbol: getChainSymbol(chainId: chainIdInt)),
                    chainId: chainIdInt
                )
                allTokens.append(token)
            }
            
            // Fetch token balances
            if let tokenBalances = await fetchTokenBalances(address: address, chainId: chainIdInt) {
                allTokens.append(contentsOf: tokenBalances)
            }
        }
        
        await MainActor.run {
            self.tokens = allTokens
            self.isLoading = false
        }
    }
    
    // MARK: - Transaction Operations
    
    func sendTransaction(to: String, amount: Double, chainId: Int64, tokenAddress: String? = nil) async throws -> String {
        guard let wallet = currentWallet else {
            throw WalletError.noWallet
        }
        
        guard let fromAddress = wallet.chainAddresses[String(chainId)] else {
            throw WalletError.invalidAddress
        }
        
        isLoading = true
        
        // Build and sign transaction
        let signedTx = try await ServiceLocator.shared.walletService.buildAndSignTransaction(
            from: fromAddress,
            to: to,
            amount: amount,
            chainId: chainId,
            tokenAddress: tokenAddress
        )
        
        // Broadcast transaction
        let txHash = try await ServiceLocator.shared.blockchainService.broadcastTransaction(
            signedTx: signedTx,
            chainId: chainId
        )
        
        // Add to transactions
        let transaction = Transaction(
            id: UUID().uuidString,
            hash: txHash,
            from: fromAddress,
            to: to,
            amount: amount,
            symbol: getChainSymbol(chainId: chainId),
            decimals: getChainDecimals(chainId: chainId),
            chainId: chainId,
            status: .pending,
            timestamp: Date(),
            type: .send,
            gasUsed: nil,
            gasPrice: nil
        )
        
        await MainActor.run {
            self.transactions.insert(transaction, at: 0)
            self.isLoading = false
        }
        
        return txHash
    }
    
    // MARK: - Private Helpers
    
    private func generateMnemonic() -> [String] {
        // Use CryptoKit to generate secure random bytes
        var bytes = [UInt8](repeating: 0, count: 32)
        _ = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
        
        // For demo purposes, return a 12-word mnemonic
        // In production, use proper BIP39 wordlist
        return [
            "abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
            "absurd", "abuse", "access", "accident"
        ]
    }
    
    private func deriveWalletFromSeed(_ mnemonic: [String]) -> (address: String, publicKey: String)? {
        // Simplified - in production use proper key derivation (BIP39/BIP32)
        let seed = mnemonic.joined(separator: " ")
        let hash = SHA256.hash(data: Data(seed.utf8))
        let address = "0x" + hash.compactMap { String(format: "%02x", $0) }.joined().prefix(40)
        
        return (
            address: String(address),
            publicKey: hash.compactMap { String(format: "%02x", $0) }.joined()
        )
    }
    
    private func deriveAddress(from mnemonic: [String], chainId: Int64, isEVM: Bool) -> String? {
        guard isEVM else {
            // For non-EVM chains, generate different address format
            return deriveWalletFromSeed(mnemonic)?.address
        }
        return deriveWalletFromSeed(mnemonic)?.address
    }
    
    private func validateMnemonic(_ mnemonic: [String]) -> Bool {
        // Simplified validation - check word count
        return mnemonic.count == 12 || mnemonic.count == 24
    }
    
    private func fetchNativeBalance(address: String, chainId: Int64) async -> Double? {
        // In production, call actual RPC
        return nil
    }
    
    private func fetchTokenBalances(address: String, chainId: Int64) async -> [TokenBalance]? {
        // In production, call token balance API
        return nil
    }
    
    private func getChainSymbol(chainId: Int64) -> String {
        switch chainId {
        case 1: return "ETH"
        case 56: return "BNB"
        case 137: return "MATIC"
        case 42161: return "ETH"
        case 10: return "ETH"
        case 43114: return "AVAX"
        case 0: return "SOL"
        case 195: return "TRX"
        default: return "UNKNOWN"
        }
    }
    
    private func getChainName(chainId: Int64) -> String {
        switch chainId {
        case 1: return "Ethereum"
        case 56: return "BNB Smart Chain"
        case 137: return "Polygon"
        case 42161: return "Arbitrum"
        case 10: return "Optimism"
        case 43114: return "Avalanche"
        case 0: return "Solana"
        case 195: return "Tron"
        default: return "Unknown"
        }
    }
    
    private func getChainDecimals(chainId: Int64) -> Int {
        switch chainId {
        case 0: return 9 // Solana
        case 195: return 6 // Tron
        default: return 18 // EVM
        }
    }
    
    private func getTokenPrice(symbol: String) -> Double {
        // In production, fetch from price API
        switch symbol {
        case "ETH": return 3500.0
        case "BNB": return 600.0
        case "MATIC": return 0.8
        case "SOL": return 100.0
        case "TRX": return 0.1
        default: return 0.0
        }
    }
    
    // MARK: - Persistence
    
    private func saveWallets() {
        if let encoded = try? JSONEncoder().encode(wallets) {
            UserDefaults.standard.set(encoded, forKey: userDefaultsKey)
        }
    }
    
    private func loadWallets() {
        if let data = UserDefaults.standard.data(forKey: userDefaultsKey),
           let decoded = try? JSONDecoder().decode([Wallet].self, from: data) {
            wallets = decoded
            currentWallet = wallets.first
        }
    }
}

// MARK: - Errors

enum WalletError: LocalizedError {
    case noWallet
    case invalidAddress
    case insufficientBalance
    case transactionFailed
    
    var errorDescription: String? {
        switch self {
        case .noWallet: return "No wallet selected"
        case .invalidAddress: return "Invalid address"
        case .insufficientBalance: return "Insufficient balance"
        case .transactionFailed: return "Transaction failed"
        }
    }
}
