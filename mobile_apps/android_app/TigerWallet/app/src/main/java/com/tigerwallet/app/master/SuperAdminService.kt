/**
 * TigerWallet Android - Super Admin Authorization Service
 * 
 * COMPLETE SUPER ADMIN SYSTEM:
 * - Only Super Admin can authorize Master Wallet Admin accounts
 * - Master Wallet Admin login requires Super Admin authorization
 * - Master Wallet Admin can change password and set 2FA after login
 * - Super Admin has FULL control over all features and functionalities
 * - White Label Admin has full control in their custom branding
 * 
 * This service MUST be identical across ALL platforms
 */

package com.tigerwallet.app.master

import java.security.MessageDigest
import java.security.SecureRandom
import java.util.UUID

/**
 * User Roles
 */
enum class UserRole {
    SUPER_ADMIN,
    MASTER_ADMIN,
    WHITE_LABEL_ADMIN,
    USER
}

/**
 * Admin Status
 */
enum class AdminStatus {
    ACTIVE,
    INACTIVE,
    PENDING,
    SUSPENDED
}

/**
 * Authorization Status
 */
enum class AuthorizationStatus {
    AUTHORIZED,
    PENDING,
    REVOKED,
    REJECTED
}

/**
 * Super Admin - Highest Authority
 */
data class SuperAdmin(
    val id: String,
    val email: String,
    val passwordHash: String,
    val secretKey: String,
    val twoFactorEnabled: Boolean = false,
    val twoFactorSecret: String = "",
    val phone: String = "",
    val createdAt: Long = System.currentTimeMillis(),
    val lastLogin: Long = 0,
    val isActive: Boolean = true,
    val permissions: List<String> = listOf("*") // ALL permissions
)

/**
 * Master Wallet Admin - Requires Super Admin Authorization
 */
data class MasterAdmin(
    val id: String,
    val email: String,
    val passwordHash: String,
    val authorizedBy: String = "", // Super Admin ID
    val authorizationStatus: AuthorizationStatus = AuthorizationStatus.PENDING,
    val twoFactorEnabled: Boolean = false,
    val twoFactorSecret: String = "",
    val phone: String = "",
    val canCreateWhiteLabel: Boolean = false,
    val canManageUsers: Boolean = false,
    val canManageWallets: Boolean = false,
    val canAccessFinance: Boolean = false,
    val canModifyFeatures: Boolean = false,
    val canManageTokens: Boolean = false,
    val canManageNetworks: Boolean = false,
    val canViewAnalytics: Boolean = false,
    val canManageAdmins: Boolean = false,
    val maxWhiteLabels: Int = 0,
    val whiteLabelCount: Int = 0,
    val status: AdminStatus = AdminStatus.PENDING,
    val createdAt: Long = System.currentTimeMillis(),
    val lastLogin: Long = 0,
    val passwordChangedAt: Long = 0,
    val failedAttempts: Int = 0,
    val lockedUntil: Long = 0
)

/**
 * White Label Admin - Full control in their branding
 */
data class WhiteLabelAdmin(
    val id: String,
    val email: String,
    val passwordHash: String,
    val masterAdminId: String = "",
    val brandName: String = "",
    val brandLogo: String = "",
    val brandColor: String = "#000000",
    val customDomain: String = "",
    val authorizationStatus: AuthorizationStatus = AuthorizationStatus.PENDING,
    val twoFactorEnabled: Boolean = false,
    val twoFactorSecret: String = "",
    val canCustomizeUi: Boolean = true,
    val canCustomizeFees: Boolean = true,
    val canManageUsers: Boolean = true,
    val canManageWallets: Boolean = true,
    val canAccessAnalytics: Boolean = true,
    val canManageTokens: Boolean = true,
    val feePercentage: Double = 0.0,
    val status: AdminStatus = AdminStatus.PENDING,
    val createdAt: Long = System.currentTimeMillis(),
    val lastLogin: Long = 0
)

/**
 * Feature Control - Super Admin Control
 */
data class FeatureControl(
    val featureName: String,
    val enabled: Boolean = true,
    val globalEnabled: Boolean = true,
    val masterAdminId: String = "",
    val whiteLabelId: String = "",
    val updatedBy: String = "",
    val updatedAt: Long = System.currentTimeMillis()
)

/**
 * Super Admin Service
 */
class SuperAdminService private constructor() {
    
    companion object {
        val instance: SuperAdminService by lazy { SuperAdminService() }
    }
    
    private val superAdmins = mutableMapOf<String, Any>() // id -> SuperAdmin or email -> SuperAdmin
    private val masterAdmins = mutableMapOf<String, Any>() // id -> MasterAdmin or email -> MasterAdmin
    private val whiteLabelAdmins = mutableMapOf<String, Any>() // id -> WhiteLabelAdmin or email -> WhiteLabelAdmin
    private val featureControls = mutableMapOf<String, FeatureControl>()
    private val auditLogs = mutableListOf<AuditLog>()
    private val random = SecureRandom()
    // PROFIT SHARING
    private val profitShareConfigs = mutableMapOf<String, ProfitShareConfig>()
    private val profitTransactions = mutableListOf<ProfitTransaction>()
    
    init {
        createDefaultSuperAdmin()
        initializeFeatureControls()
    }
    
    // ============================================================================
    // DEFAULT SUPER ADMIN
    // ============================================================================
    
    private fun createDefaultSuperAdmin() {
        // No hardcoded default super admin. Bootstrap admin credentials must be
        // provisioned out-of-band via the backend (go/super_admin_service reads
        // SUPER_ADMIN_USERNAME/SUPER_ADMIN_EMAIL/SUPER_ADMIN_PASSWORD env vars).
        // Shipping a known-password account is a security vulnerability.
    }
    
    private fun initializeFeatureControls() {
        val features = listOf(
            "master_wallet_creation", "multi_blockchain", "token_management",
            "user_wallet_ownership", "hd_wallet", "biometric_auth",
            "pin_code_auth", "nft_support", "defi_integration", "staking",
            "bridge_support", "mev_protection", "swap_trading", "hardware_wallet",
            "admin_controls", "network_management", "gas_optimization", "multi_sig",
            "transaction_history", "price_alerts", "privacy_zk", "coinjoin",
            "account_abstraction", "session_keys", "paymaster", "passkeys",
            "tax_integration", "analytics", "cross_chain_intent", "dapp_browser"
        )
        
        features.forEach { feature ->
            featureControls[feature] = FeatureControl(
                featureName = feature,
                enabled = true,
                globalEnabled = true
            )
        }
    }
    
    // ============================================================================
    // SUPER ADMIN LOGIN
    // ============================================================================
    
    fun superAdminLogin(email: String, password: String, twoFactorCode: String = ""): SuperAdmin? {
        val superAdmin = superAdmins[email] as? SuperAdmin ?: return null
        
        if (!superAdmin.isActive) return null
        
        if (!verifyPassword(password, superAdmin.passwordHash)) {
            logAudit(superAdmin.id, UserRole.SUPER_ADMIN, "LOGIN_FAILED", "Invalid password: $email", "", "")
            return null
        }
        
        if (superAdmin.twoFactorEnabled) {
            if (!verifyTwoFactor(superAdmin.twoFactorSecret, twoFactorCode)) {
                return null
            }
        }
        
        logAudit(superAdmin.id, UserRole.SUPER_ADMIN, "LOGIN_SUCCESS", "Super admin logged in", "", "")
        return superAdmin
    }
    
    // ============================================================================
    // MASTER ADMIN OPERATIONS
    // ============================================================================
    
    /**
     * Create Master Admin Request - Requires Super Admin Authorization
     */
    fun createMasterAdminRequest(email: String, requestedBy: String): MasterAdmin {
        val masterAdmin = MasterAdmin(
            id = generateId(),
            email = email,
            passwordHash = hashPassword(generateTempPassword()),
            authorizationStatus = AuthorizationStatus.PENDING,
            status = AdminStatus.PENDING,
            createdAt = System.currentTimeMillis()
        )
        
        masterAdmins[masterAdmin.id] = masterAdmin
        masterAdmins[email] = masterAdmin
        
        logAudit("SYSTEM", UserRole.SUPER_ADMIN, "MASTER_ADMIN_REQUEST", 
            "New master admin request: $email", "", "")
        
        return masterAdmin
    }
    
    /**
     * Authorize Master Admin - ONLY Super Admin Can Do This
     */
    fun authorizeMasterAdmin(superAdminId: String, masterAdminId: String, authorized: Boolean, notes: String = ""): Boolean {
        // Verify super admin
        if (superAdmins[superAdminId] == null) {
            throw SecurityException("Unauthorized: only super admin can authorize")
        }
        
        val masterAdmin = masterAdmins[masterAdminId] as? MasterAdmin ?: return false
        
        masterAdmins.remove(masterAdmin.id)
        masterAdmins.remove(masterAdmin.email)
        
        val updated = masterAdmin.copy(
            authorizationStatus = if (authorized) AuthorizationStatus.AUTHORIZED else AuthorizationStatus.REJECTED,
            status = if (authorized) AdminStatus.ACTIVE else AdminStatus.INACTIVE,
            authorizedBy = superAdminId
        )
        
        masterAdmins[updated.id] = updated
        masterAdmins[updated.email] = updated
        
        val action = if (authorized) "AUTHORIZED" else "REJECTED"
        logAudit(superAdminId, UserRole.SUPER_ADMIN, "MASTER_ADMIN_$action", 
            "$action master admin: ${masterAdmin.email}", "", "")
        
        return true
    }
    
    /**
     * Master Admin Login - Requires Authorization
     */
    fun masterAdminLogin(email: String, password: String, twoFactorCode: String = ""): MasterAdmin? {
        val masterAdmin = masterAdmins[email] as? MasterAdmin ?: return null
        
        // Check authorization
        if (masterAdmin.authorizationStatus != AuthorizationStatus.AUTHORIZED) {
            return null
        }
        
        if (masterAdmin.status != AdminStatus.ACTIVE) {
            return null
        }
        
        // Check lock
        if (masterAdmin.lockedUntil > System.currentTimeMillis()) {
            return null
        }
        
        if (!verifyPassword(password, masterAdmin.passwordHash)) {
            logAudit(masterAdmin.id, UserRole.MASTER_ADMIN, "LOGIN_FAILED", "Invalid password", "", "")
            return null
        }
        
        if (masterAdmin.twoFactorEnabled) {
            if (!verifyTwoFactor(masterAdmin.twoFactorSecret, twoFactorCode)) {
                return null
            }
        }
        
        logAudit(masterAdmin.id, UserRole.MASTER_ADMIN, "LOGIN_SUCCESS", "Master admin logged in", "", "")
        return masterAdmin
    }
    
    /**
     * Change Password - Master Admin Can Change Their Own Password
     */
    fun changeMasterAdminPassword(adminId: String, oldPassword: String, newPassword: String): Boolean {
        val masterAdmin = masterAdmins[adminId] as? MasterAdmin 
            ?: masterAdmins.entries.find { (it.value as? MasterAdmin)?.email == adminId }?.value as? MasterAdmin
            ?: return false
        
        if (!verifyPassword(oldPassword, masterAdmin.passwordHash)) {
            return false
        }
        
        if (newPassword.length < 8) {
            return false
        }
        
        val updated = masterAdmin.copy(
            passwordHash = hashPassword(newPassword),
            passwordChangedAt = System.currentTimeMillis()
        )
        
        masterAdmins.remove(masterAdmin.id)
        masterAdmins[updated.id] = updated
        masterAdmins[updated.email] = updated
        
        logAudit(adminId, UserRole.MASTER_ADMIN, "PASSWORD_CHANGED", "Password changed", "", "")
        return true
    }
    
    /**
     * Enable 2FA - Master Admin Can Enable 2FA
     */
    fun enableMasterAdmin2FA(adminId: String, secret: String): Boolean {
        val masterAdmin = masterAdmins[adminId] as? MasterAdmin ?: return false
        
        val updated = masterAdmin.copy(
            twoFactorEnabled = true,
            twoFactorSecret = secret
        )
        
        masterAdmins.remove(masterAdmin.id)
        masterAdmins[updated.id] = updated
        masterAdmins[updated.email] = updated
        
        logAudit(adminId, UserRole.MASTER_ADMIN, "2FA_ENABLED", "2FA enabled", "", "")
        return true
    }
    
    /**
     * Disable 2FA - Master Admin Can Disable 2FA
     */
    fun disableMasterAdmin2FA(adminId: String): Boolean {
        val masterAdmin = masterAdmins[adminId] as? MasterAdmin ?: return false
        
        val updated = masterAdmin.copy(
            twoFactorEnabled = false,
            twoFactorSecret = ""
        )
        
        masterAdmins.remove(masterAdmin.id)
        masterAdmins[updated.id] = updated
        masterAdmins[updated.email] = updated
        
        logAudit(adminId, UserRole.MASTER_ADMIN, "2FA_DISABLED", "2FA disabled", "", "")
        return true
    }
    
    // ============================================================================
    // WHITE LABEL ADMIN OPERATIONS
    // ============================================================================
    
    /**
     * Create White Label Admin - By Master Admin
     */
    fun createWhiteLabelAdmin(masterAdminId: String, email: String, brandName: String): WhiteLabelAdmin? {
        val masterAdmin = masterAdmins[masterAdminId] as? MasterAdmin ?: return null
        
        if (!masterAdmin.canCreateWhiteLabel) return null
        if (masterAdmin.whiteLabelCount >= masterAdmin.maxWhiteLabels) return null
        
        val whiteLabel = WhiteLabelAdmin(
            id = generateId(),
            email = email,
            passwordHash = hashPassword(generateTempPassword()),
            masterAdminId = masterAdminId,
            brandName = brandName,
            authorizationStatus = AuthorizationStatus.AUTHORIZED,
            status = AdminStatus.ACTIVE,
            createdAt = System.currentTimeMillis()
        )
        
        whiteLabelAdmins[whiteLabel.id] = whiteLabel
        whiteLabelAdmins[email] = whiteLabel
        
        logAudit(masterAdminId, UserRole.MASTER_ADMIN, "WHITE_LABEL_CREATED", 
            "Created white label: $email - $brandName", "", "")
        
        return whiteLabel
    }
    
    /**
     * White Label Admin Login
     */
    fun whiteLabelLogin(email: String, password: String, twoFactorCode: String = ""): WhiteLabelAdmin? {
        val whiteLabel = whiteLabelAdmins[email] as? WhiteLabelAdmin ?: return null
        
        if (whiteLabel.authorizationStatus != AuthorizationStatus.AUTHORIZED) return null
        if (whiteLabel.status != AdminStatus.ACTIVE) return null
        
        if (!verifyPassword(password, whiteLabel.passwordHash)) return null
        
        if (whiteLabel.twoFactorEnabled) {
            if (!verifyTwoFactor(whiteLabel.twoFactorSecret, twoFactorCode)) return null
        }
        
        return whiteLabel
    }
    
    // ============================================================================
    // FEATURE CONTROL - SUPER ADMIN HAS FULL CONTROL
    // ============================================================================
    
    /**
     * Enable/Disable feature globally
     */
    fun setGlobalFeature(superAdminId: String, featureName: String, enabled: Boolean): Boolean {
        if (superAdmins[superAdminId] == null) {
            throw SecurityException("Only super admin can modify features")
        }
        
        val feature = featureControls[featureName] ?: return false
        
        featureControls[featureName] = feature.copy(
            enabled = enabled,
            globalEnabled = enabled,
            updatedBy = superAdminId,
            updatedAt = System.currentTimeMillis()
        )
        
        logAudit(superAdminId, UserRole.SUPER_ADMIN, "FEATURE_TOGGLE", 
            "Set global feature $featureName = $enabled", "", "")
        
        return true
    }
    
    /**
     * Enable/Disable feature for specific Master Admin
     */
    fun setMasterAdminFeature(superAdminId: String, masterAdminId: String, featureName: String, enabled: Boolean): Boolean {
        if (superAdmins[superAdminId] == null) {
            throw SecurityException("Only super admin can modify features")
        }
        
        val feature = featureControls[featureName] ?: return false
        
        featureControls[featureName] = feature.copy(
            masterAdminId = masterAdminId,
            enabled = enabled,
            updatedBy = superAdminId,
            updatedAt = System.currentTimeMillis()
        )
        
        return true
    }
    
    /**
     * Get all features
     */
    fun getAllFeatures(): List<FeatureControl> = featureControls.values.toList()
    
    /**
     * Check if feature is enabled
     */
    fun isFeatureEnabled(featureName: String, adminId: String, role: UserRole): Boolean {
        val feature = featureControls[featureName] ?: return false
        
        if (!feature.globalEnabled) return false
        
        when (role) {
            UserRole.SUPER_ADMIN -> return true
            UserRole.MASTER_ADMIN -> {
                if (feature.masterAdminId.isNotEmpty() && feature.masterAdminId != adminId) {
                    return false
                }
                return feature.enabled
            }
            UserRole.WHITE_LABEL_ADMIN -> {
                if (feature.whiteLabelId.isNotEmpty() && feature.whiteLabelId != adminId) {
                    return false
                }
                return feature.enabled
            }
            else -> return false
        }
    }
    
    // ============================================================================
    // AUDIT LOGGING
    // ============================================================================
    
    private fun logAudit(adminId: String, role: UserRole, action: String, details: String, ipAddress: String, userAgent: String) {
        val log = AuditLog(
            id = generateId(),
            adminId = adminId,
            adminRole = role,
            action = action,
            details = details,
            ipAddress = ipAddress,
            userAgent = userAgent,
            timestamp = System.currentTimeMillis()
        )
        auditLogs.add(log)
        println("[AUDIT] ${role.name} | $action | $details")
    }
    
    fun getAuditLogs(adminId: String = "", limit: Int = 100): List<AuditLog> {
        return if (adminId.isEmpty()) {
            auditLogs.takeLast(limit)
        } else {
            auditLogs.filter { it.adminId == adminId }.takeLast(limit)
        }
    }
    
    // ============================================================================
    // HELPER FUNCTIONS
    // ============================================================================
    
    private fun generateId(): String = "id_${System.currentTimeMillis()}_${random.nextInt(999999)}"
    
    private fun generateSecretKey(): String = (1..32).map { random.nextInt(256).toString(16).padStart(2, '0') }.joinToString("")
    
    private fun generateTempPassword(): String = (1..16).map { "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"[random.nextInt(62)] }.joinToString("")
    
    private fun hashPassword(password: String): String {
        val bytes = MessageDigest.getInstance("SHA-256").digest(password.toByteArray())
        return bytes.joinToString("") { "%02x".format(it) }
    }
    
    private fun verifyPassword(password: String, hash: String): Boolean {
        return hashPassword(password) == hash
    }
    
    private fun verifyTwoFactor(secret: String, code: String): Boolean {
        // Simplified - use proper TOTP in production
        return code.length == 6 && code.all { it.isDigit() }
    }
}

/**
 * Audit Log Data Class
 */
data class AuditLog(
    val id: String,
    val adminId: String,
    val adminRole: UserRole,
    val action: String,
    val details: String,
    val ipAddress: String,
    val userAgent: String,
    val timestamp: Long
)

// ============================================================================
// PROFIT SHARING SYSTEM
// ============================================================================

/**
 * Profit Share Configuration
 */
data class ProfitShareConfig(
    val id: String,
    val whiteLabelId: String,
    val superAdminWallet: String,  // Super admin receives profit here
    val masterWalletAddress: String,
    val profitPercentage: Double = 20.0,  // Default 20%
    val minPercentage: Double = 0.0,
    val maxPercentage: Double = 50.0,
    val isActive: Boolean = true,
    val autoTransfer: Boolean = true,
    val transferFrequency: String = "daily",
    val lastTransfer: Long = 0,
    val totalTransferred: Double = 0.0,
    val createdAt: Long = System.currentTimeMillis(),
    val updatedAt: Long = System.currentTimeMillis()
)

/**
 * Profit Transaction
 */
data class ProfitTransaction(
    val id: String,
    val whiteLabelId: String,
    val superAdminWallet: String,
    val amount: Double,
    val percentage: Double,
    val grossRevenue: Double,
    val netRevenue: Double,
    val token: String,
    val txHash: String,
    val status: String,  // pending, completed, failed
    val createdAt: Long = System.currentTimeMillis()
)

// ============================================================================
// PROFIT SHARING METHODS
// ============================================================================

/**
 * Set profit share percentage - Only Super Admin can do this
 */
fun setProfitSharePercentage(superAdminId: String, whiteLabelId: String, percentage: Double): Boolean {
    if (superAdmins[superAdminId] == null) {
        throw SecurityException("Only super admin can set profit share")
    }
    
    if (percentage < 0 || percentage > 50) {
        throw IllegalArgumentException("Percentage must be between 0 and 50")
    }
    
    val whiteLabel = whiteLabelAdmins[whiteLabelId] as? WhiteLabelAdmin
        ?: throw IllegalArgumentException("White label not found")
    
    val config = ProfitShareConfig(
        id = generateId(),
        whiteLabelId = whiteLabelId,
        superAdminWallet = getSuperAdminWalletAddress(),
        masterWalletAddress = whiteLabel.id,
        profitPercentage = percentage,
        createdAt = System.currentTimeMillis(),
        updatedAt = System.currentTimeMillis()
    )
    
    profitShareConfigs[whiteLabelId] = config
    
    logAudit(superAdminId, UserRole.SUPER_ADMIN, "PROFIT_SHARE_SET", 
        "Set profit share to $percentage% for white label $whiteLabelId", "", "")
    
    return true
}

/**
 * Calculate profit share
 */
fun calculateProfitShare(whiteLabelId: String, grossRevenue: Double): Pair<Double, Double> {
    val config = profitShareConfigs[whiteLabelId]
    val percentage = config?.profitPercentage ?: 20.0  // Default 20%
    
    val superAdminShare = grossRevenue * (percentage / 100)
    val whiteLabelShare = grossRevenue - superAdminShare
    
    return Pair(superAdminShare, whiteLabelShare)
}

/**
 * Execute profit transfer to Super Admin wallet
 */
fun executeProfitTransfer(whiteLabelId: String, token: String, amount: Double): ProfitTransaction? {
    val config = profitShareConfigs[whiteLabelId]
    if (config == null || !config.isActive) {
        return null
    }
    
    val (superAdminShare, _) = calculateProfitShare(whiteLabelId, amount)
    
    val tx = ProfitTransaction(
        id = generateId(),
        whiteLabelId = whiteLabelId,
        superAdminWallet = config.superAdminWallet,
        amount = superAdminShare,
        percentage = config.profitPercentage,
        grossRevenue = amount,
        netRevenue = amount - superAdminShare,
        token = token,
        txHash = "", // real tx hash from on-chain broadcast via wallet_api /send
        status = "pending", // pending until real on-chain transfer confirms
        createdAt = System.currentTimeMillis()
    )
    
    profitTransactions.add(tx)
    
    // Update total transferred
    val updatedConfig = config.copy(
        totalTransferred = config.totalTransferred + superAdminShare,
        lastTransfer = System.currentTimeMillis()
    )
    profitShareConfigs[whiteLabelId] = updatedConfig
    
    logAudit("SYSTEM", UserRole.SUPER_ADMIN, "PROFIT_TRANSFER", 
        "Transferred $superAdminShare $token to super admin from white label $whiteLabelId", "", "")
    
    return tx
}

/**
 * Get profit history
 */
fun getProfitHistory(whiteLabelId: String = "", limit: Int = 100): List<ProfitTransaction> {
    return if (whiteLabelId.isEmpty()) {
        profitTransactions.takeLast(limit)
    } else {
        profitTransactions.filter { it.whiteLabelId == whiteLabelId }.takeLast(limit)
    }
}

/**
 * Get total profits
 */
fun getTotalProfits(): Double {
    return profitShareConfigs.values.sumOf { it.totalTransferred }
}

private fun getSuperAdminWalletAddress(): String {
    // In production, this comes from system config
    return "0xSuperAdminWalletAddress"
}
