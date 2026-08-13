/**
 * PaymasterService - Flutter Implementation
 *
 * Gas prices are the LIVE values from the backend's public GET /api/v1/gas
 * endpoint (real eth_feeHistory/eth_gasPrice RPC). Sponsor signatures are
 * obtained from the backend verifying-paymaster relayer. This client NEVER
 * fabricates gas values, base fees, or sponsor signatures.
 */

import 'dart:convert';
import 'package:http/http.dart' as http;

class PaymasterService {
  static PaymasterService? _instance;
  static PaymasterService get instance {
    _instance ??= PaymasterService._();
    return _instance!;
  }

  PaymasterService._();

  static const String API_BASE = String.fromEnvironment(
    'MASTER_WALLET_API_URL',
    defaultValue: 'http://localhost:8450',
  );

  String? _token;
  void setToken(String? token) => _token = token;

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (_token != null) 'Authorization': 'Bearer $_token',
      };

  // ==================== Gas (real, from backend) ====================

  /// Fetch live gas prices from GET /api/v1/gas?chain_id=N (canonical public
  /// endpoint, real eth_feeHistory/eth_gasPrice RPC). Throws on any failure so
  /// callers never fall back to fabricated gas values.
  Future<GasEstimate> fetchGasPrices({int chainId = 1}) async {
    final r = await http.get(
      Uri.parse('$API_BASE/api/v1/gas').replace(
        queryParameters: {'chain_id': chainId.toString()},
      ),
      headers: _headers,
    );
    if (r.statusCode != 200) {
      throw PaymasterException(
        'fetchGasPrices failed (${r.statusCode}): ${r.body}',
      );
    }
    return GasEstimate.fromJson(jsonDecode(r.body) as Map<String, dynamic>);
  }

  // ==================== Sponsorship (no canonical endpoint) ================
  // The canonical backend contract (:8450) exposes NO verifying-paymaster /
  // sponsor-signature endpoint. Sponsor signing therefore fails closed rather
  // than calling a non-existent route or fabricating a signature.

  /// Request a sponsor signature. NOT supported by the canonical backend.
  Future<PaymasterSponsorship> sponsorUserOperation(
    Map<String, dynamic> userOp, {
    int? chainId,
    String? sponsorshipPolicy,
  }) async {
    throw PaymasterException(
      'paymaster sponsorship is not supported by the canonical backend '
      'contract. The backend exposes no verifying-paymaster / sponsor endpoint. '
      'Submit the transaction via the canonical master-wallet sign route and '
      'have the user pay gas directly.',
    );
  }

  // ==================== Fail-closed (no canonical endpoint) ====================

  /// Local sponsor signing is NOT supported: the paymaster signing key never
  /// lives on this client. Use [sponsorUserOperation] to obtain a real
  /// signature from the backend.
  Future<String> signSponsorship(String userOpHash, String privateKey) async {
    throw PaymasterException(
      'Local sponsor-signature signing is not supported. Request a sponsor '
      'signature via sponsorUserOperation() so the backend verifying-paymaster '
      'signs with the provisioned paymaster key.',
    );
  }
}

class PaymasterException implements Exception {
  final String message;
  PaymasterException(this.message);
  @override
  String toString() => 'PaymasterException: $message';
}

class GasEstimate {
  final BigInt gasPrice;
  final BigInt maxFeePerGas;
  final BigInt maxPriorityFeePerGas;
  final int? chainId;
  final String source;

  GasEstimate({
    required this.gasPrice,
    required this.maxFeePerGas,
    required this.maxPriorityFeePerGas,
    this.chainId,
    this.source = 'live_rpc',
  });

  factory GasEstimate.fromJson(Map<String, dynamic> json) {
    BigInt parse(dynamic v) {
      if (v == null) return BigInt.zero;
      final s = v.toString();
      if (s.startsWith('0x')) {
        return BigInt.tryParse(s.substring(2), radix: 16) ?? BigInt.zero;
      }
      return BigInt.tryParse(s) ?? BigInt.zero;
    }
    return GasEstimate(
      gasPrice: parse(json['gas_price']),
      maxFeePerGas: parse(json['max_fee']),
      maxPriorityFeePerGas: parse(json['priority_fee']),
      chainId: (json['chain_id'] as num?)?.toInt(),
      source: json['source'] as String? ?? 'live_rpc',
    );
  }
}

class PaymasterSponsorship {
  final String paymasterAddress;
  final String sponsorSignature;
  final String paymasterAndData;
  final BigInt validUntil;
  final BigInt validAfter;

  PaymasterSponsorship({
    required this.paymasterAddress,
    required this.sponsorSignature,
    required this.paymasterAndData,
    required this.validUntil,
    required this.validAfter,
  });

  factory PaymasterSponsorship.fromJson(Map<String, dynamic> json) =>
      PaymasterSponsorship(
        paymasterAddress: json['paymaster_address'] as String? ?? '',
        sponsorSignature: json['sponsor_signature'] as String? ?? '',
        paymasterAndData: json['paymaster_and_data'] as String? ?? '',
        validUntil: _parseHex(json['valid_until']),
        validAfter: _parseHex(json['valid_after']),
      );

  static BigInt _parseHex(dynamic v) {
    if (v == null) return BigInt.zero;
    final s = v.toString();
    if (s.startsWith('0x')) {
      return BigInt.tryParse(s.substring(2), radix: 16) ?? BigInt.zero;
    }
    return BigInt.tryParse(s) ?? BigInt.zero;
  }
}
