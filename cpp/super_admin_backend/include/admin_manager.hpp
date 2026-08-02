/**
 * TigerWallet Super Admin - Admin Manager
 * Complete CRUD operations for admin management
 * No stubs - Production ready
 */

#ifndef TIGERWALLET_ADMIN_MANAGER_HPP
#define TIGERWALLET_ADMIN_MANAGER_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include "database_manager.hpp"
#include "authentication_manager.hpp"

namespace tigerwallet {
namespace super_admin {

// Admin manager result
struct AdminResult {
    bool success;
    std::string error;
    Admin admin;
    
    AdminResult() : success(false) {}
    AdminResult(bool s, const std::string& e = "") : success(s), error(e) {}
};

// White label result
struct WhiteLabelResult {
    bool success;
    std::string error;
    WhiteLabel white_label;
    
    WhiteLabelResult() : success(false) {}
    WhiteLabelResult(bool s, const std::string& e = "") : success(s), error(e) {}
};

class AdminManager {
public:
    static AdminManager& getInstance();
    
    // Initialize with dependencies
    void initialize(DatabaseManager* db, AuthenticationManager* auth);
    
    // ==================== ADMIN CRUD ====================
    
    // Create new admin
    AdminResult createAdmin(const std::string& username, const std::string& password,
                          const std::string& email, AdminRole role,
                          const std::vector<std::string>& permissions,
                          SecurityLevel security_level, const std::string& creator_id);
    
    // Get admin by ID
    std::optional<Admin> getAdminById(const std::string& id);
    
    // Get admin by username
    std::optional<Admin> getAdminByUsername(const std::string& username);
    
    // Get all admins
    std::vector<Admin> getAllAdmins();
    
    // Update admin
    bool updateAdmin(const std::string& admin_id, const std::map<std::string, std::string>& updates,
                   const std::string& updater_id);
    
    // Update permissions
    bool updatePermissions(const std::string& admin_id, const std::vector<std::string>& permissions,
                         const std::string& updater_id);
    
    // Suspend admin
    bool suspendAdmin(const std::string& admin_id, const std::string& suspender_id,
                     const std::string& reason);
    
    // Activate admin
    bool activateAdmin(const std::string& admin_id, const std::string& activator_id);
    
    // Block admin
    bool blockAdmin(const std::string& admin_id, const std::string& blocker_id,
                   const std::string& reason);
    
    // Delete admin
    bool deleteAdmin(const std::string& admin_id, const std::string& deleter_id);
    
    // Get admins by role
    std::vector<Admin> getAdminsByRole(AdminRole role);
    
    // Get admins by status
    std::vector<Admin> getAdminsByStatus(AdminStatus status);
    
    // ==================== WHITE LABEL CRUD ====================
    
    // Create white label
    WhiteLabelResult createWhiteLabel(const std::string& name, const std::string& domain,
                                     const std::string& creator_id);
    
    // Get white label by ID
    std::optional<WhiteLabel> getWhiteLabelById(const std::string& id);
    
    // Get white label by domain
    std::optional<WhiteLabel> getWhiteLabelByDomain(const std::string& domain);
    
    // Get all white labels
    std::vector<WhiteLabel> getAllWhiteLabels();
    
    // Get white labels by status
    std::vector<WhiteLabel> getWhiteLabelsByStatus(int status);
    
    // Approve white label
    bool approveWhiteLabel(const std::string& wl_id, const std::string& approver_id);
    
    // Suspend white label
    bool suspendWhiteLabel(const std::string& wl_id, const std::string& suspender_id,
                          const std::string& reason);
    
    // Revoke white label
    bool revokeWhiteLabel(const std::string& wl_id, const std::string& revoker_id,
                         const std::string& reason);
    
    // Update white label fee
    bool updateWhiteLabelFee(const std::string& wl_id, double fee_percent,
                            const std::string& updater_id);
    
    // Update white label
    bool updateWhiteLabel(const std::string& wl_id,
                         const std::map<std::string, std::string>& updates,
                         const std::string& updater_id);
    
    // Delete white label
    bool deleteWhiteLabel(const std::string& wl_id, const std::string& deleter_id);
    
    // Validate API key
    std::optional<WhiteLabel> validateAPIKey(const std::string& api_key);
    
    // Regenerate API key
    std::string regenerateAPIKey(const std::string& wl_id, const std::string& requester_id);
    
    // ==================== AUDIT LOGS ====================
    
    // Get audit logs
    std::vector<AuditLog> getAuditLogs(const std::string& admin_id, int limit = 100);
    
    // Get audit logs by action
    std::vector<AuditLog> getAuditLogsByAction(const std::string& action, int limit = 100);
    
    // Search audit logs
    std::vector<AuditLog> searchAuditLogs(const std::string& query, int64_t start_time,
                                         int64_t end_time, int limit = 100);
    
    // Export audit logs
    std::string exportAuditLogs(const std::string& admin_id, const std::string& format);
    
    // ==================== PROFIT SHARING ====================
    
    // Set profit share percentage
    bool setProfitSharePercentage(const std::string& white_label_id, double percentage,
                                 const std::string& super_admin_id);
    
    // Get profit share config
    std::optional<ProfitShareConfig> getProfitShareConfig(const std::string& white_label_id);
    
    // Calculate profit share
    void calculateProfitShare(const std::string& white_label_id, double gross_revenue,
                            double& super_admin_share, double& white_label_share);
    
    // Execute profit transfer
    bool executeProfitTransfer(const std::string& white_label_id, const std::string& token,
                              double amount, const std::string& executor_id);
    
    // Get profit history
    std::vector<ProfitTransaction> getProfitHistory(const std::string& white_label_id, int limit = 50);
    
    // Get total profits
    double getTotalProfits();
    
    // ==================== FEATURE FLAGS ====================
    
    // Get all features
    std::vector<FeatureFlag> getAllFeatures();
    
    // Get feature by name
    std::optional<FeatureFlag> getFeatureByName(const std::string& name);
    
    // Set global feature
    bool setGlobalFeature(const std::string& feature_name, bool enabled,
                         const std::string& super_admin_id);
    
    // Set feature for master admin
    bool setMasterAdminFeature(const std::string& feature_name, const std::string& master_admin_id,
                              bool enabled, const std::string& super_admin_id);
    
    // Check if feature is enabled
    bool isFeatureEnabled(const std::string& feature_name, const std::string& admin_id,
                          AdminRole role);

private:
    AdminManager();
    ~AdminManager();
    
    AdminManager(const AdminManager&) = delete;
    AdminManager& operator=(const AdminManager&) = delete;
    
    DatabaseManager* db_ = nullptr;
    AuthenticationManager* auth_ = nullptr;
    
    std::mutex manager_mutex_;
    
    // Helper functions
    std::string generateID();
    std::string generateAPIKey();
    int64_t getCurrentTimestamp();
    bool hasPermission(const std::string& admin_id, const std::string& permission);
    bool isSuperAdmin(const std::string& admin_id);
    
    // Row mapping
    Admin mapRowToAdmin(const RowData& row);
    WhiteLabel mapRowToWhiteLabel(const RowData& row);
    ProfitShareConfig mapRowToProfitConfig(const RowData& row);
    ProfitTransaction mapRowToProfitTransaction(const RowData& row);
    FeatureFlag mapRowToFeatureFlag(const RowData& row);
    AuditLog mapRowToAuditLog(const RowData& row);
};

} // namespace super_admin
} // namespace tigerwallet

#endif // TIGERWALLET_ADMIN_MANAGER_HPP
