import SwiftUI

struct WalletsView: View {
    @EnvironmentObject var onboardingManager: OnboardingManager
    @State private var wallets: [WalletRecord] = []
    @State private var isLoading = true
    @State private var showingAddWallet = false
    @State private var backupPayload: (mnemonic: String, id: String, password: String)?

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
                AddWalletView { newWallet, mnemonic, password in
                    if let w = newWallet {
                        wallets.append(w)
                        // Remember the wallet id so the onboarded gate stays true
                        // (mirrors web Wallets.tsx onCreated -> rememberWallet).
                        onboardingManager.rememberWallet(w.id)
                        if let mnemonic = mnemonic, !mnemonic.isEmpty {
                            backupPayload = (mnemonic, w.id, password)
                        }
                    }
                }
            }
            .sheet(item: Binding<BackupPayload?>(
                get: { backupPayload.map { BackupPayload($0) } },
                set: { if $0 == nil { backupPayload = nil } }
            )) { payload in
                BackupView(mnemonic: payload.mnemonic,
                           walletId: payload.walletId,
                           walletPassword: payload.password) {
                    backupPayload = nil
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

// Lightweight wrapper so BackupView can be presented via .sheet(item:).
// `walletId` is the stored backend wallet id; `id` (Identifiable) derives
// from it so each wallet maps to a stable, unique sheet identity.
struct BackupPayload: Identifiable {
    let mnemonic: String
    let walletId: String
    let password: String
    var id: String { walletId }
    init(_ tuple: (mnemonic: String, id: String, password: String)) {
        self.mnemonic = tuple.mnemonic
        self.walletId = tuple.id
        self.password = tuple.password
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
    let onCreated: (WalletRecord?, String?, String) -> Void

    var body: some View {
        NavigationView {
            Form {
                Section("Wallet") {
                    TextField("Wallet Name", text: $label)
                    Picker("Chain", selection: $chainId) {
                        Text("Ethereum").tag(1)
                        Text("BNB Chain").tag(56)
                        Text("Polygon").tag(137)
                        Text("Arbitrum").tag(42161)
                        Text("Optimism").tag(10)
                        Text("Base").tag(8453)
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
                    onCreated(w, w.mnemonic, password)
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
