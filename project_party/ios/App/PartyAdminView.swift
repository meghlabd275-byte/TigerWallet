import SwiftUI

// MARK: - Admin console (visible only to admin-role users)

struct PartyAdminView: View {
    @State private var tokens: [[String: Any]] = []
    @State private var listings: [[String: Any]] = []
    @State private var orders: [[String: Any]] = []
    @State private var error: String?
    @State private var actionError: String?
    @State private var loading = true

    private var pendingTokens: [[String: Any]] {
        tokens.filter {
            let s = ($0["status"] as? String ?? "").lowercased()
            return s == "submitted" || s == "pending" || s == "in_review"
        }
    }

    private var pendingListings: [[String: Any]] {
        listings.filter {
            let s = ($0["status"] as? String ?? "").lowercased()
            return s == "pending" || s == "submitted" || s == "draft"
        }
    }

    var body: some View {
        NavigationStack {
            List {
                if let actionError {
                    Section { Text(actionError).font(.callout).foregroundStyle(.red) }
                }
                if let error {
                    Section { PartyErrorState(message: error) { await load() } }
                }

                Section("Token review queue") {
                    if pendingTokens.isEmpty {
                        Text("Nothing awaiting review").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(pendingTokens.enumerated()), id: \.offset) { _, token in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(PartyFormat.value(token["name"])).font(.callout).bold()
                                PartyDetailRow(label: "Symbol", value: PartyFormat.value(token["symbol"]))
                                PartyDetailRow(label: "Contract", value: PartyFormat.value(token["contract_address"]))
                                HStack(spacing: 8) {
                                    Button("Approve") { Task { await approve(token) } }
                                        .buttonStyle(.borderedProminent)
                                        .controlSize(.small)
                                    Button("Reject") { Task { await reject(token) } }
                                        .buttonStyle(.bordered)
                                        .controlSize(.small)
                                    Button("Verify contract") { Task { await verifyContract(token) } }
                                        .buttonStyle(.bordered)
                                        .controlSize(.small)
                                }
                            }
                        }
                    }
                }

                Section("Listing queue") {
                    if pendingListings.isEmpty {
                        Text("No listings awaiting review").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(pendingListings.enumerated()), id: \.offset) { _, listing in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(PartyFormat.value(listing["token_symbol"] ?? listing["token_id"]))
                                    .font(.callout).bold()
                                PartyDetailRow(label: "Status", value: PartyFormat.value(listing["status"]))
                                Button("Feature listing") { Task { await feature(listing) } }
                                    .buttonStyle(.bordered)
                                    .controlSize(.small)
                            }
                        }
                    }
                }

                Section("All tokens") {
                    ForEach(Array(tokens.enumerated()), id: \.offset) { _, token in
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(PartyFormat.value(token["name"])).font(.callout)
                                Text(PartyFormat.value(token["status"]))
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                        }
                    }
                }

                Section("Market-maker orders") {
                    if orders.isEmpty {
                        Text("No orders").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(orders.enumerated()), id: \.offset) { _, order in
                            VStack(alignment: .leading, spacing: 3) {
                                Text("\(PartyFormat.value(order["side"])) \(PartyFormat.value(order["price"]))")
                                    .font(.callout)
                                PartyDetailRow(label: "Qty", value: PartyFormat.value(order["quantity"]))
                                PartyDetailRow(label: "Status", value: PartyFormat.value(order["status"]))
                            }
                        }
                    }
                }
            }
            .navigationTitle("Admin")
            .refreshable { await load() }
            .overlay { if loading && tokens.isEmpty { ProgressView() } }
            .task { await load() }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        var failures: [String] = []
        do { tokens = PartyFormat.array(try await PartyApiService.shared.listTokens(), keys: ["tokens"]) }
        catch { failures.append("tokens: \(error.localizedDescription)") }
        do { listings = PartyFormat.array(try await PartyApiService.shared.listListings(), keys: ["listings"]) }
        catch { failures.append("listings: \(error.localizedDescription)") }
        do { orders = PartyFormat.array(try await PartyApiService.shared.getMakerOrders(), keys: ["orders"]) }
        catch { failures.append("orders: \(error.localizedDescription)") }
        if !failures.isEmpty { error = failures.joined(separator: "\n") }
    }

    private func approve(_ token: [String: Any]) async {
        let id = PartyFormat.value(token["id"])
        guard id != "—" else { return }
        actionError = nil
        do { _ = try await PartyApiService.shared.approveToken(id: id); await load() }
        catch { actionError = error.localizedDescription }
    }

    private func reject(_ token: [String: Any]) async {
        let id = PartyFormat.value(token["id"])
        guard id != "—" else { return }
        actionError = nil
        do { _ = try await PartyApiService.shared.rejectToken(id: id); await load() }
        catch { actionError = error.localizedDescription }
    }

    private func verifyContract(_ token: [String: Any]) async {
        let id = PartyFormat.value(token["id"])
        guard id != "—" else { return }
        actionError = nil
        do {
            _ = try await PartyApiService.shared.requestAudit([
                "token_id": id,
                "contract_address": PartyFormat.value(token["contract_address"]),
            ])
            await load()
        } catch {
            actionError = error.localizedDescription
        }
    }

    private func feature(_ listing: [String: Any]) async {
        let id = PartyFormat.value(listing["id"])
        guard id != "—" else { return }
        actionError = nil
        do { _ = try await PartyApiService.shared.featureListing(id: id); await load() }
        catch { actionError = error.localizedDescription }
    }
}

// MARK: - Settings

struct PartySettingsView: View {
    @EnvironmentObject var session: PartySession
    @EnvironmentObject var themeManager: PartyThemeManager

    @State private var baseURL = PartyApiConfig.baseURL
    @State private var savedNote: String?
    @State private var health: [String: Any] = [:]
    @State private var healthError: String?
    @State private var fees: [String: Any] = [:]

    var body: some View {
        NavigationStack {
            Form {
                Section("Backend") {
                    TextField("API base URL", text: $baseURL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .keyboardType(.URL)
                    Button("Save endpoint") {
                        PartyApiConfig.baseURL = baseURL
                        baseURL = PartyApiConfig.baseURL
                        savedNote = "Saved"
                        Task { await checkHealth() }
                    }
                    if let savedNote {
                        Text(savedNote).font(.caption).foregroundStyle(.secondary)
                    }
                    if !health.isEmpty {
                        PartyDetailRow(label: "Health", value: PartyFormat.value(health["status"]))
                    } else if let healthError {
                        Text(healthError).font(.caption).foregroundStyle(.red)
                    }
                    Button("Test connection") { Task { await checkHealth() } }
                }

                Section("Appearance") {
                    Toggle("Dark mode", isOn: $themeManager.isDarkMode)
                }

                Section("Listing fees") {
                    if fees.isEmpty {
                        Text("No fee schedule published").foregroundStyle(.secondary)
                    } else {
                        ForEach(fees.keys.sorted(), id: \.self) { key in
                            PartyDetailRow(label: key, value: PartyFormat.value(fees[key]))
                        }
                    }
                }

                Section("Account") {
                    PartyDetailRow(label: "User", value: session.username ?? "—")
                    PartyDetailRow(label: "Role", value: session.role ?? "—")
                }

                Section {
                    Button("Sign out", role: .destructive) { session.signOut() }
                }
            }
            .navigationTitle("Settings")
            .task { await load() }
        }
    }

    private func load() async {
        await checkHealth()
        do { fees = try await PartyApiService.shared.getListingFees() }
        catch { /* fee schedule is informational */ }
    }

    private func checkHealth() async {
        healthError = nil
        do { health = try await PartyApiService.shared.getHealth() }
        catch {
            health = [:]
            healthError = error.localizedDescription
        }
    }
}
