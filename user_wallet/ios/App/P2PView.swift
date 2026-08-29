import SwiftUI

// P2P trading: browse adverts (GET /p2p/adverts) and create orders
// (POST /p2p/orders — KYC-gated backend-side; 403 surfaces to the user).
struct P2PView: View {
    @State private var adverts: [[String: Any]] = []
    @State private var advertId = ""
    @State private var amount = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showSuccess = false
    @State private var successDetail = ""

    var body: some View {
        NavigationView {
            Form {
                Section("Adverts") {
                    if adverts.isEmpty {
                        Text(isLoading ? "Loading…" : "No P2P adverts")
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(Array(adverts.enumerated()), id: \.offset) { _, ad in
                            let id = (ad["id"] ?? ad["advert_id"] ?? "") as Any
                            let asset = (ad["asset"] ?? ad["token"] ?? "") as Any
                            let price = (ad["price"] ?? "") as Any
                            Text("• \(String(describing: id)): \(String(describing: asset)) @ \(String(describing: price))")
                                .font(.caption.monospaced())
                        }
                    }
                }
                Section("Create Order") {
                    TextField("Advert ID", text: $advertId)
                        .autocapitalization(.none).disableAutocorrection(true)
                    TextField("Amount", text: $amount)
                        .keyboardType(.decimalPad)
                    Button("Create Order") { createOrder() }
                        .disabled(advertId.trimmingCharacters(in: .whitespaces).isEmpty
                                  || amount.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("P2P Trading")
            .onAppear { load() }
            .alert(isPresented: $showSuccess) {
                Alert(title: Text("\u{2713} Order submitted"),
                      message: Text(successDetail),
                      dismissButton: .default(Text("OK")))
            }
        }
    }

    private func load() {
        isLoading = true
        Task {
            do {
                let res = try await UserWalletApiService.shared.getP2PAdverts()
                let list = (res["adverts"] as? [[String: Any]]) ?? []
                await MainActor.run { self.adverts = list; self.isLoading = false }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }

    private func createOrder() {
        errorMessage = nil
        Task {
            do {
                let res = try await UserWalletApiService.shared.createP2POrder(body: [
                    "advert_id": advertId.trimmingCharacters(in: .whitespaces),
                    "amount": amount.trimmingCharacters(in: .whitespaces),
                ])
                await MainActor.run {
                    self.successDetail = String(describing: res["id"] ?? res["order_id"] ?? res)
                    self.showSuccess = true
                    self.amount = ""
                }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}
