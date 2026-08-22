import Foundation
import Security

// KeychainService — device-bound storage for the no-registration transparent
// session (mirrors web OnboardingContext's localStorage-backed SessionBlob +
// WALLET_IDS_KEY).
//
// The user NEVER sees a login form. On first launch the app auto-provisions a
// random device-bound identity (UUID-style email + CSPRNG password) stored in
// Keychain, registers it transparently, and logs in to obtain a JWT. The JWT
// (held by UserWalletApiService in UserDefaults) authenticates every backend
// call. `onboarded` = at least one wallet id stored in Keychain.

enum KeychainService {
    private static let service = "com.tigerwallet.userwallet"
    private static let sessionAccount = "transparent-session"
    private static let walletIdsAccount = "wallet-ids"

    // MARK: - Transparent identity (CSPRNG, mirrors web randomIdentity)

    struct Identity {
        let email: String      // <16-hex>@device.local
        let password: String   // 32-hex (128 bits) ephemeral account password
    }

    /// 32 random bytes via SecRandomCopyBytes → hex. Email uses the first 16
    /// hex chars; the full 32-char hex is the ephemeral account password.
    static func randomIdentity() -> Identity {
        var bytes = [UInt8](repeating: 0, count: 32)
        let status = bytes.withUnsafeMutableBufferPointer { ptr -> OSStatus in
            SecRandomCopyBytes(kSecRandomDefault, ptr.count, ptr.baseAddress!)
        }
        guard status == errSecSuccess else {
            // Fall back to Foundation.random — still 128 bits, but not the
            // security-focused path. Surfaces a real failure only if both
            // CSPRNG and fallback exhaust, which does not happen in practice.
            bytes = (0..<32).map { _ in UInt8.random(in: 0...255) }
        }
        let hex = bytes.map { String(format: "%02x", $0) }.joined()
        let email = String(hex.prefix(16)) + "@device.local"
        return Identity(email: email, password: hex)
    }

    // MARK: - Session blob

    struct Session: Codable {
        let email: String
        let password: String
        let token: String
        let userId: String
    }

    static func loadSession() -> Session? {
        guard let data = read(account: sessionAccount) else { return nil }
        return try? JSONDecoder().decode(Session.self, from: data)
    }

    static func saveSession(_ s: Session) {
        guard let data = try? JSONEncoder().encode(s) else { return }
        write(account: sessionAccount, data: data)
    }

    static func deleteSession() {
        delete(account: sessionAccount)
    }

    // MARK: - Wallet ids (onboarded gate)

    static func loadWalletIds() -> [String] {
        guard let data = read(account: walletIdsAccount) else { return [] }
        return (try? JSONDecoder().decode([String].self, from: data)) ?? []
    }

    static func saveWalletIds(_ ids: [String]) {
        guard let data = try? JSONEncoder().encode(ids) else { return }
        write(account: walletIdsAccount, data: data)
    }

    static func appendWalletId(_ id: String) {
        var ids = loadWalletIds()
        guard !ids.contains(id) else { return }
        ids.append(id)
        saveWalletIds(ids)
    }

    // MARK: - Core Keychain CRUD

    @discardableResult
    private static func write(account: String, data: Data) -> Bool {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(query as CFDictionary)
        var attrs = query
        attrs[kSecValueData as String] = data
        attrs[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlock
        let status = SecItemAdd(attrs as CFDictionary, nil)
        return status == errSecSuccess
    }

    private static func read(account: String) -> Data? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var item: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        guard status == errSecSuccess, let data = item as? Data else { return nil }
        return data
    }

    @discardableResult
    private static func delete(account: String) -> Bool {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        return SecItemDelete(query as CFDictionary) == errSecSuccess
    }
}
