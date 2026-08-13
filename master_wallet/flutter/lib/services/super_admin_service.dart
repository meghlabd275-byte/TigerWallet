/**
 * SuperAdminService - Flutter Implementation
 *
 * The canonical Go backend (:8450) does NOT expose dedicated super-admin /
 * white-label / feature-flag / profit-distribution endpoints. Admin
 * authentication therefore delegates to the canonical auth login route
 * (POST /api/v1/auth/login), and every other admin operation fails closed
 * (throws UnimplementedError) rather than maintaining fabricated in-memory
 * admin tables, fake feature flags, or simulated profit distributions.
 *
 * NO hardcoded admins, NO local PBKDF2 auth, NO fake tokens, NO fake tx
 * hashes. The backend is the sole source of truth for identity and state.
 */

import 'dart:convert';
import 'package:http/http.dart' as http;

class SuperAdminService {
  static SuperAdminService? _instance;
  static SuperAdminService get instance {
    _instance ??= SuperAdminService._();
    return _instance!;
  }

  SuperAdminService._();

  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );

  String? _token;
  void setToken(String? token) => _token = token;

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  Exception _unsupported(String op) => UnimplementedError(
        'super-admin $op is not supported by the canonical backend contract. '
        'The backend exposes no super-admin / white-label / feature-flag / '
        'profit-distribution endpoints. Authenticate via authenticate() then '
        'use the canonical master-wallet endpoints for wallet operations.');

  /// Authenticate an admin against the canonical backend auth login route.
  /// Returns the JWT + identity returned by the backend; never fabricates a
  /// token or admin record locally. Throws on invalid credentials or backend
  /// error.
  Future<Map<String, dynamic>> authenticate(
    String email,
    String password, {
    String ipAddress = '0.0.0.0',
  }) async {
    final r = await http.post(
      Uri.parse('$API_BASE/api/v1/auth/login'),
      headers: _headers,
      body: json.encode({'email': email, 'password': password}),
    );
    if (r.statusCode != 200) {
      throw SuperAdminException(
        'authentication failed (${r.statusCode}): ${r.body}',
      );
    }
    final data = json.decode(r.body) as Map<String, dynamic>;
    // Persist the returned JWT for subsequent authenticated calls.
    final token = data['token'] as String?;
    if (token != null) _token = token;
    return {
      'success': true,
      'token': token,
      'admin': {
        'id': data['user_id'],
        'email': data['email'],
        'role': data['role'],
      },
    };
  }

  Future<void> logout(String token) async {
    // The backend manages session validity via JWT expiry; nothing to clear
    // locally. If a token revocation endpoint is added later, call it here.
    _token = null;
  }

  /// Verify a token. The canonical backend is the authority; without a
  /// dedicated super-admin token-verification endpoint we cannot confirm an
  /// admin session, so this fails closed.
  Future<AdminUser?> verifyToken(String token) async {
    throw _unsupported('verifyToken');
  }

  // ==================== Master Admin Management (unsupported) ====================

  Future<Map<String, dynamic>> createMasterAdmin(
    String superAdminToken,
    String email,
    String password,
    String masterWalletId,
  ) async {
    throw _unsupported('createMasterAdmin');
  }

  Future<List<AdminUser>> getMasterAdmins(String superAdminToken) async {
    throw _unsupported('getMasterAdmins');
  }

  // ==================== White Label Admin (unsupported) ====================

  Future<Map<String, dynamic>> createWhiteLabelAdmin(
    String masterAdminToken,
    String email,
    String password,
    Map<String, dynamic> config,
  ) async {
    throw _unsupported('createWhiteLabelAdmin');
  }

  Future<List<WhiteLabelConfig>> getWhiteLabels(String adminToken) async {
    throw _unsupported('getWhiteLabels');
  }

  // ==================== Feature Flags (unsupported) ====================

  Future<List<FeatureConfig>> getFeatureFlags(String adminToken) async {
    throw _unsupported('getFeatureFlags');
  }

  Future<Map<String, dynamic>> updateFeatureFlag(
    String adminToken,
    String featureName,
    bool enabled, {
    Map<String, dynamic>? config,
  }) async {
    throw _unsupported('updateFeatureFlag');
  }

  Future<bool> isFeatureEnabled(String featureName) async {
    throw _unsupported('isFeatureEnabled');
  }

  // ==================== Profit Distribution (unsupported) ====================

  Future<Map<String, dynamic>> executeProfitDistribution(
    String adminToken,
    String whiteLabelId,
    String amount,
    String token,
  ) async {
    throw _unsupported('executeProfitDistribution');
  }

  Future<List<ProfitDistribution>> getProfitDistributions(
    String adminToken, {
    String? whiteLabelId,
  }) async {
    throw _unsupported('getProfitDistributions');
  }

  // ==================== Audit Logs (unsupported) ====================
  // Use AuditService (GET /api/v1/master-wallet/:id/audit) for the canonical
  // audit log; there is no cross-wallet super-admin audit endpoint.

  Future<List<AuditLogEntry>> getAuditLogs(
    String adminToken, {
    String? adminId,
    String? action,
    String? entityType,
    DateTime? startTime,
    DateTime? endTime,
    int? limit,
  }) async {
    throw _unsupported('getAuditLogs');
  }

  // ==================== Dashboard Stats (unsupported) ====================

  Future<Map<String, dynamic>> getDashboardStats(String adminToken) async {
    throw _unsupported('getDashboardStats');
  }
}

class SuperAdminException implements Exception {
  final String message;
  SuperAdminException(this.message);
  @override
  String toString() => 'SuperAdminException: $message';
}

// ==================== DTOs ====================

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
        id: json['id'] ?? '',
        email: json['email'] ?? '',
        role: json['role'] ?? '',
        masterWalletId: json['masterWalletId'],
        permissions: List<String>.from(json['permissions'] ?? const []),
        isActive: json['isActive'] ?? false,
        twoFactorEnabled: json['twoFactorEnabled'] ?? false,
        createdAt: json['createdAt'] != null
            ? DateTime.parse(json['createdAt'])
            : DateTime.now(),
        lastLoginAt: json['lastLoginAt'] != null
            ? DateTime.parse(json['lastLoginAt'])
            : null,
        failedAttempts: json['failedAttempts'] ?? 0,
        lockedUntil: json['lockedUntil'] != null
            ? DateTime.parse(json['lockedUntil'])
            : null,
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

  factory FeatureConfig.fromJson(Map<String, dynamic> json) => FeatureConfig(
        name: json['name'] ?? '',
        enabled: json['enabled'] ?? false,
        description: json['description'] ?? '',
        config: json['config'] as Map<String, dynamic>?,
      );
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

  factory WhiteLabelConfig.fromJson(Map<String, dynamic> json) =>
      WhiteLabelConfig(
        id: json['id'] ?? '',
        name: json['name'] ?? '',
        domain: json['domain'] ?? '',
        branding: Map<String, String>.from(json['branding'] ?? const {}),
        feePercentage: (json['feePercentage'] ?? 0).toDouble(),
        isActive: json['isActive'] ?? false,
      );
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
