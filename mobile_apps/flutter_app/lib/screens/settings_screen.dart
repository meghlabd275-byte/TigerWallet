import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../services/wallet_service.dart';
import '../services/chain_service.dart';
import '../services/api_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';

/// Settings Screen - App settings and preferences
class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  bool _isDarkMode = true;
  bool _biometricEnabled = false;
  bool _pushNotifications = true;
  bool _emailNotifications = false;
  String _currency = 'USD';
  String _language = 'English';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        elevation: 0,
        title: const Text('Settings', style: TextStyle(color: AppColors.textPrimary)),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSection(
            'Appearance',
            [
              _buildSwitchTile(
                'Dark Mode',
                'Use dark theme',
                Icons.dark_mode,
                _isDarkMode,
                (value) => setState(() => _isDarkMode = value),
              ),
            ],
          ),
          _buildSection(
            'Security',
            [
              _buildSwitchTile(
                'Biometric Login',
                'Use fingerprint or face to unlock',
                Icons.fingerprint,
                _biometricEnabled,
                (value) => setState(() => _biometricEnabled = value),
              ),
              _buildNavigationTile(
                'Change Password',
                'Update your wallet password',
                Icons.lock,
                () {},
              ),
              _buildNavigationTile(
                'View Recovery Phrase',
                'Show your 24-word seed phrase',
                Icons.key,
                () => _showRecoveryPhrase(),
              ),
              _buildNavigationTile(
                'Auto-Lock',
                'Lock wallet after inactivity',
                Icons.timer,
                () {},
              ),
            ],
          ),
          _buildSection(
            'Notifications',
            [
              _buildSwitchTile(
                'Push Notifications',
                'Receive transaction alerts',
                Icons.notifications,
                _pushNotifications,
                (value) => setState(() => _pushNotifications = value),
              ),
              _buildSwitchTile(
                'Email Notifications',
                'Receive email updates',
                Icons.email,
                _emailNotifications,
                (value) => setState(() => _emailNotifications = value),
              ),
            ],
          ),
          _buildSection(
            'Preferences',
            [
              _buildNavigationTile(
                'Currency',
                _currency,
                Icons.attach_money,
                () => _showCurrencySelector(),
              ),
              _buildNavigationTile(
                'Language',
                _language,
                Icons.language,
                () => _showLanguageSelector(),
              ),
            ],
          ),
          _buildSection(
            'Network',
            [
              _buildNavigationTile(
                'Default Network',
                'Ethereum',
                Icons.grid_4x4,
                () {},
              ),
              _buildNavigationTile(
                'RPC Endpoints',
                'Manage custom RPC URLs',
                Icons.dns,
                () {},
              ),
            ],
          ),
          _buildSection(
            'Advanced',
            [
              _buildNavigationTile(
                'Gas Settings',
                'Configure gas preferences',
                Icons.local_gas_station,
                () {},
              ),
              _buildNavigationTile(
                'Slippage Tolerance',
                '0.5%',
                Icons.tune,
                () {},
              ),
              _buildNavigationTile(
                'Transaction Deadline',
                '20 minutes',
                Icons.schedule,
                () {},
              ),
            ],
          ),
          _buildSection(
            'Wallet',
            [
              _buildNavigationTile(
                'Manage Wallets',
                'Add or remove blockchain wallets',
                Icons.account_balance_wallet,
                () {},
              ),
              _buildNavigationTile(
                'Connected DApps',
                'View and manage connections',
                Icons.link,
                () {},
              ),
              _buildNavigationTile(
                'Approved Tokens',
                'Manage token approvals',
                Icons.verified,
                () {},
              ),
            ],
          ),
          _buildSection(
            'About',
            [
              _buildNavigationTile(
                'Terms of Service',
                '',
                Icons.description,
                () {},
              ),
              _buildNavigationTile(
                'Privacy Policy',
                '',
                Icons.privacy_tip,
                () {},
              ),
              _buildNavigationTile(
                'Version',
                '1.0.0',
                Icons.info,
                () {},
              ),
            ],
          ),
          const SizedBox(height: 20),
          _buildDangerSection(),
          const SizedBox(height: 40),
        ],
      ),
    );
  }

  Widget _buildSection(String title, List<Widget> children) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 16),
          child: Text(title,
              style: const TextStyle(
                  fontWeight: FontWeight.bold,
                  fontSize: 16,
                  color: AppColors.textPrimary)),
        ),
        Container(
          decoration: BoxDecoration(
            color: AppColors.cardBackground,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: AppColors.border.withOpacity(0.1)),
          ),
          child: Column(children: children),
        ),
      ],
    );
  }

  Widget _buildSwitchTile(
    String title,
    String subtitle,
    IconData icon,
    bool value,
    ValueChanged<bool> onChanged,
  ) {
    return ListTile(
      leading: Container(
        width: 40,
        height: 40,
        decoration: BoxDecoration(
          color: AppColors.primary.withOpacity(0.1),
          borderRadius: BorderRadius.circular(10),
        ),
        child: Icon(icon, color: AppColors.primary, size: 20),
      ),
      title: Text(title,
          style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 14)),
      subtitle: Text(subtitle,
          style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
      trailing: Switch(
        value: value,
        onChanged: onChanged,
        activeColor: AppColors.primary,
      ),
    );
  }

  Widget _buildNavigationTile(
    String title,
    String value,
    IconData icon,
    VoidCallback onTap,
  ) {
    return ListTile(
      onTap: onTap,
      leading: Container(
        width: 40,
        height: 40,
        decoration: BoxDecoration(
          color: AppColors.primary.withOpacity(0.1),
          borderRadius: BorderRadius.circular(10),
        ),
        child: Icon(icon, color: AppColors.primary, size: 20),
      ),
      title: Text(title,
          style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 14)),
      subtitle: value.isNotEmpty
          ? Text(value, style: TextStyle(color: AppColors.textSecondary, fontSize: 12))
          : null,
      trailing: const Icon(Icons.chevron_right, color: AppColors.textSecondary),
    );
  }

  Widget _buildDangerSection() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Padding(
          padding: EdgeInsets.symmetric(vertical: 16),
          child: Text('Danger Zone',
              style: TextStyle(
                  fontWeight: FontWeight.bold,
                  fontSize: 16,
                  color: AppColors.error)),
        ),
        Container(
          decoration: BoxDecoration(
            color: AppColors.cardBackground,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: AppColors.error.withOpacity(0.3)),
          ),
          child: Column(
            children: [
              ListTile(
                onTap: () => _showResetDialog(),
                leading: Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: AppColors.error.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: const Icon(Icons.delete_forever,
                      color: AppColors.error, size: 20),
                ),
                title: const Text('Reset Wallet',
                    style: TextStyle(
                        fontWeight: FontWeight.w500,
                        fontSize: 14,
                        color: AppColors.error)),
                subtitle: Text('Remove wallet and start fresh',
                    style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
                trailing:
                    const Icon(Icons.chevron_right, color: AppColors.error),
              ),
            ],
          ),
        ),
      ],
    );
  }

  void _showRecoveryPhrase() {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        backgroundColor: AppColors.cardBackground,
        title: const Text('Recovery Phrase',
            style: TextStyle(color: AppColors.textPrimary)),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text(
              'Your 24-word recovery phrase is:',
              style: TextStyle(color: AppColors.textSecondary),
            ),
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: AppColors.background,
                borderRadius: BorderRadius.circular(12),
              ),
              child: const Text(
                'word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12 word13 word14 word15 word16 word17 word18 word19 word20 word21 word22 word23 word24',
                style: TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
            ),
            const SizedBox(height: 16),
            const Text(
              'Warning: Never share this phrase with anyone!',
              style: TextStyle(color: AppColors.error, fontSize: 12),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Close'),
          ),
        ],
      ),
    );
  }

  void _showCurrencySelector() {
    showModalBottomSheet(
      context: context,
      backgroundColor: AppColors.cardBackground,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => Container(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Select Currency',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            ...['USD', 'EUR', 'GBP', 'JPY', 'CNY', 'INR'].map((currency) => ListTile(
              title: Text(currency),
              trailing: _currency == currency
                  ? const Icon(Icons.check, color: AppColors.primary)
                  : null,
              onTap: () {
                setState(() => _currency = currency);
                Navigator.pop(context);
              },
            )),
          ],
        ),
      ),
    );
  }

  void _showLanguageSelector() {
    showModalBottomSheet(
      context: context,
      backgroundColor: AppColors.cardBackground,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => Container(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Select Language',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            ...['English', 'Spanish', 'French', 'German', 'Japanese', 'Chinese']
                .map((lang) => ListTile(
                      title: Text(lang),
                      trailing: _language == lang
                          ? const Icon(Icons.check, color: AppColors.primary)
                          : null,
                      onTap: () {
                        setState(() => _language = lang);
                        Navigator.pop(context);
                      },
                    )),
          ],
        ),
      ),
    );
  }

  void _showResetDialog() {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        backgroundColor: AppColors.cardBackground,
        title: const Text('Reset Wallet?',
            style: TextStyle(color: AppColors.textPrimary)),
        content: const Text(
          'This will permanently delete your wallet. Make sure you have backed up your recovery phrase!',
          style: TextStyle(color: AppColors.textSecondary),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () {
              // Reset wallet
              Navigator.pop(context);
            },
            style: ElevatedButton.styleFrom(backgroundColor: AppColors.error),
            child: const Text('Reset'),
          ),
        ],
      ),
    );
  }
}
