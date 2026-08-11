// MasterWallet Policy Engine Service - Flutter

import 'dart:convert';
import 'package:http/http.dart' as http;

class PolicyEngineService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  PolicyEngineService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Create policy
  Future<Policy> createPolicy({
    required String name,
    required String type,
    required Map<String, dynamic> conditions,
    required String action,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/policies'),
      headers: _headers,
      body: json.encode({
        'name': name,
        'type': type,
        'conditions': conditions,
        'action': action,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return Policy.fromJson(data['data']);
    }
    throw Exception('Failed to create policy');
  }
  
  // Get policies
  Future<List<Policy>> getPolicies() async {
    final response = await http.get(
      Uri.parse('$API_BASE/policies'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((p) => Policy.fromJson(p)).toList();
    }
    return [];
  }
  
  // Update policy
  Future<Policy> updatePolicy(String policyId, Map<String, dynamic> updates) async {
    final response = await http.put(
      Uri.parse('$API_BASE/policies/$policyId'),
      headers: _headers,
      body: json.encode(updates),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return Policy.fromJson(data['data']);
    }
    throw Exception('Failed to update policy');
  }
  
  // Delete policy
  Future<bool> deletePolicy(String policyId) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/policies/$policyId'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Test policy
  Future<PolicyTestResult> testPolicy(String policyId, Map<String, dynamic> transaction) async {
    final response = await http.post(
      Uri.parse('$API_BASE/policies/$policyId/test'),
      headers: _headers,
      body: json.encode(transaction),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return PolicyTestResult.fromJson(data['data']);
    }
    throw Exception('Failed to test policy');
  }
  
  // Get policy logs
  Future<List<PolicyLog>> getPolicyLogs({int limit = 100}) async {
    final response = await http.get(
      Uri.parse('$API_BASE/policies/logs?limit=$limit'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((l) => PolicyLog.fromJson(l)).toList();
    }
    return [];
  }
  
  // Create spend limit
  Future<SpendLimit> createSpendLimit({
    required String token,
    required double dailyLimit,
    required double monthlyLimit,
    required double perTxLimit,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/policies/spend-limits'),
      headers: _headers,
      body: json.encode({
        'token': token,
        'dailyLimit': dailyLimit,
        'monthlyLimit': monthlyLimit,
        'perTxLimit': perTxLimit,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return SpendLimit.fromJson(data['data']);
    }
    throw Exception('Failed to create spend limit');
  }
  
  // Get spend limits
  Future<List<SpendLimit>> getSpendLimits() async {
    final response = await http.get(
      Uri.parse('$API_BASE/policies/spend-limits'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((s) => SpendLimit.fromJson(s)).toList();
    }
    return [];
  }
}

class Policy {
  final String id;
  final String name;
  final String type;
  final Map<String, dynamic> conditions;
  final String action;
  final String status;
  final DateTime createdAt;
  
  Policy({
    required this.id,
    required this.name,
    required this.type,
    required this.conditions,
    required this.action,
    required this.status,
    required this.createdAt,
  });
  
  factory Policy.fromJson(Map<String, dynamic> json) {
    return Policy(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      type: json['type'] ?? '',
      conditions: Map<String, dynamic>.from(json['conditions'] ?? {}),
      action: json['action'] ?? 'ALLOW',
      status: json['status'] ?? 'ACTIVE',
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class PolicyTestResult {
  final bool allowed;
  final String? reason;
  final String? policyId;
  
  PolicyTestResult({
    required this.allowed,
    this.reason,
    this.policyId,
  });
  
  factory PolicyTestResult.fromJson(Map<String, dynamic> json) {
    return PolicyTestResult(
      allowed: json['allowed'] ?? false,
      reason: json['reason'],
      policyId: json['policyId'],
    );
  }
}

class PolicyLog {
  final String id;
  final String policyId;
  final String policyName;
  final String action;
  final Map<String, dynamic> transaction;
  final bool result;
  final DateTime timestamp;
  
  PolicyLog({
    required this.id,
    required this.policyId,
    required this.policyName,
    required this.action,
    required this.transaction,
    required this.result,
    required this.timestamp,
  });
  
  factory PolicyLog.fromJson(Map<String, dynamic> json) {
    return PolicyLog(
      id: json['id'] ?? '',
      policyId: json['policyId'] ?? '',
      policyName: json['policyName'] ?? '',
      action: json['action'] ?? '',
      transaction: Map<String, dynamic>.from(json['transaction'] ?? {}),
      result: json['result'] ?? false,
      timestamp: DateTime.parse(json['timestamp']),
    );
  }
}

class SpendLimit {
  final String id;
  final String token;
  final double dailyLimit;
  final double dailyUsed;
  final double monthlyLimit;
  final double monthlyUsed;
  final double perTxLimit;
  final String status;
  
  SpendLimit({
    required this.id,
    required this.token,
    required this.dailyLimit,
    required this.dailyUsed,
    required this.monthlyLimit,
    required this.monthlyUsed,
    required this.perTxLimit,
    required this.status,
  });
  
  factory SpendLimit.fromJson(Map<String, dynamic> json) {
    return SpendLimit(
      id: json['id'] ?? '',
      token: json['token'] ?? '',
      dailyLimit: (json['dailyLimit'] ?? 0).toDouble(),
      dailyUsed: (json['dailyUsed'] ?? 0).toDouble(),
      monthlyLimit: (json['monthlyLimit'] ?? 0).toDouble(),
      monthlyUsed: (json['monthlyUsed'] ?? 0).toDouble(),
      perTxLimit: (json['perTxLimit'] ?? 0).toDouble(),
      status: json['status'] ?? 'ACTIVE',
    );
  }
}
