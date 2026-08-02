import SwiftUI

// Bridge Screen - Cross-chain bridging
struct BridgeScreen: View {
    @State private var fromChain: String = "Ethereum"
    @State private var toChain: String = "Polygon"
    @State private var token: String = "ETH"
    @State private var fromAmount: String = ""
    @State private var toAmount: String = ""
    @State private var isBridging: Bool = false
    
    let chains = ["Ethereum", "BNB Chain", "Polygon", "Arbitrum", "Optimism", "Avalanche", "Solana", "Base", "Linea", "ZKSync"]
    let tokens = ["ETH", "USDT", "USDC", "BNB", "MATIC", "AVAX", "SOL"]
    
    var body: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 20) {
                    // From Chain
                    chainSelector(title: "From", chain: $fromChain, tokens: tokens, selectedToken: $token)
                    
                    // Swap Button
                    Button(action: swapChains) {
                        Image(systemName: "arrow.up.arrow.down.circle.fill")
                            .font(.title)
                            .foregroundColor(.orange)
                    }
                    
                    // To Chain
                    chainSelector(title: "To", chain: $toChain, tokens: tokens, selectedToken: $token)
                    
                    // Amount
                    amountSection
                    
                    // Bridge Button
                    bridgeButton
                    
                    // Info
                    infoSection
                }
                .padding()
            }
            .navigationTitle("Bridge")
        }
    }
    
    func chainSelector(title: String, chain: Binding<String>, tokens: [String], selectedToken: Binding<String>) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title)
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            HStack {
                Menu {
                    ForEach(chains, id: \.self) { c in
                        Button(action: { chain.wrappedValue = c }) {
                            Text(c)
                        }
                    }
                } label: {
                    HStack {
                        Text(chainIcon(for: chain.wrappedValue))
                        Text(chain.wrappedValue)
                            .fontWeight(.semibold)
                        Image(systemName: "chevron.down")
                            .font(.caption)
                    }
                    .padding()
                    .background(Color.gray.opacity(0.1))
                    .cornerRadius(8)
                }
                
                Spacer()
            }
            
            // Token selector for this chain
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(tokens, id: \.self) { t in
                        Button(action: { selectedToken.wrappedValue = t }) {
                            Text(t)
                                .padding(.horizontal, 12)
                                .padding(.vertical, 6)
                                .background(selectedToken.wrappedValue == t ? Color.orange : Color.gray.opacity(0.2))
                                .foregroundColor(selectedToken.wrappedValue == t ? .white : .primary)
                                .cornerRadius(16)
                        }
                    }
                }
            }
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(12)
    }
    
    var amountSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Amount")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                Spacer()
                Button(action: { fromAmount = "1.0" }) {
                    Text("MAX")
                        .foregroundColor(.orange)
                }
            }
            
            HStack {
                TextField("0.0", text: $fromAmount)
                    .keyboardType(.decimalPad)
                    .font(.title2)
                
                Text(token)
                    .foregroundColor(.secondary)
            }
            .padding()
            .background(Color.gray.opacity(0.1))
            .cornerRadius(8)
        }
    }
    
    var bridgeButton: some View {
        Button(action: performBridge) {
            HStack {
                if isBridging {
                    ProgressView()
                        .progressViewStyle(CircularProgressViewStyle(tint: .white))
                } else {
                    Image(systemName: "bridge.fill")
                    Text("Bridge")
                        .fontWeight(.bold)
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(Color.orange)
            .foregroundColor(.white)
            .cornerRadius(12)
        }
        .disabled(isBridging)
    }
    
    var infoSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Bridge Info")
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            HStack {
                Text("Estimated Time")
                Spacer()
                Text("~10-30 minutes")
                    .foregroundColor(.secondary)
            }
            .font(.caption)
            
            HStack {
                Text("Fee")
                Spacer()
                Text("~0.1%")
                    .foregroundColor(.secondary)
            }
            .font(.caption)
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(8)
    }
    
    func swapChains() {
        let temp = fromChain
        fromChain = toChain
        toChain = temp
    }
    
    func performBridge() {
        guard !fromAmount.isEmpty else { return }
        isBridging = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
            isBridging = false
            fromAmount = ""
        }
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
        case "Base": return "🔷"
        case "Linea": return "🟢"
        case "ZKSync": return "⚡"
        default: return "⬡"
        }
    }
}

#Preview {
    BridgeScreen()
}
