import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @EnvironmentObject var onboardingManager: OnboardingManager
    @State private var showLogoutAlert = false

    var body: some View {
        NavigationView {
            List {
                Section("Appearance") {
                    Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
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
            .alert("Reset local wallet?", isPresented: $showLogoutAlert) {
                Button("Cancel", role: .cancel) {}
                Button("Reset", role: .destructive) { onboardingManager.reset() }
            } message: {
                Text("This clears the on-device wallet ids and the transparent session. Your funds remain on-chain — re-import a wallet with its recovery phrase to regain access. Your keys, your crypto.")
            }
        }
    }
}
