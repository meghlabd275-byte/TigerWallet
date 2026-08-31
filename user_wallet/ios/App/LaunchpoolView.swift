import SwiftUI

// Launchpool staking: pool info + user stakes, stake/unstake with wallet
// password (POST /launchpool/stake|unstake — broadcast server-side).
struct LaunchpoolView: View {
    @State private var poolInfo = ""
    @State private var stakes: [[String: Any]] = []
    @State private var amount = ""
    @State private var password = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showSuccess = false
    @State private var successDetail = ""

    var body: some View {
        NavigationView {
            Form {
                Section("Pool") {
                    Text(poolInfo.isEmpty ? (isLoading ? "Loading…" : "Launchpool data unavailable") : poolInfo)
                        .font(.subheadline)
                }
                Section("My Stakes") {
                    if stakes.isEmpty {
                        Text("No active stakes").foregroundColor(.secondary)
                    } else {
                        ForEach(Array(stakes.enumerated()), id: \.offset) { _, s in
                            let id = (s["id"] ?? "?") as Any
                            let amt = (s["amount"] ?? "?") as Any
                            let token = (s["token"] ?? "") as Any
                            let status = (s["status"] ?? "?") as Any
                            Text("• \(String(describing: id)): \(String(describing: amt)) \(String(describing: token)) (\(String(describing: status)))")
                                .font(.caption.monospaced())
                        }
                    }
                }
                Section("Stake / Unstake") {
                    TextField("Amount", text: $amount).keyboardType(.decimalPad)
                    SecureField("Wallet password", text: $password)
                    HStack {
                        Button("Stake") { act(stake: true) }
                            .disabled(amount.trimmingCharacters(in: .whitespaces).isEmpty || password.isEmpty)
                        Button("Unstake") { act(stake: false) }
                            .disabled(amount.trimmingCharacters(in: .whitespaces).isEmpty || password.isEmpty)
                    }
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("Launchpool")
            .onAppear { load() }
            .alert(isPresented: $showSuccess) {
                Alert(title: Text("\u{2713} Submitted"),
                      message: Text(successDetail),
                      dismissButton: .default(Text("OK")))
            }
        }
    }

    private func load() {
        isLoading = true
        Task {
            do {
                let pool = try await UserWalletApiService.shared.getLaunchpool()
                let stakesRes = try await UserWalletApiService.shared.getLaunchpoolStakes()
                let stakeList = (stakesRes["data"] as? [[String: Any]]) ?? (stakesRes["stakes"] as? [[String: Any]]) ?? []
                let token = String(describing: pool["token"] ?? pool["asset"] ?? "?")
                let apy = String(describing: pool["apy"] ?? "?")
                let tvl = String(describing: pool["tvl"] ?? pool["total_staked"] ?? "?")
                await MainActor.run {
                    self.poolInfo = "Pool: \(token) | APY: \(apy)% | TVL: \(tvl)"
                    self.stakes = stakeList
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

    private func act(stake: Bool) {
        errorMessage = nil
        let amt = amount.trimmingCharacters(in: .whitespaces)
        let pw = password
        Task {
            do {
                let wallets = try await UserWalletApiService.shared.getWallets()
                guard let walletId = wallets.first?.id else {
                    await MainActor.run { self.errorMessage = "No wallet available" }
                    return
                }
                let res = stake
                    ? try await UserWalletApiService.shared.launchpoolStake(walletId: walletId, password: pw, amount: amt)
                    : try await UserWalletApiService.shared.launchpoolUnstake(walletId: walletId, password: pw, amount: amt)
                await MainActor.run {
                    let tx = String(describing: res["tx_hash"] ?? "")
                    self.successDetail = tx.isEmpty || tx == "<null>"
                        ? (stake ? "Stake submitted" : "Unstake submitted")
                        : "Transaction submitted to the blockchain network: \(tx)"
                    self.showSuccess = true
                    self.password = ""
                }
                load()
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}
