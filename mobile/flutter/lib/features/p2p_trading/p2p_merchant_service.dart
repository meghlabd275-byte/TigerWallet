// P2P Merchant Service - Flutter Implementation
// Merchant system with collateral and security deposits

class P2PMerchant {
  final String id;
  final String userId;
  final String username;
  final String avatar;
  final String status; // PENDING, ACTIVE, SUSPENDED, BANNED
  final double collateralAmount;
  final String collateralToken;
  final double totalTrades;
  final double totalVolume;
  final double completionRate;
  final double avgResponseTime;
  final double avgReleaseTime;
  final double rating;
  final int totalReviews;
  final DateTime joinedAt;
  final DateTime? lastActiveAt;
  final bool isVerified;
  final String? kycLevel;

  P2PMerchant({
    required this.id,
    required this.userId,
    required this.username,
    required this.avatar,
    required this.status,
    required this.collateralAmount,
    required this.collateralToken,
    required this.totalTrades,
    required this.totalVolume,
    required this.completionRate,
    required this.avgResponseTime,
    required this.avgReleaseTime,
    required this.rating,
    required this.totalReviews,
    required this.joinedAt,
    this.lastActiveAt,
    required this.isVerified,
    this.kycLevel,
  });
}

class MerchantCollateral {
  final String id;
  final String merchantId;
  final String token;
  final double amount;
  final double usdValue;
  final String status; // LOCKED, RELEASED, SLASHED
  final DateTime lockedAt;
  final DateTime? unlockedAt;
  final String? reason;

  MerchantCollateral({
    required this.id,
    required this.merchantId,
    required this.token,
    required this.amount,
    required this.usdValue,
    required this.status,
    required this.lockedAt,
    this.unlockedAt,
    this.reason,
  });
}

class MerchantStats {
  final double totalTrades;
  final double totalVolume;
  final double completedTrades;
  final double cancelledTrades;
  final double disputeCount;
  final double avgRating;
  final int totalReviews;
  final double avgResponseTime; // minutes
  final double avgReleaseTime; // minutes
  final String traderLevel; // NEWBIE, BRONZE, SILVER, GOLD, PLATINUM, DIAMOND

  MerchantStats({
    required this.totalTrades,
    required this.totalVolume,
    required this.completedTrades,
    required this.cancelledTrades,
    required this.disputeCount,
    required this.avgRating,
    required this.totalReviews,
    required this.avgResponseTime,
    required this.avgReleaseTime,
    required this.traderLevel,
  });
}

class P2PSecurityDeposit {
  final String id;
  final String userId;
  final String orderId;
  final String token;
  final double amount;
  final double usdValue;
  final String type; // BUYER_PROTECTION, SELLER_BOND
  final String status; // LOCKED, RELEASED, FORFEITED
  final DateTime lockedAt;
  final DateTime? releasedAt;

  P2PSecurityDeposit({
    required this.id,
    required this.userId,
    required this.orderId,
    required this.token,
    required this.amount,
    required this.usdValue,
    required this.type,
    required this.status,
    required this.lockedAt,
    this.releasedAt,
  });
}

class AntiFraudProtection {
  // Check if user has sufficient trading history
  static bool canTradeWithoutVerification({
    required double tradeAmount,
    required double userCompletionRate,
    required int userTotalTrades,
  }) {
    // For small amounts (< $100), allow trading with lower requirements
    if (tradeAmount < 100) {
      return userCompletionRate >= 80 && userTotalTrades >= 5;
    }
    // For medium amounts ($100-$1000), require better history
    if (tradeAmount < 1000) {
      return userCompletionRate >= 90 && userTotalTrades >= 20;
    }
    // For large amounts, require excellent history + verification
    return userCompletionRate >= 95 && userTotalTrades >= 50;
  }

  // Calculate required collateral based on volume
  static double calculateRequiredCollateral(double monthlyVolume) {
    if (monthlyVolume < 1000) return 100; // $100 for < $1k/month
    if (monthlyVolume < 10000) return 500; // $500 for < $10k/month
    if (monthlyVolume < 50000) return 2000; // $2k for < $50k/month
    if (monthlyVolume < 100000) return 5000; // $5k for < $100k/month
    return 10000; // $10k for > $100k/month
  }

  // Get trader level based on total volume
  static String getTraderLevel(double totalVolume) {
    if (totalVolume < 1000) return 'NEWBIE';
    if (totalVolume < 10000) return 'BRONZE';
    if (totalVolume < 50000) return 'SILVER';
    if (totalVolume < 100000) return 'GOLD';
    if (totalVolume < 500000) return 'PLATINUM';
    return 'DIAMOND';
  }

  // Calculate security deposit amount
  static double calculateSecurityDeposit({
    required double tradeAmount,
    required String traderLevel,
    required double completionRate,
  }) {
    double basePercent = 5.0; // 5% base
    
    // Reduce based on trader level
    switch (traderLevel) {
      case 'DIAMOND': basePercent = 1.0; break;
      case 'PLATINUM': basePercent = 2.0; break;
      case 'GOLD': basePercent = 3.0; break;
      case 'SILVER': basePercent = 4.0; break;
      default: basePercent = 5.0;
    }
    
    // Reduce based on completion rate (higher = lower deposit)
    if (completionRate >= 99) basePercent *= 0.5;
    else if (completionRate >= 95) basePercent *= 0.75;
    
    return tradeAmount * (basePercent / 100);
  }
}

class P2PMerchantService {
  static final Map<String, P2PMerchant> _merchants = {};
  static final Map<String, MerchantCollateral> _collaterals = {};
  static final Map<String, MerchantStats> _stats = {};
  static final Map<String, List<P2PSecurityDeposit>> _deposits = {};

  // Minimum collateral requirements by level
  static const Map<String, double> collateralRequirements = {
    'NEWBIE': 100,
    'BRONZE': 250,
    'SILVER': 500,
    'GOLD': 1000,
    'PLATINUM': 2500,
    'DIAMOND': 5000,
  };

  // Apply to become a merchant
  static Future<P2PMerchant> applyAsMerchant({
    required String userId,
    required String username,
    required String avatar,
    required String collateralToken,
    required double collateralAmount,
  }) async {
    final merchant = P2PMerchant(
      id: 'merchant_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      username: username,
      avatar: avatar,
      status: 'PENDING',
      collateralAmount: collateralAmount,
      collateralToken: collateralToken,
      totalTrades: 0,
      totalVolume: 0,
      completionRate: 0,
      avgResponseTime: 0,
      avgReleaseTime: 0,
      rating: 0,
      totalReviews: 0,
      joinedAt: DateTime.now(),
      isVerified: false,
    );
    
    _merchants[userId] = merchant;
    
    // Lock collateral
    final collateral = MerchantCollateral(
      id: 'col_${DateTime.now().millisecondsSinceEpoch}',
      merchantId: merchant.id,
      token: collateralToken,
      amount: collateralAmount,
      usdValue: collateralAmount * 43250, // BTC price
      status: 'LOCKED',
      lockedAt: DateTime.now(),
    );
    _collaterals[merchant.id] = collateral;
    
    return merchant;
  }

  // Get merchant profile
  static Future<P2PMerchant?> getMerchant(String userId) async {
    return _merchants[userId];
  }

  // Get merchant collateral status
  static Future<MerchantCollateral?> getCollateral(String merchantId) async {
    return _collaterals[merchantId];
  }

  // Get merchant trading statistics
  static Future<MerchantStats> getMerchantStats(String userId) async {
    // Generate mock stats
    final level = AntiFraudProtection.getTraderLevel(50000 + (DateTime.now().millisecond * 1000));
    
    return MerchantStats(
      totalTrades: 150 + DateTime.now().millisecond,
      totalVolume: 50000 + (DateTime.now().millisecond * 100),
      completedTrades: 145,
      cancelledTrades: 3,
      disputeCount: 2,
      avgRating: 4.8,
      totalReviews: 120,
      avgResponseTime: 2.5,
      avgReleaseTime: 3.2,
      traderLevel: level,
    );
  }

  // Verify collateral is sufficient
  static bool verifyCollateral({
    required String merchantId,
    required double requiredAmount,
  }) {
    final collateral = _collaterals[merchantId];
    if (collateral == null) return false;
    return collateral.usdValue >= requiredAmount;
  }

  // Slash collateral (penalty for fraud)
  static Future<void> slashCollateral({
    required String merchantId,
    required String reason,
    required double amount,
  }) async {
    final collateral = _collaterals[merchantId];
    if (collateral != null) {
      final newAmount = collateral.amount - amount;
      _collaterals[merchantId] = MerchantCollateral(
        id: collateral.id,
        merchantId: collateral.merchantId,
        token: collateral.token,
        amount: newAmount > 0 ? newAmount : 0,
        usdValue: (newAmount > 0 ? newAmount : 0) * 43250,
        status: newAmount > 0 ? 'LOCKED' : 'SLASHED',
        lockedAt: collateral.lockedAt,
        unlockedAt: DateTime.now(),
        reason: reason,
      );
    }
  }

  // Lock security deposit for order
  static Future<P2PSecurityDeposit> lockSecurityDeposit({
    required String userId,
    required String orderId,
    required double tradeAmount,
    required String type, // BUYER_PROTECTION or SELLER_BOND
    required String token,
  }) async {
    final depositAmount = AntiFraudProtection.calculateSecurityDeposit(
      tradeAmount: tradeAmount,
      traderLevel: 'SILVER',
      completionRate: 95,
    );
    
    final deposit = P2PSecurityDeposit(
      id: 'deposit_${DateTime.now().millisecondsSinceEpoch}',
      userId: userId,
      orderId: orderId,
      token: token,
      amount: depositAmount,
      usdValue: depositAmount * 43250,
      type: type,
      status: 'LOCKED',
      lockedAt: DateTime.now(),
    );
    
    _deposits[userId] = [...(_deposits[userId] ?? []), deposit];
    return deposit;
  }

  // Release security deposit after successful trade
  static Future<void> releaseSecurityDeposit(String userId, String orderId) async {
    final deposits = _deposits[userId] ?? [];
    final index = deposits.indexWhere((d) => d.orderId == orderId);
    
    if (index != -1) {
      final deposit = deposits[index];
      deposits[index] = P2PSecurityDeposit(
        id: deposit.id,
        userId: deposit.userId,
        orderId: deposit.orderId,
        token: deposit.token,
        amount: deposit.amount,
        usdValue: deposit.usdValue,
        type: deposit.type,
        status: 'RELEASED',
        lockedAt: deposit.lockedAt,
        releasedAt: DateTime.now(),
      );
      _deposits[userId] = deposits;
    }
  }

  // Forfeit deposit (penalty for fraud)
  static Future<void> forfeitDeposit(String userId, String orderId) async {
    final deposits = _deposits[userId] ?? [];
    final index = deposits.indexWhere((d) => d.orderId == orderId);
    
    if (index != -1) {
      final deposit = deposits[index];
      deposits[index] = P2PSecurityDeposit(
        id: deposit.id,
        userId: deposit.userId,
        orderId: deposit.orderId,
        token: deposit.token,
        amount: deposit.amount,
        usdValue: deposit.usdValue,
        type: deposit.type,
        status: 'FORFEITED',
        lockedAt: deposit.lockedAt,
        releasedAt: DateTime.now(),
      );
      _deposits[userId] = deposits;
    }
  }

  // Get security deposits
  static Future<List<P2PSecurityDeposit>> getDeposits(String userId) async {
    return _deposits[userId] ?? [];
  }
}
