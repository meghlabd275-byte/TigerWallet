import SwiftUI

// MARK: - Admin console (visible only to admin-role users)

struct AdminView: View {
    @State private var stats: [String: Any] = [:]
    @State private var users: [[String: Any]] = []
    @State private var feeAddresses: [[String: Any]] = []
    @State private var allBots: [[String: Any]] = []
    @State private var feeConfigs: [[String: Any]] = []
    @State private var error: String?
    @State private var actionError: String?
    @State private var loading = true

    var body: some View {
        NavigationStack {
            List {
                if let actionError {
                    Section { Text(actionError).font(.callout).foregroundStyle(.red) }
                }
                if let error {
                    Section { ErrorStateView(message: error) { await load() } }
                }

                Section("Platform statistics") {
                    if stats.isEmpty {
                        Text("No statistics reported").foregroundStyle(.secondary)
                    } else {
                        ForEach(stats.keys.sorted(), id: \.self) { key in
                            DetailRow(label: key, value: BotFormat.value(stats[key]))
                        }
                    }
                }

                Section("Users") {
                    if users.isEmpty {
                        Text("No users").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(users.enumerated()), id: \.offset) { _, user in
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(BotFormat.value(user["username"])).font(.callout).bold()
                                    Text(BotFormat.value(user["role"]))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Button(isActive(user) ? "Deactivate" : "Activate") {
                                    Task { await toggle(user) }
                                }
                                .buttonStyle(.bordered)
                                .controlSize(.small)
                            }
                        }
                    }
                }

                Section("All bots") {
                    if allBots.isEmpty {
                        Text("No bots").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(allBots.enumerated()), id: \.offset) { _, bot in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(BotFormat.value(bot["name"])).font(.callout).bold()
                                DetailRow(label: "Status", value: BotFormat.value(bot["status"]))
                                DetailRow(label: "Type", value: BotFormat.value(bot["bot_type"]))
                            }
                            .swipeActions {
                                Button("Start") { Task { await setStatus(bot, "running") } }
                                Button("Stop", role: .destructive) { Task { await setStatus(bot, "stopped") } }
                            }
                        }
                    }
                }

                Section("Fee addresses") {
                    if feeAddresses.isEmpty {
                        Text("No fee addresses configured").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(feeAddresses.enumerated()), id: \.offset) { _, addr in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(BotFormat.value(addr["label"])).font(.callout).bold()
                                DetailRow(label: "Chain", value: BotFormat.value(addr["chain_id"]))
                                DetailRow(label: "Address", value: BotFormat.value(addr["address"]))
                            }
                            .swipeActions {
                                Button("Delete", role: .destructive) {
                                    Task { await deleteFeeAddress(addr) }
                                }
                            }
                        }
                    }
                }

                Section("Fee configuration") {
                    if feeConfigs.isEmpty {
                        Text("No fee configuration").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(feeConfigs.enumerated()), id: \.offset) { _, cfg in
                            DetailRow(label: BotFormat.value(cfg["label"] ?? cfg["bot_type"]),
                                      value: BotFormat.value(cfg["fee_bps"] ?? cfg["value"]))
                        }
                    }
                }
            }
            .navigationTitle("Admin")
            .refreshable { await load() }
            .overlay { if loading && stats.isEmpty && users.isEmpty { ProgressView() } }
            .task { await load() }
        }
    }

    private func isActive(_ user: [String: Any]) -> Bool {
        (user["is_active"] as? Bool) ?? (user["active"] as? Bool) ?? false
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        var failures: [String] = []
        do { stats = try await BotApiService.shared.adminStats() }
        catch { failures.append("stats: \(error.localizedDescription)") }
        do { users = BotFormat.array(try await BotApiService.shared.adminListUsers(), keys: ["users"]) }
        catch { failures.append("users: \(error.localizedDescription)") }
        do { allBots = BotFormat.array(try await BotApiService.shared.listBots(), keys: ["bots"]) }
        catch { failures.append("bots: \(error.localizedDescription)") }
        do { feeAddresses = BotFormat.array(try await BotApiService.shared.adminGetFeeAddresses(), keys: ["fee_addresses", "addresses"]) }
        catch { failures.append("fee addresses: \(error.localizedDescription)") }
        do { feeConfigs = BotFormat.array(try await BotApiService.shared.getFeeConfigs(), keys: ["fees", "fee_configs"]) }
        catch { failures.append("fee config: \(error.localizedDescription)") }
        if !failures.isEmpty { error = failures.joined(separator: "\n") }
    }

    private func toggle(_ user: [String: Any]) async {
        guard let id = user["id"] as? String else { return }
        do {
            _ = try await BotApiService.shared.adminUserStatus(id: id, active: !isActive(user))
            await load()
        } catch {
            actionError = error.localizedDescription
        }
    }

    private func setStatus(_ bot: [String: Any], _ status: String) async {
        guard let id = bot["id"] as? String else { return }
        do {
            _ = try await BotApiService.shared.adminBotStatus(id: id, status: status)
            await load()
        } catch {
            actionError = error.localizedDescription
        }
    }

    private func deleteFeeAddress(_ addr: [String: Any]) async {
        guard let id = addr["id"] as? String else { return }
        do {
            _ = try await BotApiService.shared.adminDeleteFeeAddress(id: id)
            await load()
        } catch {
            actionError = error.localizedDescription
        }
    }
}

// MARK: - Settings

struct SettingsView: View {
    @EnvironmentObject var session: BotSession
    @EnvironmentObject var themeManager: ThemeManager

    @State private var baseURL = BotApiConfig.baseURL
    @State private var savedNote: String?
    @State private var tiers: [[String: Any]] = []
    @State private var subscription: [String: Any] = [:]
    @State private var health: BotHealth?
    @State private var healthError: String?

    var body: some View {
        NavigationStack {
            Form {
                Section("Backend") {
                    TextField("API base URL", text: $baseURL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .keyboardType(.URL)
                    Button("Save endpoint") {
                        BotApiConfig.baseURL = baseURL
                        baseURL = BotApiConfig.baseURL
                        savedNote = "Saved"
                        Task { await checkHealth() }
                    }
                    if let savedNote {
                        Text(savedNote).font(.caption).foregroundStyle(.secondary)
                    }
                    if let health {
                        DetailRow(label: "Health", value: "\(health.status) · \(health.service)")
                    } else if let healthError {
                        Text(healthError).font(.caption).foregroundStyle(.red)
                    }
                    Button("Test connection") { Task { await checkHealth() } }
                }

                Section("Appearance") {
                    Toggle("Dark mode", isOn: $themeManager.isDarkMode)
                }

                Section("Subscription") {
                    if subscription.isEmpty {
                        Text("No active subscription").foregroundStyle(.secondary)
                    } else {
                        DetailRow(label: "Tier", value: BotFormat.value(subscription["tier"]))
                        DetailRow(label: "Status", value: BotFormat.value(subscription["status"]))
                        DetailRow(label: "Expires", value: BotFormat.shortDate(subscription["expires_at"]))
                    }
                }

                Section("Available tiers") {
                    if tiers.isEmpty {
                        Text("No tiers published").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(tiers.enumerated()), id: \.offset) { _, tier in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(BotFormat.value(tier["name"] ?? tier["tier"])).font(.callout).bold()
                                DetailRow(label: "Price", value: BotFormat.value(tier["price"] ?? tier["price_usd"]))
                                DetailRow(label: "Max bots", value: BotFormat.value(tier["max_bots"]))
                            }
                        }
                    }
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
        do { tiers = BotFormat.array(try await BotApiService.shared.publicTiers(), keys: ["tiers"]) }
        catch { /* tiers are informational */ }
        do { subscription = try await BotApiService.shared.getSubscription() }
        catch { /* absence of a subscription is a valid state */ }
    }

    private func checkHealth() async {
        healthError = nil
        do { health = try await BotApiService.shared.health() }
        catch {
            health = nil
            healthError = error.localizedDescription
        }
    }
}
