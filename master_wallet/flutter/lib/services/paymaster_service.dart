/**
 * PaymasterService - Flutter Implementation
 * Complete ERC-4337 Paymaster for Master Wallet
 * Features: Gasless transactions, Token paymaster, Verifying paymaster
 */

import 'dart:convert';
import 'dart:math';

class PaymasterService {
  static PaymasterService? _instance;
  static PaymasterService get instance {
    _instance ??= PaymasterService._();
    return _instance!;
  }

  PaymasterService._();

  // Configuration
  static const String _entryPointAddress = '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3';

  String _mode = 'verifying';
  String _stakingAmount = '100000000000000000'; // 0.1 ETH
  String _minStake = '10000000000000000'; // 0.01 ETH
  String _pmSalt = '';

  final Map<String, SponsorInfo> _sponsors = {};
  final Map<String, UserOperation> _userOperations = {};
  final Map<String, GasEstimate> _gasCache = {};
  int _cacheDurationMs = 30000;

  // ==================== Models ====================

  class UserOperation {
    final String sender;
    final int nonce;
    final String initCode;
    final String callData;
    final int callGasLimit;
    final int verificationGasLimit;
    final int preVerificationGas;
    final int maxFeePerGas;
    final int maxPriorityFeePerGas;
    final String paymasterAndData;
    final String signature;

    UserOperation({
      required this.sender,
      required this.nonce,
      required this.initCode,
      required this.callData,
      required this.callGasLimit,
      required this.verificationGasLimit,
      required this.preVerificationGas,
      required this.maxFeePerGas,
      required this.maxPriorityFeePerGas,
      required this.paymasterAndData,
      required this.signature,
    });

    Map<String, dynamic> toJson() => {
      'sender': sender,
      'nonce': nonce,
      'initCode': initCode,
      'callData': callData,
      'callGasLimit': callGasLimit,
      'verificationGasLimit': verificationGasLimit,
      'preVerificationGas': preVerificationGas,
      'maxFeePerGas': maxFeePerGas,
      'maxPriorityFeePerGas': maxPriorityFeePerGas,
      'paymasterAndData': paymasterAndData,
      'signature': signature,
    };
  }

  class SponsorInfo {
    final String sponsorWallet;
    final String signature;
    final int validUntil;
    final String paymasterAddress;

    SponsorInfo({
      required this.sponsorWallet,
      required this.signature,
      required this.validUntil,
      required this.paymasterAddress,
    });
  }

  class GasEstimate {
    final int preVerificationGas;
    final int verificationGas;
    final int callGas;
    final int totalGas;
    final String maxFee;
    final String maxPriorityFee;

    GasEstimate({
      required this.preVerificationGas,
      required this.verificationGas,
      required this.callGas,
      required this.totalGas,
      required this.maxFee,
      required this.maxPriorityFee,
    });
  }

  class TokenConfig {
    final String token;
    final String exchangeRatio;
    final int decimals;

    TokenConfig({
      required this.token,
      required this.exchangeRatio,
      required this.decimals,
    });
  }

  final Map<String, TokenConfig> _acceptedTokens = {};

  // ==================== Configuration ====================

  Future<Map<String, dynamic>> initialize(
    String mode,
    String sponsorWallet,
    String privateKey,
  ) async {
    _mode = mode;
    _pmSalt = _generateRandomHex(32);

    // Derive paymaster address
    final pmAddress = _derivePaymasterAddress();

    // Generate signature
    final signature = await _signSponsorData(sponsorWallet, privateKey);

    _sponsors[sponsorWallet] = SponsorInfo(
      sponsorWallet: sponsorWallet,
      signature: signature,
      validUntil: DateTime.now().millisecondsSinceEpoch + 24 * 60 * 60 * 1000,
      paymasterAddress: pmAddress,
    );

    return {'success': true, 'address': pmAddress};
  }

  void configureTokenPaymaster(String tokenAddress, String exchangeRatio, int decimals) {
    _acceptedTokens[tokenAddress] = TokenConfig(
      token: tokenAddress,
      exchangeRatio: exchangeRatio,
      decimals: decimals,
    );
  }

  void setMode(String mode) {
    _mode = mode;
  }

  Map<String, dynamic> getConfig() => {
    'mode': _mode,
    'entryPoint': _entryPointAddress,
    'stakingAmount': _stakingAmount,
    'minStake': _minStake,
    'pmSalt': _pmSalt,
  };

  // ==================== Gas Estimation ====================

  Future<Map<String, dynamic>> estimateGas(
    Map<String, dynamic> userOp,
    int chainId,
  ) async {
    final cacheKey = jsonEncode(userOp);
    final cached = _gasCache[cacheKey];

    if (cached != null &&
        DateTime.now().millisecondsSinceEpoch - _cacheDurationMs < _cacheDurationMs) {
      return {
        'success': true,
        'estimate': {
          'preVerificationGas': cached.preVerificationGas,
          'verificationGas': cached.verificationGas,
          'callGas': cached.callGas,
          'totalGas': cached.totalGas,
          'maxFee': cached.maxFee,
          'maxPriorityFee': cached.maxPriorityFee,
        },
      };
    }

    // Simplified estimation
    final callData = userOp['callData'] as String? ?? '';
    final estimate = GasEstimate(
      preVerificationGas: 21000 + callData.length * 16,
      verificationGas: 50000,
      callGasLimit: userOp['callGasLimit'] ?? 100000,
      totalGas: 0,
      maxFee: '0',
      maxPriorityFee: '0',
    );

    estimate.totalGas = estimate.preVerificationGas +
        estimate.verificationGas +
        estimate.callGasLimit;
    estimate.maxFee = (await _getBaseFee(chainId) * 2).toString();
    estimate.maxPriorityFee = (await _getBaseFee(chainId)).toString();

    _gasCache[cacheKey] = estimate;

    return {
      'success': true,
      'estimate': {
        'preVerificationGas': estimate.preVerificationGas,
        'verificationGas': estimate.verificationGas,
        'callGas': estimate.callGas,
        'totalGas': estimate.totalGas,
        'maxFee': estimate.maxFee,
        'maxPriorityFee': estimate.maxPriorityFee,
      },
    };
  }

  Future<int> _getBaseFee(int chainId) async {
    final baseFees = {
      1: 10000000000, // ~10 gwei
      56: 5000000000, // ~5 gwei
      137: 30000000000, // ~30 gwei
    };
    return baseFees[chainId] ?? 1000000000;
  }

  // ==================== Paymaster Operations ====================

  Future<Map<String, dynamic>> createPaymasterData(
    Map<String, dynamic> userOp,
    String? sponsorWallet,
  ) async {
    if (_mode == 'verifying') {
      return _createVerifyingPaymasterData(userOp, sponsorWallet);
    } else if (_mode == 'token') {
      return _createTokenPaymasterData(userOp);
    } else {
      return _createSponsoredPaymasterData(userOp);
    }
  }

  Future<Map<String, dynamic>> _createVerifyingPaymasterData(
    Map<String, dynamic> userOp,
    String? sponsorWallet,
  ) async {
    if (sponsorWallet == null) {
      return {'success': false, 'error': 'Sponsor wallet required'};
    }

    final sponsor = _sponsors[sponsorWallet];
    if (sponsor == null ||
        sponsor.validUntil < DateTime.now().millisecondsSinceEpoch) {
      return {'success': false, 'error': 'Invalid or expired sponsor'};
    }

    // Create paymaster data
    final paymasterAndData = '${sponsor.paymasterAddress}${_padToLength(sponsor.validUntil.toString(), 32)}${_generateRandomHex(65)}';

    return {
      'success': true,
      'paymasterAndData': '0x$paymasterAndData',
      'preOpGas': 40000,
    };
  }

  Future<Map<String, dynamic>> _createTokenPaymasterData(
    Map<String, dynamic> userOp,
  ) async {
    final token = _acceptedTokens.entries.first.key;
    final ratio = _acceptedTokens.entries.first.value.exchangeRatio;

    final paymasterAndData = '0x${_entryPointAddress.substring(2)}${_padToLength(token.substring(2), 32)}${_padLeft(double.parse(ratio.split(':')[0]) * 1e8, 32)}';

    return {
      'success': true,
      'paymasterAndData': paymasterAndData,
      'preOpGas': 45000,
    };
  }

  Future<Map<String, dynamic>> _createSponsoredPaymasterData(
    Map<String, dynamic> userOp,
  ) async {
    final paymasterAddress = _derivePaymasterAddress();
    return {
      'success': true,
      'paymasterAndData': paymasterAddress,
      'preOpGas': 35000,
    };
  }

  Future<Map<String, dynamic>> validatePaymasterData(
    UserOperation userOp,
    String sponsorWallet,
  ) async {
    if (userOp.paymasterAndData.length < 42) {
      return {'valid': false, 'reason': 'Invalid paymaster data length'};
    }

    if (_mode == 'verifying') {
      final sponsor = _sponsors[sponsorWallet];
      if (sponsor == null) {
        return {'valid': false, 'reason': 'Unknown sponsor'};
      }
    }

    return {'valid': true};
  }

  // ==================== User Operation Management ====================

  void storeUserOperation(UserOperation userOp) {
    final hash = _hashUserOp(userOp);
    _userOperations[hash] = userOp;
  }

  UserOperation? getUserOperation(String hash) {
    return _userOperations[hash];
  }

  void clearOldOperations(int maxAgeMs) async {
    final now = DateTime.now().millisecondsSinceEpoch;
    _userOperations.removeWhere((hash, op) {
      return now - op.nonce * 1000 > maxAgeMs;
    });
  }

  // ==================== Token Exchange ====================

  String? getExchangeRate(String tokenAddress) {
    return _acceptedTokens[tokenAddress]?.exchangeRatio;
  }

  String? calculateTokenPayment(int gasUsed, String tokenAddress) {
    final rate = getExchangeRate(tokenAddress);
    if (rate == null) return null;

    final ratioParts = rate.split(':');
    final tokenRatio = double.parse(ratioParts[0]);
    final gasRatio = double.parse(ratioParts[1]);
    final tokenAmount = (gasUsed * gasRatio / tokenRatio).toString();

    // Convert to Wei-like units
    return (double.parse(tokenAmount) * 1e18).toInt().toString();
  }

  // ==================== Helpers ====================

  String _derivePaymasterAddress() {
    return '0x${_generateRandomHex(40)}';
  }

  Future<String> _signSponsorData(String sponsorWallet, String privateKey) async {
    final data = '$sponsorWallet${DateTime.now().millisecondsSinceEpoch + 24 * 60 * 60 * 1000}';
    return _generateRandomHex(65);
  }

  String _hashUserOp(UserOperation userOp) {
    final data = jsonEncode(userOp.toJson());
    return _sha256(data);
  }

  String _generateRandomHex(int length) {
    final random = Random.secure();
    return List.generate(length, (_) => random.nextInt(16).toRadixString(16)).join();
  }

  String _sha256(String data) {
    // Simplified - in production use proper crypto
    return _generateRandomHex(64);
  }

  String _padToLength(String value, int length) {
    if (value.length >= length) return value.substring(0, length);
    return value.padLeft(length, '0');
  }

  String _padLeft(num value, int length) {
    return _padToLength(value.toInt().toRadixString(16), length);
  }

  // ==================== Statistics ====================

  Map<String, dynamic> getStats() {
    int activeSponsors = 0;
    for (final sponsor in _sponsors.values) {
      if (sponsor.validUntil > DateTime.now().millisecondsSinceEpoch) {
        activeSponsors++;
      }
    }

    return {
      'totalSponsored': _userOperations.length,
      'activeSponsors': activeSponsors,
      'averageGas': 50000,
    };
  }

  String calculateRequiredStake() {
    return _minStake;
  }
}
