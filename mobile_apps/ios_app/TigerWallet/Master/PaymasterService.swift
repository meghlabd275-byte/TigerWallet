//
//  PaymasterService.swift
//  TigerWallet
//
//  Complete Paymaster Service - Identical across ALL platforms
//
//  Fail-closed: gas sponsorship requires a REAL paymaster that signs the
//  userOpHash with a real secp256k1 ECDSA key (the on-chain Paymaster contract
//  verifies ecrecover(signature, hash) == paymaster owner). There is no
//  paymaster/sponsor endpoint on the backend (go/wallet_api) and no on-device
//  secp256k1 library, so a real sponsor signature cannot be produced or
//  verified here. sponsorUserOp therefore throws fail-closed rather than
//  returning "0xPaymasterAddress" + a fabricated hash. getBalance throws
//  rather than returning a fabricated "1000000000000000000".
//
//  The duplicate `UserOperation` struct and `Data.sha256()` extension that
//  collided with AccountAbstractionService are removed; this file reuses the
//  canonical `UserOperation` defined in AccountAbstractionService.swift.
//

import Foundation

class PaymasterService {
    static let shared = PaymasterService()

    private var whitelistedDApps: [String: WhitelistEntry] = [:]
    private var gasToken: String?

    /// Optional backend paymaster/sponsor endpoint. If a real sponsor endpoint
    /// is later added to go/wallet_api (e.g. POST /api/v1/paymaster/sponsor),
    /// set this URL and sponsorUserOp will POST the full userOp to it and use
    /// the returned real sponsor signature + paymaster address. Empty by
    /// default → sponsorUserOp throws fail-closed.
    var sponsorEndpoint: String = ""

    private init() {}

    /// Gas sponsorship requires a REAL paymaster signature over the userOpHash
    /// (verified on-chain by ecrecover). With no sponsor endpoint configured
    /// and no on-device secp256k1 signer, this throws fail-closed. The previous
    /// implementation returned "0xPaymasterAddress" + a sha256 hash as
    /// `paymasterAndData` — that is removed.
    ///
    /// The real ERC-4337 userOpHash uses Keccak-256, which is not available on
    /// device (CommonCrypto only provides NIST SHA-256). We therefore POST the
    /// full userOp fields to the sponsor endpoint, which computes the real
    /// Keccak userOpHash server-side and signs it with real secp256k1. We never
    /// fabricate a hash or signature locally.
    func sponsorUserOp(_ userOp: UserOperation) async throws -> PaymasterData {
        guard !sponsorEndpoint.isEmpty, let url = URL(string: sponsorEndpoint) else {
            throw PaymasterError.noSponsorConfigured
        }
        let body: [String: Any] = [
            "user_op": [
                "sender": userOp.sender,
                "nonce": userOp.nonce,
                "init_code": userOp.initCode,
                "call_data": userOp.callData,
                "call_gas_limit": userOp.callGasLimit,
                "verification_gas_limit": userOp.verificationGasLimit,
                "pre_verification_gas": userOp.preVerificationGas,
                "max_fee_per_gas": userOp.maxFeePerGas,
                "max_priority_fee_per_gas": userOp.maxPriorityFeePerGas,
                "paymaster_and_data": userOp.paymasterAndData,
                "signature": userOp.signature
            ],
            "entry_point": "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3"
        ]
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let auth = (try? BackendClient.authHeader()) ?? ""
        if !auth.isEmpty {
            request.setValue(auth, forHTTPHeaderField: "Authorization")
        }
        request.httpBody = try JSONSerialization.data(withJSONObject: body)

        let (data, response): (Data, URLResponse)
        do {
            (data, response) = try await URLSession.shared.data(for: request)
        } catch {
            throw PaymasterError.sponsorUnreachable(error.localizedDescription)
        }
        guard let http = response as? HTTPURLResponse else {
            throw PaymasterError.sponsorUnreachable("malformed response")
        }
        guard (200..<300).contains(http.statusCode) else {
            throw PaymasterError.sponsorRejected(http.statusCode, errorMessage(data))
        }
        guard let json = try JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            throw PaymasterError.sponsorUnreachable("malformed JSON")
        }
        // The sponsor returns the REAL paymaster address + its REAL secp256k1
        // ECDSA signature over the Keccak userOpHash it computed server-side.
        // paymasterAndData is encoded as paymasterAddress || validUntil ||
        // signature, per EIP-4337. The signature is verified on-chain by the
        // EntryPoint's ecrecover — the client never trusts it blindly.
        guard let paymasterAddress = json["paymaster_address"] as? String,
              paymasterAddress.hasPrefix("0x"), paymasterAddress.count == 42,
              let signature = json["signature"] as? String, signature.hasPrefix("0x"),
              let validUntil = json["valid_until"] as? String
        else {
            throw PaymasterError.sponsorUnreachable("missing paymaster address or signature")
        }
        let paymasterAndData = paymasterAddress
            + validUntil
            + signature.dropFirst(2)
        return PaymasterData(
            paymasterAndData: paymasterAndData,
            preVerificationGas: userOp.preVerificationGas,
            verificationGasLimit: userOp.verificationGasLimit,
            callGasLimit: userOp.callGasLimit
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

    /// Real paymaster balance requires an on-chain `balanceOf(paymaster)`
    /// eth_call against the EntryPoint. With no paymaster configured, this
    /// throws fail-closed rather than returning a fabricated
    /// "1000000000000000000".
    func getBalance() throws -> String {
        throw PaymasterError.noSponsorConfigured
    }

    // MARK: - Private

    private func errorMessage(_ data: Data) -> String? {
        if let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
            return json["error"] as? String
        }
        return String(data: data, encoding: .utf8)
    }
}

enum PaymasterError: Error, LocalizedError {
    case noSponsorConfigured
    case sponsorUnreachable(String)
    case sponsorRejected(Int, String?)
    case invalidSignature

    var errorDescription: String? {
        switch self {
        case .noSponsorConfigured:
            return "No real paymaster/sponsor endpoint is configured; cannot sponsor a UserOperation or report a balance."
        case .sponsorUnreachable(let msg):
            return "Sponsor endpoint unreachable: \(msg)"
        case .sponsorRejected(let code, let msg):
            return "Sponsor rejected the request (HTTP \(code)\(msg.map { ": \($0)" } ?? ""))."
        case .invalidSignature:
            return "Sponsor signature failed real secp256k1 ECDSA verification."
        }
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
