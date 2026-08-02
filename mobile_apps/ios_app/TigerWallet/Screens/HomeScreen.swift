import SwiftUI

// Home Screen - Main Dashboard
struct HomeScreen: View {
    @State private var selectedTab: String = "Wallet"
    @State private var totalBalance: Double = 12450.00
    
    let tabs = ["Wallet", "DApps", "Activity"]
    
    let tokens = [
        TokenItem(symbol: "ETH", name: "Ethereum", balance: 4.2, value: 8400.00, icon: "◈"),
        TokenItem(symbol: "USDT", name: "Tether", balance: 2500.00, value: 2500.00, icon: "₮"),
        TokenItem(symbol: "BNB", name: "BNB", balance: 1.5, value: 450.00, icon: "🟡"),
        TokenItem(symbol: "SOL", name: "Solana", balance: 22.0, value: 1100.00, icon: "☀️")
    ]
    
    var body: some View {
        TabView(selection: $selectedTab) {
            walletView
                .tabItem { Label("Wallet", systemImage: "wallet.pass") }
                .tag("Wallet")
            
            dappsView
                .tabItem { Label("DApps", systemImage: "app.badge") }
                .tag("DApps")
            
            activityView
                .tabItem { Label("Activity", systemImage: "clock.arrow.circlepath") }
                .tag("Activity")
        }
    }
    
    var walletView: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 20) {
                    // Balance Card
                    balanceCard
                    
                    // Quick Actions
                    quickActions
                    
                    // Token List
                    tokenList
                }
                .padding()
            }
            .navigationTitle("TigerWallet")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: {}) {
                        Image(systemName: "bell")
                    }
                }
            }
        }
    }
    
    var balanceCard: some View {
        VStack(spacing: 16) {
            VStack(spacing: 4) {
                Text("Total Balance")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                Text("$\(totalBalance, specifier: "%.2f")")
                    .font(.system(size: 36, weight: .bold))
            }
            
            HStack(spacing: 20) {
                Button(action: {}) {
                    HStack {
                        Image(systemName: "arrow.up.circle.fill")
                        Text("Send")
                    }
                    .padding(.horizontal, 20)
                    .padding(.vertical, 10)
                    .background(Color.orange)
                    .foregroundColor(.white)
                    .cornerRadius(20)
                }
                
                Button(action: {}) {
                    HStack {
                        Image(systemName: "arrow.down.circle.fill")
                        Text("Receive")
                    }
                    .padding(.horizontal, 20)
                    .padding(.vertical, 10)
                    .background(Color.gray.opacity(0.2))
                    .foregroundColor(.primary)
                    .cornerRadius(20)
                }
            }
        }
        .padding(24)
        .background(
            LinearGradient(
                gradient: Gradient(colors: [Color.orange.opacity(0.8), Color.orange]),
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        )
        .foregroundColor(.white)
        .cornerRadius(16)
    }
    
    var quickActions: some View {
        HStack(spacing: 16) {
            quickActionButton(icon: "arrow.triangle.2.circlepath", label: "Swap", color: .blue)
            quickActionButton(icon: "bridge.fill", label: "Bridge", color: .purple)
            quickActionButton(icon: "chart.line.uptrend.xyaxis", label: "Stake", color: .green)
            quickActionButton(icon: "photo.on.rectangle", label: "NFTs", color: .pink)
        }
    }
    
    func quickActionButton(icon: String, label: String, color: Color) -> some View {
        VStack(spacing: 8) {
            Button(action: {}) {
                Image(systemName: icon)
                    .font(.title2)
                    .frame(width: 50, height: 50)
                    .background(color.opacity(0.2))
                    .foregroundColor(color)
                    .cornerRadius(12)
            }
            Text(label)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }
    
    var tokenList: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("Assets")
                    .font(.headline)
                Spacer()
                Button(action: {}) {
                    Text("Add Token")
                        .font(.caption)
                        .foregroundColor(.orange)
                }
            }
            
            ForEach(tokens) { token in
                tokenRow(token: token)
            }
        }
    }
    
    func tokenRow(token: TokenItem) -> some View {
        HStack(spacing: 12) {
            // Icon
            ZStack {
                Circle()
                    .fill(Color.gray.opacity(0.2))
                    .frame(width: 44, height: 44)
                Text(token.icon)
                    .font(.title3)
            }
            
            // Name & Symbol
            VStack(alignment: .leading, spacing: 2) {
                Text(token.name)
                    .font(.subheadline)
                    .fontWeight(.medium)
                Text(token.symbol)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            // Balance & Value
            VStack(alignment: .trailing, spacing: 2) {
                Text("\(token.balance, specifier: "%.4f")")
                    .font(.subheadline)
                    .fontWeight(.medium)
                Text("$\(token.value, specifier: "%.2f")")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(12)
    }
    
    var dappsView: some View {
        NavigationView {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    // Featured DApps
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Featured")
                            .font(.headline)
                        
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 12) {
                                dappCard(name: "Uniswap", category: "DeFi", icon: "🦄")
                                dappCard(name: "OpenSea", category: "NFT", icon: "🌊")
                                dappCard(name: "Aave", category: "Lending", icon: "👻")
                            }
                        }
                    }
                    
                    // All DApps
                    VStack(alignment: .leading, spacing: 12) {
                        Text("All DApps")
                            .font(.headline)
                        
                        ForEach(0..<10, id: \.self) { _ in
                            dappListItem()
                        }
                    }
                }
                .padding()
            }
            .navigationTitle("DApps")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: {}) {
                        Image(systemName: "magnifyingglass")
                    }
                }
            }
        }
    }
    
    func dappCard(name: String, category: String, icon: String) -> some View {
        VStack {
            ZStack {
                RoundedRectangle(cornerRadius: 12)
                    .fill(Color.gray.opacity(0.2))
                    .frame(width: 100, height: 100)
                Text(icon)
                    .font(.largeTitle)
            }
            Text(name)
                .font(.subheadline)
                .fontWeight(.medium)
            Text(category)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }
    
    func dappListItem() -> some View {
        HStack(spacing: 12) {
            ZStack {
                Circle()
                    .fill(Color.gray.opacity(0.2))
                    .frame(width: 44, height: 44)
                Text("🪙")
            }
            
            VStack(alignment: .leading) {
                Text("DEX Protocol")
                    .font(.subheadline)
                Text("DeFi")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            Button(action: {}) {
                Text("Connect")
                    .font(.caption)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
                    .background(Color.orange)
                    .foregroundColor(.white)
                    .cornerRadius(8)
            }
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(12)
    }
    
    var activityView: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 12) {
                    ForEach(0..<20, id: \.self) { _ in
                        activityItem()
                    }
                }
                .padding()
            }
            .navigationTitle("Activity")
        }
    }
    
    func activityItem() -> some View {
        HStack(spacing: 12) {
            Image(systemName: "arrow.up.circle.fill")
                .font(.title2)
                .foregroundColor(.orange)
            
            VStack(alignment: .leading, spacing: 2) {
                Text("Sent ETH")
                    .font(.subheadline)
                Text("To: 0x742d...f123")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            VStack(alignment: .trailing, spacing: 2) {
                Text("-0.5 ETH")
                    .font(.subheadline)
                    .foregroundColor(.red)
                Text("Just now")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(12)
    }
}

struct TokenItem: Identifiable {
    let id = UUID()
    let symbol: String
    let name: String
    let balance: Double
    let value: Double
    let icon: String
}

#Preview {
    HomeScreen()
}
