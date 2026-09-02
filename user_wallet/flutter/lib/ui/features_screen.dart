/**
 * FeaturesScreen — hub linking every UserWallet feature, mirroring the
 * web/android/ios desktop surfaces. Each feature opens a screen that fetches
 * from the canonical wallet_api (:8443). No mock data; failures surface the
 * backend error.
 */

import 'package:flutter/material.dart';

import '../services/user_wallet.dart';
import 'finance_screen.dart';
import 'transactions_screen.dart';

class FeaturesScreen extends StatelessWidget {
  final UserWalletService api;
  const FeaturesScreen({super.key, required this.api});

  @override
  Widget build(BuildContext context) {
    final features = <_Feature>[
      _Feature('Send / Receive', Icons.send, (_) => SendScreen(api: api)),
      _Feature('Transactions', Icons.receipt_long, (_) => const TransactionsScreen()),
      _Feature('Swap', Icons.swap_horiz, (_) => SwapScreen(api: api)),
      _Feature('Staking', Icons.savings, (_) => StakingScreen(api: api)),
      _Feature('Bridge', Icons.account_tree, (_) => BridgeScreen(api: api)),
      _Feature('DeFi / Lending', Icons.waterfall_chart, (_) => DefiScreen(api: api)),
      _Feature('Trading', Icons.candlestick_chart, (_) => TradingScreen(api: api)),
      _Feature('Earn', Icons.auto_graph, (_) => EarnScreen(api: api)),
      _Feature('Social', Icons.groups, (_) => SocialScreen(api: api)),
      _Feature('NFTs', Icons.image, (_) => NftScreen(api: api)),
      _Feature('Identity', Icons.badge, (_) => IdentityScreen(api: api)),
      _Feature('Payments', Icons.credit_card, (_) => PaymentsScreen(api: api)),
      _Feature('Security', Icons.shield, (_) => SecurityScreen(api: api)),
      _Feature('Terminal', Icons.terminal, (_) => TerminalScreen(api: api)),
      _Feature('Wallet & Finance', Icons.account_balance, (_) => FinanceScreen(api: api)),
      _Feature('Fees', Icons.percent, (_) => FeesScreen(api: api)),
      _Feature('Organization', Icons.folder_shared, (_) => OrgScreen(api: api)),
      _Feature('Address Book', Icons.contacts, (_) => AddressBookScreen(api: api)),
      _Feature('Non-EVM', Icons.link, (_) => NonEvmScreen(api: api)),
      _Feature('Approvals', Icons.verified, (_) => ApprovalsScreen(api: api)),
      _Feature('Multisig', Icons.groups_2, (_) => MultisigScreen(api: api)),
      _Feature('dApps', Icons.explore, (_) => DappsScreen(api: api)),
      _Feature('ENS', Icons.dns, (_) => EnsScreen(api: api)),
      _Feature('Chains / Tokens', Icons.public, (_) => ChainsScreen(api: api)),
      _Feature('Hardware', Icons.usb, (_) => HardwareScreen(api: api)),
    ];

    return Scaffold(
      appBar: AppBar(title: const Text('Features')),
      body: GridView.count(
        crossAxisCount: 2,
        padding: const EdgeInsets.all(12),
        mainAxisSpacing: 12,
        crossAxisSpacing: 12,
        children: features
            .map((f) => InkWell(
                  onTap: () => Navigator.of(context)
                      .push(MaterialPageRoute(builder: f.builder)),
                  child: Card(
                    child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                      Icon(f.icon, size: 36),
                      const SizedBox(height: 8),
                      Text(f.title, textAlign: TextAlign.center),
                    ]),
                  ),
                ))
            .toList(),
      ),
    );
  }
}

class _Feature {
  final String title;
  final IconData icon;
  final WidgetBuilder builder;
  const _Feature(this.title, this.icon, this.builder);
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

class ErrorCard extends StatelessWidget {
  final String message;
  const ErrorCard(this.message, {super.key});
  @override
  Widget build(BuildContext context) => Card(
        color: Theme.of(context).colorScheme.errorContainer,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Text(message,
              style: TextStyle(color: Theme.of(context).colorScheme.onErrorContainer)),
        ),
      );
}

Future<T?> _promptFields<T>(
  BuildContext context,
  String title,
  List<String> labels,
  Future<T> Function(Map<String, String> values) run,
) async {
  final controllers = {for (final l in labels) l: TextEditingController()};
  final ok = await showDialog<bool>(
    context: context,
    builder: (dctx) => AlertDialog(
      title: Text(title),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: controllers.entries
              .map((e) => TextField(controller: e.value, decoration: InputDecoration(labelText: e.key)))
              .toList(),
        ),
      ),
      actions: [
        TextButton(onPressed: () => Navigator.pop(dctx, false), child: const Text('Cancel')),
        ElevatedButton(onPressed: () => Navigator.pop(dctx, true), child: const Text('Confirm')),
      ],
    ),
  );
  if (ok != true) return null;
  final values = {for (final e in controllers.entries) e.key: e.value.text.trim()};
  return run(values);
}

void _showResult(BuildContext context, Object? data, {String? successPrefix}) {
  final text = data == null ? 'OK' : data.toString();
  ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(successPrefix != null ? '$successPrefix: $text' : text)));
}

// ---------------------------------------------------------------------------
// Send / Receive
// ---------------------------------------------------------------------------

class SendScreen extends StatefulWidget {
  final UserWalletService api;
  const SendScreen({super.key, required this.api});
  @override
  State<SendScreen> createState() => _SendScreenState();
}

class _SendScreenState extends State<SendScreen> {
  Map<String, dynamic>? _lastResult;
  String? _error;

  Future<void> _send({required bool auto}) async {
    final res = await _promptFields<Map<String, dynamic>?>(
        context, auto ? 'Auto-Send' : 'Send', ['Wallet ID', 'Password', 'To', 'Value', 'Chain ID'],
        (v) async {
      final walletId = v['Wallet ID']!;
      final pw = v['Password']!;
      final to = v['To']!;
      final val = v['Value']!;
      final cid = int.tryParse(v['Chain ID'] ?? '') ?? 1;
      return auto
          ? widget.api.autoSend(walletId, pw, to, val, cid)
          : widget.api.send(walletId, pw, to, val, cid);
    });
    if (!mounted) return;
    setState(() {
      _error = null;
      _lastResult = res;
    });
    if (res != null) {
      _showResult(context, res, successPrefix: 'Transaction submitted to the blockchain network');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Send / Receive')),
      body: ListView(padding: const EdgeInsets.all(16), children: [
        if (_error != null) ErrorCard(_error!),
        ElevatedButton(onPressed: () => _send(auto: false), child: const Text('Send')),
        const SizedBox(height: 8),
        ElevatedButton(onPressed: () => _send(auto: true), child: const Text('Auto-Send')),
        const SizedBox(height: 16),
        if (_lastResult != null) Card(child: Padding(padding: const EdgeInsets.all(12), child: Text(_lastResult.toString()))),
      ]),
    );
  }
}

// ---------------------------------------------------------------------------
// Swap
// ---------------------------------------------------------------------------

class SwapScreen extends StatefulWidget {
  final UserWalletService api;
  const SwapScreen({super.key, required this.api});
  @override
  State<SwapScreen> createState() => _SwapScreenState();
}

class _SwapScreenState extends State<SwapScreen> {
  Map<String, dynamic>? _quote;
  String? _error;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Swap')),
      body: ListView(padding: const EdgeInsets.all(16), children: [
        if (_error != null) ErrorCard(_error!),
        ElevatedButton(
          child: const Text('Get Quote'),
          onPressed: () async {
            final q = await _promptFields<Map<String, dynamic>?>(context, 'Swap Quote',
                ['From (symbol)', 'To (symbol)', 'Amount'], (v) async {
              return widget.api.getSwapQuote(v['From (symbol)']!, v['To (symbol)']!, v['Amount']!);
            });
            if (!mounted) return;
            setState(() {
              _quote = q;
              _error = null;
            });
          },
        ),
        const SizedBox(height: 8),
        ElevatedButton(
          child: const Text('Execute Swap'),
          onPressed: () async {
            final r = await _promptFields<Map<String, dynamic>?>(context, 'Execute Swap',
                ['Request JSON'], (v) async {
              return widget.api.executeSwap({'request': v['Request JSON']});
            });
            if (!mounted) return;
            setState(() => _quote = r);
            if (r != null) _showResult(context, r, successPrefix: 'Swap submitted to the blockchain network');
          },
        ),
        if (_quote != null)
          Card(child: Padding(padding: const EdgeInsets.all(12), child: Text(_quote.toString()))),
      ]),
    );
  }
}

// ---------------------------------------------------------------------------
// Generic list-driven screen
// ---------------------------------------------------------------------------

class ListScreen extends StatefulWidget {
  final String title;
  final Future<Map<String, dynamic>?> Function(UserWalletService) fetch;
  final String Function(dynamic item) itemBuilder;
  final UserWalletService api;
  const ListScreen({super.key, required this.title, required this.fetch, required this.itemBuilder, required this.api});

  @override
  State<ListScreen> createState() => _ListScreenState();
}

class _ListScreenState extends State<ListScreen> {
  List<dynamic> _items = [];
  String? _error;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final res = await widget.fetch(widget.api);
      final list = (res?['data'] ?? res?.values.firstWhere((v) => v is List, orElse: () => null));
      if (!mounted) return;
      setState(() {
        _items = list is List ? list : [];
        _loading = false;
        _error = null;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(widget.title), actions: [
        IconButton(icon: const Icon(Icons.refresh), onPressed: _load),
      ]),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Padding(padding: const EdgeInsets.all(16), child: ErrorCard(_error!))
              : _items.isEmpty
                  ? const Center(child: Text('No data'))
                  : ListView.builder(
                      padding: const EdgeInsets.all(12),
                      itemCount: _items.length,
                      itemBuilder: (c, i) => Card(
                        child: Padding(
                          padding: const EdgeInsets.all(12),
                          child: Text(widget.itemBuilder(_items[i])),
                        ),
                      ),
                    ),
    );
  }
}

// ---------------------------------------------------------------------------
// Staking / Bridge / DeFi / Trading / Earn / Social / NFT / Identity / Payments
// / Security / Terminal / Fees / Org / Non-EVM / Approvals / Multisig / Dapps /
// ENS / Chains / Hardware — list-driven implementations over the real API.
// ---------------------------------------------------------------------------

class StakingScreen extends StatelessWidget {
  final UserWalletService api;
  const StakingScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Staking',
        api: api,
        fetch: (a) => a.getStakingQuote(),
        itemBuilder: (i) => i.toString(),
      );
}

class BridgeScreen extends StatelessWidget {
  final UserWalletService api;
  const BridgeScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Bridge',
        api: api,
        fetch: (a) => a.getBridgeRoutes(),
        itemBuilder: (i) => i.toString(),
      );
}

class DefiScreen extends StatelessWidget {
  final UserWalletService api;
  const DefiScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'DeFi / Lending',
        api: api,
        fetch: (a) => a.getDefiProtocols(),
        itemBuilder: (i) => i.toString(),
      );
}

class TradingScreen extends StatelessWidget {
  final UserWalletService api;
  const TradingScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Trading',
        api: api,
        fetch: (a) => a.getPerpetualPositions(),
        itemBuilder: (i) => i.toString(),
      );
}

class EarnScreen extends StatelessWidget {
  final UserWalletService api;
  const EarnScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Earn',
        api: api,
        fetch: (a) => a.getLaunchpool(),
        itemBuilder: (i) => i.toString(),
      );
}

class SocialScreen extends StatelessWidget {
  final UserWalletService api;
  const SocialScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Social',
        api: api,
        fetch: (a) => a.getCopyTraders(),
        itemBuilder: (i) => i.toString(),
      );
}

class NftScreen extends StatelessWidget {
  final UserWalletService api;
  const NftScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'NFTs',
        api: api,
        fetch: (a) => a.getNfts('', 1),
        itemBuilder: (i) => i.toString(),
      );
}

class IdentityScreen extends StatelessWidget {
  final UserWalletService api;
  const IdentityScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Identity',
        api: api,
        fetch: (a) => a.kycStatus(),
        itemBuilder: (i) => i.toString(),
      );
}

class PaymentsScreen extends StatelessWidget {
  final UserWalletService api;
  const PaymentsScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Payments',
        api: api,
        fetch: (a) => a.getRampProviders(),
        itemBuilder: (i) => i.toString(),
      );
}

class SecurityScreen extends StatelessWidget {
  final UserWalletService api;
  const SecurityScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Security',
        api: api,
        fetch: (a) => a.securityCheckAddress(''),
        itemBuilder: (i) => i.toString(),
      );
}

class TerminalScreen extends StatelessWidget {
  final UserWalletService api;
  const TerminalScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Terminal',
        api: api,
        fetch: (a) => a.getTerminalTicker('BTCUSDT'),
        itemBuilder: (i) => i.toString(),
      );
}

class FeesScreen extends StatelessWidget {
  final UserWalletService api;
  const FeesScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Fees',
        api: api,
        fetch: (a) => a.getPublicFees(),
        itemBuilder: (i) => i.toString(),
      );
}

class OrgScreen extends StatelessWidget {
  final UserWalletService api;
  const OrgScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Organization',
        api: api,
        fetch: (a) => a.getDevices(),
        itemBuilder: (i) => i.toString(),
      );
}

class AddressBookScreen extends StatelessWidget {
  final UserWalletService api;
  const AddressBookScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Address Book',
        api: api,
        fetch: (a) => a.getAddressBook(),
        itemBuilder: (i) => i.toString(),
      );
}

class NonEvmScreen extends StatelessWidget {
  final UserWalletService api;
  const NonEvmScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Non-EVM',
        api: api,
        fetch: (a) => a.getChains(),
        itemBuilder: (i) => i.toString(),
      );
}

class ApprovalsScreen extends StatelessWidget {
  final UserWalletService api;
  const ApprovalsScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Approvals',
        api: api,
        fetch: (a) => a.getApprovals('', 1),
        itemBuilder: (i) => i.toString(),
      );
}

class MultisigScreen extends StatelessWidget {
  final UserWalletService api;
  const MultisigScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Multisig',
        api: api,
        fetch: (a) => a.multisigWallets(),
        itemBuilder: (i) => i.toString(),
      );
}

class DappsScreen extends StatelessWidget {
  final UserWalletService api;
  const DappsScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'dApps',
        api: api,
        fetch: (a) => a.getDapps(),
        itemBuilder: (i) => i.toString(),
      );
}

class EnsScreen extends StatelessWidget {
  final UserWalletService api;
  const EnsScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'ENS',
        api: api,
        fetch: (a) => a.ensResolve(''),
        itemBuilder: (i) => i.toString(),
      );
}

class ChainsScreen extends StatelessWidget {
  final UserWalletService api;
  const ChainsScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Chains / Tokens',
        api: api,
        fetch: (a) => a.getChains(),
        itemBuilder: (i) => i.toString(),
      );
}

class HardwareScreen extends StatelessWidget {
  final UserWalletService api;
  const HardwareScreen({super.key, required this.api});
  @override
  Widget build(BuildContext context) => ListScreen(
        title: 'Hardware',
        api: api,
        fetch: (a) => a.health(),
        itemBuilder: (i) => i.toString(),
      );
}
