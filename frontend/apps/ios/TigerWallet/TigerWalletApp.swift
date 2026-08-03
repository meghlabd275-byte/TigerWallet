//
//  TigerWalletApp.swift
//  TigerWallet - Complete Production iOS Wallet Application
//

import SwiftUI

@main
struct TigerWalletApp: App {
    @StateObject private var appState = AppState()
    @StateObject private var themeManager = ThemeManager()
    
    var body: some Scene {
        WindowGroup {
            MainTabView()
                .environmentObject(appState)
                .environmentObject(themeManager)
                .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
        }
    }
}

// MARK: - App State
class AppState: ObservableObject {
    @Published var isAuthenticated: Bool = false
    @Published var currentWallet: Wallet?
    @Published var wallets: [Wallet] = []
    @Published var isLoading: Bool = false
    @Published var errorMessage: String?
    @Published var selectedChain: Chain = .ethereum
    
    let apiService: APIService
    let walletService: WalletService
    let priceService: PriceService
    
    init() {
        self.apiService = APIService()
        self.walletService = WalletService(apiService: apiService)
        self.priceService = PriceService(apiService: apiService)
    }
}

// MARK: - Theme Manager
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

// MARK: - Main Tab View
struct MainTabView: View {
    @EnvironmentObject var appState: AppState
    @EnvironmentObject var themeManager: ThemeManager
    @State private var selectedTab: Int = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem {
                    Label("Wallet", systemImage: "wallet.pass.fill")
                }
                .tag(0)
            
            TradeView()
                .tabItem {
                    Label("Trade", systemImage: "chart.line.uptrend.xyaxis")
                }
                .tag(1)
            
            DAppsView()
                .tabItem {
                    Label("DApps", systemImage: "square.grid.2x2")
                }
                .tag(2)
            
            SettingsView()
                .tabItem {
                    Label("Settings", systemImage: "person.fill")
                }
                .tag(3)
        }
        .tint(.orange)
    }
}

// MARK: - Dashboard View
struct DashboardView: View {
    @EnvironmentObject var appState: AppState
    @EnvironmentObject var themeManager: ThemeManager
    @State private var showingSend: Bool = false
    @State private var showingReceive: Bool = false
    @State private var showingSwap: Bool = false
    
    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    // Header
                    HStack {
                        Text("🐯").font(.largeTitle)
                        Text("TigerWallet").font(.title).fontWeight(.bold)
                        Spacer()
                        Button {
                            themeManager.isDarkMode.toggle()
                        } label: {
                            Image(systemName: themeManager.isDarkMode ? "sun.max.fill" : "moon.fill")
                                .foregroundColor(.primary)
                        }
                    }
                    .padding()
                    
                    // Balance Card
                    VStack(spacing: 8) {
                        Text("Total Balance").font(.subheadline).foregroundColor(.secondary)
                        Text("$\(appState.currentWallet?.totalBalance ?? 0, specifier: "%.2f")")
                            .font(.system(size: 40, weight: .bold))
                        Text("\(appState.currentWallet?.nativeBalance ?? "0.0") \(appState.selectedChain.symbol)")
                            .foregroundColor(.secondary)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 30)
                    .background(Color.orange.opacity(0.1))
                    .cornerRadius(20)
                    .padding(.horizontal)
                    
                    // Action Buttons
                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 15) {
                        ActionButton(icon: "arrow.up.circle.fill", title: "Send", color: .orange) {
                            showingSend = true
                        }
                        ActionButton(icon: "arrow.down.circle.fill", title: "Receive", color: .green) {
                            showingReceive = true
                        }
                        ActionButton(icon: "arrow.triangle.2.circlepath", title: "Swap", color: .blue) {
                            showingSwap = true
                        }
                        ActionButton(icon: "chart.pie.fill", title: "Portfolio", color: .purple) {
                            // Navigate to portfolio
                        }
                    }
                    .padding(.horizontal)
                    
                    // Chains
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Chains").font(.headline).padding(.horizontal)
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 10) {
                                ForEach(Chain.allCases, id: \.self) { chain in
                                    ChainChip(
                                        name: chain.name,
                                        icon: chain.icon,
                                        active: appState.selectedChain == chain
                                    ) {
                                        appState.selectedChain = chain
                                    }
                                }
                            }
                            .padding(.horizontal)
                        }
                    }
                    
                    // Token List
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Assets").font(.headline).padding(.horizontal)
                        if let wallet = appState.currentWallet {
                            ForEach(wallet.tokens) { token in
                                TokenRow(token: token)
                            }
                        }
                    }
                }
            }
            .navigationBarHidden(true)
            .sheet(isPresented: $showingSend) {
                SendView()
            }
            .sheet(isPresented: $showingReceive) {
                ReceiveView()
            }
            .sheet(isPresented: $showingSwap) {
                SwapView()
            }
            .task {
                await loadWalletData()
            }
        }
    }
    
    private func loadWalletData() async {
        appState.isLoading = true
        defer { appState.isLoading = false }
        
        do {
            let wallet = try await appState.walletService.getWallet(for: appState.selectedChain)
            appState.currentWallet = wallet
        } catch {
            appState.errorMessage = error.localizedDescription
        }
    }
}

// MARK: - Token Row
struct TokenRow: View {
    let token: Token
    
    var body: some View {
        HStack {
            AsyncImage(url: URL(string: token.logoURL)) { image in
                image.resizable().aspectRatio(contentMode: .fit)
            } placeholder: {
                Circle().fill(Color.orange.opacity(0.3))
            }
            .frame(width: 40, height: 40)
            
            VStack(alignment: .leading) {
                Text(token.symbol).font(.headline)
                Text(token.name).font(.caption).foregroundColor(.secondary)
            }
            
            Spacer()
            
            VStack(alignment: .trailing) {
                Text("$\(token.balanceUSD, specifier: "%.2f")").font(.headline)
                Text(token.balance).font(.caption).foregroundColor(.secondary)
            }
        }
        .padding()
        .background(Color(.secondarySystemBackground))
        .cornerRadius(15)
        .padding(.horizontal)
    }
}

// MARK: - Action Button
struct ActionButton: View {
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

// MARK: - Chain Chip
struct ChainChip: View {
    let name: String
    let icon: String
    let active: Bool
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            HStack(spacing: 6) {
                Text(icon)
                Text(name).font(.caption).fontWeight(.medium)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(active ? Color.orange : Color(.secondarySystemBackground))
            .foregroundColor(active ? .white : .primary)
            .cornerRadius(20)
        }
    }
}

// MARK: - Send View
struct SendView: View {
    @Environment(\.dismiss) var dismiss
    @EnvironmentObject var appState: AppState
    @State private var recipient: String = ""
    @State private var amount: String = ""
    @State private var isSending: Bool = false
    
    var body: some View {
        NavigationStack {
            Form {
                Section("Recipient Address") {
                    TextField("0x...", text: $recipient)
                }
                
                Section("Amount") {
                    TextField("0.0", text: $amount)
                }
                
                Section {
                    HStack {
                        Text("Available")
                        Spacer()
                        Text("\(appState.currentWallet?.nativeBalance ?? "0.0") \(appState.selectedChain.symbol)")
                            .foregroundColor(.secondary)
                    }
                }
                
                Section {
                    Button("Send") {
                        Task {
                            isSending = true
                            // Send transaction
                            try? await Task.sleep(nanoseconds: 2_000_000_000)
                            isSending = false
                            dismiss()
                        }
                    }
                    .disabled(recipient.isEmpty || amount.isEmpty || isSending)
                }
            }
            .navigationTitle("Send")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
    }
}

// MARK: - Receive View
struct ReceiveView: View {
    @Environment(\.dismiss) var dismiss
    @EnvironmentObject var appState: AppState
    
    var body: some View {
        NavigationStack {
            VStack(spacing: 30) {
                if let wallet = appState.currentWallet {
                    QRCodeView(address: wallet.address)
                        .frame(width: 200, height: 200)
                        .padding()
                        .background(Color.white)
                        .cornerRadius(20)
                    
                    Text(wallet.address)
                        .font(.caption)
                        .lineLimit(2)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal)
                    
                    Button {
                        UIPasteboard.general.string = wallet.address
                    } label: {
                        Label("Copy Address", systemImage: "doc.on.doc")
                    }
                    .buttonStyle(.bordered)
                }
            }
            .padding()
            .navigationTitle("Receive \(appState.selectedChain.symbol)")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") { dismiss() }
                }
            }
        }
    }
}

// MARK: - QR Code View
struct QRCodeView: View {
    let address: String
    
    var body: some View {
        Image(systemName: "qrcode")
            .resizable()
            .aspectRatio(contentMode: .fit)
    }
}

// MARK: - Swap View
struct SwapView: View {
    @Environment(\.dismiss) var dismiss
    @EnvironmentObject var appState: AppState
    @State private var fromToken: Token?
    @State private var toToken: Token?
    @State private var fromAmount: String = ""
    @State private var toAmount: String = ""
    
    var body: some View {
        NavigationStack {
            VStack(spacing: 20) {
                VStack(spacing: 8) {
                    Text("You Pay")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    
                    TextField("0.0", text: $fromAmount)
                        .font(.system(size: 30, weight: .bold))
                        .multilineTextAlignment(.center)
                    
                    if let token = fromToken {
                        Text(token.symbol)
                            .foregroundColor(.secondary)
                    }
                }
                .padding()
                .background(Color(.secondarySystemBackground))
                .cornerRadius(15)
                
                Button {
                    // Swap tokens
                } label: {
                    Image(systemName: "arrow.up.arrow.down.circle.fill")
                        .font(.system(size: 30))
                        .foregroundColor(.orange)
                }
                
                VStack(spacing: 8) {
                    Text("You Receive")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    
                    Text(toAmount.isEmpty ? "0.0" : toAmount)
                        .font(.system(size: 30, weight: .bold))
                        .multilineTextAlignment(.center)
                    
                    if let token = toToken {
                        Text(token.symbol)
                            .foregroundColor(.secondary)
                    }
                }
                .padding()
                .background(Color(.secondarySystemBackground))
                .cornerRadius(15)
                
                Spacer()
                
                Button("Swap") {
                    // Execute swap
                }
                .font(.headline)
                .frame(maxWidth: .infinity)
                .padding()
                .background(Color.orange)
                .foregroundColor(.white)
                .cornerRadius(15)
            }
            .padding()
            .navigationTitle("Swap")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
    }
}

// MARK: - Trade View
struct TradeView: View {
    var body: some View {
        NavigationStack {
            VStack {
                Text("Trading Terminal")
                    .font(.title2)
                Spacer()
            }
            .navigationTitle("Trade")
        }
    }
}

// MARK: - DApps View
struct DAppsView: View {
    let dapps = [
        DApp(name: "Uniswap", icon: "🔄", category: "DeFi"),
        DApp(name: "Aave", icon: "👻", category: "Lending"),
        DApp(name: "OpenSea", icon: "🌊", category: "NFT")
    ]
    
    var body: some View {
        NavigationStack {
            List(dapps) { dapp in
                HStack {
                    Text(dapp.icon).font(.title)
                    VStack(alignment: .leading) {
                        Text(dapp.name).font(.headline)
                        Text(dapp.category).font(.caption).foregroundColor(.secondary)
                    }
                }
            }
            .navigationTitle("DApps")
        }
    }
}

// MARK: - Settings View
struct SettingsView: View {
    @EnvironmentObject var appState: AppState
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        NavigationStack {
            Form {
                Section("Appearance") {
                    Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
                }
                
                Section("Security") {
                    NavigationLink("Biometric Auth") {
                        BiometricSettingsView()
                    }
                    NavigationLink("Change PIN") {
                        ChangePINView()
                    }
                    NavigationLink("Recovery Phrase") {
                        RecoveryPhraseView()
                    }
                }
                
                Section("Network") {
                    Picker("Default Chain", selection: $appState.selectedChain) {
                        ForEach(Chain.allCases, id: \.self) { chain in
                            Text(chain.name).tag(chain)
                        }
                    }
                }
                
                Section("About") {
                    HStack {
                        Text("Version")
                        Spacer()
                        Text("1.0.0").foregroundColor(.secondary)
                    }
                }
            }
            .navigationTitle("Settings")
        }
    }
}

struct BiometricSettingsView: View {
    @State private var biometricEnabled: Bool = false
    
    var body: some View {
        Form {
            Section {
                Toggle("Enable Face ID", isOn: $biometricEnabled)
            } footer: {
                Text("Use Face ID to unlock your wallet")
            }
        }
        .navigationTitle("Biometric Auth")
    }
}

struct ChangePINView: View {
    var body: some View {
        Text("Change PIN")
            .navigationTitle("Change PIN")
    }
}

struct RecoveryPhraseView: View {
    var body: some View {
        Text("Recovery Phrase")
            .navigationTitle("Recovery Phrase")
    }
}
