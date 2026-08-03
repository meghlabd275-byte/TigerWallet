import SwiftUI

@main
struct AdminAppApp: App {
    @StateObject private var authViewModel = AuthViewModel()
    @StateObject private var themeManager = ThemeManager()
    @StateObject private var networkMonitor = NetworkMonitor()
    
    var body: some Scene {
        WindowGroup {
            if authViewModel.isLoggedIn {
                MainTabView()
                    .environmentObject(authViewModel)
                    .environmentObject(themeManager)
                    .environmentObject(networkMonitor)
                    .preferredColorScheme(themeManager.colorScheme)
            } else {
                LoginView()
                    .environmentObject(authViewModel)
                    .environmentObject(themeManager)
                    .preferredColorScheme(themeManager.colorScheme)
            }
        }
    }
}

class NetworkMonitor: ObservableObject {
    @Published var isConnected = true
    
    init() {
        // Network monitoring would be implemented here
    }
}

class ThemeManager: ObservableObject {
    @Published var isDarkMode: Bool = false {
        didSet {
            UserDefaults.standard.set(isDarkMode, forKey: "isDarkMode")
            NotificationCenter.default.post(name: .themeChanged, object: nil)
        }
    }
    
    var colorScheme: ColorScheme? {
        return isDarkMode ? .dark : .light
    }
    
    init() {
        isDarkMode = UserDefaults.standard.bool(forKey: "isDarkMode")
    }
}

extension Notification.Name {
    static let themeChanged = Notification.Name("themeChanged")
}
