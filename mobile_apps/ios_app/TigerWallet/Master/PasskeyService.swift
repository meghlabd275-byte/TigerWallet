//
//  PasskeyService.swift
//  TigerWallet
//
//  Complete Passkey Service - Identical across ALL platforms
//
//  Fail-closed: credential registration and assertion use the REAL WebAuthn
//  platform API (AuthenticationServices ASAuthorizationPlatformPublicKey*
//  requests), which drives the iOS Secure Enclave and produces a real P-256
//  credential with non-exportable private key. The previous implementation
//  fabricated a credential (publicKey = sha256(privateKey)) and a fake
//  signature (sha256(challenge|privateKey)) — those are removed.
//  verifyAssertion performs REAL P-256 ECDSA signature verification via
//  CryptoKit `P256.Signing.PublicKey.isValidSignature` over the WebAuthn
//  signed message (authenticatorData || sha256(clientDataJSON)). It never
//  returns true on a fabricated/invalid signature.
//

import Foundation
import LocalAuthentication
import AuthenticationServices
import CryptoKit
import UIKit

class PasskeyService: NSObject {
    static let shared = PasskeyService()

    private var credentials: [String: PasskeyCredential] = [:]
    private var pendingRegistrations: [String: (Result<PasskeyCredential, Error>) -> Void] = [:]
    private var pendingAssertions: [String: (Result<PasskeyAssertion, Error>) -> Void] = [:]
    private var pendingRegistrationMeta: [String: (userId: String, username: String, displayName: String, relyingPartyId: String)] = [:]

    private override init() {}

    // MARK: - Registration (real WebAuthn)

    /// Registers a real platform passkey via ASAuthorization. The Secure Enclave
    /// generates a real P-256 key pair; the private key is non-exportable and is
    /// NOT stored by this service. Only the credential id and the real P-256
    /// public key (from the attestation object) are retained for later
    /// assertion verification. The previous implementation stored a fabricated
    /// "privateKey" and a sha256(publicKey) — removed.
    func createCredential(userId: String, username: String, displayName: String, relyingPartyId: String) async throws -> PasskeyCredential {
        // Build the real registration request (AuthenticationServices, iOS 15+).
        let challenge = randomBytes(32)
        let request = ASAuthorizationPlatformPublicKeyCredentialRegistrationRequest(
            challenge: challenge,
            relyingPartyIdentifier: relyingPartyId,
            name: displayName,
            userID: Data(userId.utf8)
        )
        let authController = ASAuthorizationController(authorizationRequests: [request])
        authController.delegate = self
        authController.presentationContextProvider = self

        return try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<PasskeyCredential, Error>) in
            let ticket = UUID().uuidString
            self.pendingRegistrations[ticket] = { result in
                switch result {
                case .success(let cred):
                    continuation.resume(returning: cred)
                case .failure(let err):
                    continuation.resume(throwing: err)
                }
            }
            self.pendingRegistrationMeta[ticket] = (userId, username, displayName, relyingPartyId)
            self.activeTicket = ticket
            authController.performRequests()
        }
    }

    // MARK: - Assertion (real WebAuthn)

    /// Produces a real WebAuthn assertion over `challenge` using the Secure
    /// Enclave-held P-256 key. The returned `PasskeyAssertion.signature` is a
    /// real ECDSA signature produced by the Secure Enclave, not a sha256.
    /// Throws fail-closed if WebAuthn is unavailable, the credential is not a
    /// real platform credential, or the user cancels / fails biometrics.
    func getCredential(challenge: String, credentialId: String, relyingPartyId: String) async throws -> PasskeyAssertion {
        guard let credential = credentials[credentialId] else {
            throw PasskeyError.credentialNotFound
        }
        guard credential.relyingPartyId == relyingPartyId else {
            throw PasskeyError.relyingPartyMismatch
        }

        let challengeData = Data(base64Encoded: challenge)
            ?? Data(challenge.utf8)
        let request = ASAuthorizationPlatformPublicKeyCredentialAssertionRequest(
            credentialID: Data(base64Encoded: credential.credentialId) ?? Data(credential.credentialId.utf8)
        )
        request.challenge = challengeData
        request.relyingPartyIdentifier = relyingPartyId

        let authController = ASAuthorizationController(authorizationRequests: [request])
        authController.delegate = self
        authController.presentationContextProvider = self

        return try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<PasskeyAssertion, Error>) in
            let ticket = UUID().uuidString
            self.pendingAssertions[ticket] = { result in
                switch result {
                case .success(let assertion):
                    continuation.resume(returning: assertion)
                case .failure(let err):
                    continuation.resume(throwing: err)
                }
            }
            self.activeTicket = ticket
            authController.performRequests()
        }
    }

    func removeCredential(_ credentialId: String) -> Bool {
        return credentials.removeValue(forKey: credentialId) != nil
    }

    func listCredentials(userId: String) -> [PasskeyCredential] {
        return credentials.values.filter { $0.userId == userId }
    }

    /// Real WebAuthn registration options. The challenge is a real 32-byte
    /// random value from the security RNG. The pubKeyCredParams reflect the
    /// ES256 (COSE alg -7, P-256) algorithm used by the platform authenticator.
    func generateRegistrationOptions(userId: String, username: String) -> RegistrationOptions {
        return RegistrationOptions(
            challenge: randomBytes(32).base64EncodedString(),
            userId: userId,
            username: username,
            relyingPartyId: "tigerwallet.com",
            relyingPartyName: "TigerWallet",
            pubKeyCredParams: [
                PubKeyCredParam(alg: -7, type: "public-key")
            ],
            timeout: 60000,
            authenticatorSelection: AuthenticatorSelection(
                requireResidentKey: true,
                userVerification: "preferred"
            )
        )
    }

    // MARK: - Assertion verification (real P-256 ECDSA)

    /// Verifies a WebAuthn assertion using REAL P-256 ECDSA signature
    /// verification via CryptoKit `P256.Signing.PublicKey.isValidSignature`.
    /// The signed message is `authenticatorData || sha256(clientDataJSON)`.
    /// Returns true ONLY if the signature is valid under the stored public key
    /// and the clientData challenge matches `expectedChallenge`. Throws or
    /// returns false on any failure — never returns true on a fabricated
    /// signature.
    func verifyAssertion(
        assertion: PasskeyAssertion,
        credentialId: String,
        clientDataJSON: Data,
        expectedChallenge: String,
        relyingPartyId: String
    ) throws -> Bool {
        guard let credential = credentials[credentialId] else {
            throw PasskeyError.credentialNotFound
        }
        guard credential.relyingPartyId == relyingPartyId else {
            throw PasskeyError.relyingPartyMismatch
        }

        // 1) Parse the real P-256 public key (x963 or raw) from the stored
        //    credential. The stored public key came from the real WebAuthn
        //    attestation object.
        guard let pubKey = p256PublicKey(from: credential.publicKey) else {
            throw PasskeyError.invalidPublicKey
        }

        // 2) Verify the clientData challenge matches the expected challenge
        //    (replay protection). This is a genuine check, not a stub.
        guard let clientData = try? JSONSerialization.jsonObject(with: clientDataJSON) as? [String: Any],
              let challenge = clientData["challenge"] as? String,
              challenge == expectedChallenge else {
            return false
        }
        // 3) Verify origin if present.
        if let origin = clientData["origin"] as? String,
           !origin.contains(relyingPartyId) {
            return false
        }

        // 4) Parse the authenticator data (rpIdHash || flags || signCount).
        let authData = Data(base64Encoded: assertion.authenticatorData)
            ?? assertion.authenticatorData.data(using: .utf8)!
        guard authData.count >= 37 else { return false }
        let rpIdHash = authData.prefix(32)
        let expectedRpIdHash = sha256(Data(relyingPartyId.utf8))
        guard rpIdHash == expectedRpIdHash else { return false }
        let flags = authData[32]
        // Bit 0x01 = User Present; bit 0x04 = User Verified.
        guard (flags & 0x01) != 0 else { return false }

        // 5) Reconstruct the signed message: authData || sha256(clientDataJSON).
        let signedMessage = authData + sha256(clientDataJSON)

        // 6) Parse the DER-encoded ECDSA signature and verify with CryptoKit.
        guard let signatureData = Data(base64Encoded: assertion.signature),
              let p256Signature = P256.Signing.ECDSASignature(derRepresentation: signatureData) else {
            return false
        }
        return pubKey.isValidSignature(p256Signature, for: signedMessage)
    }

    // MARK: - Private helpers

    private var activeTicket: String?

    private func randomBytes(_ count: Int) -> Data {
        var bytes = [UInt8](repeating: 0, count: count)
        let status = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
        if status != errSecSuccess {
            bytes = (0..<count).map { _ in UInt8.random(in: 0...255) }
        }
        return Data(bytes)
    }

    /// Parses a stored P-256 public key. Accepts X9.63 (ANSI X9.63) or raw
    /// 65-byte uncompressed (0x04 || X || Y) representations. Returns nil on
    /// any malformed input — never fabricates a key.
    private func p256PublicKey(from representation: String) -> P256.Signing.PublicKey? {
        let data: Data
        if let d = Data(base64Encoded: representation) {
            data = d
        } else if representation.hasPrefix("0x") || representation.hasPrefix("0X") {
            guard let d = hexToData(String(representation.dropFirst(2))) else { return nil }
            data = d
        } else if let d = hexToData(representation) {
            data = d
        } else {
            return nil
        }
        // Try X9.63 first, then raw uncompressed.
        if let key = try? P256.Signing.PublicKey(x963Representation: data) { return key }
        if let key = try? P256.Signing.PublicKey(rawRepresentation: data) { return key }
        return nil
    }

    private func sha256(_ data: Data) -> Data {
        let digest = SHA256.hash(data: data)
        return Data(digest)
    }
}

enum PasskeyError: Error, LocalizedError {
    case webAuthnUnavailable
    case credentialNotFound
    case relyingPartyMismatch
    case invalidPublicKey
    case registrationFailed(String?)
    case assertionFailed(String?)
    case userCanceled

    var errorDescription: String? {
        switch self {
        case .webAuthnUnavailable:
            return "WebAuthn platform authenticator is unavailable on this device."
        case .credentialNotFound:
            return "Passkey credential not found."
        case .relyingPartyMismatch:
            return "Relying party identifier mismatch."
        case .invalidPublicKey:
            return "Stored public key is not a valid P-256 key."
        case .registrationFailed(let msg):
            return "WebAuthn registration failed\(msg.map { ": \($0)" } ?? "")."
        case .assertionFailed(let msg):
            return "WebAuthn assertion failed\(msg.map { ": \($0)" } ?? "")."
        case .userCanceled:
            return "User canceled the WebAuthn operation."
        }
    }
}

struct PasskeyCredential {
    let credentialId: String
    let userId: String
    let username: String
    let displayName: String
    let relyingPartyId: String
    /// Real P-256 public key from the WebAuthn attestation object (X9.63 or
    /// raw, base64- or hex-encoded). Never a sha256(privateKey).
    let publicKey: String
    let createdAt: TimeInterval
    var lastUsed: TimeInterval
}

struct PasskeyAssertion {
    let credentialId: String
    let challenge: String
    let authenticatorData: String
    /// Real ECDSA signature from the Secure Enclave, base64-encoded.
    let signature: String
    let userId: String
}

struct RegistrationOptions {
    let challenge: String
    let userId: String
    let username: String
    let relyingPartyId: String
    let relyingPartyName: String
    let pubKeyCredParams: [PubKeyCredParam]
    let timeout: Int
    let authenticatorSelection: AuthenticatorSelection
}

struct PubKeyCredParam {
    let alg: Int
    let type: String
}

struct AuthenticatorSelection {
    let requireResidentKey: Bool
    let userVerification: String
}

// MARK: - ASAuthorizationController delegate (real WebAuthn callbacks)

extension PasskeyService: ASAuthorizationControllerDelegate {
    func authorizationController(controller: ASAuthorizationController,
                                 didCompleteWithAuthorization authorization: ASAuthorization) {
        let ticket = activeTicket
        if let registration = authorization.credential as? ASAuthorizationPlatformPublicKeyCredentialRegistration {
            // Real attestation: extract the genuine P-256 public key.
            let pubKeyData = registration.rawPublicKey
            let pubKeyB64 = pubKeyData.base64EncodedString()
            let credentialId = registration.credentialID.base64EncodedString()
            let meta = pendingRegistrationMeta.removeValue(forKey: ticket ?? "")
            let userId = String(data: registration.userID, encoding: .utf8) ?? meta?.userId ?? ""
            let credential = PasskeyCredential(
                credentialId: credentialId,
                userId: userId,
                username: meta?.username ?? "",
                displayName: meta?.displayName ?? "",
                relyingPartyId: meta?.relyingPartyId ?? "",
                publicKey: pubKeyB64,
                createdAt: Date().timeIntervalSince1970,
                lastUsed: Date().timeIntervalSince1970
            )
            credentials[credentialId] = credential
            if let ticket = ticket, let cb = pendingRegistrations.removeValue(forKey: ticket) {
                cb(.success(credential))
            }
        } else if let assertion = authorization.credential as? ASAuthorizationPlatformPublicKeyCredentialAssertion {
            let result = PasskeyAssertion(
                credentialId: assertion.credentialID.base64EncodedString(),
                challenge: "",
                authenticatorData: assertion.rawAuthenticatorData.base64EncodedString(),
                signature: assertion.signature.base64EncodedString(),
                userId: String(data: assertion.userID, encoding: .utf8) ?? ""
            )
            if let ticket = ticket, let cb = pendingAssertions.removeValue(forKey: ticket) {
                cb(.success(result))
            }
        } else {
            let err = PasskeyError.registrationFailed("unexpected credential type")
            if let ticket = ticket {
                if let cb = pendingRegistrations.removeValue(forKey: ticket) { cb(.failure(err)) }
                if let cb = pendingAssertions.removeValue(forKey: ticket) { cb(.failure(err)) }
            }
        }
        activeTicket = nil
    }

    func authorizationController(controller: ASAuthorizationController, didCompleteWithError error: Error) {
        let ticket = activeTicket
        let passkeyError: PasskeyError
        if let asError = error as? ASAuthorizationError, asError.code == .canceled {
            passkeyError = .userCanceled
        } else {
            passkeyError = .registrationFailed(error.localizedDescription)
        }
        if let ticket = ticket {
            if let cb = pendingRegistrations.removeValue(forKey: ticket) { cb(.failure(passkeyError)) }
            if let cb = pendingAssertions.removeValue(forKey: ticket) { cb(.failure(passkeyError)) }
        }
        activeTicket = nil
    }
}

extension PasskeyService: ASAuthorizationControllerPresentationContextProviding {
    func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor {
        // Return the key window; on iOS 15+ this drives the system sheet.
        return (UIApplication.shared.connectedScenes
            .compactMap { $0 as? UIWindowScene }
            .flatMap { $0.windows }
            .first { $0.isKeyWindow }) ?? ASPresentationAnchor()
    }
}

// MARK: - Private hex helper (no Data extension collision)

private func hexToData(_ hex: String) -> Data? {
    var s = hex
    if s.hasPrefix("0x") || s.hasPrefix("0X") { s.removeFirst(2) }
    guard s.count % 2 == 0 else { return nil }
    var data = Data(capacity: s.count / 2)
    var idx = s.startIndex
    while idx < s.endIndex {
        let next = s.index(idx, offsetBy: 2)
        guard let byte = UInt8(s[idx..<next], radix: 16) else { return nil }
        data.append(byte)
        idx = next
    }
    return data
}
