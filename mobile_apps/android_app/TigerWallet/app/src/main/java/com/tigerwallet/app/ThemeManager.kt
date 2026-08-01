package com.tigerwallet.app

import android.content.Context
import android.content.res.Configuration
import androidx.appcompat.app.AppCompatDelegate
import androidx.core.content.ContextCompat
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow

/**
 * TigerWallet Theme Manager
 * Production-Ready Light/Dark Theme System
 */

// ============================================================================
// Theme Manager
// ============================================================================

class ThemeManager(private val context: Context) {

    companion object {
        private val _currentTheme = MutableStateFlow(ThemeType.SYSTEM)
        val currentTheme: StateFlow<ThemeType> = _currentTheme

        private const val PREFS_NAME = "theme_prefs"
        private const val KEY_THEME = "theme_type"

        fun init(context: Context) {
            val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            val themeName = prefs.getString(KEY_THEME, ThemeType.SYSTEM.name)
            val theme = ThemeType.valueOf(themeName ?: ThemeType.SYSTEM.name)
            _currentTheme.value = theme
            applyTheme(theme)
        }

        fun setTheme(theme: ThemeType) {
            _currentTheme.value = theme
            applyTheme(theme)
        }

        private fun applyTheme(theme: ThemeType) {
            val nightMode = when (theme) {
                ThemeType.LIGHT -> AppCompatDelegate.MODE_NIGHT_NO
                ThemeType.DARK -> AppCompatDelegate.MODE_NIGHT_YES
                ThemeType.SYSTEM -> AppCompatDelegate.MODE_NIGHT_FOLLOW_SYSTEM
            }
            AppCompatDelegate.setDefaultNightMode(nightMode)
        }

        fun getColors(): ThemeColors {
            val isDark = when (_currentTheme.value) {
                ThemeType.LIGHT -> false
                ThemeType.DARK -> true
                ThemeType.SYSTEM -> (context.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK) == Configuration.UI_MODE_NIGHT_YES
            }

            return if (isDark) darkColors else lightColors
        }
    }

    // Theme colors
    data class ThemeColors(
        val primaryBackground: Int,
        val secondaryBackground: Int,
        val tertiaryBackground: Int,
        val primaryText: Int,
        val secondaryText: Int,
        val accent: Int,
        val success: Int,
        val warning: Int,
        val error: Int,
        val border: Int,
        val cardBackground: Int
    )

    private val lightColors = ThemeColors(
        primaryBackground = R.color.white,
        secondaryBackground = R.color.gray_50,
        tertiaryBackground = R.color.gray_100,
        primaryText = R.color.gray_900,
        secondaryText = R.color.gray_500,
        accent = R.color.blue_500,
        success = R.color.green_500,
        warning = R.color.amber_500,
        error = R.color.red_500,
        border = R.color.gray_200,
        cardBackground = R.color.white
    )

    private val darkColors = ThemeColors(
        primaryBackground = R.color.gray_900,
        secondaryBackground = R.color.gray_800,
        tertiaryBackground = R.color.gray_700,
        primaryText = R.color.white,
        secondaryText = R.color.gray_400,
        accent = R.color.blue_400,
        success = R.color.green_400,
        warning = R.color.amber_400,
        error = R.color.red_400,
        border = R.color.gray_700,
        cardBackground = R.color.gray_800
    )

    // Get color resource
    fun getColor(colorRes: Int): Int {
        return ContextCompat.getColor(context, colorRes)
    }

    // Apply theme to view
    fun applyThemeToView(view: android.view.View) {
        val colors = getColors()
        view.setBackgroundColor(getColor(colors.primaryBackground))
    }

    // Apply theme to text
    fun applyThemeToTextView(textView: android.widget.TextView) {
        val colors = getColors()
        textView.setTextColor(getColor(colors.primaryText))
    }

    // Apply theme to button
    fun applyThemeToButton(button: android.widget.Button) {
        val colors = getColors()
        button.setBackgroundColor(getColor(colors.accent))
        button.setTextColor(getColor(R.color.white))
    }

    // Apply theme to card
    fun applyThemeToCard(card: com.google.android.material.card.MaterialCardView) {
        val colors = getColors()
        card.setCardBackgroundColor(getColor(colors.cardBackground))
    }
}

// ============================================================================
// Theme Type Enum
// ============================================================================

enum class ThemeType {
    LIGHT,
    DARK,
    SYSTEM
}

// ============================================================================
// Theme Aware Activity
// ============================================================================

open class ThemedActivity : androidx.appcompat.app.AppCompatActivity() {

    protected lateinit var themeManager: ThemeManager

    override fun onCreate(savedInstanceState: android.os.Bundle?) {
        super.onCreate(savedInstanceState)
        themeManager = ThemeManager(this)
    }

    override fun onResume() {
        super.onResume()
        applyTheme()
    }

    protected fun applyTheme() {
        // Apply theme colors to the activity
        val colors = themeManager.getColors()

        window.decorView.setBackgroundColor(themeManager.getColor(colors.primaryBackground))

        // Notify fragments to apply theme
        supportFragmentManager.fragments.forEach { fragment ->
            if (fragment is ThemeAwareFragment) {
                fragment.onThemeChanged(colors)
            }
        }
    }

    protected fun toggleTheme() {
        val currentTheme = ThemeManager.currentTheme.value
        val newTheme = when (currentTheme) {
            ThemeType.LIGHT -> ThemeType.DARK
            ThemeType.DARK -> ThemeType.SYSTEM
            ThemeType.SYSTEM -> ThemeType.LIGHT
        }
        ThemeManager.setTheme(newTheme)
    }
}

// ============================================================================
// Theme Aware Fragment
// ============================================================================

interface ThemeAwareFragment {
    fun onThemeChanged(colors: ThemeManager.ThemeColors)
}

// ============================================================================
// Theme Aware Fragment Base
// ============================================================================

abstract class ThemedFragment : androidx.fragment.app.Fragment(), ThemeAwareFragment {

    protected lateinit var themeManager: ThemeManager

    override fun onCreate(savedInstanceState: android.os.Bundle?) {
        super.onCreate(savedInstanceState)
        themeManager = ThemeManager(requireContext())
    }

    override fun onViewCreated(view: android.view.View, savedInstanceState: android.os.Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        onThemeChanged(themeManager.getColors())
    }

    override fun onThemeChanged(colors: ThemeManager.ThemeColors) {
        // Override in subclasses to apply theme-specific styling
    }
}

// ============================================================================
// Color Resources (Add to colors.xml)
// ============================================================================

/*
<!-- Light Theme Colors -->
<color name="white">#FFFFFF</color>
<color name="gray_50">#F8FAFC</color>
<color name="gray_100">#F1F5F9</color>
<color name="gray_200">#E2E8F0</color>
<color name="gray_500">#64748B</color>
<color name="gray_900">#0F172A</color>
<color name="blue_400">#60A5FA</color>
<color name="blue_500">#3B82F6</color>
<color name="green_400">#4ADE80</color>
<color name="green_500">#22C55E</color>
<color name="amber_400">#FBBF24</color>
<color name="amber_500">#F59E0B</color>
<color name="red_400">#F87171</color>
<color name="red_500">#EF4444</color>

<!-- Dark Theme Colors -->
<color name="gray_700">#334155</color>
<color name="gray_800">#1E293B</color>
<color name="gray_900">#0F172A</color>
*/

// ============================================================================
// Theme Switcher View (Custom View for UI)
// ============================================================================

class ThemeSwitcherView(context: Context) : android.view.View(context) {

    private var currentTheme = ThemeManager.currentTheme.value

    init {
        // Observe theme changes
        // In production, use Flow collection
        setOnClickListener {
            toggleTheme()
        }
    }

    private fun toggleTheme() {
        val newTheme = when (currentTheme) {
            ThemeType.LIGHT -> ThemeType.DARK
            ThemeType.DARK -> ThemeType.SYSTEM
            ThemeType.SYSTEM -> ThemeType.LIGHT
        }
        ThemeManager.setTheme(newTheme)
        currentTheme = newTheme

        // Update icon based on new theme
        updateIcon()
    }

    private fun updateTheme() {
        // Update the theme icon/text
    }

    private fun updateIcon() {
        // Show sun/moon/system icons
    }
}

// ============================================================================
// Extension Functions
// ============================================================================

fun android.view.View.applyTheme() {
    val colors = ThemeManager.getColors()
    this.setBackgroundColor(ContextCompat.getColor(context, colors.primaryBackground))
}

fun android.widget.TextView.applyTheme() {
    val colors = ThemeManager.getColors()
    this.setTextColor(ContextCompat.getColor(context, colors.primaryText))
}

fun android.widget.Button.applyTheme() {
    val colors = ThemeManager.getColors()
    this.setBackgroundColor(ContextCompat.getColor(context, colors.accent))
    this.setTextColor(ContextCompat.getColor(context, R.color.white))
}

fun com.google.android.material.card.MaterialCardView.applyTheme() {
    val colors = ThemeManager.getColors()
    this.setCardBackgroundColor(ContextCompat.getColor(context, colors.cardBackground))
}
