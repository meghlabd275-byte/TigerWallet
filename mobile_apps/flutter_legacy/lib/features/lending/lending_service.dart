// Lending Service - Flutter Mobile
// Complete lending/borrowing with real backend

import 'dart:convert';
import 'package:http/http.dart' as http;

class LendingService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  LendingService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get all lending pools
  Future<List<LendingPool>> getPools() async {
    final response = await http.get(
      Uri.parse('$API_BASE/lending/pools'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((p) => LendingPool.fromJson(p)).toList();
    }
    return [];
  }
  
  // Get pool data for specific token
  Future<LendingPool> getPool(String token) async {
    final response = await http.get(
      Uri.parse('$API_BASE/lending/pools/$token'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return LendingPool.fromJson(data['data']);
    }
    throw Exception('Failed to load pool');
  }
  
  // Supply assets to pool
  Future<LendingPosition> supply(String token, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/lending/supply'),
      headers: _headers,
      body: json.encode({
        'token': token,
        'amount': amount,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return LendingPosition.fromJson(data['data']);
    }
    throw Exception('Failed to supply');
  }
  
  // Borrow from pool
  Future<LendingPosition> borrow(String token, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/lending/borrow'),
      headers: _headers,
      body: json.encode({
        'token': token,
        'amount': amount,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return LendingPosition.fromJson(data['data']);
    }
    throw Exception('Failed to borrow');
  }
  
  // Repay borrowed amount
  Future<bool> repay(String token, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/lending/repay'),
      headers: _headers,
      body: json.encode({
        'token': token,
        'amount': amount,
      }),
    );
    
    return response.statusCode == 200;
  }
  
  // Withdraw supplied assets
  Future<bool> withdraw(String token, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/lending/withdraw'),
      headers: _headers,
      body: json.encode({
        'token': token,
        'amount': amount,
      }),
    );
    
    return response.statusCode == 200;
  }
  
  // Get user's positions
  Future<List<LendingPosition>> getUserPositions() async {
    final response = await http.get(
      Uri.parse('$API_BASE/lending/positions'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((p) => LendingPosition.fromJson(p)).toList();
    }
    return [];
  }
  
  // Get health factor
  Future<double> getHealthFactor() async {
    final response = await http.get(
      Uri.parse('$API_BASE/lending/health'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] ?? 0).toDouble();
    }
    return 999.0;
  }
  
  // Liquidate position
  Future<bool> liquidate(String userId, String token) async {
    final response = await http.post(
      Uri.parse('$API_BASE/lending/liquidate'),
      headers: _headers,
      body: json.encode({
        'userId': userId,
        'token': token,
      }),
    );
    
    return response.statusCode == 200;
  }
}

class LendingPool {
  final String token;
  final String name;
  final String symbol;
  final String icon;
  final double totalSupplied;
  final double totalBorrowed;
  final double supplyAPY;
  final double borrowAPY;
  final double liquidity;
  final DateTime updatedAt;
  
  LendingPool({
    required this.token,
    required this.name,
    required this.symbol,
    required this.icon,
    required this.totalSupplied,
    required this.totalBorrowed,
    required this.supplyAPY,
    required this.borrowAPY,
    required this.liquidity,
    required this.updatedAt,
  });
  
  factory LendingPool.fromJson(Map<String, dynamic> json) {
    return LendingPool(
      token: json['token'] ?? '',
      name: json['name'] ?? '',
      symbol: json['symbol'] ?? '',
      icon: json['icon'] ?? '',
      totalSupplied: (json['totalSupplied'] ?? 0).toDouble(),
      totalBorrowed: (json['totalBorrowed'] ?? 0).toDouble(),
      supplyAPY: (json['supplyAPY'] ?? 0).toDouble(),
      borrowAPY: (json['borrowAPY'] ?? 0).toDouble(),
      liquidity: (json['liquidity'] ?? 0).toDouble(),
      updatedAt: DateTime.parse(json['updatedAt']),
    );
  }
}

class LendingPosition {
  final String id;
  final String token;
  final double supplied;
  final double borrowed;
  final double apy;
  final double accumulated;
  final String status;
  final DateTime suppliedAt;
  
  LendingPosition({
    required this.id,
    required this.token,
    required this.supplied,
    required this.borrowed,
    required this.apy,
    required this.accumulated,
    required this.status,
    required this.suppliedAt,
  });
  
  factory LendingPosition.fromJson(Map<String, dynamic> json) {
    return LendingPosition(
      id: json['id'] ?? '',
      token: json['token'] ?? '',
      supplied: (json['supplied'] ?? 0).toDouble(),
      borrowed: (json['borrowed'] ?? 0).toDouble(),
      apy: (json['apy'] ?? 0).toDouble(),
      accumulated: (json['accumulated'] ?? 0).toDouble(),
      status: json['status'] ?? 'ACTIVE',
      suppliedAt: DateTime.parse(json['suppliedAt']),
    );
  }
}
