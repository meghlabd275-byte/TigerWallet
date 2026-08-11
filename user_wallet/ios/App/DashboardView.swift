import SwiftUI

struct DashboardView: View {
    @State private var balances: [BalanceResult] = []
    @State private var isLoading = true
    @State private var errorMessage: String?

    var body: some View {
        NavigationView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Dashboard")
                    .font(.largeTitle)
                    .fontWeight(.bold)

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
            }
        }
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
