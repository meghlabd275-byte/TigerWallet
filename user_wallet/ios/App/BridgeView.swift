import SwiftUI

// Bridge tab. Real cross-chain transfer via the bridge_service proxy:
// POST /bridge/quote for a live quote, POST /bridge/transfer to initiate the
// transfer. On success shows "Transaction submitted to the blockchain network".
// The user's own address on the destination chain is the recipient by default
// (bridge-to-self).
struct BridgeView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var fromChainId: Int = 1
    @State private var toChainId: Int = 56

    @State private var token = "ETH"
    @State private var amount = ""
    @State private var recipientOverride = ""

    @State private var quote: [String: Any]?
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
        guard selectedWallet != nil else { return false }
        return !isBridging
            && !amount.trimmingCharacters(in: .whitespaces).isEmpty
            && !recipient.isEmpty
            && fromChainId != toChainId
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
                        ForEach(quote.sorted(by: { $0.key < $1.key }), id: \.key) { key, value in
                            LabeledContent(key, value: String(describing: value))
                        }
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
        Task {
            do {
                let q = try await UserWalletApiService.shared.getBridgeQuote(
                    fromChain: fromChainId, toChain: toChainId, token: t, amount: a)
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

    // Real bridge transfer via POST /bridge/transfer (bridge_service). The
    // recipient is the user's address (or override) on the destination chain.
    private func performBridge() {
        guard let wallet = selectedWallet else { return }
        isBridging = true
        errorMessage = nil
        let to = recipient
        let value = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        let t = token.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.bridgeTransfer(body: [
                    "fromChain": fromChainId,
                    "toChain": toChainId,
                    "token": t,
                    "amount": value,
                    "from_address": wallet.address,
                    "to_address": to,
                ])
                await MainActor.run {
                    self.isBridging = false
                    let id = res["id"] ?? res["tx_hash"] ?? res
                    self.successDetail = "Transfer: \(id)"
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
