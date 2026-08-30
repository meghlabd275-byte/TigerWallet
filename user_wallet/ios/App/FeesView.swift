import SwiftUI

struct FeeTier: Codable, Identifiable {
    var id: String { tier_name + fee_type }
    let tier_name: String
    let fee_type: String
    let rate_basis_points: String
    let min_amount: String
    let max_amount: String
    let chain_id: Int?
}

struct FeeTx: Codable, Identifiable {
    var id: String { created_at + fee_type + amount }
    let fee_type: String
    let currency: String
    let amount: String
    let chain_id: Int?
    let created_at: String
}

struct FeesView: View {
    @State private var fees: [FeeTier] = []
    @State private var txs: [FeeTx] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        List {
            Section(header: Text("Active Fee Tiers")) {
                if fees.isEmpty {
                    Text("No fee tiers configured. Trading is currently fee-free.")
                        .foregroundColor(.secondary)
                } else {
                    ForEach(fees) { fee in
                        VStack(alignment: .leading, spacing: 4) {
                            HStack {
                                Text(fee.tier_name).font(.headline)
                                Spacer()
                                Text(fee.fee_type).font(.caption).padding(4).background(Color.accentColor.opacity(0.2)).cornerRadius(4)
                            }
                            HStack {
                                Text("Rate: \(bpsToPercent(fee.rate_basis_points))")
                                Spacer()
                                Text("Min: \(fee.min_amount)").font(.caption)
                                Text("Max: \(fee.max_amount.isEmpty ? "—" : fee.max_amount)").font(.caption)
                            }
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                        }
                    }
                }
            }

            Section(header: Text("Recent Settled Fee Transactions")) {
                if txs.isEmpty {
                    Text("No settled fee transactions yet.")
                        .foregroundColor(.secondary)
                } else {
                    ForEach(txs) { tx in
                        VStack(alignment: .leading, spacing: 4) {
                            HStack {
                                Text(tx.fee_type).font(.headline)
                                Spacer()
                                Text("\(tx.amount) \(tx.currency)").font(.subheadline)
                            }
                            Text(tx.created_at).font(.caption).foregroundColor(.secondary)
                        }
                    }
                }
            }
        }
        .navigationTitle("Fee Transparency")
        .onAppear { loadFees() }
        .overlay {
            if loading { ProgressView() }
            if let error = error {
                Text(error).foregroundColor(.red).padding()
            }
        }
    }

    private func bpsToPercent(_ bps: String) -> String {
        guard let n = Double(bps) else { return bps }
        return String(format: "%.2f%%", n / 100)
    }

    private func loadFees() {
        Task {
            do {
                let feeData = try await UserWalletApiService.shared.getPublicFees()
                let txData = try await UserWalletApiService.shared.getPublicFeeTransactions()
                await MainActor.run {
                    self.fees = feeData
                    self.txs = txData
                    self.loading = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.loading = false
                }
            }
        }
    }
}
