/**
 * TigerWallet Admin - Main Application
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'core/theme/app_theme.dart';
import 'core/constants/app_constants.dart';
import 'core/network/api_client.dart';
import 'core/network/dio_client.dart';
import 'features/auth/presentation/screens/login_screen.dart';
import 'features/auth/presentation/screens/splash_screen.dart';
import 'features/auth/providers/auth_provider.dart';
import 'features/dashboard/presentation/screens/dashboard_screen.dart';
import 'features/users/presentation/screens/users_screen.dart';
import 'features/users/presentation/screens/user_detail_screen.dart';
import 'features/kyc/presentation/screens/kyc_screen.dart';
import 'features/transactions/presentation/screens/transactions_screen.dart';
import 'features/transactions/presentation/screens/transaction_detail_screen.dart';
import 'features/tokens/presentation/screens/tokens_screen.dart';
import 'features/pairs/presentation/screens/pairs_screen.dart';
import 'features/blockchains/presentation/screens/blockchains_screen.dart';
import 'features/whitelabels/presentation/screens/whitelabels_screen.dart';
import 'features/tickets/presentation/screens/tickets_screen.dart';
import 'features/analytics/presentation/screens/analytics_screen.dart';
import 'features/settings/presentation/screens/settings_screen.dart';
import 'features/settings/presentation/screens/profile_screen.dart';
import 'features/settings/presentation/screens/more_screen.dart';
import 'features/crypto_cards/crypto_cards_screen.dart';
import 'features/margin_trading/margin_trading_screen.dart';
import 'features/feature_flags/feature_flags_screen.dart';
import 'features/billing/billing_screen.dart';
import 'features/p2p_merchant/p2p_merchant_screen.dart';
import 'features/liquidity/liquidity_screen.dart';
import 'features/master_wallet/master_wallet_screen.dart';
import 'features/domains/domain_screen.dart';

// API Client Provider
final apiClientProvider = Provider<ApiClient>((ref) {
  return DioClient();
});

// Router Provider
final routerProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authStateProvider);
  
  return GoRouter(
    initialLocation: AppConstants.splashRoute,
    redirect: (context, state) {
      final isLoggedIn = authState.isAuthenticated;
      final isLoggingIn = state.matchedLocation == AppConstants.loginRoute;
      final isSplash = state.matchedLocation == AppConstants.splashRoute;
      
      if (!isLoggedIn && !isLoggingIn && !isSplash) {
        return AppConstants.loginRoute;
      }
      
      if (isLoggedIn && (isLoggingIn || isSplash)) {
        return AppConstants.dashboardRoute;
      }
      
      return null;
    },
    routes: [
      // Splash
      GoRoute(
        path: AppConstants.splashRoute,
        builder: (context, state) => const SplashScreen(),
      ),
      
      // Auth
      GoRoute(
        path: AppConstants.loginRoute,
        builder: (context, state) => const LoginScreen(),
      ),
      
      // Main Shell with Navigation
      ShellRoute(
        builder: (context, state, child) => MainShell(child: child),
        routes: [
          // Dashboard
          GoRoute(
            path: AppConstants.dashboardRoute,
            builder: (context, state) => const DashboardScreen(),
          ),
          
          // Users
          GoRoute(
            path: AppConstants.usersRoute,
            builder: (context, state) => const UsersScreen(),
            routes: [
              GoRoute(
                path: ':id',
                builder: (context, state) => UserDetailScreen(
                  userId: state.pathParameters['id']!,
                ),
              ),
            ],
          ),
          
          // KYC
          GoRoute(
            path: AppConstants.kycRoute,
            builder: (context, state) => const KycScreen(),
          ),
          
          // Transactions
          GoRoute(
            path: AppConstants.transactionsRoute,
            builder: (context, state) => const TransactionsScreen(),
            routes: [
              GoRoute(
                path: ':id',
                builder: (context, state) => TransactionDetailScreen(
                  transactionId: state.pathParameters['id']!,
                ),
              ),
            ],
          ),
          
          // Tokens
          GoRoute(
            path: AppConstants.tokensRoute,
            builder: (context, state) => const TokensScreen(),
          ),
          
          // Pairs
          GoRoute(
            path: AppConstants.pairsRoute,
            builder: (context, state) => const PairsScreen(),
          ),
          
          // Blockchains
          GoRoute(
            path: AppConstants.blockchainsRoute,
            builder: (context, state) => const BlockchainsScreen(),
          ),
          
          // White Labels
          GoRoute(
            path: AppConstants.whitelabelsRoute,
            builder: (context, state) => const WhiteLabelsScreen(),
          ),
          
          // Tickets
          GoRoute(
            path: AppConstants.ticketsRoute,
            builder: (context, state) => const TicketsScreen(),
          ),
          
          // Analytics
          GoRoute(
            path: AppConstants.analyticsRoute,
            builder: (context, state) => const AnalyticsScreen(),
          ),
          
          // Settings
          GoRoute(
            path: AppConstants.settingsRoute,
            builder: (context, state) => const SettingsScreen(),
            routes: [
              GoRoute(
                path: 'profile',
                builder: (context, state) => const ProfileScreen(),
              ),
            ],
          ),

          // More hub
          GoRoute(
            path: AppConstants.moreRoute,
            builder: (context, state) => const MoreScreen(),
          ),

          // Feature surfaces
          GoRoute(
            path: '/crypto-cards',
            builder: (context, state) => const CryptoCardsScreen(),
          ),
          GoRoute(
            path: '/margin-trading',
            builder: (context, state) => const MarginTradingScreen(),
          ),
          GoRoute(
            path: '/feature-flags',
            builder: (context, state) => const FeatureFlagsScreen(),
          ),
          GoRoute(
            path: '/billing',
            builder: (context, state) => const BillingScreen(),
          ),
          GoRoute(
            path: '/p2p-merchants',
            builder: (context, state) => const P2PMerchantScreen(),
          ),
          GoRoute(
            path: '/liquidity',
            builder: (context, state) => const LiquidityScreen(),
          ),
          GoRoute(
            path: '/master-wallet',
            builder: (context, state) => const MasterWalletScreen(),
          ),

          // Generic domain governance surfaces (14 domains)
          GoRoute(
            path: '/domain/:domain',
            builder: (context, state) {
              final segment = state.pathParameters['domain'] ?? '';
              final cfg = kAdminDomains.firstWhere(
                (d) => d.domain == segment,
                orElse: () => kAdminDomains.first,
              );
              return DomainScreen(config: cfg);
            },
          ),
        ],
      ),
    ],
  );
});

// Main Shell with Bottom Navigation
class MainShell extends ConsumerStatefulWidget {
  final Widget child;
  
  const MainShell({super.key, required this.child});
  
  @override
  ConsumerState<MainShell> createState() => _MainShellState();
}

class _MainShellState extends ConsumerState<MainShell> {
  int _currentIndex = 0;
  
  final List<_NavItem> _navItems = [
    _NavItem(icon: Icons.dashboard, label: 'Dashboard', route: AppConstants.dashboardRoute),
    _NavItem(icon: Icons.people, label: 'Users', route: AppConstants.usersRoute),
    _NavItem(icon: Icons.verified_user, label: 'KYC', route: AppConstants.kycRoute),
    _NavItem(icon: Icons.swap_horiz, label: 'Transactions', route: AppConstants.transactionsRoute),
    _NavItem(icon: Icons.more_horiz, label: 'More', route: AppConstants.moreRoute),
  ];
  
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: widget.child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: (index) {
          setState(() => _currentIndex = index);
          context.go(_navItems[index].route);
        },
        destinations: _navItems.map((item) => NavigationDestination(
          icon: Icon(item.icon),
          label: item.label,
        )).toList(),
      ),
    );
  }
}

class _NavItem {
  final IconData icon;
  final String label;
  final String route;
  
  _NavItem({required this.icon, required this.label, required this.route});
}

// Main App Widget
class TigerAdminApp extends ConsumerWidget {
  const TigerAdminApp({super.key});
  
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeMode = ref.watch(themeModeProvider);
    final router = ref.watch(routerProvider);
    
    return MaterialApp.router(
      title: AppConstants.appName,
      debugShowCheckedModeBanner: false,
      theme: AppTheme.lightTheme,
      darkTheme: AppTheme.darkTheme,
      themeMode: themeMode,
      routerConfig: router,
    );
  }
}
