// Account Abstraction Service - Flutter Mobile

import 'dart:convert';
import 'package:http/http.dart' as http;

class AccountAbstractionService {
  static const String API_BASE = 'https://api.tigerwallet.com/api/v1';
  String? _token;
  
  AccountAbstractionService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Create smart account
  Future<SmartAccount> createAccount({
    required String ownerAddress,
    String? salt,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/account/create'),
      headers: _headers,
      body: json.encode({
        'ownerAddress': ownerAddress,
        'salt': salt,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return SmartAccount.fromJson(data['data']);
    }
    throw Exception('Failed to create account');
  }
  
  // Execute user operation
  Future<UserOperation> executeOp(UserOperation op) async {
    final response = await http.post(
      Uri.parse('$API_BASE/account/execute'),
      headers: _headers,
      body: json.encode(op.toJson()),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return UserOperation.fromJson(data['data']);
    }
    throw Exception('Failed to execute operation');
  }
  
  // Get user's accounts
  Future<List<SmartAccount>> getAccounts() async {
    final response = await http.get(
      Uri.parse('$API_BASE/account/list'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((a) => SmartAccount.fromJson(a)).toList();
    }
    return [];
  }
  
  // Get account details
  Future<SmartAccount> getAccount(String address) async {
    final response = await http.get(
      Uri.parse('$API_BASE/account/$address'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return SmartAccount.fromJson(data['data']);
    }
    throw Exception('Failed to get account');
  }
  
  // Add signer
  Future<bool> addSigner(String accountAddress, String signerAddress, int weight) async {
    final response = await http.post(
      Uri.parse('$API_BASE/account/signers'),
      headers: _headers,
      body: json.encode({
        'accountAddress': accountAddress,
        'signerAddress': signerAddress,
        'weight': weight,
      }),
    );
    
    return response.statusCode == 201;
  }
  
  // Remove signer
  Future<bool> removeSigner(String accountAddress, String signerAddress) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/account/signers?account=$accountAddress&signer=$signerAddress'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get signers
  Future<List<AccountSigner>> getSigners(String accountAddress) async {
    final response = await http.get(
      Uri.parse('$API_BASE/account/signers?account=$accountAddress'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((s) => AccountSigner.fromJson(s)).toList();
    }
    return [];
  }
  
  // Add session key
  Future<bool> addSessionKey(String accountAddress, String sessionKey, Map<String, dynamic> permissions, DateTime expiresAt) async {
    final response = await http.post(
      Uri.parse('$API_BASE/account/sessions'),
      headers: _headers,
      body: json.encode({
        'accountAddress': accountAddress,
        'sessionKey': sessionKey,
        'permissions': permissions,
        'expiresAt': expiresAt.toIso8601String(),
      }),
    );
    
    return response.statusCode == 201;
  }
  
  // Verify session key
  Future<bool> verifySessionKey(String accountAddress, String sessionKey) async {
    final response = await http.get(
      Uri.parse('$API_BASE/account/sessions/verify?account=$accountAddress&key=$sessionKey'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return data['data'] ?? false;
    }
    return false;
  }
  
  // Get user operations
  Future<List<UserOperation>> getUserOps(String accountAddress) async {
    final response = await http.get(
      Uri.parse('$API_BASE/account/ops?account=$accountAddress'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((o) => UserOperation.fromJson(o)).toList();
    }
    return [];
  }
  
  // Enable paymaster
  Future<bool> enablePaymaster(String accountAddress, String paymasterAddress) async {
    final response = await http.post(
      Uri.parse('$API_BASE/account/paymaster'),
      headers: _headers,
      body: json.encode({
        'accountAddress': accountAddress,
        'paymasterAddress': paymasterAddress,
      }),
    );
    
    return response.statusCode == 201;
  }
}

class SmartAccount {
  final String id;
  final String accountAddress;
  final String ownerAddress;
  final int nonce;
  final int threshold;
  final String status;
  final bool deployed;
  final DateTime createdAt;
  
  SmartAccount({
    required this.id,
    required this.accountAddress,
    required this.ownerAddress,
    required this.nonce,
    required this.threshold,
    required this.status,
    required this.deployed,
    required this.createdAt,
  });
  
  factory SmartAccount.fromJson(Map<String, dynamic> json) {
    return SmartAccount(
      id: json['id'] ?? '',
      accountAddress: json['accountAddress'] ?? '',
      ownerAddress: json['ownerAddress'] ?? '',
      nonce: json['nonce'] ?? 0,
      threshold: json['threshold'] ?? 1,
      status: json['status'] ?? 'ACTIVE',
      deployed: json['deployed'] ?? false,
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class AccountSigner {
  final String id;
  final String signerAddress;
  final int weight;
  final String status;
  
  AccountSigner({
    required this.id,
    required this.signerAddress,
    required this.weight,
    required this.status,
  });
  
  factory AccountSigner.fromJson(Map<String, dynamic> json) {
    return AccountSigner(
      id: json['id'] ?? '',
      signerAddress: json['signerAddress'] ?? '',
      weight: json['weight'] ?? 1,
      status: json['status'] ?? 'ACTIVE',
    );
  }
}

class UserOperation {
  final String id;
  final String userOpHash;
  final String sender;
  final int nonce;
  final String? initCode;
  final String callData;
  final int callGasLimit;
  final int verificationGasLimit;
  final int preVerificationGas;
  final String maxFeePerGas;
  final String maxPriorityFeePerGas;
  final String signature;
  final String status;
  final DateTime createdAt;
  
  UserOperation({
    required this.id,
    required this.userOpHash,
    required this.sender,
    required this.nonce,
    this.initCode,
    required this.callData,
    required this.callGasLimit,
    required this.verificationGasLimit,
    required this.preVerificationGas,
    required this.maxFeePerGas,
    required this.maxPriorityFeePerGas,
    required this.signature,
    required this.status,
    required this.createdAt,
  });
  
  factory UserOperation.fromJson(Map<String, dynamic> json) {
    return UserOperation(
      id: json['id'] ?? '',
      userOpHash: json['userOpHash'] ?? '',
      sender: json['sender'] ?? '',
      nonce: json['nonce'] ?? 0,
      initCode: json['initCode'],
      callData: json['callData'] ?? '',
      callGasLimit: json['callGasLimit'] ?? 21000,
      verificationGasLimit: json['verificationGasLimit'] ?? 100000,
      preVerificationGas: json['preVerificationGas'] ?? 21000,
      maxFeePerGas: json['maxFeePerGas'] ?? '0',
      maxPriorityFeePerGas: json['maxPriorityFeePerGas'] ?? '0',
      signature: json['signature'] ?? '',
      status: json['status'] ?? 'PENDING',
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
  
  Map<String, dynamic> toJson() => {
    'sender': sender,
    'nonce': nonce,
    'initCode': initCode,
    'callData': callData,
    'callGasLimit': callGasLimit,
    'verificationGasLimit': verificationGasLimit,
    'preVerificationGas': preVerificationGas,
    'maxFeePerGas': maxFeePerGas,
    'maxPriorityFeePerGas': maxPriorityFeePerGas,
    'signature': signature,
  };
}
