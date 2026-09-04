//
//  MasterWalletApp.swift
//  TigerMasterWallet - Complete Production iOS Master Wallet Application
//  MasterWallet: Handles enterprise wallet management, multi-user wallets, automated signing
//

import SwiftUI

extension Notification.Name {
    static let masterWalletOpenSettings = Notification.Name("masterWalletOpenSettings")
}

@main
struct MasterWalletApp: App {
    @StateObject private var appState = MasterAppState()
    @StateObject private var themeManager = MasterThemeManager()
    
    var body: some Scene {
        WindowGroup {
            if appState.isAuthenticated {
                MasterMainTabView()
                    .environmentObject(appState)
                    .environmentObject(themeManager)
                    .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
            } else {
                LoginView()
                    .environmentObject(appState)
                    .environmentObject(themeManager)
                    .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
            }
        }
    }
}

// MARK: - Master App State
class MasterAppState: ObservableObject {
    @Published var isAuthenticated: Bool = false
    @Published var masterWallet: MasterWallet?
    @Published var subWallets: [SubWallet] = []
    @Published var recentTransactions: [MasterTransaction] = []
    @Published var totalVolumeUSD: Double = 0
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
        // Restore an existing JWT session (if any) on launch.
        if let token = apiService.authToken, !token.isEmpty {
            self.isAuthenticated = true
            Task { await self.loadMasterWalletAfterAuth() }
        }
    }

    /// Authenticate against the real backend (POST /api/v1/auth/login or
    /// /register). On success the JWT is stored by MasterAPIService and the
    /// first master wallet (if any) is selected so the dashboard can load.
    @MainActor
    func authenticate(email: String, password: String, registering: Bool, name: String) async {
        do {
            if registering {
                _ = try await apiService.register(email: email, password: password, name: name)
            } else {
                _ = try await apiService.login(email: email, password: password)
            }
            self.isAuthenticated = true
            await loadMasterWalletAfterAuth()
        } catch {
            self.errorMessage = error.localizedDescription
            self.isAuthenticated = false
        }
    }

    /// After authentication, fetch the user's master wallets and select the
    /// first one. If the user has none, `masterWallet` stays nil and the UI
    /// prompts them to create one (no fake wallet is ever synthesized).
    @MainActor
    func loadMasterWalletAfterAuth() async {
        do {
            let resp = try await apiService.listMasterWallets()
            self.masterWallet = resp.wallets.first
            if let wid = masterWallet?.id {
                await MainActor.run { self.loadDashboardData(walletId: wid) }
            }
        } catch {
            self.errorMessage = error.localizedDescription
        }
    }

    @MainActor
    func logout() {
        apiService.authToken = nil
        self.isAuthenticated = false
        self.masterWallet = nil
        self.subWallets = []
        self.recentTransactions = []
        self.totalVolumeUSD = 0
        self.permissions = nil
        self.errorMessage = nil
    }

    /// Fetch the master wallet, its sub-wallets, recent transactions, and the
    /// treasury total from the real backend. All values are real; nothing is
    /// hardcoded.
    @MainActor
    func loadDashboardData(walletId: String) {
        isLoading = true
        errorMessage = nil

        Task {
            do {
                async let wallet = apiService.getMasterWallet(id: walletId)
                async let subs = apiService.getSubWallets(masterWalletId: walletId)
                async let txns = apiService.listTransactions(walletId: walletId).transactions
                async let treasury = apiService.getTreasury(walletId: walletId)

                let (w, s, t, tr) = try await (wallet, subs, txns, treasury)
                self.masterWallet = w
                self.subWallets = s
                self.recentTransactions = t
                self.totalVolumeUSD = tr.totalValueUSD
                self.isLoading = false
            } catch {
                self.errorMessage = error.localizedDescription
                self.isLoading = false
            }
        }
    }

    /// Refresh only the sub-wallet list for the selected master wallet.
    @MainActor
    func loadSubWallets(walletId: String) {
        Task {
            do {
                self.subWallets = try await apiService.getSubWallets(masterWalletId: walletId)
            } catch {
                self.errorMessage = error.localizedDescription
            }
        }
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

            MoreView()
                .tabItem {
                    Label("More", systemImage: "square.grid.2x2.fill")
                }
                .tag(4)

            SettingsView()
                .tabItem {
                    Label("Settings", systemImage: "gearshape.fill")
                }
                .tag(3)
        }
        .tint(.blue)
        .onReceive(NotificationCenter.default.publisher(for: .masterWalletOpenSettings)) { _ in
            selectedTab = 3
        }
    }
}

// MARK: - Dashboard View
struct DashboardView: View {
    @EnvironmentObject var appState: MasterAppState
    @EnvironmentObject var themeManager: MasterThemeManager
    @State private var showingCreateWallet = false
    @StateObject private var liveFeed = LiveFeedModel()

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

                    // Stats Cards (real values from appState)
                    HStack(spacing: 15) {
                        StatCard(title: "Total Wallets", value: "\(appState.subWallets.count)", icon: "wallet.pass")
                        StatCard(title: "Total Volume", value: formatUSD(appState.totalVolumeUSD), icon: "dollarsign.circle")
                    }
                    .padding(.horizontal)

                    if appState.isLoading {
                        ProgressView().padding()
                    } else if let error = appState.errorMessage {
                        Text(error).font(.caption).foregroundColor(.red).padding(.horizontal)
                    } else if appState.masterWallet == nil {
                        ContentUnavailableView(
                            "No Master Wallet",
                            systemImage: "wallet.pass",
                            description: Text("Create a master wallet to begin.")
                        ).padding()

                        Button("Create Master Wallet") {
                            showingCreateWallet = true
                        }
                        .buttonStyle(.borderedProminent)
                        .padding(.horizontal)
                    }

                    // Quick Actions
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Quick Actions").font(.headline).padding(.horizontal)
                        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 15) {
                            MasterActionButton(icon: "plus.circle.fill", title: "Create Wallet", color: .blue) {
                                showingCreateWallet = true
                            }
                            MasterActionButton(icon: "person.badge.plus", title: "Add User", color: .green) {
                                NotificationCenter.default.post(name: .masterWalletOpenSettings, object: nil)
                            }
                            MasterActionButton(icon: "key.fill", title: "Auto Sign", color: .orange) {
                                NotificationCenter.default.post(name: .masterWalletOpenSettings, object: nil)
                            }
                            MasterActionButton(icon: "chart.bar.fill", title: "Analytics", color: .purple) {
                                NotificationCenter.default.post(name: .masterWalletOpenSettings, object: nil)
                            }
                        }
                        .padding(.horizontal)
                    }

                    // Recent Activity (real transactions from the backend)
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Recent Activity").font(.headline).padding(.horizontal)
                        if appState.recentTransactions.isEmpty {
                            Text("No recent transactions").font(.caption).foregroundColor(.secondary).padding(.horizontal)
                        } else {
                            ForEach(appState.recentTransactions) { tx in
                                ActivityRow(transaction: tx)
                            }
                        }
                    }
                }
            }
            .navigationTitle("MasterWallet")
            .refreshable {
                if let wid = appState.masterWallet?.id {
                    appState.loadDashboardData(walletId: wid)
                }
            }
            .task {
                if let wid = appState.masterWallet?.id {
                    if appState.subWallets.isEmpty {
                        appState.loadDashboardData(walletId: wid)
                    }
                    // Live backend /ws feed: real balance/transaction events
                    // refresh the dashboard instantly.
                    liveFeed.start(walletId: wid, token: appState.apiService.authToken) {
                        if let wid = appState.masterWallet?.id {
                            appState.loadDashboardData(walletId: wid)
                        }
                    }
                }
            }
            .overlay(alignment: .top) {
                if let evt = liveFeed.lastEvent {
                    Text("Live: \(evt)")
                        .font(.caption)
                        .padding(8)
                        .background(.thinMaterial)
                        .cornerRadius(8)
                        .padding(.top, 4)
                }
            }
            .onDisappear { liveFeed.stop() }
            .sheet(isPresented: $showingCreateWallet) {
                CreateMasterWalletSheet()
                    .environmentObject(appState)
            }
        }
    }

    private func formatUSD(_ value: Double) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        formatter.currencyCode = "USD"
        formatter.maximumFractionDigits = 2
        return formatter.string(from: NSNumber(value: value)) ?? "$0"
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
    let transaction: MasterTransaction

    var body: some View {
        HStack {
            Image(systemName: transaction.type.lowercased().contains("receive") ? "arrow.down.left" : "arrow.up.right")
                .foregroundColor(transaction.type.lowercased().contains("receive") ? .green : .blue)
            VStack(alignment: .leading) {
                Text(transaction.type.capitalized).font(.subheadline)
                Text(shortAddress(transaction.to)).font(.caption).foregroundColor(.secondary)
            }
            Spacer()
            VStack(alignment: .trailing) {
                Text("\(transaction.amount)").font(.subheadline)
                Text(transaction.status.capitalized).font(.caption).foregroundColor(statusColor)
            }
        }
        .padding()
        .background(Color(.secondarySystemBackground))
        .cornerRadius(10)
        .padding(.horizontal)
    }

    private var statusColor: Color {
        switch transaction.status.lowercased() {
        case "confirmed", "success", "completed": return .green
        case "pending": return .orange
        case "failed", "rejected": return .red
        default: return .secondary
        }
    }

    private func shortAddress(_ address: String) -> String {
        guard address.count > 10 else { return address }
        let prefix = address.prefix(6)
        let suffix = address.suffix(4)
        return "\(prefix)...\(suffix)"
    }
}

// MARK: - Wallets View
struct WalletsView: View {
    @EnvironmentObject var appState: MasterAppState
    @EnvironmentObject var themeManager: MasterThemeManager
    @State private var showingAddSubWallet = false

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
                        showingAddSubWallet = true
                    } label: {
                        Image(systemName: "plus")
                    }
                    .disabled(appState.masterWallet == nil)
                }
            }
            .refreshable {
                if let wid = appState.masterWallet?.id {
                    appState.loadSubWallets(walletId: wid)
                }
            }
            .task {
                if let wid = appState.masterWallet?.id, appState.subWallets.isEmpty {
                    appState.loadSubWallets(walletId: wid)
                }
            }
            .sheet(isPresented: $showingAddSubWallet) {
                CreateSubWalletSheet()
                    .environmentObject(appState)
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
    @EnvironmentObject var appState: MasterAppState
    @EnvironmentObject var themeManager: MasterThemeManager

    var body: some View {
        NavigationStack {
            Group {
                if appState.recentTransactions.isEmpty {
                    ContentUnavailableView(
                        "No Transactions",
                        systemImage: "arrow.left.arrow.right",
                        description: Text("Transactions will appear here once the master wallet is loaded.")
                    )
                } else {
                    List {
                        ForEach(appState.recentTransactions) { tx in
                            TransactionRow(transaction: tx)
                        }
                    }
                }
            }
            .navigationTitle("Transactions")
            .refreshable {
                if let wid = appState.masterWallet?.id {
                    appState.loadDashboardData(walletId: wid)
                }
            }
            .task {
                if let wid = appState.masterWallet?.id, appState.recentTransactions.isEmpty {
                    appState.loadDashboardData(walletId: wid)
                }
            }
        }
    }
}

struct TransactionRow: View {
    let transaction: MasterTransaction

    var body: some View {
        HStack {
            Image(systemName: transaction.type.lowercased().contains("receive") ? "arrow.down.circle.fill" : "arrow.up.circle.fill")
                .foregroundColor(transaction.type.lowercased().contains("receive") ? .green : .blue)
                .font(.title2)
            VStack(alignment: .leading) {
                Text(transaction.type.capitalized).font(.headline)
                Text("To: \(shortAddress(transaction.to))").font(.caption).foregroundColor(.secondary)
            }
            Spacer()
            VStack(alignment: .trailing) {
                Text("\(transaction.amount) \(transaction.chain.uppercased())").foregroundColor(.primary)
                Text(transaction.status.capitalized).font(.caption).foregroundColor(statusColor)
            }
        }
    }

    private var statusColor: Color {
        switch transaction.status.lowercased() {
        case "confirmed", "success", "completed": return .green
        case "pending": return .orange
        case "failed", "rejected": return .red
        default: return .secondary
        }
    }

    private func shortAddress(_ address: String) -> String {
        guard address.count > 10 else { return address }
        let prefix = address.prefix(6)
        let suffix = address.suffix(4)
        return "\(prefix)...\(suffix)"
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

                KillSwitchSettingsSection()

                BackendStatusSettingsSection()

                Section("Network") {
                    Picker("Default Chain", selection: $appState.selectedChain) {
                        ForEach(Chain.allCases, id: \.self) { chain in
                            Text(chain.name).tag(chain)
                        }
                    }
                }

                Section("About") {
                    HStack { Text("Version"); Spacer(); Text("1.0.0").foregroundColor(.secondary) }
                    HStack { Text("Backend"); Spacer(); Text(MasterAPIService.defaultBaseURL).foregroundColor(.secondary).font(.caption) }
                }

                Section {
                    Button(role: .destructive) {
                        appState.logout()
                    } label: {
                        Text("Log Out")
                    }
                }
            }
            .navigationTitle("Settings")
        }
    }
}

struct AutoSignSettingsView: View {
    @EnvironmentObject var appState: MasterAppState
    @State private var rules: [AutoSignRule] = []
    @State private var policy: AutoSignPolicy?
    @State private var errorText: String?
    @State private var loaded = false

    var body: some View {
        List {
            if let policy = policy {
                Section("Daemon Policy") {
                    Toggle("Auto-Sign Enabled", isOn: policyBinding(\.enabled))
                    Toggle("Allow Transfers", isOn: policyBinding(\.allowTransfer))
                    Toggle("Allow Swaps", isOn: policyBinding(\.allowSwap))
                    Toggle("Allow Staking", isOn: policyBinding(\.allowStake))
                    Toggle("Allow NFT Transfers", isOn: policyBinding(\.allowNftTransfer))
                    Toggle("Allow Personal Sign", isOn: policyBinding(\.allowPersonalSign))
                    Toggle("Allow Typed-Data Sign", isOn: policyBinding(\.allowTypedDataSign))
                    HStack {
                        Text("Max Auto Value (wei)")
                        Spacer()
                        Text(policy.maxAutoValueWei == "0" ? "Unlimited" : policy.maxAutoValueWei)
                            .foregroundColor(.secondary).font(.caption)
                    }
                }
            }
            Section("Rules") {
                ForEach(rules) { rule in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(rule.name).font(.subheadline)
                            Text(rule.ruleType).font(.caption).foregroundColor(.secondary)
                        }
                        Spacer()
                        if let maxAmount = rule.maxAmount, !maxAmount.isEmpty {
                            Text(maxAmount).font(.caption).foregroundColor(.secondary)
                        }
                    }
                }
                if loaded && rules.isEmpty { Text("No auto-sign rules.").foregroundColor(.secondary) }
            }
            if let errorText = errorText { Section { Text(errorText).font(.caption).foregroundColor(.red) } }
        }
        .navigationTitle("Auto-Sign Rules")
        .task { await load() }
    }

    private func policyBinding(_ keyPath: WritableKeyPath<AutoSignPolicy, Bool>) -> Binding<Bool> {
        Binding(
            get: { policy?[keyPath: keyPath] ?? false },
            set: { newValue in
                guard var p = policy, let wid = appState.masterWallet?.id else { return }
                p[keyPath: keyPath] = newValue
                policy = p
                Task {
                    do { policy = try await appState.apiService.updateAutoSignPolicy(walletId: wid, policy: p) }
                    catch { errorText = error.localizedDescription }
                }
            }
        )
    }

    private func load() async {
        guard let wid = appState.masterWallet?.id else { errorText = "No master wallet selected"; loaded = true; return }
        do {
            rules = try await appState.apiService.getAutoSignRules(walletId: wid)
            policy = try await appState.apiService.getAutoSignPolicy(walletId: wid)
            errorText = nil
        } catch { errorText = error.localizedDescription }
        loaded = true
    }
}

struct PermissionsView: View {
    @EnvironmentObject var appState: MasterAppState
    @State private var users: [MasterUser] = []
    @State private var errorText: String?
    @State private var loaded = false

    var body: some View {
        List {
            Section("User Roles & Access") {
                ForEach(users) { u in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(u.name).font(.subheadline)
                            Text(u.email).font(.caption).foregroundColor(.secondary)
                        }
                        Spacer()
                        Text(u.role).font(.caption).foregroundColor(.secondary)
                        Toggle("", isOn: activeBinding(u)).labelsHidden()
                    }
                }
                if loaded && users.isEmpty { Text("No users.").foregroundColor(.secondary) }
            }
            if let errorText = errorText { Section { Text(errorText).font(.caption).foregroundColor(.red) } }
        }
        .navigationTitle("Permissions")
        .task { await load() }
    }

    private func activeBinding(_ user: MasterUser) -> Binding<Bool> {
        Binding(
            get: { users.first(where: { $0.id == user.id })?.isActive ?? false },
            set: { newValue in
                guard let wid = appState.masterWallet?.id else { return }
                Task {
                    do {
                        try await appState.apiService.updateUser(walletId: wid, userId: user.id, updates: ["is_active": newValue])
                        await load()
                    } catch { errorText = error.localizedDescription }
                }
            }
        )
    }

    private func load() async {
        guard let wid = appState.masterWallet?.id else { errorText = "No master wallet selected"; loaded = true; return }
        do {
            users = try await appState.apiService.getUsers(walletId: wid)
            errorText = nil
        } catch { errorText = error.localizedDescription }
        loaded = true
    }
}

struct KillSwitchSettingsSection: View {
    @EnvironmentObject var appState: MasterAppState
    @State private var status: KillSwitchStatus?
    @State private var errorText: String?

    var body: some View {
        Section {
            HStack {
                Text("Global Halt State")
                Spacer()
                if let status = status {
                    Text(status.halted ? "HALTED" : "Operational")
                        .foregroundColor(status.halted ? .red : .green)
                        .fontWeight(.bold)
                } else {
                    Text(errorText == nil ? "Loading…" : "Unavailable").foregroundColor(.secondary)
                }
            }
            if let status = status, !status.reason.isEmpty {
                HStack { Text("Reason"); Spacer(); Text(status.reason).foregroundColor(.secondary) }
            }
            if let status = status {
                HStack { Text("Source"); Spacer(); Text(status.source).foregroundColor(.secondary).font(.caption) }
            }
            if let note = status?.note, !note.isEmpty {
                Text(note).font(.caption2).foregroundColor(.secondary)
            }
            Button("Refresh") { Task { await load() } }
        } header: {
            Text("Kill Switch (SuperAdmin)")
        } footer: {
            if let errorText = errorText { Text(errorText).font(.caption2) }
        }
        .task { await load() }
    }

    private func load() async {
        do {
            status = try await appState.apiService.getKillSwitchStatus()
            errorText = nil
        } catch { errorText = error.localizedDescription }
    }
}

struct APIKeysView: View {
    var body: some View {
        List {
            Section {
                Text("API keys are managed server-side. Create or rotate keys via the backend admin endpoints; the app does not generate or store raw secrets.")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .navigationTitle("API Keys")
    }
}

/// Liveness (/health) + readiness (/readyz) probes so the operator sees
/// cluster-degraded states (PostgreSQL down) before any API call fails.
struct BackendStatusSettingsSection: View {
    @EnvironmentObject var appState: MasterAppState
    @State private var health: HealthResponse?
    @State private var ready: [String: Any]?
    @State private var unreachable = false

    var body: some View {
        Section {
            HStack {
                Text("Liveness (/health)")
                Spacer()
                Text(unreachable ? "down" : (health?.status ?? "up"))
                    .foregroundColor(unreachable ? .red : .green)
                    .fontWeight(.bold)
            }
            HStack {
                Text("Readiness (/readyz)")
                Spacer()
                Text(ready?["status"] as? String ?? "degraded")
                    .foregroundColor(ready != nil ? .green : .red)
                    .fontWeight(.bold)
            }
            if let db = ready?["database"] as? String, !db.isEmpty {
                HStack { Text("PostgreSQL"); Spacer(); Text(db).foregroundColor(.secondary).font(.caption) }
            }
            if let r = ready?["redis"] as? String, !r.isEmpty {
                HStack { Text("Redis"); Spacer(); Text(r).foregroundColor(.secondary).font(.caption) }
            }
            if let n = ready?["node_id"] as? String, !n.isEmpty {
                HStack { Text("Node"); Spacer(); Text(n).foregroundColor(.secondary).font(.caption) }
            }
            Button("Refresh") { Task { await load() } }
        } header: {
            Text("Backend Status")
        }
        .task { await load() }
    }

    private func load() async {
        do {
            health = try await appState.apiService.getHealth()
            unreachable = false
        } catch {
            unreachable = true
            health = nil
        }
        ready = try? await appState.apiService.getReady()
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

    /// Canonical EVM chain id used by the backend.
    var chainId: Int {
        switch self {
        case .ethereum: return 1
        case .bsc: return 56
        case .polygon: return 137
        }
    }
}

// MARK: - Models
// MasterWallet, SubWallet, MasterPermissions, MasterTransaction, AutoSignRule,
// MasterUser, etc. are defined in Sources/Services/MasterAPIService.swift and
// are shared across the app. They are NOT redeclared here to avoid duplicate
// type collisions.

// MARK: - Login View
struct LoginView: View {
    @EnvironmentObject var appState: MasterAppState
    @EnvironmentObject var themeManager: MasterThemeManager

    @State private var isRegistering = false
    @State private var email = ""
    @State private var password = ""
    @State private var name = ""
    @State private var isAuthenticating = false

    var body: some View {
        NavigationStack {
            Form {
                if isRegistering {
                    Section("Account") {
                        TextField("Name", text: $name)
                            .textInputAutocapitalization(.words)
                            .autocorrectionDisabled()
                    }
                }
                Section("Credentials") {
                    TextField("Email", text: $email)
                        .keyboardType(.emailAddress)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    SecureField("Password", text: $password)
                }
                if let error = appState.errorMessage {
                    Section {
                        Text(error).font(.caption).foregroundColor(.red)
                    }
                }
                Section {
                    Button {
                        Task {
                            isAuthenticating = true
                            await appState.authenticate(
                                email: email,
                                password: password,
                                registering: isRegistering,
                                name: name
                            )
                            isAuthenticating = false
                        }
                    } label: {
                        HStack {
                            if isAuthenticating { ProgressView() }
                            Text(isRegistering ? "Register" : "Log In")
                        }
                    }
                    .disabled(email.isEmpty || password.isEmpty || (isRegistering && name.isEmpty) || isAuthenticating)

                    Button(isRegistering ? "Already have an account? Log in" : "New here? Create an account") {
                        isRegistering.toggle()
                        appState.errorMessage = nil
                    }
                }
            }
            .navigationTitle(isRegistering ? "Register" : "MasterWallet")
        }
    }
}

// MARK: - Create Master Wallet Sheet
struct CreateMasterWalletSheet: View {
    @EnvironmentObject var appState: MasterAppState
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var password = ""
    @State private var chain: Chain = .ethereum
    @State private var isCreating = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Wallet") {
                    TextField("Wallet Name", text: $name)
                    SecureField("Wallet Password", text: $password)
                    Picker("Chain", selection: $chain) {
                        ForEach(Chain.allCases, id: \.self) { c in Text(c.name).tag(c) }
                    }
                }
                if let error = appState.errorMessage {
                    Section { Text(error).font(.caption).foregroundColor(.red) }
                }
            }
            .navigationTitle("Create Master Wallet")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") {
                        Task {
                            isCreating = true
                            do {
                                let wallet = try await appState.apiService.createMasterWallet(
                                    name: name, password: password, chainId: chain.chainId)
                                appState.masterWallet = wallet
                                appState.loadDashboardData(walletId: wallet.id)
                                dismiss()
                            } catch {
                                appState.errorMessage = error.localizedDescription
                            }
                            isCreating = false
                        }
                    }
                    .disabled(name.isEmpty || password.isEmpty || isCreating)
                }
            }
        }
    }
}

// MARK: - Create Sub Wallet Sheet
struct CreateSubWalletSheet: View {
    @EnvironmentObject var appState: MasterAppState
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var password = ""
    @State private var chain: Chain = .ethereum
    @State private var isCreating = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Sub-Wallet") {
                    TextField("Sub-Wallet Name", text: $name)
                    SecureField("Master Wallet Password", text: $password)
                    Picker("Chain", selection: $chain) {
                        ForEach(Chain.allCases, id: \.self) { c in Text(c.name).tag(c) }
                    }
                }
                if let error = appState.errorMessage {
                    Section { Text(error).font(.caption).foregroundColor(.red) }
                }
            }
            .navigationTitle("Add Sub-Wallet")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Add") {
                        guard let masterId = appState.masterWallet?.id else {
                            appState.errorMessage = "No master wallet selected."
                            return
                        }
                        Task {
                            isCreating = true
                            do {
                                _ = try await appState.apiService.createSubWallet(
                                    masterWalletId: masterId,
                                    name: name,
                                    password: password,
                                    chainId: chain.chainId)
                                appState.loadSubWallets(walletId: masterId)
                                dismiss()
                            } catch {
                                appState.errorMessage = error.localizedDescription
                            }
                            isCreating = false
                        }
                    }
                    .disabled(name.isEmpty || password.isEmpty || appState.masterWallet == nil || isCreating)
                }
            }
        }
    }
}
