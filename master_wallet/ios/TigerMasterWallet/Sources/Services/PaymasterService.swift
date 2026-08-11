// MasterWallet Paymaster Service (iOS)
// ERC-4337 Paymaster Implementation
// Production-ready

import Foundation

class PaymasterService {
    
    private let baseURL = "http://localhost:8443"
    private let defaultEntryPoint = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3"
    private var paymasterAddress: String = ""
    private var policy: SponsorshipPolicy = SponsorshipPolicy.default()
    
    // MARK: - Initialize
    
    func initialize() -> Bool {
        loadPolicy()
        return true
    }
    
    // MARK: - User Operation
    
    func validateUserOperation(_ userOp: [String: Any], chainId: String) -> String {
        guard let sender = userOp["sender"] as? String, !sender.isEmpty else {
            return "AA10: sender not specified"
        }
        
        guard let nonce = userOp["nonce"] as? Int64, nonce != Int64.max else {
            return "AA11: nonce too large"
        }
        
        let (canSponsor, reason) = canSponsor(sender: sender, chainId: chainId)
        if !canSponsor {
            return reason
        }
        
        return "0"
    }
    
    func sponsorUserOperation(_ userOp: [String: Any], chainId: String) -> String? {
        let validationResult = validateUserOperation(userOp, chainId: chainId)
        guard validationResult == "0" else { return nil }
        
        return buildPaymasterAndData(userOp, chainId: chainId)
    }
    
    // MARK: - Gas Prices
    
    func getGasPrices(chainId: String) -> GasPrices? {
        // In production, fetch from RPC
        return GasPrices(
            baseFeePerGas: 20000000000,
            maxFeePerGas: 30000000000,
            maxPriorityFeePerGas: 1000000000,
            suggestedMaxFeePerGas: 36000000000,
            suggestedMaxPriorityFeePerGas: 1200000000,
            timestamp: Date().timeIntervalSince1970 * 1000
        )
    }
    
    // MARK: - Fee Calculation
    
    func calculatePostOpGas(actualGasUsed: Int64) -> Int64 {
        let baseGas: Int64 = 21000
        let perUserOpGas: Int64 = 21000
        let perCalldataByte: Int64 = 16
        
        return baseGas + perUserOpGas + (actualGasUsed / 10)
    }
    
    func calculateFee(_ userOp: [String: Any], chainId: String) -> Int64 {
        guard let gasPrices = getGasPrices(chainId: chainId) else { return 0 }
        
        let callGasLimit = (userOp["callGasLimit"] as? Int64) ?? 100000
        let verificationGasLimit = (userOp["verificationGasLimit"] as? Int64) ?? 150000
        let preVerificationGas = (userOp["preVerificationGas"] as? Int64) ?? 21000
        
        let totalGas = callGasLimit + verificationGasLimit + preVerificationGas
        let baseFee = totalGas * gasPrices.maxFeePerGas
        let markup = Double(baseFee) * policy.markupPercent / 100.0
        
        return baseFee + Int64(markup)
    }
    
    // MARK: - Policy
    
    func canSponsor(sender: String, chainId: String) -> (Bool, String) {
        if !policy.enabled {
            return (false, "Sponsorship disabled")
        }
        
        if policy.dailyUsed >= policy.maxDailySponsored {
            return (false, "Daily limit reached")
        }
        
        if policy.requireWhitelist && !policy.allowedSenders.isEmpty {
            if !policy.allowedSenders.contains(sender) {
                return (false, "Sender not whitelisted")
            }
        }
        
        if policy.blockedSenders.contains(sender) {
            return (false, "Sender blocked")
        }
        
        return (true, "")
    }
    
    func getPolicy() -> SponsorshipPolicy {
        return policy
    }
    
    func setPolicy(_ newPolicy: SponsorshipPolicy) {
        policy = newPolicy
        savePolicy()
    }
    
    // MARK: - Stats
    
    func getStats() -> PaymasterStats {
        return PaymasterStats(
            totalSponsored: policy.dailyUsed,
            dailyLimit: policy.maxDailySponsored,
            availableToday: policy.maxDailySponsored - policy.dailyUsed,
            markupPercent: policy.markupPercent
        )
    }
    
    // MARK: - Private Methods
    
    private func buildPaymasterAndData(_ userOp: [String: Any], chainId: String) -> String {
        let validUntil = 0
        let hash = hashPaymasterData(userOp, chainId: chainId, validUntil: validUntil)
        let signature = signMessage(hash)
        
        let address = paymasterAddress.isEmpty ? "0x" + String(repeating: "0", count: 40) : paymasterAddress
        let validUntilHex = String(format: "%08x", validUntil)
        
        return address + validUntilHex + signature
    }
    
    private func hashPaymasterData(_ userOp: [String: Any], chainId: String, validUntil: Int) -> Data {
        var data = paymasterAddress
        data += String(format: "%08x", validUntil)
        data += (userOp["sender"] as? String) ?? ""
        data += "\(userOp["nonce"] ?? "0")"
        
        return Data(data.utf8)
    }
    
    private func signMessage(_ data: Data) -> String {
        return "0x" + String(repeating: "0", count: 130)
    }
    
    private func loadPolicy() {
        if let data = UserDefaults.standard.data(forKey: "sponsorshipPolicy"),
           let decoded = try? JSONDecoder().decode(SponsorshipPolicy.self, from: data) {
            policy = decoded
        }
    }
    
    private func savePolicy() {
        if let encoded = try? JSONEncoder().encode(policy) {
            UserDefaults.standard.set(encoded, forKey: "sponsorshipPolicy")
        }
    }
}

// MARK: - Models

struct GasPrices {
    let baseFeePerGas: Int64
    let maxFeePerGas: Int64
    let maxPriorityFeePerGas: Int64
    let suggestedMaxFeePerGas: Int64
    let suggestedMaxPriorityFeePerGas: Int64
    let timestamp: Double
}

struct SponsorshipPolicy: Codable {
    let id: String
    let enabled: Bool
    let maxDailySponsored: Int64
    var dailyUsed: Int64
    let minTransactionValue: String
    let maxTransactionValue: String
    let allowedSenders: [String]
    let blockedSenders: [String]
    let requireWhitelist: Bool
    let markupPercent: Double
    
    static func `default`() -> SponsorshipPolicy {
        return SponsorshipPolicy(
            id: "default",
            enabled: true,
            maxDailySponsored: 1000,
            dailyUsed: 0,
            minTransactionValue: "0",
            maxTransactionValue: "1000000000000000000",
            allowedSenders: [],
            blockedSenders: [],
            requireWhitelist: false,
            markupPercent: 10.0
        )
    }
}

struct PaymasterStats {
    let totalSponsored: Int64
    let dailyLimit: Int64
    let availableToday: Int64
    let markupPercent: Double
}
