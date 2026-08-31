import SwiftUI

// Token sales: list active sales (GET /token-sales) and participate
// (POST /token-sales/:id/participate).
struct TokenSalesView: View {
    @State private var sales: [[String: Any]] = []
    @State private var saleId = ""
    @State private var amount = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showSuccess = false
    @State private var successDetail = ""

    var body: some View {
        NavigationView {
            Form {
                Section("Active Sales") {
                    if sales.isEmpty {
                        Text(isLoading ? "Loading…" : "No active token sales")
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(Array(sales.enumerated()), id: \.offset) { _, s in
                            let id = (s["id"] ?? "?") as Any
                            let name = (s["name"] ?? s["token"] ?? "?") as Any
                            let price = (s["price"] ?? "?") as Any
                            let status = (s["status"] ?? "?") as Any
                            Text("• \(String(describing: id)): \(String(describing: name)) @ \(String(describing: price)) (\(String(describing: status)))")
                                .font(.caption.monospaced())
                        }
                    }
                }
                Section("Participate") {
                    TextField("Sale ID", text: $saleId)
                        .autocapitalization(.none).disableAutocorrection(true)
                    TextField("Amount", text: $amount).keyboardType(.decimalPad)
                    Button("Participate") { participate() }
                        .disabled(saleId.trimmingCharacters(in: .whitespaces).isEmpty
                                  || amount.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("Token Sales")
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
                let res = try await UserWalletApiService.shared.getTokenSales()
                let list = (res["sales"] as? [[String: Any]]) ?? (res["data"] as? [[String: Any]]) ?? []
                await MainActor.run { self.sales = list; self.isLoading = false }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }

    private func participate() {
        errorMessage = nil
        let id = saleId.trimmingCharacters(in: .whitespaces)
        let amt = amount.trimmingCharacters(in: .whitespaces)
        Task {
            do {
                let res = try await UserWalletApiService.shared.participateTokenSale(saleId: id, amount: amt)
                await MainActor.run {
                    let tx = String(describing: res["tx_hash"] ?? "")
                    self.successDetail = tx.isEmpty || tx == "<null>"
                        ? "Participation submitted"
                        : "Transaction submitted to the blockchain network: \(tx)"
                    self.showSuccess = true
                    self.amount = ""
                }
                load()
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}
