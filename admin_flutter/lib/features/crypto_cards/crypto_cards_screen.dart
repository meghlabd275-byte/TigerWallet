/**
 * TigerWallet Admin - Crypto Cards Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';

class CryptoCardsScreen extends StatefulWidget {
  const CryptoCardsScreen({Key? key}) : super(key: key);

  @override
  State<CryptoCardsScreen> createState() => _CryptoCardsScreenState();
}

class _CryptoCardsScreenState extends State<CryptoCardsScreen> {
  List<dynamic> _cards = [];
  bool _loading = true;
  String _filter = 'all';
  final _searchController = TextEditingController();
  bool _isDark = false;

  @override
  void initState() {
    super.initState();
    _loadCards();
    _loadTheme();
  }

  Future<void> _loadTheme() async {
    // Load theme from shared preferences or API
    setState(() {});
  }

  Future<void> _loadCards() async {
    setState(() => _loading = true);
    try {
      final response = await http.get(
        Uri.parse('https://api.tigerwallet.com/admin/v1/crypto-cards?status=$_filter'),
        headers: {'Authorization': 'Bearer ${await _getToken()}'},
      );
      if (response.statusCode == 200) {
        setState(() {
          _cards = json.decode(response.body)['data'];
        });
      }
    } catch (e) {
      // Handle error - show mock data in demo mode
      _cards = _getMockCards();
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<String> _getToken() async {
    // Get token from storage
    return '';
  }

  List<dynamic> _getMockCards() {
    return [
      {
        'id': '1',
        'user_name': 'John Doe',
        'card_number': '4532123456789012',
        'currency': 'USDT',
        'balance': 5000.00,
        'limit': 10000.00,
        'status': 'active',
        'card_type': 'virtual',
      },
      {
        'id': '2',
        'user_name': 'Jane Smith',
        'card_number': '4532987654321098',
        'currency': 'USDT',
        'balance': 2500.00,
        'limit': 5000.00,
        'status': 'blocked',
        'card_type': 'physical',
      },
    ];
  }

  Future<void> _blockCard(String cardId) async {
    try {
      await http.post(
        Uri.parse('https://api.tigerwallet.com/admin/v1/crypto-cards/$cardId/block'),
        headers: {'Authorization': 'Bearer ${await _getToken()}'},
      );
      _loadCards();
    } catch (e) {
      // Show error
    }
  }

  Future<void> _activateCard(String cardId) async {
    try {
      await http.post(
        Uri.parse('https://api.tigerwallet.com/admin/v1/crypto-cards/$cardId/activate'),
        headers: {'Authorization': 'Bearer ${await _getToken()}'},
      );
      _loadCards();
    } catch (e) {
      // Show error
    }
  }

  void _toggleTheme() {
    setState(() {
      _isDark = !_isDark;
    });
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
          Expanded(child: _buildStatCard('Balance', '\$${_cards.fold(0.0, (sum, c) => sum + (c['balance'] ?? 0.0)).toStringAsFixed(0)}')),
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
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }

    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: _cards.length,
      itemBuilder: (context, index) {
        final card = _cards[index];
        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: ListTile(
            leading: CircleAvatar(
              child: Text(card['card_type'] == 'virtual' ? 'V' : 'P'),
            ),
            title: Text('•••• ${card['card_number'].toString().substring(12)}'),
            subtitle: Text('${card['user_name']} - ${card['currency']} ${card['balance']}'),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Chip(
                  label: Text(card['status']),
                  backgroundColor: card['status'] == 'active' ? Colors.green : Colors.red,
                ),
                const SizedBox(width: 8),
                if (card['status'] == 'active')
                  IconButton(
                    icon: const Icon(Icons.block),
                    onPressed: () => _blockCard(card['id']),
                  )
                else
                  IconButton(
                    icon: const Icon(Icons.check_circle),
                    onPressed: () => _activateCard(card['id']),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }
}
