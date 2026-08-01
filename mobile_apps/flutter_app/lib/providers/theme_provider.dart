/**
 * TigerWallet Theme Provider
 * Complete theme management with light/dark mode support
 */

import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';

enum AppThemeMode {
  system,
  light,
  dark,
}

class ThemeProvider extends ChangeNotifier {
  AppThemeMode _themeMode = AppThemeMode.system;
  ThemeMode _flutterThemeMode = ThemeMode.system;
  
  AppThemeMode get themeMode => _themeMode;
  ThemeMode get flutterThemeMode => _flutterThemeMode;
  
  bool get isDarkMode {
    if (_themeMode == AppThemeMode.dark) return true;
    if (_themeMode == AppThemeMode.light) return false;
    
    // System mode
    final brightness = SchedulerBinding.instance.platformDispatcher.platformBrightness;
    return brightness == Brightness.dark;
  }
  
  void setThemeMode(AppThemeMode mode) {
    _themeMode = mode;
    
    switch (mode) {
      case AppThemeMode.light:
        _flutterThemeMode = ThemeMode.light;
        break;
      case AppThemeMode.dark:
        _flutterThemeMode = ThemeMode.dark;
        break;
      case AppThemeMode.system:
        _flutterThemeMode = ThemeMode.system;
        break;
    }
    
    notifyListeners();
  }
  
  void toggleTheme() {
    if (_themeMode == AppThemeMode.light) {
      setThemeMode(AppThemeMode.dark);
    } else {
      setThemeMode(AppThemeMode.light);
    }
  }
  
  // Light theme colors
  static const Color lightBackground = Color(0xFFF5F5F5);
  static const Color lightSurface = Color(0xFFFFFFFF);
  static const Color lightCard = Color(0xFFFFFFFF);
  static const Color lightTextPrimary = Color(0xFF1A1A1A);
  static const Color lightTextSecondary = Color(0xFF666666);
  static const Color lightBorder = Color(0xFFE0E0E0);
  
  // Dark theme colors
  static const Color darkBackground = Color(0xFF121212);
  static const Color darkSurface = Color(0xFF1E1E1E);
  static const Color darkCard = Color(0xFF2C2C2C);
  static const Color darkTextPrimary = Color(0xFFFFFFFF);
  static const Color darkTextSecondary = Color(0xFFB3B3B3);
  static const Color darkBorder = Color(0xFF424242);
  
  // Primary colors
  static const Color primary = Color(0xFFFF6B00);
  static const Color primaryLight = Color(0xFFFF8C33);
  static const Color primaryDark = Color(0xFFCC5500);
  
  // Secondary colors
  static const Color secondary = Color(0xFFFF4444);
  
  // Status colors
  static const Color success = Color(0xFF00C853);
  static const Color warning = Color(0xFFFFAB00);
  static const Color error = Color(0xFFFF1744);
  static const Color info = Color(0xFF2196F3);
  
  // Get current background color
  Color get backgroundColor => isDarkMode ? darkBackground : lightBackground;
  Color get surfaceColor => isDarkMode ? darkSurface : lightSurface;
  Color get cardColor => isDarkMode ? darkCard : lightCard;
  Color get textPrimary => isDarkMode ? darkTextPrimary : lightTextPrimary;
  Color get textSecondary => isDarkMode ? darkTextSecondary : lightTextSecondary;
  Color get borderColor => isDarkMode ? darkBorder : lightBorder;
  
  // Get theme data for Material widgets
  ThemeData getThemeData() {
    return isDarkMode ? _darkTheme : _lightTheme;
  }
  
  static final ThemeData _lightTheme = ThemeData(
    useMaterial3: true,
    brightness: Brightness.light,
    primaryColor: primary,
    scaffoldBackgroundColor: lightBackground,
    colorScheme: const ColorScheme.light(
      primary: primary,
      secondary: secondary,
      surface: lightSurface,
      error: error,
      onPrimary: Colors.white,
      onSecondary: Colors.white,
      onSurface: lightTextPrimary,
      onError: Colors.white,
    ),
    appBarTheme: const AppBarTheme(
      backgroundColor: lightSurface,
      foregroundColor: lightTextPrimary,
      elevation: 0,
      centerTitle: true,
    ),
    cardTheme: CardTheme(
      color: lightCard,
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: primary,
        foregroundColor: Colors.white,
        elevation: 0,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: lightSurface,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: const BorderSide(color: lightBorder),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: const BorderSide(color: lightBorder),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: const BorderSide(color: primary, width: 2),
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
    ),
    bottomNavigationBarTheme: const BottomNavigationBarThemeData(
      backgroundColor: lightSurface,
      selectedItemColor: primary,
      unselectedItemColor: lightTextSecondary,
      type: BottomNavigationBarType.fixed,
    ),
  );
  
  static final ThemeData _darkTheme = ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    primaryColor: primary,
    scaffoldBackgroundColor: darkBackground,
    colorScheme: const ColorScheme.dark(
      primary: primary,
      secondary: secondary,
      surface: darkSurface,
      error: error,
      onPrimary: Colors.white,
      onSecondary: Colors.white,
      onSurface: darkTextPrimary,
      onError: Colors.white,
    ),
    appBarTheme: const AppBarTheme(
      backgroundColor: darkSurface,
      foregroundColor: darkTextPrimary,
      elevation: 0,
      centerTitle: true,
    ),
    cardTheme: CardTheme(
      color: darkCard,
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: primary,
        foregroundColor: Colors.white,
        elevation: 0,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: darkSurface,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: const BorderSide(color: darkBorder),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: const BorderSide(color: darkBorder),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: const BorderSide(color: primary, width: 2),
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
    ),
    bottomNavigationBarTheme: const BottomNavigationBarThemeData(
      backgroundColor: darkSurface,
      selectedItemColor: primary,
      unselectedItemColor: darkTextSecondary,
      type: BottomNavigationBarType.fixed,
    ),
  );
}

// Extension for easy theme access in widgets
extension ThemeExtension on BuildContext {
  ThemeProvider get themeProvider => ThemeProvider.of(this);
  ThemeData get theme => Theme.of(this);
  bool get isDarkMode => theme.brightness == Brightness.dark;
  
  Color get backgroundColor => isDarkMode ? ThemeProvider.darkBackground : ThemeProvider.lightBackground;
  Color get surfaceColor => isDarkMode ? ThemeProvider.darkSurface : ThemeProvider.lightSurface;
  Color get cardColor => isDarkMode ? ThemeProvider.darkCard : ThemeProvider.lightCard;
  Color get textPrimary => isDarkMode ? ThemeProvider.darkTextPrimary : ThemeProvider.lightTextPrimary;
  Color get textSecondary => isDarkMode ? ThemeProvider.darkTextSecondary : ThemeProvider.lightTextSecondary;
}
