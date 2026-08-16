import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'screens/splash_screen.dart';
import 'screens/home_screen.dart';
import 'screens/wallet_screen.dart';
import 'screens/swap_screen.dart';
import 'screens/bridge_screen.dart';
import 'screens/staking_screen.dart';
import 'screens/nft_screen.dart';
import 'screens/settings_screen.dart';
import 'screens/login_screen.dart';
import 'screens/create_wallet_screen.dart';
import 'services/wallet_service.dart';
import 'services/chain_service.dart';
import 'services/api_service.dart';
import 'utils/theme.dart';
import 'branding_config.dart';
import 'utils/constants.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  // Set preferred orientations
  await SystemChrome.setPreferredOrientations([
    DeviceOrientation.portraitUp,
    DeviceOrientation.portraitDown,
  ]);
  
  // Set system UI overlay style
  SystemChrome.setSystemUIOverlayStyle(
    const SystemUiOverlayStyle(
      statusBarColor: Colors.transparent,
      statusBarIconBrightness: Brightness.dark,
      systemNavigationBarColor: AppColors.background,
      systemNavigationBarIconBrightness: Brightness.light,
    ),
  );
  
  // Initialize white-label branding (no-op for stock TigerWallet).
  await BrandingConfig.instance.init();

  runApp(const TigerWalletApp());
}

class TigerWalletApp extends StatefulWidget {
  const TigerWalletApp({super.key});

  @override
  State<TigerWalletApp> createState() => _TigerWalletAppState();
}

class _TigerWalletAppState extends State<TigerWalletApp> {
  final FlutterSecureStorage _secureStorage = const FlutterSecureStorage();
  late WalletService _walletService;
  late ChainService _chainService;
  late ApiService _apiService;
  
  bool _isInitialized = false;

  @override
  void initState() {
    super.initState();
    _initializeServices();
  }

  Future<void> _initializeServices() async {
    // Initialize API Service
    _apiService = ApiService(baseUrl: API_BASE_URL);
    
    // Initialize Chain Service
    _chainService = ChainService();
    await _chainService.loadChains();
    
    // Initialize Wallet Service
    _walletService = WalletService(
      secureStorage: _secureStorage,
      chainService: _chainService,
    );
    
    // Check if wallet exists
    await _walletService.checkWalletExists();
    
    setState(() {
      _isInitialized = true;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'TigerWallet',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.lightTheme,
      darkTheme: AppTheme.darkTheme,
      themeMode: ThemeMode.dark,
      home: _isInitialized
          ? const SplashScreen()
          : const Scaffold(
              body: Center(
                child: CircularProgressIndicator(
                  color: AppColors.primary,
                ),
              ),
            ),
      routes: {
        '/home': (context) => const HomeScreen(),
        '/wallet': (context) => const WalletScreen(),
        '/swap': (context) => const SwapScreen(),
        '/bridge': (context) => const BridgeScreen(),
        '/staking': (context) => const StakingScreen(),
        '/nft': (context) => const NFTScreen(),
        '/settings': (context) => const SettingsScreen(),
        '/login': (context) => const LoginScreen(),
        '/create-wallet': (context) => const CreateWalletScreen(),
      },
    );
  }
}

// Export services for use in widgets
export 'services/wallet_service.dart';
export 'services/chain_service.dart';
export 'services/api_service.dart';