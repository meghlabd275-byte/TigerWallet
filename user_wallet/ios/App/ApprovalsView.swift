import SwiftUI

// Approvals tab. Pick a wallet, fetch its real ERC-20 token approvals via
// getApprovals(address, chainId), and revoke each via revokeApproval.
// Mirrors the web /approvals page.
struct ApprovalsView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var approvals: [Approval] = []
    @State private var isLoading = false
    @State private var revokingId: String?
    @State private var errorMessage: String?

    struct Approval: Identifiable {
        let id: String
        let spender: String
        let contract: String
        let symbol: String
        let amount: String
    }

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    var body: some View {
        NavigationView {
            Group {
                if wallets.isEmpty {
                    VStack(spacing: 12) {
                        if isLoading {
                            ProgressView("Loading wallets...")
                        } else {
                            Text("No wallets yet").foregroundColor(.secondary)
                            Text("Create or import a wallet to check approvals.")
                                .font(.caption).foregroundColor(.secondary)
                                .multilineTextAlignment(.center)
                        }
                    }.padding()
                } else {
                    content
                }
            }
            .navigationTitle("Approvals")
            .onAppear { loadWallets() }
        }
    }

    @ViewBuilder
    private var content: some View {
        VStack(spacing: 0) {
            Picker("Wallet", selection: $selectedWalletId) {
                ForEach(wallets) { wallet in
                    Text("\(wallet.label) - \(wallet.address.prefix(8))...")
                        .tag(Optional(wallet.id))
                }
            }
            .pickerStyle(.menu)
            .padding()

            if isLoading {
                ProgressView("Loading approvals...").padding()
            } else if let errorMessage = errorMessage {
                VStack(spacing: 8) {
                    Text(errorMessage).foregroundColor(.red).font(.subheadline)
                        .multilineTextAlignment(.center)
                    Button("Retry", action: loadApprovals).buttonStyle(.bordered)
                }.padding()
            } else if approvals.isEmpty {
                Text("No token approvals found for this wallet.")
                    .foregroundColor(.secondary).padding()
            } else {
                List {
                    ForEach(approvals) { approval in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(approval.symbol.isEmpty ? approval.contract : approval.symbol)
                                .font(.headline)
                            Text("Spender: \(approval.spender.prefix(14))...")
                                .font(.caption.monospaced()).foregroundColor(.secondary)
                            Text("Approved: \(approval.amount)")
                                .font(.caption).foregroundColor(.secondary)
                        }
                        .swipeActions(edge: .trailing) {
                            Button(role: .destructive) { revoke(approval) } label: {
                                Label("Revoke", systemImage: "xmark.shield")
                            }
                        }
                    }
                }
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
                        self.loadApprovals()
                    }
                }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func loadApprovals() {
        guard let wallet = selectedWallet else { return }
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let raw = try await UserWalletApiService.shared.getApprovals(
                    address: wallet.address, chainId: wallet.chain_id)
                let parsed = Self.parseApprovals(raw)
                await MainActor.run {
                    self.approvals = parsed
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

    // Opaque JSON -> typed rows. Tolerates { approvals: [...] } or bare array.
    static func parseApprovals(_ raw: [String: Any]) -> [Approval] {
        let arr: [[String: Any]]
        if let list = raw["approvals"] as? [[String: Any]] {
            arr = list
        } else if let list = raw["data"] as? [[String: Any]] {
            arr = list
        } else {
            arr = []
        }
        return arr.enumerated().map { idx, item in
            Approval(
                id: (item["id"] as? String) ?? (item["approval_id"] as? String) ?? "\(idx)",
                spender: (item["spender"] as? String) ?? "",
                contract: (item["contract_address"] as? String)
                    ?? (item["contract"] as? String)
                    ?? (item["token"] as? String) ?? "",
                symbol: (item["symbol"] as? String) ?? (item["token_symbol"] as? String) ?? "",
                amount: (item["amount"] as? String) ?? (item["allowance"] as? String) ?? "")
        }
    }

    private func revoke(_ approval: Approval) {
        revokingId = approval.id
        errorMessage = nil
        Task {
            do {
                _ = try await UserWalletApiService.shared.revokeApproval(approvalId: approval.id)
                await MainActor.run {
                    self.approvals.removeAll { $0.id == approval.id }
                    self.revokingId = nil
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.revokingId = nil
                }
            }
        }
    }
}
