/**
 * TigerWallet Admin - Theme Manager
 * Complete Dark/Light Theme Support
 */

import UIKit

class ThemeManager {
    static let shared = ThemeManager()
    
    private let defaults = UserDefaults.standard
    private let themeKey = "app_theme"
    
    enum Theme: Int {
        case system = 0
        case light = 1
        case dark = 2
        
        var userInterfaceStyle: UIUserInterfaceStyle {
            switch self {
            case .system: return .unspecified
            case .light: return .light
            case .dark: return .dark
            }
        }
    }
    
    var currentTheme: Theme {
        get {
            return Theme(rawValue: defaults.integer(forKey: themeKey)) ?? .system
        }
        set {
            defaults.set(newValue.rawValue, forKey: themeKey)
            applyTheme()
        }
    }
    
    private init() {}
    
    func applyTheme() {
        DispatchQueue.main.async {
            if let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene {
                windowScene.windows.forEach { window in
                    window.overrideUserInterfaceStyle = self.currentTheme.userInterfaceStyle
                }
            }
        }
    }
    
    // Brand Colors
    static let primaryColor = UIColor(red: 1.0, green: 0.42, blue: 0.21, alpha: 1.0) // #FF6B35
    static let secondaryColor = UIColor(red: 0.0, green: 0.83, blue: 0.67, alpha: 1.0) // #00D4AA
    static let accentColor = UIColor(red: 0.42, green: 0.36, blue: 0.91, alpha: 1.0) // #6C5CE7
    static let errorColor = UIColor(red: 0.91, green: 0.30, blue: 0.24, alpha: 1.0) // #E74C3C
    static let warningColor = UIColor(red: 0.95, green: 0.61, blue: 0.07, alpha: 1.0) // #F39C12
    static let successColor = UIColor(red: 0.15, green: 0.68, blue: 0.38, alpha: 1.0) // #27AE60
    static let infoColor = UIColor(red: 0.20, green: 0.60, blue: 0.86, alpha: 1.0) // #3498DB
}
