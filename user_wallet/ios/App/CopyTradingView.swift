import SwiftUI

// Copy trading: list top traders (GET /copytrading/traders), follow and stop
// (POST /copytrading/follow, POST /copytrading/copiers/:id/stop).
struct CopyTradingView: View {
    @State private var traders: [[String: Any]] = []
    @State private var traderId = ""
    @State private var allocation = ""
    @State private var copierId = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showSuccess = false
    @State private var successDetail = ""

    var body: some View {
        NavigationView {
            Form {
                Section("Top Traders") {
                    if traders.isEmpty {
                        Text(isLoading ? "Loading…" : "No traders available")
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(Array(traders.enumerated()), id: \.offset) { _, t in
                            let id = (t["id"] ?? t["trader_id"] ?? "?") as Any
                            let name = (t["name"] ?? t["address"] ?? "?") as Any
                            let roi = (t["roi"] ?? "?") as Any
                            Text("• \(String(describing: id)): \(String(describing: name)) roi:\(String(describing: roi))%")
                                .font(.caption.monospaced())
                        }
                    }
                }
                Section("Follow Trader") {
                    TextField("Trader ID", text: $traderId)
                        .autocapitalization(.none).disableAutocorrection(true)
                    TextField("Allocation (optional)", text: $allocation)
                        .keyboardType(.decimalPad)
                    Button("Follow") { follow() }
                        .disabled(traderId.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                Section("Stop Copying") {
                    TextField("Copier ID", text: $copierId)
                        .autocapitalization(.none).disableAutocorrection(true)
                    Button("Stop Copying") { stop() }
                        .disabled(copierId.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("Copy Trading")
            .onAppear { load() }
            .alert(isPresented: $showSuccess) {
                Alert(title: Text("\u{2713} Done"),
                      message: Text(successDetail),
                      dismissButton: .default(Text("OK")))
            }
        }
    }

    private func load() {
        isLoading = true
        Task {
            do {
                let res = try await UserWalletApiService.shared.getCopyTraders()
                let list = (res["traders"] as? [[String: Any]]) ?? (res["data"] as? [[String: Any]]) ?? []
                await MainActor.run { self.traders = list; self.isLoading = false }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }

    private func follow() {
        errorMessage = nil
        let id = traderId.trimmingCharacters(in: .whitespaces)
        let alloc = allocation.trimmingCharacters(in: .whitespaces)
        Task {
            do {
                _ = try await UserWalletApiService.shared.followTrader(
                    traderId: id, allocation: alloc.isEmpty ? nil : alloc)
                await MainActor.run {
                    self.successDetail = "Now copying trader \(id)"
                    self.showSuccess = true
                }
                load()
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func stop() {
        errorMessage = nil
        let id = copierId.trimmingCharacters(in: .whitespaces)
        Task {
            do {
                _ = try await UserWalletApiService.shared.stopCopyTrader(copierId: id)
                await MainActor.run {
                    self.successDetail = "Stopped copying"
                    self.showSuccess = true
                    self.copierId = ""
                }
                load()
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}
