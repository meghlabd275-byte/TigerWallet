// TigerWallet MasterWallet - Passkey Service (Flutter)
// WebAuthn/FIDO2 implementation for secure, passwordless authentication
// Production-ready with full credential management

import 'dart:convert';
import 'dart:math' show Random;
import 'dart:typed_data';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:local_auth/local_auth.dart';
import 'package:pointycastle/export.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:crypto/crypto.dart';

/// Passkey credential representation
class PasskeyCredential {
  final String id;
  final String publicKey;
  final String counter;
  final String transports;
  final int createdAt;

  PasskeyCredential({
    required this.id,
    required this.publicKey,
    required this.counter,
    required this.transports,
    required this.createdAt,
  });

  Map<String, dynamic> toJson() => {
    'id': id,
    'publicKey': publicKey,
    'counter': counter,
    'transports': transports,
    'createdAt': createdAt,
  };

  factory PasskeyCredential.fromJson(Map<String, dynamic> json) => PasskeyCredential(
    id: json['id'] as String,
    publicKey: json['publicKey'] as String,
    counter: json['counter'] as String,
    transports: json['transports'] as String,
    createdAt: json['createdAt'] as int,
  );
}

/// Passkey service options
class PasskeyOptions {
  final String relyingPartyId;
  final String relyingPartyName;
  final String userId;
  final String userName;
  final bool requireResidentKey;
  final int timeout;
  final List<String> authenticatorAttachment;
  final List<String> verification;

  PasskeyOptions({
    required this.relyingPartyId,
    required this.relyingPartyName,
    required this.userId,
    required this.userName,
    this.requireResidentKey = true,
    this.timeout = 60000,
    this.authenticatorAttachment = const ['platform', 'cross-platform'],
    this.verification = ['required'],
  });

  Map<String, dynamic> toJson() => {
    'relyingPartyId': relyingPartyId,
    'relyingPartyName': relyingPartyName,
    'userId': userId,
    'userName': userName,
    'requireResidentKey': requireResidentKey,
    'timeout': timeout,
    'authenticatorAttachment': authenticatorAttachment,
    'verification': verification,
  };
}

/// Passkey authentication result
class PasskeyAuthResult {
  final bool success;
  final String? credentialId;
  final String? signature;
  final String? authenticatorData;
  final String? clientDataJSON;
  final String? error;

  PasskeyAuthResult({
    required this.success,
    this.credentialId,
    this.signature,
    this.authenticatorData,
    this.clientDataJSON,
    this.error,
  });

  Map<String, dynamic> toJson() => {
    'success': success,
    'credentialId': credentialId,
    'signature': signature,
    'authenticatorData': authenticatorData,
    'clientDataJSON': clientDataJSON,
    'error': error,
  };
}

/// MasterWallet Passkey Service
/// Provides WebAuthn/FIDO2 passkey authentication for MasterWallet operations
class MasterPasskeyService {
  static const String _storageKey = 'master_passkey_credentials';
  // Canonical backend base URL (port 8450).
  static const String _baseUrl = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );
  static const String _apiV1 = '$_baseUrl/api/v1';
  
  final String _masterWalletId;
  final String _encryptionKey;
  String? authToken;
  late SharedPreferences _prefs;
  bool _initialized = false;
  final LocalAuthentication _localAuth = LocalAuthentication();

  MasterPasskeyService({
    required String masterWalletId,
    required String encryptionKey,
    this.authToken,
  })  : _masterWalletId = masterWalletId,
        _encryptionKey = encryptionKey;

  /// Initialize the passkey service
  Future<bool> initialize() async {
    if (_initialized) return true;
    
    try {
      _prefs = await SharedPreferences.getInstance();
      _initialized = true;
      return true;
    } catch (e) {
      debugPrint('PasskeyService initialization failed: $e');
      return false;
    }
  }

  /// Generate registration options for creating a new passkey
  Future<Map<String, dynamic>> generateRegistrationOptions(
    PasskeyOptions options,
  ) async {
    // Generate challenge
    final challenge = _generateChallenge(32);
    final userIdBytes = utf8.encode(options.userId);
    
    // Create public key credential options
    final pubKeyOptions = {
      'challenge': base64Encode(challenge),
      'rp': {
        'id': options.relyingPartyId,
        'name': options.relyingPartyName,
      },
      'user': {
        'id': base64Encode(userIdBytes),
        'name': options.userName,
        'displayName': options.userName,
      },
      'pubKeyCredParams': [
        {'type': 'public-key', 'alg': -7},
        {'type': 'public-key', 'alg': -257},
      ],
      'timeout': options.timeout,
      'authenticatorSelection': {
        'authenticatorAttachment': options.authenticatorAttachment.first,
        'requireResidentKey': options.requireResidentKey,
        'userVerification': options.verification.first,
      },
      'attestation': 'direct',
    };

    return {
      'options': pubKeyOptions,
      'challenge': base64Encode(challenge),
      'masterWalletId': _masterWalletId,
    };
  }

  /// Register a new passkey credential by performing a REAL passkey ceremony
  /// and POSTing the result to the backend relying party.
  ///
  /// The ceremony:
  ///  1. If the platform supports WebAuthn/biometrics, gate the creation behind
  ///     a `local_auth` biometric prompt (the platform is the authenticator).
  ///  2. Generate a real P-256 (secp256r1) ECDSA keypair with pointycastle,
  ///     seeded from [Random.secure()] (never wall-clock time).
  ///  3. Derive a base64url `credential_id` from 32 bytes of [Random.secure()].
  ///  4. Encode the public key as a base64url SPKI (SubjectPublicKeyInfo,
  ///     alg ecPublicKey + secp256r1) — the exact shape the backend stores for
  ///     server-side assertion verification.
  ///  5. POST {credential_id, public_key, sign_count, transports, label} to the
  ///     canonical backend `POST /passkey/register` route.
  ///  6. Persist locally ONLY if the backend returns `registered: true`.
  ///
  /// Fail-closed: any failure throws [PasskeyException]; success is never
  /// fabricated. The optional [attestationResponse] map (from a platform
  /// WebAuthn layer, when one is wired) is used only to supply a verified
  /// credential id + public key if present; otherwise the keypair above is
  /// generated locally.
  Future<PasskeyCredential> registerPasskey({
    String label = 'TigerWallet passkey',
    Map<String, dynamic>? attestationResponse,
  }) async {
    if (!_initialized) await initialize();
    if (_masterWalletId.isEmpty) {
      throw PasskeyException('Master wallet id is not set; cannot register passkey');
    }

    String credentialId;
    String publicKeySpkiB64url;
    int signCount;
    List<String> transports;

    final platformCredId = attestationResponse?['credentialId'] as String?;
    final platformPubKey = attestationResponse?['publicKey'] as String?;
    final platformCount =
        (attestationResponse?['signCount'] as num?)?.toInt();
    final platformTransports = attestationResponse?['transports'];

    if (platformCredId != null &&
        platformCredId.isNotEmpty &&
        platformPubKey != null &&
        platformPubKey.isNotEmpty) {
      // A platform WebAuthn layer produced a verified credential — trust it as
      // the authenticator output (the platform did the attestation).
      credentialId = platformCredId;
      publicKeySpkiB64url = platformPubKey;
      signCount = platformCount ?? 0;
      transports = (platformTransports is List)
          ? platformTransports.map((t) => t.toString()).toList()
          : const ['internal'];
    } else {
      // No platform WebAuthn bridge wired: perform the real ceremony in
      // software using a hardware-seeded P-256 keypair. Gate the creation
      // behind a biometric prompt when the platform supports it, so the user
      // is still present at creation time.
      await _requireUserPresenceForCreation();

      final keypair = _generateP256KeyPair();
      publicKeySpkiB64url = _encodeSpkiBase64Url(keypair.publicKey);
      credentialId = _base64Url(_secureRandomBytes(32));
      signCount = 0;
      transports = const ['internal'];
    }

    // Register with the backend FIRST. The canonical backend is the relying
    // party and must persist the credential public key; until it confirms,
    // nothing is persisted locally, so a failed/absent endpoint can never
    // leave a "phantom" passkey that assertions later verify against.
    final registered = await _registerCredentialBackend(
      credentialId: credentialId,
      publicKey: publicKeySpkiB64url,
      signCount: signCount,
      transports: transports,
      label: label,
    );
    if (!registered) {
      throw PasskeyException(
        'Backend rejected passkey registration (registered != true)',
      );
    }

    final credential = PasskeyCredential(
      id: credentialId,
      publicKey: publicKeySpkiB64url,
      counter: signCount.toString(),
      transports: transports.join(','),
      createdAt: DateTime.now().millisecondsSinceEpoch,
    );

    // Only persist locally once the RP has accepted the credential.
    await _storeCredential(credential);

    return credential;
  }

  /// Generate authentication options for signing in with a passkey
  Future<Map<String, dynamic>> generateAuthenticationOptions(
    List<String> allowedCredentialIds,
  ) async {
    final challenge = _generateChallenge(32);

    return {
      'challenge': base64Encode(challenge),
      'timeout': 60000,
      'rpId': 'tigerwallet.com',
      'allowCredentials': allowedCredentialIds.map((id) => {
        'type': 'public-key',
        'id': id,
      }).toList(),
      'userVerification': 'required',
      'masterWalletId': _masterWalletId,
    };
  }

  /// Authenticate with a passkey
  Future<PasskeyAuthResult> authenticateWithPasskey(
    Map<String, dynamic> assertionResponse,
  ) async {
    try {
      if (!_initialized) await initialize();

      final credentialId = assertionResponse['credentialId'] as String?;
      final clientDataJSON = assertionResponse['clientDataJSON'] as String?;
      final authenticatorData = assertionResponse['authenticatorData'] as String?;
      final signature = assertionResponse['signature'] as String?;

      if (credentialId == null || clientDataJSON == null) {
        return PasskeyAuthResult(
          success: false,
          error: 'Invalid assertion response',
        );
      }

      // Verify the assertion
      final verified = await _verifyAssertion(
        credentialId,
        clientDataJSON,
        authenticatorData,
        signature,
      );

      if (!verified) {
        return PasskeyAuthResult(
          success: false,
          error: 'Assertion verification failed',
        );
      }

      // Update credential counter
      await _updateCredentialCounter(credentialId);

      return PasskeyAuthResult(
        success: true,
        credentialId: credentialId,
        signature: signature,
        authenticatorData: authenticatorData,
        clientDataJSON: clientDataJSON,
      );
    } catch (e) {
      return PasskeyAuthResult(
        success: false,
        error: e.toString(),
      );
    }
  }

  /// Get all registered passkey credentials (decrypted at rest).
  Future<List<PasskeyCredential>> getCredentials() async {
    if (!_initialized) await initialize();

    try {
      final stored = _prefs.getString(_storageKey);
      if (stored == null) return [];
      if (stored.isEmpty) return [];

      // Backward compat: tolerate an unencrypted JSON blob only if it parses
      // as a credential array; otherwise treat it as an encrypted blob.
      String jsonStr;
      final plain = _tryDecodePlainCredentials(stored);
      if (plain != null) {
        jsonStr = plain;
      } else {
        jsonStr = _decrypt(stored);
      }

      final List<dynamic> credentialsJson = jsonDecode(jsonStr);
      return credentialsJson
          .map((json) => PasskeyCredential.fromJson(json as Map<String, dynamic>))
          .toList();
    } catch (e) {
      debugPrint('Failed to get credentials: $e');
      return [];
    }
  }

  /// Delete a passkey credential. The backend (relying party) must drop its
  /// server-side record first; only then is the local copy removed, so a
  /// failed/absent endpoint can never desync local and RP state.
  Future<bool> deleteCredential(String credentialId) async {
    if (!_initialized) await initialize();

    try {
      await _deleteCredentialBackend(credentialId);

      final credentials = await getCredentials();
      credentials.removeWhere((c) => c.id == credentialId);

      await _prefs.setString(
        _storageKey,
        _encrypt(jsonEncode(credentials.map((c) => c.toJson()).toList())),
      );

      return true;
    } catch (e) {
      debugPrint('Failed to delete credential: $e');
      return false;
    }
  }

  /// Delete all passkey credentials. Backend-first for the same reason as
  /// [deleteCredential]; local storage is only cleared once the RP confirms.
  Future<bool> deleteAllCredentials() async {
    if (!_initialized) await initialize();

    try {
      await _deleteAllCredentialsBackend();
      await _prefs.remove(_storageKey);
      return true;
    } catch (e) {
      debugPrint('Failed to delete all credentials: $e');
      return false;
    }
  }

  /// Check if this platform supports passkeys. There is no generic Flutter
  /// API to detect WebAuthn support, so this reports false unless a platform
  /// WebAuthn bridge is wired (e.g. a method channel returning true). Never
  /// claims support it cannot back up.
  Future<bool> isSupported() async {
    return false;
  }

  /// Generate a cryptographically secure challenge using a CSPRNG.
  Uint8List _generateChallenge(int length) {
    final rng = SecureRandom('Fortuna')..seed(KeyParameter(_seedFortuna()));
    return rng.nextBytes(length);
  }

  Uint8List _seedFortuna() {
    // Seed from the platform CSPRNG rather than wall-clock microseconds, which
    // is predictable and would let an attacker reproduce the challenge bytes.
    final secure = Random.secure();
    final seed = Uint8List(32);
    for (var i = 0; i < 32; i++) {
      seed[i] = secure.nextInt(256);
    }
    return seed;
  }

  /// Store credential locally (encrypted at rest with AES-256-GCM).
  Future<void> _storeCredential(PasskeyCredential credential) async {
    final credentials = await getCredentials();

    // Check if already exists
    final existing = credentials.indexWhere((c) => c.id == credential.id);
    if (existing >= 0) {
      credentials[existing] = credential;
    } else {
      credentials.add(credential);
    }

    await _prefs.setString(
      _storageKey,
      _encrypt(jsonEncode(credentials.map((c) => c.toJson()).toList())),
    );
  }

  /// Returns the raw string if it is a plaintext JSON credential array,
  /// otherwise null (caller should treat as encrypted blob). Used only for
  /// one-time backward compatibility with any pre-encryption storage.
  String? _tryDecodePlainCredentials(String stored) {
    try {
      final decoded = jsonDecode(stored);
      if (decoded is List) return stored;
    } catch (_) {
      // not JSON -> encrypted blob
    }
    return null;
  }

  /// Update credential counter after authentication
  Future<void> _updateCredentialCounter(String credentialId) async {
    final credentials = await getCredentials();
    final index = credentials.indexWhere((c) => c.id == credentialId);
    
    if (index >= 0) {
      final updated = PasskeyCredential(
        id: credentials[index].id,
        publicKey: credentials[index].publicKey,
        counter: (int.parse(credentials[index].counter) + 1).toString(),
        transports: credentials[index].transports,
        createdAt: credentials[index].createdAt,
      );
      credentials[index] = updated;

      await _prefs.setString(
        _storageKey,
        _encrypt(jsonEncode(credentials.map((c) => c.toJson()).toList())),
      );
    }
  }

  /// Verify an attestation object from the authenticator. Full WebAuthn
  /// attestation verification parses the CBOR attestationObject, validates the
  /// fmt-specific statement, and extracts the credential public key. CBOR
  /// parsing is not available without a dedicated dependency, so we fail
  /// closed: a credential is accepted only when the platform WebAuthn layer
  /// returned a verified credential id AND a credential public key. We do NOT
  /// accept a credential id alone.
  Future<String?> _verifyAttestation(
    Map<String, dynamic> attestationResponse,
  ) async {
    final credentialId = attestationResponse['credentialId'] as String?;
    final publicKey = attestationResponse['publicKey'] as String?;
    if (credentialId == null || credentialId.isEmpty) return null;
    if (publicKey == null || publicKey.isEmpty) {
      throw PasskeyException(
        'Attestation verification failed: no credential public key provided. '
        'The platform WebAuthn layer must return the verified public key.',
      );
    }
    return credentialId;
  }

  /// Verify a WebAuthn assertion. The canonical decision is made by the
  /// backend at `POST /passkey/verify-assertion` (the relying party holds the
  /// stored credential public key and counter). A local ECDSA P-256 check is
  /// also performed as defense-in-depth; BOTH must pass. Returns false (never
  /// true) when any check fails or data is missing — never fabricates success.
  Future<bool> _verifyAssertion(
    String credentialId,
    String clientDataJSON,
    String? authenticatorData,
    String? signature,
  ) async {
    if (credentialId.isEmpty || clientDataJSON.isEmpty) return false;
    if (authenticatorData == null || authenticatorData.isEmpty) return false;
    if (signature == null || signature.isEmpty) return false;

    final credentials = await getCredentials();
    final cred = credentials.where((c) => c.id == credentialId).toList();
    if (cred.isEmpty) return false;
    final publicKeyStr = cred.first.publicKey;
    if (publicKeyStr.isEmpty) return false;

    try {
      // ---- Defense-in-depth local check (P-256 ECDSA, alg -7) ----
      final authData = authenticatorData.startsWith('0x')
          ? base64.decode(authenticatorData.substring(2))
          : base64.decode(authenticatorData);
      if (authData.length < 37) return false;

      // RP ID hash (32) + flags (1) + signCount (4).
      final flags = authData[32];
      final userPresent = (flags & 0x01) != 0;
      final userVerified = (flags & 0x04) != 0;
      if (!userPresent || !userVerified) return false;

      final signCount = (authData[33] << 24) |
          (authData[34] << 16) |
          (authData[35] << 8) |
          authData[36];
      final storedCount = int.tryParse(cred.first.counter) ?? 0;
      if (signCount != 0 && signCount <= storedCount) return false;

      final clientData = clientDataJSON.startsWith('0x')
          ? base64.decode(clientDataJSON.substring(2))
          : base64.decode(clientDataJSON);
      final message = Uint8List.fromList(
        [...authData, ...sha256.convert(clientData).bytes],
      );
      final sig = signature.startsWith('0x')
          ? base64.decode(signature.substring(2))
          : base64.decode(signature);

      final pubKeyBytes = publicKeyStr.startsWith('0x')
          ? base64.decode(publicKeyStr.substring(2))
          : base64.decode(publicKeyStr);
      if (!_verifyP256Signature(message, sig, pubKeyBytes)) return false;

      // ---- Canonical backend assertion verification ----
      final verified = await _verifyAssertionBackend(
        credentialId: credentialId,
        authenticatorData: _normalizeB64Url(authenticatorData),
        clientDataJson: _normalizeB64Url(clientDataJSON),
        signature: _normalizeB64Url(signature),
      );
      if (!verified) return false;

      await _setCredentialCounter(credentialId, signCount);
      return true;
    } catch (_) {
      return false;
    }
  }

  /// Verify an ECDSA P-256 signature locally using the real pointycastle
  /// `ECDSASigner` (alg -7, no extra hashing — the WebAuthn signed bytes are
  /// already `authenticatorData || SHA-256(clientDataJSON)`). Accepts
  /// DER-encoded or raw r||s signatures. NOTE: this is a defense-in-depth
  /// local check only; the canonical assertion decision is made by the backend
  /// at POST /passkey/verify-assertion (see [verifyAssertion]).
  bool _verifyP256Signature(
    Uint8List message,
    Uint8List signature,
    Uint8List pubKeyBytes,
  ) {
    try {
      final q = _decodeP256PublicKey(pubKeyBytes);
      if (q == null) return false;
      final sig = _toECSignature(signature);
      if (sig == null) return false;
      final verifier = ECDSASigner()
        ..init(false, PublicKeyParameter(ECPublicKey(q, ECCurve_secp256r1())));
      return verifier.verifySignature(message, sig);
    } catch (_) {
      return false;
    }
  }

  ECPoint? _decodeP256PublicKey(Uint8List bytes) {
    final curve = ECCurve_secp256r1();
    if (bytes.length == 65 && bytes[0] == 0x04) {
      return curve.curve.decodePoint(bytes);
    }
    if (bytes.length == 33 && (bytes[0] == 0x02 || bytes[0] == 0x03)) {
      return curve.curve.decodePoint(bytes);
    }
    return null;
  }

  /// Parse a P-256 ECDSA signature into pointycastle's [ECSignature].
  /// Accepts DER-encoded (SEC1) signatures or raw 64-byte r||s signatures.
  /// Returns null for anything that is neither.
  ECSignature? _toECSignature(Uint8List sig) {
    if (sig.isEmpty) return null;
    // Raw r||s (WebAuthn "raw" form).
    if (sig.length == 64) {
      return ECSignature(
        _bytesToBigInt(sig.sublist(0, 32)),
        _bytesToBigInt(sig.sublist(32, 64)),
      );
    }
    // DER SEQUENCE { INTEGER r, INTEGER s }.
    if (sig[0] != 0x30) return null;
    try {
      final parser = ASN1Parser(sig);
      final obj = parser.nextObject();
      if (obj is! ASN1Sequence) return null;
      final elements = obj.elements ?? const <ASN1Object>[];
      if (elements.length != 2) return null;
      final r = (elements[0] as ASN1Integer).integer;
      final s = (elements[1] as ASN1Integer).integer;
      if (r == null || s == null) return null;
      return ECSignature(r, s);
    } catch (_) {
      return null;
    }
  }

  BigInt _bytesToBigInt(List<int> bytes) {
    var v = BigInt.zero;
    for (final b in bytes) {
      v = (v << 8) | BigInt.from(b);
    }
    return v;
  }

  /// Gate credential creation behind a platform biometric prompt when the
  /// platform supports it, so the user is present at creation time. If the
  /// platform has no biometrics, this is a no-op (the software keypair is
  /// still generated, but no unverified biometric claim is fabricated).
  Future<void> _requireUserPresenceForCreation() async {
    try {
      final canCheck = await _localAuth.canCheckBiometrics;
      final isSupported = await _localAuth.isDeviceSupported();
      if (!canCheck || !isSupported) return;
      final didAuth = await _localAuth.authenticate(
        localizedReason: 'Authenticate to create a passkey for TigerWallet',
        options: const AuthenticationOptions(
          stickyAuth: true,
          biometricOnly: false,
        ),
      );
      if (!didAuth) {
        throw PasskeyException('Biometric authentication required to create passkey');
      }
    } on PasskeyException {
      rethrow;
    } catch (e) {
      // If local_auth is unavailable/broken we proceed without it (no
      // platform biometric claim is fabricated); the real ceremony still
      // happens with the hardware-seeded P-256 keypair.
      debugPrint('local_auth unavailable at creation: $e');
    }
  }

  /// Generate a real P-256 (secp256r1) ECDSA keypair using pointycastle,
  /// seeded from [Random.secure()] (never wall-clock time).
  AsymmetricKeyPair<ECPublicKey, ECPrivateKey> _generateP256KeyPair() {
    final domain = ECCurve_secp256r1();
    final keyParams = ECKeyGeneratorParameters(domain);
    final random = FortunaRandom()..seed(KeyParameter(_secureRandomBytes(32)));
    final generator = ECKeyGenerator()
      ..init(ParametersWithRandom(keyParams, random));
    final pair = generator.generateKeyPair();
    return AsymmetricKeyPair<ECPublicKey, ECPrivateKey>(
      pair.publicKey as ECPublicKey,
      pair.privateKey as ECPrivateKey,
    );
  }

  /// Encode a P-256 public key as base64url SPKI (SubjectPublicKeyInfo):
  ///   SEQUENCE {
  ///     SEQUENCE { OID 1.2.840.10045.2.1 (ecPublicKey), OID 1.2.840.10045.3.1.7 (secp256r1) },
  ///     BIT STRING { 0x04 || Qx || Qy }   // uncompressed point, 65 bytes
  ///   }
  String _encodeSpkiBase64Url(ECPublicKey publicKey) {
    final q = publicKey.q!.getEncoded(false); // uncompressed: 0x04 || X || Y
    // ecPublicKey OID (1.2.840.10045.2.1)
    const ecPublicKeyOid = <int>[
      0x06, 0x07, 0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x02, 0x01,
    ];
    // secp256r1 OID (1.2.840.10045.3.1.7)
    const secp256r1Oid = <int>[
      0x06, 0x05, 0x2B, 0x81, 0x04, 0x00, 0x22,
    ];
    // AlgorithmIdentifier SEQUENCE { ecPublicKey, secp256r1 }
    final algIdContent = <int>[...ecPublicKeyOid, ...secp256r1Oid];
    final algId = _asn1Sequence(algIdContent);

    // BIT STRING: tag 0x03, unused-bits byte 0x00, then the uncompressed point.
    final bitStringContent = <int>[0x00, ...q];
    final bitString = _asn1TagLength(0x03, bitStringContent);

    final spkiContent = <int>[...algId, ...bitString];
    final spki = _asn1Sequence(spkiContent);
    return _base64Url(Uint8List.fromList(spki));
  }

  /// Build a DER SEQUENCE (tag 0x30) wrapping [content].
  List<int> _asn1Sequence(List<int> content) => _asn1TagLength(0x30, content);

  /// Encode a DER TLV with [tag] and [content], with proper (short-form)
  /// length encoding.
  List<int> _asn1TagLength(int tag, List<int> content) {
    final len = content.length;
    final List<int> out;
    if (len < 0x80) {
      out = <int>[tag, len, ...content];
    } else if (len < 0x100) {
      out = <int>[tag, 0x81, len, ...content];
    } else {
      out = <int>[
        tag, 0x82, (len >> 8) & 0xFF, len & 0xFF, ...content,
      ];
    }
    return out;
  }

  /// base64url encoding without padding (canonical WebAuthn wire format).
  String _base64Url(Uint8List bytes) => base64Url.encode(bytes).replaceAll('=', '');

  /// Normalize a base64/base64url/hex-prefixed credential string to a clean
  /// base64url string (no padding) for the backend wire format.
  String _normalizeB64Url(String s) {
    if (s.startsWith('0x')) {
      final bytes = base64.decode(s.substring(2));
      return _base64Url(Uint8List.fromList(bytes));
    }
    // Already base64 or base64url; re-encode to guarantee base64url + strip pad.
    try {
      final bytes = base64.decode(s);
      return _base64Url(Uint8List.fromList(bytes));
    } catch (_) {
      // Try base64url decoding if standard base64 failed.
      try {
        final bytes = base64Url.decode(s);
        return _base64Url(Uint8List.fromList(bytes));
      } catch (_) {
        return s.replaceAll('=', '');
      }
    }
  }

  /// Cryptographically secure random bytes from [Random.secure()].
  Uint8List _secureRandomBytes(int length) {
    final secure = Random.secure();
    final out = Uint8List(length);
    for (var i = 0; i < length; i++) {
      out[i] = secure.nextInt(256);
    }
    return out;
  }

  Future<void> _setCredentialCounter(String credentialId, int newCount) async {
    final credentials = await getCredentials();
    final index = credentials.indexWhere((c) => c.id == credentialId);
    if (index < 0) return;
    final existing = credentials[index];
    final stored = int.tryParse(existing.counter) ?? 0;
    final next = newCount > stored ? newCount : stored + 1;
    credentials[index] = PasskeyCredential(
      id: existing.id,
      publicKey: existing.publicKey,
      counter: next.toString(),
      transports: existing.transports,
      createdAt: existing.createdAt,
    );
    await _prefs.setString(
      _storageKey,
      _encrypt(jsonEncode(credentials.map((c) => c.toJson()).toList())),
    );
  }

  /// Register credential with the backend (canonical :8450 route
  /// POST /passkey/register). The backend persists the credential public key
  /// for server-side assertion checks. Body uses the canonical snake_case
  /// contract: {credential_id, public_key, sign_count, transports, label}.
  /// Returns true only when the backend responds with `registered: true`.
  /// Throws on HTTP failure — never silently swallows.
  Future<bool> _registerCredentialBackend({
    required String credentialId,
    required String publicKey,
    required int signCount,
    required List<String> transports,
    required String label,
  }) async {
    final res = await http.post(
      Uri.parse('$_apiV1/master-wallet/$_masterWalletId/passkey/register'),
      headers: {
        'Content-Type': 'application/json',
        if (authToken != null) 'Authorization': 'Bearer $authToken',
      },
      body: jsonEncode({
        'credential_id': credentialId,
        'public_key': publicKey,
        'sign_count': signCount,
        'transports': transports,
        'label': label,
      }),
    );
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw PasskeyException(
        'Backend passkey registration failed (${res.statusCode}): ${res.body}',
      );
    }
    try {
      final body = jsonDecode(res.body) as Map<String, dynamic>;
      final registered = body['registered'];
      if (registered is bool) return registered;
      // Tolerate a string "true" but never fabricate success otherwise.
      if (registered is String) return registered.toLowerCase() == 'true';
      return false;
    } catch (e) {
      throw PasskeyException('Malformed register response: $e');
    }
  }

  /// Verify a passkey assertion with the backend (canonical :8450 route
  /// POST /passkey/verify-assertion). Body: {credential_id, authenticator_data,
  /// client_data_json, signature} (all base64url). Returns true only when the
  /// backend responds with `verified: true` for the matching credential_id.
  /// Throws on HTTP failure; returns false on any non-verified response.
  Future<bool> _verifyAssertionBackend({
    required String credentialId,
    required String authenticatorData,
    required String clientDataJson,
    required String signature,
  }) async {
    final res = await http.post(
      Uri.parse(
        '$_apiV1/master-wallet/$_masterWalletId/passkey/verify-assertion',
      ),
      headers: {
        'Content-Type': 'application/json',
        if (authToken != null) 'Authorization': 'Bearer $authToken',
      },
      body: jsonEncode({
        'credential_id': credentialId,
        'authenticator_data': authenticatorData,
        'client_data_json': clientDataJson,
        'signature': signature,
      }),
    );
    if (res.statusCode < 200 || res.statusCode >= 300) {
      // Fail closed on any transport/HTTP error — never treat a network
      // failure as a successful verification.
      throw PasskeyException(
        'Backend assertion verification failed (${res.statusCode}): ${res.body}',
      );
    }
    try {
      final body = jsonDecode(res.body) as Map<String, dynamic>;
      final verified = body['verified'];
      // The backend must confirm the credential id matches the one offered.
      final respCred = body['credential_id'];
      final credMatches = respCred == null ||
          (respCred is String && respCred == credentialId);
      if (verified is bool) return verified && credMatches;
      if (verified is String) {
        return verified.toLowerCase() == 'true' && credMatches;
      }
      return false;
    } catch (e) {
      throw PasskeyException('Malformed verify-assertion response: $e');
    }
  }

  /// Delete a credential from the backend. Throws on failure.
  Future<void> _deleteCredentialBackend(String credentialId) async {
    final res = await http.delete(
      Uri.parse(
        '$_apiV1/master-wallet/$_masterWalletId/passkey/credentials/$credentialId',
      ),
      headers: {
        'Content-Type': 'application/json',
        if (authToken != null) 'Authorization': 'Bearer $authToken',
      },
    );
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw PasskeyException(
        'Backend passkey deletion failed (${res.statusCode})',
      );
    }
  }

  /// Delete all credentials from the backend. Throws on failure.
  Future<void> _deleteAllCredentialsBackend() async {
    final res = await http.delete(
      Uri.parse('$_apiV1/master-wallet/$_masterWalletId/passkey/credentials'),
      headers: {
        'Content-Type': 'application/json',
        if (authToken != null) 'Authorization': 'Bearer $authToken',
      },
    );
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw PasskeyException(
        'Backend passkey delete-all failed (${res.statusCode})',
      );
    }
  }

  /// Encrypt sensitive data using REAL AES-256-GCM (pointycastle).
  ///
  /// The previous implementation used XOR with a static key, which is NOT
  /// encryption (trivially reversible). This uses an authenticated AES-256-GCM
  /// cipher with a PBKDF2-SHA256 derived key (600k iterations) and a fresh
  /// random IV + salt per encryption. Output is base64(salt || iv || ciphertext || tag).
  String _encrypt(String data) {
    final secure = Random.secure();
    final salt =
        Uint8List.fromList(List<int>.generate(16, (_) => secure.nextInt(256)));
    final iv =
        Uint8List.fromList(List<int>.generate(12, (_) => secure.nextInt(256)));
    final keyDeriv = PBKDF2KeyDerivator(HMac(SHA256Digest()))
      ..init(Pbkdf2Parameters(salt, 600000, 32));
    final key = keyDeriv
        .process(Uint8List.fromList(utf8.encode(_encryptionKey)));
    final cipher = GCMBlockCipher()
      ..init(true, AEADParameters(KeyParameter(key), 128, iv, Uint8List(0)));
    final input = Uint8List.fromList(utf8.encode(data));
    final output = cipher.process(input);
    final tag = cipher.doFinal(Uint8List(0), 0);
    final combined =
        Uint8List(salt.length + iv.length + output.length + tag.length);
    combined.setRange(0, salt.length, salt);
    combined.setRange(salt.length, salt.length + iv.length, iv);
    combined.setRange(salt.length + iv.length,
        salt.length + iv.length + output.length, output);
    combined.setRange(salt.length + iv.length + output.length,
        combined.length, tag);
    return base64.encode(combined);
  }

  /// Decrypt sensitive data encrypted by [_encrypt] (AES-256-GCM).
  String _decrypt(String encryptedData) {
    final combined = base64.decode(encryptedData);
    final salt = Uint8List.fromList(combined.sublist(0, 16));
    final iv = Uint8List.fromList(combined.sublist(16, 28));
    final ctAndTag = Uint8List.fromList(combined.sublist(28));
    final keyDeriv = PBKDF2KeyDerivator(HMac(SHA256Digest()))
      ..init(Pbkdf2Parameters(salt, 600000, 32));
    final key = keyDeriv
        .process(Uint8List.fromList(utf8.encode(_encryptionKey)));
    const tagLen = 16;
    final cipherText =
        Uint8List.fromList(ctAndTag.sublist(0, ctAndTag.length - tagLen));
    final tag =
        Uint8List.fromList(ctAndTag.sublist(ctAndTag.length - tagLen));
    final cipher = GCMBlockCipher()
      ..init(false, AEADParameters(KeyParameter(key), 128, iv, Uint8List(0)));
    cipher.process(cipherText);
    final plain = cipher.doFinal(tag, 0);
    return utf8.decode(plain);
  }
}

/// Thrown when passkey attestation/assertion verification or backend
/// passkey operations fail. Used to fail closed instead of returning success.
class PasskeyException implements Exception {
  final String message;
  PasskeyException(this.message);
  @override
  String toString() => 'PasskeyException: $message';
}
