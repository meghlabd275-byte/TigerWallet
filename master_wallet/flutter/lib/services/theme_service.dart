/**
 * ThemeService - Flutter Implementation
 * Light/Dark theme switching for Master Wallet
 * Production-ready with persistence
 */

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

class ThemeService extends ChangeNotifier {
  static ThemeService? _instance;
  static ThemeService get instance {
    _instance ??= ThemeService._();
    return _instance!;
  }

  ThemeService._();

  ThemeMode _themeMode = ThemeMode.light;
  bool _isInitialized = false;

  ThemeMode get themeMode => _themeMode;
  bool get isDark => _themeMode == ThemeMode.dark;
  bool get isInitialized => _isInitialized;

  // ==================== Colors ====================

  // Light Theme Colors
  static const Color _lightBackground = Color(0xFFFFFFFF);
  static const Color _lightSurface = Color(0xFFF9FAFB);
  static const Color _lightSurfaceElevated = Color(0xFFFFFFFF);
  static const Color _lightPrimary = Color(0xFF3B82F6);
  static const Color _lightPrimaryHover = Color(0xFF2563EB);
  static const Color _lightSecondary = Color(0xFF6366F1);
  static const Color _lightAccent = Color(0xFF8B5CF6);
  static const Color _lightText = Color(0xFF171717);
  static const Color _lightTextSecondary = Color(0xFF525252);
  static const Color _lightTextMuted = Color(0xFFA3A3A3);
  static const Color _lightHeading = Color(0xFF0A0A0A);
  static const Color _lightLink = Color(0xFF2563EB);
  static const Color _lightBorder = Color(0xFFE5E5E5);
  static const Color _lightInput = Color(0xFFFFFFFF);
  static const Color _lightSuccess = Color(0xFF16A34A);
  static const Color _lightWarning = Color(0xFFD97706);
  static const Color _lightError = Color(0xFFDC2626);
  static const Color _lightInfo = Color(0xFF2563EB);

  // Dark Theme Colors
  static const Color _darkBackground = Color(0xFF0A0A0A);
  static const Color _darkSurface = Color(0xFF1A1A1A);
  static const Color _darkSurfaceElevated = Color(0xFF242424);
  static const Color _darkPrimary = Color(0xFF3B82F6);
  static const Color _darkPrimaryHover = Color(0xFF2563EB);
  static const Color _darkSecondary = Color(0xFF6366F1);
  static const Color _darkAccent = Color(0xFF8B5CF6);
  static const Color _darkText = Color(0xFFE5E5E5);
  static const Color _darkTextSecondary = Color(0xFFA3A3A3);
  static const Color _darkTextMuted = Color(0xFF737373);
  static const Color _darkHeading = Color(0xFFF5F5F5);
  static const Color _darkLink = Color(0xFF60A5FA);
  static const Color _darkBorder = Color(0xFF333333);
  static const Color _darkInput = Color(0xFF262626);
  static const Color _darkSuccess = Color(0xFF22C55E);
  static const Color _darkWarning = Color(0xFFF59E0B);
  static const Color _darkError = Color(0xFFEF4444);
  static const Color _darkInfo = Color(0xFF3B82F6);

  // ==================== Initialization ====================

  Future<void> initialize() async {
    if (_isInitialized) return;

    try {
      // Load saved theme preference
      await _loadThemePreference();

      // Apply theme
      _applyTheme();

      // Watch system theme changes
      _watchSystemTheme();

      _isInitialized = true;
    } catch (e) {
      debugPrint('ThemeService initialization failed: $e');
    }
  }

  Future<void> _loadThemePreference() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final savedTheme = prefs.getString('themePreference');

      if (savedTheme != null) {
        _themeMode = savedTheme == 'dark' ? ThemeMode.dark : ThemeMode.light;
      } else {
        // Check system preference
        // Note: In Flutter, we'd need platform channels to check system preference
        _themeMode = ThemeMode.light;
      }
    } catch (e) {
      debugPrint('Failed to load theme preference: $e');
    }
  }

  Future<void> _saveThemePreference() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(
        'themePreference',
        _themeMode == ThemeMode.dark ? 'dark' : 'light',
      );
    } catch (e) {
      debugPrint('Failed to save theme preference: $e');
    }
  }

  void _applyTheme() {
    // Theme is applied via MaterialApp theme parameter
    notifyListeners();
  }

  void _watchSystemTheme() {
    // Note: Would need platform channels for actual system theme watching
    // For now, theme persists until explicitly changed
  }

  // ==================== Theme Control ====================

  Future<void> setTheme(ThemeMode mode) async {
    if (mode != _themeMode) {
      _themeMode = mode;
      _applyTheme();
      await _saveThemePreference();
    }
  }

  Future<void> setLightTheme() async => await setTheme(ThemeMode.light);
  Future<void> setDarkTheme() async => await setTheme(ThemeMode.dark);
  Future<void> toggleTheme() async {
    await setTheme(_themeMode == ThemeMode.dark ? ThemeMode.light : ThemeMode.dark);
  }

  // ==================== Theme Data ====================

  ThemeData get lightTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,
      colorScheme: const ColorScheme.light(
        primary: _lightPrimary,
        onPrimary: Colors.white,
        secondary: _lightSecondary,
        onSecondary: Colors.white,
        surface: _lightSurface,
        onSurface: _lightText,
        error: _lightError,
        onError: Colors.white,
      ),
      scaffoldBackgroundColor: _lightBackground,
      cardColor: _lightSurface,
      dividerColor: _lightBorder,
      appBarTheme: const AppBarTheme(
        backgroundColor: _lightSurface,
        foregroundColor: _lightHeading,
        elevation: 0,
        centerTitle: false,
      ),
      textTheme: const TextTheme(
        displayLarge: TextStyle(color: _lightHeading),
        displayMedium: TextStyle(color: _lightHeading),
        displaySmall: TextStyle(color: _lightHeading),
        headlineLarge: TextStyle(color: _lightHeading),
        headlineMedium: TextStyle(color: _lightHeading),
        headlineSmall: TextStyle(color: _lightHeading),
        titleLarge: TextStyle(color: _lightHeading),
        titleMedium: TextStyle(color: _lightHeading),
        titleSmall: TextStyle(color: _lightHeading),
        bodyLarge: TextStyle(color: _lightText),
        bodyMedium: TextStyle(color: _lightText),
        bodySmall: TextStyle(color: _lightTextSecondary),
        labelLarge: TextStyle(color: _lightText),
        labelMedium: TextStyle(color: _lightTextSecondary),
        labelSmall: TextStyle(color: _lightTextMuted),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: _lightInput,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: _lightBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: _lightBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: _lightPrimary, width: 2),
        ),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: _lightPrimary,
          foregroundColor: Colors.white,
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: _lightPrimary,
          side: const BorderSide(color: _lightPrimary),
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
      ),
      cardTheme: CardTheme(
        color: _lightSurface,
        elevation: 1,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      bottomNavigationBarTheme: const BottomNavigationBarThemeData(
        backgroundColor: _lightSurface,
        selectedItemColor: _lightPrimary,
        unselectedItemColor: _lightTextMuted,
      ),
    );
  }

  ThemeData get darkTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      colorScheme: const ColorScheme.dark(
        primary: _darkPrimary,
        onPrimary: Colors.white,
        secondary: _darkSecondary,
        onSecondary: Colors.white,
        surface: _darkSurface,
        onSurface: _darkText,
        error: _darkError,
        onError: Colors.white,
      ),
      scaffoldBackgroundColor: _darkBackground,
      cardColor: _darkSurface,
      dividerColor: _darkBorder,
      appBarTheme: const AppBarTheme(
        backgroundColor: _darkSurface,
        foregroundColor: _darkHeading,
        elevation: 0,
        centerTitle: false,
      ),
      textTheme: const TextTheme(
        displayLarge: TextStyle(color: _darkHeading),
        displayMedium: TextStyle(color: _darkHeading),
        displaySmall: TextStyle(color: _darkHeading),
        headlineLarge: TextStyle(color: _darkHeading),
        headlineMedium: TextStyle(color: _darkHeading),
        headlineSmall: TextStyle(color: _darkHeading),
        titleLarge: TextStyle(color: _darkHeading),
        titleMedium: TextStyle(color: _darkHeading),
        titleSmall: TextStyle(color: _darkHeading),
        bodyLarge: TextStyle(color: _darkText),
        bodyMedium: TextStyle(color: _darkText),
        bodySmall: TextStyle(color: _darkTextSecondary),
        labelLarge: TextStyle(color: _darkText),
        labelMedium: TextStyle(color: _darkTextSecondary),
        labelSmall: TextStyle(color: _darkTextMuted),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: _darkInput,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: _darkBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: _darkBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: _darkPrimary, width: 2),
        ),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: _darkPrimary,
          foregroundColor: Colors.white,
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: _darkPrimary,
          side: const BorderSide(color: _darkPrimary),
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
        ),
      ),
      cardTheme: CardTheme(
        color: _darkSurface,
        elevation: 1,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      bottomNavigationBarTheme: const BottomNavigationBarThemeData(
        backgroundColor: _darkSurface,
        selectedItemColor: _darkPrimary,
        unselectedItemColor: _darkTextMuted,
      ),
    );
  }

  // ==================== Helper Methods ====================

  Color getBackgroundColor(BuildContext context) {
    return Theme.of(context).brightness == Brightness.dark
        ? _darkBackground
        : _lightBackground;
  }

  Color getSurfaceColor(BuildContext context) {
    return Theme.of(context).brightness == Brightness.dark
        ? _darkSurface
        : _lightSurface;
  }

  Color getPrimaryColor(BuildContext context) {
    return Theme.of(context).colorScheme.primary;
  }

  Color getTextColor(BuildContext context) {
    return Theme.of(context).brightness == Brightness.dark
        ? _darkText
        : _lightText;
  }

  Color getBorderColor(BuildContext context) {
    return Theme.of(context).brightness == Brightness.dark
        ? _darkBorder
        : _lightBorder;
  }
}
