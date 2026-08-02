///
/// Account Abstraction Service - Flutter Implementation
/// Identical across ALL platforms
///

class AccountAbstractionService {
  static final AccountAbstractionService _instance = AccountAbstractionService._internal();
  factory AccountAbstractionService() => _instance;
  AccountAbstractionService._internal();

  static const String entryPointAddress = '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3';
  
  Map<String, SessionKey> _sessionKeys = {};
  SmartAccount? _smartAccount;

  SmartAccount initialize(String ownerAddress) {
    _smartAccount = SmartAccount(
      address: _deriveSmartAccountAddress(ownerAddress),
      owner: ownerAddress,
      nonce: 0,
      isDeployed: false,
      entryPoint: entryPointAddress,
    );
    return _smartAccount!;
  }

  String getAccountAddress() => _smartAccount?.address ?? '';

  Future<String> sendUserOp({
    required String to,
    required String value,
    required List<int> data,
    bool paymaster = true,
  }) async {
    final userOp = _createUserOperation(to, value, data, paymaster);
    final hash = _hashUserOperation(userOp);
    return '0x$hash${DateTime.now().millisecondsSinceEpoch}';
  }

  SessionKey createSessionKey({
    required String dAppAddress,
    required int validUntil,
    required List<String> allowedContracts,
    required List<String> allowedSelectors,
    required String spendingLimit,
  }) {
    final key = SessionKey(
      keyAddress: _generateKeyAddress(),
      dAppAddress: dAppAddress,
      validUntil: validUntil,
      allowedContracts: allowedContracts,
      allowedSelectors: allowedSelectors,
      spendingLimit: spendingLimit,
      spentAmount: '0',
      isRevoked: false,
    );
    _sessionKeys[key.keyAddress] = key;
    return key;
  }

  bool revokeSessionKey(String keyAddress) {
    _sessionKeys[keyAddress]?.isRevoked = true;
    return true;
  }

  List<SessionKey> getActiveSessionKeys() {
    final now = DateTime.now().millisecondsSinceEpoch;
    return _sessionKeys.values
        .where((k) => !k.isRevoked && k.validUntil > now)
        .toList();
  }

  Future<String> executeWithSessionKey({
    required String keyAddress,
    required String to,
    required List<int> data,
  }) async {
    final key = _sessionKeys[keyAddress];
    if (key == null) throw Exception('Session key not found');
    if (key.isRevoked) throw Exception('Session key revoked');
    if (DateTime.now().millisecondsSinceEpoch > key.validUntil) {
      throw Exception('Session key expired');
    }
    return '0x${_hash('$to${data.join()}')}';
  }

  String _deriveSmartAccountAddress(String owner) {
    final hash = _hash('$owner.smart_account');
    return '0x${hash.substring(0, 40)}';
  }

  String _generateKeyAddress() {
    final bytes = List.generate(32, (_) => DateTime.now().microsecond % 256);
    final hash = _hash(bytes.join());
    return '0x${hash.substring(0, 40)}';
  }

  UserOperation _createUserOperation(String to, String value, List<int> data, bool paymaster) {
    return UserOperation(
      sender: _smartAccount?.address ?? '',
      nonce: _smartAccount?.nonce.toString() ?? '0',
      initCode: _smartAccount?.isDeployed == false ? '0x' : '0x',
      callData: _encodeCallData(to, value, data),
      callGasLimit: '0x5208',
      verificationGasLimit: '0x186A0',
      preVerificationGas: '0x5208',
      maxFeePerGas: '0x3B9ACA00',
      maxPriorityFeePerGas: '0x3B9ACA00',
      paymasterAndData: paymaster ? '0xPaymasterAddress' : '0x',
      signature: '0x',
    );
  }

  String _encodeCallData(String to, String value, List<int> data) {
    return '0x${to.replaceAll('0x', '')}${value.padLeft(64, '0')}${data.length.toRadixString(16).padLeft(64, '0')}${data.map((e) => e.toRadixString(16).padLeft(2, '0')).join()}';
  }

  String _hashUserOperation(UserOperation op) {
    return _hash('${op.sender}${op.nonce}${op.callData}');
  }

  String _hash(String input) {
    return input.codeUnits.fold(0, (prev, e) => (prev * 31 + e) % 0xFFFFFFFF).toRadixString(16).padLeft(64, '0');
  }
}

class SmartAccount {
  final String address;
  final String owner;
  int nonce;
  bool isDeployed;
  final String entryPoint;

  SmartAccount({
    required this.address,
    required this.owner,
    required this.nonce,
    required this.isDeployed,
    required this.entryPoint,
  });
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
}

class SessionKey {
  final String keyAddress;
  final String dAppAddress;
  final int validUntil;
  final List<String> allowedContracts;
  final List<String> allowedSelectors;
  final String spendingLimit;
  String spentAmount;
  bool isRevoked;

  SessionKey({
    required this.keyAddress,
    required this.dAppAddress,
    required this.validUntil,
    required this.allowedContracts,
    required this.allowedSelectors,
    required this.spendingLimit,
    required this.spentAmount,
    required this.isRevoked,
  });
}
