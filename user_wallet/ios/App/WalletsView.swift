import SwiftUI

struct WalletsView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var isLoading = true
    @State private var addMode: AddWalletView.Mode = .create
    @State private var showingAddWallet = false
    @State private var showingPasskeyCreate = false
    @State private var lockWallet: WalletRecord?

    var body: some View {
        NavigationView {
            Group {
                if isLoading {
                    ProgressView("Loading wallets...")
                } else if wallets.isEmpty {
                    VStack(spacing: 12) {
                        Text("No wallets yet")
                            .foregroundColor(.secondary)
                        Text("Tap + to create or import a wallet.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                } else {
                    List(wallets) { wallet in
                        WalletRow(wallet: wallet) {
                            lockWallet = wallet
                        }
                            .swipeActions(edge: .trailing) {
                                Button {
                                    lockWallet = wallet
                                } label: {
                                    Label("Lock", systemImage: "lock.fill")
                                }
                                .tint(.orange)
                            }
                    }
                }
            }
            .navigationTitle("Wallets")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Menu {
                        Button {
                            addMode = .create
                            showingAddWallet = true
                        } label: {
                            Label("Create Wallet", systemImage: "plus.circle")
                        }
                        Button {
                            addMode = .import
                            showingAddWallet = true
                        } label: {
                            Label("Import Wallet", systemImage: "square.and.arrow.down")
                        }
                        Button {
                            showingPasskeyCreate = true
                        } label: {
                            Label("Create with Passkey", systemImage: "key.fill")
                        }
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(isPresented: $showingAddWallet) {
                AddWalletView(mode: addMode) { newWallet, _ in
                    if let w = newWallet { wallets.append(w) }
                    loadWallets()
                }
            }
            .sheet(isPresented: $showingPasskeyCreate) {
                PasskeyCreateWalletView { newWallet in
                    if let w = newWallet { wallets.append(w) }
                    loadWallets()
                }
            }
            .sheet(item: $lockWallet) { wallet in
                LockSetupView(wallet: wallet)
            }
            .onAppear { loadWallets() }
        }
    }

    private func loadWallets() {
        isLoading = true
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    self.isLoading = false
                }
            } catch {
                await MainActor.run { self.isLoading = false }
            }
        }
    }
}

struct WalletRow: View {
    let wallet: WalletRecord
    var onSetupLock: (() -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(wallet.label)
                .font(.headline)
            Text("Chain #\(wallet.chain_id)")
                .font(.subheadline)
                .foregroundColor(.secondary)
            Text(wallet.address)
                .font(.caption)
                .foregroundColor(.secondary)

            if let onSetupLock = onSetupLock {
                Button(action: onSetupLock) {
                    Label("Setup App Lock", systemImage: "lock.fill")
                        .font(.caption)
                }
                .buttonStyle(.bordered)
                .tint(.orange)
            }
        }
    }
}

// Unified create + import view. In .create mode it generates a fresh mnemonic
// (returned by the backend) and shows it with a Copy button. In .import mode it
// accepts an existing mnemonic + password + chain and imports it.
struct AddWalletView: View {
    enum Mode {
        case create
        case `import`
    }

    let mode: Mode
    let onCreated: (WalletRecord?, String?) -> Void

    @State private var label = ""
    @State private var password = ""
    @State private var chainId = 1
    @State private var mnemonic = ""
    @State private var error: String?
    @State private var isCreating = false

    // Revealed mnemonic after a successful create. Shown inline with a Copy
    // button so the user can back up their seed phrase.
    @State private var revealedMnemonic: String?
    @State private var copied = false

    @Environment(\.dismiss) var dismiss

    private var isImport: Bool { mode == .import }

    private var canSubmit: Bool {
        if isCreating { return false }
        if label.isEmpty || password.count < 8 { return false }
        if isImport && mnemonic.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return false
        }
        return true
    }

    var body: some View {
        NavigationView {
            Form {
                Section("Wallet") {
                    TextField("Wallet Name", text: $label)
                    Picker("Chain", selection: $chainId) {
                        Text("Ethereum").tag(1)
                        Text("BNB Chain").tag(56)
                        Text("Polygon").tag(137)
                    }
                }

                if isImport {
                    Section("Seed Phrase") {
                        TextEditor(text: $mnemonic)
                            .frame(minHeight: 80)
                            .autocapitalization(.none)
                            .disableAutocorrection(true)
                        Text("Enter your 12/24-word BIP-39 mnemonic.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                Section("Security") {
                    SecureField("Password (min 8 chars)", text: $password)
                }

                if let revealedMnemonic = revealedMnemonic {
                    Section {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Your recovery phrase")
                                .font(.headline)
                            Text(revealedMnemonic)
                                .font(.system(.body, design: .monospaced))
                                .padding(8)
                                .background(Color(.systemGray6))
                                .cornerRadius(8)
                            HStack {
                                Button {
                                    UIPasteboard.general.string = revealedMnemonic
                                    copied = true
                                } label: {
                                    Label("Copy", systemImage: "doc.on.doc")
                                }
                                if copied {
                                    Text("Copied!")
                                        .font(.caption)
                                        .foregroundColor(.green)
                                }
                                Spacer()
                            }
                            Text("Store this safely. It will not be shown again.")
                                .font(.caption)
                                .foregroundColor(.red)
                        }
                    }
                }

                if let error = error {
                    Section { Text(error).foregroundColor(.red) }
                }
            }
            .navigationTitle(isImport ? "Import Wallet" : "Create Wallet")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                if revealedMnemonic == nil {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { dismiss() }
                    }
                }
                if revealedMnemonic != nil {
                    // After create: replace the Create button with Done so the
                    // user can close the sheet once they've backed up the seed.
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Done") { dismiss() }
                            .fontWeight(.semibold)
                    }
                } else {
                    ToolbarItem(placement: .confirmationAction) {
                        Button(action: submit) {
                            if isCreating {
                                ProgressView().tint(.orange)
                            } else {
                                Text(isImport ? "Import" : "Create")
                                    .fontWeight(.semibold)
                            }
                        }
                        .disabled(!canSubmit)
                    }
                }
            }
        }
    }

    private func submit() {
        isCreating = true
        error = nil
        Task {
            do {
                if isImport {
                    _ = try await UserWalletApiService.shared.importWallet(
                        label: label,
                        password: password,
                        mnemonic: mnemonic.trimmingCharacters(in: .whitespacesAndNewlines),
                        chainId: chainId,
                        passphrase: nil
                    )
                    await MainActor.run {
                        self.isCreating = false
                        self.onCreated(nil, nil)
                        self.dismiss()
                    }
                } else {
                    let w = try await UserWalletApiService.shared.createWallet(
                        label: label,
                        password: password,
                        chainId: chainId
                    )
                    await MainActor.run {
                        self.isCreating = false
                        self.revealedMnemonic = w.mnemonic
                        self.onCreated(w, w.mnemonic)
                    }
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isCreating = false
                }
            }
        }
    }
}


// Setup App Lock — per-wallet sheet offering a real passcode lock and a genuine
// platform passkey lock. Mirrors the Android `showLockSetupDialog` flow.
//
// Passcode path: `setupLock(walletId, LockSetupParams(passcode: passcode))`.
// Passkey path: drives a real `ASAuthorizationPlatformPublicKeyCredentialRegistrationRequest`
// via `PasskeyRegistrar`, extracts credentialId + SPKI publicKey, and posts them to
// `setupLock`. The passkey path is never faked: if the platform refuses or the
// credential public key cannot be parsed, an error is surfaced and no lock is set.
struct LockSetupView: View {
    let wallet: WalletRecord

    @State private var passcode = ""
    @State private var confirm = ""
    @State private var usePasskey = false
    @State private var isSaving = false
    @State private var error: String?
    @State private var success: String?

    @Environment(\.dismiss) var dismiss

    private var passcodeValid: Bool {
        passcode.count >= 4 && passcode == confirm
    }

    var body: some View {
        NavigationView {
            Form {
                Section {
                    Text("Setup App Lock — \(wallet.label)")
                        .font(.headline)
                    Text(wallet.address)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                Section("Passcode") {
                    SecureField("Passcode (min 4 digits)", text: $passcode)
                        .keyboardType(.numberPad)
                    SecureField("Confirm passcode", text: $confirm)
                        .keyboardType(.numberPad)
                }

                Section {
                    Toggle("Use Passkey", isOn: $usePasskey)
                    if usePasskey {
                        Text("A platform passkey (Face/Touch ID) will be registered for this wallet.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                if let error = error {
                    Section { Text(error).foregroundColor(.red).font(.subheadline) }
                }
                if let success = success {
                    Section { Text(success).foregroundColor(.green).font(.subheadline) }
                }
            }
            .navigationTitle("App Lock")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(action: save) {
                        if isSaving {
                            ProgressView().tint(.orange)
                        } else {
                            Text("Save").fontWeight(.semibold)
                        }
                    }
                    .disabled(isSaving || (!usePasskey && !passcodeValid))
                }
            }
        }
    }

    private func save() {
        isSaving = true
        error = nil
        success = nil
        if usePasskey {
            setupWithPasskey()
        } else {
            setupWithPasscode()
        }
    }

    private func setupWithPasscode() {
        Task {
            do {
                let res = try await UserWalletApiService.shared.setupLock(
                    walletId: wallet.id,
                    params: .init(passcode: passcode,
                                  passkey_credential_id: nil,
                                  passkey_public_key: nil)
                )
                await MainActor.run {
                    self.isSaving = false
                    self.success = "App lock set (passcode)" + (res.has_passkey ? " + passkey" : "")
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isSaving = false
                }
            }
        }
    }

    private func setupWithPasskey() {
        // Real platform passkey registration. Wrapped in do/catch so the app
        // never crashes if passkeys are unavailable or the user cancels.
        Task {
            do {
                let registrar = PasskeyRegistrar()
                let userHandle = Data(wallet.id.utf8)
                let cred = try await registrar.register(
                    relyingPartyID: "TigerWallet",
                    userHandle: userHandle,
                    name: "wallet_\(wallet.id)",
                    displayName: wallet.label
                )
                let res = try await UserWalletApiService.shared.setupLock(
                    walletId: wallet.id,
                    params: .init(passcode: passcode.isEmpty ? nil : passcode,
                                  passkey_credential_id: cred.credentialID,
                                  passkey_public_key: cred.publicKey)
                )
                await MainActor.run {
                    self.isSaving = false
                    self.success = "App lock set (passkey)" + (res.has_passcode ? " + passcode" : "")
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isSaving = false
                }
            }
        }
    }
}

// Create with Passkey — drives a real platform passkey registration via
// `PasskeyRegistrar`, then posts the credential to `passkeyCreateWallet` and
// reveals the returned mnemonic with a Copy button (UIPasteboard). Mirrors the
// Android `createPasskeyWallet` flow. No mock fallback.
struct PasskeyCreateWalletView: View {
    let onCreated: (WalletRecord?) -> Void

    @State private var label = "Passkey Wallet"
    @State private var chainId = 1
    @State private var isCreating = false
    @State private var error: String?
    @State private var result: UserWalletApiService.PasskeyWalletResult?
    @State private var createdWallet: WalletRecord?
    @State private var revealedMnemonic: String?
    @State private var copied = false

    @Environment(\.dismiss) var dismiss

    private let chains: [(name: String, id: Int)] = [
        ("Ethereum", 1),
        ("BNB Chain", 56),
        ("Polygon", 137),
    ]

    private var canSubmit: Bool {
        !isCreating && !label.trimmingCharacters(in: .whitespaces).isEmpty
    }

    var body: some View {
        NavigationView {
            Form {
                Section("Wallet") {
                    TextField("Wallet Name", text: $label)
                    Picker("Chain", selection: $chainId) {
                        ForEach(chains, id: \.id) { Text($0.name).tag($0.id) }
                    }
                }

                Section {
                    Text("A platform passkey (Face/Touch ID) will secure this wallet's seed. No password is required.")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                if let revealedMnemonic = revealedMnemonic {
                    Section {
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Your recovery phrase")
                                .font(.headline)
                            Text(revealedMnemonic)
                                .font(.system(.body, design: .monospaced))
                                .padding(8)
                                .background(Color(.systemGray6))
                                .cornerRadius(8)
                            HStack {
                                Button {
                                    UIPasteboard.general.string = revealedMnemonic
                                    copied = true
                                } label: {
                                    Label("Copy", systemImage: "doc.on.doc")
                                }
                                if copied {
                                    Text("Copied!")
                                        .font(.caption)
                                        .foregroundColor(.green)
                                }
                                Spacer()
                            }
                            Text("Store this safely. It will not be shown again.")
                                .font(.caption)
                                .foregroundColor(.red)
                        }
                    }
                }

                if let error = error {
                    Section { Text(error).foregroundColor(.red) }
                }
            }
            .navigationTitle("Create with Passkey")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                if revealedMnemonic == nil {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { dismiss() }
                    }
                }
                if revealedMnemonic != nil {
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Done") {
                            onCreated(createdWallet)
                            dismiss()
                        }
                        .fontWeight(.semibold)
                    }
                } else {
                    ToolbarItem(placement: .confirmationAction) {
                        Button(action: create) {
                            if isCreating {
                                ProgressView().tint(.orange)
                            } else {
                                Text("Create").fontWeight(.semibold)
                            }
                        }
                        .disabled(!canSubmit)
                    }
                }
            }
        }
    }

    private func create() {
        isCreating = true
        error = nil
        Task {
            do {
                // Step 1: real platform passkey registration.
                let registrar = PasskeyRegistrar()
                // userHandle: a stable per-wallet handle. Pre-allocate a random
                // UUID now so the credential is bound before the wallet id is
                // known (the backend assigns the wallet id from this request).
                let userHandle = UUID().uuidData
                let cred = try await registrar.register(
                    relyingPartyID: "TigerWallet",
                    userHandle: userHandle,
                    name: label,
                    displayName: label
                )
                // Step 2: post the credential to create the passkey-secured wallet.
                let res = try await UserWalletApiService.shared.passkeyCreateWallet(
                    params: .init(label: label,
                                  chain_id: chainId,
                                  account_index: 0,
                                  entropy_bits: 160,
                                  credential_id: cred.credentialID,
                                  public_key: cred.publicKey,
                                  sign_count: Int(cred.signCount),
                                  attestation: cred.attestation)
                )
                let wallet = WalletRecord(id: res.wallet_id,
                                           label: res.label,
                                           chain_id: res.chain_id,
                                           address: res.address,
                                           derivation_path: res.derivation_path,
                                           mnemonic: res.mnemonic)
                await MainActor.run {
                    self.isCreating = false
                    self.result = res
                    self.createdWallet = wallet
                    self.revealedMnemonic = res.mnemonic
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isCreating = false
                }
            }
        }
    }
}

