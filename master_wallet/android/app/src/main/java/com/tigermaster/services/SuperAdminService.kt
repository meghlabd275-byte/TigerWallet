package com.tigermaster.services

import android.content.Context
import android.util.Base64
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL
import java.security.MessageDigest

/**
 * MasterWallet Super Admin Service (Android)
 * Admin controls for MasterWallet management
 * Production-ready with full functionality
 */
class SuperAdminService(private val context: Context) {
    
    companion object {
        private const val BASE_URL = "https://api.tigerwallet.com"
        private const val PREFS_NAME = "super_admin_prefs"
    }
    
    private val masterKey: MasterKey by lazy {
        MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
    }
    
    private val encryptedPrefs by lazy {
        EncryptedSharedPreferences.create(
            context,
            PREFS_NAME,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    }
    
    private var adminId: String? = null
    private var role: String? = null
    private var isAuthenticated: Boolean = false
    
    /**
     * Initialize the service
     */
    fun initialize(): Boolean {
        try {
            loadSession()
            loadFeatureFlags()
            return true
        } catch (e: Exception) {
            e.printStackTrace()
            return false
        }
    }
    
    /**
     * Authenticate admin user
     */
    suspend fun authenticate(email: String, password: String): Boolean = withContext(Dispatchers.IO) {
        try {
            val result = makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/authenticate",
                body = mapOf("email" to email, "password" to password)
            )
            
            if (result.has("adminId")) {
                adminId = result.getString("adminId")
                role = result.getString("role")
                isAuthenticated = true
                
                // Save session
                saveSession()
                
                // Log audit
                logAudit("LOGIN", "session", null, true)
                
                true
            } else {
                logAudit("LOGIN_FAILED", "session", null, false)
                false
            }
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Logout
     */
    suspend fun logout(): Boolean {
        logAudit("LOGOUT", "session", adminId, true)
        
        adminId = null
        role = null
        isAuthenticated = false
        clearSession()
        
        return true
    }
    
    /**
     * Change password
     */
    suspend fun changePassword(oldPassword: String, newPassword: String): Boolean {
        if (!isAuthenticated) return false
        
        return try {
            val result = makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/change-password",
                body = mapOf(
                    "adminId" to adminId!!,
                    "oldPassword" to oldPassword,
                    "newPassword" to newPassword
                )
            )
            
            logAudit("PASSWORD_CHANGED", "admin", adminId, result.has("success"))
            result.has("success")
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Enable 2FA
     */
    suspend fun enable2FA(): String? {
        if (!isAuthenticated) return null
        
        return try {
            val result = makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/enable-2fa",
                body = mapOf("adminId" to adminId!!)
            )
            
            if (result.has("secret")) {
                result.getString("secret")
            } else null
        } catch (e: Exception) {
            e.printStackTrace()
            null
        }
    }
    
    /**
     * Verify 2FA code
     */
    suspend fun verify2FA(code: String): Boolean {
        if (!isAuthenticated) return false
        
        return try {
            val result = makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/verify-2fa",
                body = mapOf("adminId" to adminId!!, "code" to code)
            )
            
            logAudit("2FA_ENABLED", "admin", adminId, result.has("success"))
            result.has("success")
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Disable 2FA
     */
    suspend fun disable2FA(code: String): Boolean {
        if (!isAuthenticated) return false
        
        return try {
            val result = makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/disable-2fa",
                body = mapOf("adminId" to adminId!!, "code" to code)
            )
            
            logAudit("2FA_DISABLED", "admin", adminId, result.has("success"))
            result.has("success")
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Set feature flag
     */
    suspend fun setFeatureFlag(name: String, enabled: Boolean): Boolean {
        if (!isAuthenticated || role != "SUPER_ADMIN") return false
        
        return try {
            // Update locally
            val flags = getFeatureFlags().toMutableMap()
            flags[name] = mapOf("enabled" to enabled, "updatedAt" to System.currentTimeMillis())
            saveFeatureFlags(flags)
            
            // Send to backend
            makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/feature-flag",
                body = mapOf("name" to name, "enabled" to enabled)
            )
            
            logAudit("FEATURE_UPDATED", "feature", name, true)
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Get feature flag
     */
    fun getFeatureFlag(name: String): Map<String, Any>? {
        return getFeatureFlags()[name] as? Map<String, Any>
    }
    
    /**
     * List all feature flags
     */
    fun listFeatureFlags(): Map<String, Any> {
        return getFeatureFlags()
    }
    
    /**
     * Check if feature is enabled
     */
    fun isFeatureEnabled(name: String): Boolean {
        val flag = getFeatureFlag(name)
        return (flag?.get("enabled") as? Boolean) ?: false
    }
    
    /**
     * Create admin user
     */
    suspend fun createAdmin(
        email: String,
        name: String,
        role: String,
        password: String
    ): String? {
        if (!isAuthenticated || role != "SUPER_ADMIN") return null
        
        return try {
            val result = makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/create-admin",
                body = mapOf(
                    "email" to email,
                    "name" to name,
                    "role" to role,
                    "password" to password
                )
            )
            
            if (result.has("adminId")) {
                logAudit("ADMIN_CREATED", "admin", result.getString("adminId"), true)
                result.getString("adminId")
            } else null
        } catch (e: Exception) {
            e.printStackTrace()
            null
        }
    }
    
    /**
     * List admins
     */
    suspend fun listAdmins(roleFilter: String? = null): List<Map<String, Any>> {
        if (!isAuthenticated) return emptyList()
        
        return try {
            val endpoint = if (roleFilter != null) {
                "/api/super-admin/admins?role=$roleFilter"
            } else {
                "/api/super-admin/admins"
            }
            
            val result = makeRequest(method = "GET", endpoint = endpoint)
            
            if (result.has("admins")) {
                val adminsArray = result.getJSONArray("admins")
                (0 until adminsArray.length()).map { i ->
                    adminsArray.getJSONObject(i).toMap()
                }
            } else emptyList()
        } catch (e: Exception) {
            e.printStackTrace()
            emptyList()
        }
    }
    
    /**
     * Deactivate admin
     */
    suspend fun deactivateAdmin(adminId: String): Boolean {
        if (!isAuthenticated || role != "SUPER_ADMIN") return false
        
        return try {
            makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/admin/$adminId/deactivate"
            )
            
            logAudit("ADMIN_DEACTIVATED", "admin", adminId, true)
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Authorize master admin
     */
    suspend fun authorizeMasterAdmin(masterWalletId: String): Boolean {
        if (!isAuthenticated || role != "SUPER_ADMIN") return false
        
        return try {
            makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/authorize-master",
                body = mapOf("masterWalletId" to masterWalletId)
            )
            
            logAudit("MASTER_AUTHORIZED", "master_wallet", masterWalletId, true)
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
    
    /**
     * Get audit logs
     */
    suspend fun getAuditLogs(
        adminId: String? = null,
        action: String? = null,
        startTime: Long? = null,
        endTime: Long? = null,
        limit: Int = 100
    ): List<Map<String, Any>> {
        if (!isAuthenticated) return emptyList()
        
        return try {
            val params = mutableListOf<String>()
            adminId?.let { params.add("adminId=$it") }
            action?.let { params.add("action=$it") }
            startTime?.let { params.add("startTime=$it") }
            endTime?.let { params.add("endTime=$it") }
            params.add("limit=$limit")
            
            val endpoint = "/api/super-admin/audit?" + params.joinToString("&")
            val result = makeRequest(method = "GET", endpoint = endpoint)
            
            if (result.has("logs")) {
                val logsArray = result.getJSONArray("logs")
                (0 until logsArray.length()).map { i ->
                    logsArray.getJSONObject(i).toMap()
                }
            } else emptyList()
        } catch (e: Exception) {
            e.printStackTrace()
            emptyList()
        }
    }
    
    /**
     * Get statistics
     */
    suspend fun getStats(): Map<String, Any> {
        return try {
            makeRequest(method = "GET", endpoint = "/api/super-admin/stats")
                .toMap()
        } catch (e: Exception) {
            e.printStackTrace()
            emptyMap()
        }
    }
    
    /**
     * Check if user is authenticated
     */
    fun isAuthenticated(): Boolean = isAuthenticated
    
    /**
     * Get current admin ID
     */
    fun getAdminId(): String? = adminId
    
    /**
     * Get current role
     */
    fun getRole(): String? = role
    
    /**
     * Check if user is super admin
     */
    fun isSuperAdmin(): Boolean = role == "SUPER_ADMIN"
    
    // Private helper methods
    
    private fun loadSession() {
        adminId = encryptedPrefs.getString("adminId", null)
        role = encryptedPrefs.getString("role", null)
        isAuthenticated = encryptedPrefs.getBoolean("isAuthenticated", false)
    }
    
    private fun saveSession() {
        encryptedPrefs.edit()
            .putString("adminId", adminId)
            .putString("role", role)
            .putBoolean("isAuthenticated", isAuthenticated)
            .apply()
    }
    
    private fun clearSession() {
        encryptedPrefs.edit()
            .remove("adminId")
            .remove("role")
            .putBoolean("isAuthenticated", false)
            .apply()
    }
    
    private fun getFeatureFlags(): Map<String, Any> {
        val stored = encryptedPrefs.getString("featureFlags", null)
        return if (stored != null) {
            try {
                JSONObject(stored).toMap()
            } catch (e: Exception) {
                getDefaultFeatureFlags()
            }
        } else {
            getDefaultFeatureFlags().also { saveFeatureFlags(it) }
        }
    }
    
    private fun getDefaultFeatureFlags(): Map<String, Any> {
        return mapOf(
            "master_wallet_creation" to mapOf("enabled" to true, "updatedAt" to 0L),
            "multi_blockchain" to mapOf("enabled" to true, "updatedAt" to 0L),
            "token_management" to mapOf("enabled" to true, "updatedAt" to 0L),
            "user_wallet_ownership" to mapOf("enabled" to true, "updatedAt" to 0L),
            "hd_wallet" to mapOf("enabled" to true, "updatedAt" to 0L),
            "biometric_auth" to mapOf("enabled" to true, "updatedAt" to 0L),
            "pin_code_auth" to mapOf("enabled" to true, "updatedAt" to 0L),
            "nft_support" to mapOf("enabled" to true, "updatedAt" to 0L),
            "defi_integration" to mapOf("enabled" to true, "updatedAt" to 0L),
            "staking" to mapOf("enabled" to true, "updatedAt" to 0L),
            "bridge_support" to mapOf("enabled" to true, "updatedAt" to 0L),
            "mev_protection" to mapOf("enabled" to true, "updatedAt" to 0L),
            "swap_trading" to mapOf("enabled" to true, "updatedAt" to 0L),
            "hardware_wallet" to mapOf("enabled" to true, "updatedAt" to 0L),
            "admin_controls" to mapOf("enabled" to true, "updatedAt" to 0L),
            "network_management" to mapOf("enabled" to true, "updatedAt" to 0L),
            "gas_optimization" to mapOf("enabled" to true, "updatedAt" to 0L),
            "multi_sig" to mapOf("enabled" to true, "updatedAt" to 0L),
            "transaction_history" to mapOf("enabled" to true, "updatedAt" to 0L),
            "price_alerts" to mapOf("enabled" to true, "updatedAt" to 0L),
            "privacy_zk" to mapOf("enabled" to true, "updatedAt" to 0L),
            "coinjoin" to mapOf("enabled" to true, "updatedAt" to 0L),
            "account_abstraction" to mapOf("enabled" to true, "updatedAt" to 0L),
            "session_keys" to mapOf("enabled" to true, "updatedAt" to 0L),
            "paymaster" to mapOf("enabled" to true, "updatedAt" to 0L),
            "passkeys" to mapOf("enabled" to true, "updatedAt" to 0L),
            "tax_integration" to mapOf("enabled" to true, "updatedAt" to 0L),
            "analytics" to mapOf("enabled" to true, "updatedAt" to 0L),
            "cross_chain_intent" to mapOf("enabled" to true, "updatedAt" to 0L),
            "dapp_browser" to mapOf("enabled" to true, "updatedAt" to 0L)
        )
    }
    
    private fun saveFeatureFlags(flags: Map<String, Any>) {
        val json = JSONObject(flags).toString()
        encryptedPrefs.edit().putString("featureFlags", json).apply()
    }
    
    private fun loadFeatureFlags() {
        getFeatureFlags() // This will initialize defaults if needed
    }
    
    private suspend fun logAudit(
        action: String,
        resourceType: String,
        resourceId: String?,
        success: Boolean
    ) {
        try {
            makeRequest(
                method = "POST",
                endpoint = "/api/super-admin/audit",
                body = mapOf(
                    "adminId" to (adminId ?: ""),
                    "action" to action,
                    "resourceType" to resourceType,
                    "resourceId" to (resourceId ?: ""),
                    "success" to success,
                    "timestamp" to System.currentTimeMillis()
                )
            )
        } catch (e: Exception) {
            // Silent fail for audit logging
        }
    }
    
    private suspend fun makeRequest(
        method: String,
        endpoint: String,
        body: Map<String, Any>? = null
    ): JSONObject {
        return withContext(Dispatchers.IO) {
            val url = URL("$BASE_URL$endpoint")
            val connection = url.openConnection() as HttpURLConnection
            
            connection.requestMethod = method
            connection.setRequestProperty("Content-Type", "application/json")
            connection.doOutput = true
            
            if (isAuthenticated && adminId != null) {
                connection.setRequestProperty("X-Admin-ID", adminId)
            }
            
            body?.let {
                val bodyBytes = JSONObject(it).toString().toByteArray()
                connection.outputStream.write(bodyBytes)
            }
            
            val responseCode = connection.responseCode
            val responseBody = if (responseCode in 200..299) {
                connection.inputStream.bufferedReader().readText()
            } else {
                connection.errorStream?.bufferedReader()?.readText() ?: ""
            }
            
            if (responseBody.isNotEmpty()) {
                JSONObject(responseBody)
            } else {
                JSONObject()
            }
        }
    }
    
    private fun JSONObject.toMap(): Map<String, Any> {
        val map = mutableMapOf<String, Any>()
        keys().forEach { key ->
            val value = get(key)
            map[key] = when (value) {
                is JSONObject -> value.toMap()
                is JSONArray -> value.toList()
                else -> value
            }
        }
        return map
    }
    
    private fun JSONArray.toList(): List<Any> {
        return (0 until length()).map { i ->
            val value = get(i)
            when (value) {
                is JSONObject -> value.toMap()
                is JSONArray -> value.toList()
                else -> value
            }
        }
    }
}
