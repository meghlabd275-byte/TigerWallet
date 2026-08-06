import SwiftUI

@main
struct ProjectPartyApp: App {
    @StateObject private var themeManager = ThemeManager()
    
    var body: some Scene {
        WindowGroup {
            ProjectPartyContentView()
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

struct ProjectPartyContentView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @State private var selectedTab = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            PPDashboardView()
                .tabItem { Image(systemName: "chart.bar"); Text("Market") }
                .tag(0)
            
            CoinsView()
                .tabItem { Image(systemName: "bitcoinsign.circle"); Text("Coins") }
                .tag(1)
            
            TokensView()
                .tabItem { Image(systemName: "token"); Text("Tokens") }
                .tag(2)
            
            PPSettingsView()
                .tabItem { Image(systemName: "gear"); Text("Settings") }
                .tag(3)
        }
        .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
    }
}

struct PPDashboardView: View {
    var body: some View {
        NavigationView {
            VStack(alignment: .leading) {
                Text("Token Marketplace")
                    .font(.largeTitle)
                    .fontWeight(.bold)
            }.padding()
        }
    }
}

struct CoinsView: View {
    var body: some View {
        NavigationView {
            Text("Coins")
        }.navigationTitle("Coins")
    }
}

struct TokensView: View {
    var body: some View {
        NavigationView {
            Text("Tokens")
        }.navigationTitle("Tokens")
    }
}

struct PPSettingsView: View {
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        NavigationView {
            List {
                Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
            }
        }.navigationTitle("Settings")
    }
}
