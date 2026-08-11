import SwiftUI

struct WalletsView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var isLoading = true
    @State private var showingAddWallet = false
    @State private var newMnemonic: String?

    var body: some View {
        NavigationView {
            Group {
                if isLoading {
                    ProgressView("Loading wallets...")
                } else if wallets.isEmpty {
                    Text("No wallets yet")
                        .foregroundColor(.secondary)
                } else {
                    List(wallets) { wallet in
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
                AddWalletView { newWallet, mnemonic in
                    if let w = newWallet { wallets.append(w) }
                    newMnemonic = mnemonic
                }
            }
            .onAppear { loadWallets() }
        }
    }

    private func loadWallets() {
        isLoading = true
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    self.isLoading = false
                }
            } catch {
                await MainActor.run { self.isLoading = false }
            }
        }
    }
}

struct WalletRow: View {
    let wallet: WalletRecord

    var body: some View {
        VStack(alignment: .leading) {
            Text(wallet.label)
                .font(.headline)
            Text("Chain #\(wallet.chain_id)")
                .font(.subheadline)
                .foregroundColor(.secondary)
            Text(wallet.address)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }
}

struct AddWalletView: View {
    @State private var label = ""
    @State private var password = ""
    @State private var chainId = 1
    @State private var error: String?
    @State private var isCreating = false
    @Environment(\.dismiss) var dismiss
    let onCreated: (WalletRecord?, String?) -> Void

    var body: some View {
        NavigationView {
            Form {
                Section("Wallet") {
                    TextField("Wallet Name", text: $label)
                    Picker("Chain", selection: $chainId) {
                        Text("Ethereum").tag(1)
                        Text("BNB Chain").tag(56)
                        Text("Polygon").tag(137)
                    }
                }
                Section("Security") {
                    SecureField("Password (min 8 chars)", text: $password)
                }
                if let error = error {
                    Section { Text(error).foregroundColor(.red) }
                }
            }
            .navigationTitle("Add Wallet")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") { createWallet() }
                        .disabled(isCreating || label.isEmpty || password.count < 8)
                }
            }
        }
    }

    private func createWallet() {
        isCreating = true
        error = nil
        Task {
            do {
                let w = try await UserWalletApiService.shared.createWallet(label: label, password: password, chainId: chainId)
                await MainActor.run {
                    onCreated(w, w.mnemonic)
                    dismiss()
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isCreating = false
                }
            }
        }
    }
}
