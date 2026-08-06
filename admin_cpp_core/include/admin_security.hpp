/**
 * TigerAdmin C++ Core - Security & Encryption
 */

#ifndef TIGER_ADMIN_SECURITY_HPP
#define TIGER_ADMIN_SECURITY_HPP

#include <string>
#include <vector>
#include <optional>

namespace tiger {
namespace admin {

// ============================================================================
// Security Service
// ============================================================================

class SecurityService {
public:
    static SecurityService& instance();
    
    void initialize();
    
    // Encryption
    std::string encrypt(const std::string& plaintext);
    std::string decrypt(const std::string& ciphertext);
    
    // Hashing
    std::string hash_sha256(const std::string& data);
    std::string hash_sha512(const std::string& data);
    
    // Password
    std::string hash_password(const std::string& password);
    bool verify_password(const std::string& password,
                        const std::string& hash);
    
    // Token generation
    std::string generate_token(int length = 32);
    std::string generate_api_key();
    
    // Validation
    bool is_valid_email(const std::string& email);
    bool is_valid_password(const std::string& password);
    bool is_strong_password(const std::string& password);
    
    // Sanitization
    std::string sanitize_string(const std::string& input);
    std::string sanitize_sql(const std::string& input);
    
    // IP validation
    bool is_valid_ip(const std::string& ip);
    bool is_private_ip(const std::string& ip);
    
private:
    SecurityService() = default;
    std::string encryption_key_;
    
    std::string aes_encrypt(const std::string& plaintext);
    std::string aes_decrypt(const std::string& ciphertext);
};

// ============================================================================
// Feature Flags
// ============================================================================

class FeatureFlagService {
public:
    static FeatureFlagService& instance();
    
    void initialize();
    
    // Get flag
    std::optional<FeatureFlag> get_flag(const std::string& name);
    std::vector<FeatureFlag> list_flags();
    
    // Set flag
    bool set_flag(const std::string& name,
                 const std::string& description,
                 bool is_enabled,
                 int rollout_percentage);
    
    // Update flag
    bool update_flag(const std::string& name,
                    const std::optional<bool>& is_enabled,
                    const std::optional<int>& rollout_percentage);
    
    // Delete flag
    bool delete_flag(const std::string& name);
    
    // Check if enabled
    bool is_enabled(const std::string& name);
    bool is_enabled_for_user(const std::string& name, AdminID admin_id);
    
private:
    FeatureFlagService() = default;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_SECURITY_HPP
