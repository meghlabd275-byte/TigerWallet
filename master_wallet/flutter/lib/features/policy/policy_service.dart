// MasterWallet Policy Engine Service - Flutter
//
// Thin REST client over the canonical Go backend (:8450). Policy routes are
// nested under /api/v1/master-wallet/:id/policies. No fabricated data; on
// backend failure methods throw rather than returning empty/fake results.

import 'dart:convert';
import 'package:http/http.dart' as http;

class PolicyEngineService {
  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );
  static const String _apiV1 = '$API_BASE/api/v1';

  final String masterWalletId;
  String? _token;

  PolicyEngineService({required this.masterWalletId, String? token}) : _token = token;

  void setToken(String? token) => _token = token;

  String get _base => '$_apiV1/master-wallet/$masterWalletId/policies';

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Exception _err(http.Response r, String op) =>
      Exception('policy $op failed (${r.statusCode}): ${r.body}');

  dynamic _body(http.Response r) {
    final data = json.decode(r.body);
    return data['data'] ?? data;
  }

  /// Create a policy (real backend). The canonical contract accepts
  /// {rule_type, threshold, ...} rather than {name,type,conditions,action};
  /// we forward a permissive payload so the backend can validate the rule.
  Future<Policy> createPolicy({
    required String name,
    required String type,
    required Map<String, dynamic> conditions,
    required String action,
  }) async {
    final payload = {
      'name': name,
      'rule_type': type,
      ...conditions,
      'action': action,
    };
    final r = await http.post(
      Uri.parse(_base),
      headers: _headers,
      body: json.encode(payload),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _err(r, 'createPolicy');
    return Policy.fromJson(_body(r) as Map<String, dynamic>);
  }

  /// List policies (real backend).
  Future<List<Policy>> getPolicies() async {
    final r = await http.get(Uri.parse(_base), headers: _headers);
    if (r.statusCode != 200) throw _err(r, 'getPolicies');
    final list = (_body(r) as List?) ?? const [];
    return list.map((p) => Policy.fromJson(p as Map<String, dynamic>)).toList();
  }

  /// Update a policy (real backend).
  Future<Policy> updatePolicy(String policyId, Map<String, dynamic> updates) async {
    final r = await http.put(
      Uri.parse('$_base/$policyId'),
      headers: _headers,
      body: json.encode(updates),
    );
    if (r.statusCode != 200) throw _err(r, 'updatePolicy');
    return Policy.fromJson(_body(r) as Map<String, dynamic>);
  }

  /// Delete a policy (real backend).
  Future<bool> deletePolicy(String policyId) async {
    final r = await http.delete(
      Uri.parse('$_base/$policyId'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 204) throw _err(r, 'deletePolicy');
    return true;
  }

  // Policy testing, policy logs, and spend limits are not part of the canonical
  // policy contract. They fail closed rather than hitting non-existent routes.
  Future<PolicyTestResult> testPolicy(
    String policyId,
    Map<String, dynamic> transaction,
  ) async {
    throw UnimplementedError(
      'policy testPolicy is not supported by the canonical backend contract.',
    );
  }

  Future<List<PolicyLog>> getPolicyLogs({int limit = 100}) async {
    throw UnimplementedError(
      'policy logs are not supported by the canonical backend contract. '
      'Use the audit endpoint for activity history.',
    );
  }

  Future<SpendLimit> createSpendLimit({
    required String token,
    required double dailyLimit,
    required double monthlyLimit,
    required double perTxLimit,
  }) async {
    throw UnimplementedError(
      'policy spend limits are not supported by the canonical backend contract. '
      'Model spend thresholds as policies instead.',
    );
  }

  Future<List<SpendLimit>> getSpendLimits() async {
    throw UnimplementedError(
      'policy spend limits are not supported by the canonical backend contract.',
    );
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
