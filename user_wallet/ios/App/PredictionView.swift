import SwiftUI

// Prediction markets: list markets (GET /prediction/markets) and place bets
// (POST /prediction/markets/:id/bet).
struct PredictionView: View {
    @State private var markets: [[String: Any]] = []
    @State private var marketId = ""
    @State private var side = "yes"
    @State private var amount = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showSuccess = false

    var body: some View {
        NavigationView {
            Form {
                Section("Markets") {
                    if markets.isEmpty {
                        Text(isLoading ? "Loading…" : "No active markets")
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(Array(markets.enumerated()), id: \.offset) { _, m in
                            let id = (m["id"] ?? "?") as Any
                            let q = (m["question"] ?? m["title"] ?? "?") as Any
                            let status = (m["status"] ?? "?") as Any
                            Text("• \(String(describing: id)): \(String(describing: q)) (\(String(describing: status)))")
                                .font(.caption.monospaced())
                        }
                    }
                }
                Section("Place Bet") {
                    TextField("Market ID", text: $marketId)
                        .autocapitalization(.none).disableAutocorrection(true)
                    Picker("Side", selection: $side) {
                        Text("Yes").tag("yes")
                        Text("No").tag("no")
                    }
                    TextField("Amount", text: $amount).keyboardType(.decimalPad)
                    Button("Place Bet") { placeBet() }
                        .disabled(marketId.trimmingCharacters(in: .whitespaces).isEmpty
                                  || amount.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("Prediction Markets")
            .onAppear { load() }
            .alert(isPresented: $showSuccess) {
                Alert(title: Text("\u{2713} Bet submitted"),
                      dismissButton: .default(Text("OK")))
            }
        }
    }

    private func load() {
        isLoading = true
        Task {
            do {
                let res = try await UserWalletApiService.shared.getPredictionMarkets()
                let list = (res["markets"] as? [[String: Any]]) ?? (res["data"] as? [[String: Any]]) ?? []
                await MainActor.run { self.markets = list; self.isLoading = false }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }

    private func placeBet() {
        errorMessage = nil
        let id = marketId.trimmingCharacters(in: .whitespaces)
        let amt = amount.trimmingCharacters(in: .whitespaces)
        Task {
            do {
                _ = try await UserWalletApiService.shared.placePredictionBet(marketId: id, side: side, amount: amt)
                await MainActor.run {
                    self.showSuccess = true
                    self.amount = ""
                }
                load()
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}
