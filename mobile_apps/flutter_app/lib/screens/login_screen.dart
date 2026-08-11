import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../services/wallet_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';

/// Login Screen - Authenticate to access wallet
class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _passwordController = TextEditingController();
  final _secureStorage = const FlutterSecureStorage();
  
  bool _obscurePassword = true;
  bool _isLoading = false;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    _checkBiometric();
  }

  Future<void> _checkBiometric() async {
    // Check if biometric is available and enabled
    // For demo, we'll show biometric option
  }

  Future<void> _login() async {
    if (_passwordController.text.isEmpty) {
      setState(() => _errorMessage = 'Please enter password');
      return;
    }

    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      // Verify password
      final storedPassword = await _secureStorage.read(key: 'wallet_password');
      
      if (storedPassword == null) {
        // First time - set password and create a REAL BIP-39 wallet.
        final mnemonic = await WalletService().generateMnemonic();
        await _secureStorage.write(
          key: 'wallet_password',
          value: _passwordController.text,
        );
        await _secureStorage.write(
          key: 'wallet_seed_phrase',
          value: mnemonic,
        );
        
        if (mounted) {
          Navigator.pushReplacementNamed(context, '/home');
        }
      } else if (storedPassword == _passwordController.text) {
        if (mounted) {
          Navigator.pushReplacementNamed(context, '/home');
        }
      } else {
        setState(() => _errorMessage = 'Incorrect password');
      }
    } catch (e) {
      setState(() => _errorMessage = 'Login failed: $e');
    }

    setState(() => _isLoading = false);
  }

  String _generateSeedPhrase() {
    // DEPRECATED: returned the first 24 BIP-39 words (a constant, identical
    // for every wallet). New wallet creation MUST use
    // WalletService().generateMnemonic() (real BIP-39 entropy + checksum).
    throw StateError(
      'Do not use _generateSeedPhrase(). Use WalletService().generateMnemonic().',
    );
  }

  Future<void> _loginWithBiometric() async {
    // Biometric authentication requires a platform plugin (e.g. local_auth).
    // The previous implementation just navigated to home ("for demo") without
    // any biometric check — an authentication bypass. Until local_auth is
    // wired, biometric login is disabled (no-op).
    if (mounted) {
      setState(() => _errorMessage = 'Biometric login is not available.');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              const Spacer(),
              // Logo
              Container(
                width: 80,
                height: 80,
                decoration: BoxDecoration(
                  color: AppColors.primary,
                  borderRadius: BorderRadius.circular(20),
                ),
                child: const Icon(
                  Icons.currency_bitcoin,
                  size: 50,
                  color: Colors.white,
                ),
              ),
              const SizedBox(height: 24),
              const Text(
                'TigerWallet',
                style: TextStyle(
                  fontSize: 28,
                  fontWeight: FontWeight.bold,
                  color: AppColors.textPrimary,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                'Enter password to unlock',
                style: TextStyle(
                  color: AppColors.textSecondary,
                  fontSize: 14,
                ),
              ),
              const SizedBox(height: 48),
              // Password field
              TextField(
                controller: _passwordController,
                obscureText: _obscurePassword,
                decoration: InputDecoration(
                  labelText: 'Password',
                  hintText: 'Enter your password',
                  prefixIcon: const Icon(Icons.lock_outline),
                  suffixIcon: IconButton(
                    icon: Icon(
                      _obscurePassword
                          ? Icons.visibility_off
                          : Icons.visibility,
                    ),
                    onPressed: () {
                      setState(() => _obscurePassword = !_obscurePassword);
                    },
                  ),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(12),
                  ),
                  filled: true,
                  fillColor: AppColors.cardBackground,
                ),
                onSubmitted: (_) => _login(),
              ),
              if (_errorMessage != null) ...[
                const SizedBox(height: 12),
                Text(
                  _errorMessage!,
                  style: const TextStyle(color: AppColors.error, fontSize: 12),
                ),
              ],
              const SizedBox(height: 24),
              // Login button
              SizedBox(
                width: double.infinity,
                height: 56,
                child: ElevatedButton(
                  onPressed: _isLoading ? null : _login,
                  style: ElevatedButton.styleFrom(
                    backgroundColor: AppColors.primary,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(16),
                    ),
                  ),
                  child: _isLoading
                      ? const CircularProgressIndicator(color: Colors.white)
                      : const Text(
                          'Unlock',
                          style: TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                ),
              ),
              const SizedBox(height: 16),
              // Biometric login
              TextButton.icon(
                onPressed: _loginWithBiometric,
                icon: const Icon(Icons.fingerprint),
                label: const Text('Use Biometric'),
              ),
              const Spacer(),
              // Footer
              Text(
                'Your keys, your crypto',
                style: TextStyle(
                  color: AppColors.textSecondary,
                  fontSize: 12,
                ),
              ),
              const SizedBox(height: 8),
              TextButton(
                onPressed: () {
                  showModalBottomSheet(
                    context: context,
                    backgroundColor: AppColors.cardBackground,
                    shape: const RoundedRectangleBorder(
                      borderRadius: BorderRadius.vertical(
                        top: Radius.circular(20),
                      ),
                    ),
                    builder: (context) => _buildRecoverySheet(),
                  );
                },
                child: const Text('Forgot Password?'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildRecoverySheet() {
    return Container(
      padding: const EdgeInsets.all(20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Recover Wallet',
            style: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 16),
          const Text(
            'If you have forgotten your password, you can recover your wallet using your 24-word recovery phrase.',
            style: TextStyle(color: AppColors.textSecondary),
          ),
          const SizedBox(height: 24),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton(
              onPressed: () {
                Navigator.pop(context);
                Navigator.pushNamed(context, '/create-wallet');
              },
              style: ElevatedButton.styleFrom(
                backgroundColor: AppColors.primary,
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
              child: const Text('Import Recovery Phrase'),
            ),
          ),
          const SizedBox(height: 20),
        ],
      ),
    );
  }

  @override
  void dispose() {
    _passwordController.dispose();
    super.dispose();
  }
}
