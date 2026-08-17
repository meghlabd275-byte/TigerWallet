import SwiftUI

// Keystore tab. Export a wallet's V3 keystore JSON via exportKeystore (shown
// with Copy), and import a keystore via importKeystore. Mirrors the web
// /keystore page. No mock data.
struct KeystoreView: View {
    @State private var mode: Mode = .export

    enum Mode: String, CaseIterable, Identifiable {
        case export, `import`
        var id: String { rawValue }
        var label: String { self == .export ? "Export" : "Import" }
    }

    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                Picker("Mode", selection: $mode) {
                    ForEach(Mode.allCases) { Text($0.label).tag($0) }
                }
                .pickerStyle(.segmented)
                .padding()

                if mode == .export {
                    ExportKeystoreView()
                } else {
                    ImportKeystoreView()
                }
            }
            .navigationTitle("Keystore")
        }
    }
}

struct ExportKeystoreView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var password = ""
    @State private var keystoreJSON: String?
    @State private var isExporting = false
    @State private var copied = false
    @State private var errorMessage: String?

    private var canExport: Bool {
        !isExporting && selectedWalletId != nil && password.count >= 8
    }

    var body: some View {
        Form {
            Section("Wallet") {
                if wallets.isEmpty {
                    Text("No wallets yet. Create or import one first.")
                        .foregroundColor(.secondary)
                } else {
                    Picker("Wallet", selection: $selectedWalletId) {
                        ForEach(wallets) { wallet in
                            Text("\(wallet.label) - \(wallet.address.prefix(8))...")
                                .tag(Optional(wallet.id))
                        }
                    }
                }
            }
            Section("Security") {
                SecureField("Wallet password", text: $password)
            }
            Section {
                Button(action: exportKeystore) {
                    HStack {
                        Text("Export Keystore")
                        Spacer()
                        if isExporting { ProgressView().tint(.orange) }
                    }
                }
                .disabled(!canExport)
            }
            if let errorMessage = errorMessage {
                Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
            }
            if let keystoreJSON = keystoreJSON {
                Section("Keystore JSON") {
                    Text(keystoreJSON)
                        .font(.system(.caption, design: .monospaced))
                        .textSelection(.enabled)
                    Button {
                        UIPasteboard.general.string = keystoreJSON
                        copied = true
                    } label: {
                        Label("Copy", systemImage: "doc.on.doc")
                    }
                    if copied {
                        Text("Copied!").font(.caption).foregroundColor(.green)
                    }
                }
            }
        }
        .onAppear { loadWallets() }
    }

    private func loadWallets() {
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    if self.selectedWalletId == nil { self.selectedWalletId = result.first?.id }
                }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func exportKeystore() {
        guard let id = selectedWalletId else { return }
        isExporting = true
        errorMessage = nil
        keystoreJSON = nil
        Task {
            do {
                let res = try await UserWalletApiService.shared.exportKeystore(
                    walletId: id, password: password)
                let json: String
                if let ks = res["keystore"] as? String {
                    json = ks
                } else if let data = try? JSONSerialization.data(
                    withJSONObject: res, options: .prettyPrinted),
                    let str = String(data: data, encoding: .utf8) {
                    json = str
                } else {
                    json = ""
                }
                await MainActor.run {
                    self.isExporting = false
                    self.keystoreJSON = json
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isExporting = false
                }
            }
        }
    }
}

struct ImportKeystoreView: View {
    @State private var keystore = ""
    @State private var password = ""
    @State private var label = ""
    @State private var isImporting = false
    @State private var resultMessage: String?
    @State private var errorMessage: String?

    private var canImport: Bool {
        !isImporting
            && !keystore.trimmingCharacters(in: .whitespaces).isEmpty
            && !password.trimmingCharacters(in: .whitespaces).isEmpty
    }

    var body: some View {
        Form {
            Section("Keystore JSON") {
                TextEditor(text: $keystore)
                    .frame(minHeight: 120)
                    .font(.system(.caption, design: .monospaced))
                    .autocapitalization(.none).disableAutocorrection(true)
            }
            Section("Details") {
                SecureField("Password", text: $password)
                TextField("Label (optional)", text: $label)
            }
            Section {
                Button(action: importKeystore) {
                    HStack {
                        Text("Import Keystore")
                        Spacer()
                        if isImporting { ProgressView().tint(.orange) }
                    }
                }
                .disabled(!canImport)
            }
            if let errorMessage = errorMessage {
                Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
            }
            if let resultMessage = resultMessage {
                Section {
                    Label(resultMessage, systemImage: "checkmark.circle.fill")
                        .foregroundColor(.green).font(.subheadline)
                }
            }
        }
    }

    private func importKeystore() {
        isImporting = true
        errorMessage = nil
        resultMessage = nil
        let ks = keystore.trimmingCharacters(in: .whitespacesAndNewlines)
        let pwd = password.trimmingCharacters(in: .whitespacesAndNewlines)
        let lbl = label.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.importKeystore(
                    keystore: ks, password: pwd, label: lbl.isEmpty ? nil : lbl)
                let address = (res["address"] as? String) ?? ""
                let id = (res["wallet_id"] as? String) ?? (res["id"] as? String) ?? ""
                await MainActor.run {
                    self.isImporting = false
                    var msg = "Wallet imported"
                    if !address.isEmpty { msg += " (\(address.prefix(10))...)" }
                    if !id.isEmpty { msg += " — id: \(id.prefix(8))" }
                    self.resultMessage = msg
                    self.keystore = ""
                    self.password = ""
                    self.label = ""
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isImporting = false
                }
            }
        }
    }
}
