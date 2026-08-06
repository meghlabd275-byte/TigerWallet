/**
 * TigerWallet Admin - Transactions Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/constants/app_constants.dart';

class TransactionsScreen extends ConsumerWidget {
  const TransactionsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return Scaffold(
      appBar: AppBar(title: const Text('Transactions')),
      body: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: 20,
        itemBuilder: (context, index) => _buildTransactionCard(context, index, isDark),
      ),
    );
  }

  Widget _buildTransactionCard(BuildContext context, int index, bool isDark) {
    final types = ['Deposit', 'Withdraw', 'Transfer', 'Swap'];
    final statuses = ['Completed', 'Pending', 'Failed', 'Flagged'];
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: InkWell(
        onTap: () => context.go('${AppConstants.transactionsRoute}/$index'),
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              CircleAvatar(
                radius: 24,
                backgroundColor: _getTypeColor(index % 4).withOpacity(0.1),
                child: Icon(_getTypeIcon(index % 4), color: _getTypeColor(index % 4)),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(types[index % 4], style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                    Text('Tx: 0x${index.toRadixString(16).padLeft(10, '0')}', style: Theme.of(context).textTheme.bodySmall),
                  ],
                ),
              ),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text('\$${(index + 1) * 100}.00', style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                  Text(statuses[index % 4], style: TextStyle(color: _getStatusColor(index % 4), fontSize: 12)),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  IconData _getTypeIcon(int type) {
    switch (type) { case 0: return Icons.arrow_downward; case 1: return Icons.arrow_upward; case 2: return Icons.swap_horiz; default: return Icons.swap_calls; }
  }

  Color _getTypeColor(int type) {
    switch (type) { case 0: return AppTheme.successColor; case 1: return AppTheme.errorColor; case 2: return AppTheme.infoColor; default: return AppTheme.accentColor; }
  }

  Color _getStatusColor(int status) {
    switch (status) { case 0: return AppTheme.successColor; case 1: return AppTheme.warningColor; case 2: return AppTheme.errorColor; default: return AppTheme.errorColor; }
  }
}
