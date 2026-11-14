import SwiftUI

let BOT_TYPES = [
    "market_maker", "liquidity_provider", "sniper", "front_run", "mev",
    "sandwich", "flash_loan", "cross_chain", "perp_hedge", "grid",
    "dca", "momentum", "mean_reversion", "scalping", "ai_trading",
    "signal", "arbitrage", "custom",
]

struct BotsView: View {
    @State private var bots: [[String: Any]] = []
    @State private var error: String?
    @State private var loading = true
    @State private var showCreate = false
    @State private var busyId: String?
    @State private var actionError: String?

    var body: some View {
        NavigationStack {
            List {
                if let actionError {
                    Section {
                        Text(actionError)
                            .font(.callout)
                            .foregroundStyle(.red)
                    }
                }
                if let error {
                    Section { ErrorStateView(message: error) { await load() } }
                } else if bots.isEmpty && !loading {
                    Section {
                        EmptyStateView(
                            title: "No bots yet",
                            systemImage: "cpu",
                            detail: "Create a bot to start trading."
                        )
                    }
                } else {
                    ForEach(Array(bots.enumerated()), id: \.offset) { _, bot in
                        NavigationLink {
                            BotDetailView(botId: BotFormat.value(bot["id"]), onChanged: { await load() })
                        } label: {
                            BotRow(bot: bot, busy: busyId == (bot["id"] as? String)) { action in
                                await transition(bot: bot, action: action)
                            }
                        }
                    }
                    .onDelete { offsets in
                        Task { await delete(offsets: offsets) }
                    }
                }
            }
            .navigationTitle("Bots")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button { showCreate = true } label: { Image(systemName: "plus") }
                }
            }
            .refreshable { await load() }
            .overlay { if loading && bots.isEmpty { ProgressView() } }
            .task { await load() }
            .sheet(isPresented: $showCreate) {
                CreateBotView { await load() }
            }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            bots = BotFormat.array(try await BotApiService.shared.listBots(), keys: ["bots"])
        } catch {
            self.error = error.localizedDescription
            bots = []
        }
    }

    private func transition(bot: [String: Any], action: String) async {
        guard let id = bot["id"] as? String else { return }
        busyId = id
        actionError = nil
        defer { busyId = nil }
        do {
            switch action {
            case "start": _ = try await BotApiService.shared.startBot(id: id)
            case "stop": _ = try await BotApiService.shared.stopBot(id: id)
            case "pause": _ = try await BotApiService.shared.pauseBot(id: id)
            default: break
            }
            await load()
        } catch {
            // Fail-closed: the backend refuses to start bot types with no
            // execution runner, and that message is surfaced verbatim.
            actionError = "\(action) failed: \(error.localizedDescription)"
        }
    }

    private func delete(offsets: IndexSet) async {
        for index in offsets {
            guard let id = bots[index]["id"] as? String else { continue }
            do { _ = try await BotApiService.shared.deleteBot(id: id) }
            catch { actionError = "delete failed: \(error.localizedDescription)" }
        }
        await load()
    }
}

struct BotRow: View {
    let bot: [String: Any]
    let busy: Bool
    let onAction: (String) async -> Void

    private var status: String { (bot["status"] as? String ?? "unknown").lowercased() }

    private var statusColor: Color {
        switch status {
        case "running": return .green
        case "paused": return .orange
        case "stopped", "failed": return .red
        default: return .secondary
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(BotFormat.value(bot["name"])).font(.headline)
                Spacer()
                Text(status)
                    .font(.caption).bold()
                    .padding(.horizontal, 8).padding(.vertical, 3)
                    .background(statusColor.opacity(0.18), in: Capsule())
                    .foregroundStyle(statusColor)
            }
            Text("\(BotFormat.value(bot["bot_type"])) · \(BotFormat.value(bot["pair"]))")
                .font(.caption)
                .foregroundStyle(.secondary)
            if busy {
                ProgressView()
            } else {
                HStack(spacing: 8) {
                    if status != "running" {
                        Button("Start") { Task { await onAction("start") } }
                            .buttonStyle(.borderedProminent)
                            .controlSize(.small)
                    }
                    if status == "running" {
                        Button("Pause") { Task { await onAction("pause") } }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                    }
                    if status != "stopped" {
                        Button("Stop") { Task { await onAction("stop") } }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }
}

struct CreateBotView: View {
    @Environment(\.dismiss) private var dismiss
    let onCreated: () async -> Void

    @State private var name = ""
    @State private var botType = "grid"
    @State private var exchange = "binance"
    @State private var pair = "BTC/USDT"
    @State private var error: String?
    @State private var busy = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Bot") {
                    TextField("Name", text: $name)
                    Picker("Type", selection: $botType) {
                        ForEach(BOT_TYPES, id: \.self) { Text($0).tag($0) }
                    }
                }
                Section("Market") {
                    TextField("Exchange", text: $exchange)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    TextField("Pair", text: $pair)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red).font(.callout) }
                }
            }
            .navigationTitle("New bot")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create") { Task { await create() } }
                        .disabled(busy || name.isEmpty)
                }
            }
        }
    }

    private func create() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            _ = try await BotApiService.shared.createBot(
                name: name, botType: botType, exchange: exchange, pair: pair
            )
            await onCreated()
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

struct BotDetailView: View {
    let botId: String
    let onChanged: () async -> Void

    @State private var bot: [String: Any] = [:]
    @State private var executions: [[String: Any]] = []
    @State private var logs: [[String: Any]] = []
    @State private var error: String?
    @State private var loading = true

    var body: some View {
        List {
            if let error {
                Section { ErrorStateView(message: error) { await load() } }
            }

            Section("Configuration") {
                DetailRow(label: "Name", value: BotFormat.value(bot["name"]))
                DetailRow(label: "Type", value: BotFormat.value(bot["bot_type"]))
                DetailRow(label: "Status", value: BotFormat.value(bot["status"]))
                DetailRow(label: "Exchange", value: BotFormat.value(bot["exchange"]))
                DetailRow(label: "Pair", value: BotFormat.value(bot["pair"]))
                DetailRow(label: "Created", value: BotFormat.shortDate(bot["created_at"]))
                DetailRow(label: "Config", value: BotFormat.value(bot["config"]))
            }

            Section("Executions") {
                if executions.isEmpty {
                    Text("No executions recorded").foregroundStyle(.secondary)
                } else {
                    ForEach(Array(executions.enumerated()), id: \.offset) { _, exec in
                        VStack(alignment: .leading, spacing: 4) {
                            HStack {
                                Text(BotFormat.value(exec["status"])).font(.callout).bold()
                                Spacer()
                                Text("PnL \(BotFormat.value(exec["pnl"]))").font(.caption)
                            }
                            DetailRow(label: "Started", value: BotFormat.shortDate(exec["started_at"]))
                            DetailRow(label: "Ended", value: BotFormat.shortDate(exec["ended_at"]))
                        }
                    }
                }
            }

            Section("Logs") {
                if logs.isEmpty {
                    Text("No log entries").foregroundStyle(.secondary)
                } else {
                    ForEach(Array(logs.enumerated()), id: \.offset) { _, entry in
                        VStack(alignment: .leading, spacing: 2) {
                            Text(BotFormat.value(entry["message"] ?? entry["msg"]))
                                .font(.callout)
                            Text(BotFormat.shortDate(entry["timestamp"] ?? entry["created_at"]))
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
            }
        }
        .navigationTitle(BotFormat.value(bot["name"]))
        .refreshable { await load() }
        .overlay { if loading && bot.isEmpty { ProgressView() } }
        .task { await load() }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let detail = try await BotApiService.shared.getBot(id: botId)
            bot = detail["bot"] as? [String: Any] ?? detail
        } catch {
            self.error = error.localizedDescription
            return
        }
        do { executions = BotFormat.array(try await BotApiService.shared.listBotExecutions(id: botId), keys: ["executions"]) }
        catch { /* executions are optional detail; the config still renders */ }
        do { logs = BotFormat.array(try await BotApiService.shared.listBotLogs(id: botId), keys: ["logs"]) }
        catch { /* logs are optional detail */ }
    }
}
