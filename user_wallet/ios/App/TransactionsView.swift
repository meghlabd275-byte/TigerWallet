import SwiftUI

// TransactionsView — mirrors the web Transactions page: a send form plus the
// transaction history list. After a successful sendTransaction the real tx
// hash + active-wallet chain id are surfaced via TxSubmittedBanner (with an
// explorer link and 30s auto-dismiss), exactly as on web.

struct TransactionsView: View {
    @State private var transactions: [TransactionRecord] = []
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String = ""
    @State private var isLoading = true
    @State private var errorMessage: String?

    // Send form (mirrors web `send` state: to, amount, password).
    @State private var sendTo = ""
    @State private var sendAmount = ""
    @State private var sendPassword = ""
    @State private var sendError: String?
    @State private var isSending = false
    @State private var sendResult: (hash: String, chainId: Int)?

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                if let res = sendResult {
                    TxSubmittedBanner(txHash: res.hash, chainId: res.chainId) {
                        withAnimation { sendResult = nil }
                    }
                    .padding(.horizontal)
                    .padding(.top, 8)
                    .transition(.move(edge: .top).combined(with: .opacity))
                }

                Form {
                    Section("Send") {
                        Picker("Wallet", selection: $selectedWalletId) {
                            Text("Select a wallet").tag("")
                            ForEach(wallets) { w in
                                Text("\(w.label) — \(shortAddr(w.address))").tag(w.id)
                            }
                        }
                        TextField("Recipient address (0x…)", text: $sendTo)
                            .autocapitalization(.none)
                        TextField("Amount", text: $sendAmount)
                            .keyboardType(.decimalPad)
                        SecureField("Wallet password", text: $sendPassword)
                        if let sendError = sendError {
                            Text(sendError).foregroundColor(.red).font(.caption)
                        }
                        Button(action: handleSend) {
                            if isSending {
                                HStack { Spacer(); ProgressView().tint(.white); Spacer() }
                            } else {
                                Text("Send")
                                    .frame(maxWidth: .infinity)
                            }
                        }
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .background(canSend ? Color.orange : Color.gray.opacity(0.4))
                        .foregroundColor(.white)
                        .cornerRadius(8)
                        .disabled(!canSend || isSending)
                    }

                    Section("History") {
                        if isLoading {
                            ProgressView("Loading transactions...")
                        } else if let errorMessage = errorMessage {
                            Text(errorMessage).foregroundColor(.red)
                        } else if transactions.isEmpty {
                            Text("No transactions yet")
                                .foregroundColor(.secondary)
                        } else {
                            ForEach(transactions) { tx in
                                TransactionRow(transaction: tx)
                            }
                        }
                    }
                }
            }
            .navigationTitle("Transactions")
            .onAppear { loadAll() }
        }
    }

    private var canSend: Bool {
        !selectedWalletId.isEmpty && !sendTo.isEmpty && !sendAmount.isEmpty && sendPassword.count >= 8
    }

    private func shortAddr(_ a: String) -> String {
        guard a.count > 12 else { return a }
        return "\(a.prefix(6))…\(a.suffix(4))"
    }

    private func loadAll() {
        Task { await loadWalletsThenTransactions() }
    }

    private func loadWalletsThenTransactions() async {
        isLoading = true
        errorMessage = nil
        do {
            let ws = try await UserWalletApiService.shared.getWallets()
            await MainActor.run {
                self.wallets = ws
                if selectedWalletId.isEmpty { self.selectedWalletId = ws.first?.id ?? "" }
            }
            await loadTransactions()
        } catch {
            await MainActor.run {
                self.errorMessage = error.localizedDescription
                self.isLoading = false
            }
        }
    }

    private func loadTransactions() async {
        guard let w = selectedWallet else {
            await MainActor.run { self.isLoading = false }
            return
        }
        do {
            let result = try await UserWalletApiService.shared.getTransactions(address: w.address, chainId: w.chain_id)
            await MainActor.run {
                self.transactions = result
                self.isLoading = false
            }
        } catch {
            await MainActor.run {
                self.errorMessage = error.localizedDescription
                self.isLoading = false
            }
        }
    }

    // Mirrors web handleSend: validate -> sendTransaction -> banner with real
    // tx hash + active-wallet chain id -> reload history.
    private func handleSend() {
        sendError = nil
        guard let w = selectedWallet else { sendError = "Select a wallet first"; return }
        guard sendPassword.count >= 8 else { sendError = "Password must be at least 8 characters"; return }
        isSending = true
        Task {
            do {
                let res = try await UserWalletApiService.shared.sendTransaction(
                    walletId: w.id, password: sendPassword, to: sendTo, value: sendAmount, chainId: w.chain_id)
                await MainActor.run {
                    self.sendResult = (hash: res.tx_hash, chainId: w.chain_id)
                    self.sendTo = ""
                    self.sendAmount = ""
                    self.sendPassword = ""
                    self.isSending = false
                }
                await loadTransactions()
            } catch {
                await MainActor.run {
                    self.sendError = error.localizedDescription
                    self.isSending = false
                }
            }
        }
    }
}

struct TransactionRow: View {
    let transaction: TransactionRecord

    private var success: Bool { transaction.isError == "0" }

    var body: some View {
        VStack(alignment: .leading) {
            HStack {
                Text(String(transaction.hash.prefix(16)) + "...")
                    .font(.headline)
                    .font(.system(.headline, design: .monospaced))
                Spacer()
                Text(success ? "Success" : "Failed")
                    .font(.caption)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(success ? Color.green : Color.red)
                    .foregroundColor(.white)
                    .cornerRadius(4)
            }
            Text("Value: \(transaction.value)")
                .font(.subheadline)
            if let date = date {
                Text(date, style: .date)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
    }

    private var date: Date? {
        guard let ts = TimeInterval(transaction.timeStamp) else { return nil }
        return Date(timeIntervalSince1970: ts)
    }
}
