/**
 * ThemeService - UserWallet Flutter
 * Light/Dark theme switching with persistence (same contract as MasterWallet
 * Flutter so every surface behaves identically).
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

  // Light palette
  static const Color _lightBackground = Color(0xFFFFFFFF);
  static const Color _lightSurface = Color(0xFFF9FAFB);
  static const Color _lightPrimary = Color(0xFF3B82F6);
  static const Color _lightText = Color(0xFF171717);
  static const Color _lightTextSecondary = Color(0xFF525252);
  static const Color _lightBorder = Color(0xFFE5E5E5);
  static const Color _lightError = Color(0xFFDC2626);

  // Dark palette
  static const Color _darkBackground = Color(0xFF0A0A0A);
  static const Color _darkSurface = Color(0xFF1A1A1A);
  static const Color _darkPrimary = Color(0xFF3B82F6);
  static const Color _darkText = Color(0xFFE5E5E5);
  static const Color _darkTextSecondary = Color(0xFFA3A3A3);
  static const Color _darkBorder = Color(0xFF333333);
  static const Color _darkError = Color(0xFFEF4444);

  Future<void> initialize() async {
    if (_isInitialized) return;
    final prefs = await SharedPreferences.getInstance();
    final saved = prefs.getString('userwallet_theme');
    _themeMode = saved == 'dark'
        ? ThemeMode.dark
        : saved == 'light'
            ? ThemeMode.light
            : ThemeMode.system;
    _isInitialized = true;
    notifyListeners();
  }

  Future<void> toggle() async {
    _themeMode = isDark ? ThemeMode.light : ThemeMode.dark;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('userwallet_theme', isDark ? 'dark' : 'light');
    notifyListeners();
  }

  ThemeData get lightTheme {
    return ThemeData(
      brightness: Brightness.light,
      scaffoldBackgroundColor: _lightBackground,
      primaryColor: _lightPrimary,
      cardColor: _lightSurface,
      dividerColor: _lightBorder,
      colorScheme: const ColorScheme.light(
        primary: _lightPrimary,
        onPrimary: Colors.white,
        surface: _lightSurface,
        onSurface: _lightText,
        error: _lightError,
      ),
      textTheme: const TextTheme(
        bodyLarge: TextStyle(color: _lightText),
        bodyMedium: TextStyle(color: _lightTextSecondary),
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: _lightSurface,
        foregroundColor: _lightText,
        elevation: 0,
      ),
      snackBarTheme: const SnackBarThemeData(
        backgroundColor: _lightSurface,
        contentTextStyle: TextStyle(color: _lightText),
      ),
    );
  }

  ThemeData get darkTheme {
    return ThemeData(
      brightness: Brightness.dark,
      scaffoldBackgroundColor: _darkBackground,
      primaryColor: _darkPrimary,
      cardColor: _darkSurface,
      dividerColor: _darkBorder,
      colorScheme: const ColorScheme.dark(
        primary: _darkPrimary,
        onPrimary: Colors.white,
        surface: _darkSurface,
        onSurface: _darkText,
        error: _darkError,
      ),
      textTheme: const TextTheme(
        bodyLarge: TextStyle(color: _darkText),
        bodyMedium: TextStyle(color: _darkTextSecondary),
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: _darkSurface,
        foregroundColor: _darkText,
        elevation: 0,
      ),
      snackBarTheme: const SnackBarThemeData(
        backgroundColor: _darkSurface,
        contentTextStyle: TextStyle(color: _darkText),
      ),
    );
  }
}
