import SwiftUI

/// Wallet & finance plane: multi-chain ledger accounts, deterministic deposit
/// addresses with QR + copy, signed withdrawals, instant convert, KYC-gated
/// internal transfers, escrowed P2P marketplace, full ledger history.
struct FinanceView: View {
    @State private var accountsText = "Loading accounts…"
    @State private var deposits: [[String: Any]] = []
    @State private var ratesText = ""
    @State private var escrow: [[String: Any]] = []
    @State private var historyText = ""
    @State private var currency = "BTC"
    @State private var transferTo = ""
    @State private var amount = ""
    @State private var withdrawAddress = ""
    @State private var statusText = ""
    private let assets = ["BTC", "ETH", "USDT", "USDC", "BNB", "SOL", "TRX", "MATIC", "LTC", "DOGE"]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                Text("Accounts").font(.headline)
                Text(accountsText).font(.system(.caption, design: .monospaced))

                Text("Deposit addresses (tap to copy)").font(.headline)
                ForEach(deposits.indices, id: \.self) { i in
                    let d = deposits[i]
                    HStack {
                        Text("\(d["asset"] as? String ?? ""): \(d["address"] as? String ?? "")")
                            .font(.system(.caption2, design: .monospaced))
                            .lineLimit(1).truncationMode(.middle)
                        Spacer()
                        Button("Copy") { UIPasteboard.general.string = d["address"] as? String }
                            .font(.caption)
                    }
                }

                Picker("Currency", selection: $currency) {
                    ForEach(assets, id: \.self) { Text($0) }
                }

                TextField("Recipient email (internal transfer)", text: $transferTo)
                    .textFieldStyle(.roundedBorder).autocapitalization(.none)
                TextField("Amount", text: $amount)
                    .textFieldStyle(.roundedBorder).keyboardType(.decimalPad)
                Button("Transfer (KYC-gated)") { transfer() }

                TextField("Withdrawal destination address", text: $withdrawAddress)
                    .textFieldStyle(.roundedBorder).autocapitalization(.none)
                HStack {
                    Button("Withdraw") { withdraw() }
                    Button("Convert → USDC") { convert() }
                }

                Text("Convert rates").font(.headline)
                Text(ratesText).font(.system(.caption, design: .monospaced))

                Text("P2P escrow market").font(.headline)
                ForEach(escrow.indices, id: \.self) { i in
                    let o = escrow[i]
                    let status = o["status"] as? String ?? ""
                    VStack(alignment: .leading, spacing: 2) {
                        Text("\(o["amount"] as? String ?? "") \(o["currency"] as? String ?? "") @ \(o["fiat_amount"] as? String ?? "") \(o["fiat_currency"] as? String ?? "") (\(status))")
                            .font(.caption2)
                        HStack(spacing: 4) {
                            if status == "open" {
                                Button("Buy") { escrowAction(o, "accept") }
                                Button("Cancel") { escrowAction(o, "cancel") }
                            }
                            if status == "escrowed" {
                                Button("Mark paid") { escrowAction(o, "paid") }
                                Button("Dispute") { escrowAction(o, "dispute", "disputed") }
                            }
                            if status == "paid" {
                                Button("Release") { escrowAction(o, "release") }
                                Button("Dispute") { escrowAction(o, "dispute", "disputed") }
                            }
                        }.font(.caption)
                    }
                }

                Text("Ledger history").font(.headline)
                Text(historyText).font(.system(.caption2, design: .monospaced))

                if !statusText.isEmpty {
                    Text(statusText).foregroundColor(.secondary).font(.caption)
                }
            }
            .padding()
        }
        .navigationTitle("Wallet & Finance")
        .task { await load() }
    }

    private func load() async {
        do {
            let res = try await UserWalletApiService.shared.getFinanceAccounts()
            let arr = res["accounts"] as? [[String: Any]] ?? []
            accountsText = arr.isEmpty ? "No accounts yet" : arr.map {
                "\($0["currency"] as? String ?? ""): \($0["balance"] as? String ?? "") (available \($0["available"] as? String ?? ""))"
            }.joined(separator: "\n")
        } catch { accountsText = "Accounts unavailable: \(error.localizedDescription)" }
        do {
            let res = try await UserWalletApiService.shared.getDepositAddresses()
            deposits = res["addresses"] as? [[String: Any]] ?? []
        } catch { deposits = [] }
        do {
            let res = try await UserWalletApiService.shared.getConvertRates()
            let arr = res["rates"] as? [[String: Any]] ?? []
            ratesText = arr.isEmpty ? "No rates configured" : arr.map {
                "\($0["from_currency"] as? String ?? "")/\($0["to_currency"] as? String ?? ""): \($0["rate"] as? String ?? "")"
            }.joined(separator: "\n")
        } catch { ratesText = "Rates unavailable" }
        await loadEscrow()
        await loadHistory()
    }

    private func loadEscrow() async {
        do {
            let res = try await UserWalletApiService.shared.getEscrowOrders()
            escrow = res["orders"] as? [[String: Any]] ?? []
        } catch { escrow = [] }
    }

    private func loadHistory() async {
        do {
            let res = try await UserWalletApiService.shared.getFinanceHistory()
            let arr = res["history"] as? [[String: Any]] ?? []
            historyText = arr.isEmpty ? "No ledger history yet" : arr.prefix(30).map {
                let dir = ($0["direction"] as? String) == "debit" ? "−" : "+"
                return "\($0["kind"] as? String ?? "") \(dir)\($0["amount"] as? String ?? "") \($0["currency"] as? String ?? "")"
            }.joined(separator: "\n")
        } catch { historyText = "History unavailable" }
    }

    private func transfer() {
        Task {
            do {
                _ = try await UserWalletApiService.shared.financeTransfer(toEmail: transferTo, currency: currency, amount: amount)
                statusText = "Transfer completed"
                transferTo = ""; amount = ""
                await load()
            } catch { statusText = "Transfer failed: \(error.localizedDescription)" }
        }
    }

    private func withdraw() {
        Task {
            do {
                let res = try await UserWalletApiService.shared.createWithdrawal(currency: currency, amount: amount, toAddress: withdrawAddress)
                statusText = (res["status"] as? String) == "auto_approved" ? "Auto-approved" : "Queued for superadmin sign-off"
                withdrawAddress = ""; amount = ""
                await load()
            } catch { statusText = "Withdrawal failed: \(error.localizedDescription)" }
        }
    }

    private func convert() {
        Task {
            do {
                let res = try await UserWalletApiService.shared.financeConvert(fromCurrency: currency, toCurrency: "USDC", amount: amount)
                statusText = "Converted \(res["from_amount"] as? String ?? "") \(res["from_currency"] as? String ?? "") → \(res["to_amount"] as? String ?? "") USDC"
                amount = ""
                await load()
            } catch { statusText = "Convert failed: \(error.localizedDescription)" }
        }
    }

    private func escrowAction(_ order: [String: Any], _ action: String, _ reason: String? = nil) {
        guard let id = order["id"] as? String else { return }
        Task {
            do {
                _ = try await UserWalletApiService.shared.escrowAction(id: id, action: action, reason: reason)
                statusText = "Escrow \(action) done"
                await loadEscrow()
            } catch { statusText = "Escrow failed: \(error.localizedDescription)" }
        }
    }
}
