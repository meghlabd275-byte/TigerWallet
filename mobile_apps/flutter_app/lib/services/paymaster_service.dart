///
/// Paymaster Service - Flutter Implementation
/// Identical across ALL platforms
///

class PaymasterService {
  static final PaymasterService _instance = PaymasterService._internal();
  factory PaymasterService() => _instance;
  PaymasterService._internal();

  Map<String, WhitelistEntry> _whitelistedDApps = {};
  String? _gasToken;

  Future<PaymasterData> sponsorUserOp(UserOperation userOp) async {
    return PaymasterData(
      paymasterAndData: _buildPaymasterData(userOp),
      preVerificationGas: '0x5208',
      verificationGasLimit: '0x186A0',
      callGasLimit: '0x5208',
    );
  }

  bool setPaymentToken(String tokenAddress) {
    _gasToken = tokenAddress;
    return true;
  }

  String? getPaymentToken() => _gasToken;

  bool whitelistDApp(String dAppAddress, String limit, int expiry) {
    _whitelistedDApps[dAppAddress] = WhitelistEntry(
      address: dAppAddress,
      sponsorLimit: limit,
      expiry: expiry,
      isActive: true,
    );
    return true;
  }

  WhitelistStatus? getWhitelistStatus(String address) {
    final entry = _whitelistedDApps[address];
    if (entry == null) return null;
    return WhitelistStatus(
      isWhitelisted: entry.isActive,
      limit: entry.sponsorLimit,
      expiry: entry.expiry,
      used: '0',
    );
  }

  String getBalance() => '1000000000000000000';

  String _buildPaymasterData(UserOperation userOp) {
    final hash = _hash('${userOp.sender}${userOp.nonce}${_gasToken ?? ""}');
    return '0xPaymasterAddress${'0' * 64}${hash.substring(0, 32)}';
  }

  String _hash(String input) {
    return input.codeUnits.fold(0, (prev, e) => (prev * 31 + e) % 0xFFFFFFFF).toRadixString(16).padLeft(64, '0');
  }
}

class WhitelistEntry {
  final String address;
  final String sponsorLimit;
  final int expiry;
  final bool isActive;

  WhitelistEntry({
    required this.address,
    required this.sponsorLimit,
    required this.expiry,
    required this.isActive,
  });
}

class WhitelistStatus {
  final bool isWhitelisted;
  final String limit;
  final int expiry;
  final String used;

  WhitelistStatus({
    required this.isWhitelisted,
    required this.limit,
    required this.expiry,
    required this.used,
  });
}

class PaymasterData {
  final String paymasterAndData;
  final String preVerificationGas;
  final String verificationGasLimit;
  final String callGasLimit;

  PaymasterData({
    required this.paymasterAndData,
    required this.preVerificationGas,
    required this.verificationGasLimit,
    required this.callGasLimit,
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
