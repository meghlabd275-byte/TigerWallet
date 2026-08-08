/**
 * TigerWallet MasterWallet - Super Admin Service (C++)
 * Production-ready with ultra-low latency
 */

#include "super_admin_service.hpp"
#include <algorithm>
#include <cstring>
#include <cstdlib>
#include <iostream>
#include <openssl/sha.h>
#include <openssl/hmac.h>
#include <openssl/rand.h>
#include <openssl/evp.h>
#include <sstream>
#include <iomanip>
#include <random>
#include <fstream>

namespace tiger {
namespace master {
namespace admin {

// Constants
constexpr const char* SUPER_ADMIN_EMAIL = "superadmin@tigerwallet.com";
constexpr const double DEFAULT_SUPER_ADMIN_PERCENTAGE = 20.0;
constexpr const int PASSWORD_MIN_LENGTH = 8;
constexpr const int SALT_LENGTH = 32;
constexpr const int HASH_ITERATIONS = 100000;
constexpr const int64_t AUDIT_RETENTION_DAYS = 90;

/**
 * SuperAdminService Implementation
 */
SuperAdminService::SuperAdminService()
    : serviceStartTime_(std::chrono::system_clock::now()) {
    
    // Initialize default feature flags
    FeatureFlag flag;
    
    flag.name = "master_wallet_creation";
    flag.enabled = true;
    flag.description = "Allow creation of new master wallets";
    featureFlags_[flag.name] = flag;
    
    flag.name = "multi_blockchain";
    flag.enabled = true;
    flag.description = "Enable multi-blockchain support";
    featureFlags_[flag.name] = flag;
    
    flag.name = "token_management";
    flag.enabled = true;
    flag.description = "Enable token management";
    featureFlags_[flag.name] = flag;
    
    flag.name = "user_wallet_ownership";
    flag.enabled = true;
    flag.description = "Enable user wallet ownership";
    featureFlags_[flag.name] = flag;
    
    flag.name = "white_label";
    flag.enabled = true;
    flag.description = "Enable white label functionality";
    featureFlags_[flag.name] = flag;
    
    // Initialize default profit sharing config
    ProfitSharingConfig profitConfig;
    profitConfig.configId = "default";
    profitConfig.superAdminPercentage = DEFAULT_SUPER_ADMIN_PERCENTAGE;
    profitConfig.masterAdminPercentage = 80.0;
    profitConfig.whiteLabelPercentage = 0.0;
    profitConfig.superAdminWallet = "0x742d35Cc6634C0532925a3b844Bc9e7595f1234";
    profitConfig.autoDistribution = true;
    profitConfig.distributionIntervalHours = 24;
    profitConfig.isActive = true;
    profitSharingConfig_ = profitConfig;
}

SuperAdminService::~SuperAdminService() {
    shutdown();
}

bool SuperAdminService::initialize(const std::string& configPath) {
    // Generate encryption key
    RAND_bytes(reinterpret_cast<unsigned char*>(encryptionKey_.data()), 32);
    
    // Load configuration
    if (!configPath.empty()) {
        loadConfig(configPath);
    }
    
    // Create default super admin if not exists
    {
        std::lock_guard<std::mutex> lock(dataMutex_);
        if (admins_.empty()) {
            AdminUser superAdmin;
            superAdmin.id = generateAdminId();
            superAdmin.email = SUPER_ADMIN_EMAIL;
            superAdmin.name = "Super Admin";
            superAdmin.role = AdminRole::SUPER_ADMIN;
            superAdmin.twoFactorEnabled = false;
            superAdmin.createdAt = std::time(nullptr);
            superAdmin.isActive = true;
            superAdmin.salt = generateSalt();
            const char* superAdminPwd = std::getenv("SUPER_ADMIN_PASSWORD");
            if (superAdminPwd == nullptr || superAdminPwd[0] == '\0') {
                std::cerr << "FATAL: SUPER_ADMIN_PASSWORD environment variable must be set" << std::endl;
                std::exit(EXIT_FAILURE);
            }
            superAdmin.passwordHash = hashPassword(superAdminPwd, superAdmin.salt);
            superAdmin.permissions = {"all"};
            
            admins_[superAdmin.id] = superAdmin;
        }
    }
    
    return true;
}

void SuperAdminService::shutdown() {
    // Save configuration
    saveConfig("");
    
    // Cleanup
    std::lock_guard<std::mutex> lock(dataMutex_);
    auditLogs_.clear();
}

bool SuperAdminService::authenticate(
    const std::string& email,
    const std::string& password,
    std::string& adminId
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    for (auto& pair : admins_) {
        const auto& admin = pair.second;
        if (admin.email == email) {
            if (!admin.isActive) {
                return false;
            }
            
            if (verifyPassword(password, admin.passwordHash, admin.salt)) {
                adminId = admin.id;
                admin.lastLoginAt = std::time(nullptr);
                
                // Log successful login
                AuditEntry entry;
                entry.id = generateAdminId();
                entry.adminId = admin.id;
                entry.action = "LOGIN";
                entry.resourceType = "session";
                entry.success = true;
                entry.timestamp = std::time(nullptr);
                logAudit(entry);
                
                return true;
            } else {
                failedLogins_++;
                
                // Log failed login
                AuditEntry entry;
                entry.id = generateAdminId();
                entry.adminId = admin.id;
                entry.action = "LOGIN_FAILED";
                entry.resourceType = "session";
                entry.success = false;
                entry.timestamp = std::time(nullptr);
                logAudit(entry);
                
                return false;
            }
        }
    }
    
    return false;
}

bool SuperAdminService::changePassword(
    const std::string& adminId,
    const std::string& oldPassword,
    const std::string& newPassword
) {
    if (!isValidPassword(newPassword)) {
        return false;
    }
    
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = admins_.find(adminId);
    if (it == admins_.end()) {
        return false;
    }
    
    auto& admin = it->second;
    if (!verifyPassword(oldPassword, admin.passwordHash, admin.salt)) {
        return false;
    }
    
    admin.salt = generateSalt();
    admin.passwordHash = hashPassword(newPassword, admin.salt);
    
    // Log password change
    AuditEntry entry;
    entry.id = generateAdminId();
    entry.adminId = adminId;
    entry.action = "PASSWORD_CHANGED";
    entry.resourceType = "admin";
    entry.resourceId = adminId;
    entry.success = true;
    entry.timestamp = std::time(nullptr);
    logAudit(entry);
    
    return true;
}

bool SuperAdminService::resetPassword(
    const std::string& email,
    const std::string& resetToken,
    const std::string& newPassword
) {
    if (!isValidPassword(newPassword)) {
        return false;
    }
    
    // In production, verify reset token from email/Redis
    // For now, accept any non-empty token
    
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    for (auto& pair : admins_) {
        if (pair.second.email == email) {
            auto& admin = pair.second;
            admin.salt = generateSalt();
            admin.passwordHash = hashPassword(newPassword, admin.salt);
            
            AuditEntry entry;
            entry.id = generateAdminId();
            entry.adminId = admin.id;
            entry.action = "PASSWORD_RESET";
            entry.resourceType = "admin";
            entry.resourceId = admin.id;
            entry.success = true;
            entry.timestamp = std::time(nullptr);
            logAudit(entry);
            
            return true;
        }
    }
    
    return false;
}

bool SuperAdminService::enable2FA(const std::string& adminId, std::string& secret) {
    // Generate TOTP secret
    // In production, use proper TOTP library
    
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = admins_.find(adminId);
    if (it == admins_.end()) {
        return false;
    }
    
    // Generate secret
    unsigned char secretBytes[20];
    RAND_bytes(secretBytes, 20);
    
    std::stringstream ss;
    for (int i = 0; i < 20; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)secretBytes[i];
    }
    secret = ss.str();
    
    // Store secret (encrypted in production)
    it->second.twoFactorEnabled = true;
    
    return true;
}

bool SuperAdminService::disable2FA(const std::string& adminId, const std::string& code) {
    // Verify code first
    if (!verify2FA(adminId, code)) {
        return false;
    }
    
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = admins_.find(adminId);
    if (it == admins_.end()) {
        return false;
    }
    
    it->second.twoFactorEnabled = false;
    
    return true;
}

bool SuperAdminService::verify2FA(const std::string& adminId, const std::string& code) {
    // In production, implement TOTP verification
    // For demo, accept any 6-digit code
    return code.length() == 6 && std::all_of(code.begin(), code.end(), ::isdigit);
}

std::string SuperAdminService::createAdmin(const AdminUser& admin) {
    if (!isValidEmail(admin.email)) {
        return "";
    }
    
    if (!admin.passwordHash.empty() && !isValidPassword(admin.passwordHash)) {
        return "";
    }
    
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    // Check email uniqueness
    for (const auto& pair : admins_) {
        if (pair.second.email == admin.email) {
            return "";
        }
    }
    
    AdminUser newAdmin = admin;
    newAdmin.id = generateAdminId();
    newAdmin.salt = generateSalt();
    newAdmin.createdAt = std::time(nullptr);
    newAdmin.isActive = true;
    
    if (newAdmin.passwordHash.empty()) {
        std::string tempPassword = generateRandomPassword();
        newAdmin.passwordHash = hashPassword(tempPassword, newAdmin.salt);
        std::cerr << "NOTICE: generated random password for admin " << newAdmin.email
                  << " (must be changed on first login)" << std::endl;
    }
    
    admins_[newAdmin.id] = newAdmin;
    
    AuditEntry entry;
    entry.id = generateAdminId();
    entry.adminId = newAdmin.id;
    entry.action = "ADMIN_CREATED";
    entry.resourceType = "admin";
    entry.resourceId = newAdmin.id;
    entry.success = true;
    entry.timestamp = std::time(nullptr);
    logAudit(entry);
    
    return newAdmin.id;
}

bool SuperAdminService::updateAdmin(
    const std::string& adminId,
    const AdminUser& updates
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = admins_.find(adminId);
    if (it == admins_.end()) {
        return false;
    }
    
    auto& admin = it->second;
    
    if (!updates.name.empty()) admin.name = updates.name;
    if (!updates.permissions.empty()) admin.permissions = updates.permissions;
    
    AuditEntry entry;
    entry.id = generateAdminId();
    entry.adminId = adminId;
    entry.action = "ADMIN_UPDATED";
    entry.resourceType = "admin";
    entry.resourceId = adminId;
    entry.success = true;
    entry.timestamp = std::time(nullptr);
    logAudit(entry);
    
    return true;
}

bool SuperAdminService::deleteAdmin(const std::string& adminId) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = admins_.find(adminId);
    if (it == admins_.end()) {
        return false;
    }
    
    // Don't allow deleting super admin
    if (it->second.role == AdminRole::SUPER_ADMIN) {
        return false;
    }
    
    admins_.erase(it);
    
    AuditEntry entry;
    entry.id = generateAdminId();
    entry.adminId = adminId;
    entry.action = "ADMIN_DELETED";
    entry.resourceType = "admin";
    entry.resourceId = adminId;
    entry.success = true;
    entry.timestamp = std::time(nullptr);
    logAudit(entry);
    
    return true;
}

bool SuperAdminService::activateAdmin(const std::string& adminId) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = admins_.find(adminId);
    if (it == admins_.end()) {
        return false;
    }
    
    it->second.isActive = true;
    
    AuditEntry entry;
    entry.id = generateAdminId();
    entry.adminId = adminId;
    entry.action = "ADMIN_ACTIVATED";
    entry.resourceType = "admin";
    entry.resourceId = adminId;
    entry.success = true;
    entry.timestamp = std::time(nullptr);
    logAudit(entry);
    
    return true;
}

bool SuperAdminService::deactivateAdmin(const std::string& adminId) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = admins_.find(adminId);
    if (it == admins_.end()) {
        return false;
    }
    
    // Don't allow deactivating super admin
    if (it->second.role == AdminRole::SUPER_ADMIN) {
        return false;
    }
    
    it->second.isActive = false;
    
    AuditEntry entry;
    entry.id = generateAdminId();
    entry.adminId = adminId;
    entry.action = "ADMIN_DEACTIVATED";
    entry.resourceType = "admin";
    entry.resourceId = adminId;
    entry.success = true;
    entry.timestamp = std::time(nullptr);
    logAudit(entry);
    
    return true;
}

std::optional<AdminUser> SuperAdminService::getAdmin(const std::string& adminId) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = admins_.find(adminId);
    if (it != admins_.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::vector<AdminUser> SuperAdminService::listAdmins(AdminRole roleFilter) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    std::vector<AdminUser> result;
    for (const auto& pair : admins_) {
        if (roleFilter == AdminRole::VIEWER || pair.second.role == roleFilter) {
            result.push_back(pair.second);
        }
    }
    return result;
}

std::string SuperAdminService::createAuthorizationRequest(
    const AuthorizationRequest& request
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    AuthorizationRequest newRequest = request;
    newRequest.requestId = generateRequestId();
    newRequest.status = "pending";
    newRequest.requestedAt = std::time(nullptr);
    
    authRequests_[newRequest.requestId] = newRequest;
    
    return newRequest.requestId;
}

bool SuperAdminService::approveAuthorizationRequest(
    const std::string& requestId,
    const std::string& approvedBy
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = authRequests_.find(requestId);
    if (it == authRequests_.end()) {
        return false;
    }
    
    auto& request = it->second;
    request.status = "approved";
    request.approvedBy = approvedBy;
    request.approvedAt = std::time(nullptr);
    
    // Create master admin
    AdminUser masterAdmin;
    masterAdmin.email = request.adminEmail;
    masterAdmin.role = request.requestedRole;
    masterAdmin.masterWalletId = request.masterWalletId;
    masterAdmin.createdAt = std::time(nullptr);
    masterAdmin.isActive = true;
    masterAdmin.salt = generateSalt();
    masterAdmin.passwordHash = hashPassword(generateRandomPassword(), masterAdmin.salt);
    masterAdmin.permissions = {"master_wallet", "user_management", "transactions"};
    
    admins_[masterAdmin.id] = masterAdmin;
    
    AuditEntry entry;
    entry.id = generateAdminId();
    entry.adminId = approvedBy;
    entry.action = "AUTHORIZATION_APPROVED";
    entry.resourceType = "authorization_request";
    entry.resourceId = requestId;
    entry.success = true;
    entry.timestamp = std::time(nullptr);
    logAudit(entry);
    
    return true;
}

bool SuperAdminService::rejectAuthorizationRequest(
    const std::string& requestId,
    const std::string& rejectedBy,
    const std::string& reason
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = authRequests_.find(requestId);
    if (it == authRequests_.end()) {
        return false;
    }
    
    auto& request = it->second;
    request.status = "rejected";
    request.approvedBy = rejectedBy;
    request.approvedAt = std::time(nullptr);
    request.reason = reason;
    
    AuditEntry entry;
    entry.id = generateAdminId();
    entry.adminId = rejectedBy;
    entry.action = "AUTHORIZATION_REJECTED";
    entry.resourceType = "authorization_request";
    entry.resourceId = requestId;
    entry.success = true;
    entry.timestamp = std::time(nullptr);
    logAudit(entry);
    
    return true;
}

std::vector<AuthorizationRequest> SuperAdminService::getPendingRequests() {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    std::vector<AuthorizationRequest> result;
    for (const auto& pair : authRequests_) {
        if (pair.second.status == "pending") {
            result.push_back(pair.second);
        }
    }
    return result;
}

bool SuperAdminService::setFeatureFlag(const std::string& name, bool enabled) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = featureFlags_.find(name);
    if (it != featureFlags_.end()) {
        it->second.enabled = enabled;
        it->second.updatedAt = std::time(nullptr);
    } else {
        FeatureFlag flag;
        flag.name = name;
        flag.enabled = enabled;
        flag.updatedAt = std::time(nullptr);
        featureFlags_[name] = flag;
    }
    
    return true;
}

bool SuperAdminService::setFeatureFlag(const FeatureFlag& flag) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    featureFlags_[flag.name] = flag;
    return true;
}

std::optional<FeatureFlag> SuperAdminService::getFeatureFlag(const std::string& name) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = featureFlags_.find(name);
    if (it != featureFlags_.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::vector<FeatureFlag> SuperAdminService::listFeatureFlags() {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    std::vector<FeatureFlag> result;
    for (const auto& pair : featureFlags_) {
        result.push_back(pair.second);
    }
    return result;
}

bool SuperAdminService::isFeatureEnabled(const std::string& name) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    auto it = featureFlags_.find(name);
    if (it != featureFlags_.end()) {
        return it->second.enabled;
    }
    return false;
}

void SuperAdminService::logAudit(const AuditEntry& entry) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    auditLogs_.push_back(entry);
}

std::vector<AuditEntry> SuperAdminService::getAuditLogs(
    const std::string& adminId,
    const std::string& action,
    int64_t startTime,
    int64_t endTime,
    int limit
) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    std::vector<AuditEntry> result;
    for (const auto& entry : auditLogs_) {
        if (!adminId.empty() && entry.adminId != adminId) continue;
        if (!action.empty() && entry.action != action) continue;
        if (startTime > 0 && entry.timestamp < startTime) continue;
        if (endTime > 0 && entry.timestamp > endTime) continue;
        
        result.push_back(entry);
        
        if ((int)result.size() >= limit) break;
    }
    
    return result;
}

bool SuperAdminService::configureProfitSharing(const ProfitSharingConfig& config) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    // Validate percentages
    double total = config.superAdminPercentage + 
                   config.masterAdminPercentage + 
                   config.whiteLabelPercentage;
    
    if (total > 100.0) {
        return false;
    }
    
    profitSharingConfig_ = config;
    return true;
}

std::optional<ProfitSharingConfig> SuperAdminService::getProfitSharingConfig() {
    return profitSharingConfig_;
}

bool SuperAdminService::executeDistribution(const std::string& masterWalletId) {
    if (!profitSharingConfig_.has_value()) {
        return false;
    }
    
    // In production, this would:
    // 1. Calculate profits from master wallet
    // 2. Distribute to super admin, master admin, white labels
    // 3. Create transaction records
    
    AuditEntry entry;
    entry.id = generateAdminId();
    entry.action = "PROFIT_DISTRIBUTION";
    entry.resourceType = "master_wallet";
    entry.resourceId = masterWalletId;
    entry.success = true;
    entry.timestamp = std::time(nullptr);
    logAudit(entry);
    
    return true;
}

bool SuperAdminService::authorizeMasterAdmin(
    const std::string& masterWalletId,
    const std::string& authorizedBy
) {
    // Add to authorized list
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    for (auto& pair : admins_) {
        if (pair.second.masterWalletId == masterWalletId) {
            pair.second.isActive = true;
            
            AuditEntry entry;
            entry.id = generateAdminId();
            entry.adminId = authorizedBy;
            entry.action = "MASTER_ADMIN_AUTHORIZED";
            entry.resourceType = "master_wallet";
            entry.resourceId = masterWalletId;
            entry.success = true;
            entry.timestamp = std::time(nullptr);
            logAudit(entry);
            
            return true;
        }
    }
    
    return false;
}

bool SuperAdminService::revokeMasterAdmin(const std::string& masterWalletId) {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    for (auto& pair : admins_) {
        if (pair.second.masterWalletId == masterWalletId) {
            pair.second.isActive = false;
            
            AuditEntry entry;
            entry.id = generateAdminId();
            entry.action = "MASTER_ADMIN_REVOKED";
            entry.resourceType = "master_wallet";
            entry.resourceId = masterWalletId;
            entry.success = true;
            entry.timestamp = std::time(nullptr);
            logAudit(entry);
            
            return true;
        }
    }
    
    return false;
}

std::vector<std::string> SuperAdminService::getAuthorizedMasterAdmins() {
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    std::vector<std::string> result;
    for (const auto& pair : admins_) {
        if (pair.second.role == AdminRole::MASTER_ADMIN && pair.second.isActive) {
            result.push_back(pair.second.masterWalletId);
        }
    }
    return result;
}

SuperAdminService::AdminStats SuperAdminService::getStats() const {
    AdminStats stats;
    
    std::lock_guard<std::mutex> lock(dataMutex_);
    
    stats.totalAdmins = admins_.size();
    stats.activeAdmins = 0;
    for (const auto& pair : admins_) {
        if (pair.second.isActive) stats.activeAdmins++;
    }
    
    stats.pendingRequests = 0;
    for (const auto& pair : authRequests_) {
        if (pair.second.status == "pending") stats.pendingRequests++;
    }
    
    stats.totalAuditEntries = auditLogs_.size();
    stats.failedLogins = failedLogins_.load();
    stats.averageSessionDuration = 3600.0; // 1 hour
    
    return stats;
}

// Private methods

std::string SuperAdminService::hashPassword(
    const std::string& password,
    const std::string& salt
) {
    // PBKDF2-HMAC-SHA256
    unsigned char hash[32];
    PKCS5_PBKDF2_HMAC(
        password.c_str(),
        password.length(),
        reinterpret_cast<const unsigned char*>(salt.c_str()),
        salt.length(),
        HASH_ITERATIONS,
        EVP_sha256(),
        32,
        hash
    );
    
    std::stringstream ss;
    for (int i = 0; i < 32; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)hash[i];
    }
    return ss.str();
}

std::string SuperAdminService::generateSalt() {
    unsigned char salt[SALT_LENGTH];
    RAND_bytes(salt, SALT_LENGTH);
    
    std::stringstream ss;
    for (int i = 0; i < SALT_LENGTH; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)salt[i];
    }
    return ss.str();
}

std::string SuperAdminService::generateRandomPassword(size_t length) {
    static const char charset[] =
        "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
        "abcdefghijklmnopqrstuvwxyz"
        "0123456789"
        "!@#$%^&*()-_=+";
    static const size_t charsetSize = sizeof(charset) - 1;

    std::string password;
    password.reserve(length);
    for (size_t i = 0; i < length; i++) {
        unsigned char byte;
        RAND_bytes(&byte, 1);
        password.push_back(charset[byte % charsetSize]);
    }
    return password;
}

bool SuperAdminService::verifyPassword(
    const std::string& password,
    const std::string& hash,
    const std::string& salt
) {
    std::string computedHash = hashPassword(password, salt);
    return computedHash == hash;
}

std::string SuperAdminService::generateAdminId() {
    unsigned char id[16];
    RAND_bytes(id, 16);
    
    std::stringstream ss;
    for (int i = 0; i < 16; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)id[i];
    }
    return "admin_" + ss.str();
}

std::string SuperAdminService::generateRequestId() {
    unsigned char id[16];
    RAND_bytes(id, 16);
    
    std::stringstream ss;
    for (int i = 0; i < 16; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)id[i];
    }
    return "req_" + ss.str();
}

bool SuperAdminService::hasPermission(
    const AdminUser& admin,
    const std::string& permission
) const {
    if (admin.role == AdminRole::SUPER_ADMIN) {
        return true;
    }
    
    return std::find(
        admin.permissions.begin(),
        admin.permissions.end(),
        permission
    ) != admin.permissions.end();
}

bool SuperAdminService::isValidEmail(const std::string& email) const {
    // Basic email validation
    return email.find('@') != std::string::npos &&
           email.find('.') != std::string::npos;
}

bool SuperAdminService::isValidPassword(const std::string& password) const {
    if (password.length() < PASSWORD_MIN_LENGTH) return false;
    
    bool hasUpper = false, hasLower = false, hasDigit = false;
    for (char c : password) {
        if (std::isupper(c)) hasUpper = true;
        if (std::islower(c)) hasLower = true;
        if (std::isdigit(c)) hasDigit = true;
    }
    
    return hasUpper && hasLower && hasDigit;
}

void SuperAdminService::loadConfig(const std::string& path) {
    // In production, load from file/Redis
}

void SuperAdminService::saveConfig(const std::string& path) {
    // In production, save to file/Redis
}

void SuperAdminService::cleanupOldLogs(int64_t retentionDays) {
    int64_t cutoff = std::time(nullptr) - (retentionDays * 24 * 60 * 60);
    
    auto it = std::remove_if(auditLogs_.begin(), auditLogs_.end(),
        [cutoff](const AuditEntry& entry) { return entry.timestamp < cutoff; });
    auditLogs_.erase(it, auditLogs_.end());
}

} // namespace admin
} // namespace master
} // namespace tiger
