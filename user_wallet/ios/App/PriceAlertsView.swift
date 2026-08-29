import SwiftUI

// Price alerts: list / create / delete via /price-alerts (the backend
// watch_alerts engine evaluates them against live prices).
struct PriceAlertsView: View {
    @State private var alerts: [[String: Any]] = []
    @State private var symbol = "ETH"
    @State private var target = ""
    @State private var direction = "above"
    @State private var errorMessage: String?

    private let directions = ["above", "below"]

    var body: some View {
        NavigationView {
            Form {
                Section("New Alert") {
                    TextField("Symbol", text: $symbol)
                        .autocapitalization(.allCharacters)
                    TextField("Target price (USD)", text: $target)
                        .keyboardType(.decimalPad)
                    Picker("Direction", selection: $direction) {
                        ForEach(directions, id: \.self) { Text($0) }
                    }
                    Button("Create Alert") { create() }
                        .disabled(target.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                Section("My Alerts") {
                    if alerts.isEmpty {
                        Text("No price alerts").foregroundColor(.secondary)
                    } else {
                        ForEach(Array(alerts.enumerated()), id: \.offset) { _, alert in
                            HStack {
                                let sym = alert["symbol"] ?? ""
                                let dir = alert["direction"] ?? ""
                                let tgt = alert["target_price"] ?? ""
                                Text("\(String(describing: sym)) \(String(describing: dir)) $\(String(describing: tgt))")
                                    .font(.caption.monospaced())
                                Spacer()
                                Button("Delete") {
                                    if let id = alert["id"] as? String { remove(id) }
                                    else if let id = alert["id"] { remove(String(describing: id)) }
                                }
                                .font(.caption)
                                .foregroundColor(.red)
                            }
                        }
                    }
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("Price Alerts")
            .onAppear { load() }
        }
    }

    private func load() {
        Task {
            do {
                let res = try await UserWalletApiService.shared.getPriceAlerts()
                let list = (res["alerts"] as? [[String: Any]]) ?? []
                await MainActor.run { self.alerts = list }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func create() {
        errorMessage = nil
        Task {
            do {
                _ = try await UserWalletApiService.shared.createPriceAlert(
                    symbol: symbol.trimmingCharacters(in: .whitespaces).uppercased(),
                    targetPrice: target.trimmingCharacters(in: .whitespaces),
                    direction: direction)
                await MainActor.run { self.target = ""; self.load() }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func remove(_ id: String) {
        Task {
            do {
                _ = try await UserWalletApiService.shared.deletePriceAlert(id: id)
                await MainActor.run { self.load() }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }
}
