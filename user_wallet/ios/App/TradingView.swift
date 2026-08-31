import SwiftUI

// Trading: perpetual + margin positions, open and close.
// GET/POST /perpetual/positions (+:id/close), GET/POST /margin/positions (+:id/close).
struct TradingView: View {
    @State private var perpPositions: [[String: Any]] = []
    @State private var marginPositions: [[String: Any]] = []
    @State private var pair = ""
    @State private var side = "long"
    @State private var size = ""
    @State private var leverage = "1"
    @State private var positionId = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showSuccess = false
    @State private var successDetail = ""

    var body: some View {
        NavigationView {
            Form {
                Section("Perpetual Positions") {
                    positionList(perpPositions)
                }
                Section("Margin Positions") {
                    positionList(marginPositions)
                }
                Section("Open Position") {
                    TextField("Pair (e.g. ETH/USDT)", text: $pair)
                        .autocapitalization(.none).disableAutocorrection(true)
                    Picker("Side", selection: $side) {
                        Text("Long").tag("long")
                        Text("Short").tag("short")
                    }
                    TextField("Size", text: $size).keyboardType(.decimalPad)
                    TextField("Leverage", text: $leverage).keyboardType(.numberPad)
                    HStack {
                        Button("Open Perp") { open(perp: true) }.disabled(!canOpen)
                        Button("Open Margin") { open(perp: false) }.disabled(!canOpen)
                    }
                }
                Section("Close Position") {
                    TextField("Position ID", text: $positionId)
                        .autocapitalization(.none).disableAutocorrection(true)
                    HStack {
                        Button("Close Perp") { close(perp: true) }
                            .disabled(positionId.trimmingCharacters(in: .whitespaces).isEmpty)
                        Button("Close Margin") { close(perp: false) }
                            .disabled(positionId.trimmingCharacters(in: .whitespaces).isEmpty)
                    }
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("Trading")
            .onAppear { load() }
            .alert(isPresented: $showSuccess) {
                Alert(title: Text("\u{2713} Submitted"),
                      message: Text(successDetail),
                      dismissButton: .default(Text("OK")))
            }
        }
    }

    private var canOpen: Bool {
        !pair.trimmingCharacters(in: .whitespaces).isEmpty
            && !size.trimmingCharacters(in: .whitespaces).isEmpty
    }

    @ViewBuilder
    private func positionList(_ list: [[String: Any]]) -> some View {
        if list.isEmpty {
            Text(isLoading ? "Loading…" : "No open positions").foregroundColor(.secondary)
        } else {
            ForEach(Array(list.enumerated()), id: \.offset) { _, p in
                let id = (p["id"] ?? "?") as Any
                let pr = (p["pair"] ?? "?") as Any
                let sd = (p["side"] ?? "?") as Any
                let sz = (p["size"] ?? "?") as Any
                let lev = (p["leverage"] ?? "?") as Any
                let pnl = (p["pnl"] ?? p["unrealized_pnl"] ?? "?") as Any
                Text("• \(String(describing: id)): \(String(describing: pr)) \(String(describing: sd)) size:\(String(describing: sz)) \(String(describing: lev))x pnl:\(String(describing: pnl))")
                    .font(.caption.monospaced())
            }
        }
    }

    private func load() {
        isLoading = true
        Task {
            do {
                let perps = try await UserWalletApiService.shared.getPerpetualPositions()
                let margins = try await UserWalletApiService.shared.getMarginPositions()
                let perpList = (perps["positions"] as? [[String: Any]]) ?? (perps["data"] as? [[String: Any]]) ?? []
                let marginList = (margins["positions"] as? [[String: Any]]) ?? (margins["data"] as? [[String: Any]]) ?? []
                await MainActor.run {
                    self.perpPositions = perpList
                    self.marginPositions = marginList
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

    private func open(perp: Bool) {
        errorMessage = nil
        let lev = Int(leverage.trimmingCharacters(in: .whitespaces)) ?? 1
        Task {
            do {
                let res = perp
                    ? try await UserWalletApiService.shared.createPerpetualPosition(
                        pair: pair.trimmingCharacters(in: .whitespaces), side: side,
                        size: size.trimmingCharacters(in: .whitespaces), leverage: lev)
                    : try await UserWalletApiService.shared.createMarginPosition(
                        pair: pair.trimmingCharacters(in: .whitespaces), side: side,
                        size: size.trimmingCharacters(in: .whitespaces), leverage: lev)
                await MainActor.run {
                    let tx = String(describing: res["tx_hash"] ?? "")
                    self.successDetail = tx.isEmpty || tx == "<null>"
                        ? "Position opened: \(String(describing: res["id"] ?? "ok"))"
                        : "Transaction submitted to the blockchain network: \(tx)"
                    self.showSuccess = true
                    self.size = ""
                }
                load()
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func close(perp: Bool) {
        errorMessage = nil
        let id = positionId.trimmingCharacters(in: .whitespaces)
        Task {
            do {
                _ = perp
                    ? try await UserWalletApiService.shared.closePerpetualPosition(positionId: id)
                    : try await UserWalletApiService.shared.closeMarginPosition(positionId: id)
                await MainActor.run {
                    self.successDetail = "Position close submitted"
                    self.showSuccess = true
                    self.positionId = ""
                }
                load()
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}
