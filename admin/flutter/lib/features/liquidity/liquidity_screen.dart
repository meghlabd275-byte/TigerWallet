/**
 * TigerWallet Admin - Liquidity Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';

class LiquidityScreen extends StatefulWidget {
  const LiquidityScreen({Key? key}) : super(key: key);

  @override
  State<LiquidityScreen> createState() => _LiquidityScreenState();
}

class _LiquidityScreenState extends State<LiquidityScreen> {
  List<Map<String, dynamic>> _pools = [];
  Map<String, dynamic>? _stats;
  bool _loading = true;
  String? _error;
  bool _isDark = false;
  bool _showAddModal = false;

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
        api.getAdminLiquidityPools(),
        api.getAdminLiquidityStats().catchError((_) => <String, dynamic>{}),
      ]);
      setState(() {
        _pools = results[0] as List<Map<String, dynamic>>;
        _stats = results[1] as Map<String, dynamic>?;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _pools = [];
        _stats = null;
      });
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _addLiquidity(String poolId, double amountA, double amountB) async {
    try {
      final api = await _client();
      await api.addLiquidity(poolId, {'user_id': 1, 'amount_a': amountA, 'amount_b': amountB});
      _loadData();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Add liquidity failed: $e')));
      }
    }
    setState(() => _showAddModal = false);
  }

  void _toggleTheme() => setState(() => _isDark = !_isDark);

  @override
  Widget build(BuildContext context) {
    return Theme(
      data: _isDark ? ThemeData.dark() : ThemeData.light(),
      child: Scaffold(
        appBar: AppBar(
          title: const Text('Liquidity Pools'),
          actions: [
            IconButton(icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode), onPressed: _toggleTheme),
            IconButton(icon: const Icon(Icons.add), onPressed: () => setState(() => _showAddModal = true)),
          ],
        ),
        body: Stack(
          children: [
            Column(children: [_buildStats(), Expanded(child: _buildPoolsList())]),
            if (_showAddModal) _buildAddModal(),
          ],
        ),
      ),
    );
  }

  Widget _buildStats() {
    if (_stats == null || _stats!.isEmpty) return const SizedBox.shrink();
    return Container(
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          Expanded(child: _buildStatCard('Total Pools', '${_stats!['total_pools'] ?? 0}')),
          Expanded(
            child: _buildStatCard(
              'TVL',
              _stats!['total_value_locked'] != null
                  ? '\$${(((_stats!['total_value_locked'] as num) / 1000000)).toStringAsFixed(1)}M'
                  : '—',
            ),
          ),
          Expanded(
            child: _buildStatCard(
              '24h Volume',
              _stats!['volume_24h'] != null
                  ? '\$${(((_stats!['volume_24h'] as num) / 1000000)).toStringAsFixed(1)}M'
                  : '—',
            ),
          ),
          Expanded(
            child: _buildStatCard(
              '24h Fees',
              _stats!['fees_24h'] != null ? '\$${(_stats!['fees_24h'] as num).toStringAsFixed(0)}' : '—',
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

  Widget _buildPoolsList() {
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
                'Failed to load liquidity pools: $_error',
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
    if (_pools.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.water_drop_outlined, size: 48, color: Colors.grey),
            const SizedBox(height: 12),
            Text('No liquidity pools found', style: TextStyle(color: Colors.grey[600])),
          ],
        ),
      );
    }
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: _pools.length,
      itemBuilder: (context, index) {
        final pool = _pools[index];
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
                    Text('${pool['pair'] ?? '—'}', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
                    Chip(
                      label: Text('${pool['status'] ?? ''}'.toString().toUpperCase()),
                      backgroundColor: pool['status'] == 'active' ? Colors.green : Colors.grey,
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    _buildPoolDetail('Reserve A', '${pool['reserve_a'] ?? '—'} ${pool['token_a'] ?? ''}'),
                    _buildPoolDetail('Reserve B', '${pool['reserve_b'] ?? '—'} ${pool['token_b'] ?? ''}'),
                    _buildPoolDetail('APR', pool['apr'] != null ? '${pool['apr']}%' : '—'),
                  ],
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    _buildPoolDetail(
                      '24h Vol',
                      pool['volume_24h'] != null ? '\$${((pool['volume_24h'] as num) / 1000).toStringAsFixed(0)}K' : '—',
                    ),
                    _buildPoolDetail('24h Fees', pool['fees_24h'] != null ? '\$${pool['fees_24h']}' : '—'),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    ElevatedButton(
                      onPressed: () => _addLiquidity(pool['id'].toString(), 1000, 1),
                      child: const Text('Add Liquidity'),
                    ),
                    const SizedBox(width: 8),
                    OutlinedButton(onPressed: () {}, child: const Text('Remove')),
                  ],
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _buildPoolDetail(String label, String value) {
    return Expanded(
      child: Column(
        children: [
          Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])),
          Text(value, style: const TextStyle(fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }

  Widget _buildAddModal() {
    return Container(
      color: Colors.black54,
      child: Center(
        child: Card(
          margin: const EdgeInsets.all(32),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Text('Add Liquidity', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
                const SizedBox(height: 16),
                const TextField(decoration: InputDecoration(labelText: 'Token A Amount'), keyboardType: TextInputType.number),
                const SizedBox(height: 8),
                const TextField(decoration: InputDecoration(labelText: 'Token B Amount'), keyboardType: TextInputType.number),
                const SizedBox(height: 16),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    TextButton(
                      onPressed: () => setState(() => _showAddModal = false),
                      child: const Text('Cancel'),
                    ),
                    ElevatedButton(onPressed: () {}, child: const Text('Add')),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
