//
//  TigerAdminSystemApp.swift
//  TigerAdminSystem
//

import SwiftUI

@main
struct TigerAdminSystemApp: App {
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
        didSet { UserDefaults.standard.set(isDarkMode, forKey: "dark_mode") }
    }
    init() { self.isDarkMode = UserDefaults.standard.bool(forKey: "dark_mode") }
}
