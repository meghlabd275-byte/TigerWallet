// MasterWallet Multi-Sig Service - Flutter
// Complete multi-signature wallet functionality with real backend

import 'dart:convert';
import 'package:http/http.dart' as http;

class MultiSigService {
  static const String API_BASE = 'https://master-api.tigerwallet.com/api/v1';
  String? _token;
  
  MultiSigService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Create multi-sig wallet
  Future<MultiSigWallet> createWallet({
    required String name,
    required List<String> owners,
    required int threshold,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/multisig/wallets'),
      headers: _headers,
      body: json.encode({
        'name': name,
        'owners': owners,
        'threshold': threshold,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return MultiSigWallet.fromJson(data['data']);
    }
    throw Exception('Failed to create wallet');
  }
  
  // Get user's wallets
  Future<List<MultiSigWallet>> getWallets() async {
    final response = await http.get(
      Uri.parse('$API_BASE/multisig/wallets'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((w) => MultiSigWallet.fromJson(w)).toList();
    }
    return [];
  }
  
  // Get wallet details
  Future<MultiSigWallet> getWalletDetails(String walletId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/multisig/wallets/$walletId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return MultiSigWallet.fromJson(data['data']);
    }
    throw Exception('Failed to get wallet');
  }
  
  // Create transaction
  Future<MultiSigTx> createTransaction({
    required String walletId,
    required String to,
    required String token,
    required double amount,
    String? data,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/multisig/wallets/$walletId/transactions'),
      headers: _headers,
      body: json.encode({
        'to': to,
        'token': token,
        'amount': amount,
        'data': data,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return MultiSigTx.fromJson(data['data']);
    }
    throw Exception('Failed to create transaction');
  }
  
  // Approve transaction
  Future<bool> approveTransaction(String walletId, String txId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/multisig/wallets/$walletId/transactions/$txId/approve'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Execute transaction (after threshold met)
  Future<bool> executeTransaction(String walletId, String txId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/multisig/wallets/$walletId/transactions/$txId/execute'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Cancel transaction
  Future<bool> cancelTransaction(String walletId, String txId) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/multisig/wallets/$walletId/transactions/$txId'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get pending transactions
  Future<List<MultiSigTx>> getPendingTransactions(String walletId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/multisig/wallets/$walletId/transactions?status=PENDING'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((t) => MultiSigTx.fromJson(t)).toList();
    }
    return [];
  }
  
  // Get transaction history
  Future<List<MultiSigTx>> getTransactionHistory(String walletId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/multisig/wallets/$walletId/transactions'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((t) => MultiSigTx.fromJson(t)).toList();
    }
    return [];
  }
  
  // Add owner
  Future<bool> addOwner(String walletId, String newOwner) async {
    final response = await http.post(
      Uri.parse('$API_BASE/multisig/wallets/$walletId/owners'),
      headers: _headers,
      body: json.encode({'owner': newOwner}),
    );
    
    return response.statusCode == 201;
  }
  
  // Remove owner
  Future<bool> removeOwner(String walletId, String owner) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/multisig/wallets/$walletId/owners'),
      headers: _headers,
      body: json.encode({'owner': owner}),
    );
    
    return response.statusCode == 200;
  }
  
  // Update threshold
  Future<bool> updateThreshold(String walletId, int newThreshold) async {
    final response = await http.put(
      Uri.parse('$API_BASE/multisig/wallets/$walletId/threshold'),
      headers: _headers,
      body: json.encode({'threshold': newThreshold}),
    );
    
    return response.statusCode == 200;
  }
}

class MultiSigWallet {
  final String id;
  final String name;
  final String address;
  final List<String> owners;
  final int threshold;
  final String status;
  final double balance;
  final DateTime createdAt;
  
  MultiSigWallet({
    required this.id,
    required this.name,
    required this.address,
    required this.owners,
    required this.threshold,
    required this.status,
    required this.balance,
    required this.createdAt,
  });
  
  factory MultiSigWallet.fromJson(Map<String, dynamic> json) {
    return MultiSigWallet(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      address: json['address'] ?? '',
      owners: List<String>.from(json['owners'] ?? []),
      threshold: json['threshold'] ?? 1,
      status: json['status'] ?? 'ACTIVE',
      balance: (json['balance'] ?? 0).toDouble(),
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class MultiSigTx {
  final String id;
  final String walletId;
  final String to;
  final String token;
  final double amount;
  final String status;
  final int approvalCount;
  final int requiredApprovals;
  final String? txHash;
  final DateTime createdAt;
  final DateTime? executedAt;
  
  MultiSigTx({
    required this.id,
    required this.walletId,
    required this.to,
    required this.token,
    required this.amount,
    required this.status,
    required this.approvalCount,
    required this.requiredApprovals,
    this.txHash,
    required this.createdAt,
    this.executedAt,
  });
  
  factory MultiSigTx.fromJson(Map<String, dynamic> json) {
    return MultiSigTx(
      id: json['id'] ?? '',
      walletId: json['walletId'] ?? '',
      to: json['to'] ?? '',
      token: json['token'] ?? 'ETH',
      amount: (json['amount'] ?? 0).toDouble(),
      status: json['status'] ?? 'PENDING',
      approvalCount: json['approvalCount'] ?? 0,
      requiredApprovals: json['requiredApprovals'] ?? 1,
      txHash: json['txHash'],
      createdAt: DateTime.parse(json['createdAt']),
      executedAt: json['executedAt'] != null ? DateTime.parse(json['executedAt']) : null,
    );
  }
}
