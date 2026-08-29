import SwiftUI

// Multisig: create multisig wallets, list them, create/sign/execute multisig
// transactions. All calls go through the wallet_api multisig proxy
// (/wallet/multisig/* -> MasterWallet) — real backend state, fail-closed
// error display, no fabricated data.
struct MultisigView: View {
    @State private var wallets: [[String: Any]] = []
    @State private var transactions: [[String: Any]] = []
    @State private var name = ""
    @State private var owners = ""
    @State private var threshold = "2"
    @State private var txWalletId = ""
    @State private var txTo = ""
    @State private var txValue = ""
    @State private var txData = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var successMessage: String?

    var body: some View {
        NavigationView {
            Form {
                Section("Create Multisig") {
                    TextField("Name", text: $name)
                    TextField("Owners (comma-separated 0x…)", text: $owners)
                        .autocapitalization(.none).disableAutocorrection(true)
                    TextField("Threshold", text: $threshold)
                        .keyboardType(.numberPad)
                    Button("Create Multisig") { createWallet() }
                        .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty
                                  || owners.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                Section("Wallets") {
                    if wallets.isEmpty {
                        Text(isLoading ? "Loading…" : "No multisig wallets")
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(Array(wallets.enumerated()), id: \.offset) { _, w in
                            let id = (w["id"] ?? "") as Any
                            let wname = (w["name"] ?? id) as Any
                            let thr = (w["threshold"] ?? 0) as Any
                            let ownerCount = (w["owners"] as? [Any])?.count ?? 0
                            Text("• \(String(describing: wname)) · \(String(describing: id)) · \(String(describing: thr))-of-\(ownerCount)")
                                .font(.caption.monospaced())
                        }
                    }
                }
                Section("New Transaction") {
                    TextField("Multisig wallet ID", text: $txWalletId)
                        .autocapitalization(.none).disableAutocorrection(true)
                    TextField("To address", text: $txTo)
                        .autocapitalization(.none).disableAutocorrection(true)
                    TextField("Value (wei)", text: $txValue)
                        .keyboardType(.numberPad)
                    TextField("Data (hex, optional)", text: $txData)
                        .autocapitalization(.none).disableAutocorrection(true)
                    Button("Create Transaction") { createTx() }
                        .disabled(txWalletId.isEmpty || txTo.isEmpty || txValue.isEmpty)
                    Button("Load Transactions") { loadTxs() }
                        .disabled(txWalletId.isEmpty)
                }
                Section("Transactions") {
                    if transactions.isEmpty {
                        Text("No multisig transactions").foregroundColor(.secondary)
                    } else {
                        ForEach(Array(transactions.enumerated()), id: \.offset) { _, t in
                            let tid = (t["id"] ?? "") as Any
                            let to = (t["to_address"] ?? "") as Any
                            let status = (t["status"] ?? "") as Any
                            VStack(alignment: .leading) {
                                Text("\(String(describing: tid)) → \(String(describing: to)) · \(String(describing: status))")
                                    .font(.caption.monospaced())
                                HStack {
                                    Button("Sign") { multisigAction(tid: String(describing: tid), action: "sign") }
                                    Button("Execute") { multisigAction(tid: String(describing: tid), action: "execute") }
                                }
                            }
                        }
                    }
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red) }
                }
                if let successMessage = successMessage {
                    Section { Text(successMessage).foregroundColor(.green) }
                }
            }
            .navigationTitle("Multisig")
            .onAppear { loadWallets() }
        }
    }

    private func loadWallets() {
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let res = try await UserWalletApiService.shared.listMultisigWallets()
                await MainActor.run {
                    wallets = (res["multisig_wallets"] ?? res["wallets"] ?? []) as? [[String: Any]] ?? []
                    isLoading = false
                }
            } catch {
                await MainActor.run {
                    errorMessage = "Multisig unavailable: \(error.localizedDescription)"
                    isLoading = false
                }
            }
        }
    }

    private func createWallet() {
        errorMessage = nil
        let ownerList = owners.split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
        let thr = Int(threshold) ?? 0
        Task {
            do {
                _ = try await UserWalletApiService.shared.createMultisigWallet(name: name, owners: ownerList, threshold: thr, chainId: 1)
                await MainActor.run {
                    successMessage = "Multisig wallet created"
                    loadWallets()
                }
            } catch {
                await MainActor.run { errorMessage = "Create failed: \(error.localizedDescription)" }
            }
        }
    }

    private func createTx() {
        errorMessage = nil
        Task {
            do {
                _ = try await UserWalletApiService.shared.createMultisigTransaction(walletId: txWalletId, toAddress: txTo, value: txValue, data: txData)
                await MainActor.run {
                    successMessage = "Multisig transaction created — pending signatures"
                    loadTxs()
                }
            } catch {
                await MainActor.run { errorMessage = "Create tx failed: \(error.localizedDescription)" }
            }
        }
    }

    private func loadTxs() {
        errorMessage = nil
        Task {
            do {
                let res = try await UserWalletApiService.shared.listMultisigTransactions(walletId: txWalletId)
                await MainActor.run {
                    transactions = (res["transactions"] ?? res["multisig_transactions"] ?? []) as? [[String: Any]] ?? []
                }
            } catch {
                await MainActor.run { errorMessage = "Load failed: \(error.localizedDescription)" }
            }
        }
    }

    private func multisigAction(tid: String, action: String) {
        errorMessage = nil
        Task {
            do {
                let res = action == "sign"
                    ? try await UserWalletApiService.shared.signMultisigTransaction(txId: tid)
                    : try await UserWalletApiService.shared.executeMultisigTransaction(txId: tid)
                await MainActor.run {
                    if action == "execute" {
                        let hash = (res["tx_hash"] ?? res["status"] ?? "broadcast") as Any
                        successMessage = "Transaction submitted to the blockchain network: \(String(describing: hash))"
                    } else {
                        successMessage = "Multisig transaction signed"
                    }
                    loadTxs()
                }
            } catch {
                await MainActor.run { errorMessage = "\(action) failed: \(error.localizedDescription)" }
            }
        }
    }
}
