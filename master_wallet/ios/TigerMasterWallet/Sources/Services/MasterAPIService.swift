//
//  MasterAPIService.swift
//  TigerMasterWallet - Master Wallet API Service
//

import Foundation

/// Canonical REST client for the MasterWallet Go backend (port 8450).
/// All protected routes carry `Authorization: Bearer <JWT>`. The backend is the
/// sole signer / key holder; this client never fabricates balances, signatures,
/// or transaction hashes.
class MasterAPIService {
    static let defaultBaseURL = "http://localhost:8450"

    private let baseURL: String
    private let session: URLSession
    private let tokenStoreKey = "master_wallet_jwt"

    /// JWT bearer token used for every protected route. Set after login/register.
    var authToken: String? {
        get { UserDefaults.standard.string(forKey: tokenStoreKey) }
        set { UserDefaults.standard.set(newValue, forKey: tokenStoreKey) }
    }

    init(baseURL: String = MasterAPIService.defaultBaseURL, session: URLSession = .shared) {
        self.baseURL = baseURL
        self.session = session
    }

    // MARK: - Auth

    func register(email: String, password: String, name: String) async throws -> AuthResponse {
        let body = try JSONEncoder().encode(["email": email, "password": password, "name": name])
        let resp: AuthResponse = try await request(endpoint: "/api/v1/auth/register", method: "POST", body: body, auth: false)
        authToken = resp.token
        return resp
    }

    func login(email: String, password: String) async throws -> AuthResponse {
        let body = try JSONEncoder().encode(["email": email, "password": password])
        let resp: AuthResponse = try await request(endpoint: "/api/v1/auth/login", method: "POST", body: body, auth: false)
        authToken = resp.token
        return resp
    }

    // MARK: - Master Wallets

    func listMasterWallets() async throws -> MasterWalletsResponse {
        return try await request(endpoint: "/api/v1/master-wallet")
    }

    func createMasterWallet(name: String, password: String, chainId: Int) async throws -> MasterWallet {
        let body = try JSONEncoder().encode(["name": name, "password": password, "chain_id": chainId])
        return try await request(endpoint: "/api/v1/master-wallet", method: "POST", body: body)
    }

    func getMasterWallet(id: String) async throws -> MasterWallet {
        return try await request(endpoint: "/api/v1/master-wallet/\(id)")
    }

    func deleteMasterWallet(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(id)", method: "DELETE")
    }

    func getBalance(walletId: String, chainId: Int? = nil) async throws -> BalanceResponse {
        var endpoint = "/api/v1/master-wallet/\(walletId)/balance"
        if let chainId = chainId {
            endpoint += "?chain_id=\(chainId)"
        }
        return try await request(endpoint: endpoint)
    }

    /// Real sign + broadcast performed by the backend (secp256k1). Returns the
    /// on-chain transaction hash; never fabricated client-side.
    func sign(walletId: String, to: String, amount: String, password: String, token: String? = nil) async throws -> SignResponse {
        var payload: [String: Any] = ["to": to, "amount": amount, "password": password]
        if let token = token { payload["token"] = token }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/sign", method: "POST", body: body)
    }

    // MARK: - Sub Wallets

    func getSubWallets(masterWalletId: String) async throws -> [SubWallet] {
        return try await request(endpoint: "/api/v1/master-wallet/\(masterWalletId)/sub-wallets")
    }

    func createSubWallet(masterWalletId: String, name: String, password: String, chainId: Int) async throws -> SubWallet {
        let body = try JSONEncoder().encode(["name": name, "password": password, "chain_id": chainId])
        return try await request(endpoint: "/api/v1/master-wallet/\(masterWalletId)/sub-wallets", method: "POST", body: body)
    }

    func getSubWalletBalance(masterWalletId: String, subWalletId: String) async throws -> BalanceResponse {
        return try await request(endpoint: "/api/v1/master-wallet/\(masterWalletId)/sub-wallets/\(subWalletId)/balance")
    }

    func transferSubWallet(masterWalletId: String, subWalletId: String, to: String, amount: String, password: String, token: String? = nil) async throws -> SignResponse {
        var payload: [String: Any] = ["to": to, "amount": amount, "password": password]
        if let token = token { payload["token"] = token }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await request(endpoint: "/api/v1/master-wallet/\(masterWalletId)/sub-wallets/\(subWalletId)/transfer", method: "POST", body: body)
    }

    // MARK: - Transactions

    func listTransactions(walletId: String) async throws -> TransactionsResponse {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/transactions")
    }

    func createTransaction(walletId: String, to: String, amount: String, password: String, token: String? = nil) async throws -> SignResponse {
        var payload: [String: Any] = ["to": to, "amount": amount, "password": password]
        if let token = token { payload["token"] = token }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/transactions", method: "POST", body: body)
    }

    func approveTransaction(walletId: String, transactionId: String) async throws -> MasterTransaction {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/transactions/\(transactionId)/approve", method: "POST")
    }

    func rejectTransaction(walletId: String, transactionId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/transactions/\(transactionId)/reject", method: "POST")
    }

    // MARK: - Policies / Fees / Auto-Sign / Users

    func getPolicies(walletId: String) async throws -> [Policy] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/policies")
    }

    func createPolicy(walletId: String, policy: Policy) async throws -> Policy {
        let body = try JSONEncoder().encode(policy)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/policies", method: "POST", body: body)
    }

    func deletePolicy(walletId: String, policyId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/policies/\(policyId)", method: "DELETE")
    }

    func getFees(walletId: String) async throws -> [Fee] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/fees")
    }

    func getAutoSignRules(walletId: String) async throws -> [AutoSignRule] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/auto-sign")
    }

    func createAutoSignRule(walletId: String, rule: AutoSignRule) async throws -> AutoSignRule {
        let body = try JSONEncoder().encode(rule)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/auto-sign", method: "POST", body: body)
    }

    func deleteAutoSignRule(walletId: String, ruleId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/auto-sign/\(ruleId)", method: "DELETE")
    }

    func getUsers(walletId: String) async throws -> [MasterUser] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/users")
    }

    func createUser(walletId: String, user: CreateUserRequest) async throws -> MasterUser {
        let body = try JSONEncoder().encode(user)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/users", method: "POST", body: body)
    }

    // MARK: - Audit + Analytics

    func getAudit(walletId: String) async throws -> [AuditEntry] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/audit")
    }

    func getAnalyticsVolume(walletId: String) async throws -> [VolumeData] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/analytics/volume")
    }

    func getAnalyticsTransactions(walletId: String) async throws -> MasterAnalytics {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/analytics/transactions")
    }

    func getAnalyticsWallets(walletId: String) async throws -> [SubWallet] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/analytics/wallets")
    }

    // MARK: - Treasury

    func getTreasury(walletId: String) async throws -> TreasuryOverview {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/treasury")
    }

    // MARK: - Multisig

    func getMultisigWallets(walletId: String) async throws -> [SubWallet] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/multisig/wallets")
    }

    // MARK: - Public (no auth)

    func getChains() async throws -> ChainsResponse {
        return try await request(endpoint: "/api/v1/chains", auth: false)
    }

    func getGas(chainId: Int) async throws -> GasResponse {
        return try await request(endpoint: "/api/v1/gas?chain_id=\(chainId)", auth: false)
    }

    func getPrice(coinId: String = "ethereum") async throws -> PriceResponse {
        return try await request(endpoint: "/api/v1/price?coin_id=\(coinId)", auth: false)
    }

    func getHealth() async throws -> HealthResponse {
        return try await request(endpoint: "/health", auth: false)
    }

    // MARK: - Generic Request

    private func request<T: Decodable>(_ endpoint: String, method: String = "GET", body: Data? = nil, auth: Bool = true) async throws -> T {
        guard let url = URL(string: "\(baseURL)\(endpoint)") else {
            throw APIError(code: "INVALID_URL", message: "Invalid URL: \(endpoint)")
        }

        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if auth, let token = authToken, !token.isEmpty {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        } else if auth {
            throw APIError(code: "UNAUTHENTICATED", message: "No auth token; login required")
        }

        if let body = body {
            request.httpBody = body
        }

        let (data, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError(code: "INVALID_RESPONSE", message: "Invalid response")
        }

        guard (200...299).contains(httpResponse.statusCode) else {
            let bodyText = String(data: data, encoding: .utf8) ?? ""
            throw APIError(code: "HTTP_\(httpResponse.statusCode)", message: bodyText.isEmpty ? "HTTP \(httpResponse.statusCode)" : bodyText)
        }

        // Allow empty 2xx bodies (e.g. DELETE) to decode to EmptyResponse.
        if data.isEmpty, T.self is EmptyResponse.Type {
            return EmptyResponse() as! T
        }

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601

        return try decoder.decode(T.self, from: data)
    }
}

struct APIError: Error, LocalizedError {
    let code: String
    let message: String
    var errorDescription: String? { "\(code): \(message)" }
}

struct EmptyResponse: Codable {}

// MARK: - Auth Models

struct AuthResponse: Codable {
    let token: String
    let userId: String
    let email: String
    let role: String

    enum CodingKeys: String, CodingKey {
        case token
        case userId = "user_id"
        case email
        case role
    }
}

// MARK: - Wallet Models

struct MasterWallet: Codable, Identifiable {
    let id: String
    let address: String
    let publicKey: String?
    let name: String
    let createdAt: Date
    var totalValueUSD: Double

    enum CodingKeys: String, CodingKey {
        case id, address
        case publicKey = "public_key"
        case name
        case createdAt = "created_at"
        case totalValueUSD = "total_value_usd"
    }
}

struct MasterWalletsResponse: Codable {
    let wallets: [MasterWallet]
}

struct SubWallet: Codable, Identifiable {
    let id: String
    let name: String
    let address: String
    var balanceUSD: Double
    var status: String
    var permissions: [String]

    enum CodingKeys: String, CodingKey {
        case id, name, address
        case balanceUSD = "balance_usd"
        case status, permissions
    }
}

struct BalanceResponse: Codable {
    let address: String
    let chainId: Int
    let native: NativeBalance
    let tokens: [TokenBalance]

    struct NativeBalance: Codable {
        let symbol: String
        let balance: String
        let decimals: Int
        let usdValue: Double

        enum CodingKeys: String, CodingKey {
            case symbol, balance, decimals
            case usdValue = "usd_value"
        }
    }

    struct TokenBalance: Codable, Identifiable {
        let contract: String
        let symbol: String
        let balance: String
        let decimals: Int
        let usdValue: Double
        var id: String { contract }
    }

    enum CodingKeys: String, CodingKey {
        case address
        case chainId = "chain_id"
        case native, tokens
    }
}

struct SignResponse: Codable {
    let transactionHash: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case transactionHash = "transaction_hash"
        case status
    }
}

struct TransactionsResponse: Codable {
    let transactions: [MasterTransaction]
}

struct MasterTransaction: Codable, Identifiable {
    let id: String
    let subWalletId: String?
    let from: String
    let to: String
    let amount: String
    let chain: String
    let status: String
    let type: String
    let createdAt: Date
    var approvedAt: Date?

    enum CodingKeys: String, CodingKey {
        case id
        case subWalletId = "sub_wallet_id"
        case from, to, amount, chain, status, type
        case createdAt = "created_at"
        case approvedAt = "approved_at"
    }
}

struct Policy: Codable {
    let id: String?
    let ruleType: String
    let threshold: Double

    enum CodingKeys: String, CodingKey {
        case id
        case ruleType = "rule_type"
        case threshold
    }
}

struct Fee: Codable {
    let id: String
    let name: String
    let bps: Int
}

struct AuditEntry: Codable {
    let id: String
    let actor: String
    let action: String
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id, actor, action
        case createdAt = "created_at"
    }
}

struct TreasuryOverview: Codable {
    let totalValueUSD: Double
    let chains: [ChainValue]

    struct ChainValue: Codable {
        let chainId: Int
        let symbol: String
        let balance: String
        let usdValue: Double
    }

    enum CodingKeys: String, CodingKey {
        case totalValueUSD = "total_value_usd"
        case chains
    }
}

// MARK: - Auto-Sign / Users / Analytics Models

struct AutoSignRule: Codable, Identifiable {
    let id: String
    let name: String
    let maxAmount: Double
    let chain: String
    let enabled: Bool
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id, name
        case maxAmount = "max_amount"
        case chain, enabled
        case createdAt = "created_at"
    }
}

struct MasterUser: Codable, Identifiable {
    let id: String
    let email: String
    let name: String
    let permissions: MasterPermissions
    let createdAt: Date
    var lastLoginAt: Date?

    enum CodingKeys: String, CodingKey {
        case id, email, name, permissions
        case createdAt = "created_at"
        case lastLoginAt = "last_login_at"
    }
}

struct CreateUserRequest: Codable {
    let email: String
    let name: String
    let permissions: MasterPermissions
}

struct MasterPermissions: Codable {
    var canAutoSign: Bool
    var canAirdrop: Bool
    var canClaim: Bool
    var canAdjustFees: Bool
    var maxTransactionLimit: Double
}

struct MasterAnalytics: Codable {
    let totalWallets: Int
    let totalVolumeUSD: Double
    let totalTransactions: Int
    let pendingTransactions: Int
}

struct VolumeData: Codable {
    let date: Date
    let volumeUSD: Double
}

// MARK: - Public Models

struct ChainsResponse: Codable {
    let chains: [ChainInfo]
}

struct ChainInfo: Codable, Identifiable {
    let id: Int
    let name: String
    let symbol: String
    var idValue: Int { id }
}

struct GasResponse: Codable {
    let gasPrice: String
    let maxFee: String
    let priorityFee: String

    enum CodingKeys: String, CodingKey {
        case gasPrice = "gas_price"
        case maxFee = "max_fee"
        case priorityFee = "priority_fee"
    }
}

struct PriceResponse: Codable {
    let usd: Double
    let usd24hChange: Double

    enum CodingKeys: String, CodingKey {
        case usd
        case usd24hChange = "usd_24h_change"
    }
}

struct HealthResponse: Codable {
    let status: String
}

// MARK: - Auto Sign Service
class AutoSignService {
    private let apiService: MasterAPIService

    init(apiService: MasterAPIService) {
        self.apiService = apiService
    }

    func getRules(walletId: String) async throws -> [AutoSignRule] {
        return try await apiService.getAutoSignRules(walletId: walletId)
    }

    func createRule(walletId: String, rule: AutoSignRule) async throws -> AutoSignRule {
        return try await apiService.createAutoSignRule(walletId: walletId, rule: rule)
    }

    func approveTransaction(walletId: String, id: String) async throws -> MasterTransaction {
        return try await apiService.approveTransaction(walletId: walletId, transactionId: id)
    }

    func rejectTransaction(walletId: String, id: String) async throws {
        try await apiService.rejectTransaction(walletId: walletId, transactionId: id)
    }
}
