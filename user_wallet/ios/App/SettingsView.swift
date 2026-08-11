import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var themeManager: ThemeManager
    @State private var showLogoutAlert = false

    var body: some View {
        NavigationView {
            List {
                Section("Appearance") {
                    Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
                }
                Section("Account") {
                    Button(role: .destructive, action: { showLogoutAlert = true }) {
                        Text("Logout")
                    }
                }
            }
            .navigationTitle("Settings")
            .alert("Logout?", isPresented: $showLogoutAlert) {
                Button("Cancel", role: .cancel) {}
                Button("Logout", role: .destructive) { logout() }
            }
        }
    }

    private func logout() {
        UserWalletApiService.shared.logout()
        NotificationCenter.default.post(name: .didLogout, object: nil)
    }
}
