// Admin Platform User Management Service - Flutter

import 'dart:convert';
import 'package:http/http.dart' as http;

class UserManagementService {
  static const String API_BASE = 'https://admin-api.tigerwallet.com/api/v1';
  String? _token;
  
  UserManagementService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get all users
  Future<List<User>> getUsers({
    int page = 1,
    int limit = 50,
    String? status,
    String? search,
  }) async {
    String url = '$API_BASE/users?page=$page&limit=$limit';
    if (status != null) url += '&status=$status';
    if (search != null) url += '&search=$search';
    
    final response = await http.get(Uri.parse(url), headers: _headers);
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((u) => User.fromJson(u)).toList();
    }
    return [];
  }
  
  // Get user details
  Future<User> getUserDetails(String userId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/users/$userId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return User.fromJson(data['data']);
    }
    throw Exception('Failed to get user');
  }
  
  // Update user status
  Future<bool> updateUserStatus(String userId, String status) async {
    final response = await http.put(
      Uri.parse('$API_BASE/users/$userId/status'),
      headers: _headers,
      body: json.encode({'status': status}),
    );
    
    return response.statusCode == 200;
  }
  
  // Get user transactions
  Future<List<Transaction>> getUserTransactions(String userId, {int limit = 50}) async {
    final response = await http.get(
      Uri.parse('$API_BASE/users/$userId/transactions?limit=$limit'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((t) => Transaction.fromJson(t)).toList();
    }
    return [];
  }
  
  // Freeze user account
  Future<bool> freezeUser(String userId, String reason) async {
    final response = await http.post(
      Uri.parse('$API_BASE/users/$userId/freeze'),
      headers: _headers,
      body: json.encode({'reason': reason}),
    );
    
    return response.statusCode == 200;
  }
  
  // Unfreeze user account
  Future<bool> unfreezeUser(String userId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/users/$userId/unfreeze'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get user sessions
  Future<List<UserSession>> getUserSessions(String userId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/users/$userId/sessions'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((s) => UserSession.fromJson(s)).toList();
    }
    return [];
  }
  
  // Revoke user session
  Future<bool> revokeSession(String userId, String sessionId) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/users/$userId/sessions/$sessionId'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get user balances
  Future<Map<String, double>> getUserBalances(String userId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/users/$userId/balances'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return Map<String, double>.from(data['data']);
    }
    return {};
  }
}

class User {
  final String id;
  final String email;
  final String username;
  final String status;
  final String kycStatus;
  final String kycLevel;
  final DateTime createdAt;
  final DateTime? lastLoginAt;
  final int totalTransactions;
  final double totalVolume;
  
  User({
    required this.id,
    required this.email,
    required this.username,
    required this.status,
    required this.kycStatus,
    required this.kycLevel,
    required this.createdAt,
    this.lastLoginAt,
    required this.totalTransactions,
    required this.totalVolume,
  });
  
  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'] ?? '',
      email: json['email'] ?? '',
      username: json['username'] ?? '',
      status: json['status'] ?? 'ACTIVE',
      kycStatus: json['kycStatus'] ?? 'NONE',
      kycLevel: json['kycLevel'] ?? '0',
      createdAt: DateTime.parse(json['createdAt']),
      lastLoginAt: json['lastLoginAt'] != null ? DateTime.parse(json['lastLoginAt']) : null,
      totalTransactions: json['totalTransactions'] ?? 0,
      totalVolume: (json['totalVolume'] ?? 0).toDouble(),
    );
  }
}

class Transaction {
  final String id;
  final String type;
  final String token;
  final double amount;
  final double fee;
  final String status;
  final String? txHash;
  final DateTime createdAt;
  
  Transaction({
    required this.id,
    required this.type,
    required this.token,
    required this.amount,
    required this.fee,
    required this.status,
    this.txHash,
    required this.createdAt,
  });
  
  factory Transaction.fromJson(Map<String, dynamic> json) {
    return Transaction(
      id: json['id'] ?? '',
      type: json['type'] ?? '',
      token: json['token'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      fee: (json['fee'] ?? 0).toDouble(),
      status: json['status'] ?? '',
      txHash: json['txHash'],
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class UserSession {
  final String id;
  final String ipAddress;
  final String device;
  final String location;
  final DateTime createdAt;
  final DateTime? lastActivityAt;
  final bool current;
  
  UserSession({
    required this.id,
    required this.ipAddress,
    required this.device,
    required this.location,
    required this.createdAt,
    this.lastActivityAt,
    required this.current,
  });
  
  factory UserSession.fromJson(Map<String, dynamic> json) {
    return UserSession(
      id: json['id'] ?? '',
      ipAddress: json['ipAddress'] ?? '',
      device: json['device'] ?? '',
      location: json['location'] ?? '',
      createdAt: DateTime.parse(json['createdAt']),
      lastActivityAt: json['lastActivityAt'] != null ? DateTime.parse(json['lastActivityAt']) : null,
      current: json['current'] ?? false,
    );
  }
}
