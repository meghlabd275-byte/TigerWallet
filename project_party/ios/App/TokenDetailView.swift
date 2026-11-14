import SwiftUI

// MARK: - Token detail (config, pricing, listings, liquidity, compliance)

struct TokenDetailView: View {
    let tokenId: String
    var onChanged: (() async -> Void)?

    @State private var token: [String: Any] = [:]
    @State private var price: [String: Any] = [:]
    @State private var listing: [String: Any] = [:]
    @State private var liquidity: [String: Any] = [:]
    @State private var kyc: [String: Any] = [:]
    @State private var audit: [String: Any] = [:]
    @State private var error: String?
    @State private var actionError: String?
    @State private var loading = true
    @State private var showPrice = false
    @State private var showLiquidity = false
    @State private var showCompliance = false

    private var status: String { (token["status"] as? String ?? "").lowercased() }

    var body: some View {
        List {
            if let actionError {
                Section { Text(actionError).font(.callout).foregroundStyle(.red) }
            }
            if let error {
                Section { PartyErrorState(message: error) { await load() } }
            }

            Section("Token") {
                PartyDetailRow(label: "Name", value: PartyFormat.value(token["name"]))
                PartyDetailRow(label: "Symbol", value: PartyFormat.value(token["symbol"]))
                PartyDetailRow(label: "Chain", value: PartyFormat.value(token["chain"] ?? token["chain_id"]))
                PartyDetailRow(label: "Status", value: PartyFormat.value(token["status"]))
                PartyDetailRow(label: "Contract", value: PartyFormat.value(token["contract_address"]))
                PartyDetailRow(label: "Contract verified", value: PartyFormat.value(token["contract_verified"]))
                PartyDetailRow(label: "Featured", value: PartyFormat.value(token["is_featured"]))
            }

            Section("Pricing") {
                if price.isEmpty {
                    Text("No price published").foregroundStyle(.secondary)
                } else {
                    ForEach(price.keys.sorted(), id: \.self) { key in
                        PartyDetailRow(label: key, value: PartyFormat.value(price[key]))
                    }
                }
                Button("Update price") { showPrice = true }
            }

            Section("Listing") {
                if listing.isEmpty {
                    Text("Not listed").foregroundStyle(.secondary)
                } else {
                    PartyDetailRow(label: "Status", value: PartyFormat.value(listing["status"]))
                    PartyDetailRow(label: "Featured", value: PartyFormat.value(listing["is_featured"]))
                    PartyDetailRow(label: "Created", value: PartyFormat.shortDate(listing["created_at"]))
                }
            }

            Section("Liquidity") {
                if liquidity.isEmpty {
                    Text("No liquidity recorded").foregroundStyle(.secondary)
                } else {
                    ForEach(liquidity.keys.sorted(), id: \.self) { key in
                        PartyDetailRow(label: key, value: PartyFormat.value(liquidity[key]))
                    }
                }
                Button("Add liquidity") { showLiquidity = true }
            }

            Section("Compliance") {
                PartyDetailRow(label: "KYC", value: PartyFormat.value(kyc["status"]))
                PartyDetailRow(label: "Audit", value: PartyFormat.value(audit["status"]))
                Button("Submit compliance") { showCompliance = true }
            }

            Section("Lifecycle") {
                if status == "draft" {
                    Button("Submit for review") { Task { await lifecycle("submit") } }
                }
                if status != "listed" && status != "draft" {
                    Text("Awaiting administrator decision")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
                if status == "listed" {
                    Text("Listed on the public market")
                        .font(.callout)
                        .foregroundStyle(.green)
                }
            }
        }
        .navigationTitle(PartyFormat.value(token["name"] ?? token["symbol"]))
        .refreshable { await load() }
        .overlay { if loading && token.isEmpty { ProgressView() } }
        .task { await load() }
        .sheet(isPresented: $showPrice) { PriceUpdateView(tokenId: tokenId) { await load() } }
        .sheet(isPresented: $showLiquidity) { LiquidityView(tokenId: tokenId) { await load() } }
        .sheet(isPresented: $showCompliance) { ComplianceView(tokenId: tokenId) { await load() } }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let res = try await PartyApiService.shared.getToken(id: tokenId)
            token = res["token"] as? [String: Any] ?? res
        } catch {
            self.error = error.localizedDescription
            return
        }
        // Secondary panels are independent: one 404 must not hide the token.
        price = (try? await PartyApiService.shared.getTokenPrice(tokenId: tokenId)) ?? [:]
        listing = (try? await PartyApiService.shared.getListing(id: tokenId)) ?? [:]
        liquidity = (try? await PartyApiService.shared.getMarketMakerStatus(tokenId: tokenId)) ?? [:]
        kyc = (try? await PartyApiService.shared.getKYCStatus(tokenId: tokenId)) ?? [:]
        audit = (try? await PartyApiService.shared.getAuditStatus(tokenId: tokenId)) ?? [:]
    }

    private func lifecycle(_ action: String) async {
        actionError = nil
        do {
            switch action {
            case "submit": _ = try await PartyApiService.shared.submitToken(id: tokenId)
            default: break
            }
            await load()
            await onChanged?()
        } catch {
            actionError = error.localizedDescription
        }
    }
}

struct PriceUpdateView: View {
    @Environment(\.dismiss) private var dismiss
    let tokenId: String
    let onDone: () async -> Void

    @State private var price = ""
    @State private var error: String?
    @State private var busy = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Price") {
                    TextField("Price (USD)", text: $price)
                        .keyboardType(.decimalPad)
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red).font(.callout) }
                }
            }
            .navigationTitle("Update price")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") { Task { await save() } }
                        .disabled(busy || price.isEmpty)
                }
            }
        }
    }

    private func save() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            _ = try await PartyApiService.shared.setTokenPrice(tokenId: tokenId, price: price)
            await onDone()
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

struct LiquidityView: View {
    @Environment(\.dismiss) private var dismiss
    let tokenId: String
    let onDone: () async -> Void

    @State private var amount = ""
    @State private var action = "add"
    @State private var error: String?
    @State private var busy = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Liquidity") {
                    Picker("Action", selection: $action) {
                        Text("Add").tag("add")
                        Text("Remove").tag("remove")
                    }
                    .pickerStyle(.segmented)
                    TextField("Amount", text: $amount)
                        .keyboardType(.decimalPad)
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red).font(.callout) }
                }
            }
            .navigationTitle("Liquidity")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit") { Task { await submit() } }
                        .disabled(busy || amount.isEmpty)
                }
            }
        }
    }

    private func submit() async {
        busy = true
        error = nil
        defer { busy = false }
        let body: [String: Any] = ["token_id": tokenId, "amount": amount]
        do {
            if action == "add" {
                _ = try await PartyApiService.shared.addLiquidity(body)
            } else {
                _ = try await PartyApiService.shared.removeLiquidity(body)
            }
            await onDone()
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

struct ComplianceView: View {
    @Environment(\.dismiss) private var dismiss
    let tokenId: String
    let onDone: () async -> Void

    @State private var jurisdiction = ""
    @State private var entityName = ""
    @State private var error: String?
    @State private var busy = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Compliance submission") {
                    TextField("Legal entity", text: $entityName)
                    TextField("Jurisdiction", text: $jurisdiction)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }
                if let error {
                    Section { Text(error).foregroundStyle(.red).font(.callout) }
                }
            }
            .navigationTitle("Compliance")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit") { Task { await submit() } }
                        .disabled(busy || entityName.isEmpty)
                }
            }
        }
    }

    private func submit() async {
        busy = true
        error = nil
        defer { busy = false }
        var body: [String: Any] = ["token_id": tokenId, "entity_name": entityName]
        if !jurisdiction.isEmpty { body["jurisdiction"] = jurisdiction }
        do {
            _ = try await PartyApiService.shared.submitKYC(body)
            _ = try? await PartyApiService.shared.requestAudit(["token_id": tokenId])
            await onDone()
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
