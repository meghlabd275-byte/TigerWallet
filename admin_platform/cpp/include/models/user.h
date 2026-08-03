#pragma once

#include <string>
#include <vector>
#include <optional>
#include <chrono>
#include <nlohmann/json.hpp>

namespace tiger::models {

using json = nlohmann::json;

// User model
struct User {
    std::string id;
    std::string user_id;
    std::string username;
    std::string email;
    std::optional<std::string> phone;
    std::string password_hash;
    std::optional<std::string> master_wallet_address;
    admin::UserStatus status;
    int tier;
    bool is_email_verified;
    bool is_phone_verified;
    admin::KYCStatus kyc_status;
    int kyc_level;
    std::optional<std::string> white_label_id;
    std::optional<std::string> referrer_id;
    std::string referral_code;
    std::optional<std::chrono::system_clock::time_point> last_login;
    int failed_login_count;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static User from_json(const json& j);
    
    bool is_active() const;
    bool is_verified() const;
};

// Wallet address model
struct WalletAddress {
    std::string id;
    std::string user_id;
    std::string chain;
    std::string address;
    std::string type;
    std::optional<std::string> label;
    bool is_primary;
    std::chrono::system_clock::time_point created_at;
    
    json to_json() const;
    static WalletAddress from_json(const json& j);
};

// User session model
struct UserSession {
    std::string id;
    std::string user_id;
    std::string token;
    std::string ip_address;
    std::string user_agent;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point expires_at;
    bool is_active;
    
    json to_json() const;
    static UserSession from_json(const json& j);
};

} // namespace tiger::models
