// MasterWallet Batch Transaction Service - Flutter

import 'dart:convert';
import 'package:http/http.dart' as http;

class BatchTransactionService {
  static const String API_BASE = 'https://master-api.tigerwallet.com/api/v1';
  String? _token;
  
  BatchTransactionService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Create batch transaction
  Future<BatchTransaction> createBatch({
    required String name,
    required List<BatchTxItem> transactions,
    required int approvalRequired,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/batch/transactions'),
      headers: _headers,
      body: json.encode({
        'name': name,
        'transactions': transactions.map((t) => t.toJson()).toList(),
        'approvalRequired': approvalRequired,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return BatchTransaction.fromJson(data['data']);
    }
    throw Exception('Failed to create batch');
  }
  
  // Get batch transactions
  Future<List<BatchTransaction>> getBatches() async {
    final response = await http.get(
      Uri.parse('$API_BASE/batch/transactions'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((b) => BatchTransaction.fromJson(b)).toList();
    }
    return [];
  }
  
  // Get batch details
  Future<BatchTransaction> getBatchDetails(String batchId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/batch/transactions/$batchId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return BatchTransaction.fromJson(data['data']);
    }
    throw Exception('Failed to get batch');
  }
  
  // Approve batch
  Future<bool> approveBatch(String batchId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/batch/transactions/$batchId/approve'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Execute batch
  Future<bool> executeBatch(String batchId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/batch/transactions/$batchId/execute'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Cancel batch
  Future<bool> cancelBatch(String batchId) async {
    final response = await http.delete(
      Uri.parse('$API_BASE/batch/transactions/$batchId'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get batch status
  Future<BatchStatus> getBatchStatus(String batchId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/batch/transactions/$batchId/status'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return BatchStatus.fromJson(data['data']);
    }
    throw Exception('Failed to get status');
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
