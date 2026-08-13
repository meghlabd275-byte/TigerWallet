/**
 * PrivacyService - Flutter Implementation
 *
 * Real symmetric encryption (AES-256-GCM) is provided locally via pointycastle
 * for encrypting secrets at rest. Zero-knowledge proofs, stealth addresses,
 * and CoinJoin-style mixing have NO canonical backend implementation; those
 * methods throw fail-closed rather than emitting placeholder/XOR output.
 *
 * NO fake crypto, NO XOR "encryption", NO placeholder ZK proofs.
 */

import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:pointycastle/export.dart';

class PrivacyService {
  static PrivacyService? _instance;
  static PrivacyService get instance {
    _instance ??= PrivacyService._();
    return _instance!;
  }

  PrivacyService._();

  static const String _encVersion = '1';

  // ==================== AES-256-GCM (real) ====================

  /// Encrypt [plaintext] with AES-256-GCM. [password] is stretched via
  /// PBKDF2-HMAC-SHA256 (100k iterations). Returns a base64 blob:
  /// `v1.<base64(salt)>.<base64(iv)>.<base64(ciphertext+tag)>`.
  String encrypt(String plaintext, String password) {
    final salt = _randomBytes(16);
    final iv = _randomBytes(12);
    final key = _deriveKey(password, salt, iterations: 100000, keyLength: 32);
    final ciphertext = _aesGcmEncrypt(key, iv, utf8.encode(plaintext));
    return [
      'v$_encVersion',
      base64.encode(salt),
      base64.encode(iv),
      base64.encode(ciphertext),
    ].join('.');
  }

  /// Decrypt a blob produced by [encrypt]. Throws on wrong password (GCM auth
  /// tag mismatch) or malformed input - never returns garbage.
  String decrypt(String blob, String password) {
    final parts = blob.split('.');
    if (parts.length != 4 || parts[0] != 'v$_encVersion') {
      throw PrivacyException('malformed encrypted blob');
    }
    final salt = base64.decode(parts[1]);
    final iv = base64.decode(parts[2]);
    final ciphertext = base64.decode(parts[3]);
    final key = _deriveKey(password, salt, iterations: 100000, keyLength: 32);
    final plaintext = _aesGcmDecrypt(key, iv, ciphertext);
    return utf8.decode(plaintext);
  }

  Uint8List _deriveKey(String password, Uint8List salt,
      {required int iterations, required int keyLength}) {
    final pbkdf2 = PBKDF2KeyDerivator(HMac(SHA256Digest(), 64))
      ..init(Pbkdf2Parameters(salt, iterations, keyLength));
    return pbkdf2.process(Uint8List.fromList(utf8.encode(password)));
  }

  Uint8List _aesGcmEncrypt(Uint8List key, Uint8List iv, List<int> plaintext) {
    final cipher = GCMBlockCipher(AESEngine())
      ..init(true, AEADParameters(KeyParameter(key), 128, iv, Uint8List(0)));
    final input = Uint8List.fromList(plaintext);
    final output = cipher.process(input);
    final tag = cipher.doFinal(Uint8List(0), 0);
    final combined = Uint8List(output.length + tag.length);
    combined.setRange(0, output.length, output);
    combined.setRange(output.length, combined.length, tag);
    return Uint8List.fromList(combined);
  }

  Uint8List _aesGcmDecrypt(Uint8List key, Uint8List iv, List<int> ciphertext) {
    final cipher = GCMBlockCipher(AESEngine())
      ..init(false, AEADParameters(KeyParameter(key), 128, iv, Uint8List(0)));
    final ct = Uint8List.fromList(ciphertext);
    const tagLen = 16;
    final cipherText =
        Uint8List.fromList(ct.sublist(0, ct.length - tagLen));
    final tag = Uint8List.fromList(ct.sublist(ct.length - tagLen));
    cipher.process(cipherText);
    return cipher.doFinal(tag, 0);
  }

  Uint8List _randomBytes(int length) {
    final rng = SecureRandom('Fortuna')
      ..seed(KeyParameter(_seedFortuna()));
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

  // ==================== Fail-closed (no canonical implementation) ==========

  /// No ZK proof backend exists. Generating a proof locally would require the
  /// proving key and circuit, which are not provisioned on this client.
  Future<String> generateZkProof(
    String circuitId,
    Map<String, dynamic> privateInputs,
    Map<String, dynamic> publicInputs,
  ) async {
    throw PrivacyException(
      'ZK proof generation is not supported: no canonical proving backend or '
      'proving key is available. Do not submit placeholder proofs.',
    );
  }

  Future<bool> verifyZkProof(
    String circuitId,
    String proof,
    Map<String, dynamic> publicInputs,
  ) async {
    throw PrivacyException(
      'ZK proof verification is not supported: no canonical verifier backend '
      'is available. Never accept unverified proofs.',
    );
  }

  /// Stealth addresses require an on-chain stealth-pool/registry contract and
  /// a scanning service. Neither exists in the canonical backend.
  Future<Map<String, dynamic>> createStealthAddress(String owner) async {
    throw PrivacyException(
      'Stealth-address generation is not supported by the canonical backend.',
    );
  }

  /// CoinJoin / mixing is not implemented. Never fabricate mixed outputs.
  Future<List<String>> coinJoin(List<String> inputs, int mixCount) async {
    throw PrivacyException(
      'CoinJoin mixing is not supported by the canonical backend.',
    );
  }
}

class PrivacyException implements Exception {
  final String message;
  PrivacyException(this.message);
  @override
  String toString() => 'PrivacyException: $message';
}
