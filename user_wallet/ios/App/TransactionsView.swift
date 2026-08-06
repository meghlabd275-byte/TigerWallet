import SwiftUI

struct TransactionsView: View {
    @State private var transactions: [Transaction] = []
    
    var body: some View {
        NavigationView {
            Group {
                if transactions.isEmpty {
                    Text("No transactions yet")
                        .foregroundColor(.secondary)
                } else {
                    List(transactions, id: \.id) { transaction in
                        TransactionRow(transaction: transaction)
                    }
                }
            }
            .navigationTitle("Transactions")
            .onAppear {
                loadTransactions()
            }
        }
    }
    
    func loadTransactions() {
        // API call to fetch transactions
    }
}

struct TransactionRow: View {
    let transaction: Transaction
    
    var body: some View {
        VStack(alignment: .leading) {
            HStack {
                Text(transaction.type)
                    .font(.headline)
                Spacer()
                Text(transaction.status)
                    .font(.caption)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(transaction.status == "Completed" ? Color.green : Color.orange)
                    .foregroundColor(.white)
                    .cornerRadius(4)
            }
            Text("\(transaction.amount) \(transaction.token)")
                .font(.subheadline)
            Text(transaction.date)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }
}

struct Transaction: Identifiable {
    let id = UUID()
    let type: String
    let status: String
    let amount: String
    let token: String
    let date: String
}
