/**
 * TigerWallet Admin - Feature Flags Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';

class FeatureFlagsScreen extends StatefulWidget {
  const FeatureFlagsScreen({Key? key}) : super(key: key);

  @override
  State<FeatureFlagsScreen> createState() => _FeatureFlagsScreenState();
}

class _FeatureFlagsScreenState extends State<FeatureFlagsScreen> {
  List<Map<String, dynamic>> _features = [];
  bool _loading = true;
  String? _error;
  String _selectedCategory = 'all';
  bool _isDark = false;
  bool _showAddModal = false;

  ApiClient? _api;

  @override
  void initState() {
    super.initState();
    _loadFeatures();
  }

  Future<ApiClient> _client() async {
    if (_api != null) return _api!;
    final prefs = await SharedPreferences.getInstance();
    _api = DioClient.withPrefs(prefs);
    return _api!;
  }

  Future<void> _loadFeatures() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = await _client();
      _features = await api.getAdminFeatureFlags();
      setState(() {});
    } catch (e) {
      setState(() {
        _error = e.toString();
        _features = [];
      });
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<void> _toggleFeature(String id) async {
    try {
      final api = await _client();
      await api.toggleFeatureFlag(id);
      _loadFeatures();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Toggle failed: $e')));
      }
    }
  }

  Future<void> _createFeature(String name, String description, String category, int rollout) async {
    try {
      final api = await _client();
      await api.createFeatureFlag2({
        'name': name,
        'description': description,
        'category': category,
        'rollout_percentage': rollout,
      });
      _loadFeatures();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Create failed: $e')));
      }
    }
    setState(() => _showAddModal = false);
  }

  void _toggleTheme() => setState(() => _isDark = !_isDark);

  List<String> get _categories => ['all', ..._features.map((f) => f['category']?.toString() ?? '').toSet().toList()];

  @override
  Widget build(BuildContext context) {
    return Theme(
      data: _isDark ? ThemeData.dark() : ThemeData.light(),
      child: Scaffold(
        appBar: AppBar(
          title: const Text('Feature Flags'),
          actions: [
            IconButton(icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode), onPressed: _toggleTheme),
            IconButton(icon: const Icon(Icons.add), onPressed: () => setState(() => _showAddModal = true)),
          ],
        ),
        body: Stack(
          children: [
            Column(children: [_buildCategoryTabs(), Expanded(child: _buildFeaturesList())]),
            if (_showAddModal) _buildAddModal(),
          ],
        ),
      ),
    );
  }

  Widget _buildCategoryTabs() {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.all(16),
      child: Row(
        children: _categories.map((cat) {
          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: FilterChip(
              label: Text(cat == 'all' ? 'All' : cat),
              selected: _selectedCategory == cat,
              onSelected: (_) => setState(() => _selectedCategory = cat),
            ),
          );
        }).toList(),
      ),
    );
  }

  Widget _buildFeaturesList() {
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
                'Failed to load feature flags: $_error',
                textAlign: TextAlign.center,
                style: TextStyle(color: Colors.grey[600]),
              ),
            ),
            const SizedBox(height: 12),
            ElevatedButton(onPressed: _loadFeatures, child: const Text('Retry')),
          ],
        ),
      );
    }
    final filtered = _selectedCategory == 'all'
        ? _features
        : _features.where((f) => (f['category']?.toString() ?? '') == _selectedCategory).toList();
    if (filtered.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.flag_outlined, size: 48, color: Colors.grey),
            const SizedBox(height: 12),
            Text('No feature flags found', style: TextStyle(color: Colors.grey[600])),
          ],
        ),
      );
    }
    return ListView.builder(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      itemCount: filtered.length,
      itemBuilder: (context, index) {
        final feature = filtered[index];
        final rollout = (feature['rollout_percentage'] as num?) ?? 0;
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
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('${feature['name'] ?? ''}', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                          const SizedBox(height: 4),
                          Text('${feature['description'] ?? ''}', style: TextStyle(color: Colors.grey[600])),
                        ],
                      ),
                    ),
                    Switch(
                      value: feature['enabled'] == true,
                      onChanged: (_) => _toggleFeature(feature['id'].toString()),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Chip(label: Text('${feature['category'] ?? ''}'), backgroundColor: Colors.blue.withOpacity(0.2)),
                    const SizedBox(width: 16),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('Rollout: $rollout%', style: const TextStyle(fontSize: 12)),
                          const SizedBox(height: 4),
                          LinearProgressIndicator(value: rollout / 100),
                        ],
                      ),
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

  Widget _buildAddModal() {
    final nameController = TextEditingController();
    final descController = TextEditingController();
    final categoryController = TextEditingController();
    final rolloutController = TextEditingController(text: '0');
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
                const Text('Add Feature', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
                const SizedBox(height: 16),
                TextField(controller: nameController, decoration: const InputDecoration(labelText: 'Feature Name')),
                const SizedBox(height: 8),
                TextField(controller: descController, decoration: const InputDecoration(labelText: 'Description')),
                const SizedBox(height: 8),
                TextField(controller: categoryController, decoration: const InputDecoration(labelText: 'Category')),
                const SizedBox(height: 8),
                TextField(
                  controller: rolloutController,
                  decoration: const InputDecoration(labelText: 'Rollout % (0-100)'),
                  keyboardType: TextInputType.number,
                ),
                const SizedBox(height: 16),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    TextButton(
                      onPressed: () => setState(() => _showAddModal = false),
                      child: const Text('Cancel'),
                    ),
                    ElevatedButton(
                      onPressed: () => _createFeature(
                        nameController.text,
                        descController.text,
                        categoryController.text,
                        int.tryParse(rolloutController.text) ?? 0,
                      ),
                      child: const Text('Create'),
                    ),
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
