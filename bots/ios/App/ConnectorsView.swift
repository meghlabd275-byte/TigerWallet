import SwiftUI

// MARK: - Exchange + DEX connectors, API keys

struct ConnectorsView: View {
    @State private var cex: [[String: Any]] = []
    @State private var dex: [[String: Any]] = []
    @State private var keys: [[String: Any]] = []
    @State private var error: String?
    @State private var actionError: String?
    @State private var loading = true
    @State private var showAddCEX = false
    @State private var showAddDEX = false
    @State private var showAddKey = false

    var body: some View {
        NavigationStack {
            List {
                if let actionError {
                    Section { Text(actionError).font(.callout).foregroundStyle(.red) }
                }
                if let error {
                    Section { ErrorStateView(message: error) { await load() } }
                }

                Section("Centralised exchanges") {
                    if cex.isEmpty {
                        Text("No CEX connectors").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(cex.enumerated()), id: \.offset) { index, conn in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(BotFormat.value(conn["name"])).font(.callout).bold()
                                DetailRow(label: "ID", value: BotFormat.value(conn["id"]))
                                DetailRow(label: "Active", value: BotFormat.value(conn["is_active"]))
                            }
                            .swipeActions {
                                Button("Remove", role: .destructive) {
                                    Task { await removeCEX(conn) }
                                }
                            }
                        }
                    }
                    Button("Add CEX connector") { showAddCEX = true }
                }

                Section("Decentralised exchanges") {
                    if dex.isEmpty {
                        Text("No DEX connectors").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(dex.enumerated()), id: \.offset) { _, conn in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(BotFormat.value(conn["name"])).font(.callout).bold()
                                DetailRow(label: "ID", value: BotFormat.value(conn["id"]))
                            }
                            .swipeActions {
                                Button("Remove", role: .destructive) {
                                    Task { await removeDEX(conn) }
                                }
                            }
                        }
                    }
                    Button("Add DEX connector") { showAddDEX = true }
                }

                Section("API keys") {
                    if keys.isEmpty {
                        Text("No API keys stored").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(keys.enumerated()), id: \.offset) { _, key in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(BotFormat.value(key["exchange"])).font(.callout).bold()
                                DetailRow(label: "Label", value: BotFormat.value(key["label"] ?? key["name"]))
                                DetailRow(label: "Created", value: BotFormat.shortDate(key["created_at"]))
                            }
                            .swipeActions {
                                Button("Delete", role: .destructive) {
                                    Task { await removeKey(key) }
                                }
                            }
                        }
                    }
                    Button("Add API key") { showAddKey = true }
                }
            }
            .navigationTitle("Connectors")
            .refreshable { await load() }
            .overlay { if loading && cex.isEmpty && dex.isEmpty && keys.isEmpty { ProgressView() } }
            .task { await load() }
            .sheet(isPresented: $showAddCEX) { AddConnectorView(kind: .cex) { await load() } }
            .sheet(isPresented: $showAddDEX) { AddConnectorView(kind: .dex) { await load() } }
            .sheet(isPresented: $showAddKey) { AddAPIKeyView { await load() } }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        var failures: [String] = []
        do { cex = BotFormat.array(try await BotApiService.shared.listCEX(), keys: ["connectors", "cex"]) }
        catch { failures.append("CEX: \(error.localizedDescription)") }
        do { dex = BotFormat.array(try await BotApiService.shared.listDEX(), keys: ["connectors", "dex"]) }
        catch { failures.append("DEX: \(error.localizedDescription)") }
        do { keys = BotFormat.array(try await BotApiService.shared.listAPIKeys(), keys: ["api_keys", "keys"]) }
        catch { failures.append("API keys: \(error.localizedDescription)") }
        if !failures.isEmpty { error = failures.joined(separator: "\n") }
    }

    private func removeCEX(_ conn: [String: Any]) async {
        guard let id = conn["id"] as? String else { return }
        do { _ = try await BotApiService.shared.removeCEX(id: id); await load() }
        catch { actionError = error.localizedDescription }
    }

    private func removeDEX(_ conn: [String: Any]) async {
        guard let id = conn["id"] as? String else { return }
        do { _ = try await BotApiService.shared.removeDEX(id: id); await load() }
        catch { actionError = error.localizedDescription }
    }

    private func removeKey(_ key: [String: Any]) async {
        guard let id = key["id"] as? String else { return }
        do { _ = try await BotApiService.shared.deleteAPIKey(id: id); await load() }
        catch { actionError = error.localizedDescription }
    }
}

struct AddConnectorView: View {
    enum Kind { case cex, dex }

    @Environment(\.dismiss) private var dismiss
    let kind: Kind
    let onAdded: () async -> Void

    @State private var name = ""
    @State private var endpoint = ""
    @State private var error: String?
    @State private var busy = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Connector") {
                    TextField("Name", text: $name)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    if kind == .dex {
                        TextField("RPC / endpoint (optional)", text: $endpoint)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                    }
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red).font(.callout) }
                }
            }
            .navigationTitle(kind == .cex ? "Add CEX" : "Add DEX")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Add") { Task { await add() } }
                        .disabled(busy || name.isEmpty)
                }
            }
        }
    }

    private func add() async {
        busy = true
        error = nil
        defer { busy = false }
        var config: [String: Any] = [:]
        if kind == .dex, !endpoint.isEmpty { config["endpoint"] = endpoint }
        do {
            if kind == .cex {
                _ = try await BotApiService.shared.addCEX(name: name, config: config)
            } else {
                _ = try await BotApiService.shared.addDEX(name: name, config: config)
            }
            await onAdded()
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

struct AddAPIKeyView: View {
    @Environment(\.dismiss) private var dismiss
    let onAdded: () async -> Void

    @State private var exchange = "binance"
    @State private var apiKey = ""
    @State private var error: String?
    @State private var busy = false

    var body: some View {
        NavigationStack {
            Form {
                Section("API key") {
                    TextField("Exchange", text: $exchange)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    SecureField("API key", text: $apiKey)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red).font(.callout) }
                }
            }
            .navigationTitle("Add API key")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Add") { Task { await add() } }
                        .disabled(busy || apiKey.isEmpty)
                }
            }
        }
    }

    private func add() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            _ = try await BotApiService.shared.createAPIKey(exchange: exchange, apiKey: apiKey)
            await onAdded()
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
