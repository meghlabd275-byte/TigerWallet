// Admin Platform Monitoring Service - Flutter

import 'dart:convert';
import 'package:http/http.dart' as http;

class MonitoringService {
  static const String API_BASE = 'https://admin-api.tigerwallet.com/api/v1';
  String? _token;
  
  MonitoringService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get system health
  Future<SystemHealth> getSystemHealth() async {
    final response = await http.get(
      Uri.parse('$API_BASE/monitoring/health'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return SystemHealth.fromJson(data['data']);
    }
    throw Exception('Failed to get health');
  }
  
  // Get metrics
  Future<Metrics> getMetrics({int duration = 60}) async {
    final response = await http.get(
      Uri.parse('$API_BASE/monitoring/metrics?duration=$duration'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return Metrics.fromJson(data['data']);
    }
    throw Exception('Failed to get metrics');
  }
  
  // Get alerts
  Future<List<Alert>> getAlerts({int limit = 50}) async {
    final response = await http.get(
      Uri.parse('$API_BASE/monitoring/alerts?limit=$limit'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((a) => Alert.fromJson(a)).toList();
    }
    return [];
  }
  
  // Acknowledge alert
  Future<bool> acknowledgeAlert(String alertId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/monitoring/alerts/$alertId/acknowledge'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get transaction monitor
  Future<TransactionMonitor> getTransactionMonitor({
    String? status,
    int limit = 100,
  }) async {
    String url = '$API_BASE/monitoring/transactions?limit=$limit';
    if (status != null) url += '&status=$status';
    
    final response = await http.get(Uri.parse(url), headers: _headers);
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return TransactionMonitor.fromJson(data['data']);
    }
    throw Exception('Failed to get transactions');
  }
  
  // Get withdrawal queue
  Future<List<WithdrawalRequest>> getWithdrawalQueue() async {
    final response = await http.get(
      Uri.parse('$API_BASE/monitoring/withdrawals'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((w) => WithdrawalRequest.fromJson(w)).toList();
    }
    return [];
  }
  
  // Approve withdrawal
  Future<bool> approveWithdrawal(String withdrawalId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/monitoring/withdrawals/$withdrawalId/approve'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Reject withdrawal
  Future<bool> rejectWithdrawal(String withdrawalId, String reason) async {
    final response = await http.post(
      Uri.parse('$API_BASE/monitoring/withdrawals/$withdrawalId/reject'),
      headers: _headers,
      body: json.encode({'reason': reason}),
    );
    
    return response.statusCode == 200;
  }
  
  // Get node status
  Future<List<NodeStatus>> getNodeStatus() async {
    final response = await http.get(
      Uri.parse('$API_BASE/monitoring/nodes'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((n) => NodeStatus.fromJson(n)).toList();
    }
    return [];
  }
}

class SystemHealth {
  final String status;
  final double uptime;
  final double cpuUsage;
  final double memoryUsage;
  final double diskUsage;
  final Map<String, String> services;
  
  SystemHealth({
    required this.status,
    required this.uptime,
    required this.cpuUsage,
    required this.memoryUsage,
    required this.diskUsage,
    required this.services,
  });
  
  factory SystemHealth.fromJson(Map<String, dynamic> json) {
    return SystemHealth(
      status: json['status'] ?? 'healthy',
      uptime: (json['uptime'] ?? 0).toDouble(),
      cpuUsage: (json['cpuUsage'] ?? 0).toDouble(),
      memoryUsage: (json['memoryUsage'] ?? 0).toDouble(),
      diskUsage: (json['diskUsage'] ?? 0).toDouble(),
      services: Map<String, String>.from(json['services'] ?? {}),
    );
  }
}

class Metrics {
  final double rps;
  final double latency;
  final double errorRate;
  final int activeConnections;
  final Map<String, double> byEndpoint;
  
  Metrics({
    required this.rps,
    required this.latency,
    required this.errorRate,
    required this.activeConnections,
    required this.byEndpoint,
  });
  
  factory Metrics.fromJson(Map<String, dynamic> json) {
    return Metrics(
      rps: (json['rps'] ?? 0).toDouble(),
      latency: (json['latency'] ?? 0).toDouble(),
      errorRate: (json['errorRate'] ?? 0).toDouble(),
      activeConnections: json['activeConnections'] ?? 0,
      byEndpoint: Map<String, double>.from(json['byEndpoint'] ?? {}),
    );
  }
}

class Alert {
  final String id;
  final String severity;
  final String message;
  final String source;
  final bool acknowledged;
  final DateTime createdAt;
  
  Alert({
    required this.id,
    required this.severity,
    required this.message,
    required this.source,
    required this.acknowledged,
    required this.createdAt,
  });
  
  factory Alert.fromJson(Map<String, dynamic> json) {
    return Alert(
      id: json['id'] ?? '',
      severity: json['severity'] ?? 'info',
      message: json['message'] ?? '',
      source: json['source'] ?? '',
      acknowledged: json['acknowledged'] ?? false,
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class TransactionMonitor {
  final int total;
  final int pending;
  final int confirmed;
  final int failed;
  final List<TransactionEntry> recent;
  
  TransactionMonitor({
    required this.total,
    required this.pending,
    required this.confirmed,
    required this.failed,
    required this.recent,
  });
  
  factory TransactionMonitor.fromJson(Map<String, dynamic> json) {
    return TransactionMonitor(
      total: json['total'] ?? 0,
      pending: json['pending'] ?? 0,
      confirmed: json['confirmed'] ?? 0,
      failed: json['failed'] ?? 0,
      recent: (json['recent'] as List? ?? []).map((t) => TransactionEntry.fromJson(t)).toList(),
    );
  }
}

class TransactionEntry {
  final String id;
  final String type;
  final String status;
  final double amount;
  final DateTime createdAt;
  
  TransactionEntry({
    required this.id,
    required this.type,
    required this.status,
    required this.amount,
    required this.createdAt,
  });
  
  factory TransactionEntry.fromJson(Map<String, dynamic> json) {
    return TransactionEntry(
      id: json['id'] ?? '',
      type: json['type'] ?? '',
      status: json['status'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class WithdrawalRequest {
  final String id;
  final String userId;
  final String token;
  final double amount;
  final String address;
  final String status;
  final DateTime createdAt;
  
  WithdrawalRequest({
    required this.id,
    required this.userId,
    required this.token,
    required this.amount,
    required this.address,
    required this.status,
    required this.createdAt,
  });
  
  factory WithdrawalRequest.fromJson(Map<String, dynamic> json) {
    return WithdrawalRequest(
      id: json['id'] ?? '',
      userId: json['userId'] ?? '',
      token: json['token'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      address: json['address'] ?? '',
      status: json['status'] ?? 'PENDING',
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class NodeStatus {
  final String name;
  final String type;
  final String status;
  final double syncStatus;
  final int peers;
  final double latency;
  
  NodeStatus({
    required this.name,
    required this.type,
    required this.status,
    required this.syncStatus,
    required this.peers,
    required this.latency,
  });
  
  factory NodeStatus.fromJson(Map<String, dynamic> json) {
    return NodeStatus(
      name: json['name'] ?? '',
      type: json['type'] ?? '',
      status: json['status'] ?? '',
      syncStatus: (json['syncStatus'] ?? 0).toDouble(),
      peers: json['peers'] ?? 0,
      latency: (json['latency'] ?? 0).toDouble(),
    );
  }
}
