import SwiftUI

struct DashboardView: View {
    @State private var balances: [BalanceResult] = []
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var livePrices: [String: (Double, Double)] = [:]
    @State private var liveFeed: LiveFeedSocket?

    var body: some View {
        NavigationView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Dashboard")
                    .font(.largeTitle)
                    .fontWeight(.bold)

                if !livePrices.isEmpty {
                    Text(livePrices.sorted(by: { $0.key < $1.key }).map { sym, p in
                        "\(sym) $\(String(format: "%.2f", p.0)) (\(String(format: "%+.2f", p.1))%)"
                    }.joined(separator: "   "))
                        .font(.caption.monospaced())
                        .foregroundColor(.secondary)
                }

                if let errorMessage = errorMessage {
                    Text(errorMessage)
                        .foregroundColor(.red)
                        .font(.subheadline)
                }

                if isLoading {
                    ProgressView("Loading balances...")
                } else if balances.isEmpty {
                    Text("No wallets found. Create one to get started.")
                        .foregroundColor(.secondary)
                } else {
                    ScrollView {
                        VStack(spacing: 12) {
                            ForEach(balances) { balance in
                                BalanceCard(balance: balance)
                            }
                        }
                    }
                }
            }
            .padding()
            .onAppear {
                loadBalances()
                connectLiveFeed()
            }
            .onDisappear {
                liveFeed?.close()
                liveFeed = nil
            }
        }
    }

    /// Public live price feed (WebSocket /api/v1/ws) for the dashboard ticker.
    private func connectLiveFeed() {
        let feed = LiveFeedSocket()
        liveFeed = feed
        feed.onTicker = { frame in
            guard let symbol = frame["symbol"] as? String, !symbol.isEmpty else { return }
            let price = (frame["last_price"] as? NSNumber)?.doubleValue ?? 0
            let change = (frame["change_24h_pct"] as? NSNumber)?.doubleValue ?? 0
            DispatchQueue.main.async {
                livePrices[symbol] = (price, change)
            }
        }
        feed.connect(symbols: ["BTC", "ETH"])
    }

    private func loadBalances() {
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let result = try await UserWalletApiService.shared.getBalances()
                await MainActor.run {
                    self.balances = result
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
}

struct BalanceCard: View {
    let balance: BalanceResult

    var body: some View {
        VStack(alignment: .leading) {
            Text(balance.symbol)
                .font(.headline)
            Text("Chain #\(balance.chain_id)")
                .font(.subheadline)
                .foregroundColor(.secondary)
            Text(String(format: "%.6f", balance.balance_f))
                .font(.title2)
                .fontWeight(.bold)
            Text(String(format: "$%.2f", balance.usd_value))
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(.systemGray6))
        .cornerRadius(12)
    }
}
