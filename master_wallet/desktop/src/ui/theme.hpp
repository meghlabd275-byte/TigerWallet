/**
 * MasterWallet Desktop - Theme Manager
 * Light/Dark/System theme support. The desktop client renders HTML strings
 * that the React frontend consumes; ThemeManager injects CSS custom
 * properties (variables) so every page re-themes consistently.
 */

#ifndef MASTER_WALLET_THEME_HPP
#define MASTER_WALLET_THEME_HPP

#include <string>

namespace tiger {
namespace master {
namespace ui {

enum class ThemeMode { Light, Dark, System };

struct ThemeColors {
    std::string backgroundColor;
    std::string surfaceColor;
    std::string primaryColor;
    std::string secondaryColor;
    std::string textColor;
    std::string textSecondaryColor;
    std::string borderColor;
    std::string successColor;
    std::string errorColor;
    std::string warningColor;
    std::string accentColor;
};

class ThemeManager {
public:
    static ThemeManager& getInstance();

    void setMode(ThemeMode mode);
    ThemeMode getMode() const;
    bool isDark() const;
    void toggleTheme();

    const ThemeColors& getColors() const;

    // CSS custom-property block (":root { --bg: #...; ... }") for the active
    // palette. Inject this into every rendered page so theme switching applies
    // uniformly.
    std::string getCssVariables() const;

    void loadFromFile(const std::string& filepath);
    void saveToFile(const std::string& filepath) const;

private:
    ThemeManager();
    ThemeMode mode_;
    ThemeColors darkColors_;
    ThemeColors lightColors_;

    void detectSystemTheme();
};

} // namespace ui
} // namespace master
} // namespace tiger

#endif // MASTER_WALLET_THEME_HPP
