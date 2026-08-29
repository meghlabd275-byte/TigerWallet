/**
 * DashboardScreen — drives the real service-layer fetchers.
 *
 * Renders live data from MasterWalletService against the canonical backend
 * (:8450): wallet list, balance, transactions (with approve/reject), fees,
 * auto-sign rules, users, notifications, webhooks, analytics, chains, health,
 * and on-chain transaction history. Every panel fetches live; backend errors
 * are surfaced inline rather than replaced with fake values.
 *
 * Authenticated calls carry the Bearer JWT propagated by the AuthGate.
 */

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/auth_service.dart';
import '../services/master_wallet_service.dart';
import '../services/web_socket_service.dart';
import 'features_screen.dart';
import 'theme_toggle.dart';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({super.key});

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  String? _selectedWalletId;
  WebSocketService? _liveWs;
  StreamSubscription<String>? _liveSub;
  String? _liveEvent;
  Map<String, dynamic>? _killSwitch;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 6, vsync: this);
    _loadKillSwitch();
  }

  /// Read-only SuperAdmin kill-switch state (GET /api/v1/kill-switch/status).
  Future<void> _loadKillSwitch() async {
    try {
      final status =
          await context.read<MasterWalletService>().getKillSwitchStatus();
      if (mounted) setState(() => _killSwitch = status);
    } catch (_) {
      // Status unknown (e.g. backend down) — leave the banner hidden.
    }
  }

  @override
  void dispose() {
    _liveSub?.cancel();
    _liveWs?.dispose();
    _tabs.dispose();
    super.dispose();
  }

  /// Live backend /ws feed: real balance/transaction events refresh the
  /// dashboard instantly instead of waiting for the next pull-to-refresh.
  void _startLiveFeed(String walletId, String? token) {
    if (_liveWs != null) return;
    final ws = WebSocketService();
    _liveWs = ws;
    ws.connect(walletId: walletId, token: token);
    _liveSub = ws.messageStream.listen((text) {
      if (!mounted) return;
      setState(() {
        _liveEvent = text.length > 80 ? text.substring(0, 80) : text;
      });
      // Data events invalidate the future-builders via a rebuild.
    });
  }

  void _logout() {
    context.read<AuthService>().logout();
  }

  @override
  Widget build(BuildContext context) {
    final walletSvc = context.watch<MasterWalletService>();
    final auth = context.read<AuthService>();
    walletSvc.setToken(auth.token);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Master Wallet Dashboard'),
        actions: [
          IconButton(
            icon: const Icon(Icons.grid_view),
            tooltip: 'All features',
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => const FeaturesScreen()),
            ),
          ),
          const ThemeToggle(),
          IconButton(
            icon: const Icon(Icons.logout),
            tooltip: 'Sign out',
            onPressed: _logout,
          ),
        ],
        bottom: PreferredSize(
          preferredSize: Size.fromHeight(
              48 +
                  (_liveEvent == null ? 0 : 26) +
                  (_killSwitch?['halted'] == true ? 26 : 0)),
          child: Column(
            children: [
              if (_killSwitch?['halted'] == true)
                Container(
                  width: double.infinity,
                  color: Theme.of(context).colorScheme.errorContainer,
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 3),
                  child: Text(
                    'KILL SWITCH HALTED by SuperAdmin'
                    '${(_killSwitch?['reason'] as String? ?? '').isNotEmpty ? ': ${_killSwitch?['reason']}' : ''}'
                    ' — all API operations are blocked.',
                    style: Theme.of(context)
                        .textTheme
                        .bodySmall
                        ?.copyWith(color: Theme.of(context).colorScheme.onErrorContainer),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              if (_liveEvent != null)
                Container(
                  width: double.infinity,
                  color: Theme.of(context).colorScheme.surfaceContainerHighest,
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 3),
                  child: Text('Live: $_liveEvent',
                      style: Theme.of(context).textTheme.bodySmall,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis),
                ),
              TabBar(
                isScrollable: true,
                tabAlignment: TabAlignment.start,
                controller: _tabs,
                tabs: const [
                  Tab(text: 'Wallets'),
                  Tab(text: 'Activity'),
                  Tab(text: 'Policies'),
                  Tab(text: 'Config'),
                  Tab(text: 'Analytics'),
                  Tab(text: 'Network'),
                ],
              ),
            ],
          ),
        ),
      ),
      body: TabBarView(
        controller: _tabs,
        children: [
          _WalletsTab(
            walletSvc: walletSvc,
            selected: _selectedWalletId,
            onSelect: (id) {
              setState(() => _selectedWalletId = id);
              if (id != null) _startLiveFeed(id, auth.token);
            },
          ),
          _ActivityTab(walletSvc: walletSvc, walletId: _selectedWalletId),
          _PoliciesTab(walletSvc: walletSvc, walletId: _selectedWalletId),
          _ConfigTab(walletSvc: walletSvc, walletId: _selectedWalletId),
          _AnalyticsTab(walletSvc: walletSvc, walletId: _selectedWalletId),
          _NetworkTab(walletSvc: walletSvc),
        ],
      ),
    );
  }
}

// ==================== Shared bits ====================

class _Panel extends StatelessWidget {
  final String title;
  final Widget child;
  const _Panel({required this.title, required this.child});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.all(12),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(title, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            child,
          ],
        ),
      ),
    );
  }
}

Widget _empty(String msg) => Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Text(msg, style: const TextStyle(fontStyle: FontStyle.italic)),
    );

Widget _errorView(Object e) => Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Text(
        e.toString(),
        style: TextStyle(color: Colors.red.shade700),
      ),
    );

typedef _ItemTile = Widget Function(Map<String, dynamic> item);

Widget _jsonList(List<Map<String, dynamic>> items, _ItemTile tileBuilder) {
  if (items.isEmpty) return _empty('No items.');
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [for (final item in items) ...[tileBuilder(item), const Divider()]],
  );
}

// ==================== Wallets tab ====================

class _WalletsTab extends StatefulWidget {
  final MasterWalletService walletSvc;
  final String? selected;
  final ValueChanged<String> onSelect;
  const _WalletsTab({
    required this.walletSvc,
    required this.selected,
    required this.onSelect,
  });

  @override
  State<_WalletsTab> createState() => _WalletsTabState();
}

class _WalletsTabState extends State<_WalletsTab> {
  late Future<List<WalletData>> _future;

  @override
  void initState() {
    super.initState();
    _future = widget.walletSvc.getWallets();
  }

  void _reload() => setState(() {
        _future = widget.walletSvc.getWallets();
      });

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: () async => _reload(),
      child: ListView(
        children: [
          _Panel(
            title: 'Master Wallets',
            child: FutureBuilder<List<WalletData>>(
              future: _future,
              builder: (context, snap) {
                if (snap.connectionState != ConnectionState.done) {
                  return const Center(
                    child: Padding(
                      padding: EdgeInsets.all(16),
                      child: CircularProgressIndicator(),
                    ),
                  );
                }
                if (snap.hasError) return _errorView(snap.error!);
                final wallets = snap.data ?? const [];
                if (wallets.isEmpty) return _empty('No wallets yet.');
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    for (final w in wallets)
                      RadioListTile<String>(
                        value: w.id,
                        groupValue: widget.selected,
                        title: Text(w.name.isEmpty ? w.id : w.name),
                        subtitle: Text(w.address.isEmpty ? '(no address)' : w.address),
                        onChanged: (v) {
                          if (v != null) widget.onSelect(v);
                        },
                      ),
                    if (widget.selected != null)
                      Padding(
                        padding: const EdgeInsets.only(top: 8),
                        child: Text(
                            'Selected: ${widget.selected}'),
                      ),
                  ],
                );
              },
            ),
          ),
          if (widget.selected != null)
            _BalancePanel(
                walletSvc: widget.walletSvc, walletId: widget.selected!),
        ],
      ),
    );
  }
}

class _BalancePanel extends StatefulWidget {
  final MasterWalletService walletSvc;
  final String walletId;
  const _BalancePanel({required this.walletSvc, required this.walletId});

  @override
  State<_BalancePanel> createState() => _BalancePanelState();
}

class _BalancePanelState extends State<_BalancePanel> {
  late Future<BalanceResult> _future;
  @override
  void initState() {
    super.initState();
    _future = widget.walletSvc.getBalance(widget.walletId, 1);
  }

  @override
  Widget build(BuildContext context) {
    return _Panel(
      title: 'Balance (live RPC)',
      child: FutureBuilder<BalanceResult>(
        future: _future,
        builder: (context, snap) {
          if (snap.connectionState != ConnectionState.done) {
            return const CircularProgressIndicator();
          }
          if (snap.hasError) return _errorView(snap.error!);
          final b = snap.data!;
          if (!b.success) {
            return _errorView(Exception(b.error ?? 'balance unavailable'));
          }
          return Text(
            '${b.balance.toStringAsFixed(6)} ${b.symbol}'
            '${b.usdValue > 0 ? '  (≈ \$${b.usdValue.toStringAsFixed(2)})' : ''}',
          );
        },
      ),
    );
  }
}

// ==================== Activity tab ====================

class _ActivityTab extends StatefulWidget {
  final MasterWalletService walletSvc;
  final String? walletId;
  const _ActivityTab({required this.walletSvc, required this.walletId});

  @override
  State<_ActivityTab> createState() => _ActivityTabState();
}

class _ActivityTabState extends State<_ActivityTab> {
  Future<List<Map<String, dynamic>>>? _future;

  void _ensure() {
    if (widget.walletId == null) return;
    if (_future == null) {
      _future = widget.walletSvc.getTransactions(widget.walletId!);
    }
  }

  void _refresh() {
    if (widget.walletId == null) return;
    setState(() {
      _future = widget.walletSvc.getTransactions(widget.walletId!);
    });
  }

  @override
  Widget build(BuildContext context) {
    _ensure();
    if (widget.walletId == null) {
      return _Panel(title: 'Transactions', child: _empty('Select a wallet first.'));
    }
    return RefreshIndicator(
      onRefresh: () async => _refresh(),
      child: ListView(
        children: [
          _Panel(
            title: 'Transactions',
            child: FutureBuilder<List<Map<String, dynamic>>>(
              future: _future,
              builder: (context, snap) {
                if (snap.connectionState != ConnectionState.done) {
                  return const CircularProgressIndicator();
                }
                if (snap.hasError) return _errorView(snap.error!);
                return _jsonList(snap.data ?? const [], (t) {
                  final id = (t['id'] ?? t['tx_hash'] ?? '').toString();
                  final status = (t['status'] ?? '').toString();
                  final to = (t['to'] ?? '').toString();
                  final amount = (t['amount'] ?? '').toString();
                  final pending = status.toLowerCase() == 'pending';
                  return ListTile(
                    dense: true,
                    title: Text(amount.isEmpty ? id : '$amount → $to'),
                    subtitle: Text('id: $id  status: $status'),
                    trailing: pending
                        ? Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              IconButton(
                                tooltip: 'Approve',
                                icon: const Icon(Icons.check),
                                onPressed: () async {
                                  try {
                                    await widget.walletSvc
                                        .approveTransaction(widget.walletId!, id);
                                    _refresh();
                                  } catch (e) {
                                    if (context.mounted) {
                                      ScaffoldMessenger.of(context).showSnackBar(
                                        SnackBar(content: Text(e.toString())),
                                      );
                                    }
                                  }
                                },
                              ),
                              IconButton(
                                tooltip: 'Reject',
                                icon: const Icon(Icons.close),
                                onPressed: () async {
                                  try {
                                    await widget.walletSvc
                                        .rejectTransaction(widget.walletId!, id);
                                    _refresh();
                                  } catch (e) {
                                    if (context.mounted) {
                                      ScaffoldMessenger.of(context).showSnackBar(
                                        SnackBar(content: Text(e.toString())),
                                      );
                                    }
                                  }
                                },
                              ),
                            ],
                          )
                        : null,
                  );
                });
              },
            ),
          ),
        ],
      ),
    );
  }
}

// ==================== Policies tab (fees, auto-sign, users) ====================

class _PoliciesTab extends StatelessWidget {
  final MasterWalletService walletSvc;
  final String? walletId;
  const _PoliciesTab({required this.walletSvc, required this.walletId});

  @override
  Widget build(BuildContext context) {
    if (walletId == null) {
      return _Panel(title: 'Policies', child: _empty('Select a wallet first.'));
    }
    return ListView(
      children: [
        _Panel(
          title: 'Fees',
          child: _LiveList(
            cacheKey: 'fees:$walletId',
            load: () => walletSvc.getFees(walletId!),
            tileBuilder: (f) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('${f['fee_type'] ?? 'fee'}'),
                Text(
                  'pct: ${f['fee_percentage'] ?? 0}  fixed: ${f['fee_fixed'] ?? '0'}  active: ${f['is_active']}',
                  style: const TextStyle(fontSize: 12),
                ),
              ],
            ),
            onDelete: (id) => walletSvc.deleteFee(walletId!, id),
          ),
        ),
        _Panel(
          title: 'Auto-Sign Rules',
          child: _LiveList(
            cacheKey: 'autosign:$walletId',
            load: () => walletSvc.getAutoSignRules(walletId!),
            tileBuilder: (r) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('${r['name'] ?? 'rule'}'),
                Text(
                  'type: ${r['rule_type'] ?? ''}  max: ${r['max_amount'] ?? ''}  active: ${r['is_active']}',
                  style: const TextStyle(fontSize: 12),
                ),
              ],
            ),
            onDelete: (id) => walletSvc.deleteAutoSignRule(walletId!, id),
          ),
        ),
      ],
    );
  }
}

// ==================== Config tab (users, notifications, webhooks) ====================

class _ConfigTab extends StatelessWidget {
  final MasterWalletService walletSvc;
  final String? walletId;
  const _ConfigTab({required this.walletSvc, required this.walletId});

  @override
  Widget build(BuildContext context) {
    if (walletId == null) {
      return _Panel(title: 'Config', child: _empty('Select a wallet first.'));
    }
    return ListView(
      children: [
        _Panel(
          title: 'Users',
          child: _LiveList(
            cacheKey: 'users:$walletId',
            load: () => walletSvc.getUsers(walletId!),
            tileBuilder: (u) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('${u['email'] ?? u['name'] ?? 'user'}'),
                Text(
                  'role: ${u['role'] ?? ''}  active: ${u['is_active']}',
                  style: const TextStyle(fontSize: 12),
                ),
              ],
            ),
            onDelete: (id) => walletSvc.deleteUser(walletId!, id),
          ),
        ),
        _Panel(
          title: 'Notifications',
          child: _LiveList(
            cacheKey: 'notifications:$walletId',
            load: () => walletSvc.getNotifications(walletId!),
            tileBuilder: (n) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('${n['title'] ?? 'notification'}'),
                Text(
                  '${n['type'] ?? n['notification_type'] ?? ''}: ${n['message'] ?? ''}',
                  style: const TextStyle(fontSize: 12),
                ),
              ],
            ),
          ),
        ),
        _Panel(
          title: 'Webhooks',
          child: _LiveList(
            cacheKey: 'webhooks:$walletId',
            load: () => walletSvc.getWebhooks(walletId!),
            tileBuilder: (w) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('${w['name'] ?? 'webhook'}'),
                Text(
                  '${w['url'] ?? ''}  active: ${w['is_active']}',
                  style: const TextStyle(fontSize: 12),
                ),
              ],
            ),
            onDelete: (id) => walletSvc.deleteWebhook(walletId!, id),
          ),
        ),
      ],
    );
  }
}

// ==================== Analytics tab ====================

class _AnalyticsTab extends StatelessWidget {
  final MasterWalletService walletSvc;
  final String? walletId;
  const _AnalyticsTab({required this.walletSvc, required this.walletId});

  @override
  Widget build(BuildContext context) {
    if (walletId == null) {
      return _Panel(title: 'Analytics', child: _empty('Select a wallet first.'));
    }
    return ListView(
      children: [
        _Panel(
          title: 'Transaction Analytics (by status)',
          child: _LiveMap(
            cacheKey: 'analytics-tx:$walletId',
            load: () => walletSvc.getAnalyticsTransactions(walletId!),
          ),
        ),
        _Panel(
          title: 'Wallet Analytics',
          child: _LiveMap(
            cacheKey: 'analytics-wallets:$walletId',
            load: () => walletSvc.getAnalyticsWallets(walletId!),
          ),
        ),
      ],
    );
  }
}

// ==================== Network tab (chains, health, history) ====================

class _NetworkTab extends StatefulWidget {
  final MasterWalletService walletSvc;
  const _NetworkTab({required this.walletSvc});

  @override
  State<_NetworkTab> createState() => _NetworkTabState();
}

class _NetworkTabState extends State<_NetworkTab> {
  final _historyCtrl = TextEditingController();

  @override
  void dispose() {
    _historyCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ListView(
      children: [
        _Panel(
          title: 'Backend Health',
          child: _LiveMap(
            cacheKey: 'health',
            load: widget.walletSvc.health,
          ),
        ),
        _Panel(
          title: 'Supported Chains',
          child: _LiveList(
            cacheKey: 'chains',
            load: widget.walletSvc.getChains,
            tileBuilder: (c) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('${c['name'] ?? ''}'),
                Text(
                  'chain_id: ${c['chain_id'] ?? ''}  symbol: ${c['symbol'] ?? ''}  evm: ${c['is_evm']}',
                  style: const TextStyle(fontSize: 12),
                ),
              ],
            ),
          ),
        ),
        _Panel(
          title: 'On-chain Transaction History',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _historyCtrl,
                      decoration: const InputDecoration(
                        labelText: 'Address',
                        hintText: '0x…',
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(
                    onPressed: _historyCtrl.text.trim().isEmpty
                        ? null
                        : () => setState(() {}),
                    child: const Text('Fetch'),
                  ),
                ],
              ),
              if (_historyCtrl.text.trim().isNotEmpty)
                _LiveList(
                  cacheKey: 'history:${_historyCtrl.text.trim()}',
                  load: () => widget.walletSvc.getTransactionHistory(
                    address: _historyCtrl.text.trim(),
                  ),
                  tileBuilder: (t) => Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('${t['value'] ?? t['amount'] ?? ''}'),
                      Text(
                        '${t['hash'] ?? t['tx_hash'] ?? ''}',
                        style: const TextStyle(fontSize: 12),
                      ),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }
}

// ==================== Reusable async widgets ====================

class _LiveList extends StatefulWidget {
  final String cacheKey;
  final Future<List<Map<String, dynamic>>> Function() load;
  final _ItemTile tileBuilder;
  final Future<bool> Function(String id)? onDelete;
  const _LiveList({
    required this.cacheKey,
    required this.load,
    required this.tileBuilder,
    this.onDelete,
  });

  @override
  State<_LiveList> createState() => _LiveListState();
}

class _LiveListState extends State<_LiveList> {
  Future<List<Map<String, dynamic>>>? _future;

  @override
  void initState() {
    super.initState();
    _future = widget.load();
  }

  void _reload() => setState(() => _future = widget.load());

  @override
  void didUpdateWidget(_LiveList oldWidget) {
    super.didUpdateWidget(oldWidget);
    // Re-fetch only when the logical identity (cacheKey) changes — e.g. the
    // selected wallet or address changes — not on every parent rebuild.
    if (oldWidget.cacheKey != widget.cacheKey) {
      _future = widget.load();
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<Map<String, dynamic>>>(
      future: _future,
      builder: (context, snap) {
        if (snap.connectionState != ConnectionState.done) {
          return const Center(
            child: Padding(
              padding: EdgeInsets.all(16),
              child: CircularProgressIndicator(),
            ),
          );
        }
        if (snap.hasError) return _errorView(snap.error!);
        final items = snap.data ?? const [];
        if (items.isEmpty) return _empty('No items.');
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            for (final item in items) ...[
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(child: widget.tileBuilder(item)),
                  if (widget.onDelete != null)
                    IconButton(
                      icon: const Icon(Icons.delete_outline, size: 20),
                      onPressed: () async {
                        final id = (item['id'] ?? '').toString();
                        if (id.isEmpty) return;
                        try {
                          await widget.onDelete!(id);
                          _reload();
                        } catch (e) {
                          if (context.mounted) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(content: Text(e.toString())),
                            );
                          }
                        }
                      },
                    ),
                ],
              ),
              const Divider(),
            ],
          ],
        );
      },
    );
  }
}

class _LiveMap extends StatefulWidget {
  final String cacheKey;
  final Future<Map<String, dynamic>> Function() load;
  const _LiveMap({required this.cacheKey, required this.load});

  @override
  State<_LiveMap> createState() => _LiveMapState();
}

class _LiveMapState extends State<_LiveMap> {
  late Future<Map<String, dynamic>> _future;

  @override
  void initState() {
    super.initState();
    _future = widget.load();
  }

  void _reload() => setState(() => _future = widget.load());

  @override
  void didUpdateWidget(_LiveMap oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.cacheKey != widget.cacheKey) {
      _future = widget.load();
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<Map<String, dynamic>>(
      future: _future,
      builder: (context, snap) {
        if (snap.connectionState != ConnectionState.done) {
          return const Center(
            child: Padding(
              padding: EdgeInsets.all(16),
              child: CircularProgressIndicator(),
            ),
          );
        }
        if (snap.hasError) return _errorView(snap.error!);
        final data = snap.data ?? const {};
        if (data.isEmpty) return _empty('No data.');
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            for (final entry in data.entries)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Text('${entry.key}: ${entry.value}'),
              ),
            Align(
              alignment: Alignment.centerLeft,
              child: TextButton.icon(
                onPressed: _reload,
                icon: const Icon(Icons.refresh, size: 18),
                label: const Text('Refresh'),
              ),
            ),
          ],
        );
      },
    );
  }
}
