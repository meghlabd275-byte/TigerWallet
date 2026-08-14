/**
 * TigerWallet Admin - Transactions Screen
 * Real backend-backed list (wallet_api :8443 /api/v1/transactions).
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../../core/constants/app_constants.dart';
import '../../../../core/network/api_client.dart';
import '../../../../core/network/dio_client.dart';
import '../../../../core/theme/app_theme.dart';

class TransactionsScreen extends ConsumerStatefulWidget {
  const TransactionsScreen({super.key});

  @override
  ConsumerState<TransactionsScreen> createState() => _TransactionsScreenState();
}

class _TransactionsScreenState extends ConsumerState<TransactionsScreen> {
  List<Map<String, dynamic>> _transactions = [];
  bool _loading = true;
  String? _error;
  String _statusFilter = 'all';
  bool _isDark = false;

  ApiClient? _api;

  Future<ApiClient> _client() async {
    if (_api != null) return _api!;
    final prefs = await SharedPreferences.getInstance();
    _api = DioClient.withPrefs(prefs);
    return _api!;
  }

  @override
  void initState() {
    super.initState();
    _loadTransactions();
  }

  Future<void> _loadTransactions() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = await _client();
      final result = await api.getTransactions(
        page: 1,
        pageSize: 50,
        status: _statusFilter == 'all' ? null : _statusFilter,
      );
      final data = result['data'];
      _transactions = data is List
          ? List<Map<String, dynamic>>.from(data)
          : <Map<String, dynamic>>[];
      setState(() {});
    } catch (e) {
      setState(() {
        _error = e.toString();
        _transactions = [];
      });
    } finally {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final isDark = _isDark;
    return Scaffold(
      appBar: AppBar(
        title: const Text('Transactions'),
        actions: [
          IconButton(
            icon: Icon(isDark ? Icons.light_mode : Icons.dark_mode),
            onPressed: () => setState(() => _isDark = !isDark),
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: DropdownButton<String>(
              value: _statusFilter,
              isExpanded: true,
              items: const [
                DropdownMenuItem(value: 'all', child: Text('All statuses')),
                DropdownMenuItem(value: 'pending', child: Text('Pending')),
                DropdownMenuItem(value: 'completed', child: Text('Completed')),
                DropdownMenuItem(value: 'failed', child: Text('Failed')),
                DropdownMenuItem(value: 'flagged', child: Text('Flagged')),
              ],
              onChanged: (v) {
                if (v == null) return;
                _statusFilter = v;
                _loadTransactions();
              },
            ),
          ),
          if (_loading)
            const Expanded(child: Center(child: CircularProgressIndicator()))
          else if (_error != null)
            Expanded(
              child: Center(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text('Failed to load transactions: $_error',
                      textAlign: TextAlign.center),
                ),
              ),
            )
          else if (_transactions.isEmpty)
            const Expanded(child: Center(child: Text('No transactions found')))
          else
            Expanded(
              child: RefreshIndicator(
                onRefresh: _loadTransactions,
                child: ListView.builder(
                  padding: const EdgeInsets.all(16),
                  itemCount: _transactions.length,
                  itemBuilder: (context, index) =>
                      _buildTransactionCard(context, _transactions[index], isDark),
                ),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildTransactionCard(BuildContext context, Map<String, dynamic> tx, bool isDark) {
    final id = (tx['id'] ?? tx['tx_hash'] ?? '').toString();
    final type = (tx['type'] ?? 'Transaction').toString();
    final status = (tx['status'] ?? 'unknown').toString();
    final amount = (tx['amount'] ?? tx['value'] ?? 0).toString();
    final from = (tx['from'] ?? tx['from_address'] ?? '-').toString();
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: InkWell(
        onTap: id.isEmpty ? null : () => context.go('${AppConstants.transactionsRoute}/$id'),
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              CircleAvatar(
                radius: 24,
                backgroundColor: _getTypeColor(type).withOpacity(0.1),
                child: Icon(_getTypeIcon(type), color: _getTypeColor(type)),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(type, style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                    Text('Tx: ${_shortId(id)}', style: Theme.of(context).textTheme.bodySmall),
                    Text('From: ${_shortId(from)}', style: Theme.of(context).textTheme.bodySmall),
                  ],
                ),
              ),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(amount, style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  Text(status, style: TextStyle(color: _getStatusColor(status), fontSize: 12)),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  String _shortId(String s) {
    if (s.isEmpty) return '-';
    if (s.length <= 14) return s;
    return '${s.substring(0, 8)}...${s.substring(s.length - 6)}';
  }

  IconData _getTypeIcon(String type) {
    final t = type.toLowerCase();
    if (t.contains('deposit')) return Icons.arrow_downward;
    if (t.contains('withdraw')) return Icons.arrow_upward;
    if (t.contains('swap')) return Icons.swap_horiz;
    if (t.contains('transfer')) return Icons.swap_calls;
    return Icons.receipt_long;
  }

  Color _getTypeColor(String type) {
    final t = type.toLowerCase();
    if (t.contains('deposit')) return AppTheme.successColor;
    if (t.contains('withdraw')) return AppTheme.errorColor;
    if (t.contains('swap')) return AppTheme.infoColor;
    return AppTheme.accentColor;
  }

  Color _getStatusColor(String status) {
    final s = status.toLowerCase();
    if (s.contains('completed') || s.contains('success')) return AppTheme.successColor;
    if (s.contains('pending')) return AppTheme.warningColor;
    if (s.contains('failed')) return AppTheme.errorColor;
    if (s.contains('flagged')) return AppTheme.errorColor;
    return Colors.grey;
  }
}
