/**
 * TigerWallet Admin - Crypto Cards Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';

class CryptoCardsScreen extends StatefulWidget {
  const CryptoCardsScreen({Key? key}) : super(key: key);

  @override
  State<CryptoCardsScreen> createState() => _CryptoCardsScreenState();
}

class _CryptoCardsScreenState extends State<CryptoCardsScreen> {
  List<Map<String, dynamic>> _cards = [];
  bool _loading = true;
  String? _error;
  String _filter = 'all';
  final _searchController = TextEditingController();
  bool _isDark = false;

  ApiClient? _api;

  @override
  void initState() {
    super.initState();
    _loadCards();
  }

  Future<ApiClient> _client() async {
    if (_api != null) return _api!;
    final prefs = await SharedPreferences.getInstance();
    _api = DioClient.withPrefs(prefs);
    return _api!;
  }

  Future<void> _loadCards() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = await _client();
      _cards = await api.getAdminCryptoCards(status: _filter);
      setState(() {});
    } catch (e) {
      setState(() {
        _error = e.toString();
        _cards = [];
      });
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _blockCard(String cardId) async {
    try {
      final api = await _client();
      await api.blockCryptoCard(cardId);
      _loadCards();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Block failed: $e')));
      }
    }
  }

  Future<void> _activateCard(String cardId) async {
    try {
      final api = await _client();
      await api.activateCryptoCard(cardId);
      _loadCards();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Activate failed: $e')));
      }
    }
  }

  void _toggleTheme() {
    setState(() => _isDark = !_isDark);
  }

  @override
  Widget build(BuildContext context) {
    return Theme(
      data: _isDark ? ThemeData.dark() : ThemeData.light(),
      child: Scaffold(
        appBar: AppBar(
          title: const Text('Crypto Cards'),
          actions: [
            IconButton(
              icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode),
              onPressed: _toggleTheme,
            ),
          ],
        ),
        body: Column(
          children: [
            _buildStats(),
            _buildFilters(),
            Expanded(child: _buildCardsList()),
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
          Expanded(child: _buildStatCard('Total Cards', '${_cards.length}')),
          Expanded(child: _buildStatCard('Active', '${_cards.where((c) => c['status'] == 'active').length}')),
          Expanded(
            child: _buildStatCard(
              'Balance',
              '\$${_cards.fold(0.0, (sum, c) => sum + ((c['balance'] as num?) ?? 0.0)).toStringAsFixed(0)}',
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildStatCard(String label, String value) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            Text(value, style: const TextStyle(fontSize: 24, fontWeight: FontWeight.bold)),
            Text(label, style: TextStyle(color: Colors.grey[600])),
          ],
        ),
      ),
    );
  }

  Widget _buildFilters() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          Expanded(
            child: DropdownButton<String>(
              value: _filter,
              onChanged: (value) {
                setState(() => _filter = value!);
                _loadCards();
              },
              items: const [
                DropdownMenuItem(value: 'all', child: Text('All')),
                DropdownMenuItem(value: 'active', child: Text('Active')),
                DropdownMenuItem(value: 'blocked', child: Text('Blocked')),
                DropdownMenuItem(value: 'pending', child: Text('Pending')),
              ],
            ),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: TextField(
              controller: _searchController,
              decoration: const InputDecoration(
                hintText: 'Search...',
                border: OutlineInputBorder(),
              ),
              onSubmitted: (_) => _loadCards(),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildCardsList() {
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
                'Failed to load crypto cards: $_error',
                textAlign: TextAlign.center,
                style: TextStyle(color: Colors.grey[600]),
              ),
            ),
            const SizedBox(height: 12),
            ElevatedButton(onPressed: _loadCards, child: const Text('Retry')),
          ],
        ),
      );
    }
    if (_cards.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.credit_card_outlined, size: 48, color: Colors.grey),
            const SizedBox(height: 12),
            Text('No crypto cards found', style: TextStyle(color: Colors.grey[600])),
          ],
        ),
      );
    }
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: _cards.length,
      itemBuilder: (context, index) {
        final card = _cards[index];
        final cardNumber = card['card_number']?.toString() ?? '—';
        final masked = cardNumber.length >= 4 ? cardNumber.substring(cardNumber.length - 4) : cardNumber;
        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: ListTile(
            leading: CircleAvatar(child: Text(card['card_type'] == 'virtual' ? 'V' : 'P')),
            title: Text('•••• $masked'),
            subtitle: Text('${card['user_name'] ?? '—'} - ${card['currency'] ?? ''} ${card['balance'] ?? ''}'),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Chip(
                  label: Text('${card['status'] ?? ''}'),
                  backgroundColor: card['status'] == 'active' ? Colors.green : Colors.red,
                ),
                const SizedBox(width: 8),
                if (card['status'] == 'active')
                  IconButton(icon: const Icon(Icons.block), onPressed: () => _blockCard(card['id'].toString()))
                else
                  IconButton(
                    icon: const Icon(Icons.check_circle),
                    onPressed: () => _activateCard(card['id'].toString()),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }
}
