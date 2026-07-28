// Theme Provider - Complete Light/Dark Mode System
// Works on every screen throughout the app

import 'package:flutter/material.dart';

enum ThemeModeOption {
  system,
  light,
  dark,
}

class ThemeProvider extends ChangeNotifier {
  ThemeModeOption _themeMode = ThemeModeOption.system;
  
  ThemeModeOption get themeModeOption => _themeMode;
  
  ThemeMode get themeMode {
    switch (_themeMode) {
      case ThemeModeOption.light:
        return ThemeMode.light;
      case ThemeModeOption.dark:
        return ThemeMode.dark;
      case ThemeModeOption.system:
        return ThemeMode.system;
    }
  }
  
  bool get isDarkMode {
    if (_themeMode == ThemeModeOption.dark) return true;
    if (_themeMode == ThemeModeOption.system) {
      return WidgetsBinding.instance.platformDispatcher.platformBrightness == 
          Brightness.dark;
    }
    return false;
  }
  
  void setThemeMode(ThemeModeOption mode) {
    _themeMode = mode;
    _saveThemePreference();
    notifyListeners();
  }
  
  void toggleTheme() {
    if (_themeMode == ThemeModeOption.light) {
      _themeMode = ThemeModeOption.dark;
    } else if (_themeMode == ThemeModeOption.dark) {
      _themeMode = ThemeModeOption.system;
    } else {
      _themeMode = ThemeModeOption.light;
    }
    _saveThemePreference();
    notifyListeners();
  }
  
  void setLightMode() => setThemeMode(ThemeModeOption.light);
  void setDarkMode() => setThemeMode(ThemeModeOption.dark);
  void setSystemMode() => setThemeMode(ThemeModeOption.system);
  
  Future<void> _saveThemePreference() async {
    // Save to local storage
    // await StorageService.set('theme_mode', _themeMode.index);
  }
  
  Future<void> loadThemePreference() async {
    // Load from local storage
    // final savedMode = await StorageService.get('theme_mode');
    // if (savedMode != null) {
    //   _themeMode = ThemeModeOption.values[savedMode];
    //   notifyListeners();
    // }
  }
}

// Theme mode display helpers
extension ThemeModeExtension on ThemeModeOption {
  String get displayName {
    switch (this) {
      case ThemeModeOption.system:
        return 'System';
      case ThemeModeOption.light:
        return 'Light';
      case ThemeModeOption.dark:
        return 'Dark';
    }
  }
  
  IconData get icon {
    switch (this) {
      case ThemeModeOption.system:
        return Icons.brightness_auto;
      case ThemeModeOption.light:
        return Icons.light_mode;
      case ThemeModeOption.dark:
        return Icons.dark_mode;
    }
  }
}
