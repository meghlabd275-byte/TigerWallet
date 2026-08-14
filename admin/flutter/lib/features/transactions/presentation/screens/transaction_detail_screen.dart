/**
 * TigerWallet Admin - Transaction Detail Screen
 * Real backend-backed detail (wallet_api :8443 /api/v1/transactions/:id).
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../../core/constants/app_constants.dart';
import '../../../../core/network/api_client.dart';
import '../../../../core/network/dio_client.dart';
import '../../../../core/theme/app_theme.dart';

class TransactionDetailScreen extends ConsumerStatefulWidget {
  final String transactionId;
  const TransactionDetailScreen({super.key, required this.transactionId});

  @override
  ConsumerState<TransactionDetailScreen> createState() =>
      _TransactionDetailScreenState();
}

class _TransactionDetailScreenState
    extends ConsumerState<TransactionDetailScreen> {
  Map<String, dynamic>? _tx;
  bool _loading = true;
  String? _error;
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
    _loadTransaction();
  }

  Future<void> _loadTransaction() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = await _client();
      _tx = await api.getTransaction(widget.transactionId);
      setState(() {});
    } catch (e) {
      setState(() {
        _error = e.toString();
        _tx = null;
      });
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _flagTransaction() async {
    try {
      final api = await _client();
      await api.flagTransaction(widget.transactionId, 'flagged from admin UI');
      _loadTransaction();
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Transaction flagged')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Flag failed: $e')),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final isDark = _isDark;
    return Scaffold(
      appBar: AppBar(
        title: Text('Transaction: ${_shortId(widget.transactionId)}'),
        leading: IconButton(
            icon: const Icon(Icons.arrow_back),
            onPressed: () => context.go(AppConstants.transactionsRoute)),
        actions: [
          IconButton(
            icon: Icon(isDark ? Icons.light_mode : Icons.dark_mode),
            onPressed: () => setState(() => _isDark = !isDark),
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(
                  child: Padding(
                    padding: const EdgeInsets.all(24),
                    child: Text('Failed to load transaction: $_error',
                        textAlign: TextAlign.center),
                  ),
                )
              : _tx == null
                  ? const Center(child: Text('Transaction not found'))
                  : _buildDetail(context, _tx!, isDark),
    );
  }

  Widget _buildDetail(BuildContext context, Map<String, dynamic> tx, bool isDark) {
    final id = (tx['id'] ?? tx['tx_hash'] ?? widget.transactionId).toString();
    final from = (tx['from'] ?? tx['from_address'] ?? '-').toString();
    final to = (tx['to'] ?? tx['to_address'] ?? '-').toString();
    final chain = (tx['chain'] ?? tx['chain_id'] ?? '-').toString();
    final fee = (tx['fee'] ?? tx['gas_price'] ?? '-').toString();
    final time = (tx['timestamp'] ?? tx['created_at'] ?? '-').toString();
    final amount = (tx['amount'] ?? tx['value'] ?? 0).toString();
    final status = (tx['status'] ?? 'unknown').toString();
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          Container(
            padding: const EdgeInsets.all(24),
            decoration: BoxDecoration(
              color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
              borderRadius: BorderRadius.circular(16),
            ),
            child: Column(
              children: [
                const Icon(Icons.swap_horiz, size: 48, color: AppTheme.primaryColor),
                const SizedBox(height: 16),
                Text(amount,
                    style: Theme.of(context)
                        .textTheme
                        .headlineMedium
                        ?.copyWith(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                  decoration: BoxDecoration(
                    color: _getStatusColor(status).withOpacity(0.1),
                    borderRadius: BorderRadius.circular(20),
                  ),
                  child: Text(status,
                      style: TextStyle(color: _getStatusColor(status))),
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
                color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
                borderRadius: BorderRadius.circular(16)),
            child: Column(
              children: [
                _buildDetailRow(context, 'Transaction ID', id, isDark),
                _buildDetailRow(context, 'From', _shortId(from), isDark),
                _buildDetailRow(context, 'To', _shortId(to), isDark),
                _buildDetailRow(context, 'Chain', chain, isDark),
                _buildDetailRow(context, 'Fee', fee, isDark),
                _buildDetailRow(context, 'Time', time, isDark),
              ],
            ),
          ),
          const SizedBox(height: 24),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: _flagTransaction,
                  icon: const Icon(Icons.flag),
                  label: const Text('Flag'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton.icon(
                  onPressed: () {},
                  icon: const Icon(Icons.info),
                  label: const Text('View Blockchain'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  String _shortId(String s) {
    if (s.isEmpty) return '-';
    if (s.length <= 14) return s;
    return '${s.substring(0, 8)}...${s.substring(s.length - 6)}';
  }

  Color _getStatusColor(String status) {
    final s = status.toLowerCase();
    if (s.contains('completed') || s.contains('success')) return AppTheme.successColor;
    if (s.contains('pending')) return AppTheme.warningColor;
    if (s.contains('failed')) return AppTheme.errorColor;
    if (s.contains('flagged')) return AppTheme.errorColor;
    return Colors.grey;
  }

  Widget _buildDetailRow(BuildContext context, String label, String value, bool isDark) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label,
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: isDark ? Colors.grey[400] : Colors.grey[600])),
          Flexible(
            child: Text(value,
                textAlign: TextAlign.end,
                style: Theme.of(context)
                    .textTheme
                    .bodyMedium
                    ?.copyWith(fontWeight: FontWeight.w600)),
          ),
        ],
      ),
    );
  }
}
