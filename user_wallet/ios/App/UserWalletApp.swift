import SwiftUI

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

struct RootView: View {
    @EnvironmentObject var authState: AuthState

    var body: some View {
        Group {
            if authState.isAuthenticated {
                ContentView()
            } else {
                LoginView()
            }
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

            TransactionsView()
                .tabItem { Image(systemName: "list.bullet.rectangle"); Text("Transactions") }
                .tag(2)

            SettingsView()
                .tabItem { Image(systemName: "gear"); Text("Settings") }
                .tag(3)
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
                    _ = try await UserWalletApiService.shared.register(email: email, username: username.isEmpty ? email : username, password: password)
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
