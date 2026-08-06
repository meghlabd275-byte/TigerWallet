/**
 * TigerWallet Admin - Pairs Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../../core/theme/app_theme.dart';

class PairsScreen extends ConsumerWidget {
  const PairsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final pairs = [
      {'base': 'BTC', 'quote': 'USDT', 'price': '\$45,234.56', 'volume24h': '\$1.2B', 'status': 'Active'},
      {'base': 'ETH', 'quote': 'USDT', 'price': '\$2,456.78', 'volume24h': '\$890M', 'status': 'Active'},
      {'base': 'BNB', 'quote': 'USDT', 'price': '\$312.45', 'volume24h': '\$456M', 'status': 'Halted'},
      {'base': 'SOL', 'quote': 'USDT', 'price': '\$98.76', 'volume24h': '\$234M', 'status': 'Active'},
    ];

    return Scaffold(
      appBar: AppBar(title: const Text('Trading Pairs')),
      floatingActionButton: FloatingActionButton(backgroundColor: AppTheme.primaryColor, onPressed: (){}, child: const Icon(Icons.add, color: Colors.white)),
      body: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: pairs.length,
        itemBuilder: (context, index) {
          final pair = pairs[index];
          return Card(
            margin: const EdgeInsets.only(bottom: 12),
            color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            child: ListTile(
              title: Text('${pair['base']}/${pair['quote']}', style: const TextStyle(fontWeight: FontWeight.bold)),
              subtitle: Text('24h: ${pair['volume24h']}'),
              trailing: Row(mainAxisSize: MainAxisSize.min, children: [
                Column(mainAxisAlignment: MainAxisAlignment.center, crossAxisAlignment: CrossAxisAlignment.end, children: [Text(pair['price']!, style: const TextStyle(fontWeight: FontWeight.bold)), Container(padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2), decoration: BoxDecoration(color: pair['status'] == 'Active' ? AppTheme.successColor.withOpacity(0.1) : AppTheme.errorColor.withOpacity(0.1), borderRadius: BorderRadius.circular(10)), child: Text(pair['status']!, style: TextStyle(color: pair['status'] == 'Active' ? AppTheme.successColor : AppTheme.errorColor, fontSize: 12)))]),
                const SizedBox(width: 8),
                const Icon(Icons.more_vert),
              ]),
            ),
          );
        },
      ),
    );
  }
}
