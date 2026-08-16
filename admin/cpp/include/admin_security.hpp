/**
 * TigerAdmin C++ Core - Security Header
 *
 * Declares the shared base type aliases (AdminID, JSON) and the AdminRole
 * enum used across the admin module. These are defined here (and only here)
 * so that every other admin header can reference them by including this
 * header without risking duplicate definitions.
 */
#pragma once

#include "admin_config.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>

namespace tiger {
namespace admin {

// Shared type aliases — defined once here, included by other admin headers.
using AdminID = uint64_t;
using JSON = std::map<std::string, std::string>;

enum class AdminRole {
    SUPER_ADMIN = 0,
    ADMIN = 1,
    MODERATOR = 2,
    SUPPORT = 3,
    READ_ONLY = 4
};

class SecurityService {
public:
    static SecurityService& instance();

    void initialize();

    std::string encrypt(const std::string& plaintext);
    std::string decrypt(const std::string& ciphertext);

    std::string hash_sha256(const std::string& data);
    std::string hash_sha512(const std::string& data);

    std::string hash_password(const std::string& password);
    bool verify_password(const std::string& password, const std::string& hash);

    std::string generate_token(int length);
    std::string generate_api_key();

    bool is_valid_email(const std::string& email);
    bool is_valid_password(const std::string& password);
    bool is_strong_password(const std::string& password);

    std::string sanitize_string(const std::string& input);
    std::string sanitize_sql(const std::string& input);

    bool is_valid_ip(const std::string& ip);
    bool is_private_ip(const std::string& ip);

private:
    std::string aes_encrypt(const std::string& plaintext);
    std::string aes_decrypt(const std::string& ciphertext);

    std::string encryption_key_;
};

struct FeatureFlag {
    std::string name;
    std::string description;
    bool is_enabled = false;
    int rollout_percentage = 0;
};

class FeatureFlagService {
public:
    static FeatureFlagService& instance();

    void initialize();

    std::optional<FeatureFlag> get_flag(const std::string& name);
    std::vector<FeatureFlag> list_flags();

    bool set_flag(const std::string& name, const std::string& description,
                  bool is_enabled, int rollout_percentage);
    bool update_flag(const std::string& name, const std::optional<bool>& is_enabled,
                     const std::optional<int>& rollout_percentage);
    bool delete_flag(const std::string& name);

    bool is_enabled(const std::string& name);
    bool is_enabled_for_user(const std::string& name, AdminID admin_id);
};

} // namespace admin
} // namespace tiger
