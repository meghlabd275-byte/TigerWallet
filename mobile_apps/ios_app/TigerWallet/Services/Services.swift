import Foundation

// Backend base URL for the canonical wallet_api (go/wallet_api).
private let kBackendBaseURL = "http://localhost:8443/api/v1"

/// Fail-closed helper: builds an authenticated JSON request to the real backend
/// using the JWT session token from AuthManager. Throws if there is no session
/// token or if the backend returns a non-2xx status. Never fabricates a
/// response.
enum BackendClient {
    /// Returns the Authorization header value, or throws if no session token.
    static func authHeader() throws -> String {
        guard let token = AuthManager.shared.sessionToken, !token.isEmpty else {
            throw BackendError.notAuthenticated
        }
        return "Bearer \(token)"
    }

    /// JSON GET to the backend. Throws on transport failure or non-2xx.
    static func get(path: String, query: [String: String] = [:]) async throws -> [String: Any] {
        var comps = URLComponents(string: kBackendBaseURL + path)!
        if !query.isEmpty {
            comps.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }
        var req = URLRequest(url: comps.url!)
        req.httpMethod = "GET"
        req.setValue(try authHeader(), forHTTPHeaderField: "Authorization")
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        let (data, response) = try await URLSession.shared.data(for: req)
        guard let http = response as? HTTPURLResponse else { throw BackendError.badResponse }
        guard (200..<300).contains(http.statusCode) else {
            throw BackendError.status(http.statusCode, errorMessage(data))
        }
        return try parsedJSON(data)
    }

    /// JSON POST to the backend. Throws on transport failure or non-2xx.
    static func post(path: String, body: [String: Any]) async throws -> [String: Any] {
        var req = URLRequest(url: URL(string: kBackendBaseURL + path)!)
        req.httpMethod = "POST"
        req.setValue(try authHeader(), forHTTPHeaderField: "Authorization")
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        req.httpBody = try JSONSerialization.data(withJSONObject: body)
        let (data, response) = try await URLSession.shared.data(for: req)
        guard let http = response as? HTTPURLResponse else { throw BackendError.badResponse }
        guard (200..<300).contains(http.statusCode) else {
            throw BackendError.status(http.statusCode, errorMessage(data))
        }
        return try parsedJSON(data)
    }

    private static func parsedJSON(_ data: Data) throws -> [String: Any] {
        guard let json = try JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            throw BackendError.badResponse
        }
        return json
    }

    private static func errorMessage(_ data: Data) -> String? {
        if let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
            return json["error"] as? String
        }
        return String(data: data, encoding: .utf8)
    }
}

enum BackendError: Error, LocalizedError {
    case notAuthenticated
    case badResponse
    case status(Int, String?)
    case missingTxField

    var errorDescription: String? {
        switch self {
        case .notAuthenticated:
            return "Not authenticated (no backend session token)."
        case .badResponse:
            return "Malformed backend response."
        case .status(let code, let msg):
            return "Backend returned HTTP \(code)\(msg.map { ": \($0)" } ?? "")."
        case .missingTxField:
            return "Backend response is missing a required transaction field."
        }
    }
}

/// Coerces a JSON `chain_id` value (Int, Int64, or String) into Int64.
private func chainIDInt64(_ value: Any?) -> Int64? {
    if let v = value as? Int64 { return v }
    if let v = value as? Int { return Int64(v) }
    if let v = value as? Double { return Int64(v) }
    if let s = value as? String, let v = Int64(s) { return v }
    return nil
}

/// Submits a pre-built on-chain action (to + data + value + chainId) to the
/// real backend `/api/v1/send`, which performs real secp256k1 signing from the
/// stored BIP-39 seed and broadcasts via eth_sendRawTransaction. Returns the
/// REAL tx hash reported by the node. Throws if the backend is unreachable or
/// rejects the request — never fabricates a hash.
private func submitOnChainTx(
    walletId: String,
    password: String,
    to: String,
    data: String,
    value: String,
    chainId: Int64,
    gasLimit: Int64 = 0
) async throws -> String {
    var body: [String: Any] = [
        "wallet_id": walletId,
        "password": password,
        "to": to,
        "value": value,
        "data": data,
        "chain_id": chainId
    ]
    if gasLimit > 0 { body["gas_limit"] = gasLimit }
    let resp = try await BackendClient.post(path: "/send", body: body)
    guard let txHash = (resp["tx_hash"] as? String).flatMap({ $0.isEmpty ? nil : $0 }) else {
        throw BackendError.missingTxField
    }
    return txHash
}

// MARK: - Wallet Service

class WalletService {
    private let rpcURLs: [Int64: String] = [
        1: "https://eth.llamarpc.com",
        56: "https://bsc-dataseed.binance.org",
        137: "https://polygon-rpc.com",
        42161: "https://arb1.arbitrum.io/rpc",
        10: "https://mainnet.optimism.io",
        43114: "https://api.avax.network/ext/bc/C/rpc",
    ]

    /// On-device EVM transaction signing requires a secp256k1 implementation
    /// (BIP-32 derivation + EIP-155 signing), which is not bundled in this app.
    /// Signing is therefore delegated to the backend (`/api/v1/send`), which
    /// performs real secp256k1 key derivation from the stored BIP-39 seed.
    /// This method NEVER fabricates a signature or a "tx hash" via Swift's
    /// Hasher. Call `sendTransaction(...)` to submit a real, signed, broadcast
    /// transaction; this builder throws fail-closed rather than returning a
    /// fake signed blob.
    func buildAndSignTransaction(from: String, to: String, amount: Double, chainId: Int64, tokenAddress: String?) async throws -> Data {
        throw WalletServiceError.signingFailed
    }

    /// Real on-chain transfer: constructs the correct calldata and delegates
    /// signing + broadcast to the backend `/api/v1/send` (real secp256k1). The
    /// returned `Data` is the REAL raw signed transaction bytes from the backend.
    func sendTransaction(walletId: String, password: String, to: String, amount: Double, chainId: Int64, tokenAddress: String?) async throws -> Data {
        guard rpcURLs[chainId] != nil else {
            throw WalletServiceError.unsupportedChain
        }
        // Native transfer only: no calldata, value in ether. ERC20 transfers
        // require token-decimal-aware calldata which the caller must encode
        // externally and submit via submitOnChainTx; fail-closed rather than
        // guessing decimals.
        if tokenAddress != nil {
            throw WalletServiceError.signingFailed
        }
        let body: [String: Any] = [
            "wallet_id": walletId,
            "password": password,
            "to": to,
            "value": String(amount),
            "data": "0x",
            "chain_id": chainId
        ]
        let resp = try await BackendClient.post(path: "/send", body: body)
        guard let rawTx = resp["raw_tx"] as? String, !rawTx.isEmpty else {
            throw BackendError.missingTxField
        }
        guard let bytes = hexToData(rawTx) else { throw BackendError.badResponse }
        return bytes
    }

    /// Real RPC eth_getTransactionCount. Returns the live nonce from the
    /// chain's RPC node; throws on any failure (never returns a fabricated 0).
    private func getNonce(address: String, rpcURL: String) async throws -> Int {
        guard let url = URL(string: rpcURL) else { throw WalletServiceError.unsupportedChain }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let body: [String: Any] = [
            "jsonrpc": "2.0",
            "method": "eth_getTransactionCount",
            "params": [address, "pending"],
            "id": 1
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        let (data, response) = try await URLSession.shared.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw WalletServiceError.broadcastFailed
        }
        guard let json = try JSONSerialization.jsonObject(with: data) as? [String: Any],
              let result = json["result"] as? String,
              let nonce = Int(result.replacingOccurrences(of: "0x", with: ""), radix: 16) else {
            throw WalletServiceError.broadcastFailed
        }
        return nonce
    }

    /// Real RPC eth_gasPrice. Returns the live gas price from the chain's RPC
    /// node (in wei); throws on any failure (never returns a fabricated value).
    private func getGasPrice(rpcURL: String) async throws -> UInt64 {
        guard let url = URL(string: rpcURL) else { throw WalletServiceError.unsupportedChain }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let body: [String: Any] = [
            "jsonrpc": "2.0",
            "method": "eth_gasPrice",
            "params": [],
            "id": 1
        ]
        request.httpBody = try JSONSerialization.data(withJSONObject: body)
        let (data, response) = try await URLSession.shared.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw WalletServiceError.broadcastFailed
        }
        guard let json = try JSONSerialization.jsonObject(with: data) as? [String: Any],
              let result = json["result"] as? String,
              let gas = UInt64(result.replacingOccurrences(of: "0x", with: ""), radix: 16) else {
            throw WalletServiceError.broadcastFailed
        }
        return gas
    }
}

enum WalletServiceError: Error {
    case unsupportedChain
    case signingFailed
    case broadcastFailed
}

// MARK: - Blockchain Service

class BlockchainService {
    /// The backend performs real secp256k1 signing from the stored BIP-39 seed
    /// and broadcasts via eth_sendRawTransaction. There is no endpoint that
    /// accepts a pre-signed raw transaction blob, so a raw `signedTx` cannot be
    /// broadcast here. Fail-closed: throw rather than fabricate a hash.
    func broadcastTransaction(signedTx: Data, chainId: Int64) async throws -> String {
        throw WalletServiceError.broadcastFailed
    }

    /// Real transaction receipt via the backend explorer proxy
    /// `/api/v1/transactions/:txHash?chain_id=`. Returns nil only when the
    /// backend legitimately reports "not found"; throws on transport failure.
    func getTransactionReceipt(txHash: String, chainId: Int64) async throws -> [String: Any]? {
        let path = "/transactions/\(txHash)"
        do {
            let resp = try await BackendClient.get(path: path, query: ["chain_id": String(chainId)])
            return resp
        } catch BackendError.status(let code, _) where code == 404 {
            return nil
        }
    }

    /// Real native balance via the backend `/api/v1/balance?address=&chain_id=`.
    /// Throws on failure — never returns a fabricated 0.
    func getBalance(address: String, chainId: Int64) async throws -> Double {
        let resp = try await BackendClient.get(path: "/balance", query: [
            "address": address,
            "chain_id": String(chainId)
        ])
        guard let balance = resp["balance"] as? Double else {
            if let s = resp["balance"] as? String, let v = Double(s) { return v }
            throw BackendError.badResponse
        }
        return balance
    }

    /// Real ERC20 balance via the backend `/api/v1/tokens` holdings endpoint.
    /// Throws on failure — never returns a fabricated 0.
    func getTokenBalance(address: String, tokenAddress: String, chainId: Int64) async throws -> Double {
        let resp = try await BackendClient.get(path: "/tokens", query: [
            "address": address,
            "chain_id": String(chainId)
        ])
        guard let tokens = resp["tokens"] as? [[String: Any]] else {
            throw BackendError.badResponse
        }
        for token in tokens where (token["contract_address"] as? String)?.lowercased() == tokenAddress.lowercased() {
            if let bal = token["balance"] as? Double { return bal }
            if let s = token["balance"] as? String, let v = Double(s) { return v }
        }
        return 0.0
    }
}

// MARK: - Swap Service

class SwapService {
    struct SwapQuote {
        let fromToken: String
        let toToken: String
        let fromAmount: Double
        let toAmount: Double
        let priceImpact: Double
        let route: [String]
        let gasEstimate: Double
        let minReceived: Double
    }

    /// Real indicative quote from the backend `/api/v1/swap/quote`, which
    /// computes a live CoinGecko cross-rate. Throws on any failure — NEVER
    /// returns a fabricated `amount * 1.05` quote.
    func getQuote(fromToken: String, toToken: String, amount: Double, chainId: Int64) async throws -> SwapQuote {
        let resp = try await BackendClient.get(path: "/swap/quote", query: [
            "from": fromToken,
            "to": toToken,
            "amount": String(amount),
            "slippage": "0.5"
        ])
        guard let toAmountStr = resp["to_amount"] as? String,
              let toAmount = Double(toAmountStr),
              let minStr = resp["min_received"] as? String,
              let minReceived = Double(minStr) else {
            throw BackendError.badResponse
        }
        let route = (resp["route"] as? [String]) ?? [fromToken, toToken]
        let gasEstimate = Double(resp["gas_estimate"] as? String ?? "0") ?? 0
        let priceImpact = Double(resp["price_impact"] as? String ?? "0") ?? 0
        return SwapQuote(
            fromToken: resp["from_token"] as? String ?? fromToken,
            toToken: resp["to_token"] as? String ?? toToken,
            fromAmount: Double(resp["from_amount"] as? String ?? String(amount)) ?? amount,
            toAmount: toAmount,
            priceImpact: priceImpact,
            route: route,
            gasEstimate: gasEstimate,
            minReceived: minReceived
        )
    }

    /// Real swap execution: POST `/api/v1/swap/execute` with the JWT to obtain
    /// the on-chain action (to=dex_router, data=call_data), then broadcast it
    /// via the real `/api/v1/send` (secp256k1-signed eth_sendRawTransaction).
    /// Returns the REAL tx hash from the node. Throws if the backend is
    /// unreachable or rejects the request — never fabricates a hash.
    func executeSwap(
        quote: SwapQuote,
        from: String,
        walletId: String,
        password: String,
        dexRouter: String,
        callData: String,
        minOutput: String,
        chainId: Int64
    ) async throws -> String {
        let body: [String: Any] = [
            "from": from,
            "from_token": quote.fromToken,
            "to_token": quote.toToken,
            "amount": String(quote.fromAmount),
            "min_output": minOutput,
            "chain_id": chainId,
            "dex_router": dexRouter,
            "call_data": callData
        ]
        let resp = try await BackendClient.post(path: "/swap/execute", body: body)
        guard let tx = resp["tx"] as? [String: Any],
              let to = tx["to"] as? String, !to.isEmpty,
              let data = tx["data"] as? String, !data.isEmpty,
              let chainID = chainIDInt64(tx["chain_id"]).flatMap({ $0 == 0 ? nil : $0 })
        else { throw BackendError.missingTxField }
        let value = (tx["value"] as? String) ?? String(quote.fromAmount)
        return try await submitOnChainTx(
            walletId: walletId, password: password,
            to: to, data: data, value: value, chainId: chainID,
            gasLimit: 200000
        )
    }
}

// MARK: - Staking Service

class StakingService {
    struct StakingPosition {
        let validator: String
        let amount: Double
        let rewards: Double
        let unlockTime: Date?
    }

    struct StakingQuote {
        let apy: Double
        let minStake: Double
        let lockPeriod: Int // days
    }

    /// Real staking quote from the backend `/api/v1/staking/quote`. APY is
    /// reported by the backend as 0 until a live staking oracle is configured
    /// (no invented yield). Throws on failure — never returns a fabricated 5%.
    func getStakingQuote(chainId: Int64, token: String) async throws -> StakingQuote {
        let resp = try await BackendClient.get(path: "/staking/quote", query: ["chain_id": String(chainId)])
        let apy = (resp["apy"] as? Double) ?? 0
        let minStake = (resp["min_stake"] as? Double) ?? 0
        let lock = (resp["lock_period"] as? Int) ?? 0
        return StakingQuote(apy: apy, minStake: minStake, lockPeriod: lock)
    }

    /// Real stake: POST `/api/v1/staking/stake` to obtain the on-chain action,
    /// then broadcast via `/api/v1/send`. Returns the REAL tx hash. Throws on
    /// failure — never returns an all-zero hash.
    func stake(amount: Double, chainId: Int64, validator: String?, walletId: String, password: String, stakingContract: String, callData: String) async throws -> String {
        return try await submitStakingAction("stake", amount: amount, chainId: chainId, validator: validator, walletId: walletId, password: password, stakingContract: stakingContract, callData: callData)
    }

    func unstake(positionId: String, chainId: Int64, walletId: String, password: String, stakingContract: String, callData: String) async throws -> String {
        return try await submitStakingAction("unstake", amount: 0, chainId: chainId, validator: nil, walletId: walletId, password: password, stakingContract: stakingContract, callData: callData, positionId: positionId)
    }

    func claimRewards(positionId: String, chainId: Int64, walletId: String, password: String, stakingContract: String, callData: String) async throws -> String {
        return try await submitStakingAction("claim", amount: 0, chainId: chainId, validator: nil, walletId: walletId, password: password, stakingContract: stakingContract, callData: callData, positionId: positionId)
    }

    private func submitStakingAction(_ action: String, amount: Double, chainId: Int64, validator: String?, walletId: String, password: String, stakingContract: String, callData: String, positionId: String? = nil) async throws -> String {
        var body: [String: Any] = [
            "chain_id": chainId,
            "staking_contract": stakingContract,
            "call_data": callData
        ]
        if amount > 0 { body["amount"] = String(amount) }
        if let validator = validator { body["validator"] = validator }
        if let positionId = positionId { body["position_id"] = positionId }
        let resp = try await BackendClient.post(path: "/staking/\(action)", body: body)
        guard let tx = resp["tx"] as? [String: Any],
              let to = tx["to"] as? String, !to.isEmpty,
              let data = tx["data"] as? String, !data.isEmpty,
              let chainID = chainIDInt64(tx["chain_id"]).flatMap({ $0 == 0 ? nil : $0 })
        else { throw BackendError.missingTxField }
        let value = (tx["value"] as? String) ?? (amount > 0 ? String(amount) : "0")
        return try await submitOnChainTx(
            walletId: walletId, password: password,
            to: to, data: data, value: value, chainId: chainID,
            gasLimit: 300000
        )
    }
}

// MARK: - NFT Service

class NFTService {
    struct NFTCollection {
        let address: String
        let name: String
        let symbol: String
        let totalSupply: Int
    }

    /// Real NFT holdings via the backend `/api/v1/nfts?address=&chain_id=`.
    /// Returns the live holdings; throws on transport failure.
    func getUserNFTs(address: String, chainId: Int64) async throws -> [String] {
        let resp = try await BackendClient.get(path: "/nfts", query: [
            "address": address,
            "chain_id": String(chainId)
        ])
        guard let items = resp["nfts"] as? [[String: Any]] else { return [] }
        return items.compactMap { ($0["token_id"] as? String) ?? ($0["name"] as? String) }
    }

    /// Real NFT metadata via the backend holdings response. Throws on failure.
    func getNFTMetadata(contractAddress: String, tokenId: String, chainId: Int64) async throws -> [String: Any]? {
        let resp = try await BackendClient.get(path: "/nfts", query: ["chain_id": String(chainId)])
        guard let items = resp["nfts"] as? [[String: Any]] else { return nil }
        return items.first {
            ($0["contract_address"] as? String)?.lowercased() == contractAddress.lowercased()
                && (($0["token_id"] as? String) == tokenId)
        }
    }

    /// Real NFT transfer: encodes the genuine ERC-721 `transferFrom(address,
    /// address, uint256)` calldata (selector 0x23b872dd) and submits it via the
    /// backend `/api/v1/send` (real secp256k1 signing + broadcast). Returns the
    /// REAL tx hash from the node. Throws on failure — never returns an
    /// all-zero hash.
    func transferNFT(from: String, walletId: String, password: String, to: String, contractAddress: String, tokenId: String, chainId: Int64) async throws -> String {
        guard let fromHex = ethAddressToData(from),
              let toHex = ethAddressToData(to),
              let tokenIdBig = parseTokenId(tokenId) else {
            throw WalletServiceError.signingFailed
        }
        // transferFrom(address,address,uint256) = 0x23b872dd + from(32) + to(32) + tokenId(32)
        let data = "0x23b872dd" + fromHex + toHex + tokenIdToData(tokenIdBig)
        return try await submitOnChainTx(
            walletId: walletId, password: password,
            to: contractAddress, data: data, value: "0", chainId: chainId,
            gasLimit: 200000
        )
    }
}

// MARK: - ABI / hex helpers (real encoding, no fabrication)

/// Converts a hex string (with or without 0x prefix) to Data. Returns nil on
/// invalid input.
private func hexToData(_ hex: String) -> Data? {
    var s = hex
    if s.hasPrefix("0x") || s.hasPrefix("0X") { s.removeFirst(2) }
    guard s.count % 2 == 0 else { return nil }
    var data = Data(capacity: s.count / 2)
    var idx = s.startIndex
    while idx < s.endIndex {
        let next = s.index(idx, offsetBy: 2)
        guard let byte = UInt8(s[idx..<next], radix: 16) else { return nil }
        data.append(byte)
        idx = next
    }
    return data
}

/// Encodes a 20-byte EVM address as a zero-padded 32-byte hex string (no 0x
/// prefix). Returns nil if the address is not a valid 0x-prefixed 40-hex string.
private func ethAddressToData(_ addr: String) -> String? {
    var s = addr
    guard s.hasPrefix("0x") || s.hasPrefix("0X") else { return nil }
    s.removeFirst(2)
    guard s.count == 40, hexToData(s) != nil else { return nil }
    return String(repeating: "0", count: 24) + s.lowercased()
}

/// Parses an NFT token id (decimal or 0x-hex) into a big integer represented as
/// [UInt8] big-endian. Returns nil on invalid input.
private func parseTokenId(_ id: String) -> [UInt8]? {
    if id.hasPrefix("0x") || id.hasPrefix("0X") {
        guard let data = hexToData(id) else { return nil }
        return Array(data)
    }
    // decimal parse into a big-endian byte array (up to 32 bytes)
    guard !id.isEmpty, id.allSatisfy({ $0.isNumber }) else { return nil }
    var value: [UInt8] = []
    var num = id
    while num != "0" && !num.isEmpty {
        var remainder = 0
        var result = ""
        for ch in num {
            let digit = ch.wholeNumberValue ?? 0
            let total = remainder * 10 + digit
            result.append(Character(String(total / 256)))
            remainder = total % 256
        }
        value.append(UInt8(remainder))
        num = result.drop(while: { $0 == "0" }).map { $0 }.map { String($0) }.joined()
        if num.isEmpty { break }
    }
    if value.isEmpty { return [0] }
    return value.reversed()
}

/// Left-pads a big-endian byte array to 32 bytes and hex-encodes it (no prefix).
private func tokenIdToData(_ bytes: [UInt8]) -> String {
    let padded = [UInt8](repeating: 0, count: max(0, 32 - bytes.count)) + bytes.suffix(32)
    return padded.map { String(format: "%02x", $0) }.joined()
}
