import SwiftUI

@main
struct TigerProjectPartyApp: App {
    @StateObject private var themeManager = PartyThemeManager()
    @StateObject private var session = PartySession()

    var body: some Scene {
        WindowGroup {
            PartyRootView()
                .environmentObject(themeManager)
                .environmentObject(session)
        }
    }
}

/// Persisted light/dark preference shared by every screen.
final class PartyThemeManager: ObservableObject {
    @Published var isDarkMode: Bool {
        didSet { UserDefaults.standard.set(isDarkMode, forKey: "projectparty.isDarkMode") }
    }

    init() {
        if UserDefaults.standard.object(forKey: "projectparty.isDarkMode") == nil {
            self.isDarkMode = true
        } else {
            self.isDarkMode = UserDefaults.standard.bool(forKey: "projectparty.isDarkMode")
        }
    }
}

/// Holds the authenticated user. The stored JWT is revalidated at launch by
/// calling an authenticated endpoint; a revoked token returns the user to the
/// sign-in screen rather than a shell of empty screens.
@MainActor
final class PartySession: ObservableObject {
    @Published var isAuthenticated = false
    @Published var username: String?
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
        guard PartyApiService.shared.token != nil else { return }
        do {
            // /favorites is authenticated; a 200 proves the token is still valid.
            _ = try await PartyApiService.shared.listFavorites()
            self.isAuthenticated = true
        } catch PartyApiError.unauthorized {
            PartyApiService.shared.clearToken()
            self.isAuthenticated = false
        } catch {
            // A network failure must not sign the user out; only an explicit
            // 401 does. Keep the session and let screens surface the error.
            self.isAuthenticated = true
        }
    }

    func signedIn(_ auth: PartyAuthResponse) {
        self.username = auth.username
        self.role = auth.role
        self.isAuthenticated = true
    }

    func signOut() {
        PartyApiService.shared.clearToken()
        self.username = nil
        self.role = nil
        self.isAuthenticated = false
    }
}

struct PartyRootView: View {
    @EnvironmentObject var session: PartySession

    var body: some View {
        Group {
            if session.isRestoring {
                ProgressView("Connecting…")
            } else if session.isAuthenticated {
                PartyMainTabView()
            } else {
                PartyAuthView()
            }
        }
    }
}

struct PartyMainTabView: View {
    @EnvironmentObject var session: PartySession
    @State private var selectedTab = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            MarketDiscoveryView()
                .tabItem { Label("Market", systemImage: "chart.line.uptrend.xyaxis") }
                .tag(0)

            TokensView()
                .tabItem { Label("Tokens", systemImage: "circle.grid.2x2") }
                .tag(1)

            LaunchpadView()
                .tabItem { Label("Launchpad", systemImage: "rocket") }
                .tag(2)

            if session.isAdmin {
                PartyAdminView()
                    .tabItem { Label("Admin", systemImage: "shield.lefthalf.filled") }
                    .tag(3)
            }

            PartySettingsView()
                .tabItem { Label("Settings", systemImage: "gear") }
                .tag(4)
        }
    }
}
