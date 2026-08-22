import SwiftUI

// OnboardingView — the no-registration landing page (mirrors the web Onboarding
// page).
//
// The user opens UserWallet and sees exactly two choices: Create Wallet or
// Import Wallet. No login/register/email wall. Behind the scenes the
// OnboardingManager provisions a transparent ephemeral session so the
// JWT-backed backend is satisfied, but the user only ever interacts with the
// wallet (create-with-password + backup, or import-with-seed).

struct OnboardingView: View {
    @EnvironmentObject var onboardingManager: OnboardingManager
    @EnvironmentObject var themeManager: ThemeManager

    enum Mode { case choose, create, importWallet, backup }
    @State private var mode: Mode = .choose

    @State private var label = "My Wallet"
    @State private var password = ""
    @State private var confirmPassword = ""
    @State private var chainId = 1
    @State private var seed = ""
    @State private var error: String?
    @State private var busy = false
    @State private var createdMnemonic = ""
    @State private var createdId = ""

    // Mirrors the web CHAINS list (Ethereum/BNB/Polygon/Arbitrum/Optimism/Base).
    private let chains: [(id: Int, name: String, symbol: String)] = [
        (1, "Ethereum", "ETH"),
        (56, "BNB Chain", "BNB"),
        (137, "Polygon", "MATIC"),
        (42161, "Arbitrum", "ETH"),
        (10, "Optimism", "ETH"),
        (8453, "Base", "ETH"),
    ]

    var body: some View {
        NavigationView {
            ZStack {
                if !onboardingManager.ready {
                    bootScreen
                } else if mode == .backup && !createdMnemonic.isEmpty {
                    BackupView(mnemonic: createdMnemonic,
                               walletId: createdId,
                               walletPassword: password) {
                        onboardingManager.rememberWallet(createdId)
                    }
                    .background(Color(.systemBackground).ignoresSafeArea())
                } else {
                    chooseAndForms
                }
            }
            .navigationBarTitleDisplayMode(.inline)
            .animation(.default, value: mode)
        }
        .navigationViewStyle(.stack)
    }

    private var bootScreen: some View {
        VStack(spacing: 16) {
            ProgressView()
            Text("Initializing secure wallet…")
                .foregroundColor(.secondary)
            if let err = onboardingManager.sessionError {
                Text(err)
                    .font(.caption)
                    .foregroundColor(.red)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal)
                Button("Retry") {
                    Task { await onboardingManager.ensureSession() }
                }
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var chooseAndForms: some View {
        VStack {
            switch mode {
            case .choose:
                chooseScreen
            case .create:
                createForm
            case .importWallet:
                importForm
            case .backup:
                EmptyView()
            }
        }
    }

    // MARK: - Choose

    private var chooseScreen: some View {
        VStack(spacing: 24) {
            Text("🐯 UserWallet")
                .font(.system(size: 40))
            Text("Welcome")
                .font(.largeTitle)
                .fontWeight(.bold)
            Text("Your keys, your crypto. Get started in seconds — no account needed.")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            VStack(spacing: 12) {
                Button(action: { resetForm(); mode = .create }) {
                    Label("Create a new wallet", systemImage: "plus")
                        .fontWeight(.semibold)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(Color.orange)
                        .foregroundColor(.white)
                        .cornerRadius(12)
                }
                Button(action: { resetForm(); mode = .importWallet }) {
                    Label("Import an existing wallet", systemImage: "arrow.uturn.backward")
                        .fontWeight(.semibold)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(Color(.systemGray5))
                        .cornerRadius(12)
                }
            }
            .padding(.horizontal)

            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(.top, 60)
    }

    // MARK: - Create form

    private var createForm: some View {
        Form {
            Section {
                Text("Create Wallet").font(.title2).fontWeight(.bold)
                Text("Your password encrypts your private key. We cannot recover it.")
                    .font(.caption).foregroundColor(.secondary)
            }
            Section {
                TextField("Wallet name", text: $label)
                Picker("Network", selection: $chainId) {
                    ForEach(chains, id: \.id) { c in
                        Text("\(c.name) (\(c.symbol))").tag(c.id)
                    }
                }
                SecureField("Password (min 8 chars)", text: $password)
                SecureField("Confirm password", text: $confirmPassword)
            }
            if let error = error {
                Section { Text(error).foregroundColor(.red).font(.caption) }
            }
        }
        .formToolbar(title: "Create Wallet",
                     busyLabel: "Creating…", idleLabel: "Create wallet",
                     busy: busy,
                     back: { mode = .choose },
                     submit: handleCreate)
    }

    // MARK: - Import form

    private var importForm: some View {
        Form {
            Section {
                Text("Import Wallet").font(.title2).fontWeight(.bold)
                Text("Enter your 12 or 24-word recovery phrase.")
                    .font(.caption).foregroundColor(.secondary)
            }
            Section {
                TextField("Wallet name", text: $label)
                Picker("Network", selection: $chainId) {
                    ForEach(chains, id: \.id) { c in
                        Text("\(c.name) (\(c.symbol))").tag(c.id)
                    }
                }
                TextEditor(text: $seed)
                    .frame(minHeight: 80)
                    .overlay(
                        Group {
                            if seed.isEmpty {
                                Text("word1 word2 … word12")
                                .foregroundColor(Color(.placeholderText))
                                .padding(.horizontal, 4).padding(.vertical, 8)
                            }
                        }
                    )
                    .autocapitalization(.none)
                SecureField("New password (min 8 chars)", text: $password)
                SecureField("Confirm password", text: $confirmPassword)
            }
            if let error = error {
                Section { Text(error).foregroundColor(.red).font(.caption) }
            }
        }
        .formToolbar(title: "Import Wallet",
                     busyLabel: "Importing…", idleLabel: "Import wallet",
                     busy: busy,
                     back: { mode = .choose },
                     submit: handleImport)
    }

    // MARK: - Handlers (mirror Onboarding.tsx handleCreate / handleImport)

    private func handleCreate() {
        error = nil
        if password.count < 8 { error = "Password must be at least 8 characters"; return }
        if password != confirmPassword { error = "Passwords do not match"; return }
        busy = true
        let label = label.isEmpty ? "My Wallet" : label
        let password = password
        let chainId = chainId
        Task {
            do {
                let res = try await onboardingManager.createWallet(
                    label: label, password: password, chainId: chainId)
                await MainActor.run {
                    self.createdMnemonic = res.mnemonic
                    self.createdId = res.id
                    self.mode = .backup
                    self.busy = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.busy = false
                }
            }
        }
    }

    private func handleImport() {
        error = nil
        let trimmed = seed.trimmingCharacters(in: .whitespacesAndNewlines)
        let words = trimmed.split { $0.isWhitespace }.count
        if words != 12 && words != 24 {
            error = "Recovery phrase must be 12 or 24 words"; return
        }
        if password.count < 8 { error = "Password must be at least 8 characters"; return }
        if password != confirmPassword { error = "Passwords do not match"; return }
        busy = true
        let label = label.isEmpty ? "Imported Wallet" : label
        let password = password
        let chainId = chainId
        Task {
            do {
                let res = try await onboardingManager.importWallet(
                    mnemonic: trimmed, label: label, password: password, chainId: chainId)
                // Imported wallets: remember + go to dashboard (the user already
                // has the mnemonic — no backup screen shown).
                await MainActor.run {
                    self.onboardingManager.rememberWallet(res.id)
                    self.busy = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.busy = false
                }
            }
        }
    }

    private func resetForm() {
        label = "My Wallet"
        password = ""
        confirmPassword = ""
        chainId = 1
        seed = ""
        error = nil
        busy = false
    }
}

// Shared toolbar (Back + submit) for the create/import forms.
private extension Form {
    func formToolbar(title: String, busyLabel: String, idleLabel: String,
                     busy: Bool, back: @escaping () -> Void,
                     submit: @escaping () -> Void) -> some View {
        self
            .navigationTitle(Text(""))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Back", action: back)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? busyLabel : idleLabel, action: submit)
                        .disabled(busy)
                }
            }
    }
}
