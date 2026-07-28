// Staking Service - Complete Staking Functionality

import '../../services/api_service.dart';

class StakingService {
  final ApiService _api = ApiService.instance;
  
  // Get staking pools
  Future<List<StakingPool>> getPools({String? chainId}) async {
    final response = await _api.get('/staking/pools', queryParams: {
      if (chainId != null) 'chainId': chainId,
    });
    
    if (response.success) {
      return (response.data as List).map((p) => StakingPool.fromJson(p)).toList();
    }
    return [];
  }
  
  // Get staking positions
  Future<List<StakingPosition>> getPositions(String walletAddress) async {
    final response = await _api.get('/staking/positions/$walletAddress');
    
    if (response.success) {
      return (response.data as List).map((p) => StakingPosition.fromJson(p)).toList();
    }
    return [];
  }
  
  // Stake tokens
  Future<StakingResult> stake({
    required String poolId,
    required String amount,
    required String walletAddress,
  }) async {
    final response = await _api.post('/staking/stake', body: {
      'poolId': poolId,
      'amount': amount,
      'walletAddress': walletAddress,
    });
    
    if (response.success) {
      return StakingResult.fromJson(response.data);
    }
    throw Exception(response.error);
  }
  
  // Unstake tokens
  Future<StakingResult> unstake({
    required String positionId,
    required String amount,
  }) async {
    final response = await _api.post('/staking/unstake', body: {
      'positionId': positionId,
      'amount': amount,
    });
    
    if (response.success) {
      return StakingResult.fromJson(response.data);
    }
    throw Exception(response.error);
  }
  
  // Claim rewards
  Future<StakingResult> claimRewards(String positionId) async {
    final response = await _api.post('/staking/claim/$positionId');
    
    if (response.success) {
      return StakingResult.fromJson(response.data);
    }
    throw Exception(response.error);
  }
  
  // Get rewards info
  Future<StakingRewards> getRewardsInfo(String positionId) async {
    final response = await _api.get('/staking/rewards/$positionId');
    
    if (response.success) {
      return StakingRewards.fromJson(response.data);
    }
    throw Exception(response.error);
  }
}

class StakingPool {
  final String id;
  final String name;
  final String token;
  final String rewardToken;
  final String chainId;
  final String apy;
  final String minStake;
  final String lockPeriod;
  final bool isActive;
  final String totalStaked;
  final List<StakingTier> tiers;
  
  StakingPool({
    required this.id,
    required this.name,
    required this.token,
    required this.rewardToken,
    required this.chainId,
    required this.apy,
    required this.minStake,
    required this.lockPeriod,
    required this.isActive,
    required this.totalStaked,
    required this.tiers,
  });
  
  factory StakingPool.fromJson(Map<String, dynamic> json) {
    return StakingPool(
      id: json['id'],
      name: json['name'],
      token: json['token'],
      rewardToken: json['rewardToken'],
      chainId: json['chainId'],
      apy: json['apy'],
      minStake: json['minStake'],
      lockPeriod: json['lockPeriod'],
      isActive: json['isActive'] ?? true,
      totalStaked: json['totalStaked'],
      tiers: (json['tiers'] as List?)
          ?.map((t) => StakingTier.fromJson(t))
          .toList() ?? [],
    );
  }
}

class StakingTier {
  final String name;
  final String minAmount;
  final String apy;
  final String lockDays;
  
  StakingTier({
    required this.name,
    required this.minAmount,
    required this.apy,
    required this.lockDays,
  });
  
  factory StakingTier.fromJson(Map<String, dynamic> json) {
    return StakingTier(
      name: json['name'],
      minAmount: json['minAmount'],
      apy: json['apy'],
      lockDays: json['lockDays'],
    );
  }
}

class StakingPosition {
  final String id;
  final String poolId;
  final String poolName;
  final String stakedAmount;
  final String rewardAmount;
  final String startTime;
  final String unlockTime;
  final String status;
  
  StakingPosition({
    required this.id,
    required this.poolId,
    required this.poolName,
    required this.stakedAmount,
    required this.rewardAmount,
    required this.startTime,
    required this.unlockTime,
    required this.status,
  });
  
  factory StakingPosition.fromJson(Map<String, dynamic> json) {
    return StakingPosition(
      id: json['id'],
      poolId: json['poolId'],
      poolName: json['poolName'],
      stakedAmount: json['stakedAmount'],
      rewardAmount: json['rewardAmount'],
      startTime: json['startTime'],
      unlockTime: json['unlockTime'],
      status: json['status'],
    );
  }
}

class StakingResult {
  final String txHash;
  final String status;
  final String positionId;
  
  StakingResult({
    required this.txHash,
    required this.status,
    required this.positionId,
  });
  
  factory StakingResult.fromJson(Map<String, dynamic> json) {
    return StakingResult(
      txHash: json['txHash'],
      status: json['status'],
      positionId: json['positionId'],
    );
  }
}

class StakingRewards {
  final String pendingRewards;
  final String totalClaimed;
  final String nextClaimTime;
  
  StakingRewards({
    required this.pendingRewards,
    required this.totalClaimed,
    required this.nextClaimTime,
  });
  
  factory StakingRewards.fromJson(Map<String, dynamic> json) {
    return StakingRewards(
      pendingRewards: json['pendingRewards'],
      totalClaimed: json['totalClaimed'],
      nextClaimTime: json['nextClaimTime'],
    );
  }
}
