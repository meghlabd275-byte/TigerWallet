import SwiftUI

// Send tab. Lets the user broadcast a real on-chain transaction via
// UserWalletApiService.shared.sendTransaction / autoSendTransaction.
//
// On a successful send an alert is shown:
//   "<checkmark> Transaction submitted to the blockchain network"
// together with the returned tx hash.
struct SendView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var chainId: Int = 1
    @State private var recipient = ""
    @State private var amount = ""
    @State private var password = ""
    @State private var isSending = false
    @State private var errorMessage: String?

    // Success alert state.
    @State private var showSuccess = false
    @State private var successTxHash = ""

    // Passwordless unlock: an unlock_token obtained via `unlockWallet` (passcode)
    // so the wallet can be signed for without re-entering its raw password. When
    // present, the password field becomes optional and `unlockToken` is forwarded
    // to `sendTransaction` / `autoSendTransaction`.
    @State private var unlockToken: String?
    @State private var unlockPasscode = ""
    @State private var isUnlocking = false
    @State private var unlockError: String?

    private let chains: [(name: String, id: Int)] = [
        ("Ethereum", 1),
        ("BNB Chain", 56),
        ("Polygon", 137),
    ]

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    private var canSend: Bool {
        guard let w = selectedWallet else { return false }
        return !isSending
            && !recipient.isEmpty
            && !amount.isEmpty
            && (unlockToken != nil || password.count >= 8)
            && w.chain_id == chainId
    }

    var body: some View {
        NavigationView {
            Form {
                Section("From") {
                    if wallets.isEmpty {
                        Text("No wallets yet. Create or import one first.")
                            .foregroundColor(.secondary)
                    } else {
                        Picker("Wallet", selection: $selectedWalletId) {
                            ForEach(wallets) { wallet in
                                Text("\(wallet.label) - \(wallet.address.prefix(8))...")
                                    .tag(Optional(wallet.id))
                            }
                        }
                        .onChange(of: selectedWalletId) { newValue in
                            if let id = newValue, let w = wallets.first(where: { $0.id == id }) {
                                chainId = w.chain_id
                            }
                            // A different wallet invalidates any cached unlock token.
                            unlockToken = nil
                            unlockError = nil
                        }
                    }
                }

                Section("Network") {
                    Picker("Chain", selection: $chainId) {
                        ForEach(chains, id: \.id) { chain in
                            Text(chain.name).tag(chain.id)
                        }
                    }
                    if let w = selectedWallet {
                        Text("Address: \(w.address.prefix(12))...")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                Section("Transaction") {
                    TextField("Recipient address (0x...)", text: $recipient)
                        .keyboardType(.asciiCapable)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                    TextField("Amount", text: $amount)
                        .keyboardType(.decimalPad)
                }

                Section("Security") {
                    SecureField("Wallet password", text: $password)
                        .disabled(unlockToken != nil)
                    if unlockToken != nil {
                        Label("Unlocked (passwordless) — password not required", systemImage: "checkmark.lock.fill")
                            .font(.caption)
                            .foregroundColor(.green)
                    }
                }

                Section {
                    Button(action: unlockWallet) {
                        HStack {
                            Image(systemName: "key.fill")
                            Text(unlockToken == nil ? "Unlock Wallet (passwordless)" : "Re-unlock Wallet")
                            Spacer()
                            if isUnlocking {
                                ProgressView().tint(.orange)
                            }
                        }
                    }
                    .disabled(isUnlocking || selectedWallet == nil || (unlockToken == nil && unlockPasscode.trimmingCharacters(in: .whitespaces).isEmpty))
                    if unlockToken == nil {
                        SecureField("Passcode", text: $unlockPasscode)
                            .keyboardType(.numberPad)
                    }
                    if let unlockError = unlockError {
                        Text(unlockError)
                            .foregroundColor(.red)
                            .font(.subheadline)
                    }
                }

                if let errorMessage = errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundColor(.red)
                            .font(.subheadline)
                    }
                }

                Section {
                    Button(action: autoSend) {
                        HStack {
                            Image(systemName: "bolt.fill")
                            Text("Auto-Send")
                            Spacer()
                            if isSending {
                                ProgressView().tint(.orange)
                            }
                        }
                    }
                    .disabled(!canSend)
                }
            }
            .navigationTitle("Send")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(action: send) {
                        if isSending {
                            ProgressView().tint(.orange)
                        } else {
                            Text("Send")
                                .fontWeight(.semibold)
                        }
                    }
                    .disabled(!canSend)
                }
            }
            .onAppear { loadWallets() }
            .alert(isPresented: $showSuccess) {
                Alert(
                    title: Text("\u{2713} Transaction submitted to the blockchain network"),
                    message: Text("Tx hash: \(successTxHash)"),
                    dismissButton: .default(Text("OK")) {
                        recipient = ""
                        amount = ""
                        password = ""
                        unlockToken = nil
                        unlockPasscode = ""
                    }
                )
            }
        }
    }

    private func loadWallets() {
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    if self.selectedWalletId == nil {
                        self.selectedWalletId = result.first?.id
                        if let first = result.first {
                            self.chainId = first.chain_id
                        }
                    }
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    private func send() {
        guard let wallet = selectedWallet else { return }
        performSend(auto: false, wallet: wallet)
    }

    private func autoSend() {
        guard let wallet = selectedWallet else { return }
        performSend(auto: true, wallet: wallet)
    }

    private func performSend(auto: Bool, wallet: WalletRecord) {
        isSending = true
        errorMessage = nil
        let to = recipient.trimmingCharacters(in: .whitespacesAndNewlines)
        let value = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let hash: String
                if auto {
                    let res = try await UserWalletApiService.shared.autoSendTransaction(
                        walletId: wallet.id,
                        password: password,
                        to: to,
                        value: value,
                        chainId: chainId,
                        unlockToken: unlockToken
                    )
                    hash = res.tx_hash
                } else {
                    let res = try await UserWalletApiService.shared.sendTransaction(
                        walletId: wallet.id,
                        password: password,
                        to: to,
                        value: value,
                        chainId: chainId,
                        unlockToken: unlockToken
                    )
                    hash = res.tx_hash
                }
                await MainActor.run {
                    self.isSending = false
                    self.successTxHash = hash
                    self.showSuccess = true
                }
            } catch {
                await MainActor.run {
                    self.isSending = false
                    self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    // Passwordless unlock: exchange the wallet's passcode for a short-lived
    // `unlock_token` via `unlockWallet`, then forward it to `sendTransaction`
    // / `autoSendTransaction`. With a token present the wallet password is
    // optional (canSend relaxes the >=8 requirement).
    private func unlockWallet() {
        guard let wallet = selectedWallet else { return }
        isUnlocking = true
        unlockError = nil
        let passcode = unlockPasscode.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.unlockWallet(
                    walletId: wallet.id,
                    params: .init(passcode: passcode,
                                  password: nil,
                                  passkey_assertion: nil,
                                  passkey_auth_data: nil,
                                  passkey_client_data: nil,
                                  unwrapped_unlock_key: nil)
                )
                await MainActor.run {
                    self.isUnlocking = false
                    self.unlockToken = res.unlock_token
                    self.unlockError = nil
                }
            } catch {
                await MainActor.run {
                    self.isUnlocking = false
                    self.unlockError = error.localizedDescription
                }
            }
        }
    }
}
