import SwiftUI

// DeFi hub — mirrors the web /defi page (single view, sectioned tabs). One
// view surfaces all DeFi: Lending, Copy Trading, DAO, Perpetual, Margin,
// Prediction, Launchpool, Token Sales. Each section fetches real data via
// UserWalletApiService.shared on selection and drives real actions. On success
// shows "Transaction submitted to the blockchain network".
struct DeFiView: View {
    @State private var section: DeFiSection = .lending

    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var password = ""
    @State private var asset = "ETH"
    @State private var amount = ""
    @State private var pair = "ETH/USDC"
    @State private var side = "long"
    @State private var leverage: Double = 2
    @State private var supportVote = true

    @State private var data: Any?
    @State private var isLoading = false
    @State private var isWorking = false
    @State private var errorMessage: String?
    @State private var resultMessage: String?
    @State private var showSuccess = false
    @State private var successDetail = ""

    enum DeFiSection: String, CaseIterable, Identifiable {
        case lending, copytrading, governance, perpetual, margin, prediction, launchpool, tokensales
        var id: String { rawValue }
        var label: String {
            switch self {
            case .lending: return "Lending"
            case .copytrading: return "Copy Trading"
            case .governance: return "DAO"
            case .perpetual: return "Perpetual"
            case .margin: return "Margin"
            case .prediction: return "Prediction"
            case .launchpool: return "Launchpool"
            case .tokensales: return "Token Sales"
            }
        }
        var icon: String {
            switch self {
            case .lending: return "banknote"
            case .copytrading: return "person.2.crop.square.stack"
            case .governance: return "checkmark.circle"
            case .perpetual: return "chart.line.uptrend.xyaxis"
            case .margin: return "percent"
            case .prediction: return "dice"
            case .launchpool: return "flame"
            case .tokensales: return "tag"
            }
        }
    }

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(DeFiSection.allCases) { s in
                            Button {
                                section = s
                                loadSection()
                            } label: {
                                Label(s.label, systemImage: s.icon)
                                    .font(.subheadline.bold())
                                    .padding(.horizontal, 12).padding(.vertical, 8)
                                    .background(section == s ? Color.orange : Color(.systemGray5))
                                    .foregroundColor(section == s ? .white : .primary)
                                    .cornerRadius(20)
                            }
                        }
                    }
                    .padding(.horizontal)
                    .padding(.vertical, 8)
                }

                Form {
                    if isLoading {
                        Section { ProgressView("Loading \(section.label)...") }
                    } else if let errorMessage = errorMessage {
                        Section {
                            Text(errorMessage).foregroundColor(.red).font(.subheadline)
                            Button("Retry", action: loadSection)
                        }
                    } else {
                        sectionContent
                    }

                    if let resultMessage = resultMessage {
                        Section {
                            Label(resultMessage, systemImage: "checkmark.circle.fill")
                                .foregroundColor(.green).font(.subheadline)
                        }
                    }

                    if let data = data {
                        Section("Response") {
                            Text(Self.pretty(data))
                                .font(.system(.caption, design: .monospaced))
                                .textSelection(.enabled)
                        }
                    }
                }
            }
            .navigationTitle("DeFi")
            .onAppear {
                loadWallets()
                loadSection()
            }
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

    @ViewBuilder
    private var sectionContent: some View {
        switch section {
        case .lending: lendingSection
        case .copytrading: copytradingSection
        case .governance: governanceSection
        case .perpetual: perpetualSection
        case .margin: marginSection
        case .prediction: predictionSection
        case .launchpool: launchpoolSection
        case .tokensales: tokenSalesSection
        }
    }

    // MARK: - Sections

    @ViewBuilder
    private var lendingSection: some View {
        Section("Wallet") { walletPicker }
        Section("Supply / Borrow") {
            TextField("Asset (e.g. ETH)", text: $asset)
                .autocapitalization(.none).disableAutocorrection(true)
            TextField("Amount", text: $amount).keyboardType(.decimalPad)
            SecureField("Wallet password", text: $password)
        }
        Section {
            actionButton("Supply", { perform { try await UserWalletApiService.shared.lendingSupply(walletId: $0, password: password, asset: asset, amount: amount) } })
            actionButton("Borrow", { perform { try await UserWalletApiService.shared.lendingBorrow(walletId: $0, password: password, asset: asset, amount: amount) } })
            actionButton("Withdraw", { perform { try await UserWalletApiService.shared.lendingWithdraw(walletId: $0, password: password, asset: asset, amount: amount) } })
            actionButton("Repay", { perform { try await UserWalletApiService.shared.lendingRepay(walletId: $0, password: password, asset: asset, amount: amount) } })
        }
    }

    @ViewBuilder
    private var copytradingSection: some View {
        Section("Follow a trader") {
            TextField("Trader ID", text: $amount)
                .autocapitalization(.none).disableAutocorrection(true)
            TextField("Allocation (optional)", text: $asset)
                .keyboardType(.decimalPad)
        }
        Section {
            actionButton("Follow Trader", {
                performOptionalWallet { _ in
                    try await UserWalletApiService.shared.followTrader(
                        traderId: amount, allocation: asset.isEmpty ? nil : asset)
                }
            })
            actionButton("Stop Copying", {
                performOptionalWallet { _ in
                    try await UserWalletApiService.shared.stopCopyTrader(copierId: amount)
                }
            })
        }
    }

    @ViewBuilder
    private var governanceSection: some View {
        Section("New proposal") {
            TextField("Title", text: $pair).autocapitalization(.words)
            TextField("Description", text: $amount).autocapitalization(.sentences)
        }
        Section {
            actionButton("Create Proposal", {
                performOptionalWallet { _ in
                    try await UserWalletApiService.shared.createDaoProposal(title: pair, description: amount)
                }
            })
        }
        Section("Vote") {
            Toggle("Vote: Support", isOn: $supportVote)
            TextField("Proposal ID", text: $asset)
                .autocapitalization(.none).disableAutocorrection(true)
            actionButton("Cast Vote", {
                performOptionalWallet { _ in
                    try await UserWalletApiService.shared.voteDaoProposal(
                        proposalId: asset, support: supportVote)
                }
            })
        }
    }

    @ViewBuilder
    private var perpetualSection: some View {
        Section("Wallet") { walletPicker }
        Section("Open position") {
            TextField("Pair (e.g. ETH/USDC)", text: $pair)
                .autocapitalization(.none).disableAutocorrection(true)
            Picker("Side", selection: $side) {
                Text("Long").tag("long")
                Text("Short").tag("short")
            }
            TextField("Size", text: $amount).keyboardType(.decimalPad)
            VStack(alignment: .leading) {
                Text("Leverage: \(Int(leverage))x")
                Slider(value: $leverage, in: 1...20, step: 1)
            }
        }
        Section {
            actionButton("Open Perpetual", {
                perform { walletId in
                    _ = walletId
                    try await UserWalletApiService.shared.createPerpetualPosition(
                        pair: pair, side: side, size: amount, leverage: Int(leverage))
                }
            })
        }
    }

    @ViewBuilder
    private var marginSection: some View {
        Section("Wallet") { walletPicker }
        Section("Open margin position") {
            TextField("Pair (e.g. ETH/USDC)", text: $pair)
                .autocapitalization(.none).disableAutocorrection(true)
            Picker("Side", selection: $side) {
                Text("Long").tag("long")
                Text("Short").tag("short")
            }
            TextField("Size", text: $amount).keyboardType(.decimalPad)
            VStack(alignment: .leading) {
                Text("Leverage: \(Int(leverage))x")
                Slider(value: $leverage, in: 1...20, step: 1)
            }
        }
        Section {
            actionButton("Open Margin", {
                perform { walletId in
                    _ = walletId
                    try await UserWalletApiService.shared.createMarginPosition(
                        pair: pair, side: side, size: amount, leverage: Int(leverage))
                }
            })
        }
    }

    @ViewBuilder
    private var predictionSection: some View {
        Section("Place a bet") {
            TextField("Market ID", text: $pair)
                .autocapitalization(.none).disableAutocorrection(true)
            Picker("Side", selection: $side) {
                Text("Yes").tag("yes")
                Text("No").tag("no")
            }
            TextField("Amount", text: $amount).keyboardType(.decimalPad)
        }
        Section {
            actionButton("Place Bet", {
                performOptionalWallet { _ in
                    try await UserWalletApiService.shared.placePredictionBet(
                        marketId: pair, side: side, amount: amount)
                }
            })
        }
    }

    @ViewBuilder
    private var launchpoolSection: some View {
        Section("Wallet") { walletPicker }
        Section("Stake") {
            TextField("Amount", text: $amount).keyboardType(.decimalPad)
            SecureField("Wallet password", text: $password)
        }
        Section {
            actionButton("Launchpool Stake", { perform { try await UserWalletApiService.shared.launchpoolStake(walletId: $0, password: password, amount: amount) } })
            actionButton("Launchpool Unstake", { perform { try await UserWalletApiService.shared.launchpoolUnstake(walletId: $0, password: password, amount: amount) } })
        }
    }

    @ViewBuilder
    private var tokenSalesSection: some View {
        Section("Participate") {
            TextField("Sale ID", text: $pair)
                .autocapitalization(.none).disableAutocorrection(true)
            TextField("Amount", text: $amount).keyboardType(.decimalPad)
        }
        Section {
            actionButton("Participate", {
                performOptionalWallet { _ in
                    try await UserWalletApiService.shared.participateTokenSale(
                        saleId: pair, amount: amount)
                }
            })
        }
    }

    // MARK: - Shared controls

    @ViewBuilder
    private var walletPicker: some View {
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
        }
    }

    @ViewBuilder
    private func actionButton(_ title: String, _ action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack {
                Text(title)
                Spacer()
                if isWorking { ProgressView().tint(.orange) }
            }
        }
        .disabled(isWorking)
    }

    // MARK: - Loading

    private func loadWallets() {
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    if self.selectedWalletId == nil { self.selectedWalletId = result.first?.id }
                }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func loadSection() {
        isLoading = true
        errorMessage = nil
        data = nil
        Task {
            do {
                let res: Any
                switch section {
                case .lending:
                    let markets = try await UserWalletApiService.shared.getLendingMarkets()
                    res = ["markets": markets] as Any
                case .copytrading:
                    res = try await UserWalletApiService.shared.getCopyTraders()
                case .governance:
                    res = try await UserWalletApiService.shared.getDaoProposals()
                case .perpetual:
                    res = try await UserWalletApiService.shared.getPerpetualPositions()
                case .margin:
                    res = try await UserWalletApiService.shared.getMarginPositions()
                case .prediction:
                    res = try await UserWalletApiService.shared.getPredictionMarkets()
                case .launchpool:
                    let pool = try await UserWalletApiService.shared.getLaunchpool()
                    let stakes = try await UserWalletApiService.shared.getLaunchpoolStakes()
                    res = ["pool": pool, "stakes": stakes] as Any
                case .tokensales:
                    res = try await UserWalletApiService.shared.getTokenSales()
                }
                await MainActor.run {
                    self.data = res
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

    // Perform a wallet-bound action (supply/borrow/perp/margin/launchpool).
    private func perform(_ block: (_ walletId: String) async throws -> [String: Any]) {
        guard let wallet = selectedWallet else {
            errorMessage = "Select a wallet first."
            return
        }
        runAction { try await block(wallet.id) }
    }

    // Perform a wallet-independent action (follow/copy, DAO, prediction, sale).
    private func performOptionalWallet(
        _ block: (_ walletId: String?) async throws -> [String: Any]
    ) {
        runAction { try await block(selectedWalletId) }
    }

    private func runAction(_ block: () async throws -> [String: Any]) {
        isWorking = true
        errorMessage = nil
        resultMessage = nil
        Task {
            do {
                let res = try await block()
                let hash = (res["tx_hash"] as? String) ?? (res["hash"] as? String) ?? ""
                await MainActor.run {
                    self.isWorking = false
                    self.successDetail = hash.isEmpty ? "" : "Tx hash: \(hash)"
                    if hash.isEmpty {
                        // Non-tx actions (follow/vote/bet/sale): show inline result.
                        self.resultMessage = "Done."
                    } else {
                        self.showSuccess = true
                    }
                    self.loadSection()
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isWorking = false
                }
            }
        }
    }

    // Pretty-print opaque JSON for the "Response" section.
    static func pretty(_ data: Any) -> String {
        var json: Any = data
        if let dict = data as? [String: Any], let inner = dict["data"] {
            json = inner
        }
        guard JSONSerialization.isValidJSONObject(json) || json is String || json is NSNumber else {
            return "(no data)"
        }
        if let jsonData = try? JSONSerialization.data(
            withJSONObject: json, options: [.fragmentsAllowed, .prettyPrinted]),
           let str = String(data: jsonData, encoding: .utf8) {
            let trimmed = str.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmed.isEmpty ? "(empty)" : String(trimmed.prefix(800))
        }
        return "(no data)"
    }
}
