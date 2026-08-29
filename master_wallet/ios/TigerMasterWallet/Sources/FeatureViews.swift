//
//  FeatureViews.swift
//  TigerMasterWallet — full feature views wired to MasterAPIService.
//
//  Every view fetches from the canonical backend (:8450) via
//  MasterAppState.apiService. No mock data: failures surface the real error.
//  Theming: all views inherit preferredColorScheme from MasterWalletApp
//  (light/dark), so the theme switch applies on every page.
//

import SwiftUI

// MARK: - More hub

struct MoreView: View {
    @EnvironmentObject var appState: MasterAppState

    private let features: [(String, String, String)] = [
        ("Treasury", "building.columns.fill", "treasury"),
        ("Multisig", "lock.shield.fill", "multisig"),
        ("Auto-Sign", "key.fill", "autosign"),
        ("Fees", "percent", "fees"),
        ("Policies", "ruler.fill", "policies"),
        ("Users", "person.3.fill", "users"),
        ("Chains", "link", "chains"),
        ("Tokens", "circle.hexagongrid.fill", "tokens"),
        ("Feature Flags", "flag.fill", "flags"),
        ("Webhooks & Alerts", "bell.fill", "webhooks"),
        ("Audit Log", "doc.text.magnifyingglass", "audit"),
        ("Analytics", "chart.bar.fill", "analytics"),
        ("Passkeys", "person.badge.key.fill", "passkeys"),
        ("Withdraw", "arrow.up.forward.app.fill", "withdraw"),
        ("Sub-Wallets", "rectangle.stack.fill", "subwallets"),
        ("Send", "paperplane.fill", "send"),
        ("Auto-Sign Ops", "wrench.and.screwdriver.fill", "ops"),
    ]

    private let columns = [GridItem(.flexible()), GridItem(.flexible())]

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVGrid(columns: columns, spacing: 12) {
                    ForEach(features, id: \.2) { title, icon, route in
                        NavigationLink(destination: destination(for: route)) {
                            VStack(spacing: 8) {
                                Image(systemName: icon).font(.title2)
                                Text(title).font(.subheadline).fontWeight(.semibold)
                            }
                            .frame(maxWidth: .infinity)
                            .padding()
                            .background(Color(.secondarySystemBackground))
                            .cornerRadius(12)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding()
            }
            .navigationTitle("All Features")
        }
    }

    @ViewBuilder
    private func destination(for route: String) -> some View {
        switch route {
        case "treasury": TreasuryView()
        case "multisig": MultisigView()
        case "autosign": AutoSignView()
        case "fees": FeesView()
        case "policies": PoliciesView()
        case "users": MasterUsersView()
        case "chains": ChainsView()
        case "tokens": TokensView()
        case "flags": FlagsView()
        case "webhooks": WebhooksView()
        case "audit": AuditView()
        case "analytics": MasterAnalyticsView()
        case "passkeys": PasskeysView()
        case "withdraw": WithdrawView()
        case "subwallets": MasterSubWalletsView()
        case "send": SendView()
        case "ops": AutoSignOpsView()
        default: EmptyView()
        }
    }
}

// MARK: - Live feed

/// Wraps the real backend /ws stream. Publishes the latest event line and
/// reloads nothing itself — consumers refresh on balance/transaction events.
final class LiveFeedModel: ObservableObject {
    @Published var lastEvent: String?
    private let ws = WebSocketService()
    private var started = false

    func start(walletId: String, token: String?, onDataEvent: @escaping () -> Void) {
        guard !started else { return }
        started = true
        ws.onMessage = { [weak self] text in
            guard let data = text.data(using: .utf8),
                  let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return }
            let type = (obj["type"] as? String) ?? (obj["channel"] as? String) ?? "event"
            Task { @MainActor in
                self?.lastEvent = type + " · " + String(text.prefix(80))
                if ["balance", "transaction", "tx", "transactions"].contains(type) { onDataEvent() }
            }
        }
        ws.connect(walletId: walletId, token: token)
    }

    func stop() { ws.disconnect(); started = false }
}

// MARK: - Sub-Wallets

struct MasterSubWalletsView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var subs = FeatureLoadState<[SubWallet]>([])
    @State private var sid = ""
    @State private var to = ""
    @State private var amount = ""
    @State private var password = ""
    @State private var actionMessage: String?

    var body: some View {
        Form {
            Section("Sub-Wallets") {
                ForEach(subs.items) { s in
                    Button { sid = s.id } label: {
                        VStack(alignment: .leading) {
                            Text(s.name.isEmpty ? "Sub-wallet" : s.name).font(.subheadline)
                            Text("\(s.address) · \(s.status)").font(.caption).foregroundColor(.secondary)
                        }
                    }
                }
                if subs.loaded && subs.items.isEmpty { Text("No sub-wallets.").foregroundColor(.secondary) }
            }
            Section("Transfer from sub-wallet") {
                TextField("Sub-wallet ID", text: $sid).textInputAutocapitalization(.never)
                TextField("To address", text: $to).textInputAutocapitalization(.never)
                TextField("Amount", text: $amount).keyboardType(.decimalPad)
                SecureField("Wallet password", text: $password)
                Button("Transfer") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            let r = try await appState.apiService.transferSubWallet(masterWalletId: wid, subWalletId: sid, to: to, amount: amount, password: password)
                            actionMessage = "Transfer submitted to the blockchain network: \(r.transactionHash)"
                            password = ""
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(sid.isEmpty || to.isEmpty || amount.isEmpty || password.isEmpty)
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: subs.error) }
        }
        .navigationTitle("Sub-Wallets")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { subs.error = "No master wallet selected"; return }
        do {
            subs.items = try await appState.apiService.getSubWallets(masterWalletId: wid)
            subs.loaded = true
        } catch { subs.error = error.localizedDescription; subs.loaded = true }
    }
}

// MARK: - Send (wallet-level sign + broadcast)

struct SendView: View {
    @EnvironmentObject var appState: MasterAppState
    @State private var to = ""
    @State private var amount = ""
    @State private var token = ""
    @State private var password = ""
    @State private var result: String?

    var body: some View {
        Form {
            Section("Send (sign + broadcast)") {
                TextField("To address", text: $to).textInputAutocapitalization(.never)
                TextField("Amount (e.g. 0.5)", text: $amount).keyboardType(.decimalPad)
                TextField("Token contract (empty = native)", text: $token).textInputAutocapitalization(.never)
                SecureField("Wallet password", text: $password)
                Button("Sign & broadcast") {
                    guard let wid = walletId(of: appState) else { result = "No master wallet selected"; return }
                    Task {
                        do {
                            let r = try await appState.apiService.createTransaction(
                                walletId: wid, to: to, amount: amount, password: password,
                                token: token.isEmpty ? nil : token)
                            result = "Transaction submitted to the blockchain network: \(r.transactionHash)"
                            password = ""
                        } catch { result = error.localizedDescription }
                    }
                }.disabled(to.isEmpty || amount.isEmpty || password.isEmpty)
            }
            if let result = result { Section { Text(result).font(.caption) } }
        }
        .navigationTitle("Send")
    }
}

// MARK: - Auto-Sign Ops

struct AutoSignOpsView: View {
    @EnvironmentObject var appState: MasterAppState
    @State private var chkType = ""
    @State private var chkValue = ""
    @State private var mnemonic = ""
    @State private var chainId = "1"
    @State private var chainType = "evm"
    @State private var txType = "send"
    @State private var to = ""
    @State private var value = ""
    @State private var tokenAddr = ""
    @State private var rpTo = ""
    @State private var rpAmount = ""
    @State private var rpPassword = ""
    @State private var rpWid = ""
    @State private var result: String?

    var body: some View {
        Form {
            Section("Check auto-sign policy") {
                TextField("Tx type (send/claim/swap/trade)", text: $chkType).textInputAutocapitalization(.never)
                TextField("Value (e.g. 1.5)", text: $chkValue).keyboardType(.decimalPad)
                Button("Check") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            let r = try await appState.apiService.checkAutoSignPolicy(id: wid, body: ["tx_type": chkType, "value": chkValue])
                            result = ((r["allowed"] as? Bool) == true ? "ALLOWED" : "DENIED") + " — " + ((r["reason"] as? String) ?? "")
                        } catch { result = error.localizedDescription }
                    }
                }
            }
            Section("Auto-sign transaction (24-word seed)") {
                TextField("24-word mnemonic", text: $mnemonic).textInputAutocapitalization(.never)
                TextField("Chain ID", text: $chainId).keyboardType(.numberPad)
                TextField("Chain type", text: $chainType).textInputAutocapitalization(.never)
                TextField("Tx type", text: $txType).textInputAutocapitalization(.never)
                TextField("To address", text: $to).textInputAutocapitalization(.never)
                TextField("Value", text: $value).keyboardType(.decimalPad)
                TextField("Token contract (optional)", text: $tokenAddr).textInputAutocapitalization(.never)
                Button("Auto-sign tx") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            let r = try await appState.apiService.autoSignTransaction(
                                id: wid, mnemonic: mnemonic, chainId: Int(chainId) ?? 1,
                                chainType: chainType, txType: txType, toAddress: to, value: value,
                                tokenAddress: tokenAddr.isEmpty ? nil : tokenAddr)
                            let hash = (r["transaction_hash"] ?? r["tx_hash"] ?? r["hash"]) as? String ?? ""
                            result = "Transaction submitted to the blockchain network" + (hash.isEmpty ? "" : ": \(hash)")
                            mnemonic = ""
                        } catch { result = error.localizedDescription }
                    }
                }.disabled(mnemonic.isEmpty || to.isEmpty || value.isEmpty)
                Button("UserWallet auto-sign") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            _ = try await appState.apiService.userWalletAutoSign(id: wid, body: [
                                "mnemonic": mnemonic, "chain_id": Int(chainId) ?? 1,
                                "chain_type": chainType, "tx_type": txType])
                            result = "UserWallet auto-sign configuration saved."
                            mnemonic = ""
                        } catch { result = error.localizedDescription }
                    }
                }.disabled(mnemonic.isEmpty)
            }
            Section("Revenue payout (SuperAdmin co-sign required)") {
                TextField("Destination address", text: $rpTo).textInputAutocapitalization(.never)
                TextField("Amount", text: $rpAmount).keyboardType(.decimalPad)
                SecureField("Wallet password", text: $rpPassword)
                TextField("Withdrawal ID (co-signed)", text: $rpWid).textInputAutocapitalization(.never)
                Button("Execute payout") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            let r = try await appState.apiService.revenuePayout(
                                masterId: wid, to: rpTo, amount: rpAmount,
                                password: rpPassword, gasLimit: nil, withdrawalId: rpWid)
                            result = "Payout submitted: \(r.status)"
                            rpPassword = ""
                        } catch { result = error.localizedDescription }
                    }
                }.disabled(rpTo.isEmpty || rpAmount.isEmpty || rpPassword.isEmpty || rpWid.isEmpty)
            }
            if let result = result { Section { Text(result).font(.caption) } }
        }
        .navigationTitle("Auto-Sign Ops")
    }
}

// MARK: - Shared helpers

/// Common state for a loaded list/summary surface.
final class FeatureLoadState<Items>: ObservableObject {
    @Published var items: Items
    @Published var error: String?
    @Published var loaded = false
    init(_ initial: Items) { self.items = initial }
}

struct FeatureErrorText: View {
    let message: String?
    var body: some View {
        if let message = message, !message.isEmpty {
            Text(message).font(.caption).foregroundColor(.red).padding(.horizontal)
        }
    }
}

private func walletId(of appState: MasterAppState) -> String? {
    appState.masterWallet?.id
}

private func anyStr(_ dict: [String: Any], _ keys: [String]) -> String {
    for k in keys {
        if let v = dict[k], !(v is NSNull), !"\(v)".isEmpty { return "\(v)" }
    }
    return ""
}

// MARK: - Treasury

struct TreasuryView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var overview = FeatureLoadState<TreasuryOverview?>(nil)
    @StateObject private var txs = FeatureLoadState<[MasterTransaction]>([])
    @State private var to = ""
    @State private var amount = ""
    @State private var password = ""
    @State private var actionMessage: String?

    var body: some View {
        Form {
            Section("Overview") {
                if let ov = overview.items {
                    LabeledContent("Total USD", value: String(format: "%.2f", ov.totalValueUSD))
                    ForEach(ov.chains, id: \.chainId) { c in
                        LabeledContent("\(c.symbol) (chain \(c.chainId))", value: c.balance)
                    }
                } else {
                    Text(overview.loaded ? "No treasury data." : "Loading…")
                        .foregroundColor(.secondary)
                }
            }
            Section("Transfer") {
                TextField("Destination address", text: $to).textInputAutocapitalization(.never)
                TextField("Amount", text: $amount).keyboardType(.decimalPad)
                SecureField("Wallet password", text: $password)
                Button("Transfer") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            let r = try await appState.apiService.treasuryTransfer(walletId: wid, to: to, amount: amount, password: password)
                            actionMessage = "Transfer broadcast: \(r.transactionHash)"
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(to.isEmpty || amount.isEmpty || password.isEmpty)
            }
            Section("Treasury Transactions") {
                ForEach(txs.items) { t in
                    VStack(alignment: .leading) {
                        Text("\(t.type) — \(t.amount)").font(.subheadline)
                        Text(t.status).font(.caption).foregroundColor(.secondary)
                    }
                }
                if txs.loaded && txs.items.isEmpty { Text("None.").foregroundColor(.secondary) }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
        }
        .navigationTitle("Treasury")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { overview.error = "No master wallet selected"; return }
        do {
            overview.items = try await appState.apiService.getTreasury(walletId: wid)
            overview.loaded = true
        } catch { overview.error = error.localizedDescription; overview.loaded = true }
        do {
            txs.items = try await appState.apiService.getTreasuryTransactions(walletId: wid)
            txs.loaded = true
        } catch { txs.error = error.localizedDescription; txs.loaded = true }
    }
}

// MARK: - Multisig

struct MultisigView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var wallets = FeatureLoadState<[MultisigWallet]>([])
    @StateObject private var txs = FeatureLoadState<[MultisigTransaction]>([])
    @State private var name = ""
    @State private var owners = ""
    @State private var threshold = ""
    @State private var actionMessage: String?

    var body: some View {
        Form {
            Section("Create Multisig Wallet") {
                TextField("Name", text: $name)
                TextField("Owners (comma-separated 0x addresses)", text: $owners).textInputAutocapitalization(.never)
                TextField("Threshold", text: $threshold).keyboardType(.numberPad)
                Button("Create") {
                    guard let wid = walletId(of: appState) else { return }
                    let ownerList = owners.split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
                    Task {
                        do {
                            _ = try await appState.apiService.createMultisigWallet(
                                walletId: wid, name: name, owners: ownerList, threshold: Int(threshold) ?? 0)
                            actionMessage = "Multisig wallet created."
                            await loadWallets()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(name.isEmpty || owners.isEmpty)
            }
            Section("Multisig Wallets") {
                ForEach(wallets.items) { w in
                    Button {
                        Task { await loadTxs(multisigId: w.id) }
                    } label: {
                        VStack(alignment: .leading) {
                            Text(w.name).font(.subheadline).fontWeight(.semibold)
                            Text("\(w.threshold)/\(w.owners.count) · \(w.address)").font(.caption).foregroundColor(.secondary)
                        }
                    }
                }
                if wallets.loaded && wallets.items.isEmpty { Text("None.").foregroundColor(.secondary) }
            }
            if !txs.items.isEmpty {
                Section("Pending Transactions") {
                    ForEach(txs.items) { t in
                        HStack {
                            VStack(alignment: .leading) {
                                Text("\(t.to) — \(t.amount)").font(.caption)
                                Text("\(t.status) · \(t.confirmations)/\(t.threshold)").font(.caption2).foregroundColor(.secondary)
                            }
                            Spacer()
                            Button("Sign") { act { try await appState.apiService.signMultisigTransaction(walletId: walletId(of: appState)!, transactionId: t.id) } }
                            Button("Exec") { act { try await appState.apiService.executeMultisigTransaction(walletId: walletId(of: appState)!, transactionId: t.id) } }
                        }
                    }
                }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: wallets.error) }
        }
        .navigationTitle("Multisig")
        .task { await loadWallets() }
    }

    private func act(_ fn: @escaping () async throws -> MultisigTransaction) {
        Task {
            do {
                _ = try await fn()
                actionMessage = "Submitted."
                await loadWallets()
            } catch { actionMessage = error.localizedDescription }
        }
    }

    private func loadWallets() async {
        guard let wid = walletId(of: appState) else { wallets.error = "No master wallet selected"; return }
        do {
            wallets.items = try await appState.apiService.getMultisigWallets(walletId: wid)
            wallets.loaded = true
        } catch { wallets.error = error.localizedDescription; wallets.loaded = true }
    }

    private func loadTxs(multisigId: String) async {
        guard let wid = walletId(of: appState) else { return }
        do {
            txs.items = try await appState.apiService.getMultisigTransactions(walletId: wid, multisigWalletId: multisigId)
            txs.loaded = true
        } catch { txs.error = error.localizedDescription }
    }
}

// MARK: - Auto-Sign

struct AutoSignView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var rules = FeatureLoadState<[AutoSignRule]>([])
    @StateObject private var logs = FeatureLoadState<[[String: Any]]>([])
    @State private var name = ""
    @State private var maxAmount = ""
    @State private var actionMessage: String?

    var body: some View {
        Form {
            Section("Add Rule") {
                TextField("Rule name", text: $name)
                TextField("Max amount", text: $maxAmount).keyboardType(.decimalPad)
                Button("Add") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            try await appState.apiService.createAutoSignRule(
                                walletId: wid, name: name, ruleType: "max_amount",
                                maxAmount: maxAmount, isActive: true)
                            actionMessage = "Rule added."
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(name.isEmpty)
            }
            Section("Rules") {
                ForEach(rules.items) { r in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(r.name).font(.subheadline)
                            Text("\(r.ruleType ?? "—") · max \(r.maxAmount ?? "—") · \((r.isActive ?? false) ? "active" : "off")")
                                .font(.caption).foregroundColor(.secondary)
                        }
                        Spacer()
                        Button((r.isActive ?? false) ? "Disable" : "Enable") {
                            guard let wid = walletId(of: appState) else { return }
                            Task {
                                do {
                                    try await appState.apiService.updateAutoSignRule(walletId: wid, ruleId: r.id, updates: ["is_active": !(r.isActive ?? false)])
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                        Button("Delete", role: .destructive) {
                            guard let wid = walletId(of: appState) else { return }
                            Task {
                                do {
                                    try await appState.apiService.deleteAutoSignRule(walletId: wid, ruleId: r.id)
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                    }
                }
                if rules.loaded && rules.items.isEmpty { Text("No rules.").foregroundColor(.secondary) }
            }
            Section("Auto-Sign Logs") {
                ForEach(Array(logs.items.enumerated()), id: \.offset) { _, l in
                    VStack(alignment: .leading) {
                        Text(anyStr(l, ["action", "status"])).font(.caption)
                        Text("\(anyStr(l, ["tx_hash"])) \(anyStr(l, ["created_at"]))")
                            .font(.caption2).foregroundColor(.secondary)
                    }
                }
                if logs.loaded && logs.items.isEmpty { Text("No logs.").foregroundColor(.secondary) }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: rules.error) }
        }
        .navigationTitle("Auto-Sign")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { rules.error = "No master wallet selected"; return }
        do {
            rules.items = try await appState.apiService.getAutoSignRules(walletId: wid)
            rules.loaded = true
        } catch { rules.error = error.localizedDescription; rules.loaded = true }
        do {
            logs.items = try await appState.apiService.listAutoSignLogs(id: wid)
            logs.loaded = true
        } catch { logs.error = error.localizedDescription; logs.loaded = true }
    }
}

// MARK: - Fees

struct FeesView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var fees = FeatureLoadState<[Fee]>([])
    @State private var name = ""
    @State private var bps = ""
    @State private var actionMessage: String?

    var body: some View {
        Form {
            Section("Add Fee") {
                TextField("Fee name", text: $name)
                TextField("Basis points", text: $bps).keyboardType(.numberPad)
                Button("Add") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            _ = try await appState.apiService.createFee(walletId: wid, fee: Fee(id: "", name: name, bps: Int(bps) ?? 0))
                            actionMessage = "Fee added."
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(name.isEmpty)
            }
            Section("Fees") {
                ForEach(fees.items, id: \.id) { f in
                    HStack {
                        Text(f.name)
                        Spacer()
                        Text("\(f.bps) bps").foregroundColor(.secondary)
                        Button("Delete", role: .destructive) {
                            guard let wid = walletId(of: appState) else { return }
                            Task {
                                do {
                                    try await appState.apiService.deleteFee(walletId: wid, feeId: f.id)
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                    }
                }
                if fees.loaded && fees.items.isEmpty { Text("No fees.").foregroundColor(.secondary) }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: fees.error) }
        }
        .navigationTitle("Fees")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { fees.error = "No master wallet selected"; return }
        do {
            fees.items = try await appState.apiService.getFees(walletId: wid)
            fees.loaded = true
        } catch { fees.error = error.localizedDescription; fees.loaded = true }
    }
}

// MARK: - Policies

struct PoliciesView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var policies = FeatureLoadState<[Policy]>([])
    @State private var ruleType = ""
    @State private var threshold = ""
    @State private var actionMessage: String?

    var body: some View {
        Form {
            Section("Add Policy") {
                TextField("Rule type (e.g. withdrawal_limit)", text: $ruleType).textInputAutocapitalization(.never)
                TextField("Threshold", text: $threshold).keyboardType(.decimalPad)
                Button("Add") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            _ = try await appState.apiService.createPolicy(walletId: wid, policy: Policy(id: nil, ruleType: ruleType, threshold: Double(threshold) ?? 0))
                            actionMessage = "Policy added."
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(ruleType.isEmpty)
            }
            Section("Policies") {
                ForEach(policies.items, id: \.ruleType) { p in
                    HStack {
                        Text(p.ruleType)
                        Spacer()
                        Text("\(p.threshold)").foregroundColor(.secondary)
                        if let pid = p.id {
                            Button("Delete", role: .destructive) {
                                guard let wid = walletId(of: appState) else { return }
                                Task {
                                    do {
                                        try await appState.apiService.deletePolicy(walletId: wid, policyId: pid)
                                        await load()
                                    } catch { actionMessage = error.localizedDescription }
                                }
                            }.font(.caption)
                        }
                    }
                }
                if policies.loaded && policies.items.isEmpty { Text("No policies.").foregroundColor(.secondary) }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: policies.error) }
        }
        .navigationTitle("Policies")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { policies.error = "No master wallet selected"; return }
        do {
            policies.items = try await appState.apiService.getPolicies(walletId: wid)
            policies.loaded = true
        } catch { policies.error = error.localizedDescription; policies.loaded = true }
    }
}

// MARK: - Users

struct MasterUsersView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var users = FeatureLoadState<[MasterUser]>([])
    @State private var email = ""
    @State private var name = ""
    @State private var password = ""
    @State private var role = "operator"
    @State private var actionMessage: String?

    var body: some View {
        Form {
            Section("Add User") {
                TextField("Email", text: $email).textInputAutocapitalization(.never).keyboardType(.emailAddress)
                TextField("Name", text: $name)
                SecureField("Password (min 8 chars)", text: $password)
                Picker("Role", selection: $role) {
                    Text("Operator").tag("operator")
                    Text("Viewer").tag("viewer")
                    Text("Admin").tag("admin")
                }
                Button("Add") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            _ = try await appState.apiService.createUser(walletId: wid, user: CreateUserRequest(email: email, password: password, name: name, role: role))
                            actionMessage = "User added."
                            email = ""; name = ""; password = ""
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(email.isEmpty || password.count < 8)
            }
            Section("Users") {
                ForEach(users.items) { u in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(u.name).font(.subheadline)
                            Text(u.email).font(.caption).foregroundColor(.secondary)
                            Text("Role: \(u.role)").font(.caption2).foregroundColor(.secondary)
                        }
                        Spacer()
                        Text(u.isActive ? "Active" : "Disabled")
                            .font(.caption)
                            .foregroundColor(u.isActive ? .green : .red)
                        Button("Delete", role: .destructive) {
                            guard let wid = walletId(of: appState) else { return }
                            Task {
                                do {
                                    try await appState.apiService.deleteUser(walletId: wid, userId: u.id)
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                    }
                }
                if users.loaded && users.items.isEmpty { Text("No users.").foregroundColor(.secondary) }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: users.error) }
        }
        .navigationTitle("Users")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { users.error = "No master wallet selected"; return }
        do {
            users.items = try await appState.apiService.getUsers(walletId: wid)
            users.loaded = true
        } catch { users.error = error.localizedDescription; users.loaded = true }
    }
}

// MARK: - Chains

struct ChainsView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var evm = FeatureLoadState<[[String: Any]]>([])
    @StateObject private var nonEvm = FeatureLoadState<[[String: Any]]>([])
    @State private var eChainId = ""
    @State private var eName = ""
    @State private var eRpc = ""
    @State private var eSymbol = ""
    @State private var nChainId = ""
    @State private var nName = ""
    @State private var nType = ""
    @State private var nRpc = ""
    @State private var nPath = ""
    @State private var actionMessage: String?
    @State private var editingEvm: Int?

    var body: some View {
        Form {
            Section(editingEvm != nil ? "Edit EVM Chain" : "Add EVM Chain") {
                TextField("Chain ID", text: $eChainId).keyboardType(.numberPad)
                TextField("Name", text: $eName)
                TextField("RPC URL", text: $eRpc).textInputAutocapitalization(.never)
                TextField("Symbol", text: $eSymbol).textInputAutocapitalization(.characters)
                Button(editingEvm != nil ? "Save EVM chain" : "Add EVM chain") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            if let editId = editingEvm {
                                _ = try await appState.apiService.updateUserEVMChain(
                                    id: wid, chainId: editId, name: eName, symbol: eSymbol, rpcUrl: eRpc)
                                actionMessage = "EVM chain updated."
                                editingEvm = nil
                            } else {
                                _ = try await appState.apiService.addUserEVMChain(
                                    id: wid, chainId: Int(eChainId) ?? 0, name: eName, symbol: eSymbol,
                                    rpcUrl: eRpc, explorerUrl: "", decimals: 18, derivationPath: "m/44'/60'/0'/0/0")
                                actionMessage = "EVM chain added."
                            }
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }
            }
            Section("EVM Chains") {
                ForEach(Array(evm.items.enumerated()), id: \.offset) { _, c in
                    HStack {
                        VStack(alignment: .leading) {
                            Text("\(anyStr(c, ["name"])) (\(anyStr(c, ["chain_id"])))").font(.subheadline)
                            Text(anyStr(c, ["rpc_url"])).font(.caption).foregroundColor(.secondary)
                        }
                        Spacer()
                        Button("Edit") {
                            eChainId = anyStr(c, ["chain_id"]); eName = anyStr(c, ["name"])
                            eRpc = anyStr(c, ["rpc_url"]); eSymbol = anyStr(c, ["symbol"])
                            editingEvm = Int(anyStr(c, ["chain_id"]))
                        }.font(.caption)
                        Button("Remove", role: .destructive) {
                            guard let wid = walletId(of: appState), let cid = Int(anyStr(c, ["chain_id"])) else { return }
                            Task {
                                do {
                                    try await appState.apiService.removeUserEVMChain(id: wid, chainId: cid)
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                    }
                }
                if evm.loaded && evm.items.isEmpty { Text("None.").foregroundColor(.secondary) }
            }
            Section("Add Non-EVM Chain") {
                TextField("Chain ID (SLIP-44)", text: $nChainId).keyboardType(.numberPad)
                TextField("Name", text: $nName)
                TextField("Chain type (solana/bitcoin/cosmos)", text: $nType).textInputAutocapitalization(.never)
                TextField("RPC / node URL", text: $nRpc).textInputAutocapitalization(.never)
                TextField("Derivation path", text: $nPath).textInputAutocapitalization(.never)
                Button("Add non-EVM chain") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            _ = try await appState.apiService.addUserNonEVMChain(
                                id: wid, chainId: Int(nChainId) ?? 0, name: nName, symbol: "",
                                chainType: nType, rpcUrl: nRpc, derivationPath: nPath, addressPrefix: "")
                            actionMessage = "Non-EVM chain added."
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }
            }
            Section("Non-EVM Chains") {
                ForEach(Array(nonEvm.items.enumerated()), id: \.offset) { _, c in
                    HStack {
                        VStack(alignment: .leading) {
                            Text("\(anyStr(c, ["name"])) (\(anyStr(c, ["chain_type"])))").font(.subheadline)
                            Text("id \(anyStr(c, ["chain_id"]))").font(.caption).foregroundColor(.secondary)
                        }
                        Spacer()
                        Button("Remove", role: .destructive) {
                            guard let wid = walletId(of: appState), let cid = Int(anyStr(c, ["chain_id"])) else { return }
                            Task {
                                do {
                                    try await appState.apiService.removeUserNonEVMChain(id: wid, chainId: cid)
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                    }
                }
                if nonEvm.loaded && nonEvm.items.isEmpty { Text("None.").foregroundColor(.secondary) }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: evm.error ?? nonEvm.error) }
        }
        .navigationTitle("Chains")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { evm.error = "No master wallet selected"; return }
        do {
            evm.items = try await appState.apiService.listUserEVMChains(id: wid)
            evm.loaded = true
        } catch { evm.error = error.localizedDescription; evm.loaded = true }
        do {
            nonEvm.items = try await appState.apiService.listUserNonEVMChains(id: wid)
            nonEvm.loaded = true
        } catch { nonEvm.error = error.localizedDescription; nonEvm.loaded = true }
    }
}

// MARK: - Tokens

struct TokensView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var tokens = FeatureLoadState<[[String: Any]]>([])
    @State private var chainId = ""
    @State private var symbol = ""
    @State private var name = ""
    @State private var address = ""
    @State private var decimals = "18"
    @State private var actionMessage: String?
    @State private var editingToken: String?

    var body: some View {
        Form {
            Section(editingToken != nil ? "Edit Token" : "Add Token") {
                TextField("Chain ID", text: $chainId).keyboardType(.numberPad)
                TextField("Symbol", text: $symbol).textInputAutocapitalization(.characters)
                TextField("Name", text: $name)
                TextField("Contract address", text: $address).textInputAutocapitalization(.never)
                TextField("Decimals", text: $decimals).keyboardType(.numberPad)
                Button(editingToken != nil ? "Save token" : "Add token") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            if let editId = editingToken {
                                _ = try await appState.apiService.updateUserToken(
                                    id: wid, tokenId: editId, symbol: symbol, name: name,
                                    decimals: Int(decimals))
                                actionMessage = "Token updated."
                                editingToken = nil
                            } else {
                                _ = try await appState.apiService.addUserToken(
                                    id: wid, chainId: Int(chainId) ?? 0, contractAddress: address,
                                    symbol: symbol, name: name, decimals: Int(decimals) ?? 18, isNative: address.isEmpty)
                                actionMessage = "Token added."
                            }
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }
            }
            Section("Tokens") {
                ForEach(Array(tokens.items.enumerated()), id: \.offset) { _, t in
                    HStack {
                        VStack(alignment: .leading) {
                            Text("\(anyStr(t, ["symbol"])) — \(anyStr(t, ["name"]))").font(.subheadline)
                            Text("chain \(anyStr(t, ["chain_id"])) · \(anyStr(t, ["contract_address"]))")
                                .font(.caption).foregroundColor(.secondary)
                        }
                        Spacer()
                        Button("Edit") {
                            chainId = anyStr(t, ["chain_id"]); symbol = anyStr(t, ["symbol"])
                            name = anyStr(t, ["name"]); address = anyStr(t, ["contract_address"])
                            decimals = anyStr(t, ["decimals"])
                            let tid = anyStr(t, ["id"])
                            editingToken = tid.isEmpty ? nil : tid
                        }.font(.caption)
                        Button("Remove", role: .destructive) {
                            let tid = anyStr(t, ["id"])
                            guard let wid = walletId(of: appState), !tid.isEmpty else { return }
                            Task {
                                do {
                                    try await appState.apiService.removeUserToken(id: wid, tokenId: tid)
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                    }
                }
                if tokens.loaded && tokens.items.isEmpty { Text("No tokens.").foregroundColor(.secondary) }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: tokens.error) }
        }
        .navigationTitle("Tokens")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { tokens.error = "No master wallet selected"; return }
        do {
            tokens.items = try await appState.apiService.listUserTokens(id: wid, chainId: nil)
            tokens.loaded = true
        } catch { tokens.error = error.localizedDescription; tokens.loaded = true }
    }
}

// MARK: - Feature Flags

struct FlagsView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var flags = FeatureLoadState<[[String: Any]]>([])
    @State private var flagKey = ""
    @State private var actionMessage: String?

    var body: some View {
        Form {
            Section("Add Flag") {
                TextField("Flag key (e.g. enable_swap)", text: $flagKey).textInputAutocapitalization(.never)
                Button("Add") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            _ = try await appState.apiService.addFeatureFlag(id: wid, flagKey: flagKey, flagValue: "true", isEnabled: true)
                            actionMessage = "Flag added."
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(flagKey.isEmpty)
            }
            Section("Feature Flags") {
                ForEach(Array(flags.items.enumerated()), id: \.offset) { _, f in
                    let enabled = anyStr(f, ["is_enabled"]) == "true" || anyStr(f, ["is_enabled"]) == "1"
                    HStack {
                        VStack(alignment: .leading) {
                            Text(anyStr(f, ["flag_key"])).font(.subheadline)
                            Text(anyStr(f, ["description"])).font(.caption).foregroundColor(.secondary)
                        }
                        Spacer()
                        Text(enabled ? "✅" : "⛔")
                        Button(enabled ? "Disable" : "Enable") {
                            let fid = anyStr(f, ["id"])
                            guard let wid = walletId(of: appState), !fid.isEmpty else { return }
                            Task {
                                do {
                                    _ = try await appState.apiService.updateFeatureFlag(id: wid, flagId: fid, isEnabled: !enabled)
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                        Button("Remove", role: .destructive) {
                            let fid = anyStr(f, ["id"])
                            guard let wid = walletId(of: appState), !fid.isEmpty else { return }
                            Task {
                                do {
                                    try await appState.apiService.removeFeatureFlag(id: wid, flagId: fid)
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                    }
                }
                if flags.loaded && flags.items.isEmpty { Text("No flags.").foregroundColor(.secondary) }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: flags.error) }
        }
        .navigationTitle("Feature Flags")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { flags.error = "No master wallet selected"; return }
        do {
            flags.items = try await appState.apiService.listFeatureFlags(id: wid)
            flags.loaded = true
        } catch { flags.error = error.localizedDescription; flags.loaded = true }
    }
}

// MARK: - Webhooks & Notifications

struct WebhooksView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var hooks = FeatureLoadState<[Webhook]>([])
    @StateObject private var notifs = FeatureLoadState<[MasterNotification]>([])
    @State private var url = ""
    @State private var events = ""
    @State private var nTitle = ""
    @State private var nMessage = ""
    @State private var actionMessage: String?

    var body: some View {
        Form {
            Section("Add Webhook") {
                TextField("URL (https://…)", text: $url).textInputAutocapitalization(.never)
                TextField("Events (comma-separated)", text: $events).textInputAutocapitalization(.never)
                Button("Add webhook") {
                    guard let wid = walletId(of: appState) else { return }
                    let list = events.split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
                    Task {
                        do {
                            _ = try await appState.apiService.createWebhook(walletId: wid, webhook: CreateWebhookRequest(url: url, events: list))
                            actionMessage = "Webhook added."
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(url.isEmpty)
            }
            Section("Webhooks") {
                ForEach(hooks.items) { w in
                    HStack {
                        VStack(alignment: .leading) {
                            Text(w.url).font(.caption)
                            Text(w.events.joined(separator: ", ")).font(.caption2).foregroundColor(.secondary)
                        }
                        Spacer()
                        Button("Delete", role: .destructive) {
                            guard let wid = walletId(of: appState) else { return }
                            Task {
                                do {
                                    try await appState.apiService.deleteWebhook(walletId: wid, webhookId: w.id)
                                    await load()
                                } catch { actionMessage = error.localizedDescription }
                            }
                        }.font(.caption)
                    }
                }
                if hooks.loaded && hooks.items.isEmpty { Text("No webhooks.").foregroundColor(.secondary) }
            }
            Section("Send Notification") {
                TextField("Title", text: $nTitle)
                TextField("Message", text: $nMessage)
                Button("Send") {
                    guard let wid = walletId(of: appState) else { return }
                    Task {
                        do {
                            _ = try await appState.apiService.createNotification(walletId: wid, notification: CreateNotificationRequest(type: "alert", title: nTitle, message: nMessage))
                            actionMessage = "Notification sent."
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }.disabled(nTitle.isEmpty || nMessage.isEmpty)
            }
            Section("Notifications") {
                ForEach(notifs.items) { n in
                    VStack(alignment: .leading) {
                        Text(n.title).font(.subheadline)
                        Text(n.message).font(.caption).foregroundColor(.secondary)
                    }
                }
                if notifs.loaded && notifs.items.isEmpty { Text("None.").foregroundColor(.secondary) }
            }
            if let actionMessage = actionMessage { Section { Text(actionMessage).font(.caption) } }
            Section { FeatureErrorText(message: hooks.error ?? notifs.error) }
        }
        .navigationTitle("Webhooks & Alerts")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { hooks.error = "No master wallet selected"; return }
        do {
            hooks.items = try await appState.apiService.getWebhooks(walletId: wid)
            hooks.loaded = true
        } catch { hooks.error = error.localizedDescription; hooks.loaded = true }
        do {
            notifs.items = try await appState.apiService.getNotifications(walletId: wid)
            notifs.loaded = true
        } catch { notifs.error = error.localizedDescription; notifs.loaded = true }
    }
}

// MARK: - Audit

struct AuditView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var entries = FeatureLoadState<[AuditEntry]>([])

    var body: some View {
        List {
            ForEach(entries.items, id: \.id) { e in
                VStack(alignment: .leading) {
                    Text(e.action).font(.subheadline)
                    Text("\(e.actor) · \(e.createdAt.formatted())").font(.caption).foregroundColor(.secondary)
                }
            }
            if entries.loaded && entries.items.isEmpty {
                Text("No audit events.").foregroundColor(.secondary)
            }
        }
        .navigationTitle("Audit Log")
        .task {
            guard let wid = walletId(of: appState) else { entries.error = "No master wallet selected"; return }
            do {
                entries.items = try await appState.apiService.getAudit(walletId: wid)
                entries.loaded = true
            } catch { entries.error = error.localizedDescription; entries.loaded = true }
        }
    }
}

// MARK: - Analytics

struct MasterAnalyticsView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var volume = FeatureLoadState<[VolumeData]>([])
    @StateObject private var wallets = FeatureLoadState<[SubWallet]>([])

    var body: some View {
        List {
            Section("Volume") {
                ForEach(volume.items, id: \.date) { v in
                    LabeledContent(v.date.formatted(date: .abbreviated, time: .omitted),
                                   value: String(format: "%.2f USD", v.volumeUSD))
                }
                if volume.loaded && volume.items.isEmpty { Text("No volume data.").foregroundColor(.secondary) }
            }
            Section("Wallets") {
                LabeledContent("Sub-wallet count", value: "\(wallets.items.count)")
            }
        }
        .navigationTitle("Analytics")
        .task {
            guard let wid = walletId(of: appState) else { volume.error = "No master wallet selected"; return }
            do {
                volume.items = try await appState.apiService.getAnalyticsVolume(walletId: wid)
                volume.loaded = true
            } catch { volume.error = error.localizedDescription; volume.loaded = true }
            do {
                wallets.items = try await appState.apiService.getAnalyticsWallets(walletId: wid)
                wallets.loaded = true
            } catch { wallets.error = error.localizedDescription; wallets.loaded = true }
        }
    }
}

// MARK: - Passkeys

struct PasskeysView: View {
    @EnvironmentObject var appState: MasterAppState
    @StateObject private var passkeys = FeatureLoadState<[PasskeyCredential]>([])
    @State private var label = ""
    @State private var actionMessage: String?

    var body: some View {
        List {
            Section("Register Passkey") {
                TextField("Label (e.g. this iPhone)", text: $label)
                Button("Register passkey") {
                    guard let wid = walletId(of: appState) else { actionMessage = "No master wallet selected"; return }
                    Task {
                        do {
                            let svc = PasskeyService(apiService: appState.apiService)
                            _ = try await svc.register(
                                masterId: wid, relyingPartyId: "tigerwallet.app",
                                relyingPartyName: "TigerWallet Master",
                                userId: wid, userName: "master-owner", label: label)
                            actionMessage = "Passkey registered."
                            label = ""
                            await load()
                        } catch { actionMessage = error.localizedDescription }
                    }
                }
            }
            ForEach(passkeys.items) { p in
                HStack {
                    VStack(alignment: .leading) {
                        Text(p.label.isEmpty ? "Passkey" : p.label).font(.subheadline)
                        Text(String(p.credentialId.prefix(24))).font(.caption).foregroundColor(.secondary)
                    }
                    Spacer()
                    Button("Delete", role: .destructive) {
                        guard let wid = walletId(of: appState) else { return }
                        Task {
                            do {
                                try await appState.apiService.deletePasskey(masterId: wid, credId: p.credentialId)
                                await load()
                            } catch { actionMessage = error.localizedDescription }
                        }
                    }.font(.caption)
                }
            }
            if passkeys.loaded && passkeys.items.isEmpty {
                Text("No passkeys registered.").foregroundColor(.secondary)
            }
            if let actionMessage = actionMessage { Text(actionMessage).font(.caption) }
        }
        .navigationTitle("Passkeys")
        .task { await load() }
    }

    private func load() async {
        guard let wid = walletId(of: appState) else { passkeys.error = "No master wallet selected"; return }
        do {
            passkeys.items = try await appState.apiService.listPasskeys(masterId: wid)
            passkeys.loaded = true
        } catch { passkeys.error = error.localizedDescription; passkeys.loaded = true }
    }
}

// MARK: - Withdraw

struct WithdrawView: View {
    @EnvironmentObject var appState: MasterAppState
    @State private var to = ""
    @State private var amountWei = ""
    @State private var currency = ""
    @State private var chainId = "1"
    @State private var result: String?

    var body: some View {
        Form {
            Section {
                Text("Funds never move without TigerWallet SuperAdmin two-party co-sign. This only files the request.")
                    .font(.caption).foregroundColor(.secondary)
            }
            Section("Withdrawal Request") {
                TextField("Destination address", text: $to).textInputAutocapitalization(.never)
                TextField("Amount (wei)", text: $amountWei).keyboardType(.numberPad)
                TextField("Currency (e.g. ETH)", text: $currency).textInputAutocapitalization(.characters)
                TextField("Chain ID", text: $chainId).keyboardType(.numberPad)
                Button("Request withdrawal") {
                    guard let wid = walletId(of: appState) else { result = "No master wallet selected"; return }
                    Task {
                        do {
                            let r = try await appState.apiService.requestWithdrawal(
                                masterId: wid, toAddress: to, amountWei: amountWei,
                                currency: currency.isEmpty ? nil : currency, chainId: Int(chainId))
                            result = "Withdrawal request \(r.withdrawalId): \(r.status)"
                        } catch { result = error.localizedDescription }
                    }
                }.disabled(to.isEmpty || amountWei.isEmpty)
            }
            if let result = result { Section { Text(result).font(.caption) } }
        }
        .navigationTitle("Withdraw")
    }
}
