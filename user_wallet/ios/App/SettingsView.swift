import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @EnvironmentObject var onboardingManager: OnboardingManager
    @State private var showLogoutAlert = false
    @State private var backendURL = ""

    var body: some View {
        NavigationView {
            List {
                Section("Appearance") {
                    Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
                }
                Section("Backend Server") {
                    TextField("http://host:8443/api/v1", text: $backendURL)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        .keyboardType(.URL)
                        .font(.system(.footnote, design: .monospaced))
                    Button("Save Server URL") {
                        let url = backendURL.trimmingCharacters(in: .whitespacesAndNewlines)
                        if url.hasPrefix("http://") || url.hasPrefix("https://") {
                            UserWalletApiService.shared.baseURL = url
                        }
                    }
                    .disabled(!(backendURL.hasPrefix("http://") || backendURL.hasPrefix("https://")))
                    Text("API base URL of the UserWallet backend. Change this for a self-hosted deployment.")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                Section("Self-custody session") {
                    HStack {
                        Text("Backed up wallets")
                        Spacer()
                        Text("\(onboardingManager.localWalletIds.count)")
                            .foregroundColor(.secondary)
                    }
                    if let email = onboardingManager.transparentEmail {
                        HStack {
                            Text("Device session")
                            Spacer()
                            Text(email)
                                .font(.system(.footnote, design: .monospaced))
                                .foregroundColor(.secondary)
                                .lineLimit(1)
                                .truncationMode(.middle)
                        }
                    }
                }
                Section {
                    Button(role: .destructive, action: { showLogoutAlert = true }) {
                        Text("Reset local wallet")
                    }
                }
            }
            .navigationTitle("Settings")
            .onAppear { backendURL = UserWalletApiService.shared.baseURL }
            .alert("Reset local wallet?", isPresented: $showLogoutAlert) {
                Button("Cancel", role: .cancel) {}
                Button("Reset", role: .destructive) { onboardingManager.reset() }
            } message: {
                Text("This clears the on-device wallet ids and the transparent session. Your funds remain on-chain — re-import a wallet with its recovery phrase to regain access. Your keys, your crypto.")
            }
        }
    }
}
