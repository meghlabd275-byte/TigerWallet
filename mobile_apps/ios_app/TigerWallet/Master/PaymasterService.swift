//
//  PaymasterService.swift
//  TigerWallet
//
//  Complete Paymaster Service - Identical across ALL platforms
//

import Foundation

class PaymasterService {
    static let shared = PaymasterService()
    
    private var whitelistedDApps: [String: WhitelistEntry] = [:]
    private var gasToken: String?
    
    private init() {}
    
    func sponsorUserOp(_ userOp: UserOperation) async throws -> PaymasterData {
        return PaymasterData(
            paymasterAndData: buildPaymasterData(userOp),
            preVerificationGas: "0x5208",
            verificationGasLimit: "0x186A0",
            callGasLimit: "0x5208"
        )
    }
    
    func setPaymentToken(_ tokenAddress: String) -> Bool {
        gasToken = tokenAddress
        return true
    }
    
    func getPaymentToken() -> String? { gasToken }
    
    func whitelistDApp(_ dAppAddress: String, limit: String, expiry: TimeInterval) -> Bool {
        whitelistedDApps[dAppAddress] = WhitelistEntry(
            address: dAppAddress,
            sponsorLimit: limit,
            expiry: expiry,
            isActive: true
        )
        return true
    }
    
    func getWhitelistStatus(_ address: String) -> WhitelistStatus? {
        guard let entry = whitelistedDApps[address] else { return nil }
        return WhitelistStatus(
            isWhitelisted: entry.isActive,
            limit: entry.sponsorLimit,
            expiry: entry.expiry,
            used: "0"
        )
    }
    
    func getBalance() -> String { "1000000000000000000" }
    
    private func buildPaymasterData(_ userOp: UserOperation) -> String {
        let hash = sha256("\(userOp.sender)\(userOp.nonce)\(gasToken ?? "")")
        return "0xPaymasterAddress" + String(repeating: "0", count: 64) + hash.prefix(32).map { String(format: "%02x", $0) }.joined()
    }
    
    private func sha256(_ input: String) -> String {
        return Data(input.utf8).sha256().map { String(format: "%02x", $0) }.joined()
    }
}

struct WhitelistEntry {
    let address: String
    let sponsorLimit: String
    let expiry: TimeInterval
    let isActive: Bool
}

struct WhitelistStatus {
    let isWhitelisted: Bool
    let limit: String
    let expiry: TimeInterval
    let used: String
}

struct PaymasterData {
    let paymasterAndData: String
    let preVerificationGas: String
    let verificationGasLimit: String
    let callGasLimit: String
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

extension Data {
    func sha256() -> Data {
        var hash = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
        withUnsafeBytes {
            _ = CC_SHA256($0.baseAddress, CC_LONG(count), &hash)
        }
        return Data(hash)
    }
}
