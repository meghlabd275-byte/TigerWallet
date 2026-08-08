/**
 * TigerWallet Admin - Users Screen
 * Complete Dark/Light Theme Support
 */

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/constants/app_constants.dart';

class UsersScreen extends ConsumerStatefulWidget {
  const UsersScreen({super.key});

  @override
  ConsumerState<UsersScreen> createState() => _UsersScreenState();
}

class _UsersScreenState extends ConsumerState<UsersScreen> {
  String _searchQuery = '';
  String _selectedStatus = 'all';
  int _currentPage = 1;

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Users'),
        actions: [
          IconButton(
            icon: const Icon(Icons.filter_list),
            onPressed: () => _showFilterDialog(context),
          ),
        ],
      ),
      body: Column(
        children: [
          // Search and Filter
          Container(
            padding: const EdgeInsets.all(16),
            child: TextField(
              decoration: InputDecoration(
                hintText: 'Search users...',
                prefixIcon: const Icon(Icons.search),
                filled: true,
                fillColor: isDark ? AppTheme.darkCard : AppTheme.lightCard,
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                  borderSide: BorderSide.none,
                ),
              ),
              onChanged: (value) {
                setState(() => _searchQuery = value);
              },
            ),
          ),
          
          // Status Tabs
          Container(
            height: 50,
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: ListView(
              scrollDirection: Axis.horizontal,
              children: [
                _buildStatusChip(context, 'All', 'all', isDark),
                _buildStatusChip(context, 'Active', 'active', isDark),
                _buildStatusChip(context, 'Suspended', 'suspended', isDark),
                _buildStatusChip(context, 'Banned', 'banned', isDark),
                _buildStatusChip(context, 'Pending', 'pending', isDark),
              ],
            ),
          ),
          
          // Users List
          Expanded(
            child: ListView.builder(
              padding: const EdgeInsets.all(16),
              itemCount: 10,
              itemBuilder: (context, index) {
                return _buildUserCard(context, index, isDark);
              },
            ),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () {},
        backgroundColor: AppTheme.primaryColor,
        child: const Icon(Icons.add, color: Colors.white),
      ),
    );
  }

  Widget _buildStatusChip(BuildContext context, String label, String value, bool isDark) {
    final isSelected = _selectedStatus == value;
    return Padding(
      padding: const EdgeInsets.only(right: 8),
      child: FilterChip(
        label: Text(label),
        selected: isSelected,
        onSelected: (selected) {
          setState(() => _selectedStatus = value);
        },
        selectedColor: AppTheme.primaryColor.withOpacity(0.2),
        checkmarkColor: AppTheme.primaryColor,
        backgroundColor: isDark ? AppTheme.darkCard : AppTheme.lightCard,
      ),
    );
  }

  Widget _buildUserCard(BuildContext context, int index, bool isDark) {
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      color: isDark ? AppTheme.darkCard : AppTheme.lightCard,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: InkWell(
        onTap: () => context.go('${AppConstants.usersRoute}/$index'),
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              CircleAvatar(
                radius: 24,
                backgroundColor: AppTheme.primaryColor.withOpacity(0.1),
                child: Text(
                  'U${index + 1}',
                  style: const TextStyle(
                    color: AppTheme.primaryColor,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'User ${index + 1}',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                        color: isDark ? Colors.white : const Color(0xFF1A1A2E),
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      'user${index + 1}@example.com',
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: isDark ? Colors.grey[400] : Colors.grey[600],
                      ),
                    ),
                  ],
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                decoration: BoxDecoration(
                  color: _getStatusColor(index % 4).withOpacity(0.1),
                  borderRadius: BorderRadius.circular(20),
                ),
                child: Text(
                  _getStatusText(index % 4),
                  style: TextStyle(
                    color: _getStatusColor(index % 4),
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
              const SizedBox(width: 8),
              Icon(
                Icons.chevron_right,
                color: isDark ? Colors.grey[600] : Colors.grey[400],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Color _getStatusColor(int status) {
    switch (status) {
      case 0: return AppTheme.successColor;
      case 1: return AppTheme.warningColor;
      case 2: return AppTheme.errorColor;
      default: return AppTheme.infoColor;
    }
  }

  String _getStatusText(int status) {
    switch (status) {
      case 0: return 'Active';
      case 1: return 'Suspended';
      case 2: return 'Banned';
      default: return 'Pending';
    }
  }

  void _showFilterDialog(BuildContext context) {
    showModalBottomSheet(
      context: context,
      builder: (context) => Container(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Filter Users',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 24),
            ListTile(
              leading: const Icon(Icons.calendar_today),
              title: const Text('Date Range'),
              onTap: () {},
            ),
            ListTile(
              leading: const Icon(Icons.language),
              title: const Text('Country'),
              onTap: () {},
            ),
            ListTile(
              leading: const Icon(Icons.verified_user),
              title: const Text('KYC Status'),
              onTap: () {},
            ),
            const SizedBox(height: 16),
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Apply Filters'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
