import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import '../services/wallet_service.dart';
import '../services/api_service.dart';
import '../utils/theme.dart';

/// Notifications Screen - Real-time transaction and price alerts
class NotificationsScreen extends StatefulWidget {
  const NotificationsScreen({super.key});

  @override
  State<NotificationsScreen> createState() => _NotificationsScreenState();
}

class _NotificationsScreenState extends State<NotificationsScreen> {
  List<Map<String, dynamic>> _notifications = [];
  bool _isLoading = true;
  bool _hasUnread = true;

  @override
  void initState() {
    super.initState();
    _loadNotifications();
  }

  Future<void> _loadNotifications() async {
    setState(() => _isLoading = true);
    try {
      final response = await http.get(
        Uri.parse('$API_BASE_URL/api/v1/notifications'),
        headers: {'Content-Type': 'application/json'},
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          _notifications = List<Map<String, dynamic>>.from(data['notifications'] ?? []);
          _hasUnread = data['has_unread'] ?? false;
        });
      }
    } catch (e) {
      _loadDemoNotifications();
    }
    setState(() => _isLoading = false);
  }

  void _loadDemoNotifications() {
    setState(() {
      _notifications = [
        {
          'id': '1',
          'type': 'transaction',
          'title': 'Transaction Sent',
          'body': 'You sent 0.5 ETH to 0x742d...',
          'timestamp': DateTime.now().subtract(const Duration(minutes: 5)).toIso8601String(),
          'read': false,
          'icon': 'send',
          'color': Colors.orange,
        },
        {
          'id': '2',
          'type': 'price',
          'title': 'Price Alert: ETH',
          'body': 'Ethereum reached \$2,500 - Up 5%',
          'timestamp': DateTime.now().subtract(const Duration(hours: 1)).toIso8601String(),
          'read': false,
          'icon': 'trending_up',
          'color': Colors.green,
        },
        {
          'id': '3',
          'type': 'swap',
          'title': 'Swap Completed',
          'body': 'Swapped 1 ETH for 2,500 USDT',
          'timestamp': DateTime.now().subtract(const Duration(hours: 3)).toIso8601String(),
          'read': true,
          'icon': 'swap',
          'color': Colors.blue,
        },
        {
          'id': '4',
          'type': 'staking',
          'title': 'Staking Rewards',
          'body': 'You earned 0.01 ETH from staking',
          'timestamp': DateTime.now().subtract(const Duration(days: 1)).toIso8601String(),
          'read': true,
          'icon': 'savings',
          'color': Colors.purple,
        },
        {
          'id': '5',
          'type': 'security',
          'title': 'New Device Login',
          'body': 'A new device accessed your wallet',
          'timestamp': DateTime.now().subtract(const Duration(days: 2)).toIso8601String(),
          'read': true,
          'icon': 'security',
          'color': Colors.red,
        },
        {
          'id': '6',
          'type': 'system',
          'title': 'System Update',
          'body': 'TigerWallet v1.0.1 is now available',
          'timestamp': DateTime.now().subtract(const Duration(days: 3)).toIso8601String(),
          'read': true,
          'icon': 'system_update',
          'color': Colors.teal,
        },
      ];
      _hasUnread = true;
    });
  }

  Future<void> _markAsRead(String id) async {
    try {
      await http.post(
        Uri.parse('$API_BASE_URL/api/v1/notifications/read'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({'notification_id': id}),
      );
    } catch (e) {
      // Continue locally
    }

    setState(() {
      final index = _notifications.indexWhere((n) => n['id'] == id);
      if (index != -1) {
        _notifications[index]['read'] = true;
      }
      _hasUnread = _notifications.any((n) => n['read'] == false);
    });
  }

  Future<void> _markAllAsRead() async {
    try {
      await http.post(
        Uri.parse('$API_BASE_URL/api/v1/notifications/read-all'),
        headers: {'Content-Type': 'application/json'},
      );
    } catch (e) {
      // Continue locally
    }

    setState(() {
      for (var notification in _notifications) {
        notification['read'] = true;
      }
      _hasUnread = false;
    });
  }

  String _formatTimestamp(String timestamp) {
    final date = DateTime.parse(timestamp);
    final now = DateTime.now();
    final diff = now.difference(date);

    if (diff.inMinutes < 60) {
      return '${diff.inMinutes}m ago';
    } else if (diff.inHours < 24) {
      return '${diff.inHours}h ago';
    } else if (diff.inDays < 7) {
      return '${diff.inDays}d ago';
    } else {
      return '${date.day}/${date.month}/${date.year}';
    }
  }

  IconData _getIcon(String iconName) {
    switch (iconName) {
      case 'send':
        return Icons.send;
      case 'trending_up':
        return Icons.trending_up;
      case 'swap':
        return Icons.swap_horiz;
      case 'savings':
        return Icons.savings;
      case 'security':
        return Icons.security;
      case 'system_update':
        return Icons.system_update;
      default:
        return Icons.notifications;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        elevation: 0,
        title: const Text('Notifications', style: TextStyle(color: AppColors.textPrimary)),
        actions: [
          if (_hasUnread)
            TextButton(
              onPressed: _markAllAsRead,
              child: const Text('Mark all read'),
            ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : _notifications.isEmpty
              ? _buildEmptyState()
              : _buildNotificationList(),
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.notifications_none, size: 64, color: AppColors.textSecondary),
          const SizedBox(height: 16),
          const Text('No notifications yet',
              style: TextStyle(color: AppColors.textSecondary)),
          const SizedBox(height: 8),
          const Text('You\'ll see updates here',
              style: TextStyle(color: AppColors.textSecondary, fontSize: 12)),
        ],
      ),
    );
  }

  Widget _buildNotificationList() {
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: _notifications.length,
      itemBuilder: (context, index) {
        final notification = _notifications[index];
        return _buildNotificationItem(notification);
      },
    );
  }

  Widget _buildNotificationItem(Map<String, dynamic> notification) {
    final isUnread = notification['read'] == false;
    
    return GestureDetector(
      onTap: () {
        if (isUnread) {
          _markAsRead(notification['id']);
        }
      },
      child: Container(
        margin: const EdgeInsets.only(bottom: 12),
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: isUnread 
              ? AppColors.primary.withOpacity(0.05) 
              : AppColors.cardBackground,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: isUnread 
                ? AppColors.primary.withOpacity(0.2) 
                : AppColors.border.withOpacity(0.1),
          ),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(
                color: (notification['color'] as Color).withOpacity(0.1),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(
                _getIcon(notification['icon'] ?? 'notifications'),
                color: notification['color'] as Color,
                size: 22,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Expanded(
                        child: Text(
                          notification['title'] ?? '',
                          style: TextStyle(
                            fontWeight: isUnread ? FontWeight.bold : FontWeight.w600,
                            fontSize: 14,
                            color: AppColors.textPrimary,
                          ),
                        ),
                      ),
                      Text(
                        _formatTimestamp(notification['timestamp'] ?? ''),
                        style: TextStyle(
                          color: AppColors.textSecondary,
                          fontSize: 11,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(
                    notification['body'] ?? '',
                    style: TextStyle(
                      color: AppColors.textSecondary,
                      fontSize: 13,
                    ),
                  ),
                ],
              ),
            ),
            if (isUnread) ...[
              const SizedBox(width: 8),
              Container(
                width: 8,
                height: 8,
                decoration: const BoxDecoration(
                  color: AppColors.primary,
                  shape: BoxShape.circle,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
