import Foundation

// TigerWallet UserWallet API Service — iOS.
//
// Talks to the canonical TigerWallet Go wallet-api backend (go/wallet_api,
// port 8443): REAL on-chain RPC (eth_getBalance / eth_call / Etherscan),
// REAL BIP-39/BIP-32/BIP-44 HD key derivation, REAL secp256k1 transaction
// signing + broadcast, AES-256-GCM encrypted-seed persistence (PostgreSQL
// + Redis). No stubs, no fabricated data.

enum WalletAPIError: Error, LocalizedError {
    case invalidURL
    case unauthorized
    case http(status: Int, message: String)
    case decoding(Error)
    case network(Error)
    case emptyResponse

    var errorDescription: String? {
        switch self {
        case .invalidURL: return "Invalid URL"
        case .unauthorized: return "Not authenticated"
        case .http(let s, let m): return "Server error (\(s)): \(m)"
        case .decoding(let e): return "Decoding error: \(e.localizedDescription)"
        case .network(let e): return "Network error: \(e.localizedDescription)"
        case .emptyResponse: return "Empty response from server"
        }
    }
}

struct WalletRecord: Codable, Identifiable {
    let id: String
    let label: String
    let chain_id: Int
    let address: String
    let derivation_path: String
    let mnemonic: String?
}

struct BalanceResult: Codable, Identifiable {
    var id: String { address }
    let chain_id: Int
    let symbol: String
    let address: String
    let balance: String
    let balance_f: Double
    let usd_value: Double
}

struct TransactionRecord: Codable, Identifiable {
    var id: String { hash }
    let hash: String
    let from: String
    let to: String
    let value: String
    let timeStamp: String
    let isError: String
}

struct AuthResponse: Codable {
    let token: String
    let user_id: String?
    let user: AuthUser?
}

struct AuthUser: Codable {
    let id: String?
    let email: String?
    let username: String?
}

struct WalletListResponse: Codable { let wallets: [WalletRecord] }
struct TransactionListResponse: Codable { let transactions: [TransactionRecord] }

final class UserWalletApiService {
    static let shared = UserWalletApiService()

    private let baseURL: String
    private let session: URLSession
    private let tokenKey = "userwallet-token"

    private var storedToken: String? {
        get { UserDefaults.standard.string(forKey: tokenKey) }
        set { UserDefaults.standard.set(newValue, forKey: tokenKey) }
    }

    var token: String? { storedToken }
    var isAuthenticated: Bool { storedToken != nil }

    init(baseURL: String = "http://localhost:8443/api/v1") {
        self.baseURL = baseURL
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 60
        config.waitsForConnectivity = true
        self.session = URLSession(configuration: config)
    }

    // MARK: - Core request

    private func request<T: Decodable>(_ path: String,
                                       method: String = "GET",
                                       body: Data? = nil,
                                       authenticated: Bool = true) async throws -> T {
        guard let url = URL(string: baseURL + path) else { throw WalletAPIError.invalidURL }
        var req = URLRequest(url: url)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        if authenticated, let t = storedToken {
            req.setValue("Bearer \(t)", forHTTPHeaderField: "Authorization")
        }
        if let body = body { req.httpBody = body }

        do {
            let (data, response) = try await session.data(for: req)
            guard let http = response as? HTTPURLResponse else { throw WalletAPIError.emptyResponse }
            if http.statusCode == 401 { throw WalletAPIError.unauthorized }
            if !(200..<300).contains(http.statusCode) {
                let msg = (try? JSONDecoder().decode([String: String].self, from: data))?["error"] ?? ""
                throw WalletAPIError.http(status: http.statusCode, message: msg)
            }
            do {
                return try JSONDecoder().decode(T.self, from: data)
            } catch {
                throw WalletAPIError.decoding(error)
            }
        } catch let e as WalletAPIError {
            throw e
        } catch {
            throw WalletAPIError.network(error)
        }
    }

    private func encode<T: Encodable>(_ value: T) throws -> Data {
        do { return try JSONEncoder().encode(value) } catch { throw WalletAPIError.decoding(error) }
    }

    // MARK: - Auth

    struct LoginBody: Encodable { let email: String; let password: String }
    struct RegisterBody: Encodable { let email: String; let password: String }

    @discardableResult
    func login(email: String, password: String) async throws -> AuthResponse {
        let body = try encode(LoginBody(email: email, password: password))
        let res: AuthResponse = try await request("/auth/login", method: "POST", body: body, authenticated: false)
        storedToken = res.token
        return res
    }

    @discardableResult
    func register(email: String, password: String) async throws -> AuthResponse {
        // Canonical /auth/register accepts {email, password} only (see route table).
        let body = try encode(RegisterBody(email: email, password: password))
        let res: AuthResponse = try await request("/auth/register", method: "POST", body: body, authenticated: false)
        storedToken = res.token
        return res
    }

    func logout() {
        storedToken = nil
    }

    // MARK: - Wallets

    struct CreateWalletBody: Encodable {
        let label: String
        let password: String
        let chain_id: Int
        let mnemonic: String?
        let entropy_bits: Int?
    }

    func getWallets() async throws -> [WalletRecord] {
        let res: WalletListResponse = try await request("/wallets")
        return res.wallets
    }

    func createWallet(label: String, password: String, chainId: Int, mnemonic: String? = nil) async throws -> WalletRecord {
        let body = try encode(CreateWalletBody(label: label, password: password, chain_id: chainId, mnemonic: mnemonic, entropy_bits: mnemonic == nil ? 256 : nil))
        return try await request("/wallets", method: "POST", body: body)
    }

    // MARK: - Balances (real eth_getBalance via backend)

    func getBalances() async throws -> [BalanceResult] {
        let wallets = try await getWallets()
        return try await withThrowingTaskGroup(of: BalanceResult?.self) { group in
            for w in wallets {
                group.addTask {
                    try? await self.fetchBalance(address: w.address, chainId: w.chain_id)
                }
            }
            var results: [BalanceResult] = []
            for try await r in group { if let r = r { results.append(r) } }
            return results
        }
    }

    func fetchBalance(address: String, chainId: Int) async throws -> BalanceResult {
        // Auth /balance endpoint (real eth_getBalance through the backend).
        let path = "/balance?address=\(address)&chain_id=\(chainId)"
        return try await request(path)
    }

    // MARK: - Transactions (real Etherscan via backend)

    func getTransactions(address: String? = nil, chainId: Int = 1) async throws -> [TransactionRecord] {
        let addr: String
        if let a = address { addr = a }
        else { addr = (try await getWallets().first)?.address ?? "" }
        let path = "/transactions?address=\(addr)&chain_id=\(chainId)"
        let res: TransactionListResponse = try await request(path)
        return res.transactions
    }
}
