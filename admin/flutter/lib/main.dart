/**
 * TigerWallet Admin Flutter App
 * Complete Admin Management System
 * 
 * Features:
 * - Dark/Light Theme Support
 * - Complete User Management
 * - KYC Processing
 * - Transaction Management
 * - Token/Pair Management
 * - White Label Management
 * - Analytics Dashboard
 * - Real-time Notifications
 * - Biometric Authentication
 */

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app.dart';
import 'core/theme/app_theme.dart';
import 'core/constants/app_constants.dart';
import 'core/utils/logger.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  // Set preferred orientations
  await SystemChrome.setPreferredOrientations([
    DeviceOrientation.portraitUp,
    DeviceOrientation.portraitDown,
    DeviceOrientation.landscapeLeft,
    DeviceOrientation.landscapeRight,
  ]);
  
  // Initialize shared preferences
  final sharedPreferences = await SharedPreferences.getInstance();
  
  // Initialize logger
  Logger.init();
  
  // Set system UI overlay style
  SystemChrome.setSystemUIOverlayStyle(
    const SystemUiOverlayStyle(
      statusBarColor: Colors.transparent,
      statusBarIconBrightness: Brightness.dark,
      systemNavigationBarColor: Colors.white,
      systemNavigationBarIconBrightness: Brightness.dark,
    ),
  );
  
  runApp(
    ProviderScope(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(sharedPreferences),
      ],
      child: const TigerAdminApp(),
    ),
  );
}

// Shared Preferences Provider
final sharedPreferencesProvider = Provider<SharedPreferences>((ref) {
  throw UnimplementedError('SharedPreferences not initialized');
});

// Theme Mode Provider
final themeModeProvider = StateNotifierProvider<ThemeModeNotifier, ThemeMode>((ref) {
  return ThemeModeNotifier(ref);
});

class ThemeModeNotifier extends StateNotifier<ThemeMode> {
  final Ref ref;
  
  ThemeModeNotifier(this.ref) : super(ThemeMode.system) {
    _loadTheme();
  }
  
  Future<void> _loadTheme() async {
    final prefs = ref.read(sharedPreferencesProvider);
    final themeIndex = prefs.getInt(AppConstants.themeKey) ?? 0;
    state = ThemeMode.values[themeIndex];
  }
  
  Future<void> setTheme(ThemeMode mode) async {
    final prefs = ref.read(sharedPreferencesProvider);
    await prefs.setInt(AppConstants.themeKey, mode.index);
    state = mode;
  }
  
  void toggleTheme() {
    if (state == ThemeMode.light) {
      setTheme(ThemeMode.dark);
    } else if (state == ThemeMode.dark) {
      setTheme(ThemeMode.system);
    } else {
      setTheme(ThemeMode.light);
    }
  }
}
