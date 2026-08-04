// Social Recovery Service - Flutter Mobile

import 'dart:convert';
import 'package:http/http.dart' as http;

class SocialRecoveryService {
  static const String API_BASE = 'https://api.tigerwallet.com/api/v1';
  String? _token;
  
  SocialRecoveryService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Setup social recovery
  Future<bool> setupRecovery(List<Guardian> guardians) async {
    final response = await http.post(
      Uri.parse('$API_BASE/recovery/setup'),
      headers: _headers,
      body: json.encode({
        'guardians': guardians.map((g) => g.toJson()).toList(),
      }),
    );
    
    return response.statusCode == 201;
  }
  
  // Get guardians
  Future<List<Guardian>> getGuardians() async {
    final response = await http.get(
      Uri.parse('$API_BASE/recovery/guardians'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((g) => Guardian.fromJson(g)).toList();
    }
    return [];
  }
  
  // Add guardian
  Future<bool> addGuardian(Guardian guardian) async {
    final response = await http.post(
      Uri.parse('$API_BASE/recovery/guardians'),
      headers: _headers,
      body: json.encode(guardian.toJson()),
    );
    
    return response.statusCode == 201;
  }
  
  // Remove guardian
  Future<bool> removeGuardian(String guardianId) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/recovery/guardians/$guardianId'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Initiate recovery
  Future<String> initiateRecovery(String guardianId, String signature) async {
    final response = await http.post(
      Uri.parse('$API_BASE/recovery/initiate'),
      headers: _headers,
      body: json.encode({
        'guardianId': guardianId,
        'signature': signature,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return data['data']['requestId'];
    }
    throw Exception('Failed to initiate recovery');
  }
  
  // Confirm recovery
  Future<bool> confirmRecovery(String requestId, String guardianId, String signature) async {
    final response = await http.post(
      Uri.parse('$API_BASE/recovery/confirm'),
      headers: _headers,
      body: json.encode({
        'requestId': requestId,
        'guardianId': guardianId,
        'signature': signature,
      }),
    );
    
    return response.statusCode == 200;
  }
  
  // Execute recovery with key
  Future<bool> executeRecovery(String recoveryKey) async {
    final response = await http.post(
      Uri.parse('$API_BASE/recovery/execute'),
      headers: _headers,
      body: json.encode({'recoveryKey': recoveryKey}),
    );
    
    return response.statusCode == 200;
  }
}

class Guardian {
  final String? id;
  final String address;
  final String name;
  final String relationship;
  final String status;
  final DateTime? addedAt;
  
  Guardian({
    this.id,
    required this.address,
    required this.name,
    required this.relationship,
    this.status = 'PENDING',
    this.addedAt,
  });
  
  factory Guardian.fromJson(Map<String, dynamic> json) {
    return Guardian(
      id: json['id'],
      address: json['address'] ?? '',
      name: json['name'] ?? '',
      relationship: json['relationship'] ?? '',
      status: json['status'] ?? 'PENDING',
      addedAt: json['addedAt'] != null ? DateTime.parse(json['addedAt']) : null,
    );
  }
  
  Map<String, dynamic> toJson() => {
    'address': address,
    'name': name,
    'relationship': relationship,
  };
}
