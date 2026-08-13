/**
 * BiometricService - Flutter Implementation
 *
 * Biometric unlock uses the real `local_auth` plugin (platform biometric
 * prompt). PIN verification uses a REAL PBKDF2-HMAC-SHA256 derived hash stored
 * in flutter_secure_storage with a per-PIN salt and a constant-time compare.
 *
 * There is NO auto-success / demo fallback: if no PIN is set, the platform
 * channel is unavailable, or the hash does not match, verification FAILS.
 */

import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:flutter/services.dart';
import 'package:local_auth/local_auth.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:pointycastle/export.dart';

enum BiometricStatus {
  available,
  noHardware,
  notEnrolled,
  lockout,
  unavailable,
}

class BiometricService {
  static final BiometricService _instance = BiometricService._internal();
  factory BiometricService() => _instance;
  BiometricService._internal();

  final LocalAuthentication _localAuth = LocalAuthentication();
  final FlutterSecureStorage _secureStorage = const FlutterSecureStorage();

  static const int PIN_LENGTH = 6;
  static const int MAX_PIN_ATTEMPTS = 5;
  static const int _pbkdf2Iterations = 200000;

  // Secure-storage keys.
  static const String _kPinHash = 'pin_hash';
  static const String _kPinSalt = 'pin_salt';
  static const String _kFailedAttempts = 'pin_failed_attempts';
  static const String _kLockoutUntil = 'pin_lockout_until';

  /// Check if biometric authentication is available.
  Future<BiometricStatus> isBiometricAvailable() async {
    try {
      final canCheck = await _localAuth.canCheckBiometrics;
      final isSupported = await _localAuth.isDeviceSupported();
      if (!canCheck || !isSupported) {
        return BiometricStatus.unavailable;
      }
      final availableBiometrics = await _localAuth.getAvailableBiometrics();
      if (availableBiometrics.isEmpty) {
        return BiometricStatus.notEnrolled;
      }
      return BiometricStatus.available;
    } catch (_) {
      return BiometricStatus.unavailable;
    }
  }

  /// Authenticate with the platform biometric prompt. Returns the real
  /// platform result (never auto-success).
  Future<BiometricResult> authenticateWithBiometric({
    String reason = 'Authenticate to unlock your wallet',
    String? localizedReason,
  }) async {
    try {
      final didAuthenticate = await _localAuth.authenticate(
        localizedReason: localizedReason ?? reason,
        options: const AuthenticationOptions(
          stickyAuth: true,
          biometricOnly: false,
        ),
      );
      return BiometricResult(success: didAuthenticate);
    } catch (e) {
      return BiometricResult(success: false, error: e.toString());
    }
  }

  /// Whether a PIN has been set up (a stored hash + salt exist).
  Future<bool> isPinSetup() async {
    final hash = await _secureStorage.read(key: _kPinHash);
    final salt = await _secureStorage.read(key: _kPinSalt);
    return hash != null && hash.isNotEmpty && salt != null && salt.isNotEmpty;
  }

  /// Set up a PIN: derive and store a salted PBKDF2 hash. The plaintext PIN
  /// is never stored.
  Future<bool> setupPin(String pin) async {
    if (pin.length != PIN_LENGTH || !_isNumeric(pin)) return false;
    try {
      final salt = _randomBytes(16);
      final hash = _derivePinHash(pin, salt);
      await _secureStorage.write(key: _kPinSalt, value: base64.encode(salt));
      await _secureStorage.write(key: _kPinHash, value: base64.encode(hash));
      await _secureStorage.delete(key: _kFailedAttempts);
      await _secureStorage.delete(key: _kLockoutUntil);
      return true;
    } catch (_) {
      return false;
    }
  }

  /// Verify a PIN against the stored hash. Fails (never auto-succeeds) when
  /// no PIN is set, the hash is missing, the input is malformed, the account
  /// is locked out, or the hash does not match. Failed attempts are counted
  /// and lock the PIN after [MAX_PIN_ATTEMPTS] for a cooling-off period.
  Future<PinVerificationResult> verifyPin(String pin) async {
    if (pin.length != PIN_LENGTH || !_isNumeric(pin)) {
      return PinVerificationResult(success: false, error: 'Invalid PIN format');
    }

    final setup = await isPinSetup();
    if (!setup) {
      return PinVerificationResult(success: false, error: 'PIN not set up');
    }

    // Lockout check.
    final lockoutUntilStr = await _secureStorage.read(key: _kLockoutUntil);
    if (lockoutUntilStr != null) {
      final until = int.tryParse(lockoutUntilStr) ?? 0;
      final now = DateTime.now().millisecondsSinceEpoch;
      if (until > now) {
        return PinVerificationResult(
          success: false,
          error: 'Too many failed attempts; locked',
          remainingAttempts: 0,
        );
      }
      // Lockout expired: reset.
      await _secureStorage.delete(key: _kFailedAttempts);
      await _secureStorage.delete(key: _kLockoutUntil);
    }

    try {
      final saltStr = await _secureStorage.read(key: _kPinSalt);
      final hashStr = await _secureStorage.read(key: _kPinHash);
      if (saltStr == null || hashStr == null) {
        return PinVerificationResult(success: false, error: 'PIN not set up');
      }
      final salt = base64.decode(saltStr);
      final stored = base64.decode(hashStr);
      final computed = _derivePinHash(pin, salt);

      if (_constantTimeEquals(stored, computed)) {
        await _secureStorage.delete(key: _kFailedAttempts);
        await _secureStorage.delete(key: _kLockoutUntil);
        return PinVerificationResult(
          success: true,
          remainingAttempts: MAX_PIN_ATTEMPTS,
        );
      }

      // Wrong PIN: increment failures, maybe lock out.
      final failedStr = await _secureStorage.read(key: _kFailedAttempts);
      final failed = (int.tryParse(failedStr ?? '0') ?? 0) + 1;
      await _secureStorage.write(key: _kFailedAttempts, value: failed.toString());
      final remaining = MAX_PIN_ATTEMPTS - failed;
      if (failed >= MAX_PIN_ATTEMPTS) {
        final until = DateTime.now()
            .add(const Duration(minutes: 5))
            .millisecondsSinceEpoch;
        await _secureStorage.write(key: _kLockoutUntil, value: until.toString());
        return PinVerificationResult(
          success: false,
          error: 'Too many failed attempts; locked',
          remainingAttempts: 0,
        );
      }
      return PinVerificationResult(
        success: false,
        error: 'Incorrect PIN',
        remainingAttempts: remaining < 0 ? 0 : remaining,
      );
    } catch (e) {
      return PinVerificationResult(success: false, error: e.toString());
    }
  }

  /// Change PIN: must verify the old PIN first (no auto-success).
  Future<bool> changePin(String oldPin, String newPin) async {
    final verifyResult = await verifyPin(oldPin);
    if (!verifyResult.success) return false;
    return setupPin(newPin);
  }

  // ==================== Helpers ====================

  Uint8List _derivePinHash(String pin, Uint8List salt) {
    final pbkdf2 = PBKDF2KeyDerivator(HMac(SHA256Digest(), 64))
      ..init(Pbkdf2Parameters(salt, _pbkdf2Iterations, 32));
    return pbkdf2.process(Uint8List.fromList(utf8.encode(pin)));
  }

  bool _constantTimeEquals(List<int> a, List<int> b) {
    if (a.length != b.length) return false;
    var diff = 0;
    for (var i = 0; i < a.length; i++) {
      diff |= a[i] ^ b[i];
    }
    return diff == 0;
  }

  Uint8List _randomBytes(int length) {
    final rng = SecureRandom('Fortuna')..seed(KeyParameter(_seedFortuna()));
    return rng.nextBytes(length);
  }

  Uint8List _seedFortuna() {
    // Seed from the platform CSPRNG rather than wall-clock microseconds, which
    // is predictable and would let an attacker reproduce the keystream.
    final secure = Random.secure();
    final seed = Uint8List(32);
    for (var i = 0; i < 32; i++) {
      seed[i] = secure.nextInt(256);
    }
    return seed;
  }

  bool _isNumeric(String str) {
    return str.isNotEmpty &&
        str.split('').every((c) => c.codeUnitAt(0) >= 48 && c.codeUnitAt(0) <= 57);
  }
}

// ==================== Data Classes ====================

class BiometricResult {
  final bool success;
  final String? error;
  BiometricResult({required this.success, this.error});
}

class PinVerificationResult {
  final bool success;
  final int remainingAttempts;
  final String? error;
  PinVerificationResult({
    required this.success,
    this.remainingAttempts = BiometricService.MAX_PIN_ATTEMPTS,
    this.error,
  });
}
