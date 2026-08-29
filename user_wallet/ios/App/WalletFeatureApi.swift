import Foundation

/// Feature endpoints that mirror the web/desktop/extension clients:
/// multisig (proxied to MasterWallet), non-EVM chains (real derivation +
/// signing), and dApp/WalletConnect pairing (proxied dapp_browser).
/// Every method is a real backend call — no stubs, no fabricated data.
extension UserWalletApiService {

    // ==================== Multisig (/wallet/multisig/* -> MasterWallet) ====================

    func listMultisigWallets() async throws -> [String: Any] {
        return try await requestRaw("/wallet/multisig/wallets")
    }

    func createMultisigWallet(name: String, owners: [String], threshold: Int, chainId: Int) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "name": name, "owners": owners, "threshold": threshold, "chain_id": chainId,
        ])
        return try await requestRaw("/wallet/multisig/wallets", method: "POST", body: body)
    }

    func listMultisigTransactions(walletId: String) async throws -> [String: Any] {
        let enc = walletId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? walletId
        return try await requestRaw("/wallet/multisig/wallets/\(enc)/transactions")
    }

    func createMultisigTransaction(walletId: String, toAddress: String, value: String, data: String) async throws -> [String: Any] {
        let enc = walletId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? walletId
        let body = try JSONSerialization.data(withJSONObject: [
            "to_address": toAddress, "value": value, "data": data,
        ])
        return try await requestRaw("/wallet/multisig/wallets/\(enc)/transactions", method: "POST", body: body)
    }

    func signMultisigTransaction(txId: String) async throws -> [String: Any] {
        let enc = txId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? txId
        return try await requestRaw("/wallet/multisig/transactions/\(enc)/sign", method: "POST", body: Data("{}".utf8))
    }

    func executeMultisigTransaction(txId: String) async throws -> [String: Any] {
        let enc = txId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? txId
        return try await requestRaw("/wallet/multisig/transactions/\(enc)/execute", method: "POST", body: Data("{}".utf8))
    }

    // ==================== Non-EVM chains (real derivation + signing) ====================

    func deriveNonEvmAddress(walletId: String, password: String, chainType: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "chain_type": chainType,
        ])
        return try await requestRaw("/non_evm/address", method: "POST", body: body)
    }

    func nonEvmSignMessage(walletId: String, password: String, message: String, chainType: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: [
            "wallet_id": walletId, "password": password, "message": message, "chain_type": chainType,
        ])
        return try await requestRaw("/non_evm/sign", method: "POST", body: body)
    }

    /// Signs a non-EVM transaction; `extras` carries chain-specific fields
    /// (bitcoin_inputs/bitcoin_outputs/cosmos_sign_doc). Returns the raw
    /// signed payload for broadcast.
    func nonEvmSend(walletId: String, password: String, chainType: String, extras: [String: Any]) async throws -> [String: Any] {
        var dict: [String: Any] = [
            "wallet_id": walletId, "password": password, "chain_type": chainType,
        ]
        for (k, v) in extras { dict[k] = v }
        let body = try JSONSerialization.data(withJSONObject: dict)
        return try await requestRaw("/non_evm/send", method: "POST", body: body)
    }

    // ==================== dApp browser / WalletConnect (proxied dapp_browser :8083) ====================

    func getDappPairings() async throws -> [String: Any] {
        return try await requestRaw("/dapp/pairings")
    }

    func createDappPairing(uri: String) async throws -> [String: Any] {
        let body = try JSONSerialization.data(withJSONObject: ["uri": uri])
        return try await requestRaw("/dapp/pairings", method: "POST", body: body)
    }

    func approveDappPairing(topic: String) async throws -> [String: Any] {
        let enc = topic.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? topic
        return try await requestRaw("/dapp/pairings/\(enc)/approve", method: "POST", body: Data("{}".utf8))
    }

    func rejectDappPairing(topic: String) async throws -> [String: Any] {
        let enc = topic.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? topic
        return try await requestRaw("/dapp/pairings/\(enc)/reject", method: "POST", body: Data("{}".utf8))
    }

    func getDappSessions() async throws -> [String: Any] {
        return try await requestRaw("/dapp/sessions")
    }

    func getDappRequests(topic: String) async throws -> [String: Any] {
        let enc = topic.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? topic
        return try await requestRaw("/dapp/sessions/\(enc)/request")
    }

    func respondToDappRequest(topic: String, requestId: String, approve: Bool, result: String?) async throws -> [String: Any] {
        let encT = topic.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? topic
        let encR = requestId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? requestId
        var dict: [String: Any] = ["approved": approve]
        if let result = result { dict["result"] = result }
        let body = try JSONSerialization.data(withJSONObject: dict)
        return try await requestRaw("/dapp/sessions/\(encT)/request/\(encR)/respond", method: "POST", body: body)
    }
}
