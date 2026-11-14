import SwiftUI

// MARK: - Dashboard

struct DashboardView: View {
    @EnvironmentObject var session: BotSession
    @State private var bots: [[String: Any]] = []
    @State private var instances: [[String: Any]] = []
    @State private var subscription: [String: Any] = [:]
    @State private var health: BotHealth?
    @State private var error: String?
    @State private var loading = true

    private var counts: (total: Int, running: Int, paused: Int, stopped: Int) {
        var running = 0, paused = 0, stopped = 0
        for bot in bots {
            switch (bot["status"] as? String ?? "").lowercased() {
            case "running": running += 1
            case "paused": paused += 1
            case "stopped": stopped += 1
            default: break
            }
        }
        return (bots.count, running, paused, stopped)
    }

    var body: some View {
        NavigationStack {
            List {
                if let error {
                    Section { ErrorStateView(message: error) { await load() } }
                }

                Section("Bots") {
                    let c = counts
                    StatGrid(items: [
                        ("Total", "\(c.total)"),
                        ("Running", "\(c.running)"),
                        ("Paused", "\(c.paused)"),
                        ("Stopped", "\(c.stopped)"),
                    ])
                    .listRowInsets(EdgeInsets())
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

                Section("Running instances") {
                    if instances.isEmpty {
                        Text("No instances reported").foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(instances.enumerated()), id: \.offset) { _, inst in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(BotFormat.value(inst["bot_id"])).font(.callout).bold()
                                DetailRow(label: "Status", value: BotFormat.value(inst["status"]))
                                DetailRow(label: "Started", value: BotFormat.shortDate(inst["started_at"]))
                            }
                        }
                    }
                }

                Section("Backend") {
                    if let health {
                        DetailRow(label: "Status", value: health.status)
                        DetailRow(label: "Service", value: health.service)
                    } else {
                        Text("Health unavailable").foregroundStyle(.secondary)
                    }
                    DetailRow(label: "Endpoint", value: BotApiConfig.baseURL)
                }
            }
            .navigationTitle("Dashboard")
            .refreshable { await load() }
            .overlay { if loading && bots.isEmpty { ProgressView() } }
            .task { await load() }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        // Each call is independent: one failing endpoint must not blank the
        // whole dashboard, so failures are recorded and the rest still render.
        var failures: [String] = []
        do { bots = BotFormat.array(try await BotApiService.shared.listBots(), keys: ["bots"]) }
        catch { failures.append("bots: \(error.localizedDescription)") }
        do { instances = BotFormat.array(try await BotApiService.shared.listBotInstances(), keys: ["instances"]) }
        catch { failures.append("instances: \(error.localizedDescription)") }
        do { subscription = try await BotApiService.shared.getSubscription() }
        catch { failures.append("subscription: \(error.localizedDescription)") }
        do { health = try await BotApiService.shared.health() }
        catch { failures.append("health: \(error.localizedDescription)") }
        if !failures.isEmpty { error = failures.joined(separator: "\n") }
    }
}

struct StatGrid: View {
    let items: [(String, String)]

    var body: some View {
        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
            ForEach(Array(items.enumerated()), id: \.offset) { _, item in
                VStack(spacing: 4) {
                    Text(item.1).font(.title2).bold()
                    Text(item.0).font(.caption).foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 10)
                .background(.quaternary, in: RoundedRectangle(cornerRadius: 10))
            }
        }
        .padding(12)
    }
}
