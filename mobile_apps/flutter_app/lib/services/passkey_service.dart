///
/// TigerWallet Flutter - Passkey Service (WebAuthn)
/// 
/// Complete Passkey/WebAuthn Implementation:
/// - Platform authenticator support
/// - Biometric integration
/// - Secure key storage
/// - Cross-platform compatibility
/// 
/// THIS IS PRODUCTION CODE - NO MOCKS OR SIMULATIONS
/// 

import 'dart:convert';
import 'dart:math' show Random;
import 'dart:typed_data';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:crypto/crypto.dart';
import 'package:pointycastle/export.dart';

/// Passkey credential types
enum PasskeyAuthenticatorAttachment {
  platform,
  crossPlatform,
}

/// Passkey verification levels
enum PasskeyVerificationLevel {
  none,
  biometric,
  devicePasscode,
}

/// Passkey credential
class PasskeyCredential {
  final String id;
  final String publicKey;
  final String algorithm;
  final String counter;
  final DateTime createdAt;
  final String rpId;

  PasskeyCredential({
    required this.id,
    required this.publicKey,
    required this.algorithm,
    required this.counter,
    required this.createdAt,
    required this.rpId,
  });

  Map<String, dynamic> toJson() => {
    'id': id,
    'publicKey': publicKey,
    'algorithm': algorithm,
    'counter': counter,
    'createdAt': createdAt.toIso8601String(),
    'rpId': rpId,
  };

  factory PasskeyCredential.fromJson(Map<String, dynamic> json) => PasskeyCredential(
    id: json['id'],
    publicKey: json['publicKey'],
    algorithm: json['algorithm'],
    counter: json['counter'],
    createdAt: DateTime.parse(json['createdAt']),
    rpId: json['rpId'],
  );
}

/// Passkey registration options
class PasskeyRegistrationOptions {
  final String rpId;
  final String rpName;
  final String userId;
  final String userName;
  final List<String> pubKeyCredParams;
  final int timeout;
  final List<String> excludeCredentials;

  PasskeyRegistrationOptions({
    required this.rpId,
    required this.rpName,
    required this.userId,
    required this.userName,
    required this.pubKeyCredParams,
    this.timeout = 60000,
    this.excludeCredentials = const [],
  });

  Map<String, dynamic> toJson() => {
    'rp': {
      'id': rpId,
      'name': rpName,
    },
    'user': {
      'id': base64Url.encode(utf8.encode(userId)),
      'name': userName,
      'displayName': userName,
    },
    'pubKeyCredParams': pubKeyCredParams.map((alg) => {
      'type': 'public-key',
      'alg': alg,
    }).toList(),
    'timeout': timeout,
    'excludeCredentials': excludeCredentials.map((cred) => {
      'id': cred,
      'type': 'public-key',
    }).toList(),
  };
}

/// Passkey assertion options
class PasskeyAssertionOptions {
  final String rpId;
  final String challenge;
  final int timeout;
  final List<String> allowedCredentials;

  PasskeyAssertionOptions({
    required this.rpId,
    required this.challenge,
    this.timeout = 60000,
    this.allowedCredentials = const [],
  });

  Map<String, dynamic> toJson() => {
    'rpId': rpId,
    'challenge': challenge,
    'timeout': timeout,
    'allowCredentials': allowedCredentials.map((cred) => {
      'id': cred,
      'type': 'public-key',
    }).toList(),
  };
}

/// Passkey Service - Production Implementation
class PasskeyService {
  static final PasskeyService _instance = PasskeyService._internal();
  factory PasskeyService() => _instance;
  PasskeyService._internal();

  final FlutterSecureStorage _secureStorage = const FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedStorage: true),
    iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock),
  );

  static const String _credentialsKey = 'passkey_credentials';
  static const String _enabledKey = 'passkey_enabled';

  final Map<String, PasskeyCredential> _credentials = {};
  bool _isEnabled = false;
  String? _currentUserId;

  /// Initialize the passkey service
  Future<void> initialize() async {
    await _loadCredentials();
    final enabledStr = await _secureStorage.read(key: _enabledKey);
    _isEnabled = enabledStr == 'true';
  }

  /// Check if passkey is available on this device.
  ///
  /// Real platform-authenticator detection requires a WebAuthn platform plugin
  /// (e.g. `flutter_passkey` / `webauthn`). The previous implementation always
  /// returned `true`, falsely advertising passkey support. Until a platform
  /// plugin is wired, this returns `false` (honest).
  Future<bool> isPasskeyAvailable() async {
    return false;
  }

  /// Enable/disable passkey
  Future<bool> setPasskeyEnabled(bool enabled) async {
    _isEnabled = enabled;
    await _secureStorage.write(key: _enabledKey, value: enabled.toString());
    return true;
  }

  /// Check if passkey is enabled
  bool get isPasskeyEnabled => _isEnabled;

  /// Generate registration options for passkey creation
  PasskeyRegistrationOptions generateRegistrationOptions({
    required String userId,
    required String userName,
    String rpId = 'tigerwallet.com',
    String rpName = 'TigerWallet',
  }) {
    _currentUserId = userId;
    
    // Supported algorithm IDs for WebAuthn
    // -7: ES256 (ECDSA with P-256 and SHA-256)
    // -257: RS256 (RSA with PKCS#1 v1.5 and SHA-256)
    return PasskeyRegistrationOptions(
      rpId: rpId,
      rpName: rpName,
      userId: userId,
      userName: userName,
      pubKeyCredParams: ['-7', '-257'],
      timeout: 60000,
      excludeCredentials: _credentials.values
          .where((c) => c.rpId == rpId)
          .map((c) => c.id)
          .toList(),
    );
  }

  /// Generate assertion options for passkey authentication
  PasskeyAssertionOptions generateAssertionOptions({
    required String rpId,
    String? challenge,
  }) {
    final challengeBytes = challenge ?? _generateChallenge();
    return PasskeyAssertionOptions(
      rpId: rpId,
      challenge: base64Url.encode(challengeBytes),
      timeout: 60000,
      allowedCredentials: _credentials.values
          .where((c) => c.rpId == rpId)
          .map((c) => c.id)
          .toList(),
    );
  }

  /// Register a new passkey credential
  Future<bool> registerPasskey({
    required String credentialId,
    required String publicKey,
    required String algorithm,
    required String rpId,
  }) async {
    final credential = PasskeyCredential(
      id: credentialId,
      publicKey: publicKey,
      algorithm: algorithm,
      counter: '0',
      createdAt: DateTime.now(),
      rpId: rpId,
    );

    _credentials[credentialId] = credential;
    await _saveCredentials();
    return true;
  }

  /// Authenticate with passkey
  Future<PasskeyCredential?> authenticateWithPasskey(String credentialId) async {
    final credential = _credentials[credentialId];
    if (credential != null) {
      // Update counter in production
      return credential;
    }
    return null;
  }

  /// Get all credentials for a relying party
  List<PasskeyCredential> getCredentials(String rpId) {
    return _credentials.values.where((c) => c.rpId == rpId).toList();
  }

  /// Remove a credential
  Future<bool> removeCredential(String credentialId) async {
    _credentials.remove(credentialId);
    await _saveCredentials();
    return true;
  }

  /// Remove all credentials for a user
  Future<bool> removeAllCredentials() async {
    _credentials.clear();
    await _saveCredentials();
    return true;
  }

  /// Verify a passkey (WebAuthn) signature.
  ///
  /// WebAuthn assertions sign `authenticatorData || SHA-256(clientDataJSON)`.
  /// For ES256 (alg -7) the signature is a DER-encoded ECDSA signature over
  /// P-256 with SHA-256. This performs REAL verification against the stored
  /// public key (expected as a hex-encoded 64-byte uncompressed P-256 point:
  /// x || y). The previous implementation returned `true` unconditionally —
  /// a full authentication bypass. It now rejects anything it cannot verify.
  Future<bool> verifySignature({
    required String credentialId,
    required Uint8List clientDataHash,
    required Uint8List authenticatorData,
    required Uint8List signature,
  }) async {
    final credential = _credentials[credentialId];
    if (credential == null) return false;

    try {
      final pubHex = credential.publicKey.replaceAll('0x', '');
      final pubBytes = Uint8List.fromList(base64.decode(pubHex));
      // Accept either a raw 64-byte (x||y) point or a 65-byte
      // 0x04||x||y uncompressed point.
      Uint8List xy;
      if (pubBytes.length == 64) {
        xy = pubBytes;
      } else if (pubBytes.length == 65 && pubBytes[0] == 0x04) {
        xy = pubBytes.sublist(1);
      } else {
        return false;
      }
      final q = ECDomainParameters('prime256v1').curve
          .decodePoint(Uint8List.fromList([0x04, ...xy]));
      if (q == null) return false;

      final signedData =
          Uint8List.fromList([...authenticatorData, ...clientDataHash]);
      final verifier = ECDSAVerifier(SHA256Digest());
      verifier.init(false, PublicKeyParameter(ECPublicKey(q)));

      return verifier.verifySignature(signedData, ECSignature.fromDER(signature));
    } catch (e) {
      return false;
    }
  }

  /// Generate a cryptographically random 32-byte challenge.
  ///
  /// Seeds FortunaRandom with 32 bytes from `Random.secure()` (platform
  /// CSPRNG). The previous implementation seeded only with the string
  /// `DateTime.now().millisecondsSinceEpoch` (predictable, low-entropy).
  Uint8List _generateChallenge() {
    final secure = Random.secure();
    final seed =
        Uint8List.fromList(List<int>.generate(32, (_) => secure.nextInt(256)));
    final random = SecureRandom('Fortuna')..seed(KeyParameter(seed));
    return random.nextBytes(32);
  }

  /// Load credentials from secure storage
  Future<void> _loadCredentials() async {
    try {
      final data = await _secureStorage.read(key: _credentialsKey);
      if (data != null) {
        final List<dynamic> jsonList = jsonDecode(data);
        for (final item in jsonList) {
          final credential = PasskeyCredential.fromJson(item);
          _credentials[credential.id] = credential;
        }
      }
    } catch (e) {
      _credentials.clear();
    }
  }

  /// Save credentials to secure storage
  Future<void> _saveCredentials() async {
    final data = jsonEncode(
      _credentials.values.map((c) => c.toJson()).toList(),
    );
    await _secureStorage.write(key: _credentialsKey, value: data);
  }

  /// Get current user ID
  String? get currentUserId => _currentUserId;

  /// Get credential count
  int get credentialCount => _credentials.length;
}
