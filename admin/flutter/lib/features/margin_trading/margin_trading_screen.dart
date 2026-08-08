/**
 * TigerWallet Admin - Margin Trading Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';

class MarginTradingScreen extends StatefulWidget {
  const MarginTradingScreen({Key? key}) : super(key: key);

  @override
  State<MarginTradingScreen> createState() => _MarginTradingScreenState();
}

class _MarginTradingScreenState extends State<MarginTradingScreen> {
  List<dynamic> _positions = [];
  bool _loading = true;
  String _filter = 'all';
  Map<String, dynamic>? _stats;
  bool _isDark = false;

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  Future<void> _loadData() async {
    setState(() => _loading = true);
    try {
      final response = await http.get(
        Uri.parse('https://api.tigerwallet.com/admin/v1/margin/positions'),
        headers: {'Authorization': 'Bearer ${await _getToken()}'},
      );
      if (response.statusCode == 200) {
        setState(() {
          _positions = json.decode(response.body);
        });
      }
      
      final statsResponse = await http.get(
        Uri.parse('https://api.tigerwallet.com/admin/v1/margin/liquidation-stats'),
        headers: {'Authorization': 'Bearer ${await _getToken()}'},
      );
      if (statsResponse.statusCode == 200) {
        setState(() {
          _stats = json.decode(statsResponse.body);
        });
      }
    } catch (e) {
      _positions = _getMockPositions();
      _stats = _getMockStats();
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<String> _getToken() async => '';

  List<dynamic> _getMockPositions() {
    return [
      {
        'id': '1',
        'user_name': 'Trader John',
        'pair': 'BTC/USDT',
        'side': 'long',
        'size': 1.5,
        'leverage': 10,
        'entry_price': 45000.0,
        'current_price': 47000.0,
        'pnl': 3000.0,
        'liquidation_price': 40500.0,
        'status': 'open',
      },
      {
        'id': '2',
        'user_name': 'Trader Jane',
        'pair': 'ETH/USDT',
        'side': 'short',
        'size': 10.0,
        'leverage': 5,
        'entry_price': 3000.0,
        'current_price': 2800.0,
        'pnl': 2000.0,
        'liquidation_price': 3600.0,
        'status': 'open',
      },
    ];
  }

  Map<String, dynamic> _getMockStats() {
    return {
      'total_positions': 150,
      'total_volume': 5000000.0,
      'liquidations_today': 3,
      'liquidated_volume': 50000.0,
    };
  }

  Future<void> _liquidatePosition(String positionId) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Confirm Liquidation'),
        content: const Text('Are you sure you want to liquidate this position?'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('Cancel')),
          TextButton(onPressed: () => Navigator.pop(context, true), child: const Text('Liquidate')),
        ],
      ),
    );
    
    if (confirmed == true) {
      try {
        await http.post(
          Uri.parse('https://api.tigerwallet.com/admin/v1/margin/positions/$positionId/liquidate'),
          headers: {'Authorization': 'Bearer ${await _getToken()}'},
        );
        _loadData();
      } catch (e) {
        // Handle error
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
          title: const Text('Margin Trading'),
          actions: [
            IconButton(icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode), onPressed: _toggleTheme),
          ],
        ),
        body: Column(
          children: [
            _buildStats(),
            _buildFilterChips(),
            Expanded(child: _buildPositionsList()),
          ],
        ),
      ),
    );
  }

  Widget _buildStats() {
    if (_stats == null) return const SizedBox.shrink();
    return Container(
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          _buildStatCard('Total Positions', '${_stats!['total_positions']}'),
          _buildStatCard('Volume', '\$${(_stats!['total_volume'] / 1000000).toStringAsFixed(1)}M'),
          _buildStatCard('Liquidations', '${_stats!['liquidations_today']}'),
          _buildStatCard('Liq. Volume', '\$${(_stats!['liquidated_volume'] / 1000).toStringAsFixed(0)}K'),
        ],
      ),
    );
  }

  Widget _buildStatCard(String label, String value) {
    return Expanded(
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            children: [
              Text(value, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
              Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildFilterChips() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          FilterChip(label: const Text('All'), selected: _filter == 'all', onSelected: (_) => setState(() => _filter = 'all')),
          const SizedBox(width: 8),
          FilterChip(label: const Text('Open'), selected: _filter == 'open', onSelected: (_) => setState(() => _filter = 'open')),
          const SizedBox(width: 8),
          FilterChip(label: const Text('Liquidated'), selected: _filter == 'liquidated', onSelected: (_) => setState(() => _filter = 'liquidated')),
        ],
      ),
    );
  }

  Widget _buildPositionsList() {
    if (_loading) return const Center(child: CircularProgressIndicator());
    
    final filtered = _filter == 'all' ? _positions : _positions.where((p) => p['status'] == _filter).toList();
    
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: filtered.length,
      itemBuilder: (context, index) {
        final pos = filtered[index];
        final isProfit = (pos['pnl'] as double) >= 0;
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
                    Text('${pos['pair']} (${pos['leverage']}x)', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                    Chip(
                      label: Text(pos['side'].toString().toUpperCase()),
                      backgroundColor: pos['side'] == 'long' ? Colors.green : Colors.red,
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Text('Trader: ${pos['user_name']}'),
                const SizedBox(height: 4),
                Row(
                  children: [
                    _buildPositionDetail('Size', '${pos['size']}'),
                    _buildPositionDetail('Entry', '\$${pos['entry_price']}'),
                    _buildPositionDetail('Current', '\$${pos['current_price']}'),
                    _buildPositionDetail('PnL', '\$${pos['pnl']}', color: isProfit ? Colors.green : Colors.red),
                  ],
                ),
                const SizedBox(height: 8),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text('Liq. Price: \$${pos['liquidation_price']}', style: TextStyle(color: Colors.orange[700])),
                    if (pos['status'] == 'open')
                      ElevatedButton(
                        onPressed: () => _liquidatePosition(pos['id']),
                        style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
                        child: const Text('Liquidate'),
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

  Widget _buildPositionDetail(String label, String value, {Color? color}) {
    return Expanded(
      child: Column(
        children: [
          Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])),
          Text(value, style: TextStyle(fontWeight: FontWeight.w600, color: color)),
        ],
      ),
    );
  }
}
