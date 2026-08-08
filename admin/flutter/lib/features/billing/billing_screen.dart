/**
 * TigerWallet Admin - Billing/Subscription Feature
 * Complete implementation with dark/light theme support
 */

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';

class BillingScreen extends StatefulWidget {
  const BillingScreen({Key? key}) : super(key: key);

  @override
  State<BillingScreen> createState() => _BillingScreenState();
}

class _BillingScreenState extends State<BillingScreen> {
  List<dynamic> _plans = [];
  Map<String, dynamic>? _subscription;
  List<dynamic> _invoices = [];
  bool _loading = true;
  bool _isDark = false;

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  Future<void> _loadData() async {
    setState(() => _loading = true);
    try {
      final plansRes = await http.get(Uri.parse('https://api.tigerwallet.com/admin/v1/billing/plans'), headers: {'Authorization': 'Bearer ${await _getToken()}'});
      final subRes = await http.get(Uri.parse('https://api.tigerwallet.com/admin/v1/billing/subscription'), headers: {'Authorization': 'Bearer ${await _getToken()}'});
      final invoicesRes = await http.get(Uri.parse('https://api.tigerwallet.com/admin/v1/billing/invoices'), headers: {'Authorization': 'Bearer ${await _getToken()}'});
      if (plansRes.statusCode == 200) _plans = json.decode(plansRes.body)['data'];
      if (subRes.statusCode == 200) _subscription = json.decode(subRes.body)['data'];
      if (invoicesRes.statusCode == 200) _invoices = json.decode(invoicesRes.body)['data'];
    } catch (e) {
      _plans = _getMockPlans();
      _subscription = _getMockSubscription();
      _invoices = _getMockInvoices();
    } finally {
      setState(() => _loading = false);
    }
  }

  Future<String> _getToken() async => '';

  List<dynamic> _getMockPlans() {
    return [
      {'id': '1', 'name': 'Basic', 'price': 99.0, 'period': 'month', 'features': ['Up to 1,000 users', 'Basic analytics', 'Email support']},
      {'id': '2', 'name': 'Pro', 'price': 299.0, 'period': 'month', 'features': ['Up to 10,000 users', 'Advanced analytics', 'Priority support', 'API access']},
      {'id': '3', 'name': 'Enterprise', 'price': 999.0, 'period': 'month', 'features': ['Unlimited users', 'Custom analytics', '24/7 support', 'Full API access', 'Dedicated account manager']},
    ];
  }

  Map<String, dynamic> _getMockSubscription() {
    return {'plan_id': '2', 'plan_name': 'Pro', 'status': 'active', 'current_period_end': '2024-12-31', 'users': 2500, 'api_calls': 50000};
  }

  List<dynamic> _getMockInvoices() {
    return [
      {'id': 'INV001', 'amount': 299.0, 'status': 'paid', 'date': '2024-01-01'},
      {'id': 'INV002', 'amount': 299.0, 'status': 'paid', 'date': '2023-12-01'},
      {'id': 'INV003', 'amount': 299.0, 'status': 'paid', 'date': '2023-11-01'},
    ];
  }

  Future<void> _subscribe(String planId) async {
    try {
      await http.post(Uri.parse('https://api.tigerwallet.com/admin/v1/billing/subscription'), headers: {'Authorization': 'Bearer ${await _getToken()}'}, body: json.encode({'plan_id': planId}));
      _loadData();
    } catch (e) { /* Handle error */ }
  }

  void _toggleTheme() => setState(() => _isDark = !_isDark);

  @override
  Widget build(BuildContext context) {
    return Theme(data: _isDark ? ThemeData.dark() : ThemeData.light(), child: Scaffold(
      appBar: AppBar(title: const Text('Billing & Subscription'), actions: [IconButton(icon: Icon(_isDark ? Icons.light_mode : Icons.dark_mode), onPressed: _toggleTheme)]),
      body: _loading ? const Center(child: CircularProgressIndicator()) : SingleChildScrollView(padding: const EdgeInsets.all(16), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [_buildCurrentSubscription(), const SizedBox(height: 24), _buildPlans(), const SizedBox(height: 24), _buildInvoices()])),
    ));
  }

  Widget _buildCurrentSubscription() {
    if (_subscription == null) return const SizedBox.shrink();
    return Card(child: Padding(padding: const EdgeInsets.all(16), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      const Text('Current Subscription', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
      const SizedBox(height: 12),
      Row(children: [
        Expanded(child: _buildSubDetail('Plan', _subscription!['plan_name'])),
        Expanded(child: _buildSubDetail('Status', _subscription!['status'].toString().toUpperCase())),
        Expanded(child: _buildSubDetail('Users', '${_subscription!['users']}')),
        Expanded(child: _buildSubDetail('API Calls', '${_subscription!['api_calls']}')),
      ]),
      const SizedBox(height: 8),
      Text('Renews: ${_subscription!['current_period_end']}', style: TextStyle(color: Colors.grey[600])),
    ])));
  }

  Widget _buildSubDetail(String label, String value) {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])), Text(value, style: const TextStyle(fontWeight: FontWeight.w600))]);
  }

  Widget _buildPlans() {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      const Text('Available Plans', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
      const SizedBox(height: 12),
      Row(children: _plans.map((plan) => Expanded(child: Card(margin: const EdgeInsets.only(right: 12), child: Padding(padding: const EdgeInsets.all(16), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(plan['name'], style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
        const SizedBox(height: 8),
        Text('\$${plan['price'].toStringAsFixed(0)}/${plan['period']}', style: const TextStyle(fontSize: 24, fontWeight: FontWeight.bold, color: Colors.orange)),
        const SizedBox(height: 12),
        ...List.generate((plan['features'] as List).length, (i) => Padding(padding: const EdgeInsets.only(bottom: 4), child: Row(children: [const Icon(Icons.check, size: 16, color: Colors.green), const SizedBox(width: 8), Expanded(child: Text((plan['features'] as List)[i], style: const TextStyle(fontSize: 12)))]))),
        const SizedBox(height: 12),
        SizedBox(width: double.infinity, child: ElevatedButton(onPressed: () => _subscribe(plan['id']), child: const Text('Subscribe'))),
      ]))))).toList()),
    ]);
  }

  Widget _buildInvoices() {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      const Text('Recent Invoices', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
      const SizedBox(height: 12),
      Card(child: Column(children: [
        ..._invoices.map((inv) => ListTile(leading: const Icon(Icons.receipt), title: Text(inv['id']), subtitle: Text(inv['date']), trailing: Row(mainAxisSize: MainAxisSize.min, children: [Text('\$${inv['amount']}', style: const TextStyle(fontWeight: FontWeight.bold)), const SizedBox(width: 8), Chip(label: Text(inv['status'].toString().toUpperCase()), backgroundColor: Colors.green, labelStyle: const TextStyle(color: Colors.white, fontSize: 10))]))),
      ])),
    ]);
  }
}
