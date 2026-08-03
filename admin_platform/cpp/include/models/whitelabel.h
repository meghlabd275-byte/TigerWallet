#pragma once

#include <string>
#include <vector>
#include <optional>
#include <chrono>
#include <nlohmann/json.hpp>

namespace tiger::models {

using json = nlohmann::json;

// White label model
struct WhiteLabel {
    std::string id;
    std::string client_id;
    std::string name;
    std::string domain;
    bool domain_verified;
    std::string admin_user_id;
    admin::WhiteLabelStatus status;
    
    // Branding
    std::optional<std::string> logo_url;
    std::optional<std::string> primary_color;
    std::optional<std::string> secondary_color;
    std::optional<std::string> theme_mode;
    
    // Features
    std::vector<std::string> features;
    
    // Limits
    int max_users;
    double max_daily_volume;
    
    // Fees
    double platform_fee_percent;
    double custom_fee_percent;
    
    // Liquidity
    std::optional<std::string> liquidity_source;
    std::optional<std::string> trading_pairs_import;
    
    // Contacts
    std::optional<std::string> contact_email;
    std::optional<std::string> contact_phone;
    
    std::optional<std::chrono::system_clock::time_point> activated_at;
    std::optional<std::chrono::system_clock::time_point> expires_at;
    std::string created_by;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static WhiteLabel from_json(const json& j);
    
    bool is_active() const;
    bool is_expired() const;
};

// White label admin model
struct WhiteLabelAdmin {
    std::string id;
    std::string white_label_id;
    std::string user_id;
    std::string username;
    std::string email;
    std::string password_hash;
    std::string role;
    std::vector<std::string> permissions;
    std::string status;
    std::chrono::system_clock::time_point created_at;
    
    json to_json() const;
    static WhiteLabelAdmin from_json(const json& j);
};

// White label branding model
struct WhiteLabelBranding {
    std::string id;
    std::string white_label_id;
    std::optional<std::string> logo_url;
    std::optional<std::string> favicon_url;
    std::string primary_color;
    std::string secondary_color;
    std::string accent_color;
    std::string background_color;
    std::string text_color;
    std::optional<std::string> font_family;
    std::optional<std::string> terms_url;
    std::optional<std::string> privacy_url;
    std::optional<std::string> support_email;
    std::optional<std::string> support_url;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static WhiteLabelBranding from_json(const json& j);
};

} // namespace tiger::models
