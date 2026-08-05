//
//  TigerWallet Admin Panel - iOS Application
//  Complete trading platform admin for iOS
//

import SwiftUI

@main
struct TigerAdminPanelApp: App {
    @StateObject private var authManager = AuthManager()
    @StateObject private var themeManager = ThemeManager()
    
    var body: some Scene {
        WindowGroup {
            if authManager.isAuthenticated {
                MainView()
                    .environmentObject(authManager)
                    .environmentObject(themeManager)
                    .preferredColorScheme(themeManager.colorScheme)
            } else {
                LoginView()
                    .environmentObject(authManager)
                    .environmentObject(themeManager)
            }
        }
    }
}

// MARK: - Models
struct PlatformStats: Codable {
    let totalUsers: Int
    let activeUsers: Int
    let totalVolume: Double
    let totalTransactions: Int
    let activeBots: Int
    let totalBots: Int
    let activeDexConnections: Int
    let activeCexConnections: Int
    
    enum CodingKeys: String, CodingKey {
        case totalUsers = "total_users"
        case activeUsers = "active_users"
        case totalVolume = "total_volume"
        case totalTransactions = "total_transactions"
        case activeBots = "active_bots"
        case totalBots = "total_bots"
        case activeDexConnections = "active_dex_connections"
        case activeCexConnections = "active_cex_connections"
    }
}

struct User: Codable, Identifiable {
    let id: String
    let email: String
    let username: String
    let status: String
    let kycStatus: String
    let balance: [String: Double]
}

struct TradingPair: Codable, Identifiable {
    let id: String
    let base: String
    let quote: String
    let price: Double
    let volume24h: Double
    let status: String
    
    enum CodingKeys: String, CodingKey {
        case id, base, quote, price, status
        case volume24h = "volume_24h"
    }
}

struct BotInstance: Codable, Identifiable {
    let id: String
    let name: String
    let botType: String
    let status: String
    let totalPnl: Double
    let totalVolume: Double
    
    enum CodingKeys: String, CodingKey {
        case id, name, status
        case botType = "bot_type"
        case totalPnl = "total_pnl"
        case totalVolume = "total_volume"
    }
}

// MARK: - Views
struct DashboardView: View {
    @State private var stats: PlatformStats?
    @State private var isLoading = true
    
    var body: some View {
        NavigationView {
            ScrollView {
                if isLoading {
                    ProgressView()
                } else if let stats = stats {
                    VStack(spacing: 16) {
                        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 16) {
                            StatCard(title: "Total Users", value: "\(stats.totalUsers)")
                            StatCard(title: "Active Users", value: "\(stats.activeUsers)")
                            StatCard(title: "Total Volume", value: "$\(Int(stats.totalVolume))")
                            StatCard(title: "Transactions", value: "\(stats.totalTransactions)")
                            StatCard(title: "Active Bots", value: "\(stats.activeBots)")
                            StatCard(title: "Total Bots", value: "\(stats.totalBots)")
                            StatCard(title: "DEX Connections", value: "\(stats.activeDexConnections)")
                            StatCard(title: "CEX Connections", value: "\(stats.activeCexConnections)")
                        }
                    }
                    .padding()
                }
            }
            .navigationTitle("Dashboard")
            .onAppear(loadStats)
        }
    }
    
    private func loadStats() {
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
            stats = PlatformStats(
                totalUsers: 12543,
                activeUsers: 8234,
                totalVolume: 98765432,
                totalTransactions: 456789,
                activeBots: 234,
                totalBots: 567,
                activeDexConnections: 12,
                activeCexConnections: 8
            )
            isLoading = false
        }
    }
}

struct UsersView: View {
    @State private var users: [User] = []
    
    var body: some View {
        NavigationView {
            List(users) { user in
                VStack(alignment: .leading) {
                    Text(user.username).font(.headline)
                    Text(user.email).font(.caption).foregroundColor(.secondary)
                    HStack {
                        Text(user.status)
                            .font(.caption2)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(user.status == "active" ? Color.green.opacity(0.2) : Color.red.opacity(0.2))
                            .cornerRadius(4)
                        Text(user.kycStatus)
                            .font(.caption2)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.orange.opacity(0.2))
                            .cornerRadius(4)
                    }
                }
                .padding(.vertical, 4)
            }
            .navigationTitle("Users")
            .onAppear {
                users = [
                    User(id: "1", email: "user1@example.com", username: "user1", status: "active", kycStatus: "verified", balance: ["USDT": 1000]),
                    User(id: "2", email: "user2@example.com", username: "user2", status: "active", kycStatus: "pending", balance: ["BTC": 0.5]),
                    User(id: "3", email: "user3@example.com", username: "user3", status: "suspended", kycStatus: "rejected", balance: ["ETH": 2])
                ]
            }
        }
    }
}

struct TradingPairsView: View {
    @State private var pairs: [TradingPair] = []
    
    var body: some View {
        NavigationView {
            List(pairs) { pair in
                VStack(alignment: .leading) {
                    Text("\(pair.base)/\(pair.quote)").font(.headline)
                    Text("Price: $\(pair.price, specifier: "%.2f")").font(.caption)
                    HStack {
                        Text("Vol: $\(pair.volume24h, specifier: "%.0f")").font(.caption2)
                        Spacer()
                        Text(pair.status)
                            .font(.caption2)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(pair.status == "active" ? Color.green.opacity(0.2) : Color.orange.opacity(0.2))
                            .cornerRadius(4)
                    }
                }
                .padding(.vertical, 4)
            }
            .navigationTitle("Trading Pairs")
            .onAppear {
                pairs = [
                    TradingPair(id: "1", base: "BTC", quote: "USDT", price: 43250.50, volume24h: 1250000, status: "active"),
                    TradingPair(id: "2", base: "ETH", quote: "USDT", price: 2280.25, volume24h: 850000, status: "active"),
                    TradingPair(id: "3", base: "SOL", quote: "USDT", price: 98.50, volume24h: 320000, status: "active")
                ]
            }
        }
    }
}

struct BotsView: View {
    @State private var bots: [BotInstance] = []
    
    var body: some View {
        NavigationView {
            List(bots) { bot in
                VStack(alignment: .leading) {
                    Text(bot.name).font(.headline)
                    Text(bot.botType).font(.caption).foregroundColor(.secondary)
                    HStack {
                        Text("PnL: $\(bot.totalPnl, specifier: "%.2f")").font(.caption)
                        Spacer()
                        Text(bot.status)
                            .font(.caption2)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(bot.status == "running" ? Color.green.opacity(0.2) : Color.orange.opacity(0.2))
                            .cornerRadius(4)
                    }
                }
                .padding(.vertical, 4)
            }
            .navigationTitle("Trading Bots")
            .onAppear {
                bots = [
                    BotInstance(id: "1", name: "Bot Alpha", botType: "Grid", status: "running", totalPnl: 1250.50, totalVolume: 50000),
                    BotInstance(id: "2", name: "Bot Beta", botType: "DCA", status: "running", totalPnl: 890.25, totalVolume: 35000),
                    BotInstance(id: "3", name: "Bot Gamma", botType: "Scalping", status: "stopped", totalPnl: 450.00, totalVolume: 15000)
                ]
            }
        }
    }
}

// Placeholder views
struct KYCView: View { var body: some View { NavigationView { Text("KYC Management").navigationTitle("KYC") } }
struct FeesView: View { var body: some View { NavigationView { Text("Fee Management").navigationTitle("Fees") } }
struct WithdrawalsView: View { var body: some View { NavigationView { Text("Withdrawals").navigationTitle("Withdrawals") } }
struct BlockchainsView: View { var body: some View { NavigationView { Text("Blockchains").navigationTitle("Blockchains") } }
struct TokensView: View { var body: some View { NavigationView { Text("Tokens").navigationTitle("Tokens") } }
struct WhiteLabelsView: View { var body: some View { NavigationView { Text("White Labels").navigationTitle("White Labels") } } }
struct AnalyticsView: View { var body: some View { NavigationView { Text("Analytics").navigationTitle("Analytics") } } }
struct MasterWalletView: View { var body: some View { NavigationView { Text("Master Wallet").navigationTitle("Master Wallet") } } }
struct SettingsView: View { var body: some View { NavigationView { Text("Settings").navigationTitle("Settings") } } }

// MARK: - Main View with TabView
struct MainView: View {
    @EnvironmentObject var authManager: AuthManager
    @EnvironmentObject var themeManager: ThemeManager
    
    @State private var selectedTab = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem { Label("Dashboard", systemImage: "chart.bar") }.tag(0)
            UsersView()
                .tabItem { Label("Users", systemImage: "person.2") }.tag(1)
            TradingPairsView()
                .tabItem { Label("Pairs", systemImage: "arrow.left.arrow.right") }.tag(2)
            BotsView()
                .tabItem { Label("Bots", systemImage: "cpu") }.tag(3)
            WhiteLabelsView()
                .tabItem { Label("White Labels", systemImage: "building.2") }.tag(4)
        }
    }
}

// MARK: - Login View
struct LoginView: View {
    @EnvironmentObject var authManager: AuthManager
    @EnvironmentObject var themeManager: ThemeManager
    
    @State private var email = ""
    @State private var password = ""
    @State private var isLoading = false
    
    var body: some View {
        ZStack {
            Color.primary.edgesIgnoringSafeArea(.all)
            
            VStack(spacing: 24) {
                Text("🐯").font(.system(size: 64))
                Text("TigerWallet").font(.largeTitle).fontWeight(.bold)
                Text("Admin Panel").foregroundColor(.secondary)
                
                VStack(spacing: 16) {
                    TextField("Email", text: $email)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                        .autocapitalization(.none)
                    SecureField("Password", text: $password)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                }
                .padding(.horizontal, 32)
                
                Button(action: login) {
                    if isLoading {
                        ProgressView().progressViewStyle(CircularProgressViewStyle(tint: .white))
                    } else {
                        Text("Sign In")
                    }
                }
                .frame(maxWidth: .infinity)
                .padding()
                .background(Color.blue)
                .foregroundColor(.white)
                .cornerRadius(10)
                .padding(.horizontal, 32)
                
                Button(action: { themeManager.toggleTheme() }) {
                    HStack {
                        Image(systemName: themeManager.isDarkMode ? "moon.fill" : "sun.max.fill")
                        Text(themeManager.isDarkMode ? "Dark" : "Light")
                    }
                }
            }
        }
    }
    
    private func login() {
        isLoading = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 1) {
            authManager.isAuthenticated = true
            isLoading = false
        }
    }
}

// Supporting classes
class AuthManager: ObservableObject {
    @Published var isAuthenticated = false
}

class ThemeManager: ObservableObject {
    @Published var isDarkMode = false
    
    var colorScheme: ColorScheme? {
        isDarkMode ? .dark : .light
    }
    
    func toggleTheme() {
        isDarkMode.toggle()
    }
}

struct StatCard: View {
    let title: String
    let value: String
    
    var body: some View {
        VStack(alignment: .leading) {
            Text(title).font(.caption).foregroundColor(.secondary)
            Text(value).font(.title2).fontWeight(.bold)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
    }
}
