package com.tigerwallet.admin.util

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

class PreferencesManager(context: Context) {
    private val masterKey = MasterKey.Builder(context).setKeyScheme(MasterKey.KeyScheme.AES256_GCM).build()
    private val prefs: SharedPreferences = EncryptedSharedPreferences.create(
        context, "admin_prefs", masterKey, EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV, EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
    )

    companion object {
        private const val KEY_TOKEN = "auth_token"
        private const val KEY_REFRESH_TOKEN = "refresh_token"
        private const val KEY_ADMIN_ID = "admin_id"
        private const val KEY_ADMIN_EMAIL = "admin_email"
        private const val KEY_ADMIN_ROLE = "admin_role"
        private const val KEY_THEME = "theme_mode"
        private const val KEY_LANGUAGE = "language"
        private const val KEY_NOTIFICATIONS_ENABLED = "notifications_enabled"
    }

    fun saveToken(token: String) { prefs.edit().putString(KEY_TOKEN, token).apply() }
    fun getToken(): String? = prefs.getString(KEY_TOKEN, null)
    fun clearToken() { prefs.edit().remove(KEY_TOKEN).apply() }

    fun saveRefreshToken(token: String) { prefs.edit().putString(KEY_REFRESH_TOKEN, token).apply() }
    fun getRefreshToken(): String? = prefs.getString(KEY_REFRESH_TOKEN, null)

    fun saveAdminId(id: String) { prefs.edit().putString(KEY_ADMIN_ID, id).apply() }
    fun getAdminId(): String? = prefs.getString(KEY_ADMIN_ID, null)

    fun saveAdminEmail(email: String) { prefs.edit().putString(KEY_ADMIN_EMAIL, email).apply() }
    fun getAdminEmail(): String? = prefs.getString(KEY_ADMIN_EMAIL, null)

    fun saveAdminRole(role: String) { prefs.edit().putString(KEY_ADMIN_ROLE, role).apply() }
    fun getAdminRole(): String? = prefs.getString(KEY_ADMIN_ROLE, null)

    fun saveTheme(theme: String) { prefs.edit().putString(KEY_THEME, theme).apply() }
    fun getTheme(): String = prefs.getString(KEY_THEME, "system") ?: "system"

    fun saveLanguage(language: String) { prefs.edit().putString(KEY_LANGUAGE, language).apply() }
    fun getLanguage(): String = prefs.getString(KEY_LANGUAGE, "en") ?: "en"

    fun setNotificationsEnabled(enabled: Boolean) { prefs.edit().putBoolean(KEY_NOTIFICATIONS_ENABLED, enabled).apply() }
    fun isNotificationsEnabled(): Boolean = prefs.getBoolean(KEY_NOTIFICATIONS_ENABLED, true)

    fun clearAll() { prefs.edit().clear().apply() }
    fun isLoggedIn(): Boolean = getToken() != null
}
