import SwiftUI

// Security Center: check URLs/addresses against the threat registry
// (/security/check-url|check-address) or run a full scan (/security/scan).
// Results come from the backend's live checkers; an empty threat list means
// "clean", not "unchecked".
struct SecurityView: View {
    @State private var target = ""
    @State private var checkResult = ""
    @State private var isWorking = false

    var body: some View {
        NavigationView {
            Form {
                Section("Check URL / Address") {
                    TextField("https://… or 0x…", text: $target)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        .font(.system(.body, design: .monospaced))
                    Button("Check") { check() }
                        .disabled(target.trimmingCharacters(in: .whitespaces).isEmpty)
                    Button("Full Scan") { scan() }
                        .disabled(target.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                if !checkResult.isEmpty {
                    Section("Result") {
                        Text(checkResult).font(.caption.monospaced())
                    }
                }
            }
            .navigationTitle("Security")
        }
    }

    private func check() {
        isWorking = true
        checkResult = "Checking…"
        let t = target.trimmingCharacters(in: .whitespacesAndNewlines)
        let isUrl = t.hasPrefix("http://") || t.hasPrefix("https://")
        Task {
            do {
                let res = isUrl
                    ? try await UserWalletApiService.shared.checkUrl(t)
                    : try await UserWalletApiService.shared.checkAddress(t)
                let safe = (res["safe"] as? Bool) ?? false
                let reason = (res["reason"] as? String) ?? (safe ? "no threats" : "threat detected")
                await MainActor.run {
                    self.checkResult = safe ? "✓ Safe: \(reason)" : "⚠ Flagged: \(reason)"
                    self.isWorking = false
                }
            } catch {
                await MainActor.run {
                    self.checkResult = "Check failed: \(error.localizedDescription)"
                    self.isWorking = false
                }
            }
        }
    }

    private func scan() {
        isWorking = true
        checkResult = "Scanning…"
        let t = target.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.securityScan(target: t)
                let threats = (res["threats"] as? [Any]) ?? []
                await MainActor.run {
                    if threats.isEmpty {
                        self.checkResult = "✓ Safe: no threats detected"
                    } else {
                        self.checkResult = "⚠ Threats: \(threats.map { String(describing: $0) }.joined(separator: "; "))"
                    }
                    self.isWorking = false
                }
            } catch {
                await MainActor.run {
                    self.checkResult = "Scan failed: \(error.localizedDescription)"
                    self.isWorking = false
                }
            }
        }
    }
}
