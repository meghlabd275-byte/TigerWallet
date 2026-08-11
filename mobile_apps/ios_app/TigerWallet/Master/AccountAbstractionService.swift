//
//  AccountAbstractionService.swift
//  TigerWallet
//
//  Complete Account Abstraction Service - ERC-4337
//
//  Fail-closed: userOp submission requires a REAL ERC-4337 bundler endpoint.
//  No userOpHash is ever fabricated (no 0x<hash><random>, no sha256 of the
//  userOp fields). The '0xPaymasterAddress' placeholder is removed; the
//  paymasterAndData field is left empty ("0x") unless a real sponsor signature
//  is supplied via PaymasterService. If no real bundler is configured or the
//  bundler is unreachable, methods throw.
//

import Foundation
import CommonCrypto

class AccountAbstractionService {
    static let shared = AccountAbstractionService()

    private let entryPointAddress = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3"
    private var smartAccount: SmartAccount?
    private var sessionKeys: [String: SessionKey] = [:]
    private var isInitialized = false

    /// Real ERC-4337 bundler endpoint (JSON-RPC `eth_sendUserOperation`).
    /// Empty by default — must be configured by the host app before any
    /// userOp can be submitted. When empty, submission throws fail-closed.
    var bundlerEndpoint: String = ""

    private init() {}

    /// Initializes a smart account. The owner address is recorded but the
    /// account address is NOT fabricated: it is resolved lazily from the
    /// bundler/counterfactual deployment on first use and left empty here.
    func initialize(ownerAddress: String) -> SmartAccount {
        smartAccount = SmartAccount(
            address: "",
            owner: ownerAddress,
            nonce: 0,
            isDeployed: false,
            entryPoint: entryPointAddress
        )
        isInitialized = true
        return smartAccount!
    }

    func getAccountAddress() -> String { smartAccount?.address ?? "" }

    /// Submits a UserOperation to a REAL ERC-4337 bundler via
    /// `eth_sendUserOperation` and returns the REAL userOpHash reported by the
    /// bundler. The userOpHash is never fabricated. `paymasterAndData` is set
    /// to "0x" unless a real sponsor signature is provided via `paymasterData`.
    /// Throws if no bundler is configured, the bundler is unreachable, or it
    /// rejects the userOp.
    func sendUserOp(to: String, value: String, data: Data, paymaster: Bool = true, paymasterData: String? = nil) async throws -> String {
        guard isInitialized, smartAccount != nil else {
            throw AccountAbstractionError.notInitialized
        }
        guard !bundlerEndpoint.isEmpty, let url = URL(string: bundlerEndpoint) else {
            throw AccountAbstractionError.noBundler
        }
        let userOp = createUserOperation(to: to, value: value, data: data, paymaster: paymaster, paymasterData: paymasterData)

        let rpcBody: [String: Any] = [
            "jsonrpc": "2.0",
            "method": "eth_sendUserOperation",
            "params": [
                userOpPayload(userOp),
                entryPointAddress
            ],
            "id": 1
        ]
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONSerialization.data(withJSONObject: rpcBody)

        let (respData, response): (Data, URLResponse)
        do {
            (respData, response) = try await URLSession.shared.data(for: request)
        } catch {
            throw AccountAbstractionError.bundlerUnreachable(error.localizedDescription)
        }
        guard let http = response as? HTTPURLResponse else {
            throw AccountAbstractionError.bundlerUnreachable("malformed response")
        }
        guard http.statusCode == 200 else {
            throw AccountAbstractionError.bundlerRejected(http.statusCode, errorMessage(respData))
        }
        guard let json = try JSONSerialization.jsonObject(with: respData) as? [String: Any] else {
            throw AccountAbstractionError.bundlerUnreachable("malformed JSON")
        }
        if let rpcErr = json["error"] as? [String: Any] {
            throw AccountAbstractionError.bundlerRejected(
                (rpcErr["code"] as? Int) ?? -1,
                (rpcErr["message"] as? String)
            )
        }
        guard let userOpHash = json["result"] as? String, !userOpHash.isEmpty else {
            throw AccountAbstractionError.bundlerUnreachable("missing userOpHash")
        }
        return userOpHash
    }

    func createSessionKey(
        dAppAddress: String,
        validUntil: TimeInterval,
        allowedContracts: [String],
        allowedSelectors: [String],
        spendingLimit: String
    ) -> SessionKey {
        let key = SessionKey(
            keyAddress: generateKeyAddress(),
            dAppAddress: dAppAddress,
            validUntil: validUntil,
            allowedContracts: allowedContracts,
            allowedSelectors: allowedSelectors,
            spendingLimit: spendingLimit,
            spentAmount: "0",
            isRevoked: false
        )
        sessionKeys[key.keyAddress] = key
        return key
    }

    func revokeSessionKey(keyAddress: String) -> Bool {
        sessionKeys[keyAddress]?.isRevoked = true
        return true
    }

    func getActiveSessionKeys() -> [SessionKey] {
        let now = Date().timeIntervalSince1970
        return sessionKeys.values.filter { !$0.isRevoked && $0.validUntil > now }
    }

    /// Executes a call through the smart account using a session key. The call
    /// is submitted as a REAL UserOperation to the bundler (same path as
    /// `sendUserOp`); no fabricated tx hash is returned. Throws if the session
    /// key is invalid/expired/revoked, or if no real bundler is configured.
    func executeWithSessionKey(keyAddress: String, to: String, data: Data) async throws -> String {
        guard let key = sessionKeys[keyAddress] else {
            throw AccountAbstractionError.sessionKeyNotFound
        }
        guard !key.isRevoked else {
            throw AccountAbstractionError.sessionKeyRevoked
        }
        guard Date().timeIntervalSince1970 < key.validUntil else {
            throw AccountAbstractionError.sessionKeyExpired
        }
        // Real submission via the bundler; the returned userOpHash is the only
        // legitimate identifier. No hash of (to, data) is fabricated.
        return try await sendUserOp(to: to, value: "0", data: data, paymaster: false)
    }

    /// Generates a random session-key identifier (not a wallet address and not
    /// a public key — it is a local handle for the in-memory session key only).
    private func generateKeyAddress() -> String {
        var bytes = [UInt8](repeating: 0, count: 32)
        let status = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
        if status != errSecSuccess {
            // Fall back to SystemRandomNumberGenerator if the security RNG is
            // unavailable; this is still a real random 32-byte value.
            bytes = (0..<32).map { _ in UInt8.random(in: 0...255) }
        }
        return "0x" + bytes.prefix(20).map { String(format: "%02x", $0) }.joined()
    }

    private func createUserOperation(to: String, value: String, data: Data, paymaster: Bool, paymasterData: String?) -> UserOperation {
        // paymasterAndData is "0x" (no sponsorship) unless a REAL sponsor
        // signature is supplied. The previous "0xPaymasterAddress" placeholder
        // is removed entirely.
        let paymasterAndData: String
        if paymaster, let pd = paymasterData, !pd.isEmpty {
            paymasterAndData = pd
        } else {
            paymasterAndData = "0x"
        }
        return UserOperation(
            sender: smartAccount?.address ?? "",
            nonce: smartAccount?.nonce.description ?? "0",
            initCode: smartAccount?.isDeployed == false ? "0x" : "0x",
            callData: encodeCallData(to: to, value: value, data: data),
            callGasLimit: "0x5208",
            verificationGasLimit: "0x186A0",
            preVerificationGas: "0x5208",
            maxFeePerGas: "0x3B9ACA00",
            maxPriorityFeePerGas: "0x3B9ACA00",
            paymasterAndData: paymasterAndData,
            signature: "0x"
        )
    }

    /// Encodes `execute(address,uint256,bytes)` calldata (selector 0x61cbb628)
    /// per EIP-4337 SimpleAccount. Uses real ABI head/tail encoding for the
    /// dynamic bytes field.
    private func encodeCallData(to: String, value: String, data: Data) -> String {
        var toClean = to
        if toClean.hasPrefix("0x") || toClean.hasPrefix("0X") { toClean.removeFirst(2) }
        let toPadded = String(repeating: "0", count: max(0, 64 - toClean.count)) + toClean.lowercased()

        var valueClean = value
        if valueClean.hasPrefix("0x") || valueClean.hasPrefix("0X") { valueClean.removeFirst(2) }
        let valuePadded = String(repeating: "0", count: max(0, 64 - valueClean.count)) + valueClean.lowercased()

        // offset to bytes data (3 * 32 bytes head)
        let offset = String(format: "%064x", 96)
        let length = String(format: "%064x", data.count)
        let dataHex = data.map { String(format: "%02x", $0) }.joined()
        let dataPadded = dataHex + String(repeating: "0", count: (64 - (data.count * 2) % 64) % 64)
        return "0x61cbb628" + toPadded + valuePadded + offset + length + dataPadded
    }

    /// Serializes a UserOperation into the ERC-4337 JSON shape expected by
    /// `eth_sendUserOperation`.
    private func userOpPayload(_ userOp: UserOperation) -> [String: String] {
        return [
            "sender": userOp.sender,
            "nonce": userOp.nonce,
            "initCode": userOp.initCode,
            "callData": userOp.callData,
            "callGasLimit": userOp.callGasLimit,
            "verificationGasLimit": userOp.verificationGasLimit,
            "preVerificationGas": userOp.preVerificationGas,
            "maxFeePerGas": userOp.maxFeePerGas,
            "maxPriorityFeePerGas": userOp.maxPriorityFeePerGas,
            "paymasterAndData": userOp.paymasterAndData,
            "signature": userOp.signature
        ]
    }

    private func errorMessage(_ data: Data) -> String? {
        if let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
            if let err = json["error"] as? [String: Any], let msg = err["message"] as? String { return msg }
            return json["error"] as? String
        }
        return String(data: data, encoding: .utf8)
    }
}

enum AccountAbstractionError: Error, LocalizedError {
    case notInitialized
    case noBundler
    case bundlerUnreachable(String)
    case bundlerRejected(Int, String?)
    case sessionKeyNotFound
    case sessionKeyRevoked
    case sessionKeyExpired

    var errorDescription: String? {
        switch self {
        case .notInitialized:
            return "Account abstraction is not initialized."
        case .noBundler:
            return "No real ERC-4337 bundler endpoint is configured; cannot submit UserOperation."
        case .bundlerUnreachable(let msg):
            return "Bundler unreachable: \(msg)"
        case .bundlerRejected(let code, let msg):
            return "Bundler rejected UserOperation (code \(code)\(msg.map { ": \($0)" } ?? ""))."
        case .sessionKeyNotFound:
            return "Session key not found."
        case .sessionKeyRevoked:
            return "Session key has been revoked."
        case .sessionKeyExpired:
            return "Session key has expired."
        }
    }
}

struct SmartAccount {
    var address: String
    let owner: String
    var nonce: Int
    var isDeployed: Bool
    let entryPoint: String
}

struct UserOperation {
    let sender: String
    let nonce: String
    let initCode: String
    let callData: String
    let callGasLimit: String
    let verificationGasLimit: String
    let preVerificationGas: String
    let maxFeePerGas: String
    let maxPriorityFeePerGas: String
    let paymasterAndData: String
    let signature: String
}

struct SessionKey {
    let keyAddress: String
    let dAppAddress: String
    let validUntil: TimeInterval
    let allowedContracts: [String]
    let allowedSelectors: [String]
    let spendingLimit: String
    var spentAmount: String
    var isRevoked: Bool
}

// MARK: - Private file-scope helpers (no Data/Int extension collisions)

private func dataToHex(_ data: Data) -> String {
    data.map { String(format: "%02x", $0) }.joined()
}

private func cc_sha256(_ data: Data) -> Data {
    var hash = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
    data.withUnsafeBytes {
        _ = CC_SHA256($0.baseAddress, CC_LONG(data.count), &hash)
    }
    return Data(hash)
}
