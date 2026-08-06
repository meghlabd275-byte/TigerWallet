import SwiftUI

@main
struct UserWalletApp: App {
    @StateObject private var themeManager = ThemeManager()
    
    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(themeManager)
        }
    }
}

class ThemeManager: ObservableObject {
    @Published var isDarkMode: Bool {
        didSet {
            UserDefaults.standard.set(isDarkMode, forKey: "isDarkMode")
        }
    }
    
    init() {
        self.isDarkMode = UserDefaults.standard.bool(forKey: "isDarkMode")
    }
}

struct ContentView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @State private var selectedTab = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem {
                    Image(systemName: "chart.bar")
                    Text("Dashboard")
                }
                .tag(0)
            
            WalletsView()
                .tabItem {
                    Image(systemName: "wallet.pass")
                    Text("Wallets")
                }
                .tag(1)
            
            TransactionsView()
                .tabItem {
                    Image(systemName: "list.bullet.rectangle")
                    Text("Transactions")
                }
                .tag(2)
            
            SettingsView()
                .tabItem {
                    Image(systemName: "gear")
                    Text("Settings")
                }
                .tag(3)
        }
        .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
    }
}
