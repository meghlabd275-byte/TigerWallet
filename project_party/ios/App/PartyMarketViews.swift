import SwiftUI

// MARK: - Market discovery (public)

struct MarketDiscoveryView: View {
    @State private var featured: [[String: Any]] = []
    @State private var trending: [[String: Any]] = []
    @State private var volume: [String: Any] = [:]
    @State private var liquidity: [String: Any] = [:]
    @State private var health: [String: Any] = [:]
    @State private var error: String?
    @State private var loading = true
    @State private var query = ""
    @State private var searchResults: [[String: Any]] = []
    @State private var searching = false

    var body: some View {
        NavigationStack {
            List {
                if let error {
                    Section { PartyErrorState(message: error) { await load() } }
                }

                Section {
                    HStack {
                        TextField("Search tokens", text: $query)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .onSubmit { Task { await search() } }
                        if searching { ProgressView() }
                        Button("Go") { Task { await search() } }
                            .disabled(query.isEmpty)
                    }
                }

                if !searchResults.isEmpty {
                    Section("Search results") {
                        ForEach(Array(searchResults.enumerated()), id: \.offset) { _, token in
                            NavigationLink {
                                TokenDetailView(tokenId: PartyFormat.value(token["id"]))
                            } label: {
                                TokenRow(token: token)
                            }
                        }
                    }
                }

                Section("Platform") {
                    if health.isEmpty {
                        Text("No health reported").foregroundStyle(.secondary)
                    } else {
                        DetailRow(label: "Status", value: PartyFormat.value(health["status"]))
                        DetailRow(label: "Service", value: PartyFormat.value(health["service"]))
                    }
                }

                Section("Analytics") {
                    if volume.isEmpty && liquidity.isEmpty {
                        Text("No analytics reported").foregroundStyle(.secondary)
                    } else {
                        ForEach(volume.keys.sorted(), id: \.self) { key in
                            DetailRow(label: "volume.\(key)", value: PartyFormat.value(volume[key]))
                        }
                        ForEach(liquidity.keys.sorted(), id: \.self) { key in
                            DetailRow(label: "liquidity.\(key)", value: PartyFormat.value(liquidity[key]))
                        }
                    }
                }

                Section("Featured") {
                    if featured.isEmpty {
                        Text("Nothing featured").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(featured.enumerated()), id: \.offset) { _, token in
                            NavigationLink {
                                TokenDetailView(tokenId: PartyFormat.value(token["id"]))
                            } label: {
                                TokenRow(token: token)
                            }
                        }
                    }
                }

                Section("Trending") {
                    if trending.isEmpty {
                        Text("Nothing trending").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(trending.enumerated()), id: \.offset) { _, token in
                            NavigationLink {
                                TokenDetailView(tokenId: PartyFormat.value(token["id"]))
                            } label: {
                                TokenRow(token: token)
                            }
                        }
                    }
                }
            }
            .navigationTitle("Market")
            .refreshable { await load() }
            .overlay { if loading && featured.isEmpty && trending.isEmpty { ProgressView() } }
            .task { await load() }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        var failures: [String] = []
        do { featured = PartyFormat.array(try await PartyApiService.shared.getFeatured(), keys: ["featured", "tokens"]) }
        catch { failures.append("featured: \(error.localizedDescription)") }
        do { trending = PartyFormat.array(try await PartyApiService.shared.getTrending(), keys: ["trending", "tokens"]) }
        catch { failures.append("trending: \(error.localizedDescription)") }
        do { volume = try await PartyApiService.shared.getTradingVolume() }
        catch { failures.append("volume: \(error.localizedDescription)") }
        do { liquidity = try await PartyApiService.shared.getLiquidity() }
        catch { failures.append("liquidity: \(error.localizedDescription)") }
        do { health = try await PartyApiService.shared.getHealth() }
        catch { failures.append("health: \(error.localizedDescription)") }
        if !failures.isEmpty { error = failures.joined(separator: "\n") }
    }

    private func search() async {
        searching = true
        defer { searching = false }
        do {
            let res = try await PartyApiService.shared.searchTokens(q: query)
            searchResults = PartyFormat.array(res, keys: ["tokens", "results"])
        } catch {
            error = "search: \(error.localizedDescription)"
            searchResults = []
        }
    }
}

struct TokenRow: View {
    let token: [String: Any]

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(PartyFormat.value(token["name"] ?? token["symbol"]))
                .font(.callout).bold()
            HStack(spacing: 8) {
                Text(PartyFormat.value(token["symbol"]))
                    .font(.caption).foregroundStyle(.secondary)
                Text(PartyFormat.value(token["status"]))
                    .font(.caption2)
                    .padding(.horizontal, 6).padding(.vertical, 2)
                    .background(.quaternary, in: Capsule())
            }
        }
    }
}

// MARK: - Tokens (authenticated CRUD + lifecycle)

struct TokensView: View {
    @State private var tokens: [[String: Any]] = []
    @State private var error: String?
    @State private var actionError: String?
    @State private var loading = true
    @State private var showCreate = false
    @State private var statusFilter: String?

    var body: some View {
        NavigationStack {
            List {
                if let actionError {
                    Section { Text(actionError).font(.callout).foregroundStyle(.red) }
                }
                if let error {
                    Section { PartyErrorState(message: error) { await load() } }
                }

                Section {
                    Picker("Status", selection: $statusFilter) {
                        Text("All").tag(String?.none)
                        Text("Draft").tag(String?.some("draft"))
                        Text("Submitted").tag(String?.some("submitted"))
                        Text("Listed").tag(String?.some("listed"))
                        Text("Rejected").tag(String?.some("rejected"))
                    }
                    .onChange(of: statusFilter) { _ in Task { await load() } }
                }

                if tokens.isEmpty && !loading {
                    Section {
                        PartyEmptyState(
                            title: "No tokens",
                            systemImage: "circle.grid.2x2",
                            detail: "Create a token to start the listing process."
                        )
                    }
                } else {
                    ForEach(Array(tokens.enumerated()), id: \.offset) { _, token in
                        NavigationLink {
                            TokenDetailView(tokenId: PartyFormat.value(token["id"]), onChanged: { await load() })
                        } label: {
                            TokenRow(token: token)
                        }
                    }
                    .onDelete { offsets in Task { await delete(offsets: offsets) } }
                }
            }
            .navigationTitle("Tokens")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button { showCreate = true } label: { Image(systemName: "plus") }
                }
            }
            .refreshable { await load() }
            .overlay { if loading && tokens.isEmpty { ProgressView() } }
            .task { await load() }
            .sheet(isPresented: $showCreate) { CreateTokenView { await load() } }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let res = try await PartyApiService.shared.listTokens(status: statusFilter)
            tokens = PartyFormat.array(res, keys: ["tokens"])
        } catch {
            self.error = error.localizedDescription
            tokens = []
        }
    }

    private func delete(offsets: IndexSet) async {
        for index in offsets {
            let id = PartyFormat.value(tokens[index]["id"])
            guard id != "—" else { continue }
            do { _ = try await PartyApiService.shared.deleteToken(id: id) }
            catch { actionError = "delete: \(error.localizedDescription)" }
        }
        await load()
    }
}

struct CreateTokenView: View {
    @Environment(\.dismiss) private var dismiss
    let onCreated: () async -> Void

    @State private var name = ""
    @State private var symbol = ""
    @State private var chain = "ethereum"
    @State private var contract = ""
    @State private var description = ""
    @State private var error: String?
    @State private var busy = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Token") {
                    TextField("Name", text: $name)
                    TextField("Symbol", text: $symbol)
                        .textInputAutocapitalization(.characters)
                        .autocorrectionDisabled()
                }
                Section("Deployment") {
                    TextField("Chain", text: $chain)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    TextField("Contract address", text: $contract)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    TextField("Description", text: $description, axis: .vertical)
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red).font(.callout) }
                }
            }
            .navigationTitle("New token")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") { Task { await create() } }
                        .disabled(busy || name.isEmpty || symbol.isEmpty || chain.isEmpty)
                }
            }
        }
    }

    private func create() async {
        busy = true
        error = nil
        defer { busy = false }
        // The backend owns status: a created token starts in `draft` and must
        // be submitted for review, so no status is sent from the client.
        var body: [String: Any] = ["name": name, "symbol": symbol, "chain": chain]
        if !contract.isEmpty { body["contract_address"] = contract }
        if !description.isEmpty { body["description"] = description }
        do {
            _ = try await PartyApiService.shared.createToken(body)
            await onCreated()
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
