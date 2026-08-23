// MPC Wallet Service - Flutter Mobile
// Multi-party computation wallet

import 'dart:convert';
import 'package:http/http.dart' as http;

class MPCWalletService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  MPCWalletService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Create wallet share for new device
  Future<MPCShare> createShare({
    required String deviceId,
    required String publicKey,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/mpc/shares'),
      headers: _headers,
      body: json.encode({
        'deviceId': deviceId,
        'publicKey': publicKey,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return MPCShare.fromJson(data['data']);
    }
    throw Exception('Failed to create share');
  }
  
  // Sign transaction
  Future<String> signTransaction(String txHash) async {
    final response = await http.post(
      Uri.parse('$API_BASE/mpc/sign'),
      headers: _headers,
      body: json.encode({'txHash': txHash}),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['data']['signature'];
    }
    throw Exception('Failed to sign');
  }
  
  // Sign message
  Future<String> signMessage(String message) async {
    final response = await http.post(
      Uri.parse('$API_BASE/mpc/sign-message'),
      headers: _headers,
      body: json.encode({'message': message}),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['data']['signature'];
    }
    throw Exception('Failed to sign');
  }
  
  // Get wallet address
  Future<String> getWalletAddress() async {
    final response = await http.get(
      Uri.parse('$API_BASE/mpc/address'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['data']['address'];
    }
    throw Exception('Failed to get address');
  }
  
  // Get user shares
  Future<List<MPCShare>> getShares() async {
    final response = await http.get(
      Uri.parse('$API_BASE/mpc/shares'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((s) => MPCShare.fromJson(s)).toList();
    }
    return [];
  }
  
  // Add new device share
  Future<MPCShare> addDevice({
    required String deviceId,
    required String publicKey,
  }) async {
    return createShare(deviceId: deviceId, publicKey: publicKey);
  }
  
  // Remove device share
  Future<bool> removeShare(String shareId) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/mpc/shares/$shareId'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Verify device
  Future<bool> verifyDevice(String deviceId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/mpc/verify/$deviceId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['data'] ?? false;
    }
    return false;
  }
}

class MPCShare {
  final String id;
  final String deviceId;
  final String publicKey;
  final String status;
  final DateTime createdAt;
  final DateTime lastUsedAt;
  
  MPCShare({
    required this.id,
    required this.deviceId,
    required this.publicKey,
    required this.status,
    required this.createdAt,
    required this.lastUsedAt,
  });
  
  factory MPCShare.fromJson(Map<String, dynamic> json) {
    return MPCShare(
      id: json['id'] ?? '',
      deviceId: json['deviceId'] ?? '',
      publicKey: json['publicKey'] ?? '',
      status: json['status'] ?? 'ACTIVE',
      createdAt: DateTime.parse(json['createdAt']),
      lastUsedAt: DateTime.parse(json['lastUsedAt']),
    );
  }
}
