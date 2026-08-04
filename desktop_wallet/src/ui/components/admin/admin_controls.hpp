/**
 * TigerWallet Desktop Admin Controls UI - C++ Implementation
 * Production-ready administrative controls for MasterWallet
 */

#ifndef ADMIN_CONTROLS_HPP
#define ADMIN_CONTROLS_HPP

#include <string>
#include <vector>
#include <memory>
#include <functional>
#include <unordered_map>
#include <optional>
#include <chrono>

#include <nlohmann/json.hpp>

using json = nlohmann::json;

namespace tigerwallet {
namespace ui {
namespace admin {

struct AdminUser {
    std::string id;
    std::string username;
    std::string email;
    std::string role;
    std::vector<std::string> permissions;
    bool is_active;
    bool two_factor_enabled;
    std::chrono::system_clock::time_point last_login;
    std::chrono::system_clock::time_point created_at;
};

struct AdminRole {
    std::string id;
    std::string name;
    std::vector<std::string> permissions;
    std::string description;
    uint32_t user_count;
};

struct SystemConfig {
    std::string key;
    std::string value;
    std::string type;
    std::string category;
    bool is_sensitive;
    std::string description;
    std::chrono::system_clock::time_point updated_at;
    std::string updated_by;
};

struct AuditLogEntry {
    std::string id;
    std::string user_id;
    std::string username;
    std::string action;
    std::string resource;
    std::string resource_id;
    std::string ip_address;
    json old_value;
    json new_value;
    bool success;
    std::chrono::system_clock::time_point timestamp;
};

struct SystemMetrics {
    uint64_t total_users;
    uint64_t active_users_24h;
    uint64_t total_transactions;
    uint64_t transaction_volume_24h;
    uint64_t total_wallets;
    double system_load;
    double memory_usage;
    double cpu_usage;
    uint64_t api_requests_24h;
    std::chrono::system_clock::time_point updated_at;
};

struct FeeStructure {
    std::string id;
    std::string name;
    std::string type;
    std::string asset;
    std::string fee_type;
    std::string fee_value;
    bool is_active;
};

struct GlobalSettings {
    std::string platform_name;
    std::string support_email;
    std::string default_language;
    std::string default_currency;
    bool maintenance_mode;
    std::string maintenance_message;
    bool registration_enabled;
    bool trading_enabled;
};

class AdminService {
public:
    static AdminService& getInstance();
    
    bool initialize();
    void shutdown();
    
    std::string createUser(const AdminUser& user, const std::string& password);
    std::optional<AdminUser> getUser(const std::string& user_id);
    std::vector<AdminUser> getAllUsers();
    bool updateUser(const std::string& user_id, const AdminUser& user);
    bool deleteUser(const std::string& user_id);
    bool activateUser(const std::string& user_id);
    bool deactivateUser(const std::string& user_id);
    
    std::string createRole(const AdminRole& role);
    std::vector<AdminRole> getAllRoles();
    
    std::string setConfig(const SystemConfig& config);
    std::vector<SystemConfig> getAllConfigs();
    
    std::vector<AuditLogEntry> getAuditLogs(const std::string& user_id, int limit = 100);
    SystemMetrics getSystemMetrics();
    
    std::string createFeeStructure(const FeeStructure& fee);
    std::vector<FeeStructure> getAllFeeStructures();
    
    GlobalSettings getGlobalSettings();
    bool updateGlobalSettings(const GlobalSettings& settings);
    
    bool enableMaintenanceMode(const std::string& message);
    bool disableMaintenanceMode();

private:
    AdminService() = default;
    ~AdminService() = default;
    
    std::mutex mutex_;
    bool initialized_;
    std::unordered_map<std::string, AdminUser> users_;
    std::unordered_map<std::string, AdminRole> roles_;
    std::unordered_map<std::string, SystemConfig> configs_;
    std::vector<AuditLogEntry> audit_logs_;
    GlobalSettings global_settings_;
};

class AdminUIWidget {
public:
    AdminUIWidget();
    ~AdminUIWidget() = default;
    
    std::string renderDashboard(const SystemMetrics& metrics);
    std::string renderUserList(const std::vector<AdminUser>& users);
    std::string renderRoleList(const std::vector<AdminRole>& roles);
    std::string renderConfigList(const std::vector<SystemConfig>& configs);
    std::string renderAuditLogList(const std::vector<AuditLogEntry>& logs);
    std::string renderFeeList(const std::vector<FeeStructure>& fees);
    std::string renderGlobalSettings(const GlobalSettings& settings);
    std::string renderButton(const std::string& id, const std::string& text, const std::string& style);
    std::string renderTable(const std::vector<std::string>& headers, const std::vector<std::vector<std::string>>& rows);
    std::string renderModal(const std::string& id, const std::string& title, const std::string& content);
    std::string renderStatusBadge(bool is_active);

private:
    std::mutex mutex_;
};

} // namespace admin
} // namespace ui
} // namespace tigerwallet

#endif // ADMIN_CONTROLS_HPP
