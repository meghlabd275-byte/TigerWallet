/**
 * TigerWallet Admin - Generic Domain Governance Screen
 * Renders any admin domain (futures, options, copy-trading, convert, onramp,
 * offramp, p2p-clients, partners, rewards, marketing, bots, bots-clients,
 * project-teams, liquidity-sources) backed by admin/go :9093 with real CRUD,
 * status control, and approve/reject where the domain supports it.
 * Dark/light theme toggle included. No mock data.
 */

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/dio_client.dart';

class DomainConfig {
  final String domain;
  final String title;
  final IconData icon;
  final bool supportsStatus;
  final bool supportsApprove;
  final List<String> statusActions;

  const DomainConfig({
    required this.domain,
    required this.title,
    required this.icon,
    this.supportsStatus = true,
    this.supportsApprove = false,
    this.statusActions = const ['start', 'stop', 'pause', 'resume'],
  });
}

const List<DomainConfig> kAdminDomains = [
  DomainConfig(domain: 'futures', title: 'Futures', icon: Icons.trending_up),
  DomainConfig(domain: 'options', title: 'Options', icon: Icons.tune),
  DomainConfig(domain: 'copy-trading', title: 'Copy Trading', icon: Icons.copy),
  DomainConfig(domain: 'convert', title: 'Convert', icon: Icons.swap_vert),
  DomainConfig(domain: 'onramp', title: 'On-Ramp', icon: Icons.login, supportsStatus: false, supportsApprove: true),
  DomainConfig(domain: 'offramp', title: 'Off-Ramp', icon: Icons.logout, supportsStatus: false, supportsApprove: true),
  DomainConfig(domain: 'p2p-clients', title: 'P2P Clients', icon: Icons.people_alt),
  DomainConfig(domain: 'partners', title: 'Partners', icon: Icons.handshake, supportsApprove: true),
  DomainConfig(domain: 'rewards', title: 'Rewards', icon: Icons.card_giftcard),
  DomainConfig(domain: 'marketing', title: 'Marketing', icon: Icons.campaign),
  DomainConfig(domain: 'bots', title: 'Bots', icon: Icons.smart_toy),
  DomainConfig(domain: 'bots-clients', title: 'Bots Clients', icon: Icons.groups),
  DomainConfig(domain: 'project-teams', title: 'Project Teams', icon: Icons.engineering),
  DomainConfig(domain: 'liquidity-sources', title: 'Liquidity Sources', icon: Icons.water_drop),
];

class DomainScreen extends StatefulWidget {
  final DomainConfig config;

  const DomainScreen({Key? key, required this.config}) : super(key: key);

  @override
  State<DomainScreen> createState() => _DomainScreenState();
}

class _DomainScreenState extends State<DomainScreen> {
  List<Map<String, dynamic>> _items = [];
  bool _loading = true;
  String? _error;
  bool _isDark = false;
  ApiClient? _api;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<ApiClient> _client() async {
    if (_api != null) return _api!;
    final prefs = await SharedPreferences.getInstance();
    _api = DioClient.withPrefs(prefs);
    return _api!;
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = await _client();
      _items = await api.getDomainItems(widget.config.domain);
    } catch (e) {
      _items = [];
      _error = e.toString();
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _run(Future<void> Function(ApiClient api) action) async {
    try {
      final api = await _client();
      await action(api);
      _load();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('Action failed: $e')));
      }
    }
  }

  void _showRejectDialog(String id) {
    final controller = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Reject'),
        content: TextField(
          controller: controller,
          decoration: const InputDecoration(labelText: 'Reason'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () {
              Navigator.pop(ctx);
              _run((api) =>
                  api.rejectDomainItem(widget.config.domain, id, controller.text));
            },
            child: const Text('Reject'),
          ),
        ],
      ),
    );
  }

  String _describe(Map<String, dynamic> item) {
    for (final key in ['name', 'title', 'symbol', 'pair', 'email', 'id']) {
      final v = item[key];
      if (v != null && v.toString().isNotEmpty) return v.toString();
    }
    return '(unnamed)';
  }

  @override
  Widget build(BuildContext context) {
    final cfg = widget.config;
    return Theme(
      data: _isDark ? ThemeData.dark() : ThemeData.light(),
      child: Scaffold(
        appBar: AppBar(
          title: Text(cfg.title),
          actions: [
            IconButton(
              icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode),
              onPressed: () => setState(() => _isDark = !_isDark),
            ),
            IconButton(icon: const Icon(Icons.refresh), onPressed: _load),
          ],
        ),
        body: _loading
            ? const Center(child: CircularProgressIndicator())
            : _error != null
                ? Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text('Failed to load: $_error',
                            style: const TextStyle(color: Colors.red)),
                        const SizedBox(height: 8),
                        ElevatedButton(
                            onPressed: _load, child: const Text('Retry')),
                      ],
                    ),
                  )
                : _items.isEmpty
                    ? const Center(child: Text('No records'))
                    : RefreshIndicator(
                        onRefresh: _load,
                        child: ListView.builder(
                          itemCount: _items.length,
                          itemBuilder: (ctx, i) {
                            final item = _items[i];
                            final id = item['id']?.toString() ?? '';
                            final status = item['status']?.toString() ?? '';
                            return Card(
                              margin: const EdgeInsets.symmetric(
                                  horizontal: 12, vertical: 6),
                              child: ListTile(
                                leading: Icon(cfg.icon),
                                title: Text(_describe(item)),
                                subtitle: Text(status.isEmpty
                                    ? (item['id']?.toString() ?? '')
                                    : 'status: $status'),
                                trailing: PopupMenuButton<String>(
                                  onSelected: (action) {
                                    if (action == 'approve') {
                                      _run((api) => api.approveDomainItem(
                                          cfg.domain, id));
                                    } else if (action == 'reject') {
                                      _showRejectDialog(id);
                                    } else if (action == 'delete') {
                                      _run((api) =>
                                          api.deleteDomainItem(cfg.domain, id));
                                    } else {
                                      _run((api) => api.setDomainStatus(
                                          cfg.domain, id, action));
                                    }
                                  },
                                  itemBuilder: (ctx) => [
                                    if (cfg.supportsStatus)
                                      ...cfg.statusActions.map((a) =>
                                          PopupMenuItem(
                                              value: a,
                                              child: Text(
                                                  'Set status: $a'))),
                                    if (cfg.supportsApprove)
                                      const PopupMenuItem(
                                          value: 'approve',
                                          child: Text('Approve')),
                                    if (cfg.supportsApprove)
                                      const PopupMenuItem(
                                          value: 'reject',
                                          child: Text('Reject')),
                                    const PopupMenuItem(
                                        value: 'delete', child: Text('Delete')),
                                  ],
                                ),
                              ),
                            );
                          },
                        ),
                      ),
      ),
    );
  }
}
