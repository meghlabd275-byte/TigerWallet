import SwiftUI

extension Notification.Name {
    static let didLogout = Notification.Name("UserWalletDidLogout")
    static let didLogin = Notification.Name("UserWalletDidLogin")
}

@main
struct UserWalletApp: App {
    @StateObject private var themeManager = ThemeManager()
    @StateObject private var authState = AuthState()
    @StateObject private var onboardingManager = OnboardingManager.shared

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(themeManager)
                .environmentObject(authState)
                .environmentObject(onboardingManager)
                .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
                .task {
                    await onboardingManager.ensureSession()
                }
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
    @EnvironmentObject var onboardingManager: OnboardingManager

    // Root gate (mirrors web App.tsx):
    //   !ready        → boot screen (transparent session provisioning)
    //   !onboarded    → OnboardingView (Create / Import wallet)
    //   else          → ContentView (Dashboard / Wallets / Transactions / Settings)
    // The user NEVER sees a login form.
    var body: some View {
        Group {
            if !onboardingManager.ready {
                VStack(spacing: 16) {
                    ProgressView()
                    Text("Initializing secure wallet…")
                        .foregroundColor(.secondary)
                    if let err = onboardingManager.sessionError {
                        Text(err)
                            .font(.caption)
                            .foregroundColor(.red)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal)
                        Button("Retry") {
                            Task { await onboardingManager.ensureSession() }
                        }
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if !onboardingManager.onboarded {
                OnboardingView()
            } else {
                ContentView()
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

            FeaturesView()
                .tabItem { Image(systemName: "square.grid.2x2"); Text("More") }
                .tag(3)

            SettingsView()
                .tabItem { Image(systemName: "gear"); Text("Settings") }
                .tag(4)
        }
    }
}
