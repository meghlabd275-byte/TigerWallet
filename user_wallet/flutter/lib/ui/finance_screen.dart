import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../services/user_wallet.dart';

/// Wallet & finance plane: multi-chain ledger accounts, deterministic deposit
/// addresses with copy, signed withdrawals, instant convert, KYC-gated
/// internal transfers, escrowed P2P marketplace, full ledger history.
class FinanceScreen extends StatefulWidget {
  final UserWalletService api;
  const FinanceScreen({super.key, required this.api});

  @override
  State<FinanceScreen> createState() => _FinanceScreenState();
}

class _FinanceScreenState extends State<FinanceScreen> {
  static const assets = ['BTC', 'ETH', 'USDT', 'USDC', 'BNB', 'SOL', 'TRX', 'MATIC', 'LTC', 'DOGE'];
  String currency = 'BTC';
  final toCtrl = TextEditingController();
  final amountCtrl = TextEditingController();
  final addressCtrl = TextEditingController();
  String status = '';
  Map<String, dynamic>? accounts;
  Map<String, dynamic>? deposits;
  Map<String, dynamic>? rates;
  Map<String, dynamic>? escrow;
  Map<String, dynamic>? history;
  Map<String, dynamic>? converts;
  Map<String, dynamic>? switches;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final a = await widget.api.getFinanceAccounts();
    final d = await widget.api.getDepositAddresses();
    final r = await widget.api.getConvertRates();
    final e = await widget.api.getEscrowOrders();
    final h = await widget.api.getFinanceHistory();
    final c = await widget.api.getConvertHistory();
    final sw = await widget.api.getFinanceSwitches();
    if (!mounted) return;
    setState(() {
      accounts = a;
      deposits = d;
      rates = r;
      escrow = e;
      history = h;
      converts = c;
      switches = sw;
    });
  }

  void _toast(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  Future<void> _transfer() async {
    final res = await widget.api.financeTransfer(toCtrl.text.trim(), currency, amountCtrl.text.trim());
    if (res != null) {
      _toast('Transfer completed');
      toCtrl.clear();
      amountCtrl.clear();
      _load();
    } else {
      _toast('Transfer failed');
    }
  }

  Future<void> _withdraw() async {
    final res = await widget.api.createWithdrawal(currency, amountCtrl.text.trim(), addressCtrl.text.trim());
    if (res != null) {
      _toast(res['status'] == 'auto_approved' ? 'Auto-approved' : 'Queued for superadmin sign-off');
      addressCtrl.clear();
      amountCtrl.clear();
      _load();
    } else {
      _toast('Withdrawal failed');
    }
  }

  Future<void> _convert() async {
    final res = await widget.api.financeConvert(currency, 'USDC', amountCtrl.text.trim());
    if (res != null) {
      _toast('Converted ${res['from_amount']} ${res['from_currency']} → ${res['to_amount']} USDC');
      amountCtrl.clear();
      _load();
    } else {
      _toast('Convert failed');
    }
  }

  Future<void> _escrowAction(Map<String, dynamic> order, String action, {String? reason}) async {
    final res = await widget.api.escrowAction('${order['id']}', action, reason: reason);
    _toast(res != null ? 'Escrow $action done' : 'Escrow action failed');
    _load();
  }

  Future<void> _assetDepositDetail(String asset) async {
    final detail = await widget.api.getDepositAddressByAsset(asset);
    final qr = await widget.api.getDepositAddressQr(asset);
    if (!mounted) return;
    showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Deposit $asset'),
        content: SingleChildScrollView(child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text('Address: ${detail?['address'] ?? 'unavailable'}',
              style: const TextStyle(fontSize: 12, fontFamily: 'monospace')),
          const SizedBox(height: 8),
          if (qr?['qr_png_b64'] != null)
            Image.memory(base64Decode(qr!['qr_png_b64'] as String), width: 180, height: 180, gaplessPlayback: true)
          else
            const Text('QR unavailable', style: TextStyle(fontSize: 11)),
        ])),
        actions: [
          TextButton(onPressed: () => Navigator.of(ctx).pop(), child: const Text('Close')),
          TextButton(
            onPressed: () {
              Clipboard.setData(ClipboardData(text: '${detail?['address'] ?? ''}'));
              Navigator.of(ctx).pop();
              _toast('Address copied');
            },
            child: const Text('Copy'),
          ),
        ],
      ),
    );
  }

  List<Widget> _escrowRows() {
    final orders = (escrow?['orders'] as List?) ?? const [];
    if (orders.isEmpty) return [const Text('No open orders')];
    return orders.map<Widget>((o) {
      final st = '${o['status']}';
      final actions = <Widget>[];
      if (st == 'open') {
        actions.add(TextButton(onPressed: () => _escrowAction(o, 'accept'), child: const Text('Buy')));
        actions.add(TextButton(onPressed: () => _escrowAction(o, 'cancel'), child: const Text('Cancel')));
      }
      if (st == 'escrowed') {
        actions.add(TextButton(onPressed: () => _escrowAction(o, 'paid'), child: const Text('Mark paid')));
        actions.add(TextButton(onPressed: () => _escrowAction(o, 'dispute', reason: 'disputed'), child: const Text('Dispute')));
      }
      if (st == 'paid') {
        actions.add(TextButton(onPressed: () => _escrowAction(o, 'release'), child: const Text('Release')));
        actions.add(TextButton(onPressed: () => _escrowAction(o, 'dispute', reason: 'disputed'), child: const Text('Dispute')));
      }
      return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('${o['amount']} ${o['currency']} @ ${o['fiat_amount']} ${o['fiat_currency']} (${o['payment_method_name'] ?? o['payment_method_code']}, $st)',
            style: const TextStyle(fontSize: 12)),
        Row(children: actions),
      ]);
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    final accts = (accounts?['accounts'] as List?) ?? const [];
    final deps = (deposits?['addresses'] as List?) ?? const [];
    final rateList = (rates?['rates'] as List?) ?? const [];
    final hist = (history?['history'] as List?) ?? const [];
    final convList = (converts?['conversions'] as List?) ?? const [];
    final swList = (switches?['switches'] as List?) ?? const [];
    return Scaffold(
      appBar: AppBar(title: const Text('Wallet & Finance')),
      body: ListView(padding: const EdgeInsets.all(12), children: [
        const Text('Accounts', style: TextStyle(fontWeight: FontWeight.bold)),
        if (accts.isEmpty) const Text('No accounts yet'),
        ...accts.map((a) => Text('${a['currency']}: ${a['balance']} (available ${a['available']})',
            style: const TextStyle(fontSize: 12, fontFamily: 'monospace'))),
        const SizedBox(height: 12),
        const Text('Deposit addresses (tap to copy)', style: TextStyle(fontWeight: FontWeight.bold)),
        if (deps.isEmpty) const Text('Deposit addresses unavailable on this node'),
        ...deps.map((d) => ListTile(
              dense: true,
              title: Text('${d['asset']}: ${d['address']}',
                  style: const TextStyle(fontSize: 11, fontFamily: 'monospace'), overflow: TextOverflow.ellipsis),
              trailing: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  IconButton(
                    icon: const Icon(Icons.qr_code, size: 16),
                    tooltip: 'QR / details',
                    onPressed: () => _assetDepositDetail('${d['asset']}'),
                  ),
                  IconButton(
                    icon: const Icon(Icons.copy, size: 16),
                    onPressed: () {
                      Clipboard.setData(ClipboardData(text: '${d['address']}'));
                      _toast('Address copied');
                    },
                  ),
                ],
              ),
            )),
        const SizedBox(height: 12),
        DropdownButton<String>(
          value: currency,
          items: assets.map((a) => DropdownMenuItem(value: a, child: Text(a))).toList(),
          onChanged: (v) => setState(() => currency = v ?? 'BTC'),
        ),
        TextField(controller: toCtrl, decoration: const InputDecoration(labelText: 'Recipient email (internal transfer)')),
        TextField(controller: amountCtrl, decoration: const InputDecoration(labelText: 'Amount'), keyboardType: TextInputType.number),
        ElevatedButton(onPressed: _transfer, child: const Text('Transfer (KYC-gated)')),
        TextField(controller: addressCtrl, decoration: const InputDecoration(labelText: 'Withdrawal destination address')),
        Row(children: [
          Expanded(child: ElevatedButton(onPressed: _withdraw, child: const Text('Withdraw'))),
          const SizedBox(width: 8),
          Expanded(child: ElevatedButton(onPressed: _convert, child: const Text('Convert → USDC'))),
        ]),
        const SizedBox(height: 12),
        const Text('Convert rates', style: TextStyle(fontWeight: FontWeight.bold)),
        if (rateList.isEmpty) const Text('No rates configured'),
        ...rateList.map((r) => Text('${r['from_currency']}/${r['to_currency']}: ${r['rate']}',
            style: const TextStyle(fontSize: 12, fontFamily: 'monospace'))),
        const SizedBox(height: 12),
        const Text('P2P escrow market', style: TextStyle(fontWeight: FontWeight.bold)),
        ..._escrowRows(),
        const SizedBox(height: 12),
        const Text('Ledger history', style: TextStyle(fontWeight: FontWeight.bold)),
        if (hist.isEmpty) const Text('No ledger history yet'),
        ...hist.take(30).map((h) => Text(
            '${h['kind']} ${h['direction'] == 'debit' ? '−' : '+'}${h['amount']} ${h['currency']}',
            style: const TextStyle(fontSize: 11, fontFamily: 'monospace'))),
        const SizedBox(height: 12),
        const Text('Convert history', style: TextStyle(fontWeight: FontWeight.bold)),
        if (convList.isEmpty) const Text('No conversions yet'),
        ...convList.take(20).map((cv) => Text(
            '${cv['from_currency']} ${cv['from_amount']} → ${cv['to_currency']} ${cv['to_amount']} @ ${cv['rate']}',
            style: const TextStyle(fontSize: 11, fontFamily: 'monospace'))),
        const SizedBox(height: 12),
        const Text('Feature switches', style: TextStyle(fontWeight: FontWeight.bold)),
        if (swList.isEmpty) const Text('No feature switches'),
        ...swList.map((sw) => Text('${sw['token'] ?? sw['asset']}: ${sw['enabled'] == true || sw['on'] == true ? 'on' : 'off'}',
            style: const TextStyle(fontSize: 12))),
        if (status.isNotEmpty) Text(status),
      ]),
    );
  }
}
