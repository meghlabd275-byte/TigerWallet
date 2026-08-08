/**
 * TigerWallet Admin - Analytics Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:fl_chart/fl_chart.dart';
import '../../../../core/theme/app_theme.dart';

class AnalyticsScreen extends ConsumerWidget {
  const AnalyticsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return Scaffold(
      appBar: AppBar(title: const Text('Analytics')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Overview', style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            SizedBox(
              height: 200,
              child: LineChart(
                LineChartData(
                  gridData: FlGridData(show: false),
                  titlesData: FlTitlesData(show: false),
                  borderData: FlBorderData(show: false),
                  lineBarsData: [
                    LineChartBarData(spots: [const FlSpot(0, 3), const FlSpot(1, 1), const FlSpot(2, 4), const FlSpot(3, 2), const FlSpot(4, 5), const FlSpot(5, 3), const FlSpot(6, 4)], isCurved: true, color: AppTheme.primaryColor, barWidth: 3, dotData: FlDotData(show: false), belowBarData: BarAreaData(show: true, color: AppTheme.primaryColor.withOpacity(0.1))),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 24),
            Text('User Analytics', style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(color: isDark ? AppTheme.darkCard : AppTheme.lightCard, borderRadius: BorderRadius.circular(16)),
              child: Column(
                children: [
                  _buildStatRow(context, 'Total Users', '12,543', isDark),
                  _buildStatRow(context, 'Active Users', '8,921', isDark),
                  _buildStatRow(context, 'New Users (30d)', '1,234', isDark),
                  _buildStatRow(context, 'KYC Verified', '10,567', isDark),
                ],
              ),
            ),
            const SizedBox(height: 24),
            Text('Transaction Analytics', style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(color: isDark ? AppTheme.darkCard : AppTheme.lightCard, borderRadius: BorderRadius.circular(16)),
              child: Column(
                children: [
                  _buildStatRow(context, 'Total Transactions', '456,789', isDark),
                  _buildStatRow(context, 'Volume (30d)', '\$12,567,890', isDark),
                  _buildStatRow(context, 'Fees Collected', '\$234,567', isDark),
                  _buildStatRow(context, 'Avg Transaction', '\$27.50', isDark),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStatRow(BuildContext context, String label, String value, bool isDark) {
    return Padding(padding: const EdgeInsets.symmetric(vertical: 8), child: Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [Text(label, style: Theme.of(context).textTheme.bodyMedium?.copyWith(color: isDark ? Colors.grey[400] : Colors.grey[600])), Text(value, style: Theme.of(context).textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.bold))]));
  }
}
