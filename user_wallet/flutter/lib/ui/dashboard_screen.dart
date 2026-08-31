/**
 * DashboardScreen — balances, live price feed banner, quick actions, and the
 * Features hub. Live data only; failures surface the backend error.
 */

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/auth_service.dart';
import '../services/live_feed.dart';
import '../services/theme_service.dart';
import '../services/user_wallet.dart';
import 'features_screen.dart';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({super.key});

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen> {
  final LiveFeedSocket _feed = LiveFeedSocket();
  List<dynamic> _wallets = [];
  String? _error;
  Map<String, dynamic>? _feedBanner;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final api = context.read<UserWalletService>();
    try {
      final res = await api.listWallets();
      final w = res?['wallets'] ?? res?['data'];
      if (!mounted) return;
      setState(() {
        _wallets = w is List ? w : [];
        _error = null;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
    }
    _feed.connect(api);
    _feed.stream.listen((frame) {
      if (!mounted) return;
      setState(() => _feedBanner = frame);
    });
  }

  @override
  void dispose() {
    _feed.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final api = context.read<UserWalletService>();
    final auth = context.watch<AuthService>();
    final themeService = context.watch<ThemeService>();

    return Scaffold(
      appBar: AppBar(
        title: const Text('UserWallet'),
        actions: [
          IconButton(
            tooltip: themeService.isDark ? 'Light' : 'Dark',
            icon: Icon(themeService.isDark ? Icons.light_mode : Icons.dark_mode),
            onPressed: () => themeService.toggle(),
          ),
          IconButton(
            tooltip: 'Sign out',
            icon: const Icon(Icons.logout),
            onPressed: () async {
              await auth.logout();
              if (!mounted) return;
              Navigator.of(context).pushReplacement(
                  MaterialPageRoute(builder: (_) => const _Reauth()));
            },
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (_feedBanner != null)
            Card(
              color: theme.cardColor,
              child: ListTile(
                leading: const Icon(Icons.show_chart),
                title: Text(_feedBanner!['type']?.toString() ?? 'live'),
                subtitle: Text(_feedBanner!.toString(),
                    maxLines: 2, overflow: TextOverflow.ellipsis),
              ),
            ),
          if (_error != null)
            Card(
              color: theme.colorScheme.errorContainer,
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Text(_error!, style: TextStyle(color: theme.colorScheme.onErrorContainer)),
              ),
            ),
          Text('Wallets', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          ..._wallets.map((w) => Card(
                child: ListTile(
                  title: Text((w['label'] ?? w['address'] ?? 'Wallet').toString()),
                  subtitle: Text('${w['address'] ?? ''}\nchain ${w['chain_id'] ?? ''}',
                      maxLines: 2, overflow: TextOverflow.ellipsis),
                  isThreeLine: true,
                ),
              )),
          const SizedBox(height: 16),
          Row(children: [
            Expanded(
              child: OutlinedButton.icon(
                icon: const Icon(Icons.refresh),
                label: const Text('Refresh'),
                onPressed: _load,
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: ElevatedButton.icon(
                icon: const Icon(Icons.apps),
                label: const Text('Features'),
                onPressed: () => Navigator.of(context).push(
                    MaterialPageRoute(builder: (_) => FeaturesScreen(api: api))),
              ),
            ),
          ]),
          const SizedBox(height: 8),
          OutlinedButton.icon(
            icon: const Icon(Icons.settings),
            label: const Text('Backend / Chain settings'),
            onPressed: () async {
              final ctl = TextEditingController(text: await api.baseUrl());
              final saved = await showDialog<String>(
                context: context,
                builder: (dctx) => AlertDialog(
                  title: const Text('Backend Server'),
                  content: TextField(
                      controller: ctl,
                      decoration: const InputDecoration(
                          labelText: 'wallet_api base URL (e.g. https://api.example.com)')),
                  actions: [
                    TextButton(onPressed: () => Navigator.pop(dctx), child: const Text('Cancel')),
                    ElevatedButton(onPressed: () => Navigator.pop(dctx, ctl.text), child: const Text('Save')),
                  ],
                ),
              );
              if (saved != null && saved.isNotEmpty) {
                await api.setBaseUrl(saved.trim());
                if (!mounted) return;
                ScaffoldMessenger.of(context)
                    .showSnackBar(const SnackBar(content: Text('Backend URL updated')));
              }
            },
          ),
        ],
      ),
    );
  }
}

class _Reauth extends StatelessWidget {
  const _Reauth();
  @override
  Widget build(BuildContext context) => Scaffold(
        body: Center(
          child: ElevatedButton(
            onPressed: () => Navigator.of(context).pushReplacement(
                MaterialPageRoute(builder: (_) => const DashboardScreen())),
            child: const Text('Continue as guest'),
          ),
        ),
      );
}
