/**
 * TigerWallet Admin - Theme
 * Complete Dark/Light Theme Support
 */

package com.tigerwallet.admin.ui.theme

import android.app.Activity
import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

// Brand Colors
val PrimaryColor = Color(0xFFFF6B35)
val SecondaryColor = Color(0xFF00D4AA)
val AccentColor = Color(0xFF6C5CE7)
val ErrorColor = Color(0xFFE74C3C)
val WarningColor = Color(0xFFF39C12)
val SuccessColor = Color(0xFF27AE60)
val InfoColor = Color(0xFF3498DB)

// Light Theme Colors
val LightBackground = Color(0xFFF8F9FA)
val LightSurface = Color(0xFFFFFFFF)
val LightCard = Color(0xFFFFFFFF)
val LightDivider = Color(0xFFE0E0E0)
val LightText = Color(0xFF1A1A2E)
val LightTextSecondary = Color(0xFF6B7280)

// Dark Theme Colors
val DarkBackground = Color(0xFF0A0A0F)
val DarkSurface = Color(0xFF141419)
val DarkCard = Color(0xFF1A1A24)
val DarkDivider = Color(0xFF2A2A35)
val DarkText = Color(0xFFFFFFFF)
val DarkTextSecondary = Color(0xFFA0A0A0)

private val LightColorScheme = lightColorScheme(
    primary = PrimaryColor,
    secondary = SecondaryColor,
    tertiary = AccentColor,
    error = ErrorColor,
    background = LightBackground,
    surface = LightSurface,
    onPrimary = Color.White,
    onSecondary = Color.White,
    onTertiary = Color.White,
    onError = Color.White,
    onBackground = LightText,
    onSurface = LightText,
    surfaceVariant = LightCard,
    outline = LightDivider,
)

private val DarkColorScheme = darkColorScheme(
    primary = PrimaryColor,
    secondary = SecondaryColor,
    tertiary = AccentColor,
    error = ErrorColor,
    background = DarkBackground,
    surface = DarkSurface,
    onPrimary = Color.White,
    onSecondary = Color.White,
    onTertiary = Color.White,
    onError = Color.White,
    onBackground = DarkText,
    onSurface = DarkText,
    surfaceVariant = DarkCard,
    outline = DarkDivider,
)

@Composable
fun TigerAdminTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit
) {
    val colorScheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val context = LocalContext.current
            if (darkTheme) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        }
        darkTheme -> DarkColorScheme
        else -> LightColorScheme
    }

    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as Activity).window
            window.statusBarColor = colorScheme.surface.toArgb()
            WindowCompat.getInsetsController(window, view).isAppearanceLightStatusBars = !darkTheme
        }
    }

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        content = content
    )
}
