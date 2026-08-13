/**
 * TigerWallet Master Wallet — Flutter app shell.
 *
 * Wires ThemeService (ChangeNotifierProvider) into a runnable MaterialApp and
 * exposes a dashboard that drives the real service-layer fetchers in
 * lib/services/* against the canonical Go backend (:8450). There is NO mock
 * data: every value rendered is fetched live; on failure the UI surfaces the
 * backend error rather than a fake.
 *
 * See master_wallet/CANONICAL_API_CONTRACT.md for the endpoint contract.
 */

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'services/auth_service.dart';
import 'services/master_wallet_service.dart';
import 'services/theme_service.dart';
import 'ui/auth_screen.dart';
import 'ui/dashboard_screen.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const TigerWalletApp());
}

class TigerWalletApp extends StatelessWidget {
  const TigerWalletApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider<ThemeService>(
          create: (_) => ThemeService.instance..initialize(),
        ),
        ChangeNotifierProvider<AuthService>(create: (_) => AuthService()),
        Provider<MasterWalletService>(create: (_) => MasterWalletService()),
      ],
      child: Consumer<ThemeService>(
        builder: (context, theme, _) {
          // Default route; AuthGate switches between auth + dashboard.
          return MaterialApp(
            title: 'TigerWallet Master',
            debugShowCheckedModeBanner: false,
            theme: theme.lightTheme,
            darkTheme: theme.darkTheme,
            themeMode: theme.themeMode,
            home: const AuthGate(),
          );
        },
      ),
    );
  }
}

/// Routes to the auth screen when there is no JWT, and to the dashboard once
/// the AuthService holds a token. Propagates the token to every downstream
/// service so authenticated fetchers carry the Bearer JWT.
class AuthGate extends StatefulWidget {
  const AuthGate({super.key});

  @override
  State<AuthGate> createState() => _AuthGateState();
}

class _AuthGateState extends State<AuthGate> {
  bool _ready = false;

  @override
  void initState() {
    super.initState();
    // Allow the ThemeService (async initialize in provider create) to settle
    // before the first frame paints.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) setState(() => _ready = true);
    });
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthService>();
    final walletSvc = context.read<MasterWalletService>();
    // Keep the shared MasterWalletService token in sync with auth state.
    walletSvc.setToken(auth.token);

    if (!_ready) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    if (!auth.tokenIsValid) {
      return const AuthScreen();
    }
    return const DashboardScreen();
  }
}

/// Extension used by the gate to test auth state without leaking the nullable
/// token detail into the UI.
extension on AuthService {
  bool get tokenIsValid => token != null && token!.isNotEmpty;
}
