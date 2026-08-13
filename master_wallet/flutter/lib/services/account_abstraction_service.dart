/**
 * AccountAbstractionService - Flutter Implementation
 *
 * ERC-4337 operations are performed by the canonical backend (verifying
 * paymaster + entrypoint relayer). This client is a thin REST wrapper and
 * NEVER fabricates signatures, addresses, hashes, or gas values. Any operation
 * the backend does not expose is fail-closed (throws) rather than simulated.
 */

class AccountAbstractionService {
  static AccountAbstractionService? _instance;
  static AccountAbstractionService get instance {
    _instance ??= AccountAbstractionService._();
    return _instance!;
  }

  AccountAbstractionService._();

  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );

  // Canonical EIP-4337 EntryPoint address (mainnet, shared by all chains).
  static const String entryPointAddress =
      '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3';

  String? _token;
  void setToken(String? token) => _token = token;

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  // ==================== Backend operations ====================

  // The canonical backend contract (:8450) exposes NO EIP-4337 / account-
  // abstraction endpoints. Every AA operation therefore fails closed rather
  // than calling a non-existent route or fabricating signatures/addresses.
  Exception _unsupported(String op) => AccountAbstractionException(
        'account-abstraction $op is not supported by the canonical backend '
        'contract. The backend exposes no /aa bundler, account-registry, or '
        'verifying-paymaster endpoint. Submit transactions via the canonical '
        'master-wallet sign route instead.');

  /// Submit a UserOperation to the backend bundler/relayer. NOT supported by
  /// the canonical backend; fails closed. The client never signs locally.
  Future<Map<String, dynamic>> submitUserOperation(
    UserOperation userOp, {
    int? chainId,
    String? beneficiary,
  }) async {
    throw _unsupported('submitUserOperation');
  }

  /// Fetch a smart account by owner. NOT supported by the canonical backend.
  Future<SmartAccount> getAccount(String owner, {int? chainId}) async {
    throw _unsupported('getAccount');
  }

  // ==================== Fail-closed (no canonical backend endpoint) ==========

  /// Local ECDSA signing of a UserOperation is NOT supported. The private key
  /// for the smart-account owner never lives on this client. Submit unsigned
  /// UserOperations via [submitUserOperation] and the backend signs/broadcasts.
  Future<UserOperation> signUserOperation(
    UserOperation userOp,
    String privateKey,
    int chainId,
  ) async {
    throw AccountAbstractionException(
      'Local UserOperation signing is not supported: submit the unsigned '
      'UserOperation via submitUserOperation() so the backend signs and '
      'broadcasts it with the provisioned owner key.',
    );
  }

  /// Deploying a smart account requires an on-chain transaction handled by the
  /// backend factory relayer. There is no client-side deployment.
  Future<Map<String, dynamic>> deployAccount(
    String owner, [
    String? salt,
    String? factoryAddress,
  ]) async {
    throw AccountAbstractionException(
      'Client-side smart-account deployment is not supported. Use the '
      'backend account-factory endpoint to deploy a counterfactual account.',
    );
  }
}

class AccountAbstractionException implements Exception {
  final String message;
  AccountAbstractionException(this.message);
  @override
  String toString() => 'AccountAbstractionException: $message';
}

class UserOperation {
  final String sender;
  final String nonce;
  final String initCode;
  final String callData;
  final String callGasLimit;
  final String verificationGasLimit;
  final String preVerificationGas;
  final String maxFeePerGas;
  final String maxPriorityFeePerGas;
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

  factory UserOperation.fromJson(Map<String, dynamic> json) => UserOperation(
        sender: json['sender'] as String? ?? '',
        nonce: json['nonce']?.toString() ?? '0',
        initCode: json['initCode'] as String? ?? '0x',
        callData: json['callData'] as String? ?? '0x',
        callGasLimit: json['callGasLimit']?.toString() ?? '0',
        verificationGasLimit: json['verificationGasLimit']?.toString() ?? '0',
        preVerificationGas: json['preVerificationGas']?.toString() ?? '0',
        maxFeePerGas: json['maxFeePerGas']?.toString() ?? '0',
        maxPriorityFeePerGas: json['maxPriorityFeePerGas']?.toString() ?? '0',
        paymasterAndData: json['paymasterAndData'] as String? ?? '0x',
        signature: json['signature'] as String? ?? '0x',
      );

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

class SmartAccount {
  final String address;
  final String owner;
  final bool isDeployed;
  final String nonce;

  SmartAccount({
    required this.address,
    required this.owner,
    required this.isDeployed,
    required this.nonce,
  });

  factory SmartAccount.fromJson(Map<String, dynamic> json) => SmartAccount(
        address: json['address'] as String? ?? '',
        owner: json['owner'] as String? ?? '',
        isDeployed: json['isDeployed'] as bool? ?? false,
        nonce: json['nonce']?.toString() ?? '0',
      );
}
