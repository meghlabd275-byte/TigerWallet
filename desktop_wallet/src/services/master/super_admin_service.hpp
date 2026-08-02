/**
 * Super Admin Service - C++ Desktop Implementation
 * Identical across ALL platforms
 */

#ifndef SUPER_ADMIN_SERVICE_HPP
#define SUPER_ADMIN_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <ctime>
#include <random>

namespace tigerwallet {

// Enums
enum class UserRole { SUPER_ADMIN, MASTER_ADMIN, WHITE_LABEL_ADMIN, USER };
enum class AdminStatus { ACTIVE, INACTIVE, PENDING, SUSPENDED };
enum class AuthorizationStatus { AUTHORIZED, PENDING, REVOKED, REJECTED };

// Data Structures
struct SuperAdmin {
    std::string id;
    std::string email;
    std::string passwordHash;
    std::string secretKey;
    bool twoFactorEnabled;
    std::string twoFactorSecret;
    std::string phone;
    time_t createdAt;
    time_t lastLogin;
    bool isActive;
    std::vector<std::string> permissions;
};

struct MasterAdmin {
    std::string id;
    std::string email;
    std::string passwordHash;
    std::string authorizedBy;
    AuthorizationStatus authorizationStatus;
    bool twoFactorEnabled;
    std::string twoFactorSecret;
    std::string phone;
    bool canCreateWhiteLabel;
    bool canManageUsers;
    bool canManageWallets;
    bool canAccessFinance;
    bool canModifyFeatures;
    bool canManageTokens;
    bool canManageNetworks;
    bool canViewAnalytics;
    bool canManageAdmins;
    int maxWhiteLabels;
    int whiteLabelCount;
    AdminStatus status;
    time_t createdAt;
    time_t lastLogin;
    time_t passwordChangedAt;
    int failedAttempts;
    time_t lockedUntil;
};

struct WhiteLabelAdmin {
    std::string id;
    std::string email;
    std::string passwordHash;
    std::string masterAdminId;
    std::string brandName;
    std::string brandLogo;
    std::string brandColor;
    std::string customDomain;
    AuthorizationStatus authorizationStatus;
    bool twoFactorEnabled;
    std::string twoFactorSecret;
    bool canCustomizeUi;
    bool canCustomizeFees;
    bool canManageUsers;
    bool canManageWallets;
    bool canAccessAnalytics;
    bool canManageTokens;
    double feePercentage;
    AdminStatus status;
    time_t createdAt;
    time_t lastLogin;
};

struct FeatureControl {
    std::string featureName;
    bool enabled;
    bool globalEnabled;
    std::string masterAdminId;
    std::string whiteLabelId;
    std::string updatedBy;
    time_t updatedAt;
};

struct AuditLog {
    std::string id;
    std::string adminId;
    UserRole adminRole;
    std::string action;
    std::string details;
    std::string ipAddress;
    std::string userAgent;
    time_t timestamp;
};

// Super Admin Service
class SuperAdminService {
public:
    static SuperAdminService& getInstance();
    
    // Super Admin Login
    SuperAdmin* superAdminLogin(const std::string& email, const std::string& password, const std::string& twoFactorCode = "");
    
    // Master Admin Operations
    MasterAdmin* createMasterAdminRequest(const std::string& email, const std::string& requestedBy);
    bool authorizeMasterAdmin(const std::string& superAdminId, const std::string& masterAdminId, bool authorized, const std::string& notes = "");
    MasterAdmin* masterAdminLogin(const std::string& email, const std::string& password, const std::string& twoFactorCode = "");
    bool changeMasterAdminPassword(const std::string& adminId, const std::string& oldPassword, const std::string& newPassword);
    bool enableMasterAdmin2FA(const std::string& adminId, const std::string& secret);
    
    // White Label Admin
    WhiteLabelAdmin* createWhiteLabelAdmin(const std::string& masterAdminId, const std::string& email, const std::string& brandName);
    
    // Feature Control
    bool setGlobalFeature(const std::string& superAdminId, const std::string& featureName, bool enabled);
    std::vector<FeatureControl> getAllFeatures();
    bool isFeatureEnabled(const std::string& featureName, const std::string& adminId, UserRole role);
    
    // Audit
    std::vector<AuditLog> getAuditLogs(const std::string& adminId = "", int limit = 100);

private:
    SuperAdminService();
    ~SuperAdminService() = default;
    SuperAdminService(const SuperAdminService&) = delete;
    SuperAdminService& operator=(const SuperAdminService&) = delete;
    
    std::map<std::string, SuperAdmin> superAdmins_;
    std::map<std::string, MasterAdmin> masterAdmins_;
    std::map<std::string, WhiteLabelAdmin> whiteLabelAdmins_;
    std::map<std::string, FeatureControl> featureControls_;
    std::vector<AuditLog> auditLogs_;
    std::mt19937 rng_;
    
    void initialize();
    void createDefaultSuperAdmin();
    void initializeFeatureControls();
    void logAudit(const std::string& adminId, UserRole role, const std::string& action, const std::string& details);
    std::string generateId();
    std::string generateSecretKey();
    std::string generateTempPassword();
    std::string hashPassword(const std::string& password);
    bool verifyTwoFactor(const std::string& secret, const std::string& code);
};

} // namespace tigerwallet

#endif // SUPER_ADMIN_SERVICE_HPP
