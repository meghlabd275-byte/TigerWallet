// Admin Platform Fee Management Service - Flutter

import 'dart:convert';
import 'package:http/http.dart' as http;

class FeeManagementService {
  static const String API_BASE = 'https://admin-api.tigerwallet.com/api/v1';
  String? _token;
  
  FeeManagementService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get all fee structures
  Future<List<FeeStructure>> getFeeStructures() async {
    final response = await http.get(
      Uri.parse('$API_BASE/fees'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((f) => FeeStructure.fromJson(f)).toList();
    }
    return [];
  }
  
  // Create fee structure
  Future<FeeStructure> createFeeStructure({
    required String type,
    required String token,
    required double makerFee,
    required double takerFee,
    required double withdrawalFee,
    double? depositFee,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/fees'),
      headers: _headers,
      body: json.encode({
        'type': type,
        'token': token,
        'makerFee': makerFee,
        'takerFee': takerFee,
        'withdrawalFee': withdrawalFee,
        'depositFee': depositFee,
      }),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return FeeStructure.fromJson(data['data']);
    }
    throw Exception('Failed to create fee structure');
  }
  
  // Update fee structure
  Future<FeeStructure> updateFeeStructure(String feeId, Map<String, dynamic> updates) async {
    final response = await http.put(
      Uri.parse('$API_BASE/fees/$feeId'),
      headers: _headers,
      body: json.encode(updates),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return FeeStructure.fromJson(data['data']);
    }
    throw Exception('Failed to update fee');
  }
  
  // Get fee breakdown for user
  Future<FeeBreakdown> calculateFee({
    required String type,
    required String token,
    required double amount,
    String? userId,
  }) async {
    final response = await http.post(
      Uri.parse('$API_BASE/fees/calculate'),
      headers: _headers,
      body: json.encode({
        'type': type,
        'token': token,
        'amount': amount,
        'userId': userId,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return FeeBreakdown.fromJson(data['data']);
    }
    throw Exception('Failed to calculate fee');
  }
  
  // Get fee revenue
  Future<FeeRevenue> getFeeRevenue({
    required DateTime startDate,
    required DateTime endDate,
  }) async {
    final response = await http.get(
      Uri.parse('$API_BASE/fees/revenue?start=${startDate.toIso8601String()}&end=${endDate.toIso8601String()}'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return FeeRevenue.fromJson(data['data']);
    }
    throw Exception('Failed to get revenue');
  }
  
  // Set VIP fee tier
  Future<bool> setVIPTier(String userId, int tier) async {
    final response = await http.post(
      Uri.parse('$API_BASE/fees/vip'),
      headers: _headers,
      body: json.encode({'userId': userId, 'tier': tier}),
    );
    
    return response.statusCode == 200;
  }
  
  // Get VIP tiers
  Future<List<VIPTier>> getVIPTiers() async {
    final response = await http.get(
      Uri.parse('$API_BASE/fees/vip-tiers'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((t) => VIPTier.fromJson(t)).toList();
    }
    return [];
  }
}

class FeeStructure {
  final String id;
  final String type;
  final String token;
  final double makerFee;
  final double takerFee;
  final double withdrawalFee;
  final double? depositFee;
  final bool isActive;
  final DateTime createdAt;
  
  FeeStructure({
    required this.id,
    required this.type,
    required this.token,
    required this.makerFee,
    required this.takerFee,
    required this.withdrawalFee,
    this.depositFee,
    required this.isActive,
    required this.createdAt,
  });
  
  factory FeeStructure.fromJson(Map<String, dynamic> json) {
    return FeeStructure(
      id: json['id'] ?? '',
      type: json['type'] ?? '',
      token: json['token'] ?? '',
      makerFee: (json['makerFee'] ?? 0).toDouble(),
      takerFee: (json['takerFee'] ?? 0).toDouble(),
      withdrawalFee: (json['withdrawalFee'] ?? 0).toDouble(),
      depositFee: json['depositFee']?.toDouble(),
      isActive: json['isActive'] ?? true,
      createdAt: DateTime.parse(json['createdAt']),
    );
  }
}

class FeeBreakdown {
  final double grossAmount;
  final double fee;
  final double netAmount;
  final String feeType;
  final double feePercentage;
  
  FeeBreakdown({
    required this.grossAmount,
    required this.fee,
    required this.netAmount,
    required this.feeType,
    required this.feePercentage,
  });
  
  factory FeeBreakdown.fromJson(Map<String, dynamic> json) {
    return FeeBreakdown(
      grossAmount: (json['grossAmount'] ?? 0).toDouble(),
      fee: (json['fee'] ?? 0).toDouble(),
      netAmount: (json['netAmount'] ?? 0).toDouble(),
      feeType: json['feeType'] ?? '',
      feePercentage: (json['feePercentage'] ?? 0).toDouble(),
    );
  }
}

class FeeRevenue {
  final double totalRevenue;
  final double makerRevenue;
  final double takerRevenue;
  final double withdrawalRevenue;
  final double depositRevenue;
  final Map<String, double> byToken;
  
  FeeRevenue({
    required this.totalRevenue,
    required this.makerRevenue,
    required this.takerRevenue,
    required this.withdrawalRevenue,
    required this.depositRevenue,
    required this.byToken,
  });
  
  factory FeeRevenue.fromJson(Map<String, dynamic> json) {
    return FeeRevenue(
      totalRevenue: (json['totalRevenue'] ?? 0).toDouble(),
      makerRevenue: (json['makerRevenue'] ?? 0).toDouble(),
      takerRevenue: (json['takerRevenue'] ?? 0).toDouble(),
      withdrawalRevenue: (json['withdrawalRevenue'] ?? 0).toDouble(),
      depositRevenue: (json['depositRevenue'] ?? 0).toDouble(),
      byToken: Map<String, double>.from(json['byToken'] ?? {}),
    );
  }
}

class VIPTier {
  final int tier;
  final String name;
  final double makerDiscount;
  final double takerDiscount;
  final double? minVolume;
  
  VIPTier({
    required this.tier,
    required this.name,
    required this.makerDiscount,
    required this.takerDiscount,
    this.minVolume,
  });
  
  factory VIPTier.fromJson(Map<String, dynamic> json) {
    return VIPTier(
      tier: json['tier'] ?? 0,
      name: json['name'] ?? '',
      makerDiscount: (json['makerDiscount'] ?? 0).toDouble(),
      takerDiscount: (json['takerDiscount'] ?? 0).toDouble(),
      minVolume: json['minVolume']?.toDouble(),
    );
  }
}
