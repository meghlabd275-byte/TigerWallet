/**
 * TigerWallet Desktop - Theme Manager
 * Light/Dark/System theme support for the desktop wallet UI.
 *
 * The desktop wallet does not embed a native GUI toolkit; UI components render
 * HTML strings. ThemeManager centralizes color definitions and exposes the
 * active palette so that HTML-rendering widgets can style themselves
 * consistently, and so theme preference can be persisted to a JSON file.
 */

#pragma once
#include <string>
#include <unordered_map>

namespace TigerWallet {

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

    // Load/save theme preference
    void loadFromFile(const std::string& filepath);
    void saveToFile(const std::string& filepath) const;

private:
    ThemeManager();
    ThemeMode mode_;
    ThemeColors darkColors_;
    ThemeColors lightColors_;

    void detectSystemTheme();
};

} // namespace TigerWallet
