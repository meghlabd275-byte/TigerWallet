// Bridge Service - Flutter Mobile
// Cross-chain bridging with real backend

import 'dart:convert';
import 'package:http/http.dart' as http;

class BridgeService {
  static const String API_BASE = 'https://api.tigerwallet.com/api/v1';
  String? _token;
  
  BridgeService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get supported chains
  Future<List<Chain>> getSupportedChains() async {
    final response = await http.get(
      Uri.parse('$API_BASE/bridge/chains'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((c) => Chain.fromJson(c)).toList();
    }
    throw Exception('Failed to load chains');
  }
  
  // Get supported tokens for a chain
  Future<List<BridgeToken>> getTokens(String chain) async {
    final response = await http.get(
      Uri.parse('$API_BASE/bridge/tokens?chain=$chain'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((t) => BridgeToken.fromJson(t)).toList();
    }
    throw Exception('Failed to load tokens');
  }
  
  // Get estimated receive amount
  Future<BridgeEstimate> getEstimate({
    required String fromChain,
    required String toChain,
    required String token,
    required double amount,
  }) async {
    final response = await http.get(
      Uri.parse(
        '$API_BASE/bridge/estimate?fromChain=$fromChain&toChain=$toChain&token=$token&amount=$amount',
      ),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return BridgeEstimate.fromJson(data['data']);
    }
    throw Exception('Failed to get estimate');
  }
  
  // Initiate bridge transaction
  Future<BridgeTransaction> initiateBridge({
    required String fromChain,
    required String toChain,
    required String token,
    required double amount,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/bridge/transactions'),
      headers: _headers,
      body: json.encode({
        'fromChain': fromChain,
        'toChain': toChain,
        'token': token,
        'amount': amount,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return BridgeTransaction.fromJson(data['data']);
    }
    throw Exception('Failed to initiate bridge');
  }
  
  // Confirm source deposit (after user sends tokens)
  Future<bool> confirmDeposit(String txId, String sourceTxHash) async {
    final response = await http.post(
      Uri.parse('$API_BASE/bridge/transactions/$txId/confirm'),
      headers: _headers,
      body: json.encode({'sourceTxHash': sourceTxHash}),
    );
    
    return response.statusCode == 200;
  }
  
  // Complete bridge (after bridge completes on dest chain)
  Future<bool> completeBridge(String txId, String destTxHash) async {
    final response = await http.post(
      Uri.parse('$API_BASE/bridge/transactions/$txId/complete'),
      headers: _headers,
      body: json.encode({'destTxHash': destTxHash}),
    );
    
    return response.statusCode == 200;
  }
  
  // Cancel bridge
  Future<bool> cancelBridge(String txId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/bridge/transactions/$txId/cancel'),
      headers: _headers,
    );
    
    return response.statusCode == 200;
  }
  
  // Get transaction status
  Future<BridgeTransaction> getTransactionStatus(String txId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/bridge/transactions/$txId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return BridgeTransaction.fromJson(data['data']);
    }
    throw Exception('Failed to get transaction');
  }
  
  // Get user transactions
  Future<List<BridgeTransaction>> getUserTransactions() async {
    final response = await http.get(
      Uri.parse('$API_BASE/bridge/transactions'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((t) => BridgeTransaction.fromJson(t)).toList();
    }
    return [];
  }
  
  // Get bridge statistics
  Future<BridgeStats> getStats() async {
    final response = await http.get(
      Uri.parse('$API_BASE/bridge/stats'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return BridgeStats.fromJson(data['data']);
    }
    throw Exception('Failed to load stats');
  }
}

class Chain {
  final String id;
  final String name;
  final String symbol;
  final String icon;
  final bool isActive;
  final List<String> features;
  
  Chain({
    required this.id,
    required this.name,
    required this.symbol,
    required this.icon,
    required this.isActive,
    required this.features,
  });
  
  factory Chain.fromJson(Map<String, dynamic> json) {
    return Chain(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      symbol: json['symbol'] ?? '',
      icon: json['icon'] ?? '',
      isActive: json['isActive'] ?? false,
      features: List<String>.from(json['features'] ?? []),
    );
  }
}

class BridgeToken {
  final String token;
  final String name;
  final String symbol;
  final String icon;
  final double minAmount;
  final double maxAmount;
  final bool isActive;
  
  BridgeToken({
    required this.token,
    required this.name,
    required this.symbol,
    required this.icon,
    required this.minAmount,
    required this.maxAmount,
    required this.isActive,
  });
  
  factory BridgeToken.fromJson(Map<String, dynamic> json) {
    return BridgeToken(
      token: json['token'] ?? '',
      name: json['name'] ?? '',
      symbol: json['symbol'] ?? '',
      icon: json['icon'] ?? '',
      minAmount: (json['minAmount'] ?? 0).toDouble(),
      maxAmount: (json['maxAmount'] ?? 0).toDouble(),
      isActive: json['isActive'] ?? false,
    );
  }
}

class BridgeEstimate {
  final double receivedAmount;
  final double fee;
  final double feePercentage;
  final String estimatedTime;
  
  BridgeEstimate({
    required this.receivedAmount,
    required this.fee,
    required this.feePercentage,
    required this.estimatedTime,
  });
  
  factory BridgeEstimate.fromJson(Map<String, dynamic> json) {
    return BridgeEstimate(
      receivedAmount: (json['receivedAmount'] ?? 0).toDouble(),
      fee: (json['fee'] ?? 0).toDouble(),
      feePercentage: (json['feePercentage'] ?? 0).toDouble(),
      estimatedTime: json['estimatedTime'] ?? '',
    );
  }
}

class BridgeTransaction {
  final String id;
  final String fromChain;
  final String toChain;
  final String token;
  final double amount;
  final double fee;
  final double receivedAmount;
  final String status;
  final String? sourceTxHash;
  final String? destTxHash;
  final DateTime createdAt;
  final DateTime? updatedAt;
  
  BridgeTransaction({
    required this.id,
    required this.fromChain,
    required this.toChain,
    required this.token,
    required this.amount,
    required this.fee,
    required this.receivedAmount,
    required this.status,
    this.sourceTxHash,
    this.destTxHash,
    required this.createdAt,
    this.updatedAt,
  });
  
  factory BridgeTransaction.fromJson(Map<String, dynamic> json) {
    return BridgeTransaction(
      id: json['id'] ?? '',
      fromChain: json['fromChain'] ?? '',
      toChain: json['toChain'] ?? '',
      token: json['token'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
      fee: (json['fee'] ?? 0).toDouble(),
      receivedAmount: (json['receivedAmount'] ?? 0).toDouble(),
      status: json['status'] ?? 'PENDING',
      sourceTxHash: json['sourceTxHash'],
      destTxHash: json['destTxHash'],
      createdAt: DateTime.parse(json['createdAt']),
      updatedAt: json['updatedAt'] != null ? DateTime.parse(json['updatedAt']) : null,
    );
  }
}

class BridgeStats {
  final double totalVolume;
  final int totalTransactions;
  final int supportedChains;
  final int supportedTokens;
  
  BridgeStats({
    required this.totalVolume,
    required this.totalTransactions,
    required this.supportedChains,
    required this.supportedTokens,
  });
  
  factory BridgeStats.fromJson(Map<String, dynamic> json) {
    return BridgeStats(
      totalVolume: (json['totalVolume'] ?? 0).toDouble(),
      totalTransactions: json['totalTransactions'] ?? 0,
      supportedChains: json['supportedChains'] ?? 0,
      supportedTokens: json['supportedTokens'] ?? 0,
    );
  }
}
