/**
 * TigerWallet Master Wallet — full feature screens.
 *
 * Every screen fetches from the canonical Go backend (:8450) through the real
 * service layer (MasterWalletService / TreasuryService / MultiSigService /
 * AuditService / PolicyEngineService). No mock data: backend errors surface
 * verbatim in the UI. All screens inherit the app ThemeService (light/dark),
 * so the theme switch applies everywhere.
 */

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../features/audit/audit_service.dart';
import '../features/multisig/multisig_service.dart';
import '../features/policy/policy_service.dart';
import '../features/treasury/treasury_service.dart';
import '../services/auth_service.dart';
import '../services/master_wallet_service.dart';

/// Resolve the first master wallet id via the shared service. Returns null
/// (and the screens show an honest empty state) when the account has none.
Future<String?> _firstWalletId(MasterWalletService svc) async {
  final wallets = await svc.getWallets();
  if (wallets.isEmpty) return null;
  return wallets.first.id;
}

/// Shared async-state wrapper: renders loading / error / empty honestly.
class AsyncSection<T> extends StatelessWidget {
  final Future<T> future;
  final Widget Function(T data) builder;
  final String emptyHint;

  const AsyncSection({
    super.key,
    required this.future,
    required this.builder,
    this.emptyHint = 'Nothing returned.',
  });

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<T>(
      future: future,
      builder: (context, snap) {
        if (snap.connectionState != ConnectionState.done) {
          return const Padding(
            padding: EdgeInsets.all(16),
            child: Center(child: CircularProgressIndicator()),
          );
        }
        if (snap.hasError) {
          return Padding(
            padding: const EdgeInsets.all(12),
            child: Text('${snap.error}',
                style: TextStyle(color: Theme.of(context).colorScheme.error)),
          );
        }
        final data = snap.data;
        if (data == null || (data is List && data.isEmpty)) {
          return Padding(
            padding: const EdgeInsets.all(12),
            child: Text(emptyHint,
                style: Theme.of(context).textTheme.bodySmall),
          );
        }
        return builder(data);
      },
    );
  }
}

class FeaturesScreen extends StatelessWidget {
  const FeaturesScreen({super.key});

  static const _features = <(String, IconData, Widget Function())>[
    ('Sub-Wallets', Icons.account_tree, SubWalletsScreen.new),
    ('Send', Icons.send, SendScreen.new),
    ('Auto-Sign Ops', Icons.handyman, AutoSignOpsScreen.new),
    ('Treasury', Icons.account_balance, TreasuryScreen.new),
    ('Multisig', Icons.lock, MultisigScreen.new),
    ('Auto-Sign', Icons.key, AutoSignScreen.new),
    ('Fees', Icons.percent, FeesScreen.new),
    ('Policies', Icons.rule, PoliciesScreen.new),
    ('Users', Icons.people, UsersScreen.new),
    ('Chains', Icons.link, ChainsScreen.new),
    ('Tokens', Icons.token, TokensScreen.new),
    ('Feature Flags', Icons.flag, FlagsScreen.new),
    ('Webhooks & Alerts', Icons.notifications, WebhooksScreen.new),
    ('Audit Log', Icons.receipt_long, AuditScreen.new),
    ('Analytics', Icons.bar_chart, AnalyticsScreen.new),
    ('Passkeys', Icons.badge, PasskeysScreen.new),
    ('Withdraw', Icons.upload, WithdrawScreen.new),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('All Features')),
      body: GridView.count(
        crossAxisCount: 2,
        padding: const EdgeInsets.all(12),
        mainAxisSpacing: 12,
        crossAxisSpacing: 12,
        children: [
          for (final (label, icon, make) in _features)
            Card(
              child: InkWell(
                onTap: () => Navigator.of(context)
                    .push(MaterialPageRoute(builder: (_) => make())),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(icon, size: 32),
                    const SizedBox(height: 8),
                    Text(label, textAlign: TextAlign.center),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}

/// Shared helpers for the simple CRUD screens.
class _CrudControllers {
  final map = <String, TextEditingController>{};
  TextEditingController call(String key) =>
      map.putIfAbsent(key, () => TextEditingController());
  void dispose() {
    for (final c in map.values) {
      c.dispose();
    }
  }
}

Widget _field(String label, TextEditingController c,
        {bool obscure = false, TextInputType? keyboard}) =>
    Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: TextField(
        controller: c,
        obscureText: obscure,
        keyboardType: keyboard,
        decoration: InputDecoration(labelText: label, border: const OutlineInputBorder()),
      ),
    );

void _snack(BuildContext context, String msg) =>
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));

// ============================== Sub-Wallets ==================================

class SubWalletsScreen extends StatefulWidget {
  const SubWalletsScreen({super.key});
  @override
  State<SubWalletsScreen> createState() => _SubWalletsScreenState();
}

class _SubWalletsScreenState extends State<SubWalletsScreen> {
  final _c = _CrudControllers();
  int _reload = 0;

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('Sub-Wallets')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.getSubWallets(wid),
                builder: (subs) => Column(
                  children: [
                    for (final s in subs)
                      ListTile(
                        dense: true,
                        title: Text('${s['label'] ?? s['name'] ?? 'Sub-wallet'}'),
                        subtitle: Text('${s['address'] ?? s['id']}'),
                        trailing: TextButton(
                          onPressed: () => _c('sid').text = '${s['id']}',
                          child: const Text('Use'),
                        ),
                      ),
                  ],
                ),
              ),
              const Divider(),
              _field('Sub-wallet ID', _c('sid')),
              _field('To address', _c('to')),
              _field('Amount', _c('amount'), keyboard: TextInputType.number),
              _field('Wallet password', _c('password'), obscure: true),
              FilledButton(
                onPressed: () async {
                  try {
                    final r = await mws.transferSubWallet(
                      walletId: wid,
                      subWalletId: _c('sid').text.trim(),
                      to: _c('to').text.trim(),
                      amount: _c('amount').text.trim(),
                      password: _c('password').text,
                    );
                    if (mounted) {
                      _snack(context,
                          'Transfer submitted to the blockchain network: ${r['transaction_hash'] ?? r['tx_hash'] ?? ''}');
                    }
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: const Text('Transfer'),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Send (sign + broadcast) ======================

class SendScreen extends StatefulWidget {
  const SendScreen({super.key});
  @override
  State<SendScreen> createState() => _SendScreenState();
}

class _SendScreenState extends State<SendScreen> {
  final _c = _CrudControllers();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    return Scaffold(
      appBar: AppBar(title: const Text('Send')),
      body: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          _field('To address', _c('to')),
          _field('Amount (e.g. 0.5)', _c('amount'), keyboard: TextInputType.number),
          _field('Token contract (empty = native)', _c('token')),
          _field('Wallet password', _c('password'), obscure: true),
          FilledButton(
            onPressed: () async {
              try {
                final wid = await _firstWalletId(mws);
                if (wid == null) {
                  if (mounted) _snack(context, 'No master wallet.');
                  return;
                }
                final r = await mws.sendTransaction(
                  walletId: wid,
                  chainId: 1,
                  toAddress: _c('to').text.trim(),
                  amount: BigInt.parse(
                      (double.tryParse(_c('amount').text) ?? 0) == 0
                          ? '0'
                          : ((double.parse(_c('amount')) * 1e18).toInt()).toString()),
                  password: _c('password').text,
                  token: _c('token').text.trim().isEmpty ? null : _c('token').text.trim(),
                );
                if (mounted) {
                  _snack(context,
                      'Transaction submitted to the blockchain network: ${r.txHash ?? ''}');
                  _c('password').clear();
                }
              } catch (e) {
                if (mounted) _snack(context, '$e');
              }
            },
            child: const Text('Sign & broadcast'),
          ),
        ],
      ),
    );
  }
}

// ============================== Auto-Sign Ops ================================

class AutoSignOpsScreen extends StatefulWidget {
  const AutoSignOpsScreen({super.key});
  @override
  State<AutoSignOpsScreen> createState() => _AutoSignOpsScreenState();
}

class _AutoSignOpsScreenState extends State<AutoSignOpsScreen> {
  final _c = _CrudControllers();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  Map<String, dynamic> _autoSignBody() => {
        'mnemonic': _c('mnemonic').text.trim(),
        'chain_id': int.tryParse(_c('chainId').text) ?? 1,
        'chain_type': _c('chainType').text.trim().isEmpty ? 'evm' : _c('chainType').text.trim(),
        'tx_type': _c('txType').text.trim().isEmpty ? 'send' : _c('txType').text.trim(),
        'to_address': _c('to').text.trim(),
        'value': _c('value').text.trim(),
        if (_c('tokenAddr').text.trim().isNotEmpty) 'token_address': _c('tokenAddr').text.trim(),
      };

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    return Scaffold(
      appBar: AppBar(title: const Text('Auto-Sign Ops')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              const Text('Check auto-sign policy',
                  style: TextStyle(fontWeight: FontWeight.bold)),
              _field('Tx type (send/claim/swap/trade)', _c('chkType')),
              _field('Value', _c('chkValue'), keyboard: TextInputType.number),
              FilledButton(
                onPressed: () async {
                  try {
                    final r = await mws.checkAutoSignPolicy(wid, {
                      'tx_type': _c('chkType').text.trim(),
                      'value': _c('chkValue').text.trim(),
                    });
                    if (mounted) {
                      _snack(context,
                          '${r['allowed'] == true ? 'ALLOWED' : 'DENIED'} — ${r['reason'] ?? ''}');
                    }
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: const Text('Check policy'),
              ),
              const Divider(),
              const Text('Auto-sign transaction (24-word seed)',
                  style: TextStyle(fontWeight: FontWeight.bold)),
              _field('24-word mnemonic', _c('mnemonic')),
              _field('Chain ID', _c('chainId'), keyboard: TextInputType.number),
              _field('Chain type (evm/solana/bitcoin/cosmos)', _c('chainType')),
              _field('Tx type', _c('txType')),
              _field('To address', _c('to')),
              _field('Value', _c('value'), keyboard: TextInputType.number),
              _field('Token contract (optional)', _c('tokenAddr')),
              Row(
                children: [
                  Expanded(
                    child: FilledButton(
                      onPressed: () async {
                        try {
                          final r = await mws.autoSignTransaction(wid, _autoSignBody());
                          if (mounted) {
                            _snack(context,
                                'Transaction submitted to the blockchain network: ${r['transaction_hash'] ?? r['tx_hash'] ?? ''}');
                            _c('mnemonic').clear();
                          }
                        } catch (e) {
                          if (mounted) _snack(context, '$e');
                        }
                      },
                      child: const Text('Auto-sign tx'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () async {
                        try {
                          await mws.userWalletAutoSign(wid, _autoSignBody());
                          if (mounted) {
                            _snack(context, 'UserWallet auto-sign configuration saved.');
                            _c('mnemonic').clear();
                          }
                        } catch (e) {
                          if (mounted) _snack(context, '$e');
                        }
                      },
                      child: const Text('UW auto-sign'),
                    ),
                  ),
                ],
              ),
              const Divider(),
              const Text('Revenue payout (SuperAdmin co-sign required)',
                  style: TextStyle(fontWeight: FontWeight.bold)),
              _field('Destination address', _c('rpTo')),
              _field('Amount', _c('rpAmount'), keyboard: TextInputType.number),
              _field('Wallet password', _c('rpPassword'), obscure: true),
              _field('Withdrawal ID (co-signed)', _c('rpWid')),
              FilledButton(
                onPressed: () async {
                  try {
                    final r = await mws.revenuePayout(
                      wid,
                      _c('rpTo').text.trim(),
                      _c('rpAmount').text.trim(),
                      _c('rpPassword').text,
                      withdrawalId: _c('rpWid').text.trim(),
                    );
                    if (mounted) {
                      _snack(context, 'Payout submitted: ${r?.status ?? ''}');
                      _c('rpPassword').clear();
                    }
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: const Text('Execute payout'),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Treasury =====================================

class TreasuryScreen extends StatefulWidget {
  const TreasuryScreen({super.key});
  @override
  State<TreasuryScreen> createState() => _TreasuryScreenState();
}

class _TreasuryScreenState extends State<TreasuryScreen> {
  final _c = _CrudControllers();
  String? _wid;
  TreasuryService? _svc;
  int _reload = 0;

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    final mws = context.read<MasterWalletService>();
    final wid = await _firstWalletId(mws);
    if (!mounted) return;
    setState(() {
      _wid = wid;
      if (wid != null) {
        _svc = TreasuryService(
            masterWalletId: wid, token: context.read<AuthService>().token);
      }
    });
  }

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final svc = _svc;
    _reload; // rebuild trigger
    return Scaffold(
      appBar: AppBar(title: const Text('Treasury')),
      body: svc == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(12),
              children: [
                AsyncSection<TreasuryOverview>(
                  future: svc.getOverview(),
                  builder: (ov) => Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Total value: ${ov.totalValue}'),
                      Text('Hot: ${ov.hotWalletValue} · Cold: ${ov.coldWalletValue}'),
                      Text('Today: ${ov.todayTransactions} txs · ${ov.todayVolume}'),
                    ],
                  ),
                ),
                const Divider(),
                _field('To', _c('to')),
                _field('Amount', _c('amount'), keyboard: TextInputType.number),
                _field('Password', _c('password'), obscure: true),
                FilledButton(
                  onPressed: () async {
                    try {
                      await svc.transfer(
                        to: _c('to').text.trim(),
                        amount: double.tryParse(_c('amount').text) ?? 0,
                        password: _c('password').text,
                      );
                      if (mounted) _snack(context, 'Transfer submitted');
                      setState(() => _reload++);
                    } catch (e) {
                      if (mounted) _snack(context, '$e');
                    }
                  },
                  child: const Text('Transfer'),
                ),
                const Divider(),
                AsyncSection<List<TreasuryTransaction>>(
                  future: svc.getTransactions(),
                  builder: (txs) => Column(
                    children: [
                      for (final t in txs)
                        ListTile(
                          dense: true,
                          title: Text('${t.type} — ${t.amount} ${t.token}'),
                          subtitle: Text(t.fromAccount),
                        ),
                    ],
                  ),
                ),
              ],
            ),
    );
  }
}

// ============================== Multisig =====================================

class MultisigScreen extends StatefulWidget {
  const MultisigScreen({super.key});
  @override
  State<MultisigScreen> createState() => _MultisigScreenState();
}

class _MultisigScreenState extends State<MultisigScreen> {
  final _c = _CrudControllers();
  MultiSigService? _svc;
  int _reload = 0;

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    final mws = context.read<MasterWalletService>();
    final wid = await _firstWalletId(mws);
    if (!mounted) return;
    setState(() {
      if (wid != null) {
        _svc = MultiSigService(
            masterWalletId: wid, token: context.read<AuthService>().token);
      }
    });
  }

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final svc = _svc;
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('Multisig')),
      body: svc == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(12),
              children: [
                _field('Name', _c('name')),
                _field('Owners (comma-separated)', _c('owners')),
                _field('Threshold', _c('threshold'), keyboard: TextInputType.number),
                FilledButton(
                  onPressed: () async {
                    try {
                      final owners = _c('owners')
                          .text
                          .split(',')
                          .map((s) => s.trim())
                          .where((s) => s.isNotEmpty)
                          .toList();
                      await svc.createWallet(
                        name: _c('name').text.trim(),
                        owners: owners,
                        threshold: int.tryParse(_c('threshold').text) ?? 0,
                      );
                      if (mounted) _snack(context, 'Multisig wallet created');
                      setState(() => _reload++);
                    } catch (e) {
                      if (mounted) _snack(context, '$e');
                    }
                  },
                  child: const Text('Create multisig wallet'),
                ),
                const Divider(),
                AsyncSection<List<MultiSigWallet>>(
                  future: svc.getWallets(),
                  builder: (wallets) => Column(
                    children: [
                      for (final w in wallets)
                        Card(
                          child: ListTile(
                            title: Text('${w.name} (${w.threshold}/${w.owners.length})'),
                            subtitle: Text(w.address),
                            trailing: const Icon(Icons.chevron_right),
                            onTap: () => Navigator.of(context).push(
                              MaterialPageRoute(
                                builder: (_) =>
                                    _MultisigTxScreen(svc: svc, wallet: w),
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ],
            ),
    );
  }
}

class _MultisigTxScreen extends StatelessWidget {
  final MultiSigService svc;
  final MultiSigWallet wallet;
  const _MultisigTxScreen({required this.svc, required this.wallet});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(wallet.name)),
      body: AsyncSection<List<MultiSigTx>>(
        future: svc.getTransactions(wallet.id),
        builder: (txs) => ListView(
          children: [
            for (final t in txs)
              ListTile(
                title: Text('${t.to} — ${t.amount} ${t.token}'),
                subtitle:
                    Text('${t.status} · ${t.approvalCount}/${t.requiredApprovals}'),
                trailing: Wrap(
                  spacing: 4,
                  children: [
                    TextButton(
                      onPressed: () async {
                        try {
                          await svc.signTransaction(t.id);
                          if (context.mounted) _snack(context, 'Signed');
                        } catch (e) {
                          if (context.mounted) _snack(context, '$e');
                        }
                      },
                      child: const Text('Sign'),
                    ),
                    TextButton(
                      onPressed: () async {
                        try {
                          await svc.executeTransaction(t.id);
                          if (context.mounted) _snack(context, 'Executed');
                        } catch (e) {
                          if (context.mounted) _snack(context, '$e');
                        }
                      },
                      child: const Text('Exec'),
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}

// ============================== Auto-Sign ====================================

class AutoSignScreen extends StatefulWidget {
  const AutoSignScreen({super.key});
  @override
  State<AutoSignScreen> createState() => _AutoSignScreenState();
}

class _AutoSignScreenState extends State<AutoSignScreen> {
  final _c = _CrudControllers();
  int _reload = 0;

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('Auto-Sign')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) {
            return const Center(child: Text('No master wallet.'));
          }
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              _field('Rule name', _c('name')),
              _field('Max amount', _c('max'), keyboard: TextInputType.number),
              FilledButton(
                onPressed: () async {
                  try {
                    await mws.createAutoSignRule(wid, {
                      'name': _c('name').text.trim(),
                      'rule_type': 'transfer',
                      'max_amount': _c('max').text.trim(),
                    });
                    if (mounted) _snack(context, 'Rule added');
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: const Text('Add rule'),
              ),
              const Divider(),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.getAutoSignRules(wid),
                builder: (rules) => Column(
                  children: [
                    for (final r in rules)
                      ListTile(
                        dense: true,
                        title: Text('${r['name'] ?? r['id']}'),
                        subtitle: Text(
                            '${r['rule_type'] ?? ''} · max ${r['max_amount'] ?? ''} · ${r['is_active'] == true ? 'active' : 'off'}'),
                        trailing: IconButton(
                          icon: const Icon(Icons.delete_outline),
                          onPressed: () async {
                            await mws.deleteAutoSignRule(wid, '${r['id']}');
                            setState(() => _reload++);
                          },
                        ),
                      ),
                  ],
                ),
              ),
              const Divider(),
              const Text('Daemon policy',
                  style: TextStyle(fontWeight: FontWeight.bold)),
              AsyncSection<Map<String, dynamic>>(
                future: mws.getAutoSignPolicy(wid),
                builder: (policy) => Column(
                  children: [
                    for (final entry in const [
                      ['enabled', 'Auto-Sign Daemon Enabled'],
                      ['allow_transfer', 'Allow Transfers'],
                      ['allow_swap', 'Allow Swaps'],
                      ['allow_stake', 'Allow Staking'],
                      ['allow_nft_transfer', 'Allow NFT Transfers'],
                      ['allow_personal_sign', 'Allow Personal Sign'],
                      ['allow_typed_data_sign', 'Allow Typed-Data Sign'],
                    ])
                      SwitchListTile(
                        dense: true,
                        title: Text(entry[1]),
                        value: policy[entry[0]] == true,
                        onChanged: (v) async {
                          try {
                            await mws.updateAutoSignPolicy(wid, {entry[0]: v});
                            if (mounted) _snack(context, 'Policy saved');
                            setState(() => _reload++);
                          } catch (e) {
                            if (mounted) _snack(context, '$e');
                          }
                        },
                      ),
                    ListTile(
                      dense: true,
                      title: const Text('Max Auto Value (wei)'),
                      subtitle: Text(
                          '${policy['max_auto_value_wei'] ?? '0'} (0 = unlimited)'),
                    ),
                  ],
                ),
              ),
              const Divider(),
              const Text('Auto-sign logs',
                  style: TextStyle(fontWeight: FontWeight.bold)),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.listAutoSignLogs(wid),
                builder: (logs) => Column(
                  children: [
                    for (final l in logs)
                      ListTile(
                        dense: true,
                        title: Text('${l['action'] ?? l['status'] ?? 'log'}'),
                        subtitle:
                            Text('${l['tx_hash'] ?? ''} ${l['created_at'] ?? ''}'),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Fees =========================================

class FeesScreen extends StatefulWidget {
  const FeesScreen({super.key});
  @override
  State<FeesScreen> createState() => _FeesScreenState();
}

class _FeesScreenState extends State<FeesScreen> {
  final _c = _CrudControllers();
  int _reload = 0;

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('Fees')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              _field('Fee type', _c('type')),
              _field('Fee percentage', _c('pct'), keyboard: TextInputType.number),
              _field('Fee fixed (wei)', _c('fixed'), keyboard: TextInputType.number),
              FilledButton(
                onPressed: () async {
                  try {
                    await mws.createFee(wid, {
                      'fee_type': _c('type').text.trim(),
                      'fee_percentage':
                          double.tryParse(_c('pct').text) ?? 0,
                      'fee_fixed': _c('fixed').text.trim(),
                    });
                    if (mounted) _snack(context, 'Fee added');
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: const Text('Add fee'),
              ),
              const Divider(),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.getFees(wid),
                builder: (fees) => Column(
                  children: [
                    for (final f in fees)
                      ListTile(
                        dense: true,
                        title: Text(
                            '${f['fee_type'] ?? ''} — ${f['fee_percentage'] ?? 0}%'),
                        subtitle: Text('${f['fee_fixed'] ?? ''}'),
                        trailing: Wrap(
                          children: [
                            IconButton(
                              icon: Icon((f['is_active'] == true)
                                  ? Icons.toggle_on
                                  : Icons.toggle_off),
                              onPressed: () async {
                                await mws.updateFee(wid, '${f['id']}',
                                    {'is_active': !(f['is_active'] == true)});
                                setState(() => _reload++);
                              },
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline),
                              onPressed: () async {
                                await mws.deleteFee(wid, '${f['id']}');
                                setState(() => _reload++);
                              },
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Policies =====================================

class PoliciesScreen extends StatefulWidget {
  const PoliciesScreen({super.key});
  @override
  State<PoliciesScreen> createState() => _PoliciesScreenState();
}

class _PoliciesScreenState extends State<PoliciesScreen> {
  final _c = _CrudControllers();
  PolicyEngineService? _svc;
  int _reload = 0;

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    final mws = context.read<MasterWalletService>();
    final wid = await _firstWalletId(mws);
    if (!mounted) return;
    setState(() {
      if (wid != null) {
        _svc = PolicyEngineService(
            masterWalletId: wid, token: context.read<AuthService>().token);
      }
    });
  }

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final svc = _svc;
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('Policies')),
      body: svc == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(12),
              children: [
                _field('Policy name', _c('name')),
                _field('Policy type', _c('type')),
                FilledButton(
                  onPressed: () async {
                    try {
                      await svc.createPolicy(
                        name: _c('name').text.trim(),
                        type: _c('type').text.trim(),
                        conditions: const {},
                        action: 'enforce',
                      );
                      if (mounted) _snack(context, 'Policy added');
                      setState(() => _reload++);
                    } catch (e) {
                      if (mounted) _snack(context, '$e');
                    }
                  },
                  child: const Text('Add policy'),
                ),
                const Divider(),
                AsyncSection<List<Policy>>(
                  future: svc.getPolicies(),
                  builder: (policies) => Column(
                    children: [
                      for (final p in policies)
                        ListTile(
                          dense: true,
                          title: Text('${p.name} (${p.type})'),
                          subtitle: Text(p.status),
                          trailing: IconButton(
                            icon: const Icon(Icons.delete_outline),
                            onPressed: () async {
                              await svc.deletePolicy(p.id);
                              setState(() => _reload++);
                            },
                          ),
                        ),
                    ],
                  ),
                ),
              ],
            ),
    );
  }
}

// ============================== Users ========================================

class UsersScreen extends StatefulWidget {
  const UsersScreen({super.key});
  @override
  State<UsersScreen> createState() => _UsersScreenState();
}

class _UsersScreenState extends State<UsersScreen> {
  final _c = _CrudControllers();
  int _reload = 0;

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('Users')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              _field('Email', _c('email')),
              _field('Password (min 8)', _c('password'), obscure: true),
              _field('Name', _c('name')),
              _field('Role', _c('role')),
              FilledButton(
                onPressed: () async {
                  try {
                    await mws.createUser(wid, {
                      'email': _c('email').text.trim(),
                      'password': _c('password').text,
                      'name': _c('name').text.trim(),
                      'role': _c('role').text.trim(),
                    });
                    if (mounted) _snack(context, 'User added');
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: const Text('Add user'),
              ),
              const Divider(),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.getUsers(wid),
                builder: (users) => Column(
                  children: [
                    for (final u in users)
                      ListTile(
                        dense: true,
                        title: Text('${u['name'] ?? u['email'] ?? ''}'),
                        subtitle: Text('${u['email'] ?? ''} · ${u['role'] ?? ''}'),
                        trailing: IconButton(
                          icon: const Icon(Icons.delete_outline),
                          onPressed: () async {
                            await mws.deleteUser(wid, '${u['id']}');
                            setState(() => _reload++);
                          },
                        ),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Chains =======================================

class ChainsScreen extends StatefulWidget {
  const ChainsScreen({super.key});
  @override
  State<ChainsScreen> createState() => _ChainsScreenState();
}

class _ChainsScreenState extends State<ChainsScreen> {
  final _c = _CrudControllers();
  int _reload = 0;
  String? _editingEvm;
  String? _editingNonEvm;

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('UserWallet Chains')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              const Text('Add EVM chain',
                  style: TextStyle(fontWeight: FontWeight.bold)),
              _field('Chain ID', _c('eid'), keyboard: TextInputType.number),
              _field('Name', _c('ename')),
              _field('RPC URL', _c('erpc')),
              _field('Symbol', _c('esym')),
              FilledButton(
                onPressed: () async {
                  try {
                    final editId = _editingEvm;
                    if (editId != null) {
                      await mws.updateUserEVMChain(wid, editId, {
                        'name': _c('ename').text.trim(),
                        'rpc_url': _c('erpc').text.trim(),
                        'symbol': _c('esym').text.trim(),
                      });
                      _editingEvm = null;
                      if (mounted) _snack(context, 'EVM chain updated');
                    } else {
                      await mws.addUserEVMChain(wid, {
                        'chain_id': int.tryParse(_c('eid').text) ?? 0,
                        'name': _c('ename').text.trim(),
                        'rpc_url': _c('erpc').text.trim(),
                        'symbol': _c('esym').text.trim(),
                      });
                      if (mounted) _snack(context, 'EVM chain added');
                    }
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: Text(_editingEvm != null ? 'Save EVM chain' : 'Add EVM chain'),
              ),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.listUserEVMChains(wid),
                builder: (chains) => Column(
                  children: [
                    for (final ch in chains)
                      ListTile(
                        dense: true,
                        title: Text('${ch['name']} (${ch['chain_id']})'),
                        subtitle: Text('${ch['rpc_url'] ?? ''}'),
                        trailing: Wrap(
                          children: [
                            IconButton(
                              icon: const Icon(Icons.edit_outlined),
                              onPressed: () {
                                _c('eid').text = '${ch['chain_id'] ?? ''}';
                                _c('ename').text = '${ch['name'] ?? ''}';
                                _c('erpc').text = '${ch['rpc_url'] ?? ''}';
                                _c('esym').text = '${ch['symbol'] ?? ''}';
                                setState(() => _editingEvm = '${ch['chain_id']}');
                              },
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline),
                              onPressed: () async {
                                await mws.removeUserEVMChain(
                                    wid, '${ch['chain_id']}');
                                setState(() => _reload++);
                              },
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
              const Divider(),
              const Text('Add non-EVM chain',
                  style: TextStyle(fontWeight: FontWeight.bold)),
              _field('Chain ID (SLIP-44)', _c('nid'), keyboard: TextInputType.number),
              _field('Name', _c('nname')),
              _field('Chain type', _c('ntype')),
              _field('RPC / node URL', _c('nrpc')),
              _field('Derivation path', _c('npath')),
              FilledButton(
                onPressed: () async {
                  try {
                    final editId = _editingNonEvm;
                    if (editId != null) {
                      await mws.updateUserNonEVMChain(wid, editId, {
                        'name': _c('nname').text.trim(),
                        'chain_type': _c('ntype').text.trim(),
                        'rpc_url': _c('nrpc').text.trim(),
                        'derivation_path': _c('npath').text.trim(),
                      });
                      _editingNonEvm = null;
                      if (mounted) _snack(context, 'Non-EVM chain updated');
                    } else {
                      await mws.addUserNonEVMChain(wid, {
                        'chain_id': int.tryParse(_c('nid').text) ?? 0,
                        'name': _c('nname').text.trim(),
                        'chain_type': _c('ntype').text.trim(),
                        'rpc_url': _c('nrpc').text.trim(),
                        'derivation_path': _c('npath').text.trim(),
                      });
                      if (mounted) _snack(context, 'Non-EVM chain added');
                    }
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: Text(_editingNonEvm != null ? 'Save non-EVM chain' : 'Add non-EVM chain'),
              ),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.listUserNonEVMChains(wid),
                builder: (chains) => Column(
                  children: [
                    for (final ch in chains)
                      ListTile(
                        dense: true,
                        title: Text('${ch['name']} (${ch['chain_type']})'),
                        subtitle: Text('id ${ch['chain_id']}'),
                        trailing: Wrap(
                          children: [
                            IconButton(
                              icon: const Icon(Icons.edit_outlined),
                              onPressed: () {
                                _c('nid').text = '${ch['chain_id'] ?? ''}';
                                _c('nname').text = '${ch['name'] ?? ''}';
                                _c('ntype').text = '${ch['chain_type'] ?? ''}';
                                _c('nrpc').text = '${ch['rpc_url'] ?? ''}';
                                _c('npath').text = '${ch['derivation_path'] ?? ''}';
                                setState(() => _editingNonEvm = '${ch['chain_id']}');
                              },
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline),
                              onPressed: () async {
                                await mws.removeUserNonEVMChain(
                                    wid, '${ch['chain_id']}');
                                setState(() => _reload++);
                              },
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Tokens =======================================

class TokensScreen extends StatefulWidget {
  const TokensScreen({super.key});
  @override
  State<TokensScreen> createState() => _TokensScreenState();
}

class _TokensScreenState extends State<TokensScreen> {
  final _c = _CrudControllers();
  int _reload = 0;
  String? _editingToken;

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('UserWallet Tokens')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              _field('Chain ID', _c('cid'), keyboard: TextInputType.number),
              _field('Symbol', _c('sym')),
              _field('Name', _c('name')),
              _field('Contract address', _c('addr')),
              _field('Decimals', _c('dec'), keyboard: TextInputType.number),
              FilledButton(
                onPressed: () async {
                  try {
                    final editId = _editingToken;
                    if (editId != null) {
                      await mws.updateUserToken(wid, editId, {
                        'symbol': _c('sym').text.trim(),
                        'name': _c('name').text.trim(),
                        'decimals': int.tryParse(_c('dec').text) ?? 18,
                      });
                      _editingToken = null;
                      if (mounted) _snack(context, 'Token updated');
                    } else {
                      await mws.addUserToken(wid, {
                        'chain_id': int.tryParse(_c('cid').text) ?? 0,
                        'symbol': _c('sym').text.trim(),
                        'name': _c('name').text.trim(),
                        'contract_address': _c('addr').text.trim(),
                        'decimals': int.tryParse(_c('dec').text) ?? 18,
                      });
                      if (mounted) _snack(context, 'Token added');
                    }
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: Text(_editingToken != null ? 'Save token' : 'Add token'),
              ),
              const Divider(),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.listUserTokens(wid),
                builder: (tokens) => Column(
                  children: [
                    for (final t in tokens)
                      ListTile(
                        dense: true,
                        title: Text('${t['symbol']} — ${t['name']}'),
                        subtitle: Text(
                            'chain ${t['chain_id']} · ${t['contract_address'] ?? 'native'}'),
                        trailing: Wrap(
                          children: [
                            IconButton(
                              icon: const Icon(Icons.edit_outlined),
                              onPressed: () {
                                _c('cid').text = '${t['chain_id'] ?? ''}';
                                _c('sym').text = '${t['symbol'] ?? ''}';
                                _c('name').text = '${t['name'] ?? ''}';
                                _c('addr').text = '${t['contract_address'] ?? ''}';
                                _c('dec').text = '${t['decimals'] ?? 18}';
                                setState(() => _editingToken = '${t['id']}');
                              },
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline),
                              onPressed: () async {
                                await mws.removeUserToken(wid, '${t['id']}');
                                setState(() => _reload++);
                              },
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Feature Flags ================================

class FlagsScreen extends StatefulWidget {
  const FlagsScreen({super.key});
  @override
  State<FlagsScreen> createState() => _FlagsScreenState();
}

class _FlagsScreenState extends State<FlagsScreen> {
  final _c = _CrudControllers();
  int _reload = 0;

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('Feature Flags')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              _field('Flag key', _c('key')),
              FilledButton(
                onPressed: () async {
                  try {
                    await mws.addFeatureFlag(wid, {
                      'flag_key': _c('key').text.trim(),
                      'is_enabled': true,
                    });
                    if (mounted) _snack(context, 'Flag added');
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: const Text('Add flag'),
              ),
              const Divider(),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.listFeatureFlags(wid),
                builder: (flags) => Column(
                  children: [
                    for (final f in flags)
                      ListTile(
                        dense: true,
                        title: Text('${f['flag_key']}'),
                        subtitle: Text('${f['description'] ?? ''}'),
                        trailing: Wrap(
                          children: [
                            IconButton(
                              icon: Icon((f['is_enabled'] == true)
                                  ? Icons.toggle_on
                                  : Icons.toggle_off),
                              onPressed: () async {
                                await mws.updateFeatureFlag(
                                    wid, '${f['id']}',
                                    {'is_enabled': !(f['is_enabled'] == true)});
                                setState(() => _reload++);
                              },
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline),
                              onPressed: () async {
                                await mws.removeFeatureFlag(wid, '${f['id']}');
                                setState(() => _reload++);
                              },
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Webhooks & Alerts ============================

class WebhooksScreen extends StatefulWidget {
  const WebhooksScreen({super.key});
  @override
  State<WebhooksScreen> createState() => _WebhooksScreenState();
}

class _WebhooksScreenState extends State<WebhooksScreen> {
  final _c = _CrudControllers();
  int _reload = 0;

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('Webhooks & Alerts')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              _field('Webhook name', _c('wname')),
              _field('URL', _c('wurl')),
              _field('Events (comma-separated)', _c('wevents')),
              FilledButton(
                onPressed: () async {
                  try {
                    await mws.createWebhook(wid, {
                      'name': _c('wname').text.trim(),
                      'url': _c('wurl').text.trim(),
                      'events': _c('wevents')
                          .text
                          .split(',')
                          .map((s) => s.trim())
                          .where((s) => s.isNotEmpty)
                          .toList(),
                    });
                    if (mounted) _snack(context, 'Webhook added');
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: const Text('Add webhook'),
              ),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.getWebhooks(wid),
                builder: (hooks) => Column(
                  children: [
                    for (final w in hooks)
                      ListTile(
                        dense: true,
                        title: Text('${w['name'] ?? w['url']}'),
                        subtitle: Text('${w['url'] ?? ''}'),
                        trailing: IconButton(
                          icon: const Icon(Icons.delete_outline),
                          onPressed: () async {
                            await mws.deleteWebhook(wid, '${w['id']}');
                            setState(() => _reload++);
                          },
                        ),
                      ),
                  ],
                ),
              ),
              const Divider(),
              const Text('Send notification',
                  style: TextStyle(fontWeight: FontWeight.bold)),
              _field('Title', _c('ntitle')),
              _field('Message', _c('nmsg')),
              FilledButton(
                onPressed: () async {
                  try {
                    await mws.createNotification(wid, {
                      'notification_type': 'alert',
                      'title': _c('ntitle').text.trim(),
                      'message': _c('nmsg').text.trim(),
                    });
                    if (mounted) _snack(context, 'Notification sent');
                    setState(() => _reload++);
                  } catch (e) {
                    if (mounted) _snack(context, '$e');
                  }
                },
                child: const Text('Send'),
              ),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.getNotifications(wid),
                builder: (notifs) => Column(
                  children: [
                    for (final n in notifs)
                      ListTile(
                        dense: true,
                        title: Text('${n['title'] ?? ''}'),
                        subtitle: Text('${n['message'] ?? ''}'),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Audit ========================================

class AuditScreen extends StatefulWidget {
  const AuditScreen({super.key});
  @override
  State<AuditScreen> createState() => _AuditScreenState();
}

class _AuditScreenState extends State<AuditScreen> {
  AuditService? _svc;

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    final mws = context.read<MasterWalletService>();
    final wid = await _firstWalletId(mws);
    if (!mounted) return;
    setState(() {
      if (wid != null) {
        _svc = AuditService(
            masterWalletId: wid, token: context.read<AuthService>().token);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final svc = _svc;
    return Scaffold(
      appBar: AppBar(title: const Text('Audit Log')),
      body: svc == null
          ? const Center(child: CircularProgressIndicator())
          : AsyncSection<List<AuditLog>>(
              future: svc.getLogs(),
              builder: (logs) => ListView(
                children: [
                  for (final l in logs)
                    ListTile(
                      dense: true,
                      title: Text(l.action),
                      subtitle: Text(
                          '${l.userName.isNotEmpty ? l.userName : l.userId} · ${l.entityType} ${l.entityId}'),
                    ),
                ],
              ),
            ),
    );
  }
}

// ============================== Analytics ====================================

class AnalyticsScreen extends StatelessWidget {
  const AnalyticsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    return Scaffold(
      appBar: AppBar(title: const Text('Analytics')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              AsyncSection<VolumeAnalytics>(
                future: mws.getVolumeAnalytics(wid),
                builder: (v) => Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Total volume: ${v.totalVolume}'),
                    Text('Transactions: ${v.transactionCount}'),
                  ],
                ),
              ),
              const Divider(),
              AsyncSection<List<Map<String, dynamic>>>(
                future: mws.getAnalyticsWallets(wid),
                builder: (wallets) => Column(
                  children: [
                    for (final w in wallets.take(20))
                      ListTile(
                        dense: true,
                        title: Text('${w['name'] ?? w['address'] ?? ''}'),
                        subtitle: Text('${w['tx_count'] ?? ''}'),
                      ),
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ============================== Passkeys =====================================

class PasskeysScreen extends StatefulWidget {
  const PasskeysScreen({super.key});
  @override
  State<PasskeysScreen> createState() => _PasskeysScreenState();
}

class _PasskeysScreenState extends State<PasskeysScreen> {
  int _reload = 0;

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    _reload;
    return Scaffold(
      appBar: AppBar(title: const Text('Passkeys')),
      body: FutureBuilder<String?>(
        future: _firstWalletId(mws),
        builder: (context, wsnap) {
          final wid = wsnap.data;
          if (wid == null) return const Center(child: Text('No master wallet.'));
          return AsyncSection<List<PasskeyCredential>>(
            future: mws.listPasskeys(wid),
            builder: (keys) => ListView(
              children: [
                for (final p in keys)
                  ListTile(
                    dense: true,
                    title: Text(p.label.isNotEmpty ? p.label : 'Passkey'),
                    subtitle: Text(p.credentialId.length > 24
                        ? p.credentialId.substring(0, 24)
                        : p.credentialId),
                    trailing: IconButton(
                      icon: const Icon(Icons.delete_outline),
                      onPressed: () async {
                        await mws.deletePasskey(wid, p.credentialId);
                        setState(() => _reload++);
                      },
                    ),
                  ),
              ],
            ),
          );
        },
      ),
    );
  }
}

// ============================== Withdraw =====================================

class WithdrawScreen extends StatefulWidget {
  const WithdrawScreen({super.key});
  @override
  State<WithdrawScreen> createState() => _WithdrawScreenState();
}

class _WithdrawScreenState extends State<WithdrawScreen> {
  final _c = _CrudControllers();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final mws = context.read<MasterWalletService>();
    return Scaffold(
      appBar: AppBar(title: const Text('Withdraw')),
      body: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          const Text(
            'Funds never move without TigerWallet SuperAdmin two-party '
            'co-sign. This only files the request.',
          ),
          _field('Destination address', _c('to')),
          _field('Amount (wei)', _c('amount'), keyboard: TextInputType.number),
          _field('Currency (e.g. ETH)', _c('currency')),
          _field('Chain ID', _c('chain'), keyboard: TextInputType.number),
          FilledButton(
            onPressed: () async {
              try {
                final wid = await _firstWalletId(mws);
                if (wid == null) {
                  if (mounted) _snack(context, 'No master wallet.');
                  return;
                }
                final r = await mws.requestWithdrawal(
                  wid,
                  _c('to').text.trim(),
                  _c('amount').text.trim(),
                  currency: _c('currency').text.trim(),
                  chainId: int.tryParse(_c('chain').text),
                );
                if (mounted) {
                  _snack(context,
                      'Withdrawal request: ${r?.withdrawalId ?? 'failed'} (${r?.status ?? ''})');
                }
              } catch (e) {
                if (mounted) _snack(context, '$e');
              }
            },
            child: const Text('Request withdrawal'),
          ),
        ],
      ),
    );
  }
}
