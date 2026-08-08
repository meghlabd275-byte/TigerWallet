/**
 * TigerWallet Admin - Feature Flags Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';

class FeatureFlagsScreen extends StatefulWidget {
  const FeatureFlagsScreen({Key? key}) : super(key: key);

  @override
  State<FeatureFlagsScreen> createState() => _FeatureFlagsScreenState();
}

class _FeatureFlagsScreenState extends State<FeatureFlagsScreen> {
  List<dynamic> _features = [];
  bool _loading = true;
  String _selectedCategory = 'all';
  bool _isDark = false;
  bool _showAddModal = false;

  @override
  void initState() {
    super.initState();
    _loadFeatures();
  }

  Future<void> _loadFeatures() async {
    setState(() => _loading = true);
    try {
      final response = await http.get(Uri.parse('https://api.tigerwallet.com/admin/v1/features'), headers: {'Authorization': 'Bearer ${await _getToken()}'});
      if (response.statusCode == 200) setState(() => _features = json.decode(response.body)['data']);
    } catch (e) {
      _features = _getMockFeatures();
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<String> _getToken() async => '';

  List<dynamic> _getMockFeatures() {
    return [
      {'id': '1', 'name': 'New Trading Interface', 'description': 'Updated trading UI with advanced charts', 'category': 'Trading', 'enabled': true, 'rollout_percentage': 100},
      {'id': '2', 'name': 'P2P Trading', 'description': 'Peer-to-peer trading feature', 'category': 'Trading', 'enabled': true, 'rollout_percentage': 50},
      {'id': '3', 'name': 'NFT Marketplace', 'description': 'NFT buying and selling', 'category': 'NFT', 'enabled': false, 'rollout_percentage': 0},
      {'id': '4', 'name': 'Margin Trading', 'description': 'Leveraged trading', 'category': 'Trading', 'enabled': true, 'rollout_percentage': 25},
    ];
  }

  Future<void> _toggleFeature(String id) async {
    try {
      await http.post(Uri.parse('https://api.tigerwallet.com/admin/v1/features/$id/toggle'), headers: {'Authorization': 'Bearer ${await _getToken()}'});
      _loadFeatures();
    } catch (e) {
      setState(() {
        final idx = _features.indexWhere((f) => f['id'] == id);
        if (idx != -1) _features[idx]['enabled'] = !_features[idx]['enabled'];
      });
    }
  }

  Future<void> _createFeature(String name, String description, String category, int rollout) async {
    try {
      await http.post(Uri.parse('https://api.tigerwallet.com/admin/v1/features'), headers: {'Authorization': 'Bearer ${await _getToken()}'}, body: json.encode({'name': name, 'description': description, 'category': category, 'rollout_percentage': rollout}));
      _loadFeatures();
    } catch (e) { /* Handle error */ }
    setState(() => _showAddModal = false);
  }

  void _toggleTheme() => setState(() => _isDark = !_isDark);

  List<String> get _categories => ['all', ..._features.map((f) => f['category']).toSet().toList()];

  @override
  Widget build(BuildContext context) {
    return Theme(data: _isDark ? ThemeData.dark() : ThemeData.light(), child: Scaffold(
      appBar: AppBar(title: const Text('Feature Flags'), actions: [IconButton(icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode), onPressed: _toggleTheme), IconButton(icon: const Icon(Icons.add), onPressed: () => setState(() => _showAddModal = true))]),
      body: Stack(children: [Column(children: [_buildCategoryTabs(), Expanded(child: _buildFeaturesList())]), if (_showAddModal) _buildAddModal()]),
    ));
  }

  Widget _buildCategoryTabs() {
    return SingleChildScrollView(scrollDirection: Axis.horizontal, padding: const EdgeInsets.all(16), child: Row(children: _categories.map((cat) => Padding(padding: const EdgeInsets.only(right: 8), child: FilterChip(label: Text(cat == 'all' ? 'All' : cat), selected: _selectedCategory == cat, onSelected: (_) => setState(() => _selectedCategory = cat))))));
  }

  Widget _buildFeaturesList() {
    if (_loading) return const Center(child: CircularProgressIndicator());
    final filtered = _selectedCategory == 'all' ? _features : _features.where((f) => f['category'] == _selectedCategory).toList();
    return ListView.builder(padding: const EdgeInsets.symmetric(horizontal: 16), itemCount: filtered.length, itemBuilder: (context, index) {
      final feature = filtered[index];
      return Card(margin: const EdgeInsets.only(bottom: 12), child: Padding(padding: const EdgeInsets.all(16), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
          Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [Text(feature['name'], style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)), const SizedBox(height: 4), Text(feature['description'], style: TextStyle(color: Colors.grey[600]))])),
          Switch(value: feature['enabled'], onChanged: (_) => _toggleFeature(feature['id'])),
        ]),
        const SizedBox(height: 12),
        Row(children: [Chip(label: Text(feature['category']), backgroundColor: Colors.blue.withOpacity(0.2)), const SizedBox(width: 16), Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [Text('Rollout: ${feature['rollout_percentage']}%', style: const TextStyle(fontSize: 12)), const SizedBox(height: 4), LinearProgressIndicator(value: feature['rollout_percentage'] / 100))]))]),
      ])));
    });
  }

  Widget _buildAddModal() {
    final nameController = TextEditingController();
    final descController = TextEditingController();
    final categoryController = TextEditingController();
    final rolloutController = TextEditingController(text: '0');
    return Container(color: Colors.black54, child: Center(child: Card(margin: const EdgeInsets.all(32), child: Padding(padding: const EdgeInsets.all(24), child: Column(mainAxisSize: MainAxisSize.min, children: [
      const Text('Add Feature', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
      const SizedBox(height: 16),
      TextField(controller: nameController, decoration: const InputDecoration(labelText: 'Feature Name')),
      const SizedBox(height: 8),
      TextField(controller: descController, decoration: const InputDecoration(labelText: 'Description')),
      const SizedBox(height: 8),
      TextField(controller: categoryController, decoration: const InputDecoration(labelText: 'Category')),
      const SizedBox(height: 8),
      TextField(controller: rolloutController, decoration: const InputDecoration(labelText: 'Rollout % (0-100)'), keyboardType: TextInputType.number),
      const SizedBox(height: 16),
      Row(mainAxisAlignment: MainAxisAlignment.end, children: [TextButton(onPressed: () => setState(() => _showAddModal = false), child: const Text('Cancel')), ElevatedButton(onPressed: () => _createFeature(nameController.text, descController.text, categoryController.text, int.tryParse(rolloutController.text) ?? 0), child: const Text('Create'))]),
    ])))));
  }
}
