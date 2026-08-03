#pragma once

#include <string>
#include <vector>
#include <optional>
#include <chrono>
#include <nlohmann/json.hpp>

namespace tiger::models {

using json = nlohmann::json;

// Admin model
struct Admin {
    std::string id;
    std::string username;
    std::string email;
    std::string password_hash;
    admin::AdminRole role;
    std::vector<std::string> permissions;
    admin::AdminStatus status;
    bool two_factor_enabled;
    std::optional<std::string> two_factor_secret;
    std::vector<std::string> backup_codes;
    int security_level;
    int failed_attempts;
    std::optional<std::chrono::system_clock::time_point> locked_until;
    std::optional<std::chrono::system_clock::time_point> last_login;
    std::optional<std::string> last_ip;
    std::vector<std::string> ip_whitelist;
    std::string created_by;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static Admin from_json(const json& j);
    
    bool has_permission(const std::string& permission) const;
    bool is_active() const;
    bool is_locked() const;
};

// Admin session model
struct AdminSession {
    std::string id;
    std::string admin_id;
    std::string token;
    std::string ip_address;
    std::string user_agent;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point last_activity;
    std::chrono::seconds expires_in;
    bool is_active;
    
    json to_json() const;
    static AdminSession from_json(const json& j);
};

// Audit log model
struct AuditLog {
    std::string id;
    std::string admin_id;
    std::string action;
    std::string resource;
    std::optional<std::string> resource_id;
    json details;
    std::optional<std::string> ip_address;
    bool success;
    std::optional<std::string> error;
    std::chrono::system_clock::time_point created_at;
    
    json to_json() const;
    static AuditLog from_json(const json& j);
};

// IP whitelist model
struct IPWhitelist {
    std::string id;
    std::string admin_id;
    std::string ip_cidr;
    std::string description;
    bool is_active;
    std::chrono::system_clock::time_point created_at;
    
    json to_json() const;
    static IPWhitelist from_json(const json& j);
};

// API key model
struct APIKey {
    std::string id;
    std::string key_hash;
    std::string name;
    std::string admin_id;
    std::vector<std::string> permissions;
    int rate_limit_minute;
    int rate_limit_day;
    std::string tier;
    bool is_active;
    std::optional<std::chrono::system_clock::time_point> expires_at;
    std::optional<std::chrono::system_clock::time_point> last_used;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static APIKey from_json(const json& j);
};

} // namespace tiger::models
