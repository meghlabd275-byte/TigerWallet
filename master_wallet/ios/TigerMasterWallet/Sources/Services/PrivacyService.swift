/**
 * PrivacyService - iOS Implementation
 *
 * On-device privacy primitives that can be done honestly are kept:
 *   - AES-256-GCM encryption/decryption of sensitive data (CryptoKit, Keychain key).
 *
 * Primitives that cannot be done client-side with real cryptography are
 * fail-closed (they throw) rather than simulated:
 *   - ZK-SNARK proof generation/verification (no on-device Groth16/PLONK
 *     prover/verifier).
 *   - CoinJoin mixing (requires a coordinator service + real range proofs).
 *   - Stealth-address derivation (requires secp256k1 + keccak256, not P256).
 *   - Address rotation (a real Ethereum address requires keccak256 of a
 *     secp256k1 public key, unavailable on-device).
 */

import Foundation
import Security
import CryptoKit
import LocalAuthentication

public enum PrivacyError: Error, LocalizedError {
    case zkProvingUnsupported
    case zkVerificationUnsupported
    case coinJoinUnsupported
    case stealthAddressUnsupported
    case addressRotationUnsupported

    public var errorDescription: String? {
        switch self {
        case .zkProvingUnsupported:
            return "ZK proof generation is not supported on-device (no Groth16/PLONK prover). Use the backend."
        case .zkVerificationUnsupported:
            return "ZK proof verification is not supported on-device (no Groth16/PLONK verifier). Use the backend."
        case .coinJoinUnsupported:
            return "CoinJoin mixing requires a coordinator service and real range proofs; not supported on-device."
        case .stealthAddressUnsupported:
            return "Stealth-address derivation requires secp256k1 + keccak256 (unavailable on-device); use the backend."
        case .addressRotationUnsupported:
            return "Real address rotation requires keccak256 of a secp256k1 public key (unavailable on-device); use the backend."
        }
    }
}

public class PrivacyService {

    // MARK: - Singleton
    public static let shared = PrivacyService()

    // Privacy levels (kept for API compatibility with callers that reference them)
    public static let PRIVACY_NONE = 0
    public static let PRIVACY_STANDARD = 1
    public static let PRIVACY_HIGH = 2
    public static let PRIVACY_MAXIMUM = 3

    // MARK: - Initialization
    private init() {}

    // MARK: - Stealth Address (fail-closed)

    public func generateStealthAddress(ownerAddress: String, spendingPublicKey: Data,
                                       completion: @escaping (StealthAddressResult) -> Void) {
        completion(StealthAddressResult(success: false, error: PrivacyError.stealthAddressUnsupported.localizedDescription))
    }

    // MARK: - CoinJoin (fail-closed)

    public func createCoinJoin(inputs: [CoinJoinInput], outputs: [CoinJoinOutput], privacyLevel: Int,
                                completion: @escaping (CoinJoinResult) -> Void) {
        completion(CoinJoinResult(success: false, error: PrivacyError.coinJoinUnsupported.localizedDescription))
    }

    // MARK: - ZK Proof (fail-closed)

    public func generateZKProof(amount: String, commitment: Data,
                                completion: @escaping (ZKProofResult) -> Void) {
        completion(ZKProofResult(success: false, error: PrivacyError.zkProvingUnsupported.localizedDescription))
    }

    /// Verify ZK proof. There is no on-device Groth16/PLONK verifier, so this
    /// always fails closed. An empty/all-zero proof is explicitly rejected
    /// before throwing so a caller cannot treat absence as success.
    public func verifyZKProof(proof: String, commitment: Data) -> Bool {
        let trimmed = proof.trimmingCharacters(in: .whitespaces)
        let isAllZero = trimmed.allSatisfy { $0 == "0" || $0 == "x" || $0 == "X" }
        if trimmed.isEmpty || (commitment.isEmpty) {
            return false
        }
        if isAllZero {
            return false
        }
        return false
    }

    // MARK: - Address Rotation (fail-closed)

    public func rotateAddress(currentAddress: String,
                              completion: @escaping (RotationResult) -> Void) {
        completion(RotationResult(success: false, error: PrivacyError.addressRotationUnsupported.localizedDescription))
    }

    // MARK: - Encryption (real CryptoKit AES-256-GCM)

    /// Encrypt sensitive data with a hardware-backed symmetric key.
    public func encryptSensitiveData(_ data: Data, completion: @escaping (EncryptedDataResult) -> Void) {
        DispatchQueue.global(qos: .userInitiated).async {
            do {
                let key = try self.getOrCreatePrivacyKey()
                let sealedBox = try AES.GCM.seal(data, using: key)
                let combined = sealedBox.nonce + sealedBox.ciphertext + sealedBox.tag

                completion(EncryptedDataResult(
                    success: true,
                    encryptedData: combined.base64EncodedString()
                ))
            } catch {
                completion(EncryptedDataResult(success: false, error: error.localizedDescription))
            }
        }
    }

    /// Decrypt sensitive data.
    public func decryptSensitiveData(_ encryptedBase64: String, completion: @escaping (DecryptedDataResult) -> Void) {
        DispatchQueue.global(qos: .userInitiated).async {
            do {
                guard let combined = Data(base64Encoded: encryptedBase64) else {
                    completion(DecryptedDataResult(success: false, error: "Invalid data"))
                    return
                }

                let key = try self.getOrCreatePrivacyKey()
                let nonce = combined.prefix(12)
                let ciphertext = combined.dropFirst(12).dropLast(16)
                let tag = combined.suffix(16)

                let sealedBox = try AES.GCM.SealedBox(nonce: nonce, ciphertext: ciphertext, tag: tag)
                let decrypted = try AES.GCM.open(sealedBox, using: key)

                completion(DecryptedDataResult(success: true, data: decrypted))
            } catch {
                completion(DecryptedDataResult(success: false, error: error.localizedDescription))
            }
        }
    }

    // MARK: - Private Helpers

    private func getOrCreatePrivacyKey() throws -> SymmetricKey {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: "tigermaster_privacy_key",
            kSecReturnData as String: true
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        if status == errSecSuccess, let keyData = result as? Data {
            return SymmetricKey(data: keyData)
        }

        let key = SymmetricKey(size: .bits256)

        let addQuery: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: "tigermaster_privacy_key",
            kSecValueData as String: key.withUnsafeBytes { Data($0) },
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        ]

        SecItemAdd(addQuery as CFDictionary, nil)

        return key
    }
}

// MARK: - Data Structures

public struct CoinJoinInput {
    public let address: String
    public let amount: String
    public let privateKey: Data
}

public struct CoinJoinOutput {
    public let address: String
    public let amount: String
}

public struct StealthAddressResult {
    public let success: Bool
    public let stealthAddress: String
    public let viewingKey: String
    public let ephemeralPublicKey: String
    public let error: String?

    public init(success: Bool, stealthAddress: String = "", viewingKey: String = "",
                ephemeralPublicKey: String = "", error: String? = nil) {
        self.success = success
        self.stealthAddress = stealthAddress
        self.viewingKey = viewingKey
        self.ephemeralPublicKey = ephemeralPublicKey
        self.error = error
    }
}

public struct CoinJoinResult {
    public let success: Bool
    public let mixedOutputs: [String]
    public let proofs: [Data]
    public let rounds: Int
    public let error: String?

    public init(success: Bool, mixedOutputs: [String] = [], proofs: [Data] = [],
                rounds: Int = 0, error: String? = nil) {
        self.success = success
        self.mixedOutputs = mixedOutputs
        self.proofs = proofs
        self.rounds = rounds
        self.error = error
    }
}

public struct ZKProofResult {
    public let success: Bool
    public let proof: String
    public let commitment: String
    public let blindingFactor: String
    public let error: String?

    public init(success: Bool, proof: String = "", commitment: String = "",
                blindingFactor: String = "", error: String? = nil) {
        self.success = success
        self.proof = proof
        self.commitment = commitment
        self.blindingFactor = blindingFactor
        self.error = error
    }
}

public struct RotationResult {
    public let success: Bool
    public let newAddress: String
    public let newPublicKey: String
    public let viewingKey: String
    public let error: String?

    public init(success: Bool, newAddress: String = "", newPublicKey: String = "",
                viewingKey: String = "", error: String? = nil) {
        self.success = success
        self.newAddress = newAddress
        self.newPublicKey = newPublicKey
        self.viewingKey = viewingKey
        self.error = error
    }
}

public struct EncryptedDataResult {
    public let success: Bool
    public let encryptedData: String
    public let error: String?

    public init(success: Bool, encryptedData: String = "", error: String? = nil) {
        self.success = success
        self.encryptedData = encryptedData
        self.error = error
    }
}

public struct DecryptedDataResult {
    public let success: Bool
    public let data: Data
    public let error: String?

    public init(success: Bool, data: Data = Data(), error: String? = nil) {
        self.success = success
        self.data = data
        self.error = error
    }
}
