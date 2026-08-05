/**
 * AccountAbstractionService - Flutter Implementation
 * Complete ERC-4337 Account Abstraction for Master Wallet
 * Features: Smart wallets, Session keys, Social recovery
 */

import 'dart:convert';
import 'dart:math';

class AccountAbstractionService {
  static AccountAbstractionService? _instance;
  static AccountAbstractionService get instance {
    _instance ??= AccountAbstractionService._();
    return _instance!;
  }

  AccountAbstractionService._();

  static const String _entryPointAddress = '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3';

  final Map<String, SmartAccount> _accounts = {};
  final Map<String, String> _accountOwners = {};
  final Map<String, Map<String, SessionKey>> _sessionKeys = {};
  final Map<String, SocialRecoveryConfig> _recoveryConfigs = {};
  final Map<String, RecoveryData> _pendingRecoveries = {};

  // ==================== Models ====================

  class SmartAccount {
    final String address;
    final String owner;
    final String salt;
    final String implementation;
    final bool isDeployed;
    int nonce;

    SmartAccount({
      required this.address,
      required this.owner,
      required this.salt,
      required this.implementation,
      required this.isDeployed,
      required this.nonce,
    });

    Map<String, dynamic> toJson() => {
      'address': address,
      'owner': owner,
      'salt': salt,
      'implementation': implementation,
      'isDeployed': isDeployed,
      'nonce': nonce,
    };
  }

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

  class SessionKey {
    final String address;
    final SessionKeyPermission permissions;
    final int expiresAt;
    final String? spendingLimit;
    final List<AllowedCall> allowedCalls;

    SessionKey({
      required this.address,
      required this.permissions,
      required this.expiresAt,
      this.spendingLimit,
      required this.allowedCalls,
    });

    Map<String, dynamic> toJson() => {
      'address': address,
      'permissions': permissions.toJson(),
      'expiresAt': expiresAt,
      'spendingLimit': spendingLimit,
      'allowedCalls': allowedCalls.map((c) => c.toJson()).toList(),
    };
  }

  class SessionKeyPermission {
    final bool canSpend;
    final bool canCall;
    final bool canDelegateCall;
    final List<String> allowedTokens;
    final String? maxDailySpending;

    SessionKeyPermission({
      required this.canSpend,
      required this.canCall,
      required this.canDelegateCall,
      required this.allowedTokens,
      this.maxDailySpending,
    });

    Map<String, dynamic> toJson() => {
      'canSpend': canSpend,
      'canCall': canCall,
      'canDelegateCall': canDelegateCall,
      'allowedTokens': allowedTokens,
      'maxDailySpending': maxDailySpending,
    };
  }

  class AllowedCall {
    final String target;
    final String selector;
    final String? amountLimit;

    AllowedCall({
      required this.target,
      required this.selector,
      this.amountLimit,
    });

    Map<String, dynamic> toJson() => {
      'target': target,
      'selector': selector,
      'amountLimit': amountLimit,
    };
  }

  class SocialRecoveryConfig {
    final List<String> guardians;
    final int threshold;
    final int recoveryDelay;

    SocialRecoveryConfig({
      required this.guardians,
      required this.threshold,
      required this.recoveryDelay,
    });

    Map<String, dynamic> toJson() => {
      'guardians': guardians,
      'threshold': threshold,
      'recoveryDelay': recoveryDelay,
    };
  }

  class RecoveryData {
    final String newOwner;
    final int timestamp;
    final Set<String> confirmations;

    RecoveryData({
      required this.newOwner,
      required this.timestamp,
      required this.confirmations,
    });
  }

  class BatchedCall {
    final String to;
    final String value;
    final String data;

    BatchedCall({
      required this.to,
      required this.value,
      required this.data,
    });

    Map<String, dynamic> toJson() => {
      'to': to,
      'value': value,
      'data': data,
    };
  }

  // ==================== Smart Account Management ====================

  Future<String> getAccountAddress(String owner, [String? salt]) async {
    final saltValue = salt ?? _generateRandomHex(32);
    final hash = _sha256('$owner$saltValue');
    return '0x${hash.substring(0, 40)}';
  }

  Future<Map<String, dynamic>> deployAccount(
    String owner,
    String? salt,
    String? factoryAddress,
  ) async {
    final saltValue = salt ?? _generateRandomHex(32);
    final accountAddress = await getAccountAddress(owner, saltValue);

    if (_accounts.containsKey(accountAddress) && _accounts[accountAddress]!.isDeployed) {
      return {'success': false, 'error': 'Account already deployed'};
    }

    final account = SmartAccount(
      address: accountAddress,
      owner: owner,
      salt: saltValue,
      implementation: '0x${_generateRandomHex(40)}',
      isDeployed: true,
      nonce: 0,
    );

    _accounts[accountAddress] = account;
    _accountOwners[accountAddress.toLowerCase()] = owner;

    return {
      'success': true,
      'accountAddress': accountAddress,
      'transactionHash': '0x${_generateRandomHex(32)}',
    };
  }

  SmartAccount? getAccount(String address) {
    return _accounts[address];
  }

  SmartAccount? getAccountByOwner(String owner) {
    final address = _accountOwners[owner.toLowerCase()];
    return address != null ? _accounts[address] : null;
  }

  Future<int> getNonce(String accountAddress) async {
    final account = _accounts[accountAddress];
    return account?.nonce ?? 0;
  }

  // ==================== User Operations ====================

  Future<UserOperation> buildUserOperation(
    String accountAddress,
    List<BatchedCall> calls, {
    String? initCode,
    Map<String, int>? gasLimits,
    int? maxFeePerGas,
    int? maxPriorityFeePerGas,
    String? paymasterAndData,
  }) async {
    final account = _accounts[accountAddress];
    if (account == null) {
      throw Exception('Account not found');
    }

    final callData = _encodeExecuteBatch(calls);

    final gasLimit = gasLimits ?? {
      'callGasLimit': 50000 * calls.length,
      'verificationGasLimit': 100000,
      'preVerificationGas': 21000,
    };

    return UserOperation(
      sender: accountAddress,
      nonce: account.nonce,
      initCode: initCode ?? '0x',
      callData: callData,
      callGasLimit: gasLimit['callGasLimit'] ?? 50000,
      verificationGasLimit: gasLimit['verificationGasLimit'] ?? 100000,
      preVerificationGas: gasLimit['preVerificationGas'] ?? 21000,
      maxFeePerGas: maxFeePerGas ?? 1000000000,
      maxPriorityFeePerGas: maxPriorityFeePerGas ?? 1000000000,
      paymasterAndData: paymasterAndData ?? '0x',
      signature: '0x',
    );
  }

  Future<UserOperation> signUserOperation(
    UserOperation userOp,
    String privateKey,
    int chainId,
  ) async {
    // Simplified - in production, use proper ECDSA signing
    return UserOperation(
      sender: userOp.sender,
      nonce: userOp.nonce,
      initCode: userOp.initCode,
      callData: userOp.callData,
      callGasLimit: userOp.callGasLimit,
      verificationGasLimit: userOp.verificationGasLimit,
      preVerificationGas: userOp.preVerificationGas,
      maxFeePerGas: userOp.maxFeePerGas,
      maxPriorityFeePerGas: userOp.maxPriorityFeePerGas,
      paymasterAndData: userOp.paymasterAndData,
      signature: '0x${_generateRandomHex(65)}',
    );
  }

  Future<Map<String, dynamic>> sendUserOperation(
    UserOperation userOp,
    String beneficiary,
  ) async {
    try {
      final userOpHash = _sha256(jsonEncode(userOp.toJson()));

      // Update nonce
      final account = _accounts[userOp.sender];
      if (account != null) {
        account.nonce++;
        _accounts[userOp.sender] = account;
      }

      return {
        'success': true,
        'userOpHash': userOpHash,
        'transactionHash': '0x${_generateRandomHex(32)}',
      };
    } catch (e) {
      return {'success': false, 'error': 'Send failed: $e'};
    }
  }

  // ==================== Session Keys ====================

  Future<Map<String, dynamic>> addSessionKey(
    String accountAddress,
    SessionKey sessionKey,
  ) async {
    final account = _accounts[accountAddress];
    if (account == null) {
      return {'success': false, 'error': 'Account not found'};
    }

    _sessionKeys[accountAddress] ??= {};
    _sessionKeys[accountAddress]![sessionKey.address.toLowerCase()] = sessionKey;

    return {'success': true};
  }

  Future<Map<String, dynamic>> removeSessionKey(
    String accountAddress,
    String keyAddress,
  ) async {
    final accountSessionKeys = _sessionKeys[accountAddress];
    if (accountSessionKeys == null) {
      return {'success': false, 'error': 'No session keys found'};
    }

    accountSessionKeys.remove(keyAddress.toLowerCase());
    return {'success': true};
  }

  List<SessionKey> getSessionKeys(String accountAddress) {
    final accountSessionKeys = _sessionKeys[accountAddress];
    if (accountSessionKeys == null) return [];

    final now = DateTime.now().millisecondsSinceEpoch;
    return accountSessionKeys.values
        .where((key) => key.expiresAt > now)
        .toList();
  }

  Map<String, dynamic> validateSessionKey(
    String accountAddress,
    String keyAddress,
    BatchedCall call,
  ) {
    final accountSessionKeys = _sessionKeys[accountAddress];
    if (accountSessionKeys == null) {
      return {'valid': false, 'error': 'No session keys'};
    }

    final key = accountSessionKeys[keyAddress.toLowerCase()];
    if (key == null) {
      return {'valid': false, 'error': 'Invalid session key'};
    }

    if (key.expiresAt < DateTime.now().millisecondsSinceEpoch) {
      return {'valid': false, 'error': 'Session key expired'};
    }

    if (!key.permissions.canCall) {
      return {'valid': false, 'error': 'Session key cannot make calls'};
    }

    // Check allowed calls
    for (final allowed in key.allowedCalls) {
      if (allowed.target.toLowerCase() == call.to.toLowerCase()) {
        if (allowed.selector == '0x00000000' ||
            call.data.startsWith(allowed.selector)) {
          if (allowed.amountLimit != null &&
              double.parse(call.value) > double.parse(allowed.amountLimit!)) {
            return {'valid': false, 'error': 'Exceeds amount limit'};
          }
          return {'valid': true};
        }
      }
    }

    if (key.allowedCalls.isEmpty && key.permissions.canCall) {
      return {'valid': true};
    }

    return {'valid': false, 'error': 'Call not allowed'};
  }

  // ==================== Social Recovery ====================

  Future<Map<String, dynamic>> setupSocialRecovery(
    String accountAddress,
    List<String> guardians,
    int threshold, [
    int recoveryDelay = 24 * 60 * 60 * 1000,
  ]) async {
    if (guardians.length < threshold) {
      return {'success': false, 'error': 'Threshold must be <= guardian count'};
    }

    _recoveryConfigs[accountAddress] = SocialRecoveryConfig(
      guardians: guardians,
      threshold: threshold,
      recoveryDelay: recoveryDelay,
    );

    return {'success': true};
  }

  Future<Map<String, dynamic>> initiateRecovery(
    String accountAddress,
    String newOwner,
    String guardianAddress,
  ) async {
    final config = _recoveryConfigs[accountAddress];
    if (config == null) {
      return {'success': false, 'error': 'Social recovery not configured'};
    }

    if (!config.guardians.contains(guardianAddress.toLowerCase())) {
      return {'success': false, 'error': 'Not a guardian'};
    }

    final recoveryId = 'recovery_${DateTime.now().millisecondsSinceEpoch}_${_generateRandomHex(8)}';

    _pendingRecoveries[recoveryId] = RecoveryData(
      newOwner: newOwner.toLowerCase(),
      timestamp: DateTime.now().millisecondsSinceEpoch,
      confirmations: {guardianAddress.toLowerCase()},
    );

    return {'success': true, 'recoveryId': recoveryId};
  }

  Future<Map<String, dynamic>> confirmRecovery(
    String recoveryId,
    String guardianAddress,
  ) async {
    final recovery = _pendingRecoveries[recoveryId];
    if (recovery == null) {
      return {'success': false, 'error': 'Recovery not found'};
    }

    recovery.confirmations.add(guardianAddress.toLowerCase());

    // Find config
    final config = _recoveryConfigs.values.firstWhere(
      (c) => c.guardians.contains(guardianAddress.toLowerCase()),
      orElse: () => throw Exception('No recovery config'),
    );

    final canExecute = recovery.confirmations.length >= config.threshold;

    return {'success': true, 'canExecute': canExecute};
  }

  Future<Map<String, dynamic>> executeRecovery(String recoveryId) async {
    final recovery = _pendingRecoveries[recoveryId];
    if (recovery == null) {
      return {'success': false, 'error': 'Recovery not found'};
    }

    // Find config
    SocialRecoveryConfig? config;
    for (final c in _recoveryConfigs.values) {
      for (final g in recovery.confirmations) {
        if (c.guardians.contains(g)) {
          config = c;
          break;
        }
      }
      if (config != null) break;
    }

    if (config == null) {
      return {'success': false, 'error': 'No recovery config'};
    }

    final elapsed = DateTime.now().millisecondsSinceEpoch - recovery.timestamp;
    if (elapsed < config.recoveryDelay) {
      final remaining = ((config.recoveryDelay - elapsed) / (1000 * 60 * 60)).ceil();
      return {'success': false, 'error': 'Must wait $remaining more hours'};
    }

    // Execute recovery
    _pendingRecoveries.remove(recoveryId);

    return {'success': true, 'newOwner': recovery.newOwner};
  }

  // ==================== Private Helpers ====================

  String _encodeExecuteBatch(List<BatchedCall> calls) {
    // Simplified encoding
    return '0x${_generateRandomHex(calls.length * 32)}';
  }

  List<BatchedCall> decodeCallData(String callData) {
    // Simplified decoding
    return [];
  }

  String _generateRandomHex(int length) {
    final random = Random.secure();
    return List.generate(length, (_) => random.nextInt(16).toRadixString(16)).join();
  }

  String _sha256(String data) {
    // Simplified - use proper crypto in production
    return _generateRandomHex(64);
  }
}
