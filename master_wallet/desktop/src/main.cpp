/**
 * MasterWallet Desktop - C++ entry point.
 *
 * Initializes the real libcurl API client against the canonical backend
 * (http://localhost:8450, overridable via MASTER_WALLET_API_URL), loads the
 * ThemeManager (which injects light/dark CSS variables), and probes the
 * backend health endpoint. No fake data, no simulation: every operation
 * either reaches the real backend or reports an honest failure.
 */

#include "api_client.hpp"
#include "theme.hpp"

#include <cstdlib>
#include <iostream>

using tiger::master::api::APIClient;
using tiger::master::api::backend;
using tiger::master::ui::ThemeManager;

int main() {
    std::string baseUrl = "http://localhost:8450";
    if (const char* env = std::getenv("MASTER_WALLET_API_URL")) {
        if (env && *env) baseUrl = env;
    }

    try {
        APIClient::instance()->initialize(baseUrl);
        std::cout << "MasterWallet desktop backend: " << backend()->baseUrl() << "\n";
    } catch (const std::exception& e) {
        std::cerr << "Failed to initialize API client: " << e.what() << "\n";
        return 1;
    }

    // Load + persist the theme; emit the CSS variable block the React UI injects.
    auto& theme = ThemeManager::getInstance();
    theme.loadFromFile("theme.json");
    std::cout << "Theme: " << (theme.isDark() ? "dark" : "light") << "\n";
    std::cout << "---- CSS variables ----\n" << theme.getCssVariables() << "-----------------------\n";
    theme.saveToFile("theme.json");

    // Real backend health probe. Honest report only.
    try {
        std::string health = backend()->get("/health");
        std::cout << "Backend /health: " << health << "\n";
    } catch (const tiger::master::api::APIException& e) {
        std::cerr << "Backend /health failed: " << e.what()
                  << " (status " << e.statusCode() << ")\n";
        return 2;
    }

    // Public chain/gas/price endpoints (no auth) confirm connectivity.
    try {
        std::cout << "Chains: " << backend()->get("/api/v1/chains") << "\n";
    } catch (const tiger::master::api::APIException& e) {
        std::cerr << "Chains fetch failed: " << e.what() << "\n";
    }

    backend()->shutdown();
    return 0;
}
