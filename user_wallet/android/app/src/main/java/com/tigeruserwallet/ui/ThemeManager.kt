package com.tigeruserwallet.ui

import android.content.Context
import androidx.appcompat.app.AppCompatDelegate

/**
 * ThemeManager — global light/dark switching (mirrors the web ThemeContext).
 *
 * The existing fragments toggled AppCompatDelegate directly without
 * persistence; this centralizes it: the chosen mode is persisted in plain
 * SharedPreferences (theme choice is not a secret) and re-applied on every
 * screen. DayNight base theme applies it everywhere automatically.
 */
object ThemeManager {
    private const val PREFS = "userwallet_ui_prefs"
    private const val KEY_MODE = "theme_mode"

    const val MODE_LIGHT = "light"
    const val MODE_DARK = "dark"
    const val MODE_SYSTEM = "system"

    @Volatile
    private var prefs: android.content.SharedPreferences? = null

    fun init(context: Context) {
        if (prefs != null) return
        prefs = context.applicationContext.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        applyCurrent()
    }

    private fun requirePrefs(): android.content.SharedPreferences =
        prefs ?: throw IllegalStateException("ThemeManager.init() not called")

    fun current(): String = requirePrefs().getString(KEY_MODE, MODE_SYSTEM) ?: MODE_SYSTEM

    fun isDark(): Boolean = when (current()) {
        MODE_DARK -> true
        MODE_LIGHT -> false
        else -> {
            val nightMode = AppCompatDelegate.getDefaultNightMode()
            nightMode == AppCompatDelegate.MODE_NIGHT_YES
        }
    }

    fun set(mode: String) {
        requirePrefs().edit().putString(KEY_MODE, mode).apply()
        applyCurrent()
    }

    fun toggle() {
        set(if (isDark()) MODE_LIGHT else MODE_DARK)
    }

    fun applyCurrent() {
        val nightMode = when (current()) {
            MODE_DARK -> AppCompatDelegate.MODE_NIGHT_YES
            MODE_LIGHT -> AppCompatDelegate.MODE_NIGHT_NO
            else -> AppCompatDelegate.MODE_NIGHT_FOLLOW_SYSTEM
        }
        AppCompatDelegate.setDefaultNightMode(nightMode)
    }
}
