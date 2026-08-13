// MasterWallet Batch Transaction Service - Flutter
//
// The canonical Go backend (:8450) exposes single-transaction sign/broadcast
// at POST /api/v1/master-wallet/:id/transactions and multi-signature
// proposals under /master-wallet/:id/multisig/... . There is NO atomic
// "batch transaction" endpoint in the canonical contract, so this client
// fails closed for batch-specific operations rather than fabricating batch
// state or silently fanning out to single transactions (which would lose
// atomicity). Callers that need multi-tx flows should use MultiSigService.

import 'dart:convert';
import 'package:http/http.dart' as http;

// (Imports retained for API_BASE/JSON helpers if future canonical batch
// endpoints are added; currently all operations fail closed with no I/O.)

class BatchTransactionService {
  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );
  static const String _apiV1 = '$API_BASE/api/v1';

  final String masterWalletId;
  String? _token;

  BatchTransactionService({required this.masterWalletId, String? token})
      : _token = token;

  void setToken(String? token) => _token = token;

  String get _txBase => '$_apiV1/master-wallet/$masterWalletId/transactions';

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Exception _unsupported(String op) => UnimplementedError(
        'batch $op is not supported by the canonical backend contract. '
        'There is no atomic batch-transaction endpoint; use single '
        '/master-wallet/:id/transactions or MultiSigService for multi-tx flows.');

  /// Create a batch transaction. NOT supported by the canonical backend.
  Future<BatchTransaction> createBatch({
    required String name,
    required List<BatchTxItem> transactions,
    required int approvalRequired,
  }) async {
    throw _unsupported('createBatch');
  }

  /// List batch transactions. NOT supported by the canonical backend.
  Future<List<BatchTransaction>> getBatches() async {
    throw _unsupported('getBatches');
  }

  /// Get batch details. NOT supported by the canonical backend.
  Future<BatchTransaction> getBatchDetails(String batchId) async {
    throw _unsupported('getBatchDetails');
  }

  /// Approve a batch. NOT supported by the canonical backend.
  Future<bool> approveBatch(String batchId) async {
    throw _unsupported('approveBatch');
  }

  /// Execute a batch. NOT supported by the canonical backend.
  Future<bool> executeBatch(String batchId) async {
    throw _unsupported('executeBatch');
  }

  /// Cancel a batch. NOT supported by the canonical backend.
  Future<bool> cancelBatch(String batchId) async {
    throw _unsupported('cancelBatch');
  }

  /// Get batch status. NOT supported by the canonical backend.
  Future<BatchStatus> getBatchStatus(String batchId) async {
    throw _unsupported('getBatchStatus');
  }
}

class BatchTxItem {
  final String to;
  final String token;
  final double amount;
  final String? data;
  
  BatchTxItem({
    required this.to,
    required this.token,
    required this.amount,
    this.data,
  });
  
  Map<String, dynamic> toJson() => {
    'to': to,
    'token': token,
    'amount': amount,
    'data': data,
  };
}

class BatchTransaction {
  final String id;
  final String name;
  final List<BatchTxItem> transactions;
  final String status;
  final int approvalRequired;
  final int approvalCount;
  final int totalCount;
  final int successCount;
  final int failCount;
  final DateTime createdAt;
  final DateTime? executedAt;
  
  BatchTransaction({
    required this.id,
    required this.name,
    required this.transactions,
    required this.status,
    required this.approvalRequired,
    required this.approvalCount,
    required this.totalCount,
    required this.successCount,
    required this.failCount,
    required this.createdAt,
    this.executedAt,
  });
  
  factory BatchTransaction.fromJson(Map<String, dynamic> json) {
    return BatchTransaction(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      transactions: (json['transactions'] as List? ?? [])
          .map((t) => BatchTxItem(
            to: t['to'] ?? '',
            token: t['token'] ?? 'ETH',
            amount: (t['amount'] ?? 0).toDouble(),
            data: t['data'],
          )).toList(),
      status: json['status'] ?? 'PENDING',
      approvalRequired: json['approvalRequired'] ?? 1,
      approvalCount: json['approvalCount'] ?? 0,
      totalCount: json['totalCount'] ?? 0,
      successCount: json['successCount'] ?? 0,
      failCount: json['failCount'] ?? 0,
      createdAt: DateTime.parse(json['createdAt']),
      executedAt: json['executedAt'] != null ? DateTime.parse(json['executedAt']) : null,
    );
  }
}

class BatchStatus {
  final String batchId;
  final String status;
  final int pending;
  final int processing;
  final int success;
  final int failed;
  final List<String> txHashes;
  
  BatchStatus({
    required this.batchId,
    required this.status,
    required this.pending,
    required this.processing,
    required this.success,
    required this.failed,
    required this.txHashes,
  });
  
  factory BatchStatus.fromJson(Map<String, dynamic> json) {
    return BatchStatus(
      batchId: json['batchId'] ?? '',
      status: json['status'] ?? '',
      pending: json['pending'] ?? 0,
      processing: json['processing'] ?? 0,
      success: json['success'] ?? 0,
      failed: json['failed'] ?? 0,
      txHashes: List<String>.from(json['txHashes'] ?? []),
    );
  }
}
