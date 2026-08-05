// Admin Models - Flutter
// Complete data models for TigerWallet Admin Platform

class AdminUser {
  final int id;
  final String username;
  final String email;
  final String? firstName;
  final String? lastName;
  final AdminRole role;
  final List<String> permissions;
  final AdminStatus status;
  final bool twoFactorEnabled;
  final String? lastLoginAt;
  final String createdAt;
  final String updatedAt;
  final String? avatarUrl;
  final String? phone;
  final String? department;

  AdminUser({
    required this.id,
    required this.username,
    required this.email,
    this.firstName,
    this.lastName,
    required this.role,
    required this.permissions,
    required this.status,
    this.twoFactorEnabled = false,
    this.lastLoginAt,
    required this.createdAt,
    required this.updatedAt,
    this.avatarUrl,
    this.phone,
    this.department,
  });

  String get fullName => '${firstName ?? ''} ${lastName ?? ''}'.trim();

  factory AdminUser.fromJson(Map<String, dynamic> json) {
    return AdminUser(
      id: json['id'] as int,
      username: json['username'] as String,
      email: json['email'] as String,
      firstName: json['first_name'] as String?,
      lastName: json['last_name'] as String?,
      role: AdminRole.values.firstWhere(
        (e) => e.name == json['role'],
        orElse: () => AdminRole.admin,
      ),
      permissions: List<String>.from(json['permissions'] ?? []),
      status: AdminStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => AdminStatus.active,
      ),
      twoFactorEnabled: json['two_factor_enabled'] as bool? ?? false,
      lastLoginAt: json['last_login_at'] as String?,
      createdAt: json['created_at'] as String,
      updatedAt: json['updated_at'] as String,
      avatarUrl: json['avatar_url'] as String?,
      phone: json['phone'] as String?,
      department: json['department'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'username': username,
      'email': email,
      'first_name': firstName,
      'last_name': lastName,
      'role': role.name,
      'permissions': permissions,
      'status': status.name,
      'two_factor_enabled': twoFactorEnabled,
      'last_login_at': lastLoginAt,
      'created_at': createdAt,
      'updated_at': updatedAt,
      'avatar_url': avatarUrl,
      'phone': phone,
      'department': department,
    };
  }
}

enum AdminRole { superAdmin, admin, support, analyst, moderator }

enum AdminStatus { active, suspended, inactive }

class PlatformUser {
  final int id;
  final String email;
  final String? username;
  final String? walletAddress;
  final UserStatus status;
  final KYCStatus kycStatus;
  final int kycLevel;
  final int riskScore;
  final String createdAt;
  final String? lastLoginAt;
  final String? registrationIp;
  final List<String> tags;
  final String? referredBy;
  final int? whiteLabelId;

  PlatformUser({
    required this.id,
    required this.email,
    this.username,
    this.walletAddress,
    required this.status,
    required this.kycStatus,
    required this.kycLevel,
    required this.riskScore,
    required this.createdAt,
    this.lastLoginAt,
    this.registrationIp,
    required this.tags,
    this.referredBy,
    this.whiteLabelId,
  });

  factory PlatformUser.fromJson(Map<String, dynamic> json) {
    return PlatformUser(
      id: json['id'] as int,
      email: json['email'] as String,
      username: json['username'] as String?,
      walletAddress: json['wallet_address'] as String?,
      status: UserStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => UserStatus.active,
      ),
      kycStatus: KYCStatus.values.firstWhere(
        (e) => e.name == json['kyc_status'],
        orElse: () => KYCStatus.none,
      ),
      kycLevel: json['kyc_level'] as int? ?? 0,
      riskScore: json['risk_score'] as int? ?? 0,
      createdAt: json['created_at'] as String,
      lastLoginAt: json['last_login_at'] as String?,
      registrationIp: json['registration_ip'] as String?,
      tags: List<String>.from(json['tags'] ?? []),
      referredBy: json['referred_by'] as String?,
      whiteLabelId: json['white_label_id'] as int?,
    );
  }
}

enum UserStatus { active, pending, suspended, banned }

enum KYCStatus { none, pending, level1, level2, level3, rejected }

class Transaction {
  final int id;
  final String hash;
  final TransactionType type;
  final String chain;
  final String fromAddress;
  final String toAddress;
  final String amount;
  final String token;
  final String? tokenAmount;
  final TransactionStatus status;
  final int? blockNumber;
  final String? gasUsed;
  final String? gasPrice;
  final String timestamp;
  final bool flagged;
  final String? flagReason;
  final int userId;

  Transaction({
    required this.id,
    required this.hash,
    required this.type,
    required this.chain,
    required this.fromAddress,
    required this.toAddress,
    required this.amount,
    required this.token,
    this.tokenAmount,
    required this.status,
    this.blockNumber,
    this.gasUsed,
    this.gasPrice,
    required this.timestamp,
    required this.flagged,
    this.flagReason,
    required this.userId,
  });

  String get shortHash =>
      '${hash.substring(0, 10)}...${hash.substring(hash.length - 8)}';

  factory Transaction.fromJson(Map<String, dynamic> json) {
    return Transaction(
      id: json['id'] as int,
      hash: json['hash'] as String,
      type: TransactionType.values.firstWhere(
        (e) => e.name == json['type'],
        orElse: () => TransactionType.transfer,
      ),
      chain: json['chain'] as String,
      fromAddress: json['from_address'] as String,
      toAddress: json['to_address'] as String,
      amount: json['amount'] as String,
      token: json['token'] as String,
      tokenAmount: json['token_amount'] as String?,
      status: TransactionStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => TransactionStatus.pending,
      ),
      blockNumber: json['block_number'] as int?,
      gasUsed: json['gas_used'] as String?,
      gasPrice: json['gas_price'] as String?,
      timestamp: json['timestamp'] as String,
      flagged: json['flagged'] as bool? ?? false,
      flagReason: json['flag_reason'] as String?,
      userId: json['user_id'] as int,
    );
  }
}

enum TransactionType { transfer, swap, stake, unstake, bridge, withdraw, deposit, mint, burn }

enum TransactionStatus { pending, confirmed, failed }

class KYCApplication {
  final int id;
  final int userId;
  final String userEmail;
  final int level;
  final KYCApplicationStatus status;
  final String submittedAt;
  final String? reviewedAt;
  final String? reviewedBy;
  final String? rejectionReason;
  final List<KYCDocument> documents;
  final String? ipAddress;
  final String? notes;

  KYCApplication({
    required this.id,
    required this.userId,
    required this.userEmail,
    required this.level,
    required this.status,
    required this.submittedAt,
    this.reviewedAt,
    this.reviewedBy,
    this.rejectionReason,
    required this.documents,
    this.ipAddress,
    this.notes,
  });

  factory KYCApplication.fromJson(Map<String, dynamic> json) {
    return KYCApplication(
      id: json['id'] as int,
      userId: json['user_id'] as int,
      userEmail: json['user_email'] as String,
      level: json['level'] as int,
      status: KYCApplicationStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => KYCApplicationStatus.pending,
      ),
      submittedAt: json['submitted_at'] as String,
      reviewedAt: json['reviewed_at'] as String?,
      reviewedBy: json['reviewed_by'] as String?,
      rejectionReason: json['rejection_reason'] as String?,
      documents: (json['documents'] as List<dynamic>?)
              ?.map((e) => KYCDocument.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      ipAddress: json['ip_address'] as String?,
      notes: json['notes'] as String?,
    );
  }
}

enum KYCApplicationStatus { pending, approved, rejected }

class KYCDocument {
  final String type;
  final String url;
  final String status;
  final String? verifiedAt;

  KYCDocument({
    required this.type,
    required this.url,
    required this.status,
    this.verifiedAt,
  });

  factory KYCDocument.fromJson(Map<String, dynamic> json) {
    return KYCDocument(
      type: json['type'] as String,
      url: json['url'] as String,
      status: json['status'] as String,
      verifiedAt: json['verified_at'] as String?,
    );
  }
}

class Token {
  final int id;
  final String name;
  final String symbol;
  final String contractAddress;
  final String chain;
  final int decimals;
  final String totalSupply;
  final String? logoUrl;
  final String? website;
  final String? description;
  final String? price;
  final String? marketCap;
  final String? volume24h;
  final String? priceChange24h;
  final bool isActive;
  final bool isVerified;
  final String? listingFee;
  final String? listedAt;

  Token({
    required this.id,
    required this.name,
    required this.symbol,
    required this.contractAddress,
    required this.chain,
    required this.decimals,
    required this.totalSupply,
    this.logoUrl,
    this.website,
    this.description,
    this.price,
    this.marketCap,
    this.volume24h,
    this.priceChange24h,
    required this.isActive,
    required this.isVerified,
    this.listingFee,
    this.listedAt,
  });

  factory Token.fromJson(Map<String, dynamic> json) {
    return Token(
      id: json['id'] as int,
      name: json['name'] as String,
      symbol: json['symbol'] as String,
      contractAddress: json['contract_address'] as String,
      chain: json['chain'] as String,
      decimals: json['decimals'] as int? ?? 18,
      totalSupply: json['total_supply'] as String,
      logoUrl: json['logo_url'] as String?,
      website: json['website'] as String?,
      description: json['description'] as String?,
      price: json['price'] as String?,
      marketCap: json['market_cap'] as String?,
      volume24h: json['volume_24h'] as String?,
      priceChange24h: json['price_change_24h'] as String?,
      isActive: json['is_active'] as bool? ?? true,
      isVerified: json['is_verified'] as bool? ?? false,
      listingFee: json['listing_fee'] as String?,
      listedAt: json['listed_at'] as String?,
    );
  }
}

class WithdrawalRequest {
  final int id;
  final int userId;
  final String userEmail;
  final String amount;
  final String token;
  final String chain;
  final String toAddress;
  final WithdrawalStatus status;
  final String? approvedAt;
  final String? approvedBy;
  final String? rejectedAt;
  final String? rejectionReason;
  final String? processedAt;
  final String? txHash;
  final String? fee;
  final String createdAt;

  WithdrawalRequest({
    required this.id,
    required this.userId,
    required this.userEmail,
    required this.amount,
    required this.token,
    required this.chain,
    required this.toAddress,
    required this.status,
    this.approvedAt,
    this.approvedBy,
    this.rejectedAt,
    this.rejectionReason,
    this.processedAt,
    this.txHash,
    this.fee,
    required this.createdAt,
  });

  factory WithdrawalRequest.fromJson(Map<String, dynamic> json) {
    return WithdrawalRequest(
      id: json['id'] as int,
      userId: json['user_id'] as int,
      userEmail: json['user_email'] as String,
      amount: json['amount'] as String,
      token: json['token'] as String,
      chain: json['chain'] as String,
      toAddress: json['to_address'] as String,
      status: WithdrawalStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => WithdrawalStatus.pending,
      ),
      approvedAt: json['approved_at'] as String?,
      approvedBy: json['approved_by'] as String?,
      rejectedAt: json['rejected_at'] as String?,
      rejectionReason: json['rejection_reason'] as String?,
      processedAt: json['processed_at'] as String?,
      txHash: json['tx_hash'] as String?,
      fee: json['fee'] as String?,
      createdAt: json['created_at'] as String,
    );
  }
}

enum WithdrawalStatus { pending, approved, rejected, processing, completed, failed }

class WhiteLabel {
  final int id;
  final String name;
  final String slug;
  final String? domain;
  final String? logoUrl;
  final String? faviconUrl;
  final String primaryColor;
  final String? secondaryColor;
  final WhiteLabelStatus status;
  final String? contactEmail;
  final String? contactPhone;
  final String? address;
  final String? description;
  final List<String> features;
  final FeeStructure? feeStructure;
  final String createdAt;
  final String? expiresAt;

  WhiteLabel({
    required this.id,
    required this.name,
    required this.slug,
    this.domain,
    this.logoUrl,
    this.faviconUrl,
    required this.primaryColor,
    this.secondaryColor,
    required this.status,
    this.contactEmail,
    this.contactPhone,
    this.address,
    this.description,
    required this.features,
    this.feeStructure,
    required this.createdAt,
    this.expiresAt,
  });

  factory WhiteLabel.fromJson(Map<String, dynamic> json) {
    return WhiteLabel(
      id: json['id'] as int,
      name: json['name'] as String,
      slug: json['slug'] as String,
      domain: json['domain'] as String?,
      logoUrl: json['logo_url'] as String?,
      faviconUrl: json['favicon_url'] as String?,
      primaryColor: json['primary_color'] as String,
      secondaryColor: json['secondary_color'] as String?,
      status: WhiteLabelStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => WhiteLabelStatus.active,
      ),
      contactEmail: json['contact_email'] as String?,
      contactPhone: json['contact_phone'] as String?,
      address: json['address'] as String?,
      description: json['description'] as String?,
      features: List<String>.from(json['features'] ?? []),
      feeStructure: json['fee_structure'] != null
          ? FeeStructure.fromJson(json['fee_structure'] as Map<String, dynamic>)
          : null,
      createdAt: json['created_at'] as String,
      expiresAt: json['expires_at'] as String?,
    );
  }
}

enum WhiteLabelStatus { active, suspended, pending }

class FeeStructure {
  final String tradingFee;
  final String withdrawalFee;
  final String depositFee;
  final String listingFee;

  FeeStructure({
    required this.tradingFee,
    required this.withdrawalFee,
    required this.depositFee,
    required this.listingFee,
  });

  factory FeeStructure.fromJson(Map<String, dynamic> json) {
    return FeeStructure(
      tradingFee: json['trading_fee'] as String,
      withdrawalFee: json['withdrawal_fee'] as String,
      depositFee: json['deposit_fee'] as String,
      listingFee: json['listing_fee'] as String,
    );
  }
}

class SystemStatus {
  final String serviceName;
  final String status;
  final String uptime;
  final String latency;
  final String lastCheck;

  SystemStatus({
    required this.serviceName,
    required this.status,
    required this.uptime,
    required this.latency,
    required this.lastCheck,
  });

  bool get isHealthy => status == 'running' || status == 'healthy';

  factory SystemStatus.fromJson(Map<String, dynamic> json) {
    return SystemStatus(
      serviceName: json['service_name'] as String,
      status: json['status'] as String,
      uptime: json['uptime'] as String,
      latency: json['latency'] as String,
      lastCheck: json['last_check'] as String,
    );
  }
}

class AnalyticsData {
  final int totalUsers;
  final int activeUsers;
  final String totalVolume;
  final int dailyTransactions;
  final String totalFees;
  final int pendingKyc;
  final String systemHealth;
  final String timestamp;

  AnalyticsData({
    required this.totalUsers,
    required this.activeUsers,
    required this.totalVolume,
    required this.dailyTransactions,
    required this.totalFees,
    required this.pendingKyc,
    required this.systemHealth,
    required this.timestamp,
  });

  factory AnalyticsData.fromJson(Map<String, dynamic> json) {
    return AnalyticsData(
      totalUsers: json['total_users'] as int,
      activeUsers: json['active_users'] as int,
      totalVolume: json['total_volume'] as String,
      dailyTransactions: json['daily_transactions'] as int,
      totalFees: json['total_fees'] as String,
      pendingKyc: json['pending_kyc'] as int,
      systemHealth: json['system_health'] as String,
      timestamp: json['timestamp'] as String,
    );
  }
}

class BotInstance {
  final int id;
  final int userId;
  final String userEmail;
  final String botType;
  final String name;
  final BotStatus status;
  final int connectedDexs;
  final int connectedCexs;
  final String totalPnl;
  final String totalVolume;
  final int totalOrders;
  final int avgLatencyUs;
  final String createdAt;
  final String? lastTradeAt;

  BotInstance({
    required this.id,
    required this.userId,
    required this.userEmail,
    required this.botType,
    required this.name,
    required this.status,
    required this.connectedDexs,
    required this.connectedCexs,
    required this.totalPnl,
    required this.totalVolume,
    required this.totalOrders,
    required this.avgLatencyUs,
    required this.createdAt,
    this.lastTradeAt,
  });

  factory BotInstance.fromJson(Map<String, dynamic> json) {
    return BotInstance(
      id: json['id'] as int,
      userId: json['user_id'] as int,
      userEmail: json['user_email'] as String,
      botType: json['bot_type'] as String,
      name: json['name'] as String,
      status: BotStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => BotStatus.stopped,
      ),
      connectedDexs: json['connected_dexs'] as int,
      connectedCexs: json['connected_cexs'] as int,
      totalPnl: json['total_pnl'] as String,
      totalVolume: json['total_volume'] as String,
      totalOrders: json['total_orders'] as int,
      avgLatencyUs: json['avg_latency_us'] as int,
      createdAt: json['created_at'] as String,
      lastTradeAt: json['last_trade_at'] as String?,
    );
  }
}

enum BotStatus { running, stopped, error, paused }

class FeeConfig {
  final int id;
  final String feeType;
  final int? chainId;
  final String? tokenSymbol;
  final String feeAmountUsd;
  final String feePercentage;
  final String minFeeUsd;
  final String? maxFeeUsd;
  final bool isActive;

  FeeConfig({
    required this.id,
    required this.feeType,
    this.chainId,
    this.tokenSymbol,
    required this.feeAmountUsd,
    required this.feePercentage,
    required this.minFeeUsd,
    this.maxFeeUsd,
    required this.isActive,
  });

  factory FeeConfig.fromJson(Map<String, dynamic> json) {
    return FeeConfig(
      id: json['id'] as int,
      feeType: json['fee_type'] as String,
      chainId: json['chain_id'] as int?,
      tokenSymbol: json['token_symbol'] as String?,
      feeAmountUsd: json['fee_amount_usd'] as String,
      feePercentage: json['fee_percentage'] as String,
      minFeeUsd: json['min_fee_usd'] as String,
      maxFeeUsd: json['max_fee_usd'] as String?,
      isActive: json['is_active'] as bool? ?? true,
    );
  }
}

class Blockchain {
  final String id;
  final String name;
  final String symbol;
  final int chainId;
  final String? chainIdHex;
  final bool isEvm;
  final bool isActive;
  final String? explorerUrl;
  final String? rpcUrl;
  final String nativeTokenSymbol;
  final double avgGasPriceGwei;

  Blockchain({
    required this.id,
    required this.name,
    required this.symbol,
    required this.chainId,
    this.chainIdHex,
    required this.isEvm,
    required this.isActive,
    this.explorerUrl,
    this.rpcUrl,
    required this.nativeTokenSymbol,
    required this.avgGasPriceGwei,
  });

  factory Blockchain.fromJson(Map<String, dynamic> json) {
    return Blockchain(
      id: json['id'] as String,
      name: json['name'] as String,
      symbol: json['symbol'] as String,
      chainId: json['chain_id'] as int,
      chainIdHex: json['chain_id_hex'] as String?,
      isEvm: json['is_evm'] as bool? ?? false,
      isActive: json['is_active'] as bool? ?? true,
      explorerUrl: json['explorer_url'] as String?,
      rpcUrl: json['rpc_url'] as String?,
      nativeTokenSymbol: json['native_token_symbol'] as String,
      avgGasPriceGwei: (json['avg_gas_price_gwei'] as num?)?.toDouble() ?? 0,
    );
  }
}

class Pagination {
  final int page;
  final int limit;
  final int total;
  final int totalPages;

  Pagination({
    required this.page,
    required this.limit,
    required this.total,
    required this.totalPages,
  });

  bool get hasNextPage => page < totalPages;
  bool get hasPreviousPage => page > 1;

  factory Pagination.fromJson(Map<String, dynamic> json) {
    return Pagination(
      page: json['page'] as int,
      limit: json['limit'] as int,
      total: json['total'] as int,
      totalPages: json['total_pages'] as int,
    );
  }
}

class ApiResponse<T> {
  final List<T> data;
  final Pagination pagination;

  ApiResponse({
    required this.data,
    required this.pagination,
  });
}
