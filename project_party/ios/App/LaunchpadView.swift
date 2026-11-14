import SwiftUI

// MARK: - Launchpad (public list, authenticated contribute/claim)

struct LaunchpadView: View {
    @State private var launchpads: [[String: Any]] = []
    @State private var favorites: [[String: Any]] = []
    @State private var error: String?
    @State private var loading = true
    @State private var showCreate = false

    var body: some View {
        NavigationStack {
            List {
                if let error {
                    Section { PartyErrorState(message: error) { await load() } }
                }

                if launchpads.isEmpty && !loading {
                    Section {
                        PartyEmptyState(
                            title: "No launchpads",
                            systemImage: "rocket",
                            detail: "Create a launchpad to raise for your token."
                        )
                    }
                } else {
                    ForEach(Array(launchpads.enumerated()), id: \.offset) { _, pad in
                        NavigationLink {
                            LaunchpadDetailView(padId: PartyFormat.value(pad["id"])) { await load() }
                        } label: {
                            LaunchpadRow(pad: pad)
                        }
                    }
                }

                if !favorites.isEmpty {
                    Section("Favorites") {
                        ForEach(Array(favorites.enumerated()), id: \.offset) { _, fav in
                            VStack(alignment: .leading, spacing: 3) {
                                Text(PartyFormat.value(fav["symbol"] ?? fav["token_id"]))
                                    .font(.callout).bold()
                                Text(PartyFormat.value(fav["note"]))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .swipeActions {
                                Button("Remove", role: .destructive) {
                                    Task { await removeFavorite(fav) }
                                }
                            }
                        }
                    }
                }
            }
            .navigationTitle("Launchpad")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button { showCreate = true } label: { Image(systemName: "plus") }
                }
            }
            .refreshable { await load() }
            .overlay { if loading && launchpads.isEmpty { ProgressView() } }
            .task { await load() }
            .sheet(isPresented: $showCreate) { CreateLaunchpadView { await load() } }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        var failures: [String] = []
        do {
            let res = try await PartyApiService.shared.listLaunchpads()
            launchpads = PartyFormat.array(res, keys: ["launchpads", "launchpad"])
        } catch { failures.append("launchpads: \(error.localizedDescription)") }
        // Favorites require auth; a failure here is not a launchpad failure.
        do {
            let res = try await PartyApiService.shared.listFavorites()
            favorites = PartyFormat.array(res, keys: ["favorites"])
        } catch { /* optional panel */ }
        if !failures.isEmpty { error = failures.joined(separator: "\n") }
    }

    private func removeFavorite(_ fav: [String: Any]) async {
        let id = PartyFormat.value(fav["id"])
        guard id != "—" else { return }
        do { _ = try await PartyApiService.shared.removeFavorite(id: id); await load() }
        catch { self.error = error.localizedDescription }
    }
}

struct LaunchpadRow: View {
    let pad: [String: Any]

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(PartyFormat.value(pad["name"] ?? pad["token_symbol"]))
                .font(.callout).bold()
            HStack(spacing: 8) {
                Text(PartyFormat.value(pad["status"]))
                    .font(.caption2)
                    .padding(.horizontal, 6).padding(.vertical, 2)
                    .background(.quaternary, in: Capsule())
                Text("Target \(PartyFormat.value(pad["target_amount"] ?? pad["hard_cap"]))")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }
}

struct LaunchpadDetailView: View {
    let padId: String
    let onChanged: () async -> Void

    @State private var pad: [String: Any] = [:]
    @State private var error: String?
    @State private var actionError: String?
    @State private var loading = true
    @State private var amount = ""
    @State private var busy = false

    private var status: String { (pad["status"] as? String ?? "").lowercased() }

    var body: some View {
        List {
            if let actionError {
                Section { Text(actionError).font(.callout).foregroundStyle(.red) }
            }
            if let error {
                Section { PartyErrorState(message: error) { await load() } }
            }

            Section("Launchpad") {
                PartyDetailRow(label: "Name", value: PartyFormat.value(pad["name"]))
                PartyDetailRow(label: "Token", value: PartyFormat.value(pad["token_symbol"]))
                PartyDetailRow(label: "Status", value: PartyFormat.value(pad["status"]))
                PartyDetailRow(label: "Target", value: PartyFormat.value(pad["target_amount"] ?? pad["hard_cap"]))
                PartyDetailRow(label: "Raised", value: PartyFormat.value(pad["raised_amount"] ?? pad["total_raised"]))
                PartyDetailRow(label: "Starts", value: PartyFormat.shortDate(pad["start_time"] ?? pad["start_at"]))
                PartyDetailRow(label: "Ends", value: PartyFormat.shortDate(pad["end_time"] ?? pad["end_at"]))
            }

            if status == "active" || status == "live" || status == "upcoming" {
                Section("Contribute") {
                    TextField("Amount", text: $amount)
                        .keyboardType(.decimalPad)
                    Button {
                        Task { await contribute() }
                    } label: {
                        HStack {
                            if busy { ProgressView() }
                            Text("Contribute")
                        }
                    }
                    .disabled(busy || amount.isEmpty)
                }
            }

            Section("Claim") {
                Button("Claim allocated tokens") { Task { await claim() } }
            }

            if status == "active" || status == "live" {
                Section {
                    Button("Cancel launchpad", role: .destructive) { Task { await cancel() } }
                }
            }
        }
        .navigationTitle(PartyFormat.value(pad["name"] ?? "Launchpad"))
        .refreshable { await load() }
        .overlay { if loading && pad.isEmpty { ProgressView() } }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let res = try await PartyApiService.shared.getLaunchpad(id: padId)
            pad = res["launchpad"] as? [String: Any] ?? res
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func contribute() async {
        busy = true
        actionError = nil
        defer { busy = false }
        do {
            _ = try await PartyApiService.shared.contribute(id: padId, amount: amount)
            amount = ""
            await load()
            await onChanged()
        } catch {
            actionError = error.localizedDescription
        }
    }

    private func claim() async {
        actionError = nil
        do {
            _ = try await PartyApiService.shared.claimTokens(id: padId)
            await load()
            await onChanged()
        } catch {
            actionError = error.localizedDescription
        }
    }

    private func cancel() async {
        actionError = nil
        do {
            _ = try await PartyApiService.shared.cancelLaunchpad(id: padId)
            await load()
            await onChanged()
        } catch {
            actionError = error.localizedDescription
        }
    }
}

struct CreateLaunchpadView: View {
    @Environment(\.dismiss) private var dismiss
    let onCreated: () async -> Void

    @State private var tokenId = ""
    @State private var name = ""
    @State private var target = ""
    @State private var error: String?
    @State private var busy = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Launchpad") {
                    TextField("Token ID", text: $tokenId)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    TextField("Name", text: $name)
                    TextField("Target amount", text: $target)
                        .keyboardType(.decimalPad)
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red).font(.callout) }
                }
            }
            .navigationTitle("New launchpad")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") { Task { await create() } }
                        .disabled(busy || tokenId.isEmpty || name.isEmpty || target.isEmpty)
                }
            }
        }
    }

    private func create() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            _ = try await PartyApiService.shared.createLaunchpad([
                "token_id": tokenId,
                "name": name,
                "target_amount": target,
            ])
            await onCreated()
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
