///
/// Super Admin Service - Flutter Implementation
/// Identical across ALL platforms
///

import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';
import 'package:crypto/crypto.dart' as crypto;

// Enums
enum UserRole { superAdmin, masterAdmin, whiteLabelAdmin, user }
enum AdminStatus { active, inactive, pending, suspended }
enum AuthorizationStatus { authorized, pending, revoked, rejected }

// Data Models
class SuperAdmin {
  final String id;
  final String email;
  final String passwordHash;
  final String secretKey;
  bool twoFactorEnabled;
  String twoFactorSecret;
  String phone;
  final int createdAt;
  int lastLogin;
  final bool isActive;
  final List<String> permissions;

  SuperAdmin({
    required this.id,
    required this.email,
    required this.passwordHash,
    required this.secretKey,
    this.twoFactorEnabled = false,
    this.twoFactorSecret = '',
    this.phone = '',
    required this.createdAt,
    this.lastLogin = 0,
    this.isActive = true,
    this.permissions = const ['*'],
  });
}

class MasterAdmin {
  final String id;
  final String email;
  String passwordHash;
  String authorizedBy;
  AuthorizationStatus authorizationStatus;
  bool twoFactorEnabled;
  String twoFactorSecret;
  String phone;
  bool canCreateWhiteLabel;
  bool canManageUsers;
  bool canManageWallets;
  bool canAccessFinance;
  bool canModifyFeatures;
  bool canManageTokens;
  bool canManageNetworks;
  bool canViewAnalytics;
  bool canManageAdmins;
  int maxWhiteLabels;
  int whiteLabelCount;
  AdminStatus status;
  final int createdAt;
  int lastLogin;
  int passwordChangedAt;
  int failedAttempts;
  int lockedUntil;

  MasterAdmin({
    required this.id,
    required this.email,
    required this.passwordHash,
    this.authorizedBy = '',
    this.authorizationStatus = AuthorizationStatus.pending,
    this.twoFactorEnabled = false,
    this.twoFactorSecret = '',
    this.phone = '',
    this.canCreateWhiteLabel = false,
    this.canManageUsers = false,
    this.canManageWallets = false,
    this.canAccessFinance = false,
    this.canModifyFeatures = false,
    this.canManageTokens = false,
    this.canManageNetworks = false,
    this.canViewAnalytics = false,
    this.canManageAdmins = false,
    this.maxWhiteLabels = 0,
    this.whiteLabelCount = 0,
    this.status = AdminStatus.pending,
    required this.createdAt,
    this.lastLogin = 0,
    this.passwordChangedAt = 0,
    this.failedAttempts = 0,
    this.lockedUntil = 0,
  });
}

class WhiteLabelAdmin {
  final String id;
  final String email;
  String passwordHash;
  final String masterAdminId;
  String brandName;
  String brandLogo;
  String brandColor;
  String customDomain;
  AuthorizationStatus authorizationStatus;
  bool twoFactorEnabled;
  String twoFactorSecret;
  bool canCustomizeUi;
  bool canCustomizeFees;
  bool canManageUsers;
  bool canManageWallets;
  bool canAccessAnalytics;
  bool canManageTokens;
  double feePercentage;
  AdminStatus status;
  final int createdAt;
  int lastLogin;

  WhiteLabelAdmin({
    required this.id,
    required this.email,
    required this.passwordHash,
    required this.masterAdminId,
    this.brandName = '',
    this.brandLogo = '',
    this.brandColor = '#000000',
    this.customDomain = '',
    this.authorizationStatus = AuthorizationStatus.pending,
    this.twoFactorEnabled = false,
    this.twoFactorSecret = '',
    this.canCustomizeUi = true,
    this.canCustomizeFees = true,
    this.canManageUsers = true,
    this.canManageWallets = true,
    this.canAccessAnalytics = true,
    this.canManageTokens = true,
    this.feePercentage = 0.0,
    this.status = AdminStatus.pending,
    required this.createdAt,
    this.lastLogin = 0,
  });
}

class FeatureControl {
  String featureName;
  bool enabled;
  bool globalEnabled;
  String masterAdminId;
  String whiteLabelId;
  String updatedBy;
  int updatedAt;

  FeatureControl({
    required this.featureName,
    this.enabled = true,
    this.globalEnabled = true,
    this.masterAdminId = '',
    this.whiteLabelId = '',
    this.updatedBy = '',
    required this.updatedAt,
  });
}

class AuditLog {
  final String id;
  final String adminId;
  final UserRole adminRole;
  final String action;
  final String details;
  final String ipAddress;
  final String userAgent;
  final int timestamp;

  AuditLog({
    required this.id,
    required this.adminId,
    required this.adminRole,
    required this.action,
    required this.details,
    this.ipAddress = '',
    this.userAgent = '',
    required this.timestamp,
  });
}

// Super Admin Service
class SuperAdminService {
  static final SuperAdminService _instance = SuperAdminService._internal();
  factory SuperAdminService() => _instance;
  SuperAdminService._internal();

  final Map<String, dynamic> _superAdmins = {};
  final Map<String, dynamic> _masterAdmins = {};
  final Map<String, dynamic> _whiteLabelAdmins = {};
  final Map<String, FeatureControl> _featureControls = {};
  final List<AuditLog> _auditLogs = [];

  SuperAdminService() {
    _createDefaultSuperAdmin();
    _initializeFeatureControls();
  }

  void _createDefaultSuperAdmin() {
    _superAdmins['super_admin_001'] = SuperAdmin(
      id: 'super_admin_001',
      email: 'superadmin@tigerwallet.com',
      passwordHash: _hashPassword('SuperAdmin@2024!'),
      secretKey: _generateSecretKey(),
      createdAt: DateTime.now().millisecondsSinceEpoch,
    );
    _superAdmins['superadmin@tigerwallet.com'] = _superAdmins['super_admin_001'];
  }

  void _initializeFeatureControls() {
    final features = [
      'master_wallet_creation', 'multi_blockchain', 'token_management',
      'user_wallet_ownership', 'hd_wallet', 'biometric_auth',
      'pin_code_auth', 'nft_support', 'defi_integration', 'staking',
      'bridge_support', 'mev_protection', 'swap_trading', 'hardware_wallet',
      'admin_controls', 'network_management', 'gas_optimization', 'multi_sig',
      'transaction_history', 'price_alerts', 'privacy_zk', 'coinjoin',
      'account_abstraction', 'session_keys', 'paymaster', 'passkeys',
      'tax_integration', 'analytics', 'cross_chain_intent', 'dapp_browser',
    ];
    for (var feature in features) {
      _featureControls[feature] = FeatureControl(
        featureName: feature,
        enabled: true,
        globalEnabled: true,
        updatedAt: DateTime.now().millisecondsSinceEpoch,
      );
    }
  }

  // Super Admin Login
  SuperAdmin? superAdminLogin(String email, String password, [String twoFactorCode = '']) {
    final superAdmin = _superAdmins[email] as SuperAdmin?;
    if (superAdmin == null || !superAdmin.isActive) return null;
    
    if (!_verifyPassword(password, superAdmin.passwordHash)) {
      _logAudit(superAdmin.id, UserRole.superAdmin, 'LOGIN_FAILED', 'Invalid password');
      return null;
    }
    
    if (superAdmin.twoFactorEnabled && !_verifyTwoFactor(superAdmin.twoFactorSecret, twoFactorCode)) {
      return null;
    }
    
    _logAudit(superAdmin.id, UserRole.superAdmin, 'LOGIN_SUCCESS', 'Super admin logged in');
    return superAdmin;
  }

  // Master Admin Request
  MasterAdmin createMasterAdminRequest(String email, String requestedBy) {
    final masterAdmin = MasterAdmin(
      id: _generateId(),
      email: email,
      passwordHash: _hashPassword(_generateTempPassword()),
      createdAt: DateTime.now().millisecondsSinceEpoch,
    );
    _masterAdmins[masterAdmin.id] = masterAdmin;
    _masterAdmins[email] = masterAdmin;
    _logAudit('SYSTEM', UserRole.superAdmin, 'MASTER_ADMIN_REQUEST', 'New request: $email');
    return masterAdmin;
  }

  // Authorize Master Admin
  bool authorizeMasterAdmin(String superAdminId, String masterAdminId, bool authorized, [String notes = '']) {
    if (_superAdmins[superAdminId] == null) {
      throw Exception('Only super admin can authorize');
    }
    
    final masterAdmin = _masterAdmins[masterAdminId] as MasterAdmin?;
    if (masterAdmin == null) return false;
    
    final updated = MasterAdmin(
      id: masterAdmin.id,
      email: masterAdmin.email,
      passwordHash: masterAdmin.passwordHash,
      authorizedBy: superAdminId,
      authorizationStatus: authorized ? AuthorizationStatus.authorized : AuthorizationStatus.rejected,
      twoFactorEnabled: masterAdmin.twoFactorEnabled,
      twoFactorSecret: masterAdmin.twoFactorSecret,
      phone: masterAdmin.phone,
      canCreateWhiteLabel: masterAdmin.canCreateWhiteLabel,
      canManageUsers: masterAdmin.canManageUsers,
      canManageWallets: masterAdmin.canManageWallets,
      canAccessFinance: masterAdmin.canAccessFinance,
      canModifyFeatures: masterAdmin.canModifyFeatures,
      canManageTokens: masterAdmin.canManageTokens,
      canManageNetworks: masterAdmin.canManageNetworks,
      canViewAnalytics: masterAdmin.canViewAnalytics,
      canManageAdmins: masterAdmin.canManageAdmins,
      maxWhiteLabels: masterAdmin.maxWhiteLabels,
      whiteLabelCount: masterAdmin.whiteLabelCount,
      status: authorized ? AdminStatus.active : AdminStatus.inactive,
      createdAt: masterAdmin.createdAt,
      lastLogin: masterAdmin.lastLogin,
      passwordChangedAt: masterAdmin.passwordChangedAt,
      failedAttempts: masterAdmin.failedAttempts,
      lockedUntil: masterAdmin.lockedUntil,
    );
    
    _masterAdmins.remove(masterAdmin.id);
    _masterAdmins[updated.email] = updated;
    
    _logAudit(superAdminId, UserRole.superAdmin, authorized ? 'AUTHORIZED' : 'REJECTED', 
        '${authorized ? "Authorized" : "Rejected"} ${masterAdmin.email}');
    return true;
  }

  // Master Admin Login
  MasterAdmin? masterAdminLogin(String email, String password, [String twoFactorCode = '']) {
    final masterAdmin = _masterAdmins[email] as MasterAdmin?;
    if (masterAdmin == null) return null;
    
    if (masterAdmin.authorizationStatus != AuthorizationStatus.authorized) return null;
    if (masterAdmin.status != AdminStatus.active) return null;
    if (masterAdmin.lockedUntil > DateTime.now().millisecondsSinceEpoch) return null;
    
    if (!_verifyPassword(password, masterAdmin.passwordHash)) {
      _logAudit(masterAdmin.id, UserRole.masterAdmin, 'LOGIN_FAILED', 'Invalid password');
      return null;
    }
    
    if (masterAdmin.twoFactorEnabled && !_verifyTwoFactor(masterAdmin.twoFactorSecret, twoFactorCode)) {
      return null;
    }
    
    _logAudit(masterAdmin.id, UserRole.masterAdmin, 'LOGIN_SUCCESS', 'Master admin logged in');
    return masterAdmin;
  }

  // Change Password
  bool changeMasterAdminPassword(String adminId, String oldPassword, String newPassword) {
    final masterAdmin = (_masterAdmins[adminId] as MasterAdmin?) ?? 
        (_masterAdmins.values.firstWhere((e) => e is MasterAdmin && e.email == adminId, orElse: () => null) as MasterAdmin?);
    if (masterAdmin == null) return false;
    
    if (!_verifyPassword(oldPassword, masterAdmin.passwordHash)) return false;
    if (newPassword.length < 8) return false;
    
    masterAdmin.passwordHash = _hashPassword(newPassword);
    masterAdmin.passwordChangedAt = DateTime.now().millisecondsSinceEpoch;
    
    _logAudit(adminId, UserRole.masterAdmin, 'PASSWORD_CHANGED', 'Password changed');
    return true;
  }

  // Enable 2FA
  bool enableMasterAdmin2FA(String adminId, String secret) {
    final masterAdmin = _masterAdmins[adminId] as MasterAdmin?;
    if (masterAdmin == null) return false;
    
    masterAdmin.twoFactorEnabled = true;
    masterAdmin.twoFactorSecret = secret;
    
    _logAudit(adminId, UserRole.masterAdmin, '2FA_ENABLED', '2FA enabled');
    return true;
  }

  // White Label Admin
  WhiteLabelAdmin? createWhiteLabelAdmin(String masterAdminId, String email, String brandName) {
    final masterAdmin = _masterAdmins[masterAdminId] as MasterAdmin?;
    if (masterAdmin == null || !masterAdmin.canCreateWhiteLabel) return null;
    if (masterAdmin.whiteLabelCount >= masterAdmin.maxWhiteLabels) return null;
    
    final whiteLabel = WhiteLabelAdmin(
      id: _generateId(),
      email: email,
      passwordHash: _hashPassword(_generateTempPassword()),
      masterAdminId: masterAdminId,
      brandName: brandName,
      authorizationStatus: AuthorizationStatus.authorized,
      status: AdminStatus.active,
      createdAt: DateTime.now().millisecondsSinceEpoch,
    );
    
    _whiteLabelAdmins[whiteLabel.id] = whiteLabel;
    _whiteLabelAdmins[email] = whiteLabel;
    
    _logAudit(masterAdminId, UserRole.masterAdmin, 'WHITE_LABEL_CREATED', 'Created: $email - $brandName');
    return whiteLabel;
  }

  // Feature Control
  bool setGlobalFeature(String superAdminId, String featureName, bool enabled) {
    if (_superAdmins[superAdminId] == null) {
      throw Exception('Only super admin can modify features');
    }
    
    final feature = _featureControls[featureName];
    if (feature == null) return false;
    
    feature.enabled = enabled;
    feature.globalEnabled = enabled;
    feature.updatedBy = superAdminId;
    feature.updatedAt = DateTime.now().millisecondsSinceEpoch;
    
    _logAudit(superAdminId, UserRole.superAdmin, 'FEATURE_TOGGLE', 'Set $featureName = $enabled');
    return true;
  }

  List<FeatureControl> getAllFeatures() => _featureControls.values.toList();

  bool isFeatureEnabled(String featureName, String adminId, UserRole role) {
    final feature = _featureControls[featureName];
    if (feature == null || !feature.globalEnabled) return false;
    
    switch (role) {
      case UserRole.superAdmin:
        return true;
      case UserRole.masterAdmin:
        if (feature.masterAdminId.isNotEmpty && feature.masterAdminId != adminId) return false;
        return feature.enabled;
      case UserRole.whiteLabelAdmin:
        if (feature.whiteLabelId.isNotEmpty && feature.whiteLabelId != adminId) return false;
        return feature.enabled;
      default:
        return false;
    }
  }

  // Audit
  void _logAudit(String adminId, UserRole role, String action, String details) {
    final log = AuditLog(
      id: _generateId(),
      adminId: adminId,
      adminRole: role,
      action: action,
      details: details,
      timestamp: DateTime.now().millisecondsSinceEpoch,
    );
    _auditLogs.add(log);
    print('[AUDIT] ${role.name} | $action | $details');
  }

  List<AuditLog> getAuditLogs([String adminId = '', int limit = 100]) {
    if (adminId.isEmpty) return _auditLogs.take(limit).toList();
    return _auditLogs.where((l) => l.adminId == adminId).take(limit).toList();
  }

  // Helpers
  // Cryptographically-secure randomness for IDs / secrets / passwords.
  // The previous implementation used dart:math Random() (a non-secure PRNG)
  // to generate secret keys and temporary passwords — a security flaw.
  String _generateId() =>
      'id_${DateTime.now().millisecondsSinceEpoch}_${Random.secure().nextInt(999999)}';

  String _generateSecretKey() =>
      List.generate(32, (_) => Random.secure().nextInt(256).toRadixString(16).padLeft(2, '0')).join();

  String _generateTempPassword() {
    const chars =
        'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    return List.generate(16, (_) => chars[Random.secure().nextInt(chars.length)]).join();
  }

  /// Real PBKDF2-HMAC-SHA256 password hashing with a per-hash salt.
  /// Replaces the previous toy hash (fold *31+e mod 2^32) which is NOT a
  /// password KDF and is trivially invertible. Output: "pbkdf2:iterations:saltHex:hashHex".
  String _hashPassword(String password) {
    final iterations = 100000;
    final salt = Uint8List.fromList(
        List<int>.generate(16, (_) => Random.secure().nextInt(256)));
    final pwBytes = Uint8List.fromList(utf8.encode(password));
    final derived = _pbkdf2(pwBytes, salt, iterations, 32);
    return 'pbkdf2:$iterations:${_toHex(salt)}:${_toHex(derived)}';
  }

  bool _verifyPassword(String password, String stored) {
    final parts = stored.split(':');
    if (parts.length != 4 || parts[0] != 'pbkdf2') return false;
    final iterations = int.tryParse(parts[1]) ?? 0;
    final salt = _fromHex(parts[2]);
    final expected = _fromHex(parts[3]);
    final derived = _pbkdf2(Uint8List.fromList(utf8.encode(password)), salt, iterations, expected.length);
    // Constant-time comparison.
    var diff = derived.length ^ expected.length;
    for (var i = 0; i < derived.length && i < expected.length; i++) {
      diff |= derived[i] ^ expected[i];
    }
    return diff == 0;
  }

  Uint8List _pbkdf2(Uint8List password, Uint8List salt, int iterations, int keyLength) {
    // RFC 2898 PBKDF2 with HMAC-SHA256.
    final hmacLength = 32;
    final blocks = (keyLength + hmacLength - 1) ~/ hmacLength;
    final out = BytesBuilder();
    for (var blockNum = 1; blockNum <= blocks; blockNum++) {
      final u = Uint8List.fromList([...salt, (blockNum >> 24) & 0xff, (blockNum >> 16) & 0xff, (blockNum >> 8) & 0xff, blockNum & 0xff]);
      var t = crypto.Hmac(crypto.sha256, password).convert(u).bytes;
      var result = Uint8List.fromList(t);
      for (var i = 1; i < iterations; i++) {
        t = crypto.Hmac(crypto.sha256, password).convert(t).bytes;
        for (var j = 0; j < hmacLength; j++) {
          result[j] ^= t[j];
        }
      }
      out.add(result);
    }
    return out.toBytes().sublist(0, keyLength);
  }

  String _toHex(Uint8List bytes) =>
      bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();

  Uint8List _fromHex(String hex) {
    final out = <int>[];
    for (var i = 0; i < hex.length; i += 2) {
      out.add(int.parse(hex.substring(i, i + 2), radix: 16));
    }
    return Uint8List.fromList(out);
  }

  /// Real RFC 6238 TOTP verification (HMAC-SHA1, 30s step, 6 digits).
  /// Replaces the previous check that only validated length==6 & numeric
  /// (which accepted ANY 6-digit code — a 2FA bypass).
  bool _verifyTwoFactor(String secret, String code) {
    if (code.length != 6) return false;
    final timeStep = DateTime.now().millisecondsSinceEpoch ~/ 1000 ~/ 30;
    final keyBytes = _base32Decode(secret.toUpperCase());
    // Allow a +/-1 window to tolerate clock drift.
    for (var offset = -1; offset <= 1; offset++) {
      final counter = timeStep + offset;
      final msg = ByteData(8)..setInt64(0, counter);
      final hmac = crypto.Hmac(crypto.sha1, keyBytes).convert(msg.buffer.asUint8List()).bytes;
      final offsetIdx = hmac[hmac.length - 1] & 0x0f;
      final binary = ((hmac[offsetIdx] & 0x7f) << 24) |
          ((hmac[offsetIdx + 1] & 0xff) << 16) |
          ((hmac[offsetIdx + 2] & 0xff) << 8) |
          (hmac[offsetIdx + 3] & 0xff);
      final expected = (binary % 1000000).toString().padLeft(6, '0');
      if (_constantTimeEquals(code, expected)) return true;
    }
    return false;
  }

  bool _constantTimeEquals(String a, String b) {
    if (a.length != b.length) return false;
    var diff = 0;
    for (var i = 0; i < a.length; i++) {
      diff |= a.codeUnitAt(i) ^ b.codeUnitAt(i);
    }
    return diff == 0;
  }

  Uint8List _base32Decode(String input) {
    const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
    final cleaned = input.replaceAll('=', '');
    var bits = 0, value = 0;
    final out = <int>[];
    for (final ch in cleaned.split('')) {
      final idx = alphabet.indexOf(ch.toUpperCase());
      if (idx < 0) continue;
      value = (value << 5) | idx;
      bits += 5;
      if (bits >= 8) {
        out.add((value >> (bits - 8)) & 0xff);
        bits -= 8;
      }
    }
    return Uint8List.fromList(out);
  }
}
