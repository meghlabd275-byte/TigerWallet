// MasterWallet Audit Service - Flutter

import 'dart:convert';
import 'package:http/http.dart' as http;

class AuditService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  AuditService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get audit logs
  Future<List<AuditLog>> getLogs({
    String? userId,
    String? action,
    DateTime? startDate,
    DateTime? endDate,
    int limit = 100,
    int offset = 0,
  }) async {
    String url = '$API_BASE/audit/logs?limit=$limit&offset=$offset';
    if (userId != null) url += '&userId=$userId';
    if (action != null) url += '&action=$action';
    if (startDate != null) url += '&startDate=${startDate.toIso8601String()}';
    if (endDate != null) url += '&endDate=${endDate.toIso8601String()}';
    
    final response = await http.get(Uri.parse(url), headers: _headers);
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((l) => AuditLog.fromJson(l)).toList();
    }
    return [];
  }
  
  // Get audit summary
  Future<AuditSummary> getSummary({DateTime? startDate, DateTime? endDate}) async {
    String url = '$API_BASE/audit/summary';
    if (startDate != null || endDate != null) {
      url += '?';
      if (startDate != null) url += 'startDate=${startDate.toIso8601String()}';
      if (endDate != null) url += '&endDate=${endDate.toIso8601String()}';
    }
    
    final response = await http.get(Uri.parse(url), headers: _headers);
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return AuditSummary.fromJson(data['data']);
    }
    throw Exception('Failed to get summary');
  }
  
  // Get user activity
  Future<List<AuditLog>> getUserActivity(String userId, {int limit = 50}) async {
    final response = await http.get(
      Uri.parse('$API_BASE/audit/users/$userId/activity?limit=$limit'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((l) => AuditLog.fromJson(l)).toList();
    }
    return [];
  }
  
  // Get transaction audit trail
  Future<List<AuditLog>> getTransactionAudit(String txId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/audit/transactions/$txId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((l) => AuditLog.fromJson(l)).toList();
    }
    return [];
  }
  
  // Export audit logs
  Future<String> exportLogs({
    required String format,
    DateTime? startDate,
    DateTime? endDate,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/audit/export'),
      headers: _headers,
      body: json.encode({
        'format': format,
        'startDate': startDate?.toIso8601String(),
        'endDate': endDate?.toIso8601String(),
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return data['data']['downloadUrl'];
    }
    throw Exception('Failed to export');
  }
  
  // Get compliance report
  Future<ComplianceReport> getComplianceReport({
    required DateTime startDate,
    required DateTime endDate,
  }) async {
    final response = await http.get(
      Uri.parse('$API_BASE/audit/compliance?start=${startDate.toIso8601String()}&end=${endDate.toIso8601String()}'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return ComplianceReport.fromJson(data['data']);
    }
    throw Exception('Failed to get compliance report');
  }
  
  // Get suspicious activities
  Future<List<AuditLog>> getSuspiciousActivities({int limit = 50}) async {
    final response = await http.get(
      Uri.parse('$API_BASE/audit/suspicious?limit=$limit'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((l) => AuditLog.fromJson(l)).toList();
    }
    return [];
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
