/**
 * TigerWallet Admin - Tickets Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../../core/theme/app_theme.dart';

class TicketsScreen extends ConsumerWidget {
  const TicketsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final tickets = [
      {'title': 'Cannot withdraw funds', 'type': 'Withdrawal', 'priority': 'High', 'status': 'Open'},
      {'title': 'KYC verification pending', 'type': 'KYC', 'priority': 'Medium', 'status': 'In Progress'},
      {'title': 'Account locked', 'type': 'Account', 'priority': 'Urgent', 'status': 'Open'},
      {'title': 'Transaction not confirmed', 'type': 'Transaction', 'priority': 'Low', 'status': 'Resolved'},
    ];

    return Scaffold(
      appBar: AppBar(title: const Text('Support Tickets')),
      floatingActionButton: FloatingActionButton(backgroundColor: AppTheme.primaryColor, onPressed: (){}, child: const Icon(Icons.add, color: Colors.white)),
      body: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: tickets.length,
        itemBuilder: (context, index) {
          final ticket = tickets[index];
          return Card(
            margin: const EdgeInsets.only(bottom: 12),
            color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            child: ListTile(
              title: Text(ticket['title']!, style: const TextStyle(fontWeight: FontWeight.bold)),
              subtitle: Text('${ticket['type']} • ${ticket['status']}'),
              trailing: Container(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6), decoration: BoxDecoration(color: _getPriorityColor(ticket['priority']!).withOpacity(0.1), borderRadius: BorderRadius.circular(20)), child: Text(ticket['priority']!, style: TextStyle(color: _getPriorityColor(ticket['priority']!), fontWeight: FontWeight.w600, fontSize: 12))),
            ),
          );
        },
      ),
    );
  }

  Color _getPriorityColor(String priority) {
    switch (priority) { case 'Urgent': return AppTheme.errorColor; case 'High': return AppTheme.warningColor; case 'Medium': return AppTheme.infoColor; default: return Colors.grey; }
  }
}
