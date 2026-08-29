import SwiftUI

// ENS: forward resolution (GET /ens/resolve) and reverse lookup
// (GET /ens/lookup) against the real on-chain ENS registry via the backend.
struct ENSView: View {
    @State private var name = ""
    @State private var address = ""
    @State private var resolveResult = ""
    @State private var lookupResult = ""
    @State private var isWorking = false

    var body: some View {
        NavigationView {
            Form {
                Section("Resolve Name → Address") {
                    TextField("name.eth", text: $name)
                        .autocapitalization(.none).disableAutocorrection(true)
                    Button("Resolve") { resolve() }
                        .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty || isWorking)
                    if !resolveResult.isEmpty {
                        Text(resolveResult).font(.caption.monospaced())
                    }
                }
                Section("Reverse Lookup Address → Name") {
                    TextField("0x…", text: $address)
                        .autocapitalization(.none).disableAutocorrection(true)
                        .font(.system(.body, design: .monospaced))
                    Button("Lookup") { lookup() }
                        .disabled(address.trimmingCharacters(in: .whitespaces).isEmpty || isWorking)
                    if !lookupResult.isEmpty {
                        Text(lookupResult).font(.caption.monospaced())
                    }
                }
            }
            .navigationTitle("ENS")
        }
    }

    private func resolve() {
        isWorking = true
        resolveResult = "Resolving…"
        let n = name.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.resolveENS(name: n)
                await MainActor.run {
                    self.resolveResult = res.address.isEmpty ? "No address found" : "\(n) → \(res.address)"
                    self.isWorking = false
                }
            } catch {
                await MainActor.run {
                    self.resolveResult = "Resolution failed: \(error.localizedDescription)"
                    self.isWorking = false
                }
            }
        }
    }

    private func lookup() {
        isWorking = true
        lookupResult = "Looking up…"
        let a = address.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.lookupENS(address: a)
                await MainActor.run {
                    self.lookupResult = res.name.isEmpty ? "No name found" : "\(a) → \(res.name)"
                    self.isWorking = false
                }
            } catch {
                await MainActor.run {
                    self.lookupResult = "Lookup failed: \(error.localizedDescription)"
                    self.isWorking = false
                }
            }
        }
    }
}
