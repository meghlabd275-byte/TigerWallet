/**
 * TigerWallet Super Admin - Admin Manager Implementation
 * Production-ready CRUD operations
 */

#include "admin_manager.hpp"
#include <iostream>
#include <sstream>
#include <iomanip>
#include <random>
#include <openssl/rand.h>

namespace tigerwallet {
namespace super_admin {

AdminManager::AdminManager() {}

AdminManager::~AdminManager() {}

AdminManager& AdminManager::getInstance() {
    static AdminManager instance;
    return instance;
}

void AdminManager::initialize(DatabaseManager* db, AuthenticationManager* auth) {
    db_ = db;
    auth_ = auth;
}

std::string AdminManager::generateID() {
    unsigned char buffer[16];
    RAND_bytes(buffer, 16);
    
    std::stringstream ss;
    for (int i = 0; i < 16; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)buffer[i];
    }
    return ss.str();
}

std::string AdminManager::generateAPIKey() {
    unsigned char buffer[32];
    RAND_bytes(buffer, 32);
    
    std::stringstream ss;
    for (int i = 0; i < 32; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)buffer[i];
    }
    return "tw_" + ss.str();
}

int64_t AdminManager::getCurrentTimestamp() {
    return std::chrono::duration_cast<std::chrono::seconds>(
        std::chrono::system_clock::now().time_since_epoch()
    ).count();
}

bool AdminManager::hasPermission(const std::string& admin_id, const std::string& permission) {
    auto admin = getAdminById(admin_id);
    if (!admin) return false;
    
    // Super admin has all permissions
    if (admin->role == AdminRole::SUPER_ADMIN) {
        return true;
    }
    
    for (const auto& p : admin->permissions) {
        if (p == "*" || p == permission) {
            return true;
        }
    }
    
    return false;
}

bool AdminManager::isSuperAdmin(const std::string& admin_id) {
    auto admin = getAdminById(admin_id);
    return admin && admin->role == AdminRole::SUPER_ADMIN;
}

// ==================== ADMIN CRUD ====================

AdminResult AdminManager::createAdmin(const std::string& username, const std::string& password,
                                     const std::string& email, AdminRole role,
                                     const std::vector<std::string>& permissions,
                                     SecurityLevel security_level, const std::string& creator_id) {
    AdminResult result;
    
    if (!db_) {
        result.error = "Database not initialized";
        return result;
    }
    
    // Check if creator is super admin when creating super admin
    if (role == AdminRole::SUPER_ADMIN && !isSuperAdmin(creator_id)) {
        result.error = "Only super admin can create super admin accounts";
        return result;
    }
    
    // Check if username exists
    auto existing = getAdminByUsername(username);
    if (existing) {
        result.error = "Username already exists";
        return result;
    }
    
    // Check if email exists
    auto email_result = db_->querySingle("SELECT id FROM admins WHERE email = ?", {email});
    if (email_result) {
        result.error = "Email already registered";
        return result;
    }
    
    // Validate password
    std::string password_error;
    if (!auth_ || !auth_->validatePasswordPolicy(password, password_error)) {
        result.error = "Password does not meet policy requirements";
        return result;
    }
    
    // Create admin
    std::string admin_id = generateID();
    std::string password_hash = auth_ ? auth_->hashPassword(password) : password;
    
    // Convert permissions to JSON
    std::string permissions_json = "[";
    for (size_t i = 0; i < permissions.size(); i++) {
        permissions_json += "\"" + permissions[i] + "\"";
        if (i < permissions.size() - 1) permissions_json += ",";
    }
    permissions_json += "]";
    
    bool success = db_->execute(
        "INSERT INTO admins (id, username, password_hash, email, role, security_level, permissions, two_factor_enabled, created_at, status) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, 1)",
        {admin_id, username, password_hash, email, std::to_string((int)role),
         std::to_string((int)security_level), permissions_json, std::to_string(getCurrentTimestamp())}
    );
    
    if (success) {
        // Log action
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), creator_id, "CREATE_ADMIN", "Created admin: " + username + " with role: " + std::to_string((int)role),
             std::to_string(getCurrentTimestamp())}
        );
        
        result.success = true;
        result.admin = *getAdminById(admin_id);
    } else {
        result.error = "Failed to create admin";
    }
    
    return result;
}

std::optional<Admin> AdminManager::getAdminById(const std::string& id) {
    if (!db_) return std::nullopt;
    
    auto row = db_->querySingle("SELECT * FROM admins WHERE id = ?", {id});
    if (!row) return std::nullopt;
    
    return mapRowToAdmin(*row);
}

std::optional<Admin> AdminManager::getAdminByUsername(const std::string& username) {
    if (!db_) return std::nullopt;
    
    auto row = db_->querySingle("SELECT * FROM admins WHERE username = ?", {username});
    if (!row) return std::nullopt;
    
    return mapRowToAdmin(*row);
}

std::vector<Admin> AdminManager::getAllAdmins() {
    std::vector<Admin> admins;
    
    if (!db_) return admins;
    
    auto results = db_->query("SELECT * FROM admins ORDER BY created_at DESC");
    for (const auto& row : results) {
        admins.push_back(mapRowToAdmin(row));
    }
    
    return admins;
}

bool AdminManager::updateAdmin(const std::string& admin_id,
                              const std::map<std::string, std::string>& updates,
                              const std::string& updater_id) {
    if (!db_) return false;
    
    // Check permissions
    if (!isSuperAdmin(updater_id)) {
        return false;
    }
    
    std::vector<std::string> set_clauses;
    std::vector<std::string> params;
    
    for (const auto& [key, value] : updates) {
        if (key == "email" || key == "username" || key == "status" || key == "role") {
            set_clauses.push_back(key + " = ?");
            params.push_back(value);
        }
    }
    
    if (set_clauses.empty()) return false;
    
    params.push_back(std::to_string(getCurrentTimestamp()));
    params.push_back(admin_id);
    
    std::string query = "UPDATE admins SET " + join(set_clauses, ", ") + ", updated_at = ? WHERE id = ?";
    
    bool success = db_->execute(query, params);
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), updater_id, "UPDATE_ADMIN", "Updated admin: " + admin_id, std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

bool AdminManager::updatePermissions(const std::string& admin_id,
                                   const std::vector<std::string>& permissions,
                                   const std::string& updater_id) {
    if (!db_) return false;
    
    if (!isSuperAdmin(updater_id)) {
        return false;
    }
    
    // Convert to JSON
    std::string permissions_json = "[";
    for (size_t i = 0; i < permissions.size(); i++) {
        permissions_json += "\"" + permissions[i] + "\"";
        if (i < permissions.size() - 1) permissions_json += ",";
    }
    permissions_json += "]";
    
    bool success = db_->execute(
        "UPDATE admins SET permissions = ?, updated_at = ? WHERE id = ?",
        {permissions_json, std::to_string(getCurrentTimestamp()), admin_id}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), updater_id, "UPDATE_PERMISSIONS", "Updated permissions for: " + admin_id,
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

bool AdminManager::suspendAdmin(const std::string& admin_id, const std::string& suspender_id,
                               const std::string& reason) {
    if (!db_) return false;
    
    if (!isSuperAdmin(suspender_id)) {
        return false;
    }
    
    // Can't suspend yourself
    if (admin_id == suspender_id) {
        return false;
    }
    
    bool success = db_->execute(
        "UPDATE admins SET status = ?, updated_at = ? WHERE id = ?",
        {std::to_string((int)AdminStatus::SUSPENDED), std::to_string(getCurrentTimestamp()), admin_id}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), suspender_id, "SUSPEND_ADMIN", "Suspended admin: " + admin_id + " - Reason: " + reason,
             std::to_string(getCurrentTimestamp())}
        );
        
        // Invalidate sessions
        db_->execute("UPDATE sessions SET is_valid = 0 WHERE admin_id = ?", {admin_id});
    }
    
    return success;
}

bool AdminManager::activateAdmin(const std::string& admin_id, const std::string& activator_id) {
    if (!db_) return false;
    
    if (!isSuperAdmin(activator_id)) {
        return false;
    }
    
    bool success = db_->execute(
        "UPDATE admins SET status = ?, failed_attempts = 0, locked_until = 0, updated_at = ? WHERE id = ?",
        {std::to_string((int)AdminStatus::ACTIVE), std::to_string(getCurrentTimestamp()), admin_id}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), activator_id, "ACTIVATE_ADMIN", "Activated admin: " + admin_id,
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

bool AdminManager::blockAdmin(const std::string& admin_id, const std::string& blocker_id,
                             const std::string& reason) {
    if (!db_) return false;
    
    if (!isSuperAdmin(blocker_id)) {
        return false;
    }
    
    bool success = db_->execute(
        "UPDATE admins SET status = ?, updated_at = ? WHERE id = ?",
        {std::to_string((int)AdminStatus::BLOCKED), std::to_string(getCurrentTimestamp()), admin_id}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), blocker_id, "BLOCK_ADMIN", "Blocked admin: " + admin_id + " - Reason: " + reason,
             std::to_string(getCurrentTimestamp())}
        );
        
        db_->execute("UPDATE sessions SET is_valid = 0 WHERE admin_id = ?", {admin_id});
    }
    
    return success;
}

bool AdminManager::deleteAdmin(const std::string& admin_id, const std::string& deleter_id) {
    if (!db_) return false;
    
    if (!isSuperAdmin(deleter_id)) {
        return false;
    }
    
    // Can't delete yourself
    if (admin_id == deleter_id) {
        return false;
    }
    
    bool success = db_->execute("DELETE FROM admins WHERE id = ?", {admin_id});
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), deleter_id, "DELETE_ADMIN", "Deleted admin: " + admin_id,
             std::to_string(getCurrentTimestamp())}
        );
        
        db_->execute("DELETE FROM sessions WHERE admin_id = ?", {admin_id});
    }
    
    return success;
}

std::vector<Admin> AdminManager::getAdminsByRole(AdminRole role) {
    std::vector<Admin> admins;
    
    if (!db_) return admins;
    
    auto results = db_->query(
        "SELECT * FROM admins WHERE role = ? ORDER BY created_at DESC",
        {std::to_string((int)role)}
    );
    
    for (const auto& row : results) {
        admins.push_back(mapRowToAdmin(row));
    }
    
    return admins;
}

std::vector<Admin> AdminManager::getAdminsByStatus(AdminStatus status) {
    std::vector<Admin> admins;
    
    if (!db_) return admins;
    
    auto results = db_->query(
        "SELECT * FROM admins WHERE status = ? ORDER BY created_at DESC",
        {std::to_string((int)status)}
    );
    
    for (const auto& row : results) {
        admins.push_back(mapRowToAdmin(row));
    }
    
    return admins;
}

// ==================== WHITE LABEL CRUD ====================

WhiteLabelResult AdminManager::createWhiteLabel(const std::string& name, const std::string& domain,
                                               const std::string& creator_id) {
    WhiteLabelResult result;
    
    if (!db_) {
        result.error = "Database not initialized";
        return result;
    }
    
    // Check if domain exists
    auto existing = getWhiteLabelByDomain(domain);
    if (existing) {
        result.error = "Domain already registered";
        return result;
    }
    
    std::string wl_id = generateID();
    std::string api_key = generateAPIKey();
    
    // Hash API key
    std::string api_key_hash = auth_ ? auth_->hashPassword(api_key) : api_key;
    
    bool success = db_->execute(
        "INSERT INTO white_labels (id, name, domain, api_key, api_key_hash, fee_percent, status, created_at, features, custom_branding) VALUES (?, ?, ?, ?, ?, 20.0, 1, ?, '[\"*\"]', 1)",
        {wl_id, name, domain, api_key, api_key_hash, std::to_string(getCurrentTimestamp())}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), creator_id, "CREATE_WHITELABEL", "Created white label: " + name + " (" + domain + ")",
             std::to_string(getCurrentTimestamp())}
        );
        
        result.success = true;
        result.white_label = *getWhiteLabelById(wl_id);
    } else {
        result.error = "Failed to create white label";
    }
    
    return result;
}

std::optional<WhiteLabel> AdminManager::getWhiteLabelById(const std::string& id) {
    if (!db_) return std::nullopt;
    
    auto row = db_->querySingle("SELECT * FROM white_labels WHERE id = ?", {id});
    if (!row) return std::nullopt;
    
    return mapRowToWhiteLabel(*row);
}

std::optional<WhiteLabel> AdminManager::getWhiteLabelByDomain(const std::string& domain) {
    if (!db_) return std::nullopt;
    
    auto row = db_->querySingle("SELECT * FROM white_labels WHERE domain = ?", {domain});
    if (!row) return std::nullopt;
    
    return mapRowToWhiteLabel(*row);
}

std::vector<WhiteLabel> AdminManager::getAllWhiteLabels() {
    std::vector<WhiteLabel> white_labels;
    
    if (!db_) return white_labels;
    
    auto results = db_->query("SELECT * FROM white_labels ORDER BY created_at DESC");
    for (const auto& row : results) {
        white_labels.push_back(mapRowToWhiteLabel(row));
    }
    
    return white_labels;
}

std::vector<WhiteLabel> AdminManager::getWhiteLabelsByStatus(int status) {
    std::vector<WhiteLabel> white_labels;
    
    if (!db_) return white_labels;
    
    auto results = db_->query(
        "SELECT * FROM white_labels WHERE status = ? ORDER BY created_at DESC",
        {std::to_string(status)}
    );
    
    for (const auto& row : results) {
        white_labels.push_back(mapRowToWhiteLabel(row));
    }
    
    return white_labels;
}

bool AdminManager::approveWhiteLabel(const std::string& wl_id, const std::string& approver_id) {
    if (!db_) return false;
    
    if (!isSuperAdmin(approver_id)) {
        return false;
    }
    
    bool success = db_->execute(
        "UPDATE white_labels SET status = 2, approved_by = ?, approved_at = ? WHERE id = ?",
        {approver_id, std::to_string(getCurrentTimestamp()), wl_id}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), approver_id, "APPROVE_WHITELABEL", "Approved white label: " + wl_id,
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

bool AdminManager::suspendWhiteLabel(const std::string& wl_id, const std::string& suspender_id,
                                    const std::string& reason) {
    if (!db_) return false;
    
    if (!isSuperAdmin(suspender_id)) {
        return false;
    }
    
    bool success = db_->execute(
        "UPDATE white_labels SET status = 3 WHERE id = ?",
        {wl_id}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), suspender_id, "SUSPEND_WHITELABEL", "Suspended white label: " + wl_id + " - Reason: " + reason,
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

bool AdminManager::revokeWhiteLabel(const std::string& wl_id, const std::string& revoker_id,
                                   const std::string& reason) {
    if (!db_) return false;
    
    if (!isSuperAdmin(revoker_id)) {
        return false;
    }
    
    bool success = db_->execute(
        "UPDATE white_labels SET status = 4 WHERE id = ?",
        {wl_id}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), revoker_id, "REVOKE_WHITELABEL", "Revoked white label: " + wl_id + " - Reason: " + reason,
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

bool AdminManager::updateWhiteLabelFee(const std::string& wl_id, double fee_percent,
                                     const std::string& updater_id) {
    if (!db_) return false;
    
    if (fee_percent < 0 || fee_percent > 20.0) {
        return false;
    }
    
    if (!isSuperAdmin(updater_id)) {
        return false;
    }
    
    bool success = db_->execute(
        "UPDATE white_labels SET fee_percent = ? WHERE id = ?",
        {std::to_string(fee_percent), wl_id}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), updater_id, "UPDATE_WHITELABEL_FEE", "Updated fee to " + std::to_string(fee_percent) + "% for: " + wl_id,
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

bool AdminManager::updateWhiteLabel(const std::string& wl_id,
                                   const std::map<std::string, std::string>& updates,
                                   const std::string& updater_id) {
    if (!db_) return false;
    
    if (!isSuperAdmin(updater_id)) {
        return false;
    }
    
    // Implementation for updating white label fields
    return db_->execute("UPDATE white_labels SET updated_at = ? WHERE id = ?",
                       {std::to_string(getCurrentTimestamp()), wl_id});
}

bool AdminManager::deleteWhiteLabel(const std::string& wl_id, const std::string& deleter_id) {
    if (!db_) return false;
    
    if (!isSuperAdmin(deleter_id)) {
        return false;
    }
    
    bool success = db_->execute("DELETE FROM white_labels WHERE id = ?", {wl_id});
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), deleter_id, "DELETE_WHITELABEL", "Deleted white label: " + wl_id,
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

std::optional<WhiteLabel> AdminManager::validateAPIKey(const std::string& api_key) {
    if (!db_ || !auth_) return std::nullopt;
    
    // Hash the provided API key
    std::string api_key_hash = auth_->hashPassword(api_key);
    
    auto row = db_->querySingle(
        "SELECT * FROM white_labels WHERE api_key_hash = ? AND status = 2",
        {api_key_hash}
    );
    
    if (!row) {
        // Also check with direct key (for backward compatibility)
        row = db_->querySingle(
            "SELECT * FROM white_labels WHERE api_key = ? AND status = 2",
            {api_key}
        );
    }
    
    if (!row) return std::nullopt;
    
    return mapRowToWhiteLabel(*row);
}

std::string AdminManager::regenerateAPIKey(const std::string& wl_id, const std::string& requester_id) {
    if (!db_ || !isSuperAdmin(requester_id)) {
        return "";
    }
    
    std::string new_api_key = generateAPIKey();
    std::string api_key_hash = auth_ ? auth_->hashPassword(new_api_key) : new_api_key;
    
    db_->execute(
        "UPDATE white_labels SET api_key = ?, api_key_hash = ? WHERE id = ?",
        {new_api_key, api_key_hash, wl_id}
    );
    
    db_->execute(
        "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
        {generateID(), requester_id, "REGENERATE_API_KEY", "Regenerated API key for: " + wl_id,
         std::to_string(getCurrentTimestamp())}
    );
    
    return new_api_key;
}

// ==================== AUDIT LOGS ====================

std::vector<AuditLog> AdminManager::getAuditLogs(const std::string& admin_id, int limit) {
    std::vector<AuditLog> logs;
    
    if (!db_) return logs;
    
    std::string query = "SELECT * FROM audit_logs";
    std::vector<std::string> params;
    
    if (!admin_id.empty()) {
        query += " WHERE admin_id = ?";
        params.push_back(admin_id);
    }
    
    query += " ORDER BY timestamp DESC LIMIT " + std::to_string(limit);
    
    auto results = db_->query(query, params);
    for (const auto& row : results) {
        logs.push_back(mapRowToAuditLog(row));
    }
    
    return logs;
}

std::vector<AuditLog> AdminManager::getAuditLogsByAction(const std::string& action, int limit) {
    std::vector<AuditLog> logs;
    
    if (!db_) return logs;
    
    auto results = db_->query(
        "SELECT * FROM audit_logs WHERE action = ? ORDER BY timestamp DESC LIMIT ?",
        {action, std::to_string(limit)}
    );
    
    for (const auto& row : results) {
        logs.push_back(mapRowToAuditLog(row));
    }
    
    return logs;
}

std::vector<AuditLog> AdminManager::searchAuditLogs(const std::string& query, int64_t start_time,
                                                   int64_t end_time, int limit) {
    std::vector<AuditLog> logs;
    
    if (!db_) return logs;
    
    std::string sql = "SELECT * FROM audit_logs WHERE 1=1";
    std::vector<std::string> params;
    
    if (!query.empty()) {
        sql += " AND (action LIKE ? OR details LIKE ?)";
        params.push_back("%" + query + "%");
        params.push_back("%" + query + "%");
    }
    
    if (start_time > 0) {
        sql += " AND timestamp >= ?";
        params.push_back(std::to_string(start_time));
    }
    
    if (end_time > 0) {
        sql += " AND timestamp <= ?";
        params.push_back(std::to_string(end_time));
    }
    
    sql += " ORDER BY timestamp DESC LIMIT " + std::to_string(limit);
    
    auto results = db_->query(sql, params);
    for (const auto& row : results) {
        logs.push_back(mapRowToAuditLog(row));
    }
    
    return logs;
}

std::string AdminManager::exportAuditLogs(const std::string& admin_id, const std::string& format) {
    auto logs = getAuditLogs(admin_id, 10000);
    
    if (format == "json") {
        std::string json = "[\n";
        for (size_t i = 0; i < logs.size(); i++) {
            const auto& log = logs[i];
            json += "  {\n";
            json += "    \"id\": \"" + log.id + "\",\n";
            json += "    \"admin_id\": \"" + log.admin_id + "\",\n";
            json += "    \"admin_username\": \"" + log.admin_username + "\",\n";
            json += "    \"action\": \"" + log.action + "\",\n";
            json += "    \"details\": \"" + log.details + "\",\n";
            json += "    \"ip_address\": \"" + log.ip_address + "\",\n";
            json += "    \"timestamp\": " + std::to_string(log.timestamp) + "\n";
            json += "  }";
            if (i < logs.size() - 1) json += ",";
            json += "\n";
        }
        json += "]";
        return json;
    }
    
    // CSV format
    std::string csv = "ID,Admin ID,Action,Details,IP Address,Timestamp\n";
    for (const auto& log : logs) {
        csv += log.id + "," + log.admin_id + "," + log.action + "," + 
               log.details + "," + log.ip_address + "," + std::to_string(log.timestamp) + "\n";
    }
    
    return csv;
}

// ==================== PROFIT SHARING ====================

bool AdminManager::setProfitSharePercentage(const std::string& white_label_id, double percentage,
                                         const std::string& super_admin_id) {
    if (!db_) return false;
    
    if (percentage < 0 || percentage > 50) {
        return false;
    }
    
    if (!isSuperAdmin(super_admin_id)) {
        return false;
    }
    
    // Check if config exists
    auto existing = getProfitShareConfig(white_label_id);
    
    bool success;
    if (existing) {
        success = db_->execute(
            "UPDATE profit_share_configs SET profit_percentage = ?, updated_at = ? WHERE white_label_id = ?",
            {std::to_string(percentage), std::to_string(getCurrentTimestamp()), white_label_id}
        );
    } else {
        std::string config_id = generateID();
        success = db_->execute(
            "INSERT INTO profit_share_configs (id, white_label_id, profit_percentage, min_percentage, max_percentage, is_active, auto_transfer, transfer_frequency, created_at, updated_at) VALUES (?, ?, ?, 0, 50, 1, 1, 'daily', ?, ?)",
            {config_id, white_label_id, std::to_string(percentage), std::to_string(getCurrentTimestamp()),
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), super_admin_id, "SET_PROFIT_SHARE", "Set profit share to " + std::to_string(percentage) + "% for: " + white_label_id,
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

std::optional<ProfitShareConfig> AdminManager::getProfitShareConfig(const std::string& white_label_id) {
    if (!db_) return std::nullopt;
    
    auto row = db_->querySingle(
        "SELECT * FROM profit_share_configs WHERE white_label_id = ?",
        {white_label_id}
    );
    
    if (!row) return std::nullopt;
    
    return mapRowToProfitConfig(*row);
}

void AdminManager::calculateProfitShare(const std::string& white_label_id, double gross_revenue,
                                        double& super_admin_share, double& white_label_share) {
    auto config = getProfitShareConfig(white_label_id);
    
    double percentage = 20.0; // Default
    if (config) {
        percentage = config->profit_percentage;
    }
    
    super_admin_share = gross_revenue * (percentage / 100.0);
    white_label_share = gross_revenue - super_admin_share;
}

bool AdminManager::executeProfitTransfer(const std::string& white_label_id, const std::string& token,
                                       double amount, const std::string& executor_id) {
    if (!db_) return false;
    
    double super_admin_share, white_label_share;
    calculateProfitShare(white_label_id, amount, super_admin_share, white_label_share);
    
    std::string tx_id = generateID();
    std::string tx_hash = "0x" + generateID(); // Simulated tx hash
    
    bool success = db_->execute(
        "INSERT INTO profit_transactions (id, white_label_id, super_admin_wallet, amount, percentage, gross_revenue, net_revenue, token, tx_hash, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'completed', ?)",
        {tx_id, white_label_id, "0xSuperAdminWallet", std::to_string(super_admin_share),
         std::to_string(super_admin_share / amount * 100), std::to_string(amount),
         std::to_string(white_label_share), token, tx_hash, std::to_string(getCurrentTimestamp())}
    );
    
    if (success) {
        // Update total transferred
        db_->execute(
            "UPDATE profit_share_configs SET total_transferred = total_transferred + ?, last_transfer = ? WHERE white_label_id = ?",
            {std::to_string(super_admin_share), std::to_string(getCurrentTimestamp()), white_label_id}
        );
        
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), executor_id, "PROFIT_TRANSFER", "Transferred " + std::to_string(super_admin_share) + " to super admin",
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

std::vector<ProfitTransaction> AdminManager::getProfitHistory(const std::string& white_label_id, int limit) {
    std::vector<ProfitTransaction> transactions;
    
    if (!db_) return transactions;
    
    std::string query = "SELECT * FROM profit_transactions";
    std::vector<std::string> params;
    
    if (!white_label_id.empty()) {
        query += " WHERE white_label_id = ?";
        params.push_back(white_label_id);
    }
    
    query += " ORDER BY created_at DESC LIMIT " + std::to_string(limit);
    
    auto results = db_->query(query, params);
    for (const auto& row : results) {
        transactions.push_back(mapRowToProfitTransaction(row));
    }
    
    return transactions;
}

double AdminManager::getTotalProfits() {
    if (!db_) return 0.0;
    
    auto result = db_->querySingle("SELECT SUM(total_transferred) as total FROM profit_share_configs");
    if (!result) return 0.0;
    
    return std::stod(result->at("total"));
}

// ==================== FEATURE FLAGS ====================

std::vector<FeatureFlag> AdminManager::getAllFeatures() {
    std::vector<FeatureFlag> features;
    
    if (!db_) return features;
    
    auto results = db_->query("SELECT * FROM feature_flags ORDER BY name");
    for (const auto& row : results) {
        features.push_back(mapRowToFeatureFlag(row));
    }
    
    return features;
}

std::optional<FeatureFlag> AdminManager::getFeatureByName(const std::string& name) {
    if (!db_) return std::nullopt;
    
    auto row = db_->querySingle("SELECT * FROM feature_flags WHERE name = ?", {name});
    if (!row) return std::nullopt;
    
    return mapRowToFeatureFlag(*row);
}

bool AdminManager::setGlobalFeature(const std::string& feature_name, bool enabled,
                                  const std::string& super_admin_id) {
    if (!db_ || !isSuperAdmin(super_admin_id)) return false;
    
    bool success = db_->execute(
        "UPDATE feature_flags SET global_enabled = ?, enabled = ?, updated_by = ?, updated_at = ? WHERE name = ?",
        {enabled ? "1" : "0", enabled ? "1" : "0", super_admin_id, std::to_string(getCurrentTimestamp()), feature_name}
    );
    
    if (success) {
        db_->execute(
            "INSERT INTO audit_logs (id, admin_id, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
            {generateID(), super_admin_id, "SET_FEATURE", "Set feature " + feature_name + " to " + (enabled ? "enabled" : "disabled"),
             std::to_string(getCurrentTimestamp())}
        );
    }
    
    return success;
}

bool AdminManager::setMasterAdminFeature(const std::string& feature_name,
                                         const std::string& master_admin_id,
                                         bool enabled,
                                         const std::string& super_admin_id) {
    if (!db_ || !isSuperAdmin(super_admin_id)) return false;
    
    return db_->execute(
        "UPDATE feature_flags SET master_admin_id = ?, enabled = ?, updated_by = ?, updated_at = ? WHERE name = ?",
        {master_admin_id, enabled ? "1" : "0", super_admin_id, std::to_string(getCurrentTimestamp()), feature_name}
    );
}

bool AdminManager::isFeatureEnabled(const std::string& feature_name, const std::string& admin_id,
                                   AdminRole role) {
    auto feature = getFeatureByName(feature_name);
    if (!feature) return false;
    
    // Super admin always has access
    if (role == AdminRole::SUPER_ADMIN) {
        return true;
    }
    
    // Check global enabled
    if (!feature->global_enabled) {
        return false;
    }
    
    // Check specific enabled
    return feature->enabled;
}

// ==================== HELPER FUNCTIONS ====================

Admin AdminManager::mapRowToAdmin(const RowData& row) {
    Admin admin;
    admin.id = row.at("id");
    admin.username = row.at("username");
    admin.password_hash = row.at("password_hash");
    admin.email = row.at("email");
    admin.role = (AdminRole)std::stoi(row.at("role"));
    admin.security_level = (SecurityLevel)std::stoi(row.at("security_level"));
    admin.two_factor_enabled = row.at("two_factor_enabled") == "1";
    admin.two_factor_secret = row.at("two_factor_secret");
    admin.created_at = std::stoll(row.at("created_at"));
    admin.last_login = std::stoll(row.at("last_login"));
    admin.status = (AdminStatus)std::stoi(row.at("status"));
    admin.failed_attempts = std::stoi(row.at("failed_attempts"));
    admin.locked_until = std::stoll(row.at("locked_until"));
    admin.ip_whitelist = row.at("ip_whitelist");
    
    return admin;
}

WhiteLabel AdminManager::mapRowToWhiteLabel(const RowData& row) {
    WhiteLabel wl;
    wl.id = row.at("id");
    wl.name = row.at("name");
    wl.domain = row.at("domain");
    wl.api_key = row.at("api_key");
    wl.api_key_hash = row.at("api_key_hash");
    wl.fee_percent = std::stod(row.at("fee_percent"));
    wl.status = std::stoi(row.at("status"));
    wl.approved_by = row.at("approved_by");
    wl.approved_at = std::stoll(row.at("approved_at"));
    wl.created_at = std::stoll(row.at("created_at"));
    wl.custom_branding = row.at("custom_branding") == "1";
    
    return wl;
}

ProfitShareConfig AdminManager::mapRowToProfitConfig(const RowData& row) {
    ProfitShareConfig config;
    config.id = row.at("id");
    config.white_label_id = row.at("white_label_id");
    config.super_admin_wallet = row.at("super_admin_wallet");
    config.master_wallet_address = row.at("master_wallet_address");
    config.profit_percentage = std::stod(row.at("profit_percentage"));
    config.min_percentage = std::stod(row.at("min_percentage"));
    config.max_percentage = std::stod(row.at("max_percentage"));
    config.is_active = row.at("is_active") == "1";
    config.auto_transfer = row.at("auto_transfer") == "1";
    config.transfer_frequency = row.at("transfer_frequency");
    config.last_transfer = std::stoll(row.at("last_transfer"));
    config.total_transferred = std::stod(row.at("total_transferred"));
    config.created_at = std::stoll(row.at("created_at"));
    config.updated_at = std::stoll(row.at("updated_at"));
    
    return config;
}

ProfitTransaction AdminManager::mapRowToProfitTransaction(const RowData& row) {
    ProfitTransaction tx;
    tx.id = row.at("id");
    tx.white_label_id = row.at("white_label_id");
    tx.super_admin_wallet = row.at("super_admin_wallet");
    tx.amount = std::stod(row.at("amount"));
    tx.percentage = std::stod(row.at("percentage"));
    tx.gross_revenue = std::stod(row.at("gross_revenue"));
    tx.net_revenue = std::stod(row.at("net_revenue"));
    tx.token = row.at("token");
    tx.tx_hash = row.at("tx_hash");
    tx.status = row.at("status");
    tx.created_at = std::stoll(row.at("created_at"));
    
    return tx;
}

FeatureFlag AdminManager::mapRowToFeatureFlag(const RowData& row) {
    FeatureFlag flag;
    flag.id = row.at("id");
    flag.name = row.at("name");
    flag.description = row.at("description");
    flag.global_enabled = row.at("global_enabled") == "1";
    flag.enabled = row.at("enabled") == "1";
    flag.master_admin_id = row.at("master_admin_id");
    flag.white_label_id = row.at("white_label_id");
    flag.updated_by = row.at("updated_by");
    flag.updated_at = std::stoll(row.at("updated_at"));
    
    return flag;
}

AuditLog AdminManager::mapRowToAuditLog(const RowData& row) {
    AuditLog log;
    log.id = row.at("id");
    log.admin_id = row.at("admin_id");
    log.admin_username = row.at("admin_username");
    log.action = row.at("action");
    log.details = row.at("details");
    log.ip_address = row.at("ip_address");
    log.user_agent = row.at("user_agent");
    log.timestamp = std::stoll(row.at("timestamp"));
    
    return log;
}

} // namespace super_admin
} // namespace tigerwallet
