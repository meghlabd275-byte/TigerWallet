import SwiftUI
import UIKit

extension Notification.Name {
    static let didLogout = Notification.Name("UserWalletDidLogout")
    static let didLogin = Notification.Name("UserWalletDidLogin")
}

@main
struct UserWalletApp: App {
    @StateObject private var themeManager = ThemeManager()
    @StateObject private var authState = AuthState()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(themeManager)
                .environmentObject(authState)
                .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
        }
    }
}

class ThemeManager: ObservableObject {
    @Published var isDarkMode: Bool {
        didSet { UserDefaults.standard.set(isDarkMode, forKey: "isDarkMode") }
    }

    init() {
        self.isDarkMode = UserDefaults.standard.object(forKey: "isDarkMode") as? Bool
            ?? (UITraitCollection.current.userInterfaceStyle == .dark)
    }
}

class AuthState: ObservableObject {
    @Published var isAuthenticated: Bool = UserWalletApiService.shared.isAuthenticated

    init() {
        NotificationCenter.default.addObserver(forName: .didLogin, object: nil, queue: .main) { [weak self] _ in
            self?.isAuthenticated = true
        }
        NotificationCenter.default.addObserver(forName: .didLogout, object: nil, queue: .main) { [weak self] _ in
            self?.isAuthenticated = false
        }
    }
}

enum OnboardingAction {
    case create
    case `import`
}

struct RootView: View {
    @EnvironmentObject var authState: AuthState

    // Set right after a successful guestAuth() so the create/import flow is
    // presented on top of ContentView. Cleared when that flow is dismissed.
    @State private var onboardingMode: AddWalletView.Mode?
    @State private var showOnboardingCover = false
    @State private var guestError: String?
    @State private var isAuthenticatingGuest = false

    var body: some View {
        Group {
            if authState.isAuthenticated {
                // Returning users (token already present at launch) unlock
                // straight into the TabView with no re-entry to the guest
                // screen. First-open guests reach here after guestAuth() and
                // get the create/import cover presented below.
                ContentView()
            } else {
                GuestEntryView(
                    isAuthenticating: $isAuthenticatingGuest,
                    error: $guestError,
                    onChoose: { chosen in startGuestAuth(action: chosen) }
                )
            }
        }
        .fullScreenCover(isPresented: $showOnboardingCover) {
            AddWalletView(mode: onboardingMode ?? .create) { _, _ in
                // Wallet created/imported. AddWalletView dismisses itself on
                // import, or reveals the mnemonic for copy then dismiss on
                // create. Either way we land on ContentView next.
            }
        }
    }

    // Stable per-device identifier: prefer identifierForVendor, fall back to a
    // persisted UUID in UserDefaults so the same guest account is reused.
    private static func resolveDeviceId() -> String {
        if let vendor = UIDevice.current.identifierForVendor?.uuidString, !vendor.isEmpty {
            return vendor
        }
        let key = "userwallet-guest-device-id"
        if let stored = UserDefaults.standard.string(forKey: key), !stored.isEmpty {
            return stored
        }
        let fallback = UUID().uuidString
        UserDefaults.standard.set(fallback, forKey: key)
        return fallback
    }

    private func startGuestAuth(action: OnboardingAction) {
        isAuthenticatingGuest = true
        guestError = nil
        let deviceId = Self.resolveDeviceId()
        Task {
            do {
                _ = try await UserWalletApiService.shared.guestAuth(deviceId: deviceId)
                await MainActor.run {
                    self.onboardingMode = (action == .create) ? .create : .import
                    // Flip auth state so ContentView is the root, then present
                    // the create/import flow on top of it.
                    NotificationCenter.default.post(name: .didLogin, object: nil)
                    self.showOnboardingCover = true
                    self.isAuthenticatingGuest = false
                }
            } catch {
                await MainActor.run {
                    self.guestError = error.localizedDescription
                    self.isAuthenticatingGuest = false
                }
            }
        }
    }
}

// First-launch screen: no email/password, just Create / Import. Email login is
// an optional toggle so the original flow still works for users who registered.
struct GuestEntryView: View {
    @Binding var isAuthenticating: Bool
    @Binding var error: String?
    let onChoose: (OnboardingAction) -> Void

    @State private var showEmailLogin = false

    var body: some View {
        NavigationView {
            VStack(spacing: 24) {
                Text("TigerWallet")
                    .font(.largeTitle)
                    .fontWeight(.bold)
                    .foregroundColor(.orange)

                Text("No registration required. Create or import a wallet to get started.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal)

                if let error = error {
                    Text(error)
                        .foregroundColor(.red)
                        .font(.subheadline)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal)
                }

                if isAuthenticating {
                    ProgressView("Setting up guest session...")
                        .padding(.top, 8)
                }

                VStack(spacing: 14) {
                    Button {
                        onChoose(.create)
                    } label: {
                        Label("Create Wallet", systemImage: "plus.circle.fill")
                            .frame(maxWidth: .infinity, minHeight: 54)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.orange)
                    .disabled(isAuthenticating)

                    Button {
                        onChoose(.import)
                    } label: {
                        Label("Import Wallet", systemImage: "square.and.arrow.down")
                            .frame(maxWidth: .infinity, minHeight: 54)
                    }
                    .buttonStyle(.bordered)
                    .disabled(isAuthenticating)
                }
                .padding(.horizontal)

                Button(showEmailLogin ? "Hide email sign in" : "Sign in with email") {
                    withAnimation { showEmailLogin.toggle() }
                }
                .font(.subheadline)
                .foregroundColor(.orange)

                if showEmailLogin {
                    LoginView()
                        .frame(maxHeight: 380)
                        .transition(.opacity)
                }

                Spacer()
            }
            .padding(.top, 36)
            .navigationBarHidden(true)
        }
    }
}

struct ContentView: View {
    @State private var selectedTab = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem { Image(systemName: "chart.bar"); Text("Dashboard") }
                .tag(0)

            WalletsView()
                .tabItem { Image(systemName: "wallet.pass"); Text("Wallets") }
                .tag(1)

            SendView()
                .tabItem { Image(systemName: "arrow.up.arrow.down"); Text("Send") }
                .tag(2)

            TransactionsView()
                .tabItem { Image(systemName: "list.bullet.rectangle"); Text("Transactions") }
                .tag(3)

            SettingsView()
                .tabItem { Image(systemName: "gear"); Text("Settings") }
                .tag(4)
        }
    }
}

struct LoginView: View {
    @State private var email = ""
    @State private var username = ""
    @State private var password = ""
    @State private var isRegister = false
    @State private var error: String?
    @State private var isLoading = false

    var body: some View {
        NavigationView {
            VStack(spacing: 20) {
                Text("TigerWallet")
                    .font(.largeTitle)
                    .fontWeight(.bold)
                    .foregroundColor(.orange)

                Text(isRegister ? "Create Account" : "Login")
                    .font(.title2)

                if let error = error {
                    Text(error).foregroundColor(.red).font(.subheadline)
                }

                Form {
                    Section {
                        TextField("Email", text: $email)
                            .keyboardType(.emailAddress)
                            .autocapitalization(.none)
                        if isRegister {
                            TextField("Username", text: $username)
                                .autocapitalization(.none)
                        }
                        SecureField("Password (min 8 chars)", text: $password)
                    }
                }
                .frame(maxHeight: 280)

                Button(action: authenticate) {
                    if isLoading {
                        ProgressView().tint(.white)
                    } else {
                        Text(isRegister ? "Register" : "Login")
                            .fontWeight(.semibold)
                    }
                }
                .frame(maxWidth: .infinity, minHeight: 50)
                .background(Color.orange)
                .foregroundColor(.white)
                .cornerRadius(12)
                .padding(.horizontal)
                .disabled(isLoading || email.isEmpty || password.count < 8)

                Button(isRegister ? "Already have an account? Login" : "Don't have an account? Register") {
                    isRegister.toggle()
                    error = nil
                }
                .font(.subheadline)
                .foregroundColor(.orange)

                Spacer()
            }
            .padding(.top, 40)
        }
    }

    private func authenticate() {
        isLoading = true
        error = nil
        Task {
            do {
                if isRegister {
                    _ = try await UserWalletApiService.shared.register(email: email, password: password)
                } else {
                    _ = try await UserWalletApiService.shared.login(email: email, password: password)
                }
                await MainActor.run {
                    NotificationCenter.default.post(name: .didLogin, object: nil)
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }
}
