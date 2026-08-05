/**
 * SuperAdminService - Flutter Implementation
 * Complete admin control system for Master Wallet
 * Features: Master Admin management, White Label Admin, Feature toggles, Audit logs
 */

import 'dart:convert';
import 'dart:math';
import 'package:crypto/crypto.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

class SuperAdminService {
  static SuperAdminService? _instance;
  static SuperAdminService get instance {
    _instance ??= SuperAdminService._();
    return _instance!;
  }

  SuperAdminService._();

  final FlutterSecureStorage _secureStorage = const FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );

  // Configuration
  static const String _superAdminEmail = 'superadmin@tigerwallet.com';
  static const String _superAdminPassword = 'SuperAdmin@2024!';
  static const String _superAdminWallet = '0x742d35Cc6634C0532925a3b844Bc9e7595f1234';
  static const double _profitSharePercentage = 20.0;

  // Feature flags
  static const List<String> _featureFlags = [
    'master_wallet_creation',
    'multi_blockchain',
    'token_management',
    'user_wallet_ownership',
    'hd_wallet',
    'biometric_auth',
    'pin_code_auth',
    'nft_support',
    'defi_integration',
    'staking',
    'bridge_support',
    'mev_protection',
    'swap_trading',
    'hardware_wallet',
    'admin_controls',
    'network_management',
    'gas_optimization',
    'multi_sig',
    'transaction_history',
    'price_alerts',
    'privacy_zk',
    'coinjoin',
    'account_abstraction',
    'session_keys',
    'paymaster',
    'passkeys',
    'tax_integration',
    'analytics',
    'cross_chain_intent',
    'dapp_browser',
  ];

  // In-memory storage
  final Map<String, AdminUser> _admins = {};
  final Map<String, FeatureConfig> _featureConfigs = {};
  final List<AuditLogEntry> _auditLogs = [];
  final Map<String, WhiteLabelConfig> _whiteLabels = [];
  final List<ProfitDistribution> _profitDistributions = [];
  final Map<String, SessionData> _sessions = {};

  // ==================== Models ====================

  class AdminUser {
    final String id;
    final String email;
    final String role;
    final String? masterWalletId;
    final List<String> permissions;
    final bool isActive;
    final bool twoFactorEnabled;
    final DateTime createdAt;
    DateTime? lastLoginAt;
    int failedAttempts;
    DateTime? lockedUntil;

    AdminUser({
      required this.id,
      required this.email,
      required this.role,
      this.masterWalletId,
      required this.permissions,
      required this.isActive,
      required this.twoFactorEnabled,
      required this.createdAt,
      this.lastLoginAt,
      this.failedAttempts = 0,
      this.lockedUntil,
    });

    Map<String, dynamic> toJson() => {
      'id': id,
      'email': email,
      'role': role,
      'masterWalletId': masterWalletId,
      'permissions': permissions,
      'isActive': isActive,
      'twoFactorEnabled': twoFactorEnabled,
      'createdAt': createdAt.toIso8601String(),
      'lastLoginAt': lastLoginAt?.toIso8601String(),
      'failedAttempts': failedAttempts,
      'lockedUntil': lockedUntil?.toIso8601String(),
    };

    factory AdminUser.fromJson(Map<String, dynamic> json) => AdminUser(
      id: json['id'],
      email: json['email'],
      role: json['role'],
      masterWalletId: json['masterWalletId'],
      permissions: List<String>.from(json['permissions']),
      isActive: json['isActive'],
      twoFactorEnabled: json['twoFactorEnabled'],
      createdAt: DateTime.parse(json['createdAt']),
      lastLoginAt: json['lastLoginAt'] != null ? DateTime.parse(json['lastLoginAt']) : null,
      failedAttempts: json['failedAttempts'] ?? 0,
      lockedUntil: json['lockedUntil'] != null ? DateTime.parse(json['lockedUntil']) : null,
    );
  }

  class FeatureConfig {
    final String name;
    bool enabled;
    final String description;
    Map<String, dynamic>? config;

    FeatureConfig({
      required this.name,
      required this.enabled,
      required this.description,
      this.config,
    });

    Map<String, dynamic> toJson() => {
      'name': name,
      'enabled': enabled,
      'description': description,
      'config': config,
    };
  }

  class WhiteLabelConfig {
    final String id;
    final String name;
    final String domain;
    final Map<String, String> branding;
    final double feePercentage;
    final bool isActive;

    WhiteLabelConfig({
      required this.id,
      required this.name,
      required this.domain,
      required this.branding,
      required this.feePercentage,
      required this.isActive,
    });

    Map<String, dynamic> toJson() => {
      'id': id,
      'name': name,
      'domain': domain,
      'branding': branding,
      'feePercentage': feePercentage,
      'isActive': isActive,
    };
  }

  class AuditLogEntry {
    final String id;
    final String adminId;
    final String action;
    final String entityType;
    final String? entityId;
    final Map<String, dynamic> details;
    final String ipAddress;
    final String userAgent;
    final DateTime timestamp;

    AuditLogEntry({
      required this.id,
      required this.adminId,
      required this.action,
      required this.entityType,
      this.entityId,
      required this.details,
      required this.ipAddress,
      required this.userAgent,
      required this.timestamp,
    });

    Map<String, dynamic> toJson() => {
      'id': id,
      'adminId': adminId,
      'action': action,
      'entityType': entityType,
      'entityId': entityId,
      'details': details,
      'ipAddress': ipAddress,
      'userAgent': userAgent,
      'timestamp': timestamp.toIso8601String(),
    };
  }

  class ProfitDistribution {
    final String whiteLabelId;
    final String amount;
    final String token;
    final DateTime timestamp;
    final String? txHash;

    ProfitDistribution({
      required this.whiteLabelId,
      required this.amount,
      required this.token,
      required this.timestamp,
      this.txHash,
    });

    Map<String, dynamic> toJson() => {
      'whiteLabelId': whiteLabelId,
      'amount': amount,
      'token': token,
      'timestamp': timestamp.toIso8601String(),
      'txHash': txHash,
    };
  }

  class SessionData {
    final String adminId;
    final DateTime expiresAt;

    SessionData({
      required this.adminId,
      required this.expiresAt,
    });
  }

  // ==================== Initialization ====================

  Future<void> initialize() async {
    // Create super admin
    final superAdmin = AdminUser(
      id: _generateId(),
      email: _superAdminEmail,
      role: 'super_admin',
      permissions: ['all'],
      isActive: true,
      twoFactorEnabled: false,
      createdAt: DateTime.now(),
    );
    _admins[superAdmin.id] = superAdmin;

    // Initialize feature flags
    for (final flag in _featureFlags) {
      _featureConfigs[flag] = FeatureConfig(
        name: flag,
        enabled: true,
        description: 'Feature flag for $flag',
      );
    }
  }

  // ==================== Authentication ====================

  Future<Map<String, dynamic>> authenticate(
    String email,
    String password, {
    String ipAddress = '0.0.0.0',
  }) async {
    final admin = _admins.values.firstWhere(
      (a) => a.email == email,
      orElse: () => throw Exception('User not found'),
    );

    // Check lockout
    if (admin.lockedUntil != null && admin.lockedUntil!.isAfter(DateTime.now())) {
      return {'success': false, 'error': 'Account locked. Try again later.'};
    }

    // Verify password
    if (password != _superAdminPassword) {
      admin.failedAttempts++;
      if (admin.failedAttempts >= 5) {
        admin.lockedUntil = DateTime.now().add(const Duration(minutes: 15));
      }
      _logAudit(admin.id, 'LOGIN_FAILED', 'admin', {'reason': 'Invalid password'}, ipAddress, 'Unknown');
      return {'success': false, 'error': 'Invalid credentials'};
    }

    // Reset attempts and update login
    admin.failedAttempts = 0;
    admin.lastLoginAt = DateTime.now();

    // Generate token
    final token = _generateToken();
    _sessions[token] = SessionData(
      adminId: admin.id,
      expiresAt: DateTime.now().add(const Duration(hours: 24)),
    );

    _logAudit(admin.id, 'LOGIN_SUCCESS', 'admin', {}, ipAddress, 'Unknown');

    return {
      'success': true,
      'token': token,
      'admin': {
        'id': admin.id,
        'email': admin.email,
        'role': admin.role,
        'permissions': admin.permissions,
      },
    };
  }

  Future<void> logout(String token) async {
    _sessions.remove(token);
  }

  Future<AdminUser?> verifyToken(String token) async {
    final session = _sessions[token];
    if (session == null || session.expiresAt.isBefore(DateTime.now())) {
      _sessions.remove(token);
      return null;
    }
    return _admins[session.adminId];
  }

  // ==================== Master Admin Management ====================

  Future<Map<String, dynamic>> createMasterAdmin(
    String superAdminToken,
    String email,
    String password,
    String masterWalletId,
  ) async {
    final superAdmin = await verifyToken(superAdminToken);
    if (superAdmin == null || superAdmin.role != 'super_admin') {
      return {'success': false, 'error': 'Unauthorized'};
    }

    // Check if email exists
    if (_admins.values.any((a) => a.email == email)) {
      return {'success': false, 'error': 'Email already exists'};
    }

    final masterAdmin = AdminUser(
      id: _generateId(),
      email: email,
      role: 'master_admin',
      masterWalletId: masterWalletId,
      permissions: ['admin', 'manage_users', 'view_analytics', 'manage_fees'],
      isActive: true,
      twoFactorEnabled: false,
      createdAt: DateTime.now(),
    );

    _admins[masterAdmin.id] = masterAdmin;
    _logAudit(superAdmin.id, 'MASTER_ADMIN_CREATED', 'admin', {
      'newAdminId': masterAdmin.id,
      'email': email,
      'masterWalletId': masterWalletId,
    }, '0.0.0.0', 'Unknown');

    return {'success': true, 'adminId': masterAdmin.id};
  }

  Future<List<AdminUser>> getMasterAdmins(String superAdminToken) async {
    final superAdmin = await verifyToken(superAdminToken);
    if (superAdmin == null || superAdmin.role != 'super_admin') {
      return [];
    }
    return _admins.values.where((a) => a.role == 'master_admin').toList();
  }

  // ==================== White Label Admin ====================

  Future<Map<String, dynamic>> createWhiteLabelAdmin(
    String masterAdminToken,
    String email,
    String password,
    Map<String, dynamic> config,
  ) async {
    final masterAdmin = await verifyToken(masterAdminToken);
    if (masterAdmin == null ||
        (masterAdmin.role != 'super_admin' && masterAdmin.role != 'master_admin')) {
      return {'success': false, 'error': 'Unauthorized'};
    }

    // Check if email exists
    if (_admins.values.any((a) => a.email == email)) {
      return {'success': false, 'error': 'Email already exists'};
    }

    final whiteLabelId = _generateId();
    final adminId = _generateId();

    final whiteLabel = WhiteLabelConfig(
      id: whiteLabelId,
      name: config['name'],
      domain: config['domain'],
      branding: Map<String, String>.from(config['branding'] ?? {}),
      feePercentage: (config['feePercentage'] ?? 20).toDouble(),
      isActive: true,
    );

    final whiteLabelAdmin = AdminUser(
      id: adminId,
      email: email,
      role: 'white_label_admin',
      masterWalletId: whiteLabelId,
      permissions: ['view', 'manage_own_users', 'view_own_analytics'],
      isActive: true,
      twoFactorEnabled: false,
      createdAt: DateTime.now(),
    );

    _admins[adminId] = whiteLabelAdmin;
    _whiteLabels.add(whiteLabel);

    _logAudit(masterAdmin.id, 'WHITE_LABEL_CREATED', 'admin', {
      'adminId': adminId,
      'whiteLabelId': whiteLabelId,
      'name': config['name'],
    }, '0.0.0.0', 'Unknown');

    return {'success': true, 'adminId': adminId, 'whiteLabelId': whiteLabelId};
  }

  Future<List<WhiteLabelConfig>> getWhiteLabels(String adminToken) async {
    final admin = await verifyToken(adminToken);
    if (admin == null) return [];

    if (admin.role == 'super_admin' || admin.role == 'master_admin') {
      return _whiteLabels;
    }

    if (admin.masterWalletId != null) {
      return _whiteLabels.where((w) => w.id == admin.masterWalletId).toList();
    }

    return [];
  }

  // ==================== Feature Flags ====================

  Future<List<FeatureConfig>> getFeatureFlags(String adminToken) async {
    final admin = await verifyToken(adminToken);
    if (admin == null) return [];
    return _featureConfigs.values.toList();
  }

  Future<Map<String, dynamic>> updateFeatureFlag(
    String adminToken,
    String featureName,
    bool enabled, {
    Map<String, dynamic>? config,
  }) async {
    final admin = await verifyToken(adminToken);
    if (admin == null || admin.role != 'super_admin') {
      return {'success': false, 'error': 'Only Super Admin can modify feature flags'};
    }

    final feature = _featureConfigs[featureName];
    if (feature == null) {
      return {'success': false, 'error': 'Feature not found'};
    }

    feature.enabled = enabled;
    if (config != null) feature.config = config;

    _logAudit(admin.id, 'FEATURE_FLAG_UPDATED', 'admin', {
      'featureName': featureName,
      'enabled': enabled,
      'config': config,
    }, '0.0.0.0', 'Unknown');

    return {'success': true};
  }

  Future<bool> isFeatureEnabled(String featureName) async {
    return _featureConfigs[featureName]?.enabled ?? false;
  }

  // ==================== Profit Distribution ====================

  Future<Map<String, dynamic>> executeProfitDistribution(
    String adminToken,
    String whiteLabelId,
    String amount,
    String token,
  ) async {
    final admin = await verifyToken(adminToken);
    if (admin == null) {
      return {'success': false, 'error': 'Unauthorized'};
    }

    final whiteLabel = _whiteLabels.firstWhere(
      (w) => w.id == whiteLabelId,
      orElse: () => throw Exception('White Label not found'),
    );

    // Calculate profit share
    final profitAmount = (double.parse(amount) * _profitSharePercentage / 100).toString();

    final distribution = ProfitDistribution(
      whiteLabelId: whiteLabelId,
      amount: profitAmount,
      token: token,
      timestamp: DateTime.now(),
      txHash: '0x${_generateRandomHex(32)}',
    );

    _profitDistributions.add(distribution);
    _logAudit(admin.id, 'PROFIT_DISTRIBUTION', 'admin', {
      'whiteLabelId': whiteLabelId,
      'amount': profitAmount,
      'token': token,
    }, '0.0.0.0', 'Unknown');

    return {'success': true, 'txHash': distribution.txHash};
  }

  Future<List<ProfitDistribution>> getProfitDistributions(
    String adminToken, {
    String? whiteLabelId,
  }) async {
    final admin = await verifyToken(adminToken);
    if (admin == null) return [];

    if (whiteLabelId != null) {
      return _profitDistributions.where((d) => d.whiteLabelId == whiteLabelId).toList();
    }

    return _profitDistributions;
  }

  // ==================== Audit Logs ====================

  Future<List<AuditLogEntry>> getAuditLogs(
    String adminToken, {
    String? adminId,
    String? action,
    String? entityType,
    DateTime? startTime,
    DateTime? endTime,
    int? limit,
  }) async {
    final admin = await verifyToken(adminToken);
    if (admin == null) return [];

    var logs = List<AuditLogEntry>.from(_auditLogs);

    if (adminId != null) {
      logs = logs.where((l) => l.adminId == adminId).toList();
    }
    if (action != null) {
      logs = logs.where((l) => l.action.contains(action)).toList();
    }
    if (entityType != null) {
      logs = logs.where((l) => l.entityType == entityType).toList();
    }
    if (startTime != null) {
      logs = logs.where((l) => l.timestamp.isAfter(startTime)).toList();
    }
    if (endTime != null) {
      logs = logs.where((l) => l.timestamp.isBefore(endTime)).toList();
    }

    logs.sort((a, b) => b.timestamp.compareTo(a.timestamp));

    if (limit != null && logs.length > limit) {
      logs = logs.sublist(0, limit);
    }

    return logs;
  }

  // ==================== Dashboard Stats ====================

  Future<Map<String, dynamic>> getDashboardStats(String adminToken) async {
    final admin = await verifyToken(adminToken);
    if (admin == null) {
      return {
        'totalAdmins': 0,
        'totalWhiteLabels': 0,
        'totalProfitDistributed': '0',
        'featureFlagsEnabled': 0,
        'recentAuditLogs': 0,
      };
    }

    final totalProfit = _profitDistributions.fold<double>(
      0,
      (sum, d) => sum + double.parse(d.amount),
    );
    final featuresEnabled = _featureConfigs.values.where((f) => f.enabled).length;
    final recentLogs = _auditLogs
        .where((l) => l.timestamp.isAfter(
            DateTime.now().subtract(const Duration(days: 1))))
        .length;

    return {
      'totalAdmins': _admins.length,
      'totalWhiteLabels': _whiteLabels.length,
      'totalProfitDistributed': totalProfit.toString(),
      'featureFlagsEnabled': featuresEnabled,
      'recentAuditLogs': recentLogs,
    };
  }

  // ==================== Private Helpers ====================

  void _logAudit(
    String adminId,
    String action,
    String entityType,
    Map<String, dynamic> details,
    String ipAddress,
    String userAgent,
  ) {
    final log = AuditLogEntry(
      id: _generateId(),
      adminId: adminId,
      action: action,
      entityType: entityType,
      details: details,
      ipAddress: ipAddress,
      userAgent: userAgent,
      timestamp: DateTime.now(),
    );
    _auditLogs.add(log);

    // Keep only last 10000 logs
    if (_auditLogs.length > 10000) {
      _auditLogs.removeRange(0, _auditLogs.length - 10000);
    }
  }

  String _generateId() {
    return 'id_${DateTime.now().millisecondsSinceEpoch}_${_generateRandomHex(8)}';
  }

  String _generateToken() {
    return 'tok_${_generateRandomHex(32)}';
  }

  String _generateRandomHex(int length) {
    final random = Random.secure();
    return List.generate(length, (_) => random.nextInt(16).toRadixString(16)).join();
  }
}
