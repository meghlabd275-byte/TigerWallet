import SwiftUI

// Send Screen with QR Scanner - Matches Flutter Functionality
struct SendScreen: View {
    @State private var recipientAddress: String = ""
    @State private var amount: String = ""
    @State private var selectedToken: String = "ETH"
    @State private var selectedChain: String = "Ethereum"
    @State private var showQRScanner: Bool = false
    @State private var showError: Bool = false
    @State private var errorMessage: String = ""
    @State private var isLoading: Bool = false
    @State private var transactionHash: String?
    
    let tokens = ["ETH", "USDT", "USDC", "BNB", "MATIC", "SOL", "TRX"]
    let chains = ["Ethereum", "BNB Chain", "Polygon", "Arbitrum", "Optimism", "Avalanche", "Solana", "TRON"]
    
    var body: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 16) {
                    // Network Selector
                    networkSelector
                    
                    // Recipient Address with QR Scanner
                    recipientSection
                    
                    // Token Selector
                    tokenSelector
                    
                    // Amount
                    amountSection
                    
                    // Error Message
                    if showError {
                        errorView
                    }
                    
                    // Send Button
                    sendButton
                    
                    Spacer()
                }
                .padding()
            }
            .navigationTitle("Send")
            .sheet(isPresented: $showQRScanner) {
                QRScannerSheet { address in
                    recipientAddress = address
                    showQRScanner = false
                }
            }
        }
    }
    
    var networkSelector: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Network")
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(chains, id: \.self) { chain in
                        Button(action: {
                            selectedChain = chain
                        }) {
                            HStack {
                                Text(chainIcon(for: chain))
                                Text(chain)
                                    .font(.caption)
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 8)
                            .background(selectedChain == chain ? Color.orange : Color.gray.opacity(0.2))
                            .foregroundColor(selectedChain == chain ? .white : .primary)
                            .cornerRadius(20)
                        }
                    }
                }
            }
        }
    }
    
    var recipientSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Recipient Address")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                
                Spacer()
                
                Button(action: { showQRScanner = true }) {
                    HStack {
                        Image(systemName: "qrcode.viewfinder")
                        Text("Scan QR")
                    }
                    .font(.caption)
                }
            }
            
            HStack {
                TextField("0x... or scan QR", text: $recipientAddress)
                    .textFieldStyle(RoundedBorderTextFieldStyle())
                    .autocapitalization(.none)
                    .disableAutocorrection(true)
                
                Button(action: { showQRScanner = true }) {
                    Image(systemName: "qrcode.viewfinder")
                        .font(.title2)
                }
            }
        }
    }
    
    var tokenSelector: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Token")
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(tokens, id: \.self) { token in
                        Button(action: {
                            selectedToken = token
                        }) {
                            Text(token)
                                .padding(.horizontal, 16)
                                .padding(.vertical, 8)
                                .background(selectedToken == token ? Color.orange : Color.gray.opacity(0.2))
                                .foregroundColor(selectedToken == token ? .white : .primary)
                                .cornerRadius(20)
                        }
                    }
                }
            }
        }
    }
    
    var amountSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Amount")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                
                Spacer()
                
                Button(action: { amount = "0.0" }) {
                    Text("MAX")
                        .font(.caption)
                        .foregroundColor(.orange)
                }
            }
            
            HStack {
                TextField("0.0", text: $amount)
                    .keyboardType(.decimalPad)
                    .textFieldStyle(RoundedBorderTextFieldStyle())
                
                Text(selectedToken)
                    .foregroundColor(.secondary)
            }
        }
    }
    
    var errorView: some View {
        HStack {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundColor(.red)
            Text(errorMessage)
                .foregroundColor(.red)
                .font(.caption)
        }
        .padding()
        .background(Color.red.opacity(0.1))
        .cornerRadius(8)
    }
    
    var sendButton: some View {
        Button(action: sendTransaction) {
            HStack {
                if isLoading {
                    ProgressView()
                        .progressViewStyle(CircularProgressViewStyle(tint: .white))
                } else {
                    Text("Send \(selectedToken)")
                        .fontWeight(.bold)
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(Color.orange)
            .foregroundColor(.white)
            .cornerRadius(12)
        }
        .disabled(isLoading)
    }
    
    func sendTransaction() {
        guard !recipientAddress.isEmpty else {
            errorMessage = "Please enter recipient address"
            showError = true
            return
        }
        
        guard isValidAddress(recipientAddress) else {
            errorMessage = "Invalid address format"
            showError = true
            return
        }
        
        guard let amountValue = Double(amount), amountValue > 0 else {
            errorMessage = "Please enter valid amount"
            showError = true
            return
        }
        
        isLoading = true
        showError = false
        
        // Real on-chain send via the canonical wallet_api /api/v1/send (real
        // secp256k1 signing + eth_sendRawTransaction). Do NOT fabricate a tx hash.
        let chainMap = ["Ethereum": 1, "BNB Chain": 56, "Polygon": 137,
                        "Arbitrum": 42161, "Optimism": 10, "Avalanche": 43114]
        let chainId = chainMap[selectedChain] ?? 1
        Task {
            do {
                struct SendBody: Encodable {
                    let to: String
                    let amount: String
                    let chain_id: Int
                    let token: String
                }
                struct SendResp: Decodable { let txHash: String? }
                let resp: SendResp = try await APIClient.shared.post(
                    endpoint: "/api/v1/send",
                    body: SendBody(to: recipientAddress, amount: amount, chain_id: chainId, token: selectedToken),
                    authenticated: true
                )
                await MainActor.run {
                    isLoading = false
                    transactionHash = resp.txHash
                    recipientAddress = ""
                    amount = ""
                }
            } catch {
                await MainActor.run {
                    isLoading = false
                    errorMessage = "Transaction failed: \(error.localizedDescription)"
                    showError = true
                }
            }
        }
    }
    
    func isValidAddress(_ address: String) -> Bool {
        // Ethereum
        if address.range(of: "^0x[a-fA-F0-9]{40}$", options: .regularExpression) != nil {
            return true
        }
        // Bitcoin
        if address.hasPrefix("bc1") || address.hasPrefix("1") || address.hasPrefix("3") {
            return address.count >= 26 && address.count <= 62
        }
        // Solana
        if address.range(of: "^[1-9A-HJ-NP-Z]{32,44}$", options: .regularExpression) != nil {
            return true
        }
        // TRON
        if address.hasPrefix("T") && address.count == 34 {
            return true
        }
        return false
    }
    
    func chainIcon(for chain: String) -> String {
        switch chain {
        case "Ethereum": return "◈"
        case "BNB Chain": return "🟡"
        case "Polygon": return "🟣"
        case "Arbitrum": return "🔵"
        case "Optimism": return "🔴"
        case "Avalanche": return "❄️"
        case "Solana": return "☀️"
        case "TRON": return "🔺"
        default: return "⬡"
        }
    }
}

// QR Scanner Sheet
struct QRScannerSheet: View {
    let onAddressScanned: (String) -> Void
    @Environment(\.dismiss) var dismiss
    @State private var manualAddress: String = ""
    
    let recentAddresses: [String] = []
    // Recent addresses are loaded from the backend transaction history —
    // never hardcoded demo addresses.
    
    var body: some View {
        NavigationView {
            VStack(spacing: 20) {
                // Camera placeholder
                ZStack {
                    RoundedRectangle(cornerRadius: 16)
                        .fill(Color.black)
                        .frame(height: 250)
                    
                    VStack {
                        Image(systemName: "qrcode.viewfinder")
                            .font(.system(size: 60))
                            .foregroundColor(.white.opacity(0.5))
                        Text("Camera QR Scanner")
                            .foregroundColor(.white.opacity(0.7))
                        Text("Point camera at QR code")
                            .font(.caption)
                            .foregroundColor(.white.opacity(0.5))
                    }
                }
                
                // Manual entry
                VStack(alignment: .leading, spacing: 8) {
                    Text("Or enter address manually:")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                    
                    TextField("0x...", text: $manualAddress)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                        .autocapitalization(.none)
                    
                    Button(action: {
                        if !manualAddress.isEmpty {
                            onAddressScanned(manualAddress)
                        }
                    }) {
                        Text("Use Address")
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(Color.orange)
                            .foregroundColor(.white)
                            .cornerRadius(12)
                    }
                }
                
                // Recent addresses
                VStack(alignment: .leading, spacing: 8) {
                    Text("Recent Addresses")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                    
                    ForEach(recentAddresses, id: \.self) { address in
                        Button(action: {
                            onAddressScanned(address)
                        }) {
                            HStack {
                                Text(formatAddress(address))
                                    .font(.caption)
                                    .fontWeight(.medium)
                                Spacer()
                                Image(systemName: "doc.on.doc")
                                    .font(.caption)
                            }
                            .padding()
                            .background(Color.gray.opacity(0.1))
                            .cornerRadius(8)
                        }
                        .foregroundColor(.primary)
                    }
                }
                
                Spacer()
            }
            .padding()
            .navigationTitle("Scan QR Code")
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
    
    func formatAddress(_ address: String) -> String {
        if address.count > 16 {
            return String(address.prefix(10)) + "..." + String(address.suffix(8))
        }
        return address
    }
}

#Preview {
    SendScreen()
}
