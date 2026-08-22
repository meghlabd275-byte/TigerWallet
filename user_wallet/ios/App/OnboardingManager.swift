import Foundation
import Combine

// OnboardingManager — the no-registration self-custody entry model (mirrors the
// web OnboardingContext.tsx).
//
// The user NEVER sees a register/login form. On first launch a transparent
// ephemeral account is auto-provisioned (random device-bound identity stored in
// Keychain) so the JWT-backed backend security is preserved. The wallet
// password the user enters encrypts the seed (server-side scrypt + AES-GCM),
// independent of the ephemeral account.
//
//   ensureSession()  — if no local token, auto-register a random identity +
//                      login to obtain a JWT. One-time, invisible to the user.
//   createWallet()   — password -> backend POST /wallets -> mnemonic (backup).
//   importWallet()   — seed + password -> backend POST /wallets { mnemonic }.
//
// `onboarded` (a wallet id exists in Keychain) gates the app:
// false => OnboardingView (Create/Import); true => ContentView (Dashboard).

final class OnboardingManager: ObservableObject {
    static let shared = OnboardingManager()

    @Published private(set) var ready = false
    @Published private(set) var onboarded = false
    @Published private(set) var sessionError: String?

    private var walletIds: [String] = KeychainService.loadWalletIds()

    var localWalletIds: [String] { walletIds }
    var transparentEmail: String? { KeychainService.loadSession()?.email }

    private init() {
        onboarded = walletIds.count > 0
    }

    // MARK: - Session bootstrap (mirrors ensureSession)

    /// Provisions / re-validates the transparent session. One-time on first
    /// launch; invisible to the user. Sets `ready` on completion (success or
    /// failure — the landing page renders and surfaces a retry on error).
    func ensureSession() async {
        if UserWalletApiService.shared.isAuthenticated {
            await MainActor.run { self.ready = true }
            return
        }
        var s = KeychainService.loadSession()
        if s == nil {
            let id = KeychainService.randomIdentity()
            // Register silently; if it fails (collision/network) fall through to
            // login which surfaces the real error.
            _ = try? await UserWalletApiService.shared.register(email: id.email, password: id.password)
            do {
                let res = try await UserWalletApiService.shared.login(email: id.email, password: id.password)
                s = KeychainService.Session(email: id.email, password: id.password, token: res.token, userId: res.user_id ?? res.user?.id ?? "")
                KeychainService.saveSession(s!)
            } catch {
                // Cannot provision a transparent session — surface a real error.
                await MainActor.run {
                    self.sessionError = error.localizedDescription
                    self.ready = true
                }
                return
            }
        } else if let existing = s {
            // Re-validate the stored token; if it no longer authenticates,
            // re-login transparently with the stored ephemeral credentials.
            if !UserWalletApiService.shared.isAuthenticated {
                do {
                    let res = try await UserWalletApiService.shared.login(email: existing.email, password: existing.password)
                    s = KeychainService.Session(email: existing.email, password: existing.password, token: res.token, userId: res.user_id ?? res.user?.id ?? "")
                    KeychainService.saveSession(s!)
                } catch {
                    await MainActor.run {
                        self.sessionError = error.localizedDescription
                        self.ready = true
                    }
                    return
                }
            }
        }
        await MainActor.run { self.ready = true }
    }

    // MARK: - Create / Import (mirror createWallet / importWallet)

    struct CreatedWallet {
        let mnemonic: String
        let id: String
        let address: String
    }

    func createWallet(label: String, password: String, chainId: Int) async throws -> CreatedWallet {
        await ensureSession()
        let w = try await UserWalletApiService.shared.createWallet(label: label, password: password, chainId: chainId)
        guard let mnemonic = w.mnemonic, !mnemonic.isEmpty else {
            throw WalletAPIError.emptyResponse
        }
        return CreatedWallet(mnemonic: mnemonic, id: w.id, address: w.address)
    }

    func importWallet(mnemonic: String, label: String, password: String, chainId: Int) async throws -> CreatedWallet {
        await ensureSession()
        let w = try await UserWalletApiService.shared.createWallet(label: label, password: password, chainId: chainId, mnemonic: mnemonic)
        return CreatedWallet(mnemonic: mnemonic, id: w.id, address: w.address)
    }

    // MARK: - rememberWallet (mirror rememberWallet)

    func rememberWallet(_ id: String) {
        walletIds = KeychainService.loadWalletIds()
        guard !walletIds.contains(id) else { return }
        walletIds.append(id)
        KeychainService.saveWalletIds(walletIds)
        onboarded = walletIds.count > 0
    }

    // MARK: - Reset (clears local onboarding state; mirrors logout path)

    func reset() {
        KeychainService.deleteSession()
        KeychainService.saveWalletIds([])
        walletIds = []
        onboarded = false
        UserWalletApiService.shared.logout()
    }
}
