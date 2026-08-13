/**
 * TigerWallet Admin - Margin Trading Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';

class MarginTradingScreen extends StatefulWidget {
  const MarginTradingScreen({Key? key}) : super(key: key);

  @override
  State<MarginTradingScreen> createState() => _MarginTradingScreenState();
}

class _MarginTradingScreenState extends State<MarginTradingScreen> {
  List<Map<String, dynamic>> _positions = [];
  Map<String, dynamic>? _liquidationStats;
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
        api.getAdminMarginPositions(),
        api.getAdminMarginLiquidationStats().catchError((_) => <String, dynamic>{}),
      ]);
      setState(() {
        _positions = results[0] as List<Map<String, dynamic>>;
        _liquidationStats = results[1] as Map<String, dynamic>?;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        _positions = [];
        _liquidationStats = null;
      });
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _liquidatePosition(String id) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Confirm Liquidation'),
        content: const Text('Are you sure you want to liquidate this position?'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('Cancel')),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Liquidate', style: TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );
    if (confirm != true) return;
    try {
      final api = await _client();
      await api.liquidateMarginPosition(id);
      _loadData();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Liquidation failed: $e')));
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
          actions: [IconButton(icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode), onPressed: _toggleTheme)],
        ),
        body: Column(
          children: [
            if (_liquidationStats != null && _liquidationStats!.isNotEmpty) _buildStats(),
            Expanded(child: _buildPositionsList()),
          ],
        ),
      ),
    );
  }

  Widget _buildStats() {
    final stats = _liquidationStats!;
    return Container(
      padding: const EdgeInsets.all(16),
      child: Row(
        children: [
          Expanded(child: _buildStatCard('Open', '${stats['open_positions'] ?? 0}')),
          Expanded(child: _buildStatCard('Liquidations', '${stats['liquidations_24h'] ?? 0}')),
          Expanded(
            child: _buildStatCard(
              'Borrowed',
              stats['total_borrowed'] != null ? '\$${((stats['total_borrowed'] as num) / 1000000).toStringAsFixed(1)}M' : '—',
            ),
          ),
          Expanded(
            child: _buildStatCard(
              'Risk',
              stats['system_risk_level']?.toString() ?? '—',
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

  Widget _buildPositionsList() {
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
                'Failed to load margin positions: $_error',
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
    if (_positions.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.trending_down, size: 48, color: Colors.grey),
            const SizedBox(height: 12),
            Text('No margin positions found', style: TextStyle(color: Colors.grey[600])),
          ],
        ),
      );
    }
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: _positions.length,
      itemBuilder: (context, index) {
        final position = _positions[index];
        final symbol = '${position['symbol'] ?? '—'}';
        final side = position['side']?.toString() ?? '';
        final leverage = position['leverage'];
        final liquidationPrice = position['liquidation_price'];
        final status = position['status']?.toString() ?? '';
        final risk = position['risk_level']?.toString() ?? 'low';
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
                    Row(
                      children: [
                        Text(symbol, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                        const SizedBox(width: 8),
                        Chip(
                          label: Text(side.toUpperCase()),
                          backgroundColor: side == 'long' ? Colors.green : Colors.red,
                        ),
                      ],
                    ),
                    Chip(
                      label: Text(status.toUpperCase()),
                      backgroundColor: _getStatusColor(status),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    _buildDetail('User', '${position['user_id'] ?? '—'}'),
                    _buildDetail('Leverage', leverage != null ? '${leverage}x' : '—'),
                    _buildDetail(
                      'Size',
                      position['size'] != null ? '\$${(position['size'] as num).toStringAsFixed(0)}' : '—',
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    _buildDetail(
                      'Entry',
                      position['entry_price'] != null ? '\$${position['entry_price']}' : '—',
                    ),
                    _buildDetail(
                      'Liq. Price',
                      liquidationPrice != null ? '\$${liquidationPrice}' : '—',
                    ),
                    _buildDetail('PNL', position['pnl'] != null ? '${position['pnl']}' : '—'),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: LinearProgressIndicator(
                        value: _getRiskValue(risk),
                        color: _getRiskColor(risk),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Text('Risk: ${risk.toUpperCase()}', style: const TextStyle(fontSize: 12)),
                    const SizedBox(width: 16),
                    if (status == 'open')
                      OutlinedButton(
                        onPressed: () => _liquidatePosition(position['id'].toString()),
                        style: OutlinedButton.styleFrom(foregroundColor: Colors.red),
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

  double _getRiskValue(String risk) {
    switch (risk.toLowerCase()) {
      case 'high':
        return 0.9;
      case 'medium':
        return 0.5;
      default:
        return 0.2;
    }
  }

  Color _getRiskColor(String risk) {
    switch (risk.toLowerCase()) {
      case 'high':
        return Colors.red;
      case 'medium':
        return Colors.orange;
      default:
        return Colors.green;
    }
  }

  Color _getStatusColor(String status) {
    switch (status) {
      case 'open':
        return Colors.green;
      case 'liquidated':
        return Colors.red;
      case 'closed':
        return Colors.grey;
      default:
        return Colors.grey;
    }
  }
}
