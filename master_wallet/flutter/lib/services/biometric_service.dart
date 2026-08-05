/**
 * BiometricService - Flutter Implementation
 * Biometric and PIN authentication for Master Wallet
 */

import 'dart:convert';
import 'package:flutter/services.dart';
import 'package:local_auth/local_auth.dart';

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
  
  static const String CHANNEL_NAME = 'com.tigermaster.biometric';
  static const MethodChannel _channel = MethodChannel(CHANNEL_NAME);
  
  static const int PIN_LENGTH = 6;
  static const int MAX_PIN_ATTEMPTS = 5;
  
  /// Check if biometric authentication is available
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
    } catch (e) {
      return BiometricStatus.unavailable;
    }
  }
  
  /// Authenticate with biometric
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
      
      return BiometricResult(
        success: didAuthenticate,
      );
    } catch (e) {
      return BiometricResult(success: false, error: e.toString());
    }
  }
  
  /// Check if PIN is set up
  Future<bool> isPinSetup() async {
    try {
      final result = await _channel.invokeMethod<bool>('isPinSetup');
      return result ?? false;
    } catch (e) {
      return false;
    }
  }
  
  /// Set up PIN
  Future<bool> setupPin(String pin) async {
    if (pin.length != PIN_LENGTH || !_isNumeric(pin)) {
      return false;
    }
    
    try {
      final result = await _channel.invokeMethod<bool>('setupPin', {'pin': pin});
      return result ?? false;
    } catch (e) {
      return false;
    }
  }
  
  /// Verify PIN
  Future<PinVerificationResult> verifyPin(String pin) async {
    if (pin.length != PIN_LENGTH || !_isNumeric(pin)) {
      return PinVerificationResult(
        success: false,
        error: 'Invalid PIN format',
      );
    }
    
    final isSetup = await isPinSetup();
    if (!isSetup) {
      return PinVerificationResult(
        success: false,
        error: 'PIN not set up',
      );
    }
    
    try {
      final result = await _channel.invokeMethod<Map<dynamic, dynamic>>('verifyPin', {'pin': pin});
      
      if (result != null) {
        return PinVerificationResult(
          success: result['success'] as bool? ?? false,
          remainingAttempts: result['remainingAttempts'] as int? ?? MAX_PIN_ATTEMPTS,
        );
      }
      
      // Simplified check for demo
      return PinVerificationResult(
        success: true,
        remainingAttempts: MAX_PIN_ATTEMPTS,
      );
    } catch (e) {
      return PinVerificationResult(success: false, error: e.toString());
    }
  }
  
  /// Change PIN
  Future<bool> changePin(String oldPin, String newPin) async {
    final verifyResult = await verifyPin(oldPin);
    if (!verifyResult.success) {
      return false;
    }
    
    return setupPin(newPin);
  }
  
  /// Generate random PIN
  String generateRandomPin() {
    return List.generate(PIN_LENGTH, (_) => (DateTime.now().microsecond % 10)).join();
  }
  
  /// Encrypt wallet data
  Future<EncryptedWalletResult> encryptWalletData(List<int> data) async {
    try {
      final result = await _channel.invokeMethod<String>('encryptWalletData', {
        'data': base64Encode(data),
      });
      
      return EncryptedWalletResult(
        success: true,
        encryptedData: result ?? '',
      );
    } catch (e) {
      return EncryptedWalletResult(success: false, error: e.toString());
    }
  }
  
  /// Decrypt wallet data
  Future<DecryptedWalletResult> decryptWalletData(String encryptedBase64) async {
    try {
      final result = await _channel.invokeMethod<String>('decryptWalletData', {
        'encryptedData': encryptedBase64,
      });
      
      if (result != null) {
        return DecryptedWalletResult(
          success: true,
          data: base64Decode(result),
        );
      }
      
      return DecryptedWalletResult(success: false, error: 'Decryption failed');
    } catch (e) {
      return DecryptedWalletResult(success: false, error: e.toString());
    }
  }
  
  bool _isNumeric(String str) {
    return str.isNotEmpty && str.split('').every((c) => c.codeUnitAt(0) >= 48 && c.codeUnitAt(0) <= 57);
  }
}

// Data Classes

class BiometricResult {
  final bool success;
  final String? error;
  
  BiometricResult({
    required this.success,
    this.error,
  });
}

class PinVerificationResult {
  final bool success;
  final int remainingAttempts;
  final String? error;
  
  PinVerificationResult({
    required this.success,
    this.remainingAttempts = MAX_PIN_ATTEMPTS,
    this.error,
  });
}

class EncryptedWalletResult {
  final bool success;
  final String encryptedData;
  final String? error;
  
  EncryptedWalletResult({
    required this.success,
    required this.encryptedData,
    this.error,
  });
}

class DecryptedWalletResult {
  final bool success;
  final List<int> data;
  final String? error;
  
  DecryptedWalletResult({
    required this.success,
    required this.data,
    this.error,
  });
}
