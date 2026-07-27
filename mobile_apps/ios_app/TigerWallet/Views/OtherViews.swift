import SwiftUI

// MARK: - Swap View

struct SwapView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @EnvironmentObject var networkManager: NetworkManager
    @State private var fromToken = "ETH"
    @State private var toToken = "USDT"
    @State private var fromAmount = ""
    @State private var toAmount = ""
    @State private var isLoading = false
    @State private var slippage = 0.5
    
    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    // From Token
                    tokenInput(
                        title: "From",
                        token: $fromToken,
                        amount: $fromAmount,
                        isEditable: true
                    )
                    
                    // Swap Button
                    Button(action: {
                        // Swap tokens
                        let temp = fromToken
                        fromToken = toToken
                        toToken = temp
                    }) {
                        Image(systemName: "arrow.up.arrow.down.circle.fill")
                            .font(.title)
                            .foregroundColor(.orange)
                    }
                    
                    // To Token
                    tokenInput(
                        title: "To",
                        token: $toToken,
                        amount: $toAmount,
                        isEditable: false
                    )
                    
                    // Exchange Rate
                    if !fromAmount.isEmpty && !toAmount.isEmpty {
                        HStack {
                            Text("Rate")
                                .foregroundColor(.secondary)
                            Spacer()
                            Text("1 \(fromToken) ≈ \(exchangeRate) \(toToken)")
                        }
                        .font(.subheadline)
                        .padding()
                    }
                    
                    // Slippage Settings
                    slippageSettings
                    
                    // Swap Button
                    Button(action: executeSwap) {
                        if isLoading {
                            ProgressView()
                                .progressViewStyle(.linear)
                                .frame(maxWidth: .infinity)
                        } else {
                            Text("Swap")
                                .font(.headline)
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .disabled(fromAmount.isEmpty || isLoading)
                    .padding()
                    .background(fromAmount.isEmpty ? Color.gray : Color.orange)
                    .foregroundColor(.white)
                    .cornerRadius(16)
                }
                .padding()
            }
            .navigationTitle("Swap")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: {
                        themeManager.toggleTheme()
                    }) {
                        Image(systemName: themeManager.isDarkMode ? "sun.max.fill" : "moon.fill")
                    }
                }
            }
        }
    }
    
    private func tokenInput(title: String, token: Binding<String>, amount: Binding<String>, isEditable: Bool) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    if isEditable {
                        TextField("0.0", text: amount)
                            .keyboardType(.decimalPad)
                            .font(.title2)
                    } else {
                        Text(amount.wrappedValue.isEmpty ? "0.0" : amount.wrappedValue)
                            .font(.title2)
                    }
                    
                    Text("$0.00")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                
                Spacer()
                
                // Token Selector
                Menu {
                    ForEach(tokens, id: \.self) { t in
                        Button(action: {
                            token.wrappedValue = t
                        }) {
                            Text(t)
                        }
                    }
                } label: {
                    HStack {
                        Text(token.wrappedValue)
                        Image(systemName: "chevron.down")
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 8)
                    .background(Color(.secondarySystemBackground))
                    .cornerRadius(8)
                }
            }
            .padding()
            .background(Color(.secondarySystemBackground))
            .cornerRadius(16)
        }
    }
    
    private var slippageSettings: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Slippage Tolerance")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                Spacer()
                Text("\(slippage)%")
                    .font(.subheadline)
            }
            
            HStack(spacing: 8) {
                ForEach([0.1, 0.5, 1.0], id: \.self) { value in
                    Button(action: { slippage = value }) {
                        Text("\(value)%")
                            .font(.subheadline)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 8)
                            .background(slippage == value ? Color.orange : Color(.secondarySystemBackground))
                            .foregroundColor(slippage == value ? .white : .primary)
                            .cornerRadius(8)
                    }
                }
                
                // Custom slippage
                TextField("Custom", value: $slippage, format: .number)
                    .keyboardType(.decimalPad)
                    .textFieldStyle(.roundedBorder)
                    .frame(width: 80)
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(16)
    }
    
    private var tokens: [String] {
        ["ETH", "USDT", "USDC", "BNB", "MATIC", "AVAX", "SOL", "TRX"]
    }
    
    private var exchangeRate: String {
        guard let amount = Double(fromAmount) else { return "0" }
        return String(format: "%.4f", amount * 1.05) // Simplified
    }
    
    private func executeSwap() {
        isLoading = true
        // Implement swap logic
        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
            isLoading = false
        }
    }
}

// MARK: - Portfolio View

struct PortfolioView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @EnvironmentObject var walletStore: WalletStore
    @State private var selectedPeriod: Period = .day
    
    enum Period: String, CaseIterable {
        case day = "24H"
        case week = "7D"
        case month = "1M"
        case year = "1Y"
        case all = "ALL"
    }
    
    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    // Total Balance
                    VStack(spacing: 8) {
                        Text("Total Balance")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                        
                        Text(totalBalance)
                            .font(.system(size: 36, weight: .bold))
                        
                        HStack {
                            Image(systemName: "arrow.up.right")
                                .foregroundColor(.green)
                            Text("+$1,234 (2.3%)")
                                .foregroundColor(.green)
                        }
                        .font(.subheadline)
                    }
                    
                    // Period Selector
                    HStack(spacing: 8) {
                        ForEach(Period.allCases, id: \.self) { period in
                            Button(action: { selectedPeriod = period }) {
                                Text(period.rawValue)
                                    .font(.subheadline)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 8)
                                    .background(selectedPeriod == period ? Color.orange : Color(.secondarySystemBackground))
                                    .foregroundColor(selectedPeriod == period ? .white : .primary)
                                    .cornerRadius(8)
                            }
                        }
                    }
                    
                    // Chart placeholder
                    chartPlaceholder
                    
                    // Asset Allocation
                    assetAllocation
                    
                    // Recent Transactions
                    recentTransactions
                }
                .padding()
            }
            .navigationTitle("Portfolio")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: {
                        themeManager.toggleTheme()
                    }) {
                        Image(systemName: themeManager.isDarkMode ? "sun.max.fill" : "moon.fill")
                    }
                }
            }
        }
    }
    
    private var chartPlaceholder: some View {
        VStack {
            // Simplified chart - in production use Charts framework
            GeometryReader { geometry in
                Path { path in
                    let width = geometry.size.width
                    let height = geometry.size.height
                    
                    path.move(to: CGPoint(x: 0, y: height * 0.8))
                    path.addLine(to: CGPoint(x: width * 0.2, y: height * 0.6))
                    path.addLine(to: CGPoint(x: width * 0.4, y: height * 0.7))
                    path.addLine(to: CGPoint(x: width * 0.6, y: height * 0.4))
                    path.addLine(to: CGPoint(x: width * 0.8, y: height * 0.5))
                    path.addLine(to: CGPoint(x: width, y: height * 0.3))
                }
                .stroke(Color.orange, lineWidth: 2)
            }
            .frame(height: 200)
        }
        .padding()
        .background(Color(.secondarySystemBackground))
        .cornerRadius(16)
    }
    
    private var assetAllocation: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Asset Allocation")
                .font(.headline)
            
            ForEach(walletStore.tokens.prefix(5)) { token in
                HStack {
                    Circle()
                        .fill(Color.orange.opacity(0.5))
                        .frame(width: 12, height: 12)
                    
                    Text(token.name)
                        .font(.subheadline)
                    
                    Spacer()
                    
                    Text(String(format: "%.1f%%", percentage(token)))
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(16)
    }
    
    private var recentTransactions: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("Recent Transactions")
                    .font(.headline)
                Spacer()
                Button("See All") { }
            }
            
            if walletStore.transactions.isEmpty {
                Text("No transactions yet")
                    .foregroundColor(.secondary)
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding()
            } else {
                ForEach(walletStore.transactions.prefix(5)) { tx in
                    TransactionRow(transaction: tx)
                }
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(16)
    }
    
    private var totalBalance: String {
        let total = walletStore.tokens.reduce(0) { $0 + $1.usdValue }
        return String(format: "$%.2f", total)
    }
    
    private func percentage(_ token: TokenBalance) -> Double {
        let total = walletStore.tokens.reduce(0) { $0 + $1.usdValue }
        guard total > 0 else { return 0 }
        return (token.usdValue / total) * 100
    }
}

struct TransactionRow: View {
    let transaction: Transaction
    
    var body: some View {
        HStack {
            Image(systemName: icon)
                .foregroundColor(iconColor)
                .frame(width: 40, height: 40)
                .background(iconColor.opacity(0.1))
                .cornerRadius(20)
            
            VStack(alignment: .leading, spacing: 2) {
                Text(transaction.type.rawValue.capitalized)
                    .font(.subheadline)
                Text(formattedDate)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            VStack(alignment: .trailing, spacing: 2) {
                Text("\(transaction.type == .send ? "-" : "+")\(String(format: "%.4f", transaction.amount))")
                    .font(.subheadline)
                Text(transaction.status.rawValue.capitalized)
                    .font(.caption)
                    .foregroundColor(statusColor)
            }
        }
    }
    
    private var icon: String {
        switch transaction.type {
        case .send: return "arrow.up.circle"
        case .receive: return "arrow.down.circle"
        case .swap: return "arrow.left.arrow.right.circle"
        case .stake: return "lock.circle"
        case .unstake: return "lock.open.circle"
        default: return "circle"
        }
    }
    
    private var iconColor: Color {
        switch transaction.type {
        case .send: return .red
        case .receive: return .green
        case .swap: return .blue
        case .stake, .unstake: return .orange
        default: return .gray
        }
    }
    
    private var statusColor: Color {
        switch transaction.status {
        case .confirmed: return .green
        case .pending: return .orange
        case .failed: return .red
        }
    }
    
    private var formattedDate: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .short
        return formatter.string(from: transaction.timestamp)
    }
}

// MARK: - DApp Browser View

struct DAppBrowserView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @EnvironmentObject var walletStore: WalletStore
    @State private var url = ""
    @State private var currentURL = "https://app.uniswap.org"
    @State private var showURLBar = false
    
    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // URL Bar
                if showURLBar {
                    HStack {
                        TextField("Enter URL", text: $url)
                            .textFieldStyle(.roundedBorder)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                        
                        Button("Go") {
                            currentURL = url.hasPrefix("http") ? url : "https://\(url)"
                            showURLBar = false
                        }
                    }
                    .padding()
                    .background(Color(.secondarySystemBackground))
                }
                
                // WebView placeholder
                VStack {
                    Spacer()
                    VStack(spacing: 16) {
                        Image(systemName: "globe")
                            .font(.system(size: 60))
                            .foregroundColor(.orange)
                        Text("DApp Browser")
                            .font(.title2)
                        Text(currentURL)
                            .font(.caption)
                            .foregroundColor(.secondary)
                        Text("Connect to Web3 dApps securely")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                            .multilineTextAlignment(.center)
                    }
                    .padding()
                    Spacer()
                }
                
                // Bottom toolbar
                HStack(spacing: 40) {
                    Button(action: { currentURL = "https://app.uniswap.org" }) {
                        Image(systemName: "arrow.left")
                    }
                    
                    Button(action: { currentURL = "https://app.uniswap.org" }) {
                        Image(systemName: "arrow.clockwise")
                    }
                    
                    Button(action: { showURLBar.toggle() }) {
                        Image(systemName: "magnifyingglass")
                    }
                    
                    Button(action: {}) {
                        Image(systemName: "square.and.arrow.up")
                    }
                    
                    Button(action: {}) {
                        Image(systemName: "house")
                    }
                }
                .padding()
                .background(Color(.secondarySystemBackground))
            }
            .navigationTitle("Browser")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: {
                        themeManager.toggleTheme()
                    }) {
                        Image(systemName: themeManager.isDarkMode ? "sun.max.fill" : "moon.fill")
                    }
                }
            }
        }
    }
}

// MARK: - Settings View

struct SettingsView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @EnvironmentObject var walletStore: WalletStore
    @State private var showSecuritySettings = false
    @State private var showNetworkSettings = false
    
    var body: some View {
        NavigationStack {
            List {
                // Account Section
                Section("Account") {
                    if let wallet = walletStore.currentWallet {
                        HStack {
                            Image(systemName: "person.circle.fill")
                                .font(.title)
                                .foregroundColor(.orange)
                            VStack(alignment: .leading) {
                                Text(wallet.name)
                                    .font(.headline)
                                Text(wallet.address.prefix(10) + "..." + wallet.address.suffix(4))
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                        }
                    }
                    
                    Button(action: {}) {
                        Label("Switch Wallet", systemImage: "arrow.left.arrow.right")
                    }
                    
                    Button(action: {}) {
                        Label("Backup Wallet", systemImage: "icloud.and.arrow.up")
                    }
                }
                
                // Preferences Section
                Section("Preferences") {
                    Toggle(isOn: $themeManager.isDarkMode) {
                        Label("Dark Mode", systemImage: themeManager.isDarkMode ? "moon.fill" : "sun.max.fill")
                    }
                    
                    Button(action: { showNetworkSettings = true }) {
                        Label("Networks", systemImage: "network")
                    }
                    
                    Button(action: {}) {
                        Label("Currency", systemImage: "dollarsign.circle")
                    }
                }
                
                // Security Section
                Section("Security") {
                    Button(action: { showSecuritySettings = true }) {
                        Label("Security Settings", systemImage: "lock.shield")
                    }
                    
                    Button(action: {}) {
                        Label("Biometric Authentication", systemImage: "faceid")
                    }
                    
                    Button(action: {}) {
                        Label("Auto-Lock", systemImage: "timer")
                    }
                }
                
                // Support Section
                Section("Support") {
                    Button(action: {}) {
                        Label("Help Center", systemImage: "questionmark.circle")
                    }
                    
                    Button(action: {}) {
                        Label("Contact Us", systemImage: "envelope")
                    }
                    
                    Button(action: {}) {
                        Label("About", systemImage: "info.circle")
                    }
                }
                
                // Danger Zone
                Section {
                    Button(role: .destructive, action: {}) {
                        Label("Reset Wallet", systemImage: "trash")
                    }
                }
            }
            .navigationTitle("Settings")
        }
    }
}

// MARK: - Previews

struct SwapView_Previews: PreviewProvider {
    static var previews: some View {
        SwapView()
            .environmentObject(WalletStore.shared)
            .environmentObject(ThemeManager.shared)
            .environmentObject(NetworkManager.shared)
    }
}

struct PortfolioView_Previews: PreviewProvider {
    static var previews: some View {
        PortfolioView()
            .environmentObject(WalletStore.shared)
            .environmentObject(ThemeManager.shared)
    }
}

struct SettingsView_Previews: PreviewProvider {
    static var previews: some View {
        SettingsView()
            .environmentObject(WalletStore.shared)
            .environmentObject(ThemeManager.shared)
    }
}
