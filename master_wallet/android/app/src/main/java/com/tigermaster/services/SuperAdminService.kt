package com.tigermaster.services

import android.content.Context

/**
 * MasterWallet Super Admin Service (Android)
 *
 * Super-admin / white-label / feature-toggle administration is an Admin-app
 * feature and is NOT part of the canonical MasterWallet backend contract
 * (port 8450). The previous implementation POSTed to fabricated
 * `/api/super-admin/*` routes that do not exist on the canonical backend.
 *
 * To preserve the MasterWallet/Admin app isolation guarantee, this service is
 * FAIL-CLOSED: every operation throws a descriptive error instead of hitting
 * non-canonical endpoints or returning fabricated data. This mirrors the web
 * client's `SuperAdminService` (which returns descriptive "not supported"
 * errors). The class and its method signatures are kept for parity; callers
 * must handle the thrown [UnsupportedOperationException].
 */
class SuperAdminService(private val context: Context) {

    private fun unsupported(): Nothing {
        throw UnsupportedOperationException(
            "SuperAdmin is an Admin-app feature, not available in MasterWallet"
        )
    }

    /** Initialize the service. */
    fun initialize(): Boolean = unsupported()

    /** Authenticate admin user. */
    suspend fun authenticate(email: String, password: String): Boolean = unsupported()

    /** Logout. */
    suspend fun logout(): Boolean = unsupported()

    /** Change password. */
    suspend fun changePassword(oldPassword: String, newPassword: String): Boolean = unsupported()

    /** Enable 2FA. */
    suspend fun enable2FA(): String? = unsupported()

    /** Verify 2FA. */
    suspend fun verify2FA(code: String): Boolean = unsupported()

    /** Disable 2FA. */
    suspend fun disable2FA(code: String): Boolean = unsupported()

    /** Set a feature flag. */
    suspend fun setFeatureFlag(name: String, enabled: Boolean): Boolean = unsupported()

    /** Get a feature flag. */
    fun getFeatureFlag(name: String): Map<String, Any>? = unsupported()

    /** List feature flags. */
    fun listFeatureFlags(): Map<String, Any> = unsupported()

    /** Check whether a feature is enabled. */
    fun isFeatureEnabled(name: String): Boolean = unsupported()

    /** Create an admin. */
    suspend fun createAdmin(
        email: String,
        name: String,
        role: String,
        password: String
    ): String? = unsupported()

    /** List admins. */
    suspend fun listAdmins(roleFilter: String? = null): List<Map<String, Any>> = unsupported()

    /** Deactivate an admin. */
    suspend fun deactivateAdmin(adminId: String): Boolean = unsupported()

    /** Authorize master admin. */
    suspend fun authorizeMasterAdmin(masterWalletId: String): Boolean = unsupported()

    /** Get audit logs. */
    @Suppress("UNUSED_PARAMETER")
    suspend fun getAuditLogs(
        page: Int = 1,
        pageSize: Int = 50,
        actionFilter: String? = null,
        adminIdFilter: String? = null,
        startDate: String? = null,
        endDate: String? = null
    ): List<Map<String, Any>> = unsupported()

    /** Get aggregate stats. */
    suspend fun getStats(): Map<String, Any> = unsupported()

    /** Check if user is authenticated. */
    fun isAuthenticated(): Boolean = unsupported()

    /** Get current admin ID. */
    fun getAdminId(): String? = unsupported()

    /** Get current role. */
    fun getRole(): String? = unsupported()

    /** Check if user is super admin. */
    fun isSuperAdmin(): Boolean = unsupported()
}
