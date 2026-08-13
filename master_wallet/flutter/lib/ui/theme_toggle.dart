/**
 * ThemeToggle — reusable light/dark switch backed by ThemeService.
 *
 * Provided on every page's AppBar so theme switching is globally available.
 * ThemeService is a ChangeNotifier exposed via provider; toggling it rebuilds
 * the MaterialApp (lightTheme/darkTheme/themeMode swap).
 */

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/theme_service.dart';

class ThemeToggle extends StatelessWidget {
  const ThemeToggle({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = context.watch<ThemeService>();
    return IconButton(
      icon: Icon(theme.isDark ? Icons.light_mode_outlined : Icons.dark_mode_outlined),
      tooltip: theme.isDark ? 'Switch to light theme' : 'Switch to dark theme',
      onPressed: () => theme.toggleTheme(),
    );
  }
}
