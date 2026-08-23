// Claim Service - Flutter Implementation

class ClaimableReward {
  final String id;
  final String userId;
  final String type; // airdrop, bonus, reward, rebate, cashback
  final String title;
  final String description;
  final double amount;
  final String token;
  final String status; // pending, approved, rejected, claimed
  final String source;
  final DateTime expiresAt;
  final DateTime createTime;
  final DateTime? claimTime;
  final String? txHash;

  ClaimableReward({
    required this.id,
    required this.userId,
    required this.type,
    required this.title,
    required this.description,
    required this.amount,
    required this.token,
    required this.status,
    required this.source,
    required this.expiresAt,
    required this.createTime,
    this.claimTime,
    this.txHash,
  });

  bool get isExpired => DateTime.now().isAfter(expiresAt);
  bool get isClaimable => status == 'approved' && !isExpired;
  bool get isPending => status == 'pending';
}

class ClaimHistory {
  final String id;
  final String userId;
  final String type;
  final String title;
  final double amount;
  final String token;
  final DateTime claimedAt;
  final String txHash;

  ClaimHistory({
    required this.id,
    required this.userId,
    required this.type,
    required this.title,
    required this.amount,
    required this.token,
    required this.claimedAt,
    required this.txHash,
  });
}

class ClaimSettings {
  final bool enabled;
  final bool autoApprove;
  final double minAmount;
  final double maxAmount;

  ClaimSettings({
    required this.enabled,
    required this.autoApprove,
    required this.minAmount,
    required this.maxAmount,
  });

  factory ClaimSettings.defaultSettings() {
    return ClaimSettings(
      enabled: true,
      autoApprove: false,
      minAmount: 1,
      maxAmount: 1000000,
    );
  }
}

class ClaimService {
  static final Map<String, ClaimableReward> _rewards = {};
  static final Map<String, List<ClaimHistory>> _history = {};
  static ClaimSettings _settings = ClaimSettings.defaultSettings();

  static ClaimSettings get settings => _settings;

  static void updateSettings(ClaimSettings newSettings) {
    _settings = newSettings;
  }

  static ClaimableReward createReward({
    required String userId,
    required String type,
    required String title,
    required String description,
    required double amount,
    required String token,
    required String source,
    required int expiryDays,
  }) {
    final now = DateTime.now();
    final id = 'claim-${now.millisecondsSinceEpoch}';
    
    String status = 'pending';
    if (_settings.autoApprove) {
      status = 'approved';
    }

    final reward = ClaimableReward(
      id: id,
      userId: userId,
      type: type,
      title: title,
      description: description,
      amount: amount,
      token: token,
      status: status,
      source: source,
      expiresAt: now.add(Duration(days: expiryDays)),
      createTime: now,
    );

    _rewards[id] = reward;
    return reward;
  }

  static List<ClaimableReward> getAvailableRewards(String userId) {
    return _rewards.values
        .where((r) => r.userId == userId && (r.isClaimable || r.isPending) && !r.isExpired)
        .toList();
  }

  static List<ClaimHistory> getUserHistory(String userId) {
    return _history[userId] ?? [];
  }

  static bool approveReward(String rewardId) {
    final reward = _rewards[rewardId];
    if (reward == null) return false;

    _rewards[rewardId] = ClaimableReward(
      id: reward.id,
      userId: reward.userId,
      type: reward.type,
      title: reward.title,
      description: reward.description,
      amount: reward.amount,
      token: reward.token,
      status: 'approved',
      source: reward.source,
      expiresAt: reward.expiresAt,
      createTime: reward.createTime,
    );
    return true;
  }

  static bool rejectReward(String rewardId) {
    final reward = _rewards[rewardId];
    if (reward == null) return false;

    _rewards[rewardId] = ClaimableReward(
      id: reward.id,
      userId: reward.userId,
      type: reward.type,
      title: reward.title,
      description: reward.description,
      amount: reward.amount,
      token: reward.token,
      status: 'rejected',
      source: reward.source,
      expiresAt: reward.expiresAt,
      createTime: reward.createTime,
    );
    return true;
  }

  static ClaimHistory? claimReward(String rewardId) {
    final reward = _rewards[rewardId];
    if (reward == null || !reward.isClaimable) return null;

    final now = DateTime.now();
    final txHash = '0x${now.millisecondsSinceEpoch.toRadixString(16)}';

    _rewards[rewardId] = ClaimableReward(
      id: reward.id,
      userId: reward.userId,
      type: reward.type,
      title: reward.title,
      description: reward.description,
      amount: reward.amount,
      token: reward.token,
      status: 'claimed',
      source: reward.source,
      expiresAt: reward.expiresAt,
      createTime: reward.createTime,
      claimTime: now,
      txHash: txHash,
    );

    final history = ClaimHistory(
      id: 'history-${now.millisecondsSinceEpoch}',
      userId: reward.userId,
      type: reward.type,
      title: reward.title,
      amount: reward.amount,
      token: reward.token,
      claimedAt: now,
      txHash: txHash,
    );

    _history[reward.userId] = [...(_history[reward.userId] ?? []), history];
    return history;
  }

  static List<ClaimableReward> getAllRewards() {
    return _rewards.values.toList();
  }
}
