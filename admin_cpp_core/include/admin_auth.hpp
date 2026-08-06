/**
 * TigerAdmin C++ Core - Authentication Handler
 * JWT, 2FA, Session Management
 */

#ifndef TIGER_ADMIN_AUTH_HPP
#define TIGER_ADMIN_AUTH_HPP

#include <string>
#include <memory>
#include <optional>
#include "admin_models.hpp"
#include "admin_connection_pool.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Auth Service
// ============================================================================

class AuthService {
public:
    static AuthService& instance();
    
    void initialize();
    
    // JWT Operations
    std::string generate_token(const Admin& admin);
    std::optional<std::string> generate_refresh_token(const Admin& admin);
    std::optional<Admin> validate_token(const std::string& token);
    std::optional<Admin> validate_refresh_token(const std::string& token);
    
    // Login/Logout
    struct LoginResult {
        bool success;
        std::string token;
        std::string refresh_token;
        Admin admin;
        std::string error_message;
    };
    
    LoginResult login(const std::string& email, 
                      const std::string& password,
                      const std::string& ip_address,
                      const std::string& user_agent);
    
    bool logout(const std::string& token);
    
    // 2FA
    std::string generate_2fa_secret(const AdminID admin_id);
    bool verify_2fa(const AdminID admin_id, const std::string& code);
    bool enable_2fa(const AdminID admin_id, const std::string& secret,
                     const std::string& code);
    bool disable_2fa(const AdminID admin_id, const std::string& code);
    
    // Password
    bool change_password(AdminID admin_id, 
                         const std::string& old_password,
                         const std::string& new_password);
    bool reset_password(const std::string& email);
    
    // Session Management
    bool create_session(const Admin& admin, 
                        const std::string& ip_address,
                        const std::string& user_agent);
    bool revoke_session(const std::string& token);
    bool revoke_all_sessions(AdminID admin_id);
    std::vector<Session> get_sessions(AdminID admin_id);
    
    // IP Whitelist
    bool is_ip_allowed(AdminID admin_id, const std::string& ip_address);
    bool add_ip_whitelist(AdminID admin_id, const std::string& ip_address,
                          const std::string& description);
    bool remove_ip_whitelist(AdminID admin_id, uint64_t whitelist_id);
    
private:
    AuthService() = default;
    std::string jwt_secret_;
    std::string encryption_key_;
    
    std::string hash_password(const std::string& password);
    bool verify_password(const std::string& password, 
                        const std::string& hash);
    
    std::string encrypt(const std::string& data);
    std::string decrypt(const std::string& data);
};

// ============================================================================
// Admin Management
// ============================================================================

class AdminService {
public:
    static AdminService& instance();
    
    void initialize();
    
    // CRUD
    std::optional<Admin> get_admin(AdminID id);
    std::optional<Admin> get_admin_by_email(const std::string& email);
    std::vector<Admin> list_admins(const std::string& role = "",
                                     bool active_only = true);
    
    Admin create_admin(const std::string& username,
                       const std::string& email,
                       const std::string& password,
                       AdminRole role,
                       const std::vector<std::string>& permissions);
    
    bool update_admin(AdminID id, const std::string& username,
                      const std::string& email, AdminRole role);
    bool delete_admin(AdminID id);
    
    // Status
    bool activate_admin(AdminID id);
    bool suspend_admin(AdminID id);
    bool set_permissions(AdminID id, 
                         const std::vector<std::string>& permissions);
    
private:
    AdminService() = default;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_AUTH_HPP
