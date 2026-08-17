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
            && password.count >= 8
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
                        chainId: chainId
                    )
                    hash = res.tx_hash
                } else {
                    let res = try await UserWalletApiService.shared.sendTransaction(
                        walletId: wallet.id,
                        password: password,
                        to: to,
                        value: value,
                        chainId: chainId
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
}
