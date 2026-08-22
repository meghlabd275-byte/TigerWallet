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

    func updatePolicy(walletId: String, policyId: String, policy: Policy) async throws -> Policy {
        let body = try JSONEncoder().encode(policy)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/policies/\(policyId)", method: "PUT", body: body)
    }

    func getFees(walletId: String) async throws -> [Fee] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/fees")
    }

    func createFee(walletId: String, fee: Fee) async throws -> Fee {
        let body = try JSONEncoder().encode(fee)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/fees", method: "POST", body: body)
    }

    func updateFee(walletId: String, feeId: String, updates: [String: Any]) async throws {
        let body = try JSONSerialization.data(withJSONObject: updates)
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/fees/\(feeId)", method: "PUT", body: body)
    }

    func deleteFee(walletId: String, feeId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/fees/\(feeId)", method: "DELETE")
    }

    func getAutoSignRules(walletId: String) async throws -> [AutoSignRule] {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/auto-sign")
    }

    func createAutoSignRule(walletId: String, rule: AutoSignRule) async throws -> AutoSignRule {
        let body = try JSONEncoder().encode(rule)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/auto-sign", method: "POST", body: body)
    }

    func updateAutoSignRule(walletId: String, ruleId: String, updates: [String: Any]) async throws {
        let body = try JSONSerialization.data(withJSONObject: updates)
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/auto-sign/\(ruleId)", method: "PUT", body: body)
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

    func updateUser(walletId: String, userId: String, updates: [String: Any]) async throws {
        let body = try JSONSerialization.data(withJSONObject: updates)
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/users/\(userId)", method: "PUT", body: body)
    }

    func deleteUser(walletId: String, userId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/users/\(userId)", method: "DELETE")
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

    // MARK: - Notifications

    func getNotifications(walletId: String) async throws -> [MasterNotification] {
        let resp: NotificationsResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/notifications")
        return resp.notifications
    }

    func createNotification(walletId: String, notification: CreateNotificationRequest) async throws -> MasterNotification {
        let body = try JSONEncoder().encode(notification)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/notifications", method: "POST", body: body)
    }

    func updateNotification(walletId: String, notificationId: String, updates: [String: Any]) async throws {
        let body = try JSONSerialization.data(withJSONObject: updates)
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/notifications/\(notificationId)", method: "PUT", body: body)
    }

    // MARK: - Webhooks

    func getWebhooks(walletId: String) async throws -> [Webhook] {
        let resp: WebhooksResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/webhooks")
        return resp.webhooks
    }

    func createWebhook(walletId: String, webhook: CreateWebhookRequest) async throws -> Webhook {
        let body = try JSONEncoder().encode(webhook)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/webhooks", method: "POST", body: body)
    }

    func updateWebhook(walletId: String, webhookId: String, updates: [String: Any]) async throws {
        let body = try JSONSerialization.data(withJSONObject: updates)
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/webhooks/\(webhookId)", method: "PUT", body: body)
    }

    func deleteWebhook(walletId: String, webhookId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/webhooks/\(webhookId)", method: "DELETE")
    }

    // MARK: - Treasury

    func getTreasury(walletId: String) async throws -> TreasuryOverview {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/treasury")
    }

    func getTreasuryTransactions(walletId: String) async throws -> [MasterTransaction] {
        let resp: TransactionsResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/treasury/transactions")
        return resp.transactions
    }

    func treasuryTransfer(walletId: String, to: String, amount: String, password: String) async throws -> SignResponse {
        let body = try JSONEncoder().encode(["to": to, "amount": amount, "password": password])
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/treasury/transfer", method: "POST", body: body)
    }

    func treasurySweep(walletId: String, to: String, password: String) async throws -> SignResponse {
        let body = try JSONEncoder().encode(["to": to, "password": password])
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/treasury/sweep", method: "POST", body: body)
    }

    // MARK: - Multisig

    func getMultisigWallets(walletId: String) async throws -> [MultisigWallet] {
        let resp: MultisigWalletsResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/multisig/wallets")
        return resp.wallets
    }

    func createMultisigWallet(walletId: String, name: String, owners: [String], threshold: Int) async throws -> MultisigWallet {
        let body = try JSONEncoder().encode(["name": name, "owners": owners, "threshold": threshold])
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/multisig/wallets", method: "POST", body: body)
    }

    func getMultisigTransactions(walletId: String, multisigWalletId: String) async throws -> [MultisigTransaction] {
        let resp: MultisigTransactionsResponse = try await request(endpoint: "/api/v1/master-wallet/\(walletId)/multisig/wallets/\(multisigWalletId)/transactions")
        return resp.transactions
    }

    func createMultisigTransaction(walletId: String, multisigWalletId: String, payload: CreateMultisigTransactionRequest) async throws -> MultisigTransaction {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/multisig/wallets/\(multisigWalletId)/transactions", method: "POST", body: body)
    }

    func signMultisigTransaction(walletId: String, transactionId: String) async throws -> MultisigTransaction {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/multisig/transactions/\(transactionId)/sign", method: "POST")
    }

    func executeMultisigTransaction(walletId: String, transactionId: String) async throws -> MultisigTransaction {
        return try await request(endpoint: "/api/v1/master-wallet/\(walletId)/multisig/transactions/\(transactionId)/execute", method: "POST")
    }

    // MARK: - Master Wallet Update / Detail (new endpoints)

    /// Update a master wallet's mutable fields.
    /// PUT /api/v1/master-wallet/:id — body {name?, is_active?, daily_limit?,
    /// per_transaction_limit?, metadata?} → {id, updated:bool}
    func updateMasterWallet(
        masterId: String,
        name: String? = nil,
        isActive: Bool? = nil,
        dailyLimit: Double? = nil,
        perTransactionLimit: Double? = nil,
        metadata: [String: Any]? = nil
    ) async throws -> MasterWalletUpdateResult {
        var payload: [String: Any] = [:]
        if let name = name { payload["name"] = name }
        if let isActive = isActive { payload["is_active"] = isActive }
        if let dailyLimit = dailyLimit { payload["daily_limit"] = dailyLimit }
        if let perTransactionLimit = perTransactionLimit { payload["per_transaction_limit"] = perTransactionLimit }
        if let metadata = metadata { payload["metadata"] = metadata }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await request(endpoint: "/api/v1/master-wallet/\(masterId)", method: "PUT", body: body)
    }

    /// Fetch a single transaction by id.
    /// GET /api/v1/master-wallet/:id/transactions/:tid → {transaction: {...}}
    func getTransaction(masterId: String, txId: String) async throws -> MasterTransaction {
        let resp: TransactionResponse = try await request(endpoint: "/api/v1/master-wallet/\(masterId)/transactions/\(txId)")
        return resp.transaction
    }

    /// Fetch a multisig wallet's detail (owners, threshold, address, pending txs).
    /// GET /api/v1/master-wallet/:id/multisig/wallets/:wid → {multisig_wallet: {...}}
    func getMultisigWalletDetail(masterId: String, walletId: String) async throws -> MultisigWalletDetail {
        let resp: MultisigWalletDetailResponse = try await request(endpoint: "/api/v1/master-wallet/\(masterId)/multisig/wallets/\(walletId)")
        return resp.multisigWallet
    }

    // MARK: - Passkeys (new endpoints)

    /// Register a platform passkey with the backend. All credential material is
    /// base64url-encoded as required by POST /passkey/register.
    func registerPasskey(
        masterId: String,
        credentialId: String,
        publicKey: String,
        signCount: UInt32,
        transports: [String],
        label: String
    ) async throws -> PasskeyRegisterResult {
        let payload: [String: Any] = [
            "credential_id": credentialId,
            "public_key": publicKey,
            "sign_count": signCount,
            "transports": transports,
            "label": label
        ]
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await request(endpoint: "/api/v1/master-wallet/\(masterId)/passkey/register", method: "POST", body: body)
    }

    /// List passkey credentials registered for a master wallet.
    /// GET /api/v1/master-wallet/:id/passkey/credentials → {passkeys: [...]}
    func listPasskeys(masterId: String) async throws -> [PasskeyCredential] {
        let resp: PasskeysResponse = try await request(endpoint: "/api/v1/master-wallet/\(masterId)/passkey/credentials")
        return resp.passkeys
    }

    /// Delete a registered passkey credential.
    /// DELETE /api/v1/master-wallet/:id/passkey/credentials/:credId → 204
    func deletePasskey(masterId: String, credId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(masterId)/passkey/credentials/\(credId)", method: "DELETE")
    }

    /// Verify a passkey assertion server-side.
    /// POST /api/v1/master-wallet/:id/passkey/verify-assertion → {verified:bool, credential_id}
    func verifyPasskeyAssertion(
        masterId: String,
        credentialId: String,
        authData: String,
        clientDataJson: String,
        signature: String
    ) async throws -> PasskeyVerifyResult {
        let payload: [String: Any] = [
            "credential_id": credentialId,
            "authenticator_data": authData,
            "client_data_json": clientDataJson,
            "signature": signature
        ]
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await request(endpoint: "/api/v1/master-wallet/\(masterId)/passkey/verify-assertion", method: "POST", body: body)
    }

    // MARK: - User Wallet: EVM Chains

    /// List UserWallet-managed EVM chains.
    /// GET /api/v1/master-wallet/:id/user-chains/evm
    func listUserEVMChains(id: String) async throws -> [[String: Any]] {
        return try await requestJSONArray(endpoint: "/api/v1/master-wallet/\(id)/user-chains/evm")
    }

    /// Add an EVM chain to a UserWallet.
    /// POST /api/v1/master-wallet/:id/user-chains/evm
    func addUserEVMChain(
        id: String,
        chainId: Int,
        name: String,
        symbol: String,
        rpcUrl: String,
        explorerUrl: String,
        decimals: Int,
        derivationPath: String
    ) async throws -> [String: Any] {
        let payload: [String: Any] = [
            "chain_id": chainId,
            "name": name,
            "symbol": symbol,
            "rpc_url": rpcUrl,
            "explorer_url": explorerUrl,
            "decimals": decimals,
            "derivation_path": derivationPath
        ]
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/user-chains/evm", method: "POST", body: body)
    }

    /// Update a UserWallet EVM chain.
    /// PUT /api/v1/master-wallet/:id/user-chains/evm/:chainId
    func updateUserEVMChain(
        id: String,
        chainId: Int,
        name: String? = nil,
        symbol: String? = nil,
        rpcUrl: String? = nil,
        explorerUrl: String? = nil,
        decimals: Int? = nil,
        derivationPath: String? = nil
    ) async throws -> [String: Any] {
        var payload: [String: Any] = [:]
        if let name = name { payload["name"] = name }
        if let symbol = symbol { payload["symbol"] = symbol }
        if let rpcUrl = rpcUrl { payload["rpc_url"] = rpcUrl }
        if let explorerUrl = explorerUrl { payload["explorer_url"] = explorerUrl }
        if let decimals = decimals { payload["decimals"] = decimals }
        if let derivationPath = derivationPath { payload["derivation_path"] = derivationPath }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/user-chains/evm/\(chainId)", method: "PUT", body: body)
    }

    /// Remove a UserWallet EVM chain.
    /// DELETE /api/v1/master-wallet/:id/user-chains/evm/:chainId
    func removeUserEVMChain(id: String, chainId: Int) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(id)/user-chains/evm/\(chainId)", method: "DELETE")
    }

    // MARK: - User Wallet: Non-EVM Chains

    /// List UserWallet-managed non-EVM chains.
    /// GET /api/v1/master-wallet/:id/user-chains/nonevm
    func listUserNonEVMChains(id: String) async throws -> [[String: Any]] {
        return try await requestJSONArray(endpoint: "/api/v1/master-wallet/\(id)/user-chains/nonevm")
    }

    /// Add a non-EVM chain to a UserWallet.
    /// POST /api/v1/master-wallet/:id/user-chains/nonevm
    func addUserNonEVMChain(
        id: String,
        chainId: Int,
        name: String,
        symbol: String,
        chainType: String,
        rpcUrl: String,
        derivationPath: String,
        addressPrefix: String
    ) async throws -> [String: Any] {
        let payload: [String: Any] = [
            "chain_id": chainId,
            "name": name,
            "symbol": symbol,
            "chain_type": chainType,
            "rpc_url": rpcUrl,
            "derivation_path": derivationPath,
            "address_prefix": addressPrefix
        ]
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/user-chains/nonevm", method: "POST", body: body)
    }

    /// Update a UserWallet non-EVM chain.
    /// PUT /api/v1/master-wallet/:id/user-chains/nonevm/:chainId
    func updateUserNonEVMChain(
        id: String,
        chainId: Int,
        name: String? = nil,
        symbol: String? = nil,
        chainType: String? = nil,
        rpcUrl: String? = nil,
        derivationPath: String? = nil,
        addressPrefix: String? = nil
    ) async throws -> [String: Any] {
        var payload: [String: Any] = [:]
        if let name = name { payload["name"] = name }
        if let symbol = symbol { payload["symbol"] = symbol }
        if let chainType = chainType { payload["chain_type"] = chainType }
        if let rpcUrl = rpcUrl { payload["rpc_url"] = rpcUrl }
        if let derivationPath = derivationPath { payload["derivation_path"] = derivationPath }
        if let addressPrefix = addressPrefix { payload["address_prefix"] = addressPrefix }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/user-chains/nonevm/\(chainId)", method: "PUT", body: body)
    }

    /// Remove a UserWallet non-EVM chain.
    /// DELETE /api/v1/master-wallet/:id/user-chains/nonevm/:chainId
    func removeUserNonEVMChain(id: String, chainId: Int) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(id)/user-chains/nonevm/\(chainId)", method: "DELETE")
    }

    // MARK: - User Wallet: Tokens

    /// List UserWallet-managed tokens (optionally filtered by chain).
    /// GET /api/v1/master-wallet/:id/user-tokens?chain_id=
    func listUserTokens(id: String, chainId: Int? = nil) async throws -> [[String: Any]] {
        var endpoint = "/api/v1/master-wallet/\(id)/user-tokens"
        if let chainId = chainId {
            endpoint += "?chain_id=\(chainId)"
        }
        return try await requestJSONArray(endpoint: endpoint)
    }

    /// Add a token to a UserWallet.
    /// POST /api/v1/master-wallet/:id/user-tokens
    func addUserToken(
        id: String,
        chainId: Int,
        contractAddress: String,
        symbol: String,
        name: String,
        decimals: Int,
        isNative: Bool
    ) async throws -> [String: Any] {
        let payload: [String: Any] = [
            "chain_id": chainId,
            "contract_address": contractAddress,
            "symbol": symbol,
            "name": name,
            "decimals": decimals,
            "is_native": isNative
        ]
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/user-tokens", method: "POST", body: body)
    }

    /// Update a UserWallet token.
    /// PUT /api/v1/master-wallet/:id/user-tokens/:tokenId
    func updateUserToken(
        id: String,
        tokenId: String,
        symbol: String? = nil,
        name: String? = nil,
        decimals: Int? = nil,
        isNative: Bool? = nil
    ) async throws -> [String: Any] {
        var payload: [String: Any] = [:]
        if let symbol = symbol { payload["symbol"] = symbol }
        if let name = name { payload["name"] = name }
        if let decimals = decimals { payload["decimals"] = decimals }
        if let isNative = isNative { payload["is_native"] = isNative }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/user-tokens/\(tokenId)", method: "PUT", body: body)
    }

    /// Remove a UserWallet token.
    /// DELETE /api/v1/master-wallet/:id/user-tokens/:tokenId
    func removeUserToken(id: String, tokenId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(id)/user-tokens/\(tokenId)", method: "DELETE")
    }

    // MARK: - User Wallet: Address Derivation

    /// Derive a UserWallet address (mnemonic processed server-side).
    /// POST /api/v1/master-wallet/:id/derive-user-address
    func deriveUserAddress(
        id: String,
        mnemonic: String,
        chainId: Int,
        chainType: String,
        derivationPath: String,
        accountIndex: Int
    ) async throws -> [String: Any] {
        let payload: [String: Any] = [
            "mnemonic": mnemonic,
            "chain_id": chainId,
            "chain_type": chainType,
            "derivation_path": derivationPath,
            "account_index": accountIndex
        ]
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/derive-user-address", method: "POST", body: body)
    }

    /// List derived UserWallet addresses.
    /// GET /api/v1/master-wallet/:id/user-wallet-addresses
    func listUserWalletAddresses(id: String) async throws -> [[String: Any]] {
        return try await requestJSONArray(endpoint: "/api/v1/master-wallet/\(id)/user-wallet-addresses")
    }

    // MARK: - User Wallet: Auto-Sign

    /// Auto-sign a transaction for a UserWallet (mnemonic processed server-side).
    /// POST /api/v1/master-wallet/:id/auto-sign-transaction
    func autoSignTransaction(
        id: String,
        mnemonic: String,
        chainId: Int,
        chainType: String,
        txType: String,
        toAddress: String,
        value: String,
        tokenAddress: String? = nil
    ) async throws -> [String: Any] {
        var payload: [String: Any] = [
            "mnemonic": mnemonic,
            "chain_id": chainId,
            "chain_type": chainType,
            "tx_type": txType,
            "to_address": toAddress,
            "value": value
        ]
        if let tokenAddress = tokenAddress { payload["token_address"] = tokenAddress }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/auto-sign-transaction", method: "POST", body: body)
    }

    /// List UserWallet auto-sign logs.
    /// GET /api/v1/master-wallet/:id/auto-sign-logs
    func listAutoSignLogs(id: String) async throws -> [[String: Any]] {
        return try await requestJSONArray(endpoint: "/api/v1/master-wallet/\(id)/auto-sign-logs")
    }

    // MARK: - Auto-sign bridge (MasterWallet-owner policy auto-approval)

    /// POST /api/v1/master-wallet/:id/user-wallet-auto-sign
    func userWalletAutoSign(id: String, body: [String: Any]) async throws -> [String: Any] {
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/user-wallet-auto-sign", method: "POST", body: data)
    }

    /// POST /api/v1/master-wallet/:id/check-auto-sign-policy
    func checkAutoSignPolicy(id: String, body: [String: Any]) async throws -> [String: Any] {
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/check-auto-sign-policy", method: "POST", body: data)
    }

    // MARK: - User Wallet: Feature Flags

    /// List UserWallet feature flags.
    /// GET /api/v1/master-wallet/:id/feature-flags
    func listFeatureFlags(id: String) async throws -> [[String: Any]] {
        return try await requestJSONArray(endpoint: "/api/v1/master-wallet/\(id)/feature-flags")
    }

    /// Add a UserWallet feature flag.
    /// POST /api/v1/master-wallet/:id/feature-flags
    func addFeatureFlag(
        id: String,
        flagKey: String,
        flagValue: String,
        description: String? = nil,
        isEnabled: Bool
    ) async throws -> [String: Any] {
        var payload: [String: Any] = [
            "flag_key": flagKey,
            "flag_value": flagValue,
            "is_enabled": isEnabled
        ]
        if let description = description { payload["description"] = description }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/feature-flags", method: "POST", body: body)
    }

    /// Update a UserWallet feature flag.
    /// PUT /api/v1/master-wallet/:id/feature-flags/:flagId
    func updateFeatureFlag(
        id: String,
        flagId: String,
        flagValue: String? = nil,
        description: String? = nil,
        isEnabled: Bool? = nil
    ) async throws -> [String: Any] {
        var payload: [String: Any] = [:]
        if let flagValue = flagValue { payload["flag_value"] = flagValue }
        if let description = description { payload["description"] = description }
        if let isEnabled = isEnabled { payload["is_enabled"] = isEnabled }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await requestJSON(endpoint: "/api/v1/master-wallet/\(id)/feature-flags/\(flagId)", method: "PUT", body: body)
    }

    /// Remove a UserWallet feature flag.
    /// DELETE /api/v1/master-wallet/:id/feature-flags/:flagId
    func removeFeatureFlag(id: String, flagId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/master-wallet/\(id)/feature-flags/\(flagId)", method: "DELETE")
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

    /// GET /api/v1/health (alias of /health).
    func getApiHealth() async throws -> HealthResponse {
        return try await request(endpoint: "/api/v1/health", auth: false)
    }

    func getTransactionHistory(address: String, chainId: Int) async throws -> [MasterTransaction] {
        let resp: TransactionsResponse = try await request(
            endpoint: "/api/v1/transactions/history?address=\(address)&chain_id=\(chainId)",
            auth: false
        )
        return resp.transactions
    }

    // MARK: - Generic Request

    /// POST /api/v1/master-wallet/:id/withdrawal-request — two-party gate.
    /// Body {to_address, amount_wei, currency?, chain_id?} → {withdrawal_id, status}.
    func requestWithdrawal(masterId: String, toAddress: String, amountWei: String, currency: String?, chainId: Int?) async throws -> WithdrawalRequestResult {
        var payload: [String: Any] = [
            "to_address": toAddress,
            "amount_wei": amountWei
        ]
        if let currency = currency { payload["currency"] = currency }
        if let chainId = chainId { payload["chain_id"] = chainId }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await request(endpoint: "/api/v1/master-wallet/\(masterId)/withdrawal-request", method: "POST", body: body)
    }

    /// POST /api/v1/master-wallet/:id/revenue-payout — executes a two-party
    /// gate payout. Body {to, amount, password, gas_limit?, withdrawal_id}
    /// → {transaction_hash, status, withdrawal_id?, from?, chain_id?}.
    func revenuePayout(masterId: String, to: String, amount: String, password: String, gasLimit: Int?, withdrawalId: String) async throws -> RevenuePayoutResult {
        var payload: [String: Any] = [
            "to": to,
            "amount": amount,
            "password": password,
            "withdrawal_id": withdrawalId
        ]
        if let gasLimit = gasLimit { payload["gas_limit"] = gasLimit }
        let body = try JSONSerialization.data(withJSONObject: payload)
        return try await request(endpoint: "/api/v1/master-wallet/\(masterId)/revenue-payout", method: "POST", body: body)
    }

    // MARK: - Generic Request Helpers

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

    /// Performs a real authenticated HTTP request and returns the JSON object
    /// (dict) from the response body. Used by UserWallet routes whose response
    /// shapes are not modeled as Codable structs.
    private func requestJSON(endpoint: String, method: String = "GET", body: Data? = nil) async throws -> [String: Any] {
        let data = try await rawData(endpoint: endpoint, method: method, body: body, auth: true)
        let json = try JSONSerialization.jsonObject(with: data, options: [.allowFragments])
        guard let dict = json as? [String: Any] else {
            throw APIError(code: "PARSE_ERROR", message: "Expected JSON object for \(endpoint)")
        }
        return dict
    }

    /// Performs a real authenticated HTTP GET and returns the JSON array from
    /// the response body. Tries the common canonical list-wrapper keys before
    /// falling back to the top-level array.
    private func requestJSONArray(endpoint: String) async throws -> [[String: Any]] {
        let data = try await rawData(endpoint: endpoint, method: "GET", body: nil, auth: true)
        let json = try JSONSerialization.jsonObject(with: data, options: [.allowFragments])

        if let arr = json as? [[String: Any]] {
            return arr
        }
        if let dict = json as? [String: Any] {
            for key in ["data", "items", "chains", "tokens", "addresses", "logs", "flags"] {
                if let arr = dict[key] as? [[String: Any]] {
                    return arr
                }
            }
        }
        throw APIError(code: "PARSE_ERROR", message: "Expected JSON array for \(endpoint)")
    }

    /// Shared low-level request that returns the raw response bytes. Carries
    /// the Bearer JWT for protected routes.
    private func rawData(endpoint: String, method: String, body: Data?, auth: Bool) async throws -> Data {
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

        return data
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

// MARK: - Notification / Webhook Models

struct MasterNotification: Codable, Identifiable {
    let id: String
    let type: String
    let title: String
    let message: String
    let read: Bool
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id, type, title, message, read
        case createdAt = "created_at"
    }
}

struct NotificationsResponse: Codable {
    let notifications: [MasterNotification]
}

struct CreateNotificationRequest: Codable {
    let type: String
    let title: String
    let message: String
}

struct Webhook: Codable, Identifiable {
    let id: String
    let url: String
    let events: [String]
    let active: Bool
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id, url, events, active
        case createdAt = "created_at"
    }
}

struct WebhooksResponse: Codable {
    let webhooks: [Webhook]
}

struct CreateWebhookRequest: Codable {
    let url: String
    let events: [String]
}

// MARK: - Multisig Models

struct MultisigWallet: Codable, Identifiable {
    let id: String
    let name: String
    let address: String
    let owners: [String]
    let threshold: Int
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id, name, address, owners, threshold
        case createdAt = "created_at"
    }
}

struct MultisigWalletsResponse: Codable {
    let wallets: [MultisigWallet]
}

struct MultisigTransaction: Codable, Identifiable {
    let id: String
    let multisigWalletId: String
    let to: String
    let amount: String
    let data: String?
    let status: String
    let confirmations: Int
    let threshold: Int
    let createdAt: Date
    var executedAt: Date?

    enum CodingKeys: String, CodingKey {
        case id
        case multisigWalletId = "multisig_wallet_id"
        case to, amount, data, status, confirmations, threshold
        case createdAt = "created_at"
        case executedAt = "executed_at"
    }
}

struct MultisigTransactionsResponse: Codable {
    let transactions: [MultisigTransaction]
}

struct CreateMultisigTransactionRequest: Codable {
    let to: String
    let amount: String
    let data: String?
}

// MARK: - Master Wallet Update / Transaction Detail Models

struct MasterWalletUpdateResult: Codable {
    let id: String
    let updated: Bool
}

/// Wrapping response for GET /transactions/:tid → {transaction: {...}}.
struct TransactionResponse: Codable {
    let transaction: MasterTransaction
}

// MARK: - Multisig Detail Model

struct MultisigWalletDetail: Codable, Identifiable {
    let id: String
    let name: String
    let owners: [String]
    let threshold: Int
    let chainId: Int
    let address: String
    var pendingTransactions: [MultisigTransaction]?

    enum CodingKeys: String, CodingKey {
        case id, name, owners, threshold, address
        case chainId = "chain_id"
        case pendingTransactions = "pending_transactions"
    }
}

struct MultisigWalletDetailResponse: Codable {
    let multisigWallet: MultisigWalletDetail

    enum CodingKeys: String, CodingKey {
        case multisigWallet = "multisig_wallet"
    }
}

// MARK: - Passkey Models

/// Backend representation of a registered passkey credential.
/// GET /passkey/credentials → {passkeys: [...]}
struct PasskeyCredential: Codable, Identifiable {
    let id: String
    let credentialId: String
    let signCount: UInt32
    let transports: [String]
    let label: String
    let createdAt: Date
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case credentialId = "credential_id"
        case signCount = "sign_count"
        case transports, label
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct PasskeysResponse: Codable {
    let passkeys: [PasskeyCredential]
}

/// POST /passkey/register → {passkey_id, credential_id, registered:bool}
struct PasskeyRegisterResult: Codable {
    let passkeyId: String
    let credentialId: String
    let registered: Bool

    enum CodingKeys: String, CodingKey {
        case passkeyId = "passkey_id"
        case credentialId = "credential_id"
        case registered
    }
}

/// POST /passkey/verify-assertion → {verified:bool, credential_id}
struct PasskeyVerifyResult: Codable {
    let verified: Bool
    let credentialId: String

    enum CodingKeys: String, CodingKey {
        case verified
        case credentialId = "credential_id"
    }
}

// MARK: - Two-Party Gate Models

/// POST /api/v1/master-wallet/:id/withdrawal-request → {withdrawal_id, status}
struct WithdrawalRequestResult: Codable {
    let withdrawalId: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case withdrawalId = "withdrawal_id"
        case status
    }
}

/// POST /api/v1/master-wallet/:id/revenue-payout →
/// {transaction_hash, status, withdrawal_id?, from?, chain_id?}
struct RevenuePayoutResult: Codable {
    let transactionHash: String
    let status: String
    let withdrawalId: String?
    let from: String?
    let chainId: Int?

    enum CodingKeys: String, CodingKey {
        case transactionHash = "transaction_hash"
        case status
        case withdrawalId = "withdrawal_id"
        case from
        case chainId = "chain_id"
    }
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
