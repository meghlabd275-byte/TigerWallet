// Hardware Wallet Service - Flutter Mobile
// Support for Ledger, Trezor, etc.

import 'dart:convert';
import 'package:http/http.dart' as http;

class HardwareWalletService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  HardwareWalletService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Supported device types
  static const List<String> SUPPORTED_DEVICES = [
    'LEDGER_NANO_X',
    'LEDGER_NANO_S',
    'TREZOR_MODEL_T',
    'TREZOR_ONE',
    'KEYSTONE',
    'COLDCAED',
  ];
  
  // Register device
  Future<HardwareWallet> registerDevice({
    required String deviceType,
    required String serialNumber,
    required String firmwareVersion,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/hardware/register'),
      headers: _headers,
      body: json.encode({
        'deviceType': deviceType,
        'serialNumber': serialNumber,
        'firmwareVersion': firmwareVersion,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return HardwareWallet.fromJson(data['data']);
    }
    throw Exception('Failed to register device');
  }
  
  // Sign transaction
  Future<String> signTransaction(String walletId, String txHash) async {
    final response = await http.post(
      Uri.parse('$API_BASE/hardware/sign'),
      headers: _headers,
      body: json.encode({
        'walletId': walletId,
        'txHash': txHash,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['data']['signature'];
    }
    throw Exception('Failed to sign transaction');
  }
  
  // Sign message
  Future<String> signMessage(String walletId, String message) async {
    final response = await http.post(
      Uri.parse('$API_BASE/hardware/sign-message'),
      headers: _headers,
      body: json.encode({
        'walletId': walletId,
        'message': message,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['data']['signature'];
    }
    throw Exception('Failed to sign message');
  }
  
  // Get user's hardware wallets
  Future<List<HardwareWallet>> getWallets() async {
    final response = await http.get(
      Uri.parse('$API_BASE/hardware/wallets'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((w) => HardwareWallet.fromJson(w)).toList();
    }
    return [];
  }
  
  // Remove wallet
  Future<bool> removeWallet(String walletId) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/hardware/wallets/$walletId'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Verify connection
  Future<bool> verifyConnection(String walletId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/hardware/verify/$walletId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['data'] ?? false;
    }
    return false;
  }
}

class HardwareWallet {
  final String id;
  final String deviceType;
  final String serialNumber;
  final String firmwareVersion;
  final String status;
  final DateTime registeredAt;
  final DateTime lastUsedAt;
  
  HardwareWallet({
    required this.id,
    required this.deviceType,
    required this.serialNumber,
    required this.firmwareVersion,
    required this.status,
    required this.registeredAt,
    required this.lastUsedAt,
  });
  
  factory HardwareWallet.fromJson(Map<String, dynamic> json) {
    return HardwareWallet(
      id: json['id'] ?? '',
      deviceType: json['deviceType'] ?? '',
      serialNumber: json['serialNumber'] ?? '',
      firmwareVersion: json['firmwareVersion'] ?? '',
      status: json['status'] ?? 'ACTIVE',
      registeredAt: DateTime.parse(json['registeredAt']),
      lastUsedAt: DateTime.parse(json['lastUsedAt']),
    );
  }
}
