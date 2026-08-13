/**
 * MasterWallet Desktop - Theme Manager Implementation
 */

#include "theme.hpp"

#include <algorithm>
#include <cctype>
#include <cstdlib>
#include <fstream>
#include <sstream>

namespace tiger {
namespace master {
namespace ui {

namespace {

std::string trim(const std::string& s) {
    const size_t a = s.find_first_not_of(" \t\r\n");
    if (a == std::string::npos) return "";
    const size_t b = s.find_last_not_of(" \t\r\n");
    return s.substr(a, b - a + 1);
}

std::string toLower(std::string s) {
    std::transform(s.begin(), s.end(), s.begin(),
                   [](unsigned char c) { return static_cast<char>(std::tolower(c)); });
    return s;
}

ThemeColors makeDarkColors() {
    ThemeColors c;
    c.backgroundColor = "#1a1a2e";
    c.surfaceColor = "#16213e";
    c.primaryColor = "#0f3460";
    c.secondaryColor = "#533483";
    c.textColor = "#e4e6eb";
    c.textSecondaryColor = "#a0a3b1";
    c.borderColor = "#2a2a4e";
    c.successColor = "#2ecc71";
    c.errorColor = "#e74c3c";
    c.warningColor = "#f39c12";
    c.accentColor = "#0f3460";
    return c;
}

ThemeColors makeLightColors() {
    ThemeColors c;
    c.backgroundColor = "#ffffff";
    c.surfaceColor = "#f5f5f5";
    c.primaryColor = "#0f3460";
    c.secondaryColor = "#3a6ea5";
    c.textColor = "#1a1a2e";
    c.textSecondaryColor = "#555770";
    c.borderColor = "#d9dce1";
    c.successColor = "#27ae60";
    c.errorColor = "#c0392b";
    c.warningColor = "#d68910";
    c.accentColor = "#0f3460";
    return c;
}

std::string modeToString(ThemeMode mode) {
    switch (mode) {
        case ThemeMode::Light: return "light";
        case ThemeMode::Dark: return "dark";
        case ThemeMode::System: return "system";
    }
    return "system";
}

std::string extractJsonString(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\"";
    size_t k = json.find(needle);
    if (k == std::string::npos) return "";
    size_t colon = json.find(':', k + needle.size());
    if (colon == std::string::npos) return "";
    size_t quote = json.find('"', colon + 1);
    if (quote == std::string::npos) return "";
    std::string out;
    for (size_t i = quote + 1; i < json.size(); ++i) {
        const char ch = json[i];
        if (ch == '\\' && i + 1 < json.size()) {
            out.push_back(json[i + 1]);
            ++i;
        } else if (ch == '"') {
            break;
        } else {
            out.push_back(ch);
        }
    }
    return out;
}

} // namespace

ThemeManager::ThemeManager()
    : mode_(ThemeMode::System),
      darkColors_(makeDarkColors()),
      lightColors_(makeLightColors()) {}

ThemeManager& ThemeManager::getInstance() {
    static ThemeManager instance;
    return instance;
}

void ThemeManager::setMode(ThemeMode mode) { mode_ = mode; }
ThemeMode ThemeManager::getMode() const { return mode_; }

void ThemeManager::detectSystemTheme() {
    if (const char* t = std::getenv("TIGERWALLET_THEME")) {
        const std::string v = toLower(trim(t));
        if (v == "dark") { mode_ = ThemeMode::Dark; return; }
        if (v == "light") { mode_ = ThemeMode::Light; return; }
        if (v == "system") { mode_ = ThemeMode::System; return; }
    }
    if (const char* d = std::getenv("DARKMODE")) {
        const std::string v = toLower(trim(d));
        if (v == "1" || v == "true" || v == "dark") { mode_ = ThemeMode::Dark; return; }
        if (v == "0" || v == "false" || v == "light") { mode_ = ThemeMode::Light; return; }
    }
    if (const char* g = std::getenv("GTK_THEME")) {
        const std::string v = toLower(trim(g));
        if (v.find("dark") != std::string::npos) { mode_ = ThemeMode::Dark; return; }
        if (v.find("light") != std::string::npos) { mode_ = ThemeMode::Light; return; }
    }
    mode_ = ThemeMode::Light;
}

bool ThemeManager::isDark() const {
    switch (mode_) {
        case ThemeMode::Dark: return true;
        case ThemeMode::Light: return false;
        case ThemeMode::System: {
            const_cast<ThemeManager*>(this)->detectSystemTheme();
            const ThemeMode resolved = mode_;
            const_cast<ThemeManager*>(this)->mode_ = ThemeMode::System;
            return resolved == ThemeMode::Dark;
        }
    }
    return false;
}

void ThemeManager::toggleTheme() {
    mode_ = isDark() ? ThemeMode::Light : ThemeMode::Dark;
}

const ThemeColors& ThemeManager::getColors() const {
    return isDark() ? darkColors_ : lightColors_;
}

std::string ThemeManager::getCssVariables() const {
    const ThemeColors& c = getColors();
    std::ostringstream css;
    css << ":root {\n"
        << "  --bg-color: " << c.backgroundColor << ";\n"
        << "  --surface-color: " << c.surfaceColor << ";\n"
        << "  --primary-color: " << c.primaryColor << ";\n"
        << "  --secondary-color: " << c.secondaryColor << ";\n"
        << "  --text-color: " << c.textColor << ";\n"
        << "  --text-secondary-color: " << c.textSecondaryColor << ";\n"
        << "  --border-color: " << c.borderColor << ";\n"
        << "  --success-color: " << c.successColor << ";\n"
        << "  --error-color: " << c.errorColor << ";\n"
        << "  --warning-color: " << c.warningColor << ";\n"
        << "  --accent-color: " << c.accentColor << ";\n"
        << "}\n";
    return css.str();
}

void ThemeManager::loadFromFile(const std::string& filepath) {
    std::ifstream in(filepath);
    if (!in.is_open()) {
        detectSystemTheme();
        return;
    }
    std::stringstream ss;
    ss << in.rdbuf();
    const std::string contents = ss.str();
    const std::string value = toLower(trim(extractJsonString(contents, "theme")));
    if (value == "dark") mode_ = ThemeMode::Dark;
    else if (value == "light") mode_ = ThemeMode::Light;
    else if (value == "system") mode_ = ThemeMode::System;
    else detectSystemTheme();
}

void ThemeManager::saveToFile(const std::string& filepath) const {
    std::ofstream out(filepath);
    if (!out.is_open()) return;
    out << "{\n  \"theme\": \"" << modeToString(mode_) << "\"\n}\n";
}

} // namespace ui
} // namespace master
} // namespace tiger
