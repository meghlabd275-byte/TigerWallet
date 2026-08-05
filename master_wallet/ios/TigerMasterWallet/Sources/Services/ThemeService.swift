// MasterWallet Theme Service (iOS)
// Light/Dark theme switching
// Production-ready

import UIKit

class ThemeService {
    
    static let shared = ThemeService()
    
    private let themeKey = "theme_mode"
    private var currentTheme: String = "light"
    private var listeners: [ThemeChangeListener] = []
    
    // MARK: - Colors
    
    struct ThemeColors {
        let background: UIColor
        let surface: UIColor
        let surfaceElevated: UIColor
        let primary: UIColor
        let primaryVariant: UIColor
        let secondary: UIColor
        let accent: UIColor
        let text: UIColor
        let textSecondary: UIColor
        let textMuted: UIColor
        let heading: UIColor
        let link: UIColor
        let border: UIColor
        let success: UIColor
        let warning: UIColor
        let error: UIColor
        let onPrimary: UIColor
        let isDark: Bool
    }
    
    // MARK: - Initialize
    
    private init() {}
    
    func initialize() {
        currentTheme = UserDefaults.standard.string(forKey: themeKey) ?? getSystemTheme()
        applyTheme()
    }
    
    // MARK: - Theme Control
    
    func getTheme() -> String {
        return currentTheme
    }
    
    func isDarkMode() -> Bool {
        return currentTheme == "dark"
    }
    
    func setTheme(_ theme: String) {
        guard theme == "light" || theme == "dark" else { return }
        
        currentTheme = theme
        UserDefaults.standard.set(theme, forKey: themeKey)
        
        applyTheme()
        notifyListeners()
    }
    
    func toggleTheme() {
        setTheme(currentTheme == "light" ? "dark" : "light")
    }
    
    // MARK: - Apply Theme
    
    private func applyTheme() {
        let style: UIUserInterfaceStyle = currentTheme == "dark" ? .dark : .light
        
        if let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene {
            windowScene.windows.forEach { window in
                window.overrideUserInterfaceStyle = style
            }
        }
        
        // Apply global tint
        UIView.appearance().tintColor = getThemeColors().primary
    }
    
    // MARK: - Theme Colors
    
    func getThemeColors() -> ThemeColors {
        if currentTheme == "dark" {
            return ThemeColors(
                background: UIColor(hex: 0x0A0A0A),
                surface: UIColor(hex: 0x1A1A1A),
                surfaceElevated: UIColor(hex: 0x242424),
                primary: UIColor(hex: 0x3B82F6),
                primaryVariant: UIColor(hex: 0x2563EB),
                secondary: UIColor(hex: 0x6366F1),
                accent: UIColor(hex: 0x8B5CF6),
                text: UIColor(hex: 0xE5E5E5),
                textSecondary: UIColor(hex: 0xA3A3A3),
                textMuted: UIColor(hex: 0x737373),
                heading: UIColor(hex: 0xF5F5F5),
                link: UIColor(hex: 0x60A5FA),
                border: UIColor(hex: 0x333333),
                success: UIColor(hex: 0x22C55E),
                warning: UIColor(hex: 0xF59E0B),
                error: UIColor(hex: 0xEF4444),
                onPrimary: UIColor(hex: 0xFFFFFFFF),
                isDark: true
            )
        } else {
            return ThemeColors(
                background: UIColor(hex: 0xFFFFFF),
                surface: UIColor(hex: 0xF9FAFB),
                surfaceElevated: UIColor(hex: 0xFFFFFF),
                primary: UIColor(hex: 0x3B82F6),
                primaryVariant: UIColor(hex: 0x2563EB),
                secondary: UIColor(hex: 0x6366F1),
                accent: UIColor(hex: 0x8B5CF6),
                text: UIColor(hex: 0x171717),
                textSecondary: UIColor(hex: 0x525252),
                textMuted: UIColor(hex: 0xA3A3A3),
                heading: UIColor(hex: 0x0A0A0A),
                link: UIColor(hex: 0x2563EB),
                border: UIColor(hex: 0xE5E5E5),
                success: UIColor(hex: 0x16A34A),
                warning: UIColor(hex: 0xD97706),
                error: UIColor(hex: 0xDC2626),
                onPrimary: UIColor(hex: 0xFFFFFFFF),
                isDark: false
            )
        }
    }
    
    func getColor(named name: String) -> UIColor {
        let colors = getThemeColors()
        
        switch name {
        case "background": return colors.background
        case "surface": return colors.surface
        case "surfaceElevated": return colors.surfaceElevated
        case "primary": return colors.primary
        case "primaryVariant": return colors.primaryVariant
        case "secondary": return colors.secondary
        case "accent": return colors.accent
        case "text": return colors.text
        case "textSecondary": return colors.textSecondary
        case "textMuted": return colors.textMuted
        case "heading": return colors.heading
        case "link": return colors.link
        case "border": return colors.border
        case "success": return colors.success
        case "warning": return colors.warning
        case "error": return colors.error
        case "onPrimary": return colors.onPrimary
        default: return colors.text
        }
    }
    
    // MARK: - Listeners
    
    func addListener(_ listener: ThemeChangeListener) {
        if !listeners.contains(where: { $0 === listener }) {
            listeners.append(listener)
        }
    }
    
    func removeListener(_ listener: ThemeChangeListener) {
        listeners.removeAll { $0 === listener }
    }
    
    private func notifyListeners() {
        for listener in listeners {
            listener.onThemeChanged(currentTheme)
        }
    }
    
    // MARK: - Private Methods
    
    private func getSystemTheme() -> String {
        let style = UITraitCollection.current.userInterfaceStyle
        return style == .dark ? "dark" : "light"
    }
}

// MARK: - Protocol

protocol ThemeChangeListener: AnyObject {
    func onThemeChanged(_ theme: String)
}

// MARK: - UIColor Extension

extension UIColor {
    convenience init(hex: Int, alpha: CGFloat = 1.0) {
        let red = CGFloat((hex >> 16) & 0xFF) / 255.0
        let green = CGFloat((hex >> 8) & 0xFF) / 255.0
        let blue = CGFloat(hex & 0xFF) / 255.0
        self.init(red: red, green: green, blue: blue, alpha: alpha)
    }
}
