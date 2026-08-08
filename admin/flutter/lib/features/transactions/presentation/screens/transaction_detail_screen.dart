/**
 * TigerWallet Admin - Transaction Detail Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/constants/app_constants.dart';

class TransactionDetailScreen extends ConsumerWidget {
  final String transactionId;
  const TransactionDetailScreen({super.key, required this.transactionId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return Scaffold(
      appBar: AppBar(
        title: Text('Transaction: $transactionId'),
        leading: IconButton(icon: const Icon(Icons.arrow_back), onPressed: () => context.go(AppConstants.transactionsRoute)),
      ),
      body: SingleChildScrollView(
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
                  Text('\$${(int.tryParse(transactionId) ?? 0 + 1) * 100}.00', style: Theme.of(context).textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  Container(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6), decoration: BoxDecoration(color: AppTheme.successColor.withOpacity(0.1), borderRadius: BorderRadius.circular(20)), child: const Text('Completed', style: TextStyle(color: AppTheme.successColor))),
                ],
              ),
            ),
            const SizedBox(height: 24),
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(color: isDark ? AppTheme.darkCard : AppTheme.lightCard, borderRadius: BorderRadius.circular(16)),
              child: Column(
                children: [
                  _buildDetailRow(context, 'Transaction ID', '0x1234567890abcdef', isDark),
                  _buildDetailRow(context, 'From', '0xabcd...1234', isDark),
                  _buildDetailRow(context, 'To', '0xefgh...5678', isDark),
                  _buildDetailRow(context, 'Chain', 'Ethereum (ETH)', isDark),
                  _buildDetailRow(context, 'Fee', '0.005 ETH', isDark),
                  _buildDetailRow(context, 'Time', '2024-01-15 10:30:00', isDark),
                ],
              ),
            ),
            const SizedBox(height: 24),
            Row(
              children: [
                Expanded(child: OutlinedButton.icon(onPressed: (){}, icon: const Icon(Icons.flag), label: const Text('Flag'))),
                const SizedBox(width: 12),
                Expanded(child: ElevatedButton.icon(onPressed: (){}, icon: const Icon(Icons.info), label: const Text('View Blockchain'))),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildDetailRow(BuildContext context, String label, String value, bool isDark) {
    return Padding(padding: const EdgeInsets.symmetric(vertical: 8), child: Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [Text(label, style: Theme.of(context).textTheme.bodyMedium?.copyWith(color: isDark ? Colors.grey[400] : Colors.grey[600])), Text(value, style: Theme.of(context).textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.w600))]));
  }
}
