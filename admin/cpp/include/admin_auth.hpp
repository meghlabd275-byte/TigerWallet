/**
 * TigerAdmin C++ Core - Auth Header
 */
#pragma once

#include "admin_security.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>

namespace tiger {
namespace admin {

struct Admin {
    uint64_t id = 0;
    std::string username;
    std::string email;
    std::string password_hash;
    AdminRole role = AdminRole::ADMIN;
    std::vector<std::string> permissions;
    std::string ip_whitelist;
    bool is_active = true;
    bool two_factor_enabled = false;
    std::string two_factor_secret;
    std::optional<int64_t> locked_until;
    std::string last_login_ip;
    int64_t last_login_at = 0;
    int64_t created_at = 0;
    int64_t updated_at = 0;
};

struct Session {
    std::string token;
    AdminID admin_id = 0;
    std::string ip_address;
    std::string user_agent;
    int64_t created_at = 0;
    int64_t expires_at = 0;
    bool is_active = true;
};

class AuthService {
public:
    struct LoginResult {
        bool success = false;
        std::string error_message;
        std::string token;
        std::string refresh_token;
        Admin admin;
    };

    static AuthService& instance();

    void initialize();

    std::string generate_token(const Admin& admin);
    std::optional<std::string> generate_refresh_token(const Admin& admin);
    std::optional<Admin> validate_token(const std::string& token);
    std::optional<Admin> validate_refresh_token(const std::string& token);

    LoginResult login(const std::string& email, const std::string& password,
                      const std::string& ip_address, const std::string& user_agent);
    bool logout(const std::string& token);

    std::string generate_2fa_secret(AdminID admin_id);
    bool verify_2fa(AdminID admin_id, const std::string& code);
    bool enable_2fa(AdminID admin_id, const std::string& secret,
                    const std::string& code);
    bool disable_2fa(AdminID admin_id, const std::string& code);

    bool change_password(AdminID admin_id, const std::string& old_password,
                         const std::string& new_password);
    bool reset_password(const std::string& email);

    bool create_session(const Admin& admin, const std::string& ip_address,
                        const std::string& user_agent);
    bool revoke_session(const std::string& token);
    bool revoke_all_sessions(AdminID admin_id);
    std::vector<Session> get_sessions(AdminID admin_id);

    bool is_ip_allowed(AdminID admin_id, const std::string& ip_address);
    bool add_ip_whitelist(AdminID admin_id, const std::string& ip_address,
                          const std::string& description);
    bool remove_ip_whitelist(AdminID admin_id, uint64_t whitelist_id);

    std::string hash_password(const std::string& password);
    bool verify_password(const std::string& password, const std::string& hash);
    std::string encrypt(const std::string& data);
    std::string decrypt(const std::string& data);
};

class AdminService {
public:
    static AdminService& instance();

    void initialize();

    std::optional<Admin> get_admin(AdminID id);
    std::optional<Admin> get_admin_by_email(const std::string& email);
    std::vector<Admin> list_admins(const std::string& role, bool active_only);

    Admin create_admin(const std::string& username, const std::string& email,
                       const std::string& password, AdminRole role,
                       const std::vector<std::string>& permissions);
    bool update_admin(AdminID id, const std::string& username,
                      const std::string& email, AdminRole role);
    bool delete_admin(AdminID id);
    bool activate_admin(AdminID id);
    bool suspend_admin(AdminID id);
    bool set_permissions(AdminID id, const std::vector<std::string>& permissions);
};

} // namespace admin
} // namespace tiger
