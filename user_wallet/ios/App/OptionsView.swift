import SwiftUI

/// Options engine — real /options/* backend (series list, live quote,
/// open buy/sell position, close position). No fabricated premiums or hashes.

struct OptionsView: View {
    @State private var series = ""
    @State private var quote = ""
    @State private var positions = ""
    @State private var seriesId = ""
    @State private var side = "buy"
    @State private var contracts = ""
    @State private var closeId = ""
    @State private var status = ""

    var body: some View {
        List {
            Section("Series") {
                Text(series)
                    .font(.caption.monospaced())
            }
            Section("Open position") {
                TextField("Series ID", text: $seriesId)
                    .textInputAutocapitalization(.never)
                TextField("Side (buy/sell)", text: $side)
                    .textInputAutocapitalization(.never)
                TextField("Contracts", text: $contracts)
                    .keyboardType(.decimalPad)
                Button("Open Options Position") {
                    Task { await open() }
                }
                Text(status).font(.caption)
            }
            Section("Quote") {
                Text(quote)
                    .font(.caption.monospaced())
            }
            Section("Positions") {
                Text(positions)
                    .font(.caption.monospaced())
                TextField("Position ID to close", text: $closeId)
                    .textInputAutocapitalization(.never)
                Button("Close Options Position") {
                    Task { await close() }
                }
            }
            .task { await load() }
        }
        .navigationTitle("Options Engine")
    }

    private func load() async {
        do {
            let api = UserWalletApiService.shared
            let s = try await api.getOptionsSeries()
            let arr = (s["series"] as? [[String: Any]]) ?? []
            series = arr.map { "• \($0["id"] ?? "?") \($0["underlying"] ?? "?")-\($0["strike"] ?? "?") \($0["style"] ?? "?") exp \($0["expiry_unix"] ?? "?")" }
                .joined(separator: "\n")
            if series.isEmpty { series = "No active options series. An operator must add series first." }
            let p = try await api.getOptionsPositions()
            let parr = (p["positions"] as? [[String: Any]]) ?? []
            positions = parr.map { "• \($0["id"] ?? "?") \($0["underlying"] ?? "?")-\($0["strike"] ?? "?") \($0["side"] ?? "?") x\($0["contracts"] ?? "?") \($0["status"] ?? "?") pnl:\($0["pnl"] ?? "?")" }
                .joined(separator: "\n")
            if positions.isEmpty { positions = "No open options positions" }
        } catch {
            series = "Options unavailable: \(error.localizedDescription)"
        }
    }

    private func open() async {
        guard !seriesId.isEmpty, !side.isEmpty, !contracts.isEmpty else {
            status = "Enter series id, side and contracts"
            return
        }
        status = "Opening options position…"
        do {
            let api = UserWalletApiService.shared
            let res = try await api.openOptionsPosition(seriesId: seriesId, side: side.lowercased(), contracts: contracts)
            let tx = res["tx_hash"] as? String ?? ""
            if !tx.isEmpty {
                status = "Transaction submitted to the blockchain network: \(tx)"
            } else {
                status = "Options position opened: \(res["id"] ?? "ok")"
            }
            await load()
        } catch {
            status = "Open failed: \(error.localizedDescription)"
        }
    }

    private func close() async {
        guard !closeId.isEmpty else {
            status = "Enter a position ID"
            return
        }
        status = "Closing options position…"
        do {
            let api = UserWalletApiService.shared
            _ = try await api.closeOptionsPosition(positionId: closeId)
            status = "Options position close submitted"
            await load()
        } catch {
            status = "Close failed: \(error.localizedDescription)"
        }
    }
}