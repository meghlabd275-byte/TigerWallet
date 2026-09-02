/**
 * TransactionsScreen — on-chain transaction history for the user's wallets
 * with a tap-through to the real on-chain receipt (GET
 * /api/v1/transactions/:txHash?chain_id=N). Fail-closed: backend errors and
 * empty states are surfaced, nothing is fabricated.
 */

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../services/user_wallet.dart';

class TransactionsScreen extends StatefulWidget {
  const TransactionsScreen({super.key});

  @override
  State<TransactionsScreen> createState() => _TransactionsScreenState();
}

class _TransactionsScreenState extends State<TransactionsScreen> {
  List<dynamic> _wallets = [];
  Map<String, dynamic>? _selected;
  List<dynamic> _txs = [];
  String? _error;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _loadWallets();
  }

  Future<void> _loadWallets() async {
    final api = context.read<UserWalletService>();
    try {
      final res = await api.listWallets();
      final w = res?['wallets'] ?? res?['data'];
      final list = w is List ? w : <dynamic>[];
      if (!mounted) return;
      setState(() {
        _wallets = list;
        _selected = list.isNotEmpty ? list.first as Map<String, dynamic> : null;
        _loading = false;
      });
      if (_selected != null) _loadTxs();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  Future<void> _loadTxs() async {
    final api = context.read<UserWalletService>();
    final w = _selected;
    if (w == null) return;
    setState(() => _loading = true);
    try {
      final chainId = (w['chain_id'] as num?)?.toInt() ?? 1;
      final res = await api.getTransactions('${w['address']}', chainId);
      final list = res?['transactions'] ?? res?['data'];
      if (!mounted) return;
      setState(() {
        _txs = list is List ? list : [];
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

  Future<void> _showReceipt(Map<String, dynamic> tx) async {
    final api = context.read<UserWalletService>();
    final hash = '${tx['hash'] ?? ''}';
    if (hash.isEmpty) return;
    final chainId = (_selected?['chain_id'] as num?)?.toInt() ?? 1;
    try {
      final res = await api.getTransactionReceipt(hash, chainId);
      if (!mounted) return;
      final status = res?['status']?.toString() ?? 'pending';
      final block = res?['block_number']?.toString() ?? res?['blockNumber']?.toString() ?? '—';
      showDialog<void>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: const Text('On-chain receipt'),
          content: SingleChildScrollView(
            child: Text(
              'Hash: $hash\nStatus: $status\nBlock: $block\n\n${res ?? {}}',
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () {
                Clipboard.setData(ClipboardData(text: hash));
                Navigator.of(ctx).pop();
              },
              child: const Text('Copy hash'),
            ),
            TextButton(onPressed: () => Navigator.of(ctx).pop(), child: const Text('Close')),
          ],
        ),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('Receipt unavailable: $e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Transactions')),
      body: Column(children: [
        if (_wallets.isNotEmpty)
          Padding(
            padding: const EdgeInsets.all(12),
            child: DropdownButton<String>(
              isExpanded: true,
              value: '${_selected?['address']}',
              items: _wallets
                  .map((w) => DropdownMenuItem(
                        value: '${w['address']}',
                        child: Text('${w['label'] ?? w['address']} (chain ${w['chain_id'] ?? ''})',
                            overflow: TextOverflow.ellipsis),
                      ))
                  .toList(),
              onChanged: (addr) {
                setState(() {
                  _selected = _wallets.firstWhere((w) => '${w['address']}' == addr)
                      as Map<String, dynamic>;
                });
                _loadTxs();
              },
            ),
          ),
        if (_error != null)
          Padding(padding: const EdgeInsets.all(12), child: Text(_error!)),
        Expanded(
          child: _loading
              ? const Center(child: CircularProgressIndicator())
              : _txs.isEmpty
                  ? const Center(child: Text('No transactions yet'))
                  : RefreshIndicator(
                      onRefresh: _loadTxs,
                      child: ListView.builder(
                        itemCount: _txs.length,
                        itemBuilder: (_, i) {
                          final tx = _txs[i];
                          return ListTile(
                            dense: true,
                            title: Text('${tx['hash'] ?? ''}',
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(fontFamily: 'monospace', fontSize: 12)),
                            subtitle: Text('to ${tx['to'] ?? ''} — ${tx['value'] ?? ''}'),
                            trailing: const Icon(Icons.receipt_long, size: 18),
                            onTap: () => _showReceipt(tx is Map<String, dynamic>
                                ? tx
                                : Map<String, dynamic>.from(tx as Map)),
                          );
                        },
                      ),
                    ),
        ),
      ]),
    );
  }
}
