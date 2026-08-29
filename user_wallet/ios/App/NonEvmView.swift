import SwiftUI

// Non-EVM chains: derive native addresses (bitcoin/solana/cosmos) from the
// stored seed and sign messages. Real key derivation + signing on the
// backend (mainnet only); fail-closed errors, no fabricated addresses.
struct NonEvmView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var chain = "solana"
    @State private var password = ""
    @State private var message = ""
    @State private var derivedAddress = ""
    @State private var signature = ""
    @State private var errorMessage: String?

    private let chains = ["bitcoin", "solana", "cosmos"]

    var body: some View {
        NavigationView {
            Form {
                Section("Derive native address") {
                    Picker("Chain", selection: $chain) {
                        ForEach(chains, id: \.self) { Text($0) }
                    }
                    SecureField("Wallet password", text: $password)
                    Button("Derive Address") { derive() }
                        .disabled(password.isEmpty)
                    if !derivedAddress.isEmpty {
                        Text(derivedAddress)
                            .font(.caption.monospaced())
                            .textSelection(.enabled)
                    }
                }
                Section("Sign message") {
                    TextField("Message", text: $message)
                    Button("Sign Message") { sign() }
                        .disabled(password.isEmpty || message.isEmpty)
                    if !signature.isEmpty {
                        Text(signature)
                            .font(.caption.monospaced())
                            .textSelection(.enabled)
                    }
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red) }
                }
            }
            .navigationTitle("Non-EVM Chains")
            .onAppear { loadWallets() }
        }
    }

    private func loadWallets() {
        Task {
            if let list = try? await UserWalletApiService.shared.getWallets() {
                await MainActor.run { wallets = list }
            }
        }
    }

    private func derive() {
        errorMessage = nil
        guard let wallet = wallets.first else {
            errorMessage = "No wallet available"
            return
        }
        Task {
            do {
                let res = try await UserWalletApiService.shared.deriveNonEvmAddress(
                    walletId: wallet.id, password: password, chainType: chain)
                await MainActor.run {
                    derivedAddress = (res["address"] ?? "") as? String ?? ""
                    if derivedAddress.isEmpty { errorMessage = "No address returned" }
                }
            } catch {
                await MainActor.run { errorMessage = "Derive failed: \(error.localizedDescription)" }
            }
        }
    }

    private func sign() {
        errorMessage = nil
        guard let wallet = wallets.first else {
            errorMessage = "No wallet available"
            return
        }
        Task {
            do {
                let res = try await UserWalletApiService.shared.nonEvmSignMessage(
                    walletId: wallet.id, password: password, message: message, chainType: chain)
                await MainActor.run {
                    signature = (res["signature"] ?? "") as? String ?? ""
                    if signature.isEmpty { errorMessage = "No signature returned" }
                }
            } catch {
                await MainActor.run { errorMessage = "Sign failed: \(error.localizedDescription)" }
            }
        }
    }
}
