import SwiftUI

// Staking tab. Fetches the real supported-asset list + global APY via
// getStakingQuote, then drives stake/unstake/claim on the selected wallet.
// On success shows "Transaction submitted to the blockchain network".
struct StakingView: View {
    @State private var quote: UserWalletApiService.StakingQuote?
    @State private var isLoading = true
    @State private var errorMessage: String?

    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?

    @State private var selectedAsset = "ETH"
    @State private var amount = ""
    @State private var password = ""
    @State private var action: StakingAction = .stake
    @State private var isWorking = false

    @State private var showSuccess = false
    @State private var successDetail = ""

    enum StakingAction: String, CaseIterable, Identifiable {
        case stake, unstake, claim
        var id: String { rawValue }
    }

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    private var canAct: Bool {
        guard let w = selectedWallet else { return false }
        if isWorking { return false }
        if password.count < 8 { return false }
        if action != .claim && amount.trimmingCharacters(in: .whitespaces).isEmpty {
            return false
        }
        return w.chain_id == quote?.assets.first(where: { $0.symbol == selectedAsset })?.chain_id
            || quote == nil
    }

    var body: some View {
        NavigationView {
            Form {
                if isLoading {
                    Section { ProgressView("Loading staking options...") }
                } else if let errorMessage = errorMessage {
                    Section {
                        Text(errorMessage).foregroundColor(.red).font(.subheadline)
                        Button("Retry", action: loadQuote)
                    }
                } else if let quote = quote {
                    Section("Global") {
                        LabeledContent("APY", value: String(format: "%.2f%%", quote.apy))
                        LabeledContent("Min stake", value: String(quote.min_stake))
                        LabeledContent("Lock period", value: "\(quote.lock_period) days")
                    }

                    Section("Assets") {
                        if quote.assets.isEmpty {
                            Text("No staking assets available.")
                                .foregroundColor(.secondary)
                        } else {
                            ForEach(quote.assets, id: \.symbol) { asset in
                                VStack(alignment: .leading, spacing: 4) {
                                    HStack {
                                        Text(asset.symbol).font(.headline)
                                        if asset.verified {
                                            Image(systemName: "checkmark.seal.fill")
                                                .foregroundColor(.green).font(.caption)
                                        }
                                    }
                                    LabeledContent("APY", value: String(format: "%.2f%%", asset.apy))
                                    LabeledContent("Chain", value: "#\(asset.chain_id)")
                                    LabeledContent("Min", value: String(asset.min_stake))
                                    LabeledContent("Lock", value: "\(asset.lock_period) days")
                                }
                            }
                        }
                    }
                }

                Section("Wallet") {
                    if wallets.isEmpty {
                        Text("No wallets yet.").foregroundColor(.secondary)
                    } else {
                        Picker("Wallet", selection: $selectedWalletId) {
                            ForEach(wallets) { wallet in
                                Text("\(wallet.label) - \(wallet.address.prefix(8))...")
                                    .tag(Optional(wallet.id))
                            }
                        }
                    }
                }

                Section("Action") {
                    Picker("Action", selection: $action) {
                        ForEach(StakingAction.allCases) { Text($0.rawValue.capitalized).tag($0) }
                    }
                    Picker("Asset", selection: $selectedAsset) {
                        ForEach(quote?.assets ?? [], id: \.symbol) { Text($0.symbol).tag($0.symbol) }
                    }
                    if action != .claim {
                        TextField("Amount", text: $amount)
                            .keyboardType(.decimalPad)
                    }
                    SecureField("Wallet password", text: $password)
                }

                Section {
                    Button(action: performAction) {
                        HStack {
                            Text(action.rawValue.capitalized)
                            Spacer()
                            if isWorking { ProgressView().tint(.orange) }
                        }
                    }
                    .disabled(!canAct)
                }
            }
            .navigationTitle("Staking")
            .onAppear { loadQuote(); loadWallets() }
            .alert(isPresented: $showSuccess) {
                Alert(
                    title: Text("\u{2713} Transaction submitted to the blockchain network"),
                    message: Text(successDetail),
                    dismissButton: .default(Text("OK")) {
                        amount = ""
                        password = ""
                    }
                )
            }
        }
    }

    private func loadQuote() {
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let q = try await UserWalletApiService.shared.getStakingQuote(nil)
                await MainActor.run {
                    self.quote = q
                    if let first = q.assets.first { self.selectedAsset = first.symbol }
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }

    private func loadWallets() {
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    if self.selectedWalletId == nil { self.selectedWalletId = result.first?.id }
                }
            } catch {
                // Surface wallet-load failures on the next action attempt.
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func performAction() {
        guard let wallet = selectedWallet else { return }
        isWorking = true
        errorMessage = nil
        let asset = selectedAsset
        let amt = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        let pwd = password
        let chainId = wallet.chain_id
        Task {
            do {
                let res: [String: Any]
                switch action {
                case .stake:
                    res = try await UserWalletApiService.shared.stake(
                        walletId: wallet.id, password: pwd, asset: asset, amount: amt, chainId: chainId)
                case .unstake:
                    res = try await UserWalletApiService.shared.unstake(
                        walletId: wallet.id, password: pwd, asset: asset, amount: amt, chainId: chainId)
                case .claim:
                    res = try await UserWalletApiService.shared.claim(
                        walletId: wallet.id, password: pwd, asset: asset, chainId: chainId)
                }
                let hash = (res["tx_hash"] as? String) ?? (res["hash"] as? String) ?? ""
                await MainActor.run {
                    self.isWorking = false
                    self.successDetail = hash.isEmpty ? "" : "Tx hash: \(hash)"
                    self.showSuccess = true
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isWorking = false
                }
            }
        }
    }
}
