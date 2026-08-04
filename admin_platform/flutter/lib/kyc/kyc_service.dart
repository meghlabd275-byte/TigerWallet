// Admin Platform KYC Service - Flutter

import 'dart:convert';
import 'package:http/http.dart' as http;

class KYCService {
  static const String API_BASE = 'https://admin-api.tigerwallet.com/api/v1';
  String? _token;
  
  KYCService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get pending KYC applications
  Future<List<KYCApplication>> getPendingApplications({int limit = 50}) async {
    final response = await http.get(
      Uri.parse('$API_BASE/kyc/applications?status=PENDING&limit=$limit'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((a) => KYCApplication.fromJson(a)).toList();
    }
    return [];
  }
  
  // Get all KYC applications
  Future<List<KYCApplication>> getApplications({
    String? status,
    int page = 1,
    int limit = 50,
  }) async {
    String url = '$API_BASE/kyc/applications?page=$page&limit=$limit';
    if (status != null) url += '&status=$status';
    
    final response = await http.get(Uri.parse(url), headers: _headers);
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((a) => KYCApplication.fromJson(a)).toList();
    }
    return [];
  }
  
  // Get application details
  Future<KYCApplication> getApplicationDetails(String applicationId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/kyc/applications/$applicationId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return KYCApplication.fromJson(data['data']);
    }
    throw Exception('Failed to get application');
  }
  
  // Approve KYC
  Future<bool> approveApplication(String applicationId, String notes) async {
    final response = await http.post(
      Uri.parse('$API_BASE/kyc/applications/$applicationId/approve'),
      headers: _headers,
      body: json.encode({'notes': notes}),
    );
    
    return response.statusCode == 200;
  }
  
  // Reject KYC
  Future<bool> rejectApplication(String applicationId, String reason) async {
    final response = await http.post(
      Uri.parse('$API_BASE/kyc/applications/$applicationId/reject'),
      headers: _headers,
      body: json.encode({'reason': reason}),
    );
    
    return response.statusCode == 200;
  }
  
  // Request additional info
  Future<bool> requestInfo(String applicationId, List<String> documents) async {
    final response = await http.post(
      Uri.parse('$API_BASE/kyc/applications/$applicationId/request-info'),
      headers: _headers,
      body: json.encode({'documents': documents}),
    );
    
    return response.statusCode == 200;
  }
  
  // Get KYC statistics
  Future<KYCStats> getStats() async {
    final response = await http.get(
      Uri.parse('$API_BASE/kyc/stats'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return KYCStats.fromJson(data['data']);
    }
    throw Exception('Failed to get stats');
  }
  
  // Get verification levels
  Future<List<VerificationLevel>> getVerificationLevels() async {
    final response = await http.get(
      Uri.parse('$API_BASE/kyc/levels'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((l) => VerificationLevel.fromJson(l)).toList();
    }
    return [];
  }
  
  // Update verification level
  Future<bool> updateVerificationLevel(String levelId, Map<String, dynamic> updates) async {
    final response = await http.put(
      Uri.parse('$API_BASE/kyc/levels/$levelId'),
      headers: _headers,
      body: json.encode(updates),
    );
    
    return response.statusCode == 200;
  }
}

class KYCApplication {
  final String id;
  final String userId;
  final String userEmail;
  final String level;
  final String status;
  final Map<String, dynamic> documents;
  final String? rejectionReason;
  final DateTime submittedAt;
  final DateTime? reviewedAt;
  final String? reviewedBy;
  
  KYCApplication({
    required this.id,
    required this.userId,
    required this.userEmail,
    required this.level,
    required this.status,
    required this.documents,
    this.rejectionReason,
    required this.submittedAt,
    this.reviewedAt,
    this.reviewedBy,
  });
  
  factory KYCApplication.fromJson(Map<String, dynamic> json) {
    return KYCApplication(
      id: json['id'] ?? '',
      userId: json['userId'] ?? '',
      userEmail: json['userEmail'] ?? '',
      level: json['level'] ?? '1',
      status: json['status'] ?? 'PENDING',
      documents: Map<String, dynamic>.from(json['documents'] ?? {}),
      rejectionReason: json['rejectionReason'],
      submittedAt: DateTime.parse(json['submittedAt']),
      reviewedAt: json['reviewedAt'] != null ? DateTime.parse(json['reviewedAt']) : null,
      reviewedBy: json['reviewedBy'],
    );
  }
}

class KYCStats {
  final int pending;
  final int approved;
  final int rejected;
  final int total;
  final Map<String, int> byLevel;
  
  KYCStats({
    required this.pending,
    required this.approved,
    required this.rejected,
    required this.total,
    required this.byLevel,
  });
  
  factory KYCStats.fromJson(Map<String, dynamic> json) {
    return KYCStats(
      pending: json['pending'] ?? 0,
      approved: json['approved'] ?? 0,
      rejected: json['rejected'] ?? 0,
      total: json['total'] ?? 0,
      byLevel: Map<String, int>.from(json['byLevel'] ?? {}),
    );
  }
}

class VerificationLevel {
  final String id;
  final String name;
  final int level;
  final List<String> requiredDocuments;
  final double? maxWithdrawal;
  final double? maxDeposit;
  final bool isActive;
  
  VerificationLevel({
    required this.id,
    required this.name,
    required this.level,
    required this.requiredDocuments,
    this.maxWithdrawal,
    this.maxDeposit,
    required this.isActive,
  });
  
  factory VerificationLevel.fromJson(Map<String, dynamic> json) {
    return VerificationLevel(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      level: json['level'] ?? 1,
      requiredDocuments: List<String>.from(json['requiredDocuments'] ?? []),
      maxWithdrawal: json['maxWithdrawal']?.toDouble(),
      maxDeposit: json['maxDeposit']?.toDouble(),
      isActive: json['isActive'] ?? true,
    );
  }
}
