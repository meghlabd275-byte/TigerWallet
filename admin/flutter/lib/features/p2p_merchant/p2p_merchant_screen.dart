/**
 * TigerWallet Admin - P2P Merchant Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';

class P2PMerchantScreen extends StatefulWidget {
  const P2PMerchantScreen({Key? key}) : super(key: key);

  @override
  State<P2PMerchantScreen> createState() => _P2PMerchantScreenState();
}

class _P2PMerchantScreenState extends State<P2PMerchantScreen> {
  List<Map<String, dynamic>> _merchants = [];
  bool _loading = true;
  String? _error;
  String _filter = 'all';
  bool _isDark = false;

  ApiClient? _api;

  @override
  void initState() {
    super.initState();
    _loadMerchants();
  }

  Future<ApiClient> _client() async {
    if (_api != null) return _api!;
    final prefs = await SharedPreferences.getInstance();
    _api = DioClient.withPrefs(prefs);
    return _api!;
  }

  Future<void> _loadMerchants() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = await _client();
      _merchants = await api.getAdminP2PMerchants(status: _filter);
      setState(() {});
    } catch (e) {
      setState(() {
        _error = e.toString();
        _merchants = [];
      });
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _approveMerchant(String id) async {
    try {
      final api = await _client();
      await api.approveP2PMerchant(id);
      _loadMerchants();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Approve failed: $e')));
      }
    }
  }

  Future<void> _rejectMerchant(String id) async {
    final reason = await showDialog<String>(
      context: context,
      builder: (context) {
        final controller = TextEditingController();
        return AlertDialog(
          title: const Text('Reject Merchant'),
          content: TextField(controller: controller, decoration: const InputDecoration(labelText: 'Reason')),
          actions: [
            TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
            TextButton(onPressed: () => Navigator.pop(context, controller.text), child: const Text('Reject')),
          ],
        );
      },
    );
    if (reason != null && reason.isNotEmpty) {
      try {
        final api = await _client();
        await api.rejectP2PMerchant(id, reason);
        _loadMerchants();
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Reject failed: $e')));
        }
      }
    }
  }

  void _toggleTheme() => setState(() => _isDark = !_isDark);

  @override
  Widget build(BuildContext context) {
    return Theme(
      data: _isDark ? ThemeData.dark() : ThemeData.light(),
      child: Scaffold(
        appBar: AppBar(
          title: const Text('P2P Merchants'),
          actions: [IconButton(icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode), onPressed: _toggleTheme)],
        ),
        body: Column(
          children: [
            _buildStats(),
            _buildFilterChips(),
            Expanded(child: _buildMerchantsList()),
          ],
        ),
      ),
    );
  }

  Widget _buildStats() {
    return Container(
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          Expanded(child: _buildStatCard('Total', '${_merchants.length}')),
          Expanded(child: _buildStatCard('Approved', '${_merchants.where((m) => m['status'] == 'approved').length}')),
          Expanded(child: _buildStatCard('Pending', '${_merchants.where((m) => m['status'] == 'pending').length}')),
          Expanded(
            child: _buildStatCard(
              'Volume',
              '\$${(_merchants.fold(0.0, (sum, m) => sum + ((m['total_volume'] as num?) ?? 0.0)) / 1000).toStringAsFixed(0)}K',
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildStatCard(String label, String value) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          children: [
            Text(value, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
            Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])),
          ],
        ),
      ),
    );
  }

  Widget _buildFilterChips() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          FilterChip(
            label: const Text('All'),
            selected: _filter == 'all',
            onSelected: (_) {
              setState(() => _filter = 'all');
              _loadMerchants();
            },
          ),
          const SizedBox(width: 8),
          FilterChip(
            label: const Text('Pending'),
            selected: _filter == 'pending',
            onSelected: (_) {
              setState(() => _filter = 'pending');
              _loadMerchants();
            },
          ),
          const SizedBox(width: 8),
          FilterChip(
            label: const Text('Approved'),
            selected: _filter == 'approved',
            onSelected: (_) {
              setState(() => _filter = 'approved');
              _loadMerchants();
            },
          ),
        ],
      ),
    );
  }

  Widget _buildMerchantsList() {
    if (_loading) return const Center(child: CircularProgressIndicator());
    if (_error != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.error_outline, size: 48, color: Colors.red),
            const SizedBox(height: 12),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 32),
              child: Text(
                'Failed to load P2P merchants: $_error',
                textAlign: TextAlign.center,
                style: TextStyle(color: Colors.grey[600]),
              ),
            ),
            const SizedBox(height: 12),
            ElevatedButton(onPressed: _loadMerchants, child: const Text('Retry')),
          ],
        ),
      );
    }
    if (_merchants.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.store_mall_directory_outlined, size: 48, color: Colors.grey),
            const SizedBox(height: 12),
            Text('No P2P merchants found', style: TextStyle(color: Colors.grey[600])),
          ],
        ),
      );
    }
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: _merchants.length,
      itemBuilder: (context, index) {
        final merchant = _merchants[index];
        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text('${merchant['business_name'] ?? '—'}', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                    Chip(
                      label: Text('${merchant['status'] ?? ''}'.toString().toUpperCase()),
                      backgroundColor: _getStatusColor(merchant['status']?.toString() ?? ''),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Text('${merchant['email'] ?? '—'} • ${merchant['country'] ?? '—'}'),
                const SizedBox(height: 8),
                Row(
                  children: [
                    _buildDetail(
                      'Volume',
                      merchant['total_volume'] != null
                          ? '\$${((merchant['total_volume'] as num) / 1000).toStringAsFixed(1)}K'
                          : '—',
                    ),
                    _buildDetail('Txns', '${merchant['transaction_count'] ?? '—'}'),
                    _buildDetail('Rating', '${merchant['rating'] ?? '—'} ★'),
                  ],
                ),
                if (merchant['status'] == 'pending') ...[
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      ElevatedButton(
                        onPressed: () => _approveMerchant(merchant['id'].toString()),
                        style: ElevatedButton.styleFrom(backgroundColor: Colors.green),
                        child: const Text('Approve'),
                      ),
                      const SizedBox(width: 8),
                      ElevatedButton(
                        onPressed: () => _rejectMerchant(merchant['id'].toString()),
                        style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
                        child: const Text('Reject'),
                      ),
                    ],
                  ),
                ],
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildDetail(String label, String value) {
    return Expanded(
      child: Column(
        children: [
          Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])),
          Text(value, style: const TextStyle(fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }

  Color _getStatusColor(String status) {
    switch (status) {
      case 'approved':
        return Colors.green;
      case 'pending':
        return Colors.orange;
      case 'rejected':
        return Colors.red;
      default:
        return Colors.grey;
    }
  }
}
