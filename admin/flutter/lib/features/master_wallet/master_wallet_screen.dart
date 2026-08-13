/**
 * TigerWallet Admin - Master Wallet Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';

class MasterWalletScreen extends StatefulWidget {
  const MasterWalletScreen({Key? key}) : super(key: key);

  @override
  State<MasterWalletScreen> createState() => _MasterWalletScreenState();
}

class _MasterWalletScreenState extends State<MasterWalletScreen> {
  List<Map<String, dynamic>> _wallets = [];
  Map<String, dynamic>? _stats;
  bool _loading = true;
  String? _error;
  bool _isDark = false;

  ApiClient? _api;

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  Future<ApiClient> _client() async {
    if (_api != null) return _api!;
    final prefs = await SharedPreferences.getInstance();
    _api = DioClient.withPrefs(prefs);
    return _api!;
  }

  Future<void> _loadData() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = await _client();
      final results = await Future.wait([
        api.getAdminWallets(),
        api.getAdminStats(),
      ]);
      setState(() {
        _wallets = results[0] as List<Map<String, dynamic>>;
        _stats = results[1] as Map<String, dynamic>;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _wallets = [];
        _stats = null;
      });
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _transfer(String walletId, String toAddress, double amount) async {
    try {
      final api = await _client();
      await api.transferMasterWallet(walletId, toAddress, amount);
      _loadData();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Transfer failed: $e')),
        );
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
          title: const Text('Master Wallets'),
          actions: [
            IconButton(
              icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode),
              onPressed: _toggleTheme,
            ),
            IconButton(icon: const Icon(Icons.add), onPressed: () {}),
          ],
        ),
        body: Column(
          children: [_buildStats(), Expanded(child: _buildBody())],
        ),
      ),
    );
  }

  Widget _buildStats() {
    final totalWallets = _wallets.length;
    return Container(
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          Expanded(child: _buildStatCard('Total Wallets', '$totalWallets')),
          Expanded(
            child: _buildStatCard(
              'Users',
              '${_stats != null && _stats!['total_users'] != null ? _stats!['total_users'] : '—'}',
            ),
          ),
          Expanded(
            child: _buildStatCard(
              'Active Users',
              '${_stats != null && _stats!['active_users'] != null ? _stats!['active_users'] : '—'}',
            ),
          ),
          Expanded(
            child: _buildStatCard(
              'Volume',
              _stats != null && _stats!['total_volume'] != null
                  ? '\$${((_stats!['total_volume'] as num) / 1000000).toStringAsFixed(1)}M'
                  : '—',
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

  Widget _buildBody() {
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
                'Failed to load wallets: $_error',
                textAlign: TextAlign.center,
                style: TextStyle(color: Colors.grey[600]),
              ),
            ),
            const SizedBox(height: 12),
            ElevatedButton(onPressed: _loadData, child: const Text('Retry')),
          ],
        ),
      );
    }
    if (_wallets.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.account_balance_wallet_outlined, size: 48, color: Colors.grey),
            const SizedBox(height: 12),
            Text('No wallets found', style: TextStyle(color: Colors.grey[600])),
          ],
        ),
      );
    }
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: _wallets.length,
      itemBuilder: (context, index) {
        final wallet = _wallets[index];
        final name = (wallet['label'] ?? wallet['name'] ?? 'Wallet').toString();
        final address = (wallet['address'] ?? '—').toString();
        final chain = (wallet['chain_id'] ?? wallet['chain'] ?? '—').toString();
        final currency = (wallet['currency'] ?? '').toString();
        final balance = wallet['balance'];
        final status = (wallet['status'] ?? 'active').toString();
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
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(name, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                        const SizedBox(height: 4),
                        Text(address, style: TextStyle(color: Colors.grey[600], fontSize: 12)),
                      ],
                    ),
                    Chip(label: Text(chain)),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    _buildWalletDetail('Chain', chain),
                    _buildWalletDetail(
                      'Balance',
                      balance != null ? '$balance $currency'.trim() : '—',
                    ),
                    _buildWalletDetail('Status', status.toUpperCase()),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    ElevatedButton.icon(
                      icon: const Icon(Icons.send),
                      label: const Text('Transfer'),
                      onPressed: () => _showTransferDialog(wallet),
                    ),
                    const SizedBox(width: 8),
                    OutlinedButton.icon(
                      icon: const Icon(Icons.refresh),
                      label: const Text('Refresh'),
                      onPressed: _loadData,
                    ),
                  ],
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildWalletDetail(String label, String value) {
    return Expanded(
      child: Column(
        children: [
          Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])),
          Text(value, style: const TextStyle(fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }

  void _showTransferDialog(Map<String, dynamic> wallet) {
    final addressController = TextEditingController();
    final amountController = TextEditingController();
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Transfer'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: addressController,
              decoration: const InputDecoration(labelText: 'To Address'),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: amountController,
              decoration: const InputDecoration(labelText: 'Amount'),
              keyboardType: TextInputType.number,
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
          ElevatedButton(
            onPressed: () {
              Navigator.pop(context);
              _transfer(wallet['id'].toString(), addressController.text, double.tryParse(amountController.text) ?? 0);
            },
            child: const Text('Transfer'),
          ),
        ],
      ),
    );
  }
}
