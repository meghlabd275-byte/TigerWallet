/**
 * TigerWallet Admin - Tokens Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../../core/theme/app_theme.dart';

class TokensScreen extends ConsumerWidget {
  const TokensScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final tokens = [
      {'name': 'Bitcoin', 'symbol': 'BTC', 'price': '\$45,234.56', 'volume': '\$1.2B', 'status': 'Active'},
      {'name': 'Ethereum', 'symbol': 'ETH', 'price': '\$2,456.78', 'volume': '\$890M', 'status': 'Active'},
      {'name': 'Tether', 'symbol': 'USDT', 'price': '\$1.00', 'volume': '\$2.1B', 'status': 'Active'},
      {'name': 'BNB', 'symbol': 'BNB', 'price': '\$312.45', 'volume': '\$456M', 'status': 'Active'},
      {'name': 'Solana', 'symbol': 'SOL', 'price': '\$98.76', 'volume': '\$234M', 'status': 'Active'},
    ];

    return Scaffold(
      appBar: AppBar(title: const Text('Tokens')),
      floatingActionButton: FloatingActionButton(backgroundColor: AppTheme.primaryColor, onPressed: (){}, child: const Icon(Icons.add, color: Colors.white)),
      body: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: tokens.length,
        itemBuilder: (context, index) {
          final token = tokens[index];
          return Card(
            margin: const EdgeInsets.only(bottom: 12),
            color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            child: ListTile(
              leading: CircleAvatar(backgroundColor: AppTheme.primaryColor.withOpacity(0.1), child: Text(token['symbol']![0], style: const TextStyle(color: AppTheme.primaryColor, fontWeight: FontWeight.bold))),
              title: Text(token['name']!, style: const TextStyle(fontWeight: FontWeight.bold)),
              subtitle: Text(token['symbol']!),
              trailing: Column(mainAxisAlignment: MainAxisAlignment.center, crossAxisAlignment: CrossAxisAlignment.end, children: [Text(token['price']!, style: const TextStyle(fontWeight: FontWeight.bold)), Text(token['volume']!, style: Theme.of(context).textTheme.bodySmall)]),
            ),
          );
        },
      ),
    );
  }
}
