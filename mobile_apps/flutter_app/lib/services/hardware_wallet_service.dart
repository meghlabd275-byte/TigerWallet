import 'dart:typed_data';
import 'package:flutter/services.dart';

/// Hardware Wallet Service for Flutter App
/// Ledger and Trezor support
class HardwareWalletService {
  static final HardwareWalletService _instance = HardwareWalletService._internal();
  factory HardwareWalletService() => _instance;
  HardwareWalletService._internal();

  static const MethodChannel _channel = MethodChannel('tigerwallet/hardware');

  bool _isConnected = false;
  String? _deviceType;
  String? _deviceId;

  bool get isConnected => _isConnected;
  String? get deviceType => _deviceType;
  String? get deviceId => _deviceId;

  /// Connect to hardware wallet
  Future<bool> connect(String deviceType) async {
    try {
      final result = await _channel.invokeMethod<Map>('connect', {
        'deviceType': deviceType,
      });

      if (result != null && result['success'] == true) {
        _isConnected = true;
        _deviceType = deviceType;
        _deviceId = result['deviceId'];
        return true;
      }
      return false;
    } catch (e) {
      print('Failed to connect to hardware wallet: $e');
      return false;
    }
  }

  /// Disconnect from hardware wallet
  Future<void> disconnect() async {
    try {
      await _channel.invokeMethod('disconnect');
      _isConnected = false;
      _deviceType = null;
      _deviceId = null;
    } catch (e) {
      print('Failed to disconnect: $e');
    }
  }

  /// Get device information
  Future<Map<String, dynamic>?> getDeviceInfo() async {
    try {
      final result = await _channel.invokeMethod<Map>('getDeviceInfo');
      return result?.cast<String, dynamic>();
    } catch (e) {
      print('Failed to get device info: $e');
      return null;
    }
  }

  /// Get address for derivation path
  Future<String?> getAddress(String derivationPath) async {
    try {
      final result = await _channel.invokeMethod<String>('getAddress', {
        'derivationPath': derivationPath,
      });
      return result;
    } catch (e) {
      print('Failed to get address: $e');
      return null;
    }
  }

  /// Sign transaction
  Future<String?> signTransaction({
    required String derivationPath,
    required Uint8List transaction,
  }) async {
    try {
      final result = await _channel.invokeMethod<String>('signTransaction', {
        'derivationPath': derivationPath,
        'transaction': transaction,
      });
      return result;
    } catch (e) {
      print('Failed to sign transaction: $e');
      return null;
    }
  }

  /// Sign message
  Future<String?> signMessage({
    required String derivationPath,
    required String message,
  }) async {
    try {
      final result = await _channel.invokeMethod<String>('signMessage', {
        'derivationPath': derivationPath,
        'message': message,
      });
      return result;
    } catch (e) {
      print('Failed to sign message: $e');
      return null;
    }
  }

  /// Sign typed data (EIP-712)
  Future<String?> signTypedData({
    required String derivationPath,
    required String typedDataJson,
  }) async {
    try {
      final result = await _channel.invokeMethod<String>('signTypedData', {
        'derivationPath': derivationPath,
        'typedDataJson': typedDataJson,
      });
      return result;
    } catch (e) {
      print('Failed to sign typed data: $e');
      return null;
    }
  }

  /// Get public key
  Future<Uint8List?> getPublicKey(String derivationPath) async {
    try {
      final result = await _channel.invokeMethod<Uint8List>('getPublicKey', {
        'derivationPath': derivationPath,
      });
      return result;
    } catch (e) {
      print('Failed to get public key: $e');
      return null;
    }
  }

  /// Get supported derivation paths
  List<String> getSupportedDerivationPaths() {
    return [
      "m/44'/60'/0'/0/0",   // Ledger Live (Ethereum)
      "m/44'/60'/0'/0",     // Legacy (Ethereum)
      "m/44'/60'/0'/0/0",   // MetaMask (Ethereum)
      "m/44'/501'/0'/0'",    // Solana
      "m/44'/195'/0'/0'",   // Tron
    ];
  }

  /// Validate derivation path
  bool isValidDerivationPath(String path) {
    // Basic validation for BIP44 paths
    final regex = RegExp(r"^m/44'/\d+'/(\d+)'/(\d+)/(\d+)$");
    return regex.hasMatch(path);
  }
}

/// Supported hardware wallet types
enum HardwareWalletType {
  ledger,
  trezor,
  gridPlus,
}

extension HardwareWalletTypeExtension on HardwareWalletType {
  String get displayName {
    switch (this) {
      case HardwareWalletType.ledger:
        return 'Ledger';
      case HardwareWalletType.trezor:
        return 'Trezor';
      case HardwareWalletType.gridPlus:
        return 'GridPlus';
    }
  }

  String get id {
    switch (this) {
      case HardwareWalletType.ledger:
        return 'ledger';
      case HardwareWalletType.trezor:
        return 'trezor';
      case HardwareWalletType.gridPlus:
        return 'gridplus';
    }
  }
}
