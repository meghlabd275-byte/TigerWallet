import SwiftUI

@main
struct TigerBotsApp: App {
    @StateObject private var themeManager = ThemeManager()
    @StateObject private var session = BotSession()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(themeManager)
                .environmentObject(session)
        }
    }
}

/// Persisted light/dark preference shared by every screen.
final class ThemeManager: ObservableObject {
    @Published var isDarkMode: Bool {
        didSet { UserDefaults.standard.set(isDarkMode, forKey: "isDarkMode") }
    }

    init() {
        if UserDefaults.standard.object(forKey: "isDarkMode") == nil {
            self.isDarkMode = true
        } else {
            self.isDarkMode = UserDefaults.standard.bool(forKey: "isDarkMode")
        }
    }
}

/// Holds the authenticated user for the lifetime of the app. A stored JWT in
/// the Keychain is revalidated against `/bots/me` at launch, so a revoked or
/// expired token drops the user back to the sign-in screen instead of showing
/// a shell of empty screens.
@MainActor
final class BotSession: ObservableObject {
    @Published var isAuthenticated = false
    @Published var userId: String?
    @Published var role: String?
    @Published var isRestoring = true

    var isAdmin: Bool {
        (role ?? "").lowercased() == "admin"
    }

    init() {
        Task { await restore() }
    }

    func restore() async {
        defer { isRestoring = false }
        guard BotApiService.shared.token != nil else { return }
        do {
            let me = try await BotApiService.shared.currentBotUser()
            let user = me["user"] as? [String: Any] ?? me
            self.userId = user["id"] as? String
            self.role = user["role"] as? String
            self.isAuthenticated = true
        } catch {
            BotApiService.shared.clearToken()
            self.isAuthenticated = false
        }
    }

    func signedIn(_ auth: BotAuthResponse) {
        self.userId = auth.userId
        self.role = auth.role
        self.isAuthenticated = true
    }

    func signOut() {
        BotApiService.shared.clearToken()
        self.userId = nil
        self.role = nil
        self.isAuthenticated = false
    }
}

struct RootView: View {
    @EnvironmentObject var session: BotSession

    var body: some View {
        Group {
            if session.isRestoring {
                ProgressView("Connecting…")
            } else if session.isAuthenticated {
                MainTabView()
            } else {
                AuthView()
            }
        }
    }
}

struct MainTabView: View {
    @EnvironmentObject var session: BotSession
    @State private var selectedTab = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem { Label("Dashboard", systemImage: "chart.bar") }
                .tag(0)

            BotsView()
                .tabItem { Label("Bots", systemImage: "cpu") }
                .tag(1)

            ConnectorsView()
                .tabItem { Label("Connectors", systemImage: "link") }
                .tag(2)

            if session.isAdmin {
                AdminView()
                    .tabItem { Label("Admin", systemImage: "shield.lefthalf.filled") }
                    .tag(3)
            }

            SettingsView()
                .tabItem { Label("Settings", systemImage: "gear") }
                .tag(4)
        }
    }
}
