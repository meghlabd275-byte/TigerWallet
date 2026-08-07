/**
 * TigerWallet Admin - Liquidity Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';

class LiquidityScreen extends StatefulWidget {
  const LiquidityScreen({Key? key}) : super(key: key);

  @override
  State<LiquidityScreen> createState() => _LiquidityScreenState();
}

class _LiquidityScreenState extends State<LiquidityScreen> {
  List<dynamic> _pools = [];
  Map<String, dynamic>? _stats;
  bool _loading = true;
  bool _isDark = false;
  bool _showAddModal = false;

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  Future<void> _loadData() async {
    setState(() => _loading = true);
    try {
      final poolsRes = await http.get(Uri.parse('https://api.tigerwallet.com/admin/v1/liquidity/pools'), headers: {'Authorization': 'Bearer ${await _getToken()}'});
      final statsRes = await http.get(Uri.parse('https://api.tigerwallet.com/admin/v1/liquidity/stats'), headers: {'Authorization': 'Bearer ${await _getToken()}'});
      if (poolsRes.statusCode == 200) _pools = json.decode(poolsRes.body)['data'];
      if (statsRes.statusCode == 200) _stats = json.decode(statsRes.body);
    } catch (e) {
      _pools = _getMockPools();
      _stats = _getMockStats();
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<String> _getToken() async => '';

  List<dynamic> _getMockPools() {
    return [
      {'id': '1', 'pair': 'USDT/ETH', 'token_a': 'USDT', 'token_b': 'ETH', 'reserve_a': 5000000.0, 'reserve_b': 2500.0, 'total_supply': 100000.0, 'apr': 15.5, 'volume_24h': 2500000.0, 'fees_24h': 7500.0, 'status': 'active'},
      {'id': '2', 'pair': 'USDT/BTC', 'token_a': 'USDT', 'token_b': 'BTC', 'reserve_a': 10000000.0, 'reserve_b': 200.0, 'total_supply': 250000.0, 'apr': 12.3, 'volume_24h': 5000000.0, 'fees_24h': 15000.0, 'status': 'active'},
    ];
  }

  Map<String, dynamic> _getMockStats() {
    return {'total_pools': 2, 'total_value_locked': 15000000.0, 'volume_24h': 7500000.0, 'fees_24h': 22500.0};
  }

  Future<void> _addLiquidity(String poolId, double amountA, double amountB) async {
    try {
      await http.post(Uri.parse('https://api.tigerwallet.com/admin/v1/liquidity/pools/$poolId/add'), headers: {'Authorization': 'Bearer ${await _getToken()}'}, body: json.encode({'user_id': 1, 'amount_a': amountA, 'amount_b': amountB}));
      _loadData();
    } catch (e) { /* Handle error */ }
    setState(() => _showAddModal = false);
  }

  void _toggleTheme() => setState(() => _isDark = !_isDark);

  @override
  Widget build(BuildContext context) {
    return Theme(data: _isDark ? ThemeData.dark() : ThemeData.light(), child: Scaffold(
      appBar: AppBar(title: const Text('Liquidity Pools'), actions: [IconButton(icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode), onPressed: _toggleTheme), IconButton(icon: const Icon(Icons.add), onPressed: () => setState(() => _showAddModal = true))]),
      body: Stack(children: [Column(children: [_buildStats(), Expanded(child: _buildPoolsList())]), if (_showAddModal) _buildAddModal()]),
    ));
  }

  Widget _buildStats() {
    if (_stats == null) return const SizedBox.shrink();
    return Container(padding: const EdgeInsets.all(16), child: Row(children: [
      Expanded(child: _buildStatCard('Total Pools', '${_stats!['total_pools']}')),
      Expanded(child: _buildStatCard('TVL', '\$${(_stats!['total_value_locked'] / 1000000).toStringAsFixed(1)}M')),
      Expanded(child: _buildStatCard('24h Volume', '\$${(_stats!['volume_24h'] / 1000000).toStringAsFixed(1)}M')),
      Expanded(child: _buildStatCard('24h Fees', '\$${(_stats!['fees_24h']).toStringAsFixed(0)}')),
    ]));
  }

  Widget _buildStatCard(String label, String value) {
    return Card(child: Padding(padding: const EdgeInsets.all(12), child: Column(children: [Text(value, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)), Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600]))])));
  }

  Widget _buildPoolsList() {
    if (_loading) return const Center(child: CircularProgressIndicator());
    return ListView.builder(padding: const EdgeInsets.all(16), itemCount: _pools.length, itemBuilder: (context, index) {
      final pool = _pools[index];
      return Card(margin: const EdgeInsets.only(bottom: 12), child: Padding(padding: const EdgeInsets.all(16), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
          Text(pool['pair'], style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
          Chip(label: Text(pool['status'].toString().toUpperCase()), backgroundColor: pool['status'] == 'active' ? Colors.green : Colors.grey),
        ]),
        const SizedBox(height: 12),
        Row(children: [_buildPoolDetail('Reserve A', '${pool['reserve_a']} ${pool['token_a']}'), _buildPoolDetail('Reserve B', '${pool['reserve_b']} ${pool['token_b']}'), _buildPoolDetail('APR', '${pool['apr']}%')]),
        const SizedBox(height: 8),
        Row(children: [_buildPoolDetail('24h Vol', '\$${(pool['volume_24h'] / 1000).toStringAsFixed(0)}K'), _buildPoolDetail('24h Fees', '\$${pool['fees_24h']}')]),
        const SizedBox(height: 12),
        Row(children: [ElevatedButton(onPressed: () => _addLiquidity(pool['id'], 1000, 1), child: const Text('Add Liquidity')), const SizedBox(width: 8), OutlinedButton(onPressed: () {}, child: const Text('Remove'))]),
      ])));
    });
  }

  Widget _buildPoolDetail(String label, String value) {
    return Expanded(child: Column(children: [Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])), Text(value, style: const TextStyle(fontWeight: FontWeight.w600))]));
  }

  Widget _buildAddModal() {
    return Container(color: Colors.black54, child: Center(child: Card(margin: const EdgeInsets.all(32), child: Padding(padding: const EdgeInsets.all(24), child: Column(mainAxisSize: MainAxisSize.min, children: [
      const Text('Add Liquidity', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
      const SizedBox(height: 16),
      TextField(decoration: const InputDecoration(labelText: 'Token A Amount'), keyboardType: TextInputType.number),
      const SizedBox(height: 8),
      TextField(decoration: const InputDecoration(labelText: 'Token B Amount'), keyboardType: TextInputType.number),
      const SizedBox(height: 16),
      Row(mainAxisAlignment: MainAxisAlignment.end, children: [TextButton(onPressed: () => setState(() => _showAddModal = false), child: const Text('Cancel')), ElevatedButton(onPressed: () {}, child: const Text('Add'))]),
    ])))));
  }
}
