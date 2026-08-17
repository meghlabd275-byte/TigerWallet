import SwiftUI

// Swap tab. Fetches a real cross-rate quote via getSwapQuote, displays it,
// then broadcasts the on-chain swap via ammSwap. Wallet picker from
// getWallets. On success shows "Transaction submitted to the blockchain
// network" — mirrors SendView's success alert. No mock data.
struct SwapView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var chainId: Int = 1

    @State private var fromToken = "ETH"
    @State private var toToken = "USDC"
    @State private var amount = ""
    @State private var password = ""

    @State private var quote: UserWalletApiService.SwapQuote?
    @State private var isQuoting = false
    @State private var isSwapping = false
    @State private var errorMessage: String?

    @State private var showSuccess = false
    @State private var successDetail = ""

    private let chains: [(name: String, id: Int)] = [
        ("Ethereum", 1),
        ("BNB Chain", 56),
        ("Polygon", 137),
    ]

    private var canQuote: Bool {
        !fromToken.trimmingCharacters(in: .whitespaces).isEmpty
            && !toToken.trimmingCharacters(in: .whitespaces).isEmpty
            && !amount.trimmingCharacters(in: .whitespaces).isEmpty
            && !isQuoting
    }

    private var canSwap: Bool {
        guard let w = selectedWallet else { return false }
        return !isSwapping
            && quote != nil
            && !amount.trimmingCharacters(in: .whitespaces).isEmpty
            && password.count >= 8
            && w.chain_id == chainId
    }

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    var body: some View {
        NavigationView {
            Form {
                Section("Wallet") {
                    if wallets.isEmpty {
                        Text("No wallets yet. Create or import one first.")
                            .foregroundColor(.secondary)
                    } else {
                        Picker("Wallet", selection: $selectedWalletId) {
                            ForEach(wallets) { wallet in
                                Text("\(wallet.label) - \(wallet.address.prefix(8))...")
                                    .tag(Optional(wallet.id))
                            }
                        }
                        .onChange(of: selectedWalletId) { newValue in
                            if let id = newValue, let w = wallets.first(where: { $0.id == id }) {
                                chainId = w.chain_id
                            }
                            quote = nil
                        }
                    }
                }

                Section("Network") {
                    Picker("Chain", selection: $chainId) {
                        ForEach(chains, id: \.id) { Text($0.name).tag($0.id) }
                    }
                    .onChange(of: chainId) { _ in quote = nil }
                }

                Section("Swap") {
                    TextField("From token (e.g. ETH)", text: $fromToken)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                    TextField("To token (e.g. USDC)", text: $toToken)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                    TextField("Amount", text: $amount)
                        .keyboardType(.decimalPad)
                }

                Section("Security") {
                    SecureField("Wallet password", text: $password)
                }

                Section {
                    Button(action: fetchQuote) {
                        HStack {
                            Text("Get Quote")
                            Spacer()
                            if isQuoting { ProgressView().tint(.orange) }
                        }
                    }
                    .disabled(!canQuote)
                }

                if let quote = quote {
                    Section("Quote") {
                        LabeledContent("You pay", value: "\(quote.from_amount) \(quote.from_token)")
                        LabeledContent("You receive", value: "\(quote.to_amount) \(quote.to_token)")
                        LabeledContent("Price impact", value: String(format: "%.2f%%", quote.price_impact))
                        LabeledContent("Route", value: quote.route)
                    }
                }

                if let errorMessage = errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundColor(.red)
                            .font(.subheadline)
                    }
                }

                Section {
                    Button(action: performSwap) {
                        HStack {
                            Image(systemName: "arrow.left.arrow.right")
                            Text("Swap")
                            Spacer()
                            if isSwapping { ProgressView().tint(.orange) }
                        }
                    }
                    .disabled(!canSwap)
                }
            }
            .navigationTitle("Swap")
            .onAppear { loadWallets() }
            .alert(isPresented: $showSuccess) {
                Alert(
                    title: Text("\u{2713} Transaction submitted to the blockchain network"),
                    message: Text(successDetail),
                    dismissButton: .default(Text("OK")) {
                        amount = ""
                        password = ""
                        quote = nil
                    }
                )
            }
        }
    }

    private func loadWallets() {
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    if self.selectedWalletId == nil {
                        self.selectedWalletId = result.first?.id
                        if let first = result.first { self.chainId = first.chain_id }
                    }
                }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func fetchQuote() {
        isQuoting = true
        errorMessage = nil
        quote = nil
        let f = fromToken.trimmingCharacters(in: .whitespacesAndNewlines)
        let t = toToken.trimmingCharacters(in: .whitespacesAndNewlines)
        let a = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let q = try await UserWalletApiService.shared.getSwapQuote(
                    fromToken: f, toToken: t, fromAmount: a, chainId: chainId)
                await MainActor.run {
                    self.quote = q
                    self.isQuoting = false
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isQuoting = false
                }
            }
        }
    }

    private func performSwap() {
        guard let wallet = selectedWallet else { return }
        isSwapping = true
        errorMessage = nil
        let f = fromToken.trimmingCharacters(in: .whitespacesAndNewlines)
        let t = toToken.trimmingCharacters(in: .whitespacesAndNewlines)
        let a = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.ammSwap(
                    walletId: wallet.id, password: password,
                    fromToken: f, toToken: t, fromAmount: a, chainId: chainId)
                let hash = (res["tx_hash"] as? String)
                    ?? (res["hash"] as? String)
                    ?? ""
                await MainActor.run {
                    self.isSwapping = false
                    self.successDetail = hash.isEmpty ? "" : "Tx hash: \(hash)"
                    self.showSuccess = true
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isSwapping = false
                }
            }
        }
    }
}
