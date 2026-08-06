/**
 * TigerAdmin C++ Core - Audit Logger
 */

#ifndef TIGER_ADMIN_AUDIT_HPP
#define TIGER_ADMIN_AUDIT_HPP

#include <string>
#include <vector>
#include <optional>
#include "admin_models.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Audit Service
// ============================================================================

class AuditService {
public:
    static AuditService& instance();
    
    void initialize();
    
    // Log action
    void log(AdminID admin_id,
             const std::string& action,
             const std::string& resource_type,
             const std::string& resource_id,
             const JSON& details,
             const std::string& ip_address,
             const std::string& user_agent,
             bool success,
             const std::string& error_message = "");
    
    // Convenience methods
    void log_login(AdminID admin_id, bool success, 
                   const std::string& ip_address);
    void log_logout(AdminID admin_id, const std::string& ip_address);
    void log_create(AdminID admin_id, const std::string& resource_type,
                    const std::string& resource_id);
    void log_update(AdminID admin_id, const std::string& resource_type,
                    const std::string& resource_id, const JSON& changes);
    void log_delete(AdminID admin_id, const std::string& resource_type,
                    const std::string& resource_id);
    void log_access_denied(AdminID admin_id, const std::string& action,
                          const std::string& resource);
    
    // Query logs
    struct AuditListParams {
        std::optional<AdminID> admin_id;
        std::optional<std::string> action;
        std::optional<std::string> resource_type;
        std::optional<std::string> resource_id;
        std::optional<bool> success;
        std::string start_date;
        std::string end_date;
        int page = 1;
        int page_size = 50;
    };
    
    struct AuditListResult {
        std::vector<AuditLog> logs;
        int64_t total;
        int page;
        int page_size;
    };
    
    AuditListResult list_audit_logs(const AuditListParams& params);
    
    // Get single log
    std::optional<AuditLog> get_audit_log(uint64_t id);
    
    // Export
    std::string export_logs(const AuditListParams& params,
                           const std::string& format);
    
    // Cleanup
    int cleanup_old_logs(int days_to_keep);
    
private:
    AuditService() = default;
};

// ============================================================================
// Backup Service
// ============================================================================

class BackupService {
public:
    static BackupService& instance();
    
    void initialize();
    
    // Create backup
    struct BackupResult {
        bool success;
        std::string backup_id;
        std::string file_path;
        int64_t file_size;
        std::string message;
    };
    
    BackupResult create_backup(const std::string& backup_type,
                              AdminID admin_id);
    
    // List backups
    struct BackupInfo {
        std::string id;
        std::string type;
        std::string file_path;
        int64_t file_size;
        std::string status;
        AdminID created_by;
        std::string created_at;
        std::string completed_at;
    };
    
    std::vector<BackupInfo> list_backups();
    
    // Restore
    struct RestoreResult {
        bool success;
        std::string message;
    };
    
    RestoreResult restore_backup(const std::string& backup_id,
                                 AdminID admin_id);
    
    // Delete
    bool delete_backup(const std::string& backup_id);
    
    // Schedule
    bool create_schedule(const std::string& schedule,
                        const std::string& retention);
    
private:
    BackupService() = default;
};

// ============================================================================
// Data Archival
// ============================================================================

class ArchivalService {
public:
    static ArchivalService& instance();
    
    void initialize();
    
    // Policies
    struct ArchivalPolicy {
        uint64_t id;
        std::string name;
        std::string table_name;
        int retention_days;
        std::string archive_after;
        bool is_active;
    };
    
    std::vector<ArchivalPolicy> list_policies();
    bool create_policy(const std::string& name,
                      const std::string& table_name,
                      int retention_days);
    bool update_policy(uint64_t id, int retention_days);
    bool delete_policy(uint64_t id);
    
    // Run archival
    bool run_archival(uint64_t policy_id, AdminID admin_id);
    
    // Archive records
    struct ArchivalRecord {
        uint64_t id;
        std::string policy_id;
        std::string table_name;
        int64_t records_archived;
        std::string status;
        AdminID run_by;
        std::string started_at;
        std::string completed_at;
    };
    
    std::vector<ArchivalRecord> list_archival_records(uint64_t policy_id);
    
private:
    ArchivalService() = default;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_AUDIT_HPP
