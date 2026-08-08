#ifndef MASTER_WALLET_SUPER_ADMIN_SERVICE_HPP
#define MASTER_WALLET_SUPER_ADMIN_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <optional>
#include <chrono>
#include <mutex>
#include <atomic>
#include <atomic>

namespace tiger {
namespace master {
namespace admin {

// Forward declarations
class SuperAdminService;
class AuthorizationManager;
class FeatureController;

/**
 * AdminRole - Master Admin role levels
 */
enum class AdminRole {
    SUPER_ADMIN,
    MASTER_ADMIN,
    WHITE_LABEL_ADMIN,
    SUB_ADMIN,
    VIEWER
};

/**
 * AdminUser - Admin user representation
 */
struct AdminUser {
    std::string id;
    std::string email;
    std::string name;
    AdminRole role;
    std::string masterWalletId;
    std::vector<std::string> permissions;
    bool twoFactorEnabled;
    int64_t createdAt;
    int64_t lastLoginAt;
    bool isActive;
    std::string passwordHash;
    std::string salt;
};

/**
 * FeatureFlag - Feature flag configuration
 */
struct FeatureFlag {
    std::string name;
    bool enabled;
    std::string description;
    std::map<std::string, std::string> metadata;
    int64_t updatedAt;
    std::string updatedBy;
    
    FeatureFlag() : enabled(false), updatedAt(0) {}
};

/**
 * AuditEntry - Audit log entry
 */
struct AuditEntry {
    std::string id;
    std::string adminId;
    std::string action;
    std::string resourceType;
    std::string resourceId;
    std::map<std::string, std::string> details;
    std::string ipAddress;
    std::string userAgent;
    int64_t timestamp;
    bool success;
};

/**
 * AuthorizationRequest - Admin authorization request
 */
struct AuthorizationRequest {
    std::string requestId;
    std::string adminEmail;
    AdminRole requestedRole;
    std::string masterWalletId;
    std::string status;  // pending, approved, rejected
    std::string requestedBy;
    int64_t requestedAt;
    std::string approvedBy;
    int64_t approvedAt;
    std::string reason;
};

/**
 * ProfitSharingConfig - Profit sharing configuration
 */
struct ProfitSharingConfig {
    std::string configId;
    double superAdminPercentage;  // 0-100
    double masterAdminPercentage; // 0-100
    double whiteLabelPercentage;  // 0-100
    std::string superAdminWallet;
    std::map<std::string, double> tokenPercentages;
    bool autoDistribution;
    uint32_t distributionIntervalHours;
    bool isActive;
};

/**
 * SuperAdminService - Super Admin management for MasterWallet
 */
class SuperAdminService {
public:
    SuperAdminService();
    ~SuperAdminService();
    
    // Service lifecycle
    bool initialize(const std::string& configPath);
    void shutdown();
    
    // Authentication
    bool authenticate(
        const std::string& email,
        const std::string& password,
        std::string& adminId
    );
    
    bool changePassword(
        const std::string& adminId,
        const std::string& oldPassword,
        const std::string& newPassword
    );
    
    bool resetPassword(
        const std::string& email,
        const std::string& resetToken,
        const std::string& newPassword
    );
    
    // Two-factor authentication
    bool enable2FA(const std::string& adminId, std::string& secret);
    bool disable2FA(const std::string& adminId, const std::string& code);
    bool verify2FA(const std::string& adminId, const std::string& code);
    
    // Admin management
    std::string createAdmin(const AdminUser& admin);
    bool updateAdmin(const std::string& adminId, const AdminUser& updates);
    bool deleteAdmin(const std::string& adminId);
    bool activateAdmin(const std::string& adminId);
    bool deactivateAdmin(const std::string& adminId);
    
    std::optional<AdminUser> getAdmin(const std::string& adminId);
    std::vector<AdminUser> listAdmins(AdminRole roleFilter = AdminRole::VIEWER);
    
    // Authorization requests
    std::string createAuthorizationRequest(const AuthorizationRequest& request);
    bool approveAuthorizationRequest(
        const std::string& requestId,
        const std::string& approvedBy
    );
    bool rejectAuthorizationRequest(
        const std::string& requestId,
        const std::string& rejectedBy,
        const std::string& reason
    );
    std::vector<AuthorizationRequest> getPendingRequests();
    
    // Feature flags
    bool setFeatureFlag(const std::string& name, bool enabled);
    bool setFeatureFlag(const FeatureFlag& flag);
    std::optional<FeatureFlag> getFeatureFlag(const std::string& name);
    std::vector<FeatureFlag> listFeatureFlags();
    bool isFeatureEnabled(const std::string& name);
    
    // Audit logging
    void logAudit(const AuditEntry& entry);
    std::vector<AuditEntry> getAuditLogs(
        const std::string& adminId = "",
        const std::string& action = "",
        int64_t startTime = 0,
        int64_t endTime = 0,
        int limit = 100
    );
    
    // Profit sharing
    bool configureProfitSharing(const ProfitSharingConfig& config);
    std::optional<ProfitSharingConfig> getProfitSharingConfig();
    bool executeDistribution(const std::string& masterWalletId);
    
    // Master Admin management
    bool authorizeMasterAdmin(
        const std::string& masterWalletId,
        const std::string& authorizedBy
    );
    bool revokeMasterAdmin(const std::string& masterWalletId);
    std::vector<std::string> getAuthorizedMasterAdmins();
    
    // Statistics
    struct AdminStats {
        uint64_t totalAdmins;
        uint64_t activeAdmins;
        uint64_t pendingRequests;
        uint64_t totalAuditEntries;
        uint64_t failedLogins;
        double averageSessionDuration;
    };
    
    AdminStats getStats() const;

private:
    std::map<std::string, AdminUser> admins_;
    std::map<std::string, AuthorizationRequest> authRequests_;
    std::map<std::string, FeatureFlag> featureFlags_;
    std::vector<AuditEntry> auditLogs_;
    std::optional<ProfitSharingConfig> profitSharingConfig_;
    
    mutable std::mutex dataMutex_;
    std::atomic<uint64_t> failedLogins_{0};
    std::chrono::system_clock::time_point serviceStartTime_;
    
    std::string encryptionKey_;
    
    // Private methods
    std::string hashPassword(const std::string& password, const std::string& salt);
    std::string generateSalt();
    std::string generateRandomPassword(size_t length = 24);
    bool verifyPassword(
        const std::string& password,
        const std::string& hash,
        const std::string& salt
    );
    
    std::string generateAdminId();
    std::string generateRequestId();
    
    bool hasPermission(
        const AdminUser& admin,
        const std::string& permission
    ) const;
    
    bool isValidEmail(const std::string& email) const;
    bool isValidPassword(const std::string& password) const;
    
    void loadConfig(const std::string& path);
    void saveConfig(const std::string& path);
    
    void cleanupOldLogs(int64_t retentionDays);
};

} // namespace admin
} // namespace master
} // namespace tiger

#endif // MASTER_WALLET_SUPER_ADMIN_SERVICE_HPP
