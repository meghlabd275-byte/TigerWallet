import SwiftUI

// Swap Screen - DEX swapping functionality
struct SwapScreen: View {
    @State private var fromToken: String = "ETH"
    @State private var toToken: String = "USDT"
    @State private var fromAmount: String = ""
    @State private var toAmount: String = ""
    @State private var slippage: Double = 0.5
    @State private var isSwapping: Bool = false
    @State private var showSettings: Bool = false
    
    let tokens = ["ETH", "USDT", "USDC", "BNB", "MATIC", "SOL", "TRX", "BTC", "WBTC"]
    
    var body: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 20) {
                    // From Token
                    tokenInputSection(
                        title: "You Pay",
                        token: $fromToken,
                        amount: $fromAmount,
                        isFrom: true
                    )
                    
                    // Swap Button
                    Button(action: swapTokens) {
                        Image(systemName: "arrow.up.arrow.down.circle.fill")
                            .font(.title)
                            .foregroundColor(.orange)
                    }
                    
                    // To Token
                    tokenInputSection(
                        title: "You Receive",
                        token: $toToken,
                        amount: $toAmount,
                        isFrom: false
                    )
                    
                    // Exchange Rate
                    exchangeRateSection
                    
                    // Swap Button
                    swapButton
                    
                    // Settings
                    slippageSection
                }
                .padding()
            }
            .navigationTitle("Swap")
            .sheet(isPresented: $showSettings) {
                slippageSettingsSheet
            }
        }
    }
    
    func tokenInputSection(title: String, token: Binding<String>, amount: Binding<String>, isFrom: Bool) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title)
                .font(.subheadline)
                .foregroundColor(.secondary)
            
            HStack {
                // Token Picker
                Menu {
                    ForEach(tokens, id: \.self) { t in
                        Button(action: { token.wrappedValue = t }) {
                            Text(t)
                        }
                    }
                } label: {
                    HStack {
                        Text(token.wrappedValue)
                            .fontWeight(.semibold)
                        Image(systemName: "chevron.down")
                            .font(.caption)
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 8)
                    .background(Color.gray.opacity(0.2))
                    .cornerRadius(8)
                }
                
                // Amount Input
                TextField("0.0", text: amount)
                    .keyboardType(.decimalPad)
                    .multilineTextAlignment(.trailing)
                    .font(.title2)
            }
            
            // Balance
            HStack {
                Text("Balance: 0.0 \(token.wrappedValue)")
                    .font(.caption)
                    .foregroundColor(.secondary)
                
                Spacer()
                
                Button(action: { amount.wrappedValue = "1.0" }) {
                    Text("MAX")
                        .font(.caption)
                        .foregroundColor(.orange)
                }
            }
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(12)
    }
    
    var exchangeRateSection: some View {
        VStack(spacing: 8) {
            if !fromAmount.isEmpty && !toAmount.isEmpty {
                let rate = (Double(toAmount) ?? 0) / (Double(fromAmount) ?? 1)
                Text("1 \(fromToken) = \(String(format: "%.6f", rate)) \(toToken)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            HStack {
                Text("Rate")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Spacer()
                Text("Slippage: \(slippage)%")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
    }
    
    var swapButton: some View {
        Button(action: performSwap) {
            HStack {
                if isSwapping {
                    ProgressView()
                        .progressViewStyle(CircularProgressViewStyle(tint: .white))
                } else {
                    Text("Swap")
                        .fontWeight(.bold)
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(Color.orange)
            .foregroundColor(.white)
            .cornerRadius(12)
        }
        .disabled(isSwapping)
    }
    
    var slippageSection: some View {
        HStack {
            Text("Slippage Tolerance")
                .font(.subheadline)
            Spacer()
            Button(action: { showSettings = true }) {
                Text("\(slippage)%")
                    .foregroundColor(.orange)
            }
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(8)
    }
    
    var slippageSettingsSheet: some View {
        NavigationView {
            VStack(spacing: 20) {
                Text("Slippage Tolerance")
                    .font(.headline)
                
                HStack(spacing: 10) {
                    ForEach([0.1, 0.5, 1.0], id: \.self) { value in
                        Button(action: { slippage = value }) {
                            Text("\(value)%")
                                .padding()
                                .background(slippage == value ? Color.orange : Color.gray.opacity(0.2))
                                .foregroundColor(slippage == value ? .white : .primary)
                                .cornerRadius(8)
                        }
                    }
                }
                
                HStack {
                    Text("Custom:")
                    TextField("0.5", value: $slippage, format: .number)
                        .keyboardType(.decimalPad)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                    Text("%")
                }
                .padding()
                
                Spacer()
            }
            .padding()
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        showSettings = false
                    }
                }
            }
        }
    }
    
    func swapTokens() {
        let temp = fromToken
        fromToken = toToken
        toToken = temp
    }
    
    func performSwap() {
        guard !fromAmount.isEmpty, !toAmount.isEmpty else { return }
        
        isSwapping = true
        
        // Simulate swap
        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
            isSwapping = false
            fromAmount = ""
            toAmount = ""
        }
    }
}

#Preview {
    SwapScreen()
}
