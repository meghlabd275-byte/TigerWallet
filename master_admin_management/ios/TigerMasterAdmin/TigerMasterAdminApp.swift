//
//  TigerMasterAdminApp.swift
//  TigerMasterAdmin
//

import SwiftUI

@main
struct TigerMasterAdminApp: App {
    @StateObject private var themeManager = ThemeManager()
    
    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(themeManager)
                .preferredColorScheme(themeManager.isDarkMode ? .dark : .light)
        }
    }
}

class ThemeManager: ObservableObject {
    @Published var isDarkMode: Bool {
        didSet {
            UserDefaults.standard.set(isDarkMode, forKey: "dark_mode")
        }
    }
    
    init() {
        self.isDarkMode = UserDefaults.standard.bool(forKey: "dark_mode")
    }
}
