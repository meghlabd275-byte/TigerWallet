// MasterWallet Treasury Service - Flutter

import 'dart:convert';
import 'package:http/http.dart' as http;

class TreasuryService {
  static const String API_BASE = 'https://master-api.tigerwallet.com/api/v1';
  String? _token;
  
  TreasuryService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get treasury overview
  Future<TreasuryOverview> getOverview() async {
    final response = await http.get(
      Uri.parse('$API_BASE/treasury/overview'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return TreasuryOverview.fromJson(data['data']);
    }
    throw Exception('Failed to get treasury');
  }
  
  // Get treasury balances
  Future<List<TreasuryBalance>> getBalances() async {
    final response = await http.get(
      Uri.parse('$API_BASE/treasury/balances'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((b) => TreasuryBalance.fromJson(b)).toList();
    }
    return [];
  }
  
  // Get transactions
  Future<List<TreasuryTransaction>> getTransactions({int limit = 50}) async {
    final response = await http.get(
      Uri.parse('$API_BASE/treasury/transactions?limit=$limit'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((t) => TreasuryTransaction.fromJson(t)).toList();
    }
    return [];
  }
  
  // Create allocation
  Future<Allocation> createAllocation({
    required String name,
    required String token,
    required double amount,
    required String purpose,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/treasury/allocations'),
      headers: _headers,
      body: json.encode({
        'name': name,
        'token': token,
        'amount': amount,
        'purpose': purpose,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return Allocation.fromJson(data['data']);
    }
    throw Exception('Failed to create allocation');
  }
  
  // Get allocations
  Future<List<Allocation>> getAllocations() async {
    final response = await http.get(
      Uri.parse('$API_BASE/treasury/allocations'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((a) => Allocation.fromJson(a)).toList();
    }
    return [];
  }
  
  // Transfer between accounts
  Future<TreasuryTransaction> transfer({
    required String fromAccount,
    required String toAccount,
    required String token,
    required double amount,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/treasury/transfer'),
      headers: _headers,
      body: json.encode({
        'fromAccount': fromAccount,
        'toAccount': toAccount,
        'token': token,
        'amount': amount,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return TreasuryTransaction.fromJson(data['data']);
    }
    throw Exception('Failed to transfer');
  }
  
  // Sweep to cold wallet
  Future<bool> sweepToCold(String token, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/treasury/sweep'),
      headers: _headers,
      body: json.encode({
        'token': token,
        'amount': amount,
      }),
    );
    
    return response.statusCode == 200;
  }
  
  // Get reports
  Future<TreasuryReport> getReport({
    required DateTime startDate,
    required DateTime endDate,
  }) async {
    final response = await http.get(
      Uri.parse('$API_BASE/treasury/report?start=${startDate.toIso8601String()}&end=${endDate.toIso8601String()}'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return TreasuryReport.fromJson(data['data']);
    }
    throw Exception('Failed to get report');
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
