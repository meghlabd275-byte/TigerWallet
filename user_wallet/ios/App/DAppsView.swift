import SwiftUI

// dApps & WalletConnect: create pairings from a WalletConnect URI, list
// pairings (approve/reject), and list active sessions. All state comes from
// the canonical dapp_browser backend via the wallet_api proxy — no
// fabricated pairings/sessions.
struct DAppsView: View {
    @State private var uri = ""
    @State private var pairings: [[String: Any]] = []
    @State private var sessions: [[String: Any]] = []
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var successMessage: String?

    var body: some View {
        NavigationView {
            Form {
                Section("New Pairing") {
                    TextField("WalletConnect URI (wc:…)", text: $uri)
                        .autocapitalization(.none).disableAutocorrection(true)
                    Button("Pair") { pair() }
                        .disabled(uri.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                Section("Pairings") {
                    if pairings.isEmpty {
                        Text(isLoading ? "Loading…" : "No pairings")
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(Array(pairings.enumerated()), id: \.offset) { _, p in
                            let topic = (p["topic"] ?? "") as Any
                            let name = (p["peer_name"] ?? p["name"] ?? topic) as Any
                            let status = (p["status"] ?? "pending") as Any
                            VStack(alignment: .leading) {
                                Text("\(String(describing: name)) · \(String(describing: status))")
                                    .font(.caption.monospaced())
                                HStack {
                                    Button("Approve") { pairingAction(topic: String(describing: topic), approve: true) }
                                    Button("Reject") { pairingAction(topic: String(describing: topic), approve: false) }
                                }
                            }
                        }
                    }
                }
                Section("Sessions") {
                    if sessions.isEmpty {
                        Text("No active sessions").foregroundColor(.secondary)
                    } else {
                        ForEach(Array(sessions.enumerated()), id: \.offset) { _, s in
                            let topic = (s["topic"] ?? "") as Any
                            let name = (s["peer_name"] ?? s["name"] ?? topic) as Any
                            Text("\(String(describing: name)) · \(String(describing: topic))")
                                .font(.caption.monospaced())
                        }
                    }
                }
                Section {
                    Button("Refresh") { refresh() }
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red) }
                }
                if let successMessage = successMessage {
                    Section { Text(successMessage).foregroundColor(.green) }
                }
            }
            .navigationTitle("dApps & WalletConnect")
            .onAppear { refresh() }
        }
    }

    private func refresh() {
        loadPairings()
        loadSessions()
    }

    private func pair() {
        errorMessage = nil
        Task {
            do {
                _ = try await UserWalletApiService.shared.createDappPairing(uri: uri)
                await MainActor.run {
                    successMessage = "Pairing created — approve it below"
                    refresh()
                }
            } catch {
                await MainActor.run { errorMessage = "Pairing failed: \(error.localizedDescription)" }
            }
        }
    }

    private func loadPairings() {
        isLoading = true
        Task {
            do {
                let res = try await UserWalletApiService.shared.getDappPairings()
                await MainActor.run {
                    pairings = (res["pairings"] ?? res["data"] ?? []) as? [[String: Any]] ?? []
                    isLoading = false
                }
            } catch {
                await MainActor.run {
                    errorMessage = "Pairings unavailable: \(error.localizedDescription)"
                    isLoading = false
                }
            }
        }
    }

    private func loadSessions() {
        Task {
            do {
                let res = try await UserWalletApiService.shared.getDappSessions()
                await MainActor.run {
                    sessions = (res["sessions"] ?? res["data"] ?? []) as? [[String: Any]] ?? []
                }
            } catch {
                await MainActor.run { errorMessage = "Sessions unavailable: \(error.localizedDescription)" }
            }
        }
    }

    private func pairingAction(topic: String, approve: Bool) {
        errorMessage = nil
        Task {
            do {
                if approve {
                    _ = try await UserWalletApiService.shared.approveDappPairing(topic: topic)
                } else {
                    _ = try await UserWalletApiService.shared.rejectDappPairing(topic: topic)
                }
                await MainActor.run {
                    successMessage = approve ? "Pairing approved" : "Pairing rejected"
                    refresh()
                }
            } catch {
                await MainActor.run { errorMessage = "Action failed: \(error.localizedDescription)" }
            }
        }
    }
}
