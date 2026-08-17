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

    // POST /auth/guest { device_id } -> { user_id, token, guest: true }. Public
    // (no auth required). Provisions an anonymous guest account so the user can
    // Create/Import a wallet without registering. The token is persisted exactly
    // like login (storedToken -> UserDefaults "userwallet-token").
    struct GuestAuthBody: Encodable { let device_id: String }
    struct GuestAuthResponse: Codable {
        let token: String
        let user_id: String?
        let guest: Bool?
    }

    @discardableResult
    func guestAuth(deviceId: String) async throws -> GuestAuthResponse {
        let body = try encode(GuestAuthBody(device_id: deviceId))
        let res: GuestAuthResponse = try await request("/auth/guest", method: "POST", body: body, authenticated: false)
        if !res.token.isEmpty { storedToken = res.token }
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

    // MARK: - Send / Sign (real on-chain secp256k1 broadcast via backend)

    struct SendBody: Encodable {
        let wallet_id: String
        let password: String
        let to: String
        let value: String
        let chain_id: Int
    }
    struct SendResult: Codable { let tx_hash: String; let raw_tx: String?; let nonce: Int? }

    @discardableResult
    func sendTransaction(walletId: String, password: String, to: String, value: String, chainId: Int = 1) async throws -> SendResult {
        let body = try encode(SendBody(wallet_id: walletId, password: password, to: to, value: value, chain_id: chainId))
        return try await request("/send", method: "POST", body: body)
    }

    // POST /auto-send with the SAME body as /send, plus optional
    // ?master_wallet_id=<id> query. Same Bearer JWT auth as /send. Returns the
    // existing send response PLUS { auto_approved, auto_approval_reason }.
    struct AutoSendResult: Codable {
        let tx_hash: String
        let raw_tx: String?
        let nonce: Int?
        let auto_approved: Bool?
        let auto_approval_reason: String?
    }

    @discardableResult
    func autoSendTransaction(walletId: String, password: String, to: String, value: String, chainId: Int = 1, masterWalletId: String? = nil) async throws -> AutoSendResult {
        let body = try encode(SendBody(wallet_id: walletId, password: password, to: to, value: value, chain_id: chainId))
        var path = "/auto-send"
        if let mw = masterWalletId {
            path += "?master_wallet_id=\(mw)"
        }
        return try await request(path, method: "POST", body: body)
    }

    // GET /transactions/:txHash?chain_id=N -> { status, block_number?, confirmations? }.
    // Transaction-status proxy (explorer receipt lookup).
    struct TransactionStatus: Codable {
        let status: String
        let block_number: Int?
        let confirmations: Int?
    }

    func getTransactionStatus(txHash: String, chainId: Int = 1) async throws -> TransactionStatus {
        let path = "/transactions/\(txHash)?chain_id=\(chainId)"
        return try await request(path)
    }

    struct SignBody: Encodable { let wallet_id: String; let password: String; let message: String }
    struct SignResult: Codable { let signature: String }

    func signMessage(walletId: String, password: String, message: String) async throws -> String {
        let body = try encode(SignBody(wallet_id: walletId, password: password, message: message))
        let res: SignResult = try await request("/sign", method: "POST", body: body)
        return res.signature
    }

    // MARK: - Tokens (real ERC-20 balanceOf via backend)

    struct TokenBalance: Codable, Identifiable {
        var id: String { contract_address }
        let contract_address: String
        let symbol: String
        let name: String
        let decimals: Int
        let balance: String
        let balance_f: Double
        let usd_value: Double
    }
    struct TokenListResponse: Codable { let tokens: [TokenBalance] }

    func getTokenBalances(address: String, chainId: Int) async throws -> [TokenBalance] {
        let path = "/tokens?address=\(address)&chain_id=\(chainId)"
        let res: TokenListResponse = try await request(path)
        return res.tokens
    }

    // MARK: - NFTs (real on-chain ERC-721 inventory via backend)

    struct NFT: Codable, Identifiable {
        var id: String { contract_address + ":" + token_id }
        let contract_address: String
        let token_id: String
        let name: String
        let symbol: String
        let token_uri: String
        let image_uri: String
    }
    struct NFTListResponse: Codable { let nfts: [NFT] }

    func getNFTs(address: String, chainId: Int) async throws -> [NFT] {
        let path = "/nfts?address=\(address)&chain_id=\(chainId)"
        let res: NFTListResponse = try await request(path)
        return res.nfts
    }

    // MARK: - Gas / Price / Chains (real RPC + CoinGecko via backend)

    struct GasPrice: Codable {
        let gas_price: String
        let base_fee: String
        let priority_fee: String
        let estimated_cost: String
    }

    func getGasPrice(chainId: Int) async throws -> GasPrice {
        let path = "/gas?chain_id=\(chainId)"
        return try await request(path)
    }

    struct TokenPrice: Codable { let usd: Double; let usd_24h_change: Double }

    func getTokenPrice(token: String) async throws -> TokenPrice {
        let path = "/price?token=\(token)"
        return try await request(path)
    }

    struct ChainInfo: Codable, Identifiable {
        var id: Int { chain_id }
        let chain_id: Int
        let name: String
        let symbol: String
        let rpc_endpoint: String?
        let derivation_path: String?
        let explorer_api: String?
        let explorer_url: String?
        let chain_type: String?
        let decimals: Int?
        let coin_type: Int?
        let is_testnet: Bool?
    }
    struct ChainListResponse: Codable { let chains: [ChainInfo] }

    func getChains() async throws -> [ChainInfo] {
        let res: ChainListResponse = try await request("/chains")
        return res.chains
    }

    struct NetworkStatus: Codable {
        let chain_id: Int
        let block_number: Int
        let connected: Bool
    }

    // Mirrors the web client: derive connected/chain-id from /chains (no
    // dedicated status route on wallet_api). block_number is not exposed by the
    // chains list endpoint, so we report 0 honestly rather than fabricate.
    func getNetworkStatus(chainId: Int) async throws -> NetworkStatus {
        let chains = try await getChains()
        let chain = chains.first { $0.chain_id == chainId }
        return NetworkStatus(chain_id: chain?.chain_id ?? chainId, block_number: 0, connected: chain != nil)
    }

    // MARK: - Swap (real CoinGecko cross-rate + on-chain via backend)

    struct SwapQuote: Codable {
        let from_token: String
        let to_token: String
        let from_amount: String
        let to_amount: String
        let price_impact: Double
        let route: String
    }

    func getSwapQuote(fromToken: String, toToken: String, fromAmount: String, chainId: Int = 1) async throws -> SwapQuote {
        let path = "/swap/quote?from_token=\(fromToken)&to_token=\(toToken)&from_amount=\(fromAmount)&chain_id=\(chainId)"
        return try await request(path)
    }

    // MARK: - Staking (real on-chain action via backend /send)

    // The backend returns the full supported-asset list and ignores ?asset=.
    // Response shape: { success, assets[], apy, min_stake, lock_period }.
    struct StakingAsset: Codable {
        let symbol: String
        let chain_id: Int
        let apy: Double
        let min_stake: Double
        let lock_period: Int
        let verified: Bool
    }

    struct StakingQuote: Codable {
        let success: Bool
        let assets: [StakingAsset]
        let apy: Double
        let min_stake: Double
        let lock_period: Int
    }

    func getStakingQuote(_ asset: String? = nil) async throws -> StakingQuote {
        // asset is accepted for client-parity but ignored by the backend,
        // which returns every supported staking asset.
        return try await request("/staking/quote")
    }

    // MARK: - Auxiliary DeFi (fiat ramp, crypto card, P2P, convert)
    // All delegate to the canonical backend proxy routes (real CoinGecko
    // prices, real provider checkout URLs, real PostgreSQL-backed listings).

    // requestRaw returns an untyped [String: Any] JSON object for endpoints
    // whose response shapes are service-specific (fiat/card/p2p).
    private func requestRaw(_ path: String, method: String = "GET", body: Data? = nil, authenticated: Bool = true) async throws -> [String: Any] {
        guard let url = URL(string: baseURL + path) else { throw WalletAPIError.invalidURL }
        var req = URLRequest(url: url)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        if authenticated, let t = storedToken {
            req.setValue("Bearer \(t)", forHTTPHeaderField: "Authorization")
        }
        if let body = body { req.httpBody = body }
        let (data, response) = try await session.data(for: req)
        guard let http = response as? HTTPURLResponse else { throw WalletAPIError.emptyResponse }
        if http.statusCode == 401 { throw WalletAPIError.unauthorized }
        if !(200..<300).contains(http.statusCode) {
            let msg = (try? JSONDecoder().decode([String: String].self, from: data))?["error"] ?? ""
            throw WalletAPIError.http(status: http.statusCode, message: msg)
        }
        return (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]
    }

    func getFiatProviders() async throws -> [String: Any] {
        return try await requestRaw("/ramp/providers")
    }

    func getFiatQuote(providerId: String, amount: String, fiat: String, crypto: String, method: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "providerId": providerId, "amount": amount,
            "fiatCurrency": fiat, "cryptoCurrency": crypto, "paymentMethod": method,
        ])
        return try await requestRaw("/ramp/quote", method: "POST", body: body)
    }

    func getFiatOfframpQuote(providerId: String, amount: String, fiat: String, crypto: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "providerId": providerId, "amount": amount,
            "fiatCurrency": fiat, "cryptoCurrency": crypto,
        ])
        return try await requestRaw("/ramp/offramp-quote", method: "POST", body: body)
    }

    func getCryptoCardBalance() async throws -> [String: Any] {
        return try await requestRaw("/card/balance")
    }

    func getP2PAdverts() async throws -> [String: Any] {
        return try await requestRaw("/p2p/adverts")
    }

    // Convert is the same path as swap (cross-token conversion).
    func getConvertQuote(fromToken: String, toToken: String, fromAmount: String, chainId: Int = 1) async throws -> SwapQuote {
        return try await getSwapQuote(fromToken: fromToken, toToken: toToken, fromAmount: fromAmount, chainId: chainId)
    }

    // MARK: - Profile / Health

    // No /profile route on the backend — decode the JWT payload locally
    // (no network call), exactly like the web client.
    func getProfile() async throws -> [String: Any] {
        guard let t = storedToken else { throw WalletAPIError.unauthorized }
        let parts = t.split(separator: ".")
        guard parts.count >= 2 else { throw WalletAPIError.unauthorized }
        var b64 = parts[1].replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
        while b64.count % 4 != 0 { b64 += "=" }
        guard let payloadData = Data(base64Encoded: b64) else { throw WalletAPIError.unauthorized }
        do {
            guard let payload = try JSONSerialization.jsonObject(with: payloadData) as? [String: Any] else {
                throw WalletAPIError.emptyResponse
            }
            let id = payload["sub"] ?? payload["user_id"] ?? ""
            let email = payload["email"] ?? ""
            let username = payload["email"] ?? payload["username"] ?? ""
            return ["id": id, "email": email, "username": username]
        } catch let e as WalletAPIError {
            throw e
        } catch {
            throw WalletAPIError.decoding(error)
        }
    }

    // /health lives at the server root (outside /api/v1): strip the /api/v1
    // suffix from baseURL and GET /health. No auth header required.
    func health() async throws -> [String: Any] {
        let root: String
        if baseURL.hasSuffix("/api/v1/") {
            root = String(baseURL.dropLast("/api/v1/".count))
        } else if baseURL.hasSuffix("/api/v1") {
            root = String(baseURL.dropLast("/api/v1".count))
        } else {
            root = baseURL
        }
        guard let url = URL(string: root + "/health") else { throw WalletAPIError.invalidURL }
        var req = URLRequest(url: url)
        req.httpMethod = "GET"
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        let (data, response) = try await session.data(for: req)
        guard let http = response as? HTTPURLResponse else { throw WalletAPIError.emptyResponse }
        if http.statusCode == 401 { throw WalletAPIError.unauthorized }
        if !(200..<300).contains(http.statusCode) {
            let msg = (try? JSONDecoder().decode([String: String].self, from: data))?["error"] ?? ""
            throw WalletAPIError.http(status: http.statusCode, message: msg)
        }
        return (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]
    }

    // MARK: - Wallet import / NFT transfer / Tx receipt / Gas estimate

    // POST /wallets with a mnemonic (optionally a BIP-39 passphrase) imports
    // an existing wallet. Mirrors createWallet but accepts a mnemonic + passphrase.
    func importWallet(label: String, password: String, mnemonic: String, chainId: Int = 1, passphrase: String? = nil) async throws -> [String: Any] {
        var body: [String: Any] = ["label": label, "password": password, "chain_id": chainId, "mnemonic": mnemonic]
        if let p = passphrase { body["passphrase"] = p }
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/wallets", method: "POST", body: data)
    }

    // POST /nft/transfer { wallet_id, password, to, token_id, contract_address, chain_id }
    func transferNFT(walletId: String, password: String, to: String, tokenId: String, contractAddress: String, chainId: Int) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "to": to,
            "token_id": tokenId, "contract_address": contractAddress, "chain_id": chainId,
        ])
        return try await requestRaw("/nft/transfer", method: "POST", body: body)
    }

    // GET /transactions/{txHash}?chain_id=N -> full receipt.
    func getTransactionReceipt(txHash: String, chainId: Int = 1) async throws -> [String: Any] {
        let safeHash = txHash.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? txHash
        let path = "/transactions/\(safeHash)?chain_id=\(chainId)"
        return try await requestRaw(path)
    }

    // POST /gas/estimate { from, to, value?, data?, chain_id } -> { gas_limit }
    func estimateGas(from: String, to: String, value: String? = nil, data: String? = nil, chainId: Int = 1) async throws -> [String: Any] {
        var body: [String: Any] = ["from": from, "to": to, "chain_id": chainId]
        if let v = value { body["value"] = v }
        if let d = data { body["data"] = d }
        let payload = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/gas/estimate", method: "POST", body: payload)
    }

    // MARK: - Swap execution / AMM

    // POST /swap/execute -> real on-chain action via /send (no fabricated hash).
    func executeSwap(walletId: String, password: String, fromToken: String, toToken: String, fromAmount: String, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password,
            "from_token": fromToken, "to_token": toToken, "from_amount": fromAmount, "chain_id": chainId,
        ])
        return try await requestRaw("/swap/execute", method: "POST", body: body)
    }

    // GET /amm/quote (real on-chain Uniswap-V2 getAmountsOut).
    func getAmmQuote(fromToken: String, toToken: String, fromAmount: String, chainId: Int = 1) async throws -> [String: Any] {
        let f = fromToken.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? fromToken
        let t = toToken.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? toToken
        let a = fromAmount.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? fromAmount
        let path = "/amm/quote?from_token=\(f)&to_token=\(t)&from_amount=\(a)&chain_id=\(chainId)"
        return try await requestRaw(path)
    }

    // POST /amm/swap (real on-chain swapExactTokensForTokens calldata).
    func ammSwap(walletId: String, password: String, fromToken: String, toToken: String, fromAmount: String, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password,
            "from_token": fromToken, "to_token": toToken, "from_amount": fromAmount, "chain_id": chainId,
        ])
        return try await requestRaw("/amm/swap", method: "POST", body: body)
    }

    // MARK: - Staking actions (real on-chain via backend /send)

    // POST /staking/stake { wallet_id, password, asset, amount, chain_id }
    func stake(walletId: String, password: String, asset: String, amount: String, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "asset": asset, "amount": amount, "chain_id": chainId,
        ])
        return try await requestRaw("/staking/stake", method: "POST", body: body)
    }

    // POST /staking/unstake { wallet_id, password, asset, amount, chain_id }
    func unstake(walletId: String, password: String, asset: String, amount: String, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "asset": asset, "amount": amount, "chain_id": chainId,
        ])
        return try await requestRaw("/staking/unstake", method: "POST", body: body)
    }

    // POST /staking/claim { wallet_id, password, asset, chain_id }
    func claim(walletId: String, password: String, asset: String, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "asset": asset, "chain_id": chainId,
        ])
        return try await requestRaw("/staking/claim", method: "POST", body: body)
    }

    // MARK: - Networks alias

    // getNetworks is an alias for /chains (same endpoint, untyped payload).
    func getNetworks() async throws -> [String: Any] {
        return try await requestRaw("/chains")
    }

    // MARK: - Non-EVM (Solana / Bitcoin / Cosmos)

    // POST /non_evm/address { seed, chain_type, chain_id, path? } -> { address }
    func nonEvmAddress(seed: String, chainType: String, chainId: Int, path: String? = nil) async throws -> [String: Any] {
        var body: [String: Any] = ["seed": seed, "chain_type": chainType, "chain_id": chainId]
        if let p = path { body["path"] = p }
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/non_evm/address", method: "POST", body: data)
    }

    // POST /non_evm/sign { seed, chain_type, chain_id, message_hash, path? } -> { signature }
    func nonEvmSign(seed: String, chainType: String, chainId: Int, messageHash: String, path: String? = nil) async throws -> [String: Any] {
        var body: [String: Any] = ["seed": seed, "chain_type": chainType, "chain_id": chainId, "message_hash": messageHash]
        if let p = path { body["path"] = p }
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/non_evm/sign", method: "POST", body: data)
    }

    // POST /non_evm/send { seed, chain_type, chain_id, to, value, path? } -> { signature, raw_tx?, tx_hash? }
    func nonEvmSend(seed: String, chainType: String, chainId: Int, to: String, value: String, path: String? = nil) async throws -> [String: Any] {
        var body: [String: Any] = ["seed": seed, "chain_type": chainType, "chain_id": chainId, "to": to, "value": value]
        if let p = path { body["path"] = p }
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/non_evm/send", method: "POST", body: data)
    }

    // MARK: - Address book

    func getAddressBookContacts() async throws -> [String: Any] {
        return try await requestRaw("/address-book/contacts")
    }

    func addContact(name: String, address: String, chainId: Int? = nil) async throws -> [String: Any] {
        var body: [String: Any] = ["name": name, "address": address]
        if let c = chainId { body["chain_id"] = c }
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/address-book/contacts", method: "POST", body: data)
    }

    func updateContact(id: String, name: String? = nil, address: String? = nil, chainId: Int? = nil) async throws -> [String: Any] {
        var body: [String: Any] = [:]
        if let n = name { body["name"] = n }
        if let a = address { body["address"] = a }
        if let c = chainId { body["chain_id"] = c }
        let safeId = id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? id
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/address-book/contacts/\(safeId)", method: "PUT", body: data)
    }

    func deleteContact(id: String) async throws -> [String: Any] {
        let safeId = id.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? id
        return try await requestRaw("/address-book/contacts/\(safeId)", method: "DELETE")
    }

    // MARK: - Devices

    func getDevices() async throws -> [String: Any] {
        return try await requestRaw("/devices")
    }

    func registerDevice(name: String, deviceType: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: ["name": name, "device_type": deviceType])
        return try await requestRaw("/devices", method: "POST", body: body)
    }

    func syncDevice(deviceId: String) async throws -> [String: Any] {
        let safeId = deviceId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? deviceId
        let body = try JSONSerialization.data(withJSONObject: [String: Any]())
        return try await requestRaw("/devices/\(safeId)/sync", method: "POST", body: body)
    }

    func deleteDevice(deviceId: String) async throws -> [String: Any] {
        let safeId = deviceId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? deviceId
        return try await requestRaw("/devices/\(safeId)", method: "DELETE")
    }

    // MARK: - Token approvals

    func getApprovals(address: String, chainId: Int) async throws -> [String: Any] {
        let a = address.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? address
        let path = "/approvals?address=\(a)&chain_id=\(chainId)"
        return try await requestRaw(path)
    }

    func revokeApproval(approvalId: String) async throws -> [String: Any] {
        let safeId = approvalId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? approvalId
        return try await requestRaw("/approvals/\(safeId)", method: "DELETE")
    }

    // MARK: - Keystore V3 (Web3 Secret Storage)

    // POST /keystore/export { wallet_id, password } -> { keystore }
    func exportKeystore(walletId: String, password: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: ["wallet_id": walletId, "password": password])
        return try await requestRaw("/keystore/export", method: "POST", body: body)
    }

    // POST /keystore/import { keystore, password, label? } -> { wallet_id, address }
    func importKeystore(keystore: String, password: String, label: String? = nil) async throws -> [String: Any] {
        var body: [String: Any] = ["keystore": keystore, "password": password]
        if let l = label { body["label"] = l }
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/keystore/import", method: "POST", body: data)
    }

    // MARK: - Encrypted-seed backup

    // POST /wallets/{walletId}/export-encrypted-seed { password }
    // -> { encrypted_seed, salt, nonce } (real AES-256-GCM).
    func exportEncryptedSeed(walletId: String, password: String) async throws -> [String: Any] {
        let safeId = walletId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? walletId
        let body = try JSONSerialization.data(withJSONObject: ["password": password])
        return try await requestRaw("/wallets/\(safeId)/export-encrypted-seed", method: "POST", body: body)
    }

    // POST /wallets/import-encrypted-seed { encrypted_seed, password, label? }
    // -> { wallet_id, address }
    func importEncryptedSeed(encryptedSeed: String, password: String, label: String? = nil) async throws -> [String: Any] {
        var body: [String: Any] = ["encrypted_seed": encryptedSeed, "password": password]
        if let l = label { body["label"] = l }
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/wallets/import-encrypted-seed", method: "POST", body: data)
    }

    // MARK: - Security scan (scam URL / address check)

    // GET /security/check-url?url= -> { safe, reason? }
    func checkUrl(_ url: String) async throws -> [String: Any] {
        let u = url.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? url
        return try await requestRaw("/security/check-url?url=\(u)")
    }

    // GET /security/check-address?address= -> { safe, reason? }
    func checkAddress(_ address: String) async throws -> [String: Any] {
        let a = address.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? address
        return try await requestRaw("/security/check-address?address=\(a)")
    }

    // POST /security/scan { target } -> { safe, threats[] }
    func securityScan(target: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: ["target": target])
        return try await requestRaw("/security/scan", method: "POST", body: body)
    }

    // MARK: - Lending

    func getLendingMarkets() async throws -> [String: Any] {
        return try await requestRaw("/lending/markets")
    }

    func getLendingPositions() async throws -> [String: Any] {
        return try await requestRaw("/lending/positions")
    }

    func lendingSupply(walletId: String, password: String, asset: String, amount: String, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "asset": asset, "amount": amount, "chain_id": chainId,
        ])
        return try await requestRaw("/lending/supply", method: "POST", body: body)
    }

    func lendingBorrow(walletId: String, password: String, asset: String, amount: String, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "asset": asset, "amount": amount, "chain_id": chainId,
        ])
        return try await requestRaw("/lending/borrow", method: "POST", body: body)
    }

    func lendingWithdraw(walletId: String, password: String, asset: String, amount: String, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "asset": asset, "amount": amount, "chain_id": chainId,
        ])
        return try await requestRaw("/lending/withdraw", method: "POST", body: body)
    }

    func lendingRepay(walletId: String, password: String, asset: String, amount: String, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "asset": asset, "amount": amount, "chain_id": chainId,
        ])
        return try await requestRaw("/lending/repay", method: "POST", body: body)
    }

    // MARK: - Copy trading

    func getCopyTraders() async throws -> [String: Any] {
        return try await requestRaw("/copytrading/traders")
    }

    func followTrader(traderId: String, allocation: String? = nil) async throws -> [String: Any] {
        var body: [String: Any] = ["trader_id": traderId]
        if let a = allocation { body["allocation"] = a }
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestRaw("/copytrading/follow", method: "POST", body: data)
    }

    func stopCopyTrader(copierId: String) async throws -> [String: Any] {
        let safeId = copierId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? copierId
        let body = try JSONSerialization.data(withJSONObject: [String: Any]())
        return try await requestRaw("/copytrading/copiers/\(safeId)/stop", method: "POST", body: body)
    }

    func getCopySignals() async throws -> [String: Any] {
        return try await requestRaw("/copytrading/signals")
    }

    // MARK: - DAO / Governance

    func getDaoProposals() async throws -> [String: Any] {
        return try await requestRaw("/dao/proposals")
    }

    func createDaoProposal(title: String, description: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: ["title": title, "description": description])
        return try await requestRaw("/dao/proposals", method: "POST", body: body)
    }

    func voteDaoProposal(proposalId: String, support: Bool) async throws -> [String: Any] {
        let safeId = proposalId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? proposalId
        let body = try JSONSerialization.data(withJSONObject: ["support": support])
        return try await requestRaw("/dao/proposals/\(safeId)/vote", method: "POST", body: body)
    }

    func getDaoDelegates() async throws -> [String: Any] {
        return try await requestRaw("/dao/delegates")
    }

    // MARK: - Perpetual positions

    func getPerpetualPositions() async throws -> [String: Any] {
        return try await requestRaw("/perpetual/positions")
    }

    func createPerpetualPosition(pair: String, side: String, size: String, leverage: Int, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "pair": pair, "side": side, "size": size, "leverage": leverage, "chain_id": chainId,
        ])
        return try await requestRaw("/perpetual/positions", method: "POST", body: body)
    }

    func closePerpetualPosition(positionId: String) async throws -> [String: Any] {
        let safeId = positionId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? positionId
        let body = try JSONSerialization.data(withJSONObject: [String: Any]())
        return try await requestRaw("/perpetual/positions/\(safeId)/close", method: "POST", body: body)
    }

    // MARK: - Margin positions

    func getMarginPositions() async throws -> [String: Any] {
        return try await requestRaw("/margin/positions")
    }

    func createMarginPosition(pair: String, side: String, size: String, leverage: Int, chainId: Int = 1) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "pair": pair, "side": side, "size": size, "leverage": leverage, "chain_id": chainId,
        ])
        return try await requestRaw("/margin/positions", method: "POST", body: body)
    }

    func closeMarginPosition(positionId: String) async throws -> [String: Any] {
        let safeId = positionId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? positionId
        let body = try JSONSerialization.data(withJSONObject: [String: Any]())
        return try await requestRaw("/margin/positions/\(safeId)/close", method: "POST", body: body)
    }

    // MARK: - Prediction markets

    func getPredictionMarkets() async throws -> [String: Any] {
        return try await requestRaw("/prediction/markets")
    }

    func placePredictionBet(marketId: String, side: String, amount: String) async throws -> [String: Any] {
        let safeId = marketId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? marketId
        let body = try JSONSerialization.data(withJSONObject: ["side": side, "amount": amount])
        return try await requestRaw("/prediction/markets/\(safeId)/bet", method: "POST", body: body)
    }

    // MARK: - Launchpool

    func getLaunchpool() async throws -> [String: Any] {
        return try await requestRaw("/launchpool")
    }

    func getLaunchpoolStakes() async throws -> [String: Any] {
        return try await requestRaw("/launchpool/stakes")
    }

    func launchpoolStake(walletId: String, password: String, amount: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "amount": amount,
        ])
        return try await requestRaw("/launchpool/stake", method: "POST", body: body)
    }

    func launchpoolUnstake(walletId: String, password: String, amount: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "amount": amount,
        ])
        return try await requestRaw("/launchpool/unstake", method: "POST", body: body)
    }

    // MARK: - Token sales

    func getTokenSales() async throws -> [String: Any] {
        return try await requestRaw("/token-sales")
    }

    func participateTokenSale(saleId: String, amount: String) async throws -> [String: Any] {
        let safeId = saleId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? saleId
        let body = try JSONSerialization.data(withJSONObject: ["amount": amount])
        return try await requestRaw("/token-sales/\(safeId)/participate", method: "POST", body: body)
    }

    // MARK: - dApps / Chart / DeFi protocols

    func getDapps() async throws -> [String: Any] {
        return try await requestRaw("/dapps")
    }

    func getDappCategories() async throws -> [String: Any] {
        return try await requestRaw("/dapps/categories")
    }

    func getChartHistory(token: String, days: Int? = nil) async throws -> [String: Any] {
        let t = token.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? token
        let d = days ?? 30
        let path = "/chart/history?token=\(t)&days=\(d)"
        return try await requestRaw(path)
    }

    func getDefiProtocols() async throws -> [String: Any] {
        return try await requestRaw("/defi/protocols")
    }
}

// parsePaymentUri — decodes a scanned QR string (bare 0x address, ethereum:
// URI, or EIP-681 payment URI) into an address + optional amount/chainId/
// tokenAddress. Returns nil when no address can be extracted (fail-closed —
// never a guessed value). Mirrors the web client's parsePaymentUri 1:1.
func parsePaymentUri(_ input: String) -> [String: Any]? {
    let s = input.trimmingCharacters(in: .whitespacesAndNewlines)
    if s.isEmpty { return nil }
    if s.range(of: "^0x[a-fA-F0-9]{40}$", options: .regularExpression) != nil {
        return ["address": s]
    }
    guard s.hasPrefix("ethereum:") else { return nil }
    let body = String(s.dropFirst("ethereum:".count))
    let qIdx = body.firstIndex(of: "?")
    let target: String
    let query: String
    if let q = qIdx {
        target = String(body[body.startIndex..<q])
        query = String(body[body.index(after: q)...])
    } else {
        target = body
        query = ""
    }
    var address = ""
    var tokenAddress: String? = nil
    if let slash = target.firstIndex(of: "/") {
        address = String(target[target.startIndex..<slash])
        let funcPart = String(target[target.index(after: slash)...])
        if funcPart.hasPrefix("transfer") { tokenAddress = "" }
    } else {
        address = target
    }
    if address.range(of: "^0x[a-fA-F0-9]{40}$", options: .regularExpression) == nil { return nil }
    var amount: String? = nil
    var chainId: Int? = nil
    if !query.isEmpty {
        for pair in query.split(separator: "&") {
            let parts = pair.split(separator: "=", maxSplits: 1)
            guard parts.count == 2 else { continue }
            let k = String(parts[0])
            let v = String(parts[1])
            if k == "value" { amount = v }
            else if k == "chainId" { chainId = Int(v) }
            else if k == "address", tokenAddress != nil { tokenAddress = v }
        }
    }
    var result: [String: Any] = ["address": address]
    if let a = amount { result["amount"] = a }
    if let c = chainId { result["chainId"] = c }
    if let t = tokenAddress, !t.isEmpty { result["tokenAddress"] = t }
    return result
}
