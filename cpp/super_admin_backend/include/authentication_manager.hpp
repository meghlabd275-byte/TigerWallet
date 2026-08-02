/**
 * TigerWallet Super Admin - Authentication Manager
 * Real implementation: bcrypt password hashing, TOTP 2FA
 * No stubs - Production ready
 */

#ifndef TIGERWALLET_AUTH_MANAGER_HPP
#define TIGERWALLET_AUTH_MANAGER_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <chrono>
#include <ctime>
#include "database_manager.hpp"

namespace tigerwallet {
namespace super_admin {

// Auth result
struct AuthResult {
    bool success;
    std::string error;
    std::string session_token;
    std::string admin_id;
    std::string username;
    AdminRole role;
    
    AuthResult() : success(false) {}
    AuthResult(bool s, const std::string& e = "") : success(s), error(e) {}
};

// 2FA types
enum class TwoFactorType {
    NONE = 0,
    TOTP = 1,      // Time-based OTP (Google Authenticator, etc.)
    SMS = 2,       // SMS code
    EMAIL = 3,     // Email code
    HARDWARE = 4   // Hardware key (YubiKey, etc.)
};

// Password requirements
struct PasswordPolicy {
    size_t min_length = 8;
    size_t max_length = 128;
    bool require_uppercase = true;
    bool require_lowercase = true;
    bool require_numbers = true;
    bool require_special = true;
    int max_age_days = 90;
    int history_count = 5;
};

// Login attempt tracking
struct LoginAttempt {
    std::string identifier; // username or IP
    int count;
    int64_t first_attempt;
    int64_t last_attempt;
    bool locked;
    int64_t locked_until;
};

// IP whitelist entry
struct IPWhitelistEntry {
    std::string id;
    std::string admin_id;
    std::string ip_address;
    std::string description;
    int64_t created_at;
    bool is_active;
};

class AuthenticationManager {
public:
    static AuthenticationManager& getInstance();
    
    // Initialize with database reference
    void initialize(DatabaseManager* db);
    
    // Password operations
    std::string hashPassword(const std::string& password);
    bool verifyPassword(const std::string& password, const std::string& hash);
    bool validatePasswordPolicy(const std::string& password, std::string& error);
    bool changePassword(const std::string& admin_id, const std::string& old_password, 
                       const std::string& new_password, std::string& error);
    bool resetPassword(const std::string& admin_id, const std::string& new_password, std::string& error);
    bool enforcePasswordHistory(const std::string& admin_id, const std::string& new_password);
    
    // Two-factor authentication
    std::string generateTOTPSecret(const std::string& admin_id);
    bool enableTOTP(const std::string& admin_id, const std::string& secret);
    bool disableTOTP(const std::string& admin_id);
    bool verifyTOTP(const std::string& admin_id, const std::string& code);
    std::string getTOTPUri(const std::string& admin_id, const std::string& secret, const std::string& issuer = "TigerWallet");
    
    // Login/logout
    AuthResult login(const std::string& username, const std::string& password, 
                    const std::string& two_factor_code, const std::string& ip_address,
                    const std::string& user_agent);
    bool logout(const std::string& token);
    bool logoutAllSessions(const std::string& admin_id);
    bool validateSession(const std::string& token);
    Session getSession(const std::string& token);
    std::vector<Session> getAdminSessions(const std::string& admin_id);
    
    // Session management
    bool extendSession(const std::string& token, int64_t new_expiry);
    bool revokeSession(const std::string& token);
    bool revokeAllOtherSessions(const std::string& admin_id, const std::string& current_token);
    
    // IP whitelist management
    bool addToIPWhitelist(const std::string& admin_id, const std::string& ip_address, 
                          const std::string& description);
    bool removeFromIPWhitelist(const std::string& entry_id);
    std::vector<IPWhitelistEntry> getIPWhitelist(const std::string& admin_id);
    bool isIPAllowed(const std::string& admin_id, const std::string& ip_address);
    
    // Rate limiting
    bool checkRateLimit(const std::string& identifier, int max_requests, int window_seconds);
    void recordRequest(const std::string& identifier);
    bool isRateLimited(const std::string& ip_address);
    bool isAccountLocked(const std::string& username);
    
    // Brute force protection
    void recordFailedAttempt(const std::string& identifier);
    void recordSuccessfulAttempt(const std::string& identifier);
    void clearFailedAttempts(const std::string& identifier);
    int getFailedAttemptCount(const std::string& identifier);
    
    // Password policy
    void setPasswordPolicy(const PasswordPolicy& policy);
    PasswordPolicy getPasswordPolicy();
    
    // Cleanup expired sessions
    void cleanupExpiredSessions();
    
private:
    AuthenticationManager();
    ~AuthenticationManager();
    
    AuthenticationManager(const AuthenticationManager&) = delete;
    AuthenticationManager& operator=(const AuthenticationManager&) = delete;
    
    DatabaseManager* db_ = nullptr;
    PasswordPolicy password_policy_;
    
    // In-memory rate limiting
    std::map<std::string, LoginAttempt> login_attempts_;
    std::map<std::string, std::pair<int64_t, int>> rate_limit_cache_;
    
    std::mutex auth_mutex_;
    
    // Constants
    static constexpr int MAX_FAILED_ATTEMPTS = 3;
    static constexpr int64_t LOCKOUT_DURATION_SECONDS = 900; // 15 minutes
    static constexpr int64_t SESSION_DURATION_SECONDS = 86400; // 24 hours
    
    // Helper functions
    std::string generateToken();
    std::string generateSessionId();
    int64_t getCurrentTimestamp();
    bool isPasswordInHistory(const std::string& admin_id, const std::string& password_hash);
    void savePasswordToHistory(const std::string& admin_id, const std::string& password_hash);
    
    // TOTP implementation
    std::string computeTOTP(const std::string& secret, int64_t timestamp);
};

} // namespace super_admin
} // namespace tigerwallet

#endif // TIGERWALLET_AUTH_MANAGER_HPP
