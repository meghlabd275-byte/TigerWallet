import SwiftUI

struct WalletsView: View {
    @State private var wallets: [Wallet] = []
    @State private var showingAddWallet = false
    
    var body: some View {
        NavigationView {
            VStack {
                if wallets.isEmpty {
                    Text("No wallets yet")
                        .foregroundColor(.secondary)
                } else {
                    List(wallets, id: \.id) { wallet in
                        WalletRow(wallet: wallet)
                    }
                }
            }
            .navigationTitle("Wallets")
            .toolbar {
                Button(action: { showingAddWallet = true }) {
                    Image(systemName: "plus")
                }
            }
            .sheet(isPresented: $showingAddWallet) {
                AddWalletView()
            }
        }
    }
}

struct WalletRow: View {
    let wallet: Wallet
    
    var body: some View {
        VStack(alignment: .leading) {
            Text(wallet.name)
                .font(.headline)
            Text(wallet.walletType)
                .font(.subheadline)
                .foregroundColor(.secondary)
            Text(wallet.address)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }
}

struct Wallet: Identifiable {
    let id = UUID()
    let name: String
    let walletType: String
    let address: String
}

struct AddWalletView: View {
    @State private var name = ""
    @Environment(\.dismiss) var dismiss
    
    var body: some View {
        NavigationView {
            Form {
                TextField("Wallet Name", text: $name)
            }
            .navigationTitle("Add Wallet")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                Button("Save") {
                    // Create wallet API call
                    dismiss()
                }
            }
        }
    }
}
