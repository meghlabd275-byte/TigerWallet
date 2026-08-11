import SwiftUI

struct TransactionsView: View {
    @State private var transactions: [TransactionRecord] = []
    @State private var isLoading = true
    @State private var errorMessage: String?

    var body: some View {
        NavigationView {
            Group {
                if isLoading {
                    ProgressView("Loading transactions...")
                } else if let errorMessage = errorMessage {
                    Text(errorMessage).foregroundColor(.red)
                } else if transactions.isEmpty {
                    Text("No transactions yet")
                        .foregroundColor(.secondary)
                } else {
                    List(transactions) { tx in
                        TransactionRow(transaction: tx)
                    }
                }
            }
            .navigationTitle("Transactions")
            .onAppear { loadTransactions() }
        }
    }

    private func loadTransactions() {
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let result = try await UserWalletApiService.shared.getTransactions()
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
