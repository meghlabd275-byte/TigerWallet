import SwiftUI

struct WalletsView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var isLoading = true
    @State private var addMode: AddWalletView.Mode = .create
    @State private var showingAddWallet = false

    var body: some View {
        NavigationView {
            Group {
                if isLoading {
                    ProgressView("Loading wallets...")
                } else if wallets.isEmpty {
                    VStack(spacing: 12) {
                        Text("No wallets yet")
                            .foregroundColor(.secondary)
                        Text("Tap + to create or import a wallet.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                } else {
                    List(wallets) { wallet in
                        WalletRow(wallet: wallet)
                    }
                }
            }
            .navigationTitle("Wallets")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Menu {
                        Button {
                            addMode = .create
                            showingAddWallet = true
                        } label: {
                            Label("Create Wallet", systemImage: "plus.circle")
                        }
                        Button {
                            addMode = .import
                            showingAddWallet = true
                        } label: {
                            Label("Import Wallet", systemImage: "square.and.arrow.down")
                        }
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(isPresented: $showingAddWallet) {
                AddWalletView(mode: addMode) { newWallet, _ in
                    if let w = newWallet { wallets.append(w) }
                    loadWallets()
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

// Unified create + import view. In .create mode it generates a fresh mnemonic
// (returned by the backend) and shows it with a Copy button. In .import mode it
// accepts an existing mnemonic + password + chain and imports it.
struct AddWalletView: View {
    enum Mode {
        case create
        case `import`
    }

    let mode: Mode
    let onCreated: (WalletRecord?, String?) -> Void

    @State private var label = ""
    @State private var password = ""
    @State private var chainId = 1
    @State private var mnemonic = ""
    @State private var error: String?
    @State private var isCreating = false

    // Revealed mnemonic after a successful create. Shown inline with a Copy
    // button so the user can back up their seed phrase.
    @State private var revealedMnemonic: String?
    @State private var copied = false

    @Environment(\.dismiss) var dismiss

    private var isImport: Bool { mode == .import }

    private var canSubmit: Bool {
        if isCreating { return false }
        if label.isEmpty || password.count < 8 { return false }
        if isImport && mnemonic.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return false
        }
        return true
    }

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

                if isImport {
                    Section("Seed Phrase") {
                        TextEditor(text: $mnemonic)
                            .frame(minHeight: 80)
                            .autocapitalization(.none)
                            .disableAutocorrection(true)
                        Text("Enter your 12/24-word BIP-39 mnemonic.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                Section("Security") {
                    SecureField("Password (min 8 chars)", text: $password)
                }

                if let revealedMnemonic = revealedMnemonic {
                    Section {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Your recovery phrase")
                                .font(.headline)
                            Text(revealedMnemonic)
                                .font(.system(.body, design: .monospaced))
                                .padding(8)
                                .background(Color(.systemGray6))
                                .cornerRadius(8)
                            HStack {
                                Button {
                                    UIPasteboard.general.string = revealedMnemonic
                                    copied = true
                                } label: {
                                    Label("Copy", systemImage: "doc.on.doc")
                                }
                                if copied {
                                    Text("Copied!")
                                        .font(.caption)
                                        .foregroundColor(.green)
                                }
                                Spacer()
                            }
                            Text("Store this safely. It will not be shown again.")
                                .font(.caption)
                                .foregroundColor(.red)
                        }
                    }
                }

                if let error = error {
                    Section { Text(error).foregroundColor(.red) }
                }
            }
            .navigationTitle(isImport ? "Import Wallet" : "Create Wallet")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                if revealedMnemonic == nil {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { dismiss() }
                    }
                }
                if revealedMnemonic != nil {
                    // After create: replace the Create button with Done so the
                    // user can close the sheet once they've backed up the seed.
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Done") { dismiss() }
                            .fontWeight(.semibold)
                    }
                } else {
                    ToolbarItem(placement: .confirmationAction) {
                        Button(action: submit) {
                            if isCreating {
                                ProgressView().tint(.orange)
                            } else {
                                Text(isImport ? "Import" : "Create")
                                    .fontWeight(.semibold)
                            }
                        }
                        .disabled(!canSubmit)
                    }
                }
            }
        }
    }

    private func submit() {
        isCreating = true
        error = nil
        Task {
            do {
                if isImport {
                    _ = try await UserWalletApiService.shared.importWallet(
                        label: label,
                        password: password,
                        mnemonic: mnemonic.trimmingCharacters(in: .whitespacesAndNewlines),
                        chainId: chainId,
                        passphrase: nil
                    )
                    await MainActor.run {
                        self.isCreating = false
                        self.onCreated(nil, nil)
                        self.dismiss()
                    }
                } else {
                    let w = try await UserWalletApiService.shared.createWallet(
                        label: label,
                        password: password,
                        chainId: chainId
                    )
                    await MainActor.run {
                        self.isCreating = false
                        self.revealedMnemonic = w.mnemonic
                        self.onCreated(w, w.mnemonic)
                    }
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
