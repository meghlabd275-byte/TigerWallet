import SwiftUI

struct DashboardView: View {
    @State private var balances: [Balance] = []
    @State private var isLoading = true
    
    var body: some View {
        NavigationView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Dashboard")
                    .font(.largeTitle)
                    .fontWeight(.bold)
                
                if isLoading {
                    ProgressView()
                } else {
                    ScrollView {
                        VStack(spacing: 12) {
                            ForEach(balances, id: \.id) { balance in
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
    
    func loadBalances() {
        // API call to fetch balances
        isLoading = false
    }
}

struct BalanceCard: View {
    let balance: Balance
    
    var body: some View {
        VStack(alignment: .leading) {
            Text(balance.token)
                .font(.headline)
            Text(balance.network)
                .font(.subheadline)
                .foregroundColor(.secondary)
            Text(balance.balance)
                .font(.title2)
                .fontWeight(.bold)
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(.systemGray6))
        .cornerRadius(12)
    }
}

struct Balance: Identifiable {
    let id = UUID()
    let token: String
    let network: String
    let balance: String
}
