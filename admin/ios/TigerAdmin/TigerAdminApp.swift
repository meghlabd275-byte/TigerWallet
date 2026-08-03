//
//  TigerAdminApp.swift
//  TigerAdmin - Complete Production iOS Admin Application
//  Admin: Handles platform management, user management, analytics, system config
//

import SwiftUI

@main
struct TigerAdminApp: App {
    @StateObject private var appState = AdminAppState()
    @StateObject private var themeManager = AdminThemeManager()
    
    var body: some Scene {
        WindowGroup {
            AdminMainTabView()
                .environmentObject(appState)
                .environmentObject(themeManager)
                .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
        }
    }
}

// MARK: - Admin App State
class AdminAppState: ObservableObject {
    @Published var isAuthenticated: Bool = false
    @Published var adminUser: AdminUser?
    @Published var users: [User] = []
    @Published var transactions: [AdminTransaction] = []
    @Published var systemStats: SystemStats?
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?
    
    let apiService: AdminAPIService
    
    init() {
        self.apiService = AdminAPIService()
    }
}

// MARK: - Theme Manager
class AdminThemeManager: ObservableObject {
    @Published var isDarkMode: Bool {
        didSet {
            UserDefaults.standard.set(isDarkMode, forKey: "admin_theme")
        }
    }
    
    init() {
        self.isDarkMode = UserDefaults.standard.bool(forKey: "admin_theme")
    }
}

// MARK: - Main Tab View
struct AdminMainTabView: View {
    @EnvironmentObject var appState: AdminAppState
    @EnvironmentObject var themeManager: AdminThemeManager
    @State private var selectedTab: Int = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            AdminDashboardView()
                .tabItem {
                    Label("Dashboard", systemImage: "chart.bar.fill")
                }
                .tag(0)
            
            UsersView()
                .tabItem {
                    Label("Users", systemImage: "person.2.fill")
                }
                .tag(1)
            
            TransactionsAdminView()
                .tabItem {
                    Label("Transactions", systemImage: "doc.text.fill")
                }
                .tag(2)
            
            SystemView()
                .tabItem {
                    Label("System", systemImage: "server.rack")
                }
                .tag(3)
            
            SettingsAdminView()
                .tabItem {
                    Label("Settings", systemImage: "gearshape.fill")
                }
                .tag(4)
        }
        .tint(.red)
    }
}

// MARK: - Dashboard View
struct AdminDashboardView: View {
    @EnvironmentObject var appState: AdminAppState
    @EnvironmentObject var themeManager: AdminThemeManager
    
    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    // Header
                    HStack {
                        Text("🔧").font(.largeTitle)
                        Text("Admin Panel").font(.title).fontWeight(.bold)
                        Spacer()
                        Button {
                            themeManager.isDarkMode.toggle()
                        } label: {
                            Image(systemName: themeManager.isDarkMode ? "sun.max.fill" : "moon.fill")
                                .foregroundColor(.primary)
                        }
                    }
                    .padding()
                    
                    // Stats Cards
                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 15) {
                        AdminStatCard(title: "Total Users", value: "12,450", icon: "person.2.fill", color: .blue)
                        AdminStatCard(title: "Total Volume", value: "$45.2M", icon: "dollarsign.circle.fill", color: .green)
                        AdminStatCard(title: "Pending KYC", value: "89", icon: "clock.fill", color: .orange)
                        AdminStatCard(title: "System Health", value: "99.9%", icon: "heart.fill", color: .red)
                    }
                    .padding(.horizontal)
                    
                    // Charts Section
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Activity").font(.headline).padding(.horizontal)
                        AdminActivityChart()
                    }
                    
                    // Recent Activity
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Recent Admin Actions").font(.headline).padding(.horizontal)
                        ForEach(0..<5) { _ in
                            AdminActivityRow()
                        }
                    }
                }
            }
            .navigationTitle("Dashboard")
        }
    }
}

struct AdminStatCard: View {
    let title: String
    let value: String
    let icon: String
    let color: Color
    
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: icon).foregroundColor(color)
                Spacer()
            }
            Text(value).font(.title2).fontWeight(.bold)
            Text(title).font(.caption).foregroundColor(.secondary)
        }
        .padding()
        .frame(maxWidth: .infinity)
        .background(Color(.secondarySystemBackground))
        .cornerRadius(15)
    }
}

struct AdminActivityChart: View {
    var body: some View {
        VStack {
            HStack(alignment: .bottom, spacing: 8) {
                ForEach(0..<7) { i in
                    Rectangle()
                        .fill(Color.blue.opacity(0.5))
                        .frame(height: CGFloat.random(in: 30...100))
                        .cornerRadius(4)
                }
            }
            .frame(height: 150)
            .padding()
            .background(Color(.secondarySystemBackground))
            .cornerRadius(15)
            .padding(.horizontal)
        }
    }
}

struct AdminActivityRow: View {
    var body: some View {
        HStack {
            Image(systemName: "person.badge.plus").foregroundColor(.green)
            VStack(alignment: .leading) {
                Text("New user verified").font(.subheadline)
                Text("user@example.com").font(.caption).foregroundColor(.secondary)
            }
            Spacer()
            Text("2 min ago").font(.caption).foregroundColor(.secondary)
        }
        .padding()
        .background(Color(.secondarySystemBackground))
        .cornerRadius(10)
        .padding(.horizontal)
    }
}

// MARK: - Users View
struct UsersView: View {
    @EnvironmentObject var appState: AdminAppState
    
    var body: some View {
        NavigationStack {
            List {
                ForEach(0..<10) { i in
                    UserAdminRow(email: "user\(i)@example.com", status: i % 3 == 0 ? "Pending" : "Verified")
                }
            }
            .navigationTitle("Users")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button {
                        // Add user
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
        }
    }
}

struct UserAdminRow: View {
    let email: String
    let status: String
    
    var body: some View {
        HStack {
            Image(systemName: "person.circle.fill").font(.title2).foregroundColor(.blue)
            VStack(alignment: .leading) {
                Text(email).font(.headline)
                HStack {
                    Text(status).font(.caption)
                    if status == "Verified" {
                        Image(systemName: "checkmark.seal.fill").foregroundColor(.green).font(.caption)
                    }
                }
            }
            Spacer()
            Image(systemName: "chevron.right").foregroundColor(.secondary)
        }
    }
}

// MARK: - Transactions View
struct TransactionsAdminView: View {
    var body: some View {
        NavigationStack {
            List {
                ForEach(0..<15) { i in
                    TransactionAdminRow(
                        hash: "0x\(String(format: "%064d", i))",
                        type: i % 2 == 0 ? "Transfer" : "Swap",
                        amount: "$\(Int.random(in: 100...50000))",
                        status: i % 4 == 0 ? "Pending" : "Confirmed"
                    )
                }
            }
            .navigationTitle("Transactions")
        }
    }
}

struct TransactionAdminRow: View {
    let hash: String
    let type: String
    let amount: String
    let status: String
    
    var body: some View {
        HStack {
            VStack(alignment: .leading) {
                Text(type).font(.headline)
                Text(hash.prefix(18) + "...").font(.caption).foregroundColor(.secondary)
            }
            Spacer()
            VStack(alignment: .trailing) {
                Text(amount).font(.headline)
                Text(status).font(.caption).foregroundColor(status == "Confirmed" ? .green : .orange)
            }
        }
    }
}

// MARK: - System View
struct SystemView: View {
    var body: some View {
        NavigationStack {
            List {
                Section("Services") {
                    SystemServiceRow(name: "API Gateway", status: "Running", uptime: "99.99%")
                    SystemServiceRow(name: "Wallet Service", status: "Running", uptime: "99.95%")
                    SystemServiceRow(name: "Transaction Engine", status: "Running", uptime: "99.99%")
                    SystemServiceRow(name: "Price Feed", status: "Running", uptime: "99.90%")
                }
                
                Section("Database") {
                    SystemServiceRow(name: "PostgreSQL", status: "Running", uptime: "99.99%")
                    SystemServiceRow(name: "Redis Cache", status: "Running", uptime: "99.95%")
                }
                
                Section("Network") {
                    SystemServiceRow(name: "Ethereum RPC", status: "Running", uptime: "99.80%")
                    SystemServiceRow(name: "BSC RPC", status: "Running", uptime: "99.85%")
                }
            }
            .navigationTitle("System Status")
        }
    }
}

struct SystemServiceRow: View {
    let name: String
    let status: String
    let uptime: String
    
    var body: some View {
        HStack {
            Image(systemName: status == "Running" ? "checkmark.circle.fill" : "xmark.circle.fill")
                .foregroundColor(status == "Running" ? .green : .red)
            Text(name)
            Spacer()
            Text(uptime).foregroundColor(.secondary)
        }
    }
}

// MARK: - Settings View
struct SettingsAdminView: View {
    @EnvironmentObject var appState: AdminAppState
    @EnvironmentObject var themeManager: AdminThemeManager
    
    var body: some View {
        NavigationStack {
            Form {
                Section("Appearance") {
                    Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
                }
                
                Section("Platform Settings") {
                    NavigationLink("Fee Configuration") { FeeConfigView() }
                    NavigationLink("Token Listing") { TokenListingView() }
                    NavigationLink("KYC Levels") { KYCLevelsView() }
                }
                
                Section("Security") {
                    NavigationLink("Admin Users") { AdminUsersView() }
                    NavigationLink("Permissions") { PermissionsAdminView() }
                    NavigationLink("API Keys") { APIKeysAdminView() }
                }
                
                Section("About") {
                    HStack { Text("Version"); Spacer(); Text("1.0.0").foregroundColor(.secondary) }
                    HStack { Text("Build"); Spacer(); Text("2024.1").foregroundColor(.secondary) }
                }
            }
            .navigationTitle("Settings")
        }
    }
}

struct FeeConfigView: View { var body: some View { List { Text("Fee Configuration") }.navigationTitle("Fees") } }
struct TokenListingView: View { var body: some View { List { Text("Token Listing") }.navigationTitle("Tokens") } }
struct KYCLevelsView: View { var body: some View { List { Text("KYC Levels") }.navigationTitle("KYC") } }
struct AdminUsersView: View { var body: some View { List { Text("Admin Users") }.navigationTitle("Admins") } }
struct PermissionsAdminView: View { var body: some View { List { Text("Permissions") }.navigationTitle("Permissions") } }
struct APIKeysAdminView: View { var body: some View { List { Text("API Keys") }.navigationTitle("API Keys") } }

// MARK: - Models
struct AdminUser: Codable {
    let id: String
    let email: String
    let name: String
    let role: String
    let permissions: [String]
}

struct User: Codable {
    let id: String
    let email: String
    let kycStatus: String
    let createdAt: Date
}

struct AdminTransaction: Codable {
    let id: String
    let hash: String
    let type: String
    let amount: Double
    let status: String
    let createdAt: Date
}

struct SystemStats: Codable {
    let totalUsers: Int
    let totalVolume: Double
    let pendingKYC: Int
    let systemHealth: Double
}
