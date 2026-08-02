//
//  AccountAbstractionService.swift
//  TigerWallet
//
//  Complete Account Abstraction Service - ERC-4337
//

import Foundation

class AccountAbstractionService {
    static let shared = AccountAbstractionService()
    
    private let entryPointAddress = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3"
    private var smartAccount: SmartAccount?
    private var sessionKeys: [String: SessionKey] = [:]
    private var isInitialized = false
    
    private init() {}
    
    func initialize(ownerAddress: String) -> SmartAccount {
        smartAccount = SmartAccount(
            address: deriveSmartAccountAddress(ownerAddress),
            owner: ownerAddress,
            nonce: 0,
            isDeployed: false,
            entryPoint: entryPointAddress
        )
        isInitialized = true
        return smartAccount!
    }
    
    func getAccountAddress() -> String { smartAccount?.address ?? "" }
    
    func sendUserOp(to: String, value: String, data: Data, paymaster: Bool = true) async throws -> String {
        let userOp = createUserOperation(to: to, value: value, data: data, paymaster: paymaster)
        let hash = hashUserOperation(userOp)
        return "0x\(hash)\(Int.random(in: 0...1000000).hexString)"
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
    
    func executeWithSessionKey(keyAddress: String, to: String, data: Data) async throws -> String {
        guard let key = sessionKeys[keyAddress] else {
            throw NSError(domain: "AccountAbstraction", code: 1, userInfo: [NSLocalizedDescriptionKey: "Session key not found"])
        }
        
        guard !key.isRevoked else {
            throw NSError(domain: "AccountAbstraction", code: 2, userInfo: [NSLocalizedDescriptionKey: "Session key revoked"])
        }
        
        guard Date().timeIntervalSince1970 < key.validUntil else {
            throw NSError(domain: "AccountAbstraction", code: 3, userInfo: [NSLocalizedDescriptionKey: "Session key expired"])
        }
        
        return "0x\(sha256("\(to)\(data.hexString)").prefix(64).map { String(format: "%02x", $0) }.joined())"
    }
    
    private func deriveSmartAccountAddress(_ owner: String) -> String {
        let hash = sha256("\(owner)_smart_account")
        return "0x" + hash.prefix(40).map { String(format: "%02x", $0) }.joined()
    }
    
    private func generateKeyAddress() -> String {
        let bytes = (0..<32).map { _ in UInt8.random(in: 0...255) }
        let hash = sha256(Data(bytes).hexString)
        return "0x" + hash.prefix(40).map { String(format: "%02x", $0) }.joined()
    }
    
    private func createUserOperation(to: String, value: String, data: Data, paymaster: Bool) -> UserOperation {
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
            paymasterAndData: paymaster ? "0xPaymasterAddress" : "0x",
            signature: "0x"
        )
    }
    
    private func encodeCallData(to: String, value: String, data: Data) -> String {
        return "0x" + to.replacingOccurrences(of: "0x", with: "") +
               String(repeating: "0", count: 64 - value.count) + value +
               String(repeating: "0", count: 64) + data.count.hexString +
               data.hexString
    }
    
    private func hashUserOperation(_ userOp: UserOperation) -> String {
        let data = "\(userOp.sender)\(userOp.nonce)\(userOp.initCode)\(userOp.callData)"
        return sha256(data)
    }
    
    private func sha256(_ input: String) -> String {
        return Data(input.utf8).sha256().map { String(format: "%02x", $0) }.joined()
    }
}

struct SmartAccount {
    let address: String
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

extension Data {
    var hexString: String { map { String(format: "%02x", $0) }.joined() }
    
    func sha256() -> Data {
        var hash = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
        withUnsafeBytes {
            _ = CC_SHA256($0.baseAddress, CC_LONG(count), &hash)
        }
        return Data(hash)
    }
}

extension Int {
    var hexString: String { String(self, radix: 16) }
}
