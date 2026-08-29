import SwiftUI

// Fiat on/off ramp: live providers (GET /ramp/providers) plus real on-ramp
// and off-ramp quotes (POST /ramp/quote, /ramp/offramp-quote). No fabricated
// rates — every value comes from the backend.
struct RampView: View {
    @State private var providers: [[String: Any]] = []
    @State private var providerId = ""
    @State private var amount = ""
    @State private var fiat = "USD"
    @State private var crypto = "ETH"
    @State private var quote: [String: Any]?
    @State private var isQuoting = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationView {
            Form {
                Section("Providers") {
                    if providers.isEmpty {
                        Text("No fiat providers configured").foregroundColor(.secondary)
                    } else {
                        ForEach(Array(providers.enumerated()), id: \.offset) { _, p in
                            let id = (p["id"] ?? p["provider_id"] ?? p["name"] ?? "") as Any
                            let name = (p["name"] ?? "") as Any
                            Text("• \(String(describing: id)): \(String(describing: name))")
                                .font(.caption.monospaced())
                        }
                    }
                }
                Section("Quote") {
                    TextField("Provider ID", text: $providerId)
                        .autocapitalization(.none).disableAutocorrection(true)
                    TextField("Amount", text: $amount)
                        .keyboardType(.decimalPad)
                    TextField("Fiat currency", text: $fiat)
                        .autocapitalization(.allCharacters)
                    TextField("Crypto", text: $crypto)
                        .autocapitalization(.allCharacters)
                    Button(action: { fetchQuote(offramp: false) }) {
                        HStack {
                            Text("Buy Quote")
                            Spacer()
                            if isQuoting { ProgressView().tint(.orange) }
                        }
                    }
                    .disabled(!canQuote)
                    Button("Sell Quote") { fetchQuote(offramp: true) }
                        .disabled(!canQuote)
                }
                if let quote = quote {
                    Section("Result") {
                        ForEach(quote.sorted(by: { $0.key < $1.key }), id: \.key) { key, value in
                            LabeledContent(key, value: String(describing: value))
                        }
                    }
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
            }
            .navigationTitle("Fiat Ramp")
            .onAppear { load() }
        }
    }

    private var canQuote: Bool {
        !providerId.trimmingCharacters(in: .whitespaces).isEmpty
            && !amount.trimmingCharacters(in: .whitespaces).isEmpty
            && !isQuoting
    }

    private func load() {
        Task {
            do {
                let res = try await UserWalletApiService.shared.getFiatProviders()
                let list = (res["providers"] as? [[String: Any]]) ?? []
                await MainActor.run {
                    self.providers = list
                    if self.providerId.isEmpty, let first = list.first {
                        self.providerId = String(describing: first["id"] ?? first["provider_id"] ?? first["name"] ?? "")
                    }
                }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func fetchQuote(offramp: Bool) {
        isQuoting = true
        errorMessage = nil
        quote = nil
        let pid = providerId.trimmingCharacters(in: .whitespaces)
        let amt = amount.trimmingCharacters(in: .whitespaces)
        let f = fiat.trimmingCharacters(in: .whitespaces).uppercased()
        let c = crypto.trimmingCharacters(in: .whitespaces).uppercased()
        Task {
            do {
                let res = offramp
                    ? try await UserWalletApiService.shared.getFiatOfframpQuote(providerId: pid, amount: amt, fiat: f, crypto: c)
                    : try await UserWalletApiService.shared.getFiatQuote(providerId: pid, amount: amt, fiat: f, crypto: c, method: "card")
                await MainActor.run { self.quote = res; self.isQuoting = false }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isQuoting = false
                }
            }
        }
    }
}
