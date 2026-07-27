import SwiftUI

// MARK: - Wallet Home View

struct WalletHomeView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @EnvironmentObject var walletStore: WalletStore
    @EnvironmentObject var networkManager: NetworkManager
    @State private var showReceive = false
    @State private var showSend = false
    @State private var showQRScanner = false
    
    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 20) {
                    // Network Selector
                    networkSelector
                    
                    // Balance Card
                    balanceCard
                    
                    // Quick Actions
                    quickActions
                    
                    // Token List
                    tokenList
                }
                .padding()
            }
            .navigationTitle("Wallet")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    themeToggle
                }
            }
            .sheet(isPresented: $showReceive) {
                ReceiveView()
            }
            .sheet(isPresented: $showSend) {
                SendView()
            }
        }
    }
    
    private var networkSelector: some View {
        Menu {
            ForEach(networkManager.networks) { network in
                Button(action: {
                    networkManager.selectedNetwork = network
                }) {
                    HStack {
                        Text(network.name)
                        if networkManager.selectedNetwork.id == network.id {
                            Image(systemName: "checkmark")
                        }
                    }
                }
            }
        } label: {
            HStack {
                Image(systemName: "network")
                Text(networkManager.selectedNetwork.name)
                Image(systemName: "chevron.down")
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(Color.secondary.opacity(0.1))
            .cornerRadius(8)
        }
    }
    
    private var balanceCard: some View {
        VStack(spacing: 16) {
            VStack(spacing: 4) {
                Text("Total Balance")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                
                Text(totalBalance)
                    .font(.system(size: 36, weight: .bold))
                
                Text(change24h)
                    .font(.subheadline)
                    .foregroundColor(changeColor)
            }
            
            HStack(spacing: 16) {
                ActionButton(title: "Send", icon: "arrow.up.circle.fill") {
                    showSend = true
                }
                
                ActionButton(title: "Receive", icon: "arrow.down.circle.fill") {
                    showReceive = true
                }
                
                ActionButton(title: "Buy", icon: "creditcard.fill") {
                    // Show buy crypto sheet
                }
                
                ActionButton(title: "Swap", icon: "arrow.left.arrow.right.circle.fill") {
                    // Navigate to swap
                }
            }
        }
        .padding(20)
        .background(
            LinearGradient(
                gradient: Gradient(colors: [Color.orange.opacity(0.8), Color.orange]),
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        )
        .foregroundColor(.white)
        .cornerRadius(20)
    }
    
    private var quickActions: some View {
        LazyVGrid(columns: [
            GridItem(.flexible()),
            GridItem(.flexible()),
            GridItem(.flexible())
        ], spacing: 16) {
            QuickActionButton(icon: "doc.on.doc", title: "Copy Address") {
                UIPasteboard.general.string = walletStore.currentWallet?.address ?? ""
            }
            
            QuickActionButton(icon: "qrcode.viewfinder", title: "Scan QR") {
                showQRScanner = true
            }
            
            QuickActionButton(icon: "arrow.triangle.2.circlepath", title: "Refresh") {
                Task {
                    await walletStore.fetchBalances()
                }
            }
        }
    }
    
    private var tokenList: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("Assets")
                    .font(.headline)
                Spacer()
                Button("See All") {
                    // Navigate to full list
                }
            }
            
            if walletStore.tokens.isEmpty {
                Text("No assets yet")
                    .foregroundColor(.secondary)
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding()
            } else {
                ForEach(walletStore.tokens.prefix(5)) { token in
                    TokenRow(token: token)
                }
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(16)
    }
    
    private var themeToggle: some View {
        Button(action: {
            themeManager.toggleTheme()
        }) {
            Image(systemName: themeManager.isDarkMode ? "sun.max.fill" : "moon.fill")
        }
    }
    
    private var totalBalance: String {
        let total = walletStore.tokens.reduce(0) { $0 + $1.usdValue }
        return String(format: "$%.2f", total)
    }
    
    private var change24h: String {
        return "+$123.45 (2.3%)"
    }
    
    private var changeColor: Color {
        return .green
    }
}

// MARK: - Supporting Views

struct ActionButton: View {
    let title: String
    let icon: String
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            VStack(spacing: 8) {
                Image(systemName: icon)
                    .font(.title2)
                Text(title)
                    .font(.caption)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 12)
            .background(Color.white.opacity(0.2))
            .cornerRadius(12)
        }
        .foregroundColor(.white)
    }
}

struct QuickActionButton: View {
    let icon: String
    let title: String
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            VStack(spacing: 8) {
                Image(systemName: icon)
                    .font(.title3)
                Text(title)
                    .font(.caption)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 12)
            .background(Color(.secondarySystemBackground))
            .cornerRadius(12)
        }
        .foregroundColor(.primary)
    }
}

struct TokenRow: View {
    let token: TokenBalance
    
    var body: some View {
        HStack {
            // Token Icon
            ZStack {
                Circle()
                    .fill(Color.orange.opacity(0.2))
                    .frame(width: 44, height: 44)
                Text(token.symbol.prefix(1))
                    .font(.headline)
                    .foregroundColor(.orange)
            }
            
            VStack(alignment: .leading, spacing: 2) {
                Text(token.name)
                    .font(.headline)
                Text(token.symbol)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            VStack(alignment: .trailing, spacing: 2) {
                Text(String(format: "%.4f", token.balance))
                    .font(.headline)
                Text(String(format: "$%.2f", token.usdValue))
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .padding(.vertical, 8)
    }
}

// MARK: - Receive View

struct ReceiveView: View {
    @EnvironmentObject var walletStore: WalletStore
    @EnvironmentObject var networkManager: NetworkManager
    @Environment(\.dismiss) var dismiss
    
    var body: some View {
        NavigationStack {
            VStack(spacing: 30) {
                Text("Receive \(networkManager.selectedNetwork.symbol)")
                    .font(.headline)
                
                // QR Code placeholder
                ZStack {
                    RoundedRectangle(cornerRadius: 16)
                        .fill(Color.white)
                        .frame(width: 250, height: 250)
                    if let address = walletStore.currentWallet?.chainAddresses[String(networkManager.selectedNetwork.chainId)] {
                        QRCodeView(data: address)
                            .frame(width: 200, height: 200)
                    }
                }
                
                // Address
                VStack(spacing: 8) {
                    Text("Wallet Address")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                    
                    if let address = walletStore.currentWallet?.chainAddresses[String(networkManager.selectedNetwork.chainId)] {
                        HStack {
                            Text(address)
                                .font(.system(.body, design: .monospaced))
                                .lineLimit(2)
                            
                            Button(action: {
                                UIPasteboard.general.string = address
                            }) {
                                Image(systemName: "doc.on.doc")
                            }
                        }
                        .padding()
                        .background(Color(.secondarySystemBackground))
                        .cornerRadius(12)
                    }
                }
                
                Spacer()
            }
            .padding()
            .navigationTitle("Receive")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
        }
    }
}

// MARK: - Send View

struct SendView: View {
    @EnvironmentObject var walletStore: WalletStore
    @EnvironmentObject var networkManager: NetworkManager
    @Environment(\.dismiss) var dismiss
    
    @State private var recipient = ""
    @State private var amount = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    
    var body: some View {
        NavigationStack {
            VStack(spacing: 20) {
                // Recipient
                VStack(alignment: .leading, spacing: 8) {
                    Text("Recipient Address")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                    
                    TextField("Enter address or scan QR", text: $recipient)
                        .textFieldStyle(.roundedBorder)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }
                
                // Amount
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        Text("Amount")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                        Spacer()
                        Button("Max") {
                            // Set max amount
                        }
                        .font(.subheadline)
                    }
                    
                    HStack {
                        TextField("0.0", text: $amount)
                            .keyboardType(.decimalPad)
                            .font(.title)
                        
                        Text(networkManager.selectedNetwork.symbol)
                            .font(.headline)
                    }
                    .padding()
                    .background(Color(.secondarySystemBackground))
                    .cornerRadius(12)
                }
                
                // Error
                if let error = errorMessage {
                    Text(error)
                        .foregroundColor(.red)
                        .font(.subheadline)
                }
                
                Spacer()
                
                // Send Button
                Button(action: sendTransaction) {
                    if isLoading {
                        ProgressView()
                            .progressViewStyle(.linear)
                            .frame(maxWidth: .infinity)
                    } else {
                        Text("Send \(networkManager.selectedNetwork.symbol)")
                            .font(.headline)
                            .frame(maxWidth: .infinity)
                    }
                }
                .disabled(recipient.isEmpty || amount.isEmpty || isLoading)
                .padding()
                .background(recipient.isEmpty || amount.isEmpty ? Color.gray : Color.orange)
                .foregroundColor(.white)
                .cornerRadius(16)
            }
            .padding()
            .navigationTitle("Send")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") {
                        dismiss()
                    }
                }
            }
        }
    }
    
    private func sendTransaction() {
        guard let amountDouble = Double(amount) else {
            errorMessage = "Invalid amount"
            return
        }
        
        isLoading = true
        errorMessage = nil
        
        Task {
            do {
                let _ = try await walletStore.sendTransaction(
                    to: recipient,
                    amount: amountDouble,
                    chainId: networkManager.selectedNetwork.chainId
                )
                await MainActor.run {
                    dismiss()
                }
            } catch {
                await MainActor.run {
                    errorMessage = error.localizedDescription
                    isLoading = false
                }
            }
        }
    }
}

// MARK: - QR Code View

struct QRCodeView: View {
    let data: String
    
    var body: some View {
        // Simplified QR code representation
        // In production, use a QR code library
        ZStack {
            Color.white
            Image(systemName: "qrcode")
                .font(.system(size: 100))
                .foregroundColor(.black)
        }
    }
}

// MARK: - Preview

struct WalletHomeView_Previews: PreviewProvider {
    static var previews: some View {
        WalletHomeView()
            .environmentObject(WalletStore.shared)
            .environmentObject(ThemeManager.shared)
            .environmentObject(NetworkManager.shared)
    }
}
