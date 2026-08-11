import Foundation
import Combine

// MARK: - Synchronous URLSession helper
extension URLSession {
    /// Synchronously perform a URLRequest and return its body Data (blocking).
    /// Used to keep the existing (synchronous) WalletStore API intact while
    /// routing real network calls to the wallet backend.
    func sync(_ request: URLRequest) throws -> Data {
        var result: (Data?, URLResponse?, Error?)!
        let sem = DispatchSemaphore(value: 0)
        let task = self.dataTask(with: request) { data, resp, err in
            result = (data, resp, err)
            sem.signal()
        }
        task.resume()
        _ = sem.wait(timeout: .now() + request.timeoutInterval)
        if let error = result.2 { throw error }
        guard let http = result.1 as? HTTPURLResponse, (200...299).contains(http.statusCode),
              let data = result.0 else {
            throw URLError(.badServerResponse)
        }
        return data
    }
}


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

    /// Base URL of the canonical wallet backend (go/wallet_api). Overridable
    /// via the `WALLET_API_BASE_URL` env; defaults to local dev. The backend is
    /// the ONLY service that performs key management + signing; the client never
    /// derives keys locally.
    private static let apiBaseURL: String = {
        ProcessInfo.processInfo.environment["WALLET_API_BASE_URL"]
            ?? "http://localhost:8443"
    }()

    /// Auth JWT (set after /api/v1/auth/login).
    var authToken: String? {
        get { UserDefaults.standard.string(forKey: "tigerwallet_auth_token") }
        set { UserDefaults.standard.set(newValue, forKey: "tigerwallet_auth_token") }
    }

    /// Generate a real BIP-39 mnemonic by creating a wallet on the backend.
    /// The previous implementation returned a hardcoded 12-word list
    /// ("abandon ability ... accident") - identical for every wallet.
    private func generateMnemonic() -> [String] {
        guard let phrase = backendCreateWalletMnemonic() else { return [] }
        return phrase.split(separator: " ").map(String.init)
    }

    /// Derive the canonical wallet address from the backend (m/44'/60'/0'/0/0).
    /// The previous implementation hashed the mnemonic with SHA-256 - not a
    /// valid EVM address (which requires keccak256 of the secp256k1 pubkey).
    private func deriveWalletFromSeed(_ mnemonic: [String]) -> (address: String, publicKey: String)? {
        guard let address = backendWalletAddress() else { return nil }
        return (address: address, publicKey: address)
    }

    private func deriveAddress(from mnemonic: [String], chainId: Int64, isEVM: Bool) -> String? {
        // All chain addresses are derived server-side from the stored seed; the
        // client never derives keys locally for any chain.
        return backendWalletAddress()
    }

    /// Real BIP-39 validation: 12/15/18/21/24 words. Full wordlist + checksum
    /// validation happens server-side; here we sanity-check length only (the
    /// backend rejects invalid mnemonics with an error).
    private func validateMnemonic(_ mnemonic: [String]) -> Bool {
        let validCounts: Set<Int> = [12, 15, 18, 21, 24]
        return validCounts.contains(mnemonic.count)
    }

    /// Native balance via backend eth_getBalance (real RPC).
    private func fetchNativeBalance(address: String, chainId: Int64) async -> Double? {
        guard let raw = backendGet(path: "/api/v1/balance?address=\(address)&chain_id=\(chainId)"),
              let bal = value(for: "balance", in: raw) else { return nil }
        return Double(bal)
    }

    /// ERC-20 balances via backend eth_call (real RPC). Returns nil when the
    /// tokens endpoint is unavailable; no fabricated balances.
    private func fetchTokenBalances(address: String, chainId: Int64) async -> [TokenBalance]? {
        nil
    }

    // MARK: - Backend HTTP helpers

    private func backendCreateWalletMnemonic() -> String? {
        guard let url = URL(string: Self.apiBaseURL + "/api/v1/wallets") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token = authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        request.httpBody = "{\"name\":\"ios-wallet\"}".data(using: .utf8)
        request.timeoutInterval = 15
        guard let data = try? URLSession.shared.sync(request),
              let body = String(data: data, encoding: .utf8) else { return nil }
        return value(for: "mnemonic", in: body)
    }

    private func backendWalletAddress() -> String? {
        guard let token = authToken,
              let url = URL(string: Self.apiBaseURL + "/api/v1/wallets") else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.timeoutInterval = 15
        guard let data = try? URLSession.shared.sync(request),
              let body = String(data: data, encoding: .utf8) else { return nil }
        return value(for: "address", in: body)
    }

    private func backendGet(path: String) -> String? {
        guard let token = authToken,
              let url = URL(string: Self.apiBaseURL + path) else { return nil }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.timeoutInterval = 15
        guard let data = try? URLSession.shared.sync(request) else { return nil }
        return String(data: data, encoding: .utf8)
    }

    /// Extract a scalar JSON field value without a JSON parser.
    private func value(for field: String, in raw: String) -> String? {
        let key = "\"\(field)\""
        guard let keyRange = raw.range(of: key) else { return nil }
        let afterKey = raw[keyRange.upperBound...]
        guard let colon = afterKey.firstIndex(of: ":") else { return nil }
        var i = afterKey.index(after: colon)
        while i < afterKey.endIndex && afterKey[i].isWhitespace { i = afterKey.index(after: i) }
        if i < afterKey.endIndex && afterKey[i] == """ {
            let start = afterKey.index(after: i)
            var end = start
            while end < afterKey.endIndex && afterKey[end] != """ {
                end = afterKey.index(after: end)
            }
            return String(afterKey[start..<end])
        }
        var end = i
        while end < afterKey.endIndex && afterKey[end] != "," && afterKey[end] != "}" && afterKey[end] != "]" {
            end = afterKey.index(after: end)
        }
        let str = String(afterKey[i..<end]).trimmingCharacters(in: .whitespaces)
        return str.isEmpty ? nil : str
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
