// MasterWallet Audit Service - Flutter
//
// Thin REST client over the canonical Go backend (:8450). The canonical audit
// endpoint is GET /api/v1/master-wallet/:id/audit. Summaries, compliance
// reports, suspicious-activity feeds, and exports are not part of the
// canonical contract and fail closed rather than returning fabricated data.

import 'dart:convert';
import 'package:http/http.dart' as http;

class AuditService {
  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );
  static const String _apiV1 = '$API_BASE/api/v1';

  final String masterWalletId;
  String? _token;

  AuditService({required this.masterWalletId, String? token}) : _token = token;

  void setToken(String? token) => _token = token;

  String get _base => '$_apiV1/master-wallet/$masterWalletId/audit';

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Exception _err(http.Response r, String op) =>
      Exception('audit $op failed (${r.statusCode}): ${r.body}');

  dynamic _body(http.Response r) {
    final data = json.decode(r.body);
    return data['data'] ?? data;
  }

  /// Get audit logs from the canonical backend audit endpoint. Supports
  /// limit/offset pagination; user/action/date filters are forwarded as query
  /// params for the backend to apply (or ignore) - never fabricated locally.
  Future<List<AuditLog>> getLogs({
    String? userId,
    String? action,
    DateTime? startDate,
    DateTime? endDate,
    int limit = 100,
    int offset = 0,
  }) async {
    final q = Uri.parse(_base).replace(
      queryParameters: {
        'limit': limit.toString(),
        'offset': offset.toString(),
        if (userId != null) 'user_id': userId,
        if (action != null) 'action': action,
        if (startDate != null) 'start': startDate.toIso8601String(),
        if (endDate != null) 'end': endDate.toIso8601String(),
      },
    );
    final r = await http.get(q, headers: _headers);
    if (r.statusCode != 200) throw _err(r, 'getLogs');
    final list = (_body(r) as List?) ?? const [];
    return list.map((l) => AuditLog.fromJson(l as Map<String, dynamic>)).toList();
  }

  // The canonical backend exposes only the flat audit log endpoint. Derived
  // reports / activity feeds / exports are not supported and fail closed.
  Future<AuditSummary> getSummary({DateTime? startDate, DateTime? endDate}) async {
    throw UnimplementedError(
      'audit summary is not supported by the canonical backend contract. '
      'Aggregate the audit log endpoint client-side if needed.',
    );
  }

  Future<List<AuditLog>> getUserActivity(String userId, {int limit = 50}) async {
    return getLogs(userId: userId, limit: limit);
  }

  Future<List<AuditLog>> getTransactionAudit(String txId) async {
    throw UnimplementedError(
      'audit transaction trail is not supported by the canonical backend contract.',
    );
  }

  Future<String> exportLogs({
    required String format,
    DateTime? startDate,
    DateTime? endDate,
  }) async {
    throw UnimplementedError(
      'audit export is not supported by the canonical backend contract.',
    );
  }

  Future<ComplianceReport> getComplianceReport({
    required DateTime startDate,
    required DateTime endDate,
  }) async {
    throw UnimplementedError(
      'audit compliance reports are not supported by the canonical backend contract.',
    );
  }

  Future<List<AuditLog>> getSuspiciousActivities({int limit = 50}) async {
    throw UnimplementedError(
      'audit suspicious-activity feed is not supported by the canonical backend contract.',
    );
  }
}

class AuditLog {
  final String id;
  final String userId;
  final String userName;
  final String action;
  final String entityType;
  final String entityId;
  final Map<String, dynamic> details;
  final String ipAddress;
  final String userAgent;
  final String status;
  final DateTime timestamp;
  
  AuditLog({
    required this.id,
    required this.userId,
    required this.userName,
    required this.action,
    required this.entityType,
    required this.entityId,
    required this.details,
    required this.ipAddress,
    required this.userAgent,
    required this.status,
    required this.timestamp,
  });
  
  factory AuditLog.fromJson(Map<String, dynamic> json) {
    return AuditLog(
      id: json['id'] ?? '',
      userId: json['userId'] ?? '',
      userName: json['userName'] ?? '',
      action: json['action'] ?? '',
      entityType: json['entityType'] ?? '',
      entityId: json['entityId'] ?? '',
      details: Map<String, dynamic>.from(json['details'] ?? {}),
      ipAddress: json['ipAddress'] ?? '',
      userAgent: json['userAgent'] ?? '',
      status: json['status'] ?? 'SUCCESS',
      timestamp: DateTime.parse(json['timestamp']),
    );
  }
}

class AuditSummary {
  final int totalActions;
  final int successCount;
  final int failureCount;
  final Map<String, int> actionBreakdown;
  final Map<String, int> userActivity;
  
  AuditSummary({
    required this.totalActions,
    required this.successCount,
    required this.failureCount,
    required this.actionBreakdown,
    required this.userActivity,
  });
  
  factory AuditSummary.fromJson(Map<String, dynamic> json) {
    return AuditSummary(
      totalActions: json['totalActions'] ?? 0,
      successCount: json['successCount'] ?? 0,
      failureCount: json['failureCount'] ?? 0,
      actionBreakdown: Map<String, int>.from(json['actionBreakdown'] ?? {}),
      userActivity: Map<String, int>.from(json['userActivity'] ?? {}),
    );
  }
}

class ComplianceReport {
  final DateTime startDate;
  final DateTime endDate;
  final int totalTransactions;
  final int approvedTransactions;
  final int rejectedTransactions;
  final double totalVolume;
  final List<String> complianceNotes;
  
  ComplianceReport({
    required this.startDate,
    required this.endDate,
    required this.totalTransactions,
    required this.approvedTransactions,
    required this.rejectedTransactions,
    required this.totalVolume,
    required this.complianceNotes,
  });
  
  factory ComplianceReport.fromJson(Map<String, dynamic> json) {
    return ComplianceReport(
      startDate: DateTime.parse(json['startDate']),
      endDate: DateTime.parse(json['endDate']),
      totalTransactions: json['totalTransactions'] ?? 0,
      approvedTransactions: json['approvedTransactions'] ?? 0,
      rejectedTransactions: json['rejectedTransactions'] ?? 0,
      totalVolume: (json['totalVolume'] ?? 0).toDouble(),
      complianceNotes: List<String>.from(json['complianceNotes'] ?? []),
    );
  }
}
