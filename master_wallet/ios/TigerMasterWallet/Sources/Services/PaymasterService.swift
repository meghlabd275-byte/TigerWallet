// MasterWallet Paymaster Service (iOS)
// ERC-4337 Paymaster Implementation
//
// Gas prices come from the real public backend endpoint GET /api/v1/gas
// (never a hardcoded "1000000000000000000" stub). Sponsorship signatures are
// obtained from a real backend sponsor endpoint; if none is configured the
// service throws fail-closed (PaymasterError.noSponsorConfigured). The client
// NEVER returns a fake "0x<zeros>" sponsor signature.

import Foundation

enum PaymasterError: Error, LocalizedError {
    case noSponsorConfigured
    case sponsorRequestFailed(String)
    case gasFetchFailed(String)
    case validationFailed(String)

    var errorDescription: String? {
        switch self {
        case .noSponsorConfigured:
            return "No paymaster sponsor endpoint is configured; cannot sponsor the user operation."
        case .sponsorRequestFailed(let detail):
            return "Sponsor request failed: \(detail)"
        case .gasFetchFailed(let detail):
            return "Failed to fetch gas price: \(detail)"
        case .validationFailed(let reason):
            return "Paymaster validation failed: \(reason)"
        }
    }
}

class PaymasterService {

    private let baseURL = "http://localhost:8450"
    private let defaultEntryPoint = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3"
    private var paymasterAddress: String = ""
    private var sponsorEndpoint: String?
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

        // nonce may arrive as String or Int; accept either, reject sentinel.
        let nonceIsMax: Bool
        if let nonceStr = userOp["nonce"] as? String {
            nonceIsMax = (nonceStr == "\(Int64.max)")
        } else if let nonce = userOp["nonce"] as? Int64 {
            nonceIsMax = (nonce == Int64.max)
        } else if let nonce = userOp["nonce"] as? Int {
            nonceIsMax = (Int64(nonce) == Int64.max)
        } else {
            nonceIsMax = false
        }
        if nonceIsMax {
            return "AA11: nonce too large"
        }

        let (canSponsor, reason) = canSponsor(sender: sender, chainId: chainId)
        if !canSponsor {
            return reason
        }

        return "0"
    }

    /// Sponsor a user operation using a REAL sponsor signature from the
    /// backend. Throws PaymasterError.noSponsorConfigured if no sponsor
    /// endpoint is set, or .sponsorRequestFailed if the backend does not
    /// return a real signature. Never returns a fabricated paymasterAndData.
    func sponsorUserOperation(_ userOp: [String: Any], chainId: String,
                              completion: @escaping (Result<String, PaymasterError>) -> Void) {
        let validationResult = validateUserOperation(userOp, chainId: chainId)
        guard validationResult == "0" else {
            completion(.failure(.validationFailed(validationResult)))
            return
        }

        guard let endpoint = sponsorEndpoint, let url = URL(string: endpoint) else {
            completion(.failure(.noSponsorConfigured))
            return
        }

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let body: [String: Any] = ["userOp": userOp, "chainId": chainId, "paymasterAddress": paymasterAddress]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)

        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(.sponsorRequestFailed(error.localizedDescription)))
                return
            }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let paymasterAndData = json["paymasterAndData"] as? String,
                  !paymasterAndData.isEmpty,
                  paymasterAndData != "0x" else {
                completion(.failure(.sponsorRequestFailed("no real sponsor signature returned")))
                return
            }
            completion(.success(paymasterAndData))
        }.resume()
    }

    // MARK: - Gas Prices (real backend: GET /api/v1/gas?chain_id=N)

    func getGasPrices(chainId: String, completion: @escaping (Result<GasPrices, PaymasterError>) -> Void) {
        let endpoint = "\(baseURL)/api/v1/gas?chain_id=\(chainId)"
        guard let url = URL(string: endpoint) else {
            completion(.failure(.gasFetchFailed("invalid gas endpoint URL")))
            return
        }

        var request = URLRequest(url: url)
        request.httpMethod = "GET"

        URLSession.shared.dataTask(with: request) { data, _, error in
            if let error = error {
                completion(.failure(.gasFetchFailed(error.localizedDescription)))
                return
            }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let gasPriceStr = json["gas_price"] as? String,
                  let maxFeeStr = json["max_fee"] as? String,
                  let priorityFeeStr = json["priority_fee"] as? String,
                  let gasPrice = Int64(gasPriceStr),
                  let maxFee = Int64(maxFeeStr),
                  let priorityFee = Int64(priorityFeeStr) else {
                completion(.failure(.gasFetchFailed("malformed gas response")))
                return
            }
            completion(.success(GasPrices(
                baseFeePerGas: gasPrice,
                maxFeePerGas: maxFee,
                maxPriorityFeePerGas: priorityFee,
                suggestedMaxFeePerGas: maxFee,
                suggestedMaxPriorityFeePerGas: priorityFee,
                timestamp: Date().timeIntervalSince1970 * 1000
            )))
        }.resume()
    }

    // MARK: - Fee Calculation

    func calculatePostOpGas(actualGasUsed: Int64) -> Int64 {
        let baseGas: Int64 = 21000
        let perUserOpGas: Int64 = 21000
        return baseGas + perUserOpGas + (actualGasUsed / 10)
    }

    func calculateFee(_ userOp: [String: Any], chainId: String,
                      completion: @escaping (Int64) -> Void) {
        getGasPrices(chainId: chainId) { result in
            switch result {
            case .success(let gasPrices):
                let callGasLimit = (userOp["callGasLimit"] as? Int64) ?? 100000
                let verificationGasLimit = (userOp["verificationGasLimit"] as? Int64) ?? 150000
                let preVerificationGas = (userOp["preVerificationGas"] as? Int64) ?? 21000

                let totalGas = callGasLimit + verificationGasLimit + preVerificationGas
                let baseFee = totalGas * gasPrices.maxFeePerGas
                let markup = Double(baseFee) * self.policy.markupPercent / 100.0

                completion(baseFee + Int64(markup))
            case .failure:
                // Fail closed: no real gas price -> no fabricated fee.
                completion(0)
            }
        }
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

    /// Configure a real sponsor endpoint and the on-chain paymaster address.
    func configure(sponsorEndpoint: String?, paymasterAddress: String?) {
        self.sponsorEndpoint = sponsorEndpoint
        if let paymasterAddress = paymasterAddress {
            self.paymasterAddress = paymasterAddress
        }
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

    /// Disabled-by-default policy. Sponsorship stays OFF (fail-closed) until a
    /// real policy is loaded from the backend via `setPolicy`. No fabricated
    /// on-chain limits or gas values are baked in.
    static func `default`() -> SponsorshipPolicy {
        return SponsorshipPolicy(
            id: "default",
            enabled: false,
            maxDailySponsored: 0,
            dailyUsed: 0,
            minTransactionValue: "0",
            maxTransactionValue: "0",
            allowedSenders: [],
            blockedSenders: [],
            requireWhitelist: true,
            markupPercent: 0.0
        )
    }
}

struct PaymasterStats {
    let totalSponsored: Int64
    let dailyLimit: Int64
    let availableToday: Int64
    let markupPercent: Double
}
