// Liquid Staking Service - Flutter Mobile
// Complete liquid staking with real backend

import 'dart:convert';
import 'package:http/http.dart' as http;

class LiquidStakingService {
  static const String API_BASE = 'http://localhost:8443/api/v1';
  String? _token;
  
  LiquidStakingService({String? token}) : _token = token;
  
  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (_token != null) 'Authorization': 'Bearer $_token',
  };
  
  // Get available liquid staking pools
  Future<List<LiquidStakingPool>> getPools() async {
    final response = await http.get(
      Uri.parse('$API_BASE/liquid-staking/pools'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((p) => LiquidStakingPool.fromJson(p)).toList();
    }
    return [];
  }
  
  // Stake tokens and receive liquid token
  Future<StakingPosition> stake(String poolId, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/liquid-staking/stake'),
      headers: _headers,
      body: json.encode({'poolId': poolId, 'amount': amount}),
    );
    
    if (response.statusCode == 201) {
      final data = json.decode(response.body);
      return StakingPosition.fromJson(data['data']);
    }
    throw Exception('Failed to stake');
  }
  
  // Unstake (burn liquid token)
  Future<bool> unstake(String poolId, double amount) async {
    final response = await http.post(
      Uri.parse('$API_BASE/liquid-staking/unstake'),
      headers: _headers,
      body: json.encode({'poolId': poolId, 'amount': amount}),
    );
    
    return response.statusCode == 200;
  }
  
  // Claim rewards
  Future<double> claimRewards(String poolId) async {
    final response = await http.post(
      Uri.parse('$API_BASE/liquid-staking/claim'),
      headers: _headers,
      body: json.encode({'poolId': poolId}),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] ?? 0).toDouble();
    }
    return 0;
  }
  
  // Get user positions
  Future<List<StakingPosition>> getUserPositions() async {
    final response = await http.get(
      Uri.parse('$API_BASE/liquid-staking/positions'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] as List).map((p) => StakingPosition.fromJson(p)).toList();
    }
    return [];
  }
  
  // Get rewards
  Future<double> getPendingRewards(String poolId) async {
    final response = await http.get(
      Uri.parse('$API_BASE/liquid-staking/rewards?poolId=$poolId'),
      headers: _headers,
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return (data['data'] ?? 0).toDouble();
    }
    return 0;
  }
}

class LiquidStakingPool {
  final String id;
  final String name;
  final String token;
  final String liquidToken;
  final double apy;
  final double totalStaked;
  final double rewards;
  final bool isActive;
  
  LiquidStakingPool({
    required this.id,
    required this.name,
    required this.token,
    required this.liquidToken,
    required this.apy,
    required this.totalStaked,
    required this.rewards,
    required this.isActive,
  });
  
  factory LiquidStakingPool.fromJson(Map<String, dynamic> json) {
    return LiquidStakingPool(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      token: json['token'] ?? '',
      liquidToken: json['liquidToken'] ?? '',
      apy: (json['apy'] ?? 0).toDouble(),
      totalStaked: (json['totalStaked'] ?? 0).toDouble(),
      rewards: (json['rewards'] ?? 0).toDouble(),
      isActive: json['isActive'] ?? false,
    );
  }
}

class StakingPosition {
  final String id;
  final String poolId;
  final double stakedAmount;
  final double liquidTokenAmount;
  final double pendingRewards;
  final double claimedRewards;
  final DateTime stakedAt;
  
  StakingPosition({
    required this.id,
    required this.poolId,
    required this.stakedAmount,
    required this.liquidTokenAmount,
    required this.pendingRewards,
    required this.claimedRewards,
    required this.stakedAt,
  });
  
  factory StakingPosition.fromJson(Map<String, dynamic> json) {
    return StakingPosition(
      id: json['id'] ?? '',
      poolId: json['poolId'] ?? '',
      stakedAmount: (json['stakedAmount'] ?? 0).toDouble(),
      liquidTokenAmount: (json['liquidTokenAmount'] ?? 0).toDouble(),
      pendingRewards: (json['pendingRewards'] ?? 0).toDouble(),
      claimedRewards: (json['claimedRewards'] ?? 0).toDouble(),
      stakedAt: DateTime.parse(json['stakedAt']),
    );
  }
}
