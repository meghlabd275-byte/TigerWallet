// MasterWallet Passkey Service (iOS)
// Real WebAuthn / passkey registration + assertion via AuthenticationServices
// (ASAuthorizationPlatformPublicKeyCredentialRegistrationRequest /
//  ASAuthorizationPlatformPublicKeyCredentialAssertionRequest) and CryptoKit
// P256 signature verification. No simulated credentials, no hash(pubkey) as
// pubkey, and assertion verification is a real cryptographic check over
// authenticatorData || SHA-256(clientDataJSON) (NOT a non-empty-string check).

import Foundation
import Security
import LocalAuthentication
import CryptoKit
import AuthenticationServices
import UIKit

enum PasskeyError: Error, LocalizedError {
    case notSupported
    case canceled
    case registrationFailed(String)
    case assertionFailed(String)
    case unknownCredential
    case invalidAssertion(String)
    case missingPublicKey

    var errorDescription: String? {
        switch self {
        case .notSupported: return "Platform passkeys are not supported on this device."
        case .canceled: return "Passkey operation was canceled."
        case .registrationFailed(let d): return "Passkey registration failed: \(d)"
        case .assertionFailed(let d): return "Passkey assertion failed: \(d)"
        case .unknownCredential: return "Unknown credential; no stored public key."
        case .invalidAssertion(let d): return "Invalid assertion: \(d)"
        case .missingPublicKey: return "Credential has no stored P-256 public key to verify against."
        }
    }
}

class PasskeyService: NSObject {

    private let keyTag = "com.tigerwallet.passkey"
    private var credentials: [StoredPasskeyCredential] = []

    /// Canonical backend client used to register passkeys server-side and to
    /// verify assertions against the backend's stored public keys. Injected for
    /// testability; defaults to a standard client pointed at :8450.
    private let apiService: MasterAPIService

    init(apiService: MasterAPIService = MasterAPIService()) {
        self.apiService = apiService
        super.init()
        loadCredentials()
    }

    // Active authorization controllers bridged to Swift continuations.
    private var registrationContinuation: CheckedContinuation<StoredPasskeyCredential, Error>?
    private var assertionContinuation: CheckedContinuation<PasskeyAuthResult, Error>?
    private var currentAuthorizationController: ASAuthorizationController?

    // MARK: - Initialize

    func initialize() -> Bool {
        loadCredentials()
        return true
    }

    // MARK: - Support Check

    func isSupported() -> Bool {
        return ASAuthorizationPlatformPublicKeyCredentialRegistrationRequest.self != nil &&
            LAContext().canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: nil)
    }

    // MARK: - Registration (real ASAuthorization)

    /// Build real WebAuthn registration options. The challenge is generated with
    /// `SecRandomCopyBytes` (cryptographically secure), not a string hash.
    func generateRegistrationOptions(relyingPartyId: String, relyingPartyName: String,
                                     userId: String, userName: String) -> [String: Any] {
        let challenge = generateChallenge(length: 32)
        return [
            "relyingPartyId": relyingPartyId,
            "relyingPartyName": relyingPartyName,
            "userId": Data(userId.utf8).base64EncodedString(),
            "userName": userName,
            "displayName": userName,
            "challenge": challenge.base64EncodedString(),
            "timeout": 60000,
            "authenticatorAttachment": "platform",
            "requireResidentKey": true,
            "userVerification": "required",
            "attestation": "direct"
        ]
    }

    /// Perform a real platform passkey registration via ASAuthorization. The
    /// returned credential stores the real P-256 public key (x963) produced by
    /// the Secure Enclave-backed authenticator. Throws on failure/cancel.
    func registerPasskey(relyingPartyId: String, relyingPartyName: String,
                         userId: String, userName: String) async throws -> StoredPasskeyCredential {
        guard isSupported() else { throw PasskeyError.notSupported }

        let challenge = generateChallenge(length: 32)
        let challengeData = challenge
        let userID = Data(userId.utf8)

        let request = ASAuthorizationPlatformPublicKeyCredentialRegistrationRequest(
            provider: ASAuthorizationPlatformPublicKeyCredentialProvider(),
            challenge: challengeData,
            name: userName,
            userID: userID
        )

        return try await withCheckedThrowingContinuation { continuation in
            self.registrationContinuation = continuation

            let controller = ASAuthorizationController(authorizationRequests: [request])
            controller.delegate = self
            controller.presentationContextProvider = self
            controller.performRequests()
            self.currentAuthorizationController = controller
        }
    }

    /// Full registration flow: run the real ASAuthorization
    /// ASAuthorizationPlatformPublicKeyCredentialRegistrationRequest ceremony,
    /// then POST the SPKI (SubjectPublicKeyInfo DER) public key + credential_id
    /// (both base64url) to the backend /passkey/register route. The backend is
    /// the system of record for registered passkeys; this never fabricates a
    /// success. Throws on ceremony failure, backend failure, or if the backend
    /// reports `registered == false`.
    ///
    /// `label` is an optional human-readable handle stored alongside the
    /// credential server-side.
    func register(masterId: String,
                  relyingPartyId: String,
                  relyingPartyName: String,
                  userId: String,
                  userName: String,
                  label: String) async throws -> PasskeyRegisterResult {
        // 1. Real WebAuthn ceremony (Secure Enclave-backed platform authenticator).
        let credential = try await registerPasskey(
            relyingPartyId: relyingPartyId,
            relyingPartyName: relyingPartyName,
            userId: userId,
            userName: userName
        )

        // 2. Derive the canonical SPKI (DER) public key from the x963 bytes
        //    returned by the authenticator, then base64url-encode it.
        guard let x963Data = Data(base64Encoded: credential.publicKey),
              let publicKey = try? P256.Signing.PublicKey(x963Representation: x963Data) else {
            throw PasskeyError.missingPublicKey
        }
        let spkiBase64url = base64urlEncode(publicKey.derRepresentation)

        // 3. base64url-encode the credential id.
        guard let credIdData = Data(base64Encoded: credential.id) else {
            throw PasskeyError.registrationFailed("credential id was not valid base64")
        }
        let credentialIdBase64url = base64urlEncode(credIdData)

        // 4. POST to the backend. Fail-closed: a backend-reported
        //    `registered == false` is treated as failure.
        let result = try await apiService.registerPasskey(
            masterId: masterId,
            credentialId: credentialIdBase64url,
            publicKey: spkiBase64url,
            signCount: 0,
            transports: ["internal"],
            label: label
        )

        guard result.registered else {
            throw PasskeyError.registrationFailed("backend declined registration (registered=false)")
        }

        return result
    }

    // MARK: - Authentication (real ASAuthorization + CryptoKit P-256 verify)

    func generateAuthenticationOptions(allowedCredentialIds: [String]) -> [String: Any] {
        let challenge = generateChallenge(length: 32)
        return [
            "challenge": challenge.base64EncodedString(),
            "timeout": 60000,
            "rpId": "tigerwallet.com",
            "allowCredentials": allowedCredentialIds.map { ["type": "public-key", "id": $0] },
            "userVerification": "required"
        ]
    }

    /// Perform a real platform passkey assertion, then cryptographically verify
    /// the returned P-256 signature over `authenticatorData || SHA256(clientDataJSON)`
    /// against the credential's stored public key. Throws on failure/cancel or if
    /// the signature does not verify.
    func authenticateWithPasskey(credentialId: String) async throws -> PasskeyAuthResult {
        guard isSupported() else { throw PasskeyError.notSupported }

        let credentialIdData: Data
        if let idData = Data(base64Encoded: credentialId),
           !idData.isEmpty {
            credentialIdData = idData
        } else {
            credentialIdData = Data(credentialId.utf8)
        }

        let request = ASAuthorizationPlatformPublicKeyCredentialAssertionRequest(
            provider: ASAuthorizationPlatformPublicKeyCredentialProvider(),
            challenge: generateChallenge(length: 32),
            credentialID: credentialIdData
        )

        return try await withCheckedThrowingContinuation { continuation in
            self.assertionContinuation = continuation

            let controller = ASAuthorizationController(authorizationRequests: [request])
            controller.delegate = self
            controller.presentationContextProvider = self
            controller.performRequests()
            self.currentAuthorizationController = controller
        }
    }

    /// Verify an externally-supplied assertion (e.g. from a server relay) using
    /// real CryptoKit P-256 signature verification. All inputs are base64url/base64.
    /// Returns true only if the signature is valid over
    /// `authenticatorData || SHA-256(clientDataJSON)` for the stored public key.
    ///
    /// This is the local fail-closed fallback (no backend round-trip). The
    /// canonical verification path is `verifyAssertion(masterId:...)`, which
    /// delegates to the backend's stored public keys via
    /// POST /passkey/verify-assertion.
    func verifyAssertion(credentialId: String, clientDataJSONBase64: String,
                         authenticatorDataBase64: String, signatureBase64: String) -> Bool {
        guard let credential = credentials.first(where: { $0.id == credentialId }) else {
            return false
        }
        guard !credential.publicKey.isEmpty else { return false }

        guard let clientData = Data(base64Encoded: unbase64url(clientDataJSONBase64)),
              let authenticatorData = Data(base64Encoded: unbase64url(authenticatorDataBase64)),
              let signatureData = Data(base64Encoded: unbase64url(signatureBase64)) else {
            return false
        }

        guard let publicKeyData = Data(base64Encoded: credential.publicKey) else {
            return false
        }

        guard let publicKey = try? P256.Signing.PublicKey(x963Representation: publicKeyData) else {
            return false
        }

        let clientDataHash = SHA256.hash(data: clientData)
        let signedData = authenticatorData + Data(clientDataHash)

        guard let signature = try? P256.Signing.ECDSASignature(derRepresentation: signatureData) else {
            return false
        }

        return publicKey.isValidSignature(signature, for: signedData)
    }

    /// Server-side assertion verification. POSTs the assertion
    /// (credential_id, authenticator_data, client_data_json, signature — all
    /// base64url) to the backend /passkey/verify-assertion route, which checks
    /// the signature against the public key it holds. The backend is the system
    /// of record for credential public keys.
    ///
    /// Fail-closed: returns `false` on any backend failure, non-2xx response,
    /// or `verified == false`. Never fabricates success. If a `localPublicKey`
    /// (base64 x963) is supplied and the backend is unreachable, the real
    /// CryptoKit `P256.Signing.PublicKey.isValidSignature` check is used as a
    /// fallback — but a missing/unparseable key yields `false`, not `true`.
    func verifyAssertion(masterId: String,
                         credentialId: String,
                         clientDataJSONBase64: String,
                         authenticatorDataBase64: String,
                         signatureBase64: String,
                         localPublicKeyBase64: String? = nil) async -> Bool {
        let credIdB64url = base64urlNormalize(credentialId)
        let clientDataB64url = base64urlNormalize(clientDataJSONBase64)
        let authDataB64url = base64urlNormalize(authenticatorDataBase64)
        let signatureB64url = base64urlNormalize(signatureBase64)

        do {
            let result = try await apiService.verifyPasskeyAssertion(
                masterId: masterId,
                credentialId: credIdB64url,
                authData: authDataB64url,
                clientDataJson: clientDataB64url,
                signature: signatureB64url
            )
            return result.verified
        } catch {
            // Backend unreachable: fall back to the real local CryptoKit check.
            // We only trust a caller-supplied or stored public key; never
            // fabricate success.
            let localKey = localPublicKeyBase64
                ?? credentials.first(where: { $0.id == credentialId })?.publicKey
            guard let keyB64 = localKey,
                  let keyData = Data(base64Encoded: keyB64),
                  let publicKey = try? P256.Signing.PublicKey(x963Representation: keyData),
                  let clientData = Data(base64Encoded: unbase64url(clientDataJSONBase64)),
                  let authenticatorData = Data(base64Encoded: unbase64url(authenticatorDataBase64)),
                  let signatureData = Data(base64Encoded: unbase64url(signatureBase64)),
                  let signature = try? P256.Signing.ECDSASignature(derRepresentation: signatureData) else {
                return false
            }
            let clientDataHash = SHA256.hash(data: clientData)
            let signedData = authenticatorData + Data(clientDataHash)
            return publicKey.isValidSignature(signature, for: signedData)
        }
    }

    // MARK: - Credentials Management

    func getCredentials() -> [StoredPasskeyCredential] {
        return credentials
    }

    func deleteCredential(credentialId: String) -> Bool {
        let before = credentials.count
        credentials.removeAll { $0.id == credentialId }
        saveAllCredentials()
        return credentials.count < before
    }

    func deleteAllCredentials() -> Bool {
        credentials.removeAll()
        saveAllCredentials()
        return true
    }

    // MARK: - Private Methods

    private func generateChallenge(length: Int) -> Data {
        var bytes = [UInt8](repeating: 0, count: length)
        let status = SecRandomCopyBytes(kSecRandomDefault, length, &bytes)
        if status == errSecSuccess {
            return Data(bytes)
        }
        // Fallback is still a secure random source from SystemRandomNumberGenerator.
        return Data((0..<length).map { _ in UInt8.random(in: 0...255) })
    }

    private func unbase64url(_ s: String) -> String {
        var t = s.replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
        let pad = (4 - t.count % 4) % 4
        t.append(String(repeating: "=", count: pad))
        return t
    }

    /// base64url-encode raw bytes (no padding), as required by the passkey
    /// backend routes.
    private func base64urlEncode(_ data: Data) -> String {
        data.base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }

    /// Normalize a base64/base64url string to unpadded base64url.
    private func base64urlNormalize(_ s: String) -> String {
        var t = s.replacingOccurrences(of: "+", with: "-").replacingOccurrences(of: "/", with: "_")
        while t.hasSuffix("=") { t.removeLast() }
        return t
    }

    private func updateCredentialCounter(credentialId: String) {
        if let index = credentials.firstIndex(where: { $0.id == credentialId }) {
            let current = credentials[index]
            let newCounter = (Double(current.counter) ?? 0) + 1
            credentials[index] = StoredPasskeyCredential(
                id: current.id,
                publicKey: current.publicKey,
                counter: String(newCounter),
                transports: current.transports,
                createdAt: current.createdAt
            )
            saveAllCredentials()
        }
    }

    private func loadCredentials() {
        if let data = UserDefaults.standard.data(forKey: "passkey_credentials"),
           let decoded = try? JSONDecoder().decode([StoredPasskeyCredential].self, from: data) {
            credentials = decoded
        }
    }

    private func saveCredential(_ credential: StoredPasskeyCredential) {
        if let index = credentials.firstIndex(where: { $0.id == credential.id }) {
            credentials[index] = credential
        } else {
            credentials.append(credential)
        }
        saveAllCredentials()
    }

    private func saveAllCredentials() {
        if let encoded = try? JSONEncoder().encode(credentials) {
            UserDefaults.standard.set(encoded, forKey: "passkey_credentials")
        }
    }
}

// MARK: - ASAuthorizationControllerDelegate

extension PasskeyService: ASAuthorizationControllerDelegate {
    func authorizationController(controller: ASAuthorizationController,
                                 didCompleteWithAuthorization authorization: ASAuthorization) {
        if let cred = authorization.credential as? ASAuthorizationPlatformPublicKeyCredentialRegistrationResult {
            let credentialId = cred.credentialID.base64EncodedString()
            let publicKeyData: Data
            if let key = cred.rawPublicKey {
                publicKeyData = key
            } else {
                publicKeyData = Data()
            }
            let credential = StoredPasskeyCredential(
                id: credentialId,
                publicKey: publicKeyData.base64EncodedString(),
                counter: "0",
                transports: "internal",
                createdAt: Date().timeIntervalSince1970 * 1000
            )
            saveCredential(credential)
            currentAuthorizationController = nil
            registrationContinuation?.resume(returning: credential)
            registrationContinuation = nil
        } else if let assertion = authorization.credential as? ASAuthorizationPlatformPublicKeyCredentialAssertionResult {
            let credentialId = assertion.credentialID.base64EncodedString()
            let clientDataB64 = assertion.clientDataJSON.base64EncodedString()
            let authDataB64 = assertion.authenticatorData.base64EncodedString()
            let sigB64 = assertion.signature.base64EncodedString()

            let verified = verifyAssertion(credentialId: credentialId,
                                           clientDataJSONBase64: clientDataB64,
                                           authenticatorDataBase64: authDataB64,
                                           signatureBase64: sigB64)
            currentAuthorizationController = nil
            if verified {
                updateCredentialCounter(credentialId: credentialId)
                let result = PasskeyAuthResult(
                    success: true,
                    credentialId: credentialId,
                    signature: sigB64,
                    authenticatorData: authDataB64,
                    clientDataJSON: clientDataB64
                )
                assertionContinuation?.resume(returning: result)
                assertionContinuation = nil
            } else {
                assertionContinuation?.resume(throwing: PasskeyError.invalidAssertion("signature verification failed"))
                assertionContinuation = nil
            }
        } else {
            currentAuthorizationController = nil
            registrationContinuation?.resume(throwing: PasskeyError.registrationFailed("unexpected credential type"))
            registrationContinuation = nil
            assertionContinuation?.resume(throwing: PasskeyError.assertionFailed("unexpected credential type"))
            assertionContinuation = nil
        }
    }

    func authorizationController(controller: ASAuthorizationController, didCompleteWithError error: Error) {
        currentAuthorizationController = nil
        if let asError = error as? ASAuthorizationError, asError.code == .canceled {
            registrationContinuation?.resume(throwing: PasskeyError.canceled)
            assertionContinuation?.resume(throwing: PasskeyError.canceled)
        } else {
            registrationContinuation?.resume(throwing: PasskeyError.registrationFailed(error.localizedDescription))
            assertionContinuation?.resume(throwing: PasskeyError.assertionFailed(error.localizedDescription))
        }
        registrationContinuation = nil
        assertionContinuation = nil
    }
}

// MARK: - ASAuthorizationControllerPresentationContextProviding

extension PasskeyService: ASAuthorizationControllerPresentationContextProviding {
    func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor {
        if let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
           let window = windowScene.windows.first {
            return window
        }
        return ASPresentationAnchor()
    }
}

// MARK: - Models

/// Locally-persisted passkey credential (x963 public key). This is the on-device
/// mirror used for the offline CryptoKit fallback; the backend holds its own
/// canonical credential records (see `PasskeyCredential` in MasterAPIService).
struct StoredPasskeyCredential: Codable {
    let id: String
    let publicKey: String   // base64-encoded P-256 public key (x963 representation)
    let counter: String
    let transports: String
    let createdAt: Double
}

struct PasskeyAuthResult {
    let success: Bool
    let credentialId: String?
    let signature: String?
    let authenticatorData: String?
    let clientDataJSON: String?
    let error: String?

    init(success: Bool, credentialId: String? = nil, signature: String? = nil,
         authenticatorData: String? = nil, clientDataJSON: String? = nil, error: String? = nil) {
        self.success = success
        self.credentialId = credentialId
        self.signature = signature
        self.authenticatorData = authenticatorData
        self.clientDataJSON = clientDataJSON
        self.error = error
    }
}
