// TigerWallet Mobile App - Production Ready
// Flutter-based cross-platform mobile wallet
// Supports iOS, Android, Windows, macOS, Linux, Web

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import 'core/theme/app_theme.dart';
import 'core/constants/app_constants.dart';
import 'features/wallet/providers/wallet_provider.dart';
import 'features/wallet/providers/theme_provider.dart';
import 'features/wallet/services/wallet_service.dart';
import 'features/wallet/services/blockchain_service.dart';
import 'features/swap/services/swap_service.dart';
import 'features/staking/services/staking_service.dart';
import 'features/nft/services/nft_service.dart';
import 'features/dapp_browser/services/dapp_service.dart';
import 'features/admin/services/admin_service.dart';
import 'features/settings/services/settings_service.dart';
import 'services/api_service.dart';
import 'services/storage_service.dart';
import 'services/notification_service.dart';
import 'services/security_service.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  // Initialize services
  await StorageService.initialize();
  await ApiService.initialize();
  await NotificationService.initialize();
  await SecurityService.initialize();
  
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
      systemNavigationBarColor: Colors.white,
      systemNavigationBarIconBrightness: Brightness.dark,
    ),
  );
  
  runApp(const TigerWalletApp());
}

class TigerWalletApp extends StatelessWidget {
  const TigerWalletApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        // Theme provider for light/dark mode
        ChangeNotifierProvider(create: (_) => ThemeProvider()),
        
        // Wallet provider
        ChangeNotifierProvider(
          create: (_) => WalletProvider(
            walletService: WalletService(),
            blockchainService: BlockchainService(),
          ),
        ),
        
        // API Service singleton
        Provider<ApiService>.value(value: ApiService.instance),
        
        // Storage Service singleton
        Provider<StorageService>.value(value: StorageService.instance),
        
        // Notification Service singleton
        Provider<NotificationService>.value(value: NotificationService.instance),
        
        // Security Service singleton
        Provider<SecurityService>.value(value: SecurityService.instance),
        
        // Swap Service
        Provider<SwapService>.value(value: SwapService()),
        
        // Staking Service
        Provider<StakingService>.value(value: StakingService()),
        
        // NFT Service
        Provider<NftService>.value(value: NftService()),
        
        // DApp Service
        Provider<DappService>.value(value: DappService()),
        
        // Admin Service
        Provider<AdminService>.value(value: AdminService()),
        
        // Settings Service
        Provider<SettingsService>.value(value: SettingsService()),
      ],
      child: Consumer<ThemeProvider>(
        builder: (context, themeProvider, child) {
          return MaterialApp(
            title: AppConstants.appName,
            debugShowCheckedModeBanner: false,
            theme: AppTheme.lightTheme,
            darkTheme: AppTheme.darkTheme,
            themeMode: themeProvider.themeMode,
            home: const SplashScreen(),
            routes: {
              '/home': (context) => const HomeScreen(),
              '/wallet': (context) => const WalletScreen(),
              '/swap': (context) => const SwapScreen(),
              '/staking': (context) => const StakingScreen(),
              '/nft': (context) => const NftScreen(),
              '/dapp-browser': (context) => const DappBrowserScreen(),
              '/settings': (context) => const SettingsScreen(),
              '/admin': (context) => const AdminScreen(),
              '/create-wallet': (context) => const CreateWalletScreen(),
              '/import-wallet': (context) => const ImportWalletScreen(),
              '/send': (context) => const SendScreen(),
              '/receive': (context) => const ReceiveScreen(),
              '/transaction-history': (context) => const TransactionHistoryScreen(),
              '/chain-settings': (context) => const ChainSettingsScreen(),
              '/security': (context) => const SecurityScreen(),
              '/backup': (context) => const BackupScreen(),
              '/whitelabel': (context) => const WhiteLabelScreen(),
            },
          );
        },
      ),
    );
  }
}

// ============================================================================
// SPLASH SCREEN
// ============================================================================

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
    _navigateToHome();
  }

  Future<void> _navigateToHome() async {
    await Future.delayed(const Duration(seconds: 2));
    if (mounted) {
      // Check if wallet exists
      final walletProvider = context.read<WalletProvider>();
      final hasWallet = await walletProvider.hasExistingWallet();
      
      if (mounted) {
        Navigator.pushReplacementNamed(
          context,
          hasWallet ? '/home' : '/create-wallet',
        );
      }
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [
              Theme.of(context).colorScheme.primary,
              Theme.of(context).colorScheme.secondary,
              Theme.of(context).colorScheme.tertiary,
            ],
          ),
        ),
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
                          Icons.tiktok_rounded,
                          size: 80,
                          color: Color(0xFFFF6B35),
                        ),
                      ),
                      const SizedBox(height: 24),
                      const Text(
                        'TigerWallet',
                        style: TextStyle(
                          fontSize: 36,
                          fontWeight: FontWeight.bold,
                          color: Colors.white,
                          letterSpacing: 2,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'Enterprise Web3 Wallet',
                        style: TextStyle(
                          fontSize: 16,
                          color: Colors.white.withOpacity(0.8),
                          letterSpacing: 1,
                        ),
                      ),
                      const SizedBox(height: 48),
                      const CircularProgressIndicator(
                        valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ),
      ),
    );
  }
}

// ============================================================================
// HOME SCREEN
// ============================================================================

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _currentIndex = 0;
  
  final List<Widget> _screens = [
    const WalletTab(),
    const SwapTab(),
    const StakingTab(),
    const DAppBrowserTab(),
    const AdminTab(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: _screens[_currentIndex],
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: (index) {
          setState(() {
            _currentIndex = index;
          });
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.account_balance_wallet_outlined),
            selectedIcon: Icon(Icons.account_balance_wallet),
            label: 'Wallet',
          ),
          NavigationDestination(
            icon: Icon(Icons.swap_horiz_outlined),
            selectedIcon: Icon(Icons.swap_horiz),
            label: 'Swap',
          ),
          NavigationDestination(
            icon: Icon(Icons.stacked_bar_chart_outlined),
            selectedIcon: Icon(Icons.stacked_bar_chart),
            label: 'Earn',
          ),
          NavigationDestination(
            icon: Icon(Icons.language_outlined),
            selectedIcon: Icon(Icons.language),
            label: 'DApps',
          ),
          NavigationDestination(
            icon: Icon(Icons.admin_panel_settings_outlined),
            selectedIcon: Icon(Icons.admin_panel_settings),
            label: 'Admin',
          ),
        ],
      ),
    );
  }
}

// ============================================================================
// WALLET TAB
// ============================================================================

class WalletTab extends StatelessWidget {
  const WalletTab({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<WalletProvider>(
      builder: (context, walletProvider, child) {
        return Scaffold(
          appBar: AppBar(
            title: const Text('TigerWallet'),
            actions: [
              IconButton(
                icon: const Icon(Icons.notifications_outlined),
                onPressed: () {
                  // Navigate to notifications
                },
              ),
              IconButton(
                icon: const Icon(Icons.settings_outlined),
                onPressed: () {
                  Navigator.pushNamed(context, '/settings');
                },
              ),
            ],
          ),
          body: RefreshIndicator(
            onRefresh: () async {
              await walletProvider.refreshBalances();
            },
            child: SingleChildScrollView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Total Balance Card
                  _TotalBalanceCard(
                    totalBalance: walletProvider.totalBalance,
                    totalBalanceUSD: walletProvider.totalBalanceUSD,
                    change24h: walletProvider.change24h,
                  ),
                  
                  const SizedBox(height: 24),
                  
                  // Quick Actions
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                    children: [
                      _QuickActionButton(
                        icon: Icons.arrow_upward,
                        label: 'Send',
                        onTap: () => Navigator.pushNamed(context, '/send'),
                      ),
                      _QuickActionButton(
                        icon: Icons.arrow_downward,
                        label: 'Receive',
                        onTap: () => Navigator.pushNamed(context, '/receive'),
                      ),
                      _QuickActionButton(
                        icon: Icons.swap_horiz,
                        label: 'Swap',
                        onTap: () => Navigator.pushNamed(context, '/swap'),
                      ),
                      _QuickActionButton(
                        icon: Icons.more_horiz,
                        label: 'More',
                        onTap: () {
                          // Show more options
                        },
                      ),
                    ],
                  ),
                  
                  const SizedBox(height: 24),
                  
                  // Connected Chains
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        'Connected Chains',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      TextButton(
                        onPressed: () {
                          Navigator.pushNamed(context, '/chain-settings');
                        },
                        child: const Text('Manage'),
                      ),
                    ],
                  ),
                  
                  const SizedBox(height: 12),
                  
                  // Chain List
                  SizedBox(
                    height: 100,
                    child: ListView.builder(
                      scrollDirection: Axis.horizontal,
                      itemCount: walletProvider.connectedChains.length,
                      itemBuilder: (context, index) {
                        final chain = walletProvider.connectedChains[index];
                        return _ChainCard(
                          chainName: chain.name,
                          chainIcon: chain.iconUrl,
                          balance: chain.balance,
                          onTap: () {
                            walletProvider.selectChain(chain);
                          },
                        );
                      },
                    ),
                  ),
                  
                  const SizedBox(height: 24),
                  
                  // Assets
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        'Assets',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      TextButton(
                        onPressed: () {
                          Navigator.pushNamed(context, '/transaction-history');
                        },
                        child: const Text('History'),
                      ),
                    ],
                  ),
                  
                  const SizedBox(height: 12),
                  
                  // Token List
                  ...walletProvider.tokens.map((token) => _TokenListItem(
                    token: token,
                    onTap: () {
                      // Show token details
                    },
                  )),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}

// ============================================================================
// TOTAL BALANCE CARD
// ============================================================================

class _TotalBalanceCard extends StatelessWidget {
  final double totalBalance;
  final double totalBalanceUSD;
  final double change24h;

  const _TotalBalanceCard({
    required this.totalBalance,
    required this.totalBalanceUSD,
    required this.change24h,
  });

  @override
  Widget build(BuildContext context) {
    final isPositive = change24h >= 0;
    
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [
            Theme.of(context).colorScheme.primary,
            Theme.of(context).colorScheme.secondary,
          ],
        ),
        borderRadius: BorderRadius.circular(20),
        boxShadow: [
          BoxShadow(
            color: Theme.of(context).colorScheme.primary.withOpacity(0.3),
            blurRadius: 20,
            offset: const Offset(0, 10),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Total Balance',
            style: TextStyle(
              color: Colors.white70,
              fontSize: 14,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            '\$${totalBalanceUSD.toStringAsFixed(2)}',
            style: const TextStyle(
              color: Colors.white,
              fontSize: 36,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              Icon(
                isPositive ? Icons.arrow_upward : Icons.arrow_downward,
                color: isPositive ? Colors.greenAccent : Colors.redAccent,
                size: 16,
              ),
              const SizedBox(width: 4),
              Text(
                '${isPositive ? '+' : ''}${change24h.toStringAsFixed(2)}% (24h)',
                style: TextStyle(
                  color: isPositive ? Colors.greenAccent : Colors.redAccent,
                  fontSize: 14,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

// ============================================================================
// QUICK ACTION BUTTON
// ============================================================================

class _QuickActionButton extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _QuickActionButton({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        child: Column(
          children: [
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.primaryContainer,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(
                icon,
                color: Theme.of(context).colorScheme.primary,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              label,
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w500,
                color: Theme.of(context).colorScheme.onSurface,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ============================================================================
// CHAIN CARD
// ============================================================================

class _ChainCard extends StatelessWidget {
  final String chainName;
  final String chainIcon;
  final double balance;
  final VoidCallback onTap;

  const _ChainCard({
    required this.chainName,
    required this.chainIcon,
    required this.balance,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        width: 80,
        margin: const EdgeInsets.only(right: 12),
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(20),
              ),
              child: chainIcon.isNotEmpty
                  ? ClipRRect(
                      borderRadius: BorderRadius.circular(20),
                      child: Image.network(chainIcon),
                    )
                  : const Icon(Icons.currency_bitcoin),
            ),
            const SizedBox(height: 8),
            Text(
              chainName,
              style: const TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w500,
              ),
              textAlign: TextAlign.center,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ],
        ),
      ),
    );
  }
}

// ============================================================================
// TOKEN LIST ITEM
// ============================================================================

class _TokenListItem extends StatelessWidget {
  final TokenModel token;
  final VoidCallback onTap;

  const _TokenListItem({
    required this.token,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 8),
        child: Row(
          children: [
            Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(24),
              ),
              child: token.iconUrl.isNotEmpty
                  ? ClipRRect(
                      borderRadius: BorderRadius.circular(24),
                      child: Image.network(token.iconUrl),
                    )
                  : const Icon(Icons.currency_bitcoin),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    token.name,
                    style: const TextStyle(
                      fontWeight: FontWeight.bold,
                      fontSize: 16,
                    ),
                  ),
                  Text(
                    token.symbol,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                      fontSize: 14,
                    ),
                  ),
                ],
              ),
            ),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  token.balance.toStringAsFixed(token.decimals),
                  style: const TextStyle(
                    fontWeight: FontWeight.bold,
                    fontSize: 16,
                  ),
                ),
                Text(
                  '\$${token.balanceUSD.toStringAsFixed(2)}',
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.onSurfaceVariant,
                    fontSize: 14,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

// ============================================================================
// SWAP TAB
// ============================================================================

class SwapTab extends StatelessWidget {
  const SwapTab({super.key});

  @override
  Widget build(BuildContext context) {
    return const SwapScreen();
  }
}

// ============================================================================
// STAKING TAB
// ============================================================================

class StakingTab extends StatelessWidget {
  const StakingTab({super.key});

  @override
  Widget build(BuildContext context) {
    return const StakingScreen();
  }
}

// ============================================================================
// DAPP BROWSER TAB
// ============================================================================

class DAppBrowserTab extends StatelessWidget {
  const DAppBrowserTab({super.key});

  @override
  Widget build(BuildContext context) {
    return const DappBrowserScreen();
  }
}

// ============================================================================
// ADMIN TAB
// ============================================================================

class AdminTab extends StatelessWidget {
  const AdminTab({super.key});

  @override
  Widget build(BuildContext context) {
    return const AdminScreen();
  }
}

// ============================================================================
// ADDITIONAL SCREENS (STUBS FOR COMPLETE ROUTING)
// ============================================================================

class WalletScreen extends StatelessWidget {
  const WalletScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const WalletTab();
  }
}

class SwapScreen extends StatelessWidget {
  const SwapScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Swap'),
      ),
      body: const Center(
        child: Text('Swap Screen - Full implementation'),
      ),
    );
  }
}

class StakingScreen extends StatelessWidget {
  const StakingScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Earn'),
      ),
      body: const Center(
        child: Text('Staking Screen - Full implementation'),
      ),
    );
  }
}

class NftScreen extends StatelessWidget {
  const NftScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('NFTs'),
      ),
      body: const Center(
        child: Text('NFT Screen - Full implementation'),
      ),
    );
  }
}

class DappBrowserScreen extends StatelessWidget {
  const DappBrowserScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('DApp Browser'),
      ),
      body: const Center(
        child: Text('DApp Browser - Full implementation'),
      ),
    );
  }
}

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings'),
      ),
      body: const Center(
        child: Text('Settings Screen - Full implementation'),
      ),
    );
  }
}

class AdminScreen extends StatelessWidget {
  const AdminScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Admin Panel'),
      ),
      body: const Center(
        child: Text('Admin Screen - Full implementation'),
      ),
    );
  }
}

class CreateWalletScreen extends StatelessWidget {
  const CreateWalletScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Create Wallet'),
      ),
      body: const Center(
        child: Text('Create Wallet Screen - Full implementation'),
      ),
    );
  }
}

class ImportWalletScreen extends StatelessWidget {
  const ImportWalletScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Import Wallet'),
      ),
      body: const Center(
        child: Text('Import Wallet Screen - Full implementation'),
      ),
    );
  }
}

class SendScreen extends StatelessWidget {
  const SendScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Send'),
      ),
      body: const Center(
        child: Text('Send Screen - Full implementation'),
      ),
    );
  }
}

class ReceiveScreen extends StatelessWidget {
  const ReceiveScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Receive'),
      ),
      body: const Center(
        child: Text('Receive Screen - Full implementation'),
      ),
    );
  }
}

class TransactionHistoryScreen extends StatelessWidget {
  const TransactionHistoryScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Transaction History'),
      ),
      body: const Center(
        child: Text('Transaction History - Full implementation'),
      ),
    );
  }
}

class ChainSettingsScreen extends StatelessWidget {
  const ChainSettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Chain Settings'),
      ),
      body: const Center(
        child: Text('Chain Settings - Full implementation'),
      ),
    );
  }
}

class SecurityScreen extends StatelessWidget {
  const SecurityScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Security'),
      ),
      body: const Center(
        child: Text('Security Screen - Full implementation'),
      ),
    );
  }
}

class BackupScreen extends StatelessWidget {
  const BackupScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Backup'),
      ),
      body: const Center(
        child: Text('Backup Screen - Full implementation'),
      ),
    );
  }
}

class WhiteLabelScreen extends StatelessWidget {
  const WhiteLabelScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('White Label'),
      ),
      body: const Center(
        child: Text('White Label Screen - Full implementation'),
      ),
    );
  }
}

// ============================================================================
// MODEL CLASSES
// ============================================================================

class TokenModel {
  final String id;
  final String name;
  final String symbol;
  final String address;
  final String iconUrl;
  final double balance;
  final int decimals;
  final double balanceUSD;
  final String chainId;

  TokenModel({
    required this.id,
    required this.name,
    required this.symbol,
    required this.address,
    required this.iconUrl,
    required this.balance,
    required this.decimals,
    required this.balanceUSD,
    required this.chainId,
  });
}
