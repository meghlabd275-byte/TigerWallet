// MasterWallet Treasury Service - Flutter
//
// Thin REST client over the canonical Go backend (:8450). Treasury routes
// are nested under /api/v1/master-wallet/:id/treasury. No fabricated data;
// on backend failure methods throw rather than returning fake balances.

import 'dart:convert';
import 'package:http/http.dart' as http;

class TreasuryService {
  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );
  static const String _apiV1 = '$API_BASE/api/v1';

  final String masterWalletId;
  String? _token;

  TreasuryService({required this.masterWalletId, String? token}) : _token = token;

  void setToken(String? token) => _token = token;

  String get _base => '$_apiV1/master-wallet/$masterWalletId/treasury';

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Exception _err(http.Response r, String op) =>
      Exception('treasury $op failed (${r.statusCode}): ${r.body}');

  /// Get treasury overview (real balances from the backend).
  Future<TreasuryOverview> getOverview() async {
    final r = await http.get(Uri.parse(_base), headers: _headers);
    if (r.statusCode != 200) throw _err(r, 'overview');
    final data = json.decode(r.body);
    return TreasuryOverview.fromJson((data['data'] ?? data) as Map<String, dynamic>);
  }

  /// Get treasury balances (real, from the backend). The canonical contract
  /// exposes a single treasury overview at GET /treasury (which carries the real
  /// balances); there is no separate /treasury/balances route, so this derives
  /// the balance list from the overview response. Throws on any backend error.
  Future<List<TreasuryBalance>> getBalances() async {
    final r = await http.get(Uri.parse(_base), headers: _headers);
    if (r.statusCode != 200) throw _err(r, 'balances');
    final data = json.decode(r.body);
    final root = (data['data'] ?? data) as Map<String, dynamic>;
    final list = (root['balances'] as List?) ??
        (data['balances'] as List?) ??
        const [];
    return list
        .map((b) => TreasuryBalance.fromJson(b as Map<String, dynamic>))
        .toList();
  }

  /// Get treasury transactions (real).
  Future<List<TreasuryTransaction>> getTransactions({int limit = 50}) async {
    final r = await http.get(
      Uri.parse('$_base/transactions?limit=$limit'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _err(r, 'transactions');
    final data = json.decode(r.body);
    final list = (data['data'] as List?) ?? (data['transactions'] as List?) ?? const [];
    return list
        .map((t) => TreasuryTransaction.fromJson(t as Map<String, dynamic>))
        .toList();
  }

  /// Transfer treasury funds (real backend sign + broadcast).
  Future<TreasuryTransaction> transfer({
    required String to,
    required double amount,
    required String password,
    String? token,
  }) async {
    final r = await http.post(
      Uri.parse('$_base/transfer'),
      headers: _headers,
      body: json.encode({
        'to': to,
        'amount': amount,
        'password': password,
        if (token != null) 'token': token,
      }),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _err(r, 'transfer');
    final data = json.decode(r.body);
    return TreasuryTransaction.fromJson(
      (data['data'] ?? data) as Map<String, dynamic>,
    );
  }

  /// Sweep treasury funds to a cold wallet (real backend broadcast).
  Future<bool> sweep({required String to, required String password}) async {
    final r = await http.post(
      Uri.parse('$_base/sweep'),
      headers: _headers,
      body: json.encode({'to': to, 'password': password}),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _err(r, 'sweep');
    return true;
  }

  /// Allocations are not part of the canonical treasury contract. Fetching or
  /// creating them is not supported against the live backend - fail-closed.
  Future<Allocation> createAllocation({
    required String name,
    required String token,
    required double amount,
    required String purpose,
  }) async {
    throw UnimplementedError(
      'Treasury allocations are not supported by the canonical backend. '
      'Use the treasury transfer/sweep endpoints for fund movement.',
    );
  }

  Future<List<Allocation>> getAllocations() async {
    throw UnimplementedError(
      'Treasury allocations are not supported by the canonical backend.',
    );
  }

  /// Treasury reports are not a canonical endpoint. Surface an error instead of
  /// returning fabricated inflow/outflow numbers.
  Future<TreasuryReport> getReport({
    required DateTime startDate,
    required DateTime endDate,
  }) async {
    throw UnimplementedError(
      'Treasury reports are not supported by the canonical backend. '
      'Derive reports from the treasury transactions endpoint instead.',
    );
  }
}


class TreasuryOverview {
  final double totalValue;
  final double hotWalletValue;
  final double coldWalletValue;
  final double pendingValue;
  final int todayTransactions;
  final double todayVolume;
  
  TreasuryOverview({
    required this.totalValue,
    required this.hotWalletValue,
    required this.coldWalletValue,
    required this.pendingValue,
    required this.todayTransactions,
    required this.todayVolume,
  });
  
  factory TreasuryOverview.fromJson(Map<String, dynamic> json) {
    return TreasuryOverview(
      totalValue: (json['totalValue'] ?? 0).toDouble(),
      hotWalletValue: (json['hotWalletValue'] ?? 0).toDouble(),
      coldWalletValue: (json['coldWalletValue'] ?? 0).toDouble(),
      pendingValue: (json['pendingValue'] ?? 0).toDouble(),
      todayTransactions: json['todayTransactions'] ?? 0,
      todayVolume: (json['todayVolume'] ?? 0).toDouble(),
    );
  }
}

class TreasuryBalance {
  final String token;
  final String name;
  final double balance;
  final double value;
  final String account;
  
  TreasuryBalance({
    required this.token,
    required this.name,
    required this.balance,
    required this.value,
    required this.account,
  });
  
  factory TreasuryBalance.fromJson(Map<String, dynamic> json) {
    return TreasuryBalance(
      token: json['token'] ?? '',
      name: json['name'] ?? '',
      balance: (json['balance'] ?? 0).toDouble(),
      value: (json['value'] ?? 0).toDouble(),
      account: json['account'] ?? 'hot',
    );
  }
}

class TreasuryTransaction {
  final String id;
  final String type;
  final String token;
  final double amount;
  final double value;
  final String fromAccount;
  final String toAccount;
  final String? txHash;
  final String status;
  final DateTime createdAt;
  
  TreasuryTransaction({
    required this.id,
    required this.type,
    required this.token,
    required this.amount,
    required this.value,
    required this.fromAccount,
    required this.toAccount,
    this.txHash,
    required this.status,
    required this.createdAt,
  });
  
  factory TreasuryTransaction.fromJson(Map<String, dynamic> json) {
    return TreasuryTransaction(
      id: json['id'] ?? '',
      type: json['type'] ?? '',
      token: json['token'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      value: (json['value'] ?? 0).toDouble(),
      fromAccount: json['fromAccount'] ?? '',
      toAccount: json['toAccount'] ?? '',
      txHash: json['txHash'],
      status: json['status'] ?? 'PENDING',
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class Allocation {
  final String id;
  final String name;
  final String token;
  final double allocated;
  final double spent;
  final String purpose;
  final String status;
  final DateTime createdAt;
  
  Allocation({
    required this.id,
    required this.name,
    required this.token,
    required this.allocated,
    required this.spent,
    required this.purpose,
    required this.status,
    required this.createdAt,
  });
  
  factory Allocation.fromJson(Map<String, dynamic> json) {
    return Allocation(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      token: json['token'] ?? '',
      allocated: (json['allocated'] ?? 0).toDouble(),
      spent: (json['spent'] ?? 0).toDouble(),
      purpose: json['purpose'] ?? '',
      status: json['status'] ?? 'ACTIVE',
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class TreasuryReport {
  final double totalInflow;
  final double totalOutflow;
  final double netChange;
  final int transactionCount;
  final Map<String, double> tokenBreakdown;
  
  TreasuryReport({
    required this.totalInflow,
    required this.totalOutflow,
    required this.netChange,
    required this.transactionCount,
    required this.tokenBreakdown,
  });
  
  factory TreasuryReport.fromJson(Map<String, dynamic> json) {
    return TreasuryReport(
      totalInflow: (json['totalInflow'] ?? 0).toDouble(),
      totalOutflow: (json['totalOutflow'] ?? 0).toDouble(),
      netChange: (json['netChange'] ?? 0).toDouble(),
      transactionCount: json['transactionCount'] ?? 0,
      tokenBreakdown: Map<String, double>.from(json['tokenBreakdown'] ?? {}),
    );
  }
}
