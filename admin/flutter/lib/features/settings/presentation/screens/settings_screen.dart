/**
 * TigerWallet Admin - Settings Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/constants/app_constants.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSection(context, 'Account', [
            _buildSettingTile(context, Icons.person, 'Profile', 'Manage your profile', () => context.go('${AppConstants.settingsRoute}/profile'), isDark),
            _buildSettingTile(context, Icons.lock, 'Security', 'Password and 2FA', () {}, isDark),
            _buildSettingTile(context, Icons.notifications, 'Notifications', 'Push and email notifications', () {}, isDark),
          ], isDark),
          const SizedBox(height: 16),
          _buildSection(context, 'Appearance', [
            _buildSettingTile(context, Icons.dark_mode, 'Theme', 'Dark/Light mode', () {}, isDark),
            _buildSettingTile(context, Icons.language, 'Language', 'English, Spanish, etc.', () {}, isDark),
          ], isDark),
          const SizedBox(height: 16),
          _buildSection(context, 'System', [
            _buildSettingTile(context, Icons.storage, 'Backup', 'Database backups', () {}, isDark),
            _buildSettingTile(context, Icons.webhook, 'Webhooks', 'Manage webhooks', () {}, isDark),
            _buildSettingTile(context, Icons.api, 'API Keys', 'Manage API keys', () {}, isDark),
          ], isDark),
          const SizedBox(height: 16),
          _buildSection(context, 'Support', [
            _buildSettingTile(context, Icons.help, 'Help Center', 'Documentation and guides', () {}, isDark),
            _buildSettingTile(context, Icons.info, 'About', 'Version 1.0.0', () {}, isDark),
          ], isDark),
          const SizedBox(height: 24),
          ElevatedButton(
            onPressed: () {},
            style: ElevatedButton.styleFrom(backgroundColor: AppTheme.errorColor),
            child: const Text('Sign Out'),
          ),
        ],
      ),
    );
  }

  Widget _buildSection(BuildContext context, String title, List<Widget> children, bool isDark) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title, style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold, color: isDark ? Colors.white : const Color(0xFF1A1A2E))),
        const SizedBox(height: 12),
        Container(
          decoration: BoxDecoration(color: isDark ? AppTheme.darkCard : AppTheme.lightCard, borderRadius: BorderRadius.circular(16)),
          child: Column(children: children),
        ),
      ],
    );
  }

  Widget _buildSettingTile(BuildContext context, IconData icon, String title, String subtitle, VoidCallback onTap, bool isDark) {
    return ListTile(
      leading: Icon(icon, color: AppTheme.primaryColor),
      title: Text(title, style: const TextStyle(fontWeight: FontWeight.w600)),
      subtitle: Text(subtitle, style: Theme.of(context).textTheme.bodySmall),
      trailing: const Icon(Icons.chevron_right),
      onTap: onTap,
    );
  }
}
