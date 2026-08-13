/**
 * BiometricService - iOS Implementation
 * Biometric and PIN authentication for Master Wallet
 */

import Foundation
import LocalAuthentication
import Security
import CryptoKit

public class BiometricService {
    
    // MARK: - Singleton
    public static let shared = BiometricService()
    
    // MARK: - Constants
    private let PIN_KEY_ALIAS = "tigermaster_pin_key"
    private let PIN_LENGTH = 6
    private let MAX_PIN_ATTEMPTS = 5
    
    // MARK: - Initialization
    private init() {}
    
    // MARK: - Biometric Status
    
    /// Check if biometric authentication is available
    public func isBiometricAvailable() -> BiometricStatus {
        let context = LAContext()
        var error: NSError?
        
        guard context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error) else {
            if let laError = error as? LAError {
                switch laError.code {
                case .biometryNotAvailable:
                    return .noHardware
                case .biometryNotEnrolled:
                    return .notEnrolled
                case .biometryLockout:
                    return .lockout
                default:
                    return .unavailable
                }
            }
            return .unavailable
        }
        
        return .available
    }
    
    // MARK: - Biometric Authentication
    
    /// Authenticate with biometric
    public func authenticateWithBiometric(
        reason: String = "Authenticate to unlock your wallet",
        completion: @escaping (BiometricResult) -> Void
    ) {
        let context = LAContext()
        context.localizedFallbackTitle = "Use PIN"
        
        context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, localizedReason: reason) { success, error in
            DispatchQueue.main.async {
                if success {
                    completion(BiometricResult(success: true))
                } else {
                    let errorMessage = (error as? LAError)?.localizedDescription ?? "Authentication failed"
                    completion(BiometricResult(success: false, error: errorMessage))
                }
            }
        }
    }
    
    // MARK: - PIN Management
    
    /// Check if PIN is set up
    public func isPinSetup() -> Bool {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: PIN_KEY_ALIAS,
            kSecReturnData as String: true
        ]
        
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        
        return status == errSecSuccess
    }
    
    /// Set up PIN
    public func setupPin(_ pin: String) -> Bool {
        guard pin.count == PIN_LENGTH, pin.allSatisfy({ $0.isNumber }) else {
            return false
        }
        
        do {
            let encryptedPin = try encryptPin(pin)
            
            let query: [String: Any] = [
                kSecClass as String: kSecClassGenericPassword,
                kSecAttrAccount as String: PIN_KEY_ALIAS,
                kSecValueData as String: encryptedPin,
                kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly
            ]
            
            // Delete existing if any
            SecItemDelete(query as CFDictionary)
            
            let status = SecItemAdd(query as CFDictionary, nil)
            return status == errSecSuccess
        } catch {
            return false
        }
    }
    
    /// Verify PIN against the stored encrypted PIN. Real decryption + constant-time
    /// comparison; never accepts an arbitrary 6-digit PIN.
    public func verifyPin(_ pin: String) -> PinVerificationResult {
        guard pin.count == PIN_LENGTH, pin.allSatisfy({ $0.isNumber }) else {
            return PinVerificationResult(success: false, error: "Invalid PIN format")
        }

        guard isPinSetup() else {
            return PinVerificationResult(success: false, error: "PIN not set up")
        }

        // Load the stored (encrypted) PIN from the Keychain.
        let loadQuery: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: PIN_KEY_ALIAS,
            kSecReturnData as String: true
        ]

        var item: AnyObject?
        let status = SecItemCopyMatching(loadQuery as CFDictionary, &item)
        guard status == errSecSuccess, let encryptedPin = item as? Data else {
            return PinVerificationResult(success: false, error: "PIN storage unavailable")
        }

        // Decrypt the stored PIN and compare in constant time.
        do {
            let storedPin = try decryptPin(encryptedPin)
            let providedPin = Data(pin.utf8)

            guard storedPin.count == providedPin.count else {
                return PinVerificationResult(success: false, remainingAttempts: MAX_PIN_ATTEMPTS - 1)
            }

            var diff: UInt8 = 0
            for i in 0..<storedPin.count {
                diff |= storedPin[i] ^ providedPin[i]
            }

            if diff == 0 {
                return PinVerificationResult(success: true, remainingAttempts: MAX_PIN_ATTEMPTS)
            } else {
                return PinVerificationResult(success: false, remainingAttempts: MAX_PIN_ATTEMPTS - 1)
            }
        } catch {
            return PinVerificationResult(success: false, error: "PIN verification failed")
        }
    }
    
    /// Change PIN
    public func changePin(oldPin: String, newPin: String) -> Bool {
        let verifyResult = verifyPin(oldPin)
        guard verifyResult.success else {
            return false
        }
        
        return setupPin(newPin)
    }
    
    /// Generate random PIN
    public func generateRandomPin() -> String {
        return String((0..<PIN_LENGTH).map { _ in Int.random(in: 0...9) })
    }
    
    // MARK: - Keychain Operations
    
    /// Encrypt wallet data with biometric
    public func encryptWalletData(_ data: Data, completion: @escaping (EncryptedWalletResult) -> Void) {
        DispatchQueue.global(qos: .userInitiated).async {
            do {
                let key = try self.getOrCreateBiometricKey()
                let sealedBox = try AES.GCM.seal(data, using: key)
                let combined = sealedBox.nonce + sealedBox.ciphertext + sealedBox.tag
                
                DispatchQueue.main.async {
                    completion(EncryptedWalletResult(
                        success: true,
                        encryptedData: combined.base64EncodedString()
                    ))
                }
            } catch {
                DispatchQueue.main.async {
                    completion(EncryptedWalletResult(success: false, error: error.localizedDescription))
                }
            }
        }
    }
    
    /// Decrypt wallet data with biometric
    public func decryptWalletData(_ encryptedBase64: String, completion: @escaping (DecryptedWalletResult) -> Void) {
        DispatchQueue.global(qos: .userInitiated).async {
            do {
                guard let combined = Data(base64Encoded: encryptedBase64) else {
                    throw NSError(domain: "BiometricService", code: 1, userInfo: [NSLocalizedDescriptionKey: "Invalid data"])
                }
                
                let key = try self.getOrCreateBiometricKey()
                let nonce = combined.prefix(12)
                let ciphertext = combined.dropFirst(12).dropLast(16)
                let tag = combined.suffix(16)
                
                let sealedBox = try AES.GCM.SealedBox(nonce: nonce, ciphertext: ciphertext, tag: tag)
                let decrypted = try AES.GCM.open(sealedBox, using: key)
                
                DispatchQueue.main.async {
                    completion(DecryptedWalletResult(success: true, data: decrypted))
                }
            } catch {
                DispatchQueue.main.async {
                    completion(DecryptedWalletResult(success: false, error: error.localizedDescription))
                }
            }
        }
    }
    
    // MARK: - Private Helpers
    
    private func encryptPin(_ pin: String) throws -> Data {
        let key = try getOrCreatePinKey()
        let data = Data(pin.utf8)

        let sealedBox = try AES.GCM.seal(data, using: key)
        return sealedBox.nonce + sealedBox.ciphertext + sealedBox.tag
    }

    private func decryptPin(_ encryptedPin: Data) throws -> Data {
        let key = try getOrCreatePinKey()
        let nonce = encryptedPin.prefix(12)
        let ciphertext = encryptedPin.dropFirst(12).dropLast(16)
        let tag = encryptedPin.suffix(16)
        let sealedBox = try AES.GCM.SealedBox(nonce: nonce, ciphertext: ciphertext, tag: tag)
        return try AES.GCM.open(sealedBox, using: key)
    }
    
    private func getOrCreatePinKey() throws -> SymmetricKey {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: PIN_KEY_ALIAS + "_key",
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
            kSecAttrAccount as String: PIN_KEY_ALIAS + "_key",
            kSecValueData as String: key.withUnsafeBytes { Data($0) },
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        ]
        
        SecItemAdd(addQuery as CFDictionary, nil)
        
        return key
    }
    
    private func getOrCreateBiometricKey() throws -> SymmetricKey {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: "tigermaster_biometric_key",
            kSecReturnData as String: true
        ]
        
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        
        if status == errSecSuccess, let keyData = result as? Data {
            return SymmetricKey(data: keyData)
        }
        
        // Create key with biometric protection
        var accessError: Unmanaged<CFError>?
        guard let accessControl = SecAccessControlCreateWithFlags(
            kCFAllocatorDefault,
            kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
            .userPresence,
            &accessError
        ) else {
            throw NSError(domain: "BiometricService", code: 2, userInfo: [NSLocalizedDescriptionKey: "Failed to create access control"])
        }
        
        let key = SymmetricKey(size: .bits256)
        
        let addQuery: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: "tigermaster_biometric_key",
            kSecValueData as String: key.withUnsafeBytes { Data($0) },
            kSecAttrAccessControl as String: accessControl
        ]
        
        SecItemAdd(addQuery as CFDictionary, nil)
        
        return key
    }
}

// MARK: - Data Structures

public enum BiometricStatus {
    case available
    case noHardware
    case notEnrolled
    case lockout
    case unavailable
}

public struct BiometricResult {
    public let success: Bool
    public let error: String?
}

public struct PinVerificationResult {
    public let success: Bool
    public let remainingAttempts: Int
    public let error: String?
}

public struct EncryptedWalletResult {
    public let success: Bool
    public let encryptedData: String
    public let error: String?
}

public struct DecryptedWalletResult {
    public let success: Bool
    public let data: Data
    public let error: String?
}
