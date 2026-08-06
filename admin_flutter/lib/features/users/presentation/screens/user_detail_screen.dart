/**
 * TigerWallet Admin - User Detail Screen
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/constants/app_constants.dart';

class UserDetailScreen extends ConsumerWidget {
  final String userId;

  const UserDetailScreen({super.key, required this.userId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isDark = Theme.of(context).brightness == Brightness.dark;

    return Scaffold(
      appBar: AppBar(
        title: Text('User: $userId'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.go(AppConstants.usersRoute),
        ),
        actions: [
          PopupMenuButton<String>(
            onSelected: (value) {
              // Handle actions
            },
            itemBuilder: (context) => [
              const PopupMenuItem(value: 'ban', child: Text('Ban User')),
              const PopupMenuItem(value: 'suspend', child: Text('Suspend User')),
              const PopupMenuItem(value: 'delete', child: Text('Delete User')),
            ],
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // User Info Card
            _buildInfoCard(context, isDark),
            const SizedBox(height: 24),
            
            // Actions
            _buildActions(context, isDark),
            const SizedBox(height: 24),
            
            // Activity
            _buildActivitySection(context, isDark),
          ],
        ),
      ),
    );
  }

  Widget _buildInfoCard(BuildContext context, bool isDark) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        children: [
          CircleAvatar(
            radius: 50,
            backgroundColor: AppTheme.primaryColor.withOpacity(0.1),
            child: const Icon(Icons.person, size: 50, color: AppTheme.primaryColor),
          ),
          const SizedBox(height: 16),
          Text(
            'User $userId',
            style: Theme.of(context).textTheme.titleLarge?.copyWith(
              fontWeight: FontWeight.bold,
              color: isDark ? Colors.white : const Color(0xFF1A1A2E),
            ),
          ),
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: AppTheme.successColor.withOpacity(0.1),
              borderRadius: BorderRadius.circular(20),
            ),
            child: const Text(
              'Active',
              style: TextStyle(color: AppTheme.successColor, fontWeight: FontWeight.w600),
            ),
          ),
          const SizedBox(height: 24),
          _buildInfoRow(context, 'Email', 'user@example.com', isDark),
          _buildInfoRow(context, 'Phone', '+1234567890', isDark),
          _buildInfoRow(context, 'KYC Level', 'Level 2', isDark),
          _buildInfoRow(context, 'Country', 'United States', isDark),
          _buildInfoRow(context, 'Registered', 'Jan 15, 2024', isDark),
          _buildInfoRow(context, 'Last Login', 'Today', isDark),
        ],
      ),
    );
  }

  Widget _buildInfoRow(BuildContext context, String label, String value, bool isDark) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(
            label,
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
              color: isDark ? Colors.grey[400] : Colors.grey[600],
            ),
          ),
          Text(
            value,
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
              fontWeight: FontWeight.w600,
              color: isDark ? Colors.white : const Color(0xFF1A1A2E),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildActions(BuildContext context, bool isDark) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Actions',
          style: Theme.of(context).textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
            color: isDark ? Colors.white : const Color(0xFF1A1A2E),
          ),
        ),
        const SizedBox(height: 16),
        Row(
          children: [
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () {},
                icon: const Icon(Icons.ban, color: AppTheme.errorColor),
                label: const Text('Ban', style: TextStyle(color: AppTheme.errorColor)),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () {},
                icon: const Icon(Icons.pause, color: AppTheme.warningColor),
                label: const Text('Suspend', style: TextStyle(color: AppTheme.warningColor)),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () {},
                icon: const Icon(Icons.edit),
                label: const Text('Edit'),
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildActivitySection(BuildContext context, bool isDark) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Recent Activity',
          style: Theme.of(context).textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
            color: isDark ? Colors.white : const Color(0xFF1A1A2E),
          ),
        ),
        const SizedBox(height: 16),
        Container(
          decoration: BoxDecoration(
            color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
            borderRadius: BorderRadius.circular(16),
          ),
          child: ListView.separated(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            itemCount: 5,
            separatorBuilder: (_, __) => Divider(
              height: 1,
              color: isDark ? AppTheme.darkDivider : AppTheme.lightDivider,
            ),
            itemBuilder: (context, index) {
              return ListTile(
                leading: const CircleAvatar(
                  backgroundColor: AppTheme.successColor,
                  child: Icon(Icons.swap_horiz, color: Colors.white, size: 16),
                ),
                title: Text('Transaction #${index + 1}'),
                subtitle: Text('\$${(index + 1) * 100}.00'),
                trailing: const Text('Completed'),
              );
            },
          ),
        ),
      ],
    );
  }
}
