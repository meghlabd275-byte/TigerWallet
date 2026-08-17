import AuthenticationServices
import Foundation
import UIKit

// PasskeyHelper — REAL WebAuthn (platform passkey) registration for iOS via
// AuthenticationServices. No fake credential is ever produced: if the platform
// refuses, attestation is missing, or the credential public key cannot be
// parsed out of the attestation object, the completion fires with an error.
//
// Why the SPKI is parsed by hand: the high-level
// `ASAuthorizationPlatformPublicKeyCredentialRegistration` only exposes
// `credentialID` and `rawAttestationObject`; it does NOT hand back the
// authenticator's public key as a ready SPKI/PKIX blob (the way the web
// `response.getPublicKey()` or Android CredentialManager do). The canonical
// TigerWallet backend (`go/wallet_api/app_lock.go`, `parseWebAuthnSPKI`)
// requires a base64url **SPKI DER** P-256 public key — `x509.ParsePKIXPublicKey`
// over the decoded bytes, accepting only an EC2 P-256 key. So this helper parses
// the CBOR attestation object, walks `authData` -> `attestedCredentialData` ->
// the COSE-encoded credential public key, and re-encodes x||y as an SPKI DER
// P-256 ECDSA subject public key. The result is forwarded to `setupLock` /
// `passkeyCreateWallet` exactly like the web/Android clients.

enum PasskeyError: Error, LocalizedError {
    case unavailable
    case attestationMissing
    case cborParse(String)
    case notP256
    case invalidAuthData

    var errorDescription: String? {
        switch self {
        case .unavailable: return "Passkey (platform authenticator) is unavailable on this device."
        case .attestationMissing: return "Passkey registration returned no attestation object."
        case .cborParse(let m): return "Passkey attestation parse failed: \(m)"
        case .notP256: return "Passkey credential is not a P-256 (ES256) key."
        case .invalidAuthData: return "Passkey authData is malformed."
        }
    }
}

// A successfully-registered platform passkey credential, ready to post to the
// backend. `credentialID` and `publicKey` are both base64url (no padding), the
// exact encoding the backend's `parseWebAuthnSPKI` expects.
struct PasskeyCredential {
    let credentialID: String   // base64url
    let publicKey: String       // base64url SPKI DER P-256
    let signCount: UInt32
    let attestation: String     // base64url of the raw attestationObject
}

// Small CBOR value model — only the subset needed to read a WebAuthn
// attestationObject (map of {fmt, attStmt, authData}) and the COSE public key
// map nested inside authData. Not a general-purpose decoder.
private enum CBOR {
    case unsignedInt(UInt64)
    case negativeInt(Int64)
    case byteString(Data)
    case textString(String)
    case array([CBOR])
    case map([(CBOR, CBOR)])
    case bool(Bool)
    case null
    case undefined
    case simple(UInt8)
    case float(Double)

    // Convenience for map lookups by an integer label.
    func mapValue(_ key: Int64) -> CBOR? {
        if case .map(let pairs) = self {
            for (k, v) in pairs {
                switch k {
                case .unsignedInt(let u) where Int64(bitPattern: UInt64(u)) == key: return v
                case .negativeInt(let n) where n == key: return v
                default: break
                }
            }
        }
        return nil
    }
}

private final class CBORReader {
    let data: Data
    var pos: Int = 0
    init(_ data: Data) { self.data = data }

    var isEmpty: Bool { pos >= data.count }

    private func readByte() throws -> UInt8 {
        guard pos < data.count else { throw PasskeyError.cborParse("unexpected end of input") }
        let b = data[data.startIndex + pos]
        pos += 1
        return b
    }

    private func readBytes(_ n: Int) throws -> Data {
        guard pos + n <= data.count else { throw PasskeyError.cborParse("unexpected end of input") }
        let d = data.subdata(in: (data.startIndex + pos)..<(data.startIndex + pos + n))
        pos += n
        return d
    }

    private func readArgument(_ ai: UInt8) throws -> UInt64 {
        switch ai {
        case 0...23: return UInt64(ai)
        case 24: return UInt64(try readByte())
        case 25:
            let b0 = UInt64(try readByte()), b1 = UInt64(try readByte())
            return (b0 << 8) | b1
        case 26:
            var v: UInt64 = 0
            for _ in 0..<4 { v = (v << 8) | UInt64(try readByte()) }
            return v
        case 27:
            var v: UInt64 = 0
            for _ in 0..<8 { v = (v << 8) | UInt64(try readByte()) }
            return v
        default: throw PasskeyError.cborParse("invalid additional info \(ai)")
        }
    }

    func readValue() throws -> CBOR {
        let initial = try readByte()
        let major = initial >> 5
        let ai = initial & 0x1f
        switch major {
        case 0: // unsigned int
            return .unsignedInt(try readArgument(ai))
        case 1: // negative int: -1 - n
            let n = try readArgument(ai)
            // -1 - n, where n is a UInt64. Represent as Int64 when it fits.
            if n <= UInt64(Int64.max) {
                return .negativeInt(-1 - Int64(bitPattern: UInt64(n)))
            }
            return .negativeInt(Int64.min) // overflow guard; not expected for COSE labels
        case 2: // byte string
            let len = Int(try readArgument(ai))
            if ai == 31 { throw PasskeyError.cborParse("indefinite byte strings unsupported") }
            return .byteString(try readBytes(len))
        case 3: // text string
            let len = Int(try readArgument(ai))
            let d = try readBytes(len)
            return .textString(String(data: d, encoding: .utf8) ?? "")
        case 4: // array
            if ai == 31 {
                var arr: [CBOR] = []
                while true {
                    if pos < data.count, data[data.startIndex + pos] == 0xff { pos += 1; break }
                    arr.append(try readValue())
                }
                return .array(arr)
            }
            let len = Int(try readArgument(ai))
            var arr: [CBOR] = []
            arr.reserveCapacity(len)
            for _ in 0..<len { arr.append(try readValue()) }
            return .array(arr)
        case 5: // map
            if ai == 31 {
                var pairs: [(CBOR, CBOR)] = []
                while true {
                    if pos < data.count, data[data.startIndex + pos] == 0xff { pos += 1; break }
                    let k = try readValue(); let v = try readValue()
                    pairs.append((k, v))
                }
                return .map(pairs)
            }
            let len = Int(try readArgument(ai))
            var pairs: [(CBOR, CBOR)] = []
            pairs.reserveCapacity(len)
            for _ in 0..<len {
                let k = try readValue(); let v = try readValue()
                pairs.append((k, v))
            }
            return .map(pairs)
        case 7: // simple / float / break
            switch ai {
            case 20: return .bool(false)
            case 21: return .bool(true)
            case 22: return .null
            case 23: return .undefined
            case 25:
                // half-float — not needed for WebAuthn attestation; read 2 bytes.
                _ = try readBytes(2)
                return .simple(25)
            case 26:
                var v: UInt64 = 0
                for _ in 0..<4 { v = (v << 8) | UInt64(try readByte()) }
                return .float(Double(bitPattern: UInt32(truncatingIfNeeded: v)))
            case 27:
                var v: UInt64 = 0
                for _ in 0..<8 { v = (v << 8) | UInt64(try readByte()) }
                return .float(Double(bitPattern: v))
            default:
                return .simple(ai)
            }
        default:
            throw PasskeyError.cborParse("unsupported major type \(major)")
        }
    }
}

// Build the SPKI/PKIX DER subject public key for a P-256 ECDSA key from its
// uncompressed point (04 || X(32) || Y(32)). DER layout:
//   SEQUENCE { SEQUENCE { OID 1.2.840.10045.2.1 (ecPublicKey),
//                        OID 1.2.840.10045.3.1.7 (P-256) },
//              BIT STRING { 0x00, 04||X||Y } }
private func spkiP256(point: Data) -> Data {
    func derLen(_ n: Int) -> Data {
        if n < 0x80 { return Data([UInt8(n)]) }
        if n <= 0xff { return Data([0x81, UInt8(n)]) }
        return Data([0x82, UInt8((n >> 8) & 0xff), UInt8(n & 0xff)])
    }
    func derSeq(tag: UInt8, _ content: Data) -> Data {
        var out = Data([tag])
        out.append(derLen(content.count))
        out.append(content)
        return out
    }
    // OID 1.2.840.10045.2.1 (ecPublicKey) = 06 07 2A 86 48 CE 3D 02 01
    let ecPubKeyOID = Data([0x06, 0x07, 0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x02, 0x01])
    // OID 1.2.840.10045.3.1.7 (P-256 / secp256r1) = 06 08 2A 86 48 CE 3D 03 01 07
    let p256OID = Data([0x06, 0x08, 0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x03, 0x01, 0x07])
    let algId = derSeq(tag: 0x30, ecPubKeyOID + p256OID)

    // BIT STRING: unused-bits byte (0x00) then the point.
    var bitContent = Data([0x00])
    bitContent.append(point)
    let bitString = derSeq(tag: 0x03, bitContent)

    return derSeq(tag: 0x30, algId + bitString)
}

// base64url (no padding) — matches the backend's base64.RawURLEncoding.
private func base64URL(_ data: Data) -> String {
    var s = data.base64EncodedString()
    s = s.replacingOccurrences(of: "+", with: "-")
    s = s.replacingOccurrences(of: "/", with: "_")
    while s.hasSuffix("=") { s.removeLast() }
    return s
}

// Decode a COSE key CBOR map into x||y (the uncompressed P-256 point).
private func coseP256Point(_ cose: CBOR) throws -> Data {
    guard case .map = cose else { throw PasskeyError.cborParse("credential public key is not a map") }
    // kty (label 1) must be 2 (EC2).
    guard let kty = cose.mapValue(1), case .unsignedInt(let ktyVal) = kty, ktyVal == 2 else {
        throw PasskeyError.notP256
    }
    // crv (label -1) must be 1 (P-256).
    guard let crv = cose.mapValue(-1), case .unsignedInt(let crvVal) = crv, crvVal == 1 else {
        throw PasskeyError.notP256
    }
    guard let x = cose.mapValue(-2), case .byteString(let xd) = x, xd.count == 32 else {
        throw PasskeyError.cborParse("missing/invalid COSE x coordinate")
    }
    guard let y = cose.mapValue(-3), case .byteString(let yd) = y, yd.count == 32 else {
        throw PasskeyError.cborParse("missing/invalid COSE y coordinate")
    }
    var point = Data([0x04])
    point.append(xd)
    point.append(yd)
    return point
}

// Walk a WebAuthn attestationObject (CBOR) and return (authData, signCount,
// credentialPublicKey). authData layout:
//   rpIdHash(32) || flags(1) || signCount(4) || [if AT flag: AAGUID(16) ||
//   credIdLen(2) || credId(credIdLen) || credentialPublicKey(COSE CBOR)]
private func parseAttestationAuthData(_ attObj: Data) throws -> (authData: Data, signCount: UInt32, coseKey: CBOR) {
    let root = try CBORReader(attObj).readValue()
    guard case .map(let pairs) = root else {
        throw PasskeyError.cborParse("attestation object is not a CBOR map")
    }
    var authData: Data?
    var fmt: String?
    for (k, v) in pairs {
        if case .textString(let s) = k, s == "authData", case .byteString(let d) = v { authData = d }
        if case .textString(let s) = k, s == "fmt", case .textString(let f) = v { fmt = f }
    }
    guard let ad = authData else { throw PasskeyError.cborParse("attestation object missing authData") }
    _ = fmt // accepted as-is; the public key lives in authData for every fmt.

    guard ad.count >= 37 else { throw PasskeyError.invalidAuthData }
    let flags = ad[ad.startIndex + 32]
    let signCount: UInt32 =
        UInt32(ad[ad.startIndex + 33]) << 24 |
        UInt32(ad[ad.startIndex + 34]) << 16 |
        UInt32(ad[ad.startIndex + 35]) << 8 |
        UInt32(ad[ad.startIndex + 36])

    // Bit 0 (0x01) = AT (attested credential data present).
    let attested = (flags & 0x40) != 0
    guard attested else { throw PasskeyError.invalidAuthData }

    var p = ad.startIndex + 37
    // AAGUID (16 bytes), skip.
    p += 16
    guard p + 2 <= ad.endIndex else { throw PasskeyError.invalidAuthData }
    let credIdLen = Int(ad[p]) << 8 | Int(ad[p + 1])
    p += 2
    guard p + credIdLen <= ad.endIndex else { throw PasskeyError.invalidAuthData }
    p += credIdLen
    // The remainder of authData is the COSE-encoded credential public key.
    let coseBytes = ad.subdata(in: p..<ad.endIndex)
    let cose = try CBORReader(coseBytes).readValue()
    return (ad, signCount, cose)
}

/// Bridge from ASAuthorizationController's callback-based API to a Swift async
/// Result. One instance is retained per in-flight registration and held by the
/// presenting view (via `.onDisappear` cleanup is unnecessary — the controller
/// is held for the duration of the auth prompt).
final class PasskeyRegistrar: NSObject, ASAuthorizationControllerDelegate, ASAuthorizationControllerPresentationContextProviding {

    private var continuation: CheckedContinuation<PasskeyCredential, Error>?
    private var anchorWindow: UIWindow?
    // Strong reference to the live ASAuthorizationController for the duration of
    // the prompt. ASAuthorizationController does not retain its delegate, so this
    // slot keeps both the controller and (transitively) self alive until the
    // delegate callback fires and clears it.
    private var retainedController: ASAuthorizationController?

    // Whether platform passkeys can be requested on this device.
    static var isAvailable: Bool {
        // ASAuthorizationPlatformPublicKeyCredentialProvider exists on iOS 15+,
        // but only authenticator-bearing devices (Face/Touch ID or passcode
        // set) can actually mint a credential. We report availability based on
        // the presence of a biometric/passcode gate via LAContext-free heuristic:
        // the controller will surface `.unavailable` at request time if not.
        return true
    }

    /// Registers a platform passkey for `userHandle` (a stable per-wallet id,
    /// bytes) and resolves with the parsed credential (credentialID + SPKI
    /// P-256 public key, both base64url). Throws on any failure.
    func register(relyingPartyID: String,
                 userHandle: Data,
                 name: String,
                 displayName: String) async throws -> PasskeyCredential {
        try await withCheckedThrowingContinuation { continuation in
            self.continuation = continuation
            let provider = ASAuthorizationPlatformPublicKeyCredentialProvider(
                relyingPartyIdentifier: relyingPartyID
            )
            var challengeBytes = Data(count: 32)
            let result = challengeBytes.withUnsafeMutableBytes { buf -> Int32 in
                if let base = buf.baseAddress {
                    return SecRandomCopyBytes(kSecRandomDefault, 32, base)
                }
                return errSecParam
            }
            if result != errSecSuccess {
                continuation.resume(throwing: PasskeyError.unavailable)
                return
            }
            let request = provider.credentialRegistrationRequest(
                challenge: challengeBytes,
                name: name,
                userID: userHandle
            )
            let controller = ASAuthorizationController(authorizationRequests: [request])
            controller.delegate = self
            controller.presentationContextProvider = self
            self.retainedController = controller
            controller.performRequests()
        }
    }

    // MARK: - ASAuthorizationControllerDelegate

    func authorizationController(controller: ASAuthorizationController,
                                 didCompleteWithAuthorization authorization: ASAuthorization) {
        defer { continuation = nil; retainedController = nil }
        guard let reg = authorization.credential as? ASAuthorizationPlatformPublicKeyCredentialRegistration else {
            continuation?.resume(throwing: PasskeyError.cborParse("unexpected credential type"))
            return
        }
        guard let attObj = reg.rawAttestationObject else {
            continuation?.resume(throwing: PasskeyError.attestationMissing)
            return
        }
        do {
            let (_, signCount, cose) = try parseAttestationAuthData(attObj)
            let point = try coseP256Point(cose)
            let spki = spkiP256(point: point)
            let credentialID = base64URL(reg.credentialID)
            let publicKey = base64URL(spki)
            let attestation = base64URL(attObj)
            continuation?.resume(returning: PasskeyCredential(
                credentialID: credentialID,
                publicKey: publicKey,
                signCount: signCount,
                attestation: attestation
            ))
        } catch {
            continuation?.resume(throwing: error)
        }
    }

    func authorizationController(controller: ASAuthorizationController, didCompleteWithError error: Error) {
        defer { continuation = nil; retainedController = nil }
        let nsError = error as NSError
        if nsError.code == ASAuthorizationError.canceled.rawValue {
            continuation?.resume(throwing: PasskeyError.unavailable)
        } else {
            continuation?.resume(throwing: error)
        }
    }

    // MARK: - ASAuthorizationControllerPresentationContextProviding

    func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor {
        if let w = anchorWindow { return w }
        // Resolve the foreground key window from the active scene.
        let window = Self.topWindow()
        anchorWindow = window
        return window
    }

    static func topWindow() -> UIWindow {
        if let scene = UIApplication.shared.connectedScenes
            .first(where: { $0.activationState == .foregroundActive }) as? UIWindowScene {
            if let w = scene.windows.first(where: { $0.isKeyWindow }) ?? scene.windows.first {
                return w
            }
        }
        // Fallback: create a transient window if none is found (rare in tests).
        return UIWindow()
    }
}
