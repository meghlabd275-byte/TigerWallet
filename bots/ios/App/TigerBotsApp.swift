import SwiftUI

@main
struct TigerBotsApp: App {
    @StateObject private var themeManager = ThemeManager()
    
    var body: some Scene {
        WindowGroup {
            BotsContentView()
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

struct BotsContentView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @State private var selectedTab = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            BotsDashboardView()
                .tabItem { Image(systemName: "chart.bar"); Text("Dashboard") }
                .tag(0)
            
            MyBotsView()
                .tabItem { Image(systemName: "robot"); Text("Bots") }
                .tag(1)
            
            StrategiesView()
                .tabItem { Image(systemName: "brain"); Text("Strategies") }
                .tag(2)
            
            BotsSettingsView()
                .tabItem { Image(systemName: "gear"); Text("Settings") }
                .tag(3)
        }
        .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
    }
}

struct BotsDashboardView: View {
    var body: some View {
        NavigationView {
            VStack(alignment: .leading) {
                Text("Bot Dashboard")
                    .font(.largeTitle)
                    .fontWeight(.bold)
            }.padding()
        }
    }
}

struct MyBotsView: View {
    var body: some View {
        NavigationView {
            Text("My Bots")
        }.navigationTitle("Bots")
    }
}

struct StrategiesView: View {
    var body: some View {
        NavigationView {
            Text("Strategies")
        }.navigationTitle("Strategies")
    }
}

struct BotsSettingsView: View {
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        NavigationView {
            List {
                Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
            }
        }.navigationTitle("Settings")
    }
}
