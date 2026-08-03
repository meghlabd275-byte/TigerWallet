//
//  MasterWalletApp.swift
//  TigerMasterWallet - Complete Production iOS Master Wallet Application
//  MasterWallet: Handles enterprise wallet management, multi-user wallets, automated signing
//

import SwiftUI

@main
struct MasterWalletApp: App {
    @StateObject private var appState = MasterAppState()
    @StateObject private var themeManager = MasterThemeManager()
    
    var body: some Scene {
        WindowGroup {
            MasterMainTabView()
                .environmentObject(appState)
                .environmentObject(themeManager)
                .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
        }
    }
}

// MARK: - Master App State
class MasterAppState: ObservableObject {
    @Published var isAuthenticated: Bool = false
    @Published var masterWallet: MasterWallet?
    @Published var subWallets: [SubWallet] = []
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?
    @Published var selectedChain: Chain = .ethereum
    @Published var permissions: MasterPermissions?
    
    let apiService: MasterAPIService
    let walletService: MasterWalletService
    let autoSignService: AutoSignService
    
    init() {
        self.apiService = MasterAPIService()
        self.walletService = MasterWalletService(apiService: apiService)
        self.autoSignService = AutoSignService(apiService: apiService)
    }
}

// MARK: - Theme Manager
class MasterThemeManager: ObservableObject {
    @Published var isDarkMode: Bool {
        didSet {
            UserDefaults.standard.set(isDarkMode, forKey: "master_wallet_theme")
        }
    }
    
    init() {
        self.isDarkMode = UserDefaults.standard.bool(forKey: "master_wallet_theme")
    }
}

// MARK: - Main Tab View
struct MasterMainTabView: View {
    @EnvironmentObject var appState: MasterAppState
    @EnvironmentObject var themeManager: MasterThemeManager
    @State private var selectedTab: Int = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem {
                    Label("Dashboard", systemImage: "building.columns.fill")
                }
                .tag(0)
            
            WalletsView()
                .tabItem {
                    Label("Wallets", systemImage: "wallet.pass.fill")
                }
                .tag(1)
            
            TransactionsView()
                .tabItem {
                    Label("Transactions", systemImage: "arrow.left.arrow.right")
                }
                .tag(2)
            
            SettingsView()
                .tabItem {
                    Label("Settings", systemImage: "gearshape.fill")
                }
                .tag(3)
        }
        .tint(.blue)
    }
}

// MARK: - Dashboard View
struct DashboardView: View {
    @EnvironmentObject var appState: MasterAppState
    @EnvironmentObject var themeManager: MasterThemeManager
    
    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    // Header
                    HStack {
                        Text("🏦").font(.largeTitle)
                        Text("MasterWallet").font(.title).fontWeight(.bold)
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
                    HStack(spacing: 15) {
                        StatCard(title: "Total Wallets", value: "\(appState.subWallets.count)", icon: "wallet.pass")
                        StatCard(title: "Total Volume", value: "$12.5M", icon: "dollarsign.circle")
                    }
                    .padding(.horizontal)
                    
                    // Quick Actions
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Quick Actions").font(.headline).padding(.horizontal)
                        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 15) {
                            MasterActionButton(icon: "plus.circle.fill", title: "Create Wallet", color: .blue) { }
                            MasterActionButton(icon: "person.badge.plus", title: "Add User", color: .green) { }
                            MasterActionButton(icon: "key.fill", title: "Auto Sign", color: .orange) { }
                            MasterActionButton(icon: "chart.bar.fill", title: "Analytics", color: .purple) { }
                        }
                        .padding(.horizontal)
                    }
                    
                    // Recent Activity
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Recent Activity").font(.headline).padding(.horizontal)
                        ForEach(0..<5) { _ in
                            ActivityRow()
                        }
                    }
                }
            }
            .navigationTitle("MasterWallet")
        }
    }
}

struct StatCard: View {
    let title: String
    let value: String
    let icon: String
    
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: icon).foregroundColor(.blue)
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

struct MasterActionButton: View {
    let icon: String
    let title: String
    let color: Color
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            VStack(spacing: 8) {
                Image(systemName: icon).font(.system(size: 30)).foregroundColor(color)
                Text(title).font(.caption)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 20)
            .background(Color(.secondarySystemBackground))
            .cornerRadius(15)
        }
    }
}

struct ActivityRow: View {
    var body: some View {
        HStack {
            Image(systemName: "arrow.up.right").foregroundColor(.green)
            VStack(alignment: .leading) {
                Text("Transaction Sent").font(.subheadline)
                Text("0x742d...12eB3").font(.caption).foregroundColor(.secondary)
            }
            Spacer()
            Text("+$5,000").foregroundColor(.green)
        }
        .padding()
        .background(Color(.secondarySystemBackground))
        .cornerRadius(10)
        .padding(.horizontal)
    }
}

// MARK: - Wallets View
struct WalletsView: View {
    @EnvironmentObject var appState: MasterAppState
    
    var body: some View {
        NavigationStack {
            List {
                ForEach(appState.subWallets) { wallet in
                    SubWalletRow(wallet: wallet)
                }
            }
            .navigationTitle("Sub-Wallets")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button {
                        // Add wallet
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
        }
    }
}

struct SubWalletRow: View {
    let wallet: SubWallet
    
    var body: some View {
        HStack {
            VStack(alignment: .leading) {
                Text(wallet.name).font(.headline)
                Text(wallet.address).font(.caption).foregroundColor(.secondary)
            }
            Spacer()
            VStack(alignment: .trailing) {
                Text("$\(wallet.balanceUSD)").font(.headline)
                Text(wallet.status).font(.caption).foregroundColor(wallet.status == "Active" ? .green : .orange)
            }
        }
    }
}

// MARK: - Transactions View
struct TransactionsView: View {
    var body: some View {
        NavigationStack {
            List {
                ForEach(0..<10) { _ in
                    TransactionRow()
                }
            }
            .navigationTitle("Transactions")
        }
    }
}

struct TransactionRow: View {
    var body: some View {
        HStack {
            Image(systemName: "arrow.up.circle.fill").foregroundColor(.blue).font(.title2)
            VStack(alignment: .leading) {
                Text("ETH Transfer").font(.headline)
                Text("To: 0x742d...12eB3").font(.caption).foregroundColor(.secondary)
            }
            Spacer()
            VStack(alignment: .trailing) {
                Text("-2.5 ETH").foregroundColor(.red)
                Text("Confirmed").font(.caption).foregroundColor(.green)
            }
        }
    }
}

// MARK: - Settings View
struct SettingsView: View {
    @EnvironmentObject var appState: MasterAppState
    @EnvironmentObject var themeManager: MasterThemeManager
    
    var body: some View {
        NavigationStack {
            Form {
                Section("Appearance") {
                    Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
                }
                
                Section("Security") {
                    NavigationLink("Auto-Sign Rules") { AutoSignSettingsView() }
                    NavigationLink("User Permissions") { PermissionsView() }
                    NavigationLink("API Keys") { APIKeysView() }
                }
                
                Section("Network") {
                    Picker("Default Chain", selection: $appState.selectedChain) {
                        ForEach(Chain.allCases, id: \.self) { chain in
                            Text(chain.name).tag(chain)
                        }
                    }
                }
                
                Section("About") {
                    HStack { Text("Version"); Spacer(); Text("1.0.0").foregroundColor(.secondary) }
                }
            }
            .navigationTitle("Settings")
        }
    }
}

struct AutoSignSettingsView: View {
    var body: some View {
        List {
            Text("Configure automatic transaction signing rules")
        }
        .navigationTitle("Auto-Sign Rules")
    }
}

struct PermissionsView: View {
    var body: some View {
        List {
            Text("Manage user permissions")
        }
        .navigationTitle("Permissions")
    }
}

struct APIKeysView: View {
    var body: some View {
        List {
            Text("Manage API keys for integration")
        }
        .navigationTitle("API Keys")
    }
}

// MARK: - Chain Enum
enum Chain: String, CaseIterable, Codable {
    case ethereum = "ethereum"
    case bsc = "bsc"
    case polygon = "polygon"
    
    var name: String {
        switch self {
        case .ethereum: return "Ethereum"
        case .bsc: return "BNB Chain"
        case .polygon: return "Polygon"
        }
    }
}

// MARK: - Models
struct MasterWallet: Codable {
    let id: String
    let address: String
    let publicKey: String
    let name: String
    let createdAt: Date
    var totalValueUSD: Double
}

struct SubWallet: Codable, Identifiable {
    let id: String
    let name: String
    let address: String
    var balanceUSD: Double
    var status: String
    var permissions: [String]
}

struct MasterPermissions: Codable {
    var canAutoSign: Bool
    var canAirdrop: Bool
    var canClaim: Bool
    var canAdjustFees: Bool
    var maxTransactionLimit: Double
}
