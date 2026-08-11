import SwiftUI

// Receive Screen - Displays QR Code for receiving crypto
struct ReceiveScreen: View {
    @State private var selectedChain: String = "Ethereum"
    @State private var walletAddress: String = ""
    @State private var showCopied: Bool = false
    
    let chains = ["Ethereum", "BNB Chain", "Polygon", "Arbitrum", "Optimism", "Avalanche", "Solana", "TRON"]
    
    var body: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 24) {
                    // Chain Selector
                    chainSelector
                    
                    // QR Code Display
                    qrCodeSection
                    
                    // Address Section
                    addressSection
                    
                    // Share Button
                    shareButton
                }
                .padding()
            }
            .navigationTitle("Receive")
        }
    }
    
    var chainSelector: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Select Network")
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
    
    var qrCodeSection: some View {
        VStack(spacing: 16) {
            // QR Code placeholder
            ZStack {
                RoundedRectangle(cornerRadius: 16)
                    .fill(Color.white)
                    .frame(width: 200, height: 200)
                
                Image(systemName: "qrcode")
                    .font(.system(size: 150))
                    .foregroundColor(.black)
            }
            .shadow(radius: 10)
            
            Text(selectedChain)
                .font(.headline)
            
            Text("Scan to send \(tokenSymbol()) to this address")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(16)
    }
    
    var addressSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Your Address")
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            HStack {
                Text(walletAddress)
                    .font(.system(.caption, design: .monospaced))
                    .lineLimit(2)
                    .padding()
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color.gray.opacity(0.1))
                    .cornerRadius(8)
                
                Button(action: copyAddress) {
                    Image(systemName: showCopied ? "checkmark" : "doc.on.doc")
                        .font(.title2)
                        .foregroundColor(showCopied ? .green : .orange)
                }
            }
        }
    }
    
    var shareButton: some View {
        Button(action: shareAddress) {
            HStack {
                Image(systemName: "square.and.arrow.up")
                Text("Share Address")
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(Color.orange)
            .foregroundColor(.white)
            .cornerRadius(12)
        }
    }
    
    func copyAddress() {
        UIPasteboard.general.string = walletAddress
        withAnimation {
            showCopied = true
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
            showCopied = false
        }
    }
    
    func shareAddress() {
        // Share sheet
    }
    
    func tokenSymbol() -> String {
        switch selectedChain {
        case "Ethereum", "Arbitrum", "Optimism": return "ETH"
        case "BNB Chain": return "BNB"
        case "Polygon": return "MATIC"
        case "Avalanche": return "AVAX"
        case "Solana": return "SOL"
        case "TRON": return "TRX"
        default: return "ETH"
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
        case "TRON": return "🔺"
        default: return "⬡"
        }
    }
}

#Preview {
    ReceiveScreen()
}
