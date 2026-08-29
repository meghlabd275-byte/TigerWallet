import SwiftUI

// Crypto card: live per-user card balance, funding rates, and card
// transactions — all real fetches from /card/{balance,rates,transactions}.
struct CardsView: View {
    @State private var balance: [String: Any]?
    @State private var rates: [String: Any]?
    @State private var transactions: [String: Any]?
    @State private var errorMessage: String?

    var body: some View {
        NavigationView {
            Form {
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
                Section("Balance") {
                    if let balance = balance {
                        ForEach(balance.sorted(by: { $0.key < $1.key }), id: \.key) { key, value in
                            LabeledContent(key, value: String(describing: value))
                        }
                    } else {
                        Text("Loading…").foregroundColor(.secondary)
                    }
                }
                Section("Rates") {
                    if let rates = rates {
                        ForEach(rates.sorted(by: { $0.key < $1.key }), id: \.key) { key, value in
                            LabeledContent(key, value: String(describing: value))
                        }
                    } else {
                        Text("Loading…").foregroundColor(.secondary)
                    }
                }
                Section("Card Transactions") {
                    if let txs = transactions {
                        let list = (txs["transactions"] as? [[String: Any]]) ?? []
                        if list.isEmpty {
                            Text("No card transactions").foregroundColor(.secondary)
                        } else {
                            ForEach(Array(list.enumerated()), id: \.offset) { _, tx in
                                Text(String(describing: tx))
                                    .font(.caption.monospaced())
                            }
                        }
                    } else {
                        Text("Loading…").foregroundColor(.secondary)
                    }
                }
            }
            .navigationTitle("Crypto Card")
            .onAppear { load() }
        }
    }

    private func load() {
        Task {
            do { let b = try await UserWalletApiService.shared.getCryptoCardBalance()
                await MainActor.run { self.balance = b } } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription } }
            do { let r = try await UserWalletApiService.shared.getCryptoCardRates()
                await MainActor.run { self.rates = r } } catch { }
            do { let t = try await UserWalletApiService.shared.getCardTransactions()
                await MainActor.run { self.transactions = t } } catch { }
        }
    }
}
