import SwiftUI

// Send tab. Lets the user broadcast a real on-chain transaction via
// UserWalletApiService.shared.sendTransaction / autoSendTransaction.
//
// The recipient field accepts a raw 0x address OR an ENS name (name.eth) —
// ENS names are resolved via GET /ens/resolve and the resolved address is
// shown inline. Before signing, the "Simulate" action dry-runs the exact
// transaction via POST /simulate (success / will-revert / gas estimate).
// Advanced gas (EIP-1559 max fee + priority fee, gwei) can optionally be
// overridden; empty fields are omitted from the request ("auto").
//
// On a successful send an alert is shown:
//   "<checkmark> Transaction submitted to the blockchain network"
// together with the returned tx hash.
struct SendView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var chainId: Int = 1
    @State private var recipient = ""
    @State private var amount = ""
    @State private var password = ""
    @State private var isSending = false
    @State private var errorMessage: String?

    // ENS resolution state: `resolvedAddress` is the 0x address actually sent
    // to (raw recipient, or the ENS lookup result for .eth names).
    @State private var resolvedAddress = ""
    @State private var ensName = ""
    @State private var isResolving = false

    // Optional EIP-1559 gas overrides (gwei strings; empty = auto).
    @State private var maxFeeGwei = ""
    @State private var maxPriorityGwei = ""
    @State private var showAdvancedGas = false

    // Pre-send simulation state (POST /simulate).
    @State private var simulation: UserWalletApiService.SimulationResult?
    @State private var isSimulating = false

    // Success alert state.
    @State private var showSuccess = false
    @State private var successTxHash = ""
    @State private var successAutoApproved = false
    @State private var successAutoApprovalReason = ""

    // Passwordless unlock: an unlock_token obtained via `unlockWallet` (passcode)
    // so the wallet can be signed for without re-entering its raw password. When
    // present, the password field becomes optional and `unlockToken` is forwarded
    // to `sendTransaction` / `autoSendTransaction`.
    @State private var unlockToken: String?
    @State private var unlockPasscode = ""
    @State private var isUnlocking = false
    @State private var unlockError: String?

    private let chains: [(name: String, id: Int)] = [
        ("Ethereum", 1),
        ("BNB Chain", 56),
        ("Polygon", 137),
    ]

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    /// True once the recipient has been resolved (or validated) to a raw
    /// 0x address — the only value ever sent to the backend.
    private var hasValidRecipient: Bool {
        Self.isEthAddress(resolvedAddress)
    }

    private var canSend: Bool {
        guard let w = selectedWallet else { return false }
        return !isSending
            && hasValidRecipient
            && !amount.isEmpty
            && (unlockToken != nil || password.count >= 8)
            && w.chain_id == chainId
    }

    /// Canonical 0x-address check (40 hex chars), used for both raw recipients
    /// and ENS resolution results.
    private static func isEthAddress(_ value: String) -> Bool {
        value.range(of: "^0x[a-fA-F0-9]{40}$", options: .regularExpression) != nil
    }

    var body: some View {
        NavigationView {
            Form {
                Section("From") {
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
                        .onChange(of: selectedWalletId) { newValue in
                            if let id = newValue, let w = wallets.first(where: { $0.id == id }) {
                                chainId = w.chain_id
                            }
                            // A different wallet invalidates any cached unlock token.
                            unlockToken = nil
                            unlockError = nil
                        }
                    }
                }

                Section("Network") {
                    Picker("Chain", selection: $chainId) {
                        ForEach(chains, id: \.id) { chain in
                            Text(chain.name).tag(chain.id)
                        }
                    }
                    if let w = selectedWallet {
                        Text("Address: \(w.address.prefix(12))...")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                Section("Transaction") {
                    TextField("Recipient (0x… or name.eth)", text: $recipient)
                        .keyboardType(.asciiCapable)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        .onChange(of: recipient) { _ in
                            resolveRecipient()
                        }
                    if isResolving {
                        HStack(spacing: 8) {
                            ProgressView()
                            Text("Resolving ENS…")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }
                    if !ensName.isEmpty && hasValidRecipient {
                        Label("\(ensName) → \(resolvedAddress.prefix(10))…\(resolvedAddress.suffix(6))",
                              systemImage: "checkmark.circle.fill")
                            .font(.caption)
                            .foregroundColor(.green)
                            .lineLimit(1)
                            .truncationMode(.middle)
                    } else if hasValidRecipient && !Self.isEthAddress(recipient.trimmingCharacters(in: .whitespacesAndNewlines)) && !recipient.isEmpty {
                        // Resolved via ENS but the user edited the field after
                        // resolution — force re-resolution before sending.
                        Text("Resolve the ENS name to continue")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    TextField("Amount", text: $amount)
                        .keyboardType(.decimalPad)
                }

                Section {
                    Button {
                        withAnimation { showAdvancedGas.toggle() }
                    } label: {
                        HStack {
                            Image(systemName: "fuelpump")
                            Text("Advanced gas (EIP-1559)")
                            Spacer()
                            Image(systemName: showAdvancedGas ? "chevron.up" : "chevron.down")
                                .foregroundColor(.secondary)
                        }
                    }
                    if showAdvancedGas {
                        HStack {
                            Text("Max fee (gwei)")
                            Spacer()
                            TextField("auto", text: $maxFeeGwei)
                                .keyboardType(.decimalPad)
                                .multilineTextAlignment(.trailing)
                        }
                        HStack {
                            Text("Priority fee (gwei)")
                            Spacer()
                            TextField("auto", text: $maxPriorityGwei)
                                .keyboardType(.decimalPad)
                                .multilineTextAlignment(.trailing)
                        }
                    }
                }

                Section {
                    Button(action: simulate) {
                        HStack {
                            Image(systemName: "testtube.2")
                            Text("Simulate")
                            Spacer()
                            if isSimulating {
                                ProgressView().tint(.orange)
                            }
                        }
                    }
                    .disabled(selectedWallet == nil || !hasValidRecipient || amount.isEmpty || isSimulating)
                }

                if let sim = simulation {
                    Section("Simulation") {
                        if sim.success && !(sim.will_revert ?? false) {
                            Label("Simulation succeeded", systemImage: "checkmark.circle.fill")
                                .foregroundColor(.green)
                        } else {
                            Label("Transaction will revert", systemImage: "xmark.octagon.fill")
                                .foregroundColor(.red)
                            if let reason = sim.revert_reason, !reason.isEmpty {
                                Text(reason)
                                    .font(.caption)
                                    .foregroundColor(.red)
                            }
                        }
                        if let gas = sim.gas_estimate {
                            HStack {
                                Text("Gas estimate")
                                Spacer()
                                Text("\(gas)")
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundColor(.secondary)
                            }
                        }
                        if let cost = sim.estimated_cost_wei, !cost.isEmpty {
                            HStack {
                                Text("Estimated cost")
                                Spacer()
                                Text("\(cost) wei")
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundColor(.secondary)
                            }
                        }
                    }
                }

                Section("Security") {
                    SecureField("Wallet password", text: $password)
                        .disabled(unlockToken != nil)
                    if unlockToken != nil {
                        Label("Unlocked (passwordless) — password not required", systemImage: "checkmark.lock.fill")
                            .font(.caption)
                            .foregroundColor(.green)
                    }
                }

                Section {
                    Button(action: unlockWallet) {
                        HStack {
                            Image(systemName: "key.fill")
                            Text(unlockToken == nil ? "Unlock Wallet (passwordless)" : "Re-unlock Wallet")
                            Spacer()
                            if isUnlocking {
                                ProgressView().tint(.orange)
                            }
                        }
                    }
                    .disabled(isUnlocking || selectedWallet == nil || (unlockToken == nil && unlockPasscode.trimmingCharacters(in: .whitespaces).isEmpty))
                    if unlockToken == nil {
                        SecureField("Passcode", text: $unlockPasscode)
                            .keyboardType(.numberPad)
                    }
                    if let unlockError = unlockError {
                        Text(unlockError)
                            .foregroundColor(.red)
                            .font(.subheadline)
                    }
                }

                if let errorMessage = errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundColor(.red)
                            .font(.subheadline)
                    }
                }

                Section {
                    Button(action: autoSend) {
                        HStack {
                            Image(systemName: "bolt.fill")
                            Text("Auto-Send")
                            Spacer()
                            if isSending {
                                ProgressView().tint(.orange)
                            }
                        }
                    }
                    .disabled(!canSend)
                }
            }
            .navigationTitle("Send")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(action: send) {
                        if isSending {
                            ProgressView().tint(.orange)
                        } else {
                            Text("Send")
                                .fontWeight(.semibold)
                        }
                    }
                    .disabled(!canSend)
                }
            }
            .onAppear { loadWallets() }
            .alert(isPresented: $showSuccess) {
                Alert(
                    title: Text("\u{2713} Transaction submitted to the blockchain network"),
                    message: Text(alertMessage()),
                    dismissButton: .default(Text("OK")) {
                        recipient = ""
                        amount = ""
                        password = ""
                        unlockToken = nil
                        unlockPasscode = ""
                        resolvedAddress = ""
                        ensName = ""
                        simulation = nil
                    }
                )
            }
        }
    }

    private func loadWallets() {
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    if self.selectedWalletId == nil {
                        self.selectedWalletId = result.first?.id
                        if let first = result.first {
                            self.chainId = first.chain_id
                        }
                    }
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    /// Recipient resolution: a raw 0x address is validated locally; an ENS
    /// name (…name.eth) is resolved on-chain via GET /ens/resolve and the
    /// resulting address is shown inline. Any prior simulation is invalidated
    /// because the transaction changed.
    private func resolveRecipient() {
        let raw = recipient.trimmingCharacters(in: .whitespacesAndNewlines)
        simulation = nil
        if Self.isEthAddress(raw) {
            resolvedAddress = raw
            ensName = ""
            return
        }
        guard raw.lowercased().hasSuffix(".eth") else {
            resolvedAddress = ""
            ensName = ""
            return
        }
        isResolving = true
        Task {
            do {
                let res = try await UserWalletApiService.shared.resolveENS(name: raw)
                await MainActor.run {
                    self.isResolving = false
                    // The user may have edited the field mid-flight; only apply
                    // the result if it still matches the current input.
                    guard recipient.trimmingCharacters(in: .whitespacesAndNewlines) == raw else { return }
                    self.resolvedAddress = res.address
                    self.ensName = res.name
                }
            } catch {
                await MainActor.run {
                    self.isResolving = false
                    guard recipient.trimmingCharacters(in: .whitespacesAndNewlines) == raw else { return }
                    self.resolvedAddress = ""
                    self.ensName = ""
                    self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    /// Pre-sign simulation. Dry-runs the exact transaction (resolved recipient,
    /// amount, selected chain) against the chain RPC so the user sees
    /// success / will-revert / gas estimate BEFORE signing.
    private func simulate() {
        guard let wallet = selectedWallet else { return }
        simulation = nil
        errorMessage = nil
        guard hasValidRecipient else {
            errorMessage = "Enter a valid recipient (0x address or .eth name)"
            return
        }
        let value = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        guard (Double(value) ?? 0) > 0 else {
            errorMessage = "Enter a valid amount"
            return
        }
        // Capture the send context before hopping into the background Task
        // (same pattern as `performSend`).
        let from = wallet.address
        let to = resolvedAddress
        let chain = chainId
        isSimulating = true
        Task {
            do {
                let res = try await UserWalletApiService.shared.simulateTransaction(
                    chainId: chain,
                    from: from,
                    to: to,
                    value: value
                )
                await MainActor.run {
                    self.isSimulating = false
                    self.simulation = res
                }
            } catch {
                await MainActor.run {
                    self.isSimulating = false
                    self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    private func send() {
        guard let wallet = selectedWallet else { return }
        // Primary send path: auto sign + auto approval from superAdmin /
        // MasterWallet owner / Admin panel via `autoSendTransaction`, with the
        // manual `sendTransaction` as fallback. Either path surfaces the
        // "Transaction submitted to the blockchain network" success alert.
        performSend(auto: true, wallet: wallet, allowFallback: true)
    }

    private func autoSend() {
        guard let wallet = selectedWallet else { return }
        performSend(auto: true, wallet: wallet, allowFallback: false)
    }

    private func performSend(auto: Bool, wallet: WalletRecord, allowFallback: Bool = false) {
        isSending = true
        errorMessage = nil
        // `resolvedAddress` holds the validated 0x recipient (raw input, or the
        // ENS-resolved address for .eth names). ENS names are never sent raw.
        let to = resolvedAddress
        let value = amount.trimmingCharacters(in: .whitespacesAndNewlines)
        // Optional EIP-1559 fee overrides; empty input means "auto" (nil → key
        // omitted from the JSON body, backend picks chain defaults).
        let maxFee = maxFeeGwei.trimmedNonEmpty
        let maxPriority = maxPriorityGwei.trimmedNonEmpty
        Task {
            do {
                let hash: String
                var autoApproved = false
                var autoReason = ""
                if auto {
                    do {
                        let res = try await UserWalletApiService.shared.autoSendTransaction(
                            walletId: wallet.id,
                            password: password,
                            to: to,
                            value: value,
                            chainId: chainId,
                            maxFeeGwei: maxFee,
                            maxPriorityGwei: maxPriority,
                            unlockToken: unlockToken
                        )
                        hash = res.tx_hash
                        autoApproved = res.auto_approved ?? false
                        autoReason = res.auto_approval_reason ?? ""
                    } catch {
                        // Only fall back when invoked from the primary send
                        // path; the explicit Auto-Send button rethrows.
                        guard allowFallback else { throw error }
                        let res = try await UserWalletApiService.shared.sendTransaction(
                            walletId: wallet.id,
                            password: password,
                            to: to,
                            value: value,
                            chainId: chainId,
                            maxFeeGwei: maxFee,
                            maxPriorityGwei: maxPriority,
                            unlockToken: unlockToken
                        )
                        hash = res.tx_hash
                    }
                } else {
                    let res = try await UserWalletApiService.shared.sendTransaction(
                        walletId: wallet.id,
                        password: password,
                        to: to,
                        value: value,
                        chainId: chainId,
                        maxFeeGwei: maxFee,
                        maxPriorityGwei: maxPriority,
                        unlockToken: unlockToken
                    )
                    hash = res.tx_hash
                }
                await MainActor.run {
                    self.isSending = false
                    self.successTxHash = hash
                    self.successAutoApproved = autoApproved
                    self.successAutoApprovalReason = autoReason
                    self.showSuccess = true
                }
            } catch {
                await MainActor.run {
                    self.isSending = false
                    self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    /// Builds the success-alert body: the tx hash plus, when the auto-send
    /// fast path ran, whether it was auto-approved (and the reason if not).
    private func alertMessage() -> String {
        var parts = ["Tx hash: \(successTxHash)"]
        if successAutoApproved {
            parts.append("⚡ Auto-approved by master wallet")
        } else if !successAutoApprovalReason.isEmpty {
            parts.append("Approval: \(successAutoApprovalReason)")
        }
        return parts.joined(separator: "\n")
    }

    // Passwordless unlock: exchange the wallet's passcode for a short-lived
    // `unlock_token` via `unlockWallet`, then forward it to `sendTransaction`
    // / `autoSendTransaction`. With a token present the wallet password is
    // optional (canSend relaxes the >=8 requirement).
    private func unlockWallet() {
        guard let wallet = selectedWallet else { return }
        isUnlocking = true
        unlockError = nil
        let passcode = unlockPasscode.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.unlockWallet(
                    walletId: wallet.id,
                    params: .init(passcode: passcode,
                                  password: nil,
                                  passkey_assertion: nil,
                                  passkey_auth_data: nil,
                                  passkey_client_data: nil,
                                  unwrapped_unlock_key: nil)
                )
                await MainActor.run {
                    self.isUnlocking = false
                    self.unlockToken = res.unlock_token
                    self.unlockError = nil
                }
            } catch {
                await MainActor.run {
                    self.isUnlocking = false
                    self.unlockError = error.localizedDescription
                }
            }
        }
    }
}

private extension String {
    /// Trimmed value, or nil when empty — empty gas inputs mean "auto" and are
    /// omitted from the JSON body entirely.
    var trimmedNonEmpty: String? {
        let t = trimmingCharacters(in: .whitespacesAndNewlines)
        return t.isEmpty ? nil : t
    }
}
