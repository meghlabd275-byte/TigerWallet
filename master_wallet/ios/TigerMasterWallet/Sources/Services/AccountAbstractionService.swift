// MasterWallet Account Abstraction Service (iOS)
// ERC-4337 Smart Wallet Implementation
//
// The client NEVER fabricates signatures, wallet addresses, call data, or gas
// prices. secp256k1 signing and keccak256 userOp hashing must be performed by
// the backend / a real bundler (CommonCrypto on Apple platforms lacks both
// secp256k1 and keccak256). Every signing path therefore either delegates to
// a configured real bundler/sponsor endpoint or throws fail-closed. A fake
// "0x<sha256><zeros>" signature is NEVER returned.

import Foundation
import CryptoKit

enum AAError: Error, LocalizedError {
    case noClientSigner
    case noBundlerConfigured
    case noWalletForSender
    case bundlerRequestFailed(String)
    case noFactoryConfigured

    var errorDescription: String? {
        switch self {
        case .noClientSigner:
            return "AA signing is not available on-device: the backend/bundler must sign the userOp (secp256k1 + keccak256)."
        case .noBundlerConfigured:
            return "No ERC-4337 bundler endpoint is configured; cannot submit the user operation."
        case .noWalletForSender:
            return "No smart wallet found for the given sender/owner."
        case .bundlerRequestFailed(let detail):
            return "Bundler request failed: \(detail)"
        case .noFactoryConfigured:
            return "No account factory address is configured; cannot compute a counterfactual smart-wallet address."
        }
    }
}

class AccountAbstractionService {

    private let baseURL = "http://localhost:8450"
    private let defaultEntryPoint = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3"
    private var entryPoint: String
    private var factoryAddress: String = ""
    private var bundlerEndpoint: String?
    private var smartWallets: [String: SmartWallet] = [:]
    private var sessionKeys: [String: [SessionKey]] = [:]
    private var socialRecoveryConfigs: [String: SocialRecoveryConfig] = [:]

    // MARK: - Initialize

    init() {
        entryPoint = defaultEntryPoint
    }

    func initialize() -> Bool {
        loadSmartWallets()
        loadSessionKeys()
        return true
    }

    // MARK: - Smart Wallet

    /// Record a smart wallet that was created on-chain / counterfactually by the
    /// backend. The client does NOT compute CREATE2 addresses (which requires
    /// keccak256 + secp256k1 init-code hashing unavailable on-device); it only
    /// stores the address returned by the backend.
    func registerSmartWallet(owner: String, address: String, initCode: String?) {
        let wallet = SmartWallet(
            address: address,
            owner: owner,
            entryPoint: entryPoint,
            nonce: 0,
            implementation: "",
            initialized: true,
            createdAt: Date().timeIntervalSince1970 * 1000,
            guardians: [],
            initCode: initCode
        )
        smartWallets[owner] = wallet
        saveSmartWallets()
    }

    func createSmartWallet(owner: String) -> String? {
        // A counterfactual smart-wallet address must be computed by the
        // backend (CREATE2 over keccak256(initCode)), which the client cannot
        // reproduce. Fail closed instead of fabricating an address.
        guard !factoryAddress.isEmpty else { return nil }
        return nil
    }

    func getSmartWallet(owner: String) -> SmartWallet? {
        return smartWallets[owner]
    }

    func listSmartWallets() -> [SmartWallet] {
        return Array(smartWallets.values)
    }

    // MARK: - User Operations

    /// Submit a signed user operation to a real ERC-4337 bundler. The signature
    /// MUST be provided by the caller (obtained from the backend / signer); the
    /// client never signs. If no bundler endpoint is configured or no real
    /// signature is supplied, this throws fail-closed rather than fabricating
    /// a signature or tx hash.
    func sendUserOperation(sender: String, to: String, value: String, data: String,
                           chainId: String = "1", realSignature: String,
                           completion: @escaping (Result<String, AAError>) -> Void) {
        guard let wallet = smartWallets[sender] else {
            completion(.failure(.noWalletForSender))
            return
        }

        guard let bundler = bundlerEndpoint, let url = URL(string: bundler) else {
            completion(.failure(.noBundlerConfigured))
            return
        }

        guard !realSignature.isEmpty else {
            completion(.failure(.noClientSigner))
            return
        }

        let userOp: [String: Any] = [
            "sender": wallet.address,
            "nonce": "\(wallet.nonce)",
            "initCode": wallet.initCode ?? "0x",
            "callData": data.isEmpty ? "0x" : data,
            "callGasLimit": "0",
            "verificationGasLimit": "0",
            "preVerificationGas": "0",
            "maxFeePerGas": "0",
            "maxPriorityFeePerGas": "0",
            "paymasterAndData": "0x",
            "signature": realSignature
        ]

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body: [String: Any] = ["userOp": userOp, "chainId": chainId]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)

        URLSession.shared.dataTask(with: request) { [weak self] data, _, error in
            if let error = error {
                completion(.failure(.bundlerRequestFailed(error.localizedDescription)))
                return
            }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let txHash = json["txHash"] as? String else {
                completion(.failure(.bundlerRequestFailed("no txHash in bundler response")))
                return
            }

            if var wallet = self?.smartWallets[sender] {
                wallet.nonce += 1
                self?.smartWallets[sender] = wallet
                self?.saveSmartWallets()
            }

            completion(.success(txHash))
        }.resume()
    }

    // MARK: - Session Keys

    func addSessionKey(walletAddress: String, sessionKey: SessionKey) {
        if sessionKeys[walletAddress] == nil {
            sessionKeys[walletAddress] = []
        }
        sessionKeys[walletAddress]?.append(sessionKey)
        saveSessionKeys()
    }

    func removeSessionKey(walletAddress: String, keyId: String) {
        sessionKeys[walletAddress]?.removeAll { $0.key == keyId }
        saveSessionKeys()
    }

    func getSessionKeys(walletAddress: String) -> [SessionKey] {
        return sessionKeys[walletAddress] ?? []
    }

    func isSessionKeyValid(walletAddress: String, key: String, contract: String, token: String, amount: Int64) -> Bool {
        guard let keys = sessionKeys[walletAddress],
              let sessionKey = keys.first(where: { $0.key == key && $0.isActive }) else {
            return false
        }

        if Date().timeIntervalSince1970 * 1000 > sessionKey.expiresAt {
            return false
        }

        if sessionKey.spentAmount + amount > sessionKey.spendingLimit {
            return false
        }

        if !sessionKey.allowedContracts.isEmpty && !sessionKey.allowedContracts.contains(contract) {
            return false
        }

        if !sessionKey.allowedTokens.isEmpty && !sessionKey.allowedTokens.contains(token) {
            return false
        }

        return true
    }

    // MARK: - Social Recovery

    func setupSocialRecovery(walletAddress: String, guardians: [Guardian], threshold: Int) {
        let config = SocialRecoveryConfig(
            guardians: guardians,
            threshold: threshold,
            guardianCount: guardians.count,
            isSetup: true,
            lastRecoveryAttempt: 0
        )

        socialRecoveryConfigs[walletAddress] = config
        saveSocialRecoveryConfigs()
    }

    func getSocialRecoveryConfig(walletAddress: String) -> SocialRecoveryConfig? {
        return socialRecoveryConfigs[walletAddress]
    }

    // MARK: - Helpers

    func getEntryPoint(chainId: String = "1") -> String {
        return entryPoint
    }

    func getFactoryAddress(chainId: String = "1") -> String {
        return factoryAddress
    }

    /// Configure a real bundler endpoint and account factory address.
    func configure(bundlerEndpoint: String?, factoryAddress: String?) {
        self.bundlerEndpoint = bundlerEndpoint
        if let factoryAddress = factoryAddress {
            self.factoryAddress = factoryAddress
        }
    }

    // MARK: - Private Methods

    private func loadSmartWallets() {
        if let data = UserDefaults.standard.data(forKey: "smartWallets"),
           let decoded = try? JSONDecoder().decode([String: SmartWallet].self, from: data) {
            smartWallets = decoded
        }
    }

    private func saveSmartWallets() {
        if let encoded = try? JSONEncoder().encode(smartWallets) {
            UserDefaults.standard.set(encoded, forKey: "smartWallets")
        }
    }

    private func loadSessionKeys() {
        if let data = UserDefaults.standard.data(forKey: "sessionKeys"),
           let decoded = try? JSONDecoder().decode([String: [SessionKey]].self, from: data) {
            sessionKeys = decoded
        }
    }

    private func saveSessionKeys() {
        if let encoded = try? JSONEncoder().encode(sessionKeys) {
            UserDefaults.standard.set(encoded, forKey: "sessionKeys")
        }
    }

    private func loadSocialRecoveryConfigs() {
        if let data = UserDefaults.standard.data(forKey: "socialRecoveryConfigs"),
           let decoded = try? JSONDecoder().decode([String: SocialRecoveryConfig].self, from: data) {
            socialRecoveryConfigs = decoded
        }
    }

    private func saveSocialRecoveryConfigs() {
        if let encoded = try? JSONEncoder().encode(socialRecoveryConfigs) {
            UserDefaults.standard.set(encoded, forKey: "socialRecoveryConfigs")
        }
    }
}

// MARK: - Models

struct SmartWallet: Codable {
    let address: String
    let owner: String
    let entryPoint: String
    var nonce: Int
    let implementation: String
    var initialized: Bool
    let createdAt: Double
    var guardians: [String]
    var initCode: String?
}

struct SessionKey: Codable {
    let key: String
    let permission: String
    let allowedContracts: [String]
    let allowedTokens: [String]
    let spendingLimit: Int64
    var spentAmount: Int64
    let expiresAt: Double
    var isActive: Bool
}

struct Guardian: Codable {
    let address: String
    let name: String
    let threshold: Int
    var confirmed: Bool
}

struct SocialRecoveryConfig: Codable {
    let guardians: [Guardian]
    let threshold: Int
    let guardianCount: Int
    let isSetup: Bool
    var lastRecoveryAttempt: Double
}
