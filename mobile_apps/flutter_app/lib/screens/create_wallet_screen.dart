import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../services/wallet_service.dart';
import '../services/google_drive_backup_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';

/// Create Wallet Screen - Create or import wallet
class CreateWalletScreen extends StatefulWidget {
  const CreateWalletScreen({super.key});

  @override
  State<CreateWalletScreen> createState() => _CreateWalletScreenState();
}

class _CreateWalletScreenState extends State<CreateWalletScreen> {
  final _secureStorage = const FlutterSecureStorage();
  final _importController = TextEditingController();
  final _passwordController = TextEditingController();
  final _confirmPasswordController = TextEditingController();
  
  int _currentStep = 0;
  String _seedPhrase = '';
  bool _agreedToTerms = false;
  bool _isCreating = false;

  // Google Drive backup (R2): honest status surfaced to the user. Fail-closed
  // — never fakes success.
  bool _isBackingUp = false;
  String? _backupStatusMessage;
  bool _backupSucceeded = false;

  // Wallet id of the backend-registered wallet, used by the Drive backup.
  String? _backendWalletId;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        elevation: 0,
        title: const Text('Create Wallet',
            style: TextStyle(color: AppColors.textPrimary)),
      ),
      body: Stepper(
        currentStep: _currentStep,
        onStepContinue: _onStepContinue,
        onStepCancel: _onStepCancel,
        controlsBuilder: (context, details) {
          return Padding(
            padding: const EdgeInsets.only(top: 20),
            child: Row(
              children: [
                if (_currentStep < 2)
                  ElevatedButton(
                    onPressed: details.onStepContinue,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: AppColors.primary,
                      padding: const EdgeInsets.symmetric(
                          horizontal: 24, vertical: 12),
                    ),
                    child: const Text('Continue'),
                  ),
                if (_currentStep == 2)
                  ElevatedButton(
                    onPressed: _isCreating ? null : _createWallet,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: AppColors.primary,
                      padding: const EdgeInsets.symmetric(
                          horizontal: 24, vertical: 12),
                    ),
                    child: _isCreating
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : const Text('Create Wallet'),
                  ),
                const SizedBox(width: 12),
                if (_currentStep > 0)
                  TextButton(
                    onPressed: details.onStepCancel,
                    child: const Text('Back'),
                  ),
              ],
            ),
          );
        },
        steps: [
          Step(
            title: const Text('Choose Method'),
            content: _buildMethodSelection(),
            isActive: _currentStep >= 0,
            state: _currentStep > 0 ? StepState.complete : StepState.indexed,
          ),
          Step(
            title: const Text('Set Password'),
            content: _buildPasswordSetup(),
            isActive: _currentStep >= 1,
            state: _currentStep > 1 ? StepState.complete : StepState.indexed,
          ),
          Step(
            title: const Text('Backup Phrase'),
            content: _buildBackupPhrase(),
            isActive: _currentStep >= 2,
            state: _currentStep > 2 ? StepState.complete : StepState.indexed,
          ),
        ],
      ),
    );
  }

  Widget _buildMethodSelection() {
    return Column(
      children: [
        _buildMethodCard(
          icon: Icons.add_circle_outline,
          title: 'Create New Wallet',
          description: 'Generate a new 24-word recovery phrase',
                      onTap: () async {
              _seedPhrase = await WalletService().generateMnemonic();
              setState(() => _currentStep = 1);
            },
        ),
        const SizedBox(height: 16),
        _buildMethodCard(
          icon: Icons.key,
          title: 'Import Existing Wallet',
          description: 'Import using your recovery phrase',
          onTap: () => _showImportDialog(),
        ),
      ],
    );
  }

  Widget _buildMethodCard({
    required IconData icon,
    required String title,
    required String description,
    required VoidCallback onTap,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: AppColors.cardBackground,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: AppColors.border.withOpacity(0.1)),
        ),
        child: Row(
          children: [
            Container(
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                color: AppColors.primary.withOpacity(0.1),
                borderRadius: BorderRadius.circular(14),
              ),
              child: Icon(icon, color: AppColors.primary, size: 28),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title,
                      style: const TextStyle(
                          fontWeight: FontWeight.bold,
                          fontSize: 16,
                          color: AppColors.textPrimary)),
                  const SizedBox(height: 4),
                  Text(description,
                      style: TextStyle(
                          color: AppColors.textSecondary, fontSize: 12)),
                ],
              ),
            ),
            const Icon(Icons.chevron_right, color: AppColors.textSecondary),
          ],
        ),
      ),
    );
  }

  Widget _buildPasswordSetup() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Create a password to secure your wallet',
          style: TextStyle(color: AppColors.textSecondary),
        ),
        const SizedBox(height: 20),
        TextField(
          controller: _passwordController,
          obscureText: true,
          decoration: InputDecoration(
            labelText: 'Password',
            hintText: 'Create a strong password',
            prefixIcon: const Icon(Icons.lock_outline),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            filled: true,
            fillColor: AppColors.cardBackground,
          ),
        ),
        const SizedBox(height: 16),
        TextField(
          controller: _confirmPasswordController,
          obscureText: true,
          decoration: InputDecoration(
            labelText: 'Confirm Password',
            hintText: 'Confirm your password',
            prefixIcon: const Icon(Icons.lock_outline),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            filled: true,
            fillColor: AppColors.cardBackground,
          ),
        ),
        const SizedBox(height: 16),
        Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: AppColors.error.withOpacity(0.1),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              const Icon(Icons.warning_amber, color: AppColors.error, size: 20),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  'If you forget your password, you can recover your wallet using the recovery phrase.',
                  style: TextStyle(color: AppColors.error, fontSize: 12),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildBackupPhrase() {
    final words = _seedPhrase.isNotEmpty ? _seedPhrase.split(' ') : [];
    
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: AppColors.warning.withOpacity(0.1),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: AppColors.warning.withOpacity(0.3)),
          ),
          child: Row(
            children: [
              const Icon(Icons.warning_amber, color: AppColors.warning),
              const SizedBox(width: 12),
              const Expanded(
                child: Text(
                  'Write down these 24 words in order and store them safely.',
                  style: TextStyle(color: AppColors.warning),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 20),
        GridView.builder(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: 3,
            childAspectRatio: 3,
            crossAxisSpacing: 8,
            mainAxisSpacing: 8,
          ),
          itemCount: words.length,
          itemBuilder: (context, index) {
            return Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: AppColors.cardBackground,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                children: [
                  Text(
                    '${index + 1}.',
                    style: TextStyle(
                      color: AppColors.textSecondary,
                      fontSize: 10,
                    ),
                  ),
                  const SizedBox(width: 4),
                  Expanded(
                    child: Text(
                      words[index] ?? '',
                      style: const TextStyle(fontSize: 12),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
            );
          },
        ),
        const SizedBox(height: 20),
        CheckboxListTile(
          value: _agreedToTerms,
          onChanged: (value) {
            setState(() => _agreedToTerms = value ?? false);
          },
          title: const Text(
            'I understand that if I lose my recovery phrase, I will lose access to my funds',
            style: TextStyle(fontSize: 12),
          ),
          controlAffinity: ListTileControlAffinity.leading,
          contentPadding: EdgeInsets.zero,
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            IconButton(
              onPressed: () {
                Clipboard.setData(ClipboardData(text: _seedPhrase));
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Copied to clipboard')),
                );
              },
              icon: const Icon(Icons.copy),
            ),
            const Text('Copy to clipboard'),
          ],
        ),
        const SizedBox(height: 16),
        // R2: Google Drive encrypted-seed backup. Fail-closed — when OAuth is
        // not configured the button surfaces an honest error instead of fake
        // success. The copy-to-clipboard option above remains available.
        _buildGoogleDriveBackupSection(),
      ],
    );
  }

  Widget _buildGoogleDriveBackupSection() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Divider(),
        const SizedBox(height: 8),
        Text(
          'Encrypted backup',
          style: Theme.of(context).textTheme.titleMedium,
        ),
        const SizedBox(height: 4),
        const Text(
          'Upload an AES-256-GCM encrypted copy of your seed to Google Drive '
          '(requires a registered wallet + your password).',
          style: TextStyle(fontSize: 12, color: AppColors.textSecondary),
        ),
        const SizedBox(height: 12),
        SizedBox(
          width: double.infinity,
          child: ElevatedButton.icon(
            onPressed: _isBackingUp ? null : _backupToGoogleDrive,
            icon: _isBackingUp
                ? const SizedBox(
                    height: 18,
                    width: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.cloud_upload),
            label: Text(_isBackingUp ? 'Backing up…' : 'Backup to Google Drive'),
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.primary,
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(vertical: 12),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
              ),
            ),
          ),
        ),
        if (_backupStatusMessage != null) ...[
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: (_backupSucceeded ? Colors.green : Colors.orange)
                  .withOpacity(0.1),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(
                color: (_backupSucceeded ? Colors.green : Colors.orange)
                    .withOpacity(0.3),
              ),
            ),
            child: Row(
              children: [
                Icon(
                  _backupSucceeded ? Icons.check_circle : Icons.info_outline,
                  color: _backupSucceeded ? Colors.green : Colors.orange,
                  size: 20,
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    _backupStatusMessage!,
                    style: TextStyle(
                      fontSize: 12,
                      color: _backupSucceeded ? Colors.green : Colors.orange,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ],
    );
  }

  Future<void> _backupToGoogleDrive() async {
    // Require a registered backend wallet to export the encrypted seed from.
    // If the wallet has not been registered yet (e.g. backend unreachable at
    // create time), attempt a one-off registration now using the locally-held
    // mnemonic + password.
    var walletId = _backendWalletId ??
        await _secureStorage.read(key: 'wallet_id');
    final password = _passwordController.text;
    if (password.isEmpty) {
      setState(() {
        _backupSucceeded = false;
        _backupStatusMessage =
            'Enter and confirm your wallet password first to enable Drive backup.';
      });
      return;
    }

    if (walletId == null || walletId.isEmpty) {
      final ws = WalletService(secureStorage: _secureStorage);
      final created = await ws.createBackendWallet(
        mnemonic: _seedPhrase,
        password: password,
      );
      walletId = created?['id']?.toString();
      if (walletId != null && walletId.isNotEmpty) {
        _backendWalletId = walletId;
      }
    }

    if (walletId == null || walletId.isEmpty) {
      setState(() {
        _backupSucceeded = false;
        _backupStatusMessage =
            'No backend wallet registered — cannot export the encrypted seed. '
            'Check your connection and try again.';
      });
      return;
    }

    setState(() {
      _isBackingUp = true;
      _backupStatusMessage = null;
    });

    final result = await GoogleDriveBackupService(
      secureStorage: _secureStorage,
    ).backup(walletId: walletId, password: password);

    setState(() {
      _isBackingUp = false;
      _backupSucceeded = result.success;
      _backupStatusMessage = result.success
          ? 'Encrypted seed uploaded to Google Drive (file id: ${result.fileId}).'
          : result.error;
    });
  }

  void _showImportDialog() {
    showModalBottomSheet(
      context: context,
      backgroundColor: AppColors.cardBackground,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
          left: 20,
          right: 20,
          top: 20,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Import Wallet',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _importController,
              maxLines: 3,
              decoration: InputDecoration(
                labelText: 'Recovery Phrase',
                hintText: 'Enter your 24-word recovery phrase',
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
            ),
            const SizedBox(height: 20),
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: () {
                  Navigator.pop(context);
                  _seedPhrase = _importController.text.trim();
                  if (_seedPhrase.split(' ').length == 24) {
                    setState(() => _currentStep = 1);
                  } else {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                        content: Text('Please enter exactly 24 words'),
                        backgroundColor: AppColors.error,
                      ),
                    );
                  }
                },
                style: ElevatedButton.styleFrom(
                  backgroundColor: AppColors.primary,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                ),
                child: const Text('Import'),
              ),
            ),
            const SizedBox(height: 20),
          ],
        ),
      ),
    );
  }

  void _onStepContinue() {
    if (_currentStep == 1) {
      if (_passwordController.text.length < 8) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Password must be at least 8 characters'),
            backgroundColor: AppColors.error,
          ),
        );
        return;
      }
      if (_passwordController.text != _confirmPasswordController.text) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Passwords do not match'),
            backgroundColor: AppColors.error,
          ),
        );
        return;
      }
    }
    if (_currentStep < 2) {
      setState(() => _currentStep++);
    }
  }

  void _onStepCancel() {
    if (_currentStep > 0) {
      setState(() => _currentStep--);
    }
  }

  Future<void> _createWallet() async {
    if (!_agreedToTerms) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Please agree to the terms'),
          backgroundColor: AppColors.error,
        ),
      );
      return;
    }

    setState(() => _isCreating = true);

    try {
      // Save password
      await _secureStorage.write(
        key: 'wallet_password',
        value: _passwordController.text,
      );
      
      // Save seed phrase
      await _secureStorage.write(
        key: 'wallet_seed_phrase',
        value: _seedPhrase,
      );

      // Register the wallet with the canonical Go wallet_api backend so that
      // /send, /auto-send and /export-encrypted-seed have a real wallet_id. The
      // backend performs REAL BIP-39/32/44 derivation + AES-256-GCM seed
      // encryption. A failure here is non-fatal for local wallet use; the user
      // can retry backend registration / Drive backup from the backup screen.
      final ws = WalletService(secureStorage: _secureStorage);
      final backendWallet = await ws.createBackendWallet(
        mnemonic: _seedPhrase,
        password: _passwordController.text,
      );
      if (backendWallet?['id'] != null) {
        _backendWalletId = backendWallet!['id'].toString();
      }

      if (mounted) {
        Navigator.pushReplacementNamed(context, '/home');
      }
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Failed to create wallet: $e'),
          backgroundColor: AppColors.error,
        ),
      );
    }

    setState(() => _isCreating = false);
  }

  String _generateSeedPhrase() {
    // DEPRECATED: the previous implementation returned the first 24 BIP-39
    // words - a constant identical for every wallet. New wallet creation MUST
    // use WalletService().generateMnemonic() (real BIP-39 entropy + checksum
    // seeded from Random.secure()).
    throw StateError(
      'Do not use _generateSeedPhrase(). Use WalletService().generateMnemonic() '
      'which generates a real BIP-39 mnemonic.',
    );
  }

  @override
  void dispose() {
    _importController.dispose();
    _passwordController.dispose();
    _confirmPasswordController.dispose();
    super.dispose();
  }
}
