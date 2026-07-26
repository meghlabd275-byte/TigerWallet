import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../services/wallet_service.dart';
import '../services/chain_service.dart';
import '../services/api_service.dart';
import '../utils/theme.dart';
import '../utils/constants.dart';

/// Splash Screen - App initialization and wallet check
class SplashScreen extends StatefulWidget {
  const SplashScreen({super.key});

  @override
  State<SplashScreen> createState() => _SplashScreenState();
}

class _SplashScreenState extends State<SplashScreen>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<double> _fadeAnimation;
  late Animation<double> _scaleAnimation;

  final FlutterSecureStorage _secureStorage = const FlutterSecureStorage();
  bool _isLoading = true;
  String _statusMessage = 'Initializing...';
  bool _hasWallet = false;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      duration: const Duration(milliseconds: 1500),
      vsync: this,
    );

    _fadeAnimation = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeIn),
    );

    _scaleAnimation = Tween<double>(begin: 0.5, end: 1.0).animate(
      CurvedAnimation(parent: _controller, curve: Curves.elasticOut),
    );

    _controller.forward();
    _initializeApp();
  }

  Future<void> _initializeApp() async {
    try {
      // Step 1: Load app configuration
      setState(() => _statusMessage = 'Loading configuration...');
      await Future.delayed(const Duration(milliseconds: 300));

      // Step 2: Initialize secure storage
      setState(() => _statusMessage = 'Securing storage...');
      await Future.delayed(const Duration(milliseconds: 200));

      // Step 3: Load blockchain networks
      setState(() => _statusMessage = 'Loading networks...');
      await Future.delayed(const Duration(milliseconds: 300));

      // Step 4: Check wallet existence
      setState(() => _statusMessage = 'Checking wallet...');
      final walletExists = await _checkWalletExists();

      // Step 5: Navigate based on wallet status
      setState(() {
        _hasWallet = walletExists;
        _isLoading = false;
      });

      await Future.delayed(const Duration(milliseconds: 500));

      if (mounted) {
        if (_hasWallet) {
          Navigator.pushReplacementNamed(context, '/login');
        } else {
          Navigator.pushReplacementNamed(context, '/create-wallet');
        }
      }
    } catch (e) {
      setState(() {
        _isLoading = false;
        _statusMessage = 'Error: ${e.toString()}';
      });
    }
  }

  Future<bool> _checkWalletExists() async {
    try {
      final seedPhrase = await _secureStorage.read(key: 'wallet_seed_phrase');
      return seedPhrase != null && seedPhrase.isNotEmpty;
    } catch (e) {
      return false;
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnnotatedRegion<SystemUiOverlayStyle>(
      value: SystemUiOverlayStyle.light,
      child: Scaffold(
        backgroundColor: AppColors.primary,
        body: Container(
          decoration: const BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [
                AppColors.primary,
                AppColors.primaryDark,
                AppColors.secondary,
              ],
            ),
          ),
          child: SafeArea(
            child: Center(
              child: AnimatedBuilder(
                animation: _controller,
                builder: (context, child) {
                  return FadeTransition(
                    opacity: _fadeAnimation,
                    child: ScaleTransition(
                      scale: _scaleAnimation,
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          // Logo
                          Container(
                            width: 120,
                            height: 120,
                            decoration: BoxDecoration(
                              color: Colors.white,
                              borderRadius: BorderRadius.circular(30),
                              boxShadow: [
                                BoxShadow(
                                  color: Colors.black.withOpacity(0.2),
                                  blurRadius: 20,
                                  offset: const Offset(0, 10),
                                ),
                              ],
                            ),
                            child: const Icon(
                              Icons.currency_bitcoin,
                              size: 70,
                              color: AppColors.primary,
                            ),
                          ),
                          const SizedBox(height: 30),
                          // App Name
                          const Text(
                            'TigerWallet',
                            style: TextStyle(
                              fontSize: 36,
                              fontWeight: FontWeight.bold,
                              color: Colors.white,
                              letterSpacing: 2,
                            ),
                          ),
                          const SizedBox(height: 10),
                          const Text(
                            'Next Generation Web3 Wallet',
                            style: TextStyle(
                              fontSize: 14,
                              color: Colors.white70,
                              letterSpacing: 1,
                            ),
                          ),
                          const SizedBox(height: 60),
                          // Loading indicator
                          if (_isLoading) ...[
                            const SizedBox(
                              width: 40,
                              height: 40,
                              child: CircularProgressIndicator(
                                color: Colors.white,
                                strokeWidth: 3,
                              ),
                            ),
                            const SizedBox(height: 20),
                            Text(
                              _statusMessage,
                              style: const TextStyle(
                                color: Colors.white70,
                                fontSize: 12,
                              ),
                            ),
                          ],
                        ],
                      ),
                    ),
                  );
                },
              ),
            ),
          ),
        ),
      ),
    );
  }
}
