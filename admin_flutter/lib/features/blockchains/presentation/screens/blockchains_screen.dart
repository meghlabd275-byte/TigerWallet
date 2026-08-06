/**
 * TigerWallet Admin - Blockchains Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../../core/theme/app_theme.dart';

class BlockchainsScreen extends ConsumerWidget {
  const BlockchainsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final blockchains = [
      {'name': 'Ethereum', 'symbol': 'ETH', 'chainId': 1, 'type': 'EVM', 'status': 'Active'},
      {'name': 'Bitcoin', 'symbol': 'BTC', 'chainId': 0, 'type': 'Non-EVM', 'status': 'Active'},
      {'name': 'BNB Smart Chain', 'symbol': 'BNB', 'chainId': 56, 'type': 'EVM', 'status': 'Active'},
      {'name': 'Solana', 'symbol': 'SOL', 'chainId': 101, 'type': 'Non-EVM', 'status': 'Active'},
      {'name': 'Polygon', 'symbol': 'MATIC', 'chainId': 137, 'type': 'EVM', 'status': 'Active'},
    ];

    return Scaffold(
      appBar: AppBar(title: const Text('Blockchains')),
      floatingActionButton: FloatingActionButton(backgroundColor: AppTheme.primaryColor, onPressed: (){}, child: const Icon(Icons.add, color: Colors.white)),
      body: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: blockchains.length,
        itemBuilder: (context, index) {
          final bc = blockchains[index];
          return Card(
            margin: const EdgeInsets.only(bottom: 12),
            color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            child: ListTile(
              leading: CircleAvatar(backgroundColor: AppTheme.primaryColor.withOpacity(0.1), child: Text(bc['symbol']![0], style: const TextStyle(color: AppTheme.primaryColor, fontWeight: FontWeight.bold))),
              title: Text(bc['name']!, style: const TextStyle(fontWeight: FontWeight.bold)),
              subtitle: Text('Chain ID: ${bc['chainId']} • ${bc['type']}'),
              trailing: Container(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6), decoration: BoxDecoration(color: AppTheme.successColor.withOpacity(0.1), borderRadius: BorderRadius.circular(20)), child: const Text('Active', style: TextStyle(color: AppTheme.successColor, fontWeight: FontWeight.w600))),
            ),
          );
        },
      ),
    );
  }
}
