package com.tigerwallet.admin

import android.content.Context
import android.content.SharedPreferences
import com.google.gson.Gson
import com.tigerwallet.admin.data.model.AdminUser

/**
 * Session Manager
 * Manages admin authentication and session data
 */
class SessionManager(context: Context) {

    private val prefs: SharedPreferences = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    private val gson = Gson()

    companion object {
        private const val PREFS_NAME = "tigeradmin_session"
        private const val KEY_AUTH_TOKEN = "auth_token"
        private const val KEY_REFRESH_TOKEN = "refresh_token"
        private const val KEY_EXPIRES_AT = "expires_at"
        private const val KEY_ADMIN_USER = "admin_user"
        private const val KEY_IS_LOGGED_IN = "is_logged_in"
        private const val KEY_2FA_ENABLED = "2fa_enabled"
        private const val KEY_LAST_ACTIVITY = "last_activity"
        
        @Volatile
        private var instance: SessionManager? = null

        fun getInstance(context: Context): SessionManager? {
            return instance ?: synchronized(this) {
                instance ?: SessionManager(context.applicationContext).also { instance = it }
            }
        }
    }

    /**
     * Save authentication session
     */
    fun saveSession(
        authToken: String,
        refreshToken: String,
        expiresAt: String,
        adminUser: AdminUser
    ) {
        prefs.edit().apply {
            putString(KEY_AUTH_TOKEN, authToken)
            putString(KEY_REFRESH_TOKEN, refreshToken)
            putString(KEY_EXPIRES_AT, expiresAt)
            putString(KEY_ADMIN_USER, gson.toJson(adminUser))
            putBoolean(KEY_IS_LOGGED_IN, true)
            putLong(KEY_LAST_ACTIVITY, System.currentTimeMillis())
            apply()
        }
    }

    /**
     * Get auth token
     */
    fun getAuthToken(): String? = prefs.getString(KEY_AUTH_TOKEN, null)

    /**
     * Get refresh token
     */
    fun getRefreshToken(): String? = prefs.getString(KEY_REFRESH_TOKEN, null)

    /**
     * Get expires at time
     */
    fun getExpiresAt(): String? = prefs.getString(KEY_EXPIRES_AT, null)

    /**
     * Check if session is valid
     */
    fun isSessionValid(): Boolean {
        val expiresAt = getExpiresAt() ?: return false
        // Check if token is expired
        return try {
            // Simple check - in production, parse the ISO date and compare
            !isTokenExpired(expiresAt)
        } catch (e: Exception) {
            false
        }
    }

    /**
     * Check if token is expired
     */
    private fun isTokenExpired(expiresAt: String): Boolean {
        return try {
            val parsed = java.time.Instant.parse(expiresAt)
            parsed.isBefore(java.time.Instant.now())
        } catch (e: Exception) {
            // Unparseable timestamp is treated as expired (fail-closed).
            true
        }
    }

    /**
     * Get current admin user
     */
    fun getCurrentAdmin(): AdminUser? {
        val json = prefs.getString(KEY_ADMIN_USER, null) ?: return null
        return try {
            gson.fromJson(json, AdminUser::class.java)
        } catch (e: Exception) {
            null
        }
    }

    /**
     * Check if user is logged in
     */
    fun isLoggedIn(): Boolean {
        val isLoggedIn = prefs.getBoolean(KEY_IS_LOGGED_IN, false)
        return isLoggedIn && isSessionValid()
    }

    /**
     * Update admin user
     */
    fun updateAdminUser(adminUser: AdminUser) {
        prefs.edit().apply {
            putString(KEY_ADMIN_USER, gson.toJson(adminUser))
            apply()
        }
    }

    /**
     * Set 2FA enabled
     */
    fun set2FAEnabled(enabled: Boolean) {
        prefs.edit().apply {
            putBoolean(KEY_2FA_ENABLED, enabled)
            apply()
        }
    }

    /**
     * Check if 2FA is enabled
     */
    fun is2FAEnabled(): Boolean = prefs.getBoolean(KEY_2FA_ENABLED, false)

    /**
     * Update last activity timestamp
     */
    fun updateLastActivity() {
        prefs.edit().apply {
            putLong(KEY_LAST_ACTIVITY, System.currentTimeMillis())
            apply()
        }
    }

    /**
     * Get last activity timestamp
     */
    fun getLastActivity(): Long = prefs.getLong(KEY_LAST_ACTIVITY, 0)

    /**
     * Check if session has been inactive for too long
     */
    fun isSessionExpired(inactivityTimeoutMinutes: Int = 30): Boolean {
        val lastActivity = getLastActivity()
        val currentTime = System.currentTimeMillis()
        val timeoutMillis = inactivityTimeoutMinutes * 60 * 1000L
        return (currentTime - lastActivity) > timeoutMillis
    }

    /**
     * Clear session (logout)
     */
    fun logout() {
        prefs.edit().clear().apply()
    }

    /**
     * Clear only auth token (keep user data)
     */
    fun clearAuthToken() {
        prefs.edit().apply {
            remove(KEY_AUTH_TOKEN)
            remove(KEY_REFRESH_TOKEN)
            remove(KEY_EXPIRES_AT)
            putBoolean(KEY_IS_LOGGED_IN, false)
            apply()
        }
    }
}

/**
 * Cache Manager
 * Manages local caching for offline support
 */
class CacheManager(private val context: Context) {

    private val prefs: SharedPreferences = context.getSharedPreferences(CACHE_PREFS_NAME, Context.MODE_PRIVATE)
    private val gson = Gson()

    companion object {
        private const val CACHE_PREFS_NAME = "tigeradmin_cache"
        
        // Cache keys
        const val KEY_USERS_CACHE = "users_cache"
        const val KEY_TRANSACTIONS_CACHE = "transactions_cache"
        const val KEY_TOKENS_CACHE = "tokens_cache"
        const val KEY_KYC_CACHE = "kyc_cache"
        const val KEY_SYSTEM_STATUS_CACHE = "system_status_cache"
        const val KEY_ANALYTICS_CACHE = "analytics_cache"
        
        // Cache expiration times (in milliseconds)
        const val CACHE_SHORT = 5 * 60 * 1000L // 5 minutes
        const val CACHE_MEDIUM = 15 * 60 * 1000L // 15 minutes
        const val CACHE_LONG = 60 * 60 * 1000L // 1 hour
    }

    /**
     * Save data to cache
     */
    fun <T> saveToCache(key: String, data: T, expirationTime: Long = CACHE_MEDIUM) {
        val cacheEntry = CacheEntry(
            data = gson.toJson(data),
            timestamp = System.currentTimeMillis(),
            expirationTime = expirationTime
        )
        prefs.edit().apply {
            putString(key, gson.toJson(cacheEntry))
            apply()
        }
    }

    /**
     * Get data from cache
     */
    fun <T> getFromCache(key: String, clazz: Class<T>): T? {
        val json = prefs.getString(key, null) ?: return null
        return try {
            val cacheEntry = gson.fromJson(json, CacheEntry::class.java)
            if (isCacheValid(cacheEntry)) {
                gson.fromJson(cacheEntry.data, clazz)
            } else {
                removeFromCache(key)
                null
            }
        } catch (e: Exception) {
            removeFromCache(key)
            null
        }
    }

    /**
     * Check if cache is valid
     */
    private fun isCacheValid(entry: CacheEntry): Boolean {
        val currentTime = System.currentTimeMillis()
        return (currentTime - entry.timestamp) < entry.expirationTime
    }

    /**
     * Remove data from cache
     */
    fun removeFromCache(key: String) {
        prefs.edit().remove(key).apply()
    }

    /**
     * Clear all cache
     */
    fun clearAll() {
        prefs.edit().clear().apply()
    }

    /**
     * Clear expired cache entries
     */
    fun clearExpired() {
        val allEntries = prefs.all
        allEntries.forEach { (key, value) ->
            if (value is String) {
                try {
                    val cacheEntry = gson.fromJson(value, CacheEntry::class.java)
                    if (!isCacheValid(cacheEntry)) {
                        removeFromCache(key)
                    }
                } catch (e: Exception) {
                    // Ignore invalid entries
                }
            }
        }
    }

    /**
     * Cache entry data class
     */
    data class CacheEntry(
        val data: String,
        val timestamp: Long,
        val expirationTime: Long
    )
}

/**
 * Notification Service
 * Handles push notifications for admin alerts
 */
class NotificationService(private val context: Context) {

    companion object {
        const val CHANNEL_ID = "tigeradmin_notifications"
        const val CHANNEL_NAME = "Admin Notifications"
        const val CHANNEL_DESCRIPTION = "Notifications for admin actions and alerts"
    }

    // Notification types
    object NotificationType {
        const val NEW_USER = "new_user"
        const val NEW_TRANSACTION = "new_transaction"
        const val KYC_SUBMITTED = "kyc_submitted"
        const val KYC_APPROVED = "kyc_approved"
        const val KYC_REJECTED = "kyc_rejected"
        const val LARGE_TRANSACTION = "large_transaction"
        const val SUSPICIOUS_ACTIVITY = "suspicious_activity"
        const val SYSTEM_ALERT = "system_alert"
        const val WITHDRAWAL_REQUEST = "withdrawal_request"
        const val TOKEN_LISTING = "token_listing"
    }

    /**
     * Create notification channel (for Android O and above)
     */
    fun createNotificationChannel() {
        // Implementation for Android notification channel
        // In a real app, this would use NotificationManager
    }

    /**
     * Show notification
     */
    fun showNotification(
        title: String,
        message: String,
        type: String,
        data: Map<String, String>? = null
    ) {
        // Implementation for showing local notifications
        // In a real app, this would use NotificationManager
    }

    /**
     * Handle notification tap
     */
    fun handleNotificationTap(notificationId: String) {
        // Handle navigation based on notification type
    }

    /**
     * Register device for push notifications
     */
    fun registerForPushNotifications(token: String) {
        // Send token to server for push notifications
    }

    /**
     * Unregister from push notifications
     */
    fun unregisterFromPushNotifications() {
        // Remove token from server
    }
}

/**
 * Analytics Manager
 * Tracks admin user actions for analytics
 */
class AnalyticsManager(private val context: Context) {

    companion object {
        private const val ANALYTICS_PREFS = "tigeradmin_analytics"
    }

    /**
     * Track screen view
     */
    fun trackScreenView(screenName: String) {
        // Track screen view event
    }

    /**
     * Track user action
     */
    fun trackAction(action: String, details: Map<String, Any>? = null) {
        // Track user action event
    }

    /**
     * Track error
     */
    fun trackError(error: String, stackTrace: String? = null) {
        // Track error event
    }

    /**
     * Track performance
     */
    fun trackPerformance(metric: String, value: Long) {
        // Track performance metric
    }

    /**
     * Set user properties
     */
    fun setUserProperties(properties: Map<String, Any>) {
        // Set user properties for analytics
    }

    /**
     * Clear analytics data
     */
    fun clearData() {
        // Clear all analytics data
    }
}
