/**
 * TigerWallet UserWallet — Flutter app shell.
 *
 * Onboarding-first per the product directive: if the user has no wallet, the
 * app shows Create Wallet / Import Wallet; after creation (password-protected)
 * the user is shown the backup screen (Google Drive export helper + copy).
 * From then on the full feature surface is available.
 *
 * All data comes live from go/wallet_api (:8443). No mock data.
 */

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'services/auth_service.dart';
import 'services/theme_service.dart';
import 'services/user_wallet.dart';
import 'ui/dashboard_screen.dart';
import 'ui/onboarding_screen.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const UserWalletApp());
}

class UserWalletApp extends StatelessWidget {
  const UserWalletApp({super.key});

  @override
  Widget build(BuildContext context) {
    final api = UserWalletService();
    return MultiProvider(
      providers: [
        ChangeNotifierProvider<ThemeService>(
          create: (_) => ThemeService.instance..initialize(),
        ),
        ChangeNotifierProvider<AuthService>(create: (_) => AuthService()),
        Provider<UserWalletService>.value(value: api),
      ],
      child: Consumer<ThemeService>(
        builder: (context, theme, _) {
          return MaterialApp(
            title: 'TigerWallet UserWallet',
            debugShowCheckedModeBanner: false,
            theme: theme.lightTheme,
            darkTheme: theme.darkTheme,
            themeMode: theme.themeMode,
            home: const EntryGate(),
          );
        },
      ),
    );
  }
}

/// Routes a first-time user to Onboarding (create/import/backup), and to the
/// Dashboard once at least one wallet exists locally.
class EntryGate extends StatefulWidget {
  const EntryGate({super.key});

  @override
  State<EntryGate> createState() => _EntryGateState();
}

class _EntryGateState extends State<EntryGate> {
  bool _loadingWallets = true;
  bool _hasWallet = false;

  @override
  void initState() {
    super.initState();
    _check();
  }

  Future<void> _check() async {
    final auth = context.read<AuthService>();
    await auth.initialize();
    final api = context.read<UserWalletService>();
    bool has = false;
    if (auth.isAuthenticated) {
      try {
        final res = await api.listWallets();
        final wallets = res?['wallets'] ?? res?['data'];
        has = wallets is List && wallets.isNotEmpty;
      } catch (_) {
        // guest bootstrap
        try {
          await auth.guest();
          final res = await api.listWallets();
          final wallets = res?['wallets'] ?? res?['data'];
          has = wallets is List && wallets.isNotEmpty;
        } catch (_) {
          // backend unreachable: onboarding is still shown; actions will
          // surface backend errors when attempted
        }
      }
    } else {
      try {
        await auth.guest();
        final res = await api.listWallets();
        final wallets = res?['wallets'] ?? res?['data'];
        has = wallets is List && wallets.isNotEmpty;
      } catch (_) {/* backend unreachable — onboarding attempted again below */}
    }
    if (!mounted) return;
    setState(() {
      _hasWallet = has;
      _loadingWallets = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    if (_loadingWallets) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    if (_hasWallet) return const DashboardScreen();
    return OnboardingScreen(onDone: () {
      setState(() {
        _loadingWallets = true;
      });
      _check();
    });
  }
}
