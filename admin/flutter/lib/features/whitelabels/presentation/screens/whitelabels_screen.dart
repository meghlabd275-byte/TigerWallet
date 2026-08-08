/**
 * TigerWallet Admin - White Labels Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../../core/theme/app_theme.dart';

class WhiteLabelsScreen extends ConsumerWidget {
  const WhiteLabelsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final whiteLabels = [
      {'company': 'Company A', 'domain': 'a.tigerwallet.com', 'status': 'Active', 'users': 1234},
      {'company': 'Company B', 'domain': 'b.tigerwallet.com', 'status': 'Active', 'users': 567},
      {'company': 'Company C', 'domain': 'c.tigerwallet.com', 'status': 'Pending', 'users': 0},
      {'company': 'Company D', 'domain': 'd.tigerwallet.com', 'status': 'Suspended', 'users': 89},
    ];

    return Scaffold(
      appBar: AppBar(title: const Text('White Labels')),
      floatingActionButton: FloatingActionButton(backgroundColor: AppTheme.primaryColor, onPressed: (){}, child: const Icon(Icons.add, color: Colors.white)),
      body: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: whiteLabels.length,
        itemBuilder: (context, index) {
          final wl = whiteLabels[index];
          return Card(
            margin: const EdgeInsets.only(bottom: 12),
            color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(child: Text(wl['company']!, style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold))),
                      Container(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6), decoration: BoxDecoration(color: _getStatusColor(wl['status']!).withOpacity(0.1), borderRadius: BorderRadius.circular(20)), child: Text(wl['status']!, style: TextStyle(color: _getStatusColor(wl['status']!), fontWeight: FontWeight.w600))),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Text(wl['domain']!, style: Theme.of(context).textTheme.bodySmall),
                  const SizedBox(height: 8),
                  Text('Users: ${wl['users']}', style: Theme.of(context).textTheme.bodyMedium),
                ],
              ),
            ),
          );
        },
      ),
    );
  }

  Color _getStatusColor(String status) {
    switch (status) { case 'Active': return AppTheme.successColor; case 'Pending': return AppTheme.warningColor; case 'Suspended': return AppTheme.errorColor; default: return Colors.grey; }
  }
}
