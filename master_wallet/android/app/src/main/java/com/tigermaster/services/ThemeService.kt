package com.tigermaster.services

import android.content.Context
import android.content.res.Configuration
import androidx.appcompat.app.AppCompatDelegate
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

/**
 * MasterWallet Theme Service (Android)
 * Light/Dark theme switching for all pages
 * Production-ready
 */
class ThemeService(private val context: Context) {
    
    companion object {
        private const val PREFS_NAME = "theme_prefs"
        private const val KEY_THEME = "theme_mode"
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
    
    private var currentTheme: String = "light"
    private var listeners: MutableList<ThemeChangeListener> = mutableListOf()
    
    /**
     * Initialize the theme service
     */
    fun initialize() {
        // Load saved theme
        currentTheme = encryptedPrefs.getString(KEY_THEME, getSystemTheme()) ?: "light"
        
        // Apply theme
        applyTheme()
    }
    
    /**
     * Get current theme
     */
    fun getTheme(): String = currentTheme
    
    /**
     * Check if dark mode
     */
    fun isDarkMode(): Boolean = currentTheme == "dark"
    
    /**
     * Set theme
     */
    fun setTheme(theme: String) {
        if (theme != "light" && theme != "dark") return
        
        currentTheme = theme
        encryptedPrefs.edit().putString(KEY_THEME, theme).apply()
        
        applyTheme()
        
        // Notify listeners
        notifyListeners()
    }
    
    /**
     * Toggle theme
     */
    fun toggleTheme() {
        setTheme(if (currentTheme == "light") "dark" else "light")
    }
    
    /**
     * Apply theme to the app
     */
    private fun applyTheme() {
        val nightMode = if (currentTheme == "dark") {
            AppCompatDelegate.MODE_NIGHT_YES
        } else {
            AppCompatDelegate.MODE_NIGHT_NO
        }
        
        AppCompatDelegate.setDefaultNightMode(nightMode)
    }
    
    /**
     * Get system theme preference
     */
    private fun getSystemTheme(): String {
        val nightModeFlags = context.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK
        return when (nightModeFlags) {
            Configuration.UI_MODE_NIGHT_YES -> "dark"
            Configuration.UI_MODE_NIGHT_NO -> "light"
            else -> "light"
        }
    }
    
    /**
     * Add theme change listener
     */
    fun addListener(listener: ThemeChangeListener) {
        if (!listeners.contains(listener)) {
            listeners.add(listener)
        }
    }
    
    /**
     * Remove theme change listener
     */
    fun removeListener(listener: ThemeChangeListener) {
        listeners.remove(listener)
    }
    
    /**
     * Notify all listeners
     */
    private fun notifyListeners() {
        for (listener in listeners) {
            listener.onThemeChanged(currentTheme)
        }
    }
    
    /**
     * Get theme colors
     */
    fun getThemeColors(): ThemeColors {
        return if (currentTheme == "dark") {
            ThemeColors(
                background = 0xFF0A0A0A.toInt(),
                surface = 0xFF1A1A1A.toInt(),
                surfaceElevated = 0xFF242424.toInt(),
                primary = 0xFF3B82F6.toInt(),
                primaryVariant = 0xFF2563EB.toInt(),
                secondary = 0xFF6366F1.toInt(),
                accent = 0xFF8B5CF6.toInt(),
                text = 0xFFE5E5E5.toInt(),
                textSecondary = 0xFFA3A3A3.toInt(),
                textMuted = 0xFF737373.toInt(),
                heading = 0xFFF5F5F5.toInt(),
                link = 0xFF60A5FA.toInt(),
                border = 0xFF333333.toInt(),
                success = 0xFF22C55E.toInt(),
                warning = 0xFFF59E0B.toInt(),
                error = 0xFFEF4444.toInt(),
                onPrimary = 0xFFFFFFFF.toInt(),
                isDark = true
            )
        } else {
            ThemeColors(
                background = 0xFFFFFFFF.toInt(),
                surface = 0xFFF9FAFB.toInt(),
                surfaceElevated = 0xFFFFFFFF.toInt(),
                primary = 0xFF3B82F6.toInt(),
                primaryVariant = 0xFF2563EB.toInt(),
                secondary = 0xFF6366F1.toInt(),
                accent = 0xFF8B5CF6.toInt(),
                text = 0xFF171717.toInt(),
                textSecondary = 0xFF525252.toInt(),
                textMuted = 0xFFA3A3A3.toInt(),
                heading = 0xFF0A0A0A.toInt(),
                link = 0xFF2563EB.toInt(),
                border = 0xFFE5E5E5.toInt(),
                success = 0xFF16A34A.toInt(),
                warning = 0xFFD97706.toInt(),
                error = 0xFFDC2626.toInt(),
                onPrimary = 0xFFFFFFFF.toInt(),
                isDark = false
            )
        }
    }
    
    /**
     * Get color for resource name
     */
    fun getColor(colorName: String): Int {
        return when (colorName) {
            "background" -> getThemeColors().background
            "surface" -> getThemeColors().surface
            "surface_elevated" -> getThemeColors().surfaceElevated
            "primary" -> getThemeColors().primary
            "primary_variant" -> getThemeColors().primaryVariant
            "secondary" -> getThemeColors().secondary
            "accent" -> getThemeColors().accent
            "text" -> getThemeColors().text
            "text_secondary" -> getThemeColors().textSecondary
            "text_muted" -> getThemeColors().textMuted
            "heading" -> getThemeColors().heading
            "link" -> getThemeColors().link
            "border" -> getThemeColors().border
            "success" -> getThemeColors().success
            "warning" -> getThemeColors().warning
            "error" -> getThemeColors().error
            "on_primary" -> getThemeColors().onPrimary
            else -> getThemeColors().text
        }
    }
}

/**
 * Theme change listener interface
 */
interface ThemeChangeListener {
    fun onThemeChanged(theme: String)
}

/**
 * Theme colors data class
 */
data class ThemeColors(
    val background: Int,
    val surface: Int,
    val surfaceElevated: Int,
    val primary: Int,
    val primaryVariant: Int,
    val secondary: Int,
    val accent: Int,
    val text: Int,
    val textSecondary: Int,
    val textMuted: Int,
    val heading: Int,
    val link: Int,
    val border: Int,
    val success: Int,
    val warning: Int,
    val error: Int,
    val onPrimary: Int,
    val isDark: Boolean
)
