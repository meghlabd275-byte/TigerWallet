// MasterWallet Multi-Sig Service - Flutter
//
// Thin REST client over the canonical Go backend (:8450). Multisig routes are
// nested under /api/v1/master-wallet/:id/multisig/... . No fabricated data;
// on backend failure methods throw rather than returning empty/fake results.

import 'dart:convert';
import 'package:http/http.dart' as http;

class MultiSigService {
  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );
  static const String _apiV1 = '$API_BASE/api/v1';

  final String masterWalletId;
  String? _token;

  MultiSigService({required this.masterWalletId, String? token}) : _token = token;

  void setToken(String? token) => _token = token;

  String get _base => '$_apiV1/master-wallet/$masterWalletId/multisig';

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Exception _err(http.Response r, String op) =>
      Exception('multisig $op failed (${r.statusCode}): ${r.body}');

  dynamic _body(http.Response r) {
    final data = json.decode(r.body);
    return data['data'] ?? data;
  }

  /// Create a multi-sig wallet (real backend).
  Future<MultiSigWallet> createWallet({
    required String name,
    required List<String> owners,
    required int threshold,
  }) async {
    final r = await http.post(
      Uri.parse('$_base/wallets'),
      headers: _headers,
      body: json.encode({'name': name, 'owners': owners, 'threshold': threshold}),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _err(r, 'createWallet');
    return MultiSigWallet.fromJson(_body(r) as Map<String, dynamic>);
  }

  /// List multi-sig wallets (real backend).
  Future<List<MultiSigWallet>> getWallets() async {
    final r = await http.get(Uri.parse('$_base/wallets'), headers: _headers);
    if (r.statusCode != 200) throw _err(r, 'getWallets');
    final list = (_body(r) as List?) ?? const [];
    return list.map((w) => MultiSigWallet.fromJson(w as Map<String, dynamic>)).toList();
  }

  /// Get wallet details (real backend).
  Future<MultiSigWallet> getWalletDetails(String walletId) async {
    final r = await http.get(
      Uri.parse('$_base/wallets/$walletId'),
      headers: _headers,
    );
    if (r.statusCode != 200) throw _err(r, 'getWalletDetails');
    return MultiSigWallet.fromJson(_body(r) as Map<String, dynamic>);
  }

  /// Create a multi-sig transaction proposal (real backend).
  Future<MultiSigTx> createTransaction({
    required String walletId,
    required String to,
    required double amount,
    String? token,
    String? data,
  }) async {
    final r = await http.post(
      Uri.parse('$_base/wallets/$walletId/transactions'),
      headers: _headers,
      body: json.encode({
        'to': to,
        'amount': amount,
        if (token != null) 'token': token,
        if (data != null) 'data': data,
      }),
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _err(r, 'createTransaction');
    return MultiSigTx.fromJson(_body(r) as Map<String, dynamic>);
  }

  /// Sign (approve) a multi-sig transaction. Canonical route is
  /// /multisig/transactions/:tid/sign.
  Future<bool> signTransaction(String txId) async {
    final r = await http.post(
      Uri.parse('$_base/transactions/$txId/sign'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _err(r, 'signTransaction');
    return true;
  }

  /// Execute a multi-sig transaction once the threshold is met.
  Future<bool> executeTransaction(String txId) async {
    final r = await http.post(
      Uri.parse('$_base/transactions/$txId/execute'),
      headers: _headers,
    );
    if (r.statusCode != 200 && r.statusCode != 201) throw _err(r, 'executeTransaction');
    return true;
  }

  /// List transactions for a multi-sig wallet, optionally filtered by status.
  Future<List<MultiSigTx>> getTransactions(String walletId, {String? status}) async {
    final q = Uri.parse('$_base/wallets/$walletId/transactions').replace(
      queryParameters: {if (status != null) 'status': status},
    );
    final r = await http.get(q, headers: _headers);
    if (r.statusCode != 200) throw _err(r, 'getTransactions');
    final list = (_body(r) as List?) ?? const [];
    return list.map((t) => MultiSigTx.fromJson(t as Map<String, dynamic>)).toList();
  }

  Future<List<MultiSigTx>> getPendingTransactions(String walletId) =>
      getTransactions(walletId, status: 'PENDING');

  Future<List<MultiSigTx>> getTransactionHistory(String walletId) =>
      getTransactions(walletId);

  // Owner / threshold management are not part of the canonical multisig
  // contract. They fail closed rather than posting to a non-existent route.
  Future<bool> addOwner(String walletId, String newOwner) async {
    throw UnimplementedError(
      'multisig addOwner is not supported by the canonical backend contract.',
    );
  }

  Future<bool> removeOwner(String walletId, String owner) async {
    throw UnimplementedError(
      'multisig removeOwner is not supported by the canonical backend contract.',
    );
  }

  Future<bool> updateThreshold(String walletId, int newThreshold) async {
    throw UnimplementedError(
      'multisig updateThreshold is not supported by the canonical backend contract.',
    );
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
