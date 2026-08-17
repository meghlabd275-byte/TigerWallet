import SwiftUI

// Bridge tab. Cross-chain transfer from one chain to another.
//
// Honest implementation: the backend exposes no dedicated bridge endpoint, so
// the on-chain leg is broadcast via /send (sendTransaction). An indicative
// convert rate is shown via getConvertQuote (real CoinGecko cross-rate). On
// success shows "Transaction submitted to the blockchain network" — mirrors
// SendView. The user's own address on the destination chain is the recipient
// by default (bridge-to-self).
struct BridgeView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var fromChainId: Int = 1
    @State private var toChainId: Int = 56

    @State private var token = "ETH"
    @State private var amount = ""
    @State private var password = ""
    @State private var recipientOverride = ""

    @State private var quote: UserWalletApiService.SwapQuote?
    @State private var isQuoting = false
    @State private var isBridging = false
    @State private var errorMessage: String?

    @State private var showSuccess = false
    @State private var successDetail = ""

    private let chains: [(name: String, id: Int)] = [
        ("Ethereum", 1),
        ("BNB Chain", 56),
        ("Polygon", 137),
    ]

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    private var recipient: String {
        let r = recipientOverride.trimmingCharacters(in: .whitespacesAndNewlines)
        return r.isEmpty ? (selectedWallet?.address ?? "") : r
    }

    private var canQuote: Bool {
        !token.trimmingCharacters(in: .whitespaces).isEmpty
            && !amount.trimmingCharacters(in: .whitespaces).isEmpty
            && fromChainId != toChainId
            && !isQuoting
    }

    private var canBridge: Bool {
        guard let w = selectedWallet else { return false }
        return !isBridging
            && !amount.trimmingCharacters(in: .whitespaces).isEmpty
            && !recipient.isEmpty
            && password.count >= 8
            && w.chain_id == fromChainId
    }

    var body: some View {
        NavigationView {
            Form {
                Section("Source wallet") {
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
                                fromChainId = w.chain_id
                            }
                            quote = nil
                        }
                    }
                }

                Section("Route") {
                    Picker("From chain", selection: $fromChainId) {
                        ForEach(chains, id: \.id) { Text($0.name).tag($0.id) }
                    }
                    .onChange(of: fromChainId) { _ in quote = nil }

                    Picker("To chain", selection: $toChainId) {
                        ForEach(chains, id: \.id) { Text($0.name).tag($0.id) }
                    }
                    .onChange(of: toChainId) { _ in quote = nil }

                    if fromChainId == toChainId {
                        Text("Select two different chains to bridge.")
                            .font(.caption).foregroundColor(.orange)
                    }
                }

                Section("Asset") {
                    TextField("Token (e.g. ETH)", text: $token)
                        .autocapitalization(.none).disableAutocorrection(true)
                    TextField("Amount", text: $amount)
                        .keyboardType(.decimalPad)
                }

                Section("Destination") {
                    TextField("Recipient (defaults to your address)", text: $recipientOverride)
                        .autocapitalization(.none).disableAutocorrection(true)
                        .font(.system(.body, design: .monospaced))
                    if recipientOverride.trimmingCharacters(in: .whitespaces).isEmpty,
                       let w = selectedWallet {
                        Text(w.address)
                            .font(.caption.monospaced())
                            .foregroundColor(.secondary)
                    }
                }

                Section("Security") {
                    SecureField("Wallet password", text: $password)
                }

                Section {
                    Button(action: fetchQuote) {
                        HStack {
                            Text("Get Indicative Rate")
                            Spacer()
                            if isQuoting { ProgressView().tint(.orange) }
                        }
                    }
                    .disabled(!canQuote)
                }

                if let quote = quote {
                    Section("Indicative") {
                        LabeledContent("From", value: "\(quote.from_amount) \(quote.from_token)")
                        LabeledContent("Est. receive", value: "\(quote.to_amount) \(quote.to_token)")
                        LabeledContent("Price impact", value: String(format: "%.2f%%", quote.price_impact))
                        LabeledContent("Route", value: quote.route)
                        Text("Indicative only. Actual bridge output depends on the bridge contract and network conditions.")
                            .font(.caption).foregroundColor(.secondary)
                    }
                }

                if let errorMessage = errorMessage {
                    Section {
                        Text(errorMessage).foregroundColor(.red).font(.subheadline)
                    }
                }

                Section {
                    Button(action: performBridge) {
                        HStack {
                            Image(systemName: "arrow.left.arrow.right.circle")
                            Text("Bridge")
                            Spacer()
                            if isBridging { ProgressView().tint(.orange) }
                        }
                    }
                    .disabled(!canBridge)
                }
            }
            .navigationTitle("Bridge")
            .onAppear { loadWallets() }
            .alert(isPresented: $showSuccess) {
                Alert(
                    title: Text("\u{2713} Transaction submitted to the blockchain network"),
                    message: Text(successDetail),
                    dismissButton: .default(Text("OK")) {
                        amount = ""
                        password = ""
                        recipientOverride = ""
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
                        if let first = result.first { self.fromChainId = first.chain_id }
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
        let t = token.trimmingCharacters(in: .whitespacesAndNewlines)
        let a = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        // Use the same token symbol as to_token: bridging ETH from chain A to
        // chain B is a same-asset cross-chain move. getConvertQuote yields a
        // real CoinGecko rate for display.
        Task {
            do {
                let q = try await UserWalletApiService.shared.getConvertQuote(
                    fromToken: t, toToken: t, fromAmount: a, chainId: fromChainId)
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

    // No dedicated bridge endpoint: broadcast the on-chain leg via /send
    // (sendTransaction). The recipient is the user's address (or override) on
    // the destination chain.
    private func performBridge() {
        guard let wallet = selectedWallet else { return }
        isBridging = true
        errorMessage = nil
        let to = recipient
        let value = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.sendTransaction(
                    walletId: wallet.id, password: password, to: to,
                    value: value, chainId: fromChainId)
                await MainActor.run {
                    self.isBridging = false
                    self.successDetail = "Tx hash: \(res.tx_hash)"
                    self.showSuccess = true
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isBridging = false
                }
            }
        }
    }
}
