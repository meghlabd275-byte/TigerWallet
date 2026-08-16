/**
 * TigerAdmin C++ Core - Audit Header
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

struct AuditLog {
    uint64_t id = 0;
    AdminID admin_id = 0;
    std::string action;
    std::string resource_type;
    std::string resource_id;
    JSON details;
    std::string ip_address;
    std::string user_agent;
    bool success = true;
    std::string error_message;
    int64_t created_at = 0;
};

class AuditService {
public:
    struct AuditListParams {
        int page = 1;
        int page_size = 20;
        std::optional<AdminID> admin_id;
        std::optional<std::string> action;
        std::optional<std::string> resource_type;
        std::optional<std::string> start_date;
        std::optional<std::string> end_date;
    };

    struct AuditListResult {
        int page = 1;
        int page_size = 20;
        int64_t total = 0;
        std::vector<AuditLog> logs;
    };

    static AuditService& instance();

    void initialize();

    void log(AdminID admin_id, const std::string& action,
             const std::string& resource_type, const std::string& resource_id,
             const JSON& details, const std::string& ip_address,
             const std::string& user_agent, bool success,
             const std::string& error_message);

    void log_login(AdminID admin_id, bool success, const std::string& ip_address);
    void log_logout(AdminID admin_id, const std::string& ip_address);
    void log_create(AdminID admin_id, const std::string& resource_type,
                    const std::string& resource_id);
    void log_update(AdminID admin_id, const std::string& resource_type,
                    const std::string& resource_id, const JSON& changes);
    void log_delete(AdminID admin_id, const std::string& resource_type,
                    const std::string& resource_id);
    void log_access_denied(AdminID admin_id, const std::string& action,
                           const std::string& resource);

    AuditListResult list_audit_logs(const AuditListParams& params);
    std::optional<AuditLog> get_audit_log(uint64_t id);

    std::string export_logs(const AuditListParams& params, const std::string& format);

    int cleanup_old_logs(int days_to_keep);
};

class BackupService {
public:
    struct BackupInfo {
        std::string id;
        std::string backup_type;
        std::string file_path;
        int64_t size = 0;
        int64_t created_at = 0;
        std::string status;
    };

    struct BackupResult {
        bool success = false;
        std::string backup_id;
        std::string file_path;
        int64_t size = 0;
        std::string message;
    };

    struct RestoreResult {
        bool success = false;
        std::string message;
    };

    static BackupService& instance();

    void initialize();

    BackupResult create_backup(const std::string& backup_type, AdminID admin_id);
    std::vector<BackupInfo> list_backups();
    RestoreResult restore_backup(const std::string& backup_id, AdminID admin_id);
    bool delete_backup(const std::string& backup_id);
    bool create_schedule(const std::string& schedule, const std::string& retention);
};

class ArchivalService {
public:
    struct ArchivalPolicy {
        uint64_t id = 0;
        std::string name;
        std::string table_name;
        int retention_days = 0;
        bool is_active = true;
    };

    struct ArchivalRecord {
        uint64_t id = 0;
        uint64_t policy_id = 0;
        int64_t records_archived = 0;
        int64_t archived_at = 0;
        std::string status;
    };

    static ArchivalService& instance();

    void initialize();

    std::vector<ArchivalPolicy> list_policies();
    bool create_policy(const std::string& name, const std::string& table_name,
                      int retention_days);
    bool update_policy(uint64_t id, int retention_days);
    bool delete_policy(uint64_t id);
    bool run_archival(uint64_t policy_id, AdminID admin_id);
    std::vector<ArchivalRecord> list_archival_records(uint64_t policy_id);
};

} // namespace admin
} // namespace tiger
